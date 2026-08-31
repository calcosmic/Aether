package cmd

import (
	"context"
	"errors"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

func autopilotSignalsResult(signals codexContinueAutopilotSignals) map[string]interface{} {
	return map[string]interface{}{"autopilot_signals": signals}
}

func TestRunBlockerStagePolicy(t *testing.T) {
	tests := []struct {
		name     string
		before   blockerSnapshot
		after    blockerSnapshot
		wantCode autopilotTriggerCode
		wantStop bool
	}{
		{
			name:     "ordinary_baseline_proceeds",
			before:   blockerSnapshot{Count: 1, IDs: []string{"existing"}},
			after:    blockerSnapshot{Count: 1, IDs: []string{"existing"}},
			wantStop: false,
		},
		{
			name:     "baseline_escalation_stops",
			before:   blockerSnapshot{Count: 1, IDs: []string{"urgent"}, EscalatedCount: 1, EscalatedIDs: []string{"urgent"}},
			after:    blockerSnapshot{Count: 1, IDs: []string{"urgent"}, EscalatedCount: 1, EscalatedIDs: []string{"urgent"}},
			wantCode: autopilotTriggerBlockerEscalated,
			wantStop: true,
		},
		{
			name:     "count_increase_stops",
			before:   blockerSnapshot{Count: 1, IDs: []string{"existing"}},
			after:    blockerSnapshot{Count: 2, IDs: []string{"existing", "new"}},
			wantCode: autopilotTriggerBlockerCountIncreased,
			wantStop: true,
		},
		{
			name:     "new_escalation_stops",
			before:   blockerSnapshot{Count: 1, IDs: []string{"existing"}},
			after:    blockerSnapshot{Count: 1, IDs: []string{"urgent"}, EscalatedCount: 1, EscalatedIDs: []string{"urgent"}},
			wantCode: autopilotTriggerBlockerEscalated,
			wantStop: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			decision, stop := evaluateAutopilotRunStage(nil, tc.before, tc.after, false)
			if stop != tc.wantStop {
				t.Fatalf("stop = %t, want %t (decision %+v)", stop, tc.wantStop, decision)
			}
			if stop && decision.Code != tc.wantCode {
				t.Fatalf("code = %q, want %q", decision.Code, tc.wantCode)
			}
		})
	}
}

func TestRunAuditorCriticalHighAndDeterministicPolicy(t *testing.T) {
	score59, score60 := 59, 60
	tests := []struct {
		name     string
		result   map[string]interface{}
		wantCode autopilotTriggerCode
		wantStop bool
	}{
		{
			name: "current_deterministic_failure",
			result: map[string]interface{}{
				"blocked": true,
			},
			wantCode: autopilotTriggerDeterministicVerificationFailed,
			wantStop: true,
		},
		{
			name: "auditor_59",
			result: autopilotSignalsResult(continueReviewAutopilotSignals([]codexContinueWorkerFlowStep{{
				Name: "Auditor", Caste: "auditor", Status: "completed", OverallScore: &score59,
			}})),
			wantCode: autopilotTriggerAuditorScoreBelowFloor,
			wantStop: true,
		},
		{
			name: "auditor_60",
			result: autopilotSignalsResult(continueReviewAutopilotSignals([]codexContinueWorkerFlowStep{{
				Name: "Auditor", Caste: "auditor", Status: "completed", OverallScore: &score60,
			}})),
			wantStop: false,
		},
		{
			name:     "no_auditor",
			result:   autopilotSignalsResult(continueReviewAutopilotSignals(nil)),
			wantStop: false,
		},
		{
			name: "critical_finding",
			result: autopilotSignalsResult(continueReviewAutopilotSignals([]codexContinueWorkerFlowStep{{
				Name: "Gatekeeper", Caste: "gatekeeper", Status: "completed",
				Findings: []codexReviewFinding{{Severity: "CRITICAL", Description: "credential exposed"}},
			}})),
			wantCode: autopilotTriggerCriticalReviewFinding,
			wantStop: true,
		},
		{
			name: "high_finding_does_not_stop",
			result: autopilotSignalsResult(continueReviewAutopilotSignals([]codexContinueWorkerFlowStep{{
				Name: "Auditor", Caste: "auditor", Status: "completed",
				Findings: []codexReviewFinding{{Severity: "HIGH", Description: "cleanup recommended"}},
			}})),
			wantStop: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			decision, stop := evaluateAutopilotRunStage(tc.result, blockerSnapshot{}, blockerSnapshot{}, false)
			if stop != tc.wantStop {
				t.Fatalf("stop = %t, want %t (decision %+v)", stop, tc.wantStop, decision)
			}
			if stop && decision.Code != tc.wantCode {
				t.Fatalf("code = %q, want %q", decision.Code, tc.wantCode)
			}
		})
	}
}

