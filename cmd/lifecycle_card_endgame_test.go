package cmd

// Phase 197 plan 06, task 2 -- sealing, recovering and status end with the
// card.
//
// These three are where the owner arrives when something has gone sideways,
// or when a project is finished. Status also carries two overrides that must
// reach both the screen and the machine-readable answer from one decision:
// workers still running, and a guided action (an open flag, active research,
// an unacknowledged failure) that needs attention first. Recover's own
// next-step function was the last hand-rolled decider left in the runtime.

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/colony"
)

// ---------------------------------------------------------------------------
// Sealing
// ---------------------------------------------------------------------------

// sealedEndgameState is a fully finished, just-sealed project, written the
// way runSeal itself writes one: State COMPLETED, the final milestone set.
func sealedEndgameState(t *testing.T) colony.ColonyState {
	t.Helper()
	return normalizedFixtureState(t, colony.ColonyState{
		Version:   "3.0",
		Goal:      fixtureGoal("Ship the billing rewrite"),
		State:     colony.StateCOMPLETED,
		Milestone: "Crowned Anthill",
		Plan: colony.Plan{Phases: []colony.Phase{
			fixturePhase(1, "Foundations", colony.PhaseCompleted),
			fixturePhase(2, "Billing engine", colony.PhaseCompleted),
		}},
	})
}

// sealCardRun drives renderSealVisual over a genuinely sealed project, the
// same way runSeal calls it after saving that state to disk.
func sealCardRun(t *testing.T) lifecycleRun {
	t.Helper()
	newNextActionFixtureStore(t)
	t.Setenv("AETHER_PLATFORM", "codex")
	state := sealedEndgameState(t)
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("write the fixture project: %v", err)
	}
	result := map[string]interface{}{}
	closeLifecycleRun(result, state, "signing the project off as finished")

	run := lifecycleRun{envelope: result}
	run.answer, _ = nextActionFromResult(result)
	run.card = stripANSI(renderNextActionCardForPlatform(run.answer, "codex"))
	run.visual = stripANSI(renderSealVisual(result, state, "/tmp/CROWNED-ANTHILL.md"))
	return run
}

func TestSealEndsWithTheCard(t *testing.T) {
	run := sealCardRun(t)
	if !strings.Contains(run.visual, run.card) {
		t.Errorf("seal does not end with the shared card.\n--- the card the resolver produced ---\n%s\n--- what was printed ---\n%s",
			run.card, run.visual)
	}
	for _, keep := range []string{
		"The project (this colony) stands crowned and finished (sealed).",
		"The coordinator (Queen) that decides your team keeps this wisdom in QUEEN.md for next time.",
	} {
		if !strings.Contains(run.visual, keep) {
			t.Errorf("seal lost %q from above the closing block", keep)
		}
	}
}

// TestSealCardExplainsFinishingAndArchiving is S-05 applied specifically:
// sealing is the one place the owner is most likely to be told something they
// cannot act on, so the card must say in ordinary words what finishing a
// project means and what archiving it would do.
func TestSealCardExplainsFinishingAndArchiving(t *testing.T) {
	run := sealCardRun(t)
	if !strings.Contains(run.card, "aether entomb") {
		t.Fatalf("the sealed project's card does not recommend entomb at all:\n%s", run.card)
	}
	if !strings.Contains(run.card, "archive") {
		t.Errorf("the card recommends entomb without saying what archiving it would do:\n%s", run.card)
	}
	if violations := untranslatedRepoWords(run.card); len(violations) > 0 {
		t.Errorf("the seal card uses repo-invented words without explaining them:\n  %s\n%s",
			strings.Join(violations, "\n  "), run.card)
	}
}

// TestSealEnvelopeMatchesCard (assertEnvelopeMatchesCard(t, "sealing",
// sealCardRun(t))) is now a full subset of
// TestEveryLifecycleCommandEndsWithNextAction's "sealing/finishing" subtest
// (cmd/lifecycle_next_action_coverage_test.go), which drives the identical
// sealCardRun fixture and compares the same screen-command-vs-envelope-command
// pair -- deleted here rather than kept as a duplicate (Phase 197 plan 07).

// ---------------------------------------------------------------------------
// Recovering
// ---------------------------------------------------------------------------

func recoverEndgameState(t *testing.T) colony.ColonyState {
	t.Helper()
	started := time.Now().UTC().Add(-time.Hour)
	return normalizedFixtureState(t, colony.ColonyState{
		Version:        "3.0",
		Goal:           fixtureGoal("Ship the billing rewrite"),
		State:          colony.StateEXECUTING,
		CurrentPhase:   1,
		BuildStartedAt: &started,
		Milestone:      "Open Chambers",
		Plan: colony.Plan{Phases: []colony.Phase{
			fixturePhase(1, "Foundations", colony.PhaseInProgress),
			fixturePhase(2, "Billing engine", colony.PhasePending),
		}},
	})
}

