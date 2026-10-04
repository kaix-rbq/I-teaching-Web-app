package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"aijiaoxue-api/internal/model"
)

// DraftListRow 是草稿列表行：草稿本体 + 授课记录/课程/教师上下文（一次联表取齐）。
type DraftListRow struct {
	model.EvaluationDraft
	CourseID    uint64    `gorm:"column:course_id"`
	CourseCode  string    `gorm:"column:course_code"`
	CourseName  string    `gorm:"column:course_name"`
	TeacherName string    `gorm:"column:teacher_name"`
	SessionDate time.Time `gorm:"column:session_date"`
	Period      string    `gorm:"column:period"`
	Topic       string    `gorm:"column:topic"`
}

// DraftListParams 是草稿列表查询参数（按课程名模糊查询）。
type DraftListParams struct {
	SupervisorID uint64
	Keyword      string
	Page         int
	PageSize     int
}

// DraftRepository 定义评估草稿数据访问。
type DraftRepository interface {
	Upsert(ctx context.Context, d *model.EvaluationDraft) error
	GetBySession(ctx context.Context, sessionID, supervisorID uint64) (*model.EvaluationDraft, error)
	GetByID(ctx context.Context, id, supervisorID uint64) (*DraftListRow, error)
	ListBySupervisor(ctx context.Context, p DraftListParams) ([]DraftListRow, int64, error)
	RecentBySupervisor(ctx context.Context, supervisorID uint64, limit int) ([]DraftListRow, error)
	Delete(ctx context.Context, id, supervisorID uint64) error
}

type draftRepository struct {
	db *gorm.DB
}

// NewDraftRepository 构造草稿仓储。
func NewDraftRepository(db *gorm.DB) DraftRepository {
	return &draftRepository{db: db}
}

// Upsert 按 uk_draft(session_id, supervisor_id) 幂等保存草稿。
func (r *draftRepository) Upsert(ctx context.Context, d *model.EvaluationDraft) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "session_id"}, {Name: "supervisor_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"objective_score", "content_score", "interaction_score",
				"organization_score", "frontier_score",
				"comment", "highlights", "improvements", "suggestions",
			}),
		}).Create(d).Error
}

func (r *draftRepository) GetBySession(ctx context.Context, sessionID, supervisorID uint64) (*model.EvaluationDraft, error) {
	var d model.EvaluationDraft
	err := r.db.WithContext(ctx).
		Where("session_id = ? AND supervisor_id = ?", sessionID, supervisorID).
		Take(&d).Error
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *draftRepository) GetByID(ctx context.Context, id, supervisorID uint64) (*DraftListRow, error) {
	var row DraftListRow
	err := r.draftSelect(ctx, supervisorID).
		Where("d.id = ?", id).
		Take(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// draftBase 是草稿联表的公共骨架（不含 SELECT 列表，避免与 Count 的 count(*) 冲突）。
func (r *draftRepository) draftBase(ctx context.Context, supervisorID uint64) *gorm.DB {
	return r.db.WithContext(ctx).
		Table("evaluation_drafts AS d").
		Joins("JOIN teaching_sessions AS ts ON ts.id = d.session_id").
		Joins("JOIN courses AS c ON c.id = ts.course_id").
		Joins("JOIN users AS u ON u.id = ts.teacher_id").
		Where("d.supervisor_id = ?", supervisorID)
}

// draftSelect 在骨架之上补全列表/详情所需列。
func (r *draftRepository) draftSelect(ctx context.Context, supervisorID uint64) *gorm.DB {
	return r.draftBase(ctx, supervisorID).
		Select(`d.*, ts.course_id, ts.session_date, ts.period, ts.topic,
			c.code AS course_code, c.name AS course_name, u.name AS teacher_name`)
}

func (r *draftRepository) ListBySupervisor(ctx context.Context, p DraftListParams) ([]DraftListRow, int64, error) {
	base := r.draftBase(ctx, p.SupervisorID)
	if p.Keyword != "" {
		base = base.Where("c.name LIKE ?", "%"+p.Keyword+"%")
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	rowsQuery := r.draftSelect(ctx, p.SupervisorID)
	if p.Keyword != "" {
		rowsQuery = rowsQuery.Where("c.name LIKE ?", "%"+p.Keyword+"%")
	}
	var rows []DraftListRow
	err := rowsQuery.
		Order("d.updated_at DESC, d.id DESC").
		Offset((p.Page - 1) * p.PageSize).
		Limit(p.PageSize).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *draftRepository) RecentBySupervisor(ctx context.Context, supervisorID uint64, limit int) ([]DraftListRow, error) {
	var rows []DraftListRow
	err := r.draftSelect(ctx, supervisorID).
		Order("d.updated_at DESC, d.id DESC").
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// Delete 按草稿 id + 所属督导删除，返回受影响行数由 service 判定越权/不存在。
func (r *draftRepository) Delete(ctx context.Context, id, supervisorID uint64) error {
	return r.db.WithContext(ctx).
		Where("id = ? AND supervisor_id = ?", id, supervisorID).
		Delete(&model.EvaluationDraft{}).Error
}
