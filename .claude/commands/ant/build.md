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
   - If contains `--no-visual`: set `visual_mode = false` (visual is ON by default)
   - If contains `--model <name>` or `-m <name>`: set `cli_model_override = <name>`
   - Otherwise: set `visual_mode = true` (visual is default)

If the phase number is empty or not a number:

```
Usage: /ant:build <phase_number> [--verbose|-v] [--no-visual] [--model <model>|-m <model>]

Options:
  --verbose, -v       Show full completion details (spawn tree, TDD, patterns)
  --no-visual         Disable real-time visual display (visual is on by default)
  --model, -m <name>  Override model for this build (one-time)

Examples:
  /ant:build 1              Build Phase 1 (with visual display)
  /ant:build 1 --verbose    Build Phase 1 (full details + visual)
  /ant:build 1 --no-visual  Build Phase 1 without visual display
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

### Step 4.0: Load Territory Survey

Check if territory survey exists and load relevant documents:

```bash
bash .aether/aether-utils.sh survey-load "{phase_name}" 2>/dev/null
```

**Parse the JSON response:**
- If `.ok` is false: Set `survey_docs = null` and skip survey loading
- If successful: Extract `.docs` (comma-separated list) and `.dir`

**Determine phase type from phase name:**
| Phase Contains | Documents to Load |
|----------------|-------------------|
| UI, frontend, component, button, page | DISCIPLINES.md, CHAMBERS.md |
| API, endpoint, backend, route | BLUEPRINT.md, DISCIPLINES.md |
| database, schema, model, migration | BLUEPRINT.md, PROVISIONS.md |
| test, spec, coverage | SENTINEL-PROTOCOLS.md, DISCIPLINES.md |
| integration, external, client | TRAILS.md, PROVISIONS.md |
| refactor, cleanup, debt | PATHOGENS.md, BLUEPRINT.md |
| setup, config, initialize | PROVISIONS.md, CHAMBERS.md |
| *default* | PROVISIONS.md, BLUEPRINT.md |

**Read the relevant survey documents** from `.aether/data/survey/`:
- Extract key patterns to follow
- Note file locations for new code
- Identify known concerns to avoid

**Display summary:**
```
🗺️ SURVEY LOADED
================
{for each doc loaded}
  {emoji} {filename} — {brief description}
{/for}

{if no survey}
  (No territory survey — run /ant:colonize for deeper context)
{/if}
```

**Store for builder injection:**
- `survey_patterns` — patterns to follow
- `survey_locations` — where to place files
- `survey_concerns` — concerns to avoid

### Step 4.1: Load QUEEN.md Wisdom

Call `queen-read` to extract eternal wisdom for worker priming:

```bash
bash .aether/aether-utils.sh queen-read 2>/dev/null
```

**Parse the JSON response:**
- If `.ok` is false or command fails: Set `queen_wisdom = null` and skip wisdom injection
- If successful: Extract wisdom sections from `.result.wisdom`

**Store wisdom variables:**
```
queen_philosophies = .result.wisdom.philosophies (if .result.priming.has_philosophies)
queen_patterns = .result.wisdom.patterns (if .result.priming.has_patterns)
queen_redirects = .result.wisdom.redirects (if .result.priming.has_redirects)
queen_stack_wisdom = .result.wisdom.stack_wisdom (if .result.priming.has_stack_wisdom)
queen_decrees = .result.wisdom.decrees (if .result.priming.has_decrees)
```

**Display summary (if any wisdom exists):**
```
📜 QUEEN WISDOM LOADED
=====================
{if queen_philosophies:}  📜 Philosophies: yes{/if}
{if queen_patterns:}  🧭 Patterns: yes{/if}
{if queen_redirects:}  ⚠️ Redirects: yes{/if}
{if queen_stack_wisdom:}  🔧 Stack Wisdom: yes{/if}
{if queen_decrees:}  🏛️ Decrees: yes{/if}

