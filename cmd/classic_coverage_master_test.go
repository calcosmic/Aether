package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func TestClassicCoveragePhase205Signed(t *testing.T) {
	root := findTestModuleRootForClassicCoverage199()
	document := loadClassicCoveragePhase(t, root, "205")
	if document.Version != "205/classic-coverage/v1" {
		t.Fatalf("phase 205 coverage version = %q, want 205/classic-coverage/v1", document.Version)
	}
	for _, row := range document.Rows {
		if row.Type != "GOAL" {
			t.Fatalf("phase 205 coverage contains %s/%s, want only GOAL rows", row.Type, row.ID)
		}
	}
	if err := validateClassicCoverage(document, root, "205"); err != nil {
		t.Fatal(err)
	}
}

// Load the phases the frozen ledger actually routes to. A missing file is
// fatal through the shared loader; a phase with no routed rows needs no file.
// The shared schema preserves Phase 199's non-GOAL rows for its validators,
// while only GOAL rows contribute to the master capability union.
func loadClassicCoverageMaster(t *testing.T, root string) []classicCoverageSignedFile {
	t.Helper()
	routing, err := loadClassicCoverageLedgerRouting(root)
	if err != nil {
		t.Fatal(err)
	}
	phases := map[string]bool{}
	for _, route := range routing {
		phases[route.Phase] = true
	}
	var ordered []string
	for phase := range phases {
		ordered = append(ordered, phase)
	}
	sort.Strings(ordered)
	var files []classicCoverageSignedFile
	for _, phase := range ordered {
		files = append(files, classicCoverageSignedFile{
			Phase: phase, Document: loadClassicCoveragePhase(t, root, phase),
		})
	}
	return files
}

var classicCoverageMasterIDPattern = regexp.MustCompile(`CAP-[0-9]+`)

// validateClassicCoverageMaster adds cross-file identity and routing checks
// to the existing phase validator, rather than treating a matching count as
// proof. The returned summary describes counts even on failure; only a nil
// error certifies the documents. All inputs and their backing files are read-only.
func validateClassicCoverageMaster(files []classicCoverageSignedFile, root string) (string, error) {
	routing, err := loadClassicCoverageLedgerRouting(root)
	if err != nil {
		return "", err
	}
	if len(routing) == 0 {
		return "", errors.New("classic coverage ledger has no routed capability ids")
	}
	seen := map[string][]string{}
	ids := map[string]bool{}
	for id := range routing {
		ids[id] = true
	}
	rowCount := 0
	for _, file := range files {
		for _, row := range file.Document.Rows {
			if row.Type != "GOAL" {
				continue
			}
			seen[row.ID] = append(seen[row.ID], file.Phase)
			ids[row.ID] = true
			rowCount++
		}
	}
	summary := fmt.Sprintf("Classic coverage: %d distinct capability ids; %d capability rows; %d expected ids; %d phase files.",
		len(seen), rowCount, len(routing), len(files))
	var orderedIDs []string
	for id := range ids {
		orderedIDs = append(orderedIDs, id)
	}
	sort.Strings(orderedIDs)
	var problems []string
	for _, id := range orderedIDs {
		phases := seen[id]
		sort.Strings(phases)
		route, expected := routing[id]
		if !expected {
			problems = append(problems, fmt.Sprintf("GOAL/%s is absent from the frozen ledger; found in phases %v", id, phases))
			continue
		}
		if !containsString(phases, route.Phase) {
			problems = append(problems, fmt.Sprintf("GOAL/%s is missing from routed phase %s; found in phases %v", id, route.Phase, phases))
		}
		if len(phases) > 1 {
			problems = append(problems, fmt.Sprintf("GOAL/%s appears %d times in phases %v; want exactly once in routed phase %s", id, len(phases), phases, route.Phase))
		}
		for _, phase := range phases {
			if phase != route.Phase {
				problems = append(problems, fmt.Sprintf("GOAL/%s found in phase %s (%s-CLASSIC-COVERAGE.json); ledger routes it to phase %s", id, phase, phase, route.Phase))
			}
		}
	}
	// Structural errors are already complete and more useful than redundant
	// per-phase "extra/missing" messages. Once routing is valid, validate the
	// actual evidence, status, disposition and rendered companion in every file.
	if len(problems) == 0 {
		for _, file := range files {
			if err := validateClassicCoverage(file.Document, root, file.Phase); err != nil {
				// Shared validators join individual diagnostics with "; ".
				// Sort them globally by capability, not by their phase of origin.
				for _, problem := range strings.Split(err.Error(), "; ") {
					problems = append(problems, fmt.Sprintf("%s (phase %s)", problem, file.Phase))
				}
			}
		}
	}
	if len(problems) == 0 {
		return summary, nil
	}
	sort.Slice(problems, func(i, j int) bool {
		a, b := classicCoverageMasterIDPattern.FindString(problems[i]), classicCoverageMasterIDPattern.FindString(problems[j])
		if a != b {
			return a < b
		}
		return problems[i] < problems[j]
	})
	return summary, errors.New(strings.Join(problems, "\n"))
}

