package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"aijiaoxue-api/internal/dto"
	"aijiaoxue-api/internal/model"
	"aijiaoxue-api/internal/repository"
	"aijiaoxue-api/pkg/errcode"
)

func u8ptr(v uint8) *uint8      { return &v }
func f64ptr(v float64) *float64 { return &v }

func mustDate(t *testing.T, value string) time.Time {
	t.Helper()
	d, err := time.Parse("2006-01-02", value)
	require.NoError(t, err)
	return d
}

func suggestionCourse() *repository.CourseRow {
	return &repository.CourseRow{
		ID: 1, Code: "SE3101", Name: "软件项目管理",
		TeacherID: 2, DepartmentID: 1, Semester: CurrentSemester,
	}
}

// TestRenderSuggestionEvidence 锁定 §8.15 evidence 渲染规则：
// 最低分非空维度第一条 quote → 「…」（mm:ss–mm:ss）；无 quote 回退 highlights；再无空串。
func TestRenderSuggestionEvidence(t *testing.T) {
	quoteEvidence := `{
	  "schemaVersion": 1,
	  "dimensions": {
	    "objective": {"confidence": 0.6, "quotes": [{"start": 5, "end": 9.9, "quote": "目标引用"}]},
	    "content":   {"confidence": 0.7, "quotes": [{"start": 62.5, "end": 75.2, "quote": "先独立估点再比对"}]},
	    "frontier":  {"confidence": 0.2, "quotes": [{"start": 120, "end": 130, "quote": "前沿引用"}]}
	  }
	}`
	emptyQuoteEvidence := `{
	  "schemaVersion": 1,
	  "dimensions": {
	    "interaction": {"confidence": 0.4, "quotes": []}
	  }
	}`
	badEvidence := `{不是合法 JSON`

	cases := []struct {
		name string
		row  repository.AgentSuggestionRow
		want string
	}{
		{
			name: "最低分非空维度取第一条 quote 并格式化时间",
			row: repository.AgentSuggestionRow{
				ObjectiveScore: u8ptr(4), ContentScore: u8ptr(2), FrontierScore: u8ptr(5),
				Evidence: &quoteEvidence,
			},
			want: "「先独立估点再比对」（01:02–01:15）",
		},
		{
			name: "最低分维度无 quote 时回退 highlights",
			row: repository.AgentSuggestionRow{
				InteractionScore: u8ptr(2),
				Evidence:         &emptyQuoteEvidence,
				Highlights:       "讨论参与度高",
			},
			want: "讨论参与度高",
		},
		{
			name: "无任何维度分时回退 highlights",
			row: repository.AgentSuggestionRow{
				Evidence:   &quoteEvidence,
				Highlights: "开篇结构完整",
			},
			want: "开篇结构完整",
		},
		{
			name: "无维度分且无 highlights 返回空串",
			row:  repository.AgentSuggestionRow{Evidence: &quoteEvidence},
			want: "",
		},
		{
			name: "evidence 脏数据按无引用处理并回退 highlights",
			row: repository.AgentSuggestionRow{
				ObjectiveScore: u8ptr(2),
				Evidence:       &badEvidence,
				Highlights:     "兜底亮点",
			},
			want: "兜底亮点",
		},
		{
			name: "同分时取维度顺序靠前者",
			row: repository.AgentSuggestionRow{
				ObjectiveScore: u8ptr(3), ContentScore: u8ptr(3),
				Evidence: &quoteEvidence,
			},
			want: "「目标引用」（00:05–00:09）",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, renderSuggestionEvidence(tc.row))
		})
	}
}

