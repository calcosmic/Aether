package cmd

// Phase 209 plan 01, Task 2 -- proving the route authority is real and
// single. Every fixture here is built by writing real files into a
// t.TempDir() and calling gatherJobSizeFacts; the direct unit table for
// resolveJobSizeRoute itself is the only place a jobSizeFacts literal is
// constructed by hand.

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// TestGoRouteIsComputedNotConstant builds two real temporary repositories --
// one genuinely containing the file the sentence names, one genuinely not --
// and fails if resolveJobSizeRoute returns the identical route for both.
func TestGoRouteIsComputedNotConstant(t *testing.T) {
	sentence := "fix the rounding bug in payment.go"

	present := t.TempDir()
	if err := os.WriteFile(filepath.Join(present, "payment.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write fixture file: %v", err)
	}
	absent := t.TempDir()

	presentDecision := resolveJobSizeRoute(gatherJobSizeFacts(present, sentence))
	absentDecision := resolveJobSizeRoute(gatherJobSizeFacts(absent, sentence))

	if presentDecision.Route == absentDecision.Route {
		t.Fatalf("the identical sentence produced the same route (%q) in two genuinely different repositories -- "+
			"present decision=%+v absent decision=%+v", presentDecision.Route, presentDecision, absentDecision)
	}
	if presentDecision.Route != jobSizeRouteSmall {
		t.Errorf("repository genuinely containing the named file resolved to %q, want %q", presentDecision.Route, jobSizeRouteSmall)
	}
	if absentDecision.Route != jobSizeRouteBig {
		t.Errorf("repository genuinely missing the named file resolved to %q, want %q", absentDecision.Route, jobSizeRouteBig)
	}
	if presentDecision.Reason == "" || absentDecision.Reason == "" {
		t.Error("every returned decision must carry a non-empty reason")
	}
}

// TestGoRouteIgnoresHowTheSentenceIsWorded runs the same underlying job
// against the same repository with four wordings -- plain, with a
// smallness-claiming prefix, with a bigness-claiming suffix, and in
// capitals -- and fails if any two routes differ.
func TestGoRouteIgnoresHowTheSentenceIsWorded(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "billing.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write fixture file: %v", err)
	}

	plain := "update the billing calculation"
	wordings := map[string]string{
		"plain":            plain,
		"smallness-prefix": "just a tiny quick fix: " + plain,
		"bigness-suffix":   plain + " -- this is a massive rewrite of everything",
		"capitals":         strings.ToUpper(plain),
	}

	decisions := map[string]jobSizeDecision{}
	for name, sentence := range wordings {
		decisions[name] = resolveJobSizeRoute(gatherJobSizeFacts(root, sentence))
	}

	baseline := decisions["plain"]
	for name, decision := range decisions {
		if decision.Route != baseline.Route {
			t.Errorf("wording %q resolved to route %q, but the plain wording resolved to %q -- "+
				"the route must come from the size of the change, not the words used", name, decision.Route, baseline.Route)
		}
	}
}

// TestResolveJobSizeRouteBranches exercises every branch of the priority
// order directly, asserting each carries a non-empty reason.
func TestResolveJobSizeRouteBranches(t *testing.T) {
	cases := []struct {
		name  string
		facts jobSizeFacts
		want  jobSizeRoute
	}{
		{
			name:  "accepted plan with outstanding work wins over everything",
			facts: jobSizeFacts{MatchedPaths: []string{"a.go"}, PlanAcceptedUnbuilt: true},
			want:  jobSizeRouteBig,
		},
		{
			name:  "zero matched paths",
			facts: jobSizeFacts{MatchedPaths: nil},
			want:  jobSizeRouteBig,
		},
		{
			name:  "matched paths over the budget",
			facts: jobSizeFacts{MatchedPaths: []string{"a.go", "b.go", "c.go", "d.go"}},
			want:  jobSizeRouteBig,
		},
		{
			name:  "matched paths within the budget",
			facts: jobSizeFacts{MatchedPaths: []string{"a.go"}},
			want:  jobSizeRouteSmall,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			decision := resolveJobSizeRoute(tc.facts)
			if decision.Route != tc.want {
				t.Errorf("route = %q, want %q", decision.Route, tc.want)
			}
			if strings.TrimSpace(decision.Reason) == "" {
				t.Error("decision.Reason must never be empty")
			}
		})
	}
}

// TestGoRouteDirectoryOverBudgetEscalates proves a sentence naming a
// directory holding more files than smallJobFileBudget resolves to the big
// route, with a reason naming the count.
func TestGoRouteDirectoryOverBudgetEscalates(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "internal")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	for _, name := range []string{"a.go", "b.go", "c.go", "d.go", "e.go"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("package internal\n"), 0o644); err != nil {
			t.Fatalf("write fixture file: %v", err)
		}
	}

	decision := resolveJobSizeRoute(gatherJobSizeFacts(root, "refactor the internal package"))
	if decision.Route != jobSizeRouteBig {
		t.Fatalf("route = %q, want %q for a directory holding more files than the budget", decision.Route, jobSizeRouteBig)
	}
	if !strings.Contains(decision.Reason, "5") {
		t.Errorf("reason does not name the matched-file count: %q", decision.Reason)
	}
}