func TestClassicCoverageMasterUnionIsCompleteAndUnique(t *testing.T) {
	root := findTestModuleRootForClassicCoverage199()
	files := loadClassicCoverageMaster(t, root)
	summary, err := validateClassicCoverageMaster(files, root)
	t.Log(summary)
	if err != nil {
		t.Fatal(err)
	}
	// The existing cross-phase check also constrains the shared validator's
	// re-adjudication escape hatch. A marker cannot silently widen the allowance.
	if err := validateClassicCoverageReadjudicationBounds(root); err != nil {
		t.Fatal(err)
	}
}

func cloneClassicCoverageMaster(t *testing.T) (string, []classicCoverageSignedFile) {
	t.Helper()
	root := findTestModuleRootForClassicCoverage199()
	files := loadClassicCoverageMaster(t, root)
	for i := range files {
		files[i].Document = cloneClassicCoverageDocument(t, files[i].Document)
	}
	return root, files
}

func classicCoverageMasterFirstGoal(t *testing.T, file classicCoverageSignedFile) int {
	t.Helper()
	for i, row := range file.Document.Rows {
		if row.Type == "GOAL" {
			return i
		}
	}
	t.Fatalf("phase %s has no capability row", file.Phase)
	return -1
}

func requireClassicCoverageMasterFailure(t *testing.T, files []classicCoverageSignedFile, root string, fragments ...string) string {
	t.Helper()
	summary, err := validateClassicCoverageMaster(files, root)
	if err == nil {
		t.Fatalf("invalid coverage passed: %s", summary)
	}
	for _, fragment := range fragments {
		if !strings.Contains(err.Error(), fragment) {
			t.Errorf("failure does not name %q:\n%s", fragment, err)
		}
	}
	return err.Error()
}

func TestClassicCoverageMasterFailsOnAMissingRow(t *testing.T) {
	root, files := cloneClassicCoverageMaster(t)
	i := classicCoverageMasterFirstGoal(t, files[0])
	id, phase := files[0].Document.Rows[i].ID, files[0].Phase
	files[0].Document.Rows = append(files[0].Document.Rows[:i], files[0].Document.Rows[i+1:]...)
	requireClassicCoverageMasterFailure(t, files, root, id, "missing from routed phase "+phase)
}

func TestClassicCoverageMasterFailsOnADuplicateRow(t *testing.T) {
	root, files := cloneClassicCoverageMaster(t)
	i := classicCoverageMasterFirstGoal(t, files[0])
	row := files[0].Document.Rows[i]
	files[1].Document.Rows = append(files[1].Document.Rows, row)
	requireClassicCoverageMasterFailure(t, files, root, row.ID, "appears 2 times", files[0].Phase, files[1].Phase)
}

func TestClassicCoverageMasterFailsOnAMisroutedRow(t *testing.T) {
	root, files := cloneClassicCoverageMaster(t)
	i := classicCoverageMasterFirstGoal(t, files[0])
	row := files[0].Document.Rows[i]
	files[0].Document.Rows = append(files[0].Document.Rows[:i], files[0].Document.Rows[i+1:]...)
	files[1].Document.Rows = append(files[1].Document.Rows, row)
	requireClassicCoverageMasterFailure(t, files, root, row.ID, "missing from routed phase "+files[0].Phase,
		files[1].Phase+"-CLASSIC-COVERAGE.json", "ledger routes it to phase "+files[0].Phase)
}

func TestClassicCoverageMasterFailsOnAnExtraRow(t *testing.T) {
	root, files := cloneClassicCoverageMaster(t)
	routing, err := loadClassicCoverageLedgerRouting(root)
	if err != nil {
		t.Fatal(err)
	}
	row := files[0].Document.Rows[classicCoverageMasterFirstGoal(t, files[0])]
	for n := len(routing) + 1; ; n++ {
		row.ID = fmt.Sprintf("CAP-%03d", n)
		if _, exists := routing[row.ID]; !exists {
			break
		}
	}
	files[0].Document.Rows = append(files[0].Document.Rows, row)
	requireClassicCoverageMasterFailure(t, files, root, row.ID, "absent from the frozen ledger", files[0].Phase)
}

