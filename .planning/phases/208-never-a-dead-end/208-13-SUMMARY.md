---
phase: 208-never-a-dead-end
plan: 13
subsystem: refusal-lifecycle
tags: [go, refusal-contract, colonize, unattended-session, cli, mutation-testing]

# Dependency graph
requires:
  - phase: 208-never-a-dead-end (plans 01-12)
    provides: attemptRefusalSelfRecovery and refusalSelfRecoveryTable
      (cmd/refusal_self_recovery.go, plan 11), the second gap-closure code
      review that independently reproduced two guards unable to fail
      (208-REVIEW-GAP2.md), and the re-verification that confirmed both
      by mutation (208-VERIFICATION.md)
provides:
  - "mayReadTheIsAnyoneHereFact (cmd/refusal_self_recovery_test.go): the
    closed, checked-in allow-list of the three files permitted to read the
    is-anyone-here fact; TestSelfRecoveryHasOneDecision now fails by name
    on any other non-test file that reads it at all"
  - "TestOnlyADeclaredRefusalIsEverRecoveredFrom: drives
    attemptRefusalSelfRecovery directly, covering the opt-in-table,
    protects-work and names-a-command gates each on their own, with every
    id/command derived from the real registry"
  - "attemptRefusalSelfRecovery now also re-reads the authoritative row via
    refusalForID before recovering (WR-08), so the runtime gate and the
    build-time contract checker agree on one source"
  - "renderRefusalSelfRecoveryNotice no longer claims a command has already
    run; refusalSelfRecoveryTable's stored value is now a plain,
    owner-readable sentence with no planning-decision id or filename"
  - "TestSelfRecoveryNoticeSaysOnlyWhatActuallyHappened and
    TestSelfRecoveryNoticeSpeaksPlainEnglish, the latter reusing the shared
    untranslatedRepoWords predicate"
affects: []

# Actuals (#2632)
actuals:
  tokens: 6516
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "A structural one-decision guard now enforces its invariant via a
      closed, checked-in allow-list of files rather than a textual
      call-shape heuristic -- the allow-list is the mechanism, and it may
      only shrink without an owner ruling."
    - "A safety-relevant gate re-reads its own authority (the registry row)
      rather than trusting a caller-supplied copy of the same fields a
      second time, closing the gap between a runtime gate and its
      build-time contract checker."

key-files:
  created: []
  modified:
    - cmd/refusal_self_recovery.go
    - cmd/refusal_self_recovery_test.go

key-decisions:
  - "cmd/codex_colonize.go was deliberately left untouched. The plan's own
    instruction was conditional: pass a lane flag through the two call
    sites only if a single shared sentence genuinely cannot be made true
    for both colonize lanes. 'Aether is going ahead and rebuilding the map
    of your code instead of stopping to ask' is true at the moment it
    prints on both the plan-only lane (which has only set ForceResurvey)
    and the direct lane (which goes on to re-survey synchronously), so no
    lane flag was needed and the plan's own frontmatter files_modified
    entry for cmd/codex_colonize.go was not exercised."
  - "Task 1 and Task 2 both touch the same two files in overlapping
    functions (attemptRefusalSelfRecovery's signature change threads
    through renderRefusalSelfRecoveryNotice, which Task 2 also rewords).
    Following this phase's own 208-11 precedent, Task 2's wording/table/
    test changes were temporarily reverted, Task 1 committed alone, then
    reapplied and committed alone, so each commit's diff matches only its
    own task's declared scope."
  - "WR-08's authoritative re-check reads the command for the notice from
    the registry row, not the caller's own refusal struct, but the refusal
    log record (appendRecoveredRefusalToLog) still records the caller's
    own fields -- the plan named only the notice as needing this, and
    refuse(...) already populates the caller's struct from the same
    registry row on every real call site, so this is not a behavior change
    for anything except the three adversarial test cases built to catch a
    caller/registry mismatch."

patterns-established:
  - "A structural guard's allow-list is itself the checked-in, testable
    artifact -- widen the guard's predicate and name exactly who is exempt,
    rather than widening the predicate's own shape until it happens to
    exclude the files that need to be exempt."

requirements-completed: [UED-10]

