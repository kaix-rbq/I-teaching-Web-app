package agent

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildUserPromptIncludesKnowledge(t *testing.T) {
	prompt := buildUserPrompt(EvalInput{
		CourseName: "软件工程",
		Topic:      "用户故事",
		Transcript: "今天我们讲用户故事。",
		Knowledge: []RetrievedChunk{
			{ChunkID: "kb-rubric#13", Collection: "kb-rubric", Heading: "互动维度锚点（1–5 分行为描述）", Content: "开放问题与追问充分；学生回答后有针对性反馈。"},
			{ChunkID: "kb-rubric#14", Collection: "kb-rubric", Heading: "内容维度锚点", Content: "内容准确且理论联系实际。"},
		},
	})

	assert.Contains(t, prompt, "【知识库参考片段】")
	assert.Contains(t, prompt, "[kb-rubric#13] 互动维度锚点（1–5 分行为描述）")
	assert.Contains(t, prompt, "开放问题与追问充分；学生回答后有针对性反馈。")
	assert.Contains(t, prompt, "[kb-rubric#14]")
	assert.Less(t, strings.Index(prompt, "【知识库参考片段】"), strings.Index(prompt, "【脱敏转写文本】"))
}

func TestBuildUserPromptWithoutKnowledgeOmitsSection(t *testing.T) {
	prompt := buildUserPrompt(EvalInput{CourseName: "软件工程", Transcript: "课堂内容"})
	assert.NotContains(t, prompt, "知识库参考片段")
}

func TestFilterCitedChunks(t *testing.T) {
	knowledge := []RetrievedChunk{
		{ChunkID: "kb-rubric#13"},
		{ChunkID: "kb-rubric#14"},
	}
	cases := []struct {
		name      string
		cited     []string
		knowledge []RetrievedChunk
		want      []string
	}{
		{name: "命中", cited: []string{"kb-rubric#13"}, knowledge: knowledge, want: []string{"kb-rubric#13"}},
		{name: "编造被丢弃", cited: []string{"kb-rubric#999", "kb-rubric#13"}, knowledge: knowledge, want: []string{"kb-rubric#13"}},
		{name: "重复去重", cited: []string{"kb-rubric#13", " kb-rubric#13 "}, knowledge: knowledge, want: []string{"kb-rubric#13"}},
		{name: "无KB清空", cited: []string{"kb-rubric#13"}, knowledge: nil, want: []string{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := filterCitedChunks(tc.cited, tc.knowledge)
			require.NotNil(t, got)
			assert.Equal(t, tc.want, got)
		})
	}
}
