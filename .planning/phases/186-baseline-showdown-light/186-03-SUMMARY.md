---
phase: 186-baseline-showdown-light
plan: 03
subsystem: testing
tags: [bash, jq, benchmark-harness, token-measurement, git-cleanliness, operator-log]

# Dependency graph
requires:
  - phase: 186-01
    provides: "bench/lib/hermetic-home.sh isolated-HOME pattern (fail/step helper convention this plan's libraries follow)"
provides:
  - "bench/lib/measure-tokens.sh — harness-side token measurement from a Claude Code session transcript, four-field disjoint arithmetic matching pkg/codex/usage.go's billedTotal() exactly"
  - "bench/lib/git-cleanliness.sh — orphan-branch, tool-residue, uncommitted-residue and unnecessary-modification checker with an auditable numeric cleanliness_score"
  - "bench/lib/operator-log.sh — append-only timestamped operator-input log with scripted/unscripted typing and derived autonomy"
  - "bench/lib/fixtures/transcript-sample.jsonl — committed transcript fixture with hand-computed known totals"
  - "bench/lib/selftest.sh — five-gate self-test proving all three libraries against literal expected values, runnable via `make bench-selftest`"
affects: [186-04, 186-05, 186-06, 186-07, 192-full-comparison]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Harness-side token measurement reads the provider's own on-disk session transcript, never a system's self-report — tags output source=session-transcript"
    - "Billed-total arithmetic is a disjoint four-field sum (input + cache_read + cache_creation + output), reproduced verbatim from pkg/codex/usage.go's billedTotal() comment so the same 186x-undercount mistake (summing only input+output) cannot recur here"
    - "Self-test expected values are hand-computed literal constants written into the test file itself, never re-derived by calling the parser's own formula — this is the specific defect shape that let the original 186x undercount ship green"
    - "bash 3.2 (macOS default /bin/bash) compatibility: no namerefs, array expansions use ${arr[@]:-} to avoid set -u tripping on empty arrays"
    - "operator-log.sh deliberately omits set -e at file scope (unlike this directory's other three libraries) because its functions are designed for the caller to inspect $? after the call — a top-level set -e would abort the calling shell before that inspection could run"

key-files:
  created:
    - bench/lib/measure-tokens.sh
    - bench/lib/git-cleanliness.sh
    - bench/lib/operator-log.sh
    - bench/lib/fixtures/transcript-sample.jsonl
    - bench/lib/selftest.sh
  modified:
    - Makefile

key-decisions:
  - "operator-log.sh uses `set -uo pipefail` (no -e) instead of this directory's `set -euo pipefail` convention, because its acceptance test (and any real caller) needs to inspect $? after a failing call without the sourcing shell dying first"
  - "selftest.sh redefines its own fail()/step() AFTER sourcing the three libraries, not before — each library defines a same-named fail()/step() pair (repo convention from scripts/smoke-daily-driver.sh), and sourcing silently overwrites whichever came first"

requirements-completed: [PROOF-01, PROOF-03]

# Metrics
duration: 55min
completed: 2026-08-17
---

# Phase 186 Plan 03: Measurement Libraries Summary

**Three harness-side measurement libraries — token counting from real Claude Code session transcripts, git cleanliness scoring, and a timestamped operator-input log — plus a five-gate self-test that proves each one's arithmetic against hand-computed literals rather than against its own formula.**

## Performance

- **Duration:** 55 min
- **Started:** 2026-08-17T20:05:00Z (approx, after worktree base correction)
- **Completed:** 2026-08-17T21:00:00Z
- **Tasks:** 4
- **Files modified:** 6 (5 created, 1 modified)

