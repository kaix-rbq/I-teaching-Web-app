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

// TestP1aProjectionIntegration 用真实 MySQL 事务验证 P1a 三条查询投影：
//  1. ListByCourse.supervisor_evaluated 只由督导评价行决定；
//  2. ListPending 只排除已有 agent 评价行的场次；
//  3. ListPlans.evaluated 只由督导评价行决定，AI 评价不会置 true。
//
// 全部写入在事务内完成并回滚，不污染种子数据。默认跳过，设置
// AIJIAOXUE_TEST_MYSQL_DSN 后启用（与 evaluation_upsert_test.go 同一开关）。
func TestP1aProjectionIntegration(t *testing.T) {
	dsn := os.Getenv("AIJIAOXUE_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("未设置 AIJIAOXUE_TEST_MYSQL_DSN，跳过需要 MySQL 的集成测试")
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err, "连接测试数据库失败")

	ctx := context.Background()
	tx := db.Begin()
	require.NoError(t, tx.Error)
	t.Cleanup(func() { _ = tx.Rollback().Error })

	var teacherID, supervisorID, classID uint64
	require.NoError(t, tx.Raw("SELECT teacher_id FROM courses WHERE id = 1").Scan(&teacherID).Error)
	require.NotZero(t, teacherID, "课程 1 不存在或没有授课教师，请先准备种子数据")
	require.NoError(t, tx.Raw("SELECT id FROM users WHERE role = 'supervisor' ORDER BY id LIMIT 1").
		Scan(&supervisorID).Error)
	require.NotZero(t, supervisorID, "数据库里没有 supervisor 用户")
	require.NoError(t, tx.Raw("SELECT id FROM course_classes WHERE course_id = 1 ORDER BY id LIMIT 1").
		Scan(&classID).Error)
	require.NotZero(t, classID, "课程 1 没有开课班级")

	// 计划与授课记录使用未来日期，避免与种子数据 uk_session 冲突。
	const date = "2030-01-01"
	plan := &model.SupervisionPlan{
		CourseID: 1, SupervisorID: supervisorID,
		PlannedDate: time.Date(2030, 1, 1, 0, 0, 0, 0, time.Local),
		Status:      model.PlanStatusPlanned,
	}
	require.NoError(t, tx.Create(plan).Error)

	session := &model.TeachingSession{
		CourseID: 1, ClassID: classID, TeacherID: teacherID,
		SessionDate: time.Date(2030, 1, 1, 0, 0, 0, 0, time.Local),
		Period:      "集成测试节次", Topic: "P1a 投影验证",
		PlanID: &plan.ID, Status: model.SessionStatusScheduled,
	}
	require.NoError(t, tx.Create(session).Error)

	// 第二场只写督导观察、不写 AI：必须仍出现在待 AI 评析列表（督导状态不影响该队列）。
	pendingOnly := &model.TeachingSession{
		CourseID: 1, ClassID: classID, TeacherID: teacherID,
		SessionDate: time.Date(2030, 1, 1, 0, 0, 0, 0, time.Local),
		Period:      "集成测试节次2", Topic: "仅督导已评",
		Status: model.SessionStatusScheduled,
	}
	require.NoError(t, tx.Create(pendingOnly).Error)

	sessions := NewSessionRepository(tx)
	evals := NewEvaluationRepository(tx)
	plans := NewSupervisionRepository(tx)

	findSession := func(t *testing.T) SessionListRow {
		t.Helper()
		rows, _, lerr := sessions.ListByCourse(ctx, SessionListParams{
			CourseID: 1, Page: 1, PageSize: 1000,
		})
		require.NoError(t, lerr)
		for _, row := range rows {
			if row.ID == session.ID {
				return row
			}
		}
		t.Fatalf("ListByCourse 未返回测试场次 %d", session.ID)
		return SessionListRow{}
	}
	findPlan := func(t *testing.T) PlanRow {
		t.Helper()
		rows, _, lerr := plans.ListPlans(ctx, PlanListParams{Page: 1, PageSize: 1000})
		require.NoError(t, lerr)
		for _, row := range rows {
			if row.ID == plan.ID {
				return row
			}
		}
		t.Fatalf("ListPlans 未返回测试计划 %d", plan.ID)
		return PlanRow{}
	}
	pendingContains := func(t *testing.T, sessionID uint64) bool {
		t.Helper()
		rows, lerr := sessions.ListPending(ctx, 1000)
		require.NoError(t, lerr)
		for _, row := range rows {
			if row.ID == sessionID {
				return true
			}
		}
		return false
	}

	// 只写 AI 评价：督导未评，场次应仍待 AI 列表之外（因为已有 agent 行），
	// 且计划 evaluated 必须保持 false。
	require.NoError(t, evals.Upsert(ctx, &model.Evaluation{
		SessionID: session.ID, EvaluatorType: model.EvaluatorAgent,
		FormulaVersion: "v1", Comment: "AI 评语",
	}))
	assert.False(t, findSession(t).SupervisorEvaluated, "只有 AI 评价时 supervisorEvaluated 必须为 false")
	assert.False(t, pendingContains(t, session.ID), "已有 agent 行的场次必须从待 AI 评析列表移除")
	assert.False(t, findPlan(t).Evaluated, "只有 AI 评价时 PlanItem.evaluated 必须为 false")

	// 仅督导已评、AI 未评的场次：待 AI 列表必须仍然包含。
	assert.True(t, pendingContains(t, pendingOnly.ID), "无 agent 行的场次必须出现在待 AI 评析列表")
	require.NoError(t, evals.Upsert(ctx, &model.Evaluation{
		SessionID: pendingOnly.ID, EvaluatorType: model.EvaluatorSupervisor,
		EvaluatorID: supervisorID, FormulaVersion: "v1", Comment: "督导观察",
	}))
	assert.True(t, pendingContains(t, pendingOnly.ID), "督导已评不改变待 AI 评析队列")

	// 督导提交观察后：督导投影 true；待 AI 列表仍不包含（AI 行已存在）；计划变 true。
	require.NoError(t, evals.Upsert(ctx, &model.Evaluation{
		SessionID: session.ID, EvaluatorType: model.EvaluatorSupervisor,
		EvaluatorID: supervisorID, FormulaVersion: "v1", Comment: "督导观察",
	}))
	assert.True(t, findSession(t).SupervisorEvaluated, "督导提交后 supervisorEvaluated 必须为 true")
	assert.False(t, pendingContains(t, session.ID), "督导已评不会让场次重新进入待 AI 评析列表")
	assert.True(t, findPlan(t).Evaluated, "督导提交后 PlanItem.evaluated 必须为 true")
}

