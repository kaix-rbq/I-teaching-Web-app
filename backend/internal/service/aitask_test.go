package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"aijiaoxue-api/internal/agent"
	"aijiaoxue-api/internal/config"
	"aijiaoxue-api/internal/dto"
	"aijiaoxue-api/internal/model"
	"aijiaoxue-api/internal/repository"
)

type fakeAiTaskRepoForProcess struct {
	task    *model.AiTask
	updates []map[string]any
}

func (f *fakeAiTaskRepoForProcess) GetByID(context.Context, uint64) (*model.AiTask, error) {
	return f.task, nil
}
func (f *fakeAiTaskRepoForProcess) GetByKindRef(context.Context, string, string, uint64) (*model.AiTask, error) {
	return nil, repository.ErrNotImplemented
}
func (f *fakeAiTaskRepoForProcess) CreateIfAbsent(context.Context, *model.AiTask) error { return nil }
func (f *fakeAiTaskRepoForProcess) Update(_ context.Context, _ uint64, fields map[string]any) error {
	f.updates = append(f.updates, fields)
	if status, ok := fields["status"].(string); ok {
		f.task.Status = status
	}
	if msg, ok := fields["error_message"].(string); ok {
		f.task.ErrorMessage = msg
	}
	return nil
}
func (f *fakeAiTaskRepoForProcess) ListResumable(context.Context) ([]model.AiTask, error) {
	return nil, nil
}

func (f *fakeAiTaskRepoForProcess) ListDueTeacherReportTasks(context.Context, time.Time) ([]model.AiTask, error) {
	return nil, nil
}

type fakeRecordingRepoForProcess struct {
	transcript *model.Transcript
	evaluated  *model.Evaluation
}

func (f *fakeRecordingRepoForProcess) CreateRecording(context.Context, *model.Recording) error {
	return nil
}
func (f *fakeRecordingRepoForProcess) GetRecordingBySession(context.Context, uint64) (*model.Recording, error) {
	return nil, repository.ErrNotImplemented
}
func (f *fakeRecordingRepoForProcess) GetRecording(context.Context, uint64) (*model.Recording, error) {
	return nil, repository.ErrNotImplemented
}
func (f *fakeRecordingRepoForProcess) UpdateRecordingDuration(context.Context, uint64, int) error {
	return nil
}
func (f *fakeRecordingRepoForProcess) UpdateRecording(context.Context, uint64, map[string]any) error {
	return nil
}
func (f *fakeRecordingRepoForProcess) DeleteRecording(context.Context, uint64) error { return nil }
func (f *fakeRecordingRepoForProcess) DeletePlaybackLogsByRecording(context.Context, uint64) error {
	return nil
}
func (f *fakeRecordingRepoForProcess) CreateTranscript(context.Context, *model.Transcript) error {
	return nil
}
func (f *fakeRecordingRepoForProcess) GetTranscriptBySession(context.Context, uint64) (*model.Transcript, error) {
	return f.transcript, nil
}
func (f *fakeRecordingRepoForProcess) UpdateTranscript(context.Context, uint64, map[string]any) error {
	return nil
}
func (f *fakeRecordingRepoForProcess) DeleteTranscriptBySession(context.Context, uint64) error {
	return nil
}
func (f *fakeRecordingRepoForProcess) ListStuckTranscripts(context.Context) ([]model.Transcript, error) {
	return nil, nil
}
func (f *fakeRecordingRepoForProcess) UpsertAgentEvaluation(_ context.Context, e *model.Evaluation) error {
	f.evaluated = e
	return nil
}
func (f *fakeRecordingRepoForProcess) LogPlayback(context.Context, uint64, uint64, time.Time) error {
	return nil
}

type fakeAnalyzerForProcess struct {
	got   *agent.EvalInput
	calls int
}

