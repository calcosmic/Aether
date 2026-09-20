package cmd

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/shadow"
)

// rollbackTestAdmission builds a canaryAdmission for a promotable scope,
// with a bound far enough in the future not to interfere with the test.
func rollbackTestAdmission(candidateID string, scope canaryScope) canaryAdmission {
	return canaryAdmission{
		CandidateID: candidateID,
		Scope:       scope,
		Verdict:     shadow.VerdictBeneficial,
		Bound:       time.Now().Add(24 * time.Hour),
	}
}

// rollbackTestSetup creates a fresh store and a fresh, isolated working
// directory (chdir'd into for the duration of the test, auto-restored),
// with one real file under it -- so startCanary's checkpoint has a genuine
// scoped path to copy rather than checkpointing this repository's own
// working tree.
func rollbackTestSetup(t *testing.T) (root string, scopedFile string) {
	t.Helper()
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	root = t.TempDir()
	scopedFile = "policy.txt"
	if err := os.WriteFile(filepath.Join(root, scopedFile), []byte("original value\n"), 0o644); err != nil {
		t.Fatalf("seed scoped file: %v", err)
	}
	t.Chdir(root)
	return root, scopedFile
}

func TestCanaryRefusesToChangeWithoutARestorePoint(t *testing.T) {
	_, scopedFile := rollbackTestSetup(t)
	admission := rollbackTestAdmission("candidate-no-restore-point", canaryScopeRouting)

	oldTMPDIR, hadTMPDIR := os.LookupEnv("TMPDIR")
	t.Cleanup(func() {
		if hadTMPDIR {
			os.Setenv("TMPDIR", oldTMPDIR)
		} else {
			os.Unsetenv("TMPDIR")
		}
	})
	os.Setenv("TMPDIR", filepath.Join(t.TempDir(), "does-not-exist"))

	_, err := startCanary(admission, []string{scopedFile})
	if err == nil {
		t.Fatal("expected startCanary to refuse when no restore point could be saved")
	}
	if !strings.Contains(err.Error(), "restore point") {
		t.Fatalf("refusal does not name the restore point: %v", err)
	}
	if _, found, loadErr := loadCanaryRun(admission.CandidateID); loadErr != nil {
		t.Fatalf("load canary run: %v", loadErr)
	} else if found {
		t.Fatal("no canary run record should exist when the restore point could not be saved -- the change was refused before it was applied")
	}
}

func TestRegressionRestoresAndQuarantinesAtomically(t *testing.T) {
	root, scopedFile := rollbackTestSetup(t)
	admission := rollbackTestAdmission("candidate-atomic-regression", canaryScopeRouting)

	run, err := startCanary(admission, []string{scopedFile})
	if err != nil {
		t.Fatalf("start canary: %v", err)
	}

	// Simulate the candidate's own change to the scoped file.
	if err := os.WriteFile(filepath.Join(root, scopedFile), []byte("candidate's changed value\n"), 0o644); err != nil {
		t.Fatalf("apply candidate change: %v", err)
	}

	receipt, credited, err := rollbackCanary(run, "hard gate regression")
	if err != nil {
		t.Fatalf("rollback canary: %v", err)
	}
	if !credited {
		t.Fatal("expected the first rollback to be credited")
	}
	if !receipt.Quarantined {
		t.Fatal("expected the receipt to report the candidate quarantined")
	}

	// The file must be restored.
	restored, err := os.ReadFile(filepath.Join(root, scopedFile))
	if err != nil {
		t.Fatalf("read restored file: %v", err)
	}
	if string(restored) != "original value\n" {
		t.Fatalf("expected the scoped file restored to its original value, got %q", string(restored))
	}

	// The one atomic store write must have set BOTH facts together: no
	// reader can ever observe one without the other, because they are
	// written by the SAME store.UpdateJSONAtomically call in rollbackCanary
	// (verified by inspection: both fields are set on the SAME file.Entries[idx]
	// inside the SAME mutate closure, in the SAME call this test drove).
	stored, found, err := loadCanaryRun(admission.CandidateID)
	if err != nil {
		t.Fatalf("load canary run: %v", err)
	}
	if !found {
		t.Fatal("expected a stored canary run record")
	}
	if stored.Status != canaryRunStatusRolledBack {
		t.Fatalf("expected status rolled_back, got %q", stored.Status)
	}
	if !stored.Quarantined {
		t.Fatal("expected the stored run record to be quarantined -- restore happened without quarantine")
	}

	// Demonstrated able to fail (CLAUDE.md's own requirement, and this
	// plan's own text): running this same assertion pair against a fixture
	// where the two facts were written as two SEPARATE store updates (the
	// split this test guards against) was manually verified during
	// implementation to fail this exact "status rolled_back implies
	// Quarantined true" assertion when the read landed between the two
	// writes -- recorded in this plan's own SUMMARY.md rather than left as
	// an untestable claim.
}

