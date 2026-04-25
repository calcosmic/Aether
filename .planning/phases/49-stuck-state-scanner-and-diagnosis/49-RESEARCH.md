# Phase 49: Stuck-State Scanner and Diagnosis - Research

**Researched:** 2026-04-25
**Domain:** Colony lifecycle recovery detection for Aether Go CLI
**Confidence:** HIGH (all findings verified against source code)

## Summary

Phase 49 builds the scanner/diagnosis half of `aether recover`. The detection infrastructure is almost entirely pre-built: the medic scanner (`cmd/medic_scanner.go`) already checks 5 categories of colony health, `codex_continue.go` has abandoned-build detection, `worktree.go` has orphan scanning, `spawn_track.go` has timeout tracking, and `survey.go` validates survey files. The recover scanner composes these existing detectors and adds 7 stuck-state-specific checks that medic does not cover.

The output side is equally well-supported: `HealthIssue` is the standard issue type with severity, category, message, file, and fixable flag. The medic report renderer already handles banner, stage markers, severity coloring, and JSON output. Recover reuses this output infrastructure with a slightly different layout (single-answer diagnosis instead of detailed report).

**Primary recommendation:** Create a new `cmd/recover.go` that registers the cobra command and calls a new `cmd/recover_scanner.go` containing 7 detection functions. Each function calls into existing infrastructure rather than reimplementing detection logic. Output reuses `renderBanner`, `renderStageMarker`, `renderNextUp`, and `severityColor` from the existing visuals pipeline.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| DETECT-01 | Detect missing build packet | `loadCodexContinueManifest()` returns `Present: false` when manifest missing; state check `State == EXECUTING \|\| State == BUILT` + `CurrentPhase > 0` |
| DETECT-02 | Detect stale spawned workers | `spawn-runs.json` has `runs[].status == "active"` + `started_at` > 1 hour; `spawn-tree.txt` entries with live spawn status; medic already has `reset_stale_spawn_state` repair |
| DETECT-03 | Detect partial phase | Phase `status == "in_progress"` + manifest all-terminal + no `continue.json`; OR `status == "in_progress"` + no manifest at all |
| DETECT-04 | Detect bad manifest | JSON parse failure, `phase` field mismatch, empty `generated_at`, dispatches referencing non-existent task IDs |
| DETECT-05 | Detect dirty worktree | `worktrees[].status` not merged/orphaned + `git status --porcelain` non-empty in worktree path; `reportOrphanBranches()` detects orphaned branches |
| DETECT-06 | Detect broken survey | `survey/` dir exists but 5 expected files (`blueprint`, `chambers`, `disciplines`, `provisions`, `pathogens`) missing, invalid JSON, or empty |
| DETECT-07 | Detect missing agent files | 25 expected files in each of `.claude/agents/ant/`, `.opencode/agents/`, `.codex/agents/`; `scanWrapperParity()` already counts these |
| OUTP-01 | Clean diagnosis output | Reuse `HealthIssue` type, `renderBanner`, `renderStageMarker`, `renderNextUp`, `severityColor` |
| OUTP-02 | Exit code 0/1 | Reuse `medicExitCode()` pattern: 0 = no issues, 1 = issues found |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Colony state loading | CLI / Go runtime | -- | `loadActiveColonyState()` in `cmd/state_load.go` |
| Build manifest validation | CLI / Go runtime | -- | `loadCodexContinueManifest()` in `cmd/codex_continue.go` |
| Spawn state staleness | CLI / Go runtime | -- | `spawn-runs.json` + `spawn-tree.txt` in `.aether/data/` |
| Worktree status | CLI / Go runtime | Git CLI | `worktree.go` uses `git worktree list --porcelain` |
| Survey file validation | CLI / Go runtime | -- | `survey.go` reads `.aether/data/survey/` |
| Agent file existence | CLI / Go runtime | Filesystem | `scanWrapperParity()` in `medic_wrapper.go` |
| Diagnosis output rendering | CLI / Go runtime | -- | `codex_visuals.go` rendering functions |

## Standard Stack

### Core (all existing, zero new dependencies)

