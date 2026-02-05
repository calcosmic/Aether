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

--- CONFLICT PREVENTION RULE ---
CRITICAL: Tasks that modify the SAME FILE must be assigned to the SAME WORKER.

Before creating your plan:
1. For each task, identify which files it will likely create or modify
2. If two tasks reference the same file path, they MUST go to the same worker in the same wave
3. If unsure about file overlap, group tasks conservatively (same worker)

This prevents parallel write conflicts where one builder overwrites another's work.

Example:
  Task 3.1: Add auth routes to src/routes/index.ts
  Task 3.2: Add API routes to src/routes/index.ts
  -> Both touch src/routes/index.ts -> assign to SAME builder-ant

  Task 3.3: Create middleware at src/middleware/auth.ts
  -> Different file -> can go to a different builder-ant

--- DEFAULT-PARALLEL RULE ---
CRITICAL: Tasks are PARALLEL by default. Only serialize when you have a specific reason.

For EACH task, list the files it will likely create or modify.
Two tasks are INDEPENDENT if they have zero file overlap AND no explicit dependency.
Independent tasks go in the SAME wave with different workers.
Tasks are DEPENDENT if they share files or one needs the other's output.
Dependent tasks go in SEQUENTIAL waves or the SAME worker.

DEFAULT: Assume tasks are parallel unless you can identify a specific dependency.
Do NOT default to sequential ordering just because tasks are numbered sequentially.

--- MODE-AWARE PARALLELISM ---
Read COLONY_STATE.json mode field.
- LIGHTWEIGHT: max 1 worker per wave (serialized execution)
- STANDARD: normal parallelism (default behavior)
- FULL: aggressive parallelism, up to 4 concurrent workers per wave

If mode field is null or missing, use STANDARD behavior.

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

  Wave 1 ({N} parallel workers):
    1. {caste_emoji} {caste}-ant: {task description} (tasks {ids}) -> {file paths}
    2. {caste_emoji} {caste}-ant: {task description} (tasks {ids}) -> {file paths}

  Wave 2 (depends on Wave 1):
    3. {caste_emoji} {caste}-ant: {task description} (tasks {ids}) -> {file paths}
       Needs: {what from Wave 1}

  Parallelism: {parallel_count}/{total_count} tasks in Wave 1 ({percentage}%)
  Worker count: {N}
  Wave count: {N}

