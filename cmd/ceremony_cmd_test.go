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

func TestCeremonySpawnPlanAcceptsDirectManifestPacket(t *testing.T) {
	manifestFile := writeCeremonyTestJSON(t, map[string]interface{}{
		"dispatch_manifest": ceremonyTestManifest(),
	})

	result, visual, err := renderCeremonySpawnPlanFromFile("build", manifestFile)
	if err != nil {
		t.Fatalf("render spawn plan: %v", err)
	}

	if got := stringValue(result["manifest"]); got != "dispatch_manifest" {
		t.Fatalf("manifest key = %q, want dispatch_manifest", got)
	}
	for _, want := range []string{"S P A W N   P L A N", "Brick-79", "Watch-64"} {
		if !strings.Contains(visual, want) {
			t.Fatalf("direct manifest spawn plan missing %q\n%s", want, visual)
		}
	}
}

func TestCeremonySpawnPlanExplainsQueenSpawnBudget(t *testing.T) {
	manifest := ceremonyTestManifest()
	manifest["queen_execution_policy"] = map[string]interface{}{
		"spawn_budget": map[string]interface{}{
			"worker_count":        3,
			"selected_castes":     2,
			"max_selected_castes": 4,
			"reason":              "standard build",
			"required_castes":     []string{"builder", "watcher"},
			"skipped_castes":      []string{"architect"},
			"pruned_reasons":      map[string]string{"architect": "not sent -- this phase's team is capped at 4 workers (standard build), and this pick did not make the cut"},
			"selected_reasons":    map[string]string{"builder": "there was room for it in this phase's 4-worker team (standard build)"},
		},
	}
	manifestFile := writeCeremonyTestJSON(t, map[string]interface{}{
		"dispatch_manifest": manifest,
	})

	_, visual, err := renderCeremonySpawnPlanFromFile("build", manifestFile)
	if err != nil {
		t.Fatalf("render spawn plan: %v", err)
	}

	for _, want := range []string{
		"Queen Budget:",
		"Required: builder, watcher",
		"Not spawned:",
		"architect",
		"not sent",
	} {
		if !strings.Contains(visual, want) {
			t.Fatalf("spawn budget ceremony missing %q\n%s", want, visual)
		}
	}
}

