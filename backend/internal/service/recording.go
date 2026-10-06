package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"aijiaoxue-api/internal/asr"
	"aijiaoxue-api/internal/config"
	"aijiaoxue-api/internal/dto"
	"aijiaoxue-api/internal/model"
	"aijiaoxue-api/internal/repository"
	"aijiaoxue-api/pkg/errcode"
	"aijiaoxue-api/pkg/jwtutil"
	"github.com/google/uuid"
)

// Transcriber 是 ASR 能力接口；internal/asr.Client 实现它，测试用假实现替换。
type Transcriber interface {
	Transcribe(ctx context.Context, filePath string) (*asr.Result, error)
}

// playbackTicketTTL 是音频播放票据的有效期。
// 必须覆盖整段播放：浏览器播放期间会持续发出多个 Range 请求，每个都要重新校验票据。
const playbackTicketTTL = 4 * time.Hour

// transcriptErrMaxLen 对齐 transcripts.error_message 的 VARCHAR(255)。
const transcriptErrMaxLen = 255

type RecordingService interface {
	Upload(ctx context.Context, userID, sessionID uint64, file *multipart.FileHeader) (*dto.RecordingDTO, error)
	Media(ctx context.Context, role string, deptID, userID, sessionID uint64) (*dto.RecordingDTO, *dto.TranscriptDTO, error)
	Stream(ctx context.Context, role string, deptID, userID, recordingID uint64) (*model.Recording, error)
	Retry(ctx context.Context, role string, deptID, userID, sessionID uint64) error
	// RequeueStuck 在服务启动时把上次进程中断遗留的未完成转写重新入队（幂等）。
	RequeueStuck(ctx context.Context) (int, error)
}

type recordingService struct {
	recordings    repository.RecordingRepository
	sessions      repository.SessionRepository
	upload        config.UploadConfig
	transcription config.TranscriptionConfig
	// transcriber 为 nil 表示 ASR 未启用，转写任务一律置 failed 且不影响主流程。
	transcriber Transcriber
	jwt         *jwtutil.Manager
	scrubber    Scrubber
	// sem 是并发闸门，容量 = transcription.maxConcurrency，避免同时把多个大文件推进内存。
	sem chan struct{}
}

func NewRecordingService(
	r repository.RecordingRepository,
	s repository.SessionRepository,
	upload config.UploadConfig,
	transcription config.TranscriptionConfig,
	jwt *jwtutil.Manager,
	transcriber Transcriber,
) RecordingService {
	return &recordingService{
		recordings:    r,
		sessions:      s,
		upload:        upload,
		transcription: transcription,
		transcriber:   transcriber,
		jwt:           jwt,
		scrubber:      NewScrubber(transcription.StudentNames),
		sem:           make(chan struct{}, transcription.Concurrency()),
	}
}

