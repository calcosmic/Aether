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

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/learn"
	"github.com/calcosmic/Aether/pkg/storage"
)

// ---------------------------------------------------------------------------
// Shared fixtures.
// ---------------------------------------------------------------------------

// seedLearnEntryWithStatus writes one learning entry through the real store
// writer (learn.NewColonyStore(store).Add), with a genuine status, never a
// hand-typed entries.json literal.
func seedLearnEntryWithStatus(t *testing.T, s *storage.Store, content string, phase int, status string) {
	t.Helper()
	entry := learn.Entry{
		Content:        content,
		Phase:          phase,
		Confidence:     0.8,
		Classification: learn.ClassRepoLocal,
		Status:         status,
		Evidence: learn.Evidence{
			Phase:       phase,
			Timestamp:   time.Now().UTC().Format(time.RFC3339),
			Confidence:  0.8,
			GatesPassed: 1,
			GatesTotal:  1,
		},
	}
	if err := learn.NewColonyStore(s).Add(entry); err != nil {
		t.Fatalf("seed learn entry %q: %v", content, err)
	}
}

// minimalColonyPrimeState seeds the minimum COLONY_STATE.json colony-prime
// needs to render a capsule at all.
func minimalColonyPrimeState(t *testing.T, s *storage.Store) {
	t.Helper()
	goal := "exercise the learning status vocabulary"
	state := colony.ColonyState{
		Version: "1.0", Goal: &goal, State: colony.StateREADY, CurrentPhase: 1,
		Plan: colony.Plan{Phases: []colony.Phase{{ID: 1, Name: "Test", Status: colony.PhaseReady}}},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save state: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Task 2 (204-02-PLAN.md): one status vocabulary, applied at both render
// sites.
// ---------------------------------------------------------------------------

func TestHypothesisIsNeverRenderedAsVerified(t *testing.T) {
	t.Run("all hypothesis", func(t *testing.T) {
		saveGlobals(t)
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s
		minimalColonyPrimeState(t, s)
		seedLearnEntryWithStatus(t, s, "sentinel hypothesis about cmd/foo.go never checked yet", 1, learn.StatusHypothesis)

		output := buildColonyPrimeOutput(false)
		if strings.Contains(output.Context, learnedMemoryVerifiedHeading) {
			t.Fatalf("an all-hypothesis colony rendered the verified heading:\n%s", output.Context)
		}
	})

	t.Run("mixed validated and hypothesis", func(t *testing.T) {
		saveGlobals(t)
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s
		minimalColonyPrimeState(t, s)
		const validatedContent = "sentinel validated finding about cmd/validated_marker.go"
		const hypothesisContent = "sentinel hypothesis finding about cmd/hypothesis_marker.go"
		seedLearnEntryWithStatus(t, s, validatedContent, 1, learn.StatusValidated)
		seedLearnEntryWithStatus(t, s, hypothesisContent, 1, learn.StatusHypothesis)

		output := buildColonyPrimeOutput(false)
		vStart, vEnd := sectionBoundsInContext(t, output.Context, learnedMemoryVerifiedHeading)
		uStart, uEnd := sectionBoundsInContext(t, output.Context, learnedMemoryUnverifiedHeading)

		validatedIdx := strings.Index(output.Context, validatedContent)
		hypothesisIdx := strings.Index(output.Context, hypothesisContent)
		if validatedIdx < 0 || hypothesisIdx < 0 {
			t.Fatalf("expected both entries' content to appear somewhere in the capsule, got:\n%s", output.Context)
		}
		if !(validatedIdx >= vStart && validatedIdx < vEnd) {
			t.Fatalf("validated entry (status-derived placement) did not land inside the verified block [%d,%d): idx=%d\n%s", vStart, vEnd, validatedIdx, output.Context)
		}
		if validatedIdx >= uStart && validatedIdx < uEnd {
			t.Fatalf("validated entry incorrectly ALSO appears inside the unverified block [%d,%d): idx=%d\n%s", uStart, uEnd, validatedIdx, output.Context)
		}
		if !(hypothesisIdx >= uStart && hypothesisIdx < uEnd) {
			t.Fatalf("hypothesis entry (status-derived placement) did not land inside the unverified block [%d,%d): idx=%d\n%s", uStart, uEnd, hypothesisIdx, output.Context)
		}
		if hypothesisIdx >= vStart && hypothesisIdx < vEnd {
			t.Fatalf("hypothesis entry incorrectly appears inside the verified block [%d,%d): idx=%d\n%s", vStart, vEnd, hypothesisIdx, output.Context)
		}
	})

	t.Run("disproven", func(t *testing.T) {
		saveGlobals(t)
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s
		minimalColonyPrimeState(t, s)
		const disprovenContent = "sentinel disproven finding about cmd/disproven_marker.go"
		seedLearnEntryWithStatus(t, s, disprovenContent, 1, learn.StatusDisproven)

		output := buildColonyPrimeOutput(false)
		if strings.Contains(output.Context, disprovenContent) {
			t.Fatalf("a disproven entry appeared somewhere in the capsule, want neither heading:\n%s", output.Context)
		}
	})
}

// TestUnverifiedEntriesAreStillShown proves the all-hypothesis colony's
// capsule still carries those entries' own content, under the unverified
// heading -- nothing is silently dropped.
func TestUnverifiedEntriesAreStillShown(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	minimalColonyPrimeState(t, s)
	const hypothesisContent = "sentinel unchecked observation about cmd/still_shown_marker.go"
	seedLearnEntryWithStatus(t, s, hypothesisContent, 1, learn.StatusHypothesis)

	output := buildColonyPrimeOutput(false)
	uStart, uEnd := sectionBoundsInContext(t, output.Context, learnedMemoryUnverifiedHeading)
	idx := strings.Index(output.Context, hypothesisContent)
	if idx < 0 {
		t.Fatalf("expected the hypothesis entry's own content to still appear, got:\n%s", output.Context)
	}
	if !(idx >= uStart && idx < uEnd) {
		t.Fatalf("hypothesis content did not land inside the unverified block [%d,%d): idx=%d\n%s", uStart, uEnd, idx, output.Context)
	}
}

// TestAutopilotLessonsRequireValidatedStatus drives confirmedAutopilotLessonsSincePlan
// with entries identical in every respect except status, and asserts the
// validated one is admitted and the hypothesis one is not.
func TestAutopilotLessonsRequireValidatedStatus(t *testing.T) {
	boundary := time.Date(2026, time.September, 1, 12, 0, 0, 0, time.UTC)
	plan := autopilotLessonPlan(boundary, "revision-status-gate")

	validated := autopilotLessonFixture(boundary)
	validated.ID = "lesson-validated"
	validated.Status = learn.StatusValidated

	hypothesis := autopilotLessonFixture(boundary)
	hypothesis.ID = "lesson-hypothesis"
	hypothesis.Content = "Keep provider readiness retries inside the provider layer, unverified."
	hypothesis.Status = learn.StatusHypothesis

	lessons, err := confirmedAutopilotLessonsSincePlan(plan, []learn.Entry{validated, hypothesis})
	if err != nil {
		t.Fatalf("select confirmed lessons: %v", err)
	}
	if len(lessons) != 1 {
		t.Fatalf("expected exactly 1 admitted lesson (the validated one), got %d: %+v", len(lessons), lessons)
	}
	if lessons[0].EntryID != validated.ID {
		t.Fatalf("admitted lesson = %q, want the validated entry %q", lessons[0].EntryID, validated.ID)
	}
}

// ---------------------------------------------------------------------------
// TestOneLearningStatusVocabulary: an AST scan over every non-test file in
// cmd/ that fails, naming the offending file and function, if any function
// other than the two helpers in cmd/learning_status_vocabulary.go compares
// against the learning status vocabulary (learn.Status*).
// ---------------------------------------------------------------------------

// learningStatusPredicateAllowedFunctions is the ONLY set of symbol names
// permitted to compare against a learn.Status* constant -- permitted by
// symbol name, never by file path, following TestCreditRequiresBothFacts's
// own convention (cmd/recruitment_credit_test.go).
var learningStatusPredicateAllowedFunctions = map[string]bool{
	"learningVerifiedEntries":   true,
	"learningUnverifiedEntries": true,
}

func TestOneLearningStatusVocabulary(t *testing.T) {
	violations := scanForSecondLearningStatusPredicate(t, ".")
	if len(violations) != 0 {
		t.Fatalf("found a comparison against the learning status vocabulary outside the shared helpers:\n%s", strings.Join(violations, "\n"))
	}

	t.Run("a synthetic second predicate is caught by name", func(t *testing.T) {
		fixtureSrc := `package cmd

import (
	"github.com/calcosmic/Aether/pkg/learn"
)

func rogueLearningStatusCheck(entry learn.Entry) bool {
	return entry.Status == learn.StatusValidated
}
`
		violations := scanSourceForSecondLearningStatusPredicate(t, "fixture_second_status_predicate.go", fixtureSrc)
		if len(violations) == 0 {
			t.Fatal("scanner failed to detect a synthetic second status predicate")
		}
		found := false
		for _, v := range violations {
			if strings.Contains(v, "rogueLearningStatusCheck") && strings.Contains(v, "fixture_second_status_predicate.go") {
				found = true
			}
		}
		if !found {
			t.Fatalf("violation message does not name the offending file and function: %v", violations)
		}
	})
}

func scanForSecondLearningStatusPredicate(t *testing.T, dir string) []string {
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
		fileFound, fileViolations := learningStatusPredicateViolationsInFile(fset, file)
		found = found || fileFound
		violations = append(violations, fileViolations...)
	}
	if !found {
		t.Fatal("fixture is broken: no comparison against the learning status vocabulary was found anywhere in the cmd package")
	}
	return violations
}

func scanSourceForSecondLearningStatusPredicate(t *testing.T, filename, src string) []string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, 0)
	if err != nil {
		t.Fatalf("parse fixture source: %v", err)
	}
	_, violations := learningStatusPredicateViolationsInFile(fset, file)
	return violations
}