{if none exist:}  (no eternal wisdom recorded yet){/if}
```

**Graceful handling:** If QUEEN.md doesn't exist or `queen-read` fails, continue without wisdom injection. Workers will receive standard prompts.

### Step 4.1.6: Load Active Pheromones (Signal Consumption)

**This injects current FOCUS and REDIRECT signals into worker context.**

Call `pheromone-read` to get active signals:

```bash
bash .aether/aether-utils.sh pheromone-read 2>/dev/null
```

**Parse the JSON response:**
- If `.ok` is false or command fails: Set `pheromone_section = null` and skip
- If successful: Extract `.result.priorities` and `.result.avoid`

**Display summary:**
```
🎯 ACTIVE SIGNALS
=================
Priorities (FOCUS): {N}
Constraints (REDIRECT): {M}
```

**Store for worker injection:** The `pheromone_section` markdown will be included in builder prompts (see Step 5.1 Active Signals Section).

### Step 4.2: Archaeologist Pre-Build Scan

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
   bash .aether/aether-utils.sh swarm-display-update "{archaeologist_name}" "scout" "excavating" "Pre-build archaeology scan" "Queen" '{"read":0,"grep":0,"edit":0,"bash":0}' 0 "fungus_garden" 15
   ```

   Display:
   ```
   🏺🐜 Archaeologist {archaeologist_name} spawning
       Scanning history of files to be modified...
   ```

   Spawn a Scout (using Task tool with `subagent_type="aether-archaeologist"`) with this prompt:
   # FALLBACK: If "Agent type not found", use general-purpose and inject role: "You are an Archaeologist Ant - git historian that excavates why code exists."

   ```
   You are {Archaeologist-Name}, a 🏺🐜 Archaeologist Ant.

   Mission: Pre-build archaeology scan

   Files: {list of existing files that will be modified}

   Work:
   1. Read each file to understand current state
   2. Run: git log --oneline -15 -- "{file_path}" for history
   3. Run: git log --all --grep="fix\|bug\|workaround\|hack\|revert" --oneline -- "{file_path}" for incident history
   4. Run: git blame "{file_path}" | head -40 for authorship
   5. Note TODO/FIXME/HACK markers

   Log activity: bash .aether/aether-utils.sh activity-log "READ" "{Ant-Name}" "description"

   Report (plain text):
   - WHY key code sections exist (from commits)
   - Known workarounds/hacks to preserve
   - Key architectural decisions
   - Areas of caution (high churn, reverts, emergencies)
   - Stable bedrock vs volatile sand sections
   ```

   **Wait for results** (blocking — use TaskOutput with `block: true`).

   Log completion and update swarm display:
   ```bash
   bash .aether/aether-utils.sh spawn-complete "{archaeologist_name}" "completed" "Pre-build archaeology scan"
   bash .aether/aether-utils.sh swarm-display-update "{archaeologist_name}" "scout" "completed" "Pre-build archaeology scan" "Queen" '{"read":8,"grep":5,"edit":0,"bash":2}' 100 "fungus_garden" 100
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

---

### Step 5: Initialize Swarm Display and Analyze Tasks

**YOU (the Queen) will spawn workers directly. Do NOT delegate to a single Prime Worker.**

**Initialize visual swarm tracking:**
```bash
# Generate unique build ID
build_id="build-$(date +%s)"

# Initialize swarm display for this build
bash .aether/aether-utils.sh swarm-display-init "$build_id"

# Log phase start
bash .aether/aether-utils.sh activity-log "EXECUTING" "Queen" "Phase {id}: {name} - Queen dispatching workers"

# Display animated header
bash .aether/aether-utils.sh swarm-display-update "Queen" "prime" "excavating" "Phase {id}: {name}" "Colony" '{"read":0,"grep":0,"edit":0,"bash":0}' 10 "fungus_garden" 0
```

**Show real-time display header:**
```
🔨  COLONY BUILD INITIATED
═══════════════════════════════════════════════════
Phase {id}: {name}
Build ID: {build_id}

Launching swarm display...
```

Analyze the phase tasks:

Analyze the phase tasks:

1. **Group tasks by dependencies:**
   - **Wave 1:** Tasks with `depends_on: "none"` or `depends_on: []` (can run in parallel)
   - **Wave 2:** Tasks depending on Wave 1 tasks
   - **Wave 3+:** Continue until all tasks assigned

2. **Assign castes:**
   - Implementation tasks → 🔨🐜 Builder
   - Research/docs tasks → 🔍🐜 Scout
   - Testing/validation → 👁️🐜 Watcher (ALWAYS spawn at least one)
   - Resilience testing → 🎲🐜 Chaos (ALWAYS spawn one after Watcher)

3. **Generate ant names for each worker:**
```bash
bash .aether/aether-utils.sh generate-ant-name "builder"
bash .aether/aether-utils.sh generate-ant-name "watcher"
bash .aether/aether-utils.sh generate-ant-name "chaos"
```

Display spawn plan with caste emojis:
```
🐜  SPAWN PLAN