// TestGoRouteAcceptedPlanWithOutstandingWorkAlwaysEscalates proves a
// recorded accepted plan with outstanding work resolves to the big route
// whatever the sentence names, over a repository that genuinely contains
// the named file (so the escalation is provably from the plan facts, not
// from an absent match).
func TestGoRouteAcceptedPlanWithOutstandingWorkAlwaysEscalates(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "payment.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write fixture file: %v", err)
	}
	dataDir := filepath.Join(root, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	goal := "Ship the billing rewrite"
	state := colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateEXECUTING,
		Plan: colony.Plan{Phases: []colony.Phase{
			{ID: 1, Name: "Foundations", Status: colony.PhaseInProgress},
		}},
	}
	writeGoRouteFixtureState(t, dataDir, state)

	facts := gatherJobSizeFacts(root, "fix the rounding bug in payment.go")
	if !facts.PlanAcceptedUnbuilt {
		t.Fatalf("gatherJobSizeFacts did not read the accepted plan's outstanding work: %+v", facts)
	}
	decision := resolveJobSizeRoute(facts)
	if decision.Route != jobSizeRouteBig {
		t.Fatalf("route = %q, want %q when an accepted plan has outstanding work", decision.Route, jobSizeRouteBig)
	}
}

func writeGoRouteFixtureState(t *testing.T, dataDir string, state colony.ColonyState) {
	t.Helper()
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		t.Fatalf("marshal fixture state: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "COLONY_STATE.json"), data, 0o644); err != nil {
		t.Fatalf("write fixture state: %v", err)
	}
}

// TestGoRouteHasOneAuthority has two halves. Half one drives the real
// `aether go` command (runGoJob, its actual command body) over a fixture
// repository and asserts the "route" in its result map equals what
// resolveJobSizeRoute returns for the facts gatherJobSizeFacts produces for
// that same repository and sentence. Half two parses cmd/go_cmd.go with
// go/parser and fails if that file contains any route-deciding comparison
// of its own.
func TestGoRouteHasOneAuthority(t *testing.T) {
	t.Run("aether go's own route matches the authority's answer", func(t *testing.T) {
		saveGlobals(t)
		store = nil
		root := t.TempDir()
		chdirForTest190_05(t, root)

		// A sentence naming something nowhere in this empty repository --
		// the big route, which runGoJob (this plan) reports and stops on,
		// dispatching no helper.
		sentence := "invent a brand new capability nothing here has ever heard of"
		wantDecision := resolveJobSizeRoute(gatherJobSizeFacts(root, sentence))

		result, err := runGoJob(sentence, 2*time.Second)
		if err != nil {
			t.Fatalf("runGoJob: %v", err)
		}
		if got := stringValue(result["route"]); got != string(wantDecision.Route) {
			t.Fatalf("aether go's own route = %q, want the authority's answer %q", got, wantDecision.Route)
		}
		if got := stringValue(result["route_reason"]); got != wantDecision.Reason {
			t.Fatalf("aether go's own route_reason = %q, want the authority's reason %q", got, wantDecision.Reason)
		}
	})

	t.Run("go_cmd.go spells no route decision of its own", func(t *testing.T) {
		fset := token.NewFileSet()
		parsed, err := parser.ParseFile(fset, "go_cmd.go", nil, 0)
		if err != nil {
			t.Fatalf("parse go_cmd.go: %v", err)
		}

		ast.Inspect(parsed, func(n ast.Node) bool {
			if ident, ok := n.(*ast.Ident); ok && ident.Name == "smallJobFileBudget" {
				t.Errorf("go_cmd.go refers to smallJobFileBudget directly at %v -- the route decision belongs solely to resolveJobSizeRoute", fset.Position(ident.Pos()))
			}
			if comp, ok := n.(*ast.CompositeLit); ok {
				if id, ok := comp.Type.(*ast.Ident); ok && id.Name == "jobSizeDecision" {
					t.Errorf("go_cmd.go constructs a jobSizeDecision literal of its own at %v", fset.Position(comp.Pos()))
				}
			}
			if call, ok := n.(*ast.CallExpr); ok {
				if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == "len" && len(call.Args) == 1 {
					if sel, ok := call.Args[0].(*ast.SelectorExpr); ok && sel.Sel.Name == "MatchedPaths" {
						t.Errorf("go_cmd.go compares len(...MatchedPaths) directly at %v -- route comparisons belong solely to resolveJobSizeRoute", fset.Position(call.Pos()))
					}
				}
			}
			return true
		})
	})
}
