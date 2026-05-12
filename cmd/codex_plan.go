package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

type codexPlanningDispatch struct {
	Stage         string                   `json:"stage,omitempty"`
	Wave          int                      `json:"wave,omitempty"`
	Caste         string                   `json:"caste"`
	AgentName     string                   `json:"agent_name,omitempty"`
	Name          string                   `json:"name"`
	Task          string                   `json:"task"`
	TaskID        string                   `json:"task_id,omitempty"`
	Outputs       []string                 `json:"outputs"`
	Status        string                   `json:"status"`
	Summary       string                   `json:"summary,omitempty"`
	Blockers      []string                 `json:"blockers,omitempty"`
	Duration      float64                  `json:"duration,omitempty"` // Wall-clock seconds (0 = not measured)
	Brief         string                   `json:"brief,omitempty"`
	FilesCreated  []string                 `json:"files_created,omitempty"`
	FilesModified []string                 `json:"files_modified,omitempty"`
	ScoutReport   *codexScoutReport        `json:"scout_report,omitempty"`
	PhasePlan     *codexWorkerPlanArtifact `json:"phase_plan,omitempty"`
	SkillSection  string                   `json:"skill_section,omitempty"`
	SkillCount    int                      `json:"skill_count,omitempty"`
	ColonySkills  int                      `json:"colony_skill_count,omitempty"`
	DomainSkills  int                      `json:"domain_skill_count,omitempty"`
	MatchedSkills []string                 `json:"matched_skills,omitempty"`
	Claimed       []string                 `json:"-"`
}

type codexSurveyContext struct {
	SurveyDir        string
	SurveyDocs       []string
	Languages        []string
	Frameworks       []string
	Directories      []string
	EntryPoints      []string
	Dependencies     []string
	TestFiles        []string
	Issues           []string
	SecurityPatterns []string
}

type codexScoutFinding struct {
	Area      string `json:"area"`
	Discovery string `json:"discovery"`
	Source    string `json:"source"`
}

type codexScoutReport struct {
	Findings   []codexScoutFinding `json:"findings"`
	Gaps       []string            `json:"gaps"`
	Confidence int                 `json:"confidence"`
	StudyFiles []string            `json:"study_files"`
}

type codexPlanConfidence struct {
	Knowledge    int `json:"knowledge"`
	Requirements int `json:"requirements"`
	Risks        int `json:"risks"`
	Dependencies int `json:"dependencies"`
	Effort       int `json:"effort"`
	Overall      int `json:"overall"`
}

type codexWorkerPlanArtifact struct {
	Phases     []codexWorkerPlanPhase `json:"phases"`
	Confidence codexPlanConfidence    `json:"confidence"`
	Gaps       []string               `json:"gaps,omitempty"`
}

type codexWorkerPlanPhase struct {
	Name            string                `json:"name"`
	Description     string                `json:"description"`
	Tasks           []codexWorkerPlanTask `json:"tasks"`
	SuccessCriteria []string              `json:"success_criteria,omitempty"`
}

type codexWorkerPlanTask struct {
	Goal            string   `json:"goal"`
	Constraints     []string `json:"constraints,omitempty"`
	Hints           []string `json:"hints,omitempty"`
	SuccessCriteria []string `json:"success_criteria,omitempty"`
	DependsOn       []string `json:"depends_on,omitempty"`
}

type phaseTemplate struct {
	Name            string
	Description     string
	Tasks           []phaseTaskTemplate
	SuccessCriteria []string
}

type phaseTaskTemplate struct {
	Goal            string
	Constraints     []string
	Hints           []string
	SuccessCriteria []string
	DependsOn       []string
}

type codexPlanOptions struct {
	Refresh           bool
	Synthetic         bool
	PlanOnly          bool
	Depth             string
	PlanningDepth     string
	VerificationDepth string
	WorkerTimeout     time.Duration
}

type codexPlanManifest struct {
	Goal                      string                           `json:"goal"`
	Root                      string                           `json:"root"`
	GeneratedAt               string                           `json:"generated_at"`
	ColonyMode                string                           `json:"colony_mode,omitempty"`
	Refresh                   bool                             `json:"refresh"`
	ExistingPlan              bool                             `json:"existing_plan"`
	ExistingPhaseCount        int                              `json:"existing_phase_count,omitempty"`
	Depth                     string                           `json:"depth"`
	Granularity               string                           `json:"granularity"`
	GranularityMin            int                              `json:"granularity_min"`
	GranularityMax            int                              `json:"granularity_max"`
	PlanningDepth             string                           `json:"planning_depth"`
	VerificationDepth         string                           `json:"verification_depth,omitempty"`
	Survey                    codexSurveyContext               `json:"survey"`
	Dispatches                []codexPlanningDispatch          `json:"dispatches"`
	Snapshots                 map[string]codexArtifactSnapshot `json:"snapshots,omitempty"`
	DispatchMode              string                           `json:"dispatch_mode"`
	DispatchContract          map[string]interface{}           `json:"dispatch_contract"`
	FinalizeSurface           string                           `json:"finalize_surface"`
	RequiresFinalizer         bool                             `json:"requires_finalizer"`
	BoundaryQuestions         []discussQuestion                `json:"boundary_questions,omitempty"`
	BoundaryQuestionCount     int                              `json:"boundary_question_count,omitempty"`
	BoundaryQuestionsCreated  int                              `json:"boundary_questions_created,omitempty"`
	BoundaryQuestionsExisting int                              `json:"boundary_questions_existing,omitempty"`
	OrchestratorGuidance      *orchestratorBoundaryGuidance    `json:"orchestrator_boundary_guidance,omitempty"`
}

func runCodexPlan(root string, refresh bool, synthetic bool) (map[string]interface{}, error) {
	return runCodexPlanWithOptions(root, codexPlanOptions{
		Refresh:   refresh,
		Synthetic: synthetic,
	})
}

