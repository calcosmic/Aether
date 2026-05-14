# Phase 116: Queen Orchestration - Research

**Researched:** 2026-05-13
**Domain:** Queen intelligence, workflow pattern selection, Builder-Probe Lock, tiered escalation, midden integration
**Confidence:** HIGH (verified against Go source, playbooks, and TS host prototypes)

## Summary

The Go runtime has mature Queen orchestration logic that the TS host must replicate or delegate to. The Queen's responsibilities span three layers: (1) **Pre-dispatch planning** — selecting workflow patterns, caste relevance scoring, verification depth, and execution policy; (2) **In-flight orchestration** — wave lifecycle management, recovery decisions, and tiered escalation; and (3) **Post-dispatch governance** — gate evaluation, audit consolidation, and midden threshold checks.

Phase 116 must port or wrap this intelligence into the TS host so that `/ant-build` and `/ant-continue` wrappers can make Queen-level decisions without duplicating Go logic. The recommended approach is a **hybrid delegation model**: the TS host calls Go `--plan-only` commands to obtain Queen recommendations (manifests already contain `queen_recommendation` and `queen_execution_policy`), then the TS host enforces Builder-Probe Lock and tiered escalation at dispatch time, consulting Go CLI subcommands for midden checks and recovery logging.

**Primary recommendation:** Use Go's existing `queenOrchestrate`, `recommendQueenExecutionPolicy`, and `orchestrateRecovery` as the authority. Build TS host modules that call these via `--plan-only` manifests and dedicated CLI commands (`failure-classify`, `recovery-log-read`, `midden-review`), rather than reimplementing the logic in TypeScript.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Workflow pattern selection | Go runtime (`--plan-only` manifest) | TS host (reads manifest, presents to user) | Go already computes `queen_recommendation`; TS host should not duplicate scoring |
| Builder-Probe Lock enforcement | TS host (dispatch layer) | Go runtime (status normalization in finalizer) | TS host sees worker results first; must prevent premature `completed` upgrades |
| Tiered escalation | Go runtime (`orchestrateRecovery`) | TS host (triggers retry/reassign via wave-orchestrator) | Recovery logic is complex and already tested in Go; TS host delegates decisions |
| Midden threshold checks | Go runtime (`midden-review`, `midden-write`) | TS host (reads results, emits REDIRECT pheromones) | Midden is Go-owned data; TS host reads via CLI, acts on findings |
| Verification depth mapping | Go runtime (`resolveSmartVerificationDepth`) | TS host (passes through to manifest) | Depth resolution uses phase position + risk keywords; already in Go |
| Ambassador conditional spawn | Go runtime (`phaseNeedsAmbassador`) | TS host (includes in dispatch plan) | Keyword detection lives in Go; TS host reads from manifest `dispatches` |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go `aether` CLI | v1.0.34 (current) | Authority for Queen decisions, recovery, midden | Source of truth for all state mutations and governance logic |
| TypeScript orchestration host | Node >=20 | Control plane that calls Go, enforces locks, manages dispatch | Already built in Phase 109-114 |
| `wave-orchestrator.ts` | Prototype | Wave grouping, parallel dispatch, retry loops | Already handles TS-02 and TS-03 |
| `worker-dispatch.ts` | Prototype | Single worker dispatch with spawn-log/complete | Already handles TS-01 |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `event-bridge.ts` | Prototype | Reads Go ceremony events from JSONL | For real-time midden and recovery event consumption |
| `go-bridge.ts` | Prototype | `callGoJSON` wrapper for all Go CLI calls | All Go delegation in TS host |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Go delegation for recovery | Reimplement `orchestrateRecovery` in TS | High risk — 474 lines of Go with classification registry, budget logic, circuit breaker integration; duplication invites drift |
| TS-native midden storage | Read Go `midden.json` via CLI | Midden is Go-owned; direct TS writes violate boundary contract (TS-06) |
| Manual workflow pattern selection | Go `queenOrchestrate` + playbook lookup | Playbooks already define patterns; Go's `casteRelevanceRegistry` has 27 castes with keyword scoring |

## Architecture Patterns

### System Architecture Diagram

