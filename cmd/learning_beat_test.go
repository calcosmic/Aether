package cmd

import (
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// TestCurationAntCasteIdentitiesAreDistinct pins D-06: the nine curation-ant
// castes (the eight pkg/agent/curation ants plus "curator" for the aggregate
// orchestrator line) must render as real, distinguishable colony citizens --
// not the generic "🐜"/"Ant" fallback every unmapped caste gets today.
func TestCurationAntCasteIdentitiesAreDistinct(t *testing.T) {
	t.Setenv("AETHER_FORCE_COLOR", "1")

	curationAntKeys := []string{
		"sentinel", "nurse", "critic", "herald", "janitor",
		"archivist", "librarian", "scribe", "curator",
	}

	seenEmoji := make(map[string]string, len(curationAntKeys))
	for _, key := range curationAntKeys {
		emoji := casteEmoji(key)
		if emoji == "🐜" {
			t.Errorf("casteEmoji(%q) = generic fallback 🐜, want a distinct curation-ant emoji", key)
		}
		label := casteLabel(key)
		if label == "Ant" {
			t.Errorf("casteLabel(%q) = generic fallback \"Ant\", want a distinct curation-ant label", key)
		}
		if owner, dup := seenEmoji[emoji]; dup {
			t.Errorf("casteEmoji(%q) = %q, already used by %q -- the nine curation-ant emoji must be pairwise distinct", key, emoji, owner)
		}
		seenEmoji[emoji] = key
	}
}

// TestContinueVisualAlwaysRendersLearningBeat pins D-06/D-07: a phase advance
// must render a Learning stage in exactly one of four honest states --
// populated, zero, failed, or absent -- and silence must be impossible.
func TestContinueVisualAlwaysRendersLearningBeat(t *testing.T) {
	t.Setenv("AETHER_FORCE_COLOR", "1")

	goal := "Learning beat coverage"
	state := colony.ColonyState{Version: "3.0", Goal: &goal, State: colony.StateBUILT, CurrentPhase: 1}
	phase := colony.Phase{ID: 1, Name: "Learning beat phase"}
	nextPhase := &colony.Phase{ID: 2, Name: "Next"}

	t.Run("populated", func(t *testing.T) {
		result := map[string]interface{}{
			"consolidation": map[string]interface{}{
				"ran":                  true,
				"promotion_candidates": 3,
				"queen_eligible":       1,
				"instincts_decayed":    0,
				"instincts_archived":   0,
				"observations_decayed": 0,
			},
		}
		output := renderContinueVisual(state, phase, nil, false, nextPhase, result, colony.VerificationDepthLight)
		if !strings.Contains(output, "── Learning ──") {
			t.Fatalf("expected a Learning stage marker, got:\n%s", output)
		}
		if !strings.Contains(output, "3") || !strings.Contains(output, "1") {
			t.Fatalf("expected a line naming both counts (3, 1), got:\n%s", output)
		}
	})

	t.Run("zero", func(t *testing.T) {
		result := map[string]interface{}{
			"consolidation": map[string]interface{}{
				"ran":                  true,
				"promotion_candidates": 0,
				"queen_eligible":       0,
			},
		}
		output := renderContinueVisual(state, phase, nil, false, nextPhase, result, colony.VerificationDepthLight)
		if !strings.Contains(output, "── Learning ──") {
			t.Fatalf("expected a Learning stage marker, got:\n%s", output)
		}
		if !strings.Contains(output, "colony observed nothing new this phase") {
			t.Fatalf("expected the D-07 zero-state sentence, got:\n%s", output)
		}
	})

	t.Run("failed", func(t *testing.T) {
		result := map[string]interface{}{
			"consolidation": map[string]interface{}{
				"ran":    false,
				"reason": "curation sentinel detected corrupt stores: curation: sentinel abort: corrupt stores",
			},
		}
		output := renderContinueVisual(state, phase, nil, false, nextPhase, result, colony.VerificationDepthLight)
		if !strings.Contains(output, "── Learning ──") {
			t.Fatalf("expected a Learning stage marker, got:\n%s", output)
		}
		if !strings.Contains(output, "phase advanced WITHOUT consolidation —") {
			t.Fatalf("expected the D-05 failure wording, got:\n%s", output)
		}
		if !strings.Contains(output, "curation sentinel detected corrupt stores") {
			t.Fatalf("expected the failure reason text, got:\n%s", output)
		}
	})

	t.Run("absent", func(t *testing.T) {
		result := map[string]interface{}{}
		output := renderContinueVisual(state, phase, nil, false, nextPhase, result, colony.VerificationDepthLight)
		if !strings.Contains(output, "── Learning ──") {
			t.Fatalf("expected a Learning stage marker even with no consolidation key recorded, got:\n%s", output)
		}
		if !strings.Contains(output, "no consolidation result was recorded") {
			t.Fatalf("expected an explicit no-result-recorded statement, got:\n%s", output)
		}
	})
}
