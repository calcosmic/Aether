package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

// The check screen's sections (owner's layout, 2026-09-27). Every function
// here only reads the continue result the screen already had -- nothing
// decides anything new, and the JSON result is untouched.

const (
	checkScreenMaxProven       = 4
	checkScreenMaxSafetyItems  = 3
	checkScreenMaxBlockers     = 5
	checkScreenMaxOperational  = 3
	checkScreenCheckLabelWidth = 18
)

// checkScreenSectionEmoji names the header glyph of each check-screen
// section, read from the one shared command-glyph table.
func checkScreenSectionEmoji(section string) string {
	return commandEmoji("section-" + section)
}

// verificationStepViewsFromValue reduces result["verification"] (typed or
// JSON-round-tripped) to its steps.
func verificationStepViewsFromValue(raw interface{}) []verificationStepDetailView {
	switch v := raw.(type) {
	case codexContinueVerificationReport:
		return verificationStepDetailViewsFromTyped(v.Steps)
	case map[string]interface{}:
		steps, _ := v["steps"].([]interface{})
		return verificationStepDetailViewsFromMap(steps)
	}
	return nil
}

// gateCheckViewsFromValue reduces result["gates"] (typed or JSON-round-
// tripped) to its checks.
func gateCheckViewsFromValue(raw interface{}) []gateCheckDetailView {
	switch v := raw.(type) {
	case codexContinueGateReport:
		views := make([]gateCheckDetailView, 0, len(v.Checks))
		for _, c := range v.Checks {
			views = append(views, gateCheckDetailView{Name: c.Name, Passed: c.Passed, FixHint: c.FixHint})
		}
		return views
	case map[string]interface{}:
		checks, _ := v["checks"].([]interface{})
		views := make([]gateCheckDetailView, 0, len(checks))
		for _, raw := range checks {
			entry, _ := raw.(map[string]interface{})
			if entry == nil {
				continue
			}
			views = append(views, gateCheckDetailView{
				Name:    stringValue(entry["name"]),
				Passed:  boolValue(entry["passed"]),
				FixHint: stringValue(entry["fix_hint"]),
			})
		}
		return views
	}
	return nil
}

func criterionViewsFromValue(raw interface{}) []criterionEvidenceView {
	switch v := raw.(type) {
	case codexContinueVerificationReport:
		return criterionEvidenceViewsFromTyped(v.Criteria)
	case map[string]interface{}:
		criteria, _ := v["criteria"].([]interface{})
		return criterionEvidenceViewsFromMap(criteria)
	}
	return nil
}

// checkScreenFlowStep is the little a worker-flow entry needs to say who a
// helper was and how it ended.
type checkScreenFlowStep struct {
	Stage   string
	Caste   string
	Name    string
	Status  string
	Summary string
	// Measured is a review helper's measured time and tool calls, in the
	// one shared wording (workerMeasurementFigures); empty for any other
	// step, which was never measured.
	Measured string
}

func checkScreenFlowSteps(raw interface{}) []checkScreenFlowStep {
	var steps []checkScreenFlowStep
	switch flow := raw.(type) {
	case []codexContinueWorkerFlowStep:
		for _, s := range flow {
			step := checkScreenFlowStep{Stage: s.Stage, Caste: s.Caste, Name: s.Name, Status: s.Status, Summary: s.Summary}
			if s.Stage == "review" && s.Caste != "system" {
				step.Measured = workerMeasurementFigures(s.Duration, s.DurationReported, s.ToolCount, s.ToolCountReported)
			}
			steps = append(steps, step)
		}
	case []interface{}:
		for _, rawStep := range flow {
			s, _ := rawStep.(map[string]interface{})
			if s == nil || strings.TrimSpace(stringValue(s["name"])) == "" {
				continue
			}
			step := checkScreenFlowStep{
				Stage:   stringValue(s["stage"]),
				Caste:   stringValue(s["caste"]),
				Name:    stringValue(s["name"]),
				Status:  stringValue(s["status"]),
				Summary: stringValue(s["summary"]),
			}
			if step.Stage == "review" && step.Caste != "system" {
				step.Measured = workerMeasurementFigures(
					floatValue(s["duration"]), boolValue(s["duration_reported"]),
					intValue(s["tool_count"]), boolValue(s["tool_count_reported"]),
				)
			}
			steps = append(steps, step)
		}
	}
	return steps
}

