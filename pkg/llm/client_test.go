package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

func TestNewClientDefaultModel(t *testing.T) {
	client, err := NewClient(WithAPIKey("test-key"))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if client.model != string(anthropic.ModelClaudeSonnet4_20250514) {
		t.Errorf("default model = %q, want %q", client.model, anthropic.ModelClaudeSonnet4_20250514)
	}
}

func TestNewClientWithAPIKey(t *testing.T) {
	client, err := NewClient(WithAPIKey("sk-test-key-123"))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if client.apiClient == nil {
		t.Error("apiClient is nil")
	}
}

func TestNewClientMissingKey(t *testing.T) {
	// Ensure env var is not set
	t.Setenv("ANTHROPIC_API_KEY", "")

	_, err := NewClient()
	if err == nil {
		t.Fatal("expected error when no API key available")
	}
	if !strings.Contains(err.Error(), "ANTHROPIC_API_KEY") {
		t.Errorf("error = %q, want containing 'ANTHROPIC_API_KEY'", err.Error())
	}
}

func TestNewClientWithModel(t *testing.T) {
	client, err := NewClient(
		WithAPIKey("test-key"),
		WithModel("claude-opus-4-20250514"),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if client.model != "claude-opus-4-20250514" {
		t.Errorf("model = %q, want %q", client.model, "claude-opus-4-20250514")
	}
}

func TestNewClientWithMaxTokens(t *testing.T) {
	client, err := NewClient(
		WithAPIKey("test-key"),
		WithMaxTokens(8192),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if client.maxTokens != 8192 {
		t.Errorf("maxTokens = %d, want %d", client.maxTokens, 8192)
	}
}

func TestSendMessageMock(t *testing.T) {
	// Mock Anthropic API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and path
		if r.Method != http.MethodPost {
			t.Errorf("request method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/v1/messages" {
			t.Errorf("request path = %q, want /v1/messages", r.URL.Path)
		}

		// Return a mock response matching Anthropic API format
		resp := map[string]interface{}{
			"id":         "msg_test123",
			"type":       "message",
			"role":       "assistant",
			"model":      "claude-sonnet-4-20250514",
			"stop_reason": "end_turn",
			"content": []map[string]interface{}{
				{"type": "text", "text": "Hello from mock"},
			},
			"usage": map[string]interface{}{
				"input_tokens":  10,
				"output_tokens": 5,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create client pointing to mock server
	sdkClient := anthropic.NewClient(
		option.WithAPIKey("test-key"),
		option.WithBaseURL(server.URL),
	)
	client := &Client{
		apiClient: &sdkClient,
		model:     "claude-sonnet-4-20250514",
		maxTokens: 4096,
	}

	// Send message
	messages := []anthropic.MessageParam{
		anthropic.NewUserMessage(
			anthropic.NewTextBlock("Hello"),
		),
	}
	resp, err := client.SendMessage(context.Background(), messages)
	if err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}

	// Verify response fields
	if resp.ID != "msg_test123" {
		t.Errorf("ID = %q, want %q", resp.ID, "msg_test123")
	}
	if resp.Role != "assistant" {
		t.Errorf("Role = %q, want %q", resp.Role, "assistant")
	}
	if resp.StopReason != "end_turn" {
		t.Errorf("StopReason = %q, want %q", resp.StopReason, "end_turn")
	}
	if resp.Model != "claude-sonnet-4-20250514" {
		t.Errorf("Model = %q, want %q", resp.Model, "claude-sonnet-4-20250514")
	}
	if len(resp.Content) != 1 {
		t.Fatalf("Content blocks = %d, want 1", len(resp.Content))
	}
	if resp.Content[0].Text != "Hello from mock" {
		t.Errorf("Content[0].Text = %q, want %q", resp.Content[0].Text, "Hello from mock")
	}
	if resp.Usage.InputTokens != 10 {
		t.Errorf("Usage.InputTokens = %d, want 10", resp.Usage.InputTokens)
	}
	if resp.Usage.OutputTokens != 5 {
		t.Errorf("Usage.OutputTokens = %d, want 5", resp.Usage.OutputTokens)
	}
}

func TestConvertSDKMessageNil(t *testing.T) {
	result := convertSDKMessage(nil)
	if result != nil {
		t.Errorf("convertSDKMessage(nil) = %v, want nil", result)
	}
}
