package repository

import (
	"context"
	"strings"
	"time"

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
	// ListDueTeacherReportTasks 返回到达去抖时间的待生成教师报告任务。
	ListDueTeacherReportTasks(ctx context.Context, before time.Time) ([]model.AiTask, error)
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

// ListDueTeacherReportTasks 返回到达去抖窗口的教师报告任务。
func (r *aiTaskRepository) ListDueTeacherReportTasks(ctx context.Context, before time.Time) ([]model.AiTask, error) {
	var rows []model.AiTask
	err := r.db.WithContext(ctx).
		Where("kind = ? AND status = ? AND updated_at <= ?",
			model.AiTaskKindTeacherReport, model.AiTaskStatusPending, before).
		Order("id ASC").Find(&rows).Error
	return rows, err
}

// TeacherAiReportRepository 定义教师级 AI 报告数据访问（P3 / S2）。
type TeacherAiReportRepository interface {
	GetByTeacherSemester(ctx context.Context, teacherID uint64, semester string) (*model.TeacherAiReport, error)
	Get(ctx context.Context, teacherID uint64, semester string) (*model.TeacherAiReport, error)
	Upsert(ctx context.Context, report *model.TeacherAiReport) error
	MarkStatus(ctx context.Context, teacherID uint64, semester, status, errMsg string) error
}

type teacherAiReportRepository struct{ db *gorm.DB }

// NewTeacherAiReportRepository 构造教师级 AI 报告仓储。
func NewTeacherAiReportRepository(db *gorm.DB) TeacherAiReportRepository {
	return &teacherAiReportRepository{db: db}
}

func (r *teacherAiReportRepository) GetByTeacherSemester(ctx context.Context, teacherID uint64, semester string) (*model.TeacherAiReport, error) {
	return r.Get(ctx, teacherID, semester)
}

func (r *teacherAiReportRepository) Get(ctx context.Context, teacherID uint64, semester string) (*model.TeacherAiReport, error) {
	var rep model.TeacherAiReport
	err := r.db.WithContext(ctx).
		Where("teacher_id = ? AND semester = ? AND course_id = 0", teacherID, semester).
		Take(&rep).Error
	if err != nil {
		return nil, err
	}
	return &rep, nil
}

// teacherReportUpsertColumns 是同一 (teacher_id,semester,0) 行覆盖时需要更新的列。
var teacherReportUpsertColumns = []string{
	"formula_version", "ai_model_version", "prompt_version",
	"summary", "directions", "suggestions", "highlights", "trend_comment",
	"evidence", "source_session_ids", "sample", "confidence",
	"status", "error_message", "generated_at",
}

// Upsert 按 uk_ai_report(teacher_id,semester,course_id) 整行覆盖，固定 course_id=0。
// JSON 列的空串会被归一化为 NULL，避免 MySQL 对 JSON 列报 ERROR 3140。
func (r *teacherAiReportRepository) Upsert(ctx context.Context, report *model.TeacherAiReport) error {
	row := *report
	row.CourseID = 0
	row.Directions = nilIfBlankJSON(row.Directions)
	row.Suggestions = nilIfBlankJSON(row.Suggestions)
	row.Evidence = nilIfBlankJSON(row.Evidence)
	row.SourceSessionIDs = nilIfBlankJSON(row.SourceSessionIDs)
	row.Sample = nilIfBlankJSON(row.Sample)
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "teacher_id"}, {Name: "semester"}, {Name: "course_id"}},
			DoUpdates: clause.AssignmentColumns(teacherReportUpsertColumns),
		}).Create(&row).Error
}

// MarkStatus 只更新状态与错误信息；报告内容保留，直到下一次成功生成覆盖。
func (r *teacherAiReportRepository) MarkStatus(ctx context.Context, teacherID uint64, semester, status, errMsg string) error {
	return r.db.WithContext(ctx).
		Model(&model.TeacherAiReport{}).
		Where("teacher_id = ? AND semester = ? AND course_id = 0", teacherID, semester).
		Updates(map[string]any{"status": status, "error_message": errMsg}).Error
}

// nilIfBlankJSON 保证 JSON 列只写入有效 JSON 或 NULL，绝不写入空串。
func nilIfBlankJSON(v *string) *string {
	if v == nil || strings.TrimSpace(*v) == "" {
		return nil
	}
	return v
}
