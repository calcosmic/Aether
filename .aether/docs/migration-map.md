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

---

## Milestone B: Swarm Visibility

### Scope

**What gets migrated to TS host:**

- Invocation of Go swarm rendering commands (`swarm-display-render`, `swarm-display-inline`, `swarm-display-text`) with `AETHER_OUTPUT_MODE=json`
- Presentation of structured JSON output to the user's platform (Claude Code, OpenCode, Codex)
- Live watch polling via `runLiveWatch()` at configurable interval

**What stays in Go:**

- All colony state data (reads from `COLONY_STATE.json`)
- Rendering logic (tree formatting, ANSI color codes, inline sections, text wrapping)
- ANSI formatting and visual output generation

Per D-06: Go owns the data and rendering. TS host owns the presentation layer only -- it calls Go commands, receives structured JSON, and formats output for the user's platform. No rendering logic moves to TS.

### Phases

| Phase | Name | Scope | Dependencies |
|-------|------|-------|--------------|
| B-1 | TS Host Swarm Display | Low-medium -- Add `runSwarmDisplay()` to TS host that calls existing Go `swarm-display-render --format json` via `callGoJSON()`, formats the result, and outputs to stderr. Add `runLiveWatch()` that polls at configurable interval | Milestone A complete (D-03) |
| B-2 | Swarm Integration Tests | Low -- Test that swarm display output renders correctly through TS host, all three formats (tree, JSON, flat) work, live watch polls and updates | B-1 |

### Requirements

| ID | Description | Phase | Success Criteria |
|----|-------------|-------|------------------|
| SWA-01 | TS host calls Go `swarm-display-render --format json` via `callGoJSON()` and presents structured output | B-1 | 1. `runSwarmDisplay()` calls Go command with `AETHER_OUTPUT_MODE=json`. 2. Structured output is formatted and written to stderr. 3. Error from Go is surfaced to the user. |
| SWA-02 | TS host supports all three swarm display formats (tree, JSON, flat) by passing format flag to Go | B-1 | 1. `--format tree` produces tree output. 2. `--format json` produces JSON output. 3. `--format flat` produces flat output. All via Go rendering, not TS formatting. |
| SWA-03 | TS host `runLiveWatch()` polls `swarm-display-render` at configurable interval and refreshes output | B-1 | 1. Polling loop calls Go command at default 2-second interval. 2. Interval is configurable. 3. Output refreshes only when data changes (ANSI clear only on change). |
| SWA-04 | No TS-side rendering logic -- all tree/text/ANSI rendering done by Go, TS host only presents Go output | B-1 | 1. No TS code contains ANSI escape sequences for rendering. 2. No TS code draws tree structures. 3. All visual formatting is Go-side. |
| SWA-05 | Integration tests prove swarm display works through TS host in all formats | B-2 | 1. Test invokes `runSwarmDisplay()` with each format. 2. Output is non-empty and structurally valid. 3. Live watch test completes at least 2 poll cycles. |

### Boundary Contract Compliance

Swarm display is **read-only** -- Go reads `COLONY_STATE.json` and outputs formatted display. No state mutation is involved. This is the simplest migration from a boundary perspective.

