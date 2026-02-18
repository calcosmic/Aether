# Hub System Directory Migration Plan

**Author:** Atlas (Architect)
**Date:** 2026-02-17
**Status:** Draft - Pending Approval

## Executive Summary

The Aether distribution hub at `~/.aether/` currently mixes system files (distributed) with user data (local). This creates ambiguity about which files get overwritten during updates.

**Goal:** Introduce a `~/.aether/system/` subdirectory that contains ALL distributed files, making the boundary between "system files (overwritten)" and "user data (never touched)" crystal clear.

## Problem Analysis

### Current State (Broken)

```
~/.aether/
├── aether-utils.sh      <- System file (should be in system/)
├── workers.md           <- System file (should be in system/)
├── docs/                <- System files (should be in system/)
├── utils/               <- System files (should be in system/)
├── commands/            <- System files (should be in system/)
├── agents/              <- System files (should be in system/)
├── registry.json        <- User data (local) - OK
├── chambers/            <- User data (local) - OK
├── data/                <- User data (local) - OK
└── version.json         <- Metadata - OK at root
```

**The mismatch:**
- `aether-utils.sh bootstrap-system` expects `~/.aether/system/` (line 1919)
- `bin/cli.js setupHub()` writes to `~/.aether/` directly (line 956-1063)
- `/ant:update` reads from `~/.aether/` directly (wrong paths)
- `/ant:init` bootstrap reads from `~/.aether/system/` (correct path in OpenCode, wrong in Claude)

### Target State (Fixed)

```
~/.aether/
├── system/                    <- ALL distributed files
│   ├── aether-utils.sh
│   ├── workers.md
│   ├── docs/
│   ├── utils/
│   ├── commands/
│   │   ├── claude/
│   │   └── opencode/
│   └── agents/
│
├── registry.json              <- User data (never touched)
├── chambers/                  <- User data (never touched)
├── data/                      <- User data (never touched)
└── version.json               <- Metadata (at root for easy access)
```

**Benefits:**
1. Clear separation: `system/` = overwritten, root files = preserved
2. `rm -rf ~/.aether/system/` safely clears distributable files
3. Self-documenting structure
4. Already partially implemented in aether-utils.sh

---

## Implementation Plan

### Phase 1: Update cli.js Install Flow (Priority: P0)

**File:** `bin/cli.js`

**Changes:**

1. **Update HUB_DIR constants** (lines 72-78):
```javascript
// Current:
const HUB_DIR = path.join(HOME, '.aether');
const HUB_COMMANDS_CLAUDE = path.join(HUB_DIR, 'commands', 'claude');
const HUB_COMMANDS_OPENCODE = path.join(HUB_DIR, 'commands', 'opencode');
const HUB_AGENTS = path.join(HUB_DIR, 'agents');

// Change to:
const HUB_DIR = path.join(HOME, '.aether');
const HUB_SYSTEM_DIR = path.join(HUB_DIR, 'system');
const HUB_COMMANDS_CLAUDE = path.join(HUB_SYSTEM_DIR, 'commands', 'claude');
const HUB_COMMANDS_OPENCODE = path.join(HUB_SYSTEM_DIR, 'commands', 'opencode');
const HUB_AGENTS = path.join(HUB_SYSTEM_DIR, 'agents');
```

2. **Update setupHub() function** (lines 953-1063):
   - Create `~/.aether/system/` directory structure
   - Sync runtime/ -> `~/.aether/system/` (instead of `~/.aether/`)
   - Sync commands to `~/.aether/system/commands/claude/` and `opencode/`
   - Sync agents to `~/.aether/system/agents/`
   - Keep `registry.json`, `version.json`, `manifest.json` at `~/.aether/` root
   - Remove the legacy cleanup for `~/.aether/system/` (it will now be the correct location)

3. **Update syncAetherToHub() function** (lines 857-951):
   - Destination should be `HUB_SYSTEM_DIR` instead of `HUB_DIR`
   - Exclude registry.json, version.json, manifest.json from sync

