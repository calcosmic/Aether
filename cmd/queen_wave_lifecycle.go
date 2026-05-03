package cmd

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
	"github.com/jedib0t/go-pretty/v6/table"
)

// WaveDispatchFunc dispatches a single wave's workers and returns results.
// Injected for testability; production uses dispatchCodexBuildWorkersInRepo.
type WaveDispatchFunc func(ctx context.Context, dispatches []codex.WorkerDispatch, wave int) ([]codex.DispatchResult, error)

// WaveLifecycleSummary captures the complete wave lifecycle result.
// Per D-06/D-07: produced after all waves complete, written to JSON for Phase 99.
type WaveLifecycleSummary struct {
	Phase           int          `json:"phase"`
	TotalWaves      int          `json:"total_waves"`
	TotalDispatched int          `json:"total_dispatched"`
	TotalSucceeded  int          `json:"total_succeeded"`
	TotalFailed     int          `json:"total_failed"`
	TotalRecovered  int          `json:"total_recovered"`
	TotalEscalated  int          `json:"total_escalated"`
	Waves           []WaveResult `json:"waves"`
	CompletedAt     string       `json:"completed_at"`
}

// WaveResult captures per-wave outcome data.
type WaveResult struct {
	Wave       int             `json:"wave"`
	Dispatched int             `json:"dispatched"`
	Succeeded  int             `json:"succeeded"`
	Failed     int             `json:"failed"`
	Recovered  []RecoveryEntry `json:"recovered,omitempty"`
	Escalated  int             `json:"escalated"`
	BudgetUsed int             `json:"budget_used,omitempty"`
}

// RecoveryEntry records a single recovery action within a wave.
type RecoveryEntry struct {
	WorkerName string `json:"worker_name"`
	Method     string `json:"method"`
	Detail     string `json:"detail,omitempty"`
}

