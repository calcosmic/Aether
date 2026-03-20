# Phase 20: Distribution Simplification - Research

**Researched:** 2026-02-19 (REFRESHED -- all line numbers verified against current main after xml-hardening, pheromone-consumption, and semantic-layer merges)
**Domain:** npm package distribution pipeline, Node.js file system, git hooks
**Confidence:** HIGH -- all findings are from direct codebase inspection

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Cleanup approach:**
- Delete runtime/ entirely -- no redirect, no README stub, clean removal
- Delete sync-to-runtime.sh entirely -- no archive copy
- Add a pre-packaging validation step (check required files exist in .aether/) but no file copying
- Update all documentation and code comments that reference runtime/ as part of this phase -- not deferred

**What gets published:**
- Updates via `aether update` should clean up files that were removed from distribution -- keep target repos tidy
- Unify all three distribution paths (system files, slash commands, agent definitions) into a single pipeline -- do this in Phase 20, not later

**Guard rails:**
- Auto-check before packaging to verify no private data (colony state, dream journal, research files) would be included
- Include a dry-run mode that shows exactly what would be published without actually publishing

**Migration path:**
- Auto-cleanup of old runtime/ artifacts when users run `aether update` on the new version
- Major version bump to signal structural change
- One-time migration message shown after update explaining the change
- Version-aware error messages: detect old structure and suggest running `aether update`

### Claude's Discretion

- Allowlist vs exclude-list approach for what gets published
- Pre-commit hook: remove or repurpose
- Auto-check severity: hard block vs warning on private data detection
- Pre-packaging validation implementation details
- Exact format/wording of migration message and version-aware errors

### Deferred Ideas (OUT OF SCOPE)

None -- discussion stayed within phase scope
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| PIPE-01 | runtime/ staging directory eliminated -- npm package reads directly from .aether/ | Package `files` field updated (package.json line 14), `setupHub` in cli.js (lines 1048-1058) must read from `.aether/` not `runtime/`, `.npmignore` must block private directories |
| PIPE-02 | sync-to-runtime.sh replaced with direct packaging approach | The `preinstall`/`prepublishOnly` npm scripts (package.json lines 21, 23) that call `sync-to-runtime.sh` must be replaced with a validation-only script; `setupHub`'s `runtimeSrc` path (cli.js line 1050) must change to `.aether/` |
| PIPE-03 | Pre-commit hook updated for simplified pipeline | `.git/hooks/pre-commit` (43 lines) currently blocks direct `runtime/` edits and auto-runs sync; must be rewritten or removed |
</phase_requirements>

---

## Summary

The current distribution pipeline has a three-step flow: `.aether/` (source of truth) -> `runtime/` (staged copy) -> `~/.aether/system/` (hub). The `runtime/` staging directory exists because npm's `files` field only packages explicitly listed directories, and the `.aether/` directory was historically excluded from the package to avoid shipping local colony state alongside system files. Phase 20 eliminates this indirection by restructuring the package so npm reads directly from `.aether/`, using an exclude-based approach (`.npmignore`) to keep private data out.

The core mechanical change is straightforward: swap `runtime/` for `.aether/` in three key places -- the `package.json` files field, the `setupHub()` function in `bin/cli.js`, and the npm lifecycle scripts. However, the phase also touches six related concerns that require careful sequencing: (1) the allowlist in `update-transaction.js` that mirrors the sync script's file list, (2) the pre-commit git hook, (3) four `runtime/` path references embedded in `aether-utils.sh`, (4) every documentation file that describes the old architecture, (5) the validation/privacy-check script that replaces `sync-to-runtime.sh`, and (6) the migration cleanup logic for existing users who have hub state populated from the old `runtime/` source.

A notable finding: the SYSTEM_FILES arrays in `cli.js` (lines 380-439) and `update-transaction.js` (lines 179-240) are **already out of sync**. cli.js includes `planning.md` but is missing `docs/caste-system.md`, `docs/error-codes.md`, `docs/queen-commands.md`, `rules/aether-colony.md`, and `templates/QUEEN.md.template` that update-transaction.js has. This is exactly the kind of maintenance problem that Phase 20 eliminates by moving to exclude-based sync.

**Primary recommendation:** Use an exclude-list approach (`.npmignore` with explicit blocks for private directories) rather than an allowlist. This is simpler to maintain as new files are added to `.aether/`, and the private directories are stable and well-understood (`data/`, `dreams/`, `oracle/`, `checkpoints/`, `locks/`, `temp/`, `archive/`, `chambers/`, `examples/`).

---

## Standard Stack

### Core (existing -- no new dependencies)

| Component | Version | Purpose | Notes |
|-----------|---------|---------|-------|
| Node.js `fs` | built-in | File operations in cli.js | Already used extensively |
| npm `files` field | -- | Package inclusion allowlist in package.json | Currently includes `runtime/` at line 14; must replace with `.aether/` |
| `.npmignore` | -- | Exclusion list for npm publish | Already exists (39 lines); must expand |
| bash | -- | Pre-packaging validation script | Replaces sync-to-runtime.sh |
| git hooks | -- | Pre-commit validation | `.git/hooks/pre-commit` must be rewritten |

### No New Libraries Required

This phase involves no new npm dependencies. It is entirely a restructuring of file paths and npm packaging configuration.

---

## Architecture Patterns

