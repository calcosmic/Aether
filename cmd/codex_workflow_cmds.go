package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/storage"
	"github.com/spf13/cobra"
)

var layEggsCmd = &cobra.Command{
	Use:   "lay-eggs",
	Short: "Set up Aether in the current directory from the hub",
	Args:  cobra.NoArgs,
	RunE:  runSetup,
}

var colonizeCmd = &cobra.Command{
	Use:   "colonize",
	Short: "Survey the repository, write territory reports, and record surveyor dispatches",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		forceResurvey, _ := cmd.Flags().GetBool("force-resurvey")
		forceAlias, _ := cmd.Flags().GetBool("force")
		planOnly, _ := cmd.Flags().GetBool("plan-only")
		workerTimeout, err := resolveWorkerTimeoutFlag(cmd)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		result, err := runCodexColonizeWithOptions(skillWorkspaceRoot(), codexColonizeOptions{
			ForceResurvey: forceResurvey || forceAlias,
			WorkerTimeout: workerTimeout,
			PlanOnly:      planOnly,
		})
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		closeLifecycleCommand(result, "colonize", "", "")
		outputWorkflow(result, renderColonizeVisual(result))
		return nil
	},
}

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Generate or review a survey-aware phase plan for the current colony goal",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		refresh, _ := cmd.Flags().GetBool("refresh")
		forceAlias, _ := cmd.Flags().GetBool("force")
		synthetic, _ := cmd.Flags().GetBool("synthetic")
		planOnly, _ := cmd.Flags().GetBool("plan-only")
		repairArtifact, _ := cmd.Flags().GetBool("repair-artifact")
		depth, _ := cmd.Flags().GetString("depth")
		planningDepth, _ := cmd.Flags().GetString("planning-depth")
		verificationDepth, _ := cmd.Flags().GetString("verification-depth")
		targetConfidence, _ := cmd.Flags().GetInt("target")
		maxIterations, _ := cmd.Flags().GetInt("max-iterations")
		acceptBelowTarget, _ := cmd.Flags().GetBool("accept")
		revisionType, _ := cmd.Flags().GetString("revision-type")
		revisionReason, _ := cmd.Flags().GetString("revision-reason")
		revisionEvidence, _ := cmd.Flags().GetStringArray("revision-evidence")
		researchDocs, _ := cmd.Flags().GetStringArray("research")

		if printBrief, _ := cmd.Flags().GetBool("print-brief"); printBrief {
			fullFlag, _ := cmd.Flags().GetBool("full")
			if err := printPlanningBriefs(skillWorkspaceRoot(), fullFlag); err != nil {
				outputError(1, err.Error(), nil)
			}
			return nil
		}

		workerTimeout, err := resolveWorkerTimeoutFlag(cmd)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		result, err := runCodexPlanWithOptions(skillWorkspaceRoot(), codexPlanOptions{
			Refresh:           refresh || forceAlias,
			Synthetic:         synthetic,
			PlanOnly:          planOnly,
			Depth:             depth,
			PlanningDepth:     planningDepth,
			VerificationDepth: verificationDepth,
			WorkerTimeout:     workerTimeout,
			TargetConfidence:  targetConfidence,
			MaxIterations:     maxIterations,
			Accept:            acceptBelowTarget,
			RepairArtifact:    repairArtifact,
			RevisionType:      revisionType,
			RevisionReason:    revisionReason,
			RevisionEvidence:  revisionEvidence,
			ResearchDocs:      researchDocs,
			RequireTerritory:  true,
		})
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		closeLifecycleCommand(result, "plan", "", "")
		outputWorkflow(result, renderPlanVisual(result))
		return nil
	},
}

// territoryPlanPreflight is the common automatic territory gate used by the
// plan command and the Phase 199 front door. It never dispatches workers or
// writes state. A missing/stale result carries the existing colonize manifest
// as internal work; unavailable evidence stops with a single recovery door.
func territoryPlanPreflight(root string, opts codexPlanOptions) (SurveyFreshnessResult, *codexColonizeManifest, error) {
	freshness := ensureTerritoryFreshness(root)
	switch freshness.Freshness {
	case colony.SurveyFreshnessFresh:
		return freshness, nil, nil
	case colony.SurveyFreshnessUnavailable:
		detail := strings.TrimSpace(freshness.SourceError)
		if detail == "" {
			detail = strings.Join(surveyReasonStrings(freshness.ReasonCodes), ", ")
		}
		return freshness, nil, fmt.Errorf("territory evidence is unavailable: %s; run `aether resume` to inspect and recover the retained evidence", detail)
	case colony.SurveyFreshnessMissing, colony.SurveyFreshnessStale:
		facts, err := surveyWorkspace(root)
		if err != nil {
			return freshness, nil, fmt.Errorf("prepare automatic territory refresh: %w", err)
		}
		existingSurvey := surveyDocsExist(filepath.Join(store.BasePath(), "survey"))
		colonizeOpts := codexColonizeOptions{
			ForceResurvey:         true,
			WorkerTimeout:         opts.WorkerTimeout,
			PlanOnly:              true,
			RequireCompleteSurvey: true,
		}
		manifest := buildCodexColonizeManifest(
			root,
			facts,
			colonizeOpts,
			"plan-only",
			existingSurvey,
			snapshotRelativeFiles(root, filepath.ToSlash(filepath.Join(".aether", "data", "survey"))),
			resolveCodexWorkerContext(),
		)
		manifest.TransactionID = fmt.Sprintf("territory-%d-%s", time.Now().UTC().UnixNano(), randomHex(4))
		manifest.BaselineDigest = territorySurveyBaselineDigest(root)
		manifest.PublicationMode = territoryPublicationTransactional
		manifest.RefreshReasons = append([]SurveyFreshnessReasonCode{}, freshness.ReasonCodes...)
		manifest.CandidateSurveyDir = filepath.ToSlash(filepath.Join(".aether", "data", "territory-candidates", manifest.TransactionID, "survey"))
		for i := range manifest.Dispatches {
			paths := make([]string, 0, len(manifest.Dispatches[i].Outputs))
			for _, output := range manifest.Dispatches[i].Outputs {
				paths = append(paths, filepath.ToSlash(filepath.Join(manifest.CandidateSurveyDir, output)))
			}
			manifest.Dispatches[i].OutputPaths = paths
			manifest.Dispatches[i].Brief = fmt.Sprintf(
				"Survey task: %s\n\nWrite these candidate survey outputs in the repo: %s\n\nSurvey the territory at %s. Do not write the live .aether/data/survey directory; the Go finalizer publishes the verified candidate atomically.",
				manifest.Dispatches[i].Task,
				strings.Join(paths, ", "),
				root,
			)
		}
		return freshness, &manifest, nil
	default:
		return freshness, nil, fmt.Errorf("territory freshness returned invalid value %q; run `aether resume`", freshness.Freshness)
	}
}

func surveyReasonStrings(reasons []SurveyFreshnessReasonCode) []string {
	values := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		values = append(values, string(reason))
	}
	return values
}

func territoryRefreshPlanResult(state colony.ColonyState, freshness SurveyFreshnessResult, manifest codexColonizeManifest) map[string]interface{} {
	goal := ""
	if state.Goal != nil {
		goal = strings.TrimSpace(*state.Goal)
	}
	dispatches := surveyorDispatchMaps(manifest.Dispatches)
	return map[string]interface{}{
		"planned":                    false,
		"plan_only":                  true,
		"goal":                       goal,
		"territory_refresh_required": true,
		"territory_freshness":        freshness,
		"territory_outcome_label":    freshness.OutcomeLabel(),
		"colonize_manifest":          manifest,
		"dispatches":                 dispatches,
		"surveyors":                  dispatches,
		"dispatch_mode":              manifest.DispatchMode,
		"dispatch_contract":          manifest.DispatchContract,
		"requires_finalizer":         true,
		"finalizer_command":          manifest.FinalizerCommand,
		"resume_command":             "aether plan",
		"owner_decision_required":    false,
		"next":                       "run the manifest surveyors, finalize the territory refresh, then resume `aether plan`",
	}
}

func parseCoherentJobProposals(rawValues []string) ([]coherentJobProposal, error) {
	proposals := make([]coherentJobProposal, 0, len(rawValues))
	for index, raw := range rawValues {
		if strings.TrimSpace(raw) == "" {
			return nil, fmt.Errorf("job proposal %d is empty; provide one JSON object", index+1)
		}
		decoder := json.NewDecoder(strings.NewReader(raw))
		decoder.DisallowUnknownFields()
		var proposal coherentJobProposal
		if err := decoder.Decode(&proposal); err != nil {
			return nil, fmt.Errorf("job proposal %d is invalid JSON: %w", index+1, err)
		}
		var trailing interface{}
		if err := decoder.Decode(&trailing); err != io.EOF {
			if err == nil {
				return nil, fmt.Errorf("job proposal %d must contain exactly one JSON object", index+1)
			}
			return nil, fmt.Errorf("job proposal %d has trailing JSON: %w", index+1, err)
		}
		proposals = append(proposals, proposal)
	}
	return proposals, nil
}