// learningStatusPredicateViolationsInFile walks every top-level function in
// file for a comparison against a learn.Status* constant -- either a direct
// `== ` / `!=` comparison, or a strings.EqualFold/strings.Compare call
// naming one as an argument (learningVerifiedEntries's own shape) -- and
// requires the enclosing function to be in learningStatusPredicateAllowedFunctions.
// A plain WRITE of a status value (a struct literal field, or `entry.Status
// = learn.StatusHypothesis`, e.g. cmd/codex_continue_finalize.go's fresh-
// hypothesis write) is deliberately NOT flagged -- only a comparison is the
// defect class ruling (b) addresses.
func learningStatusPredicateViolationsInFile(fset *token.FileSet, file *ast.File) (found bool, violations []string) {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		fnName := "<unknown>"
		if fn.Name != nil {
			fnName = fn.Name.Name
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			if !learningStatusComparisonFound(n) {
				return true
			}
			found = true
			if !learningStatusPredicateAllowedFunctions[fnName] {
				violations = append(violations, fmt.Sprintf(
					"%s: %s compares against the learning status vocabulary -- only learningVerifiedEntries/learningUnverifiedEntries (cmd/learning_status_vocabulary.go) may do this",
					fset.Position(n.Pos()).String(), fnName,
				))
			}
			return true
		})
	}
	return found, violations
}

