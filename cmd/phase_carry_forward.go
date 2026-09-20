package cmd

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// phaseCarryForwardBudgetChars bounds the failed/flagged item list inside
// resolvePreviousPhaseCarryForward. This is its OWN allowance, deliberately
// kept outside colonyPrimeBudgetChars / colonyPrimeCompactBudgetChars
// (cmd/colony_prime_context.go) and outside surveyDigestBudgetChars
// (cmd/helpers.go) -- the folded worker-turnaround todo caps total brief
// growth at exactly these two new named slots (D-01, D-09), so this
// constant must never draw from either existing budget. The persisted
// closing summary (outcome.md, D-11) is appended after this budget is spent
// and is never truncated against it -- D-11 requires it word-for-word, and a
// truncated "closing summary" would no longer be the text the owner read.
const phaseCarryForwardBudgetChars = 2000

// resolvePreviousPhaseCarryForward tells a builder starting a new phase what
// happened in the phase immediately before it (WIRE-07, D-09..D-11): what
// failed, what a reviewer flagged, and the closing summary the owner read --
// never the full verification.json / review.json reports those come from.
//
// It resolves the immediately preceding phase from the plan's own phase
// order -- the phase at position i-1 where the current phase sits at
// position i, not currentPhaseID-1, since phase IDs are not always
// contiguous. Records are then read through the store using the same
// read-and-guard shape criterion_owner_confirmation.go already uses for
// verification.json: a missing or unreadable file is skipped, never an
// error.
//
// From the verification report, only failed (non-skipped) checks and unmet
// or unconfirmed criteria are selected. From the review report, only
// blocking issues are selected. Each becomes one sentence. When nothing
// failed and nothing was flagged, the section collapses to a single line
// naming how many checks passed and that the review was clean -- never the
// full reports either way. The persisted outcome.md text (D-11), when
// present, is appended verbatim as the closing summary, labelled in plain
// English.
//
// Every sentence drawn from verification.json/review.json is sanitised
// through colony.SanitizeSignalContent before entering the section --
// reviewer- and worker-authored report text is untrusted input on its way
// into a new helper's instructions (T-198.2-22), the same boundary 198.1
// established for replayed failure-log text. A sentence the sanitiser
// rejects is named in the omission line, never silently dropped, matching
// the omission handling for a sentence that does not fit the budget.
//
// Returns the empty string when there is no preceding phase, or the
// preceding phase has no persisted verification.json, review.json or
// outcome.md at all (WIRE-07 unclassified truth: no carry-forward section
// at all, not an empty heading).
func resolvePreviousPhaseCarryForward(currentPhaseID int) string {
	if store == nil {
		return ""
	}
	state, err := loadActiveColonyState()
	if err != nil {
		return ""
	}
	prevID, ok := precedingPhaseID(state, currentPhaseID)
	if !ok {
		return ""
	}

	if closure := previousPhaseOutOfBandClosure(state, prevID); closure != "" {
		return closure
	}

	var verification codexContinueVerificationReport
	hasVerification := store.LoadJSON(continuePlanArtifactsPath(prevID, "verification.json"), &verification) == nil

	var review codexContinueReviewReport
	hasReview := store.LoadJSON(continuePlanArtifactsPath(prevID, "review.json"), &review) == nil

	outcomeBytes, outcomeErr := store.ReadFile(continuePlanArtifactsPath(prevID, "outcome.md"))
	outcomeText := strings.TrimSpace(string(outcomeBytes))
	hasOutcome := outcomeErr == nil && outcomeText != ""

	if !hasVerification && !hasReview && !hasOutcome {
		return ""
	}

	type flaggedItem struct {
		label    string
		sentence string
	}
	var items []flaggedItem
	passedCount := 0

	if hasVerification {
		for _, step := range verification.Steps {
			if step.Skipped {
				continue
			}
			if step.Passed {
				passedCount++
				continue
			}
			items = append(items, flaggedItem{
				label:    step.Name,
				sentence: fmt.Sprintf("Check %q failed: %s", step.Name, firstNonEmpty(step.Summary, "no summary recorded")),
			})
		}
		for _, c := range verification.Criteria {
			if c.Passed && c.State == "" {
				continue // ordinary passed criterion -- nothing to say
			}
			if !c.Passed {
				items = append(items, flaggedItem{
					label:    c.Criterion,
					sentence: fmt.Sprintf("Criterion %q was not met: %s", c.Criterion, firstNonEmpty(c.Summary, "not met")),
				})
				continue
			}
			// c.Passed && c.State != "" -- e.g. needs_owner_confirmation: the
			// phase still advanced, but the next phase's builder must still
			// be told this criterion was never actually confirmed (CR-01).
			items = append(items, flaggedItem{
				label:    c.Criterion,
				sentence: fmt.Sprintf("Criterion %q still needs the owner's confirmation: %s", c.Criterion, firstNonEmpty(c.Summary, "no summary recorded")),
			})
		}
	}

	if hasReview {
		for _, issue := range review.BlockingIssues {
			items = append(items, flaggedItem{
				label:    "review",
				sentence: fmt.Sprintf("Review flagged: %s", issue),
			})
		}
	}

	var body strings.Builder
	fmt.Fprintf(&body, "## What Happened Last Phase (Phase %d)\n\n", prevID)

	if len(items) == 0 {
		if hasVerification || hasReview {
			line := fmt.Sprintf("Phase %d: all %d checks passed", prevID, passedCount)
			if hasReview {
				line += ", review clean."
			} else {
				line += "."
			}
			body.WriteString(line)
			body.WriteString("\n")
		}
	} else {
		used := 0
		var omitted []string
		for _, item := range items {
			sanitized, sanErr := colony.SanitizeSignalContent(item.sentence)
			if sanErr != nil {
				omitted = append(omitted, fmt.Sprintf("%s (content rejected: %v)", item.label, sanErr))
				continue
			}
			remaining := phaseCarryForwardBudgetChars - used
			if remaining <= 0 || len(sanitized) > remaining {
				omitted = append(omitted, fmt.Sprintf("%s (over budget)", item.label))
				continue
			}
			fmt.Fprintf(&body, "- %s\n", sanitized)
			used += len(sanitized)
		}
		if len(omitted) > 0 {
			fmt.Fprintf(&body, "_Not included here — read directly if relevant: %s_\n", strings.Join(omitted, ", "))
		}
	}

	if hasOutcome {
		body.WriteString("\nWhat the owner was told when that phase finished:\n\n")
		body.WriteString(outcomeText)
		body.WriteString("\n")
	}

	return strings.TrimSpace(body.String())
}

