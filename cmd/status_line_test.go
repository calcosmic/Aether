package cmd

// Phase 206 plan 02 -- the permanent line.
//
// These tests drive the real cobra command (`aether status-line`) and the
// pure rendering functions behind it. Three properties are load-bearing, and
// each is asserted rather than promised, matching the discipline the
// session-start greeting's own tests already established:
//
//  1. The line DECIDES nothing -- every command it can print comes from
//     resolveNextAction, the one shared what-next decision.
//  2. The line WRITES nothing -- it is redrawn on every turn.
//  3. The registration is a FACT read out of the shipped settings file, not a
//     hope about what the file contains.

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// runStatusLineHook executes the real cobra command with an empty payload on
// stdin (the status line reads nothing from the platform) and returns
// everything it printed.
func runStatusLineHook(t *testing.T) string {
	t.Helper()
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	setHookStdin(t, "")
	rootCmd.SetArgs([]string{"status-line"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("status-line returned an error: %v", err)
	}
	return buf.String()
}

// TestStatusLineIsRegistered is what makes the line a fact rather than a
// hope: read out of the shipped settings file itself, and checked against
// the live command tree, not asserted about in prose.
func TestStatusLineIsRegistered(t *testing.T) {
	settings := readShippedHookSettings(t)

	if settings.StatusLine.Type != "command" {
		t.Fatalf(".claude/settings.json statusLine.type = %q, want %q", settings.StatusLine.Type, "command")
	}
	if settings.StatusLine.Command != "aether status-line" {
		t.Fatalf(".claude/settings.json statusLine.command = %q, want %q", settings.StatusLine.Command, "aether status-line")
	}

	target, _, err := rootCmd.Find([]string{"status-line"})
	if err != nil || target == nil || target == rootCmd {
		t.Fatal("`aether status-line` is registered in .claude/settings.json but is not a registered command")
	}
}

// TestStatusLineComesFromTheSharedDecision proves the line has no opinion of
// its own: half one drives resolveNextAction over a real fixture and checks
// the rendered line carries exactly what that one decision produced; half two
// is the structural AST guard -- a command literal written inside
// status_line.go would be a rival decider being born.
func TestStatusLineComesFromTheSharedDecision(t *testing.T) {
	t.Run("renders the phase, the open task and the shared command", func(t *testing.T) {
		longGoal := "Wire the billing reconciliation job into the nightly batch runner so it never drifts from the ledger"
		state := normalizedFixtureState(t, colony.ColonyState{
			Version:        "1.0",
			Goal:           fixtureGoal("Ship the billing rewrite"),
			State:          colony.StateEXECUTING,
			CurrentPhase:   2,
			BuildStartedAt: runningBuildStartedAt(),
			Milestone:      "Open Chambers",
			Plan: colony.Plan{Phases: []colony.Phase{
				fixturePhase(1, "Foundations", colony.PhaseCompleted),
				{
					ID:          2,
					Name:        "Billing engine",
					Description: "Billing engine description",
					Status:      colony.PhaseInProgress,
					Mode:        colony.PhaseModeProduction,
					Tasks: []colony.Task{{
						ID:     fixtureTaskID("phase-2-task-1"),
						Goal:   longGoal,
						Status: colony.TaskPending,
					}},
					SuccessCriteria: []string{"Billing engine is done"},
				},
			}},
		})

		answer := resolveNextAction(nextActionInput{State: state})
		if answer.Projection == nil {
			t.Fatal("resolveNextAction produced no lifecycle projection")
		}
		line := statusLineText(answer, "claude")
		if strings.TrimSpace(line) == "" {
			t.Fatal("the fixture produced no line at all, so this test would pass vacuously")
		}

		if !strings.Contains(line, "phase 2 of 2") {
			t.Errorf("line does not name the phase number and total:\n%s", line)
		}
		if !strings.Contains(line, "Billing engine") {
			t.Errorf("line does not name the phase name:\n%s", line)
		}

		wantTaskLabel := statusLineTaskLabel(answer.Projection.Tasks.Value)
		if wantTaskLabel == "" {
			t.Fatal("the fixture's own task label function returned nothing, so this test would prove nothing")
		}
		if !strings.Contains(line, wantTaskLabel) {
			t.Errorf("line does not carry the shortened task label %q:\n%s", wantTaskLabel, line)
		}
		if strings.Contains(line, longGoal) {
			t.Errorf("line carries the raw, unshortened task goal:\n%s", line)
		}

		wantCommand := translateHintCommandsForPlatform(answer.Command, "claude")
		if wantCommand == "" {
			t.Fatal("the fixture's own shared decision named no command, so this test would prove nothing")
		}
		if !strings.Contains(line, "next: "+wantCommand) {
			t.Errorf("line does not name the shared decision's own command %q:\n%s", wantCommand, line)
		}
	})

	t.Run("status_line.go spells no command of its own", func(t *testing.T) {
		fset := token.NewFileSet()
		parsed, err := parser.ParseFile(fset, "status_line.go", nil, 0)
		if err != nil {
			t.Fatalf("parse status_line.go: %v", err)
		}

		checkStringLiterals := func(node ast.Node, owner string) {
			ast.Inspect(node, func(n ast.Node) bool {
				lit, ok := n.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return true
				}
				text, err := strconv.Unquote(lit.Value)
				if err != nil {
					return true
				}
				if strings.Contains(text, "aether ") || strings.HasPrefix(text, "/ant-") {
					t.Errorf("%s spells a command of its own (%q). "+
						"Every command must come from the one resolver's candidate set.", owner, text)
				}
				return true
			})
		}

		found := map[string]bool{}
		for _, decl := range parsed.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Name.Name == "statusLineText" || d.Name.Name == "statusLineTaskLabel" {
					found[d.Name.Name] = true
					checkStringLiterals(d.Body, d.Name.Name)
				}
			case *ast.GenDecl:
				if d.Tok != token.VAR {
					continue
				}
				for _, spec := range d.Specs {
					value, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for i, name := range value.Names {
						if name.Name != "statusLineCmd" || i >= len(value.Values) {
							continue
						}
						found["statusLineCmd"] = true
						checkStringLiterals(value.Values[i], "statusLineCmd")
					}
				}
			}
		}

		for _, name := range []string{"statusLineText", "statusLineTaskLabel", "statusLineCmd"} {
			if !found[name] {
				t.Fatalf("%s was not found in cmd/status_line.go -- this test cannot protect what it cannot see", name)
			}
		}
	})
}

