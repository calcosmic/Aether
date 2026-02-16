# Aether State Management System Analysis

**Generated:** 2026-02-16
**Scope:** Comprehensive analysis of state-related components in Aether
**Source Files:**
- `/Users/callumcowie/repos/Aether/.aether/data/COLONY_STATE.json`
- `/Users/callumcowie/repos/Aether/.aether/data/constraints.json`
- `/Users/callumcowie/repos/Aether/.aether/data/pheromones.json`
- `/Users/callumcowie/repos/Aether/.aether/data/checkpoint-allowlist.json`
- `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh`
- `/Users/callumcowie/repos/Aether/.aether/utils/file-lock.sh`
- `/Users/callumcowie/repos/Aether/.aether/docs/pheromones.md`
- `/Users/callumcowie/repos/Aether/.aether/docs/known-issues.md`

---

## 1. COLONY_STATE.json Structure

The colony state file is the central state repository for Aether operations.

### Schema (v3.0)

```json
{
  "version": "3.0",
  "goal": null,                    // Current colony goal (string or null)
  "state": "READY",                // Colony state: READY, BUILDING, PAUSED, ERROR
  "current_phase": 0,              // Current build phase (integer)
  "milestone": "First Mound",      // Current milestone (see milestone progression)
  "milestone_updated_at": "ISO8601",
  "session_id": null,              // Active session identifier
  "initialized_at": null,          // When colony was initialized
  "build_started_at": null,        // When current build started
  "plan": {
    "generated_at": null,
    "confidence": 0,               // 0-100 confidence score
    "phases": []                   // Array of phase objects
  },
  "memory": {
    "phase_learnings": [],         // Learnings from completed phases
    "decisions": [],               // Key decisions made
    "instincts": []                // Accumulated instincts
  },
  "errors": {
    "records": [],                 // Error history
    "flagged_patterns": []         // Recurring error patterns
  },
  "signals": [],                   // Active pheromone signals
  "graveyards": [],                // Archived/removed items
  "events": [],                    // Event log
  "created_at": "ISO8601",
  "last_updated": "ISO8601",
  "paused": false,
  "model_profile": {
    "active_profile": "default",
    "profile_file": ".aether/model-profiles.yaml",
    "routing_enabled": true,
    "proxy_endpoint": "http://localhost:4000",
    "updated_at": "ISO8601"
  }
}
```

### State Lifecycle

```
UNINITIALIZED → READY → BUILDING → (PAUSED) → READY/ERROR
     │            │        │          │
     │            │        └──────────┘ (can pause during build)
     │            │
     └────────────┘ (via /ant:init or /ant:colonize)
```

**States:**
- `UNINITIALIZED`: No COLONY_STATE.json exists
- `READY`: Colony ready to accept commands
- `BUILDING`: Active build in progress
- `PAUSED`: Colony paused (preserves TTLs)
- `ERROR`: Error state requiring attention

### Milestone Progression

```
First Mound → Open Chambers → Brood Stable → Ventilated Nest → Sealed Chambers → Crowned Anthill
```

---

## 2. Pheromone System

Pheromones are the user-colony communication mechanism. They use chemical signal metaphors to influence worker behavior.

### Signal Types

| Signal | Command | Priority | Default Expiration | Use For |
|--------|---------|----------|-------------------|---------|
| **FOCUS** | `/ant:focus "<area>"` | normal | phase end | "Pay attention to this" |
| **REDIRECT** | `/ant:redirect "<avoid>"` | high | phase end | "Don't do this" (hard constraint) |
| **FEEDBACK** | `/ant:feedback "<note>"` | low | phase end | "Adjust based on this" |

### Pheromone Schema

