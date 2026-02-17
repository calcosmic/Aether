---
phase: 06-colony-lifecycle
plan: 01
subsystem: colony-lifecycle
tags: [seal, ceremony, crowned-anthill, milestone, queen-promote]

# Dependency graph
requires:
  - phase: 05-pheromone-system
    provides: "instinct-read and queen-promote subcommands for wisdom promotion"
provides:
  - "Ceremony-only /ant:seal command with maturity gate, confirmation, CROWNED-ANTHILL.md"
  - "Clean separation between seal (ceremony) and entomb (archive)"
affects: [06-02-entomb, 06-03-tunnels]

# Tech tracking
tech-stack:
  added: []
  patterns: ["ceremony-then-archive lifecycle pattern", "maturity gate before milestone promotion"]

key-files:
  created: []
  modified:
    - ".aether/commands/opencode/seal.md"

key-decisions:
  - "Source of truth seal.md already matched plan specification — only OpenCode copy needed updating"
  - "OpenCode differences: normalize-args step, $normalized_args, swarm-display-render"

patterns-established:
  - "Seal = ceremony only (no archiving), Entomb = archive only (no ceremony)"
  - "CROWNED-ANTHILL.md as the seal ceremony record"

requirements-completed: [LIF-01]

# Metrics
duration: 3min
completed: 2026-02-18
---

# Phase 06 Plan 01: Rewrite /ant:seal as Ceremony-Only Summary

**Ceremony-only seal command with maturity gate, CROWNED-ANTHILL.md, ASCII art, and wisdom promotion to QUEEN.md**

## Performance

- **Duration:** 3 min
- **Started:** 2026-02-18T00:00:00Z
- **Completed:** 2026-02-18T00:03:00Z
- **Tasks:** 2
- **Files modified:** 1

## Accomplishments
- Synced OpenCode seal.md to match the ceremony-only source of truth
- Confirmed Claude Code copy already in sync with source of truth
- Verified all three locations have correct platform-specific differences

## Task Commits

Each task was committed atomically:

1. **Task 1: Verify seal.md source of truth** — Already correct, no commit needed
2. **Task 2: Sync seal.md to OpenCode** — `e38f976` (feat)

## Files Created/Modified
- `.aether/commands/opencode/seal.md` — Rewritten from old archive-based version to ceremony-only

## Decisions Made
- Source of truth (.aether/commands/claude/seal.md) and Claude Code copy (.claude/commands/ant/seal.md) already matched the plan specification exactly — no changes needed for those files
- Only the OpenCode version needed rewriting (was still the old archive-based version)

## Deviations from Plan

None — plan executed exactly as written. The source of truth was already correct, so Task 1 was a verification rather than a rewrite.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Seal ceremony command is complete across all three locations
- CROWNED-ANTHILL.md creation ready for entomb to reference
- Plans 06-02 (entomb) and 06-03 (tunnels) can proceed in Wave 2

---
*Phase: 06-colony-lifecycle*
*Completed: 2026-02-18*
