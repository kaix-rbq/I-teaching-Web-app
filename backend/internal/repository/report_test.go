package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"aijiaoxue-api/internal/model"
)

// TestTeacherReportRepositoryIntegration 用真实 MySQL 事务验证 P3 仓储：
// Upsert 同一 (teacher,semester,0) 幂等覆盖、JSON 空串不落库、MarkStatus 生效。
// 默认跳过，设置 AIJIAOXUE_TEST_MYSQL_DSN 后启用；事务回滚不污染种子数据。
func TestTeacherReportRepositoryIntegration(t *testing.T) {
	dsn := os.Getenv("AIJIAOXUE_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("未设置 AIJIAOXUE_TEST_MYSQL_DSN，跳过需要 MySQL 的集成测试")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	ctx := context.Background()
	tx := db.Begin()
	require.NoError(t, tx.Error)
	t.Cleanup(func() { _ = tx.Rollback().Error })

	repo := NewTeacherAiReportRepository(tx)
	const teacherID, semester = uint64(2), "2026-2027-1"
	summaryA := "A1 报告"
	directions := `["方向"]`
	suggestions := `["建议"]`
	first := &model.TeacherAiReport{
		TeacherID: teacherID, Semester: semester, CourseID: 0,
		FormulaVersion: "v1", AIModelVersion: "fake-model", PromptVersion: "v1",
		Summary: &summaryA, Directions: &directions, Suggestions: &suggestions,
		Status: model.AiTaskStatusDone,
	}
	require.NoError(t, repo.Upsert(ctx, first))
	got, err := repo.Get(ctx, teacherID, semester)
	require.NoError(t, err)
	assert.Equal(t, "A1 报告", stringValueForRepoTest(got.Summary))
	assert.Equal(t, `["方向"]`, stringValueForRepoTest(got.Directions))

	summaryB := "A2 报告"
	emptyJSON := ""
	second := &model.TeacherAiReport{
		TeacherID: teacherID, Semester: semester, CourseID: 0,
		FormulaVersion: "v1", AIModelVersion: "fake-model", PromptVersion: "v1",
		Summary: &summaryB, Directions: &emptyJSON, Suggestions: &suggestions,
		Status: model.AiTaskStatusDone,
	}
	require.NoError(t, repo.Upsert(ctx, second))

	var count int64
	require.NoError(t, tx.Model(&model.TeacherAiReport{}).
		Where("teacher_id = ? AND semester = ? AND course_id = 0", teacherID, semester).
		Count(&count).Error)
	assert.EqualValues(t, 1, count, "同一 teacher/semester/0 必须只有一行")

	got, err = repo.Get(ctx, teacherID, semester)
	require.NoError(t, err)
	assert.Equal(t, "A2 报告", stringValueForRepoTest(got.Summary))
	assert.Nil(t, got.Directions, "空 JSON 串必须归一化为 NULL，避免 MySQL ERROR 3140")
	require.NoError(t, repo.MarkStatus(ctx, teacherID, semester, model.AiTaskStatusFailed, "上游失败"))
	got, err = repo.Get(ctx, teacherID, semester)
	require.NoError(t, err)
	assert.Equal(t, model.AiTaskStatusFailed, got.Status)
	assert.Equal(t, "上游失败", got.ErrorMessage)
}

// TestListDueTeacherReportTasksIntegration 验证去抖到期查询的 kind/status/time 三个条件。
func TestListDueTeacherReportTasksIntegration(t *testing.T) {
	dsn := os.Getenv("AIJIAOXUE_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("未设置 AIJIAOXUE_TEST_MYSQL_DSN，跳过需要 MySQL 的集成测试")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	ctx := context.Background()
	tx := db.Begin()
	require.NoError(t, tx.Error)
	t.Cleanup(func() { _ = tx.Rollback().Error })

	old := time.Now().Add(-11 * time.Minute)
	now := time.Now()
	due := &model.AiTask{
		Kind: model.AiTaskKindTeacherReport, RefType: model.AiTaskRefTeacher, RefID: 990001,
		Status: model.AiTaskStatusPending, UpdatedAt: old,
	}
	wrongKind := &model.AiTask{
		Kind: model.AiTaskKindSessionEval, RefType: model.AiTaskRefSession, RefID: 990002,
		Status: model.AiTaskStatusPending, UpdatedAt: old,
	}
	notDue := &model.AiTask{
		Kind: model.AiTaskKindTeacherReport, RefType: model.AiTaskRefTeacher, RefID: 990003,
		Status: model.AiTaskStatusPending, UpdatedAt: now,
	}
	failed := &model.AiTask{
		Kind: model.AiTaskKindTeacherReport, RefType: model.AiTaskRefTeacher, RefID: 990004,
		Status: model.AiTaskStatusFailed, UpdatedAt: old,
	}
	for _, task := range []*model.AiTask{due, wrongKind, notDue, failed} {
		require.NoError(t, tx.Create(task).Error)
	}

	repo := NewAiTaskRepository(tx)
	rows, err := repo.ListDueTeacherReportTasks(ctx, now.Add(-10*time.Minute))
	require.NoError(t, err)

	ids := make(map[uint64]bool, len(rows))
	for _, row := range rows {
		ids[row.RefID] = true
	}
	assert.True(t, ids[990001], "到期 teacher_report/pending 必须返回")
	assert.False(t, ids[990002], "session_eval 不得返回")
	assert.False(t, ids[990003], "未超过去抖窗口不得返回")
	assert.False(t, ids[990004], "failed 任务不得返回")
}

// TestTeacherReportRowsIntegration 验证 S2 数据源只取 agent 行且带课程/维度/评语字段。
func TestTeacherReportRowsIntegration(t *testing.T) {
	dsn := os.Getenv("AIJIAOXUE_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("未设置 AIJIAOXUE_TEST_MYSQL_DSN，跳过需要 MySQL 的集成测试")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	ctx := context.Background()
	repo := NewEvaluationRepository(db)

	rows, err := repo.ListTeacherReportRows(ctx, 2, "2026-2027-1", "v1")
	require.NoError(t, err)
	require.NotEmpty(t, rows, "种子库教师 2 应有 agent 评价行")
	assert.NotEmpty(t, rows[0].CourseName)
	assert.NotEmpty(t, rows[0].SessionDate)

	comments, err := repo.ListSupervisorCommentsByTeacherSemester(ctx, 2, "2026-2027-1")
	require.NoError(t, err)
	assert.NotEmpty(t, comments)
}

func stringValueForRepoTest(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