{if user_feedback is present:}
The user reviewed your previous plan and requested these changes: {user_feedback}
Produce a revised plan incorporating their feedback.
```

### Step 5b: Plan Checkpoint

After the Phase Lead returns:

**Auto-Approval Check:**
Read COLONY_STATE.json. Check the `mode` field.

If mode is "LIGHTWEIGHT": auto-approve the plan. Skip user confirmation.
Display: "Plan auto-approved (LIGHTWEIGHT mode). Proceeding to execution..."
Go to Step 5b-post.

If mode is "STANDARD" or "FULL" (or mode is null/missing):
  Count from the Phase Lead's plan:
  - task_count: total tasks assigned
  - worker_count: total workers assigned
  - wave_count: total waves
  - shared_files: whether any two workers in the same wave list the same file path

  If mode is "STANDARD" AND task_count <= 4 AND worker_count <= 2 AND wave_count <= 2 AND shared_files == false:
    Auto-approve. Display: "Plan auto-approved (simple phase: {tasks} tasks, {workers} workers, {waves} waves). Proceeding to execution..."
    Go to Step 5b-post.

  Otherwise (FULL mode, or STANDARD mode above threshold):
    1. Display the Phase Lead's task assignment plan to the user verbatim
    2. Ask: "Proceed with this plan? (yes / describe changes)"
    3. If user says "yes" or equivalent: proceed to Step 5b-post
    4. If user describes changes: Re-run Step 5a with the user's feedback appended
    5. After re-run, display revised plan and ask again
    6. Maximum 3 plan iterations before proceeding with the latest plan

### Step 5b-post: Record Plan Decisions

After the plan is approved, record strategic decisions to memory.json.

Read `.aether/data/memory.json`. Synthesize 2-3 strategic decisions from the approved plan (e.g., task groupings, caste assignments, wave structure rationale, conflict prevention merges). Do NOT log every individual task assignment -- only strategic choices.

Append each decision to the `decisions` array:

```json
{
  "id": "dec_<unix_timestamp>_<4_random_hex>",
  "type": "plan",
  "content": "<strategic decision -- e.g. 'Grouped tasks 3.1+3.2 to single builder because both modify routes/index.ts'>",
  "context": "Phase <id> plan -- <brief plan summary>",
  "phase": <current_phase_number>,
  "timestamp": "<ISO-8601 UTC>"
}
```

If the `decisions` array exceeds 30 entries, remove the oldest entries to keep only 30.

Write the updated memory.json.

### Step 5c: Execute Plan

This is the core execution loop. The Queen spawns workers directly.

**1. Initialize activity log:**
```
bash ~/.aether/aether-utils.sh activity-log-init {phase_number} "{phase_name}"
```

**2. Parse the plan:** Extract waves and worker assignments from the Phase Lead's plan output. Track: wave number, caste, task description, task IDs, dependencies.

**2b. Validate file overlap (Queen backup check):**

Before executing workers, scan the parsed plan for file conflicts. For each pair of worker assignments in the SAME wave:
1. Extract file paths mentioned in their task descriptions (look for paths like `src/...`, `*.ts`, `*.js`, etc.)
2. If two workers in the same wave reference the same file path, MERGE those task assignments into a single worker
3. Log the merge: `bash ~/.aether/aether-utils.sh activity-log "MERGE" "queen" "Merged tasks for {worker_a} and {worker_b}: shared file {filepath}"`

This is a backup check. If the Phase Lead followed the CONFLICT PREVENTION RULE correctly, no merges will be needed. But LLM instruction following is probabilistic, so the Queen validates.

**3. Initialize counters:** `completed_workers = 0`, `total_workers = {from plan}`, `worker_results = []`

Initialize spawn tracking: `spawn_tree = {}`, `queued_sub_spawns = []` (reset queued_sub_spawns per wave).

**4. For each wave in the plan:**

Display wave header using Bash tool (bold yellow -- Queen color):
```
bash -c 'printf "\e[1;33m--- Wave %d/%d ---\e[0m\n" {N} {total_waves}'
```

For each worker assignment in this wave:

a. **Announce spawn:**
   Use Bash tool with caste-specific color (see Color Reference above):
   ```
   bash -c 'printf "\e[{caste_color_code}m%-14s\e[0m %s\n" "Spawning {caste}..." "{task_description}"'
   ```
   Where `{caste_color_code}` maps from the Color Reference (e.g., builder=32, watcher=35, colonizer=36).

b. **Log START:**
   ```
   bash ~/.aether/aether-utils.sh activity-log "START" "{caste}-ant" "{task_description}"
   ```

c. **Read worker spec:** Use Read tool to read `~/.aether/workers/{caste}-ant.md`

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

   After spawning, record in spawn_tree: `spawn_tree[worker_id] = {id: "<phase>_wave<N>_<caste><index>", caste, task, depth: 1, parent: "queen", children: [], status: "pending", phase, wave}`

e. **After worker returns:**
   - Log COMPLETE (or ERROR if worker reported failure):
     ```
     bash ~/.aether/aether-utils.sh activity-log "COMPLETE" "{caste}-ant" "{task_description}"
     ```
     or:
     ```
     bash ~/.aether/aether-utils.sh activity-log "ERROR" "{caste}-ant" "{error_summary}"
     ```
   - Read activity log entries for this worker:
     ```
     bash ~/.aether/aether-utils.sh activity-log-read "{caste}-ant"
     ```
   - Increment `completed_workers`
   - Display condensed summary using Bash tool with caste-specific color (see Color Reference):
     For successful completion:
     ```
     bash -c 'printf "\e[{caste_color_code}m%-12s\e[0m %s ... \e[32mCOMPLETE\e[0m\n" "[{CASTE}]" "{task_description}"'
     ```
     For error (always red regardless of caste):
     ```
     bash -c 'printf "\e[31m%-12s\e[0m %s ... \e[31mERROR\e[0m\n" "[{CASTE}]" "{task_description}"'
     ```
     Where `{caste_color_code}` maps from the Color Reference (e.g., builder=32, watcher=35).

     Display progress bar with caste-colored fill:
     ```
     bash -c 'printf "\e[{caste_color_code}m[%s%s]\e[0m %d/%d workers complete\n" "{filled}" "{empty}" {completed_workers} {total_workers}'
     ```
     Where `filled` = `round(completed / total * 20)` `#` characters, `empty` = remaining `.` characters, total width 20.

   - Update spawn_tree entry: status -> "completed" or "failed" based on worker result.
   - Store worker result (report content, success/failure, task IDs) in `worker_results` for use by subsequent workers and Step 5.5.