### Current Architecture (what we're replacing)

```
Aether Repo
+-- .aether/         <-- SOURCE OF TRUTH (never published directly)
|   +-- aether-utils.sh
|   +-- workers.md
|   +-- docs/
|   +-- utils/
|   +-- data/        <-- LOCAL ONLY (colony state)
|   +-- dreams/      <-- LOCAL ONLY (never distribute)
|
+-- runtime/         <-- STAGING (copy of .aether/ minus private data)
|   +-- aether-utils.sh   <-- copied from .aether/
|   +-- workers.md        <-- copied from .aether/
|   +-- docs/, utils/ ...
|
+-- package.json
|   "files": ["bin/", ".claude/commands/ant/", "runtime/", ...]
|
+-- bin/sync-to-runtime.sh   <-- copies allowed files .aether/ -> runtime/
```

### Target Architecture (what we're building)

```
Aether Repo
+-- .aether/         <-- SOURCE OF TRUTH (published directly minus private dirs)
|   +-- aether-utils.sh
|   +-- workers.md
|   +-- docs/
|   +-- utils/
|   +-- data/        <-- LOCAL ONLY (excluded by .npmignore)
|   +-- dreams/      <-- LOCAL ONLY (excluded by .npmignore)
|
+-- package.json
|   "files": ["bin/", ".claude/commands/ant/", ".aether/", ...]
|   (npm excludes .aether/data/, .aether/dreams/ etc. via .npmignore)
|
+-- bin/validate-package.sh  <-- replaces sync-to-runtime.sh
    (checks required files exist in .aether/, no file copying)
```

### Pattern 1: npm `files` + `.npmignore` for Exclusion Control

**What:** The `package.json` `files` field is a whitelist of what npm includes. `.npmignore` is a secondary exclusion layer applied after the `files` whitelist. When a directory is in `files`, `.npmignore` rules still apply within that directory.

**Why this works:** Adding `.aether/` to `files` would include everything under `.aether/`. Adding `.aether/data/`, `.aether/dreams/`, etc. to `.npmignore` then carves out the private subdirectories. The result is: publish `.aether/` minus the private directories.

**Full list of private directories/files to exclude (verified from current `.aether/` listing):**
```
# .aether/ private directories (never publish)
.aether/data/
.aether/dreams/
.aether/oracle/
.aether/checkpoints/
.aether/locks/
.aether/temp/
.aether/archive/
.aether/chambers/
.aether/examples/
.aether/__pycache__/

# .aether/ private files (never publish)
.aether/ledger.jsonl
.aether/manifest.json
.aether/registry.json
.aether/version.json
.aether/HANDOFF.md
.aether/HANDOFF_AETHER_DEV_2026-02-15.md
.aether/PHASE-0-ANALYSIS.md
.aether/DIAGNOSIS_PROMPT.md
.aether/RESEARCH-SHARED-DATA.md
.aether/diagnose-self-reference.md
.aether/pheromone_system.py
.aether/semantic_layer.py
```

**Source:** Direct inspection of `.aether/` directory, `.npmignore`, and npm documentation patterns. [HIGH confidence]

### Pattern 2: Replacing a Sync Script with a Validation Script

The `bin/sync-to-runtime.sh` script (142 lines) does two things: (1) copies an allowlisted set of files from `.aether/` to `runtime/`, and (2) skips identical files for speed. After Phase 20, the copy step is unnecessary. The replacement script does only validation: verify that the required files exist in `.aether/` before packaging.

**What the replacement script must do:**
1. Check that critical required files exist in `.aether/` (e.g., `aether-utils.sh`, `workers.md`, `docs/`, `utils/`)
2. Check that no private data directories accidentally contain files that look like system files
3. Exit non-zero on failure (blocks `prepublishOnly`)
4. Optionally show a dry-run preview of what would be published

**Example pattern:**
```bash
#!/bin/bash
# bin/validate-package.sh -- pre-packaging validation
set -euo pipefail

AETHER_DIR="$(cd "$(dirname "$0")/../.aether" && pwd)"

REQUIRED_FILES=(
  "aether-utils.sh"
  "workers.md"
  "docs/README.md"
  "utils/atomic-write.sh"
)

for file in "${REQUIRED_FILES[@]}"; do
  if [[ ! -f "$AETHER_DIR/$file" ]]; then
    echo "ERROR: Required file missing from .aether/: $file" >&2
    exit 1
  fi
done

# Private data check
PRIVATE_DIRS=("data" "dreams" "oracle" "checkpoints" "locks" "temp" "archive" "chambers")
for dir in "${PRIVATE_DIRS[@]}"; do
  if [[ -d "$AETHER_DIR/$dir" ]]; then
    echo "NOTE: .aether/$dir/ exists locally -- excluded from package by .npmignore"
  fi
done

echo "Package validation passed."
```

### Pattern 3: setupHub() Path Change -- The Central Code Change

The `setupHub()` function in `bin/cli.js` (line 993) currently reads from `runtime/`:

```javascript
// Current (lines 1048-1058)
// Sync runtime/ -> ~/.aether/system/ (clean production files)
// runtime/ is generated during publish - explicit allowlist via sync-to-runtime.sh
const runtimeSrc = path.join(PACKAGE_DIR, 'runtime');
if (fs.existsSync(runtimeSrc)) {
  const result = syncAetherToHub(runtimeSrc, HUB_SYSTEM_DIR);
  log(`  Hub system: ${result.copied} files, ${result.skipped} unchanged -> ${HUB_SYSTEM_DIR}`);
```

