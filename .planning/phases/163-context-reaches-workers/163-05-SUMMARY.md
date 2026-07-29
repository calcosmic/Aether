---
phase: 163-context-reaches-workers
plan: 05
subsystem: infra
tags: [go, cobra, cli, pheromones, build-finalize, ceremony]

# Dependency graph
requires: []
provides:
  - "runSuggestAnalyze — the suggest-analyze pattern-detection pipeline as a callable Go function, factored out of the cobra RunE closure"
  - "A live, non-blocking call to runSuggestAnalyze inside runCodexBuildFinalize, invoked exactly once per finalize"
  - "A build-closeout tick-to-approve block (cmd/ceremony_cmd.go) showing pending suggestions with copyable approve/dismiss commands"
affects: [165-core-lifecycle-commands, ceremony_cmd, codex_build_finalize]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Extraction precedent: cobra RunE closures reduced to flag-read + call + output, matching printWorkerBriefs' existing shape"
    - "Non-blocking end-of-build hook: collectPendingSuggestions wraps a fallible subsystem call and swallows its error into a logged warning, never propagating it to the caller"
    - "Test-only invocation counters (suggestAnalyzeInvocationCount) as a seam for proving call-count invariants without introducing indirection that would obscure literal grep-based verification patterns"

key-files:
  created: []
  modified:
    - cmd/suggest_analyze.go
    - cmd/suggest_analyze_test.go
    - cmd/codex_build_finalize.go
    - cmd/codex_build_finalize_test.go
    - cmd/ceremony_cmd.go
    - cmd/ceremony_cmd_test.go

key-decisions:
  - "Called runSuggestAnalyze directly (literal call, not through a function-variable indirection) inside runCodexBuildFinalize, to satisfy the plan's must_haves key_link pattern (`runSuggestAnalyze\\(` must appear in cmd/codex_build_finalize.go) even though this makes the task-level 'grep returns 2' acceptance criterion literally read 3 (it also matches the function definition line, which the task's criterion did not account for)"
  - "Added a package-level suggestAnalyzeInvocationCount counter as a pure test seam, incremented at the top of runSuggestAnalyze, to prove 'invoked exactly once per finalize, never per dispatch' without needing indirection at the call site"
  - "Placed the suggest-analyze call after the build's colony state is atomically committed (not immediately after mergeExternalBuildResults), so a suggestion-engine problem can never affect whether a build's own state transition succeeds — still satisfies the plan's 'after worker-result validation' ordering constraint"

requirements-completed: [CONTEXT-05]

# Metrics
duration: 35min
completed: 2026-07-29
---

# Phase 163 Plan 05: suggest-analyze gets a live caller Summary

**`runSuggestAnalyze` extracted into a callable function, wired as a non-blocking call inside `runCodexBuildFinalize`, and its suggestions now render once at build closeout with copyable `aether suggest-approve` commands — its first live caller since the command was written.**

## Performance

- **Duration:** ~35 min
- **Completed:** 2026-07-29
- **Tasks:** 3 completed
- **Files modified:** 6

## Accomplishments

- Factored the entire `suggest-analyze` RunE body into `runSuggestAnalyze(target string, dryRun bool) (map[string]interface{}, error)` — the cobra RunE is now 12 lines (read flags, call, output), and `colony.SanitizeSignalContent` stays the only sanitizer inside the extracted function so every caller inherits injection rejection for free.
- Wired `runSuggestAnalyze` into `runCodexBuildFinalize` via a `collectPendingSuggestions` wrapper: called exactly once per finalize, after the build's colony state is durably committed, non-blocking on failure. This is `suggest-analyze`'s first live caller in the project's history — previously its only documented call site sat in a playbook the runtime never loaded.
- Build closeout (`cmd/ceremony_cmd.go`) now renders a `── Suggestions From This Build ──` block listing each active pending suggestion's type, content, reason, and ID, closing with real, copyable `aether suggest-approve --approve/--dismiss <id>` commands — reusing the existing `filterActiveSuggestions` filter rather than re-implementing dismissed-suggestion logic.
- Deliberate-regression proof performed and observed: removing the `collectPendingSuggestions` call from `runCodexBuildFinalize` and re-running `TestBuildFinalizeCollectsSuggestAnalyzeResults` produced 4 failing subtests (quoted below), then the call was restored and the suite went green again.

## Task Commits

Each task was committed atomically:

1. **Task 1: Make the analysis callable instead of trapped in a RunE closure** - `fc06ec17` (refactor)
2. **Task 2: Give it a live caller at the end of a real build** - `dd2c8535` (feat)
3. **Task 3: Show the suggestions once, at the end, with how to approve them** - `f62ec54b` (feat)

_Note: no TDD RED→GREEN split commits were used — tests and implementation were committed together per task, matching this plan's `tdd="true"` behavior-first authoring but a single-commit-per-task cadence consistent with the rest of this phase's plans._

## Files Created/Modified

- `cmd/suggest_analyze.go` — Extracted `runSuggestAnalyze`; RunE reduced to a thin wrapper; added `suggestAnalyzeInvocationCount` test seam.
- `cmd/suggest_analyze_test.go` — 5 new tests proving the extraction is behavior-preserving, including a CLI-vs-direct-call output comparison.
- `cmd/codex_build_finalize.go` — Added `collectPendingSuggestions` helper and its call site in `runCodexBuildFinalize`; extended the finalize result map with `suggest_analyze_ran`, `pending_suggestion_count`, `pending_suggestions_next`.
- `cmd/codex_build_finalize_test.go` — `TestBuildFinalizeCollectsSuggestAnalyzeResults` with 4 subtests covering persistence, result shape, non-blocking failure, and single-invocation-per-finalize.
- `cmd/ceremony_cmd.go` — `renderPendingSuggestionsBlock` and its wiring into `renderCeremonyCloseout`/`renderCeremonyCloseoutVisual` for the `build` workflow.
- `cmd/ceremony_cmd_test.go` — 4 new tests covering the block's presence, real copyable IDs, dismissed-suggestion exclusion, and the empty-state (no heading, no block).

## Decisions Made

- **Direct literal call, not indirection, at the finalize call site.** The plan's `must_haves.key_links` requires the literal pattern `runSuggestAnalyze\(` to appear in `cmd/codex_build_finalize.go` (this is what the automated phase verification checks against CONTEXT-05). A function-variable indirection (e.g. `runSuggestAnalyzeFn`) would have broken that literal match. Chose the direct call and proved "invoked exactly once, never per dispatch" via a separate, minimal test-only counter instead.
- **Suggest-analyze call placed after the atomic state commit**, not immediately after `mergeExternalBuildResults`. The plan's action text says "after `validateExternalWorkerResultClaimPaths` and `mergeExternalBuildResults` have accepted the worker results" as the *earliest* safe point; placing it at the true end of a successful, committed build keeps the blast radius of a suggestion-engine hiccup to zero — it cannot ever intersect with the many validation/commit failure paths in between.
- **`pending_suggestions_next` is only added to the result when `pending_suggestion_count > 0`.** A finalize result with zero suggestions has nothing to name as a next action; `suggest_analyze_ran` alone carries the "ran vs. never ran" distinction the plan asks to preserve.

## Deviations from Plan

### Auto-fixed Issues

None — no Rule 1/2/3 auto-fixes were needed; the codebase compiled and the existing behavior matched the RESEARCH document's claims exactly.

### Acceptance-criterion discrepancy (documented, not auto-fixed)

