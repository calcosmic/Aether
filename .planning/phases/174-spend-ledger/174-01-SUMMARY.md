---
phase: 174-spend-ledger
plan: 01
subsystem: infra
tags: [go, token-measurement, worker-dispatch, json]

# Dependency graph
requires:
  - phase: 173-delegation-guard
    provides: SpawnEntry.ParentName linkage used by later plans in this phase for roll-up
provides:
  - "A third WorkerUsage.Source tier (UsageSourceSessionTranscript) distinct from provider and estimate"
  - "Estimated() predicate mirroring Measured(), so the measured/estimated split never conflates the two"
  - "TotalInputTokens() and BilledTotalTokens() exported total helpers every later plan in this phase must call instead of re-deriving a sum"
  - "internalWorkerResult.Usage carries codex.WorkerResult.Usage through mapInternalWorkerResult on the direct/Codex-CLI dispatch path"
affects: [174-02, 174-03, 174-04, 174-05, 174-06, 174-07, 174-08, 174-09]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Every new total in the spend ledger routes through WorkerUsage.BilledTotalTokens()/TotalInputTokens(), never re-derived by summing fields directly"
    - "New usage tests assert literal externally-sourced numbers, never a call back into the code under test (regression discipline for the 186x undercount)"

key-files:
  created:
    - cmd/internal_worker_usage_test.go
  modified:
    - pkg/codex/usage.go
    - pkg/codex/usage_test.go
    - cmd/internal_worker_adapter.go

key-decisions:
  - "Explanatory comment for the new Usage field moved onto the struct field declaration in internalWorkerResult rather than inline above the return-literal's Usage: result.Usage, line -- a comment inline in the literal breaks gofmt's alignment group and produces spacing that does not match this plan's own acceptance-criteria grep"
  - "Task 2's third behavior bullet (zero-value Usage produces no usage key) was adjusted to match real Go encoding/json semantics: omitempty never omits a struct-typed field entirely (only pointers/slices/maps/scalars), so a zero-value Usage marshals as an empty nested object {} rather than being absent. The test asserts the correct behavior (empty object, and Empty() true after round-trip) instead of the plan's stated literal absence, since making it truly absent would require a pointer field and break the acceptance criteria's required Usage: result.Usage, literal copy"

requirements-completed: [SPEND-01, SPEND-03, SPEND-04]

# Metrics
duration: 30min
completed: 2026-08-14
---

# Phase 174 Plan 01: Usage Vocabulary and Adapter Persistence Summary

**Added a third, honest `WorkerUsage` source tier plus the two exported token totals the spend report needs, and stopped `mapInternalWorkerResult` from silently discarding a provider-measured worker's token usage.**

## Performance

- **Duration:** ~30 min
- **Started:** 2026-08-14T13:52Z (context read began)
- **Completed:** 2026-08-14T14:19Z
- **Tasks:** 2
- **Files modified:** 4 (3 modified, 1 created)

## Accomplishments
- `pkg/codex/usage.go` now has three source tiers (`provider`, `session-transcript`, `estimate`) so a wrapper-path measurement can never be mistaken for a provider-grade one (D-06), and `Measured()`'s behavior is unchanged — only its doc comment now states that plainly
- Added `Estimated()`, `TotalInputTokens()`, and `BilledTotalTokens()` — the two exported totals every later plan in this phase must call rather than re-summing fields, closing off a repeat of the 186x undercount shape
- `internalWorkerResult` now carries a `Usage` field, and `mapInternalWorkerResult` copies `codex.WorkerResult.Usage` through instead of dropping it — the direct/Codex-CLI dispatch path's provider measurement now survives into the durable adapter result and a JSON round-trip
- `codexExternalBuildWorkerResult` (the wrapper-path struct an orchestrating LLM submits) deliberately still has no `usage` field, locked by a grep in this plan's own acceptance criteria — later plans, not this one, define how a wrapper-path figure gets tagged non-provider-grade

## Task Commits

Each task was committed atomically:

1. **Task 1: Add the session-transcript source tier and the two exported totals** - `d6cf0c2f` (feat)
2. **Task 2: Stop the internal worker adapter discarding the provider measurement** - `f237df2f` (feat)

**Plan metadata:** (this commit)

