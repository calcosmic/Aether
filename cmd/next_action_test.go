package cmd

// Phase 197 plan 01 -- the one answer to "what next".
//
// These tests drive the PURE half of the resolver directly. Every fixture is
// round-tripped through JSON and then through the runtime's own
// normalizeLegacyColonyState before the resolver sees it, so no test can pass
// against a state shape the program would never write. That hand-typed-fixture
// failure mode is what produced the serious faults in Phases 195 and 196.

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// normalizedFixtureState marshals a constructed state, reads it back, and runs
// it through the runtime's own normalizer. A state that cannot survive that
// round trip is not a state the runtime can produce, and a test over it proves
// nothing.
func normalizedFixtureState(t *testing.T, state colony.ColonyState) colony.ColonyState {
	t.Helper()
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("fixture state does not marshal, so the runtime could never write it: %v", err)
	}
	var roundTripped colony.ColonyState
	if err := json.Unmarshal(data, &roundTripped); err != nil {
		t.Fatalf("fixture state does not round-trip, so the runtime could never read it back: %v", err)
	}
	return normalizeLegacyColonyState(roundTripped)
}

func fixtureGoal(text string) *string {
	goal := text
	return &goal
}

// fixturePhase builds a phase the same way the runtime's planner writes one:
// the status is always one of the declared constants, never a free string.
func fixturePhase(id int, name, status string) colony.Phase {
	switch status {
	case colony.PhasePending, colony.PhaseReady, colony.PhaseInProgress, colony.PhaseCompleted, "failed":
	default:
		panic("fixturePhase: " + status + " is not a status the runtime writes")
	}
	taskID := fmt.Sprintf("phase-%d-task-1", id)
	taskStatus := colony.TaskPending
	if status == colony.PhaseCompleted {
		taskStatus = colony.TaskCompleted
	}
	return colony.Phase{
		ID:          id,
		Name:        name,
		Description: name + " description",
		Status:      status,
		Mode:        colony.PhaseModeProduction,
		Tasks: []colony.Task{{
			ID:     &taskID,
			Goal:   "Do the " + name + " work",
			Status: taskStatus,
		}},
		SuccessCriteria: []string{name + " is done"},
	}
}

func fixtureTime(t *testing.T, value string) *time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("bad fixture timestamp %q: %v", value, err)
	}
	return &parsed
}

// writeRecoveryReportFixture writes a continue report through the runtime's own
// store and then reads it back with the runtime's own reader, so the recovery
// guidance the resolver receives is exactly the value production would build.
func writeRecoveryReportFixture(t *testing.T, state colony.ColonyState, next string) *activeRecoveryGuidance {
	t.Helper()
	rel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", state.CurrentPhase), "continue.json"))
	report := codexContinueReport{
		Phase:       state.CurrentPhase,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Summary:     "Verification failed: two tasks were never credited.",
		Next:        next,
		Recovery: codexContinueRecoveryPlan{
			ReconcileCommand: "aether build-reconcile",
		},
	}
	if err := store.SaveJSON(rel, report); err != nil {
		t.Fatalf("write recovery report fixture: %v", err)
	}
	guidance := loadActiveRecoveryGuidance(state)
	if guidance == nil {
		t.Fatalf("the runtime's own reader refused the recovery report fixture")
	}
	return guidance
}

// newNextActionFixtureStore sets up a temp project with a live store so
// fixtures can be produced by the runtime's own writers.
func newNextActionFixtureStore(t *testing.T) {
	t.Helper()
	saveGlobalsCmd(t)
	s, tmpDir := newTestStoreCmd(t)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	store = s
}

type nextActionCase struct {
	name         string
	input        func(t *testing.T) nextActionInput
	wantCommand  string
	wantContains []string
}

