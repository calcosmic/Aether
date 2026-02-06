---
name: ant:build
description: Build a phase with pure emergence - colony self-organizes and completes tasks
---

You are the **Queen**. Your only job is to emit a signal and let the colony work.

The phase to build is: `$ARGUMENTS`

## Instructions

<!-- Color Reference (ANSI 8-color codes for Bash tool printf/echo)
  COLOR_QUEEN="\e[1;33m"        # Bold yellow
  COLOR_COLONIZER="\e[36m"      # Cyan
  COLOR_ROUTESETTER="\e[33m"    # Yellow
  COLOR_BUILDER="\e[32m"        # Green
  COLOR_WATCHER="\e[35m"        # Magenta
  COLOR_SCOUT="\e[34m"          # Blue
  COLOR_ARCHITECT="\e[37m"      # White/bright
  COLOR_DEBUGGER="\e[31m"       # Red
  COLOR_REVIEWER="\e[34m"       # Blue (watcher variant)
  COLOR_RESET="\e[0m"           # Reset

  Caste-to-code mapping for printf:
    builder     -> 32 (green)
    watcher     -> 35 (magenta)
    colonizer   -> 36 (cyan)
    scout       -> 34 (blue)
    architect   -> 37 (white)
    route-setter -> 33 (yellow)
    debugger    -> 31 (red)
    reviewer    -> 34 (blue)

  Usage: All colored output MUST go through Bash tool calls (printf/echo).
  The Queen's own markdown text between bash calls remains plain.
  Use basic 8-color codes (30-37, bold 1;3X) only -- universally supported.
-->

### Step 1: Validate + Read State

If `$ARGUMENTS` is empty or not a number:

```
Usage: /ant:build <phase_number>

Example:
  /ant:build 1    Build Phase 1
  /ant:build 3    Build Phase 3
```

Stop here.

Use the Read tool to read `.aether/data/COLONY_STATE.json`.

Extract:
- `goal`, `state`, `current_phase`, `mode` from top level
- `plan.phases` for phase data
- `signals` for pheromone guidance
- `errors.records` for error context
- `memory` for decisions/learnings

**Validate:**
- If `COLONY_STATE.json` has `goal: null` -> output `No colony initialized. Run /ant:init first.` and stop.
- If `plan.phases` is empty -> output `No project plan. Run /ant:plan first.` and stop.
- Find the phase matching the requested ID in `plan.phases`. If not found -> output `Phase {id} not found.` and stop.
- If the phase status is `"completed"` -> output `Phase {id} already completed.` and stop.

### Step 2: Update State (Minimal)

Use Write tool to update `.aether/data/COLONY_STATE.json`:
- Set `state` to `"EXECUTING"`
- Set `current_phase` to the phase number
- Set `workers.builder` to `"active"`
- Add `build_started_at` field with current ISO-8601 UTC timestamp
- Set the phase's `status` to `"in_progress"` in `plan.phases[N]`

**Write Phase Started Event:** Append to the `events` array as pipe-delimited string:
`"<timestamp>|phase_started|build|Phase <id>: <name> started"`

If the `events` array exceeds 100 entries, remove the oldest entries to keep only 100.

Write the updated COLONY_STATE.json.

**CRITICAL:** Do NOT update task statuses, learnings, pheromones, or spawn_outcomes in this step. Those are handled by /ant:continue after build completion.

### Step 3: Git Checkpoint

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

### Step 4: Compute Active Pheromones

