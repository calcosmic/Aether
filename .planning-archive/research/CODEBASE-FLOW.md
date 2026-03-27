# Codebase Flow Research: Update System and Learning Pipeline

**Project:** Aether v6.0 Brownfield Milestone
**Researched:** 2026-02-22
**Focus:** Update system, learning pipeline, QUEEN.md flow

---

## Executive Summary

This research documents the actual execution flow of three critical systems that need fixing/wiring in Aether v6.0:

1. **Update System** — Has working two-phase commit but potential issues with file overwriting
2. **Learning Pipeline** — Functions exist but integration points are incomplete
3. **QUEEN.md Flow** — colony-prime extracts wisdom but verification needed on actual usage

---

## 1. Update System Flow

### 1.1 Entry Points

| Command | Location | Function Called |
|---------|----------|-----------------|
| `aether update` | `bin/cli.js:1257` | `updateRepo()` |
| `aether update --all` | `bin/cli.js:1302` | `updateRepo()` (loop) |
| `aether install` | `bin/cli.js:1182` | `syncDirWithCleanup()` |

### 1.2 syncDirWithCleanup Implementation

**Location:** `bin/cli.js:484-559`

```javascript
function syncDirWithCleanup(src, dest, opts) {
  // 1. Create dest directory if missing
  fs.mkdirSync(dest, { recursive: true });

  // 2. Copy phase with hash comparison
  for (const relPath of srcFiles) {
    // Hash comparison: only copy if file doesn't exist or hash differs
    if (fs.existsSync(destPath)) {
      const srcHash = hashFileSync(srcPath);
      const destHash = hashFileSync(destPath);
      if (srcHash === destHash) {
        shouldCopy = false;  // SKIP - hashes match
      }
    }
    if (shouldCopy) {
      fs.copyFileSync(srcPath, destPath);  // OVERWRITE
    }
  }

  // 3. Cleanup phase — remove files in dest that aren't in src
  for (const relPath of destFiles) {
    if (!srcSet.has(relPath)) {
      fs.unlinkSync(path.join(dest, relPath));  // DELETE stale files
    }
  }
}
```

### 1.3 Two-Phase Commit (UpdateTransaction)

**Location:** `bin/lib/update-transaction.js:1322-1446`

**Phase Flow:**
1. **PREPARE** (line 1339) — Create git stash checkpoint
2. **SYNC** (line 1362) — Copy files from hub to repo
3. **VERIFY** (line 1371) — Hash verification of all synced files
4. **COMMIT** (line 1392) — Update version.json

**Key Methods:**
- `syncDirWithCleanup()` — line 542-626 (in UpdateTransaction class, mirrors cli.js version)
- `syncAetherToRepo()` — line 667-767 (excludes user data directories)
- `verifyIntegrity()` — line 889-934 (compares hub vs repo hashes)

### 1.4 Excluded Directories (User Data Protection)

**Location:** `bin/lib/update-transaction.js:175`

```javascript
this.EXCLUDE_DIRS = ['data', 'dreams', 'checkpoints', 'locks', 'temp', 'agents', 'commands', 'rules', 'archive', 'chambers'];
```

These are excluded from sync TO repo (but old files may still exist IN repo).

### 1.5 Identified Gaps in Update System

| Gap | Location | Issue |
|-----|----------|-------|
| **G-1** | `syncDirWithCleanup:511-518` | If dest file exists and hash DIFFERS, it copies. But if copy fails mid-write, file is corrupted. No atomic write. |
| **G-2** | `syncAetherToRepo:698-724` | Counter bug: `copied++` happens even in dry-run mode (line 723) outside the `if (!dryRun)` block. |
| **G-3** | `cleanupStaleAetherDirs:780-821` | Only cleans `.aether/agents/`, `.aether/commands/`, `.aether/planning.md`. May miss other stale files. |
| **G-4** | `verifyIntegrity:906-919` | Only verifies files FROM hub. Files that exist ONLY in repo (stale) aren't flagged during verify. |

---

## 2. Learning Pipeline Flow

### 2.1 Function Reference

| Function | Location | Purpose |
|----------|----------|---------|
| `learning-observe` | `aether-utils.sh:4407-4551` | Record observation with content hash deduplication |
| `learning-check-promotion` | `aether-utils.sh:4553-4605` | Find observations meeting thresholds |
| `learning-approve-proposals` | `aether-utils.sh:5044-5305` | Interactive approval workflow |
| `queen-promote` | `aether-utils.sh:4102-4300` | Write promoted wisdom to QUEEN.md |

### 2.2 learning-observe Flow

**Location:** `aether-utils.sh:4407-4551`

**Data Flow:**
1. Validate content, wisdom_type, colony_name (lines 4416-4425)
2. Generate SHA256 hash of content (line 4428)
3. Initialize `learning-observations.json` if missing (lines 4437-4439)
4. Check for existing observation by hash (line 4456)
5. **If exists:** Increment count, update last_seen, add colony (lines 4458-4476)
6. **If new:** Create entry with count=1 (lines 4482-4506)
7. Return threshold status (lines 4514-4550)

