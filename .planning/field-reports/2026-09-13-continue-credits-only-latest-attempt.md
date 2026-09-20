# `aether continue` credits only the latest build attempt — Field Report

**Date:** 2026-09-13 · **Aether:** v1.0.74
**Repo:** `CosmicDashboard` (downstream consumer, reported by a peer session)
**Outcome:** ❌ **Infinite recovery loop.** A phase whose tasks were proven across two
partial build attempts can never be advanced by following the command Aether itself
surfaces. Workaround exists (`--reconcile-task`), but the surfaced path loops forever.

**Status:** recorded, not yet reproduced locally, not yet fixed. Received mid-execution
of Phase 203 wave 2; deliberately not actioned to avoid colliding with in-flight work
on `cmd/` (plan 203-02 owns the result-binding path this touches).

---

## 1. What happened

Phase 1, 10 tasks, all showing complete in `COLONY_STATE.json`. All four gates and
10/10 gates passed on every run.

| Attempt | Tasks covered | Finalized? | `aether continue` verdict | Command it told the owner to run next |
|---|---|---|---|---|
| A | 1.2–1.8, 1.10 | ✅ cleanly | ❌ blocked | `/ant-build 1 --task 1.1 --task 1.9` |
| B | 1.1, 1.9 | ✅ cleanly | ❌ blocked | `/ant-build 1 --task 1.10 --task 1.2 … 1.8` |

Attempt B's recommended command is exactly attempt A's task set, and vice versa. Following
the surfaced recovery command therefore alternates between the two halves forever.

The block message both times:

> verification passed but no implementation evidence was recorded; reconcile completed
> tasks or redispatch missing work

## 2. Reported cause (peer's reading — NOT yet verified here)

In `cmd/codex_continue.go`, the per-task assessment reads `dispatchStatuses` from the
**current attempt only**. A task proven and finalized in an *earlier* attempt of the same
phase has no status in the current attempt, so `positiveEvidence` is false unless the task
is passed explicitly to `--reconcile-task`.

Treat this as a symptom list, not a diagnosis — this repo's own rule (see
`.planning/` history) is that downstream reports name real defects but frequently
misattribute them.

## 3. Workaround in use downstream

`aether continue --reconcile-task <ids>` for the tasks proven in the earlier attempt.
The plan-only preview then shows `positive_evidence: true`.

## 4. Candidate fixes (either, per the reporter)

1. Continue credits tasks whose receipts were admitted in **any finalized attempt of the
   current phase**, not only the latest.
2. The `--task` redispatch recovery command lists **every uncredited task at once**, not
   just the complement of the latest attempt.

## 5. The test this needs

A test that **alternates two partial redispatches** over one phase and asserts the second
`continue` credits both halves. Per CLAUDE.md's own bar this is a full-rigour case — it
decides whether work is credited — so the test must be proved able to fail by disabling
the crediting branch, and its attempt fixtures must be built the way the runtime builds
them, not typed as plausible literals.

## 6. Why this matters beyond one repo

CLAUDE.md already records that partial-completion crediting is meant to survive retries:
*"Retrying picks up only the uncredited tasks… no fresh worker is ever asked to redo
proven work"*, locked by `TestPartialRetryCommandNeverRedispatchesCreditedWork`. That
guarantee is asserted on the *grouped-job* path. This report says the same guarantee does
not hold across **attempt boundaries** on the continue path — which, if true, is exactly
the repo's signature failure mode: a guarantee proven on one lane and absent on the other.
