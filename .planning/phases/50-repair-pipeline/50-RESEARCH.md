# Phase 50: Repair Pipeline - Research

**Researched:** 2026-04-25
**Domain:** Colony state repair for Aether Go CLI (wiring `aether recover --apply`)
**Confidence:** HIGH (all findings verified against source code)

## Summary

Phase 50 wires the `--apply` flag in `cmd/recover.go` so that detected stuck-state issues get automatically repaired. The heavy infrastructure already exists: `cmd/medic_repair.go` provides backup creation (`createBackup`), atomic file writes (`atomicWriteFile`), a full repair orchestrator (`performRepairs`), and category-specific repair functions. Phase 49 built the 7 scanners that return `[]HealthIssue` with `Fixable: true` for all categories. Phase 50 needs to add 7 repair functions (one per category) and an orchestrator that respects the safe-vs-destructive split.

The 5 safe repairs (missing_build_packet, stale_spawned, partial_phase, broken_survey, missing_agents) run automatically with `--apply`. The 2 destructive repairs (dirty_worktree, bad_manifest) require interactive user confirmation unless `--force` is also passed. This mirrors medic's pattern where `repairDataIssues` checks `opts.Force` for corrupted JSON recovery.

**Primary recommendation:** Add a `cmd/recover_repair.go` file with a `performRecoverRepairs` orchestrator and 7 category-specific repair functions. Each repair function follows medic's pattern: read state, apply fix, save via `atomicWriteFile`. The orchestrator creates a backup first, then dispatches by category, sorting critical before warning. Destructive categories prompt the user via stdin unless `--force` is set.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| REPAIR-01 | Auto-fix 5 safe classes | 5 repair functions: `repairMissingBuildPacket`, `repairStaleSpawned`, `repairPartialPhase`, `repairBrokenSurvey`, `repairMissingAgentFiles`. Each reuses medic patterns for state mutation + atomic write. |
| REPAIR-02 | Confirmation for dirty worktree | `repairDirtyWorktree` prompts user per-worktree to stash or discard. Skipped if `--force` set. Uses `exec.Command("git", "stash")` or discard pattern. |
| REPAIR-03 | Confirmation for bad manifest | `repairBadManifest` prompts user before rebuilding manifest from disk state. Skipped if `--force` set. Uses `findLastValidJSON` for partial recovery. |
| REPAIR-04 | Backups before mutations | Reuse `createBackup(dataPath)` from `cmd/medic_repair.go:49`. Already handles timestamped directories, manifest writing, and old backup cleanup. |
| REPAIR-05 | Atomic multi-file repairs | Reuse `atomicWriteFile(path, data)` from `cmd/medic_repair.go:684`. For multi-file repairs, collect all writes and apply sequentially; on failure, restore from backup. |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Colony state mutation | CLI / Go runtime | pkg/storage | State writes go through `atomicWriteFile` or `store.SaveJSON`; reads via `loadActiveColonyState` |
| Spawn run cleanup | CLI / Go runtime | -- | `spawn-runs.json` is a flat JSON file in `.aether/data/` |
| Manifest rebuild | CLI / Go runtime | -- | `build/phase-N/manifest.json` is a per-phase artifact |
| Worktree state sync | CLI / Go runtime | Git CLI | `git stash`, `git checkout`, state array cleanup |
| Survey file regeneration | CLI / Go runtime | -- | Delete broken files from `.aether/data/survey/` |
| Agent file restoration | CLI / Go runtime | Hub filesystem | Copy from `~/.aether/system/{platform}/agents/` to repo |
| User confirmation | CLI / Go runtime | stdin | `fmt.Fprintf(os.Stderr, ...)` + `bufio.NewReader(os.Stdin).ReadString('\n')` |
| Repair rendering | CLI / Go runtime | -- | Reuse `renderStageMarker`, `writeIssueLine` from existing visuals |

## Standard Stack

### Core (all existing, zero new dependencies)

