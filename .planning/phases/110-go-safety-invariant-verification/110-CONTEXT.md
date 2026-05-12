# Phase 110: Go Safety Invariant Verification - Context

**Gathered:** 2026-05-12
**Status:** Ready for planning

<domain>
## Phase Boundary

Prove Go remains the sole authority for state mutation, finalizers, locking, install/update/publish, and verification contracts — even with the TypeScript orchestration host (from Phase 109) present and active. This is a verification-only phase: write tests that confirm existing Go safety mechanisms hold, without adding new runtime guards or changing Go behavior.

</domain>

<decisions>
## Implementation Decisions

### Validation Strictness
- **D-01:** Verify existing code only — no new runtime guards or watchdog systems. Trust Go's existing finalizers, atomic writes, and locking; write tests that prove they work correctly.
- **D-02:** Focus on manifest validation — verify each finalizer (plan, build, continue) rejects malformed manifests: missing phase number, invalid version, no provenance timestamp, empty worker list.
- **D-03:** Test common corruption cases only — no adversarial payloads (no deeply nested JSON, no Unicode injection, no future-version manifests). Cover the integration bugs most likely to occur when TS sends data to Go.
- **D-04:** Per-finalizer test sets — plan, build, and continue finalizers each get their own dedicated validation tests. Clean separation makes it easy to identify which finalizer has a gap.

### Concurrency Testing
- **D-05:** Normal flow only — test the standard plan→build→continue lifecycle driven through the TS host and verify state is correct after each step. No stress scenarios (no concurrent Go+TS finalizer calls, no concurrent state writes).
- **D-06:** Reuse Phase 108 golden tests — run the golden workflow tests against TS host-driven execution. If the same state transitions happen, Go's invariants hold. Minimal new test code.
- **D-07:** Smoke test install/update/publish purity — verify `aether install`, `aether update`, `aether publish` have zero code path overlap with the TS host. Simple grep for TS host imports + test that commands work normally.

### Test Organization
- **D-08:** One new dedicated file `cmd/safety_invariant_test.go` covering all 6 success criteria. Easy to find, clear purpose, single place to check safety coverage.
- **D-09:** Per-criterion test functions mapping to each success criterion: TestStateMutationSoleAuthority, TestFinalizerProvenance, TestLockingUnchanged, TestInstallPureGo, TestVerificationContractsPass, TestPlanOnlyUnchanged.

### Claude's Discretion
- Exact test implementation patterns (table-driven, sequential, etc.)
- Which golden tests to reuse and how to adapt them for TS host execution
- How to structure the install/update/publish purity check (grep vs Go AST analysis vs import check)
- Error message expectations for rejected manifests
- Whether to use existing test helpers (setupBuildFlowTest, createTestColonyState) or write new ones

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Boundary Contract (from Phase 106)
- `.aether/references/contracts/runtime-boundary-contract.md` — Defines Go/TS/Assets/Bash ownership. The safety invariants being verified come directly from this contract.

### Go Finalizers (primary test targets)
- `cmd/codex_plan_finalize.go` — Plan finalizer with provenance validation
- `cmd/codex_build_finalize.go` — Build finalizer with provenance validation
- `cmd/codex_continue_finalize.go` — Continue finalizer with verification gates
- `cmd/codex_colonize_finalize.go` — Colonize finalizer (for completeness)

### State Mutation Authority
- `cmd/state_cmds.go` — `state-mutate` command with atomic COLONY_STATE.json operations and guard conditions
- `cmd/state_load.go` — State file loading
- `pkg/storage/storage.go` — Store with atomic write operations
- `pkg/storage/lock.go` — Cross-process file locking
- `pkg/storage/lock_unix.go` — Unix-specific locking implementation

### TypeScript Host (the integration partner)
- `.aether/ts-host/src/lifecycle.ts` — TS host lifecycle orchestrator
- `.aether/ts-host/src/go-bridge.ts` — Enforces boundary rules, prevents direct data writes
- `.aether/ts-host/src/boundary-reference.ts` — Defines Go-owned paths including .aether/data/

### Existing Test Infrastructure
- `cmd/finality_parity_test.go` — Existing parity tests for finalizers (pattern reference)
- `cmd/boundary_contract_test.go` — Existing boundary contract tests (pattern reference)
- `cmd/golden_workflow_test.go` — Golden tests from Phase 108 (reuse for TS host verification)
- `cmd/build_flow_cmds_test.go` — `setupBuildFlowTest` and `createTestColonyState` helpers

### Install/Update/Publish Commands
- `cmd/contracts/update.md` — Update contract
- `cmd/contracts/publish.md` — Publish contract
- `cmd/codex_dispatch_contract.go` — Dispatch contract structures

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **setupBuildFlowTest(t)** — Creates temp directory with store, sets AETHER_ROOT. Every safety test should use this for isolation.
- **createTestColonyState(t, dataDir, state)** — Creates minimal COLONY_STATE.json for test setup.
- **parseEnvelope(t, output)** — JSON output parsing helper for verifying Go responses.
- **saveGlobals(t) / resetRootCmd(t)** — Test isolation helpers.
- **Golden workflow tests** — Phase 108 already tests the full lifecycle. Reuse these by running them with TS host as the driver.
- **Boundary contract tests** — Phase 106 already tests the boundary. Extend the pattern, not the file.

### Established Patterns
- Finalizers validate manifest structure before committing state — provenance fields, version, phase number all checked.
- `store.AtomicWrite()` handles all state file writes with file locking.
- TS host already respects boundaries: no direct `.aether/data/` writes, all mutations through Go finalizers.
- Tests use table-driven patterns for multiple scenarios (e.g., multiple manifest corruption cases).

### Integration Points
- Safety tests read Go manifest output from `aether plan --plan-only` and `aether build --plan-only`
- Safety tests call finalizer commands directly with corrupted manifests to verify rejection
- Safety tests run install/update/publish commands to verify no TS involvement
- Golden tests can be rerun with TS host active to verify unchanged behavior

</code_context>

<specifics>
## Specific Ideas

- The test file name should clearly signal purpose: `cmd/safety_invariant_test.go`
- Each success criterion from the roadmap maps to exactly one test function for traceability
- Manifest corruption test cases should be realistic TS→Go integration bugs (missing fields, wrong types), not adversarial attacks
- The install/update/publish purity check should be quick — a grep for TS host package imports in Go install/update/publish files is sufficient
- If Phase 108 golden tests already pass with TS host active (from Phase 109 verification), this phase can reference that as evidence and focus on the remaining criteria (provenance validation, install purity)

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 110-Go Safety Invariant Verification*
*Context gathered: 2026-05-12*
