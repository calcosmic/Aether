# Technology Stack: Queen Authority

**Project:** Aether v1.14 -- Queen Authority
**Researched:** 2026-05-03
**Confidence:** HIGH (all findings from source code inspection of existing Aether runtime)

## Executive Summary

The queen authority milestone requires **zero new external dependencies**. Every capability the queen needs -- worker dispatch, failure detection, circuit breaking, retry with backoff, gate evaluation, output aggregation, and process lifecycle management -- already exists as discrete components in the Go runtime. The work is entirely about wiring these existing pieces into a coordinator loop inside the Go binary, not about importing new libraries.

The queen becomes an autonomous coordinator by reading existing infrastructure that is currently driven externally by wrapper markdown (Claude/OpenCode) or by manual user invocation:

| Current Driver | Queen Authority Change |
|---------------|----------------------|
| Wrapper markdown calls `aether build`, `aether continue` | Go runtime runs build/continue in a loop internally |
| User manually runs `/ant-unblock` when gates fail | Go runtime evaluates gates and dispatches Fixer automatically |
| User reads raw worker output | Go runtime filters/summarizes output before surfacing |
| User decides whether to re-dispatch a failed worker | Go runtime applies circuit breaker + retry policy |
| User manages wave progression | Go runtime manages wave lifecycle end-to-end |

## Existing Infrastructure Already Available

The following components exist and are production-tested. The queen coordinator assembles them into an autonomous loop.

### 1. Worker Dispatch (`pkg/codex/dispatch.go`)

Already has wave-based dependency ordering, lifecycle events, parallel execution, and result aggregation.

| Component | Location | Status |
|-----------|----------|--------|
| `DispatchBatchWithObserver` | `pkg/codex/dispatch.go:92` | Production, tested |
| `DispatchWaveWithObserver` | `pkg/codex/dispatch.go:124` | Production, tested |
| `DispatchObserver` callback | `pkg/codex/dispatch.go:50` | Production, tested |
| `ExtractClaims` aggregation | `pkg/codex/dispatch.go:228` | Production, tested |
| `WorkerInvoker` interface | `pkg/codex/worker.go:125` | Production, tested |
| `ProgressAwareWorkerInvoker` | `pkg/codex/worker.go:120` | Production, tested |
| `WorkerResult` struct | `pkg/codex/worker.go:60` | Production, tested |

**What this means for queen:** The queen does not need to spawn workers herself. She calls `DispatchBatchWithObserver` with a `DispatchObserver` callback that feeds her lifecycle events (starting, running, completed, failed, timeout). The observer is the queen's sensory input.

### 2. Circuit Breaker (`cmd/circuit_breaker.go`)

Already has per-worker failure tracking, threshold-based tripping, same-caste peer redistribution, and ceremony event emission.

| Component | Location | Status |
|-----------|----------|--------|
| `CircuitBreaker` struct | `cmd/circuit_breaker.go:23` | Production, tested |
| `Allow/RecordSuccess/RecordFailure` | `cmd/circuit_breaker.go:44-68` | Production, tested |
| `findSameCastePeer` | `cmd/circuit_breaker.go:102` | Production, tested |
| `Reset` (per-wave) | `cmd/circuit_breaker.go:71` | Production, tested |
| `TrippedWorkers` | `cmd/circuit_breaker.go:86` | Production, tested |
| Ceremony events | `cmd/circuit_breaker.go:120-155` | Production, tested |

**What this means for queen:** The queen already has a circuit breaker. She resets it at wave boundaries, records failures/successes from `DispatchObserver` events, checks `Allow()` before dispatching, and calls `findSameCastePeer` when a worker is tripped.

### 3. Gate System (`cmd/gate.go`, `cmd/fixer_dispatch.go`)

Already has 11 named gates, per-phase persistence, incremental re-check (skip passed), recovery templates, and Fixer dispatch with attempt caps.

| Component | Location | Status |
|-----------|----------|--------|
| `gateCheck` struct | `cmd/gate.go:22` | Production, tested |
| `shouldSkipGate` | `cmd/gate.go:557` | Production, tested |
| `gateResultsWritePhase` | `cmd/gate.go:605` | Production, tested |
| `gateResultsReadPhase` | `cmd/gate.go:613` | Production, tested |
| `gateRecoveryTemplates` | `cmd/gate.go:487-536` | Production, tested |
| `dispatchFixer` | `cmd/fixer_dispatch.go:131` | Production, tested |
| `resolveFixedGates` | `cmd/fixer_dispatch.go:217` | Production, tested |
| `checkAttemptCap` | `cmd/fixer_dispatch.go:92` | Production, tested |
| `isFixerDispatchBlocked` | `cmd/fixer_dispatch.go:105` | Production, tested |
| 11 gate recovery templates | `cmd/gate.go:487-536` | Production, tested |

