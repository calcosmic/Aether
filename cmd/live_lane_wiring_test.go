package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
)

// liveTestEvent pairs a persisted event's topic with its decoded payload --
// convenient for the ordering/grouping assertions below, which need both.
type liveTestEvent struct {
	Topic   string
	Payload events.ColonyLivePayload
}

// liveEventsSince decodes every persisted live.* event for the current
// store, in persisted (emission) order.
func liveEventsSince(t *testing.T) []liveTestEvent {
	t.Helper()
	raw := readColonyLiveEventsRaw(store, time.Time{})
	out := make([]liveTestEvent, 0, len(raw))
	for _, evt := range raw {
		var payload events.ColonyLivePayload
		if err := json.Unmarshal(evt.Payload, &payload); err != nil {
			t.Fatalf("decode live payload for topic %s: %v", evt.Topic, err)
		}
		out = append(out, liveTestEvent{Topic: evt.Topic, Payload: payload})
	}
	return out
}

func liveTestEventTopics(events []liveTestEvent) []string {
	topics := make([]string, 0, len(events))
	for _, e := range events {
		topics = append(topics, e.Topic)
	}
	return topics
}

// TestBuildLaneEmitsOrderedLiveEvents drives a real fixture build (one
// builder, one wave) through the direct build lane's real public entry
// point and proves the persisted live-event sequence matches the plan's
// behaviour contract: episode-started, wave-started, one worker-started and
// one worker-finished, wave-ended, episode-ended -- in that order, all
// sharing one episode ID and the build episode kind.
func TestBuildLaneEmitsOrderedLiveEvents(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceBuildJSONOutput(t)

	goal := "Prove the build lane speaks on the live stream"
	taskID := "1.1"
	accepted := createApprovedAcceptedBuildTestColony(t, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "full",
		CurrentPhase: 0,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:              1,
					Name:            "Live build wiring",
					Description:     "Prove the direct build lane emits live events",
					Status:          colony.PhaseReady,
					Tasks:           []colony.Task{{ID: &taskID, Goal: "Wire the live build path", Status: colony.TaskPending}},
					SuccessCriteria: []string{"Build emits live events"},
				},
			},
		},
	})
	root := accepted.Root
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("failed to chdir to test root: %v", err)
	}
	defer os.Chdir(oldDir)

	rootCmd.SetArgs([]string{"build", "1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build returned error: %v", err)
	}

	liveEvents := liveEventsSince(t)
	topics := liveTestEventTopics(liveEvents)
	want := []string{
		events.LiveTopicEpisodeStarted,
		events.LiveTopicWaveStarted,
		events.LiveTopicWorkerStarted,
		events.LiveTopicWorkerFinished,
		events.LiveTopicWaveEnded,
		// 204-15: the build lane records its outcome (evidence, gate results,
		// changed decisions) on the durable ledger before the episode closes.
		events.LiveTopicOutcomeRecorded,
		events.LiveTopicEpisodeEnded,
	}
	if len(topics) != len(want) {
		t.Fatalf("live topic sequence = %v, want %v", topics, want)
	}
	for i, topic := range want {
		if topics[i] != topic {
			t.Fatalf("live topic[%d] = %q, want %q (full sequence: %v)", i, topics[i], topic, topics)
		}
	}

	episodeID := liveEvents[0].Payload.EpisodeID
	if episodeID == "" {
		t.Fatal("expected a non-empty episode ID on the episode-started event")
	}
	for i, e := range liveEvents {
		if e.Payload.EpisodeID != episodeID {
			t.Fatalf("event[%d] (%s) EpisodeID = %q, want %q (every event in one build shares one episode)", i, e.Topic, e.Payload.EpisodeID, episodeID)
		}
		if e.Payload.EpisodeKind != events.EpisodeKindBuild {
			t.Fatalf("event[%d] (%s) EpisodeKind = %q, want %q", i, e.Topic, e.Payload.EpisodeKind, events.EpisodeKindBuild)
		}
	}
}

