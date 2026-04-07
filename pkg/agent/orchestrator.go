package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/storage"
	"golang.org/x/sync/errgroup"
)

// maxOrchestratorGoroutines limits concurrent task dispatch.
const maxOrchestratorGoroutines = 4

// taskTimeout is the per-task execution timeout.
const taskTimeout = 5 * time.Minute

// OrchestrationResult holds the aggregate outcome of orchestrating a phase.
type OrchestrationResult struct {
	PhaseID    int           `json:"phase_id"`
	Tasks      []TaskResult  `json:"tasks"`
	Succeeded  int           `json:"succeeded"`
	Failed     int           `json:"failed"`
	DurationMs int64         `json:"duration_ms"`
	Validated  bool          `json:"validated"`
}

// PhaseOrchestrator decomposes a phase into tasks using BuildTaskGraph,
// dispatches them concurrently via errgroup with bounded goroutine limits,
// collects results, and validates against success criteria.
type PhaseOrchestrator struct {
	registry *Registry
	bus      *events.Bus
	store    *storage.Store
	mu       sync.Mutex
	results  map[string]*TaskResult
}

// NewPhaseOrchestrator creates a new orchestrator with the given dependencies.
func NewPhaseOrchestrator(registry *Registry, bus *events.Bus, store *storage.Store) *PhaseOrchestrator {
	return &PhaseOrchestrator{
		registry: registry,
		bus:      bus,
		store:    store,
	}
}

// Run orchestrates a full phase: builds the task graph, dispatches tasks
// concurrently respecting dependency order, collects results, and validates.
func (o *PhaseOrchestrator) Run(ctx context.Context, phase colony.Phase) (*OrchestrationResult, error) {
	start := time.Now()

	graph, err := BuildTaskGraph(phase.Tasks)
	if err != nil {
		return nil, fmt.Errorf("orchestrator: build task graph: %w", err)
	}

	o.results = make(map[string]*TaskResult)

	totalTasks := len(graph.Nodes())
	processed := 0

	for processed < totalTasks {
		ready := graph.Ready()
		if len(ready) == 0 {
			// No ready tasks but tasks remain means a cycle (should have been
			// caught by BuildTaskGraph, but guard against runtime issues).
			return nil, fmt.Errorf("orchestrator: no ready tasks but %d remain (possible cycle)", totalTasks-processed)
		}

		g, gCtx := errgroup.WithContext(ctx)
		g.SetLimit(maxOrchestratorGoroutines)

		for i := range ready {
			task := ready[i]
			g.Go(func() error {
				return o.dispatchTask(gCtx, task, graph)
			})
		}

		_ = g.Wait()

		// Mark successful tasks as completed in the graph
		for _, task := range ready {
			if r, ok := o.results[task.ID]; ok && r.Success {
				graph.Complete(task.ID)
			}
			processed++
		}
	}

	duration := time.Since(start)

	// Validate collected results against success criteria.
	result := o.buildResult(phase.ID, duration)
	result.Validated = o.validateOutput(phase, o.results)

	// Persist orchestrator progress to COLONY_STATE.json.
	status := "completed"
	if result.Failed > 0 {
		status = "failed"
	}
	_ = o.updateState(phase.ID, status)

	return result, nil
}

// dispatchTask dispatches a single task to an agent via the registry.
// It creates a scoped context with timeout, builds a scoped event containing
// only the assigned task's data (no sibling tasks), and records the result.
// Individual task failures do not abort sibling tasks.
func (o *PhaseOrchestrator) dispatchTask(ctx context.Context, task *TaskNode, graph *TaskGraph) error {
	start := time.Now()

	scopedCtx, cancel := context.WithTimeout(ctx, taskTimeout)
	defer cancel()

	topic := fmt.Sprintf("task.%s", task.Caste)
	agents := o.registry.Match(topic)

	if len(agents) == 0 {
		o.recordResult(task, agents, false, "", fmt.Errorf("no agent for task %s (caste: %s)", task.ID, task.Caste), time.Since(start))
		return nil
	}

	// Build scoped event payload: only task_id, goal, criteria, type_hint.
	// No sibling tasks or full phase plan (per D-08 agent isolation).
	payload := map[string]interface{}{
		"task_id":   task.ID,
		"goal":      task.Goal,
		"criteria":  task.Criteria,
		"type_hint": task.TypeHint,
	}
	payloadBytes, _ := json.Marshal(payload)

	scopedEvent := events.Event{
		ID:        fmt.Sprintf("task_%s_%d", task.ID, start.Unix()),
		Topic:     topic,
		Payload:   payloadBytes,
		Source:    "phase-orchestrator",
		Timestamp: events.FormatTimestamp(start),
	}

	err := agents[0].Execute(scopedCtx, scopedEvent)
	elapsed := time.Since(start)

	if err != nil {
		o.recordResult(task, agents, false, "", err, elapsed)
	} else {
		o.recordResult(task, agents, true, "", nil, elapsed)
	}

	return nil
}

// recordResult stores a task result under mutex protection.
func (o *PhaseOrchestrator) recordResult(task *TaskNode, agents []Agent, success bool, output string, taskErr error, duration time.Duration) {
	o.mu.Lock()
	defer o.mu.Unlock()

	r := &TaskResult{
		TaskID:    task.ID,
		Caste:     task.Caste,
		Success:   success,
		Output:    output,
		Duration:  duration.Milliseconds(),
	}

	if len(agents) > 0 {
		r.AgentName = agents[0].Name()
	}

	if taskErr != nil {
		r.Error = taskErr.Error()
	}

	o.results[task.ID] = r
}

// buildResult constructs the final OrchestrationResult from collected results.
func (o *PhaseOrchestrator) buildResult(phaseID int, duration time.Duration) *OrchestrationResult {
	result := &OrchestrationResult{
		PhaseID:    phaseID,
		Tasks:      make([]TaskResult, 0, len(o.results)),
		DurationMs: duration.Milliseconds(),
	}

	for _, r := range o.results {
		result.Tasks = append(result.Tasks, *r)
		if r.Success {
			result.Succeeded++
		} else {
			result.Failed++
		}
	}

	return result
}

// validateOutput checks all task results succeeded (structural validation).
// Full criteria evaluation is deferred to the autopilot layer; this confirms
// results exist and all reported success.
func (o *PhaseOrchestrator) validateOutput(phase colony.Phase, results map[string]*TaskResult) bool {
	for _, t := range phase.Tasks {
		id := taskID(t)
		r, ok := results[id]
		if !ok || !r.Success {
			return false
		}
	}
	return true
}

// updateState writes orchestrator progress to COLONY_STATE.json.
func (o *PhaseOrchestrator) updateState(phaseID int, status string) error {
	if o.store == nil {
		return fmt.Errorf("orchestrator: no store configured")
	}

	var state colony.ColonyState
	if err := o.store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		return fmt.Errorf("orchestrator: load state: %w", err)
	}

	succeeded := 0
	failed := 0
	for _, r := range o.results {
		if r.Success {
			succeeded++
		} else {
			failed++
		}
	}

	now := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	state.OrchestratorState = &colony.OrchestratorState{
		Phase:     phaseID,
		Status:    status,
		TaskCount: len(o.results),
		Completed: succeeded,
		Failed:    failed,
		UpdatedAt: now,
	}

	if err := o.store.SaveJSON("COLONY_STATE.json", state); err != nil {
		return fmt.Errorf("orchestrator: save state: %w", err)
	}

	return nil
}
