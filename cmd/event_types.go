// Package cmd provides the Aether CLI commands.
//
// Event domain types for the Aether runtime event stream.
//
// The event stream is the shared observable truth between Go and TypeScript.
// Both sides must agree on the NDJSON line format:
//
//	{ "type": "...", "timestamp": "...", "payload": { ... } }
package cmd

import (
	"encoding/json"
	"time"
)

// EventType identifies the kind of event in the stream.
type EventType string

// Event type constants. These mirror the TypeScript EventType union.
const (
	EventTypePhaseStart         EventType = "phase_start"
	EventTypeAgentSelected      EventType = "agent_selected"
	EventTypeTaskPlanned        EventType = "task_planned"
	EventTypeWorkerSpawned      EventType = "worker_spawned"
	EventTypeToolCall           EventType = "tool_call"
	EventTypeVerificationResult EventType = "verification_result"
	EventTypeMemoryUpdate       EventType = "memory_update"
	EventTypeRunSeal            EventType = "run_seal"
	EventTypeCustom             EventType = "custom"
)

// EventPayload carries the structured data for an event.
// Using map[string]interface{} provides the same flexibility as the TS
// discriminated union while keeping the Go side lightweight.
type EventPayload map[string]interface{}

// EventLine is a single NDJSON line in the event stream.
type EventLine struct {
	Type      EventType    `json:"type"`
	Timestamp string       `json:"timestamp"`
	Payload   EventPayload `json:"payload"`
}

// NewEventLine creates a structured event line with the current ISO timestamp.
func NewEventLine(eventType EventType, payload EventPayload) EventLine {
	return EventLine{
		Type:      eventType,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Payload:   payload,
	}
}

// MarshalJSON implements custom JSON marshalling for EventLine to ensure
// consistent NDJSON output (compact, no extra whitespace).
func (e EventLine) MarshalJSON() ([]byte, error) {
	type alias EventLine // avoid recursion
	return json.Marshal(alias(e))
}

// UnmarshalJSON implements custom JSON unmarshalling for EventLine.
func (e *EventLine) UnmarshalJSON(data []byte) error {
	type alias EventLine
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*e = EventLine(a)
	return nil
}

// ToNDJSON returns the EventLine as a single NDJSON line (JSON + newline).
func (e EventLine) ToNDJSON() ([]byte, error) {
	b, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}
	b = append(b, '\n')
	return b, nil
}
