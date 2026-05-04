# Project Research Summary

**Project:** Aether v1.14 -- Queen Authority
**Domain:** Autonomous queen coordination for multi-agent colony framework
**Researched:** 2026-05-03
**Confidence:** HIGH

## Executive Summary

Queen Authority transforms the Aether queen from narrator into autonomous coordinator. The core insight from all four research streams is that the Go runtime already contains nearly every component needed -- circuit breaker, Fixer dispatch, gate evaluation, retry logic, wave management, and process tracking -- but these components require manual triggering. v1.14 wires them together into a self-driving coordination loop so the queen can detect failures, classify severity, apply bounded recovery, and escalate only when genuinely stuck.

The recommended approach is a phased build: start with gate classification and recovery data infrastructure (low risk, everything depends on it), then the smart gate pipeline and recovery orchestrator, then queen-led build/continue integration, and finally output filtering as the presentation layer. No new external dependencies are needed. The estimated new code is approximately 1,300 lines across a new `pkg/queen/` package plus modifications to existing `cmd/` files.

The key risks are cascading fix-fail cycles (queen retries a fundamentally broken task endlessly), smart gates auto-resolving legitimate security findings, and output filtering hiding critical errors from the user. All three are mitigated by the same principle: the queen advises, the Go runtime decides. Hard-block gates (watcher_veto, gatekeeper, flags, loop_detection) must never be auto-resolved. Every queen decision must be logged to an audit trail. Recovery must be bounded by a per-phase budget with mandatory human escalation when exhausted.

## Key Findings

### Recommended Stack

Zero new external dependencies. Every capability the queen needs already exists in the Go runtime.

**Core technologies (existing, no version changes):**
- `pkg/codex/dispatch.go` (`DispatchBatchWithObserver`): wave-based worker dispatch with observer callbacks -- the queen's sensory input for lifecycle events
- `cmd/codex_dispatch_contract.go` (`recommendQueenWorkflowProfile`): profile recommendation engine -- the queen applies her recommendation directly instead of emitting advice
- `cmd/circuit_breaker.go`: per-worker failure tracking with threshold-based tripping and same-caste peer redistribution
- `cmd/gate.go`: 11 named gates with per-phase persistence and recovery templates
- `cmd/fixer_dispatch.go`: Fixer dispatch with attempt caps and circuit breaker integration
- `cmd/immune.go`: error diagnosis and exponential backoff retry (`2^attempt * 2` seconds)
- `pkg/codex/process_tracker.go`: PID registry, stale worker detection, graceful termination
- `pkg/colony/state_machine.go`: state transitions and phase advancement
- `pkg/storage.Store` (`UpdateJSONAtomically`): file-locked atomic state mutations

**Rejected additions (and why):**
- `thejerf/suture` (supervisor trees) -- workers are short-lived subprocesses, not long-running services; existing dispatch + circuit breaker already handles this
- `oklog/run` (actor group) -- queen loop is sequential, not concurrent; wave-internal concurrency already handled by `DispatchWaveWithObserver`
- External backoff libraries -- single formula (`2^attempt * 2`) already implemented in `cmd/immune.go`
- State machine libraries -- `pkg/colony/state_machine.go` already exists and is tested

### Expected Features

**Must have (table stakes -- core recovery loop):**
- A1: Failure Classification Engine -- distinguish recoverable vs non-recoverable before acting
- A2: Bounded Auto-Retry with Exponential Backoff -- retry transient failures with MaxR/MaxT bounds
- A3: Peer Redistribution on Worker Failure -- use `findSameCastePeer` automatically when circuit breaker trips
- A4: Queen-Driven Fixer Dispatch -- call `dispatchFixer` automatically for `requires-attempt` failures
- A5: Escalation Protocol with Human Handoff -- generate structured escalation summary when recovery exhausted

**Should have (competitive differentiators):**
- B1: Gate Severity Classification -- add `critical`/`high`/`medium`/`low`/`info` to `GateCheckResult`
- B2: Non-Blocking Advisory Gates -- low/info severity findings logged but don't block advancement
- B4: Gate Dependency Graph -- skip dependent gates when prerequisite fails (use existing `not-reached` status)
- C1: Output Severity Filtering -- default display shows warnings and above; verbose shows everything
- C2: Phase Completion Summary -- Go runtime generates structured summary (tasks, gates, duration, next action)

**Defer (v2+):**
- B3: Auto-Resolution for Known Recoverable Patterns -- pattern matching logic needs careful safety bounds
- C3: Progressive Disclosure -- UX refinement, not core functionality
- D1-D3: Full Wave Lifecycle Ownership, Inter-Wave Decision Making, Phase Completion Decision -- highest architectural integration risk, prove the recovery loop first
- Learned severity thresholds -- requires multi-colony data
- Cross-phase pattern learning -- complexity not justified until core loop proven
- User-defined recovery strategies -- YAGNI

