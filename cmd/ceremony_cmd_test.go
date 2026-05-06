package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestCeremonySpawnPlanRendersOldStyleManifest(t *testing.T) {
	manifestFile := writeCeremonyTestJSON(t, map[string]interface{}{
		"ok": true,
		"result": map[string]interface{}{
			"dispatch_manifest": ceremonyTestManifest(),
		},
	})

	result, visual, err := renderCeremonySpawnPlanFromFile("build", manifestFile)
	if err != nil {
		t.Fatalf("render spawn plan: %v", err)
	}

	if got := intValue(result["dispatch_count"]); got != 3 {
		t.Fatalf("dispatch_count = %d, want 3", got)
	}
	for _, want := range []string{
		"S P A W N   P L A N",
		"Phase 2: Card Redesign",
		"Wave 1",
		"Builder",
		"Brick-79",
		"Watcher",
		"Watch-64",
		"Total:",
	} {
		if !strings.Contains(visual, want) {
			t.Fatalf("spawn plan missing %q\n%s", want, visual)
		}
	}
}

func TestCeremonyWaveStartRendersCasteBanner(t *testing.T) {
	manifestFile := writeCeremonyTestJSON(t, map[string]interface{}{
		"ok": true,
		"result": map[string]interface{}{
			"dispatch_manifest": ceremonyTestManifest(),
		},
	})

	result, visual, err := renderCeremonyWaveStartFromFile("build", manifestFile, 11)
	if err != nil {
		t.Fatalf("render wave start: %v", err)
	}

	if got := intValue(result["dispatch_count"]); got != 2 {
		t.Fatalf("dispatch_count = %d, want 2", got)
	}
	for _, want := range []string{
		"Spawning 2 Builders",
		"parallel",
		"Brick-79",
		"Mason-41",
	} {
		if !strings.Contains(visual, want) {
			t.Fatalf("wave start missing %q\n%s", want, visual)
		}
	}
}

func TestCeremonyWorkerCompleteRendersLine(t *testing.T) {
	workerFile := writeCeremonyTestJSON(t, map[string]interface{}{
		"name":       "Brick-79",
		"caste":      "builder",
		"status":     "completed",
		"task_id":    "2.1",
		"summary":    "Finished CardNode",
		"tool_count": 18,
	})

	result, visual, err := renderCeremonyWorkerCompleteFromFile("build", workerFile)
	if err != nil {
		t.Fatalf("render worker complete: %v", err)
	}

	if got := stringValue(result["name"]); got != "Brick-79" {
		t.Fatalf("name = %q, want Brick-79", got)
	}
	for _, want := range []string{"Brick-79", "Builder", "Finished CardNode", "18 tools"} {
		if !strings.Contains(visual, want) {
			t.Fatalf("worker complete missing %q\n%s", want, visual)
		}
	}
}

func TestCeremonyCloseoutRendersOldStyleSummary(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "Restore ceremony"
	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateBUILT,
		CurrentPhase: 2,
		Milestone:    "Open Chambers",
		Plan: colony.Plan{Phases: []colony.Phase{
			{ID: 1, Name: "Foundation"},
			{ID: 2, Name: "Card Redesign"},
		}},
	}
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save colony state: %v", err)
	}

	completionFile := writeCeremonyTestJSON(t, map[string]interface{}{
		"dispatch_manifest": ceremonyTestManifest(),
		"dispatches": []map[string]interface{}{
			{
				"name":           "Brick-79",
				"caste":          "builder",
				"status":         "completed",
				"summary":        "Finished CardNode",
				"files_modified": []string{"dashboard/components/CardNode.tsx"},
				"tool_count":     18,
			},
			{
				"name":       "Watch-64",
				"caste":      "watcher",
				"status":     "completed",
				"summary":    "Verified the phase",
				"tool_count": 7,
			},
		},
	})

	_, visual := renderCeremonyCloseout("build", completionFile)
	for _, want := range []string{
		"B U I L D   S U M M A R Y",
		"Goal: Restore ceremony",
		"Phase: 2 - Card Redesign",
		"Workers: 2 completed",
		"Worker Results",
		"Brick-79",
		"dashboard/components/CardNode.tsx",
	} {
		if !strings.Contains(visual, want) {
			t.Fatalf("closeout missing %q\n%s", want, visual)
		}
	}
}

func ceremonyTestManifest() map[string]interface{} {
	return map[string]interface{}{
		"phase":      2,
		"phase_name": "Card Redesign",
		"execution_plan": []map[string]interface{}{
			{"execution_wave": 11, "stage": "wave", "wave": 1, "strategy": "parallel", "worker_count": 2},
			{"execution_wave": 12, "stage": "verification", "strategy": "serial", "worker_count": 1},
		},
		"dispatches": []map[string]interface{}{
			{"name": "Brick-79", "caste": "builder", "task_id": "2.1", "task": "CardNode wrapper", "execution_wave": 11, "wave": 1},
			{"name": "Mason-41", "caste": "builder", "task_id": "2.2", "task": "Stats widgets", "execution_wave": 11, "wave": 1},
			{"name": "Watch-64", "caste": "watcher", "task_id": "verify", "task": "Independent verification", "execution_wave": 12, "stage": "verification"},
		},
	}
}

func writeCeremonyTestJSON(t *testing.T, value interface{}) string {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("marshal test JSON: %v", err)
	}
	path := filepath.Join(t.TempDir(), "packet.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}
