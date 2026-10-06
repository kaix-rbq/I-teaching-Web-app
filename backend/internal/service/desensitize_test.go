package service

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"aijiaoxue-api/internal/asr"
)

// TestScrubberHonorific 验证称谓正则通道：合规 §5.2-5 要求学生姓名必须被替换为匿名代号。
func TestScrubberHonorific(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"两位姓名替换", "张伟同学回答得很好", "学生A同学回答得很好"},
		{"三位姓名（复姓）", "欧阳娜同学请坐下", "学生A同学请坐下"},
		{"同一姓名映射到同一代号", "李娜同学发言，李娜同学说得对", "学生A同学发言，学生A同学说得对"},
		{"不同姓名分配不同代号", "张伟同学和李娜同学", "学生A同学和学生B同学"},
		// 连接词不得被吞进姓名（RE2 无后行断言，靠 splitName 切分）
		{"连接词-和", "张伟同学和李娜同学都答对了", "学生A同学和学生B同学都答对了"},
		{"连接词-让", "让张伟同学回答这个问题", "让学生A同学回答这个问题"},
		{"连接词-请", "请李娜同学读一下", "请学生A同学读一下"},
		{"连接词-在", "王芳同学在认真听", "学生A同学在认真听"},
		{"复数称谓", "王芳同学们都听懂了", "学生A同学们都听懂了"},
		// 否定列表：这些不是姓名，必须原样保留，否则「我们同学们」会被替换成「学生A同学们」
		{"否定列表-我们", "我们同学们要好好听讲", "我们同学们要好好听讲"},
		{"否定列表-各位", "各位同学下午好", "各位同学下午好"},
		{"否定列表-全班", "全班同学都完成了", "全班同学都完成了"},
		// 老师不在学生脱敏范围内（音频对教师不可见，但文本保留教学语境）
		{"教师称谓不替换", "李老师讲得很清楚", "李老师讲得很清楚"},
		{"无称谓的文本原样保留", "这节课讲了进程调度", "这节课讲了进程调度"},
	}

	scrubber := NewScrubber(nil)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := scrubber.Scrub(tc.in, nil)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestScrubberDict 验证词典通道：长名优先，避免「张三」割裂「张三丰」。
func TestScrubberDict(t *testing.T) {
	scrubber := NewScrubber([]string{"张三", "张三丰"})

	got, _ := scrubber.Scrub("张三丰站起来回答", nil)
	assert.Equal(t, "学生A站起来回答", got, "长名必须优先匹配，否则会得到「学生A丰」")

	// 词典可覆盖不以称谓出现的直呼全名。
	got, _ = scrubber.Scrub("请王小明说说看法", nil)
	assert.Equal(t, "请王小明说说看法", got, "未配置的名字不应被误替换")

	got, _ = NewScrubber([]string{"王小明"}).Scrub("请王小明说说看法", nil)
	assert.Equal(t, "请学生A说说看法", got)
}

// TestNameMaskerCode 直接覆盖代号分配：
//  1. 同一姓名必须复用同一代号（曾是真实 bug：code 未写回缓存，同一人拿到两个代号）；
//  2. 超过 26 个姓名时用数字兜底，不产生非法字符。
func TestNameMaskerCode(t *testing.T) {
	m := newNameMasker()

	assert.Equal(t, "学生A", m.code("甲"))
	assert.Equal(t, "学生B", m.code("乙"), "不同姓名应分配新代号")
	assert.Equal(t, "学生A", m.code("甲"), "🔴 同一姓名必须复用代号（缓存写回回归）")

	// 递增到第 26 个是 Z，第 27 个落到数字。
	for i := 3; i <= 26; i++ {
		_ = m.code(string(rune('a' + i)))
	}
	assert.Equal(t, "学生Z", m.code(string(rune('a'+26))))
	assert.Equal(t, "学生27", m.code("第27个名字"))
}

// TestSplitName 覆盖连接词切分规则。
func TestSplitName(t *testing.T) {
	cases := []struct {
		raw        string
		wantPrefix string
		wantName   string
	}{
		{"李娜", "", "李娜"},
		{"和李娜", "和", "李娜"},
		{"让张伟", "让", "张伟"},
		{"娜和张伟", "娜和", "张伟"},
		{"王小明", "", "王小明"},
		{"我们", "", "我们"},
	}
	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			prefix, name := splitName(tc.raw)
			assert.Equal(t, tc.wantPrefix, prefix)
			assert.Equal(t, tc.wantName, name)
		})
	}
}

// TestScrubberContentSegmentsConsistent 是回归用例：
// 全文与分段必须共用同一个 nameMasker，否则前端气泡与全文会出现代号分歧。
func TestScrubberContentSegmentsConsistent(t *testing.T) {
	segs := []asr.Segment{
		{Start: 0, End: 2.5, Speaker: "说话人1", Text: "张伟同学先说"},
		{Start: 2.5, End: 6, Speaker: "说话人2", Text: "李娜同学补充，张伟同学记录"},
	}
	content := "张伟同学先说\n李娜同学补充，张伟同学记录"

	gotContent, gotSegs := NewScrubber(nil).Scrub(content, segs)

	assert.Equal(t, "学生A同学先说\n学生B同学补充，学生A同学记录", gotContent)
	assert.Equal(t, "学生A同学先说", gotSegs[0].Text)
	assert.Equal(t, "学生B同学补充，学生A同学记录", gotSegs[1].Text)
	// 时间轴与说话人不得被改动。
	assert.Equal(t, 2.5, gotSegs[1].Start)
	assert.Equal(t, "说话人2", gotSegs[1].Speaker)
}

// TestScrubberEmptyInput 边界：空输入与零分段不得 panic。
func TestScrubberEmptyInput(t *testing.T) {
	content, segs := NewScrubber([]string{"张三"}).Scrub("", nil)
	assert.Equal(t, "", content)
	assert.Empty(t, segs)
}
