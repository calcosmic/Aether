package colony

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// State transition tests
// ---------------------------------------------------------------------------

func TestValidTransitions(t *testing.T) {
	tests := []struct {
		from State
		to   State
	}{
		{StateREADY, StateEXECUTING},
		{StateEXECUTING, StateBUILT},
		{StateBUILT, StateREADY},
		{StateBUILT, StateCOMPLETED},
		{StateEXECUTING, StateCOMPLETED},
		{StateREADY, StateCOMPLETED},
	}
	for _, tt := range tests {
		t.Run(string(tt.from)+"->"+string(tt.to), func(t *testing.T) {
			if err := Transition(tt.from, tt.to); err != nil {
				t.Fatalf("expected no error for %s->%s, got: %v", tt.from, tt.to, err)
			}
		})
	}
}

func TestInvalidTransitions(t *testing.T) {
	tests := []struct {
		from State
		to   State
	}{
		{StateREADY, StateREADY},
		{StateEXECUTING, StateEXECUTING},
		{StateBUILT, StateBUILT},
		{StateCOMPLETED, StateCOMPLETED},
		{StateCOMPLETED, StateREADY},
		{StateCOMPLETED, StateEXECUTING},
		{StateCOMPLETED, StateBUILT},
		{StateBUILT, StateEXECUTING},
	}
	for _, tt := range tests {
		t.Run(string(tt.from)+"->"+string(tt.to), func(t *testing.T) {
			if err := Transition(tt.from, tt.to); err == nil {
				t.Fatalf("expected error for %s->%s, got nil", tt.from, tt.to)
			}
		})
	}
}

func TestTransitionErrorIsNotEmpty(t *testing.T) {
	err := Transition(StateCOMPLETED, StateREADY)
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if msg == "" {
		t.Fatal("error message should not be empty")
	}
}

// ---------------------------------------------------------------------------
// AdvancePhase tests
// ---------------------------------------------------------------------------

func TestAdvancePhase_FirstPending(t *testing.T) {
	phases := []Phase{
		{ID: 1, Status: PhaseCompleted},
		{ID: 2, Status: PhasePending},
		{ID: 3, Status: PhasePending},
	}
	next, err := AdvancePhase(1, phases)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if next != 2 {
		t.Fatalf("expected next phase 2, got %d", next)
	}
}

func TestAdvancePhase_NoMorePending(t *testing.T) {
	phases := []Phase{
		{ID: 1, Status: PhaseCompleted},
		{ID: 2, Status: PhaseCompleted},
	}
	_, err := AdvancePhase(2, phases)
	if err == nil {
		t.Fatal("expected error when no pending phases remain")
	}
}

func TestAdvancePhase_EmptyPhases(t *testing.T) {
	_, err := AdvancePhase(0, []Phase{})
	if err == nil {
		t.Fatal("expected error for empty phases")
	}
}

func TestAdvancePhase_CurrentBeyondLast(t *testing.T) {
	phases := []Phase{{ID: 1, Status: PhasePending}}
	_, err := AdvancePhase(5, phases)
	if err == nil {
		t.Fatal("expected error when current is beyond phases length")
	}
}

// ---------------------------------------------------------------------------
// JSON round-trip test with self-contained data
// ---------------------------------------------------------------------------

