package colony

import "fmt"

// WorkOutcome is the closed six-verdict work-outcome vocabulary (D-05): every
// work-cycle result is exactly one of these six values. It is a new,
// additive field -- it is never folded into OutcomeKind. OutcomeKind is the
// existing closed durable lifecycle vocabulary an older binary validates
// strictly (validateLifecycleHeader), so widening it would make a record
// written by this version undecodable by the version before it. WorkOutcome
// instead carries a total mapping onto OutcomeKind's existing values via
// LifecycleOutcome.
type WorkOutcome string

const (
	WorkOutcomeSuccess     WorkOutcome = "success"
	WorkOutcomeNoChange    WorkOutcome = "no_change"
	WorkOutcomePartial     WorkOutcome = "partial"
	WorkOutcomeBlocker     WorkOutcome = "blocker"
	WorkOutcomeTimeout     WorkOutcome = "timeout"
	WorkOutcomeInterrupted WorkOutcome = "interrupted"
)

// AllWorkOutcomes returns the six declared verdicts. This is the runtime's
// own declared set -- callers (including tests) enumerate this instead of
// maintaining a second, parallel list that can silently drift from the
// constants above.
func AllWorkOutcomes() []WorkOutcome {
	return []WorkOutcome{
		WorkOutcomeSuccess,
		WorkOutcomeNoChange,
		WorkOutcomePartial,
		WorkOutcomeBlocker,
		WorkOutcomeTimeout,
		WorkOutcomeInterrupted,
	}
}

// Valid reports whether v is one of the six declared verdicts. The zero
// value (an absent verdict) is not valid -- callers must not treat "absent"
// as "success" or as any other verdict.
func (v WorkOutcome) Valid() bool {
	switch v {
	case WorkOutcomeSuccess,
		WorkOutcomeNoChange,
		WorkOutcomePartial,
		WorkOutcomeBlocker,
		WorkOutcomeTimeout,
		WorkOutcomeInterrupted:
		return true
	default:
		return false
	}
}

// MarshalJSON follows the exact discipline OutcomeKind already uses
// (pkg/colony/lifecycle.go): an unknown value fails to encode rather than
// being written out anyway.
func (v WorkOutcome) MarshalJSON() ([]byte, error) {
	return marshalLifecycleEnum("work outcome", string(v), v.Valid())
}

// UnmarshalJSON mirrors OutcomeKind.UnmarshalJSON: an unrecognised value
// fails to decode, naming the offending value, rather than being coerced to
// a valid one. A record with no "work outcome" key at all never calls this
// method -- the field simply stays at its zero value, which Valid() reports
// as absent and IsSuccess() reports as false.
func (v *WorkOutcome) UnmarshalJSON(data []byte) error {
	raw, err := unmarshalLifecycleEnum(data, "work outcome", func(raw string) bool {
		return WorkOutcome(raw).Valid()
	})
	if err != nil {
		return err
	}
	*v = WorkOutcome(raw)
	return nil
}

// LifecycleOutcome gives the total mapping from a work verdict onto the
// existing, closed OutcomeKind vocabulary. Every declared WorkOutcome
// returns a valid OutcomeKind and no verdict falls through to a default --
// proven by iterating AllWorkOutcomes() in the test, not by listing the six
// cases a second time there.
func (v WorkOutcome) LifecycleOutcome() (OutcomeKind, error) {
	switch v {
	case WorkOutcomeSuccess:
		return OutcomeKindCompleted, nil
	case WorkOutcomeNoChange:
		return OutcomeKindNoChange, nil
	case WorkOutcomePartial:
		return OutcomeKindInProgress, nil
	case WorkOutcomeBlocker:
		return OutcomeKindFailed, nil
	case WorkOutcomeTimeout:
		return OutcomeKindFailed, nil
	case WorkOutcomeInterrupted:
		return OutcomeKindRecoveryRequired, nil
	default:
		return "", fmt.Errorf("work outcome %q has no lifecycle outcome mapping", string(v))
	}
}

// IsSuccess reports true only for the success verdict. The zero value (an
// absent verdict) is not success.
func (v WorkOutcome) IsSuccess() bool {
	return v == WorkOutcomeSuccess
}

// WorkOutcomeLabels is the single authority for each verdict's owner-facing
// wording, in plain English with no repository-invented vocabulary. Later
// rendering reads this map rather than writing its own strings, so wording
// changes happen in exactly one place. Deliberately shares no word with any
// other entry, so a non-success card can never accidentally borrow the
// success verdict's language.
func WorkOutcomeLabels() map[WorkOutcome]string {
	return map[WorkOutcome]string{
		WorkOutcomeSuccess:     "Finished successfully",
		WorkOutcomeNoChange:    "Found nothing that needed changing",
		WorkOutcomePartial:     "Got partway through the work",
		WorkOutcomeBlocker:     "Hit something it could not get past",
		WorkOutcomeTimeout:     "Ran out of time",
		WorkOutcomeInterrupted: "Was stopped before it was done",
	}
}
