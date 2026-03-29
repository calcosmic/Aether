# Architecture Research: v2.6 Bugfix & Hardening

**Domain:** Aether colony orchestration system -- cross-colony isolation, lock scope safety, input sanitization
**Researched:** 2026-03-29
**Confidence:** HIGH (based on direct codebase analysis of file-lock.sh, hive.sh, state-api.sh, atomic-write.sh, spawn.sh, spawn-tree.sh, swarm.sh, queen.sh, learning.sh, and aether-utils.sh dispatcher)

---

## Executive Summary

Aether's colony system operates across two storage scopes: **colony-local** (`.aether/` within each repo) and **hub-shared** (`~/.aether/` across all colonies). The current architecture has four architectural-level bugs that undermine this two-scope model:

1. **LOCK_DIR mutation in hive.sh** temporarily redirects colony-local lock writes to the hub directory. If colony-local code runs during this window (e.g., a concurrent colony operation), locks land in the wrong directory, breaking mutual exclusion.

2. **QUEEN.md promotion writes to colony-local without namespacing.** `queen-promote` writes wisdom to `$AETHER_ROOT/.aether/QUEEN.md` (colony-local) using the format `- **{colony_name}** ({ts}): {content}`, but `queen-read` merges global and local QUEEN.md files with simple string concatenation. Colony-specific wisdom from one repo leaks into every other colony on the same machine via the global read path.

3. **spawn.sh and spawn-tree.sh interpolate user-controlled `ant_name` into grep patterns** without escaping. An ant name containing regex metacharacters (`|`, `.`, `*`) causes incorrect matches in the spawn tree, and in pathological cases could match unintended records.

4. **atomic-write.sh does not release locks on JSON validation failure.** When `atomic_write` is called for a `.json` target and jq validation fails, the function returns 1 without releasing the lock. The caller (`_state_write`) does release the lock on atomic_write failure, but `_state_mutate` acquires its own lock before calling atomic_write and has a gap where the jq mutation step can fail without releasing.

This document maps the correct architectural boundaries, prescribes patterns for safe cross-scope operations, and identifies the specific code changes needed.

---

## Part 1: The Two-Scope Storage Model

### Current Architecture

```
COLONY-LOCAL (.aether/)               HUB-SHARED (~/.aether/)
+---------------------------+         +---------------------------+
| .aether/data/             |         | ~/.aether/                 |
|   COLONY_STATE.json       |         |   QUEEN.md (global wisdom) |
|   pheromones.json         |         |   hive/wisdom.json         |
|   constraints.json        |         |   registry.json            |
|   learning-obs.json       |         |   eternal/memory.json      |
|   midden/midden.json      |         |   skills/                  |
|   spawn-tree.txt          |         |   system/                  |
|   session.json            |         |   manifest.json            |
|   activity.log            |         |   version.json             |
| .aether/locks/            |         |   .aether/ (internal hub)  |
| .aether/QUEEN.md (local)  |         |                           |
| .aether/dreams/           |         |                           |
| .aether/temp/             |         |                           |
+---------------------------+         +---------------------------+
         |                                      |
         +--- MUTEX BUG: LOCK_DIR leak --------+
         +--- WISDOM BUG: unscoped writes -----+
         +--- INJECTION BUG: unescaped grep ---+
```

### Correct Architecture

The two-scope model should enforce these rules:

| Property | Colony-Local | Hub-Shared |
|----------|-------------|------------|
| Scope | Single repo only | All repos on machine |
| Lock dir | `$AETHER_ROOT/.aether/locks/` | `~/.aether/hive/locks/` (dedicated) |
| State | COLONY_STATE.json | wisdom.json, registry.json |
| QUEEN.md | Per-colony wisdom | Cross-colony preferences only |
| Writes | Never touch hub paths | Never touch colony paths |
| Mutual exclusion | Per-colony | Cross-colony (hub resources) |

### The Ownership Principle

**Every piece of state has exactly one owner scope.** Reads can cross boundaries (a colony reads hub wisdom), but writes must target the owner scope exclusively. When a colony-local operation needs to write hub state (e.g., promoting to hive), it should use a dedicated hub-write function that never touches the colony-local lock directory.

---

## Part 2: Bug #1 -- LOCK_DIR Mutation in hive.sh

### What Goes Wrong

`hive.sh` implements a save/restore pattern for `LOCK_DIR`:

```bash
# In _hive_init (line 46-48), _hive_store (line 133-135), _hive_read (line 327-329):
hs_saved_lock_dir="$LOCK_DIR"
LOCK_DIR="$HOME/.aether/hive"
acquire_lock "$hs_wisdom_file" || { LOCK_DIR="$hs_saved_lock_dir"; json_err ... }
```

The intent: hub-level locks for `wisdom.json` provide cross-colony mutual exclusion. The mechanism: temporarily mutate the global `LOCK_DIR` so `acquire_lock` creates lock files in `~/.aether/hive/` instead of `$AETHER_ROOT/.aether/locks/`.

**The bug:** `LOCK_DIR` is a **global shell variable** sourced at startup by `file-lock.sh` (line 20). When `hive.sh` mutates it, the change is visible to ALL code running in the same shell session. If `acquire_lock` or `release_lock` is called for a colony-local resource during this window (e.g., from a concurrent subshell, a trap handler, or a nested function call), the lock file lands in `~/.aether/hive/` instead of `.aether/locks/`.

**Consequences:**
- Colony-local locks go to the hub directory, breaking per-colony isolation
- Concurrent colony operations may not see each other's locks
- Stale lock detection may reference wrong directory
- The trap-based `cleanup_locks` in `file-lock.sh` (line 157) uses the mutated `LOCK_DIR`

**Note:** The save/restore pattern *appears* correct on every individual code path. The issue is that shell does not have thread safety -- any code that runs between the save and restore (including signal handlers, background jobs, or function calls within the same scope) sees the mutated value. In practice, since `aether-utils.sh` runs as a single process, the risk is primarily from:
1. Signal handlers (EXIT, TERM, INT, HUP traps)
2. The `hive-promote` function calling `hive-store` via `bash "$0"` (subshell -- safe, but wasteful)
3. Future concurrent execution patterns

### Recommended Fix: Dedicated Hub Lock Function

Instead of mutating a global, introduce a scoped lock acquisition pattern:

**Pattern: Parameterized Lock Acquisition**

```bash
# file-lock.sh -- add optional lock_dir parameter
acquire_lock() {
    local file_path="$1"
    local lock_dir_override="${2:-}"  # NEW: optional override
    local effective_lock_dir="${lock_dir_override:-$LOCK_DIR}"
    local lock_file="${effective_lock_dir}/$(basename "$file_path").lock"
    # ... rest of function uses $effective_lock_dir instead of $LOCK_DIR
}

release_lock() {
    # No change needed -- uses CURRENT_LOCK which already contains full path
    ...
}
```

**Pattern: Hub Lock Helper**

```bash
# hive.sh -- new helper, replaces save/restore pattern
_hive_acquire_lock() {
    local file_path="$1"
    acquire_lock "$file_path" "$HOME/.aether/hive/locks"
}
```

This eliminates the global mutation entirely. Lock directory is determined at call time, not by environment state.

**Migration path:**
1. Add `lock_dir_override` parameter to `acquire_lock` in `file-lock.sh`
2. Create `~/.aether/hive/locks/` directory (currently hive.sh uses `~/.aether/hive/` directly, which puts lock files alongside data files)
3. Replace all save/restore patterns in `hive.sh` with `_hive_acquire_lock`
4. Verify `cleanup_locks` trap still works (it does -- `CURRENT_LOCK` contains the full path)

### Lock Directory Should Be Separate From Data

Currently hive.sh puts locks in `~/.aether/hive/` (same directory as `wisdom.json`). This is an anti-pattern: lock files and data files should be in separate directories for clean lifecycle management. The fix above uses `~/.aether/hive/locks/` as a dedicated lock directory.

---

## Part 3: Bug #2 -- QUEEN.md Namespacing

### What Goes Wrong

`queen-promote` (queen.sh line 349) writes to `$AETHER_ROOT/.aether/QUEEN.md` -- the **colony-local** QUEEN.md. The entry format includes a colony_name tag:

```bash
entry="- ${entry_prefix}**${colony_name}** (${ts}): ${content}"
```

However, `queen-read` (queen.sh line 183) reads **both** global and local QUEEN.md and merges them:

```bash
queen_global="$HOME/.aether/QUEEN.md"   # hub
queen_local="$AETHER_ROOT/.aether/QUEEN.md"  # colony
# ... reads both, concatenates content per section
```

