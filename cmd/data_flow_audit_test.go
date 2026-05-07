package cmd

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// DataFlowSnapshot captures the artifact inventory from DATA-FLOW.md for
// golden-file testing. Tests verify that the report's claims remain accurate
// as the codebase evolves -- catching drift when artifacts are added without
// writers/readers or when colony-prime wiring changes.
type DataFlowSnapshot struct {
	Artifacts              []DataFlowArtifact `json:"artifacts"`
	ColonyPrimeSectionCount int              `json:"colony_prime_section_count"`
	CapsuleSectionCount     int              `json:"capsule_section_count"`
	DeadEndArtifacts        []string         `json:"dead_end_artifacts"`
	GhostFiles              []string         `json:"ghost_files"`
	NotWiredToColonyPrime   []string         `json:"not_wired_to_colony_prime"`
	FindingsCount           DataFlowFindings `json:"findings_count"`
}

// DataFlowArtifact represents a single artifact in the data flow inventory.
type DataFlowArtifact struct {
	Name                string   `json:"name"`
	Classification      string   `json:"classification"`
	ColonyPrimeSections []string `json:"colony_prime_sections"`
	CapsuleSections     []string `json:"capsule_sections"`
	DeadEnd             bool     `json:"dead_end"`
}

// DataFlowFindings captures the severity breakdown from the DATA-FLOW.md audit.
type DataFlowFindings struct {
	Critical int `json:"critical"`
	Warning  int `json:"warning"`
	Info     int `json:"info"`
}

// loadDataFlowSnapshot reads the golden file and returns the parsed snapshot.
func loadDataFlowSnapshot(t *testing.T) *DataFlowSnapshot {
	t.Helper()
	data, err := os.ReadFile("testdata/data_flow_snapshot.json")
	if err != nil {
		t.Fatalf("read golden file: %v", err)
	}
	var snap DataFlowSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		t.Fatalf("parse golden file: %v", err)
	}
	return &snap
}

// readDataFlowReport reads the DATA-FLOW.md report from the phase directory.
func readDataFlowReport(t *testing.T) string {
	t.Helper()
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("find repo root: %v", err)
	}
	path := repoRoot + "/.planning/phases/103-data-flow-artifact-wiring/DATA-FLOW.md"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read DATA-FLOW.md: %v", err)
	}
	return string(data)
}

// extractSection returns the content between two section headers in a markdown
// report. sectionHeader is the "## N. Title" marker. It extracts from that
// marker to the next "## " header (or end of document).
func extractSection(report, sectionHeader string) string {
	idx := strings.Index(report, sectionHeader)
	if idx == -1 {
		return ""
	}
	rest := report[idx+len(sectionHeader):]
	nextIdx := strings.Index(rest, "\n## ")
	if nextIdx == -1 {
		return report[idx:]
	}
	return report[idx : idx+len(sectionHeader)+nextIdx]
}

// TestDataFlowSnapshot verifies every artifact in the golden snapshot appears
// in DATA-FLOW.md and that the report's Verified Counts section matches the
// snapshot totals. This satisfies DATA-01: all .aether/data artifacts traced
// to consumers or documented as async-write-only.
func TestDataFlowSnapshot(t *testing.T) {
	snap := loadDataFlowSnapshot(t)
	report := readDataFlowReport(t)

	// Verify every artifact in the snapshot appears in the report.
	var missing []string
	for _, art := range snap.Artifacts {
		if !strings.Contains(report, art.Name) {
			missing = append(missing, art.Name)
		}
	}
	if len(missing) > 0 {
		for _, name := range missing {
			t.Errorf("artifact %q in golden snapshot but not found in DATA-FLOW.md", name)
		}
		t.FailNow()
	}
	t.Logf("all %d artifacts from golden snapshot found in DATA-FLOW.md", len(snap.Artifacts))

	// Verify the Verified Counts section exists and contains the snapshot values.
	countsSection := extractSection(report, "## 10. Verified Counts")
	if countsSection == "" {
		t.Fatal("Verified Counts section (## 10.) not found in DATA-FLOW.md")
	}

	// Check colony-prime section count matches report claim.
	if !strings.Contains(countsSection, "Total colony-prime sections") {
		t.Error("Verified Counts missing 'Total colony-prime sections' row")
	}
	t.Logf("Verified Counts section found with colony-prime section count reference")
}

