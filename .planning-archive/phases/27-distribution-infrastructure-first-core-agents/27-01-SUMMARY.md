---
phase: 27-distribution-infrastructure-first-core-agents
plan: 01
subsystem: distribution
tags: [agents, distribution, hub, cli, init, update-transaction]
dependency_graph:
  requires: []
  provides: [claude-agent-distribution-pipeline]
  affects: [bin/cli.js, bin/lib/update-transaction.js, bin/lib/init.js, package.json]
tech_stack:
  added: []
  patterns: [syncDirWithCleanup, hash-based-idempotency, stale-file-removal]
key_files:
  created: []
  modified:
    - package.json
    - bin/cli.js
    - bin/lib/update-transaction.js
    - bin/lib/init.js
decisions:
  - "Hub path agents-claude kept separate from agents (opencode) to prevent cross-contamination"
  - "Existing syncDirWithCleanup provides hash-based idempotency and stale file removal for free"
  - "init.js stale path bug fixed: HUB_COMMANDS_CLAUDE/OPENCODE/AGENTS now use HUB_SYSTEM not HUB_DIR"
metrics:
  duration: 173s
  completed: 2026-02-20T07:01:08Z
  tasks_completed: 2
  files_modified: 4
---

# Phase 27 Plan 01: Distribution Pipeline for Claude Agents Summary

Wire the complete 5th sync path for Claude Code agent distribution — `.claude/agents/ant/` files now flow from Aether repo through hub (`~/.aether/system/agents-claude/`) to any target repo via `aether update`.

## What Was Built

Four files modified to wire the agent distribution pipeline end-to-end:

**package.json** — Added `.claude/agents/ant/` to the `files` array so npm packaging includes agent files scoped to the `ant/` subdirectory only. GSD agents in the parent `.claude/agents/` are automatically excluded by npm path matching.

**bin/cli.js** — Five additions:
1. `HUB_AGENTS_CLAUDE` constant pointing to `~/.aether/system/agents-claude/`
2. `agents-claude` directory creation in `setupHub()`
3. Sync block in `setupHub()` that copies `.claude/agents/ant/` to hub using `syncDirWithCleanup`
4. `.claude/agents/ant/**` added to `CHECKPOINT_ALLOWLIST` so git stash safety covers agent files
5. `.claude/agents/ant` added to `updateRepo()` targetDirs for git dirty checks
6. `agents_claude`/`agentsClaude` counts wired into all four `updateRepo()` display strings

**bin/lib/update-transaction.js** — Four additions:
1. `HUB_AGENTS_CLAUDE` constant in constructor
2. `.claude/agents/ant` added to `targetDirs`
3. `agents_claude` key in `syncFiles()` initial results + sync block that delivers from hub to `.claude/agents/ant/`
4. `HUB_AGENTS_CLAUDE` added to `verifyIntegrity()` and `checkHubAccessibility()`

**bin/lib/init.js** — Two changes:
1. Fixed stale hub paths: `HUB_COMMANDS_CLAUDE`, `HUB_COMMANDS_OPENCODE`, and `HUB_AGENTS` were using `HUB_DIR` instead of `HUB_SYSTEM` — corrected to use `HUB_SYSTEM`
2. Added `HUB_AGENTS_CLAUDE` constant and claude agents sync block for new repo initialization

## Commits

| Task | Commit | Description |
|------|--------|-------------|
| Task 1 | 8f28520 | feat(27-01): wire package.json and cli.js for agent distribution |
| Task 2 | b6b7b4e | feat(27-01): wire update-transaction.js and init.js for agent delivery |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed stale hub path references in init.js**
- **Found during:** Task 2
- **Issue:** `HUB_COMMANDS_CLAUDE`, `HUB_COMMANDS_OPENCODE`, and `HUB_AGENTS` in init.js used `path.join(HUB_DIR, ...)` instead of `path.join(HUB_SYSTEM, ...)`. This would cause init to look in `~/.aether/commands/claude/` instead of `~/.aether/system/commands/claude/` — the v4.0 hub structure where all system files live under `system/`.
- **Fix:** Changed all three constants to use `HUB_SYSTEM` (which is `HUB_DIR/system`) as the base path.
- **Files modified:** `bin/lib/init.js`
- **Commit:** b6b7b4e

The plan explicitly called this out as part of Task 2, so this was not an unexpected deviation — it was a documented bug fix within the task scope.

## Verification Results

- 415 tests pass (0 failures, 9 skipped)
- `HUB_AGENTS_CLAUDE` appears in all 3 JS files (11 total occurrences)
- `agents/ant` entry in package.json `files` array
- `agents-claude` path string in all 3 JS files
- CHECKPOINT_ALLOWLIST includes `.claude/agents/ant/**`
- All four `updateRepo()` display strings show claude agent counts

## Self-Check: PASSED

All committed files verified to exist. All commit hashes confirmed in git log.
