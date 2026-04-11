<!-- Generated from .aether/commands/build.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant:build
description: "🔨🐜🏗️🐜🔨 Build a phase with pure emergence - colony self-organizes and completes tasks"
---

You are the **Queen**. You DIRECTLY spawn multiple workers — do not delegate to a single Prime Worker.

The phase to build is: `$ARGUMENTS`

## Instructions

### Step 0.5: Load Colony State

Run using Bash tool: `aether load-state`

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

After displaying context, run: `aether unload-state` to release the lock.

### Step 1: Validate + Read State

**Parse $ARGUMENTS:**
1. Extract the phase number (first argument)
2. Check remaining arguments for flags:
   - If contains `--verbose` or `-v`: set `verbose_mode = true`
   - If contains `--no-visual`: set `visual_mode = false` (visual is ON by default)
   - If contains `--depth <level>`: set `cli_depth_override = <level>`
   - Otherwise: set `visual_mode = true` (visual is default)

If the phase number is empty or not a number:

```
Usage: /ant:build <phase_number> [--verbose|-v] [--no-visual] [--depth <level>]

Options:
  --verbose, -v       Show full completion details (spawn tree, TDD, patterns)
  --no-visual         Disable real-time visual display (visual is on by default)
  --depth <level>     Set colony depth for this build (light|standard|deep|full)

Examples:
  /ant:build 1              Build Phase 1 (with visual display)
  /ant:build 1 --verbose    Build Phase 1 (full details + visual)
  /ant:build 1 --no-visual  Build Phase 1 without visual display
  /ant:build 1 --depth deep Build Phase 1 with thorough investigation
```

Stop here.

**Set colony depth (if --depth flag provided):**
If `cli_depth_override` is set:
1. Run using Bash tool: `aether colony-depth set "$cli_depth_override"`
2. Parse JSON result - if `.ok` is false:
   - Display: `Error: Invalid depth "$cli_depth_override". Use: light, standard, deep, full`
   - Stop here
3. If valid: Display `Colony depth: {level}`

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
aether flag-check-blockers {phase_number}
```

Parse the JSON result (`.result.blockers`):

- **If blockers == 0:** Display nothing (or optionally a brief `No active blockers for Phase {id}.` line). Proceed to Step 2.
- **If blockers > 0:** Retrieve blocker details:
  ```bash
  aether flag-list --type blocker --json --phase {phase_number}
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
  1. Check for changes in Aether-managed directories only: `.aether .claude/commands/ant .claude/commands/st .opencode bin`
  2. **If changes exist**: `git stash push -m "aether-checkpoint: pre-phase-$PHASE_NUMBER" -- .aether .claude/commands/ant .claude/commands/st .opencode bin`
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

Run using the Bash tool with description "Showing phase progress...":
```bash
progress_bar=$(aether generate-progress-bar "$current_phase" "$total_phases" 20 2>/dev/null || echo "")
if [[ -n "$progress_bar" ]]; then
  echo "[Phase ${current_phase}/${total_phases}] ${progress_bar}"
fi
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
aether survey-load "{phase_name}" 2>/dev/null
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
━━━ 🗺️🐜 S U R V E Y   L O A D E D ━━━
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
aether queen-read 2>/dev/null
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
━━━ 📜🐜 Q U E E N   W I S D O M ━━━
{if queen_philosophies:}  📜 Philosophies: yes{/if}
{if queen_patterns:}  🧭 Patterns: yes{/if}
{if queen_redirects:}  ⚠️ Redirects: yes{/if}
{if queen_stack_wisdom:}  🔧 Stack Wisdom: yes{/if}
{if queen_decrees:}  🏛️ Decrees: yes{/if}

{if none exist:}  (no eternal wisdom recorded yet){/if}
```

**Graceful handling:** If QUEEN.md doesn't exist or `queen-read` fails, continue without wisdom injection. Workers will receive standard prompts.

### Step 4.1.6: Load Active Pheromones (Signal Consumption)

Call `pheromone-read` to extract active colony signals for worker priming:

```bash
aether pheromone-read 2>/dev/null
```

**Parse the JSON response:**
- If `.ok` is false or command fails: Set `pheromone_section = null` and skip pheromone injection
- If successful: Extract signals from `.result.signals`

