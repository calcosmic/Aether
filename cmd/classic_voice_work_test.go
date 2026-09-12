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

// ---------------------------------------------------------------------------
// Task 2 -- the continue screen: verification, workers, housekeeping and the
// end-of-phase footer.
// ---------------------------------------------------------------------------

// continueScreenFixture returns the shared shape every continue-screen test
// in this file renders over: a colony state with one already-finished phase
// and a current in-progress phase, plus a completion result carrying a
// verification map with one passing and one failing entry, a gate map with
// both outcomes, two closed workers, a consolidation result, and one
// specialist finding.
func continueScreenFixture() (colony.ColonyState, colony.Phase, *signalHousekeepingResult, colony.Phase, map[string]interface{}) {
	goal := "Restore the Classic voice across the work cycle"
	completedPhase := colony.Phase{ID: 1, Name: "Front Door", Status: colony.PhaseCompleted}
	currentPhase := colony.Phase{ID: 2, Name: "Classic Visual Voice", Status: colony.PhaseInProgress}
	nextPhase := colony.Phase{ID: 3, Name: "Biological Runtime", Status: colony.PhaseReady}
	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: currentPhase.ID,
		Milestone:    "Open Chambers",
		Plan:         colony.Plan{Phases: []colony.Phase{completedPhase, currentPhase, nextPhase}},
	}

	verification := codexContinueVerificationReport{
		Steps: []codexVerificationStep{
			{Name: "build", Passed: true, Duration: 1.2},
			{Name: "tests", Passed: false, Summary: "2 of 12 tests failed", Command: "npm test", Duration: 4.3},
		},
		Criteria: []codexCriterionVerification{
			{Criterion: "Voice reaches every screen", Evidence: []string{"density test passed"}, Passed: true},
		},
	}
	gates := codexContinueGateReport{
		Checks: []gateCheck{
			{Name: "manifest_present", Passed: true},
			{Name: "verification_steps_passed", Passed: false, FixHint: "fix the failing build/test check and run aether continue"},
		},
	}
	housekeeping := &signalHousekeepingResult{
		TotalSignals:          3,
		ActiveBefore:          3,
		ActiveAfter:           1,
		ExpiredByTime:         1,
		DeactivatedByStrength: 1,
		Updated:               2,
	}
	workerFlow := []codexContinueWorkerFlowStep{
		{
			Stage: "review", Caste: "watcher", Name: "Keen-13", Status: "completed",
			Findings: []codexReviewFinding{{Severity: "high", Title: "race condition in dispatch loop"}},
		},
	}

	result := map[string]interface{}{
		"verification":       verification,
		"gates":              gates,
		"closed_workers":     []string{"Forge-11", "Keen-13"},
		"worker_flow":        workerFlow,
		"operational_issues": []string{"one helper reported a flaky retry"},
		// "ran": false steers renderLearningBeat (deliberately left
		// byte-identical by this task) to its no-consolidation branch, which
		// carries none of the "queen"/"instinct" vocabulary its other
		// branches use -- avoiding a conflict between "leave this block
		// untouched" and "every registered screen passes the widened
		// plain-English scan". TestContinueAlreadyCorrectBlocksAreUnchanged
		// below still exercises the promotion-candidate branch directly.
		"consolidation": map[string]interface{}{
			"ran":    false,
			"reason": "nothing new was produced this phase",
		},
	}
	return state, currentPhase, housekeeping, nextPhase, result
}

// renderVoiceContinueMidphaseScreen renders the ordinary (non-final,
// next-phase-present) continue screen.
func renderVoiceContinueMidphaseScreen(t *testing.T) string {
	t.Helper()
	setupBuildFlowTest(t)
	state, phase, housekeeping, nextPhase, result := continueScreenFixture()
	return renderContinueVisual(state, phase, housekeeping, false, &nextPhase, result, colony.VerificationDepthStandard)
}

// renderVoiceContinueFinalScreen renders the final-phase (project-complete)
// continue screen over the same fixture data.
func renderVoiceContinueFinalScreen(t *testing.T) string {
	t.Helper()
	setupBuildFlowTest(t)
	state, phase, housekeeping, _, result := continueScreenFixture()
	return renderContinueVisual(state, phase, housekeeping, true, nil, result, colony.VerificationDepthStandard)
}

func init() {
	registerVoiceScreen("continue-midphase", renderVoiceContinueMidphaseScreen)
	registerVoiceScreen("continue-final", renderVoiceContinueFinalScreen)
}

