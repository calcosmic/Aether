package cmd

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// TestBlockedLastTimeIsReadFromTheStructuredReport pins lastContinueEndedBlocked
// (D-09's third "waiting on you" signal) to the structured saved report, never
// to the pipe-delimited event string ("...|continue_blocked|..."). A phase
// whose report says it advanced yields no signal; a phase whose report says it
// did not yields the blocked-last-time signal with a plain-English reason; a
// missing or unparseable report yields no signal and never errors.
func TestBlockedLastTimeIsReadFromTheStructuredReport(t *testing.T) {
	cases := []struct {
		name        string
		setup       func(t *testing.T, phaseID int)
		wantBlocked bool
	}{
		{
			name: "advanced report yields no signal",
			setup: func(t *testing.T, phaseID int) {
				rel := continuePlanArtifactsPath(phaseID, "continue.json")
				if err := store.SaveJSON(rel, codexContinueReport{Phase: phaseID, Advanced: true}); err != nil {
					t.Fatalf("SaveJSON: %v", err)
				}
			},
			wantBlocked: false,
		},
		{
			name: "not-advanced report yields the blocked-last-time signal",
			setup: func(t *testing.T, phaseID int) {
				rel := continuePlanArtifactsPath(phaseID, "continue.json")
				if err := store.SaveJSON(rel, codexContinueReport{Phase: phaseID, Advanced: false}); err != nil {
					t.Fatalf("SaveJSON: %v", err)
				}
			},
			wantBlocked: true,
		},
		{
			name:        "missing report yields no signal and no error",
			setup:       func(t *testing.T, phaseID int) {},
			wantBlocked: false,
		},
		{
			name: "unparseable report yields no signal and no error",
			setup: func(t *testing.T, phaseID int) {
				rel := continuePlanArtifactsPath(phaseID, "continue.json")
				full := filepath.Join(store.BasePath(), rel)
				if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
					t.Fatalf("MkdirAll: %v", err)
				}
				if err := os.WriteFile(full, []byte("not valid json{{{"), 0644); err != nil {
					t.Fatalf("WriteFile: %v", err)
				}
			},
			wantBlocked: false,
		},
	}

	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			saveGlobalsCmd(t)
			s, _ := newTestStoreCmd(t)
			store = s
			phaseID := 900 + i
			tc.setup(t, phaseID)

			blocked, reason := lastContinueEndedBlocked(phaseID)
			if blocked != tc.wantBlocked {
				t.Fatalf("blocked = %v, want %v (reason=%q)", blocked, tc.wantBlocked, reason)
			}
			if tc.wantBlocked && strings.TrimSpace(reason) == "" {
				t.Fatalf("expected a non-empty plain-English reason when blocked")
			}
			if !tc.wantBlocked && reason != "" {
				t.Fatalf("expected an empty reason when not blocked, got %q", reason)
			}
		})
	}

	t.Run("body contains no string-splitting of an event record", func(t *testing.T) {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "ceremony_team_checkin.go", nil, 0)
		if err != nil {
			t.Fatalf("parser.ParseFile: %v", err)
		}
		var fn *ast.FuncDecl
		ast.Inspect(file, func(n ast.Node) bool {
			if decl, ok := n.(*ast.FuncDecl); ok && decl.Name.Name == "lastContinueEndedBlocked" {
				fn = decl
				return false
			}
			return true
		})
		if fn == nil {
			t.Fatalf("lastContinueEndedBlocked not found in cmd/ceremony_team_checkin.go")
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkgIdent, ok := sel.X.(*ast.Ident)
			if !ok || pkgIdent.Name != "strings" {
				return true
			}
			switch sel.Sel.Name {
			case "Split", "SplitN", "Fields", "Cut":
				t.Fatalf("lastContinueEndedBlocked must not string-split an event record; found strings.%s", sel.Sel.Name)
			}
			return true
		})
	})
}

