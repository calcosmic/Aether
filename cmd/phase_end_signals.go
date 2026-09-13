package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// middenAutoRedirectThreshold is the number of unacknowledged failures a
// single midden.json category must hold before emitMiddenThresholdRedirect
// writes an automatic "don't do this" REDIRECT signal for it (198.1-04,
// FEED-04).
const middenAutoRedirectThreshold = 3

// signalContentSafeLimit is the character budget an emitted note is trimmed
// to before it reaches writePheromoneSignal. pkg/colony's own signal ceiling
// is 500 characters (maxSignalContentLength); this leaves headroom below
// that strict maximum because sanitisation can itself grow the content
// slightly (angle brackets are escaped to &lt;/&gt;), and because a note
// that silently fails sanitisation for being one character too long would
// defeat the whole point of these emitters -- a phase advance or a decision
// answer that produces NO note at all, with only a stderr warning nobody
// reads mid-flight.
const signalContentSafeLimit = 480

// truncateSignalContent trims s to fit within max characters, appending an
// ellipsis when truncation occurs so the cut is visible in the stored note
// rather than silent.
func truncateSignalContent(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

// countUnacknowledgedMiddenEntries returns the number of midden.json entries
// that have not been marked Acknowledged -- the "failures recorded" figure
// folded into the phase-completion note (198.1-04, FEED-04). A load failure
// (e.g. no midden.json yet, a fresh colony's normal starting state) reports
// zero rather than propagating an error: this is a bookkeeping count, never
// a gate.
func countUnacknowledgedMiddenEntries(s *storage.Store) int {
	mf, err := loadMiddenFile(s)
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range mf.Entries {
		if entry.Acknowledged == nil || !*entry.Acknowledged {
			count++
		}
	}
	return count
}

// emitPhaseCompletionFeedback writes one FEEDBACK signal naming the finished
// phase and what it produced for memory this phase: lessons captured
// (promotion candidates from learning-observations.json), failures recorded
// (unacknowledged midden.json entries), and instincts promoted to the Queen
// file. Deliberately more than "phase N completed" -- a note that only
// narrates progress carries no steering value, and this repo's own
// admissibility rules (pkg/memory.IsAdmissibleInstinctContent) exist to keep
// exactly that kind of content out of the wisdom pipeline. A signal-write
// failure warns to stderr and never blocks the phase advance that produced
// this call (D-05).
func emitPhaseCompletionFeedback(phase colony.Phase, summary phaseEndConsolidationSummary, lessonsCaptured, failuresRecorded int) {
	name := strings.TrimSpace(phase.Name)
	if name == "" {
		name = fmt.Sprintf("phase %d", phase.ID)
	}
	text := fmt.Sprintf(
		"Phase %d (%s) finished: %d lesson(s) captured, %d failure(s) recorded, %d instinct(s) promoted to the Queen file.",
		phase.ID, name, lessonsCaptured, failuresRecorded, len(summary.QueenPromoted),
	)
	text = truncateSignalContent(text, signalContentSafeLimit)
	if _, _, err := writePheromoneSignal("FEEDBACK", text, "", "aether continue", "", "", 0, nil); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not record phase completion note: %v\n", err)
	}
}

// emitDecisionFeedback writes one FEEDBACK signal carrying the owner's own
// answer to a worker's open question, so a future worker's prompt shows not
// just that a decision was made but what it was. Called only after the
// answer is durably stored (see recordDecisionAnswer) -- a failed store
// never produces a note for an answer that was not recorded.
func emitDecisionFeedback(question, answer string, phase int) {
	q := strings.TrimSpace(question)
	a := strings.TrimSpace(answer)
	text := truncateSignalContent(fmt.Sprintf("Answered %q: %s", q, a), signalContentSafeLimit)
	if _, _, err := writePheromoneSignal("FEEDBACK", text, "", "aether decision-answer", "", "", 0, nil); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not record decision answer note: %v\n", err)
	}
}

