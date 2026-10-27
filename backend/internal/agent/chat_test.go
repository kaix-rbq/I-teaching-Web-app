package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamChatMultiFrames(t *testing.T) {
	var got chatStreamRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("解析 S3 请求失败: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"你\"}}]}\n\n")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"好\"}}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	var chunks []string
	err := newTestClient(t, srv.URL).StreamChat(context.Background(), []ChatMessage{
		{Role: "system", Content: "system"},
		{Role: "user", Content: "history"},
		{Role: "agent", Content: "previous answer"},
	}, func(delta string) error {
		chunks = append(chunks, delta)
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"你", "好"}, chunks)
	assert.True(t, got.Stream)
	require.Len(t, got.Messages, 3)
	assert.Equal(t, "assistant", got.Messages[2].Role, "agent 角色必须映射为 assistant")
}

func TestStreamChatUpstreamHTTPFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "upstream down", http.StatusInternalServerError)
	}))
	defer srv.Close()

	err := newTestClient(t, srv.URL).StreamChat(context.Background(), []ChatMessage{{Role: "user", Content: "hi"}},
		func(string) error { return nil })
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestStreamChatErrorFrame(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"error\":{\"message\":\"model unavailable\",\"code\":\"500\"}}\n\n")
	}))
	defer srv.Close()

	err := newTestClient(t, srv.URL).StreamChat(context.Background(), []ChatMessage{{Role: "user", Content: "hi"}},
		func(string) error { return nil })
	require.Error(t, err)
	assert.Contains(t, err.Error(), "model unavailable")
}

func TestStreamChatInvalidFrame(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {not-json}\n\n")
	}))
	defer srv.Close()

	err := newTestClient(t, srv.URL).StreamChat(context.Background(), []ChatMessage{{Role: "user", Content: "hi"}},
		func(string) error { return nil })
	require.Error(t, err)
	assert.Contains(t, err.Error(), "解析 S3 流帧失败")
}

func TestStreamChatStopsWhenOnDeltaFails(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"a\"}}]}\n\n")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"b\"}}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	sentinel := errors.New("client disconnected")
	err := newTestClient(t, srv.URL).StreamChat(context.Background(), []ChatMessage{{Role: "user", Content: "hi"}},
		func(string) error {
			calls++
			return sentinel
		})
	require.ErrorIs(t, err, sentinel)
	assert.Equal(t, 1, calls, "onDelta 失败后必须停止读取")
}
