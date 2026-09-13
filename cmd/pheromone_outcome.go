package cmd

// BIO-08/CEC-07 (203-13-PLAN.md): outcome-weighted strength tuning. A note's
// strength moves up or down only on RECORDED EVIDENCE of what it did --
// cmd/recruitment_credit.go's own credit records -- never on delivery into a
// brief or a passing phase alone. This is the plan the owner rated costly:
// once a strength can move on its own, telling a learned strength apart from
// one the owner set becomes hard, so every automatic movement is appended to
// the same immutable history 203-11 built (cmd/pheromone_influence.go's
// appendInfluenceHistory), with actor kind "learning", and an owner-pinned
// note is never touched by any path in this file.

import (
	"errors"
	"fmt"
	"os"
)

// ---------------------------------------------------------------------------
// Declared tuning policy (Task 1) -- named constants, never inline literals.
// ---------------------------------------------------------------------------

// noteStrengthHelpfulStep/-Neutral-/-Harmful- are the declared amounts
// tuneNoteStrengthFromOutcomes moves a note's strength by, one movement per
// credit record: helpful raises it, neutral lowers it slightly, harmful
// lowers it more. These are the ONLY place this plan's tuning policy is
// expressed as a number -- every call site reads the constant, never a
// literal.
const (
	noteStrengthHelpfulStep = 0.10
	noteStrengthNeutralStep = -0.02
	noteStrengthHarmfulStep = -0.15
)

// noteStrengthFloor/noteStrengthCeiling bound every strength value this
// runtime writes, automatic or owner-set. A computation landing outside
// either bound is clamped to it, and the clamp is recorded in the history
// entry rather than silently saturating.
const (
	noteStrengthFloor   = 0.0
	noteStrengthCeiling = 1.0
)

// noteHarmfulQuarantineThreshold is the number of harmful credit records a
// single note must accumulate before tuneNoteStrengthFromOutcomes
// automatically quarantines it -- the same threshold-count shape
// middenAutoRedirectThreshold already uses in cmd/phase_end_signals.go for
// "this kind of failure keeps recurring, stop it automatically."
const noteHarmfulQuarantineThreshold = 3

// ---------------------------------------------------------------------------
// Reading credit records: distinguishing "nothing recorded yet" from
// "the store cannot be read" (Task 2's own acceptance criteria).
// ---------------------------------------------------------------------------

// pheromoneOutcomeReadCreditRecords reads every stored recruitment credit
// record. recruitmentCreditAll (cmd/recruitment_credit.go) collapses "the
// file does not exist yet" and "the file exists but cannot be parsed" to the
// same empty-slice, nil-error result -- correct for its own callers, but
// wrong here: this plan's own acceptance criteria require a tuning pass that
// cannot read the credit store to record that it could not run, distinct
// from a tuning pass that legitimately found nothing to do.
func pheromoneOutcomeReadCreditRecords() ([]recruitmentCreditRecord, error) {
	if store == nil {
		return nil, fmt.Errorf("no store initialized")
	}
	var file recruitmentCreditFile
	err := store.LoadJSON(recruitmentCreditPath, &file)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []recruitmentCreditRecord{}, nil
		}
		return nil, err
	}
	records := append([]recruitmentCreditRecord{}, file.Entries...)
	sortRecruitmentCreditRecords(records)
	return records, nil
}

// ---------------------------------------------------------------------------
// Idempotency bookkeeping: which credit records this pass has already
// applied, so a second run over the same records changes nothing.
// ---------------------------------------------------------------------------

// pheromoneOutcomeStatePath is this plan's OWN bookkeeping file. It never
// mutates credit/records.json (203-12's file, whose only writer remains
// recordRecruitmentCredit) or pheromones-history.json (whose only writer
// remains appendInfluenceHistory) -- it only remembers which credit record
// identifiers have already moved a note's strength.
const pheromoneOutcomeStatePath = "pheromone-outcome-tuning-state.json"

