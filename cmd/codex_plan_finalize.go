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
	"path"
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
	PlanManifest       *codexPlanManifest           `json:"plan_manifest,omitempty"`
	PlanningManifest   *codexPlanManifest           `json:"planning_manifest,omitempty"`
	Manifest           *codexPlanManifest           `json:"manifest,omitempty"`
	PlanningRunID      string                       `json:"planning_run_id,omitempty"`
	Iteration          int                          `json:"iteration,omitempty"`
	EvidenceHash       string                       `json:"evidence_hash,omitempty"`
	SourceSummary      string                       `json:"source_summary,omitempty"`
	Synthetic          bool                         `json:"synthetic,omitempty"`
	Dispatches         []codexPlanningDispatch      `json:"dispatches,omitempty"`
	Results            []codexPlanningDispatch      `json:"results,omitempty"`
	Workers            []codexPlanningDispatch      `json:"workers,omitempty"`
	ScoutReport        *codexScoutReport            `json:"scout_report,omitempty"`
	PhasePlan          *codexWorkerPlanArtifact     `json:"phase_plan,omitempty"`
	Synthesis          *codexPlanSynthesis          `json:"synthesis,omitempty"`
	ScoutResult        json.RawMessage              `json:"scout_result,omitempty"`
	RouteResult        json.RawMessage              `json:"route_result,omitempty"`
	DecisionResume     *planningDecisionResumeToken `json:"decision_resume_token,omitempty"`
	DecisionResolvedAt time.Time                    `json:"decision_resolved_at,omitempty"`
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

// planningRoutePlanProposal is the only plan-shaped payload Route-Setter may
// author. Authority-bearing Plan and PlanRevision fields are absent by shape;
// Go supplies those later from the current colony state.
type planningRoutePlanProposal struct {
	SemanticID            string                         `json:"semantic_id"`
	Phases                []colony.Phase                 `json:"phases"`
	TaskDeclarations      []planningRouteTaskDeclaration `json:"task_declarations"`
	UserFacingSemanticIDs []string                       `json:"user_facing_semantic_ids"`
	Removals              []planningRouteRemoval         `json:"removals,omitempty"`
}

type planningRouteTaskDeclaration struct {
	TaskSemanticID string   `json:"task_semantic_id"`
	Files          []string `json:"files,omitempty"`
	NoFileReason   string   `json:"no_file_reason,omitempty"`
	UserFacing     bool     `json:"user_facing,omitempty"`
}

type planningRouteRemoval struct {
	SemanticID     string                            `json:"semantic_id"`
	Classification colony.PlanningSemanticChangeKind `json:"classification"`
	Rationale      string                            `json:"rationale"`
}

// planningRouteStageResult deliberately omits stop, candidate, acceptance,
// activation, state-patch, and next-stage fields. Strict JSON decoding makes
// the Route-Setter a proposal producer, never an authority source.
type planningRouteStageResult struct {
	ResultType                 planningStageResultType              `json:"result_type"`
	ManifestID                 string                               `json:"manifest_id"`
	ManifestHash               string                               `json:"manifest_hash"`
	RunID                      string                               `json:"run_id"`
	Pass                       int                                  `json:"pass"`
	Caste                      planningStageWorkerCaste             `json:"caste"`
	Specification              planningStageSpecificationBinding    `json:"specification"`
	BasePlanRevisionID         string                               `json:"base_plan_revision_id"`
	BasePlanRevisionHash       string                               `json:"base_plan_revision_hash"`
	PriorCardHash              string                               `json:"prior_card_hash"`
	InputFrontierHash          string                               `json:"input_frontier_hash"`
	ScoutReceipt               planningStageReceiptRef              `json:"scout_receipt"`
	CandidateSnapshotHash      string                               `json:"candidate_snapshot_hash"`
	Proposal                   planningRoutePlanProposal            `json:"proposal"`
	ProposalEvidenceIDs        []string                             `json:"proposal_evidence_ids"`
	DimensionAssessments       []colony.PlanningDimensionAssessment `json:"dimension_assessments"`
	MaterialDecisionCandidates []planningDecisionCandidate          `json:"material_decision_candidates,omitempty"`
	SuppliedOverall            *int                                 `json:"supplied_overall,omitempty"`
}

type planningRouteStageValidation struct {
	Result           planningRouteStageResult
	Normalized       []byte
	ProposalSnapshot planningSemanticSnapshot
	ProposalHash     string
	PriorSnapshot    planningSemanticSnapshot
	SemanticDelta    colony.PlanningSemanticDelta
	Confidence       planningConfidenceEvaluation
	Evidence         []colony.PlanningEvidenceRef
	EvidenceFrontier planningEvidenceFrontier
	ScoutResult      planningScoutStageResult
	ScoutReceipt     planningStageReceiptRef
	RunHeader        planningRunHeader
}

type planningRouteStageFinalizeOptions struct {
	Fault  lifecycleTransactionFaultHook
	Rename func(oldPath, newPath string) error
}

type planningRouteStageFinalization struct {
	Validation planningRouteStageValidation
	Artifact   planningStageOutputReference
	Receipt    StageReceipt
	Card       colony.PlanningIterationCard
	StopPolicy planningStopPolicyEvaluation
}

// planningRouteScoutDispatch is the host-owned continuation envelope. The
// stage manifest remains the worker authority, while this outer content
// address makes the proposal and completed-card inputs explicit to renderers
// and wrapper dispatchers without widening the Scout result contract.
type planningRouteScoutDispatch struct {
	ID              string                       `json:"id"`
	ContentHash     string                       `json:"content_hash"`
	RunID           string                       `json:"run_id"`
	Pass            int                          `json:"pass"`
	ProposalHash    string                       `json:"proposal_hash"`
	PriorCardHash   string                       `json:"prior_card_hash"`
	Evidence        []colony.PlanningEvidenceRef `json:"evidence"`
	Authorization   planningStageAuthorization   `json:"authorization"`
	Manifest        planningStageManifest        `json:"manifest"`
	ResumeTokenHash string                       `json:"resume_token_hash,omitempty"`
}

type planningRouteStageCoordination struct {
	Route                  planningRouteStageFinalization
	ScoutDispatch          *planningRouteScoutDispatch
	DecisionCheckpoint     *planningScoutDecisionCheckpoint
	ResumeToken            *planningDecisionResumeToken
	SuccessorSpecification *specificationMutationResult
	Candidate              *colony.PlanCandidate
}

type planningScoutStageFinalization struct {
	Result   planningScoutStageResult
	Artifact planningStageOutputReference
	Receipt  StageReceipt
}

// planningScoutDecisionBoundaryToken is the content-addressed authority for
// submitting answers to one exact decision batch. It intentionally carries no
// answers: the completed planningDecisionResumeToken is issued only after the
// owner has answered every card.
type planningScoutDecisionBoundaryToken struct {
	ID                  string `json:"id"`
	ContentHash         string `json:"content_hash"`
	GoalID              string `json:"goal_id"`
	SessionID           string `json:"session_id"`
	BatchID             string `json:"batch_id"`
	BatchHash           string `json:"batch_hash"`
	FrontierReceiptHash string `json:"frontier_receipt_hash,omitempty"`
	CompletedCardHash   string `json:"completed_card_hash,omitempty"`
	RecoveryCommand     string `json:"recovery_command"`
}

type planningScoutDecisionCheckpoint struct {
	ID                    string                             `json:"id"`
	ContentHash           string                             `json:"content_hash"`
	RunID                 string                             `json:"run_id"`
	Pass                  int                                `json:"pass"`
	FrontierReceiptHash   string                             `json:"frontier_receipt_hash,omitempty"`
	CompletedCardHash     string                             `json:"completed_card_hash,omitempty"`
	ProposalHash          string                             `json:"proposal_hash,omitempty"`
	Evidence              []colony.PlanningEvidenceRef       `json:"evidence,omitempty"`
	CandidateSnapshotHash string                             `json:"candidate_snapshot_hash"`
	Scope                 planningDecisionEquivalenceScope   `json:"scope"`
	Batch                 planningDecisionBatch              `json:"batch"`
	Cards                 []planningDecisionCard             `json:"cards"`
	ResumeToken           planningScoutDecisionBoundaryToken `json:"resume_token"`
}

type planningScoutRouteDispatch struct {
	ID                         string                       `json:"id"`
	ContentHash                string                       `json:"content_hash"`
	RunID                      string                       `json:"run_id"`
	Pass                       int                          `json:"pass"`
	ScoutReceipt               planningDecisionStageReceipt `json:"scout_receipt"`
	CandidateSnapshotHash      string                       `json:"candidate_snapshot_hash"`
	MaterialDecisionCandidates []planningDecisionCandidate  `json:"material_decision_candidates,omitempty"`
	Authorization              planningStageAuthorization   `json:"authorization"`
	Manifest                   planningStageManifest        `json:"manifest"`
	ResumeTokenHash            string                       `json:"resume_token_hash,omitempty"`
}

type planningScoutStageCoordination struct {
	Scout                  planningScoutStageFinalization
	DecisionCheckpoint     *planningScoutDecisionCheckpoint
	RouteDispatch          *planningScoutRouteDispatch
	ResumeToken            *planningDecisionResumeToken
	SuccessorSpecification *specificationMutationResult
}

