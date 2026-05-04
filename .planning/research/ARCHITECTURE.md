# Architecture Research: v1.14 Queen Authority

**Domain:** Autonomous queen coordination, auto-recovery, smart gating, output filtering
**Researched:** 2026-05-03
**Overall confidence:** HIGH (based on direct source code analysis of `cmd/codex_build.go`, `cmd/codex_continue.go`, `cmd/gate.go`, `cmd/autopilot.go`, `cmd/codex_dispatch_contract.go`, `.claude/agents/ant/aether-queen.md`, and playbook files)

## Executive Summary

v1.14 wires the Queen into existing infrastructure so she can drive builds autonomously instead of just narrating them. The architecture has three distinct surfaces: (1) auto-recovery logic that intercepts worker failures during build waves and continues through the existing 4-tier escalation chain without pausing for human input, (2) smart gating that modifies the 11-gate continue pipeline to auto-resolve non-critical findings and only block on genuine problems, and (3) output filtering that collapses raw worker noise into structured summaries surfaced to the user.

The key insight is that the Go runtime already has most of the infrastructure -- `codexBuildDispatch` structs carry status/summary/blockers, `gateCheck` structs carry passed/detail/fix-hint, the circuit breaker pattern exists for loop prevention, and `runCodexContinueGates()` already has skip logic for previously-passed gates. Queen authority layers on top of these existing surfaces rather than replacing them.

**Critical design constraint:** The Go runtime owns state mutations (per CLAUDE.md architecture rules). The Queen is a wrapper-layer agent on Claude/OpenCode. This means auto-recovery decisions that mutate colony state must go through the Go runtime (`aether build-finalize`, `aether continue`), not through direct state file manipulation by the Queen agent.

## Current Architecture (What Exists)

### Build Flow

```
cmd/codex_build.go:
  runCodexBuildWithOptions()
    -> validateCodexBuildState()          (state machine check)
    -> beginRuntimeSpawnRun()             (trace)
    -> writeCodexBuildArtifacts()         (worker briefs, manifest)
    -> executeCodexBuildDispatches()      (dispatch workers via invoker)
    -> reconcileCompletedBuildTasks()     (update task statuses)
    -> atomic state commit                (StateBUILT)

  runCodexBuildQueenLed()                (plan-only variant with queen metadata)
    -> runCodexBuildPlanOnlyWithOptions() (produces dispatch_manifest)
    -> sets dispatch_mode="queen-led"     (tells wrapper to drive)
    -> wrapper spawns agents              (Claude/OpenCode Task tool)
    -> aether build-finalize              (wrapper sends results back to Go)
```

### Continue Flow

```
cmd/codex_continue.go:
  runCodexContinue()
    -> loadActiveColonyState()
    -> loadCodexContinueManifest()        (build manifest from build phase)
    -> detectAbandonedBuild()             (stale detection)
    -> runCodexContinueVerification()     (build/lint/test steps)
    -> assessCodexContinue()              (task evidence, partial success)
    -> runCodexContinueGates()           (11 gate checks)
    -> runCodexContinueReview()           (reviewer agents)
    -> atomic state commit                (PhaseCompleted, next phase)
    -> signal housekeeping                (pheromone decay, etc.)
```

### The 11 Gates (in `runCodexContinueGates()`)

| Gate | Name | Blocking? | Auto-Skippable? |
|------|------|-----------|-----------------|
| 1 | `manifest_present` | Yes | Yes (if previously passed) |
| 2 | `verification_steps_passed` | Yes | Yes (if previously passed) |
| 3 | `implementation_evidence` | Yes | Yes (if previously passed) |
| 4 | `operational_evidence` | No (informational) | Yes |
| 5 | `partial_success` | No (advisory) | No |
| 6 | `tests_pass` | Yes | No (always runs) |
| 7 | `flags` | Yes | No (always runs) |
| 8 | `watcher_veto` | Yes | No (always runs) |
| 9 | `no_critical_flags` | Yes | No (always runs) |
| 10 | `loop_detection` | Yes | No (always runs) |
| 11 | `parameter_loop` | Yes | No (always runs) |

