package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// Phase 198 plan 01 (SHOW-01/D-12) -- the tracer slice: the chat-path closing
// screen for `continue` must come from the SAME renderer the direct
// command-line path uses, fed by the finalizer's own saved result rather than
// a second, narrower hand-derivation.
//
// Fixture construction follows CLAUDE.md's "derive fixture values the way the
// runtime derives them" rule as applied by 198-PATTERNS.md's
// Fixture-construction pattern: build the typed result the finalizer builds
// (nested typed structs, not hand-typed maps), then json.Marshal +
// json.Unmarshal it into map[string]interface{} for the closeout side -- a
// real completion file has always round-tripped through JSON, so a
// hand-typed nested-struct map would be a false certificate here.

// wrapperParityColonyState is the live state both the direct render call and
// closeoutDirectVisual receive -- the source of truth for phase names and
// counts that the completion result itself does not carry.
func wrapperParityColonyState() colony.ColonyState {
	goal := "Ship the thing"
	return colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateBUILT,
		CurrentPhase: 2,
		Plan: colony.Plan{Phases: []colony.Phase{
			{ID: 1, Name: "Ship the thing"},
			{ID: 2, Name: "Ship the next thing"},
		}},
	}
}

// wrapperParityAdvanceResult builds the typed result advanceExternalContinue
// produces (the fields renderContinueVisual and closeoutContinueRenderInputs
// actually read), as a map whose nested values are the real Go structs the
// finalizer uses -- never hand-typed nested maps.
func wrapperParityAdvanceResult() map[string]interface{} {
	verification := codexContinueVerificationReport{
		Phase: 1,
		Steps: []codexVerificationStep{
			{Name: "build", Passed: true},
			{Name: "types", Passed: true},
			{Name: "tests", Passed: false, Skipped: true},
		},
		Claims:         codexClaimVerification{Present: true, Passed: true, Summary: "3 files matched the claimed changes"},
		Watcher:        codexWatcherVerification{Present: true, Passed: true, Status: "completed"},
		CriteriaPolicy: "enforced",
		ChecksPassed:   true,
		Passed:         true,
	}
	gates := codexContinueGateReport{
		Phase: 1,
		Checks: []gateCheck{
			{Name: "antipattern", Passed: true, Detail: "no debug artifacts found"},
			{Name: "secrets", Passed: true, Detail: "no exposed secrets found"},
		},
		Passed: true,
	}
	workerFlow := []codexContinueWorkerFlowStep{
		{
			Stage: "review", Caste: "watcher", Name: "Keen-12", Status: "completed",
			Summary:         "verified the phase",
			Recommendations: []string{"add a regression test for the edge case"},
		},
	}
	housekeeping := signalHousekeepingResult{
		TotalSignals: 4, ActiveBefore: 3, ActiveAfter: 2, ExpiredByTime: 1, Updated: 1,
	}
	consolidation := map[string]interface{}{
		"ran": true, "reason": "phase advanced", "promotion_candidates": 1, "queen_eligible": 1,
	}

	return map[string]interface{}{
		"advanced":             true,
		"completed":            false,
		"partial_success":      false,
		"current_phase":        2,
		"continued_phase":      1,
		"continued_phase_name": "Ship the thing",
		"next_phase":           2,
		"next_phase_name":      "Ship the next thing",
		"state":                colony.StateBUILT,
		"next":                 "aether continue",
		"review_depth":         "standard",
		"blocked":              false,
		"verification":         verification,
		"assessment": codexContinueAssessment{
			Phase: 1, VerificationPassed: true, PositiveEvidence: true, Passed: true,
			Summary: "Phase 1 verified and advanced",
		},
		"task_evidence":       []codexContinueTaskAssessment{{TaskID: "1", Goal: "ship it", Outcome: "done", Verified: true}},
		"gates":               gates,
		"verification_report": "build/phase-1/verification.json",
		"gate_report":         "build/phase-1/gates.json",
		"review_report":       "build/phase-1/review.json",
		"continue_report":     "build/phase-1/continue.json",
		"closed_workers":      []string{"Keen-12"},
		"worker_flow":         workerFlow,
		"operational_issues":  []string{"one worker reported a slow test run"},
		"recovery":            codexContinueRecoveryPlan{},
		"reconciled_tasks":    []string{},
		"signal_housekeeping": housekeeping,
		"consolidation":       consolidation,
	}
}

