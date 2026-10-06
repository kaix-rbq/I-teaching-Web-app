package service

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"aijiaoxue-api/internal/asr"
	"aijiaoxue-api/internal/dto"
)

// Scrubber 是转写文本脱敏器。
//
// 合规依据：开发计划 §5.2-5「转写文本中的学生姓名做 NER 替换（"学生A"），
// 避免评语出现可识别个人的内容」。任何情况下都不得把带真实姓名的转写文本写入 transcripts.content。
//
// 当前实现为「称谓正则 + 可配置词典」；学生名册尚未建立，因此不依赖名单也能工作。
// 后续接入 NER 模型或名单时，只需替换 Scrubber 的实现，服务层调用点不变。
type Scrubber interface {
	// Scrub 同时脱敏全文与句级分段，保证两者一致（前端气泡与全文不得出现分歧）。
	Scrub(content string, segs []asr.Segment) (string, []dto.TranscriptSegment)
}

// honorificRe 匹配「<2-4 个汉字>同学/同学们/小朋友」。
//
// 贪心匹配 + 回溯保证「张伟同学们」优先识别姓名为「张伟」而非「张伟同」。
// 局限：音近字误识别、未以称谓出现的姓名（如直呼全名）无法覆盖，故补充词典通道。
var honorificRe = regexp.MustCompile(`([\p{Han}]{2,4})(同学们|同学|小朋友们|小朋友)`)

// honorificStops 是否定列表：这些词出现在称谓前时不是姓名，必须原样保留。
// 缺了它，「我们同学们」「各位同学」会被误判成姓名为「我们」「各位」。
var honorificStops = map[string]struct{}{
	"我们": {}, "你们": {}, "他们": {}, "咱们": {}, "大家": {},
	"各位": {}, "全班": {}, "每个": {}, "所有": {}, "有的": {},
	"有些": {}, "这些": {}, "那些": {}, "这个": {}, "那个": {},
	"两个": {}, "三个": {}, "几个": {}, "很多": {}, "同学": {},
	"学生": {}, "老师": {}, "家长": {},
}

// NewScrubber 依据可选的姓名词典构造脱敏器（词典为空也安全，退化为纯称谓正则）。
func NewScrubber(dict []string) Scrubber {
	cleaned := make([]string, 0, len(dict))
	for _, name := range dict {
		if name = strings.TrimSpace(name); name != "" {
			cleaned = append(cleaned, name)
		}
	}
	// 长名优先，避免「张三」先替换导致「张三丰」被割裂。
	sort.Slice(cleaned, func(i, j int) bool { return len(cleaned[i]) > len(cleaned[j]) })
	return &regexScrubber{dict: cleaned}
}

type regexScrubber struct {
	dict []string
}

// nameMasker 保证同一真实姓名始终映射到同一代号（保持话轮可读性）。
type nameMasker struct {
	assigned map[string]string
	seq      int
}

func newNameMasker() *nameMasker {
	return &nameMasker{assigned: make(map[string]string)}
}

// code 返回该姓名对应的匿名代号，必要时分配新代号。
//
// 🔴 必须把新代号写回 assigned：漏写会让同一个姓名每次都被当成新名字，
// 结果「李娜同学…李娜同学」被替换成两个不同代号（已由 TestNameMaskerCode 回归覆盖）。
func (m *nameMasker) code(name string) string {
	if c, ok := m.assigned[name]; ok {
		return c
	}
	m.seq++
	// 1→学生A … 26→学生Z；超出后用数字兜底，避免出现奇怪字符。
	code := fmt.Sprintf("学生%d", m.seq)
	if m.seq <= 26 {
		code = fmt.Sprintf("学生%c", 'A'+rune(m.seq-1))
	}
	m.assigned[name] = code
	return code
}

// nameJoins 是可能紧贴在姓名前的中文连接词/动词/助词。
//
// Go 的 regexp 是 RE2，不支持后行断言，无法在正则里排除它们；
// 贪心匹配会把「和李娜」整体当成姓名。因此对捕获到的 2-4 个汉字再做一次切分，
// 取最后一个连接词之后的部分为姓名，前缀原样保留（见 splitName）。
var nameJoins = map[rune]struct{}{
	'和': {}, '与': {}, '跟': {}, '及': {}, '或': {}, '请': {}, '让': {},
	'给': {}, '对': {}, '向': {}, '问': {}, '叫': {}, '是': {}, '的': {},
	'了': {}, '在': {}, '又': {}, '再': {}, '就': {}, '才': {}, '都': {},
	'也': {}, '那': {}, '这': {}, '有': {}, '看': {}, '听': {}, '说': {},
}

// splitName 把捕获串拆成「前缀 + 姓名」，取最后一个连接词之后的部分为姓名。
//
//	「和李娜」   → ("和", "李娜")
//	「娜和张伟」 → ("娜和", "张伟")   ← 前一个姓名无法识别，原样保留
//	「王小明」   → ("", "王小明")
func splitName(raw string) (prefix, name string) {
	runes := []rune(raw)
	cut := 0
	for i, r := range runes {
		if _, ok := nameJoins[r]; ok {
			cut = i + 1
		}
	}
	if cut >= len(runes) {
		return raw, ""
	}
	return string(runes[:cut]), string(runes[cut:])
}

// mask 对单段文本做脱敏。
func (m *nameMasker) mask(text string, dict []string) string {
	if text == "" {
		return text
	}
	// 通道一：显式词典（可覆盖直呼全名等称谓正则抓不到的情况）。
	for _, name := range dict {
		if strings.Contains(text, name) {
			text = strings.ReplaceAll(text, name, m.code(name))
		}
	}
	// 通道二：称谓正则。
	return honorificRe.ReplaceAllStringFunc(text, func(match string) string {
		sub := honorificRe.FindStringSubmatch(match)
		if len(sub) < 3 {
			return match
		}
		prefix, name := splitName(sub[1])
		// 姓名至少 2 个汉字，否则视为切分失败，退回整串判断。
		if len([]rune(name)) < 2 {
			prefix, name = "", sub[1]
		}
		if _, isStop := honorificStops[name]; isStop {
			return match
		}
		return prefix + m.code(name) + sub[2]
	})
}

// Scrub 实现 Scrubber：全文与分段共用同一个 nameMasker，确保代号一致。
func (s *regexScrubber) Scrub(content string, segs []asr.Segment) (string, []dto.TranscriptSegment) {
	masker := newNameMasker()

	out := make([]dto.TranscriptSegment, 0, len(segs))
	for _, seg := range segs {
		out = append(out, dto.TranscriptSegment{
			Start:   seg.Start,
			End:     seg.End,
			Speaker: seg.Speaker,
			Text:    masker.mask(seg.Text, s.dict),
		})
	}
	return masker.mask(content, s.dict), out
}