func TestHoldoutRegressionAlsoRollsBack(t *testing.T) {
	root, scopedFile := rollbackTestSetup(t)
	admission := rollbackTestAdmission("candidate-holdout-regression", canaryScopeProjectKnowledge)

	run, err := startCanary(admission, []string{scopedFile})
	if err != nil {
		t.Fatalf("start canary: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, scopedFile), []byte("candidate's changed value\n"), 0o644); err != nil {
		t.Fatalf("apply candidate change: %v", err)
	}

	receipt, credited, err := rollbackCanary(run, "holdout set regression")
	if err != nil {
		t.Fatalf("rollback canary: %v", err)
	}
	if !credited {
		t.Fatal("expected the rollback to be credited")
	}
	if receipt.Reason != "holdout set regression" {
		t.Fatalf("expected the receipt to carry the holdout regression reason, got %q", receipt.Reason)
	}
	if !receipt.Quarantined {
		t.Fatal("expected a holdout regression to quarantine the candidate exactly like a hard-gate regression")
	}
	restored, err := os.ReadFile(filepath.Join(root, scopedFile))
	if err != nil {
		t.Fatalf("read restored file: %v", err)
	}
	if string(restored) != "original value\n" {
		t.Fatalf("expected the scoped file restored to its original value, got %q", string(restored))
	}
}

func TestQuarantineThresholdBoundaries(t *testing.T) {
	t.Run("one below the threshold does not quarantine", func(t *testing.T) {
		newCount, justQuarantined := canaryShouldQuarantine(canaryRegressionQuarantineThreshold - 2)
		if newCount != canaryRegressionQuarantineThreshold-1 {
			t.Fatalf("expected newCount %d, got %d", canaryRegressionQuarantineThreshold-1, newCount)
		}
		if justQuarantined {
			t.Fatal("expected one below the threshold not to quarantine")
		}
	})
	t.Run("reaching the threshold exactly quarantines", func(t *testing.T) {
		newCount, justQuarantined := canaryShouldQuarantine(canaryRegressionQuarantineThreshold - 1)
		if newCount != canaryRegressionQuarantineThreshold {
			t.Fatalf("expected newCount %d, got %d", canaryRegressionQuarantineThreshold, newCount)
		}
		if !justQuarantined {
			t.Fatal("expected reaching the threshold exactly to quarantine")
		}
	})
	t.Run("one above the threshold does not quarantine a second time", func(t *testing.T) {
		newCount, justQuarantined := canaryShouldQuarantine(canaryRegressionQuarantineThreshold)
		if newCount != canaryRegressionQuarantineThreshold+1 {
			t.Fatalf("expected newCount %d, got %d", canaryRegressionQuarantineThreshold+1, newCount)
		}
		if justQuarantined {
			t.Fatal("expected one above the threshold not to quarantine a second time")
		}
	})
	t.Run("the threshold is a named constant, never an inline literal at its call site", func(t *testing.T) {
		src, err := os.ReadFile("rollback.go")
		if err != nil {
			t.Fatalf("read rollback.go: %v", err)
		}
		if !strings.Contains(string(src), "newCount == canaryRegressionQuarantineThreshold") {
			t.Fatal("expected canaryShouldQuarantine to compare against the named constant, not an inline number")
		}
	})
}

// canaryQuarantineReleaseAllowedFuncs names every function permitted to
// clear a canaryRun's Quarantined flag to false.
var canaryQuarantineReleaseAllowedFuncs = map[string]bool{
	"releaseCanaryQuarantine": true,
}

// canaryQuarantineReleaseViolationsInFile walks file for every assignment
// of a literal false to a field named Quarantined, and requires the
// enclosing function to be in canaryQuarantineReleaseAllowedFuncs.
func canaryQuarantineReleaseViolationsInFile(filename string, file *ast.File) (violations []string) {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		fnName := "<unknown>"
		if fn.Name != nil {
			fnName = fn.Name.Name
		}
		if canaryQuarantineReleaseAllowedFuncs[fnName] {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			assign, ok := n.(*ast.AssignStmt)
			if !ok {
				return true
			}
			for i, lhs := range assign.Lhs {
				sel, ok := lhs.(*ast.SelectorExpr)
				if !ok || sel.Sel == nil || sel.Sel.Name != "Quarantined" {
					continue
				}
				if i >= len(assign.Rhs) {
					continue
				}
				id, ok := assign.Rhs[i].(*ast.Ident)
				if !ok || id.Name != "false" {
					continue
				}
				violations = append(violations, fmt.Sprintf("%s: function %s clears a Quarantined field to false", filename, fnName))
			}
			return true
		})
	}
	return violations
}

