package cmd

import (
	"fmt"

	"github.com/calcosmic/Aether/pkg/colony"
)

// writePhaseOutcomeDocument is the ONLY producer of build/phase-N/outcome.md
// (WIRE-07, 198.2 D-11).
//
// RED phase (198.2-05 task 1): not yet implemented -- this stub exists only
// so cmd/phase_outcome_198_2_test.go compiles and its four behavior tests
// fail for the right reason (a real assertion failure, not a build error).
func writePhaseOutcomeDocument(phaseID int, result map[string]interface{}, state colony.ColonyState) error {
	return fmt.Errorf("phase outcome: writePhaseOutcomeDocument not yet implemented")
}

// stripPhaseOutcomeANSI removes terminal colour escape sequences from a
// rendered closing screen before it is persisted to outcome.md.
//
// RED phase stub: not yet implemented.
func stripPhaseOutcomeANSI(value string) string {
	return value
}
