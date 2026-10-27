package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"aijiaoxue-api/internal/model"
)

// EvaluationRow 是单场评价 + 评价人姓名（agent 行的 evaluator_name 为空串）。
type EvaluationRow struct {
	model.Evaluation
	EvaluatorName string `gorm:"column:evaluator_name"`
}

// SessionDimRow 是 §2.5.4 聚合 SQL 的输出：每场次每侧的维度均分（1-5 原始标度）。
// SQL 只负责按侧取均分；(v−1)/4×100 换算、权重加权与 α 融合在 service 层的
// pkg/scoring.Aggregate 完成（线性等价，开发计划 §2.5.4 口径约定）。
// 同时带上场次/课程展示列，供「历次评价时间线」直接复用，避免二次查询。
type SessionDimRow struct {
	SessionID       uint64    `gorm:"column:session_id"`
	TeacherID       uint64    `gorm:"column:teacher_id"`
	CourseID        uint64    `gorm:"column:course_id"`
	SessionDate     time.Time `gorm:"column:session_date"`
	Period          string    `gorm:"column:period"`
	Topic           string    `gorm:"column:topic"`
	Status          string    `gorm:"column:status"`
	CourseCode      string    `gorm:"column:course_code"`
	CourseName      string    `gorm:"column:course_name"`
	HasSupervisor   bool      `gorm:"column:has_supervisor"`
	HasAgent        bool      `gorm:"column:has_agent"`
	SupObjective    *float64  `gorm:"column:sup_objective"`
	SupContent      *float64  `gorm:"column:sup_content"`
	SupInteraction  *float64  `gorm:"column:sup_interaction"`
	SupOrganization *float64  `gorm:"column:sup_organization"`
	SupFrontier     *float64  `gorm:"column:sup_frontier"`
	AiObjective     *float64  `gorm:"column:ai_objective"`
	AiContent       *float64  `gorm:"column:ai_content"`
	AiInteraction   *float64  `gorm:"column:ai_interaction"`
	AiOrganization  *float64  `gorm:"column:ai_organization"`
	AiFrontier      *float64  `gorm:"column:ai_frontier"`
}

// SessionDimFilter 是聚合范围：零值字段表示不限制。
// 学期切片通过 courses.semester 实现（§2.5.6：所有聚合接口必须支持 semester）。
type SessionDimFilter struct {
	TeacherID         uint64
	CourseID          uint64
	ScopeDepartmentID uint64 // 教师列表用：限定本教研室教师的场次
	Semester          string
	FormulaVersion    string
}

// AgentSessionRow 是「有 AI 终评的场次」行（抽样分派的输入，§4.3）：
// 含场次/课程/教师上下文 + AI 总分 + 证据（取 riskTags）+ 是否已有督导观察。
type AgentSessionRow struct {
	SessionID      uint64    `gorm:"column:session_id"`
	TeacherID      uint64    `gorm:"column:teacher_id"`
	TeacherName    string    `gorm:"column:teacher_name"`
	CourseID       uint64    `gorm:"column:course_id"`
	CourseCode     string    `gorm:"column:course_code"`
	CourseName     string    `gorm:"column:course_name"`
	SessionDate    time.Time `gorm:"column:session_date"`
	Period         string    `gorm:"column:period"`
	Topic          string    `gorm:"column:topic"`
	AITotal        *float64  `gorm:"column:ai_total"`
	Evidence       *string   `gorm:"column:evidence"`
	SupervisorCnt  int       `gorm:"column:supervisor_cnt"`
	SupervisorNote string    `gorm:"column:supervisor_note"`
}