var buildCmd = &cobra.Command{
	Use:   "build <phase>",
	Short: "Dispatch a real Codex build packet with worker briefs, claims, and spawn tracking",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		phaseNum, err := strconv.Atoi(args[0])
		if err != nil || phaseNum < 1 {
			outputError(1, fmt.Sprintf("invalid phase %q", args[0]), nil)
			return nil
		}

		// D-14: --checkin (force the pause) and --no-checkin (skip it) are a
		// named conflict, refused before ANY side effect -- no plan-only
		// attempt opened, no manifest written, no checkpoint or colony state
		// touched. This must run before the print-brief and plan-only
		// branches below.
		checkinFlag, _ := cmd.Flags().GetBool("checkin")
		noCheckinFlag, _ := cmd.Flags().GetBool("no-checkin")
		if checkinFlag && noCheckinFlag {
			outputError(1, "cannot combine --checkin and --no-checkin: --checkin forces the pre-spawn team check-in pause, --no-checkin skips it -- choose one", nil)
			return nil
		}

		selectedTasks := normalizeCLIStringList(mustGetStringArray(cmd, "task"))
		forceBuild, _ := cmd.Flags().GetBool("force")
		workerTimeout, err := resolveWorkerTimeoutFlag(cmd)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		lightFlag, _ := cmd.Flags().GetBool("light")
		heavyFlag, _ := cmd.Flags().GetBool("heavy")
		verificationDepth, _ := cmd.Flags().GetString("verification-depth")
		queenCastes, _ := cmd.Flags().GetStringArray("castes")
		queenCasteReason, _ := cmd.Flags().GetString("caste-reason")
		queenCasteWhy, _ := cmd.Flags().GetStringArray("caste-why")
		jobProposals, err := parseCoherentJobProposals(mustGetStringArray(cmd, "job-proposal"))
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}

		if printBrief, _ := cmd.Flags().GetBool("print-brief"); printBrief {
			worker, _ := cmd.Flags().GetString("worker")
			fullFlag, _ := cmd.Flags().GetBool("full")
			err := printWorkerBriefs(
				skillWorkspaceRoot(),
				phaseNum,
				selectedTasks,
				worker,
				buildPrintBriefOptions(workerTimeout, forceBuild, lightFlag, heavyFlag, fullFlag, verificationDepth),
			)
			if err != nil {
				outputError(1, err.Error(), nil)
			}
			return nil
		}

		planOnly, _ := cmd.Flags().GetBool("plan-only")
		if planOnly {
			result, state, phase, dispatches, err := runCodexBuildPlanOnlyWithOptions(skillWorkspaceRoot(), phaseNum, selectedTasks, codexBuildOptions{
				WorkerTimeout:     workerTimeout,
				Force:             forceBuild,
				LightFlag:         lightFlag,
				HeavyFlag:         heavyFlag,
				VerificationDepth: verificationDepth,
				QueenCastes:       queenCastes,
				QueenCasteReason:  queenCasteReason,
				QueenCasteWhy:     queenCasteWhy,
				JobProposals:      jobProposals,
			})
			if err != nil {
				outputError(1, err.Error(), nil)
				return nil
			}
			// The wrapper's Team Check-In stage pauses on result.checkin_requested;
			// the runtime plan itself is identical whatever this decides
			// (D-11..D-14, cmd/ceremony_team_checkin.go: decideBuildCheckin).
			buildManifest, _ := result["dispatch_manifest"].(codexBuildManifest)
			pendingOwnerDecision, pendingOwnerDecisionWhy := buildHasPendingOwnerDecision(buildManifest)
			checkinDecision := decideBuildCheckin(buildCheckinDecisionInput{
				NoCheckin:                noCheckinFlag,
				Checkin:                  checkinFlag,
				PendingOwnerDecision:     pendingOwnerDecision,
				PendingOwnerDecisionWhy:  pendingOwnerDecisionWhy,
				ImplementationDispatches: len(dispatches),
			})
			result["checkin_requested"] = checkinDecision.Requested
			result["checkin_reason"] = string(checkinDecision.Reason)
			if checkinDecision.Why != "" {
				result["checkin_reason_detail"] = checkinDecision.Why
			}
			// D-08/D-09: the build-start blocker heads-up, computed from the
			// SAME manifest before the spawn plan renders -- so a genuinely
			// stuck phase is named before any dispatch decision is even
			// shown. --no-checkin is the one non-interactive signal reachable
			// here (autopilot never calls this RunE at all; it dispatches
			// through runCodexBuildWithOptions directly). Never a refusal,
			// never a silent warning: the wrapper decides what happens next
			// from these two result keys.
			blockerAdvisory := decideBuildBlockerAdvisory(buildStartBlockerSignals(buildManifest), noCheckinFlag)
			if len(blockerAdvisory.Signals) > 0 {
				result["blocker_advisory"] = blockerAdvisory.Signals
				if blockerAdvisory.Ask {
					result["blocker_advisory_question"] = buildBlockerAdvisoryQuestion
				}
			}
			reviewDepthPlan := reviewDepthFromResult(result)
			planOnlyVisual := renderBuildPlanOnlyVisual(state, phase, dispatches, reviewDepthPlan, queenPolicyFromResult(result))
			// D-12: the automatic fast path still shows a compact, non-blocking
			// summary before dispatch -- it is never invisible, just never a
			// question. Rendered only for the exact fast-path reason so a
			// pending-decision or explicit --checkin pause never gets this
			// summary appended alongside the full check-in card.
			if !checkinDecision.Requested && checkinDecision.Reason == buildCheckinReasonOneWorkerFastPath && len(dispatches) == 1 {
				fastPathSummary, fastPathVisual := renderBuildFastPathSummary(phase, dispatches[0], checkinDecision)
				result["checkin_summary"] = fastPathSummary
				planOnlyVisual = planOnlyVisual + "\n" + fastPathVisual
			}
			// The heads-up leads -- it decides whether the build happens at
			// all, so it appears before the spawn plan it gates.
			if advisoryVisual := renderBuildBlockerAdvisory(blockerAdvisory); advisoryVisual != "" {
				planOnlyVisual = advisoryVisual + "\n\n" + planOnlyVisual
			}
			outputWorkflow(result, planOnlyVisual)
			return nil
		}

		syntheticBuild, _ := cmd.Flags().GetBool("synthetic")
		cbThreshold, _ := cmd.Flags().GetInt("circuit-breaker-threshold")
		verboseFlag, _ := cmd.Flags().GetBool("verbose")
		result, err := runCodexBuildWithOptions(skillWorkspaceRoot(), phaseNum, selectedTasks, syntheticBuild, codexBuildOptions{
			WorkerTimeout:           workerTimeout,
			Force:                   forceBuild,
			LightFlag:               lightFlag,
			HeavyFlag:               heavyFlag,
			VerificationDepth:       verificationDepth,
			CircuitBreakerThreshold: cbThreshold,
			Verbose:                 verboseFlag,
			QueenCastes:             queenCastes,
			QueenCasteReason:        queenCasteReason,
			QueenCasteWhy:           queenCasteWhy,
			JobProposals:            jobProposals,
		})
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}

		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			outputError(2, fmt.Sprintf("failed to reload colony state: %v", err), nil)
			return nil
		}

		// WR-05 (195-REVIEW.md): a partially credited build must never fall
		// through to the ordinary finished-build screen, which states that
		// verification happens next, names the following phase, and points at
		// the continue command -- none of which is true when tasks are still
		// unstarted -- while never showing the recovery command at all.
		if partial, _ := result["recovery_job"].(bool); partial {
			unfinished, _ := result["unfinished_task_ids"].([]string)
			recoveryCommand, _ := result["recovery_command"].(string)
			outputWorkflow(result, renderBuildPartialCreditVisual(state, state.Plan.Phases[phaseNum-1], unfinished, recoveryCommand))
			return nil
		}

		dispatches, planErr := plannedBuildDispatches(state.Plan.Phases[phaseNum-1], state.ColonyDepth)
		if planErr != nil {
			// WR-06: never render a planning refusal as an empty team.
			outputError(2, planErr.Error(), nil)
			return nil
		}
		if manifestPath, ok := result["manifest"].(string); ok && strings.TrimSpace(manifestPath) != "" {
			rel := strings.TrimPrefix(manifestPath, ".aether/data/")
			var manifest codexBuildManifest
			if err := store.LoadJSON(rel, &manifest); err == nil && len(manifest.Dispatches) > 0 {
				dispatches = manifest.Dispatches
			}
		}
		reviewDepthBuild := reviewDepthFromResult(result)
		// D-08/D-09: the same build-start blocker heads-up as the plan-only
		// lane above -- the direct dispatch path never derives a full
		// codexBuildManifest of its own, so the forced-reviewer signal is
		// re-derived read-only from the phase's own wording (identical
		// derivation the plan-only manifest uses), and the
		// last-continue-blocked signal needs no manifest field beyond Phase.
		// The unanswered-question signal DOES need BoundaryQuestionCount
		// (WR-01, 198-REVIEW.md): populate it with the same candidates the
		// plan-only lane uses, via checkOrchestratorBoundaryQuestions's
		// read-only check -- never the materializing call, since this lane
		// has already dispatched real work by the time this advisory runs.
		directBuildPhase := state.Plan.Phases[phaseNum-1]
		directBoundary, boundaryErr := checkOrchestratorBoundaryQuestions("build", state, directBuildPhase, buildBoundaryQuestionCandidates(directBuildPhase, selectedTasks))
		if boundaryErr != nil {
			outputError(2, fmt.Sprintf("failed to check boundary questions: %v", boundaryErr), nil)
			return nil
		}
		directBuildManifest := codexBuildManifest{
			Phase:                 phaseNum,
			ForcedReviewers:       forcedReviewerRecords(queenForcedReviewersForPhase(directBuildPhase)),
			BoundaryQuestionCount: len(directBoundary.Questions),
		}
		blockerAdvisoryDirect := decideBuildBlockerAdvisory(buildStartBlockerSignals(directBuildManifest), noCheckinFlag)
		buildVisual := appendSpendCostLine(
			renderBuildVisualWithDispatches(state, state.Plan.Phases[phaseNum-1], dispatches, reviewDepthBuild, queenPolicyFromResult(result)),
			phaseNum,
		)
		if advisoryVisual := renderBuildBlockerAdvisory(blockerAdvisoryDirect); advisoryVisual != "" {
			buildVisual = advisoryVisual + "\n\n" + buildVisual
		}
		// The one cost line ends this lane's ending screen too. The
		// plan-only path above deliberately does NOT get one: nothing has
		// been spent yet when a team is merely being planned.
		outputWorkflow(result, buildVisual)
		return nil
	},
}

var continueCmd = &cobra.Command{
	Use:   "continue",
	Short: "Verify the active build packet, close dispatched workers, and advance honestly",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		workerTimeout, err := resolveWorkerTimeoutFlag(cmd)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		verificationTimeout, _, err := resolveContinueVerificationTimeoutFlag(cmd)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		planOnly, _ := cmd.Flags().GetBool("plan-only")
		lightFlag, _ := cmd.Flags().GetBool("light")
		heavyFlag, _ := cmd.Flags().GetBool("heavy")
		skipWatchers, _ := cmd.Flags().GetBool("skip-watchers")
		continueCastes, _ := cmd.Flags().GetStringArray("castes")
		continueCasteReason, _ := cmd.Flags().GetString("caste-reason")
		continueCasteWhy, _ := cmd.Flags().GetStringArray("caste-why")
		verificationDepth, _ := cmd.Flags().GetString("verification-depth")
		classicCeremony, _ := cmd.Flags().GetBool("classic-ceremony")
		if classicCeremony {
			planOnly = true
			heavyFlag = true
			if strings.TrimSpace(verificationDepth) == "" {
				verificationDepth = string(colony.VerificationDepthHeavy)
			}
		}
		if planOnly {
			result, state, phase, dispatches, err := runCodexContinuePlanOnly(skillWorkspaceRoot(), codexContinueOptions{
				ReconcileTaskIDs:    normalizeCLIStringList(mustGetStringArray(cmd, "reconcile-task")),
				ReadOnlyArtifacts:   normalizeCLIStringList(mustGetStringArray(cmd, "read-only-artifact")),
				WorkerTimeout:       workerTimeout,
				VerificationTimeout: verificationTimeout,
				LightFlag:           lightFlag,
				HeavyFlag:           heavyFlag,
				SkipWatchers:        skipWatchers,
				VerificationDepth:   verificationDepth,
				QueenCastes:         continueCastes,
				QueenCasteReason:    continueCasteReason,
				QueenCasteWhy:       continueCasteWhy,
			})
			if err != nil {
				outputError(1, err.Error(), nil)
				return nil
			}
			outputWorkflow(result, renderContinuePlanOnlyVisual(state, phase, dispatches, reviewDepthFromResult(result)))
			return nil
		}

		result, state, phase, nextPhase, housekeeping, final, err := runCodexContinue(skillWorkspaceRoot(), codexContinueOptions{
			ReconcileTaskIDs:    normalizeCLIStringList(mustGetStringArray(cmd, "reconcile-task")),
			ReadOnlyArtifacts:   normalizeCLIStringList(mustGetStringArray(cmd, "read-only-artifact")),
			WorkerTimeout:       workerTimeout,
			VerificationTimeout: verificationTimeout,
			LightFlag:           lightFlag,
			HeavyFlag:           heavyFlag,
			SkipWatchers:        skipWatchers,
			VerificationDepth:   verificationDepth,
			QueenCastes:         continueCastes,
			QueenCasteReason:    continueCasteReason,
			QueenCasteWhy:       continueCasteWhy,
		})
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}

		// A check that blocked still spent what it spent, so its ending
		// screen carries the cost line exactly as a passing one does.
		if blocked, _ := result["blocked"].(bool); blocked {
			outputWorkflow(result, appendSpendCostLine(
				renderContinueBlockedVisual(state, phase, result, reviewDepthFromResult(result)),
				phase.ID,
			))
			return nil
		}

		reviewDepthContinue := reviewDepthFromResult(result)
		outputWorkflow(result, appendSpendCostLine(
			renderContinueVisual(state, phase, housekeeping, final, nextPhase, result, reviewDepthContinue),
			phase.ID,
		))
		return nil
	},
}

func mustGetStringArray(cmd *cobra.Command, name string) []string {
	if !cmd.Flags().Changed(name) {
		return nil
	}
	values, _ := cmd.Flags().GetStringArray(name)
	return values
}

func normalizeCLIStringList(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			normalized = append(normalized, part)
		}
	}
	return uniqueSortedStrings(normalized)
}

func resolveWorkerTimeoutFlag(cmd *cobra.Command) (time.Duration, error) {
	if !cmd.Flags().Changed("worker-timeout") {
		return 0, nil
	}
	timeout, err := cmd.Flags().GetDuration("worker-timeout")
	if err != nil {
		return 0, fmt.Errorf("invalid --worker-timeout: %w", err)
	}
	if timeout <= 0 {
		return 0, fmt.Errorf("--worker-timeout must be greater than 0")
	}
	return timeout, nil
}

func resolveContinueVerificationTimeoutFlag(cmd *cobra.Command) (time.Duration, bool, error) {
	if cmd.Flags().Changed("verification-timeout") {
		timeout, err := cmd.Flags().GetDuration("verification-timeout")
		if err != nil {
			return 0, false, fmt.Errorf("invalid --verification-timeout: %w", err)
		}
		if timeout <= 0 {
			return 0, false, fmt.Errorf("--verification-timeout must be greater than 0")
		}
		return timeout, true, nil
	}
	if envValue := strings.TrimSpace(os.Getenv("AETHER_CONTINUE_VERIFICATION_TIMEOUT")); envValue != "" {
		timeout, err := time.ParseDuration(envValue)
		if err != nil {
			return 0, false, fmt.Errorf("invalid AETHER_CONTINUE_VERIFICATION_TIMEOUT: %w", err)
		}
		if timeout <= 0 {
			return 0, false, fmt.Errorf("AETHER_CONTINUE_VERIFICATION_TIMEOUT must be greater than 0")
		}
		return timeout, true, nil
	}
	return continueVerificationTimeout, false, nil
}

// detectGitRepoName returns the repository name from git remote origin URL,
// falling back to the working directory basename if unavailable.
func detectGitRepoName() string {
	if out, err := exec.Command("git", "remote", "get-url", "origin").Output(); err == nil {
		remote := strings.TrimSpace(string(out))
		// Extract last path component, strip .git suffix
		if idx := strings.LastIndex(remote, "/"); idx >= 0 {
			name := remote[idx+1:]
			name = strings.TrimSuffix(name, ".git")
			if name != "" {
				return name
			}
		}
	}
	// Fallback: use working directory basename
	if wd, err := os.Getwd(); err == nil {
		return filepath.Base(wd)
	}
	return ""
}

var sealCmd = &cobra.Command{
	Use:   "seal",
	Short: "Seal a completed colony and write a summary artifact",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		forceFlag, _ := cmd.Flags().GetBool("force")
		forceReason, _ := cmd.Flags().GetString("reason")
		planOnly, _ := cmd.Flags().GetBool("plan-only")
		if planOnly {
			result, err := runSealPlanOnly(resolveAetherRootPath(), forceFlag, forceReason)
			if err != nil {
				renderRecoveryMenu("seal", err.Error(), nil)
				return nil
			}
			outputWorkflow(result, renderSealPlanOnlyVisual(result))
			return nil
		}

		root := resolveAetherRootPath()
		facts, err := loadLifecycleFacts(root, store, time.Now().UTC())
		if err != nil {
			renderRecoveryMenu("seal", err.Error(), nil)
			return nil
		}
		preflight, err := BuildSealPreflight(facts, SealPreflightRequest{
			Caller: SealCallerDirectOwner, Force: forceFlag, Reason: forceReason,
		})
		if err != nil {
			renderRecoveryMenu("seal", err.Error(), nil)
			return nil
		}
		proceed, pending := runSealPreflightConfirmationGate(preflight)
		if !proceed {
			outputOK(pending)
			return nil
		}
		// Curation is a durable part of a valid direct-owner seal. It runs
		// exactly once after typed preflight and owner confirmation, and its
		// resulting evidence is committed by the lifecycle transaction below.
		review := runSealWisdomReview(facts.State.Value)
		override := sealOverride{
			Forced: forceFlag, Reason: preflight.OwnerReason,
			OverriddenBlockers: len(preflight.ResidualRisks),
		}
		for _, phaseID := range preflight.IncompletePhaseIDs {
			override.IncompletePhases = append(override.IncompletePhases, fmt.Sprintf("phase %d", phaseID))
		}
		return completeSealRuntime(facts.State.Value, override, review, preflight)
	},
}

// sealOverride records what an owner-forced seal skipped — the honesty
// payload the seal event, result, and CROWNED-ANTHILL.md all carry. A
// forced seal is legitimate (work done outside the colony, a wedged gate);
// a SILENT forced seal is not.
type sealOverride struct {
	Forced                 bool
	Reason                 string
	IncompletePhases       []string
	OverriddenBlockers     int
	OverriddenReviewBlocks int
}

func (o sealOverride) overrodeAnything() bool {
	return o.Forced && (len(o.IncompletePhases) > 0 || o.OverriddenBlockers > 0 || o.OverriddenReviewBlocks > 0)
}

