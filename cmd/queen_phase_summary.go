package cmd

import (
	"fmt"
	"strings"
)

// renderActionsNeeded produces an "Actions Needed" section listing items requiring
// human attention. Per D-10: derived from wave-summary escalated count, recovery-log
// entries with action=escalate, and unrecovered failures.
// Per D-10: returns empty string when zero items need attention (clean build = no noise).
func renderActionsNeeded(summary WaveLifecycleSummary, recoveryLog RecoveryLogFile) string {
	var items []string

	// From wave summary: escalated workers per D-10
	if summary.TotalEscalated > 0 {
		for _, wave := range summary.Waves {
			if wave.Escalated > 0 {
				items = append(items, fmt.Sprintf("Wave %d: %d worker(s) escalated after recovery budget exhausted", wave.Wave, wave.Escalated))
			}
		}
	}

	// From recovery log: entries with action=escalate per D-10
	for _, entry := range recoveryLog.Entries {
		if entry.ActionTaken == "escalate" {
			items = append(items, fmt.Sprintf("  - %s: %s (%s)", entry.Failure.WorkerName, entry.Outcome, entry.Failure.ErrorMessage))
		}
	}

	// Per D-10: omit section entirely when zero items
	if len(items) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(renderStageMarker("Actions Needed"))
	for _, item := range items {
		b.WriteString(item)
		b.WriteString("\n")
	}
	return b.String()
}

// renderPhaseEndSummary renders the phase-end summary to stdout.
// Per D-09: existing wave summary table (already rendered by renderWaveSummaryTable)
// plus actions-needed section.
// Per D-11: uses existing output channels (stdout).
func renderPhaseEndSummary(summary WaveLifecycleSummary, phaseNum int) {
	if !shouldRenderVisualOutput(stdout) || store == nil {
		return
	}
	// Read recovery log for actions-needed
	recoveryLog, _ := recoveryLogReadPhase(phaseNum)

	actionsNeeded := renderActionsNeeded(summary, recoveryLog)
	if actionsNeeded != "" {
		fmt.Fprint(stdout, actionsNeeded)
	}
}