Wave 1  — Parallel
  🔨🐜 {Builder-Name}  Task {id}  {description}
  🔨🐜 {Builder-Name}  Task {id}  {description}

Wave 2  — After Wave 1
  🔨🐜 {Builder-Name}  Task {id}  {description}

Verification
  👁️🐜 {Watcher-Name}  Verify all work independently
  🎲🐜 {Chaos-Name}   Resilience testing (after Watcher)

Total: {N} Builders + 1 Watcher + 1 Chaos = {N+2} spawns
```

**Caste Emoji Legend:**
- 🔨🐜 Builder  (cyan if color enabled)
- 👁️🐜 Watcher  (green if color enabled)
- 🎲🐜 Chaos    (red if color enabled)
- 🔍🐜 Scout    (yellow if color enabled)
- 🏺🐜 Archaeologist (magenta if color enabled)
- 🥚 Queen/Prime

**Every spawn must show its caste emoji.**

### Step 5.1: Spawn Wave 1 Workers (Parallel)

**CRITICAL: Spawn ALL Wave 1 workers in a SINGLE message using multiple Task tool calls.**

**First, mark build start in context:**
```bash
bash .aether/aether-utils.sh context-update build-start {phase_id} {wave_1_worker_count} {wave_1_task_count}
```

For each Wave 1 task, use Task tool with `subagent_type="aether-builder"` (DO NOT use run_in_background - multiple Task calls in a single message run in parallel and block until complete):

Log each spawn and update swarm display:
```bash
bash .aether/aether-utils.sh spawn-log "Queen" "builder" "{ant_name}" "{task_description}"
bash .aether/aether-utils.sh swarm-display-update "{ant_name}" "builder" "excavating" "{task_description}" "Queen" '{"read":0,"grep":0,"edit":0,"bash":0}' 0 "fungus_garden" 10
bash .aether/aether-utils.sh context-update worker-spawn "{ant_name}" "builder" "{task_description}"
```

**Builder Worker Prompt (CLEAN OUTPUT):**
```
You are {Ant-Name}, a 🔨🐜 Builder Ant.

Task {id}: {description}

Goal: "{colony_goal}"

{ archaeology_context if exists }

{ queen_wisdom_section if any wisdom exists }

{ pheromone_section if pheromone_section exists }

Work:
1. Read .aether/workers.md for Builder discipline
2. Implement task, write tests
3. Log activity: bash .aether/aether-utils.sh activity-log "ACTION" "{Ant-Name}" "description"
4. Update display: bash .aether/aether-utils.sh swarm-display-update "{Ant-Name}" "builder" "excavating" "current task" "Queen" '{"read":0,"grep":0,"edit":0,"bash":0}' {progress} "fungus_garden" 50

Spawn sub-workers ONLY if 3x complexity:
- Check: bash .aether/aether-utils.sh spawn-can-spawn {depth}
- Generate name: bash .aether/aether-utils.sh generate-ant-name "builder"
- Announce: "🐜 Spawning {child_name} for {reason}"
- Log: bash .aether/aether-utils.sh spawn-log "{Ant-Name}" "builder" "{child_name}" "{task}"

Return ONLY this JSON (no other text):
{"ant_name": "{Ant-Name}", "task_id": "{id}", "status": "completed|failed|blocked", "summary": "What you did", "files_created": [], "files_modified": [], "tests_written": [], "blockers": []}
```

**Queen Wisdom Section Template (injected only if wisdom exists):**
```
--- QUEEN WISDOM (Eternal Guidance) ---
{ if queen_philosophies: }
📜 Philosophies:
{queen_philosophies}
{ endif }
{ if queen_patterns: }
🧭 Patterns:
{queen_patterns}
{ endif }
{ if queen_redirects: }
⚠️ Redirects (AVOID these):
{queen_redirects}
{ endif }
{ if queen_stack_wisdom: }
🔧 Stack Wisdom:
{queen_stack_wisdom}
{ endif }
{ if queen_decrees: }
🏛️ Decrees:
{queen_decrees}
{ endif }
--- END QUEEN WISDOM ---
```

**Active Signals Section (injected if pheromones exist):**
```
--- ACTIVE SIGNALS (From User) ---