```
User -> /ant-build wrapper
  |
  v
TS Host (control plane)
  |
  +---> Go CLI: aether build <phase> --plan-only
  |       |
  |       +---> queenOrchestrate(phase, "build", state)
  |       +---> recommendQueenExecutionPolicy(state, phase, totalPhases, input)
  |       +---> plannedBuildDispatchesForSelectionWithState(...)
  |       |
  |       +---> JSON manifest with:
  |             - queen_recommendation (review_depth, reason)
  |             - queen_execution_policy (verification_depth, review_depth)
  |             - dispatches[] (caste, wave, execution_wave, task)
  |             - wave_execution[] (strategy, worker_count, reason)
  |             - execution_plan[] (stage, strategy, castes)
  |
  +---> TS Host: QueenOrchestrator module
  |       |
  |       +---> Reads manifest
  |       +---> Enforces Builder-Probe Lock
  |       +---> Calls dispatchWaves() from wave-orchestrator.ts
  |       +---> On failure: calls Go failure-classify + recovery-log-read
  |       +---> On midden threshold: calls Go midden-review, emits REDIRECT
  |
  +---> Go CLI: aether build-finalize <phase> --completion-file <path>
          |
          +---> validateBuildProvenance()
          +---> mergeExternalBuildResults()
          +---> Atomic state commit via store.UpdateJSONAtomically()
```

### Recommended Project Structure

```
.aether/ts-host/src/
├── queen/
│   ├── orchestrator.ts       # Main Queen orchestrator (workflow selection, lock enforcement)
│   ├── workflow-patterns.ts  # Pattern definitions (SPBV, Investigate-Fix, etc.)
│   ├── builder-probe-lock.ts # Builder-Probe Lock enforcement
│   ├── escalation.ts         # Tiered escalation wrapper (calls Go recovery CLI)
│   ├── midden-check.ts       # Midden threshold checks + REDIRECT emission
│   └── types.ts              # Queen-specific TS interfaces
├── lifecycle.ts              # Updated to use QueenOrchestrator
├── wave-orchestrator.ts      # Existing (retry, parallel dispatch)
├── worker-dispatch.ts        # Existing (single worker dispatch)
├── go-bridge.ts              # Existing (callGoJSON)
└── types.ts                  # Existing (BuildManifest, WorkerResult, etc.)
```

### Pattern 1: Workflow Pattern Selection from Manifest
**What:** The TS host reads the Go manifest's `queen_recommendation` and `queen_execution_policy` to determine review depth and worker castes, then maps phase keywords to a named workflow pattern for display.
**When to use:** Every build dispatch. The pattern name is shown to the user in ceremony output but does not change the manifest dispatches (those are already computed by Go).
**Example:**
```typescript
// Source: .aether/docs/command-playbooks/build-full.md lines 555-566
function selectWorkflowPattern(phaseName: string): WorkflowPattern {
  const lower = phaseName.toLowerCase();
  if (lower.match(/bug|fix|error|broken|failing/)) return "Investigate-Fix";
  if (lower.match(/research|oracle|explore|investigate/)) return "Deep Research";
  if (lower.match(/refactor|restructure|clean|reorganize/)) return "Refactor";
  if (lower.match(/security|audit|compliance|accessibility|license/)) return "Compliance";
  if (lower.match(/docs|documentation|readme|guide/)) return "Documentation Sprint";
  return "SPBV"; // Standard Plan-Build-Verify (default)
}
```

### Pattern 2: Builder-Probe Lock
**What:** Builders may return `code_written` (indicating they produced code but did not self-verify). Only the Probe caste may upgrade a worker's status to `completed`. The TS host intercepts builder results and normalizes `code_written` -> `completed` only when a probe has verified.
**When to use:** In the dispatch result processing loop, before writing the completion file for `build-finalize`.
**Example:**
```typescript
// Source: cmd/codex_build_finalize.go lines 568-582 (normalizeExternalBuildStatus)
// Go already normalizes "code_written" -> "completed" in the finalizer.
// TS host should preserve "code_written" in the completion file so Go can apply the lock.
function applyBuilderProbeLock(
  results: WorkerResult[],
  dispatches: BuildDispatch[]
): WorkerResult[] {
  const probeNames = new Set(
    dispatches.filter(d => d.caste === "probe").map(d => d.name)
  );
  return results.map(r => {
    if (r.caste === "builder" && r.status === "completed") {
      // If no probe verified this builder, downgrade to code_written
      const hasProbeVerification = results.some(
        pr => pr.caste === "probe" && pr.status === "completed"
      );
      if (!hasProbeVerification) {
        return { ...r, status: "code_written" as TerminalWorkerStatus };
      }
    }
    return r;
  });
}
```

