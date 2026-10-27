package service

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"aijiaoxue-api/internal/agent"
	"aijiaoxue-api/internal/config"
	"aijiaoxue-api/internal/model"
	"aijiaoxue-api/internal/repository"
	"aijiaoxue-api/pkg/errcode"
)

type fakeReportTaskRepo struct {
	mu        sync.Mutex
	task      *model.AiTask
	updates   []map[string]any
	due       []model.AiTask
	resumable []model.AiTask
}

func (f *fakeReportTaskRepo) GetByID(context.Context, uint64) (*model.AiTask, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.task == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return f.task, nil
}

func (f *fakeReportTaskRepo) GetByKindRef(context.Context, string, string, uint64) (*model.AiTask, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.task == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return f.task, nil
}

func (f *fakeReportTaskRepo) CreateIfAbsent(_ context.Context, task *model.AiTask) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.task == nil {
		task.ID = 1
		f.task = task
	}
	return nil
}

func (f *fakeReportTaskRepo) Update(_ context.Context, _ uint64, fields map[string]any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.updates = append(f.updates, fields)
	if f.task == nil {
		return nil
	}
	if v, ok := fields["status"].(string); ok {
		f.task.Status = v
	}
	if v, ok := fields["error_message"].(string); ok {
		f.task.ErrorMessage = v
	}
	if v, ok := fields["payload"].(string); ok {
		f.task.Payload = &v
	}
	if v, ok := fields["updated_at"].(time.Time); ok {
		f.task.UpdatedAt = v
	}
	return nil
}

func (f *fakeReportTaskRepo) ListResumable(context.Context) ([]model.AiTask, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.resumable, nil
}

func (f *fakeReportTaskRepo) ListDueTeacherReportTasks(context.Context, time.Time) ([]model.AiTask, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.due, nil
}

type fakeTeacherReportRepo struct {
	mu      sync.Mutex
	report  *model.TeacherAiReport
	upserts []*model.TeacherAiReport
}

func (f *fakeTeacherReportRepo) GetByTeacherSemester(context.Context, uint64, string) (*model.TeacherAiReport, error) {
	return f.Get(context.Background(), 0, "")
}

func (f *fakeTeacherReportRepo) Get(context.Context, uint64, string) (*model.TeacherAiReport, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.report == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return f.report, nil
}

func (f *fakeTeacherReportRepo) Upsert(_ context.Context, report *model.TeacherAiReport) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	row := *report
	f.report = &row
	f.upserts = append(f.upserts, &row)
	return nil
}

func (f *fakeTeacherReportRepo) MarkStatus(_ context.Context, _ uint64, _ string, status, errMsg string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.report != nil {
		f.report.Status = status
		f.report.ErrorMessage = errMsg
	}
	return nil
}

type fakeReportAnalyzer struct {
	mu     sync.Mutex
	calls  int
	got    *agent.ReportInput
	result *agent.ReportResult
	err    error
}

func (f *fakeReportAnalyzer) AnalyzeReport(_ context.Context, in agent.ReportInput) (*agent.ReportResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	f.got = &in
	if f.err != nil {
		return nil, f.err
	}
	if f.result != nil {
		return f.result, nil
	}
	return &agent.ReportResult{
		Summary: "综合报告", Directions: []string{"方向A"}, Suggestions: []string{"建议A"},
		Highlights: "亮点", TrendComment: "趋势向上", Confidence: 0.8,
	}, nil
}

func testReportService(
	tasks *fakeReportTaskRepo, reports *fakeTeacherReportRepo, evals *fakeEvalRepo, analyzer *fakeReportAnalyzer,
) ReportService {
	users := &fakeUserRepo{getByID: func(_ context.Context, id uint64) (*model.User, error) {
		dept := uint64(1)
		return &model.User{ID: id, Name: "李明", Role: model.RoleTeacher, DepartmentID: &dept}, nil
	}}
	return NewReportService(tasks, reports, evals, users, analyzer,
		config.AgentConfig{
			Enabled: true, LLMModel: "fake-report-model", PromptVersion: "v1",
			Timeout: "3m", ReportDebounce: "10m", ReportEnabled: true,
		}, evalCfg())
}

