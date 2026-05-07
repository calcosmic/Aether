---
phase: 100-command-inventory-lifecycle-contracts
plan: 01
subsystem: cli-testing
tags: [cobra, golden-test, audit, catalog, ci-regression]

# Dependency graph
requires: []
provides:
  - "audit-catalog CLI command producing structured JSON of all registered Cobra commands"
  - "Golden test freezing catalog output for CI drift detection"
  - "Completeness test asserting 16 lifecycle commands and >= 300 entries"
  - "Schema test validating required fields on every catalog entry"
affects: [100-02, 100-03, 100-04, 100-05]

# Tech tracking
tech-stack:
  added: []
  patterns: [golden-test-tdd, cobra-tree-walk, catalog-entry-struct]

key-files:
  created:
    - cmd/audit_catalog.go
    - cmd/audit_catalog_test.go
    - cmd/testdata/command_catalog.json
  modified:
    - cmd/root.go

key-decisions:
  - "Filter Cobra auto-generated --help flag from catalog for deterministic golden output"
  - "Use actual Cobra names (resume-colony, patrol-check, profile-read) instead of wrapper names (resume, patrol, profile) in completeness test"

patterns-established:
  - "Golden test pattern: buildAuditCatalog(rootCmd) in-process, -update-golden flag for refresh"
  - "Catalog entry schema: name, short_description, flags, parent_command, has_subcommands, output_mode"

requirements-completed: [LIFE-02]

# Metrics
duration: 28min
completed: 2026-05-07
---

# Phase 100 Plan 01: Command Catalog Summary

**audit-catalog CLI command walks full Cobra tree to produce structured JSON catalog of all 377 registered commands, with golden test for CI drift detection**

## Performance

- **Duration:** 28 min
- **Started:** 2026-05-07T15:42:41Z
- **Completed:** 2026-05-07T16:10:41Z
- **Tasks:** 1 (TDD: RED/GREEN/REFACTOR)
- **Files modified:** 4

## Accomplishments
- `aether audit-catalog --json` produces complete JSON catalog of all 377 Cobra commands
- Golden test freezes catalog output so CI catches any command addition or removal
- Completeness test verifies all 16 lifecycle commands and >= 300 entries
- Schema test validates required fields (name, short_description, flags, parent_command, has_subcommands, output_mode) on every entry
- Full cmd test suite passes (0 failures) including new golden test

## Task Commits

TDD task committed in gate sequence:

1. **RED gate: Failing tests** - `8cb9d1cd` (test)
2. **GREEN gate: Implementation + golden file** - `14c31a7d` (feat)
3. **skipStoreInit registration** - `6f1396c8` (chore)
4. **REFACTOR gate: Filter help flag** - `308950b9` (refactor)

## Files Created/Modified
- `cmd/audit_catalog.go` - CatalogEntry struct, buildAuditCatalog, walkCommands, auditCatalogCmd, renderAuditCatalogVisual
- `cmd/audit_catalog_test.go` - TestAuditCatalogGolden, TestCatalogCompleteness, TestCatalogSchema
- `cmd/testdata/command_catalog.json` - Frozen golden snapshot of 377 command catalog entries
- `cmd/root.go` - Added "audit-catalog" to skipStoreInit

## Decisions Made
- Filtered Cobra auto-generated `--help` flag from catalog output because it is added dynamically during `rootCmd.Execute()` and causes non-deterministic golden test results depending on test execution order
- Used actual Cobra command names (resume-colony, patrol-check, profile-read) in completeness test instead of wrapper names (resume, patrol, profile) since the catalog walks the runtime Cobra tree, not the wrapper layer

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed nil vs empty slice for Flags field**
- **Found during:** Task 1 (GREEN phase)
- **Issue:** Commands with no flags produced `null` in JSON instead of `[]`
- **Fix:** Changed `extractFlags` to initialize with `make([]string, 0)` instead of `var flags []string`
- **Files modified:** cmd/audit_catalog.go
- **Verification:** TestCatalogSchema passes with zero nil-flag errors
- **Committed in:** 14c31a7d (Task 1 GREEN commit)

**2. [Rule 1 - Bug] Fixed golden test trailing newline mismatch**
- **Found during:** Task 1 (GREEN phase)
- **Issue:** Golden file was written with trailing newline but compared without, causing off-by-one byte mismatch
- **Fix:** Added `"\n"` to comparison side to match the golden file write path
- **Files modified:** cmd/audit_catalog_test.go
- **Verification:** TestAuditCatalogGolden passes in isolation
- **Committed in:** 14c31a7d (Task 1 GREEN commit)

**3. [Rule 1 - Bug] Fixed golden test failure in full test suite**
- **Found during:** Task 1 (REFACTOR phase)
- **Issue:** Cobra auto-generates `--help` flag during `rootCmd.Execute()`, which persists as a local flag on subcommands. Other tests executing commands caused `--help` to appear on commands, making golden output non-deterministic.
- **Fix:** Filter `help` from `extractFlags` output since it is a standard Cobra flag, not user-defined
- **Files modified:** cmd/audit_catalog.go
- **Verification:** Full test suite passes (0 failures) with golden test included
- **Committed in:** 308950b9 (REFACTOR commit)

---

**Total deviations:** 3 auto-fixed (all Rule 1 bugs)
**Impact on plan:** All auto-fixes necessary for correct test behavior. No scope creep.

## Issues Encountered
- Plan listed 16 lifecycle command names (resume, patrol, profile) that don't exist as standalone Cobra commands; they map to compound names (resume-colony, patrol-check, profile-read). Updated completeness test to use actual Cobra names and documented the wrapper-to-runtime mapping.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- audit-catalog command ready for Phases 101-104 to consume as machine-readable foundation
- Golden file established with 377 commands as baseline; CI will catch any drift
- Catalog JSON structure stable for downstream consumers

---
*Phase: 100-command-inventory-lifecycle-contracts*
*Completed: 2026-05-07*

## Self-Check: PASSED

All files verified: cmd/audit_catalog.go, cmd/audit_catalog_test.go, cmd/testdata/command_catalog.json, SUMMARY.md
All commits verified: 8cb9d1cd (test), 14c31a7d (feat), 6f1396c8 (chore), 308950b9 (refactor)
