package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aether-colony/aether/pkg/colony"
	"github.com/aether-colony/aether/pkg/events"
	"github.com/aether-colony/aether/pkg/storage"
)

func newTestPromoteService(t *testing.T) (*PromoteService, *storage.Store, string) {
	t.Helper()
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	bus := events.NewBus(store, events.DefaultConfig())
	t.Cleanup(func() { bus.Close() })
	svc := NewPromoteService(store, bus)
	return svc, store, dir
}

func TestPromoteNewInstinct(t *testing.T) {
	svc, store, _ := newTestPromoteService(t)
	ctx := context.Background()

	obs := colony.Observation{
		ContentHash:      "abc123",
		Content:          "Use table-driven tests for all verification",
		WisdomType:       "testing",
		ObservationCount: 3,
		FirstSeen:        "2026-03-01T10:00:00Z",
		LastSeen:         "2026-03-15T10:00:00Z",
	}

	result, err := svc.Promote(ctx, obs, "test-colony")
	if err != nil {
		t.Fatalf("Promote: %v", err)
	}

	if !result.IsNew {
		t.Error("expected IsNew=true for first promotion")
	}
	if result.WasDeduped {
		t.Error("expected WasDeduped=false for first promotion")
	}

	entry := result.Instinct

	// ID format: inst_{unix}_{6hex}
	if !strings.HasPrefix(entry.ID, "inst_") {
		t.Errorf("ID should start with 'inst_', got %q", entry.ID)
	}
	parts := strings.SplitN(entry.ID, "_", 3)
	if len(parts) != 3 {
		t.Errorf("ID should have 3 parts separated by _, got %d parts", len(parts))
	}
	if len(parts[2]) != 6 {
		t.Errorf("ID hex suffix should be 6 chars, got %d", len(parts[2]))
	}

	// Trigger matches observation content
	if entry.Trigger != obs.Content {
		t.Errorf("Trigger = %q, want %q", entry.Trigger, obs.Content)
	}

	// Domain matches wisdom type
	if entry.Domain != obs.WisdomType {
		t.Errorf("Domain = %q, want %q", entry.Domain, obs.WisdomType)
	}

	// Trust score is calculated
	if entry.TrustScore < 0.2 {
		t.Errorf("TrustScore = %f, want >= 0.2", entry.TrustScore)
	}

	// Trust tier is assigned
	if entry.TrustTier == "" {
		t.Error("TrustTier should not be empty")
	}

	// Confidence matches recurrence formula
	wantConf := RecurrenceConfidence(obs.ObservationCount)
	if math.Abs(entry.Confidence-wantConf) > 1e-6 {
		t.Errorf("Confidence = %f, want %f", entry.Confidence, wantConf)
	}

	// Full provenance
	if entry.Provenance.Source != obs.ContentHash {
		t.Errorf("Provenance.Source = %q, want %q", entry.Provenance.Source, obs.ContentHash)
	}
	if entry.Provenance.SourceType != obs.SourceType {
		t.Errorf("Provenance.SourceType = %q, want %q", entry.Provenance.SourceType, obs.SourceType)
	}
	if entry.Provenance.CreatedAt == "" {
		t.Error("Provenance.CreatedAt should not be empty")
	}
	if entry.Provenance.LastApplied != nil {
		t.Error("Provenance.LastApplied should be nil for new instinct")
	}
	if entry.Provenance.ApplicationCount != 0 {
		t.Errorf("Provenance.ApplicationCount = %d, want 0", entry.Provenance.ApplicationCount)
	}

	// Not archived
	if entry.Archived {
		t.Error("new instinct should not be archived")
	}

	// Verify file was saved
	var file colony.InstinctsFile
	if err := store.LoadJSON("instincts.json", &file); err != nil {
		t.Fatalf("load instincts file: %v", err)
	}
	if len(file.Instincts) != 1 {
		t.Fatalf("file has %d instincts, want 1", len(file.Instincts))
	}
	if file.Instincts[0].ID != entry.ID {
		t.Errorf("saved ID = %q, want %q", file.Instincts[0].ID, entry.ID)
	}
}

