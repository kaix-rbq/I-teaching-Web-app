package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"aijiaoxue-api/internal/agent"
	"aijiaoxue-api/internal/config"
	"aijiaoxue-api/internal/dto"
	"aijiaoxue-api/internal/repository"
	"aijiaoxue-api/pkg/errcode"
)

type fakeChatStreamer struct {
	messages []agent.ChatMessage
	deltas   []string
	err      error
}

func (f *fakeChatStreamer) StreamChat(_ context.Context, messages []agent.ChatMessage, onDelta func(string) error) error {
	f.messages = append([]agent.ChatMessage(nil), messages...)
	for _, delta := range f.deltas {
		if err := onDelta(delta); err != nil {
			return err
		}
	}
	return f.err
}

func chatCourse() *repository.CourseRow {
	return &repository.CourseRow{
		ID: 1, Code: "SE3101", Name: "软件项目管理", TeacherID: 2, TeacherName: "李明",
		DepartmentID: 1, Semester: "2026-2027-1",
	}
}

func chatEvalRows() []repository.SessionDimRow {
	f := func(v float64) *float64 { return &v }
	return []repository.SessionDimRow{
		{
			SessionID: 1, CourseID: 1, TeacherID: 2,
			SessionDate: time.Date(2026, 9, 12, 0, 0, 0, 0, time.UTC),
			HasAgent:    true, AiObjective: f(4), AiContent: f(3), AiInteraction: f(2),
			AiOrganization: f(4), AiFrontier: f(3),
		},
		{
			SessionID: 2, CourseID: 1, TeacherID: 2,
			SessionDate: time.Date(2026, 9, 19, 0, 0, 0, 0, time.UTC),
			HasAgent:    true, AiObjective: f(5), AiContent: f(4), AiInteraction: f(3),
			AiOrganization: f(4), AiFrontier: f(3),
		},
	}
}

func chatSuggestions() []repository.AgentSuggestionRow {
	return []repository.AgentSuggestionRow{
		{
			ID: 2, SessionID: 2, CourseID: 1,
			SessionDate: time.Date(2026, 9, 19, 0, 0, 0, 0, time.UTC),
			Topic:       "需求调研", Comment: "讨论充分", Highlights: "小组产出明确",
			Improvements: "个别小组偏题", EvaluatorType: "agent",
		},
		{
			ID: 1, SessionID: 1, CourseID: 1,
			SessionDate: time.Date(2026, 9, 12, 0, 0, 0, 0, time.UTC),
			Topic:       "项目立项", Comment: "框架完整", Highlights: "目标清晰",
			Improvements: "等待时间偏短", EvaluatorType: "agent",
		},
	}
}

func newChatService(
	t *testing.T, streamer agent.ChatStreamer, retriever agent.Retriever,
) AgentChatService {
	t.Helper()
	courses := &fakeCourseRepo{getDetail: func(context.Context, uint64) (*repository.CourseRow, error) {
		return chatCourse(), nil
	}}
	evals := &fakeEvalRepo{
		listSessionDimRows: func(context.Context, repository.SessionDimFilter) ([]repository.SessionDimRow, error) {
			return chatEvalRows(), nil
		},
		listAgentSuggestions: func(context.Context, repository.AgentSuggestionFilter) ([]repository.AgentSuggestionRow, error) {
			return chatSuggestions(), nil
		},
	}
	return NewAgentChatService(courses, evals, retriever, streamer,
		config.AgentConfig{MaxConcurrency: 1}, evalCfg())
}

func TestAgentChatHappyPathAndHistoryTruncation(t *testing.T) {
	streamer := &fakeChatStreamer{deltas: []string{"你", "好"}}
	retriever := &retrieverWithHits{hits: []agent.RetrievedChunk{
		{ChunkID: "kb-rubric#3", Collection: "kb-rubric", Heading: "互动锚点", Content: "提问后留思考时间。"},
		{ChunkID: "kb-strategy#7", Collection: "kb-strategy", Heading: "Formative Feedback", Content: "给出具体反馈。"},
	}}
	svc := newChatService(t, streamer, retriever)

	history := make([]dto.AgentChatHistoryItem, 0, 13)
	for i := 1; i <= 13; i++ {
		role := "user"
		if i%2 == 0 {
			role = "agent"
		}
		history = append(history, dto.AgentChatHistoryItem{
			Role: role, Content: fmt.Sprintf("h%d", i),
		})
	}
	var chunks []string
	citations, err := svc.Chat(context.Background(), RoleTeacher, 0, 2, dto.AgentChatReq{
		CourseID: 1, Question: "如何提升课堂互动？", History: history,
	}, func(delta string) error {
		chunks = append(chunks, delta)
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"你", "好"}, chunks)
	require.Len(t, citations, 2)
	assert.Equal(t, "kb-rubric#3", citations[0].ChunkID)
	assert.Equal(t, "互动锚点", citations[0].Title)
	assert.NotEmpty(t, citations[0].Quote)

	require.Len(t, streamer.messages, 1+agentChatHistoryLimit+1)
	assert.Equal(t, "system", streamer.messages[0].Role)
	assert.Contains(t, streamer.messages[0].Content, "软件项目管理")
	assert.Contains(t, streamer.messages[0].Content, "kb-rubric#3")
	assert.NotContains(t, streamer.messages[0].Content, "Transcript")
	assert.NotContains(t, streamer.messages[0].Content, "脱敏转写文本")
	// 超 6 轮（12 条）只保留最近 12 条：h1 被截掉，h2 成为第一条历史。
	assert.Equal(t, "h2", streamer.messages[1].Content)
	assert.Equal(t, "user", streamer.messages[len(streamer.messages)-1].Role)
	assert.Equal(t, "如何提升课堂互动？", streamer.messages[len(streamer.messages)-1].Content)
}

