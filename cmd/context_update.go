package cmd

import (
	"fmt"
	"strings"
	"time"
)

const contextFileName = "CONTEXT.md"

// runContextSubAction dispatches to the appropriate sub-action handler based on args[0].
func runContextSubAction(args []string) error {
	action := args[0]
	rest := args[1:]

	switch action {
	case "init":
		return runContextInit(rest)
	case "build-start":
		return runContextBuildStart(rest)
	case "build-progress":
		return runContextBuildProgress(rest)
	case "build-complete":
		return runContextBuildComplete(rest)
	case "worker-spawn":
		return runContextWorkerSpawn(rest)
	case "worker-complete":
		return runContextWorkerComplete(rest)
	case "activity":
		return runContextActivity(rest)
	case "update-phase":
		return runContextUpdatePhase(rest)
	case "decision":
		return runContextDecision(rest)
	case "safe-to-clear":
		return runContextSafeToClear(rest)
	default:
		outputError(1, fmt.Sprintf("unknown context-update action: %q. Use one of: init, build-start, build-progress, build-complete, worker-spawn, worker-complete, activity, update-phase, decision, safe-to-clear", action), nil)
		return nil
	}
}

// runContextInit creates an initial CONTEXT.md file.
func runContextInit(args []string) error {
	goal := ""
	if len(args) > 0 {
		goal = args[0]
	}

	ts := time.Now().UTC().Format(time.RFC3339)

	content := `# Aether Colony — Current Context

> **This document is the colony's memory. If context collapses, read this file first.**

---

## System Status

| Field | Value |
|-------|-------|
| **Last Updated** | ` + ts + ` |
| **Current Phase** | 1 |
| **Phase Name** | initialization |
| **Milestone** | First Mound |
| **Colony Status** | initializing |
| **Safe to Clear?** | NO — Colony just initialized |

---

## Current Goal

` + goal + `

---

## What's In Progress

Colony initialization in progress...

---

## Active Constraints (REDIRECT Signals)

| Constraint | Source | Date Set |
|------------|--------|----------|
| In the Aether repo, ` + "`" + `.aether/` + "`" + ` IS the source of truth — published directly via npm (private dirs excluded by .npmignore) | CLAUDE.md | Permanent |
| Never push without explicit user approval | CLAUDE.md Safety | Permanent |

---

## Active Pheromones (FOCUS Signals)

*None active*

---

## Recent Decisions

| Date | Decision | Rationale | Made By |
|------|----------|-----------|---------|

---

## Recent Activity (Last 10 Actions)

| Timestamp | Command | Result | Files Changed |
|-----------|---------|--------|---------------|
| ` + ts + ` | init | Colony initialized | — |

---

## Next Steps

1. Run ` + "`" + `/ant:plan` + "`" + ` to generate phases for the goal
2. Run ` + "`" + `/ant:build 1` + "`" + ` to start building

---

## If Context Collapses

**READ THIS SECTION FIRST**

### Immediate Recovery

1. **Read this file** — You're looking at it. Good.
2. **Check git status** — ` + "`" + `git status` + "`" + ` and ` + "`" + `git log --oneline -5` + "`" + `
3. **Verify COLONY_STATE.json** — ` + "`" + `cat .aether/data/COLONY_STATE.json | jq .current_phase` + "`" + `
4. **Resume work** — Continue from "Next Steps" above

### What We Were Doing

Colony was just initialized with goal: ` + goal + `

### Is It Safe to Continue?

- Colony is initialized
- No work completed yet
- All state in COLONY_STATE.json

**You can proceed safely.**

---

## Colony Health

` + "```" + `
Milestone:    First Mound   0%
Phase:        1             initializing
Context:      Active        0%
Git Commits:  0
` + "```" + `

---

*This document updates automatically with every ant command. If you see old timestamps, run ` + "`" + `/ant:status` + "`" + ` to refresh.*

**Colony Memory Active**
`

	if err := store.AtomicWrite(contextFileName, []byte(content)); err != nil {
		outputErrorMessage(fmt.Sprintf("failed to write CONTEXT.md: %v", err))
		return nil
	}

	outputOK(map[string]interface{}{
		"updated": true,
		"action":  "init",
	})
	return nil
}

