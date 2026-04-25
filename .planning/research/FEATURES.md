# Feature Landscape: `aether recover` Stuck-State Detection and Fix

**Domain:** Colony lifecycle recovery for Aether Go CLI
**Researched:** 2026-04-25
**Confidence:** HIGH (all findings verified against source code)

## Table Stakes

Features users expect from a "colony got stuck, fix it" command. Missing = the command feels incomplete or dangerous.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Detect all 7 stuck classes | User ran the command to find out what is wrong | Medium | Each class has distinct disk markers |
| Explain in plain English | Non-technical users need to understand what broke | Low | Status command pattern already exists |
| Safe auto-fix with `--apply` | User expects one button to fix everything | Medium | Backup-first pattern from medic repair |
| Dry-run by default | User needs to see what would change before committing | Low | Medic already runs read-only by default |
| Non-destructive guarantees | User must not lose data | Medium | Backup before any write, like medic |
| Clear next-step output | User needs to know what to do after recovery | Low | `renderNextUp` pattern exists |

## The 7 Stuck States: Detection and Fix

### 1. Missing Build Packet

**What it looks like on disk:**
- `COLONY_STATE.json` has `state: "EXECUTING"` or `state: "BUILT"` and `current_phase > 0`
- `build_started_at` is non-nil and points to a real timestamp
- The build directory `.aether/data/build/phase-{N}/manifest.json` does not exist
- OR `manifest.json` exists but has `dispatches: []` (empty dispatches)
- OR `manifest.json` has `"plan_only": true` (never actually executed)

**Detection criteria:**
```
state.State == EXECUTING || state.State == BUILT
state.CurrentPhase > 0
manifest at .aether/data/build/phase-{currentPhase}/manifest.json is missing or has no real dispatches
```

**Fix behavior (`--apply`):**
- Safe auto-fix: YES
- Action: Reset state to READY, keep `current_phase` pointing at the stuck phase
- Reset `build_started_at` to nil
- Set `state.State = READY`
- Set the stuck phase status to `"ready"` (so build can re-dispatch)
- Emit event: `build_packet_missing|recover|Reset to READY for phase N`
- The user then runs `aether build N` to re-dispatch

**User confirmation needed:** NO -- this is purely a state reset with no data loss

---

### 2. Stale Spawned Workers

**What it looks like on disk:**
- `spawn-runs.json` has a run with `status: "active"` and `started_at` older than 1 hour
- OR `spawn-tree.txt` has entries with `status: "spawned"` or `status: "active"` that belong to the current run but have no matching completion line
- The colony state is EXECUTING or BUILT (implying workers should have finished)
- `spawn-track.json` exists and references an agent no longer running

**Detection criteria:**
```
spawn-runs.json: current_run_id points to a run with status "active" and started_at > 1 hour ago
spawn-tree.txt: entries with IsLiveSpawnStatus() in the current run window
COLONY_STATE.json: state is EXECUTING or BUILT AND build_started_at is > 1 hour ago
```

