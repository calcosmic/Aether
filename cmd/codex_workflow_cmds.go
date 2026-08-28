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
		// The one cost line ends this lane's ending screen too. The
		// plan-only path above deliberately does NOT get one: nothing has
		// been spent yet when a team is merely being planned.
		outputWorkflow(result, appendSpendCostLine(
			renderBuildVisualWithDispatches(state, state.Plan.Phases[phaseNum-1], dispatches, reviewDepthBuild, queenPolicyFromResult(result)),
			phaseNum,
		))
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

		// The same readiness rules as the heavy path: with --force the
		// all-phases-completed rule becomes an owner override (recorded
		// with a reason), because sometimes the work was finished OUTSIDE
		// the colony, or the colony is wedged on its own gates, and the
		// owner's call to file the project away must win.
		state, incompletePhases, err := validateSealReady(forceFlag)
		if err != nil {
			renderRecoveryMenu("seal", err.Error(), nil)
			return nil
		}

		// Check for blocker-severity flags
		blockers, issues := checkSealBlockers(store, state)
		if len(blockers) > 0 {
			if !forceFlag {
				renderRecoveryMenu("seal", renderBlockerSummary(blockers, issues), nil)
				return nil
			}
			// --force: warn but continue
			visualFprintln(stdout, fmt.Sprintf("WARNING: Overriding %d blocker(s) with --force", len(blockers)))
		} else if len(issues) > 0 {
			visualFprintln(stdout, fmt.Sprintf("NOTE: %d unresolved issue-severity flag(s)", len(issues)))
		}

		override := sealOverride{Forced: forceFlag, Reason: strings.TrimSpace(forceReason), IncompletePhases: incompletePhases, OverriddenBlockers: len(blockers)}
		if forceFlag && (len(incompletePhases) > 0 || len(blockers) > 0) && override.Reason == "" {
			renderRecoveryMenu("seal", fmt.Sprintf("force-sealing overrides %d unverified phase(s) and %d open blocker(s) — a reason is required so the override is recorded honestly: rerun with `--reason \"why\"`", len(incompletePhases), len(blockers)), nil)
			return nil
		}

		return completeSealRuntime(state, override)
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

