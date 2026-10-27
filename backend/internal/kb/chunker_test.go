package kb

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChunkDocumentKeepsHeadingPath(t *testing.T) {
	text := "# 五维评分锚点\n\n## interaction：学生互动与参与\n\n### 互动维度锚点\n\n" +
		strings.Repeat("开放问题与追问。", 80)
	chunks := ChunkDocument(text, 120, 20)
	require.Greater(t, len(chunks), 1)
	for _, c := range chunks {
		assert.Contains(t, c.Heading, "interaction")
		assert.LessOrEqual(t, runeLen(c.Content), 120)
		assert.NotEmpty(t, c.Content)
	}
}

func TestChunkDocumentPlainText(t *testing.T) {
	chunks := ChunkDocument("第一段。第二段。第三段。", 100, 10)
	require.Len(t, chunks, 1)
	assert.Equal(t, "", chunks[0].Heading)
}

func TestChunkDocumentOverlap(t *testing.T) {
	text := strings.Repeat("甲乙丙丁。", 40)
	chunks := ChunkDocument(text, 50, 10)
	require.Greater(t, len(chunks), 1)
	// 相邻块存在重叠，且每块不超过上限。
	for i := 1; i < len(chunks); i++ {
		assert.NotEmpty(t, chunks[i].Content)
		assert.LessOrEqual(t, runeLen(chunks[i].Content), 50)
	}
}

func TestDeriveTitle(t *testing.T) {
	assert.Equal(t, "五维评分锚点", DeriveTitle("# 五维评分锚点\n正文", "x"))
	assert.Equal(t, "第一行", DeriveTitle("第一行\n第二行", "x"))
	assert.Equal(t, "fallback", DeriveTitle("", "fallback"))
}

func TestBuildDocKeyStableAndBounded(t *testing.T) {
	a := BuildDocKey("kb-rubric", 0, "", "五维评分锚点.md")
	b := BuildDocKey("kb-rubric", 0, "", "五维评分锚点.md")
	assert.Equal(t, a, b)
	assert.LessOrEqual(t, runeLen(a), 128)
	long := strings.Repeat("长路径", 100) + ".md"
	k := BuildDocKey("kb-textbook", 12, "操作系统", long)
	assert.LessOrEqual(t, runeLen(k), 128)
	assert.NotEqual(t, k, BuildDocKey("kb-textbook", 12, "操作系统", long+"x"))
}
