package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"aijiaoxue-api/internal/dto"
	"aijiaoxue-api/internal/model"
	"aijiaoxue-api/internal/repository"
	"aijiaoxue-api/pkg/errcode"
)

// captureNotifier 记录 B12 督导评语通知的场次 id。
type captureNotifier struct {
	sessionIDs []uint64
}

func (n *captureNotifier) NotifySupervisorComment(_ context.Context, sessionID uint64) {
	n.sessionIDs = append(n.sessionIDs, sessionID)
}

func (n *captureNotifier) NotifyAIIssue(context.Context, uint64, string) {}

func uint8Ptr(v uint8) *uint8 { return &v }

func float64Ptr(v float64) *float64 { return &v }

// newScoringFixture 构造一场「已有 AI 终评」的场次，并用内存 map 模拟
// evaluations 的 uk_eval 覆盖语义；ListSessionDimRows 会随写入变化，
// 从而能验证督导评分不会影响 AI 单源聚合。
func newScoringFixture(t *testing.T) (*fakeSessionRepo, *fakeEvalRepo, *model.Evaluation) {
	t.Helper()

	agentTotal := 75.0
	agent := &model.Evaluation{
		SessionID:         7,
		EvaluatorType:     model.EvaluatorAgent,
		EvaluatorID:       0,
		FormulaVersion:    "v1",
		ObjectiveScore:    uint8Ptr(4),
		ContentScore:      uint8Ptr(4),
		InteractionScore:  uint8Ptr(4),
		OrganizationScore: uint8Ptr(4),
		FrontierScore:     uint8Ptr(4),
		TotalScore:        &agentTotal,
		Comment:           "AI 评价",
	}
	supervisors := map[uint64]*model.Evaluation{}

	sessions := &fakeSessionRepo{
		getDetail: func(context.Context, uint64) (*repository.SessionDetailRow, error) {
			return sessionRow(), nil
		},
	}
	evals := &fakeEvalRepo{
		upsert: func(_ context.Context, e *model.Evaluation) error {
			copied := *e
			supervisors[e.EvaluatorID] = &copied
			return nil
		},
		listBySession: func(context.Context, uint64) ([]repository.EvaluationRow, error) {
			rows := []repository.EvaluationRow{
				{Evaluation: *agent},
			}
			for _, e := range supervisors {
				rows = append(rows, repository.EvaluationRow{Evaluation: *e, EvaluatorName: "陈静"})
			}
			return rows, nil
		},
		listSessionDimRows: func(context.Context, repository.SessionDimFilter) ([]repository.SessionDimRow, error) {
			row := repository.SessionDimRow{
				SessionID: 7, TeacherID: 2, CourseID: 1,
				SessionDate: time.Date(2026, 9, 20, 0, 0, 0, 0, time.Local),
				Period:      "3-4 节", Topic: "验收主题", Status: "evaluated",
				CourseCode: "SE3101", CourseName: "软件项目管理",
				HasAgent:    true,
				AiObjective: float64Ptr(4), AiContent: float64Ptr(4),
				AiInteraction: float64Ptr(4), AiOrganization: float64Ptr(4),
				AiFrontier: float64Ptr(4),
			}
			for _, e := range supervisors {
				row.HasSupervisor = true
				row.SupObjective = float64FromUint8(e.ObjectiveScore)
				row.SupContent = float64FromUint8(e.ContentScore)
				row.SupInteraction = float64FromUint8(e.InteractionScore)
				row.SupOrganization = float64FromUint8(e.OrganizationScore)
				row.SupFrontier = float64FromUint8(e.FrontierScore)
			}
			return []repository.SessionDimRow{row}, nil
		},
	}
	return sessions, evals, agent
}

func float64FromUint8(v *uint8) *float64 {
	if v == nil {
		return nil
	}
	f := float64(*v)
	return &f
}

func scoringUsers() *fakeUserRepo {
	dept := uint64(1)
	return &fakeUserRepo{getByID: func(_ context.Context, id uint64) (*model.User, error) {
		return &model.User{ID: id, Name: "李明", Role: model.RoleTeacher, DepartmentID: &dept}, nil
	}}
}

func scoringCourses() *fakeCourseRepo {
	return &fakeCourseRepo{getDetail: func(_ context.Context, id uint64) (*repository.CourseRow, error) {
		return &repository.CourseRow{
			ID: id, Code: "SE3101", Name: "软件项目管理",
			TeacherID: 2, DepartmentID: 1, Semester: "2026-2027-1",
		}, nil
	}}
}

func supervisorScoreReq(objective, content, interaction, organization, frontier int) dto.SupervisorEvaluationReq {
	return dto.SupervisorEvaluationReq{
		Objective: &objective, Content: &content, Interaction: &interaction,
		Organization: &organization, Frontier: &frontier,
		Comment:      "总体评语",
		Highlights:   "亮点",
		Improvements: "待改进",
		Suggestions:  "建议",
	}
}