func completeSealRuntime(state colony.ColonyState, override sealOverride) error {
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
	visualFprint(stdout, renderSealConsolidationBeats(sealConsolidation))
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

	// Ceremony Step 3: Expire all FOCUS pheromones, preserve REDIRECT (D-03)
	expiredFOCUSCount := expireSignalsByType(store, "FOCUS")

	now := time.Now().UTC().Format(time.RFC3339)
	state.State = colony.StateCOMPLETED
	state.Milestone = "Crowned Anthill"
	state.MilestoneUpdatedAt = &now
	if override.overrodeAnything() {
		// A forced seal is a real event in the colony's history, not a
		// footnote: name what was skipped and why, so the Archaeologist and
		// anyone reading history sees an honest record.
		state.Events = append(trimmedEvents(state.Events), fmt.Sprintf(
			"%s|sealed_forced|seal|Colony force-sealed by owner (%d unverified phase(s), %d overridden blocker(s)): %s",
			now, len(override.IncompletePhases), override.OverriddenBlockers+override.OverriddenReviewBlocks, override.Reason,
		))
	} else {
		state.Events = append(trimmedEvents(state.Events), fmt.Sprintf("%s|sealed|seal|Colony sealed at Crowned Anthill", now))
	}

	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		outputError(2, fmt.Sprintf("failed to save colony state: %v", err), nil)
		return nil
	}

	// Shelf candidate detection (before archiving)
	candidates, _ := detectShelfCandidates(state, store)
	if len(candidates) > 0 {
		visualFprintln(stdout, shelfCandidateSummary(candidates))
	}

	// Scan for high-severity open findings before building summary
	warnings := scanHighSeverityOpen(store)
	finalReview := loadSealFinalReviewForSummary(store)
	reviewBacklog := collectOpenReviewBacklog(store, 10)

	// Archive reviews directory alongside CROWNED-ANTHILL.md
	aetherDir := filepath.Dir(store.BasePath())
	_ = copyDirIfExists(filepath.Join(filepath.Dir(store.BasePath()), "data", "reviews"), filepath.Join(aetherDir, "reviews-archive"))

	// Build enrichment data for CROWNED-ANTHILL.md
	enrichment := sealEnrichment{
		LearningsCount:        len(state.Memory.PhaseLearnings),
		InstinctsPromoted:     promotedInstinctNames,
		HiveEligible:          hiveEligibleCount,
		HivePromoted:          hivePromotedCount,
		HivePromotionFailures: hivePromotionFailures,
		SignalsExpired:        expiredFOCUSCount,
		FlagsResolved:         countResolvedFlags(store),
		ShelfCandidates:       candidates,
		FinalReview:           finalReview,
		ReviewBacklog:         reviewBacklog,
		ConsolidationReport:   sealConsolidation.ReportPath,
		Override:              override,
	}

	summaryPath := filepath.Join(aetherDir, "CROWNED-ANTHILL.md")
	summary := buildSealSummary(state, now, warnings, enrichment)
	if err := os.WriteFile(summaryPath, []byte(summary), 0644); err != nil {
		outputError(2, fmt.Sprintf("failed to write %s: %v", summaryPath, err), nil)
		return nil
	}
	emitLifecycleCeremony(events.CeremonyTopicChamberSeal, events.CeremonyPayload{
		Phase:     state.CurrentPhase,
		PhaseName: "Crowned Anthill",
		Status:    "sealed",
		Message:   "Colony sealed at Crowned Anthill",
		Completed: completedPhaseCount(state),
		Total:     len(state.Plan.Phases),
	}, "aether-seal")
	updateSessionSummary("seal", "aether entomb", "Colony sealed")

	// Hub registry (RECLAIM-02, non-blocking): the sealed colony's entry goes
	// inactive with its final goal recorded, so `aether registry-list` reads
	// as a true history of colonies on this machine.
	sealGoal := ""
	if state.Goal != nil {
		sealGoal = strings.TrimSpace(*state.Goal)
	}
	if _, regErr := upsertColonyRegistryEntry(filepath.Dir(filepath.Dir(store.BasePath())), sealGoal, nil, false); regErr != nil {
		fmt.Fprintf(os.Stderr, "warning: could not update hub registry at seal: %v\n", regErr)
	}

	result := map[string]interface{}{
		"sealed":    true,
		"milestone": state.Milestone,
		"summary":   summaryPath,
		"next":      "aether entomb",
	}
	if override.overrodeAnything() {
		result["force_sealed"] = true
		result["force_reason"] = override.Reason
		result["unverified_phases"] = override.IncompletePhases
		result["overridden_blockers"] = override.OverriddenBlockers + override.OverriddenReviewBlocks
	}
	addOrchestratorBoundaryGuidance(result, "seal", state, "aether entomb", nil)
	outputWorkflow(result, renderSealVisual(state, summaryPath))

	if shouldRenderVisualOutput(stdout) {
		writeVisualOutput(stdout, renderStageMarker("Post-Seal: Delivery Readiness"))
		readinessSummary := buildPorterReadinessSummary()
		writeVisualOutput(stdout, readinessSummary)
		writeVisualOutput(stdout, "\nRun `aether porter check` to validate and deliver.\n")
	}
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
			result, err := createPheromoneSignal(signalType, args[0], "user", "", "", 1.0, "")
			if err != nil {
				outputError(1, err.Error(), nil)
				return nil
			}
			priorityValue := signalPriorityValue(signalType)
			if signal, ok := result["signal"].(map[string]interface{}); ok {
				if persisted, ok := signal["priority"].(string); ok && strings.TrimSpace(persisted) != "" {
					priorityValue = persisted
				}
			}
			replaced, _ := result["replaced"].(bool)
			outputWorkflow(result, renderSignalVisual(signalType, args[0], priorityValue, replaced))
			return nil
		},
	}
}

