---
phase: 200-iterative-planning
plan: 27
subsystem: storage-security
tags: [go, cobra, openat, nofollow, ntcreatefile, file-locking, tdd]

requires:
  - phase: 200-26
    provides: Durable Phase 199 evidence and protected-owner-file baselines
provides:
  - containment-first Cobra repository bootstrap with no pre-validation storage side effects
  - open physical repository authority for root-relative no-follow data and lock operations
  - full normalized-path lock digests that separate equal basenames
  - actual Cobra subprocess proof for hostile roots, links, swaps, and read-only startup
affects: [phase-200-verification, cobra-root, repository-storage, file-locking, cross-platform-filesystem-safety]

tech-stack:
  added: []
  patterns: [directory-handle capability, component-by-component no-follow open, full-path lock digest, subprocess filesystem snapshot]

key-files:
  created:
    - cmd/root_repository_bootstrap_200_test.go
  modified:
    - cmd/root.go
    - pkg/storage/storage.go
    - pkg/storage/lock.go
    - pkg/storage/lock_unix.go
    - pkg/storage/lock_windows.go

key-decisions:
  - "Repository storage keeps an open physical root directory handle and performs every data and lock open relative to that authority."
  - "Store BasePath preserves its established caller-visible spelling, while filesystem authority comes only from the canonical root handle."
  - "Lock filenames digest the complete normalized logical data path instead of filepath.Base."

patterns-established:
  - "Containment-first bootstrap: validate repository, data, and locks before status probes, store construction, tracer setup, or first-run output."
  - "Swap-resistant repository I/O: reopen each component without following links and retain the verified root handle across validation and creation."

requirements-completed: [CEC-03, PLAN-06]

duration: 22 min
completed: 2026-09-08
---

# Phase 200 Plan 27: Repository-Bound Storage Bootstrap Summary

**Cobra storage now opens data and collision-resistant locks through one verified physical repository authority, refusing hostile roots and component swaps without filesystem mutation.**

## Performance

- **Duration:** 22 min
- **Started:** 2026-09-08T20:19:22Z
- **Completed:** 2026-09-08T20:41:35Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments

- Added real Cobra subprocess attacks for outside and adjacent roots, missing parents, intermediate `.aether`/`data`/`locks` links, a deterministic post-validation swap, degenerate roots, lock basename collisions, and allowed/read-only first invocation behavior.
- Added a repository-aware store authority that physically resolves the repository once, walks existing components without following links, and creates or opens data only relative to the retained root handle.
- Bound repository lock creation to the same verified authority and replaced basename-only lock identity with a SHA-256 digest of the complete normalized logical data path.
- Moved root initialization order so containment succeeds before status probing, store construction, tracer creation, or first-run marker output.
- Implemented equivalent root-relative no-reparse behavior for Windows and verified both Windows and Linux storage builds from the Darwin development host.

## Task Commits

1. **Task 1 RED: Define bootstrap refusal through actual Cobra subprocesses** — `e69eb7ff` (test)
2. **Task 2 GREEN: Open the store and locks only through verified roots** — `40e3aad3` (feat)

## Files Created/Modified

- `cmd/root_repository_bootstrap_200_test.go` — Executes the real Cobra root in isolated child processes and snapshots both repository and outside trees around hostile invocations.
- `cmd/root.go` — Resolves and validates the repository authority before any storage-side effect, preserves read-only first status, and routes first-run probes through the verified store.
- `pkg/storage/storage.go` — Defines the retained repository authority, no-follow component walk, root-relative store operations, and zero-create status probe.
- `pkg/storage/lock.go` — Shares the repository authority with locks and derives lock identity from the full normalized data path.
- `pkg/storage/lock_unix.go` — Uses `openat`, `mkdirat`, `renameat`, and `unlinkat` with no-follow directory/file flags.
- `pkg/storage/lock_windows.go` — Uses root-relative NT opens with reparse-point refusal plus relative rename and delete operations.

## Decisions Made

