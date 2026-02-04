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
  {TYPE padded to 10 chars} [{bar of 20 chars using "█" filled, spaces empty}] {current_strength:.2f}
    "{content}"
```

Where the bar uses `round(current_strength * 20)` filled `█` characters and spaces for the remainder.

If no active signals after filtering:
```
  (no active pheromones)
```

**Per-Caste Effective Signals:** After computing the active pheromones, build a per-caste effectiveness table for display in Step 7. For each active pheromone signal, compute:

```
effective_signal = caste_sensitivity × current_strength
```

Using the sensitivity table:

```
                INIT  FOCUS  REDIRECT  FEEDBACK
  colonizer     1.0   0.7    0.3       0.5
  route-setter  1.0   0.5    0.8       0.7
  builder       0.5   0.9    0.9       0.7
  watcher       0.3   0.8    0.5       0.9
  scout         0.7   0.9    0.4       0.5
  architect     0.2   0.4    0.3       0.6
```

Store the computed table for use in Step 7 display. Format each entry as:
```
  {caste_emoji} {caste}: {signal_type} {effective:.2f} ({">0.5 PRIORITIZE" | "0.3-0.5 NOTE" | "<0.3 IGNORE"})
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

Output this header before spawning the Phase Lead:

```
+=====================================================+
|  👑 AETHER COLONY :: BUILD                           |
+=====================================================+

Phase {id}: {name}
```

### Step 5a: Spawn Phase Lead as Planner

Spawn **one Phase Lead ant** whose job is to produce a task assignment plan. It does NOT spawn workers.

Use the **Task tool** with `subagent_type="general-purpose"`:

```
You are the Phase Lead 🐜 in the Aether Queen Ant Colony.

You MUST NOT use the Task tool. You MUST NOT spawn any workers.
Your ONLY job is to produce a task assignment plan.

The Queen has signalled: plan Phase {id}.

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

--- CASTE SENSITIVITY TABLE ---
                INIT  FOCUS  REDIRECT  FEEDBACK
  colonizer     1.0   0.7    0.3       0.5
  route-setter  1.0   0.5    0.8       0.7
  builder       0.5   0.9    0.9       0.7
  watcher       0.3   0.8    0.5       0.9
  scout         0.7   0.9    0.4       0.5
  architect     0.2   0.4    0.3       0.6

Available worker castes:
  🗺️🐜 colonizer-ant — Explore/index codebase
  📋🐜 route-setter-ant — Plan and break down work (rarely needed as worker)
  🔨🐜 builder-ant — Implement code, run commands
  🔍🐜 scout-ant — Research, find information
  🏛️🐜 architect-ant — Synthesize knowledge, extract patterns

--- VISUAL IDENTITY ---
You are the Phase Lead 🐜. Use emoji in all output.

--- YOUR MISSION ---

Produce a task assignment plan. Group related tasks into waves. Independent
tasks go in the same wave. Tasks with dependencies go in later waves.
Assign each group to a worker caste.

Output format:

  Phase Lead Task Assignment Plan
  ================================

  Colony is planning...

  Wave 1 (independent):
    1. {caste_emoji} {caste}-ant: {task description} (tasks {ids})
    2. {caste_emoji} {caste}-ant: {task description} (tasks {ids})

  Wave 2 (depends on Wave 1):
    3. {caste_emoji} {caste}-ant: {task description} (tasks {ids})
       Needs: {what from Wave 1}

  Worker count: {N}
  Wave count: {N}

{if user_feedback is present:}
The user reviewed your previous plan and requested these changes: {user_feedback}
Produce a revised plan incorporating their feedback.
```

### Step 5b: Plan Checkpoint

After the Phase Lead returns:

1. Display the Phase Lead's task assignment plan to the user verbatim
2. Ask: **"Proceed with this plan? (yes / describe changes)"**
3. If user says "yes" or equivalent: proceed to Step 5c
4. If user describes changes: Re-run Step 5a with the user's feedback appended to the prompt
5. After re-run, display revised plan and ask again
6. Maximum 3 plan iterations before proceeding with the latest plan

### Step 5c: Execute Plan

This is the core execution loop. The Queen spawns workers directly.

**1. Initialize activity log:**
```
bash .aether/aether-utils.sh activity-log-init {phase_number} "{phase_name}"
```

**2. Parse the plan:** Extract waves and worker assignments from the Phase Lead's plan output. Track: wave number, caste, task description, task IDs, dependencies.

**3. Initialize counters:** `completed_workers = 0`, `total_workers = {from plan}`, `worker_results = []`

**4. For each wave in the plan:**

Display wave header:
```
--- Wave {N}/{total_waves} ---
```

For each worker assignment in this wave:

a. **Announce spawn:**
   ```
   Spawning {caste_emoji} {caste}-ant for: {task_description}...
   ```

b. **Log START:**
   ```
   bash .aether/aether-utils.sh activity-log "START" "{caste}-ant" "{task_description}"
   ```

c. **Read worker spec:** Use Read tool to read `.aether/workers/{caste}-ant.md`

d. **Spawn worker via Task tool** with `subagent_type="general-purpose"`:
   ```
   --- WORKER SPEC ---
   {full contents of the caste's spec file}

   --- ACTIVE PHEROMONES ---
   {pheromone block from Step 3 with effective signals computed for this caste}

   --- TASK ---
   {task_description}

   Colony goal: "{goal}"
   Phase {id}: {phase_name}

   Task details:
   {for each task ID assigned to this worker, include the full task description and depends_on from PROJECT_PLAN.json}

   {if this worker has dependencies on previous workers:}
   Context from previous workers:
   {relevant results from prior workers in this phase}

   You are at depth 1.
   ```

