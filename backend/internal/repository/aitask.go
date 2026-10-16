package repository

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"aijiaoxue-api/internal/model"
)

// AiTaskRepository 定义 AI 任务数据访问（D3）。
type AiTaskRepository interface {
	GetByID(ctx context.Context, id uint64) (*model.AiTask, error)
	GetByKindRef(ctx context.Context, kind, refType string, refID uint64) (*model.AiTask, error)
	CreateIfAbsent(ctx context.Context, t *model.AiTask) error
	Update(ctx context.Context, id uint64, fields map[string]any) error
	// ListResumable 返回进程重启后需重新入队的任务（pending/running）。
	ListResumable(ctx context.Context) ([]model.AiTask, error)
}

type aiTaskRepository struct{ db *gorm.DB }

// NewAiTaskRepository 构造 AI 任务仓储。
func NewAiTaskRepository(db *gorm.DB) AiTaskRepository { return &aiTaskRepository{db: db} }

func (r *aiTaskRepository) GetByID(ctx context.Context, id uint64) (*model.AiTask, error) {
	var t model.AiTask
	if err := r.db.WithContext(ctx).First(&t, id).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *aiTaskRepository) GetByKindRef(ctx context.Context, kind, refType string, refID uint64) (*model.AiTask, error) {
	var t model.AiTask
	err := r.db.WithContext(ctx).
		Where("kind = ? AND ref_type = ? AND ref_id = ?", kind, refType, refID).
		Take(&t).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// CreateIfAbsent 依赖 uk_ai_task 幂等：已存在则不改动（DoNothing），保证「不重复建」。
func (r *aiTaskRepository) CreateIfAbsent(ctx context.Context, t *model.AiTask) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "kind"}, {Name: "ref_type"}, {Name: "ref_id"}}, DoNothing: true}).
		Create(t).Error
}

func (r *aiTaskRepository) Update(ctx context.Context, id uint64, fields map[string]any) error {
	return r.db.WithContext(ctx).Model(&model.AiTask{}).Where("id = ?", id).Updates(fields).Error
}

func (r *aiTaskRepository) ListResumable(ctx context.Context) ([]model.AiTask, error) {
	var rows []model.AiTask
	err := r.db.WithContext(ctx).
		Where("status IN ?", []string{model.AiTaskStatusPending, model.AiTaskStatusRunning}).
		Order("id ASC").Find(&rows).Error
	return rows, err
}

// TeacherAiReportRepository 定义教师级 AI 报告数据访问（本轮仅落表，生成链路顺延）。
type TeacherAiReportRepository interface {
	GetByTeacherSemester(ctx context.Context, teacherID uint64, semester string) (*model.TeacherAiReport, error)
}

type teacherAiReportRepository struct{ db *gorm.DB }

// NewTeacherAiReportRepository 构造教师级 AI 报告仓储。
func NewTeacherAiReportRepository(db *gorm.DB) TeacherAiReportRepository {
	return &teacherAiReportRepository{db: db}
}

func (r *teacherAiReportRepository) GetByTeacherSemester(ctx context.Context, teacherID uint64, semester string) (*model.TeacherAiReport, error) {
	var rep model.TeacherAiReport
	err := r.db.WithContext(ctx).
		Where("teacher_id = ? AND semester = ? AND course_id = 0", teacherID, semester).
		Take(&rep).Error
	if err != nil {
		return nil, err
	}
	return &rep, nil
}
