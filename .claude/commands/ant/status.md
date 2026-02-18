---
name: ant:status
description: "📈🐜🏘️🐜📈 Show colony status at a glance"
---

You are the **Queen**. Show colony status.

## Instructions

### Step 0: Version Check (Non-blocking)

Run using the Bash tool with description "Checking colony version...": `bash .aether/aether-utils.sh version-check-cached 2>/dev/null || true`

If the command succeeds and the JSON result contains a non-empty string, display it as a one-line notice. Proceed regardless of outcome.

### Step 1: Read State + Version Check

Read `.aether/data/COLONY_STATE.json`.

If file missing or `goal: null`:
```
No colony initialized. Run /ant:init first.
```
Stop here.

**Auto-upgrade old state:**
If `version` field is missing, "1.0", or "2.0":
1. Preserve: `goal`, `state`, `current_phase`, `plan.phases` (keep phase structure)
2. Write upgraded state:
```json
{
  "version": "3.0",
  "goal": "<preserved>",
  "state": "<preserved or 'READY'>",
  "current_phase": <preserved or 0>,
  "session_id": "migrated_<timestamp>",
  "initialized_at": "<preserved or now>",
  "build_started_at": null,
  "plan": {
    "generated_at": "<preserved or null>",
    "confidence": null,
    "phases": <preserved or []>
  },
  "memory": { "phase_learnings": [], "decisions": [], "instincts": [] },
  "errors": { "records": [], "flagged_patterns": [] },
  "events": ["<now>|state_upgraded|system|Auto-upgraded from v<old> to v3.0"]
}
```
3. Output: `State auto-upgraded to v3.0`
4. Continue with command.

### Step 1.5: Load State and Show Resumption Context

Run using the Bash tool with description "Loading colony state...": `bash .aether/aether-utils.sh load-state`

If successful and goal is not null:
1. Extract current_phase from state
2. Get phase name from plan.phases[current_phase - 1].name (or "(unnamed)")
3. Get last event timestamp from events array (last element)
4. Display extended resumption context:
   ```
   🔄 Resuming: Phase X - Name
      Last activity: timestamp
   ```

5. Check for .aether/HANDOFF.md existence in the load-state output or via separate check
6. If .aether/HANDOFF.md exists:
   - Display: "Resuming from paused session"
   - Read .aether/HANDOFF.md content for additional context
   - Remove .aether/HANDOFF.md after displaying (cleanup)

Run using the Bash tool with description "Releasing colony lock...": `bash .aether/aether-utils.sh unload-state` to release lock.

### Step 2: Compute Summary

From state, extract:

### Step 2.5: Gather Dream Information

Run using the Bash tool with description "Counting dream entries...": `ls -1 .aether/dreams/*.md 2>/dev/null | wc -l`

Capture:
- Dream count: number of .md files in .aether/dreams/
- Latest dream: most recent file by name (files are timestamped: YYYY-MM-DD-HHMM.md)

To get latest dream timestamp, run using the Bash tool with description "Finding latest dream...":
```bash
ls -1 .aether/dreams/*.md 2>/dev/null | sort | tail -1 | sed 's/.*\/\([0-9]\{4\}-[0-9]\{2\}-[0-9]\{2\}\)-\([0-9]\{4\}\).*/\1 \2/'
```

Format the timestamp as: YYYY-MM-DD HH:MM

From state, extract:

**Phase info:**
- Current phase number: `current_phase`
- Total phases: `plan.phases.length`
- Phase name: `plan.phases[current_phase - 1].name` (if exists)

**Task progress:**
- If phases exist, count tasks in current phase
- Completed: tasks with `status: "completed"`
- Total: all tasks in current phase

**Constraints:**
Read `.aether/data/constraints.json` if exists:
- Focus count: `focus.length`
- Constraints count: `constraints.length`

**Flags:**
Run using the Bash tool with description "Checking for blockers...": `bash .aether/aether-utils.sh flag-check-blockers`
Extract:
- Blockers count (critical, block advancement)
- Issues count (high, warnings)
- Notes count (low, informational)

**Instincts:**
From `memory.instincts`:
- Total count: `instincts.length`
- High confidence (≥0.7): count where confidence >= 0.7
- Top 3: sorted by confidence descending

**Colony state:**
- `state` field (IDLE, READY, EXECUTING, PLANNING)

