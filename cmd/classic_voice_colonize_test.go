package cmd

// Phase "the owner sees Aether's screens" Part D -- the code survey screen
// (`aether colonize`) joins the guarded voice corpus, measured against the
// two real renderers the command calls directly: renderColonizeVisual (the
// finished survey) and renderColonizeDispatchPreview (the in-flight
// dispatch banner shown while surveyors are sent out).
//
// These renderers are exercised directly (as
// cmd/classic_voice_status_dashboard_test.go exercises renderDashboard
// directly), not through a full `rootCmd.Execute()` of `aether colonize`.
// A full command run also streams a separate, widely shared simulated-
// dispatch progress feed (cmd/codex_build_progress.go, pkg/codex/worker.go)
// between these two screens -- machinery build/continue/seal already
// depend on and this plan's Part D does not own -- and that stream's own
// "for worker X (caste: Y)" bookkeeping text is not actually part of
// either renderer this corpus is measuring.

import (
	"testing"
)

// colonizeVoiceCorpusResult mirrors the shape TestRenderColonizeVisual_FakeDispatch
// (cmd/codex_visuals_test.go) already proves renderColonizeVisual accepts
// in production ("spawned" status keeps every surveyor on the plain,
// non-execution-detail rendering branch -- hasRealExecutionData -- so this
// fixture measures the screen's own prose, not the separate simulated
// per-worker execution feed).
func colonizeVoiceCorpusResult() map[string]interface{} {
	return map[string]interface{}{
		"root":          "/Users/owner/repos/reporting-service",
		"detected_type": "go",
		"languages":     []interface{}{"go"},
		"frameworks":    []interface{}{"cobra"},
		"domains":       []interface{}{"cli"},
		"stats": map[string]interface{}{
			"files":       42,
			"directories": 7,
		},
		"survey_dir": "/Users/owner/repos/reporting-service/.aether/data/survey",
		"surveyors": []interface{}{
			map[string]interface{}{"name": "Nest-42", "caste": "surveyor-nest", "task": "Map architecture", "status": "spawned"},
			map[string]interface{}{"name": "Disc-7", "caste": "surveyor-disciplines", "task": "Map disciplines", "status": "spawned"},
			map[string]interface{}{"name": "Path-3", "caste": "surveyor-pathogens", "task": "Identify pathogens", "status": "spawned"},
			map[string]interface{}{"name": "Prov-1", "caste": "surveyor-provisions", "task": "Map provisions", "status": "spawned"},
		},
		"survey_files": []interface{}{"BLUEPRINT.md", "CHAMBERS.md", "DISCIPLINES.md", "PATHOGENS.md", "PROVISIONS.md"},
	}
}

func classicVoiceColonizeSurveyRender(t *testing.T) string {
	t.Helper()
	return renderColonizeVisual(colonizeVoiceCorpusResult())
}

func init() {
	registerVoiceScreen("colonize-survey", classicVoiceColonizeSurveyRender)
}

func classicVoiceColonizeDispatchPreviewRender(t *testing.T) string {
	t.Helper()
	dispatches := []codexSurveyorDispatch{
		{Name: "Nest-42", Caste: "surveyor-nest", Task: "Map architecture", Status: "spawned"},
		{Name: "Disc-7", Caste: "surveyor-disciplines", Task: "Map disciplines", Status: "spawned"},
		{Name: "Path-3", Caste: "surveyor-pathogens", Task: "Identify pathogens", Status: "spawned"},
		{Name: "Prov-1", Caste: "surveyor-provisions", Task: "Map provisions", Status: "spawned"},
	}
	return renderColonizeDispatchPreview("/Users/owner/repos/reporting-service", dispatches)
}

func init() {
	registerVoiceScreen("colonize-dispatch-preview", classicVoiceColonizeDispatchPreviewRender)
}

func TestColonizeScreensMeetTheReferenceDensity(t *testing.T) {
	reference := classicReferenceDensity(t)
	cases := map[string]func(t *testing.T) string{
		"colonize-survey":           classicVoiceColonizeSurveyRender,
		"colonize-dispatch-preview": classicVoiceColonizeDispatchPreviewRender,
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

func TestTheColonizeCommandRendersTheScreensTheCorpusMeasures(t *testing.T) {
	for _, name := range []string{"renderColonizeVisual", "renderColonizeDispatchPreview"} {
		if callers := productionCallersOf(t, name); len(callers) == 0 {
			t.Fatalf("%s has no production caller, so the colonize corpus entry would measure code no command runs", name)
		}
	}
}
