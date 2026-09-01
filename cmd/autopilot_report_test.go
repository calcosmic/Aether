package cmd

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

func reportTestState(status colony.State, current int) colony.ColonyState {
	phases := []colony.Phase{
		{ID: 1, Name: "First", Status: colony.PhasePending},
		{ID: 2, Name: "Second", Status: colony.PhasePending},
		{ID: 3, Name: "Third", Status: colony.PhasePending},
	}
	for i := range phases {
		switch {
		case status == colony.StateCOMPLETED || phases[i].ID < current:
			phases[i].Status = colony.PhaseCompleted
		case phases[i].ID == current && (status == colony.StateBUILT || status == colony.StateEXECUTING):
			phases[i].Status = colony.PhaseInProgress
		case phases[i].ID == current:
			phases[i].Status = colony.PhaseReady
		}
	}
	return colony.ColonyState{State: status, CurrentPhase: current, Plan: colony.Plan{Phases: phases}}
}

// installTerminalReportPersistenceFixture drives the real `run` command to a
// completed one-phase result through the existing orchestration seams. The
// failure form makes only the final autopilot report path unwritable, after
// the loop's earlier running-state writes have already succeeded. That keeps
// these command tests honest about the exact durability boundary under test.
func installTerminalReportPersistenceFixture(t *testing.T, failFinalWrite bool) {
	t.Helper()
	installAutopilotRunTestDeps(t)

	runAutopilotBuild = func(_ string, phaseNum int, _ []string, _ bool, _ codexBuildOptions) (map[string]interface{}, error) {
		mutateRunFixtureState(t, func(state *colony.ColonyState) {
			state.CurrentPhase = phaseNum
			state.State = colony.StateBUILT
			state.Plan.Phases[phaseNum-1].Status = colony.PhaseInProgress
		})
		return map[string]interface{}{
			"dispatch_count": 1,
			"dispatch_mode":  "fixture",
			"state":          colony.StateBUILT,
		}, nil
	}
	runAutopilotMaterializeVisual = func(string, int, map[string]interface{}) ([]autopilotCheckpointReference, error) {
		return nil, nil
	}
	runAutopilotContinue = func(string, codexContinueOptions) (map[string]interface{}, colony.ColonyState, colony.Phase, *colony.Phase, *signalHousekeepingResult, bool, error) {
		updated := mutateRunFixtureState(t, func(state *colony.ColonyState) {
			state.State = colony.StateCOMPLETED
			state.Plan.Phases[state.CurrentPhase-1].Status = colony.PhaseCompleted
		})
		phase := updated.Plan.Phases[updated.CurrentPhase-1]
		if failFinalWrite {
			reportPath := filepath.Join(store.BasePath(), autopilotStatePath)
			if err := os.Remove(reportPath); err != nil && !os.IsNotExist(err) {
				t.Fatalf("remove running autopilot state: %v", err)
			}
			if err := os.Mkdir(reportPath, 0755); err != nil {
				t.Fatalf("make final report path unwritable: %v", err)
			}
		}
		return map[string]interface{}{
			"advanced": true,
			"blocked":  false,
			"state":    colony.StateCOMPLETED,
		}, updated, phase, nil, nil, true, nil
	}
}

func runTerminalReportCommand(t *testing.T, mode string, failFinalWrite bool) (string, string, error) {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)
	t.Setenv("AETHER_OUTPUT_MODE", mode)
	_, _ = seedRunFixture(t, 1)
	installTerminalReportPersistenceFixture(t, failFinalWrite)

	var out, errOut bytes.Buffer
	stdout = &out
	stderr = &errOut
	rootCmd.SetArgs([]string{"run", "--continue"})
	err := Execute()
	return out.String(), errOut.String(), err
}