func TestCeremonySpawnPlanRendersQueenFrameAndSkillCards(t *testing.T) {
	manifest := ceremonyTestManifest()
	manifest["dispatch_mode"] = "plan-only"
	manifest["execution_owner"] = "host-wrapper"
	manifest["host_platform"] = "codex"
	manifest["parallel_mode"] = "in-repo"
	manifest["review_depth"] = "standard"
	manifest["colony_depth"] = "balanced"
	manifest["queen_recommendation"] = map[string]interface{}{
		"review_depth": "standard",
		"reason":       "phase has implementation and verification risk",
	}
	manifest["queen_execution_policy"] = map[string]interface{}{
		"spawn_budget": map[string]interface{}{
			"worker_count":    3,
			"selected_castes": 2,
			"selected_reasons": map[string]string{
				"builder": "there was room for it in this phase's 4-worker team (implementation risk)",
				"watcher": "required for verification",
			},
		},
	}
	manifest["dispatches"] = []map[string]interface{}{
		{
			"name":               "Brick-79",
			"agent_name":         "aether-builder",
			"caste":              "builder",
			"task_id":            "2.1",
			"task":               "CardNode wrapper",
			"execution_wave":     11,
			"wave":               1,
			"skill_count":        2,
			"colony_skill_count": 1,
			"domain_skill_count": 1,
			"matched_skills":     []string{"build-discipline", "typescript"},
		},
		{
			"name":           "Watch-64",
			"agent_name":     "aether-watcher",
			"caste":          "watcher",
			"task_id":        "verify",
			"task":           "Independent verification",
			"execution_wave": 12,
			"stage":          "verification",
			"skill_section":  "### Skill: test-writer\n\nUse focused tests.\n",
		},
	}
	manifestFile := writeCeremonyTestJSON(t, map[string]interface{}{"dispatch_manifest": manifest})

	_, visual, err := renderCeremonySpawnPlanFromFile("build", manifestFile)
	if err != nil {
		t.Fatalf("render spawn plan: %v", err)
	}

	for _, want := range []string{
		"👑🐜 Queen Orchestration",
		"mode=plan-only",
		"owner=host-wrapper",
		"platform=codex",
		"Recommendation: standard — phase has implementation and verification risk",
		"Skill Cards: 3 matched across 2 worker(s)",
		"Brick-79: build-discipline, typescript (1 colony, 1 domain)",
		"Watch-64: test-writer",
		"aether-builder",
		"there was room for it in this phase's 4-worker team",
		"Skills:2",
	} {
		if !strings.Contains(visual, want) {
			t.Fatalf("rich spawn ceremony missing %q\n%s", want, visual)
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

func TestCeremonyPlanCloseoutRendersPhaseDetails(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "Improve RytmBox UI"
	confidence := 0.86
	taskID := "1.1"
	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Milestone:    "Open Chambers",
		Plan: colony.Plan{
			Confidence: &confidence,
			Phases: []colony.Phase{
				{
					ID:          1,
					Name:        "Mixer layout overhaul",
					Description: "Make the mixer fill the panel with usable controls.",
					Tasks: []colony.Task{
						{ID: &taskID, Goal: "Remove padding and scrollbars so the mixer fills the whole panel"},
						{Goal: "Widen faders with bigger, grabbable thumbs", Hints: []string{"Use recessed tracks with raised thumbs"}},
					},
				},
				{
					ID:          2,
					Name:        "Automation lane clarity",
					Description: "Make parameter automation readable for humans.",
					Tasks: []colony.Task{
						{Goal: "Rename P-LOCK and add tooltips for cryptic labels"},
					},
				},
			},
		},
	}
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save colony state: %v", err)
	}

	completionFile := writeCeremonyTestJSON(t, map[string]interface{}{
		"plan_manifest": map[string]interface{}{"goal": goal},
		"dispatches": []map[string]interface{}{
			{"name": "Track-80", "caste": "scout", "status": "completed", "summary": "Mapped UI gaps"},
			{"name": "Route-12", "caste": "route_setter", "status": "completed", "summary": "Produced focused phases"},
		},
	})

	_, visual := renderCeremonyCloseout("plan", completionFile)
	for _, want := range []string{
		"P L A N   S U M M A R Y",
		"Planned Phases (2)",
		"Plan shape: 2 phases, 3 tasks",
		"Confidence: 86%",
		"Phase 1: Mixer layout overhaul",
		"Purpose: Make the mixer fill the panel with usable controls.",
		"1.1: Remove padding and scrollbars so the mixer fills the whole panel",
		"Hint: Use recessed tracks with raised thumbs",
		"Phase 2: Automation lane clarity",
		"Rename P-LOCK and add tooltips for cryptic labels",
	} {
		if !strings.Contains(visual, want) {
			t.Fatalf("plan closeout missing %q\n%s", want, visual)
		}
	}
}

// ---------------------------------------------------------------------------
// Task 3 (163-05): build closeout shows pending suggestions once, at the
// end, with copyable approve/dismiss commands (D-11's tick-to-approve
// presentation, at the one moment D-11 permits).
// ---------------------------------------------------------------------------

func TestCeremonyCloseoutRendersPendingSuggestionsBlock(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "Restore ceremony"
	pending := []colony.PendingSuggestion{
		{
			ID:          "sig_1",
			Type:        "REDIRECT",
			Content:     "never commit secrets or .env files to version control",
			Reason:      "detected a tracked .env file",
			ContentHash: "sha256:abc123",
			CreatedAt:   "2026-07-29T00:00:00Z",
			Dismissed:   false,
		},
	}
	state := colony.ColonyState{
		Version:            "3.0",
		Goal:               &goal,
		State:              colony.StateBUILT,
		CurrentPhase:       2,
		Milestone:          "Open Chambers",
		PendingSuggestions: &pending,
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
	})

	_, visual := renderCeremonyCloseout("build", completionFile)

	for _, want := range []string{
		"Suggestions From This Build",
		"[REDIRECT] never commit secrets or .env files to version control",
		"detected a tracked .env file",
		"sig_1",
	} {
		if !strings.Contains(visual, want) {
			t.Fatalf("closeout missing %q\n%s", want, visual)
		}
	}
}