// sealWisdomReview is the result of running the "what did we learn" pass
// exactly once, before the seal confirmation question exists at all (D-05),
// so its lessons are recorded even if the owner ultimately declines to
// finish. It carries the promoted entries, the consolidation summary, and
// the already-rendered beats text so a later caller can reuse them without
// recomputing (or re-running) the review.
type sealWisdomReview struct {
	Consolidation         sealConsolidationSummary
	PromotedInstinctNames []string
	HiveEligibleCount     int
	HivePromotedCount     int
	HivePromotionFailures int
	Beats                 string
	FinalReview           *sealFinalReviewReport
}

// runSealWisdomReview runs the eight-ant curation pass plus the local/hive
// instinct promotion loop exactly once, printing the same beats and hive
// report lines completeSealRuntime always has, so behavior for a seal that
// proceeds is byte-identical to before this function existed. state is
// accepted for symmetry with the rest of the seal call chain and future
// callers that need it; today's review reads only the package-level store.
func runSealWisdomReview(state colony.ColonyState) sealWisdomReview {
	// Snapshot the instinct entries eligible for THIS seal's own local/hive
	// promotion loop (D-08) before consolidation below decays trust scores
	// and archives stale instincts. Consolidation's archival floor operates
	// on TrustScore, an independent axis from the Confidence bar this seal
	// loop uses -- an instinct the seal ceremony judges promotion-worthy by
	// Confidence must not be silently pulled out from under it by maintenance
	// running earlier in the very same seal. Captured before
	// runSealConsolidation() runs so a mutation in one axis cannot race the
	// decision on the other.
	sealEligibleEntries, sealEligibleErr := loadActiveInstinctEntriesFromStore(store)

	// Learning consolidation (LEARN-02): run the eight-ant curation pass plus
	// decay/archive/promotion BEFORE the local promotion loop below, so its
	// QueenPromotedIDs are available to make that loop subordinate (D-09,
	// Task 2) and Task 3 can render its per-ant beats.
	sealConsolidation := runSealConsolidation()

	// D-09: pkg/memory's RunConsolidation (invoked inside runSealConsolidation
	// above) is the AUTHORITATIVE writer of QUEEN.md's "## Instincts" section
	// at seal -- it already promoted every QueenEligible instinct (confidence
	// >= 0.75 AND >= 3 recorded applications) before this function reached
	// here. The loop below is explicitly SUBORDINATE: it promotes only
	// instincts the authoritative pipeline did NOT already promote, because
	// QueenEligible's application-history bar is one a young colony never
	// clears, and deleting this loop would make seal promote nothing in
	// practice for most colonies. Do not "simplify" this back into a second
	// independent scan that double-writes an instinct pkg/memory already
	// wrote under "## Instincts" into "## Wisdom" as well.
	queenAlreadyPromoted := make(map[string]struct{}, len(sealConsolidation.QueenPromotedIDs))
	for _, id := range sealConsolidation.QueenPromotedIDs {
		queenAlreadyPromoted[id] = struct{}{}
	}

	// Ceremony Step 1: Promote high-confidence instincts to LOCAL QUEEN.md and Hive Brain (D-08, CERE-02)
	var repoName string
	if out, err := exec.Command("git", "remote", "get-url", "origin").Output(); err == nil {
		remote := strings.TrimSpace(string(out))
		// Extract repo name from remote URL (handles both https and ssh)
		parts := strings.Split(remote, "/")
		if len(parts) > 0 {
			repoName = strings.TrimSuffix(parts[len(parts)-1], ".git")
		}
	}
	if repoName == "" {
		if cwd, err := os.Getwd(); err == nil {
			repoName = filepath.Base(cwd)
		}
	}
	var promotedInstinctNames []string
	var hiveEligibleCount int
	var hivePromotedCount int
	var hivePromotionFailures int
	if entries, err := sealEligibleEntries, sealEligibleErr; err == nil {
		for _, entry := range entries {
			if entry.Confidence >= 0.8 && entry.Action != "" {
				if _, alreadyPromoted := queenAlreadyPromoted[entry.ID]; alreadyPromoted {
					// pkg/memory already wrote this instinct into QUEEN.md's
					// "## Instincts" section this seal (D-09) -- the colony
					// promoted it, a different writer did the writing. Do not
					// call promoteInstinctLocal again, but still count it so
					// sealEnrichment.InstinctsPromoted and CROWNED-ANTHILL.md
					// report the full promoted set.
					promotedInstinctNames = append(promotedInstinctNames, entry.ID)
				} else if err := promoteInstinctLocal(store, entry.ID, entry.Action); err == nil {
					promotedInstinctNames = append(promotedInstinctNames, entry.ID)
				}
			}
			if entry.Confidence >= 0.8 && entry.Action != "" {
				hiveEligibleCount++
				if !automaticHivePromotionEnabled() {
					continue
				}
				// Explicitly enabled Hive Brain promotion remains non-blocking.
				domain := entry.Domain
				if domain == "" {
					domain = "general"
				}
				if err := promoteToHiveWithReference(entry.Action, domain, repoName, entry.Confidence, "seal:"+entry.ID); err != nil {
					log.Printf("seal: hive-promote failed for %s: %v", entry.ID, err)
					hivePromotionFailures++
				} else {
					hivePromotedCount++
				}
			}
		}
	}

	// WR-03: the loop above inspects only pre-consolidation snapshot entries
	// with Confidence >= 0.8, but pkg/memory's QueenEligible bar is
	// POST-decay confidence >= 0.75 (plus 3 applications) -- so a
	// pipeline-promoted instinct whose snapshot confidence sits in
	// [0.75, 0.8) would silently vanish from sealEnrichment.InstinctsPromoted
	// and CROWNED-ANTHILL.md's "Promoted Instincts" section. Fold every
	// actually-pipeline-promoted ID into the reported set so the report
	// matches what reached QUEEN.md. Hive promotion above deliberately keeps
	// the snapshot's >= 0.8 bar: a NEW instinct created by consolidation
	// during this very seal is absent from the snapshot and becomes
	// hive-eligible at the next seal -- a documented one-seal lag, not a bug.
	promotedSeen := make(map[string]struct{}, len(promotedInstinctNames))
	for _, id := range promotedInstinctNames {
		promotedSeen[id] = struct{}{}
	}
	for _, id := range sealConsolidation.QueenPromotedIDs {
		if _, ok := promotedSeen[id]; !ok {
			promotedInstinctNames = append(promotedInstinctNames, id)
		}
	}

	// Render the eight-ant consolidation beats (D-06, LEARN-02) so the
	// ceremony reads consolidation -> promotion -> hive: printed here, after
	// the promotion loop above has run, and before the hive reporting lines
	// below.
	beats := renderSealConsolidationBeats(sealConsolidation)
	visualFprint(stdout, beats)
	emitSealConsolidationCeremony(sealConsolidation)

	// Ceremony Step 2: Report hive promotion results (replaces SUGGESTION per CERE-02)
	if hivePromotedCount > 0 {
		visualFprintln(stdout, fmt.Sprintf("Promoted %d instinct(s) to Hive Brain", hivePromotedCount))
	}
	if hivePromotionFailures > 0 {
		visualFprintln(stdout, fmt.Sprintf("WARNING: %d hive promotion(s) failed (see log)", hivePromotionFailures))
	}
	if hiveEligibleCount > 0 && !automaticHivePromotionEnabled() {
		visualFprintln(stdout, fmt.Sprintf("Hive auto-promotion is disabled; %d eligible instinct(s) remain project-local", hiveEligibleCount))
	}

	return sealWisdomReview{
		Consolidation:         sealConsolidation,
		PromotedInstinctNames: promotedInstinctNames,
		HiveEligibleCount:     hiveEligibleCount,
		HivePromotedCount:     hivePromotedCount,
		HivePromotionFailures: hivePromotionFailures,
		Beats:                 beats,
	}
}

// SealTransactionInput is the complete, already-authorized payload for one
// retained closure. Tests and the command path use the same entry point so
// crash recovery cannot diverge from ordinary seal behavior.
type SealTransactionInput struct {
	Root                       string
	DataRoot                   string
	HubRoot                    string
	Facts                      LifecycleFacts
	State                      colony.ColonyState
	Session                    colony.SessionFile
	Preflight                  SealPreflight
	FinalReview                sealFinalReviewReport
	Review                     sealWisdomReview
	Warnings                   []string
	ShelfCandidates            []colony.ShelfEntry
	ReviewBacklog              []colony.ReviewLedgerEntry
	Now                        time.Time
	Fault                      lifecycleTransactionFaultHook
	DisablePostCommitPromotion bool
}

type sealReceiptEnvelope struct {
	Disposition colony.SealDisposition  `json:"disposition"`
	OwnerReason string                  `json:"owner_reason,omitempty"`
	Receipt     colony.LifecycleReceipt `json:"receipt"`
}

type sealRollbackRecord struct {
	SchemaVersion string                    `json:"schema_version"`
	TransactionID string                    `json:"transaction_id"`
	OutcomeKind   colony.OutcomeKind        `json:"outcome_kind"`
	Disposition   colony.SealDisposition    `json:"disposition"`
	OwnerReason   string                    `json:"owner_reason,omitempty"`
	Rollback      *colony.LifecycleRollback `json:"rollback,omitempty"`
}

type sealFindingsRecord struct {
	SchemaVersion string                   `json:"schema_version"`
	TransactionID string                   `json:"transaction_id"`
	OutcomeKind   colony.OutcomeKind       `json:"outcome_kind"`
	Disposition   colony.SealDisposition   `json:"disposition"`
	OwnerReason   string                   `json:"owner_reason,omitempty"`
	Findings      []sealFinalReviewFinding `json:"findings"`
}

type sealLearningsRecord struct {
	SchemaVersion string                 `json:"schema_version"`
	TransactionID string                 `json:"transaction_id"`
	OutcomeKind   colony.OutcomeKind     `json:"outcome_kind"`
	Disposition   colony.SealDisposition `json:"disposition"`
	OwnerReason   string                 `json:"owner_reason,omitempty"`
	Learnings     []colony.PhaseLearning `json:"retained_learnings"`
	FutureWork    []SealUnresolvedItem   `json:"uncompleted_future_work"`
}

type sealCheckpointsRecord struct {
	SchemaVersion string                 `json:"schema_version"`
	TransactionID string                 `json:"transaction_id"`
	OutcomeKind   colony.OutcomeKind     `json:"outcome_kind"`
	Disposition   colony.SealDisposition `json:"disposition"`
	OwnerReason   string                 `json:"owner_reason,omitempty"`
	Checkpoints   []SealOwnerCheckpoint  `json:"owner_checkpoints"`
}

// SealTransactionID is stable across retry and replay. It uses only the
// pre-seal episode and typed authority facts, never bytes the transaction is
// about to mutate.
func SealTransactionID(input SealTransactionInput) string {
	episode := ""
	if input.State.AcceptedCharter != nil {
		episode = strings.TrimSpace(input.State.AcceptedCharter.EpisodeID)
	}
	if episode == "" && input.State.SessionID != nil {
		episode = strings.TrimSpace(*input.State.SessionID)
	}
	goal := ""
	if input.State.Goal != nil {
		goal = strings.TrimSpace(*input.State.Goal)
	}
	seed := struct {
		Episode     string
		Goal        string
		CapturedAt  string
		Disposition colony.SealDisposition
		Reason      string
	}{
		Episode:     episode,
		Goal:        goal,
		CapturedAt:  input.Preflight.FactsCapturedAt,
		Disposition: input.Preflight.Disposition,
		Reason:      input.Preflight.OwnerReason,
	}
	content, _ := json.Marshal(seed)
	digest := strings.TrimPrefix(lifecycleDigest(content), "sha256:")
	if len(digest) > 20 {
		digest = digest[:20]
	}
	return "seal-" + digest
}

// CommitSealTransaction commits the retained state, report, registry,
// evidence, signals, event, outcome and application receipt through the shared
// lifecycle coordinator. A retry resumes durable intent or returns the exact
// already-verified artifacts; it never appends a second event.
func CommitSealTransaction(input SealTransactionInput) (SealTransactionResult, error) {
	if err := input.Preflight.Validate(); err != nil {
		return SealTransactionResult{}, fmt.Errorf("seal transaction preflight: %w", err)
	}
	root, err := filepath.Abs(strings.TrimSpace(input.Root))
	if err != nil || strings.TrimSpace(input.Root) == "" {
		return SealTransactionResult{}, fmt.Errorf("seal transaction: repository root is required")
	}
	root = filepath.Clean(root)
	dataRoot := strings.TrimSpace(input.DataRoot)
	if dataRoot == "" {
		dataRoot = filepath.Join(root, ".aether", "data")
	}
	dataRoot, err = filepath.Abs(dataRoot)
	if err != nil {
		return SealTransactionResult{}, err
	}
	dataRoot = filepath.Clean(dataRoot)
	hubRoot := strings.TrimSpace(input.HubRoot)
	if hubRoot == "" {
		hubRoot = resolveHubPathQuiet()
	}
	if hubRoot == "" {
		return SealTransactionResult{}, fmt.Errorf("seal transaction: hub root is required for the registry transition")
	}
	hubRoot, err = filepath.Abs(hubRoot)
	if err != nil {
		return SealTransactionResult{}, err
	}
	hubRoot = filepath.Clean(hubRoot)
	if input.Now.IsZero() {
		input.Now = time.Now().UTC()
	} else {
		input.Now = input.Now.UTC()
	}
	if err := os.MkdirAll(filepath.Join(root, ".aether"), 0o755); err != nil {
		return SealTransactionResult{}, err
	}
	if err := os.MkdirAll(dataRoot, 0o755); err != nil {
		return SealTransactionResult{}, err
	}
	if err := os.MkdirAll(filepath.Join(hubRoot, "registry"), 0o755); err != nil {
		return SealTransactionResult{}, err
	}

	input.Root, input.DataRoot, input.HubRoot = root, dataRoot, hubRoot
	txID := SealTransactionID(input)
	config := lifecycleTransactionConfig{
		TransactionID: txID,
		Command:       "seal",
		Allowlist: lifecycleTransactionAllowlist{
			RepositoryRoot:    root,
			LifecycleDataRoot: dataRoot,
			Hub: lifecycleTransactionHubRoot{
				Channel: lifecycleTransactionHubStable,
				Path:    hubRoot,
			},
		},
		Fault: input.Fault,
	}
	journalIntent := filepath.Join(dataRoot, "transactions", txID, "intent.json")
	if _, statErr := os.Stat(journalIntent); statErr == nil {
		receipt, resumeErr := resumeLifecycleTransaction(config)
		if resumeErr != nil {
			return SealTransactionResult{}, resumeErr
		}
		if receipt.StateEffect != colony.LifecycleStateEffectCommitted {
			return SealTransactionResult{}, fmt.Errorf("seal transaction %s resumed with %s", txID, receipt.StateEffect)
		}
		result, loadErr := loadSealTransactionResult(input, txID, true)
		if loadErr != nil {
			return SealTransactionResult{}, loadErr
		}
		return finishSealPostCommitPromotions(input, result)
	} else if !os.IsNotExist(statErr) {
		return SealTransactionResult{}, statErr
	}

	artifacts, err := buildSealTransactionArtifacts(input, txID)
	if err != nil {
		return SealTransactionResult{}, err
	}
	tx, err := beginLifecycleTransaction(config)
	if err != nil {
		return SealTransactionResult{}, err
	}
	for _, target := range artifacts {
		if err := tx.DeclareWrite(target.Root, target.Path, target.Content); err != nil {
			return SealTransactionResult{}, err
		}
	}
	if _, err := tx.Commit(); err != nil {
		return SealTransactionResult{}, err
	}
	result, err := loadSealTransactionResult(input, txID, false)
	if err != nil {
		return SealTransactionResult{}, err
	}
	return finishSealPostCommitPromotions(input, result)
}

