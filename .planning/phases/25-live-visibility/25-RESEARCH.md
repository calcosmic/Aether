# Phase 25: Live Visibility - Research

**Researched:** 2026-02-04
**Domain:** Claude Code multi-agent orchestration, file-based activity logging, incremental display
**Confidence:** HIGH

## Summary

Phase 25 restructures how the Aether colony builds phases. Today, `build.md` spawns a single Phase Lead via the Task tool, which internally spawns all workers and returns a monolithic report. The user sees nothing until everything completes. This phase splits that into two roles: the Phase Lead becomes a **planner** (returns an ordered task assignment plan), and the Queen (build.md itself) becomes the **executor** (spawns workers one-by-one, displaying results after each).

The core constraint driving this design is that the Claude Code Task tool does not support streaming -- a subagent's output is invisible until it returns. The only way to show incremental progress is to have the Queen (the top-level agent) spawn workers sequentially, displaying results between spawns. This is already acknowledged in the REQUIREMENTS.md out-of-scope section.

The activity log (`activity.log`) provides a secondary visibility channel -- workers write structured lines as they work, and the Queen reads these after each worker returns to show what happened. Since workers execute as Task tool subagents (which ARE the same process, not separate OS processes), file writes from workers ARE visible to the Queen after the Task returns.

**Primary recommendation:** Restructure build.md Step 5 into three sub-steps: (5a) spawn Phase Lead for planning only, (5b) display plan and get user checkpoint, (5c) execute plan by spawning workers sequentially with activity log display after each. Add `activity-log` subcommand to aether-utils.sh for structured log writes.

## Standard Stack

### Core

| Component | Current | Purpose | Why Standard |
|-----------|---------|---------|--------------|
| `build.md` | Step 5 spawns Phase Lead | Queen-level build orchestration | Only file that controls the build flow |
| `aether-utils.sh` | 229 lines, 13 subcommands | Deterministic shell operations | All deterministic ops go through this |
| Worker spec files | 6 files in `.aether/workers/` | Worker behavior definitions | Already have progress output format |
| `.aether/data/` | JSON state files | Persistent colony state | Established data directory |

### Supporting

| Component | Purpose | When to Use |
|-----------|---------|-------------|
| `file-lock.sh` | Prevent concurrent file access | When multiple workers could write to activity.log simultaneously |
| `atomic-write.sh` | Corruption-safe file writes | For JSON state file updates (not needed for append-only log) |
| Task tool | Spawn subagents | Every worker spawn |

### No Alternatives Needed

This phase modifies existing infrastructure. No new libraries or external dependencies required. The entire system is markdown prompts + shell utilities.

## Architecture Patterns

### Current Architecture (What Exists Today)

```
build.md (Queen)
  Step 1-4: Validate, read state, compute pheromones, update state
  Step 4.5: Git checkpoint
  Step 5: Spawn Phase Lead (Task tool)
    Phase Lead internally:
      - Reads tasks, plans work
      - Spawns scouts, builders (2-4 workers)
      - Waits for each, compiles results
      - Returns monolithic report
  Step 5.5: Spawn Watcher (Task tool)
  Step 6: Record outcome
  Step 7: Display results
```

**Problem:** User sees nothing during Step 5 (which can take minutes). The Phase Lead report is the ONLY record of what happened.

### New Architecture (What Phase 25 Creates)

```
build.md (Queen)
  Step 1-4: Validate, read state, compute pheromones, update state
  Step 4.5: Git checkpoint
  Step 5a: Spawn Phase Lead as PLANNER (Task tool)
    Phase Lead:
      - Reads tasks, identifies dependencies
      - Produces ordered task assignment plan with wave groupings
      - Does NOT spawn any workers
      - Returns plan as structured output
  Step 5b: Plan Checkpoint
    - Queen displays the plan to user
    - Asks "Proceed with this plan?"
    - If rejected: re-runs Phase Lead with feedback
  Step 5c: Execute Plan (loop)
    - Clear activity log
    - Display wave header
    - For each worker in current wave:
      - Announce worker spawn
      - Spawn worker (Task tool)
      - Read activity log entries for this worker
      - Display condensed summary
      - Update progress bar
    - If worker fails: retry with failure context (max 2)
    - Advance to next wave
  Step 5.5: Spawn Watcher (unchanged)
  Step 6: Record outcome (unchanged)
  Step 7: Display results (updated with activity log integration)
```

