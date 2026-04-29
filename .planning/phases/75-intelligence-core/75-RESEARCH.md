# Phase 75: Intelligence Core - Research

**Researched:** 2026-04-29
**Domain:** Go CLI (memory pipeline + parallel worker dispatch)
**Confidence:** HIGH

## Summary

Phase 75 has two distinct workstreams: (1) wiring trust scoring into the `memory-capture` command and continue ceremony playbooks, and (2) implementing a circuit breaker for parallel worker dispatch. Both are moderate-complexity changes to existing, well-tested code.

The trust scoring engine (`pkg/memory/trust.go`) and observation capture (`pkg/memory/observe.go`) are fully implemented and tested. The `learning-observe` command already accepts `--source-type` and `--evidence-type` flags. The gap is that the simpler `memory-capture` command (used by playbooks) does not expose these flags, so all playbook-sourced learnings get the lowest trust score (observation/anecdotal). Fixing this requires adding two flags to one cobra command and switching from `Capture()` to `CaptureWithTrust()`, then updating two playbook files.

The circuit breaker is a new construct with no existing implementation. It must integrate into both `dispatchCodexBuildWorkersInRepo` (serial dispatch) and `dispatchCodexBuildWorkers` (worktree/parallel dispatch) in `cmd/codex_build_worktree.go`. The design calls for per-worker-instance consecutive failure tracking with per-wave reset and task redistribution to same-caste peers.

**Primary recommendation:** Implement trust scoring flags first (small, mechanical change), then circuit breaker (new logic requiring careful integration with existing dispatch flow).

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Extend `memory-capture` with `--source-type` and `--evidence-type` flags. Keep existing defaults (`observation`/`anecdotal`) so unflagged callers are unaffected. Playbooks pass explicit flags for better scores.
- **D-02:** Playbook-driven source/evidence types -- each ceremony step explicitly passes the appropriate flags when calling `memory-capture`. No auto-detection from colony state.
- **D-03:** Continue ceremony uses `--source-type success_pattern --evidence-type multi_phase` for learnings extracted from completed work.
- **D-04:** Consecutive failure count triggers the breaker. A worker fails N times consecutively (configurable threshold, default 3). A single success resets the counter.
- **D-05:** When the breaker trips, pending tasks for that worker are redistributed to other workers of the same caste. No tasks are lost.
- **D-06:** Per-wave reset -- the breaker resets at the start of each new build wave. A worker that tripped in wave 1 gets a fresh chance in wave 2.
- **D-07:** Per-worker instance granularity -- each worker instance (e.g., Builder-Mason-67) has its own breaker. Other workers of the same caste are unaffected.
- **D-08:** Circuit breaker applies in both parallel modes (in-repo and worktree).
- **D-09:** Build ceremony does NOT capture learnings -- that stays in continue only. Clean separation: build is for building, continue is for reflecting.
- **D-10:** Continue ceremony captures learnings with trust scoring via the extended `memory-capture` command. No other changes to the continue flow.

### Claude's Discretion
- Exact consecutive failure threshold (default 3, adjustable via flag)
- How the circuit breaker state is stored (colony state field, in-memory only, etc.)
- Visual rendering of circuit breaker events in build output
- Whether to log a summary of tripped workers at wave end
- Test coverage approach for the circuit breaker

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| INTEL-04 | Bayesian confidence scoring restored for wisdom pipeline (40/35/25 weighted, 60-day half-life) | Trust engine exists in `pkg/memory/trust.go`. `CaptureWithTrust()` exists in `observe.go`. Gap: `memory-capture` command missing flags. Fix: add `--source-type`/`--evidence-type` flags, switch to `CaptureWithTrust()`, update playbooks. |
| INTEL-05 | Circuit breaker prevents cascade failure across parallel workers | No existing implementation. New `CircuitBreaker` struct needed in `cmd/`. Integration points: `dispatchCodexBuildWorkersInRepo` (serial) and `dispatchCodexBuildWorkers` (worktree parallel). Per-worker-instance, per-wave reset, task redistribution. |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Trust score computation | Go runtime (pkg layer) | -- | Pure calculation in `pkg/memory/trust.go`, no external dependency |
| Observation capture with trust | Go runtime (cmd layer) | -- | `memory-capture` cobra command in `cmd/learning.go` calls `CaptureWithTrust()` |
| Circuit breaker state | Go runtime (in-memory) | -- | Per-build-lifetime state, no persistence needed (per-wave reset) |
| Circuit breaker enforcement | Go runtime (cmd layer) | -- | Integrated into `dispatchCodexBuildWorkers*` functions |
| Task redistribution | Go runtime (cmd layer) | -- | Same-caste peer selection at dispatch time |
| Playbook flag wiring | Wrapper markdown | -- | Playbooks pass flags to CLI; no runtime logic |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib | (project go.mod) | All logic | Zero-new-deps principle from REQUIREMENTS.md |
| cobra | (project go.mod) | CLI commands | All commands use cobra |
| pkg/memory | internal | Trust scoring + observation capture | Already implemented, tested |
| pkg/codex | internal | Worker dispatch infrastructure | Already implemented |
| pkg/colony | internal | Colony state types | Already implemented |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| pkg/storage | internal | JSON persistence | State reads/writes |
| pkg/events | internal | Event bus for observations | Learning pipeline integration |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| In-memory circuit breaker | Colony-state-persisted breaker | In-memory is simpler; per-wave reset means no persistence needed. Colony-state adds complexity for no benefit since breaker resets each wave. |
| Separate package for breaker | Inline in cmd/ | Too small for a package. A struct in `cmd/codex_build.go` or a new `cmd/circuit_breaker.go` is cleaner. |

