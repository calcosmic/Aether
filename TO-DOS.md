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

### Apply Timestamp Verification to `/ant:oracle` - 2026-02-16

Oracle spawns long-running agents that can leave stale progress files if interrupted. Apply the same timestamp verification pattern used in `/ant:colonize`.

**Files:** `.aether/aether-utils.sh`, `.claude/commands/ant/oracle.md`

---

### Convert Colony Prompts to XML Format - 2026-02-15

XML-structured prompts are more reliable than free-form markdown.

**Scope:**
1. Worker definitions (`.aether/workers.md`)
2. Command prompts (`.claude/commands/ant/*.md`)
3. Agent definitions (`.opencode/agents/*.md`)

---

### Empirically Verify Model Routing Works - 2026-02-14

Model routing infrastructure exists but hasn't been proven to work. Need to verify that spawned workers actually receive and use their assigned model.

**Test:** Run `/ant:verify-castes` and check if spawned worker reports correct `ANTHROPIC_MODEL`.

---

## Colony Lifecycle

### Implement Archive/Seal Commands - 2026-02-13

Build ability to close/archive a colony at any time and re-initialize for new work.

**Design:**
1. `/ant:archive` — Archives current colony, writes completion report, resets state for fresh init
2. Milestone labels — Auto-detected status (First Mound, Brood Stable, etc.) shown in `/ant:status`
3. `/ant:history` — Browse archived colonies

---

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

### Build summary displays before task-notification banners - 2026-02-12

Phase summary appears before background agent notifications arrive. This is confusing even though data is correct.

**Possible fixes:** Don't use `run_in_background`, or add "Waiting for notifications..." step.

---

### Auto-Load Context on Colony Commands - 2026-02-10

Commands should automatically load relevant context (TO-DOs, colony state) especially after `/clear`.

---

### Surface Dreams in /ant:status - 2026-02-11

Show recent dream summary in colony status output.

---

### Codebase Ant Pre-Flight Check - 2026-02-11

Automatic plan validation against current codebase before each phase executes. Catches plan/reality mismatches before wasted work.

---

## Context Infrastructure

### Session Continuity Marker - 2026-02-10

Track last activity for seamless resume. Store lightweight session state for instant context recovery.

---

### Chamber Specialization (Code Zones) - 2026-02-10

Categorize codebase into behavioral zones during colonization:
- **Fungus Garden (core):** Extra caution, more testing
- **Nursery (new):** Okay to iterate fast
- **Refuse Pile (deprecated):** Avoid unless explicit

---

## Enhancements

### Smart Command Suggestion - 2026-02-10

Context-aware next command suggestions based on colony state.

---

### YAML Command Generator - 2026-02-11

Eliminate manual duplication between `.claude/commands/ant/` and `.opencode/commands/ant/`. Build YAML-based generation system.

---

### Immune Memory (Pathogen Recognition) - 2026-02-10

Track recurring bug patterns and escalate response when similar errors appear.

---

## Future Research

### Research and Implement Pheromone System - 2026-02-13

Pheromones are the colony's communication mechanism but implementation is incomplete. Need research and proper implementation.

---

### Add Explicit Research Command (`/ant:forage`) - 2026-02-14

Create dedicated research command for structured domain analysis (separate from Oracle's deep research).

---

## Future Vision

Advanced colony concepts to explore:
1. **Colony Constitution** - Self-critique principles all ants reference
2. **Episodic Memory** - Full stories of how patterns were discovered
3. **Pheromone Evolution** - Signals that strengthen/decay based on outcomes
4. **Worker Quality Scores** - Reputation system for spawned workers
5. **Colony Sleep** - Memory consolidation during pause
6. **Self-Driving Mode** - Autonomous overnight building sessions

---

## Questions to Resolve

### What is the point of /ant:status? - 2026-02-11

Evaluate whether `/ant:status` is actually useful or redundant with other commands.
