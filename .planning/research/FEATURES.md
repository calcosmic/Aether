# Feature Landscape: v1.21 Live Colony -- Production Worker Orchestration

**Domain:** Autonomous multi-agent colony framework -- production TypeScript host orchestration, worker-to-worker spawning, confidence-driven iteration, cross-colony wisdom reuse
**Researched:** 2026-05-18
**Confidence:** HIGH (verified against Aether source code in `.aether/ts-host/src/`, Go runtime `cmd/`, and established multi-agent patterns from Anthropic, LangGraph, CrewAI, Erlang/OTP)

## Executive Summary

The v1.21 milestone converts Aether's TypeScript host from a simulation-only smoke harness into the production control plane. The four target capabilities -- production TS host orchestration, worker-to-worker spawning, confidence-driven iteration, and hive wisdom reuse -- are not exotic new features. They are the standard expectations for any framework that claims autonomous multi-agent orchestration.

The critical finding is that most of the infrastructure already exists. Real worker dispatch code is written but gated behind `simulateWorkers !== true`. The Oracle lifecycle already proves the confidence-driven iteration pattern. The hive-read Go command already returns structured wisdom entries. The worker claims schema already includes a `spawns` field that the TS host currently ignores. The work is primarily removing guards, wiring existing pieces together, and proving end-to-end correctness -- not building new systems from scratch.

What makes this a research-intensive milestone is the gap between "code exists" and "code works in production." The pitfalls research (PITFALLS.md) documents 16 failure modes for the hybrid architecture. The most dangerous are Pitfall 1 (TS writes state directly), Pitfall 7 (Builder-Probe Lock bypass), and Pitfall 14 (placeholder code ships as real). Every feature below must be evaluated against these pitfalls.

## Table Stakes

Features that users expect from a multi-agent orchestration framework. Missing these means the framework feels incomplete or fake.

| Feature | Why Expected | Complexity | Existing Infrastructure | Notes |
|---------|-------------|------------|------------------------|-------|
| Real worker dispatch | A framework that only simulates workers is a prototype, not a product | Medium | `worker-dispatch.ts` has full real path; `platform-dispatcher.ts` detects platforms and spawns subprocesses; `prompt-assembler.ts` builds prompts; `claims-parser.ts` extracts results | Gated behind `simulateWorkers !== true` in lifecycle.ts line 268. Remove guard, add platform detection, harden error handling. |
| Platform CLI integration (Claude, OpenCode, Codex) | Users expect workers to run on their installed platform, not a hypothetical one | Low | `platform-dispatcher.ts` already detects available platforms, builds correct CLI args per platform, handles auth checks, and manages timeouts | Claude uses `-p --output-format json --agent`; OpenCode uses `run --agent`; Codex uses `--sandbox workspace-write exec --json`. All working in code. |
| Worker lifecycle events (spawn-log, spawn-complete) | Orchestration without observability is a black box | Low | `dispatchSingleWorker()` already calls `aether spawn-log` before dispatch and `aether spawn-complete` after. Error handling logs warnings but does not block. | Fully working for simulation; needs verification with real dispatch. |
| Wave-grouped parallel dispatch | Multi-worker orchestration must respect wave ordering (dependencies) within phases | Low | `wave-orchestrator.ts` groups by wave, dispatches in parallel via `Promise.all`, retry with exponential backoff | Default retry limit: 1. Default timeout: 600000ms (10 min). |
| Completion file -> Go finalizer flow | State mutation must go through Go for atomicity and validation | Low | `writeCompletionFile()` writes to tmpdir. Go finalizers (`build-finalize`, `plan-finalize`, `continue-finalize`) read and validate | Core boundary contract pattern. Already working. |
| Build manifest consumption | The TS host must read Go's build manifest and dispatch exactly the workers it specifies | Low | `BuildManifest` type in `types.ts` matches Go struct. `buildManifest.dispatches` provides the authoritative worker list | Must dispatch from manifest only, never re-derive or re-group workers (Pitfall 2). |
| Worker handoff between waves | Workers in later waves should know what earlier waves accomplished | Medium | `WorkerHandoff` type has `changed_files`, `commands_run`, `verification_status`, `known_failures`, `open_decisions`, `assumptions`, `next_worker_instructions`, `things_not_to_repeat`, `freshness` | Type exists; injection into later worker prompts needs verification. |
| Graceful degradation on platform unavailability | Framework should not crash if a platform CLI is missing; it should tell the user what is missing | Low | `detectAvailablePlatforms()` + `formatPlatformUnavailableMessage()` | Already working. Throws clear error listing available platforms. |