// checkScreenHelperWentWrong reports whether a helper ended in a way the
// owner should see (anything other than finishing or being skipped).
func checkScreenHelperWentWrong(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", "completed", "complete", "passed", "done", "skipped", "success", "succeeded":
		return false
	}
	return true
}

func stepSkippedForNoCommand(summary string) bool {
	return strings.Contains(strings.ToLower(summary), "no command")
}

// checkScreenCheckLines is the CHECKS section: one line per build/type/lint/
// test check, the checks with no command merged into one "not set up" line,
// the gates as one line when they all passed (each failing gate named when
// not), and any operational trouble the run reported.
func checkScreenCheckLines(result map[string]interface{}) []string {
	var lines []string
	var unset []string
	for _, v := range verificationStepViewsFromValue(result["verification"]) {
		label := verificationStepDisplayName(v.Name)
		switch {
		case v.Skipped && stepSkippedForNoCommand(v.Summary):
			unset = append(unset, label)
		case v.Skipped:
			reason := strings.TrimSpace(v.Summary)
			if reason == "" {
				reason = "skipped"
			}
			lines = append(lines, voiceLine("status", label+" skipped — "+reason))
		case v.Passed:
			text := label + " passed"
			if d := plainDuration(v.Duration); d != "" {
				text = padScreenLabel(text, checkScreenCheckLabelWidth) + "(" + d + ")"
			}
			lines = append(lines, voiceLine("done", text))
		default:
			text := label + " failed"
			if reason := strings.TrimSpace(v.Summary); reason != "" {
				text += " — " + reason
			}
			if cmd := strings.TrimSpace(v.Command); cmd != "" {
				text += " (ran: " + cmd + ")"
			}
			lines = append(lines, voiceLine("failed", text))
		}
	}
	if len(unset) > 0 {
		lines = append(lines, voiceLine("unset", padScreenLabel(strings.Join(unset, ", "), checkScreenCheckLabelWidth)+"not set up"))
	}

	gates := gateCheckViewsFromValue(result["gates"])
	if len(gates) > 0 {
		passed := 0
		var failed []string
		for _, g := range gates {
			if g.Passed {
				passed++
				continue
			}
			failed = append(failed, voiceLine("failed", "Not yet true: "+gateCheckDisplayName(g.Name)))
		}
		kind := "done"
		if len(failed) > 0 {
			kind = "failed"
		}
		lines = append(lines, voiceLine(kind, fmt.Sprintf("Gates: %d/%d passed", passed, len(gates))))
		lines = append(lines, failed...)
	}

	var operational []string
	for _, issue := range stringSliceValue(result["operational_issues"]) {
		if issue = strings.TrimSpace(issue); issue != "" {
			operational = append(operational, voiceLine("warning", issue))
		}
	}
	return append(lines, capScreenLines(operational, checkScreenMaxOperational)...)
}

