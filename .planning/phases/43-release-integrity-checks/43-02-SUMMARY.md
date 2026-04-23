---
phase: 43-release-integrity-checks
plan: 02
subsystem: release-integrity
tags: [integrity, medic, version-chain, stale-publish, go-testing]

# Dependency graph
requires:
  - phase: 43-01
    provides: checkStalePublish, integrity command structure, readHubVersionAtPath
provides:
  - scanIntegrity wired into medic --deep scan
  - 14 tests covering integrity command and medic integration
  - os.Exit(2) bug fix in integrity hub-not-installed path
affects: [medic, integrity, release-diagnostics]

# Tech tracking
tech-stack:
  added: []
  patterns: [version-chain validation via checkStalePublish in medic deep scan]

key-files:
  created: [cmd/integrity_cmd_test.go]
  modified: [cmd/medic_scanner.go, cmd/integrity_cmd.go]

key-decisions:
  - "Replaced os.Exit(2) with error return in integrity hub-not-installed path for testability"
  - "TestIntegritySourceFlag expects error outside Aether repo (correct source-version-check behavior)"
  - "No duplicate companion-file counting in scanIntegrity (scanHubPublishIntegrity handles that)"

patterns-established:
  - "scanIntegrity focuses on VERSION CHAIN only; scanHubPublishIntegrity handles FILE COUNT parity"

requirements-completed: [REL-02, R063]

# Metrics
duration: 5min
completed: 2026-04-23
---

# Phase 43 Plan 02: Wire scanIntegrity into medic --deep Summary

**Version chain validation (binary vs hub agreement, stale publish detection) wired into medic --deep with 14 comprehensive tests and os.Exit bug fix**

## Performance

- **Duration:** 5 min
- **Started:** 2026-04-23T18:35:47Z
- **Completed:** 2026-04-23T18:41:22Z
- **Tasks:** 3
- **Files modified:** 3

## Accomplishments
- scanIntegrity() wired into performHealthScan deep scan path, producing integrity-category HealthIssue entries
- 14 tests covering scanIntegrity unit behavior, integrity command E2E (JSON, visual, flags, context detection), and medic deep integration
- Fixed os.Exit(2) in integrity_cmd.go that prevented testability of hub-not-installed error path

## Task Commits

Each task was committed atomically:

1. **Task 1: Add scanIntegrity to medic deep scan** - `a1e39071` (feat)
2. **Task 2: Create integrity command and medic integration tests** - `a1e39071` (feat)
3. **Task 3: Run full test suite and verify no regressions** - verified (no commit needed, all pass)

**Plan metadata:** (pending docs commit)

_Note: Tasks 1 and 2 were combined into a single commit because Task 1 code (scanIntegrity function) was already present from prior work but not wired, and the wiring, bug fix, and test adjustments were interdependent._

## Files Created/Modified
- `cmd/medic_scanner.go` - Added scanIntegrity() function and wired it into performHealthScan Deep path
- `cmd/integrity_cmd.go` - Replaced os.Exit(2) with error return for testability
- `cmd/integrity_cmd_test.go` - Created with 14 test functions: 4 scanIntegrity unit tests, 8 integrity command E2E tests, 2 medic deep integration tests

## Decisions Made
- Replaced os.Exit(2) with proper error return in integrity_cmd.go hub-not-installed path. This is consistent with all other commands in the codebase (RunE returns error) and makes the code testable.
- TestIntegritySourceFlag was adjusted to expect an error when --source is used outside the Aether repo, since checkSourceVersion fails without .aether/version.json.
- scanIntegrity deliberately does NOT count companion files -- that is scanHubPublishIntegrity's responsibility. This avoids duplicate issues (Pitfall 4 from RESEARCH.md).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed os.Exit(2) preventing testability**
- **Found during:** Task 2 (TestIntegrityExitCodeFail)
- **Issue:** integrity_cmd.go called os.Exit(2) when hub not installed, killing the test process before assertions could run
- **Fix:** Replaced os.Exit(2) with `return fmt.Errorf(...)` matching the RunE pattern used by all other commands. Both JSON and visual output paths now render their error output before returning the error.
- **Files modified:** cmd/integrity_cmd.go
- **Verification:** TestIntegrityExitCodeFail now passes
- **Committed in:** a1e39071

**2. [Rule 1 - Bug] Fixed TestIntegritySourceFlag expecting success incorrectly**
- **Found during:** Task 2 (test execution)
- **Issue:** Test expected no error from `integrity --json --source` when run from a temp dir, but --source forces checkSourceVersion which fails without .aether/version.json
- **Fix:** Changed test to capture the error and assert it is non-nil (correct behavior)
- **Files modified:** cmd/integrity_cmd_test.go
- **Verification:** TestIntegritySourceFlag now passes
- **Committed in:** a1e39071

**3. [Rule 1 - Bug] Fixed TestIntegrityVisualOutput checking wrong banner text**
- **Found during:** Task 2 (test execution)
- **Issue:** Test checked for "Release Integrity" but actual banner uses spaced "R E L E A S E   I N T E G R I T Y"
- **Fix:** Updated assertion to match actual banner rendering
- **Files modified:** cmd/integrity_cmd_test.go
- **Verification:** TestIntegrityVisualOutput now passes
- **Committed in:** a1e39071

---

**Total deviations:** 3 auto-fixed (all Rule 1 - bugs)
**Impact on plan:** All auto-fixes necessary for correctness. No scope creep.

## Issues Encountered
None beyond the deviations documented above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- REL-02 (R063) verification gap is fully closed
- All 14 integrity tests pass; no regressions in medic, E2E, or stale-publish test suites
- Ready for next phase in release-integrity-checks milestone

---
*Phase: 43-release-integrity-checks*
*Completed: 2026-04-23*

## Self-Check: PASSED

- FOUND: cmd/medic_scanner.go
- FOUND: cmd/integrity_cmd_test.go
- FOUND: cmd/integrity_cmd.go
- FOUND: 43-02-SUMMARY.md
- FOUND: a1e39071 (implementation commit)
- FOUND: 3b0a293d (docs commit)
- go test ./... -race exits 0
- go vet ./... exits 0
- go build ./cmd/aether succeeds