**Active Signals Section Template (injected into builder prompts):**
```
--- ACTIVE SIGNALS (Pheromone Consumption) ---
{focus_section if .result.signals.focus exists:}
  🎯 FOCUS: {focus_description}
{redirect_section if .result.signals.redirect exists:}
  ⚠️ AVOID: {redirect_description}
{feedback_section if .result.signals.feedback exists:}
  💬 FEEDBACK: {feedback_description}
--- END SIGNALS ---
```

**Store for builder injection:**
- `pheromone_section` — formatted signal section for builder prompts

**Display summary (if any signals exist):**
```
━━━ 🦠🐜 P H E R O M O N E S   D E T E C T E D ━━━
{focus_present:}  🎯 Focus signal: yes{/if}
{redirect_present:}  ⚠️ Redirect signal: yes{/if}
{feedback_present:}  💬 Feedback signal: yes{/if}

{if none exist:}  (no active signals){/if}
```

**Graceful handling:** If pheromone-read fails or no signals exist, continue without pheromone injection.

---

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
   aether generate-ant-name --caste "archaeologist"
   aether spawn-log --name "Queen" --caste "scout" --id "{archaeologist_name}" --description "Pre-build archaeology scan"
   ```

   Display:
   ```
   🏺🐜 Archaeologist {archaeologist_name} spawning
       Scanning history of files to be modified...
   ```

   Spawn using Task tool with `subagent_type="aether-archaeologist"`, include `description: "🏺 Archaeologist {archaeologist_name}: Pre-build history scan"`:

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

   Log activity: aether activity-log --command "READ" --details "{Ant-Name}: description"

   Report (plain text):
   - WHY key code sections exist (from commits)
   - Known workarounds/hacks to preserve
   - Key architectural decisions
   - Areas of caution (high churn, reverts, emergencies)
   - Stable bedrock vs volatile sand sections
   ```

   **Wait for results** (blocking — use TaskOutput with `block: true`).

   Log completion:
   ```bash
   aether spawn-complete --name "{archaeologist_name}" --status "completed" --summary "Pre-build archaeology scan"
   ```

3. **Store and display findings:**

   Store the archaeologist's output as `archaeology_context`.

   Display summary:
   ```
   ━━━ 🏺🐜 A R C H A E O L O G Y ━━━
   {summary of findings from archaeologist}
   ```

4. **Injection into builder prompts:**
   The `archaeology_context` will be injected into builder prompts in Step 5.1 (see below).
   If this step was skipped (no existing files modified), the archaeology section is omitted from builder prompts.

---

### Step 5: Initialize Swarm Display and Analyze Tasks

**YOU (the Queen) will spawn workers directly. Do NOT delegate to a single Prime Worker.**

**Show build header:**
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Phase {id}: {name} — {N} waves, {M} tasks
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

Where N = number of builder waves (excluding watcher/chaos) and M = total builder tasks.

Record `build_started_at_epoch=$(date +%s)` — this epoch integer is used by the BUILD SUMMARY block in Step 7 to calculate elapsed time.

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
aether generate-ant-name --caste "builder"
aether generate-ant-name --caste "watcher"
aether generate-ant-name --caste "chaos"
```

Display spawn plan with caste emojis:
```
━━━ 🐜 S P A W N   P L A N ━━━

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

### Step 5.0.5: Select and Announce Workflow Pattern

Examine the phase name and task descriptions. Select the first matching pattern:

| Phase contains | Pattern |
|----------------|---------|
| "bug", "fix", "error", "broken", "failing" | Investigate-Fix |
| "research", "oracle", "explore", "investigate" | Deep Research |
| "refactor", "restructure", "clean", "reorganize" | Refactor |
| "security", "audit", "compliance", "accessibility", "license" | Compliance |
| "docs", "documentation", "readme", "guide" | Documentation Sprint |
| (default) | SPBV |

Display the selected pattern:
```
━━ Pattern: {pattern_name} ━━
{announce_line from Queen's Workflow Patterns definition}
```

Store `selected_pattern` for inclusion in the BUILD SUMMARY (Step 7).

### Step 5.1: Spawn Wave 1 Workers (Parallel)

**CRITICAL: Spawn ALL Wave 1 workers in a SINGLE message using multiple Task tool calls.**

**CRITICAL: Spawn ALL Wave 1 workers in a SINGLE message using multiple Task tool calls.**