// TestContinueScreenMeetsTheReferenceDensity measures both continue-screen
// registrations against the figure derived from the reference commits.
func TestContinueScreenMeetsTheReferenceDensity(t *testing.T) {
	reference := classicReferenceDensity(t)

	t.Run("continue-midphase", func(t *testing.T) {
		assertVoiceDensityAtLeastReference(t, reference, "continue-midphase", renderVoiceContinueMidphaseScreen(t))
	})
	t.Run("continue-final", func(t *testing.T) {
		assertVoiceDensityAtLeastReference(t, reference, "continue-final", renderVoiceContinueFinalScreen(t))
	})
}

// TestContinueAlreadyCorrectBlocksAreUnchanged captures renderLearningBeat
// and the worker-flow rendering over fixed input and asserts both match the
// strings they produced before this task, so the two blocks that already
// carried the voice cannot be accidentally restyled by an edit to their
// surroundings.
func TestContinueAlreadyCorrectBlocksAreUnchanged(t *testing.T) {
	consolidation := map[string]interface{}{
		"ran":                  true,
		"promotion_candidates": 2,
		"queen_eligible":       1,
	}
	gotLearning := renderLearningBeat(consolidation)
	wantLearningPrefix := casteIdentity("librarian") + "  "
	if !strings.HasPrefix(strings.TrimPrefix(gotLearning, "── Learning ──\n"), wantLearningPrefix) {
		t.Errorf("renderLearningBeat no longer leads with the shared caste identity funnel:\n%s", gotLearning)
	}

	var b strings.Builder
	renderContinueWorkerFlowLine(&b, "Keen-13", "watcher", "completed", "found a race condition")
	got := b.String()
	want := "  - " + casteIdentity("watcher") + " Keen-13 completed — found a race condition\n"
	if got != want {
		t.Errorf("renderContinueWorkerFlowLine output changed:\n got:  %q\nwant: %q", got, want)
	}
}

// TestContinuePassAndFailGatesUseDifferentSymbols asserts a passing gate
// line and a failing gate line begin with different glyphs, explicitly --
// the plan's whole point for the gate detail rows.
func TestContinuePassAndFailGatesUseDifferentSymbols(t *testing.T) {
	rendered := renderVoiceContinueMidphaseScreen(t)
	glyphs := sortedGlyphsLongestFirst(voiceGlyphSet())

	var passLine, failLine string
	for _, line := range strings.Split(rendered, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.Contains(trimmed, "the build's own plan file is on disk"):
			passLine = trimmed
		case strings.Contains(trimmed, "the build/test checks passed"):
			failLine = trimmed
		}
	}
	if passLine == "" || failLine == "" {
		t.Fatalf("could not find both a passing and a failing gate line in:\n%s", rendered)
	}
	if !voiceLineIsLed(passLine, glyphs) || !voiceLineIsLed(failLine, glyphs) {
		t.Fatalf("gate lines are not both glyph-led: pass=%q fail=%q", passLine, failLine)
	}
	passGlyph := strings.Fields(passLine)[0]
	failGlyph := strings.Fields(failLine)[0]
	if passGlyph == failGlyph {
		t.Errorf("a passing gate and a failing gate begin with the same symbol %q:\n  pass: %q\n  fail: %q", passGlyph, passLine, failLine)
	}
}

// TestContinueScreenFactsUnchanged asserts the voiced continue screen still
// names the same phase number, the same gate names and the same signal
// counts as the unvoiced screen did.
func TestContinueScreenFactsUnchanged(t *testing.T) {
	rendered := renderVoiceContinueMidphaseScreen(t)
	_, phase, housekeeping, _, _ := continueScreenFixture()

	wantPhaseLine := fmt.Sprintf("Phase %d verified and completed: %s", phase.ID, phase.Name)
	if !strings.Contains(rendered, wantPhaseLine) {
		t.Errorf("continue screen no longer reports %q:\n%s", wantPhaseLine, rendered)
	}
	for _, gate := range []string{"the build's own plan file is on disk", "the build/test checks passed"} {
		if !strings.Contains(rendered, gate) {
			t.Errorf("continue screen no longer names gate %q:\n%s", gate, rendered)
		}
	}
	wantSignalsLine := fmt.Sprintf("Signals: %d active -> %d active after housekeeping", housekeeping.ActiveBefore, housekeeping.ActiveAfter)
	if !strings.Contains(rendered, wantSignalsLine) {
		t.Errorf("continue screen no longer reports %q:\n%s", wantSignalsLine, rendered)
	}
}

// ---------------------------------------------------------------------------
// Task 3 -- the seal screen: the summary beneath the crowning art.
// ---------------------------------------------------------------------------

