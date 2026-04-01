package colony

import (
	"bytes"
	"os"
	"encoding/json"
	"testing"
)

func TestLearningFileRoundTrip(t *testing.T) {
	file := LearningFile{
		Observations: []Observation{
			{ContentHash: "sha256:abc", Content: "Use explicit jq chains", WisdomType: "pattern", ObservationCount: 2, FirstSeen: "2026-03-19T19:10:44Z", LastSeen: "2026-03-19T19:10:45Z", Colonies: []string{"123"}},
			{ContentHash: "sha256:def", Content: "Regex needs multi-word support", WisdomType: "pattern", ObservationCount: 1, FirstSeen: "2026-03-20T07:20:42Z", LastSeen: "2026-03-20T07:20:42Z", Colonies: []string{"456"}},
		},
	}
	data, err := json.Marshal(file)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded LearningFile
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(decoded.Observations) != 2 {
		t.Fatalf("expected 2 observations, got %d", len(decoded.Observations))
	}
	if decoded.Observations[0].WisdomType != "pattern" {
		t.Errorf("wisdom_type mismatch")
	}
	if decoded.Observations[0].ObservationCount != 2 {
		t.Errorf("observation_count mismatch")
	}
	if decoded.Observations[1].Colonies[0] != "456" {
		t.Errorf("colonies mismatch")
	}
}

func TestGoldenLearning(t *testing.T) {
	golden, err := os.ReadFile("testdata/learning-observations.golden.json")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	var file LearningFile
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