e2. **Parse SPAWN requests from worker output:**
    If the worker's result text contains one or more `SPAWN REQUEST:` blocks:
    For each SPAWN REQUEST block, extract: caste, reason, task, context, files.
    Validate depth: current worker is depth 1, sub-spawn would be depth 2 -- ALLOWED (max depth 2).
    Queue each valid request: `queued_sub_spawns.append({parent_id: "<wave>_<caste>_<index>", caste: <requested_caste>, task: <requested_task>, context, files, parent_pheromone_context})`
    Cap: Maximum 2 sub-spawns per wave. If more requested, take first 2 and log:
    `bash ~/.aether/aether-utils.sh activity-log "SKIP" "queen" "Excess SPAWN REQUEST ignored (cap 2/wave)"`

f. **If worker failed and retry count < 1:**
   - Log retry:
     ```
     bash ~/.aether/aether-utils.sh activity-log "ERROR" "{caste}-ant" "retry {N}: {error_summary}"
     ```
   - Spawn a NEW worker (same caste, same task) with failure context appended:
     ```
     Previous attempt failed because: {error_description}. Try a different approach.
     ```
   - Increment retry counter for this task

f2. **If worker retry also failed (retry count >= 1) -- Spawn Debugger Ant:**

   Log debugger spawn:
   ```
   bash ~/.aether/aether-utils.sh activity-log "SPAWN" "queen" "debugger-ant for: {task_description}"
   ```

   Read `~/.aether/workers/builder-ant.md`. Spawn debugger via Task tool with `subagent_type="general-purpose"`:

   ```
   --- WORKER SPEC ---
   {full contents of ~/.aether/workers/builder-ant.md}

   --- ACTIVE PHEROMONES ---
   {pheromone block from Step 3}

   --- TASK ---
   You are being spawned as a DEBUGGER ANT.

   A worker failed its task twice. Your job: diagnose the failure and fix it.

   Failed task: {task_description}
   Failed caste: {caste}
   First attempt error: {error_from_attempt_1}
   Second attempt error: {error_from_attempt_2}
   Files involved: {files_from_worker_report}

   Your mission:
   1. Read the files involved in the failure
   2. DIAGNOSE the root cause -- understand WHY it failed, not just WHAT failed
   3. Identify the MINIMAL patch to fix the issue
   4. Apply the fix using Write/Edit tools
   5. Run verification (syntax check, import check, tests if available)
   6. Report what you found and what you changed

   CONSTRAINTS:
   - PATCH the existing code. Do NOT rewrite from scratch.
   - Preserve the original worker's approach and intent.
   - If the failure is in a test, fix the code to pass the test (not the other way around).
   - If you cannot diagnose the issue, report UNDIAGNOSABLE with your analysis.

   Produce a structured report with:
   - diagnosis: root cause description
   - fix_applied: boolean
   - files_modified: array of paths
   - verification: pass/fail with details
   ```