// runContextBuildStart records that a phase build has started.
func runContextBuildStart(args []string) error {
	if len(args) < 3 {
		outputErrorMessage("build-start requires: <phase_id> <workers> <tasks>")
		return nil
	}

	data, err := store.ReadFile(contextFileName)
	if err != nil {
		outputErrorMessage("CONTEXT.md not found. Run 'context-update init' first.")
		return nil
	}

	phaseID := args[0]
	workers := args[1]
	tasks := args[2]
	ts := time.Now().UTC().Format(time.RFC3339)

	content := string(data)

	// Update timestamp
	content = replaceContextTableRow(content, "Last Updated", ts)

	// Update Safe to Clear
	content = replaceContextTableRow(content, "Safe to Clear?", "NO — Build in progress")

	// Replace "What's In Progress" section content
	content = replaceContextSectionContent(content, "What's In Progress", fmt.Sprintf(
		"**Phase %s Build IN PROGRESS**\n- Workers: %s | Tasks: %s\n- Started: %s",
		phaseID, workers, tasks, ts,
	))

	if err := store.AtomicWrite(contextFileName, []byte(content)); err != nil {
		outputErrorMessage(fmt.Sprintf("failed to write CONTEXT.md: %v", err))
		return nil
	}

	outputOK(map[string]interface{}{
		"updated": true,
		"action":  "build-start",
		"workers": workers,
	})
	return nil
}

// runContextBuildProgress updates build progress percentage.
func runContextBuildProgress(args []string) error {
	if len(args) < 2 {
		outputErrorMessage("build-progress requires: <completed> <total>")
		return nil
	}

	data, err := store.ReadFile(contextFileName)
	if err != nil {
		outputErrorMessage("CONTEXT.md not found. Run 'context-update init' first.")
		return nil
	}

	var completed, total int
	if _, err := fmt.Sscanf(args[0], "%d", &completed); err != nil {
		outputErrorMessage(fmt.Sprintf("invalid completed value: %q", args[0]))
		return nil
	}
	if _, err := fmt.Sscanf(args[1], "%d", &total); err != nil {
		outputErrorMessage(fmt.Sprintf("invalid total value: %q", args[1]))
		return nil
	}
	if total <= 0 {
		total = 1
	}

	percent := (completed * 100) / total

	content := string(data)
	content = strings.Replace(content, "Build IN PROGRESS", fmt.Sprintf("Build IN PROGRESS (%d%% complete)", percent), 1)

	if err := store.AtomicWrite(contextFileName, []byte(content)); err != nil {
		outputErrorMessage(fmt.Sprintf("failed to write CONTEXT.md: %v", err))
		return nil
	}

	outputOK(map[string]interface{}{
		"updated": true,
		"action":  "build-progress",
		"percent": percent,
	})
	return nil
}

// runContextBuildComplete marks a build as completed.
func runContextBuildComplete(args []string) error {
	if len(args) < 1 {
		outputErrorMessage("build-complete requires: <status> [result]")
		return nil
	}

	data, err := store.ReadFile(contextFileName)
	if err != nil {
		outputErrorMessage("CONTEXT.md not found. Run 'context-update init' first.")
		return nil
	}

	status := args[0]
	result := "success"
	if len(args) > 1 {
		result = args[1]
	}
	ts := time.Now().UTC().Format(time.RFC3339)

	content := string(data)

	// Update timestamp
	content = replaceContextTableRow(content, "Last Updated", ts)

	// Replace the "What's In Progress" section: remove IN PROGRESS, add completed status
	content = replaceBuildInProgressWithComplete(content, status, result)

	// Update Safe to Clear
	content = replaceContextTableRow(content, "Safe to Clear?", fmt.Sprintf("YES — Build %s", status))

	if err := store.AtomicWrite(contextFileName, []byte(content)); err != nil {
		outputErrorMessage(fmt.Sprintf("failed to write CONTEXT.md: %v", err))
		return nil
	}

	outputOK(map[string]interface{}{
		"updated": true,
		"action":  "build-complete",
		"status":  status,
	})
	return nil
}

// runContextWorkerSpawn records a worker spawn in CONTEXT.md.
func runContextWorkerSpawn(args []string) error {
	if len(args) < 3 {
		outputErrorMessage("worker-spawn requires: <ant_name> <caste> <task>")
		return nil
	}

	data, err := store.ReadFile(contextFileName)
	if err != nil {
		outputErrorMessage("CONTEXT.md not found. Run 'context-update init' first.")
		return nil
	}

	antName := args[0]
	caste := args[1]
	task := args[2]
	ts := time.Now().UTC().Format(time.RFC3339)

	content := string(data)
	content = appendWorkerSpawnEntry(content, antName, caste, task, ts)

	if err := store.AtomicWrite(contextFileName, []byte(content)); err != nil {
		outputErrorMessage(fmt.Sprintf("failed to write CONTEXT.md: %v", err))
		return nil
	}

	outputOK(map[string]interface{}{
		"updated": true,
		"action":  "worker-spawn",
		"ant":     antName,
	})
	return nil
}

