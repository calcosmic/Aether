package cmd

import (
	"encoding/json"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
)

func TestCircuitBreakerTrippedCallsCeremonyEventBus(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	cb := NewCircuitBreaker(3)

	// Set up a build ceremony emitter so emitBuildCeremonyCircuitBreak has somewhere to go
	phase := colony.Phase{ID: 1, Name: "Test Phase"}
	emitter := &buildCeremonyEmitter{
		bus:       events.NewBus(s, events.DefaultConfig()),
		narrator:  &fakeCeremonyNarrator{},
		source:    "test",
		phaseID:   phase.ID,
		phaseName: phase.Name,
	}
	restore := setActiveBuildCeremony(emitter)
	defer restore()

	// Record 3 failures to trip the breaker
	cb.RecordFailure("Builder-01")
	cb.RecordFailure("Builder-01")
	tripped := cb.RecordFailure("Builder-01")
	if !tripped {
		t.Fatal("expected breaker to trip after 3 failures")
	}

	// Call the method on CircuitBreaker
	cb.emitCircuitBreakerTripped(phase, 1, "Builder-01")

	// Verify the event was published to the event bus
	persisted := readPersistedCeremonyEvents(t)
	found := false
	for _, evt := range persisted {
		if evt.Topic == events.CeremonyTopicBuildCircuitBreak {
			found = true
			var payload events.CeremonyPayload
			if err := json.Unmarshal(evt.Payload, &payload); err != nil {
				t.Fatalf("unmarshal payload: %v", err)
			}
			if payload.Name != "Builder-01" {
				t.Errorf("payload.Name = %q, want Builder-01", payload.Name)
			}
			if payload.Status != "tripped" {
				t.Errorf("payload.Status = %q, want tripped", payload.Status)
			}
			if payload.Phase != 1 {
				t.Errorf("payload.Phase = %d, want 1", payload.Phase)
			}
			break
		}
	}
	if !found {
		t.Fatal("expected circuit breaker ceremony event in persisted events")
	}
}

func TestCircuitBreakerRedistributedCallsCeremonyEventBus(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	phase := colony.Phase{ID: 2, Name: "Test Phase 2"}
	emitter := &buildCeremonyEmitter{
		bus:       events.NewBus(s, events.DefaultConfig()),
		narrator:  &fakeCeremonyNarrator{},
		source:    "test",
		phaseID:   phase.ID,
		phaseName: phase.Name,
	}
	restore := setActiveBuildCeremony(emitter)
	defer restore()

	emitCircuitBreakerRedistributed(phase, 1, "Builder-01", "Builder-02")

	persisted := readPersistedCeremonyEvents(t)
	found := false
	for _, evt := range persisted {
		if evt.Topic == events.CeremonyTopicBuildCircuitBreak {
			found = true
			var payload events.CeremonyPayload
			if err := json.Unmarshal(evt.Payload, &payload); err != nil {
				t.Fatalf("unmarshal payload: %v", err)
			}
			if payload.Name != "Builder-01" {
				t.Errorf("payload.Name = %q, want Builder-01", payload.Name)
			}
			if payload.Status != "skipped" {
				t.Errorf("payload.Status = %q, want skipped", payload.Status)
			}
			break
		}
	}
	if !found {
		t.Fatal("expected redistributed ceremony event in persisted events")
	}
}

func TestCircuitBreakerNoPeerCallsCeremonyEventBus(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	phase := colony.Phase{ID: 3, Name: "Test Phase 3"}
	emitter := &buildCeremonyEmitter{
		bus:       events.NewBus(s, events.DefaultConfig()),
		narrator:  &fakeCeremonyNarrator{},
		source:    "test",
		phaseID:   phase.ID,
		phaseName: phase.Name,
	}
	restore := setActiveBuildCeremony(emitter)
	defer restore()

	emitCircuitBreakerNoPeer(phase, 1, "Builder-01")

	persisted := readPersistedCeremonyEvents(t)
	found := false
	for _, evt := range persisted {
		if evt.Topic == events.CeremonyTopicBuildCircuitBreak {
			found = true
			var payload events.CeremonyPayload
			if err := json.Unmarshal(evt.Payload, &payload); err != nil {
				t.Fatalf("unmarshal payload: %v", err)
			}
			if payload.Name != "Builder-01" {
				t.Errorf("payload.Name = %q, want Builder-01", payload.Name)
			}
			if payload.Status != "skipped" {
				t.Errorf("payload.Status = %q, want skipped", payload.Status)
			}
			break
		}
	}
	if !found {
		t.Fatal("expected no-peer ceremony event in persisted events")
	}
}
