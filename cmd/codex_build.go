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
	Stage         string `json:"stage"`
	Wave          int    `json:"wave,omitempty"`
	ExecutionWave int    `json:"execution_wave,omitempty"`
	Caste         string `json:"caste"`
	AgentName     string `json:"agent_name,omitempty"`
	// Model is the DISPLAY name of the model this caste's agent runs on
	// (resolved from agent frontmatter slot + ANTHROPIC_DEFAULT_*_MODEL env),
	// so wrapper-rendered spawn descriptions can show it. Nothing reads it
	// to choose a model — routing stays with the platform's agent
	// frontmatter, and automatic model selection stays rejected.
	Model   string `json:"model,omitempty"`
	Name    string `json:"name"`
	Task    string `json:"task"`
	Status  string `json:"status"`
	Summary string `json:"summary,omitempty"`
	// Disposition qualifies a completed_no_change status: "verified_existing"
	// means the worker proved the required behavior already exists (ruling
	// D6). Empty for every other status.
	Disposition string `json:"disposition,omitempty"`
	TaskID      string `json:"task_id,omitempty"`
	TaskIndex   int    `json:"task_index,omitempty"`
	// Job metadata is the durable explanation for why this worker owns one or
	// more tasks. TaskID remains the primary compatibility key, while
	// CoveredTaskIDs preserves the full ordered task-credit set.
	JobName   string `json:"job_name,omitempty"`
	JobReason string `json:"job_reason,omitempty"`
	JobSource string `json:"job_source,omitempty"`
	// CoveredTaskIDs lists every task this one worker took on. It holds more
	// than one entry when a chain of dependent steps was merged into a single
	// dispatch (see coalesceSequentialDispatches). TaskID stays the first of
	// them so result matching and evidence keep working unchanged; this field
	// exists so nothing downstream can believe the later steps were unassigned.
	CoveredTaskIDs []string `json:"covered_task_ids,omitempty"`
	DependsOn      []string `json:"depends_on,omitempty"`
	DeclaredPaths  []string `json:"declared_paths,omitempty"`
	Outputs        []string `json:"outputs,omitempty"`
	Blockers       []string `json:"blockers,omitempty"`
	Duration       float64  `json:"duration,omitempty"`
	// Brief is the fully rendered worker prompt for wrapper-spawned workers.
	// Build was the only workflow whose plan-only manifest carried no brief —
	// colonize, plan, and heavy-continue all do — so everything the runtime
	// assembles (phase objective, constraints, hints, criteria, pheromone
	// signals, survey, handoffs) never reached the workers the user actually
	// watches spawn. Wrappers must inject this verbatim, never reconstruct it.
	Brief string `json:"brief,omitempty"`
	// BriefPath is the repo-display path to the file holding the verbatim
	// composed brief (the same bytes as Brief above); the wrapper may read
	// this instead of the inline Brief field.
	BriefPath         string                  `json:"brief_path,omitempty"`
	SkillSection      string                  `json:"skill_section,omitempty"`
	SkillCount        int                     `json:"skill_count,omitempty"`
	ColonySkills      int                     `json:"colony_skill_count,omitempty"`
	DomainSkills      int                     `json:"domain_skill_count,omitempty"`
	MatchedSkills     []string                `json:"matched_skills,omitempty"`
	HandoffSection    string                  `json:"handoff_section,omitempty"`
	PermissionProfile codex.PermissionProfile `json:"permission_profile"`
}

type codexBuildTaskPlan struct {
	ID        string   `json:"id,omitempty"`
	Goal      string   `json:"goal"`
	Status    string   `json:"status"`
	Wave      int      `json:"wave,omitempty"`
	DependsOn []string `json:"depends_on,omitempty"`
}

type codexBuildManifest struct {
	Phase               int                       `json:"phase"`
	PhaseName           string                    `json:"phase_name"`
	PhaseMode           colony.PhaseMode          `json:"phase_mode,omitempty"`
	Goal                string                    `json:"goal,omitempty"`
	Root                string                    `json:"root"`
	ColonyMode          string                    `json:"colony_mode,omitempty"`
	PlanOnly            bool                      `json:"plan_only,omitempty"`
	ParallelMode        string                    `json:"parallel_mode,omitempty"`
	WaveExecution       []codexWaveExecutionPlan  `json:"wave_execution,omitempty"`
	ExecutionPlan       []codexBuildExecutionPlan `json:"execution_plan,omitempty"`
	ColonyDepth         string                    `json:"colony_depth"`
	DispatchMode        string                    `json:"dispatch_mode,omitempty"`
	HostPlatform        string                    `json:"host_platform,omitempty"`
	ExecutionOwner      string                    `json:"execution_owner,omitempty"`
	WorkerDispatchOptIn bool                      `json:"worker_dispatch_opt_in,omitempty"`
	GeneratedAt         string                    `json:"generated_at"`
	PlanRevisionID      string                    `json:"plan_revision_id,omitempty"`
	PlanStateHash       string                    `json:"plan_state_hash,omitempty"`
	AttemptID           string                    `json:"attempt_id,omitempty"`
	AttemptPath         string                    `json:"attempt_path,omitempty"`
	ExecutionBinding    *codex.ExecutionBinding   `json:"execution_binding,omitempty"`
	State               string                    `json:"state"`
	Checkpoint          string                    `json:"checkpoint"`
	ClaimsPath          string                    `json:"claims_path"`
	WorkerBriefs        []string                  `json:"worker_briefs"`
	// ContextCapsule is the colony-prime grounding payload (state, decisions,
	// phase learnings, instincts, hive wisdom, prior reviews, blockers, user
	// preferences) for wrapper-spawned build workers. It is computed once per
	// plan-only manifest, carried here at the top level rather than copied into
	// every dispatch brief, and the wrapper reads it once and prepends it,
	// verbatim, to each spawned worker's prompt. Only populated when planOnly —
	// the hosted/subprocess path already computes and shares its own capsule
	// (see executeCodexBuildDispatches), and the finalize record does not
	// deliver prompts.
	ContextCapsule            string                                `json:"context_capsule,omitempty"`
	Dispatches                []codexBuildDispatch                  `json:"dispatches"`
	JobDecisions              []coherentJobDecision                 `json:"job_decisions,omitempty"`
	SelectedTasks             []string                              `json:"selected_tasks,omitempty"`
	Tasks                     []codexBuildTaskPlan                  `json:"tasks"`
	SuccessCriteria           []string                              `json:"success_criteria"`
	CriterionEvidencePolicy   string                                `json:"criterion_evidence_policy,omitempty"`
	EvidenceRequirements      []colony.CriterionEvidenceRequirement `json:"evidence_requirements,omitempty"`
	ReviewDepth               string                                `json:"review_depth,omitempty"`
	DispatchContract          map[string]interface{}                `json:"dispatch_contract,omitempty"`
	ProviderDiagnostics       string                                `json:"provider_diagnostics,omitempty"`
	ProfileContract           codexWorkflowProfileContract          `json:"profile_contract,omitempty"`
	QueenRecommendation       codexQueenWorkflowRecommendation      `json:"queen_recommendation,omitempty"`
	QueenExecutionPolicy      codexQueenExecutionPolicy             `json:"queen_execution_policy,omitempty"`
	BoundaryQuestions         []discussQuestion                     `json:"boundary_questions,omitempty"`
	BoundaryQuestionCount     int                                   `json:"boundary_question_count,omitempty"`
	BoundaryQuestionsCreated  int                                   `json:"boundary_questions_created,omitempty"`
	BoundaryQuestionsExisting int                                   `json:"boundary_questions_existing,omitempty"`
	OrchestratorGuidance      *orchestratorBoundaryGuidance         `json:"orchestrator_boundary_guidance,omitempty"`
	// CasteRoster lists every dispatchable caste and what it is good at, so a
	// Queen deciding the team is choosing from the live registry rather than
	// from memory. Without it the wrapper guesses at both the names and the
	// roles, and an unrecognised name silently costs the phase a specialist.
	CasteRoster []map[string]string `json:"caste_roster,omitempty"`
	// CasteDecision records how the team was chosen: what the Queen proposed,
	// what actually spawns, and every override the runtime applied. A Queen
	// that proposes badly must produce a visible correction, not a silent one.
	CasteDecision map[string]interface{} `json:"caste_decision,omitempty"`
	// ForcedReviewers is the D-01..D-05 named-risk-signal derivation, computed
	// once here at build and read back by both continue lanes
	// (queenForcedContinueReviewers) so a reviewer is forced from exactly one
	// derivation at exactly one boundary — closing .planning/WINDOWS.md #1's
	// "build and continue each require the same caste independently" gap.
	// The build ANNOUNCES this record; it never dispatches the reviewer
	// itself (D-05) — that happens at the checking step (continue).
	ForcedReviewers []codexForcedReviewerRecord `json:"forced_reviewers,omitempty"`
	// ForcedReviewerAnnouncement is the owner-facing sentence(s) composed from
	// ForcedReviewers (one line per forced caste) — "a security reviewer will
	// check this at the verification step — this touches logins (the plan
	// mentions "password reset")". The build never dispatches a forced
	// reviewer (D-05); this field is how the build ANNOUNCES one before any
	// worker spawns. Empty when no reviewer is forced.
	ForcedReviewerAnnouncement string `json:"forced_reviewer_announcement,omitempty"`
}

// codexForcedReviewerRecord is the durable, JSON form of a forcedReviewer
// (cmd/queen_risk_signals.go), written onto the build manifest so continue
// reads the exact same derivation instead of re-deriving it independently.
type codexForcedReviewerRecord struct {
	Caste   string   `json:"caste"`
	Signals []string `json:"signals"`
	Matches []string `json:"matches"`
	Sources []string `json:"sources"`
	Reason  string   `json:"reason"`
}

// composeForcedReviewerAnnouncement turns the build's recorded forced-reviewer
// set into the owner-facing sentences the check-in card and manifest both
// render (D-05): one sentence per forced caste, in the exact shape the ruling
// gives — "a security reviewer will check this at the verification step —
// this touches logins (the plan mentions "password reset")". Uses the plain
// human name (forcedReviewerPlainLabel), never the registry identifier
// (CLAUDE.md's plain-English mandate). Empty input renders nothing — no
// announcement, no empty heading.
func composeForcedReviewerAnnouncement(records []codexForcedReviewerRecord) string {
	if len(records) == 0 {
		return ""
	}
	sentences := make([]string, 0, len(records))
	for _, record := range records {
		reason := strings.TrimSpace(record.Reason)
		if reason == "" {
			continue
		}
		sentences = append(sentences, fmt.Sprintf(
			"a %s will check this at the verification step — %s",
			forcedReviewerPlainLabel(record.Caste), reason,
		))
	}
	return strings.Join(sentences, "\n")
}

