package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

func screenWorkerStatus(status string) (kind, label string, failed bool) {
	if strings.TrimSpace(status) == "" {
		return "warning", "result not recorded", false
	}
	switch normalizeRuntimeDispatchStatus(status) {
	case "completed", "complete", "done", "succeeded", "code_written", "passed", "success", "manually-reconciled":
		return "done", "finished", false
	case "completed_no_change":
		return "done", "verified existing work", false
	case "failed":
		return "failed", "failed", true
	case "blocked":
		return "blocked", "blocked", true
	case "timeout":
		return "warning", "ran out of time", true
	case "cancelled", "canceled", "interrupted":
		return "warning", "stopped early", true
	case "running", "spawned":
		return "status", "running", false
	case "starting":
		return "status", "starting", false
	case "planned", "pending":
		return "status", "not started", false
	case "skipped":
		return "status", "skipped", false
	default:
		return "warning", "result not recorded", false
	}
}

func buildScreenWorkerMaps(dispatches []codexBuildDispatch) []map[string]interface{} {
	workers := make([]map[string]interface{}, 0, len(dispatches))
	for _, d := range dispatches {
		workers = append(workers, map[string]interface{}{"name": d.Name, "caste": d.Caste, "status": d.Status, "summary": d.Summary, "outputs": d.Outputs, "blockers": d.Blockers})
	}
	return workers
}

