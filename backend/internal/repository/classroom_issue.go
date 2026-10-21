package repository

import (
	"context"

	"gorm.io/gorm"

	"aijiaoxue-api/internal/model"
)

// ClassroomIssueRow 是课堂问题行（含教师/处理人姓名）。
type ClassroomIssueRow struct {
	model.ClassroomIssue
	TeacherName string `gorm:"column:teacher_name"`
	HandlerName string `gorm:"column:handler_name"`
}

// ClassroomIssueRepository 定义课堂问题登记数据访问。
type ClassroomIssueRepository interface {
	List(ctx context.Context, teacherID uint64) ([]ClassroomIssueRow, error)
	GetByID(ctx context.Context, id uint64) (*model.ClassroomIssue, error)
	Create(ctx context.Context, issue *model.ClassroomIssue) error
	Handle(ctx context.Context, id, handlerID uint64, note string) error
	Reopen(ctx context.Context, id uint64) error
	Delete(ctx context.Context, id uint64) error
}

type classroomIssueRepository struct{ db *gorm.DB }

// NewClassroomIssueRepository 构造课堂问题仓储。
func NewClassroomIssueRepository(db *gorm.DB) ClassroomIssueRepository {
	return &classroomIssueRepository{db: db}
}

func (r *classroomIssueRepository) List(ctx context.Context, teacherID uint64) ([]ClassroomIssueRow, error) {
	var rows []ClassroomIssueRow
	q := r.db.WithContext(ctx).
		Table("classroom_issues AS i").
		Select("i.*, COALESCE(t.name,'') AS teacher_name, COALESCE(h.name,'') AS handler_name").
		Joins("LEFT JOIN users AS t ON t.id = i.teacher_id").
		Joins("LEFT JOIN users AS h ON h.id = i.handler_id")
	if teacherID > 0 {
		q = q.Where("i.teacher_id = ?", teacherID)
	}
	err := q.Order("i.created_at DESC, i.id DESC").Scan(&rows).Error
	return rows, err
}

func (r *classroomIssueRepository) GetByID(ctx context.Context, id uint64) (*model.ClassroomIssue, error) {
	var issue model.ClassroomIssue
	if err := r.db.WithContext(ctx).First(&issue, id).Error; err != nil {
		return nil, err
	}
	return &issue, nil
}

func (r *classroomIssueRepository) Create(ctx context.Context, issue *model.ClassroomIssue) error {
	return r.db.WithContext(ctx).Create(issue).Error
}

func (r *classroomIssueRepository) Handle(ctx context.Context, id, handlerID uint64, note string) error {
	return r.db.WithContext(ctx).Model(&model.ClassroomIssue{}).Where("id = ?", id).Updates(map[string]any{
		"status":      model.IssueStatusHandled,
		"handle_note": note,
		"handler_id":  handlerID,
		"handled_at":  gorm.Expr("NOW()"),
	}).Error
}

// Reopen 教师反馈未解决：回到进行中，保留 handle_note 供再处理。
func (r *classroomIssueRepository) Reopen(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Model(&model.ClassroomIssue{}).Where("id = ?", id).
		Update("status", model.IssueStatusOpen).Error
}

func (r *classroomIssueRepository) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.ClassroomIssue{}).Error
}
