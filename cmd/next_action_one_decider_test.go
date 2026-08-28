package cmd

// Phase 197 plan 02 -- the agreement invariant.
//
// Before this file, four separate functions each decided the next step from the
// same saved project state:
//
//	workflowSuggestionsForState (cmd/codex_visuals.go)
//	nextCommandFromState        (cmd/recovery_snapshot.go)
//	nextUpSuggestionsForState   (cmd/build_flow_cmds.go)
//	closeoutNextCommand         (cmd/closeout_cmd.go)
//
// They disagreed, and every closing message inherited whichever one it happened
// to call. This test drives all four over one saved state and requires the
// command each names to be identical to the one answer resolveNextAction gives.
// A future edit that puts a private branch back into any of them fails here.
//
// The fixture state is written to disk through the runtime's own store and read
// back through the runtime's own loader, so the input the resolver is compared
// against is the input production would build -- never a hand-typed literal.

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// suggestionCommandRe finds the runtime command inside a rendered suggestion
// sentence. The deciders return prose with the command in backticks; this pulls
// it back out so four different sentence shapes can be compared as one value.
var suggestionCommandRe = regexp.MustCompile("`(aether [^`]+)`")

// commandFromSuggestion extracts the runtime command a suggestion names.
func commandFromSuggestion(t *testing.T, label, text string) string {
	t.Helper()
	matches := suggestionCommandRe.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		t.Fatalf("%s named no runtime command at all:\n%s", label, text)
	}
	return strings.TrimSpace(matches[0][1])
}

// deciderFixture is one saved state and the answer the project should give for
// it, whichever door the question is asked through.
type deciderFixture struct {
	name        string
	state       colony.ColonyState
	wantCommand string
}

func oneDeciderFixtures(t *testing.T) []deciderFixture {
	t.Helper()
	now := time.Now().UTC()
	running := now.Add(-time.Minute)
	stalled := now.Add(-24 * time.Hour)

	return []deciderFixture{
		{
			name: "a build that is genuinely running is checked, not restarted",
			state: normalizedFixtureState(t, colony.ColonyState{
				Version:        "1.0",
				Goal:           fixtureGoal("Ship the billing rewrite"),
				State:          colony.StateEXECUTING,
				CurrentPhase:   2,
				BuildStartedAt: &running,
				Milestone:      "Open Chambers",
				Plan: colony.Plan{Phases: []colony.Phase{
					fixturePhase(1, "Foundations", colony.PhaseCompleted),
					fixturePhase(2, "Billing engine", colony.PhaseInProgress),
				}},
			}),
			wantCommand: "aether continue",
		},
		{
			// Named branch 1 of the 2 the plan requires to survive the merge:
			// a phase interrupted before it did any work.
			name: "the abandoned-build redispatch branch: an interrupted build that never started is restarted",
			state: normalizedFixtureState(t, colony.ColonyState{
				Version:      "1.0",
				Goal:         fixtureGoal("Ship the billing rewrite"),
				State:        colony.StateEXECUTING,
				CurrentPhase: 2,
				Milestone:    "Open Chambers",
				Plan: colony.Plan{Phases: []colony.Phase{
					fixturePhase(1, "Foundations", colony.PhaseCompleted),
					fixturePhase(2, "Billing engine", colony.PhaseInProgress),
				}},
			}),
			wantCommand: "aether build 2 --force",
		},
		{
			// Named branch 2 of 2: a build that started long ago with no
			// dispatch record behind it. Only nextCommandFromState had this.
			name: "the absent-manifest branch: a long-stalled build with no dispatch record is restarted",
			state: normalizedFixtureState(t, colony.ColonyState{
				Version:        "1.0",
				Goal:           fixtureGoal("Ship the billing rewrite"),
				State:          colony.StateEXECUTING,
				CurrentPhase:   2,
				BuildStartedAt: &stalled,
				Milestone:      "Open Chambers",
				Plan: colony.Plan{Phases: []colony.Phase{
					fixturePhase(1, "Foundations", colony.PhaseCompleted),
					fixturePhase(2, "Billing engine", colony.PhaseInProgress),
				}},
			}),
			wantCommand: "aether build 2 --force",
		},
		{
			name: "a paused project is picked back up",
			state: normalizedFixtureState(t, colony.ColonyState{
				Version:      "1.0",
				Goal:         fixtureGoal("Ship the billing rewrite"),
				State:        colony.State("PAUSED"),
				CurrentPhase: 2,
				Milestone:    "Open Chambers",
				Plan: colony.Plan{Phases: []colony.Phase{
					fixturePhase(1, "Foundations", colony.PhaseCompleted),
					fixturePhase(2, "Billing engine", colony.PhaseReady),
				}},
			}),
			wantCommand: "aether resume",
		},
		{
			name: "a finished but unsigned-off project is told to sign off, never to archive",
			state: normalizedFixtureState(t, colony.ColonyState{
				Version:   "1.0",
				Goal:      fixtureGoal("Ship the billing rewrite"),
				State:     colony.StateCOMPLETED,
				Milestone: "Open Chambers",
				Plan: colony.Plan{Phases: []colony.Phase{
					fixturePhase(1, "Foundations", colony.PhaseCompleted),
				}},
			}),
			wantCommand: "aether seal",
		},
		{
			name: "a signed-off project is filed away",
			state: normalizedFixtureState(t, colony.ColonyState{
				Version:   "1.0",
				Goal:      fixtureGoal("Ship the billing rewrite"),
				State:     colony.StateCOMPLETED,
				Milestone: "Crowned Anthill",
				Plan: colony.Plan{Phases: []colony.Phase{
					fixturePhase(1, "Foundations", colony.PhaseCompleted),
				}},
			}),
			wantCommand: "aether entomb",
		},
		{
			name: "a ready project starts the next unfinished phase",
			state: normalizedFixtureState(t, colony.ColonyState{
				Version:      "1.0",
				Goal:         fixtureGoal("Ship the billing rewrite"),
				State:        colony.StateREADY,
				CurrentPhase: 2,
				Milestone:    "Open Chambers",
				Plan: colony.Plan{Phases: []colony.Phase{
					fixturePhase(1, "Foundations", colony.PhaseCompleted),
					fixturePhase(2, "Billing engine", colony.PhaseReady),
					fixturePhase(3, "Reporting", colony.PhasePending),
				}},
			}),
			wantCommand: "aether build 2",
		},
		{
			name: "a project with a goal but no phases is talked through first",
			state: normalizedFixtureState(t, colony.ColonyState{
				Version:   "1.0",
				Goal:      fixtureGoal("Ship the billing rewrite"),
				State:     colony.StateREADY,
				Milestone: "First Mound",
			}),
			wantCommand: "aether discuss",
		},
	}
}

