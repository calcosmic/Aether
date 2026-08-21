# Phase 75: Intelligence Core - Pattern Map

**Mapped:** 2026-04-29
**Files analyzed:** 7
**Analogs found:** 7 / 7

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `cmd/circuit_breaker.go` | utility (new struct) | event-driven | `pkg/memory/trust.go` (struct + methods pattern) | role-match |
| `cmd/circuit_breaker_test.go` | test | transform | `pkg/memory/trust_test.go` | exact |
| `cmd/learning.go` | command (modified) | request-response | `cmd/learning.go` lines 12-66 (learning-observe cmd) | self-analog |
| `cmd/codex_build_worktree.go` | dispatch (modified) | event-driven | `cmd/codex_build_worktree.go` lines 147-319 | self-analog |
| `cmd/codex_build.go` | dispatch (modified) | event-driven | `cmd/codex_build.go` lines 1-100 | self-analog |
| `.aether/docs/command-playbooks/continue-advance.md` | playbook (modified) | request-response | `.aether/docs/command-playbooks/continue-advance.md` lines 95-113 | self-analog |
| `.aether/docs/command-playbooks/continue-full.md` | playbook (modified) | request-response | `.aether/docs/command-playbooks/continue-full.md` lines 1056-1073 | self-analog |

## Pattern Assignments

### `cmd/circuit_breaker.go` (utility, event-driven) -- NEW FILE

**Analog:** `pkg/memory/trust.go` (struct with exported methods, pure logic, no I/O)

**Imports pattern** -- follow cmd/ package convention from `cmd/codex_build_worktree.go` lines 1-18:
```go
package cmd

import (
    "fmt"
    "sort"
    "sync"

    "github.com/calcosmic/Aether/pkg/codex"
)
```

**Struct pattern** -- mirror `TrustInput`/`TrustResult` from `pkg/memory/trust.go` lines 26-40:
```go
// TrustInput holds the inputs for trust score calculation.
type TrustInput struct {
    SourceType string
    Evidence   string
    DaysSince  int
}

// TrustResult holds the output of trust score calculation.
type TrustResult struct {
    Score         float64
    SourceScore   float64
    EvidenceScore float64
    ActivityScore float64
    Tier          string
    TierIndex     int
}
```

**Core pattern** -- new `CircuitBreaker` struct with mutex-protected state (from RESEARCH.md Pattern 2):
```go
type CircuitBreaker struct {
    mu         sync.Mutex
    threshold  int
    failures   map[string]int  // workerName -> consecutive failures
    tripped    map[string]bool  // workerName -> tripped state
}

func NewCircuitBreaker(threshold int) *CircuitBreaker { ... }
func (cb *CircuitBreaker) Allow(workerName string) bool { ... }
func (cb *CircuitBreaker) RecordSuccess(workerName string) { ... }
func (cb *CircuitBreaker) RecordFailure(workerName string) bool { ... }
func (cb *CircuitBreaker) Reset() { ... }
func (cb *CircuitBreaker) TrippedWorkers() []string { ... }
```

**Helper for redistribution** -- follows `summarizeRunStatus()` from `cmd/spawn_runs.go` lines 34-49 (switch/case status categorization):
```go
func findSameCastePeer(dispatches []codex.WorkerDispatch, current codex.WorkerDispatch, cb *CircuitBreaker) *codex.WorkerDispatch {
    for i := range dispatches {
        d := &dispatches[i]
        if d.WorkerName == current.WorkerName { continue }
        if d.Caste != current.Caste { continue }
        if !cb.Allow(d.WorkerName) { continue }  // peer must not be tripped
        return d
    }
    return nil
}
```

---

### `cmd/circuit_breaker_test.go` (test) -- NEW FILE

**Analog:** `pkg/memory/trust_test.go` (table-driven tests, epsilon comparison, subtests via `t.Run`)

**Test structure pattern** -- from `pkg/memory/trust_test.go` lines 1-48:
```go
package cmd

import (
    "fmt"
    "sync"
    "testing"

    "github.com/calcosmic/Aether/pkg/codex"
)

func TestCircuitBreaker_Trip(t *testing.T) {
    cb := NewCircuitBreaker(3)
    // Record failures up to threshold
    cb.RecordFailure("Builder-Mason-67") // 1
    cb.RecordFailure("Builder-Mason-67") // 2
    tripped := cb.RecordFailure("Builder-Mason-67") // 3 -> should trip
    if !tripped { ... }
    if cb.Allow("Builder-Mason-67") { ... }  // should be false
}
```

