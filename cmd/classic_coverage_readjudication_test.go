package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// loadClassicCoveragePhase204 mirrors loadClassicCoveragePhase203
// (cmd/classic_coverage_ratchet_test.go) for phase "204".
func loadClassicCoveragePhase204(t *testing.T, root string) classicCoverageDocument {
	t.Helper()
	path, err := classicCoverageJSONPathInRoot(root, "204")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read classic coverage: %v", err)
	}
	var document classicCoverageDocument
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatalf("decode classic coverage: %v", err)
	}
	return document
}

// TestClassicCoveragePhase204Signed loads 204-CLASSIC-COVERAGE.json and runs
// the shared validator for phase "204", expecting no error. Phase 204's
// study (204-CLASSIC-SYNTHESIS.md ruling (c)) flagged CAP-001 and CAP-057 as
// frozen against a state the code has since left; both were independently
// confirmed correct in direction, so both rows carry the ledger's own
// disposition value with the ruling recorded as refined evidence -- neither
// needed the re-adjudication marker this file's other tests exercise.
func TestClassicCoveragePhase204Signed(t *testing.T) {
	root := findTestModuleRootForClassicCoverage199()
	document := loadClassicCoveragePhase204(t, root)
	if err := validateClassicCoverage(document, root, "204"); err != nil {
		t.Fatal(err)
	}
}

// classicCoverageReadjudicatedRowIDs is the exact, named set of
// "<phase>/<id>" identifiers currently allowed to carry the
// re-adjudication marker (the literal "re-adjudicated:" substring
// validateClassicCoverageDispositionMatchesLedger already recognizes as an
// escape hatch from the frozen ledger's disposition), across every signed
// coverage file in the repository -- never hand-widened without this list
// being edited by name. Phase 204's own study (ruling (c)) considered
// CAP-001 and CAP-057 for this marker and, on the evidence, concluded both
// frozen dispositions are correct in direction -- so the set stays empty.
var classicCoverageReadjudicatedRowIDs = []string{}

// classicCoverageReadjudicationCeiling bounds how many rows, across the
// whole repository, may ever carry the re-adjudication marker at once. It
// may only shrink, never grow -- a phase that legitimately needs a new
// re-adjudication raises this constant in the SAME commit that adds the
// row's own id to classicCoverageReadjudicatedRowIDs, so the two can never
// silently drift apart.
const classicCoverageReadjudicationCeiling = 0

const classicCoverageReadjudicationMarker = "re-adjudicated:"

var classicCoverageReadjudicationCitationPattern = regexp.MustCompile(`re-adjudicated:\s*"([^"]+)"`)

// classicCoverageReadjudicatedRow is one GOAL row, from one signed coverage
// file, that carries the re-adjudication marker.
type classicCoverageReadjudicatedRow struct {
	Phase    string
	ID       string
	Citation string
}

type classicCoverageSignedFile struct {
	Phase    string
	Document classicCoverageDocument
}

// classicCoverageAllSignedFiles walks every existing
// <phase>-CLASSIC-COVERAGE.json file under root's .planning/phases
// directory (whichever phases happen to be signed at the time this test
// runs), returning each file's phase id and parsed document, sorted by
// phase id.
func classicCoverageAllSignedFiles(root string) ([]classicCoverageSignedFile, error) {
	phasesDir := filepath.Join(root, ".planning", "phases")
	entries, err := os.ReadDir(phasesDir)
	if err != nil {
		return nil, fmt.Errorf("read phases directory: %w", err)
	}
	var files []classicCoverageSignedFile
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		matches, err := filepath.Glob(filepath.Join(phasesDir, entry.Name(), "*-CLASSIC-COVERAGE.json"))
		if err != nil {
			return nil, err
		}
		for _, match := range matches {
			phase := strings.TrimSuffix(filepath.Base(match), "-CLASSIC-COVERAGE.json")
			raw, err := os.ReadFile(match)
			if err != nil {
				return nil, fmt.Errorf("read %s: %w", match, err)
			}
			var document classicCoverageDocument
			if err := json.Unmarshal(raw, &document); err != nil {
				return nil, fmt.Errorf("decode %s: %w", match, err)
			}
			files = append(files, classicCoverageSignedFile{Phase: phase, Document: document})
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Phase < files[j].Phase })
	return files, nil
}

// classicCoverageExtractReadjudicationCitation pulls the quoted heading
// text following the "re-adjudicated:" marker out of a row's
// historical_evidence. Returns "" when the marker is present but the
// citation is malformed (no quoted text follows it) -- a malformed
// citation can never resolve, so it fails the same way an absent one would.
func classicCoverageExtractReadjudicationCitation(historicalEvidence string) string {
	match := classicCoverageReadjudicationCitationPattern.FindStringSubmatch(historicalEvidence)
	if len(match) != 2 {
		return ""
	}
	return match[1]
}

