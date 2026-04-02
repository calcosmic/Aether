package llm

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/ssestream"
)

func TestAccumulateStreamText(t *testing.T) {
	// Build SSE events that simulate Anthropic streaming response
	events := []sseEvent{
		{eventType: "message_start", data: `{"type":"message_start","message":{"id":"msg_stream1","type":"message","role":"assistant","model":"claude-sonnet-4-20250514","content":[],"stop_reason":null,"usage":{"input_tokens":20,"output_tokens":0}}}`},
		{eventType: "content_block_start", data: `{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`},
		{eventType: "content_block_delta", data: `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello "}}`},
		{eventType: "content_block_delta", data: `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"World"}}`},
		{eventType: "content_block_stop", data: `{"type":"content_block_stop","index":0}`},
		{eventType: "message_delta", data: `{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":5}}`},
		{eventType: "message_stop", data: `{"type":"message_stop"}`},
	}

	server := newSSEServer(t, events)
	defer server.Close()

	stream := newTestStream(t, server.URL)
	result, err := AccumulateStream(stream)
	if err != nil {
		t.Fatalf("AccumulateStream() error = %v", err)
	}

	if result.Text != "Hello World" {
		t.Errorf("Text = %q, want %q", result.Text, "Hello World")
	}
	if result.Role != "assistant" {
		t.Errorf("Role = %q, want %q", result.Role, "assistant")
	}
	if result.Model != "claude-sonnet-4-20250514" {
		t.Errorf("Model = %q, want %q", result.Model, "claude-sonnet-4-20250514")
	}
	if result.StopReason != "end_turn" {
		t.Errorf("StopReason = %q, want %q", result.StopReason, "end_turn")
	}
	if result.Usage.InputTokens != 20 {
		t.Errorf("Usage.InputTokens = %d, want 20", result.Usage.InputTokens)
	}
	if result.Usage.OutputTokens != 5 {
		t.Errorf("Usage.OutputTokens = %d, want 5", result.Usage.OutputTokens)
	}
}

func TestAccumulateStreamEmpty(t *testing.T) {
	// Stream with message start/stop but no content blocks
	events := []sseEvent{
		{eventType: "message_start", data: `{"type":"message_start","message":{"id":"msg_empty","type":"message","role":"assistant","model":"claude-sonnet-4-20250514","content":[],"stop_reason":null,"usage":{"input_tokens":10,"output_tokens":0}}}`},
		{eventType: "message_delta", data: `{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":1}}`},
		{eventType: "message_stop", data: `{"type":"message_stop"}`},
	}

	server := newSSEServer(t, events)
	defer server.Close()

	stream := newTestStream(t, server.URL)
	result, err := AccumulateStream(stream)
	if err != nil {
		t.Fatalf("AccumulateStream() error = %v", err)
	}

	if result.Text != "" {
		t.Errorf("Text = %q, want empty", result.Text)
	}
}

func TestAccumulateStreamError(t *testing.T) {
	// Server that returns an error SSE event
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "event: error\ndata: {\"type\":\"error\",\"error\":{\"type\":\"overloaded_error\",\"message\":\"Server is overloaded\"}}\n\n")
	}))
	defer server.Close()

	stream := newTestStream(t, server.URL)
	_, err := AccumulateStream(stream)
	if err == nil {
		t.Fatal("expected error from error stream")
	}
	if !strings.Contains(err.Error(), "overloaded") {
		t.Errorf("error = %q, want containing 'overloaded'", err.Error())
	}
}

// newTestStream creates a streaming connection to the test server.
func newTestStream(t *testing.T, serverURL string) *ssestream.Stream[anthropic.MessageStreamEventUnion] {
	t.Helper()
	client := anthropic.NewClient(
		option.WithAPIKey("test-key"),
		option.WithBaseURL(serverURL),
	)
	return client.Messages.NewStreaming(
		context.Background(),
		anthropic.MessageNewParams{
			Model:     anthropic.Model("claude-sonnet-4-20250514"),
			MaxTokens: 1024,
			Messages: []anthropic.MessageParam{
				anthropic.NewUserMessage(anthropic.NewTextBlock("test")),
			},
		},
	)
}

type sseEvent struct {
	eventType string
	data      string
}

// newSSEServer creates an httptest.Server that returns the given SSE events.
func newSSEServer(t *testing.T, events []sseEvent) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("response writer does not support flushing")
		}

		for _, evt := range events {
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", evt.eventType, evt.data)
			flusher.Flush()
		}
	}))
}