func nextActionLifecycleCases() []nextActionCase {
	return []nextActionCase{
		{
			name: "initialised colony with no phases is told to talk it through first",
			input: func(t *testing.T) nextActionInput {
				return nextActionInput{State: normalizedFixtureState(t, colony.ColonyState{
					Version:   "1.0",
					Goal:      fixtureGoal("Ship the billing rewrite"),
					State:     colony.StateREADY,
					Milestone: "First Mound",
				})}
			},
			wantCommand:  "aether discuss",
			wantContains: []string{"aether plan", "aether colonize"},
		},
		{
			name: "a build that is running is told to verify and advance",
			input: func(t *testing.T) nextActionInput {
				return nextActionInput{State: normalizedFixtureState(t, colony.ColonyState{
					Version:        "1.0",
					Goal:           fixtureGoal("Ship the billing rewrite"),
					State:          colony.StateEXECUTING,
					CurrentPhase:   2,
					BuildStartedAt: fixtureTime(t, "2026-08-01T10:00:00Z"),
					Milestone:      "Open Chambers",
					Plan: colony.Plan{Phases: []colony.Phase{
						fixturePhase(1, "Foundations", colony.PhaseCompleted),
						fixturePhase(2, "Billing engine", colony.PhaseInProgress),
					}},
				})}
			},
			wantCommand: "aether continue",
		},
		{
			name: "a paused colony is told to pick it back up",
			input: func(t *testing.T) nextActionInput {
				pausedAt := "2026-08-02T09:30:00Z"
				return nextActionInput{State: normalizedFixtureState(t, colony.ColonyState{
					Version:      "1.0",
					Goal:         fixtureGoal("Ship the billing rewrite"),
					State:        colony.StateREADY,
					CurrentPhase: 2,
					Paused:       true,
					PausedAt:     &pausedAt,
					Milestone:    "Open Chambers",
					Plan: colony.Plan{Phases: []colony.Phase{
						fixturePhase(1, "Foundations", colony.PhaseCompleted),
						fixturePhase(2, "Billing engine", colony.PhaseReady),
					}},
				})}
			},
			wantCommand: "aether resume",
		},
		{
			name: "every phase done but the project not closed off is told to close it off",
			input: func(t *testing.T) nextActionInput {
				return nextActionInput{State: normalizedFixtureState(t, colony.ColonyState{
					Version:      "1.0",
					Goal:         fixtureGoal("Ship the billing rewrite"),
					State:        colony.StateREADY,
					CurrentPhase: 2,
					Milestone:    "Brood Stable",
					Plan: colony.Plan{Phases: []colony.Phase{
						fixturePhase(1, "Foundations", colony.PhaseCompleted),
						fixturePhase(2, "Billing engine", colony.PhaseCompleted),
					}},
				})}
			},
			wantCommand: "aether seal",
		},
		{
			name: "a finished project is told to file it away",
			input: func(t *testing.T) nextActionInput {
				return nextActionInput{State: normalizedFixtureState(t, colony.ColonyState{
					Version:      "1.0",
					Goal:         fixtureGoal("Ship the billing rewrite"),
					State:        colony.StateCOMPLETED,
					CurrentPhase: 2,
					Milestone:    "Crowned Anthill",
					Plan: colony.Plan{Phases: []colony.Phase{
						fixturePhase(1, "Foundations", colony.PhaseCompleted),
						fixturePhase(2, "Billing engine", colony.PhaseCompleted),
					}},
				})}
			},
			wantCommand: "aether entomb",
		},
		{
			name: "a ready phase is told to start that phase by number",
			input: func(t *testing.T) nextActionInput {
				return nextActionInput{State: normalizedFixtureState(t, colony.ColonyState{
					Version:      "1.0",
					Goal:         fixtureGoal("Ship the billing rewrite"),
					State:        colony.StateREADY,
					CurrentPhase: 2,
					Milestone:    "Open Chambers",
					Plan: colony.Plan{Phases: []colony.Phase{
						fixturePhase(1, "Foundations", colony.PhaseCompleted),
						fixturePhase(2, "Billing engine", colony.PhaseReady),
					}},
				})}
			},
			wantCommand: "aether build 2",
		},
		{
			name: "an interrupted build that never started is told to restart it",
			input: func(t *testing.T) nextActionInput {
				return nextActionInput{State: normalizedFixtureState(t, colony.ColonyState{
					Version:      "1.0",
					Goal:         fixtureGoal("Ship the billing rewrite"),
					State:        colony.StateEXECUTING,
					CurrentPhase: 2,
					Milestone:    "Open Chambers",
					Plan: colony.Plan{Phases: []colony.Phase{
						fixturePhase(1, "Foundations", colony.PhaseCompleted),
						fixturePhase(2, "Billing engine", colony.PhaseInProgress),
					}},
				})}
			},
			wantCommand: "aether build 2 --force",
		},
		{
			name: "a failed phase is told to run that phase again",
			input: func(t *testing.T) nextActionInput {
				return nextActionInput{State: normalizedFixtureState(t, colony.ColonyState{
					Version:      "1.0",
					Goal:         fixtureGoal("Ship the billing rewrite"),
					State:        colony.StateREADY,
					CurrentPhase: 2,
					Milestone:    "Open Chambers",
					Plan: colony.Plan{Phases: []colony.Phase{
						fixturePhase(1, "Foundations", colony.PhaseCompleted),
						fixturePhase(2, "Billing engine", "failed"),
					}},
				})}
			},
			wantCommand: "aether build 2",
		},
		{
			name: "a planning blocker is told to look at the blocker before anything else",
			input: func(t *testing.T) nextActionInput {
				return nextActionInput{
					State: normalizedFixtureState(t, colony.ColonyState{
						Version:      "1.0",
						Goal:         fixtureGoal("Ship the billing rewrite"),
						State:        colony.StateREADY,
						CurrentPhase: 1,
						Milestone:    "Open Chambers",
						Plan: colony.Plan{Phases: []colony.Phase{
							fixturePhase(1, "Foundations", colony.PhaseReady),
						}},
					}),
					PlanBlocker: &colony.FlagEntry{
						ID:          "flag-1",
						Type:        "blocker",
						Description: "plan-finalize failed: phase 2 depends on a phase that is not in the plan",
						Source:      planFinalizeFailureSource,
					},
				}
			},
			wantCommand: "aether flags --status active",
		},
		{
			name: "no colony at all is told how to start one",
			input: func(t *testing.T) nextActionInput {
				return nextActionInput{NoColony: true}
			},
			wantCommand: "aether init \"goal\"",
		},
	}
}

