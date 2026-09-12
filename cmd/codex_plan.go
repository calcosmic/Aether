package cmd

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
)

type codexPlanningDispatch struct {
	Stage             string                   `json:"stage,omitempty"`
	Wave              int                      `json:"wave,omitempty"`
	Caste             string                   `json:"caste"`
	AgentName         string                   `json:"agent_name,omitempty"`
	Name              string                   `json:"name"`
	Task              string                   `json:"task"`
	TaskID            string                   `json:"task_id,omitempty"`
	Outputs           []string                 `json:"outputs"`
	Status            string                   `json:"status"`
	Summary           string                   `json:"summary,omitempty"`
	Blockers          []string                 `json:"blockers,omitempty"`
	Duration          float64                  `json:"duration,omitempty"` // Wall-clock seconds (0 = not measured)
	Brief             string                   `json:"brief,omitempty"`
	FilesCreated      []string                 `json:"files_created,omitempty"`
	FilesModified     []string                 `json:"files_modified,omitempty"`
	ScoutReport       *codexScoutReport        `json:"scout_report,omitempty"`
	PhasePlan         *codexWorkerPlanArtifact `json:"phase_plan,omitempty"`
	SkillSection      string                   `json:"skill_section,omitempty"`
	SkillCount        int                      `json:"skill_count,omitempty"`
	ColonySkills      int                      `json:"colony_skill_count,omitempty"`
	DomainSkills      int                      `json:"domain_skill_count,omitempty"`
	MatchedSkills     []string                 `json:"matched_skills,omitempty"`
	StageManifest     *planningStageManifest   `json:"stage_manifest,omitempty"`
	Claimed           []string                 `json:"-"`
	PermissionProfile codex.PermissionProfile  `json:"permission_profile"`
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
	SourceAnchors    []string
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
	Knowledge    planScore `json:"knowledge"`
	Requirements planScore `json:"requirements"`
	Risks        planScore `json:"risks"`
	Dependencies planScore `json:"dependencies"`
	Effort       planScore `json:"effort"`
	Overall      planScore `json:"overall"`
}

// planScore is a 0-100 confidence value that also accepts the 0-1 fractional
// scale emitted by the legacy whole-plan worker contract. The staged planning
// contract has a separate strict whole-number decoder because changing this
// compatibility type would reject already-supported phase-plan artifacts.
type planScore int

func (s *planScore) UnmarshalJSON(data []byte) error {
	var num json.Number
	if err := json.Unmarshal(data, &num); err != nil {
		return fmt.Errorf("confidence must be a number: %w", err)
	}
	text := num.String()
	if !strings.ContainsAny(text, ".eE") {
		value, err := num.Int64()
		if err != nil {
			return err
		}
		*s = planScore(value)
		return nil
	}
	value, err := num.Float64()
	if err != nil {
		return err
	}
	if value <= 1.0 {
		value *= 100
	}
	*s = planScore(math.Round(value))
	return nil
}

type codexWorkerPlanArtifact struct {
	Phases       []codexWorkerPlanPhase `json:"phases"`
	Confidence   codexPlanConfidence    `json:"confidence"`
	Gaps         []string               `json:"gaps,omitempty"`
	PlanningLoop *codexPlanningLoop     `json:"planning_loop,omitempty"`
}

type codexWorkerPlanPhase struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	// Mode is the phase's declared work mode (discovery, prototype,
	// production, maintenance). The Route-Setter states it explicitly; prose
	// keywords never decide it at runtime. A phase description containing the
	// word "research" once silently flipped an implementation phase to
	// discovery and dispatched an Oracle instead of a Builder.
	Mode string `json:"mode,omitempty"`
	// ExpectFailingTests marks a deliberately-RED phase whose deliverable is
	// failing tests that prove a defect. The Route-Setter states it
	// explicitly at authoring time; continue's verification then treats a
	// failing test run as the expected outcome and a green one as the
	// blocker. Without this the planner emitted RED-first phases the gate
	// could never advance.
	ExpectFailingTests   bool                                  `json:"expect_failing_tests,omitempty"`
	Tasks                []codexWorkerPlanTask                 `json:"tasks"`
	SuccessCriteria      []string                              `json:"success_criteria,omitempty"`
	EvidenceRequirements []colony.CriterionEvidenceRequirement `json:"evidence_requirements,omitempty"`
}

type codexWorkerPlanTask struct {
	Goal                 string                                `json:"goal"`
	Constraints          []string                              `json:"constraints,omitempty"`
	Hints                []string                              `json:"hints,omitempty"`
	SuccessCriteria      []string                              `json:"success_criteria,omitempty"`
	EvidenceRequirements []colony.CriterionEvidenceRequirement `json:"evidence_requirements,omitempty"`
	DependsOn            []string                              `json:"depends_on,omitempty"`
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
	Refresh             bool
	Synthetic           bool
	PlanOnly            bool
	Preset              string
	PresetSet           bool
	Depth               string
	DepthSet            bool
	PlanningDepth       string
	VerificationDepth   string
	WorkerTimeout       time.Duration
	TargetConfidence    int
	TargetConfidenceSet bool
	MaxIterations       int
	MaxIterationsSet    bool
	Accept              bool
	RepairArtifact      bool
	RevisionType        string
	RevisionReason      string
	RevisionEvidence    []string
	ResearchDocs        []string
	Territory           *SurveyFreshnessResult
	RequireTerritory    bool
}

const (
	planningPresetSourceRequired     = "owner_required"
	planningPresetSourceNamed        = "named_preset"
	planningPresetSourceExplicitPair = "explicit_pair"
)

type planningPresetPolicy struct {
	ID               planningStagePreset `json:"id"`
	Label            string              `json:"label"`
	TargetConfidence int                 `json:"target"`
	PassCap          int                 `json:"max_iterations"`
}

type planningPresetSelection struct {
	PresetRequired  bool                   `json:"preset_required"`
	Options         []planningPresetPolicy `json:"preset_options"`
	Policy          planningPresetPolicy   `json:"policy,omitempty"`
	SelectionSource string                 `json:"selection_source"`
}

var planningPresetPolicies = []planningPresetPolicy{
	{ID: planningStagePresetFast, Label: "Fast", TargetConfidence: 80, PassCap: 4},
	{ID: planningStagePresetBalanced, Label: "Balanced", TargetConfidence: 90, PassCap: 6},
	{ID: planningStagePresetDeep, Label: "Deep", TargetConfidence: 95, PassCap: 8},
	{ID: planningStagePresetExhaustive, Label: "Exhaustive", TargetConfidence: 99, PassCap: 12},
}

func resolvePlanningPreset(opts codexPlanOptions) (planningPresetSelection, error) {
	options := append([]planningPresetPolicy(nil), planningPresetPolicies...)
	namedValue := strings.TrimSpace(opts.Preset)
	namedSet := opts.PresetSet || namedValue != ""
	depthValue := strings.TrimSpace(opts.Depth)
	depthSet := opts.DepthSet || depthValue != ""

	var named planningPresetPolicy
	if namedSet {
		var ok bool
		named, ok = planningPresetPolicyByName(namedValue, false)
		if !ok {
			return planningPresetSelection{}, fmt.Errorf("invalid planning preset %q: choose fast, balanced, deep, or exhaustive", namedValue)
		}
	}
	if depthSet {
		legacy, ok := planningPresetPolicyByName(depthValue, true)
		if !ok {
			return planningPresetSelection{}, fmt.Errorf("invalid planning depth %q: choose fast, balanced, deep, or exhaustive", depthValue)
		}
		if namedSet && legacy.ID != named.ID {
			return planningPresetSelection{}, fmt.Errorf("conflicting planning preset flags: --preset %s does not match --depth %s", named.ID, legacy.ID)
		}
		if !namedSet {
			named = legacy
			namedSet = true
		}
	}

	targetSet := opts.TargetConfidenceSet || opts.TargetConfidence != 0
	capSet := opts.MaxIterationsSet || opts.MaxIterations != 0
	if targetSet != capSet {
		return planningPresetSelection{}, fmt.Errorf("explicit planning bounds require both --target and --max-iterations")
	}

	var explicit planningPresetPolicy
	if targetSet {
		if opts.TargetConfidence < planningLoopMinTarget || opts.TargetConfidence > planningLoopMaxTarget {
			return planningPresetSelection{}, fmt.Errorf("--target must be between %d and %d", planningLoopMinTarget, planningLoopMaxTarget)
		}
		if opts.MaxIterations < planningLoopMinIterations || opts.MaxIterations > planningLoopMaxIterations {
			return planningPresetSelection{}, fmt.Errorf("--max-iterations must be between %d and %d", planningLoopMinIterations, planningLoopMaxIterations)
		}
		var ok bool
		explicit, ok = planningPresetPolicyByBounds(opts.TargetConfidence, opts.MaxIterations)
		if !ok {
			return planningPresetSelection{}, fmt.Errorf("explicit planning bounds %d/%d do not match an exact preset: use 80/4, 90/6, 95/8, or 99/12", opts.TargetConfidence, opts.MaxIterations)
		}
	}

	if namedSet && targetSet && (named.TargetConfidence != explicit.TargetConfidence || named.PassCap != explicit.PassCap) {
		return planningPresetSelection{}, fmt.Errorf("conflicting planning preset and explicit bounds: %s is %d/%d, not %d/%d", named.ID, named.TargetConfidence, named.PassCap, explicit.TargetConfidence, explicit.PassCap)
	}
	if namedSet {
		return planningPresetSelection{Options: options, Policy: named, SelectionSource: planningPresetSourceNamed}, nil
	}
	if targetSet {
		return planningPresetSelection{Options: options, Policy: explicit, SelectionSource: planningPresetSourceExplicitPair}, nil
	}
	return planningPresetSelection{PresetRequired: true, Options: options, SelectionSource: planningPresetSourceRequired}, nil
}

func planningPresetPolicyByName(value string, aliases bool) (planningPresetPolicy, bool) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if aliases {
		switch normalized {
		case "quick", "light", string(colony.GranularitySprint):
			normalized = string(planningStagePresetFast)
		case "standard", "default", string(colony.GranularityMilestone):
			normalized = string(planningStagePresetBalanced)
		case string(colony.GranularityQuarter):
			normalized = string(planningStagePresetDeep)
		case "full", string(colony.GranularityMajor):
			normalized = string(planningStagePresetExhaustive)
		}
	}
	for _, policy := range planningPresetPolicies {
		if string(policy.ID) == normalized {
			return policy, true
		}
	}
	return planningPresetPolicy{}, false
}

func planningPresetPolicyByBounds(target, passCap int) (planningPresetPolicy, bool) {
	for _, policy := range planningPresetPolicies {
		if policy.TargetConfidence == target && policy.PassCap == passCap {
			return policy, true
		}
	}
	return planningPresetPolicy{}, false
}

func planningPresetRequiredResult(state colony.ColonyState, selection planningPresetSelection) map[string]interface{} {
	goal := ""
	if state.Goal != nil {
		goal = strings.TrimSpace(*state.Goal)
	}
	answer := nextActionForCandidateOverride(
		state,
		"plan",
		candidatePlanPreset,
		"Choose Fast, Balanced, Deep, or Exhaustive before planning starts.",
		"<name>",
	)
	result := map[string]interface{}{
		"planned":          false,
		"status":           string(planningStagePresetRequired),
		"goal":             goal,
		"preset_required":  true,
		"preset_options":   append([]planningPresetPolicy(nil), selection.Options...),
		"selected_preset":  "",
		"selection_source": selection.SelectionSource,
		"dispatches":       []codexPlanningDispatch{},
		"dispatch_count":   0,
		"state_effect":     "unchanged",
		"next":             nextActionPrimarySuggestion(answer),
	}
	applyNextActionToResult(result, answer)
	return result
}

type codexPlanningLoop struct {
	TargetConfidence    int                       `json:"target_confidence"`
	MaxIterations       int                       `json:"max_iterations"`
	StallThreshold      int                       `json:"stall_threshold"`
	StallLimit          int                       `json:"stall_limit"`
	Accept              bool                      `json:"accept,omitempty"`
	Iterations          int                       `json:"iterations"`
	StopReason          string                    `json:"stop_reason"`
	FinalConfidence     int                       `json:"final_confidence"`
	AcceptedBelowTarget bool                      `json:"accepted_below_target,omitempty"`
	Gaps                []string                  `json:"gaps,omitempty"`
	History             []codexPlanningLoopSample `json:"history,omitempty"`
}

type codexPlanningLoopSample struct {
	Iteration     int      `json:"iteration"`
	Confidence    int      `json:"confidence"`
	Delta         int      `json:"delta"`
	StallCount    int      `json:"stall_count"`
	Gaps          []string `json:"gaps,omitempty"`
	SelectedGaps  []string `json:"selected_gaps,omitempty"`
	Evidence      string   `json:"evidence,omitempty"`
	EvidenceCount int      `json:"evidence_count,omitempty"`
	EvidenceHash  string   `json:"evidence_hash,omitempty"`
}

type codexPlanManifest struct {
	Goal                      string                             `json:"goal"`
	Root                      string                             `json:"root"`
	GeneratedAt               string                             `json:"generated_at"`
	BaseRevisionID            string                             `json:"base_revision_id,omitempty"`
	BasePlanStateHash         string                             `json:"base_plan_state_hash"`
	ColonyMode                string                             `json:"colony_mode,omitempty"`
	Synthetic                 bool                               `json:"synthetic,omitempty"`
	SyntheticWarning          string                             `json:"synthetic_warning,omitempty"`
	PlanningRunID             string                             `json:"planning_run_id,omitempty"`
	Iteration                 int                                `json:"iteration,omitempty"`
	TargetConfidence          int                                `json:"target_confidence,omitempty"`
	MaxIterations             int                                `json:"max_iterations,omitempty"`
	SelectedPreset            planningStagePreset                `json:"selected_preset,omitempty"`
	PresetSelectionSource     string                             `json:"selection_source,omitempty"`
	PreviousConfidence        int                                `json:"previous_confidence,omitempty"`
	PreviousEvidenceHash      string                             `json:"previous_evidence_hash,omitempty"`
	SelectedGaps              []string                           `json:"selected_gaps,omitempty"`
	PreviousPlanDraft         *codexWorkerPlanArtifact           `json:"previous_plan_draft,omitempty"`
	ExpectedWorkers           []codexPlanningDispatch            `json:"expected_workers,omitempty"`
	Refresh                   bool                               `json:"refresh"`
	Revision                  *codexPlanRevisionContext          `json:"revision,omitempty"`
	ExistingPlan              bool                               `json:"existing_plan"`
	ExistingPhaseCount        int                                `json:"existing_phase_count,omitempty"`
	Depth                     string                             `json:"depth"`
	Granularity               string                             `json:"granularity"`
	GranularityMin            int                                `json:"granularity_min"`
	GranularityMax            int                                `json:"granularity_max"`
	PlanningDepth             string                             `json:"planning_depth"`
	VerificationDepth         string                             `json:"verification_depth,omitempty"`
	PlanningLoop              codexPlanningLoop                  `json:"planning_loop,omitempty"`
	Survey                    codexSurveyContext                 `json:"survey"`
	Territory                 *SurveyFreshnessResult             `json:"territory,omitempty"`
	TerritoryRequired         bool                               `json:"territory_required,omitempty"`
	Dispatches                []codexPlanningDispatch            `json:"dispatches"`
	Snapshots                 map[string]codexArtifactSnapshot   `json:"snapshots,omitempty"`
	DispatchMode              string                             `json:"dispatch_mode"`
	DispatchContract          map[string]interface{}             `json:"dispatch_contract"`
	FinalizeSurface           string                             `json:"finalize_surface"`
	RequiresFinalizer         bool                               `json:"requires_finalizer"`
	BoundaryQuestions         []discussQuestion                  `json:"boundary_questions,omitempty"`
	BoundaryQuestionCount     int                                `json:"boundary_question_count,omitempty"`
	BoundaryQuestionsCreated  int                                `json:"boundary_questions_created,omitempty"`
	BoundaryQuestionsExisting int                                `json:"boundary_questions_existing,omitempty"`
	OrchestratorGuidance      *orchestratorBoundaryGuidance      `json:"orchestrator_boundary_guidance,omitempty"`
	ResearchPolicy            *phaseResearchAutomaticPolicy      `json:"research_policy,omitempty"`
	PhaseResearchEvidence     []phaseResearchEvidenceAttribution `json:"phase_research_evidence,omitempty"`
	StageManifest             *planningStageManifest             `json:"stage_manifest,omitempty"`
	PlanningRunHeader         *planningRunHeader                 `json:"planning_run_header,omitempty"`
	// ContextCapsule is the colony-prime grounding payload (state, decisions,
	// phase learnings, instincts, hive wisdom, prior reviews, blockers, user
	// preferences) for wrapper-spawned planning workers (the Route-Setter and
	// planning Scout). It is computed once per plan-only manifest, carried
	// here at the top level rather than copied into every dispatch brief, and
	// the wrapper reads it once and prepends it, verbatim, to each spawned
	// worker's prompt. Only populated on this plan-only manifest path — the
	// in-process/native lane (dispatchRealPlanningWorkersWithIterationContext)
	// already computes and shares its own capsule via
	// codex.WorkerDispatch.ContextCapsule, and that lane never builds this
	// struct.
	ContextCapsule string `json:"context_capsule,omitempty"`
}

const planningRunHeaderSchemaVersion = "planning-run/v1"

// planningRunHeader is the durable, authority-neutral identity of one staged
// planning run. It snapshots only safe evidence catalogue projections; raw
// source bodies stay with their owning stores and paths.
type planningRunHeader struct {
	SchemaVersion         string                             `json:"schema_version"`
	ID                    string                             `json:"id"`
	ContentHash           string                             `json:"content_hash"`
	RunID                 string                             `json:"run_id"`
	Goal                  string                             `json:"goal"`
	GoalID                string                             `json:"goal_id"`
	SessionID             string                             `json:"session_id"`
	Specification         planningStageSpecificationBinding  `json:"specification"`
	BasePlanRevisionID    string                             `json:"base_plan_revision_id"`
	BasePlanRevisionHash  string                             `json:"base_plan_revision_hash"`
	Preset                planningStagePreset                `json:"preset"`
	TargetConfidence      int                                `json:"target_confidence"`
	PassCap               int                                `json:"pass_cap"`
	EvidenceCatalogue     []planningEvidenceRecord           `json:"evidence_catalogue"`
	EvidenceFrontier      []planningStageEvidenceBinding     `json:"evidence_frontier"`
	InputFrontierHash     string                             `json:"input_frontier_hash"`
	MissingEvidenceKinds  []colony.PlanningEvidenceKind      `json:"missing_evidence_kinds,omitempty"`
	ResearchPolicy        phaseResearchAutomaticPolicy       `json:"research_policy"`
	PhaseResearchEvidence []phaseResearchEvidenceAttribution `json:"phase_research_evidence,omitempty"`
	WeakestGap            colony.PlanningGap                 `json:"weakest_gap"`
	StageManifestID       string                             `json:"stage_manifest_id"`
	StageManifestHash     string                             `json:"stage_manifest_hash"`
	CreatedAt             time.Time                          `json:"created_at"`
}