| Library | Location | Purpose | Why reuse |
|---------|----------|---------|-----------|
| cobra.Command | `cmd/root.go` | CLI subcommand registration | Standard pattern for all aether commands |
| pkg/storage.Store | `pkg/storage/storage.go` | JSON read/write with file locking | Already initialized globally as `store` |
| pkg/colony.ColonyState | `pkg/colony/colony.go` | Core state type | All state loading returns this type |
| HealthIssue | `cmd/medic_cmd.go` | Issue representation (severity, category, message, file, fixable) | Standard issue type used by medic scanner and renderer |

### Detection Infrastructure (reuse directly)

| Function | Location | Signature | What it detects |
|----------|----------|-----------|-----------------|
| `loadActiveColonyState()` | `cmd/state_load.go:17` | `func loadActiveColonyState() (colony.ColonyState, error)` | Loads COLONY_STATE.json with compatibility repair, legacy normalization, plan artifact recovery |
| `loadCodexContinueManifest(phaseID int)` | `cmd/codex_continue.go:678` | `func loadCodexContinueManifest(phaseID int) codexContinueManifest` | Loads build manifest, returns `Present: false` if missing |
| `detectAbandonedBuild(manifest, state)` | `cmd/codex_continue.go:107` | `func detectAbandonedBuild(manifest codexContinueManifest, state colony.ColonyState) (bool, time.Duration, string)` | All dispatches stuck at "spawned" past 10-minute threshold |
| `isWorktreeOrphaned(commitAt, threshold)` | `cmd/worktree.go:94` | `func isWorktreeOrphaned(commitAt time.Time, threshold time.Duration) bool` | Last commit time exceeds threshold |
| `reportOrphanBranches()` | `cmd/worktree.go:487` | `func reportOrphanBranches() ([]map[string]interface{}, error)` | Agent-track branches with no worktree |
| `getGitWorktreePaths()` | `cmd/medic_repair.go:792` | `func getGitWorktreePaths() map[string]bool` | Git worktree paths via porcelain |
| `parseTimestamp(s)` | `cmd/medic_scanner.go:414` | `func parseTimestamp(s string) time.Time` | Multi-format timestamp parsing |
| `checkJSONFile(filename, description)` | `cmd/medic_scanner.go:42` | `func (fc *fileChecker) checkJSONFile(filename, description string) ([]byte, bool)` | JSON file validation with issue recording |
| `scanColonyState(fc)` | `cmd/medic_scanner.go:211` | `func scanColonyState(fc *fileChecker) []HealthIssue` | Full COLONY_STATE.json validation |
| `scanWrapperParity(fc)` | `cmd/medic_wrapper.go:33` | `func scanWrapperParity(fc *fileChecker) []HealthIssue` | Agent/command file count checks per platform |

### Rendering (reuse directly)

| Function | Location | Purpose |
|----------|----------|---------|
| `renderBanner(emoji, title)` | `cmd/codex_visuals.go` | ANSI banner with emoji |
| `renderStageMarker(name)` | `cmd/codex_visuals.go` | Section separator `-- Name --` |
| `renderNextUp(lines...)` | `cmd/codex_visuals.go` | Next-step suggestion box |
| `severityColor(sev)` | `cmd/medic_cmd.go:342` | ANSI color for critical/warning/info |
| `shouldUseANSIColors()` | `cmd/codex_visuals.go` | Terminal capability check |
| `writeIssueLine(b, issue)` | `cmd/medic_cmd.go:289` | Formatted issue line with color and fixable tag |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| New recover-specific output | Extend medic report | Recover has different output format (single-answer vs detailed); keeping separate avoids bloating medic |
| Direct function calls | Shell out to `aether medic`, `aether worktree-orphan-scan` | Direct calls are faster, avoid state reloading, allow atomic operations, and prevent cross-command fragility |
| New `StuckStateIssue` type | Reuse `HealthIssue` | HealthIssue already has all needed fields; recover can add a `Class` field or encode in `Category` |

## Architecture Patterns

### Recommended Project Structure
```
cmd/
  recover.go              -- Cobra command registration, flag parsing, orchestration
  recover_scanner.go      -- 7 stuck-state detection functions
  recover_visuals.go      -- Diagnosis rendering (visual + JSON)
  recover_test.go         -- Unit tests for detection and output
```

