# Phase 109: TypeScript Orchestration Host Prototype - Research

**Researched:** 2026-05-12
**Domain:** Hybrid runtime boundary, TypeScript orchestration, Go subprocess integration
**Confidence:** HIGH

## Summary

This phase builds a minimal TypeScript host that drives one complete `plan -> build 1 -> continue` lifecycle by calling Go CLI commands for manifests and finalizers, dispatching platform workers between them, and recording spawn lifecycle events. The host never writes `.aether/data/` directly -- all state mutation goes through Go finalizer commands.

The Go runtime already exposes the complete integration surface: `aether plan --plan-only` produces JSON manifests, `aether build-finalize`, `aether plan-finalize`, and `aether continue-finalize` accept completion files and commit state atomically, and `aether spawn-log` / `aether spawn-complete` record spawn lifecycle. The TypeScript host is an orchestration layer that consumes these surfaces in the correct sequence.

The `.aether/ts-host/` directory already exists with package.json, tsconfig.json, and a boundary-reference.ts file. The ceremony narrator in `.aether/ts/` provides reusable TypeScript patterns and can be imported for rendering.

**Primary recommendation:** Build a thin orchestration CLI (`host.ts`) that calls Go subprocess commands in sequence: `plan --plan-only` to get manifest, `spawn-log` before each worker, platform dispatch, `spawn-complete` after each worker, `build-finalize` with completion file, `continue --plan-only`, and `continue-finalize`. Test against Go golden workflow baselines.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** TS host runs as a Node script invoked by Go subprocess (`node .aether/ts-host/dist/host.js`). Go passes the manifest file path as a CLI argument.
- **D-02:** TS host lives in `.aether/ts-host/` as a separate package from the ceremony narrator.
- **D-03:** TS host imports ceremony rendering functions from `@aether/ceremony-narrator` when needed.
- **D-04:** This is an internal prototype only -- not shipped, not installed via `aether update`.
- **D-05:** TS host spawns platform workers as subprocess exec calls per the Go manifest.
- **D-06:** TS host records real spawn-log/spawn-complete events via Go CLI subcommands (`aether spawn-log`, `aether spawn-complete`).
- **D-07:** File-based exchange between Go and TS host. Go writes manifest JSON to file, passes path to TS host.
- **D-08:** TS host consumes Go output exclusively in `AETHER_OUTPUT_MODE=json`.
- **D-09:** Success threshold: prove the full `plan -> build 1 -> continue` lifecycle works end-to-end.
- **D-10:** Verification approach is Claude's discretion.

