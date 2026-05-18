# Project Research Summary

**Project:** Aether v1.21 Live Colony
**Domain:** CLI colony framework -- hybrid Go runtime + TypeScript orchestration host
**Researched:** 2026-05-18
**Confidence:** HIGH

## Executive Summary

The v1.21 milestone converts Aether's TypeScript host from a simulation-only smoke harness into the production control plane for multi-agent colony orchestration. The four target capabilities are: real worker dispatch (removing the simulate-only guard in `lifecycle.ts`), worker-to-worker spawning (workers requesting sub-workers mid-build), confidence-driven iteration (looping plan/build/continue until quality targets are met), and hive wisdom reuse verification (proving colony B benefits from colony A's learnings). The critical finding across all four research files is that nearly all the code already exists. The `dispatchRealWorker()` path is fully implemented. The `spawns` field exists in the worker claims schema. The Oracle lifecycle already demonstrates the confidence-driven iteration pattern. The Go runtime already owns all hive operations. What is missing is not new dependencies or architecture, but wiring -- removing guards, consuming existing fields, generalizing existing patterns, and calling existing Go commands.

The work is primarily integration, not greenfield development. Only one new runtime dependency is recommended (`p-limit` for concurrency control), and only three new TS source files are needed (`hive-injector.ts`, `worker-spawner.ts`, `build-coordinator.ts`). Zero new Go-side development is required -- all Go commands (`hive-read`, `spawn-log`, `spawn-complete`, `build-finalize`) already exist and return structured JSON. The highest-risk change is removing the simulation guard in `lifecycle.ts` line 268, which blocks all production dispatch. The most dangerous pitfall is worker-to-worker spawning creating unbounded fan-out if not budget-capped at the orchestrator level.

## Key Findings

### Recommended Stack

The existing stack is sufficient. Only one new dependency is recommended.

**Core technologies (existing, unchanged):**
- Go 1.24 + Cobra CLI: authoritative runtime, state mutations, finalizers, verification
- TypeScript (Node >=20): orchestration control plane, platform dispatch, ceremony rendering
- chalk/boxen/figlet/ora/cli-progress/log-update: rendering stack (v1.17, no bumps needed)

**New addition:**
- `p-limit@7.3.0`: concurrency cap for dynamic worker spawning -- 2KB, zero dependencies, ESM-native. Without it, worker-to-worker spawning creates unbounded fan-out. Rejected `bottleneck` (10x larger, unnecessary features) and custom semaphores (reinventing the wheel).

**Explicitly NOT adding:**
- `execa`/`zx`: Go bridge deliberately uses `child_process` for safety and JSON envelope control
- `bullmq`/`rabbitmq`: workers are short-lived CLI subprocesses, not long-running services
- Redis/SQLite for hive storage: 200-entry JSON file with file locking is sufficient
- Vector embeddings for hive search: domain tags + confidence threshold is correct for this scale

### Expected Features

**Must have (table stakes):**
- Real worker dispatch via platform CLIs (Claude, OpenCode, Codex) -- framework that only simulates is a prototype
- Wave-grouped parallel dispatch with dependency ordering -- multi-worker orchestration without wave ordering breaks causality
- Completion file to Go finalizer flow -- state mutation must go through Go for atomicity
- Worker lifecycle events (spawn-log, spawn-complete) -- orchestration without observability is a black box
- Graceful degradation on platform unavailability -- framework should not crash when a platform CLI is missing

**Should have (competitive differentiators):**
- Worker-to-worker spawning (budget-capped) -- hierarchical delegation is standard in LangGraph/CrewAI but Aether enforces an orchestrator-level budget cap, which is unique
- Confidence-driven build iteration -- Evaluator-Optimizer pattern from Anthropic's research; most frameworks run workers once
- Hive wisdom reuse proof -- cross-colony knowledge transfer is rare; Aether's Hive Brain already has domain tags, confidence scores, and multi-repo boosting
- Ceremony-preserving production dispatch -- real workers produce the same rich ceremony output as simulation