// wrapperParityBlockedResult mirrors finalizeBlockedExternalContinue's own
// result map shape the same way.
func wrapperParityBlockedResult() map[string]interface{} {
	verification := codexContinueVerificationReport{
		Phase: 1,
		Steps: []codexVerificationStep{
			{Name: "build", Passed: true},
			{Name: "tests", Passed: false, Summary: "2 of 12 failed"},
		},
		Passed: false,
	}
	gates := codexContinueGateReport{
		Phase: 1,
		Checks: []gateCheck{
			{Name: "antipattern", Passed: false, Detail: "debug print left in", FixHint: "remove the fmt.Println"},
		},
		Passed:         false,
		BlockingIssues: []string{"debug print left in"},
	}
	workerFlow := []codexContinueWorkerFlowStep{
		{Stage: "review", Caste: "watcher", Name: "Keen-12", Status: "blocked", Summary: "found a debug print"},
	}

	return map[string]interface{}{
		"advanced":             false,
		"blocked":              true,
		"partial_success":      false,
		"current_phase":        1,
		"phase_name":           "Ship the thing",
		"continued_phase":      1,
		"continued_phase_name": "Ship the thing",
		"state":                colony.StateEXECUTING,
		"next":                 "aether unblock --dispatch",
		"review_depth":         "standard",
		"verification":         verification,
		"assessment": codexContinueAssessment{
			Phase: 1, VerificationPassed: false, Passed: false,
			Summary: "Verification or review blocked phase 1: debug print left in",
		},
		"task_evidence":       []codexContinueTaskAssessment{},
		"gates":               gates,
		"verification_report": "build/phase-1/verification.json",
		"gate_report":         "build/phase-1/gates.json",
		"continue_report":     "build/phase-1/continue.json",
		"worker_flow":         workerFlow,
		"operational_issues":  []string{},
		"recovery":            codexContinueRecoveryPlan{},
		"reconciled_tasks":    []string{},
		"blocking_issues":     []string{"debug print left in"},
	}
}

// wrapperParityPlanFinalizeResult builds the typed result runCodexPlanFinalize
// (and runCodexPlanWithOptions) produce for a completed plan
// (cmd/codex_plan_finalize.go:444-478) -- confidence and plan_revision are
// the real Go structs both construction paths actually store there, never
// hand-typed nested maps.
func wrapperParityPlanFinalizeResult() map[string]interface{} {
	phases := []colony.Phase{
		{ID: 1, Name: "Ship the thing", SuccessCriteria: []string{"it ships"}},
	}
	confidence := codexPlanConfidence{Knowledge: 90, Requirements: 85, Risks: 80, Dependencies: 82, Effort: 88, Overall: 85}
	planningLoop := codexPlanningLoop{TargetConfidence: 80, MaxIterations: 3, Iterations: 1, StopReason: "target confidence reached", FinalConfidence: 85}
	return map[string]interface{}{
		"planned":                   true,
		"existing_plan":             false,
		"refreshed":                 false,
		"goal":                      "Ship the thing",
		"phases":                    phases,
		"count":                     len(phases),
		"depth":                     "standard",
		"granularity":               "standard",
		"granularity_min":           3,
		"granularity_max":           7,
		"confidence":                confidence,
		"planning_loop":             planningLoop,
		"planning_run_id":           "run-1",
		"iteration":                 1,
		"evidence_hash":             "abc123",
		"planning_dir":              ".aether/data/planning/run-1",
		"planning_files":            []string{"scout.json", "route-setter.json"},
		"plan_artifact":             "phase-plan.json",
		"research_failed_phases":    []string{},
		"research_warning":          "",
		"dispatches":                []map[string]interface{}{},
		"dispatch_mode":             "host",
		"artifact_source":           "wrapper",
		"plan_source":               "route-setter",
		"gaps":                      []string{},
		"survey_docs":               []string{},
		"unresolved_clarifications": 0,
		"planning_warning":          "",
		"next":                      "aether build 1",
	}
}