// TestOnlyTheThreeNamedSignalsRaiseTheHeadsUp proves buildStartBlockerSignals
// returns exactly D-09's three named signals and nothing else -- an
// attention note (FOCUS/FEEDBACK) or an ordinary flag never raises the
// heads-up, and each named signal in isolation produces exactly one entry
// naming itself.
func TestOnlyTheThreeNamedSignalsRaiseTheHeadsUp(t *testing.T) {
	t.Run("no signals present yields an empty list", func(t *testing.T) {
		saveGlobalsCmd(t)
		s, _ := newTestStoreCmd(t)
		store = s

		manifest := codexBuildManifest{Phase: 1}
		got := buildStartBlockerSignals(manifest)
		if len(got) != 0 {
			t.Fatalf("expected no signals for an ordinary manifest, got %+v", got)
		}
	})

	t.Run("forced reviewer alone produces exactly one entry", func(t *testing.T) {
		saveGlobalsCmd(t)
		s, _ := newTestStoreCmd(t)
		store = s

		manifest := codexBuildManifest{
			Phase: 2,
			ForcedReviewers: []codexForcedReviewerRecord{
				{
					Caste:   "gatekeeper",
					Signals: []string{"credentials/auth"},
					Matches: []string{"password"},
					Sources: []string{"plan wording"},
					Reason:  "the plan mentions a password reset",
				},
			},
		}
		got := buildStartBlockerSignals(manifest)
		if len(got) != 1 {
			t.Fatalf("expected exactly one signal for a live forced reviewer, got %+v", got)
		}
		if got[0].Name != "forced-reviewer" {
			t.Fatalf("signal name = %q, want %q", got[0].Name, "forced-reviewer")
		}
		if strings.TrimSpace(got[0].Reason) == "" {
			t.Fatalf("expected a plain-English reason")
		}
	})

	t.Run("unanswered boundary question alone produces exactly one entry", func(t *testing.T) {
		saveGlobalsCmd(t)
		s, _ := newTestStoreCmd(t)
		store = s

		manifest := codexBuildManifest{Phase: 3, BoundaryQuestionCount: 1}
		got := buildStartBlockerSignals(manifest)
		if len(got) != 1 {
			t.Fatalf("expected exactly one signal for an unanswered boundary question, got %+v", got)
		}
		if got[0].Name != "unanswered-question" {
			t.Fatalf("signal name = %q, want %q", got[0].Name, "unanswered-question")
		}
	})

	t.Run("last continue ended blocked alone produces exactly one entry", func(t *testing.T) {
		saveGlobalsCmd(t)
		s, _ := newTestStoreCmd(t)
		store = s

		phaseID := 4
		rel := continuePlanArtifactsPath(phaseID, "continue.json")
		if err := store.SaveJSON(rel, codexContinueReport{Phase: phaseID, Advanced: false}); err != nil {
			t.Fatalf("SaveJSON: %v", err)
		}

		manifest := codexBuildManifest{Phase: phaseID}
		got := buildStartBlockerSignals(manifest)
		if len(got) != 1 {
			t.Fatalf("expected exactly one signal for a blocked last continue, got %+v", got)
		}
		if got[0].Name != "last-continue-blocked" {
			t.Fatalf("signal name = %q, want %q", got[0].Name, "last-continue-blocked")
		}
	})
}