func TestResolveNextActionCoversEveryLifecycleState(t *testing.T) {
	for _, tc := range nextActionLifecycleCases() {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveNextAction(tc.input(t))
			if got.Command != tc.wantCommand {
				t.Fatalf("recommended command = %q, want %q", got.Command, tc.wantCommand)
			}
			for _, want := range tc.wantContains {
				found := false
				for _, alt := range got.Alternatives {
					if alt.Command == want {
						found = true
					}
				}
				if !found {
					t.Errorf("alternatives %v do not offer %q", got.Alternatives, want)
				}
			}
			if strings.TrimSpace(got.Recommendation) == "" {
				t.Error("recommendation prose is empty; the owner is told a command with no reason")
			}
		})
	}
}

func TestResolveNextActionUsesTheRecoveryReportCommand(t *testing.T) {
	newNextActionFixtureStore(t)
	state := normalizedFixtureState(t, colony.ColonyState{
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
	})
	guidance := writeRecoveryReportFixture(t, state, "aether build 2 --force")

	got := resolveNextAction(nextActionInput{State: state, Recovery: guidance})
	if got.Command != "aether build 2 --force" {
		t.Fatalf("blocked build recommended %q, want the targeted recovery command from the report", got.Command)
	}
	if !got.Recovery.Blocked {
		t.Error("recovery field does not report the build as blocked")
	}
}