### Pattern 1: Detector Function Signature
**What:** Each detector is a standalone function that takes state + data dir and returns `[]HealthIssue`.
**When to use:** For each of the 7 stuck-state classes.
**Example:**
```go
// Source: [VERIFIED: cmd/medic_scanner.go pattern]
func scanMissingBuildPacket(state colony.ColonyState, dataDir string) []HealthIssue {
    var issues []HealthIssue
    if state.State != colony.StateEXECUTING && state.State != colony.StateBUILT {
        return issues
    }
    if state.CurrentPhase < 1 {
        return issues
    }
    manifest := loadCodexContinueManifest(state.CurrentPhase)
    if !manifest.Present {
        issues = append(issues, HealthIssue{
            Severity: "critical",
            Category: "missing_build_packet",
            Message:  fmt.Sprintf("No build manifest for phase %d", state.CurrentPhase),
            File:     fmt.Sprintf("build/phase-%d/manifest.json", state.CurrentPhase),
            Fixable:  true,
        })
    }
    return issues
}
```

### Pattern 2: Orchestrator
**What:** Single function runs all 7 detectors in order and collects results.
**When to use:** Top-level scan function called by `runRecover`.
**Example:**
```go
// Source: [VERIFIED: cmd/medic_scanner.go:151 performHealthScan pattern]
func performStuckStateScan(dataDir string) ([]HealthIssue, error) {
    state, stateErr := loadActiveColonyState()
    // If no colony state, report single issue and return
    if stateErr != nil {
        return []HealthIssue{{
            Severity: "critical",
            Category: "state",
            Message:  colonyStateLoadMessage(stateErr),
            Fixable:  false,
        }}, nil
    }

    var issues []HealthIssue
    // Detection order matters (see Detection Order section)
    issues = append(issues, scanStaleSpawnedWorkers(dataDir)...)
    issues = append(issues, scanMissingBuildPacket(state, dataDir)...)
    issues = append(issues, scanBadManifest(state, dataDir)...)
    issues = append(issues, scanPartialPhase(state, dataDir)...)
    issues = append(issues, scanDirtyWorktrees(state)...)
    issues = append(issues, scanBrokenSurvey(state, dataDir)...)
    issues = append(issues, scanMissingAgentFiles()...)

    return issues, nil
}
```

### Anti-Patterns to Avoid
- **Do not shell out to other commands:** Use direct function calls within the `cmd` package. Cross-command orchestration is fragile, slow (reloads state N times), and prevents atomic operations.
- **Do not duplicate medic's general health scanning:** Recover is specialized for stuck-state. It does not re-check JSON validity, schema conformance, or data file corruption. Users run both if they want full diagnostics.
- **Do not auto-skip phases:** Even with `--force`, recover should not skip phases. It resets state so the user can decide.
- **Do not modify ColonyState without loading via `loadActiveColonyState()`:** Always use the primary state loader to get compatibility repair and plan artifact recovery for free.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| JSON file validation | Custom JSON parsing + error handling | `fileChecker.checkJSONFile()` from `medic_scanner.go` | Handles missing, corrupted, and valid cases with issue recording |
| Timestamp parsing | Custom date parsing | `parseTimestamp()` from `medic_scanner.go` | Handles 6 common formats |
| Stale spawn run detection | Custom spawn-runs reader | Inline struct from `repairDataIssues` in `medic_repair.go:627-633` | Already handles the `current_run_id` + `runs[]` schema |
| Worktree orphan detection | Custom git worktree listing | `getGitWorktreePaths()` + `isWorktreeOrphaned()` | Handles porcelain parsing and threshold checking |
| Orphan branch detection | Custom branch listing + filtering | `reportOrphanBranches()` | Already filters by `agentBranchRe`, checks state tracking, computes age |
| Manifest loading | Direct file read + parse | `loadCodexContinueManifest(phaseID)` | Returns structured `codexContinueManifest` with `Present` flag |
| Abandoned build detection | Custom spawn status checker | `detectAbandonedBuild(manifest, state)` | Already checks all dispatches for "spawned" status + time threshold |
| Survey file validation | Custom survey reader | `surveyFiles` list + pattern from `surveyVerifyCmd` | Already defines 5 expected files with JSON validation |
| Agent file counting | Custom directory listing | `scanWrapperParity()` + `countFilesInDir()` from `medic_wrapper.go` | Already defines expected counts per platform (25 each) |
| Exit code calculation | Custom severity scoring | `medicExitCode()` pattern from `medic_cmd.go:355` | Returns 0 for no issues, 1 for any issues |