**Installation:**
No new dependencies. All changes use existing Go stdlib + project packages.

**Version verification:** No external packages to verify.

## Architecture Patterns

### System Architecture Diagram

```
                    Trust Scoring (INTEL-04)
                    =========================

Continue Playbook                Go Runtime
-----------------               ----------
continue-advance.md
  Step 2.5:                       cmd/learning.go
  aether memory-capture            memoryCaptureCmd
    --type "learning"              --source-type (NEW FLAG)
    --content "$claim"             --evidence-type (NEW FLAG)
         |                                |
         v                                v
                              pkg/memory/observe.go
                                CaptureWithTrust()
                                      |
                                      v
                              pkg/memory/trust.go
                                Calculate()
                                (40/35/25 formula)
                                      |
                                      v
                              learning-observations.json
                              (trust_score field populated)



                    Circuit Breaker (INTEL-05)
                    =========================

Build Wave Dispatch               Go Runtime
-----------------               ----------
build-wave.md                    cmd/codex_build_worktree.go
  Step 5.1:                       dispatchCodexBuildWorkers()
  Spawn workers                         |
                                        v
                              CircuitBreaker (NEW struct)
                              .Allow(workerName) -> bool
                              .RecordSuccess(workerName)
                              .RecordFailure(workerName)
                              .Reset()  [per-wave]
                                        |
                              +---------+---------+
                              |                   |
                              v                   v
                        In-Repo Mode      Worktree Mode
                        (serial loop)     (parallel goroutines)
                              |                   |
                              v                   v
                        Worker invoked    Worker invoked
                              |                   |
                              v                   v
                        Result returned   Result returned
                              |                   |
                              v                   v
                        .RecordSuccess    .RecordSuccess
                        or .RecordFailure or .RecordFailure
                              |                   |
                              +---------+---------+
                                        |
                                        v
                              If tripped: skip worker
                              Redistribute task to
                              same-caste peer
```

### Recommended Project Structure

```
cmd/
├── circuit_breaker.go       # NEW: CircuitBreaker struct + methods
├── codex_build.go           # MODIFIED: integrate breaker into dispatch
├── codex_build_worktree.go  # MODIFIED: integrate breaker into both dispatch paths
├── learning.go              # MODIFIED: add --source-type and --evidence-type to memory-capture
└── circuit_breaker_test.go  # NEW: tests for breaker logic

.aether/docs/command-playbooks/
├── continue-advance.md      # MODIFIED: add --source-type/--evidence-type to memory-capture calls
└── continue-full.md         # MODIFIED: add --source-type/--evidence-type to memory-capture calls

pkg/memory/
├── trust.go                 # UNCHANGED
├── trust_test.go            # UNCHANGED
├── observe.go               # UNCHANGED
└── observe_test.go          # UNCHANGED
```

