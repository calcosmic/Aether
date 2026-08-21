package cmd

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

// verifyOutOfBandCmd is the named, operator-invoked reconciliation command
// FIELD-05 requires: when real work happened OUTSIDE the build pipeline
// entirely (a hand edit, a pair session -- see
// .planning/todos/pending/2026-08-21-no-reentry-path-for-out-of-band-work.md),
// the runtime correctly refuses to fabricate the worker results a normal
// close would need, and until this command existed that refusal had no
// honest exit. This command re-verifies CURRENT disk state, fresh, against
// the phase's own success criteria, and only closes the stuck work when
// that fresh verification genuinely holds -- citing the verification, never
// a synthetic worker dispatch.
//
// Its shape follows worktree-reap's precedent exactly (cmd/worktree_reap.go,
// 191.1-CONTEXT.md D-09): default = report only, nothing written; --force =
// actually close. Destruction of trust in the ledger's provenance is just
// as real a hazard as destruction of a worktree, and gets the same
// boundary: reachable ONLY when a human explicitly types this command.
// TestVerifyOutOfBandHasNoLifecycleCaller
// (cmd/verify_out_of_band_reachability_test.go) proves that boundary
// structurally, reusing worktree-reap's own AST call-graph guard rather than
// a weaker, name-grep-based check.
var verifyOutOfBandCmd = &cobra.Command{
	Use:   "verify-out-of-band <phase>",
	Short: "Re-verify current disk state against a phase's real success criteria and close stuck work",
	Long: "Operator-only. Re-runs the repository's real build/type/lint/test commands and re-hashes the " +
		"phase's declared artifacts against CURRENT disk state -- not a prior worker's report -- and, if " +
		"everything the phase actually requires genuinely passes, closes the stuck build attempt (or advances " +
		"the phase directly if none exists) citing that fresh verification. It never fabricates a worker " +
		"result: a criterion that can only be honestly satisfied by a real worker's claims or an executed " +
		"watcher review is refused, not silently passed. By default it only reports what it would check and " +
		"whether it currently passes -- nothing is written until --force.",
	Args: cobra.ExactArgs(1),
	RunE: runVerifyOutOfBand,
}

func init() {
	rootCmd.AddCommand(verifyOutOfBandCmd)
	verifyOutOfBandCmd.Flags().Bool("force", false, "actually close the phase citing fresh verification; without this flag nothing is written")
	verifyOutOfBandCmd.Flags().Bool("acknowledge-legacy-criteria", false, "required in addition to --force when the phase has only free-prose success criteria with no structured checks; confirms you read them yourself")
	verifyOutOfBandCmd.Flags().Bool("json", false, "output structured JSON")
}

func runVerifyOutOfBand(cmd *cobra.Command, args []string) error {
	phaseNum, err := parsePositivePhaseArg(args[0])
	if err != nil {
		outputError(1, err.Error(), nil)
		return nil
	}
	force, _ := cmd.Flags().GetBool("force")
	acknowledgeLegacy, _ := cmd.Flags().GetBool("acknowledge-legacy-criteria")
	jsonOut, _ := cmd.Flags().GetBool("json")

	report, result, err := executeVerifyOutOfBand(cmd.Context(), phaseNum, force, acknowledgeLegacy)
	if err != nil {
		outputError(1, err.Error(), report)
		return nil
	}

	if !force {
		if jsonOut {
			outputOK(report)
		} else {
			renderOutOfBandReport(stdout, report, false)
		}
		return nil
	}

	if jsonOut {
		outputOK(map[string]interface{}{"report": report, "result": result})
		return nil
	}
	renderOutOfBandReport(stdout, report, true)
	visualFprintf(stdout, "Phase %d closed citing fresh out-of-band verification. Next: %s\n", phaseNum, result["next"])
	return nil
}

