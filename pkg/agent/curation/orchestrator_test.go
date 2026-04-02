package curation

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/aether-colony/aether/pkg/agent"
	"github.com/aether-colony/aether/pkg/events"
	"github.com/aether-colony/aether/pkg/storage"
)

// setupTestStore creates a temp directory with valid JSON for the 6 stores
// that sentinel checks, plus an event bus backed by the store.
func setupTestStore(t *testing.T) (*storage.Store, *events.Bus) {
	t.Helper()
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	// Create valid JSON for each checked store
	for _, name := range []string{
		"learning-observations.json",
		"instincts.json",
		"instinct-graph.json",
		"pheromones.json",
		"COLONY_STATE.json",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("{}"), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	// Create empty JSONL for event bus
	if err := os.WriteFile(filepath.Join(dir, "event-bus.jsonl"), []byte(""), 0644); err != nil {
		t.Fatalf("write event-bus.jsonl: %v", err)
	}

	bus := events.NewBus(store, events.DefaultConfig())
	return store, bus
}

// TestOrchestratorImplementsAgent verifies Orchestrator satisfies agent.Agent.
func TestOrchestratorImplementsAgent(t *testing.T) {
	var _ agent.Agent = (*Orchestrator)(nil)
}

// TestOrchestratorName verifies Name returns "curation-orchestrator".
func TestOrchestratorName(t *testing.T) {
	store, bus := setupTestStore(t)
	o := NewOrchestrator(store, bus)
	if got := o.Name(); got != "curation-orchestrator" {
		t.Errorf("Name() = %q, want %q", got, "curation-orchestrator")
	}
}

// TestOrchestratorCaste verifies Caste returns CasteCurator.
func TestOrchestratorCaste(t *testing.T) {
	store, bus := setupTestStore(t)
	o := NewOrchestrator(store, bus)
	if got := o.Caste(); got != agent.CasteCurator {
		t.Errorf("Caste() = %q, want %q", got, agent.CasteCurator)
	}
}

// TestOrchestratorTriggers verifies Triggers returns consolidation.* and phase.end.
func TestOrchestratorTriggers(t *testing.T) {
	store, bus := setupTestStore(t)
	o := NewOrchestrator(store, bus)
	triggers := o.Triggers()
	if len(triggers) != 2 {
		t.Fatalf("Triggers() returned %d triggers, want 2", len(triggers))
	}
	topics := map[string]bool{}
	for _, tr := range triggers {
		topics[tr.Topic] = true
	}
	if !topics["consolidation.*"] {
		t.Error("Triggers() missing consolidation.*")
	}
	if !topics["phase.end"] {
		t.Error("Triggers() missing phase.end")
	}
}

// TestOrchestratorRunAllSteps runs orchestrator with all healthy stores and
// verifies all 8 steps executed successfully.
func TestOrchestratorRunAllSteps(t *testing.T) {
	store, bus := setupTestStore(t)
	o := NewOrchestrator(store, bus)

	result, err := o.Run(context.Background(), false)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if result.Succeeded != 8 {
		t.Errorf("Succeeded = %d, want 8", result.Succeeded)
	}
	if result.Failed != 0 {
		t.Errorf("Failed = %d, want 0", result.Failed)
	}
	if result.Skipped != 0 {
		t.Errorf("Skipped = %d, want 0", result.Skipped)
	}
	if len(result.Steps) != 8 {
		t.Errorf("Steps count = %d, want 8", len(result.Steps))
	}
}

// TestOrchestratorSentinelAbort creates a corrupt JSON file and verifies
// that sentinel fails and remaining 7 steps are skipped.
func TestOrchestratorSentinelAbort(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	// Create valid files for most stores but corrupt one
	for _, name := range []string{
		"learning-observations.json",
		"instinct-graph.json",
		"pheromones.json",
		"COLONY_STATE.json",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("{}"), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	// Corrupt instincts.json
	if err := os.WriteFile(filepath.Join(dir, "instincts.json"), []byte("{broken json!!"), 0644); err != nil {
		t.Fatalf("write corrupt instincts.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "event-bus.jsonl"), []byte(""), 0644); err != nil {
		t.Fatalf("write event-bus.jsonl: %v", err)
	}

	bus := events.NewBus(store, events.DefaultConfig())
	o := NewOrchestrator(store, bus)

	result, err := o.Run(context.Background(), false)
	if err == nil {
		t.Fatal("Run() should return error on sentinel abort")
	}

	// Sentinel should fail
	if result.Failed < 1 {
		t.Errorf("Failed = %d, want >= 1", result.Failed)
	}
	// Remaining 7 steps should be skipped
	if result.Skipped != 7 {
		t.Errorf("Skipped = %d, want 7", result.Skipped)
	}
	// Only sentinel's step result plus 7 skipped = 8 total
	if len(result.Steps) != 8 {
		t.Errorf("Steps count = %d, want 8", len(result.Steps))
	}

	// Verify sentinel step failed
	if result.Steps[0].Name != "sentinel" {
		t.Errorf("First step = %q, want %q", result.Steps[0].Name, "sentinel")
	}
	if result.Steps[0].Success {
		t.Error("Sentinel step should not be successful with corrupt stores")
	}

	// Verify remaining steps are skipped
	for i := 1; i < 8; i++ {
		if result.Steps[i].Success {
			t.Errorf("Step %d (%s) should be skipped, got success=true", i, result.Steps[i].Name)
		}
		reason, _ := result.Steps[i].Summary["reason"].(string)
		if reason == "" {
			t.Errorf("Step %d (%s) should have skip reason", i, result.Steps[i].Name)
		}
	}
}

// TestOrchestratorDryRun verifies that steps receive the dryRun flag.
func TestOrchestratorDryRun(t *testing.T) {
	store, bus := setupTestStore(t)
	o := NewOrchestrator(store, bus)

	result, err := o.Run(context.Background(), true)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if !result.DryRun {
		t.Error("DryRun = false, want true")
	}
	if result.Succeeded != 8 {
		t.Errorf("Succeeded = %d, want 8", result.Succeeded)
	}
}

// TestOrchestratorStepOrder verifies steps execute in the exact shell-matching order.
func TestOrchestratorStepOrder(t *testing.T) {
	store, bus := setupTestStore(t)
	o := NewOrchestrator(store, bus)

	result, err := o.Run(context.Background(), false)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	expected := []string{"sentinel", "nurse", "critic", "herald", "janitor", "archivist", "librarian", "scribe"}
	for i, want := range expected {
		if i >= len(result.Steps) {
			t.Errorf("Missing step %d: want %q", i, want)
			continue
		}
		if got := result.Steps[i].Name; got != want {
			t.Errorf("Step %d = %q, want %q", i, got, want)
		}
	}
}

// TestEachAntImplementsAgent verifies each of the 8 ants satisfies agent.Agent.
func TestEachAntImplementsAgent(t *testing.T) {
	store, bus := setupTestStore(t)

	ants := map[string]agent.Agent{
		"sentinel":  NewSentinel(store),
		"nurse":     NewNurse(store),
		"critic":    NewCritic(store),
		"herald":    NewHerald(store),
		"janitor":   NewJanitor(store, bus),
		"archivist": NewArchivist(store),
		"librarian": NewLibrarian(store, bus),
		"scribe":    NewScribe(),
	}

	expectedNames := map[string]string{
		"sentinel":  "sentinel",
		"nurse":     "nurse",
		"critic":    "critic",
		"herald":    "herald",
		"janitor":   "janitor",
		"archivist": "archivist",
		"librarian": "librarian",
		"scribe":    "scribe",
	}

	for key, a := range ants {
		if a.Name() != expectedNames[key] {
			t.Errorf("%s.Name() = %q, want %q", key, a.Name(), expectedNames[key])
		}
		if a.Caste() != agent.CasteCurator {
			t.Errorf("%s.Caste() = %q, want %q", key, a.Caste(), agent.CasteCurator)
		}
	}
}

// TestScribeRunResult verifies scribe returns a valid StepResult.
func TestScribeRunResult(t *testing.T) {
	s := NewScribe()
	sr, err := s.Run(context.Background(), false)
	if err != nil {
		t.Fatalf("Scribe.Run() error: %v", err)
	}
	if sr.Name != "scribe" {
		t.Errorf("Name = %q, want %q", sr.Name, "scribe")
	}
	if !sr.Success {
		t.Error("Success = false, want true")
	}
	if sr.Summary == nil {
		t.Error("Summary is nil")
	}
}

// TestSentinelMissingStores verifies sentinel passes when stores are missing.
func TestSentinelMissingStores(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	s := NewSentinel(store)
	sr, err := s.Run(context.Background(), false)
	if err != nil {
		t.Fatalf("Sentinel.Run() with missing stores should not error: %v", err)
	}
	if !sr.Success {
		t.Error("Sentinel should succeed when stores are simply missing")
	}
}

// TestJanitorCleanup verifies janitor calls bus.Cleanup.
func TestJanitorCleanup(t *testing.T) {
	store, bus := setupTestStore(t)

	// Add an expired event to the JSONL
	expiredEvent := map[string]any{
		"id":         "evt_100_abcd",
		"topic":      "test.topic",
		"payload":    json.RawMessage(`{}`),
		"source":     "test",
		"timestamp":  "2020-01-01T00:00:00Z",
		"ttl_days":   30,
		"expires_at": "2020-01-31T00:00:00Z",
	}
	data, _ := json.Marshal(expiredEvent)
	if err := store.AppendJSONL("event-bus.jsonl", expiredEvent); err != nil {
		t.Fatalf("append event: %v", err)
	}
	_ = data

	j := NewJanitor(store, bus)
	sr, err := j.Run(context.Background(), false)
	if err != nil {
		t.Fatalf("Janitor.Run() error: %v", err)
	}
	if !sr.Success {
		t.Error("Janitor should succeed")
	}
	removed, _ := sr.Summary["removed"].(int)
	if removed != 1 {
		t.Errorf("removed = %v, want 1", sr.Summary["removed"])
	}
}
