# Feature Landscape: Aether v1.14 Queen Authority

**Domain:** Autonomous queen coordination -- auto-recovery, smart gating, output filtering, wave coordination
**Researched:** 2026-05-03
**Confidence:** HIGH (findings verified against source code in `cmd/`, `pkg/colony/`, `pkg/codex/`; ecosystem patterns verified against official docs and published research)

## Executive Summary

Queen Authority transforms the queen from narrator into autonomous coordinator. The core problem: Aether's 11 continue gates, 4-tier worker escalation (retry -> reassign -> queen reassign -> user escalation), wave dispatch, and Fixer caste all exist as infrastructure, but they require manual triggering. Workers stall, gates block, and the user babysits every phase transition.

The research reveals a clear consensus across multi-agent frameworks (LangGraph, AutoGen, CrewAI, Google ADK, Erlang/OTP) on what autonomous supervisors must do: detect failures, classify severity, apply recovery strategies with bounded retry, and escalate only when genuinely stuck. The differentiator is not adding exotic features -- it is wiring existing Aether infrastructure together with a severity-classified decision loop that runs autonomously within well-defined boundaries.

The anti-feature insight is equally clear: autonomous supervisors must NOT silently skip failures, make irreversible state mutations without audit trails, or attempt to recover from fundamentally broken conditions (corrupted state, dependency conflicts). Erlang/OTP's principle is instructive: if MaxR restarts happen within MaxT seconds, the supervisor itself terminates rather than entering an infinite recovery loop.

## Feature Categories

### Category A: Queen Auto-Recovery

How the queen detects, classifies, and recovers from worker and gate failures without human intervention.

#### A1. Failure Classification Engine

**Why expected:** Every autonomous supervisor must distinguish between recoverable and non-recoverable failures before acting. Without classification, the queen either over-reacts (escalating trivial issues) or under-reacts (retrying fundamentally broken tasks). This is table stakes in Erlang/OTP (permanent vs transient vs temporary errors), LangGraph (retryable vs non-retryable exceptions), and production supervision trees.

**Current state:** Aether has `CircuitBreaker` (per-worker consecutive failure tracking with threshold), `GateCheckResult` (per-gate status/detail/fixHint/recoveryOptions), and `gateRecoveryTemplates` (manual recovery instructions). But there is no severity classification -- every gate failure is treated equally, and the Fixer dispatches with the same `propose` mode regardless of whether the failure is "a test assertion name changed" or "the entire build is broken."

**Expected behavior:**
1. Each failure gets a severity classification: `recoverable` (auto-fixable, low risk), `requires-attempt` (may be fixable, needs one try), `blocking` (genuinely stuck, needs human)
2. Classification signals come from gate metadata already present: `FixHint` presence (recoverable), `RecoveryOptions` content (requires-attempt), gate name itself (some gates like `spawn_gate` are inherently blocking)
3. Classification is deterministic and auditable -- logged to COLONY_STATE.json, not guessed by LLM
4. The queen uses classification to choose recovery strategy, not to decide *whether* to recover

**Complexity:** Medium -- classification logic is rule-based, but getting the rules right requires understanding each gate's failure semantics

**Dependencies:** Existing `GateCheckResult`, `gateRecoveryTemplates`, `CircuitBreaker`

**Confidence:** HIGH -- pattern is well-established in Erlang/OTP, LangGraph, and production systems

---

#### A2. Bounded Auto-Retry with Exponential Backoff

**Why expected:** When a worker fails for a transient reason (timeout, race condition, flaky test), the supervisor should retry automatically. This is the most basic recovery pattern in every framework: Erlang's `one_for_one` restart strategy, LangGraph's `RetryPolicy(max_attempts=3, backoff_factor=2.0)`, AutoGen's typed state transitions with retry logic.

**Current state:** `CircuitBreaker` exists but only *tracks* failures -- it does not retry. `findSameCastePeer` can redistribute to a peer, but only after the breaker trips (3 consecutive failures). The queen has no retry mechanism at all. Retry currently requires the user to run `/ant-build` or `/ant-continue` again manually.

