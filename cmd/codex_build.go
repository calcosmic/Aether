package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

type codexBuildDispatch struct {
	Stage     string   `json:"stage"`
	Wave      int      `json:"wave,omitempty"`
	Caste     string   `json:"caste"`
	Name      string   `json:"name"`
	Task      string   `json:"task"`
	Status    string   `json:"status"`
	Summary   string   `json:"summary,omitempty"`
	TaskID    string   `json:"task_id,omitempty"`
	TaskIndex int      `json:"task_index,omitempty"`
	DependsOn []string `json:"depends_on,omitempty"`
	Outputs   []string `json:"outputs,omitempty"`
	Blockers  []string `json:"blockers,omitempty"`
	Duration  float64  `json:"duration,omitempty"`
}

type codexBuildTaskPlan struct {
	ID        string   `json:"id,omitempty"`
	Goal      string   `json:"goal"`
	Status    string   `json:"status"`
	Wave      int      `json:"wave,omitempty"`
	DependsOn []string `json:"depends_on,omitempty"`
}

type codexBuildManifest struct {
	Phase           int                      `json:"phase"`
	PhaseName       string                   `json:"phase_name"`
	Goal            string                   `json:"goal,omitempty"`
	Root            string                   `json:"root"`
	ParallelMode    string                   `json:"parallel_mode,omitempty"`
	WaveExecution   []codexWaveExecutionPlan `json:"wave_execution,omitempty"`
	ColonyDepth     string                   `json:"colony_depth"`
	DispatchMode    string                   `json:"dispatch_mode,omitempty"`
	GeneratedAt     string                   `json:"generated_at"`
	State           string                   `json:"state"`
	Checkpoint      string                   `json:"checkpoint"`
	ClaimsPath      string                   `json:"claims_path"`
	Playbooks       []string                 `json:"playbooks"`
	WorkerBriefs    []string                 `json:"worker_briefs"`
	Dispatches      []codexBuildDispatch     `json:"dispatches"`
	SelectedTasks   []string                 `json:"selected_tasks,omitempty"`
	Tasks           []codexBuildTaskPlan     `json:"tasks"`
	SuccessCriteria []string                 `json:"success_criteria"`
}

type codexWaveExecutionPlan struct {
	Wave        int    `json:"wave"`
	Strategy    string `json:"strategy"`
	WorkerCount int    `json:"worker_count"`
	Reason      string `json:"reason"`
}

type codexBuildTaskClaim struct {
	TaskID        string   `json:"task_id"`
	FilesCreated  []string `json:"files_created,omitempty"`
	FilesModified []string `json:"files_modified,omitempty"`
	TestsWritten  []string `json:"tests_written,omitempty"`
}

type codexBuildClaims struct {
	FilesCreated  []string              `json:"files_created"`
	FilesModified []string              `json:"files_modified"`
	TestsWritten  []string              `json:"tests_written,omitempty"`
	TaskClaims    []codexBuildTaskClaim `json:"task_claims,omitempty"`
	BuildPhase    int                   `json:"build_phase"`
	Timestamp     string                `json:"timestamp"`
}

var newCodexWorkerInvoker = codex.NewWorkerInvoker
var newCodexWorkerInvokerOrError = codex.NewWorkerInvokerOrError