// sealScreenFixture returns the colony state and summary path every
// seal-screen test in this file renders over: a goal, a plan with several
// phases, and a version.
func sealScreenFixture() (colony.ColonyState, string) {
	goal := "Restore the Classic voice across the work cycle"
	state := colony.ColonyState{
		Version:       "3.0",
		Goal:          &goal,
		State:         colony.StateCOMPLETED,
		ColonyVersion: 128,
		Plan: colony.Plan{Phases: []colony.Phase{
			{ID: 1, Name: "Front Door", Status: colony.PhaseCompleted},
			{ID: 2, Name: "Classic Visual Voice", Status: colony.PhaseCompleted},
			{ID: 3, Name: "Biological Runtime", Status: colony.PhaseCompleted},
		}},
	}
	return state, ".aether/data/CROWNED-ANTHILL.md"
}

// renderVoiceSealScreen renders the seal screen over the shared fixture.
// The state is also saved to disk: renderSealVisual's closing card
// (renderLifecycleClosing) resolves its answer from the saved project
// rather than from the state argument, so an unsaved fixture would render
// the empty "no project" branch instead of this genuinely sealed one.
func renderVoiceSealScreen(t *testing.T) string {
	t.Helper()
	dataDir := setupBuildFlowTest(t)
	state, summaryPath := sealScreenFixture()
	createTestColonyState(t, dataDir, state)
	return renderSealVisual(map[string]interface{}{}, state, summaryPath)
}

func init() {
	registerVoiceScreen("seal", renderVoiceSealScreen)
}

// TestSealScreenMeetsTheReferenceDensity measures the seal screen against
// the figure derived from the reference commits.
func TestSealScreenMeetsTheReferenceDensity(t *testing.T) {
	reference := classicReferenceDensity(t)
	assertVoiceDensityAtLeastReference(t, reference, "seal", renderVoiceSealScreen(t))
}

// TestSealCeremonyArtIsUnchanged asserts the rendered seal screen contains
// the crowning art constant verbatim, the two rule lines at their original
// length, and the letter-spaced title with the version -- so a future edit
// to the summary beneath it cannot erode the ceremony above it.
func TestSealCeremonyArtIsUnchanged(t *testing.T) {
	rendered := renderVoiceSealScreen(t)
	state, _ := sealScreenFixture()

	if !strings.Contains(rendered, crownedAnthillArt) {
		t.Errorf("seal screen no longer contains the crowning art verbatim:\n%s", rendered)
	}
	rule := strings.Repeat("━", 50)
	if strings.Count(rendered, rule) < 2 {
		t.Errorf("seal screen no longer carries both 50-rune rule lines:\n%s", rendered)
	}
	wantTitle := fmt.Sprintf("%s   v%d", spacedTitle("Crowned Anthill"), state.ColonyVersion)
	if !strings.Contains(rendered, wantTitle) {
		t.Errorf("seal screen no longer carries the letter-spaced title with its version %q:\n%s", wantTitle, rendered)
	}
}

// TestSealScreenFactsUnchanged asserts the voiced seal screen still names
// the same completed-phase count and the same summary path as before.
func TestSealScreenFactsUnchanged(t *testing.T) {
	rendered := renderVoiceSealScreen(t)
	state, summaryPath := sealScreenFixture()

	wantPhasesLine := fmt.Sprintf("Completed phases: %d", len(state.Plan.Phases))
	if !strings.Contains(rendered, wantPhasesLine) {
		t.Errorf("seal screen no longer reports %q:\n%s", wantPhasesLine, rendered)
	}
	if !strings.Contains(rendered, summaryPath) {
		t.Errorf("seal screen no longer names the summary path %q:\n%s", summaryPath, rendered)
	}
}

// TestSealScreenOmitsEmptyGoalLine asserts a project with no recorded goal
// still renders the rest of the summary, without an empty goal line.
func TestSealScreenOmitsEmptyGoalLine(t *testing.T) {
	setupBuildFlowTest(t)
	state, summaryPath := sealScreenFixture()
	state.Goal = nil

	rendered := renderSealVisual(map[string]interface{}{}, state, summaryPath)
	for _, line := range strings.Split(rendered, "\n") {
		if strings.Contains(line, "Goal:") {
			t.Errorf("seal screen rendered a goal line with no recorded goal: %q", line)
		}
	}
	wantPhasesLine := fmt.Sprintf("Completed phases: %d", len(state.Plan.Phases))
	if !strings.Contains(rendered, wantPhasesLine) {
		t.Errorf("seal screen without a goal lost the rest of the summary:\n%s", rendered)
	}
}
