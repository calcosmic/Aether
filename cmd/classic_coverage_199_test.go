package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const classicCoverage199Path = ".planning/phases/199-front-door-and-classic-contract/199-CLASSIC-COVERAGE.json"

type classicCoverage199Document struct {
	Version string                  `json:"version"`
	Rows    []classicCoverage199Row `json:"rows"`
}

type classicCoverage199Row struct {
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

func TestClassicCoverage199ExactSets(t *testing.T) {
	document := loadClassicCoverage199(t)
	if err := validateClassicCoverage199ExactSets(document); err != nil {
		t.Fatal(err)
	}
}

func TestClassicCoverage199RequiredFields(t *testing.T) {
	document := loadClassicCoverage199(t)
	if err := validateClassicCoverage199RequiredFields(document); err != nil {
		t.Fatal(err)
	}
}

func TestClassicCoverage199ProofResolution(t *testing.T) {
	document := loadClassicCoverage199(t)
	if err := validateClassicCoverage199ProofResolution(document); err != nil {
		t.Fatal(err)
	}
}

func TestClassicCoverage199PassingStatus(t *testing.T) {
	document := loadClassicCoverage199(t)
	if err := validateClassicCoverage199PassingStatus(document); err != nil {
		t.Fatal(err)
	}
}

func TestClassicCoverage199RejectsInvalidRows(t *testing.T) {
	document := loadClassicCoverage199(t)
	for name, mutate := range map[string]func(*classicCoverage199Document){
		"duplicate":               func(d *classicCoverage199Document) { d.Rows = append(d.Rows, d.Rows[0]) },
		"empty modern home":       func(d *classicCoverage199Document) { d.Rows[0].ModernHome = "" },
		"placeholder":             func(d *classicCoverage199Document) { d.Rows[0].Artifact = "TODO" },
		"unsupported disposition": func(d *classicCoverage199Document) { d.Rows[0].Disposition = "unsupported" },
		"nonpassing":              func(d *classicCoverage199Document) { d.Rows[0].Status = "PENDING" },
		"nonexistent proof":       func(d *classicCoverage199Document) { d.Rows[0].Proofs = []string{"TestDoesNotExist199"} },
	} {
		t.Run(name, func(t *testing.T) {
			clone := cloneClassicCoverage199(t, document)
			mutate(&clone)
			if err := validateClassicCoverage199(clone); err == nil {
				t.Fatal("invalid coverage rows unexpectedly passed validation")
			}
		})
	}
}

func loadClassicCoverage199(t *testing.T) classicCoverage199Document {
	t.Helper()
	path := filepath.Join(findTestModuleRoot(t), filepath.FromSlash(classicCoverage199Path))
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read classic coverage: %v", err)
	}
	var document classicCoverage199Document
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatalf("decode classic coverage: %v", err)
	}
	return document
}

func cloneClassicCoverage199(t *testing.T, document classicCoverage199Document) classicCoverage199Document {
	t.Helper()
	raw, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	var clone classicCoverage199Document
	if err := json.Unmarshal(raw, &clone); err != nil {
		t.Fatal(err)
	}
	return clone
}

func validateClassicCoverage199(document classicCoverage199Document) error {
	if err := validateClassicCoverage199ExactSets(document); err != nil {
		return err
	}
	if err := validateClassicCoverage199RequiredFields(document); err != nil {
		return err
	}
	if err := validateClassicCoverage199ProofResolution(document); err != nil {
		return err
	}
	return validateClassicCoverage199PassingStatus(document)
}

