# Phase 111: Follow-up Migration Map - Research

**Researched:** 2026-05-12
**Domain:** Migration planning (documentation-only phase)
**Confidence:** HIGH

## Summary

This phase produces a written migration map for three deferred capabilities: Oracle/RALF confidence iteration, swarm visibility, and broader build/continue parity. Each item migrates existing Go behavior to TS host orchestration following the boundary contract established in Phase 106. The phase is documentation-only -- no new runtime code.

The Oracle RALF loop is the most complex migration target: it is a ~1000-line iterative loop in `cmd/oracle_loop.go` with confidence targets, question selection, multiple retry attempts, background controller support, and rich state management. The TS host would need to drive this loop externally (calling Go for each iteration's state and plan mutations) rather than reimplementing it.

Swarm visibility is simpler: Go already provides three rendering commands (`swarm-display-render`, `swarm-display-inline`, `swarm-display-text`) that output structured JSON. The TS host just needs to invoke these and render the output.

Parity is the broadest scope but most mechanical: it means adding TS host orchestration for `colonize`, `seal`, `oracle`, and `watch/swarm` flows, following the same plan-only/manifest/finalizer pattern already established by Phase 109 for plan/build/continue.

**Primary recommendation:** Produce a single combined migration map document with three milestone sections (Oracle, Swarm, Parity), each containing phases, requirements, success criteria, and dependency ordering.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Milestone-ready granularity -- each deferred item gets its own milestone with phases, requirements, and success criteria. The output is detailed enough that `/gsd-plan-phase` can proceed without another discuss cycle, but not so detailed that it includes task-level breakdowns.
- **D-02:** Sequential ordering: Oracle/RALF first, then swarm visibility, then broader build/continue parity. Oracle goes first because it proves the TS host can handle complex iterative flows. Swarm goes second as a simpler dispatch use case. Parity goes last because it depends on the patterns established by the first two.
- **D-03:** Dependency chain is explicit -- parity depends on Oracle and swarm patterns being proven. Do not parallelize.
- **D-04:** Migration only -- each item migrates existing Go behavior to TS host orchestration. No new features, no improvements beyond what's needed for the TS host integration. Respect the Go/TS boundary contract from Phase 106.
- **D-05:** Oracle migration = TS host drives the RALF loop with confidence targets, using Go manifests and finalizers. The Oracle loop logic stays in Go; TS host handles lifecycle orchestration.
- **D-06:** Swarm migration = TS host renders swarm display output (tree, JSON, text formats already exist in Go). Go owns the data, TS host owns the presentation.
- **D-07:** Parity migration = all remaining flows (colonize, seal, swarm, oracle) use TS host orchestration. Plan/build/continue already work through TS host (Phase 109).

### Claude's Discretion
- Exact milestone version numbers (v1.17, v1.18, etc.)
- Phase count per milestone
- Requirement ID naming convention
- Whether to produce one combined document or separate documents per item
- How to format the map (markdown document, separate milestone files, etc.)

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| MAP-01 | Written follow-up plan exists for restoring Oracle/RALF confidence iteration | Oracle loop complexity analysis (Section: Oracle/RALF Migration Complexity), established TS host pattern (Section: TS Host Lifecycle Pattern) |
| MAP-02 | Written follow-up plan exists for restoring swarm visibility | Swarm rendering command inventory (Section: Swarm Display Inventory), Go rendering capabilities (Section: Swarm Display Inventory) |
| MAP-03 | Written follow-up plan exists for broader build/continue parity (all flows use TS host) | Colonize flow analysis (Section: Remaining Flows), Seal flow analysis (Section: Remaining Flows), existing TS host commands (Section: TS Host Lifecycle Pattern) |
| MAP-04 | Map includes phase numbers, estimated scope, and dependency ordering | Dependency chain analysis (Section: Dependency Chain), scope estimates per item (Sections: Scope Estimates) |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Oracle RALF loop orchestration | TypeScript Host | Go (state/finalizer) | TS host drives the iterative loop; Go owns state writes via finalizers |
| Oracle confidence tracking | Go | -- | Go owns the oracle state file and plan file writes |
| Oracle question selection | Go | -- | Already implemented in Go; no need to migrate |
| Oracle worker invocation | TypeScript Host | Go (manifest) | TS host dispatches the worker per Go manifest |
| Swarm display rendering | TypeScript Host | Go (data source) | TS host invokes Go rendering commands and presents output |
| Swarm data (colony state) | Go | -- | Go reads COLONY_STATE.json; TS host calls Go commands |
| Colonize orchestration | TypeScript Host | Go (state/finalizer) | TS host follows plan-only/manifest/finalizer pattern |
| Colonize surveyor dispatch | TypeScript Host | Go (manifest) | TS host dispatches surveyor workers per manifest |
| Seal orchestration | TypeScript Host | Go (state/finalizer) | TS host drives the seal ceremony flow |
| Seal state mutation | Go | -- | Go owns COLONY_STATE.json writes (safety-critical) |
| Watch/swarm live refresh | TypeScript Host | Go (data source) | TS host polls Go for swarm state and renders |

## Standard Stack

N/A -- This is a documentation-only phase. No new libraries, frameworks, or runtime dependencies are introduced.

## Architecture Patterns

### Migration Map Document Structure

The output artifact is a structured markdown document with three milestone sections. Each milestone follows this internal structure:

```
## Milestone N: [Name]

### Scope
[What gets migrated, what stays in Go]

### Phases
| Phase | Name | Scope | Dependencies |
|-------|------|-------|--------------|

### Requirements
| ID | Description | Phase | Success Criteria |
|----|-------------|-------|------------------|

### Boundary Contract Compliance
[How this milestone respects the Go/TS boundary contract]

### Risk Assessment
[What could go wrong, mitigations]
```

### TS Host Lifecycle Pattern (established in Phase 109)

The existing pattern that all migrations follow:

```
1. TS host calls Go command with --plan-only flag
2. Go returns JSON manifest (no state mutation)
3. TS host processes manifest, dispatches workers
4. TS host builds completion file
5. TS host calls Go finalizer command with --completion-file
6. Go validates provenance, commits state atomically
```

**Source:** `.aether/ts-host/src/lifecycle.ts` -- `runLifecycle()` function [VERIFIED: codebase read]

Key contract points:
- TS host calls `callGoJSON()` via `go-bridge.ts` with `AETHER_OUTPUT_MODE=json` [VERIFIED: codebase read]
- Completion files written to `$TMPDIR/aether-lifecycle/` (NOT `.aether/data/`) [VERIFIED: codebase read]
- Go finalizer validates manifest provenance before any state write [VERIFIED: codebase read]
- Boundary enforcement via `assertNoDirectDataWrites()` rejects Go-owned paths [VERIFIED: codebase read]

### Oracle/RALF Loop Architecture

The Oracle RALF (Research-Analyze-Learn-Finalize) loop is an iterative research cycle:

```
User provides topic
  -> Initialize oracle workspace (state file, plan file, research plan)
  -> FOR each iteration (up to max_iterations):
       -> Select next question from plan
       -> Determine iteration phase (survey/analysis/synthesis)
       -> FOR each attempt (up to 2):
            -> Dispatch worker with research brief
            -> Parse worker response (confidence, findings, gaps)
            -> Update plan with findings
            -> If confidence target met OR all questions answered: BREAK
       -> Update oracle state (iteration count, confidence, phase)
       -> Check stop conditions (target reached, max iterations, manual stop)
  -> Finalize: write synthesis, derived reports, summary
```

**Source:** `cmd/oracle_loop.go` lines 811-960+ [VERIFIED: codebase read]

Key complexity factors:
- **Iterative state machine:** The loop has 5 phases (idle, survey, analysis, synthesis, finalization) with transitions based on question answering progress
- **Confidence targeting:** Each depth level has a target confidence (quick=60%, balanced=85%, deep=95%, exhaustive=99%)
- **Retry with narrowing:** Failed attempts get a narrower recovery prompt on retry
- **Background controller:** The loop can run as a detached background process with PID tracking
- **Workspace management:** Creates and archives workspace directories, manages state/plan/research files in `.aether/data/oracle/`
- **Worker invocation:** Uses the same `codex.WorkerInvoker` as build/continue flows

**Migration approach (D-05):** The TS host drives the outer iteration loop. Each iteration calls Go for:
1. `oracle --plan-only` equivalent (get current state + next question)
2. Worker dispatch (same pattern as build workers)
3. `oracle --finalize-iteration` equivalent (commit findings, update state)

The Go side keeps all oracle-specific logic (question selection, confidence calculation, phase transitions). The TS host just orchestrates the timing and worker dispatch.

### Swarm Display Inventory

Go already provides three swarm rendering commands with structured JSON output:

| Command | Formats | Output Fields |
|---------|---------|---------------|
| `aether swarm-display-render --format tree\|json\|flat` | tree, json, flat | `format`, `lines[]`, `total_lines` |
| `aether swarm-display-inline --section progress,memory` | inline text | `inline`, `sections{}` |
| `aether swarm-display-text --section phases --max-width 80` | multi-line text | `text`, `lines[]`, `width` |

**Source:** `cmd/swarm_display.go` [VERIFIED: codebase read]

Additionally, `cmd/compatibility_cmds.go` provides:
- `aether watch` -- live watch with configurable interval and ANSI refresh
- `buildSwarmWatchResult()` -- builds watch status from colony state
- `renderSwarmCompatibilityVisual()` -- renders visual watch output

**Source:** `cmd/compatibility_cmds.go` lines 30-227 [VERIFIED: codebase read]

**Migration approach (D-06):** TS host calls these Go commands with `AETHER_OUTPUT_MODE=json` and formats the structured output for presentation. No rendering logic needs to move to TS -- Go already handles all rendering.

### Remaining Flows for Parity

Flows that need TS host orchestration beyond what Phase 109 already provides (plan/build/continue):

**1. Colonize** (`cmd/codex_colonize.go`)
- Already has `--plan-only` support via `runCodexColonizePlanOnly()` [VERIFIED: codebase read]
- Returns `codexColonizeManifest` with dispatches, dispatch contract, and finalizer command
- Surveyor dispatches: provisions, nest, disciplines, pathogens (4 surveyors)
- Has `colonize-finalize` command for TS host to call
- Complexity: medium -- follows same plan-only/manifest/finalizer pattern as build

**2. Seal** (`cmd/codex_workflow_cmds.go`)
- Colony completion flow: promotes instincts, expires pheromones, writes CROWNED-ANTHILL.md, archives reviews
- **No existing --plan-only support** -- this is a single-shot ceremony, not a manifest-based flow
- State mutation: sets `StateCOMPLETED`, `Milestone: "Crowned Anthill"`, writes `COLONY_STATE.json`
- Complexity: low-medium -- could add a `seal-finalize` Go command that the TS host calls

**3. Oracle** (covered above -- most complex)

**4. Watch/Swarm** (covered above -- simplest)

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| JSON manifest parsing | Custom parser | `callGoJSON<T>()` from go-bridge.ts | Already handles envelope, error checking, type casting |
| Boundary enforcement | Custom path checking | `assertNoDirectDataWrites()` from go-bridge.ts | Already validates against GO_OWNED_PATHS |
| Completion file writing | Direct tmpdir manipulation | `writeCompletionFile()` from go-bridge.ts | Already handles tmpdir, boundary checks, directory creation |
| Worker dispatch | New dispatch logic | `dispatchWorkers()` / `dispatchSingleWorker()` from worker-dispatch.ts | Already handles spawn-log/complete lifecycle |
| Swarm rendering | TS-side tree/text rendering | Go `swarm-display-*` commands | Go already renders all formats; TS just invokes and presents |

**Key insight:** The Phase 109 TS host already provides all the building blocks (go-bridge, worker-dispatch, lifecycle). Oracle, swarm, and parity migrations are about wiring new Go commands into the existing TS host pattern, not building new TS infrastructure.

## Common Pitfalls

### Pitfall 1: Over-scoping the Oracle migration
**What goes wrong:** Planning the Oracle migration as "reimplement the RALF loop in TypeScript"
**Why it happens:** The loop is complex (~1000 lines) and it's tempting to migrate the whole thing
**How to avoid:** Per D-05, the loop logic stays in Go. TS host only drives the outer iteration cycle. The map should specify that Go provides `oracle-iterate --plan-only` (get next iteration spec) and `oracle-iterate-finalize --completion-file` (commit iteration results).
**Warning signs:** Migration plan includes TS-side question selection logic or confidence calculation

### Pitfall 2: Confusing "rendering" with "orchestration" for swarm
**What goes wrong:** Trying to move ANSI rendering code to TypeScript
**Why it happens:** D-06 says "TS host renders swarm display output"
**How to avoid:** "Renders" here means "invokes Go rendering commands and presents the structured JSON output." Go already does the actual rendering (tree, text, inline). TS host is the presentation layer, not the rendering engine.
**Warning signs:** Migration plan includes TS-side ANSI color codes or tree-drawing logic

### Pitfall 3: Ignoring seal's lack of --plan-only support
**What goes wrong:** Planning seal migration using the same plan-only/manifest/finalizer pattern as build
**Why it happens:** Seal is a single-shot ceremony, not a manifest-based workflow
**How to avoid:** The map should note that seal needs a new Go `seal-finalize` command (or equivalent) that the TS host calls. The Go side already has `completeSealRuntime()` -- it just needs a CLI-exposed finalizer entry point.
**Warning signs:** Migration plan references `aether seal --plan-only` (which does not exist)

### Pitfall 4: Treating parity as "copy Phase 109 for each flow"
**What goes wrong:** Assuming each flow is an identical copy of the build lifecycle pattern
**Why it happens:** Plan/build/continue all follow the same pattern
**How to avoid:** Each flow has different characteristics: colonize has 4 surveyor dispatches, seal is a ceremony, oracle is iterative, watch is continuous. The map should specify per-flow adaptation of the pattern.
**Warning signs:** Parity phase description says "same as build" without flow-specific details

### Pitfall 5: Forgetting the boundary contract
**What goes wrong:** Migration plan includes TS-side writes to `.aether/data/oracle/`
**Why it happens:** Oracle workspace files (state, plan, research plan) live in `.aether/data/oracle/`
**How to avoid:** All oracle state writes must go through Go finalizers. The TS host must not write oracle workspace files directly. This means the Go side needs new finalizer commands for oracle iteration state.
**Warning signs:** Migration plan has TS host writing `oracle-state.json` or `oracle-plan.json`

## Scope Estimates

### Oracle/RALF Migration (D-05)

| Aspect | Estimate | Rationale |
|--------|----------|-----------|
| New Go commands needed | 2-3 | `oracle-iterate --plan-only`, `oracle-iterate-finalize`, maybe `oracle-status-json` |
| New TS host functions | 2-3 | `runOracleLifecycle()`, maybe `runOracleStatus()` |
| New TS types | 3-5 | Oracle manifest, oracle completion, oracle state types |
| Phases in milestone | 2-3 | Phase 1: Go commands, Phase 2: TS host, Phase 3: integration tests |
| Risk level | High | Iterative loop is the most complex migration; confidence iteration behavior must be preserved exactly |

### Swarm Visibility Migration (D-06)

| Aspect | Estimate | Rationale |
|--------|----------|-----------|
| New Go commands needed | 0-1 | Existing `swarm-display-*` commands may suffice; maybe add `watch-status-json` |
| New TS host functions | 1-2 | `runSwarmDisplay()`, maybe `runLiveWatch()` |
| New TS types | 1-2 | Swarm display result types |
| Phases in milestone | 1-2 | Phase 1: TS host swarm rendering, Phase 2: live watch |
| Risk level | Low | Go already does all rendering; TS host just invokes commands |

### Parity Migration (D-07)

| Aspect | Estimate | Rationale |
|--------|----------|-----------|
| New Go commands needed | 2-3 | `colonize` already has `--plan-only`; need `seal-finalize`, maybe `watch-status-json` |
| New TS host functions | 3-4 | `runColonizeLifecycle()`, `runSealLifecycle()`, `runOracleLifecycle()` (if not in Oracle milestone) |
| New TS types | 2-4 | Colonize manifest, seal result types |
| Phases in milestone | 2-3 | Phase 1: colonize, Phase 2: seal + watch, Phase 3: end-to-end parity verification |
| Risk level | Medium | Most mechanical but seal needs new Go finalizer; colonize already has plan-only |

## Dependency Chain

```
Oracle Migration (Milestone A)
  |
  | proves TS host can handle iterative flows
  | establishes Go command pattern for complex state machines
  v
Swarm Visibility (Milestone B)
  |
  | proves TS host can handle continuous/polling flows
  | establishes Go command pattern for display data
  v
Parity Migration (Milestone C)
  |
  | depends on: Oracle pattern (iterative), Swarm pattern (polling)
  | brings all remaining flows under TS host
  v
Complete TS host coverage for all colony flows
```

**Milestone A must complete before Milestone B starts** (D-03).
**Milestone B must complete before Milestone C starts** (D-03).

## Code Examples

### Existing TS Host Lifecycle Pattern (for reference in map)

```typescript
// Source: .aether/ts-host/src/lifecycle.ts
// Pattern: callGoJSON -> process manifest -> dispatchWorkers -> writeCompletionFile -> callGoJSON(finalizer)

const planResult = callGoJSON<PlanManifestResult>(opts, [
  "plan",
  "--plan-only",
  "--depth",
  "fast",
]);
// ... process manifest, dispatch workers, write completion ...
callGoJSON<Record<string, unknown>>(opts, [
  "plan-finalize",
  "--completion-file",
  planCompletionPath,
]);
```

### Go Swarm Display Commands (for reference in map)

```go
// Source: cmd/swarm_display.go
// Three commands with JSON output:
// aether swarm-display-render --format tree|json|flat --max-depth 3
// aether swarm-display-inline --section progress,memory
// aether swarm-display-text --section phases --max-width 80

// All output via outputOK() with AETHER_OUTPUT_MODE=json:
// {"ok":true,"result":{"format":"tree","lines":[...],"total_lines":N}}
```

### Oracle Loop Core Structure (for reference in map)

```go
// Source: cmd/oracle_loop.go lines 837-870
// Core iteration loop:
for state.Iteration < state.MaxIterations {
    // 1. Check stop conditions
    // 2. Select next question
    // 3. Determine iteration phase
    // 4. Dispatch worker (with retry)
    // 5. Parse response, update plan
    // 6. Check confidence target
}
// Then finalize: write synthesis, derived reports
```

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `aether colonize --plan-only` returns sufficient manifest data for TS host dispatch | Remaining Flows | Colonize may need manifest extensions for TS host use |
| A2 | Seal can be adapted to the plan-only/finalizer pattern with a small Go addition | Remaining Flows | If seal ceremony is too tightly coupled to interactive flow, it may need a different approach |
| A3 | Go `swarm-display-*` commands already support `AETHER_OUTPUT_MODE=json` | Swarm Display | If they write directly to stdout instead of using outputOK(), TS host cannot parse them |
| A4 | Oracle loop can be externally driven without moving loop logic to TS | Oracle Architecture | If the loop has internal state dependencies that prevent external iteration control, more Go refactoring is needed |
| A5 | Phase 109 TS host infrastructure (go-bridge, worker-dispatch, lifecycle) is stable enough to extend | TS Host Pattern | If Phase 109 left significant gaps, more foundation work is needed before migrations |

## Open Questions

1. **Oracle workspace writes during iteration**
   - What we know: Oracle writes state file, plan file, research plan, artifacts, and derived reports to `.aether/data/oracle/` on each iteration
   - What's unclear: Can these writes be deferred to a Go finalizer, or does the loop need intermediate state writes between iterations?
   - Recommendation: Map should specify that Go provides a single `oracle-iterate-finalize` command that handles all state writes for one iteration. The TS host collects worker results and passes them to this finalizer.

2. **Seal ceremony interactive elements**
   - What we know: Seal checks blockers and can render a recovery menu if blockers exist
   - What's unclear: How should the TS host handle the blocker/recovery-menu case?
   - Recommendation: Map should specify that the TS host calls a Go `seal-check --plan-only` first, gets blocker list, and either reports blocked or calls `seal-finalize --force` if appropriate.

3. **Watch live refresh mode**
   - What we know: `aether watch` has a live refresh mode with ANSI terminal clearing and 2-second interval
   - What's unclear: Should the TS host implement its own polling loop, or should Go provide a streaming/status endpoint?
   - Recommendation: Map should specify TS host polls `swarm-display-render --format json` at configurable intervals. No streaming needed -- polling is simpler and sufficient.

## Project Constraints (from CLAUDE.md)

- Go owns all state mutation -- no other process writes `.aether/data/`
- TS host calls Go plan-only for manifests, Go finalizers for commits
- TS host never invents workers -- dispatches only from Go manifest
- Install, update, publish commands remain pure Go -- no TS involvement
- If docs and runtime disagree, runtime wins
- The TS host currently only supports plan, build, continue, and lifecycle commands

## Environment Availability

Step 2.6: SKIPPED (no external dependencies -- this is a documentation-only phase with no runtime code changes).

## Sources

### Primary (HIGH confidence)
- `.aether/references/contracts/runtime-boundary-contract.md` -- Go/TS ownership boundary [VERIFIED: codebase read]
- `.aether/ts-host/src/lifecycle.ts` -- Existing TS host lifecycle pattern [VERIFIED: codebase read]
- `.aether/ts-host/src/go-bridge.ts` -- Go command invocation and boundary enforcement [VERIFIED: codebase read]
- `.aether/ts-host/src/worker-dispatch.ts` -- Worker dispatch patterns [VERIFIED: codebase read]
- `.aether/ts-host/src/types.ts` -- TS type definitions for Go manifests [VERIFIED: codebase read]
- `.aether/ts-host/src/host.ts` -- TS host entry point and command routing [VERIFIED: codebase read]
- `.aether/ts-host/src/boundary-reference.ts` -- GO_OWNED_PATHS enforcement list [VERIFIED: codebase read]
- `cmd/oracle_loop.go` -- Oracle RALF loop implementation [VERIFIED: codebase read]
- `cmd/compatibility_cmds.go` -- Oracle command entry point, watch/swarm commands [VERIFIED: codebase read]
- `cmd/swarm_display.go` -- Swarm rendering commands [VERIFIED: codebase read]
- `cmd/codex_colonize.go` -- Colonize flow with plan-only support [VERIFIED: codebase read]
- `cmd/codex_workflow_cmds.go` -- Seal flow implementation [VERIFIED: codebase read]

### Secondary (MEDIUM confidence)
- `.planning/REQUIREMENTS.md` -- v1.16 requirements definitions [VERIFIED: codebase read]
- `.planning/STATE.md` -- Project state and deferred items [VERIFIED: codebase read]
- `.planning/phases/111-follow-up-migration-map/111-CONTEXT.md` -- User decisions [VERIFIED: codebase read]

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH (N/A -- documentation-only phase)
- Architecture: HIGH -- all source files read and cross-referenced
- Pitfalls: HIGH -- identified from concrete code patterns, not hypothetical
- Scope estimates: MEDIUM -- based on code complexity analysis, but actual effort may vary

**Research date:** 2026-05-12
**Valid until:** 90 days (documentation phase, codebase is stable)