func runCodexPlanWithOptions(root string, opts codexPlanOptions) (map[string]interface{}, error) {
	if store == nil {
		return nil, fmt.Errorf("no store initialized")
	}

	state, err := loadActiveColonyState()
	if err != nil {
		return nil, fmt.Errorf("%s", colonyStateLoadMessage(err))
	}

	granularity, planDepth, err := resolvePlanGranularityDepth(state.PlanGranularity, opts.Depth)
	if err != nil {
		return nil, err
	}
	currentPhase := firstBuildablePhase(state.Plan.Phases)
	var planningPhase colony.Phase
	if currentPhase > 0 && currentPhase <= len(state.Plan.Phases) {
		planningPhase = state.Plan.Phases[currentPhase-1]
	} else if len(state.Plan.Phases) > 0 {
		planningPhase = state.Plan.Phases[0]
	}
	planningDepth, err := resolvePlanningDepthSmart(opts.PlanningDepth, planningPhase, len(state.Plan.Phases))
	if err != nil {
		return nil, err
	}
	verificationDepth, err := resolveVerificationDepthSmart(opts.VerificationDepth, planningPhase, len(state.Plan.Phases))
	if err != nil {
		return nil, err
	}
	verificationSmartDefault := opts.VerificationDepth == ""
	planningSmartDefault := opts.PlanningDepth == ""
	pending := loadPendingDecisionFile()
	unresolvedClarifications := countPendingClarifications(pending)
	clarificationWarning := ""
	if unresolvedClarifications > 0 {
		clarificationWarning = "Unresolved clarifications exist. Run `aether discuss` to resolve them before planning, or proceed with implicit assumptions."
	}

	if opts.PlanOnly {
		return runCodexPlanPlanOnly(root, state, granularity, planDepth, unresolvedClarifications, clarificationWarning, opts)
	}

	if len(state.Plan.Phases) > 0 && !opts.Refresh {
		// Persist resolved verification depth only for non-plan-only paths.
		state.VerificationDepth = verificationDepth
		if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
			return nil, fmt.Errorf("failed to persist verification depth: %w", err)
		}
		nextPhase := firstBuildablePhase(state.Plan.Phases)
		nextCommand := "aether build 1"
		if nextPhase > 0 {
			nextCommand = fmt.Sprintf("aether build %d", nextPhase)
		}
		updateSessionSummary("plan", nextCommand, fmt.Sprintf("Loaded existing plan (%d phases)", len(state.Plan.Phases)))
		return map[string]interface{}{
			"planned":                    true,
			"existing_plan":              true,
			"colony_mode":                string(state.EffectiveColonyMode()),
			"goal":                       *state.Goal,
			"phases":                     state.Plan.Phases,
			"count":                      len(state.Plan.Phases),
			"depth":                      planDepth,
			"planning_depth":             planningDepth,
			"verification_depth":         verificationDepth,
			"verification_smart_default": verificationSmartDefault,
			"planning_smart_default":     planningSmartDefault,
			"planning_phase":             planningPhase,
			"granularity":                string(granularity),
			"dispatch_contract":          planningDispatchContractWithTimeout(opts.WorkerTimeout),
			"unresolved_clarifications":  unresolvedClarifications,
			"clarification_warning":      clarificationWarning,
			"next":                       nextCommand,
		}, nil
	}

	if codex.ShouldUseAgentDelegatePath() {
		return runCodexPlanAgentDelegate(root, state, granularity, planDepth, unresolvedClarifications, clarificationWarning, opts)
	}

	// Persist resolved verification depth only once planning will finalize in this process.
	state.VerificationDepth = verificationDepth
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		return nil, fmt.Errorf("failed to persist verification depth: %w", err)
	}

	if opts.Refresh {
		if state.CurrentPhase > 0 {
			hasCompletedPhase := false
			for _, phase := range state.Plan.Phases {
				if phase.Status == colony.PhaseCompleted {
					hasCompletedPhase = true
					break
				}
			}
			if hasCompletedPhase {
				return nil, fmt.Errorf("cannot force-replan after completed phases; archive this colony and start a new one")
			}
			// In-progress phase with no completed work — stale state. Reset for fresh plan.
			for i := range state.Plan.Phases {
				state.Plan.Phases[i].Status = colony.PhaseReady
				for j := range state.Plan.Phases[i].Tasks {
					state.Plan.Phases[i].Tasks[j].Status = colony.TaskPending
				}
			}
			state.CurrentPhase = 0
			state.State = colony.StateREADY
			if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
				return nil, fmt.Errorf("failed to reset stale phase state for force-replan: %w", err)
			}
		}
		clearFallbackPlanningArtifacts(root)
	}

	runHandle, err := beginRuntimeSpawnRun("plan", time.Now().UTC())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize planning run: %w", err)
	}
	runStatus := "failed"
	defer func() {
		finishRuntimeSpawnRun(runHandle, runStatus, time.Now().UTC())
	}()

	survey, err := loadCodexSurveyContext(root)
	if err != nil {
		return nil, err
	}

	planningDir := filepath.Join(store.BasePath(), "planning")
	phaseResearchDir := filepath.Join(store.BasePath(), "phase-research")
	if err := os.MkdirAll(planningDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create planning directory: %w", err)
	}
	if err := os.MkdirAll(phaseResearchDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create phase research directory: %w", err)
	}
	artifactSnapshots := snapshotRelativeFiles(root,
		filepath.ToSlash(filepath.Join(".aether", "data", "planning")),
		filepath.ToSlash(filepath.Join(".aether", "data", "phase-research")),
	)

	dispatches := plannedPlanningWorkersForGoal(root, *state.Goal)
	dispatchMode := "synthetic"
	artifactSource := "local-synthesis"
	planSource := "local-synthesis"
	planningWarning := ""
	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	for _, dispatch := range dispatches {
		if err := spawnTree.RecordSpawn("Queen", dispatch.Caste, dispatch.Name, dispatch.Task, 1); err != nil {
			return nil, fmt.Errorf("failed to record planning spawn: %w", err)
		}
	}

	emitVisualProgress(renderPlanDispatchPreview(*state.Goal, dispatches))

	if !opts.Synthetic {
		invoker := newCodexWorkerInvoker()
		if _, ok := invoker.(*codex.FakeInvoker); !ok && !invoker.IsAvailable(context.Background()) {
			dispatchMode = "fallback"
			planningWarning = fmt.Sprintf("Real planning workers were unavailable, so Aether fell back to local synthesis. Cause: %s", dispatchAvailabilityMessage(invoker))
		} else {
			realDispatches, dispatchErr := dispatchRealPlanningWorkersWithTimeout(context.Background(), root, survey, invoker, opts.WorkerTimeout, *state.Goal)
			if realDispatches != nil {
				dispatches = realDispatches
			}
			if dispatchErr != nil {
				if _, ok := invoker.(*codex.FakeInvoker); ok {
					dispatchMode = "simulated"
				} else {
					dispatchMode = "fallback"
					planningWarning = fmt.Sprintf("Real planning workers did not finish cleanly, so Aether fell back to local synthesis. Cause: %s", dispatchErr.Error())
				}
			} else if realDispatches != nil {
				if _, ok := invoker.(*codex.FakeInvoker); ok {
					dispatchMode = "simulated"
				} else {
					dispatchMode = "real"
				}
			}
		}
	} else {
		dispatchMode = "synthetic"
	}

	scoutReport := scoutReportForPlanningDispatches(*state.Goal, survey, dispatches)
	if scoutIndex := planningDispatchIndexByCaste(dispatches, "scout"); scoutIndex >= 0 && dispatches[scoutIndex].ScoutReport == nil {
		dispatches[scoutIndex].ScoutReport = &scoutReport
	}
	scoutDispatch, ok := planningDispatchByCaste(dispatches, "scout")
	if !ok {
		return nil, fmt.Errorf("planning dispatches missing scout worker")
	}
	routeSetterDispatch, ok := planningDispatchByCaste(dispatches, "route_setter")
	if !ok {
		return nil, fmt.Errorf("planning dispatches missing route-setter worker")
	}
	scoutFile, preservedScoutArtifact, err := writePlanningScoutArtifact(root, planningDir, *state.Goal, granularity, survey, scoutDispatch, scoutReport, artifactSnapshots)
	if err != nil {
		return nil, err
	}

	phases, confidence, unresolvedGaps := synthesizeRouteSetterPlan(*state.Goal, granularity, survey, scoutReport)
	if workerPlan, ok, note := loadWorkerPlanArtifact(root, artifactSnapshots, dispatches); ok {
		phases = buildWorkerPlanPhases(workerPlan)
		confidence = mergePlanConfidence(confidence, workerPlan.Confidence)
		unresolvedGaps = limitStrings(uniqueSortedStrings(append(unresolvedGaps, workerPlan.Gaps...)), 4)
		planSource = "worker-artifact"
	} else if note != "" {
		unresolvedGaps = limitStrings(uniqueSortedStrings(append(unresolvedGaps, note)), 4)
	}
	routeSetterFile, preservedRouteArtifact, err := writeRouteSetterArtifact(root, planningDir, *state.Goal, granularity, survey, routeSetterDispatch, confidence, unresolvedGaps, phases, artifactSnapshots)
	if err != nil {
		return nil, err
	}
	planArtifactFile, preservedPlanArtifact, err := writeWorkerPlanArtifact(root, planningDir, confidence, unresolvedGaps, phases, artifactSnapshots, dispatches)
	if err != nil {
		return nil, err
	}
	phaseResearchFiles, preservedResearchArtifacts, err := writePhaseResearchArtifacts(root, phaseResearchDir, survey, scoutReport, phases, artifactSnapshots, dispatches)
	if err != nil {
		return nil, err
	}
	if preservedScoutArtifact || preservedRouteArtifact || preservedPlanArtifact || preservedResearchArtifacts > 0 {
		artifactSource = "worker-written"
	}

	// Mark fallback artifacts so a subsequent refresh can overwrite them.
	if dispatchMode == "fallback" {
		markerPath := filepath.Join(planningDir, ".fallback-marker")
		os.WriteFile(markerPath, []byte(time.Now().UTC().Format(time.RFC3339)), 0644)
	} else {
		// Real or simulated dispatch succeeded — remove any stale marker.
		os.Remove(filepath.Join(planningDir, ".fallback-marker"))
	}
	clearPlanningBackupArtifacts(planningDir)

	for i := range dispatches {
		status := dispatches[i].Status
		if strings.TrimSpace(status) == "" || status == "spawned" {
			status = "completed"
		}
		summary := strings.TrimSpace(dispatches[i].Summary)
		if summary == "" {
			summary = strings.Join(dispatches[i].Outputs, ", ")
		}
		if summary == "" && dispatchMode != "real" {
			summary = "Local planning synthesis fallback"
		}
		if err := spawnTree.UpdateStatus(dispatches[i].Name, status, summary); err != nil {
			return nil, fmt.Errorf("failed to update planning completion: %w", err)
		}
	}
	emitPlanCeremonyDispatchSequence("aether-plan", dispatches)

	// Validate task dependency graph for cycles (LOOP-04)
	if err := colony.DetectCycles(phases); err != nil {
		var cycleErr *colony.CycleError
		if errors.As(err, &cycleErr) {
			emitLoopBreakEvent("cycle_detected",
				fmt.Sprintf("circular dependency detected: %s", cycleErr.Error()),
				"plan rejected, cycle must be removed before regeneration",
				"aether-plan")
			return nil, fmt.Errorf("plan contains circular dependency: %s. Remove the cycle and regenerate the plan", cycleErr)
		}
		return nil, fmt.Errorf("plan dependency validation failed: %w", err)
	}

	now := time.Now().UTC()
	state.State = colony.StateREADY
	state.CurrentPhase = firstBuildablePhase(phases)
	state.BuildStartedAt = nil
	state.PlanGranularity = granularity
	planConfidence := float64(confidence.Overall) / 100.0
	state.Plan = colony.Plan{
		GeneratedAt: &now,
		Confidence:  &planConfidence,
		Phases:      phases,
	}
	state.Events = append(trimmedEvents(state.Events),
		fmt.Sprintf("%s|planning_scout|plan|Scout summarized surveyed repo context", now.Format(time.RFC3339)),
		fmt.Sprintf("%s|plan_generated|plan|Generated %d phases with %d%% confidence", now.Format(time.RFC3339), len(phases), confidence.Overall),
	)
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		return nil, fmt.Errorf("failed to save colony state: %w", err)
	}

	nextPhase := firstBuildablePhase(phases)
	nextCommand := "aether build 1"
	if nextPhase > 0 {
		nextCommand = fmt.Sprintf("aether build %d", nextPhase)
	}
	updateSessionSummary("plan", nextCommand, fmt.Sprintf("Generated %d plan phases with %d%% confidence", len(phases), confidence.Overall))

	dispatchMaps := make([]map[string]interface{}, 0, len(dispatches))
	for _, dispatch := range dispatches {
		entry := map[string]interface{}{
			"caste":   dispatch.Caste,
			"name":    dispatch.Name,
			"task":    dispatch.Task,
			"outputs": dispatch.Outputs,
			"status":  dispatch.Status,
		}
		if summary := strings.TrimSpace(dispatch.Summary); summary != "" {
			entry["summary"] = summary
		}
		if dispatch.Duration > 0 {
			entry["duration"] = dispatch.Duration
		}
		if dispatch.ScoutReport != nil {
			entry["scout_report"] = dispatch.ScoutReport
		}
		dispatchMaps = append(dispatchMaps, entry)
	}

	result := map[string]interface{}{
		"planned":                    true,
		"existing_plan":              false,
		"refreshed":                  opts.Refresh,
		"colony_mode":                string(state.EffectiveColonyMode()),
		"goal":                       *state.Goal,
		"phases":                     phases,
		"count":                      len(phases),
		"depth":                      planDepth,
		"planning_depth":             planningDepth,
		"verification_depth":         verificationDepth,
		"verification_smart_default": verificationSmartDefault,
		"planning_smart_default":     planningSmartDefault,
		"planning_phase":             planningPhase,
		"granularity":                string(granularity),
		"granularity_min":            granularityMin(granularity),
		"granularity_max":            granularityMax(granularity),
		"confidence":                 confidence,
		"planning_dir":               planningDir,
		"planning_files":             []string{filepath.Base(scoutFile), filepath.Base(routeSetterFile)},
		"plan_artifact":              filepath.Base(planArtifactFile),
		"phase_research_dir":         phaseResearchDir,
		"phase_research_files":       phaseResearchFiles,
		"dispatches":                 dispatchMaps,
		"dispatch_mode":              dispatchMode,
		"dispatch_contract":          planningDispatchContractForDispatches(dispatches, opts.WorkerTimeout),
		"artifact_source":            artifactSource,
		"plan_source":                planSource,
		"gaps":                       unresolvedGaps,
		"survey_docs":                survey.SurveyDocs,
		"unresolved_clarifications":  unresolvedClarifications,
		"clarification_warning":      clarificationWarning,
		"planning_warning":           planningWarning,
		"next":                       nextCommand,
	}
	statuses := make([]string, 0, len(dispatches))
	for _, dispatch := range dispatches {
		statuses = append(statuses, dispatch.Status)
	}
	runStatus = summarizeRunStatus(statuses...)
	return result, nil
}

func runCodexPlanAgentDelegate(root string, state colony.ColonyState, granularity colony.PlanGranularity, planDepth string, unresolvedClarifications int, clarificationWarning string, opts codexPlanOptions) (map[string]interface{}, error) {
	result, err := runCodexPlanPlanOnly(root, state, granularity, planDepth, unresolvedClarifications, clarificationWarning, opts)
	if err != nil {
		return nil, err
	}
	manifest, ok := result["plan_manifest"].(codexPlanManifest)
	if !ok {
		return result, nil
	}
	manifest.DispatchMode = "agent-delegate"
	result["plan_manifest"] = manifest
	result["planning_manifest"] = manifest
	result["agent_delegate"] = true
	result["status"] = "agent-delegate"
	result["dispatch_mode"] = "agent-delegate"
	result["requires_finalizer"] = true
	result["agent_delegate_reason"] = codex.AgentDelegateFallbackReason()
	result["next"] = "dispatch host planning agents, then run `aether plan-finalize --completion-file <file>`"
	if contract, ok := result["wrapper_contract"].(map[string]interface{}); ok {
		contract["source_command"] = "AETHER_OUTPUT_MODE=json aether plan --refresh"
		contract["dispatch_mode"] = "agent-delegate"
		contract["finalize_command"] = "AETHER_OUTPUT_MODE=json aether plan-finalize --completion-file <file>"
		result["wrapper_contract"] = contract
	}
	return result, nil
}

