package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/events"
)

// --- Task 1: RED tests for emitLoopBreakEvent calls at five loop-break points ---

func TestContinueWatcherAutoSkipEmitsLoopBreak(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	// Simulate the watcher auto-skip path by directly calling emitLoopBreakEvent
	// with the same arguments that codex_continue.go will use at the watcher auto-skip point.
	emitLoopBreakEvent("watcher_skip",
		"3 consecutive watcher failures",
		"auto-skipped watcher, advancing on runtime verification",
		"aether-continue")

	persisted := readPersistedCeremonyEvents(t)
	found := false
	for _, evt := range persisted {
		if evt.Topic == events.CeremonyTopicLoopBreak {
			found = true
			var payload events.CeremonyPayload
			if err := json.Unmarshal(evt.Payload, &payload); err != nil {
				t.Fatalf("unmarshal payload: %v", err)
			}
			if payload.LoopType != "watcher_skip" {
				t.Errorf("LoopType = %q, want %q", payload.LoopType, "watcher_skip")
			}
			if !strings.Contains(payload.DetectionSignal, "consecutive watcher failures") {
				t.Errorf("DetectionSignal = %q, want to contain 'consecutive watcher failures'", payload.DetectionSignal)
			}
			if !strings.Contains(payload.ActionTaken, "auto-skipped watcher") {
				t.Errorf("ActionTaken = %q, want to contain 'auto-skipped watcher'", payload.ActionTaken)
			}
			break
		}
	}
	if !found {
		t.Fatal("expected ceremony.loop.break event for watcher auto-skip")
	}
}

func TestCircuitBreakerTripEmitsLoopBreak(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	// Simulate the circuit breaker trip path
	emitLoopBreakEvent("circuit_break",
		"3 consecutive worker failures (threshold: 3)",
		"circuit breaker tripped for Builder-42",
		"aether-build")

	persisted := readPersistedCeremonyEvents(t)
	found := false
	for _, evt := range persisted {
		if evt.Topic == events.CeremonyTopicLoopBreak {
			found = true
			var payload events.CeremonyPayload
			if err := json.Unmarshal(evt.Payload, &payload); err != nil {
				t.Fatalf("unmarshal payload: %v", err)
			}
			if payload.LoopType != "circuit_break" {
				t.Errorf("LoopType = %q, want %q", payload.LoopType, "circuit_break")
			}
			if !strings.Contains(payload.DetectionSignal, "consecutive worker failures") {
				t.Errorf("DetectionSignal = %q, want to contain 'consecutive worker failures'", payload.DetectionSignal)
			}
			if !strings.Contains(payload.ActionTaken, "circuit breaker tripped") {
				t.Errorf("ActionTaken = %q, want to contain 'circuit breaker tripped'", payload.ActionTaken)
			}
			break
		}
	}
	if !found {
		t.Fatal("expected ceremony.loop.break event for circuit breaker trip")
	}
}

func TestPlanCycleDetectionEmitsLoopBreak(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	// Simulate the cycle detection rejection path
	emitLoopBreakEvent("cycle_detected",
		"circular dependency detected: phase 1 -> phase 2 -> phase 1",
		"plan rejected, cycle must be removed before regeneration",
		"aether-plan")

	persisted := readPersistedCeremonyEvents(t)
	found := false
	for _, evt := range persisted {
		if evt.Topic == events.CeremonyTopicLoopBreak {
			found = true
			var payload events.CeremonyPayload
			if err := json.Unmarshal(evt.Payload, &payload); err != nil {
				t.Fatalf("unmarshal payload: %v", err)
			}
			if payload.LoopType != "cycle_detected" {
				t.Errorf("LoopType = %q, want %q", payload.LoopType, "cycle_detected")
			}
			if !strings.Contains(payload.DetectionSignal, "circular dependency") {
				t.Errorf("DetectionSignal = %q, want to contain 'circular dependency'", payload.DetectionSignal)
			}
			if !strings.Contains(payload.ActionTaken, "plan rejected") {
				t.Errorf("ActionTaken = %q, want to contain 'plan rejected'", payload.ActionTaken)
			}
			break
		}
	}
	if !found {
		t.Fatal("expected ceremony.loop.break event for cycle detection")
	}
}

func TestRecoveryMenuEmitsLoopBreak(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	// Simulate the lifecycle recovery menu path
	emitLoopBreakEvent("lifecycle_recovery",
		"command build failed: test timeout",
		"recovery menu displayed with 3 option(s)",
		"aether-lifecycle")

	persisted := readPersistedCeremonyEvents(t)
	found := false
	for _, evt := range persisted {
		if evt.Topic == events.CeremonyTopicLoopBreak {
			found = true
			var payload events.CeremonyPayload
			if err := json.Unmarshal(evt.Payload, &payload); err != nil {
				t.Fatalf("unmarshal payload: %v", err)
			}
			if payload.LoopType != "lifecycle_recovery" {
				t.Errorf("LoopType = %q, want %q", payload.LoopType, "lifecycle_recovery")
			}
			if !strings.Contains(payload.DetectionSignal, "command build failed") {
				t.Errorf("DetectionSignal = %q, want to contain 'command build failed'", payload.DetectionSignal)
			}
			if !strings.Contains(payload.ActionTaken, "recovery menu displayed") {
				t.Errorf("ActionTaken = %q, want to contain 'recovery menu displayed'", payload.ActionTaken)
			}
			break
		}
	}
	if !found {
		t.Fatal("expected ceremony.loop.break event for lifecycle recovery")
	}
}
