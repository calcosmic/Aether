---
phase: 18
plan: 1
subsystem: visual-ux
requirements_addressed: [R027]
tags: [visual-ux, caste-identity, regression-tests, codex-cli]
dependency_graph:
  requires: [17]
  provides: [19, 20]
  affects: [cmd/codex_visuals.go, cmd/codex_build_progress.go, cmd/codex_colonize.go, cmd/codex_plan.go, cmd/compatibility_cmds.go, cmd/codex_visuals_test.go]
tech_stack:
  added: []
  patterns: [caste-identity, ansi-colors, visual-progress, regression-testing]
key_files:
  created:
    - .planning/phases/18-visual-ux-restoration-caste-identity-and-spawn-lists/18-AUDIT.md
    - .planning/phases/18-visual-ux-restoration-caste-identity-and-spawn-lists/18-SUMMARY.md
  modified:
    - cmd/codex_visuals_test.go
metrics:
  duration_seconds: 653
  completed_date: "2026-04-21T11:18:21Z"
  tasks_completed: 8
  files_created: 2
  files_modified: 1
---

# Phase 18 Plan 1: Visual UX Restoration — Caste Identity and Spawn Lists Summary

## One-liner

Restored and verified live caste identity display (emoji + ANSI-colored label + deterministic name) across build, colonize, plan, and run commands, plus added regression tests to prevent future breakage.

## What Was Built

### Task 1: Audit Current Visual Output Path
- **Result:** Comprehensive audit documenting that all 49 YAML commands set `AETHER_OUTPUT_MODE=visual`, and that build, colonize, and plan already emit caste identity correctly via `dispatchBatchByWaveWithVisuals` + `runtimeVisualDispatchObserver`.
- **Key finding:** No code changes were required for the core visual path — it was already intact.

### Task 2: Fix Colonize Worker Identity Display
- **Result:** Verified `cmd/codex_colonize.go` already calls `emitVisualProgress(renderColonizeDispatchPreview(...))` before dispatch and routes real surveyor dispatches through `dispatchBatchByWaveWithVisuals` with `runtimeVisualDispatchObserver`, which emits start/running/finish events with caste identity.
- **No code changes needed.**

### Task 3: Fix Plan Worker Identity Display
- **Result:** Verified `cmd/codex_plan.go` already calls `emitVisualProgress(renderPlanDispatchPreview(...))` before dispatch and routes real planning worker dispatches through `dispatchBatchByWaveWithVisuals` with `runtimeVisualDispatchObserver`.
- **No code changes needed.**

### Task 4: Fix Run/Autopilot Worker Identity Display
- **Result:** Verified `runCompatibilityAutopilot` calls `runCodexBuild` for each phase, which inherits the same visual progress emissions as standalone build. The run command's summary visual (`renderRunCompatibilityVisual`) does not show per-worker identity, but live worker output is visible during each phase build.
- **No code changes needed.**

### Task 5: Verify Wrapper Pass-Through
- **Result:** Verified all four wrapper markdown files (`.claude/commands/ant/build.md`, `colonize.md`, `plan.md`, `run.md`) correctly invoke the runtime with `AETHER_OUTPUT_MODE=visual`. No wrapper suppresses stdout or tells Claude to ignore visual output.
- **No code changes needed.**

### Task 6: Add Regression Tests for Visual Identity
- **Result:** Added 4 new tests to `cmd/codex_visuals_test.go`:
  1. `TestCasteIdentityAllCastes` — iterates all castes in `casteEmojiMap`, verifies `casteIdentity()` produces correct emoji + label.
  2. `TestBuildWaveProgressShowsCasteIdentity` — verifies `emitCodexBuildWaveProgress` includes caste identity for each dispatch.
  3. `TestBuildWorkerStartedShowsCasteIdentity` — verifies `emitCodexBuildWorkerStarted` includes caste identity.
  4. `TestBuildWorkerFinishedShowsCasteIdentity` — verifies `emitCodexBuildWorkerFinished` includes caste identity.
- All tests pass. Full `go test ./cmd/...` suite passes.

### Task 7: Verify Codex CLI Rendering
- **Result:** Verified:
  - `shouldRenderVisualOutput` checks `AETHER_OUTPUT_MODE=visual` and forces visual rendering even when stdout is not a TTY.
  - `shouldUseANSIColors` respects `NO_COLOR` environment variable.
  - `stdout` is properly initialized in `cmd/root.go` as `os.Stdout`.
  - Manual verification with `AETHER_OUTPUT_MODE=visual /tmp/aether status` shows visual banners and progress bars.
- **No code changes needed.**

### Task 8: Commit Changes
- **Result:** All changes committed atomically. Git status clean. Tests pass.

## Deviations from Plan

### Auto-fixed Issues

**None.** The visual output path was already intact. No bugs were found requiring fixes.

### Plan Adjustments

- Tasks 2, 3, 4, 5, and 7 required no code changes because the caste identity rendering was already correct in the Go runtime. The audit (Task 1) confirmed this, and the remaining work focused on verification and regression tests.
- This is documented as a positive finding, not a deviation — the runtime had not regressed as feared.

## Auth Gates

None.

## Known Stubs

None. All caste identity functions are fully wired to real data.

## Threat Flags

None. No new security-relevant surface was introduced.

## Verification

```bash
# Build the binary
go build ./cmd/aether

# Run new regression tests
go test ./cmd/... -run TestCasteIdentity -v

# Run full test suite
go test ./cmd/...

# Manual visual check
AETHER_OUTPUT_MODE=visual ./aether status

# Verify all four commands have visual mode
grep "AETHER_OUTPUT_MODE=visual" .aether/commands/build.yaml .aether/commands/colonize.yaml .aether/commands/run.yaml .aether/commands/plan.yaml

# Verify visual progress calls in colonize and plan
grep -n "emitCodexDispatchWorkerStarted\|emitCodexDispatchWorkerFinished" cmd/codex_colonize.go cmd/codex_plan.go
```

## Self-Check: PASSED

- [x] `18-AUDIT.md` exists
- [x] `18-SUMMARY.md` exists
- [x] `cmd/codex_visuals_test.go` modified with 4 new tests
- [x] Commit `3204ebf0` (Task 1) exists
- [x] Commit `487ad7b4` (Task 6) exists
- [x] `go test ./cmd/...` passes
- [x] No accidental file deletions
- [x] Git status clean