// wrapperParityPlanIterationResult mirrors the "requires_next_iteration"
// mid-loop result persistIntermediatePlanningIteration produces
// (cmd/codex_plan_finalize.go:863-886) -- a partial/iterating plan run that
// also carries confidence and planning_loop, so the same normalisation must
// hold here too.
func wrapperParityPlanIterationResult() map[string]interface{} {
	confidence := codexPlanConfidence{Knowledge: 70, Requirements: 65, Risks: 60, Dependencies: 62, Effort: 68, Overall: 65}
	planningLoop := codexPlanningLoop{TargetConfidence: 80, MaxIterations: 3, Iterations: 1, StopReason: "", FinalConfidence: 65}
	return map[string]interface{}{
		"planned":                 false,
		"iteration_completed":     true,
		"requires_next_iteration": true,
		"planning_run_id":         "run-1",
		"iteration":               1,
		"goal":                    "Ship the thing",
		"depth":                   "standard",
		"planning_depth":          "balanced",
		"confidence":              confidence,
		"planning_loop":           planningLoop,
		"dispatches":              []map[string]interface{}{},
		"dispatch_mode":           "host",
		"artifact_source":         "planning-iteration",
		"plan_source":             "route-setter",
		"gaps":                    []string{"gather more evidence"},
		"selected_gaps":           []string{"gather more evidence"},
		"evidence_hash":           "def456",
		"planning_dir":            ".aether/data/planning/run-1",
		"iteration_state":         ".aether/data/planning/run-1/iteration-state.json",
		"planning_warning":        "Planning iteration validated, but confidence target has not been reached; final colony plan was not written.",
		"next":                    "aether host plan --depth standard --planning-depth balanced --target 80 --max-iterations 3",
	}
}

// wrapperParitySealResult mirrors completeSealRuntime's own result map
// (cmd/codex_workflow_cmds.go:808-819) for a plain, unforced seal.
func wrapperParitySealResult() map[string]interface{} {
	return map[string]interface{}{
		"sealed":    true,
		"milestone": "Crowned Anthill",
		"summary":   "CROWNED-ANTHILL.md",
		"next":      "aether entomb",
	}
}

// wrapperParityForceSealedResult mirrors completeSealRuntime's result map
// when override.overrodeAnything() is true (cmd/codex_workflow_cmds.go:814-818).
func wrapperParityForceSealedResult() map[string]interface{} {
	return map[string]interface{}{
		"sealed":              true,
		"milestone":           "Crowned Anthill",
		"summary":             "CROWNED-ANTHILL.md",
		"next":                "aether entomb",
		"force_sealed":        true,
		"force_reason":        "work finished outside the colony",
		"unverified_phases":   []string{"3"},
		"overridden_blockers": 1,
	}
}

// wrapperParitySealConfirmationPendingResult mirrors
// runSealConfirmationGate's own "not proceeding" result
// (cmd/seal_confirmation.go:323-329) -- seal never actually sealed, so there
// is nothing renderSealVisual could honestly show.
func wrapperParitySealConfirmationPendingResult() map[string]interface{} {
	return map[string]interface{}{
		"sealed":                      false,
		"awaiting_owner_confirmation": true,
		"named_problems":              []string{"2 checks failing"},
		"question":                    "Finish anyway with 2 check(s) failing: 2 checks failing?",
		"next":                        `aether decision-answer --question "..." --answer "yes" --source seal-force-confirmation`,
	}
}

// roundTripToMap is the CLAUDE.md-mandated fixture step: marshal the typed
// result the finalizer builds, then unmarshal it into map[string]interface{}
// -- the shape a completion file actually is.
func roundTripToMap(t *testing.T, typed map[string]interface{}) map[string]interface{} {
	t.Helper()
	data, err := json.Marshal(typed)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	var asMap map[string]interface{}
	if err := json.Unmarshal(data, &asMap); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	return asMap
}

