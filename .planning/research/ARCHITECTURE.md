# Architecture Research: v1.21 Live Colony — Production TS Host Orchestration

**Project:** Aether v1.21 Live Colony
**Domain:** CLI colony framework -- hybrid Go runtime + TypeScript orchestration host
**Researched:** 2026-05-18
**Confidence:** HIGH (based on direct source code analysis of all TS host modules, Go finalizer contracts, boundary reference, and queen orchestrator)

## Executive Summary

The v1.21 milestone converts the TypeScript host from a simulation-only orchestrator into the production control plane for Aether colonies. The existing architecture already has all the integration scaffolding in place: Go manifests (plan-only mode), Go finalizers (build/plan/continue/oracle-iterate-finalize), platform dispatchers (Claude/OpenCode/Codex), wave orchestration, Queen orchestration, event bridge, ceremony adapter, and the boundary contract. What is missing is wiring the production paths end-to-end, adding three new capabilities (worker-to-worker spawning, confidence-driven iteration for plan/build, and hive wisdom reuse proof), and removing the simulation guards that currently prevent real execution.

The four new capabilities integrate cleanly at the existing Go/TS boundary. None of them require the TS host to write to `.aether/data/` directly. Worker-to-worker spawning flows through the existing `spawn-log`/`spawn-complete` Go commands and a new Go finalizer for mid-build worker insertion. Confidence-driven iteration reuses the Oracle lifecycle pattern (plan-only -> dispatch -> finalize -> loop) applied to the build command. Hive wisdom reuse proof requires a new Go CLI surface (`hive-read` with domain filtering) that the TS host calls during the plan manifest consumption step to inject cross-colony wisdom into the context capsule.

## Recommended Architecture

### Layer Diagram

```
+-------------------------------------------------------------------------+
|  WRAPPER LAYER (.claude/commands/ant/*.md, .opencode/commands/ant/*.md) |
|  ----------------------------------------------------------------------|
|  * Host-assisted orchestrators: invoke TS host for plan/build/continue |
|  * Render ceremony via Go ceremony commands or narrator events          |
|  * NEVER mutate state; NEVER parse visual output as authority           |
+-------------------------------------------------------------------------+
                                  |
                                  | delegates to
                                  v
+-------------------------------------------------------------------------+
|  TS HOST LAYER (.aether/ts-host/src/)                                   |
|  ----------------------------------------------------------------------|
|  * host.ts: entry point, command routing                                |
|  * command-registry.ts: flag -> Go arg mapping for all 7+ commands      |
|  * go-bridge.ts: callGoJSON, writeCompletionFile, boundary enforcement  |
|  * lifecycle.ts: plan -> build -> continue orchestration                |
|  * worker-dispatch.ts: manifest -> spawn-log -> platform -> spawn-comp  |
|  * wave-orchestrator.ts: parallel wave dispatch with retry              |
|  * queen/orchestrator.ts: pattern selection, Builder-Probe Lock, midden|
|  * queen/escalation.ts: failure classification, recovery actions        |
|  * oracle-lifecycle.ts: RALF iteration loop (plan-only -> finalize)     |
|  * platform-dispatcher.ts: Claude/OpenCode/Codex subprocess spawning    |
|  * prompt-assembler.ts: agent def + context + task -> prompt             |
|  * claims-parser.ts: extract JSON claims from worker stdout              |
|  * event-bridge.ts: consume Go ceremony events via JSONL stream          |
|  * narrator.ts: render ceremony events to stdout                        |
|  * ceremony-adapter.ts: Go ceremony command wrapper (spawn-plan, etc.)  |
|  * boundary-reference.ts: GO_OWNED_PATHS enforcement                   |
|                                                                         |
|  NEW IN v1.21:                                                          |
|  * build-coordinator.ts: confidence-driven plan/build iteration loop    |
|  * worker-spawner.ts: worker-to-worker spawn request handling           |
|  * hive-injector.ts: cross-colony wisdom injection into context capsule |
+-------------------------------------------------------------------------+
                                  |
                    +-------------+-------------+
                    |             |             |
                    v             v             v
+------------------+  +----------+  +----------+
|  GO RUNTIME      |  | GO       |  | GO       |
|  (plan-only)     |  | FINALIZE |  | FINALIZE |
|  build N         |  | build    |  | continue |
|  --plan-only     |  | finalize |  | finalize |
|  plan            |  |          |  |          |
|  --plan-only     |  |          |  |          |
|  continue        |  |          |  |          |
|  --plan-only     |  |          |  |          |
|  oracle-iterate  |  |          |  |          |
|  --plan-only     |  |          |  |          |
+------------------+  +----------+  +----------+
```

