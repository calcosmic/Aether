# Phase 0 Analysis: Aether Self-Reference Cleanup

## Executive Summary

This document provides a comprehensive analysis of the Aether codebase to identify self-referential components (Aether developing Aether) versus core infrastructure that works on other projects.

**Analysis Date:** 2026-02-15
**Analyst:** Claude Code (llm-architect agent)
**Confidence:** 95%

---

## 1. Area 1: Colony/Swarm State Files

### Summary
All state files in `.aether/data/` are **self-referential** - they track Aether's own development as a project using the Aether system.

### Detailed Analysis

| File | Category | Confidence | Reasoning |
|------|----------|------------|-----------|
| `COLONY_STATE.json` | **REMOVE** | 100% | Tracks Aether's own development goal: "Wire the QUEEN.md wisdom feedback loop...". Contains phase tracking for Aether v3.1 features. |
| `activity.log` | **REMOVE** | 100% | Logs colony events for Aether's own development |
| `spawn-tree.txt` | **REMOVE** | 100% | Tracks worker spawns during Aether development |
| `flags.json` | **REMOVE** | 100% | Project flags for Aether development |
| `constraints.json` | **REMOVE** | 100% | Self-imposed constraints during Aether development |
| `telemetry.json` | **REMOVE** | 100% | Model performance tracking during Aether dev |
| `swarm-display.json` | **REMOVE** | 100% | Visualization state for Aether development |
| `view-state.json` | **REMOVE** | 100% | UI state for Aether's own swarm display |
| `archive/` | **EXTRACT** | 90% | May contain valuable historical data |
| `backups/` | **REMOVE** | 100% | Auto-generated backups of self-referential state |

### Valuable Content to Extract
From COLONY_STATE.json `memory.phase_learnings`:
- Learning: "Claude Code global sync works by copying commands from .claude/commands/ to ~/.claude/commands/"
- Learning: "OpenCode requires repo-local setup"
- Learning: "Hash comparison prevents unnecessary file writes"
- Learning: "Namespace isolation via 'ant:' prefix prevents collisions"
- Learning: "CLI sync verification catches content drift"

---

## 2. Area 2: Oracle Research System

### Summary
The oracle research is **infrastructure** (KEEP) but contains findings that must be **EXTRACTED** before clearing.

### Files Analysis

| File | Category | Confidence | Reasoning |
|------|----------|------------|-----------|
| `oracle.sh` | **KEEP** | 95% | Infrastructure for deep research - works on any project |
| `oracle.md` | **KEEP** | 95% | Documentation for oracle system |
| `progress.md` | **EXTRACT** | 100% | Contains 12 BUGs, 7 ISSUEs, 10 GAPs found in Aether |
| `research.json` | **REMOVE** | 100% | State tracking current research topic |
| `.last-topic` | **REMOVE** | 100% | Session state |
| `archive/` | **EXTRACT** | 90% | Historical research may have findings |

### Critical Findings to Extract

#### BUGs Found (All unfixed, go to TODO.md)
| ID | Severity | File | Description |
|----|----------|------|-------------|
| BUG-002 | MEDIUM | aether-utils.sh | Missing release_lock in flag-add error path |
| BUG-003 | MEDIUM | atomic-write.sh | Race condition in backup creation |
| BUG-004 | MEDIUM | aether-utils.sh:930 | Missing error code in flag-acknowledge |
| BUG-005 | HIGH | aether-utils.sh:1022 | Missing lock release in flag-auto-resolve |
| BUG-006 | MEDIUM | atomic-write.sh:66 | No lock release on JSON validation failure |
| BUG-007 | MEDIUM | aether-utils.sh | 17+ instances of missing error codes |
| BUG-008 | MEDIUM | aether-utils.sh:856 | Missing error code in flag-add jq failure |
| BUG-009 | MEDIUM | aether-utils.sh:899,933 | Missing error codes in file checks |
| BUG-010 | MEDIUM | aether-utils.sh:1758+ | Missing error codes in context-update |
| BUG-011 | HIGH | aether-utils.sh:1022 | Missing error handling in flag-auto-resolve jq |
| BUG-012 | LOW | aether-utils.sh:2947 | Missing error code in unknown command |

