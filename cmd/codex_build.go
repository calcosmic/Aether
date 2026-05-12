package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

type codexBuildDispatch struct {
	Stage          string   `json:"stage"`
	Wave           int      `json:"wave,omitempty"`
	ExecutionWave  int      `json:"execution_wave,omitempty"`
	Caste          string   `json:"caste"`
	Name           string   `json:"name"`
	Task           string   `json:"task"`
	Status         string   `json:"status"`
	Summary        string   `json:"summary,omitempty"`
	TaskID         string   `json:"task_id,omitempty"`
	TaskIndex      int      `json:"task_index,omitempty"`
	DependsOn      []string `json:"depends_on,omitempty"`
	Outputs        []string `json:"outputs,omitempty"`
	Blockers       []string `json:"blockers,omitempty"`
	Duration       float64  `json:"duration,omitempty"`
	SkillSection   string   `json:"skill_section,omitempty"`
	SkillCount     int      `json:"skill_count,omitempty"`
	ColonySkills   int      `json:"colony_skill_count,omitempty"`
	DomainSkills   int      `json:"domain_skill_count,omitempty"`
	MatchedSkills  []string `json:"matched_skills,omitempty"`
	HandoffSection string   `json:"handoff_section,omitempty"`
}

type codexBuildTaskPlan struct {
	ID        string   `json:"id,omitempty"`
	Goal      string   `json:"goal"`
	Status    string   `json:"status"`
	Wave      int      `json:"wave,omitempty"`
	DependsOn []string `json:"depends_on,omitempty"`
}

type codexBuildManifest struct {
	Phase                     int                              `json:"phase"`
	PhaseName                 string                           `json:"phase_name"`
	Goal                      string                           `json:"goal,omitempty"`
	Root                      string                           `json:"root"`
	ColonyMode                string                           `json:"colony_mode,omitempty"`
	PlanOnly                  bool                             `json:"plan_only,omitempty"`
	ParallelMode              string                           `json:"parallel_mode,omitempty"`
	WaveExecution             []codexWaveExecutionPlan         `json:"wave_execution,omitempty"`
	ExecutionPlan             []codexBuildExecutionPlan        `json:"execution_plan,omitempty"`
	ColonyDepth               string                           `json:"colony_depth"`
	DispatchMode              string                           `json:"dispatch_mode,omitempty"`
	HostPlatform              string                           `json:"host_platform,omitempty"`
	ExecutionOwner            string                           `json:"execution_owner,omitempty"`
	WorkerDispatchOptIn       bool                             `json:"worker_dispatch_opt_in,omitempty"`
	GeneratedAt               string                           `json:"generated_at"`
	State                     string                           `json:"state"`
	Checkpoint                string                           `json:"checkpoint"`
	ClaimsPath                string                           `json:"claims_path"`
	Playbooks                 []string                         `json:"playbooks"`
	WorkerBriefs              []string                         `json:"worker_briefs"`
	Dispatches                []codexBuildDispatch             `json:"dispatches"`
	SelectedTasks             []string                         `json:"selected_tasks,omitempty"`
	Tasks                     []codexBuildTaskPlan             `json:"tasks"`
	SuccessCriteria           []string                         `json:"success_criteria"`
	ReviewDepth               string                           `json:"review_depth,omitempty"`
	DispatchContract          map[string]interface{}           `json:"dispatch_contract,omitempty"`
	ProfileContract           codexWorkflowProfileContract     `json:"profile_contract,omitempty"`
	QueenRecommendation       codexQueenWorkflowRecommendation `json:"queen_recommendation,omitempty"`
	QueenExecutionPolicy      codexQueenExecutionPolicy        `json:"queen_execution_policy,omitempty"`
	BoundaryQuestions         []discussQuestion                `json:"boundary_questions,omitempty"`
	BoundaryQuestionCount     int                              `json:"boundary_question_count,omitempty"`
	BoundaryQuestionsCreated  int                              `json:"boundary_questions_created,omitempty"`
	BoundaryQuestionsExisting int                              `json:"boundary_questions_existing,omitempty"`
	OrchestratorGuidance      *orchestratorBoundaryGuidance    `json:"orchestrator_boundary_guidance,omitempty"`
}

type codexWaveExecutionPlan struct {
	Wave        int    `json:"wave"`
	Strategy    string `json:"strategy"`
	WorkerCount int    `json:"worker_count"`
	Reason      string `json:"reason"`
}

type codexBuildExecutionPlan struct {
	ExecutionWave int      `json:"execution_wave"`
	Stage         string   `json:"stage"`
	Wave          int      `json:"wave,omitempty"`
	Strategy      string   `json:"strategy"`
	WorkerCount   int      `json:"worker_count"`
	Castes        []string `json:"castes,omitempty"`
	Reason        string   `json:"reason,omitempty"`
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

var errRuntimeStateSuperseded = errors.New("runtime state superseded")

type codexBuildOptions struct {
	WorkerTimeout           time.Duration
	ParentContext           context.Context
	Force                   bool
	LightFlag               bool
	HeavyFlag               bool
	VerificationDepth       string
	DispatchWorkers         bool
	CircuitBreakerThreshold int
	Verbose                 bool
}

func runCodexBuildPlanOnly(root string, phaseNum int, selectedTaskIDs []string) (map[string]interface{}, colony.ColonyState, colony.Phase, []codexBuildDispatch, error) {
	return runCodexBuildPlanOnlyWithOptions(root, phaseNum, selectedTaskIDs, codexBuildOptions{})
}

func runCodexBuildPlanOnlyWithOptions(root string, phaseNum int, selectedTaskIDs []string, options codexBuildOptions) (map[string]interface{}, colony.ColonyState, colony.Phase, []codexBuildDispatch, error) {
	if store == nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("no store initialized")
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
	state, err = reconcilePriorCompletedPhaseTasksForPlanOnly(root, state, phaseNum)
	if err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}
	selectedTaskIDs = uniqueSortedStrings(selectedTaskIDs)
	phase := state.Plan.Phases[phaseNum-1]
	if err := validateSelectedBuildTasks(phase, selectedTaskIDs); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}
	if err := runPreBuildGates(store.BasePath(), phaseNum); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}
	if err := validateCodexBuildState(state, phaseNum, selectedTaskIDs, options.Force); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}

	generatedAt := time.Now().UTC()
	playbooks := codexBuildPlaybooks()
	policy := recommendQueenExecutionPolicy(state, phase, len(state.Plan.Phases), codexQueenExecutionPolicyInput{
		LightFlag:         options.LightFlag,
		HeavyFlag:         options.HeavyFlag,
		VerificationDepth: options.VerificationDepth,
		WorkerTimeout:     options.WorkerTimeout,
		DispatchWorkers:   options.DispatchWorkers,
	})
	reviewDepth := colony.NormalizeVerificationDepth(policy.VerificationDepth)
	dispatches := plannedBuildDispatchesForSelectionWithState(phase, state, selectedTaskIDs, reviewDepth)
	for i := range dispatches {
		dispatches[i].Status = "planned"
	}
	dispatches, err = ensureUniqueBuildDispatchNames(dispatches)
	if err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}
	attachBuildDispatchContext(phase.ID, dispatches)

	parallelMode := effectiveParallelMode(state)
	waveExecution := buildWaveExecutionPlans(dispatches, parallelMode)
	executionPlan := buildExecutionPlans(dispatches, parallelMode)
	dispatchContract := buildDispatchContractForDispatches(dispatches, parallelMode, options.WorkerTimeout)
	manifest := buildCodexBuildManifest(root, state, phase, "", "", playbooks, dispatches, generatedAt, "plan-only", selectedTaskIDs, nil, true, reviewDepth)
	manifest.DispatchContract = dispatchContract
	profileContract := workflowProfileContract(reviewDepth)
	queenRecommendation := recommendQueenWorkflowProfile(state, phase, len(state.Plan.Phases))
	manifest.QueenExecutionPolicy = policy
	boundary, err := materializeOrchestratorBoundaryQuestions("build", state, phase, buildBoundaryQuestionCandidates(phase, selectedTaskIDs))
	if err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}
	manifest.BoundaryQuestions = boundary.Questions
	manifest.BoundaryQuestionCount = len(boundary.Questions)
	manifest.BoundaryQuestionsCreated = boundary.Created
	manifest.BoundaryQuestionsExisting = boundary.Existing

	result := map[string]interface{}{
		"plan_only":                true,
		"phase":                    phaseNum,
		"colony_mode":              string(state.EffectiveColonyMode()),
		"review_depth":             string(reviewDepth),
		"phase_name":               phase.Name,
		"state":                    state.State,
		"playbooks":                playbooks,
		"next":                     "spawn wrapper agents from dispatches, then record completion",
		"currentTask":              phase.Tasks,
		"dispatches":               codexBuildDispatchMaps(dispatches),
		"dispatch_manifest":        manifest,
		"dispatch_count":           len(dispatches),
		"wave_count":               len(waveExecution),
		"parallel_waves":           countParallelWaveExecutionPlans(waveExecution),
		"parallel_mode":            string(parallelMode),
		"wave_execution":           waveExecution,
		"execution_plan":           executionPlan,
		"execution_wave_count":     len(executionPlan),
		"parallel_execution_waves": countParallelBuildExecutionPlans(executionPlan),
		"dispatch_mode":            "plan-only",
		"dispatch_contract":        dispatchContract,
		"host_platform":            string(codex.DetectActivePlatform()),
		"execution_owner":          buildExecutionOwner("plan-only", true),
		"profile_contract":         profileContract,
		"queen_recommendation":     queenRecommendation,
		"queen_execution_policy":   policy,
		"selected_tasks":           selectedTaskIDs,
		"wrapper_contract": map[string]interface{}{
			"source_command":          "AETHER_OUTPUT_MODE=json aether build <phase> --plan-only",
			"spawn_log_required":      true,
			"spawn_complete_required": true,
			"finalize_surface":        "awaiting_wrapper_completion",
		},
	}
	addBoundaryQuestionResultFields(result, boundary)
	if guidance, ok := addOrchestratorBoundaryGuidance(result, "build", state, fmt.Sprintf("aether build %d", phaseNum), boundary.Questions); ok {
		manifest.OrchestratorGuidance = &guidance
		result["dispatch_manifest"] = manifest
	}
	return result, state, phase, dispatches, nil
}

