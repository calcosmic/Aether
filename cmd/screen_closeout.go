package cmd

import "strings"

// Append the run's own warning/recommendation before the shared closing card.
// The full closeout remains in the result and the phase/status detail views.
func appendScreenCloseout(body string, closeout LifecycleCloseout, result map[string]interface{}, platform string) string {
	var detail strings.Builder
	var lines []string
	if closeout.WorkOutcome != nil {
		lines = append(lines, voiceLine("evidence", closeout.WhatHappened.Summary))
	}
	for _, issue := range closeout.Unresolved.Blockers {
		lines = append(lines, voiceLine("blocked", issue.Summary))
	}
	for _, decision := range closeout.Unresolved.Decisions {
		lines = append(lines, voiceLine("decision", decision.Summary))
	}
	var unavailable []string
	for _, issue := range closeout.Unresolved.Warnings {
		if strings.HasPrefix(issue.ID, "fact-") {
			unavailable = append(unavailable, strings.ReplaceAll(strings.TrimPrefix(issue.ID, "fact-"), "-", " "))
			continue
		}
		lines = append(lines, voiceLine("warning", issue.Summary))
	}
	if len(unavailable) > 0 {
		lines = append(lines, voiceLine("warning", "Some project details could not be read ("+strings.Join(unavailable, ", ")+"). Review `aether status --detail`."))
	}
	var debt []string
	for _, issue := range closeout.Unresolved.Debt {
		debt = append(debt, voiceLine("warning", issue.Summary))
	}
	lines = append(lines, capScreenLines(debt, 3)...)
	writeScreenSection(&detail, commandEmoji("flags"), "Run Review", lines)
	body = insertBeforeScreenClosing(body, detail.String())
	if !strings.Contains(body, renderBanner(commandEmoji("status"), "What Next")) {
		if answer, ok := nextActionFromResult(result); ok {
			body += "\n" + renderNextActionCardForPlatform(answer, platform)
		}
	}
	return appendLifecycleCloseoutSpendLine(body, closeout, result)
}

func insertBeforeScreenClosing(body, section string) string {
	if strings.TrimSpace(section) == "" {
		return body
	}
	marker := renderBanner(commandEmoji("status"), "What Next")
	if index := strings.LastIndex(body, marker); index >= 0 {
		return strings.TrimRight(body[:index], "\n") + "\n\n" + strings.Trim(section, "\n") + "\n\n" + body[index:]
	}
	return strings.TrimRight(body, "\n") + "\n\n" + strings.Trim(section, "\n") + "\n"
}

func appendBuildScreenAdvisory(body string, result map[string]interface{}) string {
	advisory, ok := buildAdvisoryFromResult(result)
	if !ok || len(advisory.Signals) == 0 {
		return body
	}
	var section strings.Builder
	var lines []string
	for _, signal := range advisory.Signals {
		lines = append(lines, voiceLine("warning", signal.Reason))
	}
	if advisory.Ask {
		lines = append(lines, voiceLine("question", buildBlockerAdvisoryQuestion))
	}
	writeScreenSection(&section, commandEmoji("flags"), "Needs You", lines)
	return insertBeforeScreenClosing(body, section.String())
}
