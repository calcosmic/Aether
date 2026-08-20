package cmd

import (
	"encoding/json"
	"os"
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
//
// Phase 190 D-01/D-04 moved the delivery mechanism: a plan-only dispatch now
// carries brief_path (a file on disk holding the composed brief) instead of
// shipping the same text inline under "brief" -- never both. This test's
// steering-reaches-the-worker guarantee still holds, it just now reads the
// file brief_path names instead of the inline field.
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
	basePath := store.BasePath()
	pathless := 0
	signalCarried := false
	for _, d := range dispatches {
		if brief, ok := d["brief"].(string); ok && strings.TrimSpace(brief) != "" {
			t.Fatalf("dispatch %v still carries an inline brief once brief_path succeeded — the same bytes must not ship twice", d["name"])
		}
		briefPath, _ := d["brief_path"].(string)
		if strings.TrimSpace(briefPath) == "" {
			pathless++
			continue
		}
		rel := strings.TrimPrefix(briefPath, ".aether/data/")
		content, err := os.ReadFile(filepath.Join(basePath, rel))
		if err != nil {
			t.Fatalf("brief_path %s for dispatch %v does not resolve to a file on disk: %v", briefPath, d["name"], err)
		}
		if strings.Contains(string(content), signal) && strings.Contains(string(content), "## Pheromone Signals") {
			signalCarried = true
		}
	}
	if pathless > 0 {
		t.Fatalf("%d of %d plan-only dispatches carry no brief_path — wrapper-spawned workers would get a bare task line again", pathless, len(dispatches))
	}
	if !signalCarried {
		t.Fatalf("the user's REDIRECT %q reached no worker brief file; the colony cannot hear steering", signal)
	}
}