func runCodexPlanPlanOnly(root string, state colony.ColonyState, granularity colony.PlanGranularity, planDepth string, unresolvedClarifications int, clarificationWarning string, opts codexPlanOptions) (map[string]interface{}, error) {
	if state.Goal == nil || strings.TrimSpace(*state.Goal) == "" {
		return nil, fmt.Errorf("No active colony goal. Run `aether init \"goal\"` first.")
	}
	planningDepth, err := resolvePlanningDepthSmart(opts.PlanningDepth, colony.Phase{ID: 1}, len(state.Plan.Phases))
	if err != nil {
		return nil, err
	}
	verificationDepth, err := resolveVerificationDepthSmart(opts.VerificationDepth, colony.Phase{ID: 1}, len(state.Plan.Phases))
	if err != nil {
		return nil, err
	}
	verificationSmartDefault := opts.VerificationDepth == ""
	planningSmartDefault := opts.PlanningDepth == ""
	planningPhase := colony.Phase{ID: 1}
	if len(state.Plan.Phases) > 0 && !opts.Refresh {
		nextPhase := firstBuildablePhase(state.Plan.Phases)
		nextCommand := "aether build 1"
		if nextPhase > 0 {
			nextCommand = fmt.Sprintf("aether build %d", nextPhase)
		}
		boundary, err := materializeOrchestratorBoundaryQuestions("plan", state, planningPhase, planBoundaryQuestionCandidates(state, granularity, planDepth, planningDepth, verificationDepth))
		if err != nil {
			return nil, err
		}
		result := map[string]interface{}{
			"plan_only":                  true,
			"planned":                    true,
			"existing_plan":              true,
			"colony_mode":                string(state.EffectiveColonyMode()),
			"goal":                       *state.Goal,
			"phases":                     state.Plan.Phases,
			"count":                      len(state.Plan.Phases),
			"depth":                      planDepth,
			"planning_depth":             planningDepth,
			"verification_depth":         verificationDepth,
			"verification_smart_default": verificationSmartDefault,
			"planning_smart_default":     planningSmartDefault,
			"planning_phase":             planningPhase,
			"granularity":                string(granularity),
			"dispatch_contract":          planningDispatchContractWithTimeout(opts.WorkerTimeout),
			"dispatch_mode":              "plan-only",
			"requires_finalizer":         false,
			"unresolved_clarifications":  unresolvedClarifications,
			"clarification_warning":      clarificationWarning,
			"next":                       nextCommand,
		}
		addBoundaryQuestionResultFields(result, boundary)
		addOrchestratorBoundaryGuidance(result, "plan", state, nextCommand, boundary.Questions)
		return result, nil
	}
	if opts.Refresh && state.CurrentPhase > 0 {
		for _, phase := range state.Plan.Phases {
			if phase.Status == colony.PhaseCompleted {
				return nil, fmt.Errorf("cannot force-replan after completed phases; archive this colony and start a new one")
			}
		}
	}

	survey, err := loadCodexSurveyContext(root)
	if err != nil {
		return nil, err
	}
	dispatches := plannedPlanningWorkersForGoal(root, *state.Goal)
	specs := planningWorkerSpecsForGoal(*state.Goal)
	for i := range dispatches {
		dispatches[i].Status = "planned"
		dispatches[i].Brief = renderPlanningWorkerBrief(root, survey, specs[i])
	}
	artifactSnapshots := snapshotRelativeFiles(root,
		filepath.ToSlash(filepath.Join(".aether", "data", "planning")),
		filepath.ToSlash(filepath.Join(".aether", "data", "phase-research")),
	)
	dispatchContract := planningDispatchContractForDispatches(dispatches, opts.WorkerTimeout)
	generatedAt := time.Now().UTC()
	manifest := codexPlanManifest{
		Goal:               *state.Goal,
		Root:               root,
		GeneratedAt:        generatedAt.Format(time.RFC3339),
		ColonyMode:         string(state.EffectiveColonyMode()),
		Refresh:            opts.Refresh,
		ExistingPlan:       len(state.Plan.Phases) > 0,
		ExistingPhaseCount: len(state.Plan.Phases),
		Depth:              planDepth,
		Granularity:        string(granularity),
		GranularityMin:     granularityMin(granularity),
		GranularityMax:     granularityMax(granularity),
		PlanningDepth:      planningDepth,
		VerificationDepth:  verificationDepth,
		Survey:             survey,
		Dispatches:         dispatches,
		Snapshots:          artifactSnapshots,
		DispatchMode:       "plan-only",
		DispatchContract:   dispatchContract,
		FinalizeSurface:    "pending",
		RequiresFinalizer:  true,
	}

	boundary, err := materializeOrchestratorBoundaryQuestions("plan", state, planningPhase, planBoundaryQuestionCandidates(state, granularity, planDepth, planningDepth, verificationDepth))
	if err != nil {
		return nil, err
	}
	manifest.BoundaryQuestions = boundary.Questions
	manifest.BoundaryQuestionCount = len(boundary.Questions)
	manifest.BoundaryQuestionsCreated = boundary.Created
	manifest.BoundaryQuestionsExisting = boundary.Existing

	result := map[string]interface{}{
		"plan_only":                  true,
		"planned":                    true,
		"existing_plan":              false,
		"refreshed":                  opts.Refresh,
		"colony_mode":                string(state.EffectiveColonyMode()),
		"goal":                       *state.Goal,
		"depth":                      planDepth,
		"planning_depth":             planningDepth,
		"verification_depth":         verificationDepth,
		"verification_smart_default": verificationSmartDefault,
		"planning_smart_default":     planningSmartDefault,
		"planning_phase":             planningPhase,
		"granularity":                string(granularity),
		"granularity_min":            granularityMin(granularity),
		"granularity_max":            granularityMax(granularity),
		"plan_manifest":              manifest,
		"planning_manifest":          manifest,
		"dispatches":                 dispatches,
		"dispatch_count":             len(dispatches),
		"dispatch_mode":              "plan-only",
		"dispatch_contract":          dispatchContract,
		"unresolved_clarifications":  unresolvedClarifications,
		"clarification_warning":      clarificationWarning,
		"next":                       "spawn wrapper planning agents, then record completion",
		"wrapper_contract": map[string]interface{}{
			"source_command":          "AETHER_OUTPUT_MODE=json aether plan --plan-only --depth <fast|balanced|deep|exhaustive> --planning-depth <light|standard|deep>",
			"spawn_log_required":      true,
			"spawn_complete_required": true,
			"finalize_surface":        "pending",
			"runtime_state_only":      true,
			"planning_depth":          planningDepth,
		},
	}
	addBoundaryQuestionResultFields(result, boundary)
	if guidance, ok := addOrchestratorBoundaryGuidance(result, "plan", state, planAfterDiscussNext(opts), boundary.Questions); ok {
		manifest.OrchestratorGuidance = &guidance
		result["plan_manifest"] = manifest
		result["planning_manifest"] = manifest
	}
	return result, nil
}

func planAfterDiscussNext(opts codexPlanOptions) string {
	if opts.Refresh {
		return "aether plan --refresh"
	}
	return "aether plan"
}

func normalizedGranularity(value colony.PlanGranularity) colony.PlanGranularity {
	if value.Valid() {
		return value
	}
	return colony.GranularityMilestone
}

func resolvePlanGranularityDepth(current colony.PlanGranularity, depth string) (colony.PlanGranularity, string, error) {
	depth = strings.ToLower(strings.TrimSpace(depth))
	if depth == "" {
		granularity := normalizedGranularity(current)
		return granularity, planningDepthForGranularity(granularity), nil
	}
	switch depth {
	case "fast", "quick", "light":
		return colony.GranularitySprint, "fast", nil
	case "balanced", "standard", "default":
		return colony.GranularityMilestone, "balanced", nil
	case "deep":
		return colony.GranularityQuarter, "deep", nil
	case "exhaustive", "full":
		return colony.GranularityMajor, "exhaustive", nil
	case string(colony.GranularitySprint):
		return colony.GranularitySprint, "fast", nil
	case string(colony.GranularityMilestone):
		return colony.GranularityMilestone, "balanced", nil
	case string(colony.GranularityQuarter):
		return colony.GranularityQuarter, "deep", nil
	case string(colony.GranularityMajor):
		return colony.GranularityMajor, "exhaustive", nil
	default:
		return "", "", fmt.Errorf("invalid planning depth %q: must be fast, balanced, deep, or exhaustive", depth)
	}
}

func resolvePlanningDepth(depth string) (string, error) {
	normalized := colony.NormalizePlanningDepth(depth)
	if depth != "" {
		// User explicitly provided a value; validate it maps to a recognized constant.
		lower := strings.ToLower(strings.TrimSpace(depth))
		switch lower {
		case "light", "minimal", "coarse", "deep", "granular", "thorough", "standard", "default":
			// known alias or canonical value
		default:
			return "", fmt.Errorf("invalid planning depth %q: must be light, standard, or deep", depth)
		}
	}
	return string(normalized), nil
}

// resolvePlanningDepthSmart wraps resolvePlanningDepth with smart defaults.
// When depth is empty (no explicit user flag), it uses resolveSmartPlanningDepth
// to auto-select based on phase position and risk signals.
func resolvePlanningDepthSmart(depth string, phase colony.Phase, totalPhases int) (string, error) {
	normalized, err := resolvePlanningDepth(depth)
	if err != nil {
		return "", err
	}
	// If user explicitly provided a depth, use it (normalized is non-default)
	if depth != "" {
		return normalized, nil
	}
	// No explicit depth -- use smart default
	return string(resolveSmartPlanningDepth(phase, totalPhases)), nil
}

func planningDepthForGranularity(granularity colony.PlanGranularity) string {
	switch granularity {
	case colony.GranularitySprint:
		return "fast"
	case colony.GranularityQuarter:
		return "deep"
	case colony.GranularityMajor:
		return "exhaustive"
	default:
		return "balanced"
	}
}

func granularityMin(value colony.PlanGranularity) int {
	min, _ := colony.GranularityRange(value)
	return min
}

func granularityMax(value colony.PlanGranularity) int {
	_, max := colony.GranularityRange(value)
	return max
}

func firstBuildablePhase(phases []colony.Phase) int {
	for _, phase := range phases {
		if phase.Status != colony.PhaseCompleted {
			if phase.ID > 0 {
				return phase.ID
			}
		}
	}
	if len(phases) > 0 {
		return phases[0].ID
	}
	return 0
}

func plannedPlanningWorkers(root string) []codexPlanningDispatch {
	return plannedPlanningWorkersForGoal(root, "")
}

func plannedPlanningWorkersForGoal(root, goal string) []codexPlanningDispatch {
	specs := planningWorkerSpecsForGoal(goal)
	dispatches := make([]codexPlanningDispatch, 0, len(specs))
	for i, spec := range specs {
		dispatches = append(dispatches, codexPlanningDispatch{
			Stage:     planningStageForCaste(spec.Caste),
			Wave:      i + 1,
			Caste:     spec.Caste,
			AgentName: strings.TrimSuffix(spec.AgentFile, ".toml"),
			Name:      deterministicAntName(spec.Caste, root+"|plan|"+spec.Caste),
			Task:      spec.Task,
			TaskID:    "plan-" + strings.ReplaceAll(spec.Caste, "_", "-"),
			Outputs:   append([]string{}, spec.Outputs...),
			Status:    "spawned",
		})
	}
	attachPlanningDispatchSkillAssignments(dispatches)
	return dispatches
}

func planningDispatchByCaste(dispatches []codexPlanningDispatch, caste string) (codexPlanningDispatch, bool) {
	index := planningDispatchIndexByCaste(dispatches, caste)
	if index < 0 {
		return codexPlanningDispatch{}, false
	}
	return dispatches[index], true
}

func planningDispatchIndexByCaste(dispatches []codexPlanningDispatch, caste string) int {
	for i, dispatch := range dispatches {
		if strings.EqualFold(dispatch.Caste, caste) {
			return i
		}
	}
	return -1
}

func planningStageForCaste(caste string) string {
	switch caste {
	case "scout":
		return "scouting"
	case "route_setter":
		return "routing"
	case "architect":
		return "architecture"
	case "oracle":
		return "research"
	case "gatekeeper":
		return "security"
	case "includer":
		return "accessibility"
	case "chronicler":
		return "documentation"
	case "keeper":
		return "knowledge"
	default:
		return "planning"
	}
}

func attachPlanningDispatchSkillAssignments(dispatches []codexPlanningDispatch) {
	for i := range dispatches {
		assignment := resolveWorkerSkillAssignmentForWorkflow("plan", dispatches[i].Caste, dispatches[i].Task)
		dispatches[i].SkillSection = assignment.Section
		dispatches[i].SkillCount = assignment.SkillCount
		dispatches[i].ColonySkills = assignment.ColonyCount
		dispatches[i].DomainSkills = assignment.DomainCount
		dispatches[i].MatchedSkills = append([]string{}, assignment.MatchedNames...)
	}
}

// planningWorkerSpec defines a single planning worker for real dispatch.
type planningWorkerSpec struct {
	Caste     string // Worker caste (scout, route_setter)
	AgentFile string // TOML filename (e.g., "aether-scout.toml")
	Task      string // Task brief
	Outputs   []string
}

// planningWorkerSpecs is the canonical list of planning workers, matching plannedPlanningWorkers order.
var planningWorkerSpecs = []planningWorkerSpec{
	{
		Caste:     "scout",
		AgentFile: "aether-scout.toml",
		Task:      "Survey the repo and distill planning findings from available territory reports",
		Outputs:   []string{"SCOUT.md"},
	},
	{
		Caste:     "route_setter",
		AgentFile: "aether-route-setter.toml",
		Task:      "Convert surveyed findings into an executable multi-phase colony plan",
		Outputs:   []string{"ROUTE-SETTER.md"},
	},
}

func planningWorkerSpecsForGoal(goal string) []planningWorkerSpec {
	phase := colony.Phase{
		Name:        strings.TrimSpace(goal),
		Description: strings.TrimSpace(goal),
		Mode:        colony.InferPhaseMode(goal, goal),
	}
	selected := queenBuildCasteSet(queenOrchestrate(phase, "plan", colony.ColonyState{}))
	specs := make([]planningWorkerSpec, 0, len(planningWorkerSpecs))
	for _, spec := range planningWorkerSpecs {
		if selected[spec.Caste] {
			specs = append(specs, spec)
		}
	}
	for _, caste := range []string{"architect", "oracle", "gatekeeper", "includer", "keeper", "chronicler"} {
		if !selected[caste] {
			continue
		}
		if spec, ok := planningWorkerSpecForCaste(caste); ok {
			specs = append(specs, spec)
		}
	}
	return specs
}

