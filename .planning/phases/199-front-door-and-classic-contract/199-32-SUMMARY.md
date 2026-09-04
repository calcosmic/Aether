---
phase: 199-front-door-and-classic-contract
plan: "32"
subsystem: lifecycle recovery and worktree safety
tags: [go, cobra, lifecycle, recovery, worktrees, runtime-vocabulary]
requires:
  - phase: 199-17
    provides: recovery and closeout semantics
  - phase: 199-26
    provides: canonical lifecycle restoration route
provides:
  - Executed recovery-route vocabulary ratchet for lifecycle and preserved worktree messages
  - One raw `aether resume` restoration door and an explicitly read-only maintenance inspection route
affects: [199-28, lifecycle rendering, worktree preservation]
tech-stack:
  added: []
  patterns:
    - Executed message-builder fixtures paired with AST string-literal vocabulary checks
    - Preserved-work messages separate inspection from lifecycle restoration
key-files:
  created:
    - cmd/runtime_recovery_routes_199_test.go
  modified:
    - cmd/codex_visuals.go
    - cmd/init_cmd.go
    - cmd/entomb_cmd.go
    - cmd/clash.go
    - cmd/worktree_safety.go
    - cmd/codex_build_worktree.go
    - cmd/worktree_reap.go
key-decisions:
  - "Use raw `aether resume` as the sole lifecycle-restoration command."
  - "Use `aether maintenance recovery-inspect` only to inspect preserved work, with `State effect: none` stated in the message."
patterns-established:
  - "Runtime vocabulary tests execute visual, NO_COLOR, JSON, and preservation message builders; AST checks inspect only string literals."
requirements-completed: [SYNTH-01, CEC-02, CEC-04, LIFE-03, LIFE-04, LIFE-05]
duration: 9min
completed: 2026-09-04
---

# Phase 199 Plan 32: Runtime Recovery Routes Summary

**Lifecycle recovery now restores only through `aether resume`, while preserved worker work is inspected safely through the explicit read-only maintenance route.**

## Performance

- **Duration:** 9 min
- **Started:** 2026-09-04T22:07:10Z
- **Completed:** 2026-09-04T22:16:54Z
- **Tasks:** 2/2
- **Files modified:** 8

## Accomplishments

- Replaced active lifecycle, init, entomb, clash, worktree, cancellation, receipt, and reap suggestions that exposed retired recovery commands.
- Added one top-level `TestRuntimeRecoveryRoutes199` and reusable `collectRuntimeRecoverySuggestions199` for Plan 199-28.
- Tested real visual, NO_COLOR, JSON, and saved-work message builders; string-literal AST checks intentionally ignore comments and identifiers.

## Task Commits

1. **Task 1 RED: failing lifecycle recovery-route contract** — `1324792e` (test)
2. **Task 1 GREEN: lifecycle renderer, init, and entomb routes** — `81ea2586` (feat)
3. **Task 2 RED: failing worktree preservation-route contract** — `f5bf7707` (test)
4. **Task 2 GREEN: clash and worktree preservation routes** — `04003065` (feat)

## Files Created/Modified

- `cmd/runtime_recovery_routes_199_test.go` — shared executed fixtures and literal-only vocabulary ratchet.
- `cmd/codex_visuals.go`, `cmd/init_cmd.go`, `cmd/entomb_cmd.go` — lifecycle-preserved-work messages name inspection and resume separately.
- `cmd/clash.go`, `cmd/worktree_safety.go`, `cmd/codex_build_worktree.go`, `cmd/worktree_reap.go` — preserve branch/path evidence while replacing the retired recovery suggestion.

## Verification

- `go test ./cmd -run '^TestRuntimeRecoveryRoutes199$' -count=1` — passed (16 assertions/subtests after Task 2).
- `go test ./cmd -run '^(TestRuntimeRecoveryRoutes199|TestResumeVisualWorktreeCleanup|TestWorktree.*(Preserv|Reap|Clash))$' -count=1` — passed (17 tests).
- `go test -race ./cmd -run '^(TestRuntimeRecoveryRoutes199|TestWorktree.*(Preserv|Reap|Clash))$' -count=1` — passed (16 tests).
- `go build ./cmd` — passed.

The known repository-wide baseline remains 8765 passing, 226 failing, and 11 skipped tests in unrelated mapped families; no focused regression was found in the owned lifecycle/worktree paths.

## Decisions Made

- Keep `aether maintenance recovery-inspect` diagnostic-only and explicitly state `State effect: none`.
- Return runnable lifecycle restoration to raw `aether resume` after every preserved-work inspection message.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added the import required by the extracted clash message builder**
- **Found during:** Task 2
- **Issue:** The new shared builder wrote to `os.Stderr`, but `clash.go` did not import `os`.
- **Fix:** Added the standard-library import and reran the exact recovery route and worktree tests.
- **Files modified:** `cmd/clash.go`
- **Verification:** Focused recovery route test and relevant worktree tests passed.
- **Committed in:** `04003065`

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Required only to compile the planned message-builder extraction; no scope expansion.

## Known Stubs

None.

## Issues Encountered

- The broad command-package suite is already known to carry unrelated mapped baseline failures. Focused lifecycle/worktree and race gates passed; no new owned failure was observed.

## User Setup Required

None.

## Next Phase Readiness

Plan 199-28 can call `collectRuntimeRecoverySuggestions199` to ratchet this same executed output set.

## Self-Check: PASSED

All eight owned files exist, all four task commits are reachable, no owned runtime string literal contains a retired recovery command, and no test process started by this executor remains.

---
*Phase: 199-front-door-and-classic-contract*
*Completed: 2026-09-04*