e. **After worker returns:**
   - Log COMPLETE (or ERROR if worker reported failure):
     ```
     bash .aether/aether-utils.sh activity-log "COMPLETE" "{caste}-ant" "{task_description}"
     ```
     or:
     ```
     bash .aether/aether-utils.sh activity-log "ERROR" "{caste}-ant" "{error_summary}"
     ```
   - Read activity log entries for this worker:
     ```
     bash .aether/aether-utils.sh activity-log-read "{caste}-ant"
     ```
   - Increment `completed_workers`
   - Display condensed summary:
     ```
     {caste_emoji} {caste}-ant: {task_description}
       Result: {COMPLETE or ERROR}
       Files: {count of created/modified from worker report}
       {if error: brief error description}

       {progress_bar} {completed_workers}/{total_workers} workers complete
     ```
     Progress bar: `filled = round(completed / total * 20)` filled characters, rest empty, total width 20.

   - Store worker result (report content, success/failure, task IDs) in `worker_results` for use by subsequent workers and Step 5.5.

f. **If worker failed and retry count < 2:**
   - Log retry:
     ```
     bash .aether/aether-utils.sh activity-log "ERROR" "{caste}-ant" "retry {N}: {error_summary}"
     ```
   - Spawn a NEW worker (same caste, same task) with failure context appended:
     ```
     Previous attempt failed because: {error_description}. Try a different approach.
     ```
   - Increment retry counter for this task

g. **If worker failed and retry count >= 2:**
   - Display: `"Task failed after 2 retries. Continuing with remaining tasks."`
   - Mark task as failed in tracking
   - Continue to next worker

**5. Compile Phase Build Report** from all `worker_results`:
```
Phase Build Report
==================

Workers spawned: {total}
Completed: {success_count}
Failed: {fail_count}

Per-worker results:
{for each worker: caste, task, result, files created/modified}

Files Modified:
  Created: {combined list}
  Modified: {combined list}

Issues: {any failures or errors}
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
2. EXECUTE the code — run syntax checks, import checks, and launch test (see your spec's Execution Verification section)
3. Run Quality mode checks at minimum
4. Verify the success criteria are met
5. If any execution check fails, quality_score CANNOT exceed 6/10
6. Produce a structured Watcher Ant Report with:
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

For each failure, use the Bash tool to run:
```
bash .aether/aether-utils.sh error-add "<category>" "<severity>" "<description>"
```

Where `category` is one of: syntax, import, runtime, type, spawning, phase, verification, api, file, logic, performance, security. And `severity` is one of: critical, high, medium, low.

Each call returns `{"ok":true,"result":"<error_id>"}`. Note the returned error IDs for event logging.

**Log Watcher Issues:** For each issue in the watcher report with severity `HIGH` or `CRITICAL`, use the Bash tool to run:
```
bash .aether/aether-utils.sh error-add "verification" "<severity_lowercased>" "<description>"
```

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

**Check Pattern Flagging:** Use the Bash tool to run:
```
bash .aether/aether-utils.sh error-pattern-check
```

This returns JSON: `{"ok":true,"result":[{"category":"...","count":N,"first_seen":"...","last_seen":"..."},...]}`

For each category in the result that is not already in `flagged_patterns`, add:

```json
{
  "category": "<the category>",
  "count": <total_count>,
  "first_seen": "<first_seen from result>",
  "last_seen": "<last_seen from result>",
  "flagged_at": "<current ISO-8601 UTC>",
  "description": "Recurring <category> errors -- <count> occurrences detected"
}
```

If the category already exists in `flagged_patterns`, update its `count`, `last_seen`, and `description`.

If the command fails, fall back to manual counting from the errors.json data already in memory.

If the `errors` array exceeds 50 entries, remove the oldest entries to keep only 50.

**Get Error Summary:** Use the Bash tool to run:
```
bash .aether/aether-utils.sh error-summary
```

This returns JSON: `{"ok":true,"result":{"total":N,"by_category":{...},"by_severity":{...}}}`. Use these counts for the issue summary in Step 7 display.

If the command fails, derive counts manually from the errors.json data already in memory.

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

🔒 Git Checkpoint: {commit_hash or "(not a git repo)"}

🐜 Colony Activity:
  {Phase Lead's delegation log — extract from the Phase Lead report:
   show each ant that was spawned, what it did, and its result}

📋 Task Results:
  {for each task: "✅ {task_id}: {what was done}" or "❌ {task_id}: {what failed}"}

🧪 Caste Pheromone Sensitivity:
  {per-caste effective signals computed in Step 3, showing which castes
   would PRIORITIZE/NOTE/IGNORE each active pheromone}

👁️🐜 Watcher Report:
  Execution Verification:
    {syntax/import/launch/test results from the watcher}
  Quality: {"⭐" repeated for round(quality_score/2)} ({quality_score}/10)
  Recommendation: {recommendation}
  Issues: {issue_count}
    🔴 Critical: {critical_count}  🟠 High: {high_count}  🟡 Medium: {medium_count}  ⚪ Low: {low_count}
  {for each issue: "  {SEVERITY}: {description}"}

⚠️ IMPORTANT: Run /ant:continue to extract learnings before building the next phase.
Skipping /ant:continue means phase learnings are lost and the feedback loop breaks.

Next:
  /ant:continue            Extract learnings and advance (recommended)
  /ant:feedback "<note>"   Give feedback first
  /ant:status              View full colony status
```