// precedingPhaseID returns the phase ID at the position immediately before
// currentPhaseID within state.Plan.Phases -- the phase's own recorded order,
// not currentPhaseID-1, since phase IDs are not always contiguous in this
// repository's own plans (e.g. 198, 198.1, 198.2 are distinct integer IDs
// with no arithmetic relationship). Returns ok=false when currentPhaseID is
// the first phase, or is not found in the plan at all.
func precedingPhaseID(state colony.ColonyState, currentPhaseID int) (int, bool) {
	phases := state.Plan.Phases
	for i, p := range phases {
		if p.ID != currentPhaseID {
			continue
		}
		if i == 0 {
			return 0, false
		}
		return phases[i-1].ID, true
	}
	return 0, false
}

// outOfBandCarryForward binds the accepted closure to its exact attempt and
// the reports it superseded. Historical reports remain byte-for-byte intact.
type outOfBandCarryForward struct {
	Phase      int                         `json:"phase"`
	AttemptID  string                      `json:"attempt_id"`
	Provenance outOfBandVerificationRecord `json:"provenance"`
	Superseded map[string]string           `json:"superseded"`
}

func carryForwardReportDigests(phaseID int) map[string]string {
	result := make(map[string]string)
	for _, name := range []string{"verification.json", "review.json", "outcome.md"} {
		if data, err := store.ReadFile(continuePlanArtifactsPath(phaseID, name)); err == nil {
			result[name] = fmt.Sprintf("%x", sha256.Sum256(data))
		}
	}
	return result
}