### Claude's Discretion
- Worker dispatch implementation details (exact spawn mechanism, error handling, timeout behavior)
- Test strategy for verifying prototype (golden test reuse vs new integration tests)
- How the TS host discovers the Go binary path
- Error handling patterns for Go subprocess failures
- Whether the TS host needs its own test framework or reuses Go test infrastructure

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| HOST-01 | Minimal TypeScript host prototype exists that can be invoked | `.aether/ts-host/` scaffold exists with package.json, tsconfig.json; add `host.ts` entry point |
| HOST-02 | Host calls Go `--plan-only` commands to obtain JSON manifests | Go exposes `aether plan --plan-only` (codex_plan.go), `aether build N --plan-only` (codex_build.go), `aether continue --plan-only` (codex_continue_plan.go) -- all produce JSON with `AETHER_OUTPUT_MODE=json` |
| HOST-03 | Host dispatches visible platform workers from manifest fields | `codexBuildManifest.dispatches` array provides `name`, `caste`, `task`, `stage`, `wave` per worker; `aether spawn-log --parent --caste --name --task --depth` records before dispatch, `aether spawn-complete --name --status --summary` records after |
| HOST-04 | Host calls Go finalizers to commit state changes | Go exposes `aether plan-finalize --completion-file`, `aether build-finalize <phase> --completion-file`, `aether continue-finalize --completion-file` |
| HOST-05 | Host never writes `.aether/data/` directly | boundary-reference.ts already defines `GO_OWNED_PATHS`; all mutations go through Go finalizer CLI calls |
| HOST-06 | Host records spawn lifecycle events via Go CLI | `aether spawn-log` and `aether spawn-complete` in cmd/spawn.go accept required flags; TS host calls these before/after each worker |
| HOST-07 | Host runs workflow end-to-end or documents blocker | Full lifecycle validated through Go golden tests in cmd/golden_workflow_test.go; TS host must produce equivalent state transitions |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Manifest generation | API / Backend (Go) | -- | Go owns plan-only output, validation, and provenance |
| State mutation | API / Backend (Go) | -- | Go owns atomic writes, locking, provenance validation |
| Lifecycle orchestration | TS Host (Node) | -- | TS drives the sequence: manifest -> dispatch -> finalize |
| Worker dispatch | TS Host (Node) | Platform CLI (Claude/OpenCode) | TS spawns platform processes per manifest |
| Spawn lifecycle tracking | TS Host (Node, via Go CLI) | Go (execution) | TS calls `aether spawn-log`/`spawn-complete` before/after each worker |
| Ceremony rendering | TS Host (Node) | -- | Imports from `@aether/ceremony-narrator` for event rendering |
| Verification gates | API / Backend (Go) | -- | Go continue-finalizer runs all verification, gates, reviews |
| Visual output | API / Backend (Go) | -- | TS host does NOT parse ANSI/visual output |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| typescript | 5.9.3 (pinned in package.json) | Type-safe host code | Already pinned in `.aether/ts-host/package.json` |
| tsx | 4.21.0 (pinned) | Development runner and test runner | Already pinned; supports `tsx --test` for built-in test runner |
| @types/node | 18.19.130 (pinned) | Node.js type definitions | Already pinned; matches Node 18+ target |
| Node.js | v25.9.0 (installed) | Runtime | Available on the machine |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| @aether/ceremony-narrator | 1.0.0 (local) | Ceremony event parsing and rendering | Import for worker activity display during orchestration |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| tsx --test | vitest or jest | tsx --test is zero-config, built into existing dev stack, and sufficient for integration tests; adding vitest would be overkill for a prototype |
| Node child_process.execFileSync | execa or zx | stdlib is sufficient for a prototype; no need for a dependency |

**Installation:**
```bash
cd .aether/ts-host && npm install
```

**Version verification:** Package versions are pinned in the existing `package.json`. Node v25.9.0 and npm 11.12.1 are installed. Go 1.26.1 is available for building the binary.

## Architecture Patterns

### System Architecture Diagram

```
                  Go Binary (aether CLI)
                  ======================
                  |                    |
 plan --plan-only |                    | plan-finalize
  --completion    |                    | --completion-file
                  v                    |
           [JSON Manifest]            |
                  |                    ^
                  v                    |
          TS Host (Node.js)           |
          =================           |
          |                          |
          | 1. Read manifest file    |
          | 2. For each dispatch:    |
          |    a. spawn-log (Go)     |
          |    b. platform worker    |
          |    c. spawn-complete (Go)|
          | 3. Build completion file |
          | 4. Call finalizer (Go)   |
          |                          |
          v                          |
   [Completion JSON File] ---------->+
                                    commit state
                                       |
                                       v
                                  COLONY_STATE.json
```

### Recommended Project Structure
```
.aether/ts-host/
  package.json              # Already exists
  tsconfig.json             # Already exists
  tsconfig.build.json       # Build config (to create)
  src/
    boundary-reference.ts   # Already exists - Go-owned path constants
    host.ts                 # Main entry point - lifecycle orchestrator
    go-bridge.ts            # Go subprocess invocation helpers
    types.ts                # TypeScript interfaces for manifest/completion JSON
    lifecycle.ts            # plan -> build -> continue sequence driver
    worker-dispatch.ts      # Platform worker spawning + spawn-log/complete
  test/
    host.test.ts            # Integration tests against Go binary
    go-bridge.test.ts       # Unit tests for subprocess helpers
```

