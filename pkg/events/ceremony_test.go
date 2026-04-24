package events

import (
	"encoding/json"
	"testing"
)

func TestCeremonyTopicsUseCeremonyNamespace(t *testing.T) {
	topics := CeremonyTopics()
	if len(topics) == 0 {
		t.Fatal("expected ceremony topics")
	}
	for _, topic := range topics {
		if !TopicMatch("ceremony.*", topic) {
			t.Fatalf("topic %q does not match ceremony wildcard", topic)
		}
	}
}

func TestCeremonyPayloadRawMessage(t *testing.T) {
	raw, err := (CeremonyPayload{
		Phase:     2,
		Wave:      1,
		SpawnID:   "spawn-1",
		Caste:     "builder",
		Name:      "Mason-67",
		TaskID:    "2.1",
		Task:      "Restore builder spawning",
		Status:    "starting",
		ToolCount: 3,
	}).RawMessage()
	if err != nil {
		t.Fatalf("RawMessage returned error: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("payload should be valid JSON: %v", err)
	}
	if decoded["caste"] != "builder" {
		t.Fatalf("caste = %v, want builder", decoded["caste"])
	}
	if _, ok := decoded["blockers"]; ok {
		t.Fatal("empty optional fields should be omitted")
	}
}