**Announce the wave before spawning:**

Display the spawn announcement immediately before firing Task calls:

For single-caste waves (typical — all builders):
```
──── 🔨🐜 Spawning {N} Builders in parallel ────
```

For mixed-caste waves (uncommon):
```
──── 🐜 Spawning {N} workers ({X} 🔨 Builder, {Y} 🔍 Scout) ────
```

For a single worker:
```
──── 🔨🐜 Spawning {ant_name} — {task_summary} ────
```

**First, mark build start in context:**
```bash
aether context-update build-start {phase_id} {wave_1_worker_count} {wave_1_task_count}
```

Before dispatching each worker, refresh colony context so new pheromones/memory are visible:
```bash
prime_result=$(aether colony-prime --compact 2>/dev/null)
```
Update `prompt_section` from `prime_result.result.prompt_section`.

For each Wave 1 task, use Task tool with `subagent_type="aether-builder"`, include `description: "🔨 Builder {Ant-Name}: {task_description}"` (DO NOT use run_in_background - multiple Task calls in a single message run in parallel and block until complete):

Log each spawn and update swarm display:
```bash
aether spawn-log --name "Queen" --caste "builder" --id "{ant_name}" --description "{task_description}"
aether context-update worker-spawn "{ant_name}" "builder" "{task_description}"
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
3. Log activity: aether activity-log --command "ACTION" --details "{Ant-Name}: description"

Spawn sub-workers ONLY if 3x complexity:
- Check: aether spawn-can-spawn {depth} --enforce
- Generate name: aether generate-ant-name --caste "builder"
- Announce: "🐜 Spawning {child_name} for {reason}"
- Log: aether spawn-log --name "{Ant-Name}" --caste "builder" --id "{child_name}" --description "{task}"

Count your total tool calls (Read + Grep + Edit + Bash + Write) and report as tool_count.

Return ONLY this JSON (no other text):
{"ant_name": "{Ant-Name}", "task_id": "{id}", "status": "completed|failed|blocked", "summary": "What you did", "tool_count": 0, "files_created": [], "files_modified": [], "tests_written": [], "blockers": []}
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

Before using any worker payload, validate schema:
```bash
aether validate-worker-response builder '{worker_json}'
```
If validation fails, treat the worker as failed with blocker `invalid_worker_response`.

**As each worker result arrives, IMMEDIATELY display a single completion line — do not wait for other workers:**

For successful workers:
```
🔨 {Ant-Name}: {task_description} ({tool_count} tools) ✓
```

For failed workers:
```
🔨 {Ant-Name}: {task_description} ✗ ({failure_reason} after {tool_count} tools)
```

Where `tool_count` comes from the worker's returned JSON `tool_count` field, and `failure_reason` is extracted from the first item in the worker's `blockers` array or "unknown error" if empty.

Log and update swarm display:
```bash
aether spawn-complete --name "{ant_name}" --status "completed" --summary "{summary}"
aether context-update worker-complete "{ant_name}" "completed"
```

**Check for total wave failure:**

After processing all worker results in this wave, check if EVERY worker returned `status: "failed"`. If ALL workers in the wave failed:

Display a prominent halt alert:
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ⚠ WAVE FAILURE — BUILD HALTED
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

All {N} workers in Wave {X} failed. Something is fundamentally wrong.

Failed workers:
  {for each failed worker in this wave:}
  {caste_emoji} {Ant-Name}: {task_description} ✗ ({failure_reason} after {tool_count} tools)
  {end for}

Next steps:
  /ant:flags      Review blockers
  /ant:swarm      Auto-repair mode
```

Then STOP — do not proceed to subsequent waves, Watcher, or Chaos. Skip directly to Step 5.9 synthesis with `status: "failed"`.

**Partial wave failure — escalation path:**

If SOME (but not all) workers in the wave failed:
1. For each failed worker, attempt Tier 3 escalation: Queen spawns a different caste for the same task
2. If Tier 3 succeeds: continue to next wave
3. If Tier 3 fails: display the Tier 4 ESCALATION banner (from Queen agent definition):

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ⚠ ESCALATION — QUEEN NEEDS YOU
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Task: {failed task description}
Phase: {phase number} — {phase name}

