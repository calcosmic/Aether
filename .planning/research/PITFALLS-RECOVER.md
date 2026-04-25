# Pitfalls: `aether recover` (v1.8 Colony Recovery)

**Domain:** Stuck-state colony recovery for the Aether Go CLI
**Researched:** 2026-04-25
**Scope:** Pitfalls specific to building `aether recover` -- detecting, diagnosing, and auto-fixing 7 classes of stuck colony state.

---

## Critical Pitfalls

These cause data loss, wrong state, or unrecoverable situations.

### Critical 1: "Fixing" a colony that is not actually stuck

**What goes wrong:** The recovery scanner detects a false positive (e.g., state=EXECUTING with a build that genuinely started seconds ago) and resets it to READY, killing a legitimate in-progress build.

**Why it happens:** The scanner cannot distinguish between "stale EXECUTING from a crashed session" and "EXECUTING because a build is currently running in another terminal." The existing medic scanner does not face this because it is always read-only by default. Recovery with `--apply` crosses that line.

**Consequences:** Active build artifacts get overwritten, spawn state gets reset, and the user loses real work that was in progress.

**Prevention:**
- Require a staleness threshold before declaring EXECUTING stuck. The existing `abandonedBuildThreshold` (used in `recovery_snapshot.go` line ~209) already defines this boundary. Recovery must use the same threshold, not instant detection.
- Check for active file locks before applying any fix. The `FileLocker` in `pkg/storage/lock.go` tracks held locks. If `COLONY_STATE.json` is locked by another process, refuse to apply.
- Add a `--force` override that bypasses the lock check only when explicitly requested, mirroring the medic `--force` pattern.

**Detection:** If the build was started less than `abandonedBuildThreshold` ago, report "build may still be in progress" instead of declaring it stuck.

---

### Critical 2: Destroying real worker artifacts during cleanup

**What goes wrong:** Recovery removes planning artifacts, build packets, or survey files that were actually written by real workers -- not stale fallback artifacts.

**Why it happens:** The recovery system cannot tell the difference between a real worker output and a stale leftover without provenance information. The existing `clearFallbackPlanningArtifacts` function (in `cmd/codex_plan.go` lines 1696-1730) solves this for the plan case by using the `.fallback-marker` timestamp: files newer than the marker are preserved because a real worker wrote them.

**Consequences:** Real scout reports, route-setter plans, and build results are deleted. The user must re-run planning or rebuilding from scratch.

**Prevention:**
- Use the existing artifact snapshot system (`snapshotRelativeFiles` + `shouldPreserveWorkerArtifact`) for any artifact cleanup. This is already proven in the planning pipeline.
- For new artifact types not covered by the snapshot system, check modification times against the colony's `BuildStartedAt` or `Plan.GeneratedAt` timestamps. Files newer than the last known command start are real worker outputs.
- Never delete the `planning/phase-plan.json` without first attempting `repairMissingPlanFromArtifacts` (already implemented in `cmd/state_repair.go`). This function can reconstruct the plan from the artifact if COLONY_STATE.json lost it.

**Detection:** Before deleting any artifact file, verify it is not the newest version of that file (i.e., there is no backup or post-dating evidence).

---

### Critical 3: Partial recovery leaving state inconsistent

**What goes wrong:** Recovery fixes one subsystem (e.g., resets EXECUTING to READY) but leaves related subsystems in an inconsistent state (e.g., spawn-runs.json still shows an active run, the build directory still has a partial continue.json).

**Why it happens:** Colony state is spread across multiple files: `COLONY_STATE.json`, `session.json`, `pheromones.json`, `spawn-runs.json`, `spawn-tree.txt`, `build/phase-N/continue.json`, and `planning/phase-plan.json`. A recovery operation that touches one must touch all related files.

**Consequences:** The colony enters an impossible state that confuses subsequent commands. For example, `aether status` shows READY but `aether build N` refuses because spawn-runs.json says a run is active.

**Prevention:**
- Recovery operations must be transactional across files. Define "recovery units" that group related file mutations. For example, the "stale EXECUTING recovery unit" must simultaneously: (1) reset state to READY in COLONY_STATE.json, (2) reset current_run_id in spawn-runs.json, (3) update session.json to match, (4) add a recovery event to COLONY_STATE.json events array.
- The existing `performRepairs` function in `cmd/medic_repair.go` already sorts by severity and deduplicates by category+message. But it processes repairs one-at-a-time without coordination. Recovery needs a higher-level orchestrator that handles inter-file dependencies.
- After all recovery mutations, perform a consistency check (similar to the existing medic post-repair re-scan at `medic_cmd.go` line 119).

**Detection:** The post-recovery re-scan should catch inconsistencies. If it finds any critical issues, report "recovery was partial" and list what still needs attention.

---

### Critical 4: Race condition with concurrent commands

**What goes wrong:** User runs `aether recover --apply` while `aether build` is running in another terminal. Both try to write COLONY_STATE.json simultaneously.

**Why it happens:** The `FileLocker` provides per-file exclusive locks, but recovery needs to acquire locks on multiple files atomically. The current locker only supports single-file locking. Two commands could each hold a lock on different files and then deadlock, or worse, one could read stale state while the other is mid-write.

