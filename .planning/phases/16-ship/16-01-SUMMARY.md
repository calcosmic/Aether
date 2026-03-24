---
phase: 16-ship
plan: 01
subsystem: packaging
tags: [npm, npmignore, validation, changelog, release]

# Dependency graph
requires:
  - phase: 13-monolith-modularization
    provides: 9 domain modules extracted to utils/
provides:
  - Clean .npmignore excluding colony-specific data files
  - Updated validate-package.sh with all domain modules and skills files
  - Content-aware checks for CROWNED-ANTHILL.md and exchange XML leaks
  - CHANGELOG header set to [2.1.0] release version
affects: [16-02-PLAN]

# Tech tracking
tech-stack:
  added: []
  patterns: [content-aware package validation, subdirectory .npmignore for npm-packlist]

key-files:
  created: []
  modified:
    - .aether/.npmignore
    - bin/validate-package.sh
    - CHANGELOG.md

key-decisions:
  - "Broad grep for CROWNED/xml/midden matches legitimate files (templates, schemas, utils) -- only actual colony data paths are excluded"
  - "midden/ added to .npmignore explicitly even though data/ covers data/midden/ -- CROWNED-ANTHILL.md and exchange/*.xml are outside data/"

patterns-established:
  - "Content-aware package checks: validate-package.sh checks both file presence AND absence of leaked files"

requirements-completed: [UX-06]

# Metrics
duration: 2min
completed: 2026-03-24
---

# Phase 16 Plan 01: Package Hygiene Summary

**Clean npm package: colony data excluded via .npmignore, 15 new required files validated, 2 content-aware leak checks added**

## Performance

- **Duration:** 2 min
- **Started:** 2026-03-24T14:12:35Z
- **Completed:** 2026-03-24T14:14:51Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- Package no longer leaks CROWNED-ANTHILL.md, exchange XML data, or midden logs
- validate-package.sh now validates all 9 domain modules, skills.sh, hive.sh, midden.sh, and 3 skills manifests
- CHANGELOG header updated from [2.1.0-rc] to [2.1.0] release
- Package verified clean: 304 files, no colony data in dry-run output

## Task Commits

Each task was committed atomically:

1. **Task 1: Fix .npmignore and update validate-package.sh** - `c41bfa5` (feat)
2. **Task 2: Update CHANGELOG header and verify dry-run** - `2ee0900` (chore)

## Files Created/Modified
- `.aether/.npmignore` - Added exclusions for CROWNED-ANTHILL.md, midden/, exchange/*.xml
- `bin/validate-package.sh` - Added 15 entries to REQUIRED_FILES, added Check 5 (CROWNED-ANTHILL) and Check 6 (exchange XML)
- `CHANGELOG.md` - Changed [2.1.0-rc] to [2.1.0]

## Decisions Made
- Broad grep for verification matches legitimate files (templates, schemas, utils) -- the specific colony data paths are all properly excluded
- midden/ added to .npmignore explicitly for safety even though data/midden/ is already covered by data/ exclusion

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Package is clean and validated, ready for install walkthrough testing (16-02)
- validate-package.sh exits 0 with all content checks passing
- npm pack --dry-run shows 304 files with no colony data leaks

## Self-Check: PASSED

All artifacts verified:
- 16-01-SUMMARY.md: FOUND
- Commit c41bfa5 (Task 1): FOUND
- Commit 2ee0900 (Task 2): FOUND
- .aether/.npmignore: FOUND
- bin/validate-package.sh: FOUND

---
*Phase: 16-ship*
*Completed: 2026-03-24*
