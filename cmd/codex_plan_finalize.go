package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

type codexExternalPlanCompletion struct {
	PlanManifest     *codexPlanManifest       `json:"plan_manifest,omitempty"`
	PlanningManifest *codexPlanManifest       `json:"planning_manifest,omitempty"`
	Manifest         *codexPlanManifest       `json:"manifest,omitempty"`
	Dispatches       []codexPlanningDispatch  `json:"dispatches,omitempty"`
	Results          []codexPlanningDispatch  `json:"results,omitempty"`
	Workers          []codexPlanningDispatch  `json:"workers,omitempty"`
	ScoutReport      *codexScoutReport        `json:"scout_report,omitempty"`
	PhasePlan        *codexWorkerPlanArtifact `json:"phase_plan,omitempty"`
}

var planFinalizeCmd = &cobra.Command{
	Use:   "plan-finalize",
	Short: "Record externally spawned wrapper planning workers as the colony plan",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		completionPath, _ := cmd.Flags().GetString("completion-file")
		completion, err := loadExternalPlanCompletion(completionPath)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		result, err := runCodexPlanFinalize(skillWorkspaceRoot(), completion)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		outputWorkflow(result, renderPlanVisual(result))
		return nil
	},
}

func loadExternalPlanCompletion(path string) (codexExternalPlanCompletion, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return codexExternalPlanCompletion{}, fmt.Errorf("flag --completion-file is required")
	}
	var data []byte
	var err error
	if path == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return codexExternalPlanCompletion{}, fmt.Errorf("read completion file: %w", err)
	}

	var completion codexExternalPlanCompletion
	if err := json.Unmarshal(data, &completion); err != nil {
		return codexExternalPlanCompletion{}, fmt.Errorf("parse completion file: %w", err)
	}
	if completion.activeManifest() != nil {
		return completion, nil
	}

	var envelope struct {
		Result codexExternalPlanCompletion `json:"result"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return codexExternalPlanCompletion{}, fmt.Errorf("parse completion envelope: %w", err)
	}
	if envelope.Result.activeManifest() == nil {
		return codexExternalPlanCompletion{}, fmt.Errorf("completion file must include plan_manifest")
	}
	return envelope.Result, nil
}

func (c codexExternalPlanCompletion) activeManifest() *codexPlanManifest {
	if c.PlanManifest != nil {
		return c.PlanManifest
	}
	if c.PlanningManifest != nil {
		return c.PlanningManifest
	}
	return c.Manifest
}

func (c codexExternalPlanCompletion) workerResults() []codexPlanningDispatch {
	results := make([]codexPlanningDispatch, 0, len(c.Dispatches)+len(c.Results)+len(c.Workers))
	results = append(results, c.Dispatches...)
	results = append(results, c.Results...)
	results = append(results, c.Workers...)
	return results
}

func runCodexPlanFinalize(root string, completion codexExternalPlanCompletion) (map[string]interface{}, error) {
	if store == nil {
		return nil, fmt.Errorf("no store initialized")
	}
	manifest := completion.activeManifest()
	if manifest == nil {
		return nil, fmt.Errorf("completion file must include plan_manifest")
	}
	if manifest.DispatchMode != "plan-only" || !manifest.RequiresFinalizer {
		return nil, fmt.Errorf("plan_manifest must come from `aether plan --plan-only`")
	}
	if len(manifest.Dispatches) == 0 {
		return nil, fmt.Errorf("plan_manifest contains no dispatches")
	}

	state, granularity, err := validateExternalPlanState(manifest)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	runHandle, err := beginRuntimeSpawnRun("plan", now)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize planning run: %w", err)
	}
	runStatus := "failed"
	defer func() {
		finishRuntimeSpawnRun(runHandle, runStatus, time.Now().UTC())
	}()

	dispatches, err := mergeExternalPlanResults(*manifest, completion.workerResults())
	if err != nil {
		return nil, err
	}
	scoutReport := completion.scoutReport(dispatches, manifest)
	phasePlan, err := completion.phasePlan(root, dispatches)
	if err != nil {
		return nil, err
	}
	if len(phasePlan.Phases) == 0 {
		return nil, fmt.Errorf("phase_plan contains no phases")
	}

	phases := buildWorkerPlanPhases(*phasePlan)
	if len(phases) == 0 {
		return nil, fmt.Errorf("phase_plan produced no buildable phases")
	}
	_, baseConfidence, baseGaps := synthesizeRouteSetterPlan(manifest.Goal, granularity, manifest.Survey, scoutReport)
	confidence := mergePlanConfidence(baseConfidence, phasePlan.Confidence)
	unresolvedGaps := limitStrings(uniqueSortedStrings(append(baseGaps, phasePlan.Gaps...)), 4)

	planningDir := filepath.Join(store.BasePath(), "planning")
	phaseResearchDir := filepath.Join(store.BasePath(), "phase-research")
	if err := os.MkdirAll(planningDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create planning directory: %w", err)
	}
	if err := os.RemoveAll(phaseResearchDir); err != nil {
		return nil, fmt.Errorf("failed to clear phase research directory: %w", err)
	}
	if err := os.MkdirAll(phaseResearchDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create phase research directory: %w", err)
	}
	for _, name := range []string{"SCOUT.md", "ROUTE-SETTER.md", "phase-plan.json", ".fallback-marker"} {
		_ = os.Remove(filepath.Join(planningDir, name))
	}

	emptySnapshots := map[string]codexArtifactSnapshot{}
	scoutFile, _, err := writePlanningScoutArtifact(root, planningDir, manifest.Goal, granularity, manifest.Survey, dispatches[0], scoutReport, emptySnapshots)
	if err != nil {
		return nil, err
	}
	routeSetterFile, _, err := writeRouteSetterArtifact(root, planningDir, manifest.Goal, granularity, manifest.Survey, dispatches[1], confidence, unresolvedGaps, phases, emptySnapshots)
	if err != nil {
		return nil, err
	}
	planArtifactFile, _, err := writeWorkerPlanArtifact(root, planningDir, confidence, unresolvedGaps, phases, emptySnapshots, nil)
	if err != nil {
		return nil, err
	}
	phaseResearchFiles, _, err := writePhaseResearchArtifacts(root, phaseResearchDir, manifest.Survey, scoutReport, phases, emptySnapshots, nil)
	if err != nil {
		return nil, err
	}

	if err := recordExternalPlanSpawnTree(dispatches); err != nil {
		return nil, err
	}

	updatedState := state
	updatedState.State = colony.StateREADY
	updatedState.CurrentPhase = firstBuildablePhase(phases)
	updatedState.BuildStartedAt = nil
	updatedState.PlanGranularity = granularity
	planConfidence := float64(confidence.Overall) / 100.0
	updatedState.Plan = colony.Plan{
		GeneratedAt: &now,
		Confidence:  &planConfidence,
		Phases:      phases,
	}
	updatedState.Events = append(trimmedEvents(updatedState.Events),
		fmt.Sprintf("%s|planning_scout|plan-finalize|External Scout summarized surveyed repo context", now.Format(time.RFC3339)),
		fmt.Sprintf("%s|plan_generated|plan-finalize|Generated %d phases with %d%% confidence from external planning workers", now.Format(time.RFC3339), len(phases), confidence.Overall),
	)
	if err := store.SaveJSON("COLONY_STATE.json", updatedState); err != nil {
		return nil, fmt.Errorf("failed to save colony state: %w", err)
	}
	emitPlanCeremonyDispatchSequence("aether-plan-finalize", dispatches)

	nextPhase := firstBuildablePhase(phases)
	nextCommand := "aether build 1"
	if nextPhase > 0 {
		nextCommand = fmt.Sprintf("aether build %d", nextPhase)
	}
	updateSessionSummary("plan-finalize", nextCommand, fmt.Sprintf("Generated %d plan phases with %d%% confidence from external planning workers", len(phases), confidence.Overall))
	runStatus = "completed"

	return map[string]interface{}{
		"planned":                   true,
		"existing_plan":             false,
		"refreshed":                 manifest.Refresh,
		"goal":                      manifest.Goal,
		"phases":                    phases,
		"count":                     len(phases),
		"depth":                     manifest.Depth,
		"granularity":               string(granularity),
		"granularity_min":           granularityMin(granularity),
		"granularity_max":           granularityMax(granularity),
		"confidence":                confidence,
		"planning_dir":              planningDir,
		"planning_files":            []string{filepath.Base(scoutFile), filepath.Base(routeSetterFile)},
		"plan_artifact":             filepath.Base(planArtifactFile),
		"phase_research_dir":        phaseResearchDir,
		"phase_research_files":      phaseResearchFiles,
		"dispatches":                planningDispatchMaps(dispatches),
		"dispatch_mode":             "external-task",
		"dispatch_contract":         manifest.DispatchContract,
		"artifact_source":           "external-task",
		"plan_source":               "external-task",
		"gaps":                      unresolvedGaps,
		"survey_docs":               manifest.Survey.SurveyDocs,
		"unresolved_clarifications": 0,
		"planning_warning":          "",
		"next":                      nextCommand,
	}, nil
}

func validateExternalPlanState(manifest *codexPlanManifest) (colony.ColonyState, colony.PlanGranularity, error) {
	state, err := loadActiveColonyState()
	if err != nil {
		return state, "", fmt.Errorf("%s", colonyStateLoadMessage(err))
	}
	if state.Goal == nil || strings.TrimSpace(*state.Goal) == "" {
		return state, "", fmt.Errorf("No active colony goal. Run `aether init \"goal\"` first.")
	}
	if strings.TrimSpace(*state.Goal) != strings.TrimSpace(manifest.Goal) {
		return state, "", fmt.Errorf("plan_manifest goal does not match active colony goal")
	}
	granularity := colony.PlanGranularity(strings.TrimSpace(manifest.Granularity))
	if !granularity.Valid() {
		return state, "", fmt.Errorf("plan_manifest granularity %q is invalid", manifest.Granularity)
	}
	if len(state.Plan.Phases) > 0 && !manifest.Refresh {
		return state, granularity, fmt.Errorf("active colony already has a plan; rerun `aether plan --plan-only --refresh` before finalizing a replacement")
	}
	if manifest.Refresh && state.CurrentPhase > 0 {
		for _, phase := range state.Plan.Phases {
			if phase.Status == colony.PhaseCompleted {
				return state, granularity, fmt.Errorf("cannot force-replan after completed phases; archive this colony and start a new one")
			}
		}
	}
	return state, granularity, nil
}

