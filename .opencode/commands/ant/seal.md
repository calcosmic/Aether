---
name: ant:seal
description: "🏺🐜🏺 Seal the colony with Crowned Anthill milestone"
---

You are the **Queen**. Seal the colony when all work is complete.

## Instructions

### Step -1: Normalize Arguments

Run: `normalized_args=$(bash .aether/aether-utils.sh normalize-args "$@")`

This ensures arguments work correctly in both Claude Code and OpenCode. Use `$normalized_args` throughout this command.

Parse `$normalized_args`:
- If contains `--no-visual`: set `visual_mode = false` (visual is ON by default)
- Otherwise: set `visual_mode = true`

### Step 1: Read State

Read `.aether/data/COLONY_STATE.json`.

If file missing or `goal: null`:
```
No colony initialized. Run /ant:init first.
```
Stop here.

### Step 2: Validate Colony Is Complete

Extract: `goal`, `current_phase`, `plan.phases`, `milestone`, `state`.

**Precondition 1: All phases must be completed**

Check if all phases in `plan.phases` have `status: "completed"`:
```
all_completed = all(phase.status == "completed" for phase in plan.phases)
```

If NOT all completed:
```
Cannot archive colony with incomplete phases.

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

### Step 3: Check Milestone Eligibility

The full milestone progression is:
- **First Mound** — Phase 1 complete (first runnable)
- **Open Chambers** — Feature work underway (2+ phases complete)
- **Brood Stable** — Tests consistently green
- **Ventilated Nest** — Perf/latency acceptable (build + lint clean)
- **Sealed Chambers** — All phases complete (interfaces frozen)
- **Crowned Anthill** — Release-ready (user confirms via /ant:seal)

**If current milestone is "Crowned Anthill":**
```
Colony is already at Crowned Anthill milestone.
No further archiving needed.

Use /ant:status to view colony state.
```
Stop here.

**If current milestone is "Sealed Chambers":**
- Proceed to Step 4 (will upgrade to Crowned Anthill)

**If current milestone is "First Mound", "Open Chambers", "Brood Stable", "Ventilated Nest", or any intermediate milestone:**
- Since all phases are complete, the colony qualifies for both Sealed Chambers and Crowned Anthill
- The current logic allows proceeding to Step 4 (seal as Crowned Anthill)
- If user wants to explicitly achieve Sealed Chambers first, they can manually update milestone via COLONY_STATE.json

**If milestone is unrecognized (not in the 6 known stages):**
```
Unknown milestone: {milestone}

The milestone "{milestone}" is not recognized.
Known milestones: First Mound, Open Chambers, Brood Stable, Ventilated Nest, Sealed Chambers, Crowned Anthill

Run /ant:status to check colony state.
```
Stop here.

### Step 4: Archive Colony State

Create archive directory:
```
archive_dir=".aether/data/archive/session_$(date -u +%s)_archive"
mkdir -p "$archive_dir"
```

Copy the following files to the archive directory:
1. `.aether/data/COLONY_STATE.json` → `$archive_dir/COLONY_STATE.json`
2. `.aether/data/activity.log` → `$archive_dir/activity.log`
3. `.aether/data/spawn-tree.txt` → `$archive_dir/spawn-tree.txt`
4. `.aether/data/flags.json` → `$archive_dir/flags.json` (if exists)
5. `.aether/data/constraints.json` → `$archive_dir/constraints.json` (if exists)

Create archive manifest file `$archive_dir/manifest.json`:
```json
{
  "archived_at": "<ISO-8601 timestamp>",
  "goal": "<colony goal>",
  "total_phases": <number>,
  "milestone": "Crowned Anthill",
  "files": [
    "COLONY_STATE.json",
    "activity.log",
    "spawn-tree.txt",
    "flags.json",
    "constraints.json"
  ]
}
```

### Step 5: Update Milestone to Crowned Anthill

Update COLONY_STATE.json:
1. Set `milestone` to `"Crowned Anthill"`
2. Set `milestone_updated_at` to current ISO-8601 timestamp
3. Append event: `"<timestamp>|milestone_reached|archive|Achieved Crowned Anthill milestone - colony archived"`

### Step 5.1: Update Changelog

**MANDATORY: Record the seal in the project changelog. This step is never skipped.**

If no `CHANGELOG.md` exists, `changelog-append` creates one automatically.

Build a summary of what the colony accomplished across all phases:
- Collect completed phase names from COLONY_STATE.json
- Summarize the goal and key outcomes in one line

```bash
bash .aether/aether-utils.sh changelog-append \
  "$(date +%Y-%m-%d)" \
  "seal-crowned-anthill" \
  "00" \
  "{key_files_csv}" \
  "Colony sealed at Crowned Anthill;{goal}" \
  "{phases_completed} phases completed;Colony wisdom promoted to QUEEN.md" \
  ""
