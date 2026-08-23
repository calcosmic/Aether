package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

type codexContinueExternalDispatch struct {
	Stage           string               `json:"stage"`
	Wave            int                  `json:"wave"`
	Caste           string               `json:"caste"`
	AgentName       string               `json:"agent_name,omitempty"`
	Name            string               `json:"name"`
	Task            string               `json:"task"`
	TaskID          string               `json:"task_id"`
	Timeout         int                  `json:"timeout_seconds,omitempty"`
	Status          string               `json:"status"`
	Summary         string               `json:"summary,omitempty"`
	Blockers        []string             `json:"blockers,omitempty"`
	Duration        float64              `json:"duration,omitempty"`
	Report          string               `json:"report,omitempty"`
	Findings        []codexReviewFinding `json:"findings,omitempty"`
	Issues          []codexReviewFinding `json:"issues,omitempty"`
	Recommendations []string             `json:"recommendations,omitempty"`
	WeakSpots       []string             `json:"weak_spots,omitempty"`
	EdgeCases       []string             `json:"edge_cases_discovered,omitempty"`
	ReusableLessons []string             `json:"reusable_lessons,omitempty"`
	Brief           string               `json:"brief,omitempty"`
	SkillSection    string               `json:"skill_section,omitempty"`
	SkillCount      int                  `json:"skill_count,omitempty"`
	ColonySkills    int                  `json:"colony_skill_count,omitempty"`
	DomainSkills    int                  `json:"domain_skill_count,omitempty"`
	MatchedSkills   []string             `json:"matched_skills,omitempty"`
	Handoff         codex.WorkerHandoff  `json:"handoff,omitempty"`
}

type codexContinuePlanManifest struct {
	Phase               int                             `json:"phase"`
	PhaseName           string                          `json:"phase_name"`
	Root                string                          `json:"root"`
	GeneratedAt         string                          `json:"generated_at"`
	ColonyMode          string                          `json:"colony_mode,omitempty"`
	BuildManifest       string                          `json:"build_manifest,omitempty"`
	Verification        codexContinueVerificationReport `json:"verification"`
	Assessment          codexContinueAssessment         `json:"assessment"`
	ReconcileTaskIDs    []string                        `json:"reconcile_task_ids,omitempty"`
	ReadOnlyArtifacts   []string                        `json:"read_only_artifacts,omitempty"`
	WorkerTimeout       int                             `json:"worker_timeout_seconds,omitempty"`
	VerificationTimeout int                             `json:"verification_timeout_seconds,omitempty"`
	SkipWatchers        bool                            `json:"skip_watchers,omitempty"`
	// ContextCapsule is colony-wide, resolved ONCE per plan-only manifest --
	// mirroring codexBuildManifest.ContextCapsule (cmd/codex_build.go) --
	// never copied per-dispatch. The wrapper reads it once from the manifest
	// and prepends it, verbatim, ahead of every spawned reviewer/watcher's
	// brief this continue run carries. It is the SOLE carrier of pheromone
	// signals on this flow (190-VERIFICATION third pass): PheromoneSection
	// below stays unset -- populating it again shipped every active signal
	// to heavy-depth reviewers twice, once per field. The field itself is
	// kept (omitempty, so it vanishes from the JSON) for wire compatibility,
	// exactly as 190-04 kept handoff_section on the build manifest.
	ContextCapsule            string                          `json:"context_capsule,omitempty"`
	PheromoneSection          string                          `json:"pheromone_section,omitempty"`
	Dispatches                []codexContinueExternalDispatch `json:"dispatches"`
	DispatchMode              string                          `json:"dispatch_mode"`
	FinalizeSurface           string                          `json:"finalize_surface"`
	RequiresFinalizer         bool                            `json:"requires_finalizer"`
	ReviewDepth               string                          `json:"review_depth,omitempty"`
	BoundaryQuestions         []discussQuestion               `json:"boundary_questions,omitempty"`
	BoundaryQuestionCount     int                             `json:"boundary_question_count,omitempty"`
	BoundaryQuestionsCreated  int                             `json:"boundary_questions_created,omitempty"`
	BoundaryQuestionsExisting int                             `json:"boundary_questions_existing,omitempty"`
	OrchestratorGuidance      *orchestratorBoundaryGuidance   `json:"orchestrator_boundary_guidance,omitempty"`
}

