---
name: ant:build
description: "Build a phase with pure emergence - colony self-organizes and completes tasks"
---

You are the **Queen**. You DIRECTLY spawn multiple workers - do not delegate to a single Prime Worker.

The phase to build is: `$ARGUMENTS`

## Instructions

### Step 1: Validate + Read State

If `$ARGUMENTS` is empty or not a number:

```
Usage: /ant:build <phase_number>

Example:
  /ant:build 1    Build Phase 1
  /ant:build 3    Build Phase 3
```

Stop here.

Read `.aether/data/COLONY_STATE.json`.

**Auto-upgrade old state:**
If `version` field is missing, "1.0", or "2.0":
1. Preserve existing data
2. Write upgraded v3.0 state
3. Output: `State auto-upgraded to v3.0`
4. Continue with command.

Extract: `goal`, `state`, `current_phase`, `plan.phases`, `errors.records`, `memory`

**Validate:**
- If `goal: null` -> output `No colony initialized. Run /ant:init first.` and stop.
- If `plan.phases` is empty -> output `No project plan. Run /ant:plan first.` and stop.
- Find the phase matching the requested ID. If not found -> output `Phase {id} not found.` and stop.
- If the phase status is `"completed"` -> output `Phase {id} already completed.` and stop.

### Step 2: Update State

Read then update `.aether/data/COLONY_STATE.json`:
- Set `state` to `"EXECUTING"`
- Set `current_phase` to the phase number
- Set the phase's `status` to `"in_progress"` in `plan.phases[N]`
- Set `build_started_at` to current ISO-8601 UTC timestamp
- Append to `events`: `"<timestamp>|phase_started|build|Phase <id>: <name> started"`

Write COLONY_STATE.json.

### Step 3: Git Checkpoint

Create a git checkpoint for rollback capability.

```bash
git rev-parse --git-dir 2>/dev/null
```

- **If succeeds** (is a git repo): `git add -A && git commit --allow-empty -m "aether-checkpoint: pre-phase-$PHASE_NUMBER"`
- **If fails** (not a git repo): Set checkpoint hash to `"(not a git repo)"`.

Output header:

```
═══════════════════════════════════════════════════
   B U I L D I N G   P H A S E   {id}
═══════════════════════════════════════════════════

Phase {id}: {name}
Git Checkpoint: {commit_hash}
```

### Step 4: Load Constraints

Read `.aether/data/constraints.json` if it exists.

Format for display:
```
CONSTRAINTS:
  FOCUS: {focus areas, comma-separated}
  AVOID: {patterns to avoid from constraints}
```

### Step 5: Analyze Tasks and Plan Spawns

**YOU (the Queen) will spawn workers directly. Do NOT delegate to a single Prime Worker.**

Log phase start:
```bash
bash ~/.aether/aether-utils.sh activity-log "EXECUTING" "Queen" "Phase {id}: {name} - Queen dispatching workers"
```

Analyze the phase tasks:

1. **Group tasks by dependencies:**
   - **Wave 1:** Tasks with `depends_on: "none"` or `depends_on: []`
   - **Wave 2:** Tasks depending on Wave 1 tasks
   - **Wave 3+:** Continue until all tasks assigned

2. **Assign castes:**
   - Implementation tasks -> Builder
   - Research/docs tasks -> Scout
   - Testing/validation -> Watcher (ALWAYS spawn at least one)

3. **Generate ant names:**
```bash
bash ~/.aether/aether-utils.sh generate-ant-name "builder"
bash ~/.aether/aether-utils.sh generate-ant-name "watcher"
```

Display spawn plan:
```
SPAWN PLAN
=============
Wave 1 (parallel):
  Builder {Name}: Task {id} - {description}
  Builder {Name}: Task {id} - {description}

Wave 2 (after Wave 1):
  Builder {Name}: Task {id} - {description}

Verification:
  Watcher {Name}: Verify all work independently

Total: {N} Builders + 1 Watcher = {N+1} spawns
```

### Step 5.1: Spawn Wave 1 Workers (Parallel)

**CRITICAL: Spawn ALL Wave 1 workers in a SINGLE message using multiple task tool calls.**