g. **Post-debugger logic (Queen handles):**

   - If debugger reports `fix_applied == true`:
     - Mark task as completed
     - Log: `bash ~/.aether/aether-utils.sh activity-log "COMPLETE" "debugger-ant" "Fixed: {diagnosis}"`
     - Display using Bash tool (red -- debugger color):
       ```
       bash -c 'printf "\e[31m%-12s\e[0m %s\n" "[DEBUGGER]" "Fixed: {diagnosis}"'
       ```

   - If debugger reports `fix_applied == false` or "UNDIAGNOSABLE":
     - Infer task criticality: if the failed task directly maps to a phase success criterion, treat as critical (display warning to user); if supporting task, skip and continue
     - Log: `bash ~/.aether/aether-utils.sh activity-log "ERROR" "debugger-ant" "Could not fix: {diagnosis}"`
     - Display using Bash tool (red -- debugger color):
       ```
       bash -c 'printf "\e[31m%-12s\e[0m %s\n" "[DEBUGGER]" "Could not fix: {diagnosis}. Task {skipped|flagged for review}."'
       ```
     - Mark task as failed
     - Continue to next worker

h. **Post-Wave Conflict Check (best-effort):**
   After all workers in this wave return, read the activity log entries for this wave's workers:
     `bash ~/.aether/aether-utils.sh activity-log-read "{caste}-ant"`

   For each pair of workers in this wave, compare their CREATED and MODIFIED log entries for file path overlap.

   If two workers in the same wave show MODIFIED or CREATED entries for the SAME file path:
     HALT execution. Display:
     ```
     CONFLICT DETECTED: Workers {A} and {B} both modified {file_path}

     This wave's changes may have overwritten each other.
     Options:
       1. Review the file and continue
       2. Rollback to git checkpoint: git reset --hard {checkpoint_hash}
     ```
     Wait for user input before continuing to the next wave.

   If no conflicts detected, proceed to the next wave silently.

