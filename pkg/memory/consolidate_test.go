package memory

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aether-colony/aether/pkg/colony"
	"github.com/aether-colony/aether/pkg/events"
	"github.com/aether-colony/aether/pkg/storage"
)

func setupConsolidationTest(t *testing.T) (*storage.Store, *events.Bus, string) {
	t.Helper()
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	bus := events.NewBus(store, events.Config{JSONLFile: "events.jsonl"})
	return store, bus, dir
}

func TestConsolidate_TrustDecay(t *testing.T) {
	store, bus, _ := setupConsolidationTest(t)

	// Pre-populate instincts.json with an old instinct
	past := time.Now().UTC().Add(-120 * 24 * time.Hour) // 120 days ago
	pastStr := past.Format("2006-01-02T15:04:05Z")
	instincts := colony.InstinctsFile{
		Version: "1.0",
		Instincts: []colony.InstinctEntry{
			{
				ID:         "inst_old_001",
				Trigger:    "test trigger for decay",
				Action:     "test action",
				Domain:     "testing",
				TrustScore: 0.85,
				TrustTier:  "trusted",
				Confidence: 0.8,
				Provenance: colony.InstinctProvenance{
					CreatedAt:        pastStr,
					ApplicationCount: 5,
				},
				Archived: false,
			},
		},
	}
	if err := store.SaveJSON("instincts.json", instincts); err != nil {
		t.Fatalf("save instincts: %v", err)
	}

	svc := NewConsolidationService(store, bus, "", "test-colony")
	result, err := svc.Run(context.Background())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.InstinctsDecayed != 1 {
		t.Errorf("InstinctsDecayed = %d, want 1", result.InstinctsDecayed)
	}

	// Verify the instinct's trust score was decayed
	var updated colony.InstinctsFile
	if err := store.LoadJSON("instincts.json", &updated); err != nil {
		t.Fatalf("load updated instincts: %v", err)
	}
	decayedScore := Decay(0.85, 120)
	if math.Abs(updated.Instincts[0].TrustScore-decayedScore) > 1e-6 {
		t.Errorf("TrustScore = %f, want %f (decayed)", updated.Instincts[0].TrustScore, decayedScore)
	}
	// Tier should be updated too
	if updated.Instincts[0].TrustTier == "" {
		t.Error("TrustTier is empty, should be updated after decay")
	}
}

func TestConsolidate_ArchiveBelowFloor(t *testing.T) {
	store, bus, _ := setupConsolidationTest(t)

	// Create a very old instinct that will decay below 0.2
	past := time.Now().UTC().Add(-365 * 24 * time.Hour) // 365 days ago
	pastStr := past.Format("2006-01-02T15:04:05Z")
	recentStr := time.Now().UTC().Format("2006-01-02T15:04:05Z") // recent for the OK one
	instincts := colony.InstinctsFile{
		Version: "1.0",
		Instincts: []colony.InstinctEntry{
			{
				ID:         "inst_old_low",
				Trigger:    "low trust trigger",
				Action:     "test action",
				Domain:     "testing",
				TrustScore: 0.25, // Low initial score, will decay further
				TrustTier:  "suspect",
				Confidence: 0.3,
				Provenance: colony.InstinctProvenance{
					CreatedAt:        pastStr,
					ApplicationCount: 1,
				},
				Archived: false,
			},
			{
				ID:         "inst_old_ok",
				Trigger:    "ok trust trigger",
				Action:     "test action",
				Domain:     "testing",
				TrustScore: 0.90,
				TrustTier:  "canonical",
				Confidence: 0.85,
				Provenance: colony.InstinctProvenance{
					CreatedAt:        recentStr, // recent so it doesn't decay much
					ApplicationCount: 10,
				},
				Archived: false,
			},
		},
	}
	if err := store.SaveJSON("instincts.json", instincts); err != nil {
		t.Fatalf("save instincts: %v", err)
	}

	svc := NewConsolidationService(store, bus, "", "test-colony")
	result, err := svc.Run(context.Background())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.InstinctsArchived != 1 {
		t.Errorf("InstinctsArchived = %d, want 1", result.InstinctsArchived)
	}

	var updated colony.InstinctsFile
	if err := store.LoadJSON("instincts.json", &updated); err != nil {
		t.Fatalf("load updated instincts: %v", err)
	}

	// First should be archived
	if !updated.Instincts[0].Archived {
		t.Error("first instinct should be archived after decay")
	}
	// Second should still be active
	if updated.Instincts[1].Archived {
		t.Error("second instinct should not be archived")
	}
}