### Pattern 1: Trust Score Flag Wiring
**What:** Add `--source-type` and `--evidence-type` flags to `memory-capture` command, switch from `Capture()` to `CaptureWithTrust()`.
**When to use:** When the `memory-capture` command needs to produce observations with meaningful trust scores.
**Example:**
```go
// Source: cmd/learning.go (existing learning-observe pattern)
var memoryCaptureCmd = &cobra.Command{
    Use:   "memory-capture [content]",
    Short: "Capture a memory observation (simplified learning-observe)",
    Args:  cobra.MaximumNArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        // ... existing store/content/type logic ...

        sourceType, _ := cmd.Flags().GetString("source-type")
        if sourceType == "" {
            sourceType = "observation"  // default preserved
        }
        evidenceType, _ := cmd.Flags().GetString("evidence-type")
        if evidenceType == "" {
            evidenceType = "anecdotal"  // default preserved
        }

        result, err := obsService.CaptureWithTrust(ctx, content, obsType, "unknown", sourceType, evidenceType)
        // ... existing output logic ...
    },
}
```

### Pattern 2: Circuit Breaker Struct
**What:** Simple counter-based breaker with per-worker tracking, configurable threshold, and per-wave reset.
**When to use:** During parallel worker dispatch to prevent cascade failures.
**Example:**
```go
// cmd/circuit_breaker.go
type CircuitBreaker struct {
    mu         sync.Mutex
    threshold  int
    failures   map[string]int  // workerName -> consecutive failures
    tripped    map[string]bool  // workerName -> tripped state
}

func NewCircuitBreaker(threshold int) *CircuitBreaker {
    return &CircuitBreaker{
        threshold: threshold,
        failures:  make(map[string]int),
        tripped:   make(map[string]bool),
    }
}

func (cb *CircuitBreaker) Allow(workerName string) bool {
    cb.mu.Lock()
    defer cb.mu.Unlock()
    return !cb.tripped[workerName]
}

func (cb *CircuitBreaker) RecordSuccess(workerName string) {
    cb.mu.Lock()
    defer cb.mu.Unlock()
    cb.failures[workerName] = 0
    cb.tripped[workerName] = false
}

func (cb *CircuitBreaker) RecordFailure(workerName string) bool {
    cb.mu.Lock()
    defer cb.mu.Unlock()
    cb.failures[workerName]++
    if cb.failures[workerName] >= cb.threshold {
        cb.tripped[workerName] = true
        return true  // tripped
    }
    return false  // not yet tripped
}

func (cb *CircuitBreaker) Reset() {
    cb.mu.Lock()
    defer cb.mu.Unlock()
    cb.failures = make(map[string]int)
    cb.tripped = make(map[string]bool)
}

func (cb *CircuitBreaker) TrippedWorkers() []string {
    cb.mu.Lock()
    defer cb.mu.Unlock()
    var names []string
    for name, tripped := range cb.tripped {
        if tripped {
            names = append(names, name)
        }
    }
    sort.Strings(names)
    return names
}
```

### Pattern 3: Breaker Integration into Dispatch
**What:** Check `cb.Allow()` before invoking a worker, record result after, redistribute if tripped.
**When to use:** In both `dispatchCodexBuildWorkersInRepo` and `dispatchCodexBuildWorkers`.
**Example:**
```go
// In dispatchCodexBuildWorkersInRepo, before invoking worker:
if !cb.Allow(dispatch.WorkerName) {
    // Find same-caste peer
    peer := findSameCastePeer(dispatches, dispatch, cb)
    if peer != nil {
        // Redistribute task to peer
        emitBuildCeremonyWorkerSkipped(dispatch, wave, "circuit breaker tripped")
        dispatch = *peer
    } else {
        // No peer available -- mark as failed
        emitBuildCeremonyWorkerSkipped(dispatch, wave, "circuit breaker tripped, no peer available")
        waveResults = append(waveResults, codex.DispatchResult{
            WorkerName: dispatch.WorkerName,
            Status:     "failed",
            Error:      fmt.Errorf("circuit breaker tripped, no same-caste peer for redistribution"),
        })
        continue
    }
}

// After worker result:
if dr.Status == "completed" {
    cb.RecordSuccess(dispatch.WorkerName)
} else {
    cb.RecordFailure(dispatch.WorkerName)
}

// At wave boundary:
cb.Reset()
```

