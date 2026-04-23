---
phase: 39-opencode-agent-frontmatter
plan: 02
subsystem: platform-parity
tags: [opencode, yaml, validation, install, update, go-testing]

# Dependency graph
requires:
  - phase: 39-01
    provides: "Valid OpenCode YAML frontmatter for all 25 agent files, schema test infrastructure"
provides:
  - validateOpenCodeAgentFile function in cmd/platform_sync.go
  - Validation wired into install (source->hub) and update (hub->downstream) pipelines
  - 15 test cases covering all validation rules plus real agent file sweep
  - Test fixtures fixed to pass new validation
affects: [39-03]

# Tech tracking
tech-stack:
  added: []
  patterns: [frontmatter validation mirroring Codex TOML validation pattern]

key-files:
  created:
    - cmd/opencode_agent_validate_test.go
  modified:
    - cmd/platform_sync.go (validateOpenCodeAgentFile already present from prior wave)
    - cmd/install_cmd_test.go (fixture fix)
    - cmd/e2e_install_setup_update_test.go (fixture fix)

key-decisions:
  - "Validation mirrors existing validateCodexAgentFile pattern in platform_sync.go"
  - "Test fixtures that create builder.md without frontmatter must be updated to have valid OpenCode frontmatter"
  - "tools-as-string error detected at YAML unmarshal level (struct field type mismatch), not at the raw map re-check level"

requirements-completed: []

# Metrics
duration: 11min
completed: 2026-04-23
---

# Phase 39 Plan 02: Add OpenCode Agent Validation to Install/Update Pipeline Summary

**OpenCode agent frontmatter validation in install/update pipeline with 15 test cases and fixture corrections**

## Performance

- **Duration:** 11 min
- **Started:** 2026-04-23T11:12:51Z
- **Completed:** 2026-04-23T11:24:00Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- validateOpenCodeAgentFile function validates 8 rules: .md extension, valid UTF-8, YAML frontmatter, description 20+ chars, tools as map/object, hex or theme color, no name field, model with provider/ format
- Validation wired into both installSyncPairs() and repoSyncPairs() preventing invalid frontmatter from reaching downstream repos
- 15 test cases covering all validation rules plus a sweep of all 25 real agent files
- Two test fixtures fixed that broke when validation was added to the install/update pipeline

## Task Commits

Each task was committed atomically:

1. **Task 1: Add validateOpenCodeAgentFile function to platform_sync.go** - `c8b3b4b2` (feat)
2. **Task 2: Add tests for validateOpenCodeAgentFile** - `6e01170e` (test)

## Files Created/Modified
- `cmd/platform_sync.go` - validateOpenCodeAgentFile function with 8 validation rules, openCodeAgentFrontmatter struct, theme color map, hex color regex
- `cmd/opencode_agent_validate_test.go` - 15 sub-tests: valid hex color, valid theme color, all 7 theme colors, missing description, short description, tools as string, missing tools, named color rejection, name field presence, model without slash, missing model, missing frontmatter, non-md extension, invalid YAML, real agent file sweep
- `cmd/install_cmd_test.go` - Fixed TestInstallCopiesOpenCodeAgents builder.md fixture to have valid frontmatter
- `cmd/e2e_install_setup_update_test.go` - Fixed TestE2EInstallSetupUpdateFlow builder.md fixture to have valid frontmatter

## Decisions Made
- Followed the existing validateCodexAgentFile pattern for structural consistency
- Used a dedicated test file (opencode_agent_validate_test.go) rather than adding to platform_sync_test.go
- Reused existing findOpenCodeRepoRoot from opencode_agent_schema_test.go instead of duplicating

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed install_cmd_test.go builder.md fixture**
- **Found during:** Task 2 (running full test suite)
- **Issue:** TestInstallCopiesOpenCodeAgents created a builder.md with `# Builder agent` content (no frontmatter). The new validator correctly rejects it during install.
- **Fix:** Updated fixture to include valid OpenCode YAML frontmatter with description, mode, model, color, and tools map.
- **Files modified:** cmd/install_cmd_test.go
- **Committed in:** 6e01170e (Task 2 commit)

**2. [Rule 1 - Bug] Fixed e2e_install_setup_update_test.go builder.md fixture**
- **Found during:** Task 2 (running full test suite)
- **Issue:** TestE2EInstallSetupUpdateFlow created a builder.md with `# OC Builder agent` content (no frontmatter). The new validator correctly rejects it during install/setup/update.
- **Fix:** Updated fixture to include valid OpenCode YAML frontmatter.
- **Files modified:** cmd/e2e_install_setup_update_test.go
- **Committed in:** 6e01170e (Task 2 commit)

**3. [Rule 1 - Bug] findOpenCodeRepoRoot redeclaration**
- **Found during:** Task 2 (compilation)
- **Issue:** Duplicate function name collided with existing function in opencode_agent_schema_test.go (from Plan 01).
- **Fix:** Removed duplicate function, reused existing one from opencode_agent_schema_test.go.
- **Files modified:** cmd/opencode_agent_validate_test.go
- **Committed in:** 6e01170e (Task 2 commit)

---

**Total deviations:** 3 auto-fixed (3 bugs)
**Impact on plan:** All auto-fixes necessary for correctness. No scope creep.

## Issues Encountered
- tools-as-string test: The struct-level YAML unmarshal rejects string tools before the raw map re-check can run (because the struct field is typed as `map[string]interface{}`). Test adjusted to accept either error path since both are valid rejection reasons.
- Pre-existing TestClaudeOpenCodeAgentContentParity failure (25/25 agents show line count drift between Claude and OpenCode) is unrelated to this plan and was already failing before our changes.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Validation runs during both install and update, preventing future frontmatter regressions
- Plan 03 (E2E test) can build on this validation infrastructure
- Only pre-existing content parity test failure remains (out of scope for this plan)

---
*Phase: 39-opencode-agent-frontmatter*
*Completed: 2026-04-23*
