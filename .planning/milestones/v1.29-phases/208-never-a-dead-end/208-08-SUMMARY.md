---
phase: 208-never-a-dead-end
plan: 08
subsystem: cli
tags: [go, journey, refusal, eval-gate, windows-ledger]

# Dependency graph
requires:
  - phase: 208-01
    provides: "The typed refusal contract (refuse(), refusalRegistry, renderRefusal) whose generated_at recovery and NextCommand naming this run's own transcript exercised live."
  - phase: 208-05
    provides: "The directory-artifact refusal and owner-confirmation routing this plan's WINDOWS row 49/52 resolution cites."
  - phase: 208-06
    provides: "The colonize-existing-survey-found refusal row (Disposition: stop, ProtectsWork: true) this run's live transcript hit."
  - phase: 208-07
    provides: "journeyRunPrintedNextCommands -- the printed-refusal-execution machinery this run's transcript would have exercised had it reached colonize-finalize."
provides:
  - "One real, owner-approved single-trial walk of the messy practice project journey ($0.93, 90s), measured entirely from its own on-disk transcript and cost-state records"
  - "A provisional 6662s journey gate budget (Makefile + gates.json) sized from the full fourteen-step per-step cap table, honestly labelled provisional pending a completed-chain run"
  - "WINDOWS.md rows 49 and 52 closed with the commits that fixed them; row 53 left honestly open (its fix shipped but was not exercised live); a new row 55 opened for a distinct, newly-observed finding (a ProtectsWork stop refusal correctly asking the owner rather than self-executing, which an automated -p chain cannot answer); row 56 catalogues the full ./cmd suite's 30 pre-existing failures against the known-red baseline"
affects: [209]

# Actuals (#2632)
actuals:
  tokens: 11800
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Provisional-then-final gate sizing, explicitly labelled: when a live run cannot honestly produce the 'final' figure a plan calls for (here, because the trial stopped before the expensive build/check steps ever ran), the provisional value is kept and the gap is documented rather than a number being invented to satisfy the letter of the task."

key-files:
  created:
    - .planning/phases/208-never-a-dead-end/208-JOURNEY-RUN.md
  modified:
    - Makefile
    - cmd/testdata/eval-gates/gates.json
    - .planning/WINDOWS.md
    - .planning/ROADMAP.md

key-decisions:
  - "Task 1's checkpoint was pre-answered by the owner (one-then-decide, one walk approved, ~£12-25/20-40min) before this executor was spawned; executed without re-asking."
  - "No 'final' re-measurement of the journey gate budget was made. The plan's own action text calls for one derived from 'the slowest trial's own wall clock' -- but this run's one trial stopped at step 2 of 14, so no real wall-clock figure exists for the expensive build/check steps (up to 1200s cap each). Inventing one would be exactly the projection this plan and CLAUDE.md's Definition of Done forbid. The provisional 6662s value (sum of all fourteen steps' own wall-clock caps + 207-06's measured per-trial headroom) stays in force, labelled provisional in both files, until a completed-chain run can measure a real figure."
  - "The run's failure at 'survey' is NOT attributed to a bug this phase's commits introduced. Git blame confirms the colonize-existing-survey-found refusal's underlying text predates Phase 208 (208-06 only wrapped an already-existing bare error in the typed-refusal envelope). The refusal fired exactly as designed (ProtectsWork: true, names the one way past); the dead end is that the driving assistant chose to ask the owner for confirmation rather than run the named --force-resurvey itself, and the automated single-shot-per-step journey harness has no owner to answer. No second paid run was started on the plan's own initiative, per the checkpoint's own instruction, since the cause is not clearly 'introduced by this phase'."
  - "WINDOWS row 53 stays open rather than being marked fixed on the strength of the (real, tested) generated_at-recovery code alone -- the row's own proposed close explicitly names a live behaviour (the survey step passing), and this run's transcript never reached the code path that fix touches (colonize-finalize was never called)."
  - "A new WINDOWS row (55) was opened for the distinct ask-vs-act finding rather than folding it into row 53, since it has a different proximate cause and needs a different owner-facing decision (whether an automated chat should auto-run a ProtectsWork stop's NextCommand)."
  - "Row and reason edits to WINDOWS.md were made by loading the ledger through the same parseLedger/renderLedger functions the CLI tool uses (rather than hand-editing the rendered markdown table), after discovering that the table is regenerated wholesale from the hidden JSON block on every `windows append`/`fixed` call -- a direct markdown-table edit is silently overwritten by the next tool invocation."
  - "The full, unscoped `go test ./cmd -count=1 -timeout 90m` (required by this plan's own <verification>) was run once in the background; its 30 pre-existing failures are cross-referenced against WINDOWS row 12's tracked list (14 match, 6 of row 12's own list now pass) and the separately-recorded 2026-09-14 baseline (1 more matches); the remaining 15 are newly catalogued as row 56 rather than fixed or silently ignored -- none touch this plan's declared files, so fixing them is out of this plan's scope per the deviation rules' scope boundary."

