package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// s2MaxTokens 为教师级报告预留输出空间；S2 输出比 S1 短，但保留足够余量避免截断。
const s2MaxTokens = 3000

// ReportInput 是 S2 教师级报告的输入：只包含结构化评价摘要，
// 绝不包含转写全文、音频或原始学生信息。
type ReportInput struct {
	TeacherID          uint64
	TeacherName        string
	Semester           string
	EvaluatedCount     int
	SessionCount       int
	MinSampleSize      int
	Flags              []string
	Sessions           []ReportSession
	Trend              []ReportTrend
	SupervisorComments []string
}

// ReportSession 是单场次的结构化评价摘要（来自 evaluations 的 agent 行）。
type ReportSession struct {
	SessionID      uint64
	CourseID       uint64
	CourseCode     string
	CourseName     string
	SessionDate    string
	Period         string
	Topic          string
	Objective      *int
	Content        *int
	Interaction    *int
	Organization   *int
	Frontier       *int
	Comment        string
	Highlights     string
	Improvements   string
	Suggestions    string
	AIModelVersion string
	AIConfidence   *float64
}

// ReportTrendPoint 是趋势序列中的单点；Value 为 nil 表示该课次无 AI 评价。
type ReportTrendPoint struct {
	Label string
	Date  string
	Value *float64
}

// ReportTrend 是趋势序列（composite + 五维），顺序与 P2 score-trend 一致。
type ReportTrend struct {
	Key           string
	Name          string
	IsObservation bool
	Points        []ReportTrendPoint
}

// ReportResult 是 S2 的结构化产出。
type ReportResult struct {
	Summary      string
	Directions   []string
	Suggestions  []string
	Highlights   string
	TrendComment string
	Confidence   float64
	CitedChunks  []string
}

// ReportAnalyzer 是教师级报告能力接口；*Client 实现它，测试用假实现替换。
type ReportAnalyzer interface {
	AnalyzeReport(ctx context.Context, in ReportInput) (*ReportResult, error)
}

// s2SystemPrompt 是教师级报告的系统提示词。输入只允许结构化摘要；
// 明确禁止臆测课堂事实，并要求样本不足时在 summary 中主动声明。
const s2SystemPrompt = `你是高校教师教学质量改进顾问，只依据给定的结构化评价摘要生成教师级综合报告。
不得臆测未提供的课堂事实，不得引用转写原文，不得把督导评语当作 AI 评分依据。
如果 evaluatedCount 小于 minSampleSize，必须在 summary 中明确说明“样本不足”，不得给出确定性结论。
报告用于教学支持与改进，不作为考核依据。

只输出一个 JSON 对象，不要输出任何解释或 Markdown 代码块，结构如下：
{
  "summary": "本学期共 N 次课……（必填）",
  "directions": ["可提优方向 1", "可提优方向 2"],
  "suggestions": ["可执行建议 1", "可执行建议 2"],
  "highlights": "本学期亮点",
  "trendComment": "历次趋势解读",
  "confidence": 0.70,
  "citedChunks": []
}

要求：
- summary / directions / suggestions / highlights / trendComment 必须存在；
- directions 与 suggestions 必须是字符串数组，允许空数组 []，但不得缺字段；
- confidence 为 0-1 的数字；
- 只依据输入中的 sessions / trend / supervisorComments 归纳，不要编造课次或分数。`

// buildReportPrompt 把结构化输入序列化进 user prompt；不包含转写全文。
func buildReportPrompt(in ReportInput) string {
	raw, _ := json.Marshal(in)
	return "【教师本学期结构化评价数据】\n" + string(raw) + "\n\n请严格按系统要求输出 JSON。"
}

// AnalyzeReport 调用 LLM 生成教师级报告。复用 Client 的 HTTP 客户端与 semaphore，
// 不创建第二套连接池；解析失败直接返回错误，由 report service 标记 failed。
func (c *Client) AnalyzeReport(ctx context.Context, in ReportInput) (*ReportResult, error) {
	select {
	case c.sem <- struct{}{}:
		defer func() { <-c.sem }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	body, err := json.Marshal(chatRequest{
		Model: c.cfg.LLMModel,
		Messages: []chatMessage{
			{Role: "system", Content: s2SystemPrompt},
			{Role: "user", Content: buildReportPrompt(in)},
		},
		Temperature:    0.2,
		ResponseFormat: &respFormat{Type: "json_object"},
		MaxTokens:      s2MaxTokens,
	})
	if err != nil {
		return nil, fmt.Errorf("agent: 构造 S2 请求失败: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("agent: 构造 S2 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("agent: 调用 S2 上游失败: %w", err)
	}
	defer resp.Body.Close()
	raw, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if readErr != nil {
		return nil, fmt.Errorf("agent: 读取 S2 上游响应失败: %w", readErr)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("agent: S2 上游返回 %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}

	var cr chatResponse
	if err := json.Unmarshal(raw, &cr); err != nil {
		return nil, fmt.Errorf("agent: 解析 S2 上游响应失败: %w", err)
	}
	if cr.Error != nil {
		return nil, fmt.Errorf("agent: S2 上游错误: %s", cr.Error.Message)
	}
	if len(cr.Choices) == 0 {
		return nil, errors.New("agent: S2 上游返回空 choices")
	}
	return ParseReportResponse(cr.Choices[0].Message.Content)
}

// reportWire 用指针数组区分「字段缺失/null」与「显式空数组」。
type reportWire struct {
	Summary      string    `json:"summary"`
	Directions   *[]string `json:"directions"`
	Suggestions  *[]string `json:"suggestions"`
	Highlights   string    `json:"highlights"`
	TrendComment string    `json:"trendComment"`
	Confidence   float64   `json:"confidence"`
	CitedChunks  []string  `json:"citedChunks"`
}

// ParseReportResponse 解析并严格校验 S2 输出：
// summary 必填；directions/suggestions 必须存在且为 string[]（允许空数组）；
// confidence 收敛到 0-1；非法 JSON 返回明确错误。
func ParseReportResponse(raw string) (*ReportResult, error) {
	obj, err := extractJSONObject(raw)
	if err != nil {
		return nil, fmt.Errorf("agent: 解析 S2 输出失败: %w", err)
	}
	var wire reportWire
	if err := json.Unmarshal(obj, &wire); err != nil {
		return nil, fmt.Errorf("agent: 解析 S2 输出失败: %w", err)
	}
	summary := strings.TrimSpace(wire.Summary)
	if summary == "" {
		return nil, errors.New("agent: S2 summary 不能为空")
	}
	if wire.Directions == nil {
		return nil, errors.New("agent: S2 directions 必须为数组")
	}
	if wire.Suggestions == nil {
		return nil, errors.New("agent: S2 suggestions 必须为数组")
	}
	return &ReportResult{
		Summary:      summary,
		Directions:   nonNilStrings(*wire.Directions),
		Suggestions:  nonNilStrings(*wire.Suggestions),
		Highlights:   strings.TrimSpace(wire.Highlights),
		TrendComment: strings.TrimSpace(wire.TrendComment),
		Confidence:   clamp01(wire.Confidence),
		CitedChunks:  nonNilStrings(wire.CitedChunks),
	}, nil
}
