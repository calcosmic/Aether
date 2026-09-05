package cmd

// Phase 197 plan 01, criterion 6: the resolver can never hand the owner a
// command this project does not have.
//
// Three proofs live here, and the third is the one the other two cannot
// substitute for.
//
// At runtime the gate DROPS a candidate that does not resolve and falls back to
// one that does. That is right for a live session -- an owner mid-build should
// get the best available advice, not a crash. But it is exactly why a sweep
// over the commands the resolver actually emitted cannot catch a rename: the
// sweep would silently measure the FALLBACK and stay green, and the gate would
// be present having never once been seen to refuse anything.
//
// TestEveryResolverCandidateResolves therefore walks the enumerable candidate
// set itself and resolves every entry BEFORE any gating happens. Renaming or
// removing a command in the tree fails the build directly on that set.

import (
	"fmt"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// blockedBuildFixtureState is a project whose phase 2 build finished but did
// not pass its check -- the state a saved recovery report belongs to.
func blockedBuildFixtureState(t *testing.T) colony.ColonyState {
	t.Helper()
	return colony.ColonyState{
		Version:        "1.0",
		Goal:           fixtureGoal("Ship the billing rewrite"),
		State:          colony.StateBUILT,
		CurrentPhase:   2,
		BuildStartedAt: fixtureTime(t, "2026-08-01T10:00:00Z"),
		Milestone:      "Open Chambers",
		Plan: colony.Plan{Phases: []colony.Phase{
			fixturePhase(1, "Foundations", colony.PhaseCompleted),
			fixturePhase(2, "Billing engine", colony.PhaseInProgress),
		}},
	}
}

// readyPhaseFixtureState is a project sitting on a numbered phase that has not
// been started, so the recommendation must carry that number.
func readyPhaseFixtureState(t *testing.T, phaseID int) colony.ColonyState {
	t.Helper()
	phases := make([]colony.Phase, 0, phaseID)
	for id := 1; id < phaseID; id++ {
		phases = append(phases, fixturePhase(id, fmt.Sprintf("Phase %d", id), colony.PhaseCompleted))
	}
	phases = append(phases, fixturePhase(phaseID, "Billing engine", colony.PhaseReady))
	return colony.ColonyState{
		Version:      "1.0",
		Goal:         fixtureGoal("Ship the billing rewrite"),
		State:        colony.StateREADY,
		CurrentPhase: phaseID,
		Milestone:    "Open Chambers",
		Plan:         colony.Plan{Phases: phases},
	}
}

// TestEveryResolverCandidateResolves is criterion 6's load-bearing proof.
//
// Every command the resolver may say is declared once, in nextActionCandidates.
// This walks that set and resolves each entry against the LIVE cobra tree with
// no gating in between, so a rename turns the build red instead of being
// absorbed by the fallback.
//
// The set is NOT a substitute for the tree. It records what the resolver may
// SAY; rootCmd remains the sole authority on what EXISTS. That is why this test
// resolves the set against the tree rather than comparing it to a second list.
func TestEveryResolverCandidateResolves(t *testing.T) {
	if len(nextActionCandidates) == 0 {
		t.Fatal("the candidate set is empty; there is nothing for this test to prove")
	}
	seen := map[nextActionCandidateKey]bool{}
	for _, candidate := range nextActionCandidates {
		if seen[candidate.Key] {
			t.Errorf("candidate key %q is declared twice", candidate.Key)
		}
		seen[candidate.Key] = true

		if _, ok := availableCommand(candidate.Template); !ok {
			t.Errorf("candidate %q spells %q, which does not resolve to a registered command.\n"+
				"Either the command was renamed or removed, or this entry was never real. "+
				"Fix the entry -- do not delete this test.",
				candidate.Key, candidate.Template)
		}
		if strings.TrimSpace(candidate.Why) == "" {
			t.Errorf("candidate %q carries no plain-English reason", candidate.Key)
		}
	}
}

// TestUnregisteredCommandIsRefused demonstrates the gate refusing. A gate that
// has never been seen to refuse anything is a rule about nothing.
func TestUnregisteredCommandIsRefused(t *testing.T) {
	refused := []struct {
		name      string
		candidate string
		why       string
	}{
		{"a verb that is not in the tree", "aether definitely-not-a-real-command", "no such command"},
		{"a verb that only prefixes a real one", "aether contin", "prefix matching must never pass for a real command"},
		{"a plausible near-miss", "aether builds 3", "a near-miss name is still not the command"},
		{"a bare binary name", "aether", "no verb at all"},
		{"a slash form", "/ant-continue", "the resolver's values are runtime form only"},
		{"empty", "", "nothing to resolve"},
		{"a command belonging to another tool", "git status", "not this program's tree"},
	}
	for _, tc := range refused {
		t.Run(tc.name, func(t *testing.T) {
			if got, ok := availableCommand(tc.candidate); ok {
				t.Fatalf("gate accepted %q (returned %q) -- %s", tc.candidate, got, tc.why)
			}
		})
	}

	// And the positive control: the gate must still accept real commands, or
	// the refusals above prove nothing.
	for _, accepted := range []string{"aether continue", "aether status", "aether build 3"} {
		if _, ok := availableCommand(accepted); !ok {
			t.Errorf("gate refused %q, which is a real command; the refusals above would then be meaningless", accepted)
		}
	}
}

// TestRecommendedCommandIsAlwaysAvailable sweeps every state in the lifecycle
// table and checks the recommendation AND every alternative.
func TestRecommendedCommandIsAlwaysAvailable(t *testing.T) {
	for _, tc := range nextActionLifecycleCases() {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveNextAction(tc.input(t))
			commands := []string{got.Command}
			for _, choice := range got.Projection.NextAction.Choices {
				commands = append(commands, choice.RuntimeCommand)
			}
			if strings.TrimSpace(got.Command) == "" && (got.Projection == nil || len(got.Projection.NextAction.Choices) == 0) {
				t.Fatal("the resolver produced neither a command nor an executable choice")
			}
			for _, command := range commands {
				if strings.TrimSpace(command) == "" {
					continue
				}
				if _, ok := availableCommand(command); !ok {
					t.Errorf("recommended command %q is not registered", command)
				}
			}
			for _, alt := range got.Alternatives {
				if _, ok := availableCommand(alt.Command); !ok {
					t.Errorf("alternative %q is not a registered command", alt.Command)
				}
			}
		})
	}
}

