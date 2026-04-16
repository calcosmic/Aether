package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/cache"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

type signalHousekeepingResult struct {
	TotalSignals           int `json:"total_signals"`
	ActiveBefore           int `json:"active_before"`
	ActiveAfter            int `json:"active_after"`
	ExpiredByTime          int `json:"expired_by_time"`
	DeactivatedByStrength  int `json:"deactivated_by_strength"`
	ExpiredWorkerContinue  int `json:"expired_worker_continue"`
	Updated                int `json:"updated"`
	DryRun                 bool `json:"dry_run"`
}

var signalHousekeepingCmd = &cobra.Command{
	Use:   "signal-housekeeping",
	Short: "Expire stale and low-value pheromone signals",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		result, err := runSignalHousekeeping(time.Now().UTC(), dryRun)
		if err != nil {
			outputError(2, fmt.Sprintf("signal housekeeping failed: %v", err), nil)
			return nil
		}

		outputOK(result)
		return nil
	},
}

func runSignalHousekeeping(now time.Time, dryRun bool) (signalHousekeepingResult, error) {
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		state = colony.ColonyState{}
	}

	var pf colony.PheromoneFile
	if err := store.LoadJSON("pheromones.json", &pf); err != nil {
		return signalHousekeepingResult{DryRun: dryRun}, nil
	}
	if pf.Signals == nil {
		pf.Signals = []colony.PheromoneSignal{}
	}

	result := applySignalHousekeeping(&pf, &state, now, dryRun)
	if dryRun || result.Updated == 0 {
		return result, nil
	}

	if err := store.SaveJSON("pheromones.json", pf); err != nil {
		return signalHousekeepingResult{}, err
	}
	_, _ = cache.NewSessionCache(store.BasePath()).Clear()
	return result, nil
}

func applySignalHousekeeping(pf *colony.PheromoneFile, state *colony.ColonyState, now time.Time, dryRun bool) signalHousekeepingResult {
	result := signalHousekeepingResult{
		TotalSignals: len(pf.Signals),
		DryRun:       dryRun,
	}

	nowRFC3339 := now.Format(time.RFC3339)

	for i := range pf.Signals {
		sig := &pf.Signals[i]
		if !sig.Active {
			continue
		}
		result.ActiveBefore++

		switch {
		case signalExpiredByTime(*sig, now):
			result.ExpiredByTime++
			result.Updated++
			if !dryRun {
				deactivateSignal(sig, nowRFC3339)
			}
		case computeEffectiveStrength(*sig, now) < 0.1:
			result.DeactivatedByStrength++
			result.Updated++
			if !dryRun {
				deactivateSignal(sig, nowRFC3339)
			}
		case sig.Source == "worker:continue" && (phaseCompletionsSince(state, sig.CreatedAt) >= 3 || signalPredatesCurrentColony(state, sig.CreatedAt)):
			result.ExpiredWorkerContinue++
			result.Updated++
			if !dryRun {
				deactivateSignal(sig, nowRFC3339)
			}
		default:
			result.ActiveAfter++
		}
	}

	if !dryRun {
		result.ActiveAfter = countActiveSignals(pf.Signals)
	}
	return result
}

func signalExpiredByTime(sig colony.PheromoneSignal, now time.Time) bool {
	if sig.ExpiresAt == nil || *sig.ExpiresAt == "" {
		return false
	}
	expiresAt, err := time.Parse(time.RFC3339, *sig.ExpiresAt)
	if err != nil {
		return false
	}
	return !expiresAt.After(now)
}

func deactivateSignal(sig *colony.PheromoneSignal, nowRFC3339 string) {
	sig.Active = false
	sig.ArchivedAt = &nowRFC3339
	sig.ExpiresAt = &nowRFC3339
}

func phaseCompletionsSince(state *colony.ColonyState, createdAt string) int {
	if state == nil || len(state.Events) == 0 {
		return 0
	}
	created, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return 0
	}

	count := 0
	for _, event := range state.Events {
		parts := strings.SplitN(event, "|", 4)
		if len(parts) < 2 {
			continue
		}
		ts, err := time.Parse(time.RFC3339, parts[0])
		if err != nil || !ts.After(created) {
			continue
		}
		switch parts[1] {
		case "phase_advanced", "phase_completed":
			count++
		}
	}
	return count
}

func signalPredatesCurrentColony(state *colony.ColonyState, createdAt string) bool {
	if state == nil || state.InitializedAt == nil {
		return false
	}
	created, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return false
	}
	return created.Before(*state.InitializedAt)
}

func countActiveSignals(signals []colony.PheromoneSignal) int {
	count := 0
	for _, sig := range signals {
		if sig.Active {
			count++
		}
	}
	return count
}

func init() {
	signalHousekeepingCmd.Flags().Bool("dry-run", false, "Report housekeeping results without modifying pheromones.json")
	rootCmd.AddCommand(signalHousekeepingCmd)
}