coverage:
  - id: D1
    description: "The one-decision guard (TestSelfRecoveryHasOneDecision) now fails by name when a second self-recovery decision is planted anywhere in the module outside a closed, checked-in allow-list of three files -- including a decision that only ever receives an already-built refusal value and never calls refuse(...) itself, which is exactly the shape that previously passed undetected."
    requirement: "UED-10"
    verification:
      - kind: unit
        ref: "cmd/refusal_self_recovery_test.go#TestSelfRecoveryHasOneDecision"
        status: pass
      - kind: other
        ref: "Mutation proof: planted secondSelfRecoveryDecision in cmd/helpers.go's outputRefusal inside a disposable worktree -- the guard failed naming cmd/helpers.go; reverted, guard passed again."
        status: pass
    human_judgment: false
  - id: D2
    description: "attemptRefusalSelfRecovery's opt-in-table, protects-work and names-a-command gates each have their own failing assertion -- deleting any one of the three individually, or all three together, now makes TestOnlyADeclaredRefusalIsEverRecoveredFrom fail, naming the specific case that wrongly recovered."
    requirement: "UED-10"
    verification:
      - kind: unit
        ref: "cmd/refusal_self_recovery_test.go#TestOnlyADeclaredRefusalIsEverRecoveredFrom"
        status: pass
      - kind: other
        ref: "Mutation proof: all-three deletion, then each of the three gates deleted individually, each applied and reverted in a disposable worktree -- all four variants turned the test red naming the exact failing case; restored state passes."
        status: pass
    human_judgment: false
  - id: D3
    description: "The owner-facing self-recovery notice no longer claims, in the past tense, that a command has already run -- true on both the direct and plan-only colonize lanes at the moment it prints -- and carries no planning-decision id, no .planning/ filename, and no unexplained invented word."
    requirement: "UED-10"
    verification:
      - kind: unit
        ref: "cmd/refusal_self_recovery_test.go#TestSelfRecoveryNoticeSaysOnlyWhatActuallyHappened"
        status: pass
      - kind: unit
        ref: "cmd/refusal_self_recovery_test.go#TestSelfRecoveryNoticeSpeaksPlainEnglish"
        status: pass
      - kind: other
        ref: "Mutation proof: restored the old past-tense wording and, separately, the old table value carrying 'D-03, 208-CONTEXT.md' -- each mutation, applied and reverted in a disposable worktree, turned its own test red naming what it found; restored state passes."
        status: pass
    human_judgment: false
  - id: D4
    description: "The end-to-end visual-mode assertion in TestNoOneHereMeansAetherRefreshesTheMapItself now anchors on the notice's own banner heading and no-one-is-here line, not merely on the command string, so it cannot pass on unrelated colonize output that happens to mention the same command."
    requirement: "UED-10"
    verification:
      - kind: unit
        ref: "cmd/refusal_self_recovery_test.go#TestNoOneHereMeansAetherRefreshesTheMapItself"
        status: pass
      - kind: other
        ref: "Mutation proof: replaced renderRefusalSelfRecoveryNotice with a version emitting only the command string, in a disposable worktree -- the subtest failed naming the missing banner heading; reverted, subtest passed again."
        status: pass
    human_judgment: false

duration: ~35min
completed: 2026-09-24
status: complete
---

# Phase 208 Plan 13: The Self-Recovery Guards Can Now Actually Fail Summary

**Widened `TestSelfRecoveryHasOneDecision` to a closed allow-list, added a direct-call test covering each of `attemptRefusalSelfRecovery`'s safety gates on its own, and rewrote the owner-facing notice to stop claiming a command already ran and stop printing a planning-decision id — all four fixes proven by applying each of 208-VERIFICATION.md's own reproduced mutations in a disposable worktree and watching the right test turn red.**

## Performance

- **Duration:** ~35 min
- **Started:** 2026-09-24 (session start)
- **Completed:** 2026-09-24T07:34:52Z
- **Tasks:** 2 planned, both completed
- **Files modified:** 2 (`cmd/refusal_self_recovery.go`, `cmd/refusal_self_recovery_test.go`)

## Accomplishments

