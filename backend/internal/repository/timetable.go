package repository

import (
	"context"

	"gorm.io/gorm"

	"aijiaoxue-api/internal/model"
)

// TimetableRow 是课表行（含课程/班级名称，前端建课下拉与教师周课表共用）。
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
	Weekday     int    `gorm:"column:weekday"`
	PeriodStart int    `gorm:"column:period_start"`
	PeriodEnd   int    `gorm:"column:period_end"`
	Classroom   string `gorm:"column:classroom"`
	Remark      string `gorm:"column:remark"`
}

// TimetableFilter 是课表查询条件（teacherID=0 不限教师，semester="" 不限学期）。
type TimetableFilter struct {
	TeacherID uint64
	Semester  string
}

// TimetableRepository 定义课表数据访问（B11 下拉 + V13 周课表）。
type TimetableRepository interface {
	List(ctx context.Context, f TimetableFilter) ([]TimetableRow, error)
	GetByID(ctx context.Context, id uint64) (*model.CourseTimetable, error)
	Create(ctx context.Context, t *model.CourseTimetable) error
	UpdateRemark(ctx context.Context, id, teacherID uint64, remark string) (int64, error)
	Delete(ctx context.Context, id uint64) error
	ExistsDuplicate(ctx context.Context, courseID, classID, teacherID uint64, semester string, weekday, periodStart int) (bool, error)
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
			tt.class_id, cc.class_name, tt.teacher_id, u.name AS teacher_name, tt.semester,
			tt.weekday, tt.period_start, tt.period_end, tt.classroom, tt.remark`).
		Joins("JOIN courses c ON c.id = tt.course_id").
		Joins("JOIN course_classes cc ON cc.id = tt.class_id").
		Joins("JOIN users u ON u.id = tt.teacher_id")
	if f.TeacherID > 0 {
		q = q.Where("tt.teacher_id = ?", f.TeacherID)
	}
	if f.Semester != "" {
		q = q.Where("tt.semester = ?", f.Semester)
	}
	if err := q.Order("tt.semester DESC, tt.weekday, tt.period_start, c.code, cc.class_name").Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *timetableRepository) GetByID(ctx context.Context, id uint64) (*model.CourseTimetable, error) {
	var t model.CourseTimetable
	if err := r.db.WithContext(ctx).First(&t, id).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *timetableRepository) Create(ctx context.Context, t *model.CourseTimetable) error {
	return r.db.WithContext(ctx).Create(t).Error
}

func (r *timetableRepository) UpdateRemark(ctx context.Context, id, teacherID uint64, remark string) (int64, error) {
	result := r.db.WithContext(ctx).Model(&model.CourseTimetable{}).
		Where("id = ? AND teacher_id = ?", id, teacherID).
		Update("remark", remark)
	return result.RowsAffected, result.Error
}

func (r *timetableRepository) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.CourseTimetable{}).Error
}

func (r *timetableRepository) ExistsDuplicate(ctx context.Context, courseID, classID, teacherID uint64, semester string, weekday, periodStart int) (bool, error) {
	var n int64
	if err := r.db.WithContext(ctx).Model(&model.CourseTimetable{}).
		Where("course_id = ? AND class_id = ? AND teacher_id = ? AND semester = ? AND weekday = ? AND period_start = ?",
			courseID, classID, teacherID, semester, weekday, periodStart).Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}
