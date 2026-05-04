package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

type codexExternalBuildCompletion struct {
	DispatchManifest *codexBuildManifest              `json:"dispatch_manifest,omitempty"`
	Manifest         *codexBuildManifest              `json:"manifest,omitempty"`
	Dispatches       []codexExternalBuildWorkerResult `json:"dispatches,omitempty"`
	Results          []codexExternalBuildWorkerResult `json:"results,omitempty"`
	Workers          []codexExternalBuildWorkerResult `json:"workers,omitempty"`
	Claims           *codexBuildClaims                `json:"claims,omitempty"`
}

type codexExternalBuildWorkerResult struct {
	Stage         string              `json:"stage,omitempty"`
	Wave          int                 `json:"wave,omitempty"`
	ExecutionWave int                 `json:"execution_wave,omitempty"`
	Caste         string              `json:"caste,omitempty"`
	Name          string              `json:"name"`
	AntName       string              `json:"ant_name,omitempty"`
	Task          string              `json:"task,omitempty"`
	Status        string              `json:"status"`
	Summary       string              `json:"summary,omitempty"`
	TaskID        string              `json:"task_id,omitempty"`
	TaskIndex     int                 `json:"task_index,omitempty"`
	DependsOn     []string            `json:"depends_on,omitempty"`
	Outputs       []string            `json:"outputs,omitempty"`
	Blockers      []string            `json:"blockers,omitempty"`
	Duration      float64             `json:"duration,omitempty"`
	ToolCount     int                 `json:"tool_count,omitempty"`
	FilesCreated  []string            `json:"files_created,omitempty"`
	FilesModified []string            `json:"files_modified,omitempty"`
	TestsWritten  []string            `json:"tests_written,omitempty"`
	Handoff       codex.WorkerHandoff `json:"handoff,omitempty"`
}

// effectiveName returns the worker name, falling back to AntName when Name is empty.
func (r codexExternalBuildWorkerResult) effectiveName() string {
	if n := strings.TrimSpace(r.Name); n != "" {
		return n
	}
	return strings.TrimSpace(r.AntName)
}

var buildFinalizeCmd = &cobra.Command{
	Use:   "build-finalize <phase>",
	Short: "Record externally spawned wrapper build workers as the phase build packet",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		phaseNum, err := parsePositivePhaseArg(args[0])
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		completionPath, _ := cmd.Flags().GetString("completion-file")
		completion, err := loadExternalBuildCompletion(completionPath)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		result, state, phase, dispatches, err := runCodexBuildFinalize(skillWorkspaceRoot(), phaseNum, completion, false)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		outputWorkflow(result, renderBuildFinalizeVisual(state, phase, dispatches))
		return nil
	},
}

func parsePositivePhaseArg(value string) (int, error) {
	phaseNum, err := strconv.Atoi(value)
	if err != nil || phaseNum < 1 {
		return 0, fmt.Errorf("invalid phase %q", value)
	}
	return phaseNum, nil
}

