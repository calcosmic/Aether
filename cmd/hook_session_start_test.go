package cmd

// Phase 197 plan 03 -- the greeting the program owns.
//
// Four shipped documents used to ASK the assistant to look at the saved
// session file on the first message of a conversation and report what it
// found. A request is not a mechanism: it runs only when it is noticed. This
// file's tests are the mechanism's proof -- a hidden hook the platform fires
// when a session opens, which prints the one card resolveNextAction produces
// and writes nothing at all.
//
// Three properties are load-bearing here, and each is asserted rather than
// promised:
//
//  1. The hook DECIDES nothing. It has no branch that chooses a command; every
//     command it can name comes from the one resolver's candidate set.
//  2. The hook WRITES nothing. It fires before the owner has typed anything,
//     and this repository has twice shipped an inspection surface that quietly
//     wrote to saved state while documenting that it did not.
//  3. The registration is a FACT, not a hope. The shipped hook settings file is
//     read as data and required to name the command at all three moments the
//     owner arrives with no context.

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// runningBuildStartedAt is a build that started recently enough that the
// runtime does not consider it abandoned. Derived from abandonedBuildThreshold
// rather than typed as a plausible-looking date, so a fixture meant to be "a
// build that is running" cannot silently become "a build that stalled".
func runningBuildStartedAt() *time.Time {
	started := time.Now().UTC().Add(-abandonedBuildThreshold / 2)
	return &started
}

// runSessionStartHook executes the real cobra command with an empty hook
// payload on stdin and returns everything it printed.
func runSessionStartHook(t *testing.T) string {
	t.Helper()
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	setHookStdin(t, `{"hook_event_name":"SessionStart","source":"startup"}`)
	rootCmd.SetArgs([]string{"hook-session-start"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("hook-session-start returned an error: %v", err)
	}
	return buf.String()
}

// newSessionStartProject sets up a temp project with a live store, pins the
// platform so the card's command spelling is the runtime form on every machine,
// isolates the hub directory (198.2 plan 03: the greeting now reads the hub's
// QUEEN.md for the preferences line, so every session-start test must run
// against an empty, machine-independent hub rather than the real ~/.aether),
// and resets the command tree afterwards.
func newSessionStartProject(t *testing.T) {
	t.Helper()
	saveGlobalsCmd(t)
	resetRootCmd(t)
	t.Setenv("AETHER_PLATFORM", "codex")
	t.Setenv("AETHER_HUB_DIR", t.TempDir())
	s, tmpDir := newTestStoreCmd(t)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	store = s
}

// TestSessionStartCardReflectsState is criterion 4's named test. Each sub-test
// writes one saved state through the runtime's own store and requires the card
// the hook prints to name that state's own facts and its own next command.
func TestSessionStartCardReflectsState(t *testing.T) {
	cases := []struct {
		name         string
		state        colony.ColonyState
		wantCommand  string
		wantContains []string
	}{
		{
			// Mid-build: the owner opens a chat with a build already running.
			name: "a project part-way through a build names the phase and what to run next",
			state: colony.ColonyState{
				Version:        "1.0",
				Goal:           fixtureGoal("Ship the billing rewrite"),
				State:          colony.StateEXECUTING,
				CurrentPhase:   2,
				BuildStartedAt: runningBuildStartedAt(),
				Milestone:      "Open Chambers",
				Plan: colony.Plan{Phases: []colony.Phase{
					fixturePhase(1, "Foundations", colony.PhaseCompleted),
					fixturePhase(2, "Billing engine", colony.PhaseInProgress),
				}},
			},
			wantCommand:  "aether continue",
			wantContains: []string{"Ship the billing rewrite", "Billing engine"},
		},
		{
			// Finished: every phase done, but the project has not been signed
			// off yet. The card must say what finishing means in plain words.
			name: "a finished project is told what signing off means",
			state: colony.ColonyState{
				Version:      "1.0",
				Goal:         fixtureGoal("Ship the billing rewrite"),
				State:        colony.StateCOMPLETED,
				CurrentPhase: 2,
				Milestone:    "Open Chambers",
				Plan: colony.Plan{Phases: []colony.Phase{
					fixturePhase(1, "Foundations", colony.PhaseCompleted),
					fixturePhase(2, "Billing engine", colony.PhaseCompleted),
				}},
			},
			wantCommand:  "aether seal",
			wantContains: []string{"Ship the billing rewrite"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			newSessionStartProject(t)
			state := normalizedFixtureState(t, tc.state)
			if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
				t.Fatalf("write fixture state through the runtime's own store: %v", err)
			}

			card := runSessionStartHook(t)
			if strings.TrimSpace(card) == "" {
				t.Fatal("the hook printed nothing for a project that has a colony")
			}
			if !strings.Contains(card, tc.wantCommand) {
				t.Errorf("the card does not name %q:\n%s", tc.wantCommand, card)
			}
			for _, want := range tc.wantContains {
				if !strings.Contains(card, want) {
					t.Errorf("the card does not mention %q:\n%s", want, card)
				}
			}
		})
	}

	// The command the card names must be the same command every other surface
	// names for that state. Compared against the one resolver, over the same
	// saved state, read back through the runtime's own loader.
	t.Run("the command named is the one every other surface names", func(t *testing.T) {
		newSessionStartProject(t)
		state := normalizedFixtureState(t, colony.ColonyState{
			Version:        "1.0",
			Goal:           fixtureGoal("Ship the billing rewrite"),
			State:          colony.StateEXECUTING,
			CurrentPhase:   2,
			BuildStartedAt: runningBuildStartedAt(),
			Milestone:      "Open Chambers",
			Plan: colony.Plan{Phases: []colony.Phase{
				fixturePhase(1, "Foundations", colony.PhaseCompleted),
				fixturePhase(2, "Billing engine", colony.PhaseInProgress),
			}},
		})
		if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
			t.Fatalf("write fixture state: %v", err)
		}

		card := runSessionStartHook(t)
		answer := resolveNextAction(loadNextActionInput())
		if !strings.Contains(card, answer.Command) {
			t.Errorf("the greeting names a different command from the one resolver (%q):\n%s", answer.Command, card)
		}
		if !strings.Contains(card, nextCommandForHookState(state)) {
			t.Errorf("the hook decider in cmd/hook_cmds.go answered %q, which the card does not name:\n%s",
				nextCommandForHookState(state), card)
		}
	})
}

