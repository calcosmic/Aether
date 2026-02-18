---
name: ant:seal
description: "🏺🐜🏺 Seal the colony with Crowned Anthill milestone"
---

You are the **Queen**. Seal the colony with a ceremony — no archiving.

## Instructions

### Step -1: Normalize Arguments

Run: `normalized_args=$(bash .aether/aether-utils.sh normalize-args "$@")`

This ensures arguments work correctly in both Claude Code and OpenCode. Use `$normalized_args` throughout this command.

Parse `$normalized_args`:
- If contains `--no-visual`: set `visual_mode = false` (visual is ON by default)
- Otherwise: set `visual_mode = true`

### Step 0: Initialize Visual Mode (if enabled)

If `visual_mode` is true:
```bash
# Generate session ID
seal_id="seal-$(date +%s)"

# Initialize swarm display
bash .aether/aether-utils.sh swarm-display-init "$seal_id"
bash .aether/aether-utils.sh swarm-display-update "Queen" "prime" "excavating" "Sealing colony" "Colony" '{"read":0,"grep":0,"edit":0,"bash":0}' 0 "fungus_garden" 0
```

### Step 1: Read State

Read `.aether/data/COLONY_STATE.json`.

If file missing or `goal: null`:
```
No colony initialized. Run /ant:init first.
```
Stop here.

Extract: `goal`, `state`, `current_phase`, `plan.phases`, `milestone`, `version`, `initialized_at`.

### Step 2: Maturity Gate

Run `bash .aether/aether-utils.sh milestone-detect` to get `milestone`, `phases_completed`, `total_phases`.

**If milestone is already "Crowned Anthill":**
```
Colony already sealed at Crowned Anthill.
Run /ant:entomb to archive this colony to chambers.
```
Stop here.

**If state is "EXECUTING":**
```
Colony is still executing. Run /ant:continue first.
```
Stop here.

**If all phases complete** (phases_completed == total_phases, or milestone is "Sealed Chambers"):
- Set `incomplete_warning = ""` (no warning needed)
- Proceed to Step 3.

**If phases are incomplete** (any other milestone — First Mound, Open Chambers, Brood Stable, Ventilated Nest, etc.):
- Set `incomplete_warning = "WARNING: {phases_completed} of {total_phases} phases complete. Sealing now will mark incomplete work as the final state."`
- Proceed to Step 3 (warn but DO NOT block).

### Step 3: Confirmation

Display what will be sealed:
```
SEAL COLONY

Goal: {goal}
Phases: {phases_completed} of {total_phases} completed
Current Milestone: {milestone}

{If incomplete_warning is not empty, display it here}

This will:
  - Award the Crowned Anthill milestone
  - Write CROWNED-ANTHILL.md ceremony record
  - Promote colony wisdom to QUEEN.md

Seal this colony? (yes/no)
```

Use `AskUserQuestion with yes/no options`.

If not "yes":
```
Sealing cancelled. Colony remains active.
```
Stop here.

### Step 4: Promote Colony Wisdom to QUEEN.md

Extract and promote significant patterns, decisions, and instincts from the colony:

```bash
# Ensure QUEEN.md exists
if [[ ! -f ".aether/docs/QUEEN.md" ]]; then
  bash .aether/aether-utils.sh queen-init >/dev/null 2>&1
fi

# Extract colony name from session_id or goal
colony_name=$(jq -r '.session_id // empty' .aether/data/COLONY_STATE.json | sed 's/^session_//' | cut -d'_' -f1-3)
[[ -z "$colony_name" ]] && colony_name=$(jq -r '.goal' .aether/data/COLONY_STATE.json | tr '[:upper:]' '[:lower:]' | tr ' ' '-' | cut -c1-30)

# Track promotion results
promotions_made=0
promotion_details=""

# Extract and promote phase learnings (validated learnings)
while IFS= read -r learning; do
  claim=$(echo "$learning" | jq -r '.claim // empty')
  status=$(echo "$learning" | jq -r '.status // empty')

  if [[ -n "$claim" && "$status" == "validated" ]]; then
    # Determine type based on content patterns
    if echo "$claim" | grep -qi "never\|avoid\|don't\|do not"; then
      type="redirect"
    elif echo "$claim" | grep -qi "always\|should\|must\|pattern\|approach"; then
      type="pattern"
    elif echo "$claim" | grep -qi "use\|prefer\|technology\|tool\|library"; then
      type="stack"
    else
      type="philosophy"
    fi

    result=$(bash .aether/aether-utils.sh queen-promote "$type" "$claim" "$colony_name" 2>/dev/null)
    if echo "$result" | jq -e '.ok' >/dev/null 2>&1; then
      promotions_made=$((promotions_made + 1))
      promotion_details="${promotion_details}  - Promoted ${type}: ${claim:0:60}...\n"
    fi
  fi
done < <(jq -c '.memory.phase_learnings[]?.learnings[]? // empty' .aether/data/COLONY_STATE.json 2>/dev/null)

# Extract and promote decisions
while IFS= read -r decision; do
  description=$(echo "$decision" | jq -r '.description // .rationale // empty')
  [[ -z "$description" ]] && description=$(echo "$decision" | jq -r '.decision // empty')

  if [[ -n "$description" ]]; then
    result=$(bash .aether/aether-utils.sh queen-promote "pattern" "$description" "$colony_name" 2>/dev/null)
    if echo "$result" | jq -e '.ok' >/dev/null 2>&1; then
      promotions_made=$((promotions_made + 1))
      promotion_details="${promotion_details}  - Promoted pattern from decision: ${description:0:60}...\n"
    fi
  fi
done < <(jq -c '.memory.decisions[]? // empty' .aether/data/COLONY_STATE.json 2>/dev/null)

# Promote high-confidence instincts
instinct_result=$(bash .aether/aether-utils.sh instinct-read --min-confidence 0.7 2>/dev/null || echo '{"ok":false}')
if echo "$instinct_result" | jq -e '.ok' >/dev/null 2>&1; then
  while IFS= read -r instinct_action; do
    if [[ -n "$instinct_action" && "$instinct_action" != "null" ]]; then
      result=$(bash .aether/aether-utils.sh queen-promote "pattern" "$instinct_action" "$colony_name" 2>/dev/null)
      if echo "$result" | jq -e '.ok' >/dev/null 2>&1; then
        promotions_made=$((promotions_made + 1))
      fi
    fi
  done < <(echo "$instinct_result" | jq -r '.result[]?.action // empty' 2>/dev/null)
fi

# Log promotion results to activity log
bash .aether/aether-utils.sh activity-log "MODIFIED" "Queen" "Promoted ${promotions_made} learnings/decisions/instincts to QUEEN.md from colony ${colony_name}"

# Store promotion summary for display
promotion_summary="${promotions_made} wisdom entries promoted"
```

### Step 5: Update Milestone to Crowned Anthill

Update COLONY_STATE.json:
1. Set `milestone` to `"Crowned Anthill"`
2. Set `milestone_updated_at` to current ISO-8601 timestamp
3. Append event: `"<timestamp>|milestone_reached|seal|Achieved Crowned Anthill milestone"`

Run `bash .aether/aether-utils.sh validate-state colony` after write.

### Step 6: Write CROWNED-ANTHILL.md

Calculate colony age:
```bash
initialized_at=$(jq -r '.initialized_at // empty' .aether/data/COLONY_STATE.json)
if [[ -n "$initialized_at" ]]; then
  init_epoch=$(date -j -f "%Y-%m-%dT%H:%M:%SZ" "$initialized_at" +%s 2>/dev/null || echo 0)
  now_epoch=$(date +%s)
  if [[ "$init_epoch" -gt 0 ]]; then
    colony_age_days=$(( (now_epoch - init_epoch) / 86400 ))
  else
    colony_age_days=0
  fi
else
  colony_age_days=0
fi
```

