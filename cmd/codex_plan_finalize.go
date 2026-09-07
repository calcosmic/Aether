package cmd

import (
	"bytes"
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
	ScoutResult      json.RawMessage          `json:"scout_result,omitempty"`
}

// planningScoutStageResult is the complete worker-owned payload permitted at
// the Scout boundary. Deliberately absent are confidence, semantic plan,
// stop, candidate, acceptance, activation, and state-patch fields. Strict
// decoding below makes those omissions enforceable rather than documentary.
type planningScoutStageResult struct {
	ResultType           planningStageResultType           `json:"result_type"`
	ManifestID           string                            `json:"manifest_id"`
	ManifestHash         string                            `json:"manifest_hash"`
	RunID                string                            `json:"run_id"`
	Pass                 int                               `json:"pass"`
	Caste                planningStageWorkerCaste          `json:"caste"`
	Specification        planningStageSpecificationBinding `json:"specification"`
	BasePlanRevisionID   string                            `json:"base_plan_revision_id"`
	BasePlanRevisionHash string                            `json:"base_plan_revision_hash"`
	InputFrontierHash    string                            `json:"input_frontier_hash"`
	Findings             []planningScoutStageFinding       `json:"findings"`
	NewEvidence          []planningEvidenceRecord          `json:"new_evidence"`
	UnresolvedGaps       []colony.PlanningGap              `json:"unresolved_gaps"`
	DecisionCandidates   []planningDecisionCandidate       `json:"material_decision_candidates"`
}

// planningScoutStageFinding is intentionally smaller than a plan proposal.
// Every claim must cite an authorized/new evidence address or say explicitly
// that the Scout could not establish the fact.
type planningScoutStageFinding struct {
	StableID      string   `json:"stable_id"`
	Summary       string   `json:"summary"`
	EvidenceIDs   []string `json:"evidence_ids,omitempty"`
	Unknown       bool     `json:"unknown,omitempty"`
	UnknownReason string   `json:"unknown_reason,omitempty"`
}

type planningScoutStageFinalization struct {
	Result   planningScoutStageResult
	Artifact planningStageOutputReference
	Receipt  StageReceipt
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
			refusal, closeErr := lifecycleCloseoutRefusalFromDisk("plan", err.Error(), "", "")
			if closeErr != nil {
				outputError(1, err.Error(), nil)
			} else {
				outputError(1, err.Error(), refusal)
			}
			return renderedErrorExit(1)
		}
		result, err := runCodexPlanFinalize(skillWorkspaceRoot(), completion)
		if err != nil {
			recordPlanFinalizeFailure(err)
			outputError(1, err.Error(), nil)
			return renderedErrorExit(1)
		}
		// WINDOWS.md entry 7 (198-REVIEW.md WR scope): runCodexPlanFinalize
		// now calls closeLifecycleRun itself (mirroring
		// runCodexContinueFinalize/completeSealRuntime), which already folds
		// its own suggested "next" command into the envelope. A second,
		// override-less closeLifecycleCommand call here would unconditionally
		// overwrite that with an independently-resolved answer
		// (applyNextActionToResult has no "already set" guard), silently
		// undoing the fold-in this exact entry was about.
		result["state_effect"] = colony.LifecycleStateEffectCommitted
		if boolValue(result["planned"]) {
			result["outcome_kind"] = colony.OutcomeKindCompleted
		} else {
			result["outcome_kind"] = colony.OutcomeKindInProgress
		}
		if err := applyLifecycleCloseout(result, "plan", planLifecycleCloseoutDetails(result)); err != nil {
			outputError(1, err.Error(), result)
			return renderedErrorExit(1)
		}
		visual := appendLifecycleCloseoutVisual(renderPlanVisual(result), result, detectPlatform())
		outputWorkflow(result, visual)
		return nil
	},
}