func mergeExternalPlanResults(manifest codexPlanManifest, results []codexPlanningDispatch) ([]codexPlanningDispatch, error) {
	resultByName := make(map[string]codexPlanningDispatch, len(results))
	for _, result := range results {
		name := strings.TrimSpace(result.Name)
		if name == "" {
			return nil, fmt.Errorf("external planning result missing name")
		}
		if _, exists := resultByName[name]; exists {
			return nil, fmt.Errorf("duplicate external planning result for %s", name)
		}
		resultByName[name] = result
	}

	dispatches := make([]codexPlanningDispatch, len(manifest.Dispatches))
	for i, dispatch := range manifest.Dispatches {
		result, ok := resultByName[dispatch.Name]
		if !ok {
			return nil, fmt.Errorf("missing external planning result for %s", dispatch.Name)
		}
		if err := validateExternalPlanIdentity(dispatch, result); err != nil {
			return nil, err
		}
		status := normalizeExternalBuildStatus(result.Status)
		if !isTerminalExternalBuildStatus(status) {
			return nil, fmt.Errorf("external planning result for %s has non-terminal status %q", dispatch.Name, result.Status)
		}
		if status != "completed" && status != "manually-reconciled" {
			return nil, fmt.Errorf("external planning result for %s did not complete cleanly: %s", dispatch.Name, status)
		}
		dispatch.Status = status
		dispatch.Summary = strings.TrimSpace(result.Summary)
		if dispatch.Summary == "" && len(result.Blockers) > 0 {
			dispatch.Summary = strings.Join(result.Blockers, "; ")
		}
		dispatch.Duration = result.Duration
		dispatch.FilesCreated = uniqueSortedStrings(result.FilesCreated)
		dispatch.FilesModified = uniqueSortedStrings(result.FilesModified)
		dispatch.Claimed = uniqueSortedStrings(append(append([]string{}, result.FilesCreated...), result.FilesModified...))
		dispatch.ScoutReport = result.ScoutReport
		dispatch.PhasePlan = result.PhasePlan
		dispatches[i] = dispatch
	}
	return dispatches, nil
}