// TestCourseSuggestions 覆盖排序、只取 agent、字段映射与过滤器透传。
func TestCourseSuggestions(t *testing.T) {
	users := &fakeUserRepo{}
	courses := &fakeCourseRepo{getDetail: func(context.Context, uint64) (*repository.CourseRow, error) {
		return suggestionCourse(), nil
	}}
	evidence := `{
	  "schemaVersion": 1,
	  "dimensions": {
	    "content": {"confidence": 0.7, "quotes": [{"start": 62.5, "end": 75.2, "quote": "先独立估点再比对"}]}
	  }
	}`
	createdAt := time.Date(2026, 9, 19, 18, 5, 0, 0, time.UTC)
	rows := []repository.AgentSuggestionRow{
		{
			ID: 10, SessionID: 2, CourseID: 1, SessionDate: mustDate(t, "2026-09-19"),
			Period: "3-4 节", Topic: "需求调研与用户故事", EvaluatorType: model.EvaluatorAgent,
			Comment: "总结二", Highlights: "亮点二", Improvements: "改进二",
			Evidence: &evidence, AIModelVersion: "qwen-audio-v1", AIConfidence: f64ptr(0.72),
			CreatedAt:      createdAt,
			ObjectiveScore: u8ptr(4), ContentScore: u8ptr(3), OrganizationScore: u8ptr(4),
		},
		{
			// 同一天但 evaluation.id 更大，必须排在 ID=10 之前。
			ID: 30, SessionID: 3, CourseID: 1, SessionDate: mustDate(t, "2026-09-19"),
			Period: "3-4 节", Topic: "迭代计划与估点", EvaluatorType: model.EvaluatorAgent,
			Comment: "同日较新", Improvements: "改进三", AIModelVersion: "qwen-audio-v1",
			CreatedAt: createdAt,
		},
		{
			ID: 20, SessionID: 1, CourseID: 1, SessionDate: mustDate(t, "2026-09-12"),
			Period: "3-4 节", Topic: "项目立项与章程", EvaluatorType: model.EvaluatorAgent,
			Comment: "最早", Highlights: "兜底亮点", AIModelVersion: "qwen-audio-v1",
		},
		{
			// 督导行必须被过滤，绝不进入建议列表。
			ID: 99, SessionID: 4, CourseID: 1, SessionDate: mustDate(t, "2026-09-26"),
			EvaluatorType: model.EvaluatorSupervisor, Comment: "督导评语",
		},
	}

	var captured repository.AgentSuggestionFilter
	evals := &fakeEvalRepo{listAgentSuggestions: func(
		_ context.Context, flt repository.AgentSuggestionFilter,
	) ([]repository.AgentSuggestionRow, error) {
		captured = flt
		return rows, nil
	}}
	svc := NewAgentImproveService(users, courses, evals, evalCfg())

	got, err := svc.CourseSuggestions(context.Background(), RoleSupervisor, 0, 9, 1, "")
	require.NoError(t, err)

	assert.Equal(t, repository.AgentSuggestionFilter{
		CourseID: 1, Semester: CurrentSemester, FormulaVersion: "v1",
	}, captured)

	require.Len(t, got, 3, "督导行不得出现在建议列表")
	assert.Equal(t, []uint64{30, 10, 20}, []uint64{got[0].ID, got[1].ID, got[2].ID},
		"必须按 session_date DESC, evaluation.id DESC")

	assert.Equal(t, uint64(1), got[1].CourseID)
	assert.Equal(t, uint64(2), got[1].SessionID)
	assert.Equal(t, "2026-09-19", got[1].SessionDate)
	assert.Equal(t, "3-4 节", got[1].Period)
	assert.Equal(t, "需求调研与用户故事", got[1].Topic)
	assert.Equal(t, "总结二", got[1].Summary)
	assert.Equal(t, "亮点二", got[1].Highlights)
	assert.Equal(t, "改进二", got[1].Improvements)
	assert.Equal(t, "「先独立估点再比对」（01:02–01:15）", got[1].Evidence)
	assert.Equal(t, "qwen-audio-v1", got[1].ModelVersion)
	assert.Equal(t, 0.72, got[1].Confidence)
	assert.Equal(t, "2026-09-19T18:05:00Z", got[1].CreatedAt)

	assert.Equal(t, "", got[0].Evidence, "无维度分且无 highlights 时 evidence 为空串")
	assert.Equal(t, 0.0, got[0].Confidence, "ai_confidence 空值按 0")
	assert.Equal(t, "兜底亮点", got[2].Evidence, "无维度分时回退 highlights")
}

