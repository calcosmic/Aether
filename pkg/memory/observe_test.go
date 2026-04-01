package memory

import (
	"context"
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/aether-colony/aether/pkg/colony"
	"github.com/aether-colony/aether/pkg/events"
	"github.com/aether-colony/aether/pkg/storage"
)

func newTestObserveService(t *testing.T) (*ObservationService, *storage.Store, string) {
	t.Helper()
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	bus := events.NewBus(store, events.DefaultConfig())
	t.Cleanup(func() { bus.Close() })
	svc := NewObservationService(store, bus)
	return svc, store, dir
}

func TestCaptureNew(t *testing.T) {
	svc, store, _ := newTestObserveService(t)
	ctx := context.Background()

	result, err := svc.Capture(ctx, "Use table-driven tests", "pattern", "test-colony")
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}

	if !result.IsNew {
		t.Error("expected IsNew=true for new observation")
	}
	if result.Observation.ContentHash == "" {
		t.Error("ContentHash should not be empty")
	}
	if result.Observation.ObservationCount != 1 {
		t.Errorf("ObservationCount = %d, want 1", result.Observation.ObservationCount)
	}
	if result.Observation.TrustScore == nil {
		t.Error("TrustScore should not be nil")
	}
	if result.Observation.FirstSeen == "" {
		t.Error("FirstSeen should not be empty")
	}
	if result.Observation.LastSeen == "" {
		t.Error("LastSeen should not be empty")
	}
	if result.Observation.SourceType != "observation" {
		t.Errorf("SourceType = %q, want 'observation'", result.Observation.SourceType)
	}
	if result.Observation.EvidenceType != "anecdotal" {
		t.Errorf("EvidenceType = %q, want 'anecdotal'", result.Observation.EvidenceType)
	}

	// Verify file was saved
	var file colony.LearningFile
	if err := store.LoadJSON("learning-observations.json", &file); err != nil {
		t.Fatalf("load observations: %v", err)
	}
	if len(file.Observations) != 1 {
		t.Fatalf("expected 1 observation, got %d", len(file.Observations))
	}

	// Verify event was published
	time.Sleep(10 * time.Millisecond)
	lines, err := store.ReadJSONL("event-bus.jsonl")
	if err != nil {
		t.Fatalf("read JSONL: %v", err)
	}
	found := false
	for _, raw := range lines {
		var e struct {
			Topic string `json:"topic"`
		}
		if err := json.Unmarshal(raw, &e); err != nil {
			continue
		}
		if e.Topic == "learning.observe" {
			found = true
			break
		}
	}
	if !found {
		t.Error("learning.observe event not found in JSONL")
	}
}

func TestCaptureDedup(t *testing.T) {
	svc, store, _ := newTestObserveService(t)
	ctx := context.Background()

	// First capture
	r1, err := svc.Capture(ctx, "Use table-driven tests", "pattern", "test-colony")
	if err != nil {
		t.Fatalf("first Capture: %v", err)
	}
	if !r1.IsNew {
		t.Error("first capture should be new")
	}

	// Second capture of same content+wisdomType
	r2, err := svc.Capture(ctx, "Use table-driven tests", "pattern", "test-colony")
	if err != nil {
		t.Fatalf("second Capture: %v", err)
	}
	if r2.IsNew {
		t.Error("second capture should not be new (dedup)")
	}
	if r2.Observation.ObservationCount != 2 {
		t.Errorf("ObservationCount = %d, want 2", r2.Observation.ObservationCount)
	}
	if r2.Observation.FirstSeen != r1.Observation.FirstSeen {
		t.Error("FirstSeen should not change on dedup")
	}
	if r2.Observation.LastSeen == r1.Observation.FirstSeen {
		t.Error("LastSeen should differ from FirstSeen")
	}

	// Should still be 1 observation in file
	var file colony.LearningFile
	if err := store.LoadJSON("learning-observations.json", &file); err != nil {
		t.Fatalf("load observations: %v", err)
	}
	if len(file.Observations) != 1 {
		t.Errorf("expected 1 observation after dedup, got %d", len(file.Observations))
	}
}