func validateExternalPlanIdentity(dispatch codexPlanningDispatch, result codexPlanningDispatch) error {
	if value := strings.TrimSpace(result.Caste); value != "" && !strings.EqualFold(value, dispatch.Caste) {
		return fmt.Errorf("external planning result %s caste = %q, want %q", dispatch.Name, value, dispatch.Caste)
	}
	if value := strings.TrimSpace(result.Stage); value != "" && !strings.EqualFold(value, dispatch.Stage) {
		return fmt.Errorf("external planning result %s stage = %q, want %q", dispatch.Name, value, dispatch.Stage)
	}
	if value := strings.TrimSpace(result.TaskID); value != "" && value != strings.TrimSpace(dispatch.TaskID) {
		return fmt.Errorf("external planning result %s task_id = %q, want %q", dispatch.Name, value, dispatch.TaskID)
	}
	if result.Wave > 0 && dispatch.Wave > 0 && result.Wave != dispatch.Wave {
		return fmt.Errorf("external planning result %s wave = %d, want %d", dispatch.Name, result.Wave, dispatch.Wave)
	}
	return nil
}

func (c codexExternalPlanCompletion) scoutReport(dispatches []codexPlanningDispatch, manifest *codexPlanManifest) codexScoutReport {
	if c.ScoutReport != nil {
		return *c.ScoutReport
	}
	for _, dispatch := range dispatches {
		if dispatch.ScoutReport != nil {
			return *dispatch.ScoutReport
		}
	}
	report := synthesizeScoutPlanningReport(manifest.Goal, manifest.Survey)
	for _, dispatch := range dispatches {
		if dispatch.Caste == "scout" && strings.TrimSpace(dispatch.Summary) != "" {
			report.Findings = append([]codexScoutFinding{{
				Area:      "External Scout",
				Discovery: dispatch.Summary,
				Source:    dispatch.Name,
			}}, report.Findings...)
			if len(report.Findings) > 5 {
				report.Findings = report.Findings[:5]
			}
			break
		}
	}
	return report
}