i. **Post-Wave Advisory Review:**

   **Mode + wave-size check:** Read COLONY_STATE.json mode field.
   - If mode is "LIGHTWEIGHT": Skip reviewer entirely. Display: "Reviewer skipped (LIGHTWEIGHT mode)." and continue to next wave.
   - If this wave had only 1 worker (single-worker waves have no cross-worker interaction to review): Skip reviewer entirely. Display: "Reviewer skipped (single-worker wave)." and continue to next wave.

   **Reviewer spawn:** Read `~/.aether/workers/watcher-ant.md`. Spawn reviewer via Task tool with `subagent_type="general-purpose"`:

   ```
   --- WORKER SPEC ---
   {full contents of ~/.aether/workers/watcher-ant.md}

   --- ACTIVE PHEROMONES ---
   {pheromone block from Step 3}

   --- TASK ---
   You are being spawned as a post-wave ADVISORY REVIEWER.

   Phase {id}: {phase_name}
   Wave {N} of {total_waves} just completed.

   Workers in this wave:
   {for each worker: caste, task, and result summary}

   Your mission:
   1. Read the files modified by workers in this wave
   2. Run Execution Verification (syntax, import, launch checks)
   3. Run Quality mode checks at minimum
   4. Produce findings with severity levels (CRITICAL, HIGH, MEDIUM, LOW)

   IMPORTANT: You are in ADVISORY mode. Your findings will be DISPLAYED to the user but will NOT block progress. Only CRITICAL severity findings will trigger a rebuild. Be concise -- this runs after every wave.

   Severity boundary definitions:
   - CRITICAL: Code does not run (syntax errors, import failures, launch crashes), security vulnerabilities, data corruption risk
   - HIGH: Tests fail, missing major requirements, breaking existing functionality
   - MEDIUM: Code quality issues, missing edge cases, convention violations
   - LOW: Style issues, minor improvements, documentation gaps

   Produce a structured report with:
   - findings: array of {severity, description, location, recommendation}
   - critical_count: number of CRITICAL findings
   - summary: one-line summary of wave quality
   ```

   **Post-reviewer logic (Queen handles):**

   - Log reviewer spawn:
     ```
     bash ~/.aether/aether-utils.sh activity-log "SPAWN" "queen" "reviewer for wave {N}"
     ```
   - After reviewer returns, log result:
     ```
     bash ~/.aether/aether-utils.sh activity-log "COMPLETE" "reviewer" "{summary}"
     ```

   - Parse `critical_count` from the reviewer's findings.

   - Display reviewer summary inline using Bash tool (blue -- reviewer color):
     ```
     bash -c 'printf "\e[34m%-12s\e[0m Wave %d: %s (%d findings)\n" "[REVIEWER]" {N} "{summary}" {finding_count}'
     ```
     For each finding, display:
     ```
     bash -c 'printf "  \e[34m%-8s\e[0m %s\n" "{severity}" "{description}"'
     ```

   - If `critical_count > 0` AND `wave_rebuild_count < 2`:
     - Display: "CRITICAL issue detected. Rebuilding wave {N}..."
     - Display urgent between-wave recommendation:
       ```
       Recommendation: The colony detected critical issues in {area from reviewer findings}. Consider pausing after this build to investigate.
       ```
       This is a single inline recommendation that appears immediately when urgent patterns emerge. It does NOT count toward the max-3 end-of-build recommendations in Step 7e.
     - Increment `wave_rebuild_count`
     - Re-run this wave's workers with findings appended to their prompts:
       ```
       Previous attempt had CRITICAL issues: {findings}. Fix these.
       ```

   - If `critical_count > 0` AND `wave_rebuild_count >= 2`:
     - Display: "CRITICAL issues persist after 2 rebuilds. Continuing to next wave."
     - Display urgent between-wave recommendation:
       ```
       Recommendation: The colony detected critical issues in {area from reviewer findings} that persist after rebuilds. Strongly consider investigating before continuing.
       ```

   - If `critical_count == 0`:
     - Display summary and continue to next wave.

j. **Fulfill SPAWN requests from this wave:**

   If `queued_sub_spawns` is empty for this wave: skip to next wave.

   For each queued sub-spawn:

   1. Record in spawn_tree: `spawn_tree[sub_id] = {id: "<phase>_sub_<caste>_<index>", caste, task, depth: 2, parent: parent_id, children: [], status: "pending", phase, wave}`. Add sub_id to parent's `children` array.
   2. Log: `bash ~/.aether/aether-utils.sh activity-log "SPAWN" "queen" "sub-{caste}-ant for: {task} (requested by {parent_id})"`
   3. Display: `bash -c 'printf "  \e[{caste_color}m%-14s\e[0m %s (sub-spawn for %s)\n" "[SUB-{CASTE}]" "{task}" "{parent_id}"'`
   4. Read the caste's spec file. Spawn via Task tool with `subagent_type="general-purpose"`:
      ```
      --- WORKER SPEC ---
      {full contents of the caste's spec file}

      --- ACTIVE PHEROMONES ---
      {pheromone block from Step 3}

      --- PARENT CONTEXT ---
      Parent worker: {parent caste} - {parent task}
      Parent pheromone context (FOCUS/REDIRECT): {inherited from parent}

      --- TASK ---
      {sub-task from SPAWN REQUEST}

      You are at depth 2. You CANNOT request further sub-spawns.
      If you need additional work done, handle it inline.
      ```
   5. After sub-worker returns: update spawn_tree status -> "completed" or "failed". Log and display completion.

   Reset `queued_sub_spawns = []` for next wave.

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

After all workers complete (Step 5c), spawn a **mandatory watcher verification**.

**Mode Check:** Read COLONY_STATE.json mode field.
If mode is "LIGHTWEIGHT": Skip watcher verification entirely. Display:
  "Watcher verification skipped (LIGHTWEIGHT mode)."
Proceed directly to Step 6.