func TestResolveNextActionReportsPausedState(t *testing.T) {
	pausedAt := "2026-08-02T09:30:00Z"
	state := normalizedFixtureState(t, colony.ColonyState{
		Version:      "1.0",
		Goal:         fixtureGoal("Ship the billing rewrite"),
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Paused:       true,
		PausedAt:     &pausedAt,
		Milestone:    "Open Chambers",
		Plan:         colony.Plan{Phases: []colony.Phase{fixturePhase(1, "Foundations", colony.PhaseReady)}},
	})
	got := resolveNextAction(nextActionInput{State: state})
	if !got.Recovery.Paused {
		t.Fatal("a paused colony does not report itself as paused")
	}
	if got.Recovery.PausedAt != pausedAt {
		t.Errorf("paused-at = %q, want %q", got.Recovery.PausedAt, pausedAt)
	}
}

func TestResolveNextActionIsPure(t *testing.T) {
	for _, tc := range nextActionLifecycleCases() {
		t.Run(tc.name, func(t *testing.T) {
			in := tc.input(t)
			first := resolveNextAction(in)
			second := resolveNextAction(in)
			if !reflect.DeepEqual(first, second) {
				t.Fatalf("same input produced different answers:\nfirst:  %+v\nsecond: %+v", first, second)
			}
		})
	}
}

func TestNextActionAlternativesAreWellFormed(t *testing.T) {
	for _, tc := range nextActionLifecycleCases() {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveNextAction(tc.input(t))
			if len(got.Alternatives) < 2 || len(got.Alternatives) > 4 {
				t.Fatalf("got %d alternatives, want between 2 and 4: %+v", len(got.Alternatives), got.Alternatives)
			}
			seen := map[string]bool{}
			for _, alt := range got.Alternatives {
				if alt.Command == got.Command {
					t.Errorf("alternative %q repeats the recommended command", alt.Command)
				}
				if seen[alt.Command] {
					t.Errorf("alternative %q is listed twice", alt.Command)
				}
				seen[alt.Command] = true
				if strings.TrimSpace(alt.Explanation) == "" {
					t.Errorf("alternative %q carries no explanation", alt.Command)
				}
			}
		})
	}
}

func TestNextActionContextHealthIsSafeOnlyWithAHandoffOnDisk(t *testing.T) {
	base := normalizedFixtureState(t, colony.ColonyState{
		Version:      "1.0",
		Goal:         fixtureGoal("Ship the billing rewrite"),
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Milestone:    "Open Chambers",
		Plan:         colony.Plan{Phases: []colony.Phase{fixturePhase(1, "Foundations", colony.PhaseReady)}},
	})

	without := resolveNextAction(nextActionInput{State: base, HandoffExists: false})
	if without.ContextHealth.Health != contextHealthKeep {
		t.Fatalf("with no handoff on disk, context health = %q, want %q", without.ContextHealth.Health, contextHealthKeep)
	}
	if strings.TrimSpace(without.ContextHealth.Reason) == "" {
		t.Error("context health carries no machine-readable reason code")
	}

	with := resolveNextAction(nextActionInput{State: base, HandoffExists: true})
	if with.ContextHealth.Health == contextHealthKeep {
		t.Fatalf("with a handoff on disk and no build running, context health = %q, want it not to be %q", with.ContextHealth.Health, contextHealthKeep)
	}
}

func TestNextActionMidBuildIsNeverSafeToClose(t *testing.T) {
	state := normalizedFixtureState(t, colony.ColonyState{
		Version:        "1.0",
		Goal:           fixtureGoal("Ship the billing rewrite"),
		State:          colony.StateEXECUTING,
		CurrentPhase:   1,
		BuildStartedAt: fixtureTime(t, "2026-08-01T10:00:00Z"),
		Milestone:      "Open Chambers",
		Plan:           colony.Plan{Phases: []colony.Phase{fixturePhase(1, "Foundations", colony.PhaseInProgress)}},
	})
	got := resolveNextAction(nextActionInput{State: state, HandoffExists: true})
	if got.ContextHealth.Health != contextHealthKeep {
		t.Fatalf("mid-build context health = %q, want %q -- a running build is never safe to walk away from",
			got.ContextHealth.Health, contextHealthKeep)
	}
}