func TestRunHeadlessAndInteractiveCheckpointDispositions(t *testing.T) {
	checkpoint := autopilotCheckpointReference{ID: "cp-runtime", Type: autopilotCheckpointTypeRuntimeVerification, Phase: 1, Question: "Confirm it", RecoveryCommand: "aether decision-answer"}
	result := autopilotSignalsResult(continueReviewAutopilotSignals(nil, []autopilotCheckpointReference{checkpoint}))

	headless, active := evaluateAutopilotRunStage(result, blockerSnapshot{}, blockerSnapshot{}, true)
	if !active || headless.Code != autopilotTriggerRuntimeVerificationNeeded || headless.Disposition != autopilotDispositionQueueAndContinue {
		t.Fatalf("headless runtime decision = %+v active=%t", headless, active)
	}
	interactive, active := evaluateAutopilotRunStage(result, blockerSnapshot{}, blockerSnapshot{}, false)
	if !active || interactive.Code != autopilotTriggerRuntimeVerificationNeeded || interactive.Disposition != autopilotDispositionPause {
		t.Fatalf("interactive runtime decision = %+v active=%t", interactive, active)
	}
}

func TestRunProviderTimeoutCancelAndGenericErrorClassification(t *testing.T) {
	providerErr := &workerProviderPreflightError{status: codex.AvailabilityStatus{
		Platform: codex.PlatformClaude,
		Category: codex.AvailabilityCategoryAuthInactive,
		Reason:   "not signed in",
	}}
	tests := []struct {
		name string
		err  error
		ctx  context.Context
		want autopilotTriggerCode
	}{
		{name: "provider", err: providerErr, ctx: context.Background(), want: autopilotTriggerProviderUnavailable},
		{name: "worker_timeout", err: context.DeadlineExceeded, ctx: context.Background(), want: autopilotTriggerWorkerTimeout},
		{name: "cancel", err: context.Canceled, ctx: context.Background(), want: autopilotTriggerCancelled},
		{name: "generic_non_runnable", err: errors.New("build packet is invalid"), ctx: context.Background(), want: autopilotTriggerColonyNotRunnable},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyAutopilotRunError(tc.ctx, tc.err); got != tc.want {
				t.Fatalf("classification = %q, want %q", got, tc.want)
			}
		})
	}
}

func mutateRunFixtureState(t *testing.T, mutate func(*colony.ColonyState)) colony.ColonyState {
	t.Helper()
	state, err := loadCompatibilityColonyState()
	if err != nil {
		t.Fatalf("load fixture state: %v", err)
	}
	mutate(&state)
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save fixture state: %v", err)
	}
	return state
}

func installAutopilotRunTestDeps(t *testing.T) {
	t.Helper()
	originalBuild := runAutopilotBuild
	originalContinue := runAutopilotContinue
	originalVisual := runAutopilotMaterializeVisual
	originalLessons := runAutopilotLoadLessons
	t.Cleanup(func() {
		runAutopilotBuild = originalBuild
		runAutopilotContinue = originalContinue
		runAutopilotMaterializeVisual = originalVisual
		runAutopilotLoadLessons = originalLessons
	})
}