## Accomplishments
- Built `measure-tokens.sh`, which reads a Claude Code session transcript directly off disk (glob-matched, never a hand-constructed path) and sums the SAME four disjoint fields pkg/codex/usage.go's `billedTotal()` sums — input + cache_read + cache_creation + output — across every assistant turn in the whole session, not just the last event
- Built `git-cleanliness.sh`, computing orphan branches, uncommitted residue, tool-internal path leakage against a per-lane allowlist, and unnecessary modifications, with an auditable numeric `cleanliness_score` whose formula is documented as a comment in the script itself
- Built `operator-log.sh`, an append-only JSONL log where every input must be typed exactly `scripted` or `unscripted` (any other value is refused by name), and where the `autonomous` verdict is always DERIVED from the unscripted count — never something a run can declare about itself
- Built a committed fixture transcript reproducing pkg/codex/usage.go's documented Anthropic worked example verbatim (50 input / 100,000 cache read / 2,000 cache creation / 500 output), plus a second turn, so the harness and the runtime's own doc comment are checkable against the identical numbers
- Wrote `selftest.sh`'s five gates with the expected totals (102,650 and 102,110) hand-computed and written as literal constants in the test file — then ACTUALLY broke the arithmetic (removed the cache_read term), confirmed the self-test caught it and printed expected-vs-actual, then restored the original code and confirmed `git diff` showed zero residual change

## Task Commits

Each task was committed atomically:

1. **Task 1: Harness-side token measurement from session transcripts** - `3190f125` (feat)
2. **Task 2: Git cleanliness and unnecessary-modification checker** - `255422e7` (feat)
3. **Task 3: Operator input log** - `e8281d80` (feat)
4. **Task 4: Fixtures and a self-test that proves the arithmetic independently** - `0bb13a0b` (feat)

_No plan-metadata commit — SUMMARY.md is committed separately per worktree executor protocol; STATE.md/ROADMAP.md are owned by the orchestrator._

## Files Created/Modified
- `bench/lib/measure-tokens.sh` - `measure_tokens HOME_DIR`: globs `$HOME_DIR/.claude/projects/*/*.jsonl`, sums the four on-disk usage fields across every assistant turn, tags `source: session-transcript`, refuses the operator's real home and a HOME with zero transcripts
- `bench/lib/git-cleanliness.sh` - `git_cleanliness REPO_PATH STARTING_BRANCH STARTING_SHA ALLOWLIST_FILE`: orphan branches, uncommitted residue, tool-internal leakage vs the allowlist's lane block, unnecessary modifications vs the full allowlist, and a `100 - 10*orphan - 5*tool_internal - 2*uncommitted` cleanliness score floored at 0
- `bench/lib/operator-log.sh` - `oplog_start/input/wait_start/wait_end/event/end/summary`: one compact JSON line per call, `autonomous` derived strictly from `unscripted_inputs == 0`
- `bench/lib/fixtures/transcript-sample.jsonl` - two-turn fixture; turn 1 is pkg/codex/usage.go's documented worked example verbatim, turn 2 is small round numbers, plus a user line and an unrelated-type line the parser must skip
- `bench/lib/selftest.sh` - five gates (token arithmetic, no-transcript refusal, git cleanliness, operator log, real-home refusal), prints `BENCH SELFTEST PASS` on success
- `Makefile` - added `bench-selftest` target invoking `bench/lib/selftest.sh`, alongside the existing `smoke` target

## Fixture's Hand-Computed Expected Totals

Recorded here and in `bench/lib/selftest.sh`'s header comment, computed by hand from the fixture — NOT by calling `measure_tokens` and trusting its own answer:

