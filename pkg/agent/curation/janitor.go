package curation

import (
	"context"

	"github.com/aether-colony/aether/pkg/agent"
	"github.com/aether-colony/aether/pkg/events"
	"github.com/aether-colony/aether/pkg/storage"
)

// Janitor cleans expired events from the event bus JSONL store. It delegates
// to bus.Cleanup which removes events past their TTL.
type Janitor struct {
	store *storage.Store
	bus   *events.Bus
}

// NewJanitor creates a new Janitor ant with access to the store and bus.
func NewJanitor(store *storage.Store, bus *events.Bus) *Janitor {
	return &Janitor{store: store, bus: bus}
}

// Name returns the unique identifier for this agent.
func (j *Janitor) Name() string { return "janitor" }

// Caste returns the agent's role category.
func (j *Janitor) Caste() agent.Caste { return agent.CasteCurator }

// Triggers returns nil because the orchestrator handles triggering.
func (j *Janitor) Triggers() []agent.Trigger { return nil }

// Execute runs the janitor. It delegates to Run.
func (j *Janitor) Execute(ctx context.Context, event events.Event) error {
	_, err := j.Run(ctx, false)
	return err
}

// Run calls bus.Cleanup to remove expired events. In dry-run mode it only
// reports what would be removed without modifying the JSONL file.
func (j *Janitor) Run(ctx context.Context, dryRun bool) (StepResult, error) {
	removed, remaining, err := j.bus.Cleanup(ctx, dryRun)
	if err != nil {
		return StepResult{
			Name:    "janitor",
			Success: false,
			Summary: map[string]any{"removed": 0, "remaining": 0},
		}, err
	}

	return StepResult{
		Name:    "janitor",
		Success: true,
		Summary: map[string]any{"removed": removed, "remaining": remaining},
	}, nil
}
