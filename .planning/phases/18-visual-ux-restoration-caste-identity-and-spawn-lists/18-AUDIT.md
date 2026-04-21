# Phase 18 Visual Output Path Audit

## Commands that set AETHER_OUTPUT_MODE=visual

All 49 slash commands in `.aether/commands/*.yaml` set `AETHER_OUTPUT_MODE=visual` before invoking the runtime CLI. This includes the four commands of interest:

- `.aether/commands/build.yaml`
- `.aether/commands/colonize.yaml`
- `.aether/commands/plan.yaml`
- `.aether/commands/run.yaml`

## Commands that call emitVisualProgress during worker execution

### Build (`cmd/codex_build.go`)
- `emitVisualProgress(renderBuildDispatchPreview(...))` — called before dispatch
- `dispatchCodexBuildWorkers` -> `dispatchBatchByWaveWithVisuals` -> `emitCodexDispatchWaveProgress` for each wave
- `runtimeVisualDispatchObserver` -> `emitCodexDispatchWorkerStarted` / `emitCodexDispatchWorkerRunning` / `emitCodexDispatchWorkerFinished`

### Colonize (`cmd/codex_colonize.go`)
- `emitVisualProgress(renderColonizeDispatchPreview(...))` — called before dispatch
- `dispatchRealSurveyors` -> `dispatchBatchByWaveWithVisuals` -> `emitCodexDispatchWaveProgress`
- `runtimeVisualDispatchObserver` -> `emitCodexDispatchWorkerStarted` / `emitCodexDispatchWorkerRunning` / `emitCodexDispatchWorkerFinished`

### Plan (`cmd/codex_plan.go`)
- `emitVisualProgress(renderPlanDispatchPreview(...))` — called before dispatch
- `dispatchRealPlanningWorkers` -> `dispatchBatchByWaveWithVisuals` -> `emitCodexDispatchWaveProgress`
- `runtimeVisualDispatchObserver` -> `emitCodexDispatchWorkerStarted` / `emitCodexDispatchWorkerRunning` / `emitCodexDispatchWorkerFinished`

### Run / Autopilot (`cmd/compatibility_cmds.go`)
- `runCompatibilityAutopilot` calls `runCodexBuild` and `runCodexContinue`
- `runCodexBuild` emits the same visual progress as the standalone build command
- `runCodexContinue` does NOT emit per-worker visual progress during its execution (it is a single-threaded verification pass)
- The run command's own visual output (`renderRunCompatibilityVisual`) shows a summary of steps taken, not live worker identity

## Commands that spawn workers but do NOT show caste identity

**None.** All three worker-spawning commands (build, colonize, plan) use `dispatchBatchByWaveWithVisuals`, which emits wave progress and per-worker start/running/finish events with caste identity.

However, the **run** command's summary visual (`renderRunCompatibilityVisual`) does NOT show caste identity for individual workers — it only lists step commands like `aether build 1` and `aether continue`. This is a gap: the user does not see the live worker list during autopilot execution.

## Summary

| Command | Wave Progress | Per-Worker Start | Per-Worker Running | Per-Worker Finish | Caste Identity |
|---------|--------------|------------------|--------------------|-------------------|----------------|
| build   | Yes          | Yes              | Yes                | Yes               | Yes            |
| colonize| Yes          | Yes              | Yes                | Yes               | Yes            |
| plan    | Yes          | Yes              | Yes                | Yes               | Yes            |
| run     | No (summary only) | No          | No                 | No                | No             |
