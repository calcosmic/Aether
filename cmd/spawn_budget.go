package cmd

// spawnTreeBudgetReason is the whole-run tree-budget check contract
// (D-02/D-03/D-04): called from spawnCanSpawnDecision as the second of its
// three checks, after depth and before ancestor-cycle. A non-empty return is
// a human-readable deny reason; an empty return means allow.
//
// D-04 requires the tree budget to remain a visibly different quantity from
// the Queen's per-wave, pre-dispatch worker-selection cap
// (cmd/queen_spawn_budget.go) — this function counts the whole run's spawn
// tree, not one wave's dispatch list, and must never be conflated with it.
//
// Plan 05 fills this body. Until then it is a contract stub: correct
// signature, wired call site, and this marker comment, which plan 10 asserts
// does not survive the phase.
func spawnTreeBudgetReason(in spawnDecisionInput) string {
	/* CONTRACT-STUB-PLAN-05 */ return ""
}