// checkScreenProvenLines is the WHAT WAS PROVEN section. A requirement only
// ever gets the proven mark when criterionEvidenceDisplayState says it is
// satisfied; anything not yet proven is listed first so the cap never hides
// it behind the proven ones.
func checkScreenProvenLines(raw interface{}) []string {
	views := criterionViewsFromValue(raw)
	if len(views) == 0 {
		return nil
	}
	sort.SliceStable(views, func(i, j int) bool {
		return criterionEvidenceDisplayState(views[i]) != "satisfied" && criterionEvidenceDisplayState(views[j]) == "satisfied"
	})
	lines := make([]string, 0, len(views))
	for _, v := range views {
		label := strings.TrimSpace(v.Criterion)
		if label == "" {
			label = "(unnamed requirement)"
		}
		switch criterionEvidenceDisplayState(v) {
		case "satisfied":
			lines = append(lines, voiceLine("proven", label))
		case "awaiting_owner":
			lines = append(lines, voiceLine("question", label+" — waiting for your confirmation"))
		case "blocked":
			reason := strings.TrimSpace(v.Summary)
			if reason == "" && len(v.BlockingIssues) > 0 {
				reason = strings.Join(v.BlockingIssues, "; ")
			}
			if reason == "" {
				reason = "not met"
			}
			lines = append(lines, voiceLine("failed", label+" — "+reason))
		default:
			lines = append(lines, voiceLine("warning", label+" — no proof recorded yet"))
		}
	}
	return capScreenLines(lines, checkScreenMaxProven)
}

// minorFindingSeverities are the severity words a reviewer uses for a note
// that is not worth leading with; they are dropped from the displayed text.
var minorFindingSeverities = []string{"low", "info", "minor", "medium", "note"}

// checkScreenSafetyLines is the SAFETY REVIEW section: one plain count line,
// then at most three of the reviewers' own notes. It says "Nothing found"
// when reviewers ran and found nothing, and is empty (no section) when no
// reviewer ran and nobody noted anything.
func checkScreenSafetyLines(result map[string]interface{}, blockedScreen bool) []string {
	reviewerRan := false
	for _, step := range checkScreenFlowSteps(result["worker_flow"]) {
		if step.Stage == "review" && step.Caste != "system" {
			reviewerRan = true
			break
		}
	}
	var views []specialistFindingView
	switch flow := result["worker_flow"].(type) {
	case []codexContinueWorkerFlowStep:
		views = specialistFindingViewsFromTyped(flow)
	case []interface{}:
		views = specialistFindingViewsFromMap(flow)
	}
	if !reviewerRan && len(views) == 0 {
		return nil
	}

	var items []string
	blocking := 0
	serious := false
	for _, v := range views {
		for _, f := range v.Findings {
			text := f
			lower := strings.ToLower(f)
			if strings.HasPrefix(lower, "high:") || strings.HasPrefix(lower, "critical:") {
				serious = true
			}
			for _, sev := range minorFindingSeverities {
				if strings.HasPrefix(lower, sev+":") {
					text = strings.TrimSpace(f[len(sev)+1:])
					break
				}
			}
			items = append(items, voiceLine("feedback", text))
		}
		for _, blocker := range v.Blockers {
			if blocker = strings.TrimSpace(blocker); blocker != "" {
				blocking++
				items = append(items, voiceLine("blocked", blocker))
			}
		}
		for _, group := range [][]string{v.WeakSpots, v.Recommendations, v.EdgeCases} {
			for _, item := range group {
				if item = strings.TrimSpace(item); item != "" {
					items = append(items, voiceLine("feedback", item))
				}
			}
		}
	}
	if len(items) == 0 {
		return []string{voiceLine("done", "Nothing found")}
	}

	noun := "note"
	if len(items) != 1 {
		noun = "notes"
	}
	adjective := "small "
	if serious || blocking > 0 {
		adjective = ""
	}
	count := fmt.Sprintf("%d %s%s", len(items), adjective, noun)
	kind := "status"
	switch {
	case blocking > 0:
		count += fmt.Sprintf(", %d blocking", blocking)
		kind = "warning"
	case !blockedScreen:
		count += ", nothing blocking"
	}
	return append([]string{voiceLine(kind, count)}, capScreenLines(items, checkScreenMaxSafetyItems)...)
}

