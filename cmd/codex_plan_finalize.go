package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
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
	Synthesis        *codexPlanSynthesis      `json:"synthesis,omitempty"`
}

type codexPlanSynthesis struct {
	Source    string                   `json:"source,omitempty"`
	Reason    string                   `json:"reason,omitempty"`
	PhasePlan *codexWorkerPlanArtifact `json:"phase_plan,omitempty"`
}

type codexPlanProvenance struct {
	Dispatches      []codexPlanningDispatch
	ScoutReport     codexScoutReport
	PhasePlan       *codexWorkerPlanArtifact
	DispatchMode    string
	ArtifactSource  string
	PlanSource      string
	SourceSummary   string
	PlanningWarning string
	RecordWorkers   bool
}

const (
	planFinalizeManifestMaxAge     = 24 * time.Hour
	planFinalizeManifestFutureSkew = 5 * time.Minute
	planFinalizeFailureSource      = "plan-finalize"
)

var planFinalizeCmd = &cobra.Command{
	Use:   "plan-finalize",
	Short: "Record externally spawned wrapper planning workers as the colony plan",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		completionPath, _ := cmd.Flags().GetString("completion-file")
		completion, err := loadExternalPlanCompletion(completionPath)
		if err != nil {
			outputError(1, err.Error(), nil)
			return renderedErrorExit(1)
		}
		result, err := runCodexPlanFinalize(skillWorkspaceRoot(), completion)
		if err != nil {
			recordPlanFinalizeFailure(err)
			outputError(1, err.Error(), nil)
			return renderedErrorExit(1)
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
	if err := validateFinalizerCompletionFilePath(path); err != nil {
		return codexExternalPlanCompletion{}, err
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

func recordPlanFinalizeFailure(cause error) {
	if store == nil || cause == nil {
		return
	}
	flags := loadPlanFinalizeFlagsFile()
	description := "Planning finalization failed: " + compactPlanFinalizeError(cause.Error())
	now := time.Now().UTC().Format(time.RFC3339)
	for i := range flags.Decisions {
		flag := &flags.Decisions[i]
		if flag.Source == planFinalizeFailureSource && !flag.Resolved {
			flag.Type = "blocker"
			flag.Description = description
			if flag.CreatedAt == "" {
				flag.CreatedAt = now
			}
			_ = store.SaveJSON("pending-decisions.json", flags)
			updateSessionSummary(planFinalizeFailureSource, "aether flags --status active", description)
			return
		}
	}
	flags.Decisions = append(flags.Decisions, colony.FlagEntry{
		ID:          generateFlagID(),
		Type:        "blocker",
		Description: description,
		Source:      planFinalizeFailureSource,
		CreatedAt:   now,
		Resolved:    false,
	})
	_ = store.SaveJSON("pending-decisions.json", flags)
	updateSessionSummary(planFinalizeFailureSource, "aether flags --status active", description)
}

func resolvePlanFinalizeFailureFlags() {
	if store == nil {
		return
	}
	flags := loadPlanFinalizeFlagsFile()
	changed := false
	now := time.Now().UTC().Format(time.RFC3339)
	for i := range flags.Decisions {
		flag := &flags.Decisions[i]
		if flag.Source != planFinalizeFailureSource || flag.Resolved {
			continue
		}
		flag.Resolved = true
		flag.ResolvedAt = now
		flag.Resolution = "plan-finalize succeeded"
		changed = true
	}
	if changed {
		_ = store.SaveJSON("pending-decisions.json", flags)
	}
}

func loadPlanFinalizeFlagsFile() colony.FlagsFile {
	flags := colony.FlagsFile{Version: "1.0", Decisions: []colony.FlagEntry{}}
	if store == nil {
		return flags
	}
	if err := store.LoadJSON("pending-decisions.json", &flags); err == nil {
		if flags.Decisions == nil {
			flags.Decisions = []colony.FlagEntry{}
		}
		return flags
	}
	if err := store.LoadJSON("flags.json", &flags); err == nil {
		if flags.Decisions == nil {
			flags.Decisions = []colony.FlagEntry{}
		}
		return flags
	}
	return flags
}

func activePlanFinalizeFailureFlag(s *storage.Store) (colony.FlagEntry, bool) {
	if s == nil {
		return colony.FlagEntry{}, false
	}
	flags, ok := loadFlagsFile(s)
	if !ok {
		return colony.FlagEntry{}, false
	}
	for _, flag := range flags.Decisions {
		if flag.Source == planFinalizeFailureSource && !flag.Resolved {
			return flag, true
		}
	}
	return colony.FlagEntry{}, false
}

func compactPlanFinalizeError(text string) string {
	text = strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
	if text == "" {
		return "unknown finalizer error"
	}
	const limit = 320
	if len(text) <= limit {
		return text
	}
	return text[:limit-3] + "..."
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
	if (manifest.DispatchMode != "plan-only" && manifest.DispatchMode != "agent-delegate") || !manifest.RequiresFinalizer {
		return nil, fmt.Errorf("plan_manifest must come from `aether plan --plan-only` or an agent-delegate planning response")
	}
	if len(manifest.Dispatches) == 0 {
		return nil, fmt.Errorf("plan_manifest contains no dispatches")
	}
	if err := validateFinalizerManifestRoot("plan_manifest", manifest.Root, root); err != nil {
		return nil, err
	}

	state, granularity, err := validateExternalPlanState(manifest)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	if err := validateCodexPlanManifestFreshness(*manifest, now); err != nil {
		return nil, err
	}
	if err := validateCodexPlanManifestWorkspace(root, *manifest); err != nil {
		return nil, err
	}

	provenance, err := completion.planProvenance(root, manifest)
	if err != nil {
		return nil, err
	}
	dispatches := provenance.Dispatches
	scoutReport := provenance.ScoutReport
	phasePlan := provenance.PhasePlan
	if len(phasePlan.Phases) == 0 {
		return nil, fmt.Errorf("phase_plan contains no phases")
	}
	normalizedPhasePlan, dependencyRepairs, err := normalizeWorkerPlanArtifactDependencies(*phasePlan)
	if err != nil {
		return nil, err
	}
	phasePlan = &normalizedPhasePlan

	phases := buildWorkerPlanPhases(*phasePlan)
	if len(phases) == 0 || buildablePlanTaskCount(phases) == 0 {
		return nil, fmt.Errorf("phase_plan contains no buildable tasks")
	}
	if err := colony.DetectCycles(phases); err != nil {
		var cycleErr *colony.CycleError
		if errors.As(err, &cycleErr) {
			return nil, fmt.Errorf("phase_plan contains circular dependency: %s. Remove the cycle and rerun `aether plan-finalize --completion-file <file>`", cycleErr)
		}
		return nil, fmt.Errorf("phase_plan dependency validation failed: %w", err)
	}
	groundingWarnings := checkPlanGrounding(phases, manifest.Survey.SourceAnchors)
	_, baseConfidence, baseGaps := synthesizeRouteSetterPlan(manifest.Goal, granularity, manifest.Survey, scoutReport)
	confidence := mergePlanConfidence(baseConfidence, phasePlan.Confidence)
	unresolvedGaps := limitStrings(uniqueSortedStrings(append(baseGaps, phasePlan.Gaps...)), 4)
	if len(dependencyRepairs) > 0 {
		repairWarning := strings.Join(dependencyRepairs, " ")
		if strings.TrimSpace(provenance.PlanningWarning) != "" {
			provenance.PlanningWarning += "; " + repairWarning
		} else {
			provenance.PlanningWarning = repairWarning
		}
	}
	planningLoop := evaluatePlanningLoop(confidence, unresolvedGaps, codexPlanOptions{
		TargetConfidence: manifest.PlanningLoop.TargetConfidence,
		MaxIterations:    manifest.PlanningLoop.MaxIterations,
		Accept:           manifest.PlanningLoop.Accept,
	}, manifest.Depth)

	runHandle, err := beginRuntimeSpawnRun("plan", now)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize planning run: %w", err)
	}
	runStatus := "failed"
	defer func() {
		finishRuntimeSpawnRun(runHandle, runStatus, time.Now().UTC())
	}()

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
	scoutDispatch, ok := planningDispatchByCaste(dispatches, "scout")
	if !ok {
		return nil, fmt.Errorf("plan_manifest missing scout dispatch")
	}
	routeSetterDispatch, ok := planningDispatchByCaste(dispatches, "route_setter")
	if !ok {
		return nil, fmt.Errorf("plan_manifest missing route-setter dispatch")
	}
	scoutFile, _, err := writePlanningScoutArtifact(root, planningDir, manifest.Goal, granularity, manifest.Survey, scoutDispatch, scoutReport, emptySnapshots)
	if err != nil {
		return nil, err
	}
	routeSetterFile, _, err := writeRouteSetterArtifact(root, planningDir, manifest.Goal, granularity, manifest.Survey, routeSetterDispatch, confidence, unresolvedGaps, phases, planningLoop, emptySnapshots)
	if err != nil {
		return nil, err
	}
	planArtifactFile, _, err := writeWorkerPlanArtifact(root, planningDir, confidence, unresolvedGaps, phases, planningLoop, emptySnapshots, nil)
	if err != nil {
		return nil, err
	}
	phaseResearchFiles, _, err := writePhaseResearchArtifacts(root, phaseResearchDir, manifest.Survey, scoutReport, phases, emptySnapshots, nil)
	if err != nil {
		return nil, err
	}

	if provenance.RecordWorkers {
		if err := recordExternalPlanSpawnTree(dispatches); err != nil {
			return nil, err
		}
	}

	updatedState := state
	updatedState.State = colony.StateREADY
	updatedState.CurrentPhase = firstBuildablePhase(phases)
	updatedState.BuildStartedAt = nil
	updatedState.PlanGranularity = granularity
	if strings.TrimSpace(manifest.VerificationDepth) != "" {
		updatedState.VerificationDepth = string(colony.NormalizeVerificationDepth(manifest.VerificationDepth))
	}
	planConfidence := float64(confidence.Overall) / 100.0
	updatedState.Plan = colony.Plan{
		GeneratedAt: &now,
		Confidence:  &planConfidence,
		Phases:      phases,
	}
	updatedState.Events = append(trimmedEvents(updatedState.Events),
		fmt.Sprintf("%s|planning_scout|plan-finalize|%s", now.Format(time.RFC3339), provenance.SourceSummary),
		fmt.Sprintf("%s|plan_generated|plan-finalize|Generated %d phases with %d%% confidence from %s; planning loop stopped: %s", now.Format(time.RFC3339), len(phases), confidence.Overall, provenance.SourceSummary, planningLoop.StopReason),
	)
	if err := store.SaveJSON("COLONY_STATE.json", updatedState); err != nil {
		return nil, fmt.Errorf("failed to save colony state: %w", err)
	}
	resolvePlanFinalizeFailureFlags()
	if provenance.RecordWorkers {
		emitPlanCeremonyDispatchSequence("aether-plan-finalize", dispatches)
	}

	nextPhase := firstBuildablePhase(phases)
	nextCommand := "aether build 1"
	if nextPhase > 0 {
		nextCommand = fmt.Sprintf("aether build %d", nextPhase)
	}
	updateSessionSummary("plan-finalize", nextCommand, fmt.Sprintf("Generated %d plan phases with %d%% confidence from %s", len(phases), confidence.Overall, provenance.SourceSummary))
	runStatus = "completed"

	result := map[string]interface{}{
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
		"planning_loop":             planningLoop,
		"planning_dir":              planningDir,
		"planning_files":            []string{filepath.Base(scoutFile), filepath.Base(routeSetterFile)},
		"plan_artifact":             filepath.Base(planArtifactFile),
		"phase_research_dir":        phaseResearchDir,
		"phase_research_files":      phaseResearchFiles,
		"dispatches":                planningDispatchMaps(dispatches),
		"dispatch_mode":             provenance.DispatchMode,
		"dispatch_contract":         manifest.DispatchContract,
		"artifact_source":           provenance.ArtifactSource,
		"plan_source":               provenance.PlanSource,
		"gaps":                      unresolvedGaps,
		"survey_docs":               manifest.Survey.SurveyDocs,
		"unresolved_clarifications": 0,
		"planning_warning":          provenance.PlanningWarning,
		"next":                      nextCommand,
	}
	if len(groundingWarnings) > 0 {
		result["grounding_warnings"] = groundingWarnings
		warningTexts := make([]string, len(groundingWarnings))
		for i, w := range groundingWarnings {
			warningTexts[i] = fmt.Sprintf("Phase %d (%s): %d tasks with no file references (%d anchors available)",
				w.PhaseID, w.PhaseName, w.UngroundedTasks, w.AnchorCount)
		}
		groundingMsg := fmt.Sprintf("Grounding gate: %s", strings.Join(warningTexts, "; "))
		if existing := provenance.PlanningWarning; existing != "" {
			result["planning_warning"] = existing + "; " + groundingMsg
		} else {
			result["planning_warning"] = groundingMsg
		}
	}
	addOrchestratorBoundaryGuidance(result, "plan", updatedState, nextCommand, manifest.BoundaryQuestions)
	return result, nil
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
	if err := validateFinalizerManifestColonyMode("plan_manifest", manifest.ColonyMode, state); err != nil {
		return state, "", err
	}
	granularity := colony.PlanGranularity(strings.TrimSpace(manifest.Granularity))
	if !granularity.Valid() {
		return state, "", fmt.Errorf("plan_manifest granularity %q is invalid", manifest.Granularity)
	}
	if len(state.Plan.Phases) > 0 && !manifest.Refresh {
		if manifest.ExistingPlan {
			// Manifest acknowledges existing phases were detected during plan-only.
			// Proceed with finalization; the manifest's phases will replace them.
			return state, granularity, nil
		}
		return state, granularity, fmt.Errorf("stale phases from a prior session are blocking finalization; rerun `aether plan --plan-only --refresh` or run `aether init` to start fresh")
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

func validateCodexPlanManifestFreshness(manifest codexPlanManifest, now time.Time) error {
	raw := strings.TrimSpace(manifest.GeneratedAt)
	if raw == "" {
		return fmt.Errorf("plan_manifest generated_at is required for freshness validation")
	}
	generatedAt, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return fmt.Errorf("plan_manifest generated_at is invalid: %w", err)
	}
	generatedAt = generatedAt.UTC()
	if generatedAt.After(now.Add(planFinalizeManifestFutureSkew)) {
		return fmt.Errorf("plan_manifest generated_at %s is too far in the future", raw)
	}
	if now.Sub(generatedAt) > planFinalizeManifestMaxAge {
		return fmt.Errorf("stale plan_manifest generated_at %s exceeds max age %s; rerun `aether plan --plan-only`", raw, planFinalizeManifestMaxAge)
	}
	return nil
}

func validateCodexPlanManifestWorkspace(root string, manifest codexPlanManifest) error {
	current, err := loadCodexSurveyContext(root)
	if err != nil {
		return fmt.Errorf("failed to validate plan_manifest workspace: %w", err)
	}
	comparisons := []struct {
		name string
		was  []string
		now  []string
	}{
		{name: "survey_docs", was: manifest.Survey.SurveyDocs, now: current.SurveyDocs},
		{name: "languages", was: manifest.Survey.Languages, now: current.Languages},
		{name: "frameworks", was: manifest.Survey.Frameworks, now: current.Frameworks},
		{name: "directories", was: manifest.Survey.Directories, now: current.Directories},
		{name: "entry_points", was: manifest.Survey.EntryPoints, now: current.EntryPoints},
		{name: "dependencies", was: manifest.Survey.Dependencies, now: current.Dependencies},
		{name: "test_files", was: manifest.Survey.TestFiles, now: current.TestFiles},
		{name: "issues", was: manifest.Survey.Issues, now: current.Issues},
		{name: "security_patterns", was: manifest.Survey.SecurityPatterns, now: current.SecurityPatterns},
	}
	for _, comparison := range comparisons {
		if !sameStringSet(comparison.was, comparison.now) {
			return fmt.Errorf("plan_manifest workspace changed since generation: %s changed; rerun `aether plan --plan-only`", comparison.name)
		}
	}
	return nil
}

func mergeExternalPlanResults(manifest codexPlanManifest, results []codexPlanningDispatch) ([]codexPlanningDispatch, error) {
	resultByName := make(map[string]codexPlanningDispatch, len(results))
	for _, result := range results {
		name := strings.TrimSpace(result.Name)
		if name == "" {
			return nil, fmt.Errorf("external planning result missing name")
		}
		if existing, exists := resultByName[name]; exists {
			if useIncoming, ok := preferCompletedResultOverTimeout(existing.Status, result.Status); ok {
				if useIncoming {
					resultByName[name] = result
				}
				continue
			}
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
	dispatchSpec := workerIdentitySpec{
		Caste:  dispatch.Caste,
		Stage:  dispatch.Stage,
		TaskID: dispatch.TaskID,
		Wave:   dispatch.Wave,
	}
	resultSpec := workerIdentitySpec{
		Caste:  result.Caste,
		Stage:  result.Stage,
		TaskID: result.TaskID,
		Wave:   result.Wave,
	}
	return validateWorkerResultIdentity(dispatch.Name, dispatchSpec, resultSpec)
}

func (c codexExternalPlanCompletion) planProvenance(root string, manifest *codexPlanManifest) (codexPlanProvenance, error) {
	if c.Synthesis != nil {
		return c.synthesisPlanProvenance(manifest)
	}
	if c.PhasePlan != nil {
		return codexPlanProvenance{}, fmt.Errorf("top-level phase_plan requires explicit synthesis.source and synthesis.reason")
	}

	dispatches, err := mergeExternalPlanResults(*manifest, c.workerResults())
	if err != nil {
		return codexPlanProvenance{}, err
	}
	phasePlan, err := routeSetterPhasePlan(root, dispatches, manifest)
	if err != nil {
		return codexPlanProvenance{}, err
	}
	return codexPlanProvenance{
		Dispatches:      dispatches,
		ScoutReport:     c.scoutReport(dispatches, manifest),
		PhasePlan:       phasePlan,
		DispatchMode:    "external-task",
		ArtifactSource:  "external-task",
		PlanSource:      "external-task",
		SourceSummary:   "external planning workers",
		PlanningWarning: "",
		RecordWorkers:   true,
	}, nil
}

func (c codexExternalPlanCompletion) synthesisPlanProvenance(manifest *codexPlanManifest) (codexPlanProvenance, error) {
	if c.PhasePlan != nil {
		return codexPlanProvenance{}, fmt.Errorf("top-level phase_plan is ambiguous; put host-created plans under synthesis.phase_plan")
	}
	if len(c.workerResults()) > 0 {
		return codexPlanProvenance{}, fmt.Errorf("synthesis completion must not include planning worker results")
	}
	source := strings.TrimSpace(c.Synthesis.Source)
	if source != "ts-host" {
		return codexPlanProvenance{}, fmt.Errorf("synthesis.source must be %q", "ts-host")
	}
	reason := strings.TrimSpace(c.Synthesis.Reason)
	if reason == "" {
		return codexPlanProvenance{}, fmt.Errorf("synthesis.reason is required")
	}
	if c.Synthesis.PhasePlan == nil {
		return codexPlanProvenance{}, fmt.Errorf("synthesis.phase_plan is required")
	}

	dispatches := synthesizedPlanDispatches(manifest.Dispatches, reason)
	return codexPlanProvenance{
		Dispatches:      dispatches,
		ScoutReport:     synthesizeScoutPlanningReport(manifest.Goal, manifest.Survey),
		PhasePlan:       c.Synthesis.PhasePlan,
		DispatchMode:    "synthesis",
		ArtifactSource:  "ts-host-synthesis",
		PlanSource:      "ts-host-synthesis",
		SourceSummary:   "explicit ts-host synthesis",
		PlanningWarning: fmt.Sprintf("Plan finalized from explicit ts-host synthesis. Reason: %s", reason),
		RecordWorkers:   false,
	}, nil
}

func synthesizedPlanDispatches(dispatches []codexPlanningDispatch, reason string) []codexPlanningDispatch {
	synthesized := make([]codexPlanningDispatch, len(dispatches))
	for i, dispatch := range dispatches {
		dispatch.Status = "skipped"
		dispatch.Summary = fmt.Sprintf("Skipped external planning worker; explicit ts-host synthesis was used. Reason: %s", reason)
		dispatch.Duration = 0
		dispatch.FilesCreated = nil
		dispatch.FilesModified = nil
		dispatch.Claimed = nil
		dispatch.ScoutReport = nil
		dispatch.PhasePlan = nil
		synthesized[i] = dispatch
	}
	return synthesized
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

func routeSetterPhasePlan(root string, dispatches []codexPlanningDispatch, manifest *codexPlanManifest) (*codexWorkerPlanArtifact, error) {
	for _, dispatch := range dispatches {
		if !strings.EqualFold(dispatch.Caste, "route_setter") {
			continue
		}
		if dispatch.PhasePlan != nil {
			return dispatch.PhasePlan, nil
		}
		for _, relPath := range dispatch.Claimed {
			if filepath.ToSlash(filepath.Clean(relPath)) != filepath.ToSlash(filepath.Join(".aether", "data", "planning", "phase-plan.json")) {
				continue
			}
			if err := validateClaimedPlanArtifactFreshness(root, relPath, manifest); err != nil {
				return nil, err
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

func buildablePlanTaskCount(phases []colony.Phase) int {
	count := 0
	for _, phase := range phases {
		count += len(phase.Tasks)
	}
	return count
}

func validateClaimedPlanArtifactFreshness(root string, relPath string, manifest *codexPlanManifest) error {
	relPath = filepath.ToSlash(filepath.Clean(strings.TrimSpace(relPath)))
	absPath := filepath.Join(root, filepath.FromSlash(relPath))
	info, err := os.Lstat(absPath)
	if err != nil {
		return fmt.Errorf("claimed phase_plan %s is missing: %w", relPath, err)
	}
	if info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("claimed phase_plan %s is not a regular file", relPath)
	}

	if manifest != nil {
		if snapshot, ok := manifest.Snapshots[relPath]; ok && snapshot.Existed {
			currentHash := artifactContentHash(absPath)
			if snapshot.ContentHash != "" && currentHash == snapshot.ContentHash {
				return fmt.Errorf("claimed phase_plan %s was already present when plan_manifest was generated; rerun `aether plan --plan-only` and use fresh route-setter output", relPath)
			}
			if snapshot.ContentHash == "" && info.ModTime().Equal(snapshot.ModTime) && info.Size() == snapshot.Size {
				return fmt.Errorf("claimed phase_plan %s was already present when plan_manifest was generated; rerun `aether plan --plan-only` and use fresh route-setter output", relPath)
			}
		}
		if generatedAt, err := time.Parse(time.RFC3339, strings.TrimSpace(manifest.GeneratedAt)); err == nil {
			if info.ModTime().Before(generatedAt.UTC().Add(-planFinalizeManifestFutureSkew)) {
				return fmt.Errorf("claimed phase_plan %s predates plan_manifest generation; rerun `aether plan --plan-only` and use fresh route-setter output", relPath)
			}
		}
	}
	return nil
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
