package cmd

// Phase "Classic Visual Voice" plan 04 -- the three work-cycle screens: the
// build screen that opens every phase, the continue screen that closes it,
// and the seal screen that finishes the project. Each task registers its own
// screen(s) into the shared corpus classic_voice_corpus_test.go built in
// plan 01, one init per task so a future plan touching the same phase never
// collides with this file's edits.

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// bareCheckboxLineRe matches a line whose first non-space characters are a
// bare square-bracket checkbox marker ("[ ]" or "[x]") -- the old,
// un-glyphed task-line style this plan replaces.
var bareCheckboxLineRe = regexp.MustCompile(`^\s*\[[ x]\]`)

// assertVoiceDensityAtLeastReference is the shared density assertion every
// task in this file uses: fail loudly if a screen produced no content lines
// to measure, and fail with the full rendered screen attached if the
// measured ratio falls below the reference figure.
func assertVoiceDensityAtLeastReference(t *testing.T, reference float64, name, rendered string) {
	t.Helper()
	led, total, ratio := voiceDensity(rendered)
	if total == 0 {
		t.Fatalf("%s produced no content lines to measure:\n%s", name, rendered)
	}
	if ratio < reference {
		t.Errorf("%s measures %v (led=%d total=%d), which is below the reference figure %v:\n%s",
			name, ratio, led, total, reference, rendered)
	}
}

// ---------------------------------------------------------------------------
// Task 1 -- the build screen: phase header, task list and stage bodies.
// ---------------------------------------------------------------------------

// buildScreenFixture returns a colony state (with phaseID selected as the
// current phase out of totalPhases), the current phase itself (a named
// phase with a description and three tasks, one of which is already
// complete), and two dispatches -- the shared shape every build-screen test
// in this file renders over.
func buildScreenFixture(phaseID, totalPhases int) (colony.ColonyState, colony.Phase, []codexBuildDispatch) {
	goal := "Restore the Classic voice across the work cycle"
	taskOneID := "task-1"
	taskTwoID := "task-2"
	taskThreeID := "task-3"
	phase := colony.Phase{
		ID:          phaseID,
		Name:        "Classic Visual Voice",
		Description: "Bring the build, continue and finish screens up to the reference density",
		Status:      colony.PhaseInProgress,
		Tasks: []colony.Task{
			{ID: &taskOneID, Goal: "Voice the build screen", Status: colony.TaskCompleted},
			{ID: &taskTwoID, Goal: "Voice the continue screen", Status: colony.TaskInProgress},
			{ID: &taskThreeID, Goal: "Voice the finish screen", Status: colony.TaskPending},
		},
	}
	phases := make([]colony.Phase, totalPhases)
	for i := range phases {
		phases[i] = colony.Phase{ID: i + 1, Name: fmt.Sprintf("Phase %d", i+1), Status: colony.PhaseReady}
	}
	phases[phaseID-1] = phase
	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: phaseID,
		Milestone:    "Open Chambers",
		Plan:         colony.Plan{Phases: phases},
	}
	dispatches := []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-11", Task: "Voice the build screen", Status: "spawned"},
		{Stage: "verification", Caste: "watcher", Name: "Keen-13", Task: "Verify the voiced screens", Status: "spawned"},
	}
	return state, phase, dispatches
}

// renderVoiceBuildScreen renders the ordinary (non-final, dispatches
// present) build screen -- the fixture both the corpus registration and the
// density test share, so the two can never quietly diverge.
func renderVoiceBuildScreen(t *testing.T) string {
	t.Helper()
	setupBuildFlowTest(t)
	state, phase, dispatches := buildScreenFixture(2, 4)
	return renderBuildVisualWithDispatches(state, phase, dispatches, colony.VerificationDepthStandard)
}

// renderVoiceBuildPartialScreen renders the partial-credit build screen over
// the same phase fixture, with one task ("task-3") still unfinished.
func renderVoiceBuildPartialScreen(t *testing.T) string {
	t.Helper()
	setupBuildFlowTest(t)
	state, phase, _ := buildScreenFixture(2, 4)
	return renderBuildPartialCreditVisual(state, phase, []string{"task-3"}, "aether build 2 --tasks task-3")
}

func init() {
	registerVoiceScreen("build", renderVoiceBuildScreen)
	registerVoiceScreen("build-partial", renderVoiceBuildPartialScreen)
}