**Expected behavior:**
1. For `recoverable`-classified failures: queen retries the same worker once, with a brief delay (configurable, default 10s)
2. For `requires-attempt`-classified failures: queen dispatches Fixer in `propose` mode (current default behavior)
3. Retry count is bounded per the Erlang model: MaxR retries within MaxT seconds, then escalate. Suggested defaults: 2 retries within 120 seconds
4. Each retry increments `RetryCount` on the `GateCheckResult` (field already exists, currently unused)
5. Retry events are emitted to the ceremony event bus for visibility

**Complexity:** Low -- the infrastructure exists, this is wiring it together

**Dependencies:** `CircuitBreaker`, `GateCheckResult.RetryCount`, ceremony event bus

**Confidence:** HIGH -- implementation is straightforward given existing infrastructure

---

#### A3. Peer Redistribution on Worker Failure

**Why expected:** When a specific worker is failing (circuit tripped), redistributing its task to a same-caste peer is a proven pattern. Erlang calls this "restart with a different worker." Aether already has `findSameCastePeer` -- it just is not called automatically.

**Current state:** `findSameCastePeer` exists and finds non-tripped same-caste peers. Circuit breaker trip events are emitted but no redistribution occurs. The user must manually redistribute.

**Expected behavior:**
1. When CircuitBreaker trips for a worker, queen automatically checks for a same-caste peer via `findSameCastePeer`
2. If a peer exists: redistribute the failed task, emit `emitCircuitBreakerRedistributed` (already implemented), mark original worker as `superseded`
3. If no peer exists: mark task as `blocked`, escalate to user with clear explanation
4. Peer redistribution counts against the retry budget (MaxR/MaxT)

**Complexity:** Low -- `findSameCastePeer` and event emission already exist

**Dependencies:** `CircuitBreaker`, `findSameCastePeer`, `emitCircuitBreakerRedistributed`

**Confidence:** HIGH -- infrastructure complete, just needs orchestration

---

#### A4. Queen-Driven Fixer Dispatch

**Why expected:** The Fixer caste exists (`dispatchFixer` with full/propose/advise modes, attempt cap, circuit breaker integration) but requires manual triggering via `/ant-unblock --dispatch`. An autonomous queen should dispatch the Fixer automatically when `requires-attempt` failures are detected.

**Current state:** `dispatchFixer` validates mode, checks circuit breaker, checks attempt cap, reads gate results, builds fix context, and outputs dispatch instruction JSON. `resolveFixedGates` updates gate results for addressed gates. `recordFixerFailure` records failures in the circuit breaker. The full pipeline exists -- it is just not called automatically.

**Expected behavior:**
1. After auto-retry exhausts its budget (A2), queen checks if any `requires-attempt` failures remain
2. For each such failure, queen calls `dispatchFixer` with `propose` mode (safe default)
3. If Fixer resolves gates (via `resolveFixedGates`), queen proceeds to verification
4. If Fixer fails (via `recordFixerFailure`), queen classifies as `blocking` and escalates
5. Fixer dispatch respects existing attempt cap (`DefaultMaxUnblockAttempts = 1`) -- this is a safety rail, not something the queen overrides

**Complexity:** Low -- the entire Fixer pipeline exists, needs orchestration only

**Dependencies:** `dispatchFixer`, `resolveFixedGates`, `recordFixerFailure`, `checkAttemptCap`

**Confidence:** HIGH -- all infrastructure exists and is tested

---

#### A5. Escalation Protocol with Human Handoff

**Why expected:** Every autonomous supervisor must have a bounded escalation path. The queen cannot recover from everything. Erlang/OTP: supervisor terminates when MaxR/MaxT exceeded, its own supervisor handles it. LangGraph: `interrupt_before` for human-in-the-loop. Google ADK: Human-in-the-Loop pattern with ApprovalTool. The key insight from research: escalation is not failure -- it is the supervisor demonstrating good judgment.