func runCodexBuild(root string, phaseNum int, selectedTaskIDs []string, synthetic bool) (map[string]interface{}, error) {
	if store == nil {
		return nil, fmt.Errorf("no store initialized")
	}

	state, err := loadActiveColonyState()
	if err != nil {
		return nil, fmt.Errorf("%s", colonyStateLoadMessage(err))
	}
	if len(state.Plan.Phases) == 0 {
		return nil, fmt.Errorf("No project plan. Run `aether plan` first.")
	}
	if phaseNum < 1 || phaseNum > len(state.Plan.Phases) {
		return nil, fmt.Errorf("phase %d not found (plan has %d phases)", phaseNum, len(state.Plan.Phases))
	}
	selectedTaskIDs = uniqueSortedStrings(selectedTaskIDs)
	phase := state.Plan.Phases[phaseNum-1]
	if err := validateSelectedBuildTasks(phase, selectedTaskIDs); err != nil {
		return nil, err
	}
	// Run pre-build gates (critical flags, phase buildability)
	if err := runPreBuildGates(store.BasePath(), phaseNum); err != nil {
		return nil, err
	}
	if err := validateCodexBuildState(state, phaseNum, selectedTaskIDs); err != nil {
		return nil, err
	}
	originalState, err := cloneColonyState(state)
	if err != nil {
		return nil, fmt.Errorf("failed to clone colony state: %w", err)
	}

	startedAt := time.Now().UTC()
	runHandle, err := beginRuntimeSpawnRun("build", startedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize build run: %w", err)
	}
	runStatus := "failed"
	defer func() {
		finishRuntimeSpawnRun(runHandle, runStatus, time.Now().UTC())
	}()

	depth := strings.TrimSpace(state.ColonyDepth)
	if depth == "" {
		depth = "standard"
	}
	playbooks := codexBuildPlaybooks()
	dispatches := plannedBuildDispatchesForSelection(phase, depth, selectedTaskIDs)
	dispatches, err = ensureUniqueBuildDispatchNames(dispatches)
	if err != nil {
		return nil, err
	}
	parallelMode := effectiveParallelMode(state)
	waveExecution := buildWaveExecutionPlans(dispatches, parallelMode)
	waveCount := len(waveExecution)
	parallelWaves := countParallelWaveExecutionPlans(waveExecution)
	checkpointRel := filepath.ToSlash(filepath.Join("checkpoints", fmt.Sprintf("pre-build-phase-%d.json", phaseNum)))
	buildDirRel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phaseNum)))
	manifestRel := filepath.ToSlash(filepath.Join(buildDirRel, "manifest.json"))
	claimsRel := "last-build-claims.json"

	if err := store.SaveJSON(checkpointRel, state); err != nil {
		return nil, fmt.Errorf("failed to checkpoint colony state: %w", err)
	}

	updatedState := state
	applyCodexBuildState(&updatedState, phaseNum, startedAt, selectedTaskIDs)
	updatedPhase := updatedState.Plan.Phases[phaseNum-1]
	if err := store.SaveJSON("COLONY_STATE.json", updatedState); err != nil {
		return nil, fmt.Errorf("failed to save colony state: %w", err)
	}

	briefPaths, dispatches, err := writeCodexBuildArtifacts(root, updatedState, updatedPhase, buildDirRel, checkpointRel, claimsRel, playbooks, dispatches, startedAt, "", selectedTaskIDs)
	if err != nil {
		rollbackCodexBuildFailure(originalState, phaseNum, startedAt, err)
		return nil, err
	}
	if err := recordCodexBuildDispatches(dispatches); err != nil {
		rollbackCodexBuildFailure(originalState, phaseNum, startedAt, err)
		return nil, err
	}
	emitVisualProgress(renderBuildDispatchPreview(updatedState, updatedPhase, dispatches))

	buildInvoker := newCodexWorkerInvoker()
	if synthetic {
		buildInvoker = &codex.FakeInvoker{}
	}
	dispatches, claims, mode, err := executeCodexBuildDispatches(context.Background(), root, updatedPhase, dispatches, playbooks, startedAt, buildInvoker, parallelMode)
	if err != nil {
		rollbackCodexBuildFailure(originalState, phaseNum, startedAt, err)
		return nil, err
	}
	if err := writeCodexBuildClaims(claimsRel, phaseNum, startedAt, claims); err != nil {
		return nil, err
	}
	if latestState, loadErr := loadActiveColonyState(); loadErr == nil {
		updatedState.Worktrees = latestState.Worktrees
	}
	updatedState.State = colony.StateBUILT
	if _, _, err := writeCodexBuildArtifacts(root, updatedState, updatedPhase, buildDirRel, checkpointRel, claimsRel, playbooks, dispatches, startedAt, mode, selectedTaskIDs); err != nil {
		return nil, err
	}

	updatedState.Events = append(trimmedEvents(updatedState.Events),
		fmt.Sprintf("%s|build_completed|build|Phase %d build packet prepared (%s dispatch)", startedAt.Format(time.RFC3339), phaseNum, mode),
	)
	if tracer != nil && updatedState.RunID != nil {
		_ = tracer.LogPhaseChange(*updatedState.RunID, phaseNum, string(colony.PhaseCompleted), "codex-build-complete")
		for _, dispatch := range dispatches {
			filesModified := 0
			if dispatch.Status == "completed" {
				for _, d := range dispatches {
					if d.Name == dispatch.Name {
						filesModified = len(d.Outputs)
						break
					}
				}
			}
			_ = tracer.LogArtifact(*updatedState.RunID, "build.worker", map[string]interface{}{
				"worker":         dispatch.Name,
				"status":         dispatch.Status,
				"files_modified": filesModified,
				"summary":        dispatch.Summary,
			})
		}
	}
	if err := store.SaveJSON("COLONY_STATE.json", updatedState); err != nil {
		return nil, fmt.Errorf("failed to save built colony state: %w", err)
	}

	updateSessionSummary("build", "aether continue", fmt.Sprintf("Phase %d dispatched to %d workers across %d waves", phaseNum, len(dispatches), max(waveCount, 1)))

	dispatchMaps := make([]map[string]interface{}, 0, len(dispatches))
	for _, dispatch := range dispatches {
		entry := map[string]interface{}{
			"stage":  dispatch.Stage,
			"caste":  dispatch.Caste,
			"name":   dispatch.Name,
			"task":   dispatch.Task,
			"status": dispatch.Status,
		}
		if dispatch.Wave > 0 {
			entry["wave"] = dispatch.Wave
		}
		if dispatch.TaskID != "" {
			entry["task_id"] = dispatch.TaskID
		}
		if len(dispatch.DependsOn) > 0 {
			entry["depends_on"] = dispatch.DependsOn
		}
		if len(dispatch.Outputs) > 0 {
			entry["outputs"] = dispatch.Outputs
		}
		if dispatch.Summary != "" {
			entry["summary"] = dispatch.Summary
		}
		if dispatch.Duration > 0 {
			entry["duration"] = dispatch.Duration
		}
		if len(dispatch.Blockers) > 0 {
			entry["blockers"] = dispatch.Blockers
		}
		dispatchMaps = append(dispatchMaps, entry)
	}

	result := map[string]interface{}{
		"phase":          phaseNum,
		"phase_name":     updatedPhase.Name,
		"state":          updatedState.State,
		"playbooks":      playbooks,
		"next":           "aether continue",
		"currentTask":    updatedPhase.Tasks,
		"dispatches":     dispatchMaps,
		"dispatch_count": len(dispatches),
		"wave_count":     waveCount,
		"parallel_waves": parallelWaves,
		"parallel_mode":  string(parallelMode),
		"wave_execution": waveExecution,
		"dispatch_mode":  mode,
		"selected_tasks": selectedTaskIDs,
		"checkpoint":     displayDataPath(checkpointRel),
		"build_dir":      displayDataPath(buildDirRel),
		"manifest":       displayDataPath(manifestRel),
		"worker_briefs":  briefPaths,
		"claims_path":    displayDataPath(claimsRel),
	}
	runStatus = dispatchRunStatus(dispatches)
	return result, nil
}