func reportTaskPayloadPtr(t *testing.T, semester string, force bool) *string {
	t.Helper()
	raw, err := json.Marshal(reportTaskPayload{Version: 1, Semester: semester, Force: force})
	require.NoError(t, err)
	s := string(raw)
	return &s
}

func reportRowsFixture() []repository.TeacherReportRow {
	u8 := func(v uint8) *uint8 { return &v }
	f := func(v float64) *float64 { return &v }
	return []repository.TeacherReportRow{
		{
			SessionID: 1, CourseID: 1, CourseCode: "SE3101", CourseName: "软件项目管理",
			SessionDate: time.Date(2026, 9, 12, 0, 0, 0, 0, time.UTC), Period: "3-4 节", Topic: "项目立项",
			ObjectiveScore: u8(4), ContentScore: u8(3), InteractionScore: u8(2),
			OrganizationScore: u8(4), FrontierScore: u8(3),
			Comment: "AI 评语", Highlights: "亮点", Improvements: "改进", Suggestions: "建议",
			AIModelVersion: "qwen", AIConfidence: f(0.7),
		},
		{
			SessionID: 2, CourseID: 1, CourseCode: "SE3101", CourseName: "软件项目管理",
			SessionDate: time.Date(2026, 9, 19, 0, 0, 0, 0, time.UTC), Period: "3-4 节", Topic: "需求调研",
			ObjectiveScore: u8(5), ContentScore: u8(4), InteractionScore: u8(3),
			OrganizationScore: u8(4), FrontierScore: u8(3),
			Comment: "AI 评语2", AIModelVersion: "qwen", AIConfidence: f(0.8),
		},
	}
}

func sessionDimRowsFixture() []repository.SessionDimRow {
	f := func(v float64) *float64 { return &v }
	return []repository.SessionDimRow{
		{
			SessionID: 1, TeacherID: 2, CourseID: 1,
			SessionDate: time.Date(2026, 9, 12, 0, 0, 0, 0, time.UTC),
			HasAgent:    true, AiObjective: f(4), AiContent: f(3), AiInteraction: f(2),
			AiOrganization: f(4), AiFrontier: f(3),
		},
		{
			SessionID: 2, TeacherID: 2, CourseID: 1,
			SessionDate: time.Date(2026, 9, 19, 0, 0, 0, 0, time.UTC),
			HasAgent:    true, AiObjective: f(5), AiContent: f(4), AiInteraction: f(3),
			AiOrganization: f(4), AiFrontier: f(3),
		},
	}
}

func TestReportProcessSkipsUnchangedSnapshot(t *testing.T) {
	task := &model.AiTask{
		ID: 11, Kind: model.AiTaskKindTeacherReport, RefType: model.AiTaskRefTeacher, RefID: 2,
		Status: model.AiTaskStatusPending, Payload: reportTaskPayloadPtr(t, "2026-2027-1", false),
	}
	tasks := &fakeReportTaskRepo{task: task}
	sourceIDs, _ := json.Marshal([]uint64{1, 2})
	sourceStr := string(sourceIDs)
	reports := &fakeTeacherReportRepo{report: &model.TeacherAiReport{
		TeacherID: 2, Semester: "2026-2027-1", CourseID: 0,
		Status: model.AiTaskStatusDone, SourceSessionIDs: &sourceStr,
	}}
	evals := &fakeEvalRepo{listTeacherReportRows: func(
		context.Context, uint64, string, string,
	) ([]repository.TeacherReportRow, error) {
		return reportRowsFixture(), nil
	}}
	analyzer := &fakeReportAnalyzer{}
	svc := testReportService(tasks, reports, evals, analyzer)

	svc.ProcessTask(context.Background(), task.ID)

	assert.Equal(t, 0, analyzer.calls, "快照未变时不得调用模型")
	assert.Equal(t, model.AiTaskStatusDone, task.Status)
	assert.Empty(t, reports.upserts, "快照未变时不得覆盖报告")
}

