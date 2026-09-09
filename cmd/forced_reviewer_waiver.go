package cmd

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// phaseDispatchWindowFileName holds, per phase, the moment worker dispatch
// actually began -- CR-01's residual (194-REVIEW.md iteration 2). The
// original CR-01 fix closed the forgery path (a --question with no runtime-
// created row is refused) but left the runtime-created row open, resolvable,
// for the ENTIRE build: the owner's default answer ("proceed") never calls
// decision-answer at all, so the pending row sat open exactly as long after
// the owner kept the reviewer as before they answered, and anything able to
// invoke the aether binary during the build (a worker's own Bash tool, a
// stray script, a prompt-injected instruction) could run the exact command
// the card legitimately displayed and silently waive a reviewer the owner
// never declined. This file closes that window: once dispatch has begun for
// a phase, no later decision-answer call can resolve that phase's
// forced-reviewer row, no matter who calls it or how correctly it is shaped.
const phaseDispatchWindowFileName = "phase-dispatch-started.json"

// phaseDispatchWindowFile is the on-disk shape of phaseDispatchWindowFileName:
// phase number (as a string map key -- JSON object keys are always strings)
// -> the RFC3339Nano time dispatch first began for that phase.
type phaseDispatchWindowFile struct {
	Phases map[string]string `json:"phases"`
}

func loadPhaseDispatchWindowFile() phaseDispatchWindowFile {
	var file phaseDispatchWindowFile
	if store != nil {
		_ = store.LoadJSON(phaseDispatchWindowFileName, &file)
	}
	if file.Phases == nil {
		file.Phases = map[string]string{}
	}
	return file
}

// phaseDispatchStartedAt reports the recorded dispatch-start time for
// phaseID, and whether one has been recorded at all. No record means
// dispatch has not yet begun for that phase (or this build never reached the
// point that records one, e.g. it failed before any worker spawned).
func phaseDispatchStartedAt(phaseID int) (time.Time, bool) {
	if phaseID <= 0 {
		return time.Time{}, false
	}
	file := loadPhaseDispatchWindowFile()
	raw := strings.TrimSpace(file.Phases[strconv.Itoa(phaseID)])
	if raw == "" {
		return time.Time{}, false
	}
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return t, true
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, true
	}
	return time.Time{}, false
}

// closeForcedReviewerWaiverWindowForPhase is called the moment dispatch has
// genuinely begun for phaseID -- from spawn-log (cmd/spawn.go), on every
// worker spawn, the runtime's own record that a worker is about to be told
// to run. This is the earliest point in the runtime that fires AFTER the
// owner has seen the check-in card (and answered it, or chose to proceed
// without answering) and BEFORE any worker's own Bash tool could possibly
// execute a command. It does two things:
//
//  1. Records the dispatch-start time for this phase, keeping the EARLIEST
//     one if called more than once (one call per worker spawned in the same
//     build) -- this is what forcedReviewerWaiver checks below.
//  2. Removes every still-PENDING (unresolved) forced-reviewer waiver row
//     for this phase. A later decision-answer call for that exact question
//     text now finds no matching row and is refused by the SAME path CR-01
//     already built for "no card was ever rendered" -- resolveForcedReviewer-
//     WaiverPendingDecision requires a matching unresolved row and creates
//     nothing new. A row the owner ALREADY resolved (a genuine decline made
//     before this call ever ran) is left untouched -- the owner's real path
//     must keep working.
//
// The marker write is authoritative and fail-closed: spawn-log must refuse
// the worker if this function cannot durably record the closed window. Pending
// row deletion remains defense in depth after that marker is guaranteed.
func closeForcedReviewerWaiverWindowForPhase(phaseID int, at time.Time) error {
	if store == nil {
		return fmt.Errorf("no store initialized")
	}
	if phaseID <= 0 {
		return fmt.Errorf("phase must be positive")
	}

	key := strconv.Itoa(phaseID)
	var windowFile phaseDispatchWindowFile
	if err := store.UpdateJSONAtomically(phaseDispatchWindowFileName, &windowFile, func() error {
		if windowFile.Phases == nil {
			windowFile.Phases = map[string]string{}
		}
		if raw := strings.TrimSpace(windowFile.Phases[key]); raw != "" {
			if _, err := time.Parse(time.RFC3339Nano, raw); err != nil {
				return fmt.Errorf("phase %d has invalid dispatch-start marker %q: %w", phaseID, raw, err)
			}
			return nil
		}
		windowFile.Phases[key] = at.UTC().Format(time.RFC3339Nano)
		return nil
	}); err != nil {
		return fmt.Errorf("persist dispatch-start marker for phase %d: %w", phaseID, err)
	}

	// The durable marker above is the authorization boundary. Removing the
	// pending row narrows the remaining attack surface, but a failure here
	// cannot reopen the window because every resolver also checks the marker.
	var pending PendingDecisionFile
	_ = store.UpdateJSONAtomically(pendingDecisionsFile, &pending, func() error {
		kept := make([]PendingDecision, 0, len(pending.Decisions))
		for _, d := range pending.Decisions {
			if !d.Resolved && d.Source == "forced-reviewer-waiver" && d.Phase != nil && *d.Phase == phaseID {
				continue
			}
			kept = append(kept, d)
		}
		pending.Decisions = kept
		return nil
	})
	return nil
}

