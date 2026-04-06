package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// AgentTokenEvent represents a single token from an agent's streaming response.
// These events are published to the event bus for real-time token delivery
// to subscribers (e.g., WebSocket clients, SSE streams, or other agents).
type AgentTokenEvent struct {
	// AgentName identifies the agent producing the token (e.g., "builder", "watcher")
	AgentName string `json:"agent_name"`
	// Content is the actual token content (may be partial word, full word, or chunk)
	Content string `json:"content"`
	// Timestamp is when the token was generated (ISO-8601 UTC)
	Timestamp string `json:"timestamp"`
	// IsComplete indicates if this is the final token in the stream
	IsComplete bool `json:"is_complete"`
	// Sequence is the zero-based sequence number of this token in the stream
	Sequence int `json:"sequence"`
}

// AgentTokenTopic returns the topic name for an agent's token stream.
// Pattern: agent.{name}.tokens
// Examples:
//   - agent.builder.tokens
//   - agent.watcher.tokens
//   - agent.scout.tokens
func AgentTokenTopic(agentName string) string {
	return fmt.Sprintf("agent.%s.tokens", agentName)
}

// NewAgentTokenEvent creates a new AgentTokenEvent with the current timestamp.
func NewAgentTokenEvent(agentName, content string, sequence int, isComplete bool) *AgentTokenEvent {
	return &AgentTokenEvent{
		AgentName:  agentName,
		Content:    content,
		Timestamp:  FormatTimestamp(time.Now().UTC()),
		IsComplete: isComplete,
		Sequence:   sequence,
	}
}

// ToPayload serializes the AgentTokenEvent to JSON for publishing to the event bus.
func (e *AgentTokenEvent) ToPayload() (json.RawMessage, error) {
	data, err := json.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("events: marshal agent token event: %w", err)
	}
	return json.RawMessage(data), nil
}

// AgentTokenEventFromPayload deserializes an AgentTokenEvent from a JSON payload.
func AgentTokenEventFromPayload(payload json.RawMessage) (*AgentTokenEvent, error) {
	var evt AgentTokenEvent
	if err := json.Unmarshal(payload, &evt); err != nil {
		return nil, fmt.Errorf("events: unmarshal agent token event: %w", err)
	}
	return &evt, nil
}

// PublishAgentToken publishes a token event to the event bus for the given agent.
// This is a convenience helper that wraps the event bus Publish method.
func (b *Bus) PublishAgentToken(ctx context.Context, agentName, content string, sequence int, isComplete bool) (*Event, error) {
	evt := NewAgentTokenEvent(agentName, content, sequence, isComplete)
	payload, err := evt.ToPayload()
	if err != nil {
		return nil, err
	}
	topic := AgentTokenTopic(agentName)
	return b.Publish(ctx, topic, payload, agentName)
}
