package repository

import (
	"context"

	"gorm.io/gorm"

	"aijiaoxue-api/internal/model"
)

// CourseRow 是课程列表/详情的联表查询结果（含聚合列）。
type CourseRow struct {
	ID            uint64 `gorm:"column:id"`
	Code          string `gorm:"column:code"`
	Name          string `gorm:"column:name"`
	Credit        int    `gorm:"column:credit"`
	Hours         int    `gorm:"column:hours"`
	Description   string `gorm:"column:description"`
	Semester      string `gorm:"column:semester"`
	DepartmentID  uint64 `gorm:"column:department_id"`
	TeacherID     uint64 `gorm:"column:teacher_id"`
	Status        string `gorm:"column:status"`
	TeacherName   string `gorm:"column:teacher_name"`
	Department    string `gorm:"column:department"`
	ClassCount    int    `gorm:"column:class_count"`
	StudentCount  int    `gorm:"column:student_count"`
	ResourceCount int    `gorm:"column:resource_count"`
}

// CourseFilter 是课程维度的过滤条件。
// Scope* 为后端强制的数据范围（角色裁剪），业务筛选参数不得覆盖它。
type CourseFilter struct {
	Semester          string
	DepartmentID      uint64
	TeacherID         uint64
	Status            string
	Keyword           string
	ScopeDepartmentID uint64
	ScopeTeacherID    uint64
}

// CourseListParams 是分页查询参数。
type CourseListParams struct {
	CourseFilter
	Page     int
	PageSize int
}

// CourseRepository 定义课程数据访问。
type CourseRepository interface {
	List(ctx context.Context, p CourseListParams) ([]CourseRow, int64, error)
	GetDetail(ctx context.Context, id uint64) (*CourseRow, error)
	ListClasses(ctx context.Context, courseID uint64) ([]model.ClassInfo, error)
	Count(ctx context.Context, f CourseFilter) (int64, error)
	CountClasses(ctx context.Context, f CourseFilter) (int64, error)
	SumStudents(ctx context.Context, f CourseFilter) (int64, error)
	CountResources(ctx context.Context, f CourseFilter) (int64, error)
	ExistsCodeSemester(ctx context.Context, code, semester string, excludeID uint64) (bool, error)
	Create(ctx context.Context, c *model.Course) error
	Update(ctx context.Context, id uint64, fields map[string]any) error
}

type courseRepository struct {
	db *gorm.DB
}

// NewCourseRepository 构造课程仓储。
func NewCourseRepository(db *gorm.DB) CourseRepository {
	return &courseRepository{db: db}
}

// courseSelect 是列表与详情共用的选择列；汇总值查询时计算，不落库。
const courseSelect = `c.id, c.code, c.name, c.credit, c.hours, c.description, c.semester,
	c.department_id, c.teacher_id, c.status,
	u.name AS teacher_name, d.name AS department,
	(SELECT COUNT(*) FROM course_classes cc WHERE cc.course_id = c.id) AS class_count,
	(SELECT COALESCE(SUM(cc.student_count), 0) FROM course_classes cc WHERE cc.course_id = c.id) AS student_count,
	(SELECT COUNT(*) FROM resources r WHERE r.course_id = c.id) AS resource_count`

func (r *courseRepository) base(ctx context.Context, f CourseFilter) *gorm.DB {
	tx := r.db.WithContext(ctx).
		Table("courses AS c").
		Joins("JOIN users AS u ON u.id = c.teacher_id").
		Joins("JOIN departments AS d ON d.id = c.department_id")

	if f.ScopeDepartmentID > 0 {
		tx = tx.Where("c.department_id = ?", f.ScopeDepartmentID)
	}
	if f.ScopeTeacherID > 0 {
		tx = tx.Where("c.teacher_id = ?", f.ScopeTeacherID)
	}
	if f.Semester != "" {
		tx = tx.Where("c.semester = ?", f.Semester)
	}
	if f.DepartmentID > 0 {
		tx = tx.Where("c.department_id = ?", f.DepartmentID)
	}
	if f.TeacherID > 0 {
		tx = tx.Where("c.teacher_id = ?", f.TeacherID)
	}
	if f.Status != "" {
		tx = tx.Where("c.status = ?", f.Status)
	}
	if f.Keyword != "" {
		like := "%" + f.Keyword + "%"
		tx = tx.Where("(c.name LIKE ? OR c.code LIKE ?)", like, like)
	}
	return tx
}

func (r *courseRepository) List(ctx context.Context, p CourseListParams) ([]CourseRow, int64, error) {
	var total int64
	if err := r.base(ctx, p.CourseFilter).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []CourseRow
	err := r.base(ctx, p.CourseFilter).
		Select(courseSelect).
		Order("c.code ASC").
		Offset((p.Page - 1) * p.PageSize).
		Limit(p.PageSize).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *courseRepository) GetDetail(ctx context.Context, id uint64) (*CourseRow, error) {
	var row CourseRow
	tx := r.db.WithContext(ctx).
		Table("courses AS c").
		Select(courseSelect).
		Joins("JOIN users AS u ON u.id = c.teacher_id").
		Joins("JOIN departments AS d ON d.id = c.department_id").
		Where("c.id = ?", id).
		Limit(1).
		Scan(&row)
	if tx.Error != nil {
		return nil, tx.Error
	}
	if tx.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &row, nil
}

func (r *courseRepository) ListClasses(ctx context.Context, courseID uint64) ([]model.ClassInfo, error) {
	var classes []model.ClassInfo
	err := r.db.WithContext(ctx).
		Where("course_id = ?", courseID).
		Order("id ASC").
		Find(&classes).Error
	if err != nil {
		return nil, err
	}
	return classes, nil
}

func (r *courseRepository) Count(ctx context.Context, f CourseFilter) (int64, error) {
	var total int64
	err := r.base(ctx, f).Count(&total).Error
	return total, err
}

func (r *courseRepository) CountClasses(ctx context.Context, f CourseFilter) (int64, error) {
	var total int64
	err := r.base(ctx, f).
		Joins("JOIN course_classes AS cc ON cc.course_id = c.id").
		Count(&total).Error
	return total, err
}

func (r *courseRepository) SumStudents(ctx context.Context, f CourseFilter) (int64, error) {
	var result struct {
		Total int64 `gorm:"column:total"`
	}
	err := r.base(ctx, f).
		Joins("JOIN course_classes AS cc ON cc.course_id = c.id").
		Select("COALESCE(SUM(cc.student_count), 0) AS total").
		Scan(&result).Error
	if err != nil {
		return 0, err
	}
	return result.Total, nil
}

func (r *courseRepository) CountResources(ctx context.Context, f CourseFilter) (int64, error) {
	var total int64
	err := r.base(ctx, f).
		Joins("JOIN resources AS res ON res.course_id = c.id").
		Count(&total).Error
	return total, err
}

func (r *courseRepository) ExistsCodeSemester(ctx context.Context, code, semester string, excludeID uint64) (bool, error) {
	tx := r.db.WithContext(ctx).
		Model(&model.Course{}).
		Where("code = ? AND semester = ?", code, semester)
	if excludeID > 0 {
		tx = tx.Where("id <> ?", excludeID)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return false, err
	}
	return total > 0, nil
}

func (r *courseRepository) Create(ctx context.Context, c *model.Course) error {
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *courseRepository) Update(ctx context.Context, id uint64, fields map[string]any) error {
	return r.db.WithContext(ctx).
		Model(&model.Course{}).
		Where("id = ?", id).
		Updates(fields).Error
}