**Anti-features (never implement):**
- Silent auto-recovery without logging -- destroys user visibility
- Queen overriding circuit breaker thresholds -- defeats the safety mechanism
- LLM-based failure classification -- non-deterministic, hard to debug; use rule-based classification
- Auto-advance without confirmation in manual mode -- violates user expectations
- Auto-resolving critical severity gates -- security/safety gates must always block
- Queen modifying worker code directly -- breaks audit trail and worker autonomy
- Infinite recovery loops -- Erlang/OTP lesson: bounded recovery with mandatory escalation

### Architecture Approach

The architecture follows a strict authority model: the Queen (wrapper layer) makes decisions, the Go runtime (authoritative) executes them. The Queen cannot write COLONY_STATE.json directly, cannot skip hard-block gates, and cannot force phase advance without verification. This preserves the existing wrapper-runtime contract.

**Major components:**

1. **`pkg/queen/` (new package)** -- coordinator loop that assembles existing components into an autonomous decision chain: dispatch wave, monitor via `DispatchObserver`, evaluate results, run gates with classification, apply recovery, filter output, advance or escalate
2. **`cmd/gate.go` (modified)** -- add `gateClassification` (hard_block/soft_block/advisory), `AutoRecoverable` field, classification-aware `runCodexContinueGates()` that processes soft-block and advisory failures differently from hard blocks
3. **`cmd/recovery.go` (new file)** -- recovery orchestrator with `attemptAutoRecovery()` that runs between gate failure and blocking; strategy per gate (re-run verification, skip spawn in queen-led mode, retry tests)
4. **`cmd/codex_build.go` (modified)** -- add `RecoveryAttempts`, `LastFailureReason` to `codexBuildDispatch`; add `codexRecoveryPolicy` to manifest
5. **`cmd/codex_continue.go` (modified)** -- add `runCodexContinuePlanOnly()` (gates without state mutation) and `runCodexContinueFinalize()` (commit with recovery context); add `codexContinueSummary` struct
6. **Wrapper playbooks (modified)** -- `build-wave.md` Step 5.2 reads recovery metadata and executes recovery tiers 1-3; `continue-gates.md` adds queen-led conditional branches per gate

### Critical Pitfalls

1. **Cascading fix-fail cycles** -- queen re-spawns a worker with a new name, circuit breaker sees a fresh worker, same underlying failure repeats. Prevention: classify failures as retryable vs non-retryable BEFORE recovery; track task-level attempt count (not just worker-level); per-phase recovery budget (max 3); when skipping a task, check downstream dependencies.
2. **Smart gates auto-resolving legitimate findings** -- queen marks a real security/quality issue as "false positive." Prevention: NEVER auto-resolve security gates (gatekeeper, anti_pattern) or watcher_veto; preserve original failure detail in `queen_gate_decisions.json` audit log; require queen to read source code before auto-resolving; flag if auto-resolution rate exceeds 30%.
3. **Output filtering hiding critical information** -- real errors suppressed as "noise." Prevention: two-tier output (summary + always-show errors); `--verbose` default for first phase; persist full output to `queen-output-{phase}.json`; never filter lines containing "error", "failed", "panic", "fatal", "blocked", "veto", "critical".
4. **Queen as single point of failure** -- crash corrupts coordination state; decision loop becomes bottleneck. Prevention: queen ADVISES not COMMANDS (existing Go functions remain decision makers); persist coordination state to COLONY_STATE.json; separate 12K char context budget for queen (don't reuse colony-prime's 8K); write decision logic as pure functions for testability.
5. **Silent failure mode** -- smooth auto-recovery masks real problems so user never knows anything went wrong. Prevention: log every queen decision to persistent file; phase-end activity summary ("Queen made 3 auto-recovery decisions -- review with `/ant-queen-log`"); transparency mode prints one line per autonomous decision by default; dual-flagged findings (two independent gates agree) always block.

## Implications for Roadmap

Based on research, suggested phase structure:

### Phase 1: Gate Classification Infrastructure
**Rationale:** Every subsequent phase depends on knowing which gates are hard-block vs soft-block vs advisory. This is the foundation everything else builds on. Additive changes only -- no existing behavior changes until consumers are added.
**Delivers:** `gateClassification` type, `gateClassifications` map, `AutoRecoverable` field on `gateCheck`, `Severity` field on `GateCheckResult`
**Addresses:** B1 (Gate Severity Classification), B4 (Gate Dependency Graph -- trivial since `not-reached` status already exists)
**Avoids:** Pitfall 2 (smart gates auto-resolving) -- hard-block classification is defined here
**Modifies:** `cmd/gate.go`
**Risk:** Low