**Current state:** Aether's 4-tier escalation exists conceptually (retry -> reassign -> queen reassign -> user escalation) but is not implemented as a state machine. The user is always the escalation target. There is no intermediate "queen tried everything, here is what happened" summary.

**Expected behavior:**
1. When all recovery strategies exhaust (retry, peer redistribution, Fixer), queen generates an escalation summary
2. Summary includes: what failed, what recovery was attempted, why each recovery failed, what the user needs to do
3. Queen marks the phase as `blocked` in COLONY_STATE.json with the escalation reason
4. The escalation summary is persisted so `/ant-resume` can pick it up
5. User sees a clean "Queen is stuck, needs your help" message, not raw gate output

**Complexity:** Medium -- the escalation summary generation is the novel part

**Dependencies:** COLONY_STATE.json phase status, gate results, recovery attempt history

**Confidence:** HIGH -- pattern is well-established; implementation is mostly formatting existing data

---

### Category B: Smart Gating

How gates transition from "everything blocks, user decides" to "non-critical auto-resolves, genuine problems block."

#### B1. Gate Severity Classification

**Why expected:** Not all gate failures are equal. A `complexity` gate finding that one file is 310 lines (threshold is 300) is fundamentally different from a `gatekeeper` gate finding a critical CVE. Research on multi-agent error cascades (arXiv 2603.04474v1) shows that treating all failures equally causes noise amplification and cascade failures.

**Current state:** All 11 gates produce `GateCheckResult` with `Status` (passed/failed/skipped/not-reached) and `Detail`. Some gates have `FixHint` and `RecoveryOptions`. But there is no severity field. The gate system treats a complexity threshold breach identically to a critical security vulnerability.

**Expected behavior:**
1. Add `Severity` field to `GateCheckResult`: `critical`, `high`, `medium`, `low`, `info`
2. Each gate defines its own severity mapping (e.g., `gatekeeper` -> critical by default, `complexity` -> medium by default)
3. Gate implementations can override severity based on finding details (e.g., `gatekeeper` finding a low-severity npm advisory -> medium, not critical)
4. Severity is persisted with gate results and visible in `/ant-status`

**Complexity:** Medium -- adding the field is trivial; getting severity mappings right per gate requires care

**Dependencies:** `GateCheckResult`, gate implementations in `codex_continue.go`

**Confidence:** HIGH -- straightforward extension of existing data model

---

#### B2. Non-Blocking Advisory Gates

**Why expected:** The Google ADK framework distinguishes between "blocking checks" and "advisory checks" in its `codexWorkflowProfile`. CrewAI's hierarchical process has result validation but not every validation blocks. The concept is universal: some findings are worth noting but not worth stopping progress for.

**Current state:** `codexWorkflowProfile` already has `BlockingChecks` and `AdvisoryChecks` fields. But the gate execution in `runCodexContinueGates` does not use them -- all failed gates are blocking. The `shouldSkipGate` function only skips previously-passed gates, not low-severity findings.

**Expected behavior:**
1. Gates classified as `low` or `info` severity become advisory by default
2. Advisory gates still run and record findings, but do not block phase advancement
3. Advisory findings are aggregated into a "Phase Notes" section in the continue report
4. The user can promote an advisory gate to blocking via a pheromone or flag (existing REDIRECT mechanism)
5. Critical and high severity gates remain blocking (unchanged behavior)

**Complexity:** Medium -- requires modifying gate execution flow without breaking existing behavior

**Dependencies:** `codexWorkflowProfile.BlockingChecks`/`AdvisoryChecks`, `runCodexContinueGates`, `shouldSkipGate`

**Confidence:** HIGH -- fields exist, just need wiring

---

#### B3. Auto-Resolution for Known Recoverable Patterns

**Why expected:** LangGraph's node-level retries handle known-recoverable patterns automatically. Erlang's supervisor distinguishes permanent from temporary errors. The EAGER framework (IJCAI 2025) uses historical failure patterns for efficient failure management. The idea: if a gate failure matches a previously-seen-and-resolved pattern, auto-resolve it.