// forcedReviewerPlainLabel names a forced-reviewer caste the way the owner
// reads it, never the registry identifier — "security reviewer", not
// "gatekeeper"; "quality reviewer", not "auditor" (D-04's two reviewer
// castes are the only ones this table can force).
func forcedReviewerPlainLabel(caste string) string {
	switch strings.TrimSpace(caste) {
	case "gatekeeper":
		return "security reviewer"
	case "auditor":
		return "quality reviewer"
	default:
		return strings.ToLower(casteLabel(caste)) + " reviewer"
	}
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
	FilesCreated     []string                     `json:"files_created"`
	FilesModified    []string                     `json:"files_modified"`
	TestsWritten     []string                     `json:"tests_written,omitempty"`
	TaskClaims       []codexBuildTaskClaim        `json:"task_claims,omitempty"`
	ArtifactEvidence []codexBuildArtifactEvidence `json:"artifact_evidence,omitempty"`
	BuildPhase       int                          `json:"build_phase"`
	Timestamp        string                       `json:"timestamp"`
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
	// Full gates the raw-prompt path of --print-brief. It has no effect on any
	// mutating build path — only printWorkerBriefs reads it.
	Full bool
	// QueenCastes is the team the Queen proposed after reading the phase.
	// Empty means no judgement was offered and the deterministic keyword
	// engine decides — the behaviour of every caller before this existed.
	QueenCastes []string
	// QueenCasteReason is the Queen's stated reasoning, surfaced to the
	// operator so a team choice is never unexplained. This is the TEAM
	// summary (D-08) -- it does not, by itself, satisfy the per-worker reason
	// requirement below.
	QueenCasteReason string
	// QueenCasteWhy is one reason per proposed caste, as "caste=reason"
	// (D-08, D-09). A caste named in QueenCastes with no matching entry here,
	// and not required by the phase, is refused by name rather than sent
	// unexplained (parseAndMergeCasteWhy, queenApplyJudgement).
	QueenCasteWhy []string
	// JobProposals are structured Queen suggestions. The coherent-job planner
	// validates them before any attempt, checkpoint, worktree, or lifecycle
	// mutation is allowed to begin.
	JobProposals []coherentJobProposal
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
	if err := validatePhaseCriterionEvidence(phase); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}
	if err := validateSelectedBuildTasks(phase, selectedTaskIDs); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}
	if err := runPreBuildGates(store.BasePath(), phaseNum); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}
	_, priorAttempt, hasPriorAttempt := loadLatestBuildAttempt(phaseNum)
	activePriorAttempt := hasPriorAttempt && buildAttemptStatusActive(priorAttempt.Status)
	forceStateValidation := options.Force
	if activePriorAttempt && state.State != colony.StateEXECUTING && state.State != colony.StateBUILT {
		forceStateValidation = false
	}
	if err := validateCodexBuildState(state, phaseNum, selectedTaskIDs, forceStateValidation); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}
	policy := recommendQueenExecutionPolicy(state, phase, len(state.Plan.Phases), codexQueenExecutionPolicyInput{
		LightFlag:         options.LightFlag,
		HeavyFlag:         options.HeavyFlag,
		VerificationDepth: options.VerificationDepth,
		WorkerTimeout:     options.WorkerTimeout,
		DispatchWorkers:   options.DispatchWorkers,
	})
	reviewDepth := colony.NormalizeVerificationDepth(policy.VerificationDepth)
	// Decode happens in the CLI before this function is entered. The pure
	// planner runs here, before an idle attempt can be superseded, so invalid
	// proposals and dependency cycles have a strict zero-side-effect boundary.
	mergedQueenCastes, queenCasteWhyReasons := parseAndMergeCasteWhy(options.QueenCastes, options.QueenCasteWhy)
	dispatches, jobDecisions, err := plannedBuildDispatchesWithJobProposals(
		phase, state, selectedTaskIDs, reviewDepth, mergedQueenCastes, options.QueenCasteReason, queenCasteWhyReasons, options.JobProposals,
	)
	if err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}
	if activePriorAttempt {
		// Once any worker has been dispatched, a public plan-only re-entry must
		// not supersede the attempt or reopen its owner-only reviewer-decline
		// channel. A legitimate retry first reaches a terminal attempt state;
		// only then may the next check-in create a fresh capability.
		if _, dispatchStarted := phaseDispatchStartedAt(phaseNum); dispatchStarted {
			return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("phase %d already has workers in flight for build attempt %s; finalize or fail that attempt before requesting a fresh plan", phaseNum, priorAttempt.ID)
		}
		// A dangling plan-only attempt still awaiting external workers holds
		// no worker output, and the wrapper needs a fresh manifest anyway. Let
		// re-entry supersede it automatically instead of demanding --force;
		// blocking here made an aborted /ant-build jam every following one.
		priorIsIdlePlanOnly := priorAttempt.Status == buildAttemptAwaiting && strings.TrimSpace(priorAttempt.DispatchMode) == "plan-only"
		if !options.Force && !priorIsIdlePlanOnly {
			return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("phase %d already has active build attempt %s (%s); finalize its completion packet or rerun with --force to supersede it", phaseNum, priorAttempt.ID, priorAttempt.Status)
		}
		reason := "superseded by an explicit plan-only force redispatch"
		if !options.Force {
			reason = "superseded by a fresh plan-only manifest request before any worker was recorded"
		}
		if err := interruptLatestBuildAttempt(phaseNum, reason); err != nil {
			return nil, colony.ColonyState{}, colony.Phase{}, nil, err
		}
	}

	generatedAt := time.Now().UTC()
	// judgementReasonsForBudget mirrors queenCasteDecisionSummary's own
	// judgement derivation (queenApplyJudgement is pure/deterministic, so
	// recomputing here costs nothing) so the spawn-budget contract's
	// selected_reasons can prefer the SAME per-worker sentence the manifest's
	// caste_decision shows, rather than a separately-derived one.
	judgementReasonsForBudget := func() map[string]string {
		judgementState := state
		judgementState.VerificationDepth = string(reviewDepth)
		return queenApplyJudgement(mergedQueenCastes, options.QueenCasteReason, phase, "build", judgementState, queenCasteWhyReasons).Reasons
	}()
	for i := range dispatches {
		dispatches[i].Status = "planned"
	}
	dispatches, err = ensureUniqueBuildDispatchNames(dispatches, phaseNum)
	if err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}
	attachBuildDispatchContext(root, phase, dispatches, generatedAt)
	buildDirRel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phaseNum)))
	// Write every dispatch's composed brief to disk and blank the inline copy
	// once the write succeeds (clearInlineBrief=true) -- the ONLY dispatch
	// path the interactive wrapper is allowed to call must never ship the
	// same composed brief twice in one JSON response. A dispatch whose write
	// fails keeps its inline Brief populated (see writeBuildWorkerBriefFiles).
	// Clear the previous manifest's brief files first (190-190/WR-01) --
	// this path writes {dispatch.Name}.md into a directory nothing else ever
	// prunes, and dispatch names change whenever task wording, task selection
	// or caste coalescing does.
	cleanupStaleWorkerBriefs(phaseNum)
	briefPaths, dispatches, err := writeBuildWorkerBriefFiles(root, phase, buildDirRel, dispatches, generatedAt, true)
	if err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}
	policy = enrichQueenExecutionPolicyWithSpawnBudget(policy, state, phase, "build", reviewDepth, dispatches, judgementReasonsForBudget)

	parallelMode := effectiveParallelMode(state)
	waveExecution := buildWaveExecutionPlans(dispatches, parallelMode)
	executionPlan := buildExecutionPlans(dispatches, parallelMode)
	dispatchContract := buildDispatchContractForDispatches(dispatches, parallelMode, options.WorkerTimeout)
	providerDiagnostics := dispatchProviderDiagnostics(newCodexWorkerInvoker())
	checkpointRel := filepath.ToSlash(filepath.Join("checkpoints", fmt.Sprintf("pre-build-phase-%d.json", phaseNum)))
	manifestRel := filepath.ToSlash(filepath.Join(buildDirRel, "manifest.json"))
	claimsRel := "last-build-claims.json"
	manifest := buildCodexBuildManifest(root, state, phase, "", "", dispatches, generatedAt, "plan-only", selectedTaskIDs, briefPaths, true, reviewDepth)
	manifest.Phase = phaseNum
	manifest.JobDecisions = append([]coherentJobDecision{}, jobDecisions...)
	manifest.DispatchContract = dispatchContract
	manifest.ProviderDiagnostics = providerDiagnostics
	profileContract := workflowProfileContract(reviewDepth)
	queenRecommendation := recommendQueenWorkflowProfile(state, phase, len(state.Plan.Phases))
	manifest.QueenExecutionPolicy = policy
	// The roster is what lets the wrapper's Queen choose from the live registry
	// instead of from memory; the decision is the audit trail for what it chose
	// and what the runtime overrode.
	manifest.CasteRoster = queenCasteRoster()
	manifest.CasteDecision = queenCasteDecisionSummary(phase, state, reviewDepth, mergedQueenCastes, options.QueenCasteReason, queenCasteWhyReasons)
	// The forced-reviewer set is derived from the phase's own wording exactly
	// once, here, and recorded — never dispatched at build (D-05). Continue
	// reads this record via queenForcedContinueReviewers.
	manifest.ForcedReviewers = forcedReviewerRecords(queenForcedReviewersForPhase(phase))
	manifest.ForcedReviewerAnnouncement = composeForcedReviewerAnnouncement(manifest.ForcedReviewers)
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
		"next":                     "spawn wrapper agents from dispatches, then record completion",
		"currentTask":              phase.Tasks,
		"dispatches":               codexBuildDispatchMaps(dispatches),
		"dispatch_manifest":        manifest,
		"job_decisions":            append([]coherentJobDecision{}, jobDecisions...),
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
		"provider_diagnostics":     providerDiagnostics,
		"host_platform":            string(codex.DetectActivePlatform()),
		"execution_owner":          buildExecutionOwner("plan-only", true),
		"profile_contract":         profileContract,
		"queen_recommendation":     queenRecommendation,
		"queen_execution_policy":   policy,
		"selected_tasks":           selectedTaskIDs,
		"wrapper_contract": map[string]interface{}{
			"source_command":          "aether build <phase> --plan-only",
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
	if manifest.OrchestratorGuidance == nil || !manifest.OrchestratorGuidance.Active {
		// CR-01 residual (194-REVIEW.md iteration 3/4): reopen phaseNum's
		// forced-reviewer decline window for this NEW attempt, before the
		// wrapper renders the check-in card from the manifest this call
		// produces. See clearPhaseDispatchWindow's doc comment
		// (cmd/forced_reviewer_waiver.go) for why this call lives here and
		// nowhere else.
		clearPhaseDispatchWindow(phaseNum)
		attemptRel, err := beginBuildAttempt(state, phaseNum, phase, generatedAt, selectedTaskIDs, checkpointRel, manifestRel, claimsRel, manifest.ExecutionOwner, dispatches)
		if err != nil {
			return nil, colony.ColonyState{}, colony.Phase{}, nil, err
		}
		_, attempt, ok := loadLatestBuildAttempt(phaseNum)
		if !ok || attemptRel == "" {
			return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("failed to reload prepared build attempt for phase %d", phaseNum)
		}
		manifest.AttemptID = attempt.ID
		manifest.AttemptPath = displayDataPath(attemptRel)
		if err := prepareBuildAttemptManifestBinding(attemptRel, &manifest); err != nil {
			_ = transitionBuildAttempt(attemptRel, buildAttemptFailed, "failed to prepare plan-only execution binding", nil, nil, "plan-only", err)
			return nil, colony.ColonyState{}, colony.Phase{}, nil, err
		}
		if err := store.SaveJSON(manifestRel, manifest); err != nil {
			_ = transitionBuildAttempt(attemptRel, buildAttemptFailed, "failed to persist plan-only manifest", nil, nil, "plan-only", err)
			return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("failed to persist plan-only build manifest: %w", err)
		}
		if err := bindBuildAttemptManifest(attemptRel, manifest); err != nil {
			_ = transitionBuildAttempt(attemptRel, buildAttemptFailed, "failed to bind plan-only manifest", nil, nil, "plan-only", err)
			return nil, colony.ColonyState{}, colony.Phase{}, nil, err
		}
		result["dispatch_manifest"] = manifest
		result["attempt"] = displayDataPath(attemptRel)
	}
	return result, state, phase, dispatches, nil
}

