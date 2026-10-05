# Phase 210: context

**Requirements:** UED-19, UED-20. **Decision:** `.planning/decisions/2026-09-21-v1.29-use-it-every-day.md`. **Research:** `.planning/research/2026-09-21-popular-frameworks.md`, `.planning/research/2026-09-21-reliability-and-delivery.md`.

**Goal (ROADMAP):** The owner uses Aether for real work and only what he hits is fixed.

## What this phase actually is

This is a **usage period**, not a build phase. The deliverable is two weeks of the owner doing
real work in his own projects, a count of what stopped him, and an honest verdict against a
stopping rule that was agreed before the fortnight began rather than after it.

The milestone decision is explicit that ceremony is the enemy here: *"GSD's full ceremony is
part of how the work turned inward. Each phase here stays small and its proof is a real run in
a real chat, never paperwork, and one review round."* A phase that builds a tracking system,
a dashboard, or a reporting harness has already failed on its own terms.

## Owner decisions (2026-09-24, before planning)

<decisions>

- **D-01 — The fortnight starts as soon as the current fix lands.** [user-facing]
  The owner chose to start on the build already installed on his machine, as soon as plan
  209-07 (the false "bigger than it looked" escalation) is done.
  - Rejected: waiting for a published release first. The milestone's own scope rules put
    republishing to npm or a GitHub Release out of scope until after this phase, so the trial
    runs on the local build. Nothing in this phase may make publishing a precondition.
  - Rejected: setting up now and starting on a later word from the owner.
  - The plan must therefore not depend on a publish, a version bump, or a release tag.

- **D-02 — Which projects count is deliberately left open until the start.** [user-facing]
  Asked which of his real projects the fortnight covers, the owner answered "I'll choose
  later."
  - The plan must **not** hardcode a project list, and must not treat any particular
    repository as required. Whatever it builds or records has to accept the projects being
    named at the moment the fortnight starts, and work for one project or several.
  - UED-19's wording ("each of his real projects") is satisfied against the list he names at
    the start, not against a list invented at planning time.
  - Using Aether on Aether itself was offered and is **not** the intent of this phase; the
    milestone exists to stop the work turning inward. It is not forbidden, but it does not
    count towards UED-19 on its own.

- **D-03 — A blocker is something that made him come and tell Aether's chat about it.** [user-facing]
  The owner chose the definition that matches the milestone's own "done means" wording: he
  stopped his real work and came to the Aether chat to report or fix Aether itself.
  - Rejected: the broader "anything that made me stop and work around it" (counts irritations
    he solved himself, produces a higher count and a harsher verdict).
  - Rejected: the narrower "only things that made the work impossible" (produces a flattering
    count, which defeats the point of asking).
  - This definition is fixed **before** the fortnight, precisely so the count cannot be argued
    either way afterwards. Nothing in this phase may quietly re-cut it once the numbers are in.

</decisions>

## The stopping rule, restated so a planner cannot plan past it

From `.planning/decisions/2026-09-21-v1.29-use-it-every-day.md`:

> If, after the freeze, he still hits a blocker more than about once a week, investment in the
> full framework stops. The parts that already earn their place (quick jobs, flags, the memory
> of his preferences and lessons) are kept on top of plain Claude.

Over a fortnight, "about once a week" is roughly two. The phase must report the real count and
apply this rule honestly, including when the honest answer is that the framework does not
survive. A phase that cannot produce that answer has not done its job.

## Standing constraints

- **Only what he hits gets fixed.** A bug found by reading the code, or by an audit, or by a
  helper during the fortnight, is not in scope. The fortnight's whole purpose is to let real
  use choose the work.
- **No new features, no new strict rules** (milestone rule, unchanged).
- **Use what already exists for recording.** `aether report` already writes a bundle describing
  what Aether refused and what went wrong (UED-15, shipped in phase 208), and `/ant-flag` /
  `/ant-flags` already record and show issues and notes. The phase should lean on these rather
  than build a parallel mechanism. If something genuinely thin is missing — a tally, a
  start/end marker — that is the most that should be added, and it needs a plain-English
  reason.
- **Recording must be near-frictionless.** The owner will be mid-task in another project when a
  blocker hits. Anything that asks him to stop and fill in a form will simply not get used, and
  an unused recorder produces a falsely flattering count.

## Open items carried in, not closed by this phase

Both are phase 209's, recorded openly on the defect register rather than fixed:

- **Entry 60 — the planning hand-off is unproven live.** `/ant-go`'s repaired hand-off to the
  planning flow has never been exercised by a real unattended session (the one run that tried,
  209-TIMING.md run 5, never reached it). This is deliberately left for the fortnight to
  exercise: real use is the only thing that has ever caught a fault of this shape. If the owner
  hits it, it is a blocker and counts as one.
- **Entry 59 — the practice-project fixture leaks into the real home.** Fixture plumbing, not
  something the owner meets in ordinary use.
