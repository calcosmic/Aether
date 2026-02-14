# Phase 5: State & Context Restoration - Research

**Researched:** 2026-02-14
**Domain:** State management, session persistence, spawn tree tracking
**Confidence:** HIGH

## Summary

This research covers the implementation of reliable cross-session memory and context for the Aether colony system. The system already has foundational state management via COLONY_STATE.json, file locking utilities, atomic writes, and validation mechanisms. This phase focuses on ensuring colony state loads on every command invocation, context restoration works after session pause/resume, and the spawn tree persists correctly across sessions.

**Primary recommendation:** Build on existing patterns in `aether-utils.sh` and `colony-state.test.js`. Implement state loading as a mandatory first step in all ant commands, use the existing file-lock + atomic-write infrastructure, and extend the current spawn-tree.txt format for full tree reconstruction.

## Standard Stack

### Core (Already Implemented)
| Component | Purpose | Location |
|-----------|---------|----------|
| COLONY_STATE.json | Primary state storage | `.aether/data/COLONY_STATE.json` |
| file-lock.sh | Concurrent access prevention | `.aether/utils/file-lock.sh` |
| atomic-write.sh | Corruption-safe writes | `.aether/utils/atomic-write.sh` |
| aether-utils.sh | State manipulation CLI | `.aether/aether-utils.sh` |
| validate-state | State validation command | `aether-utils.sh validate-state` |

### Supporting
| Component | Purpose | When to Use |
|-----------|---------|-------------|
| spawn-tree.txt | Spawn event log | `.aether/data/spawn-tree.txt` |
| activity.log | Human-readable activity | `.aether/data/activity.log` |
| flags.json | Blocker/issue tracking | `.aether/data/flags.json` |
| HANDOFF.md | Pause/resume context | `.aether/HANDOFF.md` |

### Validation Tools
| Tool | Purpose | Location |
|------|---------|----------|
| colony-state.test.js | JSON structure validation | `tests/unit/colony-state.test.js` |
| validate-state.test.js | CLI validation tests | `tests/unit/validate-state.test.js` |

## Architecture Patterns

### Recommended State Loading Sequence

Every ant command MUST follow this sequence:

```bash
# 1. Acquire lock (prevents concurrent modifications)
source .aether/utils/file-lock.sh
acquire_lock "$DATA_DIR/COLONY_STATE.json" || exit 1

# 2. Read state
state=$(cat "$DATA_DIR/COLONY_STATE.json")

# 3. Validate state
validation=$(bash .aether/aether-utils.sh validate-state colony)
if ! echo "$validation" | jq -e '.result.pass' >/dev/null 2>&1; then
  # Handle validation failure
  release_lock
  exit 1
fi

# 4. Reconstruct spawn tree (if needed)
# Parse spawn-tree.txt into in-memory structure

# 5. Check for paused state / handoff
if [[ -f "$AETHER_ROOT/.aether/HANDOFF.md" ]]; then
  # Display resumption context
  display_handoff_summary
fi

# 6. Execute command logic...

# 7. Release lock
release_lock
```

### State Validation Rules

Based on existing validation in `aether-utils.sh`:

**Required Fields (colony):**
- `goal`: null or string
- `state`: string (IDLE, READY, EXECUTING, PLANNING, COMPLETED)
- `current_phase`: number
- `plan`: object with phases array
- `memory`: object with phase_learnings, decisions, instincts
- `errors`: object with records array
- `events`: array of event objects

**Optional Fields:**
- `session_id`: string or null
- `initialized_at`: ISO-8601 string or null
- `build_started_at`: ISO-8601 string or null

**Validation Checks (from colony-state.test.js):**
1. No duplicate keys in JSON structure
2. Events in chronological order (timestamp ascending)
3. Required fields present and typed correctly
4. Event objects have timestamp, type, worker, details

### Spawn Tree Persistence Format

Current format in `spawn-tree.txt`:
```
2026-02-13T20:40:00Z|Queen|builder|Hammer-42|Implement auth|spawned
2026-02-13T20:45:00Z|Hammer-42|completed|Auth module done
```

**Extended format for full tree reconstruction:**
```
# Spawn events: timestamp|parent_id|caste|child_name|task|status
2026-02-13T20:40:00Z|Queen|builder|Hammer-42|Implement auth|spawned
2026-02-13T20:41:00Z|Hammer-42|scout|Swift-1|Research lib|spawned

# Completion events: timestamp|ant_name|status|summary
2026-02-13T20:45:00Z|Swift-1|completed|Library selected
2026-02-13T20:50:00Z|Hammer-42|completed|Auth module done
```