### Pattern 3: Tiered Escalation via Go Delegation
**What:** When a worker fails, the TS host calls Go's recovery orchestrator (via a new CLI command or by reading recovery-log and applying logic) to decide: retry, peer reassignment, fixer dispatch, or escalate to user.
**When to use:** After each wave completes and failures are detected.
**Example:**
```typescript
// Source: cmd/recovery_orchestrator.go lines 131-201 (orchestrateRecovery)
// The TS host should NOT reimplement this. Instead:
async function handleWaveFailures(
  opts: GoBridgeOptions,
  phase: number,
  wave: number,
  failures: DispatchResult[]
): Promise<RecoveryAction[]> {
  // Option A: Call a new Go CLI command (if added)
  // Option B: Read failure-classify rules and apply locally
  const actions: RecoveryAction[] = [];
  for (const f of failures) {
    const classification = await classifyFailure(opts, f.status, f.summary);
    switch (classification) {
      case "recoverable":
        actions.push({ type: "retry", worker: f.name });
        break;
      case "requires-attempt":
        actions.push({ type: "retry", worker: f.name });
        break;
      case "blocking":
        actions.push({ type: "escalate", worker: f.name });
        break;
    }
  }
  return actions;
}
```

### Anti-Patterns to Avoid
- **Reimplementing `orchestrateRecovery` in TypeScript:** The Go implementation has 474 lines covering classification registry, budget tracking, circuit breaker integration, and peer reassignment. Duplicating this invites drift and bugs.
- **Writing to `midden.json` directly:** The boundary contract (TS-06) prohibits TS host writes to `.aether/data/`. Always use `aether midden-write` CLI.
- **Hard-coding caste lists in TS:** Go's `casteRelevanceRegistry` has 27 castes with keyword scoring. The TS host should read the manifest's `dispatches` array, not maintain its own caste logic.
- **Allowing builders to self-report `completed` without probe verification:** This breaks the Builder-Probe Lock contract (ORC-02).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Worker failure classification | Custom TS failure classifier | Go `failure-classify` CLI or `classifyWorkerFailure` registry | 19 deterministic patterns with rationale, already tested |
| Recovery budget tracking | TS budget counter | Go `RecoveryBudget` in `recovery-log-{N}.json` | Per-wave reset, 3-action budget, persisted atomically |
| Circuit breaker | Custom TS breaker | Go `CircuitBreaker` (threshold, per-worker, reset per wave) | Goroutine-safe, integrated with ceremony events |
| Midden storage | TS midden file | Go `midden-write`, `midden-review`, `midden-acknowledge` CLI | Boundary contract violation if TS writes directly |
| Verification depth | TS depth heuristics | Go `resolveSmartVerificationDepth` | Combines phase mode, position, risk keywords, and user flags |
| Caste relevance scoring | TS keyword matcher | Go `queenOrchestrate` + `casteRelevanceRegistry` | 27 castes with base scores, conditions, special rules |

## Common Pitfalls

### Pitfall 1: Builder-Probe Lock Bypass
**What goes wrong:** Builders return `completed` and the TS host passes this straight to `build-finalize`, so no probe ever runs. The finalizer normalizes `code_written` -> `completed`, but if the builder already said `completed`, the lock is bypassed.
**Why it happens:** The TS host does not intercept and downgrade builder statuses before writing the completion file.
**How to avoid:** In the TS host result processing, explicitly downgrade builder `completed` to `code_written` unless a probe worker has also completed. Or, ensure the manifest always includes a probe dispatch (Go does this when `queenCastes["probe"]` is true).
**Warning signs:** Build finalizes with no probe worker in the manifest dispatches; builder statuses are all `completed` with no verification stage.

