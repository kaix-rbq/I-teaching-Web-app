package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"aijiaoxue-api/internal/model"
)

// ResourceRow 是资源联表查询结果（含上传人姓名）。
type ResourceRow struct {
	ID         uint64    `gorm:"column:id"`
	CourseID   uint64    `gorm:"column:course_id"`
	Name       string    `gorm:"column:name"`
	Type       string    `gorm:"column:type"`
	Size       int64     `gorm:"column:size"`
	FilePath   string    `gorm:"column:file_path"`
	UploaderID uint64    `gorm:"column:uploader_id"`
	Uploader   string    `gorm:"column:uploader"`
	UploadedAt time.Time `gorm:"column:uploaded_at"`
}

// ResourceRepository 定义课程资源数据访问。
type ResourceRepository interface {
	ListByCourse(ctx context.Context, courseID uint64) ([]ResourceRow, error)
	ListRecentByTeacher(ctx context.Context, teacherID uint64, limit int) ([]ResourceRow, error)
	GetByID(ctx context.Context, id uint64) (*model.Resource, error)
	Create(ctx context.Context, r *model.Resource) error
	Delete(ctx context.Context, id uint64) error
}

type resourceRepository struct {
	db *gorm.DB
}

// NewResourceRepository 构造资源仓储。
func NewResourceRepository(db *gorm.DB) ResourceRepository {
	return &resourceRepository{db: db}
}

const resourceSelect = `res.id, res.course_id, res.name, res.type, res.size, res.file_path,
	res.uploader_id, u.name AS uploader, res.uploaded_at`

func (r *resourceRepository) ListByCourse(ctx context.Context, courseID uint64) ([]ResourceRow, error) {
	var rows []ResourceRow
	err := r.db.WithContext(ctx).
		Table("resources AS res").
		Select(resourceSelect).
		Joins("JOIN users AS u ON u.id = res.uploader_id").
		Where("res.course_id = ?", courseID).
		Order("res.uploaded_at DESC, res.id DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *resourceRepository) ListRecentByTeacher(ctx context.Context, teacherID uint64, limit int) ([]ResourceRow, error) {
	var rows []ResourceRow
	err := r.db.WithContext(ctx).
		Table("resources AS res").
		Select(resourceSelect).
		Joins("JOIN users AS u ON u.id = res.uploader_id").
		Joins("JOIN courses AS c ON c.id = res.course_id").
		Where("c.teacher_id = ?", teacherID).
		Order("res.uploaded_at DESC, res.id DESC").
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *resourceRepository) GetByID(ctx context.Context, id uint64) (*model.Resource, error) {
	var res model.Resource
	if err := r.db.WithContext(ctx).Where("id = ?", id).Take(&res).Error; err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *resourceRepository) Create(ctx context.Context, res *model.Resource) error {
	return r.db.WithContext(ctx).Create(res).Error
}

func (r *resourceRepository) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.Resource{}).Error
}
