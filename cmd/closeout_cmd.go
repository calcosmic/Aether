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
		result["next"] = closeoutNextCommand(workflow, state)

		outputWorkflow(result, renderCloseoutVisual(result))
		return nil
	},
}

func init() {
	closeoutCmd.Flags().StringVar(&closeoutCompletionFile, "completion-file", "", "Completion JSON packet used by the lifecycle finalizer")
	rootCmd.AddCommand(closeoutCmd)
}

func closeoutNextCommand(workflow string, state colony.ColonyState) string {
	switch workflow {
	case "build":
		return `Run ` + "`aether continue`" + ` to verify worker claims and advance.`
	case "plan":
		if state.CurrentPhase > 0 {
			return fmt.Sprintf("Run `aether build %d` to start the first ready phase.", state.CurrentPhase)
		}
		return `Run ` + "`aether build <phase>`" + ` to start implementation.`
	case "colonize":
		return `Run ` + "`aether plan`" + ` to convert the survey into phases.`
	case "continue":
		if state.State == colony.StateCOMPLETED || allPhasesCompleted(state) {
			return `Run ` + "`aether seal`" + ` to close and archive the colony.`
		}
		if state.CurrentPhase > 0 {
			return fmt.Sprintf("Run `aether build %d` to dispatch the next phase.", state.CurrentPhase)
		}
		return `Run ` + "`aether status`" + ` to inspect the next step.`
	case "seal":
		return `Run ` + "`aether porter check`" + ` if you want delivery readiness, or start a new colony.`
	case "swarm":
		return `Run ` + "`aether status`" + ` to inspect the colony after swarm findings.`
	default:
		if state.State == colony.StateBUILT {
			return `Run ` + "`aether continue`" + ` to verify and advance.`
		}
		return `Run ` + "`aether status`" + ` to inspect the colony.`
	}
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
		status := normalizeRuntimeDispatchStatus(stringValue(worker["status"]))
		switch status {
		case "completed", "passed", "success", "manually-reconciled":
			completed++
		case "blocked":
			blocked++
		case "failed", "timeout", "cancelled":
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
