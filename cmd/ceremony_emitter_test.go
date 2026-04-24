package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
)

type fakeCeremonyNarrator struct {
	events []events.Event
	closed bool
}

func (f *fakeCeremonyNarrator) EmitEvent(evt events.Event) {
	f.events = append(f.events, evt)
}

func (f *fakeCeremonyNarrator) Close() {
	f.closed = true
}

func TestBuildCeremonyEmitterPersistsAndForwardsEvents(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	narrator := &fakeCeremonyNarrator{}
	emitter := &buildCeremonyEmitter{
		bus:       events.NewBus(s, events.DefaultConfig()),
		narrator:  narrator,
		source:    "unit-test",
		phaseID:   2,
		phaseName: "Narrator launcher",
	}
	emitter.Emit(events.CeremonyTopicBuildSpawn, events.CeremonyPayload{
		Caste:  "builder",
		Name:   "Mason-67",
		Status: "starting",
	})

	if len(narrator.events) != 1 {
		t.Fatalf("forwarded events = %d, want 1", len(narrator.events))
	}
	if narrator.events[0].Topic != events.CeremonyTopicBuildSpawn {
		t.Fatalf("forwarded topic = %q", narrator.events[0].Topic)
	}

	lines, err := s.ReadJSONL("event-bus.jsonl")
	if err != nil {
		t.Fatalf("read event bus: %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("persisted events = %d, want 1", len(lines))
	}
	var persisted events.Event
	if err := json.Unmarshal(lines[0], &persisted); err != nil {
		t.Fatalf("unmarshal persisted event: %v", err)
	}
	if persisted.ID != narrator.events[0].ID {
		t.Fatalf("narrator did not receive persisted event ID: got %q want %q", narrator.events[0].ID, persisted.ID)
	}
	var payload events.CeremonyPayload
	if err := json.Unmarshal(persisted.Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Phase != 2 || payload.PhaseName != "Narrator launcher" {
		t.Fatalf("payload missing phase defaults: %+v", payload)
	}
}

func TestBuildCeremonyEmitterTrimsUserControlledPayload(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	long := strings.Repeat("x", ceremonyTextLimit+50)
	many := make([]string, ceremonyListLimit+5)
	for i := range many {
		many[i] = long
	}
	emitter := &buildCeremonyEmitter{
		bus:       events.NewBus(s, events.DefaultConfig()),
		narrator:  &fakeCeremonyNarrator{},
		source:    "unit-test",
		phaseID:   1,
		phaseName: "Trim",
	}
	emitter.Emit(events.CeremonyTopicBuildSpawn, events.CeremonyPayload{
		Task:     long,
		Message:  long,
		Blockers: many,
	})

	lines, err := s.ReadJSONL("event-bus.jsonl")
	if err != nil {
		t.Fatalf("read event bus: %v", err)
	}
	var persisted events.Event
	if err := json.Unmarshal(lines[0], &persisted); err != nil {
		t.Fatalf("unmarshal persisted event: %v", err)
	}
	var payload events.CeremonyPayload
	if err := json.Unmarshal(persisted.Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if len(payload.Task) > ceremonyTextLimit || len(payload.Message) > ceremonyTextLimit {
		t.Fatalf("text fields were not trimmed: task=%d message=%d", len(payload.Task), len(payload.Message))
	}
	if len(payload.Blockers) != ceremonyListLimit {
		t.Fatalf("blockers length = %d, want %d", len(payload.Blockers), ceremonyListLimit)
	}
	for _, blocker := range payload.Blockers {
		if len(blocker) > ceremonyListItemLimit {
			t.Fatalf("blocker not trimmed: %d", len(blocker))
		}
	}
}

func TestLifecycleCeremonyPersistsTrimmedEvent(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	long := strings.Repeat("x", ceremonyTextLimit+50)
	emitLifecycleCeremony(events.CeremonyTopicPheromoneEmit, events.CeremonyPayload{
		PheromoneType: "FOCUS",
		Strength:      0.8,
		Status:        "created",
		Message:       long,
	}, "unit-test")

	lines, err := s.ReadJSONL("event-bus.jsonl")
	if err != nil {
		t.Fatalf("read event bus: %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("persisted events = %d, want 1", len(lines))
	}
	var persisted events.Event
	if err := json.Unmarshal(lines[0], &persisted); err != nil {
		t.Fatalf("unmarshal persisted event: %v", err)
	}
	if persisted.Topic != events.CeremonyTopicPheromoneEmit {
		t.Fatalf("topic = %q, want %q", persisted.Topic, events.CeremonyTopicPheromoneEmit)
	}
	if persisted.Source != "unit-test" {
		t.Fatalf("source = %q, want unit-test", persisted.Source)
	}
	var payload events.CeremonyPayload
	if err := json.Unmarshal(persisted.Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.PheromoneType != "FOCUS" || payload.Status != "created" || payload.Strength != 0.8 {
		t.Fatalf("payload = %+v", payload)
	}
	if len(payload.Message) > ceremonyTextLimit {
		t.Fatalf("message was not trimmed: %d", len(payload.Message))
	}
}

func TestPheromoneWriteEmitsCeremonyEvent(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, _ := newTestStore(t)
	store = s
	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"pheromone-write", "--type", "FOCUS", "--content", "Surface lifecycle context", "--strength", "0.75"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pheromone-write returned error: %v", err)
	}

	lines, err := s.ReadJSONL("event-bus.jsonl")
	if err != nil {
		t.Fatalf("read event bus: %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("persisted events = %d, want 1", len(lines))
	}
	var persisted events.Event
	if err := json.Unmarshal(lines[0], &persisted); err != nil {
		t.Fatalf("unmarshal persisted event: %v", err)
	}
	if persisted.Topic != events.CeremonyTopicPheromoneEmit {
		t.Fatalf("topic = %q, want %q", persisted.Topic, events.CeremonyTopicPheromoneEmit)
	}
	var payload events.CeremonyPayload
	if err := json.Unmarshal(persisted.Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.PheromoneType != "FOCUS" || payload.Status != "created" || payload.Strength != 0.75 {
		t.Fatalf("payload = %+v", payload)
	}
	if payload.Message != "Surface lifecycle context" {
		t.Fatalf("message = %q", payload.Message)
	}
}

func TestSealEmitsChamberCeremonyEvent(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, _ := newTestStore(t)
	store = s
	var buf bytes.Buffer
	stdout = &buf

	goal := "Seal ceremony events"
	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:     1,
			Name:   "Complete work",
			Status: colony.PhaseCompleted,
		}}},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	rootCmd.SetArgs([]string{"seal"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("seal returned error: %v", err)
	}

	lines, err := s.ReadJSONL("event-bus.jsonl")
	if err != nil {
		t.Fatalf("read event bus: %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("persisted events = %d, want 1", len(lines))
	}
	var persisted events.Event
	if err := json.Unmarshal(lines[0], &persisted); err != nil {
		t.Fatalf("unmarshal persisted event: %v", err)
	}
	if persisted.Topic != events.CeremonyTopicChamberSeal {
		t.Fatalf("topic = %q, want %q", persisted.Topic, events.CeremonyTopicChamberSeal)
	}
	var payload events.CeremonyPayload
	if err := json.Unmarshal(persisted.Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Status != "sealed" || payload.PhaseName != "Crowned Anthill" || payload.Completed != 1 || payload.Total != 1 {
		t.Fatalf("payload = %+v", payload)
	}
}

func TestPlanEmitsLifecycleCeremonyEvents(t *testing.T) {
	saveGlobals(t)
	s, root := newTestStore(t)
	store = s
	goal := "Plan ceremony events"
	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 0,
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	if _, err := runCodexPlanWithOptions(root, codexPlanOptions{Synthetic: true}); err != nil {
		t.Fatalf("plan returned error: %v", err)
	}

	assertCeremonyTopics(t, readPersistedCeremonyEvents(t),
		events.CeremonyTopicPlanWaveStart,
		events.CeremonyTopicPlanSpawn,
		events.CeremonyTopicPlanWaveEnd,
	)
}

func TestColonizeEmitsLifecycleCeremonyEvents(t *testing.T) {
	saveGlobals(t)
	s, root := newTestStore(t)
	store = s

	if _, err := runCodexColonizeWithOptions(root, codexColonizeOptions{}); err != nil {
		t.Fatalf("colonize returned error: %v", err)
	}

	assertCeremonyTopics(t, readPersistedCeremonyEvents(t),
		events.CeremonyTopicColonizeWaveStart,
		events.CeremonyTopicColonizeSpawn,
		events.CeremonyTopicColonizeWaveEnd,
	)
}

func TestContinueEmitsLifecycleCeremonyEvents(t *testing.T) {
	saveGlobals(t)
	s, root := newTestStore(t)
	store = s
	now := time.Now().UTC()
	goal := "Continue ceremony events"
	taskID := "1.1"
	state := colony.ColonyState{
		Version:        "3.0",
		Goal:           &goal,
		State:          colony.StateBUILT,
		CurrentPhase:   1,
		BuildStartedAt: &now,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:     1,
			Name:   "Continue hooks",
			Status: colony.PhaseInProgress,
			Tasks:  []colony.Task{{ID: &taskID, Goal: "Emit continue ceremony events", Status: colony.TaskPending}},
		}}},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save state: %v", err)
	}
	seedContinueBuildPacket(t, s.BasePath(), 1, "Continue hooks", goal, []codexBuildDispatch{{
		Stage:  "implementation",
		Caste:  "builder",
		Name:   "Mason-67",
		TaskID: taskID,
		Task:   "Emit continue ceremony events",
		Status: "completed",
	}})
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return &continueUnavailableInvoker{} }

	if _, _, _, _, _, _, err := runCodexContinue(root, codexContinueOptions{}); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	assertCeremonyTopics(t, readPersistedCeremonyEvents(t),
		events.CeremonyTopicContinueWaveStart,
		events.CeremonyTopicContinueSpawn,
		events.CeremonyTopicContinueWaveEnd,
	)
}

func TestResolveSkillSectionEmitsSkillActivationCeremony(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	tmpDir := t.TempDir()
	hubDir := tmpDir + "/hub"
	skillsDir := hubDir + "/skills/colony/test-skill"
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		t.Fatalf("failed to create skill dir: %v", err)
	}
	skillContent := "---\nname: test-skill\ntype: colony\ncategory: testing\nagent_roles:\n  - builder\n---\nThis is the test skill content."
	if err := os.WriteFile(filepath.Join(skillsDir, "SKILL.md"), []byte(skillContent), 0644); err != nil {
		t.Fatalf("failed to write skill: %v", err)
	}
	os.Setenv("AETHER_HUB_DIR", hubDir)
	t.Cleanup(func() { os.Unsetenv("AETHER_HUB_DIR") })

	section := resolveSkillSection("builder", "testing task")
	if section == "" {
		t.Fatal("expected matched skill section")
	}

	persisted := readPersistedCeremonyEvents(t)
	assertCeremonyTopics(t, persisted, events.CeremonyTopicSkillActivate)
	var payload events.CeremonyPayload
	if err := json.Unmarshal(persisted[0].Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Skill != "test-skill" || payload.Status != "activated" {
		t.Fatalf("payload = %+v", payload)
	}
	if !strings.Contains(payload.Message, "builder") {
		t.Fatalf("message = %q, want builder context", payload.Message)
	}
}

func TestEntombEmitsChamberEntombCeremonyEvent(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	aetherRoot := os.Getenv("AETHER_ROOT")
	var buf bytes.Buffer
	stdout = &buf

	goal := "Entomb ceremony events"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateCOMPLETED,
		CurrentPhase: 1,
		Milestone:    "Crowned Anthill",
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:     1,
			Name:   "Archive",
			Status: colony.PhaseCompleted,
		}}},
	})

	for path, content := range map[string]string{
		filepath.Join(aetherRoot, ".aether", "CROWNED-ANTHILL.md"): "# Crowned Anthill\n",
		filepath.Join(aetherRoot, ".aether", "HANDOFF.md"):         "# Old handoff\n",
		filepath.Join(aetherRoot, ".aether", "CONTEXT.md"):         "# Old context\n",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("create parent for %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("write fixture %s: %v", path, err)
		}
	}

	rootCmd.SetArgs([]string{"entomb"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("entomb returned error: %v", err)
	}

	persisted := readPersistedCeremonyEvents(t)
	assertCeremonyTopics(t, persisted, events.CeremonyTopicChamberEntomb)
	var payload events.CeremonyPayload
	if err := json.Unmarshal(persisted[len(persisted)-1].Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Status != "entombed" || payload.Message != goal || payload.Completed != 1 || payload.Total != 1 {
		t.Fatalf("payload = %+v", payload)
	}
	if payload.TaskID == "" || payload.Task == "" {
		t.Fatalf("payload missing chamber context: %+v", payload)
	}
}

func TestMiddenWriteEmitsCeremonyEvent(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, _ := newTestStore(t)
	store = s
	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"midden-write", "--category", "testing", "--message", "test failure", "--source", "unit"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("midden-write returned error: %v", err)
	}

	persisted := readPersistedCeremonyEvents(t)
	assertCeremonyTopics(t, persisted, events.CeremonyTopicMiddenRecord)
	var payload events.CeremonyPayload
	if err := json.Unmarshal(persisted[0].Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Status != "recorded" || payload.Task != "testing" || payload.Message != "test failure" || payload.TaskID == "" {
		t.Fatalf("payload = %+v", payload)
	}
}

func TestQueenPromoteEmitsCeremonyEvent(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	store = s
	var buf bytes.Buffer
	stdout = &buf

	hubDir := filepath.Join(tmpDir, "hub")
	origHub := os.Getenv("AETHER_HUB_DIR")
	os.Setenv("AETHER_HUB_DIR", hubDir)
	t.Cleanup(func() { os.Setenv("AETHER_HUB_DIR", origHub) })

	rootCmd.SetArgs([]string{"queen-promote", "pattern", "Prefer focused ceremony events"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("queen-promote returned error: %v", err)
	}

	persisted := readPersistedCeremonyEvents(t)
	assertCeremonyTopics(t, persisted, events.CeremonyTopicQueenPromote)
	var payload events.CeremonyPayload
	if err := json.Unmarshal(persisted[0].Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Status != "promoted" || payload.Task != "Patterns" || payload.Message != "Prefer focused ceremony events" {
		t.Fatalf("payload = %+v", payload)
	}
}

func TestHiveStoreAndPromoteEmitCeremonyEvents(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	store = s
	var buf bytes.Buffer
	stdout = &buf

	hubDir := filepath.Join(tmpDir, "hub")
	origHub := os.Getenv("AETHER_HUB_DIR")
	os.Setenv("AETHER_HUB_DIR", hubDir)
	t.Cleanup(func() { os.Setenv("AETHER_HUB_DIR", origHub) })

	rootCmd.SetArgs([]string{"hive-store", "Keep docs aligned", "docs", "aether"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("hive-store returned error: %v", err)
	}
	rootCmd.SetArgs([]string{"hive-promote", "--text", "Prefer focused fixes", "--source-repo", "aether", "--confidence", "0.9"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("hive-promote returned error: %v", err)
	}

	persisted := readPersistedCeremonyEvents(t)
	assertCeremonyTopics(t, persisted, events.CeremonyTopicHiveStore, events.CeremonyTopicHivePromote)
}

func TestActiveBuildCeremonyScopeRestoresPreviousEmitter(t *testing.T) {
	saveGlobals(t)
	outer := &buildCeremonyEmitter{phaseID: 1, phaseName: "outer"}
	inner := &buildCeremonyEmitter{phaseID: 2, phaseName: "inner"}

	restoreOuter := setActiveBuildCeremony(outer)
	if currentBuildCeremony() != outer {
		t.Fatal("outer emitter not active")
	}
	restoreInner := setActiveBuildCeremony(inner)
	if currentBuildCeremony() != inner {
		t.Fatal("inner emitter not active")
	}
	restoreInner()
	if currentBuildCeremony() != outer {
		t.Fatal("outer emitter was not restored")
	}
	restoreOuter()
	if currentBuildCeremony() != nil {
		t.Fatal("active emitter was not cleared")
	}
}

func readPersistedCeremonyEvents(t *testing.T) []events.Event {
	t.Helper()
	lines, err := store.ReadJSONL("event-bus.jsonl")
	if err != nil {
		t.Fatalf("read event bus: %v", err)
	}
	persisted := make([]events.Event, 0, len(lines))
	for _, line := range lines {
		var evt events.Event
		if err := json.Unmarshal(line, &evt); err != nil {
			t.Fatalf("unmarshal event: %v", err)
		}
		persisted = append(persisted, evt)
	}
	return persisted
}

func assertCeremonyTopics(t *testing.T, persisted []events.Event, wants ...string) {
	t.Helper()
	seen := map[string]bool{}
	for _, evt := range persisted {
		seen[evt.Topic] = true
	}
	for _, want := range wants {
		if !seen[want] {
			t.Fatalf("missing ceremony topic %q in %+v", want, persisted)
		}
	}
}

func testBuildState(goal, taskID string) colony.ColonyState {
	return colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 0,
		ColonyDepth:  "light",
		Plan: colony.Plan{
			Phases: []colony.Phase{{
				ID:              1,
				Name:            "Narrator launcher",
				Status:          colony.PhaseReady,
				Tasks:           []colony.Task{{ID: &taskID, Goal: "Keep JSON output clean", Status: colony.TaskPending}},
				SuccessCriteria: []string{"JSON output remains parseable"},
			}},
		},
	}
}