## Differentiators

Features that set Aether apart from other multi-agent frameworks. Not expected, but valued.

### D1. Worker-to-Worker Spawning (Budget-Capped)

**Value proposition:** Workers can dynamically request additional workers mid-build. A builder discovers a complex subtask and spawns a scout to research it. A watcher finds a potential security issue and spawns a gatekeeper to verify. This is hierarchical delegation with a hard budget cap to prevent runaway spawn chains.

**Why differentiating:** Most frameworks (LangGraph, AutoGen, CrewAI) support some form of hierarchical delegation, but they do not enforce a budget cap at the orchestrator level. CrewAI's hierarchical process can produce unpredictable delegation chains on novel inputs. LangGraph's subgraph spawning is powerful but requires explicit graph definition. Aether's approach is unique in that: (1) the Queen sets the budget in Go before the build starts, (2) the TS host enforces the budget at dispatch time, (3) child spawns are recorded via the same `spawn-log`/`spawn-complete` lifecycle as manifest workers, and (4) child results attach to the parent's handoff for downstream visibility.

**Complexity:** Medium-High

**Existing infrastructure:** `workerClaimsSchema` includes `spawns` field (array of strings). `claims-parser.ts` already parses `spawns` from worker output. `QueenSpawnBudget` type in `types.ts` has `max_workers`, `worker_count`, `pruned_workers`. The `spawn-log` and `spawn-complete` Go commands exist.

**What is new:** A `worker-spawner.ts` module that: validates spawn requests against remaining budget, dispatches child workers via the same `platform-dispatcher.ts` path, records child results via Go commands, and attaches child results to parent's `WorkerResult`.

**Confidence:** MEDIUM -- the infrastructure exists but mid-build child spawn tracking against the Queen's budget is new behavior. Need to verify Go handles child `spawn-log` entries correctly in the spawn tree.

---

### D2. Confidence-Driven Build Iteration

**Value proposition:** The build loop iterates until quality targets are met, not just once. If the first build attempt produces code that fails verification gates, the coordinator re-dispatches with feedback from the failed gates. This is the Evaluator-Optimizer pattern from Anthropic's research, applied to the build lifecycle.

**Why differentiating:** The Oracle lifecycle already proves this pattern works in Aether for research. Generalizing it to builds means: a builder writes code, verification gates evaluate it, and if confidence is below target, the builder gets another chance with specific feedback. Most frameworks run workers once and call it done. The iteration loop makes Aether's builds self-correcting.

**Complexity:** Medium

**Existing infrastructure:** `oracle-lifecycle.ts` implements the full RALF loop: `--plan-only` -> dispatch -> finalize -> evaluate stop conditions -> loop. `OracleIterationManifest` has `confidence_target`, `max_iterations`, `current_iteration`. `OracleStopConditions` has `confidence_met`, `max_iterations_met`, `manual_stop`, `no_progress`. Go's `build-finalize` returns gate results that can serve as confidence signals.

**What is new:** A `build-coordinator.ts` module that mirrors the Oracle lifecycle pattern but for builds. Key difference: Oracle loops over `oracle-iterate --plan-only`, while this loops over `build <N> --plan-only`. The coordinator evaluates gate results (not just a single confidence number) to decide whether to iterate.

**Confidence:** MEDIUM -- Oracle lifecycle proves the pattern. The risk is in determining what "confidence" means for builds (gate pass rate? test coverage? all gates passed?) and whether Go's build finalizer returns sufficient metrics for the TS host to make the decision.

---

### D3. Hive Wisdom Reuse Proof (Cross-Colony Knowledge Transfer)

**Value proposition:** A colony working on a Go project benefits from wisdom learned by a different colony working on a different Go project. When colony B starts, it receives relevant patterns from colony A's experience: "Go error handling uses `errors.Is` not `==` for unwrapping" or "this project uses testify not testing.T for mocks." Colony B starts faster and makes fewer beginner mistakes.

