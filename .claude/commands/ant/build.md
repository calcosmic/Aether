---
name: ant:build
description: "🔨🐜🏗️🐜🔨 Build a phase with pure emergence - colony self-organizes and completes tasks"
---

You are the **Queen**. You DIRECTLY spawn multiple workers — do not delegate to a single Prime Worker.

The phase to build is: `$ARGUMENTS`

## Instructions

### Step 0: Version Check (Non-blocking)

Run using the Bash tool: `bash .aether/aether-utils.sh version-check 2>/dev/null || true`

If the command succeeds and the JSON result contains a non-empty string, display it as a one-line notice. Proceed regardless of outcome.

### Step 0.6: Verify LiteLLM Proxy

Check that the LiteLLM proxy is running for model routing:

```bash
curl -s http://localhost:4000/health | grep -q "healthy" && echo "Proxy healthy" || echo "Proxy not running - workers will use default model"
```

If proxy is not healthy, log a warning but continue (workers will fall back to default routing).

### Step 0.5: Load Colony State

Run using Bash tool: `bash .aether/aether-utils.sh load-state`

If the command fails (non-zero exit or JSON has ok: false):
1. Parse error JSON
2. If error code is E_FILE_NOT_FOUND: "No colony initialized. Run /ant:init first." and stop
3. If validation error: Display error details with recovery suggestion and stop
4. For other errors: Display generic error and suggest /ant:status for diagnostics

If successful:
1. Parse the state JSON from result field
2. Check if goal is null - if so: "No colony initialized. Run /ant:init first." and stop
3. Extract current_phase and phase name from plan.phases[current_phase - 1].name
4. Display brief resumption context:
   ```
   🔄 Resuming: Phase X - Name
   ```
   (If HANDOFF.md exists, this provides orientation before the build proceeds)

After displaying context, run: `bash .aether/aether-utils.sh unload-state` to release the lock.

### Step 1: Validate + Read State

**Parse $ARGUMENTS:**
1. Extract the phase number (first argument)
2. Check remaining arguments for flags:
   - If contains `--verbose` or `-v`: set `verbose_mode = true`
   - If contains `--model <name>` or `-m <name>`: set `cli_model_override = <name>`
   - Otherwise: set defaults

If the phase number is empty or not a number:

```
Usage: /ant:build <phase_number> [--verbose|-v] [--model <model>|-m <model>]

Options:
  --verbose, -v       Show full completion details (spawn tree, TDD, patterns)
  --model, -m <name>  Override model for this build (one-time)

Examples:
  /ant:build 1              Build Phase 1 (compact output)
  /ant:build 1 --verbose    Build Phase 1 (full details)
  /ant:build 3 -v           Build Phase 3 (full details)
  /ant:build 1 --model glm-5    Build Phase 1 with glm-5 for all workers
```

Stop here.

**Validate CLI model override (if provided):**
If `cli_model_override` is set:
1. Validate the model: `bash .aether/aether-utils.sh model-profile validate "$cli_model_override"`
2. Parse JSON result - if `.result.valid` is false:
   - Display: `Error: Invalid model "$cli_model_override"`
   - Display: `Valid models: {list from .result.models}`
   - Stop here
3. If valid: Display `Using override model: {model}`

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
- If `plan.phases` is empty -> output `No project plan. Run /ant:plan first.` and stop.
- Find the phase matching the requested ID. If not found -> output `Phase {id} not found.` and stop.
- If the phase status is `"completed"` -> output `Phase {id} already completed.` and stop.

### Step 1.5: Blocker Advisory (Non-blocking)

Check for unresolved blocker flags on the requested phase:

```bash
bash .aether/aether-utils.sh flag-check-blockers {phase_number}
```

Parse the JSON result (`.result.blockers`):

