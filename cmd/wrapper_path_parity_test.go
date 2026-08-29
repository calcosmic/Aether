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

	t.Run("a workflow this bridge does not yet wire is not handled", func(t *testing.T) {
		typed := wrapperParityAdvanceResult()
		asMap := roundTripToMap(t, typed)

		for _, workflow := range []string{"plan", "seal", "build"} {
			_, handled := closeoutDirectVisual(workflow, map[string]interface{}{"completion_raw": asMap}, state)
			if handled {
				t.Fatalf("closeoutDirectVisual reported handled for workflow %q, which this plan does not wire", workflow)
			}
		}
	})
}
