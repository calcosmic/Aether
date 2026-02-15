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

### Step 5.5: Write Final Handoff

After archiving, write the final handoff documenting the completed colony:

```bash
cat > .aether/HANDOFF.md << 'HANDOFF_EOF'
# Colony Session — SEALED (Crowned Anthill)

## 🏆 Colony Complete
**Status:** Crowned Anthill — All phases completed and archived

## Archive Location
{archive_dir}

## Colony Summary
- Goal: "{goal}"
- Total Phases: {total_phases}
- Milestone: Crowned Anthill
- Sealed At: {timestamp}

## Files Archived
- COLONY_STATE.json
- activity.log
- spawn-tree.txt
- flags.json (if existed)
- constraints.json (if existed)

## Session Note
This colony has been sealed and archived. The anthill stands crowned.
To start anew, run: /ant:lay-eggs "<new goal>"
HANDOFF_EOF
```

This handoff serves as the final record of the completed colony.

### Step 6: Display Result

**If visual_mode is true, render final swarm display:**
```bash
bash .aether/aether-utils.sh swarm-display-update "Queen" "prime" "completed" "Colony sealed" "Colony" '{"read":3,"grep":0,"edit":2,"bash":3}' 100 "fungus_garden" 100
bash .aether/aether-utils.sh swarm-display-render "$seal_id"
```

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
