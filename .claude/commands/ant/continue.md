---
name: ant:continue
description: Detect build completion, reconcile state, and advance to next phase
---

You are the **Queen Ant Colony**. Detect completed work and advance to the next phase.

## Instructions

### Step 0: Parse Arguments

Check if `$ARGUMENTS` contains `--all`: set `auto_mode = true`, otherwise `auto_mode = false`.

### Step 1: Read State + Detect Completion

Read `.aether/data/COLONY_STATE.json`. Extract: `goal`, `state`, `current_phase`, `mode`, `plan.phases`, `signals`, `errors`, `memory`, `events`, `build_started_at`.

**Validation:** If `goal: null` output "No colony initialized. Run /ant:init first." and stop. If `plan.phases` empty output "No project plan. Run /ant:plan first." and stop.

**Completion Detection (SIMP-07 output-as-state pattern):**

If `state == "EXECUTING"` -- a build ran, detect what completed:

1. **Primary signal:** Check if `.planning/phases/{current_phase}-*/SUMMARY.md` exists (Glob tool)
   - SUMMARY.md exists AND non-empty: phase complete
   - SUMMARY.md missing OR empty: phase incomplete

2. **Task-level detection:** For each task in `plan.phases[current_phase].tasks`, check if output file exists. If exists: mark "completed", if missing: mark "pending".

3. **Edge cases:** Empty SUMMARY.md = incomplete. No build_started_at = legacy state, proceed normally.

**Orphan State Handling:**

If `state == "EXECUTING"` but no completion signals:
- Stale (>30 min): Display "Stale EXECUTING state detected. Build may have been interrupted." Offer rollback to git checkpoint or continue.
- Recent (<30 min): Display "Build appears to still be running. Wait or force continue with --force."

If `state != "EXECUTING"`: Normal continue flow (no build to reconcile).

### Step 1.5: Auto-Continue Loop (only if auto_mode is true)

If `auto_mode` is false, skip to Step 2.

Calculate `remaining_phases` (status != "completed"). If none remain, output "All phases already complete." and skip to Step 3.

Display AUTO-CONTINUE banner with remaining count and halt conditions (score < 4, 2 consecutive failures).

**For each remaining phase:**
1. Display phase progress
2. Use Task tool to run build (auto-approve mode)
3. Check halt conditions, break if triggered
4. Run Steps 2-3 for this phase
5. Record and display result

After loop, display cumulative results and proceed to Step 3.

### Step 2: Update State (Full Reconciliation)

Determine next phase (`current_phase + 1`). If no next phase, skip to Step 2.5 (tech debt report).

Update COLONY_STATE.json with full reconciliation:

1. **Mark tasks:** Set status based on detection from Step 1
2. **Extract learnings:** Append to `memory.phase_learnings`: `{id, phase, phase_name, learnings: ["<specific actionable>"], errors_encountered, timestamp}`
3. **Update spawn_outcomes:** Increment alpha/successes or beta/failures for contributing castes
4. **Emit FEEDBACK pheromone:** `{type: "FEEDBACK", content: "<what worked/didn't>", priority: "normal", expires_at: "<6 hours from now ISO-8601>", source: "auto:continue", auto: true}`
5. **Emit REDIRECT if flagged_patterns exist:** `{type: "REDIRECT", content: "<pattern to avoid>", priority: "high", expires_at: "<24 hours from now ISO-8601>", source: "auto:continue", auto: true}`
6. **Clean expired pheromones:** Expired signals are filtered on read. No explicit cleanup needed.
7. **Advance state:** Set `current_phase` to next, `state` to "READY", workers to "idle", append phase_advanced event
8. **Write COLONY_STATE.json**
9. **Compress memory:** Memory compression handled by cap enforcement when writing (30 decisions max, 50 events max).

### Step 2.5: Tech Debt Report (Project Completion Only)

Runs ONLY when all phases complete (no next phase).

1. Gather: `errors.records`, `errors.flagged_patterns` from COLONY_STATE.json, read activity.log
2. Display TECH DEBT REPORT: project, phases completed, persistent issues, recommendations
3. Write to `.aether/data/tech-debt-report.md`

### Step 2.5b: Promote Learnings (Project Completion Only)

If `auto_mode`: Display "Global learning promotion available. Run /ant:continue to promote." and skip.

Otherwise: Analyze `memory.phase_learnings`, categorize as promotable vs project-specific, and display them for user to manually incorporate into their global learnings.

### Step 2.5c: Completion Message

Display completion message with tech debt report path and next commands (`/ant:status`, `/ant:plan`). Stop here.

### Step 3: Display Result

Output AETHER COLONY :: CONTINUE banner.

Display phase completion summary (tasks completed, error counts).

Display learnings extracted and auto-emitted pheromones.

Display next phase preview:
```
Phase <current> approved. Advancing to Phase <next>.
  Phase <next>: <name>
  <description>
  Tasks: <count>
  State: READY

Next Steps:
  /ant:build <next>      Start building Phase <next>
  /ant:phase <next>      Review phase details first
  /ant:focus "<area>"    Guide colony attention
```
