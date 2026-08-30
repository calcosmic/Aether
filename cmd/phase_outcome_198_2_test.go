package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// Phase 198.2 plan 05 (WIRE-07, D-11): outcome.md must carry, word for word,
// the closing screen the owner saw when a check finished -- produced by the
// same D-12 renderer (closeoutContinueDirectVisual) fed the same finalizer
// result map, never a second hand-derivation.

// TestOutcomeFileMatchesTheClosingScreen builds a passing continue result the
// way a real finalize builds it (typed structs, not a JSON-round-tripped
// map -- writePhaseOutcomeDocument is called in-process on both lanes, never
// on a completion-file round trip), renders the closing screen independently
// via the same renderer writePhaseOutcomeDocument itself calls, and asserts
// the persisted file equals that stripped screen byte-for-byte -- equality,
// not containment, so the writer building even one line of its own prose
// would fail this test.
func TestOutcomeFileMatchesTheClosingScreen(t *testing.T) {
	saveGlobalsCmd(t)
	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	state := colony.ColonyState{
		Version:      "3.0",
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{Phases: []colony.Phase{
			{ID: 1, Name: "Ship the thing", Status: colony.PhaseCompleted},
		}},
	}

	verification := codexContinueVerificationReport{
		Steps:        []codexVerificationStep{{Name: "build", Passed: true, Duration: 1.0}},
		ChecksPassed: true,
	}
	gates := codexContinueGateReport{Checks: []gateCheck{{Name: "verification_steps_passed", Passed: true}}}

	result := map[string]interface{}{
		"advanced":             true,
		"completed":            false,
		"current_phase":        1,
		"continued_phase":      1,
		"continued_phase_name": "Ship the thing",
		"state":                colony.StateEXECUTING,
		"next":                 "aether build 2",
		"review_depth":         "standard",
		"verification":         verification,
		"gates":                gates,
	}

	wantVisual, ok := closeoutContinueDirectVisual(result, state)
	if !ok {
		t.Fatalf("fixture broken: closeoutContinueDirectVisual could not resolve phase from the fixture")
	}
	want := stripPhaseOutcomeANSI(wantVisual)

	if err := writePhaseOutcomeDocument(1, result, state); err != nil {
		t.Fatalf("writePhaseOutcomeDocument: %v", err)
	}

	got, err := s.ReadFile(continuePlanArtifactsPath(1, "outcome.md"))
	if err != nil {
		t.Fatalf("read outcome.md: %v", err)
	}
	if string(got) != want {
		t.Fatalf("outcome.md does not equal the closing screen:\n%s", firstDiffLine(want, string(got)))
	}
}

// TestBlockedPhaseStillGetsAnOutcome proves a blocked check also gets its
// closing screen persisted -- a phase that failed is exactly the phase whose
// outcome the next one most needs (D-11).
func TestBlockedPhaseStillGetsAnOutcome(t *testing.T) {
	saveGlobalsCmd(t)
	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	state := colony.ColonyState{
		Version:      "3.0",
		State:        colony.StateBUILT,
		CurrentPhase: 1,
		Plan:         colony.Plan{Phases: []colony.Phase{{ID: 1, Name: "Ship the thing"}}},
	}

	gates := codexContinueGateReport{Checks: []gateCheck{
		{Name: "verification_steps_passed", Passed: false, FixHint: "fix the failing build/test check and run aether continue"},
	}}
	result := map[string]interface{}{
		"advanced":             false,
		"blocked":              true,
		"current_phase":        1,
		"continued_phase":      1,
		"continued_phase_name": "Ship the thing",
		"state":                colony.StateBUILT,
		"next":                 "aether unblock --dispatch",
		"review_depth":         "standard",
		"gates":                gates,
		"blocking_issues":      []string{"build failed"},
	}

	wantVisual, ok := closeoutContinueDirectVisual(result, state)
	if !ok {
		t.Fatalf("fixture broken: closeoutContinueDirectVisual could not resolve the blocked phase")
	}
	want := stripPhaseOutcomeANSI(wantVisual)

	if err := writePhaseOutcomeDocument(1, result, state); err != nil {
		t.Fatalf("writePhaseOutcomeDocument: %v", err)
	}

	got, err := s.ReadFile(continuePlanArtifactsPath(1, "outcome.md"))
	if err != nil {
		t.Fatalf("read outcome.md: %v", err)
	}
	if string(got) != want {
		t.Fatalf("blocked outcome.md does not equal the blocked closing screen:\n%s", firstDiffLine(want, string(got)))
	}
	if !strings.Contains(string(got), "build failed") {
		t.Errorf("blocked outcome.md missing the blocking issue text, got:\n%s", string(got))
	}
}

