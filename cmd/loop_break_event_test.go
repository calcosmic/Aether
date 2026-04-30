package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/events"
)

// --- Task 1: RED tests for CeremonyPayload fields and topic constant ---

func TestCeremonyTopicLoopBreakConstant(t *testing.T) {
	if events.CeremonyTopicLoopBreak != "ceremony.loop.break" {
		t.Errorf("CeremonyTopicLoopBreak = %q, want %q", events.CeremonyTopicLoopBreak, "ceremony.loop.break")
	}
}

func TestCeremonyTopicsIncludesLoopBreak(t *testing.T) {
	topics := events.CeremonyTopics()
	found := false
	for _, topic := range topics {
		if topic == events.CeremonyTopicLoopBreak {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("CeremonyTopics() does not include %q; got %v", events.CeremonyTopicLoopBreak, topics)
	}
}

func TestCeremonyPayloadLoopFields(t *testing.T) {
	payload := events.CeremonyPayload{
		LoopType:        "watcher_skip",
		DetectionSignal: "3 failures",
		ActionTaken:     "auto-skipped",
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	s := string(data)
	if !strings.Contains(s, `"loop_type":"watcher_skip"`) {
		t.Errorf("JSON missing loop_type field: %s", s)
	}
	if !strings.Contains(s, `"detection_signal":"3 failures"`) {
		t.Errorf("JSON missing detection_signal field: %s", s)
	}
	if !strings.Contains(s, `"action_taken":"auto-skipped"`) {
		t.Errorf("JSON missing action_taken field: %s", s)
	}
}

// --- Task 2: RED tests for emitLoopBreakEvent and trim behavior ---

func TestEmitLoopBreakEvent(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	emitLoopBreakEvent("watcher_skip", "3 failures", "auto-skipped watcher", "test-source")

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
			if payload.DetectionSignal != "3 failures" {
				t.Errorf("DetectionSignal = %q, want %q", payload.DetectionSignal, "3 failures")
			}
			if payload.ActionTaken != "auto-skipped watcher" {
				t.Errorf("ActionTaken = %q, want %q", payload.ActionTaken, "auto-skipped watcher")
			}
			break
		}
	}
	if !found {
		t.Fatal("expected ceremony.loop.break event in persisted events")
	}
}

func TestEmitLoopBreakEventTrimsPayload(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	longSignal := strings.Repeat("x", ceremonyTextLimit+50)
	emitLoopBreakEvent("infinite_loop", longSignal, "terminated", "test-source")

	persisted := readPersistedCeremonyEvents(t)
	found := false
	for _, evt := range persisted {
		if evt.Topic == events.CeremonyTopicLoopBreak {
			found = true
			var payload events.CeremonyPayload
			if err := json.Unmarshal(evt.Payload, &payload); err != nil {
				t.Fatalf("unmarshal payload: %v", err)
			}
			if len(payload.DetectionSignal) > ceremonyTextLimit {
				t.Errorf("DetectionSignal not trimmed: len=%d, limit=%d", len(payload.DetectionSignal), ceremonyTextLimit)
			}
			if !strings.HasSuffix(payload.DetectionSignal, "...") {
				t.Errorf("DetectionSignal missing trim suffix: %q", payload.DetectionSignal)
			}
			break
		}
	}
	if !found {
		t.Fatal("expected ceremony.loop.break event in persisted events")
	}
}

func TestEmitLoopBreakEventNilStore(t *testing.T) {
	saveGlobals(t)
	store = nil

	// Must not panic
	emitLoopBreakEvent("timeout", "exceeded", "cancelled", "test-source")
}