### Pitfall 2: Recovery Budget Desync
**What goes wrong:** The TS host retries workers independently of Go's recovery budget, causing Go to see more retries than budget allows or missing recovery log entries.
**Why it happens:** `wave-orchestrator.ts` has its own `retryLimit` and `retryDelayMs`, but Go tracks retries in `recovery-log-{N}.json` with a `RecoveryBudget`.
**How to avoid:** Either disable TS-level retry and delegate all retry decisions to Go, or keep TS retry as a shallow wrapper (1 attempt) and let Go handle the full recovery sequence. The safest path: set `retryLimit: 1` in TS and let Go's `orchestrateRecovery` decide on deeper retry.

### Pitfall 3: Midden Threshold Missed During Build
**What goes wrong:** The build proceeds even though the midden has unacknowledged entries in the same category as the current phase, leading to repeated failures.
**Why it happens:** The TS host does not call `midden-review` before dispatching workers.
**How to avoid:** Add a pre-build midden check in the Queen orchestrator: call `aether midden-review`, count unacknowledged entries by category, and if the count exceeds a threshold (e.g., 3), emit a REDIRECT pheromone and optionally pause the build.

### Pitfall 4: Workflow Pattern Display Drift
**What goes wrong:** The TS host displays one workflow pattern (e.g., "Refactor") but the Go manifest was built with a different caste set, causing user confusion.
**Why it happens:** The TS host computes the pattern independently of Go's `queenOrchestrate`.
**How to avoid:** Derive the displayed pattern from the manifest's `dispatches` array (e.g., if no builder but oracle + scout, show "Deep Research"). Do not maintain a separate pattern selector.

## Code Examples

### Calling Go for Queen Recommendations
```typescript
// Source: cmd/codex_build.go lines 139-253 (runCodexBuildPlanOnlyWithOptions)
const result = callGoJSON<BuildManifestResult>(opts, [
  "build", String(phaseNum), "--plan-only"
]);
const manifest = result.dispatch_manifest;
const policy = manifest.queen_execution_policy; // { verification_depth, review_depth }
const recommendation = manifest.queen_recommendation; // { review_depth, reason }
```

### Reading Midden for Threshold Check
```typescript
// Source: cmd/midden_cmds.go lines 52-80 (middenReviewCmd)
const midden = callGoJSON<{ groups: Record<string, unknown[]>; total: number }>(opts, [
  "midden-review"
]);
if (midden.total > 3) {
  // Emit REDIRECT pheromone via Go CLI
  callGoJSON(opts, [
    "pheromone-write", "--type", "REDIRECT",
    "--content", `Midden threshold breached: ${midden.total} unacknowledged failures`
  ]);
}
```

