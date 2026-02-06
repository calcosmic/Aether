# Phase 33: State Foundation - Research

**Researched:** 2026-02-06
**Domain:** JSON state file consolidation / command file refactoring
**Confidence:** HIGH

## Summary

Phase 33 consolidates 6 distributed state files into a single `COLONY_STATE.json` file. This is the first phase of the v5.1 System Simplification milestone, driven by the M4L-AnalogWave postmortem which identified that 70% of context was consumed by framework overhead and state fell out of sync at context boundaries.

The current system stores state across 6 files in `.aether/data/`:
- `COLONY_STATE.json` - Colony goal, state machine status, worker statuses, spawn outcomes
- `PROJECT_PLAN.json` - Phases, tasks, success criteria
- `pheromones.json` - Active signals (INIT, FOCUS, REDIRECT, FEEDBACK)
- `memory.json` - Phase learnings, decisions, patterns
- `errors.json` - Error records, flagged patterns
- `events.json` - Event log (colony_initialized, phase_started, etc.)

Every command reads 3-6 of these files and writes to 2-4 of them, creating multiple file operations per command. The consolidation reduces this to one read and one write per command.

**Primary recommendation:** Merge all 6 files into a single JSON structure with top-level keys for each domain, then update all 12 command files to use the new unified structure.

## Standard Stack

This phase involves no external libraries. All work is JSON schema design and Claude Code command file refactoring.

### Core
| Component | Purpose | Why Standard |
|-----------|---------|--------------|
| JSON | State storage format | Already in use, native to JavaScript/Node ecosystem |
| Claude Code commands | Command definitions | Existing system architecture |
| Bash (optional) | Migration script | Shell scripting for one-time migration |

### Supporting
None needed - pure refactoring work.

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Single JSON | SQLite | Overkill for this use case, adds dependency |
| Single JSON | YAML | JSON preferred for programmatic read/write |
| Migration script | Manual migration | Script is safer, repeatable |

## Architecture Patterns

### Recommended Consolidated Structure

```json
{
  "version": "2.0",
  "goal": "string",
  "state": "IDLE|READY|PLANNING|EXECUTING",
  "current_phase": 0,
  "session_id": "string",
  "initialized_at": "ISO-8601",
  "mode": "LIGHTWEIGHT|STANDARD|FULL|null",
  "mode_set_at": "ISO-8601|null",
  "mode_indicators": { "source_files": 0, "max_depth": 0, "languages": 0 },

  "workers": {
    "colonizer": "idle|active",
    "route-setter": "idle|active",
    "builder": "idle|active",
    "watcher": "idle|active",
    "scout": "idle|active",
    "architect": "idle|active"
  },

  "spawn_outcomes": {
    "colonizer": { "alpha": 1, "beta": 1, "total_spawns": 0, "successes": 0, "failures": 0 },
    ...
  },

  "plan": {
    "generated_at": "ISO-8601|null",
    "phases": [
      {
        "id": 1,
        "name": "string",
        "description": "string",
        "status": "pending|in_progress|completed|failed",
        "tasks": [
          {
            "id": "1.1",
            "description": "string",
            "status": "pending|in_progress|completed|failed",
            "depends_on": []
          }
        ],
        "success_criteria": ["string"]
      }
    ]
  },

  "signals": [
    {
      "id": "string",
      "type": "INIT|FOCUS|REDIRECT|FEEDBACK",
      "content": "string",
      "strength": 0.0-1.0,
      "half_life_seconds": "number|null",
      "created_at": "ISO-8601",
      "source": "string|null",
      "auto": "boolean|null"
    }
  ],

  "memory": {
    "phase_learnings": [
      {
        "id": "learn_timestamp_hex",
        "phase": 1,
        "phase_name": "string",
        "learnings": ["string"],
        "errors_encountered": 0,
        "timestamp": "ISO-8601"
      }
    ],
    "decisions": [
      {
        "id": "dec_timestamp_hex",
        "type": "colonization|plan|quality|focus|redirect|feedback",
        "content": "string",
        "context": "string",
        "phase": 0,
        "timestamp": "ISO-8601"
      }
    ],
    "patterns": []
  },

  "errors": {
    "records": [
      {
        "id": "err_timestamp_hex",
        "category": "syntax|import|runtime|type|spawning|phase|verification|api|file|logic|performance|security|build|test",
        "severity": "critical|high|medium|low",
        "description": "string",
        "root_cause": "string|null",
        "phase": "number|null",
        "task_id": "string|null",
        "timestamp": "ISO-8601"
      }
    ],
    "flagged_patterns": [
      {
        "category": "string",
        "count": 0,
        "first_seen": "ISO-8601",
        "last_seen": "ISO-8601",
        "flagged_at": "ISO-8601",
        "description": "string"
      }
    ]
  },

  "events": [
    "ISO-8601 | type | source | content"
  ]
}
```

### Key Design Decisions

**1. Events as Append-Only Strings**
Per SIMP-01 requirement, events become simple strings rather than structured objects. This reduces overhead and matches the append-only log semantics. Format: `"timestamp | type | source | content"`.

