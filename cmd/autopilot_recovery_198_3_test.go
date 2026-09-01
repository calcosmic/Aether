package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

func finishRecoveryReportForTest(t *testing.T, state colony.ColonyState, decision autopilotRunDecision) *autopilotInvocationReport {
	t.Helper()
	started := time.Date(2026, 8, 31, 21, 0, 0, 0, time.UTC)
	originalNow := autopilotNow
	autopilotNow = func() time.Time { return started.Add(3 * time.Minute) }
	t.Cleanup(func() { autopilotNow = originalNow })
	invocation := autopilotInvocation{
		ID:             "run-recovery-idempotent",
		StartedAt:      started,
		BlockersBefore: captureAutopilotBlockerSnapshot(store),
		Phases:         []autopilotPhaseReport{{Phase: state.CurrentPhase, PhaseName: "Recovery phase", Outcome: "checking"}},
	}
	result := finishAutopilotInvocation(&invocation, state, runCompatibilityOptions{}, nil, 0, decision, nil)
	report, ok := autopilotReportFromValue(result["last_report"])
	if !ok {
		t.Fatalf("finalizer returned no typed report: %+v", result)
	}
	return report
}

func TestRunGenuineStopRecoveryIsClassifiedOnceAndIdempotent(t *testing.T) {
	genuine := []autopilotTriggerCode{
		autopilotTriggerDeterministicVerificationFailed,
		autopilotTriggerAuditorScoreBelowFloor,
		autopilotTriggerCriticalReviewFinding,
		autopilotTriggerBlockerCountIncreased,
		autopilotTriggerBlockerEscalated,
		autopilotTriggerColonyNotRunnable,
		autopilotTriggerProviderUnavailable,
	}
	for _, code := range genuine {
		t.Run(string(code), func(t *testing.T) {
			setupSpendTestStore(t)
			state := reportTestState(colony.StateBUILT, 2)
			decision := autopilotRunDecisionForCode(code, false, map[string]interface{}{
				"phase": 2, "evidence_marker": "typed-current-stop",
			})

			first := finishRecoveryReportForTest(t, state, decision)
			second := finishRecoveryReportForTest(t, state, decision)
			if first.Recovery == nil || second.Recovery == nil {
				t.Fatalf("genuine stop has no recovery projection: first=%+v second=%+v", first.Recovery, second.Recovery)
			}
			if first.Recovery.EntryID != second.Recovery.EntryID {
				t.Fatalf("replayed finalization changed recovery ID: %q -> %q", first.Recovery.EntryID, second.Recovery.EntryID)
			}
			log, err := recoveryLogReadPhase(2)
			if err != nil {
				t.Fatalf("read recovery log: %v", err)
			}
			if len(log.Entries) != 1 {
				t.Fatalf("recovery entries = %d, want exactly one after replay: %+v", len(log.Entries), log.Entries)
			}
			entry := log.Entries[0]
			if entry.ID != first.Recovery.EntryID || entry.Failure.Classification == "" || entry.Failure.FailureType == "" {
				t.Fatalf("unclassified recovery entry: %+v", entry)
			}
			if !strings.Contains(entry.Failure.ErrorMessage, string(code)) || !strings.Contains(entry.Failure.ErrorMessage, "typed-current-stop") {
				t.Fatalf("recovery entry lost trigger evidence: %+v", entry.Failure)
			}
		})
	}
}

func TestRunNormalAndQueueableStopsCreateNoRecovery(t *testing.T) {
	tests := []struct {
		name     string
		decision autopilotRunDecision
	}{
		{name: "runtime queued", decision: autopilotRunDecisionForCode(autopilotTriggerRuntimeVerificationNeeded, true, nil)},
		{name: "visual queued", decision: autopilotRunDecisionForCode(autopilotTriggerVisualCheckpointNeeded, true, nil)},
		{name: "replan queued", decision: autopilotRunDecisionForCode(autopilotTriggerReplanDue, true, nil)},
		{name: "interactive pause", decision: autopilotRunDecisionForCode(autopilotTriggerRuntimeVerificationNeeded, false, nil)},
		{name: "cancelled", decision: autopilotRunDecisionForCode(autopilotTriggerCancelled, false, nil)},
		{name: "timeout", decision: autopilotRunDecisionForCode(autopilotTriggerWorkerTimeout, false, nil)},
		{name: "max phases", decision: autopilotRunDecisionForCode(autopilotTriggerMaxPhasesReached, false, nil)},
		{name: "complete", decision: autopilotRunDecisionForCode(autopilotTriggerColonyComplete, false, nil)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setupSpendTestStore(t)
			state := reportTestState(colony.StateBUILT, 2)
			if tc.decision.Code == autopilotTriggerColonyComplete {
				state = reportTestState(colony.StateCOMPLETED, 3)
			}
			report := finishRecoveryReportForTest(t, state, tc.decision)
			if report.Recovery != nil {
				t.Fatalf("ordinary ending manufactured recovery: %+v", report.Recovery)
			}
			if _, err := recoveryLogReadPhase(state.CurrentPhase); err == nil {
				t.Fatal("ordinary ending wrote a recovery log")
			}
		})
	}
}