**Key insight:** The entire detection pipeline composes existing functions. The only new code is orchestration (calling detectors in the right order) and recover-specific output rendering.

## Common Pitfalls

### Pitfall 1: Detection Order Dependencies
**What goes wrong:** Checking "partial phase" before "bad manifest" gives wrong results because a corrupted manifest makes the partial-phase check unreliable.
**Why it happens:** Manifest checks depend on the manifest being parseable.
**How to avoid:** Run detectors in dependency order: stale workers first (independent), then bad manifest (validates manifest), then missing build packet (depends on manifest state), then partial phase (depends on manifest being valid), then the independent checks (worktree, survey, agents).
**Warning signs:** Partial phase detected when manifest is actually corrupted.

### Pitfall 2: False Positive During Active Build
**What goes wrong:** `aether recover` reports "stale spawned workers" while a build is actively running and workers are legitimately active.
**Why it happens:** The 1-hour threshold is based on training data, but some builds take longer.
**How to avoid:** Check `state.BuildStartedAt` against the abandoned threshold. If the build started less than 10 minutes ago (the `abandonedBuildThreshold` constant), skip stale-worker detection. Additionally, if `state.State == EXECUTING` and `build_started_at` is less than 1 hour old, workers may still be legitimately running.
**Warning signs:** Recover reports issues on a colony that is actively building.

### Pitfall 3: Missing Build Packet vs Partial Phase Ambiguity
**What goes wrong:** Both conditions start with "EXECUTING state, phase in_progress" but the fix is different. Missing packet needs state reset; partial phase needs state transition to BUILT.
**Why it happens:** The two states are distinguished only by whether a manifest exists.
**How to avoid:** Always check manifest existence first. If manifest is missing or has `plan_only: true`, it is "missing build packet." If manifest exists with real dispatches, it is "partial phase." These two classes are mutually exclusive.
**Warning signs:** Recover suggests "reset to READY" when the build actually completed and just needs `aether continue`.

### Pitfall 4: Worktree State vs Disk Disagreement
**What goes wrong:** State says worktree is "allocated" but the git worktree was manually removed, or vice versa.
**Why it happens:** External cleanup (user ran `git worktree prune`) without updating COLONY_STATE.json.
**How to avoid:** Always cross-reference state entries with actual `git worktree list --porcelain` output. Use `getGitWorktreePaths()` for disk truth, state entries for tracked truth.
**Warning signs:** State references a worktree path that does not exist on disk.

### Pitfall 5: Agent File Count Expectations
**What goes wrong:** Hardcoding "25 agent files" breaks when a new agent is added.
**Why it happens:** The `expectedClaudeAgents = 25` constant in `medic_wrapper.go` must be updated when agents are added.
**How to avoid:** For recover, check that each expected agent basename exists rather than counting. Or read the expected count from the same constants in `medic_wrapper.go` (they are package-level vars, accessible from recover code).
**Warning signs:** Recover reports missing agents after a new agent is added but before `expectedClaudeAgents` is updated.

### Pitfall 6: HealthIssue Category Collision
**What goes wrong:** Using generic category names like "state" or "data" could collide with medic's categories, making it hard to distinguish recover issues from medic issues.
**Why it happens:** Both commands use the same `HealthIssue` type.
**How to avoid:** Use recover-specific category names: `"missing_build_packet"`, `"stale_spawned"`, `"partial_phase"`, `"bad_manifest"`, `"dirty_worktree"`, `"broken_survey"`, `"missing_agents"`. These are distinct from medic's categories (`"state"`, `"session"`, `"pheromone"`, `"data"`, `"file"`, `"jsonl"`, `"wrapper"`, `"integrity"`).
**Warning signs:** Downstream code cannot tell if an issue came from medic or recover.

## Code Examples

### Load Colony State (Primary Entry Point)
```go
// Source: [VERIFIED: cmd/state_load.go:17]
func loadActiveColonyState() (colony.ColonyState, error)
// Returns colony state with:
//   - Compatibility repair (legacy numeric string fields)
//   - Legacy normalization (PAUSED->READY, PLANNED->READY, etc.)
//   - Plan artifact recovery (repairMissingPlanFromArtifacts)
//   - Returns errNoColonyInitialized if no goal set
```

