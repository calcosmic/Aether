# TO-DOS

Pending work items for Aether development.

---

## Urgent

### Deprecate old 2.x npm versions - 2026-02-12

npm registry has stale 2.x pre-release versions visible on the npm page.

**Fix:** Run:
```bash
npm deprecate aether-colony@">=2.0.0 <3.0.0" "Pre-release versions. Install latest for stable release."
```

---

## High Priority

### Deeply Integrate XML System Into Core Commands - 2026-02-16

XML utilities exist but aren't integrated into the workflow.

**Goal:** XML should be the default storage format for pheromones, queen wisdom, and cross-colony sharing.

**Integration points:**
- `/ant:seal` and `/ant:entomb` should auto-export to XML
- Add `/ant:sniff` to read from eternal XML storage
- Add `/ant:share` for colony-to-colony transfer
- Auto-import eternal XML on colony init

---

### Convert Colony Prompts to XML Format - 2026-02-15

XML-structured prompts are more reliable than free-form markdown.

**Scope:**
1. Worker definitions (`.aether/workers.md`)
2. Command prompts (`.claude/commands/ant/*.md`)
3. Agent definitions (`.opencode/agents/*.md`)

---

### Empirically Verify Model Routing Works - 2026-02-14

Model routing infrastructure exists (`verify-castes` command, `spawn-with-model.sh`) but hasn't been proven to work end-to-end. Need to verify that spawned workers actually receive and use their assigned model.

**Test:** Run `/ant:verify-castes` and check if spawned worker reports correct `ANTHROPIC_MODEL`.

---

## Colony Lifecycle

### Multi-Ant Parallel Execution - 2026-02-13

Enable colony to run multiple ant commands simultaneously without conflicts.

**Problems to solve:**
- State conflicts (two ants modifying COLONY_STATE.json)
- File conflicts (two ants editing same file)
- Resource conflicts (tests/builds)
- Queen coordination

**Status:** DO NOT IMPLEMENT - discuss approach first

---

## UX Improvements

### Codebase Ant Pre-Flight Check - 2026-02-11

Automatic plan validation against current codebase before each phase executes. Catches plan/reality mismatches before wasted work.

**Note:** surveyor-pathogens agent exists for tech debt scanning, but pre-flight plan validation is a different concern.

---

## Enhancements

### YAML Command Generator - 2026-02-11

Eliminate manual duplication between `.claude/commands/ant/` and `.opencode/commands/ant/`. Build YAML-based generation system.

---

### Chamber Specialization (Code Zones) - 2026-02-10

Categorize codebase into behavioral zones during colonization:
- **Fungus Garden (core):** Extra caution, more testing
- **Nursery (new):** Okay to iterate fast
- **Refuse Pile (deprecated):** Avoid unless explicit

---

### Add Explicit Research Command (`/ant:forage`) - 2026-02-14

Create dedicated research command for structured domain analysis (separate from Oracle's deep research).

---

## Future Vision

Advanced colony concepts to explore:
1. **Colony Constitution** - Self-critique principles all ants reference
2. **Episodic Memory** - Full stories of how patterns were discovered
3. **Worker Quality Scores** - Reputation system for spawned workers
4. **Colony Sleep** - Memory consolidation during pause

---

## Questions to Resolve

### What is the point of /ant:status? - 2026-02-11

Evaluate whether `/ant:status` is actually useful or redundant with other commands.

---

## Completed

Items below have been implemented and verified in the codebase.

### Implement Archive/Seal Commands - 2026-02-13
**Done.** `seal.md`, `entomb.md`, and `history.md` all exist in `.claude/commands/ant/`. Milestone labels shown in `/ant:status`.

### Research and Implement Pheromone System - 2026-02-13
**Done.** 13+ pheromone subcommands implemented (`pheromone-write`, `pheromone-read`, `pheromone-count`, `pheromone-prime`, `pheromone-expire`, `pheromone-display`, `pheromone-export`, `pheromone-export-xml`, `pheromone-import-xml`, `pheromone-validate-xml`, plus `suggest-analyze`, `suggest-approve`). Content deduplication, prompt injection sanitization, and decay/expiration all operational.

### Apply Timestamp Verification to /ant:oracle - 2026-02-16
**Done.** `oracle.md` calls `session-verify-fresh` at both mid-process and post-process checkpoints.

### Session Continuity Marker - 2026-02-10
**Done.** `session.json` stored in `.aether/data/`, with `session-write`, `session-read`, and `/ant:resume` for instant context recovery.

### Smart Command Suggestion - 2026-02-10
**Done.** `print-next-up` subcommand in `aether-utils.sh` provides context-aware next command suggestions based on colony state.

### Surface Dreams in /ant:status - 2026-02-11
**Done.** `/ant:status` displays dream count and latest dream timestamp (Step 2.5 in status.md).

### Build summary displays before task-notification banners - 2026-02-12
**Done.** All worker spawns now use blocking Task calls instead of `run_in_background`, eliminating the ordering issue.

### Auto-Load Context on Colony Commands - 2026-02-10
**Done.** `colony-prime` subcommand assembles unified worker context (wisdom, pheromones, learnings, preferences) for all worker spawns. Token budget system trims to 8,000 chars (4,000 in compact mode).

### Immune Memory (Pathogen Recognition) - 2026-02-10
**Done.** Midden system tracks recurring failures (`midden-write`, `midden-recent-failures`). Auto-emits REDIRECT pheromones when failure patterns reach threshold. `surveyor-pathogens` agent scans for tech debt.

### Pheromone Evolution (Future Vision) - 2026-02-13
**Done.** Signals have configurable strength, TTL-based decay, and `pheromone-expire` for cleanup. Reinforcement via content deduplication increases strength on repeat signals.

### Self-Driving Mode (Future Vision) - 2026-02-13
**Done.** `/ant:run` provides full autopilot: builds, verifies, extracts learnings, advances through phases automatically with smart pause conditions.
