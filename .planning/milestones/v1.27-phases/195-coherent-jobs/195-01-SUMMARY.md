---
phase: 195-coherent-jobs
plan: 01
subsystem: runtime-planning
tags: [go, task-graph, coherent-jobs, dependency-dag, tdd]

requires: []
provides:
  - Pure coherent-job proposal and planning contracts
  - Deterministic same-caste grouping by dependencies and exact meaningful paths
  - Dependency-safe job DAG waves and brief-allowance splitting
affects: [195-02, 195-03, 195-04, 195-05, 195-06, 195-07, 195-08, 195-09, 195-10]

tech-stack:
  added: []
  patterns: [pure pre-dispatch planner, fail-closed graph preflight, deterministic union-find components, topological plan-order tie breaking]

key-files:
  created: [cmd/coherent_jobs.go, cmd/coherent_jobs_test.go]
  modified: []

key-decisions:
  - "Automatic grouping contracts the full phase graph only when the prospective same-caste component remains dependency-safe."
  - "Only exact declared implementation paths create automatic edges; ubiquitous documentation, manifests, and lockfiles are incidental."
  - "Automatic groups split against briefTaskContentAllowanceChars, while a single oversized task or accepted Queen proposal remains intact."

patterns-established:
  - "Pure planner boundary: dependency validation, proposal decisions, grouping, and job waves complete before stateful runtime work begins."
  - "Visible local repair: refuse one unsafe Queen proposal by name and return only its members to automatic planning."
  - "Structured job reasons: every grouped job records both its relationship and worker-ownership benefit."

requirements-completed: [JOBS-01, JOBS-03]

duration: 40min
completed: 2026-08-27
---

# Phase 195 Plan 01: Coherent Job Planner Summary

**Pure Go planner that turns selected tasks into dependency-safe, same-caste jobs using Queen proposals, declared relationships, and the existing worker-brief allowance.**

## Performance

- **Duration:** 40 min
- **Started:** 2026-08-27T01:44:37Z
- **Completed:** 2026-08-27T02:25:18Z
- **Tasks:** 2
- **Files modified:** 2 implementation files

## Accomplishments

- Added stable package-private proposal, task seed, job, decision, and plan contracts with whole-phase missing-dependency and cycle preflight.
- Preserved valid Queen jobs while refusing and visibly repairing only unsafe proposals, including explicit cross-caste ownership validation.
- Grouped automatic jobs deterministically through same-caste dependencies or exact meaningful paths, while retaining dependency-safe boundaries and job-DAG waves.
- Reproduced the literal six-batch CalVault file-copy shape as one ordered automatic Builder job.
- Reused `briefTaskContentAllowanceChars` for content-based automatic splitting with no magic task-count cap and no fragmentation of Queen-owned or individually oversized work.

## Task Commits

Each task was committed atomically with explicit TDD gates:

1. **Task 1 RED: proposal and graph regressions** - `a19b3758` (test)
2. **Task 1 GREEN: proposal contracts, preflight, and local repair** - `15f74c3d` (feat)
3. **Task 2 RED: automatic grouping and CalVault regressions** - `92593493` (test)
4. **Task 2 GREEN: meaningful components, waves, and brief sizing** - `d9f48fb9` (feat)

## Files Created/Modified

- `cmd/coherent_jobs.go` - Pure proposal validation, automatic grouping, topological ordering, job DAG construction, incidental-path policy, and brief-size projection.
- `cmd/coherent_jobs_test.go` - TDD coverage for D-01 through D-07, including proposal repair, graph errors, path policy, caste boundaries, soft limits, and CalVault.
- `.planning/phases/195-coherent-jobs/deferred-items.md` - Records the unrelated full-suite black-box timeout discovered during broad verification.

## Decisions Made

- Checked each prospective automatic component against the full phase graph before union so grouping cannot contract an acyclic task graph into a cyclic job graph.
- Used dependency order inside a job, with original plan order and task ID as deterministic tie breakers between independently runnable tasks.
- Classified README/changelog/history/release-note files and Go/npm/Python/Cargo/Ruby manifests and lockfiles as incidental grouping evidence.
- Reserved accepted Queen job names before generating automatic identifiers so every job name remains deterministic and unique.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Prevented automatic identifiers from colliding with accepted Queen job names**
- **Found during:** Task 2 (automatic component naming)
- **Issue:** A Queen proposal could legally choose a name such as `single-task-a`, colliding with a generated runtime job and making name-based DAG completion ambiguous.
- **Fix:** Reserved accepted job names and applied deterministic numeric suffixes to generated collisions.
- **Files modified:** `cmd/coherent_jobs.go`
- **Verification:** All proposal, grouping, and job-DAG suites pass; generated names remain stable in plan order.
- **Committed in:** `d9f48fb9`

**2. [Rule 3 - Blocking] Restored the STATE progress target required by the GSD updater**
- **Found during:** Plan closeout
- **Issue:** `state.update-progress` could not update this project's STATE format because it had no supported `Progress:` field.
- **Fix:** Added the standard progress field and reran the SDK handler successfully (`15/24`, 63%).
- **Files modified:** `.planning/STATE.md`
- **Verification:** `gsd-sdk query state.update-progress` returned `updated: true`.
- **Committed in:** Plan metadata commit

---

**Total deviations:** 2 auto-fixed (1 Rule 1 bug, 1 Rule 3 blocker)
**Impact on plan:** Both fixes preserve deterministic job planning and required GSD bookkeeping without expanding runtime scope.

## Issues Encountered

- Both TDD RED gates failed for the intended missing contracts before their GREEN implementations.
- `go test ./...` completed 1,520 passing tests and 1 skipped test, then the unrelated `cmd` black-box harness reached Go's 10-minute package timeout. The last started `cmd` test was `TestCLIErrorEnvelopeExitsNonZero`; this plan does not modify that harness or command path. The finding is tracked in `deferred-items.md`.
- Plan verification remained green: 47 coherent-job tests pass normally and under `-race`; `go build ./cmd/aether` and `go vet ./...` pass.

## Known Stubs

None - no placeholder or unwired data paths were introduced.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plans 195-02 onward can consume one authoritative pure planner contract before manifests, attempts, briefs, or worktrees exist.
- The full-suite black-box timeout is unrelated to coherent-job planning but should be investigated before treating the repository-wide 10-minute gate as reliable.

## Self-Check: PASSED

- Both implementation files and both closeout artifacts exist.
- All four Task 1/Task 2 TDD commits are present in git history.

---
*Phase: 195-coherent-jobs*
*Completed: 2026-08-27*
