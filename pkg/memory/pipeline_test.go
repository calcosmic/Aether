package memory

import (
	"context"
	"testing"
	"time"

	"github.com/aether-colony/aether/pkg/colony"
	"github.com/aether-colony/aether/pkg/events"
	"github.com/aether-colony/aether/pkg/storage"
)

func newTestPipeline(t *testing.T) (*Pipeline, func()) {
	t.Helper()
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	bus := events.NewBus(store, events.Config{JSONLFile: "events.jsonl"})
	config := PipelineConfig{
		ColonyName: "test-colony",
		QueenPath:  "QUEEN.md",
	}
	p := NewPipeline(store, bus, config)
	return p, func() {}
}

// TestPipeline_NewPipeline verifies all services are wired.
func TestPipeline_NewPipeline(t *testing.T) {
	p, cleanup := newTestPipeline(t)
	defer cleanup()

	if p.Observe == nil {
		t.Error("Observe service is nil")
	}
	if p.Promote == nil {
		t.Error("Promote service is nil")
	}
	if p.Queen == nil {
		t.Error("Queen service is nil")
	}
	if p.Consolidate == nil {
		t.Error("Consolidate service is nil")
	}
	if p.config.ColonyName != "test-colony" {
		t.Errorf("ColonyName = %q, want %q", p.config.ColonyName, "test-colony")
	}
}

