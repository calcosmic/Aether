package cmd

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/exchange"
)

// TestPheromoneExportXMLHelp verifies the pheromone-export-xml command exists and shows help.
func TestPheromoneExportXMLHelp(t *testing.T) {
	saveGlobalsCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	// Capture rootCmd output
	origOut := rootCmd.OutOrStdout()
	rootCmd.SetOut(&buf)
	defer rootCmd.SetOut(origOut)

	rootCmd.SetArgs([]string{"pheromone-export-xml", "--help"})
	defer rootCmd.SetArgs([]string{})

	// --help causes a return error in cobra, which is fine
	_ = rootCmd.Execute()

	output := buf.String()
	if !strings.Contains(output, "pheromone-export-xml") {
		t.Errorf("expected help output to contain 'pheromone-export-xml', got: %s", output)
	}
}

// TestAllAliasCommandsExist verifies all 7 alias commands are registered.
func TestAllAliasCommandsExist(t *testing.T) {
	aliases := []string{
		"pheromone-export-xml",
		"pheromone-import-xml",
		"wisdom-export-xml",
		"wisdom-import-xml",
		"registry-export-xml",
		"registry-import-xml",
		"colony-archive-xml",
	}

	for _, name := range aliases {
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Use == name || strings.HasPrefix(cmd.Use, name+" ") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("alias command %q not registered in rootCmd", name)
		}
	}
}

// TestPheromoneDisplayEmpty verifies pheromone-display works with no signals.
func TestPheromoneDisplayEmpty(t *testing.T) {
	saveGlobalsCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"pheromone-display"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pheromone-display returned error: %v", err)
	}

	// Find the JSON envelope in output (may have display text before it)
	output := buf.String()
	var envelope map[string]interface{}
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "{") {
			if err := json.Unmarshal([]byte(line), &envelope); err == nil {
				break
			}
		}
	}

	if envelope == nil {
		t.Fatalf("no JSON envelope found in output: %q", output)
	}

	if ok, _ := envelope["ok"].(bool); !ok {
		t.Errorf("expected ok:true, got: %v", envelope)
	}

	result, _ := envelope["result"].(map[string]interface{})
	count, _ := result["count"].(float64)
	if count != 0 {
		t.Errorf("expected count 0 for empty pheromones, got: %v", count)
	}
}

// TestPheromoneDisplayWithSignals verifies pheromone-display shows signals in a table.
func TestPheromoneDisplayWithSignals(t *testing.T) {
	saveGlobalsCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Write test pheromone signals
	content, _ := json.Marshal(map[string]string{"text": "Focus on testing"})
	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{
				ID:       "sig-1",
				Type:     "FOCUS",
				Priority: "normal",
				Active:   true,
				Content:  content,
			},
			{
				ID:       "sig-2",
				Type:     "REDIRECT",
				Priority: "high",
				Active:   false,
				Content:  content,
			},
		},
	}
	if err := s.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"pheromone-display"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pheromone-display returned error: %v", err)
	}

	output := buf.String()

	// Should contain the table header
	if !strings.Contains(output, "TYPE") || !strings.Contains(output, "CONTENT") {
		t.Errorf("expected table header in output, got: %s", output)
	}

	// Should show the active FOCUS signal
	if !strings.Contains(output, "FOCUS") {
		t.Errorf("expected FOCUS signal in output, got: %s", output)
	}

	// Should NOT show inactive REDIRECT (active-only defaults true)
	if strings.Contains(output, "REDIRECT") {
		t.Errorf("expected REDIRECT to be filtered out (inactive), got: %s", output)
	}
}

// --- colony-archive-xml tests (TDD RED phase) ---

