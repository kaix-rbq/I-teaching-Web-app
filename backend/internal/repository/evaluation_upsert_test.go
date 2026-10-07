package repository

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"aijiaoxue-api/internal/model"
)

// TestEvaluationUpsertColumns 是不依赖数据库的回归守卫。
//
// 背景（真实缺陷）：督导与智能体两条写入路径各自维护 `DoUpdates` 列清单，
// 智能体侧漏掉了 comment/highlights/improvements/suggestions 四列。
// 后果是**静默**的——重新评价同一场次时分数与证据更新了，评语却停留在上一轮，
// 而 MySQL 不会报任何错。两处清单现已合并为 evaluationUpsertColumns。
func TestEvaluationUpsertColumns(t *testing.T) {
	// 缺任一列都会导致「重新评价时该列不被覆盖」。
	mustHave := []string{
		// 维度分与总分
		"objective_score", "content_score", "interaction_score",
		"organization_score", "frontier_score", "total_score",
		// AI 专属
		"ai_model_version", "ai_confidence", "evidence",
		// 口径版本
		"formula_version",
		// 🔴 四段评语：本次缺陷漏掉的正是这四列
		"comment", "highlights", "improvements", "suggestions",
	}

	set := make(map[string]struct{}, len(evaluationUpsertColumns))
	for _, c := range evaluationUpsertColumns {
		set[c] = struct{}{}
	}
	for _, want := range mustHave {
		if _, ok := set[want]; !ok {
			t.Errorf("evaluationUpsertColumns 缺列 %q：重新评价时该列不会被覆盖", want)
		}
	}

	// 冲突键必须与 uk_eval 一致，否则覆盖语义失效（会退化为插入或报 1062）。
	require.Len(t, evaluationConflictColumns, 3)
	keys := []string{
		evaluationConflictColumns[0].Name,
		evaluationConflictColumns[1].Name,
		evaluationConflictColumns[2].Name,
	}
	assert.ElementsMatch(t, []string{"session_id", "evaluator_type", "evaluator_id"}, keys)
}

// TestUpsertAgentEvaluationOverwritesCommentary 是需要真实 MySQL 的集成测试。
//
// 验收口径（阶段一 1.1）：**重复 upsert 后评语仍在**。测试对同一场次连续写入两次
// 不同的评语，断言第二次的值生效——修复前四段评语不会被覆盖，断言会失败。
//
// 默认跳过，设置 AIJIAOXUE_TEST_MYSQL_DSN 后启用：
//
//	AIJIAOXUE_TEST_MYSQL_DSN='aijiaoxue:aijiaoxue_dev@tcp(127.0.0.1:3307)/aijiaoxue?charset=utf8mb4&parseTime=True&loc=Local' \
//	  go test ./internal/repository/ -run TestUpsertAgentEvaluation -v
//
// 测试会先快照该场次原有的 agent 行，结束后原样恢复，不污染种子数据。
func TestUpsertAgentEvaluationOverwritesCommentary(t *testing.T) {
	dsn := os.Getenv("AIJIAOXUE_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("未设置 AIJIAOXUE_TEST_MYSQL_DSN，跳过需要 MySQL 的集成测试")
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err, "连接测试数据库失败")

	ctx := context.Background()
	repo := NewRecordingRepository(db)

	// 复用已有场次，避免为构造 users/courses/teaching_sessions 外键链铺大量夹具。
	var sessionID uint64
	require.NoError(t, db.WithContext(ctx).Raw("SELECT id FROM teaching_sessions ORDER BY id LIMIT 1").
		Scan(&sessionID).Error)
	require.NotZero(t, sessionID, "数据库里没有 teaching_sessions，请先执行 make seed")

	const agentWhere = "session_id = ? AND evaluator_type = ? AND evaluator_id = 0"

	// 快照原有 agent 行（可能不存在），测试结束后恢复。
	var snapshot model.Evaluation
	hadRow := db.WithContext(ctx).
		Where(agentWhere, sessionID, model.EvaluatorAgent).
		Take(&snapshot).Error == nil
	t.Cleanup(func() {
		if hadRow {
			// snapshot 由 Take 载入，主键已填充，Save 按主键整行还原。
			_ = db.WithContext(ctx).Save(&snapshot).Error
			return
		}
		_ = db.WithContext(ctx).
			Where(agentWhere, sessionID, model.EvaluatorAgent).
			Delete(&model.Evaluation{}).Error
	})

	build := func(comment, highlights, improvements, suggestions string) *model.Evaluation {
		evidence := `{"schemaVersion":1,"citedChunks":[],"dimensions":{},"notObservable":[],"promptVersion":"v1"}`
		objective, content := uint8(4), uint8(3)
		return &model.Evaluation{
			SessionID:      sessionID,
			EvaluatorType:  model.EvaluatorAgent,
			EvaluatorID:    0,
			AIModelVersion: "test-model",
			FormulaVersion: "v1",
			// interaction 故意留 NULL：智能体无法观测的维度必须允许为空。
			ObjectiveScore:   &objective,
			ContentScore:     &content,
			InteractionScore: nil,
			Evidence:         &evidence,
			Comment:          comment,
			Highlights:       highlights,
			Improvements:     improvements,
			Suggestions:      suggestions,
		}
	}

	read := func() model.Evaluation {
		var got model.Evaluation
		require.NoError(t, db.WithContext(ctx).
			Where(agentWhere, sessionID, model.EvaluatorAgent).
			Take(&got).Error)
		return got
	}

	// 第一次：写入初版评语。
	require.NoError(t, repo.UpsertAgentEvaluation(ctx, build("初版评语", "初版亮点", "初版待改进", "初版建议")))
	require.Equal(t, "初版评语", read().Comment, "首次写入应生效")

	var rowCount int64
	require.NoError(t, db.WithContext(ctx).Model(&model.Evaluation{}).
		Where(agentWhere, sessionID, model.EvaluatorAgent).Count(&rowCount).Error)
	require.EqualValues(t, 1, rowCount, "同一场次只应有一条 agent 行（uk_eval 覆盖而非追加）")

	// 第二次：同一场次重新评价，四段评语必须整行覆盖。
	require.NoError(t, repo.UpsertAgentEvaluation(ctx, build("改版评语", "改版亮点", "改版待改进", "改版建议")))
	second := read()

	assert.Equal(t, "改版评语", second.Comment, "🔴 重复 upsert 时 comment 未被覆盖（DoUpdates 漏列）")
	assert.Equal(t, "改版亮点", second.Highlights, "🔴 重复 upsert 时 highlights 未被覆盖")
	assert.Equal(t, "改版待改进", second.Improvements, "🔴 重复 upsert 时 improvements 未被覆盖")
	assert.Equal(t, "改版建议", second.Suggestions, "🔴 重复 upsert 时 suggestions 未被覆盖")

	// 分数与证据同样应覆盖。
	assert.Equal(t, "test-model", second.AIModelVersion)
	require.NotNil(t, second.Evidence)
	// 无法观测的维度保持 NULL，不得被写成 0（前端靠 null 判断"该源无法评价"）。
	assert.Nil(t, second.InteractionScore, "未观测的维度必须保持 NULL")
}