func TestConsolidate_ObservationDecay(t *testing.T) {
	store, bus, _ := setupConsolidationTest(t)

	past := time.Now().UTC().Add(-90 * 24 * time.Hour)
	pastStr := past.Format("2006-01-02T15:04:05Z")
	trustScore := 0.75

	obs := colony.LearningFile{
		Observations: []colony.Observation{
			{
				ContentHash:      "sha256:abc123",
				Content:          "test observation content",
				WisdomType:       "pattern",
				ObservationCount: 2,
				FirstSeen:        pastStr,
				LastSeen:         pastStr,
				Colonies:         []string{"test-colony"},
				TrustScore:       &trustScore,
				SourceType:       "observation",
				EvidenceType:     "anecdotal",
			},
		},
	}
	if err := store.SaveJSON("learning-observations.json", obs); err != nil {
		t.Fatalf("save observations: %v", err)
	}

	svc := NewConsolidationService(store, bus, "", "test-colony")
	result, err := svc.Run(context.Background())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.ObservationsDecayed != 1 {
		t.Errorf("ObservationsDecayed = %d, want 1", result.ObservationsDecayed)
	}

	var updated colony.LearningFile
	if err := store.LoadJSON("learning-observations.json", &updated); err != nil {
		t.Fatalf("load updated observations: %v", err)
	}
	decayed := Decay(0.75, 90)
	if updated.Observations[0].TrustScore == nil {
		t.Fatal("TrustScore is nil after decay")
	}
	if math.Abs(*updated.Observations[0].TrustScore-decayed) > 1e-6 {
		t.Errorf("TrustScore = %f, want %f", *updated.Observations[0].TrustScore, decayed)
	}
}

func TestConsolidate_CheckPromotions(t *testing.T) {
	store, bus, _ := setupConsolidationTest(t)

	nowStr := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	trustScore := 0.65
	lowTrust := 0.30

	obs := colony.LearningFile{
		Observations: []colony.Observation{
			{
				ContentHash:      "sha256:promo1",
				Content:          "eligible observation",
				WisdomType:       "pattern",
				ObservationCount: 3, // count >= 3 -> promotion eligible
				FirstSeen:        nowStr,
				LastSeen:         nowStr,
				Colonies:         []string{"test-colony"},
				TrustScore:       &trustScore,
				SourceType:       "success_pattern",
				EvidenceType:     "single_phase",
			},
			{
				ContentHash:      "sha256:nopromo",
				Content:          "not eligible observation",
				WisdomType:       "stack",
				ObservationCount: 1,
				FirstSeen:        nowStr,
				LastSeen:         nowStr,
				Colonies:         []string{"test-colony"},
				TrustScore:       &lowTrust,
				SourceType:       "observation",
				EvidenceType:     "anecdotal",
			},
		},
	}
	if err := store.SaveJSON("learning-observations.json", obs); err != nil {
		t.Fatalf("save observations: %v", err)
	}

	svc := NewConsolidationService(store, bus, "", "test-colony")
	result, err := svc.Run(context.Background())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(result.PromotionCandidates) != 1 {
		t.Errorf("PromotionCandidates = %v, want 1 entry", result.PromotionCandidates)
	}
	if len(result.PromotionCandidates) > 0 && result.PromotionCandidates[0] != "sha256:promo1" {
		t.Errorf("PromotionCandidates[0] = %q, want %q", result.PromotionCandidates[0], "sha256:promo1")
	}
}

func TestConsolidate_QueenEligible(t *testing.T) {
	store, bus, _ := setupConsolidationTest(t)

	nowStr := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	instincts := colony.InstinctsFile{
		Version: "1.0",
		Instincts: []colony.InstinctEntry{
			{
				ID:         "inst_queen_001",
				Trigger:    "queen eligible trigger",
				Action:     "test action",
				Domain:     "testing",
				TrustScore: 0.85,
				TrustTier:  "trusted",
				Confidence: 0.80,
				Provenance: colony.InstinctProvenance{
					CreatedAt:        nowStr,
					ApplicationCount: 5, // >= 3 and confidence >= 0.75
				},
				Archived: false,
			},
			{
				ID:         "inst_not_queen",
				Trigger:    "not queen eligible",
				Action:     "test action",
				Domain:     "testing",
				TrustScore: 0.50,
				TrustTier:  "emerging",
				Confidence: 0.60,
				Provenance: colony.InstinctProvenance{
					CreatedAt:        nowStr,
					ApplicationCount: 2, // too few
				},
				Archived: false,
			},
		},
	}
	if err := store.SaveJSON("instincts.json", instincts); err != nil {
		t.Fatalf("save instincts: %v", err)
	}

	svc := NewConsolidationService(store, bus, "", "test-colony")
	result, err := svc.Run(context.Background())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(result.QueenEligible) != 1 {
		t.Fatalf("QueenEligible = %v, want 1 entry", result.QueenEligible)
	}
	if result.QueenEligible[0] != "inst_queen_001" {
		t.Errorf("QueenEligible[0] = %q, want %q", result.QueenEligible[0], "inst_queen_001")
	}
}