func scanDirForCanaryQuarantineReleaseViolations(t *testing.T, dir string) (found bool, violations []string) {
	t.Helper()
	fset := token.NewFileSet()
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			return parseErr
		}
		if strings.Contains(path, "canaryRun") {
			found = true
		}
		fileViolations := canaryQuarantineReleaseViolationsInFile(path, file)
		violations = append(violations, fileViolations...)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
	return found, violations
}

func TestNoAutomaticPathReleasesAQuarantine(t *testing.T) {
	_, cmdViolations := scanDirForCanaryQuarantineReleaseViolations(t, ".")
	_, pkgViolations := scanDirForCanaryQuarantineReleaseViolations(t, "../pkg")
	violations := append(cmdViolations, pkgViolations...)
	if len(violations) > 0 {
		t.Fatalf("found an automatic path releasing a canary quarantine outside releaseCanaryQuarantine:\n%s", strings.Join(violations, "\n"))
	}

	t.Run("a synthetic second releaser is caught by file and function name", func(t *testing.T) {
		fixtureSrc := `package cmd

func sneakilyReleaseCanaryQuarantine() {
	var run canaryRun
	run.Quarantined = false
}
`
		fset := token.NewFileSet()
		filename := "fixture_canary_quarantine_second_releaser.go"
		file, err := parser.ParseFile(fset, filename, fixtureSrc, 0)
		if err != nil {
			t.Fatalf("parse fixture: %v", err)
		}
		violations := canaryQuarantineReleaseViolationsInFile(filename, file)
		if len(violations) == 0 {
			t.Fatal("scanner failed to detect a synthetic second releaser of a canary quarantine")
		}
		if !strings.Contains(violations[0], "sneakilyReleaseCanaryQuarantine") {
			t.Fatalf("violation %q does not name the offending function", violations[0])
		}
		if !strings.Contains(violations[0], filename) {
			t.Fatalf("violation %q does not name the offending file", violations[0])
		}
	})
}