**Table-driven subtests pattern** -- from `pkg/memory/trust_test.go` lines 11-47:
```go
tests := []struct {
    name           string
    failures       int
    wantTripped    bool
    wantAllow      bool
}{
    {"below_threshold", 2, false, true},
    {"at_threshold", 3, true, false},
    {"above_threshold", 5, true, false},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) { ... })
}
```

**Concurrency test pattern** -- must use `go test -race`, following `cmd/codex_build_worktree_test.go` goroutine patterns (lines 172-314 use `sync.WaitGroup`):
```go
func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
    cb := NewCircuitBreaker(3)
    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            cb.RecordFailure("Builder-Mason-67")
            cb.Allow("Builder-Mason-67")
        }()
    }
    wg.Wait()
}
```

**Test helper usage** -- from `cmd/build_flow_cmds_test.go` lines 17-41:
```go
func setupBuildFlowTest(t *testing.T) string {
    t.Helper()
    tmpDir := t.TempDir()
    dataDir := tmpDir + "/.aether/data"
    if err := os.MkdirAll(dataDir, 0755); err != nil { ... }
    // ...
    store = s
    stdout = &bytes.Buffer{}
    stderr = &bytes.Buffer{}
}
```

---

### `cmd/learning.go` (command, request-response) -- MODIFIED

**Analog:** `cmd/learning.go` lines 12-66 (`learningObserveCmd` -- the reference implementation with trust flags)

**Flag registration pattern** -- from `cmd/learning.go` lines 220-231:
```go
func init() {
    learningObserveCmd.Flags().String("source-type", "", "Source type")
    learningObserveCmd.Flags().String("evidence-type", "", "Evidence type")
    // ...
    memoryCaptureCmd.Flags().String("content", "", "Observation content (required)")
    memoryCaptureCmd.Flags().String("type", "", "Wisdom type (default: observation)")
    // NEW FLAGS GO HERE
    memoryCaptureCmd.Flags().String("source-type", "", "Source type (default: observation)")
    memoryCaptureCmd.Flags().String("evidence-type", "", "Evidence type (default: anecdotal)")
}
```

**Trust flag reading pattern** -- from `cmd/learning.go` lines 35-42 (exact pattern to copy into memoryCaptureCmd):
```go
sourceType, _ := cmd.Flags().GetString("source-type")
if sourceType == "" {
    sourceType = "observation"
}
evidenceType, _ := cmd.Flags().GetString("evidence-type")
if evidenceType == "" {
    evidenceType = "anecdotal"
}
```

**CaptureWithTrust call pattern** -- from `cmd/learning.go` lines 44-49:
```go
bus := events.NewBus(store, events.DefaultConfig())
obsService := memory.NewObservationService(store, bus)

ctx, cancel := timeoutCtx(cmd)
defer cancel()
result, err := obsService.CaptureWithTrust(ctx, content, wisdomType, colonyName, sourceType, evidenceType)
```

**Output pattern** -- from `cmd/learning.go` lines 55-63 (learning-observe) vs lines 211-215 (memory-capture). The memory-capture output should be extended to match learning-observe's richer output:
```go
// learning-observe output (rich):
outputOK(map[string]interface{}{
    "captured":           true,
    "is_new":             result.IsNew,
    "observation_id":     result.Observation.ContentHash,
    "observation_count":  result.Observation.ObservationCount,
    "trust_score":        result.Observation.TrustScore,
    "promotion_eligible": result.PromotionEligible,
    "promotion_reason":   result.PromotionReason,
})

// memory-capture current output (minimal -- extend this):
outputOK(map[string]interface{}{
    "captured":    true,
    "is_new":      result.IsNew,
    "trust_score": result.Observation.TrustScore,
})
```

**Error handling pattern** -- from `cmd/learning.go` lines 50-53:
```go
if err != nil {
    outputError(2, fmt.Sprintf("failed to capture observation: %v", err), nil)
    return nil
}
```

---

### `cmd/codex_build_worktree.go` (dispatch, event-driven) -- MODIFIED

**Analog:** `cmd/codex_build_worktree.go` lines 147-319 (both dispatch functions)