func (f *fakeAnalyzerForProcess) AnalyzeSession(_ context.Context, in agent.EvalInput) (*agent.Result, error) {
	f.calls++
	f.got = &in
	return &agent.Result{
		ModelVersion: "fake-model",
		Evidence: agent.EvidenceJSON{
			SchemaVersion: 1,
			Dimensions:    map[string]agent.DimensionEvidence{},
			CitedChunks:   []string{"kb-rubric#13"},
		},
	}, nil
}

type fakeRetrieverForProcess struct {
	err  error
	gotQ string
	gotF agent.ChunkFilter
}

func (f *fakeRetrieverForProcess) Retrieve(_ context.Context, query string, filter agent.ChunkFilter) ([]agent.RetrievedChunk, error) {
	f.gotQ = query
	f.gotF = filter
	return nil, f.err
}

func TestProcessContinuesWhenRetrieverFails(t *testing.T) {
	task := &model.AiTask{ID: 7, Kind: model.AiTaskKindSessionEval, RefType: model.AiTaskRefSession, RefID: 42, Status: model.AiTaskStatusPending}
	tasks := &fakeAiTaskRepoForProcess{task: task}
	sessions := &fakeSessionRepo{
		getDetail: func(context.Context, uint64) (*repository.SessionDetailRow, error) {
			return &repository.SessionDetailRow{
				ID: 42, CourseID: 9, CourseName: "软件工程", TeacherName: "李老师",
				SessionDate: time.Date(2026, 10, 20, 0, 0, 0, 0, time.Local),
				Period:      "第1-2节", Topic: "用户故事",
			}, nil
		},
	}
	recordings := &fakeRecordingRepoForProcess{transcript: &model.Transcript{
		ID: 1, SessionID: 42, Content: "今天我们讲用户故事。", Status: model.TranscriptDone,
	}}
	evals := &fakeEvalRepo{
		upsert: func(context.Context, *model.Evaluation) error { return nil },
	}
	analyzer := &fakeAnalyzerForProcess{}
	retriever := &fakeRetrieverForProcess{err: errors.New("vector store unavailable")}
	svc := NewAiTaskService(tasks, sessions, evals, recordings, analyzer, retriever,
		config.AgentConfig{Timeout: "3m", MaxConcurrency: 1}, config.EvaluationConfig{})

	svc.(*aiTaskService).process(context.Background(), task.ID)

	assert.Equal(t, model.AiTaskStatusDone, task.Status)
	assert.Empty(t, task.ErrorMessage)
	require.NotNil(t, analyzer.got)
	assert.Empty(t, analyzer.got.Knowledge, "retriever 失败时必须降级为无 KB 继续评析")
	require.NotNil(t, retriever.gotF.CourseID)
	assert.Equal(t, uint64(9), *retriever.gotF.CourseID)
	assert.Equal(t, []string{"kb-rubric"}, retriever.gotF.Collections)
	assert.Contains(t, retriever.gotQ, "课堂教学质量 五维评分锚点")
}

