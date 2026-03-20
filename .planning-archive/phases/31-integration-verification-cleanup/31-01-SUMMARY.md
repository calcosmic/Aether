---
phase: 31-integration-verification-cleanup
plan: 01
subsystem: agent-system
tags: [agent-integration, bash-wrapping-bug, lint-test, command-files]

# Dependency graph
requires:
  - phase: 30-niche-agents
    provides: 22 agents fully quality-validated
provides:
  - INT-01/02/03 verification of agent integration chain
  - CLEAN-03 bash wrapping bug fix across 7 command files
  - AVA lint test to prevent bash wrapping regression
affects: [all future command development]

# Tech tracking
tech-stack:
  added: []
  patterns: [description-in-prose-not-code-blocks, AVA-lint-for-command-quality]

key-files:
  created: []
  modified:
    - .claude/commands/ant/swarm.md
    - .claude/commands/ant/colonize.md
    - .claude/commands/ant/entomb.md
    - .claude/commands/ant/seal.md
    - .claude/commands/ant/init.md
    - .claude/commands/ant/plan.md
    - tests/unit/agent-quality.test.js

key-decisions:
  - "31-01: Description text belongs in instruction prose above bash blocks, not inside them"
  - "31-01: CLEAN-03 lint test scans both .claude/commands/ant/ and .opencode/commands/ant/"

patterns-established:
  - "Pattern: Run using the Bash tool with description \"...\": followed by code block (description in prose)"
  - "Pattern: AVA test findBashWrappingBug() tracks in-bash-block 'with description' pattern"

requirements-completed: [INT-01, INT-02, INT-03, CLEAN-03]

# Metrics
duration: 20m
completed: 2026-02-20
---

# Phase 31 Plan 01: Agent Integration Verification Summary

**Verified agent integration chain (INT-01/02/03) and fixed 58 instances of bash wrapping bug across 7 command files with regression test**

## Performance

- **Duration:** 20m
- **Started:** 2026-02-20T13:09:28Z
- **Completed:** 2026-02-20T13:29:40Z
- **Tasks:** 2
- **Files modified:** 8

## Accomplishments

- Verified 22 agents synced to hub at ~/.aether/system/agents-claude/
- Confirmed agent resolution path: subagent_type matches .claude/agents/ant/ filename
- Verified return format fields match between aether-builder.md and build.md Step 5.2
- Traced colony state wiring: build sets EXECUTING, continue advances to READY
- Fixed 58 instances of bash wrapping bug (description inside code blocks)
- Added CLEAN-03 AVA lint test to prevent regression

## Task Commits

Each task was committed atomically:

1. **Task 1: Verify agent integration chain (INT-01, INT-02, INT-03)** - verification only, no code changes
2. **Task 2: Fix bash wrapping bug in 7 command files and add lint test** - `e3899ea` (fix)

**Plan metadata:** (pending final commit)

## Files Created/Modified

- `.claude/commands/ant/swarm.md` - Fixed 13 bash wrapping instances
- `.claude/commands/ant/colonize.md` - Fixed 11 bash wrapping instances
- `.claude/commands/ant/entomb.md` - Fixed 11 bash wrapping instances
- `.claude/commands/ant/seal.md` - Fixed 10 bash wrapping instances
- `.claude/commands/ant/init.md` - Fixed 5 bash wrapping instances
- `.claude/commands/ant/plan.md` - Fixed 8 bash wrapping instances (inside blocks only)
- `.claude/commands/ant/pause-colony.md` - Already clean (prose descriptions only)
- `tests/unit/agent-quality.test.js` - Added CLEAN-03 lint test

## Decisions Made

- **Description placement:** The `with description "..."` text must be in instruction prose ABOVE bash code blocks, not inside them. Inside blocks causes "with: command not found" errors when Claude Code executes literally.
- **Lint test scope:** CLEAN-03 scans both `.claude/commands/ant/` and `.opencode/commands/ant/` directories to catch the bug regardless of which tool's commands are affected.

## Integration Verification Results

### INT-01: Agent Resolution
- 22 agents synced to hub at `~/.aether/system/agents-claude/`
- `aether-builder.md` exists at `.claude/agents/ant/aether-builder.md`
- Frontmatter `name: aether-builder` matches `subagent_type="aether-builder"` used in build.md
- YAML frontmatter is parseable (no syntax errors)

### INT-02: Return Format Compatibility
- aether-builder.md return_format includes: `ant_name`, `status`, `files_created`, `files_modified`, `blockers`
- build.md Step 5.2 expects same fields plus `tool_count` (injected at prompt level)
- Fields match between agent and command expectations

### INT-03: Colony State Update Wiring
- build.md Step 2: Sets `state: "EXECUTING"`
- build.md Step 5.9: Synthesizes results but does NOT advance state
- continue.md Step 2: Marks phase `status: "completed"` and sets `state: "READY"`
- Wiring confirmed: build -> EXECUTING -> agent runs -> build synthesizes -> continue -> READY

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None - all verification steps passed and bug fix was straightforward.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Agent integration fully verified for v2.0 shipping
- Bash wrapping bug eliminated across all command files
- Regression test in place to prevent reoccurrence
- Ready for Phase 31 Plans 02 and 03 (additional cleanup tasks)

---
*Phase: 31-integration-verification-cleanup*
*Completed: 2026-02-20*
