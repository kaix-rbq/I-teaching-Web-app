package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"aijiaoxue-api/internal/model"
)

// ScheduleEventRepository 定义主任事务数据访问。
type ScheduleEventRepository interface {
	List(ctx context.Context, ownerID uint64, from, to time.Time) ([]model.ScheduleEvent, error)
	GetByID(ctx context.Context, id uint64) (*model.ScheduleEvent, error)
	FindOverlap(ctx context.Context, ownerID uint64, date time.Time, start, end string, excludeID uint64) (*model.ScheduleEvent, error)
	Create(ctx context.Context, e *model.ScheduleEvent) error
	UpdateRemark(ctx context.Context, id uint64, remark string) error
	Delete(ctx context.Context, id uint64) error
}

type scheduleEventRepository struct{ db *gorm.DB }

// NewScheduleEventRepository 构造主任事务仓储。
func NewScheduleEventRepository(db *gorm.DB) ScheduleEventRepository {
	return &scheduleEventRepository{db: db}
}

func (r *scheduleEventRepository) List(ctx context.Context, ownerID uint64, from, to time.Time) ([]model.ScheduleEvent, error) {
	var rows []model.ScheduleEvent
	err := r.db.WithContext(ctx).
		Where("owner_user_id = ? AND event_date BETWEEN ? AND ?", ownerID, from, to).
		Order("event_date, start_time, id").
		Find(&rows).Error
	return rows, err
}

func (r *scheduleEventRepository) GetByID(ctx context.Context, id uint64) (*model.ScheduleEvent, error) {
	var e model.ScheduleEvent
	if err := r.db.WithContext(ctx).First(&e, id).Error; err != nil {
		return nil, err
	}
	return &e, nil
}

// FindOverlap 查找同 owner、同日期内与 [start,end) 重叠的事务。
// TIME 可直接做字符串比较（HH:MM:SS 字典序与时间序一致）。
func (r *scheduleEventRepository) FindOverlap(ctx context.Context, ownerID uint64, date time.Time, start, end string, excludeID uint64) (*model.ScheduleEvent, error) {
	var e model.ScheduleEvent
	q := r.db.WithContext(ctx).
		Where("owner_user_id = ? AND event_date = ? AND start_time < ? AND end_time > ?", ownerID, date, end, start)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	if err := q.Order("start_time").Take(&e).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &e, nil
}

func (r *scheduleEventRepository) Create(ctx context.Context, e *model.ScheduleEvent) error {
	return r.db.WithContext(ctx).Create(e).Error
}

func (r *scheduleEventRepository) UpdateRemark(ctx context.Context, id uint64, remark string) error {
	return r.db.WithContext(ctx).Model(&model.ScheduleEvent{}).
		Where("id = ?", id).Update("remark", remark).Error
}

func (r *scheduleEventRepository) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.ScheduleEvent{}).Error
}
