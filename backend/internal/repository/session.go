package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"aijiaoxue-api/internal/model"
)

// SessionListRow 是课程历史授课记录行，含每场的双侧评分摘要（§4.1）。
// 摘要取该场全部同类评价 total_score 的均分；无评价侧为 NULL。
type SessionListRow struct {
	ID              uint64    `gorm:"column:id"`
	CourseID        uint64    `gorm:"column:course_id"`
	ClassID         uint64    `gorm:"column:class_id"`
	TeacherID       uint64    `gorm:"column:teacher_id"`
	SessionDate     time.Time `gorm:"column:session_date"`
	Period          string    `gorm:"column:period"`
	Topic           string    `gorm:"column:topic"`
	Status          string    `gorm:"column:status"`
	SupervisorTotal *float64  `gorm:"column:supervisor_total"`
	AgentTotal      *float64  `gorm:"column:agent_total"`
	EvaluationCount int       `gorm:"column:evaluation_count"`
}

// SessionDetailRow 是单场授课记录 + 课程/教师/班级上下文（数据裁剪所需列一并取出）。
type SessionDetailRow struct {
	ID           uint64    `gorm:"column:id"`
	CourseID     uint64    `gorm:"column:course_id"`
	ClassID      uint64    `gorm:"column:class_id"`
	TeacherID    uint64    `gorm:"column:teacher_id"`
	SessionDate  time.Time `gorm:"column:session_date"`
	Period       string    `gorm:"column:period"`
	Topic        string    `gorm:"column:topic"`
	Status       string    `gorm:"column:status"`
	CourseCode   string    `gorm:"column:course_code"`
	CourseName   string    `gorm:"column:course_name"`
	Semester     string    `gorm:"column:semester"`
	DepartmentID uint64    `gorm:"column:department_id"`
	TeacherName  string    `gorm:"column:teacher_name"`
	ClassName    string    `gorm:"column:class_name"`
}

// SessionListParams 是授课记录分页参数。
type SessionListParams struct {
	CourseID uint64
	Page     int
	PageSize int
}

// SessionRepository 定义授课记录数据访问。
type SessionRepository interface {
	Create(ctx context.Context, s *model.TeachingSession) error
	GetByID(ctx context.Context, id uint64) (*model.TeachingSession, error)
	GetDetail(ctx context.Context, id uint64) (*SessionDetailRow, error)
	ExistsDuplicate(ctx context.Context, courseID, classID uint64, date time.Time, period string, excludeID uint64) (bool, error)
	ListByCourse(ctx context.Context, p SessionListParams) ([]SessionListRow, int64, error)
	UpdateStatus(ctx context.Context, id uint64, status string) error
	PlanExists(ctx context.Context, planID uint64) (bool, error)
}

type sessionRepository struct {
	db *gorm.DB
}

// NewSessionRepository 构造授课记录仓储。
func NewSessionRepository(db *gorm.DB) SessionRepository {
	return &sessionRepository{db: db}
}

func (r *sessionRepository) Create(ctx context.Context, s *model.TeachingSession) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *sessionRepository) GetByID(ctx context.Context, id uint64) (*model.TeachingSession, error) {
	var s model.TeachingSession
	if err := r.db.WithContext(ctx).Where("id = ?", id).Take(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *sessionRepository) GetDetail(ctx context.Context, id uint64) (*SessionDetailRow, error) {
	var row SessionDetailRow
	err := r.db.WithContext(ctx).
		Table("teaching_sessions AS ts").
		Select(`ts.id, ts.course_id, ts.class_id, ts.teacher_id, ts.session_date, ts.period, ts.topic, ts.status,
			c.code AS course_code, c.name AS course_name, c.semester, c.department_id,
			u.name AS teacher_name, COALESCE(cc.class_name, '') AS class_name`).
		Joins("JOIN courses AS c ON c.id = ts.course_id").
		Joins("JOIN users AS u ON u.id = ts.teacher_id").
		Joins("LEFT JOIN course_classes AS cc ON cc.id = ts.class_id").
		Where("ts.id = ?", id).
		Take(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// ExistsDuplicate 校验 uk_session 唯一键：同课程 + 同班级 + 同日 + 同节次。
// excludeID 用于更新场景排除自身（当前无更新接口，恒传 0）。
func (r *sessionRepository) ExistsDuplicate(
	ctx context.Context, courseID, classID uint64, date time.Time, period string, excludeID uint64,
) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.TeachingSession{}).
		Where("course_id = ? AND class_id = ? AND session_date = ? AND period = ?", courseID, classID, date, period).
		Where("id <> ?", excludeID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ListByCourse 返回课程的历史授课记录（最新在前），每场附双侧评分摘要。
// 摘要用单次 total_score 的侧别均分：仅用于列表快速展示，聚合仍走 §2.5.4 维度口径。
func (r *sessionRepository) ListByCourse(ctx context.Context, p SessionListParams) ([]SessionListRow, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).
		Model(&model.TeachingSession{}).
		Where("course_id = ?", p.CourseID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []SessionListRow
	err := r.db.WithContext(ctx).
		Table("teaching_sessions AS ts").
		Select(`ts.id, ts.course_id, ts.class_id, ts.teacher_id, ts.session_date, ts.period, ts.topic, ts.status,
			AVG(CASE WHEN e.evaluator_type = 'supervisor' THEN e.total_score END) AS supervisor_total,
			AVG(CASE WHEN e.evaluator_type = 'agent'      THEN e.total_score END) AS agent_total,
			COUNT(e.id) AS evaluation_count`).
		Joins("LEFT JOIN evaluations AS e ON e.session_id = ts.id").
		Where("ts.course_id = ?", p.CourseID).
		Group("ts.id").
		Order("ts.session_date DESC, ts.id DESC").
		Offset((p.Page - 1) * p.PageSize).
		Limit(p.PageSize).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *sessionRepository) UpdateStatus(ctx context.Context, id uint64, status string) error {
	return r.db.WithContext(ctx).
		Model(&model.TeachingSession{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// PlanExists 校验来源听评课计划存在（teaching_sessions.plan_id 外键前置校验，避免 500）。
func (r *sessionRepository) PlanExists(ctx context.Context, planID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.SupervisionPlan{}).
		Where("id = ?", planID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
