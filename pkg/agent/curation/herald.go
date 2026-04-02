package curation

import (
	"context"

	"github.com/aether-colony/aether/pkg/agent"
	"github.com/aether-colony/aether/pkg/events"
	"github.com/aether-colony/aether/pkg/storage"
)

// Herald promotes high-confidence instincts to QUEEN.md. It reads instincts
// with confidence >= 0.80 and writes their trigger patterns to the Patterns
// section of QUEEN.md, following the queen.PromoteInstinct pattern.
type Herald struct {
	store *storage.Store
}

// NewHerald creates a new Herald ant backed by the given store.
func NewHerald(store *storage.Store) *Herald {
	return &Herald{store: store}
}

// Name returns the unique identifier for this agent.
func (h *Herald) Name() string { return "herald" }

// Caste returns the agent's role category.
func (h *Herald) Caste() agent.Caste { return agent.CasteCurator }

// Triggers returns nil because the orchestrator handles triggering.
func (h *Herald) Triggers() []agent.Trigger { return nil }

// Execute runs the herald. It delegates to Run.
func (h *Herald) Execute(ctx context.Context, event events.Event) error {
	_, err := h.Run(ctx, false)
	return err
}

// Run promotes instincts with confidence >= 0.80 to QUEEN.md.
func (h *Herald) Run(ctx context.Context, dryRun bool) (StepResult, error) {
	var instincts []map[string]any
	if err := h.store.LoadJSON("instincts.json", &instincts); err != nil {
		return StepResult{
			Name:    "herald",
			Success: true,
			Summary: map[string]any{"promoted": 0},
		}, nil
	}

	promoted := 0
	var patterns []string
	for _, inst := range instincts {
		conf, _ := inst["confidence"].(float64)
		if conf >= 0.80 {
			trigger, _ := inst["trigger"].(string)
			if trigger != "" {
				patterns = append(patterns, trigger)
				promoted++
			}
		}
	}

	if !dryRun && len(patterns) > 0 {
		// Read existing QUEEN.md or create new one
		content := "# QUEEN.md\n\n## Patterns\n\n"
		if data, err := h.store.ReadFile("QUEEN.md"); err == nil {
			content = string(data)
		}
		// Append promoted patterns
		for _, p := range patterns {
			content += "- " + p + "\n"
		}
		_ = h.store.AtomicWrite("QUEEN.md", []byte(content))
	}

	return StepResult{
		Name:    "herald",
		Success: true,
		Summary: map[string]any{"promoted": promoted},
	}, nil
}
