package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// Force seal — the owner's escape hatch for work finished OUTSIDE the colony
// or a colony wedged on its own gates: `aether seal --force --reason "why"`
// files the project away and records exactly what was skipped. Never silent:
// forcing past anything requires a written reason, and the override is
// permanent history (event + CROWNED-ANTHILL.md).

// setupStuckColonyStore builds the wedge: phases the colony never verified.
func setupStuckColonyStore(t *testing.T) (string, func(args []string) (string, string)) {
	t.Helper()
	s, tmpDir := setupSealTestStore(t)

	goal := "Test colony goal"
	state := colony.ColonyState{
		Goal:         &goal,
		CurrentPhase: 2,
		State:        colony.StateREADY,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Scaffold", Status: colony.PhaseCompleted},
				{ID: 2, Name: "Engine", Status: colony.PhaseInProgress},
				{ID: 3, Name: "Polish", Status: colony.PhasePending},
			},
		},
		Memory: colony.Memory{PhaseLearnings: []colony.PhaseLearning{}},
		Events: []string{},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}
	return tmpDir, func(args []string) (string, string) {
		return runSealCmd(t, s, tmpDir, args)
	}
}

func TestForceSealOverridesIncompletePhases(t *testing.T) {
	tmpDir, run := setupStuckColonyStore(t)

	run([]string{"--force", "--reason", "finished the engine by hand outside the colony"})

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatal(err)
	}
	if state.State != colony.StateCOMPLETED {
		t.Fatalf("force seal did not seal the colony: state=%s", state.State)
	}

	// The override is a named event, not a footnote.
	forcedEvent := ""
	for _, event := range state.Events {
		if strings.Contains(event, "sealed_forced") {
			forcedEvent = event
		}
	}
	if forcedEvent == "" {
		t.Fatalf("no sealed_forced event recorded: %v", state.Events)
	}
	if !strings.Contains(forcedEvent, "2 unverified phase(s)") || !strings.Contains(forcedEvent, "finished the engine by hand") {
		t.Fatalf("forced event does not carry the honest record: %q", forcedEvent)
	}

	// CROWNED-ANTHILL.md names the skipped phases and the reason, forever.
	summary, err := os.ReadFile(filepath.Join(tmpDir, ".aether", "CROWNED-ANTHILL.md"))
	if err != nil {
		t.Fatalf("read seal summary: %v", err)
	}
	text := string(summary)
	for _, want := range []string{
		"Owner Override (Force Seal)",
		"finished the engine by hand outside the colony",
		"phase 2 — Engine",
		"phase 3 — Polish",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("CROWNED-ANTHILL.md missing override record %q:\n%s", want, text)
		}
	}
}

func TestForceSealRequiresReason(t *testing.T) {
	_, run := setupStuckColonyStore(t)

	out, errOut := run([]string{"--force"})
	combined := out + errOut
	if !strings.Contains(combined, "reason is required") {
		t.Fatalf("force without a reason was not refused with an explanation: %s", combined)
	}
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatal(err)
	}
	if state.State == colony.StateCOMPLETED {
		t.Fatalf("colony sealed despite the missing reason")
	}
}

func TestSealStillRefusesWithoutForce(t *testing.T) {
	_, run := setupStuckColonyStore(t)

	out, errOut := run(nil)
	combined := out + errOut
	if !strings.Contains(combined, "all phases must be completed") {
		t.Fatalf("incomplete phases did not refuse the seal: %s", combined)
	}
	// The refusal teaches the escape hatch instead of leaving the owner
	// stuck with no way forward.
	if !strings.Contains(combined, "--force --reason") {
		t.Fatalf("refusal does not name the force-seal escape hatch: %s", combined)
	}
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatal(err)
	}
	if state.State == colony.StateCOMPLETED {
		t.Fatalf("colony sealed without force")
	}
}

// TestNormalSealHasNoOverrideRecord: a clean seal must not carry override
// bookkeeping — the record exists only when something was actually skipped.
func TestNormalSealHasNoOverrideRecord(t *testing.T) {
	s, tmpDir := setupSealTestStore(t)

	runSealCmd(t, s, tmpDir, nil)

	var state colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatal(err)
	}
	if state.State != colony.StateCOMPLETED {
		t.Fatalf("clean seal did not complete: %s", state.State)
	}
	for _, event := range state.Events {
		if strings.Contains(event, "sealed_forced") {
			t.Fatalf("clean seal recorded a forced event: %q", event)
		}
	}
	summary, err := os.ReadFile(filepath.Join(tmpDir, ".aether", "CROWNED-ANTHILL.md"))
	if err != nil {
		t.Fatalf("read seal summary: %v", err)
	}
	if strings.Contains(string(summary), "Owner Override") {
		t.Fatalf("clean seal summary carries an override section:\n%s", summary)
	}
}

// TestForceSealWrapperAsksFirst pins the wrapper contract: forcing is the
// user's explicit choice with their own reason — never the wrapper's.
func TestForceSealWrapperAsksFirst(t *testing.T) {
	for _, path := range []string{"../.claude/commands/ant/seal.md", "../.claude/commands/ant-seal.md", "../.opencode/commands/ant/seal.md"} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		text := string(raw)
		for _, anchor := range []string{
			"AskUserQuestion",
			"Force the seal",
			"NEVER add `--force` on your own initiative",
			"never invent the\nreason",
		} {
			if !strings.Contains(text, anchor) {
				t.Fatalf("%s lost the force-seal consent anchor %q", path, anchor)
			}
		}
	}
}
