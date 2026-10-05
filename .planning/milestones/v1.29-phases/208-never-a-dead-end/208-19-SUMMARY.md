---
phase: 208-never-a-dead-end
plan: 19
subsystem: defect-register
tags: [go-test, mutation-proof-verification, windows-ledger, roadmap-closeout]

# Dependency graph
requires:
  - phase: 208-never-a-dead-end (plans 17, 18)
    provides: "the two mutation proofs this plan checks the register against:
      208-17's widened one-builder guard (composite-literal case,
      TestSavedMapPublicationHasOneBuilder) and 208-18's narrowed one-decision
      guard (five reviewed places, TestSelfRecoveryHasOneDecision), each with
      its own verbatim red-then-green mutation output already recorded in
      208-17-SUMMARY.md / 208-18-SUMMARY.md"
provides:
  - "Confirmation, by re-running every check family this round could plausibly
    have disturbed in fresh scoped commands, that nothing outside 208-17's and
    208-18's own three files moved: every failure observed is either the two
    pre-existing, already-catalogued TestCodexNativeCancellation failures
    (WINDOWS row 56) or nothing at all."
  - ".planning/WINDOWS.md rows 57 and 58 marked fixed via the register tool's
    own fixed subcommand (never hand-edited timestamps), with each row's
    description amended in place to name the exact mutation that proved it
    and to correct row 57's backwards 'Related:' clause."
  - ".planning/ROADMAP.md's Phase 208 section corrected: nineteen of nineteen
    plans executed, the Wave 16 entry ticked, and the closing account
    rewritten to say the owner reopened the phase for a fourth round that
    closed all six review findings, rather than closing it with two gaps
    left open."
affects: [defect-register, phase-208-closeout]

# Actuals (#2632)
actuals:
  tokens: 4517
  tasks: 2
  commits: 1

tech-stack:
  added: []
  patterns:
    - "A register row is marked fixed only after independently re-confirming,
      from the SUMMARY that recorded it, that the row's own named mutation
      was actually planted and actually turned the named check red -- the
      row is never marked fixed on the strength of an earlier plan's say-so
      alone."
    - "Register status changes go through the ledger tool's own fixed
      subcommand (gsd-tools windows fixed <id>), never a hand-edited status
      field or timestamp, so the frontmatter open/fixed counts stay
      mechanically consistent with the table."

key-files:
  created: []
  modified:
    - .planning/WINDOWS.md
    - .planning/ROADMAP.md

key-decisions:
  - "Row 57's 'Related:' clause was backwards -- it said the check being
    closed lacked the leftover-worktree skip its sibling had, when in fact
    the check being closed already had that skip and it was the sibling (the
    one-decision guard, closed by row 58) that was missing it. Corrected as a
    visible correction rather than a silent deletion, per the plan's
    instruction."
  - "The four scoped test families ran as separate commands, each well inside
    the eight-minute budget, rather than one combined run, so a slow family
    could not risk the whole check being pushed into the background and its
    result lost."

patterns-established: []

requirements-completed: [UED-10]

coverage:
  - id: D1
    description: "Every check family this round could plausibly have disturbed was re-run in fresh, scoped commands, and every failure observed was classified as exactly one of: already on the known-red list (named by row), genuinely new, or could not be run. No genuinely new failure was found; the only failures seen were the two already-catalogued TestCodexNativeCancellation failures (WINDOWS row 56). git status, git worktree list and git diff --stat over the sensitive paths confirm nothing outside 208-17's and 208-18's own three files moved."
    requirement: "UED-10"
    verification:
      - kind: other
        ref: "go test ./cmd -run '<this round's 10 own tests>' -count=1 -timeout 8m"
        status: pass
      - kind: other
        ref: "go test ./cmd -run 'Colonize|Territory' -count=1 -timeout 8m"
        status: pass
      - kind: other
        ref: "go test ./cmd -run 'Refusal|Refuse|Unattended|SelfRecovery' -count=1 -timeout 8m"
        status: pass
      - kind: other
        ref: "go test ./cmd -run '<refusal contract + wrapper fence + journey-gate refusals>' -count=1 -timeout 8m"
        status: pass
    human_judgment: false
  - id: D2
    description: "WINDOWS.md rows 57 and 58 are marked fixed via the register tool's own subcommand, each with a closing note naming the exact planted mutation that proved it; row 57's backwards 'Related:' clause is corrected as a visible correction. Row 53 is confirmed byte-for-byte unchanged and still open. ROADMAP.md's Phase 208 section now reads nineteen of nineteen plans executed, with Wave 16 ticked and the closing paragraph correctly describing the owner's fourth-round reopening and its outcome."
    requirement: "UED-10"
    verification:
      - kind: other
        ref: "grep -c '^| 53 .*| open |' .planning/WINDOWS.md == 1"
        status: pass
      - kind: other
        ref: "gsd-tools windows status --raw (open_count/fixed_count/total_count internally consistent: 36+0+22=58)"
        status: pass
    human_judgment: false