// TestSessionStartCardIsSilentWithoutAColony is the "do not greet a repository
// that has never used Aether" rule. Aether is installed in repositories that
// are not running a colony, and a greeting there is noise.
func TestSessionStartCardIsSilentWithoutAColony(t *testing.T) {
	t.Run("an empty project is not greeted", func(t *testing.T) {
		newSessionStartProject(t)
		if out := runSessionStartHook(t); strings.TrimSpace(out) != "" {
			t.Errorf("a project with no colony was greeted anyway:\n%s", out)
		}
	})

	// A leftover state file carrying no goal is not a project either -- the
	// loader's own no-colony signal, not an empty-state guess.
	t.Run("a leftover state file with no goal is not a project", func(t *testing.T) {
		newSessionStartProject(t)
		if err := store.SaveJSON("COLONY_STATE.json", colony.ColonyState{Version: "1.0", State: colony.StateIDLE}); err != nil {
			t.Fatalf("write leftover state: %v", err)
		}
		if out := runSessionStartHook(t); strings.TrimSpace(out) != "" {
			t.Errorf("a leftover state file with no goal produced a greeting:\n%s", out)
		}
	})
}

// TestSessionStartHookDoesNotMutate is the third lock of its kind in this
// repository. Two shipped inspection commands wrote to saved state for months
// while their own documentation said they did not; both now have a test with
// exactly this shape. The greeting fires before the owner has typed anything,
// so it must leave the project's saved data byte-identical.
func TestSessionStartHookDoesNotMutate(t *testing.T) {
	newSessionStartProject(t)

	state := normalizedFixtureState(t, colony.ColonyState{
		Version:        "1.0",
		Goal:           fixtureGoal("Ship the billing rewrite"),
		State:          colony.StateEXECUTING,
		CurrentPhase:   1,
		BuildStartedAt: runningBuildStartedAt(),
		Milestone:      "Open Chambers",
		Plan:           colony.Plan{Phases: []colony.Phase{fixturePhase(1, "Foundations", colony.PhaseInProgress)}},
	})
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("write fixture state: %v", err)
	}
	if err := store.SaveJSON("session.json", colony.SessionFile{SessionID: "s1", StartedAt: "2026-08-01T10:00:00Z"}); err != nil {
		t.Fatalf("write session fixture: %v", err)
	}

	before := snapshotProjectDataTree(t, store.BasePath())
	first := runSessionStartHook(t)
	second := runSessionStartHook(t)
	after := snapshotProjectDataTree(t, store.BasePath())

	if !reflect.DeepEqual(before, after) {
		t.Errorf("the greeting changed the project's saved data.\nbefore: %v\nafter:  %v", before, after)
	}
	if first != second {
		t.Errorf("two greetings over an unchanged project differed.\nfirst:\n%s\nsecond:\n%s", first, second)
	}
	if strings.TrimSpace(first) == "" {
		t.Fatal("the fixture produced no card at all, so this test would pass vacuously")
	}
}