func (s *recordingService) Upload(ctx context.Context, userID, sessionID uint64, file *multipart.FileHeader) (*dto.RecordingDTO, error) {
	row, err := s.sessions.GetDetail(ctx, sessionID)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, errcode.New(errcode.NotFound, "授课记录不存在")
		}
		return nil, errcode.Wrap(errcode.Internal, "查询授课记录失败", err)
	}
	if row.TeacherID == 0 {
		return nil, errcode.New(errcode.BizRule, "授课记录缺少教师")
	}
	// 只有督导调用此方法；角色校验由路由完成，用户 id 仍写入审计与记录。
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".mp3" && ext != ".wav" && ext != ".m4a" {
		return nil, errcode.New(errcode.Params, "仅支持 mp3、wav、m4a 音频")
	}
	if s.upload.MaxSize > 0 && file.Size > s.upload.MaxSize {
		return nil, errcode.Newf(errcode.Params, "音频超过大小上限 %d MB", s.upload.MaxSize/1024/1024)
	}
	if err := sniffAudio(file); err != nil {
		return nil, err
	}
	dir := filepath.Join(s.upload.Dir, "recordings", fmt.Sprintf("%d", sessionID))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, errcode.Wrap(errcode.Internal, "创建音频目录失败", err)
	}
	path := filepath.Join(dir, uuid.NewString()+ext)
	if err := copyMultipart(path, file); err != nil {
		return nil, errcode.Wrap(errcode.Internal, "保存音频失败", err)
	}
	record := &model.Recording{SessionID: sessionID, FilePath: path, OriginalName: filepath.Base(file.Filename), Format: strings.TrimPrefix(ext, "."), Size: file.Size, UploadedBy: userID, UploadedAt: time.Now()}
	if err := s.recordings.CreateRecording(ctx, record); err != nil {
		_ = os.Remove(path)
		if repository.IsDuplicate(err) {
			return nil, errcode.New(errcode.Conflict, "该课程已存在课堂录音")
		}
		return nil, errcode.Wrap(errcode.Internal, "保存录音记录失败", err)
	}
	transcript := &model.Transcript{SessionID: sessionID, RecordingID: record.ID, Content: "", Status: model.TranscriptPending}
	if err := s.recordings.CreateTranscript(ctx, transcript); err != nil {
		return nil, errcode.Wrap(errcode.Internal, "创建转写任务失败", err)
	}
	// 第二个参数必须是录音 id：转写任务靠它取磁盘文件路径。
	go s.processTranscript(context.Background(), transcript.ID, record.ID)
	return &dto.RecordingDTO{ID: record.ID, OriginalName: record.OriginalName, Format: record.Format, Size: record.Size, DurationSec: record.DurationSec, StreamURL: fmt.Sprintf("/recordings/%d/stream", record.ID)}, nil
}

func (s *recordingService) Media(ctx context.Context, role string, deptID, userID, sessionID uint64) (*dto.RecordingDTO, *dto.TranscriptDTO, error) {
	row, err := s.sessions.GetDetail(ctx, sessionID)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, nil, errcode.New(errcode.NotFound, "授课记录不存在")
		}
		return nil, nil, errcode.Wrap(errcode.Internal, "查询授课记录失败", err)
	}
	if err := checkSessionScope(ScopeFor(role, deptID, userID), row); err != nil {
		return nil, nil, err
	}
	rec, err := s.recordings.GetRecordingBySession(ctx, sessionID)
	if err != nil && !repository.IsNotFound(err) {
		return nil, nil, errcode.Wrap(errcode.Internal, "查询课堂录音失败", err)
	}
	tr, err := s.recordings.GetTranscriptBySession(ctx, sessionID)
	if err != nil && !repository.IsNotFound(err) {
		return nil, nil, errcode.Wrap(errcode.Internal, "查询转写任务失败", err)
	}
	var rd *dto.RecordingDTO
	var td *dto.TranscriptDTO
	if rec != nil {
		rd = &dto.RecordingDTO{ID: rec.ID, OriginalName: rec.OriginalName, Format: rec.Format, Size: rec.Size, DurationSec: rec.DurationSec, StreamURL: fmt.Sprintf("/recordings/%d/stream", rec.ID)}
		// 只有督导可听音频（开发计划 §5.1 权限矩阵）：教师侧不签发播放票据。
		if role == RoleSupervisor {
			ticket, err := s.jwt.SignPlayback(userID, role, deptID, rec.ID, playbackTicketTTL, time.Now())
			if err != nil {
				return nil, nil, errcode.Wrap(errcode.Internal, "签发播放票据失败", err)
			}
			rd.PlaybackURL = fmt.Sprintf("/recordings/%d/stream?ticket=%s", rec.ID, url.QueryEscape(ticket))
		}
	}
	if tr != nil {
		td = transcriptDTO(tr)
	}
	return rd, td, nil
}

func (s *recordingService) Stream(ctx context.Context, role string, deptID, userID, recordingID uint64) (*model.Recording, error) {
	if role != RoleSupervisor {
		return nil, errcode.New(errcode.ForbiddenData, "教师端不可访问课堂音频")
	}
	rec, err := s.recordings.GetRecording(ctx, recordingID)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, errcode.New(errcode.NotFound, "录音不存在")
		}
		return nil, errcode.Wrap(errcode.Internal, "查询录音失败", err)
	}
	row, err := s.sessions.GetDetail(ctx, rec.SessionID)
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "查询授课记录失败", err)
	}
	if err := checkSessionScope(ScopeFor(role, deptID, userID), row); err != nil {
		return nil, err
	}
	_ = s.recordings.LogPlayback(ctx, recordingID, userID, time.Now())
	return rec, nil
}