func TestProcessInjectsRetrievedKnowledgeIntoAnalyzer(t *testing.T) {
	task := &model.AiTask{ID: 8, Kind: model.AiTaskKindSessionEval, RefType: model.AiTaskRefSession, RefID: 43, Status: model.AiTaskStatusPending}
	tasks := &fakeAiTaskRepoForProcess{task: task}
	sessions := &fakeSessionRepo{
		getDetail: func(context.Context, uint64) (*repository.SessionDetailRow, error) {
			return &repository.SessionDetailRow{
				ID: 43, CourseID: 10, CourseName: "软件工程", TeacherName: "王老师",
				SessionDate: time.Date(2026, 10, 21, 0, 0, 0, 0, time.Local),
				Period:      "第3-4节", Topic: "需求分析",
			}, nil
		},
	}
	recordings := &fakeRecordingRepoForProcess{transcript: &model.Transcript{
		ID: 2, SessionID: 43, Content: "学生提出的问题很有价值。", Status: model.TranscriptDone,
	}}
	evals := &fakeEvalRepo{upsert: func(context.Context, *model.Evaluation) error { return nil }}
	analyzer := &fakeAnalyzerForProcess{}
	retriever := &retrieverWithHits{
		hits: []agent.RetrievedChunk{{
			ChunkID: "kb-rubric#13", Collection: "kb-rubric",
			Heading: "互动维度锚点", Content: "开放问题与追问充分。",
		}},
	}
	svc := NewAiTaskService(tasks, sessions, evals, recordings, analyzer, retriever,
		config.AgentConfig{Timeout: "3m", MaxConcurrency: 1}, config.EvaluationConfig{})

	svc.(*aiTaskService).process(context.Background(), task.ID)

	assert.Equal(t, model.AiTaskStatusDone, task.Status)
	require.NotNil(t, analyzer.got)
	require.Len(t, analyzer.got.Knowledge, 1)
	assert.Equal(t, "kb-rubric#13", analyzer.got.Knowledge[0].ChunkID)
	assert.Equal(t, "互动维度锚点", analyzer.got.Knowledge[0].Heading)
	assert.Equal(t, "开放问题与追问充分。", analyzer.got.Knowledge[0].Content)
}

type retrieverWithHits struct{ hits []agent.RetrievedChunk }

func (r *retrieverWithHits) Retrieve(context.Context, string, agent.ChunkFilter) ([]agent.RetrievedChunk, error) {
	return r.hits, nil
}

type fakeReportEnqueuer struct {
	calls     int
	teacherID uint64
	semester  string
	force     bool
}

func (f *fakeReportEnqueuer) Enqueue(teacherID uint64, semester string, force bool) error {
	f.calls++
	f.teacherID = teacherID
	f.semester = semester
	f.force = force
	return nil
}

// TestProcessEnqueuesTeacherReportAfterS1Done 锁定 P3 触发：S1 done 后异步入队 S2，
// 不影响 S1 自身完成；payload 只记录教师与学期。
func TestProcessEnqueuesTeacherReportAfterS1Done(t *testing.T) {
	task := &model.AiTask{
		ID: 15, Kind: model.AiTaskKindSessionEval, RefType: model.AiTaskRefSession, RefID: 42,
		Status: model.AiTaskStatusPending,
	}
	tasks := &fakeAiTaskRepoForProcess{task: task}
	sessions := &fakeSessionRepo{
		getDetail: func(context.Context, uint64) (*repository.SessionDetailRow, error) {
			return &repository.SessionDetailRow{
				ID: 42, CourseID: 9, CourseName: "软件工程", TeacherID: 2, TeacherName: "李老师",
				Semester:    "2026-2027-1",
				SessionDate: time.Date(2026, 10, 20, 0, 0, 0, 0, time.Local),
				Period:      "第1-2节", Topic: "用户故事",
			}, nil
		},
	}
	recordings := &fakeRecordingRepoForProcess{transcript: &model.Transcript{
		ID: 1, SessionID: 42, Content: "今天我们讲用户故事。", Status: model.TranscriptDone,
	}}
	reporter := &fakeReportEnqueuer{}
	svc := NewAiTaskService(tasks, sessions, &fakeEvalRepo{}, recordings, &fakeAnalyzerForProcess{},
		nil, config.AgentConfig{Timeout: "3m", MaxConcurrency: 1, ReportEnabled: true},
		config.EvaluationConfig{}, reporter)

	svc.(*aiTaskService).process(context.Background(), task.ID)

	assert.Equal(t, model.AiTaskStatusDone, task.Status)
	assert.Equal(t, 1, reporter.calls)
	assert.Equal(t, uint64(2), reporter.teacherID)
	assert.Equal(t, "2026-2027-1", reporter.semester)
	assert.False(t, reporter.force)
}