// firstDiffLine helps a failing assertion point at the actual divergence
// instead of dumping two whole screens at the reader.
func firstDiffLine(a, b string) string {
	linesA := strings.Split(a, "\n")
	linesB := strings.Split(b, "\n")
	for i := 0; i < len(linesA) || i < len(linesB); i++ {
		var la, lb string
		if i < len(linesA) {
			la = linesA[i]
		}
		if i < len(linesB) {
			lb = linesB[i]
		}
		if la != lb {
			return fmt.Sprintf("line %d:\n  direct : %s\n  closeout: %s", i, la, lb)
		}
	}
	return "(no differing line found, but strings are not equal)"
}

func TestWrapperPathRendersSameCeremonyAsDirectPath(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	state := wrapperParityColonyState()

	t.Run("continue advance", func(t *testing.T) {
		typed := wrapperParityAdvanceResult()
		phase := colony.Phase{ID: 1, Name: "Ship the thing"}
		nextPhase := colony.Phase{ID: 2, Name: "Ship the next thing"}
		housekeeping := typed["signal_housekeeping"].(signalHousekeepingResult)

		directOutput := renderContinueVisual(state, phase, &housekeeping, false, &nextPhase, typed, colony.VerificationDepthStandard)

		asMap := roundTripToMap(t, typed)
		closeoutOutput, handled := closeoutDirectVisual("continue", map[string]interface{}{"completion_raw": asMap}, state)
		if !handled {
			t.Fatalf("closeoutDirectVisual reported not-handled for a resolvable continue advance result")
		}
		if directOutput != closeoutOutput {
			t.Fatalf("chat-path closeout diverged from the direct continue render:\n%s", firstDiffLine(directOutput, closeoutOutput))
		}
	})

	t.Run("continue blocked", func(t *testing.T) {
		typed := wrapperParityBlockedResult()
		phase := colony.Phase{ID: 1, Name: "Ship the thing"}

		directOutput := renderContinueBlockedVisual(state, phase, typed, colony.VerificationDepthStandard)

		asMap := roundTripToMap(t, typed)
		closeoutOutput, handled := closeoutDirectVisual("continue", map[string]interface{}{"completion_raw": asMap}, state)
		if !handled {
			t.Fatalf("closeoutDirectVisual reported not-handled for a resolvable continue blocked result")
		}
		if directOutput != closeoutOutput {
			t.Fatalf("chat-path closeout diverged from the direct continue-blocked render:\n%s", firstDiffLine(directOutput, closeoutOutput))
		}
	})

	t.Run("unresolvable continued phase falls back rather than rendering a half-screen", func(t *testing.T) {
		typed := wrapperParityAdvanceResult()
		typed["continued_phase"] = 999 // not present in state.Plan.Phases
		asMap := roundTripToMap(t, typed)

		_, handled := closeoutDirectVisual("continue", map[string]interface{}{"completion_raw": asMap}, state)
		if handled {
			t.Fatalf("closeoutDirectVisual reported handled for a continued_phase that cannot be resolved from state")
		}
	})

	t.Run("a continue-shaped result is not handled by another workflow's branch", func(t *testing.T) {
		// "plan" and "seal" ARE wired (this plan); "build" is not (a later
		// plan's scope). A continue result carries neither "goal" nor
		// "sealed", so all three branches must correctly report not-handled
		// rather than rendering a half-screen from a foreign shape.
		typed := wrapperParityAdvanceResult()
		asMap := roundTripToMap(t, typed)

		for _, workflow := range []string{"plan", "seal", "build"} {
			_, handled := closeoutDirectVisual(workflow, map[string]interface{}{"completion_raw": asMap}, state)
			if handled {
				t.Fatalf("closeoutDirectVisual reported handled for workflow %q on a continue-shaped result", workflow)
			}
		}
	})

	t.Run("plan completed", func(t *testing.T) {
		typed := wrapperParityPlanFinalizeResult()

		directOutput := renderPlanVisual(typed)

		asMap := roundTripToMap(t, typed)
		closeoutOutput, handled := closeoutDirectVisual("plan", map[string]interface{}{"completion_raw": asMap}, state)
		if !handled {
			t.Fatalf("closeoutDirectVisual reported not-handled for a resolvable completed plan result")
		}
		if directOutput != closeoutOutput {
			t.Fatalf("chat-path closeout diverged from the direct plan render:\n%s", firstDiffLine(directOutput, closeoutOutput))
		}
	})

	t.Run("plan mid-loop iteration", func(t *testing.T) {
		typed := wrapperParityPlanIterationResult()

		directOutput := renderPlanVisual(typed)

		asMap := roundTripToMap(t, typed)
		closeoutOutput, handled := closeoutDirectVisual("plan", map[string]interface{}{"completion_raw": asMap}, state)
		if !handled {
			t.Fatalf("closeoutDirectVisual reported not-handled for a resolvable mid-loop plan result")
		}
		if directOutput != closeoutOutput {
			t.Fatalf("chat-path closeout diverged from the direct mid-loop plan render:\n%s", firstDiffLine(directOutput, closeoutOutput))
		}
	})

	t.Run("seal completed", func(t *testing.T) {
		typed := wrapperParitySealResult()

		directOutput := renderSealVisual(typed, state, stringValue(typed["summary"]))

		asMap := roundTripToMap(t, typed)
		closeoutOutput, handled := closeoutDirectVisual("seal", map[string]interface{}{"completion_raw": asMap}, state)
		if !handled {
			t.Fatalf("closeoutDirectVisual reported not-handled for a resolvable completed seal result")
		}
		if directOutput != closeoutOutput {
			t.Fatalf("chat-path closeout diverged from the direct seal render:\n%s", firstDiffLine(directOutput, closeoutOutput))
		}
	})

	t.Run("seal force-sealed with overrides", func(t *testing.T) {
		typed := wrapperParityForceSealedResult()

		directOutput := renderSealVisual(typed, state, stringValue(typed["summary"]))

		asMap := roundTripToMap(t, typed)
		closeoutOutput, handled := closeoutDirectVisual("seal", map[string]interface{}{"completion_raw": asMap}, state)
		if !handled {
			t.Fatalf("closeoutDirectVisual reported not-handled for a resolvable force-sealed result")
		}
		if directOutput != closeoutOutput {
			t.Fatalf("chat-path closeout diverged from the direct force-sealed render:\n%s", firstDiffLine(directOutput, closeoutOutput))
		}
	})

	t.Run("seal awaiting owner confirmation is not rendered as sealed on either path", func(t *testing.T) {
		typed := wrapperParitySealConfirmationPendingResult()
		asMap := roundTripToMap(t, typed)

		_, handled := closeoutDirectVisual("seal", map[string]interface{}{"completion_raw": asMap}, state)
		if handled {
			t.Fatalf("closeoutDirectVisual reported handled for a seal result still awaiting the owner's confirmation")
		}
	})
}

