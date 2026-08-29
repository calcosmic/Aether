package cmd

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