#### ISSUEs Found
| ID | Severity | Description |
|----|----------|-------------|
| ISSUE-001 | MEDIUM | Inconsistent error code usage (systemic) |
| ISSUE-002 | LOW | Missing exec error handling (model-get/list) |
| ISSUE-003 | LOW | Incomplete help command |
| ISSUE-004 | MEDIUM | Template path hardcoded to runtime/ |
| ISSUE-005 | LOW | Potential infinite loop edge case (spawn-tree) |
| ISSUE-006 | LOW | Fallback json_err incompatible with enhanced signature |
| ISSUE-007 | LOW | Feature detection race condition |

#### GAPs Found
| ID | Severity | Description |
|----|----------|-------------|
| GAP-001 | MEDIUM | No validation of COLONY_STATE.json schema version |
| GAP-002 | LOW | No cleanup for stale spawn-tree.txt entries |
| GAP-003 | MEDIUM | No retry logic for failed worker spawns |
| GAP-004 | LOW | Missing queen-* documentation |
| GAP-005 | MEDIUM | No validation of queen-read JSON output |
| GAP-006 | LOW | Missing queen-* command documentation |
| GAP-007 | LOW | No error code standards documentation |
| GAP-008 | MEDIUM | Missing error path test coverage |
| GAP-009 | LOW | context-update has no file locking |
| GAP-010 | MEDIUM | Missing error code standards documentation |

---

## 3. Area 3: Colony Workflow Commands

### Summary
Commands are **infrastructure** that work on ANY repo - they should be **KEPT**.

### Detailed Analysis

| Command | Category | Works on Other Repos | Notes |
|---------|----------|---------------------|-------|
| `init.md` | **KEEP** | YES | Initializes any project with colony state |
| `build.md` | **KEEP** | YES | Builds phases for any project |
| `continue.md` | **KEEP** | YES | Resumes work on any colony |
| `seal.md` | **KEEP** | YES | Archives any completed colony |
| `entomb.md` | **KEEP** | YES | Archives colony to chambers |
| `swarm.md` | **KEEP** | YES | Real-time visualization for any colony |
| `organize.md` | **KEEP** | YES | Codebase hygiene for any project |
| `colonize.md` | **KEEP** | YES | Territory survey for any project |
| `plan.md` | **KEEP** | YES | Phase planning for any project |
| `status.md` | **KEEP** | YES | Shows colony status |
| `phase.md` | **KEEP** | YES | Phase details |
| `oracle.md` | **KEEP** | YES | Deep research for any project |
| `lay-eggs.md` | **KEEP** | YES | Fresh colony start |
| `tunnels.md` | **KEEP** | YES | Browse archived colonies |
| `watch.md` | **KEEP** | YES | Real-time monitoring |
| `verify-castes.md` | **KEEP** | YES | Model verification |
| `history.md` | **KEEP** | YES | Colony event history |
| `maturity.md` | **KEEP** | YES | Milestone visualization |
| `focus.md` | **KEEP** | YES | Pheromone signals |
| `feedback.md` | **KEEP** | YES | Pheromone signals |
| `redirect.md` | **KEEP** | YES | Pheromone signals |
| `flag.md` | **KEEP** | YES | Blocker tracking |
| `flags.md` | **KEEP** | YES | List flags |
| `council.md` | **KEEP** | YES | Intent clarification |
| `dream.md` | **KEEP** | YES | Philosophical exploration |
| `interpret.md` | **KEEP** | YES | Ground dreams in reality |
| `help.md` | **KEEP** | YES | Command reference |
| `pause-colony.md` | **KEEP** | YES | Create handoff document |
| `resume-colony.md` | **KEEP** | YES | Resume from handoff |
| `update.md` | **KEEP** | YES | Update system files |
| `migrate-state.md` | **KEEP** | YES | State migration utility |
| `chaos.md` | **KEEP** | YES | Edge case testing |
| `archaeology.md` | **KEEP** | YES | Git history excavation |

---

## 4. Area 4: Agent Definitions

### Summary
Agents are **infrastructure** - they work on any project. **KEEP ALL**.

### Detailed Analysis