**Current state:** `gateRecoveryTemplates` provide manual recovery instructions. `resolveFixedGates` can mark gates as passed. But there is no auto-resolution -- the user or Fixer must always act.

**Expected behavior:**
1. Maintain a small set of auto-resolvable patterns (e.g., `tests_pass` gate failing due to a known flaky test that the Fixer already fixed once this session)
2. When a gate fails, check if the failure matches an auto-resolvable pattern
3. If matched: attempt the automated fix (re-run tests with retry), and if it passes, mark gate as passed with `auto-resolved` status
4. Auto-resolutions are logged with full provenance (what was auto-resolved, why, when)
5. Auto-resolution count is bounded per phase (default: 2 auto-resolutions max) to prevent masking real problems

**Complexity:** Medium-High -- requires pattern matching logic and careful safety bounds

**Dependencies:** `gateRecoveryTemplates`, `resolveFixedGates`, gate execution flow

**Confidence:** MEDIUM -- pattern is established in literature but implementation details need phase-specific research

---

#### B4. Gate Dependency Graph

**Why expected:** In complex multi-agent systems, gates can have dependencies. For example, `tdd_evidence` is meaningless if `spawn_gate` failed (no workers spawned). Running dependent gates when their prerequisite has already failed wastes time and produces noisy output. Google ADK's sequential pipeline pattern handles this via state management.

**Current state:** Gates run in a fixed sequence defined in `runCodexContinueGates`. There is no dependency metadata. If `spawn_gate` fails, all subsequent gates still run and produce failure output.

**Expected behavior:**
1. Define gate dependencies: some gates require others to pass first
2. If a prerequisite gate fails, dependent gates are marked `not-reached` (status already exists in `GateCheckResult`)
3. This reduces gate execution time and output noise when a fundamental problem exists
4. Dependencies are declared in a simple map, not a complex DAG

**Complexity:** Low -- `not-reached` status exists, just need skip logic

**Dependencies:** `runCodexContinueGates`, `GateCheckResult.Status`

**Confidence:** HIGH -- straightforward optimization

---

### Category C: Clean Output

How the queen filters, summarizes, and presents information so the user sees what matters.

#### C1. Output Severity Filtering

**Why expected:** Research on multi-agent noise (RCAFlow, AAAI 2025) shows that multi-agent systems "introduce low-level noise in complex multi-stage diagnostic workflows." The AgentReport system (MDPI) uses "fixed responsibilities, input/output contracts, and integration with quantitative evaluation mechanisms" for structured output. Without filtering, users drown in irrelevant details.

**Current state:** Worker output goes through `emitVisualProgress` and ceremony emitters. There is no filtering -- all worker output is displayed. The `codexBuildManifest` captures dispatches and results, but the display layer shows everything.

**Expected behavior:**
1. Each output line gets a severity tag: `progress`, `info`, `warning`, `error`, `success`
2. Default display level shows `warning` and above
3. Verbose mode (`--verbose` or `/ant-run --verbose`) shows everything
4. The queen's summary at phase end shows: tasks completed, tasks failed, gates passed, gates failed (with severity), auto-recoveries attempted, escalations needed
5. Worker output is NOT shown in real-time by default -- only the queen's summary and any escalation messages

**Complexity:** Medium -- requires tagging all output sources and a filtering layer

**Dependencies:** `emitVisualProgress`, ceremony emitters, visual output rendering

**Confidence:** HIGH -- standard practice in production systems

---

#### C2. Phase Completion Summary

**Why expected:** Every autonomous supervisor must report what happened. Google ADK's sequential pipeline produces aggregated output. CrewAI's hierarchical process has result validation. The user should not need to read raw gate output to understand phase outcomes.

**Current state:** `codexContinueReport` has `Summary` field but it is populated by the wrapper (Claude/OpenCode), not the Go runtime. The runtime produces `codexContinueGateReport` and `codexContinueVerificationReport` but does not synthesize them into a human-readable summary.