func TestRunReportPersistenceFailureJSONCommandIsNonZero(t *testing.T) {
	stdoutText, stderrText, commandErr := runTerminalReportCommand(t, "json", true)
	var renderedErr renderedCommandError
	if !errors.As(commandErr, &renderedErr) || renderedErr.code != 1 {
		t.Fatalf("command error = %#v, want rendered exit 1\nstdout:\n%s\nstderr:\n%s", commandErr, stdoutText, stderrText)
	}
	if strings.TrimSpace(stdoutText) != "" {
		t.Fatalf("failed report write emitted success output:\n%s", stdoutText)
	}
	envelope := parseEnvelopeCmd(t, stderrText)
	if envelope["ok"] != false || intValue(envelope["code"]) != 1 {
		t.Fatalf("error envelope = %+v, want ok:false code:1", envelope)
	}
	if !strings.Contains(strings.ToLower(stringValue(envelope["error"])), "report") ||
		!strings.Contains(strings.ToLower(stringValue(envelope["error"])), "persist") {
		t.Fatalf("error does not name report durability: %+v", envelope)
	}
	details, ok := envelope["details"].(map[string]interface{})
	if !ok {
		t.Fatalf("error details = %#v, want complete run result", envelope["details"])
	}
	if stringValue(details["report_persist_error"]) == "" ||
		stringValue(details["trigger_code"]) != string(autopilotTriggerColonyComplete) ||
		stringValue(details["disposition"]) != string(autopilotDispositionNormalStop) {
		t.Fatalf("missing persistence/trigger evidence: %+v", details)
	}
	report, ok := details["last_report"].(map[string]interface{})
	if !ok {
		t.Fatalf("last_report = %#v, want retained report", details["last_report"])
	}
	if stringValue(report["outcome"]) != "completed" ||
		stringValue(report["stop_reason"]) != string(autopilotTriggerColonyComplete) ||
		stringValue(report["next"]) != "aether seal" {
		t.Fatalf("retained terminal evidence = %+v", report)
	}
	if _, ok := report["elapsed_seconds"]; !ok {
		t.Fatalf("retained report lost elapsed evidence: %+v", report)
	}
	if _, ok := report["spend"].(map[string]interface{}); !ok {
		t.Fatalf("retained report lost spend evidence: %+v", report)
	}
	if phases, ok := report["phases"].([]interface{}); !ok || len(phases) != 1 {
		t.Fatalf("retained report lost primary phase evidence: %+v", report["phases"])
	}
}

func TestRunReportPersistenceFailureVisualIsErrorFirst(t *testing.T) {
	stdoutText, stderrText, commandErr := runTerminalReportCommand(t, "visual", true)
	var renderedErr renderedCommandError
	if !errors.As(commandErr, &renderedErr) || renderedErr.code != 1 {
		t.Fatalf("command error = %#v, want rendered exit 1\nstdout:\n%s\nstderr:\n%s", commandErr, stdoutText, stderrText)
	}
	upperErr := strings.ToUpper(strings.TrimSpace(stderrText))
	if !strings.HasPrefix(upperErr, "━━━") || !strings.Contains(upperErr, "REPORT DURABILITY FAILURE") {
		t.Fatalf("stderr did not lead with the durability failure:\n%s", stderrText)
	}
	wants := []string{
		"UNSAVED IN-MEMORY EVIDENCE",
		"Outcome:",
		"Stop reason:",
		"Blocker movement",
		"Elapsed:",
		"What The Helpers Cost",
		"Next:",
		"Phase evidence",
	}
	last := -1
	for _, want := range wants {
		at := strings.Index(stderrText, want)
		if at < 0 {
			t.Fatalf("unsaved evidence missing %q:\n%s", want, stderrText)
		}
		if at <= last {
			t.Fatalf("unsaved evidence %q appeared out of order:\n%s", want, stderrText)
		}
		last = at
	}
	for _, forbidden := range []string{"Autopilot Complete", "Project Complete", "Last Autopilot Run"} {
		if strings.Contains(stdoutText, forbidden) || strings.Contains(stderrText, forbidden) {
			t.Fatalf("durability failure rendered terminal success %q\nstdout:\n%s\nstderr:\n%s", forbidden, stdoutText, stderrText)
		}
	}
}