// runContextWorkerComplete marks a worker as completed in CONTEXT.md.
func runContextWorkerComplete(args []string) error {
	if len(args) < 1 {
		outputErrorMessage("worker-complete requires: <ant_name> [status]")
		return nil
	}

	data, err := store.ReadFile(contextFileName)
	if err != nil {
		outputErrorMessage("CONTEXT.md not found. Run 'context-update init' first.")
		return nil
	}

	antName := args[0]
	status := "completed"
	if len(args) > 1 {
		status = args[1]
	}
	ts := time.Now().UTC().Format(time.RFC3339)

	content := string(data)
	content = markWorkerComplete(content, antName, status, ts)

	if err := store.AtomicWrite(contextFileName, []byte(content)); err != nil {
		outputErrorMessage(fmt.Sprintf("failed to write CONTEXT.md: %v", err))
		return nil
	}

	outputOK(map[string]interface{}{
		"updated": true,
		"action":  "worker-complete",
		"ant":     antName,
	})
	return nil
}

// runContextActivity appends an entry to the "Recent Activity" table in CONTEXT.md.
func runContextActivity(args []string) error {
	if len(args) < 3 {
		outputErrorMessage("activity requires: <command> <status> <detail>")
		return nil
	}

	data, err := store.ReadFile(contextFileName)
	if err != nil {
		outputErrorMessage("CONTEXT.md not found. Run 'context-update init' first.")
		return nil
	}

	command := args[0]
	status := args[1]
	detail := args[2]
	ts := time.Now().UTC().Format(time.RFC3339)

	content := string(data)
	content = appendActivityEntry(content, ts, command, status, detail)

	if err := store.AtomicWrite(contextFileName, []byte(content)); err != nil {
		outputErrorMessage(fmt.Sprintf("failed to write CONTEXT.md: %v", err))
		return nil
	}

	outputOK(map[string]interface{}{
		"updated": true,
		"action":  "activity",
	})
	return nil
}

// runContextUpdatePhase updates the Current Phase, Phase Name, and Safe to Clear fields.
func runContextUpdatePhase(args []string) error {
	if len(args) < 4 {
		outputErrorMessage("update-phase requires: <phase_id> <phase_name> <safe_to_clear> <note>")
		return nil
	}

	data, err := store.ReadFile(contextFileName)
	if err != nil {
		outputErrorMessage("CONTEXT.md not found. Run 'context-update init' first.")
		return nil
	}

	phaseID := args[0]
	phaseName := args[1]
	safeToClear := args[2]
	note := args[3]

	content := string(data)
	content = replaceContextTableRow(content, "Last Updated", time.Now().UTC().Format(time.RFC3339))
	content = replaceContextTableRow(content, "Current Phase", phaseID)
	content = replaceContextTableRow(content, "Phase Name", phaseName)
	content = replaceContextTableRow(content, "Safe to Clear?", fmt.Sprintf("%s — %s", safeToClear, note))

	if err := store.AtomicWrite(contextFileName, []byte(content)); err != nil {
		outputErrorMessage(fmt.Sprintf("failed to write CONTEXT.md: %v", err))
		return nil
	}

	outputOK(map[string]interface{}{
		"updated":  true,
		"action":   "update-phase",
		"phase_id": phaseID,
	})
	return nil
}

// runContextDecision appends an entry to the "Recent Decisions" table in CONTEXT.md.
func runContextDecision(args []string) error {
	if len(args) < 3 {
		outputErrorMessage("decision requires: <description> <rationale> <made_by>")
		return nil
	}

	data, err := store.ReadFile(contextFileName)
	if err != nil {
		outputErrorMessage("CONTEXT.md not found. Run 'context-update init' first.")
		return nil
	}

	description := args[0]
	rationale := args[1]
	madeBy := args[2]
	ts := time.Now().UTC().Format("2006-01-02")

	content := string(data)
	content = appendDecisionEntry(content, ts, description, rationale, madeBy)

	if err := store.AtomicWrite(contextFileName, []byte(content)); err != nil {
		outputErrorMessage(fmt.Sprintf("failed to write CONTEXT.md: %v", err))
		return nil
	}

	outputOK(map[string]interface{}{
		"updated": true,
		"action":  "decision",
	})
	return nil
}

