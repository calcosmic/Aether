# Aether Development Context

> Meta-context for developing Aether itself. Read this first when working on the colony system.

---

## Critical Architecture Decisions (WHY)

| Decision | Why | Files |
|----------|-----|-------|
| `.aether/` is source of truth | runtime/ is auto-generated staging | `.aether/*.md`, `.aether/*.sh` |
| Sync script, not manual copy | Prevents drift, enforces allowlist | `bin/sync-to-runtime.sh` |
| Hub model for distribution | `npm install -g .` pushes to `~/.aether/` | `bin/cli.js` |
| Checkpoint allowlist | Bug fix: git stash nearly lost 1,145 lines of user work | `.aether/data/checkpoint-allowlist.json` |
| Session freshness detection | Stale session files silently broke workflows | `.aether/aether-utils.sh:3181-3381` |

**Edit .aether/, NOT runtime/**. Changes to runtime/ will be overwritten on sync.

---

## Known Bugs (Do Not Forget)

### Critical (Fix Now)

1. **BUG-005/BUG-011: Lock deadlock in flag-auto-resolve**
   - Location: `.aether/aether-utils.sh:1022`
   - If jq fails, lock never released -> deadlock
   - Workaround: Restart colony session if commands hang on flags

2. **ISSUE-004: Template path hardcoded to runtime/**
   - Location: `.aether/aether-utils.sh:2689`
   - queen-init fails when Aether installed via npm
   - Workaround: Use git clone instead of npm install

### Medium Priority

3. **Model routing UNVERIFIED** (P0.5 in TO-DOS)
   - Configuration exists: `model-profiles.yaml` maps castes to models
   - Execution unproven: ANTHROPIC_MODEL may not be inherited by spawned workers
   - Test: `/ant:verify-castes` Step 3 spawns test worker

4. **Error code inconsistency** (BUG-007)
   - 17+ locations use hardcoded strings instead of `$E_*` constants
   - Pattern: early commands use strings, later commands use constants

---

## Current Work In Progress

### Recently Completed (2026-02-16)

- **Session Freshness Detection** - All 9 phases done, 21/21 tests passing
  - Commands: colonize, oracle, watch, swarm, init, seal, entomb
  - Protected: init/seal/entomb never auto-clear (precious data)

### Design Plans Pending Approval

- **Aether Hardening** (`docs/plans/2026-02-16-aether-hardening-design.md`)
  - 6 phases: Modular memory, hooks, permissions, CI, OpenCode alignment, governance
  - Not started - awaiting approval

### Active TO-DOs (Priority 0-1)

1. Deprecate old 2.x npm versions (one command)
2. Apply timestamp verification to `/ant:oracle` command
3. Convert colony prompts to XML format
4. Interactive caste model configuration in Claude
5. Colony lifecycle management (archive/seal commands)

---

## Deferred Technical Debt

| Debt | Why Deferred | Impact |
|------|--------------|--------|
| YAML command generator | Works manually, not broken | 13,573 lines duplicated across .claude/ and .opencode/ |
| Test coverage audit | Tests pass, purpose unclear | May have false confidence |
| Pheromone evolution | Feature exists but unused | Telemetry collected but not consumed |

---

## Gotchas & Learnings

### Shell Scripting

1. **awk apostrophes** - Use `'\''` escape in single-quoted awk scripts
2. **stat is platform-specific** - macOS: `stat -f %m`, Linux: `stat -c %Y`
3. **No jq dependency** - Session freshness uses bash string manipulation for JSON

### Colony Behavior

1. **Goals can contradict** - COLONY_STATE.json, events, and TO-DOs may have different goal phrasings
2. **Dreams are not actions** - Dream journal has great insights but they're rarely enacted
3. **Tests pass != tests meaningful** - cli-telemetry.test.js and cli-override.test.js purpose unclear

### File Boundaries

```
NEVER TOUCH (user data):
  .aether/data/     - Colony state
  .aether/dreams/   - Dream journal
  .aether/oracle/   - Research progress
  TO-DOs.md         - User notes

SAFE TO MODIFY (system files):
  .aether/*.md      - workers.md, docs
  .aether/*.sh      - aether-utils.sh, utils/
  .claude/commands/ - Slash commands
  .opencode/        - OpenCode agents/commands
```

---

## Verification Commands

```bash
# Verify commands in sync
npm run lint:sync

# Verify model routing
aether verify-models

# Run all tests
npm test

# Test session freshness
bash tests/bash/test-session-freshness.sh
```

---

## Quick Reference: Where Things Live

| What | Where |
|------|-------|
| Worker definitions | `.aether/workers.md` |
| Utility functions | `.aether/aether-utils.sh` |
| Slash commands (Claude) | `.claude/commands/ant/*.md` |
| Slash commands (OpenCode) | `.opencode/commands/ant/*.md` |
| Agent definitions | `.opencode/agents/*.md` |
| Colony state | `.aether/data/COLONY_STATE.json` |
| Known issues | `.aether/docs/known-issues.md` |
| Implementation learnings | `.aether/docs/implementation-learnings.md` |
| Development TODOs | `TO-DOS.md` (root) |
| Dream journal | `.aether/dreams/*.md` |

---

*Generated: 2026-02-16 | Update when architecture changes or bugs are fixed*