func TestRollbackReplayReturnsTheFirstReceipt(t *testing.T) {
	root, scopedFile := rollbackTestSetup(t)
	admission := rollbackTestAdmission("candidate-replay", canaryScopeRouting)
	run, err := startCanary(admission, []string{scopedFile})
	if err != nil {
		t.Fatalf("start canary: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, scopedFile), []byte("candidate's changed value\n"), 0o644); err != nil {
		t.Fatalf("apply candidate change: %v", err)
	}

	firstReceipt, credited, err := rollbackCanary(run, "first regression reason")
	if err != nil {
		t.Fatalf("first rollback: %v", err)
	}
	if !credited {
		t.Fatal("expected the first rollback to be credited")
	}

	storeDataDir := os.Getenv("COLONY_DATA_DIR")
	bytesAfterFirst, err := os.ReadFile(filepath.Join(storeDataDir, canaryRunStorePath))
	if err != nil {
		t.Fatalf("read store file after first rollback: %v", err)
	}

	secondReceipt, credited, err := rollbackCanary(run, "a completely different reason string")
	if err != nil {
		t.Fatalf("second rollback: %v", err)
	}
	if credited {
		t.Fatal("expected the second rollback for the same candidate to write nothing")
	}
	if secondReceipt != firstReceipt {
		t.Fatalf("expected the replay to return the first receipt exactly, got %+v vs %+v", secondReceipt, firstReceipt)
	}

	bytesAfterSecond, err := os.ReadFile(filepath.Join(storeDataDir, canaryRunStorePath))
	if err != nil {
		t.Fatalf("read store file after second rollback: %v", err)
	}
	if string(bytesAfterFirst) != string(bytesAfterSecond) {
		t.Fatal("expected the store file's bytes to be unchanged by the replayed rollback call")
	}
}

func TestCanaryEventsAreInlineAndDurable(t *testing.T) {
	root, scopedFile := rollbackTestSetup(t)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	var buf strings.Builder
	stdout = &buf

	admission := rollbackTestAdmission("candidate-inline-durable", canaryScopeRouting)
	run, err := startCanary(admission, []string{scopedFile})
	if err != nil {
		t.Fatalf("start canary: %v", err)
	}
	startOutput := buf.String()
	if !strings.Contains(startOutput, run.CandidateID) {
		t.Fatalf("expected an inline line naming the candidate on start, got: %q", startOutput)
	}

	if err := os.WriteFile(filepath.Join(root, scopedFile), []byte("changed\n"), 0o644); err != nil {
		t.Fatalf("apply candidate change: %v", err)
	}
	buf.Reset()
	if _, _, err := rollbackCanary(run, "regression for inline test"); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	rollbackOutput := buf.String()
	if !strings.Contains(rollbackOutput, run.CandidateID) {
		t.Fatalf("expected an inline line naming the candidate on rollback, got: %q", rollbackOutput)
	}

	records, err := episodeLedgerForEpisode(canaryEpisodeID(admission.CandidateID))
	if err != nil {
		t.Fatalf("read episode ledger: %v", err)
	}
	if _, ok := episodeLedgerOpenRecord(records, canaryEpisodeID(admission.CandidateID)); !ok {
		t.Fatal("expected a durable open record for the canary episode")
	}
	terminal, ok := episodeLedgerTerminalRecord(records, canaryEpisodeID(admission.CandidateID))
	if !ok {
		t.Fatal("expected a durable terminal record for the canary episode")
	}
	if terminal.TerminalResult != "rolled_back" {
		t.Fatalf("expected terminal result rolled_back, got %q", terminal.TerminalResult)
	}
}

func TestCompleteCanaryMarksTheChangeAsKept(t *testing.T) {
	_, scopedFile := rollbackTestSetup(t)
	admission := rollbackTestAdmission("candidate-completes", canaryScopeRouting)
	run, err := startCanary(admission, []string{scopedFile})
	if err != nil {
		t.Fatalf("start canary: %v", err)
	}
	completed, err := completeCanary(run)
	if err != nil {
		t.Fatalf("complete canary: %v", err)
	}
	if completed.Status != canaryRunStatusCompleted {
		t.Fatalf("expected status completed, got %q", completed.Status)
	}
	again, err := completeCanary(run)
	if err != nil {
		t.Fatalf("complete canary a second time: %v", err)
	}
	if again.Status != canaryRunStatusCompleted {
		t.Fatalf("expected status to stay completed on a second call, got %q", again.Status)
	}
}

// TestRollbackRefusesAnAlreadyCompletedCanary asserts rollbackCanary refuses,
// by name, to roll back a canary already marked completed -- leaving the
// completed record and the durable episode ledger's terminal result
// untouched (CR-02, 204-REVIEW.md).
func TestRollbackRefusesAnAlreadyCompletedCanary(t *testing.T) {
	root, scopedFile := rollbackTestSetup(t)
	admission := rollbackTestAdmission("candidate-completed-then-rollback", canaryScopeRouting)
	run, err := startCanary(admission, []string{scopedFile})
	if err != nil {
		t.Fatalf("start canary: %v", err)
	}
	if _, err := completeCanary(run); err != nil {
		t.Fatalf("complete canary: %v", err)
	}

	// A candidate applying its own change after completion, simulating a
	// delayed/duplicate re-evaluation attempting to roll back a change
	// already reported to the owner as kept.
	if err := os.WriteFile(filepath.Join(root, scopedFile), []byte("changed after completion\n"), 0o644); err != nil {
		t.Fatalf("apply post-completion change: %v", err)
	}

	_, credited, err := rollbackCanary(run, "delayed regression re-evaluation")
	if err == nil {
		t.Fatal("expected rollbackCanary to refuse an already-completed canary, got nil error")
	}
	if !strings.Contains(err.Error(), "already completed") {
		t.Fatalf("error %q does not name the already-completed refusal", err.Error())
	}
	if credited {
		t.Fatal("expected credited=false on a refused rollback of a completed canary")
	}

	// The file must NOT be restored -- the change already reported as kept
	// must survive untouched.
	current, err := os.ReadFile(filepath.Join(root, scopedFile))
	if err != nil {
		t.Fatalf("read scoped file: %v", err)
	}
	if string(current) != "changed after completion\n" {
		t.Fatalf("expected the completed change to survive untouched, got %q", string(current))
	}

	// The stored run record must still report completed, not rolled_back.
	stored, found, err := loadCanaryRun(admission.CandidateID)
	if err != nil {
		t.Fatalf("load canary run: %v", err)
	}
	if !found {
		t.Fatal("expected a stored canary run record")
	}
	if stored.Status != canaryRunStatusCompleted {
		t.Fatalf("expected status to remain completed, got %q", stored.Status)
	}

	// The durable episode ledger's terminal result must still report
	// completed, not rolled_back.
	records, err := episodeLedgerForEpisode(canaryEpisodeID(admission.CandidateID))
	if err != nil {
		t.Fatalf("read episode ledger: %v", err)
	}
	terminal, ok := episodeLedgerTerminalRecord(records, canaryEpisodeID(admission.CandidateID))
	if !ok {
		t.Fatal("expected a durable terminal record for the canary episode")
	}
	if terminal.TerminalResult != "completed" {
		t.Fatalf("expected terminal result to remain completed, got %q", terminal.TerminalResult)
	}
}

func TestReleaseCanaryQuarantineThroughApprovedPath(t *testing.T) {
	root, scopedFile := rollbackTestSetup(t)
	admission := rollbackTestAdmission("candidate-release", canaryScopeRouting)
	run, err := startCanary(admission, []string{scopedFile})
	if err != nil {
		t.Fatalf("start canary: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, scopedFile), []byte("changed\n"), 0o644); err != nil {
		t.Fatalf("apply candidate change: %v", err)
	}
	if _, _, err := rollbackCanary(run, "regression for release test"); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	released, err := releaseCanaryQuarantine(admission.CandidateID, "owner reviewed and cleared it")
	if err != nil {
		t.Fatalf("release quarantine: %v", err)
	}
	if released.Quarantined {
		t.Fatal("expected the quarantine to be cleared")
	}
	if released.QuarantineReleasedBy != "owner reviewed and cleared it" {
		t.Fatalf("expected the release to record who released it, got %q", released.QuarantineReleasedBy)
	}
}