// outOfBandCriterionResult is one bound success criterion's fresh evidence
// evaluation -- the ceremony's own version of codexCriterionVerification
// (cmd/criterion_evidence.go), scoped to what re-checking current disk
// state can honestly establish rather than what a worker's build claims say.
type outOfBandCriterionResult struct {
	Criterion string `json:"criterion"`
	TaskID    string `json:"task_id,omitempty"`
	Passed    bool   `json:"passed"`
	// Unsupported marks a criterion whose Checks name "claims" or "watcher"
	// -- evidence this ceremony structurally cannot supply honestly, since
	// no worker ran and no watcher reviewed anything. Such a criterion is
	// always refused, never silently passed or silently skipped.
	Unsupported       bool     `json:"unsupported,omitempty"`
	ArtifactsVerified []string `json:"artifacts_verified,omitempty"`
	Evidence          []string `json:"evidence,omitempty"`
	BlockingIssues    []string `json:"blocking_issues,omitempty"`
}

// outOfBandReport is the ceremony's full report: exactly what --force would
// check and whether it currently passes. It is returned unchanged (with no
// close performed) by the default, report-only invocation, and mutates
// nothing itself -- CLAUDE.md's dry-run corollary.
type outOfBandReport struct {
	Phase       int    `json:"phase"`
	PhaseName   string `json:"phase_name"`
	GeneratedAt string `json:"generated_at"`
	// Policy is the phase's criterion evidence policy (bound / legacy /
	// not-required) -- see phaseCriterionEvidencePolicy.
	Policy                 string                     `json:"policy"`
	Steps                  []codexVerificationStep    `json:"steps"`
	Criteria               []outOfBandCriterionResult `json:"criteria,omitempty"`
	LegacySuccessCriteria  []string                   `json:"legacy_success_criteria,omitempty"`
	RequiresAcknowledgment bool                       `json:"requires_acknowledgment,omitempty"`
	Acknowledged           bool                       `json:"acknowledged,omitempty"`
	Passed                 bool                       `json:"passed"`
	BlockingIssues         []string                   `json:"blocking_issues,omitempty"`
	HasExistingAttempt     bool                       `json:"has_existing_attempt"`
	ExistingAttemptStatus  string                     `json:"existing_attempt_status,omitempty"`
}

// executeVerifyOutOfBand is the whole ceremony: gather fresh evidence, and
// -- only when force is true and that evidence genuinely satisfies the
// phase -- close it. This is the SAME function runVerifyOutOfBand's RunE
// calls and this file's own tests call directly; there is no separate
// test-only path that could silently drift from what actually runs.
func executeVerifyOutOfBand(ctx context.Context, phaseNum int, force, acknowledgeLegacy bool) (outOfBandReport, map[string]interface{}, error) {
	if store == nil {
		return outOfBandReport{}, nil, fmt.Errorf("no store initialized")
	}
	state, err := loadActiveColonyState()
	if err != nil {
		return outOfBandReport{}, nil, fmt.Errorf("%s", colonyStateLoadMessage(err))
	}
	if phaseNum < 1 || phaseNum > len(state.Plan.Phases) {
		return outOfBandReport{}, nil, fmt.Errorf("phase %d not found (plan has %d phases)", phaseNum, len(state.Plan.Phases))
	}
	phase := state.Plan.Phases[phaseNum-1]
	root := resolveAetherRoot()

	report := gatherOutOfBandEvidence(ctx, root, phase, acknowledgeLegacy)
	attemptRel, attemptRecord, hasAttempt := loadLatestBuildAttempt(phaseNum)
	report.HasExistingAttempt = hasAttempt
	if hasAttempt {
		report.ExistingAttemptStatus = attemptRecord.Status
	}

	if !force {
		return report, nil, nil
	}
	if !report.Passed {
		return report, nil, outOfBandRefusalError(phaseNum, report)
	}

	result, err := closeOutOfBandCeremony(phaseNum, state, report, attemptRel, attemptRecord, hasAttempt, acknowledgeLegacy)
	if err != nil {
		return report, nil, err
	}
	return report, result, nil
}