type sealTransactionArtifact struct {
	Root    lifecycleTransactionRootKind
	Path    string
	Content []byte
}

func buildSealTransactionArtifacts(input SealTransactionInput, txID string) ([]sealTransactionArtifact, error) {
	evidence, signals, queenBytes, err := buildSealClosureEvidence(input, txID)
	if err != nil {
		return nil, err
	}
	transaction := colony.LifecycleTransactionReference{
		ID:          txID,
		Stage:       colony.TransactionStageVerified,
		JournalPath: filepath.Join(input.DataRoot, "transactions", txID),
	}
	receipt := colony.LifecycleReceipt{
		SchemaVersion:      colony.LifecycleSchemaVersion,
		ReceiptID:          txID + "-seal-receipt",
		Command:            "seal",
		OutcomeKind:        input.Preflight.OutcomeKind,
		ProjectionRevision: input.Preflight.ProjectionRevision,
		Evidence:           append([]colony.LifecycleEvidence{}, input.Preflight.Evidence...),
		Verification:       sealLifecycleVerifications(input.Preflight),
		Warnings:           append([]colony.LifecycleIssue{}, input.Preflight.ResidualRisks...),
		StateEffect:        colony.LifecycleStateEffectCommitted,
		Transaction:        transaction,
		Provenance:         colony.RecoveryProvenanceConfirmed,
	}
	if input.Preflight.Disposition == colony.SealDispositionForcedIncomplete {
		receipt.Decisions = []colony.LifecycleDecision{{
			ID:          "owner-forced-incomplete-closure",
			Scope:       "seal",
			Summary:     input.Preflight.OwnerReason,
			EvidenceIDs: []string{"seal-force-authority"},
		}}
	}
	if err := receipt.Validate(); err != nil {
		return nil, fmt.Errorf("seal receipt: %w", err)
	}
	receiptBytes, err := marshalSealJSON(receipt)
	if err != nil {
		return nil, err
	}
	receiptReference := &colony.LifecycleReceiptReference{ID: receipt.ReceiptID, Digest: lifecycleDigest(receiptBytes)}
	outcome := buildSealOutcome(input.Preflight, txID, transaction, receiptReference, receipt)
	if err := outcome.Validate(); err != nil {
		return nil, fmt.Errorf("seal outcome: %w", err)
	}

	state := input.State
	now := input.Now.Format(time.RFC3339)
	state.State = colony.StateCOMPLETED
	state.Milestone = "Crowned Anthill"
	state.MilestoneUpdatedAt = &now
	state.LifecycleReceipt = &receipt
	state.SealOutcome = &outcome
	eventKind := "sealed_verified"
	eventSummary := "Verified colony closure recorded"
	if outcome.Disposition == colony.SealDispositionForcedIncomplete {
		eventKind = "sealed_forced_incomplete"
		eventSummary = "Owner-forced incomplete closure recorded: " + outcome.OwnerReason
	}
	state.Events = append(trimmedEvents(state.Events), fmt.Sprintf("%s|%s|seal|%s", now, eventKind, eventSummary))

	session := input.Session
	session.LastCommand = "seal"
	session.LastCommandAt = now
	session.CurrentPhase = state.CurrentPhase
	session.CurrentMilestone = state.Milestone
	session.SuggestedNext = "aether status"
	session.Summary = eventSummary
	session.LifecycleReceipt = &receipt
	session.SealOutcome = &outcome

	finalReview := input.FinalReview
	finalReview.TransactionID = txID
	finalReview.Disposition = outcome.Disposition
	finalReview.OwnerReason = outcome.OwnerReason
	finalReview.ClosureEvidence = &evidence
	if finalReview.GeneratedAt == "" {
		finalReview.GeneratedAt = now
	}

	findings := sealFindingsRecord{
		SchemaVersion: colony.LifecycleSchemaVersion, TransactionID: txID,
		OutcomeKind: outcome.OutcomeKind, Disposition: outcome.Disposition,
		OwnerReason: outcome.OwnerReason, Findings: append([]sealFinalReviewFinding{}, finalReview.Findings...),
	}
	learnings := sealLearningsRecord{
		SchemaVersion: colony.LifecycleSchemaVersion, TransactionID: txID,
		OutcomeKind: outcome.OutcomeKind, Disposition: outcome.Disposition,
		OwnerReason: outcome.OwnerReason, Learnings: append([]colony.PhaseLearning{}, state.Memory.PhaseLearnings...),
		FutureWork: append([]SealUnresolvedItem{}, input.Preflight.UnresolvedItems...),
	}
	checkpoints := sealCheckpointsRecord{
		SchemaVersion: colony.LifecycleSchemaVersion, TransactionID: txID,
		OutcomeKind: outcome.OutcomeKind, Disposition: outcome.Disposition,
		OwnerReason: outcome.OwnerReason, Checkpoints: append([]SealOwnerCheckpoint{}, input.Preflight.OwnerCheckpoints...),
	}
	rollback := sealRollbackRecord{
		SchemaVersion: colony.LifecycleSchemaVersion, TransactionID: txID,
		OutcomeKind: outcome.OutcomeKind, Disposition: outcome.Disposition,
		OwnerReason: outcome.OwnerReason, Rollback: outcome.Rollback,
	}
	receiptEnvelope := sealReceiptEnvelope{Disposition: outcome.Disposition, OwnerReason: outcome.OwnerReason, Receipt: receipt}

	crowned := buildTransactionalSealSummary(state, input, evidence, receipt)
	eventBytes, err := buildSealEventBusBytes(input, outcome)
	if err != nil {
		return nil, err
	}
	registryBytes, err := buildSealRegistryBytes(input, state)
	if err != nil {
		return nil, err
	}

	values := []struct {
		root  lifecycleTransactionRootKind
		path  string
		value any
	}{
		{lifecycleTransactionRootData, "COLONY_STATE.json", state},
		{lifecycleTransactionRootData, "session.json", session},
		{lifecycleTransactionRootData, "seal/outcome.json", outcome},
		{lifecycleTransactionRootData, "seal/receipt.json", receiptEnvelope},
		{lifecycleTransactionRootData, "seal/closure-evidence.json", evidence},
		{lifecycleTransactionRootData, sealFinalReviewReportRel, finalReview},
		{lifecycleTransactionRootData, "seal/findings.json", findings},
		{lifecycleTransactionRootData, "seal/learnings.json", learnings},
		{lifecycleTransactionRootData, "seal/checkpoints.json", checkpoints},
		{lifecycleTransactionRootData, "seal/rollback.json", rollback},
		{lifecycleTransactionRootData, "pheromones.json", signals},
	}
	artifacts := []sealTransactionArtifact{
		{Root: lifecycleTransactionRootRepository, Path: filepath.Join(".aether", "CROWNED-ANTHILL.md"), Content: []byte(crowned)},
		{Root: lifecycleTransactionRootRepository, Path: filepath.Join(".aether", "QUEEN.md"), Content: queenBytes},
	}
	for _, entry := range values {
		content, marshalErr := marshalSealJSON(entry.value)
		if marshalErr != nil {
			return nil, marshalErr
		}
		artifacts = append(artifacts, sealTransactionArtifact{Root: entry.root, Path: entry.path, Content: content})
	}
	artifacts = append(artifacts,
		sealTransactionArtifact{Root: lifecycleTransactionRootData, Path: "event-bus.jsonl", Content: eventBytes},
		sealTransactionArtifact{Root: lifecycleTransactionRootHubStable, Path: filepath.Join("registry", "registry.json"), Content: registryBytes},
	)
	return artifacts, nil
}

func buildSealOutcome(preflight SealPreflight, txID string, transaction colony.LifecycleTransactionReference, receipt *colony.LifecycleReceiptReference, lifecycleReceipt colony.LifecycleReceipt) colony.SealOutcome {
	outcome := colony.SealOutcome{
		SchemaVersion:      colony.LifecycleSchemaVersion,
		OutcomeID:          txID + "-outcome",
		Command:            "seal",
		OutcomeKind:        preflight.OutcomeKind,
		ProjectionRevision: preflight.ProjectionRevision,
		Disposition:        preflight.Disposition,
		CompletedPhases:    append([]int{}, preflight.CompletedPhaseIDs...),
		IncompletePhases:   append([]int{}, preflight.IncompletePhaseIDs...),
		IncompleteTaskIDs:  append([]string{}, preflight.IncompleteTaskIDs...),
		FailedGates:        sealLifecycleVerificationsForGates(preflight.FailedGates, false),
		SkippedGates:       sealLifecycleVerificationsForGates(preflight.SkippedGates, false),
		OwnerReason:        preflight.OwnerReason,
		Rollback:           preflight.Rollback,
		Evidence:           append([]colony.LifecycleEvidence{}, preflight.Evidence...),
		Verification:       append([]colony.LifecycleVerification{}, lifecycleReceipt.Verification...),
		Warnings:           append([]colony.LifecycleIssue{}, preflight.ResidualRisks...),
		StateEffect:        colony.LifecycleStateEffectCommitted,
		Transaction:        transaction,
		Receipt:            receipt,
		Provenance:         colony.RecoveryProvenanceConfirmed,
	}
	for _, issue := range preflight.MissingEvidence {
		outcome.MissingEvidence = append(outcome.MissingEvidence, colony.LifecycleEvidence{ID: issue.ID, Kind: "missing", Summary: issue.Summary})
	}
	for _, item := range preflight.UnresolvedItems {
		outcome.UnresolvedEvidence = append(outcome.UnresolvedEvidence, colony.LifecycleEvidence{ID: item.Kind + ":" + item.ID, Kind: item.Kind, Summary: item.Summary})
	}
	if preflight.Disposition == colony.SealDispositionForcedIncomplete {
		outcome.Decisions = []colony.LifecycleDecision{{ID: "owner-forced-incomplete-closure", Scope: "seal", Summary: preflight.OwnerReason, EvidenceIDs: []string{"seal-force-authority"}}}
	}
	return outcome
}

func sealLifecycleVerifications(preflight SealPreflight) []colony.LifecycleVerification {
	result := sealLifecycleVerificationsForGates(preflight.PassedGates, true)
	result = append(result, sealLifecycleVerificationsForGates(preflight.FailedGates, false)...)
	result = append(result, sealLifecycleVerificationsForGates(preflight.SkippedGates, false)...)
	return result
}

func sealLifecycleVerificationsForGates(gates []colony.GateResultEntry, passed bool) []colony.LifecycleVerification {
	result := make([]colony.LifecycleVerification, 0, len(gates))
	for _, gate := range gates {
		result = append(result, colony.LifecycleVerification{Name: gate.Name, Passed: passed, EvidenceIDs: []string{"gate:" + gate.Name}, Detail: gate.Detail})
	}
	return result
}