// runContextSafeToClear updates the "Safe to Clear?" field in CONTEXT.md.
func runContextSafeToClear(args []string) error {
	if len(args) < 1 {
		outputErrorMessage("safe-to-clear requires: <yes_or_no> [note]")
		return nil
	}

	data, err := store.ReadFile(contextFileName)
	if err != nil {
		outputErrorMessage("CONTEXT.md not found. Run 'context-update init' first.")
		return nil
	}

	value := args[0]
	note := ""
	if len(args) > 1 {
		note = args[1]
	}

	var displayValue string
	if note != "" {
		displayValue = fmt.Sprintf("%s — %s", value, note)
	} else {
		displayValue = value
	}

	content := string(data)
	content = replaceContextTableRow(content, "Safe to Clear?", displayValue)

	if err := store.AtomicWrite(contextFileName, []byte(content)); err != nil {
		outputErrorMessage(fmt.Sprintf("failed to write CONTEXT.md: %v", err))
		return nil
	}

	outputOK(map[string]interface{}{
		"updated": true,
		"action":  "safe-to-clear",
	})
	return nil
}

// runContextSectionUpdate updates a constraint/signal section in CONTEXT.md.
// This handles the --section/--key/--content flags used by focus/redirect/feedback commands.
func runContextSectionUpdate(section, key, contentText string, args []string) error {
	if key == "" || contentText == "" {
		outputErrorMessage("--section requires both --key and --content flags")
		return nil
	}

	data, err := store.ReadFile(contextFileName)
	if err != nil {
		outputErrorMessage("CONTEXT.md not found. Run 'context-update init' first.")
		return nil
	}

	ts := time.Now().UTC().Format("2006-01-02")
	content := string(data)

	switch section {
	case "constraint":
		content = appendConstraintEntry(content, ts, contentText, "user", strings.ToUpper(key))
	default:
		content = appendConstraintEntry(content, ts, contentText, "user", strings.ToUpper(key))
	}

	if err := store.AtomicWrite(contextFileName, []byte(content)); err != nil {
		outputErrorMessage(fmt.Sprintf("failed to write CONTEXT.md: %v", err))
		return nil
	}

	outputOK(map[string]interface{}{
		"updated": true,
		"section": section,
		"key":     key,
	})
	return nil
}

// appendConstraintEntry appends a row to the Active Constraints table in CONTEXT.md.
func appendConstraintEntry(content, date, constraintText, source, signalType string) string {
	lines := strings.Split(content, "\n")

	// Find the last row in the "Active Constraints" table
	var lastTableRowIdx = -1
	inConstraints := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "Active Constraints") && strings.HasPrefix(line, "## ") {
			inConstraints = true
			continue
		}
		if inConstraints && (strings.HasPrefix(trimmed, "## ") || trimmed == "---") {
			break
		}
		if inConstraints && strings.HasPrefix(trimmed, "| ") && !strings.HasPrefix(trimmed, "| Constraint") && !strings.HasPrefix(trimmed, "|---") {
			lastTableRowIdx = i
		}
	}

	if lastTableRowIdx >= 0 {
		newRow := fmt.Sprintf("| %s | %s | %s |", constraintText, source, date)
		result := make([]string, 0, len(lines)+1)
		for i, line := range lines {
			result = append(result, line)
			if i == lastTableRowIdx {
				result = append(result, newRow)
			}
		}
		return strings.Join(result, "\n")
	}

	return content
}

// --- CONTEXT.md manipulation helpers ---

// replaceContextTableRow replaces the value in a markdown table row matching "| **fieldName** | ... |".
func replaceContextTableRow(content, fieldName, newValue string) string {
	marker := fmt.Sprintf("| **%s** |", fieldName)
	idx := strings.Index(content, marker)
	if idx == -1 {
		return content
	}

	// Find the end of this line
	lineEnd := strings.Index(content[idx:], "\n")
	if lineEnd == -1 {
		lineEnd = len(content[idx:])
	}
	lineEnd += idx

	oldLine := content[idx:lineEnd]
	newLine := fmt.Sprintf("| **%s** | %s |", fieldName, newValue)
	return strings.Replace(content, oldLine, newLine, 1)
}

// replaceContextSectionContent replaces everything between a section header and the next
// "## " header (or "---" separator) with newContent.
func replaceContextSectionContent(content, sectionName, newContent string) string {
	lines := strings.Split(content, "\n")

	// Find the section header line
	headerIdx := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "## ") && strings.Contains(line, sectionName) {
			headerIdx = i
			break
		}
	}
	if headerIdx == -1 {
		return content
	}

	// Find the start of the next section (## or ---)
	nextSectionIdx := len(lines)
	for i := headerIdx + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "## ") || trimmed == "---" {
			nextSectionIdx = i
			break
		}
	}

	// Build replacement
	newSection := []string{lines[headerIdx], "", newContent}
	result := make([]string, 0, headerIdx+len(newSection)+(len(lines)-nextSectionIdx))
	result = append(result, lines[:headerIdx]...)
	result = append(result, newSection...)
	result = append(result, lines[nextSectionIdx:]...)

	return strings.Join(result, "\n")
}

