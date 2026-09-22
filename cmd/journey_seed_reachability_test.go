//go:build journey

package cmd

// CR-01 (207-REVIEW.md): locks requireJourneyPracticeProjectMarker
// (cmd/journey_seed.go) -- both hidden journey-seed-* commands must refuse
// to run, and must mutate nothing, against any directory that does not
// already carry the exact .journey-practice-project.json marker
// scripts/build-messy-practice-project.sh writes. This file carries the same
// `journey` build tag as cmd/journey_seed.go itself: the functions under
// test do not exist in a default, untagged build, so this test cannot exist
// there either.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestJourneySeedStaleSurveyRefusesWithoutMarker proves
// journeySeedStaleSurvey refuses -- and writes nothing at all -- against a
// directory lacking the practice-project marker.
func TestJourneySeedStaleSurveyRefusesWithoutMarker(t *testing.T) {
	dir := t.TempDir()
	before, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read fixture dir: %v", err)
	}
	if len(before) != 0 {
		t.Fatalf("test setup: expected an empty directory, found %d entries", len(before))
	}

	err = journeySeedStaleSurvey(dir)
	if err == nil {
		t.Fatal("journeySeedStaleSurvey succeeded against a directory with no .journey-practice-project.json marker -- want a refusal")
	}
	if !strings.Contains(err.Error(), journeyPracticeProjectMarkerFilename) {
		t.Fatalf("refusal error %q does not name the missing marker %q", err.Error(), journeyPracticeProjectMarkerFilename)
	}

	survey := filepath.Join(dir, ".aether", "data", "survey", "territory-snapshot.json")
	if _, statErr := os.Stat(survey); statErr == nil {
		t.Fatal("journeySeedStaleSurvey wrote a territory snapshot despite the missing marker -- it must mutate nothing")
	}
	after, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read fixture dir after refusal: %v", err)
	}
	if len(after) != 0 {
		t.Fatalf("journeySeedStaleSurvey created %d entries in a directory it refused to seed: %v", len(after), after)
	}
}

// TestJourneySeedSupersededPlanRefusesWithoutMarker proves
// journeySeedSupersededPlan refuses -- and writes no new planning or
// specification state -- against a directory lacking the practice-project
// marker, even when that directory otherwise looks like a real,
// already-initialised Aether project (a real ".aether/data" directory).
func TestJourneySeedSupersededPlanRefusesWithoutMarker(t *testing.T) {
	dir := t.TempDir()
	dataRoot := filepath.Join(dir, ".aether", "data")
	if err := os.MkdirAll(dataRoot, 0o755); err != nil {
		t.Fatalf("seed a real .aether/data directory: %v", err)
	}
	before, err := os.ReadDir(dataRoot)
	if err != nil {
		t.Fatalf("read .aether/data before the call: %v", err)
	}

	err = journeySeedSupersededPlan(dir)
	if err == nil {
		t.Fatal("journeySeedSupersededPlan succeeded against a directory with no .journey-practice-project.json marker -- want a refusal")
	}
	if !strings.Contains(err.Error(), journeyPracticeProjectMarkerFilename) {
		t.Fatalf("refusal error %q does not name the missing marker %q", err.Error(), journeyPracticeProjectMarkerFilename)
	}

	planningState := filepath.Join(dataRoot, "planning", journeyTrapSupersededPlanRunID, "stage-state.json")
	if _, statErr := os.Stat(planningState); statErr == nil {
		t.Fatal("journeySeedSupersededPlan wrote a parked planning run despite the missing marker -- it must mutate nothing")
	}
	after, err := os.ReadDir(dataRoot)
	if err != nil {
		t.Fatalf("read .aether/data after refusal: %v", err)
	}
	if len(after) != len(before) {
		t.Fatalf(".aether/data gained entries despite the refusal: before %v, after %v", before, after)
	}
}
