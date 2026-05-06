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

// MedicOptions holds the flag values for the medic command.
type MedicOptions struct {
	Fix   bool
	Force bool
	JSON  bool
	Deep  bool
	Trace string // path to trace export JSON
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
	medicCmd.Flags().BoolVar(new(bool), "deep", false, "scan all files including wrappers, hub publish surfaces, and ceremony integrity")
	medicCmd.Flags().String("trace", "", "analyze a trace export JSON file instead of colony data")
}

func runMedic(cmd *cobra.Command, args []string) error {
	// Trace diagnostics mode — works without an active colony
	traceFile, _ := cmd.Flags().GetString("trace")
	jsonOut, _ := cmd.Flags().GetBool("json")
	if traceFile != "" {
		entries, err := loadTraceExport(traceFile)
		if err != nil {
			fmt.Fprintf(stdout, "Error: %v\n", err)
			return nil
		}
		diag := analyzeTraceDiagnostics(entries)
		if jsonOut || !shouldRenderVisualOutput(stdout) {
			fmt.Fprint(stdout, renderTraceDiagnosticJSON(diag))
		} else {
			fmt.Fprint(stdout, renderTraceDiagnostic(diag))
		}
		return nil
	}

	state, err := loadActiveColonyState()
	if err != nil {
		message := colonyStateLoadMessage(err)
		if strings.Contains(message, "No colony initialized") {
			if shouldRenderVisualOutput(stdout) {
				fmt.Fprint(stdout, renderNoColonyMedicVisual())
			} else {
				outputOK(map[string]interface{}{"status": "no_colony", "message": message, "issues": []HealthIssue{}})
			}
			return nil
		}
		fmt.Fprintln(stdout, message)
		return nil
	}

	fix, _ := cmd.Flags().GetBool("fix")
	force, _ := cmd.Flags().GetBool("force")
	deep, _ := cmd.Flags().GetBool("deep")

	opts := MedicOptions{
		Fix:   fix,
		Force: force,
		JSON:  jsonOut || !shouldRenderVisualOutput(stdout),
		Deep:  deep,
	}

	// Run the full health scanner.
	scanResult, err := performHealthScan(opts)
	if err != nil {
		fmt.Fprintf(stdout, "Health scan failed: %v\n", err)
		return nil
	}
	issues := scanResult.Issues

	var repairResult *RepairResult
	if opts.Fix {
		dataPath := filepath.Join(resolveAetherRoot(), ".aether", "data")
		repairResult, err = performRepairs(scanResult, opts, dataPath)
		if err != nil {
			fmt.Fprintf(stdout, "Repair failed: %v\n", err)
			// Still show the original scan results
			if opts.JSON {
				fmt.Fprint(stdout, renderMedicJSON(issues, &state, nil))
				return nil
			}
			output := renderMedicReport(issues, opts, &state, nil)
			fmt.Fprint(stdout, output)
			return nil
		}

		// Re-scan to get post-repair state
		postResult, err := performHealthScan(opts)
		if err != nil {
			fmt.Fprintf(stdout, "Post-repair scan failed: %v\n", err)
			// Use original issues
		} else {
			issues = postResult.Issues
		}
	}

	// Persist scan results for colony-prime context injection
	medicDataPath := filepath.Join(resolveAetherRoot(), ".aether", "data")
	medicGoal := ""
	if state.Goal != nil {
		medicGoal = *state.Goal
	}
	_ = saveMedicLastScan(medicDataPath, issues, medicGoal, state.CurrentPhase)

	if opts.JSON {
		fmt.Fprint(stdout, renderMedicJSON(issues, &state, repairResult))
		return nil
	}

	output := renderMedicReport(issues, opts, &state, repairResult)
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

func renderMedicReport(results []HealthIssue, opts MedicOptions, state *colony.ColonyState, repairResult *RepairResult) string {
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

	// Repair log if fix mode was used and repairs were performed
	if opts.Fix && repairResult != nil {
		b.WriteString(renderStageMarker("Repair Log"))
		for _, rec := range repairResult.Repairs {
			status := "OK"
			if !rec.Success {
				status = "FAILED"
			}
			b.WriteString(fmt.Sprintf("  [%s] %s", status, rec.Action))
			if rec.File != "" {
				b.WriteString(fmt.Sprintf(" (%s)", rec.File))
			}
			if rec.Error != "" {
				b.WriteString(fmt.Sprintf(": %s", rec.Error))
			}
			b.WriteString("\n")
		}
		b.WriteString(fmt.Sprintf("Summary: %d attempted, %d succeeded, %d failed, %d skipped\n",
			repairResult.Attempted, repairResult.Succeeded,
			repairResult.Failed, repairResult.Skipped))
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

func renderMedicJSON(results []HealthIssue, state *colony.ColonyState, repairResult *RepairResult) string {
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

	if repairResult != nil {
		output["repairs"] = map[string]interface{}{
			"attempted": repairResult.Attempted,
			"succeeded": repairResult.Succeeded,
			"failed":    repairResult.Failed,
			"skipped":   repairResult.Skipped,
			"details":   repairResult.Repairs,
		}
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
