---
phase: 186-baseline-showdown-light
plan: 06
subsystem: testing
tags: [bash, jq, benchmark-harness, gsd, model-parity, results-generation]

# Dependency graph
requires:
  - phase: 186-01
    provides: "bench/lib/hermetic-home.sh, bench/lib/gsd-install.sh — isolated-HOME setup and GSD file-copy install"
  - phase: 186-02
    provides: "bench/tasks/*.md — the four task specs and their Per-lane invocation sections"
  - phase: 186-03
    provides: "bench/lib/measure-tokens.sh, bench/lib/git-cleanliness.sh, bench/lib/operator-log.sh — the three measurement libraries"
  - phase: 186-05
    provides: "bench/acceptance/*.sh, bench/acceptance/verify-predates-runs.sh — deterministic judges and the ordering checker"
provides:
  - "bench/harness/run-lane.sh — lane_prepare, lane_invocation, lane_working_state_paths, lane_force_model"
  - "bench/harness/run-cell.sh — one lane x one category end to end: clone, seed, isolate, run, measure, judge, record"
  - "bench/run.sh — the one documented command; --dry-run enumerates all 12 cells and executes nothing; refuses to run when acceptance ordering would break"
  - "bench/harness/permitted-inputs.md — the enumerated scripted answers for all 12 cells, written before any run"
  - "bench/results/generate-table.sh — jq-based markdown results table from per-run run.json evidence, bash-3.2-compatible"
  - "bench/RUNBOOK.md — the operator's full step-by-step procedure for the twelve runs"
  - "bench/README.md additions — Running the benchmark, What lands in bench/results/"
affects: [186-07, 192-full-comparison]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "run-lane.sh/run-cell.sh split: all lane-specific branching lives in run-lane.sh; run-cell.sh never contains an if-lane-== branch in its measurement or judging code"
    - "lane_invocation reads the task spec's own Per-lane invocation section (awk section extraction) rather than duplicating commands, so the runner cannot drift from the reviewed spec"
    - "MODEL-DEVIATION lines: a lane that cannot be forced to the pinned model by configuration prints a named deviation line and returns 0, never aborting the run"
    - "hallucinated_completion defaults to the literal string unknown when the claimed-success transcript is absent — an absent input is never silently favourable"
    - "bash 3.2 (macOS default) compatibility: generate-table.sh uses plain arrays instead of declare -A, matching the convention already established in bench/lib/git-cleanliness.sh"
    - "fixtures for the results-table generator live under bench/harness/fixtures/, never under bench/results/, so the acceptance-ordering checker's result-directory scan can never mistake a fixture for a real run"

