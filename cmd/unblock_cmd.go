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
showing which gates failed, why, and how to fix them. Provides three recovery options:
(1) fix manually and run /ant-continue, (2) view specific fix hints for each failed gate,
or (3) dispatch the Fixer agent to investigate and apply fixes.`,
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

		// An open blocker flag stops work just as surely as a failed check.
		// Autopilot sends the owner here for both, so answering "no check
		// results" while a blocker sat open was a dead end (Phase 210
		// blocker 14).
		var openBlockers []colony.FlagEntry
		if flags, ok := loadFlagsFile(store); ok {
			openBlockers = classifyOpenFlags(flags.Decisions).Blockers
		}

		results, err := gateResultsReadPhase(phaseNum)
		hasGateResults := err == nil && len(results) > 0
		dispatch, _ := cmd.Flags().GetBool("dispatch")
		if !hasGateResults && (dispatch || len(openBlockers) == 0) {
			outputOK(fmt.Sprintf("No gate results found for phase %d. Run /ant-continue to run gates.", phaseNum))
			return nil
		}

		if dispatch {
			fixerMode, _ := cmd.Flags().GetString("fixer-mode")
			if err := dispatchFixer(phaseNum, fixerMode); err != nil {
				outputError(1, fmt.Sprintf("Fixer dispatch failed: %s", err.Error()), nil)
				return nil
			}
			return nil
		}

		// Build recovery summary
		summary := buildOpenBlockerSummary(openBlockers)
		if hasGateResults {
			if summary != "" {
				summary += "\n"
			}
			summary += buildGateRecoverySummary(phaseNum, results)
		}
		if shouldRenderVisualOutput(stderr) {
			writeVisualOutput(stderr, summary)
		} else {
			data, _ := json.Marshal(map[string]interface{}{
				"ok":       true,
				"phase":    phaseNum,
				"summary":  summary,
				"results":  results,
				"blockers": openBlockers,
			})
			fmt.Fprintln(stdout, string(data))
		}
		return nil
	},
}

// unblockHandoverSentence marks a blocker one of Aether's own background
// helpers raised because its locked-down workspace stopped it doing a step.
const unblockHandoverSentence = "A helper working in Aether's own locked-down workspace could not do this itself. It can be done here in the chat once you say yes."

// blockerFromLockedDownHelper reports whether one of Aether's own background
// helpers -- the ones Autopilot and the direct build dispatch into a
// locked-down workspace -- raised this blocker. It is recognised from the
// build record the blocker names, never from the helper's own wording.
func blockerFromLockedDownHelper(flag colony.FlagEntry) bool {
	attemptID := strings.TrimSpace(flag.AttemptID)
	if flag.Source != "escalation" || attemptID == "" || flag.Phase == nil {
		return false
	}
	for _, record := range listBuildAttemptsForPhase(*flag.Phase) {
		if record.ID == attemptID {
			return record.ExecutionOwner == buildExecutionOwner("real", false)
		}
	}
	return false
}

// buildOpenBlockerSummary lists every open blocker in its own words -- a
// helper's blocker already says what needs doing -- with the exact command
// that marks it resolved once it is fixed. It returns "" when none is open.
func buildOpenBlockerSummary(blockers []colony.FlagEntry) string {
	if len(blockers) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Blocking work -- %d problem(s) stop progress until resolved\n", len(blockers)))
	b.WriteString(strings.Repeat("-", 40))
	b.WriteString("\n")
	for i, flag := range blockers {
		b.WriteString(fmt.Sprintf("\n  %d. %s\n", i+1, strings.Join(strings.Fields(flag.Description), " ")))
		if blockerFromLockedDownHelper(flag) {
			b.WriteString("     " + unblockHandoverSentence + "\n")
		}
		b.WriteString(fmt.Sprintf("     Once it is fixed, mark it resolved: aether flag-resolve --id %s --message \"<what was done>\"\n", flag.ID))
	}
	b.WriteString("\nWhen nothing is left blocking, run /ant-continue to check the phase.\n")
	return b.String()
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
			sanitized, _ := colony.SanitizeSignalContent(r.Detail)
			b.WriteString(fmt.Sprintf("  Issue: %s\n", sanitized))
		}
		if r.FixHint != "" {
			sanitized, _ := colony.SanitizeSignalContent(r.FixHint)
			b.WriteString(fmt.Sprintf("  Fix: %s\n", sanitized))
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
	b.WriteString("  3. Run /ant-unblock --dispatch to dispatch Fixer (propose mode by default)\n")

	return b.String()
}

func init() {
	unblockCmd.Flags().Int("phase", 0, "Phase number (default: current phase)")
	unblockCmd.Flags().String("fixer-mode", "propose", "Fixer autonomy mode: full, propose, or advise (default: propose)")
	unblockCmd.Flags().Bool("dispatch", false, "Dispatch the Fixer agent to investigate and fix failed gates")
	rootCmd.AddCommand(unblockCmd)
}