| Agent | Category | For Aether-Itself | For Other Projects | Notes |
|-------|----------|-------------------|-------------------|-------|
| `aether-queen.md` | **KEEP** | NO | YES | Prime coordination agent |
| `aether-builder.md` | **KEEP** | NO | YES | Implementation work |
| `aether-watcher.md` | **KEEP** | NO | YES | Monitoring |
| `aether-scout.md` | **KEEP** | NO | YES | Research |
| `aether-chaos.md` | **KEEP** | NO | YES | Edge case testing |
| `aether-oracle.md` | **KEEP** | NO | YES | Deep research (RALF loop) |
| `aether-architect.md` | **KEEP** | NO | YES | Planning |
| `aether-route-setter.md` | **KEEP** | NO | YES | Direction setting |
| `aether-archaeologist.md` | **KEEP** | NO | YES | Git history |
| `aether-ambassador.md` | **KEEP** | NO | YES | API integration |
| `aether-auditor.md` | **KEEP** | NO | YES | Code review |
| `aether-chronicler.md` | **KEEP** | NO | YES | Documentation |
| `aether-gatekeeper.md` | **KEEP** | NO | YES | Dependencies |
| `aether-guardian.md` | **KEEP** | NO | YES | Security audits |
| `aether-includer.md` | **KEEP** | NO | YES | Accessibility |
| `aether-keeper.md` | **KEEP** | NO | YES | Knowledge curation |
| `aether-measurer.md` | **KEEP** | NO | YES | Performance |
| `aether-probe.md` | **KEEP** | NO | YES | Test generation |
| `aether-sage.md` | **KEEP** | NO | YES | Analytics |
| `aether-tracker.md` | **KEEP** | NO | YES | Bug investigation |
| `aether-weaver.md` | **KEEP** | NO | YES | Refactoring |
| `aether-surveyor-*.md` | **KEEP** | NO | YES | Specialized surveyors |
| `workers.md` | **KEEP** | NO | YES | Worker definitions reference |

---

## 5. Area 5: aether-utils.sh Functions

### Summary
Most functions are **core infrastructure** and should be **KEPT**. Some colony-tracking functions may be removable.

### Functions to KEEP (Core Infrastructure)

```
json_ok, json_err                    - JSON response helpers
atomic_write                         - File operations
acquire_lock, release_lock           - File locking
get_caste_emoji                      - Emoji helper

validate-state                       - State validation
error-add, error-pattern-check       - Error tracking
activity-log*                        - Activity logging

spawn-log, spawn-complete            - Spawn tracking
spawn-can-spawn, spawn-get-depth     - Spawn management
spawn-tree-*                         - Spawn tree operations

learning-promote, learning-inject    - Learning system

check-antipattern                    - Pattern detection
error-flag-pattern                   - Error flagging
signature-scan, signature-match      - Signature operations

flag-add, flag-resolve               - Flag management
flag-acknowledge, flag-list          - Flag operations
flag-auto-resolve                    - Auto-resolution
flag-check-blockers                  - Blocker checking

autofix-checkpoint, autofix-rollback - Autofix operations

swarm-findings-init                  - Swarm findings
swarm-findings-add, swarm-findings-read
swarm-solution-set, swarm-cleanup
swarm-activity-log
swarm-display-init, swarm-display-update, swarm-display-get
swarm-timing-start, swarm-timing-get, swarm-timing-eta

view-state-init, view-state-get      - View state
view-state-set, view-state-toggle
view-state-expand, view-state-collapse

grave-add, grave-check               - Graveyard operations

generate-commit-message              - Git helper
version-check                        - Version checking
registry-add                         - Registry operations
bootstrap-system                     - System bootstrap

model-profile, model-get, model-list - Model routing

caste-models-list, caste-models-set  - Caste configuration
caste-models-reset

chamber-create, chamber-verify       - Chamber operations
chamber-list

milestone-detect                     - Milestone detection

queen-init, queen-read, queen-promote - QUEEN.md system

survey-load, survey-verify           - Survey operations

generate-ant-name                    - Name generation
```

### Functions to EXTRACT from then REMOVE
None - all functions serve infrastructure purposes that work on any project.

### Functions UNCLEAR
None identified.

---

## 6. Area 6: Documentation

### Summary
Documentation is a mix of **general** (KEEP) and **self-referential** (EXTRACT/REMOVE).

### Detailed Analysis