**Reconstruction algorithm:**
1. Parse all lines with status "spawned" to build parent-child relationships
2. Parse completion events to update statuses
3. Build in-memory tree: `{name: {caste, parent, children: [], status, task}}`

### Pause/Resume Handoff Pattern

**On Pause (`/ant:pause-colony`):**
1. Read COLONY_STATE.json
2. Gather active signals (non-expired pheromones)
3. Write `.aether/HANDOFF.md` with:
   - Goal, state, current_phase, session_id
   - Active pheromones list
   - Phase progress summary
   - Current phase tasks
   - What was happening
   - Next steps
4. Set `paused: true` flag in COLONY_STATE.json
5. Display confirmation

**On Resume (implicit in any command):**
1. Check for HANDOFF.md existence
2. If exists, display brief summary:
   ```
   🔄 Resuming: Phase X - Name
   Colony was working on: {summary from handoff}
   ```
3. Remove HANDOFF.md after successful display
4. Continue with command execution

### Context Restoration Tiers

| Tier | Format | When Used |
|------|--------|-----------|
| Brief | `🔄 Resuming: Phase X - Name` | Action commands (build, plan, continue) |
| Extended | Brief + Last activity timestamp | Status command |
| Full | Complete state with pheromones, workers | resume-colony command |

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| File locking | Custom lock implementation | `file-lock.sh` | Already handles stale lock cleanup, PID tracking, timeout |
| Atomic writes | Direct file write | `atomic-write.sh` | Temp file + mv pattern, JSON validation, backup rotation |
| State validation | Ad-hoc checks | `validate-state` command | Standardized field checking, JSON output, test coverage |
| JSON parsing in bash | Manual string manipulation | `jq` | Robust parsing, error handling, widely available |
| Timestamp generation | `date` without UTC | `date -u +%Y-%m-%dT%H:%M:%SZ` | ISO-8601 format, UTC timezone, consistent |
| Spawn tracking | Custom data structure | Extend `spawn-tree.txt` | Already integrated with activity logging |

## Common Pitfalls

### Pitfall 1: Forgetting to Release Lock
**What goes wrong:** Script exits mid-command, lock file remains, blocking future operations
**Why it happens:** Early exit on error without cleanup, signal interruption
**How to avoid:**
- Always use `trap cleanup_locks EXIT TERM INT` (pattern from file-lock.sh)
- Call `release_lock` in all exit paths
**Warning signs:** "Failed to acquire lock after 100 attempts" errors

### Pitfall 2: Validation Without Backup
**What goes wrong:** Corrupted state detected but no recovery path
**Why it happens:** Validation fails but script continues or exits without user option
**How to avoid:**
- Create backup before any repair attempt (atomic-write.sh does this)
- Offer user choices: auto-repair, start fresh (with backup), manual fix
- Never silently overwrite corrupted state

### Pitfall 3: Assuming State Exists
**What goes wrong:** Commands fail on uninitialized colonies with cryptic errors
**Why it happens:** Missing null checks on `goal` field
**How to avoid:**
- Check `goal: null` as first validation step
- Provide clear "No colony initialized" message with next steps
- Pattern from status.md: "No colony initialized. Run /ant:init first."

### Pitfall 4: Chronological Event Ordering
**What goes wrong:** Events appended with out-of-order timestamps
**Why it happens:** Clock changes, manual edits, race conditions
**How to avoid:**
- Always use `new Date().toISOString()` (monotonic if system clock stable)
- On load: verify events sorted, if not: log warning, sort in memory, flag for review
- Never retroactively modify event timestamps

### Pitfall 5: Spawn Tree Depth Miscalculation
**What goes wrong:** Wrong depth assigned, violating spawn limits
**Why it happens:** Parent not found in tree, circular references
**How to avoid:**
- Default to depth 1 if parent not found (pattern from spawn-get-depth)
- Safety limit of 5 levels in traversal
- Validate: depth 1 spawns count against depth 2's parent's limit

## Code Examples

### State Loading with Validation (Node.js)