// TestNextActionAnswerFieldsAreAllSerialisable guards NEXT-01's requirement
// that the whole answer is what later plans marshal into the machine-readable
// envelope: a field with no json tag is a field the wrapper cannot see.
func TestNextActionAnswerFieldsAreAllSerialisable(t *testing.T) {
	typ := reflect.TypeOf(nextAction{})
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Tag.Get("json") == "" {
			t.Errorf("nextAction.%s has no json tag, so the wrapper cannot see it", field.Name)
		}
	}
}

// ---------------------------------------------------------------------------
// The loader (task 3)
// ---------------------------------------------------------------------------

// writeFixtureColonyState persists a state through the runtime's own store, so
// the loader reads exactly what production would have written.
func writeFixtureColonyState(t *testing.T, state colony.ColonyState) {
	t.Helper()
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("write colony state fixture: %v", err)
	}
}

func TestLoadNextActionInputGathersSavedState(t *testing.T) {
	newNextActionFixtureStore(t)

	state := normalizedFixtureState(t, colony.ColonyState{
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
	})
	writeFixtureColonyState(t, state)

	flags := colony.FlagsFile{Version: "1.0", Decisions: []colony.FlagEntry{
		{ID: "flag-1", Type: "blocker", Description: "the payment provider key is missing", Source: "watcher"},
		{ID: "flag-2", Type: "note", Description: "already dealt with", Source: "watcher", Resolved: true},
	}}
	if err := store.SaveJSON("pending-decisions.json", flags); err != nil {
		t.Fatalf("write flags fixture: %v", err)
	}
	guidance := writeRecoveryReportFixture(t, state, "aether build 2 --force")

	got := loadNextActionInput()

	if got.NoColony {
		t.Fatal("the loader reported no project in a project that has one")
	}
	if got.State.CurrentPhase != 2 || got.State.State != colony.StateBUILT {
		t.Errorf("loaded state = phase %d / %s, want phase 2 / BUILT", got.State.CurrentPhase, got.State.State)
	}
	if len(got.Flags) != 1 || got.Flags[0].ID != "flag-1" {
		t.Errorf("open flags = %+v, want only the unresolved one", got.Flags)
	}
	if got.Recovery == nil || got.Recovery.Next != guidance.Next {
		t.Errorf("recovery guidance = %+v, want the report the runtime's own reader produced", got.Recovery)
	}
	if len(got.ActiveTodos) == 0 {
		t.Error("outstanding task goals were not gathered")
	}

	// The handoff document is the one fact "safe to close the chat" rests on,
	// so the loader must report its real presence, both ways.
	if got.HandoffExists {
		t.Error("the loader claims a handoff exists when none is on disk")
	}
	if err := writeHandoffDocument("# handoff\n"); err != nil {
		t.Fatalf("write handoff fixture: %v", err)
	}
	if again := loadNextActionInput(); !again.HandoffExists {
		t.Error("the loader does not see a handoff that is on disk")
	}
}

func TestLoadNextActionInputDistinguishesNoColony(t *testing.T) {
	newNextActionFixtureStore(t)

	got := loadNextActionInput()
	if !got.NoColony {
		t.Fatal("an empty project was not reported as having no project set up")
	}
	// Conflating "no project" with "a project that just started" is how a fresh
	// checkout gets told to continue a build that does not exist.
	answer := resolveNextAction(got)
	if strings.Contains(answer.Command, "continue") || strings.Contains(answer.Command, "build") {
		t.Errorf("a folder with no project was told to run %q", answer.Command)
	}

	// A state file with no goal is also not a project.
	writeFixtureColonyState(t, colony.ColonyState{Version: "1.0", State: colony.StateIDLE})
	if again := loadNextActionInput(); !again.NoColony {
		t.Error("a state file carrying no goal was treated as a live project")
	}
}

