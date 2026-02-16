# In-Conversation Swarm Display Design

> **Status:** ⏸️ BLOCKED - Waiting for XML migration to complete
> **Date:** 2026-02-16
> **Author:** Claude (with user input)
> **Goal:** Display swarm activity visually in the main Claude Code conversation, not in a separate terminal
> **Blocker:** spawn-tree.txt format may change to XML - see `.aether/schemas/` for XML work

---

## Executive Summary

**Current Problem:**
- Swarm display renders to a separate tmux/background terminal
- User sees raw Bash commands in conversation: `Bash(bash .aether/aether-utils.sh spawn-complete...)`
- The "cool" visual output (emojis, progress bars, swarm tree) is hidden in a separate display

**Desired Outcome:**
- Main Claude Code conversation shows rich visual output
- Emojis, progress bars, and swarm status appear directly in tool call results
- Future: Separate tmux display reserved for multi-colony monitoring

**Scope:**
- Modify `spawn-log`, `spawn-complete`, and related functions to return formatted output
- Create new `swarm-summary-print` function for in-conversation display
- Update build.md to use conversation-friendly output

---

## Current Architecture

### How Swarm Display Works Now

```
User runs /ant:build
       │
       ▼
build.md spawns workers via Task tool
       │
       ├── bash .aether/aether-utils.sh spawn-log "Queen" "builder" "Bolt-48" "task"
       │         │
       │         └── Writes to .aether/data/activity.log
       │         └── Writes to .aether/data/spawn-tree.txt
       │         └── Returns: {"ok": true, "result": "logged"}
       │
       ├── bash .aether/aether-utils.sh spawn-complete "Bolt-48" "completed" "summary"
       │         │
       │         └── Writes formatted line to activity.log with emoji
       │         └── Returns: {"ok": true, "result": "logged"}  ← PLAIN OUTPUT
       │
       └── bash .aether/aether-utils.sh swarm-display-render "$build_id"
                 │
                 └── Calls utils/swarm-display.sh
                 └── Uses `clear` and ANSI codes for terminal
                 └── Runs in background (causes exit code 144 errors)
```

### Key Files

| File | Purpose |
|------|---------|
| `.aether/aether-utils.sh` | Main utility with spawn-log, spawn-complete, swarm-display-render |
| `.aether/utils/swarm-display.sh` | Terminal rendering with ANSI codes, clear screen |
| `.claude/commands/ant/build.md` | Build command that calls swarm display functions |
| `.aether/data/activity.log` | Log file with formatted entries (has emojis) |
| `.aether/data/spawn-tree.txt` | Pipe-delimited spawn tree data |

### Current Output Examples

**What user sees in conversation:**
```
⏺ Bash(bash .aether/aether-utils.sh spawn-complete "Bolt-48" "completed" "Created XSD")
  ⎿  {"ok": true, "result": "logged"}
```

**What activity.log contains (hidden):**
```
[16:45:23] ✅ 🔨🐜 Bolt-48: completed - Created XSD
```

---

## Desired Architecture

### In-Conversation Display

```
User runs /ant:build
       │
       ▼
build.md spawns workers via Task tool
       │
       ├── bash .aether/aether-utils.sh spawn-log "Queen" "builder" "Bolt-48" "task"
       │         │
       │         └── Returns: {"ok": true, "result": "⚡ 🔨🐜 Bolt-48 spawned"}
       │
       ├── bash .aether/aether-utils.sh spawn-complete "Bolt-48" "completed" "Created XSD"
       │         │
       │         └── Returns: {"ok": true, "result": "✅ 🔨🐜 Bolt-48: Created XSD"}
       │
       └── bash .aether/aether-utils.sh swarm-summary-print "$build_id"
                 │
                 └── Returns multi-line formatted summary:
                     {
                       "ok": true,
                       "result": "🏛️🐜 Queen Prime-1
                                  ├── ✅ 🔨🐜 Bolt-48: Created XSD
                                  ├── ✅ 🔨🐜 Anvil-71: Created wisdom
                                  └── ⏳ 👁️🐜 Sharp-33: Verifying..."
                     }
```