func planningWorkerSpecForCaste(caste string) (planningWorkerSpec, bool) {
	switch caste {
	case "architect":
		return planningWorkerSpec{Caste: "architect", AgentFile: "aether-architect.toml", Task: "Identify architecture boundaries, interfaces, and structural risks before route-setting", Outputs: []string{"ARCHITECT.md"}}, true
	case "oracle":
		return planningWorkerSpec{Caste: "oracle", AgentFile: "aether-oracle.toml", Task: "Research unknowns and evaluate options that could change the route", Outputs: []string{"ORACLE.md"}}, true
	case "gatekeeper":
		return planningWorkerSpec{Caste: "gatekeeper", AgentFile: "aether-gatekeeper.toml", Task: "Review security, permission, dependency, and release-integrity constraints before planning", Outputs: []string{"GATEKEEPER.md"}}, true
	case "includer":
		return planningWorkerSpec{Caste: "includer", AgentFile: "aether-includer.toml", Task: "Review accessibility and inclusive-use requirements before planning", Outputs: []string{"INCLUDER.md"}}, true
	case "keeper":
		return planningWorkerSpec{Caste: "keeper", AgentFile: "aether-keeper.toml", Task: "Preserve relevant conventions, prior decisions, and reusable knowledge for the plan", Outputs: []string{"KEEPER.md"}}, true
	case "chronicler":
		return planningWorkerSpec{Caste: "chronicler", AgentFile: "aether-chronicler.toml", Task: "Map documentation surfaces and changelog obligations for the plan", Outputs: []string{"CHRONICLER.md"}}, true
	}
	return planningWorkerSpec{}, false
}

// dispatchRealPlanningWorkers attempts real worker invocation for planning.
// If the invoker is not available, it returns nil, nil (caller falls back to plannedPlanningWorkers).
func dispatchRealPlanningWorkers(ctx context.Context, root string, invoker codex.WorkerInvoker) ([]codexPlanningDispatch, error) {
	return dispatchRealPlanningWorkersWithTimeout(ctx, root, codexSurveyContext{}, invoker, 0)
}

func dispatchRealPlanningWorkersWithTimeout(ctx context.Context, root string, survey codexSurveyContext, invoker codex.WorkerInvoker, timeoutOverride time.Duration, goalOpt ...string) ([]codexPlanningDispatch, error) {
	if invoker == nil || !invoker.IsAvailable(ctx) {
		return nil, nil
	}
	goal := ""
	if len(goalOpt) > 0 {
		goal = goalOpt[0]
	}
	planned := plannedPlanningWorkersForGoal(root, goal)
	specs := planningWorkerSpecsForGoal(goal)
	capsule := resolveCodexWorkerContext()
	pheromoneSection := resolvePheromoneSection()
	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	results := make([]codex.DispatchResult, 0, len(specs))
	workerTimeout := effectivePlanningDispatchTimeout(timeoutOverride)
	scoutGuidance := ""
	for i, spec := range specs {
		agentName := strings.TrimSuffix(spec.AgentFile, ".toml")
		dispatch := codex.WorkerDispatch{
			ID:               fmt.Sprintf("planning-%d", i),
			WorkerName:       planned[i].Name,
			AgentName:        agentName,
			AgentTOMLPath:    dispatchAgentPath(root, invoker, agentName),
			Caste:            spec.Caste,
			TaskID:           fmt.Sprintf("plan-%d", i),
			TaskBrief:        renderPlanningWorkerBrief(root, survey, spec, scoutGuidance),
			ContextCapsule:   capsule,
			HandoffSection:   renderWorkerHandoffSection("plan", 0, planned[i].Name),
			Workflow:         "plan",
			SkillSection:     resolveSkillSectionForWorkflow("plan", spec.Caste, spec.Task),
			PheromoneSection: pheromoneSection,
			Root:             root,
			Wave:             i + 1,
			Timeout:          workerTimeout,
		}

		stageResults, err := dispatchBatchByWaveWithVisuals(
			ctx,
			invoker,
			[]codex.WorkerDispatch{dispatch},
			colony.ModeInRepo,
			"Planning Wave",
			false,
			func(wave int) codex.DispatchObserver {
				return runtimeVisualDispatchObserver(spawnTree, "Planning worker active", wave)
			},
		)
		if err != nil {
			return nil, err
		}
		results = append(results, stageResults...)
		if spec.Caste == "scout" && len(stageResults) > 0 {
			if report, ok := scoutReportFromWorkerResult(stageResults[0].WorkerResult); ok {
				scoutGuidance = renderScoutPlanningGuidance(report)
			}
		}
		if stageResults[0].Status != "completed" {
			dispatches := convertPlanningDispatchResults(results, root, goal)
			if i+1 < len(specs) {
				dispatches[i+1].Status = "dependency_blocked"
				dispatches[i+1].Summary = fmt.Sprintf("%s did not complete, so downstream planning stayed blocked.", dispatch.WorkerName)
			}
			return dispatches, fmt.Errorf("planning worker %s did not complete: %s", dispatch.WorkerName, stageResults[0].Status)
		}
	}

	return convertPlanningDispatchResults(results, root, goal), nil
}

// convertPlanningDispatchResults maps a slice of DispatchResult to codexPlanningDispatch.
// If results don't cover all specs, remaining specs get the planned defaults.
func convertPlanningDispatchResults(results []codex.DispatchResult, root string, goalOpt ...string) []codexPlanningDispatch {
	goal := ""
	if len(goalOpt) > 0 {
		goal = goalOpt[0]
	}
	planned := plannedPlanningWorkersForGoal(root, goal)
	dispatches := make([]codexPlanningDispatch, 0, len(planned))

	for i, planned := range planned {
		d := codexPlanningDispatch{
			Caste:   planned.Caste,
			Name:    planned.Name,
			Task:    planned.Task,
			Outputs: planned.Outputs,
			Status:  "spawned",
		}

		if i < len(results) {
			r := results[i]
			if r.WorkerName != "" {
				d.Name = r.WorkerName
			}
			d.Status = normalizeRuntimeDispatchStatus(r.Status)
			if r.WorkerResult != nil {
				d.Duration = r.WorkerResult.Duration.Seconds()
				d.Claimed = append(d.Claimed, r.WorkerResult.FilesCreated...)
				d.Claimed = append(d.Claimed, r.WorkerResult.FilesModified...)
				d.Claimed = uniqueSortedStrings(d.Claimed)
				d.Summary = strings.TrimSpace(r.WorkerResult.Summary)
				if d.Summary == "" && len(r.WorkerResult.Blockers) > 0 {
					d.Summary = strings.Join(r.WorkerResult.Blockers, "; ")
				}
				if report, ok := scoutReportFromWorkerResult(r.WorkerResult); ok {
					d.ScoutReport = &report
				}
			}
			if strings.TrimSpace(d.Summary) == "" && r.Error != nil {
				d.Summary = strings.TrimSpace(r.Error.Error())
			}
		}

		dispatches = append(dispatches, d)
	}

	return dispatches
}

func scoutReportForPlanningDispatches(goal string, survey codexSurveyContext, dispatches []codexPlanningDispatch) codexScoutReport {
	for _, dispatch := range dispatches {
		if dispatch.ScoutReport == nil {
			continue
		}
		report := normalizeScoutPlanningReport(*dispatch.ScoutReport)
		if scoutReportHasContent(report) {
			return report
		}
	}
	return synthesizeScoutPlanningReport(goal, survey)
}

func scoutReportFromWorkerResult(result *codex.WorkerResult) (codexScoutReport, bool) {
	if result == nil {
		return codexScoutReport{}, false
	}
	raw := result.ScoutReport
	if strings.TrimSpace(string(raw)) == "" && result.Artifacts != nil {
		raw = result.Artifacts["scout_report"]
	}
	if strings.TrimSpace(string(raw)) == "" || strings.TrimSpace(string(raw)) == "null" {
		return codexScoutReport{}, false
	}
	var report codexScoutReport
	if err := json.Unmarshal(raw, &report); err != nil {
		return codexScoutReport{}, false
	}
	report = normalizeScoutPlanningReport(report)
	if !scoutReportHasContent(report) {
		return codexScoutReport{}, false
	}
	return report, true
}

func normalizeScoutPlanningReport(report codexScoutReport) codexScoutReport {
	findings := make([]codexScoutFinding, 0, len(report.Findings))
	for _, finding := range report.Findings {
		area := strings.TrimSpace(finding.Area)
		discovery := strings.TrimSpace(finding.Discovery)
		source := strings.TrimSpace(finding.Source)
		if area == "" && discovery == "" {
			continue
		}
		if area == "" {
			area = "planning"
		}
		if source == "" {
			source = "worker scout result"
		}
		findings = append(findings, codexScoutFinding{
			Area:      area,
			Discovery: discovery,
			Source:    source,
		})
	}
	report.Findings = limitScoutFindings(findings, 5)
	report.Gaps = limitStrings(uniqueSortedStrings(report.Gaps), 4)
	report.StudyFiles = limitStrings(uniqueSortedStrings(report.StudyFiles), 8)
	if report.Confidence <= 0 {
		report.Confidence = 60
	} else {
		report.Confidence = clampInt(report.Confidence, 1, 100)
	}
	return report
}

func limitScoutFindings(findings []codexScoutFinding, limit int) []codexScoutFinding {
	if len(findings) <= limit {
		return append([]codexScoutFinding{}, findings...)
	}
	return append([]codexScoutFinding{}, findings[:limit]...)
}

func scoutReportHasContent(report codexScoutReport) bool {
	return len(report.Findings) > 0 || len(report.Gaps) > 0 || len(report.StudyFiles) > 0
}

func loadCodexSurveyContext(root string) (codexSurveyContext, error) {
	surveyDir := ""
	if store != nil {
		surveyDir = filepath.Join(store.BasePath(), "survey")
	}
	ctx := codexSurveyContext{
		SurveyDir:        surveyDir,
		SurveyDocs:       []string{},
		Languages:        []string{},
		Frameworks:       []string{},
		Directories:      []string{},
		EntryPoints:      []string{},
		Dependencies:     []string{},
		TestFiles:        []string{},
		Issues:           []string{},
		SecurityPatterns: []string{},
	}

	for _, name := range []string{"PROVISIONS.md", "TRAILS.md", "BLUEPRINT.md", "CHAMBERS.md", "DISCIPLINES.md", "SENTINEL-PROTOCOLS.md", "PATHOGENS.md"} {
		if surveyDir == "" {
			break
		}
		if _, err := os.Stat(filepath.Join(surveyDir, name)); err == nil {
			ctx.SurveyDocs = append(ctx.SurveyDocs, name)
		}
	}

	readSummary := func(file string) map[string]interface{} {
		if surveyDir == "" {
			return nil
		}
		data, err := os.ReadFile(filepath.Join(surveyDir, file))
		if err != nil {
			return nil
		}
		var payload map[string]interface{}
		if err := json.Unmarshal(data, &payload); err != nil {
			return nil
		}
		return payload
	}

	if payload := readSummary("provisions.json"); payload != nil {
		ctx.Languages = append(ctx.Languages, jsonStringSlice(payload["languages"])...)
		ctx.Dependencies = append(ctx.Dependencies, jsonStringSlice(payload["dependencies"])...)
	}
	if payload := readSummary("blueprint.json"); payload != nil {
		ctx.Frameworks = append(ctx.Frameworks, jsonStringSlice(payload["frameworks"])...)
		ctx.EntryPoints = append(ctx.EntryPoints, jsonStringSlice(payload["entry_points"])...)
	}
	if payload := readSummary("chambers.json"); payload != nil {
		ctx.Directories = append(ctx.Directories, jsonStringSlice(payload["directories"])...)
	}
	if payload := readSummary("disciplines.json"); payload != nil {
		ctx.TestFiles = append(ctx.TestFiles, jsonStringSlice(payload["tests"])...)
	}
	if payload := readSummary("pathogens.json"); payload != nil {
		ctx.Issues = append(ctx.Issues, jsonStringSlice(payload["issues"])...)
	}

	facts, err := surveyWorkspace(root)
	if err == nil {
		ctx.Languages = append(ctx.Languages, facts.Languages...)
		ctx.Frameworks = append(ctx.Frameworks, facts.Frameworks...)
		ctx.Directories = append(ctx.Directories, facts.TopLevelDirs...)
		ctx.EntryPoints = append(ctx.EntryPoints, facts.EntryPoints...)
		ctx.Dependencies = append(ctx.Dependencies, facts.KeyDependencies...)
		ctx.TestFiles = append(ctx.TestFiles, facts.TestFiles...)
		ctx.SecurityPatterns = append(ctx.SecurityPatterns, facts.SecurityPatterns...)
		if len(ctx.Issues) == 0 {
			ctx.Issues = identifyPathogens(facts)
		}
	}

	ctx.SurveyDocs = uniqueSortedStrings(ctx.SurveyDocs)
	ctx.Languages = uniqueSortedStrings(ctx.Languages)
	ctx.Frameworks = uniqueSortedStrings(ctx.Frameworks)
	ctx.Directories = uniqueSortedStrings(ctx.Directories)
	ctx.EntryPoints = uniqueSortedStrings(ctx.EntryPoints)
	ctx.Dependencies = uniqueSortedStrings(ctx.Dependencies)
	ctx.TestFiles = uniqueSortedStrings(ctx.TestFiles)
	ctx.Issues = uniqueSortedStrings(ctx.Issues)
	ctx.SecurityPatterns = uniqueSortedStrings(ctx.SecurityPatterns)
	return ctx, nil
}