**2. Retain Nested Structure**
Rather than flattening everything, keep logical groupings (`plan`, `memory`, `errors`) as nested objects. This preserves semantic clarity while achieving single-file access.

**3. Version Field**
Add a `version: "2.0"` field to distinguish new format from old. Migration script checks version before processing.

**4. Retention Limits Preserved**
- `phase_learnings`: max 20 entries (per existing memory-compress)
- `decisions`: max 30 entries (per existing command logic)
- `events`: max 100 entries (per existing command logic)
- `errors.records`: max 50 entries (per existing command logic)

### Anti-Patterns to Avoid

- **Splitting reads across multiple steps:** Commands currently read files in Step 1 then read more in later steps. Consolidation means ONE read at command start.
- **Multiple writes per command:** Commands currently write 2-4 files. Consolidation means ONE write, typically at command end.
- **Redundant validation:** Current commands validate JSON structure at each read. With one file, validate once.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| JSON parsing | Custom parser | Native JSON.parse | Standard, tested |
| Migration script | Manual file editing | Shell script with jq | Repeatable, safe |
| State validation | Custom validator | JSON schema (optional) | Standard format |

**Key insight:** This phase is about simplifying, not adding capabilities. Resist temptation to add new features during consolidation.

## Common Pitfalls

### Pitfall 1: Forgetting File Access Points
**What goes wrong:** Missing a command that reads/writes state files, breaking it after migration
**Why it happens:** Commands are complex, state access scattered through steps
**How to avoid:** Systematic audit of all 12 command files before implementation
**Warning signs:** Command failures after migration

### Pitfall 2: Breaking Existing Data
**What goes wrong:** Migration script loses or corrupts existing colony data
**Why it happens:** Edge cases in current file formats not handled
**How to avoid:**
- Read all 6 existing files before writing new format
- Preserve all existing data during migration
- Test migration on existing `.aether/data/` directory
**Warning signs:** Empty arrays where data should exist post-migration

### Pitfall 3: Incomplete Command Updates
**What goes wrong:** Commands still reference old file paths
**Why it happens:** Search-and-replace misses some occurrences
**How to avoid:** Grep for all old file paths after refactoring
**Warning signs:** File not found errors, commands reading from old paths

### Pitfall 4: Event Format Migration
**What goes wrong:** Events in new string format incompatible with display logic
**Why it happens:** Display logic expects structured objects, gets strings
**How to avoid:** Update display logic to parse string format
**Warning signs:** Garbled event display in /ant:status

### Pitfall 5: Parallel File Reads Now Sequential
**What goes wrong:** Commands that previously read files in parallel now slower
**Why it happens:** Single file can't be read in parallel with itself
**How to avoid:** This is expected and acceptable - single file read is still fast
**Warning signs:** None - this is acceptable behavior

## Code Examples

### Current Pattern (to remove)
```
### Step 1: Read State

Use the Read tool to read these files (in parallel):
- `.aether/data/COLONY_STATE.json`
- `.aether/data/pheromones.json`
- `.aether/data/PROJECT_PLAN.json`
- `.aether/data/errors.json`
- `.aether/data/memory.json`
- `.aether/data/events.json`
```

### New Pattern (to implement)
```
### Step 1: Read State

Use the Read tool to read `.aether/data/COLONY_STATE.json`.

Extract from the state:
- goal, state, current_phase, session_id (top level)
- workers, spawn_outcomes (top level)
- plan.phases (for phase data)
- signals (for active pheromones)
- memory.phase_learnings, memory.decisions (for learnings)
- errors.records, errors.flagged_patterns (for error tracking)
- events (for event log - string array)
```

### Event String Format
```
Current (object):
{
  "id": "evt_1770139389_d228",
  "type": "phase_started",
  "source": "build",
  "content": "Phase 1: Foundation started",
  "timestamp": "2026-02-03T17:23:09Z"
}

New (string):
"2026-02-03T17:23:09Z | phase_started | build | Phase 1: Foundation started"
```