Plus the playbook gates (Steps 1.6-1.14) that run in wrapper layer:
- spawn_gate (MANDATORY)
- anti_pattern (conditional)
- complexity (conditional, non-blocking)
- gatekeeper (conditional)
- auditor (MANDATORY)
- tdd_evidence (MANDATORY)
- runtime (MANDATORY)
- flags (MANDATORY)
- watcher_veto (MANDATORY)
- medic (conditional auto-spawn)

### Queen Agent (Current State)

The Queen agent (`.claude/agents/ant/aether-queen.md`) currently:
- Selects workflow patterns (SPBV, Investigate-Fix, etc.)
- Spawns workers via Task tool
- Processes results and synthesizes
- Escalates through 4 tiers (retry -> parent reassign -> queen reassigns -> user)
- Does NOT make state mutations directly
- Does NOT auto-recover from continue gate failures
- Does NOT filter output

### Workflow Profile Contract

`cmd/codex_dispatch_contract.go` already defines:
- `codexQueenWorkflowRecommendation` -- intent/profile/depth recommendation
- `codexWorkflowProfileContract` -- profiles with blocking/advisory check lists
- `codexDispatchContract` -- execution model, timeout, fallback behavior
- `recommendQueenWorkflowProfile()` -- intent-matching logic

This contract was designed for queen-led builds but is currently only used for metadata, not decision-making.

---

## Question 1: Where Should Auto-Recovery Logic Live?

### Answer: Two layers -- wrapper-layer recovery during build, runtime-layer recovery during continue.

**During build waves (wrapper layer):**

The Queen agent already has a 4-tier escalation chain defined in her agent markdown. The problem is that tiers 1-3 are described as instructions but not wired into the Go runtime. The solution:

1. **Go runtime provides recovery metadata** -- `codexBuildDispatch` already carries `Status`, `Summary`, `Blockers`, and `Duration`. Add a `RecoveryAttempts` counter and `LastFailureReason` field.

2. **Wrapper layer executes recovery** -- The Queen agent, during `build-wave.md` Step 5.2 (Process Wave Results), interprets the failure metadata and decides:
   - Tier 1: Re-spawn same worker (no state mutation needed -- just another Task call)
   - Tier 2: Re-spawn with modified brief (inject prior failure context into prompt)
   - Tier 3: Spawn different caste (builder fails -> tracker investigates -> builder retries)
   - Tier 4: Surface to user (create flag, pause)

3. **Go runtime records recovery state** -- Add `aether recovery-record` subcommand that writes recovery attempt metadata to the build manifest. This is read by subsequent continue runs to understand what happened.

**During continue gates (Go runtime layer):**

Continue gate failures currently return `blocked: true` and stop. Auto-recovery here means the Go runtime can attempt self-healing before blocking:

1. **New: `cmd/recovery.go`** -- Recovery orchestrator that runs between gate failure and blocking:
   ```
   runCodexContinue()
     -> runCodexContinueGates()
     -> if !gates.Passed:
       -> attemptAutoRecovery(gates, phase, manifest)  // NEW
       -> if recovery succeeded: re-run gates
       -> if recovery failed: block as before
   ```

2. **Recovery strategies per gate:**
   - `verification_steps_passed` failed: Re-run failed steps with increased timeout
   - `spawn_gate` failed: Skip if in queen-led mode (queen already spawned)
   - `implementation_evidence` failed: Attempt task reconciliation from build artifacts
   - `tests_pass` failed: Run tests once more with fresh state (race condition catch)
   - `watcher_veto` failed: Cannot auto-recover (requires judgment) -- escalate

3. **Circuit breaker integration** -- `cmd/circuit_breaker.go` already exists. Recovery attempts go through the circuit breaker to prevent infinite recovery loops.

### Component Map: Auto-Recovery

```
EXISTING                                           NEW (v1.14)
========                                           ============

cmd/codex_build.go (dispatch structs) ....... cmd/codex_build.go (RecoveryAttempts field)
.aether/docs/command-playbooks/build-wave.md  build-wave.md (recovery wiring in Step 5.2)
cmd/codex_continue.go (gate pipeline) ....... cmd/recovery.go (autoRecovery orchestrator)
cmd/circuit_breaker.go .................... cmd/recovery.go (uses existing breaker)
cmd/gate.go (gateCheck struct) ............. cmd/gate.go (AutoRecoverable bool field)
cmd/autopilot.go .......................... cmd/autopilot.go (recovery-aware pause)
```