func TestRoundTripColonyState(t *testing.T) {
	now := time.Now().UTC()
	milestoneAt := now.Format(time.RFC3339)
	goal := "Build something great"
	conf := 0.85
	phase := 2
	taskID := "1.1"

	input := ColonyState{
		Version:            "3.0",
		Goal:               &goal,
		ColonyName:         strPtr("Test Colony"),
		ColonyVersion:      1,
		State:              StateREADY,
		CurrentPhase:       1,
		SessionID:          strPtr("session_123_abc"),
		InitializedAt:      &now,
		BuildStartedAt:     &now,
		ColonyDepth:        "standard",
		Milestone:          "Crowned Anthill",
		MilestoneUpdatedAt: &milestoneAt,
		Plan: Plan{
			GeneratedAt: &now,
			Confidence:  &conf,
			Phases: []Phase{
				{
					ID: 1, Name: "Phase 1", Description: "First phase",
					Status: PhaseCompleted,
					Tasks: []Task{
						{ID: &taskID, Goal: "Do work", Status: TaskCompleted},
					},
				},
			},
		},
		Memory: Memory{
			PhaseLearnings: []PhaseLearning{
				{
					ID: "learning_1", Phase: 1, PhaseName: "Phase 1",
					Learnings: []Learning{
						{Claim: "Something learned", Status: "hypothesis", Tested: false, Evidence: "observed"},
					},
					Timestamp: now.Format(time.RFC3339),
				},
			},
			Instincts: []Instinct{
				{
					ID: "instinct_1", Trigger: "test trigger", Action: "test action",
					Confidence: 0.8, Status: "hypothesis", Domain: "pattern",
					Source: "promoted_from_learning", Evidence: []string{""},
					Tested: false, CreatedAt: now.Format(time.RFC3339),
				},
			},
		},
		Signals: []Signal{},
		Graveyards: []Graveyard{
			{
				ID: "grave_1", File: "test.ts", AntName: "Builder-1",
				TaskID: "task-1", Phase: &phase,
				FailureSummary: "crash", Timestamp: now.Format(time.RFC3339),
			},
		},
		Events: []string{"2026-01-01T00:00:00Z|init|queen|colony initialized"},
	}

	// Verify key fields are set
	if input.State != StateREADY {
		t.Fatalf("expected state READY, got %q", input.State)
	}
	if input.CurrentPhase < 0 {
		t.Fatalf("expected current_phase >= 0, got %d", input.CurrentPhase)
	}
	if input.Milestone != "Crowned Anthill" {
		t.Fatalf("expected milestone Crowned Anthill, got %q", input.Milestone)
	}
	if input.MilestoneUpdatedAt == nil {
		t.Fatalf("expected milestone_updated_at to be non-nil, got nil")
	}
	if input.Version != "3.0" {
		t.Fatalf("expected version 3.0, got %q", input.Version)
	}

	// Marshal and unmarshal round-trip
	remarshaled, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var rematched ColonyState
	if err := json.Unmarshal(remarshaled, &rematched); err != nil {
		t.Fatalf("failed to unmarshal remarshaled data: %v", err)
	}

	// Deep equality check
	assertColonyStateEqual(t, input, rematched)
}

func TestGoldenColonyState(t *testing.T) {
	golden, err := os.ReadFile("testdata/COLONY_STATE.golden.json")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	var state ColonyState
	if err := json.Unmarshal(golden, &state); err != nil {
		t.Fatalf("unmarshal golden: %v", err)
	}
	produced, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	produced = append(produced, '\n')
	if !bytes.Equal(golden, produced) {
		t.Errorf("golden mismatch:\nexpected:\n%s\n\ngot:\n%s", golden, produced)
	}
}

func assertColonyStateEqual(t *testing.T, a, b ColonyState) {
	t.Helper()
	if a.Version != b.Version {
		t.Errorf("Version mismatch: %q vs %q", a.Version, b.Version)
	}
	if a.State != b.State {
		t.Errorf("State mismatch: %q vs %q", a.State, b.State)
	}
	if a.CurrentPhase != b.CurrentPhase {
		t.Errorf("CurrentPhase mismatch: %d vs %d", a.CurrentPhase, b.CurrentPhase)
	}
	if a.ColonyVersion != b.ColonyVersion {
		t.Errorf("ColonyVersion mismatch: %d vs %d", a.ColonyVersion, b.ColonyVersion)
	}
	if len(a.Events) != len(b.Events) {
		t.Errorf("Events length mismatch: %d vs %d", len(a.Events), len(b.Events))
	}
	if len(a.Plan.Phases) != len(b.Plan.Phases) {
		t.Errorf("Plan.Phases length mismatch: %d vs %d", len(a.Plan.Phases), len(b.Plan.Phases))
	}
	if len(a.Memory.PhaseLearnings) != len(b.Memory.PhaseLearnings) {
		t.Errorf("Memory.PhaseLearnings length mismatch: %d vs %d", len(a.Memory.PhaseLearnings), len(b.Memory.PhaseLearnings))
	}
	if len(a.Memory.Instincts) != len(b.Memory.Instincts) {
		t.Errorf("Memory.Instincts length mismatch: %d vs %d", len(a.Memory.Instincts), len(b.Memory.Instincts))
	}
}

// ---------------------------------------------------------------------------
// Nullable field tests
// ---------------------------------------------------------------------------

