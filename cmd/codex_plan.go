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
	Goal               string                  `json:"goal"`
	Root               string                  `json:"root"`
	GeneratedAt        string                  `json:"generated_at"`
	Refresh            bool                    `json:"refresh"`
	ExistingPlan       bool                    `json:"existing_plan"`
	ExistingPhaseCount int                     `json:"existing_phase_count,omitempty"`
	Depth              string                  `json:"depth"`
	Granularity        string                  `json:"granularity"`
	GranularityMin     int                     `json:"granularity_min"`
	GranularityMax     int                     `json:"granularity_max"`
	PlanningDepth      string                  `json:"planning_depth"`
	VerificationDepth  string                  `json:"verification_depth,omitempty"`
	Survey             codexSurveyContext      `json:"survey"`
	Dispatches         []codexPlanningDispatch `json:"dispatches"`
	DispatchMode       string                  `json:"dispatch_mode"`
	DispatchContract   map[string]interface{}  `json:"dispatch_contract"`
	FinalizeSurface    string                  `json:"finalize_surface"`
	RequiresFinalizer  bool                    `json:"requires_finalizer"`
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
	// Persist resolved verification depth in ColonyState for downstream build consumption.
	state.VerificationDepth = verificationDepth
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		return nil, fmt.Errorf("failed to persist verification depth: %w", err)
	}
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
		nextPhase := firstBuildablePhase(state.Plan.Phases)
		nextCommand := "aether build 1"
		if nextPhase > 0 {
			nextCommand = fmt.Sprintf("aether build %d", nextPhase)
		}
		updateSessionSummary("plan", nextCommand, fmt.Sprintf("Loaded existing plan (%d phases)", len(state.Plan.Phases)))
		return map[string]interface{}{
			"planned":                    true,
			"existing_plan":              true,
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

	dispatches := plannedPlanningWorkers(root)
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
			planningWarning = fmt.Sprintf("Real planning workers were unavailable, so the saved plan is a generic survey-derived skeleton — not a goal-specific plan. Re-run plan once worker agents are available to get a real plan. Cause: %s", dispatchAvailabilityMessage(invoker))
		} else {
			realDispatches, dispatchErr := dispatchRealPlanningWorkersWithTimeout(context.Background(), root, survey, invoker, opts.WorkerTimeout)
			if realDispatches != nil {
				dispatches = realDispatches
			}
			if dispatchErr != nil {
				if _, ok := invoker.(*codex.FakeInvoker); ok {
					dispatchMode = "simulated"
				} else {
					dispatchMode = "fallback"
					planningWarning = fmt.Sprintf("Real planning workers did not finish cleanly, so the saved plan is a generic survey-derived skeleton — not a goal-specific plan. Re-run plan once worker agents are available to get a real plan. Cause: %s", dispatchErr.Error())
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

	scoutReport := synthesizeScoutPlanningReport(*state.Goal, survey)
	scoutFile, preservedScoutArtifact, err := writePlanningScoutArtifact(root, planningDir, *state.Goal, granularity, survey, dispatches[0], scoutReport, artifactSnapshots)
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
	routeSetterFile, preservedRouteArtifact, err := writeRouteSetterArtifact(root, planningDir, *state.Goal, granularity, survey, dispatches[1], confidence, unresolvedGaps, phases, artifactSnapshots)
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
		dispatchMaps = append(dispatchMaps, entry)
	}

	result := map[string]interface{}{
		"planned":                    true,
		"existing_plan":              false,
		"refreshed":                  opts.Refresh,
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
		"dispatch_contract":          planningDispatchContractWithTimeout(opts.WorkerTimeout),
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
	// Persist resolved verification depth in ColonyState for downstream build consumption.
	state.VerificationDepth = verificationDepth
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		return nil, fmt.Errorf("failed to persist verification depth: %w", err)
	}
	if len(state.Plan.Phases) > 0 && !opts.Refresh {
		nextPhase := firstBuildablePhase(state.Plan.Phases)
		nextCommand := "aether build 1"
		if nextPhase > 0 {
			nextCommand = fmt.Sprintf("aether build %d", nextPhase)
		}
		return map[string]interface{}{
			"plan_only":                  true,
			"planned":                    true,
			"existing_plan":              true,
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
		}, nil
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
	dispatches := plannedPlanningWorkers(root)
	for i := range dispatches {
		dispatches[i].Status = "planned"
		dispatches[i].Brief = renderPlanningWorkerBrief(root, survey, planningWorkerSpecs[i])
	}
	generatedAt := time.Now().UTC()
	manifest := codexPlanManifest{
		Goal:               *state.Goal,
		Root:               root,
		GeneratedAt:        generatedAt.Format(time.RFC3339),
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
		DispatchMode:       "plan-only",
		DispatchContract:   planningDispatchContractWithTimeout(opts.WorkerTimeout),
		FinalizeSurface:    "pending",
		RequiresFinalizer:  true,
	}

	return map[string]interface{}{
		"plan_only":                  true,
		"planned":                    true,
		"existing_plan":              false,
		"refreshed":                  opts.Refresh,
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
		"dispatch_contract":          planningDispatchContractWithTimeout(opts.WorkerTimeout),
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
	}, nil
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
	dispatches := []codexPlanningDispatch{
		{
			Stage:     "scouting",
			Wave:      1,
			Caste:     "scout",
			AgentName: codexAgentNameForCaste("scout"),
			Name:      deterministicAntName("scout", root+"|plan|scout"),
			Task:      "Survey the repo and distill planning findings from available territory reports",
			TaskID:    "plan-scout",
			Outputs:   []string{"SCOUT.md"},
			Status:    "spawned",
		},
		{
			Stage:     "routing",
			Wave:      2,
			Caste:     "route_setter",
			AgentName: codexAgentNameForCaste("route_setter"),
			Name:      deterministicAntName("route_setter", root+"|plan|route-setter"),
			Task:      "Convert surveyed findings into an executable multi-phase colony plan",
			TaskID:    "plan-route-setter",
			Outputs:   []string{"ROUTE-SETTER.md"},
			Status:    "spawned",
		},
	}
	attachPlanningDispatchSkillAssignments(dispatches)
	return dispatches
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

// dispatchRealPlanningWorkers attempts real worker invocation for planning.
// If the invoker is not available, it returns nil, nil (caller falls back to plannedPlanningWorkers).
func dispatchRealPlanningWorkers(ctx context.Context, root string, invoker codex.WorkerInvoker) ([]codexPlanningDispatch, error) {
	return dispatchRealPlanningWorkersWithTimeout(ctx, root, codexSurveyContext{}, invoker, 0)
}

func dispatchRealPlanningWorkersWithTimeout(ctx context.Context, root string, survey codexSurveyContext, invoker codex.WorkerInvoker, timeoutOverride time.Duration) ([]codexPlanningDispatch, error) {
	if invoker == nil || !invoker.IsAvailable(ctx) {
		return nil, nil
	}
	planned := plannedPlanningWorkers(root)
	capsule := resolveCodexWorkerContext()
	pheromoneSection := resolvePheromoneSection()
	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	results := make([]codex.DispatchResult, 0, len(planningWorkerSpecs))
	workerTimeout := effectivePlanningDispatchTimeout(timeoutOverride)
	for i, spec := range planningWorkerSpecs {
		agentName := strings.TrimSuffix(spec.AgentFile, ".toml")
		dispatch := codex.WorkerDispatch{
			ID:               fmt.Sprintf("planning-%d", i),
			WorkerName:       planned[i].Name,
			AgentName:        agentName,
			AgentTOMLPath:    dispatchAgentPath(root, invoker, agentName),
			Caste:            spec.Caste,
			TaskID:           fmt.Sprintf("plan-%d", i),
			TaskBrief:        renderPlanningWorkerBrief(root, survey, spec),
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
		if stageResults[0].Status != "completed" {
			dispatches := convertPlanningDispatchResults(results, root)
			if i+1 < len(planningWorkerSpecs) {
				dispatches[i+1].Status = "dependency_blocked"
				dispatches[i+1].Summary = fmt.Sprintf("%s did not complete, so downstream planning stayed blocked.", dispatch.WorkerName)
			}
			return dispatches, fmt.Errorf("planning worker %s did not complete: %s", dispatch.WorkerName, stageResults[0].Status)
		}
	}

	return convertPlanningDispatchResults(results, root), nil
}

// convertPlanningDispatchResults maps a slice of DispatchResult to codexPlanningDispatch.
// If results don't cover all specs, remaining specs get the planned defaults.
func convertPlanningDispatchResults(results []codex.DispatchResult, root string) []codexPlanningDispatch {
	planned := plannedPlanningWorkers(root)
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
			}
			if strings.TrimSpace(d.Summary) == "" && r.Error != nil {
				d.Summary = strings.TrimSpace(r.Error.Error())
			}
		}

		dispatches = append(dispatches, d)
	}

	return dispatches
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

func renderPlanningWorkerBrief(root string, survey codexSurveyContext, spec planningWorkerSpec) string {
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
	if spec.Caste == "scout" {
		b.WriteString("- Scout scope rule: stay read-only, do not spawn subagents, and stop after the survey docs plus targeted confirmation reads are enough to summarize planning risks.\n")
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
		b.WriteString("- Aether will persist the scout artifact after the worker completes: ")
		b.WriteString(strings.Join(primaryOutputs, ", "))
		b.WriteString("\n")
	} else {
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

	phases := make([]colony.Phase, 0, len(templates))
	for i, template := range templates {
		phase := colony.Phase{
			ID:              i + 1,
			Name:            template.Name,
			Description:     template.Description,
			Status:          colony.PhasePending,
			Tasks:           []colony.Task{},
			SuccessCriteria: append([]string{}, template.SuccessCriteria...),
		}
		if i == 0 {
			phase.Status = colony.PhaseReady
		}
		for j, taskTemplate := range template.Tasks {
			taskID := fmt.Sprintf("%d.%d", i+1, j+1)
			phase.Tasks = append(phase.Tasks, colony.Task{
				ID:              &taskID,
				Goal:            taskTemplate.Goal,
				Status:          colony.TaskPending,
				Constraints:     uniqueSortedStrings(taskTemplate.Constraints),
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
	// planningTemplates returns a generic, survey-driven skeleton. Real goal-awareness
	// belongs in LLM-driven planning workers — substring matching against the user's
	// goal historically leaked Aether-internal phase names and file paths into other
	// repos' plans, so we deliberately do not switch on goal keywords here.
	_ = goal
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

func commonConstraints(survey codexSurveyContext) []string {
	constraints := []string{
		"Follow the repository's existing structure and conventions",
		"Keep changes scoped to the current goal",
	}
	if len(survey.Issues) > 0 {
		constraints = append(constraints, survey.Issues[0])
	}
	return limitStrings(uniqueSortedStrings(constraints), 4)
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
	// Sweep stale .bak files in the planning dir. These are leftovers from older
	// plan runs (e.g. phase-plan.json.bak) and have leaked Aether-internal phase
	// content into other repos in the past. They are never authoritative.
	if planningEntries, err := os.ReadDir(planningDir); err == nil {
		for _, entry := range planningEntries {
			if entry.IsDir() {
				continue
			}
			if strings.HasSuffix(entry.Name(), ".bak") {
				os.Remove(filepath.Join(planningDir, entry.Name()))
			}
		}
	}
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
