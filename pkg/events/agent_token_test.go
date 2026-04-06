package events

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestAgentTokenTopic(t *testing.T) {
	tests := []struct {
		agentName string
		want      string
	}{
		{"builder", "agent.builder.tokens"},
		{"watcher", "agent.watcher.tokens"},
		{"scout", "agent.scout.tokens"},
		{"oracle", "agent.oracle.tokens"},
		{"my-agent-123", "agent.my-agent-123.tokens"},
	}

	for _, tt := range tests {
		t.Run(tt.agentName, func(t *testing.T) {
			got := AgentTokenTopic(tt.agentName)
			if got != tt.want {
				t.Errorf("AgentTokenTopic(%q) = %q, want %q", tt.agentName, got, tt.want)
			}
		})
	}
}

func TestNewAgentTokenEvent(t *testing.T) {
	evt := NewAgentTokenEvent("builder", "Hello", 0, false)

	if evt.AgentName != "builder" {
		t.Errorf("AgentName = %q, want %q", evt.AgentName, "builder")
	}
	if evt.Content != "Hello" {
		t.Errorf("Content = %q, want %q", evt.Content, "Hello")
	}
	if evt.Sequence != 0 {
		t.Errorf("Sequence = %d, want %d", evt.Sequence, 0)
	}
	if evt.IsComplete != false {
		t.Errorf("IsComplete = %v, want %v", evt.IsComplete, false)
	}
	if evt.Timestamp == "" {
		t.Error("Timestamp should not be empty")
	}

	// Verify timestamp is recent (within last second)
	ts, err := time.Parse("2006-01-02T15:04:05Z", evt.Timestamp)
	if err != nil {
		t.Errorf("Timestamp parse error: %v", err)
	}
	if time.Since(ts) > time.Second {
		t.Error("Timestamp should be recent")
	}
}

func TestAgentTokenEventToPayload(t *testing.T) {
	evt := NewAgentTokenEvent("watcher", "world", 1, true)
	payload, err := evt.ToPayload()
	if err != nil {
		t.Fatalf("ToPayload: %v", err)
	}

	// Verify it's valid JSON
	var decoded AgentTokenEvent
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	if decoded.AgentName != evt.AgentName {
		t.Errorf("AgentName mismatch: got %q, want %q", decoded.AgentName, evt.AgentName)
	}
	if decoded.Content != evt.Content {
		t.Errorf("Content mismatch: got %q, want %q", decoded.Content, evt.Content)
	}
	if decoded.Sequence != evt.Sequence {
		t.Errorf("Sequence mismatch: got %d, want %d", decoded.Sequence, evt.Sequence)
	}
	if decoded.IsComplete != evt.IsComplete {
		t.Errorf("IsComplete mismatch: got %v, want %v", decoded.IsComplete, evt.IsComplete)
	}
}

func TestAgentTokenEventFromPayload(t *testing.T) {
	original := NewAgentTokenEvent("scout", "test content", 5, false)
	payload, _ := original.ToPayload()

	decoded, err := AgentTokenEventFromPayload(payload)
	if err != nil {
		t.Fatalf("AgentTokenEventFromPayload: %v", err)
	}

	if decoded.AgentName != original.AgentName {
		t.Errorf("AgentName mismatch")
	}
	if decoded.Content != original.Content {
		t.Errorf("Content mismatch")
	}
	if decoded.Sequence != original.Sequence {
		t.Errorf("Sequence mismatch")
	}
	if decoded.IsComplete != original.IsComplete {
		t.Errorf("IsComplete mismatch")
	}
	if decoded.Timestamp != original.Timestamp {
		t.Errorf("Timestamp mismatch")
	}
}

