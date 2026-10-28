package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"aijiaoxue-api/internal/model"
)

// DeptCoverageRow 是分教研室覆盖率的原始计数。
type DeptCoverageRow struct {
	Department string `gorm:"column:department"`
	Total      int64  `gorm:"column:total"`
	Supervised int64  `gorm:"column:supervised"`
}

// PlanRow 是听评课安排的联表查询结果。
// SessionID 为该计划关联的授课记录（可为空：计划尚未创建授课记录），
// 用于工作台「查看授课记录」直接跳转评估页，免去多级点击。
type PlanRow struct {
	ID             uint64    `gorm:"column:id"`
	CourseID       uint64    `gorm:"column:course_id"`
	CourseName     string    `gorm:"column:course_name"`
	TeacherName    string    `gorm:"column:teacher_name"`
	SupervisorName string    `gorm:"column:supervisor_name"`
	PlannedDate    time.Time `gorm:"column:planned_date"`
	Status         string    `gorm:"column:status"`
	SessionID      *uint64   `gorm:"column:session_id"`
	// Evaluated：该计划关联的授课记录是否存在督导评价行（决定工作台是否暴露评估入口）。
	// 2026-10-27 口径：status='evaluated' 只表示 AI 已评析，不能作为督导已评依据。
	Evaluated bool `gorm:"column:evaluated"`
}

// PlanListParams 是听评课安排分页参数。
type PlanListParams struct {
	Status   string
	DateFrom string
	DateTo   string
	Page     int
	PageSize int
}

// SupervisionRepository 定义督导数据访问。
type SupervisionRepository interface {
	Coverage(ctx context.Context, semester string) (total, supervised int64, err error)
	CoverageByDepartment(ctx context.Context, semester string) ([]DeptCoverageRow, error)
	ListPlans(ctx context.Context, p PlanListParams) ([]PlanRow, int64, error)
	CountPlans(ctx context.Context, status string) (int64, error)
}

type supervisionRepository struct {
	db *gorm.DB
}

// NewSupervisionRepository 构造督导仓储。
func NewSupervisionRepository(db *gorm.DB) SupervisionRepository {
	return &supervisionRepository{db: db}
}

// 覆盖率口径（唯一）：分母 = 当前学期 status='open' 的课程数；
// 分子 = 这些课程中已有 status='completed' 听评课记录的课程数（DISTINCT course_id）。
func (r *supervisionRepository) Coverage(ctx context.Context, semester string) (int64, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).
		Model(&model.Course{}).
		Where("semester = ? AND status = ?", semester, model.CourseStatusOpen).
		Count(&total).Error; err != nil {
		return 0, 0, err
	}

	var supervised struct {
		Total int64 `gorm:"column:total"`
	}
	err := r.db.WithContext(ctx).
		Table("supervision_plans AS sp").
		Select("COUNT(DISTINCT sp.course_id) AS total").
		Joins("JOIN courses AS c ON c.id = sp.course_id").
		Where("sp.status = ?", model.PlanStatusCompleted).
		Where("c.semester = ? AND c.status = ?", semester, model.CourseStatusOpen).
		Scan(&supervised).Error
	if err != nil {
		return 0, 0, err
	}
	return total, supervised.Total, nil
}

func (r *supervisionRepository) CoverageByDepartment(ctx context.Context, semester string) ([]DeptCoverageRow, error) {
	var rows []DeptCoverageRow
	err := r.db.WithContext(ctx).
		Table("courses AS c").
		Select(`d.name AS department,
			COUNT(DISTINCT c.id) AS total,
			COUNT(DISTINCT CASE WHEN sp.status = 'completed' THEN c.id END) AS supervised`).
		Joins("JOIN departments AS d ON d.id = c.department_id").
		Joins("LEFT JOIN supervision_plans AS sp ON sp.course_id = c.id").
		Where("c.semester = ? AND c.status = ?", semester, model.CourseStatusOpen).
		Group("d.id, d.name").
		Order("d.id ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *supervisionRepository) planBase(ctx context.Context, p PlanListParams) *gorm.DB {
	tx := r.db.WithContext(ctx).
		Table("supervision_plans AS sp").
		Joins("JOIN courses AS c ON c.id = sp.course_id").
		Joins("JOIN users AS tu ON tu.id = c.teacher_id").
		Joins("JOIN users AS su ON su.id = sp.supervisor_id")
	if p.Status != "" {
		tx = tx.Where("sp.status = ?", p.Status)
	}
	if p.DateFrom != "" {
		tx = tx.Where("sp.planned_date >= ?", p.DateFrom)
	}
	if p.DateTo != "" {
		tx = tx.Where("sp.planned_date <= ?", p.DateTo)
	}
	return tx
}

// ListPlans 解析计划对应的授课记录（业务口径：教师建课、督导评价）。
//
// 优先取显式关联（teaching_sessions.plan_id）；若旧数据/教师从工作台建课未回填
// plan_id，则回退匹配「同课程 + 计划日期」的教师授课记录，避免督导点「去评估」
// 时再创建一条空记录，导致评价不到教师实际上传录音的场次。
func (r *supervisionRepository) ListPlans(ctx context.Context, p PlanListParams) ([]PlanRow, int64, error) {
	var total int64
	if err := r.planBase(ctx, p).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []PlanRow
	err := r.planBase(ctx, p).
		Select(`sp.id, sp.course_id, c.name AS course_name, tu.name AS teacher_name,
			su.name AS supervisor_name, sp.planned_date, sp.status,
			(SELECT ts.id FROM teaching_sessions AS ts
				WHERE ts.plan_id = sp.id
				   OR (ts.plan_id IS NULL AND ts.course_id = sp.course_id AND ts.session_date = sp.planned_date)
				ORDER BY (ts.plan_id = sp.id) DESC,
				         (EXISTS(SELECT 1 FROM recordings AS r WHERE r.session_id = ts.id)) DESC,
				         (ts.created_by = c.teacher_id) DESC, ts.id DESC
				LIMIT 1) AS session_id,
			EXISTS(SELECT 1 FROM teaching_sessions AS ts
				JOIN evaluations AS e ON e.session_id = ts.id
				WHERE (ts.plan_id = sp.id
				   OR (ts.plan_id IS NULL AND ts.course_id = sp.course_id AND ts.session_date = sp.planned_date))
				  AND e.evaluator_type = 'supervisor') AS evaluated`).
		Order("sp.planned_date ASC, sp.id ASC").
		Offset((p.Page - 1) * p.PageSize).
		Limit(p.PageSize).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *supervisionRepository) CountPlans(ctx context.Context, status string) (int64, error) {
	tx := r.db.WithContext(ctx).Model(&model.SupervisionPlan{})
	if status != "" {
		tx = tx.Where("status = ?", status)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}