// checkScreenHelperLines is the HELPERS section, shown only when a helper
// did not finish cleanly -- a roster of helpers that all finished is noise.
func checkScreenHelperLines(raw interface{}) []string {
	steps := checkScreenFlowSteps(raw)
	wentWrong := false
	for _, step := range steps {
		if checkScreenHelperWentWrong(step.Status) {
			wentWrong = true
			break
		}
	}
	if !wentWrong {
		return nil
	}
	lines := make([]string, 0, len(steps))
	for _, step := range steps {
		line := strings.TrimSpace(step.Name)
		if caste := strings.TrimSpace(step.Caste); caste != "" {
			line = casteIdentity(caste) + " " + line
		}
		if status := strings.TrimSpace(step.Status); status != "" {
			line += " " + status
		}
		if summary := strings.TrimSpace(step.Summary); summary != "" {
			line += " — " + summary
		}
		if step.Measured != "" {
			line += " (" + step.Measured + ")"
		}
		lines = append(lines, line)
	}
	return lines
}

// checkScreenBehindTheScenesLines groups the housekeeping, learning,
// improvement and steering beats, the phase-end footer, and the one pointer
// to the full detail the default screen no longer lists.
func checkScreenBehindTheScenesLines(state colony.ColonyState, phase colony.Phase, housekeeping *signalHousekeepingResult, result map[string]interface{}, withFooter bool) []string {
	var lines []string
	if housekeeping != nil {
		line := fmt.Sprintf("Steering notes: %d active", housekeeping.ActiveAfter)
		if expired := housekeeping.ExpiredByTime + housekeeping.DeactivatedByStrength + housekeeping.ExpiredWorkerContinue; housekeeping.Updated > 0 && expired > 0 {
			line = fmt.Sprintf("Steering notes: %d expired, %d still active", expired, housekeeping.ActiveAfter)
		}
		lines = append(lines, voiceLine("focus", line))
	}
	if closed := len(stringSliceValue(result["closed_workers"])); closed > 0 {
		noun := "helpers"
		if closed == 1 {
			noun = "helper"
		}
		lines = append(lines, voiceLine("colony", fmt.Sprintf("%d %s finished their work", closed, noun)))
	}
	lines = append(lines, checkScreenLessonLine(result["consolidation"]))
	lines = append(lines, improvementPassBeatLines(result["improvement_pass"])...)
	if state.PendingSuggestions != nil {
		if active := filterActiveSuggestions(state.PendingSuggestions); len(active) > 0 {
			noun := "suggestion"
			if len(active) != 1 {
				noun = "suggestions"
			}
			lines = append(lines, voiceLine("feedback", fmt.Sprintf("%d steering %s — nothing is written until you approve it:", len(active), noun)))
			for i, s := range active {
				lines = append(lines, fmt.Sprintf("  %d. %s [%s] %s — adopt: `aether suggest-approve --approve %s`",
					i+1, signalTypeGlyph(s.Type), strings.ToUpper(strings.TrimSpace(s.Type)), strings.TrimSpace(s.Content), s.ID))
			}
		}
	}
	if withFooter {
		for _, line := range strings.Split(strings.TrimRight(renderPhaseEndFooter(state, phase.ID), "\n"), "\n") {
			// The "no steering notes -- run aether focus ..." hint is advice
			// for before a build, not news at the end of a check.
			if strings.TrimSpace(line) != "" && !strings.HasPrefix(strings.TrimSpace(line), "Steering signals: none") {
				lines = append(lines, line)
			}
		}
	}
	lines = append(lines, voiceLine("history", fmt.Sprintf("Full detail: `aether phase %d` (every check, requirement and helper)", phase.ID)))
	return lines
}

// checkScreenFailedStepNames lists the build/type/lint/test checks that
// failed, in plain words, for the blocked verdict.
func checkScreenFailedStepNames(result map[string]interface{}) []string {
	var names []string
	for _, v := range verificationStepViewsFromValue(result["verification"]) {
		if !v.Skipped && !v.Passed {
			names = append(names, strings.ToLower(verificationStepDisplayName(v.Name)))
		}
	}
	return names
}