**Thresholds (line 4515-4523):**
```bash
philosophy) threshold=1 ;;  # Was 5
pattern) threshold=1 ;;      # Was 3
redirect) threshold=1 ;;     # Was 2
stack) threshold=1 ;;        # Unchanged
decree) threshold=0 ;;       # Unchanged
failure) threshold=1 ;;      # NEW
```

### 2.3 learning-check-promotion Flow

**Location:** `aether-utils.sh:4553-4605`

**Returns:** JSON with `proposals` array

```bash
# File doesn't exist -> empty proposals
if [[ ! -f "$observations_file" ]]; then
  json_ok '{"proposals":[]}'
fi

# Build proposals using jq - filters observations where:
# observation_count >= threshold for the wisdom_type
```

### 2.4 learning-approve-proposals Flow

**Location:** `aether-utils.sh:5044-5305`

**Execution Flow:**
1. Parse args (--verbose, --dry-run, --yes, --deferred)
2. Get colony name from COLONY_STATE.json (line 5075-5077)
3. Load proposals (from deferred file or learning-select-proposals)
4. **If no proposals:** Exit silently with `{"promoted":0,"deferred":0}`
5. Display proposals with checkbox UI (line 5147-5150)
6. Capture user selection (line 5165-5167)
7. **For each selected:** Call `queen-promote` (line 5229)
8. **For unselected:** Move to deferred (line 5257)
9. Offer undo if promotions succeeded (line 5269-5294)

### 2.5 Where learning-observe Is Called

| Caller | Location | Context |
|--------|----------|---------|
| `continue.md` | Step 2.5, lines 1073-1091 | After phase advancement, records each learning claim |
| `build.md` | Step 5.2, lines 765-768 | When builder fails, records failure observation |
| `build.md` | Step 5.7, lines 1157-1160 | When chaos finds resilience issues |
| `build.md` | Step 5.8, lines 1194-1197 | When watcher verification fails |

### 2.6 Identified Gaps in Learning Pipeline

| Gap | Location | Issue |
|-----|----------|-------|
| **L-1** | `continue.md:1073-1091` | Records observations but NO CHECK if threshold met. Should call `learning-check-promotion` after recording. |
| **L-2** | `continue.md:1219-1260` | Calls `learning-approve-proposals` but only if proposals exist. If file missing, silent skip. No creation of missing `learning-observations.json`. |
| **L-3** | `build.md:765-768` | Records failure observations but these may never be promoted because `learning-approve-proposals` only runs in `continue.md`. |
| **L-4** | `aether-utils.sh:5075-5077` | Gets colony name from COLONY_STATE.json but doesn't validate file exists first. May fail with jq error. |

---

## 3. QUEEN.md Flow

### 3.1 colony-prime Implementation

**Location:** `aether-utils.sh:6452-6654`

**Purpose:** Combine wisdom (QUEEN.md) + signals (pheromones) + instincts into unified worker context

**Execution Flow:**
1. Define paths (lines 6458-6459):
   - `cp_global_queen="$HOME/.aether/QUEEN.md"`
   - `cp_local_queen="$AETHER_ROOT/.aether/QUEEN.md"`
2. Helper function `_extract_wisdom()` (lines 6472-6501):
   - Uses awk to find section line numbers
   - Extracts: Philosophies, Patterns, Redirects, Stack Wisdom, Decrees
3. Load global QUEEN.md (lines 6504-6507)
4. Load local QUEEN.md (lines 6509-6513)
5. **FAIL HARD** if neither exists (lines 6516-6521)
6. Combine wisdom (local extends global) (lines 6525-6542)
7. Get metadata from local or global (lines 6544-6560)
8. Call `pheromone-prime` for signals (lines 6567-6571)
9. Build final prompt_section (lines 6584-6619)
10. Return JSON with wisdom, signals, prompt_section

### 3.2 queen-promote Implementation

**Location:** `aether-utils.sh:4102-4300`

**Execution Flow:**
1. Validate args: type, content, colony_name (lines 4111-4113)
2. Validate wisdom_type against valid types (lines 4115-4121)
3. Check QUEEN.md exists (lines 4123-4129)
4. Extract METADATA for thresholds (lines 4131-4145)
5. **QUEEN-04:** Check threshold against learning-observations.json (lines 4147-4168)
6. Map type to section header (lines 4172-4179)
7. Build entry: `- **${colony_name}** (${ts}): ${content}` (line 4182)
8. Find section boundaries using grep (lines 4188-4194)
9. Check for placeholder text (line 4197)
10. **If placeholder:** Replace with entry (lines 4200-4212)
11. **If no placeholder:** Insert after description paragraph (lines 4213-4234)
12. Update Evolution Log (lines 4237-4242)
13. Update METADATA stats (lines 4244-4286)
14. Atomic move: `mv "$tmp_file" "$queen_file"` (line 4302)

