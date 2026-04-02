package curation

import (
	"context"
	"fmt"
	"os"

	"github.com/aether-colony/aether/pkg/agent"
	"github.com/aether-colony/aether/pkg/events"
	"github.com/aether-colony/aether/pkg/storage"
)

// Sentinel is the health-check curation ant. It verifies that colony data
// stores are not corrupt before allowing remaining curation steps to run.
// If any checked store contains invalid JSON, Sentinel returns an error
// and the orchestrator skips remaining steps.
type Sentinel struct {
	store *storage.Store
}

// NewSentinel creates a new Sentinel ant backed by the given store.
func NewSentinel(store *storage.Store) *Sentinel {
	return &Sentinel{store: store}
}

// sentinelCheckedStores lists the data files Sentinel validates.
// Missing files are OK (colony not yet populated); corrupt JSON is not.
var sentinelCheckedStores = []string{
	"learning-observations.json",
	"instincts.json",
	"instinct-graph.json",
	"event-bus.jsonl",
	"pheromones.json",
	"COLONY_STATE.json",
}

// Name returns the unique identifier for this agent.
func (s *Sentinel) Name() string { return "sentinel" }

// Caste returns the agent's role category.
func (s *Sentinel) Caste() agent.Caste { return agent.CasteCurator }

// Triggers returns nil because the orchestrator handles triggering.
func (s *Sentinel) Triggers() []agent.Trigger { return nil }

// Execute runs the sentinel check. It delegates to Run.
func (s *Sentinel) Execute(ctx context.Context, event events.Event) error {
	_, err := s.Run(ctx, false)
	return err
}

// Run checks each of the 6 colony data stores for corruption.
// A missing file is acceptable; a file with invalid JSON is corrupt.
// Returns a StepResult listing which stores (if any) are corrupt.
func (s *Sentinel) Run(ctx context.Context, dryRun bool) (StepResult, error) {
	var corrupt []string

	for _, path := range sentinelCheckedStores {
		fullPath := s.store.BasePath() + "/" + path
		data, err := os.ReadFile(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue // missing is OK
			}
			corrupt = append(corrupt, path)
			continue
		}
		// For .jsonl files, skip validation (line-delimited, not single JSON)
		if path == "event-bus.jsonl" {
			continue
		}
		if len(data) > 0 {
			var target map[string]any
			if err := s.store.LoadJSON(path, &target); err != nil {
				corrupt = append(corrupt, path)
			}
		}
	}

	if len(corrupt) > 0 {
		return StepResult{
			Name:    "sentinel",
			Success: false,
			Summary: map[string]any{
				"corrupt": corrupt,
				"checked": len(sentinelCheckedStores),
			},
		}, fmt.Errorf("sentinel: corrupt stores: %v", corrupt)
	}

	return StepResult{
		Name:    "sentinel",
		Success: true,
		Summary: map[string]any{
			"corrupt": []string{},
			"checked": len(sentinelCheckedStores),
		},
	}, nil
}