func TestReportProcessForceRegenerates(t *testing.T) {
	task := &model.AiTask{
		ID: 12, Kind: model.AiTaskKindTeacherReport, RefType: model.AiTaskRefTeacher, RefID: 2,
		Status: model.AiTaskStatusPending, Payload: reportTaskPayloadPtr(t, "2026-2027-1", true),
	}
	tasks := &fakeReportTaskRepo{task: task}
	sourceIDs, _ := json.Marshal([]uint64{1, 2})
	sourceStr := string(sourceIDs)
	reports := &fakeTeacherReportRepo{report: &model.TeacherAiReport{
		TeacherID: 2, Semester: "2026-2027-1", CourseID: 0,
		Status: model.AiTaskStatusDone, SourceSessionIDs: &sourceStr,
	}}
	evals := &fakeEvalRepo{
		listTeacherReportRows: func(context.Context, uint64, string, string) ([]repository.TeacherReportRow, error) {
			return reportRowsFixture(), nil
		},
		listSessionDimRows: func(context.Context, repository.SessionDimFilter) ([]repository.SessionDimRow, error) {
			return sessionDimRowsFixture(), nil
		},
		listSupervisorCommentsByTeacherSemester: func(context.Context, uint64, string) ([]string, error) {
			return []string{"督导评语"}, nil
		},
	}
	analyzer := &fakeReportAnalyzer{}
	svc := testReportService(tasks, reports, evals, analyzer)

	svc.ProcessTask(context.Background(), task.ID)

	assert.Equal(t, 1, analyzer.calls, "force 必须跳过快照短路")
	require.NotNil(t, analyzer.got)
	assert.Equal(t, 2, analyzer.got.EvaluatedCount)
	assert.Equal(t, 2, analyzer.got.SessionCount)
	assert.Equal(t, []string{"督导评语"}, analyzer.got.SupervisorComments)
	assert.Equal(t, model.AiTaskStatusDone, task.Status)
	require.NotNil(t, reports.report)
	assert.Equal(t, model.AiTaskStatusDone, reports.report.Status)
	assert.Equal(t, "综合报告", stringValue(reports.report.Summary))
	assert.JSONEq(t, `["方向A"]`, stringValue(reports.report.Directions))
	assert.JSONEq(t, `["建议A"]`, stringValue(reports.report.Suggestions))
	assert.Equal(t, []uint64{1, 2}, decodeUint64Array(reports.report.SourceSessionIDs))
}

func TestReportProcessFailureWritesFailedStatus(t *testing.T) {
	task := &model.AiTask{
		ID: 13, Kind: model.AiTaskKindTeacherReport, RefType: model.AiTaskRefTeacher, RefID: 2,
		Status: model.AiTaskStatusPending, Payload: reportTaskPayloadPtr(t, "2026-2027-1", false),
	}
	tasks := &fakeReportTaskRepo{task: task}
	reports := &fakeTeacherReportRepo{}
	evals := &fakeEvalRepo{
		listTeacherReportRows: func(context.Context, uint64, string, string) ([]repository.TeacherReportRow, error) {
			return reportRowsFixture(), nil
		},
		listSessionDimRows: func(context.Context, repository.SessionDimFilter) ([]repository.SessionDimRow, error) {
			return sessionDimRowsFixture(), nil
		},
		listSupervisorCommentsByTeacherSemester: func(context.Context, uint64, string) ([]string, error) {
			return nil, nil
		},
	}
	analyzer := &fakeReportAnalyzer{err: context.DeadlineExceeded}
	svc := testReportService(tasks, reports, evals, analyzer)

	svc.ProcessTask(context.Background(), task.ID)

	assert.Equal(t, model.AiTaskStatusFailed, task.Status)
	assert.Contains(t, task.ErrorMessage, "AI 报告调用失败")
	require.NotNil(t, reports.report)
	assert.Equal(t, model.AiTaskStatusFailed, reports.report.Status)
	assert.Contains(t, reports.report.ErrorMessage, "AI 报告调用失败")
}

