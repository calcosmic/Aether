package cmd

// spawnAncestorCycleReason is the ancestor-chain cycle check contract:
// called from spawnCanSpawnDecision as the third of its three checks, after
// depth and budget. A non-empty return is a human-readable deny reason; an
// empty return means allow.
//
// Plan 06 fills this body. Until then it is a contract stub: correct
// signature, wired call site, and this marker comment, which plan 10 asserts
// does not survive the phase.
func spawnAncestorCycleReason(in spawnDecisionInput) string {
	/* CONTRACT-STUB-PLAN-06 */ return ""
}
