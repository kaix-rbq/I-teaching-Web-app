package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleTranscript = "今天我们讲用户故事。请大家思考：什么是好的用户故事？谁来回答一下。小明说：要能独立交付。"

func TestParseS1ResponseValid(t *testing.T) {
	raw := `{
	  "dimensions": {
	    "objective":    {"score": 4, "confidence": 0.55, "quotes": [{"start": 0, "end": 5, "quote": "今天我们讲用户故事"}]},
	    "content":      {"score": 3, "confidence": 0.60, "quotes": [{"start": 0, "end": 5, "quote": "今天我们讲用户故事"}]},
	    "interaction":  {"score": 2, "confidence": 0.75, "quotes": [{"start": 6, "end": 20, "quote": "请大家思考：什么是好的用户故事"}]},
	    "organization": {"score": 4, "confidence": 0.70, "quotes": [{"start": 0, "end": 5, "quote": "今天我们讲用户故事"}]},
	    "frontier":     {"score": null, "confidence": 0.0, "quotes": []}
	  },
	  "confidence": 0.68,
	  "comment": "总体良好", "highlights": "提问开放", "improvements": "互动不足", "suggestions": "留思考时间",
	  "notObservable": ["frontier"],
	  "citedChunks": ["kb-rubric#3"],
	  "riskTags": ["学生参与度低"]
	}`

	res, err := ParseS1Response(raw, sampleTranscript, "v1", "qwen-max")
	require.NoError(t, err)
	require.NotNil(t, res.Objective)
	assert.Equal(t, 4, *res.Objective)
	require.NotNil(t, res.Interaction)
	assert.Equal(t, 2, *res.Interaction)
	assert.Nil(t, res.Frontier)
	assert.Equal(t, 0.68, res.Confidence)
	assert.Equal(t, "qwen-max", res.ModelVersion)
	assert.Equal(t, "v1", res.Evidence.PromptVersion)
	assert.Equal(t, []string{"frontier"}, res.Evidence.NotObservable)
	assert.Equal(t, []string{"kb-rubric#3"}, res.Evidence.CitedChunks)
	assert.Equal(t, []string{"学生参与度低"}, res.Evidence.RiskTags)
	assert.Len(t, res.Evidence.Dimensions["interaction"].Quotes, 1)
}

func TestParseS1ResponseStripsFences(t *testing.T) {
	raw := "```json\n{\"dimensions\":{},\"confidence\":0.5,\"comment\":\"ok\"}\n```"
	res, err := ParseS1Response(raw, sampleTranscript, "v1", "m")
	require.NoError(t, err)
	assert.Equal(t, "ok", res.Comment)
	assert.Equal(t, []string{"objective", "content", "interaction", "organization", "frontier"}, res.Evidence.NotObservable,
		"空维度全部视为无法观测")
}

func TestParseS1ResponseOutOfRangeScoreBecomesNull(t *testing.T) {
	raw := `{"dimensions":{"objective":{"score":9,"confidence":0.5,"quotes":[{"quote":"今天我们讲用户故事"}]}}}`
	res, err := ParseS1Response(raw, sampleTranscript, "v1", "m")
	require.NoError(t, err)
	assert.Nil(t, res.Objective, "越界分数必须降级为 null")
	assert.Contains(t, res.Evidence.NotObservable, "objective")
}

func TestParseS1ResponseUnlocatableQuoteDropsScore(t *testing.T) {
	// quote 无法在转写中定位（模型编造）：无证据不许给分 → 该维置 null 并计入 notObservable。
	raw := `{"dimensions":{"content":{"score":5,"confidence":0.9,"quotes":[{"quote":"这句话不存在于转写中"}]}}}`
	res, err := ParseS1Response(raw, sampleTranscript, "v1", "m")
	require.NoError(t, err)
	assert.Nil(t, res.Content)
	assert.Contains(t, res.Evidence.NotObservable, "content")
	assert.Empty(t, res.Evidence.Dimensions["content"].Quotes, "定位失败的引用必须丢弃")
}

func TestParseS1ResponseRejectsNonJSON(t *testing.T) {
	_, err := ParseS1Response("没有 JSON 对象", sampleTranscript, "v1", "m")
	require.Error(t, err)
}
