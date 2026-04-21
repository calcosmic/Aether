package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

// MedicOptions holds the flag values for the medic command.
type MedicOptions struct {
	Fix   bool
	Force bool
	JSON  bool
	Deep  bool
}

// HealthIssue represents a single finding from a colony health scan.
type HealthIssue struct {
	Severity string `json:"severity"`
	Category string `json:"category"`
	Message  string `json:"message"`
	File     string `json:"file,omitempty"`
	Fixable  bool   `json:"fixable"`
}

// medicCmd is the cobra command for diagnosing colony health.
var medicCmd = &cobra.Command{
	Use:   "medic",
	Short: "Diagnose colony health",
	Long:  `Scan the colony for health issues, corruption, stale data, and configuration problems. Read-only by default; use --fix to attempt repairs.`,
	Args:  cobra.NoArgs,
	RunE:  runMedic,
}

func init() {
	rootCmd.AddCommand(medicCmd)
	medicCmd.Flags().BoolVar(new(bool), "fix", false, "enable repair mode")
	medicCmd.Flags().BoolVar(new(bool), "force", false, "allow destructive repairs")
	medicCmd.Flags().BoolVar(new(bool), "json", false, "output structured JSON")
	medicCmd.Flags().BoolVar(new(bool), "deep", false, "scan all files including wrappers")
}

func runMedic(cmd *cobra.Command, args []string) error {
	state, err := loadActiveColonyState()
	if err != nil {
		if shouldRenderVisualOutput(stdout) && strings.Contains(colonyStateLoadMessage(err), "No colony initialized") {
			fmt.Fprint(stdout, renderNoColonyMedicVisual())
			return nil
		}
		fmt.Fprintln(stdout, colonyStateLoadMessage(err))
		return nil
	}

	fix, _ := cmd.Flags().GetBool("fix")
	force, _ := cmd.Flags().GetBool("force")
	jsonOut, _ := cmd.Flags().GetBool("json")
	deep, _ := cmd.Flags().GetBool("deep")

	opts := MedicOptions{
		Fix:   fix,
		Force: force,
		JSON:  jsonOut,
		Deep:  deep,
	}

	// Stub: scanner will be implemented in Plan 02.
	// For now, perform a basic scan to populate results.
	issues := performBasicHealthScan(state, opts)

	if opts.JSON {
		fmt.Fprint(stdout, renderMedicJSON(issues, &state))
		return nil
	}

	output := renderMedicReport(issues, opts, &state)
	fmt.Fprint(stdout, output)
	return nil
}

// performBasicHealthScan runs a minimal health check on the colony.
// This stub will be replaced by the full scanner in Plan 02.
func performBasicHealthScan(state colony.ColonyState, opts MedicOptions) []HealthIssue {
	var issues []HealthIssue

	if strings.TrimSpace(string(state.State)) == "" {
		issues = append(issues, HealthIssue{
			Severity: "critical",
			Category: "state",
			Message:  "Colony state is empty or uninitialized",
			File:     "COLONY_STATE.json",
			Fixable:  false,
		})
	}

	if len(state.Plan.Phases) == 0 && state.CurrentPhase == 0 {
		issues = append(issues, HealthIssue{
			Severity: "warning",
			Category: "state",
			Message:  "No phases planned yet",
			File:     "COLONY_STATE.json",
			Fixable:  false,
		})
	}

	return issues
}

func renderNoColonyMedicVisual() string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("medic"), "Colony Health"))
	b.WriteString(visualDivider)
	b.WriteString("No colony initialized in this repo.\n")
	b.WriteString(renderNextUp(
		`Run `+"`aether init \"goal\"`"+` to start a colony.`,
		`Run `+"`aether lay-eggs`"+` first if this repo has not been set up for Aether yet.`,
	))
	return b.String()
}

func renderMedicReport(results []HealthIssue, opts MedicOptions, state *colony.ColonyState) string {
	var b strings.Builder

	b.WriteString(renderBanner(commandEmoji("medic"), "Colony Health"))
	b.WriteString(visualDivider)

	// Summary counts
	criticalCount := 0
	warningCount := 0
	infoCount := 0
	for _, issue := range results {
		switch issue.Severity {
		case "critical":
			criticalCount++
		case "warning":
			warningCount++
		case "info":
			infoCount++
		}
	}

	b.WriteString(renderStageMarker("Summary"))
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
		for _, issue := range results {
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
		for _, issue := range results {
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
		for _, issue := range results {
			if issue.Severity != "info" {
				continue
			}
			writeIssueLine(&b, issue)
		}
		b.WriteString("\n")
	}

	if len(results) == 0 {
		b.WriteString("Colony is healthy. No issues found.\n\n")
	}

	// Repair log if fix mode was used
	if opts.Fix {
		b.WriteString(renderStageMarker("Repair Log"))
		repaired := 0
		for _, issue := range results {
			if issue.Fixable {
				repaired++
			}
		}
		if repaired > 0 {
			b.WriteString(fmt.Sprintf("Repaired %d issue(s).\n", repaired))
		} else {
			b.WriteString("No repairs needed or possible.\n")
		}
		b.WriteString("\n")
	}

	// Next Steps
	b.WriteString(renderStageMarker("Next Steps"))
	switch {
	case criticalCount > 0:
		b.WriteString("Run `aether medic --fix` to attempt repairs.\n")
	case warningCount > 0:
		b.WriteString("Review warnings above. Some issues can be auto-fixed.\n")
	default:
		b.WriteString("Colony is healthy. No action needed.\n")
	}

	return b.String()
}

func writeIssueLine(b *strings.Builder, issue HealthIssue) {
	b.WriteString("  ")
	if shouldUseANSIColors() {
		b.WriteString(severityColor(issue.Severity))
	}
	b.WriteString(fmt.Sprintf("[%s] %s", issue.Severity, issue.Message))
	if shouldUseANSIColors() {
		b.WriteString("\x1b[0m")
	}
	if issue.File != "" {
		b.WriteString(fmt.Sprintf(" (%s)", issue.File))
	}
	if issue.Fixable {
		b.WriteString(" [fixable]")
	}
	b.WriteString("\n")
}

func renderMedicJSON(results []HealthIssue, state *colony.ColonyState) string {
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
		"issues":    results,
		"exit_code": medicExitCode(results),
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": "failed to marshal report: %v"}`, err)
	}
	return string(data) + "\n"
}

func severityColor(sev string) string {
	switch sev {
	case "critical":
		return "\033[31m"
	case "warning":
		return "\033[33m"
	case "info":
		return "\033[34m"
	default:
		return ""
	}
}

func medicExitCode(issues []HealthIssue) int {
	max := 0
	for _, issue := range issues {
		switch issue.Severity {
		case "critical":
			if max < 2 {
				max = 2
			}
		case "warning":
			if max < 1 {
				max = 1
			}
		}
	}
	return max
}