func TestReportRecoverStuckOnlyResetsTeacherReportRunning(t *testing.T) {
	tasks := &fakeReportTaskRepo{resumable: []model.AiTask{
		{ID: 1, Kind: model.AiTaskKindTeacherReport, Status: model.AiTaskStatusRunning},
		{ID: 2, Kind: model.AiTaskKindTeacherReport, Status: model.AiTaskStatusPending},
		{ID: 3, Kind: model.AiTaskKindSessionEval, Status: model.AiTaskStatusRunning},
	}}
	svc := testReportService(tasks, &fakeTeacherReportRepo{}, &fakeEvalRepo{}, &fakeReportAnalyzer{})

	svc.(*reportService).recoverStuck(context.Background())

	tasks.mu.Lock()
	defer tasks.mu.Unlock()
	require.Len(t, tasks.updates, 1)
	assert.Equal(t, model.AiTaskStatusPending, tasks.updates[0]["status"])
	due, ok := tasks.updates[0]["updated_at"].(time.Time)
	require.True(t, ok)
	assert.True(t, due.Before(time.Now().Add(-9*time.Minute)))
}

func TestReportEnqueueReusesTaskAndForceMakesItDue(t *testing.T) {
	existing := &model.AiTask{
		ID: 14, Kind: model.AiTaskKindTeacherReport, RefType: model.AiTaskRefTeacher, RefID: 2,
		Status: model.AiTaskStatusFailed, UpdatedAt: time.Now(),
	}
	tasks := &fakeReportTaskRepo{task: existing}
	svc := testReportService(tasks, &fakeTeacherReportRepo{}, &fakeEvalRepo{}, &fakeReportAnalyzer{})

	require.NoError(t, svc.Enqueue(2, "2026-2027-1", false))
	require.NoError(t, svc.Enqueue(2, "2026-2027-1", true))

	tasks.mu.Lock()
	defer tasks.mu.Unlock()
	require.Len(t, tasks.updates, 2, "同一 teacher 只复用一条任务")
	assert.Equal(t, model.AiTaskStatusPending, existing.Status)
	last := tasks.updates[1]
	assert.Equal(t, 0, last["attempts"])
	var payload reportTaskPayload
	require.NoError(t, json.Unmarshal([]byte(last["payload"].(string)), &payload))
	assert.True(t, payload.Force)
	due, ok := last["updated_at"].(time.Time)
	require.True(t, ok)
	assert.True(t, due.Before(time.Now().Add(-9*time.Minute)), "force 任务必须立即到期")
}

func TestReportGetReportNoRowAndPermissions(t *testing.T) {
	tasks := &fakeReportTaskRepo{}
	reports := &fakeTeacherReportRepo{}
	svc := testReportService(tasks, reports, &fakeEvalRepo{}, &fakeReportAnalyzer{})

	t.Run("无报告返回 pending + null", func(t *testing.T) {
		got, err := svc.GetReport(context.Background(), RoleSupervisor, 0, 0, 2, "")
		require.NoError(t, err)
		assert.Equal(t, model.AiTaskStatusPending, got.Status)
		assert.Nil(t, got.Report)
	})

	t.Run("教师看他人 40302", func(t *testing.T) {
		_, err := svc.GetReport(context.Background(), RoleTeacher, 0, 3, 2, "")
		require.Error(t, err)
		assert.Equal(t, errcode.ForbiddenData, errcode.From(err).Code)
	})

	t.Run("主任看他室 40302", func(t *testing.T) {
		_, err := svc.GetReport(context.Background(), RoleDirector, 2, 99, 2, "")
		require.Error(t, err)
		assert.Equal(t, errcode.ForbiddenData, errcode.From(err).Code)
	})
}

