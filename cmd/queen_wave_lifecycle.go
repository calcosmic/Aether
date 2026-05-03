package cmd

import (
	"context"
	"fmt"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
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
// Per D-10: queen owns wave grouping.
// Per D-01/D-02: always advance -- never stop between waves.
func queenWaveLifecycle(
	ctx context.Context,
	dispatches []codex.WorkerDispatch,
	dispatchFn WaveDispatchFunc,
	phase colony.Phase,
	cb *CircuitBreaker,
	phaseNum int,
) (WaveLifecycleSummary, []codex.DispatchResult, error) {
	return WaveLifecycleSummary{}, nil, nil
}

// writeWaveSummary persists a wave lifecycle summary to wave-summary-{N}.json.
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
func renderWaveSummaryTable(summary WaveLifecycleSummary) {
	_ = summary
	_ = table.NewWriter()
}