func jsonStringSlice(raw interface{}) []string {
	switch value := raw.(type) {
	case []string:
		return append([]string{}, value...)
	case []interface{}:
		result := make([]string, 0, len(value))
		for _, entry := range value {
			if text, ok := entry.(string); ok && strings.TrimSpace(text) != "" {
				result = append(result, text)
			}
		}
		return result
	default:
		return nil
	}
}

func uniqueSortedStrings(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func synthesizeScoutPlanningReport(goal string, survey codexSurveyContext) codexScoutReport {
	report := codexScoutReport{
		Findings:   []codexScoutFinding{},
		Gaps:       []string{},
		Confidence: 60,
		StudyFiles: []string{},
	}

	if len(survey.SurveyDocs) > 0 {
		report.Findings = append(report.Findings, codexScoutFinding{
			Area:      "territory survey",
			Discovery: fmt.Sprintf("Existing survey artifacts are available (%s), so planning can build on repo-specific context instead of starting blind.", strings.Join(limitStrings(survey.SurveyDocs, 4), ", ")),
			Source:    filepath.Join(".aether", "data", "survey"),
		})
	}
	if len(survey.EntryPoints) > 0 || len(survey.Directories) > 0 {
		report.Findings = append(report.Findings, codexScoutFinding{
			Area:      "architecture",
			Discovery: fmt.Sprintf("Primary execution surfaces live around %s, with key directories %s.", renderCSV(limitStrings(survey.EntryPoints, 3), "no explicit entry points"), renderCSV(limitStrings(survey.Directories, 4), "no top-level directories")),
			Source:    "BLUEPRINT.md / CHAMBERS.md",
		})
	}
	if len(survey.Languages) > 0 || len(survey.Frameworks) > 0 || len(survey.Dependencies) > 0 {
		report.Findings = append(report.Findings, codexScoutFinding{
			Area:      "stack",
			Discovery: fmt.Sprintf("The implementation surface spans %s with frameworks %s and dependencies such as %s.", renderCSV(limitStrings(survey.Languages, 3), "unknown languages"), renderCSV(limitStrings(survey.Frameworks, 4), "no framework markers"), renderCSV(limitStrings(survey.Dependencies, 5), "no obvious dependency anchors")),
			Source:    "PROVISIONS.md / TRAILS.md",
		})
	}
	if len(survey.TestFiles) > 0 {
		report.Findings = append(report.Findings, codexScoutFinding{
			Area:      "verification",
			Discovery: fmt.Sprintf("Existing test coverage already exercises representative paths like %s.", renderCSV(limitStrings(survey.TestFiles, 4), "no tests")),
			Source:    "DISCIPLINES.md / SENTINEL-PROTOCOLS.md",
		})
	} else {
		report.Gaps = append(report.Gaps, "No obvious test files were detected, so the plan must reserve explicit verification work.")
	}
	if len(survey.Issues) > 0 {
		report.Findings = append(report.Findings, codexScoutFinding{
			Area:      "risk",
			Discovery: fmt.Sprintf("Known repository concerns already include %s.", renderCSV(limitStrings(survey.Issues, 2), "no documented risks")),
			Source:    "PATHOGENS.md",
		})
	}

	if len(survey.SurveyDocs) == 0 {
		report.Gaps = append(report.Gaps, "No territory survey artifacts were present, so the plan relies on direct filesystem inspection only.")
	}
	if len(survey.EntryPoints) == 0 {
		report.Gaps = append(report.Gaps, "No obvious entry points were detected, so implementation ownership must be inferred from directories and tests.")
	}
	if len(survey.Dependencies) == 0 {
		report.Gaps = append(report.Gaps, "Dependency manifests did not provide many anchors, so integration boundaries may still need confirmation during build.")
	}
	report.Gaps = limitStrings(uniqueSortedStrings(report.Gaps), 3)

	report.StudyFiles = append(report.StudyFiles, survey.EntryPoints...)
	report.StudyFiles = append(report.StudyFiles, survey.TestFiles...)
	report.StudyFiles = limitStrings(uniqueSortedStrings(report.StudyFiles), 6)

	confidence := 55 + len(report.Findings)*6 + len(limitStrings(survey.SurveyDocs, 6))*3
	if len(report.StudyFiles) > 0 {
		confidence += 8
	}
	confidence -= len(report.Gaps) * 4
	report.Confidence = clampInt(confidence, 55, 94)
	if len(report.Findings) > 5 {
		report.Findings = append([]codexScoutFinding{}, report.Findings[:5]...)
	}
	return report
}

func limitStrings(values []string, limit int) []string {
	if len(values) <= limit {
		return append([]string{}, values...)
	}
	return append([]string{}, values[:limit]...)
}

func clampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func renderScoutPlanningGuidance(report codexScoutReport) string {
	report = normalizeScoutPlanningReport(report)
	if !scoutReportHasContent(report) {
		return ""
	}
	var b strings.Builder
	if len(report.Findings) > 0 {
		b.WriteString("Findings:\n")
		for _, finding := range report.Findings {
			b.WriteString(fmt.Sprintf("- %s: %s (source: %s)\n", finding.Area, finding.Discovery, finding.Source))
		}
	}
	if len(report.Gaps) > 0 {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString("Gaps:\n")
		for _, gap := range report.Gaps {
			b.WriteString(fmt.Sprintf("- %s\n", gap))
		}
	}
	if len(report.StudyFiles) > 0 {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString("Study files:\n")
		for _, file := range report.StudyFiles {
			b.WriteString(fmt.Sprintf("- %s\n", file))
		}
	}
	if report.Confidence > 0 {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(fmt.Sprintf("Confidence: %d%%", report.Confidence))
	}
	return strings.TrimSpace(b.String())
}

func renderPlanningWorkerBrief(root string, survey codexSurveyContext, spec planningWorkerSpec, scoutGuidanceOpt ...string) string {
	planningDir := filepath.ToSlash(filepath.Join(".aether", "data", "planning"))
	surveyDir := filepath.ToSlash(filepath.Join(".aether", "data", "survey"))
	phaseResearchDir := filepath.ToSlash(filepath.Join(".aether", "data", "phase-research"))
	primaryOutputs := make([]string, 0, len(spec.Outputs))
	for _, output := range spec.Outputs {
		primaryOutputs = append(primaryOutputs, filepath.ToSlash(filepath.Join(planningDir, output)))
	}
	surveyDocs := make([]string, 0, len(survey.SurveyDocs))
	for _, name := range survey.SurveyDocs {
		surveyDocs = append(surveyDocs, filepath.ToSlash(filepath.Join(surveyDir, name)))
	}

	var b strings.Builder
	b.WriteString("Planning task: ")
	b.WriteString(spec.Task)
	b.WriteString("\n\n")
	b.WriteString("Use the existing survey artifacts first before scanning the wider repository.\n")
	b.WriteString("- Primary survey source: ")
	b.WriteString(surveyDir)
	b.WriteString("\n")
	if len(surveyDocs) > 0 {
		b.WriteString("- Survey docs to read first: ")
		b.WriteString(strings.Join(surveyDocs, ", "))
		b.WriteString("\n")
	} else {
		b.WriteString("- Survey docs to read first: none detected; inspect repo files only when the survey is missing or ambiguous.\n")
	}
	if spec.Caste == "route_setter" {
		b.WriteString("- Read scout output before drafting phases if it exists: ")
		b.WriteString(filepath.ToSlash(filepath.Join(planningDir, "SCOUT.md")))
		b.WriteString("; if it is missing, proceed from the survey context and note the missing scout artifact in blockers only if it prevents a useful plan.")
		b.WriteString("\n")
	}
	b.WriteString("- Repo inspection rule: use targeted reads to confirm or extend survey findings; do not trawl the whole tree unless the survey lacks the needed detail.\n")
	b.WriteString("- Avoid high-noise paths unless directly relevant: .aether/backups/, .aether/chambers/, .aether/data/build/, .git/, node_modules/, dist/, build/, vendor/.\n")
	b.WriteString("- Loop guard: read each file at most once, do not reread the same command or wrapper file for confidence, and stop searching when you have enough evidence to produce the requested terminal result.\n")
	if spec.Caste == "scout" {
		b.WriteString("- Scout scope rule: stay read-only, do not spawn subagents, and stop after the survey docs plus targeted confirmation reads are enough to summarize planning risks.\n")
		b.WriteString("- Scout read budget: read survey docs first, then at most 8 targeted repository files. If evidence is still incomplete, record the gap instead of continuing to search.\n")
	}
	graphTargets := append(append([]string{}, survey.EntryPoints...), survey.TestFiles...)
	if graphContext := renderCodegraphContextForText(root, graphTargets, codegraphWorkerContextBudgetChars); graphContext != "" {
		b.WriteString("\n")
		b.WriteString(graphContext)
		b.WriteString("\n\n")
	}
	if spec.Caste == "scout" {
		b.WriteString("Return planning findings in the final worker claims JSON only.\n")
		b.WriteString("- Do not write files; Scout is read-only on every supported platform.\n")
		b.WriteString("- Final result must include scout_report with findings, gaps, confidence, and study_files. Keep it compact enough for the Route-Setter to consume directly.\n")
		b.WriteString("- Aether will persist the scout artifact after the worker completes: ")
		b.WriteString(strings.Join(primaryOutputs, ", "))
		b.WriteString("\n")
	} else if spec.Caste == "gatekeeper" || spec.Caste == "auditor" {
		b.WriteString("This is a review task. You may persist findings to your domain review ledger using `aether review-ledger-write`, but do not modify repo source files. Return status `blocked` if advancement is unsafe.\n\n")
		b.WriteString("Write planning outputs directly into the repository.\n")
		b.WriteString("- Primary outputs: ")
		b.WriteString(strings.Join(primaryOutputs, ", "))
		b.WriteString("\n")
		b.WriteString("- Planning dir: ")
		b.WriteString(planningDir)
		b.WriteString("\n")
		b.WriteString("- Phase research dir: ")
		b.WriteString(phaseResearchDir)
		b.WriteString("\n")
		b.WriteString("- Also write a machine-readable plan artifact at ")
		b.WriteString(filepath.ToSlash(filepath.Join(planningDir, "phase-plan.json")))
		b.WriteString(" using this JSON shape:\n")
		b.WriteString(`  {"phases":[{"name":"","description":"","tasks":[{"goal":"","constraints":[],"hints":[],"success_criteria":[],"depends_on":[]}],"success_criteria":[]}],"confidence":{"knowledge":0,"requirements":0,"risks":0,"dependencies":0,"effort":0,"overall":0},"gaps":[]}` + "\n")
	} else {
		scoutGuidance := ""
		if len(scoutGuidanceOpt) > 0 {
			scoutGuidance = strings.TrimSpace(scoutGuidanceOpt[0])
		}
		if scoutGuidance == "" {
			scoutGuidance = renderScoutPlanningGuidance(synthesizeScoutPlanningReport("", survey))
		}
		if scoutGuidance != "" {
			b.WriteString("## Scout Planning Guidance\n")
			b.WriteString(scoutGuidance)
			b.WriteString("\n\n")
		}
		b.WriteString("Write planning outputs directly into the repository.\n")
		b.WriteString("- Route-Setter read budget: consume the manifest survey context and the Scout terminal result provided by the wrapper, then read at most 6 targeted repository files. Do not redo the Scout survey.\n")
		b.WriteString("- If the Scout result is missing, continue from the manifest survey context and list that as a gap only if it blocks a useful plan.\n")
		b.WriteString("- Primary outputs: ")
		b.WriteString(strings.Join(primaryOutputs, ", "))
		b.WriteString("\n")
		b.WriteString("- Planning dir: ")
		b.WriteString(planningDir)
		b.WriteString("\n")
		b.WriteString("- Phase research dir: ")
		b.WriteString(phaseResearchDir)
		b.WriteString("\n")
		b.WriteString("- Also write a machine-readable plan artifact at ")
		b.WriteString(filepath.ToSlash(filepath.Join(planningDir, "phase-plan.json")))
		b.WriteString(" using this JSON shape:\n")
		b.WriteString(`  {"phases":[{"name":"","description":"","tasks":[{"goal":"","constraints":[],"hints":[],"success_criteria":[],"depends_on":[]}],"success_criteria":[]}],"confidence":{"knowledge":0,"requirements":0,"risks":0,"dependencies":0,"effort":0,"overall":0},"gaps":[]}` + "\n")
	}
	b.WriteString("\nPlan the colony at ")
	b.WriteString(root)
	return b.String()
}

func claimedPlanningFiles(dispatches []codexPlanningDispatch) map[string]bool {
	claimed := map[string]bool{}
	for _, dispatch := range dispatches {
		for relPath := range claimedArtifactSet(dispatch.Claimed) {
			claimed[relPath] = true
		}
	}
	return claimed
}

func loadWorkerPlanArtifact(root string, snapshots map[string]codexArtifactSnapshot, dispatches []codexPlanningDispatch) (codexWorkerPlanArtifact, bool, string) {
	relPath := filepath.ToSlash(filepath.Join(".aether", "data", "planning", "phase-plan.json"))
	if !shouldPreserveWorkerArtifact(root, relPath, snapshots, claimedPlanningFiles(dispatches)) {
		return codexWorkerPlanArtifact{}, false, ""
	}

	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relPath)))
	if err != nil {
		return codexWorkerPlanArtifact{}, false, "Route-setter wrote phase-plan.json but it could not be read, so planning fell back to local synthesis."
	}

	var artifact codexWorkerPlanArtifact
	if err := json.Unmarshal(data, &artifact); err != nil {
		return codexWorkerPlanArtifact{}, false, "Route-setter phase-plan.json was invalid, so planning fell back to local synthesis."
	}
	if len(artifact.Phases) == 0 {
		return codexWorkerPlanArtifact{}, false, "Route-setter phase-plan.json contained no phases, so planning fell back to local synthesis."
	}
	return artifact, true, ""
}

