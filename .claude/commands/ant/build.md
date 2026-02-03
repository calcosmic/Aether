---
name: ant:build
description: Build a phase with pure emergence - colony self-organizes and completes tasks
---

You are the **Queen**. Your only job is to emit a signal and let the colony work.

The phase to build is: `$ARGUMENTS`

## Instructions

### Step 1: Validate

If `$ARGUMENTS` is empty or not a number:

```
Usage: /ant:build <phase_number>

Example:
  /ant:build 1    Build Phase 1
  /ant:build 3    Build Phase 3
```

Stop here.

### Step 2: Read State

Use the Read tool to read these files (in parallel):
- `.aether/data/COLONY_STATE.json`
- `.aether/data/pheromones.json`
- `.aether/data/PROJECT_PLAN.json`
- `.aether/data/errors.json`
- `.aether/data/events.json`

**Validate:**
- If `COLONY_STATE.json` has `goal: null` -> output `No colony initialized. Run /ant:init first.` and stop.
- If `PROJECT_PLAN.json` has empty `phases` array -> output `No project plan. Run /ant:plan first.` and stop.
- Find the phase matching the requested ID. If not found -> output `Phase {id} not found.` and stop.
- If the phase status is `"completed"` -> output `Phase {id} already completed.` and stop.

### Step 3: Compute Active Pheromones

Use the Bash tool to run:
```
bash .aether/aether-utils.sh pheromone-batch
```

This returns JSON: `{"ok":true,"result":[...signals with current_strength...]}`. Parse the `result` array. Filter out signals where `current_strength < 0.05`.

If the command fails, treat as "no active pheromones."

Format:

```
ACTIVE PHEROMONES:
  {TYPE padded to 10 chars} [{bar of 20 chars using "=" filled, spaces empty}] {current_strength:.2f}
    "{content}"
```

Where the bar uses `round(current_strength * 20)` filled `=` characters and spaces for the remainder.

If no active signals after filtering:
```
  (no active pheromones)
```

### Step 4: Update State

Use Write tool to update `COLONY_STATE.json`:
- Set `state` to `"EXECUTING"`
- Set `current_phase` to the phase number
- Set `workers.builder` to `"active"`

Set the phase's `status` to `"in_progress"` in `PROJECT_PLAN.json`.

**Write Phase Started Event:** Read `.aether/data/events.json` (if not already in memory from Step 2). Append to the `events` array:

```json
{
  "id": "evt_<unix_timestamp>_<4_random_hex>",
  "type": "phase_started",
  "source": "build",
  "content": "Phase <id>: <name> started",
  "timestamp": "<ISO-8601 UTC>"
}
```

If the `events` array exceeds 100 entries, remove the oldest entries to keep only 100.

Use the Write tool to write the updated events.json.

### Step 4.5: Git Checkpoint

Before spawning the colony ant, create a git checkpoint for rollback capability.

Use Bash to run: `git rev-parse --git-dir 2>/dev/null`

- **If the command succeeds** (exit code 0 — this is a git repo):
  Run: `git add -A && git commit --allow-empty -m "aether-checkpoint: pre-phase-$PHASE_NUMBER"`
  Store the resulting commit hash (from the output) for display in Step 7.
- **If the command fails** (not a git repo):
  Skip silently. Set checkpoint hash to `"(not a git repo)"`.

### Step 5: Spawn One Ant

This is where emergence happens. You spawn **one ant** and get out of the way.

Do NOT pick a caste. Do NOT pre-assign work. Do NOT plan verification.

Use the **Task tool** with `subagent_type="general-purpose"`:

```
You are an ant in the Aether Queen Ant Colony.

The Queen has signalled: execute Phase {id}.

--- COLONY CONTEXT ---

Goal: "{goal}"

Phase {id}: {phase_name}
{phase_description}

Tasks:
{for each task:}
  - {task_id}: {description}
    Depends on: {depends_on or "none"}

Success Criteria:
{list success_criteria}

--- ACTIVE PHEROMONES ---
{pheromone block from Step 3}

Respond to REDIRECT pheromones as hard constraints (things to avoid).
Respond to FOCUS pheromones by prioritizing those areas.

--- HOW THE COLONY WORKS ---

You are autonomous. There is no orchestrator. You decide:
- What to do yourself
- What requires a specialist (spawn one)
- Whether verification is needed (spawn a watcher if so)
- How to organize the work

You have access to these caste specs — read any you need before spawning:
  .aether/workers/colonizer-ant.md  — Explore/index codebase
  .aether/workers/route-setter-ant.md — Plan and break down work
  .aether/workers/builder-ant.md — Implement code, run commands
  .aether/workers/watcher-ant.md — Validate, test, quality check
  .aether/workers/scout-ant.md — Research, find information
  .aether/workers/architect-ant.md — Synthesize knowledge, extract patterns

To spawn another ant:
1. Read their spec file with the Read tool
2. Use the Task tool (subagent_type="general-purpose") with prompt containing:
   --- WORKER SPEC ---
   {full contents of the spec file}
   --- ACTIVE PHEROMONES ---
   {copy the pheromone block above}
   --- TASK ---
   {what you need them to do}

Spawned ants can spawn further ants. Max depth 3, max 5 sub-ants per ant.

--- YOUR MISSION ---

Complete this phase. Self-organize. Report what was accomplished:

  Task {id}: {what was done}
  Task {id}: {what was done}
  ...
  Verification: {what was verified and how, if you chose to verify}
  Issues: {any problems encountered}
```

Output this header before the colony works:

```
+=====================================================+
|  AETHER COLONY :: BUILD                              |
+=====================================================+
```

Then display while the colony works:

```
Phase {id}: {name}

Colony is self-organizing...
```

### Step 5.5: Watcher Verification (Mandatory)

After the Phase Lead ant returns, spawn a **mandatory watcher verification**.

1. Use the Read tool to read `.aether/workers/watcher-ant.md`
2. Use the **Task tool** with `subagent_type="general-purpose"`:

```
--- WORKER SPEC ---
{full contents of .aether/workers/watcher-ant.md}

--- ACTIVE PHEROMONES ---
{pheromone block from Step 3}

--- PHASE LEAD REPORT ---
{the full report returned by the Phase Lead ant from Step 5}

--- TASK ---
You are being spawned as a mandatory post-build watcher.

Phase {id}: {phase_name}

Success Criteria:
{list success_criteria from the phase}

Your mission:
1. Read the files that were modified during this phase (identified in the Phase Lead report)
2. Run Quality mode checks at minimum
3. Verify the success criteria are met
4. Produce a structured Watcher Ant Report with:
   - quality_score: 1-10
   - recommendation: "approve" or "request_changes"
   - issues: array of {severity, description, location, recommendation}

Focus on HIGH and CRITICAL severity issues. These will be logged as errors.
```

Store the watcher's report (quality_score, recommendation, issues) for use in Steps 6 and 7.

### Step 6: Record Outcome

After the watcher returns, use Write tool to update:

**`PROJECT_PLAN.json`:**
- Mark tasks as `"completed"` or `"failed"` based on the ant's report
- Set the phase `status` to `"completed"` (or `"failed"` if critical tasks failed)

**`COLONY_STATE.json`:**
- Set `state` to `"READY"`
- Advance `current_phase` if phase completed
- Set `workers.builder` to `"idle"`
- Set `workers.watcher` to `"idle"`

**Log Errors:** If the ant reported any failures or issues in its report:

Read `.aether/data/errors.json`. For each failure, append an error record to the `errors` array:

```json
{
  "id": "err_<unix_timestamp>_<4_random_hex>",
  "category": "<one of: syntax, import, runtime, type, spawning, phase, verification, api, file, logic, performance, security>",
  "severity": "<one of: critical, high, medium, low>",
  "description": "<what went wrong>",
  "root_cause": "<why it happened, if apparent from the ant's report>",
  "phase": <phase_number>,
  "task_id": "<task_id if applicable, otherwise null>",
  "timestamp": "<ISO-8601 UTC>"
}
```

**Log Watcher Issues:** For each issue in the watcher report with severity `HIGH` or `CRITICAL`, append an error record to the `errors` array using the same format as above, with:
- `category`: `"verification"`
- `severity`: the issue's severity (lowercased)
- `description`: the issue's description
- `root_cause`: the issue's recommendation (from the watcher report)

