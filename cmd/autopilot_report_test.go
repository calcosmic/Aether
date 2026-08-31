package cmd

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

func reportTestState(status colony.State, current int) colony.ColonyState {
	phases := []colony.Phase{
		{ID: 1, Name: "First", Status: colony.PhaseCompleted},
		{ID: 2, Name: "Second", Status: colony.PhaseReady},
		{ID: 3, Name: "Third", Status: colony.PhasePending},
	}
	if status == colony.StateCOMPLETED {
		for i := range phases {
			phases[i].Status = colony.PhaseCompleted
		}
	}
	return colony.ColonyState{State: status, CurrentPhase: current, Plan: colony.Plan{Phases: phases}}
}

func TestAutopilotReportElapsedUsesInvocationWallClock(t *testing.T) {
	saveGlobals(t)
	store = nil
	started := time.Date(2026, 8, 31, 10, 0, 0, 0, time.UTC)
	finished := started.Add(12 * time.Minute)
	invocation := autopilotInvocation{
		ID:        "run-wallclock",
		StartedAt: started,
		Phases: []autopilotPhaseReport{{
			Phase: 1, PhaseName: "First", Outcome: "completed",
		}},
	}
	report := buildAutopilotInvocationReport(
		invocation,
		reportTestState(colony.StateREADY, 2),
		autopilotRunDecisionForCode(autopilotTriggerMaxPhasesReached, false, nil),
		finished,
		blockerSnapshot{},
	)
	if report.ElapsedSeconds != 12*60 {
		t.Fatalf("elapsed = %d seconds, want 720 wall-clock seconds", report.ElapsedSeconds)
	}
	if report.StartedAt != started.Format(time.RFC3339Nano) || report.FinishedAt != finished.Format(time.RFC3339Nano) {
		t.Fatalf("report timestamps = %q -> %q", report.StartedAt, report.FinishedAt)
	}

	// A later process starts a fresh invocation at its own start time. Time
	// spent offline between commands must never leak into the new report.
	restarted := finished.Add(8 * time.Hour)
	second := buildAutopilotInvocationReport(
		autopilotInvocation{ID: "run-after-offline-gap", StartedAt: restarted},
		reportTestState(colony.StateREADY, 2),
		autopilotRunDecisionForCode(autopilotTriggerCancelled, false, nil),
		restarted,
		blockerSnapshot{},
	)
	if second.ElapsedSeconds != 0 {
		t.Fatalf("new invocation elapsed = %d, want 0", second.ElapsedSeconds)
	}
}

func TestAutopilotReportRetainsHighFindingsQueuedDecisionsAndBlockers(t *testing.T) {
	saveGlobals(t)
	store = nil
	started := time.Date(2026, 8, 31, 10, 0, 0, 0, time.UTC)
	queued := autopilotQueuedDecisionReport{ID: "decision-7", Type: autopilotCheckpointTypeRuntimeVerification, Phase: 2}
	high := codexReviewFinding{Severity: "HIGH", Domain: "quality", Title: "Keep the evidence", File: "cmd/run.go", Line: 42}
	before := blockerSnapshot{Count: 1, IDs: []string{"old"}}
	after := blockerSnapshot{Count: 2, IDs: []string{"old", "urgent"}, EscalatedCount: 1, EscalatedIDs: []string{"urgent"}}

	report := buildAutopilotInvocationReport(
		autopilotInvocation{
			ID:             "run-evidence",
			StartedAt:      started,
			BlockersBefore: before,
			Phases: []autopilotPhaseReport{{
				Phase: 2, PhaseName: "Second", Outcome: "stopped",
				HighFindings:    []codexReviewFinding{high},
				QueuedDecisions: []autopilotQueuedDecisionReport{queued},
			}},
		},
		reportTestState(colony.StateBUILT, 2),
		autopilotRunDecisionForCode(autopilotTriggerBlockerEscalated, false, nil),
		started.Add(time.Minute),
		after,
	)

	if !reflect.DeepEqual(report.BlockersBefore, before) || !reflect.DeepEqual(report.BlockersAfter, after) {
		t.Fatalf("blocker movement was not retained: before=%+v after=%+v", report.BlockersBefore, report.BlockersAfter)
	}
	if len(report.QueuedDecisions) != 1 || report.QueuedDecisions[0] != queued {
		t.Fatalf("queued decisions = %+v", report.QueuedDecisions)
	}
	if len(report.Phases) != 1 || !reflect.DeepEqual(report.Phases[0].HighFindings, []codexReviewFinding{high}) {
		t.Fatalf("HIGH findings = %+v", report.Phases)
	}
}

