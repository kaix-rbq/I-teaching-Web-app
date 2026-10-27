// Package agent 封装 AI 评析（S1 单次课评价）能力：提示词组装、LLM 调用与结果校验。
//
// 与 ASR 链路同构的硬约束（RAG 方案 §1）：
//  1. 全链路与督导主流程解耦——任何失败只落 ai_tasks.status='failed'，不影响页面；
//  2. 不做同步调用——只在后台 worker 执行；
//  3. 版本化落库——formula_version / prompt_version / ai_model_version 三者缺一不可；
//  4. 证据可追溯——每个非空维度必须给出能在脱敏转写中定位的引用。
package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// ErrUnavailable 表示 AI 引擎未配置（无 Key / 未启用），任务应记为 failed 而非崩溃。
var ErrUnavailable = errors.New("agent: AI 评析引擎未配置")

// Segment 是脱敏转写的一段（秒级时间戳），与 transcripts.segments 对齐。
type Segment struct {
	Start float64
	End   float64
	Text  string
}

// EvalInput 是 S1 单次课评价的输入（不含督导评分，保证 AI 独立评价）。
type EvalInput struct {
	SessionID   uint64
	CourseName  string
	TeacherName string
	SessionDate string
	Period      string
	Topic       string
	Transcript  string
	Segments    []Segment
	// Knowledge 是本次检索命中的知识库片段；为空表示无 KB 降级运行。
	Knowledge []RetrievedChunk
}

// Quote 是一段可定位的转写引用（start/end 单位秒）。
type Quote struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Quote string  `json:"quote"`
}

// DimensionEvidence 是单个维度的置信度与引用。
type DimensionEvidence struct {
	Confidence float64 `json:"confidence"`
	Quotes     []Quote `json:"quotes"`
}

// EvidenceJSON 对应 evaluations.evidence（schemaVersion 1，RAG §4.3）。
// RiskTags 为高风险标签，仅督导可见（D8），教师侧由 service 层脱敏。
type EvidenceJSON struct {
	SchemaVersion int                          `json:"schemaVersion"`
	CitedChunks   []string                     `json:"citedChunks"`
	Dimensions    map[string]DimensionEvidence `json:"dimensions"`
	NotObservable []string                     `json:"notObservable"`
	PromptVersion string                       `json:"promptVersion"`
	RiskTags      []string                     `json:"riskTags,omitempty"`
}

// Result 是 S1 的结构化产出，映射到 evaluations（agent 行）。
type Result struct {
	Objective    *int
	Content      *int
	Interaction  *int
	Organization *int
	Frontier     *int
	Confidence   float64
	Comment      string
	Highlights   string
	Improvements string
	Suggestions  string
	Evidence     EvidenceJSON
	ModelVersion string
}

// s1Response 是 LLM 返回的原始 JSON 结构（RAG §6.2）。
type s1Response struct {
	Dimensions map[string]struct {
		Score      *int    `json:"score"`
		Confidence float64 `json:"confidence"`
		Quotes     []Quote `json:"quotes"`
	} `json:"dimensions"`
	Confidence    float64  `json:"confidence"`
	Comment       string   `json:"comment"`
	Highlights    string   `json:"highlights"`
	Improvements  string   `json:"improvements"`
	Suggestions   string   `json:"suggestions"`
	NotObservable []string `json:"notObservable"`
	CitedChunks   []string `json:"citedChunks"`
	RiskTags      []string `json:"riskTags"`
}

// dimensionOrder 固定维度顺序，保证 evidence.dimensions 稳定可测。
var dimensionOrder = []string{"objective", "content", "interaction", "organization", "frontier"}

// ParseS1Response 解析 LLM 的 S1 输出并做服务端强制校验（RAG §6.2）：
//   - 剥离 ```json 代码块、截取首个 { 到末个 }，容忍模型多话；
//   - score ∈ {1..5, null}，越界视为无法观测（置 NULL 并计入 notObservable）；
//   - 每个非空维度至少 1 条能在脱敏转写中定位的引用，定位失败即丢弃该引用；
//     若清空后无引用，则该维置 NULL + notObservable（不许无证据给分）。
//
// promptVersion 与 modelVersion 由调用方注入，不属于模型输出。
func ParseS1Response(raw, transcript, promptVersion, modelVersion string) (*Result, error) {
	obj, err := extractJSONObject(raw)
	if err != nil {
		return nil, err
	}
	var resp s1Response
	if err := json.Unmarshal(obj, &resp); err != nil {
		return nil, fmt.Errorf("agent: 解析 S1 输出失败: %w", err)
	}

	notObservable := map[string]bool{}
	for _, k := range resp.NotObservable {
		notObservable[k] = true
	}

	evidence := EvidenceJSON{
		SchemaVersion: 1,
		CitedChunks:   nonNilStrings(resp.CitedChunks),
		Dimensions:    map[string]DimensionEvidence{},
		PromptVersion: promptVersion,
		RiskTags:      nonNilStrings(resp.RiskTags),
	}

	result := &Result{
		Confidence:   clamp01(resp.Confidence),
		Comment:      strings.TrimSpace(resp.Comment),
		Highlights:   strings.TrimSpace(resp.Highlights),
		Improvements: strings.TrimSpace(resp.Improvements),
		Suggestions:  strings.TrimSpace(resp.Suggestions),
		ModelVersion: modelVersion,
	}

	for _, key := range dimensionOrder {
		d := resp.Dimensions[key]
		quotes := locateQuotes(d.Quotes, transcript)
		evidence.Dimensions[key] = DimensionEvidence{Confidence: clamp01(d.Confidence), Quotes: quotes}

		score := d.Score
		if score != nil && (*score < 1 || *score > 5) {
			score = nil
		}
		// 无证据不许给分：无法定位引用的维度一律降级为「无法观测」。
		if score != nil && len(quotes) == 0 {
			score = nil
		}
		if score == nil {
			notObservable[key] = true
		}
		assignScore(result, key, score)
	}

	result.Evidence = evidence
	result.Evidence.NotObservable = orderedNotObservable(notObservable)
	return result, nil
}

