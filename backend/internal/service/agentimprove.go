package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"aijiaoxue-api/internal/config"
	"aijiaoxue-api/internal/dto"
	"aijiaoxue-api/internal/model"
	"aijiaoxue-api/internal/repository"
	"aijiaoxue-api/pkg/errcode"
	"aijiaoxue-api/pkg/scoring"
)

// AgentImproveService 提供提优读接口（§8.15 P2）：
//   - GET /courses/:id/agent-suggestions：课程 AI 提优建议（v1 数组，不分页）；
//   - GET /teachers/:id/score-trend：教师综合分 + 五维趋势（AI 单源）。
//
// 两者都只读 evaluations 的 agent 行；督导分不参与趋势，也不产生提优建议。
type AgentImproveService interface {
	CourseSuggestions(ctx context.Context, role string, deptID, userID, courseID uint64, semester string) ([]dto.AgentSuggestionDTO, error)
	TeacherScoreTrend(ctx context.Context, role string, deptID, userID, teacherID uint64, q dto.ScoreTrendQuery) ([]dto.ScoreTrendSeriesDTO, error)
}

type agentImproveService struct {
	users   repository.UserRepository
	courses repository.CourseRepository
	evals   repository.EvaluationRepository
	cfg     config.EvaluationConfig
}

// NewAgentImproveService 构造提优读接口服务。
func NewAgentImproveService(
	users repository.UserRepository,
	courses repository.CourseRepository,
	evals repository.EvaluationRepository,
	cfg config.EvaluationConfig,
) AgentImproveService {
	return &agentImproveService{users: users, courses: courses, evals: evals, cfg: cfg}
}

// CourseSuggestions 返回课程 AI 提优建议（按课次倒序，v1 不分页）。
// 权限：登录后按课程范围裁剪——teacher 本人课程 / director 本室 / supervisor 全校；
// 越权返回 40302。semester 缺省当前学期，非法值返回 40001。
func (s *agentImproveService) CourseSuggestions(
	ctx context.Context, role string, deptID, userID, courseID uint64, semester string,
) ([]dto.AgentSuggestionDTO, error) {
	sem, err := resolveSemester(semester)
	if err != nil {
		return nil, err
	}

	course, err := s.courses.GetDetail(ctx, courseID)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, errcode.New(errcode.NotFound, "课程不存在")
		}
		return nil, errcode.Wrap(errcode.Internal, "查询课程失败", err)
	}
	if err := checkCourseScope(ScopeFor(role, deptID, userID), course); err != nil {
		return nil, err
	}

	rows, err := s.evals.ListAgentSuggestions(ctx, repository.AgentSuggestionFilter{
		CourseID:       courseID,
		Semester:       sem,
		FormulaVersion: s.cfg.FormulaVersion,
	})
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "查询 AI 提优建议失败", err)
	}

	// 防御性过滤：仓储 SQL 已限定 agent 行，service 再兜一层，避免未来查询改动
	// 把督导行混进建议列表（督导行没有 AI 建议）。
	filtered := make([]repository.AgentSuggestionRow, 0, len(rows))
	for _, row := range rows {
		if row.EvaluatorType != model.EvaluatorAgent {
			continue
		}
		filtered = append(filtered, row)
	}
	// 契约排序：session_date DESC, evaluations.id DESC。仓储已排序，这里再排一次
	// 以保证 service 单测不受 fake 返回顺序影响。
	sort.SliceStable(filtered, func(i, j int) bool {
		if !filtered[i].SessionDate.Equal(filtered[j].SessionDate) {
			return filtered[i].SessionDate.After(filtered[j].SessionDate)
		}
		return filtered[i].ID > filtered[j].ID
	})

	items := make([]dto.AgentSuggestionDTO, 0, len(filtered))
	for _, row := range filtered {
		items = append(items, dto.AgentSuggestionDTO{
			ID:           row.ID,
			CourseID:     row.CourseID,
			SessionID:    row.SessionID,
			SessionDate:  row.SessionDate.Format("2006-01-02"),
			Period:       row.Period,
			Topic:        row.Topic,
			Summary:      row.Comment,
			Highlights:   row.Highlights,
			Improvements: row.Improvements,
			Evidence:     renderSuggestionEvidence(row),
			ModelVersion: row.AIModelVersion,
			Confidence:   clampConfidence(row.AIConfidence),
			CreatedAt:    row.CreatedAt.Format(time.RFC3339),
		})
	}
	return items, nil
}