**Expected behavior:**
1. At phase completion, the Go runtime generates a structured summary with: phase number, tasks completed/total, gates passed/failed/advisory, auto-recoveries attempted, time elapsed, next action needed
2. Summary is rendered by the visual output system (Go runtime, not wrapper)
3. Summary replaces the current approach of showing raw gate-by-gate output
4. Detailed output is available via `/ant-phase N` or `--verbose`

**Complexity:** Low-Medium -- data is available, synthesis logic is new

**Dependencies:** `codexContinueReport`, `codexContinueGateReport`, `codexContinueVerificationReport`, visual rendering

**Confidence:** HIGH -- data aggregation, no architectural changes needed

---

#### C3. Progressive Disclosure

**Why expected:** Autonomous systems must not overwhelm users with information. The principle of progressive disclosure (show summary first, details on demand) is universal in UX design and is specifically called out in multi-agent system design papers as essential for trust.

**Current state:** `/ant-status` shows a dashboard, `/ant-phase N` shows phase details, `/ant-memory-details` shows memory drill-down. The progressive disclosure skeleton exists but is not applied to build/continue output.

**Expected behavior:**
1. During build: show wave progress (worker started, worker completed) but not worker output
2. During continue: show gate pass/fail summary but not individual gate details
3. On escalation: show what failed, what was attempted, and what the user needs to do
4. All detailed output available on demand via existing commands
5. The queen's output follows a consistent template: "X of Y tasks done, Z gates passed, 1 issue needs your attention"

**Complexity:** Medium -- requires consistent output format decisions across multiple command paths

**Dependencies:** Visual rendering system, ceremony emitters, wrapper commands

**Confidence:** HIGH -- well-understood UX pattern

---

### Category D: Queen Wave Coordination

How the queen manages the full wave lifecycle within a phase end-to-end.

#### D1. Wave Lifecycle Ownership

**Why expected:** The queen should own the wave lifecycle: dispatch, monitor, recover, verify, advance. Currently, the user drives each step manually. Google ADK's Coordinator pattern shows how a central agent routes and monitors. Erlang's supervisor owns child process lifecycle.

**Current state:** `dispatchBatchByWaveWithVisuals` handles wave dispatch. `codexBuildProgress` emits wave progress events. But these are passive -- they report what happened, they do not decide what to do next. The queen observes but does not act.

**Expected behavior:**
1. Queen receives wave completion events (which workers completed, which failed)
2. For failures: queen applies auto-recovery (A1-A5) before reporting to user
3. Only when queen's recovery is exhausted does the user see a failure message
4. Queen tracks wave-level progress (wave 1: 3/4 tasks done, 1 auto-recovered) and reports at wave end
5. Queen decides whether to proceed to the next wave or pause

**Complexity:** Medium-High -- requires event-driven coordination between dispatch and recovery systems

**Dependencies:** `dispatchBatchByWaveWithVisuals`, `CircuitBreaker`, Fixer dispatch, gate system

**Confidence:** MEDIUM -- architectural integration is the main risk, not individual components

---

#### D2. Inter-Wave Decision Making

**Why expected:** Between waves, the queen should evaluate whether conditions warrant continuing. If wave 1 had 3 auto-recoveries, wave 2 might need different parameters. LangGraph's conditional edges route based on state. The EAGER framework uses reasoning traces for failure pattern detection.

**Current state:** Waves are dispatched statically at build time based on `codexWaveExecutionPlan`. There is no dynamic adjustment between waves. If wave 1 reveals that the codebase has widespread issues, wave 2 still runs with the same plan.

**Expected behavior:**
1. Between waves, queen evaluates: how many failures occurred, how many auto-recoveries were needed, any new blockers detected
2. If failure rate exceeds threshold (e.g., >50% of tasks in a wave failed): queen pauses and escalates
3. If failure rate is moderate (20-50%): queen adjusts next wave parameters (smaller batches, more conservative dispatch)
4. If failure rate is low (<20%): queen proceeds normally
5. Decision and rationale are logged

**Complexity:** Medium -- decision logic is straightforward, integration with wave dispatch is the work

**Dependencies:** Wave execution plan, dispatch results, circuit breaker state