### Migration Script Structure
```bash
#!/bin/bash
# migrate-state.sh - One-time migration from 6-file to 1-file state

DATA_DIR=".aether/data"
OUTPUT="$DATA_DIR/COLONY_STATE.json.new"

# Read existing files
COLONY=$(cat "$DATA_DIR/COLONY_STATE.json" 2>/dev/null || echo '{}')
PLAN=$(cat "$DATA_DIR/PROJECT_PLAN.json" 2>/dev/null || echo '{"phases":[]}')
PHEROMONES=$(cat "$DATA_DIR/pheromones.json" 2>/dev/null || echo '{"signals":[]}')
MEMORY=$(cat "$DATA_DIR/memory.json" 2>/dev/null || echo '{}')
ERRORS=$(cat "$DATA_DIR/errors.json" 2>/dev/null || echo '{}')
EVENTS=$(cat "$DATA_DIR/events.json" 2>/dev/null || echo '{"events":[]}')

# Merge using jq (if available) or manual JSON construction
# ... construct new format ...

# Backup old files
mkdir -p "$DATA_DIR/backup-v1"
mv "$DATA_DIR/COLONY_STATE.json" "$DATA_DIR/backup-v1/" 2>/dev/null
# ... backup others ...

# Write new format
mv "$OUTPUT" "$DATA_DIR/COLONY_STATE.json"
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| 6 distributed files | Single consolidated file | v5.1 (this phase) | Reduces file I/O 6x |
| Structured event objects | Append-only strings | v5.1 (this phase) | Simpler, smaller |
| Multiple reads/writes per command | One read, one write | v5.1 (this phase) | Eliminates state sync issues |

**Deprecated/outdated:**
- `PROJECT_PLAN.json` as separate file - merged into COLONY_STATE.json
- `pheromones.json` as separate file - merged into COLONY_STATE.json
- `memory.json` as separate file - merged into COLONY_STATE.json
- `errors.json` as separate file - merged into COLONY_STATE.json
- `events.json` as separate file - merged into COLONY_STATE.json

## Commands to Update

Complete list of commands that access state files:

| Command | Files Read | Files Written |
|---------|------------|---------------|
| ant:init | COLONY_STATE | COLONY_STATE, errors, memory, events, pheromones |
| ant:status | COLONY_STATE, pheromones, PROJECT_PLAN, errors, memory, events | (display only) |
| ant:colonize | COLONY_STATE, pheromones | COLONY_STATE, memory, events, pheromones |
| ant:plan | COLONY_STATE, pheromones, PROJECT_PLAN | COLONY_STATE, PROJECT_PLAN |
| ant:build | COLONY_STATE, pheromones, PROJECT_PLAN, errors, events | COLONY_STATE, PROJECT_PLAN, errors, events, memory, pheromones |
| ant:continue | COLONY_STATE, pheromones, PROJECT_PLAN, errors, memory, events | COLONY_STATE, memory, pheromones, events |
| ant:focus | COLONY_STATE, pheromones, memory, events | pheromones, memory, events |
| ant:redirect | COLONY_STATE, pheromones, memory, events | pheromones, memory, events |
| ant:feedback | COLONY_STATE, pheromones, memory, events | pheromones, memory, events |
| ant:phase | COLONY_STATE, PROJECT_PLAN | (display only) |
| ant:pause-colony | COLONY_STATE, pheromones, PROJECT_PLAN | HANDOFF.md (not state) |
| ant:resume-colony | COLONY_STATE, pheromones, PROJECT_PLAN, HANDOFF.md | (display only) |
| ant:organize | COLONY_STATE, PROJECT_PLAN, pheromones, errors, memory, events | hygiene-report.md (not state) |

**Total: 13 commands to update** (including /ant:ant which is just help text)

## Implementation Order

Recommended order based on dependency and complexity:

1. **Design and validate new JSON schema** - Document exact structure
2. **Create migration script** - Convert existing 6-file state to new format
3. **Update ant:init** - Simplest command, creates initial state
4. **Update ant:status** - Read-only, good test of new format
5. **Update signal commands** (focus, redirect, feedback) - Similar patterns
6. **Update ant:phase** - Read-only, simple
7. **Update ant:pause-colony / ant:resume-colony** - Read-only display
8. **Update ant:plan** - Moderate complexity
9. **Update ant:colonize** - Moderate complexity
10. **Update ant:build** - Most complex command
11. **Update ant:continue** - Complex, depends on build changes
12. **Update ant:organize** - Read-heavy, moderate complexity
13. **Test end-to-end** - Full workflow validation

## Open Questions

1. **aether-utils.sh impact:** Several commands call shell utilities that operate on individual state files. These need updating or removal. This overlaps with SIMP-06 (Phase 37). Decision: Update only the utility calls used by state operations in this phase; full utility simplification in Phase 37.

2. **HANDOFF.md handling:** pause-colony writes to `.aether/HANDOFF.md` which is separate from state. This should remain separate as it's a human-readable document, not state. No change needed.

3. **activity.log handling:** organize command reads `.aether/data/activity.log`. This is a separate log file, not part of the 6-file state system. No change needed.

## Sources

### Primary (HIGH confidence)
- Current state files: `.aether/data/*.json` - Direct inspection
- Current command files: `.claude/commands/ant/*.md` - Direct inspection
- Requirements: `.planning/REQUIREMENTS.md` (SIMP-01)
- Roadmap: `.planning/ROADMAP.md` (Phase 33 definition)

### Secondary (MEDIUM confidence)
- v5 Field Notes: `.planning/v5-FIELD-NOTES.md` - Context on why simplification needed

### Tertiary (LOW confidence)
- None - all findings from direct codebase inspection

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - No external dependencies, pure JSON/command refactoring
- Architecture: HIGH - Schema derived from existing file structures
- Pitfalls: HIGH - Based on actual command file analysis
- Implementation order: MEDIUM - Reasonable but untested

**Research date:** 2026-02-06
**Valid until:** No expiration - internal refactoring, not external dependency