// gatherOutOfBandEvidence runs the SAME fresh-evidence gathering regardless
// of --force (D-10): the repository's real build/types/lint/tests commands,
// run live via runVerificationStep (cmd/codex_continue.go) -- the same
// primitive continue already trusts -- and, for a bound phase, hash-
// verification of every declared artifact against disk NOW. It writes
// nothing; only executeVerifyOutOfBand's close path (guarded by force AND
// report.Passed) mutates anything.
func gatherOutOfBandEvidence(ctx context.Context, root string, phase colony.Phase, acknowledgeLegacy bool) outOfBandReport {
	if ctx == nil {
		ctx = context.Background()
	}
	now := time.Now().UTC()
	policy := phaseCriterionEvidencePolicy(phase)
	commands := resolveCodexVerificationCommands(root)
	requiredChecks := requiredVerificationChecks(phase)
	timeout := effectiveContinueVerificationTimeout(0)
	steps := []codexVerificationStep{
		runVerificationStep(ctx, root, "build", requiredChecks["build"], commands.Build, timeout),
		runVerificationStep(ctx, root, "types", requiredChecks["types"], commands.Type, timeout),
		runVerificationStep(ctx, root, "lint", requiredChecks["lint"], commands.Lint, timeout),
		runVerificationStep(ctx, root, "tests", requiredChecks["tests"], commands.Test, timeout),
	}
	steps = applyExpectedTestFailure(steps, phase)

	report := outOfBandReport{
		Phase:       phase.ID,
		PhaseName:   phase.Name,
		GeneratedAt: now.Format(time.RFC3339),
		Policy:      policy,
		Steps:       steps,
	}

	if policy == criterionEvidencePolicyBoundV1 {
		// WR-03 (191.1-REVIEW.md): mirror evaluatePhaseCriterionEvidence's own
		// first line (cmd/criterion_evidence.go) so this ceremony can never
		// evaluate -- and vacuously pass -- a structurally malformed
		// requirement (e.g. neither Artifacts nor Checks). aether continue
		// already refuses this exact shape outright; reaching it here at all
		// requires bypassing plan acceptance (a hand-edited or migrated
		// COLONY_STATE.json), but this ceremony must agree with continue's
		// refusal rather than silently rubber-stamping what continue would
		// refuse.
		if err := validatePhaseCriterionEvidence(phase); err != nil {
			report.Passed = false
			report.BlockingIssues = []string{err.Error()}
			return report
		}
		report.Criteria = evaluateOutOfBandBoundCriteria(root, phase, steps)
		report.Passed = true
		for _, c := range report.Criteria {
			if c.Passed {
				continue
			}
			report.Passed = false
			report.BlockingIssues = append(report.BlockingIssues, fmt.Sprintf(
				"criterion %q%s: %s", c.Criterion, criterionTaskSuffix(c.TaskID), strings.Join(c.BlockingIssues, "; ")))
		}
		return report
	}

	// Legacy/unbound (including the zero-criteria "not required" case): no
	// structured binding exists, so passing shell commands alone cannot
	// prove the phase's real acceptance criteria are met. An explicit,
	// separate operator acknowledgment is required in addition to --force
	// (T-191.1-03-04) -- and the prose criteria are always surfaced so the
	// operator is confirming something they were shown, not something
	// assumed on their behalf.
	for _, v := range unboundCriterionVerifications(phase) {
		report.LegacySuccessCriteria = append(report.LegacySuccessCriteria, v.Criterion)
	}
	report.RequiresAcknowledgment = true
	report.Acknowledged = acknowledgeLegacy
	shellIssues := outOfBandShellChecksAllPassed(steps)
	report.BlockingIssues = append(report.BlockingIssues, shellIssues...)
	if !acknowledgeLegacy {
		report.BlockingIssues = append([]string{
			"this phase has only free-prose success criteria with no structured checks; pass --acknowledge-legacy-criteria after reading them yourself -- passing shell commands alone cannot confirm prose criteria are met",
		}, report.BlockingIssues...)
	}
	report.Passed = acknowledgeLegacy && len(shellIssues) == 0
	return report
}

