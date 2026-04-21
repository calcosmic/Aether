// Package trace provides structured trace logging for colony runs.
// Every run generates a run_id and appends trace entries to trace.jsonl.
package trace

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/calcosmic/Aether/pkg/storage"
)

// TraceLevel categorizes trace entries.
type TraceLevel string

const (
	TraceLevelState        TraceLevel = "state"
	TraceLevelPhase        TraceLevel = "phase"
	TraceLevelPheromone    TraceLevel = "pheromone"
	TraceLevelError        TraceLevel = "error"
	TraceLevelRecovery     TraceLevel = "recovery"
	TraceLevelIntervention TraceLevel = "intervention"
	TraceLevelToken        TraceLevel = "token"
	TraceLevelArtifact     TraceLevel = "artifact"
)

// TraceEntry is a single structured trace record.
type TraceEntry struct {
	ID        string                 `json:"id"`
	RunID     string                 `json:"run_id"`
	Timestamp string                 `json:"timestamp"`
	Level     TraceLevel             `json:"level"`
	Topic     string                 `json:"topic"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
	Source    string                 `json:"source"`
}

// Tracer wraps a storage.Store to append trace entries.
type Tracer struct {
	store *storage.Store
}

// NewTracer creates a Tracer backed by the given store.
func NewTracer(store *storage.Store) *Tracer {
	return &Tracer{store: store}
}

// Log appends a trace entry to trace.jsonl.
// Returns an error but never blocks callers; errors should be logged and ignored.
func (t *Tracer) Log(entry TraceEntry) error {
	if t.store == nil {
		return fmt.Errorf("trace: no store available")
	}
	if entry.ID == "" {
		entry.ID = generateTraceID()
	}
	if entry.Timestamp == "" {
		entry.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	return t.store.AppendJSONL("trace.jsonl", entry)
}

// LogStateTransition logs a colony state change.
func (t *Tracer) LogStateTransition(runID, from, to, source string) error {
	return t.Log(TraceEntry{
		RunID:  runID,
		Level:  TraceLevelState,
		Topic:  "state.transition",
		Source: source,
		Payload: map[string]interface{}{
			"from": from,
			"to":   to,
		},
	})
}

// LogPhaseChange logs a phase status change.
func (t *Tracer) LogPhaseChange(runID string, phase int, status, source string) error {
	return t.Log(TraceEntry{
		RunID:  runID,
		Level:  TraceLevelPhase,
		Topic:  "phase." + status,
		Source: source,
		Payload: map[string]interface{}{
			"phase":  phase,
			"status": status,
		},
	})
}

// LogError logs an error record.
func (t *Tracer) LogError(runID string, phase int, errID, severity, source string) error {
	return t.Log(TraceEntry{
		RunID:  runID,
		Level:  TraceLevelError,
		Topic:  "error.add",
		Source: source,
		Payload: map[string]interface{}{
			"phase":    phase,
			"error_id": errID,
			"severity": severity,
		},
	})
}

// LogPheromone logs a pheromone signal write.
func (t *Tracer) LogPheromone(runID, sigType, source string) error {
	return t.Log(TraceEntry{
		RunID:  runID,
		Level:  TraceLevelPheromone,
		Topic:  "pheromone.write",
		Source: source,
		Payload: map[string]interface{}{
			"type": sigType,
		},
	})
}

// LogIntervention logs a human or system intervention.
func (t *Tracer) LogIntervention(runID, topic, source string, payload map[string]interface{}) error {
	return t.Log(TraceEntry{
		RunID:   runID,
		Level:   TraceLevelIntervention,
		Topic:   topic,
		Source:  source,
		Payload: payload,
	})
}

// generateTraceID creates a short unique trace entry ID.
func generateTraceID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("trc_%d_%s", time.Now().Unix(), hex.EncodeToString(b))
}