func planLifecycleCloseoutDetails(result map[string]interface{}) LifecycleCloseoutDetails {
	details := LifecycleCloseoutDetails{Summary: "Scout and Route-Setter evidence reached the declared planning boundary."}
	if boolValue(result["planned"]) {
		details.Summary = "Scout and Route-Setter evidence produced an accepted plan."
	}
	if revision, ok := planRevisionFromValue(result["plan_revision"]); ok {
		details.Evidence = append(details.Evidence, colony.LifecycleEvidence{
			ID: revision.ID, Kind: "plan", Source: "COLONY_STATE.json", Summary: "Accepted plan revision",
		})
	}
	if evidenceHash := strings.TrimSpace(stringValue(result["evidence_hash"])); evidenceHash != "" {
		details.Evidence = append(details.Evidence, colony.LifecycleEvidence{
			ID: "planning-evidence", Kind: "digest", Digest: evidenceHash, Source: strings.TrimSpace(stringValue(result["planning_dir"])), Summary: "Scout and Route-Setter evidence digest",
		})
	}
	if len(details.Evidence) == 0 {
		details.Evidence = append(details.Evidence, colony.LifecycleEvidence{
			ID: "planning-iteration", Kind: "planning", Source: strings.TrimSpace(stringValue(result["iteration_state"])), Summary: "Persisted planning iteration evidence",
		})
	}
	for index, gap := range stringSliceValue(result["gaps"]) {
		details.Blockers = append(details.Blockers, colony.LifecycleIssue{ID: fmt.Sprintf("planning-gap-%d", index+1), Summary: gap})
	}
	for _, key := range []string{"planning_warning", "research_warning", "clarification_warning"} {
		if warning := strings.TrimSpace(stringValue(result[key])); warning != "" {
			details.Warnings = append(details.Warnings, colony.LifecycleIssue{ID: "plan-" + strings.ReplaceAll(key, "_", "-"), Summary: warning})
		}
	}
	return details
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
	if manifest.StageManifest != nil || len(bytes.TrimSpace(completion.ScoutResult)) > 0 {
		return runCodexScoutStageFinalize(root, *manifest, completion)
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
	if manifest.TerritoryRequired {
		if err := validatePlanTerritorySnapshot(root, *manifest); err != nil {
			return nil, err
		}
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
		// WINDOWS.md entry 7 (198-REVIEW.md WR scope): unlike
		// continue-finalize/completeSealRuntime, this finalizer never called
		// closeLifecycleRun, so its own suggested "next" command never folded
		// into the unified next-action envelope renderLifecycleClosing reads
		// back -- the closing card instead independently resolved a next step
		// from live colony state, which usually matched but was not
		// guaranteed to.
		closeLifecycleRun(result, state, "plan")
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
	phaseResearchFiles, _, researchFailedPhases, err := writePhaseResearchArtifacts(root, phaseResearchDir, manifest.Survey, scoutReport, phases, emptySnapshots, dispatches)
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
		"research_failed_phases":    researchFailedPhases,
		"research_warning":          renderResearchFailedWarning(researchFailedPhases),
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
	// WINDOWS.md entry 7 (198-REVIEW.md WR scope): see the identical comment
	// on the mid-loop branch above.
	closeLifecycleRun(result, updatedState, "plan")
	return result, nil
}

func runCodexScoutStageFinalize(root string, manifest codexPlanManifest, completion codexExternalPlanCompletion) (map[string]interface{}, error) {
	if manifest.StageManifest == nil {
		return nil, fmt.Errorf("Scout completion requires the exact stage_manifest")
	}
	if len(bytes.TrimSpace(completion.ScoutResult)) == 0 {
		return nil, fmt.Errorf("Scout completion requires scout_result")
	}
	if len(completion.workerResults()) != 0 || completion.ScoutReport != nil || completion.PhasePlan != nil || completion.Synthesis != nil {
		return nil, fmt.Errorf("Scout completion cannot include legacy whole-chain worker, Scout report, phase plan, or synthesis fields")
	}
	if (manifest.DispatchMode != "plan-only" && manifest.DispatchMode != "agent-delegate") || !manifest.RequiresFinalizer {
		return nil, fmt.Errorf("plan_manifest must come from `aether plan --plan-only` or an agent-delegate planning response")
	}
	if err := validateFinalizerManifestRoot("plan_manifest", manifest.Root, root); err != nil {
		return nil, err
	}
	if err := validateCodexPlanManifestFreshness(manifest, time.Now().UTC()); err != nil {
		return nil, err
	}
	stageManifest := *manifest.StageManifest
	if err := validatePlanningStageManifest(stageManifest); err != nil {
		return nil, fmt.Errorf("stage_manifest: %w", err)
	}
	if stageManifest.ExpectedCaste != planningStageCasteScout {
		return nil, fmt.Errorf("stage_manifest caste %q is not Scout", stageManifest.ExpectedCaste)
	}
	if manifest.PlanningRunID != stageManifest.RunID || manifest.Iteration != stageManifest.Pass || manifest.SelectedPreset != stageManifest.Preset ||
		manifest.BaseRevisionID != stageManifest.BasePlanRevisionID || manifest.BasePlanStateHash != stageManifest.BasePlanRevisionHash {
		return nil, fmt.Errorf("plan_manifest does not match the exact Scout stage run, pass, preset, or base plan revision")
	}
	if len(manifest.Dispatches) != 1 || len(manifest.ExpectedWorkers) != 1 {
		return nil, fmt.Errorf("Scout plan_manifest must contain exactly one Scout dispatch")
	}
	for _, dispatch := range []codexPlanningDispatch{manifest.Dispatches[0], manifest.ExpectedWorkers[0]} {
		if !strings.EqualFold(dispatch.Caste, string(planningStageCasteScout)) || dispatch.StageManifest == nil ||
			dispatch.StageManifest.ID != stageManifest.ID || dispatch.StageManifest.ContentHash != stageManifest.ContentHash {
			return nil, fmt.Errorf("Scout dispatch does not bind the exact stage_manifest")
		}
	}
	if manifest.PlanningRunHeader != nil {
		header := manifest.PlanningRunHeader
		if header.RunID != stageManifest.RunID || header.StageManifestID != stageManifest.ID || header.StageManifestHash != stageManifest.ContentHash ||
			header.Specification.RevisionID != stageManifest.Specification.RevisionID || header.Specification.ContentHash != stageManifest.Specification.ContentHash ||
			header.BasePlanRevisionID != stageManifest.BasePlanRevisionID || header.BasePlanRevisionHash != stageManifest.BasePlanRevisionHash ||
			header.InputFrontierHash != stageManifest.InputFrontierHash {
			return nil, fmt.Errorf("planning run header does not match the exact Scout stage authority")
		}
	}

	completed, err := finalizePlanningScoutStage(root, stageManifest, completion.ScoutResult)
	if err != nil {
		return nil, err
	}
	evidence := make([]colony.PlanningEvidenceRef, 0, len(completed.Result.NewEvidence))
	for _, record := range completed.Result.NewEvidence {
		evidence = append(evidence, record.Reference)
	}
	state, err := loadPlanningStageState(root, stageManifest.RunID)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"planned":                      false,
		"status":                       "scout_complete",
		"scout_complete":               true,
		"planning_run_id":              stageManifest.RunID,
		"iteration":                    stageManifest.Pass,
		"stage":                        string(state.Stage),
		"next_boundary":                string(state.Stage),
		"stage_receipt":                completed.Receipt,
		"scout_artifact":               completed.Artifact,
		"scout_result":                 completed.Result,
		"evidence_added":               evidence,
		"evidence_added_count":         len(evidence),
		"gaps_found":                   append([]colony.PlanningGap(nil), completed.Result.UnresolvedGaps...),
		"material_decision_candidates": append([]planningDecisionCandidate(nil), completed.Result.DecisionCandidates...),
		"iteration_card_created":       false,
		"next":                         "aether plan",
	}, nil
}

// finalizePlanningScoutStage consumes only the current Scout manifest. The
// normalized artifact is persisted first as the durable worker-output
// boundary; finalizePlanningStage then atomically commits its receipt and the
// resulting lifecycle state. Exact retries resolve to the same artifact and
// receipt, while changed bytes conflict before the frontier can move.
func finalizePlanningScoutStage(root string, manifest planningStageManifest, raw []byte) (planningScoutStageFinalization, error) {
	empty := planningScoutStageFinalization{}
	result, normalized, material, err := validatePlanningScoutStageResult(root, manifest, raw)
	if err != nil {
		return empty, err
	}
	artifact, err := writePlanningStageOutput(root, manifest, normalized, planningStageWriteOptions{})
	if err != nil {
		return empty, err
	}
	next := planningStageRouteReady
	decisionResume := planningStage("")
	if manifest.Pass == 1 && material {
		next = planningStageOwnerDecision
		decisionResume = planningStageRouteReady
	}
	candidateSnapshotHash, err := planningScoutCandidateSnapshotHash(manifest)
	if err != nil {
		return empty, err
	}
	receipt, err := finalizePlanningStage(root, manifest, planningStageFinalizeRequest{
		To:                    next,
		CandidateSnapshotHash: candidateSnapshotHash,
		DecisionResumeStage:   decisionResume,
	})
	if err != nil {
		return empty, err
	}
	return planningScoutStageFinalization{Result: result, Artifact: artifact, Receipt: receipt}, nil
}

func validatePlanningScoutStageResult(root string, manifest planningStageManifest, raw []byte) (planningScoutStageResult, []byte, bool, error) {
	empty := planningScoutStageResult{}
	if err := validatePlanningStageManifest(manifest); err != nil {
		return empty, nil, false, err
	}
	if manifest.ExpectedCaste != planningStageCasteScout || manifest.ExpectedResultType != planningStageResultScout {
		return empty, nil, false, fmt.Errorf("active planning stage manifest is not a Scout contract")
	}
	result, err := decodePlanningScoutStageResult(raw)
	if err != nil {
		return empty, nil, false, err
	}
	if result.ResultType != planningStageResultScout {
		return empty, nil, false, fmt.Errorf("Scout result_type must be %q", planningStageResultScout)
	}
	if result.ManifestID != manifest.ID || result.ManifestHash != manifest.ContentHash {
		return empty, nil, false, fmt.Errorf("Scout result manifest ID or hash does not match the active manifest")
	}
	if result.RunID != manifest.RunID {
		return empty, nil, false, fmt.Errorf("Scout result run does not match the active manifest")
	}
	if result.Pass != manifest.Pass {
		return empty, nil, false, fmt.Errorf("Scout result pass does not match the active manifest")
	}
	if result.Caste != planningStageCasteScout || result.Caste != manifest.ExpectedCaste {
		return empty, nil, false, fmt.Errorf("Scout result caste does not match the active Scout manifest")
	}
	if !samePlanningScoutSpecification(result.Specification, manifest.Specification) {
		return empty, nil, false, fmt.Errorf("Scout result specification does not match the exact approved specification binding")
	}
	if result.BasePlanRevisionID != manifest.BasePlanRevisionID || result.BasePlanRevisionHash != manifest.BasePlanRevisionHash {
		return empty, nil, false, fmt.Errorf("Scout result base plan revision does not match the active manifest")
	}
	if result.InputFrontierHash != manifest.InputFrontierHash {
		return empty, nil, false, fmt.Errorf("Scout result frontier does not match the active manifest")
	}

	header, err := loadPlanningScoutRunHeader(root, manifest)
	if err != nil {
		return empty, nil, false, err
	}
	result, material, err := normalizePlanningScoutStageContent(root, header, manifest, result)
	if err != nil {
		return empty, nil, false, err
	}
	normalized, err := marshalPlanningStageJSON(result)
	if err != nil {
		return empty, nil, false, fmt.Errorf("marshal normalized Scout result: %w", err)
	}
	return result, normalized, material, nil
}

func decodePlanningScoutStageResult(raw []byte) (planningScoutStageResult, error) {
	var result planningScoutStageResult
	if len(bytes.TrimSpace(raw)) == 0 {
		return result, fmt.Errorf("Scout result is empty")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return planningScoutStageResult{}, fmt.Errorf("decode Scout stage result: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return planningScoutStageResult{}, fmt.Errorf("Scout result must contain exactly one JSON value")
		}
		return planningScoutStageResult{}, fmt.Errorf("decode trailing Scout result data: %w", err)
	}
	return result, nil
}

func loadPlanningScoutRunHeader(root string, manifest planningStageManifest) (planningRunHeader, error) {
	var header planningRunHeader
	repositoryRoot, err := canonicalPlanningTimelineRoot(root)
	if err != nil {
		return header, err
	}
	headerPath := filepath.ToSlash(filepath.Join(".aether", "data", "planning", manifest.RunID, "run-header.json"))
	content, exists, err := readOptionalPlanningStageFile(repositoryRoot, headerPath)
	if err != nil {
		return header, err
	}
	if !exists {
		return header, fmt.Errorf("planning run header for %q is missing", manifest.RunID)
	}
	if err := decodePlanningStageJSON(content, &header); err != nil {
		return planningRunHeader{}, fmt.Errorf("decode planning run header: %w", err)
	}
	payload := header
	payload.ID = ""
	payload.ContentHash = ""
	wantHash, err := jsonSHA256(payload)
	if err != nil {
		return planningRunHeader{}, fmt.Errorf("hash planning run header: %w", err)
	}
	if header.SchemaVersion != planningRunHeaderSchemaVersion || header.ContentHash != wantHash || header.ID != "planning-run-header-"+wantHash[:16] {
		return planningRunHeader{}, fmt.Errorf("planning run header is not a valid immutable content address")
	}
	if header.RunID != manifest.RunID || header.Preset != manifest.Preset ||
		!samePlanningScoutSpecification(header.Specification, manifest.Specification) ||
		header.BasePlanRevisionID != manifest.BasePlanRevisionID || header.BasePlanRevisionHash != manifest.BasePlanRevisionHash {
		return planningRunHeader{}, fmt.Errorf("planning run header does not match the active Scout manifest authority")
	}
	if strings.TrimSpace(header.GoalID) == "" || strings.TrimSpace(header.SessionID) == "" {
		return planningRunHeader{}, fmt.Errorf("planning run header requires goal and session scope")
	}
	if manifest.Pass == 1 && (header.StageManifestID != manifest.ID || header.StageManifestHash != manifest.ContentHash || header.InputFrontierHash != manifest.InputFrontierHash) {
		return planningRunHeader{}, fmt.Errorf("planning run header does not bind the first Scout manifest and frontier")
	}
	return header, nil
}

func normalizePlanningScoutStageContent(root string, header planningRunHeader, manifest planningStageManifest, result planningScoutStageResult) (planningScoutStageResult, bool, error) {
	result.ManifestID = strings.TrimSpace(result.ManifestID)
	result.ManifestHash = strings.TrimSpace(result.ManifestHash)
	result.RunID = strings.TrimSpace(result.RunID)
	result.BasePlanRevisionID = strings.TrimSpace(result.BasePlanRevisionID)
	result.BasePlanRevisionHash = strings.TrimSpace(result.BasePlanRevisionHash)
	result.InputFrontierHash = strings.TrimSpace(result.InputFrontierHash)

	allowedBindings := make(map[string]string, len(manifest.EvidenceFrontier)+len(result.NewEvidence))
	allowedReferences := make(map[string]colony.PlanningEvidenceRef, len(header.EvidenceCatalogue)+len(result.NewEvidence))
	for _, binding := range manifest.EvidenceFrontier {
		if err := binding.validate(); err != nil {
			return planningScoutStageResult{}, false, fmt.Errorf("Scout evidence frontier: %w", err)
		}
		if _, duplicate := allowedBindings[binding.ID]; duplicate {
			return planningScoutStageResult{}, false, fmt.Errorf("Scout evidence frontier repeats %q", binding.ID)
		}
		allowedBindings[binding.ID] = binding.ContentHash
	}
	for _, record := range header.EvidenceCatalogue {
		if hash, ok := allowedBindings[record.Reference.ID]; ok && hash == record.Reference.ContentHash {
			allowedReferences[record.Reference.ID] = record.Reference
		}
	}

	seenNew := make(map[string]struct{}, len(result.NewEvidence))
	for _, record := range result.NewEvidence {
		if _, duplicate := seenNew[record.Reference.ID]; duplicate {
			return planningScoutStageResult{}, false, fmt.Errorf("Scout new evidence repeats %q", record.Reference.ID)
		}
		seenNew[record.Reference.ID] = struct{}{}
		if _, restated := allowedBindings[record.Reference.ID]; restated {
			return planningScoutStageResult{}, false, fmt.Errorf("Scout new evidence %q restates the prior frontier", record.Reference.ID)
		}
	}
	newCatalogue, err := collectPlanningEvidence(planningEvidenceCollectionRequest{RepositoryRoot: root, Existing: result.NewEvidence})
	if err != nil {
		return planningScoutStageResult{}, false, fmt.Errorf("validate Scout new evidence: %w", err)
	}
	if len(newCatalogue) != len(result.NewEvidence) {
		return planningScoutStageResult{}, false, fmt.Errorf("Scout new evidence contains duplicate content addresses")
	}
	for _, record := range newCatalogue {
		ref := record.Reference
		if ref.GoalID != header.GoalID || ref.SessionID != header.SessionID || ref.SpecificationRevisionID != manifest.Specification.RevisionID || ref.PlanRevisionID != manifest.BasePlanRevisionID {
			return planningScoutStageResult{}, false, fmt.Errorf("Scout new evidence %q does not match the current goal, session, specification, and base plan scope", ref.ID)
		}
		if !ref.Fresh || !ref.Admissible {
			return planningScoutStageResult{}, false, fmt.Errorf("Scout new evidence %q is not fresh and admissible", ref.ID)
		}
		allowedBindings[ref.ID] = ref.ContentHash
		allowedReferences[ref.ID] = ref
	}
	result.NewEvidence = newCatalogue

	findings := append([]planningScoutStageFinding(nil), result.Findings...)
	seenFindings := make(map[string]struct{}, len(findings))
	for index := range findings {
		finding := &findings[index]
		finding.StableID = strings.TrimSpace(finding.StableID)
		finding.Summary = normalizePlanningDecisionText(finding.Summary)
		finding.EvidenceIDs = nonEmptyPlanningDecisionIDs(finding.EvidenceIDs)
		finding.UnknownReason = normalizePlanningDecisionText(finding.UnknownReason)
		if finding.StableID == "" || finding.Summary == "" {
			return planningScoutStageResult{}, false, fmt.Errorf("Scout finding %d requires stable_id and summary", index)
		}
		if _, duplicate := seenFindings[finding.StableID]; duplicate {
			return planningScoutStageResult{}, false, fmt.Errorf("Scout finding ID %q is duplicated", finding.StableID)
		}
		seenFindings[finding.StableID] = struct{}{}
		if len(finding.EvidenceIDs) == 0 && (!finding.Unknown || finding.UnknownReason == "") {
			return planningScoutStageResult{}, false, fmt.Errorf("Scout finding %q requires evidence or an explicit unknown reason", finding.StableID)
		}
		if finding.Unknown && finding.UnknownReason == "" {
			return planningScoutStageResult{}, false, fmt.Errorf("Scout finding %q marks unknown without an unknown reason", finding.StableID)
		}
		if err := validatePlanningScoutCitationIDs("finding "+finding.StableID, finding.EvidenceIDs, allowedBindings); err != nil {
			return planningScoutStageResult{}, false, err
		}
	}
	sort.Slice(findings, func(left, right int) bool { return findings[left].StableID < findings[right].StableID })
	result.Findings = findings

	gaps := append([]colony.PlanningGap(nil), result.UnresolvedGaps...)
	seenGaps := make(map[string]struct{}, len(gaps))
	for index := range gaps {
		gaps[index].EvidenceIDs = nonEmptyPlanningDecisionIDs(gaps[index].EvidenceIDs)
		if err := gaps[index].Validate(); err != nil {
			return planningScoutStageResult{}, false, fmt.Errorf("Scout unresolved_gaps[%d]: %w", index, err)
		}
		if _, duplicate := seenGaps[gaps[index].ID]; duplicate {
			return planningScoutStageResult{}, false, fmt.Errorf("Scout unresolved gap ID %q is duplicated", gaps[index].ID)
		}
		seenGaps[gaps[index].ID] = struct{}{}
		if len(gaps[index].EvidenceIDs) == 0 && strings.TrimSpace(gaps[index].EvidenceThatWouldChange) == "" {
			return planningScoutStageResult{}, false, fmt.Errorf("Scout unresolved gap %q requires evidence or an explicit unknown", gaps[index].ID)
		}
		if err := validatePlanningScoutCitationIDs("gap "+gaps[index].ID, gaps[index].EvidenceIDs, allowedBindings); err != nil {
			return planningScoutStageResult{}, false, err
		}
	}
	sort.Slice(gaps, func(left, right int) bool { return gaps[left].ID < gaps[right].ID })
	result.UnresolvedGaps = gaps

	candidates := append([]planningDecisionCandidate(nil), result.DecisionCandidates...)
	seenCandidates := make(map[string]struct{}, len(candidates))
	material := false
	for index := range candidates {
		candidate := canonicalPlanningDecisionCandidate(candidates[index])
		if _, duplicate := seenCandidates[candidate.StableID]; duplicate {
			return planningScoutStageResult{}, false, fmt.Errorf("Scout material decision ID %q is duplicated", candidate.StableID)
		}
		seenCandidates[candidate.StableID] = struct{}{}
		for evidenceIndex, ref := range candidate.Evidence {
			if err := ref.Validate(); err != nil {
				return planningScoutStageResult{}, false, fmt.Errorf("Scout decision %q evidence[%d]: %w", candidate.StableID, evidenceIndex, err)
			}
			wantHash, ok := allowedBindings[ref.ID]
			if !ok || wantHash != ref.ContentHash {
				return planningScoutStageResult{}, false, fmt.Errorf("Scout decision %q cites evidence %q outside the current frontier", candidate.StableID, ref.ID)
			}
			if known, ok := allowedReferences[ref.ID]; ok && !samePlanningEvidenceReference(known, ref) {
				return planningScoutStageResult{}, false, fmt.Errorf("Scout decision %q changes evidence reference %q", candidate.StableID, ref.ID)
			}
			if ref.GoalID != header.GoalID || ref.SessionID != header.SessionID || ref.SpecificationRevisionID != manifest.Specification.RevisionID || ref.PlanRevisionID != manifest.BasePlanRevisionID {
				return planningScoutStageResult{}, false, fmt.Errorf("Scout decision %q cites stale evidence %q", candidate.StableID, ref.ID)
			}
		}
		classification, err := classifyPlanningDecision(candidate)
		if err != nil {
			return planningScoutStageResult{}, false, fmt.Errorf("Scout decision candidate %d: %w", index, err)
		}
		for choiceIndex := range candidate.Choices {
			if err := validatePlanningDecisionChoice(candidate.Choices[choiceIndex]); err != nil {
				return planningScoutStageResult{}, false, fmt.Errorf("Scout decision %q choice[%d]: %w", candidate.StableID, choiceIndex, err)
			}
		}
		material = material || classification.RequiresOwner
		candidates[index] = candidate
	}
	sort.Slice(candidates, func(left, right int) bool { return candidates[left].StableID < candidates[right].StableID })
	result.DecisionCandidates = candidates
	return result, material, nil
}

func validatePlanningScoutCitationIDs(label string, ids []string, allowed map[string]string) error {
	for _, id := range ids {
		if _, ok := allowed[id]; !ok {
			return fmt.Errorf("Scout %s cites evidence %q outside the current or newly validated frontier", label, id)
		}
	}
	return nil
}

func samePlanningScoutSpecification(left, right planningStageSpecificationBinding) bool {
	return left.RevisionID == right.RevisionID && left.ContentHash == right.ContentHash &&
		left.PredecessorRevisionID == right.PredecessorRevisionID && left.Status == right.Status &&
		left.ApprovalReceiptID == right.ApprovalReceiptID && left.ApprovalReceiptHash == right.ApprovalReceiptHash
}

func samePlanningEvidenceReference(left, right colony.PlanningEvidenceRef) bool {
	leftBytes, leftErr := json.Marshal(left)
	rightBytes, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftBytes, rightBytes)
}