// assignScore 把维度分写入对应字段。
func assignScore(r *Result, key string, score *int) {
	switch key {
	case "objective":
		r.Objective = score
	case "content":
		r.Content = score
	case "interaction":
		r.Interaction = score
	case "organization":
		r.Organization = score
	case "frontier":
		r.Frontier = score
	}
}

// locateQuotes 只保留能在脱敏转写中定位到子串的引用（宁缺勿滥）。
// 定位失败（模型编造或脱敏改写）的引用一律丢弃。
//
// 定位使用归一化后的文本：删除空白与标点、全角 ASCII 转半角、ASCII 转小写。
// 归一化只服务匹配，返回给前端的 Quote 仍是模型原文（TrimSpace 后）。
// 归一化后为空的引用直接丢弃，避免「只含标点」被误判为命中。
func locateQuotes(quotes []Quote, transcript string) []Quote {
	normalizedTranscript := normalizeQuoteText(transcript)
	out := make([]Quote, 0, len(quotes))
	for _, q := range quotes {
		text := strings.TrimSpace(q.Quote)
		if text == "" {
			continue
		}
		normalizedQuote := normalizeQuoteText(text)
		if normalizedQuote == "" {
			continue
		}
		// 空转写保持历史行为：无法定位时不做子串校验；非空转写必须命中。
		if transcript != "" && !strings.Contains(normalizedTranscript, normalizedQuote) {
			continue
		}
		out = append(out, Quote{Start: q.Start, End: q.End, Quote: text})
	}
	return out
}

// normalizeQuoteText 生成用于引文定位的归一化文本：
//   - 删除 unicode.IsSpace（空白）与 unicode.IsPunct（中英文/全半角标点）；
//   - U+FF01–U+FF5E 全角 ASCII 转半角；
//   - ASCII 大写转小写；
//   - 汉字、数字及其他文字保持原样。
//
// 不做模糊匹配（如编辑距离/短片段包含），编造引用仍必须定位失败。
func normalizeQuoteText(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r >= 0xFF01 && r <= 0xFF5E {
			r -= 0xFEE0
		}
		if unicode.IsSpace(r) || unicode.IsPunct(r) {
			continue
		}
		if r >= 'A' && r <= 'Z' {
			r += 'a' - 'A'
		}
		b.WriteRune(r)
	}
	return b.String()
}

// orderedNotObservable 去重并按固定维度顺序输出，保证结果稳定（含无对应维度的键则丢弃）。
func orderedNotObservable(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for _, key := range dimensionOrder {
		if set[key] {
			out = append(out, key)
		}
	}
	return out
}

// extractJSONObject 剥离代码块围栏并截取首个 { 到末个 }，容忍模型输出多余文本。
func extractJSONObject(raw string) ([]byte, error) {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start < 0 || end < start {
		return nil, errors.New("agent: 输出中未找到 JSON 对象")
	}
	return []byte(s[start : end+1]), nil
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func nonNilStrings(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}

// filterCitedChunks 只保留本次实际注入的知识库 chunk id，去重并保持原顺序。
// 没有知识片段时一律返回空切片，防止模型在无 KB 运行时编造引用。
func filterCitedChunks(cited []string, knowledge []RetrievedChunk) []string {
	allowed := make(map[string]struct{}, len(knowledge))
	for _, k := range knowledge {
		if id := strings.TrimSpace(k.ChunkID); id != "" {
			allowed[id] = struct{}{}
		}
	}
	out := make([]string, 0, len(cited))
	seen := make(map[string]struct{}, len(cited))
	for _, raw := range cited {
		id := strings.TrimSpace(raw)
		if _, ok := allowed[id]; !ok {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