// TestProcessSkipsTeacherReportWhenDisabled 验证 reportEnabled=false 时 S1 不触发 S2。
func TestProcessSkipsTeacherReportWhenDisabled(t *testing.T) {
	task := &model.AiTask{
		ID: 16, Kind: model.AiTaskKindSessionEval, RefType: model.AiTaskRefSession, RefID: 42,
		Status: model.AiTaskStatusPending,
	}
	tasks := &fakeAiTaskRepoForProcess{task: task}
	sessions := &fakeSessionRepo{
		getDetail: func(context.Context, uint64) (*repository.SessionDetailRow, error) {
			return &repository.SessionDetailRow{
				ID: 42, CourseID: 9, CourseName: "软件工程", TeacherID: 2, TeacherName: "李老师",
				Semester:    "2026-2027-1",
				SessionDate: time.Date(2026, 10, 20, 0, 0, 0, 0, time.Local),
				Period:      "第1-2节", Topic: "用户故事",
			}, nil
		},
	}
	recordings := &fakeRecordingRepoForProcess{transcript: &model.Transcript{
		ID: 1, SessionID: 42, Content: "今天我们讲用户故事。", Status: model.TranscriptDone,
	}}
	reporter := &fakeReportEnqueuer{}
	svc := NewAiTaskService(tasks, sessions, &fakeEvalRepo{}, recordings, &fakeAnalyzerForProcess{},
		nil, config.AgentConfig{Timeout: "3m", MaxConcurrency: 1, ReportEnabled: false},
		config.EvaluationConfig{}, reporter)

	svc.(*aiTaskService).process(context.Background(), task.ID)

	assert.Equal(t, model.AiTaskStatusDone, task.Status)
	assert.Equal(t, 0, reporter.calls)
}

// TestProcessStopsAtMaxAttempts 锁定 P1.5：attempts 已达上限时不得置 running、
// 不得调用 analyzer，直接 failed 并正常写 finished_at。
func TestProcessStopsAtMaxAttempts(t *testing.T) {
	task := &model.AiTask{
		ID: 9, Kind: model.AiTaskKindSessionEval, RefType: model.AiTaskRefSession, RefID: 42,
		Status: model.AiTaskStatusPending, Attempts: model.AiTaskMaxAttempts,
	}
	tasks := &fakeAiTaskRepoForProcess{task: task}
	analyzer := &fakeAnalyzerForProcess{}
	svc := NewAiTaskService(tasks, &fakeSessionRepo{}, &fakeEvalRepo{}, &fakeRecordingRepoForProcess{},
		analyzer, nil, config.AgentConfig{Timeout: "3m", MaxConcurrency: 1}, config.EvaluationConfig{})

	svc.(*aiTaskService).process(context.Background(), task.ID)

	assert.Equal(t, model.AiTaskStatusFailed, task.Status)
	assert.Contains(t, task.ErrorMessage, "AI 评析重试次数已达上限")
	assert.Equal(t, 0, analyzer.calls, "达到上限后不得调用模型")

	var ran, finished bool
	for _, fields := range tasks.updates {
		if fields["status"] == model.AiTaskStatusRunning {
			ran = true
		}
		if _, ok := fields["finished_at"]; ok {
			finished = true
		}
	}
	assert.False(t, ran, "上限检查必须在置 running 之前")
	assert.True(t, finished, "failed 状态必须写 finished_at")
}

// fakeAiTaskRepoForReset 用于 Retry/enqueue 复位测试。
// GetByID 第一次返回任务，之后返回 not implemented，避免 Retry 派生的 process goroutine 干扰断言。
type fakeAiTaskRepoForReset struct {
	mu        sync.Mutex
	task      *model.AiTask
	updates   []map[string]any
	getByIDN  int
	getByRefN int
}

func (f *fakeAiTaskRepoForReset) GetByID(context.Context, uint64) (*model.AiTask, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.getByIDN++
	if f.getByIDN > 1 {
		return nil, repository.ErrNotImplemented
	}
	return f.task, nil
}