```json
{
  "version": "1.0.0",
  "colony_id": "aether-dev",
  "generated_at": "ISO8601",
  "signals": [
    {
      "id": "sig_focus_001",
      "type": "FOCUS|REDIRECT|FEEDBACK",
      "priority": "low|normal|high",
      "source": "user|worker:*|system|global:inject",
      "created_at": "ISO8601",
      "expires_at": "ISO8601|phase_end",
      "active": true,
      "content": {
        "text": "Signal content",
        "data": { /* optional structured data */ }
      },
      "tags": [
        {"value": "tag-name", "weight": 1.0, "category": "tech|constraint|quality|..."}
      ],
      "scope": {
        "global": false,
        "castes": ["builder", "architect"],
        "paths": [".aether/utils/*.sh"]
      }
    }
  ]
}
```

### Signal Combinations

| Combination | Effect |
|-------------|--------|
| FOCUS + FEEDBACK | Concentrate on focused area + adjust approach |
| FOCUS + REDIRECT | Prioritize focused area while avoiding pattern |
| FEEDBACK + REDIRECT | Adjust approach + avoid specific patterns |
| All three | Full steering: attention, avoidance, adjustment |

### TTL (Time-To-Live)

- **Default:** `phase_end` (expires when current phase completes)
- **Wall-clock:** Use `--ttl <duration>` flag
  - `30m` = 30 minutes
  - `2h` = 2 hours
  - `1d` = 1 day
- **Pause-aware:** Wall-clock TTLs are extended by pause duration

### Constraints System

Stored separately in `constraints.json`:

```json
{
  "version": "1.0",
  "focus": ["area1", "area2"],
  "constraints": [
    {
      "id": "c_xml_001",
      "type": "AVOID",
      "content": "Description of what to avoid",
      "source": "user:redirect|council:redirect",
      "created_at": "ISO8601"
    }
  ]
}
```

---

## 3. Checkpoint System

The checkpoint system creates recovery points before potentially destructive operations.

### Mechanism

**Command:** `autofix-checkpoint [label]`

1. **Check for changes** in Aether-managed directories only:
   - `.aether`
   - `.claude/commands/ant`
   - `.claude/commands/st`
   - `.opencode`
   - `runtime`
   - `bin`

2. **Create stash** if changes exist:
   - Stash name: `aether-checkpoint: <label>`
   - Only stashes allowlisted system files

3. **Return reference** for potential rollback:
   ```json
   {"type": "stash|commit|none", "ref": "..."}
   ```

### Rollback

**Command:** `autofix-rollback <type> <ref>`

- **stash type:** Finds and pops the named stash
- **commit type:** Resets to the recorded commit hash
- **none type:** No-op

### Checkpoint Allowlist

**File:** `.aether/data/checkpoint-allowlist.json`

```json
{
  "version": "1.0.0",
  "description": "Files safe for Aether to checkpoint/modify",
  "system_files": [
    ".aether/aether-utils.sh",
    ".aether/workers.md",
    ".aether/docs/**/*.md",
    ".claude/commands/ant/**/*.md",
    ".claude/commands/st/**/*.md",
    ".opencode/commands/ant/**/*.md",
    ".opencode/agents/**/*.md",
    "runtime/**/*",
    "bin/**/*"
  ],
  "user_data_never_touch": [
    ".aether/data/",
    ".aether/dreams/",
    ".aether/oracle/",
    ".aether/COLONY_STATE.json",
    "TO-DOs.md",
    "*.log",
    ".env",
    ".env.*"
  ]
}
```

**Critical Safety Rule:** User data is NEVER touched by checkpoint operations.

---

## 4. Session Freshness System

Prevents stale session files from silently breaking workflows.

### Commands Protected

| Command | Files Checked | Protected? |
|---------|---------------|------------|
| `survey` | PROVISIONS.md, TRAILS.md, BLUEPRINT.md, CHAMBERS.md, DISCIPLINES.md, SENTINEL-PROTOCOLS.md, PATHOGENS.md | No |
| `oracle` | progress.md, research.json | No |
| `watch` | watch-status.txt, watch-progress.txt | No |
| `swarm` | findings.json, display.json, timing.json | No |
| `init` | COLONY_STATE.json, constraints.json | **YES** |
| `seal` | manifest.json | **YES** |
| `entomb` | manifest.json | **YES** |

