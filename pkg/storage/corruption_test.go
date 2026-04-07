package storage

import (
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// TestDetectCorruption_RejectsAssignmentPattern verifies that DetectCorruption
// rejects Events containing jq assignment patterns like ".path = value".
func TestDetectCorruption_RejectsAssignmentPattern(t *testing.T) {
	state := &colony.ColonyState{
		Events: []string{".current_phase = 5"},
	}
	err := DetectCorruption(state)
	if err == nil {
		t.Fatal("expected error for jq assignment pattern, got nil")
	}
	if !strings.Contains(err.Error(), "jq expression") {
		t.Errorf("error should mention jq expression, got: %v", err)
	}
}

// TestDetectCorruption_RejectsUpdatePattern verifies that DetectCorruption
// rejects Events containing jq update patterns like ".path |= expr".
func TestDetectCorruption_RejectsUpdatePattern(t *testing.T) {
	state := &colony.ColonyState{
		Events: []string{".state |= \"EXECUTING\""},
	}
	err := DetectCorruption(state)
	if err == nil {
		t.Fatal("expected error for jq update pattern, got nil")
	}
}

// TestDetectCorruption_CleanState verifies that DetectCorruption returns nil
// for clean state with normal event strings.
func TestDetectCorruption_CleanState(t *testing.T) {
	state := &colony.ColonyState{
		Events: []string{
			"Phase 1 completed successfully",
			"Build started for phase 2",
			"Worker ant-1 finished task 3",
		},
	}
	err := DetectCorruption(state)
	if err != nil {
		t.Fatalf("expected nil for clean state, got: %v", err)
	}
}

// TestDetectCorruption_EmptyEvents verifies that DetectCorruption returns nil
// for an empty Events slice.
func TestDetectCorruption_EmptyEvents(t *testing.T) {
	state := &colony.ColonyState{
		Events: []string{},
	}
	err := DetectCorruption(state)
	if err != nil {
		t.Fatalf("expected nil for empty events, got: %v", err)
	}
}

// TestDetectCorruption_ErrorIncludesFieldInfo verifies that the error message
// includes the offending value and index for debugging.
func TestDetectCorruption_ErrorIncludesFieldInfo(t *testing.T) {
	offending := `.plan.phases[0].status = "completed"`
	state := &colony.ColonyState{
		Events: []string{"safe event", offending, "another safe event"},
	}
	err := DetectCorruption(state)
	if err == nil {
		t.Fatal("expected error for corrupted state, got nil")
	}
	// Should reference the events index
	if !strings.Contains(err.Error(), "events[1]") {
		t.Errorf("error should reference events index, got: %v", err)
	}
	// Should include the jq path portion
	if !strings.Contains(err.Error(), ".plan.phases[0].status") {
		t.Errorf("error should include the jq path, got: %v", err)
	}
	// Should mention it's a jq expression
	if !strings.Contains(err.Error(), "jq expression") {
		t.Errorf("error should mention jq expression, got: %v", err)
	}
}

// TestDetectCorruption_RejectsJQOperators verifies that DetectCorruption
// catches suspicious jq operators like |=, | select(, | map(.
func TestDetectCorruption_RejectsJQOperators(t *testing.T) {
	tests := []struct {
		name  string
		event string
	}{
		{"pipe-select", "| select(.status == \"done\")"},
		{"pipe-map", "| map(.id)"},
		{"update-operator", ".memory.phase_learnings |= . + [{...}]"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			state := &colony.ColonyState{
				Events: []string{tc.event},
			}
			err := DetectCorruption(state)
			if err == nil {
				t.Errorf("expected error for %q, got nil", tc.event)
			}
		})
	}
}

// TestDetectCorruption_NilEvents verifies that DetectCorruption handles nil Events.
func TestDetectCorruption_NilEvents(t *testing.T) {
	state := &colony.ColonyState{
		Events: nil,
	}
	err := DetectCorruption(state)
	if err != nil {
		t.Fatalf("expected nil for nil events, got: %v", err)
	}
}
