package cmd

import (
	"encoding/json"
	"strings"
	"testing"

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
