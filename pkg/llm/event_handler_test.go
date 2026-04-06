package llm

import (
	"context"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/storage"
)

func newTestBus(t *testing.T) (*events.Bus, string) {
	t.Helper()
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	bus := events.NewBus(store, events.Config{JSONLFile: "event-bus.jsonl"})
	return bus, dir
}

func TestEventBusStreamHandler_BasicTokenPublishing(t *testing.T) {
	bus, _ := newTestBus(t)
	defer bus.Close()
	ctx := context.Background()

	// Subscribe to agent token events
	ch, err := bus.Subscribe("agent.builder.tokens")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	// Create handler with short batch interval for testing
	config := EventBusStreamHandlerConfig{
		AgentName:     "builder",
		Bus:           bus,
		BatchInterval: 10 * time.Millisecond,
	}
	handler := NewEventBusStreamHandler(ctx, config)
	defer handler.Stop()

	// Send tokens
	handler.OnToken("Hello ")
	handler.OnToken("World")

	// Wait for batch to flush
	time.Sleep(50 * time.Millisecond)

	// Receive batched event
	select {
	case evt := <-ch:
		tokenEvt, err := events.AgentTokenEventFromPayload(evt.Payload)
		if err != nil {
			t.Fatalf("AgentTokenEventFromPayload: %v", err)
		}
		if tokenEvt.AgentName != "builder" {
			t.Errorf("AgentName = %q, want %q", tokenEvt.AgentName, "builder")
		}
		if tokenEvt.Content != "Hello World" {
			t.Errorf("Content = %q, want %q", tokenEvt.Content, "Hello World")
		}
		if tokenEvt.Sequence != 0 {
			t.Errorf("Sequence = %d, want 0", tokenEvt.Sequence)
		}
		if tokenEvt.IsComplete {
			t.Error("IsComplete should be false for regular token")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for token event")
	}
}

func TestEventBusStreamHandler_OnComplete(t *testing.T) {
	bus, _ := newTestBus(t)
	defer bus.Close()
	ctx := context.Background()

	ch, _ := bus.Subscribe("agent.watcher.tokens")

	config := EventBusStreamHandlerConfig{
		AgentName:     "watcher",
		Bus:           bus,
		BatchInterval: 10 * time.Millisecond,
	}
	handler := NewEventBusStreamHandler(ctx, config)

	// Send a token and complete
	handler.OnToken("Final result")
	time.Sleep(20 * time.Millisecond)

	// Call OnComplete
	result := &StreamResult{
		Text:       "Final result",
		Role:       "assistant",
		Model:      "claude-test",
		StopReason: "end_turn",
	}
	handler.OnComplete(result)

	// Should receive the token batch
	select {
	case evt := <-ch:
		tokenEvt, _ := events.AgentTokenEventFromPayload(evt.Payload)
		if tokenEvt.Content != "Final result" {
			t.Errorf("Content = %q, want %q", tokenEvt.Content, "Final result")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for token")
	}
}

func TestEventBusStreamHandler_OnError(t *testing.T) {
	bus, _ := newTestBus(t)
	defer bus.Close()
	ctx := context.Background()

	ch, _ := bus.Subscribe("agent.scout.tokens")

	config := EventBusStreamHandlerConfig{
		AgentName:     "scout",
		Bus:           bus,
		BatchInterval: 10 * time.Millisecond,
	}
	handler := NewEventBusStreamHandler(ctx, config)

	// Send a token
	handler.OnToken("Partial ")
	time.Sleep(20 * time.Millisecond)

	// Call OnError
	testErr := context.Canceled
	handler.OnError(testErr)

	// Should still receive the flushed token
	select {
	case evt := <-ch:
		tokenEvt, _ := events.AgentTokenEventFromPayload(evt.Payload)
		if tokenEvt.Content != "Partial " {
			t.Errorf("Content = %q, want %q", tokenEvt.Content, "Partial ")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for token")
	}
}

func TestEventBusStreamHandler_ToolEvents(t *testing.T) {
	bus, _ := newTestBus(t)
	defer bus.Close()
	ctx := context.Background()

	ch, _ := bus.Subscribe("agent.builder.tokens")

	config := EventBusStreamHandlerConfig{
		AgentName:     "builder",
		Bus:           bus,
		BatchInterval: 50 * time.Millisecond, // Long interval to verify tools publish immediately
	}
	handler := NewEventBusStreamHandler(ctx, config)
	defer handler.Stop()

	// Tool start should publish immediately (not batched)
	handler.OnToolStart("read_file", "tool_123")

	select {
	case evt := <-ch:
		tokenEvt, _ := events.AgentTokenEventFromPayload(evt.Payload)
		if tokenEvt.Content != "[Tool: read_file]" {
			t.Errorf("Tool start content = %q, want %q", tokenEvt.Content, "[Tool: read_file]")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for tool start event")
	}

	// Tool end should also publish immediately
	handler.OnToolEnd("read_file", "tool_123", "file content")

	select {
	case evt := <-ch:
		tokenEvt, _ := events.AgentTokenEventFromPayload(evt.Payload)
		if tokenEvt.Content != "[/Tool: read_file]" {
			t.Errorf("Tool end content = %q, want %q", tokenEvt.Content, "[/Tool: read_file]")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for tool end event")
	}
}

func TestEventBusStreamHandler_SequenceIncrementing(t *testing.T) {
	bus, _ := newTestBus(t)
	defer bus.Close()
	ctx := context.Background()

	ch, _ := bus.Subscribe("agent.oracle.tokens")

	config := EventBusStreamHandlerConfig{
		AgentName:     "oracle",
		Bus:           bus,
		BatchInterval: 10 * time.Millisecond,
	}
	handler := NewEventBusStreamHandler(ctx, config)
	defer handler.Stop()

	// Send multiple batches of tokens
	for batch := 0; batch < 3; batch++ {
		handler.OnToken("token")
		time.Sleep(25 * time.Millisecond) // Wait for batch to flush
	}

	// Collect all events
	var sequences []int
	timeout := time.After(time.Second)
	for i := 0; i < 3; i++ {
		select {
		case evt := <-ch:
			tokenEvt, _ := events.AgentTokenEventFromPayload(evt.Payload)
			sequences = append(sequences, tokenEvt.Sequence)
		case <-timeout:
			t.Fatalf("timed out at event %d", i)
		}
	}

	// Verify sequences are 0, 1, 2
	for i, seq := range sequences {
		if seq != i {
			t.Errorf("sequence[%d] = %d, want %d", i, seq, i)
		}
	}
}

func TestEventBusStreamHandler_MultipleAgents(t *testing.T) {
	bus, _ := newTestBus(t)
	defer bus.Close()
	ctx := context.Background()

	builderCh, _ := bus.Subscribe("agent.builder.tokens")
	watcherCh, _ := bus.Subscribe("agent.watcher.tokens")

	// Create handlers for different agents
	builderConfig := EventBusStreamHandlerConfig{
		AgentName:     "builder",
		Bus:           bus,
		BatchInterval: 10 * time.Millisecond,
	}
	watcherConfig := EventBusStreamHandlerConfig{
		AgentName:     "watcher",
		Bus:           bus,
		BatchInterval: 10 * time.Millisecond,
	}

	builderHandler := NewEventBusStreamHandler(ctx, builderConfig)
	watcherHandler := NewEventBusStreamHandler(ctx, watcherConfig)
	defer builderHandler.Stop()
	defer watcherHandler.Stop()

	// Send tokens from both handlers
	builderHandler.OnToken("Building...")
	watcherHandler.OnToken("Watching...")
	time.Sleep(50 * time.Millisecond)

	// Verify builder subscriber only gets builder events
	select {
	case evt := <-builderCh:
		tokenEvt, _ := events.AgentTokenEventFromPayload(evt.Payload)
		if tokenEvt.AgentName != "builder" {
			t.Errorf("builderCh received event from %s", tokenEvt.AgentName)
		}
		if tokenEvt.Content != "Building..." {
			t.Errorf("builderCh content = %q, want %q", tokenEvt.Content, "Building...")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for builder event")
	}

	// Verify watcher subscriber only gets watcher events
	select {
	case evt := <-watcherCh:
		tokenEvt, _ := events.AgentTokenEventFromPayload(evt.Payload)
		if tokenEvt.AgentName != "watcher" {
			t.Errorf("watcherCh received event from %s", tokenEvt.AgentName)
		}
		if tokenEvt.Content != "Watching..." {
			t.Errorf("watcherCh content = %q, want %q", tokenEvt.Content, "Watching...")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for watcher event")
	}
}

func TestEventBusStreamHandler_EmptyTokenIgnored(t *testing.T) {
	bus, _ := newTestBus(t)
	defer bus.Close()
	ctx := context.Background()

	ch, _ := bus.Subscribe("agent.builder.tokens")

	config := EventBusStreamHandlerConfig{
		AgentName:     "builder",
		Bus:           bus,
		BatchInterval: 10 * time.Millisecond,
	}
	handler := NewEventBusStreamHandler(ctx, config)
	defer handler.Stop()

	// Send empty token (should be ignored)
	handler.OnToken("")
	handler.OnToken("valid")
	time.Sleep(50 * time.Millisecond)

	// Should only receive the valid token
	select {
	case evt := <-ch:
		tokenEvt, _ := events.AgentTokenEventFromPayload(evt.Payload)
		if tokenEvt.Content != "valid" {
			t.Errorf("Content = %q, want %q", tokenEvt.Content, "valid")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for token")
	}

	// Should not receive another event
	select {
	case <-ch:
		t.Error("should not receive event for empty token")
	case <-time.After(50 * time.Millisecond):
		// Expected
	}
}

func TestEventBusStreamHandler_StopFlushesRemaining(t *testing.T) {
	bus, _ := newTestBus(t)
	defer bus.Close()
	ctx := context.Background()

	ch, _ := bus.Subscribe("agent.builder.tokens")

	config := EventBusStreamHandlerConfig{
		AgentName:     "builder",
		Bus:           bus,
		BatchInterval: 1 * time.Hour, // Very long interval - won't flush naturally
	}
	handler := NewEventBusStreamHandler(ctx, config)

	// Send token without waiting for batch
	handler.OnToken("unflushed")

	// Stop should flush remaining tokens
	handler.Stop()

	// Should receive the flushed token
	select {
	case evt := <-ch:
		tokenEvt, _ := events.AgentTokenEventFromPayload(evt.Payload)
		if tokenEvt.Content != "unflushed" {
			t.Errorf("Content = %q, want %q", tokenEvt.Content, "unflushed")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for flushed token")
	}
}

func TestEventBusStreamHandler_DefaultBatchInterval(t *testing.T) {
	bus, _ := newTestBus(t)
	defer bus.Close()
	ctx := context.Background()

	config := EventBusStreamHandlerConfig{
		AgentName:     "builder",
		Bus:           bus,
		BatchInterval: 0, // Should default to 50ms
	}
	handler := NewEventBusStreamHandler(ctx, config)
	defer handler.Stop()

	// The handler should have been created with default interval
	// We can't directly test the interval, but we can verify it works
	if handler == nil {
		t.Fatal("handler should not be nil")
	}
}

func TestEventBusStreamHandler_ImplementsStreamHandler(t *testing.T) {
	// Verify EventBusStreamHandler implements StreamHandler interface
	var _ StreamHandler = (*EventBusStreamHandler)(nil)
}
