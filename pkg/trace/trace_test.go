package trace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/storage"
)

func TestTracerLogAppendsValidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	store, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	tracer := NewTracer(store)

	entry := TraceEntry{
		RunID:  "run_1",
		Level:  TraceLevelState,
		Topic:  "state.transition",
		Source: "test",
		Payload: map[string]interface{}{
			"from": "IDLE",
			"to":   "READY",
		},
	}
	if err := tracer.Log(entry); err != nil {
		t.Fatalf("log failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dataDir, "trace.jsonl"))
	if err != nil {
		t.Fatalf("read trace.jsonl: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	var read TraceEntry
	if err := json.Unmarshal([]byte(lines[0]), &read); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if read.RunID != "run_1" {
		t.Errorf("run_id = %q, want run_1", read.RunID)
	}
	if read.Level != TraceLevelState {
		t.Errorf("level = %q, want state", read.Level)
	}
	if read.ID == "" {
		t.Error("expected id to be auto-generated")
	}
	if read.Timestamp == "" {
		t.Error("expected timestamp to be auto-generated")
	}
}

func TestConvenienceMethods(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	store, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	tracer := NewTracer(store)

	if err := tracer.LogStateTransition("run_1", "IDLE", "READY", "state-mutate"); err != nil {
		t.Fatalf("LogStateTransition: %v", err)
	}
	if err := tracer.LogPhaseChange("run_1", 2, "start", "phase-advance"); err != nil {
		t.Fatalf("LogPhaseChange: %v", err)
	}
	if err := tracer.LogError("run_1", 2, "err_1", "critical", "error-add"); err != nil {
		t.Fatalf("LogError: %v", err)
	}
	if err := tracer.LogPheromone("run_1", "FOCUS", "pheromone-write"); err != nil {
		t.Fatalf("LogPheromone: %v", err)
	}
	if err := tracer.LogIntervention("run_1", "discuss.resolved", "discuss", map[string]interface{}{"decisions": 3}); err != nil {
		t.Fatalf("LogIntervention: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dataDir, "trace.jsonl"))
	if err != nil {
		t.Fatalf("read trace.jsonl: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 5 {
		t.Fatalf("expected 5 lines, got %d", len(lines))
	}

	expectations := []struct {
		level string
		topic string
	}{
		{"state", "state.transition"},
		{"phase", "phase.start"},
		{"error", "error.add"},
		{"pheromone", "pheromone.write"},
		{"intervention", "discuss.resolved"},
	}
	for i, exp := range expectations {
		var e TraceEntry
		if err := json.Unmarshal([]byte(lines[i]), &e); err != nil {
			t.Fatalf("line %d unmarshal: %v", i, err)
		}
		if string(e.Level) != exp.level {
			t.Errorf("line %d level = %q, want %q", i, e.Level, exp.level)
		}
		if e.Topic != exp.topic {
			t.Errorf("line %d topic = %q, want %q", i, e.Topic, exp.topic)
		}
	}
}

func TestTracerLogNoPanicOnNilStore(t *testing.T) {
	tracer := NewTracer(nil)
	if err := tracer.Log(TraceEntry{RunID: "run_1"}); err == nil {
		t.Error("expected error when store is nil")
	}
}

func TestGenerateTraceID(t *testing.T) {
	id1 := generateTraceID()
	id2 := generateTraceID()
	if id1 == id2 {
		t.Error("expected unique trace IDs")
	}
	if !strings.HasPrefix(id1, "trc_") {
		t.Errorf("expected prefix trc_, got %q", id1)
	}
}
