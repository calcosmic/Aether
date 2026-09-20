---
created: 2026-08-21T00:00:00Z
title: "[FIXED 2026-08-21] Stage/finalize deadlock — both exits pointed at each other"
area: cmd/codex_build_finalize.go + cmd/provenance.go
source: Downstream field report (Cosmic Dashboard colony, 2026-08-21) + orchestrator trace
audit_acknowledged:
  milestone: v1.26
  at: 2026-08-22
---

## Problem (reproduced downstream; recover fixed it)

Recording build results is two steps: stage, then finalize. In the failure, staging marked the
attempt "committed" but left the workers marked "planned". Then:

- finalize refused: "build attempt X is already committed, so this different completion packet
  cannot replace it… run `aether continue`" (cmd/codex_build_finalize.go:869)

- continue refused: "continue provenance: no completed worker dispatches found — build did not
  produce verifiable results" (cmd/provenance.go:148) and pointed back at the build.
Circular; no in-band exit. `aether recover` diagnosed and cleared it correctly (good — 187's
work made recover trustworthy), but the state should be unreachable.

## Root-cause shape

An inconsistent two-store write: the attempt journal and the dispatch statuses are updated
non-atomically, so a crash/failure between them leaves attempt=committed + dispatches=planned.
This is exactly Phase 188's bug class, but 188's ratchets cover COLONY_STATE.json writes — the
attempt-journal/dispatch-status pair is a different store pair and is not covered.

## Fix shape

Either make the stage step atomic across both stores (one UpdateJSONAtomically-style commit), or
make one side self-heal: finalize's "already committed" branch should notice dispatches are still
planned and either roll the attempt back to a re-finalizable state or complete the missing half —
never emit two mutually-pointing refusals. Plus a ratchet in the 188 house style covering this
store pair. Reproducible per the field report.

Relevant to ROADMAP 192's gate: "recovery failures = 0".

---

## Resolved 2026-08-21

**Reproduced, then fixed.** The trigger is an ordinary supported sequence, not a crash: build a
phase, let it finalize, then redispatch it with `--force` — which is what you do when a reviewer
finds something after sign-off. The colony still reads `BUILT` from the first attempt while the
second attempt has committed nothing.

Verbatim reproduction (throwaway probe, since promoted to a test):

```
attempt B  status=awaiting_external  completionSHA=""  claims=false  dispatches: 3/3 planned
colony     state=BUILT  currentPhase=1
finalize   "attempt X is already committed, so this different completion packet cannot
            replace it ... run `aether continue`"
continue   "no completed worker dispatches found -- build did not produce verifiable results"
```

**Root cause is misclassification, not a torn write.** The routing condition in
`runCodexBuildFinalize` sent an attempt into committed-attempt reconciliation whenever the colony
sat at `BUILT`, regardless of whether *that attempt* had committed anything. Two unlike states were
being treated as one:

| | partial commit | forced redispatch |
|---|---|---|
| what happened | this attempt's lifecycle committed, journal write lost | fresh attempt; the *previous* one built |
| completion digest | set | empty |
| claims | set | nil |
| dispatches | `completed` | `planned` |
| reconcilable? | yes | nothing to reconcile |
| does `aether continue` work? | yes | no — that is the deadlock |

Only the first is reconcilable. The second is an ordinary finalize, and the stale `BUILT` is
precisely the state `--force` exists to replace.

**Fix:** reconciliation now additionally requires `buildAttemptRecordedTerminalEvidence` — the
attempt's own completion digest and claims, written together by the terminal transition immediately
before the lifecycle commit. A fresh attempt falls through to the normal finalize path and
succeeds.

The genuine torn-write case still self-heals exactly as before: status `terminal` with evidence,
same packet, reconciles on re-run.

**Note on the finalize half of the message.** "run `aether continue`" was added on 2026-08-19 to
replace wording that sent a session chasing an impossible redispatch. It is correct advice for a
genuinely committed attempt — where the dispatches *are* completed and continue does run — and it
was being emitted in a state where it could not be followed. Narrowing the routing makes it true
everywhere it appears.

**Ratchet over the store pair** (as this ticket asked for), stated as the property rather than the
routing rule: `TestFinalizeNeverSendsUserToACommandThatRefuses` sweeps both shapes, and wherever
finalize refuses while naming a recovery command, it runs that command's own gate. A refusal naming
an exit that itself refuses fails the test however it was reached. Stated as a routing assertion
instead, it would have passed throughout the bug it exists for. Mutation-checked: reverting the fix
turns it red on the redispatch shape and leaves the legitimate shape green.

Also locked by `TestForcedRedispatchAfterBuiltIsNotADeadlock`
(cmd/build_finalize_deadlock_test.go), which asserts its own preconditions so it cannot quietly
stop exercising the deadlock.

**For anyone already stuck** in this state on disk: re-running `aether build-finalize` with the
same completion packet now works. `aether recover` still clears it and remains correct.

---

**Closed 2026-08-22:** resolved by Phase 191.1 (verification passed 6/6 on 2026-08-21) or disproven/fixed as recorded above; filed at the v1.26 close.
