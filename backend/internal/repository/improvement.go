package repository

import (
	"context"

	"gorm.io/gorm"

	"aijiaoxue-api/internal/model"
)

// ImprovementItemRow 是待改进清单行（含教师/关闭人姓名，便于列表直接展示）。
type ImprovementItemRow struct {
	model.ImprovementItem
	TeacherName  string `gorm:"column:teacher_name"`
	ClosedByName string `gorm:"column:closed_by_name"`
}

// ImprovementListFilter 是清单查询范围：零值不限制。
type ImprovementListFilter struct {
	TeacherID uint64
	Status    string
	Page      int
	PageSize  int
}

// ImprovementRepository 定义待改进清单数据访问（N4）。
type ImprovementRepository interface {
	Create(ctx context.Context, it *model.ImprovementItem) error
	GetByID(ctx context.Context, id uint64) (*model.ImprovementItem, error)
	GetRowByID(ctx context.Context, id uint64) (*ImprovementItemRow, error)
	List(ctx context.Context, f ImprovementListFilter) ([]ImprovementItemRow, int64, error)
	UpdateTitle(ctx context.Context, id uint64, title string) error
	UpdateStatus(ctx context.Context, id uint64, fields map[string]any) error
	Delete(ctx context.Context, id uint64) error
	// DeleteBySession 删除某场次关联的改进项（删课级联用）。
	DeleteBySession(ctx context.Context, sessionID uint64) error
}

type improvementRepository struct{ db *gorm.DB }

// NewImprovementRepository 构造待改进清单仓储。
func NewImprovementRepository(db *gorm.DB) ImprovementRepository {
	return &improvementRepository{db: db}
}

func (r *improvementRepository) Create(ctx context.Context, it *model.ImprovementItem) error {
	return r.db.WithContext(ctx).Create(it).Error
}

func (r *improvementRepository) GetByID(ctx context.Context, id uint64) (*model.ImprovementItem, error) {
	var it model.ImprovementItem
	if err := r.db.WithContext(ctx).First(&it, id).Error; err != nil {
		return nil, err
	}
	return &it, nil
}