// AgentSuggestionRow 是课程 AI 提优建议行（GET /courses/:id/agent-suggestions）：
// 仅 evaluator_type='agent' 的评价行；evidence 原样取出，由 service 渲染为人类可读字符串。
type AgentSuggestionRow struct {
	ID             uint64    `gorm:"column:id"`
	SessionID      uint64    `gorm:"column:session_id"`
	CourseID       uint64    `gorm:"column:course_id"`
	SessionDate    time.Time `gorm:"column:session_date"`
	Period         string    `gorm:"column:period"`
	Topic          string    `gorm:"column:topic"`
	EvaluatorType  string    `gorm:"column:evaluator_type"`
	Comment        string    `gorm:"column:comment"`
	Highlights     string    `gorm:"column:highlights"`
	Improvements   string    `gorm:"column:improvements"`
	Evidence       *string   `gorm:"column:evidence"`
	AIModelVersion string    `gorm:"column:ai_model_version"`
	AIConfidence   *float64  `gorm:"column:ai_confidence"`
	CreatedAt      time.Time `gorm:"column:created_at"`
	// 五维原始分（1-5）用于挑选「最低分非空维度」的引用。
	ObjectiveScore    *uint8 `gorm:"column:objective_score"`
	ContentScore      *uint8 `gorm:"column:content_score"`
	InteractionScore  *uint8 `gorm:"column:interaction_score"`
	OrganizationScore *uint8 `gorm:"column:organization_score"`
	FrontierScore     *uint8 `gorm:"column:frontier_score"`
}

// AgentSuggestionFilter 是课程 AI 提优建议的筛选范围。
// FormulaVersion 非空时只取该口径版本，避免公式混杂时把旧口径建议混进当前列表。
type AgentSuggestionFilter struct {
	CourseID       uint64
	Semester       string
	FormulaVersion string
}

// EvaluationRepository 定义课堂评价数据访问。
type EvaluationRepository interface {
	Upsert(ctx context.Context, e *model.Evaluation) error
	ListBySession(ctx context.Context, sessionID uint64) ([]EvaluationRow, error)
	ListBySessionIDs(ctx context.Context, sessionIDs []uint64) ([]EvaluationRow, error)
	ListSessionDimRows(ctx context.Context, f SessionDimFilter) ([]SessionDimRow, error)
	// ListAgentSessions 返回有 AI 终评的场次（抽样分派输入，§4.3），按教师 + 课次倒序。
	ListAgentSessions(ctx context.Context, semester, formulaVersion string) ([]AgentSessionRow, error)
	// ListAgentSuggestions 返回课程内 agent 评价行，按课次倒序（GET /courses/:id/agent-suggestions）。
	ListAgentSuggestions(ctx context.Context, f AgentSuggestionFilter) ([]AgentSuggestionRow, error)
	// DeleteBySession 删除某场次的全部评价（删课级联用）。
	DeleteBySession(ctx context.Context, sessionID uint64) error
}

type evaluationRepository struct {
	db *gorm.DB
}

// NewEvaluationRepository 构造课堂评价仓储。
func NewEvaluationRepository(db *gorm.DB) EvaluationRepository {
	return &evaluationRepository{db: db}
}

// evaluationConflictColumns 对应 evaluations 表的 uk_eval(session_id, evaluator_type, evaluator_id)。
var evaluationConflictColumns = []clause.Column{
	{Name: "session_id"}, {Name: "evaluator_type"}, {Name: "evaluator_id"},
}

// evaluationUpsertColumns 是 evaluations 表按 uk_eval 幂等覆盖时需要更新的列。
//
// 🔴 必须由督导（EvaluationRepository.Upsert）与智能体（RecordingRepository.UpsertAgentEvaluation）
// 两条写入路径共用同一份清单：此前两处各自维护列清单，智能体侧漏掉了四个评语列，
// 导致重复触发 AI 评价时 comment/highlights/improvements/suggestions 被**静默丢弃**
// （分数与证据更新了，评语还停留在上一轮）。见 TestEvaluationUpsertColumns。
var evaluationUpsertColumns = []string{
	"formula_version", "ai_model_version",
	"objective_score", "content_score", "interaction_score",
	"organization_score", "frontier_score", "total_score", "ai_confidence",
	"evidence", "comment", "highlights", "improvements", "suggestions",
}

// Upsert 按 uk_eval(session_id, evaluator_type, evaluator_id) 幂等覆盖：
// 督导重复提交时整行更新而非报错（§3.1 无草稿态约定，写入即生效）。
func (r *evaluationRepository) Upsert(ctx context.Context, e *model.Evaluation) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   evaluationConflictColumns,
			DoUpdates: clause.AssignmentColumns(evaluationUpsertColumns),
		}).Create(e).Error
}