```

- `{key_files_csv}` — list the most significant files created or modified across the colony's lifetime (derive from phase plans or git log)
- `{goal}` — the colony goal from COLONY_STATE.json

**Error handling:** If `changelog-append` fails, log to midden and continue — changelog failure never blocks sealing.

### Step 5.2: Update Registry (Silent)

Mark the colony as inactive in the global registry. This is silent on failure — registry is not required for the colony to work.

Run using the Bash tool (ignore errors):
```bash
bash .aether/aether-utils.sh registry-add "$(pwd)" "$(jq -r '.version // "unknown"' ~/.aether/version.json 2>/dev/null || echo 'unknown')" --active false 2>/dev/null || true
```

If the command fails, proceed silently. This is optional bookkeeping.

### Step 5.25: Hive Promotion (NON-BLOCKING)

After wisdom promotion, promote abstracted instincts to the cross-colony hive.

**Extract high-confidence instincts for hive promotion:**
```bash
# Get instincts with confidence >= 0.8
high_conf_instincts=$(jq -r '.instincts[] | select(.confidence >= 0.8) | @base64' .aether/data/COLONY_STATE.json 2>/dev/null || echo "")

# Derive source repo name from current directory
source_repo="$(pwd)"

hive_promoted_count=0
hive_errors=0
for encoded in $high_conf_instincts; do
  [[ -z "$encoded" ]] && continue

  # Extract trigger and action fields from the instinct object
  trigger=$(echo "$encoded" | base64 -d | jq -r '.trigger // empty')
  action=$(echo "$encoded" | base64 -d | jq -r '.action // empty')
  confidence=$(echo "$encoded" | base64 -d | jq -r '.confidence // 0.7')
  domain=$(echo "$encoded" | base64 -d | jq -r '.domain // empty')

  [[ -z "$trigger" || -z "$action" ]] && continue

  # Build the promotion text in "When {trigger}: {action}" format
  promote_text="When ${trigger}: ${action}"

  # Build hive-promote args with --text and --source-repo (required)
  promote_args=(hive-promote --text "$promote_text" --source-repo "$source_repo" --confidence "$confidence")
  [[ -n "$domain" ]] && promote_args+=(--domain "$domain")

  # Call hive-promote which orchestrates abstract + store
  result=$(bash .aether/aether-utils.sh "${promote_args[@]}" 2>/dev/null || echo '{}')
  was_promoted=$(echo "$result" | jq -r '.result.action // "skipped"' 2>/dev/null || echo "skipped")

  if [[ "$was_promoted" == "promoted" || "$was_promoted" == "merged" ]]; then
    hive_promoted_count=$((hive_promoted_count + 1))
  fi
done

if [[ "$hive_promoted_count" -gt 0 ]]; then
  echo "Hive promotion: $hive_promoted_count instinct(s) promoted to cross-colony hive"
fi
```

**Continue to Step 5.5 (non-blocking):**
Proceed to Step 5.5 regardless of hive promotion results — hive promotion is strictly non-blocking.

### Step 5.5: Write Final Handoff

After archiving, write the final handoff documenting the completed colony:

Resolve the handoff template path:
  Check ~/.aether/system/templates/handoff.template.md first,
  then .aether/templates/handoff.template.md.

If no template found: output "Template missing: handoff.template.md. Run aether update to fix." and stop.

Read the template file. Fill all {{PLACEHOLDER}} values:
  - {{CHAMBER_NAME}} → archive directory name
  - {{GOAL}} → goal
  - {{PHASES_COMPLETED}} → total_phases (OpenCode seal archives completed colonies)
  - {{TOTAL_PHASES}} → total_phases
  - {{MILESTONE}} → "Crowned Anthill"
  - {{ENTOMB_TIMESTAMP}} → seal timestamp

Remove the HTML comment lines at the top of the template.
Write the result to .aether/HANDOFF.md using the Write tool.

This handoff serves as the final record of the completed colony.

### Step 5.75: Export XML Archive (best-effort)

Export colony data as a combined XML archive and a standalone pheromones.xml. Both are best-effort — seal proceeds even if XML export fails.

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

  # Also export standalone pheromones.xml for cross-colony sharing
  pher_result=$(bash .aether/aether-utils.sh pheromone-export-xml ".aether/exchange/pheromones.xml" 2>&1)
  pher_ok=$(echo "$pher_result" | jq -r '.ok // false' 2>/dev/null)
  if [[ "$pher_ok" == "true" ]]; then
    pher_signal_count=$(jq '[.signals[] | select(.active != false)] | length' .aether/data/pheromones.json 2>/dev/null || echo "0")
    pher_export_line="Signal Export: pheromones.xml (${pher_signal_count} signals, importable by other colonies)"
  else
    pher_export_line="Signal Export: failed (non-blocking)"
  fi
else
  xml_export_line="XML Archive: skipped (xmllint not available)"
  pher_export_line="Signal Export: skipped (xmllint not available)"
fi
```

### Step 6: Display Result

Output:
```
🏺 ════════════════════════════════════════════════════
   C R O W N E D   A N T H I L L
══════════════════════════════════════════════════ 🏺

✅ Colony archived successfully!

👑 Goal: {goal (truncated to 60 chars)}
📍 Phases: {total_phases} completed
🏆 Milestone: Crowned Anthill

📦 Archive Location: {archive_dir}
   - COLONY_STATE.json
   - activity.log
   - spawn-tree.txt
   - flags.json (if existed)
   - constraints.json (if existed)
{xml_export_line}
{pher_export_line}

🐜 The colony has reached its final form.
   The anthill stands crowned and sealed.
   History is preserved. The colony rests.

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

**If milestone is already "Sealed Chambers" but phases are complete:**
- Proceed with archiving and upgrade to Crowned Anthill

**If any archive files are missing:**
- Archive what exists, note in manifest which files were missing

**If archive directory already exists:**
- Append timestamp to make unique: `session_<ts>_archive_<random>`