After Phase 20, it must read from `.aether/` directly:

```javascript
// After Phase 20
// Sync .aether/ -> ~/.aether/system/ (direct packaging, no staging)
const aetherSrc = path.join(PACKAGE_DIR, '.aether');
if (fs.existsSync(aetherSrc)) {
  const result = syncAetherToHub(aetherSrc, HUB_SYSTEM_DIR);
  log(`  Hub system: ${result.copied} files, ${result.skipped} unchanged -> ${HUB_SYSTEM_DIR}`);
```

There is also a secondary reference at lines 1111-1114 for syncing rules:
```javascript
// Current (line 1112)
const rulesSrc = path.join(PACKAGE_DIR, 'runtime', 'rules');
```
This must change to point to `.aether/rules/`.

The `syncAetherToHub` function (line 897) already has the exclude logic (`shouldExcludeFromHub` with `HUB_EXCLUDE_DIRS = ['data', 'dreams', 'checkpoints', 'locks', 'temp']` at line 878) that correctly skips private directories. No new exclusion logic is needed -- the existing function works correctly when pointed at `.aether/` instead of `runtime/`.

### Pattern 4: Unified Distribution Pipeline

Currently there are three separate distribution paths in `setupHub()` (lines 993-1156):

1. **System files:** `runtime/` -> `~/.aether/system/` (lines 1048-1058, via `syncAetherToHub`)
2. **Commands:** `.claude/commands/ant/` + `.opencode/commands/ant/` -> hub commands dirs (lines 1076-1098, via `syncDirWithCleanup`)
3. **Agents:** `.opencode/agents/` -> `~/.aether/system/agents/` (lines 1100-1109, via `syncDirWithCleanup`)

After Phase 20, since `.aether/` is published directly, the system files path becomes `.aether/` -> `~/.aether/system/`. Commands and agents remain separate paths because they come from different source directories (`.claude/`, `.opencode/`). The unification means all three paths use consistent exclude-based logic and are grouped into a single `setupHub()` function without any allowlist dependency.

### Pattern 5: Allowlist vs Exclude-List Decision

**Current approach:** Explicit allowlist in `sync-to-runtime.sh` (83 files) and `update-transaction.js` (62 files) -- already out of sync with each other and with cli.js (59 files).

**Recommendation: Switch to exclude-list for both publishing AND hub-to-repo sync.**

Rationale:
- For publishing (npm package): exclude-list is better. New files added to `.aether/` are automatically included without needing to update a list. The private directories are stable and won't accidentally get new publishable files.
- For `update-transaction.js` (the `syncSystemFilesWithCleanup` method): the explicit allowlist should be replaced with the exclude-based `syncAetherToRepo` method that already exists at lines 789-889. This method already uses `shouldExclude()` (line 776) which checks `EXCLUDE_DIRS = ['data', 'dreams', 'checkpoints', 'locks', 'temp', 'agents', 'commands', 'rules']` at line 173.
- The `SYSTEM_FILES` array appears in THREE places: `cli.js` (lines 380-439, 59 entries), `update-transaction.js` (lines 179-240, 62 entries), and `sync-to-runtime.sh` (lines 23-84, 62 entries). They are already out of sync. This maintenance burden disappears with exclude-based approach.

**Impact:** The `SYSTEM_FILES` array in `cli.js` (line 380), the `SYSTEM_FILES` array in `update-transaction.js` (line 179), the `copySystemFiles()` function in `cli.js` (line 441), and the `syncSystemFilesWithCleanup()` function in both cli.js (line 628) and `update-transaction.js` (line 718) all become obsolete.

### Anti-Patterns to Avoid

- **Partial runtime/ removal:** Do not leave a `runtime/` shell or redirect file. The decision is full deletion -- anything else creates confusion about which directory is authoritative.
- **Forgetting the rules sync:** Line 1112 in `setupHub()` has a separate path for `runtime/rules`. Both the main system sync and the rules sync must be updated.
- **Leaving SYSTEM_FILES in update-transaction.js:** If the allowlist remains but now points to non-existent runtime/ paths, hub-to-repo sync will silently copy nothing. The allowlist must be removed and replaced with the exclude-based pattern already used in `syncAetherToRepo`.
- **Not updating the pre-commit hook before removing runtime/:** The pre-commit hook (`.git/hooks/pre-commit`) checks for direct edits to `runtime/` and auto-runs sync. After runtime/ is deleted, this hook will error if `sync-to-runtime.sh` no longer exists. The hook must be updated at the same time.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Private dir exclusion from npm | Custom pre-publish filter | `.npmignore` patterns | npm's built-in exclusion; reliable, documented, standard |
| Package content preview | Custom file listing script | `npm pack --dry-run` | Built-in npm command; shows exactly what would be published |
| Version detection | Parsing version strings manually | Semver comparison already in Node.js | Version data already in `version.json` and `package.json` |

**Key insight:** npm's `.npmignore` + `files` field combination is the correct tool for controlling what gets published. The sync script was a workaround for having private and public files co-mingled in one directory -- now that we're accepting that co-mingling and relying on `.npmignore`, no custom tooling is needed.

