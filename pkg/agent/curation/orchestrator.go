// Package curation implements the memory maintenance system for colony data.
// It provides 8 curation ants (sentinel, nurse, critic, herald, janitor,
// archivist, librarian, scribe) and an orchestrator that runs them in a
// fixed sequential order matching the shell orchestrator.sh behavior.
// Sentinel abort prevents remaining steps when data corruption is detected.
package curation

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/storage"
)

// StepResult holds the outcome of a single curation step.
type StepResult struct {
	Name    string         `json:"name"`
	Success bool           `json:"success"`
	Error   error          `json:"-"`
	Summary map[string]any `json:"summary"`
}

// CurationResult holds the aggregate outcome of a full curation run.
type CurationResult struct {
	Steps      []StepResult `json:"steps"`
	Succeeded  int          `json:"succeeded"`
	Failed     int          `json:"failed"`
	Skipped    int          `json:"skipped"`
	DryRun     bool         `json:"dry_run"`
	DurationMs int64        `json:"duration_ms"`
}

// CurationAnt is the internal interface extended from agent.Agent that adds
// the Run method used by the orchestrator for sequential step execution.
type CurationAnt interface {
	agent.Agent
	Run(ctx context.Context, dryRun bool) (StepResult, error)
}

// Orchestrator runs all 8 curation ants in sequence. If the sentinel step
// detects corrupt stores, remaining steps are skipped. The orchestrator
// implements the Agent interface so it can be registered in the agent
// registry and triggered by the event bus.
type Orchestrator struct {
	store   *storage.Store
	bus     *events.Bus
	eventCh <-chan events.Event

	// All 8 curation ants
	sentinel  *Sentinel
	nurse     *Nurse
	critic    *Critic
	herald    *Herald
	janitor   *Janitor
	archivist *Archivist
	librarian *Librarian
	scribe    *Scribe

	// steps is the ordered slice for sequential execution
	steps []struct {
		name string
		ant  CurationAnt
	}

	mu sync.Mutex
}

// NewOrchestrator creates a curation orchestrator with all 8 ants in the
// correct sequential order matching the shell orchestrator.sh behavior:
// sentinel, nurse, critic, herald, janitor, archivist, librarian, scribe.
func NewOrchestrator(store *storage.Store, bus *events.Bus) *Orchestrator {
	o := &Orchestrator{
		store:     store,
		bus:       bus,
		sentinel:  NewSentinel(store),
		nurse:     NewNurse(store),
		critic:    NewCritic(store),
		herald:    NewHerald(store),
		janitor:   NewJanitor(store, bus),
		archivist: NewArchivist(store),
		librarian: NewLibrarian(store, bus),
		scribe:    NewScribe(),
	}

	// Fixed order matching shell orchestrator.sh
	o.steps = []struct {
		name string
		ant  CurationAnt
	}{
		{"sentinel", o.sentinel},
		{"nurse", o.nurse},
		{"critic", o.critic},
		{"herald", o.herald},
		{"janitor", o.janitor},
		{"archivist", o.archivist},
		{"librarian", o.librarian},
		{"scribe", o.scribe},
	}

	return o
}

// Name returns the unique identifier for this agent.
func (o *Orchestrator) Name() string { return "curation-orchestrator" }

// Caste returns the agent's role category.
func (o *Orchestrator) Caste() agent.Caste { return agent.CasteCurator }

// Triggers returns the event patterns that activate the orchestrator.
func (o *Orchestrator) Triggers() []agent.Trigger {
	return []agent.Trigger{
		{Topic: "consolidation.*"},
		{Topic: "phase.end"},
	}
}

// Execute runs a curation cycle triggered by an event.
func (o *Orchestrator) Execute(ctx context.Context, event events.Event) error {
	_, err := o.Run(ctx, false)
	return err
}

// Subscribe registers the orchestrator for consolidation events.
func (o *Orchestrator) Subscribe(bus *events.Bus) error {
	ch, err := bus.Subscribe("consolidation.*")
	if err != nil {
		return fmt.Errorf("curation: subscribe: %w", err)
	}
	o.eventCh = ch
	return nil
}

// Run executes all 8 curation ants sequentially. If the sentinel step
// fails (corrupt stores detected), the remaining steps are marked as
// skipped and the loop breaks early.
func (o *Orchestrator) Run(ctx context.Context, dryRun bool) (*CurationResult, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	start := time.Now()

	var result CurationResult
	result.DryRun = dryRun
	result.Steps = make([]StepResult, 0, 8)

	for i, step := range o.steps {
		// Check context cancellation
		select {
		case <-ctx.Done():
			// Mark remaining as skipped
			for _, remaining := range o.steps[i:] {
				result.Steps = append(result.Steps, StepResult{
					Name:    remaining.name,
					Success: false,
					Summary: map[string]any{"reason": "context cancelled"},
				})
				result.Skipped++
			}
			result.DurationMs = time.Since(start).Milliseconds()
			return &result, ctx.Err()
		default:
		}

		sr, err := step.ant.Run(ctx, dryRun)
		if err != nil {
			sr.Error = err
			sr.Success = false
			result.Steps = append(result.Steps, sr)
			result.Failed++

			// Sentinel abort: skip remaining steps
			if step.name == "sentinel" {
				for _, remaining := range o.steps[i+1:] {
					result.Steps = append(result.Steps, StepResult{
						Name:    remaining.name,
						Success: false,
						Summary: map[string]any{"reason": "skipped: sentinel detected corrupt stores"},
					})
					result.Skipped++
				}
				result.DurationMs = time.Since(start).Milliseconds()
				return &result, fmt.Errorf("curation: sentinel abort: %w", err)
			}
			continue
		}

		result.Steps = append(result.Steps, sr)
		result.Succeeded++
	}

	result.DurationMs = time.Since(start).Milliseconds()
	return &result, nil
}