🎯 PRIORITIES (FOCUS):
{for each priority}
- {priority}
{endfor}

⚠️ CONSTRAINTS (REDIRECT - AVOID):
{for each constraint}
- {constraint.content}
{endfor}

--- END ACTIVE SIGNALS ---
```

### Step 5.2: Process Wave 1 Results

**Task calls return results directly (no TaskOutput needed).**

**As each worker result arrives, immediately display:**
```
✅ 🔨🐜 {Builder-Name} completed Task {id}
   📖{read_count} 🔍{grep_count} ✏️{edit_count} ⚡{bash_count}  {elapsed_time}
```

Log and update swarm display:
```bash
bash .aether/aether-utils.sh spawn-complete "{ant_name}" "completed" "{summary}"
bash .aether/aether-utils.sh swarm-display-update "{ant_name}" "builder" "completed" "{task_description}" "Queen" '{"read":5,"grep":3,"edit":2,"bash":1}' 100 "fungus_garden" 100
bash .aether/aether-utils.sh context-update worker-complete "{ant_name}" "completed"
```

**Parse each worker's JSON output to collect:** status, files_created, files_modified, blockers

**Visual Mode: Render live display (if enabled):**
If `visual_mode` is true, render the swarm display after all workers complete:
```bash
bash .aether/aether-utils.sh swarm-display-render "$build_id"
```

### Step 5.3: Spawn Wave 2+ Workers (Sequential Waves)

Repeat Step 5.1-5.2 for each subsequent wave, waiting for previous wave to complete.

### Step 5.4: Spawn Watcher for Verification

**MANDATORY: Always spawn a Watcher — testing must be independent.**

Spawn the Watcher using Task tool with `subagent_type="aether-watcher"` (DO NOT use run_in_background - task blocks until complete):

```bash
bash .aether/aether-utils.sh spawn-log "Queen" "watcher" "{watcher_name}" "Independent verification"
bash .aether/aether-utils.sh swarm-display-update "{watcher_name}" "watcher" "observing" "Verification in progress" "Queen" '{"read":0,"grep":0,"edit":0,"bash":0}' 0 "nursery" 50
```

**Watcher Worker Prompt (CLEAN OUTPUT):**
```
You are {Watcher-Name}, a 👁️🐜 Watcher Ant.

Verify all work done by Builders in Phase {id}.

Files to verify:
- Created: {list from builder results}
- Modified: {list from builder results}

Verification:
1. Check files exist (Read each)
2. Run build/type-check
3. Run tests if they exist
4. Check success criteria: {list}

Spawn sub-workers if needed:
- Log: bash .aether/aether-utils.sh spawn-log "{Watcher-Name}" "watcher" "{child}" "{task}"
- Announce: "🐜 Spawning {child} to investigate {issue}"

