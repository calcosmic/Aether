# Phase 51: Recovery Verification - Context

**Gathered:** 2026-04-25
**Status:** Ready for planning

<domain>
## Phase Boundary

E2E tests proving all recovery paths work correctly: 7 individual stuck-state recoveries, a compound scenario with multiple simultaneous issues, and a healthy-colony no-false-positives test. Test-only phase — no production code changes except test files.

</domain>

<decisions>
## Implementation Decisions

### Test Structure
- **D-01:** One test function per scenario following existing `e2e_regression_test.go` pattern — `TestE2ERecoveryMissingBuildPacket`, `TestE2ERecoveryStaleSpawned`, etc. plus `TestE2ERecoveryCompoundState` and `TestE2ERecoveryHealthyColony`. Not table-driven — each scenario has unique setup that would be awkward to parameterize.
- **D-02:** Tests go in a new file `cmd/e2e_recovery_test.go` to keep recovery tests isolated from the existing E2E regression suite.

### Seeding Strategy
- **D-03:** Helper functions in the test file that create specific broken states in temp directories. Each helper writes hand-crafted JSON/files that trigger one scanner. Examples: `seedMissingPacketState(t, dir)`, `seedStaleSpawnedState(t, dir)`, etc.
- **D-04:** Compound test uses a combined seeder that applies multiple seeds to the same temp dir.

### Verification Depth
- **D-05:** Post-recovery assertions check: (1) exit code is 0, (2) re-scan returns empty issue list (no remaining problems), (3) key state file content matches expected post-repair values. This goes beyond "didn't crash" to prove the state is actually clean.
- **D-06:** For repair tests (--apply), also verify the backup was created and the specific state mutation occurred (e.g., phase status changed from EXECUTING to READY).

### Compound Scenario Scope
- **D-07:** Two compound scenarios: (1) all 5 safe states simultaneously, (2) both destructive states simultaneously. This mirrors real-world patterns where safe issues cluster together and destructive issues are rarer but also cluster.
- **D-08:** The 7-individual-state tests cover scan-only (--apply=false) behavior. The compound tests cover --apply behavior to prove the full repair pipeline works end-to-end.

### Healthy Colony Test
- **D-09:** `TestE2ERecoveryHealthyColony` creates a fully initialized, healthy colony state with no broken files and asserts: (1) exit code 0, (2) JSON output shows empty issues array, (3) text output contains "No issues detected" or equivalent.

### Claude's Discretion
- Exact helper function signatures and internal structure
- Whether to reuse existing test helpers from `recover_test.go` or write fresh E2E-specific ones
- Order of test execution
- Exact assertion messages on failure

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Recovery Implementation
- `cmd/recover.go` — Command entry point, --apply wiring, scan→repair→re-scan flow
- `cmd/recover_scanner.go` — 7 stuck-state detection functions
- `cmd/recover_repair.go` — 7 repair functions, backup, rollback, confirmation orchestrator
- `cmd/recover_visuals.go` — Output rendering (text + JSON), repair log, fixable hints
- `cmd/recover_test.go` — Existing 61 unit tests (reference for fixtures and patterns)

### E2E Test Patterns
- `cmd/e2e_regression_test.go` — Existing E2E test pattern: `saveGlobals`, `resetRootCmd`, `rootCmd.SetArgs`, temp dirs

### Supporting Infrastructure
- `cmd/medic_repair.go` — `createBackup`, `atomicWriteFile`, backup infrastructure reused by recovery
- `pkg/colony/state_machine.go` — `colony.Transition` used for state changes during repair

### Requirements
- `.planning/REQUIREMENTS.md` — TEST-01, TEST-02, TEST-03 requirements
- `.planning/ROADMAP.md` §Phase 51 — Success criteria (3 items)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **E2E test helpers:** `saveGlobals(t)`, `resetRootCmd(t)`, `createMockSourceCheckout(t, version)` in `e2e_regression_test.go` — same pattern applies here
- **Unit test fixtures:** `recover_test.go` has 61 tests with fixture setup patterns (writing COLONY_STATE.json, spawned.json, manifest files to temp dirs) — E2E tests will need similar but exercised through the full command path
- **Medic backup helpers:** `createBackup`, `restoreFromBackup`, `cleanupOldBackups` — used in repair pipeline, E2E tests verify these work end-to-end

### Established Patterns
- **E2E test flow:** `saveGlobals → resetRootCmd → temp dir → rootCmd.SetArgs → Execute → assert stdout/stderr/exit code/files`
- **Recovery state fixtures:** Tests write JSON directly to temp `.aether/data/` directories with specific field values that trigger each scanner
- **Exit code testing:** `recover.go` sets `os.Exit(1)` via `exitWithCode` helper — E2E tests need to capture this

### Integration Points
- Tests call `rootCmd.SetArgs([]string{"recover"})` and `rootCmd.SetArgs([]string{"recover", "--apply"})` for scan-only and repair paths
- JSON output tested via `--json` flag: `rootCmd.SetArgs([]string{"recover", "--json"})`
- Temp dirs need `.aether/data/` structure with realistic colony state files

</code_context>

<specifics>
## Specific Ideas

- The phase 50 code review identified 3 critical bugs (rollback undoing successful repairs, dirty worktree unnecessary state write, brittle JSON rendering). If those weren't fixed in phase 50 execution, the E2E tests should expose them — especially the compound test which would catch rollback correctness.
- Each of the 7 scanner tests should test both scan-only (report the issue) and scan+repair (--apply fixes it) paths.
- The healthy colony test is the most important — false positives would erode user trust immediately.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 51-recovery-verification*
*Context gathered: 2026-04-25*
