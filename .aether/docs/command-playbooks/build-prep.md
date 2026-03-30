---
name: ant:build
description: "🔨🐜🏗️🐜🔨 Build a phase with pure emergence - colony self-organizes and completes tasks"
---

You are the **Queen**. You DIRECTLY spawn multiple workers — do not delegate to a single Prime Worker.

## Pheromone Suggestions

At build start, the colony analyzes the codebase for patterns that might benefit
from pheromone signals (complex files, TODOs, debug artifacts, etc.). Suggested
signals are presented for approval and written as FOCUS pheromones if approved.

Use `--no-suggest` to skip this analysis.

The phase to build is: `$ARGUMENTS`

## Instructions

### Context Confirmation Rule (CRITICAL)

**Before switching to a different colony directory, you MUST confirm with the user.**

When the user mentions a colony name that doesn't match the current directory:
1. State clearly: "You're currently in [Current Directory/Colony Name]"
2. Ask: "Did you mean to switch to [Other Colony], or continue here?"
3. Wait for confirmation — DO NOT proceed until the user clarifies

**Example:**
> User: "THE FUCKING ANIMATION COLONY DO THE NEXT BUILD"
>
> Assistant: "You're currently in SonoTherapie (ACF/WPML site). Did you mean:
> 1. Continue with SonoTherapie (which has animated hero sections)
> 2. Switch to AnalogWave (NEW SITE)
>
> Which would you like?"

This prevents accidental context switches when the user is frustrated or uses imprecise language.

<failure_modes>
### Wave Failure Mid-Build
If a worker fails during a build wave:
- Do NOT continue to next wave (failed dependencies will cascade)
- Report which worker failed, what task it was on, and what was attempted
- Options: (1) Retry the failed task, (2) Skip and continue with remaining tasks, (3) Abort build

### Partial File Writes
If a builder crashes mid-write:
- Check git status for uncommitted partial changes
- If partial changes exist, offer: (1) Review and keep, (2) Revert with git checkout, (3) Stash for later

### State Corruption
If COLONY_STATE.json becomes invalid during build:
- STOP all workers immediately
- Do not attempt to fix state automatically
- Report the issue and offer to restore from last known good state
</failure_modes>

<success_criteria>
Command is complete when:
- All waves executed in order with no skipped dependencies
- Each worker's task output is verified (files exist, tests pass)
- COLONY_STATE.json reflects completed phase progress
- Build summary reports all workers' outcomes
</success_criteria>

<read_only>
Do not touch during build:
- .aether/dreams/ (user notes)
- .aether/chambers/ (archived colonies)
- .env* files
- .claude/settings.json
- .github/workflows/
- Other agents' config files (only modify files assigned to the current build task)
</read_only>

### Step 0.6: Verify LiteLLM Proxy

Check that the LiteLLM proxy is running for model routing:

Run using the Bash tool with description "Checking model proxy...":
```bash
curl -s http://localhost:4000/health | grep -q "healthy" && echo "Proxy healthy" || echo "Proxy not running - workers will use default model"
```

If proxy is not healthy, log a warning but continue (workers will fall back to default routing).

### Step 0.5: Load Colony State

Run using the Bash tool with description "Loading colony state...": `bash .aether/aether-utils.sh load-state`

If the command fails (non-zero exit or JSON has ok: false):
1. Parse error JSON
2. If error code is E_FILE_NOT_FOUND: "No colony initialized. Run /ant:init first." and stop
3. If validation error: Display error details with recovery suggestion and stop
4. For other errors: Display generic error and suggest /ant:status for diagnostics

If successful:
1. Parse the state JSON from result field
2. Check if goal is null - if so: "No colony initialized. Run /ant:init first." and stop
3. Check if `milestone` == `"Crowned Anthill"` - if so: "This colony has been sealed. Start a new colony with `/ant:init \"new goal\"`." and stop
4. Extract current_phase and phase name from plan.phases[current_phase - 1].name
4. Display brief resumption context:
   ```
   🔄 Resuming: Phase X - Name
   ```
   (If HANDOFF.md exists, this provides orientation before the build proceeds)

After displaying context, run using the Bash tool with description "Releasing colony lock...": `bash .aether/aether-utils.sh unload-state` to release the lock.

### Step 1: Validate + Read State

**Parse $ARGUMENTS:**
1. Extract the phase number (first argument)
2. Check remaining arguments for flags:
   - If contains `--verbose` or `-v`: set `verbose_mode = true`
   - If contains `--no-visual`: set `visual_mode = false` (visual is ON by default)
   - If contains `--no-suggest`: set `suggest_enabled = false` (suggestions are ON by default)
   - If contains `--depth <level>`: set `cli_depth_override = <level>`
   - Otherwise: set `visual_mode = true`, `suggest_enabled = true` (defaults)

If the phase number is empty or not a number:

```
Usage: /ant:build <phase_number> [--verbose|-v] [--no-visual] [--no-suggest] [--depth <level>]

Options:
  --verbose, -v       Show full completion details (spawn tree, TDD, patterns)
  --no-visual         Disable real-time visual display (visual is on by default)
  --no-suggest        Skip pheromone suggestion analysis
  --depth <level>     Set colony depth for this build (light|standard|deep|full)

Examples:
  /ant:build 1              Build Phase 1 (with visual display)
  /ant:build 1 --verbose    Build Phase 1 (full details + visual)
  /ant:build 1 --no-visual  Build Phase 1 without visual display
  /ant:build 1 --no-suggest Build Phase 1 without pheromone suggestions
  /ant:build 1 --depth deep Build Phase 1 with thorough investigation
```

Stop here.

**Set colony depth (if --depth flag provided):**
If `cli_depth_override` is set:
1. Run using the Bash tool with description "Setting colony depth...": `bash .aether/aether-utils.sh colony-depth set "$cli_depth_override"`
2. Parse JSON result - if `.ok` is false:
   - Display: `Error: Invalid depth "$cli_depth_override". Use: light, standard, deep, full`
   - Stop here
3. If valid: Display `Colony depth: {level}`

**Read colony depth:**

Run using the Bash tool with description "Reading colony depth...":
```bash
depth_result=$(bash .aether/aether-utils.sh colony-depth get 2>/dev/null || echo '{"ok":true,"result":{"depth":"standard","source":"default"}}')
colony_depth=$(echo "$depth_result" | jq -r '.result.depth // "standard"')
depth_source=$(echo "$depth_result" | jq -r '.result.source // "default"')
echo "colony_depth=$colony_depth"
echo "depth_source=$depth_source"
```

Store `colony_depth` as cross-stage state for use by build-wave.md and build-verify.md.

Display depth with label:
```
Depth: {colony_depth} ({label})
```

Where label maps:
- light -> "Builder only -- fastest"
- standard -> "Builder + Scout -- balanced"
- deep -> "Builder + Scout + Oracle -- thorough"
- full -> "All agents -- most thorough"

If `colony_depth` is "standard" and `depth_source` is "default" (user never explicitly set it), also display:
```
(Tip: use --depth deep for Oracle research, or --depth light for fast builds)
```

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

Run using the Bash tool with description "Checking for blockers...":
```bash
bash .aether/aether-utils.sh flag-check-blockers {phase_number}
```

Parse the JSON result (`.result.blockers`):

- **If blockers == 0:** Display nothing (or optionally a brief `No active blockers for Phase {id}.` line). Proceed to Step 2.
- **If blockers > 0:** Retrieve blocker details:
  Run using the Bash tool with description "Loading blocker details...":
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

Validate the state file:
Run using the Bash tool with description "Validating colony state...":
```bash
bash .aether/aether-utils.sh validate-state colony
```

### Step 3: Git Checkpoint

Create a git checkpoint for rollback capability.

Run using the Bash tool with description "Checking git repository...":
```bash
git rev-parse --git-dir 2>/dev/null
```

- **If succeeds** (is a git repo):
  1. Check for changes in Aether-managed directories only: `.aether .claude/commands/ant .claude/commands/st .opencode bin`
  2. **If changes exist**: Run using the Bash tool with description "Creating git checkpoint...": `git stash push -m "aether-checkpoint: pre-phase-$PHASE_NUMBER" -- .aether .claude/commands/ant .claude/commands/st .opencode bin ":(exclude).aether/data/"`
     - IMPORTANT: Never use `--include-untracked` — it stashes ALL files including user work!
     - Run using the Bash tool with description "Verifying checkpoint...": `git stash list | head -1 | grep "aether-checkpoint"` — warn if empty
     - Store checkpoint as `{type: "stash", ref: "aether-checkpoint: pre-phase-$PHASE_NUMBER"}`
  3. **If clean working tree**: Run using the Bash tool with description "Recording HEAD position...": `git rev-parse HEAD`
     - Store checkpoint as `{type: "commit", ref: "$HEAD_HASH"}`
- **If fails** (not a git repo): Set checkpoint to `{type: "none", ref: "(not a git repo)"}`.

Rollback procedure: `git stash pop` (if type is "stash") or `git reset --hard $ref` (if type is "commit").

Output header:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   B U I L D I N G   P H A S E   {id}
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📍 Phase {id}: {name}
💾 Git checkpoint saved
```

Run using the Bash tool with description "Showing phase progress...":
```bash
progress_bar=$(bash .aether/aether-utils.sh generate-progress-bar "$current_phase" "$total_phases" 20 2>/dev/null || echo "")
if [[ -n "$progress_bar" ]]; then
  echo "[Phase ${current_phase}/${total_phases}] ${progress_bar}"
fi
```