### Future: Multi-Colony Tmux Dashboard

Reserved for monitoring multiple colonies simultaneously:
- Shows multiple .aether/ directories
- Real-time updates across projects
- Not for single-colony display

---

## Implementation Plan

### Phase 1: Modify Return Values (Low Risk)

**File:** `.aether/aether-utils.sh`

#### 1.1 Update `spawn-log` return value

**Current (line ~697):**
```bash
json_ok '"logged"'
```

**Proposed:**
```bash
json_ok "\"⚡ $emoji $child_name spawned\""
```

**Variables available:**
- `$emoji` - Already computed via `get_caste_emoji "$child_caste"`
- `$child_name` - The ant's name (e.g., "Bolt-48")
- `$child_caste` - The caste (e.g., "builder")

#### 1.2 Update `spawn-complete` return value

**Current (line ~715):**
```bash
json_ok '"logged"'
```

**Proposed:**
```bash
json_ok "\"$status_icon $emoji $ant_name: ${summary:-$status}\""
```

**Variables available:**
- `$status_icon` - ✅, ❌, or 🚫 based on status
- `$emoji` - Already computed via `get_caste_emoji "$ant_name"`
- `$ant_name` - The ant's name
- `$summary` - Optional summary text
- `$status` - completed/failed/blocked

**Already implemented:** I made these changes earlier in this session. They need to be verified and possibly refined.

### Phase 2: Create `swarm-summary-print` Function (Medium Risk)

**File:** `.aether/aether-utils.sh`

**Location:** After `swarm-display-render` case (around line 2405)

**Purpose:** Output a formatted swarm tree summary suitable for in-conversation display

**Pseudocode:**
```bash
swarm-summary-print)
  # Usage: swarm-summary-print [swarm_id]
  # Outputs a formatted tree of all spawn activity

  swarm_id="${1:-default-swarm}"
  spawn_tree="$DATA_DIR/spawn-tree.txt"

  if [[ ! -f "$spawn_tree" ]]; then
    json_ok '"No spawn activity recorded"'
    exit 0
  fi

  # Build tree from spawn-tree.txt
  # Format: timestamp|parent|caste|child_name|task|model|status

  summary=""
  while IFS='|' read -r ts parent caste child_name task model status; do
    emoji=$(get_caste_emoji "$caste")
    status_icon="⏳"
    [[ "$status" == "completed" ]] && status_icon="✅"
    [[ "$status" == "failed" ]] && status_icon="❌"

    # Build tree line with indentation based on parent
    indent=""
    if [[ "$parent" != "Queen" && "$parent" != "Colony" ]]; then
      indent="│  "
    fi

    summary+="${indent}${status_icon} ${emoji} ${child_name}: ${task}\n"
  done < "$spawn_tree"

  # Return as JSON string (may need escaping)
  json_ok "\"$summary\""
  ;;
```

**Key Considerations:**
1. JSON string escaping - newlines and quotes must be escaped
2. Tree indentation - need to track parent-child relationships
3. Sort order - chronological or hierarchical?
4. Max length - avoid extremely long output

### Phase 3: Update build.md to Use New Display (Low Risk)

**File:** `.claude/commands/ant/build.md`

**Changes:**

1. **Replace `swarm-display-render` with `swarm-summary-print`**
   - Line ~531: Change to `swarm-summary-print`
   - Line ~936: Change to `swarm-summary-print`

2. **Remove background execution**
   - Ensure display functions are not run with `run_in_background`
   - Background execution causes exit code 144 errors

3. **Add periodic summary output**
   - After each wave of workers, print a summary
   - This gives visual feedback during long builds

### Phase 4: Update swarm-display.sh for Future Multi-Colony Use (Low Priority)

**File:** `.aether/utils/swarm-display.sh`

**Changes:**
1. Keep existing terminal rendering for future multi-colony dashboard
2. Add `--json` flag to output machine-readable format
3. Add `--watch` flag for continuous monitoring mode

---

## Technical Considerations

### JSON String Escaping

The `json_ok` function needs to handle multi-line strings with special characters:

```bash
# Current json_ok implementation:
json_ok() {
  echo "{\"ok\":true,\"result\":$1}"
}

# Problem: Newlines break JSON
# Solution: Escape newlines and quotes
json_ok() {
  local result="$1"
  # Escape backslashes, quotes, and newlines
  result=$(echo "$result" | sed 's/\\/\\\\/g' | sed 's/"/\\"/g' | tr '\n' '|')
  echo "{\"ok\":true,\"result\":\"$result\"}"
}
```

**Or use a different approach:** Return a JSON array instead of a string:
```json
{"ok": true, "result": ["line1", "line2", "line3"]}
```

### ANSI Codes in Claude Code

Claude Code's interface may or may not render ANSI codes:
- **Safe approach:** Use plain text with emojis
- **Risky approach:** Use ANSI colors (may show raw codes)

**Recommendation:** Stick to emojis and basic formatting. Avoid ANSI codes for in-conversation output.

### Output Length Limits

Claude Code may truncate very long outputs:
- Keep summaries concise
- Consider pagination for large swarms
- Max recommended: 50-100 lines

### Backwards Compatibility

The `swarm-display-render` function should remain for:
- Future multi-colony dashboard
- Users who prefer terminal display
- Debugging purposes

**Do not remove** - just add `swarm-summary-print` as an alternative.

---

## Testing Plan

### Unit Tests

1. **Test `spawn-log` output format**
   ```bash
   bash .aether/aether-utils.sh spawn-log "Queen" "builder" "Test-1" "test task"
   # Expected: {"ok":true,"result":"⚡ 🔨🐜 Test-1 spawned"}
   ```

2. **Test `spawn-complete` output format**
   ```bash
   bash .aether/aether-utils.sh spawn-complete "Test-1" "completed" "test summary"
   # Expected: {"ok":true,"result":"✅ 🔨🐜 Test-1: test summary"}
   ```

3. **Test `swarm-summary-print`**
   ```bash
   # Create test spawn tree
   echo "2026-02-16T12:00:00Z|Queen|builder|Bolt-1|Create XSD|default|completed" > .aether/data/spawn-tree.txt
   bash .aether/aether-utils.sh swarm-summary-print
   # Expected: Formatted tree output
   ```

### Integration Tests

1. **Run `/ant:build` in test project**
   - Verify emojis appear in tool output
   - Verify no exit code 144 errors
   - Verify activity.log still works

2. **Run `/ant:watch` separately**
   - Verify terminal display still works
   - Verify multi-colony future compatibility

### Regression Tests

1. **Existing commands still work:**
   - `/ant:status` - colony status display
   - `/ant:phase` - phase details
   - `/ant:history` - event history

2. **Log files still populated:**
   - `.aether/data/activity.log` - formatted entries
   - `.aether/data/spawn-tree.txt` - spawn hierarchy

---

## Risk Assessment

### High Risk Areas

| Area | Risk | Mitigation |
|------|------|------------|
| JSON escaping | Malformed JSON breaks parsing | Test thoroughly, use proven escaping |
| Output length | Truncation loses information | Keep summaries concise |
| Existing workflows | Breaking `/ant:watch` users | Keep both functions available |

### Medium Risk Areas

| Area | Risk | Mitigation |
|------|------|------------|
| Emoji rendering | Some terminals don't support emojis | Already using emojis widely |
| Performance | Large swarms slow output | Limit output to recent activity |
| Sync to runtime | Changes must propagate | Test `npm install -g .` flow |

### Low Risk Areas

| Area | Risk | Mitigation |
|------|------|------------|
| build.md changes | Non-breaking additions only | Keep existing commands working |
| Documentation | May become outdated | Update docs with code changes |

---

## Rollback Plan

### If Phase 1 Causes Issues

```bash
# Revert spawn-log and spawn-complete to plain output
git checkout HEAD -- .aether/aether-utils.sh
bash bin/sync-to-runtime.sh
```

### If Phase 2 Causes Issues

```bash
# Remove swarm-summary-print function
# It's new, so removal won't break anything
```

### If Phase 3 Causes Issues

```bash
# Revert build.md to use swarm-display-render
git checkout HEAD -- .claude/commands/ant/build.md
```