// checkScreenBlockedReason says in plain words what stopped the sign-off,
// read from the same result fields the blocked screen already shows.
func checkScreenBlockedReason(result map[string]interface{}) string {
	if names := checkScreenFailedStepNames(result); len(names) > 0 {
		return joinPlainList(names) + " failed"
	}
	for _, v := range criterionViewsFromValue(result["verification"]) {
		if state := criterionEvidenceDisplayState(v); state == "blocked" || state == "unproven" {
			return "a requirement is not proven yet"
		}
		if criterionEvidenceDisplayState(v) == "awaiting_owner" {
			return "waiting for your confirmation"
		}
	}
	for _, g := range gateCheckViewsFromValue(result["gates"]) {
		if !g.Passed {
			return "a final check did not pass"
		}
	}
	return "something still needs fixing"
}

func joinPlainList(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	default:
		return strings.Join(items[:len(items)-1], ", ") + " and " + items[len(items)-1]
	}
}

// checkScreenWhatToFixLines is the blocked screen's WHAT TO FIX section:
// the unfinished work written back as tasks, the blocking issues (at most
// five), and the existing way forward.
func checkScreenWhatToFixLines(result map[string]interface{}) []string {
	var lines []string
	if added := intValue(result["recovery_tasks_added"]); added > 0 {
		noun := "piece"
		if added != 1 {
			noun = "pieces"
		}
		lines = append(lines, voiceLine("task", fmt.Sprintf("%d %s of unfinished work were written back onto the phase as tasks so they can be picked up next.", added, noun)))
	}
	var blockers []string
	for _, issue := range stringSliceValue(result["blocking_issues"]) {
		if issue = strings.TrimSpace(issue); issue != "" {
			blockers = append(blockers, voiceLine("blocked", issue))
		}
	}
	lines = append(lines, capScreenLines(blockers, checkScreenMaxBlockers)...)
	// The way forward is always offered, so a stopped check is never a dead end.
	return append(lines, strings.Split(strings.TrimRight(renderBlockedWayForward(continueTypedResultMapValue(result["gates"])), "\n"), "\n")...)
}

// renderContinueVisual draws the successful check screen: the verdict box,
// then CHECKS, WHAT WAS PROVEN, SAFETY REVIEW, HELPERS (only when one went
// wrong) and BEHIND THE SCENES, then the project-complete block or the
// next-phase line, then the shared closing card.
func renderContinueVisual(state colony.ColonyState, phase colony.Phase, housekeeping *signalHousekeepingResult, final bool, nextPhase *colony.Phase, result map[string]interface{}, reviewDepth colony.VerificationDepth) string {
	var b strings.Builder
	partial, _ := result["partial_success"].(bool)
	kind := "done"
	verdict := fmt.Sprintf("PHASE %d CHECKED AND SIGNED OFF", phase.ID)
	subject := ""
	if name := strings.TrimSpace(phase.Name); name != "" {
		subject = " for " + name
	}
	sentence := "Everything passed" + subject + "."
	if reviewDepth == colony.VerificationDepthHeavy {
		sentence = "Everything passed" + subject + ", including the full review panel."
	}
	if partial {
		kind = "warning"
		verdict = fmt.Sprintf("PHASE %d SIGNED OFF WITH WARNINGS", phase.ID)
		sentence = "The checks passed, but something went wrong while it ran — see below."
	}
	if final {
		sentence += " The project is complete."
	}
	b.WriteString(renderVerdictBox(kind, verdict, sentence))

	writeScreenSection(&b, checkScreenSectionEmoji("checks"), "Checks", checkScreenCheckLines(result))
	writeScreenSection(&b, checkScreenSectionEmoji("proven"), "What Was Proven", checkScreenProvenLines(result["verification"]))
	writeScreenSection(&b, checkScreenSectionEmoji("safety"), "Safety Review", checkScreenSafetyLines(result, false))
	writeScreenSection(&b, checkScreenSectionEmoji("helpers"), "Helpers", checkScreenHelperLines(result["worker_flow"]))
	writeScreenSection(&b, checkScreenSectionEmoji("behind"), "Behind The Scenes", checkScreenBehindTheScenesLines(state, phase, housekeeping, result, !final))

	if final {
		b.WriteString("\n")
		b.WriteString(renderProjectComplete(state, len(state.Plan.Phases)))
		b.WriteString("\n\n")
		b.WriteString(voiceLine("milestone", "Every phase in the plan is finished. The project is ready to be signed off as"))
		b.WriteString("\n")
		b.WriteString("complete -- the stage this project calls Crowned Anthill.\n")
		b.WriteString("\n" + renderLifecycleClosingForState(result, state, "continue"))
		return b.String()
	}
	if nextPhase != nil {
		b.WriteString("\n")
		b.WriteString(voiceLine("next", fmt.Sprintf("Next phase ready: %d — %s", nextPhase.ID, nextPhase.Name)))
		b.WriteString("\n")
	}
	b.WriteString("\n" + renderLifecycleClosingForState(result, state, "continue"))
	return b.String()
}

