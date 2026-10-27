package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseReportResponseValid(t *testing.T) {
	raw := `{
	  "summary": "本学期共 5 次课，整体呈上升趋势。",
	  "directions": ["互动维度仍偏弱", "内容深度可加强"],
	  "suggestions": ["提问后保持 3-5 秒沉默", "补充行业案例"],
	  "highlights": "课堂组织稳定",
	  "trendComment": "综合分由 58.75 提升至 78.75",
	  "confidence": 0.7,
	  "citedChunks": ["kb-strategy#7"]
	}`
	res, err := ParseReportResponse(raw)
	require.NoError(t, err)
	assert.Equal(t, "本学期共 5 次课，整体呈上升趋势。", res.Summary)
	assert.Equal(t, []string{"互动维度仍偏弱", "内容深度可加强"}, res.Directions)
	assert.Equal(t, []string{"提问后保持 3-5 秒沉默", "补充行业案例"}, res.Suggestions)
	assert.Equal(t, "课堂组织稳定", res.Highlights)
	assert.Equal(t, "综合分由 58.75 提升至 78.75", res.TrendComment)
	assert.InDelta(t, 0.7, res.Confidence, 1e-9)
	assert.Equal(t, []string{"kb-strategy#7"}, res.CitedChunks)
}

func TestParseReportResponseAllowsEmptyArrays(t *testing.T) {
	raw := `{"summary":"样本不足，暂不给出确定性结论。","directions":[],"suggestions":[],"confidence":0.4}`
	res, err := ParseReportResponse(raw)
	require.NoError(t, err)
	assert.NotNil(t, res.Directions)
	assert.Empty(t, res.Directions)
	assert.NotNil(t, res.Suggestions)
	assert.Empty(t, res.Suggestions)
}

func TestParseReportResponseValidation(t *testing.T) {
	cases := []struct {
		name string
		raw  string
	}{
		{name: "非法 JSON", raw: `{不是 JSON`},
		{name: "空 summary", raw: `{"summary":"  ","directions":[],"suggestions":[]}`},
		{name: "缺失 directions", raw: `{"summary":"ok","suggestions":[]}`},
		{name: "缺失 suggestions", raw: `{"summary":"ok","directions":[]}`},
		{name: "directions 不是数组", raw: `{"summary":"ok","directions":"x","suggestions":[]}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseReportResponse(tc.raw)
			require.Error(t, err)
		})
	}
}

func TestParseReportResponseClampsConfidence(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want float64
	}{
		{name: "大于 1", raw: `{"summary":"ok","directions":[],"suggestions":[],"confidence":2.5}`, want: 1},
		{name: "小于 0", raw: `{"summary":"ok","directions":[],"suggestions":[],"confidence":-1}`, want: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := ParseReportResponse(tc.raw)
			require.NoError(t, err)
			assert.Equal(t, tc.want, res.Confidence)
		})
	}
}

func TestParseReportResponseUnwrapsFence(t *testing.T) {
	raw := "```json\n{\"summary\":\"ok\",\"directions\":[],\"suggestions\":[]}\n```"
	res, err := ParseReportResponse(raw)
	require.NoError(t, err)
	assert.Equal(t, "ok", res.Summary)
}

// TestAnalyzeReportUsesStructuredPrompt 验证 S2 客户端复用同一 Client，
// 请求里只有结构化摘要，不包含转写全文。
func TestAnalyzeReportUsesStructuredPrompt(t *testing.T) {
	response := chatCompletionBody(t, `{"summary":"结构化报告","directions":[],"suggestions":[],"confidence":0.6}`)
	var got chatRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("解析 S2 请求失败: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, response)
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	res, err := client.AnalyzeReport(context.Background(), ReportInput{
		TeacherID: 2, TeacherName: "李明", Semester: "2026-2027-1",
		EvaluatedCount: 1, SessionCount: 1, MinSampleSize: 3,
		Sessions: []ReportSession{{SessionID: 1, CourseName: "软件项目管理", SessionDate: "2026-09-12"}},
	})
	require.NoError(t, err)
	assert.Equal(t, "结构化报告", res.Summary)
	require.Len(t, got.Messages, 2)
	assert.Contains(t, got.Messages[1].Content, "SessionID")
	assert.NotContains(t, got.Messages[1].Content, "Transcript")
	assert.Equal(t, s2MaxTokens, got.MaxTokens)
}