func TestPromoteDedup_TriggerPrefix(t *testing.T) {
	svc, store, _ := newTestPromoteService(t)
	ctx := context.Background()

	// First 50 chars match (both have 40 'a' chars + "PREFIX_MATCH_HERE")
	content1 := strings.Repeat("a", 40) + "PREFIX_MATCH_HERE_and_more_text_after_fifty_chars"
	content2 := strings.Repeat("a", 40) + "PREFIX_MATCH_HERE_but_different_text_later_on"

	obs1 := colony.Observation{
		ContentHash:      "hash1",
		Content:          content1,
		WisdomType:       "testing",
		ObservationCount: 2,
		FirstSeen:        "2026-03-01T10:00:00Z",
	}
	obs2 := colony.Observation{
		ContentHash:      "hash2",
		Content:          content2,
		WisdomType:       "testing",
		ObservationCount: 4,
		FirstSeen:        "2026-03-01T10:00:00Z",
	}

	_, err := svc.Promote(ctx, obs1, "colony")
	if err != nil {
		t.Fatalf("first promote: %v", err)
	}

	result, err := svc.Promote(ctx, obs2, "colony")
	if err != nil {
		t.Fatalf("second promote: %v", err)
	}

	if result.IsNew {
		t.Error("expected IsNew=false for dedup")
	}
	if !result.WasDeduped {
		t.Error("expected WasDeduped=true for dedup")
	}

	// Should still only have 1 instinct in file
	var file colony.InstinctsFile
	if err := store.LoadJSON("instincts.json", &file); err != nil {
		t.Fatalf("load instincts: %v", err)
	}
	if len(file.Instincts) != 1 {
		t.Errorf("expected 1 instinct after dedup, got %d", len(file.Instincts))
	}

	// Confidence should be incremented
	if file.Instincts[0].Confidence <= RecurrenceConfidence(2) {
		t.Errorf("confidence should increase after dedup promotion, got %f", file.Instincts[0].Confidence)
	}
}

func TestPromoteDedup_DifferentAfter50(t *testing.T) {
	svc, store, _ := newTestPromoteService(t)
	ctx := context.Background()

	// These differ at position 30 (well before 50), so they DO NOT dedup
	content1 := strings.Repeat("a", 30) + "FIRST" + strings.Repeat("x", 20)
	content2 := strings.Repeat("a", 30) + "SECOND" + strings.Repeat("x", 20)

	obs1 := colony.Observation{
		ContentHash:      "hash1",
		Content:          content1,
		WisdomType:       "testing",
		ObservationCount: 2,
		FirstSeen:        "2026-03-01T10:00:00Z",
	}
	obs2 := colony.Observation{
		ContentHash:      "hash2",
		Content:          content2,
		WisdomType:       "testing",
		ObservationCount: 2,
		FirstSeen:        "2026-03-01T10:00:00Z",
	}

	_, err := svc.Promote(ctx, obs1, "colony")
	if err != nil {
		t.Fatalf("first promote: %v", err)
	}

	result, err := svc.Promote(ctx, obs2, "colony")
	if err != nil {
		t.Fatalf("second promote: %v", err)
	}

	if !result.IsNew {
		t.Error("expected IsNew=true for different content")
	}
	if result.WasDeduped {
		t.Error("expected WasDeduped=false for different content")
	}

	var file colony.InstinctsFile
	if err := store.LoadJSON("instincts.json", &file); err != nil {
		t.Fatalf("load instincts: %v", err)
	}
	if len(file.Instincts) != 2 {
		t.Errorf("expected 2 instincts for different content, got %d", len(file.Instincts))
	}
}

func TestPromote_InstinctCap(t *testing.T) {
	svc, store, _ := newTestPromoteService(t)
	ctx := context.Background()

	// Fill up to 50 instincts with unique trigger prefixes (first 50 chars must differ)
	for i := 0; i < 50; i++ {
		obs := colony.Observation{
			ContentHash:      fmt.Sprintf("hash_%02d_unique_pad", i),
			Content:          fmt.Sprintf("instinct_%02d_unique_trigger_prefix_to_avoid_dedup_pad", i),
			WisdomType:       "testing",
			ObservationCount: 2,
			FirstSeen:        "2026-03-01T10:00:00Z",
		}
		_, err := svc.Promote(ctx, obs, "colony")
		if err != nil {
			t.Fatalf("promote %d: %v", i, err)
		}
	}

	var file colony.InstinctsFile
	if err := store.LoadJSON("instincts.json", &file); err != nil {
		t.Fatalf("load instincts: %v", err)
	}
	nonArchived := 0
	for _, inst := range file.Instincts {
		if !inst.Archived {
			nonArchived++
		}
	}
	if nonArchived != 50 {
		t.Errorf("expected 50 non-archived, got %d", nonArchived)
	}

	// Promote one more -- should evict the lowest-trust
	obs := colony.Observation{
		ContentHash:      "overflow_hash",
		Content:          "overflow trigger text causing eviction of lowest",
		WisdomType:       "testing",
		ObservationCount: 3,
		FirstSeen:        "2026-03-01T10:00:00Z",
	}
	result, err := svc.Promote(ctx, obs, "colony")
	if err != nil {
		t.Fatalf("overflow promote: %v", err)
	}
	if !result.IsNew {
		t.Error("overflow should create new instinct")
	}

	if err := store.LoadJSON("instincts.json", &file); err != nil {
		t.Fatalf("load instincts after overflow: %v", err)
	}
	nonArchived = 0
	archived := 0
	for _, inst := range file.Instincts {
		if inst.Archived {
			archived++
		} else {
			nonArchived++
		}
	}
	if nonArchived != 50 {
		t.Errorf("expected 50 non-archived after eviction, got %d", nonArchived)
	}
	if archived < 1 {
		t.Error("expected at least 1 archived entry after eviction")
	}
}

