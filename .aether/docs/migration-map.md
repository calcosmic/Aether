# Follow-up Migration Map

> Migration plans for three deferred capabilities from v1.16.
> Each milestone is detailed enough for `/gsd-plan-phase` to proceed without additional research or discuss cycles.

## Ordering

Milestones are ordered strictly sequential per D-02 and D-03:

1. **Milestone A: Oracle/RALF Confidence Iteration** -- proves the TS host can handle complex iterative flows
2. **Milestone B: Swarm Visibility** -- proves the TS host can handle display/polling flows
3. **Milestone C: Build/Continue Parity** -- brings all remaining colony flows under TS host orchestration

## References

- **Boundary contract:** [.aether/references/contracts/runtime-boundary-contract.md](../references/contracts/runtime-boundary-contract.md) -- Go/TS ownership boundary
- **TS host lifecycle pattern (Phase 109):** [.aether/ts-host/src/lifecycle.ts](../ts-host/src/lifecycle.ts) -- established `callGoJSON` / manifest / finalizer pattern

---

## Milestone A: Oracle/RALF Confidence Iteration

### Scope

**What gets migrated to TS host:**

- Outer iteration loop orchestration (when to start, stop, which iteration to run)
- Worker dispatch per iteration (same dispatch pattern as build workers)
- Stop condition evaluation (confidence target met, max iterations reached, manual stop)

**What stays in Go:**

- Oracle state management (`oracle-state.json`, `oracle-plan.json`, research plan files)
- Question selection logic and iteration phase transitions (survey/analysis/synthesis)
- Confidence calculation and targeting (quick=60%, balanced=85%, deep=95%, exhaustive=99%)
- Workspace management (create/archive directories, manage `.aether/data/oracle/` files)
- Retry with narrowing logic (failed attempts get a narrower recovery prompt)

The TS host drives the outer RALF loop; Go owns the reasoning and state. Per D-05: migration does NOT move loop logic to TS. TS host orchestrates timing and worker dispatch; Go handles all question selection, confidence calculation, and state writes.

### Phases

| Phase | Name | Scope | Dependencies |
|-------|------|-------|--------------|
| A-1 | Go Oracle Iteration Commands | Medium -- Add `oracle-iterate --plan-only` (returns current state, next question, iteration spec as JSON manifest) and `oracle-iterate-finalize --completion-file` (commits iteration results, updates oracle state) | None |
| A-2 | TS Host Oracle Lifecycle | Medium-high -- Add `runOracleLifecycle()` to `lifecycle.ts` that drives the RALF loop: calls `oracle-iterate --plan-only`, dispatches worker per manifest, collects results, calls `oracle-iterate-finalize`. Handles max-iterations, confidence targets, and stop conditions from manifest | A-1 |
| A-3 | Oracle Integration Tests | Medium -- End-to-end test that runs Oracle through TS host with a known topic, verifies confidence iteration progresses, state writes go through Go finalizers, no direct `.aether/data/oracle/` writes from TS | A-2 |

### Requirements

| ID | Description | Phase | Success Criteria |
|----|-------------|-------|------------------|
| ORA-01 | Go command `oracle-iterate --plan-only` returns JSON manifest with iteration state, next question, confidence target, max iterations | A-1 | 1. Command returns valid JSON with `iteration_state`, `next_question`, `confidence_target`, `max_iterations` fields. 2. No state mutation occurs when `--plan-only` is used. |
| ORA-02 | Go command `oracle-iterate-finalize --completion-file` commits iteration results and updates oracle state atomically | A-1 | 1. State file in `.aether/data/oracle/` updated with new iteration count and confidence. 2. Plan file updated with findings. 3. Provenance validation passes before write. |
| ORA-03 | TS host `runOracleLifecycle()` drives the iteration loop using Go manifests and finalizers | A-2 | 1. Loop calls `oracle-iterate --plan-only` for each iteration. 2. Loop calls `oracle-iterate-finalize` after worker completes. 3. Loop terminates on stop conditions. |
| ORA-04 | TS host dispatches Oracle workers per manifest (same dispatch pattern as build workers) | A-2 | 1. Worker dispatched using `dispatchWorkers()` from `worker-dispatch.ts`. 2. Spawn-log/spawn-complete recorded via Go CLI. 3. Worker type matches manifest specification. |
| ORA-05 | TS host respects stop conditions from manifest (confidence target met, max iterations reached, manual stop) | A-2 | 1. Loop exits when `confidence_target` met. 2. Loop exits when `max_iterations` reached. 3. Loop exits on manual stop signal. |
| ORA-06 | No TS host direct writes to `.aether/data/oracle/` -- all state through Go finalizers | A-2 | 1. Boundary enforcement test (`assertNoDirectDataWrites`) passes. 2. No TS imports or writes to oracle data paths. |
| ORA-07 | Oracle RALF behavior preserved exactly (confidence targets, question selection, phase transitions) | A-3 | 1. Existing Oracle Go tests pass unchanged. 2. E2E test through TS host produces equivalent results to direct Go invocation. |
| ORA-08 | End-to-end integration test proves Oracle lifecycle runs through TS host | A-3 | 1. Test runs full Oracle lifecycle with known topic. 2. Confidence iteration progresses across iterations. 3. State writes go through Go finalizers only. |

### Boundary Contract Compliance

Oracle workspace files (state, plan, research plan, artifacts) in `.aether/data/oracle/` are **Go-owned**. The TS host MUST NOT write these directly.

All oracle state mutation goes through `oracle-iterate-finalize`, which:
- Validates provenance before any write
- Commits iteration results atomically
- Updates oracle state file and plan file together

Reference: boundary contract [anti-pattern #1](../references/contracts/runtime-boundary-contract.md) -- "No TS Direct State Writes."

### Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Oracle loop has internal state dependencies between iterations (question selection depends on prior iteration findings) | Medium | High | `oracle-iterate --plan-only` returns the full context needed for each iteration, including accumulated findings. The TS host does not need to maintain state between iterations -- Go provides it each time. |
| Background controller mode (detached Oracle with PID tracking) requires different lifecycle management | Medium | Low | Map background mode as a future enhancement. Initial migration covers foreground interactive mode only. Background controller adds PID tracking and detached process management, which is orthogonal to the TS host orchestration pattern. |
| Oracle iteration state is too large to pass through JSON manifest | Low | Medium | Go returns only the fields needed for orchestration (next question, confidence, stop conditions), not the full workspace. Full state stays in Go's workspace files. |