### Component Boundaries

| Component | Responsibility | Communicates With | Boundary |
|-----------|---------------|-------------------|----------|
| `host.ts` | CLI entry point, command routing, arg parsing | All TS modules | Read-only; never writes state |
| `go-bridge.ts` | JSON envelope parsing, completion file writing, boundary assertion | Go CLI (subprocess) | Never writes `.aether/data/`; writes to tmpdir only |
| `worker-dispatch.ts` | Spawn-log -> platform dispatch -> spawn-complete -> claims | `go-bridge.ts`, `platform-dispatcher.ts`, `prompt-assembler.ts`, `claims-parser.ts` | Calls Go for spawn-log/spawn-complete only |
| `wave-orchestrator.ts` | Group by wave, parallel dispatch, retry with backoff | `worker-dispatch.ts` | Delegates all Go calls to worker-dispatch |
| `queen/orchestrator.ts` | Workflow pattern, Builder-Probe Lock, midden check, wave dispatch | `wave-orchestrator.ts`, `queen/escalation.ts`, `queen/builder-probe-lock.ts` | Calls Go for midden data via CLI |
| `oracle-lifecycle.ts` | RALF loop: plan-only -> dispatch -> finalize -> repeat | `go-bridge.ts`, `worker-dispatch.ts` | Calls Go for oracle-iterate, oracle-iterate-finalize |
| `platform-dispatcher.ts` | Platform detection, CLI arg building, subprocess spawning | Claude/OpenCode/Codex CLIs | Pure I/O; no state interaction |
| `prompt-assembler.ts` | Agent def + context + handoff + skills + pheromones -> prompt | File system (agent defs), `go-bridge.ts` (colony-prime) | Reads only; no state writes |
| `claims-parser.ts` | Extract structured JSON from worker stdout | No dependencies | Pure function; no I/O |
| `event-bridge.ts` | Subscribe to Go JSONL event stream | Go CLI (event-bus-subscribe) | Read-only subscriber; never writes |
| `narrator.ts` | Render ceremony events to stdout | `event-bridge.ts` (events in) | Presentation only |
| `ceremony-adapter.ts` | Call Go ceremony rendering commands | `go-bridge.ts` | Presentation only; writes to tmpdir |
| **`build-coordinator.ts`** (NEW) | Confidence-driven plan/build iteration loop | `go-bridge.ts`, `worker-dispatch.ts`, `queen/orchestrator.ts` | Calls Go for build --plan-only, build-finalize |
| **`worker-spawner.ts`** (NEW) | Handle mid-build worker spawn requests from workers | `go-bridge.ts`, `worker-dispatch.ts`, `platform-dispatcher.ts` | Calls Go for spawn-log, spawn-complete, new mid-build-finalize |
| **`hive-injector.ts`** (NEW) | Fetch hive wisdom and inject into worker context | `go-bridge.ts` (hive-read), `prompt-assembler.ts` | Calls Go for hive-read; no state writes |

### Data Flow: Production Build (v1.21)

