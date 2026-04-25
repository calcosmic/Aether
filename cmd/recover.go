package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

// recoverCmd is the cobra command for rescuing a stuck colony.
var recoverCmd = &cobra.Command{
	Use:   "recover",
	Short: "Rescue a stuck colony",
	Long: `Scan the colony for stuck-state conditions and diagnose why it cannot make progress.
Read-only by default; use --apply to attempt automatic fixes.`,
	Args: cobra.NoArgs,
	RunE: runRecover,
}

func init() {
	rootCmd.AddCommand(recoverCmd)
	recoverCmd.Flags().Bool("apply", false, "apply fixes for detected issues")
	recoverCmd.Flags().Bool("force", false, "allow destructive repairs")
	recoverCmd.Flags().Bool("json", false, "output structured JSON")
}

func runRecover(cmd *cobra.Command, args []string) error {
	state, err := loadActiveColonyState()
	if err != nil {
		if shouldRenderVisualOutput(stdout) && strings.Contains(colonyStateLoadMessage(err), "No colony initialized") {
			fmt.Fprint(stdout, renderNoColonyRecoverVisual())
			return nil
		}
		fmt.Fprintln(stdout, colonyStateLoadMessage(err))
		return nil
	}

	dataDir := filepath.Join(resolveAetherRoot(), ".aether", "data")

	issues, scanErr := performStuckStateScan(dataDir)
	if scanErr != nil {
		fmt.Fprintf(stdout, "Scan failed: %v\n", scanErr)
		return nil
	}

	apply, _ := cmd.Flags().GetBool("apply")
	force, _ := cmd.Flags().GetBool("force")
	jsonOut, _ := cmd.Flags().GetBool("json")

	// Repair mode is handled in Plan 02 (recover_visuals.go).
	// For now, render scan results and exit.
	_ = apply
	_ = force

	if jsonOut {
		fmt.Fprint(stdout, renderRecoverJSON(issues, &state))
		return nil
	}

	output := renderRecoverReport(issues, &state)
	fmt.Fprint(stdout, output)
	return nil
}

// renderNoColonyRecoverVisual renders the visual when no colony is initialized.
func renderNoColonyRecoverVisual() string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("recover"), "Colony Recovery"))
	b.WriteString(visualDivider)
	b.WriteString("No colony initialized in this repo.\n")
	b.WriteString(renderNextUp(
		`Run `+"`aether init \"goal\"`"+` to start a colony.`,
		`Run `+"`aether lay-eggs`"+` first if this repo has not been set up for Aether yet.`,
	))
	return b.String()
}

// renderRecoverReport renders the human-readable diagnosis report.
func renderRecoverReport(issues []HealthIssue, state *colony.ColonyState) string {
	var b strings.Builder

	b.WriteString(renderBanner(commandEmoji("recover"), "Colony Recovery"))
	b.WriteString(visualDivider)

	// Summary counts
	criticalCount := 0
	warningCount := 0
	infoCount := 0
	fixableCount := 0
	for _, issue := range issues {
		switch issue.Severity {
		case "critical":
			criticalCount++
		case "warning":
			warningCount++
		case "info":
			infoCount++
		}
		if issue.Fixable {
			fixableCount++
		}
	}

	b.WriteString(renderStageMarker("Diagnosis"))
	if state != nil && state.Goal != nil {
		b.WriteString("Goal: ")
		b.WriteString(*state.Goal)
		b.WriteString("\n")
	}
	b.WriteString(fmt.Sprintf("Issues: %d critical, %d warnings, %d info\n", criticalCount, warningCount, infoCount))
	b.WriteString("\n")

	// Critical Issues
	if criticalCount > 0 {
		b.WriteString(renderStageMarker("Critical Issues"))
		for _, issue := range issues {
			if issue.Severity != "critical" {
				continue
			}
			writeIssueLine(&b, issue)
		}
		b.WriteString("\n")
	}

	// Warnings
	if warningCount > 0 {
		b.WriteString(renderStageMarker("Warnings"))
		for _, issue := range issues {
			if issue.Severity != "warning" {
				continue
			}
			writeIssueLine(&b, issue)
		}
		b.WriteString("\n")
	}

	// Info
	if infoCount > 0 {
		b.WriteString(renderStageMarker("Info"))
		for _, issue := range issues {
			if issue.Severity != "info" {
				continue
			}
			writeIssueLine(&b, issue)
		}
		b.WriteString("\n")
	}

	if len(issues) == 0 {
		b.WriteString("Colony is healthy. No stuck-state conditions detected.\n\n")
	}

	// Next Steps
	b.WriteString(renderStageMarker("Next Steps"))
	switch {
	case criticalCount > 0 && fixableCount > 0:
		b.WriteString(fmt.Sprintf("Run `aether recover --apply` to fix %d auto-fixable issue(s).\n", fixableCount))
	case criticalCount > 0:
		b.WriteString("Critical issues detected. Review the diagnosis above.\n")
	case warningCount > 0 && fixableCount > 0:
		b.WriteString(fmt.Sprintf("Run `aether recover --apply` to fix %d auto-fixable issue(s).\n", fixableCount))
	case warningCount > 0:
		b.WriteString("Review warnings above. Some issues may need manual intervention.\n")
	default:
		b.WriteString("Colony is healthy. No action needed.\n")
	}

	return b.String()
}

// renderRecoverJSON renders the JSON diagnosis report.
func renderRecoverJSON(issues []HealthIssue, state *colony.ColonyState) string {
	goal := ""
	if state != nil && state.Goal != nil {
		goal = *state.Goal
	}
	phase := 0
	if state != nil {
		phase = state.CurrentPhase
	}

	output := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"goal":      goal,
		"phase":     phase,
		"issues":    issues,
		"exit_code": recoverExitCode(issues),
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": "failed to marshal report: %v"}`, err)
	}
	return string(data) + "\n"
}

// recoverExitCode returns 0 if no issues, 1 if any issues found.
func recoverExitCode(issues []HealthIssue) int {
	if len(issues) > 0 {
		return 1
	}
	return 0
}