func runCodexContinuePlanOnly(root string, options codexContinueOptions) (map[string]interface{}, colony.ColonyState, colony.Phase, []codexContinueExternalDispatch, error) {
	if store == nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("no store initialized")
	}

	state, err := loadActiveColonyState()
	if err != nil {
		return nil, state, colony.Phase{}, nil, fmt.Errorf("%s", colonyStateLoadMessage(err))
	}
	if len(state.Plan.Phases) == 0 {
		return nil, state, colony.Phase{}, nil, fmt.Errorf("No project plan. Run `aether plan` first.")
	}
	if state.State != colony.StateEXECUTING && state.State != colony.StateBUILT {
		return nil, state, colony.Phase{}, nil, fmt.Errorf("No active phase to continue. Run `aether build <phase>` first.")
	}
	if state.CurrentPhase < 1 || state.CurrentPhase > len(state.Plan.Phases) {
		return nil, state, colony.Phase{}, nil, fmt.Errorf("No active phase to continue. Run `aether build <phase>` first.")
	}

	phase := state.Plan.Phases[state.CurrentPhase-1]
	if phase.Status != colony.PhaseInProgress {
		return nil, state, colony.Phase{}, nil, fmt.Errorf("phase %d is not in progress; run `aether build %d` first", phase.ID, phase.ID)
	}
	if err := validateContinueReconcileTasks(phase, options.ReconcileTaskIDs); err != nil {
		return nil, state, phase, nil, err
	}
	if err := validateReadOnlyArtifacts(phase, options.ReconcileTaskIDs, options.ReadOnlyArtifacts); err != nil {
		return nil, state, phase, nil, err
	}

	manifest := loadCodexContinueManifest(phase.ID)
	if state.BuildStartedAt == nil && !manifest.Present {
		return nil, state, phase, nil, fmt.Errorf("No active build packet found. Run `aether build <phase>` first.")
	}
	if abandoned, _, summary := detectAbandonedBuild(manifest, state); abandoned {
		return nil, state, phase, nil, fmt.Errorf("%s", summary)
	}

	// D-01 escape hatch: record hash-verified read-only evidence for the
	// reconcile task IDs' declared artifacts BEFORE the verification snapshot
	// (and its embedded criterion evidence evaluation) runs below, so
	// evaluatePhaseCriterionEvidence sees the recorded evidence on this same
	// plan-only invocation. Mirrors the direct path's ordering
	// (cmd/codex_continue.go:577-584). Validation above already confirmed
	// every spec's task ID both exists in the phase and was also passed to
	// --reconcile-task.
	if len(options.ReadOnlyArtifacts) > 0 {
		if err := applyReadOnlyArtifactEvidence(root, manifest, options.ReadOnlyArtifacts); err != nil {
			return nil, state, phase, nil, err
		}
	}

	now := time.Now().UTC()
	verificationTimeout := effectiveContinueVerificationTimeout(options.VerificationTimeout)
	verification := runCodexContinueVerificationSnapshot(root, phase, manifest, now, verificationTimeout, options.SkipWatchers)
	assessment := assessCodexContinue(phase, manifest, verification, options, now)
	verification = attachContinueClaimVerification(verification, assessment)
	reviewDepth := resolveEffectiveContinueDepth(phase, len(state.Plan.Phases), options.LightFlag, options.HeavyFlag, options.VerificationDepth, state.VerificationDepth)
	effectiveSkipWatchers := options.SkipWatchers || (verification.ChecksPassed && isEnvironmentBlockedWatcher(verification.Watcher))

	// Phase 97: Queen decision layer for plan-only gate evaluation (D-09, D-11)
	priorGateResults, _ := gateResultsReadPhase(phase.ID)
	if priorGateResults == nil {
		priorGateResults = []GateCheckResult{}
	}
	planGates := runCodexContinueGates(phase, manifest, verification, assessment, now, priorGateResults)
	budget := budgetFromRecoveryLog(phase.ID, 1)
	if budget == nil {
		budget = newRecoveryBudget(1)
	}
	queenDecisions := queenDecide(planGates, budget, circuitBreaker, phase.ID, string(reviewDepth))

	mergedExternalQueenCastes, externalQueenCasteWhyReasons := parseAndMergeCasteWhy(options.QueenCastes, options.QueenCasteWhy)
	dispatches := plannedExternalContinueDispatches(root, phase, manifest, verification, assessment, options.WorkerTimeout, reviewDepth, effectiveSkipWatchers, mergedExternalQueenCastes, options.QueenCasteReason, externalQueenCasteWhyReasons)
	plan := codexContinuePlanManifest{
		Phase:               phase.ID,
		PhaseName:           phase.Name,
		Root:                root,
		GeneratedAt:         now.Format(time.RFC3339),
		ColonyMode:          string(state.EffectiveColonyMode()),
		BuildManifest:       displayOptionalDataPath(manifest.Path),
		Verification:        verification,
		Assessment:          assessment,
		ReconcileTaskIDs:    append([]string{}, options.ReconcileTaskIDs...),
		ReadOnlyArtifacts:   append([]string{}, options.ReadOnlyArtifacts...),
		WorkerTimeout:       int(effectiveContinueReviewTimeout(options.WorkerTimeout) / time.Second),
		VerificationTimeout: int(verificationTimeout / time.Second),
		SkipWatchers:        effectiveSkipWatchers,
		// ContextCapsule is the SOLE carrier of pheromone signals on this
		// flow (190-VERIFICATION third pass) — resolveCodexWorkerContext()
		// already renders "## Pheromone Signals" for every active signal,
		// and continue.md used to instruct the wrapper to concatenate a
		// separate PheromoneSection verbatim as well, delivering each
		// signal to every heavy-depth reviewer twice. Mirrors build's
		// 190-03 decision; PheromoneSection deliberately left unset.
		ContextCapsule:    resolveCodexWorkerContext(),
		Dispatches:        dispatches,
		DispatchMode:      "plan-only",
		FinalizeSurface:   "awaiting_wrapper_completion",
		RequiresFinalizer: true,
		ReviewDepth:       string(reviewDepth),
	}
	boundary, err := materializeOrchestratorBoundaryQuestions("continue", state, phase, continueBoundaryQuestionCandidates(phase, verification, assessment))
	if err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}
	plan.BoundaryQuestions = boundary.Questions
	plan.BoundaryQuestionCount = len(boundary.Questions)
	plan.BoundaryQuestionsCreated = boundary.Created
	plan.BoundaryQuestionsExisting = boundary.Existing

	result := map[string]interface{}{
		"plan_only":                    true,
		"phase":                        phase.ID,
		"phase_name":                   phase.Name,
		"colony_mode":                  string(state.EffectiveColonyMode()),
		"state":                        state.State,
		"continue_manifest":            plan,
		"verification":                 verification,
		"assessment":                   assessment,
		"dispatches":                   dispatches,
		"dispatch_count":               len(dispatches),
		"wave_count":                   countContinueExternalWaves(dispatches),
		"dispatch_mode":                "plan-only",
		"next":                         "spawn wrapper continue agents, then record completion",
		"review_depth":                 string(reviewDepth),
		"skip_watchers":                effectiveSkipWatchers,
		"verification_timeout_seconds": int(verificationTimeout / time.Second),
		"queen_decisions":              queenDecisions,
		"queen_state_file":             fmt.Sprintf("queen-state-%d.json", phase.ID),
		"queen_state_persisted":        false,
		"queen_state_persisted_by":     "continue-finalize",
		"wrapper_contract": map[string]interface{}{
			"source_command":               continuePlanOnlySourceCommand(reviewDepth, effectiveSkipWatchers),
			"spawn_log_required":           true,
			"spawn_complete_required":      true,
			"worker_timeout_seconds":       int(effectiveContinueReviewTimeout(options.WorkerTimeout) / time.Second),
			"verification_timeout_seconds": int(verificationTimeout / time.Second),
			"finalize_surface":             "awaiting_wrapper_completion",
			"runtime_verification_only":    true,
			"result_artifact_paths":        []string{finalizerCompletionTempPattern},
			"result_collection_policy":     "A structurally valid completed result wins over a timeout placeholder for the same reviewer; malformed JSON, duplicate terminal results, and .aether/data completion files are rejected.",
		},
	}
	addBoundaryQuestionResultFields(result, boundary)
	if guidance, ok := addOrchestratorBoundaryGuidance(result, "continue", state, "aether continue", boundary.Questions); ok {
		plan.OrchestratorGuidance = &guidance
		result["continue_manifest"] = plan
	}
	return result, state, phase, dispatches, nil
}