func TestRunReportPersistenceFailureLeavesDurableSuccessUnchanged(t *testing.T) {
	stdoutText, stderrText, commandErr := runTerminalReportCommand(t, "json", false)
	if commandErr != nil {
		t.Fatalf("durable terminal run returned error: %v\nstdout:\n%s\nstderr:\n%s", commandErr, stdoutText, stderrText)
	}
	if strings.TrimSpace(stderrText) != "" {
		t.Fatalf("durable terminal run wrote stderr:\n%s", stderrText)
	}
	envelope := parseEnvelopeCmd(t, stdoutText)
	if envelope["ok"] != true {
		t.Fatalf("durable terminal run lost success envelope: %+v", envelope)
	}
	result, ok := envelope["result"].(map[string]interface{})
	if !ok || result["last_report"] == nil || result["report_persist_error"] != nil {
		t.Fatalf("durable terminal result changed: %+v", result)
	}
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
		availableAutopilotBlockerSnapshot(blockerSnapshot{}),
	)
	if report.ElapsedSeconds != 12*60 {
		t.Fatalf("elapsed = %d seconds, want 720 wall-clock seconds", report.ElapsedSeconds)
	}
	workerDurationSum := 4 * time.Minute
	if time.Duration(report.ElapsedSeconds)*time.Second == workerDurationSum {
		t.Fatal("report elapsed time was derived from the deliberately different worker-duration sum")
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
		availableAutopilotBlockerSnapshot(blockerSnapshot{}),
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
			BlockersBefore: availableAutopilotBlockerSnapshot(before),
			Phases: []autopilotPhaseReport{{
				Phase: 2, PhaseName: "Second", Outcome: "stopped",
				HighFindings:    []codexReviewFinding{high},
				QueuedDecisions: []autopilotQueuedDecisionReport{queued},
			}},
		},
		reportTestState(colony.StateBUILT, 2),
		autopilotRunDecisionForCode(autopilotTriggerBlockerEscalated, false, nil),
		started.Add(time.Minute),
		availableAutopilotBlockerSnapshot(after),
	)

	if report.BlockersBefore.Snapshot == nil || report.BlockersAfter.Snapshot == nil || !reflect.DeepEqual(*report.BlockersBefore.Snapshot, before) || !reflect.DeepEqual(*report.BlockersAfter.Snapshot, after) {
		t.Fatalf("blocker movement was not retained: before=%+v after=%+v", report.BlockersBefore, report.BlockersAfter)
	}
	if len(report.QueuedDecisions) != 1 || report.QueuedDecisions[0] != queued {
		t.Fatalf("queued decisions = %+v", report.QueuedDecisions)
	}
	if len(report.Phases) != 1 || !reflect.DeepEqual(report.Phases[0].HighFindings, []codexReviewFinding{high}) {
		t.Fatalf("HIGH findings = %+v", report.Phases)
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	var roundTripped autopilotInvocationReport
	if err := json.Unmarshal(encoded, &roundTripped); err != nil {
		t.Fatalf("unmarshal report: %v", err)
	}
	if !reflect.DeepEqual(roundTripped.QueuedDecisions, report.QueuedDecisions) || !reflect.DeepEqual(roundTripped.Phases[0].HighFindings, report.Phases[0].HighFindings) {
		t.Fatalf("typed evidence changed across JSON round trip: %+v", roundTripped)
	}
}