func planningScoutCandidateSnapshotHash(manifest planningStageManifest) (string, error) {
	return jsonSHA256(struct {
		RunID                string `json:"run_id"`
		Pass                 int    `json:"pass"`
		BasePlanRevisionID   string `json:"base_plan_revision_id"`
		BasePlanRevisionHash string `json:"base_plan_revision_hash"`
		PriorCardHash        string `json:"prior_card_hash"`
		InputFrontierHash    string `json:"input_frontier_hash"`
	}{manifest.RunID, manifest.Pass, manifest.BasePlanRevisionID, manifest.BasePlanRevisionHash, manifest.PriorCardHash, manifest.InputFrontierHash})
}

// attachTerritoryToPlanManifest copies the verified immutable evidence into a
// planning manifest. Maps and slices are cloned so later renderer or caller
// changes cannot silently rewrite the finalizer's input.
func attachTerritoryToPlanManifest(manifest *codexPlanManifest, freshness SurveyFreshnessResult) {
	if manifest == nil {
		return
	}
	freshness.ArtifactDigests = cloneStringMap(freshness.ArtifactDigests)
	freshness.ReasonCodes = append([]SurveyFreshnessReasonCode{}, freshness.ReasonCodes...)
	freshness.EvidencePaths = append([]string{}, freshness.EvidencePaths...)
	manifest.Territory = &freshness
	manifest.TerritoryRequired = true
}

