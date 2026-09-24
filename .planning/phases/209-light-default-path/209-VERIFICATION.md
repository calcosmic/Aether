---
phase: 209-light-default-path
verified: 2026-09-24T19:05:00Z
status: human_needed
score: 4/4 roadmap success criteria verified on their literal wording; 1 named gap against a stronger promise the plans themselves made
verifier: orchestrator (workflow.verifier is false for this project — no separate verifier agent was spawned; every finding below was checked directly against the built binary, the git history, or the full-suite logs, never taken from a plan's own claim)
requirements:
  UED-16: complete
  UED-17: reopened
  UED-18: complete
full_suite_this_phase: >
  go test ./... -count=1 -timeout 90m ran to completion twice on this phase's final tree.
  Both runs: discovered=6057 executed=6057 — equal, so neither was the silently-truncated
  run this repository's own CLAUDE.md warns reads exactly like a clean one. Run 1 had 37
  unique failing top-level tests; run 2, after this phase fixed the seven it had itself
  caused, had exactly 30. Those 30 are byte-identical in membership to the 30 recorded as
  the pre-phase baseline on 2026-09-23 (WINDOWS row 56, commit aaab3dfa). The set
  difference run2-minus-run1 is EMPTY: nothing was newly broken. Verified by the
  orchestrator directly from /tmp/209-fullsuite.log and /tmp/209-fullsuite-2.log, not
  from any plan's summary.
---

# Phase 209 — A Light Default Path: verification

**Goal:** Ordinary work takes goal, one planning pass, build, check; everything heavier is
something the owner asks for.

## The four success criteria

### 1. One entry command; picks small or big from the size of the change, says why, and moves a job back up when small was wrong — VERIFIED

`aether go` / `/ant-go` exists, is registered on all four wrapper sources, and is reachable:
its own help line reads "Do what you asked for — the program picks a quick job or a planned
one and says which", confirmed against the built binary. The route is a computed decision in
one authority (`resolveJobSizeRoute`, `cmd/go_route.go`), not a constant, and the
self-escalation from small to big lands in `cmd/go_cmd.go`.

Every test behind this claim passes in the full-suite run above — `TestGoRouteIsComputedNotConstant`,
`TestGoRouteIgnoresHowTheSentenceIsWorded`, `TestGoRouteHasOneAuthority`,
`TestGoSmallRouteRunsTheJobEndToEnd`, `TestGoNeverRefusesTheOwner`,
`TestGoEscalationNeverAsksAndNeverBlocks`. None appears in the 30-failure list.

Plan 209-01's executor additionally proved each new test can fail, by mutating
`resolveJobSizeRoute`, `gatherJobSizeFacts`, `cmd/go_cmd.go` and the voice-corpus
registration in turn and confirming the right test went red, restoring each file
byte-identical afterwards.

### 2. One planning pass by default; discuss, specification approval and deeper rounds are opt-in — VERIFIED on its literal wording, with a named gap (see below)

The depth question is gone: the `/ant-go` wrappers run the planning route with a fixed
preset, so nothing asks the owner how deep to go. The heavier steps are opt-in, not removed —
checked directly against the built binary: `discuss`, `spec`, `plan` and `build` all still
run when typed.

**The gap is against a stronger promise the plans made beyond this criterion**, namely
"one sentence in, built work out, with no owner-typed steps in between". See
"Open gap" below. Criterion 2's own wording does not contain that clause; the phase's
requirement UED-17 and plan 209-04's must_have do.

### 3. About six menu commands visible by default, the rest behind an advanced setting — VERIFIED, as amended by the owner

The owner exercised D-02's mandatory checkpoint on the real drawn screen on 2026-09-24 and
amended the proposal from six commands to **ten in two groups** — "Everyday commands"
(`/ant-go`, `/ant-init`, `/ant-status`, `/ant-continue`, `/ant-flags`, `/ant-resume`,
`/ant-seal`) and "When you need them" (`/ant-oracle`, `/ant-swarm`, `/ant-dream`). His
recorded reasons: `/ant-init` promises "start a project" where `/ant-go` promises "do this
thing", and oracle, swarm and dream are tools he actually reaches for. He also chose the
two-group layout and had the opening lines reworded to drop the untranslated word "colony".

Verified by the orchestrator building the binary and rendering the screen in a directory
with no project set up: the shipped menu matches that ruling exactly. The full list returns
(36 lines) when `aether advanced-commands set on` is used, and was set back to off.

**"Hiding is never removing" checked directly, not assumed:** `quick`, `init`, `pause`,
`oracle`, `swarm` and `improve` were each run after being demoted from the default menu.
All six still run. `TestHiddenCommandsStillRun` holds this.

`TestDefaultMenuShowsOnlyTheApprovedSet` names the owner's ruled set literally with the
ruling date, and was proved able to fail: `/ant-dream` was deleted from the menu data, the
test went red naming it, and the file was restored byte-identical.

### 4. Timed against plain Claude on the same small and medium jobs, numbers reported to the owner — VERIFIED

`209-TIMING.md` records four real sessions run on 2026-09-24 under the owner's explicit
`measure-now` ruling, in fresh throwaway practice projects, with both job sentences written
down verbatim before any run so the two sides could not drift. Wall time and cost were read
from each session's own persisted cost record, the same method the phase 207 journey used.

The numbers are unflattering and are reported as such, which is what A-02 required:

| Job | Plain Claude | Through `/ant-go` |
|---|---|---|
| Small | 17.0s / $0.40 | 36.4s / $0.44 |
| Medium | 23.5s / $0.41 | 833.4s / $6.44 |

Neither job was faster or cheaper through Aether. No timing harness, make target or gate
entry was added — the report is a once-off measurement, exactly as A-02 required it stay.

## Open gap — requires the owner's decision

**Phase 209's own headline promise is not met in a real, unattended session.**

Plan 209-04 shipped UED-17 as "one sentence in, and the program reaches a running planning
pass and then built work without the owner typing anything in between". The first time that
hand-off was ever exercised for real — criterion 4's medium-job run — `/ant-go` **stopped and
asked the owner a question partway through**: it printed the planning-route screen, said real
dispatch needs a live helper-sender, and asked whether to "just build it" or "use Aether". It
completed only after one plain-English reply.

The test standing behind the claim, `TestGoalReachesBuiltWorkWithNoExtraSteps`, drives the Go
runtime directly with the planning stage faked already-accepted, so it cannot see this. The
gap is in the four `/ant-go` wrapper sources, not in the Go runtime.

Actions already taken, so this cannot be lost between phases:

- Filed as defect register entry **60** (open), naming the failing behaviour, why the existing
  test cannot catch it, and what would close it.
- **UED-17 un-ticked in REQUIREMENTS.md.** Its own stated proof is "a real-flow test from goal
  to built work with no extra steps"; the real flow has an extra step. Ticking it on a test
  that pre-fakes the planning stage is precisely the false-certificate pattern this project's
  Definition of Done exists to stop.

Two honest resolutions were put to the owner. **He chose to fix (2026-09-24).** Plan 209-06
was written and executed the same day.

### What plan 209-06 changed, and what is now proven

`TestGoPlanningHandoffCanActuallyStartPlanning` (`cmd/go_planning_handoff_test.go`) reads the
real, on-disk text of all four `/ant-go` wrapper sources and fails when the big-route hand-off
names no way to actually start planning.

**Proved red before the fix, independently — not taken from the executor's report.** The
orchestrator created a disposable git worktree at commit `c041b515` (the test-only commit,
before any wrapper edit), ran the test there, and watched it fail on all four sources, each
failure quoting the dead-end sentence verbatim and naming `aether plan --preset fast` as the
command that starts no worker. The worktree was removed immediately; the working checkout was
never modified. A check that could not fail would have shown green there.

The hand-off now delegates rather than duplicates: it skips `/ant-plan`'s specification
preflight and preset card (both already settled by the single door), names
`aether host plan --preset fast` — the real dispatch entry point that yields a manifest — and
then names each of `/ant-plan`'s own stages to follow in order through to exact candidate
acceptance, then `/ant-build`'s flow, then `/ant-continue`. It states plainly that the
assistant must not hand any choice of how to proceed back to the owner. The 281-line planning
flow was not copied into the door, so the two cannot drift.

Green after the fix, together with the parity check and the tests this gap sits between
(`TestClassicCommandParity`, `TestGoalReachesBuiltWorkWithNoExtraSteps`,
`TestDiscussAndSpecStillBehaveExactlyAsBefore`), confirmed by the orchestrator directly.

### What is still NOT proven, and why defect 60 stays open

No real, unattended chat session has run the medium job end to end since the fix. The check
that now exists proves the hand-off *names* the right commands; it cannot prove a live
assistant *follows* them without stalling. Only another real, metered session can show that —
the same kind of run that found the bug in the first place, and the only kind that has ever
caught a fault of this shape here.

Defect register entry 60 therefore stays **open**, with the wrapper fix landed and locked by a
named test, and one live confirmation outstanding. UED-17's tick in REQUIREMENTS.md carries
that same caveat in its own words rather than claiming more than was shown.

## Also filed this phase

- Defect register entry **59** (open): the practice-project fixture's own setup steps sync
  Aether's shared slash commands into this machine's real global folders, contradicting the
  script's own isolation comment. Worked around at the call site during the measurement;
  confirmed nothing real was written. Fixing the script is out of this phase's scope.

## Regressions

None. See `full_suite_this_phase` above — the failing set is identical to the recorded
pre-phase baseline and the newly-broken set is empty.