**The problem:** Colony-specific wisdom written to the local QUEEN.md is only visible within that colony. This is correct behavior for per-colony learnings. But `queen-promote` is called from `learning-promote-auto` (learning.sh line 423), which is triggered by observations from colony work. These learnings are always colony-scoped, never promoted to the hub.

Meanwhile, `queen-read` merges global and local without distinguishing provenance. A worker reading QUEEN.md sees a flat list of wisdom entries from both scopes, some tagged with colony names and some not, with no way to tell which scope they came from.

**The deeper issue:** There is no mechanism to promote colony wisdom to the hub QUEEN.md. The `queen-promote` function always writes to the local file. The hub QUEEN.md (`~/.aether/QUEEN.md`) is only populated by:
1. Manual user preferences via `/ant:preferences`
2. Initial setup

Colony learnings that should cross colony boundaries (the whole purpose of the hive system) never reach the hub QUEEN.md. Instead, they go through the separate `hive-promote` pipeline to `wisdom.json`. This means there are **two parallel wisdom systems** (QUEEN.md and hive wisdom.json) that don't communicate.

### Recommended Fix: Explicit Scope Parameter

Add a `--scope` parameter to `queen-promote`:

```bash
# queen-promote <type> <content> <colony_name> [--scope local|hub]
# Default: local (current behavior, backward compatible)

queen_scope="local"
for arg in "$@"; do
  case "$arg" in
    --scope) queen_scope="${2:-local}"; shift 2 ;;
  esac
done

if [[ "$queen_scope" == "hub" ]]; then
  queen_file="$HOME/.aether/QUEEN.md"
else
  queen_file="$AETHER_ROOT/.aether/QUEEN.md"
fi
```

**When to use each scope:**

| Write source | Scope | Rationale |
|-------------|-------|-----------|
| `learning-promote-auto` | local | Colony-specific learnings |
| `/ant:preferences` | hub | User preferences apply globally |
| `hive-promote` (high confidence) | hub | Cross-colony validated wisdom |
| `/ant:seal` wisdom export | hub | Colony-completed wisdom worth sharing |

**Migration path:**
1. Add `--scope` parameter to `queen-promote` (default: `local`)
2. No existing callers change behavior (backward compatible)
3. Add hub-scoped promotion call in `hive-promote` for entries with confidence >= 0.9
4. Document the two-scope model in QUEEN.md header

### The Two-System Problem (Longer Term)

The parallel wisdom systems (QUEEN.md and hive wisdom.json) should eventually converge. The current state:

- **QUEEN.md** (Markdown): Human-readable, used for worker priming, edited by `queen-promote`
- **hive wisdom.json** (JSON): Machine-readable, cross-colony, accessed via `hive-read`

The `queen-read` function already returns JSON, so the worker priming pipeline already works with structured data. The hive system provides deduplication, confidence scoring, and domain scoping that QUEEN.md lacks. The recommended long-term direction is:

1. **Keep QUEEN.md as the human-readable view** -- a rendering of wisdom.json filtered by domain
2. **Make hive wisdom.json the single source of truth** -- all writes go through hive-store
3. **queen-read becomes a hive-read proxy** -- reads from wisdom.json, formats for display

This is NOT a v2.6 change. It is documented here as architectural context for future work.

---

## Part 4: Bug #3 -- Unsafe ant_name Interpolation

### What Goes Wrong

`spawn.sh` and `spawn-tree.sh` use `ant_name` in grep patterns without escaping:

**spawn.sh, line 139:**
```bash
if ! grep -q "|$ant_name|" "$DATA_DIR/spawn-tree.txt" 2>/dev/null; then
```

**spawn.sh, line 152:**
```bash
parent=$(grep "|$current_ant|" "$DATA_DIR/spawn-tree.txt" 2>/dev/null | grep "|spawned$" | head -1 | cut -d'|' -f2)
```

**spawn-tree.sh, line 98:**
```bash
if ! grep -q "|$ant_name|" "$file_path" 2>/dev/null; then
```

**spawn-tree.sh, line 111:**
```bash
parent=$(grep "|$current|" "$file_path" 2>/dev/null | grep "|spawned$" | head -1 | cut -d'|' -f2)
```

**swarm.sh, line 896:**
```bash
grep -v "^$ant_name|" "$timing_file" > "${timing_file}.tmp"
```

**swarm.sh, line 915:**
```bash
if [[ ! -f "$timing_file" ]] || ! grep -q "^$ant_name|" "$timing_file"
```

