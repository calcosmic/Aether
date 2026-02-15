---
name: ant:entomb
description: "⚰️🐜⚰️ Entomb completed colony in chambers"
---

You are the **Queen**. Archive the completed colony to chambers.

## Instructions

Parse `$ARGUMENTS`:
- If contains `--no-visual`: set `visual_mode = false` (visual is ON by default)
- Otherwise: set `visual_mode = true`

### Step 0: Initialize Visual Mode (if enabled)

If `visual_mode` is true:
```bash
# Generate session ID
entomb_id="entomb-$(date +%s)"

# Initialize swarm display
bash .aether/aether-utils.sh swarm-display-init "$entomb_id"
bash .aether/aether-utils.sh swarm-display-update "Queen" "prime" "excavating" "Entombing colony" "Colony" '{"read":0,"grep":0,"edit":0,"bash":0}' 0 "fungus_garden" 0
```

### Step 1: Read State

Read `.aether/data/COLONY_STATE.json`.

If file missing or `goal: null`:
```
No colony to entomb. Run /ant:init first.
```
Stop here.

### Step 2: Validate Colony Can Be Entombed

Extract: `goal`, `state`, `current_phase`, `plan.phases`, `memory.decisions`, `memory.phase_learnings`.

**Precondition 1: All phases must be completed**

Check if all phases in `plan.phases` have `status: "completed"`:
```
all_completed = all(phase.status == "completed" for phase in plan.phases)
```

If NOT all completed:
```
Cannot entomb incomplete colony.

Completed phases: X of Y
Remaining: {list of incomplete phase names}

Run /ant:continue to complete remaining phases first.
```
Stop here.

**Precondition 2: State must not be EXECUTING**

If `state == "EXECUTING"`:
```
Colony is still executing. Run /ant:continue to reconcile first.
```
Stop here.

**Precondition 3: No critical errors**

Check `errors.records` for any entries with `severity: "critical"`.

If critical errors exist:
```
Cannot entomb colony with critical errors.

Critical errors: {count}
Run /ant:continue to resolve errors first.
```
Stop here.

### Step 3: Compute Milestone

Determine milestone based on phases completed:
- 0 phases: "Fresh Start"
- 1 phase: "First Mound"
- 2-4 phases: "Open Chambers"
- 5+ phases: "Sealed Chambers"

If all phases completed AND user explicitly sealing: "Crowned Anthill"

For entombment, use the computed milestone or extract from state if already set.

### Step 4: User Confirmation

Display:
```
🏺 ═══════════════════════════════════════════════════
   E N T O M B   C O L O N Y
══════════════════════════════════════════════════ 🏺

Goal: {goal}
Phases: {completed}/{total} completed
Milestone: {milestone}

Archive will include:
  - COLONY_STATE.json
  - manifest.json (pheromone trails)

This will reset the active colony. Continue? (yes/no)
```

Wait for explicit "yes" response before proceeding.

If user responds with anything other than "yes", display:
```
Entombment cancelled. Colony remains active.
```
Stop here.

### Step 5: Promote Wisdom to QUEEN.md

Before creating the chamber, promote validated learnings to QUEEN.md for future colonies.

**Step 5.1: Ensure QUEEN.md exists**

```bash
queen_file=".aether/docs/QUEEN.md"
if [[ ! -f "$queen_file" ]]; then
  init_result=$(bash .aether/aether-utils.sh queen-init 2>/dev/null || echo '{"ok":false}')
  init_ok=$(echo "$init_result" | jq -r '.ok // false')
  if [[ "$init_ok" == "true" ]]; then
    created=$(echo "$init_result" | jq -r '.result.created // false')
    if [[ "$created" == "true" ]]; then
      bash .aether/aether-utils.sh activity-log "CREATED" "Queen" "Initialized QUEEN.md for wisdom promotion"
    fi
  fi
fi
```

**Step 5.2: Extract and promote validated learnings**