// TestStatusLineIsSilentWithoutAProject is the "do not draw a line for a
// folder that has never used Aether" rule, matching the session-start
// greeting's own silence rule.
func TestStatusLineIsSilentWithoutAProject(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	// A genuinely empty repository, not merely a nil in-process store --
	// rootCmd's PersistentPreRunE re-derives `store` from AETHER_ROOT /
	// COLONY_DATA_DIR on every Execute, so an isolated, state-file-free
	// repository root is what actually exercises "no project here", the
	// same authority every other command test binds through.
	newTestStoreCmd(t)

	if out := runStatusLineHook(t); strings.TrimSpace(out) != "" {
		t.Errorf("a folder with no project produced a status line:\n%s", out)
	}
}

// TestStatusLineChangesNothingAndRepeatsItself is the mutation lock: the line
// is redrawn on every turn, so a write here would run constantly. This
// repository has shipped inspection surfaces that quietly wrote to saved
// state for months while documenting that they did not; this is another lock
// of that exact shape.
func TestStatusLineChangesNothingAndRepeatsItself(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	s, tmpDir := newTestStoreCmd(t)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	store = s

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

	before := snapshotProjectDataTree(t, store.BasePath())
	var first string
	for i := 0; i < 100; i++ {
		out := runStatusLineHook(t)
		if i == 0 {
			first = out
			continue
		}
		if out != first {
			t.Fatalf("run %d produced a different line than the first.\nfirst: %q\nrun %d: %q", i, first, i, out)
		}
	}
	after := snapshotProjectDataTree(t, store.BasePath())

	if !reflect.DeepEqual(before, after) {
		t.Errorf("status-line changed the project's saved data.\nbefore: %v\nafter:  %v", before, after)
	}
	if strings.TrimSpace(first) == "" {
		t.Fatal("the fixture produced no line at all, so this test would pass vacuously")
	}
}

// TestStatusLineIsSafeUnderConcurrentReads is the concurrency guarantee, and
// it is proved under the race detector -- that is where a shared-state read
// race would actually be caught. It deliberately drives statusLineText
// directly rather than the cobra command: the package's stdout writer is
// shared, so driving the command concurrently would test the test harness's
// own locking rather than the reader this task is about.
func TestStatusLineIsSafeUnderConcurrentReads(t *testing.T) {
	state := normalizedFixtureState(t, colony.ColonyState{
		Version:        "1.0",
		Goal:           fixtureGoal("Ship the billing rewrite"),
		State:          colony.StateEXECUTING,
		CurrentPhase:   1,
		BuildStartedAt: runningBuildStartedAt(),
		Milestone:      "Open Chambers",
		Plan:           colony.Plan{Phases: []colony.Phase{fixturePhase(1, "Foundations", colony.PhaseInProgress)}},
	})
	answer := resolveNextAction(nextActionInput{State: state})
	want := statusLineText(answer, "claude")
	if strings.TrimSpace(want) == "" {
		t.Fatal("the fixture produced no line at all, so this test would pass vacuously")
	}

	const readers = 50
	var wg sync.WaitGroup
	results := make([]string, readers)
	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = statusLineText(answer, "claude")
		}(i)
	}
	wg.Wait()

	for i, got := range results {
		if got != want {
			t.Errorf("reader %d produced %q, want %q", i, got, want)
		}
	}
}