type approvedPlanningSpecification struct {
	Specification colony.Specification
	Revision      colony.SpecRevision
	Binding       planningStageSpecificationBinding
	GoalID        string
	SessionID     string
}

type planningEvidencePrimingResult struct {
	Records               []planningEvidenceRecord
	Bindings              []planningStageEvidenceBinding
	MissingKinds          []colony.PlanningEvidenceKind
	PhaseResearchEvidence []phaseResearchEvidenceAttribution
	FrontierHash          string
}

type codexPlanIterationState struct {
	PlanningRunID        string                    `json:"planning_run_id"`
	Goal                 string                    `json:"goal"`
	Root                 string                    `json:"root"`
	Depth                string                    `json:"depth"`
	PlanningDepth        string                    `json:"planning_depth"`
	TargetConfidence     int                       `json:"target_confidence"`
	MaxIterations        int                       `json:"max_iterations"`
	LastIteration        int                       `json:"last_iteration"`
	PreviousConfidence   int                       `json:"previous_confidence"`
	PreviousEvidenceHash string                    `json:"previous_evidence_hash,omitempty"`
	SelectedGaps         []string                  `json:"selected_gaps,omitempty"`
	PreviousPlanDraft    *codexWorkerPlanArtifact  `json:"previous_plan_draft,omitempty"`
	History              []codexPlanningLoopSample `json:"history,omitempty"`
	ConsecutiveStalls    int                       `json:"consecutive_stalls,omitempty"`
	Revision             *codexPlanRevisionContext `json:"revision,omitempty"`
	UpdatedAt            string                    `json:"updated_at"`
}

const planningSyntheticWarning = "Synthetic planning was explicitly requested; output is local preview/test synthesis and does not prove provider-backed Scout or Route-Setter work."

func planningSyntheticWarningForMode(synthetic bool) string {
	if synthetic {
		return planningSyntheticWarning
	}
	return ""
}

func planningWorkersUnavailableError(invoker codex.WorkerInvoker) error {
	return fmt.Errorf("real planning workers unavailable: %s. Normal planning requires provider-backed Scout and Route-Setter work. Fix provider authentication or CLI availability, then rerun `aether plan`; use `aether plan --synthetic` only for an explicitly marked preview/test plan", dispatchAvailabilityMessage(invoker))
}

func planningWorkersFailedError(err error) error {
	if err == nil {
		return fmt.Errorf("real planning workers did not finish cleanly. Normal planning requires completed provider-backed Scout and Route-Setter work; rerun `aether plan` after fixing worker dispatch, or use `aether plan --synthetic` only for an explicitly marked preview/test plan")
	}
	return fmt.Errorf("real planning workers did not finish cleanly: %s. Normal planning requires completed provider-backed Scout and Route-Setter work; rerun `aether plan` after fixing worker dispatch, or use `aether plan --synthetic` only for an explicitly marked preview/test plan", err.Error())
}

func planningSpecificationRecoveryError(detail, command string) error {
	detail = strings.TrimSpace(detail)
	if detail == "" {
		detail = "an exact approved specification is required"
	}
	command = strings.TrimSpace(command)
	if command == "" {
		command = "aether spec"
	}
	return fmt.Errorf("planning did not start: %s. State is unchanged. Run `%s`", detail, command)
}

func requireApprovedPlanningSpecification(root string, state colony.ColonyState) (approvedPlanningSpecification, error) {
	empty := approvedPlanningSpecification{}
	if state.Specification == nil {
		return empty, planningSpecificationRecoveryError("an approved specification is missing", "aether spec")
	}
	if err := validateSpecificationState(*state.Specification); err != nil {
		return empty, planningSpecificationRecoveryError("the current specification is invalid: "+err.Error(), "aether spec")
	}
	revision, ok := currentSpecificationRevision(*state.Specification)
	if !ok {
		return empty, planningSpecificationRecoveryError("the current specification revision is missing", "aether spec")
	}
	if revision.Status != colony.SpecStatusApproved || revision.Approval == nil {
		return empty, planningSpecificationRecoveryError(fmt.Sprintf("specification revision %s is %s, not approved", revision.ID, revision.Status), "aether spec")
	}
	if revision.Approval.RevisionID != revision.ID || revision.Approval.RevisionContentHash != revision.ContentHash {
		return empty, planningSpecificationRecoveryError("the current specification approval hash does not match its revision", "aether spec")
	}
	if state.SessionID != nil && strings.TrimSpace(*state.SessionID) != "" && strings.TrimSpace(revision.Scope.SessionID) != strings.TrimSpace(*state.SessionID) {
		return empty, planningSpecificationRecoveryError("the approved specification belongs to a different colony session", "aether spec")
	}
	inspection, err := inspectSpecificationProjection(root)
	if err != nil {
		return empty, planningSpecificationRecoveryError("the readable specification projection cannot be verified: "+err.Error(), "aether spec --repair-projection")
	}
	if inspection.RevisionID != revision.ID || inspection.Drifted {
		return empty, planningSpecificationRecoveryError("the readable specification projection is missing or does not match the approved revision", "aether spec --repair-projection")
	}
	approvalHash, err := jsonSHA256(*revision.Approval)
	if err != nil {
		return empty, fmt.Errorf("hash specification approval receipt: %w", err)
	}
	return approvedPlanningSpecification{
		Specification: *state.Specification,
		Revision:      revision,
		Binding: planningStageSpecificationBinding{
			RevisionID:            revision.ID,
			ContentHash:           revision.ContentHash,
			PredecessorRevisionID: revision.PredecessorID,
			Status:                revision.Status,
			ApprovalReceiptID:     revision.Approval.ID,
			ApprovalReceiptHash:   approvalHash,
		},
		GoalID:    strings.TrimSpace(state.Specification.GoalID),
		SessionID: strings.TrimSpace(revision.Scope.SessionID),
	}, nil
}

func planningBaseRevisionIdentity(plan colony.Plan, planHash string) (string, string) {
	revisionID := strings.TrimSpace(activePlanRevisionID(plan))
	if revisionID == "" {
		revisionID = "plan-unbound"
	}
	return revisionID, strings.TrimSpace(planHash)
}

func planningEvidenceDimensions(kind colony.PlanningEvidenceKind) []colony.PlanningDimension {
	switch kind {
	case colony.PlanningEvidenceCharter:
		return []colony.PlanningDimension{colony.PlanningDimensionKnowledge, colony.PlanningDimensionRequirements, colony.PlanningDimensionRisks}
	case colony.PlanningEvidenceSurvey, colony.PlanningEvidenceContext, colony.PlanningEvidenceHive:
		return []colony.PlanningDimension{colony.PlanningDimensionKnowledge, colony.PlanningDimensionRisks, colony.PlanningDimensionDependencies, colony.PlanningDimensionEffort}
	default:
		return colony.PlanningDimensions()
	}
}

func planningEvidenceSourceRevision(prefix string, content []byte) string {
	return strings.TrimSpace(prefix) + "-" + planningEvidenceSHA256(content)[:16]
}

func planningSurveyHasEvidence(survey codexSurveyContext) bool {
	return len(survey.SurveyDocs)+len(survey.Languages)+len(survey.Frameworks)+len(survey.Directories)+len(survey.EntryPoints)+len(survey.Dependencies)+len(survey.TestFiles)+len(survey.Issues)+len(survey.SecurityPatterns)+len(survey.SourceAnchors) > 0
}

func planningOutcomeEvidence(state colony.ColonyState) []string {
	values := make([]string, 0)
	for _, event := range state.Events {
		lower := strings.ToLower(event)
		if strings.Contains(lower, "phase_completed") || strings.Contains(lower, "verification") || strings.Contains(lower, "seal_") {
			values = append(values, strings.TrimSpace(event))
		}
	}
	for _, phase := range state.Plan.Phases {
		if phase.Status == colony.PhaseCompleted {
			values = append(values, fmt.Sprintf("phase %d %s completed", phase.ID, strings.TrimSpace(phase.Name)))
		}
	}
	return uniqueSortedStrings(values)
}

func primePlanningStartEvidence(root string, state colony.ColonyState, approved approvedPlanningSpecification, survey codexSurveyContext, contextCapsule, baseRevisionID, runID string, observedAt time.Time) (planningEvidencePrimingResult, error) {
	result := planningEvidencePrimingResult{}
	scope := planningEvidenceScope{
		GoalID:                  approved.GoalID,
		SessionID:               approved.SessionID,
		SpecificationRevisionID: approved.Revision.ID,
		PlanRevisionID:          baseRevisionID,
	}
	sources := make([]planningEvidenceSource, 0, 10)
	present := make(map[colony.PlanningEvidenceKind]bool)
	addJSON := func(kind colony.PlanningEvidenceKind, origin, revision string, value interface{}) error {
		content, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("encode %s planning evidence: %w", kind, err)
		}
		if len(strings.TrimSpace(string(content))) == 0 || string(content) == "null" || string(content) == "[]" {
			return nil
		}
		if strings.TrimSpace(revision) == "" {
			revision = planningEvidenceSourceRevision(string(kind), content)
		}
		sources = append(sources, planningEvidenceSource{
			Kind: kind, Origin: origin, Content: content, Scope: scope, SourceRevision: revision,
			ObservedAt: observedAt, ApplicableDimensions: planningEvidenceDimensions(kind), State: planningEvidenceSourceCurrent,
		})
		present[kind] = true
		return nil
	}

	if err := addJSON(colony.PlanningEvidenceSpecification, "state:approved-specification", approved.Revision.ID, approved.Revision); err != nil {
		return result, err
	}
	if planningSurveyHasEvidence(survey) {
		if err := addJSON(colony.PlanningEvidenceSurvey, "survey:current", planningEvidenceSourceRevision("survey", []byte(strings.Join(survey.SurveyDocs, "\x00"))), survey); err != nil {
			return result, err
		}
	}
	if state.AcceptedCharter != nil {
		if err := addJSON(colony.PlanningEvidenceCharter, "state:accepted-charter", state.AcceptedCharter.EpisodeID, state.AcceptedCharter); err != nil {
			return result, err
		}
	}
	activeDecisions, _ := filterPendingDecisionFileForScope(loadPendingDecisionFile(), pendingDecisionScopeFromState(state))
	resolvedDecisions := make([]PendingDecision, 0, len(activeDecisions.Decisions))
	for _, decision := range activeDecisions.Decisions {
		if decision.Resolved {
			resolvedDecisions = append(resolvedDecisions, decision)
		}
	}
	if len(resolvedDecisions) > 0 {
		if err := addJSON(colony.PlanningEvidenceDecision, "state:resolved-decisions", "decision-frontier-"+runID, resolvedDecisions); err != nil {
			return result, err
		}
	}
	if strings.TrimSpace(contextCapsule) != "" {
		content := []byte(contextCapsule)
		sources = append(sources, planningEvidenceSource{
			Kind: colony.PlanningEvidenceContext, Origin: "context:colony-prime", Content: content, Scope: scope,
			SourceRevision: planningEvidenceSourceRevision("context", content), ObservedAt: observedAt,
			ApplicableDimensions: planningEvidenceDimensions(colony.PlanningEvidenceContext), State: planningEvidenceSourceCurrent,
		})
		present[colony.PlanningEvidenceContext] = true
	}
	researchDocs, err := validateColonyResearchDocs(root, state.ResearchDocs)
	if err != nil {
		return result, err
	}
	approvedRoots := make([]string, 0, len(researchDocs))
	for _, document := range researchDocs {
		content, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(document)))
		if readErr != nil {
			return result, fmt.Errorf("read planning research evidence %s: %w", document, readErr)
		}
		if scan := privacyScan(string(content)); scan.Blocked {
			return result, fmt.Errorf("planning research evidence %s refused: source contains secret-bearing material", document)
		}
		approvedRoot := filepath.ToSlash(filepath.Dir(filepath.FromSlash(document)))
		if approvedRoot == "" {
			approvedRoot = "."
		}
		approvedRoots = append(approvedRoots, approvedRoot)
		sources = append(sources, planningEvidenceSource{
			Kind: colony.PlanningEvidenceResearch, Origin: document, RepositoryPath: document, Scope: scope,
			SourceRevision: planningEvidenceSourceRevision("research", content), ObservedAt: observedAt,
			ApplicableDimensions: planningEvidenceDimensions(colony.PlanningEvidenceResearch), State: planningEvidenceSourceCurrent,
		})
		present[colony.PlanningEvidenceResearch] = true
	}
	hiveEntries := readHiveWisdomEntries(resolveHubPath(), 5, nil)
	if len(hiveEntries) > 0 {
		if err := addJSON(colony.PlanningEvidenceHive, "hive:active-wisdom", planningEvidenceSourceRevision("hive", []byte(runID)), hiveEntries); err != nil {
			return result, err
		}
	}
	if outcomes := planningOutcomeEvidence(state); len(outcomes) > 0 {
		if err := addJSON(colony.PlanningEvidenceOutcome, "state:verified-outcomes", planningEvidenceSourceRevision("outcomes", []byte(strings.Join(outcomes, "\x00"))), outcomes); err != nil {
			return result, err
		}
	}

	records, err := collectPlanningEvidence(planningEvidenceCollectionRequest{
		RepositoryRoot: root,
		ApprovedRoots:  uniqueSortedStrings(approvedRoots),
		Sources:        sources,
	})
	if err != nil {
		return result, err
	}
	phaseResearchDocs, err := discoverScoutPhaseResearchDocs(root)
	if err != nil {
		return result, err
	}
	phaseResearch, err := collectScoutPhaseResearchEvidence(root, phaseResearchDocs, scope, observedAt)
	if err != nil {
		return result, err
	}
	if len(phaseResearch.Records) > 0 {
		present[colony.PlanningEvidenceResearch] = true
		records = mergePlanningEvidenceRecords(records, phaseResearch.Records)
		result.PhaseResearchEvidence = append([]phaseResearchEvidenceAttribution(nil), phaseResearch.Attributions...)
	}
	result.Records = records
	result.Bindings = make([]planningStageEvidenceBinding, 0, len(records))
	for _, record := range records {
		result.Bindings = append(result.Bindings, planningStageEvidenceBinding{ID: record.Reference.ID, ContentHash: record.Reference.ContentHash})
	}
	for _, kind := range []colony.PlanningEvidenceKind{
		colony.PlanningEvidenceSpecification,
		colony.PlanningEvidenceSurvey,
		colony.PlanningEvidenceCharter,
		colony.PlanningEvidenceDecision,
		colony.PlanningEvidenceResearch,
		colony.PlanningEvidenceHive,
		colony.PlanningEvidenceOutcome,
	} {
		if !present[kind] {
			result.MissingKinds = append(result.MissingKinds, kind)
		}
	}
	result.FrontierHash, err = jsonSHA256(struct {
		RunID        string                         `json:"run_id"`
		Scope        planningEvidenceScope          `json:"scope"`
		Evidence     []planningStageEvidenceBinding `json:"evidence"`
		MissingKinds []colony.PlanningEvidenceKind  `json:"missing_kinds,omitempty"`
	}{RunID: runID, Scope: scope, Evidence: result.Bindings, MissingKinds: result.MissingKinds})
	if err != nil {
		return planningEvidencePrimingResult{}, fmt.Errorf("hash planning evidence frontier: %w", err)
	}
	return result, nil
}

func planningStartWeakestGap(seed codexPlanIterationState, evidence planningEvidencePrimingResult) (colony.PlanningGap, error) {
	description := "Scout must verify the weakest implementation, dependency, and effort assumptions before Route-Setter may propose a plan."
	evidenceNeeded := "Fresh repository or authoritative evidence that resolves the weakest current planning assumption."
	severity := 25
	if seed.LastIteration > 0 && len(seed.SelectedGaps) > 0 {
		description = strings.TrimSpace(seed.SelectedGaps[0])
		evidenceNeeded = "Fresh applicable evidence that resolves this prior pass's weakest remaining gap."
		severity = 60
	} else if len(evidence.MissingKinds) > 0 {
		missing := make([]string, 0, len(evidence.MissingKinds))
		for _, kind := range evidence.MissingKinds {
			missing = append(missing, string(kind))
		}
		description = "Missing or stale " + strings.Join(missing, ", ") + " evidence limits initial planning readiness."
		evidenceNeeded = "Current attributable evidence for " + strings.Join(missing, ", ") + "."
		severity = 75
	}
	evidenceIDs := make([]string, 0, len(evidence.Bindings))
	for _, binding := range evidence.Bindings {
		evidenceIDs = append(evidenceIDs, binding.ID)
	}
	gap := colony.PlanningGap{
		SchemaVersion:           colony.PlanningSchemaVersion,
		Dimension:               colony.PlanningDimensionKnowledge,
		Materiality:             colony.PlanningGapNonMaterial,
		Severity:                severity,
		Description:             description,
		EvidenceIDs:             evidenceIDs,
		EvidenceThatWouldChange: evidenceNeeded,
	}
	payload := gap
	payload.ID = ""
	payload.ContentHash = ""
	hash, err := jsonSHA256(payload)
	if err != nil {
		return colony.PlanningGap{}, fmt.Errorf("hash initial planning gap: %w", err)
	}
	gap.ContentHash = hash
	gap.ID = "planning-gap-" + hash[:16]
	if err := gap.Validate(); err != nil {
		return colony.PlanningGap{}, fmt.Errorf("validate initial planning gap: %w", err)
	}
	return gap, nil
}