func validateCodexBuildState(state colony.ColonyState, phaseNum int, selectedTaskIDs []string) error {
	retryBuiltPhase := false
	recoveryBuild := len(selectedTaskIDs) > 0 && state.CurrentPhase == phaseNum
	switch state.State {
	case colony.StateEXECUTING:
		if recoveryBuild {
			return nil
		}
		if state.CurrentPhase > 0 {
			return fmt.Errorf("phase %d is already active; run `aether continue` before dispatching another build", state.CurrentPhase)
		}
		return fmt.Errorf("a build is already in progress; run `aether continue` before dispatching phase %d", phaseNum)
	case colony.StateBUILT:
		if recoveryBuild {
			return nil
		}
		if canRetryBuiltPhase(state, phaseNum) {
			retryBuiltPhase = true
			break
		}
		if state.CurrentPhase > 0 {
			return fmt.Errorf("phase %d is already built; run `aether continue` before dispatching another build", state.CurrentPhase)
		}
		return fmt.Errorf("a build is waiting for verification; run `aether continue` before dispatching phase %d", phaseNum)
	}

	for i := 0; i < phaseNum-1; i++ {
		if state.Plan.Phases[i].Status != colony.PhaseCompleted {
			return fmt.Errorf("phase %d is not complete yet; build phases in order", state.Plan.Phases[i].ID)
		}
	}

	selected := state.Plan.Phases[phaseNum-1]
	if selected.Status == colony.PhaseCompleted {
		return fmt.Errorf("phase %d is already completed", phaseNum)
	}

	if retryBuiltPhase {
		return nil
	}
	if err := colony.Transition(state.State, colony.StateEXECUTING); err != nil {
		return err
	}
	return nil
}

func validateSelectedBuildTasks(phase colony.Phase, selectedTaskIDs []string) error {
	if len(selectedTaskIDs) == 0 {
		return nil
	}
	known := make(map[string]struct{}, len(phase.Tasks))
	for idx, task := range phase.Tasks {
		known[buildTaskID(task, idx)] = struct{}{}
	}
	unknown := make([]string, 0, len(selectedTaskIDs))
	for _, taskID := range selectedTaskIDs {
		if _, ok := known[taskID]; !ok {
			unknown = append(unknown, taskID)
		}
	}
	if len(unknown) > 0 {
		return fmt.Errorf("unknown task id(s) for phase %d: %s", phase.ID, strings.Join(unknown, ", "))
	}
	return nil
}

func canRetryBuiltPhase(state colony.ColonyState, phaseNum int) bool {
	if state.State != colony.StateBUILT || state.CurrentPhase != phaseNum {
		return false
	}
	manifest := loadCodexContinueManifest(phaseNum)
	if !manifest.Present {
		return true
	}
	if manifestUsesSyntheticDispatch(manifest) {
		return true
	}
	if !allDispatchesCompleted(manifest) {
		return true
	}
	if !manifestRequiresBuilderClaims(manifest) {
		return false
	}
	claims, ok := loadCodexBuildClaims()
	if !ok || claims.BuildPhase != phaseNum {
		return true
	}
	return countCodexBuildClaimPaths(claims) == 0
}

func loadCodexBuildClaims() (codexBuildClaims, bool) {
	var claims codexBuildClaims
	if store == nil {
		return codexBuildClaims{}, false
	}
	if err := store.LoadJSON("last-build-claims.json", &claims); err != nil {
		return codexBuildClaims{}, false
	}
	return claims, true
}

func countCodexBuildClaimPaths(claims codexBuildClaims) int {
	total := 0
	for _, values := range [][]string{claims.FilesCreated, claims.FilesModified, claims.TestsWritten} {
		for _, value := range values {
			if strings.TrimSpace(value) != "" {
				total++
			}
		}
	}
	return total
}

func applyCodexBuildState(state *colony.ColonyState, phaseNum int, startedAt time.Time, selectedTaskIDs []string) {
	state.State = colony.StateEXECUTING
	state.CurrentPhase = phaseNum
	state.BuildStartedAt = &startedAt

	for i := range state.Plan.Phases {
		switch {
		case state.Plan.Phases[i].ID < phaseNum && state.Plan.Phases[i].Status != colony.PhaseCompleted:
			state.Plan.Phases[i].Status = colony.PhaseCompleted
		case state.Plan.Phases[i].ID == phaseNum:
			state.Plan.Phases[i].Status = colony.PhaseInProgress
			applyBuildTaskStatuses(&state.Plan.Phases[i], selectedTaskIDs)
		case state.Plan.Phases[i].Status == "":
			state.Plan.Phases[i].Status = colony.PhasePending
		}
	}

	phase := state.Plan.Phases[phaseNum-1]
	state.Events = append(trimmedEvents(state.Events),
		fmt.Sprintf("%s|phase_started|build|Phase %d: %s", startedAt.Format(time.RFC3339), phaseNum, phase.Name),
		fmt.Sprintf("%s|build_dispatched|build|Dispatched %d workers for phase %d", startedAt.Format(time.RFC3339), len(plannedBuildDispatchesForSelection(phase, normalizedBuildDepth(state.ColonyDepth), selectedTaskIDs)), phaseNum),
	)

	if tracer != nil && state.RunID != nil {
		_ = tracer.LogPhaseChange(*state.RunID, phaseNum, string(colony.PhaseInProgress), "codex-build-start")
	}
}