**What this means for queen:** Smart gating is mostly wiring. The queen reads gate results after continue, classifies failures as auto-resolvable vs. human-escalation, dispatches Fixer for auto-resolvable ones, and surfaces only genuine blockers to the user. The classification logic is the new part -- the infrastructure for dispatching Fixer and tracking attempts already exists.

### 4. Process Tracking (`pkg/codex/process_tracker.go`)

Already has PID registry, stale worker detection, graceful termination with SIGTERM/SIGKILL escalation, and cleanup.

| Component | Location | Status |
|-----------|----------|--------|
| `ProcessTracker` | `pkg/codex/process_tracker.go:45` | Production, tested |
| `TrackProcess/UntrackProcess` | `pkg/codex/process_tracker.go:68-106` | Production, tested |
| `KillAll` (per-root) | `pkg/codex/process_tracker.go:128` | Production, tested |
| `DetectStaleWorkers` | `pkg/codex/process_tracker.go:144` | Production, tested |
| `CleanupStaleWorkers` | `pkg/codex/process_tracker.go:185` | Production, tested |

**What this means for queen:** The queen can check `DetectStaleWorkers` at wave boundaries and call `KillAll` for cleanup. She does not need to build process management from scratch.

### 5. Immune System / Retry (`cmd/immune.go`)

Already has error diagnosis, exponential backoff retry, scar recording, and auto-scar detection from midden failures.

| Component | Location | Status |
|-----------|----------|--------|
| `diagnoseError` | `cmd/immune.go:46` | Production, tested |
| `trophallaxisRetryCmd` (backoff) | `cmd/immune.go:91` | Production, tested |
| `scarAdd/scarCheck` | `cmd/immune.go:126-238` | Production, tested |
| `immuneAutoScar` | `cmd/immune.go:240` | Production, tested |

**What this means for queen:** The queen calls `diagnoseError` on worker failures to determine retryability, then uses the existing backoff formula (`2^attempt * 2` seconds) for retry delays. She records failures as scars for future avoidance.

### 6. Autopilot State (`cmd/autopilot.go`)

Already has phase tracking, replan interval checking, headless mode, and stop/init commands.

| Component | Location | Status |
|-----------|----------|--------|
| `autopilotState` struct | `cmd/autopilot.go:20` | Production, tested |
| `autopilot-update` | `cmd/autopilot.go:85` | Production, tested |
| `autopilot-check-replan` | `cmd/autopilot.go:223` | Production, tested |
| `autopilot-set-headless` | `cmd/autopilot.go:277` | Production, tested |

**What this means for queen:** The autopilot state machine already tracks multi-phase progress. The queen authority loop extends this from "track and report" to "decide and act."

### 7. Workflow Profile System (`cmd/codex_dispatch_contract.go`)

Already has three profiles (fast, standard, final-review), queen recommendation engine, depth flags, and lifecycle command contracts.

| Component | Location | Status |
|-----------|----------|--------|
| `recommendQueenWorkflowProfile` | `cmd/codex_dispatch_contract.go:329` | Production, tested |
| `workflowProfiles` | `cmd/codex_dispatch_contract.go:138` | Production, tested |
| `codexWorkflowProfileContract` | `cmd/codex_dispatch_contract.go:62` | Production, tested |

**What this means for queen:** The queen already recommends profiles. In autonomous mode, she applies her recommendation directly instead of emitting it as advice.

### 8. Colony State (`pkg/colony/colony.go`)

Already has state machine transitions, phase advancement, worktree tracking, gate results, and charter data.

| Component | Location | Status |
|-----------|----------|--------|
| `ColonyState` struct | `pkg/colony/colony.go:272` | Production, tested |
| `Transition` (state machine) | `pkg/colony/state_machine.go:7` | Production, tested |
| `AdvancePhase` | `pkg/colony/state_machine.go:23` | Production, tested |
| `GateResultEntry` | `pkg/colony/colony.go:227` | Production, tested |
| `WorktreeEntry` | `pkg/colony/colony.go:214` | Production, tested |

