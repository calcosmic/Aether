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