func runCodexBuildQueenLed(root string, phaseNum int, selectedTaskIDs []string, options codexBuildOptions) (map[string]interface{}, colony.ColonyState, colony.Phase, []codexBuildDispatch, error) {
	result, state, phase, dispatches, err := runCodexBuildPlanOnlyWithOptions(root, phaseNum, selectedTaskIDs, options)
	if err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}
	reviewDepth := reviewDepthFromResult(result)
	profileContract := workflowProfileContract(reviewDepth)
	queenRecommendation := recommendQueenWorkflowProfile(state, phase, len(state.Plan.Phases))
	policy := recommendQueenExecutionPolicy(state, phase, len(state.Plan.Phases), codexQueenExecutionPolicyInput{
		LightFlag:         options.LightFlag,
		HeavyFlag:         options.HeavyFlag,
		VerificationDepth: options.VerificationDepth,
		WorkerTimeout:     options.WorkerTimeout,
		DispatchWorkers:   false,
	})

	if manifest, ok := result["dispatch_manifest"].(codexBuildManifest); ok {
		manifest.DispatchMode = "queen-led"
		manifest.ExecutionOwner = buildExecutionOwner("queen-led", true)
		manifest.ProfileContract = profileContract
		manifest.QueenRecommendation = queenRecommendation
		manifest.QueenExecutionPolicy = policy
		result["dispatch_manifest"] = manifest
	}
	result["queen_led"] = true
	result["dispatch_mode"] = "queen-led"
	result["execution_owner"] = buildExecutionOwner("queen-led", true)
	result["profile_contract"] = profileContract
	result["queen_recommendation"] = queenRecommendation
	result["queen_execution_policy"] = policy
	result["next"] = "Queen/main agent executes dispatch_manifest, then runs aether build-finalize"
	result["wrapper_contract"] = map[string]interface{}{
		"source_command":          "aether build <phase>",
		"spawn_log_required":      true,
		"spawn_complete_required": true,
		"finalize_surface":        "awaiting_queen_completion",
		"worker_dispatch_opt_in":  "--dispatch-workers",
	}
	return result, state, phase, dispatches, nil
}

func runCodexBuild(root string, phaseNum int, selectedTaskIDs []string, synthetic bool) (map[string]interface{}, error) {
	return runCodexBuildWithOptions(root, phaseNum, selectedTaskIDs, synthetic, codexBuildOptions{})
}

