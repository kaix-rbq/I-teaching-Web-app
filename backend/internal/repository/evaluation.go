package repository

import (
	"context"

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
type SessionDimRow struct {
	SessionID       uint64   `gorm:"column:session_id"`
	TeacherID       uint64   `gorm:"column:teacher_id"`
	CourseID        uint64   `gorm:"column:course_id"`
	HasSupervisor   bool     `gorm:"column:has_supervisor"`
	HasAgent        bool     `gorm:"column:has_agent"`
	SupObjective    *float64 `gorm:"column:sup_objective"`
	SupContent      *float64 `gorm:"column:sup_content"`
	SupInteraction  *float64 `gorm:"column:sup_interaction"`
	SupOrganization *float64 `gorm:"column:sup_organization"`
	SupFrontier     *float64 `gorm:"column:sup_frontier"`
	AiObjective     *float64 `gorm:"column:ai_objective"`
	AiContent       *float64 `gorm:"column:ai_content"`
	AiInteraction   *float64 `gorm:"column:ai_interaction"`
	AiOrganization  *float64 `gorm:"column:ai_organization"`
	AiFrontier      *float64 `gorm:"column:ai_frontier"`
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

// EvaluationRepository 定义课堂评价数据访问。
type EvaluationRepository interface {
	Upsert(ctx context.Context, e *model.Evaluation) error
	ListBySession(ctx context.Context, sessionID uint64) ([]EvaluationRow, error)
	ListSessionDimRows(ctx context.Context, f SessionDimFilter) ([]SessionDimRow, error)
}

type evaluationRepository struct {
	db *gorm.DB
}

// NewEvaluationRepository 构造课堂评价仓储。
func NewEvaluationRepository(db *gorm.DB) EvaluationRepository {
	return &evaluationRepository{db: db}
}

// Upsert 按 uk_eval(session_id, evaluator_type, evaluator_id) 幂等覆盖：
// 督导重复提交时整行更新而非报错（§3.1 无草稿态约定，写入即生效）。
func (r *evaluationRepository) Upsert(ctx context.Context, e *model.Evaluation) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "session_id"}, {Name: "evaluator_type"}, {Name: "evaluator_id"},
			},
			DoUpdates: clause.AssignmentColumns([]string{
				"formula_version", "ai_model_version",
				"objective_score", "content_score", "interaction_score",
				"organization_score", "frontier_score", "total_score", "ai_confidence",
				"evidence", "comment", "highlights", "improvements", "suggestions",
			}),
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

// ListSessionDimRows 是教师级/课程级聚合的数据出口（§2.5.4）。
// 三处易错点均已按计划处理：
//  1. 侧别计数用 MAX(CASE...THEN 1 ELSE 0)，不能 COUNT(列)——agent 的 objective 恒为 NULL；
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
