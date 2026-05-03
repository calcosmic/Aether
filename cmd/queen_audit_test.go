package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestConsolidateQueenAudit_AllSourcesPresent verifies that when all three source
// files (queen-state, recovery-log, wave-summary) exist, the audit contains
// entries from all three sources.
func TestConsolidateQueenAudit_AllSourcesPresent(t *testing.T) {
	setupBuildFlowTest(t)
	const phase = 1

	// Write queen-state-{N}.json with gate decisions and escalation
	qs := QueenStateFile{
		Phase:       phase,
		GeneratedAt: "2025-01-01T00:00:00Z",
		Decisions: []QueenDecision{
			{
				GateName:            "test-gate",
				Status:              "failed",
				ClassificationTier:  "soft_block",
				QueenRecommendation: "auto-resolve",
				Rationale:           "soft block with budget remaining",
			},
		},
		EscalationLog: []EscalationEntry{
			{
				Timestamp:        "2025-01-01T00:01:00Z",
				BreakerTripped:   []string{"worker-1"},
				EscalationAction: "escalate_to_human",
				Rationale:        "breaker tripped",
			},
		},
	}
	if err := queenStateWrite(phase, qs); err != nil {
		t.Fatalf("failed to write queen state: %v", err)
	}

	// Write recovery-log-{N}.json with recovery actions
	rl := RecoveryLogFile{
		Phase: phase,
		Entries: []RecoveryLogEntry{
			{
				ID: "rl-001",
				Failure: FailureRecord{
					WorkerName:   "worker-2",
					TaskID:       "task-2",
					Phase:        phase,
					Status:       "timeout",
					ErrorMessage: "context deadline exceeded",
					Timestamp:    "2025-01-01T00:02:00Z",
				},
				ActionTaken:   "retry",
				Outcome:       "success",
				AttemptNumber: 1,
				Timestamp:     "2025-01-01T00:02:00Z",
				Detail:        "recoverable: retrying worker",
			},
			{
				ID: "rl-002",
				Failure: FailureRecord{
					WorkerName:   "worker-3",
					TaskID:       "task-3",
					Phase:        phase,
					Status:       "bad_task_spec",
					ErrorMessage: "invalid task definition",
					Timestamp:    "2025-01-01T00:03:00Z",
				},
				ActionTaken:   "escalate",
				Outcome:       "blocking failure",
				AttemptNumber: 0,
				Timestamp:     "2025-01-01T00:03:00Z",
				Detail:        "blocking failure -- no recovery attempted",
			},
		},
	}
	if err := recoveryLogWritePhase(phase, rl.Entries); err != nil {
		t.Fatalf("failed to write recovery log: %v", err)
	}

	// Write wave-summary-{N}.json with recovery entries and escalation
	ws := WaveLifecycleSummary{
		Phase:       phase,
		TotalWaves:  1,
		CompletedAt: "2025-01-01T00:10:00Z",
		Waves: []WaveResult{
			{
				Wave:      1,
				Recovered: []RecoveryEntry{{WorkerName: "worker-4", Method: "retry", Detail: "wave retry"}},
				Escalated: 2,
			},
		},
	}
	if err := writeWaveSummary(phase, ws); err != nil {
		t.Fatalf("failed to write wave summary: %v", err)
	}

	audit := consolidateQueenAudit(phase)

	// Should have entries from all three sources:
	// 1 gate_evaluate + 1 escalation (queen-state) + 2 recovery + 1 wave_advance + 1 escalation (wave) = 6
	if len(audit.Decisions) < 4 {
		t.Fatalf("expected at least 4 audit entries, got %d", len(audit.Decisions))
	}

	// Verify we have entries from queen-state (gate_evaluate)
	foundGate := false
	for _, d := range audit.Decisions {
		if d.DecisionType == "gate_evaluate" {
			foundGate = true
			if !strings.Contains(d.InputFinding, "test-gate") {
				t.Errorf("gate_evaluate entry missing gate name: %s", d.InputFinding)
			}
		}
	}
	if !foundGate {
		t.Error("expected gate_evaluate entry from queen-state source")
	}

	// Verify we have entries from recovery-log (recovery_action)
	foundRecovery := false
	for _, d := range audit.Decisions {
		if d.DecisionType == "recovery_action" {
			foundRecovery = true
			if !strings.Contains(d.InputFinding, "worker-2") {
				t.Errorf("recovery_action entry missing worker name: %s", d.InputFinding)
			}
		}
	}
	if !foundRecovery {
		t.Error("expected recovery_action entry from recovery-log source")
	}

	// Verify escalation from recovery-log (action_taken == "escalate")
	foundEscalation := false
	for _, d := range audit.Decisions {
		if d.DecisionType == "escalation" && strings.Contains(d.InputFinding, "worker-3") {
			foundEscalation = true
		}
	}
	if !foundEscalation {
		t.Error("expected escalation entry from recovery-log source")
	}

	// Verify wave_advance entry
	foundWave := false
	for _, d := range audit.Decisions {
		if d.DecisionType == "wave_advance" {
			foundWave = true
			if !strings.Contains(d.InputFinding, "worker-4") {
				t.Errorf("wave_advance entry missing worker name: %s", d.InputFinding)
			}
		}
	}
	if !foundWave {
		t.Error("expected wave_advance entry from wave-summary source")
	}

	// Verify phase is set
	if audit.Phase != phase {
		t.Errorf("audit phase = %d, want %d", audit.Phase, phase)
	}
}