- **The one-decision guard can now fail on the exact defect it is named for.** `mayReadTheIsAnyoneHereFact` is a new, closed, checked-in allow-list of the three non-test files legitimately allowed to read the is-anyone-here fact (`cmd/unattended_session.go`, `cmd/refusal.go`, `cmd/refusal_self_recovery.go`). `TestSelfRecoveryHasOneDecision` now fails by name on any other file that either names `refusalSelfRecoveryTable` or calls `sessionHasNoOneToAsk` at all — dropping the old "must also call `refuse(...)`" requirement that let a second decision planted in `cmd/helpers.go`'s `outputRefusal` (a function that only ever receives an already-built refusal value) sail through undetected.
- **Every safety gate inside `attemptRefusalSelfRecovery` now has its own failing test.** `TestOnlyADeclaredRefusalIsEverRecoveredFrom` drives the function directly (never through a cobra command), with every id, command and flag derived from the real `refusalRegistry`/`refusalSelfRecoveryTable` rather than typed as a literal. It covers: a registered work-protecting stop that is not opted in, the opted-in row with `ProtectsWork` forced false, the opted-in row with its next command blanked, and a positive control proving the function still returns `true` for the untouched, genuinely-declared row.
- **The runtime gate and the build-time contract checker now agree on one source (WR-08).** `attemptRefusalSelfRecovery` re-reads the authoritative row through `refusalForID` before recovering, and the notice's command comes from that registered row rather than the caller's own copy.
- **The attended-behavior proof no longer depends on an unset environment variable (WR-04).** Both subtests of `TestAttendedColonizeStillStopsAndAsks` now pin `AETHER_UNATTENDED` explicitly to empty, so the proof cannot turn red for an ambient-environment reason — confirmed by re-running the scoped verify command with `AETHER_UNATTENDED=1` exported in the real shell.
- **The owner-facing notice no longer claims something false.** `renderRefusalSelfRecoveryNotice` now says Aether is going ahead and rebuilding the map of the code instead of stopping to ask — true of both the direct and plan-only colonize lanes at the moment it prints — keeping the command name as supporting evidence in parentheses rather than as the claim itself.
- **The stored table value is now plain English with no internal bookkeeping (WR-03).** `refusalSelfRecoveryTable`'s one entry no longer embeds a planning-decision id (`D-03`) or a `.planning/` filename; that developer rationale moved into a Go comment beside the row, and the table's own doc comment now says the stored value is printed verbatim to a person.
- **Two new tests lock both wording fixes**, and reuse the shared `untranslatedRepoWords` plain-English predicate (`next_action_card_test.go`) rather than a second, competing definition.
- **The end-to-end visual assertion no longer gets lucky (WR-07).** It now also anchors on the notice's own banner heading (derived from the real, unmutated `renderBanner` helper) and its no-one-is-here line, so it cannot pass on unrelated colonize output that merely happens to mention the same command.

## What the owner now sees on this screen

Before this plan, the screen an owner would have seen if they ever ran an unattended colonize said Aether had *already run* the re-survey command, and its explanation carried an internal code and a filename from this project's own planning folder. Now it says Aether is *going ahead* and rebuilding the map of the code instead of stopping to ask — true the moment it is printed on either lane — and the explanation is a plain sentence with no internal bookkeeping in it.

## Task Commits

1. **Task 1: Make both guards able to fail, proved by the verification's own two mutations** - `904a13e6` (fix) — widened `TestSelfRecoveryHasOneDecision` to the closed allow-list, added `TestOnlyADeclaredRefusalIsEverRecoveredFrom`, the WR-08 authoritative-row re-check inside `attemptRefusalSelfRecovery`, and the WR-04 explicit env pins.
2. **Task 2: The screen stops telling the owner two things that are not true** - `7e306750` (fix) — reworded `renderRefusalSelfRecoveryNotice`, split the table's stored value from its developer rationale, added `TestSelfRecoveryNoticeSaysOnlyWhatActuallyHappened` and `TestSelfRecoveryNoticeSpeaksPlainEnglish`, and anchored the WR-07 visual-mode assertion on the notice's own banner and no-one-is-here line.