// continueLiveWiringFixture writes the shared continue fixture used by both
// TestCheckLaneEmitsOneTerminalEventPerCheck and
// TestFailedLiveEmitNeverChangesLaneOutcome's check case: a phase in
// progress with one completed builder dispatch, and four fast, deterministic
// verification commands (build/types/lint/tests) documented so the
// deterministic floor actually runs all four rather than skipping them.
func continueLiveWiringFixture(t *testing.T, goal string) (dataDir, root string) {
	t.Helper()
	dataDir = setupBuildFlowTest(t)
	root = filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte("## Verification Commands\n\n```bash\n# Verify Go binary builds\nprintf live-build\n\n# Run Go tests\nprintf live-test\n```\n"), 0644); err != nil {
		t.Fatalf("write CLAUDE.md: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".aether", "data"), 0755); err != nil {
		t.Fatalf("mkdir codebase dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".aether", "data", "codebase.md"), []byte("## Commands\n- Types: `printf live-types`\n- Lint: `printf live-lint`\n"), 0644); err != nil {
		t.Fatalf("write codebase.md: %v", err)
	}

	now := time.Now().UTC()
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:        "3.0",
		Goal:           &goal,
		State:          colony.StateBUILT,
		CurrentPhase:   1,
		BuildStartedAt: &now,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Live check wiring",
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Verify using documented commands", Status: colony.TaskInProgress}},
				},
			},
		},
	})
	seedContinueBuildPacket(t, dataDir, 1, "Live check wiring", goal, []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-live-1", Task: "Verify using documented commands", Status: "completed", TaskID: taskID},
	})
	return dataDir, root
}

func isLiveCheckTopic(topic string) bool {
	switch topic {
	case events.LiveTopicCheckStarted, events.LiveTopicCheckPassed, events.LiveTopicCheckFailed:
		return true
	default:
		return false
	}
}

// TestCheckLaneEmitsOneTerminalEventPerCheck drives a real fixture continue
// through the deterministic floor with four documented, fast, passing
// verification commands and proves each check emits check-started followed
// by exactly one terminal (check-passed) event, with check names matching
// the floor's own step names.
func TestCheckLaneEmitsOneTerminalEventPerCheck(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	_, _ = continueLiveWiringFixture(t, "Prove the check lane speaks on the live stream")

	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	liveEvents := liveEventsSince(t)
	checkOrder := []string{}
	byCheck := map[string][]string{}
	for _, e := range liveEvents {
		if !isLiveCheckTopic(e.Topic) {
			continue
		}
		name := e.Payload.CheckName
		if _, seen := byCheck[name]; !seen {
			checkOrder = append(checkOrder, name)
		}
		byCheck[name] = append(byCheck[name], e.Topic)
		if e.Payload.EpisodeKind != events.EpisodeKindContinue {
			t.Fatalf("check event for %q carries EpisodeKind %q, want %q", name, e.Payload.EpisodeKind, events.EpisodeKindContinue)
		}
	}

	wantChecks := map[string]bool{"build": true, "types": true, "lint": true, "tests": true}
	if len(byCheck) != len(wantChecks) {
		t.Fatalf("checks observed on the live stream = %v, want exactly %v", checkOrder, wantChecks)
	}
	for name := range wantChecks {
		topics, ok := byCheck[name]
		if !ok {
			t.Fatalf("no live check events observed for check %q; observed checks: %v", name, checkOrder)
		}
		if len(topics) != 2 {
			t.Fatalf("check %q emitted %d live events (%v), want exactly 2 (started + one terminal)", name, len(topics), topics)
		}
		if topics[0] != events.LiveTopicCheckStarted {
			t.Fatalf("check %q's first live event = %q, want %q", name, topics[0], events.LiveTopicCheckStarted)
		}
		if topics[1] != events.LiveTopicCheckPassed {
			t.Fatalf("check %q's terminal live event = %q, want %q (documented commands should all pass)", name, topics[1], events.LiveTopicCheckPassed)
		}
	}
}