Tried:
  • Worker retry (2 attempts) — {what failed}
  • Parent tried alternate approach — {what failed}
  • Queen reassigned to {other caste} — {what failed}

Options:
  A) {recommended option} — RECOMMENDED
  B) {alternate option}
  C) Skip and continue — this task will be marked blocked

Awaiting your choice.
```

Log escalation as flag:
```bash
aether flag-add --severity "blocker" --title "{task title}" --description "{failure summary}" --source "escalation" --phase {phase_number}
```

If at least one worker succeeded, continue normally to the next wave.

**Parse each worker's validated JSON output to collect:** status, files_created, files_modified, blockers

### Step 5.3: Spawn Wave 2+ Workers (Sequential Waves)

**Before each subsequent wave, display a wave separator:**
```
━━━ 🐜 Wave {X} of {N} ━━━
```
Then display the spawn announcement (same format as Step 5.1).

Repeat Step 5.1-5.2 for each subsequent wave, waiting for previous wave to complete.

### Step 5.4: Spawn Watcher for Verification

**MANDATORY: Always spawn a Watcher — testing must be independent.**

**Announce the verification wave:**
```
━━━ 👁️🐜 V E R I F I C A T I O N ━━━
──── 👁️🐜 Spawning {watcher_name} — Independent verification ────
```

Spawn the Watcher using Task tool with `subagent_type="aether-watcher"`, include `description: "👁️ Watcher {Watcher-Name}: Independent verification"` (DO NOT use run_in_background - task blocks until complete):

```bash
aether spawn-log --name "Queen" --caste "watcher" --id "{watcher_name}" --description "Independent verification"
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
- Log: aether spawn-log --name "{Watcher-Name}" --caste "watcher" --id "{child}" --description "{task}"
- Announce: "🐜 Spawning {child} to investigate {issue}"

Count your total tool calls (Read + Grep + Edit + Bash + Write) and report as tool_count.

Return ONLY this JSON:
{"ant_name": "{Watcher-Name}", "verification_passed": true|false, "files_verified": [], "issues_found": [], "quality_score": N, "tool_count": 0, "recommendation": "proceed|fix_required"}
```

### Step 5.5: Process Watcher Results

**Task call returns results directly (no TaskOutput needed).**

Validate watcher payload first:
```bash
aether validate-worker-response watcher '{watcher_json}'
```

**Parse the Watcher's validated JSON response:** verification_passed, issues_found, quality_score, recommendation

**Display Watcher completion line:**

For successful verification:
```
👁️ {Watcher-Name}: Independent verification ({tool_count} tools) ✓
```

For failed verification:
```
👁️ {Watcher-Name}: Independent verification ✗ ({issues_found count} issues after {tool_count} tools)
```

**Store results for synthesis in Step 5.7**

### Step 5.6: Spawn Chaos Ant for Resilience Testing

**After the Watcher completes, spawn a Chaos Ant to probe the phase work for edge cases and boundary conditions.**

Generate a chaos ant name and log the spawn:
```bash
aether generate-ant-name --caste "chaos"
aether spawn-log --name "Queen" --caste "chaos" --id "{chaos_name}" --description "Resilience testing of Phase {id} work"
```

**Retrieve existing flags for this phase** (to avoid duplicate findings):
```bash
aether flag-list --phase {phase_number}
```
Parse the result and extract unresolved flag titles into a list: `{existing_flag_titles}` (comma-separated titles from `.result.flags[].title`). If no flags exist, set `{existing_flag_titles}` to "None".

**Announce the resilience testing wave:**
```
──── 🎲🐜 Spawning {chaos_name} — resilience testing ────
```

Spawn the Chaos Ant using Task tool with `subagent_type="aether-chaos"`, include `description: "🎲 Chaos {Chaos-Name}: Resilience testing"` (DO NOT use run_in_background - task blocks until complete):

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

Count your total tool calls (Read + Grep + Edit + Bash + Write) and report as tool_count.

Return ONLY this JSON:
{"ant_name": "{Chaos-Name}", "scenarios_tested": 5, "findings": [{"id": 1, "category": "edge_case|boundary|error_handling", "severity": "critical|high|medium|low", "title": "...", "description": "..."}], "overall_resilience": "strong|moderate|weak", "tool_count": 0, "summary": "..."}
```

### Step 5.7: Process Chaos Ant Results