func createPheromoneSignal(sigType, content, sourceFlag, reasonFlag, ttlFlag string, strength float64, priority string) (map[string]interface{}, error) {
	if sigType == "" || strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("signal type and content are required")
	}

	sigType = strings.ToUpper(sigType)
	switch sigType {
	case "FOCUS", "REDIRECT", "FEEDBACK":
	default:
		return nil, fmt.Errorf("invalid signal type %q", sigType)
	}

	if priority == "" {
		switch sigType {
		case "FOCUS":
			priority = "normal"
		case "REDIRECT":
			priority = "high"
		case "FEEDBACK":
			priority = "low"
		}
	}

	if strength == 0 {
		strength = 1.0
	}

	tmpCmd := &cobra.Command{}
	tmpCmd.Flags().String("type", sigType, "")
	tmpCmd.Flags().String("content", content, "")
	tmpCmd.Flags().String("priority", priority, "")
	tmpCmd.Flags().Float64("strength", strength, "")
	tmpCmd.Flags().String("source", sourceFlag, "")
	tmpCmd.Flags().String("reason", reasonFlag, "")
	tmpCmd.Flags().String("ttl", ttlFlag, "")

	var buf strings.Builder
	oldStdout := stdout
	stdout = &buf
	defer func() { stdout = oldStdout }()

	if err := pheromoneWriteCmd.RunE(tmpCmd, nil); err != nil {
		return nil, err
	}

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(buf.String()), &envelope); err != nil {
		return nil, fmt.Errorf("failed to parse pheromone-write result: %w", err)
	}
	if ok, _ := envelope["ok"].(bool); !ok {
		return nil, fmt.Errorf("failed to create pheromone signal")
	}
	result, _ := envelope["result"].(map[string]interface{})
	return result, nil
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
// outstanding needs_owner_confirmation criteria (D-05, 193-CONTEXT.md) as
// synthetic blocker-shaped entries -- computed live from each phase's
// persisted continue verification report, not from a second file on disk,
// so there is nothing extra to keep in sync when an owner answers one via
// `aether decision-answer`.
func checkSealBlockers(s *storage.Store, state colony.ColonyState) (blockers []colony.FlagEntry, issues []colony.FlagEntry) {
	var ff colony.FlagsFile
	if err := s.LoadJSON("pending-decisions.json", &ff); err == nil {
		for _, f := range ff.Decisions {
			if f.Resolved {
				continue
			}
			switch f.Type {
			case "blocker":
				blockers = append(blockers, f)
			case "issue":
				issues = append(issues, f)
			}
		}
	} else if err2 := s.LoadJSON("flags.json", &ff); err2 == nil {
		for _, f := range ff.Decisions {
			if f.Resolved {
				continue
			}
			switch f.Type {
			case "blocker":
				blockers = append(blockers, f)
			case "issue":
				issues = append(issues, f)
			}
		}
	}
	blockers = append(blockers, ownerConfirmationSealBlockers(state)...)
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
	continueCmd.Flags().Bool("no-learn", false, "Disable learning capture for this run (D-16, PRIV-05)")
	continueCmd.Flags().Bool("classic-ceremony", false, "Emit the heavy continue review manifest for wrapper-spawned classic ceremony reviewers")
	continueFinalizeCmd.Flags().String("completion-file", "", "JSON file containing continue_manifest and external review worker results")
	continueFinalizeCmd.Flags().StringArray("reconcile-task", nil, "Mark one or more task IDs as manually reconciled at finalize time, combined with any already recorded when the build was planned (repeatable or comma-separated); this command does not accept --read-only-artifact directly, so record that evidence earlier via continue --plan-only --read-only-artifact")
	continueFinalizeCmd.Flags().Duration("verification-timeout", 0, "Override deterministic verification command timeout (e.g. 30m); env: AETHER_CONTINUE_VERIFICATION_TIMEOUT")
	continueFinalizeCmd.Flags().Bool("no-learn", false, "Disable learning capture for this run (D-16, PRIV-05)")
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