requirements-completed: [UED-10, UED-11, UED-12, UED-13, UED-14, UED-15]

coverage:
  - id: D1
    description: "One real, owner-approved journey walk driven and measured entirely from its own on-disk artefacts (transcript cost-state records, go test subtest timing, process timestamps) -- no figure projected or carried over from Phase 207."
    requirement: "UED-10"
    verification:
      - kind: manual_procedural
        ref: "make eval-gate-journey (AETHER_JOURNEY_TRIALS=1), 2026-09-23T08:53:35Z-08:55:05Z"
        status: pass
    human_judgment: false
  - id: D2
    description: "The journey gate's Makefile timeout and gates.json budget_seconds are sized together (both 6662s) from a documented formula and stay in agreement, proven by the paired test passing before and after both edits."
    requirement: "UED-10"
    verification:
      - kind: unit
        ref: "cmd/eval_gates_test.go#TestEvalGateVocabularyMatchesTheManifest,TestEvalGateBudgetsArePositive"
        status: pass
    human_judgment: false
  - id: D3
    description: "The survey step's on-disk fact check (territory snapshot source_revision matches HEAD) did NOT pass in this run -- reported honestly rather than claimed proven, with the precise cause traced from the real transcript."
    requirement: "UED-10"
    verification: []
    human_judgment: true
    rationale: "This is a negative finding about live product behaviour (an LLM's own judgement call inside a real chat), not something a unit test asserts -- the owner needs to read the reasoning and decide the underlying ask-vs-act product question (WINDOWS row 55), which is out of this plan's own scope to resolve."
  - id: D4
    description: "WINDOWS.md rows 49-53 resolved from evidence: 49 and 52 marked fixed with the commits that fixed them, 50/51 already fixed, 53 left open with an honest reason; a new row 55 opened for the distinct finding this run surfaced, and row 56 catalogues the full-suite result."
    requirement: "UED-11"
    verification:
      - kind: manual_procedural
        ref: "gsd-tools windows status, .planning/WINDOWS.md"
        status: pass
    human_judgment: false
---

# Phase 208 Plan 08: The Real Run, Measured Honestly Summary

**One owner-approved real walk of the practice-project journey ($0.93, 90 seconds) stopped at
"survey" again -- not the bug this phase fixed, but a new, distinct cause the run itself
uncovered and documented rather than papered over.**

## Performance

- **Duration:** ~50 min (including a ~28-minute full `go test ./cmd` background run required by
  this plan's own `<verification>`)
- **Started:** 2026-09-23T08:47:00Z (approx, from context load)
- **Completed:** 2026-09-23T09:40:00Z (approx)
- **Tasks:** 3 (Task 1 pre-answered by the owner; Tasks 2 and 3 executed)
- **Files modified:** 5 (`Makefile`, `cmd/testdata/eval-gates/gates.json`, new
  `208-JOURNEY-RUN.md`, `.planning/WINDOWS.md`, `.planning/ROADMAP.md`)

## Accomplishments

- Sized the journey gate provisionally (6662s) for a full fourteen-step chain, from the
  documented per-step cap table plus the measured per-trial headroom -- committed before the
  real run started, as the plan requires.
- Ran the one owner-approved single-trial journey walk for real, in the background, polled to
  completion (never as a foreground call).
- Wrote `208-JOURNEY-RUN.md` in the Phase 207 report's shape, with every figure traced to a
  named log, transcript, or timestamp -- including a plain-English paragraph up front for the
  owner.
- Resolved WINDOWS.md rows 49 and 52 with the commits that fixed them; left row 53 honestly open
  since its own proposed close (the survey step passing) did not happen this run; opened a new
  row 55 for the distinct finding this run surfaced, and row 56 for the full-suite cross-check.
- Ran and recorded the full `go test ./cmd -count=1 -timeout 90m` this plan's own `<verification>`
  requires: discovered=5997, executed=5997 (not truncated), 30 pre-existing failures, all
  cross-referenced against the known-red baseline, none touching this plan's own files.

## Task Commits

Each task was committed atomically:

1. **Task 2 (sizing half): Size the harness for a full walk** - `d8f38b40` (fix)
2. **Task 3: Write the run report and close the defect register rows** - `aaab3dfa` (docs)
3. **Task 3 (verification record): record the full ./cmd suite result** - `d7edeab8` (docs)

**Plan metadata:** committed alongside this SUMMARY.

_Note: Task 2's "run the journey for real" and "measure what happened" sub-steps produced no
separate commit of their own -- they are observation/measurement, captured in the `aaab3dfa`
commit's `208-JOURNEY-RUN.md`._

## Files Created/Modified

- `.planning/phases/208-never-a-dead-end/208-JOURNEY-RUN.md` - the factual run report
- `Makefile` - `eval-gate-journey` timeout, 1500s -> 6662s (provisional), with the sizing
  rationale in a comment
- `cmd/testdata/eval-gates/gates.json` - `journey.budget_seconds`, 1100 -> 6662 (provisional),
  `purpose` text rewritten to explain why
- `.planning/WINDOWS.md` - rows 49, 52 fixed with commit-citing reasons; row 53's reason
  rewritten to state honestly what is and is not proven; rows 55 and 56 added
- `.planning/ROADMAP.md` - Phase 208's plan list ticked 8/8, one plain-English result sentence
  added (scoped edit, only that section touched)

## Decisions Made

See `key-decisions` in the frontmatter above -- six decisions, the most consequential being: no
fabricated "final" gate-sizing figure, no attribution of the survey-step failure to this phase's
own code without evidence, and WINDOWS row 53 staying open rather than being marked fixed on
unit-test strength alone.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking issue] WINDOWS.md hand-edits to the reason column were silently
overwritten by the next `windows append`/`fixed` tool call**
- **Found during:** Task 3
- **Issue:** `gsd-tools windows fixed <id>` and `windows append` both regenerate the entire
  rendered markdown table from a hidden JSON block on every write; a direct edit to the visible
  table cells (adding commit-hash reasons to rows 49/52/53) was silently lost the next time any
  `windows` subcommand ran.