| File | Category | Content Type | Notes |
|------|----------|--------------|-------|
| `docs/README.md` | **KEEP** | General | Main documentation |
| `docs/pheromones.md` | **KEEP** | General | Pheromone system guide |
| `docs/constraints.md` | **KEEP** | General | Constraint system |
| `docs/biological-reference.md` | **KEEP** | General | Caste taxonomy |
| `docs/command-sync.md` | **KEEP** | General | Sync documentation |
| `docs/namespace.md` | **KEEP** | General | Namespace reference |
| `docs/progressive-disclosure.md` | **KEEP** | General | UI patterns |
| `docs/pathogen-schema.md` | **KEEP** | General | Error schema |
| `docs/PHEROMONE-*.md` | **KEEP** | General | System design docs |
| `docs/VISUAL-OUTPUT-SPEC.md` | **KEEP** | General | Visual spec |
| `docs/MULTI-COLONY-ARCHITECTURE.md` | **KEEP** | General | Architecture |
| `docs/implementation/*.md` | **KEEP** | General | Implementation guides |
| `docs/reference/*.md` | **KEEP** | General | Reference docs |
| `workers.md` | **KEEP** | General | Worker definitions |
| `coding-standards.md` | **KEEP** | General | Standards doc |
| `learning.md` | **EXTRACT** | Both | Extract Aether-specific learnings |
| `planning.md` | **REMOVE** | Self-reference | Aether's own planning |
| `tdd.md` | **KEEP** | General | TDD methodology |
| `debugging.md` | **KEEP** | General | Debug methodology |
| `verification.md` | **KEEP** | General | Verification loop |
| `verification-loop.md` | **KEEP** | General | Verification details |
| `DISCIPLINES.md` | **KEEP** | General | Discipline guide |
| `RESEARCH-SHARED-DATA.md` | **EXTRACT** | Self-reference | Research findings |
| `QUEEN.md` | **EXTRACT** | Self-reference | Queen's wisdom for Aether - MOVE to docs/ |
| `QUEEN_ANT_ARCHITECTURE.md` | **KEEP** | General | Architecture doc |
| `RECOVERY-PLAN.md` | **EXTRACT** | Self-reference | Recovery procedures - MOVE to docs/ |
| `recover.sh` | **KEEP** | Infrastructure | Recovery script |

---

## 7. Area 7: Dependency Graph