func (f *fakeAiTaskRepoForReset) GetByKindRef(context.Context, string, string, uint64) (*model.AiTask, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.getByRefN++
	return f.task, nil
}

func (f *fakeAiTaskRepoForReset) CreateIfAbsent(context.Context, *model.AiTask) error { return nil }

func (f *fakeAiTaskRepoForReset) Update(_ context.Context, _ uint64, fields map[string]any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.updates = append(f.updates, fields)
	return nil
}

func (f *fakeAiTaskRepoForReset) ListResumable(context.Context) ([]model.AiTask, error) {
	return nil, nil
}

func (f *fakeAiTaskRepoForReset) ListDueTeacherReportTasks(context.Context, time.Time) ([]model.AiTask, error) {
	return nil, nil
}

// TestRetryResetsAttempts 锁定 P1.5：人工 Retry 必须把 attempts 复位为 0 并返回 pending。
func TestRetryResetsAttempts(t *testing.T) {
	task := &model.AiTask{
		ID: 10, Kind: model.AiTaskKindSessionEval, RefType: model.AiTaskRefSession, RefID: 42,
		Status: model.AiTaskStatusFailed, Attempts: model.AiTaskMaxAttempts,
	}
	repo := &fakeAiTaskRepoForReset{task: task}
	svc := NewAiTaskService(repo, &fakeSessionRepo{}, &fakeEvalRepo{}, &fakeRecordingRepoForProcess{},
		nil, nil, config.AgentConfig{Timeout: "3m", MaxConcurrency: 1}, config.EvaluationConfig{})

	got, err := svc.Retry(context.Background(), task.ID)
	require.NoError(t, err)
	assert.Equal(t, model.AiTaskStatusPending, got.Status)
	assert.Equal(t, 0, got.Attempts)

	repo.mu.Lock()
	defer repo.mu.Unlock()
	require.Len(t, repo.updates, 1)
	assert.Equal(t, 0, repo.updates[0]["attempts"])
	assert.Equal(t, model.AiTaskStatusPending, repo.updates[0]["status"])
}

// TestEnqueueResetsAttemptsOnTerminal 锁定 P1.5：终态任务重新入队时 attempts 同时清零。
func TestEnqueueResetsAttemptsOnTerminal(t *testing.T) {
	task := &model.AiTask{
		ID: 11, Kind: model.AiTaskKindSessionEval, RefType: model.AiTaskRefSession, RefID: 42,
		Status: model.AiTaskStatusFailed, Attempts: model.AiTaskMaxAttempts,
	}
	repo := &fakeAiTaskRepoForReset{task: task}
	svc := NewAiTaskService(repo, &fakeSessionRepo{}, &fakeEvalRepo{}, &fakeRecordingRepoForProcess{},
		nil, nil, config.AgentConfig{Timeout: "3m", MaxConcurrency: 1}, config.EvaluationConfig{})

	got, started, err := svc.(*aiTaskService).enqueue(context.Background(), 42)
	require.NoError(t, err)
	assert.True(t, started)
	assert.Equal(t, model.AiTaskStatusPending, got.Status)
	assert.Equal(t, uint8(0), got.Attempts)

	repo.mu.Lock()
	defer repo.mu.Unlock()
	require.Len(t, repo.updates, 1)
	assert.Equal(t, 0, repo.updates[0]["attempts"])
}

// captureSlog 把默认 logger 临时重定向到内存，用于断言 truncated 日志。
func captureSlog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