### Phase 2: Recovery Data Model
**Rationale:** Auto-recovery logic needs somewhere to store state. Add recovery fields to existing structs and create the `aether recovery-record` subcommand. All changes are backward-compatible with `omitempty`.
**Delivers:** `RecoveryAttempts` and `LastFailureReason` on `codexBuildDispatch`; `RecoveryLog` on `ColonyState`; `codexRecoveryPolicy` on manifest; `aether recovery-record` subcommand
**Addresses:** A1 (Failure Classification -- data structures), A5 (Escalation Protocol -- persistence)
**Avoids:** Pitfall 4 (single point of failure) -- recovery state persisted to COLONY_STATE.json via existing atomic writes
**Modifies:** `cmd/codex_build.go`, `pkg/colony/colony.go`, new `cmd/recovery.go`
**Risk:** Low

### Phase 3: Smart Gate Pipeline and Recovery Orchestrator
**Rationale:** This is the core of queen authority. Combines gate classification (Phase 1) with recovery data (Phase 2) into an autonomous recovery loop. The queen can now auto-recover from soft-block gates before blocking.
**Delivers:** `attemptAutoRecovery()` in `cmd/recovery.go`; classification-aware `runCodexContinueGates()`; `codexSmartGateSummary` struct; per-gate recovery strategies
**Addresses:** A2 (Bounded Auto-Retry), A3 (Peer Redistribution), A4 (Queen-Driven Fixer), B2 (Non-Blocking Advisory Gates)
**Avoids:** Pitfall 1 (cascading fix-fail) via circuit breaker integration; Pitfall 2 (auto-resolving) via hard-block preservation
**Modifies:** `cmd/codex_continue.go`, `cmd/recovery.go`, `cmd/gate.go`
**Risk:** Medium -- changes gate evaluation logic, must preserve existing hard-block behavior

### Phase 4: Queen-Led Continue Integration
**Rationale:** Enables the queen-led continue flow by splitting `aether continue` into `--plan-only` (evaluate without mutating) and `--finalize` (commit with recovery context). This gives the queen the information she needs to make recovery decisions without side effects.
**Delivers:** `aether continue --plan-only`, `aether continue --finalize`, queen-led conditional branches in `continue-gates.md`
**Addresses:** Queen-led mode activation for playbook gates (relaxed thresholds, auto-skip runtime, lower watcher veto threshold)
**Avoids:** Pitfall 4 (queen mutating state directly) -- plan-only is read-only; Pitfall 3 (anti-pattern 3) -- relaxed thresholds only when `QueenLedMode` is true
**Modifies:** `cmd/codex_continue.go`, `continue-gates.md`, `.claude/commands/ant/continue.md`
**Risk:** Medium -- new code paths but isolated behind flags

### Phase 5: Queen-Led Build Recovery
**Rationale:** Wrapper-layer changes that consume the infrastructure from Phases 1-4. The Queen agent now reads recovery metadata from dispatch results and executes recovery tiers 1-3 (retry, reassign, escalate) automatically during build waves.
**Delivers:** Modified `build-wave.md` Step 5.2 with recovery logic; updated Queen agent with recovery-aware spawning; `/ant-queen-log` command for reviewing decisions
**Addresses:** Full Category A (Auto-Recovery Loop), queen Fixer coordination, transparency/audit trail
**Avoids:** Pitfall 6 (queen vs Fixer conflict) -- queen is sole recovery coordinator, Fixer checks recovery lock
**Modifies:** `build-wave.md`, `.claude/agents/ant/aether-queen.md`, `.opencode/agents/aether-queen.md`
**Risk:** Medium -- changes wrapper behavior, but Go runtime is the safety net

### Phase 6: Output Filtering and Phase Summary
**Rationale:** Pure presentation layer, no dependencies on recovery/gating logic beyond reading their output. This is the polish phase that makes queen authority feel clean rather than noisy.
**Delivers:** `renderPhaseSummary()` and `renderSmartGateSummary()` in `cmd/codex_visuals.go`; `codexContinueSummary` struct; `codexBuildSummary` struct; filtered wrapper markdown
**Addresses:** C1 (Output Severity Filtering), C2 (Phase Completion Summary)
**Avoids:** Pitfall 3 (hiding critical info) via two-tier output, `--verbose` default, never-filter keywords, persistent full output
**Modifies:** `cmd/codex_visuals.go`, `cmd/codex_build.go`, `cmd/codex_continue.go`, wrapper markdown
**Risk:** Low -- additive, existing detailed output remains available via `--verbose`

### Phase Ordering Rationale

