# Technology Stack: aether recover

**Project:** Aether Colony Recovery Command
**Researched:** 2026-04-25
**Confidence:** HIGH (all findings verified by reading source code directly)

## Executive Summary

The `aether recover` command needs almost zero new infrastructure. The codebase already contains a complete health scanning pipeline (`cmd/medic_scanner.go`), a repair pipeline (`cmd/medic_repair.go`), state recovery from artifacts (`cmd/state_repair.go`), abandoned build detection (`cmd/codex_continue.go`), worktree orphan scanning (`cmd/worktree.go`), and checkpoint/rollback support (`cmd/autofix.go`). The gap is orchestration: a single command that chains these detectors and repairers into a coherent diagnosis-and-fix flow with clean output.

## Recommended Stack

### Core Framework (all existing, zero additions)

| Technology | Location | Purpose | Why reuse |
|------------|----------|---------|-----------|
| cobra.Command | `cmd/root.go` | CLI subcommand registration | Standard pattern for all aether commands |
| pkg/storage.Store | `pkg/storage/storage.go` | Atomic JSON read/write with file locking | Already initialized globally as `store` in every command |
| pkg/storage.FileLocker | `pkg/storage/lock.go` | Cross-process exclusive and shared locks | Needed when recover modifies state files |
| pkg/storage.CreateBackup + RotateBackups | `pkg/storage/backup.go` | Pre-repair backup with rotation | Medic repair already uses a parallel implementation; recover should use the pkg version |
| pkg/colony.ColonyState | `pkg/colony/colony.go` | Core state type | All state loading returns this type |
| pkg/codex.DispatchBatch | `pkg/codex/dispatch.go` | Worker batch execution (if recover redispatches) | Only needed if recover triggers targeted redispatch |

### Rendering (all existing)

| Technology | Location | Purpose | Why reuse |
|------------|----------|---------|-----------|
| renderBanner() | `cmd/codex_visuals.go` | ANSI banner with emoji | Consistent visual identity |
| renderStageMarker() | `cmd/codex_visuals.go` | Section separators | Clean output structure |
| renderNextUp() | `cmd/codex_visuals.go` | Next-step suggestion box | Recovery needs clear next-action guidance |
| severityColor() | `cmd/medic_cmd.go` | ANSI color for critical/warning/info | Already defined for medic, reuse for recover |
| shouldUseANSIColors() | `cmd/codex_visuals.go` | Terminal capability check | Automatic color handling |

### Existing Detection Infrastructure (reuse as-is)

| Detector | Location | What it finds | How recover uses it |
|----------|----------|---------------|---------------------|
| performHealthScan() | `cmd/medic_scanner.go` | 20+ issue types across state, session, pheromones, data files, JSONL, integrity | Call directly for comprehensive scan |
| scanColonyState() | `cmd/medic_scanner.go` | Invalid state, missing goal, EXECUTING with no phase, orphaned worktrees, deprecated signals, bad plan structure | Reuse as primary state detector |
| detectAbandonedBuild() | `cmd/codex_continue.go` | All dispatches stuck at "spawned" past 10-minute threshold | Call to detect stuck builds |
| loadCodexContinueManifest() | `cmd/codex_continue.go` | Loads build manifest for a phase | Use to check if build packet exists |
| reportOrphanBranches() | `cmd/worktree.go` | Git branches matching phase-N/name with no worktree | Call for branch cleanup detection |
| isWorktreeOrphaned() | `cmd/worktree.go` | Checks last commit time against threshold | Reuse for worktree staleness |
| getGitWorktreePaths() | `cmd/medic_repair.go` | Lists actual git worktree paths via porcelain | Reuse for state-vs-disk consistency |
| repairMissingPlanFromArtifacts() | `cmd/state_repair.go` | Recovers plan from planning/phase-plan.json when state loses it | Reuse for plan recovery |
| loadActiveRecoveryGuidance() | `cmd/recovery_snapshot.go` | Loads continue.json for current phase recovery context | Reuse to detect partial-phase state |
| loadActiveColonyState() | `cmd/state_load.go` | Loads colony state with compatibility repair, legacy normalization, and plan artifact recovery | Reuse as the primary state loader |

### Existing Repair Infrastructure (reuse as-is)

| Repair | Location | What it fixes | How recover uses it |
|--------|----------|---------------|---------------------|
| performRepairs() | `cmd/medic_repair.go` | Orchestrated repair cycle with backup, sort by severity, dedup | Core repair engine -- call directly |
| repairStateIssues() | `cmd/medic_repair.go` | Orphaned worktrees, deprecated signals, EXECUTING with no phase | Reuse for state-level fixes |
| repairPheromoneIssues() | `cmd/medic_repair.go` | Expired signals, missing IDs, invalid types | Reuse for pheromone fixes |
| repairSessionIssues() | `cmd/medic_repair.go` | Phase mismatch, goal mismatch between session and state | Reuse for session fixes |
| repairDataIssues() | `cmd/medic_repair.go` | Corrupted JSON, ghost constraints, stale cache, stale spawn state | Reuse for data-level fixes |
| autofix-checkpoint | `cmd/autofix.go` | Creates timestamped checkpoint before repair | Reuse for pre-recover checkpoint |
| autofix-rollback | `cmd/autofix.go` | Restores from checkpoint on failure | Reuse for recover rollback |
| atomicWriteFile() | `cmd/medic_repair.go` | Atomic file write via temp+rename | Already used by repair pipeline |
| syncSessionFromState() | `cmd/recovery_snapshot.go` | Syncs session.json with colony state | Reuse to align session after repair |

