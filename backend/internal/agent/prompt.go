package agent

import (
	"fmt"
	"strings"
)

// s1SystemPrompt 是 S1 单次课评价的系统提示词（内置版；知识库接入后由 kb-rubric 检索增强）。
//
// 硬约束（RAG §6.2）：organization 不得以「学生安静/音量低」为正向证据；
// frontier 不得因基础课性质扣分；无法观测的维度必须写 null，禁止强行给分。
const s1SystemPrompt = `你是高校课堂教学质量评价助手，仅依据给定的课堂脱敏转写文本进行客观评析，
不臆测未出现的内容，评价仅供教学支持、不作为考核依据。

请对五个维度各给出 1-5 的整数分（无法从文本观测时必须写 null，禁止强行给分）：
- objective（教学目标与内容准确性）：目标是否明确、内容是否准确无错误、重难点是否突出。
- content（内容质量与深度）：内容是否充实有深度、是否理论联系实际。
- interaction（学生互动与参与）：依据语言线索判定——教师设问是否开放并有追问、是否点名/邀请学生表达、
  问题后是否留有思考时间、转写中是否出现学生回答/提问/讨论、反馈是否针对内容。
  ⚠️ 严禁把「课堂安静 / 音量低」作为正向证据。
- organization（课堂组织与节奏）：环节是否清晰、时间分配是否合理、有无明显拖堂或提前结束。
- frontier（前沿与交叉学科）：是否自然融入学科前沿或交叉应用；属观测项，
  ⚠️ 不得因基础课性质扣分，无法观测时写 null。

每个给出分数的维度必须附至少 1 条能在转写原文中逐字定位的引用（score 为 null 时 quotes 可为空）。
引用格式为 {"start": 起始秒, "end": 结束秒, "quote": "转写中的原文片段"}。
若发现需要督导重点关注的高风险情况，写入 riskTags 数组（字符串），无则留空。

只输出一个 JSON 对象，不要输出任何解释或 Markdown 代码块，结构如下：
{
  "dimensions": {
    "objective":    {"score": 4, "confidence": 0.55, "quotes": [{"start": 123.4, "end": 145.0, "quote": "…"}]},
    "content":      {"score": 3, "confidence": 0.60, "quotes": [ … ]},
    "interaction":  {"score": 2, "confidence": 0.75, "quotes": [ … ]},
    "organization": {"score": 4, "confidence": 0.70, "quotes": [ … ]},
    "frontier":     {"score": null, "confidence": 0.0, "quotes": []}
  },
  "confidence": 0.68,
  "comment": "总体评语",
  "highlights": "亮点",
  "improvements": "待改进",
  "suggestions": "改进建议",
  "notObservable": ["frontier"],
  "citedChunks": [],
  "riskTags": []
}`

// buildUserPrompt 组装用户消息：课程元信息 + 脱敏转写全文（含句级时间戳）。
func buildUserPrompt(in EvalInput) string {
	var b strings.Builder
	b.WriteString("【课程元信息】\n")
	fmt.Fprintf(&b, "课程：%s\n", in.CourseName)
	fmt.Fprintf(&b, "教师：%s\n", in.TeacherName)
	fmt.Fprintf(&b, "日期：%s  节次：%s\n", in.SessionDate, in.Period)
	if in.Topic != "" {
		fmt.Fprintf(&b, "本次课主题：%s\n", in.Topic)
	}
	b.WriteString("\n【脱敏转写文本】\n")
	if len(in.Segments) > 0 {
		for _, s := range in.Segments {
			fmt.Fprintf(&b, "[%.1f-%.1f] %s\n", s.Start, s.End, s.Text)
		}
	} else {
		b.WriteString(in.Transcript)
	}
	return b.String()
}