func buildSealClosureEvidence(input SealTransactionInput, txID string) (SealClosureEvidence, colony.PheromoneFile, []byte, error) {
	repositoryIdentity := stableRepoIdentity(input.Root)
	if repositoryIdentity == "" {
		repositoryIdentity = "path_" + strings.TrimPrefix(lifecycleDigest([]byte(input.Root)), "sha256:")[:12]
	}
	evidence := SealClosureEvidence{
		SchemaVersion: colony.LifecycleSchemaVersion, TransactionID: txID,
		OutcomeKind: input.Preflight.OutcomeKind, Disposition: input.Preflight.Disposition,
		OwnerReason:      input.Preflight.OwnerReason,
		Checkpoints:      append([]SealOwnerCheckpoint{}, input.Preflight.OwnerCheckpoints...),
		ResidualRisks:    append([]colony.LifecycleIssue{}, input.Preflight.ResidualRisks...),
		UncompletedWork:  append([]SealUnresolvedItem{}, input.Preflight.UnresolvedItems...),
		RetainedContents: append([]SealPreservedContent{}, input.Preflight.PreservedContents...),
		PrimaryNext:      input.Preflight.PrimaryNext, OptionalNext: input.Preflight.OptionalNext,
		ProjectionRevision: input.Preflight.ProjectionRevision,
	}
	stateSource := emptyFallback(strings.TrimSpace(input.Facts.State.Source.Path), filepath.Join(input.DataRoot, "COLONY_STATE.json"))
	for _, learning := range input.State.Memory.PhaseLearnings {
		evidence.Memory = append(evidence.Memory, sealMemoryProvenance(learning.ID, "phase_learning", stateSource, "COLONY_STATE.json", repositoryIdentity, "project", learning, []string{"phase-learning:" + learning.ID}, nil, txID))
	}
	for _, decision := range input.State.Memory.Decisions {
		evidence.Memory = append(evidence.Memory, sealMemoryProvenance(decision.ID, "decision", stateSource, "COLONY_STATE.json", repositoryIdentity, "colony", decision, nil, []string{decision.ID}, txID))
	}
	instinctSource := emptyFallback(strings.TrimSpace(input.Facts.Memory.Source.Path), filepath.Join(input.DataRoot, "instincts.json"))
	policy := string(currentHiveRuntimePolicy())
	queenPath := filepath.Join(input.Root, ".aether", "QUEEN.md")
	queenBefore, readErr := os.ReadFile(queenPath)
	if readErr != nil && !os.IsNotExist(readErr) {
		return SealClosureEvidence{}, colony.PheromoneFile{}, nil, readErr
	}
	queen := string(queenBefore)
	for _, instinct := range input.Facts.Memory.Value.Instincts {
		memory := sealMemoryProvenance(instinct.ID, "instinct", instinctSource, "instincts.json", repositoryIdentity, "colony", instinct, []string{instinct.Provenance.Evidence}, nil, txID)
		evidence.Memory = append(evidence.Memory, memory)
		action := strings.TrimSpace(instinct.Action)
		sensitive := sealContentSensitive(action)
		sanitized, sanitizeErr := colony.SanitizeSignalContent(action)
		eligible := instinct.Confidence >= 0.8 && action != "" && !sensitive && sanitizeErr == nil
		allowed := eligible && input.Preflight.Disposition == colony.SealDispositionVerified && policy == string(hivePolicyPromote)
		decision := SealWisdomDecision{
			ID: instinct.ID, Source: instinctSource, Scope: "cross_project_candidate", Digest: lifecycleDigest([]byte(action)),
			Domain: instinct.Domain, Confidence: instinct.Confidence, Policy: policy, Sensitive: sensitive,
			Eligible: eligible, PromotionAllowed: allowed, TransactionID: txID,
		}
		switch {
		case sensitive:
			decision.Sanitization, decision.Reason = "blocked_sensitive", "content matched the private or credential boundary"
		case sanitizeErr != nil:
			decision.Sanitization, decision.Reason = "blocked", sanitizeErr.Error()
		case !eligible:
			decision.Sanitization, decision.Reason = "passed", "content remains local because it is below the promotion threshold"
		case input.Preflight.Disposition == colony.SealDispositionForcedIncomplete:
			decision.Sanitization, decision.Reason = "passed", "forced-incomplete closure never promotes cross-project wisdom"
		case policy != string(hivePolicyPromote):
			decision.Sanitization, decision.Reason = "passed", "resolved Hive policy prohibits promotion"
		default:
			decision.Sanitization, decision.Reason = "passed", "eligible only after the verified seal transaction commits"
		}
		evidence.Wisdom = append(evidence.Wisdom, decision)
		if sanitizeErr == nil && !sensitive && sanitized != "" && !strings.Contains(queen, sanitized) {
			if !strings.HasSuffix(queen, "\n") && queen != "" {
				queen += "\n"
			}
			queen += fmt.Sprintf("\n## Retained at seal (%s)\n- %s\n", txID, sanitized)
		}
	}
	if len(queenBefore) > 0 {
		evidence.Memory = append(evidence.Memory, sealMemoryProvenance("queen-local", "queen_memory", queenPath, "QUEEN.md", repositoryIdentity, "project", string(queenBefore), []string{"queen:pre-seal"}, nil, txID))
	}

	version := "2.0"
	signals := colony.PheromoneFile{Version: &version, Signals: append([]colony.PheromoneSignal{}, input.Facts.Signals.Value...)}
	for index := range signals.Signals {
		signal := &signals.Signals[index]
		content := strings.TrimSpace(extractText(signal.Content))
		decision := SealSignalDecision{
			ID: signal.ID, Type: signal.Type, Source: emptyFallback(signal.Source, input.Facts.Signals.Source.Path), Scope: "project",
			Digest: lifecycleDigest([]byte(content)), Privacy: "project_local", Sensitive: sealContentSensitive(content), TransactionID: txID,
		}
		if strings.EqualFold(signal.Type, "FOCUS") && signal.Active {
			decision.Classification = "expired_at_closure"
			signal.Active = false
			archivedAt := input.Now.Format(time.RFC3339)
			signal.ArchivedAt = &archivedAt
		} else if signal.Active {
			decision.Classification = "retained"
		} else {
			decision.Classification = "already_expired"
		}
		evidence.Signals = append(evidence.Signals, decision)
	}
	return evidence, signals, []byte(queen), nil
}

func sealMemoryProvenance(id, kind, source, storeName, repositoryIdentity, scope string, value any, evidenceIDs, decisionIDs []string, txID string) SealMemoryProvenance {
	content, _ := json.Marshal(value)
	sensitive := sealContentSensitive(string(content))
	sanitization := "passed"
	if sensitive {
		sanitization = "blocked_sensitive"
	}
	return SealMemoryProvenance{
		ID: id, Kind: kind, Source: source, Store: storeName, RepositoryIdentity: repositoryIdentity,
		Scope: scope, Digest: lifecycleDigest(content), EvidenceIDs: compactSealStrings(evidenceIDs), DecisionIDs: compactSealStrings(decisionIDs),
		Sanitization: sanitization, Sensitive: sensitive, TransactionID: txID,
	}
}

func compactSealStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			result = append(result, value)
		}
	}
	return result
}