// TestUnresolvableResultWritesNoOutcomeFile proves that when the renderer
// cannot resolve a phase from the given inputs, writePhaseOutcomeDocument
// writes nothing and returns a non-nil error rather than an empty file.
func TestUnresolvableResultWritesNoOutcomeFile(t *testing.T) {
	saveGlobalsCmd(t)
	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	state := colony.ColonyState{
		Version: "3.0",
		Plan:    colony.Plan{Phases: []colony.Phase{{ID: 1, Name: "Ship the thing"}}},
	}
	// Phase 99 does not exist in state -- colonyPhaseByID cannot resolve it,
	// so closeoutContinueDirectVisual must report handled=false.
	result := map[string]interface{}{"advanced": true, "completed": false}

	err := writePhaseOutcomeDocument(99, result, state)
	if err == nil {
		t.Fatalf("expected a non-nil error for an unresolvable phase, got nil")
	}

	path := filepath.Join(s.BasePath(), filepath.FromSlash(continuePlanArtifactsPath(99, "outcome.md")))
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Errorf("expected no outcome.md to be written for an unresolvable phase, but found one at %s", path)
	}
}

// TestOutcomeFileIsReplacedNotAppended proves calling
// writePhaseOutcomeDocument twice for the same phase replaces the first
// outcome rather than appending to it -- one outcome per phase, ever.
func TestOutcomeFileIsReplacedNotAppended(t *testing.T) {
	saveGlobalsCmd(t)
	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	state := colony.ColonyState{
		Version:      "3.0",
		CurrentPhase: 1,
		Plan:         colony.Plan{Phases: []colony.Phase{{ID: 1, Name: "Ship the thing"}}},
	}

	firstResult := map[string]interface{}{
		"advanced":             false,
		"blocked":              true,
		"continued_phase":      1,
		"continued_phase_name": "Ship the thing",
		"state":                colony.StateBUILT,
		"next":                 "aether unblock --dispatch",
		"blocking_issues":      []string{"first failure"},
	}
	if err := writePhaseOutcomeDocument(1, firstResult, state); err != nil {
		t.Fatalf("first write: %v", err)
	}

	secondResult := map[string]interface{}{
		"advanced":             true,
		"completed":            false,
		"continued_phase":      1,
		"continued_phase_name": "Ship the thing",
		"state":                colony.StateEXECUTING,
		"next":                 "aether build 2",
	}
	if err := writePhaseOutcomeDocument(1, secondResult, state); err != nil {
		t.Fatalf("second write: %v", err)
	}

	got, err := s.ReadFile(continuePlanArtifactsPath(1, "outcome.md"))
	if err != nil {
		t.Fatalf("read outcome.md: %v", err)
	}
	if strings.Contains(string(got), "first failure") {
		t.Errorf("outcome.md still carries the first call's text -- it was appended, not replaced:\n%s", string(got))
	}

	wantVisual, ok := closeoutContinueDirectVisual(secondResult, state)
	if !ok {
		t.Fatalf("fixture broken: could not render the second result")
	}
	if string(got) != stripPhaseOutcomeANSI(wantVisual) {
		t.Errorf("outcome.md after replacement does not equal the second call's closing screen:\n%s", firstDiffLine(stripPhaseOutcomeANSI(wantVisual), string(got)))
	}
}