func processWithTranscript(t *testing.T, tr *model.Transcript) (*fakeAnalyzerForProcess, *fakeAiTaskRepoForProcess) {
	t.Helper()
	task := &model.AiTask{
		ID: 12, Kind: model.AiTaskKindSessionEval, RefType: model.AiTaskRefSession, RefID: 42,
		Status: model.AiTaskStatusPending,
	}
	tasks := &fakeAiTaskRepoForProcess{task: task}
	sessions := &fakeSessionRepo{
		getDetail: func(context.Context, uint64) (*repository.SessionDetailRow, error) {
			return &repository.SessionDetailRow{
				ID: 42, CourseID: 9, CourseName: "软件工程", TeacherName: "李老师",
				SessionDate: time.Date(2026, 10, 20, 0, 0, 0, 0, time.Local),
				Period:      "第1-2节", Topic: "用户故事",
			}, nil
		},
	}
	recordings := &fakeRecordingRepoForProcess{transcript: tr}
	analyzer := &fakeAnalyzerForProcess{}
	svc := NewAiTaskService(tasks, sessions, &fakeEvalRepo{}, recordings, analyzer, nil,
		config.AgentConfig{Timeout: "3m", MaxConcurrency: 1}, config.EvaluationConfig{})

	svc.(*aiTaskService).process(context.Background(), task.ID)
	require.Equal(t, model.AiTaskStatusDone, task.Status)
	return analyzer, tasks
}

// TestProcessTruncatesLongContentWithoutSegments 锁定 P1.6：无 segments 时按 rune 截断 Content，
// analyzer 收到的内容不超过 80k，日志含 truncated 与原始长度。
func TestProcessTruncatesLongContentWithoutSegments(t *testing.T) {
	logs := captureSlog(t)
	content := strings.Repeat("课", maxEvalRunes+123)
	analyzer, _ := processWithTranscript(t, &model.Transcript{
		ID: 1, SessionID: 42, Content: content, Status: model.TranscriptDone,
	})

	require.NotNil(t, analyzer.got)
	assert.Len(t, []rune(analyzer.got.Transcript), maxEvalRunes)
	assert.Empty(t, analyzer.got.Segments)
	assert.Contains(t, logs.String(), "transcript truncated")
	assert.Contains(t, logs.String(), fmt.Sprintf("originalRunes=%d", maxEvalRunes+123))
	assert.Contains(t, logs.String(), fmt.Sprintf("limit=%d", maxEvalRunes))
}

// TestProcessTruncatesSegmentsConsistently 锁定 P1.6：有 segments 时累计截断 segments，
// Transcript 必须由保留的 segments 拼接而成，两者都不超过上限。
func TestProcessTruncatesSegmentsConsistently(t *testing.T) {
	logs := captureSlog(t)
	first := strings.Repeat("甲", 50000)
	second := strings.Repeat("乙", 50000)
	rawSegments, err := json.Marshal([]dto.TranscriptSegment{
		{Start: 0, End: 10, Text: first},
		{Start: 10, End: 20, Text: second},
	})
	require.NoError(t, err)
	segJSON := string(rawSegments)

	analyzer, _ := processWithTranscript(t, &model.Transcript{
		ID: 2, SessionID: 42, Content: first + second, Segments: &segJSON,
		Status: model.TranscriptDone,
	})

	require.NotNil(t, analyzer.got)
	assert.Len(t, []rune(analyzer.got.Transcript), maxEvalRunes)
	require.Len(t, analyzer.got.Segments, 2)
	total := 0
	var rebuilt strings.Builder
	for _, seg := range analyzer.got.Segments {
		total += len([]rune(seg.Text))
		rebuilt.WriteString(seg.Text)
	}
	assert.Equal(t, maxEvalRunes, total)
	assert.Equal(t, analyzer.got.Transcript, rebuilt.String(), "Transcript 必须与截断后的 segments 一致")
	assert.Equal(t, first+string([]rune(second)[:maxEvalRunes-len([]rune(first))]), analyzer.got.Transcript)
	assert.Contains(t, logs.String(), "transcript truncated")
	assert.Contains(t, logs.String(), fmt.Sprintf("originalRunes=%d", 100000))
}
