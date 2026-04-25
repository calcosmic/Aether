# Architecture Research: `aether recover` Stuck-State Colony Recovery

**Project:** Aether v1.8 Colony Recovery
**Researched:** 2026-04-25
**Confidence:** HIGH (based on direct source code analysis)

## Executive Summary

The recovery command should be a **new top-level cobra command** (`cmd/recover.go`) that **orchestrates existing subsystems** rather than duplicating their logic. The medic scanner (`cmd/medic_scanner.go`) already diagnoses 5 categories of colony health issues, but it is a general-purpose health tool. The recover command needs to specialize in the 7 classes of *stuck state* -- scenarios where the colony cannot make forward progress.

The right architecture is a **diagnosis-then-repair pipeline** built from reusable scan+fix functions extracted from existing commands. Each stuck-state class maps to an existing code path that already knows how to detect and fix that specific problem. Recover composes these rather than reinventing them.

## Recommended Architecture

### New Files

| File | Purpose |
|------|---------|
| `cmd/recover.go` | Cobra command registration, flag parsing, top-level orchestration |
| `cmd/recover_scanner.go` | 7-class stuck-state detector (new, focused on stuck states) |
| `cmd/recover_repair.go` | Repair dispatcher that composes existing fix functions |
| `cmd/recover_test.go` | Unit tests for diagnosis and repair |
| `cmd/recover_visuals.go` | Rendering for human-readable output |

### Modified Files (minor)

| File | Change |
|------|--------|
| `cmd/root.go` | Add `recoverCmd` to root |
| `cmd/medic_scanner.go` | Extract reusable scanner helpers (no behavior change) |

### No New Package Dependencies

The recover command lives entirely in `cmd/` and uses existing types from `pkg/colony`, `pkg/storage`, `pkg/agent`, and `pkg/trace`. No new packages needed.

---

## Internal Structure

### 1. Command Registration Pattern

Follow the established pattern used by `medic_cmd.go`:

```
aether recover          -- scan only, print diagnosis
aether recover --apply  -- scan + fix
aether recover --force  -- allow destructive repairs (reset state)
aether recover --json   -- structured output
```

Registration in `cmd/recover.go`:

```go
var recoverCmd = &cobra.Command{
    Use:   "recover",
    Short: "Rescue a stuck colony",
    Long:  `Scan the colony for stuck-state conditions, print a clean diagnosis, and optionally apply fixes.`,
    Args:  cobra.NoArgs,
    RunE:  runRecover,
}

func init() {
    rootCmd.AddCommand(recoverCmd)
    recoverCmd.Flags().Bool("apply", false, "apply fixes for detected issues")
    recoverCmd.Flags().Bool("force", false, "allow destructive repairs (state resets)")
    recoverCmd.Flags().Bool("json", false, "output structured JSON")
}
```

### 2. Diagnosis Pipeline

The scanner checks 7 specific stuck-state classes. Each check is a standalone function that returns a list of `StuckStateIssue` structs.

```go
type StuckStateIssue struct {
    Class       string // e.g., "missing_build_packet", "stale_spawned"
    Severity    string // "critical", "warning"
    Message     string // human-readable description
    File        string // relevant file
    Fixable     bool   // can --apply fix this?
    Destructive bool   // does fixing require --force?
    FixHint     string // what the fix does
}
```

The 7 stuck-state classes and where their detection/fix logic already lives:

| Class | Detection Source | Fix Source | New? |
|-------|-----------------|------------|------|
| 1. Missing build packet | `codex_continue.go` manifest loading | `build --force` pattern | Detection exists; fix composes existing |
| 2. Stale spawned workers | `spawn_track.go` timeout check | `spawn-track --action clear` | Detection exists; fix exists |
| 3. Partial phase (EXECUTING with no build) | `medic_scanner.go` state consistency check | `state_repair.go` EXECUTING reset | Both exist |
| 4. Bad manifest | `codex_build.go` manifest validation | Rebuild from state | New detection |
| 5. Dirty/orphaned worktrees | `worktree.go` orphan scan | `worktree-orphan-scan` + cleanup | Both exist |
| 6. Broken survey data | `survey.go` validation | Reset survey | New detection |
| 7. Missing agent files | `.claude/agents/ant/` file check | `generate_cmds.go` regeneration | New detection |

### 3. Scan Orchestrator

