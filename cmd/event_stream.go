package cmd

import (
	"fmt"
	"os"
	"path/filepath"
)

// Default event stream directory and file names.
const (
	EventsDirName     = "events"
	CurrentStreamName = "current.ndjson"
)

// InitEventStream ensures the events directory exists and returns the path
// to the current NDJSON stream file.
func InitEventStream() (string, error) {
	eventsDir := filepath.Join(".aether", EventsDirName)
	if err := os.MkdirAll(eventsDir, 0755); err != nil {
		return "", fmt.Errorf("creating events directory: %w", err)
	}
	streamPath := filepath.Join(eventsDir, CurrentStreamName)
	return streamPath, nil
}

// GetEventStreamPath returns the default event stream path relative to the
// current working directory.
func GetEventStreamPath() string {
	return filepath.Join(".aether", EventsDirName, CurrentStreamName)
}

// EmitPhaseStart writes a phase_start event to the default stream.
// This is a convenience helper for integrating event emission into existing commands.
func EmitPhaseStart(phase, plan, goal string) error {
	streamPath := GetEventStreamPath()
	payload := EventPayload{
		"phase": phase,
		"plan":  plan,
		"goal":  goal,
	}
	line := NewEventLine(EventTypePhaseStart, payload)
	return WriteEvent(streamPath, line)
}

// EmitAgentSelected writes an agent_selected event to the default stream.
func EmitAgentSelected(agentID, caste, task string) error {
	streamPath := GetEventStreamPath()
	payload := EventPayload{
		"agent_id": agentID,
		"caste":    caste,
		"task":     task,
	}
	line := NewEventLine(EventTypeAgentSelected, payload)
	return WriteEvent(streamPath, line)
}

// EmitWorkerSpawned writes a worker_spawned event to the default stream.
func EmitWorkerSpawned(workerID, caste, parentAgentID string) error {
	streamPath := GetEventStreamPath()
	payload := EventPayload{
		"worker_id":       workerID,
		"caste":           caste,
		"parent_agent_id": parentAgentID,
	}
	line := NewEventLine(EventTypeWorkerSpawned, payload)
	return WriteEvent(streamPath, line)
}

// EmitVerificationResult writes a verification_result event to the default stream.
func EmitVerificationResult(taskID string, passed bool, message string) error {
	streamPath := GetEventStreamPath()
	payload := EventPayload{
		"task_id": taskID,
		"passed":  passed,
		"message": message,
	}
	line := NewEventLine(EventTypeVerificationResult, payload)
	return WriteEvent(streamPath, line)
}

// EmitMemoryUpdate writes a memory_update event to the default stream.
func EmitMemoryUpdate(key string, value interface{}, source string) error {
	streamPath := GetEventStreamPath()
	payload := EventPayload{
		"key":    key,
		"value":  value,
		"source": source,
	}
	line := NewEventLine(EventTypeMemoryUpdate, payload)
	return WriteEvent(streamPath, line)
}

// EmitRunSeal writes a run_seal event to the default stream.
func EmitRunSeal(phase, plan, summary string) error {
	streamPath := GetEventStreamPath()
	payload := EventPayload{
		"phase":   phase,
		"plan":    plan,
		"summary": summary,
	}
	line := NewEventLine(EventTypeRunSeal, payload)
	return WriteEvent(streamPath, line)
}
