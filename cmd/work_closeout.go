package cmd

// This file is the D-03/D-05/D-07/D-08/CAP-066 wiring root cause fix
// (201-VERIFICATION.md): the closeout rendering layer (buildUnverifiedCloseoutDetails,
// buildVerifiedCloseoutDetails, renderResultFilePrecisionCard,
// recommendedActionForWorkOutcome) was individually correct and unit-tested,
// but no production call site ever supplied LifecycleCloseoutDetails.WorkOutcome
// from a real sealed build attempt. buildWorkCloseoutDetails is that one
// resolver: it turns a phase's own sealed build attempt into a verdict-
// carrying closeout, using only what the attempt itself recorded.
//
// Consumed by: buildCmd's direct-dispatch ending screen
// (cmd/codex_workflow_cmds.go) and the wrapper's build closeout
// (renderCeremonyCloseout, cmd/ceremony_cmd.go). Both call sites apply the
// resolved details through applyLifecycleCloseout and fall back to today's
// exact rendering when this resolver reports no verdict.

import (
	"fmt"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

// checkWorkOutcomeResultKey is the one new stable key codex_continue.go
// stores its derived verdict under, alongside the existing "verification"
// key, at every place it saves a continue result (201-20/D-05/D-06/D-07).
// checkWorkCloseoutDetails reads this key back -- it never recomputes the
// verdict itself.
const checkWorkOutcomeResultKey = "check_work_outcome"

// checkWorkOutcome is the check's single derivation of its own verdict
// (D-05/D-06/D-07), called once per continueAcceptVerifyAdvanceDecision a
// continue lane obtains (cmd/codex_continue.go). It maps from recorded facts
// only -- the decision's own verdict and partial-success flag, the
// verification report's executed-step count, and whether any step recorded
// that it timed out -- and is total over continueAdvanceVerdict's exact two
// values, so no branch silently returns the success verdict:
//
//   - advance, with nothing to verify (zero executed steps) -> no-change.
//   - advance, with the shared decision's own PartialSuccess flag set ->
//     partial (PartialSuccess already mirrors "operational issues were
//     recorded" -- see continueAcceptVerifyAdvanceDecision's doc comment).
//   - advance, otherwise -> success.
//   - block, with a step that recorded TimedOut (continueVerificationTimedOut,
//     the existing check) -> timeout.
//   - block, with zero executed steps -> interrupted: nothing ever ran to
//     produce a blocking reason, so the check itself never got underway.
//   - block, otherwise -> blocker.
func checkWorkOutcome(decision continueAcceptVerifyAdvanceDecision, verification codexContinueVerificationReport) colony.WorkOutcome {
	switch decision.Verdict {
	case continueAdvanceVerdictAdvance:
		if len(verification.Steps) == 0 {
			return colony.WorkOutcomeNoChange
		}
		if decision.PartialSuccess {
			return colony.WorkOutcomePartial
		}
		return colony.WorkOutcomeSuccess
	case continueAdvanceVerdictBlock:
		if continueVerificationTimedOut(verification) {
			return colony.WorkOutcomeTimeout
		}
		if len(verification.Steps) == 0 {
			return colony.WorkOutcomeInterrupted
		}
		return colony.WorkOutcomeBlocker
	default:
		// Unreachable given continueAdvanceVerdict's exact two declared
		// values, but refused by name -- the zero value -- rather than
		// silently defaulting to success.
		return ""
	}
}

// checkWorkOutcomeFromResult reads the verdict checkWorkOutcome already
// computed back off a continue result map, dual-typed: the in-process
// colony.WorkOutcome codex_continue.go stores directly, or the plain JSON
// string a completion file round-trips it through (WorkOutcome's own
// MarshalJSON/UnmarshalJSON discipline, pkg/colony/work_outcome.go).
func checkWorkOutcomeFromResult(result map[string]interface{}) (colony.WorkOutcome, bool) {
	raw, present := result[checkWorkOutcomeResultKey]
	if !present {
		return "", false
	}
	switch v := raw.(type) {
	case colony.WorkOutcome:
		return v, v.Valid()
	case string:
		outcome := colony.WorkOutcome(strings.TrimSpace(v))
		return outcome, outcome.Valid()
	default:
		return "", false
	}
}

// checkVerificationEvidenceFromValue converts a continue result's own
// verification steps into LifecycleVerification evidence, reusing the SAME
// dual-type view extraction renderContinueVerificationDetail already uses
// (verificationStepDetailViewsFromTyped/FromMap, cmd/codex_visuals.go) --
// never a second, independently typed reader. A skipped step names nothing
// to verify and is excluded, matching D-05's "the check closeout's evidence
// names the verification steps that actually ran."
func checkVerificationEvidenceFromValue(raw interface{}) []colony.LifecycleVerification {
	var views []verificationStepDetailView
	switch v := raw.(type) {
	case codexContinueVerificationReport:
		views = verificationStepDetailViewsFromTyped(v.Steps)
	case map[string]interface{}:
		steps, _ := v["steps"].([]interface{})
		views = verificationStepDetailViewsFromMap(steps)
	default:
		return nil
	}
	if len(views) == 0 {
		return nil
	}
	evidence := make([]colony.LifecycleVerification, 0, len(views))
	for _, view := range views {
		if view.Skipped {
			continue
		}
		evidence = append(evidence, colony.LifecycleVerification{
			Name:   emptyFallback(strings.TrimSpace(view.Name), "verification step"),
			Passed: view.Passed,
			Detail: strings.TrimSpace(view.Summary),
		})
	}
	return evidence
}

// checkBlockersFromResult converts a blocked continue result's own recorded
// blocking reasons (result["blocking_issues"], set on every blocked result
// map codex_continue.go saves) into LifecycleIssue blockers -- never a
// second, independent derivation of why the check blocked.
func checkBlockersFromResult(result map[string]interface{}) []colony.LifecycleIssue {
	reasons := stringSliceValue(result["blocking_issues"])
	if len(reasons) == 0 {
		return nil
	}
	blockers := make([]colony.LifecycleIssue, 0, len(reasons))
	for i, reason := range reasons {
		reason = strings.TrimSpace(reason)
		if reason == "" {
			continue
		}
		blockers = append(blockers, colony.LifecycleIssue{
			ID:      fmt.Sprintf("check-blocker-%d", i+1),
			Summary: reason,
		})
	}
	return blockers
}

// checkWorkOutcomeSummary is the plain-English wording for each verdict this
// check closeout can carry -- the single authority a caller reads rather
// than re-deriving its own phrase per verdict. A blocker names the first
// recorded blocking reason when one exists, matching
// buildBlockerStatusCloseoutDetails' own precedent of naming what stopped
// it whenever that is known.
func checkWorkOutcomeSummary(verdict colony.WorkOutcome, blockers []colony.LifecycleIssue) string {
	switch verdict {
	case colony.WorkOutcomeSuccess:
		return "The check passed cleanly and the phase advanced."
	case colony.WorkOutcomeNoChange:
		return "The check found nothing to verify -- the phase advanced with no checks to run."
	case colony.WorkOutcomePartial:
		return "The check advanced the phase, but recorded operational issues along the way."
	case colony.WorkOutcomeTimeout:
		return "A verification step ran out of time before the check could finish."
	case colony.WorkOutcomeInterrupted:
		return "The check stopped before any verification step ran."
	case colony.WorkOutcomeBlocker:
		if len(blockers) > 0 {
			return fmt.Sprintf("The check found something it could not get past: %s.", blockers[0].Summary)
		}
		return "The check found something it could not get past."
	default:
		return ""
	}
}

// checkWorkCloseoutDetails reads the already-stored verdict and the
// already-stored verification report off a continue result map -- it never
// recomputes checkWorkOutcome itself (D-05's "derive it once where the
// decision is made" rule). It reports ok=false when the result carries no
// stored verdict, so a caller holding an older result (from before this
// plan, or from a lane this plan did not wire) renders exactly what it
// rendered before this resolver existed.
func checkWorkCloseoutDetails(result map[string]interface{}) (LifecycleCloseoutDetails, bool) {
	if result == nil {
		return LifecycleCloseoutDetails{}, false
	}
	verdict, ok := checkWorkOutcomeFromResult(result)
	if !ok {
		return LifecycleCloseoutDetails{}, false
	}
	blockers := checkBlockersFromResult(result)
	return LifecycleCloseoutDetails{
		WorkOutcome:  verdict,
		Summary:      checkWorkOutcomeSummary(verdict, blockers),
		Verification: checkVerificationEvidenceFromValue(result["verification"]),
		Blockers:     blockers,
	}, true
}

// applyCheckWorkCloseout is the one shared render-time fold used by every
// check-screen call site (cmd/codex_workflow_cmds.go's continueCmd,
// cmd/ceremony_cmd.go's wrapper closeout): when result carries a stored
// verdict, fold it into result via applyLifecycleCloseout and append the
// verdict-carrying closeout (recommendation, evidence, and the one
// cost-and-time block) onto body. When no verdict resolves -- an older
// result, or a lane this plan did not wire -- body renders exactly as it did
// before this resolver existed, with its own plain cost line appended.
func applyCheckWorkCloseout(result map[string]interface{}, phaseID int, body string) string {
	if details, ok := checkWorkCloseoutDetails(result); ok {
		if err := applyLifecycleCloseout(result, "continue", details); err == nil {
			return appendLifecycleCloseoutVisual(body, result, detectPlatform())
		}
	}
	return appendSpendCostLine(body, phaseID)
}

// buildEndReviewerCastes is derived from queenBuildPostWavePlans (the ONLY
// route by which a post-wave reviewer becomes a build dispatch,
// cmd/codex_build.go) rather than a second, independently typed caste list
// that could silently drift from it.
var buildEndReviewerCastes = func() map[string]bool {
	castes := make(map[string]bool, len(queenBuildPostWavePlans))
	for _, plan := range queenBuildPostWavePlans {
		castes[plan.caste] = true
	}
	return castes
}()

// buildWorkCloseoutDetails resolves a phase's own sealed build attempt into
// a verdict-carrying LifecycleCloseoutDetails (D-05). It reports ok=false --
// never a fabricated verdict -- for a phase with no attempt at all, or one
// whose attempt has not reached a terminal status; the caller renders
// exactly what it rendered before this resolver existed.
//
// The mapping is total over every terminal attempt status
// (buildAttemptStatusTerminal's four values) and is derived ONLY from facts
// the attempt itself already recorded: its own status constant, its free-
// check report, its recorded verification-boundary decision (read, never
// re-derived, via verificationBoundaryForAttempt), and its dispatches' own
// terminal statuses (classified with the existing external-status helpers).
// Two carve-outs apply before the status switch, because neither an
// all-no-change build nor a worker timeout is fully described by the four
// attempt-level status constants alone:
//
//  1. every one of the attempt's terminal dispatches reported no change ->
//     the no-change verdict, regardless of the attempt's own status.
//  2. any dispatch's terminal status is a worker timeout -> the timeout
//     verdict, regardless of the attempt's own status.
//
// Otherwise the attempt's own status decides: built (further resolved by
// buildBuiltStatusCloseoutDetails below), partial, failed (-> blocker), or
// interrupted.
func buildWorkCloseoutDetails(phaseNum int) (LifecycleCloseoutDetails, bool) {
	attemptRel, attempt, ok := loadLatestBuildAttempt(phaseNum)
	if !ok || !buildAttemptStatusTerminal(attempt.Status) {
		return LifecycleCloseoutDetails{}, false
	}

	var details LifecycleCloseoutDetails
	switch {
	case buildDispatchesAllTerminalNoChange(attempt.Dispatches):
		details = buildNoChangeCloseoutDetails()
	case buildDispatchesAnyTimeout(attempt.Dispatches):
		details = buildTimeoutCloseoutDetails()
	case attempt.Status == buildAttemptBuilt:
		details = buildBuiltStatusCloseoutDetails(attemptRel, attempt)
	case attempt.Status == buildAttemptPartial:
		details = buildPartialStatusCloseoutDetails()
	case attempt.Status == buildAttemptFailed:
		details = buildBlockerStatusCloseoutDetails(attempt)
	case attempt.Status == buildAttemptInterrupted:
		details = buildInterruptedStatusCloseoutDetails()
	default:
		// Unreachable given buildAttemptStatusTerminal's exact four values,
		// but refused by name rather than silently defaulting to success.
		return LifecycleCloseoutDetails{}, false
	}

	// CAP-066: the attempt's own recorded decision/learning deltas reach the
	// card as evidence attached to these details, deduplicated against
	// buildLifecycleCloseout's own downstream knowledge-delta attachment
	// (cmd/lifecycle_closeout.go) by shared evidence ID so the same delta
	// never renders twice once this resolver supplies a real WorkOutcome.
	details.Evidence = append(details.Evidence, lifecycleCloseoutKnowledgeDeltaEvidence(attempt)...)
	return details, true
}

// buildBuiltStatusCloseoutDetails resolves the built-status case (D-03/D-05)
// -- every covered task is credited, but that says nothing about whether the
// program's own checks passed or whether a reviewer has judged the work.
// Checked in order: no check report at all (never claims checks passed);
// a check report that failed (blocker, naming the failed checks); a
// recorded build-end boundary whose build-end reviewers actually passed
// (the only way a build closeout may carry the success verdict); and
// otherwise the honest check-step default -- work is built, checks passed,
// not yet verified. No branch here falls through to success without a real
// passing reviewer dispatch.
func buildBuiltStatusCloseoutDetails(attemptRel string, attempt buildAttemptRecord) LifecycleCloseoutDetails {
	if attempt.FreeChecks == nil {
		return buildNoCheckReportCloseoutDetails()
	}
	if !attempt.FreeChecks.Passed {
		return buildFailingChecksCloseoutDetails(attempt)
	}
	if boundary, ok := verificationBoundaryForAttempt(attemptRel); ok &&
		boundary.Choice == verificationBoundaryChoiceBuildEnd &&
		buildEndReviewersPassed(attempt.Dispatches) {
		return buildVerifiedCloseoutDetails(attempt.FreeChecks.ChecksRun, buildEndReviewerNames(attempt.Dispatches))
	}
	return buildUnverifiedCloseoutDetails(attempt.FreeChecks.ChecksRun)
}

// buildNoCheckReportCloseoutDetails is the honest wording for a built
// attempt with no free-check report recorded at all (D-03's "never claims
// checks passed" guarantee): the work is built, and the program's own
// checks simply have not run yet -- never rendered as if they had.
func buildNoCheckReportCloseoutDetails() LifecycleCloseoutDetails {
	return LifecycleCloseoutDetails{
		WorkOutcome: colony.WorkOutcomePartial,
		Summary:     "The work is built. The program's own checks have not run yet.",
	}
}

// buildFailingChecksCloseoutDetails is the honest wording for a built
// attempt whose own free checks did not all pass: every completed task is
// credited, but the program's own checks are failing, which is a blocker --
// never a partial "not verified yet" result, and never success.
func buildFailingChecksCloseoutDetails(attempt buildAttemptRecord) LifecycleCloseoutDetails {
	report := attempt.FreeChecks
	summary := "The work is built, but the program's own checks did not all pass."
	var blockers []colony.LifecycleIssue
	if report != nil && len(report.Failed) > 0 {
		summary = fmt.Sprintf("The work is built, but the program's own checks did not all pass: %s.", strings.Join(report.Failed, ", "))
		blockers = make([]colony.LifecycleIssue, 0, len(report.Failed))
		for i, name := range report.Failed {
			blockers = append(blockers, colony.LifecycleIssue{
				ID:      fmt.Sprintf("build-check-failed-%d", i+1),
				Summary: fmt.Sprintf("`%s` failed", name),
			})
		}
	}
	return LifecycleCloseoutDetails{
		WorkOutcome: colony.WorkOutcomeBlocker,
		Summary:     summary,
		Blockers:    blockers,
	}
}

// buildPartialStatusCloseoutDetails is the attempt-level partial status
// (D-10: some, not all, covered tasks credited; a recovery job already
// covers the rest).
func buildPartialStatusCloseoutDetails() LifecycleCloseoutDetails {
	return LifecycleCloseoutDetails{
		WorkOutcome: colony.WorkOutcomePartial,
		Summary:     "The work only got partway through -- some of the tasks are still unfinished.",
	}
}

// buildBlockerStatusCloseoutDetails is the attempt-level failed status: the
// build could not be finished. attempt.Error, when recorded, names what
// stopped it.
func buildBlockerStatusCloseoutDetails(attempt buildAttemptRecord) LifecycleCloseoutDetails {
	summary := "The build hit something it could not get past."
	if reason := strings.TrimSpace(attempt.Error); reason != "" {
		summary = fmt.Sprintf("The build hit something it could not get past: %s.", reason)
	}
	return LifecycleCloseoutDetails{
		WorkOutcome: colony.WorkOutcomeBlocker,
		Summary:     summary,
	}
}

// buildInterruptedStatusCloseoutDetails is the attempt-level interrupted
// status: the build stopped before it reached a durable terminal result.
func buildInterruptedStatusCloseoutDetails() LifecycleCloseoutDetails {
	return LifecycleCloseoutDetails{
		WorkOutcome: colony.WorkOutcomeInterrupted,
		Summary:     "The build was stopped before it was done.",
	}
}

// buildNoChangeCloseoutDetails is the carve-out for a build whose every
// terminal dispatch reported no change: nothing needed changing, which is an
// honest success, not a partial result.
func buildNoChangeCloseoutDetails() LifecycleCloseoutDetails {
	return LifecycleCloseoutDetails{
		WorkOutcome: colony.WorkOutcomeNoChange,
		Summary:     "Nothing needed changing -- the work already matched what was asked for.",
	}
}

// buildTimeoutCloseoutDetails is the carve-out for a build whose failure was
// a worker running out of time, distinct from an ordinary blocker.
func buildTimeoutCloseoutDetails() LifecycleCloseoutDetails {
	return LifecycleCloseoutDetails{
		WorkOutcome: colony.WorkOutcomeTimeout,
		Summary:     "The build ran out of time before it finished.",
	}
}

// buildDispatchesAllTerminalNoChange reports whether every one of dispatches
// reached a terminal status AND that status was an honest no-change --
// vacuously false for an empty dispatch list, since "every dispatch" cannot
// be true of nothing.
func buildDispatchesAllTerminalNoChange(dispatches []codexBuildDispatch) bool {
	if len(dispatches) == 0 {
		return false
	}
	for _, dispatch := range dispatches {
		status := normalizeExternalBuildStatus(dispatch.Status)
		if !isTerminalExternalBuildStatus(status) || !isNoChangeExternalBuildStatus(status) {
			return false
		}
	}
	return true
}

// buildDispatchesAnyTimeout reports whether any dispatch's own terminal
// status is a worker timeout.
func buildDispatchesAnyTimeout(dispatches []codexBuildDispatch) bool {
	for _, dispatch := range dispatches {
		if normalizeExternalBuildStatus(dispatch.Status) == "timeout" {
			return true
		}
	}
	return false
}

// buildEndReviewerDispatches filters dispatches down to the ones a
// post-wave reviewer caste (buildEndReviewerCastes) produced.
func buildEndReviewerDispatches(dispatches []codexBuildDispatch) []codexBuildDispatch {
	var reviewers []codexBuildDispatch
	for _, dispatch := range dispatches {
		if buildEndReviewerCastes[strings.ToLower(strings.TrimSpace(dispatch.Caste))] {
			reviewers = append(reviewers, dispatch)
		}
	}
	return reviewers
}

// buildEndReviewersPassed reports whether build-end reviewers actually ran
// AND every one of them passed. An attempt with a build-end boundary but no
// reviewer dispatch at all (nothing selected to review) is never treated as
// "passed" -- there is nothing to have passed.
func buildEndReviewersPassed(dispatches []codexBuildDispatch) bool {
	reviewers := buildEndReviewerDispatches(dispatches)
	if len(reviewers) == 0 {
		return false
	}
	for _, reviewer := range reviewers {
		if !isSuccessfulExternalBuildStatus(normalizeExternalBuildStatus(reviewer.Status)) {
			return false
		}
	}
	return true
}

// buildEndReviewerNames names the build-end reviewers, in dispatch order, for
// the closeout's evidence trail.
func buildEndReviewerNames(dispatches []codexBuildDispatch) []string {
	reviewers := buildEndReviewerDispatches(dispatches)
	names := make([]string, 0, len(reviewers))
	for _, reviewer := range reviewers {
		if name := strings.TrimSpace(reviewer.Name); name != "" {
			names = append(names, name)
		}
	}
	return names
}

// renderBuildResultFileSection renders the credited/uncredited file card
// (D-08) for a phase's own sealed build attempt. It returns an empty string
// when the phase has no sealed attempt, or when that attempt recorded no
// credited and no uncredited files -- so a caller never renders a heading
// over nothing.
//
// Consumed by: buildCmd's direct-dispatch ending screen
// (cmd/codex_workflow_cmds.go) and the wrapper's build closeout
// (renderCeremonyCloseout, cmd/ceremony_cmd.go), both appending this card's
// output beneath the rest of the closeout body.
func renderBuildResultFileSection(phaseNum int) string {
	_, attempt, ok := loadLatestBuildAttempt(phaseNum)
	if !ok {
		return ""
	}
	if len(attempt.CreditedFiles) == 0 && len(attempt.UncreditedFiles) == 0 {
		return ""
	}
	return renderResultFilePrecisionCard(attempt)
}
