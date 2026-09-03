package cmd

// The lifecycle status renderers consume an already-projected snapshot. They
// never read files, inspect mutable state, or derive a second lifecycle answer.

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/colony"
)

const lifecycleStatusDefaultWidth = 100

func lifecycleStatusOutputWidth() int {
	width, err := strconv.Atoi(strings.TrimSpace(os.Getenv("COLUMNS")))
	if err != nil || width < 40 {
		return lifecycleStatusDefaultWidth
	}
	return width
}

// renderLifecycleStatus is the visual boundary for the shared lifecycle
// projection. Full and compact consume the same values; compact merely omits
// full-view sections and never recomputes a lifecycle fact.
func renderLifecycleStatus(projection LifecycleProjection, width int) string {
	if width < 24 {
		width = 24
	}
	if projection.View == LifecycleViewCompact {
		return colorLifecycleStatus(renderLifecycleStatusCompact(projection, width))
	}
	return colorLifecycleStatus(renderLifecycleStatusFull(projection, width))
}

func renderLifecycleStatusFull(projection LifecycleProjection, width int) string {
	lines := []string{"━━ 🐜 C O L O N Y   S T A T U S ━━"}
	identity := projection.Identity.Value
	name := emptyFallback(strings.TrimSpace(identity.Name), "Unnamed colony")
	goal := emptyFallback(strings.TrimSpace(projection.Goal.Value), "Not recorded")
	standing := emptyFallback(strings.TrimSpace(projection.Standing.Value), "UNKNOWN")

	lines = append(lines,
		"",
		"Colony",
		"Name: "+name,
		"Goal: "+goal,
		"Standing: "+standing,
	)
	if identity.Scope != "" || identity.Mode != "" {
		lines = append(lines, fmt.Sprintf("Scope: %s | Mode: %s", emptyFallback(identity.Scope, "Not recorded"), emptyFallback(identity.Mode, "Not recorded")))
	}

	phase := projection.Phase.Value
	lines = append(lines, "", "Phase & Tasks")
	if phase.TotalPhases == 0 {
		lines = append(lines, "Phase: Not recorded")
	} else {
		phaseName := ""
		phaseStatus := ""
		if phase.Current != nil {
			phaseName = strings.TrimSpace(phase.Current.Name)
			phaseStatus = strings.TrimSpace(phase.Current.Status)
		}
		phaseLine := fmt.Sprintf("Phase %d/%d", phase.CurrentNumber, phase.TotalPhases)
		if phaseName != "" {
			phaseLine += ": " + phaseName
		}
		if phaseStatus != "" {
			phaseLine += " [" + phaseStatus + "]"
		}
		lines = append(lines, phaseLine)
	}
	lines = append(lines, fmt.Sprintf("Tasks: %d/%d complete", phase.CompletedTasks, phase.TotalTasks))

	lines = append(lines, "", "Ants & Outcomes")
	active, terminal := lifecycleStatusActors(projection.Actors.Value)
	if len(active) == 0 {
		lines = append(lines, "No ants are active")
	} else {
		for _, actorFact := range active {
			lines = append(lines, lifecycleStatusActorLine("Active", actorFact))
		}
	}
	for _, actorFact := range lifecycleStatusTailActors(terminal, 3) {
		lines = append(lines, lifecycleStatusActorLine("Recent outcome", actorFact))
	}
	for _, lineage := range projection.Lineage.Value {
		if strings.TrimSpace(lineage.Parent) == "" || strings.TrimSpace(lineage.Actor) == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("Lineage: %s → %s", lineage.Parent, lineage.Actor))
	}

	lines = append(lines, "", "Pheromones")
	activeSignals := 0
	for _, signal := range projection.Signals.Value {
		if !signal.Active {
			continue
		}
		activeSignals++
		content := strings.TrimSpace(extractContentText(signal.Content))
		if content == "" {
			content = "Content not recorded"
		}
		lines = append(lines, fmt.Sprintf("%s: %s", emptyFallback(strings.TrimSpace(signal.Type), "Signal"), content))
	}
	if activeSignals == 0 {
		lines = append(lines, "No active pheromones recorded")
	}

	lines = append(lines, "", "Territory & Notes")
	research := projection.Research.Value
	lines = append(lines, fmt.Sprintf("Research documents: %d", len(research.Docs)))
	if latest := lifecycleStatusLatestPath(research.Docs); latest != "" {
		lines = append(lines, "Latest research: "+latest)
	}
	lines = append(lines, fmt.Sprintf("Territory records: %d", len(research.Territory)))
	if latest := lifecycleStatusLatestPath(research.Territory); latest != "" {
		lines = append(lines, "Latest territory record: "+latest)
	}
	lines = append(lines, fmt.Sprintf("Dreams notes: %d", len(research.Dreams)))
	if latest := lifecycleStatusLatestPath(research.Dreams); latest != "" {
		lines = append(lines, "Latest Dreams note (Unverified local note): "+latest)
	}

	lines = append(lines, "", "Memory, Findings & Gates")
	memory := projection.Memory.Value
	openFindings := 0
	for _, finding := range projection.Findings.Value {
		if strings.EqualFold(strings.TrimSpace(finding.Status), "open") {
			openFindings++
		}
	}
	lines = append(lines, fmt.Sprintf("Memory: %d instincts | %d observations", len(memory.Instincts), len(memory.Observations)))
	lines = append(lines, fmt.Sprintf("Findings: %d open | %d recorded", openFindings, len(projection.Findings.Value)))
	verificationLabel := "Unknown"
	if len(projection.Verification) > 0 {
		verificationLabel = "Verified"
		for _, gate := range projection.Verification {
			if !gate.Passed {
				verificationLabel = "Blocked"
				break
			}
		}
	}
	lines = append(lines, "Verification: "+verificationLabel)
	for _, gate := range projection.Verification {
		status := "Passed"
		if !gate.Passed {
			status = "Failed"
		}
		line := fmt.Sprintf("Gate %s: %s", emptyFallback(strings.TrimSpace(gate.Name), "unnamed"), status)
		if strings.TrimSpace(gate.Detail) != "" {
			line += " — " + strings.TrimSpace(gate.Detail)
		}
		lines = append(lines, line)
	}
	for _, evidence := range projection.Evidence {
		line := "Evidence: " + emptyFallback(strings.TrimSpace(evidence.ID), "unnamed")
		if strings.TrimSpace(evidence.Summary) != "" {
			line += " — " + strings.TrimSpace(evidence.Summary)
		}
		lines = append(lines, line)
	}

	lines = append(lines, "", "Elapsed & Reported Cost")
	if projection.Elapsed.Source.Provenance == LifecycleFactConfirmed {
		lines = append(lines, "Elapsed: "+lifecycleStatusDuration(projection.Elapsed.Value))
	} else {
		lines = append(lines, "Elapsed: Unknown")
	}
	lines = append(lines, "Reported cost: "+lifecycleStatusReportedCost(projection.ReportedCost))

	lines = append(lines, "", "Recent History")
	if len(projection.History.Value) == 0 {
		lines = append(lines, "No history is recorded")
	} else {
		start := len(projection.History.Value) - 5
		if start < 0 {
			start = 0
		}
		for _, event := range projection.History.Value[start:] {
			lines = append(lines, "• "+strings.TrimSpace(event))
		}
	}

	lines = append(lines, "", "Open Items")
	openCount := 0
	for _, issue := range projection.Warnings {
		openCount++
		lines = append(lines, lifecycleStatusIssueLine("Warning", issue))
	}
	for _, issue := range projection.Debt {
		openCount++
		lines = append(lines, lifecycleStatusIssueLine("Debt", issue))
	}
	for _, issue := range projection.Blockers {
		openCount++
		lines = append(lines, lifecycleStatusIssueLine("Blocker", issue))
	}
	for _, decision := range projection.OwnerDecisions {
		openCount++
		line := fmt.Sprintf("Owner decision %s: %s", emptyFallback(strings.TrimSpace(decision.ID), "unnamed"), emptyFallback(strings.TrimSpace(decision.Summary), "Reason not recorded"))
		lines = append(lines, line)
	}
	if openCount == 0 {
		lines = append(lines, "No open items recorded")
	}

	lines = append(lines, "", "Next Up")
	command := lifecycleStatusActionCommand(projection.NextAction)
	lines = append(lines, "Command: "+emptyFallback(command, "Not available"))
	lines = append(lines, "Reason: "+emptyFallback(strings.TrimSpace(projection.NextAction.Reason), "Not recorded"))
	for _, choice := range projection.NextAction.Choices {
		lines = append(lines, "Choice: "+lifecycleStatusChoiceLine(choice))
	}
	for _, alternative := range projection.Alternatives {
		lines = append(lines, "Alternative: "+lifecycleStatusChoiceLine(alternative))
	}

	return lifecycleStatusJoin(lines, width, false)
}