```go
func performStuckStateScan(dataDir string) (*StuckStateScanResult, error) {
    start := time.Now()
    var issues []StuckStateIssue

    // Load state once (shared across all checks)
    state, stateErr := loadActiveColonyState()

    issues = append(issues, scanMissingBuildPacket(state, dataDir)...)
    issues = append(issues, scanStaleSpawnedWorkers(dataDir)...)
    issues = append(issues, scanPartialPhase(state)...)
    issues = append(issues, scanBadManifest(state, dataDir)...)
    issues = append(issues, scanDirtyWorktrees(state)...)
    issues = append(issues, scanBrokenSurvey(dataDir)...)
    issues = append(issues, scanMissingAgentFiles()...)

    return &StuckStateScanResult{
        Issues:    issues,
        State:     state,
        Duration:  time.Since(start),
    }, nil
}
```

### 4. Repair Pipeline

The repair pipeline composes existing fix functions. It does **not** reimplement any repair logic.

```go
func performStuckStateRepairs(scan *StuckStateScanResult, opts RecoverOptions, dataDir string) (*RepairResult, error) {
    // 1. Create backup (reuse medic_repair.go createBackup)
    backupPath, err := createBackup(dataDir)

    // 2. Sort by severity, filter by fixable
    fixable := filterFixable(scan.Issues)

    // 3. Dispatch to existing repair functions
    for _, issue := range fixable {
        switch issue.Class {
        case "missing_build_packet":
            // Reset state to READY, clear BuildStartedAt
            // Reuse pattern from medic_repair.go repairStateIssues
        case "stale_spawned_workers":
            // Reuse spawnTrackClear pattern
        case "partial_phase":
            // Reuse medic_repair.go EXECUTING reset
        case "dirty_worktrees":
            // Reuse worktree-orphan-scan cleanup pattern
        case "bad_manifest":
            // Delete corrupt manifest, reset state
        case "broken_survey":
            // Reset survey data
        case "missing_agent_files":
            // Cannot auto-fix -- report only
        }
    }

    // 4. Re-scan to confirm fixes
    // 5. Return repair report
}
```

---

## Why Compose, Not Monolith

### Option A: Single monolithic command (REJECTED)

A monolithic `recover.go` that contains all detection and repair logic inline.

**Problems:**
- Duplicates medic scanner logic
- Duplicates spawn-track timeout logic
- Duplicates worktree orphan detection
- Maintenance burden: every fix in medic/spawn/worktree must be mirrored

### Option B: Compose existing subsystems (RECOMMENDED)

Recover orchestrates calls to existing detection and repair functions.

**Benefits:**
- Single source of truth for each repair
- Medic improvements automatically benefit recover
- Test coverage compounds (existing tests + new recover-specific tests)
- Follows the established Aether pattern: `medic --fix` already does exactly this for general health

### Why Not Subcommand Composition

One alternative would be having `recover` shell out to `aether medic --fix`, `aether worktree-orphan-scan`, etc. This is rejected because:

1. Cross-command orchestration is fragile (flag parsing, output format, exit codes)
2. Performance: loading state N times instead of once
3. Atomicity: partial failures across subcommands leave inconsistent state
4. The existing functions are in the same Go package (`cmd/`) -- direct function calls are natural

---

## Component Boundaries

```
cmd/recover.go              -- CLI entry point, flags, output routing
    |
    v
cmd/recover_scanner.go      -- 7-class stuck-state detector
    |
    |--- calls into cmd/medic_scanner.go helpers (scanColonyState, etc.)
    |--- calls into cmd/spawn_track.go (readSpawnTrack)
    |--- calls into cmd/worktree.go (worktree orphan detection)
    |
    v
cmd/recover_repair.go       -- Repair dispatcher
    |
    |--- calls into cmd/medic_repair.go (createBackup, atomicWriteFile, repairStateIssues)
    |--- calls into cmd/spawn_track.go (clear pattern)
    |--- calls into cmd/worktree.go (cleanup pattern)
    |--- calls into cmd/state_repair.go (repairMissingPlanFromArtifacts)
    |
    v
cmd/recover_visuals.go      -- Output rendering (visual + JSON)
```

### Data Flow