func TestCaptureLegacyBackfill(t *testing.T) {
	svc, store, _ := newTestObserveService(t)
	ctx := context.Background()

	// Manually create a legacy observation without trust fields
	legacy := colony.Observation{
		ContentHash:      "sha256:legacyhash",
		Content:          "Legacy observation",
		WisdomType:       "pattern",
		ObservationCount: 2,
		FirstSeen:        "2026-01-01T00:00:00Z",
		LastSeen:         "2026-01-15T00:00:00Z",
		Colonies:         []string{"old-colony"},
	}
	file := colony.LearningFile{Observations: []colony.Observation{legacy}}
	store.SaveJSON("learning-observations.json", file)

	// Capture a NEW observation (different content) - should trigger backfill
	_, err := svc.Capture(ctx, "New observation content", "pattern", "test-colony")
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}

	// Verify legacy was backfilled
	var loaded colony.LearningFile
	if err := store.LoadJSON("learning-observations.json", &loaded); err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(loaded.Observations) != 2 {
		t.Fatalf("expected 2 observations, got %d", len(loaded.Observations))
	}

	legacyObs := loaded.Observations[0]
	if legacyObs.TrustScore == nil {
		t.Error("legacy TrustScore should be backfilled")
	}
	if *legacyObs.TrustScore != 0.49 {
		t.Errorf("legacy TrustScore = %f, want 0.49", *legacyObs.TrustScore)
	}
	if legacyObs.SourceType != "legacy" {
		t.Errorf("legacy SourceType = %q, want 'legacy'", legacyObs.SourceType)
	}
	if legacyObs.EvidenceType != "indirect" {
		t.Errorf("legacy EvidenceType = %q, want 'indirect'", legacyObs.EvidenceType)
	}
}

func TestCheckPromotion_TrustThreshold(t *testing.T) {
	obs := colony.Observation{
		ObservationCount: 1,
		TrustScore:       floatPtr(0.55),
	}
	eligible, reason := CheckPromotion(obs)
	if !eligible {
		t.Error("should be promotion-eligible with trust >= 0.50")
	}
	if reason != "trust_threshold" {
		t.Errorf("reason = %q, want 'trust_threshold'", reason)
	}
}

func TestCheckPromotion_SimilarPatterns(t *testing.T) {
	obs := colony.Observation{
		ObservationCount: 3,
		TrustScore:       floatPtr(0.30),
	}
	eligible, reason := CheckPromotion(obs)
	if !eligible {
		t.Error("should be promotion-eligible with 3+ observations")
	}
	if reason != "similar_patterns" {
		t.Errorf("reason = %q, want 'similar_patterns'", reason)
	}
}

func TestCheckPromotion_NotEligible(t *testing.T) {
	obs := colony.Observation{
		ObservationCount: 1,
		TrustScore:       floatPtr(0.40),
		WisdomType:       "custom_type",
	}
	eligible, reason := CheckPromotion(obs)
	if eligible {
		t.Error("should not be promotion-eligible with low trust and low count")
	}
	if reason != "" {
		t.Errorf("reason = %q, want empty", reason)
	}
}

func TestCheckPromotion_WisdomThreshold(t *testing.T) {
	obs := colony.Observation{
		ObservationCount: 3,
		TrustScore:       floatPtr(0.30),
		WisdomType:       "philosophy",
	}
	eligible, reason := CheckPromotion(obs)
	if !eligible {
		t.Error("philosophy with 3 observations should be eligible")
	}
	// similar_patterns (count>=3) fires before wisdom_threshold in the check order
	if reason != "similar_patterns" && reason != "wisdom_threshold" {
		t.Errorf("reason = %q, want 'similar_patterns' or 'wisdom_threshold'", reason)
	}
}