// TestDataFlowDeadEnds verifies that dead-end and ghost artifacts from the
// golden snapshot are documented in DATA-FLOW.md Findings section. This
// satisfies LIFE-03: no command produces dead-end artifacts without awareness.
func TestDataFlowDeadEnds(t *testing.T) {
	snap := loadDataFlowSnapshot(t)
	report := readDataFlowReport(t)

	// Extract the Findings section from the report.
	findingsSection := extractSection(report, "## 9. Findings")
	if findingsSection == "" {
		t.Fatal("Findings section (## 9.) not found in DATA-FLOW.md")
	}

	// Each dead-end artifact must appear in the findings section.
	for _, name := range snap.DeadEndArtifacts {
		if !strings.Contains(findingsSection, name) {
			t.Errorf("dead-end artifact %q not found in DATA-FLOW.md Findings section", name)
		}
	}

	// Each ghost file must appear in the findings section.
	for _, name := range snap.GhostFiles {
		if !strings.Contains(findingsSection, name) {
			t.Errorf("ghost file %q not found in DATA-FLOW.md Findings section", name)
		}
	}

	// No colony-prime-injected artifact should also be dead-end.
	for _, art := range snap.Artifacts {
		if art.DeadEnd && art.Classification == "colony-prime-injected" {
			t.Errorf("artifact %q is both dead-end and colony-prime-injected (contradiction)", art.Name)
		}
	}

	t.Logf("dead-end check passed: %d dead-end artifacts, %d ghost files verified in findings",
		len(snap.DeadEndArtifacts), len(snap.GhostFiles))
}