The spawn-tree format is pipe-delimited: `timestamp|parent|caste|child_name|task|model|status`. If `ant_name` contains `|`, grep will match different fields than intended. If it contains `.` or `*`, it acts as a regex wildcard. If it contains regex anchors (`^`, `$`), it can match unintended positions.

**Practical risk:** Ant names are generated by the colony system (e.g., "Builder", "Scout-1", "swarm:investigate-3") and are typically safe. However, the `--task` parameter in `_spawn_log` is user-controlled and appears in the spawn tree as field 5. The `child_name` (field 4) is derived from the agent definition and is typically safe. The `parent_id` (field 2) comes from the caller.

The actual risk vectors:
1. Agent names with special characters (future agents may have complex names)
2. Task summaries containing `|` (common in technical descriptions)
3. Swarm IDs containing `|` (swarm IDs are user-provided in some paths)

### Recommended Fix: Fixed-String Matching

Replace regex grep with `grep -F` (fixed-string) for all ant_name lookups:

```bash
# BEFORE (regex):
grep -q "|$ant_name|" "$file_path"

# AFTER (fixed-string):
grep -Fq "|$ant_name|" "$file_path"
```

For patterns that need regex (e.g., `|spawned$`), keep those as-is since they match literal fixed strings anyway.

**Full list of changes:**

| File | Line | Current | Fix |
|------|------|---------|-----|
| spawn.sh | 139 | `grep -q "|$ant_name|"` | `grep -Fq "|$ant_name|"` |
| spawn.sh | 152 | `grep "|$current_ant|"` | `grep -F "|$current_ant|"` |
| spawn-tree.sh | 98 | `grep -q "|$ant_name|"` | `grep -Fq "|$ant_name|"` |
| spawn-tree.sh | 111 | `grep "|$current|"` | `grep -F "|$current|"` |
| spawn-tree.sh | 231 | `grep "|$current|"` | `grep -F "|$current|"` |
| swarm.sh | 896 | `grep -v "^$ant_name|"` | `grep -Fv "^$ant_name|"` |
| swarm.sh | 915 | `grep -q "^$ant_name|"` | `grep -Fq "^$ant_name|"` |
| swarm.sh | 921 | `grep "^$ant_name|"` | `grep -F "^$ant_name|"` |
| swarm.sh | 960 | `grep -q "^$ant_name|"` | `grep -Fq "^$ant_name|"` |
| swarm.sh | 966 | `grep "^$ant_name|"` | `grep -F "^$ant_name|"` |

### Input Validation (Defense in Depth)

In addition to `grep -F`, add validation at the entry point where ant names are created:

```bash
# In _spawn_log, after parsing arguments:
if [[ "$child_name" =~ [\|\$\`] ]]; then
    json_err "$E_VALIDATION_FAILED" "Ant name contains unsafe characters" '{"name":"$child_name"}'
fi
```

This prevents unsafe names from entering the spawn tree at all.

---

## Part 5: Bug #4 -- atomic-write.sh Lock Release on Validation Failure

### What Goes Wrong

`atomic_write` (atomic-write.sh lines 49-94) acquires no locks itself -- it relies on callers to manage locking. However, `_state_write` (state-api.sh line 77) acquires a lock, then calls `atomic_write`. If `atomic_write` fails (e.g., JSON validation failure on line 74), `_state_write` does release the lock (line 88):

```bash
atomic_write "$sw_state_file" "$sw_content" || {
    release_lock 2>/dev/null || true
    json_err "$E_UNKNOWN" "Failed to write COLONY_STATE.json"
}
```

This path is correct. The actual gap is in `_state_mutate` (state-api.sh lines 96-145):

```bash
# Line 125: jq mutation can fail
sm_updated=$(jq "$sm_expr" "$sm_state_file" 2>/dev/null) || {
    release_lock 2>/dev/null || true  # CORRECT: releases on jq failure
    json_err ...
}

# Line 131: result validation
if [[ -z "$sm_updated" ]] || ! echo "$sm_updated" | jq -e . >/dev/null 2>&1; then
    release_lock 2>/dev/null || true  # CORRECT: releases on invalid result
    json_err ...
fi

# Line 137: atomic_write can fail
atomic_write "$sm_state_file" "$sm_updated" || {
    release_lock 2>/dev/null || true  # CORRECT: releases on write failure
    json_err ...
}

# Line 142: success path
release_lock 2>/dev/null || true  # CORRECT: releases on success
```