**Defer to v2+:**
- Multi-level spawn trees (grandchildren, great-grandchildren) -- one level is sufficient; deeper trees need deadlock detection
- Cross-colony ledger sharing -- explicitly a non-goal in PROJECT.md
- Learned confidence thresholds -- requires multiple colonies' worth of data
- User-defined spawn policies -- YAGNI until one-level spawning is proven

### Architecture Approach

The hybrid Go/TS architecture is already established and well-documented. Three new modules integrate at the existing boundary. No architectural reorganization is needed.

**New components:**
1. `hive-injector.ts` -- calls `aether hive-read`, formats wisdom for prompt injection into `prompt-assembler.ts` context capsule. Read-only on the hub. Purely additive; workers without hive wisdom work identically. (~40-60 lines)
2. `worker-spawner.ts` -- validates spawn requests against Queen spawn budget, dispatches child workers via existing `platform-dispatcher.ts`, records results via Go `spawn-log`/`spawn-complete`. Attaches child results to parent's handoff. (~100-150 lines)
3. `build-coordinator.ts` -- confidence-driven plan/build iteration loop, modeled on the proven Oracle lifecycle pattern (`oracle-lifecycle.ts`). Loops `build --plan-only` -> dispatch -> finalize -> evaluate confidence. Hard cap at 3 iterations default. (~80-120 lines)

**Key patterns to follow:**
- **Manifest-Only-Then-Finalize**: every workflow calls Go `--plan-only` for manifest, dispatches workers, writes completion to tmpdir, calls Go finalizer. This is the core boundary contract.
- **Budget-Capped Spawn Tree**: Queen spawn budget caps total workers including child spawns. No exceptions.
- **Graceful Degradation**: if `hive-read` fails, proceed without hive wisdom. Log warning, never block.

### Critical Pitfalls

1. **Simulation guard silently blocks production dispatch** -- `lifecycle.ts` lines 268-275 and line 424 both contain guards that prevent non-simulated execution. Remove both in Phase 1. Add a `--simulate` opt-in flag for testing. Audit all `simulateWorkers` references.
2. **Worker claims parsing fails on real platform output** -- real platform output contains markdown wrappers, thinking traces, and API errors mixed with claims JSON. Add pre-parse classification for auth failures, rate limits, and timeouts before attempting JSON extraction. Write diagnostic artifacts to tmpdir.
3. **Worker-to-worker spawning creates unbound fan-out** -- without a hard cap, 1 worker becomes 2 becomes 4 becomes 8. Enforce max spawn depth of 2 (no grandchildren). Route all spawn requests through Go manifest for budget enforcement. Sub-workers count against the same Queen budget as manifest workers.
4. **Confidence iteration loop never converges** -- hard cap at 3 iterations default. Track confidence delta between iterations; stop early if delta is below 5% for two consecutive iterations. Cumulative budget across iterations (not per-iteration). Confidence must come from Go finalizer metrics, not worker self-reports.
5. **Hive wisdom injects stale or irrelevant advice** -- expand domain tags to include technology stack, not just broad categories. Apply relevance discount for partial domain matches. First-time colonies without hive wisdom must work identically to colonies with it.

## Implications for Roadmap

Based on the combined research, the recommended phase structure is:

### Phase 1: Production Foundation + Hive Wisdom
**Rationale:** Production dispatch is the prerequisite for everything else. Hive wisdom reuse has the fewest dependencies and can ship alongside the foundation work, proving both the core transition and the simplest new capability. The features are independent enough to parallelize within a single phase.
**Delivers:** Real worker dispatch end-to-end (plan -> build real -> continue real lifecycle). Hive wisdom injection into worker context.
**Addresses:** Table stakes (real dispatch, lifecycle events, graceful degradation). Hive wisdom reuse proof.
**Avoids:** Simulation guard blockage (Pitfall 1), claims parsing failures (Pitfall 2), completion file leaks (Pitfall 8), platform arg drift (Pitfall 9), recovery actions not executed (Pitfall 10).
**Key changes:** Remove simulation guards in `lifecycle.ts` (lines 268-275, 424) and `host.ts` (lines 308-312). Add pre-parse error classification in `claims-parser.ts`. Create `hive-injector.ts` (~40-60 lines). Modify `prompt-assembler.ts` to include hive wisdom section. Add lifecycle-level AbortController for graceful shutdown. Install `p-limit@7.3.0`.