// TestPlanLaneEmitsPlanningEpisode drives a real fixture planning pass
// (synthetic mode: scout + route_setter, no real worker invocation) through
// the planning lane's real public entry point and proves it produces
// episode/wave/worker events whose episode kind is the planning kind.
func TestPlanLaneEmitsPlanningEpisode(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("failed to chdir to test root: %v", err)
	}
	defer os.Chdir(oldDir)

	goal := "Prove the plan lane speaks on the live stream"
	fixtureState := codexPlanSpecificationFixture(t, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	}, colony.SpecStatusApproved)
	createTestColonyState(t, dataDir, fixtureState)
	writeCodexPlanSpecificationProjection(t, root, fixtureState)

	rootCmd.SetArgs([]string{"colonize"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("colonize returned error: %v", err)
	}

	rootCmd.SetArgs([]string{"plan", "--synthetic", "--preset", "balanced"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan returned error: %v", err)
	}

	liveEvents := liveEventsSince(t)
	planEvents := []liveTestEvent{}
	for _, e := range liveEvents {
		if e.Payload.EpisodeKind == events.EpisodeKindPlan {
			planEvents = append(planEvents, e)
		}
	}
	if len(planEvents) == 0 {
		t.Fatalf("expected at least one live event with EpisodeKind %q, observed topics: %v", events.EpisodeKindPlan, liveTestEventTopics(liveEvents))
	}

	haveTopic := map[string]bool{}
	for _, e := range planEvents {
		haveTopic[e.Topic] = true
	}
	for _, topic := range []string{
		events.LiveTopicEpisodeStarted,
		events.LiveTopicEpisodeEnded,
		events.LiveTopicWaveStarted,
		events.LiveTopicWaveEnded,
		events.LiveTopicWorkerStarted,
		events.LiveTopicWorkerFinished,
	} {
		if !haveTopic[topic] {
			t.Fatalf("planning episode never emitted %q; observed: %v", topic, planEvents)
		}
	}
}

// TestRecoveryLaneEmitsRecordedState drives a real recovery decision through
// buildExternalBuildRecoveryInstructions (the real public entry point every
// build-lane recovery call goes through) and proves the resulting
// recovery-changed live event's RecoveryState equals the action actually
// persisted to the durable recovery log.
func TestRecoveryLaneEmitsRecordedState(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	const phaseNum = 202

	dispatches := []codexBuildDispatch{
		{Name: "Forge-recovery-1", Caste: "builder", TaskID: "1.1", Wave: 1, Status: "failed", Summary: "hit an unrecoverable-looking snag"},
	}

	instructions, err := buildExternalBuildRecoveryInstructions(phaseNum, dispatches)
	if err != nil {
		t.Fatalf("buildExternalBuildRecoveryInstructions: %v", err)
	}
	if len(instructions) != 1 {
		t.Fatalf("expected exactly 1 recovery instruction, got %d: %+v", len(instructions), instructions)
	}
	wantAction, _ := instructions[0]["action"].(string)
	if wantAction == "" {
		t.Fatalf("fixture is broken: recovery instruction carries no action: %+v", instructions[0])
	}

	logFile, err := recoveryLogReadPhase(phaseNum)
	if err != nil {
		t.Fatalf("recoveryLogReadPhase: %v", err)
	}
	if len(logFile.Entries) == 0 {
		t.Fatal("fixture is broken: no recovery log entries were persisted")
	}
	lastEntry := logFile.Entries[len(logFile.Entries)-1]
	if lastEntry.ActionTaken != wantAction {
		t.Fatalf("recovery log's last entry ActionTaken = %q, want %q (must match the returned instruction)", lastEntry.ActionTaken, wantAction)
	}

	liveEvents := liveEventsSince(t)
	recoveryEvents := []liveTestEvent{}
	for _, e := range liveEvents {
		if e.Topic == events.LiveTopicRecoveryChanged {
			recoveryEvents = append(recoveryEvents, e)
		}
	}
	if len(recoveryEvents) != 1 {
		t.Fatalf("expected exactly 1 live.recovery.changed event, got %d: %+v", len(recoveryEvents), recoveryEvents)
	}
	recoveryPayload := recoveryEvents[0].Payload
	if recoveryPayload.EpisodeKind != events.EpisodeKindRecovery {
		t.Fatalf("recovery event EpisodeKind = %q, want %q", recoveryPayload.EpisodeKind, events.EpisodeKindRecovery)
	}
	if recoveryPayload.RecoveryState != wantAction {
		t.Fatalf("recovery event RecoveryState = %q, want %q (must equal the state written to durable storage)", recoveryPayload.RecoveryState, wantAction)
	}
}

// TestFailedLiveEmitNeverChangesLaneOutcome runs one build, one check, one
// planning pass and one recovery transition twice against isolated fixture
// roots -- once normally and once with the emission boundary forced to fail
// -- and asserts each pair produces equal result values and equal durable
// state. A failing live-event publish must never change the outcome of the
// work it describes, on every lane.
func TestFailedLiveEmitNeverChangesLaneOutcome(t *testing.T) {
	t.Run("build", func(t *testing.T) {
		normal := runFailedEmitBuildFixture(t, false)
		forced := runFailedEmitBuildFixture(t, true)
		if normal != forced {
			t.Fatalf("build outcome differs with emission forced to fail:\n normal: %+v\n forced: %+v", normal, forced)
		}
	})

	t.Run("check", func(t *testing.T) {
		normal := runFailedEmitCheckFixture(t, false)
		forced := runFailedEmitCheckFixture(t, true)
		if normal != forced {
			t.Fatalf("check outcome differs with emission forced to fail:\n normal: %+v\n forced: %+v", normal, forced)
		}
	})

	t.Run("plan", func(t *testing.T) {
		normal := runFailedEmitPlanFixture(t, false)
		forced := runFailedEmitPlanFixture(t, true)
		if normal != forced {
			t.Fatalf("plan outcome differs with emission forced to fail:\n normal: %+v\n forced: %+v", normal, forced)
		}
	})

	t.Run("recovery", func(t *testing.T) {
		normal := runFailedEmitRecoveryFixture(t, false)
		forced := runFailedEmitRecoveryFixture(t, true)
		if normal != forced {
			t.Fatalf("recovery outcome differs with emission forced to fail:\n normal: %+v\n forced: %+v", normal, forced)
		}
	})
}

// buildFixtureOutcome/checkFixtureOutcome/planFixtureOutcome/recoveryFixtureOutcome
// are small comparable (all-value-field) structs capturing exactly the
// deterministic facts each fixture below asserts equal between a normal run
// and an emission-forced-to-fail run. Timestamps/durations are deliberately
// excluded -- those differ between any two real-clock runs regardless of
// live-event emission and would make this comparison meaningless.
type buildFixtureOutcome struct {
	DispatchCount int
	WaveCount     int
	DispatchMode  string
	PhaseStatus   string
}

func runFailedEmitBuildFixture(t *testing.T, forceFailure bool) buildFixtureOutcome {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)
	forceBuildJSONOutput(t)
	colonyLiveEmissionFailureOverride = forceFailure
	t.Cleanup(func() { colonyLiveEmissionFailureOverride = false })

	goal := "Prove a failed live emission never changes the build outcome"
	taskID := "1.1"
	accepted := createApprovedAcceptedBuildTestColony(t, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "full",
		CurrentPhase: 0,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:              1,
					Name:            "Emission-failure-proof build",
					Description:     "Prove a failing live emission changes nothing",
					Status:          colony.PhaseReady,
					Tasks:           []colony.Task{{ID: &taskID, Goal: "Wire the live build path", Status: colony.TaskPending}},
					SuccessCriteria: []string{"Build emits live events"},
				},
			},
		},
	})
	root := accepted.Root
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("failed to chdir to test root: %v", err)
	}
	defer os.Chdir(oldDir)

	rootCmd.SetArgs([]string{"build", "1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build returned error (forceFailure=%v): %v", forceFailure, err)
	}

	var envelope map[string]interface{}
	if err := json.Unmarshal(stdout.(*bytes.Buffer).Bytes(), &envelope); err != nil {
		t.Fatalf("failed to parse build output: %v", err)
	}
	result, _ := envelope["result"].(map[string]interface{})

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("failed to reload state: %v", err)
	}

	dispatchCount := 0
	if v, ok := result["dispatch_count"].(float64); ok {
		dispatchCount = int(v)
	}
	waveCount := 0
	if v, ok := result["wave_count"].(float64); ok {
		waveCount = int(v)
	}
	dispatchMode := ""
	if v, ok := result["dispatch_mode"].(string); ok {
		dispatchMode = v
	}

	return buildFixtureOutcome{
		DispatchCount: dispatchCount,
		WaveCount:     waveCount,
		DispatchMode:  dispatchMode,
		PhaseStatus:   string(state.Plan.Phases[0].Status),
	}
}