// GetRowByID 回读单行（含教师/关闭人姓名）。
func (r *improvementRepository) GetRowByID(ctx context.Context, id uint64) (*ImprovementItemRow, error) {
	var row ImprovementItemRow
	err := r.db.WithContext(ctx).
		Table("improvement_items AS i").
		Select("i.*, COALESCE(u.name,'') AS teacher_name, COALESCE(cb.name,'') AS closed_by_name").
		Joins("LEFT JOIN users AS u ON u.id = i.teacher_id").
		Joins("LEFT JOIN users AS cb ON cb.id = i.closed_by").
		Where("i.id = ?", id).
		Take(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *improvementRepository) List(ctx context.Context, f ImprovementListFilter) ([]ImprovementItemRow, int64, error) {
	tx := r.db.WithContext(ctx).
		Table("improvement_items AS i").
		Joins("LEFT JOIN users AS u ON u.id = i.teacher_id").
		Joins("LEFT JOIN users AS cb ON cb.id = i.closed_by")
	if f.TeacherID > 0 {
		tx = tx.Where("i.teacher_id = ?", f.TeacherID)
	}
	if f.Status != "" {
		tx = tx.Where("i.status = ?", f.Status)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []ImprovementItemRow
	err := tx.Select("i.*, COALESCE(u.name,'') AS teacher_name, COALESCE(cb.name,'') AS closed_by_name").
		Order("i.created_at DESC, i.id DESC").
		Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize).
		Scan(&rows).Error
	return rows, total, err
}

func (r *improvementRepository) UpdateTitle(ctx context.Context, id uint64, title string) error {
	return r.db.WithContext(ctx).Model(&model.ImprovementItem{}).Where("id = ?", id).Update("title", title).Error
}

func (r *improvementRepository) UpdateStatus(ctx context.Context, id uint64, fields map[string]any) error {
	return r.db.WithContext(ctx).Model(&model.ImprovementItem{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 删除本人改进项（归属由 service 层校验）。
func (r *improvementRepository) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.ImprovementItem{}).Error
}

// DeleteBySession 删除某场次关联的改进项（删课级联用）。
func (r *improvementRepository) DeleteBySession(ctx context.Context, sessionID uint64) error {
	return r.db.WithContext(ctx).Where("session_id = ?", sessionID).Delete(&model.ImprovementItem{}).Error
}

// ReflectionRepository 定义教学反思数据访问（N5）。
type ReflectionRepository interface {
	Create(ctx context.Context, rf *model.TeachingReflection) error
	GetByID(ctx context.Context, id uint64) (*model.TeachingReflection, error)
	List(ctx context.Context, teacherID, sessionID uint64) ([]model.TeachingReflection, error)
	ListBySession(ctx context.Context, sessionID uint64) ([]model.TeachingReflection, error)
	Update(ctx context.Context, id uint64, fields map[string]any) error
}

type reflectionRepository struct{ db *gorm.DB }

// NewReflectionRepository 构造教学反思仓储。
func NewReflectionRepository(db *gorm.DB) ReflectionRepository {
	return &reflectionRepository{db: db}
}

func (r *reflectionRepository) Create(ctx context.Context, rf *model.TeachingReflection) error {
	return r.db.WithContext(ctx).Create(rf).Error
}

func (r *reflectionRepository) GetByID(ctx context.Context, id uint64) (*model.TeachingReflection, error) {
	var rf model.TeachingReflection
	if err := r.db.WithContext(ctx).First(&rf, id).Error; err != nil {
		return nil, err
	}
	return &rf, nil
}

func (r *reflectionRepository) List(ctx context.Context, teacherID, sessionID uint64) ([]model.TeachingReflection, error) {
	var rows []model.TeachingReflection
	tx := r.db.WithContext(ctx).Where("teacher_id = ?", teacherID)
	if sessionID > 0 {
		tx = tx.Where("session_id = ?", sessionID)
	}
	err := tx.Order("created_at DESC, id DESC").Find(&rows).Error
	return rows, err
}

// ListBySession 按场次查全部教师的关联反思（供督导关闭清单时阅读）。
func (r *reflectionRepository) ListBySession(ctx context.Context, sessionID uint64) ([]model.TeachingReflection, error) {
	var rows []model.TeachingReflection
	err := r.db.WithContext(ctx).Where("session_id = ?", sessionID).
		Order("created_at DESC, id DESC").Find(&rows).Error
	return rows, err
}

func (r *reflectionRepository) Update(ctx context.Context, id uint64, fields map[string]any) error {
	return r.db.WithContext(ctx).Model(&model.TeachingReflection{}).Where("id = ?", id).Updates(fields).Error
}

// NotificationRepository 定义站内信数据访问（B12）。
type NotificationRepository interface {
	Create(ctx context.Context, n *model.UserNotification) error
	List(ctx context.Context, userID uint64, page, pageSize int) ([]model.UserNotification, int64, int64, error)
	MarkRead(ctx context.Context, id, userID uint64) error
	MarkAllRead(ctx context.Context, userID uint64) error
}

type notificationRepository struct{ db *gorm.DB }

// NewNotificationRepository 构造站内信仓储。
func NewNotificationRepository(db *gorm.DB) NotificationRepository {
	return &notificationRepository{db: db}
}

func (r *notificationRepository) Create(ctx context.Context, n *model.UserNotification) error {
	return r.db.WithContext(ctx).Create(n).Error
}

// List 返回分页列表、未读数与总数。
func (r *notificationRepository) List(ctx context.Context, userID uint64, page, pageSize int) ([]model.UserNotification, int64, int64, error) {
	base := r.db.WithContext(ctx).Model(&model.UserNotification{}).Where("user_id = ?", userID)
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, 0, err
	}
	var unread int64
	if err := r.db.WithContext(ctx).Model(&model.UserNotification{}).
		Where("user_id = ? AND is_read = 0", userID).Count(&unread).Error; err != nil {
		return nil, 0, 0, err
	}
	var rows []model.UserNotification
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).
		Order("created_at DESC, id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error
	return rows, total, unread, err
}

func (r *notificationRepository) MarkRead(ctx context.Context, id, userID uint64) error {
	return r.db.WithContext(ctx).Model(&model.UserNotification{}).
		Where("id = ? AND user_id = ?", id, userID).Update("is_read", true).Error
}

func (r *notificationRepository) MarkAllRead(ctx context.Context, userID uint64) error {
	return r.db.WithContext(ctx).Model(&model.UserNotification{}).
		Where("user_id = ? AND is_read = 0", userID).Update("is_read", true).Error
}
