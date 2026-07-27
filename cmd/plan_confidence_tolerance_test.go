package cmd

import (
	"encoding/json"
	"testing"
)

// The M4L plan run of 27 July 2026 produced a complete 45KB phase-plan.json
// that was discarded because the Route-Setter wrote "knowledge": 0.85 (0-1
// scale) where Go declared int (0-100 scale). Same failure class as the
// scout payload that morning: one loose field must not cost the whole
// artifact. Fractional confidences are normalized to the 0-100 scale.
func TestPlanConfidenceAcceptsFractionalScale(t *testing.T) {
	var conf codexPlanConfidence
	payload := `{"knowledge":0.85,"requirements":0.9,"risks":0.7,"dependencies":1.0,"effort":0.75,"overall":0.85}`
	if err := json.Unmarshal([]byte(payload), &conf); err != nil {
		t.Fatalf("fractional confidence rejected: %v", err)
	}
	if conf.Knowledge != 85 || conf.Overall != 85 {
		t.Fatalf("fractional scale not normalized: knowledge=%d overall=%d, want 85/85", conf.Knowledge, conf.Overall)
	}
	if conf.Dependencies != 100 {
		t.Fatalf("1.0 should mean 100, got %d", conf.Dependencies)
	}
}

func TestPlanConfidenceAcceptsIntegerScale(t *testing.T) {
	var conf codexPlanConfidence
	payload := `{"knowledge":85,"requirements":90,"risks":70,"dependencies":100,"effort":75,"overall":85}`
	if err := json.Unmarshal([]byte(payload), &conf); err != nil {
		t.Fatalf("integer confidence rejected: %v", err)
	}
	if conf.Knowledge != 85 || conf.Dependencies != 100 {
		t.Fatalf("integer scale altered: knowledge=%d dependencies=%d", conf.Knowledge, conf.Dependencies)
	}
}

// A bare integer 1 (no decimal point) is a literal 1 on the 0-100 scale, not
// a 0-1 fraction; only values written with a fractional form are rescaled.
func TestPlanConfidenceLiteralOneStaysOne(t *testing.T) {
	var conf codexPlanConfidence
	if err := json.Unmarshal([]byte(`{"overall":1}`), &conf); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if conf.Overall != 1 {
		t.Fatalf("literal 1 became %d", conf.Overall)
	}
	var conf2 codexPlanConfidence
	if err := json.Unmarshal([]byte(`{"overall":1.0}`), &conf2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if conf2.Overall != 100 {
		t.Fatalf("1.0 should mean 100, got %d", conf2.Overall)
	}
}

// Tolerance must not become silence: non-numeric garbage still errors.
func TestPlanConfidenceRejectsGarbage(t *testing.T) {
	var conf codexPlanConfidence
	if err := json.Unmarshal([]byte(`{"overall":"high"}`), &conf); err == nil {
		t.Fatal("expected an error for a non-numeric confidence")
	}
}
