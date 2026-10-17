package repository

import (
	"context"

	"gorm.io/gorm"

	"aijiaoxue-api/internal/model"
)

// TimetableRow 是课表下拉行（含课程/班级名称，前端建课下拉直接渲染）。
type TimetableRow struct {
	ID          uint64 `gorm:"column:id"`
	CourseID    uint64 `gorm:"column:course_id"`
	CourseCode  string `gorm:"column:course_code"`
	CourseName  string `gorm:"column:course_name"`
	ClassID     uint64 `gorm:"column:class_id"`
	ClassName   string `gorm:"column:class_name"`
	TeacherID   uint64 `gorm:"column:teacher_id"`
	TeacherName string `gorm:"column:teacher_name"`
	Semester    string `gorm:"column:semester"`
}

// TimetableFilter 是课表查询条件（teacherID=0 不限教师，semester="" 不限学期）。
type TimetableFilter struct {
	TeacherID uint64
	Semester  string
}

// TimetableRepository 定义最小课表的数据访问（B11：仅下拉用）。
type TimetableRepository interface {
	List(ctx context.Context, f TimetableFilter) ([]TimetableRow, error)
	Create(ctx context.Context, t *model.CourseTimetable) error
	Delete(ctx context.Context, id uint64) error
	ExistsDuplicate(ctx context.Context, courseID, classID, teacherID uint64, semester string) (bool, error)
}

type timetableRepository struct {
	db *gorm.DB
}

// NewTimetableRepository 构造课表仓储。
func NewTimetableRepository(db *gorm.DB) TimetableRepository {
	return &timetableRepository{db: db}
}

func (r *timetableRepository) List(ctx context.Context, f TimetableFilter) ([]TimetableRow, error) {
	var rows []TimetableRow
	q := r.db.WithContext(ctx).Table("course_timetable AS tt").
		Select(`tt.id, tt.course_id, c.code AS course_code, c.name AS course_name,
			tt.class_id, cc.class_name, tt.teacher_id, u.name AS teacher_name, tt.semester`).
		Joins("JOIN courses c ON c.id = tt.course_id").
		Joins("JOIN course_classes cc ON cc.id = tt.class_id").
		Joins("JOIN users u ON u.id = tt.teacher_id")
	if f.TeacherID > 0 {
		q = q.Where("tt.teacher_id = ?", f.TeacherID)
	}
	if f.Semester != "" {
		q = q.Where("tt.semester = ?", f.Semester)
	}
	if err := q.Order("tt.semester DESC, c.code, cc.class_name").Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *timetableRepository) Create(ctx context.Context, t *model.CourseTimetable) error {
	return r.db.WithContext(ctx).Create(t).Error
}

func (r *timetableRepository) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.CourseTimetable{}).Error
}

func (r *timetableRepository) ExistsDuplicate(ctx context.Context, courseID, classID, teacherID uint64, semester string) (bool, error) {
	var n int64
	if err := r.db.WithContext(ctx).Model(&model.CourseTimetable{}).
		Where("course_id = ? AND class_id = ? AND teacher_id = ? AND semester = ?",
			courseID, classID, teacherID, semester).Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}
