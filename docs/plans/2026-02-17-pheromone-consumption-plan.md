# Pheromone Consumption Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Inject active FOCUS and REDIRECT signals into builder worker prompts so they see user priorities when spawned.

**Architecture:** Read pheromones from constraints.json, create new pheromone-read function, add Active Signals section to build command template.

**Tech Stack:** Bash shell scripting, JSON parsing with jq

---

## Task 1: Create pheromone-read Function

**Files:**
- Modify: `.aether/aether-utils.sh` (add after queen-read function ~line 3105)

**Step 1: Add pheromone-read function**

Add this function after the queen-read function (around line 3105):

```bash
  pheromone-read)
    # Read active pheromones (FOCUS/REDIRECT) from constraints.json
    constraints_file="$AETHER_ROOT/.aether/data/constraints.json"

    # Initialize defaults
    local priorities='[]'
    local avoid='[]'

    # Check if constraints file exists
    if [[ -f "$constraints_file" ]]; then
      # Read focus array as priorities
      priorities=$(jq -c '.focus // []' "$constraints_file" 2>/dev/null || echo '[]')

      # Read constraints array, extract content and source
      avoid=$(jq -c '[.constraints[]? | {content: .content, source: .source}] // []' "$constraints_file" 2>/dev/null || echo '[]')
    fi

    # Build JSON output
    local result
    result=$(jq -n \
      --argjson priorities "$priorities" \
      --argjson avoid "$avoid" \
      '{
        priorities: $priorities,
        avoid: $avoid
      }')

    json_ok "$result"
    ;;
```

**Step 2: Test pheromone-read with empty constraints**

Run: `bash .aether/aether-utils.sh pheromone-read`

Expected output:
```json
{"ok":true,"result":{"priorities":[],"avoid":[]}}
```

**Step 3: Commit**

```bash
git add .aether/aether-utils.sh
git commit -m "feat: add pheromone-read function for signal consumption"
```

---

## Task 2: Modify /ant:build Command

**Files:**
- Modify: `.claude/commands/ant/build.md`

**Step 1: Add pheromone consumption step after memory-priming**

In `.claude/commands/ant/build.md`, after Step 4.1.5 (around line 302), add:

```
### Step 4.1.6: Load Active Pheromones (Signal Consumption)

**This injects current FOCUS and REDIRECT signals into worker context.**

Call `pheromone-read` to get active signals:

```bash
bash .aether/aether-utils.sh pheromone-read 2>/dev/null
```

**Parse the JSON response:**
- If `.ok` is false or command fails: Set `pheromone_section = null` and skip
- If successful: Extract `.result.priorities` and `.result.avoid`

**Display summary:**
```
🎯 ACTIVE SIGNALS
=================
Priorities (FOCUS): {N}
Constraints (REDIRECT): {M}
```

**Store for worker injection:** The `pheromone_section` markdown will be included in builder prompts (see Step 5.1 Active Signals Section).
```

**Step 2: Add Active Signals template in builder prompt**

In `.claude/commands/ant/build.md`, after the Colony Memory Section (around line 543), add:

```
**Active Signals Section (injected if pheromones exist):**
```
--- ACTIVE SIGNALS (From User) ---

🎯 PRIORITIES (FOCUS):
{for each priority}
- {priority}
{endfor}

⚠️ CONSTRAINTS (REDIRECT - AVOID):
{for each constraint}
- {constraint.content}
{endfor}

--- END ACTIVE SIGNELS ---
```

**Step 3: Add injection point in builder prompt**

In the Builder Worker Prompt template (around line 494), add after colony_memory_section:

```
{ colony_memory_section if colony_memory exists }

{ pheromone_section if pheromone_section exists }
```

**Step 4: Commit**

```bash
git add .claude/commands/ant/build.md
git commit -m "feat: add pheromone consumption to build command"
```

---

## Task 3: Sync to OpenCode

**Files:**
- Modify: `.opencode/commands/ant/build.md`

**Step 1: Copy changes from Claude Code**

Copy the following sections from `.claude/commands/ant/build.md`:
- Step 4.1.6: Load Active Pheromones (Signal Consumption)
- Active Signals Section template
- Builder prompt injection point

**Step 2: Verify sync**

Run: `npm run lint:sync`

Expected: No sync errors

**Step 3: Commit**

```bash
git add .opencode/commands/ant/build.md
git commit -m "sync: update OpenCode build command with pheromone consumption"
```

---

## Task 4: Integration Test

**Files:**
- Test: Existing colony or create test constraints

**Step 1: Create test constraints**

```bash
echo '{
  "version": "1.0",
  "focus": ["test priority", "another focus"],
  "constraints": [
    {"id": "test_001", "type": "AVOID", "content": "test constraint", "source": "user:redirect"}
  ]
}' > .aether/data/constraints.json
```

**Step 2: Test pheromone-read**

Run: `bash .aether/aether-utils.sh pheromone-read`

Expected:
```json
{"ok":true,"result":{"priorities":["test priority","another focus"],"avoid":[{"content":"test constraint","source":"user:redirect"}]}}
```

**Step 3: Cleanup test data**

```bash
rm .aether/data/constraints.json
```

**Step 4: Final commit**

```bash
git add .
git commit -m "test: verify pheromone-read integration"
```

---

## Summary

| Task | Description |
|------|-------------|
| 1 | Add pheromone-read function to aether-utils.sh |
| 2 | Modify /ant:build command with signal consumption |
| 3 | Sync changes to OpenCode |
| 4 | Integration test |