// TestBuildScreenMeetsTheReferenceDensity measures the build screen -- both
// the corpus-registered ordinary and partial-credit renderings, and two
// variants exercised only here (a final-phase build, and a build with no
// dispatches) -- against the figure derived from the reference commits.
func TestBuildScreenMeetsTheReferenceDensity(t *testing.T) {
	reference := classicReferenceDensity(t)

	t.Run("build", func(t *testing.T) {
		assertVoiceDensityAtLeastReference(t, reference, "build", renderVoiceBuildScreen(t))
	})
	t.Run("build-partial", func(t *testing.T) {
		assertVoiceDensityAtLeastReference(t, reference, "build-partial", renderVoiceBuildPartialScreen(t))
	})
	t.Run("build/final-phase", func(t *testing.T) {
		setupBuildFlowTest(t)
		state, phase, dispatches := buildScreenFixture(4, 4)
		rendered := renderBuildVisualWithDispatches(state, phase, dispatches, colony.VerificationDepthStandard)
		assertVoiceDensityAtLeastReference(t, reference, "build/final-phase", rendered)
	})
	t.Run("build/no-dispatches", func(t *testing.T) {
		setupBuildFlowTest(t)
		state, phase, _ := buildScreenFixture(2, 4)
		rendered := renderBuildVisualWithDispatches(state, phase, nil, colony.VerificationDepthStandard)
		assertVoiceDensityAtLeastReference(t, reference, "build/no-dispatches", rendered)
	})
}

// TestBuildSteeringBlockIsUnchanged captures renderSteeringSignals over one
// active signal and asserts the output matches the exact string the
// pre-existing (already-correct) function produces for that same signal --
// so a future edit to renderBuildVisualWithDispatches around this block
// cannot quietly restyle the one piece already carrying the voice.
func TestBuildSteeringBlockIsUnchanged(t *testing.T) {
	setupBuildFlowTest(t)
	signal := colony.PheromoneSignal{
		Type:    "FOCUS",
		Content: json.RawMessage(`{"text":"keep the voice consistent"}`),
		Active:  true,
	}
	if err := store.SaveJSON("pheromones.json", colony.PheromoneFile{Signals: []colony.PheromoneSignal{signal}}); err != nil {
		t.Fatalf("seed pheromones: %v", err)
	}

	got := renderSteeringSignals()
	want := fmt.Sprintf("Steering signals: %d active — injected into every worker prompt\n", 1) +
		fmt.Sprintf("  %s [%d%%] %q\n", signalTypeGlyph(signal.Type), 100, "keep the voice consistent")

	if got != want {
		t.Errorf("renderSteeringSignals output changed for an unchanged signal set:\n got:  %q\nwant: %q", got, want)
	}
}

// TestBuildScreenHasNoBareCheckboxMarkers asserts every former "[ ] "/"[x] "
// task line has been replaced by a glyph-led line, not merely had a glyph
// added beside the bracket -- a regression this plan's whole point is to
// close, on both the ordinary and partial-credit build screens.
func TestBuildScreenHasNoBareCheckboxMarkers(t *testing.T) {
	for name, rendered := range map[string]string{
		"build":         renderVoiceBuildScreen(t),
		"build-partial": renderVoiceBuildPartialScreen(t),
	} {
		for _, line := range strings.Split(rendered, "\n") {
			if bareCheckboxLineRe.MatchString(line) {
				t.Errorf("%s still renders a bare checkbox marker line: %q", name, line)
			}
		}
	}
}

// TestBuildScreenFactsUnchanged asserts the voiced build screen still names
// the same phase, the same task goals and the same dispatch count as the
// unvoiced screen did -- presentation changed, nothing else did.
func TestBuildScreenFactsUnchanged(t *testing.T) {
	rendered := renderVoiceBuildScreen(t)
	_, phase, dispatches := buildScreenFixture(2, 4)

	if !strings.Contains(rendered, phase.Name) {
		t.Errorf("build screen no longer names the phase %q:\n%s", phase.Name, rendered)
	}
	for _, task := range phase.Tasks {
		if !strings.Contains(rendered, strings.TrimSpace(task.Goal)) {
			t.Errorf("build screen no longer names task goal %q:\n%s", task.Goal, rendered)
		}
	}
	wantDispatchLine := fmt.Sprintf("Total planned dispatches: %d", len(dispatches))
	if !strings.Contains(rendered, wantDispatchLine) {
		t.Errorf("build screen no longer reports %q:\n%s", wantDispatchLine, rendered)
	}
}