func TestAutopilotMedicAdviceUsesEligibilityAndNeverDispatches(t *testing.T) {
	tests := []struct {
		name       string
		seed       func(t *testing.T)
		wantAdvice bool
	}{
		{
			name: "eligible blocker and budget",
			seed: func(t *testing.T) {
				t.Helper()
				if err := store.SaveJSON("pending-decisions.json", PendingDecisionFile{Decisions: []PendingDecision{{
					ID: "blocker-1", Type: "blocker", Description: "needs health inspection",
				}}}); err != nil {
					t.Fatalf("seed blocker: %v", err)
				}
			},
			wantAdvice: true,
		},
		{name: "healthy colony", seed: func(t *testing.T) {}, wantAdvice: false},
		{
			name: "eligible signal but exhausted budget",
			seed: func(t *testing.T) {
				t.Helper()
				if err := store.SaveJSON("pending-decisions.json", PendingDecisionFile{Decisions: []PendingDecision{{ID: "blocker-2", Type: "blocker", Description: "blocked"}}}); err != nil {
					t.Fatalf("seed blocker: %v", err)
				}
				if err := store.SaveJSON("recovery-log-2.json", RecoveryLogFile{Phase: 2, RecoveryBudget: &RecoveryBudget{
					TotalBudget: 3, RetriesUsed: 1, ReassignsUsed: 1, FixerDispatchesUsed: 1, Wave: 1,
				}}); err != nil {
					t.Fatalf("seed exhausted budget: %v", err)
				}
			},
			wantAdvice: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setupSpendTestStore(t)
			tc.seed(t)
			dispatchFactoryCalls := 0
			originalFactory := newCodexWorkerInvoker
			newCodexWorkerInvoker = func() codex.WorkerInvoker {
				dispatchFactoryCalls++
				return originalFactory()
			}
			t.Cleanup(func() { newCodexWorkerInvoker = originalFactory })

			decision := autopilotRunDecisionForCode(autopilotTriggerDeterministicVerificationFailed, false, map[string]interface{}{"phase": 2})
			report := finishRecoveryReportForTest(t, reportTestState(colony.StateBUILT, 2), decision)
			if report.Recovery == nil {
				t.Fatal("genuine stop has no recovery report")
			}
			if tc.wantAdvice && report.Recovery.MedicAdvice != "aether medic --deep" {
				t.Fatalf("Medic advice = %q, want exact deep command", report.Recovery.MedicAdvice)
			}
			if !tc.wantAdvice && report.Recovery.MedicAdvice != "" {
				t.Fatalf("ineligible stop received Medic advice %q", report.Recovery.MedicAdvice)
			}
			if dispatchFactoryCalls != 0 {
				t.Fatalf("finalizer constructed %d Medic worker dispatcher(s), want advice only", dispatchFactoryCalls)
			}
		})
	}
}

func TestRunRecoveryLogFailureKeepsPrimaryStopReasonVisible(t *testing.T) {
	setupSpendTestStore(t)
	recoveryPath := filepath.Join(store.BasePath(), "recovery-log-2.json")
	if err := os.MkdirAll(recoveryPath, 0755); err != nil {
		t.Fatalf("create recovery write fault: %v", err)
	}
	decision := autopilotRunDecisionForCode(autopilotTriggerProviderUnavailable, false, map[string]interface{}{"phase": 2})
	report := finishRecoveryReportForTest(t, reportTestState(colony.StateBUILT, 2), decision)
	if report.StopReason != string(autopilotTriggerProviderUnavailable) {
		t.Fatalf("recovery failure replaced primary stop reason: %+v", report)
	}
	if report.Recovery == nil || strings.TrimSpace(report.Recovery.LogError) == "" {
		t.Fatalf("recovery write failure is not visible: %+v", report.Recovery)
	}
}

func TestRunRecoveryCallGraphUsesHelpersNotPublicCommands(t *testing.T) {
	for _, file := range []string{"autopilot_report.go", "compatibility_cmds.go", "recovery_classify.go", "medic_auto_spawn.go"} {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		body := string(data)
		if file == "autopilot_report.go" {
			for _, want := range []string{"recordAutopilotRecovery(", "shouldAutoSpawnMedic("} {
				if !strings.Contains(body, want) {
					t.Fatalf("%s does not call internal helper %s", file, want)
				}
			}
			for _, forbidden := range []string{"failureClassifyCmd", "recoveryLogReadCmd", "recoveryLogWriteCmd", "medicAutoSpawnCheckCmd", "rootCmd.Execute", "exec.Command"} {
				if strings.Contains(body, forbidden) {
					t.Fatalf("%s shells through public adapter %q", file, forbidden)
				}
			}
		}
	}
}
