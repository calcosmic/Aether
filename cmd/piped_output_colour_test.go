package cmd

import (
	"bytes"
	"strings"
	"testing"
)

// TestPipedVisualOutputCarriesNoEscapeCodes locks Part A of the "the owner
// sees Aether's screens, every time" release: AETHER_OUTPUT_MODE=visual used
// to force ANSI colour on even when stdout was not a real terminal -- a
// pasted-into-chat screen carried raw escape codes as junk characters. Colour
// must now depend on whether stdout is actually a terminal, not just on the
// output mode. An explicit force (AETHER_FORCE_COLOR=1 / CLICOLOR_FORCE)
// still wins regardless of the writer.
func TestPipedVisualOutputCarriesNoEscapeCodes(t *testing.T) {
	t.Run("visual mode into a non-terminal writer carries no escape codes", func(t *testing.T) {
		saveGlobals(t)
		t.Setenv("AETHER_OUTPUT_MODE", "visual")

		var buf bytes.Buffer
		oldStdout := stdout
		stdout = &buf
		defer func() { stdout = oldStdout }()

		s, root := setupTestStore(t)
		store = s
		t.Setenv("AETHER_ROOT", root)
		state := loadStatusTestState(t, s)
		if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
			t.Fatalf("persist fixture: %v", err)
		}
		statusVisual := renderDashboard(state, s, buildStatusResult(state, s))

		closeoutVisual := renderCloseoutVisual(map[string]interface{}{
			"workflow":         "build",
			"state_available":  true,
			"goal":             "Ship the reporting feature",
			"state":            "in_progress",
			"total_phases":     3,
			"completed_phases": 1,
			"current_phase":    2,
		})

		manifestFile := writeCeremonyTestJSON(t, map[string]interface{}{
			"ok": true,
			"result": map[string]interface{}{
				"dispatch_manifest": ceremonyTestManifest(),
			},
		})
		_, spawnPlanVisual, err := renderCeremonySpawnPlanFromFile("build", manifestFile)
		if err != nil {
			t.Fatalf("render spawn plan: %v", err)
		}

		for name, visual := range map[string]string{
			"status":              statusVisual,
			"build closeout":      closeoutVisual,
			"ceremony spawn plan": spawnPlanVisual,
		} {
			if strings.Contains(visual, "\x1b") {
				t.Errorf("%s screen carried an ANSI escape code while piped into a non-terminal writer:\n%s", name, visual)
			}
		}
	})

	t.Run("AETHER_FORCE_COLOR=1 still colours a non-terminal writer", func(t *testing.T) {
		saveGlobals(t)
		t.Setenv("AETHER_OUTPUT_MODE", "visual")
		t.Setenv("AETHER_FORCE_COLOR", "1")

		var buf bytes.Buffer
		oldStdout := stdout
		stdout = &buf
		defer func() { stdout = oldStdout }()

		manifestFile := writeCeremonyTestJSON(t, map[string]interface{}{
			"ok": true,
			"result": map[string]interface{}{
				"dispatch_manifest": ceremonyTestManifest(),
			},
		})
		_, spawnPlanVisual, err := renderCeremonySpawnPlanFromFile("build", manifestFile)
		if err != nil {
			t.Fatalf("render spawn plan: %v", err)
		}

		if !strings.Contains(spawnPlanVisual, "\x1b") {
			t.Fatalf("AETHER_FORCE_COLOR=1 did not colour the spawn plan screen:\n%s", spawnPlanVisual)
		}
	})
}