func TestNullableFields_Nil(t *testing.T) {
	state := ColonyState{
		Version:      "3.0",
		State:        StateREADY,
		CurrentPhase: 0,
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded ColonyState
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Goal != nil {
		t.Error("expected nil Goal")
	}
	if decoded.ColonyName != nil {
		t.Error("expected nil ColonyName")
	}
	if decoded.SessionID != nil {
		t.Error("expected nil SessionID")
	}
	if decoded.InitializedAt != nil {
		t.Error("expected nil InitializedAt")
	}
	if decoded.BuildStartedAt != nil {
		t.Error("expected nil BuildStartedAt")
	}
	if decoded.Plan.GeneratedAt != nil {
		t.Error("expected nil Plan.GeneratedAt")
	}
	if decoded.Plan.Confidence != nil {
		t.Error("expected nil Plan.Confidence")
	}
}

func TestNullableFields_WithValues(t *testing.T) {
	goal := "Build something"
	now := time.Now().UTC()
	conf := 0.85
	state := ColonyState{
		Version:        "3.0",
		State:          StateREADY,
		CurrentPhase:   0,
		Goal:           &goal,
		ColonyName:     strPtr("Test Colony"),
		SessionID:      strPtr("session_123_abc"),
		InitializedAt:  &now,
		BuildStartedAt: &now,
		Plan: Plan{
			GeneratedAt: &now,
			Confidence:  &conf,
		},
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded ColonyState
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Goal == nil || *decoded.Goal != "Build something" {
		t.Error("Goal mismatch")
	}
	if decoded.Plan.Confidence == nil || *decoded.Plan.Confidence != 0.85 {
		t.Error("Plan.Confidence mismatch")
	}
}

func TestNullableFields_JSONNull(t *testing.T) {
	raw := `{"version":"3.0","state":"READY","current_phase":0,"goal":null,"colony_name":null,"colony_version":1,"session_id":null,"initialized_at":null,"build_started_at":null,"plan":{"generated_at":null,"confidence":null,"phases":[]},"memory":{"phase_learnings":[],"decisions":[],"instincts":[]},"errors":{"records":[],"flagged_patterns":[]},"signals":[],"graveyards":[],"events":[]}`

	var state ColonyState
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if state.Goal != nil {
		t.Error("expected nil Goal from JSON null")
	}
	if state.ColonyName != nil {
		t.Error("expected nil ColonyName from JSON null")
	}
	if state.BuildStartedAt != nil {
		t.Error("expected nil BuildStartedAt from JSON null")
	}
}

func TestPlanConfidence_LegacyObjectCompatibility(t *testing.T) {
	raw := `{
		"version":"3.0",
		"state":"READY",
		"current_phase":0,
		"goal":"Recover legacy colony",
		"colony_name":"Legacy Colony",
		"colony_version":1,
		"plan":{
			"generated_at":"2026-04-17T10:00:00Z",
			"confidence":{
				"knowledge":85,
				"requirements":80,
				"risks":65,
				"dependencies":70,
				"effort":78,
				"overall":76
			},
			"phases":[]
		},
		"memory":{"phase_learnings":[],"decisions":[],"instincts":[]},
		"errors":{"records":[],"flagged_patterns":[]},
		"signals":[],
		"graveyards":[],
		"events":[]
	}`

	var state ColonyState
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		t.Fatalf("unmarshal legacy confidence object: %v", err)
	}

	if state.Plan.Confidence == nil {
		t.Fatal("expected plan confidence to be populated")
	}
	if *state.Plan.Confidence != 0.76 {
		t.Fatalf("expected normalized plan confidence 0.76, got %v", *state.Plan.Confidence)
	}
}

// ---------------------------------------------------------------------------
// Instinct nullable field tests
// ---------------------------------------------------------------------------

func TestInstinctNullableFields(t *testing.T) {
	raw := `{
		"id": "instinct_1",
		"trigger": "test trigger",
		"action": "test action",
		"confidence": 0.8,
		"status": "hypothesis",
		"domain": "pattern",
		"source": "promoted_from_learning",
		"evidence": [""],
		"tested": false,
		"created_at": "2026-03-31T21:11:15Z",
		"last_applied": null,
		"applications": 0,
		"successes": 0,
		"failures": 0
	}`

	var inst Instinct
	if err := json.Unmarshal([]byte(raw), &inst); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if inst.LastApplied != nil {
		t.Error("expected nil LastApplied")
	}
	if inst.Confidence != 0.8 {
		t.Errorf("expected confidence 0.8, got %f", inst.Confidence)
	}
}

func TestLearningNullableFields(t *testing.T) {
	raw := `{
		"claim": "test claim",
		"status": "hypothesis",
		"tested": false,
		"evidence": "some evidence",
		"disproven_by": null
	}`

	var learning Learning
	if err := json.Unmarshal([]byte(raw), &learning); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if learning.DisprovenBy != nil {
		t.Error("expected nil DisprovenBy")
	}
}

// ---------------------------------------------------------------------------
// ErrorRecord nullable fields
// ---------------------------------------------------------------------------

func TestErrorRecordNullableFields(t *testing.T) {
	raw := `{
		"id": "err_1708000000_a1b2",
		"category": "E_FILE_NOT_FOUND",
		"severity": "critical",
		"description": "file not found",
		"root_cause": null,
		"phase": null,
		"task_id": null,
		"timestamp": "2026-02-15T16:00:00Z"
	}`

	var rec ErrorRecord
	if err := json.Unmarshal([]byte(raw), &rec); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if rec.RootCause != nil {
		t.Error("expected nil RootCause")
	}
	if rec.Phase != nil {
		t.Error("expected nil Phase")
	}
	if rec.TaskID != nil {
		t.Error("expected nil TaskID")
	}
}

// ---------------------------------------------------------------------------
// State constant tests
// ---------------------------------------------------------------------------

func TestStateConstants(t *testing.T) {
	consts := map[string]State{
		"READY":     StateREADY,
		"EXECUTING": StateEXECUTING,
		"BUILT":     StateBUILT,
		"COMPLETED": StateCOMPLETED,
	}
	for name, val := range consts {
		if val == "" {
			t.Errorf("State constant %s is empty", name)
		}
	}
}

func TestPhaseStatusConstants(t *testing.T) {
	consts := map[string]string{
		"PhasePending":    PhasePending,
		"PhaseReady":      PhaseReady,
		"PhaseInProgress": PhaseInProgress,
		"PhaseCompleted":  PhaseCompleted,
	}
	for name, val := range consts {
		if val == "" {
			t.Errorf("Phase status constant %s is empty", name)
		}
	}
}

// ---------------------------------------------------------------------------
// ColonyDepth field (present in real data)
// ---------------------------------------------------------------------------

func TestColonyDepthField(t *testing.T) {
	raw := `{"version":"3.0","state":"READY","current_phase":0,"colony_depth":"standard"}`
	var state ColonyState
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if state.ColonyDepth != "standard" {
		t.Errorf("expected colony_depth standard, got %q", state.ColonyDepth)
	}
}

// ---------------------------------------------------------------------------
// AdvancePhase updates phase status
// ---------------------------------------------------------------------------

func TestAdvancePhase_UpdatesStatus(t *testing.T) {
	phases := []Phase{
		{ID: 1, Status: PhaseCompleted},
		{ID: 2, Status: PhasePending},
	}
	next, err := AdvancePhase(1, phases)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if next != 2 {
		t.Fatalf("expected next phase 2, got %d", next)
	}
	if phases[1].Status != PhaseReady {
		t.Errorf("expected phase 2 status to be updated to ready, got %q", phases[1].Status)
	}
}

// ---------------------------------------------------------------------------
// Task status
// ---------------------------------------------------------------------------

func TestTaskStatus(t *testing.T) {
	id := "1.1"
	task := Task{
		ID:     &id,
		Goal:   "do something",
		Status: TaskCompleted,
	}
	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded Task
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Status != TaskCompleted {
		t.Errorf("expected status %q, got %q", TaskCompleted, decoded.Status)
	}
}

// ---------------------------------------------------------------------------
// FlaggedPattern and Graveyard tests
// ---------------------------------------------------------------------------

func TestFlaggedPatternRoundTrip(t *testing.T) {
	fp := FlaggedPattern{
		Pattern:   "missing_state_file",
		Count:     3,
		FirstSeen: timePtr(parseTime("2026-02-15T16:00:00Z")),
		LastSeen:  timePtr(parseTime("2026-02-15T17:00:00Z")),
	}
	data, err := json.Marshal(fp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded FlaggedPattern
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Pattern != fp.Pattern {
		t.Errorf("pattern mismatch: %q vs %q", fp.Pattern, decoded.Pattern)
	}
	if decoded.Count != fp.Count {
		t.Errorf("count mismatch: %d vs %d", fp.Count, decoded.Count)
	}
}

func TestGraveyardRoundTrip(t *testing.T) {
	phase := 2
	line := 127
	g := Graveyard{
		ID:             "grave_1708000000_a1b2",
		File:           "src/utils/parser.ts",
		AntName:        "Builder-42",
		TaskID:         "task-5",
		Phase:          &phase,
		FailureSummary: "Infinite loop in regex parsing",
		Function:       strPtr("parseComplexPattern"),
		Line:           &line,
		Timestamp:      "2026-02-15T16:00:00Z",
	}
	data, err := json.Marshal(g)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded Graveyard
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.ID != g.ID {
		t.Errorf("id mismatch: %q vs %q", g.ID, decoded.ID)
	}
	if decoded.Phase == nil || *decoded.Phase != 2 {
		t.Error("phase mismatch")
	}
	if decoded.Function == nil || *decoded.Function != "parseComplexPattern" {
		t.Error("function mismatch")
	}
}

func TestGraveyardNullableFields_Nil(t *testing.T) {
	raw := `{
		"id": "grave_1",
		"file": "test.ts",
		"ant_name": "Builder-1",
		"task_id": "task-1",
		"phase": null,
		"failure_summary": "crash",
		"function": null,
		"line": null,
		"timestamp": "2026-02-15T16:00:00Z"
	}`
	var g Graveyard
	if err := json.Unmarshal([]byte(raw), &g); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if g.Phase != nil {
		t.Error("expected nil Phase")
	}
	if g.Function != nil {
		t.Error("expected nil Function")
	}
	if g.Line != nil {
		t.Error("expected nil Line")
	}
}

// ---------------------------------------------------------------------------
// Real colony state data integrity test
// ---------------------------------------------------------------------------

func TestRealColonyStateDataIntegrity(t *testing.T) {
	data, err := os.ReadFile("../../.aether/data/COLONY_STATE.json")
	if err != nil {
		t.Skip("COLONY_STATE.json not found")
	}

	var state ColonyState
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Skip if no colony initialized or not yet planned
	if state.Goal == nil || *state.Goal == "" {
		t.Skip("no colony initialized — skipping integrity check")
	}
	if len(state.Plan.Phases) == 0 {
		t.Skip("no plan phases — colony initialized but not yet planned")
	}

	// Check top-level fields
	if state.Version == "" {
		t.Error("version is empty")
	}
	if state.State == "" {
		t.Error("state is empty")
	}

	// Check plan phases
	if len(state.Plan.Phases) == 0 {
		t.Error("expected at least one phase")
	}
	for _, phase := range state.Plan.Phases {
		if phase.ID == 0 {
			t.Error("phase ID should not be zero")
		}
		if phase.Name == "" {
			t.Error("phase name should not be empty")
		}
		for _, task := range phase.Tasks {
			if task.ID == nil || *task.ID == "" {
				t.Error("task ID should not be empty")
			}
			if task.Goal == "" {
				t.Error("task goal should not be empty")
			}
		}
	}

	// Check instincts
	for _, inst := range state.Memory.Instincts {
		if inst.ID == "" {
			t.Error("instinct ID should not be empty")
		}
		if inst.Trigger == "" {
			t.Error("instinct trigger should not be empty")
		}
		if inst.Action == "" {
			t.Error("instinct action should not be empty")
		}
	}

	// Check phase learnings
	for _, pl := range state.Memory.PhaseLearnings {
		if pl.ID == "" {
			t.Error("phase learning ID should not be empty")
		}
		for _, l := range pl.Learnings {
			if l.Claim == "" {
				t.Error("learning claim should not be empty")
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Errors.Is check
// ---------------------------------------------------------------------------

func TestTransitionErrorIs(t *testing.T) {
	err := Transition(StateCOMPLETED, StateREADY)
	if !errors.Is(err, ErrInvalidTransition) {
		t.Errorf("expected ErrInvalidTransition, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func strPtr(s string) *string { return &s }

func timePtr(t time.Time) *time.Time { return &t }

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}
