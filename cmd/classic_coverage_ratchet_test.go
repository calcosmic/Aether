package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// classicCoverageDocument / classicCoverageRow generalize
// classicCoverage199Document / classicCoverage199Row (cmd/classic_coverage_199_test.go)
// into a phase-parameterized ratchet. The json tags are byte-identical to the
// Phase 199 schema on purpose -- no second CAP vocabulary.
type classicCoverageDocument struct {
	Version string               `json:"version"`
	Rows    []classicCoverageRow `json:"rows"`
}

type classicCoverageRow struct {
	Type               string   `json:"type"`
	ID                 string   `json:"id"`
	Disposition        string   `json:"disposition"`
	ModernHome         string   `json:"modern_home"`
	PlanTask           string   `json:"plan_task"`
	Artifact           string   `json:"artifact"`
	Proofs             []string `json:"proofs"`
	Status             string   `json:"status"`
	HistoricalEvidence string   `json:"historical_evidence"`
}

// classicCoverageLedgerRoute is one row of the frozen phase-mapping table in
// .planning/research/v1.28-classic-capability-ledger.md, section
// "v1.28 requirement and phase mapping" -- the only source of which CAP ids
// belong to which phase, and what disposition the ledger routed for each.
type classicCoverageLedgerRoute struct {
	Disposition string
	Requirement string
	Phase       string
}

const classicCoverageLedgerPath = ".planning/research/v1.28-classic-capability-ledger.md"
const classicCoverageLedgerSectionHeading = "## v1.28 requirement and phase mapping"

// loadClassicCoverageLedgerRouting parses the frozen routing table. Parsing
// never depends on awk/cut: table rows are split on the pipe character here,
// in Go.
func loadClassicCoverageLedgerRouting(root string) (map[string]classicCoverageLedgerRoute, error) {
	path := filepath.Join(root, filepath.FromSlash(classicCoverageLedgerPath))
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read classic capability ledger: %w", err)
	}
	routing := map[string]classicCoverageLedgerRoute{}
	inSection := false
	for _, line := range strings.Split(string(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			if inSection {
				break
			}
			if trimmed == classicCoverageLedgerSectionHeading {
				inSection = true
			}
			continue
		}
		if !inSection || !strings.HasPrefix(trimmed, "|") {
			continue
		}
		cells := splitClassicCoverageTableRow(trimmed)
		if len(cells) < 4 {
			continue
		}
		id := stripClassicCoverageTableCell(cells[0])
		if !strings.HasPrefix(id, "CAP-") {
			continue // header or separator row
		}
		dispositionRaw := stripClassicCoverageTableCell(cells[1])
		dispositionFields := strings.Fields(dispositionRaw)
		disposition := ""
		if len(dispositionFields) > 0 {
			disposition = dispositionFields[0]
		}
		routing[id] = classicCoverageLedgerRoute{
			Disposition: disposition,
			Requirement: stripClassicCoverageTableCell(cells[2]),
			Phase:       stripClassicCoverageTableCell(cells[3]),
		}
	}
	return routing, nil
}

func splitClassicCoverageTableRow(line string) []string {
	trimmed := strings.Trim(line, "|")
	return strings.Split(trimmed, "|")
}

func stripClassicCoverageTableCell(cell string) string {
	cell = strings.ReplaceAll(cell, "`", "")
	return strings.TrimSpace(cell)
}

// expectedClassicCoverageCAPIDs is the ONLY source of the expected CAP id set
// for a phase -- never hand-typed anywhere in this file.
func expectedClassicCoverageCAPIDs(routing map[string]classicCoverageLedgerRoute, phase string) []string {
	var ids []string
	for id, route := range routing {
		if route.Phase == phase {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

// classicCoveragePhaseDir resolves a phase's directory under
// .planning/phases/ by its numeric prefix (so "203" resolves to
// "203-biological-runtime").
func classicCoveragePhaseDir(root, phase string) (string, error) {
	phasesDir := filepath.Join(root, ".planning", "phases")
	entries, err := os.ReadDir(phasesDir)
	if err != nil {
		return "", fmt.Errorf("read phases directory for phase %q: %w", phase, err)
	}
	prefix := phase + "-"
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if entry.Name() == phase || strings.HasPrefix(entry.Name(), prefix) {
			return filepath.Join(phasesDir, entry.Name()), nil
		}
	}
	return "", fmt.Errorf("phase %q directory not found under %s", phase, phasesDir)
}

func classicCoverageJSONPathInRoot(root, phase string) (string, error) {
	dir, err := classicCoveragePhaseDir(root, phase)
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, phase+"-CLASSIC-COVERAGE.json")
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		return "", fmt.Errorf("phase %q coverage file not found at %s", phase, path)
	}
	return path, nil
}

func classicCoverageAuditPathInRoot(root, phase string) (string, error) {
	dir, err := classicCoveragePhaseDir(root, phase)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, phase+"-CLASSIC-COVERAGE.md"), nil
}

