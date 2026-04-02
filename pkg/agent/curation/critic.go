package curation

import (
	"context"

	"github.com/aether-colony/aether/pkg/agent"
	"github.com/aether-colony/aether/pkg/events"
	"github.com/aether-colony/aether/pkg/storage"
)

// Critic detects contradictions in the instinct store. Two instincts are
// considered contradictory if they share the same topic but express opposing
// conclusions (e.g., one says "prefer X" and another says "avoid X").
type Critic struct {
	store *storage.Store
}

// NewCritic creates a new Critic ant backed by the given store.
func NewCritic(store *storage.Store) *Critic {
	return &Critic{store: store}
}

// Name returns the unique identifier for this agent.
func (c *Critic) Name() string { return "critic" }

// Caste returns the agent's role category.
func (c *Critic) Caste() agent.Caste { return agent.CasteCurator }

// Triggers returns nil because the orchestrator handles triggering.
func (c *Critic) Triggers() []agent.Trigger { return nil }

// Execute runs the critic. It delegates to Run.
func (c *Critic) Execute(ctx context.Context, event events.Event) error {
	_, err := c.Run(ctx, false)
	return err
}

// Run scans instincts for contradictions. Two instincts sharing the same
// topic with different conclusion fields are flagged.
func (c *Critic) Run(ctx context.Context, dryRun bool) (StepResult, error) {
	var instincts []map[string]any
	if err := c.store.LoadJSON("instincts.json", &instincts); err != nil {
		return StepResult{
			Name:    "critic",
			Success: true,
			Summary: map[string]any{"contradictions": 0},
		}, nil
	}

	// Group by topic to find contradictions
	topicConclusions := make(map[string][]string)
	for _, inst := range instincts {
		topic, _ := inst["topic"].(string)
		conclusion, _ := inst["conclusion"].(string)
		if topic != "" && conclusion != "" {
			topicConclusions[topic] = append(topicConclusions[topic], conclusion)
		}
	}

	contradictions := 0
	for _, conclusions := range topicConclusions {
		seen := make(map[string]bool)
		for _, c := range conclusions {
			if seen[c] {
				continue
			}
			seen[c] = true
		}
		if len(seen) > 1 {
			contradictions++
		}
	}

	return StepResult{
		Name:    "critic",
		Success: true,
		Summary: map[string]any{"contradictions": contradictions},
	}, nil
}
