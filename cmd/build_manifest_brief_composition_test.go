package cmd

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// The keystone Phase 1 behavior, pinned at the composition level: the
// plan-only manifest every wrapper consumes must carry a real per-worker
// brief, and active pheromone signals must be inside it. Sub-resolver tests
// alone let the composition regress silently — this is the failing command
// for the milestone's core claim, "the colony hears you".
func TestPlanOnlyManifestBriefCarriesPheromones(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceBuildJSONOutput(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	goal := "Prove the manifest brief composition"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 0,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:          1,
			Name:        "Compose the brief",
			Description: "Ship the exporter wiring in cmd/exporter.go",
			Status:      colony.PhaseReady,
			Tasks: []colony.Task{
				{ID: &taskID, Goal: "Implement exporter wiring", Status: colony.TaskPending},
			},
		}}},
	})

	signal := "never-hardcode-endpoints-in-exporter"
	recent := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	pf := colony.PheromoneFile{Signals: []colony.PheromoneSignal{
		{Type: "REDIRECT", Content: json.RawMessage(`{"text":"` + signal + `"}`), Active: true, Strength: floatPtr(0.9), CreatedAt: recent},
	}}
	if err := store.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatalf("save pheromones: %v", err)
	}

	result, _, _, _, err := runCodexBuildPlanOnly(root, 1, nil)
	if err != nil {
		t.Fatalf("plan-only build: %v", err)
	}
	dispatches := result["dispatches"].([]map[string]interface{})
	if len(dispatches) == 0 {
		t.Fatal("no dispatches in plan-only result")
	}
	briefless := 0
	signalCarried := false
	for _, d := range dispatches {
		brief, _ := d["brief"].(string)
		if strings.TrimSpace(brief) == "" {
			briefless++
			continue
		}
		if strings.Contains(brief, signal) && strings.Contains(brief, "## Pheromone Signals") {
			signalCarried = true
		}
	}
	if briefless > 0 {
		t.Fatalf("%d of %d plan-only dispatches carry no brief — wrapper-spawned workers would get a bare task line again", briefless, len(dispatches))
	}
	if !signalCarried {
		t.Fatalf("the user's REDIRECT %q reached no worker brief; the colony cannot hear steering", signal)
	}
}
