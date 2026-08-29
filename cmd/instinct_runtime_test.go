package cmd

import (
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

func TestLoadRuntimeInstincts_EmptyStandaloneFileDoesNotFallback(t *testing.T) {
	saveGlobals(t)

	dataDir := t.TempDir()
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	state := colony.ColonyState{
		Memory: colony.Memory{
			Instincts: []colony.Instinct{
				{ID: "legacy-1", Trigger: "legacy trigger", Action: "legacy action", Confidence: 0.8},
			},
		},
	}

	if err := s.SaveJSON("instincts.json", colony.InstinctsFile{
		Version:   "1.0",
		Instincts: []colony.InstinctEntry{},
	}); err != nil {
		t.Fatalf("failed to save instincts.json: %v", err)
	}

	got := loadRuntimeInstincts(s, &state)
	if len(got) != 0 {
		t.Fatalf("expected no runtime instincts when standalone file is empty, got %+v", got)
	}
}

func TestLoadRecentRuntimeInstincts_PrefersUsefulAppliedInstincts(t *testing.T) {
	saveGlobals(t)

	dataDir := t.TempDir()
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	if err := s.SaveJSON("instincts.json", colony.InstinctsFile{
		Version: "1.0",
		Instincts: []colony.InstinctEntry{
			{
				ID:         "new_unused",
				Trigger:    "new trigger",
				Action:     "new action",
				Confidence: 0.92,
				TrustScore: 0.92,
				TrustTier:  "canonical",
				Provenance: colony.InstinctProvenance{
					CreatedAt: "2026-04-21T10:00:00Z",
				},
			},
			{
				ID:         "applied_reliable",
				Trigger:    "applied trigger",
				Action:     "applied action",
				Confidence: 0.72,
				TrustScore: 0.78,
				TrustTier:  "trusted",
				Provenance: colony.InstinctProvenance{
					CreatedAt:        "2026-04-10T10:00:00Z",
					LastApplied:      runtimeStrPtr("2026-04-21T11:00:00Z"),
					ApplicationCount: 4,
				},
				ApplicationHistory: []interface{}{
					map[string]interface{}{"timestamp": "2026-04-20T10:00:00Z", "success": true},
					map[string]interface{}{"timestamp": "2026-04-21T11:00:00Z", "success": true},
				},
			},
		},
	}); err != nil {
		t.Fatalf("failed to save instincts.json: %v", err)
	}

	got := loadStrongestRuntimeInstincts(s, &colony.ColonyState{}, 2)
	if len(got) != 2 {
		t.Fatalf("expected 2 instincts, got %d", len(got))
	}
	if got[0].ID != "applied_reliable" {
		t.Fatalf("expected applied instinct first, got %+v", got)
	}
}

func runtimeStrPtr(v string) *string {
	return &v
}