### Verification Flow

```bash
# 1. Capture session start time
SESSION_START=$(date +%s)

# 2. Before spawning agents, verify freshness
bash .aether/aether-utils.sh session-verify-fresh --command <name> $SESSION_START

# 3. If stale files detected:
#    - Non-protected: Auto-clear with warning
#    - Protected: Error (manual intervention required)

# 4. After spawning, verify files are fresh
```

### Response Format

```json
{
  "ok": true|false,
  "command": "survey|oracle|watch|swarm|init|seal|entomb",
  "fresh": ["file1.md", "file2.md"],
  "stale": ["old-file.md"],
  "missing": ["not-yet-created.md"],
  "total_lines": 150
}
```

### Protected Commands

Protected commands never auto-clear (contain precious data):
- `init` - COLONY_STATE.json is precious colony state
- `seal` - Archives are precious
- `entomb` - Chambers are precious

---

## 5. File Locking System

Implements concurrent access prevention for shared state files.

### Implementation

**File:** `.aether/utils/file-lock.sh`

```bash
# Configuration
LOCK_DIR="$AETHER_ROOT/.aether/locks"
LOCK_TIMEOUT=300          # 5 minutes max lock time
LOCK_RETRY_INTERVAL=0.5   # 500ms between retries
LOCK_MAX_RETRIES=100      # Total 50 seconds max wait
```

### Lock Mechanism

Uses atomic file creation with noclobber:

```bash
# Try to create lock file atomically
if (set -o noclobber; echo $$ > "$lock_file") 2>/dev/null; then
    echo $$ > "$lock_pid_file"
    export LOCK_ACQUIRED=true
    export CURRENT_LOCK="$lock_file"
    return 0
fi
```

### Stale Lock Detection

```bash
if [ -f "$lock_file" ]; then
    lock_pid=$(cat "$lock_pid_file" 2>/dev/null || echo "")
    if [ -n "$lock_pid" ]; then
        # Check if process is still running
        if ! kill -0 "$lock_pid" 2>/dev/null; then
            echo "Lock stale (PID $lock_pid not running), cleaning up..."
            rm -f "$lock_file" "$lock_pid_file"
        fi
    fi
fi
```

### API

| Function | Purpose |
|----------|---------|
| `acquire_lock <file_path>` | Acquire lock for resource |
| `release_lock` | Release current lock |
| `is_locked <file_path>` | Check if file is locked |
| `get_lock_holder <file_path>` | Get PID of lock holder |
| `wait_for_lock <file_path> [max_wait]` | Wait for lock release |
| `cleanup_locks` | Cleanup on exit (trap registered) |

### Graceful Degradation

If file locking is unavailable, operations proceed with warning:

```bash
if type feature_enabled &>/dev/null && ! feature_enabled "file_locking"; then
    json_warn "W_DEGRADED" "File locking disabled - proceeding without lock"
else
    acquire_lock "$flags_file" || json_err "$E_LOCK_FAILED" "..."
fi
```

---

## 6. Known Bugs

### Critical (Fix Immediately)

#### BUG-005: Missing lock release in flag-auto-resolve
- **Location:** `.aether/aether-utils.sh:1022` (lines 1368-1370 in current)
- **Severity:** HIGH
- **Symptom:** If jq command fails during flag resolution, lock is never released
- **Impact:** Deadlock on flags.json if jq fails (malformed JSON, disk full, etc.)
- **Workaround:** Restart colony session if commands hang on flag operations
- **Fix:** Add error handling with lock release before json_err

**Problematic Code:**
```bash
count=$(jq --arg trigger "$trigger" '
  [.flags[] | select(.auto_resolve_on == $trigger and .resolved_at == null)] | length
' "$flags_file") || {
  release_lock "$flags_file" 2>/dev/null || true
  json_err "$E_JSON_INVALID" "Failed to count flags for auto-resolve"
}
```

