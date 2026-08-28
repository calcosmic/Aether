package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

var closeoutCompletionFile string

var closeoutCmd = &cobra.Command{
	Use:   "closeout [workflow]",
	Short: "Render the visual closeout for wrapper-driven lifecycle commands",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		workflow := "status"
		if len(args) > 0 {
			workflow = strings.ToLower(strings.TrimSpace(args[0]))
		}
		result := map[string]interface{}{
			"workflow":        workflow,
			"completion_file": closeoutCompletionFile,
		}
		for key, value := range closeoutCompletionDetails(closeoutCompletionFile) {
			result[key] = value
		}
		if workflow == "seal" {
			result["porter_readiness"] = buildPorterReadinessSummary()
		}

		state, err := loadActiveColonyState()
		if err != nil {
			result["state_available"] = false
			result["message"] = colonyStateLoadMessage(err)
			outputWorkflow(result, renderCloseoutVisual(result))
			return nil
		}

		result["state_available"] = true
		result["state"] = string(state.State)
		result["current_phase"] = state.CurrentPhase
		result["total_phases"] = len(state.Plan.Phases)
		result["completed_phases"] = completedPhaseCount(state)
		result["milestone"] = state.Milestone
		result["phase_name"] = lookupPhaseName(state, state.CurrentPhase)
		if state.Goal != nil {
			result["goal"] = *state.Goal
		}
		// The owner-facing sentence and the machine-readable fields come from
		// ONE answer, so the two can never name different commands.
		answer := closeoutNextAction(workflow, state)
		result["next"] = nextActionPrimarySuggestion(answer)
		applyNextActionToResult(result, answer)

		outputWorkflow(result, renderCloseoutVisual(result))
		return nil
	},
}

func init() {
	closeoutCmd.Flags().StringVar(&closeoutCompletionFile, "completion-file", "", "Completion JSON packet used by the lifecycle finalizer")
	rootCmd.AddCommand(closeoutCmd)
}

// closeoutNextCommand is the closing line for a lifecycle command that has just
// finished. Phase 197 plan 02: the workflow name is real information -- which
// command just ran -- so it is passed to the resolver as an INPUT rather than
// used to pick a different answer. Two closeouts of the same saved state now
// name the same next command whichever command produced them.
func closeoutNextCommand(workflow string, state colony.ColonyState) string {
	return nextActionPrimarySuggestion(closeoutNextAction(workflow, state))
}

// closeoutNextAction is the resolved answer behind that sentence, so the
// command can emit the card and the machine-readable fields from one decision
// rather than asking twice and hoping the two agree.
func closeoutNextAction(workflow string, state colony.ColonyState) nextAction {
	return resolveNextAction(nextActionInputForState(state, strings.TrimSpace(workflow)))
}