type planningScoutDecisionAnswer struct {
	DecisionID string `json:"decision_id"`
	ChoiceID   string `json:"choice_id"`
	Answer     string `json:"answer"`
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
	if manifest.StageManifest != nil || len(bytes.TrimSpace(completion.ScoutResult)) > 0 || len(bytes.TrimSpace(completion.RouteResult)) > 0 {
		if manifest.StageManifest != nil && manifest.StageManifest.ExpectedCaste == planningStageCasteRouteSetter || len(bytes.TrimSpace(completion.RouteResult)) > 0 {
			return runCodexRouteStageFinalize(root, *manifest, completion)
		}
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
	if len(bytes.TrimSpace(completion.RouteResult)) != 0 {
		return nil, fmt.Errorf("Scout completion cannot include route_result")
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

	coordinated, err := coordinatePlanningScoutStage(root, stageManifest, completion.ScoutResult)
	if err != nil {
		return nil, err
	}
	if completion.DecisionResume != nil {
		resolvedAt := completion.DecisionResolvedAt
		if resolvedAt.IsZero() {
			resolvedAt = time.Now().UTC()
		}
		resumed, resumeErr := resumePlanningScoutDecision(root, stageManifest.RunID, *completion.DecisionResume, resolvedAt)
		if resumeErr != nil {
			return nil, resumeErr
		}
		resumed.Scout = coordinated.Scout
		coordinated = resumed
	}
	evidence := make([]colony.PlanningEvidenceRef, 0, len(coordinated.Scout.Result.NewEvidence))
	for _, record := range coordinated.Scout.Result.NewEvidence {
		evidence = append(evidence, record.Reference)
	}
	state, err := loadPlanningStageState(root, stageManifest.RunID)
	if err != nil {
		return nil, err
	}
	next := "aether plan"
	if state.Stage == planningStageOwnerDecision {
		next = "answer the complete owner decision batch, then rerun `aether plan-finalize` with the exact resume token"
	} else if state.Stage == planningStageRouteRunning {
		next = "dispatch Route-Setter with route_stage_manifest"
	} else if state.Stage == planningStageSpecApprovalRequired {
		next = "review and approve the exact successor specification, then reconcile its affected scope"
	}
	result := map[string]interface{}{
		"planned":                      false,
		"status":                       string(state.Stage),
		"scout_complete":               true,
		"planning_run_id":              stageManifest.RunID,
		"iteration":                    stageManifest.Pass,
		"stage":                        string(state.Stage),
		"next_boundary":                string(state.Stage),
		"stage_receipt":                coordinated.Scout.Receipt,
		"scout_artifact":               coordinated.Scout.Artifact,
		"scout_result":                 coordinated.Scout.Result,
		"evidence_added":               evidence,
		"evidence_added_count":         len(evidence),
		"gaps_found":                   append([]colony.PlanningGap(nil), coordinated.Scout.Result.UnresolvedGaps...),
		"material_decision_candidates": append([]planningDecisionCandidate(nil), coordinated.Scout.Result.DecisionCandidates...),
		"iteration_card_created":       false,
		"next":                         next,
	}
	if coordinated.DecisionCheckpoint != nil {
		result["decision_checkpoint"] = coordinated.DecisionCheckpoint
		result["decision_batch"] = coordinated.DecisionCheckpoint.Batch
		result["decision_cards"] = coordinated.DecisionCheckpoint.Cards
		result["decision_resume_token"] = coordinated.DecisionCheckpoint.ResumeToken
	}
	if coordinated.ResumeToken != nil {
		result["completed_decision_resume_token"] = coordinated.ResumeToken
	}
	if coordinated.RouteDispatch != nil {
		result["route_authorization"] = coordinated.RouteDispatch
		result["route_stage_manifest"] = coordinated.RouteDispatch.Manifest
		result["route_material_decision_candidates"] = coordinated.RouteDispatch.MaterialDecisionCandidates
	}
	if coordinated.SuccessorSpecification != nil {
		result["successor_specification"] = coordinated.SuccessorSpecification
	}
	return result, nil
}

func runCodexRouteStageFinalize(root string, manifest codexPlanManifest, completion codexExternalPlanCompletion) (map[string]interface{}, error) {
	if manifest.StageManifest == nil {
		return nil, fmt.Errorf("Route-Setter completion requires the exact stage_manifest")
	}
	if len(bytes.TrimSpace(completion.RouteResult)) == 0 {
		return nil, fmt.Errorf("Route-Setter completion requires route_result")
	}
	if len(bytes.TrimSpace(completion.ScoutResult)) != 0 || len(completion.workerResults()) != 0 || completion.ScoutReport != nil || completion.PhasePlan != nil || completion.Synthesis != nil {
		return nil, fmt.Errorf("Route-Setter completion cannot include Scout, legacy whole-chain worker, Scout report, phase plan, or synthesis fields")
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
	if stageManifest.ExpectedCaste != planningStageCasteRouteSetter {
		return nil, fmt.Errorf("stage_manifest caste %q is not Route-Setter", stageManifest.ExpectedCaste)
	}
	if manifest.PlanningRunID != stageManifest.RunID || manifest.Iteration != stageManifest.Pass || manifest.SelectedPreset != stageManifest.Preset ||
		manifest.BaseRevisionID != stageManifest.BasePlanRevisionID || manifest.BasePlanStateHash != stageManifest.BasePlanRevisionHash {
		return nil, fmt.Errorf("plan_manifest does not match the exact Route-Setter stage run, pass, preset, or base plan revision")
	}
	if len(manifest.Dispatches) != 1 || len(manifest.ExpectedWorkers) != 1 {
		return nil, fmt.Errorf("Route-Setter plan_manifest must contain exactly one Route-Setter dispatch")
	}
	for _, dispatch := range []codexPlanningDispatch{manifest.Dispatches[0], manifest.ExpectedWorkers[0]} {
		if !strings.EqualFold(dispatch.Caste, string(planningStageCasteRouteSetter)) || dispatch.StageManifest == nil ||
			dispatch.StageManifest.ID != stageManifest.ID || dispatch.StageManifest.ContentHash != stageManifest.ContentHash {
			return nil, fmt.Errorf("Route-Setter dispatch does not bind the exact stage_manifest")
		}
	}

	coordinated, err := coordinatePlanningRouteStage(root, stageManifest, completion.RouteResult)
	if err != nil {
		return nil, err
	}
	if completion.DecisionResume != nil {
		resolvedAt := completion.DecisionResolvedAt
		if resolvedAt.IsZero() {
			resolvedAt = time.Now().UTC()
		}
		resumed, resumeErr := resumePlanningRouteDecision(root, stageManifest.RunID, *completion.DecisionResume, resolvedAt)
		if resumeErr != nil {
			return nil, resumeErr
		}
		resumed.Route = coordinated.Route
		coordinated = resumed
	}
	state, err := loadPlanningStageState(root, stageManifest.RunID)
	if err != nil {
		return nil, err
	}
	next := "aether plan"
	switch state.Stage {
	case planningStageScoutRunning:
		next = "dispatch Scout with scout_stage_manifest"
	case planningStageOwnerDecision:
		next = "answer the complete owner decision batch, then rerun `aether plan-finalize` with the exact resume token"
	case planningStageSpecApprovalRequired:
		next = "review and approve the exact successor specification, then reconcile its affected scope"
	case planningStageCandidateReady:
		next = "aether plan --candidate"
	}
	result := map[string]interface{}{
		"planned":                false,
		"status":                 string(state.Stage),
		"planning_run_id":        stageManifest.RunID,
		"iteration":              stageManifest.Pass,
		"stage":                  string(state.Stage),
		"next_boundary":          string(state.Stage),
		"route_complete":         true,
		"route_stage_receipt":    coordinated.Route.Receipt,
		"route_artifact":         coordinated.Route.Artifact,
		"route_result":           coordinated.Route.Validation.Result,
		"proposal_hash":          coordinated.Route.Validation.ProposalHash,
		"iteration_card":         coordinated.Route.Card,
		"iteration_card_created": true,
		"stop_policy":            coordinated.Route.StopPolicy,
		"next":                   next,
	}
	if coordinated.ScoutDispatch != nil {
		result["scout_authorization"] = coordinated.ScoutDispatch
		result["scout_stage_manifest"] = coordinated.ScoutDispatch.Manifest
	}
	if coordinated.DecisionCheckpoint != nil {
		result["decision_checkpoint"] = coordinated.DecisionCheckpoint
		result["decision_batch"] = coordinated.DecisionCheckpoint.Batch
		result["decision_cards"] = coordinated.DecisionCheckpoint.Cards
		result["decision_resume_token"] = coordinated.DecisionCheckpoint.ResumeToken
	}
	if coordinated.ResumeToken != nil {
		result["completed_decision_resume_token"] = coordinated.ResumeToken
	}
	if coordinated.SuccessorSpecification != nil {
		result["successor_specification"] = coordinated.SuccessorSpecification
	}
	if coordinated.Candidate != nil {
		result["plan_candidate"] = coordinated.Candidate
	}
	return result, nil
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
	candidateSnapshotHash, err := planningScoutCandidateSnapshotHash(manifest, result.DecisionCandidates)
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

// coordinatePlanningScoutStage advances only the boundary made legal by the
// completed Scout receipt. Pass one either persists the complete owner batch
// or dispatches Route-Setter; later passes always dispatch Route-Setter and
// carry their material discoveries with that exact authorization.
func coordinatePlanningScoutStage(root string, manifest planningStageManifest, raw []byte) (planningScoutStageCoordination, error) {
	empty := planningScoutStageCoordination{}
	completed, err := finalizePlanningScoutStage(root, manifest, raw)
	if err != nil {
		return empty, err
	}
	result := planningScoutStageCoordination{Scout: completed}
	state, err := loadPlanningStageState(root, manifest.RunID)
	if err != nil {
		return empty, err
	}

	switch state.Stage {
	case planningStageOwnerDecision:
		checkpoint, checkpointErr := buildPlanningScoutDecisionCheckpoint(root, manifest, completed)
		if checkpointErr != nil {
			return empty, checkpointErr
		}
		if checkpoint == nil {
			return empty, fmt.Errorf("owner_decision has no material Scout decision batch")
		}
		if err := persistPlanningScoutDecisionCheckpoint(root, *checkpoint); err != nil {
			return empty, err
		}
		result.DecisionCheckpoint = checkpoint
		return result, nil

	case planningStageRouteReady:
		dispatch, dispatchErr := authorizePlanningScoutRoute(root, state, completed.Result.DecisionCandidates, nil)
		if dispatchErr != nil {
			return empty, dispatchErr
		}
		result.RouteDispatch = &dispatch
		return result, nil

	case planningStageRouteRunning:
		dispatch, dispatchErr := loadPlanningScoutRouteDispatch(root, manifest.RunID, manifest.Pass)
		if dispatchErr != nil {
			return empty, dispatchErr
		}
		result.RouteDispatch = &dispatch
		return result, nil

	case planningStageSpecApprovalRequired, planningStageReconciliationRequired:
		checkpoint, checkpointErr := loadPlanningScoutDecisionCheckpoint(root, manifest.RunID)
		if checkpointErr != nil {
			return empty, checkpointErr
		}
		result.DecisionCheckpoint = &checkpoint
		return result, nil

	default:
		return empty, fmt.Errorf("completed Scout reached unexpected planning stage %q", state.Stage)
	}
}

func buildPlanningScoutDecisionCheckpoint(root string, manifest planningStageManifest, completed planningScoutStageFinalization) (*planningScoutDecisionCheckpoint, error) {
	header, err := loadPlanningScoutRunHeader(root, manifest)
	if err != nil {
		return nil, err
	}
	boundary := planningDecisionBoundary{
		RunID: manifest.RunID,
		Pass:  manifest.Pass,
		ScoutReceipt: planningDecisionStageReceipt{
			ID:          completed.Receipt.ID,
			ContentHash: completed.Receipt.ContentHash,
			RunID:       completed.Receipt.RunID,
			Pass:        completed.Receipt.Pass,
		},
		RecoveryCommand: "aether plan",
	}
	batch, err := buildPlanningDecisionBatch(boundary, completed.Result.DecisionCandidates)
	if err != nil {
		return nil, fmt.Errorf("build first-pass Scout decision batch: %w", err)
	}
	if batch == nil {
		return nil, nil
	}
	scope := planningDecisionEquivalenceScope{
		GoalID:                          header.GoalID,
		SessionID:                       header.SessionID,
		ApprovedSpecificationRevisionID: manifest.Specification.RevisionID,
		BasePlanRevisionID:              manifest.BasePlanRevisionID,
	}
	cards := make([]planningDecisionCard, 0, len(batch.Decisions))
	for _, candidate := range batch.Decisions {
		card, cardErr := projectPlanningDecisionCard(planningDecisionCardRequest{Candidate: candidate, Scope: scope})
		if cardErr != nil {
			return nil, fmt.Errorf("project Scout decision card %q: %w", candidate.StableID, cardErr)
		}
		cards = append(cards, card)
	}
	tokenPayload := struct {
		GoalID              string `json:"goal_id"`
		SessionID           string `json:"session_id"`
		BatchID             string `json:"batch_id"`
		BatchHash           string `json:"batch_hash"`
		FrontierReceiptHash string `json:"frontier_receipt_hash,omitempty"`
		CompletedCardHash   string `json:"completed_card_hash,omitempty"`
		RecoveryCommand     string `json:"recovery_command"`
	}{scope.GoalID, scope.SessionID, batch.ID, batch.ContentHash, completed.Receipt.ContentHash, "", batch.RecoveryCommand}
	tokenHash, err := jsonSHA256(tokenPayload)
	if err != nil {
		return nil, fmt.Errorf("hash Scout decision boundary token: %w", err)
	}
	boundaryToken := planningScoutDecisionBoundaryToken{
		ID:                  "planning-decision-boundary-" + tokenHash[:16],
		ContentHash:         tokenHash,
		GoalID:              tokenPayload.GoalID,
		SessionID:           tokenPayload.SessionID,
		BatchID:             tokenPayload.BatchID,
		BatchHash:           tokenPayload.BatchHash,
		FrontierReceiptHash: tokenPayload.FrontierReceiptHash,
		CompletedCardHash:   tokenPayload.CompletedCardHash,
		RecoveryCommand:     tokenPayload.RecoveryCommand,
	}
	checkpoint := planningScoutDecisionCheckpoint{
		RunID:                 manifest.RunID,
		Pass:                  manifest.Pass,
		FrontierReceiptHash:   completed.Receipt.ContentHash,
		CompletedCardHash:     "",
		CandidateSnapshotHash: completed.Receipt.CandidateSnapshotHash,
		Scope:                 scope,
		Batch:                 *batch,
		Cards:                 cards,
		ResumeToken:           boundaryToken,
	}
	if err := addressPlanningScoutDecisionCheckpoint(&checkpoint); err != nil {
		return nil, err
	}
	return &checkpoint, nil
}

func addressPlanningScoutDecisionCheckpoint(checkpoint *planningScoutDecisionCheckpoint) error {
	if checkpoint == nil {
		return fmt.Errorf("Scout decision checkpoint is required")
	}
	payload := *checkpoint
	payload.ID = ""
	payload.ContentHash = ""
	contentHash, err := jsonSHA256(payload)
	if err != nil {
		return fmt.Errorf("hash Scout decision checkpoint: %w", err)
	}
	checkpoint.ContentHash = contentHash
	checkpoint.ID = "planning-scout-decision-" + contentHash[:16]
	return nil
}

func planningScoutDecisionCheckpointRepositoryPath(runID string) string {
	return path.Join(".aether", "data", "planning", strings.TrimSpace(runID), "decisions", "first-pass.json")
}

func persistPlanningScoutDecisionCheckpoint(root string, checkpoint planningScoutDecisionCheckpoint) error {
	content, err := marshalPlanningStageJSON(checkpoint)
	if err != nil {
		return fmt.Errorf("marshal Scout decision checkpoint: %w", err)
	}
	repositoryPath := planningScoutDecisionCheckpointRepositoryPath(checkpoint.RunID)
	return persistPlanningScoutFiles(root, "planning-scout-decision-"+checkpoint.ContentHash[:24], "planning-scout-decision", checkpoint.ID, map[string][]byte{
		repositoryPath: content,
	}, nil)
}

func loadPlanningScoutDecisionCheckpoint(root, runID string) (planningScoutDecisionCheckpoint, error) {
	var checkpoint planningScoutDecisionCheckpoint
	repositoryRoot, _, err := planningStageRoots(root)
	if err != nil {
		return checkpoint, err
	}
	content, exists, err := readOptionalPlanningStageFile(repositoryRoot, planningScoutDecisionCheckpointRepositoryPath(runID))
	if err != nil {
		return checkpoint, err
	}
	if !exists {
		return checkpoint, fmt.Errorf("Scout decision checkpoint for %q is missing: %w", runID, os.ErrNotExist)
	}
	if err := decodePlanningStageJSON(content, &checkpoint); err != nil {
		return planningScoutDecisionCheckpoint{}, fmt.Errorf("decode Scout decision checkpoint: %w", err)
	}
	originalID, originalHash := checkpoint.ID, checkpoint.ContentHash
	if err := addressPlanningScoutDecisionCheckpoint(&checkpoint); err != nil {
		return planningScoutDecisionCheckpoint{}, err
	}
	if checkpoint.ID != originalID || checkpoint.ContentHash != originalHash || checkpoint.RunID != strings.TrimSpace(runID) {
		return planningScoutDecisionCheckpoint{}, fmt.Errorf("Scout decision checkpoint is not a valid content address")
	}
	return checkpoint, nil
}

func authorizePlanningScoutRoute(root string, state planningStageState, candidates []planningDecisionCandidate, token *planningDecisionResumeToken) (planningScoutRouteDispatch, error) {
	empty := planningScoutRouteDispatch{}
	if state.Stage == planningStageRouteRunning {
		return loadPlanningScoutRouteDispatch(root, state.RunID, state.Pass)
	}
	if state.Stage != planningStageRouteReady || state.ScoutReceipt == nil {
		return empty, fmt.Errorf("Route-Setter authorization requires route_ready with the exact Scout receipt")
	}
	candidates = canonicalPlanningScoutCandidates(candidates)
	seed, err := jsonSHA256(struct {
		RunID                 string                      `json:"run_id"`
		Pass                  int                         `json:"pass"`
		InputFrontierHash     string                      `json:"input_frontier_hash"`
		ScoutReceipt          *planningStageReceiptRef    `json:"scout_receipt"`
		CandidateSnapshotHash string                      `json:"candidate_snapshot_hash"`
		Candidates            []planningDecisionCandidate `json:"material_decision_candidates,omitempty"`
	}{state.RunID, state.Pass, state.InputFrontierHash, state.ScoutReceipt, state.CandidateSnapshotHash, candidates})
	if err != nil {
		return empty, fmt.Errorf("hash Route-Setter authorization: %w", err)
	}
	authorization := planningStageAuthorization{
		ID:                    "planning-authorization-" + seed[:16],
		ExpectedCaste:         planningStageCasteRouteSetter,
		InputFrontierHash:     state.InputFrontierHash,
		ScoutReceipt:          clonePlanningStageReceiptRef(state.ScoutReceipt),
		CandidateSnapshotHash: state.CandidateSnapshotHash,
	}
	running, manifest, err := reducePlanningStage(state, planningStageTransition{To: planningStageRouteRunning, Authorization: &authorization})
	if err != nil {
		return empty, fmt.Errorf("authorize Route-Setter after Scout: %w", err)
	}
	if manifest == nil {
		return empty, fmt.Errorf("Route-Setter authorization emitted no manifest")
	}
	resumeHash := ""
	if token != nil {
		resumeHash = token.ContentHash
	}
	dispatch := planningScoutRouteDispatch{
		RunID:                      state.RunID,
		Pass:                       state.Pass,
		ScoutReceipt:               planningDecisionStageReceipt{ID: state.ScoutReceipt.ID, ContentHash: state.ScoutReceipt.ContentHash, RunID: state.ScoutReceipt.RunID, Pass: state.ScoutReceipt.Pass},
		CandidateSnapshotHash:      state.CandidateSnapshotHash,
		MaterialDecisionCandidates: candidates,
		Authorization:              authorization,
		Manifest:                   *manifest,
		ResumeTokenHash:            resumeHash,
	}
	if err := addressPlanningScoutRouteDispatch(&dispatch); err != nil {
		return empty, err
	}
	if err := persistPlanningScoutRouteDispatch(root, running, dispatch, token); err != nil {
		return empty, err
	}
	return dispatch, nil
}

func canonicalPlanningScoutCandidates(candidates []planningDecisionCandidate) []planningDecisionCandidate {
	result := make([]planningDecisionCandidate, len(candidates))
	for index := range candidates {
		result[index] = canonicalPlanningDecisionCandidate(candidates[index])
	}
	sort.Slice(result, func(left, right int) bool { return result[left].StableID < result[right].StableID })
	return result
}

func addressPlanningScoutRouteDispatch(dispatch *planningScoutRouteDispatch) error {
	if dispatch == nil {
		return fmt.Errorf("Scout Route-Setter dispatch is required")
	}
	payload := *dispatch
	payload.ID = ""
	payload.ContentHash = ""
	contentHash, err := jsonSHA256(payload)
	if err != nil {
		return fmt.Errorf("hash Scout Route-Setter dispatch: %w", err)
	}
	dispatch.ContentHash = contentHash
	dispatch.ID = "planning-scout-route-" + contentHash[:16]
	return nil
}

func planningScoutRouteDispatchRepositoryPath(runID string, pass int) string {
	return path.Join(".aether", "data", "planning", strings.TrimSpace(runID), "route-authorizations", fmt.Sprintf("pass-%04d.json", pass))
}

func planningScoutDecisionResumeRepositoryPath(runID string) string {
	return path.Join(".aether", "data", "planning", strings.TrimSpace(runID), "decisions", "completed-resume-token.json")
}

func persistPlanningScoutRouteDispatch(root string, running planningStageState, dispatch planningScoutRouteDispatch, token *planningDecisionResumeToken) error {
	if err := validatePlanningStageManifest(dispatch.Manifest); err != nil {
		return err
	}
	if err := validatePlanningStageRunningState(running, dispatch.Manifest); err != nil {
		return err
	}
	dispatchBytes, err := marshalPlanningStageJSON(dispatch)
	if err != nil {
		return err
	}
	manifestBytes, err := marshalPlanningStageJSON(dispatch.Manifest)
	if err != nil {
		return err
	}
	stateBytes, err := marshalPlanningStageJSON(running)
	if err != nil {
		return err
	}
	files := map[string][]byte{
		planningScoutRouteDispatchRepositoryPath(dispatch.RunID, dispatch.Pass):   dispatchBytes,
		planningStageManifestRepositoryPath(dispatch.RunID, dispatch.Manifest.ID): manifestBytes,
		planningStageStateRepositoryPath(dispatch.RunID):                          stateBytes,
	}
	if token != nil {
		tokenBytes, marshalErr := marshalPlanningStageJSON(token)
		if marshalErr != nil {
			return marshalErr
		}
		files[planningScoutDecisionResumeRepositoryPath(dispatch.RunID)] = tokenBytes
	}
	return persistPlanningScoutFiles(root, "planning-scout-route-"+dispatch.ContentHash[:24], "planning-scout-route", dispatch.ID, files, map[string]bool{
		planningStageStateRepositoryPath(dispatch.RunID): true,
	})
}

func loadPlanningScoutRouteDispatch(root, runID string, pass int) (planningScoutRouteDispatch, error) {
	var dispatch planningScoutRouteDispatch
	repositoryRoot, _, err := planningStageRoots(root)
	if err != nil {
		return dispatch, err
	}
	content, exists, err := readOptionalPlanningStageFile(repositoryRoot, planningScoutRouteDispatchRepositoryPath(runID, pass))
	if err != nil {
		return dispatch, err
	}
	if !exists {
		return dispatch, fmt.Errorf("Route-Setter authorization for run %q pass %d is missing: %w", runID, pass, os.ErrNotExist)
	}
	if err := decodePlanningStageJSON(content, &dispatch); err != nil {
		return planningScoutRouteDispatch{}, fmt.Errorf("decode Scout Route-Setter dispatch: %w", err)
	}
	originalID, originalHash := dispatch.ID, dispatch.ContentHash
	if err := addressPlanningScoutRouteDispatch(&dispatch); err != nil {
		return planningScoutRouteDispatch{}, err
	}
	if dispatch.ID != originalID || dispatch.ContentHash != originalHash || dispatch.RunID != strings.TrimSpace(runID) || dispatch.Pass != pass {
		return planningScoutRouteDispatch{}, fmt.Errorf("Scout Route-Setter dispatch is not a valid content address")
	}
	return dispatch, nil
}

func persistPlanningScoutFiles(root, transactionID, command, identity string, files map[string][]byte, replacePaths map[string]bool) error {
	repositoryRoot, dataRoot, err := planningStageRoots(root)
	if err != nil {
		return err
	}
	allPresent := true
	for repositoryPath, expected := range files {
		current, exists, readErr := readOptionalPlanningStageFile(repositoryRoot, repositoryPath)
		if readErr != nil {
			return readErr
		}
		if exists && !bytes.Equal(current, expected) && !replacePaths[repositoryPath] {
			return &planningStageReceiptConflictError{ManifestID: identity, Detail: "persisted Scout boundary bytes differ"}
		}
		allPresent = allPresent && exists && bytes.Equal(current, expected)
	}
	if allPresent {
		return nil
	}
	config := planningStageWriteConfig(repositoryRoot, dataRoot, transactionID, command, planningStageWriteOptions{})
	expected := make(map[string][]byte, len(files))
	for repositoryPath, content := range files {
		expected[planningStageDataRelativePath(repositoryPath)] = content
	}
	if resumed, resumeErr := resumePlanningStageWrite(config, expected, identity); resumeErr != nil {
		return resumeErr
	} else if resumed {
		return nil
	}
	tx, err := beginLifecycleTransaction(config)
	if err != nil {
		return err
	}
	paths := make([]string, 0, len(files))
	for repositoryPath := range files {
		paths = append(paths, repositoryPath)
	}
	sort.Strings(paths)
	for _, repositoryPath := range paths {
		if err := tx.DeclareWrite(lifecycleTransactionRootData, planningStageDataRelativePath(repositoryPath), files[repositoryPath]); err != nil {
			return err
		}
	}
	if err := tx.Validate(); err != nil {
		return err
	}
	_, err = tx.Commit()
	return err
}

func buildPlanningScoutDecisionResumeToken(checkpoint planningScoutDecisionCheckpoint, answers []planningScoutDecisionAnswer) (planningDecisionResumeToken, error) {
	if len(answers) != len(checkpoint.Cards) || len(checkpoint.Cards) != len(checkpoint.Batch.Decisions) {
		return planningDecisionResumeToken{}, fmt.Errorf("completed Scout decision answers must cover every card exactly once")
	}
	candidates := make(map[string]planningDecisionCandidate, len(checkpoint.Batch.Decisions))
	for _, candidate := range checkpoint.Batch.Decisions {
		candidates[candidate.StableID] = candidate
	}
	cards := make(map[string]planningDecisionCard, len(checkpoint.Cards))
	for _, card := range checkpoint.Cards {
		cards[card.DecisionID] = card
	}

	boundAnswers := make([]planningDecisionBoundAnswer, 0, len(answers))
	affectedIDs := make([]string, 0)
	revisionEvidence := make([]planningDecisionRevisionEvidence, 0)
	disposition := planningDecisionDispositionDirectResume
	seen := make(map[string]struct{}, len(answers))
	for _, submitted := range answers {
		decisionID := strings.TrimSpace(submitted.DecisionID)
		if _, duplicate := seen[decisionID]; duplicate {
			return planningDecisionResumeToken{}, fmt.Errorf("duplicate completed answer for decision %q", decisionID)
		}
		seen[decisionID] = struct{}{}
		card, ok := cards[decisionID]
		if !ok {
			return planningDecisionResumeToken{}, fmt.Errorf("completed answer names unknown decision %q", decisionID)
		}
		candidate, ok := candidates[decisionID]
		if !ok {
			return planningDecisionResumeToken{}, fmt.Errorf("decision card %q is absent from its exact batch", decisionID)
		}
		var selected *planningDecisionChoice
		for index := range card.Choices {
			if card.Choices[index].ID == strings.TrimSpace(submitted.ChoiceID) {
				choice := card.Choices[index]
				selected = &choice
				break
			}
		}
		if selected == nil {
			return planningDecisionResumeToken{}, fmt.Errorf("completed answer for %q names unknown choice %q", decisionID, submitted.ChoiceID)
		}
		answer := normalizePlanningDecisionText(submitted.Answer)
		if answer == "" {
			return planningDecisionResumeToken{}, fmt.Errorf("completed answer for %q requires answer text", decisionID)
		}
		resolution, err := resolvePlanningDecisionAnswer(planningDecisionResolutionRequest{
			DecisionID:     decisionID,
			ApprovedImpact: candidate.Impact,
			SelectedChoice: *selected,
		})
		if err != nil {
			return planningDecisionResumeToken{}, fmt.Errorf("resolve completed answer %q: %w", decisionID, err)
		}
		if resolution.Disposition == planningDecisionDispositionSuccessorSpecRequired {
			disposition = planningDecisionDispositionSuccessorSpecRequired
			affectedIDs = append(affectedIDs, resolution.AffectedSemanticIDs...)
			for _, evidence := range resolution.RevisionEvidence {
				evidence.Dimension = decisionID + ":" + evidence.Dimension
				revisionEvidence = append(revisionEvidence, evidence)
			}
		}
		boundAnswers = append(boundAnswers, planningDecisionBoundAnswer{
			DecisionID:     decisionID,
			ChoiceID:       selected.ID,
			Answer:         answer,
			EquivalenceKey: card.EquivalenceKey,
		})
	}
	if len(seen) != len(cards) {
		return planningDecisionResumeToken{}, fmt.Errorf("completed Scout decision answers are partial")
	}
	return issuePlanningDecisionResumeToken(planningDecisionResumeBinding{
		GoalID:              checkpoint.Scope.GoalID,
		SessionID:           checkpoint.Scope.SessionID,
		BatchID:             checkpoint.Batch.ID,
		BatchHash:           checkpoint.Batch.ContentHash,
		FrontierReceiptHash: checkpoint.FrontierReceiptHash,
		CompletedCardHash:   checkpoint.CompletedCardHash,
		Answers:             boundAnswers,
		Disposition:         disposition,
		AffectedSemanticIDs: nonEmptyPlanningDecisionIDs(affectedIDs),
		RevisionEvidence:    revisionEvidence,
		RecoveryCommand:     checkpoint.Batch.RecoveryCommand,
	})
}

func resumePlanningScoutDecision(root, runID string, token planningDecisionResumeToken, resolvedAt time.Time) (planningScoutStageCoordination, error) {
	empty := planningScoutStageCoordination{}
	checkpoint, err := loadPlanningScoutDecisionCheckpoint(root, runID)
	if err != nil {
		return empty, err
	}
	answers := make([]planningScoutDecisionAnswer, 0, len(token.Answers))
	for _, answer := range token.Answers {
		answers = append(answers, planningScoutDecisionAnswer{DecisionID: answer.DecisionID, ChoiceID: answer.ChoiceID, Answer: answer.Answer})
	}
	expectedToken, err := buildPlanningScoutDecisionResumeToken(checkpoint, answers)
	if err != nil {
		return empty, err
	}
	validation, err := validatePlanningDecisionResumeToken(token, planningDecisionResumeBindingFromToken(expectedToken))
	if err != nil {
		return empty, err
	}
	if !validation.Accepted {
		return empty, fmt.Errorf("Scout decision resume token is %s; recover with %s", validation.Status, validation.RecoveryCommand)
	}
	state, err := loadPlanningStageState(root, runID)
	if err != nil {
		return empty, err
	}
	result := planningScoutStageCoordination{DecisionCheckpoint: &checkpoint, ResumeToken: &expectedToken}
	if state.Stage == planningStageRouteRunning {
		dispatch, loadErr := loadPlanningScoutRouteDispatch(root, runID, state.Pass)
		if loadErr != nil {
			return empty, loadErr
		}
		if dispatch.ResumeTokenHash != expectedToken.ContentHash {
			return empty, fmt.Errorf("Route-Setter already resumed from a different decision token")
		}
		result.RouteDispatch = &dispatch
		return result, nil
	}
	if state.Stage == planningStageSpecApprovalRequired && expectedToken.Disposition == planningDecisionDispositionSuccessorSpecRequired {
		colonyState, loadErr := loadSpecificationColonyState(root)
		if loadErr != nil {
			return empty, loadErr
		}
		if state.PendingSpecification == nil || colonyState.Specification == nil {
			return empty, fmt.Errorf("successor specification checkpoint is incomplete")
		}
		current, ok := currentSpecificationRevision(*colonyState.Specification)
		if !ok || current.ID != state.PendingSpecification.RevisionID || current.ContentHash != state.PendingSpecification.ContentHash || current.Status != colony.SpecStatusDraft {
			return empty, fmt.Errorf("persisted successor specification does not match the exact planning checkpoint")
		}
		result.SuccessorSpecification = &specificationMutationResult{Specification: *colonyState.Specification, Revision: current}
		return result, nil
	}
	if state.Stage != planningStageOwnerDecision || state.ScoutReceipt == nil || state.ScoutReceipt.ContentHash != checkpoint.FrontierReceiptHash {
		return empty, fmt.Errorf("Scout decision token does not match the current owner_decision frontier")
	}
	resolution := planningDecisionResolution{
		Disposition:         expectedToken.Disposition,
		AffectedSemanticIDs: append([]string(nil), expectedToken.AffectedSemanticIDs...),
		RevisionEvidence:    append([]planningDecisionRevisionEvidence(nil), expectedToken.RevisionEvidence...),
	}
	if resolution.Disposition == planningDecisionDispositionSuccessorSpecRequired {
		return resumePlanningScoutContractDecision(root, state, checkpoint, expectedToken, resolution, resolvedAt)
	}
	ready, manifest, err := reducePlanningStage(state, planningStageTransition{
		To:                 planningStageRouteReady,
		DecisionResolution: &resolution,
	})
	if err != nil {
		return empty, fmt.Errorf("resume planning after direct owner answer: %w", err)
	}
	if manifest != nil {
		return empty, fmt.Errorf("direct owner answer unexpectedly dispatched a worker")
	}
	dispatch, err := authorizePlanningScoutRoute(root, ready, checkpoint.Batch.Decisions, &expectedToken)
	if err != nil {
		return empty, err
	}
	result.RouteDispatch = &dispatch
	return result, nil
}

func resumePlanningScoutContractDecision(root string, state planningStageState, checkpoint planningScoutDecisionCheckpoint, token planningDecisionResumeToken, resolution planningDecisionResolution, resolvedAt time.Time) (planningScoutStageCoordination, error) {
	empty := planningScoutStageCoordination{}
	if resolvedAt.IsZero() {
		return empty, fmt.Errorf("contract-affecting Scout answer requires resolved_at")
	}
	colonyState, err := loadSpecificationColonyState(root)
	if err != nil {
		return empty, err
	}
	if colonyState.Specification == nil {
		return empty, fmt.Errorf("contract-affecting Scout answer requires the approved specification lineage")
	}
	predecessorIndex := specificationRevisionIndex(*colonyState.Specification, state.Specification.RevisionID)
	if predecessorIndex < 0 {
		return empty, fmt.Errorf("contract-affecting Scout answer predecessor is absent from the specification lineage")
	}
	predecessor := colonyState.Specification.Revisions[predecessorIndex]
	if predecessor.ContentHash != state.Specification.ContentHash || (predecessor.Status != colony.SpecStatusApproved && predecessor.Status != colony.SpecStatusSuperseded) || predecessor.Approval == nil {
		return empty, fmt.Errorf("contract-affecting Scout answer does not bind the current approved specification")
	}
	changes, err := planningScoutSpecificationChanges(predecessor, checkpoint, token)
	if err != nil {
		return empty, err
	}
	successor, err := reviseSpecification(root, specificationRevisionRequest{
		PredecessorRevisionID:  predecessor.ID,
		PredecessorContentHash: predecessor.ContentHash,
		Scope:                  predecessor.Scope,
		Changes:                changes,
		DecisionResolution:     &resolution,
		CreatedAt:              resolvedAt.UTC(),
	}, specificationMutationOptions{})
	if err != nil {
		return empty, fmt.Errorf("create successor specification for Scout decision: %w", err)
	}
	draftBinding := planningStageSpecificationBinding{
		RevisionID:            successor.Revision.ID,
		ContentHash:           successor.Revision.ContentHash,
		PredecessorRevisionID: predecessor.ID,
		Status:                colony.SpecStatusDraft,
	}
	waiting, manifest, err := reducePlanningStage(state, planningStageTransition{
		To:                     planningStageSpecApprovalRequired,
		DecisionResolution:     &resolution,
		SuccessorSpecification: &draftBinding,
	})
	if err != nil {
		return empty, fmt.Errorf("record successor specification boundary: %w", err)
	}
	if manifest != nil {
		return empty, fmt.Errorf("successor specification boundary unexpectedly dispatched a worker")
	}
	stateBytes, err := marshalPlanningStageJSON(waiting)
	if err != nil {
		return empty, err
	}
	tokenBytes, err := marshalPlanningStageJSON(token)
	if err != nil {
		return empty, err
	}
	if err := persistPlanningScoutFiles(root, "planning-scout-successor-"+successor.Revision.ContentHash[:24], "planning-scout-successor", successor.Revision.ID, map[string][]byte{
		planningStageStateRepositoryPath(state.RunID):          stateBytes,
		planningScoutDecisionResumeRepositoryPath(state.RunID): tokenBytes,
	}, map[string]bool{planningStageStateRepositoryPath(state.RunID): true}); err != nil {
		return empty, err
	}
	return planningScoutStageCoordination{
		DecisionCheckpoint:     &checkpoint,
		ResumeToken:            &token,
		SuccessorSpecification: &successor,
	}, nil
}

func planningScoutSpecificationChanges(revision colony.SpecRevision, checkpoint planningScoutDecisionCheckpoint, token planningDecisionResumeToken) ([]specificationRevisionChange, error) {
	body := specificationBodyFromRevision(revision)
	candidates := make(map[string]planningDecisionCandidate, len(checkpoint.Batch.Decisions))
	for _, candidate := range checkpoint.Batch.Decisions {
		candidates[candidate.StableID] = candidate
	}
	type directive struct {
		Text       string
		EvidenceID string
	}
	directives := make(map[string][]directive)
	for _, answer := range token.Answers {
		candidate, ok := candidates[answer.DecisionID]
		if !ok {
			return nil, fmt.Errorf("successor answer %q is absent from the decision batch", answer.DecisionID)
		}
		var selected *planningDecisionChoice
		for index := range candidate.Choices {
			if candidate.Choices[index].ID == answer.ChoiceID {
				choice := canonicalPlanningDecisionChoice(candidate.Choices[index])
				selected = &choice
				break
			}
		}
		if selected == nil {
			return nil, fmt.Errorf("successor answer %q names an unknown choice", answer.DecisionID)
		}
		for _, semanticID := range selected.AffectedSemanticIDs {
			directives[semanticID] = append(directives[semanticID], directive{
				Text:       fmt.Sprintf("Owner decision %s: %s Consequence: %s", answer.DecisionID, answer.Answer, selected.Consequence),
				EvidenceID: "owner-decision:" + answer.DecisionID + ":" + answer.ChoiceID,
			})
		}
	}

	affected := nonEmptyPlanningDecisionIDs(token.AffectedSemanticIDs)
	changes := make([]specificationRevisionChange, 0, len(affected))
	for _, semanticID := range affected {
		section, item, found := planningScoutSpecificationItem(body, semanticID)
		if !found {
			return nil, fmt.Errorf("contract-affecting Scout answer targets unknown specification item %q", semanticID)
		}
		itemDirectives := directives[semanticID]
		if len(itemDirectives) == 0 {
			return nil, fmt.Errorf("contract-affecting Scout answer has no exact choice for specification item %q", semanticID)
		}
		sort.Slice(itemDirectives, func(left, right int) bool { return itemDirectives[left].Text < itemDirectives[right].Text })
		descriptionParts := []string{strings.TrimSpace(item.Description)}
		evidenceIDs := append([]string(nil), item.EvidenceIDs...)
		for _, itemDirective := range itemDirectives {
			descriptionParts = append(descriptionParts, itemDirective.Text)
			evidenceIDs = append(evidenceIDs, itemDirective.EvidenceID)
		}
		changes = append(changes, specificationRevisionChange{
			Operation: specificationChangeModify,
			Section:   section,
			TargetID:  semanticID,
			Item: specificationItemInput{
				Description:  strings.Join(descriptionParts, " "),
				Verification: item.Verification,
				Path:         item.Path,
				EvidenceIDs:  nonEmptyPlanningDecisionIDs(evidenceIDs),
			},
		})
	}
	return changes, nil
}

func planningScoutSpecificationItem(body specificationBodySnapshot, stableID string) (specificationBodySection, specificationCanonicalItem, bool) {
	for _, section := range specificationBodyOrder {
		for _, item := range body[section] {
			if item.ID == stableID {
				return section, item, true
			}
		}
	}
	return "", specificationCanonicalItem{}, false
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

func planningScoutCandidateSnapshotHash(manifest planningStageManifest, candidates []planningDecisionCandidate) (string, error) {
	return jsonSHA256(struct {
		RunID                string                      `json:"run_id"`
		Pass                 int                         `json:"pass"`
		BasePlanRevisionID   string                      `json:"base_plan_revision_id"`
		BasePlanRevisionHash string                      `json:"base_plan_revision_hash"`
		PriorCardHash        string                      `json:"prior_card_hash"`
		InputFrontierHash    string                      `json:"input_frontier_hash"`
		Candidates           []planningDecisionCandidate `json:"material_decision_candidates,omitempty"`
	}{manifest.RunID, manifest.Pass, manifest.BasePlanRevisionID, manifest.BasePlanRevisionHash, manifest.PriorCardHash, manifest.InputFrontierHash, canonicalPlanningScoutCandidates(candidates)})
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

// validatePlanningRouteStageResult consumes no worker-authored authority. It
// reconstructs the current Scout evidence and colony bindings, validates the
// proposal with the shared plan-contract kernel, and derives both confidence
// and semantic delta from canonical inputs.
func validatePlanningRouteStageResult(root string, manifest planningStageManifest, raw []byte) (planningRouteStageValidation, error) {
	empty := planningRouteStageValidation{}
	if err := validatePlanningStageManifest(manifest); err != nil {
		return empty, err
	}
	if manifest.ExpectedCaste != planningStageCasteRouteSetter || manifest.ExpectedResultType != planningStageResultRouteSetter {
		return empty, fmt.Errorf("active planning stage manifest is not a Route-Setter contract")
	}
	if err := validatePersistedPlanningStageManifest(root, manifest); err != nil {
		return empty, err
	}
	state, err := loadPlanningStageState(root, manifest.RunID)
	if err != nil {
		return empty, err
	}

	result, err := decodePlanningRouteStageResult(raw)
	if err != nil {
		return empty, err
	}
	if result.ResultType != planningStageResultRouteSetter {
		return empty, fmt.Errorf("Route-Setter result_type must be %q", planningStageResultRouteSetter)
	}
	if result.ManifestID != manifest.ID || result.ManifestHash != manifest.ContentHash {
		return empty, fmt.Errorf("Route-Setter result manifest ID or hash does not match the active manifest")
	}
	if result.RunID != manifest.RunID || result.Pass != manifest.Pass {
		return empty, fmt.Errorf("Route-Setter result run or pass does not match the active manifest")
	}
	if result.Caste != planningStageCasteRouteSetter || result.Caste != manifest.ExpectedCaste {
		return empty, fmt.Errorf("Route-Setter result caste does not match the active manifest")
	}
	if !samePlanningScoutSpecification(result.Specification, manifest.Specification) {
		return empty, fmt.Errorf("Route-Setter result specification does not match the exact approved specification binding")
	}
	if result.BasePlanRevisionID != manifest.BasePlanRevisionID || result.BasePlanRevisionHash != manifest.BasePlanRevisionHash {
		return empty, fmt.Errorf("Route-Setter result base plan revision does not match the active manifest")
	}
	if result.PriorCardHash != manifest.PriorCardHash || result.InputFrontierHash != manifest.InputFrontierHash {
		return empty, fmt.Errorf("Route-Setter result prior card or input frontier does not match the active manifest")
	}
	if manifest.ScoutReceipt == nil || !samePlanningStageReceiptReference(result.ScoutReceipt, *manifest.ScoutReceipt) {
		return empty, fmt.Errorf("Route-Setter result Scout receipt does not match the exact current Scout receipt")
	}
	if result.CandidateSnapshotHash != manifest.CandidateSnapshotHash {
		return empty, fmt.Errorf("Route-Setter result candidate snapshot does not match the active manifest")
	}

	dispatch, err := loadPlanningScoutRouteDispatch(root, manifest.RunID, manifest.Pass)
	if err != nil {
		return empty, err
	}
	if dispatch.Manifest.ID != manifest.ID || dispatch.Manifest.ContentHash != manifest.ContentHash ||
		dispatch.ScoutReceipt.ID != manifest.ScoutReceipt.ID || dispatch.ScoutReceipt.ContentHash != manifest.ScoutReceipt.ContentHash ||
		dispatch.CandidateSnapshotHash != manifest.CandidateSnapshotHash {
		return empty, fmt.Errorf("Route-Setter result does not match the exact Scout-issued authorization")
	}
	if err := validatePlanningRouteDecisionCandidates(manifest.Pass, result.MaterialDecisionCandidates, dispatch.MaterialDecisionCandidates); err != nil {
		return empty, err
	}

	chain, err := readPlanningStageReceiptChain(root, manifest.RunID)
	if err != nil {
		return empty, err
	}
	if state.Stage == planningStageRouteRunning {
		if err := validatePlanningStageRunningState(state, manifest); err != nil {
			return empty, err
		}
	} else if !planningRouteReceiptAlreadyCommitted(chain, manifest) {
		return empty, fmt.Errorf("Route-Setter manifest is not the active stage or an exactly committed replay")
	}
	priorCards, err := planningRouteCardsBeforePass(chain.Cards, manifest.Pass)
	if err != nil {
		return empty, err
	}
	scoutReceipt, scoutManifest, scoutResult, header, err := loadPlanningRouteScoutBoundary(root, manifest, chain)
	if err != nil {
		return empty, err
	}
	colonyState, err := loadSpecificationColonyState(root)
	if err != nil {
		return empty, err
	}
	if err := validatePlanningRouteAuthority(colonyState, manifest); err != nil {
		return empty, err
	}
	if err := validatePlanningRouteProposalAuthority(result.Proposal); err != nil {
		return empty, err
	}

	priorSnapshot, err := planningRoutePriorSnapshot(root, colonyState, manifest, chain)
	if err != nil {
		return empty, err
	}
	proposalContract := planningRouteProposalContract(colonyState.Plan, colonyState.Specification, result.Proposal)
	proposalSnapshot, err := validatePlanProposalContract(proposalContract, &priorSnapshot)
	if err != nil {
		return empty, fmt.Errorf("Route-Setter proposal contract: %w", err)
	}
	delta, err := comparePlanningSemanticSnapshots(priorSnapshot, proposalSnapshot)
	if err != nil {
		return empty, err
	}

	evidence, frontier, err := planningRouteEvidenceContext(root, header, priorCards, scoutManifest, scoutResult, dispatch.MaterialDecisionCandidates)
	if err != nil {
		return empty, err
	}
	result.ProposalEvidenceIDs = uniqueSortedStrings(result.ProposalEvidenceIDs)
	if err := applyPlanningRouteEvidence(&delta, result.ProposalEvidenceIDs, evidence, frontier); err != nil {
		return empty, err
	}

	priorScores, err := planningRoutePriorScores(manifest, priorCards)
	if err != nil {
		return empty, err
	}
	result.DimensionAssessments, err = normalizePlanningRouteAssessments(manifest, result.DimensionAssessments)
	if err != nil {
		return empty, err
	}
	confidence, err := evaluatePlanningConfidence(planningConfidenceProposal{
		PriorScores: priorScores, Assessments: result.DimensionAssessments,
		EvidenceCatalogue: evidence, EvidenceFrontier: frontier, SuppliedOverall: result.SuppliedOverall,
	})
	if err != nil {
		return empty, err
	}
	if err := validatePlanningRouteResolvedGaps(confidence.Assessments, scoutResult.UnresolvedGaps, chain.Cards); err != nil {
		return empty, err
	}

	result.MaterialDecisionCandidates = canonicalPlanningScoutCandidates(result.MaterialDecisionCandidates)
	result.ManifestID = strings.TrimSpace(result.ManifestID)
	result.ManifestHash = strings.TrimSpace(result.ManifestHash)
	result.RunID = strings.TrimSpace(result.RunID)
	result.BasePlanRevisionID = strings.TrimSpace(result.BasePlanRevisionID)
	result.BasePlanRevisionHash = strings.TrimSpace(result.BasePlanRevisionHash)
	result.PriorCardHash = strings.TrimSpace(result.PriorCardHash)
	result.InputFrontierHash = strings.TrimSpace(result.InputFrontierHash)
	result.CandidateSnapshotHash = strings.TrimSpace(result.CandidateSnapshotHash)
	normalized, err := marshalPlanningStageJSON(result)
	if err != nil {
		return empty, fmt.Errorf("marshal normalized Route-Setter result: %w", err)
	}
	return planningRouteStageValidation{
		Result: result, Normalized: normalized,
		ProposalSnapshot: proposalSnapshot, ProposalHash: proposalSnapshot.ContentHash,
		PriorSnapshot: priorSnapshot, SemanticDelta: delta, Confidence: confidence,
		Evidence: evidence, EvidenceFrontier: frontier,
		ScoutResult: scoutResult, ScoutReceipt: scoutReceipt.reference(), RunHeader: header,
	}, nil
}

func decodePlanningRouteStageResult(raw []byte) (planningRouteStageResult, error) {
	var result planningRouteStageResult
	if len(bytes.TrimSpace(raw)) == 0 {
		return result, fmt.Errorf("Route-Setter result is empty")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return planningRouteStageResult{}, fmt.Errorf("decode Route-Setter stage result: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return planningRouteStageResult{}, fmt.Errorf("Route-Setter result must contain exactly one JSON value")
		}
		return planningRouteStageResult{}, fmt.Errorf("decode trailing Route-Setter result data: %w", err)
	}
	return result, nil
}

func samePlanningStageReceiptReference(left, right planningStageReceiptRef) bool {
	return left.ID == right.ID && left.ContentHash == right.ContentHash && left.RunID == right.RunID &&
		left.Pass == right.Pass && left.Caste == right.Caste && left.ManifestHash == right.ManifestHash
}

func validatePlanningRouteDecisionCandidates(pass int, got, want []planningDecisionCandidate) error {
	got = canonicalPlanningScoutCandidates(got)
	want = canonicalPlanningScoutCandidates(want)
	if pass == 1 && len(got) > 0 {
		return fmt.Errorf("first-pass Route-Setter result cannot repeat Scout owner-decision candidates")
	}
	if pass < 2 {
		return nil
	}
	gotHash, err := jsonSHA256(got)
	if err != nil {
		return err
	}
	wantHash, err := jsonSHA256(want)
	if err != nil {
		return err
	}
	if gotHash != wantHash {
		return fmt.Errorf("later Route-Setter result does not carry the exact Scout material decision candidates")
	}
	return nil
}

func loadPlanningRouteScoutBoundary(root string, routeManifest planningStageManifest, chain planningStageReceiptChain) (StageReceipt, planningStageManifest, planningScoutStageResult, planningRunHeader, error) {
	emptyReceipt := StageReceipt{}
	emptyManifest := planningStageManifest{}
	emptyResult := planningScoutStageResult{}
	emptyHeader := planningRunHeader{}
	var receipt *StageReceipt
	for index := range chain.Receipts {
		candidate := chain.Receipts[index]
		if routeManifest.ScoutReceipt != nil && candidate.ID == routeManifest.ScoutReceipt.ID {
			receipt = &candidate
			break
		}
	}
	if receipt == nil || routeManifest.ScoutReceipt == nil || !samePlanningStageReceiptReference(receipt.reference(), *routeManifest.ScoutReceipt) {
		return emptyReceipt, emptyManifest, emptyResult, emptyHeader, fmt.Errorf("Route-Setter manifest Scout receipt is absent from the verified receipt chain")
	}
	scoutManifest, err := loadPlanningStageManifest(root, routeManifest.RunID, receipt.ManifestID)
	if err != nil {
		return emptyReceipt, emptyManifest, emptyResult, emptyHeader, err
	}
	_, raw, err := loadPlanningStageOutput(root, scoutManifest)
	if err != nil {
		return emptyReceipt, emptyManifest, emptyResult, emptyHeader, err
	}
	scoutResult, normalized, _, err := validatePlanningScoutStageResult(root, scoutManifest, raw)
	if err != nil {
		return emptyReceipt, emptyManifest, emptyResult, emptyHeader, fmt.Errorf("revalidate Scout receipt output: %w", err)
	}
	if !bytes.Equal(raw, normalized) {
		return emptyReceipt, emptyManifest, emptyResult, emptyHeader, fmt.Errorf("Scout receipt output is not the canonical normalized result")
	}
	header, err := loadPlanningScoutRunHeader(root, scoutManifest)
	if err != nil {
		return emptyReceipt, emptyManifest, emptyResult, emptyHeader, err
	}
	return *receipt, scoutManifest, scoutResult, header, nil
}

func validatePlanningRouteAuthority(state colony.ColonyState, manifest planningStageManifest) error {
	if state.Specification == nil {
		return fmt.Errorf("Route-Setter result requires the current approved specification")
	}
	revision, ok := currentSpecificationRevision(*state.Specification)
	if !ok || revision.ID != manifest.Specification.RevisionID || revision.ContentHash != manifest.Specification.ContentHash || revision.Status != colony.SpecStatusApproved || revision.Approval == nil {
		return fmt.Errorf("Route-Setter result specification is stale against the current approved specification")
	}
	approvalHash, err := jsonSHA256(*revision.Approval)
	if err != nil {
		return err
	}
	if revision.Approval.ID != manifest.Specification.ApprovalReceiptID || approvalHash != manifest.Specification.ApprovalReceiptHash {
		return fmt.Errorf("Route-Setter result specification approval receipt is stale")
	}
	baseHash, err := planStateHash(state.Plan)
	if err != nil {
		return fmt.Errorf("hash current base plan revision: %w", err)
	}
	baseID, baseHash := planningBaseRevisionIdentity(state.Plan, baseHash)
	if baseID != manifest.BasePlanRevisionID || baseHash != manifest.BasePlanRevisionHash {
		return fmt.Errorf("Route-Setter result base plan revision is stale against current state")
	}
	return nil
}

func validatePlanningRouteProposalAuthority(proposal planningRoutePlanProposal) error {
	for phaseIndex := range proposal.Phases {
		phase := proposal.Phases[phaseIndex]
		phasePath := fmt.Sprintf("proposal.phases[%d]", phaseIndex)
		if strings.TrimSpace(phase.Status) != "" || phase.WatcherFailureCount != 0 {
			return fmt.Errorf("%s.status and watcher_failure_count are Go-owned authority fields", phasePath)
		}
		if phase.SpecificationRevisionID != "" || phase.SpecificationRevisionHash != "" || phase.CandidateID != "" || phase.CandidateContentHash != "" || phase.PlanningTimelineID != "" || phase.PlanningTimelineDigest != "" || len(phase.AffectedSemanticIDs) != 0 || len(phase.PreservedSemanticIDs) != 0 {
			return fmt.Errorf("%s contains worker-supplied specification, candidate, timeline, or activation authority", phasePath)
		}
		for taskIndex := range phase.Tasks {
			task := phase.Tasks[taskIndex]
			taskPath := fmt.Sprintf("%s.tasks[%d]", phasePath, taskIndex)
			if strings.TrimSpace(task.Status) != "" {
				return fmt.Errorf("%s.status is a Go-owned authority field", taskPath)
			}
			if task.SpecificationRevisionID != "" || task.SpecificationRevisionHash != "" || task.CandidateID != "" || task.CandidateContentHash != "" || task.PlanningTimelineID != "" || task.PlanningTimelineDigest != "" || len(task.AffectedSemanticIDs) != 0 || len(task.PreservedSemanticIDs) != 0 {
				return fmt.Errorf("%s contains worker-supplied specification, candidate, timeline, or activation authority", taskPath)
			}
		}
	}
	return nil
}

func planningRouteProposalContract(current colony.Plan, specification *colony.Specification, proposal planningRoutePlanProposal) planProposalContract {
	plan := current
	plan.Phases = clonePhases(proposal.Phases)
	plan.AcceptancePolicy = colony.PlanAcceptanceExplicitOwner
	revision := &colony.PlanRevision{
		SchemaVersion: planRevisionSchemaVersion,
		SemanticID:    canonicalPlanningText(proposal.SemanticID),
		Phases:        clonePhases(proposal.Phases),
	}
	declarations := make([]planProposalTaskDeclaration, len(proposal.TaskDeclarations))
	for index, declaration := range proposal.TaskDeclarations {
		declarations[index] = planProposalTaskDeclaration{
			TaskSemanticID: declaration.TaskSemanticID,
			Files:          append([]string(nil), declaration.Files...), NoFileReason: declaration.NoFileReason, UserFacing: declaration.UserFacing,
		}
	}
	removals := make([]planProposalRemoval, len(proposal.Removals))
	for index, removal := range proposal.Removals {
		removals[index] = planProposalRemoval{SemanticID: removal.SemanticID, Classification: removal.Classification, Rationale: removal.Rationale}
	}
	return planProposalContract{
		Plan: plan, Revision: revision, Specification: specification,
		TaskDeclarations: declarations, UserFacingSemanticIDs: append([]string(nil), proposal.UserFacingSemanticIDs...), Removals: removals,
	}
}

func planningRoutePriorSnapshot(root string, state colony.ColonyState, manifest planningStageManifest, chain planningStageReceiptChain) (planningSemanticSnapshot, error) {
	if manifest.Pass > 1 {
		var previous *StageReceipt
		for index := range chain.Receipts {
			receipt := chain.Receipts[index]
			if receipt.Pass == manifest.Pass-1 && receipt.Caste == planningStageCasteRouteSetter {
				previous = &receipt
			}
		}
		if previous == nil {
			return planningSemanticSnapshot{}, fmt.Errorf("Route-Setter pass %d has no prior Route receipt", manifest.Pass)
		}
		previousManifest, err := loadPlanningStageManifest(root, manifest.RunID, previous.ManifestID)
		if err != nil {
			return planningSemanticSnapshot{}, err
		}
		_, raw, err := loadPlanningStageOutput(root, previousManifest)
		if err != nil {
			return planningSemanticSnapshot{}, err
		}
		previousResult, err := decodePlanningRouteStageResult(raw)
		if err != nil {
			return planningSemanticSnapshot{}, err
		}
		contract := planningRouteProposalContract(state.Plan, state.Specification, previousResult.Proposal)
		return validatePlanProposalContract(contract, nil)
	}

	var revision *colony.PlanRevision
	activeID := strings.TrimSpace(state.Plan.ActiveRevisionID)
	if activeID != "" {
		for index := range state.Plan.Revisions {
			if state.Plan.Revisions[index].ID == activeID {
				copy := state.Plan.Revisions[index]
				revision = &copy
				break
			}
		}
		if revision == nil {
			return planningSemanticSnapshot{}, fmt.Errorf("active base plan revision %q is missing", activeID)
		}
	}
	return buildPlanningSemanticSnapshot(planningSemanticSnapshotSource{
		Plan: state.Plan, Revision: revision, Specification: state.Specification,
	})
}

func planningRouteEvidenceContext(root string, header planningRunHeader, cards []colony.PlanningIterationCard, scoutManifest planningStageManifest, scout planningScoutStageResult, candidates []planningDecisionCandidate) ([]colony.PlanningEvidenceRef, planningEvidenceFrontier, error) {
	byID := make(map[string]colony.PlanningEvidenceRef)
	ordered := make([]colony.PlanningEvidenceRef, 0, len(header.EvidenceCatalogue)+len(scout.NewEvidence))
	add := func(reference colony.PlanningEvidenceRef) error {
		if err := reference.Validate(); err != nil {
			return err
		}
		if prior, exists := byID[reference.ID]; exists {
			if prior.ContentHash != reference.ContentHash {
				return fmt.Errorf("planning evidence ID %q has conflicting content hashes", reference.ID)
			}
			return nil
		}
		byID[reference.ID] = reference
		ordered = append(ordered, reference)
		return nil
	}
	for _, record := range header.EvidenceCatalogue {
		if err := add(record.Reference); err != nil {
			return nil, planningEvidenceFrontier{}, fmt.Errorf("planning run evidence: %w", err)
		}
	}
	if scoutManifest.Pass > 1 {
		continuation, err := loadPlanningRouteScoutDispatch(root, scoutManifest.RunID, scoutManifest.Pass)
		if err != nil {
			return nil, planningEvidenceFrontier{}, fmt.Errorf("load prior Route evidence frontier: %w", err)
		}
		if continuation.Manifest.ID != scoutManifest.ID || continuation.Manifest.ContentHash != scoutManifest.ContentHash {
			return nil, planningEvidenceFrontier{}, fmt.Errorf("prior Route evidence frontier does not bind the current Scout manifest")
		}
		for _, reference := range continuation.Evidence {
			if err := add(reference); err != nil {
				return nil, planningEvidenceFrontier{}, fmt.Errorf("prior Route evidence frontier: %w", err)
			}
		}
	}
	for _, record := range scout.NewEvidence {
		if err := add(record.Reference); err != nil {
			return nil, planningEvidenceFrontier{}, fmt.Errorf("Scout evidence: %w", err)
		}
	}
	for _, candidate := range candidates {
		for _, reference := range candidate.Evidence {
			if err := add(reference); err != nil {
				return nil, planningEvidenceFrontier{}, fmt.Errorf("owner-decision evidence: %w", err)
			}
		}
	}
	sort.Slice(ordered, func(left, right int) bool { return ordered[left].ID < ordered[right].ID })
	frontier := planningEvidenceFrontier{
		Scope: planningEvidenceScope{
			GoalID: header.GoalID, SessionID: header.SessionID,
			SpecificationRevisionID: scoutManifest.Specification.RevisionID,
			PlanRevisionID:          scoutManifest.BasePlanRevisionID,
		},
		CurrentSourceHashes: map[string]string{}, SourceStates: map[string]planningEvidenceSourceState{},
	}
	for _, reference := range ordered {
		key := planningEvidenceSourceKey(reference)
		frontier.CurrentSourceHashes[key] = reference.ContentHash
		frontier.SourceStates[key] = planningEvidenceSourceCurrent
	}
	for _, card := range cards {
		frontier.CitedEvidenceIDs = append(frontier.CitedEvidenceIDs, card.EvidenceIDs...)
	}
	frontier.CitedEvidenceIDs = uniqueSortedStrings(frontier.CitedEvidenceIDs)
	return ordered, frontier, nil
}

func applyPlanningRouteEvidence(delta *colony.PlanningSemanticDelta, evidenceIDs []string, catalogue []colony.PlanningEvidenceRef, frontier planningEvidenceFrontier) error {
	if delta == nil {
		return fmt.Errorf("Route-Setter semantic delta is required")
	}
	byID := make(map[string]colony.PlanningEvidenceRef, len(catalogue))
	for _, reference := range catalogue {
		byID[reference.ID] = reference
	}
	for _, evidenceID := range evidenceIDs {
		reference, ok := byID[evidenceID]
		if !ok {
			return fmt.Errorf("Route-Setter proposal evidence %q is not in the current frontier", evidenceID)
		}
		if freshness := isFreshPlanningEvidence(reference, frontier); !freshness.Allowed {
			return fmt.Errorf("Route-Setter proposal evidence %q is not current: %s", evidenceID, freshness.Reason)
		}
	}
	changed := false
	sections := []struct {
		name    colony.PlanningSemanticSection
		changes *[]colony.PlanningSemanticChange
	}{
		{name: colony.PlanningSemanticSectionPhases, changes: &delta.Phases},
		{name: colony.PlanningSemanticSectionTasks, changes: &delta.Tasks},
		{name: colony.PlanningSemanticSectionDependencies, changes: &delta.Dependencies},
		{name: colony.PlanningSemanticSectionRequirementLinks, changes: &delta.RequirementLinks},
		{name: colony.PlanningSemanticSectionAcceptanceChecks, changes: &delta.AcceptanceChecks},
		{name: colony.PlanningSemanticSectionNegativeExpectations, changes: &delta.NegativeExpectations},
		{name: colony.PlanningSemanticSectionRecoveryExpectations, changes: &delta.RecoveryExpectations},
		{name: colony.PlanningSemanticSectionPublicPaths, changes: &delta.PublicPaths},
	}
	for _, section := range sections {
		for index := range *section.changes {
			change := &(*section.changes)[index]
			if change.Kind != colony.PlanningSemanticChangePreserved {
				changed = true
				if len(evidenceIDs) == 0 {
					return fmt.Errorf("semantic change %q requires evidence from the current frontier or an explicit decision", change.SemanticID)
				}
				change.EvidenceIDs = append([]string(nil), evidenceIDs...)
			}
			if err := colony.AddressPlanningSemanticChange(section.name, change); err != nil {
				return fmt.Errorf("semantic change %q: %w", change.SemanticID, err)
			}
		}
	}
	if !changed && len(evidenceIDs) > 0 {
		// Evidence may still ground confidence movement even when the plan is
		// semantically unchanged; retaining the IDs on the result is useful.
	}
	return colony.AddressPlanningSemanticDelta(delta)
}

func planningRoutePriorScores(manifest planningStageManifest, cards []colony.PlanningIterationCard) (planningConfidenceScores, error) {
	if manifest.Pass == 1 {
		if len(cards) != 0 {
			return planningConfidenceScores{}, fmt.Errorf("Route-Setter pass 1 cannot have a prior iteration card")
		}
		return planningConfidenceScores{}, nil
	}
	if len(cards) != manifest.Pass-1 {
		return planningConfidenceScores{}, fmt.Errorf("Route-Setter pass %d requires %d prior iteration cards", manifest.Pass, manifest.Pass-1)
	}
	last := cards[len(cards)-1]
	if last.ContentHash != manifest.PriorCardHash {
		return planningConfidenceScores{}, fmt.Errorf("Route-Setter prior card hash does not match the verified timeline")
	}
	var scores planningConfidenceScores
	for _, assessment := range last.DimensionAssessments {
		scores.Set(assessment.Dimension, assessment.After)
	}
	return scores, nil
}

func normalizePlanningRouteAssessments(manifest planningStageManifest, assessments []colony.PlanningDimensionAssessment) ([]colony.PlanningDimensionAssessment, error) {
	result := make([]colony.PlanningDimensionAssessment, len(assessments))
	for index := range assessments {
		assessment := clonePlanningConfidenceAssessment(assessments[index])
		if strings.TrimSpace(assessment.ProducerReceiptID) != manifest.ID {
			return nil, fmt.Errorf("dimension_assessments[%d].producer_receipt_id must bind the exact Route-Setter worker manifest", index)
		}
		assessment.SchemaVersion = colony.PlanningSchemaVersion
		assessment.Rationale = normalizePlanningDecisionText(assessment.Rationale)
		assessment.FreshEvidenceIDs = uniqueSortedStrings(assessment.FreshEvidenceIDs)
		assessment.ResolvedGapIDs = uniqueSortedStrings(assessment.ResolvedGapIDs)
		assessment.ProducerReceiptID = strings.TrimSpace(assessment.ProducerReceiptID)
		assessment.RemainingGap.SchemaVersion = colony.PlanningSchemaVersion
		assessment.RemainingGap.Description = normalizePlanningDecisionText(assessment.RemainingGap.Description)
		assessment.RemainingGap.EvidenceThatWouldChange = normalizePlanningDecisionText(assessment.RemainingGap.EvidenceThatWouldChange)
		assessment.RemainingGap.EvidenceIDs = uniqueSortedStrings(assessment.RemainingGap.EvidenceIDs)
		if err := colony.AddressPlanningDimensionAssessment(&assessment); err != nil {
			return nil, fmt.Errorf("dimension_assessments[%d]: %w", index, err)
		}
		result[index] = assessment
	}
	return result, nil
}

func validatePlanningRouteResolvedGaps(assessments []colony.PlanningDimensionAssessment, scoutGaps []colony.PlanningGap, cards []colony.PlanningIterationCard) error {
	known := make(map[string]struct{})
	for _, gap := range scoutGaps {
		known[gap.ID] = struct{}{}
	}
	for _, card := range cards {
		for _, assessment := range card.DimensionAssessments {
			known[assessment.RemainingGap.ID] = struct{}{}
		}
	}
	for index, assessment := range assessments {
		for _, gapID := range assessment.ResolvedGapIDs {
			if _, ok := known[gapID]; !ok {
				return fmt.Errorf("dimension_assessments[%d].resolved_gap_ids names unknown gap %q", index, gapID)
			}
			if len(assessment.FreshEvidenceIDs) == 0 {
				return fmt.Errorf("dimension_assessments[%d] resolves gap %q without fresh evidence", index, gapID)
			}
		}
	}
	return nil
}

func planningRouteReceiptAlreadyCommitted(chain planningStageReceiptChain, manifest planningStageManifest) bool {
	for _, receipt := range chain.Receipts {
		if receipt.ManifestID == manifest.ID && receipt.ManifestHash == manifest.ContentHash && receipt.Caste == planningStageCasteRouteSetter && receipt.Pass == manifest.Pass {
			return true
		}
	}
	return false
}

func planningRouteCardsBeforePass(cards []colony.PlanningIterationCard, pass int) ([]colony.PlanningIterationCard, error) {
	want := pass - 1
	if want < 0 || len(cards) < want || len(cards) > pass {
		return nil, fmt.Errorf("Route-Setter pass %d has an invalid %d-card timeline", pass, len(cards))
	}
	for index := 0; index < want; index++ {
		if cards[index].Iteration != index+1 {
			return nil, fmt.Errorf("Route-Setter timeline card %d is not contiguous", index)
		}
	}
	return append([]colony.PlanningIterationCard(nil), cards[:want]...), nil
}

func finalizePlanningRouteStage(root string, manifest planningStageManifest, raw []byte) (planningRouteStageFinalization, error) {
	return finalizePlanningRouteStageWithOptions(root, manifest, raw, planningRouteStageFinalizeOptions{})
}

func finalizePlanningRouteStageWithOptions(root string, manifest planningStageManifest, raw []byte, opts planningRouteStageFinalizeOptions) (planningRouteStageFinalization, error) {
	empty := planningRouteStageFinalization{}
	validated, err := validatePlanningRouteStageResult(root, manifest, raw)
	if err != nil {
		return empty, err
	}
	artifact, err := writePlanningStageOutput(root, manifest, validated.Normalized, planningStageWriteOptions{})
	if err != nil {
		return empty, err
	}

	_, cards, _, err := readPlanningTimelineChain(root, manifest.RunID)
	if err != nil {
		return empty, err
	}
	priorCards, err := planningRouteCardsBeforePass(cards, manifest.Pass)
	if err != nil {
		return empty, err
	}
	history, err := planningRouteConfidenceHistory(priorCards, validated)
	if err != nil {
		return empty, err
	}
	stopPolicy, err := evaluatePlanningStopPolicy(planningStopPolicyInput{
		Target: validated.RunHeader.TargetConfidence, PassCap: validated.RunHeader.PassCap, History: history,
	})
	if err != nil {
		return empty, err
	}
	materialResidual := false
	for _, gap := range validated.Confidence.RankedGaps {
		materialResidual = materialResidual || gap.Materiality == colony.PlanningGapMaterial
	}
	if len(validated.Result.MaterialDecisionCandidates) > 0 || (materialResidual && stopPolicy.Decision.Reason != colony.PlanningStopContinue) {
		rationale := fmt.Sprintf("Route-Setter pass %d completed before pausing for owner authority over unresolved material planning evidence", manifest.Pass)
		decision, decisionErr := planningConfidenceStopDecision(colony.PlanningStopOwnerDecision, rationale, history[len(history)-1])
		if decisionErr != nil {
			return empty, decisionErr
		}
		stopPolicy.Decision = decision
	}
	next, decisionResume := planningRouteResultingStage(stopPolicy.Decision.Reason)
	card := colony.PlanningIterationCard{
		SchemaVersion: colony.PlanningIterationSchemaVersion,
		RunID:         manifest.RunID, Iteration: manifest.Pass,
		EvidenceIDs:          planningRouteCardEvidenceIDs(validated),
		DimensionAssessments: append([]colony.PlanningDimensionAssessment(nil), validated.Confidence.Assessments...),
		WeakestGap:           clonePlanningConfidenceGap(validated.Confidence.WeakestGap),
		SemanticDelta:        validated.SemanticDelta, Decision: stopPolicy.Decision,
		EvidenceThatWouldChange: stopPolicy.Decision.EvidenceThatWouldChange,
		CreatedAt:               planningRouteCardCreatedAt(validated),
	}
	receipt, err := finalizePlanningStage(root, manifest, planningStageFinalizeRequest{
		To: next, DecisionResumeStage: decisionResume, RouteCard: &card, Fault: opts.Fault, Rename: opts.Rename,
	})
	if err != nil {
		return empty, err
	}
	timeline, err := loadPlanningTimeline(root, manifest.RunID)
	if err != nil {
		return empty, err
	}
	var persisted *colony.PlanningIterationCard
	for index := range timeline.Cards {
		if timeline.Cards[index].Iteration == manifest.Pass && timeline.Cards[index].RouteSetterReceiptID == receipt.ID {
			copy := timeline.Cards[index]
			persisted = &copy
			break
		}
	}
	if persisted == nil {
		return empty, fmt.Errorf("completed Route-Setter receipt has no exact iteration card")
	}
	return planningRouteStageFinalization{
		Validation: validated, Artifact: artifact, Receipt: receipt, Card: *persisted, StopPolicy: stopPolicy,
	}, nil
}

func planningRouteConfidenceHistory(cards []colony.PlanningIterationCard, current planningRouteStageValidation) ([]planningConfidencePass, error) {
	history := make([]planningConfidencePass, 0, len(cards)+1)
	for index, card := range cards {
		var scores planningConfidenceScores
		for _, assessment := range card.DimensionAssessments {
			scores.Set(assessment.Dimension, assessment.After)
		}
		evaluation := planningConfidenceEvaluation{
			Scores: scores, Assessments: append([]colony.PlanningDimensionAssessment(nil), card.DimensionAssessments...),
		}
		evaluation.RankedGaps = rankPlanningConfidenceGaps(evaluation.Assessments, scores)
		if len(evaluation.RankedGaps) == 0 {
			return nil, fmt.Errorf("iteration card %d has no ranked planning gap", index+1)
		}
		evaluation.WeakestGap = clonePlanningConfidenceGap(evaluation.RankedGaps[0])
		history = append(history, planningConfidencePass{Iteration: card.Iteration, Evaluation: evaluation, SemanticDelta: card.SemanticDelta})
	}
	history = append(history, planningConfidencePass{
		Iteration: current.Result.Pass, Evaluation: current.Confidence, SemanticDelta: current.SemanticDelta,
	})
	return history, nil
}

func planningRouteResultingStage(reason colony.PlanningStopReason) (planningStage, planningStage) {
	switch reason {
	case colony.PlanningStopContinue:
		return planningStageContinueReady, ""
	case colony.PlanningStopOwnerDecision:
		return planningStageOwnerDecision, planningStageScoutReady
	default:
		return planningStageCandidateReady, ""
	}
}

func planningRouteCardEvidenceIDs(validated planningRouteStageValidation) []string {
	ids := append([]string(nil), validated.Result.ProposalEvidenceIDs...)
	for _, assessment := range validated.Confidence.Assessments {
		ids = append(ids, assessment.FreshEvidenceIDs...)
		ids = append(ids, assessment.RemainingGap.EvidenceIDs...)
	}
	return uniqueSortedStrings(ids)
}

func planningRouteCardCreatedAt(validated planningRouteStageValidation) time.Time {
	createdAt := validated.RunHeader.CreatedAt.UTC()
	for _, reference := range validated.Evidence {
		if reference.ObservedAt.After(createdAt) {
			createdAt = reference.ObservedAt.UTC()
		}
	}
	if createdAt.IsZero() {
		createdAt = time.Unix(int64(validated.Result.Pass), 0).UTC()
	}
	return createdAt
}

// coordinatePlanningRouteStage advances only after the Route artifact,
// receipt, and iteration card have committed. The next effect is exactly one
// of: a Scout dispatch, an owner checkpoint, or a non-active candidate.
func coordinatePlanningRouteStage(root string, manifest planningStageManifest, raw []byte) (planningRouteStageCoordination, error) {
	empty := planningRouteStageCoordination{}
	completed, err := finalizePlanningRouteStage(root, manifest, raw)
	if err != nil {
		return empty, err
	}
	result := planningRouteStageCoordination{Route: completed}
	state, err := loadPlanningStageState(root, manifest.RunID)
	if err != nil {
		return empty, err
	}

	switch state.Stage {
	case planningStageContinueReady:
		dispatch, dispatchErr := authorizePlanningRouteScout(root, state, completed.Card, completed.Validation.ProposalHash, completed.Validation.Evidence, nil, false)
		if dispatchErr != nil {
			return empty, dispatchErr
		}
		result.ScoutDispatch = &dispatch
		return result, nil

	case planningStageScoutRunning:
		dispatch, dispatchErr := loadPlanningRouteScoutDispatch(root, state.RunID, state.Pass)
		if dispatchErr != nil {
			return empty, dispatchErr
		}
		if dispatch.ProposalHash != completed.Validation.ProposalHash || dispatch.PriorCardHash != completed.Card.ContentHash {
			return empty, fmt.Errorf("persisted next Scout dispatch does not bind the completed Route proposal and card")
		}
		result.ScoutDispatch = &dispatch
		return result, nil

	case planningStageOwnerDecision:
		checkpoint, checkpointErr := loadPlanningRouteDecisionCheckpoint(root, state.RunID, state.Pass)
		if errors.Is(checkpointErr, os.ErrNotExist) {
			checkpoint, checkpointErr = buildPlanningRouteDecisionCheckpoint(root, completed)
			if checkpointErr == nil {
				checkpointErr = persistPlanningRouteDecisionCheckpoint(root, checkpoint)
			}
		}
		if checkpointErr != nil {
			return empty, checkpointErr
		}
		result.DecisionCheckpoint = &checkpoint
		return result, nil

	case planningStageCandidateReady:
		candidate, candidateErr := buildPlanningRouteCandidate(root, completed)
		if candidateErr != nil {
			return empty, candidateErr
		}
		if candidateErr = persistPlanningRouteCandidate(root, candidate); candidateErr != nil {
			return empty, candidateErr
		}
		result.Candidate = &candidate
		return result, nil

	case planningStageSpecApprovalRequired, planningStageReconciliationRequired:
		checkpoint, checkpointErr := loadPlanningRouteDecisionCheckpoint(root, state.RunID, state.Pass)
		if checkpointErr != nil {
			return empty, checkpointErr
		}
		result.DecisionCheckpoint = &checkpoint
		return result, nil

	default:
		return empty, fmt.Errorf("completed Route-Setter reached unexpected planning stage %q", state.Stage)
	}
}

func planningRouteEvidenceBindings(evidence []colony.PlanningEvidenceRef) ([]planningStageEvidenceBinding, error) {
	bindings := make([]planningStageEvidenceBinding, 0, len(evidence))
	seen := make(map[string]string, len(evidence))
	for _, reference := range evidence {
		if err := reference.Validate(); err != nil {
			return nil, err
		}
		if hash, exists := seen[reference.ID]; exists {
			if hash != reference.ContentHash {
				return nil, fmt.Errorf("planning evidence %q has conflicting hashes", reference.ID)
			}
			continue
		}
		seen[reference.ID] = reference.ContentHash
		bindings = append(bindings, planningStageEvidenceBinding{ID: reference.ID, ContentHash: reference.ContentHash})
	}
	sort.Slice(bindings, func(left, right int) bool { return bindings[left].ID < bindings[right].ID })
	if len(bindings) == 0 {
		return nil, fmt.Errorf("next Scout dispatch requires an evidence frontier")
	}
	return bindings, nil
}

func authorizePlanningRouteScout(root string, state planningStageState, card colony.PlanningIterationCard, proposalHash string, evidence []colony.PlanningEvidenceRef, token *planningDecisionResumeToken, afterOwnerDecision bool) (planningRouteScoutDispatch, error) {
	empty := planningRouteScoutDispatch{}
	if state.Stage == planningStageScoutRunning {
		return loadPlanningRouteScoutDispatch(root, state.RunID, state.Pass)
	}
	bindings, err := planningRouteEvidenceBindings(evidence)
	if err != nil {
		return empty, err
	}
	if !planningSHA256Pattern.MatchString(proposalHash) || !planningSHA256Pattern.MatchString(card.ContentHash) {
		return empty, fmt.Errorf("next Scout dispatch requires exact proposal and completed-card hashes")
	}
	frontierHash, err := jsonSHA256(struct {
		RunID         string                         `json:"run_id"`
		NextPass      int                            `json:"next_pass"`
		ProposalHash  string                         `json:"proposal_hash"`
		PriorCardHash string                         `json:"prior_card_hash"`
		WeakestGap    colony.PlanningGap             `json:"weakest_gap"`
		Evidence      []planningStageEvidenceBinding `json:"evidence"`
	}{state.RunID, state.Pass + 1, proposalHash, card.ContentHash, card.WeakestGap, bindings})
	if err != nil {
		return empty, fmt.Errorf("hash next Scout frontier: %w", err)
	}

	ready := state
	if state.Stage == planningStageContinueReady {
		ready, _, err = reducePlanningStage(state, planningStageTransition{
			To: planningStageScoutReady, NextInputFrontierHash: frontierHash, NextWeakestGap: &card.WeakestGap,
		})
		if err != nil {
			return empty, fmt.Errorf("prepare next Scout pass: %w", err)
		}
	} else if state.Stage == planningStageScoutReady && afterOwnerDecision {
		// A post-card owner resolution resumes at Scout-ready by reducer
		// contract. Advancing the completed pass here mirrors the
		// continue_ready reducer without manufacturing a worker result.
		ready = clonePlanningStageState(state)
		ready.Pass++
		ready.InputFrontierHash = frontierHash
		ready.PriorCardHash = card.ContentHash
		ready.WeakestGap = clonePlanningStageGap(&card.WeakestGap)
		ready.ScoutReceipt = nil
		ready.CandidateSnapshotHash = ""
		ready.DecisionResumeStage = ""
	} else {
		return empty, fmt.Errorf("next Scout authorization requires continue_ready or a completed owner-decision resume")
	}
	seed, err := jsonSHA256(struct {
		RunID         string                         `json:"run_id"`
		Pass          int                            `json:"pass"`
		ProposalHash  string                         `json:"proposal_hash"`
		PriorCardHash string                         `json:"prior_card_hash"`
		FrontierHash  string                         `json:"frontier_hash"`
		Evidence      []planningStageEvidenceBinding `json:"evidence"`
	}{ready.RunID, ready.Pass, proposalHash, card.ContentHash, frontierHash, bindings})
	if err != nil {
		return empty, err
	}
	authorization := planningStageAuthorization{
		ID: "planning-authorization-" + seed[:16], ExpectedCaste: planningStageCasteScout,
		InputFrontierHash: frontierHash, EvidenceFrontier: bindings, WeakestGap: clonePlanningStageGap(&card.WeakestGap),
	}
	running, stageManifest, err := reducePlanningStage(ready, planningStageTransition{To: planningStageScoutRunning, Authorization: &authorization})
	if err != nil {
		return empty, fmt.Errorf("authorize next Scout: %w", err)
	}
	if stageManifest == nil {
		return empty, fmt.Errorf("next Scout authorization emitted no manifest")
	}
	resumeHash := ""
	if token != nil {
		resumeHash = token.ContentHash
	}
	dispatch := planningRouteScoutDispatch{
		RunID: state.RunID, Pass: stageManifest.Pass, ProposalHash: proposalHash, PriorCardHash: card.ContentHash,
		Evidence: append([]colony.PlanningEvidenceRef(nil), evidence...), Authorization: authorization, Manifest: *stageManifest, ResumeTokenHash: resumeHash,
	}
	if err := addressPlanningRouteScoutDispatch(&dispatch); err != nil {
		return empty, err
	}
	if err := persistPlanningRouteScoutDispatch(root, running, dispatch, token); err != nil {
		return empty, err
	}
	return dispatch, nil
}

func addressPlanningRouteScoutDispatch(dispatch *planningRouteScoutDispatch) error {
	if dispatch == nil {
		return fmt.Errorf("Route continuation Scout dispatch is required")
	}
	payload := *dispatch
	payload.ID = ""
	payload.ContentHash = ""
	hash, err := jsonSHA256(payload)
	if err != nil {
		return err
	}
	dispatch.ContentHash = hash
	dispatch.ID = "planning-route-scout-" + hash[:16]
	return nil
}

func planningRouteScoutDispatchRepositoryPath(runID string, pass int) string {
	return path.Join(".aether", "data", "planning", strings.TrimSpace(runID), "scout-authorizations", fmt.Sprintf("pass-%04d.json", pass))
}

func persistPlanningRouteScoutDispatch(root string, running planningStageState, dispatch planningRouteScoutDispatch, token *planningDecisionResumeToken) error {
	if err := validatePlanningStageManifest(dispatch.Manifest); err != nil {
		return err
	}
	if err := validatePlanningStageRunningState(running, dispatch.Manifest); err != nil {
		return err
	}
	dispatchBytes, err := marshalPlanningStageJSON(dispatch)
	if err != nil {
		return err
	}
	manifestBytes, err := marshalPlanningStageJSON(dispatch.Manifest)
	if err != nil {
		return err
	}
	stateBytes, err := marshalPlanningStageJSON(running)
	if err != nil {
		return err
	}
	files := map[string][]byte{
		planningRouteScoutDispatchRepositoryPath(dispatch.RunID, dispatch.Pass):   dispatchBytes,
		planningStageManifestRepositoryPath(dispatch.RunID, dispatch.Manifest.ID): manifestBytes,
		planningStageStateRepositoryPath(dispatch.RunID):                          stateBytes,
	}
	if token != nil {
		tokenBytes, marshalErr := marshalPlanningStageJSON(token)
		if marshalErr != nil {
			return marshalErr
		}
		files[planningScoutDecisionResumeRepositoryPath(dispatch.RunID)] = tokenBytes
	}
	return persistPlanningScoutFiles(root, "planning-route-scout-"+dispatch.ContentHash[:24], "planning-route-scout", dispatch.ID, files, map[string]bool{
		planningStageStateRepositoryPath(dispatch.RunID): true,
	})
}

func loadPlanningRouteScoutDispatch(root, runID string, pass int) (planningRouteScoutDispatch, error) {
	var dispatch planningRouteScoutDispatch
	repositoryRoot, _, err := planningStageRoots(root)
	if err != nil {
		return dispatch, err
	}
	content, exists, err := readOptionalPlanningStageFile(repositoryRoot, planningRouteScoutDispatchRepositoryPath(runID, pass))
	if err != nil {
		return dispatch, err
	}
	if !exists {
		return dispatch, fmt.Errorf("next Scout authorization for run %q pass %d is missing: %w", runID, pass, os.ErrNotExist)
	}
	if err := decodePlanningStageJSON(content, &dispatch); err != nil {
		return planningRouteScoutDispatch{}, err
	}
	originalID, originalHash := dispatch.ID, dispatch.ContentHash
	if err := addressPlanningRouteScoutDispatch(&dispatch); err != nil {
		return planningRouteScoutDispatch{}, err
	}
	if dispatch.ID != originalID || dispatch.ContentHash != originalHash || dispatch.RunID != strings.TrimSpace(runID) || dispatch.Pass != pass {
		return planningRouteScoutDispatch{}, fmt.Errorf("next Scout authorization is not a valid content address")
	}
	return dispatch, nil
}

func buildPlanningRouteDecisionCheckpoint(root string, completed planningRouteStageFinalization) (planningScoutDecisionCheckpoint, error) {
	empty := planningScoutDecisionCheckpoint{}
	candidates := append([]planningDecisionCandidate(nil), completed.Validation.Result.MaterialDecisionCandidates...)
	if len(candidates) == 0 {
		candidate, err := planningRouteMaterialGapDecision(root, completed)
		if err != nil {
			return empty, err
		}
		candidates = []planningDecisionCandidate{candidate}
	}
	boundary := planningDecisionBoundary{
		RunID: completed.Receipt.RunID, Pass: completed.Receipt.Pass,
		ScoutReceipt: planningDecisionStageReceipt{
			ID: completed.Validation.ScoutReceipt.ID, ContentHash: completed.Validation.ScoutReceipt.ContentHash,
			RunID: completed.Validation.ScoutReceipt.RunID, Pass: completed.Validation.ScoutReceipt.Pass,
		},
		RouteSetterReceipt: &planningDecisionStageReceipt{ID: completed.Receipt.ID, ContentHash: completed.Receipt.ContentHash, RunID: completed.Receipt.RunID, Pass: completed.Receipt.Pass},
		IterationCard: &planningDecisionIterationCardReceipt{
			ID: completed.Card.ID, ContentHash: completed.Card.ContentHash, RunID: completed.Card.RunID, Pass: completed.Card.Iteration,
			ScoutReceiptID: completed.Card.ScoutReceiptID, ScoutReceiptHash: completed.Card.ScoutReceiptHash,
			RouteSetterReceiptID: completed.Card.RouteSetterReceiptID, RouteSetterReceiptHash: completed.Card.RouteSetterReceiptHash,
		},
		RecoveryCommand: "aether plan",
	}
	batch, err := buildPlanningDecisionBatch(boundary, candidates)
	if err != nil {
		return empty, fmt.Errorf("build completed-Route decision batch: %w", err)
	}
	if batch == nil {
		return empty, fmt.Errorf("owner_decision has no material completed-Route decision batch")
	}
	scope := planningDecisionEquivalenceScope{
		GoalID: completed.Validation.RunHeader.GoalID, SessionID: completed.Validation.RunHeader.SessionID,
		ApprovedSpecificationRevisionID: completed.Validation.Result.Specification.RevisionID,
		BasePlanRevisionID:              completed.Validation.Result.BasePlanRevisionID,
	}
	cards := make([]planningDecisionCard, 0, len(batch.Decisions))
	for _, candidate := range batch.Decisions {
		card, cardErr := projectPlanningDecisionCard(planningDecisionCardRequest{Candidate: candidate, Scope: scope})
		if cardErr != nil {
			return empty, cardErr
		}
		cards = append(cards, card)
	}
	tokenPayload := struct {
		GoalID            string `json:"goal_id"`
		SessionID         string `json:"session_id"`
		BatchID           string `json:"batch_id"`
		BatchHash         string `json:"batch_hash"`
		CompletedCardHash string `json:"completed_card_hash"`
		RecoveryCommand   string `json:"recovery_command"`
	}{scope.GoalID, scope.SessionID, batch.ID, batch.ContentHash, completed.Card.ContentHash, batch.RecoveryCommand}
	tokenHash, err := jsonSHA256(tokenPayload)
	if err != nil {
		return empty, err
	}
	checkpoint := planningScoutDecisionCheckpoint{
		RunID: completed.Receipt.RunID, Pass: completed.Receipt.Pass,
		CompletedCardHash: completed.Card.ContentHash, ProposalHash: completed.Validation.ProposalHash,
		Evidence:              append([]colony.PlanningEvidenceRef(nil), completed.Validation.Evidence...),
		CandidateSnapshotHash: completed.Receipt.CandidateSnapshotHash, Scope: scope, Batch: *batch, Cards: cards,
		ResumeToken: planningScoutDecisionBoundaryToken{
			ID: "planning-decision-boundary-" + tokenHash[:16], ContentHash: tokenHash,
			GoalID: scope.GoalID, SessionID: scope.SessionID, BatchID: batch.ID, BatchHash: batch.ContentHash,
			CompletedCardHash: completed.Card.ContentHash, RecoveryCommand: batch.RecoveryCommand,
		},
	}
	if err := addressPlanningScoutDecisionCheckpoint(&checkpoint); err != nil {
		return empty, err
	}
	return checkpoint, nil
}

func planningRouteMaterialGapDecision(root string, completed planningRouteStageFinalization) (planningDecisionCandidate, error) {
	gap := completed.Card.WeakestGap
	if gap.Materiality != colony.PlanningGapMaterial {
		return planningDecisionCandidate{}, fmt.Errorf("owner decision has no carried candidate or material residual gap")
	}
	state, err := loadSpecificationColonyState(root)
	if err != nil {
		return planningDecisionCandidate{}, err
	}
	if state.Specification == nil {
		return planningDecisionCandidate{}, fmt.Errorf("material gap decision requires the current specification")
	}
	revision, ok := currentSpecificationRevision(*state.Specification)
	if !ok || len(revision.Requirements) == 0 {
		return planningDecisionCandidate{}, fmt.Errorf("material gap decision requires a stable specification requirement")
	}
	evidenceByID := make(map[string]colony.PlanningEvidenceRef, len(completed.Validation.Evidence))
	for _, reference := range completed.Validation.Evidence {
		evidenceByID[reference.ID] = reference
	}
	evidence := make([]colony.PlanningEvidenceRef, 0, len(gap.EvidenceIDs))
	for _, id := range gap.EvidenceIDs {
		if reference, exists := evidenceByID[id]; exists {
			evidence = append(evidence, reference)
		}
	}
	if len(evidence) == 0 && len(completed.Validation.Evidence) > 0 {
		evidence = append(evidence, completed.Validation.Evidence[0])
	}
	approvedImpact := planningDecisionContractImpact{Risk: "Resolve the material planning gap before candidate generation."}
	requirementID := revision.Requirements[0].ID
	return planningDecisionCandidate{
		StableID: "route-material-gap-" + gap.ContentHash[:12], Domain: planningDecisionDomainRiskTolerance,
		Decision: "Should planning continue to resolve this material risk, or should the promised risk contract change?",
		WhyNow:   "The configured stop boundary was reached while a material gap remains: " + gap.Description,
		Evidence: evidence, QueenRecommendation: "Continue research unless the owner explicitly changes the documented risk contract.",
		Impact: approvedImpact, AffectedSemanticIDs: []string{requirementID},
		Choices: []planningDecisionChoice{
			{ID: "continue-research", Label: "Continue research", Consequence: "Planning resumes with the exact weakest gap and no contract change.", Impact: approvedImpact},
			{ID: "proceed-with-risk", Label: "Proceed despite risk", Consequence: "A successor specification must document the changed risk promise before planning can continue.", Impact: planningDecisionContractImpact{Risk: "Proceed with the documented material gap unresolved."}, AffectedSemanticIDs: []string{requirementID}},
		},
		ResumeInstruction: "Answer every card; direct answers resume Scout, while a changed risk contract creates a successor specification draft.",
	}, nil
}

func planningRouteDecisionCheckpointRepositoryPath(runID string, pass int) string {
	return path.Join(".aether", "data", "planning", strings.TrimSpace(runID), "decisions", fmt.Sprintf("pass-%04d.json", pass))
}

func persistPlanningRouteDecisionCheckpoint(root string, checkpoint planningScoutDecisionCheckpoint) error {
	content, err := marshalPlanningStageJSON(checkpoint)
	if err != nil {
		return err
	}
	repositoryPath := planningRouteDecisionCheckpointRepositoryPath(checkpoint.RunID, checkpoint.Pass)
	return persistPlanningScoutFiles(root, "planning-route-decision-"+checkpoint.ContentHash[:24], "planning-route-decision", checkpoint.ID, map[string][]byte{repositoryPath: content}, nil)
}

func loadPlanningRouteDecisionCheckpoint(root, runID string, pass int) (planningScoutDecisionCheckpoint, error) {
	var checkpoint planningScoutDecisionCheckpoint
	repositoryRoot, _, err := planningStageRoots(root)
	if err != nil {
		return checkpoint, err
	}
	content, exists, err := readOptionalPlanningStageFile(repositoryRoot, planningRouteDecisionCheckpointRepositoryPath(runID, pass))
	if err != nil {
		return checkpoint, err
	}
	if !exists {
		return checkpoint, fmt.Errorf("completed-Route decision checkpoint for %q pass %d is missing: %w", runID, pass, os.ErrNotExist)
	}
	if err := decodePlanningStageJSON(content, &checkpoint); err != nil {
		return planningScoutDecisionCheckpoint{}, err
	}
	originalID, originalHash := checkpoint.ID, checkpoint.ContentHash
	if err := addressPlanningScoutDecisionCheckpoint(&checkpoint); err != nil {
		return planningScoutDecisionCheckpoint{}, err
	}
	if checkpoint.ID != originalID || checkpoint.ContentHash != originalHash || checkpoint.RunID != strings.TrimSpace(runID) || checkpoint.Pass != pass || !planningSHA256Pattern.MatchString(checkpoint.CompletedCardHash) {
		return planningScoutDecisionCheckpoint{}, fmt.Errorf("completed-Route decision checkpoint is not a valid card-bound content address")
	}
	return checkpoint, nil
}

func resumePlanningRouteDecision(root, runID string, token planningDecisionResumeToken, resolvedAt time.Time) (planningRouteStageCoordination, error) {
	empty := planningRouteStageCoordination{}
	state, err := loadPlanningStageState(root, runID)
	if err != nil {
		return empty, err
	}
	checkpoint, err := loadPlanningRouteDecisionCheckpoint(root, runID, state.Pass)
	if err != nil {
		return empty, err
	}
	answers := make([]planningScoutDecisionAnswer, 0, len(token.Answers))
	for _, answer := range token.Answers {
		answers = append(answers, planningScoutDecisionAnswer{DecisionID: answer.DecisionID, ChoiceID: answer.ChoiceID, Answer: answer.Answer})
	}
	expectedToken, err := buildPlanningScoutDecisionResumeToken(checkpoint, answers)
	if err != nil {
		return empty, err
	}
	validation, err := validatePlanningDecisionResumeToken(token, planningDecisionResumeBindingFromToken(expectedToken))
	if err != nil {
		return empty, err
	}
	if !validation.Accepted {
		return empty, fmt.Errorf("Route decision resume token is %s; recover with %s", validation.Status, validation.RecoveryCommand)
	}
	result := planningRouteStageCoordination{DecisionCheckpoint: &checkpoint, ResumeToken: &expectedToken}
	if state.Stage == planningStageScoutRunning {
		dispatch, loadErr := loadPlanningRouteScoutDispatch(root, runID, state.Pass)
		if loadErr != nil {
			return empty, loadErr
		}
		if dispatch.ResumeTokenHash != expectedToken.ContentHash {
			return empty, fmt.Errorf("Scout already resumed from a different Route decision token")
		}
		result.ScoutDispatch = &dispatch
		return result, nil
	}
	if state.Stage == planningStageSpecApprovalRequired && expectedToken.Disposition == planningDecisionDispositionSuccessorSpecRequired {
		colonyState, loadErr := loadSpecificationColonyState(root)
		if loadErr != nil {
			return empty, loadErr
		}
		if state.PendingSpecification == nil || colonyState.Specification == nil {
			return empty, fmt.Errorf("successor specification checkpoint is incomplete")
		}
		current, ok := currentSpecificationRevision(*colonyState.Specification)
		if !ok || current.ID != state.PendingSpecification.RevisionID || current.ContentHash != state.PendingSpecification.ContentHash || current.Status != colony.SpecStatusDraft {
			return empty, fmt.Errorf("persisted successor specification does not match the exact Route checkpoint")
		}
		result.SuccessorSpecification = &specificationMutationResult{Specification: *colonyState.Specification, Revision: current}
		return result, nil
	}
	if state.Stage != planningStageOwnerDecision || state.PriorCardHash != checkpoint.CompletedCardHash {
		return empty, fmt.Errorf("Route decision token does not match the current completed-card owner boundary")
	}
	resolution := planningDecisionResolution{
		Disposition: expectedToken.Disposition, AffectedSemanticIDs: append([]string(nil), expectedToken.AffectedSemanticIDs...),
		RevisionEvidence: append([]planningDecisionRevisionEvidence(nil), expectedToken.RevisionEvidence...),
	}
	if resolution.Disposition == planningDecisionDispositionSuccessorSpecRequired {
		resumed, resumeErr := resumePlanningScoutContractDecision(root, state, checkpoint, expectedToken, resolution, resolvedAt)
		if resumeErr != nil {
			return empty, resumeErr
		}
		result.SuccessorSpecification = resumed.SuccessorSpecification
		return result, nil
	}
	ready, stageManifest, err := reducePlanningStage(state, planningStageTransition{To: planningStageScoutReady, DecisionResolution: &resolution})
	if err != nil {
		return empty, fmt.Errorf("resume planning after completed-Route owner answer: %w", err)
	}
	if stageManifest != nil {
		return empty, fmt.Errorf("completed-Route owner answer unexpectedly dispatched a worker")
	}
	card, err := planningRouteCheckpointCard(root, checkpoint)
	if err != nil {
		return empty, err
	}
	dispatch, err := authorizePlanningRouteScout(root, ready, card, checkpoint.ProposalHash, checkpoint.Evidence, &expectedToken, true)
	if err != nil {
		return empty, err
	}
	result.ScoutDispatch = &dispatch
	return result, nil
}

func planningRouteCheckpointCard(root string, checkpoint planningScoutDecisionCheckpoint) (colony.PlanningIterationCard, error) {
	timeline, err := loadPlanningTimeline(root, checkpoint.RunID)
	if err != nil {
		return colony.PlanningIterationCard{}, err
	}
	for _, card := range timeline.Cards {
		if card.Iteration == checkpoint.Pass && card.ContentHash == checkpoint.CompletedCardHash {
			return card, nil
		}
	}
	return colony.PlanningIterationCard{}, fmt.Errorf("Route decision checkpoint completed card is absent from the verified timeline")
}

func planningRouteCandidateRepositoryPath(runID string) string {
	return path.Join(".aether", "data", "planning", strings.TrimSpace(runID), "candidate.json")
}

func buildPlanningRouteCandidate(root string, completed planningRouteStageFinalization) (colony.PlanCandidate, error) {
	empty := colony.PlanCandidate{}
	if completed.Card.Decision.Reason == colony.PlanningStopContinue || completed.Card.Decision.Reason == colony.PlanningStopOwnerDecision {
		return empty, fmt.Errorf("planning stop %q is not candidate eligible", completed.Card.Decision.Reason)
	}
	timeline, err := loadPlanningTimeline(root, completed.Receipt.RunID)
	if err != nil {
		return empty, err
	}
	if timeline.Binding == nil || timeline.Index == nil || timeline.Binding.LastCardHash != completed.Card.ContentHash {
		return empty, fmt.Errorf("candidate requires the complete verified planning timeline")
	}
	state, err := loadSpecificationColonyState(root)
	if err != nil {
		return empty, err
	}
	base, _, err := candidateAcceptanceBase(state.Plan, completed.Card.CreatedAt)
	if err != nil {
		return empty, err
	}
	residual := make([]colony.PlanningGap, 0, len(completed.Validation.Confidence.RankedGaps))
	causal := make([]string, 0, len(completed.Validation.Confidence.RankedGaps))
	for _, gap := range completed.Validation.Confidence.RankedGaps {
		if gap.Materiality == colony.PlanningGapMaterial {
			return empty, fmt.Errorf("material residual gap %q requires owner decision instead of candidate persistence", gap.ID)
		}
		residual = append(residual, clonePlanningConfidenceGap(gap))
		causal = append(causal, strings.TrimSpace(gap.EvidenceThatWouldChange))
	}
	causal = uniqueSortedStrings(causal)
	if len(causal) == 0 {
		return empty, fmt.Errorf("candidate requires residual evidence_that_would_change")
	}
	proposalInput, preservedPrefix, err := planningRouteCandidateProposalInput(state.Plan, completed.Validation.Result.Proposal.Phases)
	if err != nil {
		return empty, err
	}
	phases := planningRouteCandidatePhases(proposalInput, preservedPrefix, "", "", completed.Validation.Result.Specification, *timeline.Binding, completed.Validation.SemanticDelta)
	number := base.Number + 1
	parentID := base.ID
	if base.ID == "plan-unbound" {
		number = 1
		parentID = ""
	}
	affected, preserved := planningRouteDeltaSemanticIDs(completed.Validation.SemanticDelta)
	requirements, acceptance, negative, recovery, publicPaths := planningRouteProposalProofLinks(phases)
	evidenceHash, err := jsonSHA256(completed.Validation.Evidence)
	if err != nil {
		return empty, err
	}
	proposal := colony.PlanRevision{
		SchemaVersion: planRevisionSchemaVersion, Number: number, ParentID: parentID,
		CreatedAt: completed.Card.CreatedAt.UTC().Format(time.RFC3339Nano), ReasonType: colony.PlanRevisionResearch,
		Reason: "Evidence-backed iterative planning candidate", EvidenceHash: evidenceHash,
		PlanningRunID:     completed.Receipt.RunID,
		PreservedPhaseIDs: phaseIDs(state.Plan.Phases[:preservedPrefix]), SupersededPhaseIDs: phaseIDs(state.Plan.Phases[preservedPrefix:]), ReplacementPhaseIDs: phaseIDs(phases[preservedPrefix:]),
		SemanticID:            completed.Validation.Result.Proposal.SemanticID,
		RequirementProofLinks: requirements, AcceptanceProofLinks: acceptance, NegativeProofLinks: negative, RecoveryProofLinks: recovery, PublicPathProofLinks: publicPaths,
		SpecificationRevisionID: completed.Validation.Result.Specification.RevisionID, SpecificationRevisionHash: completed.Validation.Result.Specification.ContentHash,
		PlanningTimelineID: timeline.Binding.ID, PlanningTimelineDigest: timeline.Binding.TimelineDigest,
		AffectedSemanticIDs: affected, PreservedSemanticIDs: preserved, Phases: phases,
	}
	disposition := colony.PlanRecommendationRevise
	if completed.Card.Decision.Reason == colony.PlanningStopTargetMet {
		disposition = colony.PlanRecommendationAccept
	}
	evidenceIDs := append([]string(nil), completed.Card.EvidenceIDs...)
	if len(evidenceIDs) == 0 {
		return empty, fmt.Errorf("Queen recommendation requires planning evidence")
	}
	rationale := fmt.Sprintf("Go-derived confidence %d%% stopped as %s against the %d%% target; %d non-material gap(s) remain.", completed.Validation.Confidence.Scores.Overall, completed.Card.Decision.Reason, completed.Validation.RunHeader.TargetConfidence, len(residual))
	recommendation := colony.QueenPlanRecommendation{
		SchemaVersion: colony.PlanningSchemaVersion, Disposition: disposition,
		EvidenceIDs: uniqueSortedStrings(evidenceIDs), Rationale: rationale, Producer: colony.PlanRecommendationProducerQueen,
		ProducerID: "go-queen/planning-route/v1", CreatedAt: completed.Card.CreatedAt.UTC(),
	}
	candidate := colony.PlanCandidate{
		SchemaVersion: colony.PlanCandidateSchemaVersion,
		Status:        colony.PlanCandidatePendingReview, CreatedAt: completed.Card.CreatedAt.UTC(), ExpiresAt: completed.Card.CreatedAt.UTC().Add(7 * 24 * time.Hour),
		Proposal:           proposal,
		BasePlanRevisionID: base.ID, BasePlanRevisionHash: base.Hash,
		SpecificationRevisionID: completed.Validation.Result.Specification.RevisionID, SpecificationRevisionHash: completed.Validation.Result.Specification.ContentHash,
		Timeline: *timeline.Binding, StopDecision: completed.Card.Decision,
		DimensionAssessments: append([]colony.PlanningDimensionAssessment(nil), completed.Card.DimensionAssessments...),
		SemanticDelta:        completed.Card.SemanticDelta, ResidualGaps: residual,
		EvidenceThatWouldChange: strings.Join(causal, "; "), Recommendation: recommendation,
	}
	if err := addressPlanCandidateReviewPayload(&candidate); err != nil {
		return empty, err
	}
	if err := validatePlanningRecordHashes(candidate); err != nil {
		return empty, err
	}
	if err := candidate.Validate(); err != nil {
		return empty, err
	}
	return candidate, nil
}

// planningRouteCandidateProposalInput retains the immutable completed prefix
// even when Route-Setter returns only replacement work. If Route-Setter echoes
// that prefix, the authoritative completed copies win and acceptance later
// restores their lifecycle statuses without changing proposal identity.
func planningRouteCandidateProposalInput(plan colony.Plan, input []colony.Phase) ([]colony.Phase, int, error) {
	prefix, err := completedPlanPrefix(plan.Phases)
	if err != nil {
		return nil, 0, err
	}
	if prefix == 0 {
		return clonePhases(input), 0, nil
	}
	suffix := input
	if len(input) >= prefix {
		echoesPrefix := true
		for index := 0; index < prefix; index++ {
			if strings.TrimSpace(input[index].SemanticID) == "" || input[index].SemanticID != plan.Phases[index].SemanticID {
				echoesPrefix = false
				break
			}
		}
		if echoesPrefix {
			suffix = input[prefix:]
		}
	}
	result := clonePhases(plan.Phases[:prefix])
	result = append(result, clonePhases(suffix)...)
	return result, prefix, nil
}

func planningRouteCandidatePhases(input []colony.Phase, preservedPrefix int, candidateID, candidateHash string, specification planningStageSpecificationBinding, timeline colony.PlanningTimelineBinding, delta colony.PlanningSemanticDelta) []colony.Phase {
	phases := renumberRevisionPhases(input, 0)
	if preservedPrefix > 0 && preservedPrefix < len(phases) {
		phases[preservedPrefix].Status = colony.PhaseReady
	}
	affected, preserved := planningRouteDeltaSemanticIDs(delta)
	affectedSet := make(map[string]struct{}, len(affected))
	preservedSet := make(map[string]struct{}, len(preserved))
	for _, id := range affected {
		affectedSet[id] = struct{}{}
	}
	for _, id := range preserved {
		preservedSet[id] = struct{}{}
	}
	bind := func(semanticID string) ([]string, []string) {
		var nodeAffected, nodePreserved []string
		if _, ok := affectedSet[semanticID]; ok {
			nodeAffected = []string{semanticID}
		}
		if _, ok := preservedSet[semanticID]; ok {
			nodePreserved = []string{semanticID}
		}
		return nodeAffected, nodePreserved
	}
	for phaseIndex := range phases {
		phase := &phases[phaseIndex]
		phase.SpecificationRevisionID, phase.SpecificationRevisionHash = specification.RevisionID, specification.ContentHash
		phase.CandidateID, phase.CandidateContentHash = candidateID, candidateHash
		phase.PlanningTimelineID, phase.PlanningTimelineDigest = timeline.ID, timeline.TimelineDigest
		phase.AffectedSemanticIDs, phase.PreservedSemanticIDs = bind(phase.SemanticID)
		for taskIndex := range phase.Tasks {
			task := &phase.Tasks[taskIndex]
			task.SpecificationRevisionID, task.SpecificationRevisionHash = specification.RevisionID, specification.ContentHash
			task.CandidateID, task.CandidateContentHash = candidateID, candidateHash
			task.PlanningTimelineID, task.PlanningTimelineDigest = timeline.ID, timeline.TimelineDigest
			task.AffectedSemanticIDs, task.PreservedSemanticIDs = bind(task.SemanticID)
		}
	}
	return phases
}

func planningRouteDeltaSemanticIDs(delta colony.PlanningSemanticDelta) ([]string, []string) {
	var affected, preserved []string
	for _, section := range [][]colony.PlanningSemanticChange{delta.Phases, delta.Tasks, delta.Dependencies, delta.RequirementLinks, delta.AcceptanceChecks, delta.NegativeExpectations, delta.RecoveryExpectations, delta.PublicPaths} {
		for _, change := range section {
			if change.Kind == colony.PlanningSemanticChangePreserved {
				preserved = append(preserved, change.SemanticID)
			} else {
				affected = append(affected, change.SemanticID)
			}
		}
	}
	return uniqueSortedStrings(affected), uniqueSortedStrings(preserved)
}

func planningRouteProposalProofLinks(phases []colony.Phase) ([]string, []string, []string, []string, []string) {
	var requirements, acceptance, negative, recovery, publicPaths []string
	for _, phase := range phases {
		requirements = append(requirements, phase.RequirementProofLinks...)
		acceptance = append(acceptance, phase.AcceptanceProofLinks...)
		negative = append(negative, phase.NegativeProofLinks...)
		recovery = append(recovery, phase.RecoveryProofLinks...)
		publicPaths = append(publicPaths, phase.PublicPathProofLinks...)
		for _, task := range phase.Tasks {
			requirements = append(requirements, task.RequirementProofLinks...)
			acceptance = append(acceptance, task.AcceptanceProofLinks...)
			negative = append(negative, task.NegativeProofLinks...)
			recovery = append(recovery, task.RecoveryProofLinks...)
			publicPaths = append(publicPaths, task.PublicPathProofLinks...)
		}
	}
	return uniqueSortedStrings(requirements), uniqueSortedStrings(acceptance), uniqueSortedStrings(negative), uniqueSortedStrings(recovery), uniqueSortedStrings(publicPaths)
}

func persistPlanningRouteCandidate(root string, candidate colony.PlanCandidate) error {
	content, err := marshalPlanningStageJSON(candidate)
	if err != nil {
		return err
	}
	repositoryPath := planningRouteCandidateRepositoryPath(candidate.Timeline.RunID)
	return persistPlanningScoutFiles(root, "planning-route-candidate-"+candidate.ContentHash[:24], "planning-route-candidate", candidate.ID, map[string][]byte{repositoryPath: content}, nil)
}
