# Checkpoint Allowlist Fix Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix the build checkpoint system to never stash user data, only system files on an explicit allowlist

**Architecture:** The current system already uses path-based stashing (line 156 in build.md), but the TO-DO indicates it stashed 1,145 lines of user work (Oracle spec in TO-DOs.md). We'll verify the allowlist is correct, add user warnings, and ensure the stash only touches system files.

**Tech Stack:** Bash, git, JSON validation

---

## Task 1: Audit Current Checkpoint Implementation

**Files:**
- Read: `.claude/commands/ant/build.md` lines 146-165
- Read: `.opencode/commands/ant/build.md` lines 146-165
- Read: `TO-DOs.md` lines 9-11 (the bug description)

**Step 1: Verify current implementation**

Read both build.md files and document:
1. What paths are currently in the stash command
2. Whether `--include-untracked` is used (it shouldn't be)
3. What the checkpoint verification logic does

**Step 2: Identify the gap**

The TO-DO says the stash captured 1,145 lines from TO-DOs.md. Check:
- Is TO-DOs.md in the stash path list? (It shouldn't be)
- Was this a different code path (update vs build)?

**Step 3: Document findings**

Write findings to scratch file:
```
Current stash paths: .aether .claude/commands/ant .claude/commands/st .opencode runtime bin
TO-DOs.md included? No (and shouldn't be)
Root cause: [to be determined]
```

**Step 4: Commit**

```bash
git add docs/plans/
git commit -m "docs: checkpoint allowlist fix plan created"
```

---

## Task 2: Define Explicit System File Allowlist

**Files:**
- Create: `.aether/data/checkpoint-allowlist.json`
- Modify: `.aether/aether-utils.sh` (add checkpoint helper functions)

**Step 1: Create allowlist JSON**

```json
{
  "version": "1.0.0",
  "description": "Files safe for Aether to checkpoint/modify. NEVER touch files outside this list.",
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

**Step 2: Add checkpoint helper to aether-utils.sh**

Add function `checkpoint-check` that:
1. Reads allowlist JSON
2. Gets list of dirty files from git: `git status --porcelain`
3. Categorizes each file as "system" or "user"
4. If any user files are dirty, outputs warning with list
5. Returns only system files for stashing

```bash
checkpoint-check() {
  local allowlist_file=".aether/data/checkpoint-allowlist.json"

  if [ ! -f "$allowlist_file" ]; then
    echo '{"ok": false, "error": "Allowlist not found"}'
    return 1
  fi

  # Get dirty files
  local dirty_files=$(git status --porcelain | awk '{print $2}')

  # Check each against allowlist
  local system_files=()
  local user_files=()

  for file in $dirty_files; do
    if is_in_allowlist "$file"; then
      system_files+=("$file")
    else
      user_files+=("$file")
    fi
  done

  # Output JSON result
  jq -n \
    --argjson system "$(printf '%s\n' "${system_files[@]}" | jq -R . | jq -s .)" \
    --argjson user "$(printf '%s\n' "${user_files[@]}" | jq -R . | jq -s .)" \
    '{"ok": true, "system_files": $system, "user_files": $user, "has_user_files": ($user | length > 0)}'
}
```

**Step 3: Run test**

```bash
bash .aether/aether-utils.sh checkpoint-check
cat /tmp/checkpoint-test.json | jq .
```

Expected: JSON with system_files and user_files arrays

**Step 4: Commit**

```bash
git add .aether/data/checkpoint-allowlist.json .aether/aether-utils.sh
git commit -m "feat: add checkpoint allowlist system to protect user data"
```

---

## Task 3: Update Build.md Checkpoint Logic (Claude)

**Files:**
- Modify: `.claude/commands/ant/build.md` lines 146-165

**Step 1: Replace checkpoint section**

Current code (lines 146-165):
```bash
git rev-parse --git-dir 2>/dev/null
```

Replace with new checkpoint logic:
```markdown
### Step 3: Git Checkpoint

Create a git checkpoint for rollback capability (system files only).

```bash
git rev-parse --git-dir 2>/dev/null
```

- **If succeeds** (is a git repo):
  1. Run checkpoint check: `bash .aether/aether-utils.sh checkpoint-check`
  2. Parse JSON result
  3. **If `.has_user_files` is true**:
     - Display warning: `⚠️  User files detected in working tree - will NOT be stashed`
     - List user files: `echo "  - {file}"` for each in `.user_files`
     - Display: `Only system files will be checkpointed`
  4. **If `.system_files` array is non-empty**:
     - Create stash with ONLY system files: `git stash push -m "aether-checkpoint: pre-phase-$PHASE_NUMBER" -- {system_files...}`
     - Verify: `git stash list | head -1 | grep "aether-checkpoint"` — warn if empty
     - Store checkpoint as `{type: "stash", ref: "aether-checkpoint: pre-phase-$PHASE_NUMBER", files: [...]}`
  5. **If no system files dirty**:
     - Record `HEAD` hash via `git rev-parse HEAD`
     - Store checkpoint as `{type: "commit", ref: "$HEAD_HASH"}`
- **If fails** (not a git repo): Set checkpoint to `{type: "none", ref: "(not a git repo)"}`.

Rollback procedure: `git stash pop` (if type is "stash") or `git reset --hard $ref` (if type is "commit").

Output header:

```
🔨🐜🏗️🐜🔨 ═══════════════════════════════════════════════════
   B U I L D I N G   P H A S E   {id}
═══════════════════════════════════════════════════ 🔨🐜🏗️🐜🔨

📍 Phase {id}: {name}
💾 Git Checkpoint: {checkpoint_type} → {checkpoint_ref}
   {if user_files were skipped: "(user files preserved — not stashed)"}
🔄 Rollback: `git stash pop` (stash) or `git reset --hard {ref}` (commit)
```
```

**Step 2: Test the change**

1. Make a change to TO-DOs.md
2. Run `/ant:build 1`
3. Verify: Warning about user files appears
4. Verify: TO-DOs.md is NOT stashed
5. Verify: System files (if dirty) ARE stashed

**Step 3: Commit**

```bash
git add .claude/commands/ant/build.md
git commit -m "fix: checkpoint only stashes system files, warns about user data"
```

---

## Task 4: Update Build.md Checkpoint Logic (OpenCode)

**Files:**
- Modify: `.opencode/commands/ant/build.md` lines 146-165

**Step 1: Apply same changes**

Copy the exact same changes from Task 3 to the OpenCode version.

**Step 2: Verify sync**

```bash
diff .claude/commands/ant/build.md .opencode/commands/ant/build.md | grep -A5 -B5 "checkpoint"
```

Should show only the subagent_type differences, not checkpoint logic differences.

**Step 3: Commit**

```bash
git add .opencode/commands/ant/build.md
git commit -m "fix: sync checkpoint allowlist fix to OpenCode"
```

---

## Task 5: Add Update System Checkpoint Fix

**Files:**
- Read: `bin/cli.js` or update-related files
- Modify: Any file that does `git stash` during update

**Step 1: Find update stash code**

Search for other places that use `git stash`:
```bash
grep -r "git stash" --include="*.md" --include="*.js" --include="*.sh" .
```

**Step 2: Apply allowlist fix**

Any file using `git stash` must:
1. Use `checkpoint-check` first
2. Only stash system files from allowlist
3. Warn if user files are present

**Step 3: Commit**

```bash
git add [modified files]
git commit -m "fix: apply checkpoint allowlist to update system"
```

---

## Task 6: Integration Test

**Step 1: Create test scenario**

```bash
# Setup: Make changes to both system and user files
echo "# Test" >> .aether/aether-utils.sh
echo "# User note" >> TO-DOs.md
```

**Step 2: Run build**

Run `/ant:build 1 --no-visual`

**Step 3: Verify behavior**

Check output shows:
- Warning about user files (TO-DOs.md)
- System files checkpointed
- TO-DOs.md NOT in stash

Verify:
```bash
git stash list
git stash show -p  # Should NOT include TO-DOs.md
git diff TO-DOs.md  # Should still show your change
```

**Step 4: Commit test results**

```bash
git reset --hard HEAD  # Clean up test changes
git commit --allow-empty -m "test: verify checkpoint allowlist protects user data"
```

---

## Task 7: Documentation

**Files:**
- Modify: `.aether/docs/known-issues.md` (or create)

**Step 1: Document the fix**

```markdown
## Checkpoint System (Fixed 2026-02-15)

**Issue:** Build checkpoint could stash user work (TO-DOs.md, dreams, etc.)

**Fix:** Explicit allowlist system
- Only `.aether/aether-utils.sh`, `.claude/commands/ant/`, etc. are stashed
- User data (`.aether/data/`, `.aether/dreams/`, `TO-DOs.md`) is never touched
- Warning displayed if user files are present during checkpoint

**Files:** `.aether/data/checkpoint-allowlist.json`, `build.md`
```

**Step 2: Commit**

```bash
git add .aether/docs/known-issues.md
git commit -m "docs: document checkpoint allowlist fix"
```

---

## Summary

| Task | Description | Files Modified |
|------|-------------|----------------|
| 1 | Audit current implementation | None (research) |
| 2 | Create allowlist JSON + helpers | `checkpoint-allowlist.json`, `aether-utils.sh` |
| 3 | Update Claude build.md | `.claude/commands/ant/build.md` |
| 4 | Update OpenCode build.md | `.opencode/commands/ant/build.md` |
| 5 | Fix update system stashing | Various |
| 6 | Integration test | None |
| 7 | Documentation | `known-issues.md` |

**Expected outcome:** Checkpoint system will never stash user data; only system files on explicit allowlist are touched.