### Pattern 1: Go Bridge - Subprocess Invocation
**What:** A module that wraps `child_process.execFileSync` for calling the Go binary with `AETHER_OUTPUT_MODE=json`.
**When to use:** Every Go CLI call from the TS host.
**Example:**
```typescript
// src/go-bridge.ts
import { execFileSync } from "node:child_process";
import { resolve } from "node:path";

export interface GoBridgeOptions {
  goBinaryPath: string;
  cwd: string;
}

export function callGoJSON<T>(
  opts: GoBridgeOptions,
  args: string[]
): T {
  const result = execFileSync(opts.goBinaryPath, args, {
    cwd: opts.cwd,
    env: { ...process.env, AETHER_OUTPUT_MODE: "json" },
    encoding: "utf-8",
    maxBuffer: 10 * 1024 * 1024,
  });
  const parsed = JSON.parse(result);
  // Go outputs either { ok: result } or { error: message }
  if (parsed.error) {
    throw new Error(`Go command failed: ${args.join(" ")}: ${parsed.error}`);
  }
  return (parsed.ok ?? parsed) as T;
}
```

### Pattern 2: Completion File Assembly
**What:** Build the JSON completion file that Go finalizers expect.
**When to use:** Before calling `build-finalize`, `plan-finalize`, or `continue-finalize`.
**Example:**
```typescript
// The build-finalize completion file must contain:
interface BuildCompletion {
  dispatch_manifest: BuildManifest;  // from plan-only output
  dispatches: WorkerResult[];        // one per manifest dispatch
  claims?: BuildClaims;              // optional aggregate claims
}

// Each WorkerResult must include:
interface WorkerResult {
  name: string;          // must match manifest dispatch name
  status: string;        // "completed" | "failed" | "timeout"
  summary: string;       // worker output summary
  outputs?: string[];    // files created/modified
  files_created?: string[];
  files_modified?: string[];
  tests_written?: string[];
}
```

### Pattern 3: Spawn Lifecycle Recording
**What:** Call `aether spawn-log` before each worker and `aether spawn-complete` after.
**When to use:** Every worker dispatch.
**Example:**
```typescript
// Before dispatch:
callGoJSON(bridge, [
  "spawn-log",
  "--parent", "Queen",
  "--caste", dispatch.caste,
  "--name", dispatch.name,
  "--task", dispatch.task,
  "--depth", "1",
]);

// ... dispatch worker ...

// After dispatch:
callGoJSON(bridge, [
  "spawn-complete",
  "--name", dispatch.name,
  "--status", workerStatus,  // "completed" or "failed"
  "--summary", workerSummary,
]);
```

### Anti-Patterns to Avoid
- **Writing .aether/data/ directly:** All state files are Go-owned. Use finalizer commands exclusively.
- **Parsing ANSI/visual output:** Never parse Go's visual mode output. Always use `AETHER_OUTPUT_MODE=json`.
- **Inventing workers not in manifest:** Only dispatch workers from the manifest's `dispatches` array.
- **Skipping provenance validation:** The Go finalizer validates manifest provenance. Do not try to bypass it.
- **Hardcoding the Go binary path:** Use `which aether` or configurable path, with fallback to `$HOME/.local/bin/aether`.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| State mutation | Custom JSON writes to COLONY_STATE.json | `aether build-finalize`, `aether plan-finalize`, `aether continue-finalize` | Go validates provenance, locks files, writes atomically |
| Spawn tracking | Custom spawn-log file writes | `aether spawn-log` and `aether spawn-complete` CLI commands | Go manages the spawn-tree.txt file with correct format |
| Manifest generation | Custom manifest assembly in TS | `aether build N --plan-only` and `aether plan --plan-only` | Go assembles all dispatch metadata, caste assignments, queen policy |
| Ceremony rendering | Custom ANSI formatting | `@aether/ceremony-narrator` functions | Already handles event parsing, caste identity, color detection |
| Verification/gates | Custom test/gate logic in TS | `aether continue-finalize` | Go runs all verification commands, gates, reviews, and learning |