func buildWorkerPlanPhases(artifact codexWorkerPlanArtifact) []colony.Phase {
	phases := make([]colony.Phase, 0, len(artifact.Phases))
	for i, sourcePhase := range artifact.Phases {
		phase := colony.Phase{
			ID:              i + 1,
			Name:            strings.TrimSpace(sourcePhase.Name),
			Description:     strings.TrimSpace(sourcePhase.Description),
			Status:          colony.PhasePending,
			Mode:            colony.InferPhaseMode(sourcePhase.Name, sourcePhase.Description),
			Tasks:           []colony.Task{},
			SuccessCriteria: uniqueSortedStrings(sourcePhase.SuccessCriteria),
		}
		if phase.Name == "" {
			phase.Name = fmt.Sprintf("Phase %d", i+1)
		}
		if i == 0 {
			phase.Status = colony.PhaseReady
		}
		for j, sourceTask := range sourcePhase.Tasks {
			goal := strings.TrimSpace(sourceTask.Goal)
			if goal == "" {
				continue
			}
			taskID := fmt.Sprintf("%d.%d", i+1, j+1)
			phase.Tasks = append(phase.Tasks, colony.Task{
				ID:              &taskID,
				Goal:            goal,
				Status:          colony.TaskPending,
				Constraints:     uniqueSortedStrings(sourceTask.Constraints),
				Hints:           uniqueSortedStrings(sourceTask.Hints),
				SuccessCriteria: uniqueSortedStrings(sourceTask.SuccessCriteria),
				DependsOn:       uniqueSortedStrings(sourceTask.DependsOn),
			})
		}
		phases = append(phases, phase)
	}
	return phases
}

func mergePlanConfidence(base codexPlanConfidence, override codexPlanConfidence) codexPlanConfidence {
	if override.Knowledge > 0 {
		base.Knowledge = override.Knowledge
	}
	if override.Requirements > 0 {
		base.Requirements = override.Requirements
	}
	if override.Risks > 0 {
		base.Risks = override.Risks
	}
	if override.Dependencies > 0 {
		base.Dependencies = override.Dependencies
	}
	if override.Effort > 0 {
		base.Effort = override.Effort
	}
	if override.Overall > 0 {
		base.Overall = override.Overall
	} else {
		base.Overall = int(float64(base.Knowledge)*0.25 +
			float64(base.Requirements)*0.25 +
			float64(base.Risks)*0.20 +
			float64(base.Dependencies)*0.15 +
			float64(base.Effort)*0.15 + 0.5)
	}
	return base
}