func TestAgentChatValidation(t *testing.T) {
	svc := newChatService(t, &fakeChatStreamer{}, nil)
	cases := []struct {
		name string
		req  dto.AgentChatReq
	}{
		{name: "courseId 为 0", req: dto.AgentChatReq{CourseID: 0, Question: "q"}},
		{name: "question 为空", req: dto.AgentChatReq{CourseID: 1, Question: "  "}},
		{
			name: "history role 非法",
			req: dto.AgentChatReq{CourseID: 1, Question: "q", History: []dto.AgentChatHistoryItem{
				{Role: "system", Content: "x"},
			}},
		},
		{
			name: "history 最后一条不是 user",
			req: dto.AgentChatReq{CourseID: 1, Question: "q", History: []dto.AgentChatHistoryItem{
				{Role: "user", Content: "x"}, {Role: "agent", Content: "y"},
			}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Chat(context.Background(), RoleTeacher, 0, 2, tc.req, func(string) error { return nil })
			require.Error(t, err)
			assert.Equal(t, errcode.Params, errcode.From(err).Code)
		})
	}
}

func TestAgentChatTeacherOtherCourseForbidden(t *testing.T) {
	svc := newChatService(t, &fakeChatStreamer{}, nil)
	_, err := svc.Chat(context.Background(), RoleTeacher, 0, 3,
		dto.AgentChatReq{CourseID: 1, Question: "q"}, func(string) error { return nil })
	require.Error(t, err)
	assert.Equal(t, errcode.ForbiddenData, errcode.From(err).Code)
}

func TestAgentChatKBFailureStillStreams(t *testing.T) {
	streamer := &fakeChatStreamer{deltas: []string{"继续"}}
	retriever := &fakeRetrieverForProcess{err: errors.New("kb unavailable")}
	svc := newChatService(t, streamer, retriever)

	var chunks []string
	citations, err := svc.Chat(context.Background(), RoleSupervisor, 0, 9,
		dto.AgentChatReq{CourseID: 1, Question: "q"}, func(delta string) error {
			chunks = append(chunks, delta)
			return nil
		})
	require.NoError(t, err)
	assert.Equal(t, []string{"继续"}, chunks)
	assert.Empty(t, citations)
	assert.Contains(t, streamer.messages[0].Content, "软件项目管理")
}

func TestAgentChatUpstreamFailureMaps50003(t *testing.T) {
	streamer := &fakeChatStreamer{err: errors.New("upstream failed")}
	svc := newChatService(t, streamer, nil)
	_, err := svc.Chat(context.Background(), RoleSupervisor, 0, 9,
		dto.AgentChatReq{CourseID: 1, Question: "q"}, func(string) error { return nil })
	require.Error(t, err)
	appErr := errcode.From(err)
	assert.Equal(t, errcode.DependencyUnavailable, appErr.Code)
	assert.Equal(t, "智能体暂时不可用", appErr.Msg)
}

func TestAgentChatOnDeltaErrorStops(t *testing.T) {
	streamer := &fakeChatStreamer{deltas: []string{"a", "b"}}
	svc := newChatService(t, streamer, nil)
	sentinel := errors.New("client disconnected")
	_, err := svc.Chat(context.Background(), RoleSupervisor, 0, 9,
		dto.AgentChatReq{CourseID: 1, Question: "q"}, func(string) error {
			return sentinel
		})
	require.ErrorIs(t, err, sentinel)
	assert.NotEqual(t, errcode.DependencyUnavailable, errcode.From(err).Code)
}

func TestAgentChatHistoryKeepsRoles(t *testing.T) {
	streamer := &fakeChatStreamer{}
	svc := newChatService(t, streamer, nil)
	history := []dto.AgentChatHistoryItem{{Role: "user", Content: "之前的问题"}, {Role: "agent", Content: "之前的回答"}, {Role: "user", Content: "继续"}}
	_, err := svc.Chat(context.Background(), RoleSupervisor, 0, 9,
		dto.AgentChatReq{CourseID: 1, Question: "现在的问题", History: history}, func(string) error { return nil })
	require.NoError(t, err)
	require.Len(t, streamer.messages, 1+3+1)
	assert.Equal(t, "user", streamer.messages[1].Role)
	assert.Equal(t, "agent", streamer.messages[2].Role)
	assert.Equal(t, "user", streamer.messages[3].Role)
}