func TestRunReportTerminalCategoriesPersistVersionedRecord(t *testing.T) {
	tests := []struct {
		name     string
		state    colony.ColonyState
		decision autopilotRunDecision
		outcome  string
	}{
		{name: "successful completion", state: reportTestState(colony.StateCOMPLETED, 3), decision: autopilotRunDecisionForCode(autopilotTriggerColonyComplete, false, nil), outcome: "completed"},
		{name: "genuine stop", state: reportTestState(colony.StateBUILT, 2), decision: autopilotRunDecisionForCode(autopilotTriggerDeterministicVerificationFailed, false, nil), outcome: "genuine_stop"},
		{name: "normal stop", state: reportTestState(colony.StateREADY, 2), decision: autopilotRunDecisionForCode(autopilotTriggerCancelled, false, nil), outcome: "normal_stop"},
		{name: "interactive pause", state: reportTestState(colony.StateBUILT, 2), decision: autopilotRunDecisionForCode(autopilotTriggerRuntimeVerificationNeeded, false, nil), outcome: "paused"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setupSpendTestStore(t)
			started := time.Date(2026, 8, 31, 10, 0, 0, 0, time.UTC)
			originalNow := autopilotNow
			autopilotNow = func() time.Time { return started.Add(time.Minute) }
			t.Cleanup(func() { autopilotNow = originalNow })
			invocation := autopilotInvocation{ID: "run-terminal", StartedAt: started, Phases: []autopilotPhaseReport{}}
			result := finishAutopilotInvocation(&invocation, tc.state, runCompatibilityOptions{}, nil, 0, tc.decision, nil)

			var persisted autopilotState
			if err := store.LoadJSON(autopilotStatePath, &persisted); err != nil {
				t.Fatalf("load persisted report: %v", err)
			}
			if persisted.SchemaVersion != autopilotStateSchemaVersion || persisted.LastReport == nil {
				t.Fatalf("missing versioned last_report: %+v", persisted)
			}
			if persisted.LastReport.Outcome != tc.outcome || persisted.LastReport.SchemaVersion != autopilotReportSchemaVersion {
				t.Fatalf("persisted outcome = %+v, want %q", persisted.LastReport, tc.outcome)
			}
			resultJSON, _ := json.Marshal(result["last_report"])
			storedJSON, _ := json.Marshal(persisted.LastReport)
			if string(resultJSON) != string(storedJSON) {
				t.Fatalf("run result and stored report differ: result=%s stored=%s", resultJSON, storedJSON)
			}
		})
	}
}

func TestRunReportStartsFreshInvocationIdentity(t *testing.T) {
	saveGlobals(t)
	store = nil
	fixed := time.Date(2026, 8, 31, 10, 0, 0, 0, time.UTC)
	originalNow := autopilotNow
	autopilotNow = func() time.Time { return fixed }
	t.Cleanup(func() { autopilotNow = originalNow })

	first := beginAutopilotInvocation(reportTestState(colony.StateREADY, 1))
	second := beginAutopilotInvocation(reportTestState(colony.StateREADY, 1))
	if first.ID == second.ID {
		t.Fatalf("two real invocations reused ID %q", first.ID)
	}
	if !first.StartedAt.Equal(fixed) || !second.StartedAt.Equal(fixed) {
		t.Fatalf("fresh starts = %s and %s, want %s", first.StartedAt, second.StartedAt, fixed)
	}
}

func TestRunReportAllPostStartTerminalBranchesUseOneFinalizer(t *testing.T) {
	source, err := os.ReadFile("compatibility_cmds.go")
	if err != nil {
		t.Fatalf("read compatibility source: %v", err)
	}
	body := string(source)
	start := strings.Index(body, "func runCompatibilityAutopilot(")
	end := strings.Index(body[start:], "\nfunc loadCompatibilityColonyState(")
	if start < 0 || end < 0 {
		t.Fatal("locate runCompatibilityAutopilot source body")
	}
	runBody := body[start : start+end]
	postStart := runBody[strings.Index(runBody, "invocation := beginAutopilotInvocation"):]
	if strings.Contains(postStart, "finishAutopilotRunDecision") {
		t.Fatal("post-start terminal branch bypasses finishAutopilotInvocation")
	}
	if strings.Contains(postStart, "return nil, err") {
		t.Fatal("post-start error return bypasses the report finalizer")
	}
	if strings.Count(postStart, "finishAutopilotInvocation(") != 1 {
		t.Fatalf("run body has %d direct finalizer calls, want the one closure funnel", strings.Count(postStart, "finishAutopilotInvocation("))
	}
}