**What this means for queen:** All state mutations go through `store.UpdateJSONAtomically`, which is file-locked and atomic. The queen reads state, makes decisions, writes state -- all through existing safe patterns.

## New Code Needed (No New Dependencies)

### Queen Coordinator Loop

A single new package `pkg/queen/` that assembles the existing components into an autonomous decision loop.

```
pkg/queen/
├── coordinator.go      # Main loop: dispatch -> monitor -> recover -> advance
├── gate_classifier.go  # Classifies gate failures as auto-resolvable vs. escalate
├── output_filter.go    # Filters and summarizes worker output for clean display
├── retry_policy.go     # Retry decisions using existing immune.go backoff
├── wave_manager.go     # Wave lifecycle: allocate, execute, collect, cleanup
├── coordinator_test.go
├── gate_classifier_test.go
├── output_filter_test.go
├── retry_policy_test.go
└── wave_manager_test.go
```

### cmd/ Additions

| New Command | Purpose | Pattern |
|-------------|---------|---------|
| `queen-coordinator` | Run the autonomous coordinator loop (replaces wrapper-driven build/continue) | Extends `cmd/autopilot.go` pattern |
| `queen-gate-classify` | Classify a gate failure as auto-resolvable or human-escalation | New, uses existing gate data |
| `queen-output-filter` | Filter and summarize worker output | New, pure text processing |
| `queen-retry-decide` | Decide whether to retry a failed worker | Uses existing `diagnoseError` + `trophallaxisRetryCmd` logic |
| `queen-wave-status` | Current wave status and worker health | Aggregates existing dispatch + process tracker data |

### Gate Classification Logic (New)

The core new intellectual work: deciding which gate failures the queen can auto-resolve and which require human intervention.

```go
// pkg/queen/gate_classifier.go
type GateClassification struct {
    Name         string
    Severity     string // "auto", "retry", "escalate"
    Reason       string
    RecoveryHint string
}

// Auto-resolvable gates: the queen can dispatch Fixer and verify
var autoResolvableGates = map[string]bool{
    "tests_pass":       true,  // Fixer re-runs tests, fixes failures
    "watcher_veto":     true,  // Fixer addresses quality issues
    "tdd_evidence":     true,  // Fixer writes missing tests
    "anti_pattern":     true,  // Fixer removes critical patterns
    "complexity":       true,  // Fixer refactors complex code
    "medic":            true,  // Fixer applies medic repairs
}

// Escalation gates: require human judgment
var escalationGates = map[string]bool{
    "gatekeeper":    true,  // Security CVEs need human review
    "auditor":       true,  // Quality below threshold needs human review
    "runtime":       true,  // User-reported issues need human confirmation
    "flags":         true,  // Blocker flags represent user decisions
}
```

**Confidence: HIGH** -- The gate names and recovery templates already exist in `cmd/gate.go:487-536`. Classification is a mapping from existing data to new behavior.

### Output Filtering Logic (New)

The queen needs to suppress raw worker noise and surface only actionable summaries.

```go
// pkg/queen/output_filter.go
type FilteredOutput struct {
    Summary    string            // Human-readable phase summary
    Workers    []WorkerSummary   // Per-worker status line
    Warnings   []string          // Notable but non-blocking items
    Blockers   []string          // Items requiring human attention
    Artifacts  map[string]string // Key outputs (test results, coverage, etc.)
}

type WorkerSummary struct {
    Name    string
    Caste   string
    Status  string // "completed", "failed", "retried", "skipped"
    Summary string // Worker's self-reported summary (from WorkerResult.Summary)
    Files   int    // Count of files created/modified
    Duration string
}
```

**Approach:** The queen reads `WorkerResult.Summary` (already a concise self-report), `WorkerResult.RawOutput` (full stdout, only surfaced on failure), and `DispatchResult.Error` (invocation errors). Normal-path output shows one line per worker. Failure-path output shows the worker's summary plus the last 20 lines of raw output.

**Confidence: HIGH** -- All data structures already exist. This is formatting logic, not new infrastructure.

### Retry Policy (New, Wraps Existing)