func applyBuildTaskStatuses(phase *colony.Phase, selectedTaskIDs []string) {
	selected := make(map[string]struct{}, len(selectedTaskIDs))
	for _, taskID := range selectedTaskIDs {
		selected[taskID] = struct{}{}
	}
	if len(selected) > 0 {
		for i := range phase.Tasks {
			if phase.Tasks[i].Status == colony.TaskCompleted {
				continue
			}
			if _, ok := selected[buildTaskID(phase.Tasks[i], i)]; ok {
				phase.Tasks[i].Status = colony.TaskInProgress
				continue
			}
			if phase.Tasks[i].Status == "" {
				phase.Tasks[i].Status = colony.TaskPending
			}
		}
		return
	}

	waves := taskWaves(phase.Tasks)
	firstWave := map[int]bool{}
	if len(waves) > 0 {
		for _, idx := range waves[0] {
			firstWave[idx] = true
		}
	}

	for i := range phase.Tasks {
		if phase.Tasks[i].Status == colony.TaskCompleted {
			continue
		}
		if firstWave[i] {
			phase.Tasks[i].Status = colony.TaskInProgress
			continue
		}
		if phase.Tasks[i].Status == "" {
			phase.Tasks[i].Status = colony.TaskPending
		}
	}
}

func codexBuildPlaybooks() []string {
	return []string{
		".aether/docs/command-playbooks/build-prep.md",
		".aether/docs/command-playbooks/build-wave.md",
		".aether/docs/command-playbooks/build-verify.md",
		".aether/docs/command-playbooks/build-complete.md",
	}
}

func plannedBuildDispatches(phase colony.Phase, depth string) []codexBuildDispatch {
	return plannedBuildDispatchesForSelection(phase, depth, nil)
}

func plannedBuildDispatchesForSelection(phase colony.Phase, depth string, selectedTaskIDs []string) []codexBuildDispatch {
	depth = normalizedBuildDepth(depth)
	selected := make(map[string]struct{}, len(selectedTaskIDs))
	for _, taskID := range selectedTaskIDs {
		selected[taskID] = struct{}{}
	}
	waves := taskWaves(phase.Tasks)
	dispatches := make([]codexBuildDispatch, 0, len(phase.Tasks)+4)

	for waveIdx, wave := range waves {
		for _, taskIdx := range wave {
			task := phase.Tasks[taskIdx]
			taskID := buildTaskID(task, taskIdx)
			if len(selected) > 0 {
				if _, ok := selected[taskID]; !ok {
					continue
				}
			}
			dispatches = append(dispatches, codexBuildDispatch{
				Stage:     "wave",
				Wave:      waveIdx + 1,
				Caste:     suggestedBuildCaste(task),
				Name:      deterministicAntName(suggestedBuildCaste(task), fmt.Sprintf("phase:%d:task:%d:%s", phase.ID, taskIdx, task.Goal)),
				Task:      strings.TrimSpace(task.Goal),
				Status:    "spawned",
				TaskID:    taskID,
				TaskIndex: taskIdx,
				DependsOn: append([]string{}, task.DependsOn...),
			})
		}
	}

	if len(waves) == 0 && len(selected) == 0 {
		dispatches = append(dispatches, codexBuildDispatch{
			Stage:  "wave",
			Wave:   1,
			Caste:  "builder",
			Name:   deterministicAntName("builder", fmt.Sprintf("phase:%d:default", phase.ID)),
			Task:   "Build the phase objective",
			Status: "spawned",
		})
	}

	if len(selected) == 0 && (depth == "deep" || depth == "full") {
		dispatches = append(dispatches,
			codexBuildDispatch{
				Stage:  "strategy",
				Caste:  "oracle",
				Name:   deterministicAntName("oracle", fmt.Sprintf("phase:%d:oracle", phase.ID)),
				Task:   "Phase research and implementation risks",
				Status: "spawned",
			},
			codexBuildDispatch{
				Stage:  "strategy",
				Caste:  "architect",
				Name:   deterministicAntName("architect", fmt.Sprintf("phase:%d:architect", phase.ID)),
				Task:   "Design boundaries before coding",
				Status: "spawned",
			},
		)
	}

	dispatches = append(dispatches, codexBuildDispatch{
		Stage:  "verification",
		Caste:  "watcher",
		Name:   deterministicAntName("watcher", fmt.Sprintf("phase:%d:watcher", phase.ID)),
		Task:   "Independent verification before advancement",
		Status: "spawned",
	})
	if depth == "full" {
		dispatches = append(dispatches, codexBuildDispatch{
			Stage:  "resilience",
			Caste:  "chaos",
			Name:   deterministicAntName("chaos", fmt.Sprintf("phase:%d:chaos", phase.ID)),
			Task:   "Resilience probing after verification",
			Status: "spawned",
		})
	}

	return dispatches
}

func buildWaveExecutionPlans(dispatches []codexBuildDispatch, parallelMode colony.ParallelMode) []codexWaveExecutionPlan {
	waveCounts := make(map[int]int)
	for _, dispatch := range dispatches {
		if dispatch.Stage != "wave" || dispatch.Wave <= 0 {
			continue
		}
		waveCounts[dispatch.Wave]++
	}
	if len(waveCounts) == 0 {
		return nil
	}

	waves := make([]int, 0, len(waveCounts))
	for wave := range waveCounts {
		waves = append(waves, wave)
	}
	sort.Ints(waves)

	plans := make([]codexWaveExecutionPlan, 0, len(waves))
	for _, wave := range waves {
		plans = append(plans, buildWaveExecutionPlan(wave, waveCounts[wave], parallelMode))
	}
	return plans
}