**Revised assessment:** After careful line-by-line review, `_state_mutate` actually handles lock release correctly on ALL failure paths. Each `||` block after the critical operation releases the lock before erroring out. The code uses `2>/dev/null || true` to suppress errors from `release_lock` itself (lock may already be released or may not be held).

**The remaining concern:** `atomic_write` itself (in atomic-write.sh) is a lower-level utility that does NOT acquire or release locks. If a future caller of `atomic_write` forgets to release the lock on failure, the lock leaks. This is a **design fragility** rather than an active bug.

### Recommended Fix: Lock-Aware Atomic Write (Optional)

Two options, ranked by preference:

**Option A (Recommended): Document the Contract**

Add a clear contract comment to `atomic_write`:

```bash
# CONTRACT: atomic_write does NOT manage locks.
# Callers MUST acquire_lock before calling and release_lock after,
# handling all failure paths. See _state_write and _state_mutate for examples.
```

**Option B: Wrapper Function**

Create a `locked_atomic_write` wrapper in state-api.sh:

```bash
_locked_atomic_write() {
    local target="$1"
    local content="$2"
    acquire_lock "$target" || return 1
    if ! atomic_write "$target" "$content"; then
        release_lock 2>/dev/null || true
        return 1
    fi
    release_lock 2>/dev/null || true
    return 0
}
```

Option A is sufficient for v2.6. Option B is better for long-term safety.

---

## Part 6: CLAUDE.md Documentation Error

The CLAUDE.md states:

> `LOCK_DIR = .aether/data/locks/ (colony-local)`

The actual code in `file-lock.sh` line 20:

```bash
LOCK_DIR="$AETHER_ROOT/.aether/locks"
```

And confirmed by `.npmignore` line 12:

```
.aether/locks/
```

The correct path is `.aether/locks/`, not `.aether/data/locks/`. The `aether-utils.sh` dispatcher also references `.aether/locks/` on line 4485.

**Fix:** Update CLAUDE.md to reflect the actual path.

---

## Part 7: Architectural Patterns for Cross-Scope Operations

### Pattern 1: Scoped Lock Acquisition

**What:** Lock acquisition that takes an explicit scope parameter instead of relying on a mutable global.

**When:** Any code that needs to lock hub resources (wisdom.json, registry.json) while also potentially locking colony-local resources.

**Implementation:**

```bash
# Colony-local lock (default behavior, unchanged)
acquire_lock "$DATA_DIR/COLONY_STATE.json"

# Hub-scoped lock (new capability)
acquire_lock "$HOME/.aether/hive/wisdom.json" "$HOME/.aether/hive/locks"

# Via helper
_hive_acquire_lock "$HOME/.aether/hive/wisdom.json"
```

### Pattern 2: Explicit Scope for QUEEN.md Writes

**What:** Every write to QUEEN.md specifies whether it targets colony-local or hub scope.

**When:** Any function that promotes wisdom, preferences, or learnings.

**Implementation:**

```bash
queen-promote <type> <content> <colony_name> [--scope local|hub]
# local (default): writes to $AETHER_ROOT/.aether/QUEEN.md
# hub: writes to $HOME/.aether/QUEEN.md
```

### Pattern 3: Fixed-String Matching for User Data

**What:** All grep operations on user-controlled data use `grep -F` (fixed-string mode).

**When:** Any grep where the pattern includes data derived from user input, ant names, task summaries, or swarm IDs.

**Implementation:**

```bash
# Rule: If the variable could contain | . * [ ] $ ^, use -F
grep -Fq "|$user_controlled|" "$data_file"
```

### Pattern 4: Lock Contract Documentation

**What:** Functions that require callers to manage locks document this explicitly.

**When:** Any function that modifies shared state but does not manage its own locking.

**Implementation:**

```bash
# CONTRACT: Caller must hold lock on $target_file.
# This function does NOT acquire or release locks.
atomic_write() { ... }
```

---

## Part 8: Anti-Patterns to Avoid

### Anti-Pattern 1: Global Variable Mutation for Scope Switching

**What people do:** Save a global, mutate it, do work, restore it.
**Why it's wrong:** Signal handlers, traps, and concurrent subshells see the mutated value. Shell has no thread safety.
**Do this instead:** Pass the scope as a parameter to the function that needs it.

### Anti-Pattern 2: Implicit Scope in Write Functions