// TestAgentSuggestionsQueryIntegration 用真实 MySQL 事务验证建议查询的
// formula_version / semester / course 过滤与 session_date DESC, evaluation.id DESC 排序。
// 全部写入回滚，默认跳过（开关同 TestP1aProjectionIntegration）。
func TestAgentSuggestionsQueryIntegration(t *testing.T) {
	dsn := os.Getenv("AIJIAOXUE_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("未设置 AIJIAOXUE_TEST_MYSQL_DSN，跳过需要 MySQL 的集成测试")
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err, "连接测试数据库失败")

	ctx := context.Background()
	tx := db.Begin()
	require.NoError(t, tx.Error)
	t.Cleanup(func() { _ = tx.Rollback().Error })

	type ctxRow struct {
		TeacherID    uint64
		ClassID      uint64
		DepartmentID uint64
		Semester     string
	}
	load := func(t *testing.T, courseID uint64) ctxRow {
		t.Helper()
		var row ctxRow
		require.NoError(t, tx.Raw(`SELECT c.teacher_id, c.department_id, cc.id AS class_id, c.semester
			FROM courses c JOIN course_classes cc ON cc.course_id = c.id
			WHERE c.id = ? ORDER BY cc.id LIMIT 1`, courseID).Scan(&row).Error)
		require.NotZero(t, row.ClassID, "课程 %d 没有开课班级", courseID)
		return row
	}
	// 课程 8 是种子中的 2025-2026-2 课程，无既有场次/评价，适合承载测试夹具。
	course8 := load(t, 8)

	// 同学期的第二门课程用于验证 courseId 过滤（不依赖种子数据）。
	otherCourse := &model.Course{
		Code: "ZZ9001", Name: "建议查询集成测试", Credit: 2, Hours: 32,
		Semester: course8.Semester, DepartmentID: course8.DepartmentID,
		TeacherID: course8.TeacherID, Status: model.CourseStatusOpen,
	}
	require.NoError(t, tx.Create(otherCourse).Error)
	otherClass := &model.ClassInfo{
		CourseID: otherCourse.ID, ClassName: "测试班",
		Schedule: "待定", Location: "待定",
	}
	require.NoError(t, tx.Create(otherClass).Error)

	makeSession := func(t *testing.T, courseID uint64, meta ctxRow, date, period string) *model.TeachingSession {
		t.Helper()
		s := &model.TeachingSession{
			CourseID: courseID, ClassID: meta.ClassID, TeacherID: meta.TeacherID,
			SessionDate: mustParseDate(t, date), Period: period,
			Status: model.SessionStatusEvaluated,
		}
		require.NoError(t, tx.Create(s).Error)
		return s
	}

	s1 := makeSession(t, 8, course8, "2030-02-01", "建议测试1")
	s2 := makeSession(t, 8, course8, "2030-02-03", "建议测试2")
	s3 := makeSession(t, 8, course8, "2030-02-03", "建议测试3")
	sOld := makeSession(t, 8, course8, "2030-02-04", "建议测试4")
	sOtherCourse := makeSession(t, otherCourse.ID, ctxRow{
		TeacherID: otherCourse.TeacherID, ClassID: otherClass.ID,
		Semester: otherCourse.Semester, DepartmentID: otherCourse.DepartmentID,
	}, "2030-02-02", "建议测试5")

	evals := NewEvaluationRepository(tx)
	upsert := func(t *testing.T, sessionID uint64, evaluatorType, formula, comment string) {
		t.Helper()
		require.NoError(t, evals.Upsert(ctx, &model.Evaluation{
			SessionID: sessionID, EvaluatorType: evaluatorType,
			FormulaVersion: formula, Comment: comment,
		}))
	}
	upsert(t, s1.ID, model.EvaluatorAgent, "v1", "最早 v1")
	upsert(t, s2.ID, model.EvaluatorAgent, "v1", "同日较早 v1")
	upsert(t, s3.ID, model.EvaluatorAgent, "v1", "同日较新 v1")
	upsert(t, sOld.ID, model.EvaluatorAgent, "v0", "旧口径")
	upsert(t, sOtherCourse.ID, model.EvaluatorAgent, "v1", "其他课程")
	// 督导行必须被排除。
	upsert(t, s1.ID, model.EvaluatorSupervisor, "v1", "督导行")

	rows, err := evals.ListAgentSuggestions(ctx, AgentSuggestionFilter{
		CourseID: 8, Semester: course8.Semester, FormulaVersion: "v1",
	})
	require.NoError(t, err)
	require.Len(t, rows, 3, "只取当前课程/学期/v1 的 agent 行")
	// 同一天按 evaluation.id DESC；其余按 session_date DESC。
	ordered := []uint64{rows[0].SessionID, rows[1].SessionID, rows[2].SessionID}
	assert.Equal(t, []uint64{s3.ID, s2.ID, s1.ID}, ordered,
		"必须按 session_date DESC, evaluation.id DESC")
	for _, row := range rows {
		assert.Equal(t, model.EvaluatorAgent, row.EvaluatorType, "督导行不得混入")
		assert.Equal(t, uint64(8), row.CourseID)
	}

	other, err := evals.ListAgentSuggestions(ctx, AgentSuggestionFilter{
		CourseID: otherCourse.ID, Semester: otherCourse.Semester, FormulaVersion: "v1",
	})
	require.NoError(t, err)
	require.Len(t, other, 1, "courseId 过滤必须生效")
	assert.Equal(t, sOtherCourse.ID, other[0].SessionID)

	// 学期不匹配时不得返回课程 8 的数据。
	none, err := evals.ListAgentSuggestions(ctx, AgentSuggestionFilter{
		CourseID: 8, Semester: "2026-2027-1", FormulaVersion: "v1",
	})
	require.NoError(t, err)
	assert.Empty(t, none)
}

// mustParseDate 解析 YYYY-MM-DD（集成测试夹具用）。
func mustParseDate(t *testing.T, value string) time.Time {
	t.Helper()
	date, err := time.Parse("2006-01-02", value)
	require.NoError(t, err)
	return date
}