// clearPhaseDispatchWindow reopens phaseID's forced-reviewer decline window
// for a brand-new build attempt -- CR-01's residual (194-REVIEW.md iteration
// 3/4). closeForcedReviewerWaiverWindowForPhase records the FIRST-EVER
// dispatch moment for a phase and never expired it, so once a phase had had
// even one worker spawned in ANY prior attempt, phaseDispatchStartedAt
// returned true forever after -- refusing the owner's genuine decline on
// every later retry of that same phase (build, fail or get reviewed,
// rebuild is this repo's own normal loop, not an edge case). This function
// deletes ONLY the dispatch-start timestamp for phaseID; it never touches
// pending-decisions.json, so a waiver the owner already resolved on an
// earlier attempt of this exact phase and signal (D-03: "one signal, one
// phase", not "one attempt") is left completely alone and keeps counting.
//
// Production build-start callers no longer invoke this helper. Plan 200-34
// moved reopen/close together with the attempt and manifest into
// commitBuildStart, preventing a crash or stale-authority race between those
// writes. The helper remains temporarily for older focused fixtures until the
// legacy writer retirement plans remove that test surface.
//
// TestForcedReviewerDeclineWindowReopensOnRetriedBuild
// (cmd/forced_reviewer_waiver_test.go) proves the retry scenario this fixes;
// TestForcedReviewerDeclineWindowClosesWhenDispatchBegins and
// TestOwnerDeclineBeforeDispatchStaysHonoredOnceDispatchBegins (both
// pre-existing) keep proving the within-attempt guarantee is untouched.
func clearPhaseDispatchWindow(phaseID int) {
	if store == nil || phaseID <= 0 {
		return
	}
	windowFile := loadPhaseDispatchWindowFile()
	key := strconv.Itoa(phaseID)
	if _, present := windowFile.Phases[key]; !present {
		return
	}
	delete(windowFile.Phases, key)
	_ = store.SaveJSON(phaseDispatchWindowFileName, windowFile)
}

// This file holds the owner's ONLY way to decline a forced reviewer (D-03,
// .planning/phases/194-the-queen-decides-the-team/194-CONTEXT.md). It follows
// cmd/criterion_owner_confirmation.go line for line in structure: a stable
// deterministic question text, a matcher against resolved decision-answer
// entries, and a shell-quoted command shown to the owner. Recording an
// answer happens only through the existing `aether decision-answer` command
// (cmd/handoff_decisions_cmd.go), the same path a worker's open_decisions
// question already goes through. T-194-11.
//
// CR-01 (194-REVIEW.md) closed the gap in that last sentence: before this
// fix, `aether decision-answer` had no authentication and no requirement
// that its --question text match anything the program had actually shown
// anyone -- since forcedReviewerWaiverQuestionText is fully deterministic
// from public information (phaseID + one of five fixed PlainEnglish
// strings, all readable in queenRiskSignalTable or the rendered check-in
// card), ANY process able to invoke the `aether` binary could forge the
// exact command and silently waive a reviewer with no owner ever having
// seen the question. The fix is two functions in this file plus one call
// added at the CLI boundary (decisionAnswerCmd.RunE,
// cmd/handoff_decisions_cmd.go):
//
//  1. ensureForcedReviewerWaiverPendingDecision writes the pending,
//     UNRESOLVED question into pending-decisions.json the moment (and ONLY
//     the moment) the check-in card renders a LIVE forced reviewer
//     (cmd/ceremony_team_checkin.go) -- this is the runtime's own record
//     that it actually showed this exact question to whoever is looking at
//     the card.
//  2. resolveForcedReviewerWaiverPendingDecision is the only way a
//     forced-reviewer-shaped --question can be answered: it looks up that
//     SAME pending row and marks it resolved. It never creates a new
//     resolved entry the way the general clarification path
//     (recordDecisionAnswer) does -- so a --question with no matching
//     runtime-created row is refused and nothing is recorded.
//
// TestDecisionAnswerCannotForgeAForcedReviewerWaiver
// (cmd/forced_reviewer_waiver_test.go) proves the forgery path this closes:
// calling the decision-answer COMMAND (not just recordDecisionAnswer) with a
// correctly-shaped question and no prior card render still leaves the
// forced reviewer in the dispatch list.