**Validation command:** `npm pack --dry-run` shows exactly what files would be in the package before publishing. This is the implementation mechanism for the requested "dry-run mode that shows exactly what would be published."

---

## Common Pitfalls

### Pitfall 1: SYSTEM_FILES Allowlist Left in update-transaction.js

**What goes wrong:** The `SYSTEM_FILES` array in `update-transaction.js` (lines 179-240) is used by `syncSystemFilesWithCleanup()` (line 718) to know which files to copy from hub to repo. If runtime/ is deleted but this allowlist remains, it will still work during hub-to-repo sync (because the hub content is unchanged). However, it becomes a maintenance liability -- new files added to `.aether/` won't be distributed unless manually added to this list.

**Why it happens:** The allowlist was a deliberate safety measure to prevent accidental distribution of private files. After Phase 20, the safety net moves to `.npmignore` and `shouldExclude()`.

**How to avoid:** Replace `syncSystemFilesWithCleanup` usages (line 718 in update-transaction.js, line 628 in cli.js) with `syncAetherToRepo` calls (line 789 in update-transaction.js, already implemented). Remove the `SYSTEM_FILES` array from all three locations. The `EXCLUDE_DIRS` array `['data', 'dreams', 'checkpoints', 'locks', 'temp', 'agents', 'commands', 'rules']` at update-transaction.js line 173 already captures the right exclusions.

**Warning signs:** If you see `SYSTEM_FILES` referenced after Phase 20, it's a sign the old allowlist pattern wasn't fully removed.

### Pitfall 2: Private Data Exposure via .npmignore Mistakes

**What goes wrong:** If a new private directory is created in `.aether/` but not added to `.npmignore`, it gets published in the next npm release.

**Why it happens:** Moving from allowlist to exclude-list means new additions are opt-out by default. With the old allowlist, new directories were opt-in.

**How to avoid:** The pre-packaging validation script should enumerate known private directories and warn if any are found that are NOT in `.npmignore`. Document the convention clearly in the validation script and in the updated CLAUDE.md.

**Auto-check severity recommendation:** Hard block (non-zero exit) on detecting known private directories (`data/`, `dreams/`, `oracle/`, `checkpoints/`) not covered by `.npmignore`. Warn (but continue) on detecting unexpected new directories in `.aether/` that aren't explicitly allowed or excluded.

### Pitfall 3: Hub Already Populated -- Migration is Structural Not Data

**What goes wrong:** Developers assume existing hub users (`~/.aether/system/`) need their files re-synced after Phase 20. They don't -- the files at `~/.aether/system/` were already synced from the same source (`.aether/` via `runtime/`). The content is identical.

**Why it happens:** Confusion about what changed: the packaging pipeline changed, not the distributed content.

**How to avoid:** The migration message should be informational only: "The distribution pipeline has been simplified -- runtime/ staging directory has been removed. Your colony is unaffected." No file migration needed.

**Warning signs:** If you write migration code that copies files at `aether update` time, that's over-engineering. The only cleanup needed is a cosmetic removal of any stale `runtime/` reference in documentation or error messages.

### Pitfall 4: Pre-Commit Hook Timing

**What goes wrong:** The pre-commit hook (`.git/hooks/pre-commit`, 43 lines) currently:
1. Blocks direct edits to `runtime/` (lines 7-19)
2. Runs `sync-to-runtime.sh` if `.aether/` files changed (lines 22-33)
3. Stages synced `runtime/` changes (line 32)

After deletion of `runtime/`, step 2 will fail because `bin/sync-to-runtime.sh` won't exist.

**Why it happens:** The hook runs `bash bin/sync-to-runtime.sh` which won't exist after Phase 20.

**How to avoid:** Update the pre-commit hook as part of the same task that deletes `sync-to-runtime.sh`. The replacement hook should: (1) run `bin/validate-package.sh` to check `.aether/` integrity, (2) remove the runtime/ check entirely.