**Task 2's literal grep-count acceptance criterion does not account for the function definition line.**
- **Found during:** Task 2 verification
- **Issue:** The task's acceptance criterion states `grep -rn 'runSuggestAnalyze(' cmd/ | grep -v _test | wc -l` should return 2 ("the CLI RunE and the finalize caller"). In practice this pattern also matches the function *definition* line (`func runSuggestAnalyze(target string, dryRun bool) (map[string]interface{}, error) {`), which was already present after Task 1 and already made the count 2 (definition + CLI call) *before* Task 2 added anything. After Task 2's finalize call, the true count is 3 (definition + CLI call + finalize call).
- **Resolution:** Kept the literal, direct call in `codex_build_finalize.go` because the plan's `must_haves.key_links` (a higher-priority, mechanically-checked gate tied to CONTEXT-05) requires exactly this literal pattern to appear in that file. The semantically important invariant the criterion was protecting against — "a third live caller means it leaked into a loop" — is independently verified: `grep -n 'runSuggestAnalyze' cmd/codex_build.go` returns nothing, and `TestBuildFinalizeCollectsSuggestAnalyzeResults`'s fourth subtest proves the call happens exactly once per finalize even with 3 dispatches in the manifest.
- **Files affected:** `cmd/codex_build_finalize.go` (no changes needed beyond what Task 2 already required).
- **Verification:** `grep -rn 'runSuggestAnalyze(' cmd/ | grep -v _test | wc -l` → 3 (documented discrepancy); `grep -n 'runSuggestAnalyze' cmd/codex_build.go` → empty (criterion's real intent satisfied); invocation-count test passes.

---

**Total deviations:** 0 auto-fixes; 1 documented acceptance-criterion discrepancy (no code change required).
**Impact on plan:** None on scope or correctness — the higher-priority must_have (key_link pattern, tied to the phase's requirement-completion gate) is satisfied exactly, and the criterion's underlying safety concern (no loop-based invocation) is independently proven by a passing test.

## Issues Encountered

None beyond the acceptance-criterion discrepancy documented above.

## Deliberate-Regression Proof (Task 2, required by plan)

Removed the `collectPendingSuggestions(root)` call from `runCodexBuildFinalize` (temporarily replaced with `suggestAnalyzeRan, pendingSuggestionCount := false, 0`), then ran:

```
go test ./cmd -run 'TestBuildFinalizeCollectsSuggestAnalyzeResults' -count=1 -v
```

Observed failure (4 of 4 subtests failed):

```
=== RUN   TestBuildFinalizeCollectsSuggestAnalyzeResults/populates_PendingSuggestions_in_COLONY_STATE.json
    codex_build_finalize_test.go:1079: expected pending_suggestions to be populated after a completed build with analysable patterns
=== RUN   TestBuildFinalizeCollectsSuggestAnalyzeResults/result_carries_a_pending_suggestion_count_and_the_approve_next_action
    codex_build_finalize_test.go:1095: expected suggest_analyze_ran=true, got false
    codex_build_finalize_test.go:1099: expected pending_suggestion_count > 0, got 0 (int)
=== RUN   TestBuildFinalizeCollectsSuggestAnalyzeResults/a_suggest-analyze_failure_leaves_finalize_succeeding_with_a_zero_count
    codex_build_finalize_test.go:1137: expected suggest_analyze_ran=true (ran, found nothing), got false
=== RUN   TestBuildFinalizeCollectsSuggestAnalyzeResults/suggest-analyze_is_invoked_exactly_once_per_finalize,_never_per_dispatch
    codex_build_finalize_test.go:1158: expected runSuggestAnalyze to be invoked exactly once for 3 dispatches, got 0 invocations
--- FAIL: TestBuildFinalizeCollectsSuggestAnalyzeResults (0.85s)
FAIL
```

Restored the real call, re-ran the same command: all 4 subtests pass. This is the observed evidence that `suggest-analyze` now has a real, load-bearing caller — not just a command that runs and exits.

## User Setup Required

None — no external service configuration required.

## Verification Results

```
go build ./cmd/aether && go vet ./...          # exit 0
go test ./cmd -run 'TestSuggestAnalyze|TestSuggestApprove|TestBuildFinalize|TestCloseout' -count=1   # ok
go test ./cmd -count=1                          # ok (455.614s, full cmd package)
go test ./pkg/... -count=1                      # 1 pre-existing, unrelated failure (see Known Issues below)
grep -rn 'runSuggestAnalyze(' cmd/ | grep -v _test   # 3 matches: definition + CLI RunE + finalize caller (see Deviations)
grep -n 'runSuggestAnalyze' cmd/codex_build.go       # empty — confirms no per-dispatch loop invocation
```

## Known Issues (pre-existing, out of scope)

`TestCodexReadOnlyProfileSelectsReadOnlySandbox` (`pkg/codex/permission_profile_test.go:83`) fails in this environment with `codex login status failed: timed out` — a network/auth-dependent test. Confirmed unrelated: zero diff under `pkg/codex/` in this plan's commits. Logged to `.planning/phases/163-context-reaches-workers/deferred-items.md` per the scope-boundary rule; not fixed here.

## Next Phase Readiness

CONTEXT-05 is now satisfied under CLAUDE.md's Definition of Done: a command exists (`aether build --plan-only` → external completion → `aether build-finalize`, the real Codex-CLI end-to-end path) that runs `suggest-analyze` and would fail its own regression-proof test if the caller were ever removed again. Suggestions are visible at closeout with a real, copyable path to approval. No blockers for subsequent 163 plans or Phase 165 (which owns `build.md` structurally and was explicitly not touched by Task 3, per the plan's own guardrail).

---
*Phase: 163-context-reaches-workers*
*Completed: 2026-07-29*