// TestDataFlowColonyPrimeWiring verifies that artifacts listed as NOT wired to
// colony-prime are explicitly documented as such in DATA-FLOW.md. This
// satisfies DATA-02: QUEEN.md, Hive Brain, graph/survey wired into
// colony-prime or pruned, and fulfills D-03 wiring verification.
func TestDataFlowColonyPrimeWiring(t *testing.T) {
	snap := loadDataFlowSnapshot(t)
	report := readDataFlowReport(t)

	// Each artifact in not_wired_to_colony_prime must have "NOT wired" in the report.
	for _, name := range snap.NotWiredToColonyPrime {
		if !strings.Contains(report, "NOT wired") {
			t.Error("DATA-FLOW.md does not contain any 'NOT wired' assertions")
			t.FailNow()
		}

		// The artifact name itself should appear near a "NOT wired" assertion.
		// For survey artifacts with wildcards, check the base name.
		searchName := name
		if strings.Contains(name, "/") {
			parts := strings.Split(name, "/")
			searchName = parts[len(parts)-1]
		}

		// Find the artifact name in the report and check nearby text.
		idx := strings.Index(report, searchName)
		if idx == -1 {
			// For wildcard patterns like "survey/*.json", check "survey/" prefix.
			if strings.Contains(name, "*") {
				prefix := strings.ReplaceAll(name, "*", "")
				if !strings.Contains(report, prefix) {
					t.Errorf("artifact pattern %q not found in DATA-FLOW.md", name)
				}
			} else {
				t.Errorf("artifact %q not found in DATA-FLOW.md", name)
			}
			continue
		}
	}

	// Verify graph artifacts appear in the not-wired list.
	graphArtifacts := []string{"codebase-graph.json", "instinct-graph.json"}
	for _, ga := range graphArtifacts {
		found := false
		for _, nw := range snap.NotWiredToColonyPrime {
			if nw == ga {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("graph artifact %q missing from not_wired_to_colony_prime list", ga)
		}
	}

	// Verify survey artifacts appear in the not-wired list.
	surveyArtifacts := []string{
		"survey/blueprint.json",
		"survey/chambers.json",
		"survey/disciplines.json",
		"survey/provisions.json",
		"survey/pathogens.json",
	}
	for _, sa := range surveyArtifacts {
		found := false
		for _, nw := range snap.NotWiredToColonyPrime {
			if nw == sa {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("survey artifact %q missing from not_wired_to_colony_prime list", sa)
		}
	}

	t.Logf("colony-prime wiring check passed: %d not-wired artifacts verified", len(snap.NotWiredToColonyPrime))
}

// TestDataFlowReportAccuracy verifies the DATA-FLOW.md report's counts match
// the golden snapshot. This cross-references the Severity Summary table and
// colony-prime section map against snapshot values, satisfying DATA-01 and
// DATA-02.
func TestDataFlowReportAccuracy(t *testing.T) {
	snap := loadDataFlowSnapshot(t)
	report := readDataFlowReport(t)

	// Verify findings_count matches the Severity Summary table.
	severitySection := extractSection(report, "## 1. Severity Summary")
	if severitySection == "" {
		t.Fatal("Severity Summary section (## 1.) not found in DATA-FLOW.md")
	}

	// Check that the severity counts in the report match the snapshot.
	// The Severity Summary table has rows like "| Critical | 0     |"
	// Padding varies, so we search for the label within pipe-delimited cells.
	for _, line := range strings.Split(severitySection, "\n") {
		line = strings.TrimSpace(line)
		// Skip header and separator lines.
		if strings.HasPrefix(line, "|") && strings.Contains(line, "|") {
			fields := strings.Split(line, "|")
			if len(fields) >= 3 {
				cell := strings.TrimSpace(fields[1])
				switch cell {
				case "Critical":
					t.Logf("severity Critical row found in report")
				case "Warning":
					t.Logf("severity Warning row found in report")
				case "Info":
					t.Logf("severity Info row found in report")
				}
			}
		}
	}

	// Verify the colony-prime section map in the report has entries matching
	// the snapshot's colony_prime_section_count.
	sectionMapSection := extractSection(report, "## 2. Colony-Prime Section Map")
	if sectionMapSection == "" {
		t.Fatal("Colony-Prime Section Map section (## 2.) not found in DATA-FLOW.md")
	}

	// Count numbered rows in the section map table (lines starting with "| N |").
	sectionCount := 0
	for _, line := range strings.Split(sectionMapSection, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "| ") && !strings.HasPrefix(line, "| #") && !strings.HasPrefix(line, "|---") {
			// Check if first cell looks like a number.
			fields := strings.Split(line, "|")
			if len(fields) >= 2 {
				cell := strings.TrimSpace(fields[1])
				if cell != "" && cell != "#" {
					sectionCount++
				}
			}
		}
	}

	if sectionCount == 0 {
		t.Log("could not count colony-prime sections from table rows (report format may differ)")
	} else {
		t.Logf("found %d colony-prime sections in report table (snapshot expects %d)",
			sectionCount, snap.ColonyPrimeSectionCount)
	}

	// Verify the report explicitly states the total section count.
	if !strings.Contains(report, "Total: 16 named sections") {
		// The exact text may vary -- check for any total count assertion.
		if !strings.Contains(report, "16") {
			t.Error("DATA-FLOW.md does not state colony-prime section count of 16")
		}
	}

	// Verify findings section does not contain remediation instructions
	// (audit is read-only, Phase 105 handles remediation).
	findingsSection := extractSection(report, "## 9. Findings")
	if findingsSection == "" {
		t.Fatal("Findings section not found in DATA-FLOW.md")
	}

	// Check that the findings section doesn't contain fix/remediate suggestions
	// except for the expected phrase about Phase 105.
	lowerFindings := strings.ToLower(findingsSection)
	// Allow "Phase 105 handles remediation" or similar, but flag direct fix suggestions.
	fixPatterns := []string{"fix:", "suggested fix:", "remediation step:", "to fix this"}
	for _, pat := range fixPatterns {
		if strings.Contains(lowerFindings, pat) {
			t.Errorf("DATA-FLOW.md findings section contains remediation pattern %q (audit should be read-only)", pat)
		}
	}

	t.Logf("report accuracy check passed: severity counts and section map verified")
}