func renderLifecycleStatusCompact(projection LifecycleProjection, width int) string {
	identity := projection.Identity.Value
	name := emptyFallback(strings.TrimSpace(identity.Name), "Unnamed colony")
	standing := emptyFallback(strings.TrimSpace(projection.Standing.Value), "UNKNOWN")
	phase := projection.Phase.Value
	active, terminal := lifecycleStatusActors(projection.Actors.Value)
	openCount := len(projection.Warnings) + len(projection.Debt) + len(projection.Blockers) + len(projection.OwnerDecisions)
	research := projection.Research.Value

	lines := []string{
		"━━ 🐜 C O L O N Y   S T A T U S ━━",
		fmt.Sprintf("%s — %s", name, standing),
		"Goal: " + emptyFallback(strings.TrimSpace(projection.Goal.Value), "Not recorded"),
		fmt.Sprintf("Phase %d/%d | Tasks %d/%d", phase.CurrentNumber, phase.TotalPhases, phase.CompletedTasks, phase.TotalTasks),
	}
	if len(active) == 0 {
		lines = append(lines, "No ants are active")
	} else {
		lines = append(lines, fmt.Sprintf("Active ants: %d | %s", len(active), active[0].Name))
	}
	if len(terminal) > 0 {
		latest := terminal[len(terminal)-1]
		lines = append(lines, fmt.Sprintf("Latest outcome: %s — %s", emptyFallback(latest.Name, "Unnamed ant"), emptyFallback(latest.Status, "status unknown")))
	}
	lines = append(lines,
		fmt.Sprintf("Territory: %d records | Research: %d docs | Notes: %d unverified", len(research.Territory), len(research.Docs), len(research.Dreams)),
		fmt.Sprintf("Open items: %d | Blockers: %d | Owner decisions: %d", openCount, len(projection.Blockers), len(projection.OwnerDecisions)),
		fmt.Sprintf("Elapsed: %s | Reported cost: %s", lifecycleStatusCompactElapsed(projection.Elapsed), lifecycleStatusReportedCost(projection.ReportedCost)),
		"Next Up: "+emptyFallback(lifecycleStatusActionCommand(projection.NextAction), "Not available"),
	)
	return lifecycleStatusJoin(lines, width, true)
}