### Summary
The system has clean separation between:
1. **Core utilities** (aether-utils.sh functions) - used by all commands
2. **State files** (data/*) - read/written by utilities
3. **Commands** - call utilities, don't directly touch state
4. **Agents** - spawned by commands

### Critical Dependencies

```
Commands (.claude/commands/ant/*.md)
    |
    ├── call ──> aether-utils.sh functions
    |               |
    |               ├── read/write ──> data/COLONY_STATE.json
    |               ├── read/write ──> data/activity.log
    |               ├── read/write ──> data/flags.json
    |               └── read/write ──> data/spawn-tree.txt
    |
    └── spawn ──> Agents (.opencode/agents/*.md)
```

### Safe Removal Order

1. **Extract** findings from oracle/progress.md → TODO.md
2. **Extract** learnings from COLONY_STATE.json → TODO.md
3. **Backup** then remove data/ files:
   - COLONY_STATE.json
   - activity.log
   - spawn-tree.txt
   - flags.json
   - constraints.json
   - telemetry.json
   - swarm-display.json
   - view-state.json
   - archive/ (after checking)
4. **Clear** oracle research state:
   - research.json
   - .last-topic
5. **Remove** self-referential docs:
   - planning.md
6. **Extract** then remove:
   - learning.md (extract to implementation-learnings.md)
   - RESEARCH-SHARED-DATA.md (extract findings)
7. **Move** to docs/:
   - QUEEN.md → docs/QUEEN.md
   - RECOVERY-PLAN.md → docs/RECOVERY-PLAN.md

---

## 8. Final Deliverables

### 8.1 Files to DELETE

```
.aether/data/COLONY_STATE.json
.aether/data/activity.log
.aether/data/activity-phase-1.log
.aether/data/spawn-tree.txt
.aether/data/spawn-tree-phase1.txt
.aether/data/flags.json
.aether/data/flags.json.backup
.aether/data/constraints.json
.aether/data/telemetry.json
.aether/data/timing.log
.aether/data/swarm-display.json
.aether/data/swarm-findings-*.json
.aether/data/view-state.json
.aether/data/watch-progress.txt
.aether/data/watch-status.txt
.aether/data/completion-report.md
.aether/data/codebase.md
.aether/data/survey-system-plan.md
.aether/data/archive/*
.aether/data/backups/*
.aether/data/swarm-archive/*
.aether/oracle/research.json
.aether/oracle/.last-topic
.aether/oracle/archive/*
.aether/planning.md
```

### 8.2 Files to KEEP (All Core Infrastructure)

```
.aether/aether-utils.sh
.aether/utils/*.sh
.aether/workers.md
.aether/workers-new-castes.md
.aether/coding-standards.md
.aether/debugging.md
.aether/tdd.md
.aether/verification.md
.aether/verification-loop.md
.aether/DISCIPLINES.md
.aether/QUEEN_ANT_ARCHITECTURE.md
.aether/recover.sh
.aether/model-profiles.yaml
.aether/oracle/oracle.sh
.aether/oracle/oracle.md
.aether/docs/* (all)
.aether/visualizations/* (all)
.aether/utils/* (all)
.claude/commands/ant/* (all)
.opencode/commands/ant/* (all)
.opencode/agents/* (all)
```

### 8.3 Files to MOVE

```
.aether/QUEEN.md → .aether/docs/QUEEN.md
.aether/RECOVERY-PLAN.md → .aether/docs/RECOVERY-PLAN.md
```

### 8.4 Content to EXTRACT

Create `.aether/docs/TODO.md`:

```markdown
# Aether System TODOs

## Unfixed Bugs
- BUG-002: Missing release_lock in flag-add error path (aether-utils.sh:814)
- BUG-003: Race condition in backup creation (atomic-write.sh:75)
- BUG-004: Missing error code in flag-acknowledge (aether-utils.sh:930)
- BUG-005: Missing lock release in flag-auto-resolve (aether-utils.sh:1022) [HIGH]
- BUG-006: No lock release on JSON validation failure (atomic-write.sh:66)
- BUG-007: 17+ instances of missing error codes (aether-utils.sh various)
- BUG-008: Missing error code in flag-add jq failure (aether-utils.sh:856)
- BUG-009: Missing error codes in file checks (aether-utils.sh:899,933)
- BUG-010: Missing error codes in context-update (aether-utils.sh:1758+)
- BUG-011: Missing error handling in flag-auto-resolve jq (aether-utils.sh:1022) [HIGH]
- BUG-012: Missing error code in unknown command (aether-utils.sh:2947)

## Unfixed Issues
- ISSUE-001: Inconsistent error code usage across codebase
- ISSUE-002: Missing exec error handling in model-get/list
- ISSUE-003: Incomplete help command
- ISSUE-004: Template path hardcoded to runtime/
- ISSUE-005: Potential infinite loop edge case in spawn-tree
- ISSUE-006: Fallback json_err incompatible with enhanced signature
- ISSUE-007: Feature detection race condition

## Architecture Gaps
- GAP-001: No validation of COLONY_STATE.json schema version
- GAP-002: No cleanup for stale spawn-tree.txt entries
- GAP-003: No retry logic for failed worker spawns
- GAP-004: Missing queen-* documentation
- GAP-005: No validation of queen-read JSON output
- GAP-006: Missing queen-* command documentation
- GAP-007: No error code standards documentation
- GAP-008: Missing error path test coverage
- GAP-009: context-update has no file locking
- GAP-010: Missing error code standards documentation
```

Create `.aether/docs/implementation-learnings.md`:

```markdown
# Implementation Learnings

From Aether v3.1 development:

1. Claude Code global sync works by copying commands from .claude/commands/ to ~/.claude/commands/
2. OpenCode requires repo-local setup - each repo that wants ant commands must set them up locally
3. Hash comparison prevents unnecessary file writes and preserves file timestamps
4. Namespace isolation via 'ant:' prefix prevents collisions with other agents
5. CLI sync verification (generate-commands.sh check) catches content drift using SHA-1 checksums
```

### 8.5 UNCLEAR Items

None identified - all components have clear categorization.

---

## 9. Risk Assessment

### What Could Break?

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Missing data dependencies | Low | All data/ files are self-contained state |
| Commands depend on specific state | Low | Commands call utilities, not direct file access |
| Archive data loss | Low | Extract valuable findings first |
| Oracle research lost | Low | progress.md contains all findings |

### Verification Steps After Cleanup

1. Run `npm test` - verify all tests pass
2. Run `npm run lint:sync` - verify command sync
3. Run `aether verify-models` - verify model routing
4. Create test colony in fresh directory:
   ```
   mkdir /tmp/test-colony && cd /tmp/test-colony
   /ant:init "Test project"
   /ant:status
   ```

---

## 10. Execution Checklist

- [ ] Create TODO.md with all BUGs/ISSUEs/GAPs
- [ ] Create implementation-learnings.md
- [ ] Backup data/ directory
- [ ] Delete data/ files (except directory itself)
- [ ] Delete oracle state files
- [ ] Delete planning.md
- [ ] Move QUEEN.md and RECOVERY-PLAN.md to docs/
- [ ] Run verification tests
- [ ] Create test colony to confirm system works

---

*Analysis Complete - Ready for Phase 0 Execution*