// TestSessionStartHookChoosesNoCommandOfItsOwn is the structural half. The hook
// is a renderer: a command name written inside it is a fifth decider being
// born, which is the drift this whole phase exists to end.
func TestSessionStartHookChoosesNoCommandOfItsOwn(t *testing.T) {
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, "hook_cmds.go", nil, 0)
	if err != nil {
		t.Fatalf("parse hook_cmds.go: %v", err)
	}

	found := false
	for _, decl := range parsed.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		for _, spec := range gen.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range value.Names {
				if name.Name != "hookSessionStartCmd" || i >= len(value.Values) {
					continue
				}
				found = true
				ast.Inspect(value.Values[i], func(node ast.Node) bool {
					lit, ok := node.(*ast.BasicLit)
					if !ok || lit.Kind != token.STRING {
						return true
					}
					text, err := strconv.Unquote(lit.Value)
					if err != nil {
						return true
					}
					if strings.Contains(text, "aether ") {
						t.Errorf("hookSessionStartCmd spells a command of its own (%q). "+
							"Every command must come from the one resolver's candidate set.", text)
					}
					return true
				})
			}
		}
	}

	if !found {
		t.Fatal("hookSessionStartCmd was not found in cmd/hook_cmds.go -- this test cannot protect what it cannot see")
	}
}

// ---------------------------------------------------------------------------
// The registration (task 2)
// ---------------------------------------------------------------------------

// sessionStartArrivalMoments are the three moments the owner arrives with no
// context: a session opened fresh, a session resumed, and a conversation
// continued after being cleared.
var sessionStartArrivalMoments = []string{"startup", "resume", "clear"}

// claudeHookSettingsFile is the shape of the shipped hook settings file, read
// as data. The reachability scanner reads the same shape.
type claudeHookSettingsFile struct {
	Hooks map[string][]struct {
		Matcher string `json:"matcher"`
		Hooks   []struct {
			Type    string `json:"type"`
			Command string `json:"command"`
			Timeout int    `json:"timeout"`
		} `json:"hooks"`
	} `json:"hooks"`
}

func readShippedHookSettings(t *testing.T) claudeHookSettingsFile {
	t.Helper()
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("findRepoRoot: %v", err)
	}
	path := filepath.Join(root, ".claude", "settings.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var parsed claudeHookSettingsFile
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("%s is not valid JSON: %v", path, err)
	}
	return parsed
}

// TestSessionStartHookIsRegistered is what makes the greeting a fact rather
// than a hope. This repository has shipped several features whose only defect
// was that nothing ever called them; the registration is therefore read out of
// the shipped settings file itself, not asserted about in prose.
func TestSessionStartHookIsRegistered(t *testing.T) {
	settings := readShippedHookSettings(t)

	entries, ok := settings.Hooks["SessionStart"]
	if !ok || len(entries) == 0 {
		t.Fatal(".claude/settings.json registers no SessionStart hook at all, so the greeting never runs")
	}

	covered := map[string]bool{}
	sawTimeout := false
	for _, entry := range entries {
		names := false
		for _, h := range entry.Hooks {
			if strings.TrimSpace(h.Command) == "aether hook-session-start" {
				names = true
				if h.Timeout > 0 {
					sawTimeout = true
				}
			}
		}
		if !names {
			continue
		}
		for _, matcher := range strings.Split(entry.Matcher, "|") {
			covered[strings.TrimSpace(matcher)] = true
		}
	}

	if len(covered) == 0 {
		t.Fatal("no SessionStart entry runs `aether hook-session-start` -- the command is registered with cobra but nothing fires it")
	}
	for _, moment := range sessionStartArrivalMoments {
		if !covered[moment] {
			t.Errorf("the session-start hook does not cover the %q moment; the owner arrives with no context at all three of %v",
				moment, sessionStartArrivalMoments)
		}
	}
	if !sawTimeout {
		t.Error("the session-start hook entry carries no timeout; a hook that can hang blocks the session opening")
	}

	// The command the settings file names must be a command the program
	// actually has. A registration naming a command that does not exist is the
	// same defect as a command nothing calls, pointed the other way.
	target, _, err := rootCmd.Find([]string{"hook-session-start"})
	if err != nil || target == nil || target == rootCmd {
		t.Fatal("`aether hook-session-start` is registered in .claude/settings.json but is not a registered command")
	}
}
