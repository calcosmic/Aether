---
phase: 186-baseline-showdown-light
plan: 05
subsystem: benchmark-harness
tags: [bench, acceptance, deterministic-verdict, git-history-proof, ordering-baseline]

# Dependency graph
requires:
  - phase: 186-baseline-showdown-light
    provides: "bench/tasks/*.md, bench/tasks/allowlists/*.txt (plan 02) — the Done means sections these scripts implement"
  - phase: 186-baseline-showdown-light
    provides: "bench/lib/git-cleanliness.sh (plan 03) — the allowlist parsing convention these scripts' companion checker shares"
provides:
  - "bench/acceptance/01-bug-fix.sh, 02-brownfield-feature.sh, 03-interrupted-execution.sh, 04-fresh-repo-lifecycle.sh — one deterministic exit-code judge per task category"
  - "bench/acceptance/verify-predates-runs.sh — proves from git history that acceptance predates any run"
  - "bench/acceptance/README.md — plain-English explainer plus the recorded ordering baseline commit"
affects: [186-06, 186-07, 192-full-comparison]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Uniform judge contract: one positional arg (path to a fresh clone), a check/record helper that runs every assertion without exiting early, FAIL <name>: <expected> / <found> per failure, PASS <task id> on full pass, JSON detail via ACCEPTANCE_JSON or fd 3"
    - "JSON assembly via jq -n --arg (not sed-escaped string concatenation) — sed cannot safely escape raw newlines/tabs from captured go test / npm test / jest output inside a hand-built JSON string"
    - "Repository detection via 'git rev-parse --is-inside-work-tree' rather than '[ -d .git ]' — a git worktree's .git is a file, not a directory, and the bare directory check silently rejected legitimate worktrees"
    - "Vocabulary rule stated by category, not by literal-quoting the banned tokens, in every script's own header — quoting the banned words to name them would itself match the phase's own literal grep verification, so the rule points to bench/acceptance/README.md's Vocabulary rule section for the full list instead"
    - "git-history-only ordering proof: introducing commit via 'git log --diff-filter=A --follow --format=%H|%ct -- FILE | tail -1', most-recent-modification via 'git log --format=%ct -- FILE | head -1', both checked against the earliest bench/results/*/ directory's own introducing commit; vacuous pass names the exact count checked so it cannot be mistaken for a full check"

key-files:
  created:
    - bench/acceptance/01-bug-fix.sh
    - bench/acceptance/02-brownfield-feature.sh
    - bench/acceptance/03-interrupted-execution.sh
    - bench/acceptance/04-fresh-repo-lifecycle.sh
    - bench/acceptance/verify-predates-runs.sh
    - bench/acceptance/README.md
  modified:
    - Makefile

key-decisions:
  - "02-brownfield-feature.sh's reachability probe runs as a throwaway jest test file (src/__tests__/acceptance-probe-lowercase.test.ts, written then deleted) rather than a standalone ts-node script — a live test run showed npx ts-node's bin symlink resolution is broken on this machine's fresh npm install (Node v26.5.0), while jest (already proven working by the existing-suite assertion in the same script) runs the identical probe correctly"
  - "Repository-check switched from '[ -d RESULT_DIR/.git ]' to 'git rev-parse --is-inside-work-tree' across all four scripts after live-testing 04-fresh-repo-lifecycle.sh's AETHER_REPO check against this very worktree and discovering the bare directory test rejects worktrees (whose .git is a text pointer file, not a directory) — a real bug that would have silently failed every Aether-lane worktree-mode run in Phase 187+"
  - "JSON assertion records are built with 'jq -n --arg' instead of sed-based escaping, after ACCEPTANCE_JSON output from a real go test failure was fed to 'jq -e .' and failed to parse — captured test output contains raw newlines/tabs that sed's s/\\\\/\\\\\\\\/g; s/\"/\\\\\"/g pattern cannot safely encode into a JSON string"
  - "Ordering baseline commit (763572ba) was pinned to the last commit that touched bench/acceptance/ before this SUMMARY, not the first — Task 3's own instructions treat either the introducing or a documentation-amending later commit as acceptable so long as it precedes any results directory, and pinning the most recent one keeps the recorded SHA and the git-log evidence in the README trivially self-consistent for a reader checking it by hand"

requirements-completed: [PROOF-01]