Extract phase recap:
```bash
phase_recap=""
while IFS= read -r phase_line; do
  phase_name=$(echo "$phase_line" | jq -r '.name')
  phase_status=$(echo "$phase_line" | jq -r '.status')
  phase_recap="${phase_recap}  - ${phase_name}: ${phase_status}\n"
done < <(jq -c '.plan.phases[]' .aether/data/COLONY_STATE.json 2>/dev/null)
```

Write the seal document:
```bash
version=$(jq -r '.version // "3.0"' .aether/data/COLONY_STATE.json)
seal_date=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

cat > .aether/CROWNED-ANTHILL.md << SEAL_EOF
# Crowned Anthill — ${goal}

**Sealed:** ${seal_date}
**Milestone:** Crowned Anthill
**Version:** ${version}

## Colony Stats
- Total Phases: ${total_phases}
- Phases Completed: ${phases_completed} of ${total_phases}
- Colony Age: ${colony_age_days} days
- Wisdom Promoted: ${promotions_made} entries

## Phase Recap
$(echo -e "$phase_recap")

## Pheromone Legacy
- Instincts and validated learnings promoted to QUEEN.md
- ${promotions_made} total entries promoted

## The Work
${goal}
SEAL_EOF
```

### Step 6.5: Export XML Archive (best-effort)

Export colony data as a combined XML archive. This is best-effort — seal proceeds even if XML export fails.

```bash
# Check if xmllint is available
if command -v xmllint >/dev/null 2>&1; then
  xml_result=$(bash .aether/aether-utils.sh colony-archive-xml ".aether/exchange/colony-archive.xml" 2>&1)
  xml_ok=$(echo "$xml_result" | jq -r '.ok // false' 2>/dev/null)
  if [[ "$xml_ok" == "true" ]]; then
    xml_pheromone_count=$(echo "$xml_result" | jq -r '.result.pheromone_count // 0' 2>/dev/null)
    xml_export_line="XML Archive: colony-archive.xml (${xml_pheromone_count} active signals)"
  else
    xml_export_line="XML Archive: export failed (non-blocking)"
  fi
else
  xml_export_line="XML Archive: skipped (xmllint not available)"
fi
```

### Step 7: Display Ceremony

**If visual_mode is true, render swarm display BEFORE the ASCII art:**
```bash
bash .aether/aether-utils.sh swarm-display-update "Queen" "prime" "completed" "Colony sealed" "Colony" '{"read":3,"grep":0,"edit":2,"bash":3}' 100 "fungus_garden" 100
bash .aether/aether-utils.sh swarm-display-render "$seal_id"
```

Display the ASCII art ceremony:
```
        .     .
       /|\   /|\
      / | \ / | \
     /  |  X  |  \
    /   | / \ |   \
   /    |/   \|    \
  /     /     \     \
 /____ /  ___  \ ____\
      / /   \ \
     / /     \ \
    /_/       \_\
     |  CROWNED |
     | ANTHILL  |
     |__________|
```

Below the ASCII art, display:
```
C R O W N E D   A N T H I L L

Goal: {goal}
Phases: {phases_completed} of {total_phases} completed
{If incomplete_warning is not empty: display it}
Wisdom Promoted: {promotion_summary}

Seal Document: .aether/CROWNED-ANTHILL.md
{xml_export_line}

The colony stands crowned and sealed.
Its wisdom lives on in QUEEN.md.
The anthill has reached its final form.

Run /ant:entomb to archive this colony to chambers.
```

### Edge Cases

**Colony already at Crowned Anthill:**
- Display message and guide to /ant:entomb. Do NOT re-seal.

**Phases incomplete:**
- Warn but allow. The seal proceeds after confirmation.

**Missing QUEEN.md:**
- queen-init creates it. If that fails, skip promotion (non-fatal).

**Missing initialized_at:**
- Colony age defaults to 0 days.

**Empty phases array:**
- Can seal a colony with 0 phases (rare but valid). phases_completed = 0, total_phases = 0.