func continuePlanOnlySourceCommand(reviewDepth colony.VerificationDepth, skipWatchers bool) string {
	parts := []string{
		"aether",
		"host",
		"continue",
		"--verification-depth",
		string(colony.NormalizeVerificationDepth(string(reviewDepth))),
	}
	if skipWatchers {
		parts = append(parts, "--skip-watchers")
	}
	parts = append(parts, "$ARGUMENTS")
	return strings.Join(parts, " ")
}

// runCodexContinueVerificationSnapshot serves BOTH runCodexContinuePlanOnly
// (codex_continue_plan.go:110) and runCodexContinueFinalize
// (codex_continue_finalize.go:175) -- the two paths an external wrapper
// actually drives. It shares runDeterministicFloor with the in-process lane
// (codex_continue.go's runCodexContinueVerification), so the shell steps,
// claims verification, and criterion evidence evaluation -- including the
// executed-check counting, the environment-issue warning downgrade, and the
// zero-executed-checks plain-English warning -- are structurally identical on
// both lanes rather than a discipline to keep in sync by hand
// (TestBothContinueLanesApplyTheSameFloor). Criterion evidence evaluation
// used to be entirely absent here, which made the criterion gate -- and
// therefore the --read-only-artifact escape hatch (readonly_evidence.go) --
// dead on the external-review path: only the direct `aether continue` path
// ever called evaluatePhaseCriterionEvidence. That gap is now closed by
// sharing the floor.
func runCodexContinueVerificationSnapshot(root string, phase colony.Phase, manifest codexContinueManifest, now time.Time, verificationTimeout time.Duration, skipWatchers bool) codexContinueVerificationReport {
	watcher := evaluateContinueWatcherVerification(manifest)
	if skipWatchers {
		watcher = codexWatcherVerification{Present: true, Passed: true, Status: "skipped", Worker: "skip-watchers", Summary: "watcher skipped; relying on runtime-owned verification commands"}
	}

	floor := runDeterministicFloor(context.Background(), root, phase, manifest, watcher, verificationTimeout)

	checksPassed := floor.ChecksPassed
	blockers := append([]string{}, floor.BlockingIssues...)
	// A watcher that was never dispatched (Present:false) or whose status is
	// "skipped" must not contribute a block here -- only a dispatched
	// watcher that did not pass does. This corrects the asymmetry the
	// in-process lane never had: `watcher.Present && !watcher.Passed` alone
	// treated a skipped-status watcher the same as a genuinely failed one,
	// because isSuccessfulExternalBuildStatus("skipped") is false, so
	// Passed is false for a skip too.
	if watcher.Present && !watcher.Passed && !strings.EqualFold(strings.TrimSpace(watcher.Status), "skipped") {
		checksPassed = false
		summary := strings.TrimSpace(watcher.Summary)
		if summary == "" {
			summary = "build watcher verification did not complete cleanly"
		}
		blockers = append(blockers, summary)
	}

	return codexContinueVerificationReport{
		Phase:                      phase.ID,
		GeneratedAt:                now.Format(time.RFC3339),
		VerificationTimeoutSeconds: int(effectiveContinueVerificationTimeout(verificationTimeout) / time.Second),
		Steps:                      floor.Steps,
		Claims:                     floor.Claims,
		Watcher:                    watcher,
		CriteriaPolicy:             floor.Criteria.Policy,
		CriteriaEnforced:           floor.Criteria.Enforced,
		CriteriaPassed:             floor.Criteria.Passed,
		Criteria:                   floor.Criteria.Criteria,
		ChecksPassed:               checksPassed,
		Passed:                     checksPassed,
		BlockingIssues:             blockers,
		Warnings:                   floor.Warnings,
	}
}

