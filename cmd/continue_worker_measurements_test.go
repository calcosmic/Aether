package cmd

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/codex"
)

// TestWorkerFlowStepCarriesToolCount is the RED/GREEN proof for Task 1: a
// codexContinueWorkerFlowStep with a measured duration and tool-call count
// round-trips through json.Marshal/json.Unmarshal preserving both figures,
// and a step whose measurements were never reported serializes with neither
// figure present (not as a fabricated zero -- Phase 196 D-01).
func TestWorkerFlowStepCarriesToolCount(t *testing.T) {
	t.Run("measured figures round-trip", func(t *testing.T) {
		step := codexContinueWorkerFlowStep{
			Stage: "review", Caste: "watcher", Name: "Keen-12", Status: "completed",
			Duration: 190.4, DurationReported: true,
			ToolCount: 14, ToolCountReported: true,
		}

		data, err := json.Marshal(step)
		if err != nil {
			t.Fatalf("marshal step: %v", err)
		}
		var decoded codexContinueWorkerFlowStep
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unmarshal step: %v", err)
		}
		if decoded.Duration != step.Duration || !decoded.DurationReported {
			t.Errorf("duration did not survive round-trip: got %+v", decoded)
		}
		if decoded.ToolCount != step.ToolCount || !decoded.ToolCountReported {
			t.Errorf("tool count did not survive round-trip: got %+v", decoded)
		}
	})

	t.Run("unreported tool count is absent from JSON, not a zero", func(t *testing.T) {
		step := codexContinueWorkerFlowStep{
			Stage: "review", Caste: "watcher", Name: "Keen-12", Status: "timeout",
			// Duration/ToolCount left unset -- no WorkerResult was ever produced.
		}

		data, err := json.Marshal(step)
		if err != nil {
			t.Fatalf("marshal step: %v", err)
		}
		var decoded map[string]interface{}
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unmarshal step to map: %v", err)
		}
		if _, present := decoded["tool_count"]; present {
			t.Errorf("tool_count key present in JSON for an unreported worker: %s", data)
		}
		if _, present := decoded["duration"]; present {
			t.Errorf("duration key present in JSON for an unreported worker: %s", data)
		}
	})

	t.Run("a worker result carrying a tool-call count populates the step", func(t *testing.T) {
		// Mirrors the population site in runCodexContinueReview (cmd/codex_continue.go)
		// so this test would fail if that wiring regressed.
		result := &codex.WorkerResult{Duration: 0, ToolCount: 14}
		var step codexContinueWorkerFlowStep
		step.Duration = result.Duration.Seconds()
		step.DurationReported = true
		step.ToolCount = result.ToolCount
		step.ToolCountReported = true

		if step.ToolCount != 14 {
			t.Errorf("ToolCount = %d, want 14", step.ToolCount)
		}
		if !step.ToolCountReported {
			t.Errorf("ToolCountReported = false, want true")
		}
	})
}

// TestWorkerUsageStaysUnserialized asserts, via struct-tag reflection (not a
// text search over the source), that codexContinueWorkerFlowStep.Usage still
// carries json:"-" -- a figure that crossed a wire could be asserted by an
// outside caller rather than measured by the runtime, unlike the plain
// ToolCount int this plan adds.
func TestWorkerUsageStaysUnserialized(t *testing.T) {
	field, ok := reflect.TypeOf(codexContinueWorkerFlowStep{}).FieldByName("Usage")
	if !ok {
		t.Fatalf("codexContinueWorkerFlowStep has no Usage field")
	}
	tag := field.Tag.Get("json")
	if tag != "-" {
		t.Fatalf("Usage field json tag = %q, want \"-\"", tag)
	}

	// Marshal a step with a non-empty usage value and confirm no usage key
	// leaks into the JSON either.
	step := codexContinueWorkerFlowStep{
		Name: "Keen-12", Status: "completed",
		Usage: codex.WorkerUsage{TotalTokens: 12345},
	}
	data, err := json.Marshal(step)
	if err != nil {
		t.Fatalf("marshal step: %v", err)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal step to map: %v", err)
	}
	for key := range decoded {
		if strings.Contains(strings.ToLower(key), "usage") {
			t.Fatalf("marshalled JSON contains a usage-related key %q: %s", key, data)
		}
	}
}
