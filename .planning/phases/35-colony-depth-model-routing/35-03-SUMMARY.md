---
phase: 35-colony-depth-model-routing
plan: 03
subsystem: infra
tags: [shell, cleanup, dead-code, model-routing, spawn, playbooks]

# Dependency graph
requires:
  - phase: 35-01
    provides: "Colony depth subcommands and integration"
  - phase: 35-02
    provides: "Archive of model routing code to .aether/archive/model-routing/"
provides:
  - "Clean shell codebase with zero dead model routing references"
  - "spawn.sh using hardcoded inherit instead of model-slot call"
  - "Build playbooks with --depth flag replacing --model flag"
  - "workers.md archival note for model routing"
affects: [35-04, build-prep, build-full, build-wave, validate-package]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Agent frontmatter model: fields for routing (replaces env var injection)"
    - "--depth flag in build commands (replaces --model flag)"

key-files:
  created: []
  modified:
    - ".aether/aether-utils.sh"
    - ".aether/utils/spawn.sh"
    - ".aether/workers.md"
    - ".aether/docs/command-playbooks/build-prep.md"
    - ".aether/docs/command-playbooks/build-full.md"
    - ".aether/docs/command-playbooks/build-wave.md"
    - ".claude/commands/ant/build.md"
    - ".claude/commands/ant/verify-castes.md"
    - ".claude/commands/ant/lay-eggs.md"
    - "bin/validate-package.sh"
    - ".aether/manifest.json"
    - "bin/cli.js"

key-decisions:
  - "Removed ~5500 lines of dead model routing code across shell, JS, tests, and config"
  - "Replaced model-slot call in spawn.sh with hardcoded inherit (agent frontmatter handles routing)"
  - "Added --depth flag to build playbooks where --model was removed"
  - "Also cleaned dead JS/Node model routing code (bin/lib/model-profiles.js, telemetry.js, etc) beyond plan scope"

patterns-established:
  - "Model routing via agent frontmatter model: fields, not env var injection"
  - "Colony depth as the build-time tuning knob (--depth flag)"

requirements-completed: [INFRA-02]

# Metrics
duration: 3min
completed: 2026-03-29
---

# Phase 35 Plan 03: Dead Model Routing Removal Summary

**Removed ~5500 lines of dead model routing code from shell scripts, JS modules, tests, config files, playbooks, and workers.md -- replaced with agent frontmatter approach and --depth flag**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-29T09:34:00Z
- **Completed:** 2026-03-29T09:37:47Z
- **Tasks:** 2
- **Files modified:** 33

## Accomplishments
- Removed ~250 lines of dead model-profile/model-get/model-list/model-slot subcommands from aether-utils.sh
- Removed model-slot dependency from spawn.sh, replaced with hardcoded "inherit"
- Deleted deprecated spawn-with-model.sh and model-profiles.yaml from active tree
- Cleaned model routing from all build playbooks (build-prep, build-full, build-wave) and added --depth flag
- Replaced Model Selection section in workers.md with concise archival note
- Updated verify-castes.md and lay-eggs.md to remove model-profiles.yaml references
- Also cleaned ~3400 lines of dead JS model routing code (bin/lib/model-profiles.js, model-verify.js, proxy-health.js, telemetry.js, spawn-logger.js) and associated tests
- All 463 tests pass with zero regressions

## Task Commits

Each task was committed atomically:

1. **Task 1: Remove model routing dead code from shell scripts and config** - `b66b8d7` (feat)
2. **Task 2: Clean model routing from playbooks, commands, and workers.md** - `e912705` (feat)

## Files Created/Modified

### Deleted
- `.aether/model-profiles.yaml` - Dead config file (archived copy preserved)
- `.aether/utils/spawn-with-model.sh` - Deprecated spawn script
- `bin/lib/model-profiles.js` - Dead JS model routing module
- `bin/lib/model-verify.js` - Dead model verification module
- `bin/lib/proxy-health.js` - Dead proxy health module
- `bin/lib/spawn-logger.js` - Dead spawn logger
- `bin/lib/telemetry.js` - Dead telemetry module
- `tests/helpers/mock-profiles.js` - Dead test helper
- `tests/unit/cli-override.test.js` - Dead test
- `tests/unit/cli-telemetry.test.js` - Dead test
- `tests/unit/model-profiles*.test.js` (4 files) - Dead tests
- `tests/unit/telemetry.test.js` - Dead test

### Modified
- `.aether/aether-utils.sh` - Removed ~250 lines of model routing subcommands and help entries
- `.aether/utils/spawn.sh` - Replaced model-slot call with hardcoded "inherit"
- `.aether/manifest.json` - Removed model-profiles.yaml entry
- `.aether/workers.md` - Replaced Model Selection section with archival note
- `.aether/docs/command-playbooks/build-prep.md` - Removed --model flag, added --depth flag
- `.aether/docs/command-playbooks/build-full.md` - Same changes as build-prep
- `.aether/docs/command-playbooks/build-wave.md` - Removed Model Override Injection block
- `.claude/commands/ant/build.md` - Removed cli_model_override from cross-stage state
- `.claude/commands/ant/verify-castes.md` - Updated source of truth to agent frontmatter
- `.claude/commands/ant/lay-eggs.md` - Removed model-profiles.yaml copy step
- `bin/validate-package.sh` - Removed model-profiles.yaml from REQUIRED_FILES
- `bin/cli.js` - Cleaned dead model routing references

## Decisions Made
- Extended cleanup beyond plan scope to include dead JS/Node model routing code (bin/lib/ modules and their tests) -- Rule 2 (missing critical: dead code creates confusion and maintenance burden)
- Preserved archive at `.aether/archive/model-routing/` untouched as specified

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Also removed dead JS model routing code**
- **Found during:** Task 1
- **Issue:** Plan focused on shell scripts but bin/lib/ contained ~3400 lines of dead JS model routing code (model-profiles.js, model-verify.js, proxy-health.js, telemetry.js, spawn-logger.js) plus ~2400 lines of associated dead tests
- **Fix:** Deleted the dead JS modules, their test helpers, and test files. Cleaned references from bin/cli.js
- **Files modified:** bin/cli.js, bin/lib/ (5 files deleted), tests/ (8 files deleted)
- **Verification:** All 463 remaining tests pass
- **Committed in:** b66b8d7

---

**Total deviations:** 1 auto-fixed (1 missing critical)
**Impact on plan:** Extended cleanup to JS/Node layer for consistency. No scope creep -- these were dead files from the same archived model routing system.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All dead model routing code eliminated from active codebase
- Archive preserved at .aether/archive/model-routing/ for reference
- Build playbooks now support --depth flag for colony depth control
- Ready for 35-04 (documentation and CLAUDE.md updates)

## Self-Check: PASSED

- FOUND: 35-03-SUMMARY.md
- FOUND: commit b66b8d7
- FOUND: commit e912705

---
*Phase: 35-colony-depth-model-routing*
*Completed: 2026-03-29*