```bash
# Extract colony name from goal (sanitized)
colony_name=$(jq -r '.goal' .aether/data/COLONY_STATE.json | tr '[:upper:]' '[:lower:]' | tr -cs '[:alnum:]' '-' | sed 's/^-//;s/-$//' | cut -c1-30)

# Extract validated learnings from phase_learnings
learnings=$(jq -c '.memory.phase_learnings // []' .aether/data/COLONY_STATE.json)

# Extract decisions
decisions=$(jq -c '.memory.decisions // []' .aether/data/COLONY_STATE.json)

promotion_count=0

# Promote patterns from validated learnings
if [[ -f "$queen_file" ]]; then
  # Process each phase's learnings
  echo "$learnings" | jq -c '.[]' 2>/dev/null | while read -r learning_group; do
    phase=$(echo "$learning_group" | jq -r '.phase // "unknown"')
    # Extract individual learnings and promote as patterns
    echo "$learning_group" | jq -r '.learnings[]? | select(.status == "validated") | .claim' 2>/dev/null | while read -r claim; do
      if [[ -n "$claim" && "$claim" != "null" ]]; then
        # Truncate if too long
        content=$(echo "$claim" | cut -c1-200)
        result=$(bash .aether/aether-utils.sh queen-promote "pattern" "$content" "$colony_name" 2>/dev/null || echo '{"ok":false}')
        if [[ $(echo "$result" | jq -r '.ok // false') == "true" ]]; then
          promotion_count=$((promotion_count + 1))
        fi
      fi
    done
  done

  # Promote high-confidence instincts as patterns
  instincts=$(jq -c '.memory.instincts // []' .aether/data/COLONY_STATE.json)
  echo "$instincts" | jq -c '.[]' 2>/dev/null | while read -r instinct; do
    confidence=$(echo "$instinct" | jq -r '.confidence // 0')
    status=$(echo "$instinct" | jq -r '.status // ""')
    action=$(echo "$instinct" | jq -r '.action // ""')
    # Promote validated instincts with high confidence (>= 0.7)
    if [[ "$status" == "validated" && $(echo "$confidence >= 0.7" | bc -l 2>/dev/null || echo 0) -eq 1 && -n "$action" ]]; then
      content=$(echo "$action" | cut -c1-200)
      result=$(bash .aether/aether-utils.sh queen-promote "pattern" "$content" "$colony_name" 2>/dev/null || echo '{"ok":false}')
      if [[ $(echo "$result" | jq -r '.ok // false') == "true" ]]; then
        promotion_count=$((promotion_count + 1))
      fi
    fi
  done

  # Log promotion results
  bash .aether/aether-utils.sh activity-log "MODIFIED" "Queen" "Promoted $promotion_count validated learnings to QUEEN.md from entombed colony"
fi
```

**Step 5.3: Display promotion summary**

```
---
Wisdom Promotion Summary
---
Colony: {colony_name}
Promoted: {promotion_count} validated patterns to QUEEN.md
---
```

### Step 6: Create Chamber

Generate chamber name:
```bash
sanitized_goal=$(echo "{goal}" | tr '[:upper:]' '[:lower:]' | tr -cs '[:alnum:]' '-' | sed 's/^-//;s/-$//' | cut -c1-50)
timestamp=$(date -u +%Y%m%d-%H%M%S)
chamber_name="${sanitized_goal}-${timestamp}"
```

Handle name collision: if directory exists, append counter:
```bash
counter=1
original_name="$chamber_name"
while [[ -d ".aether/chambers/$chamber_name" ]]; do
  chamber_name="${original_name}-${counter}"
  counter=$((counter + 1))
done
```

### Step 7: Create Chamber Using Utilities

Extract decisions and learnings as JSON arrays:
```bash
decisions_json=$(jq -c '.memory.decisions // []' .aether/data/COLONY_STATE.json)
learnings_json=$(jq -c '.memory.phase_learnings // []' .aether/data/COLONY_STATE.json)
phases_completed=$(jq '[.plan.phases[] | select(.status == "completed")] | length' .aether/data/COLONY_STATE.json)
total_phases=$(jq '.plan.phases | length' .aether/data/COLONY_STATE.json)
version=$(jq -r '.version // "3.0"' .aether/data/COLONY_STATE.json)
```