// recoverCardRun drives the real `aether recover` command (text mode) over a
// prepared project.
func recoverCardRun(t *testing.T, state colony.ColonyState) lifecycleRun {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStoreCmd(t)
	t.Cleanup(func() { _ = tmpDir })
	store = s
	t.Setenv("AETHER_PLATFORM", "codex")
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("write the fixture project: %v", err)
	}

	var buf bytes.Buffer
	stdout = &buf
	rootCmd.SetArgs([]string{"recover"})
	_ = rootCmd.Execute()

	run := lifecycleRun{}
	run.visual = stripANSI(buf.String())
	run.answer = recoverNextAction(nil, colony.ColonyState{})
	return run
}

func TestRecoverEndsWithTheCard(t *testing.T) {
	run := recoverCardRun(t, recoverEndgameState(t))
	if !strings.Contains(run.visual, "legacy-recover-migration/v1") || !strings.Contains(run.visual, "aether resume") {
		t.Fatalf("recover must be an explicit migration tombstone to canonical resume:\n%s", run.visual)
	}
	if strings.Contains(run.visual, spacedTitle("What Next")) {
		t.Errorf("the retired recover surface must not masquerade as a current shared-card lifecycle command:\n%s", run.visual)
	}
}

// TestRecoveryNextStepComesFromTheResolver is the sixth and last hand-rolled
// next-step function in the runtime: recover's own decider must return the
// one resolver's answer, fed this scan's own finding as an override.
func TestRecoveryNextStepComesFromTheResolver(t *testing.T) {
	state := recoverEndgameState(t)
	newNextActionFixtureStore(t)
	t.Setenv("AETHER_PLATFORM", "codex")
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("write the fixture project: %v", err)
	}

	issues := []HealthIssue{{Severity: "critical", Category: "missing_build_packet", Message: "No build packet"}}
	answer := recoverNextAction(issues, state)
	want := "aether build 1 --force"
	if answer.Command != want {
		t.Errorf("recover's next-step decider recommends %q for a missing build packet on phase 1; want %q",
			answer.Command, want)
	}

	// No override case: the ordinary answer for this project, resolved the
	// same way every other lifecycle card resolves it.
	plain := recoverNextAction(nil, state)
	want = resolveNextAction(lifecycleNextActionInputFixture(t, state)).Command
	if plain.Command != want {
		t.Errorf("a healthy recover scan recommends %q; the one answer for this project is %q",
			plain.Command, want)
	}
}

// lifecycleNextActionInputFixture builds the same input lifecycleNextActionForState
// builds, for comparing against recoverNextAction's own resolve.
func lifecycleNextActionInputFixture(t *testing.T, state colony.ColonyState) nextActionInput {
	t.Helper()
	return nextActionInputForState(state, "recover")
}

// ---------------------------------------------------------------------------
// Status
// ---------------------------------------------------------------------------

func runStatusCommand(t *testing.T, jsonMode bool) (visual string, envelope map[string]interface{}) {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)
	if jsonMode {
		t.Setenv("AETHER_OUTPUT_MODE", "json")
	} else {
		t.Setenv("AETHER_OUTPUT_MODE", "visual")
	}
	t.Setenv("AETHER_PLATFORM", "codex")

	var buf bytes.Buffer
	stdout = &buf
	rootCmd.SetArgs([]string{"status"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("status returned an error: %v\noutput:\n%s", err, buf.String())
	}

	if jsonMode {
		var parsed map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
			t.Fatalf("status output is not valid JSON: %v\n%s", err, buf.String())
		}
		if inner, ok := parsed["result"].(map[string]interface{}); ok {
			return "", inner
		}
		return "", parsed
	}
	return stripANSI(buf.String()), nil
}

func statusReadyState(t *testing.T) colony.ColonyState {
	t.Helper()
	return normalizedFixtureState(t, colony.ColonyState{
		Version:      "3.0",
		Goal:         fixtureGoal("Ship the billing rewrite"),
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Milestone:    "Open Chambers",
		Plan: colony.Plan{Phases: []colony.Phase{
			fixturePhase(1, "Foundations", colony.PhaseReady),
			fixturePhase(2, "Billing engine", colony.PhasePending),
		}},
	})
}