// forcedReviewerWaiverReviewerLabel names the reviewer a signal would force,
// the way the owner reads it -- "security reviewer" or "quality reviewer",
// never the registry identifier (gatekeeper/auditor). Reuses
// forcedReviewerPlainLabel (cmd/codex_build.go) via the signal's own Caste so
// this sentence and the pre-build announcement never describe the same
// reviewer two different ways.
func forcedReviewerWaiverReviewerLabel(signal string) string {
	signal = strings.TrimSpace(signal)
	for _, s := range queenRiskSignalTable {
		if s.Name == signal {
			return forcedReviewerPlainLabel(s.Caste)
		}
	}
	return "reviewer"
}

// forcedReviewerWaiverQuestionText builds the stable, deterministic sentence
// a waiver answer is matched against, in the shape: "Phase 7: a security
// reviewer is being added because this touches logins and passwords. Waive
// it?". phaseID and plainEnglish (unique per signal -- see
// queenRiskSignalTable's own PlainEnglish values) together make this text
// unique per phase AND per signal: that uniqueness IS the scoping rule in
// D-03, exactly as ownerConfirmationQuestionText scopes by phase and task.
// signal only selects the reviewer label (forcedReviewerWaiverReviewerLabel)
// -- the raw signal identifier (e.g. "credentials/auth") never appears in
// text the owner is shown, per CLAUDE.md's plain-English mandate.
func forcedReviewerWaiverQuestionText(phaseID int, signal string, plainEnglish string) string {
	plainEnglish = strings.TrimSpace(plainEnglish)
	return fmt.Sprintf(
		"Phase %d: a %s is being added because this touches %s. Waive it?",
		phaseID, forcedReviewerWaiverReviewerLabel(signal), plainEnglish,
	)
}

// forcedReviewerWaiverCommand is the exact `aether decision-answer`
// invocation the owner runs to decline one forced reviewer -- surfaced
// verbatim on the check-in card so the owner never has to construct the
// question text by hand. The question text is shell-quoted (shellQuote,
// cmd/criterion_owner_confirmation.go), not Go-quoted (%q), because it is
// pasted into a real shell by a reader this repo's own CLAUDE.md describes as
// non-technical (CR-02, 193-REVIEW.md; T-194-12). The answer is a
// placeholder the owner replaces with their own reason -- D-03 requires the
// reason be written down, and this module never writes one on the owner's
// behalf.
func forcedReviewerWaiverCommand(phaseID int, signal string, plainEnglish string, capability string) string {
	question := forcedReviewerWaiverQuestionText(phaseID, signal, plainEnglish)
	return fmt.Sprintf(
		"aether decision-answer --question %s --answer '<say why here>' --phase %d --waiver-capability %s",
		shellQuote(question), phaseID, shellQuote(capability),
	)
}