**Milestone:**
- `milestone` field (First Mound, Open Chambers, Brood Stable, Ventilated Nest, Sealed Chambers, Crowned Anthill)
- `milestone_updated_at` field (timestamp of last milestone change)

### Step 2.6: Detect Milestone

Run using the Bash tool with description "Detecting colony milestone...": `bash .aether/aether-utils.sh milestone-detect`

Extract from JSON result:
- `milestone`: Current milestone name
- `version`: Computed version string
- `phases_completed`: Number of completed phases
- `total_phases`: Total phases in plan

### Step 2.7: Generate Progress Bars

Calculate progress metrics and generate visual bars.

Run using the Bash tool with description "Computing phase progress...":
```bash
current_phase=$(jq -r '.current_phase // 0' .aether/data/COLONY_STATE.json)
total_phases=$(jq -r '.plan.phases | length' .aether/data/COLONY_STATE.json)

# Calculate task progress in current phase
if [[ "$current_phase" -gt 0 && "$current_phase" -le "$total_phases" ]]; then
  phase_idx=$((current_phase - 1))
  tasks_completed=$(jq -r ".plan.phases[$phase_idx].tasks // [] | map(select(.status == \"completed\")) | length" .aether/data/COLONY_STATE.json)
  tasks_total=$(jq -r ".plan.phases[$phase_idx].tasks // [] | length" .aether/data/COLONY_STATE.json)
  phase_name=$(jq -r ".plan.phases[$phase_idx].name // \"Unnamed\"" .aether/data/COLONY_STATE.json)
else
  tasks_completed=0
  tasks_total=0
  phase_name="No plan created"
fi

# Generate progress bars
phase_bar=$(bash .aether/aether-utils.sh generate-progress-bar "$current_phase" "$total_phases" 20)
task_bar=$(bash .aether/aether-utils.sh generate-progress-bar "$tasks_completed" "$tasks_total" 20)

echo "phase_bar=$phase_bar"
echo "task_bar=$task_bar"
echo "phase_name=$phase_name"
```

Store `phase_bar`, `task_bar`, and `phase_name` values for display in Step 3.

### Step 3: Display

Output format:

```
       .-.
      (o o)  AETHER COLONY
      | O |  Status Report
       `-`
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

👑 Goal: <goal (truncated to 60 chars)>

📍 Progress
   Phase: [████████░░░░░░░░░░░░] <N>/<M> phases
   Tasks: [████████████████░░░░] <completed>/<total> tasks in Phase <N>

🎯 Focus: <focus_count> areas | 🚫 Avoid: <constraints_count> patterns
🧠 Instincts: <total> learned (<high_confidence> strong)
🚩 Flags: <blockers> blockers | <issues> issues | <notes> notes
🏆 Milestone: <milestone> (<version>)
💭 Dreams: <dream_count> recorded (latest: <latest_dream>)

State: <state>
```

Use the `phase_bar` and `task_bar` values computed in Step 2.7 for the actual bar characters and counts.

**If instincts exist, also show top 3:**
```
🧠 Colony Instincts:
   [0.9] 🐜 testing: Always run tests before completion
   [0.8] 🐜 architecture: Use composition over inheritance
   [0.7] 🐜 debugging: Trace to root cause first
```

**Dream display:**
- If no dreams exist: `💭 Dreams: None recorded`
- If dreams exist: `💭 Dreams: <count> recorded (latest: YYYY-MM-DD HH:MM)`

**Edge cases:**
- No phases yet: show `[░░░░░░░░░░░░░░░░░░░░] 0/0 phases`
- No tasks in phase: show `[░░░░░░░░░░░░░░░░░░░░] 0/0 tasks in Phase 0`
- No constraints file: "Constraints: 0 focus, 0 avoid"

**At the end of the output, generate the Next Up block:**

Run using the Bash tool with description "Generating next steps...":
```bash
state=$(jq -r '.state // "IDLE"' .aether/data/COLONY_STATE.json)
current_phase=$(jq -r '.current_phase // 0' .aether/data/COLONY_STATE.json)
total_phases=$(jq -r '.plan.phases | length' .aether/data/COLONY_STATE.json)

bash .aether/aether-utils.sh print-next-up "$state" "$current_phase" "$total_phases"
```

This auto-generates state-based recommendations (IDLE → init, READY → build, EXECUTING → continue, PLANNING → plan).