### New vs Modified

| Component | Change Type | What Changes |
|-----------|-------------|-------------|
| `cmd/codex_build.go` | Modified | Add `RecoveryAttempts int` and `LastFailureReason string` to `codexBuildDispatch` |
| `cmd/gate.go` | Modified | Add `AutoRecoverable bool` to `gateCheck`; add `maxRecoveryAttempts int` |
| `cmd/recovery.go` | New | Recovery orchestrator: `attemptAutoRecovery()`, `recoverVerificationSteps()`, `recoverSpawnGate()`, `recoverTestsPass()` |
| `build-wave.md` | Modified | Step 5.2 reads `RecoveryAttempts` from dispatch metadata, executes tiers 1-3 automatically |
| `cmd/autopilot.go` | Modified | Autopilot respects recovery state -- does not advance past unrecovered phases |
| `colony.go` types | Modified | Add `RecoveryLog []RecoveryEntry` to `ColonyState` |

---

## Question 2: How Should Smart Gating Modify the Existing 11-Gate Pipeline?

### Answer: Add a "severity classification" layer that demotes non-critical failures to advisory, and an "auto-resolve" layer that fixes recoverable issues before blocking.

### Gate Severity Classification

Currently all gate failures are equal -- any failure blocks advancement. Smart gating introduces three tiers:

| Tier | Behavior | Example |
|------|----------|---------|
| **Hard block** | Always blocks, no auto-recovery | `watcher_veto` with critical issues, `loop_detection` tripped |
| **Soft block** | Blocks but auto-recoverable | `verification_steps_passed` (re-run), `tests_pass` (retry), `spawn_gate` (queen-led skip) |
| **Advisory** | Never blocks, logs to midden | `operational_evidence`, `partial_success`, non-critical `auditor` findings |

### Implementation: Gate Classification Table

Add to `cmd/gate.go`:

```go
// gateClassification determines how a gate failure should be handled.
type gateClassification string

const (
    gateHardBlock   gateClassification = "hard_block"   // always blocks
    gateSoftBlock   gateClassification = "soft_block"   // blocks, but auto-recoverable
    gateAdvisory    gateClassification = "advisory"      // never blocks
)

// gateClassifications maps gate names to their severity tier.
// Gates not in this map default to "hard_block".
var gateClassifications = map[string]gateClassification{
    "manifest_present":          "hard_block",
    "verification_steps_passed": "soft_block",  // can re-run steps
    "implementation_evidence":   "soft_block",  // can reconcile
    "operational_evidence":      "advisory",    // informational only
    "partial_success":           "advisory",    // advisory only
    "tests_pass":                "soft_block",  // can retry
    "flags":                     "hard_block",  // requires user judgment
    "watcher_veto":              "hard_block",  // requires judgment
    "no_critical_flags":         "hard_block",
    "loop_detection":            "hard_block",  // safety mechanism
    "parameter_loop":            "hard_block",  // safety mechanism
}
```

### Smart Gate Pipeline

Modify `runCodexContinueGates()` to add classification-aware processing:

```
runCodexContinueGates():
  1. Run all gates (existing logic)
  2. NEW: Classify each failure by severity
  3. NEW: For soft_block failures, attempt auto-recovery
  4. NEW: For advisory failures, log to midden but don't block
  5. Return gate report with:
     - hard_blocks: gates that genuinely block
     - soft_blocks_resolved: gates that failed but were auto-recovered
     - advisory findings: gates that produced findings but don't block
```

### Playbook Gate Smartening

The playbook gates (Steps 1.6-1.14) run in the wrapper layer and already have conditional skip logic. Smart gating for these means:

| Playbook Gate | Current | Smart Behavior |
|---------------|---------|----------------|
| spawn_gate | HARD REJECT if no spawns | Skip in queen-led mode (queen already spawned workers) |
| anti_pattern | HARD REJECT if critical | Demote "warnings" to advisory; keep "critical" as hard block |
| complexity | Non-blocking | Keep as-is (already non-blocking) |
| gatekeeper | Conditional on package.json | Demote "high" from soft-block to advisory in queen-led mode |
| auditor | Hard block if critical or score < 60 | In queen-led mode: auto-accept score >= 50 (lower threshold), log findings to midden |
| tdd_evidence | Hard reject if fabricated | Keep as-is (fabrication is a trust violation) |
| runtime | Hard block if user says no | In queen-led/autopilot mode: auto-skip (no human to ask) |
| flags | Hard block if blockers | Keep as-is (blockers require resolution) |
| watcher_veto | Hard block + user choice | In queen-led mode: auto-accept if score >= 5 (lower threshold from 7) |
| medic | Conditional auto-spawn | Keep as-is (already auto-spawns) |

### Component Map: Smart Gating

```
EXISTING                                           NEW (v1.14)
========                                           ============

cmd/gate.go (gateCheck) ................... cmd/gate.go (Classification, AutoRecoverable)
cmd/gate.go (runCodexContinueGates) ........ cmd/gate.go (classification-aware processing)
cmd/codex_continue.go ..................... cmd/codex_continue.go (reads smart gate report)
cmd/codex_dispatch_contract.go (profiles) .. cmd/codex_dispatch_contract.go (queen-led thresholds)
build-wave.md (Step 5.2) ................. build-wave.md (auto-recovery wiring)
continue-gates.md (Steps 1.6-1.14) ........ continue-gates.md (queen-led conditionals)
```

### New vs Modified

| Component | Change Type | What Changes |
|-----------|-------------|-------------|
| `cmd/gate.go` | Modified | Add `gateClassification`, `gateClassifications` map, classification-aware `runCodexContinueGates()` |
| `cmd/codex_continue.go` | Modified | Read smart gate report; advisory findings go to midden instead of blockers |
| `cmd/codex_dispatch_contract.go` | Modified | Queen-led workflow profiles get relaxed thresholds |
| `continue-gates.md` | Modified | Each gate step adds queen-led conditional branch |
| `cmd/codex_build.go` | Modified | `codexBuildManifest` includes `QueenLed bool` for gate threshold selection |

---

## Question 3: Where Should Output Filtering Happen?

### Answer: Three-layer filtering -- Go runtime filters raw dispatch data, Queen agent synthesizes summaries, wrapper presents filtered output.

### Layer 1: Go Runtime (Structural Filtering)

The Go runtime already produces structured JSON output with `map[string]interface{}` envelopes. Output filtering here means:

1. **Gate report simplification** -- Instead of returning all 11 gate check details, return a summary:
   ```go
   type codexSmartGateSummary struct {
       HardBlocks       int      `json:"hard_blocks"`
       SoftBlocksResolved int    `json:"soft_blocks_resolved"`
       AdvisoryFindings int      `json:"advisory_findings"`
       BlockingGates    []string `json:"blocking_gates,omitempty"`
       ResolvedGates    []string `json:"resolved_gates,omitempty"`
   }
   ```

2. **Dispatch result compression** -- Instead of returning full dispatch details (skill sections, pheromone sections), return only:
   - Worker name, caste, status, summary, blockers
   - Skip: full skill assignment, pheromone injection, worker brief content

3. **Continue report streamlining** -- `codexContinueReport` currently has 20+ fields. Add a `codexContinueSummary` that collapses to:
   - Phase, passed/failed, blocking issues (if any), next command
   - Skip: full worker flow, operational issues, task assessments (available on demand)

### Layer 2: Queen Agent (Synthesis)

The Queen agent, as the coordinator, produces a single synthesis per phase:

```
Phase N: {name}
  Workers: {completed}/{total} succeeded
  Gates: {passed}/{total} passed ({N} auto-resolved)
  Duration: {elapsed}

  Key findings:
  - {finding 1} (advisory)
  - {finding 2} (advisory)

  Blockers:
  - {blocker 1} (hard -- needs attention)

  Next: {command}
```

This replaces the current firehose of per-worker completion lines, per-gate pass/fail, and raw verification output.

### Layer 3: Wrapper Presentation (Visual Filtering)

The wrapper markdown (`.claude/commands/ant/build.md`, `.claude/commands/ant/continue.md`) controls what the user sees. Changes:

1. **Build wrapper** -- Instead of showing every worker spawn/result:
   - Show wave header: "Wave 1: 3 builders dispatched"
   - Show summary: "Wave 1: 2/3 succeeded, 1 auto-recovered"
   - Show only failures (not successes) in detail

2. **Continue wrapper** -- Instead of showing every gate:
   - Show summary: "11 gates: 9 passed, 1 auto-resolved, 1 advisory"
   - Show only hard blocks in detail
   - Skip advisory findings entirely (available via `/ant-status`)

### Component Map: Output Filtering

```
EXISTING                                           NEW (v1.14)
========                                           ============

cmd/codex_build.go (dispatch maps) ......... cmd/codex_build.go (filtered dispatch maps)
cmd/codex_continue.go (report) ............ cmd/codex_continue.go (summary struct)
cmd/codex_visuals.go ...................... cmd/codex_visuals.go (summary rendering)
cmd/gate.go (gate report) ................. cmd/gate.go (smart summary struct)
.claude/commands/ant/build.md ............. build.md (filtered presentation)
.claude/commands/ant/continue.md ........... continue.md (filtered presentation)
```

### New vs Modified

| Component | Change Type | What Changes |
|-----------|-------------|-------------|
| `cmd/codex_visuals.go` | Modified | Add `renderSmartGateSummary()`, `renderPhaseSummary()` functions |
| `cmd/codex_continue.go` | Modified | Add `codexContinueSummary` struct alongside full report |
| `cmd/codex_build.go` | Modified | Add `codexBuildSummary` struct alongside full dispatch maps |
| `cmd/gate.go` | Modified | Add `codexSmartGateSummary` struct |
| `.claude/commands/ant/build.md` | Modified | Use summary output instead of full dispatch list |
| `.claude/commands/ant/continue.md` | Modified | Use summary output instead of full gate list |

---

## Question 4: How Should Queen Authority Integrate With Existing Go Runtime?

### Answer: Queen authority is a coordination layer, not a state mutation layer. The Queen makes decisions; the Go runtime executes them.

### The Authority Model

```
Queen (wrapper layer)                    Go Runtime (authoritative)
=====================                    =========================

Decides:                                 Executes:
- Which workers to spawn                 - aether build --plan-only (dispatch plan)
- When to retry/skip/reassign            - aether build-finalize (commit results)
- Which gates to auto-resolve            - aether continue (verification + gates)
- What to surface to user                - aether state-mutate (task status)
- How to summarize output                - aether pheromone-write (signals)
                                         - aether gate-results-write (gate state)
                                         - aether recovery-record (recovery log)

Cannot:
- Write COLONY_STATE.json directly       (violates wrapper-runtime contract)
- Skip hard-block gates                  (safety mechanism)
- Force phase advance without verification
```

### Queen-Led Build Path (Existing + Enhanced)

The existing `runCodexBuildQueenLed()` already produces a `dispatch_manifest` with `dispatch_mode: "queen-led"`. The wrapper then:
1. Reads the manifest
2. Spawns agents via Task tool
3. Calls `aether build-finalize` to commit

**Enhancement:** The manifest includes recovery metadata so the Queen can make informed decisions:

```go
// Add to codexBuildManifest
RecoveryPolicy  codexRecoveryPolicy `json:"recovery_policy,omitempty"`

type codexRecoveryPolicy struct {
    MaxWorkerRetries     int `json:"max_worker_retries"`      // default: 2
    MaxCasteReassignments int `json:"max_caste_reassignments"` // default: 1
    AutoSkipThreshold     int `json:"auto_skip_threshold"`     // default: 3 (failures before skip)
    QueenLedGates         bool `json:"queen_led_gates"`        // relaxed gate thresholds
}
```

### Queen-Led Continue Path (New)

Currently continue is a single `aether continue` call. Queen-led continue adds a pre-check and post-check:

```
Queen-led continue flow:
  1. Queen calls: aether continue --plan-only    (NEW -- produces gate plan without executing)
  2. Queen evaluates: which gates can auto-resolve
  3. Queen executes: recovery actions for soft-block gates
  4. Queen calls: aether continue --finalize     (NEW -- commits with recovery context)
  5. Queen summarizes: filtered output to user
```