// TestBuildStartBlockerAdvisory is the pure, table-driven regression guard for
// decideBuildBlockerAdvisory/renderBuildBlockerAdvisory (D-08/D-09): with no
// signals nothing is printed and no question is added; with signals present,
// printing is unconditional and only the question is gated by interactivity.
func TestBuildStartBlockerAdvisory(t *testing.T) {
	sig := buildBlockerSignal{
		Name:   "last-continue-blocked",
		Reason: "the last time this phase's work was checked, it did not pass and stopped",
	}

	t.Run("no signals: nothing printed, no question added", func(t *testing.T) {
		advisory := decideBuildBlockerAdvisory(nil, false)
		if len(advisory.Signals) != 0 || advisory.Ask {
			t.Fatalf("expected an empty advisory with nothing to ask, got %+v", advisory)
		}
		if got := renderBuildBlockerAdvisory(advisory); got != "" {
			t.Fatalf("expected no rendered output for an empty advisory, got %q", got)
		}
	})

	t.Run("signals present, interactive: names each signal and asks the one question", func(t *testing.T) {
		advisory := decideBuildBlockerAdvisory([]buildBlockerSignal{sig}, false)
		if !advisory.Ask {
			t.Fatalf("expected Ask=true for an interactive run with a signal present")
		}
		rendered := renderBuildBlockerAdvisory(advisory)
		if !strings.Contains(rendered, sig.Reason) {
			t.Fatalf("rendered advisory missing the signal's reason:\n%s", rendered)
		}
		if !strings.Contains(rendered, buildBlockerAdvisoryQuestion) {
			t.Fatalf("rendered advisory missing the carry-on-or-stop question:\n%s", rendered)
		}
	})

	t.Run("signals present, non-interactive: still prints, never asks", func(t *testing.T) {
		advisory := decideBuildBlockerAdvisory([]buildBlockerSignal{sig}, true)
		if advisory.Ask {
			t.Fatalf("expected Ask=false for a non-interactive run")
		}
		rendered := renderBuildBlockerAdvisory(advisory)
		if !strings.Contains(rendered, sig.Reason) {
			t.Fatalf("rendered advisory missing the signal's reason even though printing is unconditional:\n%s", rendered)
		}
		if strings.Contains(rendered, buildBlockerAdvisoryQuestion) {
			t.Fatalf("a non-interactive run must never add the question:\n%s", rendered)
		}
	})

	t.Run("decideBuildBlockerAdvisory's parameter list contains no store and no rendered string", func(t *testing.T) {
		data, err := os.ReadFile("build_blocker_advisory.go")
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}
		src := string(data)
		idx := strings.Index(src, "func decideBuildBlockerAdvisory(")
		if idx == -1 {
			t.Fatalf("decideBuildBlockerAdvisory not found in cmd/build_blocker_advisory.go")
		}
		closeIdx := strings.Index(src[idx:], ")")
		if closeIdx == -1 {
			t.Fatalf("could not find the end of decideBuildBlockerAdvisory's parameter list")
		}
		signature := src[idx : idx+closeIdx+1]
		if strings.Contains(strings.ToLower(signature), "store") {
			t.Fatalf("decideBuildBlockerAdvisory's signature must not take a store: %s", signature)
		}
		if !strings.Contains(signature, "signals []buildBlockerSignal") || !strings.Contains(signature, "nonInteractive bool") {
			t.Fatalf("unexpected decideBuildBlockerAdvisory signature: %s", signature)
		}
	})
}

// TestBlockerHeadsUpDispatchesNoWorkers pins that the whole blocker-advisory
// path -- signal computation, the pure decision, and the render -- spawns no
// additional helper. A static source scan for dispatch/spawn call names in
// cmd/build_blocker_advisory.go, not an intermediate record: if a future
// change wires a dispatch call into this file, this test fails.
func TestBlockerHeadsUpDispatchesNoWorkers(t *testing.T) {
	data, err := os.ReadFile("build_blocker_advisory.go")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	src := strings.ToLower(string(data))
	for _, forbidden := range []string{"dispatch(", "spawn(", "workerinvoker", "newcodexworkerinvoker", "agent("} {
		if strings.Contains(src, forbidden) {
			t.Fatalf("cmd/build_blocker_advisory.go must never dispatch or spawn a worker; found %q", forbidden)
		}
	}
}