func TestRunHeadlessQueuesVisualRuntimeAndReplanThenCompletes(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)
	_, root := seedRunFixture(t, 2)
	installAutopilotRunTestDeps(t)

	boundary := time.Now().UTC().Add(-time.Hour)
	mutateRunFixtureState(t, func(state *colony.ColonyState) {
		state.Plan = autopilotLessonPlan(boundary, "revision-overnight")
		state.Plan.Phases = []colony.Phase{
			{ID: 1, Name: "First", Status: colony.PhaseReady},
			{ID: 2, Name: "Second", Status: colony.PhasePending},
		}
		state.CurrentPhase = 1
		state.State = colony.StateREADY
	})

	buildCalls, continueCalls := 0, 0
	runAutopilotBuild = func(_ string, phaseNum int, _ []string, _ bool, _ codexBuildOptions) (map[string]interface{}, error) {
		buildCalls++
		mutateRunFixtureState(t, func(state *colony.ColonyState) {
			state.CurrentPhase = phaseNum
			state.State = colony.StateBUILT
			state.Plan.Phases[phaseNum-1].Status = colony.PhaseInProgress
		})
		return map[string]interface{}{"state": colony.StateBUILT, "claims_path": "fixture-claims.json"}, nil
	}
	runAutopilotMaterializeVisual = func(_ string, phaseNum int, _ map[string]interface{}) ([]autopilotCheckpointReference, error) {
		decision, _, err := upsertAutopilotCheckpoint(PendingDecision{
			Type: autopilotCheckpointTypeVisual, Description: "Review visual checkpoint", Source: "test",
		}, phaseNum, "visual")
		if err != nil {
			return nil, err
		}
		return []autopilotCheckpointReference{checkpointReference(decision)}, nil
	}
	runAutopilotContinue = func(_ string, _ codexContinueOptions) (map[string]interface{}, colony.ColonyState, colony.Phase, *colony.Phase, *signalHousekeepingResult, bool, error) {
		continueCalls++
		current, err := loadCompatibilityColonyState()
		if err != nil {
			t.Fatalf("load state in continue stub: %v", err)
		}
		phase := current.Plan.Phases[current.CurrentPhase-1]
		var checkpoints []autopilotCheckpointReference
		if phase.ID == 1 {
			decision, _, checkpointErr := upsertAutopilotCheckpoint(PendingDecision{
				Type: autopilotCheckpointTypeRuntimeVerification, Description: "Confirm runtime checkpoint", Source: "test",
			}, phase.ID, "runtime")
			if checkpointErr != nil {
				t.Fatalf("persist runtime checkpoint: %v", checkpointErr)
			}
			checkpoints = []autopilotCheckpointReference{checkpointReference(decision)}
		}
		final := phase.ID == 2
		updated := mutateRunFixtureState(t, func(state *colony.ColonyState) {
			state.Plan.Phases[phase.ID-1].Status = colony.PhaseCompleted
			if final {
				state.State = colony.StateCOMPLETED
				return
			}
			state.CurrentPhase = phase.ID + 1
			state.Plan.Phases[phase.ID].Status = colony.PhaseReady
			state.State = colony.StateREADY
		})
		result := map[string]interface{}{
			"advanced":          true,
			"state":             updated.State,
			"next":              "aether build",
			"autopilot_signals": continueReviewAutopilotSignals(nil, checkpoints),
		}
		return result, updated, phase, nil, nil, final, nil
	}
	runAutopilotLoadLessons = func(colony.Plan) ([]confirmedAutopilotLesson, error) {
		return []confirmedAutopilotLesson{{
			EntryID: "lesson-1", Content: "Keep work resumable", ContentHash: "hash-lesson", Phase: 1,
			PlanRevisionID: "revision-overnight", PlanRevisionAt: boundary.Format(time.RFC3339Nano),
		}}, nil
	}

	result, err := runCompatibilityAutopilot(root, runCompatibilityOptions{Headless: true, ReplanInterval: 1, Context: context.Background()})
	if err != nil {
		t.Fatalf("headless run: %v", err)
	}
	if result["stopped_reason"] != string(autopilotTriggerColonyComplete) || intValue(result["phases_completed"]) != 2 {
		t.Fatalf("headless run did not complete both phases: %+v", result)
	}
	if buildCalls != 2 || continueCalls != 2 {
		t.Fatalf("calls build=%d continue=%d, want 2/2", buildCalls, continueCalls)
	}

	wanted := map[string]bool{
		autopilotCheckpointTypeVisual:              false,
		autopilotCheckpointTypeRuntimeVerification: false,
		autopilotReplanDecisionType:                false,
	}
	for _, decision := range loadPendingDecisionFile().Decisions {
		if !decision.Resolved {
			if _, ok := wanted[decision.Type]; ok {
				wanted[decision.Type] = true
			}
		}
	}
	for decisionType, found := range wanted {
		if !found {
			t.Errorf("missing queued %s decision", decisionType)
		}
	}
}