func runCodexBuildQueenLed(root string, phaseNum int, selectedTaskIDs []string, options codexBuildOptions) (map[string]interface{}, colony.ColonyState, colony.Phase, []codexBuildDispatch, error) {
	result, state, phase, dispatches, err := runCodexBuildPlanOnlyWithOptions(root, phaseNum, selectedTaskIDs, options)
	if err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}
	if guidance, ok := result["orchestrator_boundary_guidance"].(orchestratorBoundaryGuidance); ok && guidance.Active {
		return result, state, phase, dispatches, nil
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
	policy = enrichQueenExecutionPolicyWithSpawnBudget(policy, state, phase, "build", reviewDepth, dispatches)

	if manifest, ok := result["dispatch_manifest"].(codexBuildManifest); ok {
		manifest.DispatchMode = "queen-led"
		manifest.ExecutionOwner = buildExecutionOwner("queen-led", true)
		manifest.ProfileContract = profileContract
		manifest.QueenRecommendation = queenRecommendation
		manifest.QueenExecutionPolicy = policy
		if strings.TrimSpace(manifest.AttemptPath) != "" {
			attemptRel := strings.TrimPrefix(filepath.ToSlash(manifest.AttemptPath), ".aether/data/")
			manifestRel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phaseNum), "manifest.json"))
			if err := prepareBuildAttemptManifestBinding(attemptRel, &manifest); err != nil {
				return nil, colony.ColonyState{}, colony.Phase{}, nil, err
			}
			if err := store.SaveJSON(manifestRel, manifest); err != nil {
				return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("failed to persist queen-led build manifest: %w", err)
			}
			if err := bindBuildAttemptManifest(attemptRel, manifest); err != nil {
				return nil, colony.ColonyState{}, colony.Phase{}, nil, err
			}
		}
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
		"source_command":          "aether build <phase> --plan-only",
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
	if err := validatePhaseCriterionEvidence(phase); err != nil {
		return nil, err
	}
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
	policy := recommendQueenExecutionPolicy(state, phase, len(state.Plan.Phases), codexQueenExecutionPolicyInput{
		LightFlag:         options.LightFlag,
		HeavyFlag:         options.HeavyFlag,
		VerificationDepth: options.VerificationDepth,
		WorkerTimeout:     options.WorkerTimeout,
		DispatchWorkers:   true,
	})
	reviewDepth := colony.NormalizeVerificationDepth(policy.VerificationDepth)
	mergedQueenCastes, queenCasteWhyReasons := parseAndMergeCasteWhy(options.QueenCastes, options.QueenCasteWhy)
	dispatches, jobDecisions, err := plannedBuildDispatchesWithJobProposals(
		phase, state, selectedTaskIDs, reviewDepth, mergedQueenCastes, options.QueenCasteReason, queenCasteWhyReasons, options.JobProposals,
	)
	if err != nil {
		return nil, err
	}
	casteDecision := queenCasteDecisionSummary(phase, state, reviewDepth, mergedQueenCastes, options.QueenCasteReason, queenCasteWhyReasons)

	// Detect orphaned worktrees from prior interrupted builds
	orphans := detectOrphanedWorktrees(phaseNum)
	if len(orphans) > 0 && !options.Force {
		var orphanBranches []string
		for _, o := range orphans {
			orphanBranches = append(orphanBranches, fmt.Sprintf("%s (phase %d)", o.Branch, o.Phase))
		}
		return nil, fmt.Errorf("orphaned worktree branches detected: %s. Run with --force to proceed anyway, or run `aether worktree-merge-back` to recover", strings.Join(orphanBranches, ", "))
	}

	originalState, err := cloneColonyState(state)
	if err != nil {
		return nil, fmt.Errorf("failed to clone colony state: %w", err)
	}
	if options.Force {
		if err := interruptLatestBuildAttempt(phaseNum, "superseded by an explicit force redispatch"); err != nil {
			return nil, err
		}
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

	dispatches, err = ensureUniqueBuildDispatchNames(dispatches, phaseNum)
	if err != nil {
		return nil, err
	}
	// Check the whole plan against the remaining helper budget before the first
	// worker starts. The per-spawn ceiling is checked one worker at a time, so a
	// ten-worker build with three slots left used to begin and stall in the
	// middle -- paying for every worker that had already run. A budget that
	// cannot be read is not treated as free.
	if budgetState, budgetErr := spawnTreeBudgetState(); budgetErr == nil {
		if fits, reason := spawnBudgetPreflight(len(dispatches), budgetState); !fits {
			return nil, fmt.Errorf("%s", reason)
		}
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
	plannedDispatchMode := "real"
	if synthetic {
		plannedDispatchMode = "simulated"
	}

	cleanupStaleBuildAttemptArtifacts(phaseNum)

	if err := store.SaveJSON(checkpointRel, state); err != nil {
		return nil, fmt.Errorf("failed to checkpoint colony state: %w", err)
	}
	attemptRel, err := beginBuildAttempt(state, phaseNum, phase, startedAt, selectedTaskIDs, checkpointRel, manifestRel, claimsRel, buildExecutionOwner(plannedDispatchMode, false), dispatches)
	if err != nil {
		return nil, err
	}
	attemptFinished := false
	finishAttempt := func(status, summary string, transitionErr error) {
		if attemptFinished {
			return
		}
		if err := transitionBuildAttempt(attemptRel, status, summary, nil, nil, "", transitionErr); err == nil && buildAttemptStatusTerminal(status) {
			attemptFinished = true
		}
	}
	defer func() {
		if !attemptFinished {
			_ = transitionBuildAttempt(attemptRel, buildAttemptInterrupted, "build command ended before durable finalization", nil, nil, "", fmt.Errorf("build command ended before durable finalization"))
		}
	}()

	updatedState := state
	applyCodexBuildState(&updatedState, phaseNum, startedAt, selectedTaskIDs, reviewDepth)
	updatedPhase := updatedState.Plan.Phases[phaseNum-1]
	if err := store.SaveJSON("COLONY_STATE.json", updatedState); err != nil {
		finishAttempt(buildAttemptFailed, "failed to project executing lifecycle state", err)
		return nil, fmt.Errorf("failed to save colony state: %w", err)
	}
	if progress != nil {
		progress.Advance("Prepare")
	}

	briefPaths, dispatches, err := writeCodexBuildArtifacts(root, updatedState, updatedPhase, buildDirRel, checkpointRel, claimsRel, dispatches, startedAt, plannedDispatchMode, selectedTaskIDs, reviewDepth, policy, jobDecisions)
	if err != nil {
		finishAttempt(buildAttemptFailed, "failed to prepare worker artifacts", err)
		rollbackCodexBuildFailure(originalState, phaseNum, startedAt, err)
		return nil, err
	}
	if err := recordCodexBuildDispatches(dispatches); err != nil {
		finishAttempt(buildAttemptFailed, "failed to record planned dispatches", err)
		rollbackCodexBuildFailure(originalState, phaseNum, startedAt, err)
		return nil, err
	}
	var dispatchManifest codexBuildManifest
	if err := store.LoadJSON(manifestRel, &dispatchManifest); err != nil {
		finishAttempt(buildAttemptFailed, "failed to reload direct build manifest", err)
		rollbackCodexBuildFailure(originalState, phaseNum, startedAt, err)
		return nil, fmt.Errorf("failed to reload direct build manifest: %w", err)
	}
	dispatchManifest.CasteDecision = casteDecision
	dispatchManifest.JobDecisions = append([]coherentJobDecision{}, jobDecisions...)
	dispatchManifest.AttemptID = strings.TrimSpace(strings.TrimSuffix(filepath.Base(attemptRel), filepath.Ext(attemptRel)))
	dispatchManifest.AttemptPath = displayDataPath(attemptRel)
	if err := prepareBuildAttemptManifestBinding(attemptRel, &dispatchManifest); err != nil {
		finishAttempt(buildAttemptFailed, "failed to prepare direct build execution binding", err)
		rollbackCodexBuildFailure(originalState, phaseNum, startedAt, err)
		return nil, err
	}
	if err := store.SaveJSON(manifestRel, dispatchManifest); err != nil {
		finishAttempt(buildAttemptFailed, "failed to persist bound direct build manifest", err)
		rollbackCodexBuildFailure(originalState, phaseNum, startedAt, err)
		return nil, fmt.Errorf("failed to persist bound direct build manifest: %w", err)
	}
	if err := bindBuildAttemptManifest(attemptRel, dispatchManifest); err != nil {
		finishAttempt(buildAttemptFailed, "failed to bind direct build manifest", err)
		rollbackCodexBuildFailure(originalState, phaseNum, startedAt, err)
		return nil, err
	}
	if err := transitionBuildAttempt(attemptRel, buildAttemptDispatching, "worker dispatch started", dispatches, nil, "", nil); err != nil {
		finishAttempt(buildAttemptFailed, "failed to persist dispatch start", err)
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
	dispatches, claims, mode, err := executeCodexBuildDispatches(ctx, root, updatedPhase, dispatches, startedAt, buildInvoker, parallelMode, options.WorkerTimeout, options.CircuitBreakerThreshold, options.Verbose, dispatchManifest.ExecutionBinding)
	terminalClaims, attemptErr := recordBuildAttemptTerminal(root, attemptRel, phaseNum, startedAt, dispatches, claims, mode, err)
	if attemptErr != nil {
		finishAttempt(buildAttemptFailed, "failed to persist terminal worker results", attemptErr)
		rollbackCodexBuildFailure(originalState, phaseNum, startedAt, attemptErr)
		return nil, attemptErr
	}
	if err != nil {
		attemptFinished = true
		rollbackCodexBuildFailure(originalState, phaseNum, startedAt, err)
		// Classic failure theatre: a halted build is announced as a framed
		// operator moment, never a silent error return.
		emitVisualProgress(renderDecisionBlock("⚠", "Wave Failure — Build Halted",
			fmt.Sprintf("Phase %d dispatch failed: %v", phaseNum, err),
			"The phase's state was rolled back; nothing half-done was kept.",
			"Fix the cause, then rerun the build for this phase."))
		return nil, err
	}
	if err := store.SaveJSON(claimsRel, terminalClaims); err != nil {
		finishAttempt(buildAttemptFailed, "failed to persist current-build claims", err)
		rollbackCodexBuildFailure(originalState, phaseNum, startedAt, err)
		return nil, err
	}
	updatedState.State = colony.StateBUILT
	reconcileCompletedBuildTasks(&updatedState, phaseNum, dispatches)
	updatedPhase = updatedState.Plan.Phases[phaseNum-1]
	policy = enrichQueenExecutionPolicyWithSpawnBudget(policy, updatedState, updatedPhase, "build", reviewDepth, dispatches)
	if _, finalDispatches, err := writeCodexBuildArtifacts(root, updatedState, updatedPhase, buildDirRel, checkpointRel, claimsRel, dispatches, startedAt, mode, selectedTaskIDs, reviewDepth, policy, jobDecisions); err != nil {
		finishAttempt(buildAttemptFailed, "failed to persist final build artifacts", err)
		rollbackCodexBuildFailure(originalState, phaseNum, startedAt, err)
		return nil, err
	} else {
		dispatches = finalDispatches
	}
	var finalManifest codexBuildManifest
	if err := store.LoadJSON(manifestRel, &finalManifest); err != nil {
		finishAttempt(buildAttemptFailed, "failed to reload final build manifest", err)
		rollbackCodexBuildFailure(originalState, phaseNum, startedAt, err)
		return nil, fmt.Errorf("failed to reload final build manifest: %w", err)
	}
	finalManifest.CasteDecision = casteDecision
	finalManifest.JobDecisions = append([]coherentJobDecision{}, jobDecisions...)
	if err := store.SaveJSON(manifestRel, finalManifest); err != nil {
		finishAttempt(buildAttemptFailed, "failed to persist final caste decision", err)
		rollbackCodexBuildFailure(originalState, phaseNum, startedAt, err)
		return nil, fmt.Errorf("failed to persist final caste decision: %w", err)
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
		finishAttempt(buildAttemptFailed, "failed to commit built lifecycle state", err)
		rollbackCodexBuildFailure(originalState, phaseNum, startedAt, err)
		return nil, fmt.Errorf("failed to save built colony state: %w", err)
	}
	if err := transitionBuildAttempt(attemptRel, buildAttemptBuilt, "built lifecycle state committed", dispatches, terminalClaims, mode, nil); err != nil {
		visualFprintf(stderr, "warning: built state committed but build attempt final status could not be recorded: %v\n", err)
	}
	attemptFinished = true
	updatedState = committedState
	updatedPhase = updatedState.Plan.Phases[phaseNum-1]
	policy = enrichQueenExecutionPolicyWithSpawnBudget(policy, updatedState, updatedPhase, "build", reviewDepth, dispatches)
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

	// Suggestion analysis at build end — the same non-blocking hook the
	// host-manifest finalize path runs; without this the direct build path
	// never generated steering recommendations at all.
	suggestAnalyzeRan, pendingSuggestionCount := collectPendingSuggestions(root)

	result := map[string]interface{}{
		"phase":                    phaseNum,
		"colony_mode":              string(updatedState.EffectiveColonyMode()),
		"suggest_analyze_ran":      suggestAnalyzeRan,
		"pending_suggestions":      pendingSuggestionCount,
		"review_depth":             string(reviewDepth),
		"phase_name":               updatedPhase.Name,
		"state":                    updatedState.State,
		"next":                     "aether continue",
		"currentTask":              updatedPhase.Tasks,
		"dispatches":               dispatchMaps,
		"job_decisions":            append([]coherentJobDecision{}, jobDecisions...),
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
		"attempt":                  displayDataPath(attemptRel),
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

// coalesceSequentialDispatches merges a chain of dependent single-task steps
// into one worker.
//
// This is the one place the rule lives, so it can be read in full:
//
//	Two consecutive wave dispatches merge when the second is the only task in
//	its wave, the first was the only task in its wave, they share a caste, and
//	the second depends on the first and on nothing else.
//
// Everything else stays as it is. A wave holding more than one task is genuine
// parallel work and is never touched; a chain that changes caste part way
// through is two kinds of job and is left as two.
//
// The cost this removes is not theoretical. A phase of six dependent copy steps
// became six workers -- "waves 11-16 each contain exactly one builder, reason:
// single task in this wave" -- and each was a fresh agent that re-read the same
// source list from scratch before doing its share. One of those pairs was
// literally "copy the first six categories" and "copy the remaining six".
func coalesceSequentialDispatches(dispatches []codexBuildDispatch) []codexBuildDispatch {
	if len(dispatches) < 2 {
		return dispatches
	}

	// A wave is eligible only if it holds exactly one dispatch. Count first.
	perWave := map[int]int{}
	for _, d := range dispatches {
		if d.Stage == "wave" {
			perWave[d.Wave]++
		}
	}

	out := make([]codexBuildDispatch, 0, len(dispatches))
	// chainEnd is the wave of the last step folded into the dispatch currently
	// at the end of out. Read from the struct instead, a three-step chain would
	// merge steps 1 and 2 and then reject step 3, because the merged dispatch
	// still reports wave 1.
	chainEnd := -1
	for _, d := range dispatches {
		if d.Stage != "wave" || len(out) == 0 {
			out = append(out, d)
			chainEnd = d.Wave
			continue
		}
		prev := &out[len(out)-1]
		if !dispatchesFormOneJob(*prev, d, perWave, chainEnd) {
			out = append(out, d)
			chainEnd = d.Wave
			continue
		}
		mergeDispatchInto(prev, d)
		chainEnd = d.Wave
	}
	return out
}

func dispatchesFormOneJob(prev, next codexBuildDispatch, perWave map[int]int, chainEnd int) bool {
	if prev.Stage != "wave" || next.Stage != "wave" {
		return false
	}
	if prev.Caste != next.Caste {
		return false
	}
	if next.Wave != chainEnd+1 {
		return false
	}
	if perWave[next.Wave] != 1 || perWave[chainEnd] != 1 {
		return false
	}
	// The chain must be explicit. A task that declares no dependency may simply
	// have been listed in order, and merging it would serialise work that was
	// free to run alongside something else.
	if len(next.DependsOn) != 1 {
		return false
	}
	prevIDs := prev.CoveredTaskIDs
	if len(prevIDs) == 0 {
		prevIDs = []string{prev.TaskID}
	}
	for _, id := range prevIDs {
		if next.DependsOn[0] == id {
			return true
		}
	}
	return false
}

func mergeDispatchInto(target *codexBuildDispatch, next codexBuildDispatch) {
	if len(target.CoveredTaskIDs) == 0 {
		target.CoveredTaskIDs = []string{target.TaskID}
		// Number the first step only once, and only when a second joins it.
		target.Task = "1. " + strings.TrimSpace(target.Task)
	}
	target.CoveredTaskIDs = append(target.CoveredTaskIDs, next.TaskID)
	target.Task = strings.TrimSpace(target.Task) + "\n" +
		fmt.Sprintf("%d. %s", len(target.CoveredTaskIDs), strings.TrimSpace(next.Task))
	target.DeclaredPaths = uniqueSortedStrings(append(append([]string{}, target.DeclaredPaths...), next.DeclaredPaths...))
	// Wave and ExecutionWave stay at the first step's. The merged worker can
	// start as soon as the chain's first step could, and it does the rest in
	// order itself. Moving it to the last step's wave would make it wait for
	// nothing.
}

// plannedBuildDispatchesForSelectionWithState plans with no Queen proposal, so
// the deterministic keyword engine decides. Callers that have the Queen's
// judgement use the ...WithJudgement variant.
func plannedBuildDispatchesForSelectionWithState(phase colony.Phase, state colony.ColonyState, selectedTaskIDs []string, reviewDepth colony.VerificationDepth) []codexBuildDispatch {
	return plannedBuildDispatchesWithJudgement(phase, state, selectedTaskIDs, reviewDepth, nil, "")
}

func plannedBuildDispatchesWithJudgement(phase colony.Phase, state colony.ColonyState, selectedTaskIDs []string, reviewDepth colony.VerificationDepth, proposedCastes []string, casteReason string, reasons ...map[string]string) []codexBuildDispatch {
	var reasonMap map[string]string
	if len(reasons) > 0 {
		reasonMap = reasons[0]
	}
	dispatches, _, err := plannedBuildDispatchesWithJobProposals(
		phase, state, selectedTaskIDs, reviewDepth, proposedCastes, casteReason, reasonMap, nil,
	)
	if err != nil {
		return nil
	}
	return dispatches
}

// plannedBuildDispatchesWithJobProposals is the single production bridge from
// Queen judgement to the pure coherent-job planner. It resolves exactly one
// owner seed per selected task, plans jobs before assigning dispatch waves,
// and returns proposal decisions so callers can persist the full audit trail.
func plannedBuildDispatchesWithJobProposals(
	phase colony.Phase,
	state colony.ColonyState,
	selectedTaskIDs []string,
	reviewDepth colony.VerificationDepth,
	proposedCastes []string,
	casteReason string,
	reasons map[string]string,
	proposals []coherentJobProposal,
) ([]codexBuildDispatch, []coherentJobDecision, error) {
	depth := normalizedBuildDepth(state.ColonyDepth)
	selected := make(map[string]struct{}, len(selectedTaskIDs))
	for _, taskID := range selectedTaskIDs {
		selected[taskID] = struct{}{}
	}
	queenState := state
	queenState.ColonyDepth = depth
	queenState.VerificationDepth = string(reviewDepth)
	var reasonMaps []map[string]string
	if reasons != nil {
		reasonMaps = append(reasonMaps, reasons)
	}
	queenJudgement := queenApplyJudgement(proposedCastes, casteReason, phase, "build", queenState, reasonMaps...)
	queenCastes := stringSet(queenJudgement.Final)
	// queenAskedFor is the Queen's explicit proposal, not the effective team
	// after the required-caste floor unions itself in. D-08 (deterministic
	// checks are the floor, reviewers are judgement) and ruling D11 rule 4
	// (a phase is verified once) both rest on this distinction: the build's
	// verification-stage watcher below is gated on queenAskedFor, not
	// queenCastes, so the required-caste floor still guarantees a checker
	// somewhere in the pipeline (continue's deterministic floor) without
	// forcing a second, redundant reviewer dispatch at the build boundary.
	queenAskedFor := stringSet(queenJudgement.Proposed)
	applyBuildDispatchPolicyCastes(queenCastes, phase, depth, reviewDepth, queenAskedFor)

	seeds := make([]coherentJobTask, 0, len(phase.Tasks))
	for taskIdx, task := range phase.Tasks {
		taskID := buildTaskID(task, taskIdx)
		if len(selected) > 0 {
			if _, ok := selected[taskID]; !ok {
				continue
			}
		}
		seeds = append(seeds, coherentJobTask{
			Task:          task,
			ID:            taskID,
			TaskIndex:     taskIdx,
			Caste:         queenBuildTaskCaste(task, queenCastes),
			DeclaredPaths: declaredPathsForTask(task),
		})
	}
	jobPlan, err := planCoherentJobs(phase, seeds, proposals)
	if err != nil {
		return nil, nil, err
	}

	taskWaveBase := 10
	lastTaskExecutionWave := taskWaveBase + max(len(jobPlan.Waves), 1)
	dispatches := make([]codexBuildDispatch, 0, len(jobPlan.Jobs)+8)

	if len(selected) == 0 {
		dispatches = append(dispatches, queenBuildPreWaveDispatches(phase, queenCastes)...)
	}

	for _, job := range jobPlan.Jobs {
		if len(job.Tasks) == 0 {
			continue
		}
		primary := job.Tasks[0]
		coveredTaskIDs := append([]string{}, job.TaskIDs...)
		if len(coveredTaskIDs) == 1 {
			coveredTaskIDs = nil
		}
		dispatches = append(dispatches, codexBuildDispatch{
			Stage:          "wave",
			Wave:           job.Wave,
			ExecutionWave:  taskWaveBase + job.Wave,
			Caste:          job.OwnerCaste,
			Name:           coherentJobDispatchName(phase.ID, job),
			Task:           coherentJobDispatchTask(job.Tasks),
			Status:         "spawned",
			TaskID:         primary.ID,
			TaskIndex:      primary.TaskIndex,
			JobName:        job.Name,
			JobReason:      job.JobReason,
			JobSource:      job.Source,
			CoveredTaskIDs: coveredTaskIDs,
			DependsOn:      append([]string{}, job.DependsOn...),
			DeclaredPaths:  coherentJobDeclaredPaths(job.Tasks),
		})
	}

	if len(jobPlan.Jobs) == 0 && len(selected) == 0 {
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

	// One review wave. Probe, Auditor, Measurer and Chaos are independent
	// reviewers of the same finished code; none reads another's output, so they
	// spawn together. The Watcher below stays in the wave after them because it
	// is the final gate, not another reviewer.
	reviewWave := lastTaskExecutionWave + 1
	nextVerificationWave := reviewWave
	reviewersSpawned := false
	if len(selected) == 0 && queenCastes["probe"] {
		dispatches = append(dispatches, codexBuildSpecialistDispatch(phase, "probe", reviewWave, "probe", "Independent probe verification of builder claims"))
		reviewersSpawned = true
	}
	if len(selected) == 0 {
		postWaveDispatches := queenBuildPostWaveDispatches(phase, queenCastes, reviewWave)
		dispatches = append(dispatches, postWaveDispatches...)
		reviewersSpawned = reviewersSpawned || len(postWaveDispatches) > 0
	}
	if reviewersSpawned {
		nextVerificationWave = reviewWave + 1
	}

	// D-08 / ruling D11 rule 4: each phase is verified once. The program's
	// free checks (build, types, lint, tests, claimed-files-exist, criterion
	// evidence) are the deterministic floor and run on every phase regardless
	// -- agent review lives in `continue`, not here. The build side dispatches
	// a watcher into its own "verification" stage only when the Queen's
	// proposal explicitly named one; the required-caste floor restoring
	// "watcher" into queenCastes no longer forces a build-time dispatch, so
	// the same caste is not summoned at both the build and continue
	// boundaries unless the Queen asks.
	if queenAskedFor["watcher"] {
		dispatches = append(dispatches, codexBuildDispatch{
			Stage:         "verification",
			ExecutionWave: nextVerificationWave,
			Caste:         "watcher",
			Name:          deterministicAntName("watcher", fmt.Sprintf("phase:%d:watcher", phase.ID)),
			Task:          "Independent verification before advancement" + findingsInjectionForCaste("watcher"),
			Status:        "spawned",
		})
	}

	return dispatches, append([]coherentJobDecision{}, jobPlan.Decisions...), nil
}

func coherentJobDispatchTask(tasks []coherentJobTask) string {
	if len(tasks) == 0 {
		return ""
	}
	if len(tasks) == 1 {
		return strings.TrimSpace(tasks[0].Task.Goal)
	}
	items := make([]string, 0, len(tasks))
	for index, task := range tasks {
		items = append(items, fmt.Sprintf("%d. %s", index+1, strings.TrimSpace(task.Task.Goal)))
	}
	return strings.Join(items, "\n")
}

func coherentJobDeclaredPaths(tasks []coherentJobTask) []string {
	var paths []string
	for _, task := range tasks {
		paths = append(paths, task.DeclaredPaths...)
	}
	return uniqueSortedStrings(paths)
}

func coherentJobDispatchName(phaseID int, job coherentJob) string {
	if len(job.Tasks) == 1 {
		task := job.Tasks[0]
		return deterministicAntName(job.OwnerCaste, fmt.Sprintf("phase:%d:task:%d:%s", phaseID, task.TaskIndex, task.Task.Goal))
	}
	return deterministicAntName(job.OwnerCaste, fmt.Sprintf("phase:%d:job:%s:%s", phaseID, job.OwnerCaste, strings.Join(job.TaskIDs, ",")))
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

// applyBuildDispatchPolicyCastes applies the depth policy for the two most
// expensive optional specialists.
//
// queenChose names castes the Queen asked for explicitly after reading the
// phase. Those are exempt from the deletion below. Until they were, this
// function ran one line after the Queen's team was computed and unconditionally
// removed both — so a Queen that read "the dashboard feels sluggish" and asked
// for a Measurer was overruled by a depth rule that had never seen the phase.
// The decision record still said measurer; nothing spawned. Judgement that a
// later line silently discards is worse than no judgement, because it reads as
// working.
func applyBuildDispatchPolicyCastes(queenCastes map[string]bool, phase colony.Phase, depth string, reviewDepth colony.VerificationDepth, queenChose map[string]bool) {
	if !queenChose["measurer"] {
		delete(queenCastes, "measurer")
	}
	if !queenChose["chaos"] {
		delete(queenCastes, "chaos")
	}

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

// queenBuildPreWavePlans and queenBuildPostWavePlans are the only routes by
// which a selected specialist caste becomes a build dispatch. They are
// package-level so TestEveryBuildSelectableCasteCanDispatch can prove that
// every caste the Queen may select for a build appears in some dispatch
// path — a caste selectable but absent here consumes a budget slot and
// silently displaces a specialist that would actually have run.
var queenBuildPreWavePlans = []struct {
	caste string
	stage string
	wave  int
	task  string
}{
	// Wave 1 — evidence gatherers. These two write artifacts (git history
	// findings, phase research) that a planner may consult, so they go first.
	{"archaeologist", "prep", 1, "Git history analysis before implementation"},
	{"oracle", "research", 1, "Phase research and implementation risks"},

	// Wave 2 — planners. Every one of these reads the same phase brief and
	// produces an independent plan; none consumes another's output.
	// findingsInjectionForCaste only appends a *write* instruction for four
	// castes, so there is no read dependency between any pair here.
	//
	// They previously occupied waves 3-8, one or two per wave, which made a
	// build serial before a line of code was written: a real phase spawned
	// eleven dispatches across nine waves and took an hour, mostly waiting.
	// Same workers, same coverage — concurrent instead of queued.
	{"architect", "design", 2, "Design boundaries before coding"},
	{"ambassador", "integration", 2, "External integration design before implementation"},
	{"gatekeeper", "security", 2, "Security boundaries and auth risk review before implementation"},
	{"includer", "accessibility", 2, "Accessibility requirements and inclusive interaction review"},
	{"weaver", "refactor", 2, "Refactoring seams and simplification plan before implementation"},
	{"tracker", "diagnosis", 2, "Root-cause investigation and regression context before implementation"},
	{"keeper", "knowledge", 2, "Knowledge preservation plan for reusable patterns"},
	{"chronicler", "documentation", 2, "Documentation surface and changelog planning"},
	{"medic", "health", 2, "Runtime health and repair risk review"},
	{"fixer", "repair", 2, "Repair strategy and remediation boundaries"},
	{"porter", "delivery", 2, "Delivery, packaging, and release handling review"},
	{"sage", "wisdom", 2, "Learning synthesis and reusable pattern capture"},
}

var queenBuildPostWavePlans = []struct {
	caste string
	stage string
	task  string
}{
	{"auditor", "audit", "Quality and compliance review after implementation"},
	{"measurer", "measurement", "Performance and cost surface review after implementation"},
	{"chaos", "resilience", "Resilience probing after specialist verification"},
}

func queenBuildPreWaveDispatches(phase colony.Phase, queenCastes map[string]bool) []codexBuildDispatch {
	dispatches := make([]codexBuildDispatch, 0, len(queenBuildPreWavePlans))
	for _, plan := range queenBuildPreWavePlans {
		if !queenCastes[plan.caste] {
			continue
		}
		dispatches = append(dispatches, codexBuildSpecialistDispatch(phase, plan.stage, plan.wave, plan.caste, plan.task+findingsInjectionForCaste(plan.caste)))
	}
	return dispatches
}

func queenBuildPostWaveDispatches(phase colony.Phase, queenCastes map[string]bool, startExecutionWave int) []codexBuildDispatch {
	// All post-wave reviewers examine the same finished code and share no
	// inputs, so they occupy one wave. Each previously took its own
	// incrementing wave, which serialised the review phase for no reason: an
	// Auditor cannot learn anything from waiting for a Measurer.
	dispatches := make([]codexBuildDispatch, 0, len(queenBuildPostWavePlans))
	for _, plan := range queenBuildPostWavePlans {
		if !queenCastes[plan.caste] {
			continue
		}
		dispatches = append(dispatches, codexBuildSpecialistDispatch(phase, plan.stage, startExecutionWave, plan.caste, plan.task+findingsInjectionForCaste(plan.caste)))
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

// queenBuildTaskFallbackCastes is the ordered chain of castes a phase task can
// be assigned to when its suggested caste was not selected. Package-level for
// the same dispatchability invariant as the wave plan tables above.
var queenBuildTaskFallbackCastes = []string{"builder", "scout", "oracle", "weaver", "tracker", "fixer"}

func queenBuildFallbackTaskCaste(queenCastes map[string]bool) string {
	for _, caste := range queenBuildTaskFallbackCastes {
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
			Strategy:      executionStrategyForCastes(stage, castes, parallelMode),
			WorkerCount:   len(dispatches),
			Castes:        castes,
			Reason:        executionReasonForBuildStep(stage, taskWave, len(dispatches), parallelMode),
		})
	}
	return plans
}

// nonSourceWritingCastes are castes whose dispatch brief confines them to
// analysis, review evidence, or a scoped .aether/ artifact directory — never
// project source or tests. Two of them running at once cannot collide on a
// file, so a step containing only these is safe to spawn concurrently even in
// in-repo mode, where every worker shares one working tree.
//
// Derived from behavioralRestrictionsForCaste in pkg/codex/permission_profile.go
// and kept deliberately conservative: castes with no restriction at all
// (builder, weaver, fixer, medic, porter, ambassador, keeper, chaos) may touch
// anything, and probe writes test files, so any step containing one of those
// stays serial. Note these are prompt-level promises, not a sandbox — which is
// exactly why the list is short and errs toward serial.
var nonSourceWritingCastes = map[string]bool{
	"includer":             true, // the only enforced repository_read_only caste
	"architect":            true,
	"route_setter":         true,
	"sage":                 true,
	"archaeologist":        true,
	"auditor":              true,
	"gatekeeper":           true,
	"measurer":             true,
	"tracker":              true,
	"watcher":              true,
	"scout":                true,
	"oracle":               true,
	"chronicler":           true,
	"surveyor_nest":        true,
	"surveyor_disciplines": true,
	"surveyor_pathogens":   true,
	"surveyor_provisions":  true,
}

// executionStrategyForCastes decides whether a step's workers may spawn
// together. Collapsing independent specialists into one wave achieves nothing
// on its own: the wrapper obeys this field, so a wave marked serial is still
// executed one worker at a time.
//
// Worktree mode isolates every worker in its own checkout, so anything may run
// concurrently. In-repo mode shares one tree, so only steps composed entirely of
// non-source-writing castes qualify.
func executionStrategyForCastes(stage string, castes []string, parallelMode colony.ParallelMode) string {
	if len(castes) < 2 {
		return "serial"
	}
	if parallelMode == colony.ModeWorktree {
		return "parallel"
	}
	if stage == "wave" {
		// Task waves are builders sharing one tree. Never concurrent in-repo.
		return "serial"
	}
	for _, caste := range castes {
		if !nonSourceWritingCastes[strings.ToLower(strings.TrimSpace(caste))] {
			return "serial"
		}
	}
	return "parallel"
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
		ExecutionModel:         executionModel,
		WaveCount:              len(executionPlan),
		WorkerCount:            len(dispatches),
		SharedTimeoutSeconds:   0,
		WorkerTimeoutSeconds:   int(effectiveBuildDispatchTimeout(workerTimeout) / time.Second),
		DeadlinePolicy:         "Each build worker gets its own timeout. The Queen runtime advances execution_wave stages in order and records each terminal worker result.",
		DependencyBehavior:     "Builder task waves run before post-build specialist verification. Watcher verification is the final build execution stage before finalization.",
		FallbackBehavior:       "Runtime worker dispatch rolls back failed direct builds; host-orchestrated plan-only builds must call build-finalize with fresh worker results.",
		FallbackVisibility:     []string{"dispatch_mode", "dispatches", "execution_plan", "provider_diagnostics", "worker_handoffs", "result_collection"},
		CoordinationPath:       dataContractPath("spawn-tree.txt"),
		ResultArtifactPaths:    []string{finalizerCompletionTempPattern},
		ResultCollectionPolicy: "A structurally valid completed result wins over a timeout placeholder for the same worker; malformed JSON, duplicate terminal results, stale manifests, and .aether/data completion files are rejected.",
		ArtifactPaths: []string{
			dataContractPath("build", "phase-<phase>", "manifest.json"),
			dataContractPath("build", "phase-<phase>", "worker-briefs"),
			dataContractPath("build", "phase-<phase>", "result-collection.json"),
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

func executeCodexBuildDispatches(ctx context.Context, root string, phase colony.Phase, dispatches []codexBuildDispatch, startedAt time.Time, invoker codex.WorkerInvoker, parallelMode colony.ParallelMode, workerTimeout time.Duration, circuitBreakerThreshold int, verbose bool, executionBinding *codex.ExecutionBinding) ([]codexBuildDispatch, *codex.ClaimsSummary, string, error) {
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

	// PheromoneSection is deliberately left unset here (D-190-03-A, closed by
	// 190-05). capsule (resolveCodexWorkerContext(), above) already renders its
	// own "## Pheromone Signals" section unconditionally whenever a signal is
	// active (cmd/colony_prime_context.go:571) -- populating a second,
	// independent PheromoneSection field on top of it would deliver the same
	// steering text twice into AssemblePrompt/AssembleHostedPrompt
	// (pkg/codex/prompt.go), under two different headings, to every
	// native-dispatched worker (including every autopilot /ant-run build).
	// The capsule is this path's sole steering channel now, mirroring the
	// "one home" decision 190-03 already made for the wrapper plan-only flow.
	// See resolvePheromoneSection's doc comment for which callers still need it.
	workerDispatches := make([]codex.WorkerDispatch, 0, len(dispatches))
	indexByName := make(map[string]int, len(dispatches))
	dispatchByName := make(map[string]codex.WorkerDispatch, len(dispatches))
	for i, dispatch := range dispatches {
		agentName := codexAgentNameForCaste(dispatch.Caste)
		providerRunID, err := codex.NewExecutionRunID()
		if err != nil {
			return nil, nil, "", err
		}
		workerDispatch := codex.WorkerDispatch{
			ID:                fmt.Sprintf("phase-%d-dispatch-%d", phase.ID, i+1),
			WorkerName:        dispatch.Name,
			AgentName:         agentName,
			AgentTOMLPath:     dispatchAgentPath(root, invoker, agentName),
			Caste:             dispatch.Caste,
			TaskID:            normalizedDispatchTaskID(dispatch),
			TaskBrief:         renderCodexBuildWorkerBrief(root, phase, dispatch, startedAt),
			ContextCapsule:    capsule,
			HandoffSection:    dispatch.HandoffSection,
			Workflow:          "build",
			Phase:             phase.ID,
			SkillSection:      resolveSkillSectionForWorkflow("build", dispatch.Caste, dispatch.Task),
			Root:              root,
			TrackingRoot:      root,
			Timeout:           workerTimeout,
			Wave:              normalizedDispatchWave(dispatch),
			PermissionProfile: dispatch.PermissionProfile,
			ExecutionBinding:  executionBinding,
			ProviderRunID:     providerRunID,
			DeclaredPaths:     append([]string{}, dispatch.DeclaredPaths...),
		}
		workerDispatches = append(workerDispatches, workerDispatch)
		indexByName[dispatch.Name] = i
		dispatchByName[dispatch.Name] = workerDispatch
	}

	if parallelMode == colony.ModeWorktree {
		if err := validateDeclaredWorktreeOwnership(workerDispatches); err != nil {
			return nil, nil, "", err
		}
	}

	cb := NewCircuitBreaker(circuitBreakerThreshold)
	// Per D-02/D-04: set verbose flag before dispatch so filtered functions work correctly
	setBuildVerbose(verbose)

	// Per D-09/D-12: queen owns the wave loop. Build calls queen once.
	waveDispatchFn := func(ctx context.Context, waveDispatches []codex.WorkerDispatch, waveNum int) ([]codex.DispatchResult, error) {
		return dispatchCodexBuildWorkersWithReconciliation(ctx, root, phase, waveDispatches, invoker, startedAt, parallelMode, cb)
	}
	summary, results, err := queenWaveLifecycle(ctx, workerDispatches, waveDispatchFn, phase, cb, phase.ID)
	// Persist wave summary JSON for Phase 99 consumption (D-07)
	_ = writeWaveSummary(phase.ID, summary)

	// Per D-05: consolidate queen decisions into audit file (Phase 99 OUT-02)
	audit := consolidateQueenAudit(phase.ID)
	_ = writeAuditFile(phase.ID, audit)

	// Per D-09: render phase-end summary with actions needed (Phase 99 OUT-01)
	renderPhaseEndSummary(summary, phase.ID)

	// Clean up any worktrees that weren't properly finalized during dispatch.
	// Per D-02, a destructive-capable path must never handle its own error
	// silently — cleanupBuildWorktrees itself never destroys unsaved or
	// unmerged work (CR-05), but if the state update it needs fails, that
	// failure must be visible rather than swallowed, since it means the
	// leftover Allocated/InProgress entries from this build were not
	// examined at all.
	cleaned, orphaned, cleanupErr := cleanupBuildWorktrees(phase.ID)
	if cleanupErr != nil {
		emitVisualProgress(fmt.Sprintf("Worktree cleanup could not run (%v) — any unfinalized worker workspaces from this build were left untouched", cleanupErr))
	} else if cleaned > 0 || orphaned > 0 {
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
			filteredFprintln(stdout, codex.SanitizeWorkerDiagnosticOutput(result.WorkerResult.RawOutput))
		}

		if result.Error != nil && len(dispatches[idx].Blockers) == 0 {
			dispatches[idx].Blockers = []string{codex.SanitizeWorkerDiagnosticOutput(result.Error.Error())}
		}
	}

	claims := codex.ExtractClaims(results)
	if err := validateRuntimeNoChangeEvidence(results); err != nil {
		return dispatches, claims, mode, err
	}
	requireFileEvidence := false
	switch codex.PlatformFromInvoker(invoker) {
	case codex.PlatformCodex, codex.PlatformClaude, codex.PlatformOpenCode:
		requireFileEvidence = true
	}
	if err := validateRuntimeBuildDispatchResults(phase, dispatches, claims, requireFileEvidence); err != nil {
		return dispatches, claims, mode, err
	}
	return dispatches, claims, mode, nil
}

// validateRuntimeNoChangeEvidence applies the no-change evidence rule on the
// in-process dispatch lane. A completed_no_change claim buys an exemption
// from the file-changes requirement, so it must pay for it with the same
// evidence the external lane demands: a summary saying why, a passing
// handoff verification, and the commands actually run.
func validateRuntimeNoChangeEvidence(results []codex.DispatchResult) error {
	for _, result := range results {
		if !isNoChangeExternalBuildStatus(normalizeExternalBuildStatus(result.Status)) {
			continue
		}
		if result.WorkerResult == nil {
			return fmt.Errorf("worker %s claims completed_no_change with no result payload -- an honest no-change needs the verification it ran", result.WorkerName)
		}
		missing := noChangeEvidenceMissingFrom(
			result.WorkerResult.Summary,
			result.WorkerResult.Handoff.VerificationStatus,
			result.WorkerResult.Handoff.CommandsRun,
		)
		if len(missing) > 0 {
			return fmt.Errorf("worker %s claims completed_no_change without evidence -- missing: %s", result.WorkerName, strings.Join(missing, "; "))
		}
	}
	return nil
}

func validateRuntimeBuildDispatchResults(phase colony.Phase, dispatches []codexBuildDispatch, claims *codex.ClaimsSummary, requireFileEvidence bool) error {
	if len(dispatches) == 0 {
		return fmt.Errorf("build dispatch produced no worker results")
	}

	failed := make([]string, 0)
	noChangeCount := 0
	successCount := 0
	for _, dispatch := range dispatches {
		status := strings.ToLower(strings.TrimSpace(dispatch.Status))
		if isSuccessfulExternalBuildStatus(status) {
			successCount++
			if isNoChangeExternalBuildStatus(status) {
				noChangeCount++
			}
			continue
		}
		if status == "" {
			status = "missing"
		}
		detail := firstNonEmpty(strings.Join(dispatch.Blockers, "; "), dispatch.Summary)
		if detail != "" {
			failed = append(failed, fmt.Sprintf("%s=%s (%s)", dispatch.Name, status, detail))
			continue
		}
		failed = append(failed, fmt.Sprintf("%s=%s", dispatch.Name, status))
	}
	if len(failed) > 0 {
		return fmt.Errorf("build dispatch did not complete cleanly: %s", strings.Join(failed, ", "))
	}

	if !requireFileEvidence {
		return nil
	}
	manifest := codexBuildManifest{Tasks: codexBuildTaskPlans(phase)}
	if isVerificationOnlyBuildManifest(manifest) {
		return nil
	}
	if phase.Mode == colony.PhaseModeDiscovery && hasDurableDiscoveryDispatchEvidence(dispatches) {
		return nil
	}
	// Every successful dispatch honestly reported completed_no_change
	// (ruling D6): the phase's work was to verify existing behavior, so
	// demanding file changes here would force the fake edit the accounting
	// contract exists to forbid. This exemption is only safe because the
	// evidence rule ran first -- validateRuntimeNoChangeEvidence on this
	// lane, the merge path's no_change_evidence gate on the external one.
	if successCount > 0 && noChangeCount == successCount {
		return nil
	}
	if claims == nil || len(claims.FilesCreated)+len(claims.FilesModified)+len(claims.TestsWritten) == 0 {
		return fmt.Errorf("build dispatch completed without observed file changes for an implementation phase")
	}
	return nil
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

func buildCodexBuildManifest(root string, state colony.ColonyState, phase colony.Phase, checkpointRel, claimsRel string, dispatches []codexBuildDispatch, startedAt time.Time, dispatchMode string, selectedTaskIDs []string, workerBriefs []string, planOnly bool, reviewDepth colony.VerificationDepth) codexBuildManifest {
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
	policy := recommendQueenExecutionPolicy(state, phase, len(state.Plan.Phases), codexQueenExecutionPolicyInput{
		VerificationDepth: string(reviewDepth),
		DispatchWorkers:   buildWorkerDispatchOptIn(dispatchMode),
	})
	policy = enrichQueenExecutionPolicyWithSpawnBudget(policy, state, phase, "build", reviewDepth, dispatches)
	planHash, _ := planStateHash(state.Plan)

	// Compute the colony-prime capsule once, only for the plan-only wrapper
	// manifest — the hosted path (executeCodexBuildDispatches) already
	// computes and shares its own capsule, and the finalize record does not
	// deliver prompts. This is the single call site for this field; it must
	// never be computed inside a per-dispatch loop (that would reintroduce
	// the duplication CONTEXT-03 exists to prevent).
	contextCapsule := ""
	if planOnly {
		contextCapsule = resolveCodexWorkerContext()
	}

	return codexBuildManifest{
		Phase:                   phase.ID,
		PhaseName:               phase.Name,
		PhaseMode:               phase.Mode,
		Goal:                    goal,
		Root:                    root,
		ColonyMode:              string(state.EffectiveColonyMode()),
		PlanOnly:                planOnly,
		ParallelMode:            string(effectiveParallelMode(state)),
		WaveExecution:           buildWaveExecutionPlans(dispatches, effectiveParallelMode(state)),
		ExecutionPlan:           buildExecutionPlans(dispatches, effectiveParallelMode(state)),
		ColonyDepth:             normalizedBuildDepth(state.ColonyDepth),
		DispatchMode:            strings.TrimSpace(dispatchMode),
		HostPlatform:            buildHostPlatform(),
		ExecutionOwner:          buildExecutionOwner(dispatchMode, planOnly),
		WorkerDispatchOptIn:     buildWorkerDispatchOptIn(dispatchMode),
		GeneratedAt:             startedAt.Format(time.RFC3339),
		PlanRevisionID:          activePlanRevisionID(state.Plan),
		PlanStateHash:           planHash,
		State:                   string(state.State),
		Checkpoint:              checkpoint,
		ClaimsPath:              claimsPath,
		WorkerBriefs:            briefs,
		ContextCapsule:          contextCapsule,
		Dispatches:              append([]codexBuildDispatch{}, dispatches...),
		SelectedTasks:           append([]string{}, selectedTaskIDs...),
		Tasks:                   codexBuildTaskPlans(phase),
		SuccessCriteria:         append([]string{}, phase.SuccessCriteria...),
		CriterionEvidencePolicy: phaseCriterionEvidencePolicy(phase),
		EvidenceRequirements:    flattenPhaseCriterionEvidenceRequirements(phase),
		ReviewDepth:             string(reviewDepth),
		DispatchContract:        buildDispatchContractForDispatches(dispatches, effectiveParallelMode(state), 0),
		ProfileContract:         workflowProfileContract(reviewDepth),
		QueenRecommendation:     recommendQueenWorkflowProfile(state, phase, len(state.Plan.Phases)),
		QueenExecutionPolicy:    policy,
	}
}

func hasDurableDiscoveryDispatchEvidence(dispatches []codexBuildDispatch) bool {
	foundTask := false
	for _, dispatch := range dispatches {
		if strings.TrimSpace(dispatch.TaskID) == "" {
			continue
		}
		foundTask = true
		if !strings.EqualFold(strings.TrimSpace(dispatch.Status), "completed") || strings.TrimSpace(dispatch.Summary) == "" {
			return false
		}
	}
	return foundTask
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
			"stage":              dispatch.Stage,
			"execution_wave":     normalizedDispatchWave(dispatch),
			"caste":              dispatch.Caste,
			"agent_name":         codexAgentNameForCaste(dispatch.Caste),
			"name":               dispatch.Name,
			"task":               dispatch.Task,
			"status":             dispatch.Status,
			"permission_profile": dispatch.PermissionProfile,
		}
		if dispatch.Wave > 0 {
			entry["wave"] = dispatch.Wave
		}
		if dispatch.TaskID != "" {
			entry["task_id"] = dispatch.TaskID
		}
		if len(dispatch.CoveredTaskIDs) > 0 {
			entry["covered_task_ids"] = append([]string{}, dispatch.CoveredTaskIDs...)
		}
		if dispatch.JobName != "" {
			entry["job_name"] = dispatch.JobName
		}
		if dispatch.JobReason != "" {
			entry["job_reason"] = dispatch.JobReason
		}
		if dispatch.JobSource != "" {
			entry["job_source"] = dispatch.JobSource
		}
		if len(dispatch.DependsOn) > 0 {
			entry["depends_on"] = dispatch.DependsOn
		}
		if len(dispatch.DeclaredPaths) > 0 {
			entry["declared_paths"] = append([]string{}, dispatch.DeclaredPaths...)
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
		if strings.TrimSpace(dispatch.Brief) != "" {
			entry["brief"] = dispatch.Brief
		}
		if strings.TrimSpace(dispatch.BriefPath) != "" {
			entry["brief_path"] = dispatch.BriefPath
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

// writeBuildWorkerBriefFiles writes each dispatch's composed worker brief to
// a file on disk under buildDirRel/worker-briefs/{name}.md, falling back to
// composeBuildManifestBrief when .Brief is already empty (mirroring the
// direct dispatch path's long-standing fallback: some callers of
// writeCodexBuildArtifacts never run attachBuildDispatchContext, so Brief can
// arrive empty here).
//
// writeCodexBuildArtifacts (direct/native `aether build <phase>`) and
// runCodexBuildPlanOnlyWithOptions (`aether build <phase> --plan-only`) both
// converge on this single helper, distinguished only by clearInlineBrief:
// the direct path needs .Brief to survive on disk in the manifest (two
// existing tests require it -- TestWorkerBriefFileHoldsComposedBrief,
// TestDispatchEntryCarriesBriefPath), so it passes false. The plan-only path
// needs .Brief cleared once the file write succeeds, so the composed brief
// never ships twice in the same JSON envelope (result.dispatches[] and
// result.dispatch_manifest.dispatches[] both read off this same slice), so it
// passes true.
//
// This function's OWN fallback composition (below, reached only when .Brief
// arrives empty -- i.e. the direct path, since runCodexBuildPlanOnlyWithOptions
// always runs attachBuildDispatchContext first) always passes
// includeSteeringSections=true to composeBuildManifestBrief: the direct
// path's manifest.json carries no ContextCapsule alongside this Brief (see
// codexBuildManifest.ContextCapsule's doc comment -- "Only populated when
// planOnly"), so this artifact must stay self-contained (190-03,
// D-190-01-A). Do not change this to false -- that would zero out pheromone
// and prior-handoff delivery on the one path that has no other channel for
// them.
//
// A write failure returns (nil, nil, err) immediately, before any blanking
// for that dispatch (or any dispatch after it) happens. The returned slice is
// nil, not partially populated: every caller treats a non-nil error as fatal
// and never inspects the dispatches value, so discarding it is simpler than
// reasoning about a half-written slice. (190-190/IN-02: this comment used to
// describe returning the slice with .Brief still populated and .BriefPath
// empty, which is not what the code does.) The no-silent-drop guarantee is
// still real and is what matters: the inline Brief is only ever cleared after
// its file write has succeeded, so no path can lose a worker's prompt.
func writeBuildWorkerBriefFiles(root string, phase colony.Phase, buildDirRel string, dispatches []codexBuildDispatch, startedAt time.Time, clearInlineBrief bool) ([]string, []codexBuildDispatch, error) {
	briefPaths := make([]string, 0, len(dispatches))

	for i := range dispatches {
		briefRel := filepath.ToSlash(filepath.Join(buildDirRel, "worker-briefs", fmt.Sprintf("%s.md", dispatches[i].Name)))
		content := dispatches[i].Brief
		if strings.TrimSpace(content) == "" {
			content = composeBuildManifestBrief(root, phase, dispatches[i], startedAt, true)
			dispatches[i].Brief = content
		}
		if err := store.AtomicWrite(briefRel, []byte(content)); err != nil {
			return nil, nil, fmt.Errorf("failed to write worker brief for %s: %w", dispatches[i].Name, err)
		}
		displayPath := displayDataPath(briefRel)
		briefPaths = append(briefPaths, displayPath)
		dispatches[i].BriefPath = displayPath
		if clearInlineBrief {
			// The file on disk is now the single source of truth for this
			// dispatch's brief -- nothing downstream (codexBuildDispatchMaps or
			// the manifest's own Dispatches value copy) may ship the same bytes
			// a second time under the inline "brief" key.
			dispatches[i].Brief = ""
		}
	}
	sort.Strings(briefPaths)

	return briefPaths, dispatches, nil
}

func writeCodexBuildArtifacts(root string, state colony.ColonyState, phase colony.Phase, buildDirRel, checkpointRel, claimsRel string, dispatches []codexBuildDispatch, startedAt time.Time, dispatchMode string, selectedTaskIDs []string, reviewDepth colony.VerificationDepth, policy codexQueenExecutionPolicy, jobDecisions []coherentJobDecision) ([]string, []codexBuildDispatch, error) {
	briefPaths, dispatches, err := writeBuildWorkerBriefFiles(root, phase, buildDirRel, dispatches, startedAt, false)
	if err != nil {
		return nil, nil, err
	}

	briefOutputs := map[string]string{}
	for i := range dispatches {
		briefOutputs[dispatches[i].Name] = dispatches[i].BriefPath
	}
	finalOutputs := map[string][]string{}

	if isFinalBuildDispatchMode(dispatchMode) {
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

	manifest := buildCodexBuildManifest(root, state, phase, checkpointRel, claimsRel, dispatches, startedAt, dispatchMode, selectedTaskIDs, briefPaths, false, reviewDepth)
	manifest.JobDecisions = append([]coherentJobDecision{}, jobDecisions...)
	manifest.QueenExecutionPolicy = enrichQueenExecutionPolicyWithSpawnBudget(policy, state, phase, "build", reviewDepth, dispatches)
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
		// completed_no_change completes its tasks too (ruling D6): the task
		// was to make the behavior true, and the worker proved it already
		// is. Skipping it here would leave the task unfinished forever and
		// block phase advance — the same trap the merged-chain fix below
		// closed for coverage. interrupted stays excluded: terminal, not
		// success.
		status := strings.TrimSpace(dispatch.Status)
		if status != "completed" && !isNoChangeExternalBuildStatus(status) {
			continue
		}
		// A worker that owns a merged chain finishes every step in it, so every
		// step it covered is complete. Reading TaskID alone left the later steps
		// marked unfinished forever: the phase could never advance, and the one
		// worker that did the work looked like it had only done the first bit.
		for _, taskID := range dispatchCoveredTaskIDs(dispatch) {
			completed[taskID] = struct{}{}
		}
	}
	return completed
}

// dispatchCoveredTaskIDs returns every task a dispatch is responsible for. For
// an ordinary dispatch that is just its own task; for a merged chain it is all
// of them.
func dispatchCoveredTaskIDs(dispatch codexBuildDispatch) []string {
	if len(dispatch.CoveredTaskIDs) > 0 {
		out := make([]string, 0, len(dispatch.CoveredTaskIDs))
		for _, id := range dispatch.CoveredTaskIDs {
			if trimmed := strings.TrimSpace(id); trimmed != "" {
				out = append(out, trimmed)
			}
		}
		return out
	}
	if taskID := strings.TrimSpace(dispatch.TaskID); taskID != "" {
		return []string{taskID}
	}
	return nil
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
		rollback.Worktrees = mergeBuildFailureWorktrees(rollback.Worktrees, current.Worktrees)
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

func mergeBuildFailureWorktrees(previous, observed []colony.WorktreeEntry) []colony.WorktreeEntry {
	merged := append([]colony.WorktreeEntry{}, previous...)
	indexByKey := make(map[string]int, len(merged))
	for i, entry := range merged {
		key := firstNonEmpty(strings.TrimSpace(entry.ID), strings.TrimSpace(entry.Branch))
		if key != "" {
			indexByKey[key] = i
		}
	}
	for _, entry := range observed {
		key := firstNonEmpty(strings.TrimSpace(entry.ID), strings.TrimSpace(entry.Branch))
		if key == "" {
			continue
		}
		if idx, ok := indexByKey[key]; ok {
			merged[idx] = entry
			continue
		}
		indexByKey[key] = len(merged)
		merged = append(merged, entry)
	}
	return merged
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

func renderCodexBuildWorkerBrief(root string, phase colony.Phase, dispatch codexBuildDispatch, startedAt time.Time) string {
	var b strings.Builder
	// Platform-neutral heading: this brief is delivered verbatim to workers on
	// every host platform (Claude, OpenCode, Codex), so it must not claim one.
	b.WriteString("# Build Dispatch\n\n")
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

	// Read-cache discipline removed: 683 chars per prompt telling the worker not
	// to re-read unchanged files. Claude Code and OpenCode both cache reads and
	// tell the model directly when a file is unchanged since its last read, so
	// this restates a message the harness already delivers more reliably.

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

	relatedTasks := findDispatchTasks(phase, dispatch)
	renderDispatchTaskItemsSection(&b, "Task Constraints", relatedTasks, func(t *colony.Task) []string { return t.Constraints })
	renderDispatchTaskItemsSection(&b, "Hints", relatedTasks, func(t *colony.Task) []string { return t.Hints })
	renderDispatchTaskItemsSection(&b, "Task Success Criteria", relatedTasks, func(t *colony.Task) []string { return t.SuccessCriteria })
	b.WriteString(renderUnresolvedDispatchTaskNotice(relatedTasks))

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

	// Heartbeat protocol removed: it asked the worker to write a file "roughly
	// every 30 seconds" during its own turn. A model has no timer and cannot
	// act between turns, so this instruction has never been satisfiable. It
	// cost ~380 chars of every prompt and taught workers to ignore an
	// instruction, which is worse than costing nothing.

	if graphContext := renderCodegraphContextForText(root, codegraphTextPartsForBuildBrief(phase, dispatch), codegraphWorkerContextBudgetChars); graphContext != "" {
		b.WriteString("\n")
		b.WriteString(graphContext)
		b.WriteString("\n")
	}

	// Playbook injection removed. Measured on a real brief, it was 5,733 of
	// 7,485 characters — 76.6% — against an assignment of 79. Worse than the
	// size: the old playbook filter fed workers orchestrator playbooks,
	// truncated at 2,800 chars, so a Builder received the opening of
	// build-wave.md instructing it "YOU (the Queen) will spawn workers
	// directly. Do NOT delegate to a single Prime Worker." That is a direct
	// role contradiction carried at five times the mass of the real task, and
	// its own "## " headings collided with the brief's structure so a worker
	// could not tell where its instructions ended.
	//
	// Playbooks are no longer loaded by the runtime at all: workers get the
	// lean brief, and orchestration lives in the host-manifest flow. The
	// command-playbook docs remain as reference material only.

	if surveySection := resolveSurveySection(); surveySection != "" {
		b.WriteString("\n")
		b.WriteString(surveySection)
		b.WriteString("\n")
	}

	if researchSection := resolvePhaseResearchSection(root, phase.ID); researchSection != "" {
		b.WriteString("\n")
		b.WriteString(researchSection)
		b.WriteString("\n")
	}

	// A smaller share than planning gets: the task must stay the bulk of a
	// build brief (TestBuildWorkerBriefIsMostlyTask).
	if colonyResearch := resolveColonyResearchSection(root, loadColonyResearchDocs(root), colonyResearchBuildBudgetChars); colonyResearch != "" {
		b.WriteString("\n")
		b.WriteString(colonyResearch)
		b.WriteString("\n")
	}

	b.WriteString("\n## Expected Output\n\n")
	b.WriteString("- ")
	b.WriteString(expectedDispatchOutcome(dispatch))
	b.WriteString("\n")

	b.WriteString(renderVerificationCommandSection())
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
	_ = os.RemoveAll(filepath.Join(buildDir, "worker-reports"))
	cleanupStaleWorkerBriefs(phaseNum)
}

// cleanupStaleWorkerBriefs removes a phase's worker-brief directory so the
// next manifest writes into an empty one.
//
// Split out of cleanupStaleBuildAttemptArtifacts for the plan-only path
// (190-190/WR-01), which must NOT do what the rest of that function does:
// clearing verification.json / gates.json / continue.json / review.json would
// destroy a previous attempt's evidence, and clearing worker-reports would
// destroy real worker output — neither is this planning step's to discard.
//
// Without this, repeated `--plan-only` runs for the same phase accumulated
// dead files forever. Brief filenames are "{dispatch.Name}.md", and dispatch
// names hash phase:task-index:task-goal-text, so any edit to a task's wording,
// any --selected-tasks filter, or any different caste/coalescing decision
// produced a NEW name and orphaned the old file. Re-running --plan-only
// without --force is an ordinary supported flow (see the idle-plan-only
// auto-supersede above, which exists precisely so it does not need --force),
// and nothing else ever cleared this directory: clearActiveColonyRuntimeFiles
// (entomb/abandon) and `aether init`'s sweep both leave .aether/data/build/
// alone.
//
// Safe at the plan-only call site because any prior attempt has already been
// superseded or interrupted by the time it runs — the manifest that referenced
// those brief files is dead.
//
// Locked by TestPlanOnlyRerunDoesNotAccumulateStaleWorkerBriefs.
func cleanupStaleWorkerBriefs(phaseNum int) {
	if store == nil || phaseNum < 1 {
		return
	}
	_ = os.RemoveAll(filepath.Join(store.BasePath(), "build", fmt.Sprintf("phase-%d", phaseNum), "worker-briefs"))
}

// coveredDispatchTask pairs a merged dispatch's covered task ID with its
// resolved colony.Task (nil when the ID has no matching phase.Tasks entry)
// and its 1-based POSITION within the original covered-ID chain --
// dispatchCoveredTaskIDs's own order, stable across a stale or missing ID.
// WR-01 (189-REVIEW.md): a filtered slice's own index used to double as the
// "Task N:" label, so dropping one unresolved ID silently shifted every
// later task's label down by one, misattributing its content. Carrying
// Position separately makes that shift impossible.
type coveredDispatchTask struct {
	Position int
	ID       string
	Task     *colony.Task
}

// findDispatchTasks resolves every task a dispatch covers (via the
// already-merge-aware dispatchCoveredTaskIDs) against phase.Tasks, in covered
// order. It does not reimplement "which task IDs does this dispatch cover" --
// that is dispatchCoveredTaskIDs's job, also used by completedBuildTaskIDs.
//
// A covered ID with no matching phase.Tasks entry is NEVER dropped from the
// returned slice -- doing so used to corrupt every later task's "Task N:"
// label (WR-01, 189-REVIEW.md). Its slot is kept with Task == nil and
// Position set to its original 1-based place in the covered-ID chain, so
// callers can both render subsequent tasks under their correct number
// (renderDispatchTaskItemsSection) and visibly flag the gap instead of
// silently vanishing it (renderUnresolvedDispatchTaskNotice). A stale or
// renamed task ID must never panic or drop the rest of the brief.
func findDispatchTasks(phase colony.Phase, dispatch codexBuildDispatch) []coveredDispatchTask {
	coveredIDs := dispatchCoveredTaskIDs(dispatch)
	if len(coveredIDs) == 0 {
		return nil
	}
	byID := make(map[string]*colony.Task, len(phase.Tasks))
	for i := range phase.Tasks {
		byID[buildTaskID(phase.Tasks[i], i)] = &phase.Tasks[i]
	}
	tasks := make([]coveredDispatchTask, 0, len(coveredIDs))
	for i, id := range coveredIDs {
		tasks = append(tasks, coveredDispatchTask{Position: i + 1, ID: id, Task: byID[id]})
	}
	return tasks
}

// renderDispatchTaskItemsSection appends a "## <heading>" section gathering
// extract's items across every task in tasks. An entry whose Task could not
// be resolved (Task == nil) contributes no items here -- it renders no
// content in any of the three sections this function backs, and
// renderUnresolvedDispatchTaskNotice is what visibly flags it, once, rather
// than three times.
//
// Exactly one covered task renders byte-identical to the pre-merge-aware
// brief: the heading appears whenever that task's raw item slice is
// non-empty (matching the original single-task code's own behavior, even in
// the edge case where every item trims to empty), with no "Task N:" label --
// just that task's bullets.
//
// More than one covered task labels each task's block "**Task N:**" (N =
// the task's 1-based POSITION within the original covered-ID chain --
// dispatchCoveredTaskIDs's own order, matching mergeDispatchInto's own
// numbering of dispatch.Task -- NOT the index into this function's own
// possibly-gapped tasks slice; conflating the two is the exact numbering
// desync WR-01 fixed, where a missing task shifted every later task's label
// down by one), including only tasks with at least one non-empty item for
// this section; a task with none (resolved or not) is skipped entirely
// here, so no empty "Task N:" label with nothing under it is ever emitted.
func renderDispatchTaskItemsSection(b *strings.Builder, heading string, tasks []coveredDispatchTask, extract func(*colony.Task) []string) {
	if len(tasks) == 0 {
		return
	}

	if len(tasks) == 1 {
		if tasks[0].Task == nil {
			return
		}
		items := extract(tasks[0].Task)
		if len(items) == 0 {
			return
		}
		b.WriteString("\n## ")
		b.WriteString(heading)
		b.WriteString("\n\n")
		for _, item := range items {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			b.WriteString("- ")
			b.WriteString(item)
			b.WriteString("\n")
		}
		return
	}

	var body strings.Builder
	any := false
	for _, task := range tasks {
		if task.Task == nil {
			continue
		}
		var written []string
		for _, item := range extract(task.Task) {
			item = strings.TrimSpace(item)
			if item != "" {
				written = append(written, item)
			}
		}
		if len(written) == 0 {
			continue
		}
		any = true
		body.WriteString(fmt.Sprintf("**Task %d:**\n", task.Position))
		for _, item := range written {
			body.WriteString("- ")
			body.WriteString(item)
			body.WriteString("\n")
		}
	}
	if !any {
		return
	}
	b.WriteString("\n## ")
	b.WriteString(heading)
	b.WriteString("\n\n")
	b.WriteString(body.String())
}

// renderUnresolvedDispatchTaskNotice returns a "## Task Resolution Notice"
// section listing every covered task ID findDispatchTasks could not resolve
// against phase.Tasks, or "" when every covered task resolved (the
// overwhelming common case, keeping ordinary briefs byte-identical to
// before this fix). WR-01 (189-REVIEW.md): a stale or renamed task ID used
// to vanish from the brief with no trace at all -- a worker had no way to
// tell "this task has no constraints" apart from "this task's constraints
// could not be found." Silence there is how a worker ends up judged on a
// task it never saw.
func renderUnresolvedDispatchTaskNotice(tasks []coveredDispatchTask) string {
	var unresolved []coveredDispatchTask
	for _, task := range tasks {
		if task.Task == nil {
			unresolved = append(unresolved, task)
		}
	}
	if len(unresolved) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n## Task Resolution Notice\n\n")
	for _, task := range unresolved {
		b.WriteString(fmt.Sprintf(
			"- Task %d (id %q) could not be resolved against this phase's task list. Its constraints, hints, and success criteria are NOT included above -- treat them as unknown, not absent, and check the phase plan directly before treating this task as unconstrained.\n",
			task.Position, task.ID,
		))
	}
	return b.String()
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

// ensureUniqueBuildDispatchNames allocates a collision-free worker name for
// each dispatch. Names are checked against the full spawn-tree history so a
// name reused from a genuinely different phase, or from a previously sealed
// (finalized, i.e. successfully built) attempt of this same phase, still gets
// a `-rN` suffix (D-09 spoofing guard, T-163.1-20).
//
// Spawn-tree entries carry no phase context (SpawnEntry has no phase field),
// so re-planning the CURRENT phase's still-open attempt would otherwise see
// its own prior names as "used" and rename every worker on every re-plan. To
// keep names stable across re-plans, phaseNum's latest attempt record
// (buildAttemptRecord.Dispatches) is consulted directly and its names are
// excluded from the collision set, UNLESS that attempt already reached
// buildAttemptBuilt -- the one status meaning workers actually ran and
// produced attributable results under those names.
//
// Note this is intentionally not gated on buildAttemptStatusActive: whenever
// this function actually runs, a same-phase prior attempt that WAS active has
// already been forced through interruptLatestBuildAttempt by the caller (see
// runCodexBuildPlanOnlyWithOptions / runCodexBuildWithOptions), so by the
// time name allocation happens its status is always terminal (built, failed,
// or interrupted) or nonexistent -- never one of the "active" enum values.
func ensureUniqueBuildDispatchNames(dispatches []codexBuildDispatch, phaseNum int) ([]codexBuildDispatch, error) {
	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	entries, err := spawnTree.Parse()
	if err != nil {
		return nil, fmt.Errorf("failed to read spawn tree for name allocation: %w", err)
	}

	used := make(map[string]bool, len(entries)+len(dispatches))
	for _, entry := range entries {
		used[entry.AgentName] = true
	}

	if phaseNum > 0 {
		if _, record, ok := loadLatestBuildAttempt(phaseNum); ok && record.Status != buildAttemptBuilt {
			for _, dispatch := range record.Dispatches {
				delete(used, dispatch.Name)
			}
		}
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

func attachBuildDispatchContext(root string, phase colony.Phase, dispatches []codexBuildDispatch, startedAt time.Time) {
	for i := range dispatches {
		// The wrapper spawns each dispatch with agent_name as the subagent
		// type; the TS host used to enrich this and the direct plan-only
		// path must carry it too.
		dispatches[i].AgentName = codexAgentNameForCaste(dispatches[i].Caste)
		dispatches[i].Model = resolveCasteModel(dispatches[i].Caste)
		dispatches[i].PermissionProfile = codex.PermissionProfileForCaste(dispatches[i].Caste)
		assignment := resolveWorkerSkillAssignmentForWorkflow("build", dispatches[i].Caste, dispatches[i].Task)
		dispatches[i].SkillSection = assignment.Section
		dispatches[i].SkillCount = assignment.SkillCount
		dispatches[i].ColonySkills = assignment.ColonyCount
		dispatches[i].DomainSkills = assignment.DomainCount
		dispatches[i].MatchedSkills = append([]string{}, assignment.MatchedNames...)
		// HandoffSection stays EMPTY on this path (190-190/CR-01). The
		// manifest-level capsule (resolveCodexWorkerContext(), which renders
		// "## Previous Worker Handoffs" at cmd/colony_prime_context.go:695)
		// is the sole channel for prior-worker handoffs on every caller of
		// attachBuildDispatchContext, and .claude/commands/ant/build.md says
		// so to the wrapper in as many words: the capsule "is the SOLE source
		// of pheromone signals and prior worker handoffs".
		//
		// Populating this field anyway shipped that same content a second
		// time in result.dispatches[] AND (via the json:"handoff_section"
		// tag on the typed manifest) in result.dispatch_manifest.dispatches[],
		// contradicting 190-03's own "one home each" invariant, doubling the
		// response payload, and leaving a trap for any wrapper that reads
		// "here is this dispatch's handoff context" as an instruction to use
		// it. Gating the brief's copy was not enough; this adjacent field
		// carried the duplicate.
		//
		// The direct/native dispatch path is unaffected: it never calls this
		// function and sets codex.WorkerDispatch.HandoffSection itself.
		//
		// Locked by TestPlanOnlyDispatchesCarryNoHandoffSection.
		dispatches[i].HandoffSection = ""
		// Brief is composed with includeSteeringSections=false, so it does
		// not embed HandoffSection either (see below).
		//
		// includeSteeringSections=false (190-03, closing D-190-01-A): every
		// caller of attachBuildDispatchContext also carries or inspects the
		// manifest-level capsule alongside this brief -- the plan-only wrapper
		// flow (runCodexBuildPlanOnlyWithOptions, which sets
		// manifest.ContextCapsule from resolveCodexWorkerContext()) and the
		// --print-brief inspector (printWorkerBriefs, which explicitly
		// resolves and prepends the same capsule to simulate that exact
		// flow). The capsule already renders "## Pheromone Signals" (D-190-01-A;
		// cmd/colony_prime_context.go:571) and "## Previous Worker Handoffs"
		// (cmd/colony_prime_context.go:695) unconditionally whenever either is
		// active, so composing them again here would ship the same content
		// twice to a wrapper-spawned worker. This is the ONLY caller that
		// composes Brief ahead of a capsule prepend -- writeBuildWorkerBriefFiles's
		// own fallback composition (used only when no capsule accompanies the
		// artifact) keeps the old self-contained behavior.
		dispatches[i].Brief = composeBuildManifestBrief(root, phase, dispatches[i], startedAt, false)
	}
}

// composeBuildManifestBrief is the single source of the worker prompt that
// ships in the plan-only manifest and (as a self-contained artifact) the
// direct-dispatch manifest.json. It is the base task brief plus the handoff
// schema sentence (codex.HandoffFieldsSummary) -- always stated, since no
// other channel states the OUTPUT contract this composer's callers need --
// plus, when includeSteeringSections is true, the steering sections the
// caller has no other channel for: pheromone signals and prior worker
// handoffs (the INPUT context, distinct from the schema sentence above).
//
// includeSteeringSections distinguishes composeBuildManifestBrief's two
// still-valid callers (190-03, D-190-01-A): attachBuildDispatchContext (the
// wrapper plan-only flow and its --print-brief simulation) passes false,
// because those callers also carry the manifest-level capsule
// (resolveCodexWorkerContext()) alongside this brief, and that capsule
// already renders both "## Pheromone Signals" and "## Previous Worker
// Handoffs" whenever either is active -- composing them again here would
// duplicate what colony-prime already delivers (CLAUDE.md documents
// pheromone signals as colony-prime-injected and the highest-retention-priority
// section in its trim order; capsule is the established, multiply-consumed
// channel, matching the same "one canonical source" reasoning Phase 190's own
// criterion 3 already applied to hive wisdom). writeBuildWorkerBriefFiles's
// own fallback composition (used only for the direct/native
// `aether build <phase>` path, whose manifest.json carries no ContextCapsule
// at all -- see codexBuildManifest.ContextCapsule's doc comment) passes true,
// because that artifact has no accompanying capsule in the same JSON envelope
// and must stay self-contained (TestWorkerBriefFileHoldsComposedBrief pins
// this).
//
// The Go subprocess path (executeCodexBuildDispatches) deliberately does NOT
// use this composition at all — it delivers PheromoneSection and
// HandoffSection separately through WorkerConfig and pkg/codex/prompt.go, and
// states the handoff schema itself via renderResponseContract on that same
// separate channel, so embedding any of this in the shared renderer would
// duplicate it there regardless of includeSteeringSections. --print-brief
// uses this composer (via attachBuildDispatchContext) so what the user
// inspects is exactly what the manifest carries.
func composeBuildManifestBrief(root string, phase colony.Phase, dispatch codexBuildDispatch, startedAt time.Time, includeSteeringSections bool) string {
	var b strings.Builder
	b.WriteString(renderCodexBuildWorkerBrief(root, phase, dispatch, startedAt))
	b.WriteString(fmt.Sprintf("\nYour final result's handoff object must include %s. An empty handoff is rejected. %s\n", codex.HandoffFieldsSummary, codex.HandoffOpenDecisionsGuidance))

	if includeSteeringSections {
		if pheromoneSection := resolvePheromoneSection(); pheromoneSection != "" {
			// The resolver emits its own "### Active Pheromone Signals" heading;
			// rewrap under the "## Pheromone Signals" heading every caste agent
			// definition's <pheromone_protocol> block is written against.
			content := strings.TrimSpace(strings.TrimPrefix(pheromoneSection, "### Active Pheromone Signals"))
			if content != "" {
				b.WriteString("\n## Pheromone Signals\n\n")
				b.WriteString(content)
				b.WriteString("\n")
			}
		}
	}

	// Verifying castes need to know what the tree already looked like: phases
	// do not commit between themselves, so prior phases' verified work shows
	// up as uncommitted changes and reads as this phase overreaching. Not a
	// steering section in the pheromone/handoff sense -- the capsule carries
	// neither a phase-baseline concept nor a per-caste gate, so this always
	// renders regardless of includeSteeringSections.
	if baselineAwareCastes[strings.ToLower(strings.TrimSpace(dispatch.Caste))] {
		if section := renderPhaseBaselineSection(capturePhaseBaseline(root)); section != "" {
			b.WriteString(section)
		}
	}

	if includeSteeringSections {
		if handoff := strings.TrimSpace(dispatch.HandoffSection); handoff != "" {
			b.WriteString("\n")
			b.WriteString(handoff)
			b.WriteString("\n")
		}
	}

	return b.String()
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
//
// Calling contract (D-190-03-A / 190-05): this renders the SAME content
// resolveCodexWorkerContext()'s capsule already includes under its own
// "## Pheromone Signals" heading (cmd/colony_prime_context.go:571) whenever a
// signal is active. A caller that sets codex.WorkerDispatch/WorkerConfig's
// ContextCapsule to resolveCodexWorkerContext() must NOT also populate
// PheromoneSection from this function -- AssemblePrompt/AssembleHostedPrompt
// (pkg/codex/prompt.go) join ContextCapsule and PheromoneSection as two
// separate, both-included parts with no dedup between them, so doing both
// ships the same steering text twice under two different headings. This is
// why executeCodexBuildDispatches (build, native/direct), plannedContinueReviewDispatches
// and plannedContinueWatcherDispatch (continue), dispatchRealPlanningWorkersWithIterationContext
// (plan), dispatchRealSurveyorsWithTimeout (colonize), plannedSealFinalReviewDispatches
// (seal), and invokeSwarmWorker (swarm) do not call this function.
//
// Call this ONLY when the caller's ContextCapsule is a custom, lighter capsule
// that does not itself render "## Pheromone Signals" -- today that means
// runQuickScout (quick, renderQuickContextCapsule) and the Oracle worker
// config builder (oracle, renderOracleContextCapsule), where this function is
// the pheromone signals' sole delivery channel. Removing it from those two
// callers would silently drop pheromone steering to zero, not deduplicate it.
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