func TestPromote_ArchivedSkip(t *testing.T) {
	svc, store, _ := newTestPromoteService(t)
	ctx := context.Background()

	content := "archived test trigger that is long enough to be meaningful"

	obs := colony.Observation{
		ContentHash:      "hash1",
		Content:          content,
		WisdomType:       "testing",
		ObservationCount: 2,
		FirstSeen:        "2026-03-01T10:00:00Z",
	}

	// Promote first
	_, err := svc.Promote(ctx, obs, "colony")
	if err != nil {
		t.Fatalf("first promote: %v", err)
	}

	// Archive it manually
	var file colony.InstinctsFile
	if err := store.LoadJSON("instincts.json", &file); err != nil {
		t.Fatalf("load: %v", err)
	}
	file.Instincts[0].Archived = true
	if err := store.SaveJSON("instincts.json", file); err != nil {
		t.Fatalf("save archived: %v", err)
	}

	// Promote same content again -- should create NEW (not dedup) since archived is skipped
	result, err := svc.Promote(ctx, obs, "colony")
	if err != nil {
		t.Fatalf("second promote: %v", err)
	}

	if !result.IsNew {
		t.Error("expected IsNew=true when existing is archived")
	}
	if result.WasDeduped {
		t.Error("expected WasDeduped=false when existing is archived")
	}

	if err := store.LoadJSON("instincts.json", &file); err != nil {
		t.Fatalf("load final: %v", err)
	}
	if len(file.Instincts) != 2 {
		t.Errorf("expected 2 instincts (1 archived + 1 new), got %d", len(file.Instincts))
	}
}

func TestPromote_GraphEdge(t *testing.T) {
	svc, _, dir := newTestPromoteService(t)
	ctx := context.Background()

	obs := colony.Observation{
		ContentHash:      "graph_test_hash",
		Content:          "graph edge test trigger content",
		WisdomType:       "testing",
		ObservationCount: 2,
		FirstSeen:        "2026-03-01T10:00:00Z",
	}

	result, err := svc.Promote(ctx, obs, "colony")
	if err != nil {
		t.Fatalf("Promote: %v", err)
	}

	// Verify graph file was created
	graphPath := filepath.Join(dir, "instinct-graph.json")
	data, err := os.ReadFile(graphPath)
	if err != nil {
		t.Fatalf("read graph file: %v", err)
	}

	var graph struct {
		Edges []struct {
			From      string `json:"from"`
			To        string `json:"to"`
			EdgeType  string `json:"edge_type"`
			CreatedAt string `json:"created_at"`
		} `json:"edges"`
	}
	if err := json.Unmarshal(data, &graph); err != nil {
		t.Fatalf("parse graph: %v", err)
	}

	if len(graph.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(graph.Edges))
	}

	edge := graph.Edges[0]
	if edge.From != obs.ContentHash {
		t.Errorf("edge.From = %q, want %q", edge.From, obs.ContentHash)
	}
	if edge.To != result.Instinct.ID {
		t.Errorf("edge.To = %q, want %q", edge.To, result.Instinct.ID)
	}
	if edge.EdgeType != "promoted_from" {
		t.Errorf("edge.EdgeType = %q, want 'promoted_from'", edge.EdgeType)
	}
	if edge.CreatedAt == "" {
		t.Error("edge.CreatedAt should not be empty")
	}
}

func TestPromote_Event(t *testing.T) {
	svc, store, _ := newTestPromoteService(t)
	ctx := context.Background()

	obs := colony.Observation{
		ContentHash:      "event_test_hash",
		Content:          "event publishing test trigger content",
		WisdomType:       "testing",
		ObservationCount: 2,
		FirstSeen:        "2026-03-01T10:00:00Z",
	}

	_, err := svc.Promote(ctx, obs, "colony")
	if err != nil {
		t.Fatalf("Promote: %v", err)
	}

	// Verify event was published to JSONL
	time.Sleep(10 * time.Millisecond) // small delay for persistence
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
		if e.Topic == "instinct.promote" {
			found = true
			break
		}
	}
	if !found {
		t.Error("instinct.promote event not found in JSONL")
	}
}
