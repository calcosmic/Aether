package cmd

import (
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

// seedZeroStateImprovementPassFixtures seeds an empty-but-valid consolidation
// fixture (matching TestRunPhaseEndConsolidationZeroState's own pattern) so
// runPhaseEndConsolidation's own learning pipeline runs cleanly and reports
// ZeroState, leaving these tests free to assert only on the improvement pass
// half of the summary.
func seedZeroStateImprovementPassFixtures(t *testing.T) {
	t.Helper()
	if err := store.SaveJSON("instincts.json", colony.InstinctsFile{Instincts: []colony.InstinctEntry{}}); err != nil {
		t.Fatalf("seed empty instincts.json: %v", err)
	}
	if err := store.SaveJSON("learning-observations.json", colony.LearningFile{Observations: []colony.Observation{}}); err != nil {
		t.Fatalf("seed empty learning-observations.json: %v", err)
	}
}

// TestAutomaticImprovementPassTracerEndToEnd is the plan's Test 5: a colony
// with one declared, unexpired, beneficial candidate whose scope is exactly
// "project knowledge" runs runPhaseEndConsolidation once and ends with a
// canary run recorded in status running, its checkpoint saved, and an
// improvementPassSummary naming the candidate and the action taken.
func TestAutomaticImprovementPassTracerEndToEnd(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s
	seedZeroStateImprovementPassFixtures(t)

	bank, err := loadFixtureBank()
	if err != nil {
		t.Fatalf("load the real committed fixture bank: %v", err)
	}
	var unguardedWords []string
	unguardedCount := 0
	for _, f := range bank.Fixtures {
		if f.Guard == nil {
			unguardedCount++
			unguardedWords = append(unguardedWords, shadowFixtureSubjectWords(f)...)
		}
	}
	if unguardedCount == 0 {
		t.Fatal("fixture-bank honesty check failed: found zero unguarded fixtures in the real bank -- cannot build a beneficial candidate")
	}

	expires := time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339)
	record, isNew, err := declareShadowCandidate(
		"candidate-tracer",
		string(canaryScopeProjectKnowledge),
		"addresses "+strings.Join(unguardedWords, " "),
		shadowGraderBenignHarms,
		expires,
		"revert the declared change",
	)
	if err != nil {
		t.Fatalf("declare candidate: %v", err)
	}
	if !isNew {
		t.Fatal("expected a new declaration")
	}

	summary := runPhaseEndConsolidation(1)
	if !summary.Ran {
		t.Fatalf("expected consolidation Ran == true, got %+v", summary)
	}

	pass := summary.ImprovementPass
	if !pass.Ran {
		t.Fatalf("expected the improvement pass Ran == true, got %+v", pass)
	}
	if pass.CandidatesConsidered != 1 {
		t.Fatalf("expected 1 candidate considered, got %d (failures: %v)", pass.CandidatesConsidered, pass.Failures)
	}
	if pass.Compared != 1 {
		t.Fatalf("expected 1 comparison run, got %d (failures: %v)", pass.Compared, pass.Failures)
	}
	if pass.Admitted != 1 {
		t.Fatalf("expected the candidate to be admitted, got Admitted=%d Refused=%d failures=%v", pass.Admitted, pass.Refused, pass.Failures)
	}
	if pass.CanariesStarted != 1 {
		t.Fatalf("expected 1 canary started, got %d", pass.CanariesStarted)
	}
	if len(pass.Events) != 1 {
		t.Fatalf("expected exactly 1 event, got %+v", pass.Events)
	}
	if pass.Events[0].Kind != improvementPassEventStarted {
		t.Fatalf("expected a 'started' event, got %+v", pass.Events[0])
	}
	if pass.Events[0].CandidateID != record.ID {
		t.Fatalf("event names candidate %q, want %q", pass.Events[0].CandidateID, record.ID)
	}

	run, found, err := loadCanaryRun(record.ID)
	if err != nil {
		t.Fatalf("load canary run: %v", err)
	}
	if !found {
		t.Fatal("expected a canary run to be recorded for the admitted candidate")
	}
	if run.Status != canaryRunStatusRunning {
		t.Fatalf("expected canary run status running, got %q", run.Status)
	}
	if run.CheckpointID == "" {
		t.Fatal("expected the canary run to carry a saved checkpoint id")
	}
}