Note: The error handling exists but relies on `||` which may not catch all jq failures.

#### BUG-011: Missing error handling in flag-auto-resolve jq
- **Location:** Same as BUG-005
- **Severity:** HIGH
- **Combined Impact:** With BUG-005, causes deadlock

### Medium Priority

#### BUG-002: Missing release_lock in flag-add error path
- **Location:** `.aether/aether-utils.sh:814` (line ~1207 in current)
- **Severity:** MEDIUM
- **Symptom:** If acquire_lock succeeds but jq fails, lock may not be released
- **Fix:** Use trap-based cleanup or ensure release_lock in all exit paths

#### BUG-007: 17+ instances of missing error codes
- **Location:** Various lines in aether-utils.sh
- **Severity:** MEDIUM
- **Symptom:** Commands use hardcoded strings instead of `$E_*` constants
- **Pattern:** Early commands use strings, later commands use constants

#### BUG-008: Missing error code in flag-add jq failure
- **Location:** `.aether/aether-utils.sh:856` (line ~1209 in current)
- **Severity:** HIGH
- **Symptom:** Lock released but error code missing on jq failure

### Architecture Gaps

#### GAP-009: No file locking in context-update
- **Description:** Race condition possible during concurrent context updates
- **Impact:** Potential data corruption
- **Location:** context-update command (~line 1685)

#### GAP-002: No cleanup for stale spawn-tree entries
- **Description:** spawn-tree.txt grows indefinitely
- **Impact:** File could grow very large over many sessions

---

## 7. State-Related Commands

### COLONY_STATE Operations

| Command | Purpose | Location |
|---------|---------|----------|
| `colony-init` | Initialize new colony | aether-utils.sh:~1643 |
| `colony-goal` | Set/update colony goal | aether-utils.sh:~1694 |
| `colony-pause` | Pause colony | aether-utils.sh:~1747 |
| `colony-resume` | Resume colony | aether-utils.sh:~1786 |
| `context-update` | Update context | aether-utils.sh:~2182 |

### Flag Operations

| Command | Purpose | Lock? |
|---------|---------|-------|
| `flag-add` | Add new flag | Yes |
| `flag-check-blockers` | Count unresolved blockers | No |
| `flag-resolve` | Resolve a flag | Yes |
| `flag-acknowledge` | Acknowledge a flag | Yes |
| `flag-list` | List all flags | Yes |
| `flag-auto-resolve` | Auto-resolve on trigger | Yes |

### Session Operations

| Command | Purpose |
|---------|---------|
| `session-init` | Initialize session tracking |
| `session-verify-fresh` | Verify file freshness |
| `session-clear` | Clear session files |
| `checkpoint-check` | Verify checkpoint allowlist |

### Pheromone Operations

| Command | Purpose |
|---------|---------|
| `pheromone-export` | Export to eternal XML format |

---

## 8. Summary

### Strengths

1. **Explicit allowlist** prevents data loss (checkpoint system)
2. **Session freshness** prevents stale file issues
3. **Graceful degradation** when features unavailable
4. **Structured JSON** for all state operations
5. **Pheromone metaphor** provides intuitive user interface

### Weaknesses

1. **Lock release bugs** can cause deadlocks (BUG-005, BUG-011)
2. **Inconsistent error codes** across commands (BUG-007)
3. **Missing lock coverage** in some operations (GAP-009)
4. **No stale entry cleanup** for spawn-tree.txt (GAP-002)

### Recommendations

1. **Fix BUG-005/BUG-011 immediately** - Add comprehensive lock release in all error paths
2. **Standardize error codes** - Audit all json_err calls
3. **Add lock coverage** to context-update
4. **Implement spawn-tree cleanup** - Periodic compaction or size limit
5. **Document error codes** - Create error code reference

---

*Analysis complete. All state-related components documented.*