func writePlanningScoutArtifact(root, planningDir, goal string, granularity colony.PlanGranularity, survey codexSurveyContext, dispatch codexPlanningDispatch, report codexScoutReport, snapshots map[string]codexArtifactSnapshot) (string, bool, error) {
	path := filepath.Join(planningDir, "SCOUT.md")
	relPath := filepath.ToSlash(filepath.Join(".aether", "data", "planning", "SCOUT.md"))
	if shouldPreserveWorkerArtifact(root, relPath, snapshots, claimedArtifactSet(dispatch.Claimed)) {
		return path, true, nil
	}
	var b strings.Builder
	b.WriteString("# Planning Scout Report\n\n")
	b.WriteString(fmt.Sprintf("- Generated: %s\n", time.Now().UTC().Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("- Scout: %s\n", dispatch.Name))
	b.WriteString(fmt.Sprintf("- Goal: %s\n", goal))
	b.WriteString(fmt.Sprintf("- Granularity: %s\n\n", granularity))
	b.WriteString("## Findings\n")
	for _, finding := range report.Findings {
		b.WriteString(fmt.Sprintf("- **%s:** %s (Source: %s)\n", finding.Area, finding.Discovery, finding.Source))
	}
	if len(report.Findings) == 0 {
		b.WriteString("- No significant findings were synthesized.\n")
	}
	b.WriteString("\n## Gaps\n")
	b.WriteString(bulletList(report.Gaps, "No material knowledge gaps remain at planning time."))
	b.WriteString("\n\n## Study Files\n")
	b.WriteString(bulletList(report.StudyFiles, "No representative files were identified."))
	b.WriteString("\n\n## Survey Inputs\n")
	b.WriteString(bulletList(survey.SurveyDocs, "No territory survey docs were available."))
	b.WriteString("\n")
	if err := os.WriteFile(path, []byte(b.String()), 0644); err != nil {
		return "", false, fmt.Errorf("failed to write planning scout report: %w", err)
	}
	return path, false, nil
}

func synthesizeRouteSetterPlan(goal string, granularity colony.PlanGranularity, survey codexSurveyContext, report codexScoutReport) ([]colony.Phase, codexPlanConfidence, []string) {
	templates := planningTemplates(goal, survey, report)
	minPhases, maxPhases := colony.GranularityRange(granularity)
	count := len(templates)
	if count < minPhases {
		count = minPhases
	}
	if count > maxPhases {
		count = maxPhases
	}
	if len(templates) > count {
		templates = append([]phaseTemplate{}, templates[:count]...)
	}

	redirectConstraints := activeRedirectPlanConstraints()
	phases := make([]colony.Phase, 0, len(templates))
	for i, template := range templates {
		phase := colony.Phase{
			ID:              i + 1,
			Name:            template.Name,
			Description:     template.Description,
			Status:          colony.PhasePending,
			Mode:            colony.InferPhaseMode(template.Name, template.Description),
			Tasks:           []colony.Task{},
			SuccessCriteria: append([]string{}, template.SuccessCriteria...),
		}
		if i == 0 {
			phase.Status = colony.PhaseReady
		}
		for j, taskTemplate := range template.Tasks {
			taskID := fmt.Sprintf("%d.%d", i+1, j+1)
			constraints := append([]string{}, redirectConstraints...)
			constraints = append(constraints, taskTemplate.Constraints...)
			phase.Tasks = append(phase.Tasks, colony.Task{
				ID:              &taskID,
				Goal:            taskTemplate.Goal,
				Status:          colony.TaskPending,
				Constraints:     uniqueSortedStrings(constraints),
				Hints:           uniqueSortedStrings(taskTemplate.Hints),
				SuccessCriteria: uniqueSortedStrings(taskTemplate.SuccessCriteria),
				DependsOn:       uniqueSortedStrings(taskTemplate.DependsOn),
			})
		}
		phases = append(phases, phase)
	}

	confidence := codexPlanConfidence{
		Knowledge:    clampInt(report.Confidence, 55, 96),
		Requirements: clampInt(70+len(templates)*2, 68, 94),
		Risks:        clampInt(88-len(survey.Issues)*4-len(report.Gaps)*5, 55, 90),
		Dependencies: clampInt(60+len(survey.EntryPoints)*5+len(survey.Dependencies)*2, 58, 92),
		Effort:       clampInt(80-len(templates), 62, 88),
	}
	confidence.Overall = int(float64(confidence.Knowledge)*0.25 +
		float64(confidence.Requirements)*0.25 +
		float64(confidence.Risks)*0.20 +
		float64(confidence.Dependencies)*0.15 +
		float64(confidence.Effort)*0.15 + 0.5)

	unresolved := append([]string{}, report.Gaps...)
	if len(survey.Issues) > 0 {
		unresolved = append(unresolved, fmt.Sprintf("Known repo risks remain active: %s.", renderCSV(limitStrings(survey.Issues, 2), "none")))
	}
	return phases, confidence, limitStrings(uniqueSortedStrings(unresolved), 3)
}

func planningTemplates(goal string, survey codexSurveyContext, report codexScoutReport) []phaseTemplate {
	goalLower := strings.ToLower(goal)
	switch {
	case isAetherOrchestrationGoal(goalLower):
		return []phaseTemplate{
			{
				Name:        "Contract and gap mapping",
				Description: "Lock the expected ant-process behavior against the current Go command surface.",
				Tasks: []phaseTaskTemplate{
					{
						Goal:            "Compare the documented ant workflow with the current Codex command behavior",
						Constraints:     []string{"Use Claude/OpenCode command specs as the external contract", "Keep the Go binary as the execution source of truth"},
						Hints:           []string{".claude/commands/ant/colonize.md", ".claude/commands/ant/plan.md", "cmd/codex_workflow_cmds.go"},
						SuccessCriteria: []string{"The parity gaps are explicit", "The next implementation slices are dependency-ordered"},
					},
					{
						Goal:            "Decide the observable ant-process outputs Codex must emit during each core command",
						Constraints:     []string{"Do not claim capabilities the Go binary does not actually perform"},
						Hints:           append(commonHints(survey), report.StudyFiles...),
						SuccessCriteria: []string{"Each command has a concrete dispatch contract", "Spawn tree and artifacts are part of the contract"},
					},
				},
				SuccessCriteria: []string{"The parity contract is explicit", "The colony has an honest execution target"},
			},
			{
				Name:        "Colonize orchestration",
				Description: "Deliver surveyor-driven territory mapping that writes usable survey artifacts and spawn records.",
				Tasks: []phaseTaskTemplate{
					{
						Goal:            "Implement the surveyor workflow for Codex colonize",
						Constraints:     commonConstraints(survey),
						Hints:           []string{"cmd/codex_colonize.go", "spawn-tree.txt", ".aether/data/survey/"},
						SuccessCriteria: []string{"Survey artifacts are written", "Surveyor spawns are recorded and completed"},
					},
					{
						Goal:            "Keep survey artifacts stable enough for planning to consume",
						Constraints:     []string{"Avoid polluting reports with cache or generated artifacts"},
						Hints:           []string{"PATHOGENS.md should only mention real repo concerns"},
						SuccessCriteria: []string{"Survey artifacts are reusable inputs for plan", "Noise from generated/cache files is excluded"},
					},
				},
				SuccessCriteria: []string{"Territory survey is reliable", "Planning can trust the survey output"},
			},
			{
				Name:        "Planning orchestration",
				Description: "Add a scout plus route-setter planning pass that produces artifacts and a grounded phase plan.",
				Tasks: []phaseTaskTemplate{
					{
						Goal:            "Implement a scout planning pass that summarizes survey findings into buildable guidance",
						Constraints:     []string{"Scout output must be persisted as a planning artifact", "Planning must still work when survey docs are missing"},
						Hints:           []string{"SCOUT.md", "phase-research/", "cmd/codex_plan.go"},
						SuccessCriteria: []string{"Planning produces a scout artifact", "The route-setter consumes scout output and survey context"},
					},
					{
						Goal:            "Generate a route-setter plan with task constraints, hints, and success criteria",
						Constraints:     []string{"The first phase must become ready", "The saved plan must match the displayed plan"},
						Hints:           []string{"COLONY_STATE.json", "renderPlanVisual"},
						SuccessCriteria: []string{"Plan generation is grounded in repo context", "Spawn records show scout and route-setter activity"},
					},
				},
				SuccessCriteria: []string{"Plan generation is ant-driven", "The colony has a grounded next phase"},
			},
			{
				Name:        "Build orchestration",
				Description: "Replace visual-only build dispatch with real worker execution slices and artifact handling.",
				Tasks: []phaseTaskTemplate{
					{
						Goal:            "Define and implement the builder, watcher, and specialist work sequence for build",
						Constraints:     []string{"Keep command behavior inside Go", "Recorded spawns must reflect real work performed"},
						Hints:           []string{"cmd/codex_workflow_cmds.go", "cmd/codex_visuals.go", ".claude/commands/ant/build.md"},
						SuccessCriteria: []string{"Build does more than set state", "The spawn plan matches actual command behavior"},
					},
					{
						Goal:            "Persist build artifacts and phase context needed by continue",
						Constraints:     []string{"Continue must not rely on implied work that never happened"},
						Hints:           commonHints(survey),
						SuccessCriteria: []string{"Continue can verify concrete outputs", "Phase state stays consistent across reruns"},
					},
				},
				SuccessCriteria: []string{"Build behavior matches its visuals", "Continue has real evidence to consume"},
			},
			{
				Name:        "Continue orchestration",
				Description: "Make continue perform the real verification and advancement work instead of only flipping statuses.",
				Tasks: []phaseTaskTemplate{
					{
						Goal:            "Implement watcher-led verification and housekeeping before phase advancement",
						Constraints:     []string{"Signal housekeeping should stay wired into continue finalize", "Completed phases must only advance after verification"},
						Hints:           []string{"cmd/signal_housekeeping.go", ".claude/commands/ant/continue.md"},
						SuccessCriteria: []string{"Continue verifies actual artifacts", "Advance only happens after verification succeeds"},
					},
					{
						Goal:            "Record the continue worker flow in state, spawn logs, and user-facing output",
						Constraints:     []string{"The Next Up block must stay valid for the resulting state"},
						Hints:           []string{"renderContinueVisual", "print-next-up"},
						SuccessCriteria: []string{"Continue output matches actual work", "The next phase becomes ready when appropriate"},
					},
				},
				SuccessCriteria: []string{"Continue is no longer a pure state flip", "Phase advancement is defensible"},
			},
			{
				Name:        "End-to-end verification",
				Description: "Run the colony from init through seal and prove the Codex ant process is now honest.",
				Tasks: []phaseTaskTemplate{
					{
						Goal:            "Add tests that prove colonize, plan, build, and continue record real worker activity",
						Constraints:     []string{"Test the user path, not just helper functions"},
						Hints:           append([]string{"cmd/codex_colonize_test.go"}, survey.TestFiles...),
						SuccessCriteria: []string{"Core parity regressions are caught by tests", "Spawn tree entries are asserted in tests"},
					},
					{
						Goal:            "Run a live colony loop and compare its outputs with the documented ant process",
						Constraints:     []string{"Call out any remaining parity gap explicitly"},
						Hints:           []string{"spawn-tree.txt", "COLONY_STATE.json", ".aether/data/survey/"},
						SuccessCriteria: []string{"The live loop proves the implemented parity", "Remaining gaps are small and explicit"},
					},
				},
				SuccessCriteria: []string{"The live colony loop is credible", "The remaining parity gap is narrow and testable"},
			},
		}
	case isLanguageDesignGoal(goalLower):
		return []phaseTemplate{
			{
				Name:        "Research charter and communication target",
				Description: "Define what the language or protocol is for, who writes it, and what efficiency or expressiveness problem it must solve.",
				Tasks: []phaseTaskTemplate{
					{
						Goal:            "Define the communication problem, target agents, and success criteria for the language",
						Constraints:     []string{"Keep the first charter narrow enough to prototype quickly", "State what efficiency or context-saving win should be measurable"},
						Hints:           append([]string{"README.md", "SCOUT.md"}, report.StudyFiles...),
						SuccessCriteria: []string{"The language has a bounded first mission", "Non-goals and evaluation criteria are explicit"},
					},
					{
						Goal:            "Capture the core semantic primitives the language needs to express",
						Constraints:     []string{"Separate semantics from syntax", "Avoid inventing syntax before the information model is clear"},
						Hints:           []string{"Grammar sketch", "Message categories", "Information density targets"},
						SuccessCriteria: []string{"Core message categories are named", "The minimum expressive set is explicit"},
					},
				},
				SuccessCriteria: []string{"The language charter is explicit", "The research target is narrow enough to explore concretely"},
			},
			{
				Name:        "Representation and grammar design",
				Description: "Design the structural representation, encoding rules, and grammar needed to carry the chosen semantic primitives.",
				Tasks: []phaseTaskTemplate{
					{
						Goal:            "Choose the first representation model for messages and state transitions",
						Constraints:     []string{"Prefer one reference representation first", "Document how the representation optimizes context or token use"},
						Hints:           []string{"Abstract syntax tree", "Schema sketch", "Field density trade-offs"},
						SuccessCriteria: []string{"A first representation model exists", "Trade-offs are recorded"},
					},
					{
						Goal:            "Draft the first grammar, syntax, or encoding rules for that representation",
						Constraints:     []string{"Keep the first grammar intentionally small", "Show at least one end-to-end example message"},
						Hints:           []string{"Parser/lexer sketch", "Serialization rules", "Example transcripts"},
						SuccessCriteria: []string{"The first grammar can encode representative examples", "Syntax and semantics stay aligned"},
					},
				},
				SuccessCriteria: []string{"The representation is concrete", "The grammar is specific enough to prototype"},
			},
			{
				Name:        "Reference prototype and translation path",
				Description: "Build the smallest reference implementation that can read, write, or validate the first slice of the language.",
				Tasks: []phaseTaskTemplate{
					{
						Goal:            "Create a minimal reference prototype for parsing, validating, or emitting the first message slice",
						Constraints:     []string{"Prototype only the first useful slice", "Prefer a small reference tool over a full compiler/runtime"},
						Hints:           append(commonHints(survey), "Parser", "Validator", "Encoder"),
						SuccessCriteria: []string{"A concrete prototype exists", "Examples can flow through the prototype"},
					},
					{
						Goal:            "Create worked examples that demonstrate the language's communication advantage",
						Constraints:     []string{"Examples must compare the new format with a plain-language baseline"},
						Hints:           []string{"Before/after transcripts", "Compression examples", "Agent-to-agent exchange"},
						SuccessCriteria: []string{"Representative examples exist", "The examples expose strengths and weaknesses honestly"},
					},
				},
				SuccessCriteria: []string{"The language is no longer only conceptual", "There is a concrete translation path for examples"},
			},
			{
				Name:        "Evaluation and next design loop",
				Description: "Evaluate the first prototype, capture what worked, and turn the findings into the next research or implementation slice.",
				Tasks: []phaseTaskTemplate{
					{
						Goal:            "Evaluate the prototype against the original communication and efficiency criteria",
						Constraints:     []string{"Do not hide unresolved ambiguities or poor trade-offs"},
						Hints:           []string{"Token count comparison", "Ambiguity review", "Error cases"},
						SuccessCriteria: []string{"The prototype is assessed against explicit criteria", "Weak points are visible"},
					},
					{
						Goal:            "Record design decisions, open questions, and the next experimental slice",
						Constraints:     []string{"Keep follow-ups framed as concrete experiments"},
						Hints:           survey.Issues,
						SuccessCriteria: []string{"The next iteration path is explicit", "The colony can continue from a grounded base"},
					},
				},
				SuccessCriteria: []string{"The first design loop is evaluated", "The next iteration is specific and evidence-driven"},
			},
		}
	case isGreenfieldResearchGoal(goalLower, survey):
		return []phaseTemplate{
			{
				Name:        "Problem framing and boundaries",
				Description: "Define the research target, the first hard constraints, and what a meaningful outcome would look like.",
				Tasks: []phaseTaskTemplate{
					{
						Goal:            "Turn the raw goal into a bounded research charter with explicit outcomes",
						Constraints:     []string{"Do not jump into implementation before the research target is concrete"},
						Hints:           append([]string{"README.md", "SCOUT.md"}, report.StudyFiles...),
						SuccessCriteria: []string{"The goal is narrowed into a research charter", "The first outcome is testable"},
					},
					{
						Goal:            "Identify the hardest unknowns and the assumptions most likely to invalidate the project",
						Constraints:     []string{"Surface the unknowns before structure ossifies"},
						Hints:           survey.Issues,
						SuccessCriteria: []string{"High-risk unknowns are explicit", "The next phase is shaped by the hardest questions"},
					},
				},
				SuccessCriteria: []string{"The problem is bounded", "The riskiest unknowns are visible"},
			},
			{
				Name:        "Research model and architecture groundwork",
				Description: "Design the first information model, architecture boundary, or experimental structure required to explore the goal.",
				Tasks: []phaseTaskTemplate{
					{
						Goal:            "Choose the first architecture or information model to explore the goal",
						Constraints:     []string{"Prefer a minimal model that exposes the core trade-offs"},
						Hints:           []string{"System boundaries", "Core entities", "Flow diagram"},
						SuccessCriteria: []string{"The core architecture is sketched", "Dependencies and interfaces are explicit"},
					},
					{
						Goal:            "Document the exploration structure and the artifacts the prototype will need",
						Constraints:     []string{"Keep the first artifact set lean"},
						Hints:           []string{"Spec doc", "Reference implementation", "Examples"},
						SuccessCriteria: []string{"The groundwork is concrete enough to build from", "The prototype path is explicit"},
					},
				},
				SuccessCriteria: []string{"The research has a concrete structure", "The first build slice is grounded"},
			},
			{
				Name:        "Prototype the first end-to-end slice",
				Description: "Build the smallest possible slice that exercises the core idea end to end.",
				Tasks: []phaseTaskTemplate{
					{
						Goal:            "Implement a minimal prototype for the first meaningful slice",
						Constraints:     []string{"Keep scope tight", "Demonstrate the core idea, not every edge case"},
						Hints:           commonHints(survey),
						SuccessCriteria: []string{"A first end-to-end slice exists", "The prototype exposes the real trade-offs"},
					},
					{
						Goal:            "Capture examples, inputs, and outputs that explain what the prototype is proving",
						Constraints:     []string{"Examples must be understandable without hidden context"},
						Hints:           []string{"Example inputs", "Example outputs", "Reference walkthrough"},
						SuccessCriteria: []string{"The prototype is explainable", "The experiment can be repeated"},
					},
				},
				SuccessCriteria: []string{"There is a real prototype", "The prototype demonstrates the core claim"},
			},
			{
				Name:        "Evaluate, document, and re-scope",
				Description: "Evaluate the first results, capture decisions, and choose the next slice based on evidence.",
				Tasks: []phaseTaskTemplate{
					{
						Goal:            "Evaluate what the first prototype proved and where it failed",
						Constraints:     []string{"Do not overstate the result"},
						Hints:           []string{"Limitations", "Unexpected findings", "Operational risks"},
						SuccessCriteria: []string{"The result is judged honestly", "Evidence and limitations are both visible"},
					},
					{
						Goal:            "Record the next iteration path, including what to deepen, abandon, or test next",
						Constraints:     []string{"Next steps should be concrete experiments or build slices"},
						Hints:           survey.Issues,
						SuccessCriteria: []string{"The next loop is explicit", "The colony can continue from a stronger foundation"},
					},
				},
				SuccessCriteria: []string{"The first research loop is closed honestly", "The next loop is evidence-driven"},
			},
		}
	default:
		return []phaseTemplate{
			{
				Name:        "Discovery and boundaries",
				Description: "Map the relevant code paths, constraints, and success criteria before implementation.",
				Tasks: []phaseTaskTemplate{
					{
						Goal:            "Read the current implementation paths relevant to the goal",
						Constraints:     commonConstraints(survey),
						Hints:           commonHints(survey),
						SuccessCriteria: []string{"The working surface is explicit", "Key dependencies and boundaries are known"},
					},
					{
						Goal:            "Capture risks, constraints, and a testable target state",
						Constraints:     []string{"Keep the scope bounded to the requested outcome"},
						Hints:           survey.Issues,
						SuccessCriteria: []string{"Success criteria are explicit", "Known risks are visible before coding"},
					},
				},
				SuccessCriteria: []string{"The colony has a bounded target", "Implementation risks are known"},
			},
			{
				Name:        "Architecture and interfaces",
				Description: "Lock the first architecture boundary, ownership surface, and integration path before deeper coding.",
				Tasks: []phaseTaskTemplate{
					{
						Goal:            "Define the primary architecture boundary or module surface for the change",
						Constraints:     []string{"Prefer a narrow first ownership surface", "Keep the design anchored to the existing repo shape"},
						Hints:           append(commonHints(survey), report.StudyFiles...),
						SuccessCriteria: []string{"The implementation surface is chosen", "The integration path is explicit"},
					},
					{
						Goal:            "Identify the interfaces, contracts, or data boundaries that the implementation must respect",
						Constraints:     []string{"Document the boundaries before broad code changes begin"},
						Hints:           survey.Dependencies,
						SuccessCriteria: []string{"Key interfaces are explicit", "The build phase has a stable target"},
					},
				},
				SuccessCriteria: []string{"The architecture surface is explicit", "The implementation can proceed without guessing boundaries"},
			},
			{
				Name:        "Implementation",
				Description: "Make the core changes required to land the goal.",
				Tasks: []phaseTaskTemplate{
					{
						Goal:            "Implement the main behavior changes required by the goal",
						Constraints:     commonConstraints(survey),
						Hints:           commonHints(survey),
						SuccessCriteria: []string{"The main behavior lands", "The change integrates cleanly with the existing structure"},
					},
					{
						Goal:            "Update or add focused automated coverage",
						Constraints:     []string{"Use existing test patterns where possible"},
						Hints:           survey.TestFiles,
						SuccessCriteria: []string{"The new behavior is covered", "Important adjacent behavior is exercised"},
					},
				},
				SuccessCriteria: []string{"The goal is implemented", "Coverage exists for the changed behavior"},
			},
			{
				Name:        "Verification and polish",
				Description: "Verify the result, tighten loose ends, and prepare the colony for seal.",
				Tasks: []phaseTaskTemplate{
					{
						Goal:            "Run focused verification and address regressions",
						Constraints:     []string{"Prefer the smallest verification set that proves the result"},
						Hints:           survey.TestFiles,
						SuccessCriteria: []string{"Relevant verification is green", "Regressions are addressed"},
					},
					{
						Goal:            "Capture decisions, follow-ups, and user-visible changes",
						Constraints:     []string{"Do not hide residual risk"},
						Hints:           survey.Issues,
						SuccessCriteria: []string{"Key decisions are documented", "Remaining follow-ups are explicit"},
					},
				},
				SuccessCriteria: []string{"The result is verified", "The colony can move toward seal cleanly"},
			},
		}
	}
}

func isLanguageDesignGoal(goalLower string) bool {
	return containsAnyWholeWord(goalLower, []string{
		"language", "grammar", "syntax", "parser", "lexer", "compiler", "dsl",
		"protocol", "serialization", "encode", "decode", "format", "schema", "spec",
		"communication",
	}) || containsAny(goalLower, []string{
		"transpil", "token efficien", "context efficien", "ai-to-ai",
	})
}

func isAetherOrchestrationGoal(goalLower string) bool {
	if !containsAny(goalLower, []string{
		"parity", "orchestrat", "workflow", "command", "spawn",
		"lifecycle", "reliability", "platform", "dispatch", "finalizer", "read-loop",
	}) {
		return false
	}
	return containsAny(goalLower, []string{
		"aether", "codex", "claude", "opencode", "colony", "ant",
		"worker", "watcher", "builder", "scout", "route-setter",
		"plan-only", "finalize", "spawn tree", "dispatch",
	})
}

func isGreenfieldResearchGoal(goalLower string, survey codexSurveyContext) bool {
	if !containsAny(goalLower, []string{"research", "discover", "invent", "foundation", "groundwork", "architecture", "explore", "investigat", "design"}) {
		return false
	}
	return len(survey.EntryPoints) == 0 && len(survey.Dependencies) == 0 && len(survey.TestFiles) == 0 && len(survey.Frameworks) == 0
}

func containsAny(text string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}

func containsAnyWholeWord(text string, needles []string) bool {
	fields := strings.FieldsFunc(text, func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9')
	})
	for _, field := range fields {
		for _, needle := range needles {
			if field == needle {
				return true
			}
		}
	}
	return false
}

