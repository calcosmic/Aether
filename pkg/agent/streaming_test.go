package agent

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/llm"
)

// mockStreamHandler captures streaming events for testing.
type mockStreamHandler struct {
	mu           sync.Mutex
	tokens       []string
	toolsStarted []string
	toolsEnded   []string
	completed    bool
	err          error
	result       *llm.StreamResult
}

func (m *mockStreamHandler) OnToken(token string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokens = append(m.tokens, token)
}

func (m *mockStreamHandler) OnToolStart(toolName, toolID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.toolsStarted = append(m.toolsStarted, toolName)
}

func (m *mockStreamHandler) OnToolEnd(toolName, toolID, result string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.toolsEnded = append(m.toolsEnded, toolName)
}

func (m *mockStreamHandler) OnComplete(result *llm.StreamResult) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.completed = true
	m.result = result
}

func (m *mockStreamHandler) OnError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

func (m *mockStreamHandler) IsCompleted() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.completed
}

// Test that BuilderAgent implements Agent interface.
func TestBuilderAgentImplementsAgent(t *testing.T) {
	builder := NewBuilderAgent("test-builder")

	var _ Agent = builder

	if builder.Name() != "test-builder" {
		t.Errorf("Name() = %q, want %q", builder.Name(), "test-builder")
	}

	if builder.Caste() != CasteBuilder {
		t.Errorf("Caste() = %q, want %q", builder.Caste(), CasteBuilder)
	}

	triggers := builder.Triggers()
	if len(triggers) != 2 {
		t.Errorf("len(Triggers()) = %d, want 2", len(triggers))
	}
}

// Test that BuilderAgent implements StreamingAgent interface.
func TestBuilderAgentImplementsStreamingAgent(t *testing.T) {
	builder := NewBuilderAgent("test-builder")

	var _ StreamingAgent = builder

	// Verify it can be type-asserted
	_, ok := IsStreamingAgent(builder)
	if !ok {
		t.Error("BuilderAgent should be detected as StreamingAgent")
	}
}

// Test non-streaming execution (backward compatibility).
func TestBuilderAgentNonStreamingExecute(t *testing.T) {
	builder := NewBuilderAgent("test-builder")
	event := events.Event{Topic: "build.start"}

	ctx := context.Background()
	err := builder.Execute(ctx, event)

	if err != nil {
		t.Errorf("Execute() error = %v, want nil", err)
	}
}

// Test streaming execution with handler.
func TestBuilderAgentStreamingExecute(t *testing.T) {
	builder := NewBuilderAgent("test-builder")
	event := events.Event{Topic: "build.start"}
	handler := &mockStreamHandler{}

	ctx := context.Background()
	err := builder.ExecuteStreaming(ctx, event, handler)

	if err != nil {
		t.Errorf("ExecuteStreaming() error = %v, want nil", err)
	}

	// Verify tokens were streamed
	if len(handler.tokens) == 0 {
		t.Error("Expected tokens to be streamed, got none")
	}

	// Check for expected content in tokens
	var allTokens string
	for _, token := range handler.tokens {
		allTokens += token
	}

	expectedPhrases := []string{"Starting build", "Analyzing", "Implementing", "verification"}
	for _, phrase := range expectedPhrases {
		if !strings.Contains(allTokens, phrase) {
			t.Errorf("Expected token to contain %q, tokens were: %s", phrase, allTokens)
		}
	}

	// Verify completion callback
	if !handler.completed {
		t.Error("Expected OnComplete to be called")
	}

	if handler.result == nil {
		t.Error("Expected result in OnComplete callback")
	} else if !strings.Contains(handler.result.Text, "completed successfully") {
		t.Errorf("Expected result text to contain 'completed successfully', got: %s", handler.result.Text)
	}
}

// Test context cancellation during streaming.
func TestBuilderAgentStreamingContextCancellation(t *testing.T) {
	builder := NewBuilderAgent("test-builder")
	event := events.Event{Topic: "build.start"}
	handler := &mockStreamHandler{}

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := builder.ExecuteStreaming(ctx, event, handler)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("ExecuteStreaming() error = %v, want context.Canceled", err)
	}
}

// Test context cancellation with timeout.
func TestBuilderAgentStreamingTimeout(t *testing.T) {
	builder := NewBuilderAgent("test-builder")
	event := events.Event{Topic: "build.start"}
	handler := &mockStreamHandler{}

	// Create a context that times out very quickly
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Small sleep to ensure timeout
	time.Sleep(10 * time.Millisecond)

	err := builder.ExecuteStreaming(ctx, event, handler)

	if !errors.Is(err, context.DeadlineExceeded) {
		// It might also be canceled depending on timing, so accept either
		if !errors.Is(err, context.Canceled) {
			t.Errorf("ExecuteStreaming() error = %v, want context.DeadlineExceeded or context.Canceled", err)
		}
	}
}

// Test ExecuteWithOptions with streaming enabled.
func TestExecuteWithOptionsStreaming(t *testing.T) {
	builder := NewBuilderAgent("test-builder")
	event := events.Event{Topic: "build.start"}
	handler := &mockStreamHandler{}

	ctx := context.Background()
	err := ExecuteWithOptions(ctx, builder, event, WithStreaming(handler))

	if err != nil {
		t.Errorf("ExecuteWithOptions() error = %v, want nil", err)
	}

	// Verify streaming occurred
	if len(handler.tokens) == 0 {
		t.Error("Expected tokens to be streamed when WithStreaming is used")
	}

	if !handler.completed {
		t.Error("Expected OnComplete to be called")
	}
}

