package curation

import (
	"context"

	"github.com/aether-colony/aether/pkg/agent"
	"github.com/aether-colony/aether/pkg/events"
	"github.com/aether-colony/aether/pkg/storage"
)

// Librarian produces inventory statistics across colony data stores. It
// counts entries in observations, instincts, events, and pheromones to
// give an overview of colony memory health.
type Librarian struct {
	store *storage.Store
	bus   *events.Bus
}

// NewLibrarian creates a new Librarian ant with access to the store and bus.
func NewLibrarian(store *storage.Store, bus *events.Bus) *Librarian {
	return &Librarian{store: store, bus: bus}
}

// Name returns the unique identifier for this agent.
func (l *Librarian) Name() string { return "librarian" }

// Caste returns the agent's role category.
func (l *Librarian) Caste() agent.Caste { return agent.CasteCurator }

// Triggers returns nil because the orchestrator handles triggering.
func (l *Librarian) Triggers() []agent.Trigger { return nil }

// Execute runs the librarian. It delegates to Run.
func (l *Librarian) Execute(ctx context.Context, event events.Event) error {
	_, err := l.Run(ctx, false)
	return err
}

// Run counts entries across data stores and returns inventory statistics.
func (l *Librarian) Run(ctx context.Context, dryRun bool) (StepResult, error) {
	countEntries := func(path string) int {
		var data []map[string]any
		if err := l.store.LoadJSON(path, &data); err != nil {
			return 0
		}
		return len(data)
	}

	observations := countEntries("learning-observations.json")
	instincts := countEntries("instincts.json")
	pheromones := countEntries("pheromones.json")

	// Count events from JSONL
	events := 0
	if lines, err := l.store.ReadJSONL("event-bus.jsonl"); err == nil {
		events = len(lines)
	}

	return StepResult{
		Name:    "librarian",
		Success: true,
		Summary: map[string]any{
			"observations": observations,
			"instincts":    instincts,
			"events":       events,
			"pheromones":   pheromones,
		},
	}, nil
}
