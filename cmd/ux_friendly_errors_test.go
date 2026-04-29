package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestFriendlyErrorPatternMatchNoColony(t *testing.T) {
	entry, ok := friendlyErrorForPattern("no colony initialized")
	if !ok {
		t.Fatal("expected match for 'no colony initialized'")
	}
	if !strings.Contains(entry.Explanation, "colony to work with") {
		t.Errorf("expected explanation to contain 'colony to work with', got: %s", entry.Explanation)
	}
	if len(entry.NextSteps) != 2 {
		t.Errorf("expected 2 next steps, got %d", len(entry.NextSteps))
	}
}

func TestFriendlyErrorPatternMatchFlag(t *testing.T) {
	_, ok := friendlyErrorForPattern("flag --phase is required")
	if !ok {
		t.Fatal("expected match for 'flag --phase is required'")
	}
}

func TestFriendlyErrorNoMatch(t *testing.T) {
	_, ok := friendlyErrorForPattern("some unknown error xyz123")
	if ok {
		t.Fatal("expected no match for unknown error")
	}
}

func TestRenderFriendlyErrorFormat(t *testing.T) {
	entry := friendlyError{
		Pattern:    "test pattern",
		Explanation: "Something went wrong with the test.",
		NextSteps:   []string{"Run `aether patrol`.", "Check your config."},
	}
	output := renderFriendlyError(entry, "test pattern error message")

	if !strings.Contains(output, "E R R O R") {
		t.Errorf("expected banner to contain 'ERROR', got: %s", output)
	}
	if !strings.Contains(output, "Something went wrong") {
		t.Errorf("expected output to contain explanation, got: %s", output)
	}
	if !strings.Contains(output, "Next steps:") {
		t.Errorf("expected output to contain 'Next steps:', got: %s", output)
	}
	if !strings.Contains(output, "  - Run `aether patrol`.") {
		t.Errorf("expected output to contain indented next step, got: %s", output)
	}
}

func TestRenderVisualErrorFriendlyPath(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	var buf bytes.Buffer
	oldStderr := stderr
	stderr = &buf
	defer func() { stderr = oldStderr }()

	outputError(1, "no colony initialized", nil)

	output := buf.String()
	if !strings.Contains(output, "colony to work with") {
		t.Errorf("expected friendly error output to contain 'colony to work with', got: %s", output)
	}
}

func TestRenderVisualErrorGenericHint(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	var buf bytes.Buffer
	oldStderr := stderr
	stderr = &buf
	defer func() { stderr = oldStderr }()

	outputError(1, "something completely unexpected", nil)

	output := buf.String()
	if !strings.Contains(output, "something completely unexpected") {
		t.Errorf("expected output to contain raw error message, got: %s", output)
	}
	if !strings.Contains(output, "aether patrol") {
		t.Errorf("expected output to contain generic hint 'aether patrol', got: %s", output)
	}
}

func TestRenderVisualErrorJSONUnchanged(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")

	var buf bytes.Buffer
	oldStderr := stderr
	stderr = &buf
	defer func() { stderr = oldStderr }()

	outputError(1, "no colony initialized", nil)

	output := strings.TrimSpace(buf.String())

	// Must be valid JSON envelope
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(output), &m); err != nil {
		t.Fatalf("expected valid JSON, got: %s", output)
	}
	if m["ok"] != false {
		t.Error("expected ok=false")
	}
	if m["error"] != "no colony initialized" {
		t.Errorf("expected raw error in JSON, got: %v", m["error"])
	}
	if m["code"] != float64(1) {
		t.Errorf("expected code=1, got: %v", m["code"])
	}
	// Must NOT contain friendly text
	if strings.Contains(output, "colony to work with") {
		t.Error("JSON output should not contain friendly error text")
	}
}

func TestFriendlyErrorCaseInsensitive(t *testing.T) {
	// Test that pattern matching is case-insensitive
	_, ok := friendlyErrorForPattern("No Colony Initialized")
	if !ok {
		t.Fatal("expected case-insensitive match for 'No Colony Initialized'")
	}
	_, ok = friendlyErrorForPattern("PERMISSION DENIED")
	if !ok {
		t.Fatal("expected case-insensitive match for 'PERMISSION DENIED'")
	}
}

func TestFriendlyErrorPatternMatchMissingFlag(t *testing.T) {
	_, ok := friendlyErrorForPattern("missing flag --phase")
	if !ok {
		t.Fatal("expected match for 'missing flag --phase'")
	}
}

func TestFriendlyErrorPatternMatchFailedLoadColony(t *testing.T) {
	entry, ok := friendlyErrorForPattern("failed to load colony state: file not found")
	if !ok {
		t.Fatal("expected match for 'failed to load colony state'")
	}
	if !strings.Contains(entry.Explanation, "could not read the colony data file") {
		t.Errorf("unexpected explanation: %s", entry.Explanation)
	}
}

func TestFriendlyErrorPatternMatchFailedInitStore(t *testing.T) {
	entry, ok := friendlyErrorForPattern("failed to initialize store: permission denied")
	if !ok {
		t.Fatal("expected match for 'failed to initialize store'")
	}
	// "failed to initialize store" is more specific and comes before "permission denied"
	// in the pattern map, so it should match first.
	if !strings.Contains(entry.Explanation, "data storage") {
		t.Errorf("expected 'failed to initialize store' match (more specific), got: %s", entry.Explanation)
	}
}

func TestFriendlyErrorPatternMatchJSONParse(t *testing.T) {
	entry, ok := friendlyErrorForPattern("json: cannot unmarshal string into Go value")
	if !ok {
		t.Fatal("expected match for JSON parse error containing 'json'")
	}
	if !strings.Contains(entry.Explanation, "corrupted") {
		t.Errorf("expected explanation to mention corruption, got: %s", entry.Explanation)
	}
}

// TestOutputErrorVisualUsesBanner verifies that visual error output uses
// the banner format (not raw text).
func TestOutputErrorVisualUsesBanner(t *testing.T) {
	// Ensure we're in visual mode (default for non-TTY bytes.Buffer will
	// use JSON path since isTerminalWriter returns false for bytes.Buffer).
	// Force visual output via env var.
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	var buf bytes.Buffer
	oldStderr := stderr
	stderr = &buf
	defer func() { stderr = oldStderr }()

	outputError(1, "test error", nil)

	output := buf.String()
	if !strings.Contains(output, "E R R O R") {
		t.Errorf("expected visual output to contain banner, got: %s", output)
	}
	if !strings.Contains(output, "aether patrol") {
		t.Errorf("expected visual output to contain generic hint, got: %s", output)
	}
}