**Key insight:** The TS host is an orchestration layer only. It sequences Go calls and dispatches platform workers. Every state-affecting operation goes through a Go CLI command.

## Common Pitfalls

### Pitfall 1: Completion File Schema Mismatch
**What goes wrong:** The completion file passed to `build-finalize` has incorrect field names or missing required fields. The finalizer rejects it.
**Why it happens:** The Go finalizer expects specific field names (`dispatch_manifest` not `manifest`, worker `name` must exactly match manifest dispatch names).
**How to avoid:** Use the exact types from `codexExternalBuildCompletion` in `codex_build_finalize.go` as the TypeScript interface definition. The completion must include `dispatch_manifest` (from plan-only output) and `dispatches`/`results`/`workers` (worker outcomes).
**Warning signs:** `build-finalize` returns "completion file must include dispatch_manifest" or "missing external worker result for X".

### Pitfall 2: Non-Terminal Worker Status
**What goes wrong:** A worker result has status `"running"` or `"spawned"` instead of a terminal status. The finalizer rejects it.
**Why it happens:** The Go finalizer calls `isTerminalExternalBuildStatus()` which only accepts `"completed"`, `"failed"`, `"blocked"`, `"timeout"`, or `"manually-reconciled"`.
**How to avoid:** Normalize all worker statuses to terminal values before building the completion file. Map common variants: `"done"` -> `"completed"`, `"error"` -> `"failed"`, `"timed_out"` -> `"timeout"`.
**Warning signs:** `build-finalize` returns "external worker result for X has non-terminal status".

### Pitfall 3: Plan-Only Manifest Staleness
**What goes wrong:** The `continue --plan-only` or `plan --plan-only` manifest is older than 24 hours, causing the finalizer to reject it.
**Why it happens:** `validateCodexPlanManifestFreshness()` checks `generated_at` is within 24 hours and not in the future.
**How to avoid:** Generate plan-only manifests immediately before using them. Do not cache manifests between runs.
**Warning signs:** `plan-finalize` returns "stale plan_manifest generated_at".

### Pitfall 4: Worker Name Mismatch
**What goes wrong:** The worker name in the completion file does not match the name in the manifest dispatch. The finalizer cannot merge results.
**Why it happens:** Manifest uses deterministic names like `"Builder-XX-YY"`. If the TS host generates a different name, the merge fails.
**How to avoid:** Use the exact `name` field from each manifest dispatch entry. Do not generate or modify worker names.
**Warning signs:** `build-finalize` returns "missing external worker result for X".

### Pitfall 5: Colony State Not in Expected State
**What goes wrong:** `build-finalize` rejects the call because the colony state is not in the expected state (e.g., not `EXECUTING` or no active phase).
**Why it happens:** The Go runtime validates state transitions. `build-finalize` expects the state to be transitioned by the plan-only manifest generation step, but only the full build command (not plan-only) transitions to EXECUTING.
**How to avoid:** `build-finalize` itself handles the state transition. The colony must be in a state where `validateCodexBuildState` passes. For the first build, the state should be `READY` with `CurrentPhase` set to the target phase.
**Warning signs:** `build-finalize` returns state validation errors.

## Code Examples