// TestImprovementPassClosingLineIsPlainEnglish is the plan's Test 6: the
// tracer fixture's result["improvement_pass"] renders one plain-English
// closing-card line naming what happened, with no candidate identifier, no
// verdict token, and no raw key=value pair in the rendered text. Exercised
// against BOTH dual-typed shapes: the in-process struct and the
// snake_case-keyed map attachConsolidationSummary produces.
func TestImprovementPassClosingLineIsPlainEnglish(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s
	seedZeroStateImprovementPassFixtures(t)

	bank, err := loadFixtureBank()
	if err != nil {
		t.Fatalf("load the real committed fixture bank: %v", err)
	}
	var unguardedWords []string
	for _, f := range bank.Fixtures {
		if f.Guard == nil {
			unguardedWords = append(unguardedWords, shadowFixtureSubjectWords(f)...)
		}
	}
	if len(unguardedWords) == 0 {
		t.Fatal("fixture-bank honesty check failed: found zero unguarded fixtures in the real bank")
	}

	expires := time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339)
	record, _, err := declareShadowCandidate(
		"candidate-closing-line",
		string(canaryScopeProjectKnowledge),
		"addresses "+strings.Join(unguardedWords, " "),
		shadowGraderBenignHarms,
		expires,
		"revert the declared change",
	)
	if err != nil {
		t.Fatalf("declare candidate: %v", err)
	}

	summary := runPhaseEndConsolidation(1)
	pass := summary.ImprovementPass
	if pass.Admitted != 1 {
		t.Fatalf("test setup: expected the candidate to be admitted, got %+v", pass)
	}

	forbidden := []string{
		record.ID,
		"beneficial", "not_beneficial", "overfit", "tied", "inconclusive",
		"project knowledge", "routing",
		"running", "completed", "rolled_back",
		"=",
	}

	assertPlainEnglish := func(t *testing.T, label, rendered string) {
		t.Helper()
		if strings.TrimSpace(rendered) == "" {
			t.Fatalf("%s: expected a non-empty closing-card line", label)
		}
		for _, f := range forbidden {
			if strings.Contains(rendered, f) {
				t.Fatalf("%s: rendered text contains forbidden token %q:\n%s", label, f, rendered)
			}
		}
	}

	t.Run("struct shape (in-process)", func(t *testing.T) {
		rendered := renderImprovementPassBeat(pass)
		assertPlainEnglish(t, "struct shape", rendered)
	})

	t.Run("map shape (attachConsolidationSummary / JSON round-trip)", func(t *testing.T) {
		result := map[string]interface{}{}
		attachConsolidationSummary(result, summary)
		rendered := renderImprovementPassBeat(result["improvement_pass"])
		assertPlainEnglish(t, "map shape", rendered)
	})
}

