---
phase: 36-yaml-command-generator
plan: 03
subsystem: infra
tags: [yaml, code-generation, multi-provider, command-templating, provider-exclusive-blocks]

# Dependency graph
requires:
  - phase: 36-01
    provides: "YAML-to-markdown generator engine (bin/generate-commands.js)"
provides:
  - "22 YAML source files for complex commands including build.yaml and continue.yaml"
  - "Provider-exclusive blocks for constraints.json vs subcommand logic in pheromone commands"
  - "body_claude/body_opencode pattern for structurally different commands"
affects: [36-04]

# Tech tracking
tech-stack:
  added: []
  patterns: ["body_claude/body_opencode for structurally different commands (build, continue, plan, seal, etc.)", "Provider-exclusive blocks for pheromone commands (focus, redirect, feedback)", "Claude-only section wrapping for status.yaml Data Safety/Pheromone Summary/Colony Depth"]

key-files:
  created:
    - .aether/commands/focus.yaml
    - .aether/commands/redirect.yaml
    - .aether/commands/feedback.yaml
    - .aether/commands/status.yaml
    - .aether/commands/init.yaml
    - .aether/commands/flag.yaml
    - .aether/commands/plan.yaml
    - .aether/commands/watch.yaml
    - .aether/commands/resume-colony.yaml
    - .aether/commands/chaos.yaml
    - .aether/commands/organize.yaml
    - .aether/commands/archaeology.yaml
    - .aether/commands/build.yaml
    - .aether/commands/continue.yaml
    - .aether/commands/swarm.yaml
    - .aether/commands/pause-colony.yaml
    - .aether/commands/colonize.yaml
    - .aether/commands/oracle.yaml
    - .aether/commands/skill-create.yaml
    - .aether/commands/tunnels.yaml
    - .aether/commands/entomb.yaml
    - .aether/commands/seal.yaml
  modified: []

key-decisions:
  - "Used body_claude/body_opencode for 16 of 22 commands where provider bodies are structurally different"
  - "Used standard body with provider-exclusive blocks for focus, redirect, feedback, status, init, flag (6 commands with mixed shared/exclusive content)"
  - "build.yaml body_claude is 1.7KB (playbook orchestrator), body_opencode is 40KB (inline logic)"
  - "continue.yaml body_claude is 1.6KB (playbook orchestrator), body_opencode is 50KB (inline logic)"

patterns-established:
  - "body_claude/body_opencode as preferred approach for commands with >80% different content between providers"
  - "Provider-exclusive blocks ({{#claude}}/{{#opencode}}) for commands with mixed shared/exclusive content"
  - "Python extraction script for converting existing .md pairs to YAML with Step -1 removal"

requirements-completed: [INFRA-03]

# Metrics
duration: 14min
completed: 2026-03-29
---

# Phase 36 Plan 03: Complex Command YAML Conversion Summary

**22 complex commands converted to YAML source format with provider-exclusive blocks and body_claude/body_opencode for structurally different commands**

## Performance

- **Duration:** 14 min
- **Started:** 2026-03-29T11:01:33Z
- **Completed:** 2026-03-29T11:15:54Z
- **Tasks:** 2
- **Files modified:** 22

## Accomplishments
- Converted 12 medium-complexity commands (focus, redirect, feedback, status, init, flag, plan, watch, resume-colony, chaos, organize, archaeology) using provider-exclusive blocks and body_claude/body_opencode
- Converted 10 large/structural commands (build, continue, seal, entomb, tunnels, oracle, swarm, pause-colony, colonize, skill-create) using body_claude/body_opencode
- build.yaml and continue.yaml correctly preserve Claude's playbook orchestrator pattern and OpenCode's full inline logic
- All 22 YAML files parse without error; generator produces correct output for all providers

## Task Commits

Each task was committed atomically:

1. **Task 1: Convert medium-complexity commands to YAML (12 commands)** - `8d48729` (feat)
2. **Task 2: Convert large/structurally-different commands to YAML (10 commands)** - `9e0f944` (feat)

## Files Created/Modified
- `.aether/commands/focus.yaml` - FOCUS pheromone signal with constraints.json vs subcommand provider blocks
- `.aether/commands/redirect.yaml` - REDIRECT pheromone signal with constraints.json vs subcommand provider blocks
- `.aether/commands/feedback.yaml` - FEEDBACK pheromone signal with instinct creation differences
- `.aether/commands/status.yaml` - Colony dashboard with Claude-only Data Safety, Pheromone Summary, Colony Depth sections
- `.aether/commands/init.yaml` - Colony initialization with shared body and provider-exclusive blocks
- `.aether/commands/flag.yaml` - Flag creation with banner style differences
- `.aether/commands/plan.yaml` - Planning orchestrator with body_claude/body_opencode (43KB)
- `.aether/commands/watch.yaml` - tmux watch session with body_claude/body_opencode
- `.aether/commands/resume-colony.yaml` - Session restore with body_claude/body_opencode
- `.aether/commands/chaos.yaml` - Resilience testing with body_claude/body_opencode
- `.aether/commands/organize.yaml` - Hygiene report with body_claude/body_opencode
- `.aether/commands/archaeology.yaml` - Git history analysis with body_claude/body_opencode
- `.aether/commands/build.yaml` - Build orchestrator: Claude 1.7KB playbook, OpenCode 40KB inline (45KB total)
- `.aether/commands/continue.yaml` - Continue orchestrator: Claude 1.6KB playbook, OpenCode 50KB inline (57KB total)
- `.aether/commands/seal.yaml` - Colony sealing with body_claude/body_opencode (44KB)
- `.aether/commands/entomb.yaml` - Colony archiving with body_claude/body_opencode (26KB)
- `.aether/commands/tunnels.yaml` - Colony browsing with body_claude/body_opencode (28KB)
- `.aether/commands/oracle.yaml` - Deep research with body_claude/body_opencode (42KB)
- `.aether/commands/swarm.yaml` - Bug investigation with body_claude/body_opencode
- `.aether/commands/pause-colony.yaml` - Session pausing with body_claude/body_opencode
- `.aether/commands/colonize.yaml` - Codebase analysis with body_claude/body_opencode
- `.aether/commands/skill-create.yaml` - Custom skill creation with body_claude/body_opencode

## Decisions Made
- Used body_claude/body_opencode for 16 of 22 commands because their provider differences exceed 80% of content. This avoids fragile template marker placement in deeply different command structures.
- Used standard body with provider-exclusive blocks for 6 commands (focus, redirect, feedback, status, init, flag) where shared content is substantial and differences are localized (banners, constraints.json logic, Claude-only sections).
- Extracted OpenCode body content programmatically to avoid manual transcription errors in 1000+ line files.
- Preserved the Parse `$normalized_args` block when extracting OpenCode bodies (it was inside Step -1 but is needed in the body since the generator only injects the normalize-args preamble, not the parse block).

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Known Stubs
None - all YAML files contain complete command content from their source .md files.

## Next Phase Readiness
- All 22 complex command YAML files are ready for Plan 04 (sync validation and integration)
- Generator produces correct output for all converted commands
- Combined with Plan 02's 22 simple commands, the full set of 44 YAML source files will be complete

## Self-Check: PASSED

All 22 created YAML files verified present. Both task commits (8d48729, 9e0f944) verified in git history.

---
*Phase: 36-yaml-command-generator*
*Completed: 2026-03-29*
