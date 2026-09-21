package cmd

// Phase "the owner sees Aether's screens" Part D -- the cards drawn while
// helpers are sent out (`aether ceremony spawn-plan | wave-start |
// worker-complete | closeout`) join the guarded voice corpus. Each screen
// is measured through the exact `...FromFile` / `renderCeremonyCloseout`
// entry points the real subcommands call (cmd/ceremony_cmd.go), fed the
// same ceremonyTestManifest() fixture the existing rich-content tests in
// cmd/ceremony_cmd_test.go already use -- derived, not hand-retyped.

import (
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// ceremonyVoiceCorpusManifest builds on ceremonyTestManifest() the same way
// TestCeremonySpawnPlanRendersQueenFrameAndSkillCards
// (cmd/ceremony_cmd_test.go) does -- these fields belong ON the manifest
// map itself, not as siblings of a "dispatch_manifest" key, because
// extractCeremonyManifest resolves the manifest down to that key's value.
// "chat-app" (not the existing test's "host-wrapper") avoids the ordinary
// English word "wrapper" this repository's plain-English check also
// happens to police as one of its invented words.
func ceremonyVoiceCorpusManifest() map[string]interface{} {
	manifest := ceremonyTestManifest()
	// ceremonyTestManifest()'s "CardNode wrapper" task text is ordinary
	// fixture prose, not this repository's invented "wrapper" (a `/ant-...`
	// menu command) -- but the plain-English check cannot tell the two
	// apart, so this corpus's own copy of the fixture uses different task
	// wording rather than editing the widely shared helper.
	if dispatches, ok := manifest["dispatches"].([]map[string]interface{}); ok {
		for _, d := range dispatches {
			if d["task"] == "CardNode wrapper" {
				d["task"] = "CardNode component"
			}
		}
	}
	manifest["dispatch_mode"] = "plan-only"
	manifest["execution_owner"] = "chat-app"
	manifest["host_platform"] = "codex"
	manifest["queen_recommendation"] = map[string]interface{}{
		"review_depth": "standard",
		"reason":       "phase has implementation and verification risk",
	}
	manifest["queen_execution_policy"] = map[string]interface{}{
		"spawn_budget": map[string]interface{}{
			"worker_count":        3,
			"selected_castes":     2,
			"max_selected_castes": 4,
			"reason":              "there was room for it in this phase's 4-worker team",
			"required_castes":     []string{"builder"},
		},
	}
	return manifest
}

func classicVoiceCeremonySpawnPlanRender(t *testing.T) string {
	t.Helper()
	manifestFile := writeCeremonyTestJSON(t, map[string]interface{}{"dispatch_manifest": ceremonyVoiceCorpusManifest()})
	_, visual, err := renderCeremonySpawnPlanFromFile("build", manifestFile)
	if err != nil {
		t.Fatalf("render spawn plan: %v", err)
	}
	return visual
}

func init() {
	registerVoiceScreen("ceremony-spawn-plan", classicVoiceCeremonySpawnPlanRender)
}

func classicVoiceCeremonyWaveStartRender(t *testing.T) string {
	t.Helper()
	manifestFile := writeCeremonyTestJSON(t, map[string]interface{}{"dispatch_manifest": ceremonyVoiceCorpusManifest()})
	_, visual, err := renderCeremonyWaveStartFromFile("build", manifestFile, 11)
	if err != nil {
		t.Fatalf("render wave start: %v", err)
	}
	return visual
}

func init() {
	registerVoiceScreen("ceremony-wave-start", classicVoiceCeremonyWaveStartRender)
}

func classicVoiceCeremonyWorkerCompleteRender(t *testing.T) string {
	t.Helper()
	workerFile := writeCeremonyTestJSON(t, map[string]interface{}{
		"name":       "Brick-79",
		"caste":      "builder",
		"status":     "completed",
		"task_id":    "2.1",
		"summary":    "Finished CardNode",
		"tool_count": 18,
	})
	_, visual, err := renderCeremonyWorkerCompleteFromFile("build", workerFile)
	if err != nil {
		t.Fatalf("render worker complete: %v", err)
	}
	return visual
}

func init() {
	registerVoiceScreen("ceremony-worker-complete", classicVoiceCeremonyWorkerCompleteRender)
}

func classicVoiceCeremonyCloseoutRender(t *testing.T) string {
	t.Helper()
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	t.Cleanup(func() { _ = tmpDir })
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
	return stripANSI(visual)
}

func init() {
	registerVoiceScreen("ceremony-closeout", classicVoiceCeremonyCloseoutRender)
}

// TestCeremonyScreensMeetTheReferenceDensity holds every ceremony screen to
// the same Classic-voice bar the rest of the corpus meets.
func TestCeremonyScreensMeetTheReferenceDensity(t *testing.T) {
	reference := classicReferenceDensity(t)
	cases := map[string]func(t *testing.T) string{
		"ceremony-spawn-plan":      classicVoiceCeremonySpawnPlanRender,
		"ceremony-wave-start":      classicVoiceCeremonyWaveStartRender,
		"ceremony-worker-complete": classicVoiceCeremonyWorkerCompleteRender,
		"ceremony-closeout":        classicVoiceCeremonyCloseoutRender,
	}
	for name, render := range cases {
		name, render := name, render
		t.Run(name, func(t *testing.T) {
			rendered := render(t)
			led, total, ratio := voiceDensity(rendered)
			if total == 0 {
				t.Fatalf("%s produced no content lines to measure", name)
			}
			if ratio < reference {
				t.Errorf("%s measures %v (led=%d total=%d), below the reference figure %v.\n%s",
					name, ratio, led, total, reference, rendered)
			}
		})
	}
}

// TestTheCeremonyCommandsRenderTheScreensTheCorpusMeasures guards against
// the same drift class the status-dashboard fix closed: every ceremony
// renderer the corpus measures must have a real, non-test caller.
func TestTheCeremonyCommandsRenderTheScreensTheCorpusMeasures(t *testing.T) {
	for _, name := range []string{
		"renderCeremonySpawnPlan",
		"renderCeremonyWaveStart",
		"renderCeremonyWorkerComplete",
		"renderCeremonyCloseoutVisualBody",
	} {
		if callers := productionCallersOf(t, name); len(callers) == 0 {
			t.Fatalf("%s has no production caller, so the ceremony corpus entry would measure code no command runs", name)
		}
	}
}