### 3.3 Where colony-prime Output Is Used

| Consumer | Location | How Used |
|----------|----------|----------|
| `build.md` | Step 4, lines 228-257 | Called via bash, extracts `prompt_section`, displayed to user, stored for worker injection |
| Builder prompts | `build.md:677` | Injected as `{ prompt_section }` variable |
| Watcher prompts | `build.md:871` | Injected as `{ prompt_section }` variable |
| Chaos prompts | `build.md:1098` | Injected as `{ prompt_section }` variable |

### 3.4 Identified Gaps in QUEEN.md Flow

| Gap | Location | Issue |
|-----|----------|-------|
| **Q-1** | `aether-utils.sh:6567-6571` | `pheromone-prime` called but errors silently swallowed with `|| true`. If pheromone-prime fails, signals are empty with no warning. |
| **Q-2** | `build.md:233-240` | colony-prime output parsed but if `.ok` is false, it "FAIL HARD" and stops. But what if JSON is malformed? No fallback. |
| **Q-3** | `build.md:256` | `prompt_section` is stored for injection but NO VERIFICATION that workers actually receive it. Agent definitions don't explicitly reference this variable. |
| **Q-4** | `aether-utils.sh:4150-4168` | Threshold check requires `learning-observations.json` to exist. If it doesn't exist, promotion fails with "No observations found". |

---

## 4. Data File Locations

| File | Location | Created By | Used By |
|------|----------|------------|---------|
| `learning-observations.json` | `.aether/data/` | `learning-observe` | `learning-check-promotion`, `queen-promote` |
| `learning-deferred.json` | `.aether/data/` | `learning-defer-proposals` | `learning-approve-proposals --deferred` |
| `.promotion-undo.json` | `.aether/data/` | `learning-approve-proposals` | `learning-undo-promotions` |
| `QUEEN.md` | `.aether/` or `~/.aether/` | `queen-promote` | `colony-prime` |
| `pheromones.json` | `.aether/data/` | `pheromone-write` | `colony-prime`, `pheromone-prime` |

---

## 5. Implementation Recommendations

### Phase 42: Fix Update System

1. **Fix atomic writes** (G-1): Use temp file + rename pattern in `syncDirWithCleanup`
2. **Fix counter bug** (G-2): Move `copied++` inside the `if (!dryRun)` block
3. **Expand stale cleanup** (G-3): Add more patterns to `cleanupStaleAetherDirs`
4. **Add stale file detection** (G-4): Verify no extra files exist in repo after sync

### Phase 43: Wire Learning Pipeline

1. **Add threshold check** (L-1): After recording observations in `continue.md`, call `learning-check-promotion`
2. **Ensure file creation** (L-2): Initialize `learning-observations.json` in `/ant:init` if not exists
3. **Add approval to build** (L-3): Optionally call `learning-approve-proposals` at build end if failures recorded
4. **Add validation** (L-4): Check COLONY_STATE.json exists before jq in `learning-approve-proposals`

### Phase 45: Verify QUEEN.md Flow

1. **Add error handling** (Q-1): Warn if pheromone-prime fails, don't silently continue
2. **Add JSON validation** (Q-2): Validate colony-prime output before parsing
3. **Verify worker injection** (Q-3): Check agent definitions reference `prompt_section` correctly
4. **Auto-create observations file** (Q-4): Create empty `learning-observations.json` if missing during promotion check

---

## 6. Source References

### Update System
- `bin/cli.js:484-559` — syncDirWithCleanup
- `bin/cli.js:1073-1159` — updateRepo function
- `bin/cli.js:1257-1454` — update command
- `bin/lib/update-transaction.js:1322-1446` — UpdateTransaction.execute

### Learning Pipeline
- `.aether/aether-utils.sh:4407-4551` — learning-observe
- `.aether/aether-utils.sh:4553-4605` — learning-check-promotion
- `.aether/aether-utils.sh:5044-5305` — learning-approve-proposals
- `.claude/commands/ant/continue.md:1073-1091` — observation recording
- `.claude/commands/ant/continue.md:1219-1260` — approval workflow

### QUEEN.md Flow
- `.aether/aether-utils.sh:6452-6654` — colony-prime
- `.aether/aether-utils.sh:4102-4300` — queen-promote
- `.claude/commands/ant/build.md:228-257` — colony-prime usage
- `.claude/commands/ant/build.md:677` — prompt_section injection

---

## 7. Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Update System | HIGH | Code paths clear, tested in production |
| Learning Pipeline | MEDIUM | Functions exist but integration points need verification |
| QUEEN.md Flow | MEDIUM | Extraction works but worker injection unverified |
| Gap Analysis | HIGH | Based on direct code reading with line numbers |

---

*Research complete. Ready for implementation planning.*
