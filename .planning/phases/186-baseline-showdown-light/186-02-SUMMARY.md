---
phase: 186-baseline-showdown-light
plan: 02
subsystem: testing
tags: [benchmark-harness, task-specs, substrate-selection, gsd, allowlists]

# Dependency graph
requires:
  - "bench/lib/hermetic-home.sh, bench/lib/gsd-install.sh (186-01) — later runner plans source these"
provides:
  - "bench/tasks/substrate.md — two pinned, verified-neutral substrate repos with clone/test commands"
  - "bench/tasks/01-bug-fix.md, 02-brownfield-feature.md, 03-interrupted-execution.md, 04-fresh-repo-lifecycle.md — the four task specs, identical prompts across lanes"
  - "bench/tasks/allowlists/*.txt — one file-allowlist per task for the unnecessary-modification metric"
affects: [186-03, 186-04, 186-05, 186-06, 186-07, 192-full-comparison]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Task spec six-section structure: Substrate / The task / Setup / Done means / Allowlist / Per-lane invocation"
    - "Allowlist two-block convention: task-content patterns, then a block marked '# lane working state — not counted as unnecessary'"
    - "Seeded-bug verification is a live gate, not an assumption: the seeded test was run failing (before fix) and passing (after fix) in a throwaway copy of the substrate before being written into the task spec"

key-files:
  created:
    - bench/tasks/substrate.md
    - bench/tasks/01-bug-fix.md
    - bench/tasks/02-brownfield-feature.md
    - bench/tasks/03-interrupted-execution.md
    - bench/tasks/04-fresh-repo-lifecycle.md
    - bench/tasks/allowlists/01-bug-fix.txt
    - bench/tasks/allowlists/02-brownfield-feature.txt
    - bench/tasks/allowlists/03-interrupted-execution.txt
    - bench/tasks/allowlists/04-fresh-repo-lifecycle.txt
  modified: []

key-decisions:
  - "Chose gorilla/mux (Go, zero dependencies) and zod v3.23.8 (TypeScript) as substrate after 7 other candidates were cloned and rejected — sindresorhus/is and date-fns both had a CLAUDE.md/AGENTS.md at their root, which is exactly the neutrality check this phase's own rule is written to catch"
  - "Task 03 (interrupted execution) reuses Task 01's exact prompt and seeded bug rather than a different one, so the only variable that category tests is kill/resume behavior, not fix difficulty"
  - "Task 02's brownfield feature (.lowercase() string check) was chosen by tracing an existing zod check (cuid2) across the codebase first, confirming it touches exactly three files (types.ts, ZodError.ts, locales/en.ts) before writing the spec, rather than guessing at a plausible-sounding multi-file feature"
  - "Task 04's allowlist lists specific expected filenames (main.go, main.ts, package.json, go.mod, etc.) instead of a bare */** catch-all, even though the fresh-repo task has no pre-existing files to derive an allowlist from — CONTEXT.md and the plan explicitly forbid a catch-all since it would make the unnecessary-modification metric always report zero"

requirements-completed: [PROOF-01]

# Metrics
duration: 58min
completed: 2026-08-17
---

# Phase 186 Plan 02: Substrate Selection and Task Specs Summary

**Pinned gorilla/mux (Go) and zod v3.23.8 (TypeScript) as the benchmark's two neutral substrate repositories after live-testing 9 candidates, then wrote all four task specs and their file allowlists — the declarative inputs the later acceptance-script and runner plans consume.**

## Performance

- **Duration:** 58 min
- **Started:** 2026-08-17T19:18:00Z (approx, worktree reset to correct base commit)
- **Completed:** 2026-08-17T20:16:17Z
- **Tasks:** 3
- **Files modified:** 9 (all created, 0 modified)

## Accomplishments

- Cloned and screened 9 open-source repository candidates against the phase's seven substrate criteria, live — not on reputation. Rejected 7: two TypeScript candidates (`sindresorhus/is`, `date-fns/date-fns`) failed the neutrality check by carrying their own `CLAUDE.md`/`AGENTS.md`; two more (`typebox`, `date-fns` again) were too large; three (`type-is`, `p-limit`, `dotenv`) were too small to support a genuine multi-file feature task; `express` failed the language requirement (plain JS, not TypeScript)
- Verified both final picks by actually running them: `gorilla/mux`'s 72 pre-existing tests pass in 2.2 seconds with zero dependencies to fetch; `zod`'s 488 tests pass in 4.3 seconds after a 39-second `npm install`
- Designed, seeded, and live-verified a genuine one-line bug in `gorilla/mux`'s `cleanPath` (checking `p[0]` instead of `p[len(p)-1]`) together with a test that fails before the fix and passes after — confirmed both states by actually running `go test` against a throwaway copy, not by inspection
- Traced an existing zod validation check (`cuid2`) across the real codebase to confirm the brownfield feature spec (`.lowercase()` string validation) requires genuine edits in exactly three files, before writing the task spec around that shape
- Wrote all four task specs with identical, jargon-free `## The task` prompt text verified free of both systems' internal vocabulary, and confirmed the Aether resume commands named in the interrupted-execution spec (`/ant-resume`, `/ant-recover`) and GSD's (`/gsd-resume-work`) correspond to real, existing command files rather than assumed names
- Wrote four file allowlists, each derived from its task's own `## Done means`, each separating task content from lane working state with the exact required comment marker, none containing a catch-all pattern

## Task Commits

Each task was committed atomically:

1. **Task 1: Choose, verify and pin the two substrate repositories** - `2fac9cf2` (feat)
2. **Task 2: Write the four task specs** - `59e07fce` (feat)
3. **Task 3: Write the four per-task file allowlists** - `0981edff` (feat)

_No plan-metadata commit — SUMMARY.md is committed separately per worktree executor protocol; STATE.md/ROADMAP.md are owned by the orchestrator._

## Files Created/Modified

- `bench/tasks/substrate.md` - both repos, pinned full SHAs, clone/dependency-fetch/test commands, measured LOC and test duration, five neutrality checks per repo, license notes, 7 rejected candidates with the criterion each failed, and the fresh-clone contract
- `bench/tasks/01-bug-fix.md` - the seeded `cleanPath` defect and failing test against `gorilla/mux`, with the exact patch text and live-verified before/after test results
- `bench/tasks/02-brownfield-feature.md` - the `.lowercase()` string-validation feature against `zod`, naming the three files a correct solution must touch
- `bench/tasks/03-interrupted-execution.md` - Task 01's prompt reused under the system-neutral SIGKILL-at-first-write+120s kill rule, with the verified per-lane resume commands
- `bench/tasks/04-fresh-repo-lifecycle.md` - the shared temperature-conversion CLI product goal, with per-lane invocation sequences and the Aether-repo-clean assertion
- `bench/tasks/allowlists/01-bug-fix.txt`, `03-interrupted-execution.txt` - `mux.go` only as task content (mux_test.go deliberately excluded)
- `bench/tasks/allowlists/02-brownfield-feature.txt` - the three feature files plus the test directory
- `bench/tasks/allowlists/04-fresh-repo-lifecycle.txt` - named expected filenames per ecosystem, no catch-all

## Decisions Made

- Rejected `sindresorhus/is` and `date-fns` specifically because they carry AI-tool configuration files (`CLAUDE.md`/`AGENTS.md`) at their repository root — this is exactly the class of contamination the neutrality check exists to catch, and it was caught by running the check, not by assumption
- Chose `zod` over the initially-checked `sindresorhus/is` (too small a structure for a 3-file feature) and `typebox`/`date-fns` (too large) by measuring actual non-test source LOC (6,338 lines) rather than trusting a general size reputation
- Designed Task 02's feature by tracing a real existing check (`cuid2`) through `zod`'s source first, so the "touches at least three files" claim in the task spec is derived from the codebase's actual shape, not asserted
- Task 04's allowlist names specific expected files instead of using directory-level `src/**`/`tests/**` alone, to keep the metric meaningful even though this is the one task category with no pre-existing repository to derive paths from

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Worktree was based on a stale, unrelated commit (`6577f51c`, an old CI fix) instead of the wave-1 completion commit**
- **Found during:** initial worktree branch check at agent startup
- **Issue:** `git merge-base HEAD <expected-base>` did not equal the expected base, and `git merge-base --is-ancestor <expected-base> HEAD` failed — the worktree's HEAD predated Plan 01's work entirely, meaning `bench/lib/*.sh`, `bench/README.md`, and the phase's own CONTEXT/PATTERNS files from Plan 01 were not present
- **Fix:** Confirmed the working tree was clean (no uncommitted changes to lose), then ran `git reset --hard 32d6629d3da42a151dfd1dcf280bad6ce5378dd1` (the expected base commit, "docs(phase-186): update tracking after wave 1") per the worktree branch check's own documented recovery step
- **Files modified:** none (reset only, no code changes)
- **Verification:** post-reset `git log --oneline -1` showed the expected commit; all Plan 01 outputs and phase context files were then present and readable
- **Committed in:** N/A (git reset, not a commit)

---

**Total deviations:** 1 auto-fixed (blocking worktree base correction, required before any task could start)
**Impact on plan:** None on scope — this was a pre-existing worktree setup problem unrelated to Plan 02's actual task content, corrected before any file was touched.

## Issues Encountered

None beyond the worktree base correction above. All three tasks' automated verification commands passed on the first attempt after the corresponding files were written; no rework was needed.

## User Setup Required

None. This plan only writes static specification and allowlist files — no live Claude CLI or GSD run happens in this plan (that begins in later plans, 186-05 onward, and remains blocked on the `ANTHROPIC_API_KEY` provisioning step recorded in the 186-01 summary).

## Next Phase Readiness

**Ready:** `bench/tasks/substrate.md`, the four task specs, and the four allowlists are the complete, self-contained declarative input every later plan in this phase consumes — the acceptance scripts (plan 03/05) read the task specs' `## Done means` sections, and the runner (plan 06) reads `## Per-lane invocation` and the allowlists directly. Both substrate repos are proven to build and test cleanly at their pinned SHAs on this machine, so a fresh clone by any later script will reproduce the same starting point.

**Not addressed here (by design, per plan scope):** no acceptance script exists yet to programmatically check the `## Done means` sections against a real run's output — that is the next plan's job. No live run of any of the four tasks against any of the three lanes has happened yet.

## Self-Check: PASSED

All nine created files confirmed present on disk (`bench/tasks/substrate.md`, the four task specs, the four allowlists, this SUMMARY.md). All three task commits (`2fac9cf2`, `59e07fce`, `0981edff`) confirmed present in `git log --oneline`. `git diff --name-only` against the plan's expected base commit shows exactly the nine files listed in the plan's `files_modified` frontmatter, no more and no fewer.

---
*Phase: 186-baseline-showdown-light*
*Completed: 2026-08-17*