// forcedReviewerWaiver reports whether the owner has already answered this
// phase-and-signal's waiver question through the decision-answer path, and
// the owner's recorded reason. It matches the normalised question text
// against resolved pending-decision entries the SAME way
// answeredDecisionTexts/ownerConfirmationAnswered already do (matching either
// the parsed clarification question or the raw description) -- reading the
// pending-decision file directly rather than introducing a second
// normalisation, because answeredDecisionTexts only returns presence and
// this caller also needs the reason text for the card (D-03: "the card shows
// waived by owner: <reason>").
func forcedReviewerWaiver(phaseID int, signal string) (waived bool, reason string) {
	signal = strings.TrimSpace(signal)
	plainEnglish := ""
	for _, s := range queenRiskSignalTable {
		if s.Name == signal {
			plainEnglish = s.PlainEnglish
			break
		}
	}
	if plainEnglish == "" {
		return false, ""
	}
	target := normalizeDecisionText(forcedReviewerWaiverQuestionText(phaseID, signal, plainEnglish))
	if target == "" {
		return false, ""
	}
	// CR-01 residual (194-REVIEW.md iteration 2): defense in depth against a
	// resolved row that reached pending-decisions.json some way other than
	// resolveForcedReviewerWaiverPendingDecision (e.g. a direct edit of the
	// JSON file, bypassing the CLI and its dispatch-start check entirely).
	// Even a genuinely resolved, correctly-worded entry is only honored if
	// it was resolved BEFORE dispatch began for this phase -- the owner's
	// real decline, made at the check-in pause. A resolution timestamped at
	// or after dispatch start could not have been the owner's answer at that
	// pause (closeForcedReviewerWaiverWindowForPhase already deletes any
	// still-pending row at that moment), so it is never honored here either.
	dispatchedAt, dispatchStarted := phaseDispatchStartedAt(phaseID)
	file := loadPendingDecisionFile()
	for _, decision := range file.Decisions {
		if !decision.Resolved || strings.TrimSpace(decision.Resolution) == "" {
			continue
		}
		if decision.Source != "forced-reviewer-waiver" || decision.Phase == nil || *decision.Phase != phaseID ||
			strings.TrimSpace(decision.AttemptID) == "" || !forcedReviewerWaiverDecisionHasCapability(decision) {
			continue
		}
		question, _ := parseClarificationDescription(decision.Description)
		if normalizeDecisionText(question) != target && normalizeDecisionText(decision.Description) != target {
			continue
		}
		if dispatchStarted {
			resolvedAt, err := time.Parse(time.RFC3339, strings.TrimSpace(decision.ResolvedAt))
			if err != nil || !resolvedAt.Before(dispatchedAt) {
				continue
			}
		}
		return true, strings.TrimSpace(decision.Resolution)
	}
	return false, ""
}

// applyForcedReviewerWaivers filters hits at the SIGNAL level -- before
// collapseToForcedReviewers merges signals into castes -- so a waived
// credentials signal on a phase that also carries a live payments signal
// still forces the security reviewer, with a reason naming payments alone
// (D-03's "one signal for one phase" rule; key_links in
// .planning/phases/194-the-queen-decides-the-team/194-07-PLAN.md). Called
// from inside queenForcedContinueReviewers (cmd/queen_risk_signals.go) --
// the one function both continue lanes already go through -- so the waiver
// applies identically on the in-process and wrapper lanes with no second
// implementation, and because that function unions plan-wording hits with
// changed-file hits BEFORE this filter runs, a signal waived from plan
// wording stays waived when the same signal is re-detected from the files
// the builder changed.
func applyForcedReviewerWaivers(phaseID int, hits []riskSignalHit) (kept []riskSignalHit, waived []riskSignalHit) {
	for _, hit := range hits {
		if ok, reason := forcedReviewerWaiver(phaseID, hit.Signal.Name); ok {
			hit.WaiverReason = reason
			waived = append(waived, hit)
			continue
		}
		kept = append(kept, hit)
	}
	return kept, waived
}

// ensureForcedReviewerWaiverPendingDecision writes the pending, UNRESOLVED
// decision row the owner's decline command (forcedReviewerWaiverCommand)
// can later resolve -- CR-01's fix (194-REVIEW.md). Called ONLY from the
// check-in card's render step (cmd/ceremony_team_checkin.go), once per LIVE
// forced hit -- a hit applyForcedReviewerWaivers has already filtered out
// (an already-waived signal) is never re-offered a fresh row here. A repeat
// render of the same live question keeps one row and adds only another
// capability hash, so every raw command already shown stays usable while no
// raw capability is persisted. An already-resolved row remains a no-op.
func newForcedReviewerWaiverCapability() (string, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	capability := base64.RawURLEncoding.EncodeToString(raw)
	digest := sha256.Sum256([]byte(capability))
	return capability, hex.EncodeToString(digest[:]), nil
}

