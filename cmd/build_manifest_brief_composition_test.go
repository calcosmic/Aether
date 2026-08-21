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
// brief, and active pheromone signals must reach the wrapper-spawned worker
// somehow. Sub-resolver tests alone let the composition regress silently —
// this is the failing command for the milestone's core claim, "the colony
// hears you".
//
// Phase 190 D-01/D-04 moved the delivery mechanism once already: a plan-only
// dispatch carries brief_path (a file on disk holding the composed brief)
// instead of shipping the same text inline under "brief" -- never both.
//
// Phase 190 Plan 03 (D-190-01-A) moved it again, closing the duplication D-01
// discovered but deferred: the per-dispatch brief_path file no longer embeds
// "## Pheromone Signals" at all -- the manifest-level context_capsule
// (already prepended once ahead of every dispatch by the wrapper contract)
// is now the SOLE channel. This test's steering-reaches-the-worker guarantee
// still holds; it now reads the capsule instead of the brief_path file, and
// asserts the brief_path file does NOT also carry the signal -- proving the
// duplication is gone, not just that one copy still exists.
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

	manifest, ok := result["dispatch_manifest"].(codexBuildManifest)
	if !ok {
		t.Fatalf("result.dispatch_manifest is not a codexBuildManifest: %#v", result["dispatch_manifest"])
	}
	if !strings.Contains(manifest.ContextCapsule, "## Pheromone Signals") || !strings.Contains(manifest.ContextCapsule, signal) {
		t.Fatalf("the user's REDIRECT %q did not reach the manifest-level context_capsule; the colony cannot hear steering:\n%s", signal, manifest.ContextCapsule)
	}

	basePath := store.BasePath()
	pathless := 0
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
		// 190-03 (D-190-01-A): the brief_path file must NOT also carry the
		// pheromone section any more -- the capsule (asserted above) is now
		// the sole channel for the plan-only wrapper flow. A file that still
		// carried both would mean the duplication this plan closes is still
		// live.
		if strings.Contains(string(content), "## Pheromone Signals") {
			t.Fatalf("dispatch %v's brief_path file %s still carries \"## Pheromone Signals\" — the manifest-level capsule already delivers it once; this is the duplication 190-03 closes:\n%s", d["name"], briefPath, content)
		}
	}
	if pathless > 0 {
		t.Fatalf("%d of %d plan-only dispatches carry no brief_path — wrapper-spawned workers would get a bare task line again", pathless, len(dispatches))
	}
}