**Recommendation:** Repurpose the pre-commit hook as a validation-only check. Run `bin/validate-package.sh --check` in non-blocking mode (warn but don't block commits). The hook's original purpose was guarding against runtime/ drift -- that problem disappears when runtime/ disappears.

### Pitfall 5: Duplicate SYSTEM_FILES in Three Files

**What goes wrong:** The `SYSTEM_FILES` array appears in THREE places with DIFFERENT contents:
- `bin/cli.js` lines 380-439 (59 entries -- missing `docs/caste-system.md`, `docs/error-codes.md`, `docs/queen-commands.md`, `rules/aether-colony.md`, `templates/QUEEN.md.template`; has `planning.md` which the others don't)
- `bin/lib/update-transaction.js` lines 179-240 (62 entries)
- `bin/sync-to-runtime.sh` lines 23-84 (62 entries, matching update-transaction.js)

These are ALREADY out of sync. After Phase 20, all three must be removed consistently.

**How to avoid:** Remove all three occurrences in the same plan step. Do not update one and leave the others.

### Pitfall 6: queen-init template path references runtime/

**What goes wrong:** In `aether-utils.sh` at line 3383, `queen-init` checks `"$AETHER_ROOT/runtime/templates/QUEEN.md.template"` as one of its template lookup paths. After runtime/ is deleted, this path silently fails (the loop continues to the next path). This is acceptable behavior -- the hub path at `$HOME/.aether/system/templates/` is checked first -- but the dead reference should be cleaned up.

**How to avoid:** Remove the `runtime/` path from the queen-init template lookup array in `aether-utils.sh`. The lookup order after Phase 20 should be:
1. `$HOME/.aether/system/templates/QUEEN.md.template` (hub -- primary, line 3382)
2. `$AETHER_ROOT/.aether/templates/QUEEN.md.template` (dev repo -- secondary, line 3384)
3. `$HOME/.aether/templates/QUEEN.md.template` (legacy hub -- fallback, line 3385)

Also update line 3405 where the error message JSON lists `runtime/templates/QUEEN.md.template` in the checked paths array.

### Pitfall 7: autofix-checkpoint references runtime in target_dirs

**What goes wrong:** In `aether-utils.sh` at line 1724, the `autofix-checkpoint` command includes `runtime` in its `target_dirs` string: `target_dirs=".aether .claude/commands/ant .claude/commands/st .opencode runtime bin"`. After runtime/ is deleted, git status/stash commands targeting `runtime` will produce harmless warnings but should be cleaned up.

**How to avoid:** Remove `runtime` from the `target_dirs` variable at line 1724.

---

## Code Examples

Verified patterns from direct codebase inspection:

### Current setupHub() -- runtime/ read path (MUST CHANGE)
```javascript
// bin/cli.js lines 1048-1058 -- CURRENT
// Sync runtime/ -> ~/.aether/system/ (clean production files)
// runtime/ is generated during publish - explicit allowlist via sync-to-runtime.sh
const runtimeSrc = path.join(PACKAGE_DIR, 'runtime');
if (fs.existsSync(runtimeSrc)) {
  const result = syncAetherToHub(runtimeSrc, HUB_SYSTEM_DIR);
  log(`  Hub system: ${result.copied} files, ${result.skipped} unchanged -> ${HUB_SYSTEM_DIR}`);
  // ...
}
```

```javascript
// bin/cli.js -- AFTER Phase 20
// Sync .aether/ -> ~/.aether/system/ (direct packaging, no staging)
const aetherSrc = path.join(PACKAGE_DIR, '.aether');
if (fs.existsSync(aetherSrc)) {
  const result = syncAetherToHub(aetherSrc, HUB_SYSTEM_DIR);
  log(`  Hub system: ${result.copied} files, ${result.skipped} unchanged -> ${HUB_SYSTEM_DIR}`);
  // ...
}
```

### Current rules sync path (MUST CHANGE)
```javascript
// bin/cli.js lines 1111-1119 -- CURRENT
// Sync rules/ from runtime -> ~/.aether/system/rules/
const rulesSrc = path.join(PACKAGE_DIR, 'runtime', 'rules');
```

```javascript
// AFTER Phase 20
const rulesSrc = path.join(PACKAGE_DIR, '.aether', 'rules');
```

### package.json files field (MUST CHANGE)
```json
// CURRENT (lines 8-19)
"files": [
  "bin/",
  ".claude/commands/ant/",
  ".opencode/commands/ant/",
  ".opencode/agents/",
  ".opencode/opencode.json",
  "runtime/",
  "README.md",
  "LICENSE",
  "DISCLAIMER.md",
  "CHANGELOG.md"
]
```

```json
// AFTER Phase 20
"files": [
  "bin/",
  ".claude/commands/ant/",
  ".opencode/commands/ant/",
  ".opencode/agents/",
  ".opencode/opencode.json",
  ".aether/",
  "README.md",
  "LICENSE",
  "DISCLAIMER.md",
  "CHANGELOG.md"
]
```

### npm lifecycle scripts (MUST CHANGE)
```json
// CURRENT (lines 21-24)
"preinstall": "bash bin/sync-to-runtime.sh 2>/dev/null || true",
"postinstall": "node bin/cli.js install --quiet",
"prepublishOnly": "bash bin/sync-to-runtime.sh",
"postpublish": "rm -rf runtime/"
```

```json
// AFTER Phase 20
"preinstall": "bash bin/validate-package.sh 2>/dev/null || true",
"postinstall": "node bin/cli.js install --quiet",
"prepublishOnly": "bash bin/validate-package.sh"
// postpublish removed entirely (no runtime/ to clean up)
```

### shouldExcludeFromHub() -- already correct, no change needed
```javascript
// bin/cli.js lines 877-888 -- NO CHANGE NEEDED
const HUB_EXCLUDE_DIRS = ['data', 'dreams', 'checkpoints', 'locks', 'temp'];

function shouldExcludeFromHub(relPath) {
  const parts = relPath.split(path.sep);
  return parts.some(part => HUB_EXCLUDE_DIRS.includes(part));
}
```

This function already correctly filters private directories. When `syncAetherToHub` is called with `.aether/` as source instead of `runtime/`, it will automatically skip `data/`, `dreams/`, etc.

### npm pack --dry-run (the dry-run mode)
```bash
# Shows exactly what would be published -- no publish occurs
npm pack --dry-run

# Example output:
# npm notice Files included in package:
# npm notice 485B   .aether/aether-utils.sh
# npm notice 12.4kB .aether/workers.md
# ...
# (data/, dreams/ etc. will NOT appear due to .npmignore)
```

This is the implementation of the requested dry-run mode. No custom code needed.

### Pre-commit hook -- AFTER Phase 20
```bash
#!/bin/bash
# Aether Pre-Commit Hook (v2 - simplified pipeline)
# runtime/ staging directory removed in v4.0.0

set -euo pipefail

# Run validation check (non-blocking -- warn but don't stop commits)
if [ -f "bin/validate-package.sh" ]; then
    echo "Running package validation..."
    bash bin/validate-package.sh 2>/dev/null || echo "Package validation completed with warnings"
fi

# Run lint check (optional, non-blocking)
if [ -f "package.json" ] && grep -q '"lint"' package.json; then
    echo "Running lint check..."
    npm run lint 2>/dev/null || echo "Lint check completed with warnings"
fi

exit 0
```

### Migration message format
```javascript
// In setupHub() or in aether update command output
const OLD_VERSION_THRESHOLD = '4.0.0';
const isUpgradeFromOldVersion = semver.lt(previousVersion, OLD_VERSION_THRESHOLD);

if (isUpgradeFromOldVersion) {
  log('');
  log('  Distribution pipeline simplified (v4.0 change):');
  log('  - runtime/ staging directory has been removed');
  log('  - .aether/ is now published directly (private dirs excluded)');
  log('  - Your colony state and data are unaffected');
  log('  - See CHANGELOG.md for details');
  log('');
}
```

---

## Complete Change Inventory

All files that require changes in Phase 20, organized by change type. **All line numbers verified against current main on 2026-02-19.**

### Delete (clean removal, no backup)
| File | Why |
|------|-----|
| `bin/sync-to-runtime.sh` (142 lines) | Replaced by `bin/validate-package.sh` |
| `runtime/` (entire directory) | Staging directory eliminated |

### Create
| File | Why |
|------|-----|
| `bin/validate-package.sh` | Replaces sync-to-runtime.sh with validation-only logic |

### Modify: npm/packaging config
| File | Lines | Change |
|------|-------|--------|
| `package.json` | Line 14 | Replace `"runtime/"` with `".aether/"` in `files` array |
| `package.json` | Line 21 | Change `preinstall` to call `validate-package.sh` |
| `package.json` | Line 23 | Change `prepublishOnly` to call `validate-package.sh` |
| `package.json` | Line 24 | Delete `postpublish` script (no runtime/ to clean) |
| `.npmignore` | Append | Add `.aether/data/`, `.aether/dreams/`, `.aether/oracle/`, `.aether/checkpoints/`, `.aether/locks/`, `.aether/temp/`, `.aether/archive/`, `.aether/chambers/`, `.aether/examples/`, `.aether/__pycache__/`, plus private root files |
| `.gitignore` | Lines 106-108 | Remove `runtime/` entry and associated comments |

### Modify: bin/cli.js (core logic)
| Lines | Change |
|-------|--------|
| 68 `COMMANDS_SRC` | Verify if `path.join(PACKAGE_DIR, 'commands', 'ant')` is still needed or can be simplified (fallback at line 1077-1079 already handles `.claude/commands/ant/`) |
| 379-439 `SYSTEM_FILES` | Delete the entire `SYSTEM_FILES` array (60 lines, becomes obsolete) |
| 441-456 `copySystemFiles()` | Delete function (uses SYSTEM_FILES) |
| 628-677 `syncSystemFilesWithCleanup()` | Delete function (uses SYSTEM_FILES) |
| 679-681 | Update comment "Note: runtime/ is generated during publish only, not checkpointed" |
| 1048-1058 `setupHub()` | Change `runtimeSrc` from `'runtime'` to `'.aether'` |
| 1111-1114 `setupHub()` | Change `rulesSrc` from `path.join(PACKAGE_DIR, 'runtime', 'rules')` to `path.join(PACKAGE_DIR, '.aether', 'rules')` |

### Modify: bin/lib/update-transaction.js
| Lines | Change |
|-------|--------|
| 178 | Update comment "must match bin/sync-to-runtime.sh SYSTEM_FILES exactly" |
| 179-240 `SYSTEM_FILES` | Delete the entire `SYSTEM_FILES` array (62 entries, becomes obsolete) |
| 718-768 `syncSystemFilesWithCleanup()` | Delete method (uses SYSTEM_FILES); the `syncFiles()` method at line 951 already uses `syncAetherToRepo()` (line 966) for system file sync |

### Modify: .aether/aether-utils.sh
| Line | Change |
|------|--------|
| 316 | Update CONTEXT block: remove `runtime/` mention from constraint text |
| 1724 | Remove `runtime` from `target_dirs` in `autofix-checkpoint` |
| 3379 | Update comment: remove "dev (runtime/)" from template search order description |
| 3383 | Remove `"$AETHER_ROOT/runtime/templates/QUEEN.md.template"` from template lookup array |
| 3405 | Remove `"runtime/templates/QUEEN.md.template"` from error message JSON's `templates_checked` array |
| 3792-3793 | Remove `elif [[ "$file" == runtime/* ]]; then is_system=true` branch |

### Modify: .aether/data/checkpoint-allowlist.json
| Change |
|--------|
| Remove `"runtime/**/*"` from `system_files` array |

### Modify: git hook
| File | Change |
|------|--------|
| `.git/hooks/pre-commit` (43 lines) | Rewrite completely: remove runtime/ guard and sync-to-runtime.sh call; replace with validate-package.sh call (non-blocking) |

### Modify: slash commands
| File | Lines | Change |
|------|-------|--------|
| `.claude/commands/ant/build.md` | 165-166 | Remove `runtime` from target directory list in checkpoint stash command |
| `.opencode/commands/ant/build.md` | 155-156 | Same as above (mirror) |

### Modify: documentation (all must be updated per locked decision)
| File | Change |
|------|--------|
| `CLAUDE.md` | Remove all runtime/ references; update architecture diagram (lines 23-28, 51, 58-68, 73, 88, 99, 103) and workflow table |
| `.opencode/OPENCODE.md` | Same as CLAUDE.md (lines 15-17, 32, 39, 46-48, 54, 76-77) |
| `RUNTIME UPDATE ARCHITECTURE.md` | Major rewrite -- remove runtime/ from all diagrams and flow descriptions; rename file to reflect new architecture |
| `.aether/CONTEXT.md` | Line 316: update constraint text to remove runtime/ mention |
| `.claude/rules/aether-specific.md` | Lines 7, 29: remove runtime/ references |
| `.claude/rules/git-workflow.md` | Lines 32, 42: remove runtime/ references |
| `.claude/rules/aether-development.md` | Lines 11-12, 17, 30-31: remove runtime/ references |
| `CHANGELOG.md` | Add entry for v4.0.0 structural change |
| `.aether/docs/known-issues.md` | Lines 26, 144-148: update ISSUE-004 (runtime/ path issue) as resolved; remove `runtime/**/*` from checkpoint classification |
| `.aether/docs/queen-commands.md` | Line 27: remove runtime/ from template search description |
| `.aether/docs/QUEEN-SYSTEM.md` | Line 88: update example output to not show runtime/ path |
| `.aether/docs/RECOVERY-PLAN.md` | Extensive runtime/ references throughout (lines 14, 25, 29, 99-100, 106, 108, 142-194, 219-222, 233-235, 241, 271) -- this entire doc needs updating |

### Verify/check (may not need change)
| File | Note |
|------|------|
| `tests/e2e/test-sta.sh` | STA-02 test checks for runtime/ path references in commands (lines 114-137). After Phase 20, runtime/ references in commands are legitimately gone. The test may need its description updated but should still PASS since no commands will reference runtime/. |
| `tests/e2e/test-ctx.sh` | Lines 88-100 verify COLONY_STATE.json is NOT referenced via runtime/. These tests remain valid -- they're checking that commands use `.aether/data/` paths. |
| `tests/bash/test-aether-utils.sh` | Lines 757-758, 766, 822-823 reference runtime/ in queen-init template tests. After removing the runtime/ lookup path from aether-utils.sh, these tests will need updating: line 766 tries to copy from `$PROJECT_ROOT/runtime/templates/QUEEN.md.template` -- change to `.aether/templates/`. Lines 757-758 and 822-823 delete runtime/ to simulate npm-install scenario -- this removal step becomes unnecessary. |

---

## State of the Art

| Old Approach | Current Approach | Impact |
|--------------|-----------------|--------|
| Separate staging directory copied on every install | Direct packaging from source | Eliminates double-maintenance of allowlist, removes copy step |
| Allowlist-based sync (3 separate lists of 59-62 files, already out of sync) | Exclude-list (9 private directories blocked) | New files auto-distributed without list updates |
| 3-step pipeline: edit -> sync -> package | 2-step: edit -> package (with validation) | Developer workflow simplified |

---

## Open Questions

1. **COMMANDS_SRC path at cli.js line 68**
   - What we know: `COMMANDS_SRC = path.join(PACKAGE_DIR, 'commands', 'ant')` -- this is a `commands/ant/` path at package root, not `runtime/commands/`. The fallback at lines 1077-1079 already handles the `.claude/commands/ant/` path correctly.
   - What's unclear: Whether this path is ever populated in the published package. In the current structure, `runtime/` has no `commands/` subdirectory -- the Claude commands come from `.claude/commands/ant/` directly.
   - Recommendation: Leave COMMANDS_SRC as-is; the fallback logic at lines 1077-1079 already handles "running from source" vs "running from package" correctly.

2. **CHECKPOINT_ALLOWLIST comment at cli.js line 681**
   - What we know: The comment says "Note: runtime/ is generated during publish only, not checkpointed."
   - What's unclear: Whether this comment needs updating after Phase 20 (it will be self-evident, but CLAUDE.md updates may cover it).
   - Recommendation: Update the comment to remove runtime/ reference; no logic change needed since runtime/ was never checkpointed.

3. **Major version bump number**
   - Current version: 3.1.19
   - Decision: Major version bump to signal structural change.
   - Recommendation: Bump to 4.0.0. The breaking change is that `runtime/` no longer exists -- any third-party tooling (unlikely, but possible) that references `runtime/` would break.

4. **HUB_EXCLUDE_DIRS alignment**
   - cli.js line 878: `HUB_EXCLUDE_DIRS = ['data', 'dreams', 'checkpoints', 'locks', 'temp']`
   - update-transaction.js line 173: `EXCLUDE_DIRS = ['data', 'dreams', 'checkpoints', 'locks', 'temp', 'agents', 'commands', 'rules']`
   - These differ: update-transaction.js also excludes `agents`, `commands`, `rules` (because those are synced separately from `.opencode/` and `.claude/`).
   - After Phase 20 when `.aether/` is the source, `syncAetherToHub` at cli.js line 897 will be called with `.aether/` as source. It must NOT sync `.aether/rules/` as system files -- rules are synced separately at line 1114. The existing `HUB_EXCLUDE_DIRS` may need `rules` added to match update-transaction.js behavior, OR the separate rules sync (line 1111) should be removed in favor of letting `syncAetherToHub` handle rules too.
   - Recommendation: Add `rules` to `HUB_EXCLUDE_DIRS` in cli.js since rules are synced by a separate dedicated step at line 1114. This keeps behavior identical to the current pipeline.

---

## Sources

### Primary (HIGH confidence -- direct codebase inspection)
- `/Users/callumcowie/repos/Aether/bin/sync-to-runtime.sh` -- full allowlist (62 files), script behavior (142 lines)
- `/Users/callumcowie/repos/Aether/bin/cli.js` -- setupHub() function (line 993), SYSTEM_FILES (lines 380-439), copySystemFiles (line 441), syncSystemFilesWithCleanup (line 628), HUB_EXCLUDE_DIRS (line 878), shouldExcludeFromHub (line 885), syncAetherToHub (line 897), runtimeSrc (line 1050), rulesSrc (line 1112), CHECKPOINT_ALLOWLIST (line 682), COMMANDS_SRC (line 68)
- `/Users/callumcowie/repos/Aether/bin/lib/update-transaction.js` -- SYSTEM_FILES (lines 179-240), EXCLUDE_DIRS (line 173), syncSystemFilesWithCleanup (line 718), syncAetherToRepo (line 789)
- `/Users/callumcowie/repos/Aether/package.json` -- files field (lines 8-19), lifecycle scripts (lines 21-24), version 3.1.19
- `/Users/callumcowie/repos/Aether/.npmignore` -- current exclusion patterns (39 lines)
- `/Users/callumcowie/repos/Aether/.gitignore` -- runtime/ entry at lines 106-108
- `/Users/callumcowie/repos/Aether/.git/hooks/pre-commit` -- current hook behavior (43 lines)
- `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh` -- runtime/ path references at lines 316, 1724, 3379, 3383, 3405, 3792-3793 (5557 total lines)
- `/Users/callumcowie/repos/Aether/.aether/data/checkpoint-allowlist.json` -- `runtime/**/*` in system_files
- `/Users/callumcowie/repos/Aether/.claude/commands/ant/build.md` -- runtime in target dirs at lines 165-166
- `/Users/callumcowie/repos/Aether/.opencode/commands/ant/build.md` -- runtime in target dirs at lines 155-156
- `/Users/callumcowie/repos/Aether/tests/bash/test-aether-utils.sh` -- queen-init template tests referencing runtime/ at lines 757-758, 766, 822-823
- `/Users/callumcowie/repos/Aether/tests/e2e/test-sta.sh` -- STA-02 test checking for runtime/ references at lines 114-137
- `/Users/callumcowie/repos/Aether/tests/e2e/test-ctx.sh` -- runtime/ reference validation at lines 88-100
- `/Users/callumcowie/repos/Aether/CLAUDE.md` -- documentation with runtime/ references at lines 25-27, 51, 58, 68, 73, 88, 99, 103
- `/Users/callumcowie/repos/Aether/.opencode/OPENCODE.md` -- documentation with runtime/ references at lines 15-17, 32, 39, 48, 54, 76-77
- `/Users/callumcowie/repos/Aether/RUNTIME UPDATE ARCHITECTURE.md` -- architecture doc (179 lines, needs major rewrite)
- `/Users/callumcowie/repos/Aether/.claude/rules/aether-specific.md` -- lines 7, 29
- `/Users/callumcowie/repos/Aether/.claude/rules/git-workflow.md` -- lines 32, 42
- `/Users/callumcowie/repos/Aether/.claude/rules/aether-development.md` -- lines 11-12, 17, 30-31
- `/Users/callumcowie/repos/Aether/.aether/docs/known-issues.md` -- lines 26, 144-148
- `/Users/callumcowie/repos/Aether/.aether/docs/queen-commands.md` -- line 27
- `/Users/callumcowie/repos/Aether/.aether/docs/QUEEN-SYSTEM.md` -- line 88
- `/Users/callumcowie/repos/Aether/.aether/docs/RECOVERY-PLAN.md` -- extensive runtime/ references (30+ lines)

### Secondary (MEDIUM confidence)
- npm documentation pattern for `files` + `.npmignore` interaction -- standard npm behavior, well-documented

---

## Metadata

**Confidence breakdown:**
- Complete change inventory: HIGH -- every file touching runtime/ found via comprehensive grep across entire repo, all line numbers re-verified against current main
- Core code changes (cli.js, package.json): HIGH -- exact line numbers and current code verified by reading actual files
- SYSTEM_FILES drift: HIGH -- directly compared all three arrays, confirmed 5 missing entries in cli.js version
- .npmignore pattern: HIGH -- existing .npmignore inspected, npm behavior is documented standard
- Migration approach: HIGH -- existing migration code in setupHub() (lines 998-1031) provides pattern
- Test impact: HIGH -- all test files with runtime/ references found and analyzed
- HUB_EXCLUDE_DIRS alignment concern: HIGH -- directly compared both exclude arrays

**Research date:** 2026-02-19
**Valid until:** 2026-03-19 (stable codebase -- no fast-moving external dependencies)