func buildPlanningRunHeader(goal string, policy planningPresetPolicy, approved approvedPlanningSpecification, baseRevisionID, baseRevisionHash string, evidence planningEvidencePrimingResult, gap colony.PlanningGap, researchPolicy phaseResearchAutomaticPolicy, manifest planningStageManifest, createdAt time.Time) (planningRunHeader, error) {
	header := planningRunHeader{
		SchemaVersion:         planningRunHeaderSchemaVersion,
		RunID:                 manifest.RunID,
		Goal:                  strings.TrimSpace(goal),
		GoalID:                approved.GoalID,
		SessionID:             approved.SessionID,
		Specification:         approved.Binding,
		BasePlanRevisionID:    baseRevisionID,
		BasePlanRevisionHash:  baseRevisionHash,
		Preset:                policy.ID,
		TargetConfidence:      policy.TargetConfidence,
		PassCap:               policy.PassCap,
		EvidenceCatalogue:     append([]planningEvidenceRecord(nil), evidence.Records...),
		EvidenceFrontier:      append([]planningStageEvidenceBinding(nil), evidence.Bindings...),
		InputFrontierHash:     evidence.FrontierHash,
		MissingEvidenceKinds:  append([]colony.PlanningEvidenceKind(nil), evidence.MissingKinds...),
		ResearchPolicy:        researchPolicy,
		PhaseResearchEvidence: append([]phaseResearchEvidenceAttribution(nil), evidence.PhaseResearchEvidence...),
		WeakestGap:            gap,
		StageManifestID:       manifest.ID,
		StageManifestHash:     manifest.ContentHash,
		CreatedAt:             createdAt.UTC(),
	}
	payload := header
	payload.ID = ""
	payload.ContentHash = ""
	hash, err := jsonSHA256(payload)
	if err != nil {
		return planningRunHeader{}, fmt.Errorf("hash planning run header: %w", err)
	}
	header.ContentHash = hash
	header.ID = "planning-run-header-" + hash[:16]
	return header, nil
}

func preparePlanningScoutStage(root string, state colony.ColonyState, goal string, policy planningPresetPolicy, approved approvedPlanningSpecification, baseRevisionID, baseRevisionHash, contextCapsule string, survey codexSurveyContext, seed codexPlanIterationState, refresh bool, generatedAt time.Time) (planningRunHeader, planningStageState, planningStageManifest, error) {
	emptyHeader := planningRunHeader{}
	emptyState := planningStageState{}
	emptyManifest := planningStageManifest{}
	runID := strings.TrimSpace(seed.PlanningRunID)
	if runID == "" {
		runID = planningRunID(goal, root, generatedAt)
	}
	pass := seed.LastIteration + 1
	if pass < 1 {
		pass = 1
	}
	evidence, err := primePlanningStartEvidence(root, state, approved, survey, contextCapsule, baseRevisionID, runID, generatedAt)
	if err != nil {
		return emptyHeader, emptyState, emptyManifest, err
	}
	if len(evidence.Bindings) == 0 {
		return emptyHeader, emptyState, emptyManifest, fmt.Errorf("approved planning start produced no evidence frontier")
	}
	gap, err := planningStartWeakestGap(seed, evidence)
	if err != nil {
		return emptyHeader, emptyState, emptyManifest, err
	}
	researchPolicy := computeAutomaticPhaseResearchPolicy(
		policy.ID,
		&gap,
		survey,
		phaseResearchCandidates(state, seed),
		state.Plan.Phases,
		evidence.Records,
		refresh,
	)
	priorCardHash := planningEvidenceSHA256([]byte("planning-card-origin"))
	initial := planningStageState{
		Stage:                planningStagePresetRequired,
		RunID:                runID,
		Pass:                 pass,
		Specification:        approved.Binding,
		BasePlanRevisionID:   baseRevisionID,
		BasePlanRevisionHash: baseRevisionHash,
		PriorCardHash:        priorCardHash,
		InputFrontierHash:    evidence.FrontierHash,
		WeakestGap:           &gap,
	}
	ready, manifest, err := reducePlanningStage(initial, planningStageTransition{To: planningStageScoutReady, Preset: policy.ID})
	if err != nil {
		return emptyHeader, emptyState, emptyManifest, fmt.Errorf("select planning preset: %w", err)
	}
	if manifest != nil {
		return emptyHeader, emptyState, emptyManifest, fmt.Errorf("preset selection unexpectedly authorized a worker")
	}
	authorizationSeed, err := jsonSHA256(struct {
		RunID        string                         `json:"run_id"`
		Pass         int                            `json:"pass"`
		Caste        planningStageWorkerCaste       `json:"caste"`
		FrontierHash string                         `json:"frontier_hash"`
		Evidence     []planningStageEvidenceBinding `json:"evidence"`
		GapHash      string                         `json:"gap_hash"`
	}{RunID: runID, Pass: pass, Caste: planningStageCasteScout, FrontierHash: evidence.FrontierHash, Evidence: evidence.Bindings, GapHash: gap.ContentHash})
	if err != nil {
		return emptyHeader, emptyState, emptyManifest, fmt.Errorf("hash Scout authorization: %w", err)
	}
	authorization := planningStageAuthorization{
		ID:                "planning-authorization-" + authorizationSeed[:16],
		ExpectedCaste:     planningStageCasteScout,
		InputFrontierHash: evidence.FrontierHash,
		EvidenceFrontier:  evidence.Bindings,
		WeakestGap:        &gap,
	}
	running, stageManifest, err := reducePlanningStage(ready, planningStageTransition{To: planningStageScoutRunning, Authorization: &authorization})
	if err != nil {
		return emptyHeader, emptyState, emptyManifest, fmt.Errorf("authorize Scout planning stage: %w", err)
	}
	if stageManifest == nil {
		return emptyHeader, emptyState, emptyManifest, fmt.Errorf("Scout planning stage did not emit a manifest")
	}
	header, err := buildPlanningRunHeader(goal, policy, approved, baseRevisionID, baseRevisionHash, evidence, gap, researchPolicy, *stageManifest, generatedAt)
	if err != nil {
		return emptyHeader, emptyState, emptyManifest, err
	}
	return header, running, *stageManifest, nil
}

func persistPlanningScoutStage(root string, header planningRunHeader, running planningStageState, manifest planningStageManifest) error {
	return withPlanningMutationSession(root, "planning-scout-stage", func(session *planningMutationSession) error {
		return persistPlanningScoutStageInSession(session, header, running, manifest)
	})
}

func persistPlanningScoutStageInSession(session *planningMutationSession, header planningRunHeader, running planningStageState, manifest planningStageManifest) error {
	return persistPlanningScoutStageAndStateInSession(session, header, running, manifest, nil)
}

func persistPlanningScoutStageAndStateInSession(session *planningMutationSession, header planningRunHeader, running planningStageState, manifest planningStageManifest, state *colony.ColonyState) error {
	if err := validatePlanningStageManifest(manifest); err != nil {
		return err
	}
	if err := validatePlanningStageRunningState(running, manifest); err != nil {
		return err
	}
	headerBytes, err := marshalPlanningStageJSON(header)
	if err != nil {
		return fmt.Errorf("marshal planning run header: %w", err)
	}
	manifestBytes, err := marshalPlanningStageJSON(manifest)
	if err != nil {
		return fmt.Errorf("marshal planning stage manifest: %w", err)
	}
	stateBytes, err := marshalPlanningStageJSON(running)
	if err != nil {
		return fmt.Errorf("marshal planning stage state: %w", err)
	}
	headerPath := filepath.ToSlash(filepath.Join("planning", manifest.RunID, "run-header.json"))
	targets := []planningSessionTarget{
		{Root: lifecycleTransactionRootData, Path: headerPath, Content: headerBytes},
		{Root: lifecycleTransactionRootData, Path: planningStageDataRelativePath(planningStageManifestRepositoryPath(manifest.RunID, manifest.ID)), Content: manifestBytes},
		{Root: lifecycleTransactionRootData, Path: planningStageDataRelativePath(planningStageStateRepositoryPath(manifest.RunID)), Content: stateBytes},
	}
	if state != nil {
		colonyStateBytes, err := marshalSpecificationState(*state)
		if err != nil {
			return err
		}
		targets = append(targets, planningSessionTarget{Root: lifecycleTransactionRootData, Path: "COLONY_STATE.json", Content: colonyStateBytes})
	}
	if err := commitPlanningSessionTargets(session, "planning-scout-stage-"+manifest.ContentHash[:24], "planning-scout-stage", targets); err != nil {
		return fmt.Errorf("persist Scout run header and stage dispatch: %w", err)
	}
	return nil
}

type planningSessionTarget struct {
	Root    lifecycleTransactionRootKind
	Path    string
	Content []byte
	Remove  bool
}

// commitPlanningSessionTargets is the small common boundary used by the
// remaining planning writers in this file. It captures every target baseline
// while the repository session is held, derives a baseline-and-output-bound
// transaction identity, and publishes the complete target set together.
func commitPlanningSessionTargets(session *planningMutationSession, transactionPrefix, command string, targets []planningSessionTarget) error {
	if session == nil {
		return fmt.Errorf("%s requires a planning mutation session", command)
	}
	if len(targets) == 0 {
		return nil
	}
	sortedTargets := append([]planningSessionTarget(nil), targets...)
	sort.Slice(sortedTargets, func(left, right int) bool {
		if sortedTargets[left].Root != sortedTargets[right].Root {
			return sortedTargets[left].Root < sortedTargets[right].Root
		}
		return sortedTargets[left].Path < sortedTargets[right].Path
	})
	digest := sha256.New()
	allApplied := true
	for _, target := range sortedTargets {
		current, exists, err := session.ReadFile(target.Root, target.Path)
		if err != nil {
			return err
		}
		fmt.Fprintf(digest, "%s\x00%s\x00%t\x00", target.Root, target.Path, exists)
		digest.Write(current)
		digest.Write([]byte{0})
		fmt.Fprintf(digest, "%t\x00", target.Remove)
		digest.Write(target.Content)
		digest.Write([]byte{0})
		if target.Remove {
			allApplied = allApplied && !exists
		} else {
			allApplied = allApplied && exists && bytes.Equal(current, target.Content)
		}
	}
	if allApplied {
		return nil
	}
	prefix := strings.Trim(strings.TrimSpace(transactionPrefix), "-")
	if prefix == "" {
		prefix = "planning-session"
	}
	transactionID := prefix + "-" + hex.EncodeToString(digest.Sum(nil))[:20]
	tx, err := beginLifecycleTransaction(lifecycleTransactionConfig{
		TransactionID: transactionID,
		Command:       command,
		Allowlist: lifecycleTransactionAllowlist{
			RepositoryRoot: session.RepositoryRoot(), LifecycleDataRoot: session.DataRoot(),
		},
		Session: session,
	})
	if err != nil {
		return err
	}
	for _, target := range sortedTargets {
		if target.Remove {
			err = tx.DeclareRemoval(target.Root, target.Path)
		} else {
			err = tx.DeclareWrite(target.Root, target.Path, target.Content)
		}
		if err != nil {
			return err
		}
	}
	if err := tx.Validate(); err != nil {
		return err
	}
	if _, err = tx.Commit(); err != nil {
		return err
	}
	for _, target := range sortedTargets {
		if err := refreshPlanningSessionBaseline(session, target.Root, target.Path); err != nil {
			return err
		}
	}
	return nil
}

func refreshPlanningSessionBaseline(session *planningMutationSession, root lifecycleTransactionRootKind, relativePath string) error {
	state, err := session.currentFileState(root, relativePath)
	if err != nil {
		return err
	}
	return session.updateBaseline(root, relativePath, state)
}

func persistPlanningColonyStateInSession(session *planningMutationSession, operation string, state colony.ColonyState) error {
	content, err := marshalSpecificationState(state)
	if err != nil {
		return err
	}
	return commitPlanningSessionTargets(session, operation, operation, []planningSessionTarget{{
		Root: lifecycleTransactionRootData, Path: "COLONY_STATE.json", Content: content,
	}})
}

func planningSessionSummaryTargets(session *planningMutationSession, state colony.ColonyState, commandName, suggestedNext, summary string, now time.Time) ([]planningSessionTarget, error) {
	var sessionFile colony.SessionFile
	exists, err := session.LoadJSON(lifecycleTransactionRootData, "session.json", &sessionFile)
	if err != nil {
		return nil, fmt.Errorf("load session summary: %w", err)
	}
	now = now.UTC()
	nowText := now.Format(time.RFC3339)
	goal := ""
	if state.Goal != nil {
		goal = strings.TrimSpace(*state.Goal)
	}
	if !exists {
		sessionID := fmt.Sprintf("%d_%s", now.Unix(), randomHex(4))
		if state.SessionID != nil && strings.TrimSpace(*state.SessionID) != "" {
			sessionID = strings.TrimSpace(*state.SessionID)
		}
		startedAt := nowText
		if state.InitializedAt != nil {
			startedAt = state.InitializedAt.UTC().Format(time.RFC3339)
		}
		sessionFile = colony.SessionFile{
			SessionID: sessionID, StartedAt: startedAt, ColonyGoal: goal,
			ColonyMode: state.EffectiveColonyMode(), CurrentPhase: state.CurrentPhase,
			CurrentMilestone: state.Milestone, SuggestedNext: "aether status",
			ActiveTodos: []string{}, Summary: "Session initialized",
		}
	}
	if strings.TrimSpace(sessionFile.SessionID) == "" {
		sessionFile.SessionID = fmt.Sprintf("%d_%s", now.Unix(), randomHex(4))
	}
	if strings.TrimSpace(sessionFile.StartedAt) == "" {
		sessionFile.StartedAt = nowText
	}
	sessionFile.ColonyGoal = goal
	sessionFile.ColonyMode = state.EffectiveColonyMode()
	sessionFile.CurrentPhase = state.CurrentPhase
	sessionFile.CurrentMilestone = state.Milestone
	sessionFile.ActiveTodos = mergeShelfTodos(sessionFile.ActiveTodos, sessionActiveTodosFromState(state))
	if strings.TrimSpace(commandName) != "" {
		sessionFile.LastCommand = commandName
		sessionFile.LastCommandAt = nowText
	}
	if strings.TrimSpace(suggestedNext) != "" {
		sessionFile.SuggestedNext = suggestedNext
	} else if strings.TrimSpace(sessionFile.SuggestedNext) == "" {
		sessionFile.SuggestedNext = nextCommandFromState(state)
	}
	if strings.TrimSpace(summary) != "" {
		sessionFile.Summary = summary
	}
	if head := strings.TrimSpace(getGitHEAD()); head != "" {
		sessionFile.BaselineCommit = head
	}
	sessionBytes, err := json.MarshalIndent(sessionFile, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal session summary: %w", err)
	}
	sessionBytes = append(sessionBytes, '\n')
	next := sessionFile.SuggestedNext
	if strings.TrimSpace(next) == "" {
		next = nextCommandFromState(state)
	}
	contextBytes := []byte(renderContextSnapshot(state, sessionFile, next, summary, defaultSafeToClear(state)))
	handoffBytes := []byte(renderHandoffSnapshot(state, sessionFile, "Session Snapshot", next, summary))
	return []planningSessionTarget{
		{Root: lifecycleTransactionRootData, Path: "session.json", Content: sessionBytes},
		{Root: lifecycleTransactionRootRepository, Path: filepath.ToSlash(filepath.Join(".aether", "CONTEXT.md")), Content: contextBytes},
		{Root: lifecycleTransactionRootRepository, Path: filepath.ToSlash(filepath.Join(".aether", "HANDOFF.md")), Content: handoffBytes},
	}, nil
}

func runCodexPlan(root string, refresh bool, synthetic bool) (map[string]interface{}, error) {
	return runCodexPlanWithOptions(root, codexPlanOptions{
		Refresh:          refresh,
		Synthetic:        synthetic,
		RequireTerritory: true,
	})
}

func runCodexPlanWithOptions(root string, opts codexPlanOptions) (map[string]interface{}, error) {
	var result map[string]interface{}
	sessionRoot := root
	if store != nil {
		if authoritativeRoot := aetherRootFromDataPath(store.BasePath()); authoritativeRoot != "" && filepath.Clean(filepath.Join(root, ".aether", "data")) != filepath.Clean(store.BasePath()) {
			sessionRoot = authoritativeRoot
		}
	}
	// A standard-mode plan-only request that merely reports an existing plan is
	// provably read-only. Avoid creating the session's durable lock sentinel so
	// the long-standing --plan-only no-mutation contract remains byte exact.
	if readOnly, handled, err := runCodexExistingPlanReadOnly(sessionRoot, opts); handled {
		return readOnly, err
	}
	err := withPlanningMutationSession(sessionRoot, "codex-plan", func(session *planningMutationSession) error {
		var runErr error
		result, runErr = runCodexPlanWithOptionsInSession(session, opts)
		return runErr
	})
	return result, err
}

func runCodexExistingPlanReadOnly(_ string, opts codexPlanOptions) (map[string]interface{}, bool, error) {
	if !opts.PlanOnly || opts.Refresh || opts.RepairArtifact || opts.Accept || len(opts.ResearchDocs) != 0 || store == nil {
		return nil, false, nil
	}
	state, err := loadActiveColonyState()
	if err != nil || len(state.Plan.Phases) == 0 || state.EffectiveColonyMode() == colony.ColonyModeOrchestrator {
		return nil, false, nil
	}
	if state.Goal == nil || strings.TrimSpace(*state.Goal) == "" {
		return nil, true, fmt.Errorf("No active colony goal. Run `aether init \"goal\"` first.")
	}
	preset, err := resolvePlanningPreset(opts)
	if err != nil {
		return nil, true, err
	}
	if preset.PresetRequired {
		return planningPresetRequiredResult(state, preset), true, nil
	}
	opts.Preset = string(preset.Policy.ID)
	opts.Depth = string(preset.Policy.ID)
	opts.TargetConfidence = preset.Policy.TargetConfidence
	opts.MaxIterations = preset.Policy.PassCap
	granularity, planDepth, err := resolvePlanGranularityDepth(state.PlanGranularity, opts.Depth)
	if err != nil {
		return nil, true, err
	}
	currentPhase := firstBuildablePhase(state.Plan.Phases)
	planningPhase := state.Plan.Phases[0]
	if currentPhase > 0 && currentPhase <= len(state.Plan.Phases) {
		planningPhase = state.Plan.Phases[currentPhase-1]
	}
	planningDepth, err := resolvePlanningDepthSmart(opts.PlanningDepth, planningPhase, len(state.Plan.Phases))
	if err != nil {
		return nil, true, err
	}
	verificationDepth, err := resolveVerificationDepthSmart(opts.VerificationDepth, planningPhase, len(state.Plan.Phases))
	if err != nil {
		return nil, true, err
	}
	pending := loadPendingDecisionFile()
	unresolvedClarifications := countPendingClarifications(pending)
	clarificationWarning := ""
	if unresolvedClarifications > 0 {
		clarificationWarning = "Unresolved clarifications exist. Run `aether discuss` to resolve them before planning, or proceed with implicit assumptions."
	}
	nextPhase := firstBuildablePhase(state.Plan.Phases)
	nextCommand := "aether build 1"
	if nextPhase > 0 {
		nextCommand = fmt.Sprintf("aether build %d", nextPhase)
	}
	result := map[string]interface{}{
		"plan_only": true, "planned": true, "existing_plan": true,
		"colony_mode": string(state.EffectiveColonyMode()), "goal": *state.Goal,
		"phases": state.Plan.Phases, "plan_revision": planRevisionSummary(state.Plan), "count": len(state.Plan.Phases),
		"depth": planDepth, "planning_depth": planningDepth, "verification_depth": verificationDepth,
		"planning_loop": resolvePlanningLoopOptions(planDepth, opts), "preset_required": false,
		"preset_options": preset.Options, "selected_preset": string(preset.Policy.ID), "selection_source": preset.SelectionSource,
		"target_confidence": preset.Policy.TargetConfidence, "max_iterations": preset.Policy.PassCap,
		"verification_smart_default": opts.VerificationDepth == "", "planning_smart_default": opts.PlanningDepth == "",
		"planning_phase": planningPhase, "granularity": string(granularity),
		"dispatch_contract": planningDispatchContractWithTimeout(opts.WorkerTimeout), "dispatch_mode": "plan-only",
		"requires_finalizer": false, "unresolved_clarifications": unresolvedClarifications,
		"clarification_warning": clarificationWarning, "next": nextCommand,
	}
	addBoundaryQuestionResultFields(result, emptyOrchestratorBoundaryQuestionsResult())
	addOrchestratorBoundaryGuidance(result, "plan", state, nextCommand, nil)
	return result, true, nil
}