func TestReportGetMapsExistingRow(t *testing.T) {
	summary := "本学期共 2 次课"
	highlights := "组织稳定"
	trend := "趋势向上"
	directions := `["互动偏弱"]`
	suggestions := `["延长等待时间"]`
	sourceIDs := `[1,2]`
	sample := `{"evaluatedCount":2,"sessionCount":2,"flags":["ai_only"]}`
	confidence := 0.8
	generated := time.Date(2026, 10, 27, 10, 0, 0, 0, time.UTC)
	reports := &fakeTeacherReportRepo{report: &model.TeacherAiReport{
		TeacherID: 2, Semester: "2026-2027-1", CourseID: 0,
		Summary: &summary, Directions: &directions, Suggestions: &suggestions,
		Highlights: &highlights, TrendComment: &trend, SourceSessionIDs: &sourceIDs,
		Sample: &sample, Confidence: &confidence, Status: model.AiTaskStatusDone,
		AIModelVersion: "fake-model", PromptVersion: "v1", GeneratedAt: &generated,
	}}
	svc := testReportService(&fakeReportTaskRepo{}, reports, &fakeEvalRepo{}, &fakeReportAnalyzer{})

	got, err := svc.GetReport(context.Background(), RoleSupervisor, 0, 9, 2, "2026-2027-1")
	require.NoError(t, err)
	require.NotNil(t, got.Report)
	assert.Equal(t, model.AiTaskStatusDone, got.Status)
	assert.Equal(t, "本学期共 2 次课", got.Report.Summary)
	assert.Equal(t, []string{"互动偏弱"}, got.Report.Directions)
	assert.Equal(t, []string{"延长等待时间"}, got.Report.Suggestions)
	assert.Equal(t, 2, got.Report.EvaluatedCount)
	assert.Equal(t, 2, got.Report.SessionCount)
	assert.Equal(t, []uint64{1, 2}, got.Report.SourceSessionIDs)
	assert.InDelta(t, 0.8, *got.Report.Confidence, 1e-9)
	assert.Equal(t, generated.Format(time.RFC3339), got.Report.GeneratedAt)
}

func TestReportRegeneratePermissionsAndPending(t *testing.T) {
	tasks := &fakeReportTaskRepo{}
	reports := &fakeTeacherReportRepo{}
	svc := testReportService(tasks, reports, &fakeEvalRepo{}, &fakeReportAnalyzer{})

	t.Run("教师 40301", func(t *testing.T) {
		_, err := svc.Regenerate(context.Background(), RoleTeacher, 1, 2, 2, "")
		require.Error(t, err)
		assert.Equal(t, errcode.ForbiddenRole, errcode.From(err).Code)
	})

	t.Run("主任他室 40302", func(t *testing.T) {
		_, err := svc.Regenerate(context.Background(), RoleDirector, 2, 99, 2, "")
		require.Error(t, err)
		assert.Equal(t, errcode.ForbiddenData, errcode.From(err).Code)
	})

	t.Run("督导返回 pending 并创建任务", func(t *testing.T) {
		got, err := svc.Regenerate(context.Background(), RoleSupervisor, 0, 9, 2, "")
		require.NoError(t, err)
		assert.Equal(t, model.AiTaskStatusPending, got.Status)
		require.NotNil(t, reports.report)
		assert.Equal(t, model.AiTaskStatusPending, reports.report.Status)
		require.NotNil(t, tasks.task)
		var payload reportTaskPayload
		require.NoError(t, json.Unmarshal([]byte(*tasks.task.Payload), &payload))
		assert.True(t, payload.Force)
	})
}

func TestReportDTOArraysAreNeverNull(t *testing.T) {
	empty := "null"
	report := &model.TeacherAiReport{
		TeacherID: 2, Semester: "2026-2027-1", Status: model.AiTaskStatusDone,
		Directions: &empty, Suggestions: &empty, SourceSessionIDs: &empty,
	}
	got := toAiReportDTO(report)
	assert.NotNil(t, got.Directions)
	assert.Empty(t, got.Directions)
	assert.NotNil(t, got.Suggestions)
	assert.Empty(t, got.Suggestions)
	assert.NotNil(t, got.SourceSessionIDs)
	assert.Empty(t, got.SourceSessionIDs)
	assert.Equal(t, []string{}, got.Directions)
}