// TestLoadNextActionInputDoesNotMutate is the corollary CLAUDE.md names: an
// inspection must not write. This repository has shipped a --dry-run that
// quietly wrote to saved state for months, so the assertion compares the
// modification time and content of every file under the project's data
// directory before and after.
func TestLoadNextActionInputDoesNotMutate(t *testing.T) {
	newNextActionFixtureStore(t)

	state := normalizedFixtureState(t, colony.ColonyState{
		Version:        "1.0",
		Goal:           fixtureGoal("Ship the billing rewrite"),
		State:          colony.StateEXECUTING,
		CurrentPhase:   1,
		BuildStartedAt: fixtureTime(t, "2026-08-01T10:00:00Z"),
		Milestone:      "Open Chambers",
		Plan:           colony.Plan{Phases: []colony.Phase{fixturePhase(1, "Foundations", colony.PhaseInProgress)}},
	})
	writeFixtureColonyState(t, state)
	if err := store.SaveJSON("session.json", colony.SessionFile{SessionID: "s1", StartedAt: "2026-08-01T10:00:00Z"}); err != nil {
		t.Fatalf("write session fixture: %v", err)
	}

	before := snapshotProjectDataTree(t, store.BasePath())
	first := loadNextActionInput()
	second := loadNextActionInput()
	after := snapshotProjectDataTree(t, store.BasePath())

	if !reflect.DeepEqual(before, after) {
		t.Errorf("the loader changed the project's saved data.\nbefore: %v\nafter:  %v", before, after)
	}
	if !reflect.DeepEqual(first, second) {
		t.Error("loading twice over an unchanged project returned different inputs")
	}
}

type fileFingerprint struct {
	ModTime time.Time
	Size    int64
	Sum     string
}

func snapshotProjectDataTree(t *testing.T, dir string) map[string]fileFingerprint {
	t.Helper()
	snapshot := map[string]fileFingerprint{}
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return relErr
		}
		snapshot[rel] = fileFingerprint{
			ModTime: info.ModTime(),
			Size:    info.Size(),
			Sum:     fmt.Sprintf("%x", sha256.Sum256(data)),
		}
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot data directory: %v", err)
	}
	return snapshot
}

// TestLoaderContainsNoCommandDecision keeps the split honest: a branch that
// chooses a command belongs in the pure resolver, where it can be tested
// without a filesystem.
func TestLoaderContainsNoCommandDecision(t *testing.T) {
	source, err := os.ReadFile("next_action_input.go")
	if err != nil {
		t.Fatalf("read loader source: %v", err)
	}
	for _, forbidden := range []string{"candidateCommand(", "availableCommand(", "nextActionCandidates", `"aether `} {
		if strings.Contains(string(source), forbidden) {
			t.Errorf("cmd/next_action_input.go references %q; choosing a command is the pure resolver's job", forbidden)
		}
	}
}

// TestResolverIsFreeOfImpureReferences reads the resolver's own source and
// fails if it reaches for the store, the environment, or platform detection.
// Purity is the property that makes every behaviour above testable without a
// filesystem; asserting it in prose only would not keep it.
func TestResolverIsFreeOfImpureReferences(t *testing.T) {
	// Parse and re-print so comments are dropped: a comment explaining that the
	// resolver never touches the store must not itself trip the check.
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "next_action.go", nil, 0)
	if err != nil {
		t.Fatalf("parse resolver source: %v", err)
	}
	var code bytes.Buffer
	if err := printer.Fprint(&code, fset, file); err != nil {
		t.Fatalf("print resolver source: %v", err)
	}
	source := code.Bytes()

	// Word-boundary matching, so ordinary prose containing "restore" is not
	// mistaken for a reference to the package store.
	for _, forbidden := range []string{`\bstore\b`, `os\.Getenv`, `detectPlatform`, `os\.ReadFile`, `os\.Stat`, `os\.Open`} {
		if regexp.MustCompile(forbidden).Match(source) {
			t.Errorf("cmd/next_action.go matches %q in its code; the resolver must be pure and read nothing", forbidden)
		}
	}
}