# Metrics
duration: 95min
completed: 2026-08-17
---

# Phase 186 Plan 05: Deterministic Acceptance Scripts Summary

**Four exit-code judges (one per benchmark task) plus a git-history checker that proves they were all written and last touched before any run — live-verified against real fresh clones of gorilla/mux, zod, and a hand-built temperature-converter CLI, each failing on the unsolved substrate and passing on a constructed reference fix.**

## Performance

- **Duration:** 95 min (includes live substrate cloning, a real bug found and fixed in the JSON-emission code, and a real worktree-detection bug found and fixed in the repository-check)
- **Started:** 2026-08-17 (worktree reset to base commit 6928e754, then context reading)
- **Completed:** 2026-08-17
- **Tasks:** 3
- **Files modified:** 7 (6 created, 1 modified)

## Accomplishments

- Wrote all four acceptance scripts to the plan's uniform contract: one positional argument, a `record` helper that runs every assertion without stopping at the first failure, `FAIL <name>: <expected> / <found>` per failure line, `PASS <task id>` on full success, and a JSON detail object via `ACCEPTANCE_JSON` or file descriptor 3
- Live-cloned both pinned substrates (`gorilla/mux` at its pinned SHA, `colinhacks/zod` at v3.23.8) into the scratchpad, applied the seeded bug/reused it as-is for the fresh clone, and confirmed `01-bug-fix.sh` and `03-interrupted-execution.sh` both fail with more than one `FAIL` line on the unsolved clone and pass with `PASS` on a hand-reverted reference fix
- Hand-implemented the `.lowercase()` string-validation feature in a copy of the pinned zod clone (following the library's own `startsWith`/`endsWith` pattern across `src/types.ts`, `src/ZodError.ts`, and `src/locales/en.ts`), confirmed the existing 488-test suite still passed, and confirmed `02-brownfield-feature.sh` fails on the unimplemented clone and passes on the implementation
- Hand-built a working Go temperature-converter CLI with tests satisfying Task 04's `## Done means`, confirmed both fixed points (0C=32F, 100C=212F) convert correctly in both directions by actually running the program, and confirmed `04-fresh-repo-lifecycle.sh` fails on an empty repository and passes on the reference program — including live-testing the `AETHER_REPO` clean/dirty branch both ways
- Built `verify-predates-runs.sh`, which reads `git log --diff-filter=A` and `git log --format=%ct` directly rather than trusting any script's self-report, live-tested its untracked-file refusal (twice — once naming itself before it was committed, once naming a deliberately-created `zz-temp.sh` after), and confirmed its vacuous-pass message names the exact count of scripts checked
- Recorded the ordering baseline: `bench/results/` has zero commits in this repository's history at the time `bench/acceptance/` first landed, and the fixed-point commit SHA is written into `bench/acceptance/README.md` alongside the evidence a reader can independently re-check

## Task Commits

Each task was committed atomically:

1. **Task 1: Write the four acceptance scripts** - `afd27e7c` (feat)
2. **Task 2: Write the acceptance-predates-runs checker** - `763572ba` (feat)
3. **Task 3: Commit the acceptance scripts and record the ordering baseline** - `1b780ea8` (docs, README amendment; the underlying commit-before-any-results property was already satisfied by Tasks 1-2's commits, since `bench/results/` did not yet exist when they landed)

_No plan-metadata commit — SUMMARY.md is committed separately per worktree executor protocol; STATE.md/ROADMAP.md are owned by the orchestrator._

## Files Created/Modified

- `bench/acceptance/01-bug-fix.sh` - judges the seeded `gorilla/mux` `cleanPath` bug fix: seeded single-test pass, seeded test file byte-unchanged, seeded test name intact, full suite pass, and an independent probe (different input strings) that catches a fix special-cased to the seeded test's exact string
- `bench/acceptance/02-brownfield-feature.sh` - judges the `.lowercase()` zod feature: existing suite still passes, the feature's actual behaviour (rejects uppercase, accepts lowercase and empty string) exercised via a throwaway jest probe, and the builder method's presence on the public string-schema surface in `src/types.ts`
- `bench/acceptance/03-interrupted-execution.sh` - judges the same seeded bug fix as Task 01 plus interruption-specific assertions: the seeded defective line is gone (pre-kill work survived) and the corrected condition appears exactly once (no duplicate/conflicting partial work from a bad resume)
- `bench/acceptance/04-fresh-repo-lifecycle.sh` - judges the fresh-repo lifecycle: at least one commit, clean working tree, the project's own documented test command passes, the program itself converts both fixed points in both directions, and — the category's own extra requirement — `AETHER_REPO`'s `git status --porcelain` is empty
- `bench/acceptance/verify-predates-runs.sh` - proves from git history that every acceptance script's introducing AND most-recent-modification commit predate the earliest `bench/results/*/` directory's introducing commit; exits 0 vacuously (and says so by name) when no results exist
- `bench/acceptance/README.md` - plain-English explainer of what the judges are and why they predate any run, the one command (`bench/acceptance/verify-predates-runs.sh`) that proves the ordering, the vocabulary rule and its one-sentence reason, and the recorded Ordering baseline section
- `Makefile` - added `bench-acceptance-order` target invoking the ordering checker

## Decisions Made

- Chose jest over ts-node for the zod reachability probe after ts-node's bin resolution genuinely failed on this machine's fresh install — this is documented as a live-discovered bug fix (Rule 1), not a design preference stated up front
- Chose `git rev-parse --is-inside-work-tree` over a bare `.git` directory check after the exact same bug surfaced while live-testing `04-fresh-repo-lifecycle.sh`'s `AETHER_REPO` assertion against this very worktree
- Chose `jq -n --arg` JSON assembly over sed-escaped string concatenation after `ACCEPTANCE_JSON` output from a real failing `go test` run produced invalid JSON (raw control characters that sed cannot safely encode)
- Stated the vocabulary rule in each script's header by category rather than by quoting the literal banned words, so the header itself does not trip the phase's own literal grep verification command while still satisfying "state the rule in each script's header" — the full literal list lives once, in `bench/acceptance/README.md`

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] ACCEPTANCE_JSON output was invalid JSON when test output contained control characters**
- **Found during:** Task 1, first live run with `ACCEPTANCE_JSON` set against a real `go test` failure
- **Issue:** `build_json`'s sed-based escaping (`s/\\/\\\\/g; s/"/\\"/g`) only escapes backslashes and quotes; raw newlines and tabs embedded in captured `go test`/`npm test` output broke `jq -e .` parsing (`Invalid string: control characters from U+0000 through U+001F must be escaped`)
- **Fix:** Rewrote `build_json` in all four scripts to build each assertion record via `jq -n --arg`, which escapes control characters correctly, then accumulate records with `jq --argjson '. + [$r]'`
- **Files modified:** all four `bench/acceptance/*.sh` task scripts
- **Verification:** re-ran `ACCEPTANCE_JSON=... 01-bug-fix.sh` against both the unsolved and fixed clones; `jq -e .` parsed both outputs cleanly afterward
- **Committed in:** `afd27e7c` (Task 1 commit)

**2. [Rule 1 - Bug] Repository-check rejected git worktrees**
- **Found during:** Task 1, live-testing `04-fresh-repo-lifecycle.sh`'s `AETHER_REPO` assertion against this executor's own worktree
- **Issue:** `[ -d "$PATH/.git" ]` returned false for the worktree, whose `.git` is a text-pointer file (`gitdir: .../.git/worktrees/<name>`), not a directory — every script's repository-check would have silently rejected a legitimate worktree-mode run's resulting clone
- **Fix:** Replaced with `(cd "$PATH" && git rev-parse --is-inside-work-tree)`, which correctly recognizes both plain clones and worktrees, in all four task scripts
- **Files modified:** all four `bench/acceptance/*.sh` task scripts
- **Verification:** re-ran the no-argument and non-git-path tests on all four scripts (still correctly refuse), then confirmed `04-fresh-repo-lifecycle.sh` correctly evaluates this worktree's own dirty/clean `git status --porcelain` state via `AETHER_REPO`
- **Committed in:** `afd27e7c` (Task 1 commit)

**3. [Rule 1 - Bug] ts-node's bin symlink resolution failed on this machine's fresh zod npm install**
- **Found during:** Task 1, first live run of the `.lowercase()` reachability probe against the reference-fixed zod clone
- **Issue:** `npx --yes ts-node --compiler-options ... probe.ts` failed with `Cannot find module './util'` — the `node_modules/.bin/ts-node` symlink resolved incorrectly under Node v26.5.0 in a fresh install, unrelated to the feature under test (confirmed by running the identical logic directly via `jest`, which succeeded)
- **Fix:** Rewrote the probe to run as a throwaway `src/__tests__/acceptance-probe-lowercase.test.ts` executed via `node_modules/.bin/jest` (the same test runner the script's own existing-suite assertion already proves works), written then deleted regardless of pass/fail
- **Files modified:** `bench/acceptance/02-brownfield-feature.sh`
- **Verification:** re-ran against both the unsolved clone (fails, no leftover probe file) and the reference-fixed clone (passes, `PASS 02-brownfield-feature`)
- **Committed in:** `afd27e7c` (Task 1 commit)

---

**Total deviations:** 3 auto-fixed bugs, all found via live execution against real substrate clones rather than static inspection — consistent with this plan's own instruction to test scripts for real where cheap. No scope creep: every fix stayed inside the acceptance scripts the plan already scoped for Task 1.

## Issues Encountered

None blocking. All three deviations above were caught during the plan's own required live-verification step (fail-on-unsolved, pass-on-reference-fix) and fixed before commit.

## What Was Exercised vs. Reasoned

**Exercised (ran for real):**
- `01-bug-fix.sh` and `03-interrupted-execution.sh` against a live clone of `gorilla/mux` at its pinned SHA, both with the seeded defect present (fails, 3-4 `FAIL` lines each) and with a hand-reverted reference fix (passes)
- `02-brownfield-feature.sh` against a live clone of `zod` at its pinned SHA (`npm install` run for real, ~40s), both unimplemented (fails, 2 `FAIL` lines) and with a hand-implemented `.lowercase()` feature across all three required files, including running the existing 488-test suite both before and after
- `04-fresh-repo-lifecycle.sh` against a genuinely empty git repository (fails, 2 `FAIL` lines) and a hand-built, hand-tested Go CLI satisfying the task's `## Done means` (passes) — including live-testing the `AETHER_REPO` branch both with a dirty repo (fails, names the exact modified file) and a clean one (passes), and with the variable unset (fails with the named message)
- `verify-predates-runs.sh`'s untracked-script refusal, twice: once naming itself before Task 2's commit landed, once naming a deliberately-created `bench/acceptance/zz-temp.sh` after Tasks 1-2 were committed
- `verify-predates-runs.sh`'s vacuous-pass path, confirmed to report the correct script count (5) both before and after Task 3's README-only follow-up commit
- `make bench-acceptance-order` end-to-end

**Reasoned (not separately re-run beyond the above):** the exact wording of every `FAIL` message was read for plain-English clarity but not exhaustively screenshotted; the JSON schema's stability across a much larger real benchmark transcript (thousands of lines of `go test -v` output) was not stress-tested beyond the failures already captured live.

## Next Phase Readiness

**Ready:** all four acceptance scripts and the ordering checker are proven against real substrate clones in both directions (fail on unsolved, pass on solved) and are committed strictly before any `bench/results/` directory exists. Plan 06 (the harness) and Plan 07 (the twelve runs) can call these scripts directly with the path to each run's resulting clone as `$1`.

**Not addressed here (by design, per plan scope):** no live run of any of the three lanes (GSD, Aether interactive, Aether autopilot) against any task has happened yet — that remains blocked on the `ANTHROPIC_API_KEY` provisioning step recorded in 186-01's summary, and is Plan 06/07's job, not this plan's.

## Self-Check: PASSED

All seven created/modified files confirmed present on disk (`bench/acceptance/01-bug-fix.sh`, `02-brownfield-feature.sh`, `03-interrupted-execution.sh`, `04-fresh-repo-lifecycle.sh`, `verify-predates-runs.sh`, `README.md`, and `Makefile`). All three task commits (`afd27e7c`, `763572ba`, `1b780ea8`) confirmed present in `git log --oneline`. `git diff --stat 6928e754 HEAD` shows exactly the seven files this plan's `files_modified` frontmatter names, no more and no fewer. `bench/acceptance/verify-predates-runs.sh` re-run as the final action before writing this summary, still exits 0 with the vacuous-pass message naming 5 scripts checked.

---
*Phase: 186-baseline-showdown-light*
*Completed: 2026-08-17*
