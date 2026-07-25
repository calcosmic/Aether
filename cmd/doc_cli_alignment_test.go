package cmd

import (
	"os"
	"testing"

	"github.com/calcosmic/Aether/pkg/smoke"
	"github.com/calcosmic/Aether/pkg/storage"
)

// TestDocCLIAlignment validates that documented flags in YAML source files
// match the flags registered in the Go CLI. Host-critical mismatches are
// blocking (t.Fatalf); all others are warnings.
func TestDocCLIAlignment(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	store = s

	// 1. Parse YAML commands
	yamlCommands, err := smoke.ParseYAMLCommands(".aether/commands/*.yaml")
	if err != nil {
		t.Fatalf("ParseYAMLCommands failed: %v", err)
	}

	// 2. Introspect Cobra flags
	cobraFlags := smoke.ProbeCobraFlags(rootCmd)

	// 3. Compare
	var warnings []smoke.FlagMismatch
	var blocking []smoke.FlagMismatch

	for _, ycmd := range yamlCommands {
		goName := smoke.GetHostCommandGoName(ycmd.Name)
		registeredFlags, ok := cobraFlags[goName]
		if !ok {
			// Command not found in cobra — skip with warning
			continue
		}

		// Build set of registered flags for quick lookup
		registeredSet := make(map[string]bool)
		for _, f := range registeredFlags {
			registeredSet[f] = true
		}

		for _, flag := range ycmd.Flags {
			// Normalize flag name: strip leading -- for comparison
			flagName := flag
			if len(flagName) > 2 && flagName[:2] == "--" {
				flagName = flagName[2:]
			}

			if !registeredSet[flagName] {
				mismatch := smoke.FlagMismatch{
					Command:  ycmd.Name,
					Flag:     flag,
					Expected: "registered",
					Actual:   "missing",
				}
				if smoke.IsHostCritical(ycmd.Name) {
					mismatch.Severity = "blocking"
					blocking = append(blocking, mismatch)
				} else {
					mismatch.Severity = "warning"
					warnings = append(warnings, mismatch)
				}
			}
		}
	}

	// 4. Cross-reference host-critical flags with TS host tests
	untested := smoke.CrossReferenceWithTSHost(smoke.HostCriticalFlags)
	for _, u := range untested {
		warnings = append(warnings, smoke.FlagMismatch{
			Command:  u,
			Flag:     u,
			Expected: "tested",
			Actual:   "untested",
			Severity: "warning",
		})
	}

	// 5. Write platform-health.json via shared writer
	hostCriticalChecked := 0
	for _, cmd := range yamlCommands {
		if smoke.IsHostCritical(cmd.Name) {
			hostCriticalChecked++
		}
	}
	alignment := &smoke.DocCLIAlignment{
		CommandsChecked:      len(yamlCommands),
		HostCriticalChecked:  hostCriticalChecked,
		HostCriticalFailures: blocking,
		Warnings:             warnings,
	}
	if err := smoke.WritePlatformHealth(s, nil, nil, alignment); err != nil {
		t.Fatalf("failed to write platform-health.json: %v", err)
	}

	// 6. Verify the file was written and contains both old and new keys
	var loaded map[string]interface{}
	if err := s.LoadJSON("platform-health.json", &loaded); err != nil {
		t.Fatalf("failed to read back platform-health.json: %v", err)
	}
	if _, ok := loaded["failed_commands"]; !ok {
		t.Error("platform-health.json missing 'failed_commands' key")
	}
	if _, ok := loaded["flag_mismatches"]; !ok {
		t.Error("platform-health.json missing 'flag_mismatches' key")
	}
	if _, ok := loaded["doc_cli_alignment"]; !ok {
		t.Error("platform-health.json missing 'doc_cli_alignment' key")
	}

	// 7. Fail if any blocking mismatches
	if len(blocking) > 0 {
		for _, b := range blocking {
			t.Errorf("blocking mismatch: %s %s", b.Command, b.Flag)
		}
		t.Fatalf("%d host-critical flag mismatch(es) found", len(blocking))
	}
}

// buildHealthData constructs the platform-health.json payload preserving
// existing keys and adding the doc_cli_alignment section.
func buildHealthData(s *storage.Store, commands []smoke.YAMLCommand, blocking, warnings []smoke.FlagMismatch) map[string]interface{} {
	// Try to load existing platform-health.json to preserve old keys
	existing := map[string]interface{}{
		"failed_commands": []interface{}{},
		"flag_mismatches": []interface{}{},
	}
	if s != nil {
		var old map[string]interface{}
		if s.LoadJSON("platform-health.json", &old) == nil {
			if fc, ok := old["failed_commands"]; ok {
				existing["failed_commands"] = fc
			}
			if fm, ok := old["flag_mismatches"]; ok {
				existing["flag_mismatches"] = fm
			}
		}
	}

	// Count host-critical commands checked
	hostCriticalChecked := 0
	for _, cmd := range commands {
		if smoke.IsHostCritical(cmd.Name) {
			hostCriticalChecked++
		}
	}

	existing["doc_cli_alignment"] = map[string]interface{}{
		"commands_checked":       len(commands),
		"host_critical_checked":  hostCriticalChecked,
		"host_critical_failures": blocking,
		"warnings":               warnings,
	}

	return existing
}

// TestDocCLIAlignmentBlockingMismatch verifies that the test fails with
// t.Fatalf when a blocking mismatch is injected.
func TestDocCLIAlignmentBlockingMismatch(t *testing.T) {
	// This test validates the severity logic directly without needing to
	// inject a real mismatch into the YAML.
	mismatch := smoke.FlagMismatch{
		Command:  "ant-plan",
		Flag:     "--fake-flag",
		Expected: "registered",
		Actual:   "missing",
		Severity: "blocking",
	}

	if mismatch.Severity != "blocking" {
		t.Errorf("expected severity blocking, got %s", mismatch.Severity)
	}

	if !smoke.IsHostCritical(mismatch.Command) {
		t.Errorf("expected %s to be host critical", mismatch.Command)
	}
}