func lifecycleStatusActors(actors []LifecycleActorFact) (active, terminal []LifecycleActorFact) {
	for _, actorFact := range actors {
		switch {
		case agent.IsLiveSpawnStatus(actorFact.Status):
			active = append(active, actorFact)
		case agent.IsTerminalSpawnStatus(actorFact.Status):
			terminal = append(terminal, actorFact)
		}
	}
	return active, terminal
}

func lifecycleStatusTailActors(actors []LifecycleActorFact, limit int) []LifecycleActorFact {
	if len(actors) <= limit {
		return actors
	}
	return actors[len(actors)-limit:]
}

func lifecycleStatusActorLine(prefix string, actorFact LifecycleActorFact) string {
	name := emptyFallback(strings.TrimSpace(actorFact.Name), "Unnamed ant")
	caste := strings.TrimSpace(actorFact.Caste)
	status := emptyFallback(strings.TrimSpace(actorFact.Status), "status unknown")
	line := fmt.Sprintf("%s: %s", prefix, name)
	if caste != "" {
		line += " (" + caste + ")"
	}
	line += " — " + status
	if strings.TrimSpace(actorFact.Summary) != "" {
		line += ": " + strings.TrimSpace(actorFact.Summary)
	} else if strings.TrimSpace(actorFact.Task) != "" {
		line += ": " + strings.TrimSpace(actorFact.Task)
	}
	return line
}

