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

// lifecycleStatusColonyTitle and lifecycleStatusPheromoneTitle/NoSignals are
// the reworded section headings CLAUDE.md's own vocabulary table requires:
// the plain-English word leads, the repo-invented word this project coined
// follows in parentheses in the same sentence -- never deleted, never a
// separate glossary line.
const (
	lifecycleStatusColonyTitle     = "This Project (Colony)"
	lifecycleStatusPheromoneTitle  = "Standing Instructions (Pheromones)"
	lifecycleStatusNoPheromoneLine = "No active steering notes (pheromones) are recorded"
)

func renderLifecycleStatusFull(projection LifecycleProjection, width int) string {
	lines := []string{"━━ 🐜 C O L O N Y   S T A T U S ━━"}
	identity := projection.Identity.Value
	name := emptyFallback(strings.TrimSpace(identity.Name), "Unnamed colony")
	goal := emptyFallback(strings.TrimSpace(projection.Goal.Value), "Not recorded")
	standing := emptyFallback(strings.TrimSpace(projection.Standing.Value), "UNKNOWN")

	lines = append(lines,
		"",
		voiceLine("colony", lifecycleStatusColonyTitle),
		voiceLine("colony", "Name: "+name),
		voiceLine("goal", "Goal: "+goal),
		voiceLine("status", "Standing: "+standing),
	)
	if identity.Scope != "" || identity.Mode != "" {
		lines = append(lines, voiceLine("status", fmt.Sprintf("Scope: %s | Mode: %s", emptyFallback(identity.Scope, "Not recorded"), emptyFallback(identity.Mode, "Not recorded"))))
	}

	phase := projection.Phase.Value
	lines = append(lines, "", voiceLine("phase", "Phase & Tasks"))
	if phase.TotalPhases == 0 {
		lines = append(lines, voiceLine("phase", "Phase: Not recorded"))
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
		lines = append(lines, voiceLine("phase", phaseLine))
	}
	lines = append(lines, voiceLine("task", fmt.Sprintf("Tasks: %d/%d complete", phase.CompletedTasks, phase.TotalTasks)))

	lines = append(lines, "", voiceLine("colony", "Ants & Outcomes"))
	active, terminal := lifecycleStatusActors(projection.Actors.Value)
	if len(active) == 0 {
		lines = append(lines, voiceLine("colony", "No ants are active"))
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
		lines = append(lines, voiceLine("colony", fmt.Sprintf("Lineage: %s → %s", lifecycleStatusLineageActorLabel(lineage.Parent), lifecycleStatusLineageActorLabel(lineage.Actor))))
	}

	lines = append(lines, "", voiceLine("focus", lifecycleStatusPheromoneTitle))
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
		line := fmt.Sprintf("%s: %s", emptyFallback(strings.TrimSpace(signal.Type), "Signal"), content)
		lines = append(lines, signalTypeGlyph(signal.Type)+" "+line)
	}
	if activeSignals == 0 {
		lines = append(lines, voiceLine("focus", lifecycleStatusNoPheromoneLine))
	}

	lines = append(lines, "", voiceLine("artifact", "Territory & Notes"))
	research := projection.Research.Value
	lines = append(lines, voiceLine("artifact", fmt.Sprintf("Research documents: %d", len(research.Docs))))
	if latest := lifecycleStatusLatestPath(research.Docs); latest != "" {
		lines = append(lines, voiceLine("artifact", "Latest research: "+latest))
	}
	lines = append(lines, voiceLine("artifact", fmt.Sprintf("Territory records: %d", len(research.Territory))))
	if latest := lifecycleStatusLatestPath(research.Territory); latest != "" {
		lines = append(lines, voiceLine("artifact", "Latest territory record: "+latest))
	}
	lines = append(lines, voiceLine("artifact", fmt.Sprintf("Dreams notes: %d", len(research.Dreams))))
	if latest := lifecycleStatusLatestPath(research.Dreams); latest != "" {
		lines = append(lines, voiceLine("artifact", "Latest Dreams note (Unverified local note): "+latest))
	}

	lines = append(lines, "", voiceLine("learning", "Memory, Findings & Gates"))
	memory := projection.Memory.Value
	openFindings := 0
	for _, finding := range projection.Findings.Value {
		if strings.EqualFold(strings.TrimSpace(finding.Status), "open") {
			openFindings++
		}
	}
	lines = append(lines, voiceLine("learning", fmt.Sprintf("Memory: %d instincts (lessons learned) | %d observations", len(memory.Instincts), len(memory.Observations))))
	lines = append(lines, voiceLine("flag", fmt.Sprintf("Findings: %d open | %d recorded", openFindings, len(projection.Findings.Value))))
	verificationLabel := "Unknown"
	verificationKind := "status"
	if len(projection.Verification) > 0 {
		verificationLabel = "Verified"
		verificationKind = "done"
		for _, gate := range projection.Verification {
			if !gate.Passed {
				verificationLabel = "Blocked"
				verificationKind = "failed"
				break
			}
		}
	}
	lines = append(lines, voiceLine(verificationKind, "Verification: "+verificationLabel))
	for _, gate := range projection.Verification {
		status := "Passed"
		kind := "done"
		if !gate.Passed {
			status = "Failed"
			kind = "failed"
		}
		line := fmt.Sprintf("Gate %s: %s", emptyFallback(strings.TrimSpace(gate.Name), "unnamed"), status)
		if strings.TrimSpace(gate.Detail) != "" {
			line += " — " + strings.TrimSpace(gate.Detail)
		}
		lines = append(lines, voiceLine(kind, line))
	}
	for _, evidence := range projection.Evidence {
		line := "Evidence: " + emptyFallback(strings.TrimSpace(evidence.ID), "unnamed")
		if strings.TrimSpace(evidence.Summary) != "" {
			line += " — " + strings.TrimSpace(evidence.Summary)
		}
		lines = append(lines, voiceLine("evidence", line))
	}

	lines = append(lines, "", voiceLine("elapsed", "Elapsed & Reported Cost"))
	if projection.Elapsed.Source.Provenance == LifecycleFactConfirmed {
		lines = append(lines, voiceLine("elapsed", "Elapsed: "+lifecycleStatusDuration(projection.Elapsed.Value)))
	} else {
		lines = append(lines, voiceLine("elapsed", "Elapsed: Unknown"))
	}
	lines = append(lines, voiceLine("cost", "Reported cost: "+lifecycleStatusReportedCost(projection.ReportedCost)))

	lines = append(lines, "", voiceLine("history", "Recent History"))
	historyLines := lifecycleStatusHistoryLines(projection.History.Value)
	if len(historyLines) == 0 {
		lines = append(lines, voiceLine("history", "No history is recorded"))
	} else {
		lines = append(lines, historyLines...)
	}

	lines = append(lines, "", voiceLine("flag", "Open Items"))
	openCount := 0
	for _, issue := range projection.Warnings {
		openCount++
		lines = append(lines, voiceLine("warning", lifecycleStatusIssueLine("Warning", issue)))
	}
	for _, issue := range projection.Debt {
		openCount++
		lines = append(lines, voiceLine("warning", lifecycleStatusIssueLine("Debt", issue)))
	}
	for _, issue := range projection.Blockers {
		openCount++
		lines = append(lines, voiceLine("blocked", lifecycleStatusIssueLine("Blocker", issue)))
	}
	for _, decision := range projection.OwnerDecisions {
		openCount++
		line := fmt.Sprintf("Owner decision %s: %s", emptyFallback(strings.TrimSpace(decision.ID), "unnamed"), emptyFallback(strings.TrimSpace(decision.Summary), "Reason not recorded"))
		lines = append(lines, voiceLine("question", line))
	}
	if openCount == 0 {
		lines = append(lines, voiceLine("done", "No open items recorded"))
	}

	lines = append(lines, "", voiceLine("next", "Next Up"))
	command := lifecycleStatusActionCommand(projection.NextAction)
	lines = append(lines, voiceLine("next", "Command: "+emptyFallback(command, "Not available")))
	lines = append(lines, voiceLine("status", "Reason: "+emptyFallback(strings.TrimSpace(projection.NextAction.Reason), "Not recorded")))
	for _, choice := range projection.NextAction.Choices {
		lines = append(lines, voiceLine("alternative", "Choice: "+lifecycleStatusChoiceLine(choice)))
	}
	for _, alternative := range projection.Alternatives {
		lines = append(lines, voiceLine("alternative", "Alternative: "+lifecycleStatusChoiceLine(alternative)))
	}

	return lifecycleStatusJoin(lines, width, false)
}