func (s *recordingService) Retry(ctx context.Context, role string, deptID, userID, sessionID uint64) error {
	if role != RoleSupervisor {
		return errcode.New(errcode.ForbiddenRole, "仅督导可重试转写")
	}
	row, err := s.sessions.GetDetail(ctx, sessionID)
	if err != nil {
		return errcode.Wrap(errcode.Internal, "查询授课记录失败", err)
	}
	if err := checkSessionScope(ScopeFor(role, deptID, userID), row); err != nil {
		return err
	}
	tr, err := s.recordings.GetTranscriptBySession(ctx, sessionID)
	if err != nil {
		return errcode.New(errcode.NotFound, "转写任务不存在")
	}
	if err := s.recordings.UpdateTranscript(ctx, tr.ID, map[string]any{"status": model.TranscriptPending, "error_message": ""}); err != nil {
		return errcode.Wrap(errcode.Internal, "重试转写失败", err)
	}
	go s.processTranscript(context.Background(), tr.ID, tr.RecordingID)
	return nil
}

// RequeueStuck 启动时清扫僵尸任务：上次进程中断会把 running 永久卡住，
// pending 则说明入队后进程即退出、分发丢失。两者都需要重新入队。
//
// 幂等：可重复调用；已 done/failed 的记录不受影响。
func (s *recordingService) RequeueStuck(ctx context.Context) (int, error) {
	rows, err := s.recordings.ListStuckTranscripts(ctx)
	if err != nil {
		return 0, errcode.Wrap(errcode.Internal, "查询未完成转写失败", err)
	}
	requeued := 0
	for _, tr := range rows {
		if tr.RecordingID == 0 {
			continue
		}
		if err := s.recordings.UpdateTranscript(ctx, tr.ID,
			map[string]any{"status": model.TranscriptPending, "error_message": ""}); err != nil {
			slog.Warn("重排转写任务失败", "transcriptID", tr.ID, "err", err)
			continue
		}
		go s.processTranscript(context.Background(), tr.ID, tr.RecordingID)
		requeued++
	}
	return requeued, nil
}

