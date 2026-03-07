---
phase: 01-instinct-pipeline
plan: 02
subsystem: workflow
tags: [instincts, pheromone-prime, colony-prime, domain-grouping, aether-utils]

# Dependency graph
requires:
  - "01-01: Instinct write-side (instinct-create, instinct-read fix, continue-advance wiring)"
provides:
  - "Domain-grouped instinct formatting in pheromone-prime output"
  - "Instincts visible in build output via colony-prime log_line"
  - "Builder prompts receive domain-grouped instincts via prompt_section injection"
affects: [01-03, colony-prime, build]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "jq group_by for domain clustering in bash shell output"
    - "Domain headers capitalized (first letter uppercase) for readability"

key-files:
  created: []
  modified:
    - ".aether/aether-utils.sh"

key-decisions:
  - "Same domain-grouped format for both compact and non-compact modes (only header line differs)"
  - "No changes needed to build-context.md or build-wave.md -- existing pipeline chain works correctly"

patterns-established:
  - "Instinct display format: domain header followed by indented confidence-tagged entries"

requirements-completed: [LEARN-03]

# Metrics
duration: 2min
completed: 2026-03-06
---

# Phase 1 Plan 2: Instinct Pipeline Read-Side Summary

**Domain-grouped instinct formatting in pheromone-prime using jq group_by, injected into builder prompts via colony-prime prompt_section**

## Performance

- **Duration:** 2 min
- **Started:** 2026-03-06T20:59:41Z
- **Completed:** 2026-03-06T21:02:00Z
- **Tasks:** 2
- **Files modified:** 1

## Accomplishments
- Replaced flat instinct listing with domain-grouped format using jq group_by
- Instincts now display under capitalized domain headers (e.g., "Architecture:", "Testing:")
- Verified end-to-end pipeline: build-context.md -> colony-prime -> pheromone-prime -> instinct-read
- Confirmed instinct count visible in build output via "Primed: N signals, M instincts" log_line

## Task Commits

Each task was committed atomically:

1. **Task 1: Add domain-grouped instinct formatting to pheromone-prime** - `066c0b0` (feat)
2. **Task 2: Verify end-to-end instinct visibility in build output** - no commit (verification-only, no file changes)

## Files Created/Modified
- `.aether/aether-utils.sh` - Replaced flat instinct listing in pheromone-prime with jq group_by domain-grouped format

## Decisions Made
- Same domain-grouped format used for both compact and non-compact modes; only difference is the header explanation line ("Weight by confidence...") in non-compact
- No changes needed to build-context.md or build-wave.md: the existing chain (build-context Step 4 -> colony-prime --compact -> pheromone-prime -> instinct-read) already passes prompt_section through correctly

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Instinct read-side is complete: builders now see domain-grouped instincts in their prompts
- Ready for Plan 03 (confidence decay and instinct lifecycle)
- Full pipeline working: continue creates instincts (Plan 01) -> colony-prime displays them grouped by domain (Plan 02)

## Self-Check: PASSED

All files exist. All commits verified (066c0b0).

---
*Phase: 01-instinct-pipeline*
*Completed: 2026-03-06*
