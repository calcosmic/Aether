# Architecture Research

**Domain:** Aether Hybrid Runtime — TypeScript orchestration control plane over Go safety kernel
**Researched:** 2026-05-12
**Confidence:** HIGH

## Standard Architecture

### System Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    Platform Surfaces                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │ Claude Code  │  │  OpenCode    │  │  Codex CLI   │          │
│  │   wrappers   │  │   wrappers   │  │  (native)    │          │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘          │
├─────────┴──────────────────┴──────────────────┴─────────────────┤
│              TypeScript Orchestration Host                       │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │   Host      │  │   Worker    │  │  Ceremony   │             │
│  │  Adapter    │  │  Dispatcher │  │   Renderer  │             │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘             │
│         │                │                │                     │
│  ┌──────┴────────────────┴────────────────┴──────┐             │
│  │        Manifest Parser & Contract Validator     │             │
│  └──────────────────────┬─────────────────────────┘             │
├─────────────────────────┴───────────────────────────────────────┤
│                     Go Safety Kernel                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │  Manifest   │  │   State     │  │  Finalizer  │             │
│  │  Generator  │  │   Mutator   │  │   Engine    │             │
│  │  (plan-only)│  │  (atomic)   │  │  (validate) │             │
│  └─────────────┘  └─────────────┘  └─────────────┘             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │  Dispatch   │  │  Recovery   │  │  Publish/   │             │
│  │  Contract   │  │   Engine    │  │   Update    │             │
│  │  Builder    │  │             │  │             │             │
│  └─────────────┘  └─────────────┘  └─────────────┘             │
├─────────────────────────────────────────────────────────────────┤
│                  Editable Colony Brain                           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │   YAML      │  │  Markdown   │  │    TOML     │             │
│  │  Commands   │  │  Playbooks  │  │   Agents    │             │
│  └─────────────┘  └─────────────┘  └─────────────┘             │
└─────────────────────────────────────────────────────────────────┘
```

### Component Responsibilities

| Component | Responsibility | Typical Implementation |
|-----------|----------------|------------------------|
| Go Manifest Generator | Produce JSON manifests for plan/build/continue without mutating state | `aether plan --plan-only`, `aether build --plan-only` |
| Go State Mutator | Atomic JSON writes with file locking, rollback on validation failure | `pkg/storage/storage.go`, `UpdateJSONAtomically` |
| Go Finalizer Engine | Validate manifest provenance, merge worker results, commit state | `aether build-finalize`, `aether continue-finalize` |
| TS Host Adapter | Shell out to Go for manifests, parse JSON, orchestrate dispatch | New: `.aether/ts/host.ts` |
| TS Worker Dispatcher | Spawn platform workers from manifest dispatches, record lifecycle | New: `.aether/ts/dispatcher.ts` |
| TS Ceremony Renderer | Consume Go ceremony events, render live worker stacks and wave banners | Extend existing `.aether/ts/narrator.ts` |
| Editable Assets | Command playbooks, agent definitions, skills, ceremonies | `.aether/commands/*.yaml`, `.claude/`, `.opencode/`, `.codex/` |

## Recommended Project Structure

```
.aether/ts/
├── package.json              # Existing: @aether/ceremony-narrator
├── tsconfig.json             # Existing
├── src/
│   ├── index.ts              # Re-export public surface
│   ├── narrator.ts           # Existing: ceremony event parser/renderer
│   ├── host.ts               # NEW: orchestration host adapter
│   ├── manifest.ts           # NEW: manifest parser + contract validator
│   ├── dispatcher.ts         # NEW: platform worker dispatcher
│   ├── platform/
│   │   ├── claude.ts         # NEW: Claude Code subagent adapter
│   │   ├── opencode.ts       # NEW: OpenCode agent adapter
│   │   └── codex.ts          # NEW: Codex CLI process adapter
│   ├── contracts/
│   │   ├── build-manifest.ts # NEW: codexBuildManifest type mirror
│   │   ├── plan-manifest.ts  # NEW: codexPlanManifest type mirror
│   │   └── dispatch-contract.ts # NEW: codexDispatchContract type mirror
│   └── types/
│       └── events.ts         # NEW: ceremony event type definitions
├── test/
│   ├── narrator.test.ts      # Existing
│   ├── host.test.ts          # NEW: host adapter unit tests
│   ├── manifest.test.ts      # NEW: manifest parsing tests
│   └── dispatcher.test.ts    # NEW: dispatch logic tests
└── dist/                     # Build output (gitignored)
```

### Structure Rationale

- **`src/host.ts`:** Single entry point for the orchestration host. Owns the lifecycle loop: manifest -> ceremony -> dispatch -> finalize. Keeps platform specifics behind adapter interfaces.
- **`src/manifest.ts`:** Mirrors Go manifest structures in TypeScript for type-safe parsing. Validates JSON shape before dispatch to fail fast on contract drift.
- **`src/dispatcher.ts`:** Abstracts worker spawning across Claude Code (Task tool), OpenCode (agent spawn), and Codex (subprocess). Records `spawn-log` and `spawn-complete` via Go CLI calls.
- **`src/platform/*.ts`:** Platform adapters isolate platform-specific spawn mechanics. Claude uses the Task tool; OpenCode uses its agent protocol; Codex spawns child processes. Each adapter implements a common `PlatformAdapter` interface.
- **`src/contracts/*.ts`:** TypeScript interfaces derived from Go structs (`codexBuildManifest`, `codexPlanManifest`, `codexDispatchContract`). These are the integration contract. When Go structs change, these must update.
- **`src/narrator.ts`:** Existing ceremony event consumer. Extended to render live worker stacks and wave banners from manifest-driven events, not just raw JSON lines.

## Architectural Patterns

### Pattern 1: Plan-Only Manifest + Finalizer Handoff

**What:** Go generates a manifest without mutating state. TypeScript dispatches workers from the manifest. Go finalizer validates and commits state atomically.

**When to use:** All lifecycle commands that mutate colony state (plan, build, continue, seal).

**Trade-offs:**
- Pro: Go remains sole state authority. TypeScript cannot corrupt state.
- Pro: Manifests are testable JSON contracts.
- Con: Two process hops (Go -> TS -> Go) add latency. Acceptable for developer tools.
- Con: Requires discipline: TypeScript must never write `.aether/data/` directly.

**Example:**
```typescript
// host.ts — build lifecycle
async function runBuild(phase: number): Promise<void> {
  // 1. Go generates manifest without state mutation
  const manifestJson = await goExec('build', `--plan-only`, `--phase`, String(phase));
  const manifest = parseBuildManifest(manifestJson);

  // 2. TypeScript renders ceremony from manifest
  renderCeremony(manifest);

  // 3. TypeScript dispatches workers per manifest waves
  const results = await dispatchWaves(manifest.dispatches, manifest.contract);

  // 4. Write completion JSON to temp file (NOT .aether/data)
  const completionFile = await writeCompletionJson(results);

  // 5. Go finalizer validates and commits state
  await goExec('build-finalize', String(phase), `--completion-file`, completionFile);
}
```

### Pattern 2: Spawn-Log / Spawn-Complete Bookkeeping

**What:** Every worker spawn and completion is recorded via Go CLI commands, not by writing state files directly.

**When to use:** Every worker dispatch in the TypeScript control plane.

**Trade-offs:**
- Pro: Go owns the spawn ledger. TypeScript only requests entries.
- Pro: Audit trail is consistent across platforms.
- Con: Requires Go CLI subprocess calls for every spawn. Batch where possible.

**Example:**
```typescript
// dispatcher.ts
async function dispatchWorker(worker: DispatchWorker): Promise<WorkerResult> {
  await goExec('spawn-log', `--caste`, worker.caste, `--task`, worker.taskId);

  const result = await platform.spawn(worker); // Platform-specific

  await goExec('spawn-complete', `--caste`, worker.caste, `--task`, worker.taskId, `--status`, result.status);
  return result;
}
```

### Pattern 3: Platform Adapter Interface

**What:** A common interface for spawning workers across Claude Code, OpenCode, and Codex. Each platform implements the interface.

**When to use:** When the control plane needs to dispatch workers without knowing the platform specifics.

**Trade-offs:**
- Pro: Platform parity is enforced by shared interface.
- Pro: New platforms add an adapter, not a rewrite.
- Con: Lowest common denominator may miss platform-specific optimizations.

**Example:**
```typescript
// platform/types.ts
interface PlatformAdapter {
  readonly name: 'claude' | 'opencode' | 'codex';
  spawn(worker: DispatchWorker): Promise<WorkerResult>;
  isAvailable(): Promise<boolean>;
}

// platform/claude.ts
class ClaudeAdapter implements PlatformAdapter {
  readonly name = 'claude';
  async spawn(worker: DispatchWorker): Promise<WorkerResult> {
    // Use Claude Code Task tool
    return await taskToolSpawn(worker);
  }
  async isAvailable(): Promise<boolean> {
    return process.env.CLAUDE_CODE === '1';
  }
}
```

### Pattern 4: Ceremony Event Stream

**What:** Go emits structured JSON events to stdout. TypeScript narrator consumes and renders them. Visual output is never parsed as truth.

**When to use:** All user-visible lifecycle commands.

**Trade-offs:**
- Pro: Clean separation: Go owns event truth, TS owns rendering.
- Pro: Events are testable and versionable.
- Con: Requires `AETHER_OUTPUT_MODE=json` or similar protocol.

**Example:**
```typescript
// narrator.ts
interface CeremonyEvent {
  type: 'spawn-plan' | 'wave-start' | 'worker-complete' | 'closeout';
  phase?: number;
  wave?: number;
  caste?: string;
  workerName?: string;
  status?: 'success' | 'failure' | 'skipped';
  message?: string;
}

function renderEvent(event: CeremonyEvent): string {
  switch (event.type) {
    case 'wave-start':
      return `\n── Wave ${event.wave} ──\n`;
    case 'worker-complete':
      return `${casteEmoji(event.caste)} ${event.workerName}  ${event.status}\n`;
    // ...
  }
}
```

## Data Flow

### Request Flow (Build Lifecycle)

```
User: /ant-build 1
    ↓
[Wrapper] → calls TS host adapter (or TS host is the wrapper)
    ↓
[TS Host] → shells out: aether build 1 --plan-only
    ↓
[Go Runtime] → generates codexBuildManifest JSON (no state mutation)
    ↓
[TS Host] → parseBuildManifest(manifestJson)
    ↓
[TS Host] → renderCeremony(manifest) → stdout/events
    ↓
[TS Dispatcher] → for each wave in manifest.dispatches:
    → for each worker in wave:
        → platformAdapter.spawn(worker)
        → aether spawn-log --caste X --task Y
        → await worker result
        → aether spawn-complete --caste X --task Y --status Z
    ↓
[TS Host] → writeCompletionJson(results) → /tmp/aether-completion-*.json
    ↓
[TS Host] → shells out: aether build-finalize 1 --completion-file /tmp/...
    ↓
[Go Finalizer] → validate manifest provenance
                → merge worker results
                → UpdateJSONAtomically(COLONY_STATE.json)
    ↓
[Go Runtime] → emit final ceremony event
    ↓
[TS Narrator] → render closeout
```

### State Management

```
Go State Store (.aether/data/)
    ↑ (atomic writes, file locking)
Go Finalizer Engine
    ↑ (validation, provenance checks)
Go Manifest Generator
    ↑ (plan-only, no mutation)
TypeScript Host Adapter
    ↑ (orchestration, never writes .aether/data/)
Platform Wrappers / User
```

### Key Data Flows

1. **Manifest Flow:** Go generates JSON manifest -> TS parses and validates -> TS dispatches from manifest fields -> Go finalizer receives completion JSON. This is the core integration boundary.
2. **Event Flow:** Go emits ceremony events -> TS narrator consumes and renders -> User sees live worker stacks. Events are read-only; TS never writes state based on events.
3. **Spawn Ledger Flow:** TS requests spawn-log/spawn-complete via Go CLI -> Go updates internal spawn tracking -> Go finalizer references spawn ledger for provenance. TS never writes spawn ledger files directly.

## Scaling Considerations

| Scale | Architecture Adjustments |
|-------|--------------------------|
| 1 colony, 1 user | TS host runs inline in wrapper process. No background service needed. |
| Multiple colonies, 1 user | TS host is stateless per invocation. Go state remains per-repo. No shared TS state. |
| Multiple users / CI | TS host runs in CI process. Go binary is the only installed artifact. No TS daemon. |

### Scaling Priorities

1. **First bottleneck:** Subprocess latency from TS -> Go CLI calls. Mitigation: batch spawn-log calls, use `--plan-only` JSON output directly without temp files where possible.
2. **Second bottleneck:** Platform worker concurrency limits (Claude Code Task tool limits, OpenCode agent slots). Mitigation: respect `codexDispatchContract` worker counts and timeouts; queue workers if platform limits exceeded.

## Anti-Patterns

### Anti-Pattern 1: TypeScript Writes `.aether/data/` Directly

**What people do:** The TS control plane writes COLONY_STATE.json, session files, or spawn ledgers directly.

**Why it's wrong:** Violates the safety kernel boundary. Corrupts state if TS has a bug or races with Go. Breaks atomic write guarantees.

**Do this instead:** All state writes go through Go finalizers. TS writes completion JSON to temp files outside `.aether/data/` and passes paths to Go finalizers.

### Anti-Pattern 2: TypeScript Parses Visual Output as Truth

**What people do:** TS parses ANSI-colored banners or progress bars to determine worker status or phase state.

**Why it's wrong:** Visual output is for humans, not machines. It changes between versions. Parsing it creates brittle integrations.

**Do this instead:** Use `AETHER_OUTPUT_MODE=json` for machine-readable output. Parse JSON manifests and events, not visual text.

### Anti-Pattern 3: TypeScript Reimplements Planning Logic

**What people do:** TS control plane decides which workers to spawn, in what order, with what skills — duplicating Go manifest generation.

**Why it's wrong:** Splits the source of truth. Go tests validate manifest logic; TS reimplementation would not be covered. Drift is inevitable.

**Do this instead:** TS consumes Go-generated manifests and dispatches exactly what the manifest specifies. If the manifest is wrong, fix Go.

### Anti-Pattern 4: Classic Wrapper-Owned State Mutation

**What people do:** Restore Classic v5.4.0 behavior where wrappers read and wrote COLONY_STATE.json, watch files, and session state directly.

**Why it's wrong:** This was the behavior that caused the migration regressions. It bypasses Go validation, atomic writes, and provenance checks.

**Do this instead:** Restore Classic orchestration *feel* (live workers, wave banners, caste labels) but not Classic state ownership. Use plan-only manifests and finalizers.

## Integration Points

### External Services

| Service | Integration Pattern | Notes |
|---------|---------------------|-------|
| Claude Code Task tool | TS platform adapter spawns subagents via Task tool | Requires Claude Code environment. Task tool has concurrency limits. |
| OpenCode agents | TS platform adapter spawns agents via OpenCode protocol | Requires OpenCode environment. Agent availability varies. |
| Codex CLI | TS platform adapter spawns `aether` subprocesses | Codex is runtime-native; TS host may be minimal or skipped for Codex. |
| Go CLI (`aether`) | TS shells out for manifests, finalizers, spawn-log, spawn-complete | Use JSON output mode. Parse stdout, not stderr or visual output. |

### Internal Boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| TS Host <-> Go Manifest Generator | Subprocess + JSON stdout | TS calls `aether build --plan-only`; Go returns JSON manifest. |
| TS Host <-> Go Finalizer | Subprocess + temp file path | TS writes completion JSON to temp file; Go reads and validates. |
| TS Host <-> Go Spawn Ledger | Subprocess CLI calls | TS calls `aether spawn-log` / `aether spawn-complete`. |
| TS Host <-> TS Narrator | In-process function calls | Host passes ceremony events to narrator for rendering. |
| TS Host <-> Platform Adapters | In-process async calls | Adapters spawn platform-specific workers. |
| Go Runtime <-> Editable Assets | File reads (YAML, Markdown, TOML) | Go reads command playbooks and agent definitions at runtime. |

## Sources

- [S1] Wrapper-runtime ownership contract — `.aether/docs/wrapper-runtime-ux-contract.md`
- [S3] Current command-guide catalog — `cmd/command_guide.go`
- [S9] Current build wrapper YAML with restore-real-wrapper-orchestration contract — `.aether/commands/build.yaml`
- [S15] build-finalize manifest and provenance validation — `cmd/codex_build_finalize.go`
- [S55] codexBuildManifest owns dispatch metadata, execution plans, skill sections, boundary guidance, and review depth — `cmd/codex_build.go`
- [S58] Go ceremony command renders spawn-plan, wave-start, worker-complete, and closeout from lifecycle JSON — `cmd/ceremony_cmd.go`
- [S60] Current TypeScript package is a ceremony narrator, not a lifecycle control plane — `.aether/ts/package.json`; `.aether/ts/narrator.ts`
- [S65] Ceremony Revival via Bundled TypeScript Narrator — `.aether/docs/ceremony-revival-v1.6-plan.md`
- [S67] Node.js child_process documentation — https://nodejs.org/api/child_process.html
- [S94] Current plan wrapper orchestration contract — `.aether/commands/plan.yaml:26-45`
- [S95] Current build wrapper orchestration contract — `.aether/commands/build.yaml:20-33`
- Oracle synthesis — `.aether/oracle/synthesis.md`
- Hybrid runtime strategy research — `.aether/docs/hybrid-runtime-strategy-research.md`

---
*Architecture research for: Aether Hybrid Runtime Boundary and Orchestration Recovery*
*Researched: 2026-05-12*