```
Production Build Flow
---------------------
TS Host: host.ts build <N>
  |
  +-> Go: aether build <N> --plan-only
  |     +-> Go returns JSON: { dispatch_manifest, dispatches, queen_execution_policy }
  |     +-> Go emits: ceremony.build.prewave, ceremony.build.wave.start
  |
  +-> TS: hive-injector reads cross-colony wisdom
  |     +-> Go: aether hive-read --domain <tags> --threshold 0.7
  |     +-> Returns wisdom entries scoped to current project domain
  |     +-> Injects into prompt assembly context capsule
  |
  +-> TS: queen/orchestrator.ts runBuild(manifest)
  |     +-> Step 1: midden threshold check (existing)
  |     +-> Step 2: derive workflow pattern (existing)
  |     +-> Step 3: wave-orchestrator dispatches workers
  |           |
  |           +-> For each wave:
  |                 +-> narrator renders wave-start
  |                 +-> For each dispatch in parallel:
  |                 |     +-> go-bridge: aether spawn-log
  |                 |     +-> platform-dispatcher: spawn worker subprocess
  |                 |     |     +-> prompt-assembler: build prompt
  |                 |     |     +-> Claude/OpenCode/Codex runs agent
  |                 |     |     +-> claims-parser: extract JSON from stdout
  |                 |     +-> go-bridge: aether spawn-complete
  |                 |     +-> NEW: worker-spawner checks for spawn requests
  |                 |           +-> If worker's claims include spawns[]:
  |                 |                 +-> go-bridge: aether spawn-log (child worker)
  |                 |                 +-> platform-dispatcher: spawn child worker
  |                 |                 +-> go-bridge: aether spawn-complete (child)
  |                 |                 +-> Add child results to parent's handoff
  |                 +-> narrator renders worker-complete
  |                 +-> narrator renders wave-end
  |     +-> Step 4: Builder-Probe Lock (existing)
  |     +-> Step 5: failure classification + recovery (existing)
  |
  +-> TS: build-completion file written to tmpdir
  |     +-> go-bridge: writeCompletionFile()
  |
  +-> Go: aether build-finalize <N> --completion-file <path>
        +-> Go validates provenance, commits state, emits ceremony
        +-> Go returns: phase state, gate results, next-phase info
```

### Data Flow: Confidence-Driven Iteration (NEW)

```
Confidence-Driven Build Iteration (NEW in v1.21)
------------------------------------------------
TS Host: host.ts build <N> --target <confidence>
  |
  +-> Loop:
        +-> Go: aether build <N> --plan-only
        +-> TS: dispatch workers (real, not simulated)
        +-> TS: collect results
        +-> Go: aether build-finalize <N> --completion-file <path>
        +-> TS: evaluate build result against confidence target
        |     +-> If gate results include quality metrics below target:
        |           +-> Emit ceremony event for re-build
        |           +-> Loop back (max 3 iterations by default)
        |     +-> If target met OR max iterations exhausted:
        |           +-> Break loop
        +-> TS: return final build result
```

This mirrors the existing Oracle lifecycle pattern (oracle-lifecycle.ts) but applied to the build command. The key difference is that Oracle loops over `oracle-iterate --plan-only`, while this loops over `build <N> --plan-only`.

### Data Flow: Worker-to-Worker Spawning (NEW)

```
Worker-to-Worker Spawning (NEW in v1.21)
-----------------------------------------
During wave dispatch, after a worker returns:
  |
  +-> claims-parser extracts spawns[] field from worker claims
  |
  +-> worker-spawner.ts evaluates spawn requests:
        +-> Each spawn is { caste, task, brief? }
        +-> Validate against Queen spawn budget (from manifest.queen_execution_policy.spawn_budget)
        +-> If budget remaining:
              +-> go-bridge: aether spawn-log --parent <worker.name> --caste <caste> --name <child> --task <task>
              +-> platform-dispatcher: spawn child worker with assembled prompt
              +-> claims-parser: extract child claims
              +-> go-bridge: aether spawn-complete --name <child> --status <status>
              +-> Attach child results to parent worker's handoff.next_worker_instructions
        +-> If budget exhausted:
              +-> Log warning, skip spawn
              +-> Emit ceremony event: ceremony.build.spawn (blocked, budget exhausted)
  |
  +-> Child results are included in the parent's WorkerResult for the finalizer
```