// TestCourseSuggestionsFiltersAndPermissions 覆盖 semester 透传、非法参数与三角色越权。
func TestCourseSuggestionsFiltersAndPermissions(t *testing.T) {
	courses := &fakeCourseRepo{getDetail: func(context.Context, uint64) (*repository.CourseRow, error) {
		return suggestionCourse(), nil
	}}
	var captured repository.AgentSuggestionFilter
	evals := &fakeEvalRepo{listAgentSuggestions: func(
		_ context.Context, flt repository.AgentSuggestionFilter,
	) ([]repository.AgentSuggestionRow, error) {
		captured = flt
		return []repository.AgentSuggestionRow{}, nil
	}}
	svc := NewAgentImproveService(&fakeUserRepo{}, courses, evals, evalCfg())

	t.Run("显式 semester 透传仓储", func(t *testing.T) {
		_, err := svc.CourseSuggestions(context.Background(), RoleSupervisor, 0, 9, 1, "2025-2026-2")
		require.NoError(t, err)
		assert.Equal(t, "2025-2026-2", captured.Semester)
	})

	t.Run("非法 semester 返回 40001", func(t *testing.T) {
		_, err := svc.CourseSuggestions(context.Background(), RoleSupervisor, 0, 9, 1, "2099-2100-1")
		require.Error(t, err)
		assert.Equal(t, errcode.Params, errcode.From(err).Code)
	})

	t.Run("教师访问他人课程返回 40302", func(t *testing.T) {
		_, err := svc.CourseSuggestions(context.Background(), RoleTeacher, 0, 3, 1, "")
		require.Error(t, err)
		assert.Equal(t, errcode.ForbiddenData, errcode.From(err).Code)
	})

	t.Run("主任访问他室课程返回 40302", func(t *testing.T) {
		_, err := svc.CourseSuggestions(context.Background(), RoleDirector, 2, 99, 1, "")
		require.Error(t, err)
		assert.Equal(t, errcode.ForbiddenData, errcode.From(err).Code)
	})

	t.Run("课程不存在返回 40401", func(t *testing.T) {
		missing := NewAgentImproveService(&fakeUserRepo{}, &fakeCourseRepo{
			getDetail: func(context.Context, uint64) (*repository.CourseRow, error) {
				return nil, gorm.ErrRecordNotFound
			},
		}, evals, evalCfg())
		_, err := missing.CourseSuggestions(context.Background(), RoleSupervisor, 0, 9, 1, "")
		require.Error(t, err)
		assert.Equal(t, errcode.NotFound, errcode.From(err).Code)
	})
}