func TestRunInteractiveVisualCheckpointPausesBeforeContinue(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)
	_, root := seedRunFixture(t, 2)
	installAutopilotRunTestDeps(t)

	continueCalls := 0
	runAutopilotBuild = func(_ string, phaseNum int, _ []string, _ bool, _ codexBuildOptions) (map[string]interface{}, error) {
		mutateRunFixtureState(t, func(state *colony.ColonyState) {
			state.State = colony.StateBUILT
			state.Plan.Phases[phaseNum-1].Status = colony.PhaseInProgress
		})
		return map[string]interface{}{"state": colony.StateBUILT, "claims_path": "fixture-claims.json"}, nil
	}
	runAutopilotMaterializeVisual = func(_ string, phaseNum int, _ map[string]interface{}) ([]autopilotCheckpointReference, error) {
		return []autopilotCheckpointReference{{ID: "cp-visual", Type: autopilotCheckpointTypeVisual, Phase: phaseNum, RecoveryCommand: "aether decision-answer"}}, nil
	}
	runAutopilotContinue = func(string, codexContinueOptions) (map[string]interface{}, colony.ColonyState, colony.Phase, *colony.Phase, *signalHousekeepingResult, bool, error) {
		continueCalls++
		return nil, colony.ColonyState{}, colony.Phase{}, nil, nil, false, errors.New("continue should not run")
	}

	result, err := runCompatibilityAutopilot(root, runCompatibilityOptions{Context: context.Background()})
	if err != nil {
		t.Fatalf("interactive run: %v", err)
	}
	if result["trigger_code"] != autopilotTriggerVisualCheckpointNeeded || continueCalls != 0 {
		t.Fatalf("interactive checkpoint result=%+v continue_calls=%d", result, continueCalls)
	}
}

func TestRunNoWholeBuildRetryOnGenericError(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)
	_, root := seedRunFixture(t, 1)
	installAutopilotRunTestDeps(t)

	buildCalls := 0
	runAutopilotBuild = func(string, int, []string, bool, codexBuildOptions) (map[string]interface{}, error) {
		buildCalls++
		return nil, errors.New("generic build failure")
	}
	result, err := runCompatibilityAutopilot(root, runCompatibilityOptions{Context: context.Background()})
	if err != nil {
		t.Fatalf("generic build failure should return a typed terminal result: %v", err)
	}
	if buildCalls != 1 {
		t.Fatalf("build calls = %d, want exactly 1", buildCalls)
	}
	if result["trigger_code"] != autopilotTriggerColonyNotRunnable || intValue(result["current_phase"]) != 1 {
		t.Fatalf("generic error result = %+v", result)
	}
}

func TestRunCancelMaxCompleteAndNoSealAreNamedNormalStops(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if got := classifyAutopilotRunError(cancelled, cancelled.Err()); got != autopilotTriggerCancelled {
		t.Fatalf("cancel classification = %q", got)
	}

	for _, code := range []autopilotTriggerCode{autopilotTriggerCancelled, autopilotTriggerWorkerTimeout, autopilotTriggerMaxPhasesReached, autopilotTriggerColonyComplete} {
		decision := autopilotRunDecisionForCode(code, false, nil)
		if decision.Disposition != autopilotDispositionNormalStop {
			t.Errorf("%s disposition = %s, want normal_stop", code, decision.Disposition)
		}
	}

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate test source")
	}
	source, err := os.ReadFile(strings.TrimSuffix(file, "run_autopilot_198_3_test.go") + "compatibility_cmds.go")
	if err != nil {
		t.Fatalf("read compatibility source: %v", err)
	}
	body := string(source)
	if strings.Contains(body, "runCodexSeal(") {
		t.Fatal("run compatibility command invokes seal")
	}
}

func TestRunLiveLoopHasNoLegacyScannerOrWholeBuildRetry(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate test source")
	}
	source, err := os.ReadFile(strings.TrimSuffix(file, "run_autopilot_198_3_test.go") + "compatibility_cmds.go")
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
	if strings.Contains(runBody, "checkAutopilotPauseConditions()") {
		t.Fatal("live run loop still calls the broad legacy scanner")
	}
	if strings.Count(runBody, "runAutopilotBuild(") != 1 {
		t.Fatalf("live run body has %d build invocation sites, want 1", strings.Count(runBody, "runAutopilotBuild("))
	}
}