// queenWaveLifecycle runs the full build wave loop.
// Per D-09: single-invocation, not goroutine/daemon.
// Per D-10: queen owns wave grouping via codex.GroupByWave.
// Per D-01/D-02: always advance -- never stop between waves regardless of failures.
// Per D-11: dispatch goes through the injected WaveDispatchFunc (not direct platform calls).
// Per D-03: unrecovered failures logged to existing recovery-log files.
// Per D-04/D-05: ceremony event emitted between waves.
// Per D-06/D-07/D-08: wave summary table rendered to stdout, JSON persisted.
func queenWaveLifecycle(
	ctx context.Context,
	dispatches []codex.WorkerDispatch,
	dispatchFn WaveDispatchFunc,
	phase colony.Phase,
	cb *CircuitBreaker,
	phaseNum int,
) (WaveLifecycleSummary, []codex.DispatchResult, error) {
	// Handle empty dispatches early
	if len(dispatches) == 0 {
		return WaveLifecycleSummary{Phase: phaseNum}, nil, nil
	}

	// Group dispatches by wave and sort wave numbers
	waveGroups := codex.GroupByWave(dispatches)
	waveNumbers := make([]int, 0, len(waveGroups))
	for wave := range waveGroups {
		waveNumbers = append(waveNumbers, wave)
	}
	sort.Ints(waveNumbers)

	// Create recovery budget, reset per wave
	budget := newRecoveryBudget(waveNumbers[0])

	var allResults []codex.DispatchResult
	var waves []WaveResult
	totalWaves := len(waveNumbers)

	for i, waveNum := range waveNumbers {
		waveDispatches := waveGroups[waveNum]

		// Reset budget for this wave (D-10)
		budget.resetForWave(waveNum)

		// Dispatch this wave's workers
		waveResults, dispatchErr := dispatchFn(ctx, waveDispatches, waveNum)
		if dispatchErr != nil {
			return WaveLifecycleSummary{}, allResults, fmt.Errorf("wave %d dispatch failed: %w", waveNum, dispatchErr)
		}

		allResults = append(allResults, waveResults...)

		// Count successes and failures
		succeeded := 0
		failed := 0
		for _, result := range waveResults {
			if result.Status == "completed" {
				succeeded++
			} else {
				failed++
			}
		}

		// Process recovery for failed workers between waves
		var waveRecovered []RecoveryEntry
		waveEscalated := 0
		for _, result := range waveResults {
			if result.Status == "completed" {
				continue
			}

			// Find the original dispatch for context
			var dispatch *codex.WorkerDispatch
			for j := range waveDispatches {
				if waveDispatches[j].WorkerName == result.WorkerName {
					dispatch = &waveDispatches[j]
					break
				}
			}

			errorMsg := ""
			if result.Error != nil {
				errorMsg = result.Error.Error()
			}
			if dispatch == nil {
				dispatch = &codex.WorkerDispatch{}
			}

			// Build recovery context and call orchestrator
			recoveryCtx := RecoveryContext{
				Phase:          phaseNum,
				Wave:           waveNum,
				WorkerName:     result.WorkerName,
				TaskID:         dispatch.TaskID,
				Caste:          dispatch.Caste,
				Status:         result.Status,
				ErrorMessage:   errorMsg,
				Dispatches:     waveDispatches,
				CircuitBreaker: cb,
				Budget:         budget,
			}

			outcome := orchestrateRecovery(recoveryCtx)

			// Log recovery entries to recovery-log (D-03)
			if len(outcome.LogEntries) > 0 {
				existingLog, _ := recoveryLogReadPhase(phaseNum)
				existingLog.Entries = append(existingLog.Entries, outcome.LogEntries...)
				existingLog.Phase = phaseNum
				_ = recoveryLogWritePhase(phaseNum, existingLog.Entries)
			}

			// Track recovery action in wave summary
			if outcome.Action.Type == "escalate" {
				waveEscalated++
			} else {
				waveRecovered = append(waveRecovered, RecoveryEntry{
					WorkerName: outcome.Action.WorkerName,
					Method:     outcome.Action.Type,
					Detail:     outcome.Action.Detail,
				})
			}
		}

		// Build wave result
		waveResult := WaveResult{
			Wave:       waveNum,
			Dispatched: len(waveDispatches),
			Succeeded:  succeeded,
			Failed:     failed,
			Recovered:  waveRecovered,
			Escalated:  waveEscalated,
			BudgetUsed: budget.totalUsed(),
		}
		waves = append(waves, waveResult)

		// Emit ceremony event between waves (D-04/D-05)
		emitBuildCeremony(events.CeremonyTopicBuildWaveEnd, events.CeremonyPayload{
			Phase:     phaseNum,
			PhaseName: phase.Name,
			Wave:      waveNum,
			Status:    "completed",
			Completed: succeeded,
			Total:     len(waveDispatches),
			Message: fmt.Sprintf("wave %d/%d complete -- %d succeeded, %d recovered, %d escalated",
				i+1, totalWaves, succeeded, len(waveRecovered), waveEscalated),
		})

		// Per D-01/D-02: always advance -- no stop condition
	}

	// Build aggregate summary
	summary := WaveLifecycleSummary{
		Phase:          phaseNum,
		TotalWaves:     totalWaves,
		TotalDispatched: 0,
		TotalSucceeded: 0,
		TotalFailed:    0,
		TotalRecovered: 0,
		TotalEscalated: 0,
		Waves:          waves,
		CompletedAt:    time.Now().UTC().Format(time.RFC3339),
	}

	for _, w := range waves {
		summary.TotalDispatched += w.Dispatched
		summary.TotalSucceeded += w.Succeeded
		summary.TotalFailed += w.Failed
		summary.TotalRecovered += len(w.Recovered)
		summary.TotalEscalated += w.Escalated
	}

	// Render wave summary table to stdout (D-06)
	renderWaveSummaryTable(summary)

	return summary, allResults, nil
}

// writeWaveSummary persists a wave lifecycle summary to wave-summary-{N}.json.
// Per D-07/D-08: uses the existing per-phase persistence pattern.
func writeWaveSummary(phaseNum int, summary WaveLifecycleSummary) error {
	rel := fmt.Sprintf("wave-summary-%d.json", phaseNum)
	return store.SaveJSON(rel, summary)
}

// readWaveSummary reads a wave lifecycle summary from wave-summary-{N}.json.
func readWaveSummary(phaseNum int) (WaveLifecycleSummary, error) {
	var summary WaveLifecycleSummary
	rel := fmt.Sprintf("wave-summary-%d.json", phaseNum)
	err := store.LoadJSON(rel, &summary)
	return summary, err
}

// renderWaveSummaryTable renders the wave summary as a go-pretty table to stdout.
// Per D-06: wave-by-wave rows with totals, follows renderFailureClassifyTable pattern.
func renderWaveSummaryTable(summary WaveLifecycleSummary) {
	t := table.NewWriter()
	t.AppendHeader(table.Row{"Wave", "Dispatched", "Succeeded", "Failed", "Recovered", "Escalated"})
	for _, w := range summary.Waves {
		t.AppendRow(table.Row{w.Wave, w.Dispatched, w.Succeeded, w.Failed, len(w.Recovered), w.Escalated})
	}
	t.AppendSeparator()
	t.AppendRow(table.Row{"Total", summary.TotalDispatched, summary.TotalSucceeded, summary.TotalFailed, summary.TotalRecovered, summary.TotalEscalated})
	fmt.Fprintln(stdout, t.Render())
}