func commonConstraints(survey codexSurveyContext) []string {
	constraints := activeRedirectPlanConstraints()
	constraints = append(constraints,
		"Follow the repository's existing structure and conventions",
		"Keep changes scoped to the current goal",
	)
	if len(survey.Issues) > 0 {
		constraints = append(constraints, survey.Issues[0])
	}
	return limitStrings(appendUniqueStringsInOrder(constraints), 6)
}

func activeRedirectPlanConstraints() []string {
	if store == nil {
		return nil
	}
	var pf colony.PheromoneFile
	if err := store.LoadJSON("pheromones.json", &pf); err != nil {
		return nil
	}
	now := time.Now().UTC()
	constraints := []string{}
	for _, signal := range pf.Signals {
		if !signal.Active || !strings.EqualFold(signal.Type, "REDIRECT") || computeEffectiveStrength(signal, now) <= 0 {
			continue
		}
		text := strings.TrimSpace(extractContentText(signal.Content))
		if text != "" {
			constraints = append(constraints, text)
		}
	}
	return constraints
}

func appendUniqueStringsInOrder(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func commonHints(survey codexSurveyContext) []string {
	hints := []string{}
	hints = append(hints, survey.EntryPoints...)
	hints = append(hints, survey.TestFiles...)
	if len(survey.Directories) > 0 {
		hints = append(hints, fmt.Sprintf("Top-level dirs: %s", strings.Join(limitStrings(survey.Directories, 5), ", ")))
	}
	return limitStrings(uniqueSortedStrings(hints), 5)
}

func writeRouteSetterArtifact(root, planningDir, goal string, granularity colony.PlanGranularity, survey codexSurveyContext, dispatch codexPlanningDispatch, confidence codexPlanConfidence, unresolvedGaps []string, phases []colony.Phase, snapshots map[string]codexArtifactSnapshot) (string, bool, error) {
	path := filepath.Join(planningDir, "ROUTE-SETTER.md")
	relPath := filepath.ToSlash(filepath.Join(".aether", "data", "planning", "ROUTE-SETTER.md"))
	if shouldPreserveWorkerArtifact(root, relPath, snapshots, claimedArtifactSet(dispatch.Claimed)) {
		return path, true, nil
	}

	var b strings.Builder
	b.WriteString("# Route-Setter Plan\n\n")
	b.WriteString(fmt.Sprintf("- Generated: %s\n", time.Now().UTC().Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("- Route-Setter: %s\n", dispatch.Name))
	b.WriteString(fmt.Sprintf("- Goal: %s\n", goal))
	b.WriteString(fmt.Sprintf("- Granularity: %s (%d-%d phases)\n", granularity, granularityMin(granularity), granularityMax(granularity)))
	b.WriteString(fmt.Sprintf("- Confidence: %d%% overall\n\n", confidence.Overall))
	b.WriteString("## Unresolved Gaps\n")
	b.WriteString(bulletList(unresolvedGaps, "No planning gaps remain."))
	b.WriteString("\n\n## Survey Inputs\n")
	b.WriteString(bulletList(survey.SurveyDocs, "No survey docs were available."))
	b.WriteString("\n\n## Phase Outline\n")
	for _, phase := range phases {
		b.WriteString(fmt.Sprintf("- Phase %d: %s\n", phase.ID, phase.Name))
	}
	b.WriteString("\n")
	if err := os.WriteFile(path, []byte(b.String()), 0644); err != nil {
		return "", false, fmt.Errorf("failed to write route-setter artifact: %w", err)
	}
	return path, false, nil
}

func writeWorkerPlanArtifact(root, planningDir string, confidence codexPlanConfidence, unresolvedGaps []string, phases []colony.Phase, snapshots map[string]codexArtifactSnapshot, dispatches []codexPlanningDispatch) (string, bool, error) {
	path := filepath.Join(planningDir, "phase-plan.json")
	relPath := filepath.ToSlash(filepath.Join(".aether", "data", "planning", "phase-plan.json"))
	if shouldPreserveWorkerArtifact(root, relPath, snapshots, claimedPlanningFiles(dispatches)) {
		return path, true, nil
	}

	artifact := codexWorkerPlanArtifact{
		Confidence: confidence,
		Gaps:       limitStrings(uniqueSortedStrings(unresolvedGaps), 4),
		Phases:     make([]codexWorkerPlanPhase, 0, len(phases)),
	}
	for _, phase := range phases {
		entry := codexWorkerPlanPhase{
			Name:            phase.Name,
			Description:     phase.Description,
			Tasks:           make([]codexWorkerPlanTask, 0, len(phase.Tasks)),
			SuccessCriteria: uniqueSortedStrings(phase.SuccessCriteria),
		}
		for _, task := range phase.Tasks {
			entry.Tasks = append(entry.Tasks, codexWorkerPlanTask{
				Goal:            task.Goal,
				Constraints:     uniqueSortedStrings(task.Constraints),
				Hints:           uniqueSortedStrings(task.Hints),
				SuccessCriteria: uniqueSortedStrings(task.SuccessCriteria),
				DependsOn:       uniqueSortedStrings(task.DependsOn),
			})
		}
		artifact.Phases = append(artifact.Phases, entry)
	}

	data, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return "", false, fmt.Errorf("failed to marshal worker plan artifact: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0644); err != nil {
		return "", false, fmt.Errorf("failed to write worker plan artifact: %w", err)
	}
	return path, false, nil
}

func clearFallbackPlanningArtifacts(root string) {
	planningDir := filepath.Join(root, ".aether", "data", "planning")
	markerPath := filepath.Join(planningDir, ".fallback-marker")
	markerTime := time.Time{}
	if info, err := os.Stat(markerPath); err == nil {
		markerTime = info.ModTime()
	}

	// Always remove the marker itself
	os.Remove(markerPath)

	fallbackArtifacts := []string{
		filepath.Join(planningDir, "ROUTE-SETTER.md"),
		filepath.Join(planningDir, "phase-plan.json"),
	}
	for _, f := range fallbackArtifacts {
		// Only remove if the file predates or matches the fallback marker (it's a fallback artifact).
		// If the file is newer than the marker, a real worker wrote it — preserve it.
		if !markerTime.IsZero() {
			if info, err := os.Stat(f); err == nil && info.ModTime().After(markerTime) {
				continue
			}
		}
		os.Remove(f)
	}
	clearPlanningBackupArtifacts(planningDir)
	// Clear phase-research directory contents but keep the directory
	researchDir := filepath.Join(root, ".aether", "data", "phase-research")
	entries, err := os.ReadDir(researchDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		os.Remove(filepath.Join(researchDir, entry.Name()))
	}
}

func clearPlanningBackupArtifacts(planningDir string) {
	if bakFiles, err := filepath.Glob(filepath.Join(planningDir, "*.bak")); err == nil {
		for _, f := range bakFiles {
			os.Remove(f)
		}
	}
}

func writePhaseResearchArtifacts(root, dir string, survey codexSurveyContext, report codexScoutReport, phases []colony.Phase, snapshots map[string]codexArtifactSnapshot, dispatches []codexPlanningDispatch) ([]string, int, error) {
	written := make([]string, 0, len(phases))
	claimed := claimedPlanningFiles(dispatches)
	preserved := 0
	for _, phase := range phases {
		name := fmt.Sprintf("phase-%d-research.md", phase.ID)
		path := filepath.Join(dir, name)
		relPath := filepath.ToSlash(filepath.Join(".aether", "data", "phase-research", name))
		if shouldPreserveWorkerArtifact(root, relPath, snapshots, claimed) {
			written = append(written, name)
			preserved++
			continue
		}
		var b strings.Builder
		b.WriteString(fmt.Sprintf("# Phase %d Research: %s\n\n", phase.ID, phase.Name))
		b.WriteString(fmt.Sprintf("- Generated: %s\n", time.Now().UTC().Format(time.RFC3339)))
		b.WriteString(fmt.Sprintf("- Phase: %d - %s\n\n", phase.ID, phase.Name))
		b.WriteString("## Goal Alignment\n")
		b.WriteString(strings.TrimSpace(phase.Description))
		b.WriteString("\n\n## Key Patterns\n")
		patterns := []string{}
		for _, finding := range report.Findings {
			patterns = append(patterns, fmt.Sprintf("%s: %s", finding.Area, finding.Discovery))
			if len(patterns) == 3 {
				break
			}
		}
		b.WriteString(bulletList(patterns, "No extra repo patterns were synthesized for this phase."))
		b.WriteString("\n\n## Risks\n")
		risks := append([]string{}, report.Gaps...)
		risks = append(risks, survey.Issues...)
		b.WriteString(bulletList(limitStrings(uniqueSortedStrings(risks), 4), "No additional risks captured for this phase."))
		b.WriteString("\n\n## Files to Study\n")
		files := append([]string{}, report.StudyFiles...)
		for _, task := range phase.Tasks {
			files = append(files, task.Hints...)
		}
		b.WriteString(bulletList(limitStrings(uniqueSortedStrings(files), 6), "No specific file anchors were identified."))
		b.WriteString("\n")
		if err := os.WriteFile(path, []byte(b.String()), 0644); err != nil {
			return nil, 0, fmt.Errorf("failed to write %s: %w", name, err)
		}
		written = append(written, name)
	}
	sort.Strings(written)
	return written, preserved, nil
}