// TestCheckWithNoDeclaredCandidateCostsNothing is the plan's Test 7: a
// colony with no declared candidate performs no comparison, opens no
// canary, writes no canary or shadow file, and the closing card shows no
// improvement line at all.
func TestCheckWithNoDeclaredCandidateCostsNothing(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s
	seedZeroStateImprovementPassFixtures(t)

	summary := runPhaseEndConsolidation(1)
	if !summary.Ran {
		t.Fatalf("expected consolidation Ran == true, got %+v", summary)
	}
	pass := summary.ImprovementPass
	if !pass.Ran {
		t.Fatal("expected the improvement pass to report Ran == true even with nothing declared")
	}
	if pass.CandidatesConsidered != 0 {
		t.Fatalf("expected zero candidates considered, got %d", pass.CandidatesConsidered)
	}
	if len(pass.Events) != 0 {
		t.Fatalf("expected zero events, got %+v", pass.Events)
	}

	if _, err := os.Stat(filepath.Join(s.BasePath(), "canary", "runs.json")); !os.IsNotExist(err) {
		t.Fatalf("expected no canary/runs.json to be created; stat error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(s.BasePath(), "shadow", "candidates.json")); !os.IsNotExist(err) {
		t.Fatalf("expected no shadow/candidates.json to be created; stat error: %v", err)
	}

	result := map[string]interface{}{}
	attachConsolidationSummary(result, summary)
	if rendered := renderImprovementPassBeat(result["improvement_pass"]); rendered != "" {
		t.Fatalf("expected the closing card to show no improvement line at all, got %q", rendered)
	}
	if rendered := renderImprovementPassBeat(pass); rendered != "" {
		t.Fatalf("expected the closing card to show no improvement line at all (struct shape), got %q", rendered)
	}
}

// TestImprovementPassFailureNeverBlocksTheCheck is the plan's Test 8: with
// the candidate store made unreadable, runPhaseEndConsolidation still
// returns Ran: true and the pass records its own failure reason instead of
// propagating an error.
func TestImprovementPassFailureNeverBlocksTheCheck(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s
	seedZeroStateImprovementPassFixtures(t)

	shadowDir := filepath.Join(s.BasePath(), "shadow")
	if err := os.MkdirAll(shadowDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", shadowDir, err)
	}
	if err := os.WriteFile(filepath.Join(shadowDir, "candidates.json"), []byte("{ this is not valid json"), 0o644); err != nil {
		t.Fatalf("write corrupt candidate store: %v", err)
	}

	summary := runPhaseEndConsolidation(1)
	if !summary.Ran {
		t.Fatalf("expected consolidation Ran == true despite the unreadable candidate store, got %+v", summary)
	}
	pass := summary.ImprovementPass
	if !pass.Ran {
		t.Fatal("expected the improvement pass to report Ran == true despite an unreadable candidate store")
	}
	if len(pass.Failures) == 0 {
		t.Fatal("expected the pass to record its own failure reason for the unreadable candidate store")
	}
}

// TestAutomaticImprovementPassIsReachedFromBothCheckLanes is Task 2's Test
// 1: an AST-based call-graph guard, in the exact shape of
// cmd/application_evidence_test.go's TestPhaseApplicationCreditIsReachedFromBothCheckLanes
// (and reusing its own generic unreached-lane helper), proving
// runAutomaticImprovementPass is transitively reachable from BOTH
// runCodexContinue and runCodexContinueFinalize.
func TestAutomaticImprovementPassIsReachedFromBothCheckLanes(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	g, err := buildCmdFuncGraph(filepath.Join(repoRoot, "cmd"))
	if err != nil {
		t.Fatalf("build call graph: %v", err)
	}
	if g.funcsIndexed == 0 {
		t.Fatalf("found zero top-level functions while scanning %d files -- a guard that finds no functions to check would pass vacuously forever", g.filesScanned)
	}

	const target = "runAutomaticImprovementPass"
	lanes := []string{"runCodexContinue", "runCodexContinueFinalize"}

	if _, ok := g.calls[target]; !ok {
		t.Fatalf("expected %s to be an indexed top-level function among %d indexed functions in cmd/, but it was missing -- renamed or moved?", target, g.funcsIndexed)
	}
	for _, lane := range lanes {
		if _, ok := g.calls[lane]; !ok {
			t.Fatalf("expected %s to be an indexed top-level function among %d indexed functions in cmd/, but it was missing -- renamed or moved?", lane, g.funcsIndexed)
		}
	}

	if unreached := applicationCreditUnreachedLanes(g, lanes, target); len(unreached) > 0 {
		t.Fatalf("%s is not transitively reachable from: %v -- both check lanes must reach the automatic improvement pass", target, unreached)
	}

	t.Run("a synthetic unreachable fixture is caught by name on both lanes", func(t *testing.T) {
		synthetic := &cmdFuncGraph{calls: map[string]map[string]bool{
			"runCodexContinue":            {"someOtherHelper": true},
			"runCodexContinueFinalize":    {"anotherHelper": true},
			"runAutomaticImprovementPass": {},
		}}
		unreached := applicationCreditUnreachedLanes(synthetic, lanes, target)
		if len(unreached) != 2 {
			t.Fatalf("scanner failed to detect the synthetic unreachable fixture on both lanes, got unreached=%v", unreached)
		}
		for _, lane := range lanes {
			found := false
			for _, u := range unreached {
				if u == lane {
					found = true
				}
			}
			if !found {
				t.Fatalf("expected %q named in the unreached set %v", lane, unreached)
			}
		}
	})
}

// TestAutomaticPassNeverPromotesOutsideTheTwoScopes is Task 2's Test 2:
// table-driven over canaryRetainedAuthorityScopeNames() at run time (so a
// tenth retained category added later is covered automatically), asserting
// a candidate declaring that scope is refused, names the authority that
// retains it, and creates no canary run.
func TestAutomaticPassNeverPromotesOutsideTheTwoScopes(t *testing.T) {
	for _, scopeName := range canaryRetainedAuthorityScopeNames() {
		scopeName := scopeName
		t.Run(scopeName, func(t *testing.T) {
			saveGlobals(t)
			s, _ := newTestStore(t)
			store = s

			candidateID := "candidate-retained-" + strings.ReplaceAll(scopeName, " ", "-")
			expires := time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339)
			record, _, err := declareShadowCandidate(candidateID, scopeName, "a generic benefit naming nothing in particular", shadowGraderBenignHarms, expires, "revert the declared change")
			if err != nil {
				t.Fatalf("declare candidate: %v", err)
			}

			pass := runAutomaticImprovementPass(1)
			if len(pass.Events) != 1 {
				t.Fatalf("expected exactly one event, got %+v (failures: %v)", pass.Events, pass.Failures)
			}
			if pass.Events[0].Kind != improvementPassEventRefused {
				t.Fatalf("expected the candidate to be refused, got %+v", pass.Events[0])
			}

			authority, retained := canaryRetainedAuthorityFor(canaryScope(scopeName))
			if !retained {
				t.Fatalf("test setup broken: %q is not in canaryRetainedAuthority", scopeName)
			}
			if !strings.Contains(pass.Events[0].Detail, authority) {
				t.Fatalf("refusal for scope %q does not name its authority %q: %v", scopeName, authority, pass.Events[0].Detail)
			}

			if _, found, err := loadCanaryRun(record.ID); err != nil {
				t.Fatalf("load canary run: %v", err)
			} else if found {
				t.Fatalf("expected no canary run to be created for retained scope %q", scopeName)
			}
		})
	}
}