// TestCloseoutRefusesToHalfRenderAnIncompletePayload proves that a completion
// map missing the key a branch needs to safely render (plan's "goal", seal's
// "sealed") reports not-handled -- so the caller falls back to the generic
// renderer -- rather than rendering a partial or misleading screen (e.g. a
// "Colony sealed" screen for something that never sealed).
func TestCloseoutRefusesToHalfRenderAnIncompletePayload(t *testing.T) {
	state := wrapperParityColonyState()

	t.Run("plan missing goal", func(t *testing.T) {
		incomplete := map[string]interface{}{"phases": []interface{}{}, "count": 0}
		_, handled := closeoutDirectVisual("plan", map[string]interface{}{"completion_raw": incomplete}, state)
		if handled {
			t.Fatalf("closeoutDirectVisual reported handled for a plan payload missing \"goal\"")
		}
	})

	t.Run("plan empty payload", func(t *testing.T) {
		_, handled := closeoutDirectVisual("plan", map[string]interface{}{"completion_raw": map[string]interface{}{}}, state)
		if handled {
			t.Fatalf("closeoutDirectVisual reported handled for an empty plan payload")
		}
	})

	t.Run("seal missing sealed flag", func(t *testing.T) {
		incomplete := map[string]interface{}{"milestone": "Crowned Anthill", "next": "aether entomb"}
		_, handled := closeoutDirectVisual("seal", map[string]interface{}{"completion_raw": incomplete}, state)
		if handled {
			t.Fatalf("closeoutDirectVisual reported handled for a seal payload missing \"sealed\"")
		}
	})

	t.Run("seal sealed is false", func(t *testing.T) {
		incomplete := map[string]interface{}{"sealed": false, "next": "aether continue"}
		_, handled := closeoutDirectVisual("seal", map[string]interface{}{"completion_raw": incomplete}, state)
		if handled {
			t.Fatalf("closeoutDirectVisual reported handled for a seal payload with sealed=false")
		}
	})
}