- **Fix:** Wrote a short script using the same exported `parseLedger`/`renderLedger` functions
  the CLI itself uses, to load the ledger, set the `reason` field on the target entries, and
  write it back atomically -- surviving any later tool invocation.
- **Files modified:** `.planning/WINDOWS.md`
- **Verification:** `gsd-tools windows status` and a fresh `windows append` call afterward both
  show the hand-set reasons intact.
- **Committed in:** `aaab3dfa`

---

**Total deviations:** 1 auto-fixed (Rule 3 -- blocking issue in the ledger-editing mechanics).
**Impact on plan:** None on scope; a tooling workaround only, no runtime or product code touched.

## Issues Encountered

**The must-have truth "the survey step passes" was not met by this run.** This is not a bug this
plan introduced or failed to fix -- the code fixes for the specific defect Phase 207 found
(missing `generated_at`) shipped correctly in 208-01/208-07 and are unit-tested, but this run's
one trial hit a *different*, earlier, pre-existing refusal (`colonize-existing-survey-found`)
where the driving assistant asked the owner for confirmation rather than running the named
`--force-resurvey` itself -- and the automated journey harness sends one prompt per step with no
follow-up turn to answer that question. This is reported honestly in `208-JOURNEY-RUN.md` and
filed as WINDOWS row 55, a distinct product/UX decision (should an automated chat auto-run a
`ProtectsWork` stop's named next command, or always defer?) that is out of this plan's own
declared-file scope to resolve. No second paid run was started; per the checkpoint's own
instruction, that needs the owner's agreement first, and the cause is not one this plan's own
commits introduced.

**The full `go test ./cmd` run surfaced 15 test failures not previously catalogued anywhere.**
None touch this plan's own files; recorded as WINDOWS row 56 for future triage rather than
bisected or fixed here (out of scope, per the deviation rules' scope boundary).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Phase 208's other six plans (208-01 through 208-07) all landed and are unit-tested; this plan's
own real-world proof is partial and honestly reported: the refusal system itself works as
designed (proven live, for the refusal that did fire), but the phase's own "never a dead end"
mission is not yet proven end-to-end against the practice-project journey. Recommend the owner
weigh in on WINDOWS row 55 (auto-run vs. always-ask for a `ProtectsWork` stop in an automated
chain) before the next `make eval-gate-journey` three-trial attempt, and decide separately
whether/when to spend on a second, full-length walk once that product question is settled.

---
*Phase: 208-never-a-dead-end*
*Completed: 2026-09-23*

## Self-Check: PASSED

- `.planning/phases/208-never-a-dead-end/208-JOURNEY-RUN.md` — FOUND
- `git log --oneline --all | grep -q d8f38b40` — FOUND
- `git log --oneline --all | grep -q aaab3dfa` — FOUND
- `git log --oneline --all | grep -q d7edeab8` — FOUND
- `go build ./...` — PASSED (no output)
- `go vet ./cmd/` — PASSED (no output)
- `go test ./cmd -run 'TestEvalGate|TestJourneyGateVerdict|TestEvalGateVocabularyMatchesTheManifest' -count=1 -timeout 8m` — PASSED (ok, 51.592s)
- `gsd-tools windows status` — open=38 fixed=18 total=56, ledger parses cleanly