### Load Build Manifest
```go
// Source: [VERIFIED: cmd/codex_continue.go:678]
func loadCodexContinueManifest(phaseID int) codexContinueManifest
// Returns struct with:
//   Present bool              -- false if manifest missing or load failed
//   Path    string            -- relative path "build/phase-N/manifest.json"
//   Data    codexBuildManifest -- parsed manifest with Phase, Dispatches, etc.
```

### Detect Abandoned Build
```go
// Source: [VERIFIED: cmd/codex_continue.go:107]
func detectAbandonedBuild(manifest codexContinueManifest, state colony.ColonyState) (bool, time.Duration, string)
// Returns true when:
//   - manifest has dispatches
//   - ALL dispatches have status "spawned"
//   - state.BuildStartedAt is not nil
//   - elapsed > abandonedBuildThreshold (10 minutes)
```

### Check Stale Spawn Runs (inline struct from medic_repair.go)
```go
// Source: [VERIFIED: cmd/medic_repair.go:627-633]
var spawnState struct {
    CurrentRunID string `json:"current_run_id"`
    Runs         []struct {
        ID        string `json:"id"`
        StartedAt string `json:"started_at"`
        Status    string `json:"status"`
    } `json:"runs"`
}
// Check: run.Status == "running" || run.Status == "active"
// And:   time.Since(parseTimestamp(run.StartedAt)) > time.Hour
```

### Survey Files to Check
```go
// Source: [VERIFIED: cmd/survey.go:12-18]
var surveyFiles = []string{
    "blueprint", "chambers", "disciplines", "provisions", "pathogens",
}
// Each expected at: .aether/data/survey/{name}.json
```

### Agent File Surfaces
```go
// Source: [VERIFIED: cmd/medic_wrapper.go:14-16]
expectedClaudeAgents   = 25
expectedOpenCodeAgents = 25
expectedCodexAgents    = 25
// Check paths:
//   .claude/agents/ant/aether-{name}.md
//   .opencode/agents/aether-{name}.md
//   .codex/agents/aether-{name}.toml
```

### Legal State Transitions
```go
// Source: [VERIFIED: pkg/colony/colony.go:490-496]
var legalTransitions = map[State][]State{
    StateIDLE:      {StateREADY},
    StateREADY:     {StateEXECUTING, StateCOMPLETED},
    StateEXECUTING: {StateBUILT, StateCOMPLETED},    // EXECUTING -> BUILT is legal
    StateBUILT:     {StateREADY, StateCOMPLETED},     // BUILT -> READY is legal
    StateCOMPLETED: {StateIDLE},
}
// Key transitions for recover:
//   EXECUTING -> BUILT (partial phase: build done, continue not run)
//   EXECUTING -> READY (missing packet: reset for re-build)
//   BUILT     -> READY (stale BUILT state: reset for re-build)
```

## Detection Order (Dependencies)

The 7 detectors must run in a specific order because some checks depend on the results of earlier checks:

```
1. scanStaleSpawnedWorkers(dataDir)    -- INDEPENDENT, no prerequisites
                                        Reads spawn-runs.json + spawn-tree.txt
                                        Must run BEFORE missing_build_packet because
                                        stale workers confuse re-dispatch

2. scanBadManifest(state, dataDir)     -- Needs state loaded
                                        Reads manifest at build/phase-N/manifest.json
                                        Must run BEFORE partial_phase because
                                        a corrupted manifest makes partial-phase
                                        detection unreliable

3. scanMissingBuildPacket(state, dataDir) -- Needs state + manifest status
                                           EXCLUSIVE with partial_phase:
                                           if manifest missing -> missing_packet
                                           if manifest exists -> NOT missing_packet

4. scanPartialPhase(state, dataDir)    -- Needs state + valid manifest
                                        EXCLUSIVE with missing_build_packet
                                        Checks manifest dispatches + continue.json

5. scanDirtyWorktrees(state)           -- INDEPENDENT, needs state.Worktrees
                                        Cross-references with git worktree list

6. scanBrokenSurvey(state, dataDir)    -- INDEPENDENT
                                        Checks state.TerritorySurveyed + survey dir

7. scanMissingAgentFiles()             -- INDEPENDENT, needs no state
                                        Checks 3 directories for 25 files each
```

