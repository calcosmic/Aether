package cmd

import (
	"bytes"
	"strings"
	"testing"
)

// TestRenderActionsNeeded_EscalatedWaveWorkers verifies that when wave summary
// has escalated workers, the actions-needed section is non-empty and contains
// the wave number and escalation info.
func TestRenderActionsNeeded_EscalatedWaveWorkers(t *testing.T) {
	summary := WaveLifecycleSummary{
		Phase:          1,
		TotalEscalated: 2,
		Waves: []WaveResult{
			{Wave: 1, Escalated: 2},
		},
	}
	recoveryLog := RecoveryLogFile{Phase: 1}

	result := renderActionsNeeded(summary, recoveryLog)

	if result == "" {
		t.Fatal("expected non-empty result when wave has escalated workers")
	}
	if !strings.Contains(result, "Actions Needed") {
		t.Error("expected 'Actions Needed' stage marker in output")
	}
	if !strings.Contains(result, "Wave 1") {
		t.Error("expected 'Wave 1' in output")
	}
	if !strings.Contains(result, "escalated") {
		t.Error("expected 'escalated' in output")
	}
}

// TestRenderActionsNeeded_EscalatedRecoveryEntries verifies that recovery-log
// entries with action=escalate produce actions-needed items with worker names.
func TestRenderActionsNeeded_EscalatedRecoveryEntries(t *testing.T) {
	summary := WaveLifecycleSummary{
		Phase:          2,
		TotalEscalated: 0,
		Waves:          []WaveResult{},
	}
	recoveryLog := RecoveryLogFile{
		Phase: 2,
		Entries: []RecoveryLogEntry{
			{
				ID: "rl-esc-1",
				Failure: FailureRecord{
					WorkerName:   "builder-mason",
					Phase:        2,
					Status:       "bad_task_spec",
					ErrorMessage: "invalid task definition",
				},
				ActionTaken: "escalate",
				Outcome:     "blocking failure",
				Timestamp:   "2025-01-01T00:00:00Z",
			},
		},
	}

	result := renderActionsNeeded(summary, recoveryLog)

	if result == "" {
		t.Fatal("expected non-empty result when recovery log has escalate entries")
	}
	if !strings.Contains(result, "Actions Needed") {
		t.Error("expected 'Actions Needed' stage marker in output")
	}
	if !strings.Contains(result, "builder-mason") {
		t.Error("expected worker name 'builder-mason' in output")
	}
	if !strings.Contains(result, "blocking failure") {
		t.Error("expected outcome 'blocking failure' in output")
	}
}

// TestRenderActionsNeeded_ZeroItemsReturnsEmpty verifies that when zero items
// need attention (clean build), the function returns empty string per D-10.
func TestRenderActionsNeeded_ZeroItemsReturnsEmpty(t *testing.T) {
	summary := WaveLifecycleSummary{
		Phase:          3,
		TotalEscalated: 0,
		Waves: []WaveResult{
			{Wave: 1, Escalated: 0},
		},
	}
	recoveryLog := RecoveryLogFile{
		Phase:   3,
		Entries: []RecoveryLogEntry{},
	}

	result := renderActionsNeeded(summary, recoveryLog)

	if result != "" {
		t.Errorf("expected empty string for clean build, got: %q", result)
	}
}

// TestRenderActionsNeeded_BothSources verifies that when both wave-summary
// escalated workers AND recovery-log escalate entries exist, all items are listed.
func TestRenderActionsNeeded_BothSources(t *testing.T) {
	summary := WaveLifecycleSummary{
		Phase:          4,
		TotalEscalated: 3,
		Waves: []WaveResult{
			{Wave: 1, Escalated: 1},
			{Wave: 2, Escalated: 2},
		},
	}
	recoveryLog := RecoveryLogFile{
		Phase: 4,
		Entries: []RecoveryLogEntry{
			{
				ID: "rl-both-1",
				Failure: FailureRecord{
					WorkerName:   "watcher-sentinel",
					Phase:        4,
					Status:       "structural_error",
					ErrorMessage: "code structure error",
				},
				ActionTaken: "escalate",
				Outcome:     "unrecoverable",
				Timestamp:   "2025-01-01T00:00:00Z",
			},
		},
	}

	result := renderActionsNeeded(summary, recoveryLog)

	if result == "" {
		t.Fatal("expected non-empty result when both sources have items")
	}
	// Should contain wave escalation items
	if !strings.Contains(result, "Wave 1") {
		t.Error("expected 'Wave 1' escalation item")
	}
	if !strings.Contains(result, "Wave 2") {
		t.Error("expected 'Wave 2' escalation item")
	}
	// Should contain recovery log escalation item
	if !strings.Contains(result, "watcher-sentinel") {
		t.Error("expected 'watcher-sentinel' from recovery log")
	}
}