// TestAutomaticPassCarriesNoBypassParameter is Task 2's Test 3: a
// structural AST assertion, in the shape of
// TestNeitherCoordinatorNorAutopilotCanWaiveARetainedRefusal
// (cmd/promotion_gate_test.go), proving runAutomaticImprovementPass takes
// exactly one parameter (an int phase identifier) and that
// cmd/improvement_pass.go declares no identifier shaped like an actor,
// caller identity, coordinator, autopilot, waiver, or bypass.
func TestAutomaticPassCarriesNoBypassParameter(t *testing.T) {
	src, err := os.ReadFile("improvement_pass.go")
	if err != nil {
		t.Fatalf("read improvement_pass.go: %v", err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "improvement_pass.go", src, 0)
	if err != nil {
		t.Fatalf("parse improvement_pass.go: %v", err)
	}

	var found bool
	var params int
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil || fn.Name.Name != "runAutomaticImprovementPass" {
			continue
		}
		found = true
		for _, field := range fn.Type.Params.List {
			if len(field.Names) == 0 {
				params++
				continue
			}
			params += len(field.Names)
		}
	}
	if !found {
		t.Fatal("could not find runAutomaticImprovementPass in improvement_pass.go")
	}
	if params != 1 {
		t.Fatalf("runAutomaticImprovementPass takes %d parameters, expected exactly 1 (phaseID)", params)
	}

	forbidden := []string{"actor", "caller", "coordinator", "autopilot", "waiver", "bypass"}
	var hits []string
	ast.Inspect(file, func(n ast.Node) bool {
		id, ok := n.(*ast.Ident)
		if !ok {
			return true
		}
		lowerName := strings.ToLower(id.Name)
		for _, f := range forbidden {
			if strings.Contains(lowerName, f) {
				hits = append(hits, id.Name)
			}
		}
		return true
	})
	if len(hits) > 0 {
		t.Fatalf("cmd/improvement_pass.go declares actor/bypass-shaped identifier(s): %v", hits)
	}

	t.Run("the checker is non-vacuous: a synthetic identifier is caught", func(t *testing.T) {
		fixtureSrc := []byte(`package cmd

func improvementPassFixtureViolation(autopilotOverride bool) {
	_ = autopilotOverride
}
`)
		fset := token.NewFileSet()
		fixtureFile, err := parser.ParseFile(fset, "fixture_improvement_pass_violation.go", fixtureSrc, 0)
		if err != nil {
			t.Fatalf("parse fixture: %v", err)
		}
		var fixtureHits []string
		ast.Inspect(fixtureFile, func(n ast.Node) bool {
			id, ok := n.(*ast.Ident)
			if !ok {
				return true
			}
			lowerName := strings.ToLower(id.Name)
			for _, f := range forbidden {
				if strings.Contains(lowerName, f) {
					fixtureHits = append(fixtureHits, id.Name)
				}
			}
			return true
		})
		if len(fixtureHits) == 0 {
			t.Fatal("scanner failed to flag a synthetic autopilot-shaped identifier -- the structural check would be vacuous")
		}
	})
}

// TestUnrecognizedScopeIsRefusedAndWritesNothing is Task 2's Test 4: a
// candidate declaring a scope that is neither promotable nor retained is
// refused and creates nothing.
func TestUnrecognizedScopeIsRefusedAndWritesNothing(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	expires := time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339)
	record, _, err := declareShadowCandidate("candidate-unrecognized-scope", "an entirely unrecognized scope naming nothing declared", "a generic benefit", shadowGraderBenignHarms, expires, "revert the declared change")
	if err != nil {
		t.Fatalf("declare candidate: %v", err)
	}

	pass := runAutomaticImprovementPass(1)
	if len(pass.Events) != 1 {
		t.Fatalf("expected exactly one event, got %+v (failures: %v)", pass.Events, pass.Failures)
	}
	if pass.Events[0].Kind != improvementPassEventRefused {
		t.Fatalf("expected the candidate to be refused, got %+v", pass.Events[0])
	}
	if _, found, err := loadCanaryRun(record.ID); err != nil {
		t.Fatalf("load canary run: %v", err)
	} else if found {
		t.Fatal("expected no canary run to be created for an unrecognized scope")
	}
}
