---
name: ant:entomb
description: "⚰️🐜⚰️ Entomb completed colony in chambers"
---

You are the **Queen**. Archive the sealed colony to chambers.

## Instructions

Parse `$ARGUMENTS`:
- If contains `--no-visual`: set `visual_mode = false` (visual is ON by default)
- Otherwise: set `visual_mode = true`

<failure_modes>
### Archive Write Failure
If writing to chambers/ archive fails partway:
- Do NOT delete source data (the active colony) until archive is verified
- Report what was archived vs. what failed
- Recovery: the original colony data is intact -- user can retry

### Seal Verification Failure
If the colony was not properly sealed before entomb:
- STOP -- do not archive an incomplete colony
- Direct user to run /ant:seal first
- This is a hard gate, not a suggestion

### Chamber Naming Conflict
If the target chamber directory already exists:
- STOP -- do not overwrite an existing archive
- Report the conflict
- Options: (1) Use a different name, (2) Merge with existing, (3) Cancel
</failure_modes>

<success_criteria>
Command is complete when:
- Colony data is fully copied to chambers/ archive directory
- Archive integrity is verified (all expected files present)
- Active colony state is cleared only after archive verification
- User sees confirmation with archive location
</success_criteria>

<read_only>
Do not touch during entomb:
- .aether/dreams/ (user notes -- not archived)
- Source code files
- .env* files
- .claude/settings.json
- .github/workflows/
- Other chamber directories (only write to the new chamber)
</read_only>

### Step 0: Initialize Visual Mode (if enabled)

If `visual_mode` is true:
```bash
# Generate session ID
entomb_id="entomb-$(date +%s)"

# Initialize swarm display (consolidated)
bash .aether/aether-utils.sh swarm-display-init "$entomb_id" && bash .aether/aether-utils.sh swarm-display-update "Queen" "prime" "excavating" "Entombing colony" "Colony" '{"read":0,"grep":0,"edit":0,"bash":0}' 0 "fungus_garden" 0
```

### Step 1: Read State

Read `.aether/data/COLONY_STATE.json`.

If file missing or `goal: null`:
```
No colony to entomb. Run /ant:init first.
```
Stop here.

Extract: `goal`, `state`, `current_phase`, `plan.phases`, `milestone`, `version`, `initialized_at`, `memory.decisions`, `memory.phase_learnings`, `memory.instincts`.

### Step 2: Seal-First Enforcement

Check `milestone` in COLONY_STATE.json.

**If milestone != "Crowned Anthill":**
```
Colony has not been sealed.

Current milestone: {milestone}
Required: Crowned Anthill

Run /ant:seal first to complete the sealing ceremony.
```
Stop here.

**Belt-and-suspenders:** Also check `.aether/CROWNED-ANTHILL.md` exists. If milestone is Crowned Anthill but file is missing:
```
CROWNED-ANTHILL.md not found — seal may have been interrupted. Run /ant:seal again.
```
Stop here.

### Step 3: User Confirmation

Show what will be archived:
```
ENTOMB COLONY

Goal: {goal}
Milestone: Crowned Anthill
Phases: {phases_completed} of {total_phases}

This will:
  - Archive ALL colony data to .aether/chambers/
  - Copy CROWNED-ANTHILL.md, pheromones, dreams, session data
  - Promote colony wisdom to QUEEN.md (cross-generational learning)
  - Reset COLONY_STATE.json for a fresh start
  - Clear session data

This action is permanent. The archived colony can be browsed via /ant:tunnels.

Entomb this colony? (yes/no)
```

Use `AskUserQuestion with yes/no options`.

If not "yes":
```
Entombment cancelled. Colony remains active.
```
Stop here.

### Step 3.25: Wisdom Approval

Before archiving, review wisdom proposals accumulated during this colony's lifecycle.

```bash
# Check for pending proposals
proposals=$(bash .aether/aether-utils.sh learning-check-promotion 2>/dev/null || echo '{"proposals":[]}')
proposal_count=$(echo "$proposals" | jq '.proposals | length')

if [[ "$proposal_count" -gt 0 ]]; then
  echo ""
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo "   🧠 FINAL WISDOM REVIEW"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo ""
  echo "Review wisdom proposals before archiving this colony."
  echo "Approved proposals will be promoted to QUEEN.md."
  echo ""

  # Run approval workflow (blocking)
  bash .aether/aether-utils.sh learning-approve-proposals

  echo ""
  echo "Wisdom review complete. Proceeding with entombment..."
  echo ""
else
  echo "No wisdom proposals to review."
fi
```

### Step 3.5: Check XML Tools

XML archiving is required for entombment. Check tool availability before proceeding.
Uses `command -v xmllint` directly — consistent with seal.md's tool check.

```bash
if command -v xmllint >/dev/null 2>&1; then
  xmllint_available=true
else
  xmllint_available=false
fi
```

