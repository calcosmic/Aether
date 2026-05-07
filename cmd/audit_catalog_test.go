package cmd

import (
	"encoding/json"
	"flag"
	"os"
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
	// Note: wrapper names /ant-resume, /ant-patrol, /ant-profile map to
	// compound Cobra names resume-colony, patrol-check, profile-read.
	expectedLifecycle := []string{
		"init", "discuss", "colonize", "plan", "build", "continue",
		"seal", "entomb", "publish", "update", "recover", "status",
		"resume-colony", "watch", "patrol-check", "profile-read",
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