// renderContinueBlockedVisual draws the check screen when the phase was not
// signed off: the verdict box names what stopped it, then WHAT TO FIX, then
// the same sections as the successful screen, then the existing next-step
// card.
func renderContinueBlockedVisual(state colony.ColonyState, phase colony.Phase, result map[string]interface{}, reviewDepth colony.VerificationDepth) string {
	var b strings.Builder
	verdict := fmt.Sprintf("PHASE %d NOT SIGNED OFF — %s", phase.ID, checkScreenBlockedReason(result))
	sentence := fmt.Sprintf("Phase %d stays open: %s.", phase.ID, strings.TrimSpace(phase.Name))
	b.WriteString(renderVerdictBox("blocked", verdict, sentence))

	writeScreenSection(&b, checkScreenSectionEmoji("fix"), "What To Fix", checkScreenWhatToFixLines(result))
	writeScreenSection(&b, checkScreenSectionEmoji("checks"), "Checks", checkScreenCheckLines(result))
	writeScreenSection(&b, checkScreenSectionEmoji("proven"), "What Was Proven", checkScreenProvenLines(result["verification"]))
	writeScreenSection(&b, checkScreenSectionEmoji("safety"), "Safety Review", checkScreenSafetyLines(result, true))
	writeScreenSection(&b, checkScreenSectionEmoji("helpers"), "Helpers", checkScreenHelperLines(result["worker_flow"]))
	// D-11: a still-failing automatic repair's four-part handback, read
	// before the closing next-step card so the owner sees what happened
	// before what to do. Nil for every other outcome.
	if handback := repairHandbackFromVerificationValue(result["verification"]); handback != nil {
		b.WriteString(renderFailedRepairHandback(*handback))
	}
	b.WriteString("\n" + renderLifecycleClosingForState(result, state, "continue"))
	return b.String()
}

// checkScreenLessonLine says in plain words what this phase taught the
// project. It keeps the four learning outcomes distinct -- learned something,
// learned nothing, the learning step failed, or no result recorded -- so a
// failure is never silent or disguised as "nothing new" (D-07), just without
// the classic beat's internal vocabulary.
func checkScreenLessonLine(raw interface{}) string {
	consolidation, ok := raw.(map[string]interface{})
	if !ok || consolidation == nil {
		return voiceLine("learning", "Lessons: no learning result was recorded for this phase")
	}
	if !boolValue(consolidation["ran"]) {
		line := "Lessons: the learning step FAILED this phase"
		if reason := strings.TrimSpace(stringValue(consolidation["reason"])); reason != "" {
			line += " — " + reason
		}
		return voiceLine("warning", line)
	}
	learned := intValue(consolidation["promotion_candidates"])
	reusable := intValue(consolidation["queen_eligible"])
	if learned == 0 && reusable == 0 {
		return voiceLine("learning", "Lessons: nothing new learned this phase")
	}
	return voiceLine("learning", fmt.Sprintf("Lessons: %d new lesson(s), %d strong enough to reuse in every future job", learned, reusable))
}