func lifecycleStatusLatestPath(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	ordered := append([]string(nil), paths...)
	sort.Strings(ordered)
	return strings.TrimSpace(ordered[len(ordered)-1])
}

func lifecycleStatusDuration(value time.Duration) string {
	if value < 0 {
		return "Unknown"
	}
	return value.String()
}

func lifecycleStatusCompactElapsed(elapsed LifecycleFact[time.Duration]) string {
	if elapsed.Source.Provenance != LifecycleFactConfirmed {
		return "Unknown"
	}
	return lifecycleStatusDuration(elapsed.Value)
}

func lifecycleStatusReportedCost(cost LifecycleFact[LifecycleReportedCostFacts]) string {
	if cost.Source.Provenance != LifecycleFactConfirmed || cost.Value.Rows < 1 {
		return "Unreported"
	}
	return fmt.Sprintf("%d tokens", cost.Value.TotalTokens)
}

func lifecycleStatusIssueLine(kind string, issue colony.LifecycleIssue) string {
	return fmt.Sprintf("%s %s: %s", kind, emptyFallback(strings.TrimSpace(issue.ID), "unnamed"), emptyFallback(strings.TrimSpace(issue.Summary), "Reason not recorded"))
}

func lifecycleStatusActionCommand(action LifecycleProjectedAction) string {
	if command := strings.TrimSpace(action.DisplayCommand); command != "" {
		return command
	}
	return strings.TrimSpace(action.RuntimeCommand)
}

func lifecycleStatusChoiceLine(choice LifecycleActionChoice) string {
	command := strings.TrimSpace(choice.DisplayCommand)
	if command == "" {
		command = strings.TrimSpace(choice.RuntimeCommand)
	}
	if reason := strings.TrimSpace(choice.Reason); reason != "" {
		return command + " — " + reason
	}
	return command
}

func lifecycleStatusJoin(lines []string, width int, compact bool) string {
	rendered := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimRight(line, " \t")
		if compact {
			rendered = append(rendered, lifecycleStatusFitLine(line, width))
			continue
		}
		rendered = append(rendered, lifecycleStatusWrapLine(line, width)...)
	}
	return strings.Join(rendered, "\n") + "\n"
}

func lifecycleStatusWrapLine(line string, width int) []string {
	if line == "" || utf8.RuneCountInString(line) <= width {
		return []string{line}
	}
	words := strings.Fields(line)
	if len(words) == 0 {
		return []string{""}
	}
	result := make([]string, 0, 2)
	current := ""
	for _, word := range words {
		word = lifecycleStatusFitLine(word, width-2)
		candidate := word
		if current != "" {
			candidate = current + " " + word
		}
		if utf8.RuneCountInString(candidate) <= width {
			current = candidate
			continue
		}
		if current != "" {
			result = append(result, current)
		}
		current = "  " + word
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func lifecycleStatusFitLine(line string, width int) string {
	if width < 1 || utf8.RuneCountInString(line) <= width {
		return line
	}
	if width < 5 {
		return string([]rune(line)[:width])
	}
	runes := []rune(line)
	left := (width - 1) / 2
	right := width - 1 - left
	return string(runes[:left]) + "…" + string(runes[len(runes)-right:])
}

func colorLifecycleStatus(output string) string {
	if !shouldUseANSIColors() {
		return output
	}
	headings := map[string]bool{
		"Colony": true, "Phase & Tasks": true, "Ants & Outcomes": true,
		"Pheromones": true, "Territory & Notes": true, "Memory, Findings & Gates": true,
		"Elapsed & Reported Cost": true, "Recent History": true, "Open Items": true, "Next Up": true,
	}
	lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")
	for index, line := range lines {
		if index == 0 || headings[line] {
			lines[index] = "\x1b[96m" + line + "\x1b[0m"
		}
	}
	return strings.Join(lines, "\n") + "\n"
}