// TestSubmitSupervisorEvaluationKeepsAISingleSource 是本轮 D3/D6/B8 的核心回归：
// 督导五维评分只写 supervisor 行，不改变 status，也不影响 agentScore / composite / score-trend。
func TestSubmitSupervisorEvaluationKeepsAISingleSource(t *testing.T) {
	sessions, evals, agent := newScoringFixture(t)
	users, courses := scoringUsers(), scoringCourses()
	notifier := &captureNotifier{}
	svc := NewSessionService(sessions, evals, courses, users, evalCfg(), notifier)
	teacherSvc := NewTeacherScoreService(users, courses, evals, evalCfg())
	improveSvc := NewAgentImproveService(users, courses, evals, evalCfg())

	ctx := context.Background()
	before, err := svc.Evaluation(ctx, RoleSupervisor, 0, 9, 7)
	require.NoError(t, err)
	assert.False(t, before.SupervisorEvaluated)
	require.NotNil(t, before.AgentScore)
	assert.Equal(t, 75.0, *before.AgentScore.TotalScore)

	summaryBefore, err := teacherSvc.TeacherSummary(ctx, RoleSupervisor, 0, 9, 2, "2026-2027-1")
	require.NoError(t, err)
	require.NotNil(t, summaryBefore.CompositeScore)
	trendBefore, err := improveSvc.TeacherScoreTrend(ctx, RoleSupervisor, 0, 9, 2,
		dto.ScoreTrendQuery{Semester: "2026-2027-1"})
	require.NoError(t, err)
	require.Len(t, trendBefore, 6)
	require.Len(t, trendBefore[0].Points, 1)
	require.NotNil(t, trendBefore[0].Points[0].Value)

	got, err := svc.SubmitSupervisorEvaluation(ctx, 5, 7, supervisorScoreReq(5, 4, 4, 3, 2))
	require.NoError(t, err)
	require.NotNil(t, got.TotalScore)
	assert.Equal(t, 77.5, *got.TotalScore, "督导总分必须是 0-100")
	assert.Equal(t, 5, *got.Objective)
	assert.Equal(t, 4, *got.Content)
	assert.Empty(t, sessions.lastStatus, "status 只能由 AI 任务写为 evaluated")
	assert.Equal(t, []uint64{7}, notifier.sessionIDs, "督导评分应触发 B12 通知")

	after, err := svc.Evaluation(ctx, RoleSupervisor, 0, 9, 7)
	require.NoError(t, err)
	assert.True(t, after.SupervisorEvaluated)
	require.Len(t, after.SupervisorScores, 1)
	require.NotNil(t, after.AgentScore)
	assert.Equal(t, 75.0, *after.AgentScore.TotalScore, "agentScore 不得被督导评分改变")
	assert.Equal(t, *agent.TotalScore, *after.AgentScore.TotalScore)

	summaryAfter, err := teacherSvc.TeacherSummary(ctx, RoleSupervisor, 0, 9, 2, "2026-2027-1")
	require.NoError(t, err)
	assert.Equal(t, *summaryBefore.CompositeScore, *summaryAfter.CompositeScore, "compositeScore 不得变")
	require.NotNil(t, summaryAfter.AgentScore)
	assert.Equal(t, 75.0, *summaryAfter.AgentScore)

	trendAfter, err := improveSvc.TeacherScoreTrend(ctx, RoleSupervisor, 0, 9, 2,
		dto.ScoreTrendQuery{Semester: "2026-2027-1"})
	require.NoError(t, err)
	assert.Equal(t, trendBefore, trendAfter, "score-trend 不得因督导评分变化")
}

// TestSubmitSupervisorObservationPreservesExistingScores 覆盖旧客户端兼容路径：
// 先提交五维评分、后提交 observation，只允许更新评语，评分与总分必须保留。
func TestSubmitSupervisorObservationPreservesExistingScores(t *testing.T) {
	sessions, evals, _ := newScoringFixture(t)
	svc := NewSessionService(sessions, evals, scoringCourses(), scoringUsers(), evalCfg())

	ctx := context.Background()
	scored, err := svc.SubmitSupervisorEvaluation(ctx, 5, 7, supervisorScoreReq(5, 4, 4, 3, 2))
	require.NoError(t, err)
	require.NotNil(t, scored.TotalScore)

	observation, err := svc.SubmitSupervisorObservation(ctx, 5, 7, dto.SupervisorObservationReq{
		Comment:      "补充观察评语",
		Improvements: "补充待改进",
	})
	require.NoError(t, err)
	assert.Equal(t, "补充观察评语", observation.Comment)
	assert.Equal(t, "补充待改进", observation.Improvements)
	require.NotNil(t, observation.TotalScore)
	assert.Equal(t, 77.5, *observation.TotalScore, "observation 不得清空 total_score")
	require.NotNil(t, observation.Objective)
	require.NotNil(t, observation.Content)
	require.NotNil(t, observation.Interaction)
	require.NotNil(t, observation.Organization)
	require.NotNil(t, observation.Frontier)
	assert.Equal(t, 5, *observation.Objective)
	assert.Equal(t, 4, *observation.Content)
	assert.Equal(t, 4, *observation.Interaction)
	assert.Equal(t, 3, *observation.Organization)
	assert.Equal(t, 2, *observation.Frontier)
	assert.Empty(t, sessions.lastStatus, "observation 也不得改 status")
}

// TestSubmitSupervisorEvaluationMissingDimension 明确覆盖「新评分缺任一维度 → 40002」。
func TestSubmitSupervisorEvaluationMissingDimension(t *testing.T) {
	sessions, evals, _ := newScoringFixture(t)
	svc := NewSessionService(sessions, evals, scoringCourses(), scoringUsers(), evalCfg())

	req := supervisorScoreReq(4, 4, 4, 4, 4)
	req.Frontier = nil
	_, err := svc.SubmitSupervisorEvaluation(context.Background(), 5, 7, req)
	require.Error(t, err)
	assert.Equal(t, errcode.BizRule, errcode.From(err).Code)
}
