package cmd

import (
	"fmt"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// The short dashboard and the detailed dashboard read the same saved facts.
// Neither changes the result map or chooses a different closing action.
func renderDashboard(state colony.ColonyState, s *storage.Store, result map[string]interface{}) string {
	var b strings.Builder
	kind, verdict, sentence := statusScreenVerdict(state, result)
	b.WriteString(renderVerdictBox(kind, verdict, sentence))
	var progress, needs []string
	completed := completedPhaseCount(state)
	if phase := recoveryPhase(&state); phase != nil && phase.Status == colony.PhaseCompleted {
		if truth := resolvePhaseProgressFromDisk(*phase); truth.Disagreed && truth.Status != "complete" {
			completed--
		}
	}
	if total := len(state.Plan.Phases); total > 0 {
		progress = append(progress, voiceLine("phase", fmt.Sprintf("[Phase %d/%d] %s %d%%", completed, total, generateProgressBar(completed, total, 20), completed*100/total)))
	}
	if phase := recoveryPhase(&state); phase != nil && len(phase.Tasks) > 0 {
		done, total := 0, len(phase.Tasks)
		for _, task := range phase.Tasks {
			if task.Status == colony.TaskCompleted {
				done++
			}
		}
		if truth := resolvePhaseProgressFromDisk(*phase); truth.Disagreed {
			done, total = truth.TasksDone, truth.TasksTotal
			needs = append(needs, voiceLine("warning", fmt.Sprintf("The saved project and the last check disagree about phase %d; showing it as %s.", phase.ID, truth.Status)))
		} else if state.State == colony.StateCOMPLETED && phase.Status == colony.PhaseCompleted {
			done = total
		}
		progress = append(progress, voiceLine("task", fmt.Sprintf("[Tasks %d/%d] %s in Phase %d (%s)", done, total, generateProgressBar(done, total, 20), phase.ID, phase.Name)))
	}
	for _, worker := range statusActiveWorkers(s, state) {
		kind, label, _ := screenWorkerStatus(worker.Status)
		progress = append(progress, voiceLine(kind, worker.AgentName+" — "+label))
	}
	writeScreenSection(&b, commandEmoji("phase"), "Progress", capScreenLines(progress, 5))
	needs = append(needs, statusScreenNeedsYou(state, s, result)...)
	writeScreenSection(&b, commandEmoji("flags"), "Needs You", needs)
	if s != nil {
		var steering strings.Builder
		var pf colony.PheromoneFile
		if s.LoadJSON("pheromones.json", &pf) == nil {
			for _, sig := range pf.Signals {
				if sig.Active {
					renderPheromoneSummary(&steering, s)
					break
				}
			}
		}
		writeScreenSection(&b, commandEmoji("focus"), "Steering", capScreenLines(screenTextLines(steering.String()), 4))
	}
	instincts := loadRuntimeInstincts(s, &state)
	var memory []string
	if len(instincts) > 0 {
		strong := 0
		for _, inst := range instincts {
			if inst.Confidence >= .7 {
				strong++
			}
		}
		label := fmt.Sprintf("Lessons: %d learned", len(instincts))
		if strong > 0 {
			label += fmt.Sprintf(" (%d strong)", strong)
		}
		memory = append(memory, voiceLine("learning", label))
	}
	if n := len(state.Memory.PhaseLearnings); n > 0 {
		memory = append(memory, voiceLine("memory", screenCount(n, "phase")+" with saved lessons"))
	}
	writeScreenSection(&b, commandEmoji("memory-details"), "Memory", memory)
	if report := loadAutopilotLastReport(s); report != nil {
		outcome := "Stopped"
		if report.Outcome == "completed" {
			outcome = "Finished"
		}
		if report.Outcome == "paused" {
			outcome = "Paused"
		}
		writeScreenSection(&b, commandEmoji("run"), "Last Run", []string{voiceLine("history", fmt.Sprintf("%s after %d phases; last phase: %d.", outcome, report.PhasesCompleted, report.CurrentPhase))})
	}
	if total := computeColonyRunningSpendTotal(state); total.hasAnyFacts() {
		writeScreenSection(&b, commandEmoji("history"), "Time And Cost", screenTextLines(renderColonyRunningSpendTotal(total)))
	}
	b.WriteString("\n  Full detail: `aether status --detail`\n\n")
	b.WriteString(renderLifecycleClosing(result, "status"))
	return b.String()
}

func statusScreenVerdict(state colony.ColonyState, result map[string]interface{}) (string, string, string) {
	total, completed := len(state.Plan.Phases), completedPhaseCount(state)
	if outcome := state.SealOutcome; outcome != nil && (outcome.Disposition == colony.SealDispositionForcedIncomplete || outcome.OutcomeKind == colony.OutcomeKindForcedIncompleteClosure) {
		label := "PROJECT CLOSED EARLY — COMPLETION NOT VERIFIED"
		if len(outcome.IncompletePhases) > 0 {
			label = fmt.Sprintf("PROJECT CLOSED EARLY — %s unfinished", screenCount(len(outcome.IncompletePhases), "phase"))
		}
		return "warning", label, "Completion was not verified. " + strings.TrimSpace(outcome.OwnerReason)
	}
	if state.SealOutcome != nil && len(state.SealOutcome.IncompletePhases) > 0 {
		return "warning", "WORK REMAINS", "The closure record still lists unfinished phases. Review `aether status --detail`."
	}
	if state.Paused {
		return "paused", "PAUSED", "Run `aether resume` to pick up where you left off."
	}
	phase := recoveryPhase(&state)
	phaseLabel := "This project"
	if phase != nil {
		phaseLabel = fmt.Sprintf("Phase %d of %d: %s", phase.ID, total, phase.Name)
	}
	if available, ok := result["blocker_snapshot_available"].(bool); ok && !available {
		return "warning", "PROJECT STATUS NEEDS ATTENTION", "The list of blocking problems could not be read."
	}
	if n := intValue(result["blockers"]); n > 0 {
		verb := "need"
		if n == 1 {
			verb = "needs"
		}
		return "blocked", fmt.Sprintf("BLOCKED — %s %s you", screenCount(n, "problem"), verb), phaseLabel + ". Fix it before building on."
	}
	if phase != nil {
		if truth := resolvePhaseProgressFromDisk(*phase); truth.Disagreed {
			return "warning", fmt.Sprintf("PHASE %d — RECORDS DISAGREE", phase.ID), "Showing the less-finished record: " + truth.Status + ". Review `aether status --detail`."
		}
	}
	if state.State == colony.StateCOMPLETED && total > 0 && completed == total {
		return "finished", "PROJECT FINISHED", fmt.Sprintf("All %s are done.", screenCount(total, "phase"))
	}
	if phase == nil {
		return "plan", "READY TO PLAN", "The goal is saved. The project needs a plan."
	}
	label, kind := "READY TO BUILD", "phase"
	switch state.State {
	case colony.StateEXECUTING:
		label = "BUILD IN PROGRESS"
	case colony.StateBUILT:
		label = "READY TO CHECK"
	}
	if phase.Status == "failed" {
		label, kind = "BUILD STOPPED", "blocked"
	}
	if state.State == colony.StateCOMPLETED && completed != total {
		label, kind = "WORK REMAINS", "warning"
	}
	return kind, fmt.Sprintf("PHASE %d OF %d — %s", phase.ID, total, label), fmt.Sprintf("%s. %s finished.", phase.Name, screenCount(completed, "phase"))
}

func statusScreenNeedsYou(state colony.ColonyState, s *storage.Store, result map[string]interface{}) []string {
	var lines, flags []string
	if available, ok := result["blocker_snapshot_available"].(bool); ok && !available {
		lines = append(lines, voiceLine("warning", "Blocker truth: unavailable — the list of blocking problems could not be read. Inspect `aether status --detail`."))
	}
	if n := intValue(result["blockers"]); n > 0 {
		label := fmt.Sprintf("Existing blocker work: %d active", n)
		if escalated := intValue(result["escalated_blockers"]); escalated > 0 {
			label += fmt.Sprintf(" (%d escalated)", escalated)
		}
		lines = append(lines, voiceLine("blocked", label))
	}
	if file, ok := loadFlagsFile(s); ok {
		groups := classifyOpenFlags(file.Decisions)
		for _, group := range []struct {
			label, kind string
			entries     []colony.FlagEntry
		}{
			{"Blocking work", "blocked", groups.Blockers}, {"Issue", "flag", groups.Issues}, {"For later", "feedback", groups.Notes},
		} {
			for _, entry := range group.entries {
				text := group.label + ": " + strings.TrimSpace(entry.Description)
				if entry.Acknowledged {
					text += " — parked"
				}
				flags = append(flags, voiceLine(group.kind, text))
			}
		}
	}
	lines = append(lines, capScreenLines(flags, 5)...)
	// Warnings have a separate budget from flags: a long flag list must never
	// hide an unreadable record, failed check, or outstanding owner decision.
	for _, warning := range stringSliceValue(result["warnings"]) {
		lines = append(lines, voiceLine("warning", warning))
	}
	if s != nil && state.CurrentPhase > 0 {
		var gates []GateCheckResult
		if s.LoadJSON(fmt.Sprintf("gate-results-%d.json", state.CurrentPhase), &gates) == nil {
			for _, gate := range gates {
				if gate.Status == "failed" {
					lines = append(lines, voiceLine("blocked", gateCheckDisplayName(gate.Name)+" failed: "+gate.Detail+" "+gate.FixHint))
				}
			}
		}
	}
	if hub := readInstalledHubVersion(); hub != "" && hub != resolveVersion() {
		lines = append(lines, voiceLine("warning", fmt.Sprintf("Version MISMATCH: running %s; installed copy %s.", resolveVersion(), hub)))
	}
	if guidance := loadActiveRecoveryGuidance(state); guidance != nil {
		lines = append(lines, voiceLine("warning", guidance.Summary+" "+guidance.Next))
	}
	if _, ok := result["reconciliation"]; ok {
		lines = append(lines, voiceLine("warning", "Some changes have not been reconciled with the project record. See `aether status --detail`."))
	}
	return lines
}