func (c codexExternalPlanCompletion) phasePlan(root string, dispatches []codexPlanningDispatch) (*codexWorkerPlanArtifact, error) {
	if c.PhasePlan != nil {
		return c.PhasePlan, nil
	}
	for _, dispatch := range dispatches {
		if dispatch.PhasePlan != nil {
			return dispatch.PhasePlan, nil
		}
	}
	for _, dispatch := range dispatches {
		if !strings.EqualFold(dispatch.Caste, "route_setter") {
			continue
		}
		for _, relPath := range dispatch.Claimed {
			if filepath.ToSlash(filepath.Clean(relPath)) != filepath.ToSlash(filepath.Join(".aether", "data", "planning", "phase-plan.json")) {
				continue
			}
			data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relPath)))
			if err != nil {
				return nil, fmt.Errorf("read claimed phase_plan: %w", err)
			}
			var artifact codexWorkerPlanArtifact
			if err := json.Unmarshal(data, &artifact); err != nil {
				return nil, fmt.Errorf("parse claimed phase_plan: %w", err)
			}
			return &artifact, nil
		}
	}
	return nil, fmt.Errorf("completion file must include route-setter phase_plan")
}

func planningDispatchMaps(dispatches []codexPlanningDispatch) []map[string]interface{} {
	maps := make([]map[string]interface{}, 0, len(dispatches))
	for _, dispatch := range dispatches {
		entry := map[string]interface{}{
			"stage":      dispatch.Stage,
			"wave":       dispatch.Wave,
			"caste":      dispatch.Caste,
			"agent_name": dispatch.AgentName,
			"name":       dispatch.Name,
			"task":       dispatch.Task,
			"task_id":    dispatch.TaskID,
			"outputs":    dispatch.Outputs,
			"status":     dispatch.Status,
		}
		if summary := strings.TrimSpace(dispatch.Summary); summary != "" {
			entry["summary"] = summary
		}
		if dispatch.Duration > 0 {
			entry["duration"] = dispatch.Duration
		}
		if dispatch.SkillCount > 0 {
			entry["skill_count"] = dispatch.SkillCount
			entry["colony_skill_count"] = dispatch.ColonySkills
			entry["domain_skill_count"] = dispatch.DomainSkills
			entry["matched_skills"] = append([]string{}, dispatch.MatchedSkills...)
			entry["skill_section"] = dispatch.SkillSection
		}
		maps = append(maps, entry)
	}
	return maps
}

func recordExternalPlanSpawnTree(dispatches []codexPlanningDispatch) error {
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
				return fmt.Errorf("failed to record external planning dispatch %s: %w", dispatch.Name, err)
			}
			known[dispatch.Name] = struct{}{}
		}
		if err := spawnTree.UpdateStatus(dispatch.Name, dispatch.Status, dispatch.Summary); err != nil {
			return fmt.Errorf("failed to complete external planning dispatch %s: %w", dispatch.Name, err)
		}
	}
	return nil
}