Create the chamber:
```bash
bash .aether/aether-utils.sh chamber-create \
  ".aether/chambers/{chamber_name}" \
  ".aether/data/COLONY_STATE.json" \
  "{goal}" \
  {phases_completed} \
  {total_phases} \
  "{milestone}" \
  "{version}" \
  '{decisions_json}' \
  '{learnings_json}'
```

### Step 8: Verify Chamber Integrity

Run verification:
```bash
bash .aether/aether-utils.sh chamber-verify ".aether/chambers/{chamber_name}"
```

If verification fails, display error and stop:
```
❌ Chamber verification failed.

Error: {verification_error}

The colony has NOT been reset. Please check the chamber directory:
.aether/chambers/{chamber_name}/
```
Stop here.

### Step 9: Reset Colony State

Backup current state:
```bash
cp .aether/data/COLONY_STATE.json .aether/data/COLONY_STATE.json.bak
```

Reset state while preserving memory (pheromones):
```bash
jq '
  .goal = null |
  .state = "IDLE" |
  .current_phase = 0 |
  .plan.phases = [] |
  .plan.generated_at = null |
  .plan.confidence = null |
  .build_started_at = null |
  .session_id = null |
  .initialized_at = null |
  .events = [] |
  .errors.records = [] |
  .errors.flagged_patterns = [] |
  .signals = [] |
  .graveyards = []
' .aether/data/COLONY_STATE.json.bak > .aether/data/COLONY_STATE.json
```

Verify reset succeeded:
```bash
new_goal=$(jq -r '.goal' .aether/data/COLONY_STATE.json)
if [[ "$new_goal" != "null" ]]; then
  # Restore from backup
  mv .aether/data/COLONY_STATE.json.bak .aether/data/COLONY_STATE.json
  echo "Error: State reset failed. Restored from backup."
  exit 1
fi
```

Remove backup after successful reset:
```bash
rm -f .aether/data/COLONY_STATE.json.bak
```

### Step 9.5: Write Final Handoff

After entombing the colony, write the final handoff documenting the archived colony:

```bash
cat > .aether/HANDOFF.md << 'HANDOFF_EOF'
# Colony Session — ENTOMBED

## ⚰️ Colony Archived
**Status:** Entombed in Chambers — Colony work preserved

## Chamber Location
.aether/chambers/{chamber_name}/

## Colony Summary
- Goal: "{goal}"
- Phases: {completed} completed of {total}
- Milestone: {milestone}
- Entombed At: {timestamp}

## Chamber Contents
- colony-state.json — Full colony state
- manifest.json — Archive metadata
- activity.log — Colony activity history
- spawn-tree.txt — Worker spawn records
- flags.json — Project flags (if existed)

## Session Note
This colony has been entombed and the active state reset.
The colony rests. Its learnings are preserved in the chamber.

To start anew: /ant:lay-eggs "<new goal>"
To explore chambers: /ant:tunnels
HANDOFF_EOF
```

This handoff serves as the record of the entombed colony.

### Step 10: Display Result

```
🏺 ═══════════════════════════════════════════════════
   C O L O N Y   E N T O M B E D
══════════════════════════════════════════════════ 🏺

✅ Colony archived successfully

👑 Goal: {goal}
📍 Phases: {completed} completed
🏆 Milestone: {milestone}

📦 Chamber: .aether/chambers/{chamber_name}/

🐜 The colony rests. Its learnings are preserved.

💾 State persisted — safe to /clear

🐜 What would you like to do next?
   1. /ant:lay-eggs "<new goal>"  — Start a new colony
   2. /ant:tunnels                — Browse archived colonies
   3. /clear                      — Clear context and continue

Use AskUserQuestion with these three options.

If option 1 selected: proceed to run /ant:lay-eggs flow
If option 2 selected: run /ant:tunnels
If option 3 selected: display "Run /ant:lay-eggs to begin anew after clearing"
```

### Edge Cases

**Chamber name collision:** Automatically append counter to make unique.

**Missing files during archive:** Note in output but continue with available files.

**State reset failure:** Restore from backup, display error, do not claim success.

**Empty phases array:** Can entomb a colony that was initialized but had no phases planned (treat as 0 of 0 completed).