func forcedReviewerWaiverDecisionHasCapability(decision PendingDecision) bool {
	if strings.TrimSpace(decision.WaiverCapabilitySHA256) != "" {
		return true
	}
	for _, hash := range decision.WaiverCapabilitySHA256s {
		if strings.TrimSpace(hash) != "" {
			return true
		}
	}
	return false
}

func forcedReviewerWaiverCapabilityMatches(decision PendingDecision, providedHash string) bool {
	providedHash = strings.TrimSpace(providedHash)
	if providedHash == "" {
		return false
	}
	if subtle.ConstantTimeCompare(
		[]byte(strings.TrimSpace(decision.WaiverCapabilitySHA256)),
		[]byte(providedHash),
	) == 1 {
		return true
	}
	for _, candidate := range decision.WaiverCapabilitySHA256s {
		if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(candidate)), []byte(providedHash)) == 1 {
			return true
		}
	}
	return false
}

func ensureForcedReviewerWaiverPendingDecision(phaseID int, signalName, plainEnglish, attemptID string) (string, error) {
	if store == nil {
		return "", nil
	}
	attemptID = strings.TrimSpace(attemptID)
	_, latest, ok := loadLatestBuildAttempt(phaseID)
	if phaseID <= 0 || attemptID == "" || !ok || latest.ID != attemptID || !buildAttemptStatusActive(latest.Status) {
		return "", nil
	}
	if _, started := phaseDispatchStartedAt(phaseID); started {
		return "", nil
	}
	question := forcedReviewerWaiverQuestionText(phaseID, signalName, plainEnglish)
	target := normalizeDecisionText(question)
	if target == "" {
		return "", nil
	}

	file := loadPendingDecisionFile()
	matching := -1
	for i, d := range file.Decisions {
		q, _ := parseClarificationDescription(d.Description)
		if normalizeDecisionText(q) == target || normalizeDecisionText(d.Description) == target {
			if d.Resolved && d.Source == "forced-reviewer-waiver" {
				return "", nil
			}
			matching = i
			break
		}
	}
	capability, capabilityHash, err := newForcedReviewerWaiverCapability()
	if err != nil {
		return "", err
	}

	// A second render is another representation of the same live question,
	// not a new authorization decision. Keep the existing row and every
	// previously issued hash valid so the visual command cannot be invalidated
	// by the wrapper's immediate JSON render. Raw capabilities are never stored.
	if matching >= 0 {
		existing := &file.Decisions[matching]
		if !existing.Resolved &&
			existing.Source == "forced-reviewer-waiver" &&
			existing.Phase != nil && *existing.Phase == phaseID &&
			existing.AttemptID == attemptID {
			existing.WaiverCapabilitySHA256s = append(existing.WaiverCapabilitySHA256s, capabilityHash)
			if err := store.SaveJSON(pendingDecisionsFile, file); err != nil {
				return "", err
			}
			return capability, nil
		}
	}

	decision := PendingDecision{
		ID:                     fmt.Sprintf("pd_%d", time.Now().UnixNano()),
		Type:                   clarificationDecisionType,
		Description:            formatClarificationDescription(question, nil),
		Source:                 "forced-reviewer-waiver",
		Resolved:               false,
		CreatedAt:              time.Now().UTC().Format(time.RFC3339),
		AttemptID:              attemptID,
		WaiverCapabilitySHA256: capabilityHash,
	}
	if phaseID > 0 {
		decision.Phase = &phaseID
	}
	stampPendingDecisionScope(&decision, loadCurrentPendingDecisionScope())

	if matching >= 0 {
		file.Decisions[matching] = decision
	} else {
		file.Decisions = append(file.Decisions, decision)
	}
	if err := store.SaveJSON(pendingDecisionsFile, file); err != nil {
		return "", err
	}
	return capability, nil
}

// forcedReviewerWaiverPhasePrefix parses the phase number
// forcedReviewerWaiverQuestionText embeds at the start of every waiver
// question ("Phase 7: ..."). Used by forcedReviewerWaiverSignalForQuestion
// to recover the phase a candidate --question text claims to be about,
// independent of (and never trusting) any --phase flag the caller also
// passed.
var forcedReviewerWaiverPhasePrefix = regexp.MustCompile(`^Phase (\d+):`)