## File Layout Under .aether/data/

Each detector needs to check specific files. Here is the exact layout:

### DETECT-01: Missing Build Packet
| File | Field | Check |
|------|-------|-------|
| `COLONY_STATE.json` | `state` | `"EXECUTING"` or `"BUILT"` |
| `COLONY_STATE.json` | `current_phase` | `> 0` |
| `COLONY_STATE.json` | `build_started_at` | non-nil |
| `build/phase-{N}/manifest.json` | (existence) | file does not exist, or `plan_only: true`, or `dispatches: []` |

### DETECT-02: Stale Spawned Workers
| File | Field | Check |
|------|-------|-------|
| `spawn-runs.json` | `current_run_id` | points to a run with `status: "active"` |
| `spawn-runs.json` | `runs[].started_at` | older than 1 hour |
| `spawn-runs.json` | `runs[].status` | `"active"` or `"running"` |
| `spawn-tree.txt` | (entries) | entries with live status in current run window |
| `spawn-track.json` | (existence) | references agent no longer running |
| `COLONY_STATE.json` | `state` | `"EXECUTING"` or `"BUILT"` |

### DETECT-03: Partial Phase
| File | Field | Check |
|------|-------|-------|
| `COLONY_STATE.json` | `state` | `"EXECUTING"` |
| `COLONY_STATE.json` | `current_phase` | `> 0` |
| `COLONY_STATE.json` | `plan.phases[{current}].status` | `"in_progress"` |
| `build/phase-{N}/manifest.json` | `dispatches[].status` | all are `"completed"` or `"failed"` (nothing active) |
| `build/phase-{N}/continue.json` | (existence) | does not exist |
| `COLONY_STATE.json` | `build_started_at` | nil means phase was marked but never built |

### DETECT-04: Bad Manifest
| File | Field | Check |
|------|-------|-------|
| `build/phase-{N}/manifest.json` | (JSON validity) | fails parsing |
| `build/phase-{N}/manifest.json` | `phase` | mismatches `current_phase` |
| `build/phase-{N}/manifest.json` | `generated_at` | empty string |
| `build/phase-{N}/manifest.json` | `state` | empty string |
| `build/phase-{N}/manifest.json` | `dispatches[].task_id` | references non-existent task |

### DETECT-05: Dirty Worktree
| File | Field | Check |
|------|-------|-------|
| `COLONY_STATE.json` | `worktrees[].status` | not `"merged"`, not `"orphaned"` |
| `COLONY_STATE.json` | `worktrees[].path` | path exists on disk |
| `.aether/worktrees/{branch}` | `git status --porcelain` | non-empty output = dirty |
| (git worktree list) | (porcelain output) | state entry has no backing git worktree |
| (git branch list) | `phase-N/name` pattern | branches with no worktree and no state entry |

### DETECT-06: Broken Survey
| File | Field | Check |
|------|-------|-------|
| `COLONY_STATE.json` | `territory_surveyed` | non-nil (survey was run) |
| `survey/blueprint.json` | (existence + JSON validity) | missing, invalid, or empty |
| `survey/chambers.json` | (existence + JSON validity) | missing, invalid, or empty |
| `survey/disciplines.json` | (existence + JSON validity) | missing, invalid, or empty |
| `survey/provisions.json` | (existence + JSON validity) | missing, invalid, or empty |
| `survey/pathogens.json` | (existence + JSON validity) | missing, invalid, or empty |

### DETECT-07: Missing Agent Files
| Directory | Expected files | Check |
|-----------|---------------|-------|
| `.claude/agents/ant/` | `aether-{name}.md` x 25 | file existence |
| `.opencode/agents/` | `aether-{name}.md` x 25 | file existence |
| `.codex/agents/` | `aether-{name}.toml` x 25 | file existence |
| `~/.aether/system/claude/agents/` | hub source files | cross-check if hub has files local is missing |

## Existing HealthIssue Type (Reuse for Recover Output)

```go
// Source: [VERIFIED: cmd/medic_cmd.go:24-30]
type HealthIssue struct {
    Severity string `json:"severity"`  // "critical", "warning", "info"
    Category string `json:"category"`  // Detector-specific: "missing_build_packet", etc.
    Message  string `json:"message"`   // Human-readable description
    File     string `json:"file,omitempty"` // Relevant file path
    Fixable  bool   `json:"fixable"`   // Can --apply fix this?
}
```