func renderBuildScreen(state colony.ColonyState, phase colony.Phase, workers []map[string]interface{}, result map[string]interface{}) string {
	var b strings.Builder
	done := 0
	_, attempt, hasAttempt := loadLatestBuildAttempt(phase.ID)
	var remaining, helpers, fixes, files []string
	// The build journal credits written work before continue marks tasks
	// verified. A pending task alone must not erase a successful build receipt.
	credited := map[string]struct{}{}
	if hasAttempt && attempt.Status == buildAttemptBuilt {
		credited = completedBuildTaskIDs(attempt.Dispatches)
	}
	for index, task := range phase.Tasks {
		_, built := credited[buildTaskID(task, index)]
		if task.Status == colony.TaskCompleted || built {
			done++
		} else {
			remaining = append(remaining, voiceLine("task", task.Goal))
		}
	}
	finished, failed := 0, false
	stopReason := "helper failed"
	for _, worker := range workers {
		kind, status, stopped := screenWorkerStatus(emptyFallback(stringValue(worker["status"]), stringValue(worker["result_status"])))
		if kind == "done" {
			finished++
		}
		if stopped && (!failed || status == "failed") {
			stopReason = "a helper " + status
		}
		failed = failed || stopped
		name := emptyFallback(stringValue(worker["name"]), stringValue(worker["agent_name"]))
		helpers = append(helpers, voiceLine(kind, casteIdentity(stringValue(worker["caste"]))+" "+name+" — "+status))
		for _, blocker := range stringSliceValue(worker["blockers"]) {
			if !failed {
				stopReason = "a helper reported blocking work"
			}
			failed = true
			fixes = append(fixes, voiceLine("blocked", name+": "+blocker))
		}
		if stopped && len(stringSliceValue(worker["blockers"])) == 0 {
			fixes = append(fixes, voiceLine("blocked", name+": "+emptyFallback(stringValue(worker["summary"]), status)))
		}
		for _, field := range []string{"outputs", "files_created", "files_modified", "tests_written"} {
			files = append(files, stringSliceValue(worker[field])...)
		}
	}
	for _, key := range []string{"completion_error", "completion_error_message"} {
		if message := strings.TrimSpace(stringValue(result[key])); message != "" {
			failed = true
			stopReason = "results could not be recorded"
			fixes = append(fixes, voiceLine("blocked", message))
		}
	}
	if boolValue(result["completion_finalizer_failed"]) || boolValue(result["completion_path_blocked"]) {
		failed = true
		stopReason = "results could not be recorded"
	}
	if failed {
		if next := strings.TrimSpace(stringValue(result["completion_next"])); next != "" {
			fixes = append(fixes, voiceLine("next", next))
		}
	}
	if hasAttempt && (attempt.Status == buildAttemptFailed || attempt.Status == buildAttemptInterrupted) {
		failed = true
		stopReason = "the saved build attempt stopped"
	}
	checks := []string{voiceLine("evidence", "The work still needs checking with `aether continue`.")}
	if failed || len(remaining) > 0 {
		checks = []string{voiceLine("evidence", "Finish the remaining work and resolve problems before checking this phase.")}
	}
	if hasAttempt && attempt.FreeChecks != nil {
		checks = nil
		failedChecks := map[string]bool{}
		for _, name := range attempt.FreeChecks.Failed {
			failedChecks[name] = true
		}
		for _, name := range attempt.FreeChecks.ChecksRun {
			kind, label := "done", " passed"
			if failedChecks[name] {
				kind, label = "failed", " failed"
			}
			checks = append(checks, voiceLine(kind, verificationStepDisplayName(name)+label))
		}
		if len(attempt.FreeChecks.ChecksSkipped) > 0 {
			var skipped []string
			for _, name := range attempt.FreeChecks.ChecksSkipped {
				skipped = append(skipped, verificationStepDisplayName(name))
			}
			checks = append(checks, voiceLine("unset", strings.Join(skipped, ", ")+" — not set up"))
		}
		if len(checks) == 0 {
			checks = append(checks, voiceLine("warning", "No individual check results were recorded."))
		}
		if !attempt.FreeChecks.Passed {
			failed = true
			stopReason = "checks failed"
		}
	}
	kind, verdict := "done", fmt.Sprintf("PHASE %d BUILT — READY TO CHECK", phase.ID)
	sentence := fmt.Sprintf("All %s are built. Next: check the work with `aether continue`.", screenCount(done, "task"))
	if finished > 0 {
		verb := "finished their work."
		if finished == 1 {
			verb = "finished its work."
		}
		sentence = screenCount(finished, "helper") + " " + verb + " Next: check it with `aether continue`."
	}
	if failed {
		kind, verdict = "blocked", fmt.Sprintf("PHASE %d BUILD STOPPED — %s", phase.ID, stopReason)
		sentence = "Finished work is kept. Resolve the problems below before continuing."
	} else if len(remaining) > 0 {
		kind, verdict = "warning", fmt.Sprintf("PHASE %d PARTLY BUILT", phase.ID)
		sentence = fmt.Sprintf("%d of %d tasks done; %d still to do.", done, len(phase.Tasks), len(remaining))
	} else if len(phase.Tasks) == 0 || finished < len(workers) || (hasAttempt && attempt.Status == buildAttemptPartial) {
		kind, verdict = "warning", fmt.Sprintf("PHASE %d BUILD NOT CONFIRMED", phase.ID)
		sentence = "The records do not yet confirm that this build finished."
	}
	b.WriteString(renderVerdictBox(kind, verdict, sentence))
	writeScreenSection(&b, commandEmoji("flags"), "What To Fix", capScreenLines(fixes, 5))
	var changed []string
	uncredited := map[string]bool{}
	if hasAttempt {
		files = append(files, attempt.CreditedFiles...)
		for _, file := range attempt.UncreditedFiles {
			files = append(files, file.Path)
			uncredited[file.Path] = true
		}
	}
	for _, file := range uniqueSortedStrings(files) {
		if !screenInternalArtifact(file) {
			label := file
			if uncredited[file] {
				label += " — present, not accepted as finished work"
			}
			changed = append(changed, voiceLine("artifact", label))
		}
	}
	writeScreenSection(&b, commandEmoji("build"), "What Changed", capScreenLines(changed, 6))
	if len(workers) != 1 || finished != 1 {
		writeScreenSection(&b, commandEmoji("swarm"), "Helpers", capScreenLines(helpers, 6))
	}
	writeScreenSection(&b, checkScreenSectionEmoji("checks"), "Checks", checks)
	writeScreenSection(&b, commandEmoji("phase"), "Still To Do", capScreenLines(remaining, 5))
	behind := []string{}
	behind = append(behind, voiceLine("phase", phase.Name))
	if phaseHandoffRecordsExist(phase.ID) {
		behind = append(behind, voiceLine("memory", "The helpers' notes are saved for the next phase."))
	} else {
		behind = append(behind, voiceLine("warning", "This phase's helpers left no notes behind."))
	}
	if suggestions := filterActiveSuggestions(state.PendingSuggestions); len(suggestions) > 0 {
		behind = append(behind, voiceLine("focus", fmt.Sprintf("%d steering suggestions are waiting for review; see `aether suggest-approve`.", len(suggestions))))
	}
	behind = append(behind, voiceLine("evidence", fmt.Sprintf("Full detail: `aether phase %d`", phase.ID)))
	writeScreenSection(&b, checkScreenSectionEmoji("behind"), "Behind The Scenes", behind)
	if result != nil {
		b.WriteString("\n" + renderLifecycleClosing(result, "build"))
	} else {
		b.WriteString("\n" + renderNextActionCard(lifecycleNextActionForState(state, "build", "", "")))
	}
	return b.String()
}

func screenInternalArtifact(path string) bool {
	path = filepath.ToSlash(filepath.Clean(path))
	return strings.HasPrefix(path, ".aether/data/") || strings.Contains(path, "/.aether/data/") || strings.HasPrefix(path, "/tmp/aether-")
}

func renderBuildCeremonyScreen(result map[string]interface{}) string {
	state, err := loadActiveColonyStateReadOnly()
	if err != nil {
		return renderVerdictBox("warning", "BUILD RESULT NEEDS ATTENTION", "The project record could not be read. Inspect `aether status`.") + renderLifecycleClosing(result, "build")
	}
	phaseID := intValue(result["completion_phase"])
	if phaseID == 0 {
		phaseID = state.CurrentPhase
	}
	phase, ok := colonyPhaseByID(state, phaseID)
	if !ok {
		return renderVerdictBox("warning", "BUILD RESULT NEEDS ATTENTION", "The phase could not be found in the project record.") + renderLifecycleClosing(result, "build")
	}
	return renderBuildScreen(state, phase, mapSliceValue(result["completion_workers"]), result)
}