- A directory handle, rather than a validated path string, is the repository capability retained across validation and later I/O; this closes directory-to-symlink swap windows.
- Public `BasePath()` keeps the lexical spelling callers already observe (including macOS `/var` aliases), but repository-backed operations never treat that string as authority.
- Empty `COLONY_DATA_DIR` is distinguished at process startup so a genuinely explicit CLI value is refused while repeat in-process Cobra executions preserve the historical unset-variable contract.
- Repository locks use the same retained authority as data files; a digest of the normalized full logical path prevents unrelated equal basenames from sharing a lock.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Accepted equivalent macOS physical path spellings without weakening component checks**
- **Found during:** Task 2 focused verification
- **Issue:** macOS can expose one temporary repository through both `/var/...` and `/private/var/...`; a purely lexical comparison rejected the same physical repository.
- **Fix:** Added a canonical-root component comparison for an already-physical candidate spelling while retaining no-follow traversal for every component below the repository.
- **Files modified:** `pkg/storage/storage.go`, `cmd/root.go`
- **Verification:** The previously failing stored-handoff print-brief regression and the full 67-test focused suite pass.
- **Committed in:** `40e3aad3`

**2. [Rule 3 - Blocking] Kept subprocess harness setup outside mutation snapshots**
- **Found during:** Task 2 first GREEN run
- **Issue:** The command package's `TestMain` creates an isolated test hub under `TMPDIR`, which made a rejected command appear to mutate the hostile-root fixture even though repository storage had not run.
- **Fix:** Gave child-process runtime scratch its own temporary root outside the repository/outside-tree snapshot boundary and asserted the expected no-colony result for read-only status independently of its established non-zero status code.
- **Files modified:** `cmd/root_repository_bootstrap_200_test.go`
- **Verification:** Empty, dot, dot-dot, and first read-only status cases all pass with byte-identical scoped snapshots.
- **Committed in:** `40e3aad3`

---

**Total deviations:** 2 auto-fixed (1 bug, 1 blocking issue)
**Impact on plan:** Both fixes make the planned proof accurate across the existing test harness and macOS path aliases; no files or behavior outside the declared repository-bootstrap scope were added.

## Issues Encountered

- The RED suite initially failed at compile time because the deterministic validation-barrier hook did not exist, establishing the required fail-first gate before implementation.
- The first GREEN pass exposed repeat-execution environment cleanup and macOS lexical/physical path compatibility; the production process-start contract still rejects an explicitly empty data root.
- The known unrelated repository-wide `go test ./...` timeout was not rerun. Plan 27's focused, race, and platform compilation gates all completed without assertion failures.

## TDD Gate Compliance

- **RED:** `e69eb7ff` introduced the actual Cobra subprocess suite; `go test ./cmd -run '^TestRepositoryBootstrapContainment200' -count=1` failed because no bootstrap validation hook/API existed.
- **GREEN:** `40e3aad3` implemented the root authority, store, locks, and platform primitives; the exact focused gate passed 67 tests on two consecutive committed-tree runs.
- **REFACTOR:** No separate behavior-neutral refactor commit was needed.

## Verification

- `go test ./pkg/storage ./cmd -run '^(TestRepositoryBootstrapContainment200|Test.*Store.*|Test.*FileLock.*)' -count=1` — 67 passed, repeated twice from the committed tree.
- The same focused command with `-race` — 67 passed.
- `go test ./pkg/storage -count=1` — 45 passed.
- `go vet ./pkg/storage` — passed.
- `GOOS=windows GOARCH=amd64 go test -c ./pkg/storage` and the equivalent Linux build — passed.
- `git diff --check 7d6c83ed..HEAD` — passed; the implementation range owns exactly the six plan-declared files.
- Protected-file SHA-256 checks — `.planning/config.json`, all `.gsd` bytes, and untracked Phase 199 `199-PATTERNS.md` remain byte-identical to their starting state.

## Known Stubs

None.

## User Setup Required

None - no dependency, credential, service, or local configuration change is required.

## Next Phase Readiness

- Repository-scoped Cobra storage is ready for the remaining Phase 200 verification plans with physical containment established before first I/O.
- The known repository-wide command-package timeout remains an unrelated integration-runner condition; Plan 27's focused acceptance surface is green.

## Self-Check: PASSED

- The summary and all six scoped implementation/test files exist.
- Task commits `e69eb7ff` and `40e3aad3` are present in repository history.
- The committed implementation range contains exactly the six plan-owned paths and passes whitespace validation.
- Focused tests, race tests, cross-platform compilation, stub scan, and protected-owner-file fingerprint checks all pass.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-08*