// evidenceDimension 是证据渲染时固定遍历的维度顺序（与 pkg/scoring 五维一致）。
// 同分时取靠前的维度，保证输出稳定可测。
var evidenceDimension = []struct {
	key   string
	score func(repository.AgentSuggestionRow) *uint8
}{
	{scoring.KeyObjective, func(r repository.AgentSuggestionRow) *uint8 { return r.ObjectiveScore }},
	{scoring.KeyContent, func(r repository.AgentSuggestionRow) *uint8 { return r.ContentScore }},
	{scoring.KeyInteraction, func(r repository.AgentSuggestionRow) *uint8 { return r.InteractionScore }},
	{scoring.KeyOrganization, func(r repository.AgentSuggestionRow) *uint8 { return r.OrganizationScore }},
	{scoring.KeyFrontier, func(r repository.AgentSuggestionRow) *uint8 { return r.FrontierScore }},
}

// renderSuggestionEvidence 按 §8.15 规则把 evidence JSON 渲染为人类可读字符串：
//  1. 在该场次 agent 评价中找分数最低的非空维度；
//  2. 取该维度第一条 quote，输出「quote」（mm:ss–mm:ss）；
//  3. 无 quote 回退 highlights；再无则空串。
//
// 证据 JSON 解析失败时按「无引用」处理（与 toEvidenceDTO 的容错口径一致），
// 不允许一条脏数据打掉整个建议列表。
func renderSuggestionEvidence(row repository.AgentSuggestionRow) string {
	var lowestKey string
	var lowestScore uint8
	for _, dim := range evidenceDimension {
		score := dim.score(row)
		if score == nil {
			continue
		}
		if lowestKey == "" || *score < lowestScore {
			lowestKey = dim.key
			lowestScore = *score
		}
	}

	if lowestKey != "" {
		if evidence := toEvidenceDTO(row.Evidence); evidence != nil {
			if dim, ok := evidence.Dimensions[lowestKey]; ok {
				for _, quote := range dim.Quotes {
					if strings.TrimSpace(quote.Quote) == "" {
						continue
					}
					return fmt.Sprintf("「%s」（%s–%s）",
						quote.Quote, formatMMSS(quote.Start), formatMMSS(quote.End))
				}
			}
		}
	}
	if strings.TrimSpace(row.Highlights) != "" {
		return row.Highlights
	}
	return ""
}

// formatMMSS 把秒级时间戳渲染为 mm:ss（向下取整；负值按 0 处理）。
func formatMMSS(seconds float64) string {
	if seconds < 0 {
		seconds = 0
	}
	total := int(seconds)
	return fmt.Sprintf("%02d:%02d", total/60, total%60)
}

// clampConfidence 把模型自评置信度收敛到 0–1；空值按 0。
func clampConfidence(v *float64) float64 {
	if v == nil {
		return 0
	}
	if *v < 0 {
		return 0
	}
	if *v > 1 {
		return 1
	}
	return *v
}

// trendSeriesMeta 固定趋势序列顺序与展示名（必须与 frontend/src/types/teacher.ts
// 的 ScoreTrendSeries 对齐；维度名与 pkg/scoring 五维一致）。
var trendSeriesMeta = []struct {
	Key           string
	Name          string
	IsObservation bool
}{
	{"composite", "综合分", false},
	{scoring.KeyObjective, "教学目标与内容准确性", false},
	{scoring.KeyContent, "内容质量与深度", false},
	{scoring.KeyInteraction, "学生互动与参与", false},
	{scoring.KeyOrganization, "课堂组织与节奏", false},
	{scoring.KeyFrontier, "前沿与交叉学科", true},
}

// TeacherScoreTrend 返回教师综合分 + 五维趋势（按场次出点，AI 单源）。
// 权限：teacher 仅本人 / director 本室 / supervisor 全校；越权 40302。
// semester 缺省当前学期；courseId 可选；dimension 可选，非法值 40001。
// 无 agent 评价的场次 value=null（折线断开），不得补 0；督导分不参与任何序列。
func (s *agentImproveService) TeacherScoreTrend(
	ctx context.Context, role string, deptID, userID, teacherID uint64, q dto.ScoreTrendQuery,
) ([]dto.ScoreTrendSeriesDTO, error) {
	sem, err := resolveSemester(q.Semester)
	if err != nil {
		return nil, err
	}
	dimension := strings.TrimSpace(q.Dimension)
	if dimension != "" && !isTrendDimension(dimension) {
		return nil, errcode.New(errcode.Params, "dimension 参数非法")
	}
	if _, err := s.checkTeacherAccess(ctx, role, deptID, userID, teacherID); err != nil {
		return nil, err
	}

	rows, err := s.evals.ListSessionDimRows(ctx, repository.SessionDimFilter{
		TeacherID:      teacherID,
		CourseID:       q.CourseID,
		Semester:       sem,
		FormulaVersion: s.cfg.FormulaVersion,
	})
	if err != nil {
		return nil, errcode.Wrap(errcode.Internal, "查询教师评分趋势失败", err)
	}

	// 每场次一个点，按 sessionDate 升序；同日再按 sessionId 升序，保证标签稳定。
	sort.SliceStable(rows, func(i, j int) bool {
		if !rows[i].SessionDate.Equal(rows[j].SessionDate) {
			return rows[i].SessionDate.Before(rows[j].SessionDate)
		}
		return rows[i].SessionID < rows[j].SessionID
	})

	series := make([]dto.ScoreTrendSeriesDTO, 0, len(trendSeriesMeta))
	for _, meta := range trendSeriesMeta {
		if dimension != "" && dimension != meta.Key {
			continue
		}
		points := make([]dto.ScoreTrendPointDTO, 0, len(rows))
		for index, row := range rows {
			values := s.trendValues(row)
			points = append(points, dto.ScoreTrendPointDTO{
				Label: fmt.Sprintf("第 %d 次课", index+1),
				Date:  row.SessionDate.Format("2006-01-02"),
				Value: values.forSeries(meta.Key),
			})
		}
		series = append(series, dto.ScoreTrendSeriesDTO{
			Key:           meta.Key,
			Name:          meta.Name,
			IsObservation: meta.IsObservation,
			Points:        points,
		})
	}
	return series, nil
}

