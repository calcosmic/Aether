package learn

// IsLearningEligible checks all 4 conditions per D-01, D-02, D-04, D-16.
// Returns true only when all conditions are met:
//   - allWorkersSucceeded: every worker completed successfully (D-02 strictest)
//   - provenanceValid: continue provenance check passed (D-04)
//   - gatesPassed: all quality gates passed (D-01)
//   - learningEnabled: learning not disabled by config or --no-learn flag (D-16)
//
// Pure function -- no I/O.
func IsLearningEligible(
	allWorkersSucceeded bool,
	provenanceValid bool,
	gatesPassed bool,
	learningEnabled bool,
) bool {
	return allWorkersSucceeded && provenanceValid && gatesPassed && learningEnabled
}