// evaluateOutOfBandBoundCriteria evaluates every bound CriterionEvidenceRequirement
// against CURRENT disk state: artifacts are hash-verified fresh right now
// (reusing snapshotBuildArtifact/attachBuildArtifactEvidence,
// cmd/criterion_evidence.go -- the same evidence primitive continue already
// trusts, D-10), never compared against a worker's prior claim, since no
// worker made one. A requirement naming "claims" or "watcher" is refused
// outright -- this ceremony cannot honestly supply either.
func evaluateOutOfBandBoundCriteria(root string, phase colony.Phase, steps []codexVerificationStep) []outOfBandCriterionResult {
	requirements := flattenPhaseCriterionEvidenceRequirements(phase)

	var allArtifacts []string
	for _, requirement := range requirements {
		allArtifacts = append(allArtifacts, requirement.Artifacts...)
	}
	seedClaims := &codexBuildClaims{FilesCreated: uniqueSortedStrings(allArtifacts)}
	attachBuildArtifactEvidence(root, seedClaims)
	evidenceByPath := map[string]codexBuildArtifactEvidence{}
	for _, item := range seedClaims.ArtifactEvidence {
		evidenceByPath[filepath.ToSlash(strings.TrimSpace(item.Path))] = item
	}

	results := make([]outOfBandCriterionResult, 0, len(requirements))
	for _, requirement := range requirements {
		result := outOfBandCriterionResult{
			Criterion: requirement.Criterion,
			TaskID:    requirement.TaskID,
			Passed:    true,
		}

		if unsupported := outOfBandUnsupportedChecks(requirement); len(unsupported) > 0 {
			result.Passed = false
			result.Unsupported = true
			for _, check := range unsupported {
				result.BlockingIssues = append(result.BlockingIssues, fmt.Sprintf(
					"requires a %q check -- this ceremony re-verifies disk state, it cannot honestly supply a real worker's claims or an executed watcher review; run `aether build`/`aether continue` for this phase instead",
					check))
			}
			results = append(results, result)
			continue
		}

		for _, artifact := range requirement.Artifacts {
			item, ok := evidenceByPath[artifact]
			if !ok {
				result.Passed = false
				result.BlockingIssues = append(result.BlockingIssues, fmt.Sprintf("artifact %s is missing or unreadable on disk right now", artifact))
				continue
			}
			result.ArtifactsVerified = append(result.ArtifactsVerified, artifact)
			result.Evidence = append(result.Evidence, fmt.Sprintf("artifact %s sha256:%s (hashed fresh against current disk state)", artifact, item.SHA256))
		}

		for _, check := range requirement.Checks {
			passed, evidence, issue := evaluateCriterionCheck(check, steps, codexClaimVerification{}, codexWatcherVerification{})
			if passed {
				result.Evidence = append(result.Evidence, evidence)
				continue
			}
			result.Passed = false
			result.BlockingIssues = append(result.BlockingIssues, issue)
		}

		results = append(results, result)
	}
	return results
}

// outOfBandUnsupportedChecks returns which of a requirement's declared
// checks name evidence this ceremony structurally cannot supply: "claims"
// (a real worker's build claims) or "watcher" (an executed watcher review).
func outOfBandUnsupportedChecks(requirement colony.CriterionEvidenceRequirement) []string {
	var unsupported []string
	for _, check := range requirement.Checks {
		check = strings.ToLower(strings.TrimSpace(check))
		if check == "claims" || check == "watcher" {
			unsupported = append(unsupported, check)
		}
	}
	return unsupported
}

// outOfBandShellChecksAllPassed reports which of the four fresh shell steps
// genuinely failed or were blocked -- a step that simply had no command to
// run (Skipped, not Blocked) is not held against a legacy phase, matching
// the same "enrichment, not gate" semantics runVerificationStep already
// uses elsewhere, but any step that DID run and did NOT pass disqualifies
// the ceremony: this is what keeps a legacy-phase close from being a rubber
// stamp.
func outOfBandShellChecksAllPassed(steps []codexVerificationStep) []string {
	var issues []string
	for _, step := range steps {
		if step.Blocked {
			issues = append(issues, fmt.Sprintf("%s check is blocked: %s", step.Name, step.Summary))
			continue
		}
		if step.Skipped {
			continue
		}
		if !step.Passed {
			issues = append(issues, fmt.Sprintf("%s check failed: %s", step.Name, step.Summary))
		}
	}
	return issues
}