Use the Bash tool to run:
```
bash ~/.aether/aether-utils.sh pheromone-batch
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

**Per-Caste Effective Signals:** Build a per-caste effectiveness table for worker prompts. For each active pheromone signal, compute:

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

Store the computed table for worker prompts.

### Step 5: Execute Phase

This step handles Phase Lead planning and worker execution.

#### Step 5a: Spawn Phase Lead as Planner

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
{pheromone block from Step 4}

--- CONFLICT PREVENTION RULE ---
CRITICAL: Tasks that modify the SAME FILE must be assigned to the SAME WORKER.

Before creating your plan:
1. For each task, identify which files it will likely create or modify
2. If two tasks reference the same file path, they MUST go to the same worker in the same wave
3. If unsure about file overlap, group tasks conservatively (same worker)

--- DEFAULT-PARALLEL RULE ---
CRITICAL: Tasks are PARALLEL by default. Only serialize when you have a specific reason.

For EACH task, list the files it will likely create or modify.
Two tasks are INDEPENDENT if they have zero file overlap AND no explicit dependency.
Independent tasks go in the SAME wave with different workers.

--- MODE-AWARE PARALLELISM ---
Mode: {mode from COLONY_STATE.json}
- LIGHTWEIGHT: max 1 worker per wave (serialized execution)
- STANDARD: normal parallelism (default behavior)
- FULL: aggressive parallelism, up to 4 concurrent workers per wave

Available worker castes:
  🗺️🐜 colonizer-ant — Explore/index codebase
  📋🐜 route-setter-ant — Plan and break down work
  🔨🐜 builder-ant — Implement code, run commands
  🔍🐜 scout-ant — Research, find information
  🏛️🐜 architect-ant — Synthesize knowledge, extract patterns

--- YOUR MISSION ---

Produce a task assignment plan. Group related tasks into waves. Independent
tasks go in the same wave. Tasks with dependencies go in later waves.

Output format:

  Phase Lead Task Assignment Plan
  ================================

  Wave 1 ({N} parallel workers):
    1. {caste_emoji} {caste}-ant: {task description} (tasks {ids}) -> {file paths}
    2. {caste_emoji} {caste}-ant: {task description} (tasks {ids}) -> {file paths}

  Wave 2 (depends on Wave 1):
    3. {caste_emoji} {caste}-ant: {task description} (tasks {ids}) -> {file paths}
       Needs: {what from Wave 1}

  Parallelism: {parallel_count}/{total_count} tasks in Wave 1 ({percentage}%)
  Worker count: {N}
  Wave count: {N}
```

#### Step 5b: Plan Checkpoint

After the Phase Lead returns:

**Auto-Approval Check:**
Check the `mode` field from COLONY_STATE.json.

If mode is "LIGHTWEIGHT": auto-approve the plan. Skip user confirmation.
Display: "Plan auto-approved (LIGHTWEIGHT mode). Proceeding to execution..."

If mode is "STANDARD" or "FULL" (or mode is null/missing):
  Count from the Phase Lead's plan: task_count, worker_count, wave_count, shared_files

  If mode is "STANDARD" AND task_count <= 4 AND worker_count <= 2 AND wave_count <= 2 AND shared_files == false:
    Auto-approve. Display: "Plan auto-approved (simple phase). Proceeding to execution..."

  Otherwise: Display plan to user, ask "Proceed with this plan? (yes / describe changes)"
  Maximum 3 plan iterations before proceeding with latest plan.

#### Step 5b-post: Record Plan Decisions

After the plan is approved, record 2-3 strategic decisions to COLONY_STATE.json `memory.decisions` array:

```json
{
  "id": "dec_<unix_timestamp>_<4_random_hex>",
  "type": "plan",
  "content": "<strategic decision>",
  "context": "Phase <id> plan",
  "phase": <current_phase_number>,
  "timestamp": "<ISO-8601 UTC>"
}
```

Cap at 30 entries. Write updated COLONY_STATE.json.

#### Step 5c: Execute Plan

**1. Initialize activity log:**
```
bash ~/.aether/aether-utils.sh activity-log-init {phase_number} "{phase_name}"
```

**2. Parse the plan:** Extract waves and worker assignments. Track: wave number, caste, task description, task IDs, dependencies.

**2b. Validate file overlap:** For each pair of workers in the SAME wave, if they reference the same file path, MERGE those assignments into a single worker.

**3. Initialize counters:** `completed_workers = 0`, `total_workers = {from plan}`, `worker_results = []`
Initialize spawn tracking: `spawn_tree = {}`, `queued_sub_spawns = []`

**4. For each wave in the plan:**

Display wave header using Bash tool:
```
bash -c 'printf "\e[1;33m--- Wave %d/%d ---\e[0m\n" {N} {total_waves}'
```

For each worker assignment in this wave:

a. **Announce spawn:** Display with caste-specific color
b. **Log START:** `bash ~/.aether/aether-utils.sh activity-log "START" "{caste}-ant" "{task}"`
c. **Read worker spec:** Read `~/.aether/workers/{caste}-ant.md`
d. **Spawn worker via Task tool** with worker spec, pheromones, and task context

e. **After worker returns:**
   - Log COMPLETE or ERROR
   - Display colored completion status
   - Display progress bar
   - Update spawn_tree, store in worker_results

e2. **Parse SPAWN requests:** If worker output contains `SPAWN REQUEST:` blocks, queue them (max 2 per wave)

f. **If worker failed and retry count < 1:** Spawn new worker with failure context
f2. **If retry also failed:** Spawn debugger-ant to diagnose and fix