// processTranscript 是 ASR 异步任务体。
//
// 与督导评分主链路完全解耦（开发计划 §4.4 硬约束）：任何失败只落到
// transcripts.status='failed' + error_message，绝不向调用方传播，也绝不阻断评估流程。
func (s *recordingService) processTranscript(parent context.Context, transcriptID, recordingID uint64) {
	ctx, cancel := context.WithTimeout(parent, s.transcription.TimeoutDuration())
	defer cancel()

	setFailed := func(reason string) {
		if len(reason) > transcriptErrMaxLen {
			reason = reason[:transcriptErrMaxLen]
		}
		// 用独立短上下文：主上下文可能已超时/取消，但失败状态仍必须落库，
		// 否则前端会永远停在 running（联调时真实踩到过）。
		failCtx, cancelFail := context.WithTimeout(parent, 10*time.Second)
		defer cancelFail()
		_ = s.recordings.UpdateTranscript(failCtx, transcriptID,
			map[string]any{"status": model.TranscriptFailed, "error_message": reason})
	}

	if err := s.recordings.UpdateTranscript(ctx, transcriptID,
		map[string]any{"status": model.TranscriptRunning, "error_message": ""}); err != nil {
		slog.Error("转写状态置 running 失败", "transcriptID", transcriptID, "err", err)
		return
	}

	if s.transcriber == nil {
		setFailed("ASR 引擎未配置")
		slog.Warn("转写跳过：ASR 引擎未配置", "transcriptID", transcriptID)
		return
	}

	rec, err := s.recordings.GetRecording(ctx, recordingID)
	if err != nil {
		setFailed("查询录音失败")
		slog.Error("转写失败：查询录音", "transcriptID", transcriptID, "recordingID", recordingID, "err", err)
		return
	}

	// 并发闸门：拿不到令牌就排队，避免同时把多个大文件推进内存。
	select {
	case s.sem <- struct{}{}:
		defer func() { <-s.sem }()
	case <-ctx.Done():
		setFailed("转写排队超时")
		return
	}

	res, err := s.transcriber.Transcribe(ctx, rec.FilePath)
	if err != nil {
		// 不外泄上游原始报文，只记类别 + 服务端日志留完整原因。
		setFailed("语音识别调用失败")
		slog.Error("ASR 调用失败", "transcriptID", transcriptID, "recordingID", recordingID, "err", err)
		return
	}
	if strings.TrimSpace(res.Content) == "" {
		setFailed("识别结果为空")
		return
	}

	// 合规 §5.2-5：落库前必须做学生姓名脱敏，且全文与分段保持一致。
	content, segments := s.scrubber.Scrub(res.Content, res.Segments)
	segJSON, err := json.Marshal(segments)
	if err != nil {
		setFailed("转写分段序列化失败")
		slog.Error("转写分段序列化失败", "transcriptID", transcriptID, "err", err)
		return
	}
	segStr := string(segJSON)

	if err := s.recordings.UpdateTranscript(ctx, transcriptID, map[string]any{
		"content":        content,
		"segments":       segStr,
		"engine":         s.transcription.Engine,
		"engine_version": s.transcription.EngineVersion,
		"status":         model.TranscriptDone,
		"error_message":  "",
	}); err != nil {
		// 已付费拿到识别结果却写不进去（如列宽不足触发 1406）：必须置 failed，
		// 否则整条 UPDATE 回滚后 status 仍是 running，前端会无限轮询。
		slog.Error("写入转写结果失败", "transcriptID", transcriptID, "err", err)
		setFailed("转写结果保存失败")
		return
	}

	// 回写时长：前端播放器与时长校验都需要（ASR 只在成功时返回，缺失则不动原值）。
	if err := s.recordings.UpdateRecordingDuration(ctx, recordingID, res.DurationMS/1000); err != nil {
		slog.Warn("回写录音时长失败", "recordingID", recordingID, "err", err)
	}

	slog.Info("转写完成",
		"transcriptID", transcriptID,
		"recordingID", recordingID,
		"segments", len(segments),
		"chars", len([]rune(content)),
		"durationMs", res.DurationMS,
	)
}

func transcriptDTO(v *model.Transcript) *dto.TranscriptDTO {
	out := &dto.TranscriptDTO{ID: v.ID, Status: v.Status, Content: v.Content, Engine: v.Engine, EngineVersion: v.EngineVersion, ErrorMessage: v.ErrorMessage, Segments: []dto.TranscriptSegment{}}
	if v.Segments != nil {
		_ = json.Unmarshal([]byte(*v.Segments), &out.Segments)
	}
	return out
}

func sniffAudio(file *multipart.FileHeader) error {
	f, err := file.Open()
	if err != nil {
		return errcode.Wrap(errcode.Params, "无法读取音频", err)
	}
	defer f.Close()
	head := make([]byte, 512)
	n, err := f.Read(head)
	if err != nil && err != io.EOF {
		return errcode.Wrap(errcode.Params, "无法读取音频", err)
	}
	mime := http.DetectContentType(head[:n])
	if strings.HasPrefix(mime, "text/") || strings.Contains(mime, "html") {
		return errcode.New(errcode.Params, "音频文件格式校验失败")
	}
	return nil
}

// copyMultipart 落盘音频：0640 权限（音频含学生人声，不做全局可读），
// 并显式 Sync，避免断电/崩溃留下半截文件被后续转写当成完整音频。
func copyMultipart(target string, file *multipart.FileHeader) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o640)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		return err
	}
	if err := dst.Sync(); err != nil {
		dst.Close()
		return err
	}
	return dst.Close()
}
