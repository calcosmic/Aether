package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// renderRecoverDiagnosis renders the human-readable diagnosis report for
// aether recover. It follows the same visual patterns as medic_cmd.go
// (renderBanner, renderStageMarker, renderNextUp) for consistency.
func renderRecoverDiagnosis(issues []HealthIssue, state colony.ColonyState) string {
	var b strings.Builder

	b.WriteString(renderBanner(commandEmoji("recover"), "Colony Recovery"))
	b.WriteString(visualDivider)

	// Colony context: goal, phase, state.
	if state.Goal != nil && *state.Goal != "" {
		b.WriteString("Goal: ")
		b.WriteString(*state.Goal)
		b.WriteString("\n")
	}
	totalPhases := len(state.Plan.Phases)
	b.WriteString(fmt.Sprintf("Phase %d/%d", state.CurrentPhase, totalPhases))
	if string(state.State) != "" {
		b.WriteString(fmt.Sprintf(" -- %s", state.State))
	}
	b.WriteString("\n\n")

	// Diagnosis stage.
	b.WriteString(renderStageMarker("Diagnosis"))

	if len(issues) == 0 {
		if shouldUseANSIColors() {
			b.WriteString("\x1b[32m") // green
		}
		b.WriteString("No stuck-state conditions detected. Colony is healthy.")
		if shouldUseANSIColors() {
			b.WriteString("\x1b[0m")
		}
		b.WriteString("\n")
		return b.String()
	}

	// Group issues by severity for ordered display.
	var critical, warnings, infos []HealthIssue
	for _, issue := range issues {
		switch issue.Severity {
		case "critical":
			critical = append(critical, issue)
		case "warning":
			warnings = append(warnings, issue)
		case "info":
			infos = append(infos, issue)
		}
	}

	// Render each severity group.
	if len(critical) > 0 {
		for _, issue := range critical {
			writeRecoverIssueLine(&b, issue)
		}
	}
	if len(warnings) > 0 {
		for _, issue := range warnings {
			writeRecoverIssueLine(&b, issue)
		}
	}
	if len(infos) > 0 {
		for _, issue := range infos {
			writeRecoverIssueLine(&b, issue)
		}
	}
	b.WriteString("\n")

	// Summary stage.
	b.WriteString(renderStageMarker("Summary"))
	b.WriteString(fmt.Sprintf("%d issues found (%d critical, %d warning, %d info)\n",
		len(issues), len(critical), len(warnings), len(infos)))

	fixableCount := 0
	for _, issue := range issues {
		if issue.Fixable {
			fixableCount++
		}
	}
	if fixableCount > 0 {
		b.WriteString(fmt.Sprintf("Run `aether recover --apply` to fix %d issues automatically.\n", fixableCount))
	} else {
		b.WriteString("No automatic fixes available. Review issues above.\n")
	}
	b.WriteString("\n")

	// Next-step suggestion based on most severe category.
	b.WriteString(renderNextUp(recoverNextStep(issues)))

	return b.String()
}

// writeRecoverIssueLine writes a single issue line with fixable hint.
func writeRecoverIssueLine(b *strings.Builder, issue HealthIssue) {
	writeIssueLine(b, issue)
	if issue.Fixable {
		if shouldUseANSIColors() {
			b.WriteString("\x1b[2m") // dim
		}
		b.WriteString("    Fixable with --apply\n")
		if shouldUseANSIColors() {
			b.WriteString("\x1b[0m")
		}
	} else {
		hint := recoverFixHint(issue.Category)
		if hint != "" {
			if shouldUseANSIColors() {
				b.WriteString("\x1b[2m") // dim
			}
			b.WriteString("    ")
			b.WriteString(hint)
			b.WriteString("\n")
			if shouldUseANSIColors() {
				b.WriteString("\x1b[0m")
			}
		}
	}
}

// recoverFixHint returns a human-readable hint for non-fixable issues.
func recoverFixHint(category string) string {
	switch category {
	case "dirty_worktree":
		return "Needs --apply with confirmation"
	case "bad_manifest":
		return "Needs --force for manual repair"
	case "state":
		return "Check colony initialization"
	default:
		return ""
	}
}

// recoverNextStep returns a next-step suggestion based on the most severe issue.
func recoverNextStep(issues []HealthIssue) string {
	// Find the highest-severity issue category.
	for _, issue := range issues {
		if issue.Severity != "critical" {
			continue
		}
		switch issue.Category {
		case "missing_build_packet":
			return "Run `aether build <phase>` to re-dispatch the build."
		case "partial_phase":
			return "Run `aether continue` to advance the colony."
		case "stale_spawned":
			return "Run `aether recover --apply` to reset stale spawn state."
		case "bad_manifest":
			return "Run `aether recover --apply --force` to repair the manifest."
		case "dirty_worktree":
			return "Commit or stash changes in worktrees, then re-run recover."
		default:
			return "Review the critical issues above."
		}
	}

	// Check warnings next.
	for _, issue := range issues {
		if issue.Severity != "warning" {
			continue
		}
		switch issue.Category {
		case "missing_agents":
			return "Run `aether update --force` to restore missing agent files."
		case "broken_survey":
			return "Run `aether colonize` to regenerate survey data."
		case "partial_phase":
			return "Run `aether continue` to advance the colony."
		default:
			return "Review the warnings above."
		}
	}

	return "Run `aether recover --apply` to fix auto-fixable issues."
}

// recoverJSONOutput is the structured output for JSON rendering.
type recoverJSONOutput struct {
	Timestamp      string         `json:"timestamp"`
	Goal           string         `json:"goal"`
	Phase          int            `json:"phase"`
	TotalPhases    int            `json:"total_phases"`
	State          string         `json:"state"`
	Issues         []HealthIssue  `json:"issues"`
	Summary        recoverSummary `json:"summary"`
	ExitCode       int            `json:"exit_code"`
	ScanDurationMs int64          `json:"scan_duration_ms"`
}

type recoverSummary struct {
	Critical int `json:"critical"`
	Warning  int `json:"warning"`
	Info     int `json:"info"`
	Fixable  int `json:"fixable"`
}

// renderRecoverJSON renders the structured JSON diagnosis report.
func renderRecoverJSON(issues []HealthIssue, state colony.ColonyState, duration time.Duration) string {
	goal := ""
	if state.Goal != nil {
		goal = *state.Goal
	}

	summary := recoverSummary{}
	for _, issue := range issues {
		switch issue.Severity {
		case "critical":
			summary.Critical++
		case "warning":
			summary.Warning++
		case "info":
			summary.Info++
		}
		if issue.Fixable {
			summary.Fixable++
		}
	}

	output := recoverJSONOutput{
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		Goal:           goal,
		Phase:          state.CurrentPhase,
		TotalPhases:    len(state.Plan.Phases),
		State:          string(state.State),
		Issues:         issues,
		Summary:        summary,
		ExitCode:       recoverExitCode(issues),
		ScanDurationMs: duration.Milliseconds(),
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": "failed to marshal report: %v"}`, err)
	}
	return string(data) + "\n"
}

// recoverExitCode returns 0 if no issues, 1 if any issues found.
// The exit code enables shell script integration: healthy colonies exit 0,
// colonies with issues exit 1.
func recoverExitCode(issues []HealthIssue) int {
	if len(issues) > 0 {
		return 1
	}
	return 0
}
