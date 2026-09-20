---
created: 2026-08-23T00:00:00Z
title: One-worker build should go straight through (skip the check-in pause)
area: orchestration/UX
source: Owner feedback, 2026-08-23, plan 194-07 checkpoint (task 2)
raised_in_phase: 194
resolves_phase: 195
resolved: 2026-08-27
resolved_by: "Phase 195 (coherent jobs), plans 195-05 (runtime), 195-09 (wrappers/guide), 195-10 (owner docs)"
---

## The owner's observation (verbatim intent)

Asked directly whether the pre-build pause should stay when the team is a single
worker, or whether a one-worker build should go straight through:

> "And then for number three, one worker build should go straight through."

## Why this is not folded into plan 194-07

Plan 194-07's own `<resume-signal>` calls this out explicitly: the third
verification question "re-confirms the choice made on 2026-08-23 ('pause and
ask, same as today'); if that has changed, it goes back through
`/gsd-discuss-phase` as a scope change rather than into this plan." The owner's
answer reverses that choice (D-14 in
`.planning/phases/194-the-queen-decides-the-team/194-CONTEXT.md`, decided
2026-08-23): "The build pauses for the owner's approval on every build,
including a one-worker team; the existing opt-out flag is still the only way
to skip the pause, and autopilot never sees the card."

`194-07-PLAN.md`'s Task 1 already ships `TestOneWorkerTeamStillPauses`, which
locks the current (unchanged) behavior — a manifest whose only worker is the
implementation caste itself still renders a full check-in card. That test, and
the behavior it pins, are correct for what was asked for at discuss-time; they
are the wrong target now that the owner has said the opposite.

## What changing this touches

Not scoped or estimated here — this is a routing decision, not a plan. When
picked up, the discuss/plan pass should account for:

- `.claude/commands/ant/build.md`, `.claude/commands/ant-build.md`,
  `.opencode/commands/ant/build.md` (wrapper triplet — byte-identical by test,
  edit all three) — wherever the wrapper decides to render the check-in card
  and wait.
- `cmd/codex_build.go` — the runtime side of the pause decision.
- `cmd/ceremony_team_checkin_test.go` — `TestOneWorkerTeamStillPauses` asserts
  the behavior this todo proposes to reverse; it needs a decision (invert, or
  keep as the light/one-worker default vs. a new opt-out) before any code
  changes.
- The `--no-checkin` flag already exists as an explicit opt-out; this todo is
  about making a one-worker team an *implicit* one, which is a different
  (weaker) safety posture and should be discussed as such, not assumed.

## Suggested next step

Candidate for the very next planning pass after phase 194 closes — raise it in
`/gsd-discuss-phase` for whichever phase follows, so the size/safety trade-off
gets a real hypothesis rather than a default flip.

---

## Resolution — Phase 195, 2026-08-27

This supersedes "Suggested next step" above. The routing decision was taken: the
request was folded into Phase 195 (coherent jobs) as decisions D-11 through D-14
of `.planning/phases/195-coherent-jobs/195-CONTEXT.md`, and shipped.

**What the owner asked for, and what now happens.** A build whose team is a
single implementation worker no longer stops to ask for approval. It prints a
short, non-blocking summary — who is being sent, every task that worker is
taking, why those tasks belong together, and the plain statement that nothing
needs the owner's approval — and then dispatches. That holds however many tasks
the one worker is carrying, because Phase 195 also groups related tasks into one
job: "one worker" counts workers, not pieces of work.

**What was deliberately NOT weakened.** The todo warned that making a one-worker
team an *implicit* opt-out is a weaker safety posture than the explicit
`--no-checkin` flag, and asked for that to be discussed rather than assumed. It
was. The fast path is decision-aware, not count-aware: any live owner decision
still pauses the build even for a single worker. Exactly three records count as
a live decision — a safety reviewer forced by one of the five named risk signals
that the owner has not yet declined, an unanswered question raised before
planning, and an unanswered question handed back by a worker. Autopilot stays
non-interactive as before, and a new `--checkin` flag lets the owner force the
pause back on; asking for `--checkin` and `--no-checkin` at once is refused by
name rather than silently resolved one way.

**The test this todo named.** `TestOneWorkerTeamStillPauses` — which pinned the
behaviour this request reverses — was replaced, not deleted around. Its
unchanged half now lives as
`TestRenderCeremonyTeamCheckinStillRendersFullCardForOneWorkerWhenCalled`,
proving the full check-in card renderer itself is untouched; only the decision
about whether to call it moved into `decideBuildCheckin`.

### Executable closure evidence

Each command below fails if the behaviour this todo asked for is removed:

- `go test ./cmd -run TestOneWorkerBuildSkipsCheckin -count=1` — the request
  itself: a one-worker build with nothing pending does not pause.
- `go test ./cmd -run TestOneWorkerWithForcedReviewerWaiverStillPauses -count=1`
  — the safety posture the todo asked to protect: a one-worker build still
  pauses while a forced safety reviewer is undeclined.
- `go test ./cmd -run TestCheckinFlagConflictHasNoSideEffects -count=1` — the
  `--checkin` / `--no-checkin` conflict is refused before anything is written.

Supporting locks: `TestBuildCheckinDecisionMatrix` (the full precedence table),
`TestOneWorkerFastPathSummaryCarriesEveryFact` and
`TestFastPathSummaryIsNonBlocking` (the compact summary is complete and asks
nothing), `TestOneWorkerWithBoundaryQuestionStillPauses` and
`TestOneWorkerWithPersistedOwnerDecisionStillPauses` (the other two pending-decision
sources), and `TestPendingDecisionStillRendersFullCheckinCard`.

### Files this actually touched

- `cmd/ceremony_team_checkin.go` — `decideBuildCheckin`,
  `buildHasPendingOwnerDecision`, `renderBuildFastPathSummary` (plan 195-05).
- `cmd/codex_workflow_cmds.go` — the `--checkin` flag and its conflict check
  (plan 195-05).
- `.claude/commands/ant/build.md`, `.claude/commands/ant-build.md`,
  `.opencode/commands/ant/build.md` — the wrapper triplet, still byte-identical,
  now reading the runtime's decision instead of assuming `--no-checkin`
  (plan 195-09).
- `CLAUDE.md` — the owner-facing policy, which had described the old
  unconditional pause (plan 195-10).