4. **Update update-transaction.js HUB_DIR references**:
   - Add `HUB_SYSTEM_DIR` constant
   - Update `syncAetherToRepo()` to read from `HUB_SYSTEM_DIR`
   - Update `checkHubAccessibility()` to check `HUB_SYSTEM_DIR`

### Phase 2: Update Slash Commands (Priority: P0)

**Files:**
- `.claude/commands/ant/update.md`
- `.claude/commands/ant/init.md`
- `.opencode/commands/ant/update.md`
- `.opencode/commands/ant/init.md`

**Changes for update.md (Step 3):**

```markdown
### Step 3: Sync System Files from Hub

The hub is at `~/.aether/system/` with all system files in it.

Run ONE bash command that syncs everything:

```bash
mkdir -p .aether/docs .aether/utils && \
cp -f ~/.aether/system/aether-utils.sh .aether/ && \
cp -f ~/.aether/system/workers.md .aether/ 2>/dev/null || true && \
cp -f ~/.aether/system/CONTEXT.md .aether/ 2>/dev/null || true && \
cp -f ~/.aether/system/model-profiles.yaml .aether/ 2>/dev/null || true && \
cp -Rf ~/.aether/system/docs/* .aether/docs/ 2>/dev/null || true && \
cp -Rf ~/.aether/system/utils/* .aether/utils/ 2>/dev/null || true && \
chmod +x .aether/aether-utils.sh && \
echo "System files synced"
```
```

**Changes for Step 4 (Commands sync):**
```bash
cp -R ~/.aether/system/commands/claude/* .claude/commands/ant/
cp -R ~/.aether/system/commands/opencode/* .opencode/commands/ant/
cp -R ~/.aether/system/agents/* .opencode/agents/
```

**Changes for init.md (Step 1.5):**
- Both Claude and OpenCode versions should use `~/.aether/system/`

### Phase 3: Migration Strategy (Priority: P0)

**Goal:** Existing hubs must continue working after update.

**Implementation in cli.js setupHub():**

```javascript
function setupHub() {
  // ... existing setup code ...

  // MIGRATION: Check for old structure and migrate
  const oldStructureFiles = [
    path.join(HUB_DIR, 'aether-utils.sh'),
    path.join(HUB_DIR, 'workers.md'),
  ];

  const hasOldStructure = oldStructureFiles.some(f => fs.existsSync(f));
  const hasNewStructure = fs.existsSync(HUB_SYSTEM_DIR);

  if (hasOldStructure && !hasNewStructure) {
    log('  Migrating hub to new structure...');

    // Create system/ directory
    fs.mkdirSync(HUB_SYSTEM_DIR, { recursive: true });

    // Move system files to system/
    const systemFilePatterns = [
      '*.sh', '*.md', '*.yaml',
      'docs', 'utils', 'commands', 'agents',
    ];

    for (const pattern of systemFilePatterns) {
      // Move matching files/dirs to system/
      // ... implementation ...
    }

    log('  Migration complete: system files moved to ~/.aether/system/');
  }

  // ... rest of setup ...
}
```

**Migration safety:**
- Only migrate if old structure exists and new structure doesn't
- Preserve registry.json, version.json at root
- Never touch chambers/, data/ directories

### Phase 4: Verification Steps (Priority: P1)

After implementation, verify:

1. **Fresh install:**
   ```bash
   rm -rf ~/.aether/
   npm install -g .
   ls -la ~/.aether/system/  # Should contain system files
   ls -la ~/.aether/         # Should only have registry.json, version.json, system/
   ```

2. **Migration from old structure:**
   ```bash
   # Create old structure
   mkdir -p ~/.aether/docs
   touch ~/.aether/aether-utils.sh
   touch ~/.aether/workers.md
   touch ~/.aether/registry.json

   npm install -g .
   # Should see: "Migrating hub to new structure..."
   ls -la ~/.aether/system/  # Should contain migrated files
   ls -la ~/.aether/aether-utils.sh  # Should not exist (moved)
   ```

3. **Update flow:**
   ```bash
   cd /path/to/some/aether/repo
   aether update
   # Should read from ~/.aether/system/
   ```

4. **Bootstrap flow:**
   ```bash
   # In a repo without .aether/
   /ant:init "test goal"
   # Should bootstrap from ~/.aether/system/
   ```

### Phase 5: Rollback Plan (Priority: P1)

If migration causes issues:

1. **Revert cli.js changes:**
   ```bash
   git revert <commit-hash>
   npm install -g .
   ```

2. **Manual rollback for users:**
   ```bash
   # If new structure exists but causes issues:
   mv ~/.aether/system/* ~/.aether/
   rmdir ~/.aether/system/
   ```

3. **Preserve user data:**
   - `registry.json`, `chambers/`, `data/` are never touched
   - Only system files move between structures

---

## File Change Summary

| File | Changes Required | Priority |
|------|------------------|----------|
| `bin/cli.js` | HUB constants, setupHub(), syncAetherToHub() | P0 |
| `bin/lib/update-transaction.js` | HUB constants, syncAetherToRepo() | P0 |
| `.claude/commands/ant/update.md` | Step 3-4 paths | P0 |
| `.claude/commands/ant/init.md` | Step 1.5 paths | P0 |
| `.opencode/commands/ant/update.md` | Step 3-4 paths | P0 |
| `.opencode/commands/ant/init.md` | Step 1.5 paths | P0 |
| `bin/sync-to-runtime.sh` | No changes needed | N/A |

**Note:** `sync-to-runtime.sh` only handles the repo-internal sync from `.aether/` to `runtime/`. It does not reference the hub.

---

## Order of Changes

1. **bin/cli.js** - Core install logic (must change first)
2. **bin/lib/update-transaction.js** - Update transaction (depends on cli.js constants)
3. **Slash commands** - Can be done in parallel with cli.js
4. **Testing** - After all changes

---

## Risk Assessment

| Risk | Mitigation |
|------|------------|
| Users have uncommitted work in system files | Migration only moves files, doesn't modify content |
| Old npm versions expect old structure | Migration happens on every install, handles both cases |
| Commands reference wrong paths during transition | Update all 4 command files in same commit |
| User manually edited system files | Files are moved, not overwritten; manual edits preserved |

---

## Success Criteria

- [ ] `npm install -g .` creates `~/.aether/system/` structure
- [ ] Existing hubs migrate automatically on install
- [ ] `/ant:update` syncs from `~/.aether/system/`
- [ ] `/ant:init` bootstraps from `~/.aether/system/`
- [ ] User data (registry, chambers, data) never touched
- [ ] No "file not found" errors during update

---

## Appendix: Current Code References

### cli.js HUB_DIR usage (lines 72-78)
```javascript
const HUB_DIR = path.join(HOME, '.aether');
const HUB_COMMANDS_CLAUDE = path.join(HUB_DIR, 'commands', 'claude');
const HUB_COMMANDS_OPENCODE = path.join(HUB_DIR, 'commands', 'opencode');
const HUB_AGENTS = path.join(HUB_DIR, 'agents');
const HUB_REGISTRY = path.join(HUB_DIR, 'registry.json');
const HUB_VERSION = path.join(HUB_DIR, 'version.json');
```

### aether-utils.sh bootstrap-system (line 1919)
```bash
hub_system="$HOME/.aether/system"
```

### update.md Step 3 (current, incorrect)
```bash
cp -f ~/.aether/aether-utils.sh .aether/
```

### init.md Step 1.5 (OpenCode, already correct)
```bash
bash ~/.aether/system/aether-utils.sh bootstrap-system
```

---

*Generated by Atlas (Architect) - 2026-02-17*