func sealContentSensitive(content string) bool {
	lower := strings.ToLower(content)
	for _, marker := range []string{"password=", "password:", "api_key", "api-key", "secret=", "secret:", "access_token", "private key", "bearer "} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func buildTransactionalSealSummary(state colony.ColonyState, input SealTransactionInput, evidence SealClosureEvidence, receipt colony.LifecycleReceipt) string {
	if input.Preflight.Disposition == colony.SealDispositionForcedIncomplete {
		var b strings.Builder
		b.WriteString("# Forced seal record — completion not verified\n\n")
		b.WriteString("The owner closed this colony for recordkeeping without verification of completion.\n\n")
		b.WriteString("## Owner reason\n\n")
		b.WriteString(input.Preflight.OwnerReason + "\n\n")
		b.WriteString("## Uncompleted work\n\n")
		for _, item := range evidence.UncompletedWork {
			b.WriteString("- " + item.Summary + "\n")
		}
		b.WriteString("\n## Retained evidence\n\n")
		b.WriteString("- Transaction: " + evidence.TransactionID + "\n")
		b.WriteString("- Receipt: " + receipt.ReceiptID + "\n")
		b.WriteString("- Active state remains at `.aether/data/COLONY_STATE.json`.\n")
		return b.String()
	}
	enrichment := sealEnrichment{
		LearningsCount:        len(state.Memory.PhaseLearnings),
		InstinctsPromoted:     input.Review.PromotedInstinctNames,
		HiveEligible:          input.Review.HiveEligibleCount,
		HivePromoted:          input.Review.HivePromotedCount,
		HivePromotionFailures: input.Review.HivePromotionFailures,
		SignalsExpired:        countSealSignalClassification(evidence.Signals, "expired_at_closure"),
		FlagsResolved:         countResolvedSealCheckpoints(evidence.Checkpoints),
		ShelfCandidates:       input.ShelfCandidates,
		FinalReview:           &input.FinalReview,
		ReviewBacklog:         input.ReviewBacklog,
		ConsolidationReport:   input.Review.Consolidation.ReportPath,
	}
	return buildSealSummary(state, input.Now.Format(time.RFC3339), input.Warnings, enrichment)
}

func countSealSignalClassification(decisions []SealSignalDecision, classification string) int {
	count := 0
	for _, decision := range decisions {
		if decision.Classification == classification {
			count++
		}
	}
	return count
}

func countResolvedSealCheckpoints(checkpoints []SealOwnerCheckpoint) int {
	count := 0
	for _, checkpoint := range checkpoints {
		if checkpoint.Resolved {
			count++
		}
	}
	return count
}

func buildSealEventBusBytes(input SealTransactionInput, outcome colony.SealOutcome) ([]byte, error) {
	path := filepath.Join(input.DataRoot, "event-bus.jsonl")
	before, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if len(before) > 0 && before[len(before)-1] != '\n' {
		before = append(before, '\n')
	}
	expires := input.Now.Add(30 * 24 * time.Hour).Format(time.RFC3339)
	consolidationPayload, _ := json.Marshal(map[string]any{"status": "retained", "transaction_id": outcome.Transaction.ID})
	status := "sealed"
	if outcome.Disposition == colony.SealDispositionForcedIncomplete {
		status = "forced_incomplete"
	}
	goal := ""
	if input.State.Goal != nil {
		goal = strings.TrimSpace(*input.State.Goal)
	}
	sealPayload, err := json.Marshal(events.CeremonyPayload{
		Phase: input.State.CurrentPhase, PhaseName: "Crowned Anthill",
		TaskID: outcome.OutcomeID, Task: "seal", Status: status, Message: goal,
		Completed: len(outcome.CompletedPhases), Total: len(input.State.Plan.Phases),
	})
	if err != nil {
		return nil, err
	}
	eventsToAppend := []events.Event{
		{ID: outcome.Transaction.ID + "-consolidation", Topic: "consolidation.seal", Payload: consolidationPayload, Source: "seal", Timestamp: input.Now.Format(time.RFC3339), TTLDays: 30, ExpiresAt: expires},
		{ID: outcome.Transaction.ID + "-event", Topic: events.CeremonyTopicChamberSeal, Payload: sealPayload, Source: "aether-seal", Timestamp: input.Now.Format(time.RFC3339), TTLDays: 30, ExpiresAt: expires},
	}
	for _, event := range eventsToAppend {
		line, marshalErr := json.Marshal(event)
		if marshalErr != nil {
			return nil, marshalErr
		}
		before = append(before, line...)
		before = append(before, '\n')
	}
	return before, nil
}

func buildSealRegistryBytes(input SealTransactionInput, state colony.ColonyState) ([]byte, error) {
	path := filepath.Join(input.HubRoot, "registry", "registry.json")
	var registry registryData
	if content, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(content, &registry); err != nil {
			return nil, fmt.Errorf("decode registry: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	goal := ""
	if state.Goal != nil {
		goal = strings.TrimSpace(*state.Goal)
	}
	found := false
	for index := range registry.Colonies {
		if filepath.Clean(registry.Colonies[index].RepoPath) != input.Root {
			continue
		}
		registry.Colonies[index].Active = false
		registry.Colonies[index].LastGoal = goal
		found = true
	}
	if !found {
		registry.Colonies = append(registry.Colonies, registryEntry{RepoPath: input.Root, Active: false, LastGoal: goal, RegisteredAt: input.Now.Format(time.RFC3339)})
	}
	return marshalSealJSON(registry)
}

func marshalSealJSON(value any) ([]byte, error) {
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(content, '\n'), nil
}

func loadSealTransactionResult(input SealTransactionInput, txID string, replay bool) (SealTransactionResult, error) {
	var outcome colony.SealOutcome
	if err := readSealTransactionJSON(filepath.Join(input.DataRoot, "seal", "outcome.json"), &outcome); err != nil {
		return SealTransactionResult{}, err
	}
	var envelope sealReceiptEnvelope
	if err := readSealTransactionJSON(filepath.Join(input.DataRoot, "seal", "receipt.json"), &envelope); err != nil {
		return SealTransactionResult{}, err
	}
	var evidence SealClosureEvidence
	if err := readSealTransactionJSON(filepath.Join(input.DataRoot, "seal", "closure-evidence.json"), &evidence); err != nil {
		return SealTransactionResult{}, err
	}
	if outcome.Transaction.ID != txID || envelope.Receipt.Transaction.ID != txID || evidence.TransactionID != txID {
		return SealTransactionResult{}, fmt.Errorf("seal transaction artifacts do not agree on transaction %s", txID)
	}
	if err := outcome.Validate(); err != nil {
		return SealTransactionResult{}, err
	}
	if err := envelope.Receipt.Validate(); err != nil {
		return SealTransactionResult{}, err
	}
	return SealTransactionResult{
		TransactionID: txID, Outcome: outcome, Receipt: envelope.Receipt, Evidence: evidence,
		SummaryPath: filepath.Join(input.Root, ".aether", "CROWNED-ANTHILL.md"),
		PrimaryNext: evidence.PrimaryNext, OptionalNext: evidence.OptionalNext, Replay: replay,
	}, nil
}

func readSealTransactionJSON(path string, destination any) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(content, destination); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func finishSealPostCommitPromotions(input SealTransactionInput, result SealTransactionResult) (SealTransactionResult, error) {
	if input.DisablePostCommitPromotion || result.Outcome.Disposition != colony.SealDispositionVerified {
		return result, nil
	}
	existing, found, err := loadSealPromotionReceipts(input.DataRoot, result.TransactionID)
	if err != nil {
		return result, err
	}
	if found {
		result.Promotions = existing
		applySealPromotionReceipts(&result)
		return result, nil
	}
	for _, decision := range result.Evidence.Wisdom {
		if !decision.PromotionAllowed || decision.Promoted {
			continue
		}
		var candidate *colony.InstinctEntry
		for index := range input.Facts.Memory.Value.Instincts {
			if input.Facts.Memory.Value.Instincts[index].ID == decision.ID {
				candidate = &input.Facts.Memory.Value.Instincts[index]
				break
			}
		}
		if candidate == nil {
			continue
		}
		receipt := SealPromotionReceipt{
			SchemaVersion: colony.LifecycleSchemaVersion, ReceiptID: result.TransactionID + "-hive-" + candidate.ID,
			TransactionID: result.TransactionID, WisdomID: candidate.ID, Policy: decision.Policy, SourceDigest: decision.Digest,
		}
		reference := "seal-receipt:" + result.Receipt.ReceiptID + ":" + candidate.ID
		repoName := filepath.Base(input.Root)
		alreadyPromoted, lookupErr := sealHivePromotionAlreadyRecorded(input.HubRoot, reference)
		if lookupErr != nil {
			receipt.Reason = lookupErr.Error()
		} else if alreadyPromoted {
			receipt.Promoted = true
		} else if err := promoteToHiveWithReference(candidate.Action, candidate.Domain, repoName, candidate.Confidence, reference); err != nil {
			receipt.Reason = err.Error()
		} else {
			receipt.Promoted = true
		}
		result.Promotions = append(result.Promotions, receipt)
	}
	if len(result.Promotions) > 0 {
		if err := storeSealPromotionReceipts(input.DataRoot, result.Promotions); err != nil {
			return result, err
		}
	}
	applySealPromotionReceipts(&result)
	return result, nil
}

type sealPromotionReceiptFile struct {
	SchemaVersion string                 `json:"schema_version"`
	Receipts      []SealPromotionReceipt `json:"receipts"`
}

func loadSealPromotionReceipts(dataRoot, transactionID string) ([]SealPromotionReceipt, bool, error) {
	path := filepath.Join(dataRoot, "seal", "hive-promotion-receipts.json")
	var file sealPromotionReceiptFile
	if err := readSealTransactionJSON(path, &file); err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	for _, receipt := range file.Receipts {
		if receipt.TransactionID != transactionID {
			return nil, false, fmt.Errorf("seal promotion receipt %s belongs to transaction %s, not %s", receipt.ReceiptID, receipt.TransactionID, transactionID)
		}
	}
	return file.Receipts, true, nil
}

func sealHivePromotionAlreadyRecorded(hubRoot, reference string) (bool, error) {
	wisdom, err := loadWisdomLocked(hubRoot)
	if err != nil {
		return false, err
	}
	for _, entry := range wisdom.Entries {
		for _, evidence := range entry.Evidence {
			if evidence.Reference == reference {
				return true, nil
			}
		}
	}
	return false, nil
}

func applySealPromotionReceipts(result *SealTransactionResult) {
	if result == nil || len(result.Promotions) == 0 {
		return
	}
	byWisdomID := make(map[string]SealPromotionReceipt, len(result.Promotions))
	for _, receipt := range result.Promotions {
		byWisdomID[receipt.WisdomID] = receipt
	}
	for index := range result.Evidence.Wisdom {
		receipt, ok := byWisdomID[result.Evidence.Wisdom[index].ID]
		if !ok {
			continue
		}
		result.Evidence.Wisdom[index].Promoted = receipt.Promoted
		result.Evidence.Wisdom[index].PromotionReceiptID = receipt.ReceiptID
	}
}

func storeSealPromotionReceipts(dataRoot string, receipts []SealPromotionReceipt) error {
	content, err := marshalSealJSON(sealPromotionReceiptFile{SchemaVersion: colony.LifecycleSchemaVersion, Receipts: receipts})
	if err != nil {
		return err
	}
	path := filepath.Join(dataRoot, "seal", "hive-promotion-receipts.json")
	return atomicReplaceLifecycleTarget(path, content, 0o644, os.Rename)
}

func completeSealRuntime(state colony.ColonyState, override sealOverride, review sealWisdomReview, supplied ...SealPreflight) error {
	root := resolveAetherRootPath()
	now := time.Now().UTC()
	facts, err := loadLifecycleFacts(root, store, now)
	if err != nil {
		outputError(2, fmt.Sprintf("failed to load seal evidence: %v", err), nil)
		return nil
	}
	preflight := SealPreflight{}
	if len(supplied) > 0 {
		preflight = supplied[0]
	} else {
		request := SealPreflightRequest{Caller: SealCallerDirectOwner, Force: override.Forced, Reason: override.Reason}
		preflight, err = BuildSealPreflight(facts, request)
		if err != nil {
			renderRecoveryMenu("seal", err.Error(), nil)
			return nil
		}
	}
	finalReview := review.FinalReview
	if finalReview == nil {
		finalReview = loadSealFinalReviewForSummary(store)
	}
	if finalReview == nil {
		finalReview = &sealFinalReviewReport{GeneratedAt: now.Format(time.RFC3339), Source: "seal-transaction", Passed: preflight.Disposition == colony.SealDispositionVerified}
	}
	candidates, _ := detectShelfCandidates(state, store)
	input := SealTransactionInput{
		Root: root, DataRoot: store.BasePath(), HubRoot: resolveHubPathQuiet(), Facts: facts,
		State: state, Session: facts.Session.Value, Preflight: preflight, FinalReview: *finalReview, Review: review,
		Warnings: scanHighSeverityOpen(store), ShelfCandidates: candidates, ReviewBacklog: collectOpenReviewBacklog(store, 10), Now: now,
	}
	transactionResult, err := CommitSealTransaction(input)
	if err != nil {
		outputError(2, fmt.Sprintf("seal transaction did not commit: %v", err), nil)
		return nil
	}

	result := map[string]interface{}{
		"sealed": true, "outcome_kind": transactionResult.Outcome.OutcomeKind,
		"disposition": transactionResult.Outcome.Disposition, "seal_outcome": transactionResult.Outcome,
		"lifecycle_receipt": transactionResult.Receipt, "summary": transactionResult.SummaryPath,
		"next": transactionResult.PrimaryNext, "optional_next": transactionResult.OptionalNext,
	}
	if transactionResult.Outcome.Disposition == colony.SealDispositionForcedIncomplete {
		result["force_sealed"] = true
		result["force_reason"] = transactionResult.Outcome.OwnerReason
		result["unverified_phases"] = transactionResult.Outcome.IncompletePhases
		result["overridden_blockers"] = override.OverriddenBlockers + override.OverriddenReviewBlocks
	} else {
		result["milestone"] = "Crowned Anthill"
	}
	outputWorkflow(result, RenderSealOutcome(transactionResult))
	return nil
}

var focusCmd = newSignalShortcutCommand("focus", "FOCUS", "Guide colony attention")
var redirectCmd = newSignalShortcutCommand("redirect", "REDIRECT", "Add a hard constraint for the colony")
var feedbackCmd = newSignalShortcutCommand("feedback", "FEEDBACK", "Add gentle corrective feedback")

var preferencesCmd = &cobra.Command{
	Use:   "preferences [text]",
	Short: "Read or write user preferences stored in QUEEN.md",
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		listOnly, _ := cmd.Flags().GetBool("list")
		hub := resolveHubPath()
		queenPath := filepath.Join(hub, "QUEEN.md")

		if listOnly {
			outputOK(map[string]interface{}{
				"preferences": readUserPreferences(queenPath),
				"path":        queenPath,
			})
			return nil
		}

		text := strings.TrimSpace(strings.Join(args, " "))
		if text == "" {
			outputError(1, "preference text is required unless --list is used", nil)
			return nil
		}
		if len(text) > 500 {
			outputError(1, "preference text exceeds 500 characters", nil)
			return nil
		}

		if _, err := os.Stat(queenPath); os.IsNotExist(err) {
			if err := os.MkdirAll(hub, 0755); err != nil {
				outputError(2, fmt.Sprintf("failed to create hub directory: %v", err), nil)
				return nil
			}
			if err := os.WriteFile(queenPath, []byte(queenDefaultContent), 0644); err != nil {
				outputError(2, fmt.Sprintf("failed to create QUEEN.md: %v", err), nil)
				return nil
			}
		}

		data, err := os.ReadFile(queenPath)
		if err != nil {
			outputError(2, fmt.Sprintf("failed to read QUEEN.md: %v", err), nil)
			return nil
		}

		entry := fmt.Sprintf("- %s", text)
		sectionHeader := "## User Preferences"
		body := string(data)
		idx := strings.Index(body, sectionHeader)
		if idx == -1 {
			body += "\n## User Preferences\n" + entry + "\n"
		} else {
			insertAt := idx + len(sectionHeader)
			if nlIdx := strings.Index(body[insertAt:], "\n"); nlIdx != -1 {
				insertAt += nlIdx + 1
			}
			body = body[:insertAt] + entry + "\n" + body[insertAt:]
		}

		if err := os.WriteFile(queenPath, []byte(body), 0644); err != nil {
			outputError(2, fmt.Sprintf("failed to write QUEEN.md: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"added":      true,
			"preference": text,
			"path":       queenPath,
		})
		return nil
	},
}

func newSignalShortcutCommand(use, signalType, short string) *cobra.Command {
	return &cobra.Command{
		Use:   use + " <text>",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if store == nil {
				outputErrorMessage("no store initialized")
				return nil
			}
			result, err := createAgencySignalResult(signalType, args[0], "user", "", "", 1.0, "")
			if err != nil {
				outputError(1, err.Error(), nil)
				return nil
			}
			outputWorkflow(result, RenderAgencySignalResult(result))
			return nil
		},
	}
}

func createPheromoneSignal(sigType, content, sourceFlag, reasonFlag, ttlFlag string, strength float64, priority string) (map[string]interface{}, error) {
	signal, reinforced, total, err := persistPheromoneSignal(sigType, content, sourceFlag, reasonFlag, ttlFlag, strength, priority)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"created":  true,
		"signal":   signal,
		"total":    total,
		"replaced": reinforced,
	}, nil
}

// createAgencySignalResult performs exactly one existing signal write, then
// derives a read-only receipt from the signal and lifecycle facts returned by
// that write. It does not infer an acknowledgement, causal effect, or conflict.
func createAgencySignalResult(sigType, content, sourceFlag, reasonFlag, ttlFlag string, strength float64, priority string) (AgencySignalResult, error) {
	signal, reinforced, _, err := persistPheromoneSignal(sigType, content, sourceFlag, reasonFlag, ttlFlag, strength, priority)
	if err != nil {
		return AgencySignalResult{}, err
	}
	evidence := currentAgencyReceiptEvidence(resolveAetherRootPath(), signal)
	return BuildAgencySignalResult(signal, reinforced, evidence)
}

func persistPheromoneSignal(sigType, content, sourceFlag, reasonFlag, ttlFlag string, strength float64, priority string) (colony.PheromoneSignal, bool, int, error) {
	if strings.TrimSpace(sigType) == "" || strings.TrimSpace(content) == "" {
		return colony.PheromoneSignal{}, false, 0, fmt.Errorf("signal type and content are required")
	}
	signal, reinforced, err := writePheromoneSignal(sigType, content, priority, sourceFlag, reasonFlag, ttlFlag, strength, nil)
	if err != nil {
		return colony.PheromoneSignal{}, false, 0, err
	}
	var file colony.PheromoneFile
	total := 0
	if loadErr := store.LoadJSON("pheromones.json", &file); loadErr == nil {
		total = len(file.Signals)
	}
	return signal, reinforced, total, nil
}

func currentAgencyReceiptEvidence(root string, signal colony.PheromoneSignal) AgencyReceiptEvidence {
	facts, err := loadLifecycleFacts(root, store, time.Now().UTC())
	if err != nil {
		return AgencyReceiptEvidence{}
	}
	projection := projectLifecycle(facts, LifecycleViewFocused, "runtime")
	evidence := AgencyReceiptEvidence{}
	for _, task := range projection.Tasks.Value {
		if task.ID != nil && task.Status == colony.TaskInProgress && strings.TrimSpace(*task.ID) != "" {
			evidence.ActiveJobIDs = append(evidence.ActiveJobIDs, strings.TrimSpace(*task.ID))
		}
	}
	state := facts.State.Value
	if facts.State.Source.Provenance == LifecycleFactConfirmed &&
		(state.CurrentPhase > 0 || len(state.Plan.Phases) > 0 || strings.TrimSpace(string(state.State)) != "") {
		evidence.LifecycleBoundary = &colony.LifecycleEvidence{
			ID:      "next-safe-boundary:" + signal.ID,
			Kind:    "lifecycle_boundary",
			Source:  facts.State.Source.Path,
			Summary: "The durable signal is available to the next worker-context lifecycle boundary.",
		}
	}
	return evidence
}

func synthesizePlan(goal string, granularity colony.PlanGranularity, domains []string) []colony.Phase {
	count := 3
	switch granularity {
	case colony.GranularityMilestone:
		count = 5
	case colony.GranularityQuarter:
		count = 8
	case colony.GranularityMajor:
		count = 12
	}

	goalLower := strings.ToLower(goal)
	templates := []struct {
		name        string
		description string
		tasks       []string
	}{
		{
			name:        "Discovery and constraints",
			description: "Map the problem, existing code, and boundaries before implementation.",
			tasks: []string{
				"Review the existing code paths relevant to the goal",
				"Capture constraints, risks, and success criteria",
			},
		},
		{
			name:        "Implementation",
			description: "Make the main code changes required to achieve the goal.",
			tasks: []string{
				"Implement the primary changes for the goal",
				"Add or update automated coverage for the new behavior",
			},
		},
		{
			name:        "Verification and polish",
			description: "Verify the result, tighten loose ends, and prepare the colony for seal.",
			tasks: []string{
				"Run focused verification and address regressions",
				"Document key follow-ups, decisions, and user-visible changes",
			},
		},
	}

	if strings.Contains(goalLower, "fix") || strings.Contains(goalLower, "bug") || strings.Contains(goalLower, "broken") {
		templates = []struct {
			name        string
			description string
			tasks       []string
		}{
			{
				name:        "Reproduce and isolate",
				description: "Reproduce the failure and isolate the code path that causes it.",
				tasks:       []string{"Capture the failing behavior", "Identify the root cause and affected boundary"},
			},
			{
				name:        "Targeted fix",
				description: "Implement the smallest correct fix with focused coverage.",
				tasks:       []string{"Apply the fix", "Add regression coverage"},
			},
			{
				name:        "Regression verification",
				description: "Verify the fix and make sure adjacent behavior still holds.",
				tasks:       []string{"Run focused verification", "Document residual risk or follow-ups"},
			},
		}
	}

	if len(domains) > 0 {
		templates[0].tasks = append(templates[0].tasks, fmt.Sprintf("Validate domain-specific expectations for %s", strings.Join(domains, ", ")))
	}

	phases := make([]colony.Phase, 0, count)
	for i := 0; i < count; i++ {
		template := templates[i%len(templates)]
		if i >= len(templates) {
			template.name = fmt.Sprintf("Execution slice %d", i+1)
			template.description = fmt.Sprintf("Continue delivering the goal in bounded slice %d.", i+1)
		}
		phase := colony.Phase{
			ID:              i + 1,
			Name:            template.name,
			Description:     template.description,
			Status:          colony.PhasePending,
			Tasks:           []colony.Task{},
			SuccessCriteria: []string{"The phase outcome is testable", "The phase advances the colony goal without regressions"},
		}
		if i == 0 {
			phase.Status = colony.PhaseReady
		}
		for j, taskGoal := range template.tasks {
			taskID := fmt.Sprintf("%d.%d", i+1, j+1)
			phase.Tasks = append(phase.Tasks, colony.Task{
				ID:     &taskID,
				Goal:   taskGoal,
				Status: colony.TaskPending,
			})
		}
		phases = append(phases, phase)
	}
	return phases
}

func trimmedEvents(events []string) []string {
	if len(events) < 100 {
		return events
	}
	return append([]string{}, events[len(events)-99:]...)
}

func detectDomainsFromRoot(root string) []string {
	domains := []string{}
	checks := map[string][]string{
		"go":     {"go.mod", "go.sum"},
		"web":    {"package.json", "next.config.js", "vite.config.ts"},
		"docker": {"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"},
		"ruby":   {"Gemfile", "Rakefile"},
		"python": {"requirements.txt", "setup.py", "pyproject.toml"},
		"rust":   {"Cargo.toml"},
	}

	for domain, files := range checks {
		for _, f := range files {
			if _, err := os.Stat(filepath.Join(root, f)); err == nil {
				domains = append(domains, domain)
				break
			}
		}
	}
	if detectMDSWorkspace(root) {
		domains = append(domains, "max-for-live", "mds")
	}
	sort.Strings(domains)
	return domains
}

func detectMDSWorkspace(root string) bool {
	for _, rel := range []string{
		"devices",
		"m4l_builder",
		filepath.Join("scripts", "mds"),
		"MaxForLive_Vault",
	} {
		if _, err := os.Stat(filepath.Join(root, rel)); err == nil {
			return true
		}
	}
	for _, rel := range []string{"AGENTS.md", filepath.Join(".aether", "AGENTS.md")} {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			continue
		}
		text := strings.ToLower(string(data))
		if strings.Contains(text, "max for live") || strings.Contains(text, "maxforlive") || strings.Contains(text, "mds") {
			return true
		}
	}
	return false
}