**Why differentiating:** Cross-colony knowledge transfer is rare in multi-agent frameworks. Most frameworks are single-session or single-project. Aether's Hive Brain already stores wisdom with domain tags, confidence scores, and source repo attribution. The "proof" is demonstrating that the injection actually helps -- colony B receives wisdom and its workers reference it in their output.

**Complexity:** Low

**Existing infrastructure:** `hive-read` Go command with `--domain` and `--threshold` flags. `hive-promote` at seal promotes high-confidence instincts. Multi-repo confidence boosting (2 repos = 0.70, 4+ = 0.95). 200-entry cap with LRU eviction. Colony registry stores domain tags per repo.

**What is new:** A `hive-injector.ts` module that calls `aether hive-read` before dispatching workers and formats the results for prompt injection. The formatted wisdom goes into the context capsule assembled by `prompt-assembler.ts`. This is purely additive -- workers without hive wisdom work identically.

**Confidence:** HIGH -- `hive-read` exists and returns structured data. The integration is straightforward. The risk is proving measurable impact (colony B is measurably better), not the wiring itself.

---

### D4. Queen Workflow Pattern Selection

**Value proposition:** The Queen autonomously selects the execution strategy (wave grouping, parallelism, verification depth) based on the build manifest's characteristics. Low-risk work gets fast execution; high-risk work gets thorough verification.

**Complexity:** Low (already exists, needs production validation)

**Existing infrastructure:** `queen/orchestrator.ts` derives workflow pattern from manifest metadata. Builder-Probe Lock applied automatically. Midden threshold check before dispatch. Workflow patterns influence wave dispatch behavior.

**Notes:** This shipped conceptually in v1.20 but has never run with real workers. The production validation is the new work.

---

### D5. Ceremony-Preserving Production Dispatch

**Value proposition:** Real worker dispatch produces the same ceremony output (spawn notifications, wave markers, caste identity lines) as simulation. Users see the same rich experience whether running simulation or production.

**Complexity:** Medium

**Existing infrastructure:** `ceremony-adapter.ts` wraps Go ceremony commands. `narrator.ts` renders events. `event-bridge.ts` streams events from Go JSONL. All ceremony events (`ceremony.build.spawn`, `ceremony.build.wave.start`, etc.) defined in `CEREMONY_TOPICS`.

**Notes:** Ceremony already works in simulation mode. Production dispatch must emit the same events at the same lifecycle points.

## Anti-Features

Features to explicitly NOT build.

| Anti-Feature | Why Avoid | What to Do Instead |
|---|---|---|
| Recursive unlimited worker spawning | A worker spawns a child, which spawns a grandchild, creating an unbounded tree. Exhausts API rate limits and runs forever. | Enforce Queen spawn budget across all levels. Count manifest workers AND child spawns against the same budget. Hard cap on total workers per build. |
| TS host computes its own confidence score | Duplicates Go's verification logic. Creates drift between TS and Go confidence calculations. | TS host reads confidence metrics from Go finalizer result. If Go does not return them, use simple heuristic (all gates passed = confidence met). |
| Hive wisdom injection blocks when hive is empty | First-time colonies and repos with no prior colonies should work fine. | Treat hive wisdom as optional enrichment. Log warning when no wisdom available, proceed without it. |
| Workers spawn other workers without manifest awareness | A spawned child worker could spawn its own children, creating a tree the orchestrator cannot track. | Only one level of worker-to-worker spawning in v1.21. Child workers cannot spawn grandchildren. This is a budget and tracking simplification, not a fundamental limitation. |
| Silent auto-recovery without ceremony events | Users lose visibility into what the system is doing. | Emit ceremony events for every recovery action, spawn, retry, and budget decision. |
| Confidence iteration without phase state check | Re-building an already-completed phase corrupts state. | Before each iteration, read current phase state from Go manifest. Only re-dispatch if phase is still in a buildable state. |
| Real dispatch without platform detection | Attempting real dispatch when no platform CLI is installed produces obscure errors. | Always run `detectAvailablePlatforms()` before real dispatch. Throw `formatPlatformUnavailableMessage()` on failure. |
| Simulation code in production path | v1.20 explicitly gated simulation behind `--simulate`. v1.21 must not regress. | Remove simulation guards only after real dispatch is proven. Keep `--simulate` as an explicit opt-in for testing. |

## Feature Dependencies