func TestAutopilotReportNextMatchesSharedResolver(t *testing.T) {
	tests := []struct {
		name  string
		state colony.ColonyState
	}{
		{name: "completion", state: reportTestState(colony.StateCOMPLETED, 3)},
		{name: "genuine stop", state: reportTestState(colony.StateBUILT, 2)},
		{name: "interactive pause", state: reportTestState(colony.StateREADY, 2)},
		{name: "exact phase resume", state: reportTestState(colony.StateREADY, 3)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			want := lifecycleNextActionForState(tc.state, "run", "", "").Command
			if got := resolveAutopilotReportNext(tc.state); got != want {
				t.Fatalf("report next = %q, shared resolver = %q", got, want)
			}
		})
	}
}

func TestAutopilotReportSpendUsesMeasuredLedgerOnly(t *testing.T) {
	setupSpendTestStore(t)
	seedSpendLedgerForTest(t, 1, spendWorkflowBuild,
		measuredSpendRowForTest("Mason-1", "builder", 1_250),
		spendRow{AgentName: "Keen-1", Caste: "watcher", Task: "check", Status: "completed"},
		spendRow{AgentName: "Roam-1", Caste: "scout", Task: "look", Status: "completed", Usage: codex.WorkerUsage{TotalTokens: 900_000, Source: codex.UsageSourceEstimate}},
	)

	spend := snapshotAutopilotSpend([]autopilotPhaseReport{
		{Phase: 1, PhaseName: "First"},
		{Phase: 2, PhaseName: "Second"},
	})
	if spend.MeasuredTokens == nil || *spend.MeasuredTokens != 1_250 {
		t.Fatalf("measured total = %v, want 1250", spend.MeasuredTokens)
	}
	if spend.MeasuredRows != 1 || spend.UnreportedRows != 2 {
		t.Fatalf("row counts = measured %d, unreported %d", spend.MeasuredRows, spend.UnreportedRows)
	}
	if len(spend.Phases) != 2 || spend.Phases[1].MeasuredTokens != nil || spend.Phases[1].LedgerPresent {
		t.Fatalf("missing phase must stay unreported: %+v", spend.Phases)
	}

	report := autopilotInvocationReport{SchemaVersion: autopilotReportSchemaVersion, Outcome: "normal_stop", StopReason: "max_phases_reached", Spend: spend, Next: "aether build 2"}
	rendered := renderAutopilotInvocationReport(report)
	if !strings.Contains(rendered, "—  not reported") {
		t.Fatalf("rendered report does not expose unreported rows:\n%s", rendered)
	}
	if strings.Contains(rendered, "900K") || strings.Contains(rendered, "900000") {
		t.Fatalf("estimated usage leaked into measured report:\n%s", rendered)
	}
}

func TestAutopilotReportRendererUsesRequiredOrderAndExactNext(t *testing.T) {
	report := autopilotInvocationReport{
		SchemaVersion: autopilotReportSchemaVersion,
		InvocationID:  "run-order",
		Outcome:       "genuine_stop",
		StopReason:    "critical_review_finding",
		QueuedDecisions: []autopilotQueuedDecisionReport{{
			ID: "decision-1", Type: autopilotCheckpointTypeRuntimeVerification, Phase: 1,
		}},
		BlockersBefore: blockerSnapshot{Count: 1},
		BlockersAfter:  blockerSnapshot{Count: 2, EscalatedCount: 1},
		ElapsedSeconds: 720,
		Spend:          autopilotSpendReport{UnreportedRows: 1},
		Next:           "aether build 2",
		Phases:         []autopilotPhaseReport{{Phase: 1, PhaseName: "First", Outcome: "stopped"}},
	}
	rendered := renderAutopilotInvocationReport(report)
	wants := []string{"Outcome:", "Stop reason:", "Queued decisions", "Blocker movement", "Elapsed:", "What The Helpers Cost", "Next: aether build 2", "Phase outcomes"}
	last := -1
	for _, want := range wants {
		at := strings.Index(rendered, want)
		if at < 0 {
			t.Fatalf("rendered report missing %q:\n%s", want, rendered)
		}
		if at <= last {
			t.Fatalf("%q appeared out of order:\n%s", want, rendered)
		}
		last = at
	}
}

