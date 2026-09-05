package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// renderRecoverDiagnosis renders the human-readable diagnosis report for the
// retired recovery scanner. It follows the same visual patterns as medic_cmd.go
// (renderBanner, renderStageMarker, renderNextUp) for consistency.
func renderRecoverDiagnosis(issues []HealthIssue, state colony.ColonyState, repairResult *RepairResult) string {
	var b strings.Builder

	b.WriteString(renderBanner(commandEmoji("recover"), "Colony Recovery"))
	b.WriteString(visualDividerStr())

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
		b.WriteString("\n\n")
		b.WriteString(renderNextActionCard(recoverNextAction(issues, state)))
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

	// Repair Log stage (if repairs were performed).
	if repairResult != nil {
		b.WriteString(renderStageMarker("Repair Log"))
		b.WriteString(renderRepairLog(repairResult))
		b.WriteString("\n")
	}

	// Summary stage.
	b.WriteString(renderStageMarker("Summary"))

	b.WriteString(fmt.Sprintf("%d issues found (%d critical, %d warning, %d info)\n",
		len(issues), len(critical), len(warnings), len(infos)))

	b.WriteString("Inspect the evidence with `aether maintenance recovery-inspect`; state effect: none.\n")
	b.WriteString("Run `aether resume` only when you are ready to restore lifecycle progress.\n")
	b.WriteString("\n")

	// The one resolver's answer, fed this scan's own top finding as an
	// override -- recover.go's private next-step decider is retired.
	b.WriteString(renderNextActionCard(recoverNextAction(issues, state)))

	return b.String()
}

// writeRecoverIssueLine writes a single issue line with fixable hint.
func writeRecoverIssueLine(b *strings.Builder, issue HealthIssue) {
	writeIssueLine(b, issue)
	if issue.Fixable {
		if shouldUseANSIColors() {
			b.WriteString("\x1b[2m") // dim
		}
		b.WriteString("    Inspect with aether maintenance recovery-inspect; it does not change state.\n")
		if shouldUseANSIColors() {
			b.WriteString("\x1b[0m")
		}
	}
}

// recoverFixHint returns a human-readable hint for non-fixable issues.
func recoverFixHint(category string) string {
	switch category {
	case "dirty_worktree":
		return "Inspect preserved work before choosing an owner-authorized maintenance mutation"
	case "bad_manifest":
		return "Inspect the manifest before choosing an owner-authorized maintenance mutation"
	case "state":
		return "Check colony initialization"
	default:
		return ""
	}
}

// recoverOverrideFromIssues turns this scan's own findings into a command and
// a plain-English reason for the one decision -- what recover found is
// something the saved project alone cannot tell the resolver, so it is fed in
// as an override rather than deciding on its own, second, private path. This
// is the sixth and last hand-rolled next-step function in the runtime; the
// others were retired in plans 197-02 and 197-04.
//
// A category with no command (the two "review the ... above" defaults) is
// deliberately not a command at all: the findings are already listed above
// the card, and there is nothing more specific to run than the ordinary
// answer the resolver already gives.
func recoverOverrideFromIssues(issues []HealthIssue, state colony.ColonyState) (string, string) {
	for _, issue := range issues {
		if issue.Severity != "critical" {
			continue
		}
		switch issue.Category {
		case "missing_build_packet":
			return "aether resume",
				"A build record is missing for the current phase. Resume reconciles the durable evidence before any new work starts."
		case "partial_phase":
			return "aether continue",
				"This phase only partly finished. Checking it moves the colony on if the work holds up."
		case "stale_spawned", "bad_manifest", "dirty_worktree":
			return "aether maintenance recovery-inspect",
				"Inspect the durable evidence without changing state, then use aether resume as the only lifecycle restoration command."
		}
		return "", ""
	}

	// Check warnings next.
	for _, issue := range issues {
		if issue.Severity != "warning" {
			continue
		}
		switch issue.Category {
		case "missing_agents":
			return "aether maintenance recovery-inspect",
				"Inspect missing runtime evidence without changing state, then use aether resume as the only lifecycle restoration command."
		case "broken_survey":
			return "aether colonize",
				"The saved scan of the existing code is broken. Re-scanning rebuilds it."
		case "partial_phase":
			return "aether continue",
				"This phase only partly finished. Checking it moves the colony on if the work holds up."
		case "unreconciled_worker_changes":
			return "aether build-reconcile",
				"A helper made changes that were never recorded. This records them."
		}
		return "", ""
	}

	if len(issues) > 0 {
		return "aether maintenance recovery-inspect", "Inspect the issues without changing state, then use aether resume as the only lifecycle restoration command."
	}
	return "", ""
}

// recoverNextAction resolves the one closing answer for a recover run,
// feeding this scan's own top finding in as an override.
func recoverNextAction(issues []HealthIssue, state colony.ColonyState) nextAction {
	command, why := recoverOverrideFromIssues(issues, state)
	return lifecycleNextActionForState(state, "recover", command, why)
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
func renderRecoverJSON(issues []HealthIssue, state colony.ColonyState, duration time.Duration, repairResult *RepairResult) string {
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

	// Use a map to build the full output, adding repairs if present.
	outputMap := map[string]interface{}{
		"timestamp":        output.Timestamp,
		"goal":             output.Goal,
		"phase":            output.Phase,
		"total_phases":     output.TotalPhases,
		"state":            output.State,
		"issues":           output.Issues,
		"summary":          output.Summary,
		"exit_code":        output.ExitCode,
		"scan_duration_ms": output.ScanDurationMs,
	}
	if repairResult != nil {
		outputMap["repairs"] = map[string]interface{}{
			"attempted": repairResult.Attempted,
			"succeeded": repairResult.Succeeded,
			"failed":    repairResult.Failed,
			"skipped":   repairResult.Skipped,
			"details":   repairResult.Repairs,
		}
	}
	// The same closing answer the text report renders, carrying the stable
	// keys every migrated command's envelope carries -- a wrapper reading
	// this JSON and an owner reading the text report are told the same thing.
	applyNextActionToResult(outputMap, recoverNextAction(issues, state))
	data, err := json.MarshalIndent(outputMap, "", "  ")
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

// renderRepairLog renders a human-readable log of repair actions taken.
func renderRepairLog(result *RepairResult) string {
	var b strings.Builder
	for _, rec := range result.Repairs {
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
		result.Attempted, result.Succeeded, result.Failed, result.Skipped))
	return b.String()
}