// blockerAdvisoryFixturePhase mirrors checkinFixturePhase's wording pattern
// (TestCardNamesTheSignalForEveryForcedReviewer) that reliably names the
// credentials/auth risk signal, giving both build lanes a real, non-waived
// forced-reviewer blocker signal to render.
func blockerAdvisoryFixturePhase() colony.Phase {
	taskID := "1.1"
	return colony.Phase{
		ID:          1,
		Name:        "Password reset",
		Description: "Let users reset their password via an emailed token",
		Mode:        colony.PhaseModePrototype,
		Status:      colony.PhaseReady,
		Tasks:       []colony.Task{{ID: &taskID, Goal: "Do the work", Status: colony.TaskPending}},
	}
}

// TestBothBuildLanesEmitTheHeadsUp asserts the rendered bytes on the
// plan-only lane and on the direct build lane -- not an intermediate record
// -- both carry the D-08 heads-up for the same live forced-reviewer signal.
// A guarantee that holds only on one lane is worth nothing (CLAUDE.md).
func TestBothBuildLanesEmitTheHeadsUp(t *testing.T) {
	t.Run("plan-only lane", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		dataDir := setupBuildFlowTest(t)
		setUpCheckinFixtureColony(t, dataDir, blockerAdvisoryFixturePhase())
		t.Setenv("AETHER_OUTPUT_MODE", "visual")

		var buf bytes.Buffer
		stdout = &buf
		rootCmd.SetArgs([]string{"build", "1", "--plan-only", "--light"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("build --plan-only returned error: %v", err)
		}
		rootCmd.SetArgs([]string{})

		out := buf.String()
		if !strings.Contains(out, "a forced reviewer is still waiting for the owner's check-in decision") {
			t.Fatalf("plan-only lane missing the blocker heads-up:\n%s", out)
		}
		if !strings.Contains(out, buildBlockerAdvisoryQuestion) {
			t.Fatalf("plan-only lane missing the carry-on-or-stop question:\n%s", out)
		}
	})

	t.Run("direct build lane", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		dataDir := setupBuildFlowTest(t)
		setUpCheckinFixtureColony(t, dataDir, blockerAdvisoryFixturePhase())
		t.Setenv("AETHER_OUTPUT_MODE", "visual")

		var buf bytes.Buffer
		stdout = &buf
		rootCmd.SetArgs([]string{"build", "1", "--synthetic", "--light"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("build --synthetic returned error: %v", err)
		}
		rootCmd.SetArgs([]string{})

		out := buf.String()
		if !strings.Contains(out, "a forced reviewer is still waiting for the owner's check-in decision") {
			t.Fatalf("direct build lane missing the blocker heads-up:\n%s", out)
		}
		if !strings.Contains(out, buildBlockerAdvisoryQuestion) {
			t.Fatalf("direct build lane missing the carry-on-or-stop question:\n%s", out)
		}
	})
}

// TestNonInteractiveRunsStillPrintTheHeadsUp proves D-09's non-interactive
// distinction end-to-end: --no-checkin still prints the heads-up naming the
// live signal, but never adds the question -- printing is unconditional,
// only asking is gated.
func TestNonInteractiveRunsStillPrintTheHeadsUp(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	setUpCheckinFixtureColony(t, dataDir, blockerAdvisoryFixturePhase())
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	var buf bytes.Buffer
	stdout = &buf
	rootCmd.SetArgs([]string{"build", "1", "--plan-only", "--light", "--no-checkin"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build --plan-only --no-checkin returned error: %v", err)
	}
	rootCmd.SetArgs([]string{})

	out := buf.String()
	if !strings.Contains(out, "a forced reviewer is still waiting for the owner's check-in decision") {
		t.Fatalf("--no-checkin must still print the blocker heads-up:\n%s", out)
	}
	if strings.Contains(out, buildBlockerAdvisoryQuestion) {
		t.Fatalf("--no-checkin must never add the carry-on-or-stop question:\n%s", out)
	}
}