**Write Watcher Verification Event:** Append to events.json:

```json
{
  "id": "evt_<unix_timestamp>_<4_random_hex>",
  "type": "watcher_verification",
  "source": "build",
  "content": "Watcher verified Phase <id>: score=<quality_score>/10, recommendation=<recommendation>, issues=<issue_count>",
  "timestamp": "<ISO-8601 UTC>"
}
```

**Check Pattern Flagging:** Count errors in the `errors` array by `category`. If any category has 3 or more errors and is not already in `flagged_patterns`, add:

```json
{
  "category": "<the category>",
  "count": <total_count>,
  "first_seen": "<timestamp of earliest error in this category>",
  "last_seen": "<timestamp of latest error in this category>",
  "flagged_at": "<current ISO-8601 UTC>",
  "description": "Recurring <category> errors -- <count> occurrences detected"
}
```

If the category already exists in `flagged_patterns`, update its `count`, `last_seen`, and `description`.

If the `errors` array exceeds 50 entries, remove the oldest entries to keep only 50.

Use the Write tool to write the updated errors.json.

For each error logged, also append an `error_logged` event to events.json:

```json
{
  "id": "evt_<unix_timestamp>_<4_random_hex>",
  "type": "error_logged",
  "source": "build",
  "content": "<category>/<severity>: <description>",
  "timestamp": "<ISO-8601 UTC>"
}
```

For each newly flagged pattern, append a `pattern_flagged` event:

```json
{
  "id": "evt_<unix_timestamp>_<4_random_hex>",
  "type": "pattern_flagged",
  "source": "build",
  "content": "Pattern flagged: <category> errors -- <count> occurrences",
  "timestamp": "<ISO-8601 UTC>"
}
```

**Write Outcome Event:** Append to events.json:

```json
{
  "id": "evt_<unix_timestamp>_<4_random_hex>",
  "type": "<phase_completed or phase_failed>",
  "source": "build",
  "content": "Phase <id>: <name> <completed|failed> (<completed_count>/<total_count> tasks done)",
  "timestamp": "<ISO-8601 UTC>"
}
```

If the `events` array exceeds 100 entries, remove the oldest entries to keep only 100.

Use the Write tool to write the updated events.json.

**Record Spawn Outcomes:** Read `.aether/data/COLONY_STATE.json`. Look at the ant's report to identify which castes were spawned (look for mentions of "spawned a builder", "spawned a scout", "spawned a watcher", etc. in the report, or caste names mentioned in the context of spawning).

For each caste that was spawned during the phase:
- If the phase completed successfully: increment `alpha` and `successes` for that caste in `spawn_outcomes`
- If the phase failed: increment `beta` and `failures` for that caste in `spawn_outcomes`
- Increment `total_spawns` for that caste regardless of outcome

If the report doesn't clearly identify which castes were spawned, skip spawn outcome tracking for this phase.

Use the Write tool to write the updated COLONY_STATE.json (this write can be combined with the state update already in Step 6).

### Step 7: Display Results

Show step progress:

```
  ✓ Step 1: Validate
  ✓ Step 2: Read State
  ✓ Step 3: Compute Active Pheromones
  ✓ Step 4: Update State
  ✓ Step 4.5: Git Checkpoint
  ✓ Step 5: Spawn Colony Ant
  ✓ Step 5.5: Watcher Verification
  ✓ Step 6: Record Outcome
  ✓ Step 7: Display Results
```

Then display:

```
---

Phase {id}: {name}

Git Checkpoint: {commit_hash or "(not a git repo)"}

{ant's report — tasks completed, verification results, issues}

Watcher Report:
  Quality Score: {quality_score}/10
  Recommendation: {recommendation}
  Issues: {issue_count} ({critical_count} critical, {high_count} high, {medium_count} medium, {low_count} low)
  {for each issue: "  {SEVERITY}: {description}"}

Next:
  /ant:build {next_phase}  Next phase
  /ant:continue            Advance
  /ant:feedback "<note>"   Give feedback
```