### Anti-Patterns to Avoid
- **Don't persist circuit breaker state:** Per-wave reset means no need for colony-state persistence. In-memory is sufficient and simpler.
- **Don't block on tripped workers:** The breaker should skip and redistribute, not block the entire build.
- **Don't change trust.go or observe.go:** Both files are correct and well-tested. Changes belong in `cmd/learning.go` (flag wiring) and playbooks.
- **Don't use the breaker for specialist castes (oracle, architect, etc.):** The breaker protects wave workers. Specialist pre-wave workers (oracle, architect, archaeologist, ambassador) already have their own non-blocking failure handling.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Trust score calculation | Custom math | `memory.Calculate()` + `memory.CaptureWithTrust()` | Already implemented with correct 40/35/25 formula, 60-day half-life, 7 tiers, floor at 0.2. Thoroughly tested. |
| Observation dedup | Custom hash matching | `ObservationService.CaptureWithTrust()` | SHA-256 content hash dedup, observation count tracking, event bus publish all built in. |
| Worker dispatch | Custom spawning | `dispatchCodexBuildWorkers*` functions | Already handles wave grouping, worktree allocation, result collection, status tracking. |
| Parallel goroutine coordination | Custom WaitGroup | Existing `sync.WaitGroup` pattern in `dispatchCodexBuildWorkers` | Already proven correct with mutex-protected root operations. |

**Key insight:** The trust scoring engine is complete and tested. The observation capture with trust is complete and tested. The only gap is the `memory-capture` CLI command not exposing the flags. For the circuit breaker, the pattern is simple enough (consecutive counter + threshold) that a library would be over-engineering -- a small struct in `cmd/` is the right scope.

## Common Pitfalls

### Pitfall 1: Breaking existing `memory-capture` callers
**What goes wrong:** Adding required flags to `memory-capture` breaks all existing playbook calls that don't pass them.
**Why it happens:** Cobra flags default to empty string, but if the code switches to `CaptureWithTrust()` without checking for empty flags, it may produce different behavior.
**How to avoid:** Keep the default values as `"observation"` and `"anecdotal"` when flags are not provided (exactly matching current `Capture()` behavior). Use `GetString()` with empty-string fallback.
**Warning signs:** Existing tests fail after adding flags.

### Pitfall 2: Circuit breaker race conditions in worktree mode
**What goes wrong:** In worktree parallel mode, goroutines access the breaker concurrently without synchronization.
**Why it happens:** `dispatchCodexBuildWorkers` spawns goroutines per worker in a wave. Without mutex, concurrent `RecordFailure` calls corrupt the counter.
**How to avoid:** The `CircuitBreaker` struct MUST use `sync.Mutex` for all state access. All public methods must be goroutine-safe.
**Warning signs:** `go test -race` fails on dispatch tests.

### Pitfall 3: Task redistribution loop
**What goes wrong:** When redistributing a tripped worker's task to a peer, that peer is also tripped, causing infinite redistribution.
**Why it happens:** Naive "find any same-caste peer" without checking if the peer is also tripped.
**How to avoid:** `findSameCastePeer` must check `cb.Allow(peerName)` before selecting. If all peers are tripped, mark the task as failed rather than looping.
**Warning signs:** Build hangs or panics during wave execution.

### Pitfall 4: Playbook memory-capture calls without new flags
**What goes wrong:** After adding flags to `memory-capture`, existing playbook calls that don't use them continue producing low-trust observations.
**Why it happens:** The continue-advance.md and continue-full.md playbooks have multiple `aether memory-capture` calls. If only some are updated, the scoring is inconsistent.
**How to avoid:** Audit ALL `memory-capture` calls in both playbooks. Per D-09, build playbooks should NOT be changed (build does not capture learnings). Only continue playbooks get the flags.