key-files:
  created:
    - bench/harness/run-lane.sh
    - bench/harness/run-cell.sh
    - bench/run.sh
    - bench/harness/permitted-inputs.md
    - bench/results/generate-table.sh
    - bench/harness/fixtures/sample-results/*/run.json (12 fixture files)
    - bench/RUNBOOK.md
  modified:
    - bench/README.md
    - Makefile

key-decisions:
  - "Pinned model for the dry-run/structural exercises in this plan: the fixture value 'fixture' / 'claude-fixture-model' — no real model has been pinned yet, since no live run happens until plan 07; BENCH_MODEL is required at run.sh's real-run path but plan 06 never invokes a real cell"
  - "lane_force_model for the Aether lanes writes a hub-config override file (AETHER_HUB_DIR/benchmark-model-override.json) plus exports ANTHROPIC_MODEL, rather than editing this repo's agent frontmatter source — this was never verified against a live Aether run in this plan (that is plan 07's job); if the hub does not yet exist when lane_force_model runs, it emits a MODEL-DEVIATION line and returns 0 rather than failing the cell"
  - "Cell ordering in the runbook: gsd, then aether-interactive, then aether-autopilot, each lane's four categories run 01 through 04 in the same sequence — chosen so each lane's one-time setup is paid once and no lane benefits from growing operator familiarity with a different task order"
  - "generate-table.sh rewritten to avoid declare -A (bash 3.2 has no associative arrays) after the first fixture run failed with 'declare: -A: invalid option' on this machine's default /bin/bash"

patterns-established:
  - "Sourceable bench/harness/*.sh convention matching bench/lib/*.sh: set -euo pipefail, fail()/step() helpers, functions exported by sourcing"
  - "Permitted-inputs.md section-heading contract: '## <lane> <category>' (space-separated, no vocabulary translation needed since lane/category names are already the harness's own plain identifiers) — run-cell.sh's operator-facing print step matches this exact heading format"

requirements-completed: [PROOF-01, PROOF-03]

# Metrics
duration: 70min
completed: 2026-08-17
---

# Phase 186 Plan 06: The Harness Itself Summary

**One documented command (`bench/run.sh`) now runs any single lane x category cell end to end or, with `--dry-run`, enumerates the full twelve-cell matrix and executes nothing — backed by a lane runner, a cell runner, a written-in-advance permitted-inputs script, a jq-based results-table generator proven against twelve hand-written fixtures, and an operator runbook — with zero real benchmark runs executed.**

## Performance

- **Duration:** 70 min
- **Started:** 2026-08-17T19:39:00Z (approx, after worktree base correction)
- **Completed:** 2026-08-17T20:49:27Z
- **Tasks:** 3
- **Files modified:** 21 (19 created, 2 modified)

## Accomplishments
- Built `bench/harness/run-lane.sh`, confining every lane-specific difference (install mechanism, entry commands, working-state paths, model-forcing) behind four functions so `run-cell.sh` never branches on which lane is running
- Built `bench/harness/run-cell.sh`: fresh clone at a pinned SHA, seeded defect/test application for categories 01/03, isolated HOME + model pin, a real background SIGKILL watcher for category 03 (first file write + 120s), an operator hand-off point, script-only measurement via all three lib scripts, judgement against a *second* fresh clone, and a hallucination flag that defaults to `unknown` rather than `false` when its input is missing
- Built `bench/run.sh`, the one documented entry point — `--dry-run` verified to print exactly 12 cells and execute nothing; a real invocation verified to refuse when `BENCH_MODEL` is unset and to refuse when `bench/acceptance/verify-predates-runs.sh` fails (exercised live with a temporarily swapped-in fake-failing script, then restored, confirmed `git status` clean afterward)
- Wrote `bench/harness/permitted-inputs.md` covering all 12 cells with the governing unscripted-intervention rule stated up front
- Built `bench/results/generate-table.sh` and proved its format against 12 hand-written fixture `run.json` files under `bench/harness/fixtures/sample-results/` — all 11 required columns, the per-lane summary block, a `## Recorded deviations` section (exercised with a model deviation, a `parallel_mode_note`, and an `unknown` hallucination flag all present in the fixtures), the `--partial` incomplete-banner path, and the footer-reads-from-run.json property, all verified live
- Wrote `bench/RUNBOOK.md` (all 8 required sections, all 12 cells listed, every referenced `bench/...` path confirmed to exist) and appended two additive-only sections to `bench/README.md`

## Task Commits

Each task was committed atomically:

1. **Task 1: Write the lane runner and the cell runner** - `9002e2d8` (feat)
2. **Task 2: Write the entry point, the permitted-inputs script and the results-table generator** - `029fe5b5` (feat)
3. **Task 3: Write the operator runbook and complete the harness README** - `aec36204` (docs)

_No plan-metadata commit — SUMMARY.md is committed separately per worktree executor protocol; STATE.md/ROADMAP.md are owned by the orchestrator._

## Files Created/Modified
- `bench/harness/run-lane.sh` - `lane_prepare`, `lane_invocation`, `lane_working_state_paths`, `lane_force_model`
- `bench/harness/run-cell.sh` - the 11-stage single-cell runner
- `bench/run.sh` - the one documented command, dry-run mode, tool preflight, ordering-checker refusal
- `bench/harness/permitted-inputs.md` - all 12 cells' permitted operator inputs
- `bench/results/generate-table.sh` - markdown table generator from `run.json` evidence
- `bench/harness/fixtures/sample-results/*/run.json` - 12 fixture files proving the generator's format
- `bench/RUNBOOK.md` - the operator's twelve-run procedure
- `bench/README.md` - two additive sections (Running the benchmark, What lands in bench/results/)
- `Makefile` - `bench-dry-run` target

## Decisions Made
- See `key-decisions` in the frontmatter above for the four load-bearing decisions (pinned-model status, model-forcing mechanism and its untested status, cell ordering rationale, bash-3.2 compatibility fix)
- `lane_force_model`'s Aether-lane mechanism (hub-config override file + `ANTHROPIC_MODEL`) has not been exercised against a real Aether install in this plan — this plan is explicitly structural/dry-run-shaped per its own scope; plan 07 is where this gets a live test and where the pinned model is actually chosen and recorded

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `generate-table.sh` used `declare -A` (associative arrays), which macOS's default `/bin/bash` (3.2) does not support**
- **Found during:** Task 2, first live run of the generator against the fixture directory
- **Issue:** `declare -A FOUND_CELLS=()` failed with `declare: -A: invalid option` — bash 3.2 has no associative arrays, the same constraint `bench/lib/git-cleanliness.sh` already documents and works around
- **Fix:** Replaced the associative-array cell-presence map with two plain arrays (`RUN_JSON_FILES`, `MISSING_CELLS`) built in a single pass
- **Files modified:** `bench/results/generate-table.sh`
- **Verification:** Re-ran against the 12-file fixture directory; full table, per-lane summary, and deviations section all rendered correctly; re-ran the `--partial` and no-`--partial`-refusal paths, both correct
- **Committed in:** `029fe5b5` (Task 2 commit)

**2. [Rule 1 - Bug] `run-cell.sh`'s permitted-inputs section-heading match did not match the actual heading format written in `permitted-inputs.md`**
- **Found during:** Task 2/3 boundary, while testing the awk extraction used by both `run-lane.sh`'s `lane_invocation` (already correct) and `run-cell.sh`'s permitted-inputs print step
- **Issue:** The original pattern built `"^## " lane " . " category "$"`, which in awk regex concatenation requires a literal space, then any one character, then another literal space before the category — but the real heading is `## gsd 01-bug-fix` (a single space), so it never matched
- **Fix:** Replaced the constructed-regex match with a literal string comparison (`$0 == "## $LANE $CATEGORY"`), verified against all 12 real headings in the committed `permitted-inputs.md`
- **Files modified:** `bench/harness/run-cell.sh`
- **Verification:** Extracted section for `gsd`/`01-bug-fix` prints the correct three bullet points, nothing more, nothing less
- **Committed in:** `029fe5b5` (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (both Rule 1 bugs caught by actually running the scripts, not just syntax-checking them)
**Impact on plan:** Both fixes were necessary for the acceptance criteria's live-execution checks to be satisfiable at all. No scope creep — both fixes stayed inside files the plan already scoped.

## Issues Encountered

None beyond the two auto-fixed bugs above. The acceptance-ordering refusal path (`bench/run.sh` refusing when `verify-predates-runs.sh` fails) was exercised for real by temporarily replacing the real checker script with a fake-failing one, confirming the refusal, then restoring the original from a backup and confirming `git status` showed no diff on the real file — the plan's instruction to prove this "for real" in a way that leaves no trace was followed literally.

## User Setup Required

None - no external service configuration required. This plan builds structure only; no real model was pinned and no live cell was run (that is plan 07's job, with a human present).

## Next Phase Readiness

**Ready for plan 07 (the twelve real runs):** `bench/run.sh --lane <lane> --category <category>` is the single command plan 07 will invoke twelve times. `bench/RUNBOOK.md` is the procedure plan 07's human operator follows. `bench/results/generate-table.sh` is ready to consume plan 07's real `run.json` files the moment they exist.

**Open items plan 07 must resolve, not this plan:**
- The actual pinned model (`BENCH_MODEL`) — this plan used `fixture`/`claude-fixture-model` throughout its own dry-run and fixture exercises; plan 07 must choose and record the real value.
- Whether `lane_force_model`'s hub-config-override mechanism actually forces the Aether lanes to the pinned model, or whether a `MODEL-DEVIATION` line gets recorded instead — this has not been exercised against a live, installed Aether hub. The mechanism is implemented and syntactically sound but unverified end-to-end.
- The claimed-success transcript capture convention (`$WORK/claimed-success.txt`) is documented in the runbook but has never been exercised with a real terminal session; plan 07 is where this either works as documented or surfaces a needed correction.

## Self-Check: PASSED

All created files confirmed present on disk: `bench/harness/run-lane.sh`, `bench/harness/run-cell.sh`, `bench/run.sh`, `bench/harness/permitted-inputs.md`, `bench/results/generate-table.sh`, the 12 fixture `run.json` files under `bench/harness/fixtures/sample-results/`, `bench/RUNBOOK.md`, this SUMMARY.md. All three task commits (`9002e2d8`, `029fe5b5`, `aec36204`) confirmed present in `git log --oneline`.

---
*Phase: 186-baseline-showdown-light*
*Completed: 2026-08-17*