// TestCloseoutUnhandledWorkflowsKeepTheGenericRenderer proves the tracer did
// not change anything it was not meant to change: build, colonize, swarm and
// status must all still fall through to the pre-existing generic
// renderCeremonyCloseoutVisual, because closeoutDirectVisual has not been
// wired for them.
func TestCloseoutUnhandledWorkflowsKeepTheGenericRenderer(t *testing.T) {
	state := wrapperParityColonyState()

	representative := map[string]map[string]interface{}{
		"build": {
			"completion_raw": map[string]interface{}{
				"advanced": true, "current_phase": 1, "dispatches": []interface{}{
					map[string]interface{}{"name": "Mason-67", "caste": "builder", "status": "completed"},
				},
			},
		},
		"colonize": {
			"completion_raw": map[string]interface{}{
				"phases_generated": 3, "surveyors": []interface{}{"surveyor-nest"},
			},
		},
		"swarm": {
			"completion_raw": map[string]interface{}{
				"bug": "login fails", "workers": []interface{}{
					map[string]interface{}{"name": "Roam-90", "caste": "scout", "status": "completed"},
				},
			},
		},
		"status": {
			"completion_raw": map[string]interface{}{
				"state": "executing", "current_phase": 1,
			},
		},
	}

	for workflow, result := range representative {
		t.Run(workflow, func(t *testing.T) {
			_, handled := closeoutDirectVisual(workflow, result, state)
			if handled {
				t.Fatalf("closeoutDirectVisual reported handled=true for workflow %q, which must still use the generic renderer", workflow)
			}
		})
	}
}

// TestCloseoutContinueCostLineStaysLastAndSingle drives the full wrapper
// closeout path (renderCeremonyCloseout, exactly what
// `aether ceremony closeout --workflow continue` runs) with a real spend
// ledger present, and asserts the cost block appears exactly once and is the
// last thing on the screen -- the Phase 196 ruling still holds once the
// closeout body comes from closeoutDirectVisual instead of the generic
// renderer.
func TestCloseoutContinueCostLineStaysLastAndSingle(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	seedCostLineColonyForTest(t)

	typed := wrapperParityAdvanceResult()
	// Match the phase seedCostLineColonyForTest recorded a ledger for.
	typed["continued_phase"] = 1
	typed["next_phase"] = 2
	envelope := map[string]interface{}{"ok": true, "result": typed}
	completionFile := writeCeremonyTestJSON(t, envelope)

	_, visual := renderCeremonyCloseout("continue", completionFile)
	clean := stripANSI(visual)

	if got := countCostLineBlocks(visual); got != 1 {
		t.Fatalf("the continue closeout carries %d cost line block(s), want exactly 1:\n%s", got, visual)
	}

	block := stripANSI(renderSpendCostLine(1))
	if block == "" {
		t.Fatalf("test setup produced no cost block to compare against -- seedCostLineColonyForTest may have changed shape")
	}
	if !strings.HasSuffix(strings.TrimRight(clean, "\n"), strings.TrimRight(block, "\n")) {
		t.Fatalf("the cost line block is not the last thing on the continue closeout screen:\n%s", visual)
	}
}