// closeoutWorkflows are the workflow names closeout is invoked with in
// production. Which command just ran is real information and may shape what the
// owner is TOLD, but it may not produce a different answer to "what next" for
// the same saved state.
var closeoutWorkflows = []string{"", "status", "build", "plan", "colonize", "continue", "seal", "swarm"}

func TestEveryDeciderAgreesOnTheNextCommand(t *testing.T) {
	for _, fixture := range oneDeciderFixtures(t) {
		t.Run(fixture.name, func(t *testing.T) {
			newNextActionFixtureStore(t)
			if err := store.SaveJSON("COLONY_STATE.json", fixture.state); err != nil {
				t.Fatalf("write fixture state through the runtime's own store: %v", err)
			}

			answer := resolveNextAction(loadNextActionInput())
			if answer.Command != fixture.wantCommand {
				t.Fatalf("the one resolver answered %q, want %q", answer.Command, fixture.wantCommand)
			}

			primary, _ := workflowSuggestionsForState(fixture.state)
			got := map[string]string{
				"workflowSuggestionsForState": commandFromSuggestion(t, "workflowSuggestionsForState", primary),
				"nextCommandFromState":        strings.TrimSpace(nextCommandFromState(fixture.state)),
			}

			suggestions := nextUpSuggestionsForState(fixture.state)
			if len(suggestions) == 0 {
				t.Fatalf("nextUpSuggestionsForState returned nothing at all")
			}
			got["nextUpSuggestionsForState"] = commandFromSuggestion(t, "nextUpSuggestionsForState", suggestions[0])

			for _, workflow := range closeoutWorkflows {
				label := fmt.Sprintf("closeoutNextCommand(%q)", workflow)
				got[label] = commandFromSuggestion(t, label, closeoutNextCommand(workflow, fixture.state))
			}

			for name, command := range got {
				if command != answer.Command {
					t.Errorf("%s answered %q; the one resolver answered %q -- the deciders have separated again",
						name, command, answer.Command)
				}
			}
		})
	}
}

// TestNoSurvivingDeciderSpellsItsOwnCommand is the structural half of the
// invariant. Agreement today is worth little if a decider still owns branch
// logic that can drift tomorrow; after this plan each of the four is an
// adapter -- it gathers input, calls the resolver, and reshapes the answer into
// the return type its callers already expect.
//
// A command name written inside one of these four functions is exactly what
// re-opens the drift, so the name itself is what this test forbids.
func TestNoSurvivingDeciderSpellsItsOwnCommand(t *testing.T) {
	deciders := map[string]string{
		"workflowSuggestionsForState": "codex_visuals.go",
		"nextCommandFromState":        "recovery_snapshot.go",
		"nextUpSuggestionsForState":   "build_flow_cmds.go",
		"closeoutNextCommand":         "closeout_cmd.go",
	}

	byFile := map[string][]string{}
	for fn, file := range deciders {
		byFile[file] = append(byFile[file], fn)
	}

	fset := token.NewFileSet()
	for file, functions := range byFile {
		parsed, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		wanted := map[string]bool{}
		for _, fn := range functions {
			wanted[fn] = true
		}
		found := map[string]bool{}

		for _, decl := range parsed.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil || !wanted[fn.Name.Name] {
				continue
			}
			found[fn.Name.Name] = true
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				lit, ok := node.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return true
				}
				value, err := strconv.Unquote(lit.Value)
				if err != nil {
					return true
				}
				if strings.Contains(value, "aether ") {
					t.Errorf("%s (%s) still spells a command of its own: %q\n"+
						"Every command must come from the resolver's candidate set, or the four answers drift apart again.",
						fn.Name.Name, file, value)
				}
				return true
			})
		}

		for fn := range wanted {
			if !found[fn] {
				t.Errorf("%s was not found in %s -- this test cannot protect a function it cannot see", fn, file)
			}
		}
	}
}
