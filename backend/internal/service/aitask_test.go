package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"aijiaoxue-api/internal/agent"
	"aijiaoxue-api/internal/config"
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
	got *agent.EvalInput
}

func (f *fakeAnalyzerForProcess) AnalyzeSession(_ context.Context, in agent.EvalInput) (*agent.Result, error) {
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
