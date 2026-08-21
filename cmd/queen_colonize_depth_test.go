package cmd

import (
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// TestColonizeLightDepthTrimsSurveyors: colonize always sent exactly four
// surveyors whatever the repo or the operator's depth choice — the one flow
// where `--verification-depth light` changed nothing at all. Light now sends
// only the structure and dependency surveyors; standard keeps all four.
func TestColonizeLightDepthTrimsSurveyors(t *testing.T) {
	light := colony.ColonyState{VerificationDepth: string(colony.VerificationDepthLight)}
	standard := colony.ColonyState{}

	trimmed := queenSurveyorSpecsForState(light)
	if len(trimmed) != 2 {
		t.Fatalf("light colonize should send exactly two surveyors, got %d: %+v", len(trimmed), trimmed)
	}
	got := map[string]bool{}
	for _, spec := range trimmed {
		got[spec.Caste] = true
	}
	if !got["surveyor-nest"] || !got["surveyor-provisions"] {
		t.Fatalf("light colonize should keep the structure and dependency surveyors, got %+v", got)
	}

	full := queenSurveyorSpecsForState(standard)
	if len(full) != 4 {
		t.Fatalf("standard colonize must keep all four surveyors, got %d: %+v", len(full), full)
	}
}