### Complete Lifecycle Sequence
```typescript
// Source: Derived from Go source code analysis
// cmd/codex_build.go (plan-only), cmd/codex_build_finalize.go, cmd/spawn.go

// Step 1: Plan (generate phases)
const planManifest = callGoJSON(bridge, [
  "plan", "--plan-only",
]);

// Step 2: Plan finalize (commit phases to colony state)
const planCompletion = {
  plan_manifest: planManifest.result,  // from plan-only output
  dispatches: planManifest.result.dispatches.map(d => ({
    ...d,
    status: "completed",  // simulated worker completion
    summary: "Planning completed",
  })),
  phase_plan: planManifest.result.phase_plan,
};
writeFileSync(completionPath, JSON.stringify({ result: planCompletion }));
const planResult = callGoJSON(bridge, [
  "plan-finalize", "--completion-file", completionPath,
]);

// Step 3: Build plan-only (get worker manifest for phase 1)
const buildManifest = callGoJSON(bridge, [
  "build", "1", "--plan-only",
]);

// Step 4: Dispatch workers with spawn lifecycle
for (const dispatch of buildManifest.result.dispatch_manifest.dispatches) {
  callGoJSON(bridge, [
    "spawn-log",
    "--parent", "Queen",
    "--caste", dispatch.caste,
    "--name", dispatch.name,
    "--task", dispatch.task,
    "--depth", "1",
  ]);

  // Spawn platform worker (simulated for prototype)
  const workerResult = await dispatchWorker(dispatch);

  callGoJSON(bridge, [
    "spawn-complete",
    "--name", dispatch.name,
    "--status", workerResult.status,
    "--summary", workerResult.summary,
  ]);
}

// Step 5: Build finalize (commit worker results)
const buildCompletion = {
  dispatch_manifest: buildManifest.result.dispatch_manifest,
  dispatches: workerResults,
};
writeFileSync(buildCompletionPath, JSON.stringify({ result: buildCompletion }));
const buildResult = callGoJSON(bridge, [
  "build-finalize", "1", "--completion-file", buildCompletionPath,
]);

// Step 6: Continue plan-only
const continueManifest = callGoJSON(bridge, [
  "continue", "--plan-only",
]);

// Step 7: Continue finalize (verification, gates, advance)
const continueCompletion = {
  continue_manifest: continueManifest.result.continue_manifest,
  dispatches: continueReviewResults,
};
writeFileSync(continueCompletionPath, JSON.stringify({ result: continueCompletion }));
const continueResult = callGoJSON(bridge, [
  "continue-finalize", "--completion-file", continueCompletionPath,
]);
```