func buildWaveExecutionPlan(wave, workerCount int, parallelMode colony.ParallelMode) codexWaveExecutionPlan {
	plan := codexWaveExecutionPlan{
		Wave:        wave,
		WorkerCount: workerCount,
		Strategy:    "serial",
	}
	switch {
	case workerCount <= 1:
		plan.Reason = "single task in this wave"
	case parallelMode == colony.ModeWorktree:
		plan.Strategy = "parallel"
		plan.Reason = "dependency-independent tasks run in isolated worktrees"
	default:
		plan.Reason = "dependency-independent tasks share the main working tree in in-repo mode"
	}
	return plan
}

func countParallelWaveExecutionPlans(plans []codexWaveExecutionPlan) int {
	total := 0
	for _, plan := range plans {
		if plan.Strategy == "parallel" {
			total++
		}
	}
	return total
}

func normalizedBuildDepth(depth string) string {
	depth = strings.TrimSpace(depth)
	if depth == "" {
		return "standard"
	}
	return depth
}

func buildTaskID(task colony.Task, idx int) string {
	if task.ID != nil && strings.TrimSpace(*task.ID) != "" {
		return strings.TrimSpace(*task.ID)
	}
	return fmt.Sprintf("task-%d", idx+1)
}

func executeCodexBuildDispatches(ctx context.Context, root string, phase colony.Phase, dispatches []codexBuildDispatch, playbooks []string, startedAt time.Time, invoker codex.WorkerInvoker, parallelMode colony.ParallelMode) ([]codexBuildDispatch, *codex.ClaimsSummary, string, error) {
	if invoker == nil {
		invoker = &codex.FakeInvoker{}
	}
	if _, ok := invoker.(*codex.FakeInvoker); !ok && !invoker.IsAvailable(ctx) {
		return nil, nil, "", dispatchUnavailableError(invoker)
	}

	capsule := resolveCodexWorkerContext()
	pheromoneSection := resolvePheromoneSection()
	workerDispatches := make([]codex.WorkerDispatch, 0, len(dispatches))
	indexByName := make(map[string]int, len(dispatches))
	for i, dispatch := range dispatches {
		agentName := codexAgentNameForCaste(dispatch.Caste)
		workerDispatches = append(workerDispatches, codex.WorkerDispatch{
			ID:               fmt.Sprintf("phase-%d-dispatch-%d", phase.ID, i+1),
			WorkerName:       dispatch.Name,
			AgentName:        agentName,
			AgentTOMLPath:    dispatchAgentPath(root, invoker, agentName),
			Caste:            dispatch.Caste,
			TaskID:           normalizedDispatchTaskID(dispatch),
			TaskBrief:        renderCodexBuildWorkerBrief(root, phase, dispatch, playbooks, startedAt),
			ContextCapsule:   capsule,
			SkillSection:     resolveSkillSection(dispatch.Caste, dispatch.Task),
			PheromoneSection: pheromoneSection,
			Root:             root,
			Wave:             normalizedDispatchWave(dispatch),
		})
		indexByName[dispatch.Name] = i
	}

	results, err := dispatchCodexBuildWorkers(ctx, root, phase, workerDispatches, invoker, startedAt, parallelMode)
	// Clean up any worktrees that weren't properly finalized during dispatch
	cleaned, orphaned, _ := cleanupBuildWorktrees(phase.ID)
	if cleaned > 0 || orphaned > 0 {
		emitVisualProgress(fmt.Sprintf("Worktree cleanup: %d cleaned, %d orphaned", cleaned, orphaned))
	}
	if err != nil {
		return nil, nil, "", fmt.Errorf("dispatch build workers: %w", err)
	}

	mode := "real"
	if _, ok := invoker.(*codex.FakeInvoker); ok {
		mode = "simulated"
	}
	for _, result := range results {
		idx, ok := indexByName[result.WorkerName]
		if !ok {
			continue
		}
		dispatches[idx].Status = result.Status
		if dispatches[idx].Status == "" {
			dispatches[idx].Status = "failed"
		}
		if result.WorkerResult != nil {
			dispatches[idx].Summary = strings.TrimSpace(result.WorkerResult.Summary)
			dispatches[idx].Blockers = append([]string{}, result.WorkerResult.Blockers...)
			dispatches[idx].Duration = result.WorkerResult.Duration.Seconds()
		}
		if result.Error != nil && len(dispatches[idx].Blockers) == 0 {
			dispatches[idx].Blockers = []string{result.Error.Error()}
		}
	}

	claims := codex.ExtractClaims(results)
	return dispatches, claims, mode, nil
}

