package cmd

import (
	"bytes"
	"os"
	"testing"
)

func TestConsolidationPhaseEndPromotesEligibleObservation(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	writeTestJSON(t, s.BasePath(), "learning-observations.json", map[string]interface{}{
		"observations": []interface{}{
			map[string]interface{}{
				"content_hash":      "obs_phase_end",
				"content":           "promote this pattern found in pkg/graph/consolidate.go",
				"wisdom_type":       "pattern",
				"observation_count": 3,
				"first_seen":        "2026-04-20T10:00:00Z",
				"last_seen":         "2026-04-21T10:00:00Z",
				"colonies":          []interface{}{"test-colony"},
				"source_type":       "success_pattern",
				"evidence_type":     "single_phase",
			},
		},
	})

	rootCmd.SetArgs([]string{"consolidation-phase-end"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("consolidation-phase-end failed: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["promotion_candidates"] != float64(1) {
		t.Fatalf("promotion_candidates = %v, want 1", result["promotion_candidates"])
	}

	instincts := loadInstinctFileOrEmpty(s)
	if len(instincts.Instincts) != 1 {
		t.Fatalf("expected one promoted instinct, got %+v", instincts.Instincts)
	}
}
