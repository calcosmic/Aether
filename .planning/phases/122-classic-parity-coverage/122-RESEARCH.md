# Phase 122: Classic Parity Coverage - Research

**Gathered:** 2026-05-14
**Status:** Verified — existing tests and docs already satisfy all requirements

## Findings

### PAR-01: Golden tests verify build ceremony matches v5.4 baseline
**Status:** COVERED
**Evidence:** `cmd/golden_workflow_test.go:212-271` — `TestGoldenBuildVisualOutput`
- Compares build output against `testdata/golden_build.txt`
- Validates ceremony markers: `BUILD DISPATCH`, `SPAWN PLAN`, `Builder`, `Watcher`
- Validates stage separators: `── Context ──`, `── Tasks ──`, `── Dispatch ──`, `── Verification`, `── Housekeeping ──`, `── Colony Complete ──`
- Passes: ✓

### PAR-02: Golden tests verify continue ceremony matches v5.4 baseline
**Status:** COVERED
**Evidence:** `cmd/golden_workflow_test.go:273-337` — `TestGoldenContinueVisualOutput`
- Compares continue output against `testdata/golden_continue.txt`
- Validates `Verification` stage present
- Passes: ✓

### PAR-03: Oracle confidence loop behavior is tested against v5.4 baseline
**Status:** COVERED
**Evidence:** Multiple Oracle tests in `cmd/session_flow_cmds_test.go`
- `TestOracle_AutoIncludedForDiscovery` — Oracle auto-inclusion for discovery tasks
- `TestOracleGuideCarriesBroadScopeTimeoutGuard` — timeout guardrails
- `TestOracleWrappersAndSkillCarryTimeoutGuard` — wrapper/skill timeout guards
- `TestOracleCompatibilityRunsAutonomousLoop` — autonomous loop behavior
- `TestOracleCompatibilityStopCommandWritesMarker` — stop marker persistence
- `TestOracleCompatibilityStopKillsControllerProcessTree` — process cleanup
- `TestOracleCompatibilityStopsAtIterationBoundary` — boundary stop behavior
- `TestOracleCompatibilityPersistsWorkerErrorReason` — error persistence
- All pass: ✓

### PAR-04: Swarm/dashboard visibility is tested against v5.4 baseline
**Status:** COVERED
**Evidence:** Multiple swarm tests in `cmd/codex_visuals_test.go` and `cmd/session_flow_cmds_test.go`
- `TestSwarmCompatibilityWatchReportsActiveWorkers` — active worker reporting
- `TestSwarmCompatibilityWatchPrefersCurrentRunWorkers` — current run preference
- `TestSwarmCompatibilityWatchShowsRecoveryGuidance` — recovery guidance display
- `TestSwarmDisplayRenderTree` — tree rendering
- `TestSwarmDisplayRenderJSON` — JSON rendering
- `TestSwarmDisplayRenderFlat` — flat rendering
- `TestSwarmDisplayInline` — inline rendering
- `TestSwarmDisplayText` — text rendering
- `TestSwarmDisplayCommandsRegistered` — command registration
- `TestSwarmWrapperCeremonyContract` — wrapper ceremony contract
- All pass: ✓

### PAR-05: Install/update flow is tested against v5.4 baseline
**Status:** COVERED
**Evidence:** Multiple install/update tests in `cmd/install_update_test.go` and `cmd/regression_test.go`
- `TestInstallBufferedOutputBreaksJSONUnderVisualEnv` — buffered output behavior
- `TestInstallVisualOutput` — visual mode output
- `TestInstallJsonModeStillProducesJson` — JSON mode output
- `TestUpdateDryRunVisualOutput` — dry run visual output
- `TestUpdateRoundTripLeavesGlobalAssetsOutOfRepo` — round-trip isolation
- `TestUpdateProgress` — progress tracking
- All pass: ✓

### PAR-06: State mutation through approved APIs is tested against v5.4 baseline
**Status:** COVERED
**Evidence:** `cmd/golden_workflow_test.go:352-454` — `TestGoldenStateMutations`
- plan → READY state
- build → BUILT state, CurrentPhase = 1
- continue → PhaseCompleted, CurrentPhase advances to 2
- Multi-phase colony returns to READY (not COMPLETED)
- Passes: ✓

### PAR-07: Any Classic behavior intentionally not restored is documented
**Status:** COVERED
**Evidence:** `.aether/references/classic-baseline.md`
- Documents all 16 Classic modules with classification: Restore in TS / Keep in Go / Obsolete
- Lists 3 obsolete modules: `state-sync.js`, `interactive-setup.js`, `nestmate-loader.js`
- Documents known limitations and workarounds
- Cross-references Phase 106 boundary contract and Phase 107 research

## Summary

All 7 PAR requirements are already satisfied by existing tests and documentation. No code changes required. Phase 122 is a verification-only phase.