- Phases 1 and 2 are pure infrastructure with no behavior changes -- they create the foundation
- Phase 3 is the highest-value phase (autonomous recovery becomes real) but depends on 1 and 2
- Phase 4 splits continue into plan/finalize, enabling the queen-led flow that Phase 5 consumes
- Phase 5 wires everything together in the wrapper layer
- Phase 6 is presentation polish that can ship anytime after Phase 3

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 3:** Gate recovery strategies need per-gate research -- what exactly constitutes "auto-recoverable" for each of the 11 gates requires understanding each gate's failure semantics in detail
- **Phase 5:** Queen agent prompt engineering for recovery decisions -- how to give the queen enough context to make good recovery choices without exceeding her context budget

Phases with standard patterns (skip research-phase):
- **Phase 1:** Struct field additions and static map lookups -- well-understood
- **Phase 2:** Data model extensions with `omitempty` -- standard Go pattern
- **Phase 4:** CLI flag additions and code path splitting -- standard cobra pattern
- **Phase 6:** Output formatting and rendering -- standard visual system pattern

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All findings from direct source code inspection; zero new dependencies means no integration risk |
| Features | HIGH | Every feature maps to existing infrastructure; confidence backed by Erlang/OTP, LangGraph, Google ADK, CrewAI patterns |
| Architecture | HIGH | Authority model (queen advises, runtime decides) preserves existing wrapper-runtime contract; all integration points identified with line numbers |
| Pitfalls | HIGH | All 12 pitfalls derived from direct codebase analysis of the actual files that will be modified; prevention strategies reference specific existing functions |

**Overall confidence:** HIGH

### Gaps to Address

- **Gate-specific recovery strategies (Phase 3):** Research identified that each gate has different failure semantics, but the exact recovery strategy per gate needs to be defined during Phase 3 planning. The classification map provides the framework, but "what does auto-recovery look like for `tests_pass` vs `implementation_evidence`?" needs phase-level detail.
- **Queen context budget tuning (Phase 5):** Research recommends 12K chars for queen coordination context, but the exact contents and trim order need to be validated empirically during implementation.
- **Recovery budget defaults (Phase 3):** Research suggests max 3 auto-recovery attempts per phase, but the right number depends on colony size and phase complexity. Should be configurable with sensible defaults.
- **Watcher veto threshold in queen-led mode:** Research suggests lowering from 7 to 5, but this needs user testing. The threshold should be configurable via queen autonomy level.

## Sources

### Primary (HIGH confidence)
- Aether source code: `cmd/gate.go`, `cmd/circuit_breaker.go`, `cmd/fixer_dispatch.go`, `cmd/immune.go`, `cmd/autopilot.go`, `cmd/codex_dispatch_contract.go`, `cmd/codex_build.go`, `cmd/codex_continue.go`, `cmd/colony_prime_context.go`, `cmd/codex_visuals.go`, `cmd/recovery.go` (new)
- Aether source code: `pkg/codex/dispatch.go`, `pkg/codex/worker.go`, `pkg/codex/process_tracker.go`, `pkg/colony/colony.go`, `pkg/colony/state_machine.go`, `pkg/storage/storage.go`
- Aether playbooks: `.aether/docs/command-playbooks/build-wave.md`, `.aether/docs/command-playbooks/continue-gates.md`
- Aether agents: `.claude/agents/ant/aether-queen.md`, `.claude/agents/ant/aether-fixer.md`
- Erlang/OTP Supervisor Behaviour (erlang.org/doc/apps/stdlib/supervisor.html) -- bounded restart patterns

### Secondary (MEDIUM confidence)
- Google ADK Multi-Agent Patterns (developers.googleblog.com, 2025-12-16) -- coordinator pattern, human-in-the-loop
- LangGraph Multi-Agent Patterns (langchain-ai.github.io/langgraph/concepts/multi_agent/) -- retryable vs non-retryable exceptions
- CrewAI Hierarchical Process (docs.crewai.com/en/learn/hierarchical-process) -- result validation
- OpenAI Swarm Handoff Patterns (github.com/openai/swarm) -- experimental but relevant
- AutoGen Error Handling (github.com/microsoft/autogen) -- typed state transitions
- EAGER: Efficient Failure Management (arxiv.org/abs/2603.21522, IJCAI 2025) -- historical failure patterns
- Error Cascades in Multi-Agent Systems (arxiv.org/html/2603.04474v1) -- noise amplification
- RCAFlow: Hierarchical Planning (ojs.aaai.org, AAAI) -- multi-agent noise reduction
- Agent Response Filtering -- Upsonic AI (upsonic.ai/lexicon/agent-response-filtering) -- over-filtering risks
- Establishing Trust in AI Agents -- Medium -- monitoring, control layers
- Agentic AI Security: Threats, Defenses, Evaluation -- arXiv 2510.23883v1 -- output filtering vs sandboxing

### Tertiary (LOW confidence)
- None -- all sources are either direct codebase analysis (HIGH) or established framework documentation/published research (MEDIUM)

---
*Research completed: 2026-05-03*
*Ready for roadmap: yes*