**Consequences:** Corrupted JSON (partial writes), lost updates, or deadlocked processes.

**Prevention:**
- Recovery must acquire all necessary file locks before starting any mutations, in a consistent order (alphabetical by filename is safe). Release all locks only after all writes complete.
- If any lock acquisition fails (file is locked by another process), abort the entire recovery and report which process holds the lock. Do not attempt partial recovery.
- Add a top-level "recovery lock" file (`.aether/locks/recovery.lock`) that prevents concurrent recovery operations. This is a simpler guard than multi-file locking and prevents the worst case.

**Detection:** Use the existing `platformLockFile` (flock on Unix, LockFileEx on Windows) for the recovery lock. Non-blocking acquisition with immediate failure if already held.

---

### Critical 5: Over-aggressive phase progress reset

**What goes wrong:** Recovery infers that no phases were completed (because the events array is malformed or missing) and resets all phases to pending, erasing real progress through 5+ completed phases.

**Why it happens:** The `inferRecoveredPhaseProgress` function in `cmd/state_repair.go` has sophisticated heuristics (event parsing, continue report scanning), but these depend on data being present and correctly formatted. If both events and continue reports are missing or corrupted, it falls back to "no completed phases."

**Consequences:** The user must re-run builds for phases they already completed. Real work is not lost (files still exist on disk), but the colony's record of progress is destroyed.

**Prevention:**
- Before resetting any phase progress, check for physical evidence on disk. If `build/phase-N/` directories exist with continue.json files showing `advanced: true` or `completed: true`, those phases should be marked completed regardless of what COLONY_STATE.json says.
- The existing `highestCompletedPhaseFromContinueReports` function already does this (scans build directories). Recovery must use this as a floor -- never set completed phases below what disk evidence proves.
- Add a "recovery confidence" score to each fix. Low-confidence fixes (those relying on heuristics without disk evidence) should require `--force`.

**Detection:** Compare inferred progress against disk evidence. If they disagree by more than 1 phase, warn the user instead of auto-fixing.

---

## Moderate Pitfalls

### Moderate 1: Redundant detection with existing medic

**What goes wrong:** `aether recover` re-implements checks that `aether medic` already performs, creating two parallel diagnosis systems that can disagree.

**Prevention:**
- Recovery should build on the medic scanner, not replace it. The `performHealthScan` function and `ScannerResult` type are already well-structured. Recovery adds its own stuck-state detectors (missing build packet, stale spawned workers, partial phase) but should reuse the existing scanner for state, session, pheromone, and data file checks.
- Recovery-specific checks are: (1) stale EXECUTING/BUILT state, (2) missing build packet for current phase, (3) spawned workers that never completed, (4) stale fallback planning artifacts, (5) dirty worktree state, (6) broken survey data preventing planning, (7) missing agent files preventing dispatch. These are different from medic's corruption/malformation checks.

### Moderate 2: Confusing output when nothing is wrong

**What goes wrong:** User runs `aether recover` on a healthy colony and gets a wall of diagnostic output that makes them think something is broken.

**Prevention:**
- When no stuck-state conditions are detected, output a single clean line: "Colony is healthy. No recovery needed." Do not show the full scan results unless `--verbose` is passed.
- The medic scanner already has a "healthy" path (`medic_cmd.go` line 248-249). Follow the same pattern.

### Moderate 3: Recovery interaction with `plan --force`

**What goes wrong:** User runs `aether recover --apply` which resets planning artifacts, then runs `aether plan --force` which also resets planning artifacts. The double-reset loses the backup from the first operation.

**Prevention:**
- Recovery should not duplicate what `plan --force` does. If recovery detects stale planning artifacts, it should recommend `aether plan --force` rather than performing the cleanup itself.
- Define clear responsibility boundaries: recovery handles runtime state (EXECUTING, spawn runs, worktrees), while `plan --force` handles planning artifacts. They should not overlap.

### Moderate 4: Backup explosion

**What goes wrong:** Every `recover --apply` creates a full backup of `.aether/data/`. If the user runs it multiple times, the backup directory grows unbounded.

**Prevention:**
- The existing `cleanupOldBackups` function in `cmd/medic_repair.go` already handles this (keeps last 3). Reuse it.
- Use the same backup naming convention (`medic-TIMESTAMP`) so backups from both medic and recover share the same rotation pool.

### Moderate 5: Worktree cleanup destroying uncommitted work

**What goes wrong:** Recovery detects orphaned worktrees and removes them. But the worktrees contain uncommitted changes that the user was working on.

**Prevention:**
- The existing `repairStateIssues` for orphaned worktrees (medic_repair.go lines 297-317) already checks `getGitWorktreePaths()` -- it only removes entries where the git worktree no longer exists on disk. This is safe.
- Recovery should go further and check for uncommitted changes in non-orphaned worktrees before suggesting cleanup. If `git status --porcelain` returns anything, warn instead of auto-cleaning.
- Never auto-delete worktrees with `--apply`. Only remove state entries that reference non-existent worktrees. Physical worktree cleanup should be a separate explicit command.