func TestClassicCoverageMasterReportsCountsNotPercentages(t *testing.T) {
	root, files := cloneClassicCoverageMaster(t)
	summary, err := validateClassicCoverageMaster(files, root)
	if err != nil {
		t.Fatal(err)
	}
	pattern := regexp.MustCompile(`^Classic coverage: ([0-9]+) distinct capability ids; ([0-9]+) capability rows; ([0-9]+) expected ids; ([0-9]+) phase files\.$`)
	counts := pattern.FindStringSubmatch(summary)
	if counts == nil {
		t.Fatalf("summary must contain only whole-number counts, never a percentage or proportion: %q", summary)
	}
	routing, err := loadClassicCoverageLedgerRouting(root)
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []int{len(routing), len(routing), len(routing), len(files)} {
		if counts[i+1] != strconv.Itoa(want) {
			t.Errorf("summary count %d = %s, want %d: %s", i+1, counts[i+1], want, summary)
		}
	}
	for _, fraction := range []string{"100%", "1.0", "1/1", "1 of 1"} {
		corrupt := strings.Replace(summary, counts[1]+" distinct", fraction+" distinct", 1)
		if pattern.MatchString(corrupt) {
			t.Errorf("summary check accepted a percentage or proportion: %q", corrupt)
		}
	}
}

func TestClassicCoverageMasterFailureOutputIsOrderedAndStable(t *testing.T) {
	root, files := cloneClassicCoverageMaster(t)
	var ids []string
	for _, fileIndex := range []int{2, 0, 1} {
		file := &files[fileIndex]
		i := classicCoverageMasterFirstGoal(t, *file)
		ids = append(ids, file.Document.Rows[i].ID)
		file.Document.Rows = append(file.Document.Rows[:i], file.Document.Rows[i+1:]...)
	}
	sort.Strings(ids)
	first := requireClassicCoverageMasterFailure(t, files, root, ids...)
	second := requireClassicCoverageMasterFailure(t, files, root, ids...)
	if first != second {
		t.Fatalf("failure output changed across consecutive runs:\n%s\n%s", first, second)
	}
	for i := 1; i < len(ids); i++ {
		if strings.Index(first, ids[i-1]) >= strings.Index(first, ids[i]) {
			t.Fatalf("failure ids are not in ascending order: %s", first)
		}
	}
	// Neither file order nor row order may influence the diagnostics.
	for i, j := 0, len(files)-1; i < j; i, j = i+1, j-1 {
		files[i], files[j] = files[j], files[i]
	}
	for _, file := range files {
		rows := file.Document.Rows
		for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
			rows[i], rows[j] = rows[j], rows[i]
		}
	}
	if reordered := requireClassicCoverageMasterFailure(t, files, root, ids...); reordered != first {
		t.Fatalf("input order changed failure output:\n%s\n%s", first, reordered)
	}
}

func TestClassicCoverageMasterRejectsInvalidEvidence(t *testing.T) {
	root, baseline := cloneClassicCoverageMaster(t)
	for _, file := range baseline {
		t.Run("unresolved proof in phase "+file.Phase, func(t *testing.T) {
			_, files := cloneClassicCoverageMaster(t)
			for i := range files {
				if files[i].Phase != file.Phase {
					continue
				}
				row := &files[i].Document.Rows[classicCoverageMasterFirstGoal(t, files[i])]
				row.Proofs = []string{"TestClassicCoverageMasterNonexistentProof"}
				requireClassicCoverageMasterFailure(t, files, root, row.ID, file.Phase, "does not resolve")
			}
		})
	}
	for _, tc := range []struct {
		name   string
		mutate func(*classicCoverageRow)
	}{
		{"invalid disposition", func(row *classicCoverageRow) { row.Disposition = "unrecognized-disposition" }},
		{"nonpassing status", func(row *classicCoverageRow) { row.Status = "PENDING" }},
		{"blank home", func(row *classicCoverageRow) { row.ModernHome = "" }},
		{"unresolved artifact", func(row *classicCoverageRow) { row.Artifact = "cmd/classic_coverage_master_absent.go" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, files := cloneClassicCoverageMaster(t)
			row := &files[0].Document.Rows[classicCoverageMasterFirstGoal(t, files[0])]
			tc.mutate(row)
			requireClassicCoverageMasterFailure(t, files, root, row.ID, files[0].Phase)
		})
	}
}

func TestClassicCoverageMasterFailsWhenPhaseFilesAreAbsent(t *testing.T) {
	// Real routing, empty temporary coverage directory: no real record is
	// deleted, and the same master check must name every affected id and phase.
	root := t.TempDir()
	source := findTestModuleRootForClassicCoverage199()
	ledger, err := os.ReadFile(filepath.Join(source, classicCoverageLedgerPath))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, classicCoverageLedgerPath)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, ledger, 0600); err != nil {
		t.Fatal(err)
	}
	routing, err := loadClassicCoverageLedgerRouting(root)
	if err != nil {
		t.Fatal(err)
	}
	var fragments []string
	for id, route := range routing {
		fragments = append(fragments, "GOAL/"+id+" is missing from routed phase "+route.Phase)
	}
	requireClassicCoverageMasterFailure(t, nil, root, fragments...)
}