```go
// pkg/queen/retry_policy.go
type RetryDecision struct {
    Retry      bool
    Attempt    int
    BackoffSec int
    Reason     string
}

func DecideRetry(err error, workerName string, attempt int, cb *CircuitBreaker) RetryDecision {
    // 1. Check circuit breaker
    if !cb.Allow(workerName) {
        return RetryDecision{Retry: false, Reason: "circuit breaker tripped"}
    }

    // 2. Diagnose error using existing immune system
    diagnosis := diagnoseError(err.Error())

    // 3. Check attempt cap (reuse existing trophallaxis logic)
    if attempt >= 3 {
        return RetryDecision{Retry: false, Reason: "max attempts reached"}
    }

    // 4. Calculate backoff (existing formula: 2^attempt * 2)
    backoff := int(math.Pow(2, float64(attempt)) * 2)

    if !diagnosis["retryable"].(bool) {
        return RetryDecision{Retry: false, Reason: fmt.Sprintf("non-retryable: %s", diagnosis["strategy"])}
    }

    return RetryDecision{
        Retry:      true,
        Attempt:    attempt + 1,
        BackoffSec: backoff,
        Reason:     diagnosis["strategy"].(string),
    }
}
```

**Confidence: HIGH** -- `diagnoseError` and backoff formula already exist in `cmd/immune.go`. This is a thin orchestration layer.

## What NOT to Add

| Rejected Addition | Why |
|-------------------|-----|
| `thejerf/suture` (supervisor trees) | Suture manages long-running service goroutines with restart semantics. Aether's workers are short-lived subprocess invocations (spawn, execute, collect result). The existing `DispatchBatchWithObserver` + `CircuitBreaker` + `ProcessTracker` already handle this. Suture would add an abstraction layer that duplicates existing behavior. |
| `oklog/run` (actor group) | `oklog/run` orchestrates concurrent goroutines with graceful shutdown. The queen loop is sequential (dispatch wave, wait, evaluate, decide). Concurrency within a wave is already handled by `DispatchWaveWithObserver`. `oklog/run` solves a problem Aether does not have. |
| `cenkalti/backoff` or `sethvargo/go-retry` | The existing immune system already implements exponential backoff (`2^attempt * 2` seconds) in `cmd/immune.go:114`. Adding a backoff library for a single formula is over-engineering. |
| Event-driven architecture (channels, pub/sub) | The `DispatchObserver` callback pattern is already event-driven. Adding channels or a pub/sub system between dispatch and queen would add complexity without benefit -- the queen processes events synchronously in her loop. |
| State machine library | The colony state machine already exists in `pkg/colony/state_machine.go`. Adding a library like `fsm` or `mergo` would duplicate existing, tested transitions. |
| External queue (Redis, NATS) | Workers are local subprocesses, not distributed services. File-locked JSON persistence via `pkg/storage.Store` is the correct coordination mechanism for single-machine, single-user colonies. |

## Existing Dependency Usage (No Version Changes)

| Dependency | Current Version | Queen Authority Usage |
|------------|----------------|----------------------|
| `golang.org/x/sync` | v0.20.0 | Already used in `pkg/agent/pool.go` for `errgroup`. Wave parallel execution already uses `sync.WaitGroup` in `pkg/codex/dispatch.go`. No change needed. |
| `github.com/spf13/cobra` | v1.10.2 | New queen commands register via `rootCmd.AddCommand()`. No version change. |
| `github.com/BurntSushi/toml` | v1.5.0 | Agent TOML parsing unchanged. No version change. |
| `modernc.org/sqlite` | v1.50.0 | Hive learning store unchanged. Queen reads colony state from JSON (not SQLite). No version change. |
| `pkg/storage.Store` | existing | All state mutations via `UpdateJSONAtomically`. No change. |

## Architecture: Queen Coordinator Loop

The queen coordinator is a sequential decision loop, not a concurrent service:

```
┌──────────────────────────────────────────────────────┐
│                QUEEN COORDINATOR                      │
│                                                       │
│  for each phase in plan:                              │
│    1. SELECT PROFILE                                  │
│       └─ recommendQueenWorkflowProfile() [existing]    │
│                                                       │
│    2. DISPATCH WAVE                                   │
│       ├─ CircuitBreaker.Reset()                       │
│       ├─ DispatchBatchWithObserver(observer)           │
│       └─ observer -> queen event buffer               │
│                                                       │
│    3. EVALUATE WAVE RESULTS                           │
│       ├─ For each failed worker:                      │
│       │   ├─ CircuitBreaker.RecordFailure()            │
│       │   ├─ DecideRetry() -> uses diagnoseError       │
│       │   ├─ If retryable: re-dispatch (back to 2)     │
│       │   └─ If tripped: findSameCastePeer or skip    │
│       └─ ExtractClaims() for success summary          │
│                                                       │
│    4. RUN GATES                                       │
│       ├─ Read gate results (gateResultsReadPhase)      │
│       ├─ ClassifyGate() -> auto | retry | escalate     │
│       ├─ Auto-resolvable: dispatchFixer() [existing]   │
│       ├─ Re-verify fixed gates                        │
│       └─ Escalate remaining to user                   │
│                                                       │
│    5. FILTER OUTPUT                                   │
│       └─ Build FilteredOutput from WorkerResults       │
│                                                       │
│    6. ADVANCE OR ESCALATE                             │
│       ├─ All gates passed: AdvancePhase() [existing]   │
│       ├─ Some gates failed: surface blockers to user  │
│       └─ Record scars, emit telemetry                 │
│                                                       │
│    7. CLEANUP                                         │
│       ├─ DetectStaleWorkers() -> KillAll() [existing]  │
│       └─ spawnTrackClear() [existing]                 │
│                                                       │
└──────────────────────────────────────────────────────┘
```