// TestPipeline_CapturePromotes verifies that capturing 3 identical observations
// results in an instinct being created (either via event-driven auto-promotion
// or by checking promotion eligibility after each capture).
func TestPipeline_CapturePromotes(t *testing.T) {
	p, cleanup := newTestPipeline(t)
	defer cleanup()

	ctx := context.Background()
	if err := p.Start(ctx); err != nil {
		t.Fatalf("start pipeline: %v", err)
	}
	defer p.Stop()

	// Capture same observation 3 times
	for i := 0; i < 3; i++ {
		result, err := p.Observe.Capture(ctx, "always validate inputs before processing", "pattern", "test-colony")
		if err != nil {
			t.Fatalf("capture %d: %v", i+1, err)
		}
		if !result.IsNew && i == 0 {
			t.Error("first capture should be new")
		}
		if i == 2 && !result.PromotionEligible {
			t.Error("3rd capture should be promotion-eligible")
		}
	}

	// Give the observeLoop goroutine time to process events
	time.Sleep(100 * time.Millisecond)

	// Verify instinct was created (either by auto-promote or directly)
	var instincts colony.InstinctsFile
	if err := p.store.LoadJSON("instincts.json", &instincts); err != nil {
		t.Fatalf("load instincts: %v", err)
	}

	found := false
	for _, inst := range instincts.Instincts {
		if !inst.Archived {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected at least one non-archived instinct after 3 captures")
	}
}

// TestPipeline_InstinctToQueen verifies that a promoted instinct with
// confidence >= 0.75 and application_count >= 3 appears in QUEEN.md.
func TestPipeline_InstinctToQueen(t *testing.T) {
	p, cleanup := newTestPipeline(t)
	defer cleanup()

	ctx := context.Background()

	// Pre-populate a high-confidence instinct with application_count >= 3
	now := events.FormatTimestamp(time.Now().UTC())
	instincts := colony.InstinctsFile{
		Version: "1.0",
		Instincts: []colony.InstinctEntry{
			{
				ID:         "inst_123_abcdef",
				Trigger:    "When processing user input, always sanitize first",
				Action:     "Sanitize all user input before processing",
				Domain:     "pattern",
				TrustScore: 0.85,
				TrustTier:  "trusted",
				Confidence: 0.80,
				Provenance: colony.InstinctProvenance{
					Source:           "sha256:test",
					CreatedAt:        now,
					ApplicationCount: 4,
				},
				Archived: false,
			},
		},
	}
	if err := p.store.SaveJSON("instincts.json", instincts); err != nil {
		t.Fatalf("save instincts: %v", err)
	}

	// Run consolidation — should identify queen-eligible and promote
	result, err := p.Consolidate.Run(ctx)
	if err != nil {
		t.Fatalf("consolidate: %v", err)
	}

	// Verify queen-eligible was detected
	if len(result.QueenEligible) == 0 {
		t.Error("expected queen-eligible instincts after consolidation")
	}

	// Run the full consolidation with promotion
	_, err = p.RunConsolidation(ctx)
	if err != nil {
		t.Fatalf("run consolidation: %v", err)
	}

	// Verify QUEEN.md was created with the pattern
	data, err := p.store.ReadFile("QUEEN.md")
	if err != nil {
		t.Fatalf("read QUEEN.md: %v", err)
	}
	content := string(data)
	if !containsStr(content, "sanitize") {
		t.Errorf("QUEEN.md missing pattern content, got: %s", content)
	}
}

// TestPipeline_Consolidation verifies that consolidation after populating old
// instincts runs decay, archival, and identifies promotion candidates.
func TestPipeline_Consolidation(t *testing.T) {
	p, cleanup := newTestPipeline(t)
	defer cleanup()

	ctx := context.Background()

	// Pre-populate instincts with old timestamps (90 days ago for decay)
	oldTime := "2025-12-01T00:00:00Z"
	instincts := colony.InstinctsFile{
		Version: "1.0",
		Instincts: []colony.InstinctEntry{
			{
				ID:         "inst_old_000001",
				Trigger:    "Old low-trust pattern",
				Action:     "Some action",
				Domain:     "pattern",
				TrustScore: 0.30,
				TrustTier:  "suspect",
				Confidence: 0.50,
				Provenance: colony.InstinctProvenance{
					CreatedAt:        oldTime,
					ApplicationCount: 0,
				},
				Archived: false,
			},
			{
				ID:         "inst_new_000002",
				Trigger:    "Recent high-trust pattern",
				Action:     "Some action",
				Domain:     "pattern",
				TrustScore: 0.85,
				TrustTier:  "trusted",
				Confidence: 0.80,
				Provenance: colony.InstinctProvenance{
					CreatedAt:        oldTime,
					ApplicationCount: 4,
				},
				Archived: false,
			},
		},
	}
	if err := p.store.SaveJSON("instincts.json", instincts); err != nil {
		t.Fatalf("save instincts: %v", err)
	}

	result, err := p.Consolidate.Run(ctx)
	if err != nil {
		t.Fatalf("consolidate: %v", err)
	}

	if result.InstinctsDecayed != 2 {
		t.Errorf("InstinctsDecayed = %d, want 2", result.InstinctsDecayed)
	}
	if result.InstinctsArchived < 1 {
		t.Errorf("InstinctsArchived = %d, want >= 1 (low trust should be archived)", result.InstinctsArchived)
	}

	// High-confidence instinct with 4 applications should be queen-eligible
	found := false
	for _, id := range result.QueenEligible {
		if id == "inst_new_000002" {
			found = true
		}
	}
	if !found {
		t.Error("expected inst_new_000002 to be queen-eligible")
	}
}

// TestPipeline_FullCycle verifies the complete data flow:
// capture observation -> auto-promote to instinct -> queen promotion.
func TestPipeline_FullCycle(t *testing.T) {
	p, cleanup := newTestPipeline(t)
	defer cleanup()

	ctx := context.Background()

	if err := p.Start(ctx); err != nil {
		t.Fatalf("start pipeline: %v", err)
	}
	defer p.Stop()

	// Step 1: Capture 3 identical observations to trigger auto-promotion
	for i := 0; i < 3; i++ {
		_, err := p.Observe.Capture(ctx, "always handle errors gracefully with user-friendly messages", "pattern", "test-colony")
		if err != nil {
			t.Fatalf("capture %d: %v", i+1, err)
		}
	}

	// Give observeLoop time to process
	time.Sleep(100 * time.Millisecond)

	// Step 2: Verify instinct exists
	var instincts colony.InstinctsFile
	if err := p.store.LoadJSON("instincts.json", &instincts); err != nil {
		t.Fatalf("load instincts: %v", err)
	}

	nonArchived := 0
	var promotedInstinct *colony.InstinctEntry
	for i := range instincts.Instincts {
		if !instincts.Instincts[i].Archived {
			nonArchived++
			promotedInstinct = &instincts.Instincts[i]
		}
	}
	if nonArchived == 0 {
		t.Fatal("expected at least one instinct after 3 captures")
	}

	// Step 3: Bump application count to make it queen-eligible
	if promotedInstinct != nil {
		promotedInstinct.Provenance.ApplicationCount = 4
		promotedInstinct.Confidence = 0.80
		p.store.SaveJSON("instincts.json", instincts)
	}

	// Step 4: Run consolidation which should promote to queen
	_, err := p.RunConsolidation(ctx)
	if err != nil {
		t.Fatalf("run consolidation: %v", err)
	}

	// Step 5: Verify QUEEN.md has content
	data, err := p.store.ReadFile("QUEEN.md")
	if err != nil {
		t.Fatalf("read QUEEN.md: %v", err)
	}
	if len(data) == 0 {
		t.Error("QUEEN.md is empty after full cycle")
	}

	// Step 6: Verify observations file exists
	var obsFile colony.LearningFile
	if err := p.store.LoadJSON("learning-observations.json", &obsFile); err != nil {
		t.Fatalf("load observations: %v", err)
	}
	if len(obsFile.Observations) == 0 {
		t.Error("observations file is empty")
	}
}

// TestPipeline_RunConsolidation_PromotesCandidates verifies that consolidation
// actually promotes candidates, not just identifies them.
func TestPipeline_RunConsolidation_PromotesCandidates(t *testing.T) {
	p, cleanup := newTestPipeline(t)
	defer cleanup()

	ctx := context.Background()

	// Pre-populate observations with count >= 3 (promotion eligible)
	now := events.FormatTimestamp(time.Now().UTC())
	obsFile := colony.LearningFile{
		Observations: []colony.Observation{
			{
				ContentHash:      "sha256:testeligible",
				Content:          "test eligible observation",
				WisdomType:       "pattern",
				ObservationCount: 3,
				FirstSeen:        now,
				LastSeen:         now,
				Colonies:         []string{"test-colony"},
				SourceType:       "observation",
				EvidenceType:     "anecdotal",
			},
		},
	}
	if err := p.store.SaveJSON("learning-observations.json", obsFile); err != nil {
		t.Fatalf("save observations: %v", err)
	}

	result, err := p.RunConsolidation(ctx)
	if err != nil {
		t.Fatalf("run consolidation: %v", err)
	}

	// Should have identified the observation as eligible
	if len(result.PromotionCandidates) == 0 {
		t.Error("expected promotion candidates from consolidation")
	}

	// Should have promoted to instinct
	var instincts colony.InstinctsFile
	if err := p.store.LoadJSON("instincts.json", &instincts); err != nil {
		t.Fatalf("load instincts: %v", err)
	}
	found := false
	for _, inst := range instincts.Instincts {
		if !inst.Archived {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected instinct created from promotion candidate")
	}
}

// TestPipeline_Stop verifies that Stop cancels the context and waits for goroutines
// without panicking or hanging.
func TestPipeline_Stop(t *testing.T) {
	p, cleanup := newTestPipeline(t)
	defer cleanup()

	ctx := context.Background()
	if err := p.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}

	// Stop should complete without hanging (wg.Wait returns)
	done := make(chan struct{})
	go func() {
		p.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Stop() hung")
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