// lifecycleStatusHistoryLines turns the raw stored history slice into
// glyph-led owner-facing sentences via the existing nextActionEventSentence
// reader (cmd/next_action.go) -- never a second parser of the
// `timestamp|event_type|source|message` bookkeeping format. An event whose
// sentence is empty contributes no line; the caller falls back to the
// no-history line only when every event skipped.
func lifecycleStatusHistoryLines(events []string) []string {
	if len(events) == 0 {
		return nil
	}
	start := len(events) - 5
	if start < 0 {
		start = 0
	}
	var lines []string
	for _, event := range events[start:] {
		sentence := nextActionEventSentence(event)
		if sentence == "" {
			continue
		}
		lines = append(lines, voiceLine("history", sentence))
	}
	return lines
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
		voiceLine("colony", fmt.Sprintf("%s — %s", name, standing)),
		voiceLine("goal", "Goal: "+emptyFallback(strings.TrimSpace(projection.Goal.Value), "Not recorded")),
		voiceLine("phase", fmt.Sprintf("Phase %d/%d | Tasks %d/%d", phase.CurrentNumber, phase.TotalPhases, phase.CompletedTasks, phase.TotalTasks)),
	}
	if len(active) == 0 {
		lines = append(lines, voiceLine("colony", "No ants are active"))
	} else {
		named := active[0]
		caste := strings.TrimSpace(named.Caste)
		body := fmt.Sprintf("Active ants: %d | %s", len(active), named.Name)
		if caste != "" {
			lines = append(lines, casteIdentity(caste)+" "+body)
		} else {
			lines = append(lines, voiceLine("colony", body))
		}
	}
	if len(terminal) > 0 {
		latest := terminal[len(terminal)-1]
		outcomeKind := "done"
		if !strings.EqualFold(strings.TrimSpace(latest.Status), "completed") {
			outcomeKind = "failed"
		}
		lines = append(lines, voiceLine(outcomeKind, fmt.Sprintf("Latest outcome: %s — %s", emptyFallback(latest.Name, "Unnamed ant"), emptyFallback(latest.Status, "status unknown"))))
	}
	lines = append(lines,
		voiceLine("artifact", fmt.Sprintf("Territory: %d records | Research: %d docs | Notes: %d unverified", len(research.Territory), len(research.Docs), len(research.Dreams))),
		voiceLine("warning", fmt.Sprintf("Open items: %d | Blockers: %d | Owner decisions: %d", openCount, len(projection.Blockers), len(projection.OwnerDecisions))),
		voiceLine("cost", fmt.Sprintf("Elapsed: %s | Reported cost: %s", lifecycleStatusCompactElapsed(projection.Elapsed), lifecycleStatusReportedCost(projection.ReportedCost))),
		voiceLine("next", "Next Up: "+emptyFallback(lifecycleStatusActionCommand(projection.NextAction), "Not available")),
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

// lifecycleStatusActorLine opens with the helper's own identity -- the exact
// casteIdentity() rendering the live cockpit (cmd/watch_dashboard.go) and the
// spawn list (renderSpawnEntry, cmd/status.go) already use -- rather than a
// bare prefix word followed by a parenthetical caste name. When no caste is
// recorded, it falls back to the generic colony glyph so the line still
// opens with a symbol. Reads only the actor fact it was handed; no file
// read, no second lifecycle derivation.
func lifecycleStatusActorLine(prefix string, actorFact LifecycleActorFact) string {
	name := emptyFallback(strings.TrimSpace(actorFact.Name), "Unnamed ant")
	caste := strings.TrimSpace(actorFact.Caste)
	status := emptyFallback(strings.TrimSpace(actorFact.Status), "status unknown")
	body := fmt.Sprintf("%s: %s", prefix, name)
	body += " — " + status
	if strings.TrimSpace(actorFact.Summary) != "" {
		body += ": " + strings.TrimSpace(actorFact.Summary)
	} else if strings.TrimSpace(actorFact.Task) != "" {
		body += ": " + strings.TrimSpace(actorFact.Task)
	}
	if caste != "" {
		return casteIdentity(caste) + " " + body
	}
	return voiceLine("colony", body)
}

// lifecycleStatusLineageActorLabel translates the one repo-invented actor
// name a lineage row can carry -- "Queen", the root of every spawn tree
// (pkg/agent/spawn.go's spawnRootParentNames) -- into CLAUDE.md's own
// vocabulary shape: the ordinary word leads, the repo's name for it follows
// in parentheses in the same line. Every other actor name (a helper's own
// generated name) is not a repo-invented word and passes through unchanged.
func lifecycleStatusLineageActorLabel(name string) string {
	if strings.EqualFold(strings.TrimSpace(name), "Queen") {
		return "Coordinator (Queen)"
	}
	return name
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

// lifecycleStatusGlyphTokens splits line into printable units for width
// fitting: a lone rune, or a base rune immediately followed by a variation
// selector (U+FE0F, the modifier several voice/caste glyphs carry -- e.g.
// "⏱️", "👁️"). Every fit or truncation below operates on these tokens, never
// on raw runes, so a glyph and its selector are always kept or dropped
// together -- never split mid-sequence.
func lifecycleStatusGlyphTokens(line string) []string {
	runes := []rune(line)
	tokens := make([]string, 0, len(runes))
	for i := 0; i < len(runes); i++ {
		if i+1 < len(runes) && runes[i+1] == '️' {
			tokens = append(tokens, string(runes[i])+string(runes[i+1]))
			i++
			continue
		}
		tokens = append(tokens, string(runes[i]))
	}
	return tokens
}

func lifecycleStatusFitLine(line string, width int) string {
	if width < 1 || utf8.RuneCountInString(line) <= width {
		return line
	}
	tokens := lifecycleStatusGlyphTokens(line)
	if len(tokens) <= width {
		return line
	}
	if width < 5 {
		if width > len(tokens) {
			width = len(tokens)
		}
		return strings.Join(tokens[:width], "")
	}
	left := (width - 1) / 2
	right := width - 1 - left
	if left > len(tokens) {
		left = len(tokens)
	}
	if right > len(tokens)-left {
		right = len(tokens) - left
	}
	return strings.Join(tokens[:left], "") + "…" + strings.Join(tokens[len(tokens)-right:], "")
}

func colorLifecycleStatus(output string) string {
	if !shouldUseANSIColors() {
		return output
	}
	headings := map[string]bool{
		voiceLine("colony", lifecycleStatusColonyTitle):   true,
		voiceLine("phase", "Phase & Tasks"):               true,
		voiceLine("colony", "Ants & Outcomes"):            true,
		voiceLine("focus", lifecycleStatusPheromoneTitle): true,
		voiceLine("artifact", "Territory & Notes"):        true,
		voiceLine("learning", "Memory, Findings & Gates"): true,
		voiceLine("elapsed", "Elapsed & Reported Cost"):   true,
		voiceLine("history", "Recent History"):            true,
		voiceLine("flag", "Open Items"):                   true,
		voiceLine("next", "Next Up"):                      true,
	}
	lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")
	for index, line := range lines {
		if index == 0 || headings[line] {
			lines[index] = "\x1b[96m" + line + "\x1b[0m"
		}
	}
	return strings.Join(lines, "\n") + "\n"
}