```javascript
// Source: Pattern from cli.js error handling + aether-utils.sh validation
const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

class ColonyStateLoader {
  constructor(aetherRoot) {
    this.dataDir = path.join(aetherRoot, '.aether/data');
    this.stateFile = path.join(this.dataDir, 'COLONY_STATE.json');
  }

  load() {
    // Check file exists
    if (!fs.existsSync(this.stateFile)) {
      throw new RepoError('COLONY_STATE.json not found', { path: this.stateFile });
    }

    // Read and parse
    let state;
    try {
      const content = fs.readFileSync(this.stateFile, 'utf8');
      state = JSON.parse(content);
    } catch (err) {
      throw new ValidationError('Invalid JSON in COLONY_STATE.json', { error: err.message });
    }

    // Validate via aether-utils.sh
    const validation = this.validate();
    if (!validation.pass) {
      throw new ValidationError('State validation failed', { checks: validation.checks });
    }

    // Check for handoff (paused state)
    const handoffPath = path.join(this.dataDir, '..', 'HANDOFF.md');
    if (fs.existsSync(handoffPath)) {
      state._handoff = fs.readFileSync(handoffPath, 'utf8');
    }

    return state;
  }

  validate() {
    try {
      const output = execSync(`bash .aether/aether-utils.sh validate-state colony`, {
        encoding: 'utf8',
        cwd: process.cwd()
      });
      const result = JSON.parse(output);
      return result.result;
    } catch (err) {
      return { pass: false, error: err.message };
    }
  }
}
```

### State Loading with Validation (Bash)

```bash
# Source: Pattern from aether-utils.sh + file-lock.sh
load_colony_state() {
  local state_file="$DATA_DIR/COLONY_STATE.json"

  # Check file exists
  if [[ ! -f "$state_file" ]]; then
    json_err "$E_FILE_NOT_FOUND" "COLONY_STATE.json not found"
    return 1
  fi

  # Acquire lock
  if ! acquire_lock "$state_file"; then
    json_err "$E_LOCK_TIMEOUT" "Failed to acquire state lock"
    return 1
  fi

  # Validate before loading
  validation=$(bash "$SCRIPT_DIR/aether-utils.sh" validate-state colony 2>/dev/null)
  if ! echo "$validation" | jq -e '.result.pass' >/dev/null 2>&1; then
    release_lock
    json_err "$E_INVALID_STATE" "State validation failed"
    return 1
  fi

  # Load state into variable
  state=$(cat "$state_file")

  # Check for handoff
  if [[ -f "$AETHER_ROOT/.aether/HANDOFF.md" ]]; then
    handoff=$(cat "$AETHER_ROOT/.aether/HANDOFF.md")
    # Display brief resumption context
    echo "🔄 Resuming: Phase $(echo "$state" | jq -r '.current_phase')"
  fi

  # State is now loaded, lock is held - caller must release
  export LOADED_STATE="$state"
  export STATE_LOCK_ACQUIRED=true

  return 0
}

# Caller must call this when done
unload_colony_state() {
  if [[ "$STATE_LOCK_ACQUIRED" == "true" ]]; then
    release_lock
    STATE_LOCK_ACQUIRED=false
  fi
}
```

### Spawn Tree Reconstruction

```javascript
// Source: Pattern from spawn-get-depth in aether-utils.sh
class SpawnTree {
  constructor(spawnTreePath) {
    this.spawns = new Map(); // name -> {parent, children, caste, status, task}
    this.load(spawnTreePath);
  }

  load(path) {
    if (!fs.existsSync(path)) return;

    const lines = fs.readFileSync(path, 'utf8').split('\n').filter(Boolean);

    for (const line of lines) {
      const parts = line.split('|');

      if (parts.length === 6 && parts[5] === 'spawned') {
        // Spawn event: timestamp|parent|caste|child|task|spawned
        const [timestamp, parent, caste, child, task] = parts;
        this.spawns.set(child, {
          parent,
          caste,
          task,
          status: 'active',
          children: [],
          timestamp
        });

        // Update parent's children list
        if (this.spawns.has(parent)) {
          this.spawns.get(parent).children.push(child);
        }
      } else if (parts.length === 4) {
        // Completion event: timestamp|ant_name|status|summary
        const [timestamp, antName, status, summary] = parts;
        if (this.spawns.has(antName)) {
          this.spawns.get(antName).status = status;
          this.spawns.get(antName).completedAt = timestamp;
          this.spawns.get(antName).summary = summary;
        }
      }
    }
  }

  getDepth(antName) {
    if (antName === 'Queen') return 0;
    if (!this.spawns.has(antName)) return 1; // Default for unknown

    let depth = 1;
    let current = antName;

    while (depth < 5) { // Safety limit
      const spawn = this.spawns.get(current);
      if (!spawn || spawn.parent === 'Queen' || !spawn.parent) break;
      current = spawn.parent;
      depth++;
    }

    return depth;
  }

  getActiveSpawns() {
    return Array.from(this.spawns.entries())
      .filter(([_, s]) => s.status === 'active')
      .map(([name, s]) => ({ name, ...s }));
  }
}
```

### Event Timestamp Validation

