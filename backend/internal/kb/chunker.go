package kb

import (
	"regexp"
	"strings"
)

// DefaultChunkRunes / DefaultOverlapRunes 对应 RAG §5.3：约 500 汉字、10%–15% 重叠。
const (
	DefaultChunkRunes   = 500
	DefaultOverlapRunes = 60
)

// Chunk 是一个待入库切片；Heading 为标题路径，Content 为正文（含标题前缀，便于检索）。
type Chunk struct {
	Heading    string
	Content    string
	TokenCount int
}

var multiBlank = regexp.MustCompile(`\n{3,}`)

// ChunkDocument 按标题优先、段落/句子次之切分，保留标题路径。
func ChunkDocument(text string, maxRunes, overlapRunes int) []Chunk {
	if maxRunes <= 0 {
		maxRunes = DefaultChunkRunes
	}
	if overlapRunes < 0 || overlapRunes >= maxRunes {
		overlapRunes = DefaultOverlapRunes
		if overlapRunes >= maxRunes {
			overlapRunes = maxRunes / 5
		}
	}
	sections := splitSections(text)
	out := make([]Chunk, 0, len(sections))
	for _, sec := range sections {
		body := cleanBody(sec.body)
		if body == "" {
			continue
		}
		prefix := ""
		if sec.heading != "" {
			prefix = sec.heading + "\n"
		}
		bodyMax := maxRunes - runeLen(prefix)
		if bodyMax < 100 {
			// 标题路径很长时优先保住正文长度；标题仍写入 Heading 字段。
			prefix = ""
			bodyMax = maxRunes
		}
		parts := splitWithOverlap(body, bodyMax, overlapRunes)
		for _, part := range parts {
			content := part
			if prefix != "" {
				content = prefix + part
			}
			content = strings.TrimSpace(content)
			if content == "" {
				continue
			}
			out = append(out, Chunk{
				Heading:    truncateRunes(sec.heading, 255),
				Content:    content,
				TokenCount: runeLen(content), // 近似 token 数：中文按字计，供检索预算参考
			})
		}
	}
	return out
}

type section struct {
	heading string
	body    string
}

type headingLevel struct {
	level int
	text  string
}

// splitSections 解析 Markdown 标题层级；无标题文档作为单个 section。
func splitSections(text string) []section {
	lines := strings.Split(text, "\n")
	stack := make([]headingLevel, 0, 4)
	body := make([]string, 0, len(lines))
	out := make([]section, 0, 8)

	flush := func() {
		if len(body) == 0 {
			return
		}
		raw := strings.Join(body, "\n")
		if strings.TrimSpace(raw) == "" {
			body = body[:0]
			return
		}
		out = append(out, section{heading: headingPath(stack), body: raw})
		body = body[:0]
	}

	for _, line := range lines {
		if level, title, ok := parseHeading(line); ok {
			flush()
			for len(stack) > 0 && stack[len(stack)-1].level >= level {
				stack = stack[:len(stack)-1]
			}
			stack = append(stack, headingLevel{level: level, text: title})
			continue
		}
		body = append(body, line)
	}
	flush()
	if len(out) == 0 {
		if s := strings.TrimSpace(text); s != "" {
			out = append(out, section{body: s})
		}
	}
	return out
}

// parseHeading 识别 ATX 标题（# 1–6 个，后跟空格）。
func parseHeading(line string) (int, string, bool) {
	s := strings.TrimSpace(line)
	if !strings.HasPrefix(s, "#") {
		return 0, "", false
	}
	level := 0
	for level < len(s) && s[level] == '#' {
		level++
	}
	if level == 0 || level > 6 {
		return 0, "", false
	}
	if len(s) == level {
		return level, "", true
	}
	if s[level] != ' ' && s[level] != '\t' {
		return 0, "", false
	}
	title := strings.TrimSpace(s[level:])
	return level, title, title != ""
}

func headingPath(stack []headingLevel) string {
	if len(stack) == 0 {
		return ""
	}
	parts := make([]string, 0, len(stack))
	for _, h := range stack {
		if h.text != "" {
			parts = append(parts, h.text)
		}
	}
	return strings.Join(parts, " > ")
}

func cleanBody(s string) string {
	s = strings.TrimSpace(s)
	s = multiBlank.ReplaceAllString(s, "\n\n")
	return s
}

// splitWithOverlap 按 maxRunes 切块，优先在句号/问号/分号/换行处断开，并保留 overlapRunes 重叠。
func splitWithOverlap(text string, maxRunes, overlapRunes int) []string {
	r := []rune(strings.TrimSpace(text))
	if len(r) == 0 {
		return nil
	}
	if len(r) <= maxRunes {
		return []string{string(r)}
	}
	if overlapRunes < 0 {
		overlapRunes = 0
	}
	if overlapRunes >= maxRunes {
		overlapRunes = maxRunes / 5
	}

	out := make([]string, 0, len(r)/maxRunes+1)
	start := 0
	for start < len(r) {
		end := start + maxRunes
		if end > len(r) {
			end = len(r)
		}
		if end < len(r) {
			if cut := findBreak(r, start, end); cut > start+maxRunes/3 {
				end = cut
			}
		}
		part := strings.TrimSpace(string(r[start:end]))
		if part != "" && (len(out) == 0 || out[len(out)-1] != part) {
			out = append(out, part)
		}
		if end >= len(r) {
			break
		}
		next := end - overlapRunes
		if next <= start {
			next = start + 1
		}
		start = next
	}
	return out
}

// findBreak 在 [start, end) 的后半段寻找句末标点或换行，返回切点（不含标点后的位置）。
func findBreak(r []rune, start, end int) int {
	floor := start + (end-start)/2
	for i := end - 1; i >= floor; i-- {
		switch r[i] {
		case '。', '！', '？', '!', '?', '；', ';', '\n':
			return i + 1
		}
	}
	return end
}

func runeLen(s string) int { return len([]rune(s)) }

func truncateRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

// DeriveTitle 取首个 Markdown 标题或首个非空行；都没有时用 fallback。
func DeriveTitle(text, fallback string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if level, title, ok := parseHeading(line); ok && level >= 1 {
			return truncateRunes(title, 255)
		}
		return truncateRunes(strings.TrimSpace(strings.TrimLeft(line, "# ")), 255)
	}
	return truncateRunes(fallback, 255)
}