g. **Post-debugger logic:** Mark task completed/failed based on debugger result

h. **Post-Wave Conflict Check:** Compare MODIFIED/CREATED entries for file overlap. If conflict detected, HALT and prompt user.

i. **Post-Wave Advisory Review:** (Skip in LIGHTWEIGHT mode or single-worker waves)
   Spawn reviewer to check for CRITICAL issues. If critical_count > 0 AND wave_rebuild_count < 2, rebuild wave.

j. **Fulfill SPAWN requests:** For each queued sub-spawn, spawn at depth 2.

**5. Compile Phase Build Report** from all worker_results.

### Step 6: Watcher Verification

After all workers complete, spawn a **mandatory watcher verification**.

**Mode Check:** If mode is "LIGHTWEIGHT", skip watcher verification entirely.
Display: "Watcher verification skipped (LIGHTWEIGHT mode)." and proceed to Step 7.

Otherwise:

1. Read `~/.aether/workers/watcher-ant.md`
2. Spawn watcher via Task tool with Phase Build Report, success criteria

Store the watcher's report (quality_score, recommendation, issues) for display.

**Record Quality Decision:** Append to `memory.decisions`:
```json
{
  "id": "dec_<unix_timestamp>_<4_random_hex>",
  "type": "quality",
  "content": "<watcher verdict>",
  "context": "Phase <id> watcher verification",
  "phase": <current_phase_number>,
  "timestamp": "<ISO-8601 UTC>"
}
```

Cap at 30 entries. Write updated COLONY_STATE.json.

### Step 7: Display Results

Display the build summary header using Bash tool:
```
bash -c 'printf "\n\e[1;33m+=====================================================+\e[0m\n"'
bash -c 'printf "\e[1;33m|  BUILD COMPLETE                                     |\e[0m\n"'
bash -c 'printf "\e[1;33m+=====================================================+\e[0m\n\n"'
```

Display:

```
Phase {id}: {name}

🔒 Git Checkpoint: {commit_hash or "(not a git repo)"}

🐜 Colony Activity:
  {Per-worker results from Step 5c -- caste, task, result}
  Activity Log: .aether/data/activity.log
```

Display the Delegation Tree using Bash tool:
```
bash -c 'printf "\n\e[1;33mDelegation Tree:\e[0m\n"'
bash -c 'printf "  \e[1;33mQueen\e[0m\n"'
```
For each spawn_tree entry where depth == 1:
`bash -c 'printf "  ├── \e[{caste_color}m%s\e[0m: %s [\e[32m%s\e[0m]\n" "{caste}" "{task}" "{status}"'`
For each child (depth == 2):
`bash -c 'printf "  │   └── \e[{caste_color}m%s\e[0m (sub): %s [\e[32m%s\e[0m]\n" "{caste}" "{task}" "{status}"'`
If spawn_tree is empty: `bash -c 'printf "  (no delegation -- all tasks handled directly)\n"'`

```
📋 Task Results:
  {for each task: "✅ {task_id}: {what was done}" or "❌ {task_id}: {what failed}"}

👁️🐜 Watcher Report:
  Execution Verification:
    {syntax/import/launch/test results}
  Quality: {"⭐" repeated for round(quality_score/2)} ({quality_score}/10)
  Recommendation: {recommendation}
  Issues: {issue_count}
    🔴 Critical: {critical_count}  🟠 High: {high_count}  🟡 Medium: {medium_count}  ⚪ Low: {low_count}
  {for each issue: "  {SEVERITY}: {description}"}
```

Display Pheromone Recommendations using Bash tool:
```
bash -c 'printf "\e[33mPheromone Recommendations:\e[0m\n"'
```

Based on worker_results, watcher_report, and errors.flagged_patterns, generate max 3 natural language recommendations:

```
Pheromone Recommendations:
  Based on this build's outcomes:

  1. {natural language recommendation}
     Signal: {specific observation}

  2. {natural language recommendation}
     Signal: {specific observation}

  These are suggestions, not commands. Use /ant:focus or /ant:redirect to act on them.
```

If no meaningful patterns: "No specific recommendations -- build was clean."

```
Next:
  /ant:continue            Advance to next phase (records learnings, emits pheromones)
  /ant:feedback "<note>"   Give feedback first
  /ant:status              View full colony status
```

**IMPORTANT:** Build does NOT write final state. Run /ant:continue to record outcomes, extract learnings, and emit pheromones. This ensures state survives context boundaries.