// trendValues 是单场次的趋势取值；nil 表示该课次无 AI 评价（含无法观测该维度）。
type trendValues struct {
	values map[string]*float64
}

func (v trendValues) forSeries(key string) *float64 {
	if v.values == nil {
		return nil
	}
	return v.values[key]
}

// trendValues 计算单场次各序列的 0–100 取值，只使用 agent 侧维度分：
//   - 维度：scoring.DimScore(v)；
//   - composite：scoring.SideTotal 对 agent 五维加权（缺维度按现有规则归一化）。
//
// 督导分（Sup*）完全不参与，符合「AI 单源」口径。
func (s *agentImproveService) trendValues(row repository.SessionDimRow) trendValues {
	return sessionTrendValues(row, s.cfg.ScoringWeights())
}

// sessionTrendValues 是趋势口径的唯一实现：P2 score-trend 与 P3 教师报告共用。
// 只使用 agent 侧维度分；督导分不参与，缺维度按 scoring.SideTotal 的现有规则归一化。
func sessionTrendValues(row repository.SessionDimRow, weights scoring.Weights) trendValues {
	out := trendValues{values: map[string]*float64{}}
	if !row.HasAgent {
		return out
	}

	raw := &scoring.DimensionScores{
		Objective:    row.AiObjective,
		Content:      row.AiContent,
		Interaction:  row.AiInteraction,
		Organization: row.AiOrganization,
		Frontier:     row.AiFrontier,
	}
	if total := scoring.SideTotal(raw, weights); total != nil {
		v := round2(*total)
		out.values["composite"] = &v
	}
	addDimension := func(key string, score *float64) {
		if score == nil {
			return
		}
		v := round2(scoring.DimScore(*score))
		out.values[key] = &v
	}
	addDimension(scoring.KeyObjective, row.AiObjective)
	addDimension(scoring.KeyContent, row.AiContent)
	addDimension(scoring.KeyInteraction, row.AiInteraction)
	addDimension(scoring.KeyOrganization, row.AiOrganization)
	addDimension(scoring.KeyFrontier, row.AiFrontier)
	return out
}

// checkTeacherAccess 与 teacherScoreService.checkTeacherAccess 同口径：
// teacher 仅自己 / director 本室 / supervisor 全校；越权 40302。
// 两份实现同源于 §5.1 权限矩阵，调整时需同步。
func (s *agentImproveService) checkTeacherAccess(
	ctx context.Context, role string, deptID, userID, teacherID uint64,
) (*model.User, error) {
	teacher, err := s.users.GetByID(ctx, teacherID)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, errcode.New(errcode.NotFound, "教师不存在")
		}
		return nil, errcode.Wrap(errcode.Internal, "查询教师失败", err)
	}
	if teacher.Role != model.RoleTeacher {
		return nil, errcode.New(errcode.NotFound, "教师不存在")
	}
	switch role {
	case RoleTeacher:
		if teacherID != userID {
			return nil, errcode.New(errcode.ForbiddenData, "教师只能查看本人的评分")
		}
	case RoleDirector:
		if teacher.DeptID() != deptID {
			return nil, errcode.New(errcode.ForbiddenData, "无权查看其他教研室的教师")
		}
	case RoleSupervisor:
	default:
		return nil, errcode.New(errcode.ForbiddenRole, errcode.ForbiddenRole.Message())
	}
	return teacher, nil
}

// isTrendDimension 校验 dimension 合法值（composite + 五维）。
func isTrendDimension(key string) bool {
	for _, meta := range trendSeriesMeta {
		if meta.Key == key {
			return true
		}
	}
	return false
}
