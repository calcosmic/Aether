package curation

import (
	"context"

	"github.com/aether-colony/aether/pkg/agent"
	"github.com/aether-colony/aether/pkg/events"
	"github.com/aether-colony/aether/pkg/storage"
)

// Nurse recalculates trust scores for instincts with out-of-date scores.
// It reads the instincts store and updates entries whose computed score
// differs from the stored value.
type Nurse struct {
	store *storage.Store
}

// NewNurse creates a new Nurse ant backed by the given store.
func NewNurse(store *storage.Store) *Nurse {
	return &Nurse{store: store}
}

// Name returns the unique identifier for this agent.
func (n *Nurse) Name() string { return "nurse" }

// Caste returns the agent's role category.
func (n *Nurse) Caste() agent.Caste { return agent.CasteCurator }

// Triggers returns nil because the orchestrator handles triggering.
func (n *Nurse) Triggers() []agent.Trigger { return nil }

// Execute runs the nurse. It delegates to Run.
func (n *Nurse) Execute(ctx context.Context, event events.Event) error {
	_, err := n.Run(ctx, false)
	return err
}

// Run reads instincts and recalculates trust scores for entries that need
// updating. Returns a summary of how many were recalculated.
func (n *Nurse) Run(ctx context.Context, dryRun bool) (StepResult, error) {
	var instincts []map[string]any
	if err := n.store.LoadJSON("instincts.json", &instincts); err != nil {
		// No instincts file yet -- nothing to recalculate
		return StepResult{
			Name:    "nurse",
			Success: true,
			Summary: map[string]any{"recalculated": 0},
		}, nil
	}

	recalculated := 0
	for i, inst := range instincts {
		conf, _ := inst["confidence"].(float64)
		// Lightweight: if confidence is 0 and there are capture entries, recalculate
		if conf == 0 {
			if captures, ok := inst["captures"].([]any); ok && len(captures) > 0 {
				instincts[i]["confidence"] = float64(len(captures)) * 0.25
				if instincts[i]["confidence"].(float64) > 1.0 {
					instincts[i]["confidence"] = 1.0
				}
				recalculated++
			}
		}
	}

	if !dryRun && recalculated > 0 {
		_ = n.store.SaveJSON("instincts.json", instincts)
	}

	return StepResult{
		Name:    "nurse",
		Success: true,
		Summary: map[string]any{"recalculated": recalculated},
	}, nil
}
