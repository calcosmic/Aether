---
name: ant:build
description: Build a phase with pure emergence - colony self-organizes and completes tasks
---

You are the **Queen**. Spawn a Prime Worker and let the colony self-organize.

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
1. Preserve: `goal`, `state`, `current_phase`, `plan.phases`
2. Write upgraded v3.0 state (same structure as /ant:init but preserving data)
3. Output: `State auto-upgraded to v3.0`
4. Continue with command.

Extract:
- `goal`, `state`, `current_phase` from top level
- `plan.phases` for phase data
- `errors.records` for error context
- `memory` for decisions/learnings

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
- Add `build_started_at` field with current ISO-8601 UTC timestamp
- Append to `events`: `"<timestamp>|phase_started|build|Phase <id>: <name> started"`

If `events` exceeds 100 entries, keep only the last 100.

Write COLONY_STATE.json.

### Step 3: Git Checkpoint

Create a git checkpoint for rollback capability.

```bash
git rev-parse --git-dir 2>/dev/null
```

- **If succeeds** (is a git repo): `git add -A && git commit --allow-empty -m "aether-checkpoint: pre-phase-$PHASE_NUMBER"`
  Store the commit hash.
- **If fails** (not a git repo): Set checkpoint hash to `"(not a git repo)"`.

Output header:

```
🐜 ═══════════════════════════════════════════════════
   B U I L D I N G   P H A S E   {id}
═══════════════════════════════════════════════════ 🐜

📍 Phase {id}: {name}
💾 Git Checkpoint: {commit_hash}
```

### Step 4: Load Constraints

Read `.aether/data/constraints.json` if it exists.

Format for display:
```
CONSTRAINTS:
  FOCUS: {focus areas, comma-separated}
  AVOID: {patterns to avoid from constraints}
```

If file doesn't exist or is empty:
```
CONSTRAINTS: (none)
```

### Step 5: Spawn Prime Worker

Log phase start:
```bash
bash ~/.aether/aether-utils.sh activity-log "EXECUTING" "phase" "Phase {id}: {name} started"
```

Update watch status:
```
Write .aether/data/watch-status.txt:

🐜 AETHER COLONY :: EXECUTING
==============================

State: EXECUTING
Phase: {id}/{total_phases}

Current Work:
  Executing phase tasks...

```

**IMPORTANT: Honest Execution Model**
The system executes tasks via a single Task agent, not parallel workers.
The "colony" metaphor describes task organization, not actual parallelism.

Dispatch **Prime Worker** via Task tool with `subagent_type="general-purpose"`:

```
You are the Prime Worker for Phase {id} in the Aether Colony.

You are at depth 1. You can spawn up to 4 specialists (depth 2).
Each specialist can spawn up to 2 sub-specialists (depth 3).
Depth 3 workers cannot spawn further.

--- PHASE CONTEXT ---

Goal: "{goal}"

Phase {id}: {phase_name}
{phase_description}

Tasks:
{for each task:}
  - {task_id}: {description}
    Depends on: {depends_on or "none"}
    Status: {status}
{end for}

Success Criteria:
{list success_criteria}

--- CONSTRAINTS ---
{constraints from Step 4, or "(none)"}

--- ERROR CONTEXT ---
{if errors.records has entries for this phase:}
Previous errors in this phase:
{list relevant errors}
{else:}
No previous errors.
{end if}

--- WORKER SPECS ---
Read ~/.aether/workers.md for role definitions and spawn protocol.

--- VERIFICATION DISCIPLINE ---
Read ~/.aether/verification.md for the Iron Law.

Key rules:
- NO completion claims without fresh verification evidence
- Before reporting done: RUN verification, READ output, THEN claim
- When spawns report success: verify independently (check files, run tests)
- Red flags: "should work", "probably done", satisfaction without evidence

--- DEBUGGING DISCIPLINE ---
Read ~/.aether/debugging.md when encountering ANY error.

Key rules:
- NO fixes without root cause investigation first
- Phase 1: Read error, reproduce, trace data flow to source
- Phase 2: Find working examples, compare
- Phase 3: Single hypothesis, minimal test
- Phase 4: Create failing test, fix at root cause
- **3-Fix Rule:** If 3+ fixes fail, STOP and report architectural concern
- Red flags: "quick fix", "just try X", "might work"

--- TDD DISCIPLINE ---
Read ~/.aether/tdd.md for the Iron Law.

Key rules:
- NO production code without a failing test first
- RED: Write failing test → VERIFY it fails correctly
- GREEN: Write minimal code → VERIFY it passes
- REFACTOR: Clean up while staying green
- Coverage target: 80%+ for new code
- Red flags: "test after", "too simple to test", test passes immediately

--- COLONY INSTINCTS ---
{if memory.instincts has entries:}
Learned patterns from previous phases (apply high-confidence automatically):
{for each instinct in memory.instincts where confidence >= 0.5:}
  [{confidence}] {domain}: {action}
{end for}
{else:}
No instincts yet. Observe patterns for colony learning.
{end if}

--- LEARNING ---
Read ~/.aether/learning.md for pattern detection.

Observe and report:
- Success patterns (what worked well)
- Error resolutions (what was learned from debugging)
- User feedback (corrections, preferences)

--- PARALLEL EXECUTION (Real, Not Theatrical) ---

To achieve ACTUAL parallelism:

1. Identify tasks with NO dependencies (depends_on: "none")
2. For 2+ independent tasks, spawn them in a SINGLE message using Task tool
3. Use run_in_background: true for each
4. All Task calls in one message = true parallel execution
5. Use TaskOutput to collect results

Example: If tasks 1.1 and 1.2 are independent:
- Call Task tool TWICE in ONE message (both with run_in_background: true)
- Both agents run simultaneously
- Collect results with TaskOutput when notified

This is how the colony achieves real parallelism, not just logging.

--- YOUR MISSION ---

1. Analyze tasks - identify which have depends_on: "none" (independent)
2. For simple tasks (< 10 tool calls), do them yourself
3. For 2+ independent tasks: spawn parallel agents in ONE message
4. For dependent tasks: execute sequentially after dependencies complete
5. Spawn specialists for complex work:
   - 🔨 Builder: code implementation, file manipulation
   - 👁️ Watcher: testing, validation, quality checks
   - 🔍 Scout: research, documentation lookup
   - 🗺️ Colonizer: codebase exploration
6. Synthesize all results
7. **VERIFY with evidence:** For each success criterion, run proof and record evidence
8. Log activity: bash ~/.aether/aether-utils.sh activity-log "ACTION" "phase" "description"

--- OUTPUT FORMAT ---

Return JSON:
{
  "status": "completed" | "failed" | "blocked",
  "summary": "What the phase accomplished",
  "tasks_completed": ["1.1", "1.2"],
  "tasks_failed": [],
  "files_created": ["path1", "path2"],
  "files_modified": ["path3"],
  "spawn_tree": {
    "builder-1": {"task": "...", "status": "completed", "children": {}},
    "watcher-1": {"task": "...", "status": "completed", "children": {}}
  },
  "verification": {
    "build": {"command": "npm run build", "exit_code": 0, "passed": true},
    "tests": {"command": "npm test", "passed": 24, "failed": 0, "total": 24},
    "success_criteria": [
      {"criterion": "API endpoint exists", "evidence": "GET /api/users returns 200", "passed": true},
      {"criterion": "Tests cover happy path", "evidence": "3 tests in users.test.ts", "passed": true}
    ]
  },
  "debugging": {
    "issues_encountered": 0,
    "issues_resolved": 0,
    "fix_attempts": 0,
    "architectural_concerns": []
  },
  "tdd": {
    "cycles_completed": 5,
    "tests_added": 5,
    "tests_total": 47,
    "coverage_percent": 85,
    "all_passing": true
  },
  "learning": {
    "patterns_observed": [
      {
        "type": "success",
        "trigger": "when implementing API endpoints",
        "action": "use repository pattern with DI",
        "evidence": "All tests passed first try"
      }
    ],
    "instincts_applied": ["instinct_123"],
    "instinct_outcomes": [
      {"id": "instinct_123", "success": true}
    ]
  },
  "quality_notes": "Any concerns or recommendations",
  "ui_touched": true | false
}
```

Wait for Prime Worker to complete.

### Step 6: Visual Checkpoint (if UI touched)

Parse Prime Worker result. If `ui_touched` is true:

```
Visual Checkpoint
=================

UI changes detected. Verify appearance before continuing.

Files touched:
{list files from files_created + files_modified that match UI patterns}

Options:
  1. Approve - UI looks correct
  2. Reject - needs changes (describe issues)
  3. Skip - defer visual review
```

Use AskUserQuestion to get approval. Record in events:
- If approved: `"<timestamp>|visual_approved|build|Phase {id} UI approved"`
- If rejected: `"<timestamp>|visual_rejected|build|Phase {id} UI rejected: {reason}"`

### Step 7: Display Results

Display build summary:

```
🐜 ═══════════════════════════════════════════════════
   P H A S E   {id}   C O M P L E T E
═══════════════════════════════════════════════════ 🐜

📍 Phase {id}: {name}
📊 Status: {status}
💾 Git Checkpoint: {commit_hash}

📝 Summary:
   {summary from Prime Worker}

🐜 Colony Work Tree:
   👑 Queen
   └── 🐜 Prime Worker
{for each spawn in spawn_tree:}
       ├── {emoji} {caste}: {task} [{status}]
{for each child in spawn.children:}
       │   └── {emoji} {caste}: {task} [{status}]
{end for}
{end for}

✅ Tasks Completed:
{for each task in tasks_completed:}
   🐜 {task_id}: done
{end for}
{for each task in tasks_failed:}
   ❌ {task_id}: failed
{end for}

📁 Files: {files_created count} created, {files_modified count} modified

{if tdd.tests_added > 0:}
🧪 TDD: {tdd.cycles_completed} cycles | {tdd.tests_added} tests | {tdd.coverage_percent}% coverage
{end if}

{if learning.patterns_observed not empty:}
🧠 Patterns Learned:
{for each pattern in learning.patterns_observed:}
   🐜 {pattern.trigger} → {pattern.action}
{end for}
{end if}

{if debugging.issues_encountered > 0:}
🔧 Debugging: {debugging.issues_resolved}/{debugging.issues_encountered} resolved
{end if}

🐜 Next Steps:
   /ant:continue   ➡️  Advance to next phase
   /ant:feedback   💬 Give feedback first
   /ant:status     📊 View colony status
```

**IMPORTANT:** Build does NOT update task statuses or advance state. Run `/ant:continue` to:
- Mark tasks as completed
- Extract learnings
- Advance to next phase
