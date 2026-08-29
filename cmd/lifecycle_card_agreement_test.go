package cmd

// Phase 197 plan 04, task 3 -- seven commands, one answer.
//
// Plans 197-01 and 197-02 made one piece of logic decide what happens next and
// gave it one card to render. That is a property of the LIBRARY. This file
// tests the property criterion 2 actually asks for, which is a property of the
// SURFACE: that the seven commands the owner spends a project inside all print
// the same answer, on every platform, and never name a command the program does
// not have.
//
// It is written to be able to fail. Reverting any single one of the seven to
// its old hand-written block makes it fail by name, and that has been
// demonstrated rather than assumed (see the plan's summary).

import (
	"regexp"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// closingCommandRe pulls the command out of a rendered closing block. It
// deliberately accepts BOTH spellings -- the slash form the wrapper platforms
// show and the runtime form the command line shows -- because which one appears
// is exactly what the platform test below is measuring.
var closingCommandRe = regexp.MustCompile("Run `((?:/ant-|aether )[^`]+)`")

// commandInClosing returns the command the closing block recommends.
func commandInClosing(t *testing.T, label, rendered string) string {
	t.Helper()
	marker := strings.Index(rendered, spacedTitle("Next Up"))
	if marker < 0 {
		t.Fatalf("%s printed no closing block at all:\n%s", label, rendered)
	}
	match := closingCommandRe.FindStringSubmatch(rendered[marker:])
	if len(match) < 2 {
		t.Fatalf("%s printed a closing block that recommends no command:\n%s", label, rendered[marker:])
	}
	return strings.TrimSpace(match[1])
}

// migratedSurfaceRenderings renders all thirteen migrated closings over ONE
// saved project. Every renderer keeps the signature its existing callers use;
// the result maps are deliberately bare, because what is being measured is
// where the advice comes from, not what each command reports about its own
// run.
//
// Resuming is not in this map. Its real content depends on session.json,
// learning-observations.json and several other files this shared fixture does
// not seed, and it already has full, dedicated coverage (screen and envelope,
// both forms) in lifecycle_card_session_test.go over its own fixture.
func migratedSurfaceRenderings(t *testing.T, state colony.ColonyState) map[string]string {
	t.Helper()
	phase := state.Plan.Phases[0]
	nextPhase := state.Plan.Phases[1]
	return map[string]string{
		"starting a project":  renderInitVisual("Ship the billing rewrite", string(colony.ScopeProject), "session-1", ".aether/data", nil, 0, nil),
		"talking it through":  renderDiscussVisual(map[string]interface{}{"goal": "Ship the billing rewrite"}),
		"scanning the code":   renderColonizeVisual(map[string]interface{}{"root": "."}),
		"drawing up the plan": renderPlanVisual(map[string]interface{}{"goal": "Ship the billing rewrite"}),
		"building a phase":    renderBuildVisualWithDispatches(state, phase, nil, colony.VerificationDepthStandard),
		"checking the work":   renderContinueVisual(state, phase, nil, false, &nextPhase, nil, colony.VerificationDepthStandard),
		"the shared closeout": renderCloseoutVisual(map[string]interface{}{"workflow": "build", "state_available": true}),
		"the wrapper's closeout": renderCeremonyCloseoutVisual(map[string]interface{}{
			"workflow": "build", "state_available": true, "current_phase": state.CurrentPhase,
		}),
		"pausing the project": renderPauseVisual(map[string]interface{}{
			"goal": "Ship the billing rewrite", "current_phase": state.CurrentPhase,
		}),
		"updating the project": renderUpdateVisual("/tmp/example", "1.0.0", "1.0.0", "", false, false,
			nil, 0, 0, nil, "unchanged", true, map[string]interface{}{}),
		"sealing the project":  renderSealVisual(sealCardFixtureResult(state), state, "/tmp/CROWNED-ANTHILL.md"),
		"recovering a project": renderRecoverDiagnosis(nil, state, nil),
		"checking the status":  renderDashboard(state, store, buildStatusResult(state, store)),
	}
}

// sealCardFixtureResult folds the answer the way runSeal itself does before
// calling renderSealVisual -- the plain phrase, not the literal jargon word
// "seal", so the "what changed" sentence never needs it explained twice.
func sealCardFixtureResult(state colony.ColonyState) map[string]interface{} {
	result := map[string]interface{}{}
	closeLifecycleRun(result, state, "signing the project off as finished")
	return result
}

// oneAgreementState is a project with a plan and nothing ambiguous about it:
// phase 1 is ready, so every one of the seven has exactly one honest answer.
func oneAgreementState(t *testing.T) colony.ColonyState {
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

// TestMigratedLifecycleSurfacesAgree is the test that would have caught a
// missed variant: one saved project, seven closings, one command out of all of
// them.
func TestMigratedLifecycleSurfacesAgree(t *testing.T) {
	newNextActionFixtureStore(t)
	t.Setenv("AETHER_PLATFORM", "codex")
	state := oneAgreementState(t)
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("write the fixture project through the runtime's own store: %v", err)
	}

	want := resolveNextAction(loadNextActionInput()).Command
	if want == "" {
		t.Fatal("the one resolver named no command for this project at all")
	}

	for label, rendered := range migratedSurfaceRenderings(t, state) {
		got := commandInClosing(t, label, rendered)
		if got != want {
			t.Errorf("%s tells the owner to run %q; the one answer for this project is %q -- the surfaces have separated",
				label, got, want)
		}
	}
}

// TestMigratedLifecycleSurfacesArePlatformCorrect is S-01 stated as a test:
// the spelling on screen follows the platform, and the value a wrapper executes
// never does.
func TestMigratedLifecycleSurfacesArePlatformCorrect(t *testing.T) {
	cases := []struct {
		platform string
		prefix   string
	}{
		{"claude", "/ant-"},
		{"opencode", "/ant-"},
		{"codex", "aether "},
	}

	for _, tc := range cases {
		t.Run(tc.platform, func(t *testing.T) {
			newNextActionFixtureStore(t)
			t.Setenv("AETHER_PLATFORM", tc.platform)
			state := oneAgreementState(t)
			if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
				t.Fatalf("write the fixture project: %v", err)
			}

			answer := resolveNextAction(loadNextActionInput())
			// The value a wrapper and the automation EXECUTE is the runtime
			// form on every platform. Handing `/ant-build 1` to a shell is a
			// broken command, which is why the answer never carries one.
			if !strings.HasPrefix(answer.Command, "aether ") {
				t.Errorf("on %s the machine-readable answer is %q, which is not something that can be run",
					tc.platform, answer.Command)
			}
			for _, alternative := range answer.Alternatives {
				if !strings.HasPrefix(alternative.Command, "aether ") {
					t.Errorf("on %s an alternative is %q, which is not something that can be run",
						tc.platform, alternative.Command)
				}
			}

			for label, rendered := range migratedSurfaceRenderings(t, state) {
				got := commandInClosing(t, label, rendered)
				if !strings.HasPrefix(got, tc.prefix) {
					t.Errorf("on %s, %s shows %q; this platform's owner types commands beginning %q",
						tc.platform, label, got, tc.prefix)
				}
			}
		})
	}
}