- **If blockers == 0:** Display nothing (or optionally a brief `No active blockers for Phase {id}.` line). Proceed to Step 2.
- **If blockers > 0:** Retrieve blocker details:
  ```bash
  bash .aether/aether-utils.sh flag-list --type blocker --phase {phase_number}
  ```
  Parse `.result.flags` and display an advisory warning:
  ```
  ⚠️  BLOCKER ADVISORY: {blockers} unresolved blocker(s) for Phase {id}
  {for each flag in result.flags:}
     - [{flag.id}] {flag.title}
  {end for}

  Consider reviewing with /ant:flags or auto-fixing with /ant:swarm before building.
  Proceeding anyway...
  ```
  **This is advisory only — do NOT stop.** Continue to Step 2 regardless.

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

- **If succeeds** (is a git repo):
  1. Check for changes in Aether-managed directories only: `.aether .claude/commands/ant .claude/commands/st .opencode runtime bin`
  2. **If changes exist**: `git stash push -m "aether-checkpoint: pre-phase-$PHASE_NUMBER" -- .aether .claude/commands/ant .claude/commands/st .opencode runtime bin`
     - IMPORTANT: Never use `--include-untracked` — it stashes ALL files including user work!
     - Verify: `git stash list | head -1 | grep "aether-checkpoint"` — warn if empty
     - Store checkpoint as `{type: "stash", ref: "aether-checkpoint: pre-phase-$PHASE_NUMBER"}`
  3. **If clean working tree**: Record `HEAD` hash via `git rev-parse HEAD`
     - Store checkpoint as `{type: "commit", ref: "$HEAD_HASH"}`
- **If fails** (not a git repo): Set checkpoint to `{type: "none", ref: "(not a git repo)"}`.

Rollback procedure: `git stash pop` (if type is "stash") or `git reset --hard $ref` (if type is "commit").

Output header:

```
🔨🐜🏗️🐜🔨 ═══════════════════════════════════════════════════
   B U I L D I N G   P H A S E   {id}
═══════════════════════════════════════════════════ 🔨🐜🏗️🐜🔨

📍 Phase {id}: {name}
💾 Git Checkpoint: {checkpoint_type} → {checkpoint_ref}
🔄 Rollback: `git stash pop` (stash) or `git reset --hard {ref}` (commit)
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

### Step 4.5: Archaeologist Pre-Build Scan

**Conditional step — only fires when the phase modifies existing files.**

1. **Detect existing-file modification:**
   Examine each task in the phase. Look at task descriptions, constraints, and hints for signals:
   - Keywords: "update", "modify", "add to", "integrate into", "extend", "change", "refactor", "fix"
   - References to existing file paths (files that already exist in the repo)
   - Task type: if a task is purely "create new file X" with no references to existing code, it is new-file-only

   **If ALL tasks are new-file-only** (no existing files will be modified):
   - Skip this step silently — produce no output, no spawn
   - Proceed directly to Step 5

2. **If existing code modification detected — spawn Archaeologist Scout:**

   Generate archaeologist name and log:
   ```bash
   bash .aether/aether-utils.sh generate-ant-name "archaeologist"
   bash .aether/aether-utils.sh spawn-log "Queen" "scout" "{archaeologist_name}" "Pre-build archaeology scan"
   ```

   Display:
   ```
   🏺 Spawning Archaeologist: {archaeologist_name}
      Scanning history of files to be modified...
   ```

   Spawn a Scout (using Task tool with `subagent_type="general"`) with this prompt:

   ```
   You are {Archaeologist-Name}, a 🏺 Archaeologist Ant (Scout) in the Aether Colony.

   --- YOUR MISSION ---
   Perform a pre-build archaeology scan on files that are about to be modified.

   --- FILES TO INVESTIGATE ---
   {list of existing files that will be modified by this phase's tasks}

   --- INSTRUCTIONS ---
   For each file:
   1. Read the file to understand its current state
   2. Run: git log --oneline -15 -- "{file_path}" to see recent history
   3. Run: git log --all --grep="fix\|bug\|workaround\|hack\|revert" --oneline -- "{file_path}" to find incident history
   4. Run: git blame "{file_path}" | head -40 to see authorship of key sections
   5. Note any TODO/FIXME/HACK markers in the current code

   --- OUTPUT ---
   For each file, report:
   - WHY key code sections exist (from commit messages)
   - Known workarounds or hacks that must not be broken
   - Key architectural decisions visible in history
   - Areas of caution (high churn, reverted changes, emergency fixes)
   - Sections that are stable bedrock vs volatile sand

   Keep the report concise and actionable — builders need quick context, not a thesis.
   Format as plain text with file headers. No JSON output needed.
   ```

   **Wait for results** (blocking — use TaskOutput with `block: true`).

   Log completion:
   ```bash
   bash .aether/aether-utils.sh spawn-complete "{archaeologist_name}" "completed" "Pre-build archaeology scan"
   ```

3. **Store and display findings:**

   Store the archaeologist's output as `archaeology_context`.

   Display summary:
   ```
   🏺 ARCHAEOLOGY CONTEXT
   =====================
   {summary of findings from archaeologist}
   ```

4. **Injection into builder prompts:**
   The `archaeology_context` will be injected into builder prompts in Step 5.1 (see below).
   If this step was skipped (no existing files modified), the archaeology section is omitted from builder prompts.

### Step 5: Analyze Tasks and Plan Spawns

**YOU (the Queen) will spawn workers directly. Do NOT delegate to a single Prime Worker.**

Log phase start:
```bash
bash .aether/aether-utils.sh activity-log "EXECUTING" "Queen" "Phase {id}: {name} - Queen dispatching workers"
```

Analyze the phase tasks:

1. **Group tasks by dependencies:**
   - **Wave 1:** Tasks with `depends_on: "none"` or `depends_on: []` (can run in parallel)
   - **Wave 2:** Tasks depending on Wave 1 tasks
   - **Wave 3+:** Continue until all tasks assigned

2. **Assign castes:**
   - Implementation tasks → 🔨 Builder
   - Research/docs tasks → 🔍 Scout
   - Testing/validation → 👁️ Watcher (ALWAYS spawn at least one)
   - Resilience testing → 🎲 Chaos (ALWAYS spawn one after Watcher)

3. **Generate ant names for each worker:**
```bash
bash .aether/aether-utils.sh generate-ant-name "builder"
bash .aether/aether-utils.sh generate-ant-name "watcher"
bash .aether/aether-utils.sh generate-ant-name "chaos"
```

Display spawn plan:
```
🐜 SPAWN PLAN
=============
Wave 1 (parallel):
  🔨{Builder-Name}: Task {id} - {description}
  🔨{Builder-Name}: Task {id} - {description}

Wave 2 (after Wave 1):
  🔨{Builder-Name}: Task {id} - {description}

Verification:
  👁️{Watcher-Name}: Verify all work independently
  🎲{Chaos-Name}: Resilience testing (after Watcher)

Total: {N} Builders + 1 Watcher + 1 Chaos = {N+2} spawns
```

### Step 5.1: Spawn Wave 1 Workers (Parallel)

**CRITICAL: Spawn ALL Wave 1 workers in a SINGLE message using multiple Task tool calls.**

For each Wave 1 task, use Task tool with `subagent_type="general"` and `run_in_background: true`:

Log each spawn:
```bash
bash .aether/aether-utils.sh spawn-log "Queen" "builder" "{ant_name}" "{task_description}"
```

**Model Assignment:**

Before spawning each worker, get the optimal model for their caste with task-based routing:

```bash
# Get model assignment for this caste with task-based routing
model_info=$(bash .aether/aether-utils.sh model-profile select "{caste}" "{task_description}" "{cli_model_override}")
model=$(echo "$model_info" | jq -r '.result.model')
source=$(echo "$model_info" | jq -r '.result.source')

# Log model assignment with source
bash .aether/aether-utils.sh activity-log "MODEL" "Queen" "{ant_name} ({caste}): assigned to $model (source: $source)"
```

**Environment Setup for Workers:**

When spawning workers, the following environment variables must be set for Claude Code to use the LiteLLM proxy with the correct model:

```bash
export ANTHROPIC_BASE_URL="http://localhost:4000"
export ANTHROPIC_AUTH_TOKEN="sk-litellm-local"
export ANTHROPIC_MODEL="$model"  # From model-profile get above
```

**IMPORTANT:** The Task tool inherits environment from the parent Claude Code process. Set these variables in the shell before spawning workers, or ensure they are already set in the parent environment.

### Step 5.1.5: Export Model Environment

Before spawning workers, export the environment variables for LiteLLM proxy routing:

```bash
export ANTHROPIC_BASE_URL="http://localhost:4000"
export ANTHROPIC_AUTH_TOKEN="sk-litellm-local"
```

For each worker spawn, also export their specific model:

```bash
export ANTHROPIC_MODEL="$model"  # From model-profile get
```

**Note:** These environment variables will be inherited by spawned workers via the Task tool, enabling automatic model routing through the LiteLLM proxy.

**Builder Worker Prompt Template:**
```
You are {Ant-Name}, a 🔨 Builder Ant in the Aether Colony at depth {depth}.

--- YOUR TASK ---
Task {id}: {description}

--- MODEL CONTEXT ---
Optimal model for this task: {model} (from caste: {caste})
Model characteristics: {model_description}
Task complexity expectation: {simple|medium|complex}

The model has been pre-selected based on your caste's typical work patterns.
- glm-5: Use for complex reasoning, architecture, planning
- kimi-k2.5: Use for fast implementation, coding, refactoring
- minimax-2.5: Use for validation, research, quick checks

--- CONTEXT ---
Goal: "{colony_goal}"
Phase: {phase_name}

--- CONSTRAINTS ---
{constraints from Step 4}

--- COLONY KNOWLEDGE ---
{Include this section ONLY if memory.instincts or memory.phase_learnings exist in COLONY_STATE.json.}

Top Instincts (proven patterns — follow these):
{For each instinct in memory.instincts where confidence >= 0.5, sorted by confidence descending, max 5:}
  [{confidence}] {trigger} → {action}
{If none qualify: omit this sub-section}

Recent Learnings:
{For each learning in memory.phase_learnings (last 3 phases only) where status == "validated":}
  - {claim}
{If none qualify: omit this sub-section}

Error Patterns to Avoid:
{For each pattern in errors.flagged_patterns:}
  ⚠️ {description}
{If none: omit this sub-section}

{If archaeology_context exists (Step 4.5 produced findings):}
--- ARCHAEOLOGY CONTEXT ---
The following historical insights were discovered about files you will modify:
{archaeology_context findings}
{End if — omit this entire section if Step 4.5 was skipped}

--- INSTRUCTIONS ---
1. Read .aether/workers.md for Builder discipline
2. Implement the task completely
3. Write actual test files (not just claims)
4. Log your work: bash .aether/aether-utils.sh activity-log "CREATED" "{ant_name} (Builder)" "{file_path}"
5. Before modifying any file, check for grave markers:
   bash .aether/aether-utils.sh grave-check "{file_path}"
   If caution_level is "high": read the failure_summary, add extra test coverage for that area, mention the graveyard in your summary
   If caution_level is "low": note it and proceed carefully
   If caution_level is "none": proceed normally

--- SPAWN CAPABILITY ---
You are at depth {depth}. You MAY spawn sub-workers if you encounter genuine surprise (3x expected complexity).

Spawn limits by depth:
- Depth 1: max 4 spawns
- Depth 2: max 2 spawns
- Depth 3: NO spawns (complete inline)

When to spawn:
- Task is 3x larger than expected
- Discovered sub-domain requiring different expertise
- Found blocking dependency needing parallel investigation

DO NOT spawn for work you can complete in < 10 tool calls.

Before spawning:
  1. Check: bash .aether/aether-utils.sh spawn-can-spawn {depth}
  2. Generate name: bash .aether/aether-utils.sh generate-ant-name "{caste}"
  3. Log: bash .aether/aether-utils.sh spawn-log "{your_name}" "{caste}" "{child_name}" "{task}"
  4. Use Task tool with subagent_type="general"
  5. After completion: bash .aether/aether-utils.sh spawn-complete "{child_name}" "{status}" "{summary}"

Full spawn format: .aether/workers.md section "Spawning Sub-Workers"

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

### Step 5.2: Collect Wave 1 Results (BLOCKING)

**CRITICAL: You MUST wait for ALL Wave 1 workers to complete before proceeding.**

For each spawned worker, call TaskOutput with `block: true` to wait for completion:
- Use the task_id from each Task tool response
- Do NOT proceed to Step 5.3 until ALL workers have returned results
- Parse each worker's JSON output to collect: status, files_created, files_modified, blockers

Store all results for synthesis in Step 5.6.

For each completed worker, log:
```bash
bash .aether/aether-utils.sh spawn-complete "{ant_name}" "completed" "{summary}"
```

**Only proceed to Step 5.3 after ALL Wave 1 TaskOutput calls have returned.**

### Step 5.3: Spawn Wave 2+ Workers (Sequential Waves)

Repeat Step 5.1-5.2 for each subsequent wave, waiting for previous wave to complete.

### Step 5.4: Spawn Watcher for Verification

**MANDATORY: Always spawn a Watcher — testing must be independent.**

```bash
bash .aether/aether-utils.sh spawn-log "Queen" "watcher" "{watcher_name}" "Independent verification"
```

**Watcher Worker Prompt:**
```
You are {Watcher-Name}, a 👁️ Watcher Ant in the Aether Colony at depth {depth}.

--- YOUR MISSION ---
Independently verify all work done by Builders in Phase {id}.

--- MODEL CONTEXT ---
Optimal model for verification: {model} (from caste: watcher)
Model characteristics: Efficient validation and testing model
Verification focus: Quick, thorough checks with fast turnaround

--- WHAT TO VERIFY ---
Files created: {list from builder results}
Files modified: {list from builder results}

--- COMMAND RESOLUTION ---
Resolve build, test, type-check, and lint commands using this priority chain (stop at first match per command):
1. **CLAUDE.md** — Check project CLAUDE.md (in your system context) for explicit commands
2. **CODEBASE.md** — Read `.planning/CODEBASE.md` `## Commands` section
3. **Fallback** — Use language-specific examples below (Execution Verification section)

Use resolved commands for all verification steps below.

--- VERIFICATION CHECKLIST ---
1. Do the files exist? (Read each one)
2. Does the code compile/parse? (Run build command)
3. Do tests exist AND pass? (Run test command)
4. Are success criteria met? {list success_criteria}

--- EXECUTION VERIFICATION (MANDATORY) ---
Before assigning a quality score, you MUST attempt to execute the code:

1. Syntax check: Run the language's syntax checker
   - Python: `python3 -m py_compile {file}`
   - Swift: `swiftc -parse {file}`
   - TypeScript: `npx tsc --noEmit`

2. Import check: Verify main entry point can be imported
   - Python: `python3 -c "import {module}"`
   - Node: `node -e "require('{entry}')"`

3. Launch test: Attempt to start the application briefly
   - Run main entry point with timeout
   - If GUI, try headless mode if possible
   - If launches successfully = pass
   - If crashes = CRITICAL severity

4. Test suite: If tests exist, run them
   - Record pass/fail counts

CRITICAL: If ANY execution check fails, quality_score CANNOT exceed 6/10.

--- SPAWN CAPABILITY ---
You are at depth {depth}. You MAY spawn sub-workers for:
- Deep investigation of suspicious code patterns
- Parallel verification of independent components
- Debugging assistance for complex failures

Spawn limits: Depth 1→4, Depth 2→2, Depth 3→0

Before spawning:
  bash .aether/aether-utils.sh spawn-log "{your_name}" "{caste}" "{child_name}" "{task}"

--- CRITICAL ---
- You did NOT build this code — verify it objectively
- "Build passing" is NOT enough — check runtime execution
- Be skeptical — Builders may have cut corners

--- OUTPUT ---
Return JSON:
{
  "ant_name": "{your name}",
  "verification_passed": true | false,
  "files_verified": [],
  "execution_verification": {
    "syntax_check": {"command": "...", "passed": true|false},
    "import_check": {"command": "...", "passed": true|false},
    "launch_test": {"command": "...", "passed": true|false, "error": null},
    "test_suite": {"command": "...", "passed": N, "failed": N}
  },
  "build_result": {"command": "...", "passed": true|false},
  "test_result": {"command": "...", "passed": N, "failed": N},
  "success_criteria_results": [
    {"criterion": "...", "passed": true|false, "evidence": "..."}
  ],
  "issues_found": [],
  "quality_score": N,
  "recommendation": "proceed" | "fix_required",
  "spawns": []
}
```

### Step 5.4.1: Collect Watcher Results (BLOCKING)

**CRITICAL: You MUST wait for the Watcher to complete before proceeding.**

Call TaskOutput with `block: true` using the Watcher's task_id:
- Wait for the Watcher's JSON response
- Parse: verification_passed, issues_found, quality_score, recommendation
- Store results for synthesis in Step 5.6

**Only proceed to Step 5.4.2 after Watcher TaskOutput has returned.**

### Step 5.4.2: Spawn Chaos Ant for Resilience Testing

**After the Watcher completes, spawn a Chaos Ant to probe the phase work for edge cases and boundary conditions.**

Generate a chaos ant name and log the spawn:
```bash
bash .aether/aether-utils.sh generate-ant-name "chaos"
bash .aether/aether-utils.sh spawn-log "Queen" "chaos" "{chaos_name}" "Resilience testing of Phase {id} work"
```

**Retrieve existing flags for this phase** (to avoid duplicate findings):
```bash
bash .aether/aether-utils.sh flag-list --phase {phase_number}
```
Parse the result and extract unresolved flag titles into a list: `{existing_flag_titles}` (comma-separated titles from `.result.flags[].title`). If no flags exist, set `{existing_flag_titles}` to "None".

Spawn the Chaos Ant using Task tool with `subagent_type="general"`:

**Chaos Ant Prompt:**
```
You are {Chaos-Name}, a 🎲 Chaos Ant (Resilience Tester) in the Aether Colony at depth {depth}.

--- YOUR MISSION ---
Probe the work done by Builders in Phase {id} for edge cases, boundary conditions, and unexpected inputs.

--- MODEL CONTEXT ---
Optimal model for chaos testing: {model} (from caste: chaos)
Model characteristics: Efficient edge case exploration
Testing focus: Rapid probing of boundaries and edge cases

--- SCOPE ---
Files created: {list from builder results}
Files modified: {list from builder results}

--- EXISTING FLAGS (already known — do NOT re-report) ---
{existing_flag_titles}
These issues have already been flagged. Do NOT report findings that duplicate or overlap with the above titles. Focus your 5 scenarios on NEW, undiscovered issues only.

--- RULES ---
1. Limit to 5 edge case scenarios maximum
2. You are a TESTER, not an attacker — use investigating/probing language
3. Do NOT modify any code — read-only analysis
4. Focus on: edge cases, boundary conditions, error handling gaps, state corruption risks, unexpected inputs
5. Do NOT re-report issues listed in EXISTING FLAGS above — skip any finding that substantially overlaps with a known flag

--- OUTPUT ---
Return JSON:
{
  "ant_name": "{your name}",
  "scenarios_tested": 5,
  "findings": [
    {
      "id": 1,
      "category": "edge_case|boundary|error_handling|state|unexpected_input",
      "severity": "critical|high|medium|low|info",
      "title": "...",
      "description": "...",
      "reproduction_steps": ["..."],
      "affected_files": ["..."],
      "recommendation": "..."
    }
  ],
  "overall_resilience": "strong|moderate|weak",
  "summary": "..."
}
```

**Collect Chaos Ant results (BLOCKING):**

Call TaskOutput with `block: true` using the Chaos Ant's task_id:
- Wait for the Chaos Ant's JSON response
- Parse: findings, overall_resilience, summary
- Store results for synthesis in Step 5.6

**Flag critical/high findings:**

If any findings have severity `"critical"` or `"high"`:
```bash
# Create a blocker flag for each critical/high chaos finding
bash .aether/aether-utils.sh flag-add "blocker" "{finding.title}" "{finding.description}" "chaos-testing" {phase_number}
```

Log the flag:
```bash
bash .aether/aether-utils.sh activity-log "FLAG" "Chaos" "Created blocker: {finding.title}"
```

Log chaos ant completion:
```bash
bash .aether/aether-utils.sh spawn-complete "{chaos_name}" "completed" "{summary}"
```

**Only proceed to Step 5.5 after Chaos Ant TaskOutput has returned.**

### Step 5.5: Create Flags for Verification Failures

If the Watcher reported `verification_passed: false` or `recommendation: "fix_required"`:

For each issue in `issues_found`:
```bash
# Create a blocker flag for each verification failure
bash .aether/aether-utils.sh flag-add "blocker" "{issue_title}" "{issue_description}" "verification" {phase_number}
```

Log the flag creation:
```bash
bash .aether/aether-utils.sh activity-log "FLAG" "Watcher" "Created blocker: {issue_title}"
```

This ensures verification failures are persisted as blockers that survive context resets. Chaos Ant findings are flagged in Step 5.4.2.

### Step 5.6: Synthesize Results

**This step runs ONLY after ALL TaskOutput calls have returned (Steps 5.2, 5.3, 5.4.1, 5.4.2).**

Collect all worker outputs and create phase summary:

```json
{
  "status": "completed" | "failed" | "blocked",
  "summary": "...",
  "tasks_completed": [...],
  "tasks_failed": [...],
  "files_created": [...],
  "files_modified": [...],
  "spawn_metrics": {
    "spawn_count": {total workers spawned, including archaeologist if Step 4.5 fired},
    "builder_count": {N},
    "watcher_count": 1,
    "chaos_count": 1,
    "archaeologist_count": {0 or 1, conditional on Step 4.5},
    "parallel_batches": {number of waves}
  },
  "spawn_tree": {
    "{Archaeologist-Name}": {"caste": "archaeologist", "task": "pre-build history scan", "status": "completed"},
    "{Builder-Name}": {"caste": "builder", "task": "...", "status": "completed"},
    "{Watcher-Name}": {"caste": "watcher", "task": "verify", "status": "completed"},
    "{Chaos-Name}": {"caste": "chaos", "task": "resilience testing", "status": "completed"}
  },
  "verification": {from Watcher output},
  "resilience": {from Chaos Ant output},
  "archaeology": {from Archaeologist output, or null if Step 4.5 was skipped},
  "quality_notes": "..."
}
```

**Graveyard Recording:**
For each worker that returned `status: "failed"`:
  For each file in that worker's `files_modified` or `files_created`:
```bash
bash .aether/aether-utils.sh grave-add "{file}" "{ant_name}" "{task_id}" {phase} "{first blocker or summary}"
```
  Log the grave marker:
```bash
bash .aether/aether-utils.sh activity-log "GRAVE" "Queen" "Grave marker placed at {file} — {ant_name} failed: {summary}"
```

Only fires when workers fail. Zero impact on successful builds.

--- SPAWN TRACKING ---

The spawn tree will be visible in `/ant:watch` because each spawn is logged.

--- OUTPUT FORMAT ---

Return JSON:
{
  "status": "completed" | "failed" | "blocked",
  "summary": "What the phase accomplished",
  "tasks_completed": ["1.1", "1.2"],
  "tasks_failed": [],
  "files_created": ["path1", "path2"],
  "files_modified": ["path3"],
  "spawn_metrics": {
    "spawn_count": 6,
    "watcher_count": 1,
    "chaos_count": 1,
    "archaeologist_count": 1,
    "builder_count": 3,
    "parallel_batches": 2,
    "sequential_tasks": 1
  },
  "spawn_tree": {
    "Relic-8": {"caste": "archaeologist", "task": "pre-build history scan", "status": "completed", "children": {}},
    "Hammer-42": {"caste": "builder", "task": "...", "status": "completed", "children": {}},
    "Vigil-17": {"caste": "watcher", "task": "...", "status": "completed", "children": {}},
    "Entropy-9": {"caste": "chaos", "task": "resilience testing", "status": "completed", "children": {}}
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

### Step 6: Visual Checkpoint (if UI touched)

Parse synthesis result. If `ui_touched` is true:

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

**This step runs ONLY after synthesis is complete. All values come from actual worker results.**

Display build summary based on synthesis results AND `verbose_mode` from Step 1:

**If verbose_mode = false (compact output, ~12 lines):**

```
🔨 PHASE {id} {status_icon}
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📍 {name}
📊 {status} | 📁 {files_created count} created, {files_modified count} modified
🐜 {spawn_count} workers | 🧪 {tests_total} tests {if all_passing}passing{else}{passed}/{total}{end if}
{if learning.patterns_observed.length > 0:}🧠 +{patterns_observed.length} patterns{end if}

{if synthesis.status == "failed" OR verification.recommendation == "fix_required":}
⚠️  BLOCKERS: {first 2 issues, comma-separated}
{end if}

➡️  Next: {primary_command}
    --verbose for spawn tree, TDD details, patterns
```

**Status icon logic:** completed+proceed = checkmark, blockers = warning, failed = X

**Primary command logic:**
- completed + proceed: `/ant:continue`
- has blockers: `/ant:flags`
- failed: `/ant:swarm`

**If verbose_mode = true (full output):**

```
🔨🐜🏗️🐜🔨 ═══════════════════════════════════════════════════
   P H A S E   {id}   C O M P L E T E
═══════════════════════════════════════════════════ 🔨🐜🏗️🐜🔨

📍 Phase {id}: {name}
📊 Status: {status}
💾 Git Checkpoint: {commit_hash}

📝 Summary:
   {summary from synthesis}

🐜 Colony Work Tree:
   👑Queen
{for each spawn in spawn_tree:}
   ├── {caste_emoji}{ant_name}: {task} [{status}]
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
{if synthesis.status == "completed" AND verification.recommendation == "proceed":}
   /ant:continue   ➡️  Advance to next phase
   /ant:feedback   💬 Give feedback first
{else if synthesis.status == "failed" OR verification.recommendation == "fix_required":}
   ⚠️  BLOCKERS DETECTED - Cannot proceed until resolved
   /ant:flags      🚩 View blockers
   /ant:swarm      🔥 Auto-fix issues
{end if}

💾 State persisted — safe to /clear, then run /ant:continue
```

**Conditional Next Steps:** The suggestions above are based on actual worker results. If verification failed or blockers exist, `/ant:continue` is NOT suggested.

**IMPORTANT:** Build does NOT update task statuses or advance state. Run `/ant:continue` to:
- Mark tasks as completed
- Extract learnings
- Advance to next phase