func TestConsolidate_Event(t *testing.T) {
	store, bus, _ := setupConsolidationTest(t)

	svc := NewConsolidationService(store, bus, "", "test-colony")
	result, err := svc.Run(context.Background())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	_ = result

	// Query for consolidation.phase_end event
	evts, err := bus.Query(context.Background(), "consolidation.phase_end", time.Time{}, 10)
	if err != nil {
		t.Fatalf("Query events: %v", err)
	}
	if len(evts) != 1 {
		t.Fatalf("expected 1 consolidation.phase_end event, got %d", len(evts))
	}

	// Verify payload has summary fields
	var payload map[string]int
	if err := json.Unmarshal(evts[0].Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if _, ok := payload["instincts_decayed"]; !ok {
		t.Error("payload missing instincts_decayed")
	}
	if _, ok := payload["instincts_archived"]; !ok {
		t.Error("payload missing instincts_archived")
	}
	if _, ok := payload["observations_decayed"]; !ok {
		t.Error("payload missing observations_decayed")
	}
}

func TestConsolidate_NonBlocking(t *testing.T) {
	store, bus, _ := setupConsolidationTest(t)

	// No files exist at all - consolidation should not fail
	svc := NewConsolidationService(store, bus, "", "test-colony")
	result, err := svc.Run(context.Background())
	if err != nil {
		t.Fatalf("Run returned error with missing files: %v", err)
	}
	// Should have errors recorded but not fail
	if len(result.Errors) == 0 {
		t.Error("expected errors to be recorded for missing files, got none")
	}
	// Event should still be published
	evts, _ := bus.Query(context.Background(), "consolidation.phase_end", time.Time{}, 10)
	if len(evts) != 1 {
		t.Errorf("expected consolidation.phase_end event even on errors, got %d", len(evts))
	}
}

func TestConsolidationResult_Fields(t *testing.T) {
	store, bus, _ := setupConsolidationTest(t)

	nowStr := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	trustScore := 0.80

	// Setup instincts and observations for a full result
	instincts := colony.InstinctsFile{
		Version: "1.0",
		Instincts: []colony.InstinctEntry{
			{
				ID:         "inst_001",
				Trigger:    "test trigger",
				Action:     "test action",
				Domain:     "testing",
				TrustScore: 0.80,
				TrustTier:  "trusted",
				Confidence: 0.80,
				Provenance: colony.InstinctProvenance{
					CreatedAt:        nowStr,
					ApplicationCount: 4,
				},
				Archived: false,
			},
		},
	}
	if err := store.SaveJSON("instincts.json", instincts); err != nil {
		t.Fatalf("save instincts: %v", err)
	}

	obs := colony.LearningFile{
		Observations: []colony.Observation{
			{
				ContentHash:      "sha256:cand1",
				Content:          "candidate observation",
				WisdomType:       "pattern",
				ObservationCount: 3,
				FirstSeen:        nowStr,
				LastSeen:         nowStr,
				Colonies:         []string{"test-colony"},
				TrustScore:       &trustScore,
				SourceType:       "success_pattern",
				EvidenceType:     "single_phase",
			},
		},
	}
	if err := store.SaveJSON("learning-observations.json", obs); err != nil {
		t.Fatalf("save observations: %v", err)
	}

	svc := NewConsolidationService(store, bus, "", "test-colony")
	result, err := svc.Run(context.Background())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if result.InstinctsDecayed != 1 {
		t.Errorf("InstinctsDecayed = %d, want 1", result.InstinctsDecayed)
	}
	if result.ObservationsDecayed != 1 {
		t.Errorf("ObservationsDecayed = %d, want 1", result.ObservationsDecayed)
	}
	if len(result.PromotionCandidates) != 1 {
		t.Errorf("PromotionCandidates = %v, want 1", result.PromotionCandidates)
	}
	if len(result.QueenEligible) != 1 {
		t.Errorf("QueenEligible = %v, want 1", result.QueenEligible)
	}
	if result.Errors != nil {
		t.Errorf("Errors = %v, want nil", result.Errors)
	}
}

func TestConsolidate_QueenPathPassed(t *testing.T) {
	store, bus, dir := setupConsolidationTest(t)
	queenPath := filepath.Join(dir, "QUEEN.md")

	svc := NewConsolidationService(store, bus, queenPath, "test-colony")
	if svc.queenPath != queenPath {
		t.Errorf("queenPath = %q, want %q", svc.queenPath, queenPath)
	}
	// Verify QUEEN.md dir doesn't need to exist at construction time
	if _, err := os.Stat(queenPath); !os.IsNotExist(err) {
		t.Error("QUEEN.md should not exist yet")
	}
}
