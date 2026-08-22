---
created: 2026-08-21T00:00:00Z
title: "[DISPROVEN 2026-08-21] Build finalize credits only the primary TaskID"
area: cmd/codex_build_finalize.go
source: Downstream field report (Cosmic Dashboard colony, 2026-08-21) + orchestrator code check
audit_acknowledged:
  milestone: v1.26
  at: 2026-08-22
---

## Problem

A merged dispatch (one worker covering N tasks via `CoveredTaskIDs`) is briefed on all N tasks
since Phase 189 — but on completion, the receipt side appears to credit only the primary
`TaskID`. `grep CoveredTaskIDs cmd/codex_build_finalize.go` → zero hits; the field lives only in
cmd/codex_build.go (population + brief rendering).

Field impact (verbatim from the downstream report): "The original Phase 1 build sent one worker
to do all six jobs but recorded it as having done only job #1. Five jobs ended up with no record
of who did them. The code was all there and passing; the receipt was missing. That's what blocked
the first /ant-continue."

## Fix shape

Finalize (and continue's task-completion bookkeeping) must credit every ID in
`CoveredTaskIDs`, not just `TaskID` — with a test that builds a real merged dispatch through
`coalesceSequentialDispatches` and asserts all N tasks end `completed` after finalize. This is
the receipt-side completion of Phase 189's brief-side fix; natural pairing.

Relevant to ROADMAP Phase 192's gate: "recovery failures = 0" and "hallucinated completions = 0".

---

## Verdict: stated cause disproven by execution (2026-08-21)

Run end to end, finalize credits **every** covered task. A three-step dependent chain merged into
one dispatch, taken through the real `mergeExternalBuildResults` and `reconcileCompletedBuildTasks`,
leaves all three tasks `completed`. Locked by `TestMergedDispatchCreditsEveryCoveredTask`
(cmd/merged_dispatch_task_credit_test.go), which was mutation-checked: crediting only
`dispatch.TaskID` turns it red with exactly this report's symptom.

The diagnosis rested on `grep CoveredTaskIDs cmd/codex_build_finalize.go` returning nothing. The
identifier's absence is not the behaviour's absence: finalize starts each reconciled dispatch from
the **manifest's** dispatch — which carries `CoveredTaskIDs` verbatim through its
`json:"covered_task_ids"` tag — overlays only the reported status, and passes it to
`reconcileCompletedBuildTasks` → `completedBuildTaskIDs` → `dispatchCoveredTaskIDs`, which expands
the chain. The completion packet never needs a `covered_task_ids` field, which is why it has none.

The fix this ticket asked for would have been a no-op.

**The symptom is still unexplained and still real.** Most likely it is the stage/finalize
inconsistency filed alongside this one
(`2026-08-21-stage-finalize-deadlock.md`): when an attempt commits while its dispatches stay
`planned`, `completedBuildTaskIDs` skips them for failing the status check, long before covered
IDs are consulted. Chase it there. The covered-task machinery has now been cleared as a suspect,
which is the point of this entry.

Second possibility worth one check before assuming: the downstream colony may have been running a
binary older than `c0a4669c` (2026-08-15), the commit that introduced covered-task crediting. Ask
what `aether version` reported there.