**If xmllint is NOT available:**

Ask the user:
```
xmllint is not installed — XML archiving requires it.

Install now?
  - macOS: xcode-select --install (or brew install libxml2)
  - Linux: apt-get install libxml2-utils

Install xmllint? (yes/no)
```

Use AskUserQuestion with yes/no options.

If yes:
- On macOS: Run `xcode-select --install` or `brew install libxml2`
- On Linux: Run `sudo apt-get install -y libxml2-utils`
- After install attempt, re-check: `command -v xmllint >/dev/null 2>&1`
- If still not available after install:
  ```
  xmllint installation failed. Cannot entomb without XML archiving.
  Install xmllint manually and try again.
  ```
  Stop here.

If no:
```
Entombment requires XML archiving. Install xmllint and try again.
```
Stop here.

### Step 4: Promote Wisdom to QUEEN.md

Before creating the chamber, promote validated learnings to QUEEN.md for future colonies.

**Step 4.1: Ensure QUEEN.md exists**

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

**Step 4.2: Extract and promote validated learnings**

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

**Step 4.3: Display promotion summary**

```
---
Wisdom Promotion Summary
---
Colony: {colony_name}
Promoted: {promotion_count} validated patterns to QUEEN.md
---
```

### Step 5: Generate Chamber Name

Date-first naming:
```bash
date_prefix=$(date -u +%Y-%m-%d)
sanitized_goal=$(echo "$goal" | tr '[:upper:]' '[:lower:]' | tr -cs '[:alnum:]' '-' | sed 's/^-//;s/-$//' | cut -c1-40)
chamber_name="${date_prefix}-${sanitized_goal}"
```

Collision handling:
```bash
counter=1
original_name="$chamber_name"
while [[ -d ".aether/chambers/$chamber_name" ]]; do
  chamber_name="${original_name}-${counter}"
  counter=$((counter + 1))
done
```

### Step 6: Create Chamber

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
  ".aether/chambers/$chamber_name" \
  ".aether/data/COLONY_STATE.json" \
  "$goal" \
  "$phases_completed" \
  "$total_phases" \
  "$milestone" \
  "$version" \
  "$decisions_json" \
  "$learnings_json"
```

### Step 7: Archive Additional Files

AFTER chamber-create succeeds, copy additional data files:
```bash
chamber_dir=".aether/chambers/$chamber_name"

# Archive data files (if they exist)
for f in pheromones.json session.json activity.log flags.json constraints.json spawn-tree.txt timing.log view-state.json; do
  [[ -f ".aether/data/$f" ]] && cp ".aether/data/$f" "$chamber_dir/" 2>/dev/null || true
done

# Archive seal document (critical — this is the ceremony record)
[[ -f ".aether/CROWNED-ANTHILL.md" ]] && cp ".aether/CROWNED-ANTHILL.md" "$chamber_dir/"

# Archive HANDOFF.md if it exists
[[ -f ".aether/HANDOFF.md" ]] && cp ".aether/HANDOFF.md" "$chamber_dir/"

# Archive dreams directory (optional — may not exist)
[[ -d ".aether/dreams" ]] && cp -r ".aether/dreams" "$chamber_dir/dreams" 2>/dev/null || true
```

Do NOT copy: `.aether/data/backups/`, `.aether/data/locks/`, `.aether/data/midden/`, `.aether/data/survey/`.

### Step 7.5: Export XML Archive (hard-stop)

Export combined XML archive to the chamber. This is a HARD REQUIREMENT — entomb fails if XML export fails.

```bash
chamber_dir=".aether/chambers/$chamber_name"
xml_result=$(bash .aether/aether-utils.sh colony-archive-xml "$chamber_dir/colony-archive.xml" 2>&1)
xml_ok=$(echo "$xml_result" | jq -r '.ok // false' 2>/dev/null)

if [[ "$xml_ok" != "true" ]]; then
  # HARD STOP — remove the chamber and abort
  rm -rf "$chamber_dir"
  echo "XML archive export failed. Colony NOT entombed."
  echo ""
  echo "Error: $(echo "$xml_result" | jq -r '.error // "Unknown error"' 2>/dev/null)"
  echo ""
  echo "The chamber has been cleaned up. Fix the XML issue and try again."
  # Do NOT proceed to state reset or any further steps
fi
```

If xml_ok is true, store for display:
```bash
xml_pheromone_count=$(echo "$xml_result" | jq -r '.result.pheromone_count // 0' 2>/dev/null)
xml_archive_line="XML Archive: colony-archive.xml (${xml_pheromone_count} active signals)"
```

**Critical behavior:** If XML export fails, entomb STOPS. The chamber directory is removed (cleanup). The colony state is NOT reset. The user sees a clear error and can retry after fixing the issue.

### Step 8: Verify Chamber Integrity

Run verification:
```bash
bash .aether/aether-utils.sh chamber-verify ".aether/chambers/$chamber_name"
```

If verification fails, display error and stop:
```
Chamber verification failed.

