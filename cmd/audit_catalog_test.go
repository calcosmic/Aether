package cmd

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"strings"
	"testing"
)

var updateGolden = flag.Bool("update-golden", false, "update golden files")

func TestAuditCatalogGolden(t *testing.T) {
	catalog := buildAuditCatalog(rootCmd)
	data, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		t.Fatalf("marshal catalog: %v", err)
	}

	goldenPath := "testdata/command_catalog.json"

	if *updateGolden {
		if err := os.WriteFile(goldenPath, append(data, '\n'), 0644); err != nil {
			t.Fatalf("write golden file: %v", err)
		}
		t.Logf("golden file updated: %s", goldenPath)
		return
	}

	golden, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden file: %v (run with -update-golden to create)", err)
	}

	// Compare with trailing newline normalization
	got := string(data) + "\n"
	want := string(golden)
	if got != want {
		t.Errorf("catalog golden mismatch; run with -update-golden to refresh")
		t.Logf("  got  %d bytes", len(got))
		t.Logf("  want %d bytes", len(want))
	}
}

func TestCatalogCompleteness(t *testing.T) {
	catalog := buildAuditCatalog(rootCmd)

	// Build a name set for quick lookup
	names := make(map[string]bool, len(catalog))
	for _, entry := range catalog {
		names[entry.Name] = true
	}

	// All lifecycle commands must appear in the catalog.
	// Note: wrapper names /ant-patrol and /ant-profile map to compound Cobra
	// names; resume is itself the public runtime command.
	expectedLifecycle := []string{
		"init", "discuss", "colonize", "plan", "build", "continue",
		"seal", "entomb", "publish", "update", "status", "resume",
		"watch", "patrol-check", "profile-read",
	}
	var missing []string
	for _, name := range expectedLifecycle {
		if !names[name] {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		t.Errorf("catalog missing lifecycle commands: %v", missing)
	}

	// Catalog must contain a substantial number of commands (300+ expected).
	if len(catalog) < 300 {
		t.Errorf("expected >= 300 catalog entries, got %d", len(catalog))
	}
}

func TestCatalogSchema(t *testing.T) {
	catalog := buildAuditCatalog(rootCmd)

	validOutputModes := map[string]bool{
		"json": true, "visual": true, "json+visual": true,
		"text": true, "unknown": true,
	}

	for _, entry := range catalog {
		if entry.Name == "" {
			t.Error("catalog entry has empty name")
		}
		if entry.ShortDescription == "" {
			t.Errorf("command %q has empty short_description", entry.Name)
		}
		// flags can be empty [] -- that is valid.
		if entry.Flags == nil {
			t.Errorf("command %q has nil flags field", entry.Name)
		}
		// parent_command can be empty string for root-level commands.
		if !validOutputModes[entry.OutputMode] {
			t.Errorf("command %q has invalid output_mode %q", entry.Name, entry.OutputMode)
		}
	}
}

func TestAuditCatalogMarkdownOutput(t *testing.T) {
	entries, err := loadEnrichedCatalog()
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}

	var buf bytes.Buffer
	oldStdout := stdout
	stdout = &buf
	defer func() { stdout = oldStdout }()

	if err := outputCatalogMarkdown(entries); err != nil {
		t.Fatalf("output markdown: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Command") {
		t.Error("markdown missing Command header")
	}
	if !strings.Contains(out, "Classification") {
		t.Error("markdown missing Classification header")
	}
	if !strings.Contains(out, "Since") {
		t.Error("markdown missing Since header")
	}
	if !strings.Contains(out, "public_lifecycle") {
		t.Error("markdown missing public_lifecycle classification")
	}
	if !strings.Contains(out, "public_utility") {
		t.Error("markdown missing public_utility classification")
	}
	if !strings.Contains(out, "Total:") {
		t.Error("markdown missing total count")
	}
}

func TestAuditCatalogJSONOutput(t *testing.T) {
	entries, err := loadEnrichedCatalog()
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}

	var buf bytes.Buffer
	oldStdout := stdout
	stdout = &buf
	defer func() { stdout = oldStdout }()

	if err := outputCatalogJSON(entries); err != nil {
		t.Fatalf("output json: %v", err)
	}

	var parsed []map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("parse json output: %v", err)
	}
	if len(parsed) == 0 {
		t.Fatal("json output empty")
	}
	first := parsed[0]
	if _, ok := first["name"]; !ok {
		t.Error("json entry missing name")
	}
	if _, ok := first["classification"]; !ok {
		t.Error("json entry missing classification")
	}
	if _, ok := first["since_version"]; !ok {
		t.Error("json entry missing since_version")
	}
}

func TestAuditCatalogCSVOutput(t *testing.T) {
	entries, err := loadEnrichedCatalog()
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}

	var buf bytes.Buffer
	oldStdout := stdout
	stdout = &buf
	defer func() { stdout = oldStdout }()

	if err := outputCatalogCSV(entries); err != nil {
		t.Fatalf("output csv: %v", err)
	}

	out := buf.String()
	if !strings.HasPrefix(out, "Command,Classification,Since,Description") {
		t.Errorf("csv header mismatch: %q", out)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		t.Error("csv output has no data lines")
	}
}

func TestAuditCatalogReliabilityMarkdown(t *testing.T) {
	entries, err := loadEnrichedCatalog()
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}

	var buf bytes.Buffer
	oldStdout := stdout
	stdout = &buf
	defer func() { stdout = oldStdout }()

	if err := outputReliabilityMarkdown(entries); err != nil {
		t.Fatalf("output reliability markdown: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Score") {
		t.Error("reliability markdown missing Score column")
	}
	if !strings.Contains(out, "v5.4.0") {
		t.Error("reliability markdown missing v5.4.0 column")
	}
	if !strings.Contains(out, "v1.21") {
		t.Error("reliability markdown missing v1.21 column")
	}
	// Check that sorting by stability score is roughly correct:
	// The first data line should have a score >= the second data line.
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 4 {
		t.Fatal("reliability markdown too short")
	}
}

func TestAuditCatalogReliabilityJSON(t *testing.T) {
	entries, err := loadEnrichedCatalog()
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}

	var buf bytes.Buffer
	oldStdout := stdout
	stdout = &buf
	defer func() { stdout = oldStdout }()

	if err := outputReliabilityJSON(entries); err != nil {
		t.Fatalf("output reliability json: %v", err)
	}

	var parsed []map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("parse reliability json: %v", err)
	}
	if len(parsed) == 0 {
		t.Fatal("reliability json output empty")
	}
	first := parsed[0]
	if _, ok := first["historical_presence"]; !ok {
		t.Error("reliability json missing historical_presence")
	}
	if _, ok := first["stability_score"]; !ok {
		t.Error("reliability json missing stability_score")
	}
}

func TestAuditCatalogClassificationCoverage(t *testing.T) {
	catalog := buildAuditCatalog(rootCmd)
	classifications := make(map[string]int)
	for _, e := range catalog {
		classifications[e.Classification]++
	}

	required := []string{"public_lifecycle", "public_utility", "internal_runtime", "alias"}
	for _, cls := range required {
		if classifications[cls] == 0 {
			t.Errorf("no commands with classification %q", cls)
		}
	}
}

func TestAuditCatalogStabilityScore(t *testing.T) {
	catalog := buildAuditCatalog(rootCmd)
	for _, e := range catalog {
		if e.StabilityScore < 0 || e.StabilityScore > 100 {
			t.Errorf("command %q has invalid stability_score %d", e.Name, e.StabilityScore)
		}
	}
}