```javascript
// Source: Pattern from colony-state.test.js
function verifyChronologicalOrder(events) {
  for (let i = 1; i < events.length; i++) {
    const prevTime = new Date(events[i - 1].timestamp).getTime();
    const currTime = new Date(events[i].timestamp).getTime();

    if (currTime < prevTime) {
      return {
        inOrder: false,
        firstOutOfOrder: {
          index: i,
          current: events[i],
          previous: events[i - 1]
        }
      };
    }
  }

  return { inOrder: true };
}
```

### Duplicate Key Detection

```javascript
// Source: colony-state.test.js detectDuplicateKeys function
function detectDuplicateKeys(jsonString) {
  const duplicates = [];
  const keyStack = [];
  let currentKeys = new Set();
  let inString = false;
  let escapeNext = false;
  let currentKey = '';
  let expectingKey = true;

  for (let i = 0; i < jsonString.length; i++) {
    const char = jsonString[i];

    if (escapeNext) {
      escapeNext = false;
      continue;
    }

    if (char === '\\') {
      escapeNext = true;
      continue;
    }

    if (char === '"' && !inString) {
      inString = true;
      currentKey = '';
      continue;
    }

    if (char === '"' && inString) {
      inString = false;
      if (expectingKey) {
        if (currentKeys.has(currentKey)) {
          duplicates.push(currentKey);
        } else {
          currentKeys.add(currentKey);
        }
        expectingKey = false;
      }
      continue;
    }

    if (inString) {
      currentKey += char;
      continue;
    }

    if (char === '{') {
      keyStack.push(currentKeys);
      currentKeys = new Set();
      expectingKey = true;
    } else if (char === '}') {
      currentKeys = keyStack.pop() || new Set();
      expectingKey = false;
    } else if (char === ',' && !inString) {
      expectingKey = true;
    }
  }

  return {
    hasDuplicates: duplicates.length > 0,
    duplicates: [...new Set(duplicates)]
  };
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Direct file writes | Atomic temp + mv | Phase 4 | Corruption-safe state updates |
| No validation | validate-state command | Phase 4 | Structured validation with JSON output |
| No locking | flock-based locking | Phase 4 | Prevents concurrent modification |
| Simple spawn log | Structured spawn-tree.txt | Phase 3 | Parent-child relationships tracked |
| No handoff | HANDOFF.md for pause/resume | Phase 5 (planned) | Session continuity |

**Deprecated/outdated:**
- State version "1.0" and "2.0": Auto-upgrade to "3.0" on load (pattern from status.md)
- Direct COLONY_STATE.json writes without atomic-write.sh: Use atomic_write function

## Open Questions

1. **Handoff.md Cleanup Timing**
   - What we know: Handoff should be removed after successful resume display
   - What's unclear: Should it be kept if resume fails? For how long?
   - Recommendation: Remove after successful display; if validation fails, keep handoff for manual recovery

2. **Spawn Tree Archival**
   - What we know: Completed spawns are retained for archaeology
   - What's unclear: At what size should old spawns be archived?
   - Recommendation: Archive spawns older than 30 days or when file exceeds 1000 lines

3. **State Validation Auto-Repair**
   - What we know: Validation can detect duplicate keys, timestamp issues
   - What's unclear: Which issues are safe to auto-repair vs require user decision?
   - Recommendation: Auto-repair sortable issues (timestamp order); user decision for data loss scenarios (duplicates)

## Sources

### Primary (HIGH confidence)
- `.aether/aether-utils.sh` - validate-state implementation, spawn tracking
- `.aether/utils/file-lock.sh` - Lock acquisition/release patterns
- `.aether/utils/atomic-write.sh` - Safe write patterns
- `tests/unit/colony-state.test.js` - Validation test patterns
- `tests/unit/validate-state.test.js` - CLI validation tests

### Secondary (MEDIUM confidence)
- `.claude/commands/ant/status.md` - State reading patterns, auto-upgrade
- `.claude/commands/ant/build.md` - State update patterns
- `.claude/commands/ant/pause-colony.md` - Handoff format
- `.claude/commands/ant/resume-colony.md` - Resume context tiers
- `bin/lib/errors.js` - Error handling patterns

### Tertiary (LOW confidence)
- `.aether/data/COLONY_STATE.json` - Current state format example
- `.aether/data/spawn-tree.txt` - Current spawn log format

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All components exist and are tested
- Architecture: HIGH - Patterns established in existing commands
- Pitfalls: MEDIUM - Based on code review, limited production runtime

**Research date:** 2026-02-14
**Valid until:** 2026-03-14 (30 days for stable patterns)
