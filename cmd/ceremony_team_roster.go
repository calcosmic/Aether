package cmd

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/jedib0t/go-pretty/v6/text"
)

// The owner approves people and assignments, not just a list of castes.
// Keep this card plain text so a chat host can display the exact runtime
// result even when terminal output is hidden behind a collapsed tool call.
func renderTeamApprovalCard(manifest map[string]interface{}, dispatches []ceremonyDispatch, required map[string]bool, reasons map[string]string, live, waived []riskSignalHit) string {
	title := "PROPOSED TEAM"
	if phase := intValue(manifest["phase"]); phase > 0 {
		title += fmt.Sprintf(" — PHASE %d", phase)
	}
	lines := []string{title}
	if name := strings.TrimSpace(stringValue(manifest["phase_name"])); name != "" {
		lines = append(lines, name)
	}
	lines = append(lines, "", fmt.Sprintf("BUILD — %d planned worker(s)", len(dispatches)))
	for i, dispatch := range dispatches {
		if i > 0 {
			lines = append(lines, "")
		}
		role := emptyFallback(casteLabel(dispatch.Caste), "Worker")
		name := emptyFallback(dispatch.Name, "name not recorded")
		mark := "optional"
		if required[dispatch.Caste] {
			mark = "required"
		}
		lines = append(lines, fmt.Sprintf("%s %s — %s", role, name, mark))
		if dispatch.ExecutionWave > 0 {
			lines = append(lines, fmt.Sprintf("  When: build wave %d", dispatch.ExecutionWave))
		}
		taskIDs := dispatch.CoveredTaskIDs
		if len(taskIDs) == 0 && dispatch.TaskID != "" {
			taskIDs = []string{dispatch.TaskID}
		}
		if len(taskIDs) > 0 {
			lines = append(lines, "  Tasks: "+strings.Join(taskIDs, ", "))
		}
		lines = append(lines, "  Work: "+emptyFallback(dispatch.Task, "assignment not recorded"))
		if reason := reasons[dispatch.Caste]; reason != "" {
			lines = append(lines, "  Why: "+reason)
		}
	}
	if len(dispatches) == 0 {
		lines = append(lines, "No build workers are listed in this manifest.")
	}
	if len(live) > 0 {
		lines = append(lines, "", "AFTER BUILD — required reviews at aether continue")
		for _, reviewer := range collapseToForcedReviewers(live) {
			job := "required review"
			switch reviewer.Caste {
			case "auditor":
				job = "quality review"
			case "gatekeeper":
				job = "security review"
			}
			lines = append(lines, casteLabel(reviewer.Caste)+" — "+job, "  Why: "+reviewer.Reason)
		}
		lines = append(lines, "Reviewer names are assigned at verification.")
	}
	if len(waived) > 0 {
		lines = append(lines, "", "DECLINED REVIEW SIGNALS")
		for _, hit := range waived {
			lines = append(lines, hit.Signal.Name+": "+hit.WaiverReason)
		}
	}
	lines = append(lines, "", "Approve this team, adjust it, or redirect the work.")
	return frameTeamApprovalCard(lines)
}

// A bounded-width frame is an explicit owner request for this approval
// moment. Other runtime displays keep their existing headed-section style.
func frameTeamApprovalCard(lines []string) string {
	const width = 72
	var b strings.Builder
	b.WriteString("┌" + strings.Repeat("─", width+2) + "┐\n")
	for _, line := range lines {
		line = strings.Map(func(r rune) rune {
			if r == '\t' || r == '\r' {
				return ' '
			}
			if unicode.IsControl(r) && r != '\n' {
				return -1
			}
			return r
		}, text.StripEscape(line))
		for _, paragraph := range strings.Split(line, "\n") {
			indent := min(len(paragraph)-len(strings.TrimLeft(paragraph, " ")), width/2)
			prefix := strings.Repeat(" ", indent)
			for _, row := range strings.Split(text.WrapSoft(strings.TrimLeft(paragraph, " "), width-indent), "\n") {
				row = prefix + row
				b.WriteString("│ " + row + strings.Repeat(" ", max(0, width-text.StringWidth(row))) + " │\n")
			}
		}
	}
	b.WriteString("└" + strings.Repeat("─", width+2) + "┘\n")
	return b.String()
}