```
User runs: aether recover [--apply] [--force]

1. loadActiveColonyState()
   - loads COLONY_STATE.json
   - runs normalizeLegacyColonyState()
   - runs repairMissingPlanFromArtifacts()

2. performStuckStateScan(state, dataDir)
   - runs 7 detection checks
   - returns StuckStateIssue list

3. renderDiagnosis(issues)      [scan-only mode: stop here]
   - prints clean summary table
   - prints each issue with severity and fix hint
   - suggests: "run aether recover --apply"

4. createBackup(dataDir)        [apply mode only]
   - copies .aether/data/ to .aether/backups/recover-TIMESTAMP/

5. performStuckStateRepairs(scan, opts, dataDir)
   - dispatches each fixable issue to its repair function
   - logs each repair attempt via trace

6. re-scan to confirm
   - runs detection again
   - reports what's still stuck

7. renderRepairReport(original, postRepair, repairResult)
   - shows fixed/unfixed summary
   - suggests next command
```

---

## Integration Points with Existing Commands

### medic (cmd/medic_cmd.go)

**Relationship:** Recover is a specialized stuck-state tool. Medic is a general health scanner.

- Recover reuses `HealthIssue` struct for compatibility
- Recover reuses `createBackup()` and `atomicWriteFile()` from `medic_repair.go`
- Recover does NOT replace medic. Medic scans for data corruption, schema issues, and deep integrity. Recover scans for stuck-state specifically.
- A colony can be "healthy" by medic standards but "stuck" by recover standards (e.g., EXECUTING with a stale build -- medic sees valid state, recover sees a stuck colony)

**Implementation note:** Extract the following from `medic_scanner.go` into shared helpers:
- `issueCritical()`, `issueWarning()`, `issueInfo()`, `fixableIssue()` -- already used by both
- `checkJSONFile()` -- useful for recover's manifest and survey checks
- `parseTimestamp()` -- useful for staleness checks

These are already unexported functions in the same package, so no extraction needed -- just call them directly.

### build --force (cmd/codex_build.go)

**Relationship:** `build --force` handles one specific recovery path (redispatch an active phase).

- Recover detects the "missing build packet" condition and reports it
- When `--apply` is used, recover can reset the colony to READY state so the user can run `aether build N` next
- Recover does NOT directly invoke `build --force` internally (cross-command invocation is the anti-pattern)
- Instead, recover resets the state and tells the user to run `build N` or `build N --force`

### skip-phase --force (cmd/phase_skip.go)

**Relationship:** Skip-phase is the emergency escape hatch for an unrecoverable phase.

- Recover detects when a phase is truly stuck (no workers, stale state, no manifest)
- Recover suggests `aether skip-phase N --force` as the fix
- Recover does NOT auto-skip phases (too destructive even with `--force`)
- The repair for "partial phase" resets to READY and suggests the next command

### plan --force (cmd/codex_plan.go)

**Relationship:** `plan --force` (the `--refresh` flag) resets stale plan state.

- Recover can detect when the plan is in an inconsistent state (phases with wrong statuses)
- Recover does NOT call plan logic -- it reports and suggests `aether plan --refresh`
- For plan recovery from artifacts, `state_repair.go:repairMissingPlanFromArtifacts()` is already called during `loadActiveColonyState()` -- recover gets this for free

### worktree-orphan-scan (cmd/worktree.go)

**Relationship:** Worktree orphan detection is fully implemented.

- Recover calls the same detection logic inline (the orphan detection is straightforward)
- For repair, recover marks orphaned worktrees for cleanup and runs `git worktree remove` + `git worktree prune`
- Reuses `getGitWorktreePaths()` and `isWorktreeOrphaned()` from the existing worktree code

### spawn-track (cmd/spawn_track.go)

**Relationship:** Spawn tracking already has timeout enforcement.

- Recover reads `spawn-track.json` to find stale agents
- Recover reads `spawn-runs.json` to find stuck runs
- For repair, recover clears stale spawn state (reuse `repairDataIssues` "stale spawn" path from `medic_repair.go`)

---

## Output Format

### Visual (default)

```
Colony Recovery: "Build feature X"
Phase 3/6 -- EXECUTING

Scanning for stuck-state conditions...

  CRITICAL [missing_build_packet] No build manifest for phase 3
    Fix: aether build 3 --force

  WARNING [stale_spawned] 2 workers spawned >1h ago with no completion
    Fixable with --apply

  INFO [orphaned_worktrees] 1 orphaned worktree detected
    Fixable with --apply

3 issues found (1 critical, 1 warning, 1 info)

Run `aether recover --apply` to fix 2 issues automatically.
```

