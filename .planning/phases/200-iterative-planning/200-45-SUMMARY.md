---
phase: 200-iterative-planning
plan: 45
subsystem: storage
tags: [go, filesystem, errno, repository-containment, append-only-ledgers, cross-platform]

# Dependency graph
requires:
  - phase: 200-27
    provides: physical repository authority and root-relative no-follow storage
  - phase: 200-44
    provides: repository-bound command fixtures with exact authority cleanup
provides:
  - portable ErrNotExist identity across repository-backed storage wrappers
  - fresh repository-backed spawn and run ledger initialization
  - fail-closed proof for corrupt, linked, unreadable, and non-regular ledgers
affects: [200-40, final-gate, optional-ledgers, repository-storage, windows]

# Tech tracking
tech-stack:
  added: []
  patterns: [platform-boundary errno normalization, errors.Is wrapper traversal, corruption fingerprint proof]

key-files:
  created: []
  modified:
    - pkg/storage/storage.go
    - pkg/storage/lock_unix.go
    - pkg/storage/lock_windows.go
    - pkg/storage/storage_malformed_test.go
    - pkg/agent/spawn_tree_test.go

key-decisions:
  - "Only genuine platform ENOENT results normalize to os.ErrNotExist; every other repository-open error retains its original identity and fails closed."
  - "Repository UpdateFile uses errors.Is rather than os.IsNotExist so absence survives every contextual wrapper."
  - "Repository ledger tests fingerprint hostile bytes and prove refused writes never replace corrupt files or follow links."

patterns-established:
  - "Portable optional-file contract: absent is empty, present-but-unreadable is an error, and present-but-corrupt is a distinct error."
  - "Cross-platform repository adapters normalize absence once at the low-level open boundary."

requirements-completed: [CEC-03, PLAN-06]

# Metrics
duration: 10min
completed: 2026-09-09
---

# Phase 200 Plan 45: Portable Optional-Ledger Error Contract Summary

**Repository-backed optional ledgers now recognize a genuinely missing file on Unix and Windows while continuing to reject links, directories, unreadable nodes, and corrupt content without overwriting evidence.**

## Performance

- **Duration:** 10 min
- **Started:** 2026-09-09T10:06:36Z
- **Completed:** 2026-09-09T10:16:49Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments

- Normalized only genuine low-level missing-file errors to portable `os.ErrNotExist` identity in both Unix and Windows repository adapters.
- Replaced the legacy `os.IsNotExist` predicate at the repository read-modify-write boundary with wrapper-aware `errors.Is`, restoring first-write behavior for optional ledgers.
- Proved first `RecordSpawn` and `BeginRun` writes plus replay through a physical temporary repository authority.
- Proved corrupt, linked, unreadable, non-regular, and malformed inputs remain non-absence failures, with corrupt and linked bytes fingerprinted before and after refused mutations.
- Re-ran the eight command-level colonize, completion-packet, learning, and instinct consumers that previously failed in isolation.

## Task Commits

Each task was committed atomically:

1. **Task 1 RED: Define repository absence identity and refusal contract** - `7227fa2e` (test)
2. **Task 1 GREEN: Normalize portable repository absence identity** - `fc314839` (fix)
3. **Task 2: Prove fresh and hostile repository ledger behavior** - `99d49140` (test)

## Files Created/Modified

- `pkg/storage/storage.go` - Uses wrapper-aware absence detection for repository read-modify-write operations.
- `pkg/storage/lock_unix.go` - Normalizes Unix ENOENT at repository open boundaries without changing other errno values.
- `pkg/storage/lock_windows.go` - Normalizes CreateFile/NtCreateFile missing-path errors while retaining reparse, directory, and permission failures.
- `pkg/storage/storage_malformed_test.go` - Exercises portable absence identity through every read wrapper and keeps hostile nodes distinct.
- `pkg/agent/spawn_tree_test.go` - Proves fresh spawn/run creation, replay, corruption fingerprints, and linked-ledger refusal through repository authority.

## Decisions Made

- Kept normalization at the platform open boundary so every higher-level `%w` wrapper preserves the same portable identity without error-string matching.
- Converted only actual missing-file/path results; symlink, directory, permission, closed-authority, and malformed-content errors remain untouched.
- Retained full logical/full-path context in higher storage wrappers while using the portable sentinel only as the causal identity.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- Task 1 RED showed that direct repository reads already exposed `errors.Is(..., fs.ErrNotExist)` on Darwin, but the first `UpdateFile` still failed because legacy `os.IsNotExist` does not traverse the repository's nested contextual wrappers. The planned boundary normalization plus wrapper-aware predicate resolved that exact break.
- Task 2's new downstream ledger tests passed on their first run because Task 1 had already repaired the shared storage primitive. The tests were retained as repository-backed regression proof; no second implementation path was added.

## TDD Gate Compliance

- **RED:** `7227fa2e` added the focused storage contract; its first run failed at fresh `UpdateFile` initialization.
- **GREEN:** `fc314839` made the focused storage contract and Windows compile-only check pass.
- **Downstream proof:** `99d49140` added ledger-level regression coverage after the shared Task 1 fix; it required no additional production code.
- **REFACTOR:** No behavior-neutral refactor was needed.

## Verification

- `go test ./pkg/storage -run '^TestRepository(StoreMissingErrorIdentity200|StoreRefusesNonAbsentFailures200)$' -count=1` - PASS.
- `GOOS=windows GOARCH=amd64 go test -exec=/usr/bin/true ./pkg/storage -run '^$' -count=1` - PASS.
- `go test ./pkg/agent -run '^TestSpawnTreeRepository(StoreFresh|StoreCorrupt|RunLedgerFresh)200$' -count=1` - PASS.
- `go test ./cmd -run '^(TestApplicationHistoryShapeMatchesInstinctApply|TestLearningPropose|TestColonizeFallsBackWhenRealSurveyorsFail|TestColonizeFinalizeRecordsExternalSurveyors|TestColonizeIncludesDispatchContract|TestColonizePreservesWorkerWrittenSurveyArtifacts|TestColonizeWritesSurveyArtifactsAndUpdatesState|TestCompletionPacketSubmittedBytes)$' -count=1` - PASS.
- `go test ./pkg/storage ./pkg/agent -count=1` - PASS.
- `git diff --check 7227fa2e^..99d49140` - PASS.

## Known Stubs

None.

## Threat Flags

None. The change narrows error classification at an existing file-open boundary and introduces no new endpoint, schema, credential path, or filesystem authority.

## User Setup Required

None - no dependency, credential, service, or local configuration change is required.

## Next Phase Readiness

- Fresh repository-backed optional ledgers are available to the remaining Phase 200 fixture and integration repairs.
- Plan 46 can proceed; the unchanged final Plan 40 receipt remains deferred until every later repair and both full gates pass.

## Self-Check: PASSED

- All five modified implementation/test files and this summary exist.
- Task commits `7227fa2e`, `fc314839`, and `99d49140` exist in repository history.
- Focused storage, agent, command-consumer, full owned-package, Windows compile-only, and whitespace gates all pass.
- `.planning/config.json`, `.gsd/`, and untracked Phase 199 `199-PATTERNS.md` were neither staged nor changed by Plan 45.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-09*