The `spawns` field already exists in the worker claims schema (see `platform-dispatcher.ts` workerClaimsSchema). Workers already return `spawns: []` in their response contract. The TS host currently ignores this field. v1.21 makes it actionable.

### Data Flow: Hive Wisdom Reuse (NEW)

```
Hive Wisdom Reuse Injection (NEW in v1.21)
-------------------------------------------
Before dispatching any workers:
  |
  +-> hive-injector.ts:
        +-> Go: aether hive-read --domain <domain-tags> --threshold 0.7
        |     +-> Returns matching wisdom entries from ~/.aether/hive/wisdom.json
        |
        +-> Format wisdom entries as structured text section:
              ## Cross-Colony Wisdom
              - [confidence 0.85] <wisdom text> (from repo X)
              - [confidence 0.72] <wisdom text> (from repo Y)
        |
        +-> Inject into context capsule used by prompt-assembler.ts
        |
        +-> All workers in this build receive the wisdom section
```

The `hive-read` Go command already exists (`cmd/hive.go`). The TS host needs to call it during the pre-build phase and inject the results into the prompt assembly context capsule. This is a read-only operation on the hub; no boundary contract changes needed.

## New Components

### 1. build-coordinator.ts

**Purpose:** Confidence-driven plan/build iteration loop, analogous to oracle-lifecycle.ts but for builds.

**Interface:**
```typescript
interface BuildCoordinatorOptions extends GoBridgeOptions {
  phase: number;
  targetConfidence?: number;   // 70-99, default from Go manifest
  maxIterations?: number;      // 2-5, default 3
  acceptBelowTarget?: boolean; // --accept flag passthrough
  simulateWorkers?: boolean;
}

interface BuildCoordinatorResult {
  success: boolean;
  iterations_completed: number;
  final_confidence: number;
  confidence_target: number;
  stop_reason: string;
  worker_results: WorkerResult[];
}
```