func TestAgentTokenEventFromPayloadInvalid(t *testing.T) {
	invalidPayload := json.RawMessage(`{invalid json}`)
	_, err := AgentTokenEventFromPayload(invalidPayload)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestTopicMatchAgentToken(t *testing.T) {
	// Test that agent.*.tokens pattern matching works (wildcard at end)
	if !TopicMatch("agent.builder.*", "agent.builder.tokens") {
		t.Error("agent.builder.* should match agent.builder.tokens")
	}
	if !TopicMatch("agent.*", "agent.builder.tokens") {
		t.Error("agent.* should match agent.builder.tokens")
	}
	if TopicMatch("agent.builder.*", "agent.watcher.tokens") {
		t.Error("agent.builder.* should NOT match agent.watcher.tokens")
	}
	if TopicMatch("agent.*", "memory.store") {
		t.Error("agent.* should NOT match memory.store")
	}

	// Test specific agent subscription
	if !TopicMatch("agent.builder.tokens", "agent.builder.tokens") {
		t.Error("exact topic should match")
	}
	if TopicMatch("agent.builder.tokens", "agent.watcher.tokens") {
		t.Error("different agent should not match exact topic")
	}
}

func TestPublishAgentToken(t *testing.T) {
	bus, _ := newTestBus(t)
	defer bus.Close()
	ctx := context.Background()

	// Subscribe to agent token events
	ch, err := bus.Subscribe("agent.builder.tokens")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	// Publish a token event
	evt, err := bus.PublishAgentToken(ctx, "builder", "Hello ", 0, false)
	if err != nil {
		t.Fatalf("PublishAgentToken: %v", err)
	}
	if evt == nil {
		t.Fatal("PublishAgentToken returned nil event")
	}

	// Verify the event was published with correct topic
	if evt.Topic != "agent.builder.tokens" {
		t.Errorf("Topic = %q, want %q", evt.Topic, "agent.builder.tokens")
	}

	// Receive and verify
	select {
	case received := <-ch:
		if received.ID != evt.ID {
			t.Error("received event ID mismatch")
		}

		// Decode the payload
		tokenEvt, err := AgentTokenEventFromPayload(received.Payload)
		if err != nil {
			t.Fatalf("AgentTokenEventFromPayload: %v", err)
		}

		if tokenEvt.AgentName != "builder" {
			t.Errorf("AgentName = %q, want %q", tokenEvt.AgentName, "builder")
		}
		if tokenEvt.Content != "Hello " {
			t.Errorf("Content = %q, want %q", tokenEvt.Content, "Hello ")
		}
		if tokenEvt.Sequence != 0 {
			t.Errorf("Sequence = %d, want %d", tokenEvt.Sequence, 0)
		}
		if tokenEvt.IsComplete != false {
			t.Errorf("IsComplete = %v, want %v", tokenEvt.IsComplete, false)
		}

	case <-time.After(time.Second):
		t.Fatal("timed out waiting for token event")
	}
}

func TestPublishAgentTokenStream(t *testing.T) {
	bus, _ := newTestBus(t)
	defer bus.Close()
	ctx := context.Background()

	// Subscribe to all agent token events
	ch, _ := bus.Subscribe("agent.*")

	// Simulate a streaming response with multiple tokens
	tokens := []string{"Hello", " ", "world", "!"}
	var publishedIDs []string

	for i, token := range tokens {
		isComplete := i == len(tokens)-1
		evt, err := bus.PublishAgentToken(ctx, "oracle", token, i, isComplete)
		if err != nil {
			t.Fatalf("PublishAgentToken at seq %d: %v", i, err)
		}
		publishedIDs = append(publishedIDs, evt.ID)
	}

	// Receive all tokens
	var receivedContents []string
	var receivedSequences []int
	var lastComplete bool

	for i := 0; i < len(tokens); i++ {
		select {
		case received := <-ch:
			tokenEvt, err := AgentTokenEventFromPayload(received.Payload)
			if err != nil {
				t.Fatalf("AgentTokenEventFromPayload: %v", err)
			}
			receivedContents = append(receivedContents, tokenEvt.Content)
			receivedSequences = append(receivedSequences, tokenEvt.Sequence)
			lastComplete = tokenEvt.IsComplete

			// Verify source matches agent name
			if received.Source != "oracle" {
				t.Errorf("Source = %q, want %q", received.Source, "oracle")
			}

		case <-time.After(time.Second):
			t.Fatalf("timed out at token %d", i)
		}
	}

	// Verify all tokens received in order
	if strings.Join(receivedContents, "") != strings.Join(tokens, "") {
		t.Errorf("contents mismatch: got %v, want %v", receivedContents, tokens)
	}

	// Verify sequences are correct
	for i, seq := range receivedSequences {
		if seq != i {
			t.Errorf("sequence[%d] = %d, want %d", i, seq, i)
		}
	}

	// Verify last token is marked complete
	if !lastComplete {
		t.Error("last token should be marked complete")
	}
}

func TestPublishAgentTokenMultipleAgents(t *testing.T) {
	bus, _ := newTestBus(t)
	defer bus.Close()
	ctx := context.Background()

	// Subscribe to specific agents
	builderCh, _ := bus.Subscribe("agent.builder.tokens")
	watcherCh, _ := bus.Subscribe("agent.watcher.tokens")
	allCh, _ := bus.Subscribe("agent.*")

	// Publish from builder
	bus.PublishAgentToken(ctx, "builder", "Building...", 0, false)

	// Publish from watcher
	bus.PublishAgentToken(ctx, "watcher", "Watching...", 0, false)

	// Verify builder subscriber only gets builder events
	select {
	case evt := <-builderCh:
		tokenEvt, _ := AgentTokenEventFromPayload(evt.Payload)
		if tokenEvt.AgentName != "builder" {
			t.Errorf("builderCh received event from %s", tokenEvt.AgentName)
		}
	case <-time.After(time.Second):
		t.Fatal("builderCh timed out")
	}

	// Verify watcher subscriber only gets watcher events
	select {
	case evt := <-watcherCh:
		tokenEvt, _ := AgentTokenEventFromPayload(evt.Payload)
		if tokenEvt.AgentName != "watcher" {
			t.Errorf("watcherCh received event from %s", tokenEvt.AgentName)
		}
	case <-time.After(time.Second):
		t.Fatal("watcherCh timed out")
	}

	// Verify wildcard subscriber gets both
	receivedCount := 0
	timeout := time.After(500 * time.Millisecond)
	for receivedCount < 2 {
		select {
		case <-allCh:
			receivedCount++
		case <-timeout:
			break
		}
	}
	if receivedCount != 2 {
		t.Errorf("allCh received %d events, want 2", receivedCount)
	}
}

func TestAgentTokenEventJSONFields(t *testing.T) {
	evt := &AgentTokenEvent{
		AgentName:  "test-agent",
		Content:    "test content",
		Timestamp:  "2026-04-06T12:00:00Z",
		IsComplete: true,
		Sequence:   42,
	}

	payload, err := evt.ToPayload()
	if err != nil {
		t.Fatalf("ToPayload: %v", err)
	}

	s := string(payload)

	// Verify all expected fields are present
	expectedFields := []string{
		`"agent_name"`,
		`"content"`,
		`"timestamp"`,
		`"is_complete"`,
		`"sequence"`,
	}

	for _, field := range expectedFields {
		if !strings.Contains(s, field) {
			t.Errorf("missing field %s in JSON", field)
		}
	}

	// Verify values are correct
	if !strings.Contains(s, `"test-agent"`) {
		t.Error("agent_name value incorrect")
	}
	if !strings.Contains(s, `"test content"`) {
		t.Error("content value incorrect")
	}
	if !strings.Contains(s, `"2026-04-06T12:00:00Z"`) {
		t.Error("timestamp value incorrect")
	}
	if !strings.Contains(s, `true`) {
		t.Error("is_complete value incorrect")
	}
}