## What Needs To Be Built

### 1. New command file: cmd/recover.go

One new file registering the `recover` cobra command. This is orchestration-only: it calls existing scanners and repairers in a specific order and renders unified output.

**Why a new file:** The existing `medic_cmd.go` owns the `aether medic` command. `recover` is a different user intent (emergency rescue vs routine health check) with different output format (single-answer diagnosis vs detailed report). Keeping them separate avoids bloating medic and allows recover-specific flags (`--apply`, `--dry-run`).

### 2. Recover-specific detectors

These detect stuck-state scenarios that medic does not currently cover:

| Detector | What it checks | Why medic does not cover this |
|----------|---------------|-------------------------------|
| Stuck EXECUTING state | State=EXECUTING but no build manifest for current phase | Medic checks state validity but not "is there a build packet on disk" |
| Stale spawned workers | spawn-runs.json has runs with status=running/active older than 1 hour | Medic scanner checks spawn-runs JSON validity but not runtime staleness |
| Partial phase completion | Phase has mix of completed and pending tasks with no active build | Medic validates task statuses but not stuck-in-middle |
| Dirty worktree | State=EXECUTING with worktree mode but worktree has uncommitted changes | Medic checks orphaned worktrees but not dirty working trees |
| Missing agent files | Agent TOML/MD files referenced by dispatches do not exist on disk | Medic does not check agent file existence |
| Broken survey | territory_surveyed set but survey data missing | Not a medic concern |

### 3. Recover-specific output renderer

A renderer that produces the "single-answer" output the project spec calls for:
- One-line diagnosis summary
- Actionable fix list
- No wall of debug logs

This is a thin wrapper over existing `renderBanner()`, `renderStageMarker()`, `renderNextUp()`.

### 4. Recover-specific repair: state reset

The one truly new repair action: resetting colony state from EXECUTING back to READY when the build is abandoned and the user wants to restart. This is NOT the same as medic's `reset_executing_no_phase` (which only handles the edge case of EXECUTING with phase=0). The recover version needs to:
1. Reset state from EXECUTING to READY
2. Clear BuildStartedAt
3. Reset the current phase's tasks to pending
4. Log a recovery event

**Estimated size:** ~40 lines of Go.

## Alternatives Considered

| Category | Recommended | Alternative | Why Not |
|----------|-------------|-------------|---------|
| Command structure | New cmd/recover.go | Extend cmd/medic_cmd.go | Different user intent, different output format, avoid feature creep |
| Detection engine | Reuse medic scanner + add recover-specific checks | Build all detectors from scratch | Medic already has 20+ checks; recover just needs 6 more |
| Repair engine | Reuse performRepairs() + add state reset | Rewrite repair pipeline | Existing pipeline handles backup, severity sorting, dedup, tracing |
| State loading | Reuse loadActiveColonyState() | Direct file reads | Already handles compatibility repair, legacy normalization, plan recovery |
| Backup | Reuse medic's createBackup() | Use pkg/storage.CreateBackup() | Medic backup copies entire .aether/data/ which is the right scope for recovery |
| File writing | Reuse atomicWriteFile() | Use store.AtomicWrite() | Both work; recover should prefer store.AtomicWrite() since it validates JSON |

## Dependency Graph

```
cmd/recover.go
  -> cmd/medic_scanner.go  (performHealthScan)
  -> cmd/medic_repair.go   (performRepairs, createBackup)
  -> cmd/state_repair.go   (repairMissingPlanFromArtifacts)
  -> cmd/codex_continue.go (detectAbandonedBuild, loadCodexContinueManifest)
  -> cmd/worktree.go       (reportOrphanBranches, isWorktreeOrphaned)
  -> cmd/recovery_snapshot.go (loadActiveRecoveryGuidance, syncSessionFromState)
  -> cmd/codex_visuals.go  (renderBanner, renderStageMarker, renderNextUp)
  -> pkg/storage/           (Store.AtomicWrite, Store.LoadJSON, Store.SaveJSON)
  -> pkg/colony/            (ColonyState, State constants)
```

No new pkg/ dependencies. Everything stays within cmd/ and existing packages.

## Installation

No new packages required. All dependencies are already in go.mod.

## Sources

- Direct source code review of all files listed above (HIGH confidence -- read every line)
- pkg/storage/storage.go: atomic write, locking, backup APIs
- cmd/medic_scanner.go: 700+ lines of health scanning infrastructure
- cmd/medic_repair.go: 800+ lines of repair infrastructure with backup, tracing
- cmd/state_repair.go: plan recovery from artifacts, phase progress inference
- cmd/codex_continue.go: abandoned build detection, continue manifest loading
- cmd/worktree.go: worktree allocation, orphan scanning, branch detection
- cmd/recovery_snapshot.go: session sync, context/handoff documents, recovery guidance
- cmd/state_load.go: colony state loading with compatibility repair
- pkg/colony/colony.go: ColonyState type, all state/phase/task constants
- cmd/codex_visuals.go: ANSI rendering functions
