---
phase: 199-front-door-and-classic-contract
plan: "34"
subsystem: lifecycle recovery compatibility
tags: [go, cobra, lifecycle, recovery, maintenance, compatibility]
requires:
  - phase: 199-13
    provides: expiring pre-Cobra lifecycle token redirect
  - phase: 199-17
    provides: hidden zero-write legacy recovery migration route
  - phase: 199-18
    provides: maintenance preview, transaction, rollback, and receipt engine
  - phase: 199-32
    provides: runtime recovery route fixture collection
provides:
  - one expiring compatibility boundary for legacy persisted and CLI lifecycle tokens
  - maintenance inspection and canonical resume-only recovery guidance
  - parity evidence for read-only diagnosis followed by lifecycle restoration
affects: [Phase 199 recovery contract, lifecycle state, maintenance, worktree preservation]
tech-stack:
  added: []
  patterns: [exact-token input migration, canonical downstream command IDs, AST-backed public vocabulary ratchet]
key-files:
  created: [cmd/runtime_recovery_compat_199_test.go]
  modified: [cmd/normalize_args.go, cmd/hook_cmds.go, cmd/recovery_engine.go, cmd/recover_visuals.go, cmd/recover_repair.go, cmd/worktree.go, .aether/docs/PARITY_CLASSIC_VS_GO.md]
key-decisions:
  - "resume-colony is accepted only as an exact, expiring input token and becomes resume at the shared compatibility boundary."
  - "Recovery diagnosis is read-only maintenance inspection; aether resume is the only lifecycle-restoration route."
requirements-completed: [SYNTH-01, CEC-02, CEC-04, LIFE-03, LIFE-04, LIFE-06, PROOF-01]
duration: 6 min
completed: 2026-09-04
---

# Phase 199 Plan 34: Runtime Recovery Compatibility Summary

**Canonical persisted resume state and read-only maintenance diagnosis remove the retired recover/apply recovery door without losing worktree evidence.**

## Performance

- **Duration:** 6 min
- **Started:** 2026-09-04T23:01:51Z
- **Completed:** 2026-09-04T23:07:48Z
- **Tasks:** 2/2
- **Files modified:** 8

## Accomplishments

- Unified legacy CLI and persisted session command migration at the existing 1.29 compatibility boundary, so hooks and recovery filtering consume canonical `resume`.
- Replaced active apply/force repair recommendations with exact read-only `aether maintenance recovery-inspect` guidance and `aether resume` as the only restoration command.
- Added one execution test covering persisted compatibility, canonical writes, visual route guidance, source literals, and parity documentation.

## Task Commits

1. **Task 1: Normalize persisted legacy command state at one boundary** — `170d70fd` (RED test), `4de6dfed` (implementation)
2. **Task 2: Retire apply suggestions and route useful recovery through maintenance and resume** — `dc9a6923` (RED test), `2d94e878` (implementation)

## Files Created/Modified

- `cmd/normalize_args.go` — sole exact-token, 1.29-expiring legacy command normalizer.
- `cmd/hook_cmds.go` — canonicalizes loaded hook session state and all new session writes.
- `cmd/recovery_engine.go` — filters candidates through the shared normalizer.
- `cmd/recover_visuals.go` — presents maintenance inspection and resume instead of retired apply routes.
- `cmd/recover_repair.go` — documents that the legacy internal helper has no command entry point; current mutation is maintenance-owned.
- `cmd/worktree.go` — preserves worktree evidence while naming maintenance inspection.
- `.aether/docs/PARITY_CLASSIC_VS_GO.md` — proves inspection state-effect-none plus resume restoration.
- `cmd/runtime_recovery_compat_199_test.go` — focused lifecycle recovery contract ratchet.

## Decisions Made

- Exact old command tokens are migration input only; partial or unknown spellings are never rewritten.
- The repair scanner may describe evidence, but it cannot recommend or reopen a second mutating lifecycle entry point.

## Verification

- PASS: `go test ./cmd -run '^TestRuntimeRecoveryCompatibility199$' -count=1` — 9 tests passed.
- PASS: scoped token gate across the parity guide and six audited runtime paths.
- PASS: focused route, legacy-zero-write, maintenance mutation, and worktree preservation checks — 25 tests passed.
- PASS: `go build ./...`.
- PASS: focused `go test -race ./cmd` recovery/maintenance/worktree suite — 25 tests passed.
- Mapped expected failures: the broader legacy recovery suite has 10 tests asserting the retired `recover --apply`/force contract. They were not changed because they are outside this plan's owned files; their failures reflect this plan's intentional public-contract retirement.

## Deviations from Plan

None - plan executed exactly as written.

## Known Stubs

- `cmd/recover_repair.go:811` retains a pre-existing informational placeholder for `build-reconcile` inside an unreachable legacy internal repair helper. It is not a public route and does not affect the maintenance/resume contract; a future owner of build reconciliation should replace it.

## Issues Encountered

- Existing non-owned recovery tests still expect the retired apply/force commands. The focused current-contract tests, source gate, build, and race checks pass.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Recovery vocabulary is canonical across persisted state, renderer guidance, worktree preservation, and parity evidence.
- The next Phase 199 plan can reuse `TestRuntimeRecoveryCompatibility199` and `collectRuntimeRecoverySuggestions199` as the executable recovery-contract anchors.

## Self-Check: PASSED

All eight owned artifacts exist and all four TDD/task commits are present in git history.

---
*Phase: 199-front-door-and-classic-contract*
*Completed: 2026-09-04*
