---
phase: 08-documentation-update
plan: 02
subsystem: documentation
tags: [pheromones, injection-model, readme, inventory, commands]

# Dependency graph
requires:
  - phase: 04-pheromone-worker-integration
    provides: pheromone_protocol sections in agent definitions
  - phase: 06-xml-exchange-activation
    provides: export-signals and import-signals commands
  - phase: 02-command-audit-data-tooling
    provides: data-clean command
provides:
  - Accurate pheromone documentation with injection model framing
  - README with verified counts (40 commands, 22 agents, 110 subcommands)
  - aether-colony.md with new command listings and injection model description
  - source-of-truth-map.md with current inventory snapshot
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Injection model framing: colony-prime injects signals, workers do not independently read them"

key-files:
  created: []
  modified:
    - .aether/docs/pheromones.md
    - README.md
    - .claude/rules/aether-colony.md
    - .aether/docs/source-of-truth-map.md

key-decisions:
  - "Used passive voice in signal combination table to avoid implying worker agency"
  - "Added 'How Signals Reach Workers' section to pheromones.md rather than scattering injection explanation across existing sections"
  - "Renamed README 'Coordination' section to 'Coordination & Maintenance' to accommodate data/exchange commands"
  - "Added 'Data & Exchange' section to aether-colony.md for new commands rather than appending to Advanced"
  - "Updated test count to exact value (92) since it is a stable inventory count, not a growing metric"

patterns-established:
  - "Documentation injection model: always describe colony-prime as the actor that injects signals, never describe workers as independently reading signals"

requirements-completed: [DOCS-02, DOCS-04]

# Metrics
duration: 3min
completed: 2026-03-19
---

# Phase 08 Plan 02: Pheromone Injection Model and User-Facing Docs Summary

**Pheromone docs reframed around colony-prime injection model; README, aether-colony, and source-of-truth-map updated with verified counts and new commands**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-19T22:46:50Z
- **Completed:** 2026-03-19T22:50:41Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Replaced all "workers read/check signals" language in pheromones.md with injection model framing (colony-prime injects signals into worker prompts)
- Added "How Signals Reach Workers" section explaining the full injection pipeline (colony-prime, pheromone-prime --compact, prompt_section, pheromone_protocol)
- Updated README.md with verified counts: 22 agents, 40 commands, 110 subcommands
- Added data-clean, export-signals, and import-signals to both README and aether-colony.md command tables
- Updated source-of-truth-map.md inventory snapshot with current counts (40 Claude, 39 OpenCode, 92 tests)

## Task Commits

Each task was committed atomically:

1. **Task 1: Fix pheromone documentation to describe injection model** - `85148b0` (docs)
2. **Task 2: Update README.md, aether-colony.md, and source-of-truth-map.md** - `7400f4e` (docs)

## Files Created/Modified
- `.aether/docs/pheromones.md` - Reframed around injection model, added "How Signals Reach Workers" section
- `README.md` - Updated counts (22 agents, 40 commands, 110 subcommands), added 3 new commands, fixed "runtime" comment
- `.claude/rules/aether-colony.md` - Added Data & Exchange command section, enhanced pheromone description with injection model
- `.aether/docs/source-of-truth-map.md` - Updated inventory counts (40/39 commands, 92 tests), updated date to 2026-03-19

## Decisions Made
- Used passive voice in signal combination table to avoid implying worker agency over signal reading
- Added a dedicated "How Signals Reach Workers" section to pheromones.md for clarity rather than scattering the injection explanation
- Renamed README "Coordination" section to "Coordination & Maintenance" to accommodate new commands
- Added "Data & Exchange" section to aether-colony.md rather than appending to the existing Advanced section
- Enhanced aether-colony.md pheromone description to mention injection model and pheromone_protocol

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- All four documentation files updated with verified counts and accurate model descriptions
- DOCS-02 (pheromone injection model) and DOCS-04 (README/user-facing docs) requirements complete
- No blockers for remaining Phase 8 plans

---
*Phase: 08-documentation-update*
*Completed: 2026-03-19*