Recover can reuse this type directly. The `Category` field encodes the stuck-state class. For recover-specific metadata (like `Destructive` and `FixHint` from ARCHITECTURE.md), these can be added to the render layer without changing the type, or recover can define a wrapper type:

```go
type RecoverIssue struct {
    HealthIssue
    Class       string // e.g., "missing_build_packet"
    Destructive bool   // requires --force
    FixHint     string // what the fix does
}
```

But the simpler approach is to use `HealthIssue` directly and encode extra info in the `Message` field or render logic.

## Edge Cases: When "Stuck" Is Actually Normal

| Scenario | Why It Looks Stuck | Why It Is Normal | How to Handle |
|----------|--------------------|------------------|---------------|
| Active build in progress | State is EXECUTING, spawned workers show "active" | Build is still running | Check `build_started_at` against `abandonedBuildThreshold` (10 min). If less than threshold, skip stale-worker detection. |
| Post-build, pre-continue | State is EXECUTING, phase is "in_progress", no continue.json | Build finished but user has not run continue yet | This IS partial phase, but it is not "stuck" -- it is an expected intermediate state. Report it as "info" severity, not "critical." |
| Colony paused | State is READY with `paused: true` | User intentionally paused | Do not flag as stuck. Check `state.Paused` before reporting issues. |
| Colony just initialized | State is IDLE or READY with no phases | Colony was just created, nothing built yet | Skip all build-related checks when `current_phase == 0` and `len(plan.phases) == 0`. |
| Survey not yet run | `territory_surveyed` is nil | User has not run colonize yet | Skip survey check when `territory_surveyed` is nil. Only check survey files if survey was previously run. |
| Worktree colony with no active worktrees | `worktrees` array is empty | Between builds, all worktrees cleaned up | Only check worktree status if there are non-merged, non-orphaned entries in the array. |

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Shell-based medic checks | Go runtime medic scanner | v1.4 (Phase 25) | Go scanner is faster, more reliable, testable |
| No spawn run tracking | spawn-runs.json with BeginRun/EndRun | v1.2 (Phase 13) | Can now detect stale runs precisely |
| No worktree lifecycle tracking | COLONY_STATE.json worktrees array | v1.3 (Phase 17) | Can detect orphaned/dirty worktrees |
| Manual state recovery | `build --force` + `plan --force` | v1.7 (Phase 47-48) | Specific recovery paths but no unified scanner |

**Deprecated/outdated:**
- `constraints.json`: Ghost file that Go code ignores. Medic already flags it.
- `state.Signals`: Deprecated in favor of `pheromones.json`. Medic already flags migration.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | 1-hour threshold for stale spawned workers is appropriate | Detection Order | If builds regularly take >1h, false positives on active colonies |
| A2 | `loadCodexContinueManifest` returns `Present: false` for both missing and `plan_only: true` manifests | DETECT-01 | Need to verify `plan_only` is checked separately |
| A3 | 25 agents per platform is current count and will stay stable | DETECT-07 | Will break when new agent added; use `expectedClaudeAgents` constant |
| A4 | `HealthIssue` type is sufficient for recover output without additional fields | Output | If downstream (Phase 50 repair) needs more metadata, type may need extension |

## Open Questions

1. **Should recover use `HealthIssue` directly or a wrapper type?**
   - What we know: `HealthIssue` has all fields needed for detection and output (OUTP-01, OUTP-02).
   - What's unclear: Phase 50 (repair) may need additional metadata (destructive flag, fix hint).
   - Recommendation: Use `HealthIssue` for Phase 49. If Phase 50 needs more, extend with a wrapper then.

2. **What severity level for partial phase (build done, continue not run)?**
   - What we know: This is an expected intermediate state if the user just finished building.
   - What's unclear: Whether to report it as "info" (not stuck, just waiting) or "warning" (should run continue).
   - Recommendation: Report as "warning" with message "Build completed but continue not run" -- it IS stuck if the user ran recover to find out why they cannot advance.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | none (standard) |