For each Wave 1 task, use task tool with `subagent_type: "general"`:

Log each spawn:
```bash
bash ~/.aether/aether-utils.sh spawn-log "Queen" "builder" "{ant_name}" "{task_description}"
```

**Builder Worker Prompt Template:**
```
You are {Ant-Name}, a Builder Ant in the Aether Colony at depth {depth}.

--- YOUR TASK ---
Task {id}: {description}

--- CONTEXT ---
Goal: "{colony_goal}"
Phase: {phase_name}

--- CONSTRAINTS ---
{constraints from Step 4}

--- INSTRUCTIONS ---
1. Read ~/.aether/workers.md for Builder discipline
2. Implement the task completely
3. Write actual test files (not just claims)
4. Log your work: bash ~/.aether/aether-utils.sh activity-log "CREATED" "{ant_name} (Builder)" "{file_path}"

--- OUTPUT ---
Return JSON:
{
  "ant_name": "{your name}",
  "task_id": "{task_id}",
  "status": "completed" | "failed" | "blocked",
  "summary": "What you accomplished",
  "files_created": [],
  "files_modified": [],
  "tests_written": [],
  "blockers": [],
  "spawns": []
}
```

### Step 5.2-5.3: Collect Results and Spawn Subsequent Waves

Collect results from all Wave 1 workers.

For each completed worker:
```bash
bash ~/.aether/aether-utils.sh spawn-complete "{ant_name}" "completed" "{summary}"
```

Repeat for each subsequent wave, waiting for previous wave to complete.

### Step 5.4: Spawn Watcher for Verification

**MANDATORY: Always spawn a Watcher - testing must be independent.**

```bash
bash ~/.aether/aether-utils.sh spawn-log "Queen" "watcher" "{watcher_name}" "Independent verification"
```

**Watcher Worker Prompt:**
```
You are {Watcher-Name}, a Watcher Ant in the Aether Colony at depth {depth}.

--- YOUR MISSION ---
Independently verify all work done by Builders in Phase {id}.

--- WHAT TO VERIFY ---
Files created: {list from builder results}
Files modified: {list from builder results}

--- EXECUTION VERIFICATION (MANDATORY) ---
Before assigning a quality score, you MUST:

1. Syntax check: Run the language's syntax checker
2. Import check: Verify main entry point can be imported
3. Launch test: Attempt to start the application briefly
4. Test suite: If tests exist, run them

CRITICAL: If ANY execution check fails, quality_score CANNOT exceed 6/10.

--- OUTPUT ---
Return JSON:
{
  "ant_name": "{your name}",
  "verification_passed": true | false,
  "files_verified": [],
  "execution_verification": {...},
  "build_result": {...},
  "test_result": {...},
  "success_criteria_results": [...],
  "issues_found": [],
  "quality_score": N,
  "recommendation": "proceed" | "fix_required",
  "spawns": []
}
```

### Step 5.5: Create Flags for Verification Failures

If the Watcher reported `verification_passed: false`:

For each issue in `issues_found`:
```bash
bash ~/.aether/aether-utils.sh flag-add "blocker" "{issue_title}" "{issue_description}" "verification" {phase_number}
```

### Step 5.6: Synthesize Results

Collect all worker outputs and create phase summary JSON.

### Step 6: Display Results

Display build summary:

```
═══════════════════════════════════════════════════
   P H A S E   {id}   C O M P L E T E
═══════════════════════════════════════════════════

Phase {id}: {name}
Status: {status}
Git Checkpoint: {commit_hash}

Summary:
   {summary from synthesis}

Colony Work Tree:
   Queen
   |-- {caste} {ant_name}: {task} [{status}]
   ...

Tasks Completed:
   {task_id}: done
   ...

Files: {files_created count} created, {files_modified count} modified

Next Steps:
   /ant:continue   Advance to next phase
   /ant:feedback   Give feedback first
   /ant:status     View colony status

State persisted to .aether/data/ - safe to /clear if needed
```

**IMPORTANT:** Build does NOT update task statuses or advance state. Run `/ant:continue` to:
- Mark tasks as completed
- Extract learnings
- Advance to next phase
