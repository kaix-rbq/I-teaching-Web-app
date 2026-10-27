// Command kb 离线构建知识库：ingest / list / rebuild。
//
// 用法（在 backend 目录）：
//
//	go run ./cmd/kb ingest --dir ./kb-src/kb-rubric --collection kb-rubric
//	go run ./cmd/kb list --collection kb-rubric
//	go run ./cmd/kb rebuild --dir ./kb-src/kb-rubric --collection kb-rubric
//
// 目录约定：kb-src/<collection>/…；--collection 省略时取目录名。
// PDF 必须先 pdftotext 转成 .md/.txt；本命令不解析 PDF。
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"aijiaoxue-api/internal/agent"
	"aijiaoxue-api/internal/config"
	"aijiaoxue-api/internal/kb"
	"aijiaoxue-api/pkg/scrub"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd := os.Args[1]
	args := os.Args[2:]
	var err error
	switch cmd {
	case "ingest":
		err = runIngest(args)
	case "list":
		err = runList(args)
	case "rebuild":
		err = runRebuild(args)
	case "help", "-h", "--help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "未知子命令 %q\n\n", cmd)
		usage()
		os.Exit(2)
	}
	if err != nil {
		slog.Error("kb command failed", "cmd", cmd, "error", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `知识库 CLI

  go run ./cmd/kb ingest  --dir <目录> [--collection <集合>] [--course-id N] [--subject 学科]
  go run ./cmd/kb list    [--collection <集合>]
  go run ./cmd/kb rebuild --dir <目录> --collection <集合>

集合：kb-rubric / kb-standard / kb-textbook / kb-strategy / kb-case
PDF 请先 pdftotext 转成 .md/.txt；支持 .md/.markdown/.txt/.docx。
`)
}

func runIngest(args []string) error {
	fs := flag.NewFlagSet("ingest", flag.ExitOnError)
	configPath := fs.String("config", "config.yaml", "配置文件路径")
	dir := fs.String("dir", "", "知识库源目录（必填）")
	collection := fs.String("collection", "", "集合名；省略时取目录名")
	courseID := fs.Uint64("course-id", 0, "绑定课程 ID；0 = 通用")
	subject := fs.String("subject", "", "学科，用于教材类集合过滤")
	_ = fs.Parse(args)
	if strings.TrimSpace(*dir) == "" {
		return fmt.Errorf("ingest 需要 --dir")
	}
	coll := strings.TrimSpace(*collection)
	if coll == "" {
		coll = filepath.Base(filepath.Clean(*dir))
	}
	if err := kb.ValidateCollection(coll); err != nil {
		return err
	}
	cfg, db, err := load(*configPath)
	if err != nil {
		return err
	}
	defer closeDB(db)
	store, ing, err := buildIngester(cfg, db)
	if err != nil {
		return err
	}
	if ok, err := store.HasTables(context.Background()); err != nil {
		return err
	} else if !ok {
		return fmt.Errorf("knowledge 表不存在，请先执行：go run ./cmd/migrate -config %s up", *configPath)
	}
	return ingestDir(context.Background(), store, ing, *dir, coll, *courseID, *subject)
}

