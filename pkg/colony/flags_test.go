package colony

import (
	"encoding/json"
	"testing"
)

func TestFlagsFileRoundTrip(t *testing.T) {
	phase := 3
	file := FlagsFile{
		Version: "1.0",
		Decisions: []FlagEntry{
			{ID: "flag_001", Type: "replan", Description: "Replan phase 3", Phase: &phase, Source: "watcher", CreatedAt: "2026-03-29T00:00:00Z", Resolved: false},
			{ID: "flag_002", Type: "bug", Description: "Fix crash in parser", Phase: nil, Source: "user", CreatedAt: "2026-03-30T00:00:00Z", Resolved: true},
		},
	}
	data, err := json.Marshal(file)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded FlagsFile
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(decoded.Decisions) != 2 {
		t.Fatalf("expected 2 decisions, got %d", len(decoded.Decisions))
	}
	if decoded.Decisions[0].Phase == nil || *decoded.Decisions[0].Phase != 3 {
		t.Error("phase mismatch for flag_001")
	}
	if decoded.Decisions[1].Phase != nil {
		t.Error("expected nil phase for flag_002")
	}
	if decoded.Decisions[1].Resolved != true {
		t.Error("flag_002 should be resolved")
	}
}