**In-repo dispatch integration point** -- from `cmd/codex_build_worktree.go` lines 321-400:
```go
func dispatchCodexBuildWorkersInRepo(ctx context.Context, phase colony.Phase, dispatches []codex.WorkerDispatch, invoker codex.WorkerInvoker, parallelMode colony.ParallelMode) ([]codex.DispatchResult, error) {
    waves := codex.GroupByWave(dispatches)
    // ... wave iteration ...
    for _, wave := range waveNumbers {
        waveDispatches := waves[wave]
        emitBuildCeremonyWaveStart(phase, wave, waveDispatches, parallelMode)
        // cb.Reset() GOES HERE -- before processing workers in this wave
        for _, dispatch := range waveDispatches {
            // cb.Allow(dispatch.WorkerName) CHECK GOES HERE
            // ... invoke worker ...
            // cb.RecordSuccess/RecordFailure GOES HERE based on dr.Status
        }
        emitBuildCeremonyWaveEnd(phase, wave, waveResults)
    }
    return results, nil
}
```

**Worktree dispatch integration point** -- from `cmd/codex_build_worktree.go` lines 147-319:
```go
func dispatchCodexBuildWorkers(ctx context.Context, root string, phase colony.Phase, dispatches []codex.WorkerDispatch, invoker codex.WorkerInvoker, startedAt time.Time, parallelMode colony.ParallelMode) ([]codex.DispatchResult, error) {
    // ... same wave loop structure ...
    for _, wave := range waveNumbers {
        waveDispatches := waves[wave]
        emitBuildCeremonyWaveStart(phase, wave, waveDispatches, parallelMode)
        // cb.Reset() GOES HERE
        var wg sync.WaitGroup
        for idx, dispatch := range waveDispatches {
            wg.Add(1)
            go func(i int, dispatch codex.WorkerDispatch) {
                defer wg.Done()
                // cb.Allow(dispatch.WorkerName) CHECK GOES HERE (inside goroutine)
                // ... worker invocation ...
                // cb.RecordSuccess/RecordFailure GOES HERE (inside goroutine)
                // NOTE: CircuitBreaker methods are mutex-protected, safe for concurrent use
            }(idx, dispatch)
        }
        wg.Wait()
        emitBuildCeremonyWaveEnd(phase, wave, waveResults)
    }
}
```

**Worker status recording pattern** -- from `cmd/codex_build_worktree.go` lines 369-395:
```go
dr := codex.DispatchResult{WorkerName: dispatch.WorkerName}
if err != nil {
    dr.Status = "failed"
    dr.Error = err
} else {
    dr.Status = result.Status
    // ...
}
// Circuit breaker recording goes after this:
// if dr.Status == "completed" { cb.RecordSuccess(...) } else { cb.RecordFailure(...) }
```

**Ceremony emission pattern for skipped workers** -- from `cmd/ceremony_emitter.go` lines 407-437:
```go
// Pattern for new emitBuildCeremonyWorkerSkipped:
func emitBuildCeremonyWorkerSkipped(dispatch codex.WorkerDispatch, wave int, reason string) {
    payload := ceremonyPayloadForDispatch(dispatch, wave, "skipped", reason)
    payload.Blockers = []string{reason}
    emitBuildCeremony(events.CeremonyTopicBuildSpawn, payload)
}
```

---

### `cmd/codex_build.go` (dispatch, event-driven) -- MODIFIED

**Analog:** `cmd/codex_build.go` lines 1-100 (build dispatch types and manifest)

**Type pattern** -- from `cmd/codex_build.go` lines 19-70:
```go
type codexBuildDispatch struct {
    Stage         string   `json:"stage"`
    Wave          int      `json:"wave,omitempty"`
    Caste         string   `json:"caste"`
    Name          string   `json:"name"`
    Task          string   `json:"task"`
    Status        string   `json:"status"`
    // ... more fields ...
}
```

**Integration point:** The `codex_build.go` file is where the build command's `RunE` function creates and passes the `CircuitBreaker` to the dispatch functions. The breaker is instantiated at the start of the build and passed as a parameter to `dispatchCodexBuildWorkers*`.

---

### `.aether/docs/command-playbooks/continue-advance.md` (playbook, request-response) -- MODIFIED

**Analog:** `.aether/docs/command-playbooks/continue-advance.md` lines 95-113 (Step 2.5 memory-capture calls)

**Current memory-capture call** -- line 106:
```bash
aether memory-capture --type "learning" --content "$claim" 2>/dev/null || true
```

**Updated call with trust flags** (per D-03: `--source-type success_pattern --evidence-type multi_phase`):
```bash
aether memory-capture --type "learning" --content "$claim" \
  --source-type "success_pattern" --evidence-type "multi_phase" 2>/dev/null || true
```

**Also update the resolution capture** -- line 591 (Step 2.1c):
```bash
aether memory-capture \
  --type "resolution" \
  --content "Recurring error pattern: $category ($count occurrences)" 2>/dev/null || true
```
This one uses `--type "resolution"` and should get appropriate source/evidence types. Since this captures error patterns detected from midden, appropriate flags would be `--source-type error_resolution --evidence-type multi_phase`.

