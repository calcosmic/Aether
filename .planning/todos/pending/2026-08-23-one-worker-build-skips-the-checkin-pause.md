---
created: 2026-08-23T00:00:00Z
title: One-worker build should go straight through (skip the check-in pause)
area: orchestration/UX
source: Owner feedback, 2026-08-23, plan 194-07 checkpoint (task 2)
resolves_phase: 194
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