// pheromoneOutcomeState is the on-disk shape at pheromoneOutcomeStatePath.
type pheromoneOutcomeState struct {
	ProcessedRecordIDs map[string]bool `json:"processed_record_ids"`
}

// loadPheromoneOutcomeState reads the tuning state, returning an empty
// (never nil) map when the file does not exist yet -- a fresh colony's
// normal starting state, not a failure. A genuine read failure (corrupted
// file) is returned as an error so the caller can report "could not run"
// rather than silently treating corruption as "nothing processed yet."
func loadPheromoneOutcomeState() (pheromoneOutcomeState, error) {
	if store == nil {
		return pheromoneOutcomeState{ProcessedRecordIDs: map[string]bool{}}, fmt.Errorf("no store initialized")
	}
	var st pheromoneOutcomeState
	err := store.LoadJSON(pheromoneOutcomeStatePath, &st)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return pheromoneOutcomeState{ProcessedRecordIDs: map[string]bool{}}, nil
		}
		return pheromoneOutcomeState{}, err
	}
	if st.ProcessedRecordIDs == nil {
		st.ProcessedRecordIDs = map[string]bool{}
	}
	return st, nil
}

func savePheromoneOutcomeState(st pheromoneOutcomeState) error {
	if store == nil {
		return fmt.Errorf("no store initialized")
	}
	return store.SaveJSON(pheromoneOutcomeStatePath, st)
}

// clampNoteStrength bounds v to [noteStrengthFloor, noteStrengthCeiling],
// reporting whether clamping actually changed the value.
func clampNoteStrength(v float64) (float64, bool) {
	if v > noteStrengthCeiling {
		return noteStrengthCeiling, true
	}
	if v < noteStrengthFloor {
		return noteStrengthFloor, true
	}
	return v, false
}

// pheromoneOutcomeSkipReasonPrefix marks a history entry that recorded a
// SKIP (the pass considered the note and declined, never changing its
// strength) rather than a real movement. pheromoneStrengthOrigin below uses
// this prefix to tell a skip entry apart from a genuine strength change
// whose before/after values happen to be numerically equal (e.g. a note
// already at the ceiling reinforced again).
const pheromoneOutcomeSkipReasonPrefix = "skipped: "

// ---------------------------------------------------------------------------
// The tuning pass itself (Task 1 + Task 2).
// ---------------------------------------------------------------------------

// pheromoneOutcomeTuningResult summarizes one run of
// tuneNoteStrengthFromOutcomes -- returned by value, never as a Go error, so
// this pass can never itself fail the check that calls it (Task 2's own
// non-blocking requirement). Ran=false means the pass could not even start
// (the credit store or the tuning state could not be read); every field
// after Error is meaningless in that case.
type pheromoneOutcomeTuningResult struct {
	Ran               bool
	RecordsConsidered int
	NotesTuned        int
	Quarantined       []string
	SkippedPinned     []string
	SkippedRevoked    []string
	Error             string
}

