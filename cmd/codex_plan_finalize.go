package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
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
	PlanningRunID    string                   `json:"planning_run_id,omitempty"`
	Iteration        int                      `json:"iteration,omitempty"`
	EvidenceHash     string                   `json:"evidence_hash,omitempty"`
	SourceSummary    string                   `json:"source_summary,omitempty"`
	Synthetic        bool                     `json:"synthetic,omitempty"`
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
	if err := validatePlanningIterationContract(manifest, completion); err != nil {
		return nil, err
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
	evidenceHash := planningCompletionEvidenceHash(scoutReport, *phasePlan)
	if strings.TrimSpace(completion.EvidenceHash) != "" && strings.TrimSpace(completion.EvidenceHash) != evidenceHash {
		return nil, fmt.Errorf("completion evidence_hash does not match Scout and Route-Setter evidence")
	}
	if err := validatePlanningConfidenceEvidence(*manifest, confidence, evidenceHash); err != nil {
		return nil, err
	}
	planningLoop := evaluatePlanningLoopIteration(*manifest, confidence, unresolvedGaps, evidenceHash)

	if planningLoop.StopReason == planningLoopPendingStop {
		result, err := persistIntermediatePlanningIteration(root, *manifest, dispatches, scoutReport, *phasePlan, confidence, unresolvedGaps, evidenceHash, planningLoop, provenance)
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	if err := validateNewPlanEvidenceContract(phases); err != nil {
		return nil, fmt.Errorf("new plan evidence contract is incomplete: %w", err)
	}

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
	// Never wipe phase-research: workers write real research there during
	// planning iterations, and finalize used to RemoveAll the directory and
	// regenerate templates from empty — destroying every artifact a worker
	// produced. Prune only files that no longer map to a current phase.
	if err := os.MkdirAll(phaseResearchDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create phase research directory: %w", err)
	}
	prunePhaseResearchOrphans(phaseResearchDir, phases)
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
	phaseResearchFiles, _, err := writePhaseResearchArtifacts(root, phaseResearchDir, manifest.Survey, scoutReport, phases, emptySnapshots, dispatches)
	if err != nil {
		return nil, err
	}

	if provenance.RecordWorkers {
		if err := recordExternalPlanSpawnTree(dispatches); err != nil {
			return nil, err
		}
	}

	planConfidence := float64(confidence.Overall) / 100.0
	updatedState := state
	var revision colony.PlanRevision
	if err := store.UpdateJSONAtomically("COLONY_STATE.json", &updatedState, func() error {
		updatedState = normalizeLegacyColonyState(updatedState)
		if updatedState.Goal == nil || strings.TrimSpace(*updatedState.Goal) != strings.TrimSpace(manifest.Goal) {
			return fmt.Errorf("plan_manifest goal does not match active colony goal")
		}
		if err := validatePlanManifestBase(root, *manifest, updatedState); err != nil {
			return err
		}
		acceptedPlan, acceptedRevision, err := activateGeneratedPlan(updatedState.Plan, phases, now, &planConfidence, colony.PlanEvidenceBoundV1, *manifest, evidenceHash)
		if err != nil {
			return err
		}
		updatedState.State = colony.StateREADY
		updatedState.CurrentPhase = firstBuildablePhase(acceptedPlan.Phases)
		updatedState.BuildStartedAt = nil
		updatedState.PlanGranularity = granularity
		if strings.TrimSpace(manifest.VerificationDepth) != "" {
			updatedState.VerificationDepth = string(colony.NormalizeVerificationDepth(manifest.VerificationDepth))
		}
		updatedState.Plan = acceptedPlan
		revision = acceptedRevision
		updatedState.Events = append(trimmedEvents(updatedState.Events),
			fmt.Sprintf("%s|planning_scout|plan-finalize|%s", now.Format(time.RFC3339), provenance.SourceSummary),
			fmt.Sprintf("%s|plan_revision_activated|plan-finalize|Activated %s (%s): %s", now.Format(time.RFC3339), revision.ID, revision.ReasonType, revision.Reason),
			fmt.Sprintf("%s|plan_generated|plan-finalize|Generated %d active phases with %d%% confidence from %s; planning loop stopped: %s", now.Format(time.RFC3339), len(acceptedPlan.Phases), confidence.Overall, provenance.SourceSummary, planningLoop.StopReason),
		)
		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to atomically activate plan revision: %w", err)
	}
	phases = updatedState.Plan.Phases
	_ = os.Remove(filepath.Join(store.BasePath(), planningIterationStateRel))
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
		"planning_run_id":           manifest.PlanningRunID,
		"iteration":                 manifest.Iteration,
		"evidence_hash":             evidenceHash,
		"plan_revision":             revision,
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
	if err := validatePlanManifestBase(manifest.Root, *manifest, state); err != nil {
		return state, granularity, err
	}
	if len(state.Plan.Phases) > 0 && !manifest.Refresh {
		if manifest.ExistingPlan {
			// Manifest acknowledges existing phases were detected during plan-only.
			// Proceed with finalization; the manifest's phases will replace them.
			return state, granularity, nil
		}
		return state, granularity, fmt.Errorf("stale phases from a prior session are blocking finalization; rerun `aether plan --plan-only --refresh` or run `aether init` to start fresh")
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

func validatePlanningIterationContract(manifest *codexPlanManifest, completion codexExternalPlanCompletion) error {
	if manifest == nil {
		return fmt.Errorf("completion file must include plan_manifest")
	}
	if manifest.Synthetic || completion.Synthetic {
		return fmt.Errorf("synthetic planning packets cannot satisfy the real plan-finalize contract; rerun real Scout and Route-Setter workers or use `aether plan --synthetic` only as a preview")
	}
	if completion.Synthesis != nil {
		return fmt.Errorf("ts-host synthesis planning packets cannot satisfy the real plan-finalize contract; rerun real Scout and Route-Setter workers or use `aether plan --synthetic` only as a preview")
	}
	if strings.TrimSpace(manifest.PlanningRunID) == "" {
		return fmt.Errorf("plan_manifest planning_run_id is required")
	}
	if manifest.Iteration < 1 {
		return fmt.Errorf("plan_manifest iteration must be >= 1")
	}
	if manifest.TargetConfidence <= 0 {
		return fmt.Errorf("plan_manifest target_confidence is required")
	}
	if manifest.MaxIterations <= 0 {
		return fmt.Errorf("plan_manifest max_iterations is required")
	}
	if strings.TrimSpace(completion.PlanningRunID) != "" && strings.TrimSpace(completion.PlanningRunID) != strings.TrimSpace(manifest.PlanningRunID) {
		return fmt.Errorf("completion planning_run_id does not match plan_manifest")
	}
	if completion.Iteration > 0 && completion.Iteration != manifest.Iteration {
		return fmt.Errorf("completion iteration does not match plan_manifest")
	}
	if err := validatePlanningWorkerChain("plan_manifest dispatches", manifest.Dispatches); err != nil {
		return err
	}
	if len(manifest.ExpectedWorkers) == 0 {
		return fmt.Errorf("plan_manifest expected_workers is required")
	}
	if err := validatePlanningWorkerChain("plan_manifest expected_workers", manifest.ExpectedWorkers); err != nil {
		return err
	}
	for i := range manifest.ExpectedWorkers {
		if err := validateExternalPlanIdentity(manifest.ExpectedWorkers[i], manifest.Dispatches[i]); err != nil {
			return fmt.Errorf("plan_manifest expected_workers[%d] does not match dispatches[%d]: %w", i, i, err)
		}
	}
	if previous, ok := loadPlanningIterationState(); ok &&
		strings.TrimSpace(previous.PlanningRunID) == strings.TrimSpace(manifest.PlanningRunID) &&
		previous.LastIteration >= manifest.Iteration {
		return fmt.Errorf("reused planning completion packet for run %s iteration %d; request a fresh manifest for the next iteration", manifest.PlanningRunID, manifest.Iteration)
	}
	return nil
}

func validatePlanningWorkerChain(label string, dispatches []codexPlanningDispatch) error {
	if len(dispatches) < 2 {
		return fmt.Errorf("%s must contain exactly one Scout followed by one Route-Setter", label)
	}
	if !strings.EqualFold(dispatches[0].Caste, "scout") {
		return fmt.Errorf("%s first worker must be Scout", label)
	}
	if !strings.EqualFold(dispatches[1].Caste, "route_setter") {
		return fmt.Errorf("%s second worker must be Route-Setter", label)
	}
	if dispatches[0].Wave >= dispatches[1].Wave {
		return fmt.Errorf("%s must order Scout before Route-Setter", label)
	}
	// The only workers permitted beyond the core pair are phase_research
	// Scouts — the per-phase domain research wave. Arbitrary worker sets stay
	// rejected. Research runs BEFORE the Route-Setter so its findings can
	// inform the route.
	for i, dispatch := range dispatches[2:] {
		if dispatch.Stage != phaseResearchStage || !strings.EqualFold(dispatch.Caste, "scout") {
			return fmt.Errorf("%s worker %d must be a phase_research Scout; other dynamic planning workers are not allowed", label, i+3)
		}
		if dispatch.Wave >= dispatches[1].Wave {
			return fmt.Errorf("%s must order phase research before the Route-Setter", label)
		}
	}
	return nil
}

func validatePlanningConfidenceEvidence(manifest codexPlanManifest, confidence codexPlanConfidence, evidenceHash string) error {
	if manifest.Iteration <= 1 || manifest.PreviousConfidence <= 0 {
		return nil
	}
	if strings.TrimSpace(manifest.PreviousEvidenceHash) == "" {
		return fmt.Errorf("plan_manifest previous_evidence_hash is required after iteration 1")
	}
	if strings.TrimSpace(manifest.PreviousEvidenceHash) == strings.TrimSpace(evidenceHash) && int(confidence.Overall) != manifest.PreviousConfidence {
		return fmt.Errorf("planning confidence changed from %d%% to %d%% without new Scout or Route-Setter evidence", manifest.PreviousConfidence, confidence.Overall)
	}
	return nil
}

func evaluatePlanningLoopIteration(manifest codexPlanManifest, confidence codexPlanConfidence, unresolvedGaps []string, evidenceHash string) codexPlanningLoop {
	loop := manifest.PlanningLoop
	if loop.TargetConfidence <= 0 || loop.MaxIterations <= 0 {
		loop = resolvePlanningLoopOptions(manifest.Depth, codexPlanOptions{
			TargetConfidence: manifest.TargetConfidence,
			MaxIterations:    manifest.MaxIterations,
			Accept:           manifest.PlanningLoop.Accept,
		})
	}
	if manifest.TargetConfidence > 0 {
		loop.TargetConfidence = manifest.TargetConfidence
	}
	if manifest.MaxIterations > 0 {
		loop.MaxIterations = manifest.MaxIterations
	}
	if loop.StallThreshold <= 0 {
		loop.StallThreshold = planningLoopStallThreshold
	}
	if loop.StallLimit <= 0 {
		loop.StallLimit = planningLoopStallLimit
	}

	iteration := manifest.Iteration
	if iteration < 1 {
		iteration = 1
	}
	overall := clampInt(int(confidence.Overall), 0, 100)
	gaps := limitStrings(uniqueSortedStrings(unresolvedGaps), 4)
	history := append([]codexPlanningLoopSample{}, loop.History...)
	previousConfidence := manifest.PreviousConfidence
	previousStallCount := 0
	if len(history) > 0 {
		last := history[len(history)-1]
		previousConfidence = last.Confidence
		previousStallCount = last.StallCount
	}
	delta := 0
	stallCount := 0
	if iteration > 1 {
		delta = overall - previousConfidence
		if delta < loop.StallThreshold {
			stallCount = previousStallCount + 1
		}
	}

	history = append(history, codexPlanningLoopSample{
		Iteration:     iteration,
		Confidence:    overall,
		Delta:         delta,
		StallCount:    stallCount,
		Gaps:          gaps,
		SelectedGaps:  selectedPlanningGapsForNext(confidence, gaps),
		Evidence:      planningCompletionEvidenceSummary(confidence, gaps),
		EvidenceCount: planningEvidenceCount(confidence),
		EvidenceHash:  evidenceHash,
	})

	loop.Iterations = iteration
	loop.FinalConfidence = overall
	loop.Gaps = gaps
	loop.History = history
	loop.StopReason = planningLoopPendingStop
	if loop.Accept {
		loop.StopReason = planningLoopAccepted
		loop.AcceptedBelowTarget = overall < loop.TargetConfidence
		return loop
	}
	if overall >= loop.TargetConfidence {
		loop.StopReason = planningLoopTargetReached
		return loop
	}
	if iteration >= loop.MaxIterations {
		loop.StopReason = planningLoopMaxIterationsHit
		return loop
	}
	if stallCount >= loop.StallLimit {
		loop.StopReason = planningLoopStalled
		return loop
	}
	return loop
}

func selectedPlanningGapsForNext(confidence codexPlanConfidence, gaps []string) []string {
	selected := selectPlanningIterationGaps(gaps)
	if len(selected) > 0 {
		return selected
	}
	type dimension struct {
		name  string
		score int
	}
	dimensions := []dimension{
		{name: "knowledge evidence", score: int(confidence.Knowledge)},
		{name: "requirements evidence", score: int(confidence.Requirements)},
		{name: "risk evidence", score: int(confidence.Risks)},
		{name: "dependency evidence", score: int(confidence.Dependencies)},
		{name: "effort evidence", score: int(confidence.Effort)},
	}
	sort.SliceStable(dimensions, func(i, j int) bool {
		return dimensions[i].score < dimensions[j].score
	})
	for _, dim := range dimensions {
		if dim.score > 0 && dim.score < 85 {
			selected = append(selected, fmt.Sprintf("Gather stronger %s; current score %d%%", dim.name, dim.score))
		}
		if len(selected) == 2 {
			break
		}
	}
	if len(selected) == 0 {
		selected = append(selected, "Gather fresh repository-specific evidence before raising planning confidence")
	}
	return selected
}

func planningCompletionEvidenceHash(scout codexScoutReport, phasePlan codexWorkerPlanArtifact) string {
	scout = normalizeScoutPlanningReport(scout)
	source := struct {
		ScoutFindings []codexScoutFinding    `json:"scout_findings"`
		ScoutGaps     []string               `json:"scout_gaps"`
		StudyFiles    []string               `json:"study_files"`
		Phases        []codexWorkerPlanPhase `json:"phases"`
		Gaps          []string               `json:"gaps"`
	}{
		ScoutFindings: scout.Findings,
		ScoutGaps:     uniqueSortedStrings(scout.Gaps),
		StudyFiles:    uniqueSortedStrings(scout.StudyFiles),
		Phases:        append([]codexWorkerPlanPhase{}, phasePlan.Phases...),
		Gaps:          uniqueSortedStrings(phasePlan.Gaps),
	}
	data, _ := json.Marshal(source)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func planningCompletionEvidenceSummary(confidence codexPlanConfidence, gaps []string) string {
	return fmt.Sprintf(
		"Scout and Route-Setter evidence; confidence inputs knowledge=%d requirements=%d risks=%d dependencies=%d effort=%d overall=%d; unresolved_gaps=%d",
		confidence.Knowledge,
		confidence.Requirements,
		confidence.Risks,
		confidence.Dependencies,
		confidence.Effort,
		confidence.Overall,
		len(gaps),
	)
}

func persistIntermediatePlanningIteration(root string, manifest codexPlanManifest, dispatches []codexPlanningDispatch, scoutReport codexScoutReport, phasePlan codexWorkerPlanArtifact, confidence codexPlanConfidence, unresolvedGaps []string, evidenceHash string, planningLoop codexPlanningLoop, provenance codexPlanProvenance) (map[string]interface{}, error) {
	now := time.Now().UTC()
	planningDir := filepath.Join(store.BasePath(), "planning")
	iterationsDir := filepath.Join(planningDir, "iterations")
	if err := os.MkdirAll(iterationsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create planning iterations directory: %w", err)
	}
	iterPrefix := fmt.Sprintf("iteration-%02d", manifest.Iteration)
	if err := writePlanningJSONFile(filepath.Join(iterationsDir, iterPrefix+"-scout.json"), scoutReport); err != nil {
		return nil, err
	}
	if err := writePlanningJSONFile(filepath.Join(iterationsDir, iterPrefix+"-phase-plan.json"), phasePlan); err != nil {
		return nil, err
	}
	selectedGaps := selectedPlanningGapsForNext(confidence, unresolvedGaps)
	state := codexPlanIterationState{
		PlanningRunID:        manifest.PlanningRunID,
		Goal:                 manifest.Goal,
		Root:                 root,
		Depth:                manifest.Depth,
		PlanningDepth:        manifest.PlanningDepth,
		TargetConfidence:     manifest.TargetConfidence,
		MaxIterations:        manifest.MaxIterations,
		LastIteration:        manifest.Iteration,
		PreviousConfidence:   int(confidence.Overall),
		PreviousEvidenceHash: evidenceHash,
		SelectedGaps:         selectedGaps,
		PreviousPlanDraft:    &phasePlan,
		History:              append([]codexPlanningLoopSample{}, planningLoop.History...),
		Revision:             manifest.Revision,
		UpdatedAt:            now.Format(time.RFC3339),
	}
	if len(planningLoop.History) > 0 {
		state.ConsecutiveStalls = planningLoop.History[len(planningLoop.History)-1].StallCount
	}
	if err := store.SaveJSON(planningIterationStateRel, state); err != nil {
		return nil, fmt.Errorf("failed to save planning iteration state: %w", err)
	}
	if provenance.RecordWorkers {
		if err := recordExternalPlanSpawnTree(dispatches); err != nil {
			return nil, err
		}
	}
	nextCommand := fmt.Sprintf("aether host plan --depth %s --planning-depth %s --target %d --max-iterations %d", manifest.Depth, manifest.PlanningDepth, manifest.TargetConfidence, manifest.MaxIterations)
	if revisionArgs := planRevisionCLIArgs(manifest.Revision); revisionArgs != "" {
		nextCommand += " " + revisionArgs
	}
	updateSessionSummary("plan-finalize", nextCommand, fmt.Sprintf("Planning iteration %d reached %d%% confidence; another iteration is required", manifest.Iteration, confidence.Overall))
	return map[string]interface{}{
		"planned":                 false,
		"iteration_completed":     true,
		"requires_next_iteration": true,
		"planning_run_id":         manifest.PlanningRunID,
		"iteration":               manifest.Iteration,
		"goal":                    manifest.Goal,
		"depth":                   manifest.Depth,
		"planning_depth":          manifest.PlanningDepth,
		"confidence":              confidence,
		"planning_loop":           planningLoop,
		"dispatches":              planningDispatchMaps(dispatches),
		"dispatch_mode":           provenance.DispatchMode,
		"artifact_source":         "planning-iteration",
		"plan_source":             provenance.PlanSource,
		"gaps":                    unresolvedGaps,
		"selected_gaps":           selectedGaps,
		"evidence_hash":           evidenceHash,
		"planning_dir":            planningDir,
		"iteration_state":         planningIterationStateRel,
		"planning_warning":        "Planning iteration validated, but confidence target has not been reached; final colony plan was not written.",
		"revision_request":        manifest.Revision,
		"next":                    nextCommand,
	}, nil
}

func writePlanningJSONFile(path string, value interface{}) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", filepath.Base(path), err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
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
	scoutReport, err := c.scoutReport(dispatches)
	if err != nil {
		return codexPlanProvenance{}, err
	}
	return codexPlanProvenance{
		Dispatches:      dispatches,
		ScoutReport:     scoutReport,
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

func (c codexExternalPlanCompletion) scoutReport(dispatches []codexPlanningDispatch) (codexScoutReport, error) {
	if c.ScoutReport != nil {
		report := normalizeScoutPlanningReport(*c.ScoutReport)
		if scoutReportHasContent(report) {
			return report, nil
		}
	}
	for _, dispatch := range dispatches {
		if dispatch.ScoutReport != nil {
			report := normalizeScoutPlanningReport(*dispatch.ScoutReport)
			if scoutReportHasContent(report) {
				return report, nil
			}
		}
	}
	return codexScoutReport{}, fmt.Errorf("completion file must include Scout evidence as scout_report from the Scout worker")
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