**Fix behavior (`--apply`):**
- Safe auto-fix: YES
- Action: Mark stale runs as "failed" in `spawn-runs.json`
- Mark stale spawn entries as "timeout" in `spawn-tree.txt`
- Clear `spawn-track.json` if present
- Reset `current_run_id` to empty
- Do NOT reset colony state here (that is the missing-build-packet fix's job)
- Emit event: `stale_workers_cleaned|recover|Marked N stale workers as timeout`

**User confirmation needed:** NO -- these workers are clearly dead

**Existing similar code:** `repairDataIssues` in `medic_repair.go` already handles `"reset_stale_spawn_state"` but only resets runs older than 1 hour. The recover command should use the same threshold.

---

### 3. Partial Phase

**What it looks like on disk:**
- `COLONY_STATE.json` has `state: "EXECUTING"` and `current_phase: N`
- The phase N has `status: "in_progress"`
- Build directory `build/phase-N/manifest.json` exists with real dispatches
- But all dispatches are either `"completed"` or `"failed"` (nothing active)
- AND no `continue.json` exists at `build/phase-N/continue.json`
- This means the build finished but continue was never run

**Alternative scenario:**
- `COLONY_STATE.json` has `state: "EXECUTING"` but `build_started_at` is nil
- Phase is `"in_progress"` but no build artifacts exist at all
- The phase was marked in_progress during init or plan but never actually built

**Detection criteria:**
```
state.State == EXECUTING
state.CurrentPhase > 0
Phase has status "in_progress"
manifest exists and all dispatches are terminal (no "spawned"/"active")
No continue.json exists for this phase
```

**Fix behavior (`--apply`):**
- Safe auto-fix: YES (with nuance)
- Action for "build finished but no continue":
  - Set `state.State = BUILT` (so the user can run `aether continue`)
  - Emit event: `partial_phase_recovered|recover|Phase N build completed but never continued`
- Action for "phase marked but never built":
  - Reset phase status to `"ready"`
  - Reset `state.State = READY`
  - Clear `build_started_at`
  - Emit event: `partial_phase_reset|recover|Phase N was never actually built`

**User confirmation needed:** NO for the first case. YES for the second if any tasks show `TaskInProgress` (means something may have partially run).

---

### 4. Bad Manifest

**What it looks like on disk:**
- `.aether/data/build/phase-N/manifest.json` exists but fails JSON parsing
- OR `manifest.json` parses but has `phase: 0` when `current_phase` says N
- OR `manifest.json` has `state: ""` or `generated_at: ""`
- OR `manifest.json` dispatches reference tasks that do not exist in the phase plan
- OR `manifest.json` has duplicate worker names in dispatches

**Detection criteria:**
```
manifest.json at build/phase-{currentPhase}/ is:
  - Not valid JSON
  - Has phase field mismatching current_phase
  - Has empty generated_at
  - Has dispatches referencing non-existent task IDs
```

**Fix behavior (`--apply`):**
- Safe auto-fix: PARTIAL
- Action for corrupted JSON:
  - Requires `--force` flag (destructive repair)
  - Remove the bad manifest
  - Fall through to "missing build packet" fix
- Action for mismatched/empty fields:
  - Regenerate manifest from current state + dispatch records in spawn-tree
  - If spawn-tree is also empty, fall through to "missing build packet"
- Action for phantom task references:
  - Remove dispatches with unknown task IDs
  - This is safe because those tasks do not exist in the plan

**User confirmation needed:** YES for corrupted JSON (data loss possible). NO for field mismatches (regeneration is safe).

**Existing similar code:** `findLastValidJSON` in `medic_repair.go` attempts JSON recovery. `repairDataIssues` handles corrupted JSON with `--force`.

---

### 5. Dirty Worktree

**What it looks like on disk:**
- `COLONY_STATE.json` has worktree entries with `status: "allocated"` or `status: "in-progress"`
- The worktree path `.aether/worktrees/{branch}` exists on disk
- `git status --porcelain` in the worktree shows uncommitted changes
- OR the worktree entry exists in state but the git worktree has been removed externally
- OR orphaned branches matching `phase-N/name` pattern exist with no worktree

**Detection criteria:**
```
COLONY_STATE.json worktrees[] with status != "merged" and status != "orphaned"
For each: check if worktree path exists on disk
For each on-disk worktree: check git status --porcelain for uncommitted changes
For each: check if git worktree list includes the path
Cross-reference: find git branches matching agentBranchRe that have no worktree and no state entry
```

**Fix behavior (`--apply`):**
- Safe auto-fix: NO (requires user confirmation)
- Action for worktree with uncommitted changes:
  - **Prompt user**: "Worktree at {path} has uncommitted changes. Stash and clean, or keep?"
  - If user says clean: `git stash` in the worktree, then remove worktree
  - If user says keep: skip this worktree, mark as manual-reconciliation needed
- Action for worktree entry with no backing git worktree:
  - Mark entry as `orphaned` in COLONY_STATE.json (safe, already handled by medic)
  - This is safe auto-fix
- Action for orphaned branches:
  - List them in output but do NOT auto-delete
  - User runs `git branch -D phase-N/name` manually

**User confirmation needed:** YES for anything with uncommitted changes. NO for stale state entries.

**Existing similar code:** `worktreeOrphanScanCmd` in `worktree.go` already does orphan detection. `repairStateIssues` in `medic_repair.go` removes orphaned entries. `reportOrphanBranches` detects agent-track branches with no worktree.

---

### 6. Broken Survey

**What it looks like on disk:**
- `.aether/data/survey/` directory exists (colony was colonized)
- One or more expected survey files are missing:
  - `blueprint.json`, `chambers.json`, `disciplines.json`, `provisions.json`, `pathogens.json`
- OR a survey file exists but contains invalid JSON
- OR a survey file is `{}` or `null` (empty, never populated)

**Detection criteria:**
```
survey/ directory exists
For each of the 5 expected files:
  - File missing
  - File is not valid JSON
  - File is empty ({}, null, [])
```

**Fix behavior (`--apply`):**
- Safe auto-fix: PARTIAL
- Action for missing or empty files:
  - Cannot regenerate survey data (it came from the colonize command)
  - Remove the broken file
  - Create a blocker flag: "Survey data incomplete for {name}. Re-run aether colonize."
  - This is informational: the colony can still function without survey data
- Action for corrupted JSON:
  - Requires `--force`
  - Attempt JSON recovery with `findLastValidJSON`
  - If recovery fails, remove the file and create blocker flag
- This is a LOW severity stuck state -- survey data is advisory, not blocking

**User confirmation needed:** NO for informational flagging. YES for destructive file removal.

**Existing similar code:** `surveyVerifyCmd` in `survey.go` checks all 5 files for existence and valid JSON. The `surveyFiles` list is already defined.

---

### 7. Missing Agent Files

**What it looks like on disk:**
- `.claude/agents/ant/aether-{caste}.md` is missing for one or more of the 25 castes
- OR `.opencode/agents/aether-{caste}.md` is missing
- OR `.codex/agents/aether-{caste}.toml` is missing
- Hub source files at `~/.aether/system/` are also checked -- if hub has the files, this is fixable by re-sync

**Detection criteria:**
```
For each of the 25 expected agent names:
  Check .claude/agents/ant/aether-{name}.md exists
  Check .opencode/agents/aether-{name}.md exists
  Check .codex/agents/aether-{name}.toml exists
Report missing files per platform
Cross-check hub at ~/.aether/system/claude/agents/, ~/.aether/system/opencode/agents/, ~/.aether/system/codex/
If hub has the file but local does not: fixable
If hub also missing: needs aether publish from source repo
```

**Fix behavior (`--apply`):**
- Safe auto-fix: YES (if hub has the source files)
- Action for fixable missing files:
  - Copy from hub to local directories
  - This is exactly what `aether update` does
  - Report which files were restored
- Action for hub-missing files:
  - Create blocker flag: "Agent files missing from hub. Run aether publish from the Aether source repo."
  - Cannot auto-fix without the source files

**User confirmation needed:** NO -- this is a file sync operation with no data loss risk.

**Existing similar code:** `aether update` in `update_cmd.go` handles the sync pairs. `scanWrapperParity` in `medic_scanner.go` (deep scan) checks wrapper parity. The install command `install_cmd.go` copies agent files during initial setup.

## Feature Dependencies

```
Stale Spawned Workers (2) must be fixed BEFORE Missing Build Packet (1)
  -- stale workers would confuse the build re-dispatch

Missing Build Packet (1) and Partial Phase (3) are mutually exclusive
  -- check for manifest first; if present, it's partial phase; if absent, missing packet

Dirty Worktree (5) is independent but should be fixed BEFORE re-running build
  -- dirty worktrees can block new worktree allocation

Bad Manifest (4) should be checked before Partial Phase (3)
  -- a corrupted manifest means the "partial phase" check is unreliable

Missing Agent Files (7) should be checked early
  -- missing agents would cause build to fail again immediately

Broken Survey (6) is lowest priority and fully independent
```

## Anti-Features

Features to explicitly NOT build.

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| Auto-continue after recovery | Colony may be in a state where continuing is wrong | Stop after fix, tell user what to run next |
| Modifying source code | Recovery should not touch user code | Only fix state files, manifests, and agent definitions |
| Deleting COLONY_STATE.json | Nuclear option that loses all colony progress | Reset specific fields, always backup first |
| Running git operations beyond stash/remove | Git merges and rebases can cause data loss | Only stash, remove worktrees, delete orphan branches |
| Silent fixes | User needs to know what changed | Print every action with file paths |
| Interactive prompts (except dirty worktree) | `aether recover --apply` should be runnable without interaction | Use `--force` for destructive operations |

## MVP Recommendation

Prioritize:
1. Missing build packet detection and fix (most common stuck state)
2. Stale spawned workers cleanup (prerequisite for clean re-dispatch)
3. Partial phase recovery (common when build finishes but continue is not run)
4. Safe reporting for all 7 classes (even if --apply is not implemented for all)

Defer:
- Dirty worktree auto-stash: needs user confirmation flow, complex interaction
- Survey regeneration: not possible from recovery alone

## Detection Summary Table

| # | Stuck State | Files Checked | State Fields Checked | Safe Auto-Fix |
|---|-------------|---------------|---------------------|---------------|
| 1 | Missing build packet | `build/phase-N/manifest.json` | `state`, `current_phase`, `build_started_at` | YES |
| 2 | Stale spawned workers | `spawn-runs.json`, `spawn-tree.txt`, `spawn-track.json` | `state`, `build_started_at` | YES |
| 3 | Partial phase | `build/phase-N/manifest.json`, `build/phase-N/continue.json` | `state`, `current_phase`, phase status | YES |
| 4 | Bad manifest | `build/phase-N/manifest.json` | (cross-reference with state) | PARTIAL |
| 5 | Dirty worktree | `.aether/worktrees/`, git worktree list | `worktrees[]` status | NO (needs confirm) |
| 6 | Broken survey | `.aether/data/survey/{5 files}` | `territory_surveyed` | PARTIAL |
| 7 | Missing agent files | `.claude/agents/ant/`, `.opencode/agents/`, `.codex/agents/` | (none -- filesystem only) | YES (if hub has files) |

## Output Format

The command should produce structured output matching the medic pattern:

```
── Colony Recovery ──

Diagnosis
  [1] Missing build packet    Phase 3 manifest not found
  [2] Stale spawned workers   4 workers timed out (>1h)
  [7] Missing agent files     2 Claude agents not installed

Apply (--apply to fix):
  [1] Reset state to READY for phase 3
  [2] Mark 4 workers as timeout
  [7] Copy 2 agents from hub

── Next Steps ──
Run `aether build 3` to re-dispatch phase 3.
```

## Sources

- `cmd/medic_cmd.go`, `cmd/medic_scanner.go`, `cmd/medic_repair.go` -- existing health check and repair infrastructure
- `cmd/codex_build.go` -- build dispatch, manifest creation, state transitions
- `cmd/state_repair.go` -- plan recovery from artifacts, phase progress inference
- `cmd/worktree.go` -- worktree allocation, orphan scanning, merge-back
- `cmd/spawn_track.go`, `pkg/agent/spawn_tree.go` -- spawn tracking and run state
- `cmd/survey.go` -- survey file validation
- `pkg/colony/colony.go` -- all state types and constants
- `cmd/recovery_snapshot.go` -- recovery guidance, next-command resolution
