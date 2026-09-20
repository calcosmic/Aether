---
phase: 199-front-door-and-classic-contract
plan: "09"
subsystem: lifecycle-orientation
tags: [go, status, lifecycle-projection, health, readiness, wrappers]

requires:
  - phase: 199-front-door-and-classic-contract
    plan: "04"
    provides: Immutable lifecycle facts, pure lifecycle projection, and shared Next Up policy
provides:
  - Complete ten-section status rendering from one immutable lifecycle projection
  - Compact status as a bounded strict subset of the same semantic result
  - Evidence-backed verified, blocked, degraded, unavailable, and unknown health/readiness states
  - Runtime-authoritative status wrappers with full-default and compact argument passthrough
affects: [phase-view, history-view, watch, lifecycle-closeouts, platform-wrappers]

tech-stack:
  added: []
  patterns: [single-load orientation, pure projection rendering, evidence-backed health states, passive runtime wrappers]

key-files:
  created:
    - cmd/lifecycle_status_render.go
    - cmd/lifecycle_status_199_test.go
    - cmd/lifecycle_health_199_test.go
  modified:
    - cmd/status.go
    - cmd/command_truth.go
    - .aether/commands/status.yaml
    - .claude/commands/ant-status.md
    - .claude/commands/ant/status.md
    - .opencode/commands/ant/status.md

key-decisions:
  - "Status loads lifecycle facts once, projects once, and treats full/compact as views of that same result."
  - "Health and readiness use discrete evidence-backed states instead of percentage-derived maturity labels."
  - "Status wrappers only invoke the Go runtime with argument passthrough; they never inspect host files, costs, or activity."

patterns-established:
  - "One snapshot, multiple views: full, compact, JSON, and health/readiness reuse the versioned lifecycle vocabulary and shared Next Up answer."
  - "Absence stays explicit: no live actor is stated plainly, missing cost is Unreported, and missing verification is Unknown."
  - "Snapshot versus stream: status owns authoritative orientation while watch remains an event-stream surface."

requirements-completed: [CEC-01, CEC-02, LIFE-03]

duration: 19min
completed: 2026-09-03
---

# Phase 199 Plan 09: Authoritative Status Snapshot Summary

**Status now renders one immutable lifecycle snapshot in full or compact form, while maturity reports five evidence-backed health/readiness states and every platform wrapper delegates to the same Go authority.**

## Performance

- **Duration:** 19 minutes
- **Started:** 2026-09-03T21:27:42Z
- **Completed:** 2026-09-03T21:47:00Z
- **Tasks:** 2/2
- **Files modified:** 9 production/test files

## Accomplishments

- Replaced the legacy status command path with one causally read-only `loadLifecycleFacts` call and one pure `projectLifecycle` call per invocation.
- Added the prescribed ten-section full dashboard and an at-most-twelve-line compact subset, with active ants separated from terminal outcomes and absent evidence rendered honestly.
- Preserved research, territory, and Dreams counts/latest paths while labeling local Dreams notes as unverified rather than promoting them into verification evidence.
- Replaced inferred maturity percentages with typed health/readiness results carrying evidence IDs, timestamps, and reasons for verified, blocked, degraded, unavailable, and unknown states.
- Synchronized canonical, Claude flat/nested, and OpenCode status surfaces around the exact runtime-owned full/compact contract.

## Task Commits

Each task was committed atomically; Task 1 retained its required RED and GREEN gates:

1. **Task 1: Render full and compact status from one projection** — `42c99bba` (RED), `22449095` (GREEN)
2. **Task 2: Synchronize the authoritative status wrapper** — `054bc4c7` (wrapper contract and synchronized surfaces)

**Plan metadata:** committed with this summary.

## Files Created/Modified

