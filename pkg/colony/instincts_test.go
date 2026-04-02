package colony

import (
	"encoding/json"
	"testing"
)

func TestInstinctsFileRoundTrip(t *testing.T) {
	file := InstinctsFile{
		Version: "1.0",
		Instincts: []InstinctEntry{
			{
				ID: "inst_001", Trigger: "test trigger", Action: "test action", Domain: "pattern",
				TrustScore: 0.75, TrustTier: "medium", Confidence: 0.7,
				Provenance: InstinctProvenance{
					Source: "promoted", SourceType: "learning", Evidence: "observed twice",
					CreatedAt: "2026-03-31T00:00:00Z", LastApplied: nil, ApplicationCount: 0,
				},
				ApplicationHistory: []interface{}{},
				RelatedInstincts:   []interface{}{},
				Archived:           false,
			},
		},
	}
	data, err := json.Marshal(file)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded InstinctsFile
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Version != "1.0" {
		t.Fatalf("version mismatch: got %q", decoded.Version)
	}
	if len(decoded.Instincts) != 1 {
		t.Fatalf("expected 1 instinct, got %d", len(decoded.Instincts))
	}
	inst := decoded.Instincts[0]
	if inst.TrustScore != 0.75 {
		t.Errorf("trust_score mismatch: got %f", inst.TrustScore)
	}
	if inst.Provenance.LastApplied != nil {
		t.Error("expected nil LastApplied")
	}
	if inst.Provenance.ApplicationCount != 0 {
		t.Errorf("application_count mismatch: got %d", inst.Provenance.ApplicationCount)
	}
	if inst.Archived != false {
		t.Error("expected archived=false")
	}
}

func TestInstinctEntryArchived(t *testing.T) {
	file := InstinctsFile{
		Version: "1.0",
		Instincts: []InstinctEntry{
			{
				ID: "inst_002", Trigger: "old pattern", Action: "old action", Domain: "legacy",
				TrustScore: 0.2, TrustTier: "low", Confidence: 0.1,
				Provenance: InstinctProvenance{
					Source: "promoted", SourceType: "learning", Evidence: "disproven",
					CreatedAt: "2026-01-01T00:00:00Z", LastApplied: strPtr("2026-02-01T00:00:00Z"), ApplicationCount: 5,
				},
				ApplicationHistory: []interface{}{},
				RelatedInstincts:   []interface{}{},
				Archived:           true,
			},
		},
	}
	data, err := json.Marshal(file)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded InstinctsFile
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !decoded.Instincts[0].Archived {
		t.Error("expected archived=true")
	}
	if decoded.Instincts[0].Provenance.LastApplied == nil || *decoded.Instincts[0].Provenance.LastApplied != "2026-02-01T00:00:00Z" {
		t.Error("last_applied mismatch")
	}
}