// classicCoverageReadjudicatedRowsInRoot scans every signed coverage file
// under root for GOAL rows carrying the re-adjudication marker and returns
// one classicCoverageReadjudicatedRow per match, sorted by phase then id.
func classicCoverageReadjudicatedRowsInRoot(root string) ([]classicCoverageReadjudicatedRow, error) {
	files, err := classicCoverageAllSignedFiles(root)
	if err != nil {
		return nil, err
	}
	var rows []classicCoverageReadjudicatedRow
	for _, file := range files {
		for _, row := range file.Document.Rows {
			if row.Type != "GOAL" {
				continue
			}
			if !strings.Contains(row.HistoricalEvidence, classicCoverageReadjudicationMarker) {
				continue
			}
			rows = append(rows, classicCoverageReadjudicatedRow{
				Phase:    file.Phase,
				ID:       row.ID,
				Citation: classicCoverageExtractReadjudicationCitation(row.HistoricalEvidence),
			})
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Phase != rows[j].Phase {
			return rows[i].Phase < rows[j].Phase
		}
		return rows[i].ID < rows[j].ID
	})
	return rows, nil
}

// classicCoverageStudyHeadings reads <phase>-CLASSIC-SYNTHESIS.md and
// returns the trimmed text of every markdown heading line (stripped of
// leading '#' characters), so a re-adjudication citation can be checked
// against real section headings rather than free text.
func classicCoverageStudyHeadings(root, phase string) (map[string]bool, error) {
	dir, err := classicCoveragePhaseDir(root, phase)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, phase+"-CLASSIC-SYNTHESIS.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s study: %w", phase, err)
	}
	headings := map[string]bool{}
	for _, line := range strings.Split(string(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "#") {
			continue
		}
		heading := strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
		headings[heading] = true
	}
	return headings, nil
}

// validateClassicCoverageReadjudicationCitationResolves fails, naming the
// row, when a re-adjudicated row's citation is missing, malformed, or does
// not resolve to a real heading in that phase's own study file.
func validateClassicCoverageReadjudicationCitationResolves(root string, row classicCoverageReadjudicatedRow) error {
	if row.Citation == "" {
		return fmt.Errorf("GOAL/%s (phase %s) re-adjudication citation is missing or malformed", row.ID, row.Phase)
	}
	headings, err := classicCoverageStudyHeadings(root, row.Phase)
	if err != nil {
		return err
	}
	if !headings[row.Citation] {
		return fmt.Errorf("GOAL/%s (phase %s) re-adjudication citation %q does not resolve to a heading in %s-CLASSIC-SYNTHESIS.md", row.ID, row.Phase, row.Citation, row.Phase)
	}
	return nil
}

func classicCoverageEqualStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// validateClassicCoverageReadjudicationBounds is the one function both the
// real-repository check and the synthetic-violation check below call: it
// walks every signed coverage file under root, requires every
// re-adjudicated row's citation to resolve, requires the exact collected
// id set to equal classicCoverageReadjudicatedRowIDs (never merely a
// count), and requires the collected count to be at or below
// classicCoverageReadjudicationCeiling.
func validateClassicCoverageReadjudicationBounds(root string) error {
	rows, err := classicCoverageReadjudicatedRowsInRoot(root)
	if err != nil {
		return err
	}
	var errs []string
	for _, row := range rows {
		if err := validateClassicCoverageReadjudicationCitationResolves(root, row); err != nil {
			errs = append(errs, err.Error())
		}
	}
	var ids []string
	for _, row := range rows {
		ids = append(ids, row.Phase+"/"+row.ID)
	}
	sort.Strings(ids)
	want := append([]string(nil), classicCoverageReadjudicatedRowIDs...)
	sort.Strings(want)
	if !classicCoverageEqualStringSlices(ids, want) {
		errs = append(errs, fmt.Sprintf("re-adjudicated rows = %v, want exactly the declared set %v", ids, want))
	}
	if len(rows) > classicCoverageReadjudicationCeiling {
		errs = append(errs, fmt.Sprintf("%d re-adjudicated rows exceeds the declared ceiling of %d", len(rows), classicCoverageReadjudicationCeiling))
	}
	if len(errs) == 0 {
		return nil
	}
	sort.Strings(errs)
	return errors.New(strings.Join(errs, "; "))
}