// TestColonyArchiveXMLOutputFlag tests that colony-archive-xml --output writes
// a valid colony-archive XML file containing pheromone data loaded from the store.
func TestColonyArchiveXMLOutputFlag(t *testing.T) {
	saveGlobalsCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Seed the store with pheromone data so the archive has content.
	content, _ := json.Marshal(map[string]string{"text": "Focus on archive testing"})
	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{
				ID:       "sig-archive-1",
				Type:     "FOCUS",
				Priority: "normal",
				Source:   "user",
				CreatedAt: "2026-04-06T10:00:00Z",
				Active:   true,
				Content:  content,
			},
		},
	}
	if err := s.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatal(err)
	}

	outputPath := filepath.Join(tmpDir, "archive.xml")

	rootCmd.SetArgs([]string{"colony-archive-xml", "--output", outputPath})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("colony-archive-xml returned error: %v", err)
	}

	// The output file must exist.
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("output file not created: %v", err)
	}

	output := string(data)

	// Must have an XML header.
	if !strings.HasPrefix(output, "<?xml") {
		t.Errorf("output missing XML header, got:\n%s", output)
	}

	// Must be parseable as XML.
	var archive exchange.ColonyArchiveXML
	if err := xml.Unmarshal(data, &archive); err != nil {
		t.Fatalf("output is not valid colony-archive XML: %v\noutput:\n%s", err, output)
	}

	// Must contain the pheromone data we seeded.
	if archive.Pheromones == nil {
		t.Fatal("archive missing pheromones section")
	}
	if archive.Pheromones.Count != 1 {
		t.Errorf("pheromones.count = %d, want 1", archive.Pheromones.Count)
	}
	if len(archive.Pheromones.Signals) != 1 || archive.Pheromones.Signals[0].ID != "sig-archive-1" {
		t.Errorf("expected pheromone signal sig-archive-1, got: %v", archive.Pheromones.Signals)
	}
}

// TestColonyArchiveXMLPositionalArg tests backward compatibility: passing the
// output path as a positional argument instead of --output flag.
func TestColonyArchiveXMLPositionalArg(t *testing.T) {
	saveGlobalsCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Seed minimal colony data.
	content, _ := json.Marshal(map[string]string{"text": "positional test"})
	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{
				ID:       "sig-pos-1",
				Type:     "REDIRECT",
				Priority: "high",
				Source:   "user",
				CreatedAt: "2026-04-06T11:00:00Z",
				Active:   true,
				Content:  content,
			},
		},
	}
	if err := s.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatal(err)
	}

	outputPath := filepath.Join(tmpDir, "archive-pos.xml")

	rootCmd.SetArgs([]string{"colony-archive-xml", outputPath})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("colony-archive-xml with positional arg returned error: %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("output file not created: %v", err)
	}

	output := string(data)

	// Must have XML header.
	if !strings.HasPrefix(output, "<?xml") {
		t.Errorf("output missing XML header, got:\n%s", output)
	}

	// Must parse as valid colony-archive XML.
	var archive exchange.ColonyArchiveXML
	if err := xml.Unmarshal(data, &archive); err != nil {
		t.Fatalf("output is not valid colony-archive XML: %v\noutput:\n%s", err, output)
	}

	// Must contain our seeded pheromone.
	if archive.Pheromones == nil {
		t.Fatal("archive missing pheromones section")
	}
	if archive.Pheromones.Count != 1 {
		t.Errorf("pheromones.count = %d, want 1", archive.Pheromones.Count)
	}
	if len(archive.Pheromones.Signals) != 1 || archive.Pheromones.Signals[0].ID != "sig-pos-1" {
		t.Errorf("expected pheromone signal sig-pos-1, got: %v", archive.Pheromones.Signals)
	}
}

// TestColonyArchiveXMLHasAllSections verifies the archive contains pheromones,
// queen-wisdom, and colony-registry sections even when some are empty.
func TestColonyArchiveXMLHasAllSections(t *testing.T) {
	saveGlobalsCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	outputPath := filepath.Join(tmpDir, "archive-sections.xml")

	rootCmd.SetArgs([]string{"colony-archive-xml", "--output", outputPath})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("colony-archive-xml returned error: %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("output file not created: %v", err)
	}

	var archive exchange.ColonyArchiveXML
	if err := xml.Unmarshal(data, &archive); err != nil {
		t.Fatalf("output is not valid colony-archive XML: %v\noutput:\n%s", err, string(data))
	}

	// All three sections must be present (even if empty).
	if archive.Pheromones == nil {
		t.Error("archive missing pheromones section")
	}
	if archive.Wisdom == nil {
		t.Error("archive missing queen-wisdom section")
	}
	if archive.Registry == nil {
		t.Error("archive missing colony-registry section")
	}

	// Must have version attribute.
	if archive.Version == "" {
		t.Error("archive missing version attribute")
	}
}