_Both tasks touch `cmd/refusal_self_recovery.go` and `cmd/refusal_self_recovery_test.go` in overlapping functions (Task 1's signature change to `renderRefusalSelfRecoveryNotice` is reworded by Task 2). Following 208-11's own precedent for this exact situation, Task 2's wording/table/test changes were temporarily reverted, Task 1 was verified and committed alone, then Task 2's changes were reapplied, verified, and committed alone — so each commit's diff matches only its own task's declared scope._

## Files Created/Modified

- `cmd/refusal_self_recovery.go` — widened doc comments, the WR-08 authoritative re-check in `attemptRefusalSelfRecovery`, the reworded notice renderer, and the split table value
- `cmd/refusal_self_recovery_test.go` — `mayReadTheIsAnyoneHereFact`, the widened `TestSelfRecoveryHasOneDecision`, `TestOnlyADeclaredRefusalIsEverRecoveredFrom` + its `assertRefusalNeverRecovered` helper, the WR-04 env pins, the WR-07 anchor, and the two new notice-wording tests

## Decisions Made

See `key-decisions` in frontmatter. In brief: `cmd/codex_colonize.go` was deliberately left untouched because a single shared sentence proved true on both colonize lanes, exactly the condition the plan's own instruction made passing a lane flag through that file conditional on; the two tasks were split into separate commits by temporarily reverting/reapplying Task 2's changes, matching this phase's own 208-11 precedent for the identical situation; and the refusal-log record still carries the caller's own fields (only the notice's command was required to move to the registry's authoritative copy).

## Deviations from Plan

None — plan executed as written, including the frontmatter's own conditional escape hatch for `cmd/codex_colonize.go` (see key-decisions above; it was not needed rather than skipped).

### Mutation Proof (per CLAUDE.md's Definition of Done — a test must be able to fail)

All mutations were applied and reverted in a disposable git worktree created under the session scratchpad (named to contain "Aether" per this repo's `TestResolveAetherRoot_GitFallback` constraint), never in this checkout. `git worktree remove --force` ran after every proof; `git status --short` in the main checkout showed only the plan's own two files changed throughout.

| Mutation | Test | Observed failure (verbatim, abbreviated) |
|---|---|---|
| Gap 1: planted `secondSelfRecoveryDecision` in `cmd/helpers.go`'s `outputRefusal` | `TestSelfRecoveryHasOneDecision` | `only cmd/unattended_session.go, cmd/refusal.go and cmd/refusal_self_recovery.go may hold a self-recovery decision or read the is-anyone-here fact; found a second reader also in: [cmd/helpers.go (reads the is-anyone-here fact outside the allow-list)]` |
| Gap 2, all three gates deleted (opt-in lookup, protects-work, names-a-command) | `TestOnlyADeclaredRefusalIsEverRecoveredFrom` | `build-dispatch-manifest-wrong-source is not a key of the opt-in table and must never be recovered from` |
| Gap 2, opt-in-list lookup deleted alone | `TestOnlyADeclaredRefusalIsEverRecoveredFrom` | `build-dispatch-manifest-wrong-source is not a key of the opt-in table and must never be recovered from` |
| Gap 2, protects-work check deleted alone | `TestOnlyADeclaredRefusalIsEverRecoveredFrom` | `colonize-existing-survey-found does not protect work and must never be recovered from` |
| Gap 2, names-a-command check deleted alone | `TestOnlyADeclaredRefusalIsEverRecoveredFrom` | `colonize-existing-survey-found names no command and must never be recovered from` |
| WR-02: restored the old past-tense wording (`Aether ran \`...\` on your behalf...`) | `TestSelfRecoveryNoticeSaysOnlyWhatActuallyHappened` | `the notice claims the command has already run, which is false on the plan-only lane at the moment it prints:` (followed by the rendered notice showing `Aether ran \`aether colonize --force-resurvey\` on your behalf and is carrying on.`) |
| WR-03: restored the old stored sentence carrying `(D-03, 208-CONTEXT.md)` | `TestSelfRecoveryNoticeSpeaksPlainEnglish` | `the notice carries a planning-decision identifier ("D-03"):` |
| WR-07: replaced the notice renderer with one emitting only the command string | `TestNoOneHereMeansAetherRefreshesTheMapItself/visual_output_names_the_exact_command_Aether_carried_out` | `the self-recovery notice's own banner heading never appeared:` (output showed only `/ant-colonize --force-resurvey` and unrelated colonize banners) |

All eight reverted changes were restored and re-verified green (`go build ./...` clean, each named test passing again) before the worktree was removed.

## Issues Encountered

None.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

Both of 208-VERIFICATION.md's `gaps_found` entries for this round are closed: the two safety guards for `attemptRefusalSelfRecovery` can now genuinely fail on the mutations that previously passed them unnoticed, and the owner-facing notice they gate says only what is actually true. This plan's own scope was deliberately narrow (the guard proofs and the wording fixes on the one already-shipped self-recovery mechanism); it did not run a live journey walk and did not touch the journey code, the refusal register, or any wrapper file.

**Deliberately out of scope this round, carried forward as named, still-open observations (per Task 2's own instruction):**
- **WR-01** — the decision function announces and records the recovery but does not itself carry it out; a caller returning early or erroring after receiving `true` would leave a false record. A larger design change (the function taking the recovery action itself), not addressed here.
- **WR-05** — `emitVisualProgress`'s silence in machine-output mode is only half its gate; a future opt-in row on a command the ceremony taxonomy classifies as quiet or unlisted would recover invisibly, printing nothing to a person sitting there. Not addressed here; today's one row (`colonize`) is safe because `colonize` is worker-theatre, not quiet-classified.
- **WR-06** — the declared key_link (`opts.ForceResurvey -> manifest.ForceResurvey -> the sibling existing-survey check does not fire`) is proven only at its first hop; nothing drives the plan-only lane's recovered manifest through `runCodexColonizeFinalize` in a test. Reading the code confirms the chain holds, but this remains a proof gap in the same segment WINDOWS entry 53 lives in.

The still-open SC2 item from 208-VERIFICATION.md (the journey has never progressed past step 2 of 14 in a real walk, so `journeyRunPrintedNextCommands` remains unexercised live) is untouched by this plan and remains the larger open item for a later round — this plan's `<prohibitions>` explicitly forbade spending a real journey walk here.

## Self-Check: PASSED

- FOUND: cmd/refusal_self_recovery.go
- FOUND: cmd/refusal_self_recovery_test.go
- FOUND: .planning/phases/208-never-a-dead-end/208-13-SUMMARY.md
- FOUND commit: 904a13e6
- FOUND commit: 7e306750

---
*Phase: 208-never-a-dead-end*
*Completed: 2026-09-24*
