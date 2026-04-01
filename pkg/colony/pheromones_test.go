package colony

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestPheromoneSignalRoundTrip_Full(t *testing.T) {
	raw := `{
		"id": "sig_focus_001",
		"type": "FOCUS",
		"priority": "normal",
		"source": "user",
		"created_at": "2026-03-27T19:02:18Z",
		"expires_at": "2026-03-28T19:02:18Z",
		"active": true,
		"strength": 0.8,
		"reason": "User directed colony attention",
		"content": {"text": "Focus on error handling"},
		"content_hash": "abc123",
		"reinforcement_count": 2,
		"tags": [{"value": "safety", "weight": 1.0, "category": "constraint"}],
		"scope": {"global": true}
	}`
	var sig PheromoneSignal
	if err := json.Unmarshal([]byte(raw), &sig); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	data, err := json.Marshal(sig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded PheromoneSignal
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	if decoded.Strength == nil || *decoded.Strength != 0.8 {
		t.Errorf("strength mismatch")
	}
	if decoded.ReinforcementCount == nil || *decoded.ReinforcementCount != 2 {
		t.Errorf("reinforcement_count mismatch")
	}
	if len(decoded.Tags) != 1 || decoded.Tags[0].Value != "safety" {
		t.Errorf("tags mismatch")
	}
	if decoded.Scope == nil || !decoded.Scope.Global {
		t.Errorf("scope mismatch")
	}
}

func TestPheromoneSignalRoundTrip_Minimal(t *testing.T) {
	raw := `{"id": "imported:", "content": {"text": ""}}`
	var sig PheromoneSignal
	if err := json.Unmarshal([]byte(raw), &sig); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if sig.Type != "" {
		t.Errorf("expected empty type for minimal signal, got %q", sig.Type)
	}
}

func TestPheromoneContentRawMessage(t *testing.T) {
	raw := `{"id": "x", "content": {"text": "hello \"world\""}}`
	var sig PheromoneSignal
	if err := json.Unmarshal([]byte(raw), &sig); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	data, err := json.Marshal(sig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded PheromoneSignal
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	// Content should preserve nested JSON without double-escaping
	var content map[string]string
	if err := json.Unmarshal(decoded.Content, &content); err != nil {
		t.Fatalf("unmarshal content: %v", err)
	}
	if content["text"] != `hello "world"` {
		t.Errorf("content text mismatch: %q", content["text"])
	}
}

func TestPheromoneFileRoundTrip(t *testing.T) {
	file := PheromoneFile{
		Signals: []PheromoneSignal{
			{ID: "sig_1", Type: "FOCUS", Source: "user", CreatedAt: "2026-01-01T00:00:00Z", Active: true, Content: json.RawMessage(`{"text":"test"}`)},
			{ID: "sig_2", Type: "REDIRECT", Source: "system", CreatedAt: "2026-01-02T00:00:00Z", Active: false, Content: json.RawMessage(`{"text":"avoid"}`)},
		},
		Version: strPtr("1.0"),
	}
	data, err := json.Marshal(file)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded PheromoneFile
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(decoded.Signals) != 2 {
		t.Fatalf("expected 2 signals, got %d", len(decoded.Signals))
	}
	if decoded.Version == nil || *decoded.Version != "1.0" {
		t.Errorf("version mismatch")
	}
}

func TestGoldenPheromones(t *testing.T) {
	golden, err := os.ReadFile("testdata/pheromones.golden.json")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	var file PheromoneFile
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