// replaceBuildInProgressWithComplete replaces the "What's In Progress" section,
// swapping out any "Build IN PROGRESS" lines for a completion message.
func replaceBuildInProgressWithComplete(content, status, result string) string {
	lines := strings.Split(content, "\n")

	var inProgress bool
	var newLines []string
	skipSection := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.Contains(trimmed, "What's In Progress") && strings.HasPrefix(line, "## ") {
			inProgress = true
			newLines = append(newLines, line)
			newLines = append(newLines, "")
			newLines = append(newLines, fmt.Sprintf("**Build %s** — %s", status, result))
			skipSection = true
			continue
		}

		if inProgress && (strings.HasPrefix(trimmed, "## ") || trimmed == "---") {
			inProgress = false
			skipSection = false
		}

		if skipSection {
			continue
		}

		newLines = append(newLines, line)
	}

	return strings.Join(newLines, "\n")
}

// appendWorkerSpawnEntry appends a spawn entry after the "Workers:" line in the
// "What's In Progress" section.
func appendWorkerSpawnEntry(content, antName, caste, task, ts string) string {
	lines := strings.Split(content, "\n")
	var inProgress bool
	var result []string

	for _, line := range lines {
		result = append(result, line)

		trimmed := strings.TrimSpace(line)

		if strings.Contains(trimmed, "What's In Progress") && strings.HasPrefix(line, "## ") {
			inProgress = true
			continue
		}

		if inProgress && strings.HasPrefix(trimmed, "## ") {
			inProgress = false
		}

		if inProgress && strings.Contains(line, "Workers:") {
			result = append(result, fmt.Sprintf("  - %s: Spawned %s (%s) for: %s", ts, antName, caste, task))
		}
	}

	return strings.Join(result, "\n")
}

// markWorkerComplete replaces a worker's spawn line with a completion line.
func markWorkerComplete(content, antName, status, ts string) string {
	lines := strings.Split(content, "\n")
	var result []string

	for _, line := range lines {
		if strings.Contains(line, antName) && strings.Contains(line, "Spawned") {
			result = append(result, fmt.Sprintf("  - %s: %s (updated %s)", antName, status, ts))
			continue
		}
		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// appendActivityEntry appends a row to the "Recent Activity" table in CONTEXT.md.
func appendActivityEntry(content, ts, command, status, detail string) string {
	lines := strings.Split(content, "\n")

	// Find the last row (including separator) in the "Recent Activity" table
	var insertAfterIdx = -1
	inActivity := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "Recent Activity") && strings.HasPrefix(line, "## ") {
			inActivity = true
			continue
		}
		if inActivity && (strings.HasPrefix(trimmed, "## ") || trimmed == "---") {
			break
		}
		if inActivity && strings.HasPrefix(trimmed, "| ") {
			insertAfterIdx = i
		}
	}

	if insertAfterIdx >= 0 {
		newRow := fmt.Sprintf("| %s | %s | %s | %s |", ts, command, status, detail)
		result := make([]string, 0, len(lines)+1)
		for i, line := range lines {
			result = append(result, line)
			if i == insertAfterIdx {
				result = append(result, newRow)
			}
		}
		return strings.Join(result, "\n")
	}

	return content
}

// appendDecisionEntry appends a row to the "Recent Decisions" table in CONTEXT.md.
func appendDecisionEntry(content, date, description, rationale, madeBy string) string {
	lines := strings.Split(content, "\n")

	// Find the last row (including separator) in the "Recent Decisions" table
	var insertAfterIdx = -1
	inDecisions := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "Recent Decisions") && strings.HasPrefix(line, "## ") {
			inDecisions = true
			continue
		}
		if inDecisions && (strings.HasPrefix(trimmed, "## ") || trimmed == "---") {
			break
		}
		if inDecisions && strings.HasPrefix(trimmed, "| ") {
			insertAfterIdx = i
		}
	}

	if insertAfterIdx >= 0 {
		newRow := fmt.Sprintf("| %s | %s | %s | %s |", date, description, rationale, madeBy)
		result := make([]string, 0, len(lines)+1)
		for i, line := range lines {
			result = append(result, line)
			if i == insertAfterIdx {
				result = append(result, newRow)
			}
		}
		return strings.Join(result, "\n")
	}

	return content
}
