# Distribution Chain Audit — 2026-02-18

> Full audit of the `.aether/` → `runtime/` → `~/.aether/` → target repo update pipeline.

## Critical Bugs

### 1. update-transaction.js syncs from wrong directory (CRITICAL)

**File:** `bin/lib/update-transaction.js:909`
**Bug:** `syncFiles()` reads from `this.HUB_DIR` (`~/.aether/`) instead of `this.HUB_SYSTEM_DIR` (`~/.aether/system/`)

This means `aether update` in target repos copies files from the hub root instead of the `system/` subdirectory. The hub root contains `commands/`, `agents/`, and other non-system directories that should NOT be synced as system files.

**Fix:** Change `this.HUB_DIR` to `this.HUB_SYSTEM_DIR` on line 909.

### 2. EXCLUDE_DIRS incomplete (CRITICAL)

**File:** `bin/lib/update-transaction.js` (syncFiles exclude list)
**Bug:** Only `data` and `locks` are excluded. Missing exclusions:
- `chambers` — archived colonies
- `oracle` — research state
- `archive` — old archives
- `commands` — slash commands (separate distribution path)
- `agents` — agent definitions (separate distribution path)
- `examples` — example files

Without these exclusions, `aether update` could overwrite or create unexpected directories in target repos.

## Dead Weight / Duplicates

### 3. `.aether/agents/` is dead (duplicate of `.opencode/agents/`)

`.opencode/agents/` is in the npm package `files` list and syncs to the hub.
`.aether/agents/` is NOT in any distribution chain — it's a copy that does nothing.

**Fix:** Delete `.aether/agents/` entirely.

### 4. `.aether/commands/` is dead (duplicate of `.claude/commands/ant/`)

Same situation. `.claude/commands/ant/` is in the npm `files` list.
`.aether/commands/` is NOT in any distribution chain.

**Fix:** Delete `.aether/commands/` entirely.

### 5. Triple-copied slash commands

Commands exist in three places:
1. `.claude/commands/ant/` — SOURCE (in npm package)
2. `.opencode/commands/ant/` — OpenCode mirror
3. `.aether/commands/` — dead copy

**Fix:** Delete `.aether/commands/`, keep `.claude/` and `.opencode/`.

## Allowlist Issues

### 6. `caste-system.md` missing from sync allowlist

**File:** `bin/sync-to-runtime.sh`
**Bug:** `.aether/docs/caste-system.md` is referenced by multiple commands but not in the SYSTEM_FILES allowlist in `sync-to-runtime.sh`. It won't get distributed to target repos.

**Fix:** Add `docs/caste-system.md` to SYSTEM_FILES array.

### 7. Phantom `planning.md` in allowlist

**File:** `bin/sync-to-runtime.sh`
**Bug:** `docs/planning.md` is in the allowlist but the file doesn't exist. Silent failure on sync.

**Fix:** Remove `docs/planning.md` from SYSTEM_FILES array.

## Documentation Duplicates

### 8. Duplicate docs in `.aether/docs/` subdirectories

Multiple docs exist both at `.aether/docs/foo.md` and `.aether/docs/subdir/foo.md`. Roughly 9+ duplicates identified. This causes confusion about which version is authoritative.

**Fix:** Audit `.aether/docs/` subdirectories, remove duplicates, keep flat structure.

## Distribution Chain Summary

```
CORRECT CHAIN:
  .aether/system files → sync-to-runtime.sh → runtime/ → npm package → ~/.aether/system/
  .claude/commands/ant/ → npm package → ~/.aether/commands/claude/
  .opencode/agents/ → npm package → ~/.aether/agents/

BROKEN:
  update-transaction.js reads from ~/.aether/ (root) instead of ~/.aether/system/
  Missing EXCLUDE_DIRS means non-system dirs could leak into target repos

DEAD WEIGHT:
  .aether/agents/ — not in any chain
  .aether/commands/ — not in any chain
```

## Recommended Fix Order

1. Fix `update-transaction.js:909` — most critical, affects all target repo updates
2. Expand EXCLUDE_DIRS — prevents accidental directory leaks
3. Add `caste-system.md` to allowlist — ensures caste emojis reach target repos
4. Remove phantom `planning.md` — cleanup
5. Delete `.aether/agents/` and `.aether/commands/` — remove confusion
6. Audit `.aether/docs/` duplicates — cleanup

## Impact

These issues mean:
- Target repos MAY be getting extra files they shouldn't have
- `caste-system.md` is NOT reaching target repos (workers can't look up caste emojis)
- Developers get confused about which copy of agents/commands is real
- The allowlist has a phantom entry that silently fails