### Pitfall 5: Circuit breaker resets at wrong time
**What goes wrong:** Reset happens after each worker instead of at wave boundary, defeating the purpose.
**Why it happens:** Confusion about "per-wave" reset vs "per-worker" reset.
**How to avoid:** `Reset()` clears ALL worker state. It must be called exactly once between waves (before the next wave starts), not after each worker completes. The per-worker success reset is `RecordSuccess()` (resets only that worker's counter).
**Warning signs:** Breaker never trips because counters reset too often.

## Code Examples

Verified patterns from existing codebase:

### Existing learning-observe command with trust flags (reference implementation)
```go
// Source: cmd/learning.go lines 35-49
sourceType, _ := cmd.Flags().GetString("source-type")
if sourceType == "" {
    sourceType = "observation"
}
evidenceType, _ := cmd.Flags().GetString("evidence-type")
if evidenceType == "" {
    evidenceType = "anecdotal"
}

bus := events.NewBus(store, events.DefaultConfig())
obsService := memory.NewObservationService(store, bus)

ctx, cancel := timeoutCtx(cmd)
defer cancel()
result, err := obsService.CaptureWithTrust(ctx, content, wisdomType, colonyName, sourceType, evidenceType)
```

### Existing memory-capture command (needs modification)
```go
// Source: cmd/learning.go lines 181-218
var memoryCaptureCmd = &cobra.Command{
    Use:   "memory-capture [content]",
    Short: "Capture a memory observation (simplified learning-observe)",
    Args:  cobra.MaximumNArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        // ... store init ...
        content := mustGetStringCompat(cmd, args, "content", 0)
        obsType, _ := cmd.Flags().GetString("type")
        if obsType == "" {
            obsType = "observation"
        }

        bus := events.NewBus(store, events.DefaultConfig())
        obsService := memory.NewObservationService(store, bus)

        ctx, cancel := timeoutCtx(cmd)
        defer cancel()
        // THIS LINE needs to change from Capture() to CaptureWithTrust():
        result, err := obsService.Capture(ctx, content, obsType, "unknown")
        // ... output ...
    },
}
```

### Existing flag registration pattern
```go
// Source: cmd/learning.go lines 220-226
func init() {
    learningObserveCmd.Flags().String("content", "", "Observation content (required)")
    learningObserveCmd.Flags().String("type", "", "Wisdom type (required)")
    learningObserveCmd.Flags().String("colony-name", "", "Colony name")
    learningObserveCmd.Flags().String("source-type", "", "Source type")
    learningObserveCmd.Flags().String("evidence-type", "", "Evidence type")
    // ...
    memoryCaptureCmd.Flags().String("content", "", "Observation content (required)")
    memoryCaptureCmd.Flags().String("type", "", "Wisdom type (default: observation)")
    // NEW FLAGS GO HERE:
    // memoryCaptureCmd.Flags().String("source-type", "", "Source type (default: observation)")
    // memoryCaptureCmd.Flags().String("evidence-type", "", "Evidence type (default: anecdotal)")
}
```

### Existing wave dispatch pattern (breaker integration point)
```go
// Source: cmd/codex_build_worktree.go lines 330-398
func dispatchCodexBuildWorkersInRepo(ctx context.Context, phase colony.Phase, dispatches []codex.WorkerDispatch, invoker codex.WorkerInvoker, parallelMode colony.ParallelMode) ([]codex.DispatchResult, error) {
    waves := codex.GroupByWave(dispatches)
    // ...
    for _, wave := range waveNumbers {
        waveDispatches := waves[wave]
        // BREAKER RESET GOES HERE (before wave starts)
        for _, dispatch := range waveDispatches {
            // BREAKER CHECK GOES HERE (before invoking worker)
            result, err := invokeCodexWorkerWithRuntimeProgress(ctx, invoker, cfg, dispatch, wave)
            // BREAKER RECORD GOES HERE (after result)
        }
        // WAVE END
    }
    return results, nil
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `memory-capture` uses `Capture()` with defaults | `memory-capture` should use `CaptureWithTrust()` with explicit flags | Phase 75 | Higher trust scores for playbook-sourced learnings |
| No failure protection in parallel dispatch | Circuit breaker per worker instance | Phase 75 | Cascade failure prevention |
| All observations scored as observation/anecdotal | Explicit source/evidence types from ceremony context | Phase 75 | More meaningful trust scores for wisdom pipeline |

**Deprecated/outdated:**
- None -- this phase restores/introduces capabilities, doesn't replace anything.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The `memory-capture` command is only called from playbooks (continue-advance.md, continue-full.md, build-wave.md, build-verify.md) and never from Go code directly | Trust Scoring Integration | If Go code also calls it, those call sites would also benefit from flags but are out of scope per D-02 |
| A2 | The circuit breaker does not need to be persisted across builds or waves | Circuit Breaker Design | If users expect breaker state to survive between builds, in-memory-only would be insufficient. Per D-06 (per-wave reset), in-memory is correct. |
| A3 | Task redistribution can always find a same-caste peer | Circuit Breaker Design | If a wave has only one worker of a caste and it trips, there's no peer. The plan must handle this gracefully (mark as failed, don't loop). |
| A4 | `go test -race` is part of the existing test suite | Testing | If race detection isn't run, concurrent breaker bugs could go undetected. |

## Open Questions

1. **Circuit breaker threshold flag placement**
   - What we know: D-04 says "configurable threshold, default 3" at Claude's discretion.
   - What's unclear: Should this be a `memory-capture`-level flag, a `build` command flag, or a colony-state setting?
   - Recommendation: A `--circuit-breaker-threshold` flag on the `build` command (not persisted). Simple, per-build configuration. Default 3.

2. **Visual rendering of circuit breaker events**
   - What we know: Claude's discretion per CONTEXT.md.
   - What's unclear: How verbose should the breaker output be? Silent skip? Warning line? Full block?
   - Recommendation: A single warning line per tripped worker: `"Circuit breaker: {workerName} tripped after {N} consecutive failures -- redistributing to {peerName}"`. Plus a summary at wave end if any breakers tripped.

3. **Should the breaker be exposed as a subcommand for introspection?**
   - What we know: Not mentioned in CONTEXT.md decisions.
   - What's unclear: Users might want to see breaker state during a build.
   - Recommendation: Not in this phase. The build output should surface tripped workers, but a dedicated subcommand is over-engineering for v1.

## Environment Availability

> Step 2.6: SKIPPED (no external dependencies -- all changes are Go code and markdown playbooks within the existing repo)

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + `go test -race` |
| Config file | none (standard Go test) |
| Quick run command | `go test ./cmd/ -run TestCircuitBreaker -count=1` |
| Full suite command | `go test ./... -race` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| INTEL-04 | `memory-capture --source-type success_pattern --evidence-type multi_phase` produces observation with correct trust score | unit | `go test ./cmd/ -run TestMemoryCaptureWithTrust -count=1` | No -- Wave 0 |
| INTEL-04 | `memory-capture` without flags still uses observation/anecdotal defaults | unit | `go test ./cmd/ -run TestMemoryCaptureDefaults -count=1` | No -- Wave 0 |
| INTEL-05 | Circuit breaker trips after N consecutive failures | unit | `go test ./cmd/ -run TestCircuitBreakerTrip -count=1` | No -- Wave 0 |
| INTEL-05 | Circuit breaker resets on success | unit | `go test ./cmd/ -run TestCircuitBreakerReset -count=1` | No -- Wave 0 |
| INTEL-05 | Circuit breaker resets between waves | unit | `go test ./cmd/ -run TestCircuitBreakerWaveReset -count=1` | No -- Wave 0 |
| INTEL-05 | Circuit breaker redistributes to same-caste peer | unit | `go test ./cmd/ -run TestCircuitBreakerRedistribution -count=1` | No -- Wave 0 |
| INTEL-05 | Circuit breaker handles no-peer-available gracefully | unit | `go test ./cmd/ -run TestCircuitBreakerNoPeer -count=1` | No -- Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./pkg/memory/... ./cmd/ -run "TestMemory|TestCircuit|TestTrust" -count=1`
- **Per wave merge:** `go test ./... -race`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `cmd/circuit_breaker_test.go` -- tests for CircuitBreaker struct (Allow, RecordSuccess, RecordFailure, Reset, TrippedWorkers)
- [ ] `cmd/learning_flags_test.go` or additions to `cmd/memory_test.go` -- tests for memory-capture with new trust flags
- [ ] Framework install: none needed (Go testing is stdlib)

## Security Domain

> No security enforcement needed for this phase. Changes are:
> - CLI flag additions (no new attack surface)
> - In-memory state tracking (no persistence, no user input)
> - Playbook markdown edits (no executable code)

The circuit breaker operates on worker dispatch names (internal identifiers), not on user-supplied input. No injection risk.

## Sources

### Primary (HIGH confidence)
- `pkg/memory/trust.go` -- Trust scoring engine implementation, verified via code read
- `pkg/memory/observe.go` -- Observation capture with `CaptureWithTrust()`, verified via code read
- `cmd/learning.go` -- `memory-capture` and `learning-observe` commands, verified via code read
- `cmd/codex_build_worktree.go` -- Worker dispatch functions, verified via code read
- `cmd/codex_build.go` -- Build manifest and dispatch planning, verified via code read
- `.aether/docs/command-playbooks/continue-advance.md` -- Continue ceremony learning extraction, verified via code read
- `.aether/docs/command-playbooks/build-wave.md` -- Build wave dispatch flow, verified via code read

### Secondary (MEDIUM confidence)
- `.planning/REQUIREMENTS.md` -- INTEL-04 and INTEL-05 requirement definitions
- `75-CONTEXT.md` -- User decisions D-01 through D-10

### Tertiary (LOW confidence)
- None -- all research was conducted via direct code inspection

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - all code read directly from codebase, no external dependencies
- Architecture: HIGH - dispatch flow and memory pipeline fully understood through code reading
- Pitfalls: HIGH - identified from concrete code patterns (race conditions in goroutines, flag defaults)

**Research date:** 2026-04-29
**Valid until:** 30 days (stable codebase, no external dependencies)