func writeCodexBuildArtifacts(root string, state colony.ColonyState, phase colony.Phase, buildDirRel, checkpointRel, claimsRel string, playbooks []string, dispatches []codexBuildDispatch, startedAt time.Time, dispatchMode string, selectedTaskIDs []string) ([]string, []codexBuildDispatch, error) {
	briefPaths := make([]string, 0, len(dispatches))
	briefOutputs := map[string]string{}

	for i := range dispatches {
		briefRel := filepath.ToSlash(filepath.Join(buildDirRel, "worker-briefs", fmt.Sprintf("%s.md", dispatches[i].Name)))
		content := renderCodexBuildWorkerBrief(root, phase, dispatches[i], playbooks, startedAt)
		if err := store.AtomicWrite(briefRel, []byte(content)); err != nil {
			return nil, nil, fmt.Errorf("failed to write worker brief for %s: %w", dispatches[i].Name, err)
		}
		displayPath := displayDataPath(briefRel)
		briefPaths = append(briefPaths, displayPath)
		briefOutputs[dispatches[i].Name] = displayPath
	}
	sort.Strings(briefPaths)

	taskPlans := make([]codexBuildTaskPlan, 0, len(phase.Tasks))
	waves := taskWaves(phase.Tasks)
	taskWave := map[int]int{}
	for waveIdx, wave := range waves {
		for _, idx := range wave {
			taskWave[idx] = waveIdx + 1
		}
	}
	for idx, task := range phase.Tasks {
		taskPlans = append(taskPlans, codexBuildTaskPlan{
			ID:        buildTaskID(task, idx),
			Goal:      task.Goal,
			Status:    task.Status,
			Wave:      taskWave[idx],
			DependsOn: append([]string{}, task.DependsOn...),
		})
	}

	goal := ""
	if state.Goal != nil {
		goal = strings.TrimSpace(*state.Goal)
	}
	for i := range dispatches {
		if output := briefOutputs[dispatches[i].Name]; output != "" {
			dispatches[i].Outputs = []string{output}
		}
	}

	manifest := codexBuildManifest{
		Phase:           phase.ID,
		PhaseName:       phase.Name,
		Goal:            goal,
		Root:            root,
		ParallelMode:    string(effectiveParallelMode(state)),
		WaveExecution:   buildWaveExecutionPlans(dispatches, effectiveParallelMode(state)),
		ColonyDepth:     normalizedBuildDepth(state.ColonyDepth),
		DispatchMode:    strings.TrimSpace(dispatchMode),
		GeneratedAt:     startedAt.Format(time.RFC3339),
		State:           string(state.State),
		Checkpoint:      displayDataPath(checkpointRel),
		ClaimsPath:      displayDataPath(claimsRel),
		Playbooks:       append([]string{}, playbooks...),
		WorkerBriefs:    briefPaths,
		Dispatches:      dispatches,
		SelectedTasks:   append([]string{}, selectedTaskIDs...),
		Tasks:           taskPlans,
		SuccessCriteria: append([]string{}, phase.SuccessCriteria...),
	}
	manifestRel := filepath.ToSlash(filepath.Join(buildDirRel, "manifest.json"))
	if err := store.SaveJSON(manifestRel, manifest); err != nil {
		return nil, nil, fmt.Errorf("failed to write build manifest: %w", err)
	}

	return briefPaths, dispatches, nil
}

func rollbackCodexBuildFailure(previous colony.ColonyState, phaseNum int, startedAt time.Time, dispatchErr error) {
	if store == nil {
		return
	}

	rollback := previous
	summary := fmt.Sprintf("Build dispatch for phase %d failed", phaseNum)
	if dispatchErr != nil {
		summary = strings.TrimSpace(dispatchErr.Error())
		rollback.Events = append(trimmedEvents(rollback.Events),
			fmt.Sprintf("%s|build_dispatch_failed|build|Phase %d dispatch failed: %s", startedAt.Format(time.RFC3339), phaseNum, summary),
		)
	}

	if tracer != nil && rollback.RunID != nil {
		_ = tracer.LogPhaseChange(*rollback.RunID, phaseNum, "failed", "codex-build-fail")
	}

	if err := store.SaveJSON("COLONY_STATE.json", rollback); err != nil {
		return
	}
	_, _ = syncColonyArtifacts(rollback, colonyArtifactOptions{
		CommandName:   "build",
		SuggestedNext: nextCommandFromState(rollback),
		Summary:       summary,
		SafeToClear:   "YES — Build dispatch failed and state was restored",
		HandoffTitle:  "Build Dispatch Failed",
		WriteHandoff:  true,
	})
}

func cloneColonyState(state colony.ColonyState) (colony.ColonyState, error) {
	data, err := json.Marshal(state)
	if err != nil {
		return colony.ColonyState{}, err
	}
	var cloned colony.ColonyState
	if err := json.Unmarshal(data, &cloned); err != nil {
		return colony.ColonyState{}, err
	}
	return cloned, nil
}

