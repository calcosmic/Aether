package colony

import (
	"bytes"
	"os"
	"encoding/json"
	"testing"
)

func TestSessionFileRoundTrip(t *testing.T) {
	file := SessionFile{
		SessionID: "session_123_abc", StartedAt: "2026-03-31T19:30:07Z",
		LastCommand: "/ant:continue", LastCommandAt: "2026-04-01T17:32:39Z",
		ColonyGoal: "Build something great", CurrentPhase: 4,
		CurrentMilestone: "First Mound", SuggestedNext: "/ant:build 5",
		ContextCleared: false, BaselineCommit: "abc123",
		ResumedAt: nil,
		ActiveTodos:  []string{"Task A", "Task B"},
		Summary:      "Phase 4 complete",
	}
	data, err := json.Marshal(file)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded SessionFile
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.ResumedAt != nil {
		t.Error("expected nil ResumedAt")
	}
	if decoded.CurrentPhase != 4 {
		t.Errorf("current_phase mismatch: got %d", decoded.CurrentPhase)
	}
	if len(decoded.ActiveTodos) != 2 {
		t.Errorf("active_todos mismatch: got %d", len(decoded.ActiveTodos))
	}
}

func TestSessionFileNullResumedAt(t *testing.T) {
	raw := `{"session_id":"s1","started_at":"2026-01-01T00:00:00Z","last_command":"","last_command_at":"","colony_goal":"","current_phase":0,"current_milestone":"","suggested_next":"","context_cleared":false,"baseline_commit":"","resumed_at":null,"active_todos":[],"summary":""}`
	var sf SessionFile
	if err := json.Unmarshal([]byte(raw), &sf); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if sf.ResumedAt != nil {
		t.Error("expected nil ResumedAt from JSON null")
	}
}

func TestGoldenSession(t *testing.T) {
	golden, err := os.ReadFile("testdata/session.golden.json")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	var file SessionFile
	if err := json.Unmarshal(golden, &file); err != nil {
		t.Fatalf("unmarshal golden: %v", err)
	}
	produced, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	produced = append(produced, '\n')
	if !bytes.Equal(golden, produced) {
		t.Errorf("golden mismatch:\nexpected:\n%s\n\ngot:\n%s", golden, produced)
	}
}