// ListBySession 返回某场次的全部评价（督导 + 智能体），按侧别与提交时间排序。
func (r *evaluationRepository) ListBySession(ctx context.Context, sessionID uint64) ([]EvaluationRow, error) {
	var rows []EvaluationRow
	err := r.db.WithContext(ctx).
		Table("evaluations AS e").
		Select("e.*, COALESCE(u.name, '') AS evaluator_name").
		Joins("LEFT JOIN users AS u ON u.id = e.evaluator_id").
		Where("e.session_id = ?", sessionID).
		Order("e.evaluator_type DESC, e.created_at ASC, e.id ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// ListBySessionIDs 批量返回多个场次的全部评价（避免时间线逐场 N+1 查询）。
// 入参为空时直接返回空切片，不发起查询。
func (r *evaluationRepository) ListBySessionIDs(ctx context.Context, sessionIDs []uint64) ([]EvaluationRow, error) {
	if len(sessionIDs) == 0 {
		return []EvaluationRow{}, nil
	}
	var rows []EvaluationRow
	err := r.db.WithContext(ctx).
		Table("evaluations AS e").
		Select("e.*, COALESCE(u.name, '') AS evaluator_name").
		Joins("LEFT JOIN users AS u ON u.id = e.evaluator_id").
		Where("e.session_id IN ?", sessionIDs).
		Order("e.session_id ASC, e.evaluator_type DESC, e.created_at ASC, e.id ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// ListAgentSessions 返回有 AI 终评的场次（抽样分派输入，§4.3）。
func (r *evaluationRepository) ListAgentSessions(ctx context.Context, semester, formulaVersion string) ([]AgentSessionRow, error) {
	tx := r.db.WithContext(ctx).
		Table("teaching_sessions AS ts").
		Select(`ts.id AS session_id, ts.teacher_id, u.name AS teacher_name,
			ts.course_id, c.code AS course_code, c.name AS course_name,
			ts.session_date, ts.period, ts.topic,
			e.total_score AS ai_total, e.evidence AS evidence,
			(SELECT COUNT(*) FROM evaluations s WHERE s.session_id = ts.id AND s.evaluator_type = 'supervisor') AS supervisor_cnt,
			COALESCE((SELECT GROUP_CONCAT(s2.comment SEPARATOR '\n') FROM evaluations s2 WHERE s2.session_id = ts.id AND s2.evaluator_type = 'supervisor'), '') AS supervisor_note`).
		Joins("JOIN courses AS c ON c.id = ts.course_id").
		Joins("JOIN users AS u ON u.id = ts.teacher_id").
		Joins("JOIN evaluations AS e ON e.session_id = ts.id AND e.evaluator_type = 'agent' AND e.formula_version = ?", formulaVersion)
	if semester != "" {
		tx = tx.Where("c.semester = ?", semester)
	}
	var rows []AgentSessionRow
	err := tx.Order("ts.teacher_id ASC, ts.session_date DESC, ts.id DESC").Scan(&rows).Error
	return rows, err
}

// ListAgentSuggestions 返回课程 AI 提优建议（只取 agent 行），按 session_date DESC, evaluations.id DESC。
// 课程学期通过 courses.semester 过滤；formula_version 口径在 SQL 层隔离。
func (r *evaluationRepository) ListAgentSuggestions(
	ctx context.Context, f AgentSuggestionFilter,
) ([]AgentSuggestionRow, error) {
	tx := r.db.WithContext(ctx).
		Table("evaluations AS e").
		Select(`e.id, e.session_id, ts.course_id, ts.session_date, ts.period, ts.topic,
			e.evaluator_type, e.comment, e.highlights, e.improvements, e.evidence,
			e.ai_model_version, e.ai_confidence, e.created_at,
			e.objective_score, e.content_score, e.interaction_score,
			e.organization_score, e.frontier_score`).
		Joins("JOIN teaching_sessions AS ts ON ts.id = e.session_id").
		Joins("JOIN courses AS c ON c.id = ts.course_id").
		Where("e.evaluator_type = ?", model.EvaluatorAgent)
	if f.CourseID > 0 {
		tx = tx.Where("ts.course_id = ?", f.CourseID)
	}
	if f.Semester != "" {
		tx = tx.Where("c.semester = ?", f.Semester)
	}
	if f.FormulaVersion != "" {
		tx = tx.Where("e.formula_version = ?", f.FormulaVersion)
	}
	var rows []AgentSuggestionRow
	err := tx.Order("ts.session_date DESC, e.id DESC").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// DeleteBySession 删除某场次的全部评价（删课级联用；含督导观察与 AI 行）。
func (r *evaluationRepository) DeleteBySession(ctx context.Context, sessionID uint64) error {
	return r.db.WithContext(ctx).Where("session_id = ?", sessionID).Delete(&model.Evaluation{}).Error
}

// ListSessionDimRows 是教师级/课程级聚合的数据出口（§2.5.4）。
// 三处易错点均已按计划处理：
//  1. 侧别计数用 MAX(CASE...THEN 1 ELSE 0)，不能 COUNT(列)——允许某侧维度为 NULL（如早期 agent 的 objective）；
//  2. 同场次多督导取 AVG，不是 MAX；
//  3. 口径版本在 JOIN 条件里隔离，公式混杂时不混入旧口径数据。
func (r *evaluationRepository) ListSessionDimRows(ctx context.Context, f SessionDimFilter) ([]SessionDimRow, error) {
	tx := r.db.WithContext(ctx).
		Table("teaching_sessions AS ts").
		Joins("JOIN courses AS c ON c.id = ts.course_id").
		Joins("LEFT JOIN evaluations AS e ON e.session_id = ts.id AND e.formula_version = ?", f.FormulaVersion)
	if f.Semester != "" {
		tx = tx.Where("c.semester = ?", f.Semester)
	}
	if f.TeacherID > 0 {
		tx = tx.Where("ts.teacher_id = ?", f.TeacherID)
	}
	if f.CourseID > 0 {
		tx = tx.Where("ts.course_id = ?", f.CourseID)
	}
	if f.ScopeDepartmentID > 0 {
		tx = tx.Where("ts.teacher_id IN (?)",
			r.db.Model(&model.User{}).Select("id").
				Where("role = ? AND department_id = ?", model.RoleTeacher, f.ScopeDepartmentID))
	}

	var rows []SessionDimRow
	err := tx.Select(`ts.id AS session_id, ts.teacher_id, ts.course_id,
			ts.session_date, ts.period, ts.topic, ts.status,
			c.code AS course_code, c.name AS course_name,
			MAX(CASE WHEN e.evaluator_type = 'supervisor' THEN 1 ELSE 0 END) = 1 AS has_supervisor,
			MAX(CASE WHEN e.evaluator_type = 'agent'      THEN 1 ELSE 0 END) = 1 AS has_agent,
			AVG(CASE WHEN e.evaluator_type = 'supervisor' THEN e.objective_score    END) AS sup_objective,
			AVG(CASE WHEN e.evaluator_type = 'supervisor' THEN e.content_score      END) AS sup_content,
			AVG(CASE WHEN e.evaluator_type = 'supervisor' THEN e.interaction_score  END) AS sup_interaction,
			AVG(CASE WHEN e.evaluator_type = 'supervisor' THEN e.organization_score END) AS sup_organization,
			AVG(CASE WHEN e.evaluator_type = 'supervisor' THEN e.frontier_score     END) AS sup_frontier,
			AVG(CASE WHEN e.evaluator_type = 'agent'      THEN e.objective_score    END) AS ai_objective,
			AVG(CASE WHEN e.evaluator_type = 'agent'      THEN e.content_score      END) AS ai_content,
			AVG(CASE WHEN e.evaluator_type = 'agent'      THEN e.interaction_score  END) AS ai_interaction,
			AVG(CASE WHEN e.evaluator_type = 'agent'      THEN e.organization_score END) AS ai_organization,
			AVG(CASE WHEN e.evaluator_type = 'agent'      THEN e.frontier_score     END) AS ai_frontier`).
		Group("ts.id").
		Order("ts.id ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}