// TestClassicCoverageReadjudicationsAreCitedAndBounded proves
// validateClassicCoverageReadjudicationBounds against the real repository
// (currently zero re-adjudicated rows, matching 204-CLASSIC-SYNTHESIS.md
// ruling (c)'s conclusion that both CAP-001 and CAP-057 are correct in
// direction), and then proves the SAME function genuinely refuses, by
// name, a re-adjudicated row that the declared set does not already name --
// never a vacuous pass.
func TestClassicCoverageReadjudicationsAreCitedAndBounded(t *testing.T) {
	root := findTestModuleRootForClassicCoverage199()
	if err := validateClassicCoverageReadjudicationBounds(root); err != nil {
		t.Fatal(err)
	}

	t.Run("an undeclared re-adjudicated row is refused by name", func(t *testing.T) {
		synthRoot := t.TempDir()
		dir := filepath.Join(synthRoot, ".planning", "phases", "999-synthetic")
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		document := classicCoverageDocument{
			Version: "999/classic-coverage/v1",
			Rows: []classicCoverageRow{
				{Type: "GOAL", ID: "CAP-999", Disposition: "restore-modern", HistoricalEvidence: `re-adjudicated: "Synthetic Ruling"`},
			},
		}
		raw, err := json.Marshal(document)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "999-CLASSIC-COVERAGE.json"), raw, 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "999-CLASSIC-SYNTHESIS.md"), []byte("### Synthetic Ruling\n"), 0600); err != nil {
			t.Fatal(err)
		}

		err = validateClassicCoverageReadjudicationBounds(synthRoot)
		if err == nil {
			t.Fatal("expected an undeclared re-adjudicated row to fail validation")
		}
		if !strings.Contains(err.Error(), "999/CAP-999") {
			t.Fatalf("error does not name the offending row: %v", err)
		}
	})
}

// TestClassicCoverageReadjudicationCitationMustResolve takes a real signed
// Phase 204 row, clones it in memory, gives it a citation naming a section
// that does not exist anywhere in the phase's own study, and asserts the
// check fails naming the row. Operates entirely on an in-memory clone --
// the real coverage file is never touched.
func TestClassicCoverageReadjudicationCitationMustResolve(t *testing.T) {
	root := findTestModuleRootForClassicCoverage199()
	document := loadClassicCoveragePhase204(t, root)
	clone := cloneClassicCoverageDocument(t, document)
	targetID := clone.Rows[0].ID
	clone.Rows[0].HistoricalEvidence = `re-adjudicated: "A Section Heading That Does Not Exist In This Study"`

	row := classicCoverageReadjudicatedRow{
		Phase:    "204",
		ID:       clone.Rows[0].ID,
		Citation: classicCoverageExtractReadjudicationCitation(clone.Rows[0].HistoricalEvidence),
	}
	err := validateClassicCoverageReadjudicationCitationResolves(root, row)
	if err == nil {
		t.Fatal("expected a non-resolving citation to fail validation")
	}
	if !strings.Contains(err.Error(), targetID) {
		t.Fatalf("error does not name %s: %v", targetID, err)
	}
}

// TestClassicCoverageUncitedDispositionChangeFails clones a real signed
// Phase 204 row, changes its disposition away from the frozen ledger's
// value with no re-adjudication marker at all, and asserts
// validateClassicCoverageDispositionMatchesLedger fails naming the row,
// the signed value, and the ledger value. Operates entirely on an
// in-memory clone.
func TestClassicCoverageUncitedDispositionChangeFails(t *testing.T) {
	root := findTestModuleRootForClassicCoverage199()
	document := loadClassicCoveragePhase204(t, root)
	routing, err := loadClassicCoverageLedgerRouting(root)
	if err != nil {
		t.Fatal(err)
	}
	clone := cloneClassicCoverageDocument(t, document)
	targetID := clone.Rows[0].ID
	route := routing[targetID]
	newDisposition := "replace-better"
	if route.Disposition == "replace-better" {
		newDisposition = "restore-modern"
	}
	clone.Rows[0].Disposition = newDisposition

	err = validateClassicCoverageDispositionMatchesLedger(clone, routing)
	if err == nil {
		t.Fatal("expected an uncited disposition change to fail validation")
	}
	if !strings.Contains(err.Error(), targetID) {
		t.Fatalf("error does not name %s: %v", targetID, err)
	}
	if !strings.Contains(err.Error(), newDisposition) {
		t.Fatalf("error does not name the signed value %q: %v", newDisposition, err)
	}
	if !strings.Contains(err.Error(), route.Disposition) {
		t.Fatalf("error does not name the ledger value %q: %v", route.Disposition, err)
	}
}