## Integration Points

| Queen Action | Calls Into | Package |
|-------------|-----------|---------|
| Select profile | `recommendQueenWorkflowProfile` | `cmd/codex_dispatch_contract.go` |
| Dispatch workers | `DispatchBatchWithObserver` | `pkg/codex/dispatch.go` |
| Track failures | `CircuitBreaker.RecordFailure` | `cmd/circuit_breaker.go` |
| Check retryability | `diagnoseError` | `cmd/immune.go` |
| Redistribute task | `findSameCastePeer` | `cmd/circuit_breaker.go` |
| Evaluate gates | `gateResultsReadPhase`, `shouldSkipGate` | `cmd/gate.go` |
| Dispatch fixer | `dispatchFixer` | `cmd/fixer_dispatch.go` |
| Resolve fixed gates | `resolveFixedGates` | `cmd/fixer_dispatch.go` |
| Check attempt cap | `checkAttemptCap` | `cmd/fixer_dispatch.go` |
| Advance phase | `AdvancePhase` | `pkg/colony/state_machine.go` |
| Mutate state | `store.UpdateJSONAtomically` | `pkg/storage/storage.go` |
| Track processes | `ProcessTracker.TrackProcess` | `pkg/codex/process_tracker.go` |
| Cleanup stale | `CleanupStaleWorkers` | `pkg/codex/process_tracker.go` |
| Record scars | `scarAdd` logic (inline) | `cmd/immune.go` |
| Update autopilot | `autopilot-update` logic (inline) | `cmd/autopilot.go` |
| Emit telemetry | `emitLoopBreakEvent` | `cmd/` (existing ceremony) |

## Estimated New Code

| Component | Lines (estimate) | Complexity |
|-----------|-----------------|------------|
| `pkg/queen/coordinator.go` | ~200 | Medium -- main loop orchestration |
| `pkg/queen/gate_classifier.go` | ~80 | Low -- classification map + logic |
| `pkg/queen/output_filter.go` | ~120 | Low -- text formatting |
| `pkg/queen/retry_policy.go` | ~60 | Low -- wraps existing diagnoseError |
| `pkg/queen/wave_manager.go` | ~150 | Medium -- wave lifecycle management |
| `cmd/queen_*.go` (5 commands) | ~300 | Low -- cobra command wrappers |
| Tests | ~400 | Standard |
| **Total** | **~1,310** | |

## Sources

- Source code inspection: `pkg/codex/dispatch.go`, `pkg/codex/worker.go`, `pkg/codex/process_tracker.go`, `cmd/circuit_breaker.go`, `cmd/gate.go`, `cmd/fixer_dispatch.go`, `cmd/immune.go`, `cmd/autopilot.go`, `cmd/codex_dispatch_contract.go`, `pkg/colony/colony.go`, `pkg/colony/state_machine.go`, `pkg/storage/storage.go`
- Existing go.mod: `golang.org/x/sync v0.20.0`, `github.com/spf13/cobra v1.10.2`, `modernc.org/sqlite v1.50.0`
- Suture supervisor trees: [github.com/thejerf/suture](https://github.com/thejerf/suture) -- evaluated and rejected (see "What NOT to Add")
- oklog/run: [github.com/oklog/run](https://github.com/oklog/run) -- evaluated and rejected (see "What NOT to Add")
- errgroup SetLimit: [pkg.go.dev/golang.org/x/sync/errgroup](https://pkg.go.dev/golang.org/x/sync/errgroup) -- already in project, no changes needed