// scanHighSeverityOpen iterates all domain ledgers and collects warning strings
// for entries that are both open and HIGH severity.
func scanHighSeverityOpen(s *storage.Store) []string {
	var warnings []string
	for _, d := range colony.DomainOrder {
		ledgerPath := fmt.Sprintf("reviews/%s/ledger.json", d)
		var lf colony.ReviewLedgerFile
		if err := s.LoadJSON(ledgerPath, &lf); err != nil {
			continue
		}
		for _, e := range lf.Entries {
			if e.Status == "open" && e.Severity == colony.ReviewSeverityHigh {
				loc := ""
				if e.File != "" {
					loc = e.File
					if e.Line > 0 {
						loc = fmt.Sprintf("%s:%d", e.File, e.Line)
					}
				}
				if loc != "" {
					warnings = append(warnings, fmt.Sprintf("- [%s] %s: %s (%s)", d, e.ID, e.Description, loc))
				} else {
					warnings = append(warnings, fmt.Sprintf("- [%s] %s: %s", d, e.ID, e.Description))
				}
			}
		}
	}
	return warnings
}

func loadSealFinalReviewForSummary(s *storage.Store) *sealFinalReviewReport {
	if s == nil {
		return nil
	}
	var report sealFinalReviewReport
	if err := s.LoadJSON(sealFinalReviewReportRel, &report); err != nil {
		return nil
	}
	return &report
}

func collectOpenReviewBacklog(s *storage.Store, limit int) []colony.ReviewLedgerEntry {
	if s == nil || limit <= 0 {
		return nil
	}
	backlog := []colony.ReviewLedgerEntry{}
	for _, domain := range colony.DomainOrder {
		var lf colony.ReviewLedgerFile
		if err := s.LoadJSON(fmt.Sprintf("reviews/%s/ledger.json", domain), &lf); err != nil {
			continue
		}
		for _, entry := range lf.Entries {
			if entry.Status != "open" {
				continue
			}
			backlog = append(backlog, entry)
			if len(backlog) >= limit {
				return backlog
			}
		}
	}
	return backlog
}

// checkSealBlockers loads flags from pending-decisions.json (fallback flags.json),
// splits unresolved entries into blockers and issues, and appends any
// durable visual/runtime owner checkpoints as blocker-shaped entries at the
// seal boundary. Historical needs_owner_confirmation reports still receive
// their live compatibility blocker when no matching durable checkpoint was
// materialized by the newer continue runtime.
func checkSealBlockers(s *storage.Store, state colony.ColonyState) (blockers []colony.FlagEntry, issues []colony.FlagEntry) {
	const durabilityBlockerID = "pending-decisions-storage-unavailable"
	seenBlockers := map[string]bool{}
	durabilityFailureAdded := false
	appendBlocker := func(blocker colony.FlagEntry) {
		if blocker.ID != "" && seenBlockers[blocker.ID] {
			return
		}
		blockers = append(blockers, blocker)
		if blocker.ID != "" {
			seenBlockers[blocker.ID] = true
		}
	}
	appendPendingDecisionFailure := func(err error) {
		if err == nil || durabilityFailureAdded {
			return
		}
		failure := colony.FlagEntry{
			ID:              durabilityBlockerID,
			Type:            "blocker",
			Description:     fmt.Sprintf("Seal cannot trust %s because its required owner-work state could not be read, decoded, or durably updated: %v. Safe recovery command: aether patrol", pendingDecisionsFile, err),
			Source:          "pending_decision_storage",
			RecoveryCommand: "aether patrol",
		}
		// The durability ID is runtime-reserved. If an ordinary flag reused it,
		// replace that projection so it cannot hide the actual storage failure.
		for i := range blockers {
			if blockers[i].ID == durabilityBlockerID {
				blockers[i] = failure
				durabilityFailureAdded = true
				return
			}
		}
		appendBlocker(failure)
		durabilityFailureAdded = true
	}
	appendFlags := func(file colony.FlagsFile) {
		for _, f := range file.Decisions {
			if f.Resolved {
				continue
			}
			switch f.Type {
			case "blocker":
				appendBlocker(f)
			case "issue":
				issues = append(issues, f)
			}
		}
	}

	if s == nil {
		appendPendingDecisionFailure(fmt.Errorf("no store initialized"))
		return blockers, issues
	}
	ff, _, flagsErr := readCanonicalBlockerFlags(s)
	if flagsErr != nil {
		appendPendingDecisionFailure(flagsErr)
	} else {
		appendFlags(ff)
	}

	checkpointBlockers, checkpointErr := autopilotCheckpointSealBlockersFromStore(s, state)
	if checkpointErr != nil {
		appendPendingDecisionFailure(checkpointErr)
	}
	for _, blocker := range checkpointBlockers {
		appendBlocker(blocker)
	}
	checkpointCompatibilityIDs := map[string]bool{}
	if checkpointErr == nil {
		var compatibilityErr error
		checkpointCompatibilityIDs, compatibilityErr = autopilotCheckpointCompatibilityIDsFromStore(s, state)
		if compatibilityErr != nil {
			appendPendingDecisionFailure(compatibilityErr)
		}
	}
	for _, blocker := range ownerConfirmationSealBlockers(state) {
		if checkpointCompatibilityIDs[blocker.ID] {
			continue
		}
		appendBlocker(blocker)
	}
	return blockers, issues
}

// renderBlockerSummary formats blocker and issue flags in the classic headed
// style with resolution hints.
func renderBlockerSummary(blockers []colony.FlagEntry, issues []colony.FlagEntry) string {
	var b strings.Builder
	for _, entry := range blockers {
		b.WriteString(fmt.Sprintf("🚩 %s\n", strings.TrimSpace(entry.Description)))
		detail := entry.ID
		if entry.Type != "" {
			detail += ", " + entry.Type
		}
		if entry.CreatedAt != "" {
			detail += ", created " + entry.CreatedAt
		}
		b.WriteString("   └── " + detail + "\n")
	}
	b.WriteString("\nBLOCKED: Resolve blockers above or use --force to override.\n")
	for _, bl := range blockers {
		// A blocker with its own RecoveryCommand (e.g. an owner-confirmation
		// blocker, which is computed live and never written to
		// pending-decisions.json) is not resolvable through
		// `aether flag-resolve` -- that command can only look up IDs that
		// exist in the flags file, so printing it here would hand the
		// (non-technical) owner a command guaranteed to fail. Print the
		// blocker's real recovery command instead; only fall back to
		// flag-resolve for ordinary persisted flags (WR-01, 193-REVIEW.md).
		if bl.RecoveryCommand != "" {
			b.WriteString("  " + bl.RecoveryCommand + "\n")
			continue
		}
		b.WriteString(fmt.Sprintf("  aether flag-resolve --id %s\n", bl.ID))
	}
	if len(issues) > 0 {
		b.WriteString(fmt.Sprintf("\nNOTE: %d issue-severity flag(s) also unresolved.\n", len(issues)))
	}
	return b.String()
}

// countResolvedFlags loads the flags file and counts resolved entries.
func countResolvedFlags(s *storage.Store) int {
	var ff colony.FlagsFile
	if err := s.LoadJSON("pending-decisions.json", &ff); err != nil {
		if err2 := s.LoadJSON("flags.json", &ff); err2 != nil {
			return 0
		}
	}
	count := 0
	for _, f := range ff.Decisions {
		if f.Resolved {
			count++
		}
	}
	return count
}

// sealEnrichment holds data for enriching the CROWNED-ANTHILL.md summary.
type sealEnrichment struct {
	LearningsCount        int
	InstinctsPromoted     []string
	HiveEligible          int
	HivePromoted          int
	HivePromotionFailures int
	SignalsExpired        int
	FlagsResolved         int
	ShelfCandidates       []colony.ShelfEntry
	FinalReview           *sealFinalReviewReport
	ReviewBacklog         []colony.ReviewLedgerEntry
	// ConsolidationReport is the path to the scribe's persisted curation
	// report (LEARN-02, <.aether>/CURATION-REPORT.md), surfaced here so it
	// is discoverable from CROWNED-ANTHILL.md as well as from stdout.
	ConsolidationReport string
	// Override carries the owner's force-seal record, when one happened —
	// what was skipped and why, written into the summary permanently.
	Override sealOverride
}

func buildSealSummary(state colony.ColonyState, sealedAt string, warnings []string, enrichment sealEnrichment) string {
	goal := ""
	if state.Goal != nil {
		goal = *state.Goal
	}
	var b strings.Builder
	b.WriteString("# CROWNED-ANTHILL\n\n")
	b.WriteString(fmt.Sprintf("- Goal: %s\n", goal))
	b.WriteString(fmt.Sprintf("- Sealed at: %s\n", sealedAt))
	b.WriteString(fmt.Sprintf("- Completed phases: %d\n", len(state.Plan.Phases)))
	if state.CurrentPhase > 0 {
		b.WriteString(fmt.Sprintf("- Final phase: %d\n", state.CurrentPhase))
	}
	if enrichment.Override.overrodeAnything() {
		b.WriteString("\n## Owner Override (Force Seal)\n")
		b.WriteString("This colony was sealed by an explicit owner decision, not by its own verification finishing.\n")
		b.WriteString(fmt.Sprintf("- Reason: %s\n", enrichment.Override.Reason))
		if len(enrichment.Override.IncompletePhases) > 0 {
			b.WriteString(fmt.Sprintf("- Unverified phases (%d):\n", len(enrichment.Override.IncompletePhases)))
			for _, name := range enrichment.Override.IncompletePhases {
				b.WriteString("  - " + name + "\n")
			}
		}
		if enrichment.Override.OverriddenBlockers > 0 {
			b.WriteString(fmt.Sprintf("- Open blocker flags overridden: %d\n", enrichment.Override.OverriddenBlockers))
		}
		if enrichment.Override.OverriddenReviewBlocks > 0 {
			b.WriteString(fmt.Sprintf("- Final-review blocking findings overridden: %d\n", enrichment.Override.OverriddenReviewBlocks))
		}
	}
	// Add review warnings section if any high-severity open findings exist
	if len(warnings) > 0 {
		b.WriteString(fmt.Sprintf("\n## Review Warnings\nWARNING: %d high-severity unresolved finding(s):\n", len(warnings)))
		for _, w := range warnings {
			b.WriteString(w + "\n")
		}
	}

	if enrichment.FinalReview != nil {
		review := enrichment.FinalReview
		b.WriteString("\n## Final Review Evidence\n")
		b.WriteString(fmt.Sprintf("- Passed: %t\n", review.Passed))
		b.WriteString(fmt.Sprintf("- Workers reviewed: %d\n", len(review.Workers)))
		b.WriteString(fmt.Sprintf("- Structured findings captured: %d\n", len(review.Findings)))
		if len(review.LedgerWrites) > 0 {
			b.WriteString("- Ledger writes:")
			for _, domain := range colony.DomainOrder {
				if count := review.LedgerWrites[domain]; count > 0 {
					b.WriteString(fmt.Sprintf(" %s=%d", domain, count))
				}
			}
			b.WriteString("\n")
		}
		if review.QueenLearningsWritten > 0 {
			b.WriteString(fmt.Sprintf("- Reusable lessons promoted to QUEEN.md: %d\n", review.QueenLearningsWritten))
		}
		if len(review.BlockingIssues) > 0 {
			b.WriteString("- Blocking review issues:\n")
			for _, issue := range limitStrings(review.BlockingIssues, 5) {
				b.WriteString(fmt.Sprintf("  - %s\n", issue))
			}
		}
	}

	if len(enrichment.ReviewBacklog) > 0 {
		b.WriteString("\n## Post-Seal Review Backlog\n")
		for _, entry := range enrichment.ReviewBacklog {
			location := entry.File
			if location != "" && entry.Line > 0 {
				location = fmt.Sprintf("%s:%d", entry.File, entry.Line)
			}
			if location != "" {
				b.WriteString(fmt.Sprintf("- [%s/%s] %s: %s (%s)\n", entry.Agent, entry.Severity, entry.ID, entry.Description, location))
			} else {
				b.WriteString(fmt.Sprintf("- [%s/%s] %s: %s\n", entry.Agent, entry.Severity, entry.ID, entry.Description))
			}
		}
	}

	b.WriteString("\n## Phase Summary\n")
	for _, phase := range state.Plan.Phases {
		b.WriteString(fmt.Sprintf("- Phase %d: %s [%s]\n", phase.ID, phase.Name, phase.Status))
	}

	// Colony Statistics enrichment
	b.WriteString("\n## Colony Statistics\n")
	b.WriteString("| Metric | Count |\n|--------|-------|\n")
	b.WriteString(fmt.Sprintf("| Learnings captured | %d |\n", enrichment.LearningsCount))
	b.WriteString(fmt.Sprintf("| Instincts promoted | %d |\n", len(enrichment.InstinctsPromoted)))
	b.WriteString(fmt.Sprintf("| Hive-eligible instincts | %d |\n", enrichment.HiveEligible))
	b.WriteString(fmt.Sprintf("| Hive-promoted instincts | %d |\n", enrichment.HivePromoted))
	if enrichment.HivePromotionFailures > 0 {
		b.WriteString(fmt.Sprintf("| Hive promotion failures | %d |\n", enrichment.HivePromotionFailures))
	}
	b.WriteString(fmt.Sprintf("| FOCUS signals expired | %d |\n", enrichment.SignalsExpired))
	b.WriteString(fmt.Sprintf("| Flags resolved | %d |\n", enrichment.FlagsResolved))
	if enrichment.ConsolidationReport != "" {
		b.WriteString(fmt.Sprintf("| Curation report | %s |\n", enrichment.ConsolidationReport))
	}

	if len(enrichment.InstinctsPromoted) > 0 {
		b.WriteString("\n### Promoted Instincts\n")
		for _, id := range enrichment.InstinctsPromoted {
			b.WriteString(fmt.Sprintf("- %s\n", id))
		}
	}

	// Shelf candidates section
	if len(enrichment.ShelfCandidates) > 0 {
		b.WriteString(fmt.Sprintf("\n## Shelf Candidates\n%d shelf candidate(s) detected:\n", len(enrichment.ShelfCandidates)))
		for _, c := range enrichment.ShelfCandidates {
			auto := ""
			if c.AutoDetected {
				auto = " (auto-detected)"
			}
			b.WriteString(fmt.Sprintf("- [%s] %s%s\n", c.Category, c.Text, auto))
		}
	}

	b.WriteString(fmt.Sprintf("\n### Signal Cleanup\n- FOCUS signals expired: %d\n- REDIRECT signals preserved\n", enrichment.SignalsExpired))

	return b.String()
}