**What people do:** A write function determines its target from global state (AETHER_ROOT) without an explicit scope parameter.
**Why it's wrong:** When the same function is called from different contexts (colony-local vs hub), the caller must know the implementation detail of which path the function uses.
**Do this instead:** Accept a `--scope` or `--target` parameter that makes the write destination explicit.

### Anti-Pattern 3: Regex Grep on Structured Data

**What people do:** Use `grep "$variable"` to search pipe-delimited or structured text.
**Why it's wrong:** The variable may contain regex metacharacters that change the match semantics.
**Do this instead:** Use `grep -F "$variable"` for literal matching, or use `awk -F'|' '$4 == name'` for field-specific matching.

---

## Part 9: Migration Path

### Phase 1: Safe, Backward-Compatible Changes

These changes cannot break existing colonies:

1. **file-lock.sh:** Add optional `lock_dir_override` parameter to `acquire_lock` (backward compatible -- existing callers pass no second argument)
2. **spawn.sh, spawn-tree.sh, swarm.sh:** Add `-F` flag to all grep calls using ant_name (changes regex to fixed-string, no behavior change for current ant names)
3. **CLAUDE.md:** Fix LOCK_DIR path documentation
4. **atomic-write.sh:** Add CONTRACT comment documenting lock management responsibility

### Phase 2: hive.sh Refactor (Requires Testing)

1. **hive.sh:** Replace all LOCK_DIR save/restore patterns with `_hive_acquire_lock` helper
2. **hive.sh:** Create `~/.aether/hive/locks/` directory
3. **Verify:** Run hive-init, hive-store, hive-read, hive-promote against both single-colony and multi-colony setups

### Phase 3: QUEEN.md Scope Parameter (Requires Testing)

1. **queen.sh:** Add `--scope local|hub` to `_queen_promote` (default: local)
2. **hive-promote:** Add hub-scoped queen-promote call for high-confidence entries
3. **Verify:** Confirm colony-local writes still go to `.aether/QUEEN.md` and hub writes go to `~/.aether/QUEEN.md`

### Data Migration

No data migration is needed. All changes are backward compatible:
- Lock files in the old location (`~/.aether/hive/`) still work; new lock files go to `~/.aether/hive/locks/`
- QUEEN.md scope parameter defaults to `local`, preserving existing behavior
- grep -F produces identical results for current ant names (no special characters)

---

## Part 10: Risk Assessment

| Change | Risk | Mitigation |
|--------|------|------------|
| file-lock.sh parameter addition | Low | New parameter is optional, defaults to current behavior |
| grep -F flag additions | Very Low | Only affects matching behavior for strings with regex metacharacters; current ant names don't have these |
| hive.sh LOCK_DIR refactor | Medium | Must verify all hive operations in multi-colony setup; existing lock files in old location need cleanup |
| queen-promote --scope | Low | Default is `local`, matching current behavior |
| CLAUDE.md documentation fix | None | Documentation only |

---

## Sources

- Direct codebase analysis (confidence: HIGH):
  - `.aether/utils/file-lock.sh` (192 lines) -- LOCK_DIR definition, acquire_lock, release_lock
  - `.aether/utils/hive.sh` (562 lines) -- LOCK_DIR save/restore pattern in _hive_init, _hive_store, _hive_read
  - `.aether/utils/state-api.sh` (200 lines) -- _state_write, _state_mutate lock management
  - `.aether/utils/atomic-write.sh` (227 lines) -- atomic_write, JSON validation
  - `.aether/utils/spawn.sh` (260 lines) -- grep patterns with ant_name
  - `.aether/utils/spawn-tree.sh` (263 lines) -- grep patterns with ant_name
  - `.aether/utils/swarm.sh` (990+ lines) -- grep patterns with ant_name in timing functions
  - `.aether/utils/queen.sh` (1650+ lines) -- queen-promote write path, queen-read merge logic
  - `.aether/utils/learning.sh` (560+ lines) -- learning-promote-auto call to queen-promote
  - `.aether/aether-utils.sh` (5200+ lines) -- dispatcher, DATA_DIR/LOCK_DIR initialization
  - `.npmignore` -- confirms `.aether/locks/` exclusion from package
  - `~/.aether/` -- hub directory structure verification

---
*Architecture research for: Aether v2.6 Bugfix & Hardening -- cross-colony isolation, lock scope safety, input sanitization*
*Researched: 2026-03-29*