- Turn 1 (pkg/codex/usage.go's documented Anthropic example): input=50, cache_read=100000, cache_creation=2000, output=500 → total=**102,550**, input-side=**102,050**
- Turn 2 (small round numbers): input=10, cache_read=20, cache_creation=30, output=40 → total=**100**, input-side=**60**
- Fixture overall (sum across both turns — a session transcript's true cost is every turn added, not the last event): total_tokens=**102,650**, total_input_tokens=**102,110**

## Decisions Made
- `operator-log.sh` does not use `set -euo pipefail` at file scope, unlike this plan's other three libraries. Its acceptance test (`oplog_input scripted x` with `OPERATOR_LOG` unset, then `test $? -ne 0`) needs the calling shell to survive the failing call so it can inspect the exit code — with `set -e` inherited from a sourced library, the calling `bash -c` script died at the failing statement before `test $? -ne 0` could ever run. Confirmed by running the plan's exact literal verify line before and after the fix (failed with `set -e`, passed with `set -uo pipefail` only)
- `selftest.sh` defines its own `fail()`/`step()` helpers AFTER sourcing the three libraries, not before. All four scripts in this plan follow the repo's `fail()`/`step()` naming convention (from `scripts/smoke-daily-driver.sh`), so sourcing three libraries that each define the same two function names silently overwrote `selftest.sh`'s own definitions with whichever library sourced last — gate-1 failures were printing `GIT-CLEANLINESS FAIL:` instead of `SELFTEST FAIL:`. Fixed by moving the definitions to after the `source` lines
- Array expansions throughout `git-cleanliness.sh` use `${arr[@]:-}` rather than bare `${arr[@]}`. The development machine's `/bin/bash` is 3.2.57 (macOS default), which — unlike bash 4.4+ — throws an unbound-variable error under `set -u` when expanding a genuinely empty array. Discovered via live execution (Task 2's first real fixture-repo run), not by inspection

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `local -n` (bash namerefs) is unsupported on this machine's bash**
- **Found during:** Task 2, first live run of `git-cleanliness.sh` against the constructed fixture repo
- **Issue:** `_path_matches_any` originally took an array name and dereferenced it via `local -n patterns_ref="$2"` — a bash 4.3+ feature. macOS ships bash 3.2.57 as `/bin/bash`, so this failed immediately with `local: -n: invalid option`
- **Fix:** Rewrote `_path_matches_any` to accept patterns as remaining positional arguments (`shift; for pattern in "$@"`) instead of a nameref, and updated both call sites to pass `"${array[@]}"` directly
- **Files modified:** `bench/lib/git-cleanliness.sh`
- **Verification:** Re-ran against the fixture repo; produced correct JSON with no bash errors
- **Committed in:** `255422e7` (Task 2 commit)

**2. [Rule 1 - Bug] `set -u` on bash 3.2 throws on empty-array expansion**
- **Found during:** Task 2, same live run, after fixing the nameref issue
- **Issue:** `"${tool_internal_paths[@]}"` and similar expansions threw `unbound variable` when the array was genuinely empty (0 elements) — bash 3.2's `set -u` behavior here differs from 4.4+, which tolerates an empty array expansion
- **Fix:** Added a `_json_array_from` helper that renders a bash array as JSON with an explicit `$# -eq 0` guard, and switched every potentially-empty array expansion (in both `for` loops and JSON-encoding call sites) to the `${arr[@]:-}` default-value form
- **Files modified:** `bench/lib/git-cleanliness.sh`
- **Verification:** Re-ran with a fixture producing zero tool-internal paths; stdout stayed clean JSON with no stderr noise, `jq -e` parsed successfully
- **Committed in:** `255422e7` (Task 2 commit)

**3. [Rule 1 - Bug] `set -e` inherited from a sourced library defeated `operator-log.sh`'s own acceptance test**
- **Found during:** Task 3, running the plan's exact literal acceptance verify command
- **Issue:** With `set -euo pipefail` at file scope (matching this directory's other libraries), sourcing `operator-log.sh` into a caller's shell meant a failing `oplog_input` call aborted that shell immediately — before the caller's own `test $? -ne 0` line could execute to confirm the failure. The plan's verify command (`bash -c 'source ...; oplog_input scripted x 2>/dev/null; test $? -ne 0'`) could never pass under this configuration
- **Fix:** Removed `-e` from `operator-log.sh`'s `set` invocation (kept `-u` and `pipefail`); every function still returns 1 explicitly on every error path via `_oplog_fail`, so callers can rely on `$?` without the sourcing shell dying underneath them
- **Files modified:** `bench/lib/operator-log.sh`
- **Verification:** Re-ran the plan's exact literal verify line before and after; failed (exit 1) before, passed (exit 0) after
- **Committed in:** `e8281d80` (Task 3 commit)

**4. [Rule 1 - Bug] `jq -n` pretty-prints by default, breaking the "one JSON line per record" contract**
- **Found during:** Task 3, first manual run of the synthetic operator-log sequence
- **Issue:** Every `_oplog_append` call used `jq -n` without `-c`, which pretty-prints across multiple lines. A 7-event sequence produced a 40-line file instead of 7, breaking `oplog_summary`'s per-line JSONL assumption and the acceptance criterion "every emitted line parses as JSON"
- **Fix:** Added `-c` (compact output) to all six `jq -n` invocations in `operator-log.sh`
- **Files modified:** `bench/lib/operator-log.sh`
- **Verification:** Re-ran the synthetic sequence; file line count matched event count exactly (7 lines for 7 calls), each line individually parseable via `jq -e .`
- **Committed in:** `e8281d80` (Task 3 commit)

**5. [Rule 1 - Bug] `fail()`/`step()` name collision across sourced libraries in `selftest.sh`**
- **Found during:** Task 4, deliberate-break verification step
- **Issue:** `selftest.sh` defined its own `fail()` before sourcing `measure-tokens.sh`, `git-cleanliness.sh`, and `operator-log.sh` — each of which also defines a same-named `fail()`. Sourcing silently overwrote `selftest.sh`'s definition with `git-cleanliness.sh`'s (the last one sourced), so gate failures printed `GIT-CLEANLINESS FAIL:` instead of the intended `SELFTEST FAIL:` prefix, even though the exit behavior itself was still correct
- **Fix:** Moved `selftest.sh`'s `fail()`/`step()` definitions to after the three `source` lines
- **Files modified:** `bench/lib/selftest.sh`
- **Verification:** Re-ran the deliberate-break test (cache_read term removed); output now correctly reads `SELFTEST FAIL: gate 1: total_tokens expected=102650 actual=2630`
- **Committed in:** `0bb13a0b` (Task 4 commit)

---

**Total deviations:** 5 auto-fixed bugs, all found via live execution rather than static inspection. Every one was necessary for the scripts to run correctly and be testable at all on this machine (macOS with bash 3.2.57). No scope creep — all fixes stayed inside the files the plan already scoped for each task.

## Issues Encountered

None blocking. The five deviations above were all caught and fixed during normal task execution, each verified by re-running the specific check that exposed it.

## Next Phase Readiness

**Ready:** `bench/lib/measure-tokens.sh`, `bench/lib/git-cleanliness.sh`, and `bench/lib/operator-log.sh` are proven, reusable measurement primitives — every later lane runner and results-table generator (186-04 through 186-07) can source them directly. `bench/lib/selftest.sh` (via `make bench-selftest`) is the one-command re-check that the arithmetic still holds after any future change to these three files.

**No blockers introduced by this plan.** The `ANTHROPIC_API_KEY` provisioning blocker recorded in 186-01's SUMMARY (macOS Keychain auth does not survive a `$HOME` override) is unaffected by this plan's scope — these libraries measure whatever transcript a run produces; they do not themselves run Claude CLI dispatches.

## Self-Check: PASSED

All five created/modified files confirmed present on disk (`bench/lib/measure-tokens.sh`, `bench/lib/git-cleanliness.sh`, `bench/lib/operator-log.sh`, `bench/lib/fixtures/transcript-sample.jsonl`, `bench/lib/selftest.sh`, `Makefile`). All four task commits (`3190f125`, `255422e7`, `e8281d80`, `0bb13a0b`) confirmed present in `git log --oneline -5`. `bash bench/lib/selftest.sh` and `make bench-selftest` both re-run clean (exit 0, final line `BENCH SELFTEST PASS`) as the final action before writing this summary.

---
*Phase: 186-baseline-showdown-light*
*Completed: 2026-08-17*