func previousPhaseOutOfBandClosure(state colony.ColonyState, phaseID int) string {
	completed := false
	for _, phase := range state.Plan.Phases {
		if phase.ID == phaseID {
			completed = phase.Status == colony.PhaseCompleted
		}
	}
	if !completed {
		return ""
	}
	_, attempt, hasAttempt := loadLatestBuildAttempt(phaseID)
	if !hasAttempt {
		// An unreadable or mismatched pointer is not evidence that no attempt
		// exists; never let it revive an older no-attempt closure.
		if _, err := store.ReadFile(latestBuildAttemptPointerPath(phaseID)); !errors.Is(err, os.ErrNotExist) {
			return ""
		}
	}
	var closure outOfBandCarryForward
	err := store.LoadJSON(continuePlanArtifactsPath(phaseID, "out-of-band-closure.json"), &closure)
	if err == nil {
		if closure.Phase != phaseID || closure.Provenance.Phase != phaseID {
			return ""
		}
		if hasAttempt != (closure.AttemptID != "") || (hasAttempt && (attempt.ID != closure.AttemptID || attempt.Status != buildAttemptBuilt)) {
			return ""
		}
		current := carryForwardReportDigests(phaseID)
		if len(current) != len(closure.Superseded) {
			return ""
		}
		for name, digest := range current {
			if closure.Superseded[name] != digest {
				return ""
			}
		}
	} else {
		if !errors.Is(err, os.ErrNotExist) {
			return ""
		}
		// Compatibility for closures accepted before this context record existed.
		// Only the current, successfully closed attempt can supply this provenance.
		if !hasAttempt || attempt.Phase != phaseID || attempt.Status != buildAttemptBuilt || attempt.OutOfBandVerification == nil {
			return ""
		}
		closure.Provenance = *attempt.OutOfBandVerification
		if closure.Provenance.Phase != phaseID {
			return ""
		}
		verifiedAt, parseErr := time.Parse(time.RFC3339Nano, closure.Provenance.VerifiedAt)
		if parseErr != nil {
			return ""
		}
		// Legacy JSON timestamps have only second precision, and outcome.md
		// has none. File freshness must also pass for every historical report.
		for _, name := range []string{"verification.json", "review.json", "outcome.md"} {
			reportPath := filepath.Join(store.BasePath(), continuePlanArtifactsPath(phaseID, name))
			if info, statErr := os.Stat(reportPath); statErr == nil {
				if info.ModTime().After(verifiedAt) {
					return ""
				}
			} else if !errors.Is(statErr, os.ErrNotExist) {
				return ""
			}
		}
		for _, name := range []string{"verification.json", "review.json"} {
			var report struct {
				GeneratedAt string `json:"generated_at"`
			}
			if err := store.LoadJSON(continuePlanArtifactsPath(phaseID, name), &report); err == nil {
				generatedAt, parseErr := time.Parse(time.RFC3339Nano, report.GeneratedAt)
				if parseErr != nil || generatedAt.After(verifiedAt) {
					return ""
				}
			} else if !errors.Is(err, os.ErrNotExist) {
				return ""
			}
		}
	}
	if _, err := time.Parse(time.RFC3339Nano, closure.Provenance.VerifiedAt); err != nil {
		return ""
	}
	return fmt.Sprintf("## What Happened Last Phase (Phase %d)\n\nPhase %d closed through accepted verify-out-of-band verification at %s. Fresh checks: %d; criteria checked: %d; artifacts hashed: %d. This closure does not claim a completed worker review.\n\nEarlier verification.json, review.json, and outcome.md are superseded historical reports retained in build/phase-%d; their blocked closeout is not the current phase outcome.", phaseID, phaseID, closure.Provenance.VerifiedAt, len(closure.Provenance.ChecksRun), len(closure.Provenance.CriteriaChecked), len(closure.Provenance.ArtifactsHashed), phaseID)
}