// TestRenderActionsNeeded_NonEscalatedRecoveryEntriesIgnored verifies that
// recovery-log entries with action != "escalate" are not included.
func TestRenderActionsNeeded_NonEscalatedRecoveryEntriesIgnored(t *testing.T) {
	summary := WaveLifecycleSummary{
		Phase:          5,
		TotalEscalated: 0,
		Waves:          []WaveResult{},
	}
	recoveryLog := RecoveryLogFile{
		Phase: 5,
		Entries: []RecoveryLogEntry{
			{
				ID: "rl-retry",
				Failure: FailureRecord{
					WorkerName: "builder-stone",
					Phase:      5,
					Status:     "timeout",
				},
				ActionTaken: "retry",
				Outcome:     "success",
				Timestamp:   "2025-01-01T00:00:00Z",
			},
		},
	}

	result := renderActionsNeeded(summary, recoveryLog)

	if result != "" {
		t.Errorf("expected empty string when only non-escalate recovery entries exist, got: %q", result)
	}
}

// TestRenderPhaseEndSummary_WithItems writes to stdout when actions are needed.
func TestRenderPhaseEndSummary_WithItems(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	setupBuildFlowTest(t)
	const phase = 1

	// Write recovery-log with escalate entry
	rl := RecoveryLogFile{
		Phase: phase,
		Entries: []RecoveryLogEntry{
			{
				ID: "rl-phase-end-1",
				Failure: FailureRecord{
					WorkerName:   "builder-test",
					Phase:        phase,
					Status:       "bad_task_spec",
					ErrorMessage: "task spec invalid",
				},
				ActionTaken: "escalate",
				Outcome:     "blocking",
				Timestamp:   "2025-01-01T00:00:00Z",
			},
		},
	}
	if err := recoveryLogWritePhase(phase, rl.Entries); err != nil {
		t.Fatalf("failed to write recovery log: %v", err)
	}

	summary := WaveLifecycleSummary{
		Phase:          phase,
		TotalEscalated: 0,
		Waves:          []WaveResult{},
	}

	renderPhaseEndSummary(summary, phase)

	output := stdout.(*bytes.Buffer).String()
	if !strings.Contains(output, "Actions Needed") {
		t.Errorf("expected 'Actions Needed' in stdout, got: %q", output)
	}
	if !strings.Contains(output, "builder-test") {
		t.Errorf("expected 'builder-test' in stdout, got: %q", output)
	}
}

// TestRenderPhaseEndSummary_ZeroItems writes nothing extra for clean builds.
func TestRenderPhaseEndSummary_ZeroItems(t *testing.T) {
	setupBuildFlowTest(t)
	const phase = 2

	summary := WaveLifecycleSummary{
		Phase:          phase,
		TotalEscalated: 0,
		Waves: []WaveResult{
			{Wave: 1, Escalated: 0},
		},
	}

	renderPhaseEndSummary(summary, phase)

	output := stdout.(*bytes.Buffer).String()
	if strings.Contains(output, "Actions Needed") {
		t.Errorf("expected no 'Actions Needed' for clean build, got: %q", output)
	}
}

// TestRenderActionsNeeded_UsesStageMarker verifies that the output uses the
// existing renderStageMarker pattern per D-11.
func TestRenderActionsNeeded_UsesStageMarker(t *testing.T) {
	summary := WaveLifecycleSummary{
		Phase:          6,
		TotalEscalated: 1,
		Waves: []WaveResult{
			{Wave: 1, Escalated: 1},
		},
	}
	recoveryLog := RecoveryLogFile{Phase: 6}

	result := renderActionsNeeded(summary, recoveryLog)

	expectedMarker := "── Actions Needed ──\n"
	if !strings.Contains(result, expectedMarker) {
		t.Errorf("expected stage marker %q in output, got: %q", expectedMarker, result)
	}
}
