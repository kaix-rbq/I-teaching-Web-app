// Package kb 负责知识库源文件的解析、切片与入库。
//
// 设计边界：本包不依赖 HTTP 服务层，只依赖 GORM（存储）与 agent.Embedder（向量化）；
// cmd/kb 负责 CLI 参数、配置与脱敏器装配。
package kb

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ErrPDFUnsupported 表示源文件是 PDF；按 RAG §5.2 先离线转成 md/txt 再入库。
var ErrPDFUnsupported = errors.New("kb: 不支持直接解析 PDF，请先用 pdftotext 转成 .md/.txt 后重试")

// ExtractFile 按扩展名抽取纯文本，返回文本与 source_format（md/txt/docx）。
func ExtractFile(path string) (string, string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".md", ".markdown":
		raw, err := os.ReadFile(path)
		if err != nil {
			return "", "", err
		}
		return normalizeText(string(raw)), "md", nil
	case ".txt":
		raw, err := os.ReadFile(path)
		if err != nil {
			return "", "", err
		}
		return normalizeText(string(raw)), "txt", nil
	case ".docx":
		text, err := extractDOCX(path)
		if err != nil {
			return "", "", err
		}
		return normalizeText(text), "docx", nil
	case ".pdf":
		return "", "", ErrPDFUnsupported
	default:
		return "", "", fmt.Errorf("kb: 不支持的扩展名 %q（支持 .md/.txt/.docx）", ext)
	}
}

// normalizeText 统一换行并去掉 BOM/NUL，避免切片与检索出现不可见字符。
func normalizeText(s string) string {
	s = strings.TrimPrefix(s, "\ufeff")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.ReplaceAll(s, "\x00", "")
	return strings.TrimSpace(s)
}

// extractDOCX 解压 docx 并读取 word/document.xml 的段落文本。
//
// 只依赖标准库：遍历 <w:p> 段落，收集 <w:t> 文本；<w:br>/<w:tab> 转为换行/空格。
// 表格单元格里的文本同样会被收集；页眉页脚不在 document.xml 中，天然忽略。
func extractDOCX(path string) (string, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return "", fmt.Errorf("打开 docx 失败: %w", err)
	}
	defer zr.Close()

	var doc *zip.File
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			doc = f
			break
		}
	}
	if doc == nil {
		return "", errors.New("docx 缺少 word/document.xml")
	}
	rc, err := doc.Open()
	if err != nil {
		return "", fmt.Errorf("读取 document.xml 失败: %w", err)
	}
	defer rc.Close()

	dec := xml.NewDecoder(io.LimitReader(rc, 64<<20))
	var b strings.Builder
	inText := false
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", fmt.Errorf("解析 document.xml 失败: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "t":
				inText = true
			case "tab":
				b.WriteByte(' ')
			case "br":
				b.WriteByte('\n')
			}
		case xml.CharData:
			if inText {
				b.Write(t)
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "t":
				inText = false
			case "p":
				b.WriteByte('\n')
			}
		}
	}
	return b.String(), nil
}