func closeoutCompletionDetails(path string) map[string]interface{} {
	details := map[string]interface{}{}
	path = strings.TrimSpace(path)
	if path == "" {
		return details
	}

	var data []byte
	var err error
	if path == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		details["completion_loaded"] = false
		details["completion_error"] = fmt.Sprintf("read completion file: %v", err)
		return details
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		details["completion_loaded"] = false
		details["completion_error"] = fmt.Sprintf("parse completion file: %v", err)
		return details
	}
	if nested, ok := raw["result"].(map[string]interface{}); ok {
		if key, _ := closeoutManifest(nested); key != "" || len(closeoutWorkerMaps(nested)) > 0 {
			raw = nested
		}
	}

	details["completion_loaded"] = true
	if _, ok := raw["ok"]; ok {
		completionOK := boolValue(raw["ok"])
		details["completion_ok"] = completionOK
		if !completionOK {
			details["completion_finalizer_failed"] = true
		}
	}
	if errText := strings.TrimSpace(stringValue(raw["error"])); errText != "" {
		details["completion_error_message"] = errText
		if _, ok := raw["ok"]; ok && !boolValue(raw["ok"]) {
			details["completion_finalizer_failed"] = true
		}
	}
	if message := strings.TrimSpace(stringValue(raw["message"])); message != "" {
		details["completion_message"] = message
	}
	if next := strings.TrimSpace(stringValue(raw["next"])); next != "" {
		details["completion_next"] = next
	}
	if existingPlan, ok := raw["existing_plan"].(bool); ok {
		details["completion_existing_plan"] = existingPlan
	}
	if requiresFinalizer, ok := raw["requires_finalizer"].(bool); ok {
		details["completion_requires_finalizer"] = requiresFinalizer
	}
	if blocked, ok := raw["blocked"].(bool); ok {
		details["completion_path_blocked"] = blocked
	}
	if manifestKey, manifest := closeoutManifest(raw); manifestKey != "" {
		details["completion_manifest"] = manifestKey
		if phase := intValue(manifest["phase"]); phase > 0 {
			details["completion_phase"] = phase
		}
		if phaseName := strings.TrimSpace(stringValue(manifest["phase_name"])); phaseName != "" {
			details["completion_phase_name"] = phaseName
		}
		if dispatches := mapSliceValue(manifest["dispatches"]); len(dispatches) > 0 {
			details["completion_dispatch_count"] = len(dispatches)
		}
	}

	workers := closeoutWorkerMaps(raw)
	details["completion_workers"] = workers
	details["completion_worker_count"] = len(workers)
	completed, failed, blocked := 0, 0, 0
	blockers := []string{}
	artifacts := []string{}
	for _, worker := range workers {
		status := normalizeCloseoutWorkerStatus(worker)
		switch status {
		// completed_no_change is a success with evidence (ruling D6); leaving
		// it out silently dropped honest workers from the closeout tally.
		case "completed", "completed_no_change", "passed", "success", "manually-reconciled":
			completed++
		case "blocked":
			blocked++
		case "failed", "timeout", "cancelled", "canceled":
			failed++
		}
		name := emptyFallback(stringValue(worker["name"]), stringValue(worker["agent_name"]))
		for _, blocker := range stringSliceValue(worker["blockers"]) {
			if strings.TrimSpace(name) != "" {
				blockers = append(blockers, fmt.Sprintf("%s: %s", name, blocker))
			} else {
				blockers = append(blockers, blocker)
			}
		}
		for _, field := range []string{"outputs", "files_created", "files_modified", "tests_written"} {
			artifacts = append(artifacts, stringSliceValue(worker[field])...)
		}
	}
	details["completion_completed"] = completed
	details["completion_failed"] = failed
	details["completion_blocked"] = blocked
	details["completion_blockers"] = uniqueSortedStrings(blockers)
	details["completion_artifacts"] = uniqueSortedStrings(artifacts)
	if phases := ceremonyPhaseSummariesFromCompletion(raw); len(phases) > 0 {
		details["completion_phases"] = phases
		details["completion_phase_count"] = len(phases)
	}
	return details
}

func closeoutManifest(raw map[string]interface{}) (string, map[string]interface{}) {
	for _, key := range []string{
		"dispatch_manifest",
		"build_manifest",
		"plan_manifest",
		"planning_manifest",
		"colonize_manifest",
		"survey_manifest",
		"continue_manifest",
		"seal_manifest",
		"swarm_manifest",
		"manifest",
	} {
		if manifest := mapValue(raw[key]); len(manifest) > 0 {
			return key, manifest
		}
	}
	return "", nil
}

func closeoutWorkerMaps(raw map[string]interface{}) []map[string]interface{} {
	workers := []map[string]interface{}{}
	seen := map[string]bool{}
	for _, key := range []string{"dispatches", "results", "workers"} {
		for _, worker := range mapSliceValue(raw[key]) {
			if !isVerifiedCloseoutWorkerResult(worker) {
				continue
			}
			name := emptyFallback(stringValue(worker["name"]), stringValue(worker["agent_name"]))
			id := strings.Join([]string{name, stringValue(worker["status"]), stringValue(worker["summary"]), stringValue(worker["task_id"])}, "\x00")
			if seen[id] {
				continue
			}
			seen[id] = true
			workers = append(workers, worker)
		}
	}
	return workers
}

func isVerifiedCloseoutWorkerResult(worker map[string]interface{}) bool {
	switch normalizeCloseoutWorkerStatus(worker) {
	case "completed", "completed_no_change", "interrupted", "blocked", "failed", "timeout", "manually-reconciled":
		return true
	default:
		return false
	}
}

func normalizeCloseoutWorkerStatus(worker map[string]interface{}) string {
	status := emptyFallback(stringValue(worker["status"]), stringValue(worker["result_status"]))
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" {
		return ""
	}
	switch status {
	case "complete", "done", "success", "succeeded", "passed", "code_written":
		return "completed"
	case "fail", "error":
		return "failed"
	case "timed_out", "cancelled", "canceled":
		return "timeout"
	case "manual", "manually_reconciled":
		return "manually-reconciled"
	default:
		return normalizeRuntimeDispatchStatus(status)
	}
}
