# OpenCode Command Parsing Fix Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix OpenCode command parsing so `ant plan work on authentication` passes full text arguments to the command

**Architecture:** OpenCode appears to parse `ant plan [text]` differently than Claude Code. The `$ARGUMENTS` variable may not include text after the command name. We'll investigate the parsing difference and implement a fix that captures all arguments.

**Tech Stack:** OpenCode config, markdown command files, argument parsing

---

## Task 1: Investigate OpenCode Argument Parsing

**Files:**
- Read: `.opencode/opencode.json`
- Read: `.opencode/commands/ant/plan.md` (full)
- Compare: `.claude/commands/ant/plan.md` vs `.opencode/commands/ant/plan.md`

**Step 1: Document current behavior**

From TO-DOs.md:
> When running 'ant plan' with additional text, the command doesn't execute properly - If the user runs "ant plan" followed by any text (e.g., "ant plan work on the authentication"), the ant plan command doesn't run. Instead, it does a "plan" without actually executing the /ant:plan command

This suggests OpenCode parses `ant plan work on authentication` as just `ant plan`.

**Step 2: Check OpenCode documentation/patterns**

Research how OpenCode passes arguments:
1. Does it use `$ARGUMENTS` like Claude?
2. Is there a different variable like `$@` or `$*`?
3. Does OpenCode require explicit argument declaration in `opencode.json`?

Check if `.opencode/opencode.json` needs a `commands` section:
```json
{
  "$schema": "https://opencode.ai/config.json",
  "commands": {
    "ant:plan": {
      "arguments": [{"name": "text", "type": "string", "required": false}]
    }
  }
}
```

**Step 3: Document findings**

Write findings to scratch file:
```
Claude Code: Uses $ARGUMENTS, passes everything after command name
OpenCode: [to be determined]
Difference: [document what we find]
Fix approach: [document solution]
```

**Step 4: Commit**

```bash
git add docs/plans/
git commit -m "docs: opencode parsing fix plan created"
```

---

## Task 2: Test Current OpenCode Behavior

**Files:**
- Create: `test-opencode-args.md` (temporary test file)

**Step 1: Create debug command**