---

### `.aether/docs/command-playbooks/continue-full.md` (playbook, request-response) -- MODIFIED

**Analog:** `.aether/docs/command-playbooks/continue-full.md` lines 1056-1073 (memory-capture in learning extraction)

**Current call** -- line 1059:
```bash
aether memory-capture --type "learning" --content "$claim"
```

**Updated call** -- same pattern as continue-advance.md:
```bash
aether memory-capture --type "learning" --content "$claim" \
  --source-type "success_pattern" --evidence-type "multi_phase"
```

**Also update the resolution capture** -- line 1254:
```bash
aether memory-capture \
  --type "resolution" \
  --content "Recurring error pattern: $category ($count occurrences)" 2>/dev/null || true
```
Same update as continue-advance.md.

## Shared Patterns

### Go Command Structure
**Source:** `cmd/learning.go` lines 12-66 and 220-237
**Apply to:** `cmd/learning.go` modification
```go
// All commands follow this pattern:
var cmdName = &cobra.Command{
    Use:   "command-name [args]",
    Short: "Short description",
    Args:  cobra.NoArgs, // or cobra.MaximumNArgs(1)
    RunE: func(cmd *cobra.Command, args []string) error {
        if store == nil { outputErrorMessage("no store initialized"); return nil }
        // ... flags, service init, business logic ...
        outputOK(map[string]interface{}{...})
        return nil
    },
}
// Flag registration in init():
func init() {
    cmdName.Flags().String("flag-name", "", "Description")
    rootCmd.AddCommand(cmdName)
}
```

### Output Envelope Pattern
**Source:** `cmd/helpers_test.go` lines 1-142; `cmd/learning.go` lines 55-63
**Apply to:** `cmd/learning.go` modification
```go
// Success:
outputOK(map[string]interface{}{"key": value})

// Error:
outputError(code, "message", nil)            // code 1-5
outputErrorMessage("message")                // code 1, no details

// Both produce JSON: {"ok":true/false,"result":...,"error":...,"code":...}
```

### Test Helpers
**Source:** `cmd/testing_main_test.go` (saveGlobals, resetRootCmd); `cmd/build_flow_cmds_test.go` (setupBuildFlowTest)
**Apply to:** `cmd/circuit_breaker_test.go`
```go
// All cmd tests start with:
saveGlobals(t)
resetRootCmd(t)

// For tests needing store:
dataDir := setupBuildFlowTest(t)
defer cleanup  // handled by t.TempDir()

// For stdout/stderr capture:
stdout = &bytes.Buffer{}
stderr = &bytes.Buffer{}
```

### Mutex-Protected Concurrent State
**Source:** `cmd/codex_build_worktree.go` lines 166 (`rootOpsMu sync.Mutex`)
**Apply to:** `cmd/circuit_breaker.go`
```go
// All CircuitBreaker methods use sync.Mutex:
func (cb *CircuitBreaker) Allow(workerName string) bool {
    cb.mu.Lock()
    defer cb.mu.Unlock()
    return !cb.tripped[workerName]
}
```

### Ceremony Event Emission
**Source:** `cmd/ceremony_emitter.go` lines 379-456
**Apply to:** New `emitBuildCeremonyWorkerSkipped` function in `cmd/ceremony_emitter.go`
```go
func emitBuildCeremonyWorkerSkipped(dispatch codex.WorkerDispatch, wave int, reason string) {
    payload := ceremonyPayloadForDispatch(dispatch, wave, "skipped", reason)
    emitBuildCeremony(events.CeremonyTopicBuildSpawn, payload)
}
```

### WorkerDispatch Caste Access
**Source:** `pkg/codex/dispatch.go` lines 15-29
**Apply to:** `cmd/circuit_breaker.go` (findSameCastePeer helper)
```go
type WorkerDispatch struct {
    WorkerName string  // "Hammer-23"
    Caste      string  // "builder", "watcher", etc.
    // ... other fields ...
}
// Access: dispatch.Caste for same-caste peer lookup
// Access: dispatch.WorkerName for per-instance breaker tracking
```

## No Analog Found

All files have strong analogs in the codebase. No gaps.

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| (none) | | | All 7 files have clear analogs |

## Metadata

**Analog search scope:** `cmd/`, `pkg/memory/`, `pkg/codex/`, `.aether/docs/command-playbooks/`
**Files scanned:** 12
**Pattern extraction date:** 2026-04-29