- `cmd/lifecycle_status_render.go` — pure full/compact status renderer, responsive line fitting, active/outcome separation, honest absence labels, and semantic color boundary.
- `cmd/lifecycle_status_199_test.go` — full order, compact subset, responsive, output-mode, zero-write, absent-evidence, and wrapper-authority proofs.
- `cmd/lifecycle_health_199_test.go` — all five health/readiness states and zero-write fixture coverage.
- `cmd/status.go` — single-load/single-projection read-only status command with `--compact` selection.
- `cmd/command_truth.go` — evidence-backed health/readiness projection and maturity rendering over lifecycle facts.
- `.aether/commands/status.yaml` — canonical full-default, optional-compact, runtime-authority contract.
- `.claude/commands/ant-status.md` — generated flat Claude status wrapper.
- `.claude/commands/ant/status.md` — generated nested Claude status wrapper.
- `.opencode/commands/ant/status.md` — generated OpenCode status wrapper.
- `deferred-items.md` — broader-suite migration evidence for superseded status expectations.

## Decisions Made

- Kept compact status as a view selector over `LifecycleProjection`, not a separately computed result, so identity, progress, activity, cost, and Next Up cannot diverge from full status.
- Classified only recorded verification, blockers, closure evidence, and open findings; optional missing dashboard sources do not become invented health claims.
- Retained terminal worker records as recent outcomes while reserving active-ant wording for runtime statuses that `agent.IsLiveSpawnStatus` classifies as in flight.
- Made wrapper argument passthrough exact so an empty argument list selects full status and `--compact` reaches the same runtime without wrapper-side interpretation.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Corrected the compact cost fixture to preserve recorded evidence**

- **Found during:** Task 1 (full/compact renderer GREEN implementation)
- **Issue:** The compact fixture recorded 1,500 provider tokens but asserted `Unreported`, contradicting the plan rule that recorded elapsed/cost remains visible when available.
- **Fix:** Changed the compact assertion to require `1500 tokens`; missing-cost fixtures continue to require `Unreported` and forbid `$0`.
- **Files modified:** `cmd/lifecycle_status_199_test.go`
- **Verification:** The complete focused status/health suite passes and `TestLifecycleStatus199NoInventedEvidence` independently proves the missing-cost case.
- **Committed in:** `22449095`

---

**Total deviations:** 1 auto-fixed bug.
**Impact on plan:** The correction aligns the test with the approved compact contract and prevents recorded evidence from being discarded; no scope was added.

## Known Stubs

None. Pattern-scan matches are typed empty fixtures and ordinary control-flow initialization; no production status or health path renders mock or placeholder data.

## Verification

- Passed the exact Task 1 status/health contract: 18 focused cases covering section order, compact subset, responsive widths, visual/NO_COLOR/JSON agreement, no invented evidence, all five health states, and zero-write fixtures.
- Passed `TestLifecycleStatusWrapper199` (4 cases) and `TestCommandSourceHygiene` (7 cases); all three managed wrapper bodies are byte-identical.
- `git diff --check` passed before each task commit.
- The broader `go test ./cmd` sweep completed with 7,157 passing, 100 failing, and 10 skipped tests. The failures are the staged Phase 199 migration set: legacy status/dashboard and Next Up expectations, catalog/golden snapshots, and the already-recorded archived Plan 196 fixture. Details are recorded in `deferred-items.md`; production behavior was not weakened to satisfy retired contracts.

## Issues Encountered

- The repository-wide command suite still encodes the pre-199 dashboard and lifecycle recommendation policy. Plan-owned focused contracts pass; the remaining migrations are assigned to later Phase 199 renderer/snapshot plans and the final normal/race gate in Plan 199-29.
- No authentication gates or external services were encountered.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase and history can now consume the same identity/progress/evidence/Next Up vocabulary without building a second status truth system.
- Watch can use compact status as its honest idle snapshot while remaining distinct as the typed event stream.
- Later snapshot/golden plans must retire the legacy status expectations recorded in `deferred-items.md` before the final Plan 199-29 repository gates.

## Self-Check: PASSED

- All three created implementation/test files and this summary exist.
- RED, GREEN, and wrapper task commits resolve to committed Git objects.
- Summary/deferred-item whitespace validation passes, and no protected pre-existing path is staged.

---
*Phase: 199-front-door-and-classic-contract*
*Completed: 2026-09-03*