Per boundary contract [rule #2](../references/contracts/runtime-boundary-contract.md): TS host uses `AETHER_OUTPUT_MODE=json` for all programmatic consumption. Visual output from Go is for humans; the TS host never parses ANSI/visual output to extract state.

### Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Go `swarm-display-*` commands may not support `AETHER_OUTPUT_MODE=json` (they may write directly to stdout instead of using `outputOK()`) | Medium | Medium | Verify with `AETHER_OUTPUT_MODE=json aether swarm-display-render --format json` before planning. If unsupported, Phase B-1 adds JSON output mode to Go commands (small change -- route through `outputOK()` instead of direct `fmt.Fprintln`). |
| Live watch polling may cause terminal flickering with ANSI clear sequences | Low | Low | TS host uses ANSI clear sequences only when output actually changes. If data is identical between polls, skip the redraw. Configurable interval defaults to 2 seconds (matching existing `aether watch` behavior). |

---

## Milestone C: Build/Continue Parity

### Scope

**What gets migrated to TS host:**

- `colonize` lifecycle orchestration (4 surveyor dispatches via manifest)
- `seal` lifecycle orchestration (ceremony with blocker check)
- Oracle lifecycle (if not covered in Milestone A -- see note below)
- Watch/swarm flows (if not covered in Milestone B -- see note below)

**What stays in Go:**

- All state mutation (COLONY_STATE.json, pheromones, reviews, archives)
- Surveyor result collection and validation
- Seal ceremony logic (instinct promotion, pheromone expiration, review archival)
- Verification gates and blocking

Plan/build/continue already work through TS host (Phase 109). This milestone covers the **remaining** flows: colonize, seal, and any oracle/watch flows not already handled by Milestones A and B.

Per D-07: all 6 colony flows (plan, build, continue, colonize, seal, oracle) use TS host orchestration after this milestone completes.

### Phases

| Phase | Name | Scope | Dependencies |
|-------|------|-------|--------------|
| C-1 | Colonize TS Host Integration | Medium -- Add `runColonizeLifecycle()` to TS host. Colonize already has `colonize --plan-only` (returns `codexColonizeManifest` with 4 surveyor dispatches) and `colonize-finalize` command. TS host calls plan-only, dispatches surveyor workers, calls finalizer | Milestones A and B complete (D-03) |
| C-2 | Seal TS Host Integration | Medium -- Add `runSealLifecycle()` to TS host. Seal needs a new `seal --plan-only` (returns blocker list, ceremony steps) and `seal-finalize` command. TS host calls plan-only, checks for blockers, dispatches review workers if needed, calls finalizer | C-1 |
| C-3 | End-to-End Parity Verification | Medium -- Integration test that runs all 6 flows (plan, build, continue, colonize, seal, oracle) through TS host. Verify no direct `.aether/data/` writes, all state through Go finalizers | C-2 |

### Requirements

| ID | Description | Phase | Success Criteria |
|----|-------------|-------|------------------|
| PAR-01 | TS host `runColonizeLifecycle()` calls `colonize --plan-only`, dispatches 4 surveyor workers, calls `colonize-finalize` | C-1 | 1. `colonize --plan-only` returns manifest with 4 surveyor dispatches. 2. TS host dispatches all 4 surveyors via `dispatchWorkers()`. 3. `colonize-finalize` called with completion file. |
| PAR-02 | TS host `runSealLifecycle()` calls `seal --plan-only`, checks blockers, dispatches review workers, calls `seal-finalize` | C-2 | 1. `seal --plan-only` returns blocker list and ceremony steps. 2. If blockers exist, TS host reports blocked status. 3. If no blockers, TS host dispatches review workers and calls `seal-finalize`. |
| PAR-03 | Seal blocker handling: TS host calls `seal --plan-only`, if blockers exist, reports blocked status to user (does not auto-force) | C-2 | 1. Blocker list extracted from plan-only manifest. 2. Blocked status reported to user via stderr. 3. TS host stops -- does not call `seal-finalize` when blocked. |
| PAR-04 | All 6 colony flows (plan, build, continue, colonize, seal, oracle) use TS host orchestration | C-3 | 1. Plan/build/continue work through existing `runLifecycle()`. 2. Colonize works through `runColonizeLifecycle()`. 3. Seal works through `runSealLifecycle()`. 4. Oracle works through `runOracleLifecycle()`. |
| PAR-05 | No direct `.aether/data/` writes from TS host for any flow -- all state mutation through Go finalizers | C-3 | 1. Boundary enforcement test passes for all 6 flows. 2. No TS code writes to `.aether/data/` directly. 3. All state changes trace to Go finalizer commands. |
| PAR-06 | Existing behavior preserved: colonize surveyors produce same results, seal ceremony produces same state changes | C-3 | 1. Colonize surveyor results match direct Go invocation. 2. Seal produces same COLONY_STATE.json changes. 3. No regression in existing Go tests. |
| PAR-07 | End-to-end parity test proves all 6 flows run through TS host with no regressions | C-3 | 1. Test runs all 6 flows sequentially through TS host. 2. COLONY_STATE.json transitions match expected sequence. 3. No direct data writes detected. |

### Boundary Contract Compliance

**Colonize:** Surveyor dispatches are specified by Go manifest (`colonize --plan-only` returns the `codexColonizeManifest`). The TS host dispatches only from this manifest -- it does not invent surveyors or change their order. Reference boundary contract [rule #3](../references/contracts/runtime-boundary-contract.md) -- "TS host never invents workers."

**Seal:** Seal state mutation is **safety-critical** -- it sets `StateCOMPLETED`, writes `CROWNED-ANTHILL.md`, and archives reviews. All mutation goes through `seal-finalize`, which validates provenance and commits atomically. The TS host must never call seal-finalize when blockers are reported (reference boundary contract [rule #3](../references/contracts/runtime-boundary-contract.md) -- "No Wrapper-Owned Recovery Menus").

### Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Seal blocker/recovery-menu case requires TS host to handle interactive decisions | Medium | Medium | TS host reports blocker status from `seal --plan-only` manifest and stops. User handles blockers manually (via Go CLI or other means), then re-runs seal through TS host. No recovery menu in TS -- reference anti-pattern #3. |
| Colonize surveyor dispatches may need different handling than build workers (surveyors explore, builders implement) | Low | Low | `dispatchWorkers()` already handles arbitrary dispatch manifests. Surveyors are just different worker types dispatched from the manifest -- no special handling needed. The dispatch contract is worker-type-agnostic. |
| Seal lacks existing `--plan-only` support (unlike colonize) | Medium | Medium | Phase C-2 adds `seal --plan-only` and `seal-finalize` Go commands. Go already has `completeSealRuntime()` -- the new commands expose the ceremony as plan-only/finalizer steps, following the same pattern as build/continue. |

---

## Dependency Graph

```
Milestone A (Oracle/RALF)
    |
    | Proves: TS host can handle complex iterative flows
    |          Go command pattern for state machines established
    v
Milestone B (Swarm Visibility)
    |
    | Proves: TS host can handle polling/display flows
    |          Go JSON output mode for programmatic consumption established
    v
Milestone C (Build/Continue Parity)
    |
    | Depends on: Oracle pattern (iterative), Swarm pattern (polling)
    | Brings all remaining flows under TS host
    v
Complete TS host coverage for all colony flows
```

### Milestone Summary

| Milestone | Phases | Requirements | Estimated Risk | Depends On |
|-----------|--------|-------------|----------------|------------|
| A: Oracle/RALF | 3 (A-1, A-2, A-3) | 8 (ORA-01 through ORA-08) | High | None |
| B: Swarm Visibility | 2 (B-1, B-2) | 5 (SWA-01 through SWA-05) | Low | Milestone A |
| C: Build/Continue Parity | 3 (C-1, C-2, C-3) | 7 (PAR-01 through PAR-07) | Medium | Milestones A and B |

**Parity depends on Oracle and swarm patterns being proven. Do not parallelize.** (D-03)

---

## Summary

| Metric | Value |
|--------|-------|
| Total milestones | 3 |
| Total phases | 8 (3 Oracle + 2 Swarm + 3 Parity) |
| Total requirements | 20 (8 ORA + 5 SWA + 7 PAR) |
| Ordering | A -> B -> C, strictly sequential (D-03) |
| Key constraint | Migration only, no new features (D-04) |
| Pattern | All migrations follow the plan-only/manifest/finalizer pattern from Phase 109 |
| Boundary contract | All milestones reference and comply with the runtime boundary contract |

### Migration Architecture

Every migration follows the same established pattern from Phase 109:

1. TS host calls Go command with `--plan-only` flag (no state mutation)
2. Go returns JSON manifest with dispatches, state, and next-step information
3. TS host processes manifest, dispatches workers via `dispatchWorkers()`
4. TS host builds completion file via `writeCompletionFile()`
5. TS host calls Go finalizer command with `--completion-file`
6. Go validates provenance, commits state atomically

The TS host never writes to `.aether/data/`, never invents workers, and never parses visual output. Go remains the sole authority for state mutation, verification, and safety enforcement.