**Confidence:** MEDIUM -- requires careful threshold tuning in practice

---

#### D3. Phase Completion Decision

**Why expected:** The queen should be able to declare a phase complete when all gates pass, not wait for the user to run `/ant-continue`. This is the natural endpoint of wave coordination.

**Current state:** Phase completion requires: user runs `/ant-build N`, then user runs `/ant-continue`. The continue flow runs gates, generates reports, and advances the phase. But every step requires explicit user invocation.

**Expected behavior:**
1. When all waves in a phase complete and all tasks are done, queen automatically runs gate checks
2. If all gates pass: queen advances the phase, records learnings, and reports completion
3. If gates fail: queen applies smart gating (B1-B4) and auto-recovery (A1-A5)
4. Only when queen cannot recover does the user need to act
5. In `/ant-run` (autopilot) mode, this is the default behavior
6. In manual mode, queen still auto-recovers but asks for confirmation before phase advance

**Complexity:** High -- this is the most complex feature, touching build, continue, gate, and recovery systems

**Dependencies:** All Category A, B, C features; autopilot (`/ant-run`); phase advancement logic

**Confidence:** MEDIUM -- highest architectural integration risk

---

## Feature Dependencies

```
A1 (Failure Classification)
  -> A2 (Bounded Auto-Retry)
  -> A3 (Peer Redistribution)
  -> A4 (Queen-Driven Fixer)
  -> A5 (Escalation Protocol)

B1 (Gate Severity)
  -> B2 (Advisory Gates)
  -> B3 (Auto-Resolution)
  -> B4 (Gate Dependencies)

C1 (Output Filtering)
  -> C2 (Phase Summary)
  -> C3 (Progressive Disclosure)

D1 (Wave Ownership) depends on A1-A5
D2 (Inter-Wave Decisions) depends on D1, A1
D3 (Phase Completion) depends on D1, D2, B1-B4, C1-C3
```

## MVP Recommendation

**Phase 1 (Core Recovery Loop):**
- A1: Failure Classification Engine
- A2: Bounded Auto-Retry
- A3: Peer Redistribution
- A4: Queen-Driven Fixer Dispatch
- A5: Escalation Protocol

**Rationale:** These five features form the complete auto-recovery loop. They wire together existing infrastructure (CircuitBreaker, Fixer, gate results) into an autonomous decision chain. The user sees immediate value: fewer manual interventions, faster recovery from transient failures.

**Phase 2 (Smart Gates):**
- B1: Gate Severity Classification
- B2: Non-Blocking Advisory Gates
- B4: Gate Dependency Graph
- C1: Output Severity Filtering
- C2: Phase Completion Summary

**Rationale:** Once the queen can auto-recover, reducing gate noise becomes the next priority. Severity classification enables advisory gates, which reduces blocking. Output filtering makes the recovery visible.

**Phase 3 (Full Coordination):**
- B3: Auto-Resolution for Known Patterns
- C3: Progressive Disclosure
- D1: Wave Lifecycle Ownership
- D2: Inter-Wave Decision Making
- D3: Phase Completion Decision

**Rationale:** Full coordination is the payoff. The queen manages entire phases autonomously. This depends on both recovery and smart gating being solid first.

**Defer:**
- Learned severity thresholds (adjusting severity based on colony history) -- requires multiple colonies' worth of data
- Cross-phase pattern learning (queen learns from failures in phase 3 to prevent similar failures in phase 7) -- complexity not justified yet
- User-defined recovery strategies (letting users define custom auto-recovery rules) -- YAGNI until core loop is proven

## Anti-Features

