package cmd

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// validShadowCandidateFields returns a full, valid set of shadow-declare
// inputs so individual tests can start from a known-good baseline.
func validShadowCandidateFields() (scope, expectedBenefit, harms, expires, rollbackPlan string) {
	return "routes builder tasks to a faster model for one caste",
		"shorter turnaround on routine builder tasks",
		"a worse model could be chosen for genuinely hard tasks",
		time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339),
		"revert the routing rule to its previous value"
}

func TestCandidateStoreIsAppendOnlyAndRefusesEdits(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	scope, expectedBenefit, harms, expires, rollbackPlan := validShadowCandidateFields()

	first, isNew, err := declareShadowCandidate("shadow-t1", scope, expectedBenefit, harms, expires, rollbackPlan)
	if err != nil {
		t.Fatalf("unexpected error on first declaration: %v", err)
	}
	if !isNew {
		t.Fatal("expected the first declaration to be new")
	}

	// Replay: identical content under the same id writes nothing and
	// returns the stored record.
	replay, isNew, err := declareShadowCandidate("shadow-t1", scope, expectedBenefit, harms, expires, rollbackPlan)
	if err != nil {
		t.Fatalf("unexpected error on replay declaration: %v", err)
	}
	if isNew {
		t.Fatal("expected a replay declaration (identical content) to report is_new=false")
	}
	if replay.ContentDigest != first.ContentDigest {
		t.Fatalf("replay returned a different content digest: %q vs %q", replay.ContentDigest, first.ContentDigest)
	}

	// Edit attempt: same id, different content is refused by name, not
	// silently applied.
	_, _, err = declareShadowCandidate("shadow-t1", "a completely different scope naming nothing forbidden here", expectedBenefit, harms, expires, rollbackPlan)
	if err == nil {
		t.Fatal("expected a refusal for a second declaration under the same id with different content")
	}
	if !strings.Contains(err.Error(), "shadow-t1") {
		t.Fatalf("refusal %q does not name the offending id", err.Error())
	}

	stored, found, err := loadShadowCandidate("shadow-t1")
	if err != nil {
		t.Fatalf("load candidate: %v", err)
	}
	if !found {
		t.Fatal("expected the originally declared candidate to still be stored")
	}
	if stored.Scope != scope {
		t.Fatalf("the refused edit attempt was applied -- stored scope is %q, want the original %q", stored.Scope, scope)
	}
}

// TestShadowCandidateStoreHasOneWriter is an AST-based scan of the cmd
// package mirroring cmd/recruitment_credit_test.go's TestCreditRequiresBothFacts
// and cmd/episode_ledger_test.go's TestEpisodeLedgerHasOneWriter: it derives
// every function whose body writes shadow/candidates.json via the store,
// and asserts declareShadowCandidate is the only one.
func TestShadowCandidateStoreHasOneWriter(t *testing.T) {
	violations := scanForShadowCandidateWritesOutsideDeclare(t, ".")
	if len(violations) != 0 {
		t.Fatalf("found a shadow-candidate-store write outside declareShadowCandidate:\n%s", strings.Join(violations, "\n"))
	}

	t.Run("a synthetic second writer is caught", func(t *testing.T) {
		fixtureSrc := `package cmd

func sneakilyWriteShadowCandidates(id string) error {
	var file shadowCandidateFile
	return store.UpdateJSONAtomically(shadowCandidateStorePath, &file, func() error {
		file.Entries = append(file.Entries, shadowCandidateRecord{ID: id})
		return nil
	})
}
`
		violations := scanShadowCandidateSourceForViolations(t, "fixture_shadow_candidate_second_writer.go", fixtureSrc)
		if len(violations) == 0 {
			t.Fatal("scanner failed to detect a synthetic second writer of shadow/candidates.json")
		}
		if !strings.Contains(violations[0], "sneakilyWriteShadowCandidates") {
			t.Fatalf("violation %q does not name the offending function", violations[0])
		}
	})
}

func scanForShadowCandidateWritesOutsideDeclare(t *testing.T, dir string) []string {
	t.Helper()
	names, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatalf("glob cmd package files: %v", err)
	}
	if len(names) == 0 {
		t.Fatal("fixture is broken: no .go files found in the cmd package directory")
	}
	fset := token.NewFileSet()
	var violations []string
	found := false
	for _, name := range names {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		fileFound, fileViolations := shadowCandidateWriteViolationsInFile(fset, file)
		found = found || fileFound
		violations = append(violations, fileViolations...)
	}
	if !found {
		t.Fatal("fixture is broken: no write call referencing shadowCandidateStorePath was found anywhere in the cmd package")
	}
	return violations
}