// TestConsolidateQueenAudit_MissingRecoveryLog verifies that missing recovery-log
// still produces valid audit from queen-state and wave-summary.
func TestConsolidateQueenAudit_MissingRecoveryLog(t *testing.T) {
	setupBuildFlowTest(t)
	const phase = 2

	qs := QueenStateFile{
		Phase:       phase,
		GeneratedAt: "2025-01-01T00:00:00Z",
		Decisions: []QueenDecision{
			{
				GateName:            "gate-a",
				Status:              "passed",
				QueenRecommendation: "pass",
				Rationale:           "gate passed",
			},
		},
	}
	if err := queenStateWrite(phase, qs); err != nil {
		t.Fatalf("failed to write queen state: %v", err)
	}

	ws := WaveLifecycleSummary{
		Phase:       phase,
		TotalWaves:  1,
		CompletedAt: "2025-01-01T00:10:00Z",
		Waves: []WaveResult{
			{
				Wave:      1,
				Recovered: []RecoveryEntry{{WorkerName: "w1", Method: "peer_reassignment"}},
			},
		},
	}
	if err := writeWaveSummary(phase, ws); err != nil {
		t.Fatalf("failed to write wave summary: %v", err)
	}

	audit := consolidateQueenAudit(phase)

	if len(audit.Decisions) < 2 {
		t.Fatalf("expected at least 2 audit entries (queen-state + wave-summary), got %d", len(audit.Decisions))
	}

	// No recovery_action or fixer_dispatch entries should appear
	for _, d := range audit.Decisions {
		if d.DecisionType == "recovery_action" {
			t.Error("unexpected recovery_action entry when recovery-log is missing")
		}
	}
}

// TestConsolidateQueenAudit_MissingQueenState verifies that missing queen-state
// still produces valid audit from recovery-log and wave-summary.
func TestConsolidateQueenAudit_MissingQueenState(t *testing.T) {
	setupBuildFlowTest(t)
	const phase = 3

	rl := RecoveryLogFile{
		Phase: phase,
		Entries: []RecoveryLogEntry{
			{
				ID: "rl-003",
				Failure: FailureRecord{
					WorkerName: "worker-x",
					Phase:      phase,
					Status:     "failed",
				},
				ActionTaken: "fixer_dispatch",
				Timestamp:   "2025-01-01T00:05:00Z",
				Detail:      "dispatching fixer",
			},
		},
	}
	if err := recoveryLogWritePhase(phase, rl.Entries); err != nil {
		t.Fatalf("failed to write recovery log: %v", err)
	}

	ws := WaveLifecycleSummary{
		Phase:       phase,
		TotalWaves:  1,
		CompletedAt: "2025-01-01T00:10:00Z",
		Waves: []WaveResult{
			{
				Wave:      1,
				Escalated: 1,
			},
		},
	}
	if err := writeWaveSummary(phase, ws); err != nil {
		t.Fatalf("failed to write wave summary: %v", err)
	}

	audit := consolidateQueenAudit(phase)

	if len(audit.Decisions) < 1 {
		t.Fatalf("expected at least 1 audit entry, got %d", len(audit.Decisions))
	}

	// No gate_evaluate entries should appear
	for _, d := range audit.Decisions {
		if d.DecisionType == "gate_evaluate" {
			t.Error("unexpected gate_evaluate entry when queen-state is missing")
		}
	}

	// Should have fixer_dispatch type
	foundFixer := false
	for _, d := range audit.Decisions {
		if d.DecisionType == "fixer_dispatch" {
			foundFixer = true
		}
	}
	if !foundFixer {
		t.Error("expected fixer_dispatch entry from recovery-log")
	}
}