Otherwise: Continue with mandatory watcher verification as-is.

1. Use the Read tool to read `~/.aether/workers/watcher-ant.md`
2. Use the **Task tool** with `subagent_type="general-purpose"`:

```
--- WORKER SPEC ---
{full contents of ~/.aether/workers/watcher-ant.md}

--- ACTIVE PHEROMONES ---
{pheromone block from Step 3}

--- PHASE BUILD REPORT ---
{the Phase Build Report compiled at the end of Step 5c, containing all worker results}

--- TASK ---
You are being spawned as a mandatory post-build watcher.

Phase {id}: {phase_name}

Success Criteria:
{list success_criteria from the phase}

Your mission:
1. Read the files that were modified during this phase (identified in the Phase Build Report)
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

**Record Quality Decision:** Read `.aether/data/memory.json` (if not already in memory). Append a quality decision to the `decisions` array:

```json
{
  "id": "dec_<unix_timestamp>_<4_random_hex>",
  "type": "quality",
  "content": "<watcher verdict -- e.g. 'Phase 3 approved at 7/10, deferred 2 medium issues to tech debt'>",
  "context": "Phase <id> watcher verification",
  "phase": <current_phase_number>,
  "timestamp": "<ISO-8601 UTC>"
}
```

Cap at 30 entries (remove oldest if exceeded). Write updated memory.json.

### Step 6: Record Outcome

After the watcher returns, use Write tool to update:

**`PROJECT_PLAN.json`:**
- Mark tasks as `"completed"` or `"failed"` based on worker results from Step 5c
- Set the phase `status` to `"completed"` (or `"failed"` if critical tasks failed)

**`COLONY_STATE.json`:**
- Set `state` to `"READY"`
- Advance `current_phase` if phase completed
- Set `workers.builder` to `"idle"`
- Set `workers.watcher` to `"idle"`
- Write `spawn_tree` to COLONY_STATE.json (write `"spawn_tree": {}` if empty)

**Log Errors:** If the ant reported any failures or issues in its report:

For each failure, use the Bash tool to run:
```
bash ~/.aether/aether-utils.sh error-add "<category>" "<severity>" "<description>" <phase_number>
```

Where `category` is one of: syntax, import, runtime, type, spawning, phase, verification, api, file, logic, performance, security. And `severity` is one of: critical, high, medium, low. The `<phase_number>` is the current phase ID being built (from `$ARGUMENTS`).

Each call returns `{"ok":true,"result":"<error_id>"}`. Note the returned error IDs for event logging.

**Log Watcher Issues:** For each issue in the watcher report with severity `HIGH` or `CRITICAL`, use the Bash tool to run:
```
bash ~/.aether/aether-utils.sh error-add "verification" "<severity_lowercased>" "<description>" <phase_number>
```

Where `<phase_number>` is the current phase ID being built (from `$ARGUMENTS`).

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
bash ~/.aether/aether-utils.sh error-pattern-check
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
bash ~/.aether/aether-utils.sh error-summary
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

**Record Spawn Outcomes:** The Queen has explicit knowledge of which castes were spawned during Step 5c (it did the spawning). For each caste that was spawned:

- If the phase completed successfully: increment `alpha` and `successes` for that caste in `spawn_outcomes`
- If the phase failed: increment `beta` and `failures` for that caste in `spawn_outcomes`
- Increment `total_spawns` for that caste regardless of outcome

Use the Write tool to write the updated COLONY_STATE.json (this write can be combined with the state update already in Step 6).

### Step 7: Extract Learnings, Emit Pheromones, Display Results

#### Step 7a: Extract Phase Learnings

Read `.aether/data/memory.json`. The Queen already has in memory from prior steps: worker_results (Step 5c), watcher_report (Step 5.5), errors.json and events.json (Step 6), and PROJECT_PLAN.json task outcomes (Step 6). No redundant file reads needed beyond memory.json.

Synthesize actionable learnings from worker outcomes, watcher report, errors, and events. Each learning MUST be attributed to the worker/caste that produced it (e.g., "builder-ant: bcrypt 12 rounds caused 800ms delay"). Capture both successes AND failures. Even clean phases get learnings ("X approach worked well").

**Quality guard:** Each learning must reference a specific event, error, or outcome. Generic learnings like "Phase completed successfully" are not acceptable.

Append a learning entry to memory.json `phase_learnings` array:

```json
{
  "id": "learn_<unix_timestamp>_<4_random_hex>",
  "phase": <phase_number>,
  "phase_name": "<name>",
  "learnings": ["<caste>: <specific learning>", ...],
  "errors_encountered": <count>,
  "timestamp": "<ISO-8601 UTC>"
}
```

Write the updated memory.json. Then run:
```
bash ~/.aether/aether-utils.sh memory-compress
```

If memory-compress reports `compressed:true`, note the before/after learning count for display in Step 7e so eviction is visible to the user.

Note: spawn_outcomes already updated in Step 6 -- do NOT update them here.

#### Step 7b: Emit FEEDBACK Pheromone

Read `.aether/data/pheromones.json` (if not already in memory). Always emit a FEEDBACK pheromone with balanced summary of what worked + what failed. Include the actual learnings in the pheromone body so colony can read them without checking memory.json.

```json
{
  "id": "auto_<unix_timestamp>_<4_random_hex>",
  "type": "FEEDBACK",
  "content": "<balanced summary>",
  "strength": 0.5,
  "half_life_seconds": 21600,
  "created_at": "<ISO-8601 UTC>",
  "source": "auto:build",
  "auto": true
}
```

Validate via: `bash ~/.aether/aether-utils.sh pheromone-validate "<content>"`
- If pass:false -> skip pheromone, log `pheromone_rejected` event
- If pass:true -> append to pheromones.json
- If command fails -> append anyway (fail-open)

Conditionally emit a REDIRECT pheromone if `errors.json` has `flagged_patterns` entries related to this phase (same logic as continue.md Step 4.5). Use `source: "auto:build"`.

Log `pheromone_auto_emitted` event for each emitted pheromone (source: "build").

#### Step 7c: Clean Expired Pheromones

Run: `bash ~/.aether/aether-utils.sh pheromone-cleanup`

#### Step 7d: Write Auto-Learning Flag Event

Append to events.json:

```json
{
  "id": "evt_<unix_timestamp>_<4_random_hex>",
  "type": "auto_learnings_extracted",
  "source": "build",
  "content": "Auto-extracted <N> learnings from Phase <id>: <name>",
  "timestamp": "<ISO-8601 UTC>"
}
```

If events array exceeds 100 entries, trim oldest to 100. Write updated events.json and pheromones.json.

#### Step 7e: Display Results

Display the final build summary header using Bash tool (bold yellow -- Queen color):
```
bash -c 'printf "\n\e[1;33m+=====================================================+\e[0m\n"'
bash -c 'printf "\e[1;33m|  BUILD COMPLETE                                     |\e[0m\n"'
bash -c 'printf "\e[1;33m+=====================================================+\e[0m\n\n"'
```

Show step progress:

```
  ✓ Step 1: Validate
  ✓ Step 2: Read State
  ✓ Step 3: Compute Active Pheromones
  ✓ Step 4: Update State
  ✓ Step 4.5: Git Checkpoint
  ✓ Step 5a: Phase Lead Planning
  ✓ Step 5b: Plan Checkpoint
  ✓ Step 5c: Execute Workers
  ✓ Step 5c.j: Fulfill Sub-Spawns
  ✓ Step 5c.i: Post-Wave Review
  ✓ Step 5.5: Watcher Verification
  ✓ Step 6: Record Outcome
  ✓ Step 7a: Extract Phase Learnings
  ✓ Step 7b: Emit Pheromones
  ✓ Step 7c: Clean Expired Pheromones
  ✓ Step 7d: Write Events
  ✓ Step 7e: Display Results