func updateSessionSummary(commandName, suggestedNext, summary string) {
	if store == nil {
		return
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err == nil {
		if _, syncErr := syncColonyArtifacts(state, colonyArtifactOptions{
			CommandName:   commandName,
			SuggestedNext: suggestedNext,
			Summary:       summary,
			HandoffTitle:  "Session Snapshot",
			WriteHandoff:  true,
		}); syncErr == nil {
			return
		}
	}

	var session colony.SessionFile
	if err := store.LoadJSON("session.json", &session); err != nil {
		return
	}
	session.LastCommand = commandName
	session.LastCommandAt = time.Now().UTC().Format(time.RFC3339)
	if suggestedNext != "" {
		session.SuggestedNext = suggestedNext
	}
	if summary != "" {
		session.Summary = summary
	}
	_ = store.SaveJSON("session.json", session)
}

func init() {
	layEggsCmd.Flags().String("repo-dir", "", "Path to the repository (default: $CWD)")
	layEggsCmd.Flags().String("home-dir", "", "User home directory (default: $HOME)")
	colonizeCmd.Flags().Bool("force-resurvey", false, "Refresh survey artifacts even when an existing survey is present")
	colonizeCmd.Flags().Bool("force", false, "Alias for --force-resurvey")
	colonizeCmd.Flags().Bool("plan-only", false, "Print the surveyor dispatch manifest without mutating colony state or spawning workers")
	colonizeCmd.Flags().Duration("worker-timeout", 0, "Override per-worker timeout for real surveyor dispatches (e.g. 5m)")
	planCmd.Flags().Bool("refresh", false, "Regenerate the plan even when an existing plan is already present")
	planCmd.Flags().Bool("force", false, "Alias for --refresh")
	planCmd.Flags().Bool("plan-only", false, "Print the planning dispatch manifest without mutating colony state or spawning workers")
	planCmd.Flags().Bool("repair-artifact", false, "Repair and validate dependency references in .aether/data/planning/phase-plan.json without rerunning workers")
	planCmd.Flags().String("depth", "", "Planning depth: fast, balanced, deep, or exhaustive")
	planCmd.Flags().String("planning-depth", "", "Task decomposition depth: light, standard, or deep")
	planCmd.Flags().String("verification-depth", "", "Verification depth: light, standard, or heavy")
	planCmd.Flags().Int("target", 0, "Planning confidence target 70-99 (default from depth preset)")
	planCmd.Flags().Int("max-iterations", 0, "Planning iteration budget 2-12 (default from depth preset)")
	planCmd.Flags().Bool("accept", false, "Accept the current best plan even if confidence is below target")
	planCmd.Flags().String("revision-type", "", "Why a refreshed plan is needed: manual, user_feedback, research, verification_failure, or scope_change")
	planCmd.Flags().String("revision-reason", "", "Traceable explanation for refreshing a plan after completed work")
	planCmd.Flags().StringArray("revision-evidence", nil, "Repository-relative evidence file supporting the revision (repeatable)")
	planCmd.Flags().StringArray("research", nil, "Repository-relative research document to plan from, e.g. a saved Oracle run under .aether/research (repeatable; persisted so later plan runs keep it)")
	planCmd.Flags().Bool("print-brief", false, "Print which context sections the planning workers would receive (present/absent, size). Reads state; mutates nothing")
	planCmd.Flags().Bool("full", false, "With --print-brief, also print each planning worker's assembled brief")
	planCmd.Flags().Bool("synthetic", false, "Skip real worker dispatch and use local synthesis only")
	planCmd.Flags().Duration("worker-timeout", 0, "Override per-worker timeout for real planning dispatches (e.g. 5m)")
	planFinalizeCmd.Flags().String("completion-file", "", "JSON file containing plan_manifest and external planning worker results")
	colonizeFinalizeCmd.Flags().String("completion-file", "", "JSON file containing colonize_manifest and external surveyor worker results")
	buildCmd.Flags().StringArray("task", nil, "Redispatch only the specified task ID (repeatable or comma-separated)")
	buildCmd.Flags().Bool("force", false, "Force redispatch of the current active phase after an interrupted build")
	buildCmd.Flags().Bool("plan-only", false, "Emit the build dispatch manifest for wrapper-spawned workers; opens a durable build attempt (superseded automatically on re-entry) but spawns nothing. For a pure read, use --print-brief")
	buildCmd.Flags().Bool("print-brief", false, "Print a ten-second checklist of which context sections arrived (present/absent, size, total against budget). Add --full for the raw assembled prompt. Reads state; mutates nothing")
	buildCmd.Flags().Bool("full", false, "With --print-brief, print the raw assembled prompt and composition table instead of the checklist. No effect without --print-brief")
	buildCmd.Flags().String("worker", "", "With --print-brief, print only the named worker's prompt")
	buildCmd.Flags().Bool("synthetic", false, "Skip real worker dispatch and use local synthesis only")
	buildCmd.Flags().Duration("worker-timeout", 0, "Override per-worker timeout for build dispatches (e.g. 15m)")
	buildCmd.Flags().Bool("light", false, "Force light review (skip heavy agents on intermediate phases)")
	buildCmd.Flags().Bool("heavy", false, "Force heavy review (full quality gauntlet on any phase)")
	buildCmd.Flags().String("verification-depth", "", "Verification depth: light, standard, or heavy")
	// The Queen's team choice. Supplied by the wrapper after it has read the
	// phase; omitted means the deterministic keyword engine decides, which is
	// what every caller did before judgement existed.
	buildCmd.Flags().Bool("no-checkin", false, "Skip the wrapper's pre-spawn team check-in pause (the runtime plan is unchanged)")
	buildCmd.Flags().Bool("checkin", false, "Force the wrapper's pre-spawn team check-in pause even for a decision-free one-worker build (D-14 owner override); conflicts with --no-checkin")
	buildCmd.Flags().StringArray("castes", nil, "Queen's proposed worker castes for this phase (repeatable or comma-separated). Safety castes the phase requires are added back automatically; the worker budget still applies")
	buildCmd.Flags().String("caste-reason", "", "One line summarising the whole team's choice, shown to the operator alongside the roster. This is NOT a per-worker reason -- a worker named in --castes with no matching --caste-why entry is refused by name even if --caste-reason is set. Use --caste-why for that.")
	buildCmd.Flags().StringArray("caste-why", nil, "One reason per proposed worker, as caste=reason (repeatable; the reason may itself contain '='). A worker named in --castes with no entry here, and not required by the phase, is refused by name rather than sent unexplained")
	buildCmd.Flags().StringArray("job-proposal", nil, "Queen coherent-job proposal as one JSON object (repeatable; fields: name, task_ids, owner_caste, relationship, benefit, owner_reason)")
	buildCmd.Flags().Int("circuit-breaker-threshold", 3, "Consecutive failures before circuit breaker trips for a worker (default: 3)")
	buildCmd.Flags().Bool("no-suggest", false, "Skip pheromone suggestion analysis during build")
	buildCmd.Flags().Bool("verbose", false, "Show full worker output (default: filtered summary)")
	buildFinalizeCmd.Flags().String("completion-file", "", "JSON file containing dispatch_manifest and external worker results")
	buildCompletionStageCmd.Flags().String("completion-file", "", "JSON file containing the accepted dispatch_manifest and external worker results")
	continueCmd.Flags().StringArray("reconcile-task", nil, "Mark one or more task IDs as manually reconciled before continue gating (repeatable or comma-separated)")
	continueCmd.Flags().StringArray("read-only-artifact", nil, "Record hash-verified read-only evidence for an artifact a reconciled task did not modify, as <task-id>:<path> (repeatable or comma-separated)")
	continueCmd.Flags().Bool("plan-only", false, "Print the continue verification/review manifest without mutating colony state or spawning review workers")
	continueCmd.Flags().Bool("light", false, "Force light review (skip heavy review agents)")
	continueCmd.Flags().Bool("heavy", false, "Force heavy review (full review gauntlet)")
	continueCmd.Flags().String("verification-depth", "", "Verification depth: light, standard, or heavy")
	continueCmd.Flags().Duration("worker-timeout", 0, "Override per-worker timeout for continue verification/review dispatches (e.g. 15m)")
	continueCmd.Flags().Duration("verification-timeout", 0, "Override deterministic verification command timeout (e.g. 30m); env: AETHER_CONTINUE_VERIFICATION_TIMEOUT")
	continueCmd.Flags().Bool("skip-watchers", false, "Skip AI reviewer workers for this run; the program's own checks (build, types, lint, tests) still run either way")
	// Continue is the expensive flow: every reviewer is a full agent run. The
	// Queen chooses the team after reading the phase; without a proposal the
	// keyword engine decides, as before.
	continueCmd.Flags().StringArray("castes", nil, "Queen's proposed review castes for this phase (repeatable or comma-separated). The Watcher and any review the phase requires are added back automatically")
	continueCmd.Flags().String("caste-reason", "", "One line summarising the whole review team's choice. This is NOT a per-worker reason -- a reviewer named in --castes with no matching --caste-why entry is refused by name even if --caste-reason is set. Use --caste-why for that.")
	continueCmd.Flags().StringArray("caste-why", nil, "One reason per proposed reviewer, as caste=reason (repeatable; the reason may itself contain '='). A reviewer named in --castes with no entry here, and not required by the phase, is refused by name rather than sent unexplained")
	continueCmd.Flags().Bool("synthetic", false, "Mark continue as synthetic (skip real agent workers, use provided results)")
	continueCmd.Flags().Bool("no-learn", false, "Only turns off the legacy learning-entry capture for this run (D-16, PRIV-05); observations and failure records are still written, so a blocked run still leaves a record of what broke -- see captureContinueMemory's doc comment")
	continueCmd.Flags().Bool("classic-ceremony", false, "Emit the heavy continue review manifest for wrapper-spawned classic ceremony reviewers")
	continueFinalizeCmd.Flags().String("completion-file", "", "JSON file containing continue_manifest and external review worker results")
	continueFinalizeCmd.Flags().StringArray("reconcile-task", nil, "Mark one or more task IDs as manually reconciled at finalize time, combined with any already recorded when the build was planned (repeatable or comma-separated); this command does not accept --read-only-artifact directly, so record that evidence earlier via continue --plan-only --read-only-artifact")
	continueFinalizeCmd.Flags().Duration("verification-timeout", 0, "Override deterministic verification command timeout (e.g. 30m); env: AETHER_CONTINUE_VERIFICATION_TIMEOUT")
	continueFinalizeCmd.Flags().Bool("no-learn", false, "Only turns off the legacy learning-entry capture for this run (D-16, PRIV-05); observations and failure records are still written, so a blocked run still leaves a record of what broke -- see captureContinueMemory's doc comment")
	skipPhaseCmd.Flags().Bool("force", false, "Confirm that the phase should be abandoned and marked complete")
	skipPhaseCmd.Flags().String("reason", "", "Audit reason for force-skipping the phase")
	sealCmd.Flags().Bool("force", false, "Owner override: seal past unverified phases, open blockers, and review blocks (recorded; requires --reason when it overrides anything)")
	sealCmd.Flags().String("reason", "", "Why the seal is being forced — recorded in the colony's history and CROWNED-ANTHILL.md")
	sealCmd.Flags().Bool("plan-only", false, "Print the final seal review manifest without mutating colony state or spawning workers")
	sealFinalizeCmd.Flags().String("completion-file", "", "JSON file containing seal_manifest and external review worker results")
	preferencesCmd.Flags().Bool("list", false, "List stored preferences")

	rootCmd.AddCommand(layEggsCmd)
	rootCmd.AddCommand(colonizeCmd)
	rootCmd.AddCommand(colonizeFinalizeCmd)
	rootCmd.AddCommand(planCmd)
	rootCmd.AddCommand(planFinalizeCmd)
	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(buildFinalizeCmd)
	rootCmd.AddCommand(buildCompletionStageCmd)
	rootCmd.AddCommand(continueCmd)
	rootCmd.AddCommand(continueFinalizeCmd)
	rootCmd.AddCommand(skipPhaseCmd)
	rootCmd.AddCommand(sealCmd)
	rootCmd.AddCommand(sealFinalizeCmd)
	rootCmd.AddCommand(focusCmd)
	rootCmd.AddCommand(redirectCmd)
	rootCmd.AddCommand(feedbackCmd)
	rootCmd.AddCommand(preferencesCmd)
}