// forcedReviewerWaiverSignalForQuestion reports whether a candidate
// question text is shaped like a forced-reviewer waiver question, and if
// so, which phase and signal it names. This is CR-01's forgery check: the
// ONLY way to be sure a --question is (or is not) this shape is to
// regenerate the exact deterministic sentence the runtime would have
// produced for every signal at the phase the text itself claims, and
// compare -- never to pattern-match the prose, which a forger could dress
// up to look close enough. The phase is read from the question text itself
// (not a --phase flag the caller also supplied), because a forger could
// set an unrelated or absent --phase to try to dodge this check.
func forcedReviewerWaiverSignalForQuestion(question string) (phaseID int, signalName string, ok bool) {
	question = strings.TrimSpace(question)
	m := forcedReviewerWaiverPhasePrefix.FindStringSubmatch(question)
	if m == nil {
		return 0, "", false
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, "", false
	}
	target := normalizeDecisionText(question)
	for _, s := range queenRiskSignalTable {
		candidate := forcedReviewerWaiverQuestionText(n, s.Name, s.PlainEnglish)
		if normalizeDecisionText(candidate) == target {
			return n, s.Name, true
		}
	}
	return 0, "", false
}

// resolveForcedReviewerWaiverPendingDecision is the ONLY way a
// forced-reviewer-shaped question can be answered (CR-01, 194-REVIEW.md):
// it finds the SINGLE pending, unresolved row the runtime itself created
// for this exact phase+question
// (ensureForcedReviewerWaiverPendingDecision, called from the check-in
// card's render step) and marks it resolved with the owner's answer.
// Unlike recordDecisionAnswer, this function NEVER creates a new entry --
// if no matching unresolved row exists, the waiver did not originate from
// something the runtime actually showed anyone, and is refused. found is
// false in that case; no file write happens.
func resolveForcedReviewerWaiverPendingDecision(question, answer string, phaseID int, capability string) (decision PendingDecision, found bool, err error) {
	if store == nil {
		return PendingDecision{}, false, nil
	}
	target := normalizeDecisionText(question)
	if target == "" {
		return PendingDecision{}, false, nil
	}
	capability = strings.TrimSpace(capability)
	if capability == "" {
		return PendingDecision{}, false, nil
	}
	// CR-01 residual (194-REVIEW.md iteration 2): once dispatch has begun for
	// this phase, the decline window is closed -- refuse even if a matching
	// pending row somehow still exists (closeForcedReviewerWaiverWindowForPhase
	// deletes it the moment dispatch starts, but this is the second, explicit
	// check that makes the refusal unconditional rather than depending on
	// ordering between that deletion and this call).
	if phaseID > 0 {
		if _, started := phaseDispatchStartedAt(phaseID); started {
			return PendingDecision{}, false, nil
		}
	}
	_, activeAttempt, ok := loadLatestBuildAttempt(phaseID)
	if !ok || !buildAttemptStatusActive(activeAttempt.Status) {
		return PendingDecision{}, false, nil
	}
	providedHash := sha256.Sum256([]byte(capability))
	providedHashHex := hex.EncodeToString(providedHash[:])

	var file PendingDecisionFile
	var resolved PendingDecision
	found = false
	if err := store.UpdateJSONAtomically(pendingDecisionsFile, &file, func() error {
		// Recheck the dispatch window inside the pending-file transaction. The
		// capability is single use, and a concurrent spawn-log must win over a
		// late decline rather than leaving a resolvable row behind.
		if _, started := phaseDispatchStartedAt(phaseID); started {
			return nil
		}
		for i := range file.Decisions {
			d := &file.Decisions[i]
			if d.Resolved {
				continue
			}
			if d.Source != "forced-reviewer-waiver" || d.AttemptID != activeAttempt.ID {
				continue
			}
			if !forcedReviewerWaiverCapabilityMatches(*d, providedHashHex) {
				continue
			}
			if d.Phase == nil || *d.Phase != phaseID {
				continue
			}
			q, _ := parseClarificationDescription(d.Description)
			if normalizeDecisionText(q) != target && normalizeDecisionText(d.Description) != target {
				continue
			}
			d.Resolved = true
			d.Resolution = answer
			d.ResolvedAt = time.Now().UTC().Format(time.RFC3339)
			resolved = *d
			found = true
			break
		}
		return nil
	}); err != nil {
		return PendingDecision{}, false, err
	}
	return resolved, found, nil
}