```

Then display:

```
---

Phase {id}: {name}

🔒 Git Checkpoint: {commit_hash or "(not a git repo)"}

🐜 Colony Activity:
  {Per-worker results from Step 5c -- for each worker: caste, task, result}
  Activity Log: .aether/data/activity.log ({line_count} entries)
```

Display the Delegation Tree using Bash tool (bold yellow):
```
bash -c 'printf "\n\e[1;33mDelegation Tree:\e[0m\n"'
bash -c 'printf "  \e[1;33mQueen\e[0m\n"'
```
For each spawn_tree entry where depth == 1:
`bash -c 'printf "  ├── \e[{caste_color}m%s\e[0m: %s [\e[32m%s\e[0m]\n" "{caste}" "{task}" "{status}"'`
For each child of that entry (depth == 2):
`bash -c 'printf "  │   └── \e[{caste_color}m%s\e[0m (sub): %s [\e[32m%s\e[0m]\n" "{caste}" "{task}" "{status}"'`
If spawn_tree is empty: `bash -c 'printf "  (no delegation -- all tasks handled directly)\n"'`

```
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

📚 Learnings Extracted:
  - <learning 1>
  - <learning 2>
  {if memory-compress trimmed: "⚠️ Memory compressed: <before> -> <after> learnings (oldest evicted)"}

