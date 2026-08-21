# Playbook: Build

> Consolidated from build-prep, build-context, build-wave, build-verify, and build-complete.

## Overview

The Queen directly spawns multiple workers to build a phase with pure emergence. The colony self-organizes and completes tasks in waves.

## Stage 1: Prep

### Step 1: Validate + Read State

Parse arguments:
- Extract phase number (first argument)
- Flags: `--verbose|-v`, `--no-visual`, `--no-suggest`, `--depth <level>`

If phase number is empty or not a number, show usage and stop.

Set colony depth if `--depth` provided. Read colony depth and parallel mode from state.

### Step 2: Update State

Update `COLONY_STATE.json` via `aether state-mutate`:
- Set `state` to `"EXECUTING"`
- Set `current_phase` to the phase number
- Set phase `status` to `"in_progress"`
- Add `build_started_at` timestamp

### Step 3: Git Checkpoint

Create a git checkpoint for rollback:
- If changes exist in Aether-managed directories: stash them
- If clean: record HEAD commit SHA

## Stage 2: Context

### Step 4: Load Colony Context (colony-prime)

Run `aether colony-prime --compact` to get unified worker context.

Display active pheromones table via `aether pheromone-display`.

### Step 4.0: Load Territory Survey

Load relevant survey documents based on phase type. (No `survey-load` subcommand exists under `aether` in the Go runtime — this step describes intent only, not a runnable command.)

### Step 4.1: Archaeologist Pre-Build Scan

If the phase modifies existing files, spawn an Archaeologist to scan git history, authorship, and known workarounds.

### Step 4.2: Suggest Pheromones (Deprecated)

Skipped gracefully. The `suggest-*` commands are deprecated.

### Step 4.3: Skill Detection

Build skills index and detect domain skills matching the codebase. Display: `Skills: {indexed} indexed, {matched} matched`.

## Stage 3: Wave

### Step 5: Analyze Tasks and Spawn Workers

Group tasks by dependencies into waves:
- Wave 1: tasks with no dependencies (parallel)
- Wave 2+: tasks depending on previous waves

Assign castes:
- Implementation -> Builder
- Research/docs -> Scout (at standard+ depth)
- Testing -> Watcher (always at least one)
- Resilience -> Chaos (always one after Watcher)

Generate ant names and display spawn plan with caste emojis.

### Step 5.1: Spawn Wave 1 Workers (Parallel)

Spawn ALL Wave 1 workers in a single message using multiple visible Task tool calls. Do not use `run_in_background`.

Per worker:
- Inject worktree context (if worktree mode)
- Inject graveyard caution context
- Skills are matched and injected automatically, in-process, while the brief is assembled (`skill-inject` was deleted in Phase 191 as dead CLI surface -- SKILL-01)
- Inject colony context (`prompt_section`)

Builder prompt returns JSON with: `ant_name`, `task_id`, `status`, `summary`, `tool_count`, `files_created`, `files_modified`, `tests_written`, `blockers`.

### Step 5.2: Process Wave 1 Results

Validate each worker response. Display completion line per worker immediately.

If ALL workers failed: halt build, display wave failure alert, skip to synthesis.

If SOME failed: attempt Tier 3 escalation (Queen spawns different caste). If still fails, display escalation banner.

### Step 5.3: Spawn Wave 2+ Workers

Repeat wave spawning for subsequent waves, waiting for previous wave to complete.

### Step 5.3.5: Builder-Probe Lock (Mandatory)

After all waves, no task may be marked `completed` without Probe verification.

For each `code_written` task, spawn a Probe to independently verify:
- Run tests
- Check files exist
- Verify coverage >= 80%

If Probe passes, mark task complete via `aether state-mutate --guard`.

## Stage 4: Verify

### Step 5.4: Spawn Watcher for Verification

Mandatory independent verification. Spawn Watcher with `subagent_type="aether-watcher"`.

Watcher verifies:
1. Files exist
2. Build/type-check passes
3. Tests pass
4. Success criteria met

### Step 5.5: Process Watcher Results

Parse watcher JSON: `verification_passed`, `issues_found`, `quality_score`, `recommendation`.

If verification failed, create blocker flags and log to midden.

### Step 5.5.1: Measurer Performance Agent (Conditional)

If phase is performance-sensitive and Watcher passed, spawn Measurer to establish baselines and identify bottlenecks.

### Step 5.6: Spawn Chaos Ant for Resilience Testing

After Watcher, spawn Chaos Ant to probe edge cases and boundary conditions. Max 5 scenarios, read-only.

### Step 5.7: Process Chaos Ant Results

Flag critical/high findings. Log resilience findings to midden.

### Step 5.8: Create Flags for Verification Failures

If Watcher reported failures, create blocker flags per issue and log to midden.

## Stage 5: Complete

### Step 5.9: Synthesize Results

Collect all worker outputs into phase summary JSON including:
- `status`, `summary`, `tasks_completed`, `tasks_failed`
- `files_created`, `files_modified`
- `spawn_metrics` and `spawn_tree`
- `verification`, `performance`, `resilience`

Persist builder claims to `.aether/data/last-build-claims.json`.

If workers failed, write error handoff to `.aether/HANDOFF.md`.

### Step 6: Visual Checkpoint (if UI touched)

If `ui_touched` is true, prompt user for visual approval.

### Step 6.5: Update Handoff Document

Write build success handoff to `.aether/HANDOFF.md` and `.aether/data/last-build-result.json`.

### Step 6.6: Update Context Document

Log build activity to `.aether/CONTEXT.md`.

### Step 7: Display Results

Show BUILD SUMMARY with workers, tools, duration, measurer/ambassador results if applicable.

Call `aether print-next-up` to route based on colony state.

### Step 8: Update Session

Run `aether session-update` to enable `/ant-resume` after context clear.