// TestTheSevenClosingsSpeakPlainEnglish holds the block this plan owns -- the
// closing block of each of the seven -- to S-05: no word this repository
// invented appears in it without being explained in the same sentence.
//
// It is scoped to the closing block deliberately. The banners and section
// headings above it ("Colony Init", "Colony Complete") are the project's own
// house style, locked by their own tests and by recorded transcripts that
// another plan owns in this round; what the owner is TOLD TO DO is this plan's
// wording, and that is what is checked here.
func TestTheSevenClosingsSpeakPlainEnglish(t *testing.T) {
	newNextActionFixtureStore(t)
	t.Setenv("AETHER_PLATFORM", "codex")
	state := oneAgreementState(t)
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("write the fixture project: %v", err)
	}

	for label, rendered := range migratedSurfaceRenderings(t, state) {
		t.Run(label, func(t *testing.T) {
			marker := strings.Index(rendered, spacedTitle("What Next"))
			if marker < 0 {
				t.Fatalf("%s printed no closing card:\n%s", label, rendered)
			}
			closing := rendered[marker:]
			if violations := untranslatedRepoWords(closing); len(violations) > 0 {
				t.Errorf("%s ends with words this repository invented and never explains:\n  %s\n\nclosing block:\n%s",
					label, strings.Join(violations, "\n  "), closing)
			}
		})
	}
}

// TestEveryCommandTheSevenCanRecommendResolves is criterion 6 applied to the
// surface rather than to the resolver alone: every command any of the seven can
// put in front of the owner must be a command this build of the program
// actually has.
//
// It reuses plan 197-01's availability gate rather than reimplementing it, and
// it walks several different projects, because which branch of the decision is
// reached is what decides which commands can appear.
func TestEveryCommandTheSevenCanRecommendResolves(t *testing.T) {
	for _, fixture := range oneDeciderFixtures(t) {
		t.Run(fixture.name, func(t *testing.T) {
			newNextActionFixtureStore(t)
			t.Setenv("AETHER_PLATFORM", "codex")
			if err := store.SaveJSON("COLONY_STATE.json", fixture.state); err != nil {
				t.Fatalf("write the fixture project: %v", err)
			}
			answer := resolveNextAction(loadNextActionInput())

			commands := []string{answer.Command}
			for _, alternative := range answer.Alternatives {
				commands = append(commands, alternative.Command)
			}
			for _, command := range commands {
				if _, ok := availableCommand(command); !ok {
					t.Errorf("the closing card offers %q, which this build of the program does not have", command)
				}
			}
		})
	}
}

// TestEveryCommandTheMigratedSurfacesRenderResolves is criterion 6 walked over
// the actual rendered output of every migrated surface -- not the resolver's
// fixtures alone -- so a surface that somehow rendered a command the resolver
// never produced (a hand-typed leftover, a stale literal) would still be
// caught here.
func TestEveryCommandTheMigratedSurfacesRenderResolves(t *testing.T) {
	newNextActionFixtureStore(t)
	t.Setenv("AETHER_PLATFORM", "codex")
	state := oneAgreementState(t)
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("write the fixture project: %v", err)
	}

	for label, rendered := range migratedSurfaceRenderings(t, state) {
		t.Run(label, func(t *testing.T) {
			for _, command := range allCommandsInCard(rendered) {
				if _, ok := availableCommand(command); !ok {
					t.Errorf("%s offers %q, which this build of the program does not have", label, command)
				}
			}
		})
	}
}
