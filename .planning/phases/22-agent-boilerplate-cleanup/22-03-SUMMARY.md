---
phase: 22-agent-boilerplate-cleanup
plan: 03
subsystem: agents
tags: [opencode, agents, boilerplate, cleanup, markdown, commands]

requires:
  - phase: 22-01
    provides: Core 5 + Development 4 agents already cleaned of boilerplate
  - phase: 22-02
    provides: Knowledge 4 + Quality 4 agents verified clean

provides:
  - Special 3 agents (archaeologist, chaos, architect) verified clean of boilerplate
  - All 4 surveyor agents updated to 'Use this agent for' description format
  - Missing OpenCode resume command created (fixes pre-existing 34 vs 33 count mismatch)
  - All 21 target agents now have standardized descriptions

affects: [future-agent-work, phase-23]

tech-stack:
  added: []
  patterns:
    - "Agent descriptions: all agents use 'Use this agent for...' format"
    - "OpenCode commands: must match Claude Code command list (count and names)"

key-files:
  created:
    - .opencode/commands/ant/resume.md
  modified:
    - .opencode/agents/aether-surveyor-nest.md
    - .opencode/agents/aether-surveyor-disciplines.md
    - .opencode/agents/aether-surveyor-pathogens.md
    - .opencode/agents/aether-surveyor-provisions.md

key-decisions:
  - "Special 3 agents (archaeologist, chaos, architect) were pre-completed in 22-01 — no changes needed"
  - "Missing OpenCode resume.md was a Rule 3 deviation (blocking lint:sync) — created to fix file count mismatch"
  - "Content-level drift in commands (10+ files) is pre-existing and out of scope for this plan"
  - "Pre-existing test failures in validate-state.test.js (2 failures) are out of scope"

patterns-established:
  - "Description standardization: use exact plan-specified text for 'Use this agent for...' format"
  - "Surveyor agents: XML body is completely untouched — only YAML frontmatter description changes"

requirements-completed: [AGENT-01, AGENT-02, AGENT-03, AGENT-04]

duration: 5min
completed: 2026-02-19
---

# Phase 22 Plan 03: Special 3 + Surveyor 4 Agent Boilerplate Cleanup Summary

**Surveyor 4 descriptions standardized to 'Use this agent for' format; Special 3 verified pre-clean; OpenCode resume command gap closed**

## Performance

- **Duration:** 5 min
- **Started:** 2026-02-19T21:48:10Z
- **Completed:** 2026-02-19T21:52:38Z
- **Tasks:** 2
- **Files modified:** 5 (4 surveyor agents + 1 new OpenCode command)

## Accomplishments

- Verified Special 3 agents (archaeologist, chaos, architect) already clean of all 3 boilerplate sections (pre-completed in 22-01)
- Updated all 4 surveyor agent descriptions from old format to "Use this agent for..." format
- Created missing `.opencode/commands/ant/resume.md` to fix pre-existing file-count mismatch in lint:sync
- All 21 target agents in Phase 22 now have standardized "Use this agent for..." descriptions

## Task Commits

1. **Task 1: Verify Special 3 agents clean, fix missing OpenCode resume command** - `89f5bab` (feat)
2. **Task 2: Update Surveyor 4 agent descriptions** - `2b27118` (feat)

**Plan metadata:** TBD (docs: complete plan)

## Files Created/Modified

- `.opencode/commands/ant/resume.md` - New: OpenCode version of the resume command (matching Claude Code's resume.md)
- `.opencode/agents/aether-surveyor-nest.md` - Description updated to "Use this agent for mapping architecture..."
- `.opencode/agents/aether-surveyor-disciplines.md` - Description updated to "Use this agent for mapping coding conventions..."
- `.opencode/agents/aether-surveyor-pathogens.md` - Description updated to "Use this agent for identifying technical debt..."
- `.opencode/agents/aether-surveyor-provisions.md` - Description updated to "Use this agent for mapping technology stack..."

## Decisions Made

- Special 3 agents were pre-completed: same pattern as 22-01 and 22-02 where cleanup happened ahead of the batching
- Created OpenCode resume.md as a Rule 3 deviation (blocking lint:sync file count check)
- Content-level command drift (10+ files) and 2 failing validate-state tests are pre-existing — not addressed

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Created missing OpenCode resume command**
- **Found during:** Task 1 (lint:sync verification)
- **Issue:** `.opencode/commands/ant/resume.md` did not exist — Claude Code had 34 commands, OpenCode had 33. The lint:sync file-count check was failing (exit code 1) before any task work.
- **Fix:** Created `.opencode/commands/ant/resume.md` mirroring the Claude Code version with OpenCode frontmatter conventions (`name: ant:resume`, no `symbol` field)
- **Files modified:** `.opencode/commands/ant/resume.md`
- **Verification:** `npm run lint:sync` file-count check passes (34/34 commands)
- **Committed in:** `89f5bab` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Required to restore lint:sync file-count parity. No scope creep — the resume command existed in Claude Code and simply needed its OpenCode counterpart.

## Issues Encountered

- **Pre-existing content-level command drift:** 10+ command files have different content between Claude Code and OpenCode directories. lint:sync still exits with code 1 due to content-level checksums. This predates all Phase 22 work and is out of scope.
- **Pre-existing test failures:** 2 tests in `validate-state.test.js` fail. Also predates this plan. Out of scope.

## Next Phase Readiness

- Phase 22 complete: all 21 target agents now have boilerplate removed and standardized descriptions
- lint:sync file count is now passing (34/34) — content-level drift remains as known debt
- Ready for Phase 23 or whatever follows the agent cleanup work

## Self-Check: PASSED

All files confirmed present and all commits verified in git log.

---
*Phase: 22-agent-boilerplate-cleanup*
*Completed: 2026-02-19*