func loadExternalBuildCompletion(path string) (codexExternalBuildCompletion, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return codexExternalBuildCompletion{}, fmt.Errorf("flag --completion-file is required")
	}
	var data []byte
	var err error
	if path == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return codexExternalBuildCompletion{}, fmt.Errorf("read completion file: %w", err)
	}

	var completion codexExternalBuildCompletion
	if err := json.Unmarshal(data, &completion); err != nil {
		return codexExternalBuildCompletion{}, fmt.Errorf("parse completion file: %w", err)
	}
	if completion.activeManifest() != nil {
		return completion, nil
	}

	var envelope struct {
		Result codexExternalBuildCompletion `json:"result"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return codexExternalBuildCompletion{}, fmt.Errorf("parse completion envelope: %w", err)
	}
	if envelope.Result.activeManifest() == nil {
		return codexExternalBuildCompletion{}, fmt.Errorf("completion file must include dispatch_manifest")
	}
	return envelope.Result, nil
}

func (c codexExternalBuildCompletion) activeManifest() *codexBuildManifest {
	if c.DispatchManifest != nil {
		return c.DispatchManifest
	}
	return c.Manifest
}

func (c codexExternalBuildCompletion) workerResults() []codexExternalBuildWorkerResult {
	results := make([]codexExternalBuildWorkerResult, 0, len(c.Dispatches)+len(c.Results)+len(c.Workers))
	results = append(results, c.Dispatches...)
	results = append(results, c.Results...)
	results = append(results, c.Workers...)
	return results
}

func runCodexBuildFinalize(root string, phaseNum int, completion codexExternalBuildCompletion, skipVerify bool) (map[string]interface{}, colony.ColonyState, colony.Phase, []codexBuildDispatch, error) {
	if store == nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("no store initialized")
	}

	manifest := completion.activeManifest()
	if manifest == nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("completion file must include dispatch_manifest")
	}
	if !manifest.PlanOnly {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("dispatch_manifest must come from `aether build --plan-only`")
	}
	if manifest.Phase != phaseNum {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("completion phase %d does not match requested phase %d", manifest.Phase, phaseNum)
	}
	if len(manifest.Dispatches) == 0 {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("dispatch_manifest contains no dispatches")
	}

	state, err := loadActiveColonyState()
	if err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("%s", colonyStateLoadMessage(err))
	}
	if len(state.Plan.Phases) == 0 {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("No project plan. Run `aether plan` first.")
	}
	if phaseNum < 1 || phaseNum > len(state.Plan.Phases) {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("phase %d not found (plan has %d phases)", phaseNum, len(state.Plan.Phases))
	}
	selectedTaskIDs := uniqueSortedStrings(manifest.SelectedTasks)
	phase := state.Plan.Phases[phaseNum-1]
	if err := validateSelectedBuildTasks(phase, selectedTaskIDs); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}
	if err := runPreBuildGates(store.BasePath(), phaseNum); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}
	if err := validateCodexBuildState(state, phaseNum, selectedTaskIDs, false); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}

	dispatches, err := mergeExternalBuildResults(*manifest, completion.workerResults())
	if err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}
	// Per SAFE-01, SAFE-02: validate build provenance before proceeding.
	// Rejects phantom builds where no worker produced successful results with file modifications.
	if err := validateBuildProvenance(completion.workerResults()); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}
	startedAt := parseManifestGeneratedAt(*manifest)
	completedAt := time.Now().UTC()
	checkpointRel := filepath.ToSlash(filepath.Join("checkpoints", fmt.Sprintf("pre-build-phase-%d.json", phaseNum)))
	buildDirRel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phaseNum)))
	manifestRel := filepath.ToSlash(filepath.Join(buildDirRel, "manifest.json"))
	claimsRel := "last-build-claims.json"

	if err := store.SaveJSON(checkpointRel, state); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("failed to checkpoint colony state: %w", err)
	}

	// Prepare the updated state in memory first (needed for downstream writes).
	updatedState := state
	applyCodexBuildState(&updatedState, phaseNum, startedAt, selectedTaskIDs)
	updatedState.State = colony.StateBUILT
	reconcileCompletedBuildTasks(&updatedState, phaseNum, dispatches)
	updatedPhase := updatedState.Plan.Phases[phaseNum-1]
	updatedState.Events = append(trimmedEvents(updatedState.Events),
		fmt.Sprintf("%s|build_completed|build-finalize|Phase %d external Task workers recorded", completedAt.Format(time.RFC3339), phaseNum),
	)

	claims := completion.claimsOrAggregate(root, phaseNum, startedAt, dispatches)
	if err := store.SaveJSON(claimsRel, claims); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("failed to write build claims: %w", err)
	}

	_, dispatches, err = writeCodexBuildOutcomeReports(root, updatedPhase, buildDirRel, dispatches, completedAt, "external-task")
	if err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}

	finalManifest := buildCodexBuildManifest(root, updatedState, updatedPhase, checkpointRel, claimsRel, manifest.Playbooks, dispatches, startedAt, "external-task", selectedTaskIDs, manifest.WorkerBriefs, false, colony.NormalizeVerificationDepth(manifest.ReviewDepth))
	finalManifest.GeneratedAt = completedAt.Format(time.RFC3339)
	if err := store.SaveJSON(manifestRel, finalManifest); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("failed to write build manifest: %w", err)
	}
	if err := recordExternalBuildSpawnTree(dispatches); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}
	if err := persistExternalBuildHandoffs(root, phaseNum, dispatches, completion.workerResults()); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}
	recoveryInstructions, err := buildExternalBuildRecoveryInstructions(phaseNum, dispatches)
	if err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}

	// Atomically commit the colony state mutation.
	var committedState colony.ColonyState
	if err := store.UpdateJSONAtomically("COLONY_STATE.json", &committedState, func() error {
		committedState = updatedState
		return nil
	}); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("failed to save built colony state: %w", err)
	}
	updatedState = committedState
	updateSessionSummary("build-finalize", "aether continue", fmt.Sprintf("Phase %d external Task workers recorded (%d dispatches)", phaseNum, len(dispatches)))

	result := map[string]interface{}{
		"phase":          phaseNum,
		"phase_name":     updatedPhase.Name,
		"state":          updatedState.State,
		"plan_only":      false,
		"dispatch_mode":  "external-task",
		"dispatches":     codexBuildDispatchMaps(dispatches),
		"dispatch_count": len(dispatches),
		"wave_count":     len(buildWaveExecutionPlans(dispatches, effectiveParallelMode(updatedState))),
		"parallel_mode":  string(effectiveParallelMode(updatedState)),
		"selected_tasks": selectedTaskIDs,
		"checkpoint":     displayDataPath(checkpointRel),
		"manifest":       displayDataPath(manifestRel),
		"claims_path":    displayDataPath(claimsRel),
		"next":           "aether continue",
	}
	if len(recoveryInstructions) > 0 {
		result["recovery_instructions"] = recoveryInstructions
	}
	return result, updatedState, updatedPhase, dispatches, nil
}

func buildExternalBuildRecoveryInstructions(phaseNum int, dispatches []codexBuildDispatch) ([]map[string]interface{}, error) {
	var failed []codexBuildDispatch
	for _, dispatch := range dispatches {
		switch strings.ToLower(strings.TrimSpace(dispatch.Status)) {
		case "", "completed", "manually-reconciled":
			continue
		default:
			failed = append(failed, dispatch)
		}
	}
	if len(failed) == 0 {
		return nil, nil
	}

	wave := failed[0].Wave
	if wave <= 0 {
		wave = 1
	}
	budget := budgetFromRecoveryLog(phaseNum, wave)
	if budget == nil {
		budget = newRecoveryBudget(wave)
	}
	cb := globalCircuitBreaker
	if cb == nil {
		cb = NewCircuitBreaker(3)
	}
	workerDispatches := codexWorkerDispatchesForRecovery(dispatches, phaseNum)

	instructions := make([]map[string]interface{}, 0, len(failed))
	var logEntries []RecoveryLogEntry
	for _, dispatch := range failed {
		status := strings.ToLower(strings.TrimSpace(dispatch.Status))
		message := strings.TrimSpace(dispatch.Summary)
		if len(dispatch.Blockers) > 0 {
			message = strings.TrimSpace(message + " " + strings.Join(dispatch.Blockers, " "))
		}
		outcome := orchestrateRecovery(RecoveryContext{
			Phase:          phaseNum,
			Wave:           normalizedDispatchWave(dispatch),
			WorkerName:     dispatch.Name,
			TaskID:         dispatch.TaskID,
			Caste:          dispatch.Caste,
			Status:         status,
			ErrorMessage:   message,
			Dispatches:     workerDispatches,
			CircuitBreaker: cb,
			Budget:         budget,
		})
		logEntries = append(logEntries, outcome.LogEntries...)
		instructions = append(instructions, map[string]interface{}{
			"worker":             dispatch.Name,
			"task_id":            dispatch.TaskID,
			"caste":              dispatch.Caste,
			"status":             status,
			"action":             outcome.Action.Type,
			"peer":               outcome.Action.PeerName,
			"detail":             outcome.Action.Detail,
			"classification":     string(outcome.Classification),
			"failure_type":       string(outcome.FailureType),
			"rationale":          outcome.Rationale,
			"budget_remaining":   outcome.Action.BudgetRemaining,
			"recovery_exhausted": outcome.Exhausted,
		})
	}
	if err := appendRecoveryOutcomesToLog(phaseNum, budget, logEntries); err != nil {
		return nil, err
	}
	return instructions, nil
}

func codexWorkerDispatchesForRecovery(dispatches []codexBuildDispatch, phaseNum int) []codex.WorkerDispatch {
	workers := make([]codex.WorkerDispatch, 0, len(dispatches))
	for _, dispatch := range dispatches {
		workers = append(workers, codex.WorkerDispatch{
			ID:         normalizedDispatchTaskID(dispatch),
			WorkerName: dispatch.Name,
			AgentName:  codexAgentNameForCaste(dispatch.Caste),
			Caste:      dispatch.Caste,
			TaskID:     dispatch.TaskID,
			TaskBrief:  dispatch.Task,
			Wave:       normalizedDispatchWave(dispatch),
			Workflow:   "build",
			Phase:      phaseNum,
		})
	}
	return workers
}

func appendRecoveryOutcomesToLog(phaseNum int, budget *RecoveryBudget, entries []RecoveryLogEntry) error {
	file, err := recoveryLogReadPhase(phaseNum)
	if err != nil {
		file = RecoveryLogFile{Phase: phaseNum}
	}
	file.Entries = append(file.Entries, entries...)
	file.RecoveryBudget = budget
	rel := fmt.Sprintf("recovery-log-%d.json", phaseNum)
	return store.SaveJSON(rel, file)
}

func mergeExternalBuildResults(manifest codexBuildManifest, results []codexExternalBuildWorkerResult) ([]codexBuildDispatch, error) {
	resultByName := make(map[string]codexExternalBuildWorkerResult, len(results))
	for _, result := range results {
		name := result.effectiveName()
		if name == "" {
			return nil, fmt.Errorf("external worker result missing name")
		}
		if _, exists := resultByName[name]; exists {
			return nil, fmt.Errorf("duplicate external worker result for %s", name)
		}
		resultByName[name] = result
	}

	dispatches := make([]codexBuildDispatch, len(manifest.Dispatches))
	for i, dispatch := range manifest.Dispatches {
		result, ok := resultByName[dispatch.Name]
		if !ok {
			return nil, fmt.Errorf("missing external worker result for %s", dispatch.Name)
		}
		if err := validateExternalResultIdentity(dispatch, result); err != nil {
			return nil, err
		}
		status := normalizeExternalBuildStatus(result.Status)
		if !isTerminalExternalBuildStatus(status) {
			return nil, fmt.Errorf("external worker result for %s has non-terminal status %q", dispatch.Name, result.Status)
		}
		if err := codex.ValidateWorkerHandoff(result.Handoff); err != nil {
			return nil, fmt.Errorf("external worker result for %s has invalid handoff: %w", dispatch.Name, err)
		}
		dispatch.Status = status
		dispatch.Summary = strings.TrimSpace(result.Summary)
		dispatch.Blockers = uniqueSortedStrings(result.Blockers)
		dispatch.Duration = result.Duration
		if outputs := uniqueSortedStrings(append(append(append([]string{}, result.Outputs...), result.FilesCreated...), append(result.FilesModified, result.TestsWritten...)...)); len(outputs) > 0 {
			dispatch.Outputs = outputs
		}
		dispatches[i] = dispatch
	}
	return dispatches, nil
}

func persistExternalBuildHandoffs(root string, phaseNum int, dispatches []codexBuildDispatch, results []codexExternalBuildWorkerResult) error {
	resultByName := make(map[string]codexExternalBuildWorkerResult, len(results))
	for _, result := range results {
		if name := result.effectiveName(); name != "" {
			resultByName[name] = result
		}
	}
	for _, dispatch := range dispatches {
		result, ok := resultByName[dispatch.Name]
		if !ok {
			continue
		}
		status := normalizeExternalBuildStatus(result.Status)
		workerResult := &codex.WorkerResult{
			WorkerName:    dispatch.Name,
			Caste:         dispatch.Caste,
			TaskID:        dispatch.TaskID,
			Status:        status,
			Summary:       result.Summary,
			FilesCreated:  normalizeClaimPathsToRoot(root, result.FilesCreated),
			FilesModified: normalizeClaimPathsToRoot(root, result.FilesModified),
			TestsWritten:  normalizeClaimPathsToRoot(root, result.TestsWritten),
			Blockers:      result.Blockers,
			Handoff:       codex.NormalizeWorkerHandoff(root, result.Handoff),
		}
		if err := persistDispatchWorkerHandoff(codex.WorkerDispatch{
			WorkerName: dispatch.Name,
			Caste:      dispatch.Caste,
			TaskID:     dispatch.TaskID,
			Workflow:   "build",
			Phase:      phaseNum,
			Wave:       normalizedDispatchWave(dispatch),
			Root:       root,
		}, codex.DispatchResult{
			WorkerName:   dispatch.Name,
			Status:       status,
			WorkerResult: workerResult,
		}); err != nil {
			return err
		}
	}
	return nil
}

func validateExternalResultIdentity(dispatch codexBuildDispatch, result codexExternalBuildWorkerResult) error {
	if value := strings.TrimSpace(result.Caste); value != "" && !strings.EqualFold(value, dispatch.Caste) {
		return fmt.Errorf("external worker result %s caste = %q, want %q", dispatch.Name, value, dispatch.Caste)
	}
	if value := strings.TrimSpace(result.Stage); value != "" && !strings.EqualFold(value, dispatch.Stage) {
		return fmt.Errorf("external worker result %s stage = %q, want %q", dispatch.Name, value, dispatch.Stage)
	}
	if value := strings.TrimSpace(result.TaskID); value != "" && value != strings.TrimSpace(dispatch.TaskID) {
		return fmt.Errorf("external worker result %s task_id = %q, want %q", dispatch.Name, value, dispatch.TaskID)
	}
	if result.Wave > 0 && dispatch.Wave > 0 && result.Wave != dispatch.Wave {
		return fmt.Errorf("external worker result %s wave = %d, want %d", dispatch.Name, result.Wave, dispatch.Wave)
	}
	if result.ExecutionWave > 0 && result.ExecutionWave != normalizedDispatchWave(dispatch) {
		return fmt.Errorf("external worker result %s execution_wave = %d, want %d", dispatch.Name, result.ExecutionWave, normalizedDispatchWave(dispatch))
	}
	return nil
}

func normalizeExternalBuildStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
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
		return status
	}
}

func isTerminalExternalBuildStatus(status string) bool {
	switch status {
	case "completed", "failed", "blocked", "timeout", "manually-reconciled":
		return true
	default:
		return false
	}
}

func parseManifestGeneratedAt(manifest codexBuildManifest) time.Time {
	if ts, err := time.Parse(time.RFC3339, strings.TrimSpace(manifest.GeneratedAt)); err == nil {
		return ts.UTC()
	}
	return time.Now().UTC()
}

func (c codexExternalBuildCompletion) claimsOrAggregate(root string, phaseNum int, startedAt time.Time, dispatches []codexBuildDispatch) codexBuildClaims {
	if c.Claims != nil {
		claims := *c.Claims
		claims.BuildPhase = phaseNum
		if strings.TrimSpace(claims.Timestamp) == "" {
			claims.Timestamp = startedAt.Format(time.RFC3339)
		}
		claims.FilesCreated = uniqueSortedStrings(claims.FilesCreated)
		claims.FilesModified = uniqueSortedStrings(claims.FilesModified)
		claims.TestsWritten = uniqueSortedStrings(claims.TestsWritten)
		claims.FilesCreated = normalizeClaimPathsToRoot(root, claims.FilesCreated)
		claims.FilesModified = normalizeClaimPathsToRoot(root, claims.FilesModified)
		claims.TestsWritten = normalizeClaimPathsToRoot(root, claims.TestsWritten)
		return claims
	}

	byName := map[string]codexExternalBuildWorkerResult{}
	for _, result := range c.workerResults() {
		name := result.effectiveName()
		if name != "" {
			byName[name] = result
		}
	}
	claims := codexBuildClaims{BuildPhase: phaseNum, Timestamp: startedAt.Format(time.RFC3339)}
	taskClaims := map[string]*codexBuildTaskClaim{}
	for _, dispatch := range dispatches {
		if dispatch.Status != "completed" {
			continue
		}
		result, ok := byName[dispatch.Name]
		if !ok {
			continue
		}
		claims.FilesCreated = append(claims.FilesCreated, result.FilesCreated...)
		claims.FilesModified = append(claims.FilesModified, result.FilesModified...)
		claims.TestsWritten = append(claims.TestsWritten, result.TestsWritten...)
		taskID := strings.TrimSpace(dispatch.TaskID)
		if taskID == "" {
			continue
		}
		entry, ok := taskClaims[taskID]
		if !ok {
			entry = &codexBuildTaskClaim{TaskID: taskID}
			taskClaims[taskID] = entry
		}
		entry.FilesCreated = append(entry.FilesCreated, result.FilesCreated...)
		entry.FilesModified = append(entry.FilesModified, result.FilesModified...)
		entry.TestsWritten = append(entry.TestsWritten, result.TestsWritten...)
	}
	claims.FilesCreated = uniqueSortedStrings(claims.FilesCreated)
	claims.FilesModified = uniqueSortedStrings(claims.FilesModified)
	claims.TestsWritten = uniqueSortedStrings(claims.TestsWritten)
	claims.FilesCreated = normalizeClaimPathsToRoot(root, claims.FilesCreated)
	claims.FilesModified = normalizeClaimPathsToRoot(root, claims.FilesModified)
	claims.TestsWritten = normalizeClaimPathsToRoot(root, claims.TestsWritten)

	// Filesystem fallback: if claims are empty but builders completed, discover files via git.
	if len(claims.FilesCreated) == 0 && len(claims.FilesModified) == 0 && hasCompletedBuilders(dispatches) {
		created, modified := discoverChangedFilesFromGit()
		claims.FilesCreated = created
		claims.FilesModified = modified
	}

	if len(taskClaims) > 0 {
		taskIDs := make([]string, 0, len(taskClaims))
		for taskID := range taskClaims {
			taskIDs = append(taskIDs, taskID)
		}
		sort.Strings(taskIDs)
		for _, taskID := range taskIDs {
			entry := taskClaims[taskID]
			entry.FilesCreated = uniqueSortedStrings(entry.FilesCreated)
			entry.FilesModified = uniqueSortedStrings(entry.FilesModified)
			entry.TestsWritten = uniqueSortedStrings(entry.TestsWritten)
			entry.FilesCreated = normalizeClaimPathsToRoot(root, entry.FilesCreated)
			entry.FilesModified = normalizeClaimPathsToRoot(root, entry.FilesModified)
			entry.TestsWritten = normalizeClaimPathsToRoot(root, entry.TestsWritten)
			claims.TaskClaims = append(claims.TaskClaims, *entry)
		}
	}
	return claims
}

func recordExternalBuildSpawnTree(dispatches []codexBuildDispatch) error {
	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	entries, err := spawnTree.Parse()
	if err != nil {
		return fmt.Errorf("failed to read spawn tree: %w", err)
	}
	known := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		known[entry.AgentName] = struct{}{}
	}
	for _, dispatch := range dispatches {
		if _, ok := known[dispatch.Name]; !ok {
			if err := spawnTree.RecordSpawn("Queen", dispatch.Caste, dispatch.Name, dispatch.Task, 1); err != nil {
				return fmt.Errorf("failed to record external build dispatch %s: %w", dispatch.Name, err)
			}
			known[dispatch.Name] = struct{}{}
		}
		if err := spawnTree.UpdateStatus(dispatch.Name, dispatch.Status, dispatch.Summary); err != nil {
			return fmt.Errorf("failed to complete external build dispatch %s: %w", dispatch.Name, err)
		}
	}
	return nil
}

func hasCompletedBuilders(dispatches []codexBuildDispatch) bool {
	for _, d := range dispatches {
		if strings.EqualFold(d.Caste, "builder") && d.Status == "completed" {
			return true
		}
	}
	return false
}

func discoverChangedFilesFromGit() (created, modified []string) {
	if out, err := exec.Command("git", "diff", "--name-only", "--diff-filter=A", "HEAD").Output(); err == nil {
		created = parseGitNameOutput(out)
	}
	if out, err := exec.Command("git", "diff", "--name-only", "--diff-filter=M", "HEAD").Output(); err == nil {
		modified = parseGitNameOutput(out)
	}
	return created, modified
}

func parseGitNameOutput(out []byte) []string {
	var result []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		result = append(result, line)
	}
	return uniqueSortedStrings(result)
}

// normalizeClaimPathsToRoot resolves subdirectory-relative paths to repo-root-relative paths.
// If a path already resolves from root (file exists), it is kept as-is.
// If not found, it searches the repo for a matching file and replaces with the resolved path.
func normalizeClaimPathsToRoot(root string, paths []string) []string {
	if root == "" {
		return paths
	}
	result := make([]string, len(paths))
	for i, p := range paths {
		if p == "" {
			continue
		}
		if fileExists(filepath.Join(root, filepath.FromSlash(p))) {
			result[i] = p
			continue
		}
		if resolved := findRepoRelativePath(root, p); resolved != "" {
			result[i] = resolved
			continue
		}
		// Keep original — verification will flag it as missing
		result[i] = p
	}
	return result
}

// findRepoRelativePath searches for a file in the repo that matches the claimed path.
// Uses git ls-files for fast lookup. If git finds nothing, the file likely doesn't
// exist in the repo (no unbounded filesystem walk fallback).
func findRepoRelativePath(root, claimed string) string {
	base := filepath.Base(claimed)
	if base == "." || base == string(filepath.Separator) {
		return ""
	}

	// Try git ls-files with basename pattern for fast lookup
	out, err := exec.Command("git", "-C", root, "ls-files", "--", "*"+base).Output()
	if err == nil {
		candidates := parseGitNameOutput(out)
		if len(candidates) == 1 {
			return candidates[0]
		}
		if len(candidates) > 1 {
			if best := bestMatchForClaimedPath(claimed, candidates); best != "" {
				return best
			}
		}
	}

	// If git ls-files found nothing, the file likely doesn't exist in the repo.
	return ""
}

// bestMatchForClaimedPath scores candidates by counting matching trailing path segments.
// Tiebreaks by shortest total path length.
func bestMatchForClaimedPath(claimed string, candidates []string) string {
	claimedParts := strings.Split(filepath.ToSlash(claimed), "/")
	best := ""
	bestScore := 0
	bestLen := 0
	for _, c := range candidates {
		cParts := strings.Split(filepath.ToSlash(c), "/")
		score := 0
		minLen := len(claimedParts)
		if len(cParts) < minLen {
			minLen = len(cParts)
		}
		for i := 1; i <= minLen; i++ {
			if claimedParts[len(claimedParts)-i] == cParts[len(cParts)-i] {
				score++
			} else {
				break
			}
		}
		cLen := len(cParts)
		if best == "" || score > bestScore || (score == bestScore && cLen < bestLen) {
			best = c
			bestScore = score
			bestLen = cLen
		}
	}
	return best
}