### Phase Lead Plan Format

The Phase Lead returns a structured plan, NOT spawned workers. Format from CONTEXT.md decisions:

```
TASK ASSIGNMENT PLAN
====================

Wave 1 (independent):
  1. scout-ant: research auth middleware patterns (tasks 1, 2)
  2. builder-ant: implement utility helpers (task 6)

Wave 2 (depends on Wave 1):
  3. builder-ant: implement auth middleware (tasks 3, 5)
     Needs: scout results from #1

Wave 3 (depends on Wave 2):
  4. builder-ant: implement route guards (task 4)
     Needs: auth middleware from #3
```

### Activity Log Format

```
# Phase 25: Live Visibility — 2026-02-04T14:23:00Z
# Workers: 4 planned

[14:23:01] START scout-ant: research auth middleware patterns
[14:23:05] RESEARCH scout-ant: found 3 middleware patterns in codebase
[14:23:08] CREATED scout-ant: .aether/temp/scout-findings.md (42 lines)
[14:23:10] COMPLETE scout-ant: research auth middleware patterns

[14:23:11] START builder-ant: implement utility helpers
[14:23:15] CREATED builder-ant: src/utils/auth-helpers.ts (67 lines)
[14:23:18] MODIFIED builder-ant: src/utils/index.ts
[14:23:20] COMPLETE builder-ant: implement utility helpers

[14:23:21] SPAWN builder-ant: builder-ant -> scout-ant for: check JWT library docs
[14:23:25] ERROR builder-ant: type error in auth-helpers.ts — fixed inline

[14:23:30] START builder-ant: implement auth middleware
[14:23:35] CREATED builder-ant: src/middleware/auth.ts (120 lines)
[14:23:38] MODIFIED builder-ant: src/routes/index.ts
[14:23:40] COMPLETE builder-ant: implement auth middleware
```

Each line format: `[HH:MM:SS] ACTION caste-name: description`

Actions: START, COMPLETE, ERROR, CREATED, MODIFIED, RESEARCH, SPAWN

### Worker Activity Log Write Mechanism

**Recommendation:** Add an `activity-log` subcommand to `aether-utils.sh` that appends structured lines. This avoids workers needing to know the log file path or format.

```bash
# Worker calls:
bash .aether/aether-utils.sh activity-log "START" "builder-ant" "implement auth middleware"
bash .aether/aether-utils.sh activity-log "CREATED" "builder-ant" "src/middleware/auth.ts (120 lines)"
bash .aether/aether-utils.sh activity-log "COMPLETE" "builder-ant" "implement auth middleware"
bash .aether/aether-utils.sh activity-log "ERROR" "builder-ant" "type error in auth-helpers.ts"
```