// continueExternalBriefWithHandoffSchema appends the handoff/return schema
// note to a wrapper-external continue brief (watcher or reviewer), mirroring
// composeBuildManifestBrief's identical append (cmd/codex_build.go) for
// build's wrapper-external brief. Continue's native-Codex dispatch path
// (plannedContinueReviewDispatches, plannedContinueWatcherDispatch,
// cmd/codex_continue.go) already states this schema via a SEPARATE channel
// (AssembleHostedPrompt + renderResponseContract) -- appending it here,
// rather than inside renderCodexContinueReviewBrief/renderCodexContinueWatcherBrief
// themselves (shared by both paths), is what keeps a native-Codex continue
// worker from seeing it twice (D-06).
func continueExternalBriefWithHandoffSchema(rendered string) string {
	return rendered + fmt.Sprintf("\nYour final result's handoff object must include %s. An empty handoff is rejected. %s\n", codex.HandoffFieldsSummary, codex.HandoffOpenDecisionsGuidance)
}

func plannedExternalContinueDispatches(root string, phase colony.Phase, manifest codexContinueManifest, verification codexContinueVerificationReport, assessment codexContinueAssessment, workerTimeout time.Duration, reviewDepth colony.VerificationDepth, skipWatchers bool, queenCastes []string, queenCasteReason string, reasons ...map[string]string) []codexContinueExternalDispatch {
	timeoutSeconds := int(effectiveContinueReviewTimeout(workerTimeout) / time.Second)
	dispatches := []codexContinueExternalDispatch{}
	// The external/wrapper continue lane relays the ALREADY-COMPUTED
	// deterministic verification report (build/type/lint/test) to a worker,
	// because the wrapper itself cannot run those checks. This relay used to
	// be gated on queenContinueHasCaste(queenDispatches, "watcher") in
	// addition to !skipWatchers -- a redundant check while watcher was
	// unconditionally in queenContinueDispatches, but plan 194-05 (D-13)
	// removed that unconditional membership, which would have silently
	// dropped the relay at light/standard depth despite the deterministic
	// floor itself still running. A depth flag must never be able to remove
	// a program check (CLAUDE.md); skipWatchers alone is the correct, sole
	// gate here -- an explicit owner choice, not a caste-selection side
	// effect.
	if !skipWatchers {
		watcherSkillAssignment := resolveWorkerSkillAssignmentForWorkflow("continue", "watcher", "Independent verification before advancement")
		dispatches = append(dispatches, codexContinueExternalDispatch{
			Stage:         "verification",
			Wave:          1,
			Caste:         "watcher",
			AgentName:     codexAgentNameForCaste("watcher"),
			Name:          deterministicAntName("watcher", fmt.Sprintf("phase:%d:continue:watcher", phase.ID)),
			Task:          "Independent verification before advancement",
			TaskID:        fmt.Sprintf("continue-verification-%d", phase.ID),
			Timeout:       timeoutSeconds,
			Status:        "planned",
			Brief:         continueExternalBriefWithHandoffSchema(renderCodexContinueWatcherBrief(root, phase, manifest, verification.Steps, verification.Claims, verification.Watcher, workerTimeout)),
			SkillSection:  watcherSkillAssignment.Section,
			SkillCount:    watcherSkillAssignment.SkillCount,
			ColonySkills:  watcherSkillAssignment.ColonyCount,
			DomainSkills:  watcherSkillAssignment.DomainCount,
			MatchedSkills: append([]string{}, watcherSkillAssignment.MatchedNames...),
		})
	}
	// changedFiles feeds the D-02 file-detected forced-reviewer union — the
	// same call, the same input source (phaseChangedFilesForRiskSignals,
	// WR-01 — unions the builder's own self-report with an independent
	// `git diff`), as the in-process lane's plannedContinueReviewDispatches
	// (cmd/codex_continue.go) uses, so the two lanes can never disagree for
	// the same phase.
	changedFiles := phaseChangedFilesForRiskSignals(phase.ID)
	reviewSpecs := queenContinueReviewSpecsWithJudgement(phase, reviewDepth, queenCastes, queenCasteReason, manifest.Data.ForcedReviewers, changedFiles, reasons...)
	reviewWave := 2
	if skipWatchers {
		reviewWave = 1
	}
	for _, spec := range reviewSpecs {
		assignment := resolveWorkerSkillAssignmentForWorkflow("continue", spec.Caste, spec.Task)
		dispatches = append(dispatches, codexContinueExternalDispatch{
			Stage:         "review",
			Wave:          reviewWave,
			Caste:         spec.Caste,
			AgentName:     codexAgentNameForCaste(spec.Caste),
			Name:          deterministicAntName(spec.Caste, fmt.Sprintf("phase:%d:continue:%s", phase.ID, spec.Caste)),
			Task:          continueReviewTaskForCaste(spec.Caste),
			TaskID:        fmt.Sprintf("continue-review-%s", spec.Caste),
			Timeout:       timeoutSeconds,
			Status:        "planned",
			Brief:         continueExternalBriefWithHandoffSchema(renderCodexContinueReviewBrief(root, phase, manifest, verification, assessment, spec)),
			SkillSection:  assignment.Section,
			SkillCount:    assignment.SkillCount,
			ColonySkills:  assignment.ColonyCount,
			DomainSkills:  assignment.DomainCount,
			MatchedSkills: append([]string{}, assignment.MatchedNames...),
		})
	}
	return dispatches
}

func countContinueExternalWaves(dispatches []codexContinueExternalDispatch) int {
	seen := map[int]struct{}{}
	for _, dispatch := range dispatches {
		if dispatch.Wave > 0 {
			seen[dispatch.Wave] = struct{}{}
		}
	}
	return len(seen)
}

func continuePlanArtifactsPath(phaseID int, name string) string {
	return filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phaseID), name))
}