// outOfBandRefusalError builds the human-legible refusal error for a
// --force invocation whose fresh evidence did not pass. The blocking issues
// are folded directly into the error text so both the JSON envelope and the
// plain-text render carry the same real information -- no separate,
// vaguer "refused" summary that omits what actually failed.
func outOfBandRefusalError(phaseNum int, report outOfBandReport) error {
	if len(report.BlockingIssues) == 0 {
		return fmt.Errorf("verify-out-of-band refused for phase %d: fresh verification did not pass", phaseNum)
	}
	return fmt.Errorf("verify-out-of-band refused for phase %d: fresh verification did not pass -- %s", phaseNum, strings.Join(report.BlockingIssues, "; "))
}

// closeOutOfBandCeremony is the ceremony's ONE state-mutating close: it is
// the only function in this codebase that both (a) closes a build attempt
// carrying out-of-band provenance (closeBuildAttemptOutOfBand,
// cmd/build_attempt.go) and (b) calls advancePhase with
// Source: "verify-out-of-band". Only executeVerifyOutOfBand calls it, and
// only after confirming report.Passed is true under --force.
// TestVerifyOutOfBandHasNoLifecycleCaller
// (cmd/verify_out_of_band_reachability_test.go) proves this function is
// unreachable from every automatic lifecycle entry point, under this exact
// name, via the same AST call-graph guard worktree-reap's own reachability
// ratchet uses.
//
// CR-03 (191.1-REVIEW.md): advancePhase runs FIRST, and the build attempt
// (an irreversible write -- Recoverable:false, RecoveryCommand:"") is only
// closed AFTER advancePhase has actually committed. advancePhase's own
// validateRuntimeStateStillCurrent is the ONE currency check this whole
// ceremony ultimately depends on (a paused colony, a stale phase/build
// identity -- any of the same reasons this phase's own supersession
// machinery exists to catch); running it before the irreversible write means
// a refusal here leaves the build attempt exactly as it was -- still
// recoverable -- instead of permanently sealed with the phase never having
// advanced at all. See TestCloseOutOfBandCeremonyNeverSealsWithoutAdvancing
// (cmd/verify_out_of_band_test.go) for the reproduction this ordering fixes.
func closeOutOfBandCeremony(phaseNum int, state colony.ColonyState, report outOfBandReport, attemptRel string, attemptRecord buildAttemptRecord, hasAttempt bool, acknowledgeLegacy bool) (map[string]interface{}, error) {
	now := time.Now().UTC()
	provenance := buildOutOfBandProvenance(now, phaseNum, report, acknowledgeLegacy)

	advanceResult, err := advancePhase(advancePhaseParams{
		PhaseID:                phaseNum,
		ExpectedBuildStartedAt: state.BuildStartedAt,
		AllowedStates:          []colony.State{colony.StateEXECUTING, colony.StateBUILT},
		Source:                 "verify-out-of-band",
		Now:                    now,
	})
	if err != nil {
		return nil, err
	}

	attemptClosed := false
	if hasAttempt && strings.TrimSpace(attemptRecord.Status) != buildAttemptBuilt {
		if err := closeBuildAttemptOutOfBand(attemptRel, provenance); err != nil {
			// The phase itself has ALREADY advanced by this point -- it is
			// genuinely complete, not stuck. Only the build attempt
			// journal's out-of-band marker failed to write, a much smaller,
			// self-describing failure that must never be reported as though
			// the whole ceremony failed.
			return nil, fmt.Errorf("phase %d advanced, but recording the out-of-band provenance on its build attempt failed (the phase is NOT stuck; only the build attempt journal entry needs reconciling): %w", phaseNum, err)
		}
		attemptClosed = true
	}

	result := map[string]interface{}{
		"phase":          phaseNum,
		"closed":         true,
		"advanced":       true,
		"attempt_closed": attemptClosed,
		"state":          advanceResult.Updated.State,
		"next":           advanceResult.NextCommand,
	}
	if advanceResult.NextPhase != nil {
		result["next_phase"] = advanceResult.NextPhase.ID
		result["next_phase_name"] = advanceResult.NextPhase.Name
	}
	return result, nil
}