**Task call returns results directly (no TaskOutput needed).**

**Parse the Chaos Ant's JSON response:** findings, overall_resilience, summary

**Display Chaos completion line:**
```
🎲 {Chaos-Name}: Resilience testing ({tool_count} tools) ✓
```

**Store results for synthesis in Step 5.9**

**Flag critical/high findings:**

If any findings have severity `"critical"` or `"high"`:
```bash
# Create a blocker flag for each critical/high chaos finding
aether flag-add --severity "blocker" --title "{finding.title}" --description "{finding.description}" --source "chaos-testing" --phase {phase_number}
```

Log the flag:
```bash
aether activity-log --command "FLAG" --details "Chaos: Created blocker: {finding.title}"
```

Log chaos ant completion and update swarm display:
```bash
aether spawn-complete --name "{chaos_name}" --status "completed" --summary "{summary}"
```

### Step 5.8: Create Flags for Verification Failures

If the Watcher reported `verification_passed: false` or `recommendation: "fix_required"`:

For each issue in `issues_found`:
```bash
# Create a blocker flag for each verification failure
aether flag-add --severity "blocker" --title "{issue_title}" --description "{issue_description}" --source "verification" --phase {phase_number}
```

Log the flag creation:
```bash
aether activity-log --command "FLAG" --details "Watcher: Created blocker: {issue_title}"
```

This ensures verification failures are persisted as blockers that survive context resets. Chaos Ant findings are flagged in Step 5.7.

### Stage Audit Gate (Pre-Synthesis Check)

**This gate runs before Step 5.9. All build stages must have completed before synthesizing results.**

Verify that each of the following stages has a recorded completion status:

| Stage | Steps | Required |
|-------|-------|----------|
| Builder waves | Steps 5.1–5.3 | At least 1 worker completed (status "completed") |
| Watcher | Steps 5.4–5.5 | Watcher returned a result (any status) |
| Chaos | Steps 5.6–5.7 | Chaos ant returned a result (any status) |

Check by reviewing the in-memory worker results accumulated during Steps 5.1–5.7:
- `builder_results` — results from all builder wave tasks
- `watcher_result` — result from Step 5.5
- `chaos_result` — result from Step 5.7

**If any stage result is absent (stage did not run or returned no result):**
- HALT — do not proceed to Step 5.9
- Display:
  ```
  Stage Audit FAILED — cannot synthesize results.

  Missing or incomplete stages:
    {list each missing stage: "Builders — no results recorded" / "Watcher — did not complete" / "Chaos — did not complete"}

  Recovery options:
    1. Re-run /ant:build to restart this phase
    2. Run /ant:flags to review blockers
    3. Run /ant:swarm to auto-repair failed tasks
  ```
- Return `{"status": "failed", "summary": "Stage audit failed — stages did not complete"}` and stop.

**If all stages have results (even if some workers failed):** proceed to Step 5.9.

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
aether grave-add "{file}" "{ant_name}" "{task_id}" {phase} "{first blocker or summary}"
```
  Log the grave marker:
```bash
aether activity-log --command "GRAVE" --details "Queen: Grave marker placed at {file} — {ant_name} failed: {summary}"
```

**Error Handoff Update:**
If workers failed, update handoff with error context for recovery:

Resolve the build error handoff template path:
  Check ~/.aether/system/templates/handoff-build-error.template.md first,
  then .aether/templates/handoff-build-error.template.md.

If no template found: output "Template missing: handoff-build-error.template.md. Run aether update to fix." and stop.

Read the template file. Fill all {{PLACEHOLDER}} values:
  - {{PHASE_NUMBER}} → current phase number
  - {{PHASE_NAME}} → current phase name
  - {{BUILD_TIMESTAMP}} → current ISO-8601 UTC timestamp
  - {{FAILED_WORKERS}} → formatted list of failed workers (one "- {ant_name}: {failure_summary}" per line)
  - {{GRAVE_MARKERS}} → formatted list of grave markers (one "- {file}: {caution_level} caution" per line)

Remove the HTML comment lines at the top of the template.
Write the result to .aether/HANDOFF.md using the Write tool.

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
━━━ 🖼️🐜 V I S U A L   C H E C K P O I N T ━━━

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
```