// TestConsolidateQueenAudit_AllMissing verifies that when all source files are
// missing, the audit is valid with zero decisions (no error, no panic).
func TestConsolidateQueenAudit_AllMissing(t *testing.T) {
	setupBuildFlowTest(t)
	const phase = 99

	audit := consolidateQueenAudit(phase)

	if audit.Phase != phase {
		t.Errorf("audit phase = %d, want %d", audit.Phase, phase)
	}
	if audit.GeneratedAt == "" {
		t.Error("audit generated_at should not be empty")
	}
	if len(audit.Decisions) != 0 {
		t.Errorf("expected 0 decisions when all sources missing, got %d", len(audit.Decisions))
	}
}

// TestAuditSchema_D06 verifies each entry has all 5 D-06 fields populated.
// Rationale may be empty for some entries, but the other 4 must be non-empty.
func TestAuditSchema_D06(t *testing.T) {
	setupBuildFlowTest(t)
	const phase = 4

	qs := QueenStateFile{
		Phase:       phase,
		GeneratedAt: "2025-01-01T00:00:00Z",
		Decisions: []QueenDecision{
			{
				GateName:            "schema-gate",
				Status:              "failed",
				ClassificationTier:  "hard_block",
				QueenRecommendation: "escalate",
				Rationale:           "hard block requires escalation",
			},
		},
	}
	if err := queenStateWrite(phase, qs); err != nil {
		t.Fatalf("failed to write queen state: %v", err)
	}

	rl := RecoveryLogFile{
		Phase: phase,
		Entries: []RecoveryLogEntry{
			{
				ID: "rl-schema",
				Failure: FailureRecord{
					WorkerName:   "worker-schema",
					TaskID:       "task-schema",
					Phase:        phase,
					Status:       "timeout",
					ErrorMessage: "timeout after 30s",
				},
				ActionTaken: "retry",
				Outcome:     "success",
				Timestamp:   "2025-01-01T00:02:00Z",
				Detail:      "recoverable timeout",
			},
		},
	}
	if err := recoveryLogWritePhase(phase, rl.Entries); err != nil {
		t.Fatalf("failed to write recovery log: %v", err)
	}

	ws := WaveLifecycleSummary{
		Phase:       phase,
		TotalWaves:  1,
		CompletedAt: "2025-01-01T00:10:00Z",
		Waves: []WaveResult{
			{
				Wave:      1,
				Recovered: []RecoveryEntry{{WorkerName: "worker-schema", Method: "retry", Detail: "recovered"}},
			},
		},
	}
	if err := writeWaveSummary(phase, ws); err != nil {
		t.Fatalf("failed to write wave summary: %v", err)
	}

	audit := consolidateQueenAudit(phase)

	if len(audit.Decisions) == 0 {
		t.Fatal("expected audit entries for schema validation")
	}

	for i, entry := range audit.Decisions {
		if entry.Timestamp == "" {
			t.Errorf("entry %d: timestamp is empty", i)
		}
		if entry.DecisionType == "" {
			t.Errorf("entry %d: decision_type is empty", i)
		}
		if entry.InputFinding == "" {
			t.Errorf("entry %d: input_finding is empty", i)
		}
		if entry.ActionTaken == "" {
			t.Errorf("entry %d: action_taken is empty", i)
		}
		// Rationale is the only field that may legitimately be empty
	}
}

