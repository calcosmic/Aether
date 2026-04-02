package colony

import (
	"bytes"
	"os"
	"encoding/json"
	"testing"
)

func TestMiddenFileRoundTrip(t *testing.T) {
	archivedAtCount := 5
	entryCount := 2
	file := MiddenFile{
		Version:         "1.0.0",
		ArchivedAtCount: &archivedAtCount,
		Signals: []MiddenArchivedSignal{
			{
				ID: "sig_redirect_001", Type: "REDIRECT", Priority: "high", Source: "system",
				CreatedAt: "2026-02-16T08:00:00Z", Active: false,
				Content: json.RawMessage(`{"text":"Avoid editing runtime/"}`),
				Tags:      []PheromoneTag{{Value: "safety", Weight: 1.0, Category: "constraint"}},
				Scope:     &PheromoneScope{Global: true},
				ArchivedAt: strPtr("2026-03-20T19:50:49Z"),
			},
		},
		Entries: []MiddenEntry{
			{ID: "midden_001", Timestamp: "2026-02-21T23:50:33Z", Category: "security", Source: "gatekeeper", Message: "CVEs found", Reviewed: true},
			{ID: "midden_002", Timestamp: "2026-02-22T10:00:00Z", Category: "build", Source: "builder", Message: "Compile error"},
		},
		EntryCount: &entryCount,
	}
	data, err := json.Marshal(file)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded MiddenFile
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(decoded.Signals) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(decoded.Signals))
	}
	if len(decoded.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(decoded.Entries))
	}
	if decoded.EntryCount == nil || *decoded.EntryCount != 2 {
		t.Errorf("entry_count mismatch")
	}
	if decoded.Signals[0].Tags[0].Value != "safety" {
		t.Errorf("tag value mismatch")
	}
	if decoded.Entries[1].Reviewed != false {
		t.Errorf("midden_002 should not be reviewed")
	}
}

func TestMiddenEntryNullableFields(t *testing.T) {
	ack := true
	entry := MiddenEntry{
		ID: "midden_003", Timestamp: "2026-03-01T00:00:00Z",
		Category: "build", Source: "builder", Message: "Test failure",
		Acknowledged: &ack, AcknowledgedAt: strPtr("2026-03-02T00:00:00Z"),
		Tags: []string{"test"},
	}
	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded MiddenEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Acknowledged == nil || !*decoded.Acknowledged {
		t.Error("acknowledged should be true")
	}
	if len(decoded.Tags) != 1 || decoded.Tags[0] != "test" {
		t.Errorf("tags mismatch")
	}
}

func TestGoldenMidden(t *testing.T) {
	golden, err := os.ReadFile("testdata/midden.golden.json")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	var file MiddenFile
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