// TestStatusLineSpeaksPlainEnglish is the plain-English lock, over the three
// fixture shapes the plan names: mid-phase, every task finished, and no
// tasks at all.
func TestStatusLineSpeaksPlainEnglish(t *testing.T) {
	cases := []struct {
		name  string
		state colony.ColonyState
	}{
		{
			name: "mid-phase",
			state: colony.ColonyState{
				Version:        "1.0",
				Goal:           fixtureGoal("Ship the billing rewrite"),
				State:          colony.StateEXECUTING,
				CurrentPhase:   1,
				BuildStartedAt: runningBuildStartedAt(),
				Milestone:      "Open Chambers",
				Plan:           colony.Plan{Phases: []colony.Phase{fixturePhase(1, "Foundations", colony.PhaseInProgress)}},
			},
		},
		{
			name: "every task finished",
			state: colony.ColonyState{
				Version:      "1.0",
				Goal:         fixtureGoal("Ship the billing rewrite"),
				State:        colony.StateBUILT,
				CurrentPhase: 1,
				Milestone:    "Open Chambers",
				Plan:         colony.Plan{Phases: []colony.Phase{fixturePhase(1, "Foundations", colony.PhaseCompleted)}},
			},
		},
		{
			name: "no tasks at all",
			state: colony.ColonyState{
				Version:      "1.0",
				Goal:         fixtureGoal("Ship the billing rewrite"),
				State:        colony.StateREADY,
				CurrentPhase: 1,
				Milestone:    "Open Chambers",
				Plan: colony.Plan{Phases: []colony.Phase{{
					ID:              1,
					Name:            "Foundations",
					Description:     "Foundations description",
					Status:          colony.PhaseReady,
					Mode:            colony.PhaseModeProduction,
					SuccessCriteria: []string{"Foundations is done"},
				}}},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := normalizedFixtureState(t, tc.state)
			answer := resolveNextAction(nextActionInput{State: state})
			line := statusLineText(answer, "claude")
			if strings.TrimSpace(line) == "" {
				t.Fatal("the fixture produced no line at all, so this test would pass vacuously")
			}
			if violations := untranslatedRepoWords(line); len(violations) > 0 {
				t.Errorf("line uses an unexplained repo word: %v\nline: %s", violations, line)
			}
			if violations := rawStateTokenLeaks(line); len(violations) > 0 {
				t.Errorf("line leaks a raw bookkeeping token: %v\nline: %s", violations, line)
			}
		})
	}
}

// TestStatusLineInstallOnlyWhenTheProjectHasNone is the merge-ownership lock:
// mergeSettingsMaps only adds a top-level key Aether ships when the
// downstream file does not already have one, so a project's own status line
// survives and a project with none gets Aether's.
func TestStatusLineInstallOnlyWhenTheProjectHasNone(t *testing.T) {
	template := []byte(`{
  "hooks": {
    "SessionStart": [
      {"matcher": "startup", "hooks": [{"type": "command", "command": "aether hook-session-start", "timeout": 10}]}
    ]
  },
  "statusLine": {"type": "command", "command": "aether status-line", "padding": 0}
}`)

	t.Run("a project with its own status line keeps it", func(t *testing.T) {
		existing := []byte(`{"statusLine": {"type": "command", "command": "./my-custom-status-line.sh"}}`)
		merged, err := mergeClaudeSettings(template, existing)
		if err != nil {
			t.Fatalf("merge failed: %v", err)
		}
		if !strings.Contains(string(merged), "./my-custom-status-line.sh") {
			t.Fatalf("the project's own status line was replaced:\n%s", merged)
		}
		if strings.Contains(string(merged), "aether status-line") {
			t.Fatalf("aether's status line was installed over the project's own:\n%s", merged)
		}
	})

	t.Run("a project with none gets aether's", func(t *testing.T) {
		existing := []byte(`{"gsdMarker": "do-not-touch"}`)
		merged, err := mergeClaudeSettings(template, existing)
		if err != nil {
			t.Fatalf("merge failed: %v", err)
		}
		if !strings.Contains(string(merged), "aether status-line") {
			t.Fatalf("aether's status line was not installed:\n%s", merged)
		}

		again, err := mergeClaudeSettings(template, merged)
		if err != nil {
			t.Fatalf("second merge failed: %v", err)
		}
		if !bytes.Equal(merged, again) {
			t.Fatalf("repeated merge rewrote bytes:\nfirst:\n%s\nsecond:\n%s", merged, again)
		}
	})
}

// fixtureTaskID returns a pointer to a task ID, matching fixturePhase's own
// shape for a task built by hand rather than through that helper.
func fixtureTaskID(id string) *string {
	return &id
}