This keeps the log format controlled by the utility layer (consistent with the project's architecture principle: "code handles deterministic ops, prompts handle reasoning").

### Queen Display After Each Worker

After each worker returns, the Queen:
1. Reads new activity log entries (since last read position)
2. Displays condensed summary:

```
--- Worker 1/4 ---
scout-ant: research auth middleware patterns
  [14:23:01 - 14:23:10] 9s
  Researched: 3 middleware patterns
  Created: .aether/temp/scout-findings.md
  Result: COMPLETE

  ██████████░░░░░░░░░░ 1/4 workers complete
```

### Progress Bar Rendering

```
  ██████████░░░░░░░░░░ 2/6 workers complete
```

Computation: `filled = round(completed / total * 20)`, use `█` for filled and `░` for empty. Total bar width: 20 characters.

### Wave Boundary Display

```
─── Wave 1/3 ───
  Spawning scout-ant for: research auth middleware patterns...
  ...
  Spawning builder-ant for: implement utility helpers...
  ...

─── Wave 2/3 ───
  Spawning builder-ant for: implement auth middleware...
  ...
```

### Anti-Patterns to Avoid

- **Do NOT run workers in parallel within build.md:** The whole point is sequential spawning for visibility. Waves group independent tasks, but the Queen still spawns one-at-a-time within a wave to maintain display clarity. (Parallel execution within a wave is a future optimization if needed.)
- **Do NOT have the Phase Lead spawn workers:** This defeats the purpose. The Phase Lead ONLY produces a plan.
- **Do NOT use run_in_background for worker spawns:** Background tasks cannot have their results displayed incrementally.
- **Do NOT stream the activity log in real-time:** The Task tool blocks until the worker returns. Read the log AFTER return.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| File append with timestamps | Custom bash in each worker prompt | `aether-utils.sh activity-log` subcommand | Consistent format, single source of truth, handles rotation |
| Log rotation | Manual file management in build.md | `aether-utils.sh activity-log-rotate` or inline in activity-log init | Rotation per phase is a decision from CONTEXT.md |
| Concurrent file writes | Hope for the best | File locking via existing `file-lock.sh` for activity.log | Workers within a wave could theoretically overlap if future changes add parallelism |
| Progress bar calculation | Inline math in build.md prompt | Keep inline (it's a single division) | Too simple to justify a subcommand |

**Key insight:** The activity log is an append-only text file, not JSON. This is deliberately simpler than the JSON state files. No need for atomic writes or JSON validation -- just append lines. The existing `file-lock.sh` can guard concurrent access if ever needed.

## Common Pitfalls

### Pitfall 1: Phase Lead Still Trying to Spawn Workers

**What goes wrong:** The Phase Lead prompt currently says "Spawn 🔨🐜 builder-ants with specific tasks." If this wording remains, the Phase Lead will spawn workers instead of just planning.
**Why it happens:** The current Phase Lead prompt is heavily oriented toward delegation and spawning.
**How to avoid:** Completely rewrite the Phase Lead prompt section to emphasize PLANNING ONLY. Use explicit wording: "You MUST NOT use the Task tool. You MUST NOT spawn any workers. Your ONLY job is to produce a task assignment plan."
**Warning signs:** Phase Lead report contains "spawned builder-ant" or "delegated to" language.

### Pitfall 2: Activity Log Not Written by Workers

**What goes wrong:** Workers are told to write to the activity log but don't actually do it consistently because it's just prompt instructions.
**Why it happens:** Workers have many instructions. Activity log writes are easy to skip when the worker is focused on its actual task.
**How to avoid:** Make activity log writes part of the worker's mandatory Post-Action Validation checklist. Add it to all 6 worker specs. Also: the START and COMPLETE writes could be handled by the Queen (wrapping each Task tool call), not the worker itself -- this guarantees at least the boundaries are logged.
**Warning signs:** Activity log is empty or has only START/COMPLETE with no intermediate entries.

### Pitfall 3: build.md Prompt Gets Too Long

**What goes wrong:** build.md is already 512 lines. Adding plan checkpoint, wave execution loop, activity log reading, progress bar, failure retry logic, etc. could push it past effective prompt length.
**Why it happens:** All orchestration logic lives in a single markdown prompt.
**How to avoid:** Keep the new steps concise. Use structured templates, not verbose explanations. Consider extracting the worker execution loop into a clear, repeatable pattern rather than writing out every case.
**Warning signs:** build.md exceeds ~700 lines. Workers start ignoring instructions at the end of the prompt.

### Pitfall 4: Plan Checkpoint Blocking Autonomy

**What goes wrong:** The user checkpoint ("Proceed with this plan?") could break the flow if the user doesn't respond or if the system expects autonomous execution.
**Why it happens:** Claude Code commands are interactive -- the Queen CAN ask the user questions and wait for responses.
**How to avoid:** This is actually fine in Claude Code. The command prompt naturally pauses for user input. The key decision from CONTEXT.md is clear: the plan checkpoint IS the control point, and execution afterward is autonomous.
**Warning signs:** None expected -- this is a feature, not a bug.

### Pitfall 5: Activity Log Path Mismatch

**What goes wrong:** Workers write to a different path than the Queen reads from, or workers write to stdout (their normal output) instead of the file.
**Why it happens:** Workers run as Task tool subagents with their own context. They need explicit instructions about the file path.
**How to avoid:** Use the `aether-utils.sh activity-log` subcommand so the path is centralized. Workers call the subcommand, not direct file writes.
**Warning signs:** Activity log is empty after workers complete. Worker reports contain progress markers but the file doesn't.

### Pitfall 6: Retry Logic Creating Infinite Loops

**What goes wrong:** A worker fails, gets retried with failure context, but fails again for the same reason. The retry spawns another retry.
**Why it happens:** Max 2 retries per task (from CONTEXT.md), but the logic needs clean counting.
**How to avoid:** Track retry count explicitly per task in the execution loop. After 2 failures, escalate to user with the error context. Do NOT retry the retry.
**Warning signs:** More than 3 Task tool calls for the same task.

## Code Examples

### Activity Log Subcommand for aether-utils.sh

```bash
activity-log)
  action="${1:-}"
  caste="${2:-}"
  description="${3:-}"
  [[ -z "$action" || -z "$caste" || -z "$description" ]] && json_err "Usage: activity-log <action> <caste> <description>"

  log_file="$DATA_DIR/activity.log"
  ts=$(date -u +"%H:%M:%S")

  # Append line (no locking needed for single-writer sequential model)
  echo "[$ts] $action $caste: $description" >> "$log_file"
  json_ok '"logged"'
  ;;
```

### Activity Log Init/Clear Subcommand

```bash
activity-log-init)
  phase_num="${1:-}"
  phase_name="${2:-}"
  [[ -z "$phase_num" ]] && json_err "Usage: activity-log-init <phase_num> [phase_name]"

  log_file="$DATA_DIR/activity.log"
  archive_file="$DATA_DIR/activity-phase-${phase_num}.log"

  # Archive previous log if it exists
  if [ -f "$log_file" ] && [ -s "$log_file" ]; then
    mv "$log_file" "$archive_file"
  fi

  # Write phase header
  ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
  echo "# Phase $phase_num: $phase_name -- $ts" > "$log_file"
  json_ok "{\"archived\":$([ -f \"$archive_file\" ] && echo 'true' || echo 'false')}"
  ;;
```

### Activity Log Read (for Queen to display)

```bash
activity-log-read)
  caste_filter="${1:-}"
  log_file="$DATA_DIR/activity.log"
  [[ -f "$log_file" ]] || json_err "activity.log not found"

  if [ -n "$caste_filter" ]; then
    # Filter to specific caste
    content=$(grep "$caste_filter" "$log_file" | tail -20)
  else
    content=$(cat "$log_file")
  fi

  # Return as JSON-escaped string
  json_ok "$(echo "$content" | jq -Rs '.')"
  ;;
```

### Phase Lead Prompt (Planning Only -- replaces current Step 5)

```markdown
You are the Phase Lead Planner in the Aether Queen Ant Colony.

You are NOT an executor. You MUST NOT use the Task tool. You MUST NOT spawn any workers.

Your ONLY job: produce a task assignment plan that the Queen will execute.

--- COLONY CONTEXT ---
{goal, phase, tasks, pheromones -- same as current}

--- YOUR MISSION ---

Analyze the tasks and produce a TASK ASSIGNMENT PLAN:

1. Read the task list and identify dependencies
2. Group independent tasks into waves (tasks with no unmet dependencies can run in the same wave)
3. For each task, assign the appropriate worker caste
4. Order tasks within waves by priority
5. Note what context each worker needs from previous workers

Output format:

  Phase Lead Task Assignment Plan
  ================================

  Wave 1 (independent):
    1. {caste-emoji} {caste}-ant: {task description} (tasks {ids})
    2. {caste-emoji} {caste}-ant: {task description} (tasks {ids})

  Wave 2 (depends on Wave 1):
    3. {caste-emoji} {caste}-ant: {task description} (tasks {ids})
       Needs: {what from Wave 1}

  Worker count: {N}
  Wave count: {N}
```

### Queen Worker Execution Loop (build.md Step 5c)

```markdown
### Step 5c: Execute Plan

Clear the activity log:
  bash .aether/aether-utils.sh activity-log-init {phase_number} "{phase_name}"

For each wave in the plan:

  Display: `--- Wave {N}/{total} ---`

  For each worker assignment in this wave:

    1. Display: `Spawning {caste_emoji} {caste}-ant for: {task_description}...`

    2. Write to activity log:
       bash .aether/aether-utils.sh activity-log "START" "{caste}-ant" "{task_description}"

    3. Read the worker's spec file (.aether/workers/{caste}-ant.md)

    4. Spawn the worker via Task tool with:
       - Full worker spec
       - Active pheromones with effective signals for this caste
       - Task details and context from previous workers (if applicable)
       - Instruction to write progress to activity log via aether-utils.sh

    5. After worker returns:
       a. Write: bash .aether/aether-utils.sh activity-log "COMPLETE" "{caste}-ant" "{task_description}"
          (or "ERROR" if worker reported failure)
       b. Read activity log entries for this worker:
          bash .aether/aether-utils.sh activity-log-read "{caste}-ant"
       c. Display condensed summary:
          {caste_emoji} {caste}-ant: {task_description}
            Result: {COMPLETE/ERROR}
            Files: {created/modified count}
            {if error: error details}
       d. Update progress bar:
          {progress_bar} {completed}/{total} workers complete

    6. If worker failed and retries < 2:
       Spawn new worker with failure context
       Include: "Previous attempt failed because: {error}. Try a different approach."
       Increment retry counter

    7. If worker failed and retries >= 2:
       Display: "Task failed after 2 retries. Continuing with remaining tasks."
       Mark task as failed, continue to next worker

  Store all worker results for use in Steps 5.5-7.
```

### Worker Spec Addition (activity log instructions)

Add to each worker spec's Workflow section:

```markdown
## Activity Log (Mandatory)

Write progress to the activity log as you work. Use the Bash tool to run:

  bash .aether/aether-utils.sh activity-log "ACTION" "{your-caste}-ant" "description"

Actions to log:
- START: When beginning a task (Queen handles this — you don't need to)
- CREATED: When creating a new file — include path and line count
- MODIFIED: When modifying an existing file — include path
- RESEARCH: When finding useful information
- SPAWN: When spawning a sub-ant
- ERROR: When encountering an error — include brief description
- COMPLETE: When finishing a task (Queen handles this — you don't need to)

Log intermediate actions as you work. The Queen reads these after you return.
```

## State of the Art

| Old Approach (v4.2) | New Approach (v4.3 Phase 25) | Impact |
|---------------------|------------------------------|--------|
| Phase Lead spawns all workers internally | Phase Lead plans, Queen spawns workers | User sees incremental progress |
| Single Task tool call for entire phase | Multiple Task tool calls (one per worker) | Each worker visible as it completes |
| No activity log | `.aether/data/activity.log` | Persistent record of worker actions |
| No user checkpoint before execution | Plan displayed, user confirms | User controls what happens |
| Monolithic Phase Lead report at end | Per-worker summaries as they complete | "Colony feels alive" per user request |

**Key architectural shift:** This phase moves execution control from depth 1 (Phase Lead) to depth 0 (Queen/build.md). The Phase Lead drops from "coordinator + spawner" to "planner only." This is a significant restructuring of the colony's execution model.

**Risk:** More Task tool calls means more total tokens and latency (each call has overhead). A phase with 4 workers now requires 5 Task calls (1 planner + 4 workers) instead of 1 (Phase Lead that internally spawns 4). However, the visibility benefit outweighs the cost.

## Open Questions

1. **Wave parallelism now or later?**
   - What we know: CONTEXT.md says "wave-based execution: group independent tasks into waves, run each wave in parallel." But running workers in parallel defeats the incremental display purpose.
   - What's unclear: Does "parallel" mean truly concurrent Task tool calls, or sequential within a wave but all results shown at the wave boundary?
   - Recommendation: Run workers SEQUENTIALLY even within waves. Show results after each worker. The wave grouping serves as a dependency barrier, not a parallelism mechanism. This can be enhanced later.

2. **How does the Watcher get context from individually-spawned workers?**
   - What we know: Currently the Watcher gets the Phase Lead's full report. With workers spawned individually, the Queen needs to compile worker results into a coherent report for the Watcher.
   - What's unclear: Exact format for passing accumulated worker results to the Watcher.
   - Recommendation: Queen accumulates worker results in a structured block during Step 5c, passes this as "Phase Build Report" to the Watcher in Step 5.5 (replacing the Phase Lead report reference).

3. **Phase Lead depth and spawn limits**
   - What we know: Phase Lead currently runs at depth 1 and can spawn workers at depth 2. If Phase Lead is now PLANNING only, it doesn't need to spawn anything.
   - What's unclear: Should the Phase Lead still be allowed to spawn scouts for research? Or should all research happen as separate Queen-spawned workers?
   - Recommendation: Phase Lead is planning-only. If research is needed, include a scout as a Wave 1 worker in the plan. This keeps the model clean.

4. **Activity log size management**
   - What we know: Rotate per phase (CONTEXT.md decision). Archive as `activity-phase-{N}.log`.
   - What's unclear: How many archived logs to keep. When to clean up old archives.
   - Recommendation: Keep last 5 archives. Cleanup is a future concern -- not needed for Phase 25 scope.

## Sources

### Primary (HIGH confidence)

- **build.md** (`/Users/callumcowie/repos/Aether/.claude/commands/ant/build.md`) -- Full current build flow, 512 lines
- **aether-utils.sh** (`/Users/callumcowie/repos/Aether/.aether/aether-utils.sh`) -- Current utility layer, 229 lines, 13 subcommands
- **All 6 worker specs** (`/Users/callumcowie/repos/Aether/.aether/workers/*.md`) -- Current progress output format, workflow, spawn mechanics
- **file-lock.sh** (`/Users/callumcowie/repos/Aether/.aether/utils/file-lock.sh`) -- Existing file locking infrastructure
- **atomic-write.sh** (`/Users/callumcowie/repos/Aether/.aether/utils/atomic-write.sh`) -- Existing atomic write infrastructure
- **25-CONTEXT.md** (`/Users/callumcowie/repos/Aether/.planning/phases/25-live-visibility/25-CONTEXT.md`) -- User decisions from discussion
- **HANDOFF.md** (`/Users/callumcowie/repos/Aether/.aether/HANDOFF.md`) -- Historical context on delegation protocol, progress output
- **PROJECT.md** (`/Users/callumcowie/repos/Aether/.planning/PROJECT.md`) -- Version history, validated requirements
- **ROADMAP.md** (`/Users/callumcowie/repos/Aether/.planning/ROADMAP.md`) -- v4.3 milestone scope
- **REQUIREMENTS.md** (`/Users/callumcowie/repos/Aether/.planning/REQUIREMENTS.md`) -- VIS-01, VIS-02, VIS-03 definitions

### Secondary (MEDIUM confidence)

- **QUEEN_ANT_ARCHITECTURE.md** -- Original architecture document, some patterns have evolved since writing

### Tertiary (LOW confidence)

None -- this phase is entirely about internal system restructuring. No external libraries or APIs involved.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all components are existing codebase files thoroughly reviewed
- Architecture: HIGH -- the restructuring is well-defined by CONTEXT.md decisions and the current codebase structure
- Pitfalls: HIGH -- identified from direct analysis of current system behavior and prompt engineering patterns
- Code examples: MEDIUM -- examples are illustrative; exact implementation may need tuning during planning

**Research date:** 2026-02-04
**Valid until:** Indefinite (internal system, no external dependencies to go stale)