duration: 30min
completed: 2026-09-24
status: complete
---

# Phase 208 Plan 19: Prove Nothing Else Moved, Close the Register on Evidence Summary

**Re-ran every check family this round of fixes could have disturbed and found nothing new broken; marked the two defect-register entries fixed only after confirming each one's own recorded mutation proof, corrected a backwards sentence in one of them, and rewrote the phase's own closing account so it says what actually happened.**

## Performance

- **Duration:** ~30 min
- **Tasks:** 2 completed
- **Files modified:** 2 (`.planning/WINDOWS.md`, `.planning/ROADMAP.md`)

## What this closes, in plain English

This project keeps one running list of known problems (`.planning/WINDOWS.md`, called "the defect
register" here). Two entries on that list -- numbered 57 and 58 -- said exactly what would have to
be true before they could be marked fixed: a specific mistake, planted on purpose, would have to make
a specific safety check fail. The previous two pieces of this round's work (plans 17 and 18) did
exactly that and wrote down, word for word, what the check said when it failed and when it passed
again afterward.

This piece of work had two jobs. First: check that fixing those two things didn't quietly break
anything else, by re-running every group of the program's own automated checks that this round could
plausibly have touched, in fresh commands, and sorting whatever came back into three honest piles --
already-known problems, brand-new problems, or things that couldn't be run at all. Second: only once
that was confirmed clean, actually mark the two register entries fixed -- using the register's own
tool rather than hand-typing a status -- and write, on each entry, exactly which planted mistake
proved it. A third, older entry on the same list (the one saying the full practice-project rehearsal
has never finished all fourteen of its steps in one real run) was left completely alone, because
nothing in this round touched it and it is not this round's to close.

Separately, this project's own plan document (`.planning/ROADMAP.md`) had a paragraph describing how
Phase 208 ended that was now wrong -- it said the owner closed the phase with two problems left open,
which was true for about a day, but the owner then reopened the phase for one more short round that
fixed both of those problems plus four smaller ones. That paragraph now says what actually happened.

## Task Commits

1. **Task 1: Run the project's own checks and report them split three ways** - no commit (this task
   only ran checks and confirmed nothing needed changing; verified clean, nothing to stage)
2. **Task 2: Close the two register entries on evidence, leave the third open, and correct the
   phase's own account** - `f24465a2` (docs)

## Files Created/Modified

- `.planning/WINDOWS.md` -- rows 57 and 58 marked `fixed` via `gsd-tools windows fixed <id>` (status
  and `resolved_at` set by the tool, not by hand); each row's description amended with a closing note
  naming its own proving mutation; row 57's backwards "Related:" clause corrected. Row 53 and every
  other row untouched. Frontmatter counts (`open_count`, `fixed_count`, `last_updated`) updated
  automatically by the tool and confirmed internally consistent (36 open + 0 waived + 22 fixed = 58
  total).
- `.planning/ROADMAP.md` -- Phase 208's `**Plans:**` line now reads nineteen of nineteen executed; the
  Wave 16 entry for this plan is ticked; the closing `**Result:**` paragraph's final sentences rewritten
  to describe the owner's fourth-round reopening and its outcome, with the "what is still unproven"
  sentence left exactly as it was.

## Decisions Made

See `key-decisions` in the frontmatter: the correction to row 57's backwards "Related:" clause, and
the choice to run the four check families as separate scoped commands rather than one combined run.

## Deviations from Plan

None -- plan executed exactly as written. Both tasks matched their acceptance criteria without
requiring any Rule 1-4 auto-fixes or architectural questions.

## What Passed, What Failed, What Could Not Be Run

**What passed (every command completed inside its own eight-minute budget):**

- `go build ./...` -- clean.
- `go vet ./cmd/... ./pkg/...` -- clean.
- This round's own ten new/changed checks (the widened one-builder guard, the new lighter-survey-team
  test, the existing saved-map refresh test, the narrowed one-decision guard, the new per-row wording
  test, the opt-in contract guard, the direct-decision test, and both notice-wording tests, plus the
  existing attended-refusal-text guard) -- all pass, 6.7s.