**New Go runtime entry points:**

| Command | Purpose | Returns |
|---------|---------|---------|
| `aether continue --plan-only` | Run verification and gates, return report without mutating state | Gate plan with classifications |
| `aether continue --finalize` | Commit phase advancement with recovery context | Same as current `aether continue` but accepts recovery metadata |

### Integration Points Summary

```
                    QUEEN AUTHORITY INTEGRATION MAP
                    ================================

BUILD PHASE:
  cmd/codex_build.go
    runCodexBuildQueenLed() -----> dispatch_manifest (already exists)
    NEW: recovery_policy field ----> wrapper reads and respects
    build-wave.md Step 5.2 --------> recovery logic (wrapper executes)
    aether build-finalize ---------> wrapper commits (already exists)
    NEW: aether recovery-record ----> wrapper logs recovery attempts

CONTINUE PHASE:
  cmd/codex_continue.go
    NEW: runCodexContinuePlanOnly() -> gate plan without state mutation
    cmd/recovery.go
      attemptAutoRecovery() ---------> auto-fixes soft-block gates
    cmd/gate.go
      gateClassification ------------> determines block vs advisory
    NEW: runCodexContinueFinalize() -> commits with recovery context
    continue-gates.md --------------> queen-led conditionals per gate

OUTPUT:
  cmd/codex_visuals.go
    NEW: renderPhaseSummary() ------> collapsed phase view
    NEW: renderSmartGateSummary() --> collapsed gate view
  wrapper markdown
    build.md / continue.md ---------> filtered presentation

STATE:
  pkg/colony/colony.go
    NEW: RecoveryLog field ---------> tracks recovery attempts
    NEW: QueenLedMode field ---------> enables relaxed thresholds
```

---

## Recommended Build Order

The phases should build in this order based on dependency analysis:

### Phase 1: Gate Classification Infrastructure
**What:** Add `gateClassification` to `gate.go`, update `gateCheck` struct, add classification map.
**Why first:** Everything else depends on knowing which gates are hard-block vs soft-block vs advisory.
**Modifies:** `cmd/gate.go`
**Risk:** Low -- additive changes, no existing behavior changes until consumers are added.

### Phase 2: Recovery Data Model
**What:** Add `RecoveryAttempts`, `LastFailureReason` to `codexBuildDispatch`; add `RecoveryLog` to `ColonyState`; add `codexRecoveryPolicy` to manifest; add `aether recovery-record` subcommand.
**Why second:** Auto-recovery logic needs somewhere to store its state.
**Modifies:** `cmd/codex_build.go`, `pkg/colony/colony.go`, new `cmd/recovery.go`
**Risk:** Low -- `omitempty` fields, backward compatible.

### Phase 3: Smart Gate Pipeline
**What:** Modify `runCodexContinueGates()` to use classification; add `attemptAutoRecovery()` in `cmd/recovery.go`; add `codexSmartGateSummary` struct.
**Why third:** Needs Phase 1 (classifications) and Phase 2 (recovery data model).
**Modifies:** `cmd/codex_continue.go`, `cmd/recovery.go`, `cmd/gate.go`
**Risk:** Medium -- changes gate evaluation logic, must preserve existing hard-block behavior.

### Phase 4: Continue Plan-Only and Finalize
**What:** Add `aether continue --plan-only` (verification + gates without state mutation) and `aether continue --finalize` (commit with recovery context).
**Why fourth:** Enables queen-led continue flow; depends on Phase 3 (smart gates).
**Modifies:** `cmd/codex_continue.go`, new cobra flags
**Risk:** Medium -- new code paths but isolated behind flags.

### Phase 5: Queen-Led Build Recovery
**What:** Modify `build-wave.md` Step 5.2 to read recovery metadata and execute tiers 1-3 automatically; update Queen agent to use recovery-aware spawning.
**Why fifth:** Wrapper-layer changes that consume the infrastructure from Phases 1-4.
**Modifies:** `build-wave.md`, `.claude/agents/ant/aether-queen.md`, `.opencode/agents/aether-queen.md`
**Risk:** Medium -- changes wrapper behavior, but Go runtime is the safety net.

