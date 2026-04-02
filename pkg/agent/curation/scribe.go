package curation

import (
	"context"
	"fmt"
	"strings"

	"github.com/aether-colony/aether/pkg/agent"
	"github.com/aether-colony/aether/pkg/events"
)

// Scribe generates a text report summarizing the curation run results. It
// collects StepResults from the orchestrator and formats them into a
// human-readable report that can be persisted or displayed.
type Scribe struct{}

// NewScribe creates a new Scribe ant.
func NewScribe() *Scribe {
	return &Scribe{}
}

// Name returns the unique identifier for this agent.
func (s *Scribe) Name() string { return "scribe" }

// Caste returns the agent's role category.
func (s *Scribe) Caste() agent.Caste { return agent.CasteCurator }

// Triggers returns nil because the orchestrator handles triggering.
func (s *Scribe) Triggers() []agent.Trigger { return nil }

// Execute runs the scribe. It delegates to Run.
func (s *Scribe) Execute(ctx context.Context, event events.Event) error {
	_, err := s.Run(ctx, false)
	return err
}

// Run generates a text report from curation results. In a full integration
// it would receive prior step results, but as a lightweight stub it reports
// success with a summary indicating report generation completed.
func (s *Scribe) Run(ctx context.Context, dryRun bool) (StepResult, error) {
	var report strings.Builder
	report.WriteString("Curation Report\n")
	report.WriteString("===============\n\n")
	report.WriteString(fmt.Sprintf("Dry run: %v\n", dryRun))

	return StepResult{
		Name:    "scribe",
		Success: true,
		Summary: map[string]any{
			"report": report.String(),
			"path":   "",
		},
	}, nil
}
