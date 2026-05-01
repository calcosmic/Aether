package cmd

import (
	"bytes"
	"testing"
)

// runMemoryCapture is a test helper that executes memory-capture with given args
// and returns the parsed JSON envelope.
func runMemoryCapture(t *testing.T, args ...string) map[string]interface{} {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, _ := newTestStore(t)
	store = s

	allArgs := append([]string{"memory-capture"}, args...)
	rootCmd.SetArgs(allArgs)
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("memory-capture failed: %v", err)
	}

	return parseEnvelope(t, buf.String())
}

func TestMemoryCaptureDefaultTrustScore(t *testing.T) {
	env := runMemoryCapture(t, "test default trust scoring")

	if env["ok"] != true {
		t.Fatalf("expected ok:true, got %v", env["ok"])
	}
	data, ok := env["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected data object, got %T", env["result"])
	}

	// Default: source_type=observation(0.6), evidence_type=anecdotal(0.4), days=0(activity=1.0)
	// Score = 0.4*0.6 + 0.35*0.4 + 0.25*1.0 = 0.63
	score, _ := data["trust_score"].(float64)
	if score < 0.62 || score > 0.64 {
		t.Fatalf("expected default trust_score ~0.63, got %v", score)
	}
}

func TestMemoryCaptureExplicitFlagsHigherScore(t *testing.T) {
	env := runMemoryCapture(t, "test explicit flags",
		"--source-type", "success_pattern",
		"--evidence-type", "multi_phase")

	data, ok := env["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected data object, got %T", env["result"])
	}

	// success_pattern=0.8, multi_phase=0.9, days=0(activity=1.0)
	// Score = 0.4*0.8 + 0.35*0.9 + 0.25*1.0 = 0.885
	score, _ := data["trust_score"].(float64)
	if score < 0.88 || score > 0.89 {
		t.Fatalf("expected trust_score ~0.885 with explicit flags, got %v", score)
	}

	// Must be higher than default 0.63
	if score <= 0.63 {
		t.Fatalf("explicit flags should produce higher score than default, got %v (default is 0.63)", score)
	}
}

func TestMemoryCaptureHighestTrustScore(t *testing.T) {
	env := runMemoryCapture(t, "test highest possible score",
		"--source-type", "user_feedback",
		"--evidence-type", "test_verified")

	data, ok := env["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected data object, got %T", env["result"])
	}

	// user_feedback=1.0, test_verified=1.0, days=0(activity=1.0)
	// Score = 0.4*1.0 + 0.35*1.0 + 0.25*1.0 = 1.0
	score, _ := data["trust_score"].(float64)
	if score < 0.99 {
		t.Fatalf("expected trust_score ~1.0 with max flags, got %v", score)
	}

	// Must be the highest possible score
	if score <= 0.885 {
		t.Fatalf("user_feedback/test_verified should produce highest score, got %v", score)
	}
}

func TestMemoryCaptureOutputFields(t *testing.T) {
	env := runMemoryCapture(t, "test output fields",
		"--source-type", "error_resolution",
		"--evidence-type", "single_phase")

	data, ok := env["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected data object, got %T", env["result"])
	}

	// Verify required output fields exist
	if data["captured"] != true {
		t.Fatalf("expected captured:true, got %v", data["captured"])
	}
	if data["is_new"] != true {
		t.Fatalf("expected is_new:true for first observation, got %v", data["is_new"])
	}
	score, _ := data["trust_score"].(float64)
	if score <= 0 {
		t.Fatalf("expected positive trust_score, got %v", score)
	}
}
