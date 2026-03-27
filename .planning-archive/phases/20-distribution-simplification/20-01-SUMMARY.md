---
phase: 20-distribution-simplification
plan: 01
subsystem: infra
tags: [npm, packaging, distribution, cli]

# Dependency graph
requires: []
provides:
  - Direct .aether/ packaging pipeline (no runtime/ staging)
  - bin/validate-package.sh pre-publish validation with dry-run mode
  - Exclude-based sync replaces SYSTEM_FILES allowlists in cli.js and update-transaction.js
  - v4.0.0 package version
affects:
  - 20-02 (bash tests update)
  - 20-03 (documentation update)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Exclude-based file sync: walk directory and exclude private dirs (replaces allowlist)"
    - "Subdirectory .npmignore for npm-packlist subdirectory walker (npm 11.x behavior)"
    - "Pre-publish validation script pattern: required-files check + private-data guard"

key-files:
  created:
    - bin/validate-package.sh
    - .aether/.npmignore
  modified:
    - package.json
    - .npmignore
    - .gitignore
    - bin/cli.js
    - bin/lib/update-transaction.js
    - .aether/data/checkpoint-allowlist.json
  deleted:
    - bin/sync-to-runtime.sh
    - runtime/ (entire directory)

key-decisions:
  - "Root .npmignore is not applied by npm 11.x when files field is present — create .aether/.npmignore instead (subdirectory walker picks it up)"
  - "Migration message added for pre-4.0 upgrades — reads prevManifest version before main system sync"
  - "HUB_EXCLUDE_DIRS gets rules added — rules/ is synced by a dedicated step, not the main .aether/ walk"
  - "EXCLUDE_DIRS in update-transaction gets archive and chambers — these are private dirs that must not sync to target repos"

patterns-established:
  - "Subdirectory .npmignore pattern: when files[] includes a directory, put .npmignore inside that dir for npm-packlist subdirectory walker"
  - "Validation-first publish: prepublishOnly runs validate-package.sh which blocks on missing required files or unguarded private dirs"

requirements-completed:
  - PIPE-01
  - PIPE-02

# Metrics
duration: 11min
completed: 2026-02-19
---

# Phase 20 Plan 01: Distribution Simplification Summary

**npm package now reads directly from .aether/ with exclude-based private-dir guarding — runtime/ staging and SYSTEM_FILES allowlists fully removed**

## Performance

- **Duration:** ~11 min
- **Started:** 2026-02-19T20:06:26Z
- **Completed:** 2026-02-19T20:17:37Z
- **Tasks:** 2
- **Files modified:** 8 (plus 2 created, 2 deleted)

## Accomplishments
- Eliminated runtime/ staging directory and sync-to-runtime.sh — 2-step flow (edit -> validate + package)
- Removed SYSTEM_FILES allowlists from cli.js (~60 lines) and update-transaction.js (~62 lines)
- Created bin/validate-package.sh: required-file existence checks, private-data guard, --dry-run mode
- Updated setupHub() and update rules sync to read from .aether/ directly
- Added migration message for users upgrading from v3.x
- npm pack --dry-run confirms: 83 .aether/ files included, 0 runtime/ files, 0 private dir files

## Task Commits

Each task was committed atomically:

1. **Task 1: Restructure npm packaging and create validation script** - `e074752` (feat)
2. **Task 2: Update distribution code, delete runtime/ and sync script** - `8d25dcb` (feat)

## Files Created/Modified
- `bin/validate-package.sh` - Pre-publish validation: required files check, private-data guard, --dry-run
- `.aether/.npmignore` - Subdirectory-level ignore for npm-packlist (npm 11.x behavior requires this)
- `package.json` - Version 4.0.0, files[] uses .aether/ not runtime/, validate-package.sh in scripts
- `.npmignore` - Removed blanket .aether/ exclusion, added specific private dir/file exclusions
- `.gitignore` - Removed runtime/ entry (no longer a build artifact)
- `bin/cli.js` - Deleted SYSTEM_FILES array + syncSystemFilesWithCleanup, setupHub reads .aether/, rules source updated, migration message added, HUB_EXCLUDE_DIRS gets rules
- `bin/lib/update-transaction.js` - Deleted SYSTEM_FILES array + syncSystemFilesWithCleanup, EXCLUDE_DIRS gets archive + chambers
- `.aether/data/checkpoint-allowlist.json` - Removed runtime/**/* entry
- `bin/sync-to-runtime.sh` - Deleted
- `runtime/` - Deleted (was gitignored build artifact)

## Decisions Made
- Root .npmignore is bypassed by npm 11.x when `files` field is present (package.json ignoreRules override). Fix: create `.aether/.npmignore` — subdirectory walkers in npm-packlist DO read it.
- `rules` added to HUB_EXCLUDE_DIRS because the main aetherSrc walk would include rules/ (conflicting with the dedicated rulesSrc sync step).
- `archive` and `chambers` added to EXCLUDE_DIRS in update-transaction.js — these are private dirs that should never sync to target repos via `aether update`.
- Migration message uses prevManifest before system sync (reads old manifest before overwriting it).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] npm 11.x root .npmignore bypass when `files` field is present**
- **Found during:** Task 1 (Restructure npm packaging)
- **Issue:** Plan specified root `.npmignore` for private dir exclusions. npm 11.x (npm-packlist 10.0.3) ignores root `.npmignore` when `files` field in package.json is present. Verified via npm-packlist source: `filterEntries()` sets ignoreRules['.npmignore'] = null when package.json rules take precedence. Private dirs like chambers/, dreams/, archive/ were being included despite .npmignore entries.
- **Fix:** Created `.aether/.npmignore` with private dir exclusions. Subdirectory walkers in npm-packlist DO read it (confirmed in walkerOpt method). Also updated validate-package.sh to check `.aether/.npmignore` instead of root `.npmignore`.
- **Files modified:** `.aether/.npmignore` (created), `bin/validate-package.sh` (updated check), `.npmignore` (kept for documentation but not the effective exclusion file)
- **Verification:** `npm pack --dry-run` shows 0 matches for chambers/, dreams/, archive/, oracle/, data/, __pycache__/. 83 .aether/ system files included correctly.
- **Committed in:** e074752 (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 - npm version behavior bug)
**Impact on plan:** Fix was necessary for correctness — without it, private colony data would be published to npm. No scope creep.

## Issues Encountered
- Pre-existing test failures: `validate-state` tests (2 failing) unrelated to pipeline changes — the `validate-state` command in aether-utils.sh exits 0 without arguments when tests expect exit 1. Pre-dates this plan (confirmed by reverting changes and re-running tests).

## Self-Check: PASSED
- `bin/validate-package.sh` exists and returns "Package validation passed."
- `e074752` exists in git log
- `8d25dcb` exists in git log
- `runtime/` does not exist: confirmed
- `bin/sync-to-runtime.sh` does not exist: confirmed
- SYSTEM_FILES: zero matches in cli.js and update-transaction.js: confirmed
- `npm pack --dry-run` shows 83 .aether/ files, 0 runtime/ files, 0 private dir files

## Next Phase Readiness
- Plan 02 (bash tests) needs to update any tests referencing sync-to-runtime.sh or runtime/
- Plan 03 (documentation) should update CHANGELOG.md and CLAUDE.md for v4.0 changes
- setupHub() migration message is live for v3.x -> v4.0 upgrades

---
*Phase: 20-distribution-simplification*
*Completed: 2026-02-19*