// classicCoveragePathForPhase resolves the phase directory under
// .planning/phases/ by its numeric prefix and returns
// <dir>/<phase>-CLASSIC-COVERAGE.json. An absent directory or an absent file
// is an error naming the phase, never a skip.
func classicCoveragePathForPhase(phase string) (string, error) {
	root := findTestModuleRootForClassicCoverage199()
	return classicCoverageJSONPathInRoot(root, phase)
}

// validateClassicCoverageExactCAPSet: every type: GOAL row's id appears
// exactly once and the set of GOAL ids equals expected; a missing id, an
// extra id, and a duplicate id each produce a distinct error naming the id.
func validateClassicCoverageExactCAPSet(document classicCoverageDocument, expected []string) error {
	seen := map[string]int{}
	for _, row := range document.Rows {
		if row.Type != "GOAL" {
			continue
		}
		seen[row.ID]++
	}
	expectedSet := map[string]bool{}
	sortedExpected := append([]string(nil), expected...)
	sort.Strings(sortedExpected)
	for _, id := range sortedExpected {
		expectedSet[id] = true
	}

	var errs []string
	for _, id := range sortedExpected {
		switch seen[id] {
		case 0:
			errs = append(errs, fmt.Sprintf("GOAL/%s is missing", id))
		case 1:
			// present exactly once -- fine
		default:
			errs = append(errs, fmt.Sprintf("GOAL/%s appears %d times, want exactly one", id, seen[id]))
		}
	}
	var extraIDs []string
	for id := range seen {
		if !expectedSet[id] {
			extraIDs = append(extraIDs, id)
		}
	}
	sort.Strings(extraIDs)
	for _, id := range extraIDs {
		errs = append(errs, fmt.Sprintf("GOAL/%s is not routed to this phase by the ledger", id))
	}
	if len(errs) == 0 {
		return nil
	}
	sort.Strings(errs)
	return errors.New(strings.Join(errs, "; "))
}

// validateClassicCoverageRequiredFields mirrors
// validateClassicCoverage199RequiredFields's rule set, but collects every
// violation (in a fixed field order, never a map range) so repeated runs
// against the same bad document are byte-identical.
func validateClassicCoverageRequiredFields(document classicCoverageDocument) error {
	orderedFields := []string{"id", "disposition", "modern_home", "plan_task", "artifact", "status", "historical_evidence"}
	var errs []string
	for _, row := range document.Rows {
		values := map[string]string{
			"id": row.ID, "disposition": row.Disposition, "modern_home": row.ModernHome,
			"plan_task": row.PlanTask, "artifact": row.Artifact, "status": row.Status,
			"historical_evidence": row.HistoricalEvidence,
		}
		for _, field := range orderedFields {
			value := values[field]
			if strings.TrimSpace(value) == "" || containsClassicCoverage199Placeholder(value) {
				errs = append(errs, fmt.Sprintf("%s/%s has invalid %s %q", row.Type, row.ID, field, value))
			}
		}
		if len(row.Proofs) == 0 {
			errs = append(errs, fmt.Sprintf("%s/%s has no proof", row.Type, row.ID))
		}
		if row.Disposition != "keep-current" && row.Disposition != "restore-modern" && row.Disposition != "replace-better" && row.Disposition != "retire-with-proof" {
			errs = append(errs, fmt.Sprintf("%s/%s has unsupported disposition %q", row.Type, row.ID, row.Disposition))
		}
	}
	if len(errs) == 0 {
		return nil
	}
	sort.Strings(errs)
	return errors.New(strings.Join(errs, "; "))
}

// validateClassicCoverageProofResolution reuses buildClassicCoverage199ProofIndex
// directly -- the AST-based proof index is never rebuilt for this ratchet.
func validateClassicCoverageProofResolution(document classicCoverageDocument, root string) error {
	index, err := buildClassicCoverage199ProofIndex(root)
	if err != nil {
		return err
	}
	var errs []string
	for _, row := range document.Rows {
		artifact := filepath.Join(root, filepath.FromSlash(row.Artifact))
		if info, statErr := os.Stat(artifact); statErr != nil || info.IsDir() {
			errs = append(errs, fmt.Sprintf("%s/%s artifact %q does not resolve", row.Type, row.ID, row.Artifact))
		}
		for _, proof := range row.Proofs {
			if !index.has(proof) {
				errs = append(errs, fmt.Sprintf("%s/%s proof %q does not resolve", row.Type, row.ID, proof))
			}
		}
	}
	if len(errs) == 0 {
		return nil
	}
	sort.Strings(errs)
	return errors.New(strings.Join(errs, "; "))
}

func validateClassicCoveragePassingStatus(document classicCoverageDocument) error {
	var errs []string
	for _, row := range document.Rows {
		if row.Status != "PASS" {
			errs = append(errs, fmt.Sprintf("%s/%s status is %q, want PASS", row.Type, row.ID, row.Status))
		}
	}
	if len(errs) == 0 {
		return nil
	}
	sort.Strings(errs)
	return errors.New(strings.Join(errs, "; "))
}