// TestTeacherScoreTrend 覆盖排序、缺失点 null、AI 单源、composite 归一化、维度过滤与越权。
func TestTeacherScoreTrend(t *testing.T) {
	dept := uint64(1)
	users := &fakeUserRepo{getByID: func(_ context.Context, id uint64) (*model.User, error) {
		return &model.User{ID: id, Name: "李明", Role: model.RoleTeacher, DepartmentID: &dept}, nil
	}}
	rows := []repository.SessionDimRow{
		{
			// 最新场次：AI 五维（5/4/3/4/2）；督导分故意为 1，若混入会显著拉低结果。
			SessionID: 3, TeacherID: 2, CourseID: 1, SessionDate: mustDate(t, "2026-09-26"),
			HasSupervisor: true, HasAgent: true,
			SupObjective: f64ptr(1), SupContent: f64ptr(1), SupInteraction: f64ptr(1),
			SupOrganization: f64ptr(1), SupFrontier: f64ptr(1),
			AiObjective: f64ptr(5), AiContent: f64ptr(4), AiInteraction: f64ptr(3),
			AiOrganization: f64ptr(4), AiFrontier: f64ptr(2),
		},
		{
			// 最早场次：只有督导评价 → 所有点必须为 null。
			SessionID: 1, TeacherID: 2, CourseID: 1, SessionDate: mustDate(t, "2026-09-12"),
			HasSupervisor: true, HasAgent: false,
			SupObjective: f64ptr(5), SupContent: f64ptr(5), SupInteraction: f64ptr(5),
			SupOrganization: f64ptr(5), SupFrontier: f64ptr(5),
		},
		{
			// 中间场次：AI 只有 objective，composite 按现有规则归一化到 50。
			SessionID: 2, TeacherID: 2, CourseID: 1, SessionDate: mustDate(t, "2026-09-19"),
			HasAgent: true, AiObjective: f64ptr(3),
		},
	}

	var captured repository.SessionDimFilter
	evals := &fakeEvalRepo{listSessionDimRows: func(
		_ context.Context, flt repository.SessionDimFilter,
	) ([]repository.SessionDimRow, error) {
		captured = flt
		return rows, nil
	}}
	svc := NewAgentImproveService(users, &fakeCourseRepo{}, evals, evalCfg())

	got, err := svc.TeacherScoreTrend(context.Background(), RoleDirector, 1, 99, 2, dto.ScoreTrendQuery{})
	require.NoError(t, err)
	assert.Equal(t, repository.SessionDimFilter{
		TeacherID: 2, Semester: CurrentSemester, FormulaVersion: "v1",
	}, captured, "必须把 TeacherID/学期/口径版本下推仓储")

	require.Len(t, got, 6)
	assert.Equal(t, []string{"composite", "objective", "content", "interaction", "organization", "frontier"},
		[]string{got[0].Key, got[1].Key, got[2].Key, got[3].Key, got[4].Key, got[5].Key},
		"序列顺序固定")
	assert.Equal(t, "综合分", got[0].Name)
	assert.Equal(t, "教学目标与内容准确性", got[1].Name)
	assert.False(t, got[0].IsObservation)
	assert.True(t, got[5].IsObservation, "frontier 为观测项")

	composite := got[0].Points
	require.Len(t, composite, 3)
	assert.Equal(t, []string{"第 1 次课", "第 2 次课", "第 3 次课"},
		[]string{composite[0].Label, composite[1].Label, composite[2].Label})
	assert.Equal(t, []string{"2026-09-12", "2026-09-19", "2026-09-26"},
		[]string{composite[0].Date, composite[1].Date, composite[2].Date}, "必须按日期升序")
	assert.Nil(t, composite[0].Value, "无 agent 评价的场次必须为 null，不得补 0")
	require.NotNil(t, composite[1].Value)
	assert.Equal(t, 50.0, *composite[1].Value, "缺维度时 composite 按现有权重归一化")
	require.NotNil(t, composite[2].Value)
	assert.Equal(t, 77.5, *composite[2].Value, "AI 单源：督导 1 分不得参与")

	objective := got[1].Points
	assert.Nil(t, objective[0].Value)
	require.NotNil(t, objective[1].Value)
	assert.Equal(t, 50.0, *objective[1].Value)
	require.NotNil(t, objective[2].Value)
	assert.Equal(t, 100.0, *objective[2].Value)

	interaction := got[3].Points
	assert.Nil(t, interaction[1].Value, "该场次未观测 interaction，必须为 null")
	require.NotNil(t, interaction[2].Value)
	assert.Equal(t, 50.0, *interaction[2].Value)

	frontier := got[5].Points
	require.NotNil(t, frontier[2].Value)
	assert.Equal(t, 25.0, *frontier[2].Value)

	t.Run("dimension 过滤只返回指定序列", func(t *testing.T) {
		filtered, ferr := svc.TeacherScoreTrend(context.Background(), RoleSupervisor, 0, 9, 2,
			dto.ScoreTrendQuery{Dimension: "interaction"})
		require.NoError(t, ferr)
		require.Len(t, filtered, 1)
		assert.Equal(t, "interaction", filtered[0].Key)
		assert.Len(t, filtered[0].Points, 3)
	})

	t.Run("非法 dimension 返回 40001", func(t *testing.T) {
		_, ferr := svc.TeacherScoreTrend(context.Background(), RoleSupervisor, 0, 9, 2,
			dto.ScoreTrendQuery{Dimension: "unknown"})
		require.Error(t, ferr)
		assert.Equal(t, errcode.Params, errcode.From(ferr).Code)
	})

	t.Run("courseId 与 semester 透传仓储", func(t *testing.T) {
		_, ferr := svc.TeacherScoreTrend(context.Background(), RoleSupervisor, 0, 9, 2,
			dto.ScoreTrendQuery{Semester: "2025-2026-2", CourseID: 8})
		require.NoError(t, ferr)
		assert.Equal(t, uint64(8), captured.CourseID)
		assert.Equal(t, "2025-2026-2", captured.Semester)
	})

	t.Run("教师查看他人趋势返回 40302", func(t *testing.T) {
		_, ferr := svc.TeacherScoreTrend(context.Background(), RoleTeacher, 0, 3, 2, dto.ScoreTrendQuery{})
		require.Error(t, ferr)
		assert.Equal(t, errcode.ForbiddenData, errcode.From(ferr).Code)
	})

	t.Run("主任查看他室教师返回 40302", func(t *testing.T) {
		_, ferr := svc.TeacherScoreTrend(context.Background(), RoleDirector, 2, 99, 2, dto.ScoreTrendQuery{})
		require.Error(t, ferr)
		assert.Equal(t, errcode.ForbiddenData, errcode.From(ferr).Code)
	})
}