// emitMiddenThresholdRedirect scans midden.json for any failure category
// holding middenAutoRedirectThreshold or more unacknowledged entries and
// writes one REDIRECT signal per such category, leading with the newest
// entry's own message and then stating that this kind of failure has
// recurred at least middenAutoRedirectThreshold times. The count in the
// text is deliberately the fixed threshold, not the exact accumulated
// total: writePheromoneSignal already reinforces on a content-hash match,
// and a stable message for a given (category, newest message) pair is what
// makes repeat emission idempotent (T-198.1-16) -- an exact, ever-growing
// count would change the text (and so the hash) on every additional
// unacknowledged failure of the same kind, defeating that reinforcement.
// Deliberately no second dedup mechanism. Returns the number of categories
// that crossed the threshold this call.
func emitMiddenThresholdRedirect() int {
	if store == nil {
		return 0
	}
	mf, err := loadMiddenFile(store)
	if err != nil {
		return 0
	}

	byCategory := map[string][]colony.MiddenEntry{}
	for _, entry := range mf.Entries {
		if entry.Acknowledged != nil && *entry.Acknowledged {
			continue
		}
		byCategory[entry.Category] = append(byCategory[entry.Category], entry)
	}

	categories := make([]string, 0, len(byCategory))
	for category := range byCategory {
		categories = append(categories, category)
	}
	sort.Strings(categories)

	crossed := 0
	for _, category := range categories {
		entries := byCategory[category]
		if len(entries) < middenAutoRedirectThreshold {
			continue
		}
		newest := entries[0]
		for _, e := range entries[1:] {
			if e.Timestamp > newest.Timestamp {
				newest = e
			}
		}
		text := truncateSignalContent(fmt.Sprintf(
			"%s — this failure has recurred %d or more times unacknowledged (category: %s); avoid repeating it",
			newest.Message, middenAutoRedirectThreshold, category,
		), signalContentSafeLimit)
		if _, _, err := writePheromoneSignal("REDIRECT", text, "", "aether continue", "", "", 0, nil); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not record automatic redirect for midden category %q: %v\n", category, err)
			continue
		}
		crossed++
	}
	return crossed
}

// runPheromoneOutcomeTuning is the ONE caller of
// cmd/pheromone_outcome.go's tuneNoteStrengthFromOutcomes (plan 203-13,
// BIO-08/CEC-07's outcome-weighted strength tuning) -- both continue lanes
// (cmd/codex_continue.go, cmd/codex_continue_finalize.go) call THIS wrapper,
// immediately after promotePhaseEndInstinctsToHive, never
// tuneNoteStrengthFromOutcomes directly. This is what makes
// TestBothCheckLanesTuneNotes and TestTuningIsNotOnTheBuildPath able to name
// exactly one chokepoint.
//
// Never propagates a panic or an error to the check that called it -- the
// same non-blocking discipline promotePhaseEndInstinctsToHive already
// follows for hive promotion. A tuning failure is recorded in the returned
// result and never fails, pauses, or alters the phase advance that produced
// this call.
func runPheromoneOutcomeTuning() (result pheromoneOutcomeTuningResult) {
	defer func() {
		if r := recover(); r != nil {
			result = pheromoneOutcomeTuningResult{Error: fmt.Sprintf("panic: %v", r)}
		}
	}()
	return tuneNoteStrengthFromOutcomes()
}

// attachPheromoneOutcomeTuningSummary stores the outcome-weighted tuning
// pass's result under result["pheromone_outcome_tuning"], mirroring
// attachHivePromotionSummary's shape so both continue lanes report tuning
// the same way.
func attachPheromoneOutcomeTuningSummary(result map[string]interface{}, tuning pheromoneOutcomeTuningResult) {
	if result == nil {
		return
	}
	result["pheromone_outcome_tuning"] = map[string]interface{}{
		"ran":                tuning.Ran,
		"records_considered": tuning.RecordsConsidered,
		"notes_tuned":        tuning.NotesTuned,
		"quarantined":        tuning.Quarantined,
		"skipped_pinned":     tuning.SkippedPinned,
		"skipped_revoked":    tuning.SkippedRevoked,
		"error":              tuning.Error,
	}
}

// signalIsWorthKeeping reports whether an expiring signal is valuable enough
// to survive its own expiry in long-term (eternal) memory (198.1-04,
// FEED-04): true when the signal is a REDIRECT (a hard constraint, by
// construction worth remembering), or when it was reinforced at least once
// (repeated content the colony independently re-emitted, not a one-off
// FEEDBACK note nobody echoed).
func signalIsWorthKeeping(sig colony.PheromoneSignal) bool {
	if sig.Type == "REDIRECT" {
		return true
	}
	return sig.ReinforcementCount != nil && *sig.ReinforcementCount > 0
}

// promoteExpiredSignalToEternal preserves an expiring signal's own text in
// the hub's long-term memory when signalIsWorthKeeping reports it valuable,
// via the same appendEternalMemoryEntry the eternal-store command uses
// (198.1-04, FEED-04). A throwaway signal (not worth keeping) is a no-op:
// (false). A hub write failure warns to stderr and returns false -- it
// never fails the expiry that produced this call. Returns whether an entry
// was newly appended.
func promoteExpiredSignalToEternal(sig colony.PheromoneSignal) bool {
	if !signalIsWorthKeeping(sig) {
		return false
	}
	text := strings.TrimSpace(extractText(sig.Content))
	if text == "" {
		return false
	}
	appended, err := appendEternalMemoryEntry(text, "promoted_on_expire", 0.9, "promoted_on_expire")
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not promote expiring signal %q to long-term memory: %v\n", sig.ID, err)
		return false
	}
	return appended
}