// TestWriteAuditFile_ReadAuditFile_Roundtrip verifies write then read preserves
// all fields correctly.
func TestWriteAuditFile_ReadAuditFile_Roundtrip(t *testing.T) {
	setupBuildFlowTest(t)
	const phase = 5

	original := QueenAuditFile{
		Phase:       phase,
		GeneratedAt: "2025-06-15T12:00:00Z",
		Decisions: []QueenAuditEntry{
			{
				Timestamp:    "2025-06-15T12:00:00Z",
				DecisionType: "gate_evaluate",
				InputFinding: "Gate test: status=failed, tier=hard_block",
				ActionTaken:  "escalate",
				Rationale:    "hard block requires escalation",
			},
			{
				Timestamp:    "2025-06-15T12:01:00Z",
				DecisionType: "recovery_action",
				InputFinding: "Worker w1 (task t1): timeout",
				ActionTaken:  "retry",
				Rationale:    "recoverable timeout",
			},
		},
	}

	if err := writeAuditFile(phase, original); err != nil {
		t.Fatalf("writeAuditFile failed: %v", err)
	}

	readBack, err := readAuditFile(phase)
	if err != nil {
		t.Fatalf("readAuditFile failed: %v", err)
	}

	if readBack.Phase != original.Phase {
		t.Errorf("phase mismatch: got %d, want %d", readBack.Phase, original.Phase)
	}
	if readBack.GeneratedAt != original.GeneratedAt {
		t.Errorf("generated_at mismatch: got %s, want %s", readBack.GeneratedAt, original.GeneratedAt)
	}
	if len(readBack.Decisions) != len(original.Decisions) {
		t.Fatalf("decisions count mismatch: got %d, want %d", len(readBack.Decisions), len(original.Decisions))
	}

	for i, entry := range readBack.Decisions {
		orig := original.Decisions[i]
		if entry.Timestamp != orig.Timestamp {
			t.Errorf("entry %d timestamp mismatch: got %s, want %s", i, entry.Timestamp, orig.Timestamp)
		}
		if entry.DecisionType != orig.DecisionType {
			t.Errorf("entry %d decision_type mismatch: got %s, want %s", i, entry.DecisionType, orig.DecisionType)
		}
		if entry.InputFinding != orig.InputFinding {
			t.Errorf("entry %d input_finding mismatch: got %s, want %s", i, entry.InputFinding, orig.InputFinding)
		}
		if entry.ActionTaken != orig.ActionTaken {
			t.Errorf("entry %d action_taken mismatch: got %s, want %s", i, entry.ActionTaken, orig.ActionTaken)
		}
		if entry.Rationale != orig.Rationale {
			t.Errorf("entry %d rationale mismatch: got %s, want %s", i, entry.Rationale, orig.Rationale)
		}
	}
}

// TestWriteAuditFile_CorrectPath verifies the audit file is written to the
// expected queen-audit-{N}.json path.
func TestWriteAuditFile_CorrectPath(t *testing.T) {
	dataDir := setupBuildFlowTest(t)
	const phase = 6

	audit := QueenAuditFile{
		Phase:       phase,
		GeneratedAt: "2025-01-01T00:00:00Z",
		Decisions:   []QueenAuditEntry{},
	}

	if err := writeAuditFile(phase, audit); err != nil {
		t.Fatalf("writeAuditFile failed: %v", err)
	}

	expectedPath := filepath.Join(dataDir, "queen-audit-6.json")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("audit file not found at %s", expectedPath)
	}
}