### Normalizing Worker Status for Builder-Probe Lock
```typescript
// Source: cmd/codex_build_finalize.go lines 568-582
function normalizeWorkerStatus(status: string): TerminalWorkerStatus {
  const s = status.toLowerCase().trim();
  switch (s) {
    case "complete": case "done": case "success": case "succeeded": case "passed": case "code_written":
      return "completed";
    case "fail": case "error":
      return "failed";
    case "timed_out": case "cancelled": case "canceled":
      return "timeout";
    case "manual": case "manually_reconciled":
      return "manually-reconciled";
    default:
      return s as TerminalWorkerStatus;
  }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Wrapper markdown playbooks contain full Queen logic (build-full.md) | Go runtime owns Queen orchestration; wrappers consume manifests | v1.14 (Phases 93-99) | Playbooks still describe ceremony but Go computes dispatch plans |
| Manual `--light` / `--heavy` flags only | Smart depth resolution via `resolveSmartVerificationDepth` | v1.15 (Phase 85) | Phase position + risk keywords auto-select depth |
| Single retry in wrapper | Tiered escalation with budget, circuit breaker, peer reassignment | v1.14 (Phase 96-98) | `orchestrateRecovery` handles 4-tier escalation |
| Midden as passive log | Midden drives auto-REDIRECT and threshold checks | v1.13 (Phase 88-92) | Unacknowledged failures influence build decisions |

**Deprecated/outdated:**
- `build-full.md` playbook's manual workflow pattern table: still valid for display, but the authoritative caste selection is in Go's `queenOrchestrate`.
- Wrapper-level retry loops: should be replaced by Go's recovery orchestrator or kept as a thin 1-attempt wrapper.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Go `--plan-only` manifests already contain sufficient Queen recommendations (`queen_recommendation`, `queen_execution_policy`) for the TS host to make dispatch decisions without reimplementing scoring | Standard Stack | If manifests lack these fields, TS host must call additional Go commands or reimplement logic |
| A2 | The TS host can call a new Go CLI command for recovery decisions (or use existing `failure-classify` + `recovery-log-read` to reconstruct the same logic) | Tiered Escalation | If no such CLI exists, the TS host must either add one to Go or reimplement `orchestrateRecovery` |
| A3 | `code_written` status is preserved through the completion file and normalized to `completed` only by Go's finalizer | Builder-Probe Lock | If the TS host normalizes early, the lock is bypassed; if it never normalizes, builds fail |
| A4 | Midden threshold checks happen pre-build, not intra-wave | Midden Integration | If required intra-wave, the TS host must poll `midden-review` between waves |

## Open Questions

1. **Should the TS host add a new Go CLI subcommand for recovery decisions?**
   - What we know: `orchestrateRecovery` is a pure function in Go but not exposed as a CLI command. `failure-classify` and `recovery-log-read` exist.
   - What's unclear: Whether the TS host can reconstruct the full recovery decision from these commands, or if a new `aether recovery-decide` command is needed.
   - Recommendation: Start with `failure-classify` + `recovery-log-read` in the TS host. If gaps appear, add a thin Go CLI wrapper around `orchestrateRecovery`.

2. **How does the Builder-Probe Lock interact with simulated workers?**
   - What we know: Simulated workers always return `completed`. The finalizer normalizes statuses.
   - What's unclear: Whether simulated builds should skip the lock (no real code to verify) or enforce it for test fidelity.
   - Recommendation: Enforce the lock in all modes for consistency; simulated probe workers can be lightweight.

3. **Should workflow pattern selection affect the manifest dispatches, or only ceremony display?**
   - What we know: Go's `queenOrchestrate` selects castes based on keywords. The playbook's pattern table is for display.
   - What's unclear: Whether ORC-01 expects the TS host to modify dispatch plans based on pattern.
   - Recommendation: Pattern selection is display-only; the manifest dispatches are authoritative. If a pattern requires different castes, that should be reflected in Go's `queenOrchestrate` (future enhancement).

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go `aether` CLI | All Queen decisions | Yes | v1.0.34 | Build from source (`go build ./cmd/aether`) |
| Node.js | TS host runtime | Yes | >=20 | — |
| `aether build --plan-only` | Manifest generation | Yes | v1.0.34 | — |
| `aether build-finalize` | State commit | Yes | v1.0.34 | — |
| `aether failure-classify` | Failure classification | Yes | v1.0.34 | — |
| `aether recovery-log-read` | Recovery history | Yes | v1.0.34 | — |
| `aether midden-review` | Midden threshold checks | Yes | v1.0.34 | — |
| `aether pheromone-write` | REDIRECT emission | Yes | v1.0.34 | — |

**Missing dependencies with no fallback:** None identified.

**Missing dependencies with fallback:** None identified.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go test (for Go logic) + Node.js test runner (for TS host) |
| Config file | `package.json` scripts for TS; `go test ./...` for Go |
| Quick run command | `go test ./cmd/... -run "Queen\|Recovery\|Midden"` |
| Full suite command | `go test ./... -race` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| ORC-01 | Queen selects workflow pattern based on phase name | unit | `go test ./cmd -run TestCasteRelevance` | Yes |
| ORC-02 | Builders return `code_written`; only Probe upgrades to `completed` | integration | Manual verification via build-finalize | No — needs TS test |
| ORC-03 | Tiered escalation: retry -> peer -> fixer -> user | unit | `go test ./cmd -run TestOrchestrateRecovery` | Yes |
| ORC-04 | Midden threshold emits REDIRECT pheromone | integration | `go test ./cmd -run TestMidden` + TS integration | Partial |
| ORC-05 | Phase mode maps to verification depth | unit | `go test ./cmd -run TestReviewDepth` | Yes |
| ORC-06 | Ambassador spawned for integration tasks | unit | `go test ./cmd -run TestPhaseNeedsAmbassador` | Yes |

### Sampling Rate
- **Per task commit:** `go test ./cmd/... -run "Queen\|Recovery\|Midden"`
- **Per wave merge:** `go test ./... -race`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `cmd/queen_orchestration_regression_test.go` — covers ORC-01 through ORC-06 at Go level
- [ ] TS host integration tests for Builder-Probe Lock
- [ ] TS host integration tests for midden threshold -> REDIRECT flow
- [ ] New Go CLI command `recovery-decide` (optional, if needed)

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V5 Input Validation | Yes | Go CLI validates all inputs server-side; TS host passes through only |
| V6 Cryptography | No | Not in scope for this phase |
| V7 Error Handling | Yes | Recovery orchestrator classifies failures deterministically; no LLM inference for security-critical decisions |

### Known Threat Patterns

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Malicious worker claims | Tampering | Go finalizer validates all file claims against repo root; rejects `.aether/data/` writes |
| Boundary contract violation | Elevation of Privilege | `assertNoDirectDataWrites` in TS host; Go atomic state commits |
| Recovery budget exhaustion | Denial of Service | Per-wave budget reset (3 actions); circuit breaker prevents infinite retry |

## Sources

### Primary (HIGH confidence)
- `cmd/caste_relevance.go` — `queenOrchestrate`, `casteRelevanceRegistry`, `spawnThreshold`, `isAlwaysRequired`
- `cmd/codex_dispatch_contract.go` — `recommendQueenWorkflowProfile`, `recommendQueenExecutionPolicy`, `codexQueenExecutionPolicy`
- `cmd/recovery_orchestrator.go` — `orchestrateRecovery`, `RecoveryBudget`, `RecoveryAction`, `sequenceRecoverable`, `sequenceRequiresAttempt`
- `cmd/recovery_classify.go` — `classifyWorkerFailure`, `failureClassifications` registry (19 patterns)
- `cmd/circuit_breaker.go` — `CircuitBreaker`, `findSameCastePeer`, per-wave reset
- `cmd/queen_wave_lifecycle.go` — `queenWaveLifecycle`, wave grouping, recovery between waves
- `cmd/queen_decision.go` — `queenDecide`, gate classification, escalation logging
- `cmd/midden_cmds.go` — `midden-review`, `midden-write`, `midden-acknowledge`
- `cmd/codex_build.go` — `plannedBuildDispatchesForSelectionWithState`, `queenBuildPreWaveDispatches`, `phaseNeedsAmbassador`
- `cmd/review_depth.go` — `resolveSmartVerificationDepth`, `phaseRiskLevel`, `phasePositionLevel`
- `.aether/docs/command-playbooks/build-full.md` — Workflow pattern selection table (lines 555-566)
- `.aether/docs/command-playbooks/build-wave.md` — Spawn plan, caste emoji, depth checks

### Secondary (MEDIUM confidence)
- `.aether/ts-host/src/lifecycle.ts` — Current prototype lifecycle (plan -> build -> continue)
- `.aether/ts-host/src/worker-dispatch.ts` — Dispatch with spawn-log/complete
- `.aether/ts-host/src/wave-orchestrator.ts` — Wave grouping, retry, parallel dispatch
- `.aether/ts-host/src/types.ts` — TypeScript interfaces for manifest and worker results

### Tertiary (LOW confidence)
- `cmd/queen_orchestration_regression_test.go` — Regression tests for Queen behavior (not read in detail)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all components exist and are verified in Go source
- Architecture: HIGH — clear delegation pattern from Go to TS host
- Pitfalls: MEDIUM-HIGH — some edge cases (simulated workers, pattern vs. dispatch) need validation

**Research date:** 2026-05-13
**Valid until:** 2026-06-13 (stable Go API) / 2026-05-20 (if TS host interfaces change)