// validatePlanTerritorySnapshot prevents a planning completion from being
// accepted against absent, changed, replayed, or unverified territory input.
func validatePlanTerritorySnapshot(root string, manifest codexPlanManifest) error {
	bound := manifest.Territory
	if !manifest.TerritoryRequired || bound == nil || bound.Freshness != colony.SurveyFreshnessFresh ||
		strings.TrimSpace(bound.SnapshotID) == "" || strings.TrimSpace(bound.SourceRevision) == "" ||
		len(bound.ArtifactDigests) == 0 {
		return fmt.Errorf("plan_manifest requires a verified territory snapshot; rerun `aether plan` to refresh territory before finalizing")
	}
	current := ensureTerritoryFreshness(root)
	if current.Freshness != colony.SurveyFreshnessFresh {
		return fmt.Errorf("plan_manifest territory is no longer fresh (%s: %v); rerun `aether plan`", current.Freshness, current.ReasonCodes)
	}
	if bound.SchemaVersion != current.SchemaVersion || bound.SnapshotID != current.SnapshotID ||
		bound.RepositoryIdentity != current.RepositoryIdentity || !sameCleanPath(bound.RepositoryRoot, current.RepositoryRoot) ||
		bound.SourceRevision != current.SourceRevision || !sameTerritoryDigestSet(bound.ArtifactDigests, current.ArtifactDigests) {
		return fmt.Errorf("plan_manifest territory snapshot does not match the current verified publication; rerun `aether plan`")
	}
	return nil
}

func sameTerritoryDigestSet(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for path, digest := range left {
		if right[path] != digest {
			return false
		}
	}
	return true
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