// TestConsolidateQueenAudit_SourceFilesNotModified verifies that consolidation
// is read-only -- source files remain identical after consolidation.
func TestConsolidateQueenAudit_SourceFilesNotModified(t *testing.T) {
	setupBuildFlowTest(t)
	const phase = 7

	// Write all three source files
	qs := QueenStateFile{
		Phase:       phase,
		GeneratedAt: "2025-01-01T00:00:00Z",
		Decisions: []QueenDecision{
			{
				GateName:            "ro-gate",
				Status:              "passed",
				QueenRecommendation: "pass",
				Rationale:           "gate passed",
			},
		},
	}
	if err := queenStateWrite(phase, qs); err != nil {
		t.Fatalf("failed to write queen state: %v", err)
	}

	rl := RecoveryLogFile{
		Phase: phase,
		Entries: []RecoveryLogEntry{
			{
				ID: "rl-ro",
				Failure: FailureRecord{
					WorkerName: "worker-ro",
					Phase:      phase,
					Status:     "timeout",
				},
				ActionTaken: "retry",
				Timestamp:   "2025-01-01T00:02:00Z",
			},
		},
	}
	if err := recoveryLogWritePhase(phase, rl.Entries); err != nil {
		t.Fatalf("failed to write recovery log: %v", err)
	}

	ws := WaveLifecycleSummary{
		Phase:       phase,
		TotalWaves:  1,
		CompletedAt: "2025-01-01T00:10:00Z",
		Waves: []WaveResult{
			{
				Wave:      1,
				Recovered: []RecoveryEntry{{WorkerName: "w-ro", Method: "retry"}},
			},
		},
	}
	if err := writeWaveSummary(phase, ws); err != nil {
		t.Fatalf("failed to write wave summary: %v", err)
	}

	// Read source files before consolidation
	qsBefore, _ := json.MarshalIndent(qs, "", "  ")
	rlBefore, _ := json.MarshalIndent(rl, "", "  ")
	wsBefore, _ := json.MarshalIndent(ws, "", "  ")

	// Run consolidation
	_ = consolidateQueenAudit(phase)

	// Read source files after consolidation
	qsAfter, _ := json.MarshalIndent(qs, "", "  ")
	rlAfter, _ := json.MarshalIndent(rl, "", "  ")
	wsAfter, _ := json.MarshalIndent(ws, "", "  ")

	if string(qsBefore) != string(qsAfter) {
		t.Error("queen-state file was modified during consolidation")
	}
	if string(rlBefore) != string(rlAfter) {
		t.Error("recovery-log file was modified during consolidation")
	}
	if string(wsBefore) != string(wsAfter) {
		t.Error("wave-summary file was modified during consolidation")
	}
}

// TestConsolidateQueenAudit_RecoveryLogDecisionTypes verifies that recovery-log
// entries are mapped to the correct decision types: retry->auto_resolve,
// fixer_dispatch->fixer_dispatch, escalate->escalation, default->recovery_action.
func TestConsolidateQueenAudit_RecoveryLogDecisionTypes(t *testing.T) {
	setupBuildFlowTest(t)
	const phase = 8

	now := time.Now().UTC().Format(time.RFC3339)

	rl := RecoveryLogFile{
		Phase: phase,
		Entries: []RecoveryLogEntry{
			{
				ID: "rl-retry",
				Failure: FailureRecord{
					WorkerName: "w-retry",
					Phase:      phase,
					Status:     "timeout",
				},
				ActionTaken: "retry",
				Timestamp:   now,
				Detail:      "retrying after timeout",
			},
			{
				ID: "rl-fixer",
				Failure: FailureRecord{
					WorkerName: "w-fixer",
					Phase:      phase,
					Status:     "failed",
				},
				ActionTaken: "fixer_dispatch",
				Timestamp:   now,
				Detail:      "dispatching fixer agent",
			},
			{
				ID: "rl-escalate",
				Failure: FailureRecord{
					WorkerName: "w-escalate",
					Phase:      phase,
					Status:     "bad_task_spec",
				},
				ActionTaken: "escalate",
				Timestamp:   now,
				Detail:      "blocking failure",
			},
			{
				ID: "rl-default",
				Failure: FailureRecord{
					WorkerName: "w-default",
					Phase:      phase,
					Status:     "failed",
				},
				ActionTaken: "peer_reassignment",
				Timestamp:   now,
				Detail:      "reassigning to peer",
			},
		},
	}
	if err := recoveryLogWritePhase(phase, rl.Entries); err != nil {
		t.Fatalf("failed to write recovery log: %v", err)
	}

	audit := consolidateQueenAudit(phase)

	typeMap := map[string]string{
		"w-retry":    "auto_resolve",
		"w-fixer":    "fixer_dispatch",
		"w-escalate": "escalation",
		"w-default":  "recovery_action",
	}

	for worker, expectedType := range typeMap {
		found := false
		for _, d := range audit.Decisions {
			if strings.Contains(d.InputFinding, worker) {
				found = true
				if d.DecisionType != expectedType {
					t.Errorf("worker %s: decision_type = %q, want %q", worker, d.DecisionType, expectedType)
				}
				break
			}
		}
		if !found {
			t.Errorf("no audit entry found for worker %s", worker)
		}
	}
}