Return ONLY this JSON:
{"ant_name": "{Watcher-Name}", "verification_passed": true|false, "files_verified": [], "issues_found": [], "quality_score": N, "recommendation": "proceed|fix_required"}
```

### Step 5.5: Process Watcher Results

**Task call returns results directly (no TaskOutput needed).**

**Parse the Watcher's JSON response:** verification_passed, issues_found, quality_score, recommendation

**Store results for synthesis in Step 5.7**

**Update swarm display when Watcher completes:**
```bash
bash .aether/aether-utils.sh swarm-display-update "{watcher_name}" "watcher" "completed" "Verification complete" "Queen" '{"read":3,"grep":2,"edit":0,"bash":1}' 100 "nursery" 100
```

### Step 5.6: Spawn Chaos Ant for Resilience Testing

**After the Watcher completes, spawn a Chaos Ant to probe the phase work for edge cases and boundary conditions.**

Generate a chaos ant name and log the spawn:
```bash
bash .aether/aether-utils.sh generate-ant-name "chaos"
bash .aether/aether-utils.sh spawn-log "Queen" "chaos" "{chaos_name}" "Resilience testing of Phase {id} work"
bash .aether/aether-utils.sh swarm-display-update "{chaos_name}" "chaos" "probing" "Resilience testing" "Queen" '{"read":0,"grep":0,"edit":0,"bash":0}' 0 "refuse_pile" 75
```

**Retrieve existing flags for this phase** (to avoid duplicate findings):
```bash
bash .aether/aether-utils.sh flag-list --phase {phase_number}
```
Parse the result and extract unresolved flag titles into a list: `{existing_flag_titles}` (comma-separated titles from `.result.flags[].title`). If no flags exist, set `{existing_flag_titles}` to "None".

Spawn the Chaos Ant using Task tool with `subagent_type="aether-chaos"` (DO NOT use run_in_background - task blocks until complete):
# FALLBACK: If "Agent type not found", use general-purpose and inject role: "You are a Chaos Ant - resilience tester that probes edge cases and boundary conditions."

**Chaos Ant Prompt (CLEAN OUTPUT):**
```
You are {Chaos-Name}, a 🎲🐜 Chaos Ant.

Test Phase {id} work for edge cases and boundary conditions.

Files to test:
- {list from builder results}

Skip these known issues: {existing_flag_titles}