```
Production TS host (remove simulation guard)
  -> Real worker dispatch (already coded, just gated)
  -> Platform CLI integration (already coded)
  -> Wave-grouped dispatch (already coded)
  -> Worker lifecycle events (already coded)

Worker-to-Worker Spawning
  -> Production TS host (must dispatch real workers first)
  -> Queen spawn budget enforcement (QueenSpawnBudget type exists)
  -> spawn-log/spawn-complete Go commands (exist)

Confidence-Driven Build Iteration
  -> Production TS host (must dispatch real workers)
  -> Worker-to-Worker Spawning (optional but builds should test with spawns)
  -> Oracle lifecycle pattern (proven in oracle-lifecycle.ts)

Hive Wisdom Reuse
  -> Production TS host (injection happens during dispatch)
  -> hive-read Go command (exists)
  -> Colony registry domain tags (exist)
  -> prompt-assembler context capsule (exists)
```

**Key insight:** Hive wisdom reuse has the fewest dependencies and can ship first. Worker-to-worker spawning depends on production dispatch being proven. Confidence-driven iteration is the most integrated feature and should ship last.

## MVP Recommendation

**Phase 1 -- Production Foundation (lowest risk):**
1. Production TS host -- remove simulation guard in `lifecycle.ts`, wire platform detection, prove end-to-end plan -> build (real) -> continue (real) lifecycle
2. Hive wisdom reuse -- `hive-injector.ts` calls `hive-read`, formats for prompt injection, proves colony B receives colony A's wisdom

**Rationale:** These two features prove the core transition from simulation to production. Hive wisdom reuse is independent and simple. Production TS host is the prerequisite for everything else. Together they demonstrate that the hybrid Go/TS architecture actually works with real workers.

**Phase 2 -- Dynamic Dispatch:**
3. Worker-to-worker spawning -- `worker-spawner.ts` with budget validation, child dispatch via existing platform path, spawn-log/spawn-complete recording

**Rationale:** Once production dispatch is proven, adding dynamic spawn requests is a bounded extension. The `spawns` field already exists in the claims schema; the TS host just needs to act on it.

**Phase 3 -- Self-Correcting Builds:**
4. Confidence-driven iteration -- `build-coordinator.ts` looping over `build --plan-only` until quality targets are met

**Rationale:** This is the payoff feature. It requires real dispatch (Phase 1) to be meaningful and benefits from worker-to-worker spawning (Phase 2) for complex builds. The Oracle lifecycle proves the pattern.

**Defer:**
- Multi-level spawn trees (grandchildren, great-grandchildren) -- one level is sufficient for v1.21; deeper trees need more sophisticated budget tracking and deadlock detection
- Cross-colony ledger sharing -- explicitly listed as a non-goal in PROJECT.md
- Learned confidence thresholds (adjusting targets based on colony history) -- requires multiple colonies' worth of data
- User-defined spawn policies (letting users define which castes can spawn which castes) -- YAGNI until one-level spawning is proven

## Complexity Summary

| Feature | Complexity | Risk | Existing Infrastructure | New Code Required |
|---|---|---|---|---|
| Production TS host | Medium | High (lifecycle.ts guard removal) | Complete dispatch pipeline | Guard removal, platform detection hardening, error handling |
| Worker-to-worker spawning | Medium-High | Medium (budget enforcement) | Claims schema with spawns, spawn-log/complete, QueenSpawnBudget type | `worker-spawner.ts` module (~100-150 lines) |
| Confidence-driven iteration | Medium | Medium (confidence metric definition) | Oracle lifecycle pattern, Go build finalizer gate results | `build-coordinator.ts` module (~80-120 lines) |
| Hive wisdom reuse | Low | Low (graceful degradation) | hive-read command, domain tags, prompt-assembler context capsule | `hive-injector.ts` module (~40-60 lines) |
| Ceremony in production | Medium | Low (already works in simulation) | ceremony-adapter, narrator, event-bridge | Integration verification, not new code |

## Existing Aether Infrastructure Mapped to v1.21 Features

This section maps what already exists to what v1.21 needs, to make clear where the gaps are.

### Worker Dispatch Pipeline (Production TS Host)