**Key design decisions:**
- Reuses the `oracle-iterate --plan-only` / `oracle-iterate-finalize` loop pattern
- Go finalizer returns gate quality metrics after each build; TS host evaluates against target
- Max iterations cap prevents infinite re-build loops (hard safety net)
- The `--accept` flag allows completing below target (same as Oracle's accept behavior)

**Dependencies:** `go-bridge.ts`, `worker-dispatch.ts`, `queen/orchestrator.ts`, `ceremony-adapter.ts`

### 2. worker-spawner.ts

**Purpose:** Handle mid-build worker spawn requests from worker claims.

**Interface:**
```typescript
interface SpawnRequest {
  caste: string;
  task: string;
  brief?: string;
}

interface SpawnResult {
  name: string;
  status: TerminalWorkerStatus;
  summary: string;
  files_modified?: string[];
}

interface WorkerSpawnerOptions extends GoBridgeOptions {
  parentWorker: string;
  spawnBudget: QueenSpawnBudget;
  cwd: string;
}

async function handleSpawnRequests(
  opts: WorkerSpawnerOptions,
  spawns: SpawnRequest[]
): Promise<SpawnResult[]>;
```

**Key design decisions:**
- Spawn budget comes from the manifest's `queen_execution_policy.spawn_budget`
- Each spawn request validates against remaining budget before dispatching
- Child workers are dispatched via the same `platform-dispatcher.ts` and `prompt-assembler.ts` path
- Child worker results are recorded via Go `spawn-log`/`spawn-complete` (existing commands)
- Child results attach to parent's handoff for the Go finalizer

**Dependencies:** `go-bridge.ts`, `platform-dispatcher.ts`, `prompt-assembler.ts`, `claims-parser.ts`

### 3. hive-injector.ts

**Purpose:** Fetch cross-colony wisdom from the hive and format it for prompt injection.

**Interface:**
```typescript
interface HiveWisdomEntry {
  id: string;
  text: string;
  domain: string;
  source_repo: string;
  confidence: number;
}

interface HiveInjectorOptions extends GoBridgeOptions {
  domainTags: string[];
  confidenceThreshold?: number; // default 0.7
}

async function fetchHiveWisdom(
  opts: HiveInjectorOptions
): Promise<HiveWisdomEntry[]>;

function formatWisdomForInjection(entries: HiveWisdomEntry[]): string;
```

**Key design decisions:**
- Calls `aether hive-read --domain <tags> --threshold <n>` (existing Go command)
- Domain tags come from the colony registry (Go stores these per-repo)
- The TS host reads domain tags from the manifest or from `aether registry-read`
- Formatted wisdom is prepended to the context capsule in `prompt-assembler.ts`
- This is purely additive; workers without hive wisdom work identically

**Dependencies:** `go-bridge.ts`

## Modified Components

### 1. worker-dispatch.ts

**Change:** After `dispatchRealWorker()` returns, check `claims.spawns` for spawn requests and delegate to `worker-spawner.ts`.

**Current code (lines 237-307):** `dispatchRealWorker` parses claims and builds a DispatchResult. Add a post-processing step:
```typescript
// After claims parsing:
if (claims.spawns && claims.spawns.length > 0 && !simulate) {
  const spawnResults = await handleSpawnRequests(spawnOpts, claims.spawns);
  result.child_results = spawnResults; // Add to DispatchResult type
}
```

### 2. lifecycle.ts

**Change:** Remove the simulation-only guard (line 268-274) that returns early when `simulateWorkers` is not true. The lifecycle should work in production mode.

**Current code:**
```typescript
if (opts.simulateWorkers !== true) {
  return { success: false, steps_completed: [], error: "Lifecycle host is experimental..." };
}
```

This guard must be replaced with platform detection (already done at line 418 for the build step) and proper error handling.

### 3. prompt-assembler.ts

**Change:** Add an optional `hiveWisdomSection` to `PromptAssemblyConfig` and include it in the assembled prompt between context capsule and handoff sections.

### 4. types.ts

**Change:** Add `child_results` field to `DispatchResult` interface for worker-spawned child results. Add `BuildCoordinatorResult` type. Add `SpawnRequest` type.

### 5. command-registry.ts

**Change:** The `build` command's `buildGoArgs` function already supports `--target` and `--max-iterations` flags (via `targetConfidence` and `maxIterations` parsed args). These are already wired to the Go plan-only command. For the TS host's confidence iteration, the build-coordinator reads these flags and drives the loop in TS, not in Go.

**New host command:** Consider a `build-loop` runner type that drives the confidence iteration:
```typescript
case "build": {
  if (parsed.targetConfidence || parsed.maxIterations) {
    // Use build-coordinator for confidence-driven iteration
    const result = await runBuildCoordinator(buildCoordOpts);
  } else {
    // Single build pass (existing behavior)
    const result = callGoJSON(...);
  }
}
```

### 6. host.ts

**Change:** Wire `build-coordinator.ts` into the build command path when confidence flags are present. Wire `hive-injector.ts` into the pre-build context assembly.

## Boundary Contract Preservation

| Rule | How v1.21 Upholds It |
|------|----------------------|
| Go owns all state mutation | `hive-injector.ts` calls `aether hive-read` (read-only). `worker-spawner.ts` calls `aether spawn-log`/`aether spawn-complete` (Go-side state write). `build-coordinator.ts` calls `aether build-finalize`. No direct `.aether/data/` writes from any new module. |
| TS host calls Go plan-only for manifests | `build-coordinator.ts` loops over `aether build N --plan-only`, never mutations without finalizer. |
| TS host never invents workers | `worker-spawner.ts` only dispatches workers explicitly requested by existing manifest workers via `spawns[]` claims. Budget validation prevents runaway spawn chains. |
| Completion files go to tmpdir | All `writeCompletionFile` calls use tmpdir (enforced by `go-bridge.ts`). New coordinator follows the same pattern. |
| No visual output parsing as authority | New modules consume Go JSON output only. No ANSI parsing. |

## Go Commands Used by New Modules

| Go Command | Called By | Purpose | Existing? |
|------------|-----------|---------|-----------|
| `aether build <N> --plan-only` | build-coordinator | Get build manifest | Yes |
| `aether build-finalize <N> --completion-file <path>` | build-coordinator | Commit build state | Yes |
| `aether spawn-log --parent <name> --caste <caste> --name <name> --task <task>` | worker-spawner | Record child worker spawn | Yes |
| `aether spawn-complete --name <name> --status <status> --summary <summary>` | worker-spawner | Record child worker result | Yes |
| `aether hive-read --domain <tags> --threshold <n>` | hive-injector | Fetch cross-colony wisdom | Yes |
| `aether registry-read` | hive-injector | Get domain tags for current repo | Yes |
| `aether colony-prime --compact` | worker-dispatch | Get context capsule for prompt assembly | Yes |

All Go commands already exist. No new Go-side development is required for v1.21. The entire milestone is TS-side integration.

## Patterns to Follow

### Pattern 1: Manifest-Only-Then-Finalize

**What:** Every workflow step follows the two-phase pattern: (1) call Go with `--plan-only` to get a JSON manifest, (2) dispatch workers, (3) write a completion file to tmpdir, (4) call Go finalizer to commit state.

**When:** All orchestrated workflows (plan, build, continue, seal, oracle).

**Why:** This is the core boundary contract. The TS host never mutates state directly. Go owns the state machine and validates all inputs before committing.

**Example (from oracle-lifecycle.ts):**
```typescript
// Step 1: Get manifest (read-only)
const manifest = callGoJSON<OracleIterationManifest>(opts, [
  "oracle-iterate", "--plan-only", "--topic", topic,
]);

// Step 2: Dispatch workers
const dispatchResult = await dispatchSingleWorkerRef(opts, dispatch);

// Step 3: Write completion to tmpdir
const completionPath = writeCompletionFileRef(
  "aether-oracle", `oracle-completion-${state.current_iteration}.json`, completionData
);

// Step 4: Finalize via Go (state mutation)
const finalizeResult = callGoJSONRef<OracleIterationCompletion>(opts, [
  "oracle-iterate-finalize", "--completion-file", completionPath,
]);
```

**Apply to build-coordinator.ts:** Same pattern, but looping until confidence target is met.

### Pattern 2: Budget-Capped Spawn Tree

**What:** The Queen spawn budget (`queen_execution_policy.spawn_budget`) caps the total number of workers that can be spawned in a single build, including child spawns from worker-to-worker requests.

**When:** Every build dispatch and worker-to-worker spawn.

**Why:** Prevents exponential spawn chains (worker A spawns B, B spawns C and D, etc.). The budget is set by Go based on phase risk and colony size.

**How:**
```typescript
function hasBudgetRemaining(budget: QueenSpawnBudget, used: number): boolean {
  const max = budget.max_workers ?? budget.worker_count ?? 10;
  return used < max;
}
```

### Pattern 3: Graceful Degradation on Missing Go Commands

**What:** When a Go CLI command is unavailable or returns an error, the TS host degrades gracefully rather than crashing.

**When:** Hive wisdom injection, failure classification, midden checks.

**Why:** The TS host runs in heterogeneous environments where Go commands may be partially available or the user may be running an older binary.

**Example (from escalation.ts):**
```typescript
try {
  const result = callGoJSON<{classification?: FailureClassification}>(opts, [
    "failure-classify", "--status", status, "--summary", summary,
  ]);
  if (result.classification) return result.classification;
} catch {
  // Go command unavailable -- fall through to heuristic
}
```

**Apply to hive-injector.ts:** If `hive-read` fails or returns empty, proceed without hive wisdom. Log a warning but never block the build.

## Anti-Patterns to Avoid

### Anti-Pattern 1: Recursive Worker Spawning Without Budget Cap
**What:** A worker spawns a child, which spawns a grandchild, which spawns more, creating an unbounded tree.
**Why bad:** Exhausts API rate limits, runs forever, produces unintelligible results.
**Instead:** Enforce the Queen spawn budget across all spawn levels. Count both manifest workers and child spawns against the same budget. Log budget exhaustion clearly.

### Anti-Pattern 2: TS Host Evaluates Confidence Itself
**What:** The TS host reads test output, gate results, or file changes and computes its own confidence score.
**Why bad:** Duplicates Go's verification logic. Creates drift between TS and Go confidence calculations.
**Instead:** The TS host should rely on confidence metrics from the Go finalizer result. If Go does not return confidence metrics for build, the TS host uses a simple heuristic (all gates passed = confidence met) rather than reimplementing Go's verification.

### Anti-Pattern 3: Worker Spawns Writes to .aether/data/ Directly
**What:** The worker-spawner module tries to update COLONY_STATE.json or the spawn tree directly.
**Why bad:** Violates boundary contract. Race conditions with Go file locking.
**Instead:** Use `aether spawn-log` and `aether spawn-complete` Go commands (existing). The Go runtime handles spawn tree updates atomically.

### Anti-Pattern 4: Hive Wisdom Injection Blocks on Missing Hive
**What:** The hive-injector fails the build if `hive-read` returns no entries or the hive is not initialized.
**Why bad:** Not all repos have hive wisdom. First-time colonies should work fine without it.
**Instead:** Treat hive wisdom as optional enrichment. Log when no wisdom is available and proceed.

### Anti-Pattern 5: Confidence Iteration Re-dispatches Without Checking Phase State
**What:** The build-coordinator re-dispatches workers without checking if the Go finalizer already advanced the phase or changed state.
**Why bad:** Re-building an already-completed phase corrupts state.
**Instead:** Before each iteration, read the current phase state from the manifest. Only re-dispatch if the phase is still in a buildable state.

## Scalability Considerations

| Concern | At 5 workers (typical) | At 20 workers (large) | At 50 workers (stress) |
|---------|----------------------|----------------------|------------------------|
| Worker-to-worker spawns | 2-3 child spawns | 5-10 child spawns | Budget-capped at max_workers |
| Confidence iteration loops | 1-2 iterations | 2-3 iterations | 3 max (hard cap) |
| Hive wisdom injection | <1KB additional context | <5KB additional context | <10KB (fits within 8K budget) |
| Build completion file size | ~5 KB | ~20 KB | ~50 KB |
| Parallel wave dispatch | All in one wave | 2-3 waves | Needs worktree mode |
| Go subprocess calls per build | ~20 (spawn-log + spawn-complete per worker) | ~80 | ~200 (acceptable within 10min build timeout) |

## Suggested Build Order (Respecting Dependencies)

```
Phase A: Hive Wisdom Reuse (no dependencies on other new modules)
  1. hive-injector.ts -- call aether hive-read, format for injection
  2. prompt-assembler.ts -- add hiveWisdomSection to config
  3. Integration test: colony B receives wisdom from colony A

Phase B: Worker-to-Worker Spawning (depends on existing dispatch)
  4. worker-spawner.ts -- budget validation, child dispatch, spawn-log/complete
  5. worker-dispatch.ts -- add spawn handling post-processing
  6. types.ts -- add child_results to DispatchResult
  7. Integration test: builder spawns a watcher to verify its output

Phase C: Confidence-Driven Iteration (depends on Phase B for real dispatch)
  8. build-coordinator.ts -- loop over build plan-only + finalize
  9. host.ts -- wire build-coordinator when --target flag present
  10. command-registry.ts -- ensure --target and --max-iterations flags flow correctly
  11. Integration test: build iterates until quality gates pass

Phase D: Production Guard Removal + End-to-End
  12. lifecycle.ts -- remove simulation-only guard, add platform detection
  13. host.ts -- ensure all commands work without --simulate
  14. End-to-end test: full plan -> build (real) -> continue (real) lifecycle
  15. Golden parity tests: compare v1.21 output against expected ceremony format
```

**Phase ordering rationale:**
- Phase A is independent and proves the simplest new capability first.
- Phase B adds spawn functionality to the existing dispatch path without changing control flow.
- Phase C introduces the iteration loop, which requires real dispatch (Phase B) to be meaningful.
- Phase D removes the simulation guards last, ensuring all new functionality works before going live.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| New module design | HIGH | Based on direct source analysis of all existing TS host modules |
| Go command availability | HIGH | All Go commands (hive-read, spawn-log, spawn-complete, build-finalize) verified in cmd/ |
| Boundary contract preservation | HIGH | All new modules follow the plan-only -> finalize pattern; no direct .aether/data/ writes |
| Worker-to-worker spawn budget enforcement | MEDIUM | Go spawn_budget exists but mid-build child spawn tracking is new; need to verify Go handles child spawn-log entries correctly |
| Confidence iteration loop | MEDIUM | Oracle lifecycle provides the proven pattern; applying to build is straightforward but needs Go finalizer to return usable confidence metrics |
| Hive wisdom reuse proof | HIGH | hive-read Go command exists and returns structured data; integration is straightforward |
| Production guard removal | MEDIUM | lifecycle.ts simulation guard removal is the highest-risk change; needs careful platform detection and error handling |

## Sources

- `.aether/ts-host/src/host.ts` -- TS host entry point and command routing
- `.aether/ts-host/src/go-bridge.ts` -- Go CLI bridge, completion file writing, boundary enforcement
- `.aether/ts-host/src/worker-dispatch.ts` -- Worker dispatch with spawn-log/spawn-complete lifecycle
- `.aether/ts-host/src/wave-orchestrator.ts` -- Parallel wave dispatch with retry
- `.aether/ts-host/src/queen/orchestrator.ts` -- Queen build orchestration with workflow patterns
- `.aether/ts-host/src/queen/types.ts` -- Queen types including spawn budget
- `.aether/ts-host/src/queen/escalation.ts` -- Failure classification and recovery actions
- `.aether/ts-host/src/oracle-lifecycle.ts` -- RALF iteration loop pattern (template for build-coordinator)
- `.aether/ts-host/src/platform-dispatcher.ts` -- Platform CLI spawning and claims schema
- `.aether/ts-host/src/prompt-assembler.ts` -- Agent definition loading and prompt assembly
- `.aether/ts-host/src/claims-parser.ts` -- Worker claims extraction including spawns field
- `.aether/ts-host/src/event-bridge.ts` -- Go event stream subscription
- `.aether/ts-host/src/lifecycle.ts` -- Full lifecycle orchestrator (simulate-only guard)
- `.aether/ts-host/src/ceremony-adapter.ts` -- Go ceremony command wrapper
- `.aether/ts-host/src/boundary-reference.ts` -- GO_OWNED_PATHS and boundary enforcement
- `.aether/ts-host/src/command-registry.ts` -- Host command definitions and flag mapping
- `.aether/ts-host/src/types.ts` -- TypeScript type definitions for all Go manifest/finalizer types
- `cmd/codex_build_finalize.go` -- Go build finalizer with external completion handling
- `cmd/hive.go` -- Go hive commands (hive-read, hive-store, hive-init)
- `.planning/research/ARCHITECTURE.md` -- v1.17 Classic Restoration architecture research
- `.planning/PROJECT.md` -- Project context and milestone history
