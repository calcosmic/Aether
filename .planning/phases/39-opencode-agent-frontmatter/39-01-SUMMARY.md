---
phase: 39-opencode-agent-frontmatter
plan: 01
subsystem: platform-parity
tags: [opencode, yaml, frontmatter, agents, schema-validation]

# Dependency graph
requires: []
provides:
  - Valid OpenCode YAML frontmatter for all 25 agent files
  - Go test enforcing OpenCode agent schema constraints
affects: [39-02, 39-03]

# Tech tracking
tech-stack:
  added: [gopkg.in/yaml.v3 for frontmatter parsing]
  patterns: [schema validation test for markdown frontmatter]

key-files:
  created:
    - cmd/opencode_agent_schema_test.go
  modified:
    - .opencode/agents/aether-*.md (all 25 files)

key-decisions:
  - "Schema format was already correct from prior phase 35 work; this plan fixed color assignments to match locked mapping"
  - "Used yaml.v3 map[string]interface{} type assertion (not map[interface{}]interface{})"

requirements-completed: []

# Metrics
duration: 11min
completed: 2026-04-23
---

# Phase 39 Plan 01: Rewrite OpenCode Agent Frontmatter Summary

**Valid OpenCode YAML frontmatter with hex colors, provider/model IDs, and tools-as-object for all 25 agent files**

## Performance

- **Duration:** 11 min
- **Started:** 2026-04-23T10:48:07Z
- **Completed:** 2026-04-23T10:59:04Z
- **Tasks:** 2
- **Files modified:** 26 (25 agent files + 1 test file)

## Accomplishments
- All 25 OpenCode agent files have valid YAML frontmatter (no name field, mode: subagent, tools as object, hex colors, provider/model format)
- Color assignments corrected to match locked color mapping from CONTEXT.md (12 files changed)
- Go test `TestOpenCodeAgentSchema` validates all 6 schema rules across all 25 files

## Task Commits

Each task was committed atomically:

1. **Task 1: Rewrite frontmatter for all 25 OpenCode agent files** - `1561afdd` (fix)
2. **Task 2: Verify all 25 files parse as valid YAML frontmatter** - `a2c00e5d` (test)

## Files Created/Modified
- `.opencode/agents/aether-chaos.md` - color: cyan (#1abc9c)
- `.opencode/agents/aether-gatekeeper.md` - color: orange (#e67e22)
- `.opencode/agents/aether-medic.md` - color: red (#ff0000)
- `.opencode/agents/aether-scout.md` - color: blue (#3498db)
- `.opencode/agents/aether-auditor.md` - color: orange (#e67e22)
- `.opencode/agents/aether-keeper.md` - color: purple (#9b59b6)
- `.opencode/agents/aether-includer.md` - color: orange (#e67e22)
- `.opencode/agents/aether-measurer.md` - color: orange (#e67e22)
- `.opencode/agents/aether-sage.md` - color: orange (#e67e22)
- `.opencode/agents/aether-chronicler.md` - color: orange (#e67e22)
- `.opencode/agents/aether-weaver.md` - color: orange (#e67e22)
- `.opencode/agents/aether-tracker.md` - color: red (#ff0000)
- `cmd/opencode_agent_schema_test.go` - Schema validation test for all 25 files

## Decisions Made
- The frontmatter schema format (no name, mode: subagent, tools as object, hex colors, provider/model) was already correct from prior Phase 35 platform-parity work. Only the color hex values needed correction to match the locked mapping in CONTEXT.md.
- 12 of 25 files had incorrect color assignments. The plan's color mapping was treated as authoritative.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- `findRepoRoot` function name collided with existing function in `cmd/codex_e2e_test.go`. Renamed to `findOpenCodeRepoRoot` to avoid the redeclaration error.
- YAML v3 library unmarshals maps as `map[string]interface{}` not `map[interface{}]interface{}`. Fixed type assertion accordingly.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- All 25 agent files pass schema validation
- Plan 02 (validation in install/update pipeline) can build on the test infrastructure here
- Plan 03 (E2E test) can use these corrected files as the baseline

---
*Phase: 39-opencode-agent-frontmatter*
*Completed: 2026-04-23*