### Phase 2: Worker-to-Worker Spawning
**Rationale:** Once production dispatch is proven, adding dynamic spawn requests is a bounded extension. The `spawns` field already exists in the claims schema; the TS host just needs to act on it with budget enforcement.
**Delivers:** Workers can request additional workers mid-build. Child workers are dispatched via the same platform path, recorded via Go spawn-log/spawn-complete, and results attached to parent handoff.
**Addresses:** Worker-to-worker spawning (differentiator). Budget-capped spawn tree pattern.
**Avoids:** Unbound fan-out (Pitfall 3), finalizer rejecting unknown workers (Pitfall 6), parallel file conflicts (Pitfall 7).
**Key changes:** Create `worker-spawner.ts` (~100-150 lines). Modify `worker-dispatch.ts` to check `claims.spawns` post-dispatch. Add `child_results` to `DispatchResult` type. Implement `p-limit` concurrency cap.

### Phase 3: Confidence-Driven Build Iteration
**Rationale:** This is the payoff feature. It requires real dispatch (Phase 1) to be meaningful and benefits from worker-to-worker spawning (Phase 2) for complex builds. The Oracle lifecycle already proves the pattern.
**Delivers:** Build loop iterates until quality gates pass. If first attempt fails verification, re-dispatch with specific feedback. Hard cap prevents runaway iteration.
**Addresses:** Confidence-driven iteration (differentiator). Self-correcting builds.
**Avoids:** Loop never converging (Pitfall 4), confidence target too high (Pitfall 12), re-dispatching completed phases (Anti-Pattern 5).
**Key changes:** Create `build-coordinator.ts` (~80-120 lines). Wire into `host.ts` when `--target` flag is present. Implement diminishing returns detection and cumulative cross-iteration budget.

### Phase 4: Hardening and Validation
**Rationale:** After the three feature phases, a dedicated hardening phase ensures production readiness. This includes end-to-end integration tests, golden parity tests, and addressing the moderate/minor pitfalls that accumulate during development.
**Delivers:** Full end-to-end test coverage. Golden ceremony parity tests. Platform version detection. Temp file cleanup. Production readiness gate.
**Addresses:** Remaining pitfalls (prompt size limits, concurrent hive writes, platform version detection). Production confidence.
**Avoids:** Shipping placeholder code, shipping without ceremony parity, regression risks.

### Phase Ordering Rationale

- Phase 1 unlocks everything: the simulation guard is the single biggest blocker. Removing it and proving real dispatch works is the foundation.
- Hive wisdom is bundled with Phase 1 because it is independent, low-risk, and proves the TS host can call Go commands and inject results into prompts -- a pattern used by all subsequent phases.
- Phase 2 (spawning) depends on Phase 1 being stable because it extends the dispatch path. Trying to add spawning before proving basic dispatch would make failures harder to diagnose.
- Phase 3 (confidence iteration) is deliberately last among features because it is the most complex integration (loop over build, evaluate, re-dispatch) and benefits from both real dispatch and spawning being proven.
- Phase 4 (hardening) is separate because hardening concerns (temp cleanup, platform version detection, prompt size checks) cut across all feature phases and are best addressed holistically.

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 1:** Moderate -- claims parser error classification needs investigation into what real platform error payloads look like (auth failures, rate limits). The known-issues doc has one documented instance. Need to inventory error patterns across all three platforms.
- **Phase 2:** Low-Medium -- Go's `spawn-log`/`spawn-complete` need to be verified for mid-build child worker entries. The Go spawn tree may need minor adjustments to accept children not in the original manifest.
- **Phase 3:** Medium -- the definition of "confidence" for builds is unclear. Oracle uses a single confidence number. Builds have multiple gate results (test pass rate, coverage, quality score). Need to define the confidence aggregation function.