### Go Output Parsing
```typescript
// Source: Verified from Go cmd/output.go patterns
// Go CLI in JSON mode outputs either:
// { "ok": { ... result ... } }    for success
// { "error": "message" }          for failure
// The "ok" wrapper is used by outputOK(); "error" by outputError()

interface GoOutput<T> {
  ok?: T;
  error?: string;
}

function parseGoOutput<T>(raw: string): T {
  const parsed: GoOutput<T> = JSON.parse(raw);
  if (parsed.error) {
    throw new GoCommandError(parsed.error);
  }
  return parsed.ok as T;
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Bash/Node orchestration (Classic v5.4.0) | Go runtime (v1.0+) | Apr 2026 | Lost living orchestration behavior |
| Direct state writes from wrappers | Go finalizer commands | Apr 2026 | Safer but less flexible |
| No spawn lifecycle tracking | `aether spawn-log` / `spawn-complete` CLI | Available since Go migration | Restored via Go CLI calls |
| `--plan-only` for builds only | `--plan-only` for plan, build, and continue | v1.0.20+ | Full lifecycle manifest surface available |
| No completion file validation | Provenance validation in finalizers | v1.0.25+ | Prevents phantom builds |

**Deprecated/outdated:**
- `codex_spawn_log.go` / `codex_spawn_complete.go` references in CONTEXT.md: These files do not exist. The actual spawn commands are in `cmd/spawn.go` (`spawn-log` and `spawn-complete` subcommands). [VERIFIED: codebase grep]

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The Go binary is available as `aether` on PATH (or at `$HOME/.local/bin/aether`) | Architecture Patterns | Host cannot find Go binary; need discovery fallback |
| A2 | `aether build N --plan-only` output in JSON mode includes `result.dispatch_manifest` as a `codexBuildManifest` object | Code Examples | Completion file assembly would need different field path |
| A3 | The ceremony narrator can be imported as `@aether/ceremony-narrator` from the ts-host package without publishing to npm | Standard Stack | Need to use relative path import or workspace protocol |
| A4 | The prototype will use simulated (fake) worker results for the first iteration, since real platform worker dispatch requires live Claude/OpenCode CLI | Architecture | If real dispatch is required, additional complexity for platform CLI detection |

**If this table is empty:** All claims in this research were verified or cited -- no user confirmation needed.

## Open Questions

1. **Go binary discovery**
   - What we know: `aether` is installed at `$HOME/.local/bin/aether` based on the publish runbook. The TS host needs to find it.
   - What's unclear: Whether the Go binary is always on PATH, or if the TS host needs to search multiple locations.
   - Recommendation: Use `process.env.AETHER_BINARY_PATH ?? "aether"` with fallback to `$HOME/.local/bin/aether`. This is Claude's discretion per D-10.

2. **Simulated vs real worker dispatch**
   - What we know: The prototype needs to prove the lifecycle. Real dispatch requires Claude/OpenCode CLI.
   - What's unclear: Whether the prototype should use the Go `FakeInvoker` pattern (simulated dispatch) or attempt real platform dispatch.
   - Recommendation: Start with simulated dispatch (hardcode worker results as `"completed"`) to prove the lifecycle, then optionally add real dispatch. The Go golden tests use simulated dispatch.

3. **Test strategy**
   - What we know: Go golden tests exist in `cmd/golden_workflow_test.go` with baselines in `cmd/testdata/`. The TS host is a separate Node package.
   - What's unclear: Whether to write TS tests that call the Go binary, or Go tests that invoke the TS host.
   - Recommendation: Write TS-side integration tests using `tsx --test` that invoke the Go binary and verify state transitions. Also add a Go test that runs the TS host as a subprocess and compares output to golden baselines.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Node.js | TS host runtime | Available | v25.9.0 | -- |
| npm | Package installation | Available | 11.12.1 | -- |
| TypeScript | Compilation | Available | 5.9.3 (pinned) | -- |
| tsx | Dev/test runner | Available | 4.21.0 (pinned) | -- |
| Go (aether binary) | All manifest/finalizer calls | Available | go1.26.1 | -- |
| @aether/ceremony-narrator | Ceremony rendering | Available (local) | 1.0.0 | Relative import |

**Missing dependencies with no fallback:**
- None identified

**Missing dependencies with fallback:**
- None identified

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | tsx --test (Node built-in test runner via tsx) |
| Config file | None -- tsx --test is zero-config |
| Quick run command | `cd .aether/ts-host && npx tsx --test test/*.test.ts` |
| Full suite command | `cd .aether/ts-host && npx tsx --test test/*.test.ts && cd ../.. && go test ./cmd/... -run TestTsHost -count=1` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| HOST-01 | Host entry point runs as Node script | unit | `npx tsx --test test/host.test.ts` | Wave 0 |
| HOST-02 | Host calls plan-only and gets JSON manifest | integration | `npx tsx --test test/go-bridge.test.ts` | Wave 0 |
| HOST-03 | Host dispatches workers with spawn-log/complete | integration | `npx tsx --test test/worker-dispatch.test.ts` | Wave 0 |
| HOST-04 | Host calls finalizers and state changes | integration | `npx tsx --test test/lifecycle.test.ts` | Wave 0 |
| HOST-05 | Host never writes .aether/data/ directly | unit | `npx tsx --test test/boundary.test.ts` | Wave 0 |
| HOST-06 | Spawn lifecycle events recorded via Go CLI | integration | `npx tsx --test test/worker-dispatch.test.ts` | Wave 0 |
| HOST-07 | Full lifecycle completes or documents blocker | e2e | `npx tsx --test test/lifecycle.test.ts` | Wave 0 |

### Sampling Rate
- **Per task commit:** `cd .aether/ts-host && npx tsx --test test/*.test.ts`
- **Per wave merge:** `cd .aether/ts-host && npx tsx --test test/*.test.ts && cd ../.. && go test ./cmd/... -count=1`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `.aether/ts-host/test/host.test.ts` -- covers HOST-01 entry point
- [ ] `.aether/ts-host/test/go-bridge.test.ts` -- covers HOST-02 Go subprocess calls
- [ ] `.aether/ts-host/test/worker-dispatch.test.ts` -- covers HOST-03, HOST-06 spawn lifecycle
- [ ] `.aether/ts-host/test/lifecycle.test.ts` -- covers HOST-04, HOST-07 full lifecycle
- [ ] `.aether/ts-host/test/boundary.test.ts` -- covers HOST-05 no .aether/data writes
- [ ] `.aether/ts-host/tsconfig.build.json` -- build configuration
- [ ] Framework install: `cd .aether/ts-host && npm install` -- if node_modules missing

## Security Domain

> This phase involves subprocess execution and file-based data exchange between Go and TypeScript. Security considerations apply.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | No auth in prototype |
| V3 Session Management | no | No session management |
| V4 Access Control | no | No access control |
| V5 Input Validation | yes | JSON parsing with type checking (TypeScript strict mode) |
| V6 Cryptography | no | No crypto |

### Known Threat Patterns for TS Host + Go Subprocess

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Command injection via worker names | Tampering | Go finalizer validates all string fields; TS host uses execFileSync (no shell) |
| Path traversal in completion files | Tampering | Go finalizer validates paths against repo root; TS host writes only to temp paths |
| Malformed JSON crash | Denial of Service | TypeScript try/catch around all JSON.parse and Go subprocess calls |
| Stale manifest replay | Spoofing | Go finalizer validates generated_at freshness (24h max) |

## Sources

### Primary (HIGH confidence)
- `cmd/codex_build.go` -- plan-only manifest generation, dispatch structures
- `cmd/codex_build_finalize.go` -- build finalizer, completion file schema, provenance validation
- `cmd/codex_plan_finalize.go` -- plan finalizer, completion file schema
- `cmd/codex_continue_finalize.go` -- continue finalizer, verification gates, state advance
- `cmd/codex_dispatch_contract.go` -- dispatch contract structures
- `cmd/spawn.go` -- spawn-log and spawn-complete CLI subcommands (verified: no separate `codex_spawn_*.go` files exist)
- `.aether/references/contracts/runtime-boundary-contract.md` -- boundary ownership contract
- `.aether/ts-host/package.json` -- existing package scaffold
- `.aether/ts-host/src/boundary-reference.ts` -- existing boundary reference code
- `.aether/ts/narrator.ts` -- ceremony narrator entry point with full TypeScript API
- `.aether/ts/package.json` -- ceremony narrator package definition
- `cmd/golden_workflow_test.go` -- golden test behavioral contract

### Secondary (MEDIUM confidence)
- CONTEXT.md decisions -- verified against source code structures
- REQUIREMENTS.md HOST requirements -- verified against Go CLI surface

### Tertiary (LOW confidence)
- None -- all findings verified against codebase

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - packages pinned in existing package.json, versions verified
- Architecture: HIGH - Go integration surface fully mapped from source code
- Pitfalls: HIGH - derived from actual Go validation functions in finalizer code
- Code examples: HIGH - based on actual Go struct definitions and CLI flag analysis

**Research date:** 2026-05-12
**Valid until:** 2026-06-12 (stable Go CLI surface unlikely to change within milestone)