// TestRecoveryCommandFromDiskIsGated covers the case the candidate set cannot:
// a command read out of a saved recovery report. It is data, not a literal the
// resolver spells, so it is gated like anything else and a dead one falls back
// to a command that is itself gated -- with the substitution recorded, so the
// owner is never silently redirected.
func TestRecoveryCommandFromDiskIsGated(t *testing.T) {
	newNextActionFixtureStore(t)
	state := normalizedFixtureState(t, blockedBuildFixtureState(t))

	t.Run("a live recovery command is used verbatim", func(t *testing.T) {
		guidance := writeRecoveryReportFixture(t, state, "aether build 2 --force")
		got := resolveNextAction(nextActionInput{State: state, Recovery: guidance})
		if got.Command != "aether build 2 --force" {
			t.Fatalf("command = %q, want the report's own command", got.Command)
		}
		if len(got.Notes) != 0 {
			t.Errorf("a live command was substituted anyway: %v", got.Notes)
		}
	})

	t.Run("a dead recovery command falls back and says so", func(t *testing.T) {
		guidance := writeRecoveryReportFixture(t, state, "aether reconcile-the-thing --phase 2")
		got := resolveNextAction(nextActionInput{State: state, Recovery: guidance})
		if got.Command == "aether reconcile-the-thing --phase 2" {
			t.Fatal("a command that is not in this program was handed to the owner")
		}
		if _, ok := availableCommand(got.Command); !ok {
			t.Fatalf("the fallback %q is not a registered command either", got.Command)
		}
		if len(got.Notes) == 0 {
			t.Error("the owner was silently redirected: no note records the substitution")
		}
	})
}

// TestCommandArgumentsSurviveTheGate: the gate resolves on the verb only, so a
// phase number or a quoted goal comes through untouched.
func TestCommandArgumentsSurviveTheGate(t *testing.T) {
	cases := map[string]string{
		"aether build 7":         "aether build 7",
		"aether build 7 --force": "aether build 7 --force",
		`aether init "goal"`:     `aether init "goal"`,
	}
	for candidate, want := range cases {
		got, ok := availableCommand(candidate)
		if !ok {
			t.Fatalf("gate refused %q, which names a real command", candidate)
		}
		if got != want {
			t.Errorf("gate returned %q, want the argument preserved as %q", got, want)
		}
	}

	// And through the resolver: a phase number reaches the recommendation.
	state := normalizedFixtureState(t, readyPhaseFixtureState(t, 7))
	got := resolveNextAction(nextActionInput{State: state})
	if got.Projection == nil || len(got.Projection.NextAction.Choices) != 2 || got.Projection.NextAction.Choices[0].RuntimeCommand != "aether build 7" || got.Projection.NextAction.Choices[1].RuntimeCommand != "aether run" {
		t.Errorf("ready phase choices = %#v, want canonical build 7 and run choices", got.Projection)
	}
}

// TestNoCommandIsSpelledInlineAtABranch: every command the resolver emits over
// the whole state table is drawn from the candidate set, or is data from a
// saved report. Nothing is invented at a branch.
func TestNoCommandIsSpelledInlineAtABranch(t *testing.T) {
	// A templated candidate may be formatted with a phase number, so both sides
	// are reduced to the same shape: the command with any numeric argument (or
	// its %d placeholder) removed.
	normalise := func(command string) string {
		var out []string
		for _, field := range strings.Fields(command) {
			if field == "%d" {
				continue
			}
			if _, err := fmt.Sscanf(field, "%d", new(int)); err == nil {
				continue
			}
			out = append(out, field)
		}
		return strings.Join(out, " ")
	}

	fromSet := map[string]bool{}
	for _, candidate := range nextActionCandidates {
		fromSet[candidate.Template] = true
		fromSet[normalise(candidate.Template)] = true
	}

	for _, tc := range nextActionLifecycleCases() {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveNextAction(tc.input(t))
			commands := []string{got.Command}
			for _, alt := range got.Alternatives {
				commands = append(commands, alt.Command)
			}
			for _, command := range commands {
				shape := normalise(command)
				if fromSet[command] || fromSet[shape] {
					continue
				}
				t.Errorf("command %q (shape %q) is not drawn from nextActionCandidates; "+
					"a command spelled inline at a branch cannot be checked by "+
					"TestEveryResolverCandidateResolves", command, shape)
			}
		})
	}
}