func renderCodexBuildWorkerBrief(root string, phase colony.Phase, dispatch codexBuildDispatch, playbooks []string, startedAt time.Time) string {
	var b strings.Builder
	b.WriteString("# Codex Build Dispatch\n\n")
	b.WriteString(fmt.Sprintf("- Worker: %s\n", dispatch.Name))
	b.WriteString(fmt.Sprintf("- Caste: %s\n", dispatch.Caste))
	if dispatch.Wave > 0 {
		b.WriteString(fmt.Sprintf("- Wave: %d\n", dispatch.Wave))
	}
	b.WriteString(fmt.Sprintf("- Phase: %d — %s\n", phase.ID, phase.Name))
	b.WriteString(fmt.Sprintf("- Started: %s\n", startedAt.Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("- Workspace: %s\n", root))
	b.WriteString("\n## Assignment\n\n")
	b.WriteString(strings.TrimSpace(dispatch.Task))
	b.WriteString("\n")

	if strings.TrimSpace(phase.Description) != "" {
		b.WriteString("\n## Phase Objective\n\n")
		b.WriteString(strings.TrimSpace(phase.Description))
		b.WriteString("\n")
	}

	if len(dispatch.DependsOn) > 0 {
		b.WriteString("\n## Dependencies\n\n")
		for _, dep := range dispatch.DependsOn {
			dep = strings.TrimSpace(dep)
			if dep == "" {
				continue
			}
			b.WriteString("- ")
			b.WriteString(dep)
			b.WriteString("\n")
		}
	}

	relatedTask := findDispatchTask(phase, dispatch)
	if relatedTask != nil {
		if len(relatedTask.Constraints) > 0 {
			b.WriteString("\n## Constraints\n\n")
			for _, item := range relatedTask.Constraints {
				item = strings.TrimSpace(item)
				if item == "" {
					continue
				}
				b.WriteString("- ")
				b.WriteString(item)
				b.WriteString("\n")
			}
		}
		if len(relatedTask.Hints) > 0 {
			b.WriteString("\n## Hints\n\n")
			for _, item := range relatedTask.Hints {
				item = strings.TrimSpace(item)
				if item == "" {
					continue
				}
				b.WriteString("- ")
				b.WriteString(item)
				b.WriteString("\n")
			}
		}
		if len(relatedTask.SuccessCriteria) > 0 {
			b.WriteString("\n## Task Success Criteria\n\n")
			for _, item := range relatedTask.SuccessCriteria {
				item = strings.TrimSpace(item)
				if item == "" {
					continue
				}
				b.WriteString("- ")
				b.WriteString(item)
				b.WriteString("\n")
			}
		}
	}

	if len(phase.SuccessCriteria) > 0 {
		b.WriteString("\n## Phase Success Criteria\n\n")
		for _, item := range phase.SuccessCriteria {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			b.WriteString("- ")
			b.WriteString(item)
			b.WriteString("\n")
		}
	}

	b.WriteString("\n## Relevant Playbooks\n\n")
	for _, playbook := range buildPlaybooksForDispatch(dispatch, playbooks) {
		b.WriteString("- ")
		b.WriteString(playbook)
		b.WriteString("\n")
	}

	b.WriteString("\n## Expected Output\n\n")
	b.WriteString("- ")
	b.WriteString(expectedDispatchOutcome(dispatch))
	b.WriteString("\n")
	return b.String()
}

func findDispatchTask(phase colony.Phase, dispatch codexBuildDispatch) *colony.Task {
	if dispatch.TaskID == "" {
		return nil
	}
	for i := range phase.Tasks {
		if buildTaskID(phase.Tasks[i], i) == dispatch.TaskID {
			return &phase.Tasks[i]
		}
	}
	return nil
}

func buildPlaybooksForDispatch(dispatch codexBuildDispatch, playbooks []string) []string {
	filtered := make([]string, 0, len(playbooks))
	for _, playbook := range playbooks {
		switch dispatch.Caste {
		case "oracle", "architect":
			if strings.Contains(playbook, "build-prep") || strings.Contains(playbook, "build-wave") {
				filtered = append(filtered, playbook)
			}
		case "watcher", "chaos":
			if strings.Contains(playbook, "build-verify") || strings.Contains(playbook, "build-complete") {
				filtered = append(filtered, playbook)
			}
		default:
			if strings.Contains(playbook, "build-wave") || strings.Contains(playbook, "build-complete") {
				filtered = append(filtered, playbook)
			}
		}
	}
	if len(filtered) == 0 {
		return append([]string{}, playbooks...)
	}
	return filtered
}

func expectedDispatchOutcome(dispatch codexBuildDispatch) string {
	switch dispatch.Caste {
	case "scout":
		return "Research notes or documentation updates that unblock implementation."
	case "watcher":
		return "Independent verification notes with concrete evidence for `aether continue`."
	case "oracle":
		return "Implementation risks, unknowns, and recommended handling before deeper coding."
	case "architect":
		return "Design boundaries, interfaces, and sequencing guidance for the phase."
	case "chaos":
		return "Resilience findings and failure cases worth checking before advancement."
	default:
		return "Concrete code changes plus a truthful summary of files touched and verification run."
	}
}

func writeCodexBuildClaims(relPath string, phaseNum int, startedAt time.Time, summary *codex.ClaimsSummary) error {
	claims := codexBuildClaims{BuildPhase: phaseNum, Timestamp: startedAt.Format(time.RFC3339)}
	if summary != nil {
		claims.FilesCreated = append([]string{}, summary.FilesCreated...)
		claims.FilesModified = append([]string{}, summary.FilesModified...)
		claims.TestsWritten = append([]string{}, summary.TestsWritten...)
		if len(summary.TaskClaims) > 0 {
			claims.TaskClaims = make([]codexBuildTaskClaim, 0, len(summary.TaskClaims))
			for _, taskClaim := range summary.TaskClaims {
				claims.TaskClaims = append(claims.TaskClaims, codexBuildTaskClaim{
					TaskID:        taskClaim.TaskID,
					FilesCreated:  append([]string{}, taskClaim.FilesCreated...),
					FilesModified: append([]string{}, taskClaim.FilesModified...),
					TestsWritten:  append([]string{}, taskClaim.TestsWritten...),
				})
			}
		}
	}
	if err := store.SaveJSON(relPath, claims); err != nil {
		return fmt.Errorf("failed to write build claims: %w", err)
	}
	return nil
}

func recordCodexBuildDispatches(dispatches []codexBuildDispatch) error {
	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	for _, dispatch := range dispatches {
		if err := spawnTree.RecordSpawn("Queen", dispatch.Caste, dispatch.Name, dispatch.Task, 1); err != nil {
			return fmt.Errorf("failed to record build dispatch %s: %w", dispatch.Name, err)
		}
	}
	return nil
}

func dispatchRunStatus(dispatches []codexBuildDispatch) string {
	statuses := make([]string, 0, len(dispatches))
	for _, dispatch := range dispatches {
		statuses = append(statuses, dispatch.Status)
	}
	return summarizeRunStatus(statuses...)
}

func ensureUniqueBuildDispatchNames(dispatches []codexBuildDispatch) ([]codexBuildDispatch, error) {
	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	entries, err := spawnTree.Parse()
	if err != nil {
		return nil, fmt.Errorf("failed to read spawn tree for name allocation: %w", err)
	}

	used := make(map[string]bool, len(entries)+len(dispatches))
	for _, entry := range entries {
		used[entry.AgentName] = true
	}

	allocated := make([]codexBuildDispatch, len(dispatches))
	for i, dispatch := range dispatches {
		candidate := dispatch.Name
		if used[candidate] {
			base := candidate
			for attempt := 2; ; attempt++ {
				candidate = fmt.Sprintf("%s-r%d", base, attempt)
				if !used[candidate] {
					break
				}
			}
		}
		dispatch.Name = candidate
		used[candidate] = true
		allocated[i] = dispatch
	}
	return allocated, nil
}

func updateCodexBuildContext(phase colony.Phase, dispatches []codexBuildDispatch, parallelWaves int, startedAt time.Time) error {
	data, err := readContextDocument()
	if err != nil {
		return nil
	}

	content := string(data)
	content = replaceContextTableRow(content, "Last Updated", startedAt.Format(time.RFC3339))
	content = replaceContextTableRow(content, "Current Phase", strconv.Itoa(phase.ID))
	content = replaceContextTableRow(content, "Phase Name", phase.Name)
	content = replaceContextTableRow(content, "Safe to Clear?", "NO — Build in progress")
	content = replaceContextSectionContent(content, "What's In Progress", fmt.Sprintf(
		"**Phase %d Build IN PROGRESS**\n- Workers: %d | Tasks: %d | Waves: %d\n- Phase: %s\n- Started: %s",
		phase.ID, len(dispatches), len(phase.Tasks), max(parallelWaves, 1), phase.Name, startedAt.Format(time.RFC3339),
	))
	for _, dispatch := range dispatches {
		content = appendWorkerSpawnEntry(content, dispatch.Name, dispatch.Caste, dispatch.Task, startedAt.Format(time.RFC3339))
	}

	return writeContextDocument(content)
}

func displayDataPath(rel string) string {
	return filepath.ToSlash(filepath.Join(".aether", "data", rel))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func codexAgentFileForCaste(caste string) string {
	normalized := strings.ToLower(strings.TrimSpace(strings.ReplaceAll(caste, "_", "-")))
	if normalized == "" {
		normalized = "builder"
	}
	return "aether-" + normalized + ".toml"
}

func codexAgentNameForCaste(caste string) string {
	return strings.TrimSuffix(codexAgentFileForCaste(caste), ".toml")
}

func normalizedDispatchWave(dispatch codexBuildDispatch) int {
	if dispatch.Wave > 0 {
		return dispatch.Wave
	}
	switch dispatch.Stage {
	case "strategy":
		return 1
	case "verification":
		return 100
	case "resilience":
		return 101
	default:
		return 1
	}
}

func normalizedDispatchTaskID(dispatch codexBuildDispatch) string {
	if strings.TrimSpace(dispatch.TaskID) != "" {
		return strings.TrimSpace(dispatch.TaskID)
	}
	parts := []string{strings.TrimSpace(dispatch.Stage), strings.TrimSpace(dispatch.Caste), strings.TrimSpace(dispatch.Name)}
	joined := strings.ToLower(strings.Join(parts, "-"))
	joined = strings.ReplaceAll(joined, " ", "-")
	return strings.Trim(joined, "-")
}

func resolveSkillSectionResult(caste, task string) skillInjectResult {
	return renderSkillInjectResult(matchSkills(resolveHubPath(), caste, task))
}

// resolveSkillSection matches skills for the given role and task through the
// shared runtime resolver and returns the rendered markdown section.
func resolveSkillSection(caste, task string) string {
	return resolveSkillSectionResult(caste, task).SkillSection
}

// resolvePheromoneSection extracts active pheromone signals, groups them by
// type, and formats them into a markdown section. Returns empty string if no signals
// or if the store is not initialized.
func resolvePheromoneSection() string {
	if store == nil {
		return ""
	}
	texts := extractSignalTexts(8)
	if len(texts) == 0 {
		return ""
	}

	var focus, redirect, feedback []string
	for _, text := range texts {
		switch {
		case strings.HasPrefix(text, "FOCUS:"):
			focus = append(focus, strings.TrimPrefix(text, "FOCUS:"))
		case strings.HasPrefix(text, "REDIRECT:"):
			redirect = append(redirect, strings.TrimPrefix(text, "REDIRECT:"))
		case strings.HasPrefix(text, "FEEDBACK:"):
			feedback = append(feedback, strings.TrimPrefix(text, "FEEDBACK:"))
		}
	}

	var b strings.Builder
	b.WriteString("### Active Pheromone Signals\n\n")
	if len(focus) > 0 {
		b.WriteString("**FOCUS:**\n")
		for _, f := range focus {
			b.WriteString(fmt.Sprintf("- %s\n", strings.TrimSpace(f)))
		}
		b.WriteString("\n")
	}
	if len(redirect) > 0 {
		b.WriteString("**REDIRECT:**\n")
		for _, r := range redirect {
			b.WriteString(fmt.Sprintf("- %s\n", strings.TrimSpace(r)))
		}
		b.WriteString("\n")
	}
	if len(feedback) > 0 {
		b.WriteString("**FEEDBACK:**\n")
		for _, f := range feedback {
			b.WriteString(fmt.Sprintf("- %s\n", strings.TrimSpace(f)))
		}
	}
	return strings.TrimSpace(b.String())
}