type checkFixtureOutcome struct {
	Advanced     bool
	ChecksPassed bool
	PhaseStatus  string
}

func runFailedEmitCheckFixture(t *testing.T, forceFailure bool) checkFixtureOutcome {
	t.Helper()
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)
	colonyLiveEmissionFailureOverride = forceFailure
	t.Cleanup(func() { colonyLiveEmissionFailureOverride = false })

	_, _ = continueLiveWiringFixture(t, "Prove a failed live emission never changes the check outcome")

	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error (forceFailure=%v): %v", forceFailure, err)
	}

	env := parseLifecycleEnvelope(t, stdout.(*bytes.Buffer).String())
	result, _ := env["result"].(map[string]interface{})
	advanced, _ := result["advanced"].(bool)
	verification, _ := result["verification"].(map[string]interface{})
	checksPassed, _ := verification["checks_passed"].(bool)

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("failed to reload state: %v", err)
	}

	return checkFixtureOutcome{
		Advanced:     advanced,
		ChecksPassed: checksPassed,
		PhaseStatus:  string(state.Plan.Phases[0].Status),
	}
}

type planFixtureOutcome struct {
	PhaseCount    int
	DispatchCount int
	ExistingPlan  bool
}

func runFailedEmitPlanFixture(t *testing.T, forceFailure bool) planFixtureOutcome {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)
	colonyLiveEmissionFailureOverride = forceFailure
	t.Cleanup(func() { colonyLiveEmissionFailureOverride = false })

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("failed to chdir to test root: %v", err)
	}
	defer os.Chdir(oldDir)

	goal := "Prove a failed live emission never changes the plan outcome"
	fixtureState := codexPlanSpecificationFixture(t, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	}, colony.SpecStatusApproved)
	createTestColonyState(t, dataDir, fixtureState)
	writeCodexPlanSpecificationProjection(t, root, fixtureState)

	rootCmd.SetArgs([]string{"colonize"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("colonize returned error: %v", err)
	}

	stdout = &bytes.Buffer{}
	rootCmd.SetArgs([]string{"plan", "--synthetic", "--preset", "balanced"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan returned error (forceFailure=%v): %v", forceFailure, err)
	}

	var envelope map[string]interface{}
	if err := json.Unmarshal(stdout.(*bytes.Buffer).Bytes(), &envelope); err != nil {
		t.Fatalf("failed to parse plan output: %v", err)
	}
	result, _ := envelope["result"].(map[string]interface{})
	phaseCount := 0
	if v, ok := result["count"].(float64); ok {
		phaseCount = int(v)
	}
	dispatches, _ := result["dispatches"].([]interface{})
	existingPlan, _ := result["existing_plan"].(bool)

	return planFixtureOutcome{
		PhaseCount:    phaseCount,
		DispatchCount: len(dispatches),
		ExistingPlan:  existingPlan,
	}
}

type recoveryFixtureOutcome struct {
	Action     string
	Exhausted  bool
	LogEntries int
}

func runFailedEmitRecoveryFixture(t *testing.T, forceFailure bool) recoveryFixtureOutcome {
	t.Helper()
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s
	colonyLiveEmissionFailureOverride = forceFailure
	t.Cleanup(func() { colonyLiveEmissionFailureOverride = false })

	const phaseNum = 203
	dispatches := []codexBuildDispatch{
		{Name: "Forge-recovery-2", Caste: "builder", TaskID: "1.1", Wave: 1, Status: "failed", Summary: "hit an unrecoverable-looking snag"},
	}
	instructions, err := buildExternalBuildRecoveryInstructions(phaseNum, dispatches)
	if err != nil {
		t.Fatalf("buildExternalBuildRecoveryInstructions (forceFailure=%v): %v", forceFailure, err)
	}
	if len(instructions) != 1 {
		t.Fatalf("expected exactly 1 recovery instruction, got %d", len(instructions))
	}
	action, _ := instructions[0]["action"].(string)
	exhausted, _ := instructions[0]["recovery_exhausted"].(bool)

	logFile, err := recoveryLogReadPhase(phaseNum)
	if err != nil {
		t.Fatalf("recoveryLogReadPhase: %v", err)
	}

	return recoveryFixtureOutcome{
		Action:     action,
		Exhausted:  exhausted,
		LogEntries: len(logFile.Entries),
	}
}
