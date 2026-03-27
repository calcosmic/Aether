---
phase: 17-local-wisdom-accumulation
plan: 01
subsystem: wisdom
tags: [queen, queen.md, wisdom, pheromone, bash, colony-prime]

requires: []
provides:
  - "4-section QUEEN.md template (User Preferences, Codebase Patterns, Build Learnings, Instincts)"
  - "queen-write-learnings subcommand for direct build learning writes"
  - "queen-promote-instinct subcommand for instinct promotion"
  - "v1-to-v2 format detection and backward-compatible parsing"
affects: [17-02, 18-wisdom-injection, 19-cross-colony-wisdom, 20-hub-wisdom]

tech-stack:
  added: []
  patterns:
    - "Format detection (v1 vs v2) via header check before extraction"
    - "Atomic file writes with temp + mv pattern for QUEEN.md mutations"
    - "Phase subsection grouping in Build Learnings section"

key-files:
  created: []
  modified:
    - ".aether/templates/QUEEN.md.template"
    - ".aether/utils/queen.sh"
    - ".aether/utils/pheromone.sh"
    - ".aether/aether-utils.sh"
    - "tests/bash/test-queen-module.sh"

key-decisions:
  - "v2 format detection via '## Build Learnings' header presence -- simple, reliable, no version field needed"
  - "v1 backward compat maps old 6 sections into 2 new keys: codebase_patterns and user_prefs"
  - "New write subcommands bypass observation thresholds (threshold 0) -- every build writes"
  - "Build learnings grouped by phase subsections (### Phase N: Name) for readability"

patterns-established:
  - "QUEEN.md format detection: check for '## Build Learnings' to distinguish v2 from v1"
  - "Dedup via grep -Fq on claim/action text before writing"

requirements-completed: [QUEEN-01, QUEEN-02]

duration: 15min
completed: 2026-03-24
---

# Phase 17 Plan 01: QUEEN.md Restructure Summary

**4-section QUEEN.md template with write subcommands for build learnings and instinct promotion, plus v1 backward compatibility**

## Performance

- **Duration:** 15 min
- **Started:** 2026-03-24T23:21:06Z
- **Completed:** 2026-03-24T23:35:50Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- Restructured QUEEN.md from 6 emoji-prefixed sections to 4 clean sections matching user decisions
- Updated both parsers (queen.sh and pheromone.sh) in lockstep with v2 keys
- Added queen-write-learnings and queen-promote-instinct subcommands with dedup and atomic writes
- Maintained full backward compatibility with v1-format QUEEN.md files
- Added 7 new tests covering v2 format, write operations, dedup, and v1 compat

## Task Commits

Each task was committed atomically:

1. **Task 1: Restructure QUEEN.md template and update all section parsers** - `ec74b59` (feat)
2. **Task 2: Add queen-write-learnings and queen-promote-instinct subcommands** - `4be6ce3` (feat)

## Files Created/Modified
- `.aether/templates/QUEEN.md.template` - New 4-section format (User Preferences, Codebase Patterns, Build Learnings, Instincts)
- `.aether/utils/queen.sh` - Updated _extract_wisdom_sections with v1/v2 format detection, updated _queen_read with v2 keys, updated _queen_promote with v2 section mapping, added _queen_write_learnings and _queen_promote_instinct
- `.aether/utils/pheromone.sh` - Updated _extract_wisdom with v1/v2 format detection, updated wisdom combination and prompt assembly to use v2 keys
- `.aether/aether-utils.sh` - Added build_learning and instinct threshold types, registered new subcommands in dispatch and help
- `tests/bash/test-queen-module.sh` - Added 7 new tests (v2 init, v2 read, write-learnings, write-learnings dedup, promote-instinct, promote-instinct dedup, v1 compat)

## Decisions Made
- Used header presence check (`## Build Learnings`) for format detection rather than parsing metadata version -- simpler and more resilient
- Mapped all 6 v1 sections into 2 v2 keys for backward compat: Philosophies+Patterns+Redirects+Stack Wisdom -> codebase_patterns, Decrees+User Preferences -> user_prefs
- Set threshold 0 for both new types (build_learning, instinct) -- write every time, no observation counting needed
- Build learnings grouped under phase subsection headers for readability and organization

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- `grep -c` returning multi-line output when combined with `|| echo "0"` fallback under `set -e` -- fixed by using `|| true` instead since `grep -c` always outputs a count

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Write subcommands are ready for Plan 02 to wire into the build workflow
- colony-prime prompt assembly updated to use new section names
- All parsers produce consistent v2 JSON keys for downstream consumers

## Self-Check: PASSED

All 5 modified files verified present. Both task commits (ec74b59, 4be6ce3) verified in git log.

---
*Phase: 17-local-wisdom-accumulation*
*Completed: 2026-03-24*