func validateClassicCoverage199ExactSets(document classicCoverage199Document) error {
	expected := map[string][]string{
		"GOAL":     {"CAP-006", "CAP-007", "CAP-008", "CAP-013", "CAP-015", "CAP-016", "CAP-017", "CAP-018", "CAP-019", "CAP-020", "CAP-026", "CAP-027", "CAP-028", "CAP-032", "CAP-033", "CAP-034", "CAP-035", "CAP-036", "CAP-037", "CAP-038", "CAP-039", "CAP-040", "CAP-041", "CAP-042", "CAP-049", "CAP-050", "CAP-052", "CAP-053", "CAP-059", "CAP-060", "CAP-062", "CAP-064", "CAP-065", "CAP-068"},
		"REQ":      {"SYNTH-01", "CEC-01", "CEC-02", "CEC-04", "CEC-08", "LIFE-01", "LIFE-02", "LIFE-03", "LIFE-04", "LIFE-05", "LIFE-06", "PROOF-01"},
		"RESEARCH": {"SYN-199-01", "SYN-199-02", "SYN-199-03", "SYN-199-04", "SYN-199-05", "SYN-199-06", "SYN-199-07", "SYN-199-08", "SYN-199-09", "SYN-199-10"},
		"CONTEXT":  {"D-01", "D-02", "D-03", "D-04", "D-05", "D-06", "D-07", "D-08", "D-09", "D-10", "D-11", "D-12", "D-13", "D-14", "D-15", "D-16", "D-17"},
	}
	seen := make(map[string]map[string]int, len(expected))
	for group := range expected {
		seen[group] = map[string]int{}
	}
	for _, row := range document.Rows {
		if _, ok := seen[row.Type]; !ok {
			return fmt.Errorf("unknown coverage type %q", row.Type)
		}
		seen[row.Type][row.ID]++
	}
	for group, ids := range expected {
		if len(seen[group]) != len(ids) {
			return fmt.Errorf("%s has %d unique rows, want %d", group, len(seen[group]), len(ids))
		}
		for _, id := range ids {
			if seen[group][id] != 1 {
				return fmt.Errorf("%s/%s has %d rows, want exactly one", group, id, seen[group][id])
			}
		}
	}
	if len(document.Rows) != 73 {
		return fmt.Errorf("coverage has %d rows, want 73", len(document.Rows))
	}
	return nil
}

func validateClassicCoverage199RequiredFields(document classicCoverage199Document) error {
	for _, row := range document.Rows {
		for field, value := range map[string]string{
			"id": row.ID, "disposition": row.Disposition, "modern_home": row.ModernHome,
			"plan_task": row.PlanTask, "artifact": row.Artifact, "status": row.Status,
			"historical_evidence": row.HistoricalEvidence,
		} {
			if strings.TrimSpace(value) == "" || containsClassicCoverage199Placeholder(value) {
				return fmt.Errorf("%s/%s has invalid %s %q", row.Type, row.ID, field, value)
			}
		}
		if len(row.Proofs) == 0 {
			return fmt.Errorf("%s/%s has no proof", row.Type, row.ID)
		}
		if row.Disposition != "keep-current" && row.Disposition != "restore-modern" && row.Disposition != "replace-better" && row.Disposition != "retire-with-proof" {
			return fmt.Errorf("%s/%s has unsupported disposition %q", row.Type, row.ID, row.Disposition)
		}
	}
	return nil
}

func validateClassicCoverage199ProofResolution(document classicCoverage199Document) error {
	root := findTestModuleRootForClassicCoverage199()
	for _, row := range document.Rows {
		artifact := filepath.Join(root, filepath.FromSlash(row.Artifact))
		if info, err := os.Stat(artifact); err != nil || info.IsDir() {
			return fmt.Errorf("%s/%s artifact %q does not resolve", row.Type, row.ID, row.Artifact)
		}
		for _, proof := range row.Proofs {
			if !classicCoverage199ProofExists(root, proof) {
				return fmt.Errorf("%s/%s proof %q does not resolve", row.Type, row.ID, proof)
			}
		}
	}
	return nil
}

func validateClassicCoverage199PassingStatus(document classicCoverage199Document) error {
	for _, row := range document.Rows {
		if row.Status != "PASS" {
			return fmt.Errorf("%s/%s status is %q, want PASS", row.Type, row.ID, row.Status)
		}
	}
	return nil
}

func containsClassicCoverage199Placeholder(value string) bool {
	upper := strings.ToUpper(value)
	return strings.Contains(upper, "TODO") || strings.Contains(upper, "MISSING") || strings.Contains(upper, "UNSUPPORTED")
}

func findTestModuleRootForClassicCoverage199() string {
	root, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			return root
		}
		parent := filepath.Dir(root)
		if parent == root {
			panic("could not locate module root")
		}
		root = parent
	}
}

func classicCoverage199ProofExists(root, proof string) bool {
	if strings.HasPrefix(proof, "case:") {
		raw, err := os.ReadFile(filepath.Join(root, "cmd", "testdata", "classic-contract", "v1", "cases.json"))
		return err == nil && strings.Contains(string(raw), `"id": "`+strings.TrimPrefix(proof, "case:")+`"`)
	}
	cmdRoot := filepath.Join(root, "cmd")
	found := false
	_ = filepath.WalkDir(cmdRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil || found || entry.IsDir() || !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr == nil && strings.Contains(string(raw), "func "+proof+"(") {
			found = true
		}
		return nil
	})
	return found
}