func TestCeremonyCloseoutNamesApproveAndDismissCommandsWithRealID(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "Restore ceremony"
	pending := []colony.PendingSuggestion{
		{ID: "sig_real_id_42", Type: "FEEDBACK", Content: "high TODO density", ContentHash: "sha256:def456"},
	}
	state := colony.ColonyState{
		Version:            "3.0",
		Goal:               &goal,
		State:              colony.StateBUILT,
		CurrentPhase:       1,
		PendingSuggestions: &pending,
		Plan:               colony.Plan{Phases: []colony.Phase{{ID: 1, Name: "Foundation"}}},
	}
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save colony state: %v", err)
	}

	completionFile := writeCeremonyTestJSON(t, map[string]interface{}{
		"dispatch_manifest": ceremonyTestManifest(),
	})

	_, visual := renderCeremonyCloseout("build", completionFile)

	if !strings.Contains(visual, "aether suggest-approve --approve sig_real_id_42") {
		t.Fatalf("closeout missing a copyable approve command with the real ID\n%s", visual)
	}
	if !strings.Contains(visual, "aether suggest-approve --dismiss sig_real_id_42") {
		t.Fatalf("closeout missing a copyable dismiss command with the real ID\n%s", visual)
	}
}

func TestCeremonyCloseoutOmitsDismissedSuggestions(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "Restore ceremony"
	pending := []colony.PendingSuggestion{
		{ID: "sig_active", Type: "FEEDBACK", Content: "an active suggestion", ContentHash: "sha256:aaa", Dismissed: false},
		{ID: "sig_dismissed", Type: "FEEDBACK", Content: "a dismissed suggestion", ContentHash: "sha256:bbb", Dismissed: true},
	}
	state := colony.ColonyState{
		Version:            "3.0",
		Goal:               &goal,
		State:              colony.StateBUILT,
		CurrentPhase:       1,
		PendingSuggestions: &pending,
		Plan:               colony.Plan{Phases: []colony.Phase{{ID: 1, Name: "Foundation"}}},
	}
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save colony state: %v", err)
	}

	completionFile := writeCeremonyTestJSON(t, map[string]interface{}{
		"dispatch_manifest": ceremonyTestManifest(),
	})

	_, visual := renderCeremonyCloseout("build", completionFile)

	if !strings.Contains(visual, "an active suggestion") {
		t.Fatalf("closeout should show the active suggestion\n%s", visual)
	}
	if strings.Contains(visual, "a dismissed suggestion") {
		t.Fatalf("closeout should not show the dismissed suggestion\n%s", visual)
	}
	if strings.Contains(visual, "sig_dismissed") {
		t.Fatalf("closeout should not name the dismissed suggestion's ID\n%s", visual)
	}
}

func TestCeremonyCloseoutOmitsBlockAndHeadingWithNoPendingSuggestions(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "Restore ceremony"
	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateBUILT,
		CurrentPhase: 1,
		Plan:         colony.Plan{Phases: []colony.Phase{{ID: 1, Name: "Foundation"}}},
	}
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save colony state: %v", err)
	}

	completionFile := writeCeremonyTestJSON(t, map[string]interface{}{
		"dispatch_manifest": ceremonyTestManifest(),
	})

	_, visual := renderCeremonyCloseout("build", completionFile)

	if strings.Contains(visual, "Suggestions From This Build") {
		t.Fatalf("closeout should not render the suggestions heading with no pending suggestions\n%s", visual)
	}
	if strings.Contains(visual, "suggest-approve") {
		t.Fatalf("closeout should not mention suggest-approve with no pending suggestions\n%s", visual)
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
