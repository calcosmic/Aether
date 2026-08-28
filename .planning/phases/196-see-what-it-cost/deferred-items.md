# Phase 196 — deferred items

Things found while executing this phase that were deliberately NOT done here,
recorded so they are actionable rather than forgotten. Written by plan 196-08.

## D1 — Four workflows spawn workers and record nothing

**Found during:** 196-08, task 1 (the reachability proof).

Phase 196's promise is scoped to building a phase and checking it, and after
this plan both of those record their cost on both lanes (see the lane table in
`196-08-SUMMARY.md`). Four other workflows also spawn workers and file no token
row at all, so their cost is invisible:

| Workflow | Where its workers are spawned |
|---|---|
| `aether plan` | `cmd/codex_plan.go` |
| `aether colonize` | `cmd/codex_colonize.go` |
| `aether seal` (its final review) | `cmd/seal_final_review.go` |
| the automatic fix attempt | `cmd/check_fix_attempt.go` |

`aether quick` and `aether swarm` are in the same position.

**Why it was not done here.** The ledger is keyed by phase AND workflow word,
and the writer refuses any word other than `build` or `continue` rather than
quietly creating a third file that nothing adds up. Adding these means choosing
what a planning run or a seal is keyed under, and whether the cost line — which
says "What This Phase Has Cost" — should include work that is not a phase.
That is a scope decision, not a wiring gap, and it belongs to whoever picks it
up rather than to a closeout plan.

**What to do.** Decide the key first (a new workflow word per workflow, or one
shared `other` key), then extend `normalizeSpendRowStatus`'s sibling check in
`cmd/spend_ledger.go` and add one `writeSpendRowsForRun` call per workflow. The
end-to-end shape to copy is `fileDirectContinueSpendRows` in
`cmd/spend_writer.go`, and `TestSpendPipelineIsReachableEndToEnd` is the test
shape that would prove it.

## D2 — A directly-spawned worker whose provider emits no usage event

**Found during:** 196-08, task 1.

On the direct lanes the provider's own event is the only source of a figure —
there is no chat-platform session artifact, because the worker is a subprocess
of the runtime rather than a subagent inside somebody's chat session. A worker
whose provider emits no usage event therefore shows a dash and stays out of the
total.

That is correct and deliberate under D-01 as amended (no number is better than a
number the owner cannot trust), and it is not a bug. It is recorded here only so
a later reader who sees a dash on the direct lane knows it means "the tool said
nothing", not "the wiring is missing" — the wiring is proved by
`TestSpendPipelineIsReachableEndToEnd`.