## Files Created/Modified
- `pkg/codex/usage.go` - Added `UsageSourceSessionTranscript`, `Estimated()`, `TotalInputTokens()`, `BilledTotalTokens()`; extended doc comments on `Source` and `Measured()`
- `pkg/codex/usage_test.go` - Added `TestSessionTranscriptUsageIsNeverProviderGrade`, `TestUsageTotalsExposeDisjointInputAndBilledTotal`, `TestParseUsageStillTagsProviderAndYieldsDocumentedTotal`
- `cmd/internal_worker_adapter.go` - Added `Usage codex.WorkerUsage` field to `internalWorkerResult`; `mapInternalWorkerResult` now copies `result.Usage` through
- `cmd/internal_worker_usage_test.go` - New file: `TestInternalWorkerResultCarriesProviderUsage` (field-for-field copy, JSON round-trip, zero-value marshalling)

## Decisions Made
- Kept `billedTotal()` unexported and its body/comment byte-identical, per the plan's explicit instruction — `BilledTotalTokens()` wraps it rather than replacing it
- Placed the field-level explanatory comment on the `internalWorkerResult.Usage` struct field itself instead of directly above the `Usage: result.Usage,` line in the return literal, so gofmt keeps that literal in the same alignment group as its neighbors (see Deviations)
- Left `codexExternalBuildWorkerResult` in `cmd/codex_build_finalize.go` untouched, per plan instruction and the T-174-04 threat mitigation — a `usage` field there would make a wrapper-asserted token count structurally acceptable

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Comment placement adjusted to keep the required literal exactly matching the plan's acceptance-criteria grep**
- **Found during:** Task 2
- **Issue:** The plan's action text asked for an explanatory comment directly above `Usage: result.Usage,` inside the `mapInternalWorkerResult` return literal. gofmt treats any full-line comment inside a struct literal as breaking its column-alignment group, so the line reformats to `Usage:    result.Usage,` (4 spaces) instead of the 9-space alignment the plan's own acceptance criterion (`grep -n 'Usage:         result.Usage'`) requires.
- **Fix:** Moved the explanatory comment onto the `Usage codex.WorkerUsage` field declaration in the `internalWorkerResult` type instead, leaving the return literal's `Usage: result.Usage,` line in the same gofmt alignment group as the rest of the struct so it produces the exact spacing the acceptance criterion checks for.
- **Files modified:** `cmd/internal_worker_adapter.go`
- **Verification:** `grep -n 'Usage:         result.Usage' cmd/internal_worker_adapter.go` matches; `gofmt -l cmd/internal_worker_adapter.go` reports no formatting diff.
- **Committed in:** `f237df2f`

**2. [Rule 1 - Bug] Zero-value Usage test corrected to match real Go `encoding/json` behavior**
- **Found during:** Task 2
- **Issue:** The plan's third `<behavior>` bullet stated that a zero-value `Usage` produces marshalled JSON with no `usage` key at all, "the omitempty convention already used by `tool_count`." Go's `encoding/json` `omitempty` only omits pointers, slices, maps, and zero scalar values — it never treats a struct-typed field (like `codex.WorkerUsage`) as empty, regardless of its contents. `tool_count`'s own omission works only because `ToolCount` is a plain `int`, not a struct; the two fields are not actually the same mechanism despite sharing a tag name.
- **Fix:** Verified the actual behavior (a zero-value `Usage` marshals as an empty nested object, `"usage":{}}`, because each field inside `WorkerUsage` carries its own `omitempty` tag) and wrote the test to assert that real behavior plus `Usage.Empty()` being true after round-tripping, rather than asserting key absence. Making the key truly absent would require changing `Usage` to a pointer field, which would break this same task's `Usage: result.Usage,` acceptance-criteria literal.
- **Files modified:** `cmd/internal_worker_usage_test.go`
- **Verification:** `go test ./cmd/... -run TestInternalWorkerResultCarriesProviderUsage -count=1 -v` passes.
- **Committed in:** `f237df2f`

---

**Total deviations:** 2 auto-fixed (both Rule 1 — bugs in the plan's own literal expectations discovered by running gofmt and the actual test against Go's standard library, not in the design intent)
**Impact on plan:** Both auto-fixes preserve every acceptance-criteria grep in the plan text exactly as written. No scope creep; SPEND-01/03/04 are fully satisfied.

## Issues Encountered
None beyond the two documented deviations above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `pkg/codex/usage.go`'s vocabulary (`UsageSourceSessionTranscript`, `Estimated()`, `TotalInputTokens()`, `BilledTotalTokens()`) is ready for plan 174-02 onward to build the ledger, the wrapper-path parsers, and `aether spend` on top of
- The direct/Codex-CLI dispatch path's SPEND-01 persistence gap is closed; the wrapper-path gap (SPEND-02) remains open for plans 174-04/174-05 as scoped
- No blockers for the next wave

---
*Phase: 174-spend-ledger*
*Completed: 2026-08-14*