🧪 Auto-Emitted Pheromones:
  FEEDBACK (0.5, 6h): "<first 80 chars>"
  {if REDIRECT emitted:}
  REDIRECT (0.9, 24h): "<first 80 chars>"

Display the Pheromone Recommendations header using Bash tool (yellow):
  ```
  bash -c 'printf "\e[33mPheromone Recommendations:\e[0m\n"'
  ```

  Based on this build's outcomes, analyze the following sources and generate max 3 natural language recommendations:

  Sources:
  - worker_results from Step 5c (which tasks succeeded/failed, error summaries)
  - watcher_report from Step 5.5 (quality score, issues found)
  - errors.json flagged_patterns (recurring cross-phase issues)
  - Per-wave reviewer findings from Step 5c.i (if reviewer ran)

  Trigger patterns (recommend when these signals are detected):
  - Repeated test failures in same module -> suggest focusing colony on stabilizing that module
  - Quality score < 6 -> suggest redirect signal to avoid specific anti-pattern found
  - Multiple workers touching related files -> suggest focus on consistency in that layer
  - Clean build, high quality -> note strong patterns worth continuing
  - Flagged error patterns from errors.json -> suggest redirect to avoid recurring pattern

  Format constraints:
  - Natural language descriptive guidance, NOT copy-paste commands
  - Must reference SPECIFIC observations from THIS build
  - Must suggest a DIRECTION, not a specific action
  - Maximum 3 suggestions, force prioritization
  - Must sound like a senior engineer's observations, NOT automated alerts
  - Must NOT start with "Run:" or "/ant:" or be formatted as commands

  Display format:
  ```
  Pheromone Recommendations:
    Based on this build's outcomes:

    1. {natural language recommendation}
       Signal: {specific observation that triggered this}

    2. {natural language recommendation}
       Signal: {specific observation}

    These are suggestions, not commands. Use /ant:focus or /ant:redirect to act on them.
  ```

  If no meaningful patterns emerge from the build data:
  ```
    No specific recommendations -- build was clean.
  ```

Next:
  /ant:continue            Advance to next phase
  /ant:feedback "<note>"   Give feedback first
  /ant:status              View full colony status
```

### Step 7f: Persistence Confirmation

After displaying the "Next:" block above, run state validation.

Use the Bash tool to run: `bash ~/.aether/aether-utils.sh validate-state all`

If the result contains `"pass":true`:

```
---
All state persisted. Safe to /clear context if needed.
  State: .aether/data/ (6 files validated)
  Resume: /ant:resume-colony
```

If the result contains `"pass":false`:

```
---
WARNING: State validation issue detected. Check /ant:status before clearing.
```
