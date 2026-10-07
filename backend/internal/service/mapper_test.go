package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestToEvidenceDTO 覆盖 evaluations.evidence 的解析边界。
//
// 设计取舍：证据属增强信息，解析失败必须退化为「无证据」而不是让整个
// GET /sessions/:id/evaluation 返回 500——一条脏 JSON 不该打掉整页评估数据。
func TestToEvidenceDTO(t *testing.T) {
	t.Run("督导行 NULL 返回 nil", func(t *testing.T) {
		assert.Nil(t, toEvidenceDTO(nil))
	})

	t.Run("空串与空白返回 nil", func(t *testing.T) {
		empty := ""
		blank := "   "
		assert.Nil(t, toEvidenceDTO(&empty))
		assert.Nil(t, toEvidenceDTO(&blank))
	})

	t.Run("非法 JSON 退化为 nil 而不 panic", func(t *testing.T) {
		bad := `{不是合法 JSON`
		assert.NotPanics(t, func() {
			assert.Nil(t, toEvidenceDTO(&bad))
		})
	})

	t.Run("完整结构正确解析", func(t *testing.T) {
		raw := `{
		  "schemaVersion": 1,
		  "citedChunks": ["kb-rubric#3", "kb-textbook#12"],
		  "dimensions": {
		    "objective":   {"confidence": 0.55, "quotes": [{"start": 123.4, "end": 145.0, "quote": "这段话"}]},
		    "interaction": {"confidence": 0.75, "quotes": []}
		  },
		  "notObservable": ["frontier"],
		  "promptVersion": "v1"
		}`
		got := toEvidenceDTO(&raw)
		require.NotNil(t, got)

		assert.Equal(t, 1, got.SchemaVersion)
		assert.Equal(t, []string{"kb-rubric#3", "kb-textbook#12"}, got.CitedChunks)
		assert.Equal(t, "v1", got.PromptVersion)
		assert.Equal(t, []string{"frontier"}, got.NotObservable)

		require.Contains(t, got.Dimensions, "objective")
		obj := got.Dimensions["objective"]
		assert.InDelta(t, 0.55, obj.Confidence, 1e-9)
		require.Len(t, obj.Quotes, 1)
		// 秒级时间戳与 transcripts.segments 对齐
		assert.InDelta(t, 123.4, obj.Quotes[0].Start, 1e-9)
		assert.InDelta(t, 145.0, obj.Quotes[0].End, 1e-9)
		assert.Equal(t, "这段话", obj.Quotes[0].Quote)

		// 无法观测的维度可以没有引用
		assert.Empty(t, got.Dimensions["interaction"].Quotes)
	})

	t.Run("字段缺失时零值可用而非报错", func(t *testing.T) {
		// 早期数据可能只有部分字段；缺失字段应为零值，不得导致整体失败。
		raw := `{"schemaVersion":1}`
		got := toEvidenceDTO(&raw)
		require.NotNil(t, got)
		assert.Equal(t, 1, got.SchemaVersion)
		assert.Empty(t, got.CitedChunks)
		assert.Empty(t, got.Dimensions)
		assert.Empty(t, got.PromptVersion)
	})
}