Resolve the build success handoff template path:
  Check ~/.aether/system/templates/handoff-build-success.template.md first,
  then .aether/templates/handoff-build-success.template.md.

If no template found: output "Template missing: handoff-build-success.template.md. Run aether update to fix." and stop.

Read the template file. Fill all {{PLACEHOLDER}} values:
  - {{GOAL}} → colony goal (from COLONY_STATE.json)
  - {{PHASE_NUMBER}} → current phase number
  - {{PHASE_NAME}} → current phase name
  - {{BUILD_STATUS}} → synthesis.status
  - {{BUILD_TIMESTAMP}} → current ISO-8601 UTC timestamp
  - {{BUILD_SUMMARY}} → synthesis summary
  - {{TASKS_COMPLETED}} → count of completed tasks
  - {{TASKS_FAILED}} → count of failed tasks
  - {{FILES_CREATED}} → count of created files
  - {{FILES_MODIFIED}} → count of modified files
  - {{SESSION_NOTE}} → "Build succeeded — ready to advance." if status is completed, else "Build completed with issues — review before continuing."

Remove the HTML comment lines at the top of the template.
Write the result to .aether/HANDOFF.md using the Write tool.

This ensures the handoff always reflects the latest build state, even if the session crashes before explicit pause.

### Step 6.6: Update Context Document

Log this build activity to `.aether/CONTEXT.md`:

```bash
aether context-update activity "build {phase_id}" "{synthesis.status}" "{files_created_count + files_modified_count}"
```

Mark build as complete in context:
```bash
aether context-update build-complete "{synthesis.status}" "{synthesis.status == 'completed' ? 'success' : 'failed'}"
```

Also update safe-to-clear status:
- If build completed successfully: `context-update safe-to-clear "YES" "Build complete, ready to continue"`
- If build failed: `context-update safe-to-clear "NO" "Build failed — run /ant:swarm or /ant:flags"`

### Step 7: Display Results

**This step runs ONLY after synthesis is complete. All values come from actual worker results.**

**Display BUILD SUMMARY (always shown, replaces compact/verbose split):**

Calculate `total_tools` by summing `tool_count` from all worker return JSONs (builders + watcher + chaos).
Calculate `elapsed` using `build_started_at_epoch` (epoch integer captured at Step 5 start): `$(( $(date +%s) - build_started_at_epoch ))` formatted as Xm Ys.

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   B U I L D   S U M M A R Y
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Phase {id}: {name}
Pattern:  {selected_pattern}

Workers:  {pass_count} passed  {fail_count} failed  ({total} total)
Tools:    {total_tools} calls across all workers
Duration: {elapsed}

{if fail_count > 0:}
Failed:
  {for each failed worker:}
  {caste_emoji} {Ant-Name}: {task_description} ✗ ({failure_reason} after {tool_count} tools)
  {end for}

Retry: /ant:swarm to auto-repair failed tasks, or /ant:flags to review blockers
{end if}
```

**If verbose_mode is true**, additionally show the spawn tree and TDD details after the BUILD SUMMARY block (keep the existing verbose-only sections: Colony Work Tree, Tasks Completed, TDD, Patterns Learned, Debugging, Model Routing). Prepend with:
```
━━ Details (--verbose) ━━
```

After displaying the BUILD SUMMARY (and optional verbose details), display Next Steps based on synthesis results:
```bash
state=$(jq -r '.state // "IDLE"' .aether/data/COLONY_STATE.json 2>/dev/null || echo "IDLE")
current_phase=$(jq -r '.current_phase // 0' .aether/data/COLONY_STATE.json 2>/dev/null || echo "0")
total_phases=$(jq -r '.plan.phases | length' .aether/data/COLONY_STATE.json 2>/dev/null || echo "0")
aether print-next-up
```

**Routing Note:** The state-based Next Up block above routes based on colony state. If verification failed or blockers exist, review `/ant:flags` before continuing.

**IMPORTANT:** Build does NOT update task statuses or advance state. Run `/ant:continue` to:
- Mark tasks as completed
- Extract learnings
- Advance to next phase

### Step 8: Update Session

Update the session tracking file to enable `/ant:resume` after context clear:

```bash
aether session-update --command "/ant:build {phase_id}" --suggested-next "/ant:continue" --summary "Phase {phase_id} build completed: {synthesis.status}"
```
