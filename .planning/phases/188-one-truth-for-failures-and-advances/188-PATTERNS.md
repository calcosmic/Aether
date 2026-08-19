# Phase 188: One Truth for Failures and Advances - Pattern Map

**Mapped:** 2026-08-19
**Files analyzed:** 6 criteria, ~18 primary files touched or read, 5 net-new files
**Analogs found:** 6 / 6 (every new artifact has a same-repo precedent to copy; none is invented
from nothing)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `cmd/midden_shared.go` (new) | utility (canonical load/append) | request-response (file I/O) | `cmd/midden_cmds.go:415-467` `midden-write`'s load-then-save shape, hardened to `UpdateJSONAtomically` | exact (read shape) / stricter (write shape) |
| `cmd/autopilot.go`, `cmd/context.go` (x2), `cmd/memory_health.go`, `cmd/immune.go` (modified) | consumers (read midden) | request-response | each other — all four already have the identical (broken) shape | exact |
| `cmd/medic_scanner.go` (modified, bonus) | consumer (health scan) | batch (file walk) | same broken shape as above | exact |
| `cmd/advance_phase.go` (new) | service (atomic state-advance core) | CRUD (atomic read-modify-write) | `runCodexContinue`'s own inline "ATOMIC STATE COMMIT" block (`cmd/codex_continue.go`, ~line 908) — extracted, not invented | exact (this IS the existing correct code, moved) |
| `cmd/codex_continue.go`, `cmd/codex_continue_finalize.go` (modified) | callers (wire to shared core) | request-response | each other, post-extraction | exact |
| `cmd/state_cmds.go` (modified) | controller (CLI guard) | request-response | its own existing `enforceGuard`/`runGateCheck` machinery, made mandatory for one field | exact |
| `cmd/codex_build_finalize.go` (modified) | controller (loud warning insertion) | request-response | `cmd/init_cmd.go:200-205`'s existing stderr-warning convention | exact |
| `cmd/colony_state_atomicity_ratchet_test.go` (new) | test (static source ratchet) | batch (AST walk + call-graph BFS) | `cmd/subcommand_reachability_ratchet_test.go` (allowlist idiom) + `cmd/worktree_destruction_reachability_test.go` (AST/call-graph idiom) | partial (compose two patterns, no single exact analog — same situation Phase 187's own PATTERNS.md hit for its ratchet) |
| `cmd/testdata/colony_state_write_allowlist.json` (new) | data (checked-in baseline) | — | `cmd/testdata/orphan_allowlist.json` | exact |
| `cmd/compatibility_cmds.go` (modified) + new small helper function | service (retry-exhaustion record) | request-response | `cmd/recovery_orchestrator.go`'s `escalateOutcome` → `RecoveryLogEntry` shape (structure to imitate); `cmd/midden_shared.go`'s `appendMiddenEntry` (storage mechanism to call) | role-match (compose two existing patterns) |

## Pattern Assignments

### The midden read shape (all four broken consumers share it verbatim)

**Source of the bug, one representative instance** (`cmd/memory_health.go:76-82`, the shortest):
```go
var midden colony.MiddenFile
if err := s.LoadJSON("midden/midden.json", &midden); err == nil {
    summary.RecentFailures = len(midden.Entries)
    for _, entry := range midden.Entries {
        summary.LastFailure = latestMemoryHealthTimestamp(summary.LastFailure, entry.Timestamp)
    }
}
```
All four consumers (`autopilot.go:86-99`, `context.go:234-238`, `context.go:856-881`,
`memory_health.go:76-82`) follow this exact shape: `var midden colony.MiddenFile` (or, in
`autopilot.go`'s case, an inline anonymous struct with just `Entries`), then
`store.LoadJSON("midden/midden.json", &midden)`, `err == nil` guarding a graceful-degrade-to-empty
default. **Every one of the four already tolerates a missing/unreadable file correctly** — the
`loadMiddenFile` replacement only needs to change the *path string*, not the error-handling
contract each caller already relies on.

**The real production writer** (`cmd/midden_cmds.go:435-465`, the shape `loadMiddenFile`'s sibling
`appendMiddenEntry` should mirror, hardened to atomic):
```go
var mf colony.MiddenFile
if err := store.LoadJSON("midden.json", &mf); err != nil {
    mf = colony.MiddenFile{Version: "1.0.0", Entries: []colony.MiddenEntry{}}
}
if mf.Entries == nil {
    mf.Entries = []colony.MiddenEntry{}
}
entry := colony.MiddenEntry{
    ID:        entryID,
    Timestamp: ts.Format(time.RFC3339),
    Category:  category,
    Source:    source,
    Message:   message,
    Reviewed:  false,
    Tags:      []string{},
}
mf.Entries = append(mf.Entries, entry)
if err := store.SaveJSON("midden.json", mf); err != nil { ... }
```
`appendMiddenEntry` should follow this field-population shape exactly (same `colony.MiddenEntry`
fields, same ID convention `midden_{unix}_{pid}` or equivalent), but wrap the load-mutate-save in
`store.UpdateJSONAtomically("midden.json", &mf, func() error { ...append...; return nil })` instead
of the separate `LoadJSON`/`SaveJSON` pair shown above — see "The atomic-write primitive" under
Shared Patterns below.

**The independent second bug in `immune.go`** (`cmd/immune.go:252-278`) — do not fix only the path:
```go
var midden struct {
    Entries []map[string]interface{} `json:"entries"`
}
if err := store.LoadJSON("midden/midden.json", &midden); err != nil {
    outputOK(map[string]interface{}{"detected": 0, "reason": "no midden data"})
    return nil
}
...
for _, entry := range midden.Entries {
    category, _ := entry["category"].(string)
    description, _ := entry["description"].(string)
    if category == "" || description == "" {
        continue
    }
    ...
```
`colony.MiddenEntry`'s JSON tag for its text field is `"message"` (`pkg/colony/midden.go:11`),
never `"description"`. Even after the path fix, every entry's `description` lookup returns empty,
`description == ""` is always true, and `newScars` stays empty forever. Change `entry["description"]`
to `entry["message"]` in the same pass — or, better, since `loadMiddenFile` returns a typed
`colony.MiddenFile`, switch this function off its own hand-rolled `map[string]interface{}` decode
entirely and iterate `colony.MiddenEntry` structs directly, reading `.Category` and `.Message`.

---

### The atomic phase-advance core (extracted, not invented)

**Source — the block to extract, in full** (`cmd/codex_continue.go`, inside `runCodexContinue`,
starting around line 908):
```go
if err := store.UpdateJSONAtomically("COLONY_STATE.json", &updated, func() error {
    if err := validateRuntimeStateStillCurrent(updated, phase.ID, state.BuildStartedAt, colony.StateEXECUTING, colony.StateBUILT); err != nil {
        return err
    }
    updated.Events = append(trimmedEvents(updated.Events),
        fmt.Sprintf("%s|verification_passed|continue|Build verification passed for phase %d", now.Format(time.RFC3339), phase.ID),
        fmt.Sprintf("%s|gate_passed|continue|Continue gates passed for phase %d", now.Format(time.RFC3339), phase.ID),
    )
    updated.Plan.Phases[currentIdx].Status = colony.PhaseCompleted
    for i := range updated.Plan.Phases[currentIdx].Tasks {
        updated.Plan.Phases[currentIdx].Tasks[i].Status = colony.TaskCompleted
    }
    updated.BuildStartedAt = nil
    updated.GateResults = nil

    final = currentIdx == len(updated.Plan.Phases)-1
    nextCommand = "aether seal"
    if final {
        updated.State = colony.StateCOMPLETED
        updated.CurrentPhase = phase.ID
        updated.Events = append(updated.Events, fmt.Sprintf("%s|phase_completed|continue|Completed final phase %d", now.Format(time.RFC3339), updated.CurrentPhase))
    } else {
        nextIdx := currentIdx + 1
        if updated.Plan.Phases[nextIdx].Status == colony.PhasePending || updated.Plan.Phases[nextIdx].Status == "" {
            updated.Plan.Phases[nextIdx].Status = colony.PhaseReady
        }
        updated.CurrentPhase = nextIdx + 1
        nextPhase = &updated.Plan.Phases[nextIdx]
        updated.State = colony.StateREADY
        nextCommand = fmt.Sprintf("aether build %d", nextIdx+1)
        updated.Events = append(updated.Events, fmt.Sprintf("%s|phase_advanced|continue|Completed phase %d, ready for phase %d", now.Format(time.RFC3339), phase.ID, nextIdx+1))
    }
    return nil
}); err != nil {
    if errors.Is(err, errRuntimeStateSuperseded) {
        runStatus = "superseded"
        return continueSupersededResult(state, phase, err), state, phase, nil, nil, false, nil
    }
    return nil, state, phase, nil, nil, false, fmt.Errorf("failed to atomically advance phase: %w", err)
}
```
Note the log-message literal `|continue|` — the extracted function needs a `source string`
parameter (values `"continue"` / `"continue-finalize"`) so both call sites keep their existing,
distinct event-log wording (the finalize path's copy uses `|continue-finalize|` throughout).

**The bug to eliminate, side by side** (`cmd/codex_continue_finalize.go:1088-1091`,
`advanceExternalContinue`):
```go
if err := store.UpdateJSONAtomically("COLONY_STATE.json", &updated, func() error {
    updated = state
    updated.Events = append(trimmedEvents(updated.Events),
```
There is no `validateRuntimeStateStillCurrent` call here at all, and the very next line after the
atomic-block entry overwrites the freshly-unmarshaled `updated` with the long-stale `state`
parameter. `UpdateJSONAtomically`'s contract (`pkg/storage/storage.go:121-137`) is: unmarshal
current disk contents into `ptr` (here, `updated`), *then* call `mutate`. `updated = state` throws
that fresh read away on the very next line. The rewritten function must delete this line entirely
and use the closure's freshly-loaded `updated` value throughout, exactly as the source block does.

**The trailing non-atomic write to eliminate** (`cmd/codex_continue_finalize.go:1147`):
```go
_ = store.SaveJSON("COLONY_STATE.json", updated)
```
Compare the source path's equivalent (`cmd/codex_continue.go`, after the atomic block, same
function `runCodexContinue`):
```go
_ = appendRuntimeStateEventsIfCurrent(updated, flowEvents)
```
`appendRuntimeStateEventsIfCurrent` already exists and already re-validates currency before
appending, rather than blindly overwriting. The rewritten finalize path should call this same
helper for its own post-advance events instead of the bare `SaveJSON`.

---

### The state-mutate guard (made mandatory, not invented)

**Source — already-working, currently-optional machinery** (`cmd/state_cmds.go:26-40`):
```go
guard, _ := cmd.Flags().GetString("guard")
if guard != "" {
    if err := enforceGuard(guard); err != nil {
        return nil
    }
}
```
```go
func enforceGuard(guard string) error {
    parts := strings.SplitN(guard, ":", 2)
    ...
    switch guardType {
    case "task-complete":
        result = runGateCheck("task-complete", guardTarget, 0)
    case "phase-advance":
        phaseNum, err := strconv.Atoi(guardTarget)
        ...
        result = runGateCheck("phase-advance", "", phaseNum)
    ...
    }
    if !result.Allowed {
        outputError(1, fmt.Sprintf("guard %q blocked: %s", guard, result.Reason), result.Checks)
        return fmt.Errorf("guard blocked")
    }
    return nil
}
```
**The unguarded field-mutation to close** (`cmd/state_cmds.go:272-281`):
```go
case "current_phase":
    phaseNum := 0
    if _, err := fmt.Sscanf(value, "%d", &phaseNum); err != nil {
        outputError(1, fmt.Sprintf("invalid phase number %q", value), nil)
        return nil
    }
    if phaseNum > 0 && phaseNum <= len(state.Plan.Phases) {
        state.Plan.Phases[phaseNum-1].Status = colony.PhaseInProgress
    }
    state.CurrentPhase = phaseNum
```
This branch runs regardless of whether `--guard` was ever passed to the outer command — the
`RunE`'s `if guard != ""` check above is the only gate, and it is skippable simply by omitting the
flag. The fix belongs at the `executeFieldMode` level (or immediately before it is called for this
one field): require `cmd.Flags().Changed("guard")` to be true, the guard to parse as
`phase-advance:<N>`, and `<N>` to equal the `phaseNum` about to be written — refuse otherwise. This
mirrors the fail-closed-on-uncertainty rule already established by `validateRuntimeStateStillCurrent`
and by Phase 187's `worktreeDestructionSafety`.

---

### The plain-language warning (copied verbatim in spirit)

**Source** (`cmd/init_cmd.go:200-205`):
```go
// Clean up any leftover worktrees from previous colony
if cleaned, orphaned, err := gcOrphanedWorktrees(); err == nil && (cleaned > 0 || orphaned > 0) {
    fmt.Fprintf(os.Stderr, "warning: cleaned %d stale worktree(s), %d orphaned\n", cleaned, orphaned)
}
```
**The insertion point** (`cmd/codex_build_finalize.go:418-421`, right after the binding is
validated and before any branch acts on it):
```go
binding, err := validateBuildAttemptManifestBinding(*manifest, state)
if err != nil {
    return nil, colony.ColonyState{}, colony.Phase{}, nil, err
}
```
Add the `binding.Legacy` check here, unconditional `fmt.Fprintf(os.Stderr, "warning: ...")` in the
same style, plus (per D-11) an appended `colony.Event` string on whichever `colony.ColonyState`
this function goes on to persist — follow the existing event-string convention used throughout
this same file, e.g. the literal shape `fmt.Sprintf("%s|verification_passed|continue|...", now.Format(time.RFC3339), ...)`
seen in the criterion-2 block above; use an event type like `manifest_legacy_accepted` in the same
`timestamp|type|source|message` pipe-delimited shape.

---

### The retry-exhaustion record (composes two existing patterns)

**Structure to imitate** (`cmd/recovery_orchestrator.go:407-432`, `escalateOutcome` — not called
directly, but its *shape* — "an exhausted attempt produces a structured record naming what was
tried" — is exactly what the new autopilot-level function should produce, at a smaller scale since
there is no classification/budget machinery to replicate here):
```go
func escalateOutcome(ctx RecoveryContext, classification FailureClassification, failType FailureType, rationale string, failureRecord FailureRecord, now string, reason string) RecoveryOutcome {
    return RecoveryOutcome{
        ...
        Exhausted: true,
        LogEntries: []RecoveryLogEntry{
            {
                ID:            fmt.Sprintf("ro-%d-%d", ctx.Phase, time.Now().UnixNano()),
                Failure:       failureRecord,
                ActionTaken:   "escalate",
                Outcome:       reason,
                AttemptNumber: countRetries(ctx.RecoveryHistory),
                Timestamp:     now,
                Detail:        fmt.Sprintf("escalated: %s (%s)", reason, rationale),
            },
        },
    }
}
```
**The gap to close** (`cmd/compatibility_cmds.go:445-463`, inside `runCompatibilityAutopilot`):
```go
buildResult, err := runCodexBuildWithOptions(root, phase.ID, nil, false, codexBuildOptions{...})
if err != nil {
    emitVisualLine(fmt.Sprintf("⚠ Build failed for phase %d, attempting single retry...", phase.ID))
    select {
    case <-ctx.Done():
        ...
    case <-time.After(2 * time.Second):
    }
    buildResult, err = runCodexBuildWithOptions(root, phase.ID, nil, false, codexBuildOptions{...})
    if err != nil {
        _ = syncRunAutopilotState(state, opts, "paused", "")
        return nil, err
    }
}
```
The second `err != nil` branch is where the new record must be written, using the storage
mechanism from criterion 1 (`appendMiddenEntry`), before the `return nil, err`. Extract this branch's
body into a small named function so it is independently unit-testable (see 188-05-PLAN.md).

## Shared Patterns

### Load-then-fallback-to-empty on read
**Source:** all four midden readers, independently converged on the same shape.
**Apply to:** `loadMiddenFile` must preserve this exact contract — return an error the caller can
inspect, but every existing call site's `err == nil` / `err != nil` fallback-to-empty behavior
must be unchanged after the path swap.

### The atomic-write primitive
**Source:** `pkg/storage/storage.go:121-137`, `store.UpdateJSONAtomically`.
**Apply to:** `advancePhase()` (already uses it, being extracted from code that does), the new
`appendMiddenEntry` (D-02 upgrades it from the existing writers' plain load-then-save).

### Fail-closed on uncertainty
**Source:** `validateRuntimeStateStillCurrent` (this phase), `worktreeDestructionSafety`
(Phase 187, `cmd/worktree_safety.go`).
**Apply to:** the state-mutate guard (refuse, don't default-allow, when `--guard` is absent for
`current_phase`); the D-07 ratchet (fail loudly on zero findings, never pass vacuously).

### Grep-ratchet with self-regenerating, shrink-only allowlist
**Source:** `cmd/subcommand_reachability_ratchet_test.go` (`-update-orphan-allowlist`,
`TestOrphanAllowlistOnlyShrinks`, `testdata/orphan_allowlist.json`).
**Apply to:** `cmd/colony_state_atomicity_ratchet_test.go` /
`cmd/testdata/colony_state_write_allowlist.json`.

### Plain-language, every-occurrence stderr reporting
**Source:** `cmd/init_cmd.go:202`, Phase 187's D-02 (`reportWorktreePreservation`).
**Apply to:** the criterion-6 legacy-manifest warning; the criterion-3 refusal message.

## No Analog Found

| File/Test | Role | Data Flow | Reason |
|---|---|---|---|
| `cmd/colony_state_atomicity_ratchet_test.go`'s zero-tolerance reachability-from-`advancePhase` check | test (call-graph BFS rooted at one named function) | batch | `worktree_destruction_reachability_test.go`'s `reachableFrom` (line 1349) walks from a *set* of registered-command entry points across the whole `cmd` surface, built via cobra-registration indexing this phase does not need (there is exactly one root: the function name `advancePhase`, known at write time, not discovered from cobra registration). A smaller, purpose-built BFS over same-package `*ast.CallExpr` targets is the right scope — reusing the full cobra-indexing machinery from the model file would import far more complexity than this phase's one-root check needs. |
| The two-tier test for criterion 5 (unit-test the record function directly + structurally confirm its call site) | test | mixed (unit + structural read) | No existing test in this codebase forces `runCodexBuildWithOptions` to fail twice deterministically (confirmed by grep across `cmd/*_test.go`); the closest analog, `recovery_orchestrator_test.go`-style tests (not read in full this pass), test `RecoveryOutcome` construction directly without going through a live build — the same "test the decision function directly" shape this phase's D-14 adopts for the same reason. |

## Metadata

**Analog search scope:** `cmd/*.go`, `cmd/*_test.go`, `pkg/storage/storage.go`, `pkg/colony/midden.go`
**Files scanned directly (Read or targeted grep):** `cmd/midden_cmds.go`, `cmd/spawn_budget.go`,
`cmd/autopilot.go`, `cmd/context.go`, `cmd/memory_health.go`, `cmd/immune.go`,
`cmd/medic_scanner.go`, `cmd/medic_scanner_test.go`, `cmd/run_autopilot_test.go`,
`cmd/codex_continue.go`, `cmd/codex_continue_finalize.go`, `cmd/codex_build.go`,
`cmd/codex_workflow_cmds.go`, `cmd/state_cmds.go`, `cmd/build_attempt.go`,
`cmd/codex_build_finalize.go`, `cmd/init_cmd.go`, `cmd/compatibility_cmds.go`,
`cmd/recovery_orchestrator.go`, `cmd/queen_wave_lifecycle.go`, `cmd/circuit_breaker.go`,
`cmd/worktree_destruction_reachability_test.go` (structure only, not fully read),
`cmd/subcommand_reachability_ratchet_test.go` (targeted), `cmd/testdata/orphan_allowlist.json`,
`pkg/storage/storage.go`, `pkg/colony/midden.go`
**Pattern extraction date:** 2026-08-19