| Component | File | Status | Gap |
|---|---|---|---|
| Platform detection | `platform-dispatcher.ts` | Working | None |
| Platform auth checks | `platform-dispatcher.ts` | Working | None |
| CLI arg building (Claude) | `platform-dispatcher.ts` | Working | None |
| CLI arg building (OpenCode) | `platform-dispatcher.ts` | Working | None |
| CLI arg building (Codex) | `platform-dispatcher.ts` | Working | None |
| Subprocess spawning with timeout | `platform-dispatcher.ts` | Working | None |
| Prompt assembly | `prompt-assembler.ts` | Working | Hive wisdom section needed |
| Claims parsing | `claims-parser.ts` | Working | None |
| Spawn-log recording | `worker-dispatch.ts` | Working | None |
| Spawn-complete recording | `worker-dispatch.ts` | Working | None |
| Wave grouping | `wave-orchestrator.ts` | Working | None |
| Parallel dispatch (Promise.all) | `wave-orchestrator.ts` | Working | None |
| Retry with backoff | `wave-orchestrator.ts` | Working | None |
| Simulation guard | `lifecycle.ts:268` | **BLOCKING** | Must be removed/replaced |
| Platform unavailability handling | `worker-dispatch.ts` | Working | None |
| Worker handoff injection | `prompt-assembler.ts` | Partial | Needs verification with real dispatch |

### Confidence Iteration (Existing Pattern in Oracle)

| Component | File | Status | Gap |
|---|---|---|---|
| Loop: plan-only -> dispatch -> finalize | `oracle-lifecycle.ts` | Working | Needs to be generalized to build |
| Stop conditions (max iterations, confidence) | `oracle-lifecycle.ts` | Working | Build needs different stop conditions |
| Statelessness (re-read from Go each iteration) | `oracle-lifecycle.ts` | Working | Pattern carries over directly |
| Completion file writing | `go-bridge.ts` | Working | None |
| Go finalizer confidence return | `oracle-iterate-finalize` | Working | Build finalizer needs equivalent metrics |

### Hive Wisdom (Existing Go Commands)

| Component | File | Status | Gap |
|---|---|---|---|
| `hive-read` with domain filtering | `cmd/hive.go` | Working | None |
| `hive-store` with deduplication | `cmd/hive.go` | Working | None |
| Domain tags in colony registry | `cmd/registry.go` | Working | None |
| Multi-repo confidence boosting | `cmd/hive.go` | Working | None |
| 200-entry cap with LRU eviction | `cmd/hive.go` | Working | None |
| TS-side hive call | `hive-injector.ts` | **NEW** | Must be written |
| Prompt injection of wisdom | `prompt-assembler.ts` | Partial | Needs hiveWisdomSection config |

## Sources

| Source | Type | Confidence |
|---|---|---|
| Aether source: `.aether/ts-host/src/worker-dispatch.ts` | Source code analysis | HIGH |
| Aether source: `.aether/ts-host/src/platform-dispatcher.ts` | Source code analysis | HIGH |
| Aether source: `.aether/ts-host/src/oracle-lifecycle.ts` | Source code analysis | HIGH |
| Aether source: `.aether/ts-host/src/wave-orchestrator.ts` | Source code analysis | HIGH |
| Aether source: `.aether/ts-host/src/queen/orchestrator.ts` | Source code analysis | HIGH |
| Aether source: `.aether/ts-host/src/lifecycle.ts` | Source code analysis | HIGH |
| Aether source: `.aether/ts-host/src/types.ts` | Source code analysis | HIGH |
| Aether source: `.aether/ts-host/src/prompt-assembler.ts` | Source code analysis | HIGH |
| Aether source: `.aether/ts-host/src/claims-parser.ts` | Source code analysis | HIGH |
| Aether source: `cmd/hive.go` | Source code analysis | HIGH |
| `.planning/research/ARCHITECTURE.md` | Architecture research (v1.21) | HIGH |
| `.planning/research/PITFALLS.md` | Pitfall research (v1.17) | HIGH |
| `.planning/PROJECT.md` | Project context | HIGH |
| Anthropic: Building Effective Agents (5 agent patterns) | Published research | MEDIUM |
| LangGraph 1.0 multi-agent patterns | Official docs | MEDIUM |
| CrewAI hierarchical process | Official docs | MEDIUM |
| Erlang/OTP supervisor behaviour | Official docs | HIGH |