func runCodexPlanWithOptionsInSession(session *planningMutationSession, opts codexPlanOptions) (map[string]interface{}, error) {
	root := session.RepositoryRoot()
	if opts.Accept {
		return nil, fmt.Errorf("--accept no longer accepts or activates a plan. Review the pending candidate, then use `aether plan --accept-candidate <candidate-id>`")
	}
	preset, err := resolvePlanningPreset(opts)
	if err != nil {
		return nil, err
	}
	if store == nil {
		return nil, fmt.Errorf("no store initialized")
	}

	state, err := loadSpecificationColonyStateInSession(session)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "specification") {
			return nil, planningSpecificationRecoveryError("the current specification state is invalid: "+err.Error(), "aether spec")
		}
		return nil, fmt.Errorf("%s", colonyStateLoadMessage(err))
	}
	if state.Goal == nil || strings.TrimSpace(*state.Goal) == "" {
		return nil, fmt.Errorf("No active colony goal. Run `aether init \"goal\"` first.")
	}
	if preset.PresetRequired {
		return planningPresetRequiredResult(state, preset), nil
	}
	opts.Preset = string(preset.Policy.ID)
	opts.Depth = string(preset.Policy.ID)
	opts.TargetConfidence = preset.Policy.TargetConfidence
	opts.MaxIterations = preset.Policy.PassCap
	if len(state.Plan.Phases) == 0 || opts.Refresh {
		if _, err := requireApprovedPlanningSpecification(root, state); err != nil {
			return nil, err
		}
	}
	// Legacy non-git workspaces have no immutable source revision to bind. They
	// retain the pre-199 planning behavior; repositories with a real revision
	// use the strict automatic territory gate below.
	_, territoryRevisionErr := currentTerritoryRevision(root)
	if opts.RequireTerritory && territoryRevisionErr == nil && !opts.RepairArtifact && (len(state.Plan.Phases) == 0 || opts.Refresh) {
		freshness, refreshManifest, err := territoryPlanPreflight(root, opts)
		if err != nil {
			return nil, err
		}
		if refreshManifest != nil {
			return territoryRefreshPlanResult(state, freshness, *refreshManifest), nil
		}
		opts.Territory = &freshness
	}

	// --research is additive and persistent: point once, and every later plan
	// run keeps the document. A bad path fails the run rather than being
	// dropped -- a silently ignored research document is the failure this
	// whole handoff exists to prevent.
	merged, researchChanged, researchErr := mergeColonyResearchDocs(root, state.ResearchDocs, opts.ResearchDocs)
	if researchErr != nil {
		return nil, researchErr
	}
	if researchChanged {
		state.ResearchDocs = merged
	}

	revisionContext, err := buildPlanRevisionContext(root, state, opts)
	if err != nil {
		return nil, err
	}
	basePlanStateHash, err := planStateHash(state.Plan)
	if err != nil {
		return nil, fmt.Errorf("hash active plan before planning: %w", err)
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
		return runCodexPlanPlanOnlyInSession(session, state, granularity, planDepth, unresolvedClarifications, clarificationWarning, researchChanged, opts)
	}
	if opts.RepairArtifact {
		return runCodexPlanRepairArtifact(root)
	}

	if len(state.Plan.Phases) > 0 && !opts.Refresh {
		// Persist resolved verification depth only for non-plan-only paths.
		state.VerificationDepth = verificationDepth
		nextPhase := firstBuildablePhase(state.Plan.Phases)
		nextCommand := "aether build 1"
		if nextPhase > 0 {
			nextCommand = fmt.Sprintf("aether build %d", nextPhase)
		}
		summary := fmt.Sprintf("Loaded existing plan (%d phases)", len(state.Plan.Phases))
		stateBytes, marshalErr := marshalSpecificationState(state)
		if marshalErr != nil {
			return nil, marshalErr
		}
		targets := []planningSessionTarget{{Root: lifecycleTransactionRootData, Path: "COLONY_STATE.json", Content: stateBytes}}
		sessionTargets, summaryErr := planningSessionSummaryTargets(session, state, "plan", nextCommand, summary, time.Now().UTC())
		if summaryErr != nil {
			return nil, summaryErr
		}
		targets = append(targets, sessionTargets...)
		if err := commitPlanningSessionTargets(session, "plan-existing", "plan-existing", targets); err != nil {
			return nil, fmt.Errorf("persist existing plan metadata: %w", err)
		}
		return map[string]interface{}{
			"planned":                    true,
			"existing_plan":              true,
			"colony_mode":                string(state.EffectiveColonyMode()),
			"goal":                       *state.Goal,
			"phases":                     state.Plan.Phases,
			"plan_revision":              planRevisionSummary(state.Plan),
			"count":                      len(state.Plan.Phases),
			"depth":                      planDepth,
			"planning_depth":             planningDepth,
			"verification_depth":         verificationDepth,
			"planning_loop":              resolvePlanningLoopOptions(planDepth, opts),
			"preset_required":            false,
			"preset_options":             preset.Options,
			"selected_preset":            string(preset.Policy.ID),
			"selection_source":           preset.SelectionSource,
			"target_confidence":          preset.Policy.TargetConfidence,
			"max_iterations":             preset.Policy.PassCap,
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
		return runCodexPlanAgentDelegateInSession(session, state, granularity, planDepth, unresolvedClarifications, clarificationWarning, researchChanged, opts)
	}

	var invoker codex.WorkerInvoker
	if !opts.Synthetic {
		invoker = newCodexWorkerInvoker()
		if _, ok := invoker.(*codex.FakeInvoker); !ok && (invoker == nil || !invoker.IsAvailable(context.Background())) {
			return nil, planningWorkersUnavailableError(invoker)
		}
	}

	// Resolve verification depth before dispatch, but publish it only with the
	// selected final or intermediate branch transaction.
	state.VerificationDepth = verificationDepth

	var refreshCleanupTargets []planningSessionTarget
	if opts.Refresh {
		refreshCleanupTargets, err = fallbackPlanningArtifactRemovalTargetsInSession(session)
		if err != nil {
			return nil, err
		}
	}

	runHandle, err := beginRuntimeSpawnRun("plan", time.Now().UTC())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize planning run: %w", err)
	}
	runStatus := "failed"
	defer func() {
		finishRuntimeSpawnRun(runHandle, runStatus, time.Now().UTC())
	}()

	// 202-03 (CEC-05): the planning episode. planEpisodeID reuses the same
	// run identifier finishRuntimeSpawnRun above persists, so the cockpit
	// and the durable spawn-run record name the same episode.
	planEpisodeID := ""
	if runHandle != nil {
		planEpisodeID = runHandle.Run.ID
	}
	emitColonyLiveEpisodeStarted(planEpisodeID, events.EpisodeKindPlan)
	defer func() {
		emitColonyLiveEpisodeEnded(planEpisodeID, events.EpisodeKindPlan, runStatus)
	}()

	survey, err := loadCodexSurveyContext(root)
	if err != nil {
		return nil, err
	}
	generatedAt := time.Now().UTC()
	iterationLoop := resolvePlanningLoopOptions(planDepth, opts)
	iterationSeed := planningManifestIterationSeed(*state.Goal, root, planDepth, planningDepth, iterationLoop, revisionContext, generatedAt)
	iteration := iterationSeed.LastIteration + 1
	if iteration < 1 {
		iteration = 1
	}
	iterationLoop.History = append([]codexPlanningLoopSample{}, iterationSeed.History...)
	iterationLoop.Iterations = iterationSeed.LastIteration
	iterationLoop.FinalConfidence = iterationSeed.PreviousConfidence
	iterationLoop.Gaps = append([]string{}, iterationSeed.SelectedGaps...)
	iterationAppendix := planningIterationAppendix(iterationSeed, iteration) + renderPlanRevisionWorkerAppendix(root, revisionContext)

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
	planningWarning := planningSyntheticWarningForMode(opts.Synthetic)
	if err := ensurePlanningSpawnTreeInSession(session); err != nil {
		return nil, err
	}
	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	for _, dispatch := range dispatches {
		if err := spawnTree.RecordSpawn("Queen", dispatch.Caste, dispatch.Name, dispatch.Task, 1); err != nil {
			return nil, fmt.Errorf("failed to record planning spawn: %w", err)
		}
	}

	emitVisualProgress(renderPlanDispatchPreview(*state.Goal, dispatches))

	if !opts.Synthetic {
		realDispatches, dispatchErr := dispatchRealPlanningWorkersWithIterationContext(context.Background(), root, survey, invoker, opts.WorkerTimeout, iterationAppendix, *state.Goal)
		if realDispatches != nil {
			dispatches = realDispatches
		}
		if dispatchErr != nil {
			if _, ok := invoker.(*codex.FakeInvoker); ok {
				dispatchMode = "simulated"
			} else if isWorkerProviderPreflightError(dispatchErr) {
				return nil, dispatchErr
			} else {
				return nil, planningWorkersFailedError(dispatchErr)
			}
		} else if realDispatches == nil {
			if _, ok := invoker.(*codex.FakeInvoker); ok {
				dispatchMode = "simulated"
			} else {
				return nil, planningWorkersUnavailableError(invoker)
			}
		} else if realDispatches != nil {
			if _, ok := invoker.(*codex.FakeInvoker); ok {
				dispatchMode = "simulated"
			} else {
				dispatchMode = "real"
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
	phases, confidence, unresolvedGaps := synthesizeRouteSetterPlan(*state.Goal, granularity, survey, scoutReport)
	if workerPlan, ok, note, err := loadWorkerPlanArtifact(root, artifactSnapshots, dispatches); err != nil {
		return nil, err
	} else if ok {
		phases = buildWorkerPlanPhases(workerPlan)
		confidence = mergePlanConfidence(confidence, workerPlan.Confidence)
		unresolvedGaps = limitStrings(uniqueSortedStrings(append(unresolvedGaps, workerPlan.Gaps...)), 4)
		if len(note) > 0 {
			unresolvedGaps = limitStrings(uniqueSortedStrings(append(unresolvedGaps, note)), 4)
		}
		planSource = "worker-artifact"
	} else if note != "" {
		unresolvedGaps = limitStrings(uniqueSortedStrings(append(unresolvedGaps, note)), 4)
	}
	if (opts.Synthetic && planSource == "local-synthesis") || dispatchMode == "simulated" {
		phases = bindSyntheticPlanEvidence(phases)
	}
	phasePlanDraft := workerPlanArtifactFromPhases(confidence, unresolvedGaps, phases, codexPlanningLoop{})
	evidenceHash := planningCompletionEvidenceHash(scoutReport, phasePlanDraft)
	dispatchContract := planningDispatchContractForDispatches(dispatches, opts.WorkerTimeout)
	manifest := codexPlanManifest{
		Goal:                 *state.Goal,
		Root:                 root,
		GeneratedAt:          generatedAt.Format(time.RFC3339),
		BaseRevisionID:       activePlanRevisionID(state.Plan),
		BasePlanStateHash:    basePlanStateHash,
		ColonyMode:           string(state.EffectiveColonyMode()),
		Refresh:              opts.Refresh,
		Revision:             revisionContext,
		ExistingPlan:         len(state.Plan.Phases) > 0,
		ExistingPhaseCount:   len(state.Plan.Phases),
		Synthetic:            opts.Synthetic,
		SyntheticWarning:     planningSyntheticWarningForMode(opts.Synthetic),
		PlanningRunID:        iterationSeed.PlanningRunID,
		Iteration:            iteration,
		TargetConfidence:     iterationLoop.TargetConfidence,
		MaxIterations:        iterationLoop.MaxIterations,
		PreviousConfidence:   iterationSeed.PreviousConfidence,
		PreviousEvidenceHash: iterationSeed.PreviousEvidenceHash,
		SelectedGaps:         append([]string{}, iterationSeed.SelectedGaps...),
		PreviousPlanDraft:    iterationSeed.PreviousPlanDraft,
		ExpectedWorkers:      append([]codexPlanningDispatch{}, dispatches...),
		Depth:                planDepth,
		Granularity:          string(granularity),
		GranularityMin:       granularityMin(granularity),
		GranularityMax:       granularityMax(granularity),
		PlanningDepth:        planningDepth,
		VerificationDepth:    verificationDepth,
		PlanningLoop:         iterationLoop,
		Survey:               survey,
		Dispatches:           append([]codexPlanningDispatch{}, dispatches...),
		Snapshots:            artifactSnapshots,
		DispatchMode:         dispatchMode,
		DispatchContract:     dispatchContract,
		FinalizeSurface:      "direct-runtime",
		RequiresFinalizer:    false,
	}
	if opts.Territory != nil {
		attachTerritoryToPlanManifest(&manifest, *opts.Territory)
	}
	planningLoop := evaluatePlanningLoop(confidence, unresolvedGaps, opts, planDepth)
	if !opts.Synthetic && dispatchMode == "real" {
		if err := validatePlanningConfidenceEvidence(manifest, confidence, evidenceHash); err != nil {
			return nil, err
		}
		planningLoop = evaluatePlanningLoopIteration(manifest, confidence, unresolvedGaps, evidenceHash)
	}
	publication, err := preparePlanningArtifactPublication(session, *state.Goal, granularity, survey, scoutDispatch, scoutReport, routeSetterDispatch, confidence, unresolvedGaps, phases, planningLoop, artifactSnapshots, dispatches, true, true)
	if err != nil {
		return nil, err
	}
	scoutFile := publication.ScoutFile
	routeSetterFile := publication.RouteSetterFile
	planArtifactFile := publication.PlanArtifactFile
	phaseResearchFiles := publication.PhaseResearchFiles
	if publication.PreservedScout || publication.PreservedRouteSetter || publication.PreservedPlan || publication.PreservedPhaseResearch > 0 {
		artifactSource = "worker-written"
	}
	backupTargets, err := planningBackupRemovalTargetsInSession(session)
	if err != nil {
		return nil, err
	}
	publicationTargets := mergePlanningSessionTargets(
		refreshCleanupTargets,
		[]planningSessionTarget{planningFallbackMarkerTarget(dispatchMode, time.Now().UTC())},
		backupTargets,
		publication.Targets,
	)

	for i := range dispatches {
		status := dispatches[i].Status
		if strings.TrimSpace(status) == "" || status == "spawned" {
			status = "completed"
		}
		summary := strings.TrimSpace(dispatches[i].Summary)
		if summary == "" {
			summary = strings.Join(dispatches[i].Outputs, ", ")
		}
		if summary == "" && dispatchMode == "synthetic" {
			summary = "Explicit synthetic planning synthesis"
		}
		if summary == "" && dispatchMode == "simulated" {
			summary = "Simulated planning dispatch"
		}
		if err := spawnTree.UpdateStatus(dispatches[i].Name, status, summary); err != nil {
			return nil, fmt.Errorf("failed to update planning completion: %w", err)
		}
	}
	emitPlanCeremonyDispatchSequence("aether-plan", dispatches)
	// 202-03 (CEC-05): the planning wave and its worker started/finished
	// pairs, immediately adjacent to the ceremony dispatch sequence above --
	// dispatches are already fully resolved (Name/Caste/Status) by this
	// point, so start and finish are emitted back to back rather than
	// bracketing an async run the way build's per-worker loop does.
	emitColonyLiveWaveStarted(planEpisodeID, events.EpisodeKindPlan, 1)
	for _, dispatch := range dispatches {
		planWorker := codex.WorkerDispatch{WorkerName: dispatch.Name, Caste: dispatch.Caste, Wave: 1}
		emitColonyLiveWorkerStarted(planEpisodeID, events.EpisodeKindPlan, planWorker)
		status := strings.TrimSpace(dispatch.Status)
		if status == "" || status == "spawned" {
			status = "completed"
		}
		emitColonyLiveWorkerFinished(planEpisodeID, events.EpisodeKindPlan, planWorker, codex.DispatchResult{WorkerName: dispatch.Name, Status: status})
	}
	emitColonyLiveWaveEnded(planEpisodeID, events.EpisodeKindPlan, 1, "completed")

	statuses := make([]string, 0, len(dispatches))
	for _, dispatch := range dispatches {
		statuses = append(statuses, dispatch.Status)
	}
	runStatus = summarizeRunStatus(statuses...)
	if !opts.Synthetic && dispatchMode == "real" && planningLoop.StopReason == planningLoopPendingStop {
		phasePlanDraft = workerPlanArtifactFromPhases(confidence, unresolvedGaps, phases, planningLoop)
		result, err := persistIntermediatePlanningIterationInSession(session, manifest, dispatches, scoutReport, phasePlanDraft, confidence, unresolvedGaps, evidenceHash, planningLoop, codexPlanProvenance{
			Dispatches:     dispatches,
			ScoutReport:    scoutReport,
			PhasePlan:      &phasePlanDraft,
			DispatchMode:   dispatchMode,
			ArtifactSource: artifactSource,
			PlanSource:     planSource,
			SourceSummary:  "direct real planning workers",
			RecordWorkers:  false,
		}, &state, publicationTargets)
		if err != nil {
			return nil, err
		}
		result["existing_plan"] = false
		result["refreshed"] = opts.Refresh
		result["colony_mode"] = string(state.EffectiveColonyMode())
		result["granularity"] = string(granularity)
		result["granularity_min"] = granularityMin(granularity)
		result["granularity_max"] = granularityMax(granularity)
		result["verification_depth"] = verificationDepth
		result["planning_phase"] = planningPhase
		result["verification_smart_default"] = verificationSmartDefault
		result["planning_smart_default"] = planningSmartDefault
		result["planning_files"] = []string{filepath.Base(scoutFile), filepath.Base(routeSetterFile)}
		result["plan_artifact"] = filepath.Base(planArtifactFile)
		result["phase_research_dir"] = phaseResearchDir
		result["phase_research_files"] = phaseResearchFiles
		result["dispatch_contract"] = dispatchContract
		result["synthetic"] = false
		result["synthetic_warning"] = ""
		result["survey_docs"] = survey.SurveyDocs
		result["unresolved_clarifications"] = unresolvedClarifications
		result["clarification_warning"] = clarificationWarning
		return result, nil
	}
	if err := validateNewPlanEvidenceContract(phases); err != nil {
		return nil, fmt.Errorf("new plan evidence contract is incomplete: %w", err)
	}

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
	planConfidence := float64(confidence.Overall) / 100.0
	var revision colony.PlanRevision
	state = normalizeLegacyColonyState(state)
	if err := validatePlanManifestBase(root, manifest, state); err != nil {
		return nil, fmt.Errorf("failed to atomically activate plan revision: %w", err)
	}
	acceptedPlan, acceptedRevision, err := activateGeneratedPlan(state.Plan, phases, now, &planConfidence, colony.PlanEvidenceBoundV1, manifest, evidenceHash)
	if err != nil {
		return nil, fmt.Errorf("failed to atomically activate plan revision: %w", err)
	}
	state.State = colony.StateREADY
	state.CurrentPhase = firstBuildablePhase(acceptedPlan.Phases)
	state.BuildStartedAt = nil
	state.PlanGranularity = granularity
	state.VerificationDepth = verificationDepth
	state.Plan = acceptedPlan
	revision = acceptedRevision
	state.Events = append(trimmedEvents(state.Events),
		fmt.Sprintf("%s|planning_scout|plan|Scout summarized surveyed repo context", now.Format(time.RFC3339)),
		fmt.Sprintf("%s|plan_revision_activated|plan|Activated %s (%s): %s", now.Format(time.RFC3339), revision.ID, revision.ReasonType, revision.Reason),
		fmt.Sprintf("%s|plan_generated|plan|Generated %d active phases with %d%% confidence; planning loop stopped: %s", now.Format(time.RFC3339), len(acceptedPlan.Phases), confidence.Overall, planningLoop.StopReason),
	)
	stateBytes, err := marshalSpecificationState(state)
	if err != nil {
		return nil, err
	}
	phases = state.Plan.Phases
	nextPhase := firstBuildablePhase(phases)
	nextCommand := "aether build 1"
	if nextPhase > 0 {
		nextCommand = fmt.Sprintf("aether build %d", nextPhase)
	}
	summary := fmt.Sprintf("Generated %d plan phases with %d%% confidence", len(phases), confidence.Overall)
	targets := []planningSessionTarget{
		{Root: lifecycleTransactionRootData, Path: "COLONY_STATE.json", Content: stateBytes},
		{Root: lifecycleTransactionRootData, Path: planningIterationStateRel, Remove: true},
	}
	sessionTargets, err := planningSessionSummaryTargets(session, state, "plan", nextCommand, summary, now)
	if err != nil {
		return nil, err
	}
	targets = mergePlanningSessionTargets(publicationTargets, targets, sessionTargets)
	if err := commitPlanningSessionTargets(session, "plan-activate", "plan-activate", targets); err != nil {
		return nil, fmt.Errorf("failed to atomically activate plan revision: %w", err)
	}

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
		"evidence_policy":            string(colony.PlanEvidenceBoundV1),
		"plan_revision":              revision,
		"planning_loop":              planningLoop,
		"planning_dir":               planningDir,
		"planning_files":             []string{filepath.Base(scoutFile), filepath.Base(routeSetterFile)},
		"plan_artifact":              filepath.Base(planArtifactFile),
		"phase_research_dir":         phaseResearchDir,
		"phase_research_files":       phaseResearchFiles,
		"dispatches":                 dispatchMaps,
		"dispatch_mode":              dispatchMode,
		"dispatch_contract":          dispatchContract,
		"synthetic":                  opts.Synthetic,
		"synthetic_warning":          planningSyntheticWarningForMode(opts.Synthetic),
		"artifact_source":            artifactSource,
		"plan_source":                planSource,
		"gaps":                       unresolvedGaps,
		"survey_docs":                survey.SurveyDocs,
		"unresolved_clarifications":  unresolvedClarifications,
		"clarification_warning":      clarificationWarning,
		"planning_warning":           planningWarning,
		"next":                       nextCommand,
		"planning_run_id":            manifest.PlanningRunID,
		"iteration":                  manifest.Iteration,
		"evidence_hash":              evidenceHash,
	}
	if manifest.TerritoryRequired {
		result["territory_freshness"] = manifest.Territory
		result["territory_snapshot_id"] = manifest.Territory.SnapshotID
	}
	return result, nil
}

func ensurePlanningSpawnTreeInSession(session *planningMutationSession) error {
	_, exists, err := session.ReadFile(lifecycleTransactionRootData, "spawn-tree.txt")
	if err != nil || exists {
		return err
	}
	return commitPlanningSessionTargets(session, "planning-spawn-tree-init", "planning-spawn-tree-init", []planningSessionTarget{{
		Root: lifecycleTransactionRootData, Path: "spawn-tree.txt", Content: []byte{},
	}})
}

func runCodexPlanAgentDelegateInSession(session *planningMutationSession, state colony.ColonyState, granularity colony.PlanGranularity, planDepth string, unresolvedClarifications int, clarificationWarning string, persistState bool, opts codexPlanOptions) (map[string]interface{}, error) {
	result, err := runCodexPlanPlanOnlyInSession(session, state, granularity, planDepth, unresolvedClarifications, clarificationWarning, persistState, opts)
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

func planningScoutStageDispatchContract(dispatches []codexPlanningDispatch, workerTimeout time.Duration) map[string]interface{} {
	contract := planningDispatchContractForDispatches(dispatches, workerTimeout)
	contract["execution_model"] = "1 staged planning worker: scout only"
	contract["dependency_behavior"] = "This manifest authorizes only the Scout. Route-Setter requires a later receipt-bound manifest after Scout completion."
	return contract
}

// routeSetterResearchBudgetChars bounds the total phase-research content
// appended to the Route-Setter's brief across every candidate phase in one
// planning run. Each phase's individual excerpt is already bounded by
// phaseResearchBriefBudgetChars (resolvePhaseResearchSection); this is the
// separate aggregate ceiling for the sum across all phases, since a plan can
// have far more phases than a single build brief ever touches at once.
const routeSetterResearchBudgetChars = 12000

// renderRouteSetterResearchContent renders the research excerpt for each
// candidate phase (ascending phase order) via resolvePhaseResearchSection,
// each under its own "### Phase N: Name" heading, accumulating until the
// running total would exceed routeSetterResearchBudgetChars. Phases whose
// research exists on disk but did not fit inside the budget are named in a
// closing line rather than silently dropped. Returns "" when no candidate
// has research on disk yet, so callers must not append an empty section --
// that is what keeps the brief byte-identical to the pointer-only output on
// iteration 1, before any research Scout has written anything.
func renderRouteSetterResearchContent(root string, candidates []phaseResearchCandidate) string {
	var b strings.Builder
	total := 0
	budgetExceeded := false
	var overBudget []string
	for _, candidate := range candidates {
		section := resolvePhaseResearchSection(root, candidate.ID)
		if section == "" {
			continue
		}
		if budgetExceeded {
			overBudget = append(overBudget, fmt.Sprintf("%d", candidate.ID))
			continue
		}
		entry := fmt.Sprintf("### Phase %d: %s\n\n%s\n\n", candidate.ID, firstNonEmpty(candidate.Name, "unnamed phase"), section)
		if total+len(entry) > routeSetterResearchBudgetChars {
			budgetExceeded = true
			overBudget = append(overBudget, fmt.Sprintf("%d", candidate.ID))
			continue
		}
		b.WriteString(entry)
		total += len(entry)
	}
	if b.Len() == 0 {
		return ""
	}
	if len(overBudget) > 0 {
		b.WriteString(fmt.Sprintf("_(research for phase(s) %s exceeded the shared research budget — read from `.aether/data/phase-research/`)_\n", strings.Join(overBudget, ", ")))
	}
	return b.String()
}

func runCodexPlanPlanOnlyInSession(session *planningMutationSession, state colony.ColonyState, granularity colony.PlanGranularity, planDepth string, unresolvedClarifications int, clarificationWarning string, persistState bool, opts codexPlanOptions) (map[string]interface{}, error) {
	root := session.RepositoryRoot()
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
	preset, err := resolvePlanningPreset(opts)
	if err != nil {
		return nil, err
	}
	if preset.PresetRequired {
		return planningPresetRequiredResult(state, preset), nil
	}
	planningPhase := colony.Phase{ID: 1}
	boundary, err := materializeOrchestratorBoundaryQuestions("plan", state, planningPhase, planBoundaryQuestionCandidates(state, granularity, planDepth, planningDepth, verificationDepth))
	if err != nil {
		return nil, err
	}
	if len(state.Plan.Phases) > 0 && !opts.Refresh {
		if persistState {
			if err := persistPlanningColonyStateInSession(session, "plan-only-research-documents", state); err != nil {
				return nil, fmt.Errorf("record research documents on colony state: %w", err)
			}
		}
		nextPhase := firstBuildablePhase(state.Plan.Phases)
		nextCommand := "aether build 1"
		if nextPhase > 0 {
			nextCommand = fmt.Sprintf("aether build %d", nextPhase)
		}
		result := map[string]interface{}{
			"plan_only":                  true,
			"planned":                    true,
			"existing_plan":              true,
			"colony_mode":                string(state.EffectiveColonyMode()),
			"goal":                       *state.Goal,
			"phases":                     state.Plan.Phases,
			"plan_revision":              planRevisionSummary(state.Plan),
			"count":                      len(state.Plan.Phases),
			"depth":                      planDepth,
			"planning_depth":             planningDepth,
			"verification_depth":         verificationDepth,
			"planning_loop":              resolvePlanningLoopOptions(planDepth, opts),
			"preset_required":            false,
			"preset_options":             preset.Options,
			"selected_preset":            string(preset.Policy.ID),
			"selection_source":           preset.SelectionSource,
			"target_confidence":          preset.Policy.TargetConfidence,
			"max_iterations":             preset.Policy.PassCap,
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
	revisionContext, err := buildPlanRevisionContext(root, state, opts)
	if err != nil {
		return nil, err
	}
	basePlanStateHash, err := planStateHash(state.Plan)
	if err != nil {
		return nil, fmt.Errorf("hash active plan before planning: %w", err)
	}
	approvedSpecification, err := requireApprovedPlanningSpecification(root, state)
	if err != nil {
		return nil, err
	}
	baseRevisionID, baseRevisionHash := planningBaseRevisionIdentity(state.Plan, basePlanStateHash)

	survey, err := loadCodexSurveyContext(root)
	if err != nil {
		return nil, err
	}
	generatedAt := time.Now().UTC()

	// A run already waiting on a worker is resumed BEFORE a second run can be
	// started beside it. Until this branch existed the planner always prepared
	// a fresh Scout stage, so a run parked at route_running was unreachable:
	// re-running produced a stray second run at the Scout stage while the
	// Route-Setter stage manifest the first run had already issued had nowhere
	// to be submitted. --refresh stays the deliberate way to abandon an
	// in-flight run and start over.
	if !opts.Refresh {
		resume, resumeErr := resolvePlanningStageResume(root)
		if resumeErr != nil {
			return nil, resumeErr
		}
		if resume != nil {
			resumeManifest, buildErr := buildPlanningStageResumeManifest(root, state, *resume, granularity, planDepth, planningDepth, verificationDepth, survey, resolveCodexWorkerContext(), opts, generatedAt)
			if buildErr != nil {
				return nil, buildErr
			}
			// Questions raised at this boundary are already materialized above
			// and travel with the resumed dispatch exactly as they do with a
			// fresh one -- a resumed run must not be the one path where an
			// unanswered owner question goes unmentioned.
			resumeManifest.BoundaryQuestions = append([]discussQuestion(nil), boundary.Questions...)
			resumeManifest.BoundaryQuestionCount = len(boundary.Questions)
			resumeManifest.BoundaryQuestionsCreated = boundary.Created
			resumeManifest.BoundaryQuestionsExisting = boundary.Existing
			envelope := planningStageResumeEnvelope(state, *resume, resumeManifest, granularity, planDepth, planningDepth, verificationDepth)
			envelope["boundary_questions"] = resumeManifest.BoundaryQuestions
			envelope["boundary_question_count"] = resumeManifest.BoundaryQuestionCount
			return envelope, nil
		}
	}
	planningLoop := resolvePlanningLoopOptions(planDepth, opts)
	iterationSeed := planningManifestIterationSeed(*state.Goal, root, planDepth, planningDepth, planningLoop, revisionContext, generatedAt)
	iteration := iterationSeed.LastIteration + 1
	if iteration < 1 {
		iteration = 1
	}
	planningLoop.History = append([]codexPlanningLoopSample{}, iterationSeed.History...)
	planningLoop.Iterations = iterationSeed.LastIteration
	planningLoop.FinalConfidence = iterationSeed.PreviousConfidence
	planningLoop.Gaps = append([]string{}, iterationSeed.SelectedGaps...)

	dispatches := plannedPlanningWorkersForGoal(root, *state.Goal)
	for i := range dispatches {
		if dispatches[i].Caste == "scout" {
			dispatches = []codexPlanningDispatch{dispatches[i]}
			break
		}
	}
	specs := planningWorkerSpecsForGoal(*state.Goal)
	iterationAppendix := planningIterationAppendix(iterationSeed, iteration) + renderPlanRevisionWorkerAppendix(root, revisionContext)
	for i := range dispatches {
		dispatches[i].Status = "planned"
		dispatches[i].Brief = renderPlanningWorkerBrief(root, survey, specs[i])
		if iterationAppendix != "" {
			dispatches[i].Brief += iterationAppendix
		}
	}
	artifactSnapshots := snapshotRelativeFiles(root,
		filepath.ToSlash(filepath.Join(".aether", "data", "planning")),
		filepath.ToSlash(filepath.Join(".aether", "data", "phase-research")),
	)
	dispatchContract := planningScoutStageDispatchContract(dispatches, opts.WorkerTimeout)

	// Compute the colony-prime capsule once, for this plan-only wrapper
	// manifest only — this function (runCodexPlanPlanOnly) is reached
	// exclusively via the explicit --plan-only flag or the agent-delegate
	// route (runCodexPlanAgentDelegate), both of which hand this manifest to
	// a platform wrapper rather than dispatching workers directly. The
	// in-process/native lane (dispatchRealPlanningWorkersWithIterationContext)
	// never calls this function and computes its own capsule per dispatch.
	// This is the single call site for this field on the plan-only lane; it
	// must never be computed inside a per-dispatch loop.
	contextCapsule := resolveCodexWorkerContext()
	header, runningStage, stageManifest, err := preparePlanningScoutStage(
		root,
		state,
		*state.Goal,
		preset.Policy,
		approvedSpecification,
		baseRevisionID,
		baseRevisionHash,
		contextCapsule,
		survey,
		iterationSeed,
		opts.Refresh,
		generatedAt,
	)
	if err != nil {
		return nil, err
	}
	for i := range dispatches {
		dispatches[i].Brief += renderAutomaticPhaseResearchPolicy(header.ResearchPolicy)
	}
	var persistedState *colony.ColonyState
	if persistState {
		persistedState = &state
	}
	if err := persistPlanningScoutStageAndStateInSession(session, header, runningStage, stageManifest, persistedState); err != nil {
		return nil, err
	}
	for i := range dispatches {
		dispatches[i].Stage = string(planningStageScoutRunning)
		dispatches[i].TaskID = stageManifest.AuthorizationID
		dispatches[i].StageManifest = &stageManifest
	}
	dispatchContract = planningScoutStageDispatchContract(dispatches, opts.WorkerTimeout)

	manifest := codexPlanManifest{
		Goal:                      *state.Goal,
		Root:                      root,
		GeneratedAt:               generatedAt.Format(time.RFC3339),
		BaseRevisionID:            baseRevisionID,
		BasePlanStateHash:         basePlanStateHash,
		ColonyMode:                string(state.EffectiveColonyMode()),
		Refresh:                   opts.Refresh,
		Revision:                  revisionContext,
		ExistingPlan:              len(state.Plan.Phases) > 0,
		ExistingPhaseCount:        len(state.Plan.Phases),
		Synthetic:                 opts.Synthetic,
		SyntheticWarning:          planningSyntheticWarningForMode(opts.Synthetic),
		PlanningRunID:             stageManifest.RunID,
		Iteration:                 stageManifest.Pass,
		TargetConfidence:          planningLoop.TargetConfidence,
		MaxIterations:             planningLoop.MaxIterations,
		SelectedPreset:            preset.Policy.ID,
		PresetSelectionSource:     preset.SelectionSource,
		PreviousConfidence:        iterationSeed.PreviousConfidence,
		PreviousEvidenceHash:      iterationSeed.PreviousEvidenceHash,
		SelectedGaps:              append([]string{}, iterationSeed.SelectedGaps...),
		PreviousPlanDraft:         iterationSeed.PreviousPlanDraft,
		ExpectedWorkers:           append([]codexPlanningDispatch{}, dispatches...),
		Depth:                     planDepth,
		Granularity:               string(granularity),
		GranularityMin:            granularityMin(granularity),
		GranularityMax:            granularityMax(granularity),
		PlanningDepth:             planningDepth,
		VerificationDepth:         verificationDepth,
		PlanningLoop:              planningLoop,
		Survey:                    survey,
		Dispatches:                dispatches,
		Snapshots:                 artifactSnapshots,
		DispatchMode:              "plan-only",
		DispatchContract:          dispatchContract,
		FinalizeSurface:           "pending",
		RequiresFinalizer:         true,
		ResearchPolicy:            &header.ResearchPolicy,
		PhaseResearchEvidence:     append([]phaseResearchEvidenceAttribution(nil), header.PhaseResearchEvidence...),
		StageManifest:             &stageManifest,
		PlanningRunHeader:         &header,
		ContextCapsule:            contextCapsule,
		BoundaryQuestions:         append([]discussQuestion(nil), boundary.Questions...),
		BoundaryQuestionCount:     len(boundary.Questions),
		BoundaryQuestionsCreated:  boundary.Created,
		BoundaryQuestionsExisting: boundary.Existing,
	}
	if opts.Territory != nil {
		attachTerritoryToPlanManifest(&manifest, *opts.Territory)
	}

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
		"planning_loop":              manifest.PlanningLoop,
		"planning_run_id":            manifest.PlanningRunID,
		"iteration":                  manifest.Iteration,
		"target_confidence":          manifest.TargetConfidence,
		"max_iterations":             manifest.MaxIterations,
		"preset_required":            false,
		"preset_options":             preset.Options,
		"selected_preset":            string(preset.Policy.ID),
		"selection_source":           preset.SelectionSource,
		"previous_confidence":        manifest.PreviousConfidence,
		"selected_gaps":              manifest.SelectedGaps,
		"verification_smart_default": verificationSmartDefault,
		"planning_smart_default":     planningSmartDefault,
		"planning_phase":             planningPhase,
		"granularity":                string(granularity),
		"granularity_min":            granularityMin(granularity),
		"granularity_max":            granularityMax(granularity),
		"plan_manifest":              manifest,
		"planning_manifest":          manifest,
		"stage_manifest":             stageManifest,
		"planning_run_header":        header,
		"revision_request":           revisionContext,
		"dispatches":                 dispatches,
		"dispatch_count":             len(dispatches),
		"dispatch_mode":              "plan-only",
		"dispatch_contract":          dispatchContract,
		"unresolved_clarifications":  unresolvedClarifications,
		"clarification_warning":      clarificationWarning,
		"research_policy":            header.ResearchPolicy,
		"phase_research_evidence":    header.PhaseResearchEvidence,
		"next":                       "spawn wrapper planning agents, then record completion",
		"wrapper_contract": map[string]interface{}{
			"source_command":          "AETHER_OUTPUT_MODE=json aether plan --plan-only --preset <fast|balanced|deep|exhaustive>",
			"spawn_log_required":      true,
			"spawn_complete_required": true,
			"finalize_surface":        "pending",
			"runtime_state_only":      true,
			"planning_depth":          planningDepth,
			"planning_loop":           manifest.PlanningLoop,
		},
	}
	if manifest.TerritoryRequired {
		result["territory_freshness"] = manifest.Territory
		result["territory_snapshot_id"] = manifest.Territory.SnapshotID
	}
	addBoundaryQuestionResultFields(result, boundary)
	afterDiscussKey := candidatePlanPreset
	if opts.Refresh {
		afterDiscussKey = candidatePlanRefresh
	}
	afterDiscussNext := availableCandidateCommand(afterDiscussKey, string(preset.Policy.ID))
	addOrchestratorBoundaryGuidance(result, "plan", state, afterDiscussNext, boundary.Questions)
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
		dispatches[i].PermissionProfile = codex.PermissionProfileForCaste(dispatches[i].Caste)
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

func planningWorkerSpecsForGoal(_ string) []planningWorkerSpec {
	specs := make([]planningWorkerSpec, len(planningWorkerSpecs))
	copy(specs, planningWorkerSpecs)
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

// dispatchRealPlanningWorkers attempts worker invocation for planning.
// If the invoker is not available, it returns nil, nil so callers can choose
// whether to fail closed, delegate to a wrapper, or run explicit synthetic mode.
func dispatchRealPlanningWorkers(ctx context.Context, root string, invoker codex.WorkerInvoker) ([]codexPlanningDispatch, error) {
	return dispatchRealPlanningWorkersWithTimeout(ctx, root, codexSurveyContext{}, invoker, 0)
}

func dispatchRealPlanningWorkersWithTimeout(ctx context.Context, root string, survey codexSurveyContext, invoker codex.WorkerInvoker, timeoutOverride time.Duration, goalOpt ...string) ([]codexPlanningDispatch, error) {
	return dispatchRealPlanningWorkersWithIterationContext(ctx, root, survey, invoker, timeoutOverride, "", goalOpt...)
}

func dispatchRealPlanningWorkersWithIterationContext(ctx context.Context, root string, survey codexSurveyContext, invoker codex.WorkerInvoker, timeoutOverride time.Duration, iterationAppendix string, goalOpt ...string) ([]codexPlanningDispatch, error) {
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
	// PheromoneSection is deliberately left unset (D-190-03-A / 190-05): capsule
	// already renders "## Pheromone Signals" unconditionally whenever a signal
	// is active (cmd/colony_prime_context.go:571). Populating a second,
	// independent PheromoneSection field here would deliver the same steering
	// text twice. See resolvePheromoneSection's doc comment for which callers
	// still need it.
	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	results := make([]codex.DispatchResult, 0, len(specs))
	workerTimeout := effectivePlanningDispatchTimeout(timeoutOverride)
	scoutGuidance := ""
	for i, spec := range specs {
		agentName := strings.TrimSuffix(spec.AgentFile, ".toml")
		dispatch := codex.WorkerDispatch{
			ID:             fmt.Sprintf("planning-%d", i),
			WorkerName:     planned[i].Name,
			AgentName:      agentName,
			AgentTOMLPath:  dispatchAgentPath(root, invoker, agentName),
			Caste:          spec.Caste,
			TaskID:         fmt.Sprintf("plan-%d", i),
			ContextCapsule: capsule,
			// D-190-05-A / 190-06: renderRelatedWorkflowHandoffSection, not
			// renderWorkerHandoffSection -- capsule (above) already renders
			// "## Previous Worker Handoffs" for "build"-workflow records
			// (cmd/colony_prime_context.go:695). Same shape D-190-05-A found
			// for continue, empirically reproduced here (throwaway probe,
			// 190-06) whenever both a build- and a plan-workflow handoff exist.
			HandoffSection:    renderRelatedWorkflowHandoffSection("plan", 0, planned[i].Name),
			Workflow:          "plan",
			SkillSection:      resolveSkillSectionForWorkflow("plan", spec.Caste, spec.Task),
			Root:              root,
			Wave:              i + 1,
			Timeout:           workerTimeout,
			PermissionProfile: planned[i].PermissionProfile,
		}
		dispatch.TaskBrief = renderPlanningWorkerBrief(root, survey, spec, scoutGuidance)
		if strings.TrimSpace(iterationAppendix) != "" {
			dispatch.TaskBrief += iterationAppendix
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
			Caste:             planned.Caste,
			Name:              planned.Name,
			Task:              planned.Task,
			Outputs:           planned.Outputs,
			Status:            "spawned",
			PermissionProfile: planned.PermissionProfile,
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
		SourceAnchors:    []string{},
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
	if payload := readSummary("anchors.json"); payload != nil {
		ctx.SourceAnchors = append(ctx.SourceAnchors, jsonStringSlice(payload["source_anchors"])...)
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
	ctx.SourceAnchors = uniqueSortedStrings(ctx.SourceAnchors)
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

const (
	planningLoopMinTarget        = 70
	planningLoopMaxTarget        = 99
	planningLoopMinIterations    = 2
	planningLoopMaxIterations    = 12
	planningLoopStallThreshold   = 5
	planningLoopStallLimit       = 2
	planningLoopPendingStop      = "pending"
	planningLoopTargetReached    = "target_reached"
	planningLoopMaxIterationsHit = "max_iterations"
	planningLoopStalled          = "stalled"
	planningLoopAccepted         = "accepted"
	planningIterationStateRel    = "planning/iteration-state.json"
)

func planningLoopPreset(planDepth string) (int, int) {
	switch strings.ToLower(strings.TrimSpace(planDepth)) {
	case "fast":
		return 80, 4
	case "deep":
		return 95, 8
	case "exhaustive":
		return 99, 12
	default:
		return 90, 6
	}
}

func resolvePlanningLoopOptions(planDepth string, opts codexPlanOptions) codexPlanningLoop {
	target, maxIterations := planningLoopPreset(planDepth)
	if opts.TargetConfidence > 0 {
		target = clampInt(opts.TargetConfidence, planningLoopMinTarget, planningLoopMaxTarget)
	}
	if opts.MaxIterations > 0 {
		maxIterations = clampInt(opts.MaxIterations, planningLoopMinIterations, planningLoopMaxIterations)
	}
	return codexPlanningLoop{
		TargetConfidence: target,
		MaxIterations:    maxIterations,
		StallThreshold:   planningLoopStallThreshold,
		StallLimit:       planningLoopStallLimit,
		Accept:           opts.Accept,
		StopReason:       planningLoopPendingStop,
	}
}

func loadPlanningIterationState() (codexPlanIterationState, bool) {
	var state codexPlanIterationState
	if store == nil {
		return state, false
	}
	if err := store.LoadJSON(planningIterationStateRel, &state); err != nil {
		return codexPlanIterationState{}, false
	}
	if strings.TrimSpace(state.PlanningRunID) == "" || state.LastIteration < 1 {
		return codexPlanIterationState{}, false
	}
	return state, true
}

func planningRunID(goal, root string, generatedAt time.Time) string {
	source := fmt.Sprintf("%s|%s|%s", strings.TrimSpace(goal), strings.TrimSpace(root), generatedAt.UTC().Format(time.RFC3339Nano))
	sum := sha256.Sum256([]byte(source))
	return "plan-" + hex.EncodeToString(sum[:])[:12]
}

func planningIterationStateMatches(state codexPlanIterationState, goal, root, planDepth, planningDepth string, loop codexPlanningLoop, revision *codexPlanRevisionContext) bool {
	return strings.TrimSpace(state.Goal) == strings.TrimSpace(goal) &&
		strings.TrimSpace(state.Root) == strings.TrimSpace(root) &&
		strings.TrimSpace(state.Depth) == strings.TrimSpace(planDepth) &&
		strings.TrimSpace(state.PlanningDepth) == strings.TrimSpace(planningDepth) &&
		state.TargetConfidence == loop.TargetConfidence &&
		state.MaxIterations == loop.MaxIterations &&
		planRevisionContextsEqual(state.Revision, revision)
}

func planningManifestIterationSeed(goal, root, planDepth, planningDepth string, loop codexPlanningLoop, revision *codexPlanRevisionContext, generatedAt time.Time) codexPlanIterationState {
	if previous, ok := loadPlanningIterationState(); ok && planningIterationStateMatches(previous, goal, root, planDepth, planningDepth, loop, revision) {
		return previous
	}
	return codexPlanIterationState{
		PlanningRunID:    planningRunID(goal, root, generatedAt),
		Goal:             strings.TrimSpace(goal),
		Root:             strings.TrimSpace(root),
		Depth:            strings.TrimSpace(planDepth),
		PlanningDepth:    strings.TrimSpace(planningDepth),
		TargetConfidence: loop.TargetConfidence,
		MaxIterations:    loop.MaxIterations,
		Revision:         revision,
		UpdatedAt:        generatedAt.UTC().Format(time.RFC3339),
	}
}

func planningIterationAppendix(seed codexPlanIterationState, nextIteration int) string {
	if nextIteration <= 1 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\n## Runtime Planning Iteration Context\n")
	b.WriteString(fmt.Sprintf("- Planning run: %s\n", seed.PlanningRunID))
	b.WriteString(fmt.Sprintf("- Iteration: %d\n", nextIteration))
	b.WriteString(fmt.Sprintf("- Previous confidence: %d%%\n", seed.PreviousConfidence))
	if len(seed.SelectedGaps) > 0 {
		b.WriteString("- Targeted gaps for this iteration:\n")
		for _, gap := range seed.SelectedGaps {
			b.WriteString(fmt.Sprintf("  - %s\n", gap))
		}
	}
	if seed.PreviousPlanDraft != nil && len(seed.PreviousPlanDraft.Phases) > 0 {
		b.WriteString("- Previous plan draft summary:\n")
		for i, phase := range limitWorkerPlanPhases(seed.PreviousPlanDraft.Phases, 4) {
			b.WriteString(fmt.Sprintf("  - Phase %d: %s\n", i+1, strings.TrimSpace(phase.Name)))
		}
	}
	b.WriteString("- Gather fresh evidence or resolve the targeted gaps. Do not raise confidence by restating the previous evidence.\n")
	return b.String()
}

func limitWorkerPlanPhases(phases []codexWorkerPlanPhase, limit int) []codexWorkerPlanPhase {
	if len(phases) <= limit {
		return append([]codexWorkerPlanPhase{}, phases...)
	}
	return append([]codexWorkerPlanPhase{}, phases[:limit]...)
}

func evaluatePlanningLoop(confidence codexPlanConfidence, unresolvedGaps []string, opts codexPlanOptions, planDepth string) codexPlanningLoop {
	loop := resolvePlanningLoopOptions(planDepth, opts)
	overall := clampInt(int(confidence.Overall), 0, 100)
	gaps := limitStrings(uniqueSortedStrings(unresolvedGaps), 4)
	loop.FinalConfidence = overall
	loop.Gaps = gaps

	// Aether currently has one completed planning evidence pass at this point.
	// Do not manufacture extra loop samples without dispatching fresh workers.
	loop.Iterations = 1
	loop.History = append(loop.History, codexPlanningLoopSample{
		Iteration:     1,
		Confidence:    overall,
		Delta:         0,
		StallCount:    0,
		Gaps:          gaps,
		SelectedGaps:  selectPlanningIterationGaps(gaps),
		Evidence:      planningEvidenceSummary(confidence, gaps),
		EvidenceCount: planningEvidenceCount(confidence),
		EvidenceHash:  planningEvidenceHash(confidence, gaps),
	})

	if opts.Accept {
		loop.StopReason = planningLoopAccepted
		loop.AcceptedBelowTarget = overall < loop.TargetConfidence
		return loop
	}
	if overall >= loop.TargetConfidence {
		loop.StopReason = planningLoopTargetReached
		return loop
	}
	if loop.MaxIterations <= 1 {
		loop.StopReason = planningLoopMaxIterationsHit
		return loop
	}

	return loop
}

func selectPlanningIterationGaps(gaps []string) []string {
	return limitStrings(uniqueSortedStrings(gaps), 2)
}

func planningEvidenceSummary(confidence codexPlanConfidence, gaps []string) string {
	return fmt.Sprintf(
		"single planning evidence pass; confidence inputs knowledge=%d requirements=%d risks=%d dependencies=%d effort=%d overall=%d; unresolved_gaps=%d",
		confidence.Knowledge,
		confidence.Requirements,
		confidence.Risks,
		confidence.Dependencies,
		confidence.Effort,
		confidence.Overall,
		len(gaps),
	)
}

func planningEvidenceCount(confidence codexPlanConfidence) int {
	count := 0
	for _, score := range []int{int(confidence.Knowledge), int(confidence.Requirements), int(confidence.Risks), int(confidence.Dependencies), int(confidence.Effort), int(confidence.Overall)} {
		if score > 0 {
			count++
		}
	}
	return count
}

func planningEvidenceHash(confidence codexPlanConfidence, gaps []string) string {
	source := fmt.Sprintf(
		"%d|%d|%d|%d|%d|%d|%s",
		confidence.Knowledge,
		confidence.Requirements,
		confidence.Risks,
		confidence.Dependencies,
		confidence.Effort,
		confidence.Overall,
		strings.Join(uniqueSortedStrings(gaps), "\x00"),
	)
	sum := sha256.Sum256([]byte(source))
	return hex.EncodeToString(sum[:])
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
	// The condensed map digest (WIRE-03): the same content the build and
	// research briefs get, from the one shared resolveSurveyDigestSection
	// call site per brief. No age-line dedup needed here -- nothing else in
	// this brief renders surveyStalenessNotice(), so the digest is its only
	// source. Guarded by its own empty-string check so a colony with no
	// survey reports produces a byte-identical planning brief to before this
	// digest existed (TestBriefsAreUnchangedWithoutASurvey).
	if digestSection := resolveSurveyDigestSection(); digestSection != "" {
		b.WriteString("\n")
		b.WriteString(digestSection)
		b.WriteString("\n")
	}
	if spec.Caste == "route_setter" {
		b.WriteString("- Read scout output before drafting phases if it exists: ")
		b.WriteString(filepath.ToSlash(filepath.Join(planningDir, "SCOUT.md")))
		b.WriteString("; if it is missing, proceed from the survey context and note the missing scout artifact in blockers only if it prevents a useful plan.")
		b.WriteString("\n")
	}
	b.WriteString("- Repo inspection rule: use targeted reads to confirm or extend survey findings; do not trawl the whole tree unless the survey lacks the needed detail.\n")
	// Research the operator pointed this colony at. This is the only injection
	// point that works on a fresh colony's first plan -- phase research cannot
	// run until phases exist -- and it covers both the Scout and the
	// Route-Setter, and both the in-process and host-manifest paths, because
	// both call this function.
	if researchSection := resolveColonyResearchSection(root, loadColonyResearchDocs(root)); researchSection != "" {
		b.WriteString("\n")
		b.WriteString(researchSection)
		b.WriteString("\n\n")
	}
	if len(survey.SourceAnchors) > 0 {
		b.WriteString(fmt.Sprintf("- Source anchors available: %d repo-owned files from survey. Prefer referencing these files in task goals.\n", len(survey.SourceAnchors)))
	}
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
		// No CLI instructions for these two: they have no Bash tool by
		// design, so briefing them to run `aether review-ledger-write` was an
		// unsatisfiable task they answered by self-reporting blocked.
		b.WriteString("This is a review task. Report findings in this result's summary and blockers — do not run CLI commands and do not modify repo source files. Return status `blocked` if advancement is unsafe.\n\n")
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
		b.WriteString(" using the phase-plan schema below.\n")
		b.WriteString(renderPhasePlanSchemaGuidance())
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
		b.WriteString(" using the phase-plan schema below.\n")
		b.WriteString(renderPhasePlanSchemaGuidance())
	}
	b.WriteString("\nPlan the colony at ")
	b.WriteString(root)
	return b.String()
}

func renderPhasePlanSchemaGuidance() string {
	return strings.TrimSpace(`
- JSON shape:
  {"phases":[{"name":"","description":"","tasks":[{"goal":"","constraints":[],"hints":[],"success_criteria":["criterion"],"evidence_requirements":[{"criterion":"criterion","artifacts":["path/to/output"],"checks":["tests"]}],"depends_on":[]}],"success_criteria":["criterion"],"evidence_requirements":[{"criterion":"criterion","artifacts":["path/to/output"],"checks":["build","tests","claims","watcher"]}]}],"confidence":{"knowledge":0,"requirements":0,"risks":0,"dependencies":0,"effort":0,"overall":0},"gaps":[]}
- Do not include task ids in phase-plan.json. Aether assigns task ids by array order after empty-goal tasks are ignored.
- Dependency ids must use those assigned ids only: first task in first phase is "1.1", second is "1.2", first task in second phase is "2.1".
- depends_on must be an array of task id strings such as ["1.1"]. Do not use task text, file paths, descriptions, or custom ids like "P1-T1".
- Bind every success criterion when using evidence_requirements. artifacts are exact repository-relative project files; checks are selected from build, types, lint, tests, claims, and watcher. Do not use .aether/data files as product evidence.
- Only list a file under artifacts when THIS task changes it. artifacts is a claim that the task produced the file, and the runtime verifies it against the build's claim lists.
- Never bind an "unchanged", "untouched", "still passes", or "remains dependency-free" criterion to artifacts. Those assert pre-existing state, and the task never claims those files, so the phase blocks with "artifact <path> was not claimed by the current build" and can only be recovered with manual --reconcile-task/--read-only-artifact flags.
- Express a "nothing regressed" criterion with checks instead: {"criterion":"the suite still passes","checks":["tests"]} with no artifacts entry.
- A deliberately-RED phase — one whose deliverable is FAILING tests that prove a defect exists (TDD red-first, "reproduce and lock the bug") — must set "expect_failing_tests": true on the phase. Verification then expects the test run to fail and blocks if it passes. Never plan a red-first phase without this field: the tests check treats any failure as a blocker otherwise, and the phase can never advance.
`) + "\n"
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

func loadWorkerPlanArtifact(root string, snapshots map[string]codexArtifactSnapshot, dispatches []codexPlanningDispatch) (codexWorkerPlanArtifact, bool, string, error) {
	relPath := filepath.ToSlash(filepath.Join(".aether", "data", "planning", "phase-plan.json"))
	if !shouldPreserveWorkerArtifact(root, relPath, snapshots, claimedPlanningFiles(dispatches)) {
		return codexWorkerPlanArtifact{}, false, "", nil
	}

	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relPath)))
	if err != nil {
		return codexWorkerPlanArtifact{}, false, "", fmt.Errorf("Route-Setter wrote phase-plan.json but it could not be read: %w", err)
	}

	var artifact codexWorkerPlanArtifact
	if err := json.Unmarshal(data, &artifact); err != nil {
		return codexWorkerPlanArtifact{}, false, "", fmt.Errorf("Route-Setter phase-plan.json is invalid JSON: %w", err)
	}
	if len(artifact.Phases) == 0 {
		return codexWorkerPlanArtifact{}, false, "", fmt.Errorf("Route-Setter phase-plan.json contains no phases")
	}
	normalized, repairs, err := normalizeWorkerPlanArtifactDependencies(artifact)
	if err != nil {
		return codexWorkerPlanArtifact{}, false, "", err
	}
	return normalized, true, strings.Join(repairs, " "), nil
}

var (
	phasePlanTaskIDPattern       = regexp.MustCompile(`^\d+\.\d+$`)
	phasePlanCustomTaskIDPattern = regexp.MustCompile(`(?i)^p(?:hase)?\s*(\d+)\s*[-_:\s]*t(?:ask)?\s*(\d+)$`)
)

func normalizeWorkerPlanArtifactDependencies(artifact codexWorkerPlanArtifact) (codexWorkerPlanArtifact, []string, error) {
	normalized := artifact
	normalized.Phases = append([]codexWorkerPlanPhase{}, artifact.Phases...)
	knownIDs := workerPlanTaskIDs(artifact)
	repairs := []string{}

	for phaseIndex := range normalized.Phases {
		sourcePhase := artifact.Phases[phaseIndex]
		normalized.Phases[phaseIndex].Tasks = append([]codexWorkerPlanTask{}, sourcePhase.Tasks...)
		normalized.Phases[phaseIndex].SuccessCriteria = append([]string{}, sourcePhase.SuccessCriteria...)
		normalized.Phases[phaseIndex].EvidenceRequirements = normalizeWorkerCriterionRequirements(sourcePhase.EvidenceRequirements)
		for taskIndex := range normalized.Phases[phaseIndex].Tasks {
			task := sourcePhase.Tasks[taskIndex]
			normalized.Phases[phaseIndex].Tasks[taskIndex].EvidenceRequirements = normalizeWorkerCriterionRequirements(task.EvidenceRequirements)
			taskID := fmt.Sprintf("%d.%d", phaseIndex+1, taskIndex+1)
			if strings.TrimSpace(task.Goal) == "" {
				normalized.Phases[phaseIndex].Tasks[taskIndex].DependsOn = nil
				continue
			}
			deps, depRepairs, err := normalizeWorkerPlanTaskDependencies(taskID, task.DependsOn, knownIDs)
			if err != nil {
				return codexWorkerPlanArtifact{}, nil, err
			}
			normalized.Phases[phaseIndex].Tasks[taskIndex].DependsOn = deps
			repairs = append(repairs, depRepairs...)
		}
	}
	return normalized, uniqueSortedStrings(repairs), nil
}

func workerPlanTaskIDs(artifact codexWorkerPlanArtifact) map[string]bool {
	ids := map[string]bool{}
	for phaseIndex, phase := range artifact.Phases {
		for taskIndex, task := range phase.Tasks {
			if strings.TrimSpace(task.Goal) == "" {
				continue
			}
			ids[fmt.Sprintf("%d.%d", phaseIndex+1, taskIndex+1)] = true
		}
	}
	return ids
}

func normalizeWorkerPlanTaskDependencies(taskID string, dependsOn []string, knownIDs map[string]bool) ([]string, []string, error) {
	normalized := make([]string, 0, len(dependsOn))
	repairs := []string{}
	for _, raw := range dependsOn {
		dep := strings.TrimSpace(raw)
		if dep == "" || strings.EqualFold(dep, "none") || strings.EqualFold(dep, "null") {
			continue
		}
		switch {
		case phasePlanTaskIDPattern.MatchString(dep):
			if !knownIDs[dep] {
				return nil, nil, unknownPhasePlanDependencyError(taskID, dep, knownIDs)
			}
			normalized = append(normalized, dep)
		case phasePlanCustomTaskIDPattern.MatchString(dep):
			matches := phasePlanCustomTaskIDPattern.FindStringSubmatch(dep)
			canonical := fmt.Sprintf("%s.%s", matches[1], matches[2])
			if !knownIDs[canonical] {
				return nil, nil, unknownPhasePlanDependencyError(taskID, dep, knownIDs)
			}
			normalized = append(normalized, canonical)
			repairs = append(repairs, fmt.Sprintf("Normalized phase-plan dependency %q to %q for task %s.", dep, canonical, taskID))
		default:
			return nil, nil, fmt.Errorf("phase-plan.json task %s depends_on %q is not a valid task id; use runtime task IDs like %q. Task IDs are assigned by order, so phase 1 task 1 is 1.1. Do not use task text, file paths, or custom dependency names", taskID, dep, firstKnownTaskIDExample(knownIDs))
		}
	}
	return uniqueSortedStrings(normalized), repairs, nil
}

func unknownPhasePlanDependencyError(taskID, dep string, knownIDs map[string]bool) error {
	return fmt.Errorf("phase-plan.json task %s depends_on %q references no buildable task; known task IDs are: %s", taskID, dep, strings.Join(knownWorkerPlanTaskIDs(knownIDs), ", "))
}

func knownWorkerPlanTaskIDs(knownIDs map[string]bool) []string {
	ids := make([]string, 0, len(knownIDs))
	for id := range knownIDs {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		left := strings.Split(ids[i], ".")
		right := strings.Split(ids[j], ".")
		if len(left) != 2 || len(right) != 2 {
			return ids[i] < ids[j]
		}
		if left[0] == right[0] {
			return left[1] < right[1]
		}
		return left[0] < right[0]
	})
	return ids
}

func firstKnownTaskIDExample(knownIDs map[string]bool) string {
	ids := knownWorkerPlanTaskIDs(knownIDs)
	if len(ids) == 0 {
		return "1.1"
	}
	return ids[0]
}

func runCodexPlanRepairArtifact(root string) (map[string]interface{}, error) {
	if store == nil {
		return nil, fmt.Errorf("no store initialized")
	}
	relPath := filepath.ToSlash(filepath.Join(".aether", "data", "planning", "phase-plan.json"))
	path := filepath.Join(root, filepath.FromSlash(relPath))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", relPath, err)
	}
	var artifact codexWorkerPlanArtifact
	if err := json.Unmarshal(data, &artifact); err != nil {
		return nil, fmt.Errorf("parse %s: %w", relPath, err)
	}
	if len(artifact.Phases) == 0 {
		return nil, fmt.Errorf("%s contains no phases", relPath)
	}
	normalized, repairs, err := normalizeWorkerPlanArtifactDependencies(artifact)
	if err != nil {
		return nil, err
	}
	phases := buildWorkerPlanPhases(normalized)
	if len(phases) == 0 || buildablePlanTaskCount(phases) == 0 {
		return nil, fmt.Errorf("%s contains no buildable tasks", relPath)
	}
	if err := colony.DetectCycles(phases); err != nil {
		return nil, fmt.Errorf("phase-plan dependency validation failed: %w", err)
	}
	repaired := len(repairs) > 0
	if repaired {
		encoded, err := json.MarshalIndent(normalized, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("encode repaired %s: %w", relPath, err)
		}
		encoded = append(encoded, '\n')
		if err := os.WriteFile(path, encoded, 0644); err != nil {
			return nil, fmt.Errorf("write repaired %s: %w", relPath, err)
		}
	}
	next := "aether plan-finalize --completion-file <file>"
	updateSessionSummary("plan", next, "Repaired and validated phase-plan dependency references")
	return map[string]interface{}{
		"repaired":      repaired,
		"repairs":       repairs,
		"validated":     true,
		"phase_plan":    relPath,
		"phase_count":   len(phases),
		"task_count":    buildablePlanTaskCount(phases),
		"next":          next,
		"repair_source": "phase-plan dependency normalizer",
	}, nil
}

func buildWorkerPlanPhases(artifact codexWorkerPlanArtifact) []colony.Phase {
	phases := make([]colony.Phase, 0, len(artifact.Phases))
	for i, sourcePhase := range artifact.Phases {
		phase := colony.Phase{
			ID:                   i + 1,
			Name:                 strings.TrimSpace(sourcePhase.Name),
			Description:          strings.TrimSpace(sourcePhase.Description),
			Status:               colony.PhasePending,
			Mode:                 resolveAuthoredPhaseMode(sourcePhase.Mode, sourcePhase.Name, sourcePhase.Description),
			ExpectFailingTests:   sourcePhase.ExpectFailingTests,
			Tasks:                []colony.Task{},
			SuccessCriteria:      uniqueSortedStrings(sourcePhase.SuccessCriteria),
			EvidenceRequirements: normalizeWorkerCriterionRequirements(sourcePhase.EvidenceRequirements),
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
				ID:                   &taskID,
				Goal:                 goal,
				Status:               colony.TaskPending,
				Constraints:          uniqueSortedStrings(sourceTask.Constraints),
				Hints:                uniqueSortedStrings(sourceTask.Hints),
				SuccessCriteria:      uniqueSortedStrings(sourceTask.SuccessCriteria),
				EvidenceRequirements: normalizeWorkerCriterionRequirements(sourceTask.EvidenceRequirements),
				DependsOn:            uniqueSortedStrings(sourceTask.DependsOn),
			})
		}
		phases = append(phases, phase)
	}
	return phases
}

func workerPlanArtifactFromPhases(confidence codexPlanConfidence, unresolvedGaps []string, phases []colony.Phase, planningLoop codexPlanningLoop) codexWorkerPlanArtifact {
	artifact := codexWorkerPlanArtifact{
		Confidence:   confidence,
		Gaps:         limitStrings(uniqueSortedStrings(unresolvedGaps), 4),
		PlanningLoop: &planningLoop,
		Phases:       make([]codexWorkerPlanPhase, 0, len(phases)),
	}
	for _, phase := range phases {
		entry := codexWorkerPlanPhase{
			Name:                 phase.Name,
			Description:          phase.Description,
			Tasks:                make([]codexWorkerPlanTask, 0, len(phase.Tasks)),
			SuccessCriteria:      uniqueSortedStrings(phase.SuccessCriteria),
			EvidenceRequirements: normalizeWorkerCriterionRequirements(phase.EvidenceRequirements),
		}
		for _, task := range phase.Tasks {
			entry.Tasks = append(entry.Tasks, codexWorkerPlanTask{
				Goal:                 task.Goal,
				Constraints:          uniqueSortedStrings(task.Constraints),
				Hints:                uniqueSortedStrings(task.Hints),
				SuccessCriteria:      uniqueSortedStrings(task.SuccessCriteria),
				EvidenceRequirements: normalizeWorkerCriterionRequirements(task.EvidenceRequirements),
				DependsOn:            uniqueSortedStrings(task.DependsOn),
			})
		}
		artifact.Phases = append(artifact.Phases, entry)
	}
	return artifact
}

func normalizeWorkerCriterionRequirements(requirements []colony.CriterionEvidenceRequirement) []colony.CriterionEvidenceRequirement {
	result := make([]colony.CriterionEvidenceRequirement, 0, len(requirements))
	for _, requirement := range requirements {
		result = append(result, normalizedCriterionRequirement(requirement, ""))
	}
	return result
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
		base.Overall = planScore(float64(base.Knowledge)*0.25 +
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
			Mode:            resolveAuthoredPhaseMode("", template.Name, template.Description),
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
		Knowledge:    planScore(clampInt(report.Confidence, 55, 96)),
		Requirements: planScore(clampInt(70+len(templates)*2, 68, 94)),
		Risks:        planScore(clampInt(88-len(survey.Issues)*4-len(report.Gaps)*5, 55, 90)),
		Dependencies: planScore(clampInt(60+len(survey.EntryPoints)*5+len(survey.Dependencies)*2, 58, 92)),
		Effort:       planScore(clampInt(80-len(templates), 62, 88)),
	}
	confidence.Overall = planScore(float64(confidence.Knowledge)*0.25 +
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
	case isMDSMaxForLiveSurvey(survey):
		return []phaseTemplate{
			{
				Name:        "MDS and Max for Live surface map",
				Description: "Lock the device, builder, and script surfaces before changing patch generation behavior.",
				Tasks: []phaseTaskTemplate{
					{
						Goal:            "Map the MDS device catalog and Max for Live build entry points",
						Constraints:     commonConstraints(survey),
						Hints:           append([]string{"devices/", "m4l_builder/", "scripts/mds"}, commonHints(survey)...),
						SuccessCriteria: []string{"Relevant devices are identified", "Build scripts and generated outputs are distinguished"},
					},
					{
						Goal:            "Record the Max for Live packaging boundaries and generated-artifact policy",
						Constraints:     []string{"Do not edit generated Max artifacts before identifying their source templates"},
						Hints:           []string{"MaxForLive_Vault/", "AGENTS.md", "m4l_builder/"},
						SuccessCriteria: []string{"Generated assets and editable source files are separated", "The build plan names the correct ownership surface"},
					},
				},
				SuccessCriteria: []string{"MDS/Max for Live surfaces are explicit", "MDS-specific planning surface is explicit"},
			},
			{
				Name:        "Device implementation slice",
				Description: "Make the requested MDS or device changes against source-owned files with reproducible checks.",
				Tasks: []phaseTaskTemplate{
					{
						Goal:            "Implement the first source-owned MDS device or builder change",
						Constraints:     []string{"Keep Max patch output reproducible from source", "Preserve existing device naming and folder conventions"},
						Hints:           append([]string{"devices/", "m4l_builder/"}, survey.SourceAnchors...),
						SuccessCriteria: []string{"The requested behavior lands in source-owned files", "Generated artifacts can be rebuilt"},
					},
					{
						Goal:            "Add or update focused MDS verification for the changed device path",
						Constraints:     []string{"Use existing scripts before inventing new validation commands"},
						Hints:           append([]string{"scripts/mds"}, survey.TestFiles...),
						SuccessCriteria: []string{"The changed path has a repeatable verification command", "Failures point to the device or builder layer"},
					},
				},
				SuccessCriteria: []string{"Device changes are source-owned", "MDS verification is repeatable"},
			},
			{
				Name:        "Max for Live packaging verification",
				Description: "Rebuild or validate the Max for Live package outputs and document the release handoff.",
				Tasks: []phaseTaskTemplate{
					{
						Goal:            "Run the repository's MDS/Max for Live build or packaging command",
						Constraints:     []string{"Capture exact command output", "Do not accept stale MaxForLive_Vault contents as proof"},
						Hints:           []string{"m4l_builder/", "scripts/mds", "MaxForLive_Vault/"},
						SuccessCriteria: []string{"Build command exits cleanly or produces an actionable failure", "Expected package outputs are present or explicitly blocked"},
					},
					{
						Goal:            "Document changed devices, generated outputs, and manual Ableton/Max checks still required",
						Constraints:     []string{"Separate automated proof from manual DAW verification"},
						Hints:           []string{"README.md", "AGENTS.md"},
						SuccessCriteria: []string{"Release notes name the affected devices", "Manual verification gaps are explicit"},
					},
				},
				SuccessCriteria: []string{"Max for Live output is verified", "Manual DAW checks are not hidden"},
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

func isMDSMaxForLiveSurvey(survey codexSurveyContext) bool {
	joined := strings.ToLower(strings.Join(append(append([]string{}, survey.Frameworks...), survey.Directories...), " "))
	return containsAny(joined, []string{"max for live", "maxforlive", "m4l", "mds", "max-for-live"})
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

func writeRouteSetterArtifact(root, planningDir, goal string, granularity colony.PlanGranularity, survey codexSurveyContext, dispatch codexPlanningDispatch, confidence codexPlanConfidence, unresolvedGaps []string, phases []colony.Phase, planningLoop codexPlanningLoop, snapshots map[string]codexArtifactSnapshot) (string, bool, error) {
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
	if planningLoop.TargetConfidence > 0 {
		b.WriteString("## Planning Loop\n")
		b.WriteString(fmt.Sprintf("- Target confidence: %d%%\n", planningLoop.TargetConfidence))
		b.WriteString(fmt.Sprintf("- Max iterations: %d\n", planningLoop.MaxIterations))
		b.WriteString(fmt.Sprintf("- Iterations: %d\n", planningLoop.Iterations))
		b.WriteString(fmt.Sprintf("- Stop reason: %s\n\n", planningLoop.StopReason))
	}
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

func writeWorkerPlanArtifact(root, planningDir string, confidence codexPlanConfidence, unresolvedGaps []string, phases []colony.Phase, planningLoop codexPlanningLoop, snapshots map[string]codexArtifactSnapshot, dispatches []codexPlanningDispatch) (string, bool, error) {
	path := filepath.Join(planningDir, "phase-plan.json")
	relPath := filepath.ToSlash(filepath.Join(".aether", "data", "planning", "phase-plan.json"))
	if shouldPreserveWorkerArtifact(root, relPath, snapshots, claimedPlanningFiles(dispatches)) {
		return path, true, nil
	}

	artifact := workerPlanArtifactFromPhases(confidence, unresolvedGaps, phases, planningLoop)

	data, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return "", false, fmt.Errorf("failed to marshal worker plan artifact: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0644); err != nil {
		return "", false, fmt.Errorf("failed to write worker plan artifact: %w", err)
	}
	return path, false, nil
}

type planningArtifactPublication struct {
	ScoutFile              string
	RouteSetterFile        string
	PlanArtifactFile       string
	PhaseResearchFiles     []string
	ResearchFailedPhases   []int
	PreservedScout         bool
	PreservedRouteSetter   bool
	PreservedPlan          bool
	PreservedPhaseResearch int
	Targets                []planningSessionTarget
}

// preparePlanningArtifactPublication renders fallback-owned artifacts in an
// isolated temporary directory. Worker-authored files remain untouched and
// are captured as session baselines; locally rendered bytes become declared
// lifecycle targets that the caller can commit with state and projections.
func preparePlanningArtifactPublication(
	session *planningMutationSession,
	goal string,
	granularity colony.PlanGranularity,
	survey codexSurveyContext,
	scoutDispatch codexPlanningDispatch,
	report codexScoutReport,
	routeSetterDispatch codexPlanningDispatch,
	confidence codexPlanConfidence,
	unresolvedGaps []string,
	phases []colony.Phase,
	planningLoop codexPlanningLoop,
	snapshots map[string]codexArtifactSnapshot,
	dispatches []codexPlanningDispatch,
	preservePlanningArtifacts bool,
	preservePhaseResearch bool,
) (planningArtifactPublication, error) {
	publication := planningArtifactPublication{
		ScoutFile:        filepath.Join(session.DataRoot(), "planning", "SCOUT.md"),
		RouteSetterFile:  filepath.Join(session.DataRoot(), "planning", "ROUTE-SETTER.md"),
		PlanArtifactFile: filepath.Join(session.DataRoot(), "planning", "phase-plan.json"),
	}
	stagingRoot, err := os.MkdirTemp("", "aether-planning-artifacts-")
	if err != nil {
		return publication, fmt.Errorf("create planning artifact staging directory: %w", err)
	}
	defer os.RemoveAll(stagingRoot)
	stagingPlanning := filepath.Join(stagingRoot, "planning")
	stagingResearch := filepath.Join(stagingRoot, "phase-research")
	if err := os.MkdirAll(stagingPlanning, 0o755); err != nil {
		return publication, err
	}
	if err := os.MkdirAll(stagingResearch, 0o755); err != nil {
		return publication, err
	}
	preservationRoot := session.RepositoryRoot()
	if !preservePlanningArtifacts {
		// Finalization receives validated completion content and deliberately
		// re-renders its canonical projections. Point preservation checks at the
		// empty staging root so an older visible projection cannot win merely by
		// existing before this transaction commits.
		preservationRoot = stagingRoot
	}
	researchPreservationRoot := session.RepositoryRoot()
	if !preservePhaseResearch {
		researchPreservationRoot = stagingRoot
	}

	_, publication.PreservedScout, err = writePlanningScoutArtifact(preservationRoot, stagingPlanning, goal, granularity, survey, scoutDispatch, report, snapshots)
	if err != nil {
		return publication, err
	}
	_, publication.PreservedRouteSetter, err = writeRouteSetterArtifact(preservationRoot, stagingPlanning, goal, granularity, survey, routeSetterDispatch, confidence, unresolvedGaps, phases, planningLoop, snapshots)
	if err != nil {
		return publication, err
	}
	_, publication.PreservedPlan, err = writeWorkerPlanArtifact(preservationRoot, stagingPlanning, confidence, unresolvedGaps, phases, planningLoop, snapshots, dispatches)
	if err != nil {
		return publication, err
	}
	publication.PhaseResearchFiles, publication.PreservedPhaseResearch, publication.ResearchFailedPhases, err = writePhaseResearchArtifacts(researchPreservationRoot, stagingResearch, survey, report, phases, snapshots, dispatches)
	if err != nil {
		return publication, err
	}

	artifacts := []struct {
		name      string
		staged    string
		dataPath  string
		preserved bool
	}{
		{name: "Scout", staged: filepath.Join(stagingPlanning, "SCOUT.md"), dataPath: filepath.ToSlash(filepath.Join("planning", "SCOUT.md")), preserved: publication.PreservedScout},
		{name: "Route-Setter", staged: filepath.Join(stagingPlanning, "ROUTE-SETTER.md"), dataPath: filepath.ToSlash(filepath.Join("planning", "ROUTE-SETTER.md")), preserved: publication.PreservedRouteSetter},
		{name: "phase plan", staged: filepath.Join(stagingPlanning, "phase-plan.json"), dataPath: filepath.ToSlash(filepath.Join("planning", "phase-plan.json")), preserved: publication.PreservedPlan},
	}
	for _, artifact := range artifacts {
		if artifact.preserved {
			content, exists, readErr := session.ReadFile(lifecycleTransactionRootData, artifact.dataPath)
			if readErr != nil {
				return publication, readErr
			} else if !exists {
				return publication, fmt.Errorf("preserved %s artifact is missing", artifact.name)
			}
			publication.Targets = append(publication.Targets, planningSessionTarget{Root: lifecycleTransactionRootData, Path: artifact.dataPath, Content: content})
			continue
		}
		content, readErr := os.ReadFile(artifact.staged)
		if readErr != nil {
			return publication, fmt.Errorf("read staged %s artifact: %w", artifact.name, readErr)
		}
		publication.Targets = append(publication.Targets, planningSessionTarget{Root: lifecycleTransactionRootData, Path: artifact.dataPath, Content: content})
	}
	for _, name := range publication.PhaseResearchFiles {
		dataPath := filepath.ToSlash(filepath.Join("phase-research", name))
		content, readErr := os.ReadFile(filepath.Join(stagingResearch, name))
		if readErr == nil {
			publication.Targets = append(publication.Targets, planningSessionTarget{Root: lifecycleTransactionRootData, Path: dataPath, Content: content})
			continue
		}
		if !os.IsNotExist(readErr) {
			return publication, fmt.Errorf("read staged phase research %s: %w", name, readErr)
		}
		content, exists, baselineErr := session.ReadFile(lifecycleTransactionRootData, dataPath)
		if baselineErr != nil {
			return publication, baselineErr
		} else if !exists {
			return publication, fmt.Errorf("preserved phase research %s is missing", name)
		}
		publication.Targets = append(publication.Targets, planningSessionTarget{Root: lifecycleTransactionRootData, Path: dataPath, Content: content})
	}
	return publication, nil
}

func mergePlanningSessionTargets(groups ...[]planningSessionTarget) []planningSessionTarget {
	merged := make(map[string]planningSessionTarget)
	for _, group := range groups {
		for _, target := range group {
			key := string(target.Root) + "\x00" + filepath.ToSlash(target.Path)
			merged[key] = target
		}
	}
	result := make([]planningSessionTarget, 0, len(merged))
	for _, target := range merged {
		result = append(result, target)
	}
	return result
}

func clearFallbackPlanningArtifacts(root string) error {
	return withPlanningMutationSession(root, "planning-refresh-cleanup", func(session *planningMutationSession) error {
		return clearFallbackPlanningArtifactsInSession(session)
	})
}

func clearFallbackPlanningArtifactsInSession(session *planningMutationSession) error {
	targets, err := fallbackPlanningArtifactRemovalTargetsInSession(session)
	if err != nil {
		return err
	}
	return commitPlanningSessionTargets(session, "planning-refresh-cleanup", "planning-refresh-cleanup", targets)
}

func fallbackPlanningArtifactRemovalTargetsInSession(session *planningMutationSession) ([]planningSessionTarget, error) {
	root := session.RepositoryRoot()
	markerRelative := filepath.ToSlash(filepath.Join("planning", ".fallback-marker"))
	_, markerExists, err := session.ReadFile(lifecycleTransactionRootData, markerRelative)
	if err != nil {
		return nil, err
	}
	targets := make([]planningSessionTarget, 0, 8)
	appendFallbackRemoval := func(dataRelative, repositoryRelative string) error {
		// Capture the exact present-or-absent Plan 28 baseline before the
		// timestamp/source classifier performs its filesystem observation. The
		// final transaction must consume this same repository moment rather
		// than silently recapturing a cleanup target immediately before commit.
		if _, _, err := session.ReadFile(lifecycleTransactionRootData, dataRelative); err != nil {
			return err
		}
		if markerExists && shouldPreserveWorkerArtifact(root, repositoryRelative, nil, nil) {
			return nil
		}
		targets = append(targets, planningSessionTarget{Root: lifecycleTransactionRootData, Path: dataRelative, Remove: true})
		return nil
	}
	for _, name := range []string{"SCOUT.md", "ROUTE-SETTER.md", "phase-plan.json"} {
		if err := appendFallbackRemoval(
			filepath.ToSlash(filepath.Join("planning", name)),
			filepath.ToSlash(filepath.Join(".aether", "data", "planning", name)),
		); err != nil {
			return nil, err
		}
	}
	planningDir := filepath.Join(session.DataRoot(), "planning")
	if entries, readErr := os.ReadDir(planningDir); readErr == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".bak") {
				dataRelative := filepath.ToSlash(filepath.Join("planning", entry.Name()))
				if _, _, err := session.ReadFile(lifecycleTransactionRootData, dataRelative); err != nil {
					return nil, err
				}
				targets = append(targets, planningSessionTarget{Root: lifecycleTransactionRootData, Path: dataRelative, Remove: true})
			}
		}
	} else if !os.IsNotExist(readErr) {
		return nil, fmt.Errorf("inspect planning backup artifacts: %w", readErr)
	}
	researchDir := filepath.Join(session.DataRoot(), "phase-research")
	if entries, readErr := os.ReadDir(researchDir); readErr == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			dataRelative := filepath.ToSlash(filepath.Join("phase-research", entry.Name()))
			repositoryRelative := filepath.ToSlash(filepath.Join(".aether", "data", dataRelative))
			if err := appendFallbackRemoval(dataRelative, repositoryRelative); err != nil {
				return nil, err
			}
		}
	} else if !os.IsNotExist(readErr) {
		return nil, fmt.Errorf("inspect fallback phase research artifacts: %w", readErr)
	}
	targets = append(targets, planningSessionTarget{Root: lifecycleTransactionRootData, Path: markerRelative, Remove: true})
	return targets, nil
}

func clearPlanningBackupArtifacts(planningDir string) {
	if bakFiles, err := filepath.Glob(filepath.Join(planningDir, "*.bak")); err == nil {
		for _, f := range bakFiles {
			os.Remove(f)
		}
	}
}

func persistPlanningFallbackMarkerInSession(session *planningMutationSession, dispatchMode string, now time.Time) error {
	return commitPlanningSessionTargets(session, "planning-fallback-marker", "planning-fallback-marker", []planningSessionTarget{planningFallbackMarkerTarget(dispatchMode, now)})
}

func planningFallbackMarkerTarget(dispatchMode string, now time.Time) planningSessionTarget {
	target := planningSessionTarget{
		Root: lifecycleTransactionRootData,
		Path: filepath.ToSlash(filepath.Join("planning", ".fallback-marker")),
	}
	if dispatchMode == "fallback" {
		target.Content = []byte(now.UTC().Format(time.RFC3339))
	} else {
		target.Remove = true
	}
	return target
}

func clearPlanningBackupArtifactsInSession(session *planningMutationSession) error {
	targets, err := planningBackupRemovalTargetsInSession(session)
	if err != nil {
		return err
	}
	return commitPlanningSessionTargets(session, "planning-backup-cleanup", "planning-backup-cleanup", targets)
}

func planningBackupRemovalTargetsInSession(session *planningMutationSession) ([]planningSessionTarget, error) {
	planningDir := filepath.Join(session.DataRoot(), "planning")
	entries, err := os.ReadDir(planningDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("inspect planning backup artifacts: %w", err)
	}
	targets := make([]planningSessionTarget, 0)
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".bak") {
			targets = append(targets, planningSessionTarget{
				Root:   lifecycleTransactionRootData,
				Path:   filepath.ToSlash(filepath.Join("planning", entry.Name())),
				Remove: true,
			})
		}
	}
	return targets, nil
}

// prunePhaseResearchOrphans removes phase-N-research.md files that no longer
// correspond to a current phase. Files for live phases are never touched.
func prunePhaseResearchOrphans(dir string, phases []colony.Phase) {
	live := make(map[string]bool, len(phases))
	for _, phase := range phases {
		live[fmt.Sprintf("phase-%d-research.md", phase.ID)] = true
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasPrefix(name, "phase-") || !strings.HasSuffix(name, "-research.md") {
			continue
		}
		if !live[name] {
			_ = os.Remove(filepath.Join(dir, name))
		}
	}
}

func prunePhaseResearchOrphansInSession(session *planningMutationSession, phases []colony.Phase) error {
	targets, err := phaseResearchOrphanRemovalTargetsInSession(session, phases)
	if err != nil {
		return err
	}
	return commitPlanningSessionTargets(session, "planning-research-prune", "planning-research-prune", targets)
}

func phaseResearchOrphanRemovalTargetsInSession(session *planningMutationSession, phases []colony.Phase) ([]planningSessionTarget, error) {
	live := make(map[string]bool, len(phases))
	for _, phase := range phases {
		live[fmt.Sprintf("phase-%d-research.md", phase.ID)] = true
	}
	dir := filepath.Join(session.DataRoot(), "phase-research")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("inspect phase research artifacts: %w", err)
	}
	targets := make([]planningSessionTarget, 0)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasPrefix(name, "phase-") || !strings.HasSuffix(name, "-research.md") || live[name] {
			continue
		}
		targets = append(targets, planningSessionTarget{
			Root:   lifecycleTransactionRootData,
			Path:   filepath.ToSlash(filepath.Join("phase-research", name)),
			Remove: true,
		})
	}
	return targets, nil
}

