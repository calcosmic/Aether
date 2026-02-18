# Pheromone Consumption — Active Signals Injection

**Created:** 2026-02-17
**Purpose:** Inject FOCUS/REDIRECT signals into worker prompts so they see user priorities when spawned

---

## Overview

Workers spawned during builds do not see current user priorities. FOCUS and REDIRECT signals exist but are not consumed by workers at spawn time. This feature adds an "Active Signals" section to builder prompts.

---

## Data Sources

| Signal | Source | Location |
|--------|--------|----------|
| FOCUS | constraints.json | `.aether/data/constraints.json` → `focus` array |
| REDIRECT | constraints.json | `.aether/data/constraints.json` → `constraints[].content` |

---

## Implementation

### 1. New `pheromone-read` Function

**Location:** `.aether/aether-utils.sh`

**Purpose:** Read active pheromones from constraints.json

**Logic:**
- Read `.aether/data/constraints.json`
- If file missing: return empty focus and constraints arrays
- Extract `focus` array → `priorities` in output
- Extract `constraints[].content` → `avoid` in output
- Return JSON with structure:
  ```json
  {
    "ok": true,
    "result": {
      "priorities": ["focus area 1", "focus area 2"],
      "avoid": [
        {"content": "constraint text", "source": "user:redirect"}
      ]
    }
  }
  ```

### 2. Modify `/ant:build` Command

**File:** `.claude/commands/ant/build.md`

**Location:** After Step 4.1.5 (Memory Priming), add Step 4.1.6: Pheromone Consumption

**Steps:**
1. Call `pheromone-read` after `memory-priming`
2. Parse JSON response
3. Set `pheromone_section` if any signals exist
4. Display summary: "Active Signals: {N} priorities, {M} constraints"

### 3. New Template: Active Signals Section

**Injection point:** After Colony Memory section (line ~494 in build.md)

**Template:**
```
--- ACTIVE SIGNALS (From User) ---

🎯 PRIORITIES (FOCUS):
{ for each priority }
- {priority}
{ endfor }

⚠️ CONSTRAINTS (REDIRECT - AVOID):
{ for each constraint }
- {constraint.content}
{ endfor }

--- END ACTIVE SIGNALS ---
```

**Display conditions:**
- Show if `priorities.length > 0` OR `avoid.length > 0`
- Empty = don't show section

### 4. Sync to OpenCode

Sync `.claude/commands/ant/build.md` → `.opencode/commands/ant/build.md`

---

## What About FEEDBACK?

FEEDBACK signals are low-priority "gentle adjustments" applied after work is done. Workers don't need them at spawn time. **Skipped for now.**

---

## Testing

```bash
# Test pheromone-read with no signals
bash .aether/aether-utils.sh pheromone-read
# Expected: {"ok":true,"result":{"priorities":[],"avoid":[]}}

# Test pheromone-read with active signals
# (Create constraints.json with test data first)
bash .aether/aether-utils.sh pheromone-read
# Expected: {"ok":true,"result":{"priorities":[...],"avoid":[...]}}

# Integration test
# Run /ant:build on a project with active pheromones
# Builders should see Active Signals section
```

---

## Files Modified

| File | Change |
|------|--------|
| `.aether/aether-utils.sh` | Add `pheromone-read` function |
| `.claude/commands/ant/build.md` | Add pheromone consumption step + template |
| `.opencode/commands/ant/build.md` | Sync from Claude Code |