Rules:
- Max 5 scenarios
- Read-only (don't modify code)
- Focus: edge cases, boundaries, error handling

Return ONLY this JSON:
{"ant_name": "{Chaos-Name}", "scenarios_tested": 5, "findings": [{"id": 1, "category": "edge_case|boundary|error_handling", "severity": "critical|high|medium|low", "title": "...", "description": "..."}], "overall_resilience": "strong|moderate|weak", "summary": "..."}
```

### Step 5.7: Process Chaos Ant Results

**Task call returns results directly (no TaskOutput needed).**

**Parse the Chaos Ant's JSON response:** findings, overall_resilience, summary

**Store results for synthesis in Step 5.9**

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

Log chaos ant completion and update swarm display:
```bash
bash .aether/aether-utils.sh spawn-complete "{chaos_name}" "completed" "{summary}"
bash .aether/aether-utils.sh swarm-display-update "{chaos_name}" "chaos" "completed" "Resilience testing done" "Queen" '{"read":2,"grep":1,"edit":0,"bash":0}' 100 "refuse_pile" 100
```

### Step 5.8: Create Flags for Verification Failures

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

This ensures verification failures are persisted as blockers that survive context resets. Chaos Ant findings are flagged in Step 5.7.

### Step 5.9: Synthesize Results

**This step runs after all worker tasks have completed (Builders, Watcher, Chaos).**

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

**Error Handoff Update:**
If workers failed, update handoff with error context for recovery:
```bash
cat > .aether/HANDOFF.md << 'HANDOFF_EOF'
# Colony Session — Build Errors

## ⚠️ Build Status: ISSUES DETECTED
**Phase:** {phase_number} — {phase_name}
**Status:** Build completed with failures
**Updated:** $(date -u +%Y-%m-%dT%H:%M:%SZ)

## Failed Workers
{for each failed worker:}
- {ant_name}: {failure_summary}
{end for}

## Grave Markers Placed
{for each grave:}
- {file}: {caution_level} caution
{end for}

## Recovery Options
1. Review failures: Check `.aether/data/activity.log`
2. Fix and retry: `/ant:build {phase_number}`
3. Swarm fix: `/ant:swarm` for auto-repair
4. Manual fix: Address issues, then `/ant:continue`

## Session Note
Build completed but workers failed. Grave markers placed.
Review failures before advancing.
HANDOFF_EOF
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

### Step 6.5: Update Handoff Document

After synthesis is complete, update the handoff document with current state for session recovery:

```bash
# Update handoff with build results
jq -n \
  --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  --arg goal "$(jq -r '.goal' .aether/data/COLONY_STATE.json)" \
  --arg phase "$(jq -r '.current_phase' .aether/data/COLONY_STATE.json)" \
  --arg phase_name "{phase_name}" \
  --arg status "{synthesis.status}" \
  --arg summary "{synthesis.summary}" \
  --argjson tasks_completed '{synthesis.tasks_completed | length}' \
  --argjson tasks_failed '{synthesis.tasks_failed | length}' \
  --arg next_action "{if synthesis.status == "completed" then "/ant:continue" else "/ant:flags" end}" \
  '{
    "last_updated": $timestamp,
    "goal": $goal,
    "current_phase": $phase,
    "phase_name": $phase_name,
    "build_status": $status,
    "summary": $summary,
    "tasks_completed": $tasks_completed,
    "tasks_failed": $tasks_failed,
    "next_recommended_action": $next_action,
    "can_resume": true,
    "note": "Phase build completed. Run /ant:continue to advance if verification passed."
  }' > .aether/data/last-build-result.json

# Write handoff markdown
cat > .aether/HANDOFF.md << 'HANDOFF_EOF'
# Colony Session — Build Complete

## Quick Resume
Run `/ant:continue` to advance phase, or `/ant:resume-colony` to restore full context.

## State at Build Completion
- Goal: "$(jq -r '.goal' .aether/data/COLONY_STATE.json)"
- Phase: {phase_number} — {phase_name}
- Build Status: {synthesis.status}
- Updated: $(date -u +%Y-%m-%dT%H:%M:%SZ)

## Build Summary
{summary}

## Tasks
- Completed: {synthesis.tasks_completed | length}
- Failed: {synthesis.tasks_failed | length}

## Files Changed
- Created: {synthesis.files_created | length} files
- Modified: {synthesis.files_modified | length} files

## Next Steps
- If verification passed: `/ant:continue` to advance to next phase
- If issues found: `/ant:flags` to review blockers
- To pause: `/ant:pause-colony`

## Session Note
$(if synthesis.status == "completed" then "Build succeeded — ready to advance." else "Build completed with issues — review before continuing." end)
HANDOFF_EOF
```

This ensures the handoff always reflects the latest build state, even if the session crashes before explicit pause.

### Step 6.5: Update Context Document

Log this build activity to `.aether/CONTEXT.md`:

```bash
bash .aether/aether-utils.sh context-update activity "build {phase_id}" "{synthesis.status}" "{files_created_count + files_modified_count}"
```

Mark build as complete in context:
```bash
bash .aether/aether-utils.sh context-update build-complete "{synthesis.status}" "{synthesis.status == 'completed' ? 'success' : 'failed'}"
```

Also update safe-to-clear status:
- If build completed successfully: `context-update safe-to-clear "YES" "Build complete, ready to continue"`
- If build failed: `context-update safe-to-clear "NO" "Build failed — run /ant:swarm or /ant:flags"`

### Step 7: Display Results

**This step runs ONLY after synthesis is complete. All values come from actual worker results.**

**First, render the final swarm display showing all completed workers:**
```bash
# Final swarm display update - mark Queen as completed
bash .aether/aether-utils.sh swarm-display-update "Queen" "prime" "completed" "Phase {id} complete" "Colony" '{"read":10,"grep":5,"edit":5,"bash":2}' 100 "fungus_garden" 100

# Render the final swarm display
bash .aether/aether-utils.sh swarm-display-render "$build_id"
```

The swarm display will show:
- 🐜 All workers with their caste emojis (🔨🐜 Builder, 👁️🐜 Watcher, 🎲🐜 Chaos)
- 📖 Tool usage stats (Read, Grep, Edit, Bash counts)
- 🏠 Chamber activity map (Fungus Garden, Nursery, Refuse Pile)
- ✅ Progress bars at 100% for completed work
- 🌈 Color-coded by caste

**Then display build summary based on synthesis results AND `verbose_mode` from Step 1:**

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

🤖 Model Routing:
{for each spawn in spawn_tree where model_context exists:}
   {caste_emoji}{ant_name}: {model_context.assigned} {if model_context.assigned matches caste expectation:}✓{else}⚠️{end if}
{end for}
   Proxy: {if proxy_healthy:}✓ Healthy @ http://localhost:4000{else}✗ Not running (using default model){end if}

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

### Step 8: Update Session

Update the session tracking file to enable `/ant:resume` after context clear:

```bash
bash .aether/aether-utils.sh session-update "/ant:build {phase_id}" "/ant:continue" "Phase {phase_id} build completed: {synthesis.status}"
```