// buildOutOfBandProvenance assembles the durable provenance record
// (cmd/build_attempt.go's outOfBandVerificationRecord) from what this pass
// actually checked -- never from what a worker claimed, since no worker ran.
func buildOutOfBandProvenance(now time.Time, phaseNum int, report outOfBandReport, acknowledgeLegacy bool) outOfBandVerificationRecord {
	var checksRun []string
	for _, step := range report.Steps {
		if step.Skipped {
			continue
		}
		checksRun = append(checksRun, step.Name)
	}
	var criteriaChecked []string
	var artifactsHashed []string
	for _, c := range report.Criteria {
		criteriaChecked = append(criteriaChecked, c.Criterion)
		artifactsHashed = append(artifactsHashed, c.ArtifactsVerified...)
	}
	criteriaChecked = append(criteriaChecked, report.LegacySuccessCriteria...)

	summary := "fresh verification against current disk state, re-run at close time -- not a worker dispatch"
	acknowledged := false
	if report.Policy != criterionEvidencePolicyBoundV1 {
		summary = "fresh shell verification plus explicit operator acknowledgment of free-prose success criteria -- not a worker dispatch"
		acknowledged = acknowledgeLegacy
	}

	return outOfBandVerificationRecord{
		VerifiedAt:         now.Format(time.RFC3339Nano),
		Phase:              phaseNum,
		Policy:             report.Policy,
		CriteriaChecked:    uniqueSortedStrings(criteriaChecked),
		ChecksRun:          uniqueSortedStrings(checksRun),
		ArtifactsHashed:    uniqueSortedStrings(artifactsHashed),
		AcknowledgedLegacy: acknowledged,
		Summary:            summary,
	}
}

// renderOutOfBandReport writes a plain-language rendering of the report for
// a human operator -- what was checked, what passed or failed, and (for a
// report-only run) that nothing was changed.
func renderOutOfBandReport(w io.Writer, report outOfBandReport, forced bool) {
	visualFprintf(w, "Out-of-band verification -- phase %d: %s\n", report.Phase, report.PhaseName)
	visualFprintln(w, "This re-checks CURRENT disk state against the phase's own success criteria, fresh -- it never trusts a prior worker record.")
	visualFprintln(w, "")
	for _, step := range report.Steps {
		status := "passed"
		switch {
		case step.Blocked:
			status = "BLOCKED"
		case step.Skipped:
			status = "skipped"
		case !step.Passed:
			status = "FAILED"
		}
		visualFprintf(w, "  %-7s %s -- %s\n", status, step.Name, step.Summary)
	}
	visualFprintln(w, "")
	if report.Policy == criterionEvidencePolicyBoundV1 {
		for _, c := range report.Criteria {
			mark := "PASS"
			if !c.Passed {
				mark = "FAIL"
			}
			visualFprintf(w, "  [%s] %s%s\n", mark, c.Criterion, criterionTaskSuffix(c.TaskID))
			for _, issue := range c.BlockingIssues {
				visualFprintf(w, "        %s\n", issue)
			}
		}
	} else {
		visualFprintln(w, "This phase has only free-prose success criteria -- no file or command is bound to check them. Read them yourself:")
		if len(report.LegacySuccessCriteria) == 0 {
			visualFprintln(w, "  (none declared)")
		}
		for _, criterion := range report.LegacySuccessCriteria {
			visualFprintf(w, "  - %s\n", criterion)
		}
		if !report.Acknowledged {
			visualFprintln(w, "Refusing without --acknowledge-legacy-criteria: passing shell commands alone cannot confirm prose criteria like these are met.")
		}
	}
	if report.HasExistingAttempt {
		visualFprintf(w, "\nExisting build attempt on this phase: %s\n", report.ExistingAttemptStatus)
	}
	if len(report.BlockingIssues) > 0 {
		visualFprintln(w, "\nBlocking:")
		for _, issue := range report.BlockingIssues {
			visualFprintf(w, "  - %s\n", issue)
		}
	}
	visualFprintln(w, "")
	if !forced {
		if report.Passed {
			visualFprintln(w, "This currently passes. Nothing was changed -- add --force to actually close this phase citing this verification.")
		} else {
			visualFprintln(w, "This does not currently pass. Nothing was changed.")
		}
	}
}