func runList(args []string) error {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	configPath := fs.String("config", "config.yaml", "配置文件路径")
	collection := fs.String("collection", "", "集合名；省略时列出全部")
	_ = fs.Parse(args)
	_, db, err := load(*configPath)
	if err != nil {
		return err
	}
	defer closeDB(db)
	store := kb.NewStore(db)
	if ok, err := store.HasTables(context.Background()); err != nil {
		return err
	} else if !ok {
		return fmt.Errorf("knowledge 表不存在，请先执行：go run ./cmd/migrate -config %s up", *configPath)
	}
	docs, err := store.ListDocuments(context.Background(), strings.TrimSpace(*collection))
	if err != nil {
		return err
	}
	if len(docs) == 0 {
		fmt.Println("（无文档）")
		return nil
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tCOLLECTION\tTITLE\tCHUNKS\tMODEL\tDIM\tSTATUS\tSOURCE\tUPDATED")
	for _, d := range docs {
		errMsg := ""
		if d.ErrorMessage != "" {
			errMsg = "  error=" + d.ErrorMessage
		}
		fmt.Fprintf(tw, "%d\t%s\t%s\t%d\t%s\t%d\t%s%s\t%s\t%s\n",
			d.ID, d.Collection, d.Title, d.ChunkCount, d.EmbeddingModel, d.EmbeddingDim,
			d.Status, errMsg, d.SourcePath, d.UpdatedAt.Format("2006-01-02 15:04"))
	}
	return tw.Flush()
}

func runRebuild(args []string) error {
	fs := flag.NewFlagSet("rebuild", flag.ExitOnError)
	configPath := fs.String("config", "config.yaml", "配置文件路径")
	dir := fs.String("dir", "", "知识库源目录（必填）")
	collection := fs.String("collection", "", "集合名（必填）")
	courseID := fs.Uint64("course-id", 0, "绑定课程 ID；0 = 通用")
	subject := fs.String("subject", "", "学科，用于教材类集合过滤")
	_ = fs.Parse(args)
	if strings.TrimSpace(*dir) == "" {
		return fmt.Errorf("rebuild 需要 --dir（重建 = 清空集合后按源文件重新摄取）")
	}
	coll := strings.TrimSpace(*collection)
	if coll == "" {
		return fmt.Errorf("rebuild 需要 --collection，避免误清空其他集合")
	}
	if err := kb.ValidateCollection(coll); err != nil {
		return err
	}
	cfg, db, err := load(*configPath)
	if err != nil {
		return err
	}
	defer closeDB(db)
	store, ing, err := buildIngester(cfg, db)
	if err != nil {
		return err
	}
	if ok, err := store.HasTables(context.Background()); err != nil {
		return err
	} else if !ok {
		return fmt.Errorf("knowledge 表不存在，请先执行：go run ./cmd/migrate -config %s up", *configPath)
	}
	n, err := store.DeleteCollection(context.Background(), coll)
	if err != nil {
		return err
	}
	slog.Info("collection cleared", "collection", coll, "documents", n)
	return ingestDir(context.Background(), store, ing, *dir, coll, *courseID, *subject)
}

func load(path string) (*config.Config, *gorm.DB, error) {
	cfg, err := config.Load(path)
	if err != nil {
		return nil, nil, fmt.Errorf("加载配置失败: %w", err)
	}
	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("连接数据库失败: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, fmt.Errorf("获取 sql.DB 失败: %w", err)
	}
	sqlDB.SetMaxOpenConns(cfg.MySQL.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MySQL.MaxIdleConns)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, nil, fmt.Errorf("数据库 ping 失败: %w", err)
	}
	return cfg, db, nil
}

func buildIngester(cfg *config.Config, db *gorm.DB) (*kb.Store, *kb.Ingester, error) {
	embedder := agent.NewEmbeddingClient(cfg.Agent, cfg.Embedding)
	if embedder == nil {
		return nil, nil, fmt.Errorf("embedding 未启用或缺凭据/模型：检查 embedding.enabled、AIJIAOXUE_AGENT_API_KEY 与 embedding.model")
	}
	scrubber := scrub.NewScrubber(cfg.Transcription.StudentNames)
	scrub := func(s string) string {
		out, _ := scrubber.Scrub(s, nil)
		return out
	}
	store := kb.NewStore(db)
	return store, &kb.Ingester{Store: store, Embedder: embedder, Scrub: scrub}, nil
}

func ingestDir(ctx context.Context, store *kb.Store, ing *kb.Ingester, dir, collection string, courseID uint64, subject string) error {
	root, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("解析目录失败: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil {
		return fmt.Errorf("读取目录失败: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s 不是目录", root)
	}
	var files []string
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".md", ".markdown", ".txt", ".docx":
			files = append(files, path)
		case ".pdf":
			fmt.Fprintf(os.Stderr, "跳过 PDF（请先 pdftotext）：%s\n", path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("遍历目录失败: %w", err)
	}
	sort.Strings(files)
	if len(files) == 0 {
		return fmt.Errorf("目录中没有可摄取文件（.md/.txt/.docx）")
	}

	var okCount, skipCount, failCount, chunkCount int
	for _, path := range files {
		rel, _ := filepath.Rel(root, path)
		res, err := ing.Ingest(ctx, kb.Source{
			Path:       path,
			RelPath:    rel,
			Collection: collection,
			CourseID:   courseID,
			Subject:    subject,
		})
		if err != nil {
			failCount++
			fmt.Fprintf(os.Stderr, "FAIL  %s: %v\n", rel, err)
			continue
		}
		if res.Skipped {
			skipCount++
			fmt.Printf("SKIP  %s（checksum/model 未变，%d chunks）\n", rel, res.Chunks)
			continue
		}
		okCount++
		chunkCount += res.Chunks
		fmt.Printf("OK    %s → %d chunks（%s）\n", rel, res.Chunks, res.Title)
	}
	slog.Info("kb ingest done", "collection", collection, "ok", okCount, "skipped", skipCount, "failed", failCount, "chunks", chunkCount)
	if failCount > 0 {
		return fmt.Errorf("%d 个文件失败，详见上方 FAIL 行", failCount)
	}
	return nil
}

func closeDB(db *gorm.DB) {
	if db == nil {
		return
	}
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
}