func writePhaseResearchArtifacts(root, dir string, survey codexSurveyContext, report codexScoutReport, phases []colony.Phase, snapshots map[string]codexArtifactSnapshot, dispatches []codexPlanningDispatch) ([]string, int, []int, error) {
	written := make([]string, 0, len(phases))
	claimed := claimedPlanningFiles(dispatches)
	preserved := 0
	failed := []int{}
	// A phase counts as "approved and dispatched" when a phase_research Scout
	// dispatch exists for it -- the same stage+caste filter
	// validatePlanningWorkerChain (cmd/codex_plan_finalize.go:641) already uses
	// to identify phase_research dispatches. Only these phases can be
	// classified as a failed research worker: a phase never dispatched for
	// research (the user skipped it) falling through to the fallback template
	// is expected behaviour, not a failure.
	dispatchedResearchFiles := make(map[string]bool, len(dispatches))
	for _, d := range dispatches {
		if d.Stage != phaseResearchStage || !strings.EqualFold(d.Caste, "scout") {
			continue
		}
		for _, out := range d.Outputs {
			dispatchedResearchFiles[out] = true
		}
	}
	for _, phase := range phases {
		name := fmt.Sprintf("phase-%d-research.md", phase.ID)
		path := filepath.Join(dir, name)
		relPath := filepath.ToSlash(filepath.Join(".aether", "data", "phase-research", name))
		if shouldPreserveWorkerArtifact(root, relPath, snapshots, claimed) {
			written = append(written, name)
			preserved++
			continue
		}
		// Fallback template in the five-section RESEARCH.md format (the v5
		// Phase Domain Research contract). Real Scout research written by a
		// phase_research worker replaces this and is preserved above.
		var b strings.Builder
		b.WriteString(fmt.Sprintf("# Phase %d Research: %s\n\n", phase.ID, phase.Name))
		b.WriteString(fmt.Sprintf("**Generated:** %s\n", time.Now().UTC().Format(time.RFC3339)))
		b.WriteString(fmt.Sprintf("**Phase:** %d - %s\n", phase.ID, phase.Name))
		b.WriteString("**Research scope:** synthesized from territory survey and scout findings (no dedicated research worker ran for this phase)\n")
		if dispatchedResearchFiles[name] {
			// D-08: a phase that was approved and sent to a research worker,
			// but still fell through to the fallback template, means that
			// worker produced nothing. Name the failure loudly in the
			// artifact itself, not just in the finalize result.
			b.WriteString(fmt.Sprintf("**Research status:** phase %d planned WITHOUT its research — worker failed\n", phase.ID))
			failed = append(failed, phase.ID)
		}
		// No hive-wisdom stand-in here (WIRE-02): no research worker ran for
		// this phase, so there is no shared-lessons content to summarise --
		// the fallback template never invents one (D-16, no empty sections).
		b.WriteString("\n")
		b.WriteString("## Key Patterns\n")
		patterns := []string{}
		for _, finding := range report.Findings {
			patterns = append(patterns, fmt.Sprintf("**%s:** %s (Source: %s)", finding.Area, finding.Discovery, firstNonEmpty(finding.Source, "scout survey")))
			if len(patterns) == 3 {
				break
			}
		}
		b.WriteString(bulletList(patterns, "No extra repo patterns were synthesized for this phase."))
		b.WriteString("\n\n## External Context\n")
		b.WriteString("No external research needed for this phase\n")
		b.WriteString("\n## Gotchas\n")
		risks := append([]string{}, report.Gaps...)
		risks = append(risks, survey.Issues...)
		b.WriteString(bulletList(limitStrings(uniqueSortedStrings(risks), 4), "No additional risks captured for this phase."))
		b.WriteString("\n\n## Recommended Approach\n")
		b.WriteString(strings.TrimSpace(firstNonEmpty(phase.Description, "Follow the phase tasks in order and verify against the phase success criteria.")))
		b.WriteString("\n\n## Files to Study\n")
		files := append([]string{}, report.StudyFiles...)
		for _, task := range phase.Tasks {
			files = append(files, task.Hints...)
		}
		b.WriteString(bulletList(limitStrings(uniqueSortedStrings(files), 6), "No specific file anchors were identified."))
		b.WriteString("\n")
		if err := os.WriteFile(path, []byte(b.String()), 0644); err != nil {
			return nil, 0, nil, fmt.Errorf("failed to write %s: %w", name, err)
		}
		written = append(written, name)
	}
	sort.Strings(written)
	sort.Ints(failed)
	return written, preserved, failed, nil
}

// renderResearchFailedWarning is D-08's loud, non-blocking warning: research
// is enrichment (Phase 160's classification), so a failed research worker
// never errors or gates finalize -- it names the phases in the finalize
// result so the omission is durable and visible, not silently absorbed by
// the fallback template. Returns "" when no phase failed.
func renderResearchFailedWarning(failed []int) string {
	if len(failed) == 0 {
		return ""
	}
	return fmt.Sprintf("phase(s) %s planned WITHOUT its research — worker failed", joinInts(failed))
}