Error: {verification_error}

The colony has NOT been reset. Please check the chamber directory:
.aether/chambers/{chamber_name}/
```
Stop here.

### Step 9: Record in Eternal Memory

Write colony summary to eternal memory:
```bash
bash .aether/aether-utils.sh eternal-init  # idempotent
eternal_file="$HOME/.aether/eternal/memory.json"
if [[ -f "$eternal_file" ]]; then
  colony_entry=$(jq -n \
    --arg goal "$goal" \
    --arg milestone "$milestone" \
    --arg sealed_at "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    --arg chamber ".aether/chambers/$chamber_name" \
    '{goal: $goal, milestone: $milestone, sealed_at: $sealed_at, chamber: $chamber}')
  jq --argjson entry "$colony_entry" '.colonies += [$entry]' "$eternal_file" > /tmp/eternal-tmp.json \
    && mv /tmp/eternal-tmp.json "$eternal_file"
fi
```

### Step 10: Reset Colony State

Backup current state:
```bash
cp .aether/data/COLONY_STATE.json .aether/data/COLONY_STATE.json.bak
```

Reset state, clearing everything including promoted wisdom (already in QUEEN.md):
```bash
# Resolve jq template path (hub-first)
jq_template=""
for path in \
  "$HOME/.aether/system/templates/colony-state-reset.jq.template" \
  ".aether/templates/colony-state-reset.jq.template"; do
  if [[ -f "$path" ]]; then
    jq_template="$path"
    break
  fi
done

if [[ -z "$jq_template" ]]; then
  echo "Template missing: colony-state-reset.jq.template. Run aether update to fix."
  exit 1
fi

jq -f "$jq_template" .aether/data/COLONY_STATE.json.bak > .aether/data/COLONY_STATE.json
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

Clear session data:
```bash
rm -f .aether/data/session.json
```

Clean up seal document (it's now in the chamber):
```bash
rm -f .aether/CROWNED-ANTHILL.md
```

### Step 11: Write HANDOFF.md

Write handoff documenting the entombed colony:

Resolve the handoff template path:
  Check ~/.aether/system/templates/handoff.template.md first,
  then .aether/templates/handoff.template.md.

If no template found: output "Template missing: handoff.template.md. Run aether update to fix." and stop.

Read the template file. Fill all {{PLACEHOLDER}} values:
  - {{CHAMBER_NAME}} → chamber_name
  - {{GOAL}} → goal
  - {{PHASES_COMPLETED}} → phases completed count
  - {{TOTAL_PHASES}} → total phases count
  - {{MILESTONE}} → milestone
  - {{ENTOMB_TIMESTAMP}} → current ISO-8601 UTC timestamp

Remove the HTML comment lines at the top of the template.
Write the result to .aether/HANDOFF.md using the Write tool.

### Step 12: Display Result

**If visual_mode is true, render swarm display (consolidated):**
```bash
bash .aether/aether-utils.sh swarm-display-update "Queen" "prime" "completed" "Colony entombed" "Colony" '{"read":3,"grep":0,"edit":2,"bash":5}' 100 "fungus_garden" 100 && bash .aether/aether-utils.sh swarm-display-inline "$entomb_id"
```

Display:
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   C O L O N Y   E N T O M B E D
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Colony archived successfully

Goal: {goal}
Phases: {completed} completed
Milestone: {milestone}

Chamber: .aether/chambers/{chamber_name}/
{xml_archive_line}

The colony rests. Its learnings live on in QUEEN.md.

──────────────────────────────────────────────────
🐜 Next Up
──────────────────────────────────────────────────
   /ant:lay-eggs            🥚 Start a new colony
   /ant:tunnels             🗄️  Browse archived chambers
```

What would you like to do next?
  1. /ant:lay-eggs "<new goal>"  — Start a new colony
  2. /ant:tunnels                — Browse archived colonies
  3. /clear                      — Clear context and continue

Use AskUserQuestion with these three options.

If option 1 selected: proceed to run /ant:lay-eggs flow
If option 2 selected: run /ant:tunnels
If option 3 selected: display "Run /ant:lay-eggs to begin anew after clearing"
```

### Edge Cases

**Colony not sealed:**
- Refuse with guidance to run /ant:seal first. This is the primary gate.

**Chamber name collision:**
- Automatically append counter to make unique.

**Missing files during archive:**
- Note in output but continue with available files. The `|| true` in the copy loop handles this.

**State reset failure:**
- Restore from backup, display error, do not claim success.

**Empty phases array:**
- Can entomb a colony that was initialized but had no phases planned (treat as 0 of 0 completed).

**Missing CROWNED-ANTHILL.md:**
- Refuse with guidance to run /ant:seal again.
