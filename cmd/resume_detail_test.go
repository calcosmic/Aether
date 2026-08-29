package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// Phase 198 plan 08 (SHOW-02) -- buildResumeDashboardResult has always built
// result["recent"] (decisions and events) and result["plan_revision"], but
// renderResumeVisual has never read either. Per-phase progress is the one
// genuinely new computation: today only the overall fraction exists. This
// file proves all three now reach the screen, on both the in-process typed
// shape and the JSON-round-tripped completion-map shape
// (renderContinueWorkerFlowValue precedent, 198-PATTERNS.md), and that the
// drift note is derived from the plan's own recorded revision rather than
// any invented signal (Phase 196 D-01).

// ---- Task 1: progress, phase by phase ----

// TestResumeShowsProgressPhaseByPhase covers a mix of finished/in-progress/
// not-started phases, the display cap with an honest overflow count, a
// colony with no plan at all rendering nothing, and the pre-existing overall
// fraction line staying unchanged.
func TestResumeShowsProgressPhaseByPhase(t *testing.T) {
	t.Run("mixed statuses render in plain English", func(t *testing.T) {
		entries := []resumePhaseProgressEntry{
			{Phase: 1, Name: "Foundation", Status: colony.PhaseCompleted},
			{Phase: 2, Name: "Core Features", Status: colony.PhaseInProgress},
			{Phase: 3, Name: "Polish", Status: colony.PhasePending},
		}

		var typedBuilder strings.Builder
		renderResumePhaseProgress(&typedBuilder, entries)
		typedOutput := typedBuilder.String()

		for _, want := range []string{
			"Phase 1 — Foundation: finished",
			"Phase 2 — Core Features: in progress",
			"Phase 3 — Polish: not started",
		} {
			if !strings.Contains(typedOutput, want) {
				t.Errorf("missing %q, got:\n%s", want, typedOutput)
			}
		}
		// Never the internal snake_case status verbatim.
		for _, internal := range []string{colony.PhaseCompleted, colony.PhaseInProgress, colony.PhasePending} {
			if strings.Contains(typedOutput, ": "+internal) {
				t.Errorf("rendered output leaks internal status %q verbatim:\n%s", internal, typedOutput)
			}
		}

		asMap := roundTripToMap(t, map[string]interface{}{"phase_progress": entries})
		var mapBuilder strings.Builder
		renderResumePhaseProgress(&mapBuilder, asMap["phase_progress"])
		mapOutput := mapBuilder.String()

		if typedOutput != mapOutput {
			t.Fatalf("round-tripped render diverged from typed render:\n%s", firstDiffLine(typedOutput, mapOutput))
		}
	})

	t.Run("more phases than the cap render the cap plus an honest count", func(t *testing.T) {
		entries := make([]resumePhaseProgressEntry, 0, 10)
		for i := 1; i <= 10; i++ {
			entries = append(entries, resumePhaseProgressEntry{Phase: i, Name: "Phase", Status: colony.PhaseCompleted})
		}
		var b strings.Builder
		renderResumePhaseProgress(&b, entries)
		output := b.String()

		if got := strings.Count(output, "finished"); got != resumePhaseProgressCap {
			t.Errorf("expected exactly %d shown phase lines, got %d in:\n%s", resumePhaseProgressCap, got, output)
		}
		if !strings.Contains(output, "(+2 more)") {
			t.Errorf("expected an honest overflow count of the 2 remaining phases, got:\n%s", output)
		}
	})

	t.Run("no plan renders no per-phase block and does not error", func(t *testing.T) {
		var b strings.Builder
		renderResumePhaseProgress(&b, nil)
		if output := b.String(); output != "" {
			t.Errorf("expected no output for a colony with no plan, got:\n%s", output)
		}

		var emptyTyped strings.Builder
		renderResumePhaseProgress(&emptyTyped, []resumePhaseProgressEntry{})
		if output := emptyTyped.String(); output != "" {
			t.Errorf("expected no output for an empty phase list, got:\n%s", output)
		}
	})

	t.Run("the existing overall fraction line is unchanged and still present", func(t *testing.T) {
		saveGlobalsCmd(t)
		var buf strings.Builder
		state := colony.ColonyState{
			CurrentPhase: 2,
			Plan: colony.Plan{
				Phases: []colony.Phase{
					{ID: 1, Name: "Foundation", Status: "completed"},
					{ID: 2, Name: "Core Features", Status: "in_progress"},
					{ID: 3, Name: "Polish", Status: "pending"},
				},
			},
		}
		s, tmpDir := newTestStoreCmd(t)
		defer os.RemoveAll(tmpDir)
		store = s
		if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
			t.Fatal(err)
		}
		result := buildResumeDashboardResult()
		buf.WriteString(renderResumeVisual(result, "", false))
		output := buf.String()

		if !strings.Contains(output, "Phase: 2/3") {
			t.Errorf("expected the existing overall fraction 'Phase: 2/3' to still appear, got:\n%s", output)
		}
		if !strings.Contains(output, "Phase Progress") {
			t.Errorf("expected the new Phase Progress block to appear, got:\n%s", output)
		}
		for _, want := range []string{
			"Phase 1 — Foundation: finished",
			"Phase 2 — Core Features: in progress",
			"Phase 3 — Polish: not started",
		} {
			if !strings.Contains(output, want) {
				t.Errorf("missing %q in full resume output:\n%s", want, output)
			}
		}
	})
}