### JSON (--json flag)

```json
{
  "timestamp": "2026-04-25T...",
  "goal": "Build feature X",
  "phase": 3,
  "total_phases": 6,
  "state": "EXECUTING",
  "issues": [
    {
      "class": "missing_build_packet",
      "severity": "critical",
      "message": "No build manifest for phase 3",
      "fixable": false,
      "destructive": false,
      "fix_hint": "aether build 3 --force"
    }
  ],
  "summary": {
    "critical": 1,
    "warning": 1,
    "info": 1,
    "fixable": 2
  },
  "repairs": null
}
```

---

## Anti-Patterns to Avoid

### 1. Do Not Shell Out to Other Commands

The existing pattern in Aether is direct function calls within the `cmd/` package. Never use `exec.Command("aether", "medic", ...)` from within recover.

### 2. Do Not Duplicate medic's General Health Scanning

Recover is specialized for stuck-state. It does not re-check JSON validity, schema conformance, or data file corruption. That is medic's job. Users should run both if they want full diagnostics.

### 3. Do Not Auto-Skip Phases

Even with `--force`, recover should not skip phases. Skip-phase is a separate decision. Recover resets the state to allow the user to make that choice.

### 4. Do Not Modify ColonyState Without Backup

Follow medic's pattern: always create a timestamped backup before any state mutation.

### 5. Do Not Invent New State Transitions

Recover should only use existing valid state transitions from `pkg/colony/state_machine.go`. EXECUTING -> READY is valid. BUILT -> READY is valid. Any new transition needs to go through the state machine first.

---

## Scalability Considerations

| Concern | Current (1 colony) | Future (10 colonies) | Notes |
|---------|-------------------|---------------------|-------|
| Scan time | <100ms | ~1s | File I/O is the bottleneck, not CPU |
| State size | ~50KB COLONY_STATE.json | N/A (per-repo) | Recovery is per-colony |
| Backup space | ~500KB per backup | ~5MB with retention | Keep 3 most recent backups |
| Agent file check | O(N) file stat calls | N/A | Only checks current repo |

---

## Build Order (Dependencies)

The phases should be ordered by dependency:

1. **Phase A: Scanner foundation** (`cmd/recover.go` + `cmd/recover_scanner.go`)
   - Implement diagnosis-only mode first
   - Reuses existing detection functions
   - No mutations, safe to test
   - Validates all 7 detection classes work

2. **Phase B: Repair pipeline** (`cmd/recover_repair.go`)
   - Add `--apply` and `--force` flags
   - Implement repair dispatch for fixable issues
   - Reuses existing repair functions
   - Backup before any mutation

3. **Phase C: Visual output** (`cmd/recover_visuals.go`)
   - Human-readable rendering
   - JSON output
   - Follows existing `codex_visuals.go` patterns

4. **Phase D: Integration tests** (`cmd/recover_test.go`)
   - Test each stuck-state class in isolation
   - Test repair pipeline end-to-end
   - Test backup/restore
   - Test that fixable issues are fixed and unfixable issues are reported

**Rationale:** Scanner first because it has zero risk (no mutations). Repair second because it depends on scanner output. Visuals third because it is presentation-only. Tests throughout, but the dedicated integration test phase validates the full pipeline.

---

## Sources

- Direct source code analysis of `cmd/` and `pkg/colony/` (2900+ tests, v1.0.20+)
- `cmd/medic_cmd.go`, `cmd/medic_scanner.go`, `cmd/medic_repair.go` -- existing health scan and repair patterns
- `cmd/state_repair.go` -- plan recovery from artifacts
- `cmd/phase_skip.go` -- force-skip pattern
- `cmd/worktree.go` -- worktree lifecycle management
- `cmd/spawn_track.go` -- spawn timeout tracking
- `cmd/lifecycle_helpers.go` -- recovery guidance and next-command logic
- `cmd/codex_build.go` -- build manifest structure
- `cmd/codex_continue.go` -- continue report and recovery plan structures
- `pkg/colony/colony.go` -- ColonyState and related types
- `pkg/colony/state_machine.go` -- legal state transitions
