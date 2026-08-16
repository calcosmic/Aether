package cmd

import (
	"fmt"
	"strings"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

func emitCodexBuildWaveProgress(phase colony.Phase, wave int, dispatches []codex.WorkerDispatch, parallelMode colony.ParallelMode) {
	_ = phase
	emitCodexDispatchWaveProgress("Wave", wave, dispatches, parallelMode)
}

func emitCodexDispatchWaveProgress(label string, wave int, dispatches []codex.WorkerDispatch, parallelMode colony.ParallelMode) {
	if len(dispatches) == 0 {
		return
	}

	mode := string(parallelMode)
	if strings.TrimSpace(mode) == "" {
		mode = string(colony.ModeInRepo)
	}

	var b strings.Builder
	b.WriteString(renderStageMarker("Dispatch"))
	if strings.TrimSpace(label) == "" {
		label = "Wave"
	}
	policy := buildWaveExecutionPlan(wave, len(dispatches), parallelMode)
	b.WriteString(fmt.Sprintf("%s %d starting (%s, %s)\n", label, wave, mode, policy.Strategy))
	if reason := strings.TrimSpace(policy.Reason); reason != "" {
		b.WriteString("  Execution: ")
		b.WriteString(policy.Strategy)
		b.WriteString(" — ")
		b.WriteString(reason)
		if policy.Strategy == "serial" && policy.WorkerCount > 1 && parallelMode == colony.ModeInRepo {
			b.WriteString(" (set `aether parallel-mode set worktree` for isolated parallel builders)")
		}
		b.WriteString("\n")
	}
	b.WriteString(renderSpawnAnnouncement(dispatches, policy.Strategy))
	b.WriteString("\n")
	for _, dispatch := range dispatches {
		b.WriteString("  ")
		b.WriteString(casteIdentity(dispatch.Caste))
		b.WriteString(" ")
		b.WriteString(dispatch.WorkerName)
		b.WriteString("  ")
		b.WriteString(strings.TrimSpace(workerDispatchSummary(dispatch)))
		b.WriteString("\n")
	}

	emitVisualProgress(strings.TrimSpace(b.String()))
}

// renderSpawnAnnouncement is the classic moment-of-dispatch beat:
// `──── 🔨🐜 Spawning 3 Builders in parallel ────`. Single worker gets its
// name; a mixed team gets the caste composition.
func renderSpawnAnnouncement(dispatches []codex.WorkerDispatch, strategy string) string {
	if len(dispatches) == 0 {
		return ""
	}
	manner := "in parallel"
	if strategy == "serial" {
		manner = "serially"
	}

	casteOrder := []string{}
	counts := map[string]int{}
	for _, dispatch := range dispatches {
		key := normalizeCasteKey(dispatch.Caste)
		if counts[key] == 0 {
			casteOrder = append(casteOrder, key)
		}
		counts[key]++
	}

	if len(casteOrder) == 1 {
		caste := casteOrder[0]
		glyph := casteEmoji(caste) + "🐜"
		if len(dispatches) == 1 {
			single := dispatches[0]
			summary := strings.TrimSpace(workerDispatchSummary(single))
			if summary != "" {
				return fmt.Sprintf("──── %s Spawning %s — %s ────", glyph, single.WorkerName, summary)
			}
			return fmt.Sprintf("──── %s Spawning %s ────", glyph, single.WorkerName)
		}
		return fmt.Sprintf("──── %s Spawning %d %ss %s ────", glyph, len(dispatches), casteLabel(caste), manner)
	}

	parts := make([]string, 0, len(casteOrder))
	for _, caste := range casteOrder {
		parts = append(parts, fmt.Sprintf("%d %s %s", counts[caste], casteEmoji(caste), casteLabel(caste)))
	}
	return fmt.Sprintf("──── 🐜 Spawning %d workers (%s) %s ────", len(dispatches), strings.Join(parts, ", "), manner)
}

func emitCodexBuildWorkerStarted(dispatch codex.WorkerDispatch, wave int) {
	emitCodexDispatchWorkerStarted(dispatch, wave)
}

func emitCodexDispatchWorkerStarted(dispatch codex.WorkerDispatch, wave int) {
	var b strings.Builder
	b.WriteString("… ")
	b.WriteString(casteIdentity(dispatch.Caste))
	b.WriteString(" ")
	b.WriteString(dispatch.WorkerName)
	b.WriteString(fmt.Sprintf("  starting wave %d", wave))
	if summary := strings.TrimSpace(workerDispatchSummary(dispatch)); summary != "" {
		b.WriteString("  ")
		b.WriteString(summary)
	}
	emitVisualProgress(b.String())
}

func emitCodexDispatchWorkerRunning(dispatch codex.WorkerDispatch, wave int, note string) {
	var b strings.Builder
	b.WriteString("… ")
	b.WriteString(casteIdentity(dispatch.Caste))
	b.WriteString(" ")
	b.WriteString(dispatch.WorkerName)
	b.WriteString(fmt.Sprintf("  running wave %d", wave))
	if summary := strings.TrimSpace(workerDispatchSummary(dispatch)); summary != "" {
		b.WriteString("  ")
		b.WriteString(summary)
	}
	if note := strings.TrimSpace(note); note != "" {
		b.WriteString("  [")
		b.WriteString(note)
		b.WriteString("]")
	}
	emitVisualProgress(b.String())
}

func emitCodexBuildWorkerFinished(dispatch codex.WorkerDispatch, result codex.DispatchResult) {
	emitCodexDispatchWorkerFinished(dispatch, result)
}

func emitCodexDispatchWorkerFinished(dispatch codex.WorkerDispatch, result codex.DispatchResult) {
	status := strings.TrimSpace(result.Status)
	if status == "" {
		status = "failed"
	}

	icon := "•"
	switch status {
	case "completed":
		icon = "✓"
	case "blocked":
		icon = "!"
	case "failed", "timeout":
		icon = "✗"
	case "active", "spawned", "starting", "running":
		icon = "…"
	}

	var b strings.Builder
	b.WriteString(icon)
	b.WriteString(" ")
	b.WriteString(casteIdentity(dispatch.Caste))
	b.WriteString(" ")
	b.WriteString(dispatch.WorkerName)
	b.WriteString("  ")
	b.WriteString(status)

	if result.WorkerResult != nil && result.WorkerResult.Duration > 0 {
		b.WriteString(fmt.Sprintf(" %.1fs", result.WorkerResult.Duration.Seconds()))
	}

	if summary := strings.TrimSpace(dispatchResultSummary(dispatch, result)); summary != "" {
		b.WriteString("  ")
		b.WriteString(summary)
	}

	emitVisualProgress(b.String())
}

func updateCodexBuildDispatchRuntimeStatus(name, status, summary string) error {
	if store == nil {
		return nil
	}
	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	return spawnTree.UpdateStatus(name, status, summary)
}

func buildDispatchResultSummary(dispatch codex.WorkerDispatch, result codex.DispatchResult) string {
	return dispatchResultSummary(dispatch, result)
}

func dispatchResultSummary(dispatch codex.WorkerDispatch, result codex.DispatchResult) string {
	if result.WorkerResult != nil {
		if summary := strings.TrimSpace(result.WorkerResult.Summary); summary != "" {
			return summary
		}
		if len(result.WorkerResult.Blockers) > 0 {
			return strings.Join(result.WorkerResult.Blockers, "; ")
		}
	}
	if result.Error != nil {
		return strings.TrimSpace(result.Error.Error())
	}
	if summary := strings.TrimSpace(workerDispatchSummary(dispatch)); summary != "" {
		return summary
	}
	return strings.TrimSpace(dispatch.WorkerName)
}

func buildDispatchActiveSummary(dispatch codex.WorkerDispatch, wave int) string {
	if summary := strings.TrimSpace(workerDispatchSummary(dispatch)); summary != "" {
		return fmt.Sprintf("Wave %d running: %s", wave, summary)
	}
	return fmt.Sprintf("Wave %d running", wave)
}

func workerDispatchSummary(dispatch codex.WorkerDispatch) string {
	brief := strings.TrimSpace(dispatch.TaskBrief)
	if brief == "" {
		return dispatch.TaskID
	}
	return firstContentLine(brief)
}

func summarizeDispatchOutcome(dispatch codexBuildDispatch) string {
	if summary := strings.TrimSpace(dispatch.Summary); summary != "" {
		return summary
	}
	if len(dispatch.Blockers) > 0 {
		return strings.Join(dispatch.Blockers, "; ")
	}
	return strings.TrimSpace(dispatch.Task)
}

func continueWorkerCloseSummary(dispatch codexBuildDispatch) string {
	if summary := strings.TrimSpace(dispatch.Summary); summary != "" {
		return summary
	}
	if len(dispatch.Blockers) > 0 {
		return strings.Join(dispatch.Blockers, "; ")
	}
	if dispatch.Caste == "watcher" {
		return "Verification passed during continue"
	}
	return "Completed before continue verification"
}