Phases with standard patterns (skip research-phase):
- **Phase 4:** Well-understood hardening concerns. Standard test/validate/ship pattern.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Only one new dependency (`p-limit`). All existing deps verified current. Based on npm registry and package.json analysis. |
| Features | HIGH | All four capabilities grounded in existing code. Worker dispatch pipeline complete. Oracle lifecycle proves iteration pattern. Hive commands exist. `spawns` field in schema. |
| Architecture | HIGH | Three new modules follow established patterns. Boundary contract preserved by all new code. All Go commands already exist. Based on direct source analysis of 15+ TS modules and 5+ Go files. |
| Pitfalls | HIGH | 13 pitfalls grounded in direct code inspection with specific line numbers. Prevention strategies are concrete and testable. Based on known issues documentation and codebase analysis. |

**Overall confidence:** HIGH

### Gaps to Address

- **Confidence metric definition for builds (Phase 3):** Oracle uses a single confidence number from Go. Build finalizer returns gate results (pass/fail per gate). The aggregation function from multiple gates to a single "build confidence" needs definition during Phase 3 planning. If Go does not return usable confidence metrics, fallback to simple heuristic (all gates passed = confidence met).

- **Real platform error payload inventory (Phase 1):** Claims parser error classification needs to know what auth failures, rate limits, and timeout payloads look like across Claude Code, OpenCode, and Codex. One known instance documented. Need to test or research error formats.

- **Go spawn-tree compatibility for child workers (Phase 2):** Go's `spawn-log`/`spawn-complete` were designed for manifest workers. Need to verify they handle children spawned mid-build by the TS host (workers not in the original manifest). May need a minor Go-side change or a new `--parent` flag.

- **Domain tag granularity for hive wisdom (Phase 1/4):** Current domain tags are coarse (`["web", "api"]`). Hive wisdom injection needs technology-specific tags (`["go", "cli", "cobra"]`) to avoid irrelevant cross-colony advice. Tag expansion is a Go-side change to the colony registry and `hive-read` filtering.

## Sources

### Primary (HIGH confidence)
- Aether source: `.aether/ts-host/src/lifecycle.ts` -- simulation guards, lifecycle orchestration
- Aether source: `.aether/ts-host/src/worker-dispatch.ts` -- real dispatch path, spawn lifecycle
- Aether source: `.aether/ts-host/src/platform-dispatcher.ts` -- platform CLI args, claims schema
- Aether source: `.aether/ts-host/src/oracle-lifecycle.ts` -- confidence iteration pattern (template)
- Aether source: `.aether/ts-host/src/claims-parser.ts` -- worker claims extraction
- Aether source: `.aether/ts-host/src/prompt-assembler.ts` -- context assembly
- Aether source: `.aether/ts-host/src/go-bridge.ts` -- Go CLI bridge, boundary enforcement
- Aether source: `.aether/ts-host/src/wave-orchestrator.ts` -- parallel wave dispatch
- Aether source: `.aether/ts-host/src/queen/orchestrator.ts` -- Queen orchestration
- Go source: `cmd/hive.go` -- hive-read, hive-store, hive-promote
- Go source: `cmd/queen_spawn_budget.go` -- spawn budget calculation
- `.planning/PROJECT.md` -- project context and milestone history

### Secondary (MEDIUM confidence)
- Anthropic: Building Effective Agents (5 agent patterns) -- confidence iteration pattern validation
- LangGraph 1.0 multi-agent patterns -- hierarchical delegation comparison
- CrewAI hierarchical process -- spawn budget comparison
- Erlang/OTP supervisor behaviour -- budget-capped spawn tree validation

---
*Research completed: 2026-05-18*
*Ready for roadmap: yes*