func scanShadowCandidateSourceForViolations(t *testing.T, filename, src string) []string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, 0)
	if err != nil {
		t.Fatalf("parse fixture source: %v", err)
	}
	_, violations := shadowCandidateWriteViolationsInFile(fset, file)
	return violations
}

// shadowCandidateWriteViolationsInFile walks file for every call writing
// shadowCandidateStorePath through the store, and requires the enclosing
// function to be literally named declareShadowCandidate.
func shadowCandidateWriteViolationsInFile(fset *token.FileSet, file *ast.File) (found bool, violations []string) {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "UpdateJSONAtomically" {
				return true
			}
			if len(call.Args) == 0 {
				return true
			}
			ident, ok := call.Args[0].(*ast.Ident)
			if !ok || ident.Name != "shadowCandidateStorePath" {
				return true
			}
			found = true
			if fn.Name.Name != "declareShadowCandidate" {
				violations = append(violations, fmt.Sprintf("%s: func %s writes shadowCandidateStorePath", fset.Position(call.Pos()).Filename, fn.Name.Name))
			}
			return true
		})
	}
	return found, violations
}

func TestComparisonWithoutACandidateIsRefusedByName(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	_, err := runShadowCompare("never-declared")
	if err == nil {
		t.Fatal("expected a refusal comparing a candidate that was never declared")
	}
	if !strings.Contains(err.Error(), "never-declared") {
		t.Fatalf("refusal %q does not name the missing candidate id", err.Error())
	}
}

func TestComparisonResultReachesTheDurableLedger(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	scope, expectedBenefit, harms, expires, rollbackPlan := validShadowCandidateFields()
	if _, _, err := declareShadowCandidate("shadow-t2", scope, expectedBenefit, harms, expires, rollbackPlan); err != nil {
		t.Fatalf("declare candidate: %v", err)
	}

	outcome, err := runShadowCompare("shadow-t2")
	if err != nil {
		t.Fatalf("run comparison: %v", err)
	}
	if !outcome.Credited {
		t.Fatal("expected the first comparison run to be credited")
	}

	records, err := readEpisodeLedger()
	if err != nil {
		t.Fatalf("read episode ledger: %v", err)
	}
	episodeID := shadowComparisonEpisodeID("shadow-t2")
	terminal, ok := episodeLedgerTerminalRecord(records, episodeID)
	if !ok {
		t.Fatalf("no terminal record found in the episode ledger for episode %q", episodeID)
	}
	if terminal.TerminalResult != string(outcome.Comparison.Verdict) {
		t.Fatalf("ledger terminal result %q does not match the comparison verdict %q", terminal.TerminalResult, outcome.Comparison.Verdict)
	}
	if terminal.EvaluatorDigest == "" {
		t.Fatal("ledger terminal record carries no evaluator digest")
	}
	if terminal.PolicyVersion == "" {
		t.Fatal("ledger terminal record carries no policy version (baseline digest)")
	}
	if len(terminal.EvidenceIDs) != 4 {
		t.Fatalf("expected 4 evidence entries (one per score), got %d: %v", len(terminal.EvidenceIDs), terminal.EvidenceIDs)
	}
}

func TestComparisonReplayWritesNothing(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	store = s

	scope, expectedBenefit, harms, expires, rollbackPlan := validShadowCandidateFields()
	if _, _, err := declareShadowCandidate("shadow-t3", scope, expectedBenefit, harms, expires, rollbackPlan); err != nil {
		t.Fatalf("declare candidate: %v", err)
	}

	first, err := runShadowCompare("shadow-t3")
	if err != nil {
		t.Fatalf("first comparison run: %v", err)
	}
	if !first.Credited {
		t.Fatal("expected the first comparison run to be credited")
	}

	ledgerPath := filepath.Join(tmpDir, ".aether", "data", episodeLedgerPath)
	before, err := os.ReadFile(ledgerPath)
	if err != nil {
		t.Fatalf("read ledger file after first run: %v", err)
	}

	second, err := runShadowCompare("shadow-t3")
	if err != nil {
		t.Fatalf("second (replay) comparison run: %v", err)
	}
	if second.Credited {
		t.Fatal("expected the replayed comparison run to report credited=false")
	}
	if second.Comparison != first.Comparison {
		t.Fatalf("replay produced a different comparison result:\n  %+v\n  %+v", first.Comparison, second.Comparison)
	}

	after, err := os.ReadFile(ledgerPath)
	if err != nil {
		t.Fatalf("read ledger file after replay run: %v", err)
	}
	if string(before) != string(after) {
		t.Fatalf("the replayed comparison run changed the ledger file's bytes")
	}
}