Create `.opencode/commands/ant/test-args.md`:
```markdown
---
name: ant:test-args
description: "Test argument parsing"
---

## Debug Output

Raw arguments received: `$ARGUMENTS`

Arguments length: ${#ARGUMENTS}

Arguments array: "$@"

---

## Instructions

Output exactly what was received:

```
ARGUMENTS = '$ARGUMENTS'
Length = ${#ARGUMENTS}
```

If `$ARGUMENTS` is empty, try alternative approaches:
- Check if `$1`, `$2`, etc. are set
- Check if `$@` contains values
- Document what OpenCode actually passes
```

**Step 2: Test in OpenCode**

Run in OpenCode:
```
/ant:test-args hello world test
```

Document what is received.

**Step 3: Clean up**

Remove test command after gathering data.

**Step 4: Commit findings**

```bash
git commit --allow-empty -m "research: opencode argument parsing behavior documented"
```

---

## Task 3: Implement OpenCode-Specific Argument Handling

Based on findings from Task 2, implement one of these solutions:

### Option A: If OpenCode uses different variable

**Files:**
- Modify: All `.opencode/commands/ant/*.md` files

Update argument parsing to check multiple sources:
```markdown
Parse arguments:
- Try `$ARGUMENTS` first
- If empty, try `$@`
- If empty, try `$1 $2 ...`
```

### Option B: If OpenCode requires explicit config

**Files:**
- Modify: `.opencode/opencode.json`

Add command definitions with argument schemas.

### Option C: If OpenCode passes arguments differently

**Files:**
- Modify: `.opencode/commands/ant/plan.md`

Change the command to explicitly accept and forward arguments.

**Step 1: Update plan.md**

Current:
```markdown
Parse `$ARGUMENTS`:
- If contains `--no-visual`: set `visual_mode = false`
```

If OpenCode passes args differently, add fallback:
```markdown
Parse arguments:
- Primary: Check `$ARGUMENTS` (Claude Code style)
- Fallback: Check `$@` or explicit params (OpenCode style)
- Combined: `args="${ARGUMENTS:-$@}"`

Then parse `$args` for flags:
- If `$args` contains `--no-visual`: set `visual_mode = false`
```

**Step 2: Apply to all affected commands**

Commands that need argument parsing:
- `plan.md` - accepts planning context text
- `build.md` - accepts phase number + flags
- `init.md` - accepts goal text
- `focus.md` - accepts focus text
- `redirect.md` - accepts redirect text
- `feedback.md` - accepts feedback text
- `flag.md` - accepts flag text
- `oracle.md` - accepts research topic

**Step 3: Commit**

```bash
git add .opencode/commands/ant/
git commit -m "fix: opencode argument parsing - support both \$ARGUMENTS and \$@"
```

---

## Task 4: Add Explicit Argument Documentation

**Files:**
- Modify: `.opencode/commands/ant/help.md`

**Step 1: Update help text**

Add section:
```markdown
## Command Syntax (OpenCode)

OpenCode requires explicit quotes for multi-word arguments:

```
# Correct:
/ant:init "Build a REST API"
/ant:plan "authentication system"

# May not work:
/ant:init Build a REST API  (parsed as just "Build")
```

If your arguments aren't being passed, wrap them in quotes.
```

**Step 2: Commit**

```bash
git add .opencode/commands/ant/help.md
git commit -m "docs: add opencode argument syntax guidance"
```

---

## Task 5: Create Argument Parsing Utility

**Files:**
- Modify: `.aether/aether-utils.sh`

**Step 1: Add normalize-args function**

```bash
# Normalize arguments from Claude Code ($ARGUMENTS) or OpenCode ($@)
normalize-args() {
  local args=""

  # Try Claude Code style first
  if [ -n "$ARGUMENTS" ]; then
    args="$ARGUMENTS"
  # Fall back to OpenCode style ($@)
  elif [ $# -gt 0 ]; then
    args="$@"
  fi

  echo "$args"
}
```

**Step 2: Update commands to use it**

Modify `.opencode/commands/ant/plan.md`:
```markdown
### Step 0: Normalize Arguments

Run: `bash .aether/aether-utils.sh normalize-args`

Capture output as `normalized_args`.

Then parse `$normalized_args` instead of `$ARGUMENTS`.
```

**Step 3: Test**

```bash
bash .aether/aether-utils.sh normalize-args "test args"
echo $?
```

**Step 4: Commit**

```bash
git add .aether/aether-utils.sh
git commit -m "feat: add normalize-args utility for cross-platform argument handling"
```

---

## Task 6: Update All OpenCode Commands

**Files:**
- Modify: All `.opencode/commands/ant/*.md`

**Step 1: Add normalize step to each command**

For each command file:
1. Add Step 0: Normalize Arguments
2. Change all `$ARGUMENTS` references to `$normalized_args`

Priority order:
1. `init.md` - most critical (goal text)
2. `plan.md` - second most critical (planning context)
3. `build.md` - phase + flags
4. Others as needed

**Step 2: Verify changes**

```bash
# Check all opencode files reference normalized_args
grep -l "ARGUMENTS" .opencode/commands/ant/*.md

# Should return nothing (all updated)
```

**Step 3: Commit**

```bash
git add .opencode/commands/ant/
git commit -m "fix: update all opencode commands to use normalized argument parsing"
```

---

## Task 7: Sync Check

**Files:**
- None (verification)

**Step 1: Verify Claude files unchanged**

Claude commands should still use `$ARGUMENTS` directly (works correctly).

```bash
grep "$ARGUMENTS" .claude/commands/ant/*.md | wc -l
# Should show ~22 occurrences (one per file)
```

**Step 2: Verify OpenCode files use normalized args**

```bash
grep "normalized_args" .opencode/commands/ant/*.md | wc -l
# Should show ~22 occurrences (one per file)
```

**Step 3: Run lint:sync**

```bash
npm run lint:sync
```

Should pass (or show only expected differences).

**Step 4: Commit**

```bash
git commit --allow-empty -m "test: verify argument parsing sync between Claude and OpenCode"
```

---

## Task 8: Documentation Update

**Files:**
- Modify: `.opencode/OPENCODE.md` (or create)

**Step 1: Document the fix**

```markdown
# OpenCode-Specific Notes

## Argument Parsing (Fixed 2026-02-15)

**Issue:** OpenCode doesn't pass `$ARGUMENTS` the same way as Claude Code.

**Fix:** All commands now use `normalize-args` helper that checks:
1. `$ARGUMENTS` (Claude Code style)
2. `$@` (OpenCode style)

**For Users:**
If argument parsing issues persist, wrap multi-word arguments in quotes:
```
/ant:init "Build a REST API"   # Always works
/ant:init Build a REST API      # May be truncated
```
```

**Step 2: Commit**

```bash
git add .opencode/OPENCODE.md
git commit -m "docs: document opencode argument parsing fix"
```

---

## Summary

| Task | Description | Files Modified |
|------|-------------|----------------|
| 1 | Investigate parsing behavior | None (research) |
| 2 | Test current behavior | Temporary test file |
| 3 | Implement argument handling | Core command files |
| 4 | Update help documentation | `help.md` |
| 5 | Create normalize utility | `aether-utils.sh` |
| 6 | Update all OpenCode commands | `.opencode/commands/ant/*.md` |
| 7 | Sync verification | None |
| 8 | Documentation | `OPENCODE.md` |

**Expected outcome:** OpenCode commands correctly receive full text arguments, matching Claude Code behavior.

**Testing:** After implementation, verify:
```
/ant:init "Test goal with multiple words"
# Should receive: "Test goal with multiple words"
```
