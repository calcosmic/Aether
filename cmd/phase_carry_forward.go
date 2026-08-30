package cmd

import "github.com/calcosmic/Aether/pkg/colony"

// phaseCarryForwardBudgetChars bounds the failed/flagged item list inside
// resolvePreviousPhaseCarryForward (Task 1 RED stub -- implementation
// follows in the GREEN commit).
const phaseCarryForwardBudgetChars = 2000

// resolvePreviousPhaseCarryForward is a RED-phase stub: it always returns
// the empty string. The GREEN commit replaces this with the real
// implementation (WIRE-07, D-09..D-11).
func resolvePreviousPhaseCarryForward(currentPhaseID int) string {
	return ""
}

// precedingPhaseID is a RED-phase stub.
func precedingPhaseID(state colony.ColonyState, currentPhaseID int) (int, bool) {
	return 0, false
}