func runCodexBuildWithOptions(root string, phaseNum int, selectedTaskIDs []string, synthetic bool, options codexBuildOptions) (map[string]interface{}, error) {
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
	state, _, err = reconcilePriorCompletedPhaseTasksFromTrustedManifests(root, state, phaseNum)
	if err != nil {
		return nil, err
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
	if err := validateCodexBuildState(state, phaseNum, selectedTaskIDs, options.Force); err != nil {
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

	policy := recommendQueenExecutionPolicy(state, phase, len(state.Plan.Phases), codexQueenExecutionPolicyInput{
		LightFlag:         options.LightFlag,
		HeavyFlag:         options.HeavyFlag,
		VerificationDepth: options.VerificationDepth,
		WorkerTimeout:     options.WorkerTimeout,
		DispatchWorkers:   true,
	})
	reviewDepth := colony.NormalizeVerificationDepth(policy.VerificationDepth)
	playbooks := codexBuildPlaybooks()
	dispatches := plannedBuildDispatchesForSelectionWithState(phase, state, selectedTaskIDs, reviewDepth)
	dispatches, err = ensureUniqueBuildDispatchNames(dispatches)
	if err != nil {
		return nil, err
	}
	parallelMode := effectiveParallelMode(state)
	waveExecution := buildWaveExecutionPlans(dispatches, parallelMode)
	executionPlan := buildExecutionPlans(dispatches, parallelMode)
	waveCount := len(waveExecution)
	parallelWaves := countParallelWaveExecutionPlans(waveExecution)
	executionWaveCount := len(executionPlan)
	parallelExecutionWaves := countParallelBuildExecutionPlans(executionPlan)
	dispatchContract := buildDispatchContractForDispatches(dispatches, parallelMode, options.WorkerTimeout)

	parentCtx := options.ParentContext
	if parentCtx == nil {
		parentCtx = context.Background()
	}
	ctx, cancel := signal.NotifyContext(parentCtx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	ceremony := newBuildCeremonyEmitter(ctx, root, phase)
	restoreCeremony := setActiveBuildCeremony(ceremony)
	defer restoreCeremony()
	defer ceremony.Close()
	emitBuildCeremonyPrewave(phase, dispatches, waveCount)

	// Ceremony progress tracking (visual mode only)
	var progress *ceremonyProgress
	if shouldRenderVisualOutput(stdout) {
		buildSteps := []string{"Prepare", "Context", "Dispatch", "Verify", "Complete"}
		progress = NewCeremonyProgress(buildSteps, stdout)
	}

	checkpointRel := filepath.ToSlash(filepath.Join("checkpoints", fmt.Sprintf("pre-build-phase-%d.json", phaseNum)))
	buildDirRel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phaseNum)))
	manifestRel := filepath.ToSlash(filepath.Join(buildDirRel, "manifest.json"))
	claimsRel := "last-build-claims.json"

	cleanupStaleBuildAttemptArtifacts(phaseNum)

	if err := store.SaveJSON(checkpointRel, state); err != nil {
		return nil, fmt.Errorf("failed to checkpoint colony state: %w", err)
	}

	updatedState := state
	applyCodexBuildState(&updatedState, phaseNum, startedAt, selectedTaskIDs, reviewDepth)
	updatedPhase := updatedState.Plan.Phases[phaseNum-1]
	if err := store.SaveJSON("COLONY_STATE.json", updatedState); err != nil {
		return nil, fmt.Errorf("failed to save colony state: %w", err)
	}
	if progress != nil {
		progress.Advance("Prepare")
	}

	briefPaths, dispatches, err := writeCodexBuildArtifacts(root, updatedState, updatedPhase, buildDirRel, checkpointRel, claimsRel, playbooks, dispatches, startedAt, "", selectedTaskIDs, reviewDepth, policy)
	if err != nil {
		rollbackCodexBuildFailure(originalState, phaseNum, startedAt, err)
		return nil, err
	}
	if err := recordCodexBuildDispatches(dispatches); err != nil {
		rollbackCodexBuildFailure(originalState, phaseNum, startedAt, err)
		return nil, err
	}
	emitVisualProgress(renderBuildDispatchPreview(updatedState, updatedPhase, dispatches))
	if progress != nil {
		progress.Advance("Context")
	}

	buildInvoker := newCodexWorkerInvoker()
	if synthetic {
		buildInvoker = &codex.FakeInvoker{}
	}
	if progress != nil {
		progress.Advance("Dispatch")
	}
	dispatches, claims, mode, err := executeCodexBuildDispatches(ctx, root, updatedPhase, dispatches, playbooks, startedAt, buildInvoker, parallelMode, options.WorkerTimeout, options.CircuitBreakerThreshold, options.Verbose)
	if err != nil {
		rollbackCodexBuildFailure(originalState, phaseNum, startedAt, err)
		return nil, err
	}
	if err := writeCodexBuildClaims(claimsRel, phaseNum, startedAt, claims); err != nil {
		return nil, err
	}
	updatedState.State = colony.StateBUILT
	reconcileCompletedBuildTasks(&updatedState, phaseNum, dispatches)
	updatedPhase = updatedState.Plan.Phases[phaseNum-1]
	if _, _, err := writeCodexBuildArtifacts(root, updatedState, updatedPhase, buildDirRel, checkpointRel, claimsRel, playbooks, dispatches, startedAt, mode, selectedTaskIDs, reviewDepth, policy); err != nil {
		return nil, err
	}

	var committedState colony.ColonyState
	if err := store.UpdateJSONAtomically("COLONY_STATE.json", &committedState, func() error {
		if err := validateRuntimeStateStillCurrent(committedState, phaseNum, &startedAt, colony.StateEXECUTING); err != nil {
			return err
		}
		committedState.State = colony.StateBUILT
		reconcileCompletedBuildTasks(&committedState, phaseNum, dispatches)
		committedState.Events = append(trimmedEvents(committedState.Events),
			fmt.Sprintf("%s|build_completed|build|Phase %d build packet prepared (%s dispatch)", startedAt.Format(time.RFC3339), phaseNum, mode),
		)
		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to save built colony state: %w", err)
	}
	updatedState = committedState
	updatedPhase = updatedState.Plan.Phases[phaseNum-1]
	if progress != nil {
		progress.Advance("Verify")
	}

	if tracer != nil && updatedState.RunID != nil {
		_ = tracer.LogPhaseChange(*updatedState.RunID, phaseNum, string(colony.PhaseCompleted), "codex-build-complete")
		for _, dispatch := range dispatches {
			filesModified := 0
			if dispatch.Status == "completed" {
				filesModified = len(dispatch.Outputs)
			}
			_ = tracer.LogArtifact(*updatedState.RunID, "build.worker", map[string]interface{}{
				"worker":         dispatch.Name,
				"status":         dispatch.Status,
				"files_modified": filesModified,
				"summary":        dispatch.Summary,
			})
		}
	}
	updateSessionSummary("build", "aether continue", fmt.Sprintf("Phase %d dispatched to %d workers across %d waves", phaseNum, len(dispatches), max(waveCount, 1)))
	if progress != nil {
		progress.Advance("Complete")
		progress.Finish()
	}

	dispatchMaps := codexBuildDispatchMaps(dispatches)

	result := map[string]interface{}{
		"phase":                    phaseNum,
		"colony_mode":              string(updatedState.EffectiveColonyMode()),
		"review_depth":             string(reviewDepth),
		"phase_name":               updatedPhase.Name,
		"state":                    updatedState.State,
		"playbooks":                playbooks,
		"next":                     "aether continue",
		"currentTask":              updatedPhase.Tasks,
		"dispatches":               dispatchMaps,
		"dispatch_count":           len(dispatches),
		"wave_count":               waveCount,
		"parallel_waves":           parallelWaves,
		"parallel_mode":            string(parallelMode),
		"wave_execution":           waveExecution,
		"execution_plan":           executionPlan,
		"execution_wave_count":     executionWaveCount,
		"parallel_execution_waves": parallelExecutionWaves,
		"dispatch_mode":            mode,
		"dispatch_contract":        dispatchContract,
		"queen_execution_policy":   policy,
		"force":                    options.Force,
		"selected_tasks":           selectedTaskIDs,
		"checkpoint":               displayDataPath(checkpointRel),
		"build_dir":                displayDataPath(buildDirRel),
		"manifest":                 displayDataPath(manifestRel),
		"worker_briefs":            briefPaths,
		"claims_path":              displayDataPath(claimsRel),
	}
	runStatus = dispatchRunStatus(dispatches)
	return result, nil
}

func validateCodexBuildState(state colony.ColonyState, phaseNum int, selectedTaskIDs []string, force bool) error {
	retryBuiltPhase := false
	forceActivePhase := force && state.CurrentPhase == phaseNum
	if force && !forceActivePhase {
		if state.CurrentPhase > 0 {
			return fmt.Errorf("--force can only redispatch the active phase %d", state.CurrentPhase)
		}
		return fmt.Errorf("--force can only redispatch an active phase")
	}
	recoveryBuild := (len(selectedTaskIDs) > 0 || forceActivePhase) && state.CurrentPhase == phaseNum
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
		if !phaseTasksAllCompleted(state.Plan.Phases[i]) {
			incomplete := incompletePhaseTaskSummary(state.Plan.Phases[i])
			return fmt.Errorf("phase %d is marked completed but has incomplete task records (%s); run `aether build %d --force` to regenerate trusted task evidence", state.Plan.Phases[i].ID, incomplete, state.Plan.Phases[i].ID)
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

func applyCodexBuildState(state *colony.ColonyState, phaseNum int, startedAt time.Time, selectedTaskIDs []string, reviewDepth colony.VerificationDepth) {
	state.State = colony.StateEXECUTING
	state.CurrentPhase = phaseNum
	state.BuildStartedAt = &startedAt

	for i := range state.Plan.Phases {
		switch {
		case state.Plan.Phases[i].ID < phaseNum && state.Plan.Phases[i].Status != colony.PhaseCompleted:
			if phaseTasksAllCompleted(state.Plan.Phases[i]) {
				state.Plan.Phases[i].Status = colony.PhaseCompleted
			}
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
		fmt.Sprintf("%s|build_dispatched|build|Dispatched %d workers for phase %d", startedAt.Format(time.RFC3339), len(plannedBuildDispatchesForSelectionWithState(phase, *state, selectedTaskIDs, reviewDepth)), phaseNum),
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
	return plannedBuildDispatchesForSelection(phase, depth, nil, colony.VerificationDepthLight)
}

func plannedBuildDispatchesForSelection(phase colony.Phase, depth string, selectedTaskIDs []string, reviewDepth colony.VerificationDepth) []codexBuildDispatch {
	state := colony.ColonyState{
		ColonyDepth:       normalizedBuildDepth(depth),
		VerificationDepth: string(reviewDepth),
	}
	return plannedBuildDispatchesForSelectionWithState(phase, state, selectedTaskIDs, reviewDepth)
}

func plannedBuildDispatchesForSelectionWithState(phase colony.Phase, state colony.ColonyState, selectedTaskIDs []string, reviewDepth colony.VerificationDepth) []codexBuildDispatch {
	depth := normalizedBuildDepth(state.ColonyDepth)
	selected := make(map[string]struct{}, len(selectedTaskIDs))
	for _, taskID := range selectedTaskIDs {
		selected[taskID] = struct{}{}
	}
	waves := taskWaves(phase.Tasks)
	taskWaveBase := 10
	lastTaskExecutionWave := taskWaveBase + max(len(waves), 1)
	dispatches := make([]codexBuildDispatch, 0, len(phase.Tasks)+8)
	queenState := state
	queenState.ColonyDepth = depth
	queenState.VerificationDepth = string(reviewDepth)
	queenCastes := queenBuildCasteSet(queenOrchestrate(phase, "build", queenState))
	applyBuildDispatchPolicyCastes(queenCastes, phase, depth, reviewDepth)

	if len(selected) == 0 {
		dispatches = append(dispatches, queenBuildPreWaveDispatches(phase, queenCastes)...)
	}

	for waveIdx, wave := range waves {
		for _, taskIdx := range wave {
			task := phase.Tasks[taskIdx]
			taskID := buildTaskID(task, taskIdx)
			if len(selected) > 0 {
				if _, ok := selected[taskID]; !ok {
					continue
				}
			}
			caste := queenBuildTaskCaste(task, queenCastes)
			dispatches = append(dispatches, codexBuildDispatch{
				Stage:         "wave",
				Wave:          waveIdx + 1,
				ExecutionWave: taskWaveBase + waveIdx + 1,
				Caste:         caste,
				Name:          deterministicAntName(caste, fmt.Sprintf("phase:%d:task:%d:%s", phase.ID, taskIdx, task.Goal)),
				Task:          strings.TrimSpace(task.Goal),
				Status:        "spawned",
				TaskID:        taskID,
				TaskIndex:     taskIdx,
				DependsOn:     append([]string{}, task.DependsOn...),
			})
		}
	}

	if len(waves) == 0 && len(selected) == 0 {
		caste := "builder"
		if !queenCastes[caste] {
			caste = queenBuildFallbackTaskCaste(queenCastes)
		}
		dispatches = append(dispatches, codexBuildDispatch{
			Stage:         "wave",
			Wave:          1,
			ExecutionWave: taskWaveBase + 1,
			Caste:         caste,
			Name:          deterministicAntName(caste, fmt.Sprintf("phase:%d:default", phase.ID)),
			Task:          "Build the phase objective",
			Status:        "spawned",
		})
	}

	nextVerificationWave := lastTaskExecutionWave + 1
	if len(selected) == 0 && queenCastes["probe"] {
		dispatches = append(dispatches, codexBuildSpecialistDispatch(phase, "probe", nextVerificationWave, "probe", "Independent probe verification of builder claims"))
		nextVerificationWave++
	}
	if len(selected) == 0 {
		postWaveDispatches := queenBuildPostWaveDispatches(phase, queenCastes, nextVerificationWave)
		dispatches = append(dispatches, postWaveDispatches...)
		nextVerificationWave += len(postWaveDispatches)
	}

	if queenCastes["watcher"] {
		dispatches = append(dispatches, codexBuildDispatch{
			Stage:         "verification",
			ExecutionWave: nextVerificationWave,
			Caste:         "watcher",
			Name:          deterministicAntName("watcher", fmt.Sprintf("phase:%d:watcher", phase.ID)),
			Task:          "Independent verification before advancement" + findingsInjectionForCaste("watcher"),
			Status:        "spawned",
		})
	}

	return dispatches
}

func queenBuildCasteSet(dispatches []CasteDispatch) map[string]bool {
	castes := make(map[string]bool, len(dispatches))
	for _, dispatch := range dispatches {
		caste := strings.TrimSpace(dispatch.Caste)
		if caste == "" {
			continue
		}
		castes[caste] = true
	}
	return castes
}

func applyBuildDispatchPolicyCastes(queenCastes map[string]bool, phase colony.Phase, depth string, reviewDepth colony.VerificationDepth) {
	delete(queenCastes, "measurer")
	delete(queenCastes, "chaos")

	if (depth == "deep" || depth == "full") && reviewDepth == colony.VerificationDepthHeavy {
		queenCastes["measurer"] = true
	}
	if phase.Mode != colony.PhaseModeDiscovery {
		if depth == "full" && reviewDepth == colony.VerificationDepthHeavy {
			queenCastes["chaos"] = true
		}
		if reviewDepth == colony.VerificationDepthLight && chaosShouldRunInLightMode(phase.ID) {
			queenCastes["chaos"] = true
		}
	}
}

func queenBuildPreWaveDispatches(phase colony.Phase, queenCastes map[string]bool) []codexBuildDispatch {
	plans := []struct {
		caste string
		stage string
		wave  int
		task  string
	}{
		{"archaeologist", "prep", 1, "Git history analysis before implementation"},
		{"oracle", "research", 2, "Phase research and implementation risks"},
		{"architect", "design", 3, "Design boundaries before coding"},
		{"ambassador", "integration", 4, "External integration design before implementation"},
		{"gatekeeper", "security", 5, "Security boundaries and auth risk review before implementation"},
		{"includer", "accessibility", 6, "Accessibility requirements and inclusive interaction review"},
		{"weaver", "refactor", 7, "Refactoring seams and simplification plan before implementation"},
		{"tracker", "diagnosis", 7, "Root-cause investigation and regression context before implementation"},
		{"keeper", "knowledge", 8, "Knowledge preservation plan for reusable patterns"},
		{"chronicler", "documentation", 8, "Documentation surface and changelog planning"},
		{"medic", "health", 8, "Runtime health and repair risk review"},
		{"fixer", "repair", 8, "Repair strategy and remediation boundaries"},
		{"porter", "delivery", 8, "Delivery, packaging, and release handling review"},
		{"sage", "wisdom", 8, "Learning synthesis and reusable pattern capture"},
	}
	dispatches := make([]codexBuildDispatch, 0, len(plans))
	for _, plan := range plans {
		if !queenCastes[plan.caste] {
			continue
		}
		dispatches = append(dispatches, codexBuildSpecialistDispatch(phase, plan.stage, plan.wave, plan.caste, plan.task+findingsInjectionForCaste(plan.caste)))
	}
	return dispatches
}

func queenBuildPostWaveDispatches(phase colony.Phase, queenCastes map[string]bool, startExecutionWave int) []codexBuildDispatch {
	plans := []struct {
		caste string
		stage string
		task  string
	}{
		{"auditor", "audit", "Quality and compliance review after implementation"},
		{"measurer", "measurement", "Performance and cost surface review after implementation"},
		{"chaos", "resilience", "Resilience probing after specialist verification"},
	}
	dispatches := make([]codexBuildDispatch, 0, len(plans))
	executionWave := startExecutionWave
	for _, plan := range plans {
		if !queenCastes[plan.caste] {
			continue
		}
		dispatches = append(dispatches, codexBuildSpecialistDispatch(phase, plan.stage, executionWave, plan.caste, plan.task+findingsInjectionForCaste(plan.caste)))
		executionWave++
	}
	return dispatches
}

func queenBuildTaskCaste(task colony.Task, queenCastes map[string]bool) string {
	caste := suggestedBuildCaste(task)
	if queenCastes[caste] {
		return caste
	}
	return queenBuildFallbackTaskCaste(queenCastes)
}

func queenBuildFallbackTaskCaste(queenCastes map[string]bool) string {
	for _, caste := range []string{"builder", "scout", "oracle", "weaver", "tracker", "fixer"} {
		if queenCastes[caste] {
			return caste
		}
	}
	return "builder"
}

// findingsInjectionForCaste returns findings-path injection text for review castes
// dispatched during the build flow. Non-review castes return empty string.
// Per D-05, the agent body has generic guardrails; this adds the concrete path.
func findingsInjectionForCaste(caste string) string {
	domainMap := map[string][]string{
		"watcher":       {"testing", "quality"},
		"chaos":         {"resilience"},
		"measurer":      {"performance"},
		"archaeologist": {"history"},
	}
	domains, ok := domainMap[caste]
	if !ok {
		return ""
	}
	return fmt.Sprintf("\n\nPersist your %s findings to the domain review ledger using: aether review-ledger-write --domain <domain> --phase <N> --findings '<json>' --agent %s",
		strings.Join(domains, " and "), caste)
}

func codexBuildSpecialistDispatch(phase colony.Phase, stage string, executionWave int, caste string, task string) codexBuildDispatch {
	return codexBuildDispatch{
		Stage:         stage,
		ExecutionWave: executionWave,
		Caste:         caste,
		Name:          deterministicAntName(caste, fmt.Sprintf("phase:%d:%s", phase.ID, caste)),
		Task:          strings.TrimSpace(task),
		Status:        "spawned",
	}
}

func phaseNeedsAmbassador(phase colony.Phase, selected map[string]struct{}) bool {
	var parts []string
	parts = append(parts, phase.Name, phase.Description)
	parts = append(parts, phase.SuccessCriteria...)
	for idx, task := range phase.Tasks {
		taskID := buildTaskID(task, idx)
		if len(selected) > 0 {
			if _, ok := selected[taskID]; !ok {
				continue
			}
		}
		parts = append(parts, task.Goal)
		parts = append(parts, task.Constraints...)
		parts = append(parts, task.Hints...)
		parts = append(parts, task.SuccessCriteria...)
	}
	text := strings.ToLower(strings.Join(parts, " "))
	for _, token := range []string{"api", "sdk", "oauth", "external service", "external integration", "integration", "webhook", "third-party", "stripe", "sendgrid", "twilio", "openai", "aws", "azure", "gcp"} {
		if strings.Contains(text, token) {
			return true
		}
	}
	return false
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

func buildExecutionPlans(dispatches []codexBuildDispatch, parallelMode colony.ParallelMode) []codexBuildExecutionPlan {
	grouped := make(map[int][]codexBuildDispatch)
	for _, dispatch := range dispatches {
		wave := normalizedDispatchWave(dispatch)
		if wave <= 0 {
			wave = 1
		}
		grouped[wave] = append(grouped[wave], dispatch)
	}
	if len(grouped) == 0 {
		return nil
	}

	executionWaves := make([]int, 0, len(grouped))
	for wave := range grouped {
		executionWaves = append(executionWaves, wave)
	}
	sort.Ints(executionWaves)

	plans := make([]codexBuildExecutionPlan, 0, len(executionWaves))
	for _, executionWave := range executionWaves {
		dispatches := grouped[executionWave]
		stage := dispatches[0].Stage
		taskWave := dispatches[0].Wave
		castes := make([]string, 0, len(dispatches))
		seenCastes := map[string]struct{}{}
		for _, dispatch := range dispatches {
			if dispatch.Stage != stage {
				stage = "mixed"
			}
			if taskWave != dispatch.Wave {
				taskWave = 0
			}
			caste := strings.TrimSpace(dispatch.Caste)
			if caste == "" {
				continue
			}
			if _, ok := seenCastes[caste]; ok {
				continue
			}
			seenCastes[caste] = struct{}{}
			castes = append(castes, caste)
		}
		sort.Strings(castes)
		plans = append(plans, codexBuildExecutionPlan{
			ExecutionWave: executionWave,
			Stage:         stage,
			Wave:          taskWave,
			Strategy:      executionStrategyForBuildStep(stage, len(dispatches), parallelMode),
			WorkerCount:   len(dispatches),
			Castes:        castes,
			Reason:        executionReasonForBuildStep(stage, taskWave, len(dispatches), parallelMode),
		})
	}
	return plans
}

func executionStrategyForBuildStep(stage string, workerCount int, parallelMode colony.ParallelMode) string {
	if stage == "wave" && workerCount > 1 && parallelMode == colony.ModeWorktree {
		return "parallel"
	}
	return "serial"
}

func executionReasonForBuildStep(stage string, taskWave int, workerCount int, parallelMode colony.ParallelMode) string {
	switch stage {
	case "prep":
		return "pre-wave git history and risk context"
	case "research":
		return "pre-wave research before design and implementation"
	case "design":
		return "pre-wave architecture design before implementation"
	case "integration":
		return "external integration design before implementation"
	case "security":
		return "security and auth review before implementation"
	case "accessibility":
		return "accessibility requirements review before implementation"
	case "refactor":
		return "refactoring strategy before implementation"
	case "diagnosis":
		return "root-cause investigation before implementation"
	case "knowledge":
		return "knowledge preservation before implementation"
	case "documentation":
		return "documentation planning before implementation"
	case "health":
		return "runtime health review before implementation"
	case "repair":
		return "repair strategy before implementation"
	case "delivery":
		return "delivery and packaging review before implementation"
	case "wisdom":
		return "learning synthesis before implementation"
	case "wave":
		if taskWave > 0 {
			return buildWaveExecutionPlan(taskWave, workerCount, parallelMode).Reason
		}
		return "builder/scout task wave"
	case "probe":
		return "post-wave independent verification of builder claims"
	case "verification":
		return "post-wave watcher verification before advancement"
	case "audit":
		return "post-wave quality and compliance review"
	case "measurement":
		return "post-wave performance and cost review"
	case "resilience":
		return "post-wave resilience probing"
	default:
		return "manifest-defined build step"
	}
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

func countParallelBuildExecutionPlans(plans []codexBuildExecutionPlan) int {
	total := 0
	for _, plan := range plans {
		if plan.Strategy == "parallel" {
			total++
		}
	}
	return total
}

func buildDispatchContractForDispatches(dispatches []codexBuildDispatch, parallelMode colony.ParallelMode, workerTimeout time.Duration) map[string]interface{} {
	executionPlan := buildExecutionPlans(dispatches, parallelMode)
	executionModel := "staged build execution: builder wave(s), specialist verification, watcher verification"
	if len(executionPlan) > 0 {
		executionModel = fmt.Sprintf("%d staged build execution waves: builder wave(s), specialist verification, watcher verification", len(executionPlan))
	}
	contract := codexDispatchContract{
		ExecutionModel:       executionModel,
		WaveCount:            len(executionPlan),
		WorkerCount:          len(dispatches),
		SharedTimeoutSeconds: 0,
		WorkerTimeoutSeconds: int(effectiveBuildDispatchTimeout(workerTimeout) / time.Second),
		DeadlinePolicy:       "Each build worker gets its own timeout. The Queen runtime advances execution_wave stages in order and records each terminal worker result.",
		DependencyBehavior:   "Builder task waves run before post-build specialist verification. Watcher verification is the final build execution stage before finalization.",
		FallbackBehavior:     "Runtime worker dispatch rolls back failed direct builds; host-orchestrated plan-only builds must call build-finalize with fresh worker results.",
		FallbackVisibility:   []string{"dispatch_mode", "dispatches", "execution_plan", "worker_handoffs"},
		CoordinationPath:     dataContractPath("spawn-tree.txt"),
		ArtifactPaths: []string{
			dataContractPath("build", "phase-<phase>", "manifest.json"),
			dataContractPath("build", "phase-<phase>", "worker-briefs"),
			dataContractPath("last-build-claims.json"),
			dataContractPath("handoffs", "worker-handoffs.json"),
		},
	}.asMap()
	contract["execution_plan"] = append([]codexBuildExecutionPlan{}, executionPlan...)
	return contract
}

func effectiveBuildDispatchTimeout(workerTimeout time.Duration) time.Duration {
	if workerTimeout > 0 {
		return workerTimeout
	}
	return codex.DefaultWorkerTimeout
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

func executeCodexBuildDispatches(ctx context.Context, root string, phase colony.Phase, dispatches []codexBuildDispatch, playbooks []string, startedAt time.Time, invoker codex.WorkerInvoker, parallelMode colony.ParallelMode, workerTimeout time.Duration, circuitBreakerThreshold int, verbose bool) ([]codexBuildDispatch, *codex.ClaimsSummary, string, error) {
	if invoker == nil {
		invoker = &codex.FakeInvoker{}
	}
	if _, ok := invoker.(*codex.FakeInvoker); !ok && !invoker.IsAvailable(ctx) {
		return nil, nil, "", dispatchUnavailableError(invoker)
	}

	dataDir := filepath.Join(root, ".aether", "data")
	cleanupAllHeartbeatFiles(dataDir)
	defer cleanupAllHeartbeatFiles(dataDir)

	capsule := resolveCodexWorkerContext()
	cleanupStaleWorkersBeforeDispatch(root)

	pheromoneSection := resolvePheromoneSection()
	workerDispatches := make([]codex.WorkerDispatch, 0, len(dispatches))
	indexByName := make(map[string]int, len(dispatches))
	dispatchByName := make(map[string]codex.WorkerDispatch, len(dispatches))
	for i, dispatch := range dispatches {
		agentName := codexAgentNameForCaste(dispatch.Caste)
		workerDispatch := codex.WorkerDispatch{
			ID:               fmt.Sprintf("phase-%d-dispatch-%d", phase.ID, i+1),
			WorkerName:       dispatch.Name,
			AgentName:        agentName,
			AgentTOMLPath:    dispatchAgentPath(root, invoker, agentName),
			Caste:            dispatch.Caste,
			TaskID:           normalizedDispatchTaskID(dispatch),
			TaskBrief:        renderCodexBuildWorkerBrief(root, phase, dispatch, playbooks, startedAt),
			ContextCapsule:   capsule,
			HandoffSection:   dispatch.HandoffSection,
			Workflow:         "build",
			Phase:            phase.ID,
			SkillSection:     resolveSkillSectionForWorkflow("build", dispatch.Caste, dispatch.Task),
			PheromoneSection: pheromoneSection,
			Root:             root,
			Timeout:          workerTimeout,
			Wave:             normalizedDispatchWave(dispatch),
		}
		workerDispatches = append(workerDispatches, workerDispatch)
		indexByName[dispatch.Name] = i
		dispatchByName[dispatch.Name] = workerDispatch
	}

	cb := NewCircuitBreaker(circuitBreakerThreshold)
	// Per D-02/D-04: set verbose flag before dispatch so filtered functions work correctly
	setBuildVerbose(verbose)

	// Per D-09/D-12: queen owns the wave loop. Build calls queen once.
	waveDispatchFn := func(ctx context.Context, waveDispatches []codex.WorkerDispatch, waveNum int) ([]codex.DispatchResult, error) {
		return dispatchCodexBuildWorkers(ctx, root, phase, waveDispatches, invoker, startedAt, parallelMode, cb)
	}
	summary, results, err := queenWaveLifecycle(ctx, workerDispatches, waveDispatchFn, phase, cb, phase.ID)
	// Persist wave summary JSON for Phase 99 consumption (D-07)
	_ = writeWaveSummary(phase.ID, summary)

	// Per D-05: consolidate queen decisions into audit file (Phase 99 OUT-02)
	audit := consolidateQueenAudit(phase.ID)
	_ = writeAuditFile(phase.ID, audit)

	// Per D-09: render phase-end summary with actions needed (Phase 99 OUT-01)
	renderPhaseEndSummary(summary, phase.ID)

	// Clean up any worktrees that weren't properly finalized during dispatch
	cleaned, orphaned, _ := cleanupBuildWorktrees(phase.ID)
	if cleaned > 0 || orphaned > 0 {
		emitVisualProgress(fmt.Sprintf("Worktree cleanup: %d cleaned, %d orphaned", cleaned, orphaned))
	}
	for _, result := range results {
		if dispatch, ok := dispatchByName[result.WorkerName]; ok {
			_ = persistDispatchWorkerHandoff(dispatch, result)
		}
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
			dispatches[idx].Outputs = buildDispatchClaimOutputs(*result.WorkerResult)
		}
		// Per D-02/D-04: print raw worker output only in verbose mode
		if result.WorkerResult != nil && result.WorkerResult.RawOutput != "" {
			filteredFprintln(stdout, result.WorkerResult.RawOutput)
		}

		if result.Error != nil && len(dispatches[idx].Blockers) == 0 {
			dispatches[idx].Blockers = []string{result.Error.Error()}
		}
	}

	claims := codex.ExtractClaims(results)
	return dispatches, claims, mode, nil
}

func codexBuildTaskPlans(phase colony.Phase) []codexBuildTaskPlan {
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
	return taskPlans
}

func buildCodexBuildManifest(root string, state colony.ColonyState, phase colony.Phase, checkpointRel, claimsRel string, playbooks []string, dispatches []codexBuildDispatch, startedAt time.Time, dispatchMode string, selectedTaskIDs []string, workerBriefs []string, planOnly bool, reviewDepth colony.VerificationDepth) codexBuildManifest {
	goal := ""
	if state.Goal != nil {
		goal = strings.TrimSpace(*state.Goal)
	}

	checkpoint := ""
	if strings.TrimSpace(checkpointRel) != "" {
		checkpoint = displayDataPath(checkpointRel)
	}
	claimsPath := ""
	if strings.TrimSpace(claimsRel) != "" {
		claimsPath = displayDataPath(claimsRel)
	}
	briefs := append([]string{}, workerBriefs...)
	if briefs == nil {
		briefs = []string{}
	}

	return codexBuildManifest{
		Phase:               phase.ID,
		PhaseName:           phase.Name,
		Goal:                goal,
		Root:                root,
		ColonyMode:          string(state.EffectiveColonyMode()),
		PlanOnly:            planOnly,
		ParallelMode:        string(effectiveParallelMode(state)),
		WaveExecution:       buildWaveExecutionPlans(dispatches, effectiveParallelMode(state)),
		ExecutionPlan:       buildExecutionPlans(dispatches, effectiveParallelMode(state)),
		ColonyDepth:         normalizedBuildDepth(state.ColonyDepth),
		DispatchMode:        strings.TrimSpace(dispatchMode),
		HostPlatform:        buildHostPlatform(),
		ExecutionOwner:      buildExecutionOwner(dispatchMode, planOnly),
		WorkerDispatchOptIn: buildWorkerDispatchOptIn(dispatchMode),
		GeneratedAt:         startedAt.Format(time.RFC3339),
		State:               string(state.State),
		Checkpoint:          checkpoint,
		ClaimsPath:          claimsPath,
		Playbooks:           append([]string{}, playbooks...),
		WorkerBriefs:        briefs,
		Dispatches:          append([]codexBuildDispatch{}, dispatches...),
		SelectedTasks:       append([]string{}, selectedTaskIDs...),
		Tasks:               codexBuildTaskPlans(phase),
		SuccessCriteria:     append([]string{}, phase.SuccessCriteria...),
		ReviewDepth:         string(reviewDepth),
		DispatchContract:    buildDispatchContractForDispatches(dispatches, effectiveParallelMode(state), 0),
		ProfileContract:     workflowProfileContract(reviewDepth),
		QueenRecommendation: recommendQueenWorkflowProfile(state, phase, len(state.Plan.Phases)),
		QueenExecutionPolicy: recommendQueenExecutionPolicy(state, phase, len(state.Plan.Phases), codexQueenExecutionPolicyInput{
			VerificationDepth: string(reviewDepth),
			DispatchWorkers:   buildWorkerDispatchOptIn(dispatchMode),
		}),
	}
}

func buildHostPlatform() string {
	platform := codex.DetectActivePlatform()
	if platform == codex.PlatformUnknown {
		return ""
	}
	return string(platform)
}

func buildExecutionOwner(dispatchMode string, planOnly bool) string {
	mode := strings.ToLower(strings.TrimSpace(dispatchMode))
	switch mode {
	case "real", "simulated":
		return "runtime-worker-dispatch"
	case "external-task":
		return "platform-wrapper"
	case "queen-led", "plan-only":
		return "host-queen"
	}
	if planOnly {
		return "host-queen"
	}
	return ""
}

func buildWorkerDispatchOptIn(dispatchMode string) bool {
	switch strings.ToLower(strings.TrimSpace(dispatchMode)) {
	case "real", "simulated":
		return true
	default:
		return false
	}
}

func codexBuildDispatchMaps(dispatches []codexBuildDispatch) []map[string]interface{} {
	dispatchMaps := make([]map[string]interface{}, 0, len(dispatches))
	for _, dispatch := range dispatches {
		entry := map[string]interface{}{
			"stage":          dispatch.Stage,
			"execution_wave": normalizedDispatchWave(dispatch),
			"caste":          dispatch.Caste,
			"agent_name":     codexAgentNameForCaste(dispatch.Caste),
			"name":           dispatch.Name,
			"task":           dispatch.Task,
			"status":         dispatch.Status,
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
		if dispatch.SkillCount > 0 {
			entry["skill_count"] = dispatch.SkillCount
			entry["colony_skill_count"] = dispatch.ColonySkills
			entry["domain_skill_count"] = dispatch.DomainSkills
			entry["matched_skills"] = append([]string{}, dispatch.MatchedSkills...)
			entry["skill_section"] = dispatch.SkillSection
		}
		if strings.TrimSpace(dispatch.HandoffSection) != "" {
			entry["handoff_section"] = dispatch.HandoffSection
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
	return dispatchMaps
}

func writeCodexBuildArtifacts(root string, state colony.ColonyState, phase colony.Phase, buildDirRel, checkpointRel, claimsRel string, playbooks []string, dispatches []codexBuildDispatch, startedAt time.Time, dispatchMode string, selectedTaskIDs []string, reviewDepth colony.VerificationDepth, policy codexQueenExecutionPolicy) ([]string, []codexBuildDispatch, error) {
	briefPaths := make([]string, 0, len(dispatches))
	briefOutputs := map[string]string{}
	finalOutputs := map[string][]string{}

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

	if isFinalBuildDispatchMode(dispatchMode) {
		var err error
		finalOutputs, dispatches, err = writeCodexBuildOutcomeReports(root, phase, buildDirRel, dispatches, time.Now().UTC(), dispatchMode)
		if err != nil {
			return nil, nil, err
		}
	}

	for i := range dispatches {
		if outputs := finalOutputs[dispatches[i].Name]; len(outputs) > 0 {
			dispatches[i].Outputs = outputs
			continue
		}
		if output := briefOutputs[dispatches[i].Name]; output != "" {
			dispatches[i].Outputs = []string{output}
		}
	}

	manifest := buildCodexBuildManifest(root, state, phase, checkpointRel, claimsRel, playbooks, dispatches, startedAt, dispatchMode, selectedTaskIDs, briefPaths, false, reviewDepth)
	manifest.QueenExecutionPolicy = policy
	manifestRel := filepath.ToSlash(filepath.Join(buildDirRel, "manifest.json"))
	if err := store.SaveJSON(manifestRel, manifest); err != nil {
		return nil, nil, fmt.Errorf("failed to write build manifest: %w", err)
	}

	return briefPaths, dispatches, nil
}

func buildDispatchClaimOutputs(result codex.WorkerResult) []string {
	outputs := make([]string, 0, len(result.FilesCreated)+len(result.FilesModified)+len(result.TestsWritten))
	outputs = append(outputs, result.FilesCreated...)
	outputs = append(outputs, result.FilesModified...)
	outputs = append(outputs, result.TestsWritten...)
	return uniqueSortedStrings(outputs)
}

func reconcileCompletedBuildTasks(state *colony.ColonyState, phaseNum int, dispatches []codexBuildDispatch) []string {
	if state == nil || phaseNum < 1 || phaseNum > len(state.Plan.Phases) {
		return nil
	}
	completed := completedBuildTaskIDs(dispatches)
	if len(completed) == 0 {
		return nil
	}
	taskIDs := make([]string, 0, len(completed))
	phase := &state.Plan.Phases[phaseNum-1]
	for idx := range phase.Tasks {
		taskID := buildTaskID(phase.Tasks[idx], idx)
		if _, ok := completed[taskID]; !ok {
			continue
		}
		phase.Tasks[idx].Status = colony.TaskCompleted
		taskIDs = append(taskIDs, taskID)
	}
	return uniqueSortedStrings(taskIDs)
}

func reconcilePriorCompletedPhaseTasksForPlanOnly(root string, state colony.ColonyState, phaseNum int) (colony.ColonyState, error) {
	if phaseNum <= 1 {
		return state, nil
	}
	if _, err := applyPriorCompletedPhaseTaskRepairs(root, &state, phaseNum); err != nil {
		return state, err
	}
	return state, nil
}

func completedBuildTaskIDs(dispatches []codexBuildDispatch) map[string]struct{} {
	completed := map[string]struct{}{}
	for _, dispatch := range dispatches {
		taskID := strings.TrimSpace(dispatch.TaskID)
		if taskID == "" || strings.TrimSpace(dispatch.Status) != "completed" {
			continue
		}
		completed[taskID] = struct{}{}
	}
	return completed
}

func reconcilePriorCompletedPhaseTasksFromTrustedManifests(root string, state colony.ColonyState, phaseNum int) (colony.ColonyState, []string, error) {
	if store == nil || phaseNum <= 1 || len(state.Plan.Phases) == 0 {
		return state, nil, nil
	}

	rehearsal := state
	if repaired, err := applyPriorCompletedPhaseTaskRepairs(root, &rehearsal, phaseNum); err != nil || len(repaired) == 0 {
		return state, repaired, err
	}

	repaired := []string{}
	var updated colony.ColonyState
	if err := store.UpdateJSONAtomically("COLONY_STATE.json", &updated, func() error {
		var err error
		repaired, err = applyPriorCompletedPhaseTaskRepairs(root, &updated, phaseNum)
		return err
	}); err != nil {
		return state, nil, fmt.Errorf("failed to reconcile completed prior phase task statuses: %w", err)
	}
	return updated, repaired, nil
}

func applyPriorCompletedPhaseTaskRepairs(root string, state *colony.ColonyState, phaseNum int) ([]string, error) {
	if state == nil || phaseNum <= 1 {
		return nil, nil
	}

	repaired := []string{}
	limit := phaseNum - 1
	if limit > len(state.Plan.Phases) {
		limit = len(state.Plan.Phases)
	}
	for idx := 0; idx < limit; idx++ {
		phase := &state.Plan.Phases[idx]
		if phase.Status != colony.PhaseCompleted || phaseTasksAllCompleted(*phase) {
			continue
		}
		completed, err := trustedCompletedPhaseTaskEvidence(root, *state, *phase)
		if err != nil {
			return nil, err
		}
		phaseRepaired := []string{}
		for taskIdx := range phase.Tasks {
			taskID := buildTaskID(phase.Tasks[taskIdx], taskIdx)
			if _, ok := completed[taskID]; !ok {
				continue
			}
			if phase.Tasks[taskIdx].Status != colony.TaskCompleted {
				phase.Tasks[taskIdx].Status = colony.TaskCompleted
				phaseRepaired = append(phaseRepaired, taskID)
			}
		}
		if len(phaseRepaired) > 0 {
			repaired = append(repaired, phaseRepaired...)
			state.Events = append(trimmedEvents(state.Events),
				fmt.Sprintf("%s|phase_tasks_repaired|build|Repaired completed task statuses for phase %d from trusted build manifest: %s", time.Now().UTC().Format(time.RFC3339), phase.ID, strings.Join(uniqueSortedStrings(phaseRepaired), ", ")),
			)
		}
	}
	return uniqueSortedStrings(repaired), nil
}

func phaseTasksAllCompleted(phase colony.Phase) bool {
	for _, task := range phase.Tasks {
		if task.Status != colony.TaskCompleted {
			return false
		}
	}
	return true
}

func trustedCompletedPhaseTaskEvidence(root string, state colony.ColonyState, phase colony.Phase) (map[string]struct{}, error) {
	manifest := loadCodexContinueManifest(phase.ID)
	if !manifest.Present {
		return nil, completedPhaseTaskRepairError(phase, "no build manifest was found")
	}
	if err := validateTrustedCompletedPhaseManifest(root, state, phase, manifest); err != nil {
		return nil, completedPhaseTaskRepairError(phase, err.Error())
	}

	stateIDs := phaseTaskIDSet(phase)
	completed := completedTaskEvidenceIDs(manifest.Data)
	missing := missingTaskIDs(stateIDs, completed)
	if len(missing) > 0 {
		return nil, completedPhaseTaskRepairError(phase, fmt.Sprintf("trusted build manifest lacks completed task evidence for %s", strings.Join(missing, ", ")))
	}
	return completed, nil
}

func validateTrustedCompletedPhaseManifest(root string, state colony.ColonyState, phase colony.Phase, manifest codexContinueManifest) error {
	data := manifest.Data
	if data.Phase != phase.ID {
		return fmt.Errorf("build manifest phase %d does not match completed phase %d", data.Phase, phase.ID)
	}
	if data.PlanOnly || strings.EqualFold(strings.TrimSpace(data.DispatchMode), "plan-only") {
		return fmt.Errorf("build manifest for phase %d is plan-only, not a final runtime manifest", phase.ID)
	}
	if isSimulatedBuildDispatchMode(data.DispatchMode) {
		return fmt.Errorf("build manifest for phase %d is simulated and cannot repair persisted task state", phase.ID)
	}
	if !isFinalBuildDispatchMode(data.DispatchMode) && !manifestStateLooksFinal(data.State) {
		return fmt.Errorf("build manifest for phase %d is not final (dispatch_mode: %s)", phase.ID, strings.TrimSpace(data.DispatchMode))
	}
	if strings.TrimSpace(data.PhaseName) != "" && strings.TrimSpace(phase.Name) != "" && strings.TrimSpace(data.PhaseName) != strings.TrimSpace(phase.Name) {
		return fmt.Errorf("build manifest phase name %q does not match COLONY_STATE phase name %q", strings.TrimSpace(data.PhaseName), strings.TrimSpace(phase.Name))
	}
	if state.Goal != nil && strings.TrimSpace(data.Goal) != "" && strings.TrimSpace(*state.Goal) != strings.TrimSpace(data.Goal) {
		return fmt.Errorf("build manifest goal does not match COLONY_STATE goal")
	}
	if strings.TrimSpace(data.Root) != "" && strings.TrimSpace(root) != "" && filepath.Clean(data.Root) != filepath.Clean(root) {
		return fmt.Errorf("build manifest root %q does not match current root %q", filepath.Clean(data.Root), filepath.Clean(root))
	}
	return validateBuildManifestTaskSetForPhase(manifest, phase, false)
}

func manifestStateLooksFinal(state string) bool {
	switch colony.State(strings.TrimSpace(state)) {
	case colony.StateBUILT, colony.StateCOMPLETED:
		return true
	default:
		return false
	}
}

func completedPhaseTaskRepairError(phase colony.Phase, reason string) error {
	incomplete := incompletePhaseTaskSummary(phase)
	if incomplete == "" {
		incomplete = "none"
	}
	return fmt.Errorf("phase %d is marked completed but task rows are incomplete (%s); %s; restore .aether/data/build/phase-%d/manifest.json from the completed run or run `aether build %d --force` to regenerate trusted task evidence before building a later phase", phase.ID, incomplete, reason, phase.ID, phase.ID)
}

func incompletePhaseTaskSummary(phase colony.Phase) string {
	incomplete := []string{}
	for idx, task := range phase.Tasks {
		if task.Status == colony.TaskCompleted {
			continue
		}
		status := strings.TrimSpace(task.Status)
		if status == "" {
			status = colony.TaskPending
		}
		incomplete = append(incomplete, fmt.Sprintf("%s=%s", buildTaskID(task, idx), status))
	}
	return strings.Join(incomplete, ", ")
}

func completedTaskEvidenceIDs(manifest codexBuildManifest) map[string]struct{} {
	completed := completedBuildTaskIDs(manifest.Dispatches)
	for _, task := range manifest.Tasks {
		taskID := strings.TrimSpace(task.ID)
		if taskID == "" || strings.TrimSpace(task.Status) != colony.TaskCompleted {
			continue
		}
		completed[taskID] = struct{}{}
	}
	return completed
}

func phaseTaskIDSet(phase colony.Phase) []string {
	ids := make([]string, 0, len(phase.Tasks))
	for idx, task := range phase.Tasks {
		ids = append(ids, buildTaskID(task, idx))
	}
	return uniqueSortedStrings(ids)
}

func manifestTaskIDSet(tasks []codexBuildTaskPlan) []string {
	ids := make([]string, 0, len(tasks))
	for idx, task := range tasks {
		taskID := strings.TrimSpace(task.ID)
		if taskID == "" {
			taskID = fmt.Sprintf("task-%d", idx+1)
		}
		ids = append(ids, taskID)
	}
	return uniqueSortedStrings(ids)
}

func validateBuildManifestTaskSetForPhase(manifest codexContinueManifest, phase colony.Phase, allowMissingTasks bool) error {
	if !manifest.Present {
		return nil
	}
	if len(manifest.Data.Tasks) == 0 && allowMissingTasks {
		return nil
	}
	manifestIDs := manifestTaskIDSet(manifest.Data.Tasks)
	stateIDs := phaseTaskIDSet(phase)
	if stringSlicesEqual(manifestIDs, stateIDs) {
		return nil
	}
	return fmt.Errorf("phase %d build manifest task set does not match COLONY_STATE (manifest: %s; state: %s)", phase.ID, formatTaskIDSet(manifestIDs), formatTaskIDSet(stateIDs))
}

func stringSlicesEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for idx := range left {
		if left[idx] != right[idx] {
			return false
		}
	}
	return true
}

func missingTaskIDs(ids []string, present map[string]struct{}) []string {
	missing := []string{}
	for _, id := range ids {
		if _, ok := present[id]; !ok {
			missing = append(missing, id)
		}
	}
	return uniqueSortedStrings(missing)
}

func formatTaskIDSet(ids []string) string {
	if len(ids) == 0 {
		return "none"
	}
	return strings.Join(ids, ", ")
}

func isFinalBuildDispatchMode(dispatchMode string) bool {
	mode := strings.ToLower(strings.TrimSpace(dispatchMode))
	return mode != "" && mode != "plan-only"
}

func isSimulatedBuildDispatchMode(dispatchMode string) bool {
	mode := strings.ToLower(strings.TrimSpace(dispatchMode))
	return mode == "simulated" || mode == "synthetic"
}

func writeCodexBuildOutcomeReports(root string, phase colony.Phase, buildDirRel string, dispatches []codexBuildDispatch, recordedAt time.Time, dispatchMode string) (map[string][]string, []codexBuildDispatch, error) {
	outputsByName := make(map[string][]string, len(dispatches))
	for i := range dispatches {
		reportRel := filepath.ToSlash(filepath.Join(buildDirRel, "worker-reports", fmt.Sprintf("%s.md", dispatches[i].Name)))
		claimedOutputs := nonAssignmentBuildOutputs(dispatches[i].Outputs)
		content := renderCodexBuildWorkerOutcomeReport(root, phase, dispatches[i], claimedOutputs, recordedAt, dispatchMode)
		if err := store.AtomicWrite(reportRel, []byte(content)); err != nil {
			return nil, nil, fmt.Errorf("failed to write worker outcome report for %s: %w", dispatches[i].Name, err)
		}
		reportPath := displayDataPath(reportRel)
		outputsByName[dispatches[i].Name] = finalBuildOutputPaths(reportPath, claimedOutputs)
		dispatches[i].Outputs = outputsByName[dispatches[i].Name]
	}
	return outputsByName, dispatches, nil
}

func nonAssignmentBuildOutputs(outputs []string) []string {
	filtered := make([]string, 0, len(outputs))
	for _, output := range outputs {
		output = strings.TrimSpace(output)
		if output == "" || strings.Contains(filepath.ToSlash(output), "/worker-briefs/") {
			continue
		}
		filtered = append(filtered, output)
	}
	return uniqueSortedStrings(filtered)
}

func finalBuildOutputPaths(reportPath string, claimedOutputs []string) []string {
	rest := uniqueSortedStrings(claimedOutputs)
	outputs := []string{strings.TrimSpace(reportPath)}
	for _, output := range rest {
		if output == "" || output == reportPath {
			continue
		}
		outputs = append(outputs, output)
	}
	return outputs
}

func renderCodexBuildWorkerOutcomeReport(root string, phase colony.Phase, dispatch codexBuildDispatch, claimedOutputs []string, recordedAt time.Time, dispatchMode string) string {
	var b strings.Builder
	b.WriteString("# Worker Outcome: ")
	b.WriteString(dispatch.Name)
	b.WriteString("\n\n")
	b.WriteString("## Assignment\n")
	b.WriteString("- Phase: ")
	b.WriteString(strconv.Itoa(phase.ID))
	if phase.Name != "" {
		b.WriteString(" - ")
		b.WriteString(phase.Name)
	}
	b.WriteString("\n")
	b.WriteString("- Caste: ")
	b.WriteString(dispatch.Caste)
	b.WriteString("\n")
	if dispatch.TaskID != "" {
		b.WriteString("- Task ID: ")
		b.WriteString(dispatch.TaskID)
		b.WriteString("\n")
	}
	b.WriteString("- Task: ")
	b.WriteString(dispatch.Task)
	b.WriteString("\n")
	if root != "" {
		b.WriteString("- Root: ")
		b.WriteString(root)
		b.WriteString("\n")
	}
	b.WriteString("\n## Recorded Outcome\n")
	b.WriteString("- Status: ")
	b.WriteString(strings.TrimSpace(dispatch.Status))
	if strings.TrimSpace(dispatch.Status) == "" {
		b.WriteString("unknown")
	}
	b.WriteString("\n")
	b.WriteString("- Dispatch mode: ")
	b.WriteString(strings.TrimSpace(dispatchMode))
	b.WriteString("\n")
	b.WriteString("- Recorded at: ")
	b.WriteString(recordedAt.UTC().Format(time.RFC3339))
	b.WriteString("\n")
	if dispatch.Duration > 0 {
		b.WriteString("- Duration seconds: ")
		b.WriteString(strconv.FormatFloat(dispatch.Duration, 'f', 3, 64))
		b.WriteString("\n")
	}
	if summary := strings.TrimSpace(dispatch.Summary); summary != "" {
		b.WriteString("- Summary: ")
		b.WriteString(summary)
		b.WriteString("\n")
	}
	if len(dispatch.Blockers) > 0 {
		b.WriteString("- Blockers:\n")
		for _, blocker := range dispatch.Blockers {
			b.WriteString("  - ")
			b.WriteString(blocker)
			b.WriteString("\n")
		}
	} else {
		b.WriteString("- Blockers: none\n")
	}
	if len(claimedOutputs) > 0 {
		b.WriteString("- Claimed artifacts:\n")
		for _, output := range claimedOutputs {
			b.WriteString("  - ")
			b.WriteString(output)
			b.WriteString("\n")
		}
	} else {
		b.WriteString("- Claimed artifacts: none reported\n")
	}
	return b.String()
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

	var current colony.ColonyState
	if err := store.UpdateJSONAtomically("COLONY_STATE.json", &current, func() error {
		if err := validateRuntimeStateStillCurrent(current, phaseNum, &startedAt, colony.StateEXECUTING, colony.StateBUILT); err != nil {
			return err
		}
		current = rollback
		return nil
	}); err != nil {
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

func validateRuntimeStateStillCurrent(state colony.ColonyState, phaseNum int, expectedStartedAt *time.Time, allowedStates ...colony.State) error {
	if state.Paused {
		return runtimeStateSupersededError(phaseNum, "colony is paused")
	}
	if state.CurrentPhase != phaseNum {
		return runtimeStateSupersededError(phaseNum, fmt.Sprintf("current phase is %d", state.CurrentPhase))
	}
	if phaseNum < 1 || phaseNum > len(state.Plan.Phases) {
		return runtimeStateSupersededError(phaseNum, "phase is no longer present")
	}
	if state.Plan.Phases[phaseNum-1].Status != colony.PhaseInProgress {
		return runtimeStateSupersededError(phaseNum, fmt.Sprintf("phase status is %s", state.Plan.Phases[phaseNum-1].Status))
	}
	if !runtimeStartedAtMatches(state.BuildStartedAt, expectedStartedAt) {
		return runtimeStateSupersededError(phaseNum, "build start timestamp changed")
	}
	if len(allowedStates) == 0 {
		return nil
	}
	for _, allowed := range allowedStates {
		if state.State == allowed {
			return nil
		}
	}
	return runtimeStateSupersededError(phaseNum, fmt.Sprintf("state is %s", state.State))
}

func validateRuntimeStateMatchesExpected(state, expected colony.ColonyState) error {
	if state.Paused {
		return runtimeStateSupersededError(expected.CurrentPhase, "colony is paused")
	}
	if state.State != expected.State {
		return runtimeStateSupersededError(expected.CurrentPhase, fmt.Sprintf("state is %s", state.State))
	}
	if state.CurrentPhase != expected.CurrentPhase {
		return runtimeStateSupersededError(expected.CurrentPhase, fmt.Sprintf("current phase is %d", state.CurrentPhase))
	}
	if !runtimeStartedAtMatches(state.BuildStartedAt, expected.BuildStartedAt) {
		return runtimeStateSupersededError(expected.CurrentPhase, "build start timestamp changed")
	}
	return nil
}

func runtimeStartedAtMatches(actual, expected *time.Time) bool {
	if actual == nil || expected == nil {
		return actual == nil && expected == nil
	}
	return actual.Equal(*expected)
}

func runtimeStateSupersededError(phaseNum int, reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "state changed while runtime command was active"
	}
	return fmt.Errorf("%w for phase %d: %s", errRuntimeStateSuperseded, phaseNum, reason)
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

	b.WriteString("\n")
	b.WriteString(renderWorkerReadCacheDiscipline())
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

	heartbeatPath := filepath.ToSlash(filepath.Join(root, ".aether", "data", heartbeatFilePrefix+dispatch.Name+".json"))
	b.WriteString("\n## Heartbeat Protocol\n\n")
	b.WriteString(fmt.Sprintf("- While active, write `%s` roughly every 30 seconds.\n", heartbeatPath))
	b.WriteString(fmt.Sprintf("- Include `worker_id: %s`, `caste: %s`, `phase: %d`, and an RFC3339 `timestamp`.\n", dispatch.Name, dispatch.Caste, phase.ID))
	b.WriteString("- Remove your heartbeat file before reporting completion.\n")

	if graphContext := renderCodegraphContextForText(root, codegraphTextPartsForBuildBrief(phase, dispatch), codegraphWorkerContextBudgetChars); graphContext != "" {
		b.WriteString("\n")
		b.WriteString(graphContext)
		b.WriteString("\n")
	}

	if playbookContext := renderBuildPlaybookContext(root, dispatch, playbooks); playbookContext != "" {
		b.WriteString("\n")
		b.WriteString(playbookContext)
		b.WriteString("\n")
	}

	b.WriteString("\n## Expected Output\n\n")
	b.WriteString("- ")
	b.WriteString(expectedDispatchOutcome(dispatch))
	b.WriteString("\n")

	return b.String()
}

func cleanupStaleBuildAttemptArtifacts(phaseNum int) {
	if store == nil || phaseNum < 1 {
		return
	}
	buildDir := filepath.Join(store.BasePath(), "build", fmt.Sprintf("phase-%d", phaseNum))
	for _, name := range []string{"verification.json", "gates.json", "continue.json", "review.json"} {
		_ = os.Remove(filepath.Join(buildDir, name))
	}
	for _, name := range []string{"worker-briefs", "worker-reports"} {
		_ = os.RemoveAll(filepath.Join(buildDir, name))
	}
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
		case "oracle", "architect", "archaeologist", "ambassador", "weaver", "tracker", "keeper", "chronicler", "medic", "fixer", "porter", "sage":
			if strings.Contains(playbook, "build-prep") || strings.Contains(playbook, "build-wave") {
				filtered = append(filtered, playbook)
			}
		case "watcher", "chaos", "probe", "measurer", "gatekeeper", "auditor", "includer":
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
	case "ambassador":
		return "External integration constraints, authentication needs, and implementation sequencing."
	case "gatekeeper":
		return "Security, authentication, and secret-handling risks with concrete mitigation guidance."
	case "auditor":
		return "Quality and compliance findings with concrete follow-up risks."
	case "includer":
		return "Accessibility findings and inclusive interaction requirements."
	case "probe":
		return "Independent verification of builder claims, files, tests, and task fit."
	case "measurer":
		return "Performance, latency, and cost findings with concrete follow-up risks."
	case "chaos":
		return "Resilience findings and failure cases worth checking before advancement."
	case "archaeologist":
		return "Git history insights and risk identification from prior commits."
	case "weaver":
		return "Refactoring guidance and simplification changes that preserve behavior."
	case "tracker":
		return "Root-cause notes and regression evidence for the phase."
	case "keeper":
		return "Reusable project knowledge and patterns preserved for future workers."
	case "chronicler":
		return "Documentation updates that reflect the completed phase."
	case "medic":
		return "Health diagnosis and repair recommendations for runtime state."
	case "fixer":
		return "Focused remediation changes with verification evidence."
	case "porter":
		return "Packaging, delivery, and release-readiness notes."
	case "sage":
		return "Synthesized lessons and reusable implementation wisdom."
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
	if dispatch.ExecutionWave > 0 {
		return dispatch.ExecutionWave
	}
	if dispatch.Wave > 0 {
		return dispatch.Wave
	}
	switch dispatch.Stage {
	case "prep":
		return 1
	case "research":
		return 2
	case "design":
		return 3
	case "integration":
		return 4
	case "strategy":
		return 2
	case "probe":
		return 90
	case "verification":
		return 100
	case "measurement":
		return 101
	case "resilience":
		return 102
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
	return resolveSkillSectionResultForWorkflow("", caste, task)
}

func resolveSkillSectionResultForWorkflow(workflow, caste, task string) skillInjectResult {
	result := renderSkillInjectResult(matchSkillsForWorkflow(resolveHubPath(), workflow, caste, task))
	referenceSection := resolveReferenceSection(caste, task, "")
	result.SkillSection = appendMarkdownSections(result.SkillSection, referenceSection)
	result.Section = result.SkillSection
	return result
}

type codexWorkerSkillAssignment struct {
	Section      string
	SkillCount   int
	ColonyCount  int
	DomainCount  int
	MatchedNames []string
}

func resolveWorkerSkillAssignment(caste, task string) codexWorkerSkillAssignment {
	return resolveWorkerSkillAssignmentForWorkflow("", caste, task)
}

func resolveWorkerSkillAssignmentForWorkflow(workflow, caste, task string) codexWorkerSkillAssignment {
	result := resolveSkillSectionResultForWorkflow(workflow, caste, task)
	names := append(extractResolvedSkillNames(result.ColonySkills), extractResolvedSkillNames(result.DomainSkills)...)
	return codexWorkerSkillAssignment{
		Section:      result.SkillSection,
		SkillCount:   result.SkillCount,
		ColonyCount:  result.ColonyCount,
		DomainCount:  result.DomainCount,
		MatchedNames: uniqueSortedSkillStrings(names),
	}
}

func attachBuildDispatchContext(phaseID int, dispatches []codexBuildDispatch) {
	for i := range dispatches {
		assignment := resolveWorkerSkillAssignmentForWorkflow("build", dispatches[i].Caste, dispatches[i].Task)
		dispatches[i].SkillSection = assignment.Section
		dispatches[i].SkillCount = assignment.SkillCount
		dispatches[i].ColonySkills = assignment.ColonyCount
		dispatches[i].DomainSkills = assignment.DomainCount
		dispatches[i].MatchedSkills = append([]string{}, assignment.MatchedNames...)
		dispatches[i].HandoffSection = renderWorkerHandoffSection("build", phaseID, dispatches[i].Name)
	}
}

// resolveSkillSection matches skills for the given role and task through the
// shared runtime resolver and returns the rendered markdown section.
func resolveSkillSection(caste, task string) string {
	return resolveSkillSectionForWorkflow("", caste, task)
}

func resolveSkillSectionForWorkflow(workflow, caste, task string) string {
	result := resolveSkillSectionResultForWorkflow(workflow, caste, task)
	emitSkillActivationCeremonies(result)
	return result.SkillSection
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