// countResolveEvalGateHoldoutsCalls counts every call expression in file
// whose called function is named resolveEvalGateHoldouts, qualified or
// not.
func countResolveEvalGateHoldoutsCalls(file *ast.File) int {
	count := 0
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch fn := call.Fun.(type) {
		case *ast.Ident:
			if fn.Name == "resolveEvalGateHoldouts" {
				count++
			}
		case *ast.SelectorExpr:
			if fn.Sel.Name == "resolveEvalGateHoldouts" {
				count++
			}
		}
		return true
	})
	return count
}

// TestHoldoutResolutionHappensOnlyInTheCommandLayer is an AST scan proving
// resolveEvalGateHoldouts is called exactly once in cmd/shadow_cmds.go, and
// never at all anywhere in pkg/shadow -- the holdout resolution happens
// here and only here, so a candidate declaration sitting inside pkg/shadow
// can reach nothing that names a holdout.
func TestHoldoutResolutionHappensOnlyInTheCommandLayer(t *testing.T) {
	fset := token.NewFileSet()
	cmdFile, err := parser.ParseFile(fset, "shadow_cmds.go", nil, 0)
	if err != nil {
		t.Fatalf("parse cmd/shadow_cmds.go: %v", err)
	}
	got := countResolveEvalGateHoldoutsCalls(cmdFile)
	if got != 1 {
		t.Fatalf("expected exactly one call to resolveEvalGateHoldouts in cmd/shadow_cmds.go, found %d", got)
	}

	shadowDir := filepath.Join("..", "pkg", "shadow")
	entries, err := os.ReadDir(shadowDir)
	if err != nil {
		t.Fatalf("read pkg/shadow dir: %v", err)
	}
	scanned := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		path := filepath.Join(shadowDir, e.Name())
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		scanned++
		if n := countResolveEvalGateHoldoutsCalls(file); n != 0 {
			t.Fatalf("pkg/shadow file %s calls resolveEvalGateHoldouts %d time(s) -- holdout resolution must happen only in the command layer", e.Name(), n)
		}
	}
	if scanned == 0 {
		t.Fatal("fixture is broken: no .go files found in pkg/shadow")
	}
}

func TestComparisonResultSpeaksPlainEnglish(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	scope, expectedBenefit, harms, expires, rollbackPlan := validShadowCandidateFields()
	if _, _, err := declareShadowCandidate("shadow-t4", scope, expectedBenefit, harms, expires, rollbackPlan); err != nil {
		t.Fatalf("declare candidate: %v", err)
	}
	outcome, err := runShadowCompare("shadow-t4")
	if err != nil {
		t.Fatalf("run comparison: %v", err)
	}

	// bannedJargon names repo/package-internal terms that must never appear
	// untranslated in the owner-facing rendered result -- the owner reads
	// "the proposal" / "the current behaviour", never "candidate" /
	// "baseline" / "evaluator" / "holdout".
	bannedJargon := []string{"candidate", "baseline", "evaluator", "holdout", "FrozenEvaluator", "shadow_comparison"}

	for _, line := range strings.Split(strings.TrimRight(outcome.Rendered, "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if underscoreTokenPattern.MatchString(line) {
			t.Fatalf("line %q carries a raw internal token", line)
		}
		hasGlyph := false
		for _, glyph := range voiceGlyphMap {
			if strings.HasPrefix(line, glyph) {
				hasGlyph = true
				break
			}
		}
		if !hasGlyph {
			t.Fatalf("line %q does not open with a shared-table glyph", line)
		}
		lower := strings.ToLower(line)
		for _, jargon := range bannedJargon {
			if strings.Contains(lower, strings.ToLower(jargon)) {
				t.Fatalf("line %q uses untranslated internal term %q", line, jargon)
			}
		}
	}

	if !strings.Contains(outcome.Rendered, fmt.Sprintf("%d out of %d", outcome.Comparison.VisibleCandidate.Numerator, outcome.Comparison.VisibleCandidate.Denominator)) {
		t.Fatal("rendered result does not show the visible score as a count over a total")
	}
	if !strings.Contains(outcome.Rendered, fmt.Sprintf("%d out of %d", outcome.Comparison.HoldoutCandidate.Numerator, outcome.Comparison.HoldoutCandidate.Denominator)) {
		t.Fatal("rendered result does not show the holdout score as a count over a total")
	}
}