| Library | Location | Purpose | Why reuse |
|---------|----------|---------|-----------|
| `createBackup` | `cmd/medic_repair.go:49` | Timestamped backup of `.aether/data/` | REPAIR-04 requires backup-first; medic already implements this |
| `atomicWriteFile` | `cmd/medic_repair.go:684` | Temp file + rename pattern | REPAIR-05 requires atomic writes; existing impl handles dir creation and cleanup |
| `RepairResult` + `RepairRecord` | `cmd/medic_repair.go:20-38` | Repair outcome tracking | Reuse for recover repair reporting |
| `loadActiveColonyState` | `cmd/state_load.go:17` | Colony state loading with compatibility repair | All state-aware repairs need this |
| `store.SaveJSON` | `pkg/storage/storage.go` | JSON persistence with file locking | Official state write path; some repairs may need this over raw `atomicWriteFile` |
| `loadCodexContinueManifest` | `cmd/codex_continue.go:678` | Build manifest loading | Used by missing_build_packet and partial_phase repairs |
| `logRepairToTrace` | `cmd/medic_repair.go:137` | Trace logging for repair operations | Auditable repair history |
| `cleanupOldBackups` | `cmd/medic_repair.go:101` | Keep only N most recent backups | Prevent unbounded backup growth |

### Detection-to-Repair Mapping