- The surveying and saved-picture family (names containing "Colonize" or "Territory") -- all pass,
  24.4s, no failures at all.
- The refusal contract and wrapper-fence family (the table-behaviour match, the every-row-names-a-
  command check, the register sort-and-unique check, the registered-or-counted check, the untyped
  floor, the colonize wrapper's act-when-alone rule, and the rehearsal gate's own three refusal
  checks) -- all pass, ~1s.

**What failed (and why it is not new):**

- Two tests in the refusal/unattended/self-recovery family: `TestCodexNativeCancellationRefusalReplay`
  and `TestCodexNativeCancellationRefusalEvidence`. Both are already named on this project's own
  known-red list (WINDOWS row 56) and are explicitly called out in this plan's own operating
  constraints as pre-existing, not regressions, and not this plan's to fix. No other failure appeared
  in that family (72.9s total).

**What could not be run:** nothing. Every planned command completed and reported a result inside its
budget; none needed to be re-split or re-run.

**One plain-English line an owner can read:** apart from the two already-known, already-tracked
problems named above, nothing else in the program moved -- everything else this round could have
disturbed still works exactly as it did before.

## Confirmation nothing outside this round's own files changed

- `git status --short` showed only the one pre-existing, not-mine file (`.aether/CONTEXT.md`) before
  this plan's own edits, and only this plan's own two files (`.planning/WINDOWS.md`,
  `.planning/ROADMAP.md`) plus that same pre-existing file afterward.
- `git worktree list` showed only the pre-existing worktrees this repository already had
  (`Aether-release-1.0.86`, `.claude/worktrees/agent-af06650f4d83f8d05`) -- no leftover disposable
  worktree from any mutation proof.
- `git diff --stat` over `cmd/journey*.go`, `scripts/build-messy-practice-project.sh`,
  `cmd/refusal_register.go`, `.aether/commands/`, `.claude/commands/` and `.opencode/commands/` was
  empty.

No paid walk was run and nothing was spent this round: no `make eval-gate-journey`, no
`AETHER_JOURNEY_TRIALS` run, no real `claude -p` chain, no `make prove-journey-fix-reverts`.

## Register Rows: What Was Closed, What Stayed Open

- **Row 57** (the one-builder guard's blind spot to a struct-literal write) -- marked `fixed`. Closing
  note: the exact mutation that proved it was a function added to the workflow-commands file
  (`cmd/codex_workflow_cmds.go`) returning a manifest value with its lane-selecting field set inside
  the literal that builds the value, rather than assigned afterward on its own line -- this is verbatim
  what 208-17-SUMMARY.md recorded as observed and reverted. The row's backwards "Related:" clause is
  corrected: this check already had the leftover-worktree skip; it was the *other* check (row 58's)
  that lacked it, and that is where the skip was actually added.
- **Row 58** (the one-decision guard's blind spot inside an excused file) -- marked `fixed`. Closing
  note: the exact mutation that proved it was a rival decision added as a new function inside
  `cmd/refusal.go`, one of the three files the old exemption excused wholesale -- verbatim what
  208-18-SUMMARY.md recorded. The exemption is now keyed to five specific, already-reviewed places by
  file and function name, not three whole files, and the missing leftover-worktree skip was added to
  this same check at the same time.
- **Row 53** (the practice-project rehearsal has never finished all fourteen of its steps in a live
  run) -- left exactly as it was. `git diff` on `.planning/WINDOWS.md` shows no change touching that
  row's line, and `grep -c '^| 53 .*| open |' .planning/WINDOWS.md` returns 1.

Both rows were closed using the register's own `gsd-tools windows fixed <id>` command, which set the
status and resolved-at timestamp and recalculated the frontmatter counts automatically (36 open, 0
waived, 22 fixed, 58 total -- internally consistent). Neither row's status or timestamp was
hand-edited; only the description text was amended, via scoped edits, to carry the evidence.

## Issues Encountered

None.

## User Setup Required

None -- no external service configuration required.

## Next Phase Readiness

Phase 208 ("Never a Dead End") is now closed with all nineteen of its plans executed and its own
closing account matching what actually happened across all four gap-closure rounds. WINDOWS.md row 53
-- the practice-project rehearsal has never finished all fourteen of its steps in one live run -- is
the one entry this phase deliberately leaves open, carried forward for Phase 209/210 to pick up, exactly
as the owner ruled (D-06). No new tracking file was created anywhere; `.planning/WINDOWS.md` remains
this project's one defect register.

---
*Phase: 208-never-a-dead-end*
*Completed: 2026-09-24*