### Phase 6: Queen-Led Continue Integration
**What:** Update `continue-gates.md` to add queen-led conditional branches (relaxed thresholds, auto-skip runtime gate, lower watcher veto threshold).
**Why sixth:** Builds on all prior phases.
**Modifies:** `continue-gates.md`, `.claude/commands/ant/continue.md`, `.opencode/commands/ant/continue.md`
**Risk:** Medium -- must ensure queen-led mode is opt-in and doesn't affect normal flow.

### Phase 7: Output Filtering
**What:** Add summary rendering functions to `cmd/codex_visuals.go`; add summary structs to build and continue outputs; update wrapper markdown to use summaries.
**Why last:** Pure presentation layer, no dependencies on recovery/gating logic.
**Modifies:** `cmd/codex_visuals.go`, `cmd/codex_build.go`, `cmd/codex_continue.go`, wrapper markdown files
**Risk:** Low -- additive, existing detailed output remains available.

---

## Anti-Patterns

### Anti-Pattern 1: Queen Mutates State Directly
**What:** Queen agent writes to COLONY_STATE.json via file manipulation.
**Why bad:** Violates wrapper-runtime contract; can corrupt state; bypasses atomic write and file locking.
**Instead:** All state mutations go through `aether` CLI subcommands.

### Anti-Pattern 2: Auto-Recovery Without Circuit Breaker
**What:** Recovery logic retries indefinitely.
**Why bad:** Creates infinite loops, wastes tokens, can corrupt state.
**Instead:** Every recovery attempt goes through the existing `CircuitBreaker` in `cmd/circuit_breaker.go`.

### Anti-Pattern 3: Relaxed Gates in Normal Mode
**What:** Queen-led gate thresholds (lower watcher veto, auto-skip runtime) apply to non-queen-led builds.
**Why bad:** Removes safety net from manual builds.
**Instead:** Relaxed thresholds only activate when `QueenLedMode` is true in colony state.

### Anti-Pattern 4: Output Filtering Hides Failures
**What:** Summary view omits hard-block information.
**Why bad:** User cannot diagnose why a phase is stuck.
**Instead:** Summary always shows hard blocks; details available via `/ant-status` or `--verbose` flag.

### Anti-Pattern 5: Recovery Changes Build Intent
**What:** Auto-recovery modifies the task goal or phase description.
**Why bad:** Changes what was planned without user consent.
**Instead:** Recovery only changes execution approach (different caste, retry with context), never changes the task itself.

---

## Scalability Considerations

| Concern | At Phase 1 (1 phase) | At Phase 5 (5 phases) | At Phase 10+ |
|---------|---------------------|----------------------|--------------|
| Recovery log size | Negligible (1-2 entries) | Small (5-10 entries) | Cap at 50 entries per phase, LRU eviction |
| Gate classification overhead | None (static map lookup) | None | None |
| Summary rendering | Fast (struct formatting) | Fast | Fast |
| Queen-led continue plan-only | 1 extra CLI call | 1 extra CLI call | 1 extra CLI call |
| Circuit breaker state | 1-2 keys | 5-10 keys | Existing cap handles this |

---

## Sources

- `cmd/codex_build.go` (direct source analysis, 1943 lines) -- HIGH confidence
- `cmd/codex_continue.go` (direct source analysis, 2800+ lines) -- HIGH confidence
- `cmd/gate.go` (direct source analysis, 769 lines) -- HIGH confidence
- `cmd/autopilot.go` (direct source analysis, 150+ lines) -- HIGH confidence
- `cmd/codex_dispatch_contract.go` (direct source analysis, 360+ lines) -- HIGH confidence
- `cmd/codex_build_finalize.go` (direct source analysis) -- HIGH confidence
- `.claude/agents/ant/aether-queen.md` (direct source analysis, 332 lines) -- HIGH confidence
- `.aether/docs/command-playbooks/build-wave.md` (direct source analysis, 1050 lines) -- HIGH confidence
- `.aether/docs/command-playbooks/continue-gates.md` (direct source analysis, 1099 lines) -- HIGH confidence
- `.aether/docs/wrapper-runtime-ux-contract.md` (direct source analysis) -- HIGH confidence
- `.planning/PROJECT.md` (project context) -- HIGH confidence