---

## Minor Pitfalls

### Minor 1: Output format inconsistency between medic and recover

**Prevention:** Use the same visual rendering functions (`renderBanner`, `renderStageMarker`, `renderNextUp`) that medic uses. Do not invent new output formats.

### Minor 2: JSON mode output missing recovery-specific fields

**Prevention:** The JSON output should include `recovery_actions` (list of actions taken), `recovery_confidence` (per-action confidence score), and `rollback_path` (backup directory path). Model this on `renderMedicJSON` which already includes `repairs` with `attempted`, `succeeded`, `failed`, `skipped`.

### Minor 3: Trace logging missing for recovery operations

**Prevention:** Use the existing `logRepairToTrace` function for each recovery action. Tag them with topic `recover.repair` instead of `medic.repair` so they are distinguishable in trace analysis.

### Minor 4: Recovery not idempotent

**What goes wrong:** Running `aether recover --apply` twice applies the same fix twice, potentially creating duplicate events or redundant state mutations.

**Prevention:** Each recovery action should check whether the fix is already applied before applying it. For example, resetting EXECUTING to READY should first verify the state is still EXECUTING. Use compare-and-swap semantics: read state, verify condition, apply fix, write state -- all under the same lock.

---

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|---------------|------------|
| Detection scanner design | Critical 1: False positives on active builds | Use `abandonedBuildThreshold` as floor for staleness |
| `--apply` fix engine | Critical 3: Partial recovery inconsistency | Define recovery units that span all related files |
| Stale spawn run cleanup | Critical 4: Race with active build | Check file locks before modifying spawn-runs.json |
| Planning artifact cleanup | Critical 2: Deleting real worker artifacts | Use `.fallback-marker` timestamp guard (proven pattern) |
| Worktree state repair | Moderate 5: Destroying uncommitted work | Check `git status --porcelain` before any cleanup |
| Output and UX | Moderate 2: Scary output on healthy colony | Single-line clean output when nothing is wrong |
| Integration with medic | Moderate 1: Redundant detection | Extend scanner, don't duplicate it |
| Phase progress inference | Critical 5: Resetting completed phases | Use disk evidence as floor, never go below it |

---

## Interaction Map with Existing Systems

| Existing System | How `recover` Should Interact | Risk |
|----------------|-------------------------------|------|
| `aether medic` | Extend scanner, share `ScannerResult` type, share repair infrastructure | Low if reusing, high if duplicating |
| `aether plan --force` | Do not overlap -- recovery handles state, plan --force handles artifacts | Medium if boundary unclear |
| `aether status` | Recovery should fix the things that status reports as broken | Low -- status is read-only |
| `aether continue` | Recovery must ensure continue.json files are consistent after repair | High -- continue has complex state expectations |
| `aether build` | Recovery must not interfere with an active build (lock check) | Critical if lock check is missing |
| File locking (`pkg/storage`) | Recovery must acquire locks before any mutation | Critical for data integrity |
| Backup rotation | Share rotation pool with medic (`cleanupOldBackups`) | Low if sharing, medium if separate |
| Trace logging | Reuse `logRepairToTrace`, use distinct topic prefix | Low |

---

## Prevention Priority Order

1. **Lock acquisition before mutation** (Critical 4) -- Without this, everything else is unsafe.
2. **Staleness thresholds for detection** (Critical 1) -- Without this, recovery damages active work.
3. **Transactional recovery units** (Critical 3) -- Without this, partial recovery creates new problems.
4. **Artifact timestamp guards** (Critical 2) -- The `.fallback-marker` pattern is proven. Reuse it.
5. **Disk evidence as floor for progress** (Critical 5) -- Never infer backwards from missing data.
6. **Clear boundary with plan --force** (Moderate 3) -- Overlapping cleanup is a design smell.
7. **Clean output for healthy colonies** (Moderate 2) -- First impression matters.

---

## Sources

- `cmd/codex_plan.go` lines 1696-1730 -- `clearFallbackPlanningArtifacts` pattern
- `cmd/medic_repair.go` -- `performRepairs`, backup, rotation, per-category repair dispatch
- `cmd/medic_scanner.go` -- `performHealthScan`, `ScannerResult`, file checker infrastructure
- `cmd/medic_cmd.go` -- CLI flags (--fix, --force), post-repair re-scan, visual rendering
- `cmd/state_repair.go` -- `repairMissingPlanFromArtifacts`, `inferRecoveredPhaseProgress`, disk evidence floor
- `cmd/recovery_snapshot.go` -- `loadActiveRecoveryGuidance`, `abandonedBuildThreshold`, `nextCommandFromState`
- `cmd/state_load.go` -- `loadActiveColonyState`, compatibility repair, legacy normalization
- `cmd/immune.go` -- scar tracking, error diagnosis
- `pkg/storage/lock.go` -- `FileLocker`, platform-specific locking
- `pkg/storage/backup.go` -- backup creation, rotation
- `pkg/storage/storage.go` -- atomic write, JSON validation, lock-protected read/write