func TestAutopilotReportNextMatchesSharedResolver(t *testing.T) {
	tests := []struct {
		name     string
		state    colony.ColonyState
		decision autopilotRunDecision
	}{
		{name: "completion", state: reportTestState(colony.StateCOMPLETED, 3), decision: autopilotRunDecisionForCode(autopilotTriggerColonyComplete, false, nil)},
		{name: "genuine stop", state: reportTestState(colony.StateBUILT, 2), decision: autopilotRunDecisionForCode(autopilotTriggerDeterministicVerificationFailed, false, nil)},
		{name: "interactive pause", state: reportTestState(colony.StateREADY, 2), decision: autopilotRunDecisionForCode(autopilotTriggerReplanDue, false, nil)},
		{name: "exact phase resume", state: reportTestState(colony.StateREADY, 3), decision: autopilotRunDecision{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			want := lifecycleNextActionForState(tc.state, "run", tc.decision.Next, "The autopilot stop named this exact recovery step.").Command
			if got := resolveAutopilotReportNext(tc.state, tc.decision); got != want {
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
		BlockersBefore: availableAutopilotBlockerSnapshot(blockerSnapshot{Count: 1}),
		BlockersAfter:  availableAutopilotBlockerSnapshot(blockerSnapshot{Count: 2, EscalatedCount: 1}),
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

func TestAutopilotReportUnavailableBlockerTruthIsNeverZero(t *testing.T) {
	report := autopilotInvocationReport{
		SchemaVersion: autopilotReportSchemaVersion,
		Outcome:       "genuine_stop",
		StopReason:    string(autopilotTriggerColonyNotRunnable),
		BlockersBefore: autopilotBlockerSnapshotReport{
			Available: false,
			Snapshot:  nil,
			Error:     "blocker truth pending-decisions.json: unavailable",
		},
		BlockersAfter: availableAutopilotBlockerSnapshot(blockerSnapshot{IDs: []string{}, EscalatedIDs: []string{}}),
		Spend:         autopilotSpendReport{Phases: []autopilotPhaseSpendReport{}},
		Next:          "aether status",
		Phases:        []autopilotPhaseReport{},
	}
	for name, rendered := range map[string]string{
		"normal":              renderAutopilotInvocationReport(report),
		"persistence failure": renderRunReportPersistenceFailure(map[string]interface{}{"last_report": &report, "stopped_reason": report.StopReason}),
	} {
		if !strings.Contains(rendered, "Blocker movement: unavailable") {
			t.Errorf("%s renderer hid unavailable blocker truth:\n%s", name, rendered)
		}
		if strings.Contains(rendered, "Blocker movement: 0") {
			t.Errorf("%s renderer converted unavailable truth to zero:\n%s", name, rendered)
		}
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal unavailable report: %v", err)
	}
	if !bytes.Contains(encoded, []byte(`"available":false`)) || !bytes.Contains(encoded, []byte(`"snapshot":null`)) {
		t.Fatalf("JSON lost explicit unavailable evidence: %s", encoded)
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
	state, err := loadCompatibilityColonyState()
	if err != nil {
		t.Fatalf("load legacy colony for status: %v", err)
	}
	if result := buildStatusResult(state, store); result["last_report"] != nil {
		t.Fatalf("legacy status manufactured report: %+v", result["last_report"])
	}
}

func TestAutopilotReportCheckpointCapabilityIsImmediateOnly(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s
	phase := colony.Phase{ID: 1, Name: "Owner handoff", Status: colony.PhaseCompleted}
	state := checkpointTestState(t, phase, colony.StateCOMPLETED)
	refs, err := materializeRuntimeVerificationCheckpoints(phase.ID, []codexCriterionVerification{{
		TaskID: "1.1", Criterion: "The finished interaction feels correct", State: criterionStateNeedsOwnerConfirmation,
		Evidence: []string{"automated checks cannot judge feel"},
	}}, checkpointTestGeneration(t, "report-immediate", "report-immediate-evidence"))
	if err != nil || len(refs) != 1 {
		t.Fatalf("materialize immediate checkpoint: refs=%#v err=%v", refs, err)
	}
	immediateCommand := refs[0].RecoveryCommand
	capability := checkpointCapabilityFromReference(t, refs[0])
	if !strings.Contains(immediateCommand, "--checkpoint-capability") {
		t.Fatalf("immediate owner output has no capability: %s", immediateCommand)
	}

	decision := autopilotRunDecision{
		Code:        autopilotTriggerRuntimeVerificationNeeded,
		Disposition: autopilotDispositionPause,
		Next:        immediateCommand,
		Checkpoints: refs,
	}
	invocation := beginAutopilotInvocation(state)
	invocation.recordRunDecision(state, decision)
	report := buildAutopilotInvocationReport(invocation, state, decision, autopilotNow(), availableAutopilotBlockerSnapshot(blockerSnapshot{}))
	if report.Next != "aether seal" {
		t.Fatalf("seal-ready durable next = %q, want capability-free re-entry through aether seal", report.Next)
	}
	if len(report.QueuedDecisions) != 1 || report.QueuedDecisions[0].ID != refs[0].ID || report.QueuedDecisions[0].Question != refs[0].Question {
		t.Fatalf("durable report lost checkpoint identity/question: %#v", report.QueuedDecisions)
	}
	if err := syncRunAutopilotStateWithReport(state, runCompatibilityOptions{}, "paused", string(decision.Code), &report); err != nil {
		t.Fatalf("persist capability-safe report: %v", err)
	}

	stored := loadAutopilotLastReport(s)
	if stored == nil || stored.Next != "aether seal" || len(stored.QueuedDecisions) != 1 {
		t.Fatalf("stored last report lost safe re-entry or owner evidence: %#v", stored)
	}
	statusResult := buildStatusResult(state, s)
	statusJSON, err := json.Marshal(statusResult)
	if err != nil {
		t.Fatalf("marshal status: %v", err)
	}
	statusVisual := renderAutopilotReportFromResult(statusResult)
	if bytes.Contains(statusJSON, []byte(capability)) || strings.Contains(statusVisual, capability) {
		t.Fatalf("status reopened a stale raw capability:\njson=%s\nvisual=%s", statusJSON, statusVisual)
	}
	if !strings.Contains(statusVisual, "Next: aether seal") || !strings.Contains(statusVisual, refs[0].ID) {
		t.Fatalf("status omitted safe next action or queued identity:\n%s", statusVisual)
	}

	if err := filepath.Walk(s.BasePath(), func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if bytes.Contains(data, []byte(capability)) {
			t.Fatalf("durable file %s persisted raw checkpoint capability", path)
		}
		return nil
	}); err != nil {
		t.Fatalf("scan durable data tree: %v", err)
	}
	pending := pendingDecisionBytes(t)
	digest := sha256.Sum256([]byte(capability))
	if !bytes.Contains(pending, []byte(hex.EncodeToString(digest[:]))) || !bytes.Contains(pending, []byte(refs[0].ID)) {
		t.Fatalf("pending state lost capability hash or checkpoint identity: %s", pending)
	}

	nonSealReady := reportTestState(colony.StateREADY, 2)
	wantReentry := lifecycleNextActionForState(nonSealReady, "run", "", "").Command
	if got := resolveAutopilotReportNext(nonSealReady, decision); got != wantReentry || strings.Contains(got, "--checkpoint-capability") {
		t.Fatalf("unfinished durable next = %q, want safe lifecycle re-entry %q", got, wantReentry)
	}
}

func TestMorningHandoffCheckpointUsesSealEmittedCommand(t *testing.T) {
	readme, err := os.ReadFile("../README.md")
	if err != nil {
		t.Fatalf("read README: %v", err)
	}
	text := string(readme)
	start := strings.Index(text, "### Morning handoff")
	end := strings.Index(text[start:], "\n## 🔌 Works With")
	if start < 0 || end < 0 {
		t.Fatal("locate README morning handoff section")
	}
	section := text[start : start+end]
	for _, required := range []string{
		"aether pending-decision-list --unresolved",
		"aether seal",
		"exact",
		"emitted",
		"capability",
	} {
		if !strings.Contains(strings.ToLower(section), strings.ToLower(required)) {
			t.Fatalf("morning handoff omitted %q:\n%s", required, section)
		}
	}
	if strings.Contains(section, "aether decision-answer --question") {
		t.Fatalf("morning handoff documents a capability-free answer template instead of seal-emitted authorization:\n%s", section)
	}
}