// validateClassicCoverageDispositionMatchesLedger: a signed GOAL disposition
// must equal the ledger's routed disposition for that id, UNLESS the row's
// historical_evidence cites a recorded re-adjudication via the literal
// "re-adjudicated:" marker.
func validateClassicCoverageDispositionMatchesLedger(document classicCoverageDocument, routing map[string]classicCoverageLedgerRoute) error {
	var errs []string
	for _, row := range document.Rows {
		if row.Type != "GOAL" {
			continue
		}
		route, ok := routing[row.ID]
		if !ok {
			continue // an unrouted id is already reported by validateClassicCoverageExactCAPSet
		}
		if row.Disposition == route.Disposition {
			continue
		}
		if strings.Contains(row.HistoricalEvidence, "re-adjudicated:") {
			continue
		}
		errs = append(errs, fmt.Sprintf("%s/%s disposition %q does not match ledger disposition %q for %s", row.Type, row.ID, row.Disposition, route.Disposition, row.ID))
	}
	if len(errs) == 0 {
		return nil
	}
	sort.Strings(errs)
	return errors.New(strings.Join(errs, "; "))
}

// validateClassicCoverageAudit: the companion <phase>-CLASSIC-COVERAGE.md
// must exist and contain a rendered table row pairing each JSON row's id with
// its disposition, and a "## GOAL" heading.
func validateClassicCoverageAudit(document classicCoverageDocument, root, phase string) error {
	path, err := classicCoverageAuditPathInRoot(root, phase)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read coverage audit: %w", err)
	}
	audit := string(raw)
	var errs []string
	if !strings.Contains(audit, "## GOAL\n") {
		errs = append(errs, "coverage audit lacks GOAL heading")
	}
	for _, row := range document.Rows {
		needle := "| " + row.ID + " | " + row.Disposition + " | "
		if !strings.Contains(audit, needle) {
			errs = append(errs, fmt.Sprintf("coverage audit lacks rendered row for %s/%s", row.Type, row.ID))
		}
	}
	if len(errs) == 0 {
		return nil
	}
	sort.Strings(errs)
	return errors.New(strings.Join(errs, "; "))
}

// validateClassicCoverage runs the validators above in a fixed order: exact
// CAP set, required fields, proof resolution, passing status, disposition
// vs. ledger, and the rendered audit companion.
func validateClassicCoverage(document classicCoverageDocument, root, phase string) error {
	routing, err := loadClassicCoverageLedgerRouting(root)
	if err != nil {
		return err
	}
	expected := expectedClassicCoverageCAPIDs(routing, phase)
	if err := validateClassicCoverageExactCAPSet(document, expected); err != nil {
		return err
	}
	if err := validateClassicCoverageRequiredFields(document); err != nil {
		return err
	}
	if err := validateClassicCoverageProofResolution(document, root); err != nil {
		return err
	}
	if err := validateClassicCoveragePassingStatus(document); err != nil {
		return err
	}
	if err := validateClassicCoverageDispositionMatchesLedger(document, routing); err != nil {
		return err
	}
	return validateClassicCoverageAudit(document, root, phase)
}

func loadClassicCoveragePhase203(t *testing.T, root string) classicCoverageDocument {
	t.Helper()
	path, err := classicCoverageJSONPathInRoot(root, "203")
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

// TestClassicCoverageLedgerRoutingIsUniqueAndComplete makes the derived
// expected set trustworthy: the parsed routing covers CAP-001 through
// CAP-072, each exactly once, with no gaps and no extras.
func TestClassicCoverageLedgerRoutingIsUniqueAndComplete(t *testing.T) {
	root := findTestModuleRootForClassicCoverage199()
	routing, err := loadClassicCoverageLedgerRouting(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(routing) != 72 {
		t.Fatalf("routing has %d entries, want 72", len(routing))
	}
	for i := 1; i <= 72; i++ {
		id := fmt.Sprintf("CAP-%03d", i)
		route, ok := routing[id]
		if !ok {
			t.Errorf("routing missing %s", id)
			continue
		}
		if route.Disposition == "" {
			t.Errorf("routing for %s has no disposition", id)
		}
		if route.Phase == "" {
			t.Errorf("routing for %s has no phase", id)
		}
	}
}

// TestClassicCoveragePhase203Signed loads 203-CLASSIC-COVERAGE.json and runs
// validateClassicCoverage for phase "203", expecting no error.
func TestClassicCoveragePhase203Signed(t *testing.T) {
	root := findTestModuleRootForClassicCoverage199()
	document := loadClassicCoveragePhase203(t, root)
	if err := validateClassicCoverage(document, root, "203"); err != nil {
		t.Fatal(err)
	}
}

// TestClassicCoverageMissingFileFailsByName exercises
// classicCoveragePathForPhase's underlying, root-parameterized resolution
// against a temporary root whose coverage file the ledger routes rows to but
// is genuinely absent -- never by deleting a real file.
func TestClassicCoverageMissingFileFailsByName(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".planning", "phases", "203-biological-runtime"), 0755); err != nil {
		t.Fatal(err)
	}
	_, err := classicCoverageJSONPathInRoot(root, "203")
	if err == nil {
		t.Fatal("expected an error for a missing coverage file")
	}
	if !strings.Contains(err.Error(), "203") {
		t.Fatalf("error does not name the phase: %v", err)
	}
}