---

## Files to Modify

| File | Changes | Phase |
|------|---------|-------|
| `.aether/aether-utils.sh` | Update spawn-log, spawn-complete returns; add swarm-summary-print | 1, 2 |
| `.claude/commands/ant/build.md` | Replace swarm-display-render with swarm-summary-print | 3 |
| `.aether/utils/swarm-display.sh` | Keep for future multi-colony use | 4 (optional) |
| `runtime/aether-utils.sh` | Auto-synced from .aether/ | Auto |

---

## Acceptance Criteria

### Must Have

- [ ] `spawn-log` returns emoji-formatted result
- [ ] `spawn-complete` returns emoji-formatted result
- [ ] `swarm-summary-print` function exists and works
- [ ] build.md uses in-conversation display
- [ ] No exit code 144 errors
- [ ] activity.log still populated correctly

### Should Have

- [ ] Output is concise (under 50 lines typical)
- [ ] Tree indentation shows parent-child relationships
- [ ] Status icons (✅❌⏳) are clear

### Nice to Have

- [ ] ANSI colors work in Claude Code (test first)
- [ ] Configurable verbosity level
- [ ] Progress percentage in output

---

## Related Documentation

- `.aether/workers.md` - Caste definitions and emoji mapping
- `.aether/docs/pheromones.md` - Signal system
- `.aether/utils/swarm-display.sh` - Current terminal display
- `docs/session-freshness-handoff-v2.md` - Session context persistence

---

## Questions for Review

1. **JSON format:** Should output be a single string with `\n` escapes, or a JSON array of lines?
2. **Tree depth:** How many levels of indentation to support?
3. **Sorting:** Chronological order or hierarchical tree order?
4. **Verbosity:** Full task descriptions or truncated summaries?
5. **ANSI codes:** Test if Claude Code renders colors, or stick to emojis only?

---

*Document created: 2026-02-16*
*Ready for review by next agent session*

---

## Implementation Learnings (2026-02-16)

### Pre-requisite Fix Completed ✅

**Bug Fixed:** `.aether/utils/spawn-tree.sh` expected 5 pipes (6 fields) but spawn-log writes 6 pipes (7 fields with model).

**Changes Made:**
- Line 47: `pipe_count -eq 5` → `pipe_count -eq 6`
- Line 287: Same in `get_active_spawns()`
- Line 350: Same in `get_spawn_children()`

**This fix is committed and working.** The parser now correctly reads 7-field spawn events.

### swarm-summary-print Implementation (Reverted)

A working implementation was added to `.aether/aether-utils.sh` but reverted due to XML migration uncertainty.

**Key learnings from implementation:**
1. **Bash 3.2 compatibility required** - macOS default bash doesn't support associative arrays (`declare -A`). Must use temp files.
2. **Two-format parsing needed** - spawn-tree.txt has both 7-field spawn events and 4-field completion events
3. **Tree building is complex** - need to build parent->children map from flat data
4. **Plain text output works** - Claude Code displays plain text nicely without JSON wrapper

### Decisions Made

| Question | Decision | Rationale |
|----------|----------|-----------|
| JSON format? | **Plain text** (no JSON wrapper) | Displays better in conversation |
| Display scope? | **All ants with status** | Show complete picture |
| spawn-complete change? | **NO - keep append-only** | Race conditions, audit trail |
| Location? | **In aether-utils.sh** | Follows existing pattern |

### After XML Migration

1. **Update spawn-tree.txt format** - if changing to XML, update:
   - spawn-log to write XML
   - spawn-complete to write XML
   - spawn-tree.sh to parse XML

2. **Implement swarm-summary-print** - read XML and output formatted tree

3. **Update build.md** - replace swarm-display-render with swarm-summary-print

### Code That Was Reverted

The swarm-summary-print implementation (~130 lines) was in `.aether/aether-utils.sh` after line 2406. It used:
- Temp files for bash 3.2 compatibility
- Two-pass parsing (spawn events, then completion events)
- Recursive tree printing with └── and ├── connectors
- Emoji and status icons (✅❌⏳🚫)

To restore, see git history or ask the user for the backup.