| Anti-Feature | Why Avoid | What to Do Instead |
|---|---|---|
| Silent auto-recovery without logging | User loses visibility into what happened; cannot debug if auto-recovery makes wrong choice | Log every recovery decision with full provenance; make logs accessible via `/ant-phase N` |
| Queen overriding circuit breaker thresholds | Circuit breaker exists to prevent cascade failures; overriding defeats its purpose | Queen respects all existing safety rails (MaxR, MaxT, attempt caps) |
| LLM-based failure classification | Non-deterministic; hard to debug; adds latency and cost | Rule-based classification using gate metadata (FixHint, RecoveryOptions, gate name) |
| Auto-advance without confirmation in manual mode | User expects to control phase advancement; silent advance is surprising | In manual mode, queen recovers but asks before advancing; in autopilot mode, full auto |
| Auto-resolving critical severity gates | Critical gates exist for a reason (security, data loss); auto-resolving them is dangerous | Critical gates always block and always escalate to human |
| Queen modifying worker code directly | Creates blame attribution problems; hard to audit; breaks worker autonomy | Queen dispatches Fixer (existing mechanism) which has its own audit trail |
| Infinite recovery loops | Erlang/OTP's lesson: if recovery keeps failing, stop recovering and escalate | MaxR/MaxT bounds on all recovery; circuit breaker integration; hard escalation after budget exhausted |

## Complexity Summary

| Feature | Complexity | Risk | Existing Infrastructure |
|---|---|---|---|
| A1 Failure Classification | Medium | Low | GateCheckResult, gateRecoveryTemplates |
| A2 Bounded Auto-Retry | Low | Low | CircuitBreaker, RetryCount field |
| A3 Peer Redistribution | Low | Low | findSameCastePeer, emitCircuitBreakerRedistributed |
| A4 Queen-Driven Fixer | Low | Low | dispatchFixer, resolveFixedGates |
| A5 Escalation Protocol | Medium | Low | COLONY_STATE.json, gate results |
| B1 Gate Severity | Medium | Low | GateCheckResult (needs field) |
| B2 Advisory Gates | Medium | Medium | BlockingChecks/AdvisoryChecks fields |
| B3 Auto-Resolution | Medium-High | Medium | gateRecoveryTemplates, resolveFixedGates |
| B4 Gate Dependencies | Low | Low | not-reached status |
| C1 Output Filtering | Medium | Medium | emitVisualProgress, ceremony emitters |
| C2 Phase Summary | Low-Medium | Low | codexContinueReport data |
| C3 Progressive Disclosure | Medium | Medium | /ant-status, /ant-phase commands |
| D1 Wave Ownership | Medium-High | Medium | dispatchBatchByWaveWithVisuals |
| D2 Inter-Wave Decisions | Medium | Medium | Wave execution plan |
| D3 Phase Completion | High | Medium-High | All of the above |

## Sources

| Source | Type | Confidence |
|---|---|---|
| Erlang/OTP Supervisor Behaviour (erlang.org/doc/apps/stdlib/supervisor.html) | Official docs | HIGH |
| Google ADK Multi-Agent Patterns (developers.googleblog.com, 2025-12-16) | Official docs | HIGH |
| LangGraph Multi-Agent Patterns (langchain-ai.github.io/langgraph/concepts/multi_agent/) | Official docs | HIGH |
| CrewAI Hierarchical Process (docs.crewai.com/en/learn/hierarchical-process) | Official docs | HIGH |
| OpenAI Swarm Handoff Patterns (github.com/openai/swarm) | Official source | MEDIUM (experimental) |
| AutoGen Error Handling (github.com/microsoft/autogen) | Official source | MEDIUM |
| EAGER: Efficient Failure Management (arxiv.org/abs/2603.21522, IJCAI 2025) | Peer-reviewed research | MEDIUM |
| Error Cascades in Multi-Agent Systems (arxiv.org/html/2603.04474v1) | Peer-reviewed research | MEDIUM |
| RCAFlow: Hierarchical Planning (ojs.aaai.org, AAAI) | Peer-reviewed research | MEDIUM |
| Aether source code: cmd/gate.go, cmd/circuit_breaker.go, cmd/fixer_dispatch.go, cmd/unblock_cmd.go, cmd/dispatch_runtime.go | Source code verification | HIGH |
| Aether source code: cmd/codex_dispatch_contract.go, cmd/codex_build.go, cmd/codex_continue.go | Source code verification | HIGH |