// Test ExecuteWithOptions without streaming (backward compatibility).
func TestExecuteWithOptionsNoStreaming(t *testing.T) {
	builder := NewBuilderAgent("test-builder")
	event := events.Event{Topic: "build.start"}

	ctx := context.Background()
	err := ExecuteWithOptions(ctx, builder, event) // No WithStreaming option

	if err != nil {
		t.Errorf("ExecuteWithOptions() error = %v, want nil", err)
	}
}

// Test ExecuteWithOptions with non-streaming agent (fallback).
func TestExecuteWithOptionsFallbackToNonStreaming(t *testing.T) {
	// Create a mock agent that only implements Agent, not StreamingAgent
	mock := &mockAgent{
		name:  "non-streaming-agent",
		caste: CasteWatcher,
	}

	event := events.Event{Topic: "test.event"}
	handler := &mockStreamHandler{}

	ctx := context.Background()
	// Even with WithStreaming, it should fall back to regular Execute
	err := ExecuteWithOptions(ctx, mock, event, WithStreaming(handler))

	if err != nil {
		t.Errorf("ExecuteWithOptions() error = %v, want nil", err)
	}

	// No tokens should be streamed since the agent doesn't support it
	if len(handler.tokens) != 0 {
		t.Errorf("Expected no tokens for non-streaming agent, got: %v", handler.tokens)
	}
}

// Test IsStreamingAgent with streaming agent.
func TestIsStreamingAgentWithStreamingAgent(t *testing.T) {
	builder := NewBuilderAgent("test-builder")

	sa, ok := IsStreamingAgent(builder)
	if !ok {
		t.Error("IsStreamingAgent should return true for BuilderAgent")
	}
	if sa == nil {
		t.Error("IsStreamingAgent should return non-nil StreamingAgent")
	}
	if sa.Name() != "test-builder" {
		t.Errorf("StreamingAgent.Name() = %q, want %q", sa.Name(), "test-builder")
	}
}

// Test IsStreamingAgent with non-streaming agent.
func TestIsStreamingAgentWithNonStreamingAgent(t *testing.T) {
	mock := &mockAgent{
		name:  "non-streaming",
		caste: CasteWatcher,
	}

	sa, ok := IsStreamingAgent(mock)
	if ok {
		t.Error("IsStreamingAgent should return false for mockAgent")
	}
	if sa != nil {
		t.Error("IsStreamingAgent should return nil for non-streaming agent")
	}
}

// Test WithStreaming option.
func TestWithStreamingOption(t *testing.T) {
	handler := &mockStreamHandler{}
	opt := WithStreaming(handler)

	cfg := &executeConfig{}
	opt(cfg)

	if cfg.streamHandler == nil {
		t.Error("WithStreaming should set streamHandler")
	}
	if cfg.streamHandler != handler {
		t.Error("WithStreaming should set the correct handler")
	}
}

// Test streaming with nil handler (should not panic).
func TestBuilderAgentStreamingWithNilHandler(t *testing.T) {
	builder := NewBuilderAgent("test-builder")
	event := events.Event{Topic: "build.start"}

	ctx := context.Background()
	// Should not panic with nil handler
	err := builder.ExecuteStreaming(ctx, event, nil)

	if err != nil {
		t.Errorf("ExecuteStreaming() with nil handler error = %v, want nil", err)
	}
}

// Test multiple streaming agents can work independently.
func TestMultipleStreamingAgents(t *testing.T) {
	builder1 := NewBuilderAgent("builder-1")
	builder2 := NewBuilderAgent("builder-2")

	handler1 := &mockStreamHandler{}
	handler2 := &mockStreamHandler{}

	event := events.Event{Topic: "build.start"}
	ctx := context.Background()

	// Execute both agents concurrently with proper synchronization
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		builder1.ExecuteStreaming(ctx, event, handler1)
	}()
	err := builder2.ExecuteStreaming(ctx, event, handler2)

	if err != nil {
		t.Errorf("Second builder error = %v, want nil", err)
	}

	// Wait for first builder to complete
	wg.Wait()

	// Both should have completed
	if !handler1.IsCompleted() {
		t.Error("First builder should have completed")
	}
	if !handler2.IsCompleted() {
		t.Error("Second builder should have completed")
	}
}

// Test that regular agents still work without any changes.
func TestBackwardCompatibilityWithMockAgent(t *testing.T) {
	mock := &mockAgent{
		name:     "legacy-agent",
		caste:    CasteScout,
		triggers: []Trigger{{Topic: "scout.*"}},
	}

	// Should work with registry
	reg := NewRegistry()
	if err := reg.Register(mock); err != nil {
		t.Errorf("Register() error = %v, want nil", err)
	}

	// Should work with Execute
	ctx := context.Background()
	event := events.Event{Topic: "scout.explore"}
	if err := mock.Execute(ctx, event); err != nil {
		t.Errorf("Execute() error = %v, want nil", err)
	}
}
