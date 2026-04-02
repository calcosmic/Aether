package curation

import (
	"context"

	"github.com/aether-colony/aether/pkg/agent"
	"github.com/aether-colony/aether/pkg/events"
	"github.com/aether-colony/aether/pkg/storage"
)

// Archivist archives low-trust instincts. Instincts with confidence < 0.30
// are marked as "archived" to indicate they should not influence colony
// behavior. The instinct data is preserved but flagged for future pruning.
type Archivist struct {
	store *storage.Store
}

// NewArchivist creates a new Archivist ant backed by the given store.
func NewArchivist(store *storage.Store) *Archivist {
	return &Archivist{store: store}
}

// Name returns the unique identifier for this agent.
func (a *Archivist) Name() string { return "archivist" }

// Caste returns the agent's role category.
func (a *Archivist) Caste() agent.Caste { return agent.CasteCurator }

// Triggers returns nil because the orchestrator handles triggering.
func (a *Archivist) Triggers() []agent.Trigger { return nil }

// Execute runs the archivist. It delegates to Run.
func (a *Archivist) Execute(ctx context.Context, event events.Event) error {
	_, err := a.Run(ctx, false)
	return err
}

// Run reads instincts and marks those with confidence < 0.30 as archived.
// Returns the count of archived instincts.
func (a *Archivist) Run(ctx context.Context, dryRun bool) (StepResult, error) {
	var instincts []map[string]any
	if err := a.store.LoadJSON("instincts.json", &instincts); err != nil {
		return StepResult{
			Name:    "archivist",
			Success: true,
			Summary: map[string]any{"archived": 0},
		}, nil
	}

	archived := 0
	for i, inst := range instincts {
		conf, _ := inst["confidence"].(float64)
		status, _ := inst["status"].(string)
		if conf < 0.30 && status != "archived" {
			instincts[i]["status"] = "archived"
			archived++
		}
	}

	if !dryRun && archived > 0 {
		_ = a.store.SaveJSON("instincts.json", instincts)
	}

	return StepResult{
		Name:    "archivist",
		Success: true,
		Summary: map[string]any{"archived": archived},
	}, nil
}