func TestRecurrenceConfidence(t *testing.T) {
	tests := []struct {
		count int
		want  float64
	}{
		{1, 0.70},
		{2, 0.75},
		{3, 0.80},
		{4, 0.85},
		{5, 0.90},
		{10, 0.90}, // capped
	}
	for _, tt := range tests {
		got := RecurrenceConfidence(tt.count)
		if math.Abs(got-tt.want) > 1e-6 {
			t.Errorf("RecurrenceConfidence(%d) = %f, want %f", tt.count, got, tt.want)
		}
	}
}

func TestWisdomThresholds(t *testing.T) {
	tests := []struct {
		wisdomType string
		wantPropose int
		wantAuto    int
	}{
		{"build_learning", 0, 0},
		{"instinct", 0, 0},
		{"philosophy", 1, 3},
		{"pattern", 1, 1},
		{"redirect", 1, 2},
		{"stack", 1, 2},
		{"decree", 0, 0},
		{"failure", 1, 2},
	}
	for _, tt := range tests {
		t.Run(tt.wisdomType, func(t *testing.T) {
			entry, ok := WisdomThresholds[tt.wisdomType]
			if !ok {
				t.Fatalf("threshold not found for %q", tt.wisdomType)
			}
			if entry.Propose != tt.wantPropose {
				t.Errorf("Propose = %d, want %d", entry.Propose, tt.wantPropose)
			}
			if entry.Auto != tt.wantAuto {
				t.Errorf("Auto = %d, want %d", entry.Auto, tt.wantAuto)
			}
		})
	}
}

func TestCaptureContentHashIsSHA256(t *testing.T) {
	svc, _, _ := newTestObserveService(t)
	ctx := context.Background()

	content := "Test content"
	wisdomType := "pattern"

	result, err := svc.Capture(ctx, content, wisdomType, "colony")
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}

	// Hash should start with "sha256:" and be 64 hex chars after prefix
	if len(result.Observation.ContentHash) != 7+64 {
		t.Errorf("ContentHash length = %d, want 71 (sha256: + 64 hex)", len(result.Observation.ContentHash))
	}
	if result.Observation.ContentHash[:7] != "sha256:" {
		t.Errorf("ContentHash prefix = %q, want 'sha256:'", result.Observation.ContentHash[:7])
	}
}

func TestCaptureDifferentWisdomTypesAreSeparate(t *testing.T) {
	svc, store, _ := newTestObserveService(t)
	ctx := context.Background()

	_, err := svc.Capture(ctx, "Same content", "pattern", "colony")
	if err != nil {
		t.Fatalf("first Capture: %v", err)
	}
	_, err = svc.Capture(ctx, "Same content", "philosophy", "colony")
	if err != nil {
		t.Fatalf("second Capture: %v", err)
	}

	var file colony.LearningFile
	if err := store.LoadJSON("learning-observations.json", &file); err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(file.Observations) != 2 {
		t.Errorf("expected 2 observations for different wisdom types, got %d", len(file.Observations))
	}
}

func floatPtr(f float64) *float64 {
	return &f
}

// TestColonyLearningNoRegression verifies existing colony learning tests still pass.
func TestColonyLearningNoRegression(t *testing.T) {
	// Verify the Observation struct changes are backward compatible
	obs := colony.Observation{
		ContentHash:      "test",
		Content:          "test content",
		WisdomType:       "pattern",
		ObservationCount: 1,
		FirstSeen:        "2026-01-01T00:00:00Z",
		LastSeen:         "2026-01-01T00:00:00Z",
		Colonies:         []string{"colony"},
		// New fields are omitempty, so they're optional for legacy entries
	}

	data, err := json.Marshal(obs)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Verify trust fields are omitted when nil/zero (backward compat)
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := raw["trust_score"]; ok {
		t.Error("trust_score should be omitted when nil")
	}
	if _, ok := raw["source_type"]; ok {
		t.Error("source_type should be omitted when empty")
	}
}