// tuneNoteStrengthFromOutcomes is the ONE function in this codebase that
// moves a pheromone note's strength on RECORDED EVIDENCE of what it did.
// It reads exclusively from cmd/recruitment_credit.go's own credit records
// (recruitmentCreditRecord, kind "note") -- never delivery into a brief,
// never a passing phase, matching this plan's own must_haves truth that a
// delivered note plus a passing phase is not automatically a successful
// application.
//
// Per credit record (skipping any already consumed by a previous pass, and
// any not yet earning a real outcome -- "pending" records are left alone):
//   - A pinned note is skipped entirely, in either direction, and the skip
//     is recorded (pheromoneOutcomeSkipReasonPrefix + "pinned").
//   - A revoked note is skipped entirely, and the skip is recorded.
//   - Otherwise the note's strength moves by the declared step for the
//     record's outcome (helpful/neutral/harmful), clamped to
//     [noteStrengthFloor, noteStrengthCeiling] with the clamp recorded, and
//     the change is appended to the note's own history via
//     appendInfluenceHistory with actor kind "learning" and the credit
//     record's identifier as the reason evidence.
//   - Reaching noteHarmfulQuarantineThreshold harmful credit records for one
//     note quarantines it automatically (Quarantined=true, never cleared by
//     this or any other automatic path -- only an explicit owner-gated
//     release path may ever clear it), with a history entry naming the
//     threshold.
//
// Never returns a Go error: a store that cannot be read is reported via
// Ran=false and Error, leaving every note byte-identical, and the caller
// (cmd/phase_end_signals.go's runPheromoneOutcomeTuning) never propagates
// this as a failure of the check that called it.
func tuneNoteStrengthFromOutcomes() pheromoneOutcomeTuningResult {
	result := pheromoneOutcomeTuningResult{}
	if store == nil {
		result.Error = "no store initialized"
		return result
	}

	records, err := pheromoneOutcomeReadCreditRecords()
	if err != nil {
		result.Error = fmt.Sprintf("could not read recruitment credit store: %v", err)
		return result
	}
	result.Ran = true
	if len(records) == 0 {
		return result
	}

	state, err := loadPheromoneOutcomeState()
	if err != nil {
		result.Ran = false
		result.Error = fmt.Sprintf("could not read tuning state: %v", err)
		return result
	}

	// harmfulCounts is computed over EVERY earned harmful credit record for
	// a note (processed or not, in this run or a prior one) so the
	// quarantine threshold reflects the note's full harmful history, not
	// merely today's newly-processed slice.
	harmfulCounts := map[string]int{}
	for _, rec := range records {
		if rec.ContributionKind == recruitmentContributionNote && rec.Outcome == recruitmentCreditOutcomeHarmful {
			harmfulCounts[rec.ContributionID]++
		}
	}

	newlyProcessed := map[string]bool{}
	var stateErr error

	for _, rec := range records {
		if rec.ContributionKind != recruitmentContributionNote {
			continue
		}
		if rec.Outcome == recruitmentCreditOutcomePending {
			continue
		}
		if state.ProcessedRecordIDs[rec.RecordID] {
			continue
		}
		result.RecordsConsidered++
		noteID := rec.ContributionID

		var step float64
		switch rec.Outcome {
		case recruitmentCreditOutcomeHelpful:
			step = noteStrengthHelpfulStep
		case recruitmentCreditOutcomeNeutral:
			step = noteStrengthNeutralStep
		case recruitmentCreditOutcomeHarmful:
			step = noteStrengthHarmfulStep
		default:
			// Not one of the three earnable outcomes -- nothing to apply.
			newlyProcessed[rec.RecordID] = true
			continue
		}
		wouldBeAction := pheromoneActionReinforced
		if step < 0 {
			wouldBeAction = pheromoneActionWeakened
		}

		pf, idx, found := findPheromoneSignalByID(noteID)
		if !found {
			// The note this record names no longer exists -- nothing to
			// tune. Mark consumed so a re-run does not keep retrying a
			// permanently-gone note.
			newlyProcessed[rec.RecordID] = true
			continue
		}
		sig := pf.Signals[idx]

		if sig.Pinned != nil && *sig.Pinned {
			result.SkippedPinned = append(result.SkippedPinned, noteID)
			cur := "strength=unknown"
			if sig.Strength != nil {
				cur = fmt.Sprintf("strength=%.4f", *sig.Strength)
			}
			if _, herr := appendInfluenceHistory(noteID, wouldBeAction, pheromoneActorLearning, "", cur, cur,
				pheromoneOutcomeSkipReasonPrefix+"pinned (credit "+rec.RecordID+")"); herr != nil {
				stateErr = herr
			}
			newlyProcessed[rec.RecordID] = true
			continue
		}
		if pheromoneSignalRevoked(sig) {
			result.SkippedRevoked = append(result.SkippedRevoked, noteID)
			cur := "strength=unknown"
			if sig.Strength != nil {
				cur = fmt.Sprintf("strength=%.4f", *sig.Strength)
			}
			if _, herr := appendInfluenceHistory(noteID, wouldBeAction, pheromoneActorLearning, "", cur, cur,
				pheromoneOutcomeSkipReasonPrefix+"revoked (credit "+rec.RecordID+")"); herr != nil {
				stateErr = herr
			}
			newlyProcessed[rec.RecordID] = true
			continue
		}

		oldStrength := 0.0
		if sig.Strength != nil {
			oldStrength = *sig.Strength
		}
		newStrength, clamped := clampNoteStrength(oldStrength + step)

		pf.Signals[idx].Strength = &newStrength
		if err := pheromoneInfluenceSaveSignals(pf); err != nil {
			stateErr = err
			continue
		}

		before := fmt.Sprintf("strength=%.4f", oldStrength)
		after := fmt.Sprintf("strength=%.4f", newStrength)
		if clamped {
			after += " (clamped)"
		}
		if _, herr := appendInfluenceHistory(noteID, wouldBeAction, pheromoneActorLearning, "", before, after, rec.RecordID); herr != nil {
			stateErr = herr
			continue
		}
		newlyProcessed[rec.RecordID] = true
		result.NotesTuned++

		if rec.Outcome == recruitmentCreditOutcomeHarmful && harmfulCounts[noteID] >= noteHarmfulQuarantineThreshold {
			if refreshed, ridx, ok := findPheromoneSignalByID(noteID); ok && !pheromoneSignalQuarantined(refreshed.Signals[ridx]) {
				q := true
				refreshed.Signals[ridx].Quarantined = &q
				if err := pheromoneInfluenceSaveSignals(refreshed); err == nil {
					qAfter := fmt.Sprintf("quarantined=true (harmful_count=%d, threshold=%d)", harmfulCounts[noteID], noteHarmfulQuarantineThreshold)
					if _, herr := appendInfluenceHistory(noteID, wouldBeAction, pheromoneActorLearning, "", "quarantined=false", qAfter, rec.RecordID); herr == nil {
						result.Quarantined = append(result.Quarantined, noteID)
					} else {
						stateErr = herr
					}
				} else {
					stateErr = err
				}
			}
		}
	}

	if len(newlyProcessed) > 0 {
		merged := state.ProcessedRecordIDs
		if merged == nil {
			merged = map[string]bool{}
		}
		for id := range newlyProcessed {
			merged[id] = true
		}
		state.ProcessedRecordIDs = merged
		if err := savePheromoneOutcomeState(state); err != nil {
			stateErr = err
		}
	}

	if stateErr != nil {
		result.Error = stateErr.Error()
	}
	return result
}

// ---------------------------------------------------------------------------
// Strength origin reader (Task 2): the owner can always tell, for any note,
// whether its current strength was last set by them or learned -- derived
// from the most recent strength-changing history entry, never a separate
// flag that could drift from the truth.
// ---------------------------------------------------------------------------

// pheromoneStrengthOrigin reports the actor kind (owner/runtime/learning) of
// the most recent history entry that actually changed noteID's strength,
// skipping skip entries (pheromoneOutcomeSkipReasonPrefix) that recorded a
// decline rather than a movement. Returns "" when the note has no
// strength-changing history yet.
func pheromoneStrengthOrigin(noteID string) (string, error) {
	history, err := readInfluenceHistory(noteID)
	if err != nil {
		return "", err
	}
	for i := len(history) - 1; i >= 0; i-- {
		e := history[i]
		if e.Action != pheromoneActionReinforced && e.Action != pheromoneActionWeakened {
			continue
		}
		if len(e.Reason) >= len(pheromoneOutcomeSkipReasonPrefix) && e.Reason[:len(pheromoneOutcomeSkipReasonPrefix)] == pheromoneOutcomeSkipReasonPrefix {
			continue
		}
		return e.ActorKind, nil
	}
	return "", nil
}