| Category | Scanner Function | Repair Action | Safe/Destructive | Files Touched |
|----------|-----------------|---------------|-----------------|---------------|
| `missing_build_packet` | `scanMissingBuildPacket` | Reset state to READY, clear `build_started_at` | Safe | COLONY_STATE.json |
| `stale_spawned` | `scanStaleSpawnedWorkers` | Reset stale runs to "failed", clear `current_run_id` | Safe | spawn-runs.json |
| `partial_phase` | `scanPartialPhase` | If all dispatches done: transition EXECUTING->BUILT. If never built: reset phase status to pending | Safe | COLONY_STATE.json |
| `broken_survey` | `scanBrokenSurvey` | Delete broken survey files, clear `territory_surveyed` | Safe | survey/*.json, COLONY_STATE.json |
| `missing_agents` | `scanMissingAgentFiles` | Copy missing agent files from hub to repo | Safe | .claude/agents/ant/*, .opencode/agents/*, .codex/agents/* |
| `dirty_worktree` | `scanDirtyWorktrees` | Prompt per-worktree: stash changes, discard changes, or skip. Remove orphan state entries | Destructive (confirmation) | COLONY_STATE.json, worktree files |
| `bad_manifest` | `scanBadManifest` | Prompt: rebuild from disk state (fix phase field, timestamps) or delete | Destructive (confirmation) | build/phase-N/manifest.json |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| New `RecoverRepairResult` type | Reuse `RepairResult` + `RepairRecord` | Same types; recover adds confirmation step at dispatch level, not result level |
| Per-repair backup | Single pre-repair backup | Single backup is simpler and matches medic pattern; per-repair is over-engineering for 7 categories |
| `store.SaveJSON` for all writes | `atomicWriteFile` for all writes | `store.SaveJSON` adds file locking but requires relative paths; `atomicWriteFile` accepts absolute paths and is already used by medic repairs. Use `atomicWriteFile` for consistency with medic. |

## Architecture Patterns

### Recommended Project Structure
```
cmd/
  recover.go              -- EXISTING: add --apply wiring after scan
  recover_scanner.go      -- EXISTING (Phase 49): no changes needed
  recover_visuals.go      -- EXISTING (Phase 49): minor update for repair log rendering
  recover_repair.go       -- NEW: performRecoverRepairs + 7 repair functions
  recover_test.go         -- EXISTING: add repair tests (REPAIR-01 through REPAIR-05)
  medic_repair.go         -- EXISTING: import createBackup, atomicWriteFile, RepairResult, RepairRecord
```

### Pattern 1: Repair Function Signature
**What:** Each repair function takes an issue + data dir and returns a `RepairRecord`.
**When to use:** For each of the 7 stuck-state categories.
**Example:**
```go
// Source: [VERIFIED: cmd/medic_repair.go pattern -- repairStateIssues signature]
func repairMissingBuildPacket(issue HealthIssue, dataDir string) RepairRecord {
    record := RepairRecord{
        Category: "missing_build_packet",
        File:     issue.File,
    }

    statePath := filepath.Join(dataDir, "COLONY_STATE.json")
    data, err := os.ReadFile(statePath)
    if err != nil {
        record.Error = fmt.Sprintf("read state: %v", err)
        return record
    }

    var state colony.ColonyState
    if err := json.Unmarshal(data, &state); err != nil {
        record.Error = fmt.Sprintf("parse state: %v", err)
        return record
    }

    record.Before = string(state.State)

    // Reset to READY so user can re-dispatch
    state.State = colony.StateREADY
    state.BuildStartedAt = nil

    encoded, err := json.MarshalIndent(state, "", "  ")
    if err != nil {
        record.Error = fmt.Sprintf("marshal state: %v", err)
        return record
    }
    encoded = append(encoded, '\n')
    if err := atomicWriteFile(statePath, encoded); err != nil {
        record.Error = fmt.Sprintf("write state: %v", err)
        return record
    }

    record.Action = "reset_to_ready"
    record.After = string(colony.StateREADY)
    record.Success = true
    return record
}
```

### Pattern 2: Repair Orchestrator
**What:** Single function that backs up, filters, sorts, dispatches, and renders repair results.
**When to use:** Called by `runRecover` when `--apply` is set.
**Example:**
```go
// Source: [VERIFIED: cmd/medic_repair.go:182 performRepairs pattern]
func performRecoverRepairs(issues []HealthIssue, dataDir string, force bool) (*RepairResult, error) {
    // 1. Backup
    backupPath, err := createBackup(dataDir)
    if err != nil {
        return nil, fmt.Errorf("backup failed: %w", err)
    }
    _ = cleanupOldBackups(filepath.Dir(backupPath), 3)

    // 2. Filter to fixable issues
    var fixable []HealthIssue
    for _, issue := range issues {
        if issue.Fixable {
            fixable = append(fixable, issue)
        }
    }

    // 3. Sort: critical first, then warning, then info
    sort.SliceStable(fixable, func(i, j int) bool {
        pi, _ := severityOrder[fixable[i].Severity]
        pj, _ := severityOrder[fixable[j].Severity]
        return pi < pj
    })

    // 4. Deduplicate by category+message
    result := &RepairResult{Attempted: len(fixable)}
    seen := make(map[string]bool)
    for _, issue := range fixable {
        key := issue.Category + ":" + issue.Message
        if seen[key] {
            continue
        }
        seen[key] = true

        // Check for destructive categories requiring confirmation
        if isDestructiveCategory(issue.Category) && !force {
            confirmed := confirmRepair(issue)
            if !confirmed {
                result.Skipped++
                result.Repairs = append(result.Repairs, RepairRecord{
                    Category: issue.Category,
                    File:     issue.File,
                    Action:   "skip",
                    Error:    "user declined",
                })
                continue
            }
        }

        record := dispatchRecoverRepair(issue, dataDir, force)
        // ... track result
    }
    return result, nil
}
```

### Pattern 3: Destructive Category Check
**What:** Centralized function to determine if a category needs user confirmation.
**When to use:** In the repair orchestrator, before dispatching destructive repairs.
```go
func isDestructiveCategory(category string) bool {
    return category == "dirty_worktree" || category == "bad_manifest"
}
```

### Pattern 4: User Confirmation Prompt
**What:** Interactive prompt for destructive operations.
**When to use:** When `isDestructiveCategory` returns true and `--force` is not set.
```go
func confirmRepair(issue HealthIssue) bool {
    fmt.Fprintf(os.Stderr, "\n  [confirm] %s (%s)\n", issue.Message, issue.File)
    fmt.Fprintf(os.Stderr, "  Apply fix? [y/N]: ")

    reader := bufio.NewReader(os.Stdin)
    response, err := reader.ReadString('\n')
    if err != nil {
        return false
    }
    response = strings.TrimSpace(strings.ToLower(response))
    return response == "y" || response == "yes"
}
```

### Anti-Patterns to Avoid
- **Do not modify scanners:** Phase 49's `recover_scanner.go` is complete. Repairs are a separate concern.
- **Do not duplicate medic repairs:** Recover has different repair semantics than medic (stuck-state-specific). Keep them separate even when both touch the same files.
- **Do not skip backup for "small" repairs:** Every repair, even deleting a single survey file, must go through `createBackup` first.
- **Do not use `store.SaveJSON` for files outside `.aether/data/`:** Agent files live in `.claude/`, `.opencode/`, `.codex/` -- use `atomicWriteFile` or direct `os.WriteFile` for those.
- **Do not prompt in JSON output mode:** When `--json` flag is set, destructive repairs should be skipped (not prompted), matching how API consumers expect non-interactive behavior.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| State backup | Custom copy logic | `createBackup(dataPath)` from `cmd/medic_repair.go:49` | Handles dirs, files, manifests, and cleanup |
| Atomic file writes | Custom rename logic | `atomicWriteFile(path, data)` from `cmd/medic_repair.go:684` | Temp file + rename + cleanup on failure |
| Corrupted JSON recovery | Custom bracket-matching | `findLastValidJSON(raw)` from `cmd/medic_repair.go:768` | Scans backwards for valid JSON closings |
| Stale spawn reset | Custom spawn-runs repair | Pattern from `repairDataIssues` in `cmd/medic_repair.go:618-668` | Identical logic: parse, iterate, mark "failed", write |
| Repair tracing | Custom logging | `logRepairToTrace(record, dataPath)` from `cmd/medic_repair.go:137` | JSONL trace entries with repair metadata |
| Old backup cleanup | Custom directory pruning | `cleanupOldBackups(dir, keep)` from `cmd/medic_repair.go:101` | Keeps N most recent, removes by timestamp |
| Colony state loading | Direct file read | `loadActiveColonyState()` | Handles compatibility repair, legacy normalization |
| Agent file copying from hub | Custom copy loop | Pattern from `syncDirToHubWithExclusion` in `cmd/install_cmd.go:726` | SHA-256 comparison, validation, exclusion filters |
| State transitions | Direct `state.State = X` | `colony.Transition(current, target)` from `pkg/colony/state_machine.go` | Validates legal transitions before applying |
| Git worktree operations | Custom path manipulation | `exec.Command("git", ...)` + `getGitWorktreePaths()` | Git porcelain is the source of truth |

**Key insight:** The repair pipeline is 90% wiring. Every low-level operation (backup, atomic write, state loading, git interaction) already exists. The new code is orchestration logic (which repair to run, in what order, with what confirmation) and the 7 category-specific fix implementations.

## Common Pitfalls

### Pitfall 1: Repair Order Dependencies (Same as Detection)
**What goes wrong:** Repairing "partial phase" before "bad manifest" applies a fix based on corrupted data.
**Why it happens:** Manifest must be valid before partial-phase repair can safely transition state.
**How to avoid:** Run repairs in the same order as detection: stale_spawned -> bad_manifest -> missing_build_packet -> partial_phase -> dirty_worktree -> broken_survey -> missing_agents. Sort by severity first (critical before warning), then by this category order.
**Warning signs:** State transitions to BUILT when manifest is actually corrupted.

### Pitfall 2: Partial Repair on Multi-File Fix
**What goes wrong:** `broken_survey` needs to delete files AND update COLONY_STATE.json. If the state update fails after files are deleted, state is inconsistent.
**Why it happens:** Two separate operations without transaction semantics.
**How to avoid:** For multi-file repairs, apply the least-destructive operation first (state update), then the more-destructive one (file deletion). If the first fails, the second is skipped. The backup covers full rollback if needed.
**Warning signs:** Survey files deleted but `territory_surveyed` still set, causing re-detection.

### Pitfall 3: Dirty Worktree Stash Fails
**What goes wrong:** `git stash` fails because the worktree has untracked files that conflict with stash rules.
**Why it happens:** `git stash` by default does not include untracked files (needs `-u` flag).
**How to avoid:** Use `git stash --include-untracked` or `git checkout -- .` for tracked files only. Document the choice in the repair record.
**Warning signs:** Stash command returns non-zero exit code but repair continues.

### Pitfall 4: Agent File Restoration When Hub Is Empty
**What goes wrong:** `missing_agents` repair tries to copy from `~/.aether/system/claude/agents/` but that directory is empty or does not exist.
**Why it happens:** Hub was never populated (user never ran `aether publish` or `aether install`).
**How to avoid:** Check hub directory existence before attempting copy. If hub is empty, skip repair and report "hub has no agent files -- run `aether update` first".
**Warning signs:** Repair record shows success with 0 files copied.

### Pitfall 5: Re-Detection After Repair
**What goes wrong:** After repair, the scanner re-runs and finds the same issue because the repair did not fully address the root cause.
**Why it happens:** Some repairs have secondary effects. For example, resetting `state.State` to READY does not clear `build_started_at`, so the scanner still sees a "build was started" signal.
**How to avoid:** After repair, re-scan and report remaining issues. The `runRecover` function should scan -> repair -> re-scan -> render, matching medic's pattern in `cmd/medic_cmd.go:114-121`.
**Warning signs:** Same issue appears in consecutive `aether recover` runs.

### Pitfall 6: Confirmation Prompts in Piped/Non-Interactive Mode
**What goes wrong:** `bufio.NewReader(os.Stdin).ReadString('\n')` blocks forever when stdin is a pipe (e.g., `aether recover --apply | cat`).
**Why it happens:** Pipes have no user to provide input.
**How to avoid:** Check `os.Stdin.Stat()` for character device, or check `term.IsTerminal(int(os.Stdin.Fd()))`. If not a terminal, skip destructive repairs unless `--force` is set. Alternatively, check `os.Getenv("CI")` or use the existing `shouldRenderVisualOutput` pattern.
**Warning signs:** `aether recover --apply` hangs in CI pipelines.

### Pitfall 7: Phase 49 Visual Bug With Fixable Hints
**What goes wrong:** `recoverFixHint` in `recover_visuals.go:131` only shows hints for NON-fixable issues, but dirty_worktree and bad_manifest ARE marked `Fixable: true`. This means the "Needs --apply with confirmation" hint is never shown.
**Why it happens:** Phase 49 set all 7 categories to `Fixable: true` but kept the hint logic for non-fixable only.
**How to avoid:** Phase 50 should add a `Destructive` field to the issue or add a `isDestructiveCategory` check in the visual rendering to show "Needs confirmation" for destructive categories even though they are fixable. Update `writeRecoverIssueLine` to check destructive status.
**Warning signs:** User sees "[fixable]" for dirty_worktree but no indication that confirmation is needed.

## Code Examples

### Backup Before Repairs
```go
// Source: [VERIFIED: cmd/medic_repair.go:49]
backupPath, err := createBackup(dataDir)
if err != nil {
    return nil, fmt.Errorf("backup failed: %w", err)
}
// backupPath is like: .aether/backups/recover-20260425-143000/
```

### Reset Stale Spawn Runs
```go
// Source: [VERIFIED: cmd/medic_repair.go:618-668 -- reset_stale_spawn_state pattern]
func repairStaleSpawned(issue HealthIssue, dataDir string) RepairRecord {
    record := RepairRecord{Category: "stale_spawned", File: issue.File}

    spawnPath := filepath.Join(dataDir, "spawn-runs.json")
    raw, err := os.ReadFile(spawnPath)
    if err != nil {
        record.Error = fmt.Sprintf("read spawn-runs: %v", err)
        return record
    }

    var spawnState struct {
        CurrentRunID string `json:"current_run_id"`
        Runs         []struct {
            ID        string `json:"id"`
            StartedAt string `json:"started_at"`
            Status    string `json:"status"`
        } `json:"runs"`
    }
    if err := json.Unmarshal(raw, &spawnState); err != nil {
        record.Error = fmt.Sprintf("parse spawn-runs: %v", err)
        return record
    }

    reset := 0
    for i := range spawnState.Runs {
        run := &spawnState.Runs[i]
        if run.Status == "running" || run.Status == "active" {
            started := parseTimestamp(run.StartedAt)
            if !started.IsZero() && time.Since(started) > time.Hour {
                run.Status = "failed"
                reset++
            }
        }
    }

    if reset > 0 {
        spawnState.CurrentRunID = ""
        encoded, err := json.MarshalIndent(spawnState, "", "  ")
        if err != nil {
            record.Error = fmt.Sprintf("marshal: %v", err)
            return record
        }
        encoded = append(encoded, '\n')
        if err := atomicWriteFile(spawnPath, encoded); err != nil {
            record.Error = fmt.Sprintf("write: %v", err)
            return record
        }
    }

    record.Action = "reset_stale_spawns"
    record.After = fmt.Sprintf("reset %d stale runs to failed", reset)
    record.Success = reset > 0
    return record
}
```

### Legal State Transition
```go
// Source: [VERIFIED: pkg/colony/state_machine.go:7]
// Must validate before applying state change:
if err := colony.Transition(state.State, colony.StateREADY); err != nil {
    record.Error = fmt.Sprintf("invalid transition: %v", err)
    return record
}
state.State = colony.StateREADY
```

### Manifest Repair (Destructive)
```go
// Source: [VERIFIED: cmd/medic_repair.go:568-578 -- findLastValidJSON pattern]
func repairBadManifest(issue HealthIssue, dataDir string) RepairRecord {
    record := RepairRecord{Category: "bad_manifest", File: issue.File}

    filePath := filepath.Join(dataDir, issue.File)
    raw, err := os.ReadFile(filePath)
    if err != nil {
        record.Error = fmt.Sprintf("read manifest: %v", err)
        return record
    }

    // Try to recover valid JSON from corrupted content
    recovered := findLastValidJSON(raw)
    if recovered == nil {
        // Delete the corrupted manifest entirely -- missing packet detection
        // will handle the rest on next scan.
        if err := os.Remove(filePath); err != nil {
            record.Error = fmt.Sprintf("remove corrupt manifest: %v", err)
            return record
        }
        record.Action = "remove_corrupt_manifest"
        record.Success = true
        return record
    }

    // Fix specific fields that were wrong
    var manifest codexBuildManifest
    if err := json.Unmarshal(recovered, &manifest); err != nil {
        record.Error = fmt.Sprintf("recovered JSON still invalid: %v", err)
        return record
    }

    // Fix phase mismatch, empty fields, etc.
    // (parse state to get correct phase ID)
    // ... field corrections ...

    encoded, err := json.MarshalIndent(manifest, "", "  ")
    if err != nil {
        record.Error = fmt.Sprintf("marshal repaired: %v", err)
        return record
    }
    encoded = append(encoded, '\n')
    if err := atomicWriteFile(filePath, encoded); err != nil {
        record.Error = fmt.Sprintf("write repaired: %v", err)
        return record
    }

    record.Action = "repair_manifest_fields"
    record.Success = true
    return record
}
```

### Agent File Restoration
```go
// Source: [VERIFIED: cmd/install_cmd.go:726 -- syncDirToHubWithExclusion pattern]
func repairMissingAgentFiles(issue HealthIssue, dataDir string) RepairRecord {
    record := RepairRecord{Category: "missing_agents", File: issue.File}

    repoRoot := resolveAetherRoot()
    hubDir := filepath.Join(homeDir(), ".aether", "system")

    surfaces := []struct {
        name    string
        hubSrc  string
        repoDst string
    }{
        {"claude", filepath.Join(hubDir, "claude", "agents"), filepath.Join(repoRoot, ".claude", "agents", "ant")},
        {"opencode", filepath.Join(hubDir, "opencode", "agents"), filepath.Join(repoRoot, ".opencode", "agents")},
        {"codex", filepath.Join(hubDir, "codex", "agents"), filepath.Join(repoRoot, ".codex", "agents")},
    }

    copied := 0
    for _, surface := range surfaces {
        if _, err := os.Stat(surface.hubSrc); os.IsNotExist(err) {
            continue // No hub files for this platform
        }
        files, _ := filepath.Glob(filepath.Join(surface.hubSrc, "*"))
        for _, srcFile := range files {
            basename := filepath.Base(srcFile)
            dstFile := filepath.Join(surface.repoDst, basename)
            if _, err := os.Stat(dstFile); os.IsNotExist(err) {
                os.MkdirAll(surface.repoDst, 0755)
                if data, err := os.ReadFile(srcFile); err == nil {
                    if err := os.WriteFile(dstFile, data, 0644); err == nil {
                        copied++
                    }
                }
            }
        }
    }

    record.Action = "restore_agent_files"
    record.After = fmt.Sprintf("restored %d agent files from hub", copied)
    record.Success = copied > 0
    return record
}
```

### Dirty Worktree Repair (Destructive with Confirmation)
```go
func repairDirtyWorktree(issue HealthIssue, dataDir string, force bool) RepairRecord {
    record := RepairRecord{Category: "dirty_worktree", File: issue.File}

    statePath := filepath.Join(dataDir, "COLONY_STATE.json")
    data, err := os.ReadFile(statePath)
    if err != nil {
        record.Error = fmt.Sprintf("read state: %v", err)
        return record
    }

    var state colony.ColonyState
    if err := json.Unmarshal(data, &state); err != nil {
        record.Error = fmt.Sprintf("parse state: %v", err)
        return record
    }

    // Different sub-types need different fixes:
    if strings.Contains(issue.Message, "state-disk mismatch") || strings.Contains(issue.Message, "not in git worktree list") {
        // Remove orphaned state entry
        record.Action = "remove_orphan_worktree_entry"
        var remaining []colony.WorktreeEntry
        for _, wt := range state.Worktrees {
            if wt.Path != issue.File {
                remaining = append(remaining, wt)
            }
        }
        state.Worktrees = remaining
    } else if strings.Contains(issue.Message, "uncommitted change") {
        // User was already prompted at orchestrator level.
        // Stash changes in worktree.
        cmd := exec.Command("git", "-C", issue.File, "stash", "--include-untracked")
        if err := cmd.Run(); err != nil {
            record.Error = fmt.Sprintf("git stash failed: %v", err)
            return record
        }
        record.Action = "stash_worktree_changes"
    } else if strings.Contains(issue.Message, "Orphan branch") {
        // Delete orphan branch
        branchName := filepath.Base(issue.File)
        cmd := exec.Command("git", "branch", "-D", branchName)
        if err := cmd.Run(); err != nil {
            record.Error = fmt.Sprintf("delete branch failed: %v", err)
            return record
        }
        record.Action = "delete_orphan_branch"
    }

    // Save updated state
    encoded, err := json.MarshalIndent(state, "", "  ")
    if err != nil {
        record.Error = fmt.Sprintf("marshal state: %v", err)
        return record
    }
    encoded = append(encoded, '\n')
    if err := atomicWriteFile(statePath, encoded); err != nil {
        record.Error = fmt.Sprintf("write state: %v", err)
        return record
    }

    record.Success = true
    return record
}
```

## Repair Dispatch Table

The repair orchestrator dispatches by `issue.Category`:

| Category | Safe? | Repair Function | Key Action | State Mutated |
|----------|-------|-----------------|------------|---------------|
| `missing_build_packet` | Yes | `repairMissingBuildPacket` | State -> READY, clear `build_started_at` | COLONY_STATE.json |
| `stale_spawned` | Yes | `repairStaleSpawned` | Mark stale runs "failed", clear `current_run_id` | spawn-runs.json |
| `partial_phase` | Yes | `repairPartialPhase` | If all done: EXECUTING->BUILT. If never built: phase status -> pending | COLONY_STATE.json |
| `broken_survey` | Yes | `repairBrokenSurvey` | Delete broken files, clear `territory_surveyed` | survey/*.json + COLONY_STATE.json |
| `missing_agents` | Yes | `repairMissingAgentFiles` | Copy from hub to repo | .claude/agents/*, .opencode/agents/*, .codex/agents/* |
| `dirty_worktree` | No | `repairDirtyWorktree` | Stash/discard changes, remove orphan entries/branches | COLONY_STATE.json + git worktree state |
| `bad_manifest` | No | `repairBadManifest` | Recover JSON or delete, fix phase/field mismatches | build/phase-N/manifest.json |

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| No recovery mechanism | `build --force` manual reset | v1.7 | Specific path but no scanner |
| Manual state editing | `aether medic --fix` | v1.4 (Phase 25) | General health, not stuck-state |
| Phase 49 scanner only | Scanner + repair pipeline | Phase 50 (this) | Full detect-and-fix lifecycle |

**Deprecated/outdated:**
- The `recoverFixHint` function in `recover_visuals.go` has a logic bug: it only shows hints for non-fixable issues, but dirty_worktree and bad_manifest are fixable. Phase 50 should fix this.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `--force` flag on recover should skip user confirmation for destructive repairs, matching medic's pattern | Pattern 4 | If the intent is different (e.g., --force enables different repair behavior), the confirmation logic needs adjustment |
| A2 | Broken survey repair should delete broken files and clear `territory_surveyed`, not attempt to regenerate them | Repair Dispatch Table | If regeneration is expected, need to call into survey subsystem |
| A3 | Agent file restoration copies from hub only (not from Aether source repo) | Pattern 4 code example | If hub is empty but source repo has files, repair will not fix the issue |
| A4 | Dirty worktree "uncommitted changes" repair should stash by default, not discard | Pattern code example | If discard is preferred, need to prompt for choice (stash vs discard) |
| A5 | JSON output mode (`--json`) should skip destructive repairs rather than prompting | Anti-patterns section | If JSON mode should still repair destructively, need non-interactive confirmation mechanism |
| A6 | The `confirmRepair` function reads from `os.Stdin`; this works when recover is run directly but may not work when spawned by other commands | Pattern 4 | If recover is called from colony-prime context, stdin may not be available |

## Open Questions

1. **Should dirty worktree repair offer stash vs discard choice?**
   - What we know: REPAIR-02 says "stash or discard" -- implying a choice.
   - What's unclear: Whether the user picks per-worktree or globally.
   - Recommendation: Prompt per-worktree with options: `[s]tash / [d]iscard / [S]kip`. Default to skip (safest).

2. **Should broken survey repair delete files or attempt regeneration?**
   - What we know: `aether colonize` regenerates survey data. The visual hint says "Run `aether colonize` to regenerate."
   - What's unclear: Whether recover should call colonize internally or just clean up broken files.
   - Recommendation: Just delete broken files and clear `territory_surveyed`. Regeneration is a separate step (user runs `aether colonize`). This keeps recover focused on state cleanup, not data generation.

3. **Should the re-scan after repair be automatic?**
   - What we know: Medic re-scans after repair (`cmd/medic_cmd.go:114-121`).
   - What's unclear: Whether recover should show the post-repair scan or just the repair log.
   - Recommendation: Follow medic's pattern: scan -> repair -> re-scan -> render. The re-scan shows remaining issues that were not fixable or that the repair did not address.

## Environment Availability

Step 2.6: SKIPPED (no external dependencies -- all repair operations use Go stdlib, existing pkg/ code, and git CLI which is already a hard dependency of the project).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | none (standard) |
| Quick run command | `go test ./cmd/ -run TestRepair -v` |
| Full suite command | `go test ./cmd/ -race` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| REPAIR-01 | 5 safe auto-fixes work | unit (5 tests, one per category) | `go test ./cmd/ -run TestRepair_Missing -v` etc. | Wave 0 |
| REPAIR-02 | Confirmation for dirty worktree | unit | `go test ./cmd/ -run TestRepair_Dirty -v` | Wave 0 |
| REPAIR-03 | Confirmation for bad manifest | unit | `go test ./cmd/ -run TestRepair_Manifest -v` | Wave 0 |
| REPAIR-04 | Backup created before repair | unit | `go test ./cmd/ -run TestRepairBackup -v` | Wave 0 |
| REPAIR-05 | Atomic multi-file repair | unit | `go test ./cmd/ -run TestRepairAtomic -v` | Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./cmd/ -run TestRepair -v`
- **Per wave merge:** `go test ./... -race`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `cmd/recover_test.go` -- add repair tests (REPAIR-01 through REPAIR-05) alongside existing scanner tests
- [ ] Test helper for simulating confirmation prompts (override stdin)
- [ ] Test helper for simulating hub directory with agent files

## Sources

### Primary (HIGH confidence)
- Direct source code analysis of all files below (every line read and verified)
- `cmd/recover.go` -- existing cobra command, flag parsing, `runRecover` stub with `_ = apply`
- `cmd/recover_scanner.go` -- 7 detection functions returning `[]HealthIssue`
- `cmd/recover_visuals.go` -- output rendering, exit code, `recoverFixHint` (has bug noted in Assumptions)
- `cmd/medic_repair.go` -- `createBackup`, `atomicWriteFile`, `RepairResult`, `RepairRecord`, `performRepairs`, category dispatch pattern
- `cmd/medic_cmd.go` -- `HealthIssue` type, `writeIssueLine`, `severityColor`, medic repair flow (scan -> repair -> re-scan -> render)
- `pkg/colony/state_machine.go` -- `Transition` function for legal state validation
- `pkg/colony/colony.go` -- `ColonyState`, `WorktreeEntry`, state/worktree constants, legal transitions
- `cmd/install_cmd.go` -- `syncDirToHubWithExclusion` pattern for hub-to-repo file copying

### Secondary (MEDIUM confidence)
- `.planning/phases/49-stuck-state-scanner-and-diagnosis/49-RESEARCH.md` -- Phase 49 research with detection patterns and file layout
- `.planning/REQUIREMENTS.md` -- REPAIR-01 through REPAIR-05 definitions

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all functions verified in source code with exact signatures and line numbers
- Architecture: HIGH -- pattern follows existing medic repair pipeline, no new concepts
- Pitfalls: HIGH -- derived from actual code analysis and requirement interpretation
- Repair logic: HIGH -- each repair action maps to existing code patterns (spawn reset, state transition, JSON recovery, file copy)

**Research date:** 2026-04-25
**Valid until:** 2026-05-25 (stable codebase, low churn expected)
