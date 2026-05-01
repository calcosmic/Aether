package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

var unblockCmd = &cobra.Command{
	Use:   "unblock",
	Short: "Show gate failure summary and recovery options for the current phase",
	Long: `Reads gate-results-{N}.json for the current phase and renders a Gate Recovery Summary
showing which gates failed, why, and how to fix them. Provides two recovery options:
(1) fix manually and run /ant-continue, or (2) view specific fix hints for each failed gate.`,
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		phaseNum, _ := cmd.Flags().GetInt("phase")
		if phaseNum == 0 {
			// Read current phase from COLONY_STATE
			var state colony.ColonyState
			if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
				outputErrorMessage("no colony state found")
				return nil
			}
			phaseNum = state.CurrentPhase
		}
		if phaseNum == 0 {
			outputErrorMessage("no active phase -- specify --phase N")
			return nil
		}

		results, err := gateResultsReadPhase(phaseNum)
		if err != nil || len(results) == 0 {
			outputOK(fmt.Sprintf("No gate results found for phase %d. Run /ant-continue to run gates.", phaseNum))
			return nil
		}

		// Build recovery summary
		summary := buildGateRecoverySummary(phaseNum, results)
		if shouldRenderVisualOutput(stderr) {
			fmt.Fprint(stderr, summary)
		} else {
			data, _ := json.Marshal(map[string]interface{}{
				"ok":      true,
				"phase":   phaseNum,
				"summary": summary,
				"results": results,
			})
			fmt.Fprintln(stdout, string(data))
		}
		return nil
	},
}

// buildGateRecoverySummary renders a human-readable Gate Recovery Summary
// showing failed gates with fix hints and recovery options.
func buildGateRecoverySummary(phaseNum int, results []GateCheckResult) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Gate Recovery Summary -- Phase %d\n", phaseNum))
	b.WriteString(strings.Repeat("-", 40))
	b.WriteString("\n\n")

	failedCount := 0
	passedCount := 0
	for _, r := range results {
		if r.Status == "failed" {
			failedCount++
		} else if r.Status == "passed" || r.Status == "skipped" {
			passedCount++
		}
	}

	b.WriteString(fmt.Sprintf("Passed: %d | Failed: %d\n\n", passedCount, failedCount))

	if failedCount == 0 {
		b.WriteString("All gates passed. Run /ant-continue to proceed.\n")
		return b.String()
	}

	b.WriteString("Failed Gates:\n")
	for _, r := range results {
		if r.Status != "failed" {
			continue
		}
		b.WriteString(fmt.Sprintf("\n  Gate: %s\n", r.Name))
		if r.Detail != "" {
			b.WriteString(fmt.Sprintf("  Issue: %s\n", r.Detail))
		}
		if r.FixHint != "" {
			b.WriteString(fmt.Sprintf("  Fix: %s\n", r.FixHint))
		}
		if len(r.RecoveryOptions) > 0 {
			b.WriteString("  Options:\n")
			for i, opt := range r.RecoveryOptions {
				b.WriteString(fmt.Sprintf("    %d. %s\n", i+1, opt))
			}
		}
	}

	b.WriteString("\nRecovery Options:\n")
	b.WriteString("  1. Fix the issues above manually, then run /ant-continue\n")
	b.WriteString("  2. View detailed fix hints for each gate above\n")

	return b.String()
}

func init() {
	unblockCmd.Flags().Int("phase", 0, "Phase number (default: current phase)")
	rootCmd.AddCommand(unblockCmd)
}
