# Technology Stack: v1.21 Live Colony

**Project:** Aether v1.21 Live Colony
**Researched:** 2026-05-18
**Confidence:** HIGH (based on codebase analysis + npm registry verification)

## Executive Summary

The v1.21 milestone makes the TypeScript host a real production orchestrator. This requires four new capabilities: production worker dispatch (removing simulate-only guards), worker-to-worker spawning (workers requesting sub-workers mid-build), confidence-driven iteration (looping plan/build/continue until quality targets are met), and hive wisdom reuse verification (proving colony B benefits from colony A's learnings).

The primary finding: the TS host already has nearly all the code it needs. The `dispatchRealWorker()` path is fully implemented. The `spawns` field exists in the worker claims schema. The Oracle lifecycle already demonstrates confidence-driven iteration. The Go runtime already owns all hive operations. What is missing is not dependencies, but wiring -- removing the simulate-only guard, consuming the `spawns` field, generalizing the Oracle loop pattern, and calling `hive-read` during context assembly.

Only one new runtime dependency is recommended: `p-limit` for concurrency control of dynamic worker spawning.

## Recommended Stack Additions

### New Runtime Dependency

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| `p-limit` | 7.3.0 | Concurrency control for dynamic worker pools | Worker-to-worker spawning creates an unbounded set of sub-workers. `p-limit` provides a configurable concurrency cap (Promise pool) with zero dependencies, ESM-native, Node >=20. The existing `wave-orchestrator.ts` uses `Promise.all` which cannot cap concurrency for dynamically spawned workers. |

### No Changes Needed To Existing Dependencies

The v1.17 rendering stack (chalk, boxen, figlet, ora, cli-progress, log-update, chokidar, strip-ansi, js-yaml) remains correct and sufficient. No version bumps required.

| Current Dependency | Current Version | Status | Reason |
|-------------------|-----------------|--------|--------|
| `chalk` | 5.6.2 | Keep | No rendering changes needed |
| `boxen` | 8.0.1 | Keep | Ceremony boxes work as-is |
| `chokidar` | 5.0.0 | Keep | Event bridge already wired |
| `cli-progress` | 3.12.0 | Keep | Multi-bar dashboard works as-is |
| `figlet` | 1.11.0 | Keep | Banners work as-is |
| `js-yaml` | 4.1.1 | Keep | YAML parsing unchanged |
| `log-update` | 8.0.0 | Keep | Dashboard refresh unchanged |
| `ora` | 9.4.0 | Keep | Worker spinners work as-is |
| `strip-ansi` | 7.2.0 | Keep | Width calculation unchanged |

### Existing Code That Already Covers Each Capability

| Capability | Existing Code | What It Does | What Needs Changing |
|-----------|---------------|--------------|---------------------|
| Production worker dispatch | `dispatchRealWorker()` in `worker-dispatch.ts` lines 237-307 | Detects platform, assembles prompt, spawns subprocess, parses claims | Remove simulate-only guard in `lifecycle.ts` line 268 and `host.ts` lines 308-312 |
| Worker-to-worker spawning | `spawns` field in `platform-dispatcher.ts` schema (line 374, 388) and `claims-parser.ts` (line 46) | Schema accepts `spawns: string[]` from worker output | Consume `spawns` field in dispatch loop, apply Queen budget, dispatch sub-workers |
| Confidence-driven iteration | `oracle-lifecycle.ts` `runOracleLifecycle()` | Loops manifest -> dispatch -> finalize -> check stop conditions | Generalize into reusable `ConfidenceLoop` class, apply to plan/build/continue |
| Hive wisdom reuse | Go CLI: `hive-store`, `hive-read`, `hive-promote` in `cmd/hive.go` | Domain-scoped wisdom with 200-entry cap, multi-repo confidence boosting | Call `aether hive-read` in prompt-assembler context chain, verify injection |

## Feature 1: Production TS Host Orchestration

### What Changes (No New Dependencies)

The production unlock is a guard removal, not a new library:

**File: `lifecycle.ts` line 268-275** -- Remove the simulate-only check:
```typescript
// CURRENT (blocks production):
if (opts.simulateWorkers !== true) {
  return { success: false, error: "simulate-only" };
}

// NEW (production-ready):
// Remove the guard entirely. The lifecycle should work with real workers
// when simulateWorkers is false or undefined.
```

**File: `host.ts` lines 308-312** -- Remove the entry-point guard:
```typescript
// CURRENT (blocks production):
if (!simulate) {
  process.stderr.write("Error: lifecycle is experimental and simulate-only...");
  process.exit(1);
}

// NEW: Remove this block entirely.
```

**File: `lifecycle.ts` line 424** -- Remove the hardcoded simulation override:
```typescript
// CURRENT (forces simulation even when platform available):
const simulateWorkers = true;

// NEW: Respect the caller's option:
const simulateWorkers = opts.simulateWorkers ?? false;
```

### Integration: Lifecycle-Level AbortController

Add graceful shutdown for the full plan->build->continue lifecycle. The `AbortController` pattern already exists in `platform-dispatcher.ts` (per-worker timeout) and `watch-display.ts` (SIGINT). Extend it to the lifecycle level:

```typescript
// New: lifecycle-level cancellation controller
const lifecycleController = new AbortController();
process.on("SIGINT", () => lifecycleController.abort());

// Pass signal to all dispatch calls
const queenOpts = {
  ...opts,
  signal: lifecycleController.signal,
};
```

This requires zero new dependencies. `AbortController` is a Web API available in Node >= 15.

### What NOT to Add

| Technology | Why Avoid |
|------------|-----------|
| `execa` or `zx` | The Go bridge (`go-bridge.ts`) uses `child_process.execFileSync` deliberately -- synchronous, no shell injection, matches the JSON envelope contract. Platform dispatch uses `child_process.spawn` with explicit args. Adding a subprocess wrapper adds a dependency for no benefit. |
| `bottleneck` | More feature-rich than `p-limit` (priority queues, reservoirs, timeouts) but 10x larger. The TS host needs a simple concurrency cap, not a full job scheduler. |
| `worker_threads` | Workers are external platform CLIs (claude, opencode, codex), not JS functions. `worker_threads` is for in-process parallelism of JS code, which is not what we need. |

## Feature 2: Worker-to-Worker Spawning

### How It Works (No New Architecture Needed)

The `spawns` field already exists in the worker claims schema:

```typescript
// platform-dispatcher.ts -- workerClaimsSchema already includes:
"spawns": {
  "type": "array",
  "items": { "type": "string" },
  "description": "Names of additional workers this worker requests"
}
```

The flow is:
1. Worker completes its task and returns claims JSON with `spawns: ["sub-worker-1", "sub-worker-2"]`
2. TS host parses claims via `parseWorkerClaims()` (already works)
3. New `SpawnPool` class checks `spawns` array, applies Queen spawn budget constraints
4. For each approved spawn, call existing `dispatchSingleWorker()` with a synthetic dispatch entry
5. Sub-worker results feed back into the parent's completion

### Why `p-limit` Is the Only New Dependency

Without `p-limit`, worker-to-worker spawning creates unbounded concurrency. A single builder could request 5 sub-workers, each of which requests 5 more, etc. `p-limit` caps this:

```typescript
import pLimit from "p-limit";

// Cap total concurrent workers (including sub-workers) to Queen's budget
const concurrencyLimit = pLimit(queenBudget.max_workers ?? 4);

// Each spawn dispatch goes through the limiter
await concurrencyLimit(() => dispatchSingleWorker(opts, subDispatch));
```

### Sub-Worker Dispatch Entry Construction

The TS host must construct a synthetic `BuildDispatch` from the `spawns` field. This requires Go's `aether spawn-log` to accept ad-hoc worker names:

```typescript
// The spawns field contains task descriptions, not structured dispatches.
// The TS host creates minimal dispatch entries:
const subDispatch: BuildDispatch = {
  stage: parentDispatch.stage,
  wave: parentDispatch.wave,
  caste: "builder", // Default; could be inferred from task description
  name: `${parentDispatch.name}-sub-${i}`,
  task: spawnDescription,
  status: "pending",
};
```

### What NOT to Add

| Technology | Why Avoid |
|------------|-----------|
| `bullmq` or `agenda` | Full job queue with Redis. Workers are short-lived CLI subprocesses, not long-running services. A queue adds operational complexity (Redis dependency) for no benefit. |
| `rabbitmq` or `amqplib` | Message queue for distributed systems. The TS host runs in a single process on a single machine. IPC via `child_process.spawn` is sufficient. |
| Custom tree-based spawn tracker | Tempting to model as a spawn tree (parent -> children -> grandchildren). But the Go runtime already tracks spawn-log and spawn-complete per worker. The TS host should delegate tracking to Go, not duplicate it. |

## Feature 3: Confidence-Driven Iteration

### Template Already Exists: `oracle-lifecycle.ts`

The Oracle lifecycle demonstrates the exact pattern needed:

```
1. Get manifest from Go (--plan-only)
2. Check stop conditions (confidence target, max iterations, no progress)
3. If not met: dispatch workers
4. Build completion file
5. Call Go finalizer
6. Go returns updated confidence
7. Loop to step 2
```

### Generalization: `ConfidenceLoop` Class

Extract the Oracle loop pattern into a reusable class. No external dependency -- pure TypeScript using existing Go bridge calls:

```typescript
interface ConfidenceLoopOptions {
  goBinaryPath: string;
  cwd: string;
  targetConfidence: number;    // e.g., 90
  maxIterations: number;       // e.g., 5
  noProgressLimit: number;     // Stop after N iterations with no confidence gain
  manifestCommand: string[];   // e.g., ["plan", "--plan-only"]
  finalizeCommand: string[];   // e.g., ["plan-finalize", "--completion-file", ...]
  dispatchFn: (manifest) => Promise<WorkerResult[]>;
  extractConfidence: (finalizeResult) => number;
}
```

### Stop Conditions

Already partially defined in `OracleStopConditions` (types.ts line 456-465). Extend:

| Condition | Source | Action |
|-----------|--------|--------|
| Confidence met | Go finalize returns `confidence >= target` | Stop (success) |
| Max iterations | Counter reaches `maxIterations` | Stop (with warning) |
| No progress | Confidence unchanged for `noProgressLimit` iterations | Stop (with warning) |
| Manual stop | SIGINT / user cancellation | Stop (abort) |
| Gate failure | Go finalize returns blocked=true | Stop (requires intervention) |

### What NOT to Add

| Technology | Why Avoid |
|------------|-----------|
| `iteratop` | NPM package for convergent iteration loops. Zero stars, zero forks, v0.3.0, no updates in 2+ years. The pattern is 50 lines of TypeScript -- not worth a dependency with no community. |
| `tough-cookie` / `retry-axios` | HTTP retry libraries. The TS host does not make HTTP requests to workers; it spawns CLI subprocesses. Retry logic is already in `wave-orchestrator.ts` via `runRetryLoop()`. |

## Feature 4: Hive Wisdom Reuse Verification

### Go Already Owns Everything

All hive operations are Go CLI subcommands:

| Operation | Command | Implementation |
|-----------|---------|---------------|
| Store wisdom | `aether hive-store --text "..." --domain "web" --source-repo "repo-a"` | `cmd/hive.go` |
| Read wisdom | `aether hive-read --domain "web" --confidence-threshold 0.7` | `cmd/hive.go` |
| Promote instinct | `aether hive-promote --text "..." --source-repo "repo-a"` | `cmd/hive.go` |
| Initialize | `aether hive-init` | `cmd/hive.go` |

### What the TS Host Needs to Do

Call `aether hive-read` during context assembly and inject results into worker prompts. This is a 10-line addition to `prompt-assembler.ts`:

```typescript
// In renderContextCapsule(), after QUEEN.md fallback:
const hiveWisdom = readHiveWisdom(config.cwd, config.platform);
if (hiveWisdom) {
  parts.push("## Hive Wisdom (Cross-Colony Patterns)\n\n" + hiveWisdom);
}

// New helper function:
function readHiveWisdom(cwd: string, platform: Platform): string {
  try {
    const bridge: GoBridgeOptions = { goBinaryPath: discoverGoBinary(), cwd };
    const result = callGoJSON<{ entries?: Array<{ text: string; confidence: number }> }>(
      bridge, ["hive-read"]
    );
    if (!result.entries?.length) return "";
    return result.entries
      .filter(e => e.confidence >= 0.7)
      .map(e => `- [${(e.confidence * 100).toFixed(0)}%] ${e.text}`)
      .join("\n");
  } catch {
    return ""; // Graceful degradation
  }
}
```

### Verification Proof Approach

To prove colony B benefits from colony A:

1. **Colony A** completes work. High-confidence instincts (>= 0.8) are promoted to hive via `hive-promote` at seal.
2. **Colony B** starts in the same domain. During `assemblePrompt()`, `hive-read` retrieves colony A's wisdom scoped to the domain.
3. **Verification**: Compare colony B's first-phase build quality (tests passing, fewer retries, faster completion) against a baseline colony that did not receive hive wisdom.

This verification is a test scenario, not a code feature. No new dependency needed.

### What NOT to Add

| Technology | Why Avoid |
|------------|-----------|
| Redis / SQLite for hive storage | Go already uses a single JSON file (`~/.aether/hive/wisdom.json`) with file locking. Adding a database for 200 entries is massive over-engineering. |
| Vector embeddings / semantic search | Hive wisdom is domain-tagged and confidence-scored. Exact domain matching + confidence threshold is the right retrieval strategy for 200 entries. Semantic search adds complexity (embedding model dependency, vector DB) with no benefit at this scale. |
| gRPC / protobuf for cross-colony communication | Colonies on the same machine share `~/.aether/hive/`. No network communication needed. |

## Installation

```bash
# Single new runtime dependency
cd .aether/ts-host
npm install p-limit@7.3.0

# No other changes needed. Existing dependencies are current.
```

## Integration Points with Existing Go/TS Code

### 1. Simulate-Only Guard Removal

| File | Location | Change |
|------|----------|--------|
| `lifecycle.ts` | Lines 268-275 | Remove guard that blocks non-simulated execution |
| `lifecycle.ts` | Line 424 | Change `const simulateWorkers = true` to `opts.simulateWorkers ?? false` |
| `host.ts` | Lines 308-312 | Remove `process.exit(1)` guard for lifecycle without `--simulate` |

### 2. Worker-to-Worker Spawning

| File | Location | Change |
|------|----------|--------|
| `worker-dispatch.ts` | After `dispatchSingleWorker()` | Check `result.spawns` (currently unused), queue sub-dispatches |
| `worker-dispatch.ts` | New `SpawnPool` class | Apply `p-limit` concurrency cap, Queen budget check |
| `claims-parser.ts` | Already works | `spawns` field already parsed from worker JSON output |
| `types.ts` | `WorkerResult` | Ensure `spawns` field is preserved in `toWorkerResults()` |

### 3. Confidence Loop Generalization

| File | Location | Change |
|------|----------|--------|
| New file `confidence-loop.ts` | TS host src | Extract loop pattern from `oracle-lifecycle.ts` |
| `oracle-lifecycle.ts` | Refactor | Use `ConfidenceLoop` instead of inline loop |
| `lifecycle.ts` | Step 2 (Build) and Step 3 (Continue) | Wrap in `ConfidenceLoop` when confidence target is set |

### 4. Hive Wisdom Injection

| File | Location | Change |
|------|----------|--------|
| `prompt-assembler.ts` | `renderContextCapsule()` | Add `callGoJSON(["hive-read"])` and inject into context |
| `go-bridge.ts` | No change | `callGoJSON` already supports arbitrary Go CLI commands |

## Summary: New vs. Existing

| Category | Count | Details |
|----------|-------|---------|
| New runtime dependencies | 1 | `p-limit@7.3.0` |
| New dev dependencies | 0 | |
| Existing deps to bump | 0 | All current versions are current |
| New TS source files | 1 | `confidence-loop.ts` (extracted from oracle-lifecycle.ts) |
| Modified TS source files | 4 | `lifecycle.ts`, `worker-dispatch.ts`, `prompt-assembler.ts`, `host.ts` |
| New Go changes | 0 | Go runtime already supports all required operations |

## Alternatives Considered

### Concurrency Control: `p-limit` vs `bottleneck` vs custom

| Criterion | `p-limit` 7.3.0 | `bottleneck` 2.19.5 | Custom semaphore |
|-----------|-----------------|---------------------|------------------|
| Bundle size | ~2 KB | ~35 KB | 0 KB |
| Dependencies | 0 | 4 | 0 |
| API complexity | 2 methods | 15+ methods | Custom |
| Active maintenance | Yes (2024) | Yes (2023) | N/A |
| Node >=20 | Yes | Yes | N/A |

**Verdict:** `p-limit`. One function, one purpose, zero dependencies. `bottleneck` offers priority queues and reservoirs that we do not need. A custom semaphore is trivial but `p-limit` is battle-tested and tiny.

### Iteration Loop: Extract class vs. Keep inline

The Oracle lifecycle has a working loop. Options:

| Approach | Pros | Cons |
|----------|------|------|
| Keep inline in oracle-lifecycle.ts | No refactoring risk | Duplicates loop logic for plan/build/continue |
| Extract `ConfidenceLoop` class | Reusable, testable, single source of truth | Refactoring risk (must not break Oracle) |

**Verdict:** Extract. The loop pattern is identical for Oracle, plan, build, and continue. Duplicating it four times guarantees drift.

## Sources

- npm registry verified (2026-05-18): `p-limit@7.3.0` (node >=20, 0 deps)
- Aether codebase analysis:
  - `.aether/ts-host/src/worker-dispatch.ts` -- real dispatch path (lines 237-307), `spawns` in claims
  - `.aether/ts-host/src/lifecycle.ts` -- simulate-only guard (lines 268-275)
  - `.aether/ts-host/src/host.ts` -- entry-point guard (lines 308-312)
  - `.aether/ts-host/src/oracle-lifecycle.ts` -- confidence loop template
  - `.aether/ts-host/src/prompt-assembler.ts` -- context assembly, hive integration point
  - `.aether/ts-host/src/platform-dispatcher.ts` -- `spawns` schema, AbortController usage
  - `.aether/ts-host/src/wave-orchestrator.ts` -- retry logic, wave grouping
  - `.aether/ts-host/src/types.ts` -- OracleStopConditions, WorkerResult, BuildDispatch
  - `.aether/ts-host/src/queen/orchestrator.ts` -- Queen spawn budget integration
  - `.aether/ts-host/src/go-bridge.ts` -- Go CLI bridge, completion file pattern
  - `cmd/hive.go` -- hive-store, hive-read, hive-promote implementations
  - `cmd/oracle_loop.go` -- Go-side oracle depth levels and iteration config
  - `cmd/queen_spawn_budget.go` -- Go-side spawn budget application
  - `.aether/ts-host/package.json` -- current dependency versions