// TestStatusEndsWithTheCard (marker-presence over statusReadyState) and
// TestStatusEnvelopeCarriesTheCardsFields (command/prefix presence over the
// same fixture) are now a full subset of
// TestEveryLifecycleCommandEndsWithNextAction's "checking status" subtest
// (cmd/lifecycle_next_action_coverage_test.go), which drives the identical
// statusReadyState fixture and checks the same marker, command presence,
// "aether " prefix and resolution against the live command tree -- deleted
// here rather than kept as a duplicate (Phase 197 plan 07).

// TestStatusOverridesReachBothTheCardAndTheEnvelope proves the in-flight-
// workers case and the guided-actions case each reach BOTH the screen and the
// machine-readable answer from one decision. Failing when either override is
// applied to only one of them is the point of this test.
func TestStatusOverridesReachBothTheCardAndTheEnvelope(t *testing.T) {
	t.Run("workers in flight", func(t *testing.T) {
		newNextActionFixtureStore(t)
		state := statusReadyState(t)
		started := time.Now().UTC()
		state.State = colony.StateEXECUTING
		state.BuildStartedAt = &started
		if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
			t.Fatalf("write the fixture project: %v", err)
		}
		spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
		if err := spawnTree.RecordSpawn("Queen", "builder", "Forge-1", "Lay the foundations", 1); err != nil {
			t.Fatalf("record spawn: %v", err)
		}

		visual, _ := runStatusCommand(t, false)
		if !strings.Contains(visual, "still running") {
			t.Errorf("the in-flight-workers advice is missing from the screen:\n%s", visual)
		}

		_, envelope := runStatusCommand(t, true)
		recommendation, _ := envelope[nextActionRecommendationKey].(string)
		if !strings.Contains(recommendation, "still running") {
			t.Errorf("the machine-readable answer's recommendation does not carry the in-flight-workers fact: %q", recommendation)
		}
	})

	t.Run("guided action", func(t *testing.T) {
		newNextActionFixtureStore(t)
		state := statusReadyState(t)
		if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
			t.Fatalf("write the fixture project: %v", err)
		}
		phase := 1
		if err := store.SaveJSON("pending-decisions.json", colony.FlagsFile{
			Version: "1.0",
			Decisions: []colony.FlagEntry{{
				ID:          "flag_1",
				Type:        "blocker",
				Description: "The payment provider has not been chosen yet.",
				Phase:       &phase,
				CreatedAt:   "2026-08-28T09:00:00Z",
			}},
		}); err != nil {
			t.Fatalf("write the blocker fixture: %v", err)
		}

		visual, _ := runStatusCommand(t, false)
		if !strings.Contains(visual, "Flags needs attention") {
			t.Errorf("the guided-action advice is missing from the screen:\n%s", visual)
		}
		if !strings.Contains(visual, "aether flags --status active") {
			t.Errorf("the guided-action's own command is missing from the screen:\n%s", visual)
		}

		_, envelope := runStatusCommand(t, true)
		command, _ := envelope[nextActionCommandKey].(string)
		if command != "aether flags --status active" {
			t.Errorf("the machine-readable answer does not carry the guided action's command; got %q", command)
		}
		recommendation, _ := envelope[nextActionRecommendationKey].(string)
		if !strings.Contains(recommendation, "Flags needs attention") {
			t.Errorf("the machine-readable answer's recommendation does not carry the guided-action fact: %q", recommendation)
		}
	})
}

// ---------------------------------------------------------------------------
// All three, plain English
// ---------------------------------------------------------------------------

func TestEndgameLifecycleCardsComeFromTheResolver(t *testing.T) {
	t.Run("sealing", func(t *testing.T) {
		run := sealCardRun(t)
		if !strings.Contains(run.visual, run.card) {
			t.Errorf("sealing does not end with the shared card:\n%s", run.visual)
		}
	})
	t.Run("recovering", func(t *testing.T) {
		run := recoverCardRun(t, recoverEndgameState(t))
		if !strings.Contains(run.visual, "legacy-recover-migration/v1") || !strings.Contains(run.visual, "aether resume") {
			t.Errorf("recovering does not render the explicit migration response:\n%s", run.visual)
		}
	})
	t.Run("status", func(t *testing.T) {
		newNextActionFixtureStore(t)
		if err := store.SaveJSON("COLONY_STATE.json", statusReadyState(t)); err != nil {
			t.Fatalf("write the fixture project: %v", err)
		}
		visual, _ := runStatusCommand(t, false)
		if marker := spacedTitle("What Next"); !strings.Contains(visual, marker) {
			t.Errorf("status does not end with the shared card:\n%s", visual)
		}
	})
}

func TestEndgameLifecycleEnvelopesMatchTheirCards(t *testing.T) {
	t.Run("sealing", func(t *testing.T) {
		assertEnvelopeMatchesCard(t, "sealing", sealCardRun(t))
	})
}