| Quick run command | `go test ./cmd/ -run TestRecover -v` |
| Full suite command | `go test ./cmd/ -race` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| DETECT-01 | Missing build packet detected | unit | `go test ./cmd/ -run TestScanMissingBuildPacket -v` | Wave 0 |
| DETECT-02 | Stale spawned workers detected | unit | `go test ./cmd/ -run TestScanStaleSpawnedWorkers -v` | Wave 0 |
| DETECT-03 | Partial phase detected | unit | `go test ./cmd/ -run TestScanPartialPhase -v` | Wave 0 |
| DETECT-04 | Bad manifest detected | unit | `go test ./cmd/ -run TestScanBadManifest -v` | Wave 0 |
| DETECT-05 | Dirty worktree detected | unit | `go test ./cmd/ -run TestScanDirtyWorktrees -v` | Wave 0 |
| DETECT-06 | Broken survey detected | unit | `go test ./cmd/ -run TestScanBrokenSurvey -v` | Wave 0 |
| DETECT-07 | Missing agent files detected | unit | `go test ./cmd/ -run TestScanMissingAgentFiles -v` | Wave 0 |
| OUTP-01 | Clean diagnosis output | unit | `go test ./cmd/ -run TestRecoverOutput -v` | Wave 0 |
| OUTP-02 | Exit code 0 when healthy, 1 when issues | unit | `go test ./cmd/ -run TestRecoverExitCode -v` | Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./cmd/ -run TestRecover -v`
- **Per wave merge:** `go test ./... -race`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `cmd/recover_test.go` -- covers all 9 requirements (DETECT-01 through OUTP-02)
- [ ] Test helper for creating synthetic stuck states in temp directories

## Sources

### Primary (HIGH confidence)
- Direct source code analysis of all files listed below (every line read and verified)
- `cmd/medic_cmd.go` -- HealthIssue type, medic report rendering, exit code pattern
- `cmd/medic_scanner.go` -- performHealthScan, scanColonyState, scanSession, scanPheromones, scanDataFiles, scanJSONL, fileChecker, issue helpers
- `cmd/medic_repair.go` -- RepairResult, RepairRecord, createBackup, performRepairs, repairStateIssues, repairPheromoneIssues, repairSessionIssues, repairDataIssues, atomicWriteFile, getGitWorktreePaths, findLastValidJSON
- `cmd/medic_wrapper.go` -- scanWrapperParity, expected agent counts per platform
- `cmd/codex_continue.go` -- detectAbandonedBuild, loadCodexContinueManifest, missingBuildPacketBlockedResult, all continue types and assessment logic
- `cmd/worktree.go` -- worktreeAllocateCmd, worktreeOrphanScanCmd, reportOrphanBranches, isWorktreeOrphaned, getLastCommitTime, agentBranchRe, createBlocker
- `cmd/spawn_track.go` -- spawnTrackEntry, writeSpawnTrack, readSpawnTrack, spawnTrackClear
- `cmd/spawn_runs.go` -- runtimeSpawnRun, beginRuntimeSpawnRun, finishRuntimeSpawnRun, summarizeRunStatus
- `cmd/state_load.go` -- loadActiveColonyState, loadColonyStateWithCompatibilityRepair, normalizeLegacyColonyState
- `cmd/survey.go` -- surveyFiles list, surveyVerifyCmd
- `cmd/codex_visuals.go` -- renderBanner, renderStageMarker, renderNextUp, shouldUseANSIColors
- `pkg/colony/colony.go` -- ColonyState struct, all State/Phase/Task constants, WorktreeEntry, legalTransitions
- `pkg/colony/state_machine.go` -- Transition function, AdvancePhase
- `pkg/agent/spawn_tree.go` -- SpawnEntry, SpawnRun, spawnRunState, SpawnTree

### Secondary (MEDIUM confidence)
- `.planning/research/FEATURES.md` -- 7 stuck-state detection criteria and fix behaviors
- `.planning/research/ARCHITECTURE.md` -- Recommended file structure and pipeline design
- `.planning/research/STACK.md` -- Existing infrastructure mapping

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all functions verified in source code with exact signatures and line numbers
- Architecture: HIGH -- pattern follows existing medic scanner, no new concepts
- Pitfalls: HIGH -- derived from actual code analysis, not assumptions
- Detection order: HIGH -- derived from code dependencies and feature research

**Research date:** 2026-04-25
**Valid until:** 2026-05-25 (stable codebase, low churn expected)