// learningStatusComparisonFound reports whether n is a comparison against a
// learn.Status* constant: a BinaryExpr using == or !=, or ANY call passing
// one as an argument -- covering both a direct `status == learn.StatusX`
// shape (the pre-fix cmd/autopilot_lessons.go idiom) and a call-based
// comparison like learningEntryStatusEquals(entry.Status, learn.StatusValidated)
// (learningVerifiedEntries's own shape, cmd/learning_status_vocabulary.go).
// A plain WRITE of a status value (a struct/map literal field, or `entry.Status
// = learn.StatusHypothesis`) is neither a BinaryExpr nor a CallExpr argument
// naming the constant, so it is correctly never flagged.
func learningStatusComparisonFound(n ast.Node) bool {
	switch v := n.(type) {
	case *ast.BinaryExpr:
		if v.Op != token.EQL && v.Op != token.NEQ {
			return false
		}
		return referencesLearnStatusConstant(v.X) || referencesLearnStatusConstant(v.Y)
	case *ast.CallExpr:
		for _, arg := range v.Args {
			if referencesLearnStatusConstant(arg) {
				return true
			}
		}
	}
	return false
}

// referencesLearnStatusConstant reports whether e is a selector expression
// of the shape learn.Status<Something>.
func referencesLearnStatusConstant(e ast.Expr) bool {
	sel, ok := e.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkgIdent, ok := sel.X.(*ast.Ident)
	return ok && pkgIdent.Name == "learn" && strings.HasPrefix(sel.Sel.Name, "Status")
}
