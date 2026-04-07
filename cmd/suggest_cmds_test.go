package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// setupSuggestTest creates a temp directory with store initialized for testing.
func setupSuggestTest(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("failed to create temp data dir: %v", err)
	}
	os.Setenv("AETHER_ROOT", tmpDir)
	t.Cleanup(func() { os.Unsetenv("AETHER_ROOT") })
	return dataDir
}

// parseDeprecatedResult checks that the output is a valid JSON envelope with
// ok:true and deprecated:true.
func parseDeprecatedResult(t *testing.T, output string) map[string]interface{} {
	t.Helper()
	var env map[string]interface{}
	if err := json.Unmarshal([]byte(output), &env); err != nil {
		t.Fatalf("invalid JSON output: %v\noutput: %s", err, output)
	}
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}
	return env
}

// TestSuggestAnalyzeDeprecated tests that suggest-analyze returns deprecated response.
func TestSuggestAnalyzeDeprecated(t *testing.T) {
	_ = setupSuggestTest(t)
	store = nil
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	defer func() {
		stdout = os.Stdout
		stderr = os.Stderr
	}()

	rootCmd.SetArgs([]string{"suggest-analyze", "--max", "3"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("suggest-analyze returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	env := parseDeprecatedResult(t, output)

	result := env["result"].(map[string]interface{})
	if result["deprecated"] != true {
		t.Errorf("expected deprecated:true, got: %v", result["deprecated"])
	}
	if result["command"] != "suggest-analyze" {
		t.Errorf("expected command:suggest-analyze, got: %v", result["command"])
	}
	if result["message"] != deprecatedMessage {
		t.Errorf("expected deprecation message, got: %v", result["message"])
	}
}

// TestSuggestRecordDeprecated tests that suggest-record returns deprecated response.
func TestSuggestRecordDeprecated(t *testing.T) {
	_ = setupSuggestTest(t)
	store = nil
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	defer func() {
		stdout = os.Stdout
		stderr = os.Stderr
	}()

	rootCmd.SetArgs([]string{"suggest-record", "--content", "test"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("suggest-record returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	env := parseDeprecatedResult(t, output)

	result := env["result"].(map[string]interface{})
	if result["deprecated"] != true {
		t.Errorf("expected deprecated:true, got: %v", result["deprecated"])
	}
	if result["command"] != "suggest-record" {
		t.Errorf("expected command:suggest-record, got: %v", result["command"])
	}
}

// TestSuggestCheckDeprecated tests that suggest-check returns deprecated response.
func TestSuggestCheckDeprecated(t *testing.T) {
	_ = setupSuggestTest(t)
	store = nil
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	defer func() {
		stdout = os.Stdout
		stderr = os.Stderr
	}()

	rootCmd.SetArgs([]string{"suggest-check"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("suggest-check returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	env := parseDeprecatedResult(t, output)

	result := env["result"].(map[string]interface{})
	if result["deprecated"] != true {
		t.Errorf("expected deprecated:true, got: %v", result["deprecated"])
	}
	if result["command"] != "suggest-check" {
		t.Errorf("expected command:suggest-check, got: %v", result["command"])
	}
}

// TestSuggestApproveDeprecated tests that suggest-approve returns deprecated response.
func TestSuggestApproveDeprecated(t *testing.T) {
	_ = setupSuggestTest(t)
	store = nil
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	defer func() {
		stdout = os.Stdout
		stderr = os.Stderr
	}()

	rootCmd.SetArgs([]string{"suggest-approve"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("suggest-approve returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	env := parseDeprecatedResult(t, output)

	result := env["result"].(map[string]interface{})
	if result["deprecated"] != true {
		t.Errorf("expected deprecated:true, got: %v", result["deprecated"])
	}
	if result["command"] != "suggest-approve" {
		t.Errorf("expected command:suggest-approve, got: %v", result["command"])
	}
}

// TestSuggestQuickDismissDeprecated tests that suggest-quick-dismiss returns deprecated response.
func TestSuggestQuickDismissDeprecated(t *testing.T) {
	_ = setupSuggestTest(t)
	store = nil
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	defer func() {
		stdout = os.Stdout
		stderr = os.Stderr
	}()

	rootCmd.SetArgs([]string{"suggest-quick-dismiss"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("suggest-quick-dismiss returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	env := parseDeprecatedResult(t, output)

	result := env["result"].(map[string]interface{})
	if result["deprecated"] != true {
		t.Errorf("expected deprecated:true, got: %v", result["deprecated"])
	}
	if result["command"] != "suggest-quick-dismiss" {
		t.Errorf("expected command:suggest-quick-dismiss, got: %v", result["command"])
	}
}

// TestSuggestCommandsRegistered verifies all 5 commands are registered.
func TestSuggestCommandsRegistered(t *testing.T) {
	expectedCommands := []string{
		"suggest-analyze",
		"suggest-record",
		"suggest-check",
		"suggest-approve",
		"suggest-quick-dismiss",
	}

	for _, name := range expectedCommands {
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Use == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("command %q not registered in rootCmd", name)
		}
	}
}

// TestSuggestApproveDeprecatedStillHasFlags tests that deprecated suggest-approve
// still accepts its original flags (for backward compatibility with callers).
func TestSuggestApproveDeprecatedStillHasFlags(t *testing.T) {
	_ = setupSuggestTest(t)
	store = nil
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	defer func() {
		stdout = os.Stdout
		stderr = os.Stderr
	}()

	rootCmd.SetArgs([]string{"suggest-approve", "--id", "some-id", "--type", "FOCUS"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("suggest-approve with flags returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true with flags, got: %s", output)
	}
}

// TestSuggestDeprecatedSilenceUsage tests that deprecated commands suppress usage output.
func TestSuggestDeprecatedSilenceUsage(t *testing.T) {
	_ = setupSuggestTest(t)
	store = nil
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	defer func() {
		stdout = os.Stdout
		stderr = os.Stderr
	}()

	// Call with no required args to trigger usage error, but deprecated cmds should silence it
	rootCmd.SetArgs([]string{"suggest-analyze", "--unknown-flag"})
	defer rootCmd.SetArgs([]string{})

	// Should not error even with unknown flag (flags are registered but ignored)
	_ = rootCmd.Execute()
}