func TestStatusLastReportRemainsStoredAfterClockAndLedgerMutation(t *testing.T) {
	saveGlobals(t)
	_, _ = seedRunFixture(t, 1)
	finished := time.Date(2026, 8, 31, 10, 12, 0, 0, time.UTC)
	measured := int64(1_250)
	stored := autopilotInvocationReport{
		SchemaVersion:  autopilotReportSchemaVersion,
		InvocationID:   "run-stored",
		StartedAt:      finished.Add(-12 * time.Minute).Format(time.RFC3339Nano),
		FinishedAt:     finished.Format(time.RFC3339Nano),
		ElapsedSeconds: 720,
		Outcome:        "normal_stop",
		StopReason:     string(autopilotTriggerMaxPhasesReached),
		Spend:          autopilotSpendReport{MeasuredTokens: &measured, MeasuredRows: 1},
		Next:           "aether build 1",
	}
	if err := store.SaveJSON(autopilotStatePath, autopilotState{LastReport: &stored}); err != nil {
		t.Fatalf("seed report: %v", err)
	}

	state, err := loadCompatibilityColonyState()
	if err != nil {
		t.Fatalf("load colony: %v", err)
	}
	first := buildStatusResult(state, store)

	// Mutate both inputs a recomputing status implementation would observe.
	seedSpendLedgerForTest(t, 1, spendWorkflowBuild, measuredSpendRowForTest("Mason-1", "builder", 999_999))
	originalNow := autopilotNow
	autopilotNow = func() time.Time { return finished.Add(48 * time.Hour) }
	t.Cleanup(func() { autopilotNow = originalNow })
	second := buildStatusResult(state, store)

	if !reflect.DeepEqual(first["last_report"], second["last_report"]) || !reflect.DeepEqual(second["last_report"], &stored) {
		t.Fatalf("status recomputed persisted report:\nfirst=%+v\nsecond=%+v", first["last_report"], second["last_report"])
	}
	if got := renderAutopilotReportFromResult(second); got != renderAutopilotInvocationReport(stored) {
		t.Fatalf("status did not use shared stored renderer:\n%s", got)
	}
}

func TestRunDryRunPreservesLastReport(t *testing.T) {
	saveGlobals(t)
	_, root := seedRunFixture(t, 1)
	stored := autopilotInvocationReport{SchemaVersion: autopilotReportSchemaVersion, InvocationID: "keep-me", Outcome: "completed", Next: "aether seal"}
	if err := store.SaveJSON(autopilotStatePath, autopilotState{LastReport: &stored}); err != nil {
		t.Fatalf("seed report: %v", err)
	}
	if _, err := runCompatibilityAutopilot(root, runCompatibilityOptions{DryRun: true, Context: context.Background()}); err != nil {
		t.Fatalf("dry run: %v", err)
	}
	var after autopilotState
	if err := store.LoadJSON(autopilotStatePath, &after); err != nil {
		t.Fatalf("load report after dry run: %v", err)
	}
	if after.LastReport == nil || !reflect.DeepEqual(*after.LastReport, stored) {
		t.Fatalf("dry run changed last report: %+v", after.LastReport)
	}
}

func TestAutopilotStateWithoutLastReportStillLoads(t *testing.T) {
	saveGlobals(t)
	_, _ = seedRunFixture(t, 1)
	legacy := map[string]interface{}{
		"initialized_at": "2026-08-01T00:00:00Z",
		"total_phases":   1,
		"current_phase":  1,
		"status":         "running",
	}
	if err := store.SaveJSON(autopilotStatePath, legacy); err != nil {
		t.Fatalf("seed legacy state: %v", err)
	}
	var decoded autopilotState
	if err := store.LoadJSON(autopilotStatePath, &decoded); err != nil {
		t.Fatalf("load legacy state: %v", err)
	}
	if decoded.LastReport != nil {
		t.Fatalf("legacy state manufactured report: %+v", decoded.LastReport)
	}
}
