package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"aijiaoxue-api/internal/config"
	"aijiaoxue-api/internal/dto"
	"aijiaoxue-api/internal/model"
	"aijiaoxue-api/internal/repository"
	"aijiaoxue-api/pkg/errcode"
	"github.com/google/uuid"
)

type RecordingService interface {
	Upload(ctx context.Context, userID, sessionID uint64, file *multipart.FileHeader) (*dto.RecordingDTO, error)
	Media(ctx context.Context, role string, deptID, userID, sessionID uint64) (*dto.RecordingDTO, *dto.TranscriptDTO, error)
	Stream(ctx context.Context, role string, deptID, userID, recordingID uint64) (*model.Recording, error)
	Retry(ctx context.Context, role string, deptID, userID, sessionID uint64) error
}

type recordingService struct {
	recordings    repository.RecordingRepository
	sessions      repository.SessionRepository
	upload        config.UploadConfig
	transcription config.TranscriptionConfig
}

func NewRecordingService(r repository.RecordingRepository, s repository.SessionRepository, upload config.UploadConfig, transcription config.TranscriptionConfig) RecordingService {
	return &recordingService{recordings: r, sessions: s, upload: upload, transcription: transcription}
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
	transcript := &model.Transcript{SessionID: sessionID, RecordingID: record.ID, Status: model.TranscriptPending}
	if err := s.recordings.CreateTranscript(ctx, transcript); err != nil {
		return nil, errcode.Wrap(errcode.Internal, "创建转写任务失败", err)
	}
	go s.processTranscript(context.Background(), transcript.ID, sessionID)
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
	go s.processTranscript(context.Background(), tr.ID, sessionID)
	return nil
}

func (s *recordingService) processTranscript(ctx context.Context, transcriptID, _ uint64) {
	_ = s.recordings.UpdateTranscript(ctx, transcriptID, map[string]any{"status": model.TranscriptRunning})
	if !s.transcription.Enabled || strings.TrimSpace(s.transcription.Engine) == "" {
		_ = s.recordings.UpdateTranscript(ctx, transcriptID, map[string]any{"status": model.TranscriptFailed, "error_message": "ASR 引擎未配置"})
		return
	}
	// ASR 适配层预留：引擎可配置后在此写入脱敏文本与分段。
	_ = s.recordings.UpdateTranscript(ctx, transcriptID, map[string]any{"status": model.TranscriptFailed, "error_message": "ASR 引擎不可用"})
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
func copyMultipart(target string, file *multipart.FileHeader) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.Create(target)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	return err
}
