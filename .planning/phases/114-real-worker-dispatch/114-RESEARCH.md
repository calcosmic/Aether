# Phase 114: Real Worker Dispatch - Research

**Researched:** 2026-05-13
**Domain:** TypeScript orchestration host worker dispatch (Claude Code, OpenCode, Codex CLI)
**Confidence:** HIGH

## Summary

The TypeScript orchestration host at `.aether/ts-host/` currently simulates worker dispatch with 100ms delays. Phase 114 must replace this simulation with real platform worker spawning across Claude Code, OpenCode, and Codex CLI.

The Go runtime (`cmd/codex_build.go`) already has a mature worker dispatch system via `pkg/codex/` that supports all three platforms. The TS host's job is to replicate the *orchestration* layer (wave grouping, parallel dispatch, error handling) while delegating actual worker invocation to the platform CLIs as subprocesses, exactly as the Go runtime does.

**Primary recommendation:** Build a `PlatformDispatcher` abstraction in the TS host that shells out to `claude`, `opencode`, or `codex` CLI binaries with assembled prompts, reusing the Go runtime's prompt assembly and claims-parsing logic as the specification.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Worker prompt assembly | TS host | Go runtime (specification) | TS host must build prompts matching Go's `AssemblePrompt`/`AssembleHostedPrompt` output so platform CLIs receive identical instructions |
| Platform CLI invocation | TS host | — | TS host spawns `claude`/`opencode`/`codex` subprocesses directly from Node.js |
| Wave sequencing & parallel dispatch | TS host | — | TS host groups dispatches by wave and runs parallel waves via `Promise.all` |
| Spawn-log / spawn-complete | Go runtime | TS host (caller) | TS host calls `aether spawn-log` and `aether spawn-complete` before/after each worker |
| State finalization | Go runtime | — | TS host builds completion JSON and calls `aether build-finalize` |
| Claims parsing | TS host | — | TS host parses the trailing JSON block from CLI stdout, matching Go's `ParseWorkerOutput` |
| Agent definition resolution | TS host | — | TS host resolves `.claude/agents/ant/*.md`, `.opencode/agents/*.md`, `.codex/agents/*.toml` paths |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Node.js | >=20 | Runtime | Already required by TS host; `child_process.spawn` for subprocess dispatch |
| TypeScript | 5.9.3 | Language | Existing TS host stack |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `child_process` | built-in | Spawn platform CLIs | All real worker dispatches |
| `node:readline` | built-in | Parse NDJSON / line-delimited CLI output | Codex `--output-last-message` and hosted platform JSON streams |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Subprocess CLI dispatch | Platform SDKs (Anthropic SDK, OpenAI SDK) | Would bypass agent definitions, skill injection, and pheromone assembly that the CLI handles natively. Much higher implementation cost. |
| `execFileSync` | `spawn` + streaming | `spawn` is required for long-running workers (10 min timeout) and real-time progress observation. `execFileSync` blocks the event loop. |

## Architecture Patterns

### System Architecture Diagram

```
+-----------------------------------------------------------+
|                    TS Host (Node.js)                       |
|  +-------------------+  +-----------------------------+   |
|  | Lifecycle         |  | PlatformDispatcher          |   |
|  | - plan --plan-only|  | - detectPlatform()          |   |
|  | - build-finalize  |  | - spawnWorker(config)       |   |
|  | - dispatchWorkers |  | - parseClaims(stdout)       |   |
|  +--------+----------+  +--------------+--------------+   |
|           |                            |                  |
|  +--------v----------+  +--------------v--------------+   |
|  | WaveOrchestrator  |  | PromptAssembler             |   |
|  | - groupByWave()   |  | - loadAgentDef()            |   |
|  | - Promise.all()   |  | - injectSkills()            |   |
|  | - timeout+retry   |  | - injectPheromones()        |   |
|  +--------+----------+  +--------------+--------------+   |
|           |                            |                  |
+-----------|----------------------------|------------------+
            |                            |
            v                            v
   +--------+--------+          +--------+--------+
   | Go CLI          |          | Platform CLI    |
   | spawn-log       |          | claude -p ...   |
   | spawn-complete  |          | opencode run ...|
   | build-finalize  |          | codex exec ...  |
   +-----------------+          +-----------------+
```

### Recommended Project Structure

```
.aether/ts-host/src/
├── worker-dispatch.ts          # Existing — extend with real dispatch
├── platform-dispatcher.ts      # NEW — PlatformDispatcher abstraction
├── prompt-assembler.ts         # NEW — Agent def loading + prompt assembly
├── claims-parser.ts            # NEW — Parse worker claims JSON from stdout
├── wave-orchestrator.ts        # NEW — Wave grouping, parallel dispatch, retry
├── lifecycle.ts                # Existing — wire in real dispatch
├── types.ts                    # Existing — add WorkerConfig, WorkerResult
└── go-bridge.ts                # Existing — spawn-log / spawn-complete calls
```

### Pattern 1: PlatformDispatcher Abstraction
**What:** A strategy-pattern dispatcher that selects the correct CLI binary and argument format based on the active platform.
**When to use:** Every real worker dispatch. The dispatcher is instantiated once per lifecycle and reused.
**Example:**
```typescript
// Source: pkg/codex/platform_dispatch.go (Go specification)
interface PlatformDispatcher {
  platform: "claude" | "opencode" | "codex";
  isAvailable(): Promise<boolean>;
  spawnWorker(config: WorkerConfig): Promise<WorkerResult>;
}

// Claude: claude -p --output-format json --json-schema <schema> --agent <agent> --permission-mode bypassPermissions <prompt>
// OpenCode: opencode run --agent <agent> --format json <prompt>
// Codex: codex --sandbox workspace-write --ask-for-approval never exec --json --ephemeral --output-last-message <file> --output-schema <file> <prompt>
```

### Pattern 2: Prompt Assembly
**What:** Load the platform-specific agent definition file, prepend colony-prime context, skills, pheromones, handoffs, and the task brief, then append the response contract.
**When to use:** Before every worker spawn.
**Example:**
```typescript
// Source: pkg/codex/worker.go AssemblePrompt / AssembleHostedPrompt
function assemblePrompt(config: WorkerConfig): string {
  const agentDef = loadAgentDefinition(config.agentPath); // .md or .toml
  const contextCapsule = buildContextCapsule(config.root);
  const skillSection = resolveSkillSection(config.caste, config.task);
  const pheromoneSection = resolvePheromoneSection();
  const handoffSection = renderWorkerHandoffSection(config.workflow, config.phase, config.workerName);
  const prompt = [
    agentDef.developerInstructions,
    contextCapsule,
    handoffSection,
    skillSection,
    pheromoneSection,
    config.taskBrief,
    renderResponseContract(config),
  ].join("\n\n");
  return prompt;
}
```

### Pattern 3: Claims Parsing
**What:** Extract the trailing JSON object from CLI stdout/stderr and validate it against the worker claims schema.
**When to use:** After every worker subprocess exits.
**Example:**
```typescript
// Source: pkg/codex/worker.go ParseWorkerOutput
function parseWorkerOutput(output: string): WorkerClaims {
  // 1. Try direct JSON parse
  // 2. Strip code fences
  // 3. Walk backward from last "}" to find matching "{"
  // 4. Validate required fields: ant_name, caste, task_id, status, summary, files_created, files_modified, tests_written, tool_count, blockers, spawns, handoff
}
```

### Anti-Patterns to Avoid
- **Do NOT try to use Claude Code's or OpenCode's internal APIs or SDKs.** The Go runtime shells out to the CLI binaries; the TS host must do the same. There is no supported Node.js SDK for spawning Claude Code agents.
- **Do NOT write completion files to `.aether/data/`.** The boundary contract (`boundary-reference.ts`) forbids this. Always use `writeCompletionFile` in tmpdir and pass the path to Go finalizers.
- **Do NOT block the event loop with sync subprocess calls.** Workers have 10-minute timeouts; use `spawn` with streaming output, not `execFileSync`.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Platform CLI detection | Custom PATH search | `which` + `AETHER_CLAUDE_PATH` / `AETHER_OPENCODE_PATH` / `AETHER_CODEX_PATH` env vars | Go runtime already uses these env vars; consistency matters |
| Agent definition parsing | Hand-rolled TOML/YAML parser | `js-yaml` (already in deps) + a lightweight TOML parser (`@iarna/toml` or similar) | Agent defs are TOML for Codex, YAML frontmatter for Claude/OpenCode |
| JSON schema validation | Manual field checks | `zod` or `ajv` | Worker claims have 12 required fields; schema validation catches drift |
| Process timeout | `setTimeout` + `kill` | `AbortController` + `child.kill("SIGTERM")` | Cleaner cancellation, works with Node.js streams |

**Key insight:** The Go runtime already solved all of these problems. The TS host's implementation should be a direct port of the Go logic, not a reinvention.

## Runtime State Inventory

This phase is a greenfield implementation (replacing simulation with real dispatch). No runtime state rename/refactor/migration is required.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | None — TS host does not persist state | N/A |
| Live service config | None | N/A |
| OS-registered state | None | N/A |
| Secrets/env vars | `AETHER_CLAUDE_PATH`, `AETHER_OPENCODE_PATH`, `AETHER_CODEX_PATH`, `AETHER_CODEX_REAL_DISPATCH` | Documented in Go code; TS host should honor same vars |
| Build artifacts | `.aether/ts-host/dist/` will need rebuild after source changes | `npm run build` |

## Common Pitfalls

### Pitfall 1: Platform CLI Unavailability
**What goes wrong:** The TS host tries to spawn `claude` but the binary is not installed or not authenticated. The dispatch fails with a cryptic ENOENT or auth error.
**Why it happens:** The Go runtime probes availability with `claude auth status --json`, `opencode auth list`, and `codex login status`. The TS host must do the same.
**How to avoid:** Implement an `isAvailable()` check for each platform before dispatch, matching Go's `Availability()` methods in `pkg/codex/platform_dispatch.go`.
**Warning signs:** `spawn` returns ENOENT; stdout is empty; auth status JSON missing `loggedIn: true`.

### Pitfall 2: Prompt Assembly Drift
**What goes wrong:** The TS host assembles a prompt that differs subtly from the Go runtime's prompt. The worker behaves differently (misses skills, ignores pheromones, wrong response format).
**Why it happens:** The Go runtime's `AssemblePrompt` and `renderResponseContract` functions are the source of truth. Any deviation causes behavioral divergence.
**How to avoid:** Port the Go prompt assembly logic line-for-line. Use the same agent definition paths, skill resolution, and response contract text.
**Warning signs:** Workers return missing fields in claims JSON; skills are not activated; pheromones are ignored.

### Pitfall 3: Claims Parsing Failure
**What goes wrong:** The platform CLI returns the worker claims JSON embedded in ANSI-colored output, JSONL events, or markdown code fences. The TS host fails to extract it.
**Why it happens:** Claude and OpenCode return JSON inside streaming event structures. Codex writes to `--output-last-message`. Each platform has a different output shape.
**How to avoid:** Reuse Go's `parseHostedWorkerOutput` and `hostedJSONTextCandidates` logic. Strip ANSI codes, handle JSONL lines, and walk nested event structures.
**Warning signs:** `parseWorkerOutput` returns "no JSON found"; claims have empty required fields.

### Pitfall 4: Blocking the Event Loop
**What goes wrong:** Using `execFileSync` or `execSync` for worker dispatch blocks the TS host's event loop, preventing parallel dispatch and stalling the event bridge.
**Why it happens:** Workers run for minutes. Synchronous subprocess calls freeze the process.
**How to avoid:** Always use `spawn` with async Promise wrappers. Collect stdout/stderr into buffers, then resolve when the process exits.
**Warning signs:** Event bridge stops emitting; parallel waves run sequentially; UI freezes.

### Pitfall 5: Boundary Contract Violation
**What goes wrong:** The TS host writes worker results or completion files directly to `.aether/data/` instead of calling Go finalizers.
**Why it happens:** Convenience — "I'll just write this JSON file quickly."
**How to avoid:** The `assertNoDirectDataWrites` guard in `go-bridge.ts` throws on violation. Always use `writeCompletionFile` (tmpdir) + `aether build-finalize`.
**Warning signs:** `BoundaryViolationError` thrown; Go tests fail with `TestBoundaryContract_NoStateWritesDuringOrchestration`.

## Code Examples

### Spawning a Claude Code Worker
```typescript
// Source: pkg/codex/platform_dispatch.go ClaudeDispatcher.InvokeWithProgress
async function spawnClaudeWorker(config: WorkerConfig): Promise<WorkerResult> {
  const schemaJSON = JSON.stringify(workerClaimsSchema());
  const args = [
    "-p",
    "--output-format", "json",
    "--json-schema", schemaJSON,
    "--agent", config.agentName,
    "--permission-mode", "bypassPermissions",
    assemblePrompt(config),
  ];
  const child = spawn("claude", args, { cwd: config.root, env: process.env });
  // ... collect stdout/stderr, parse claims on exit
}
```

### Spawning an OpenCode Worker
```typescript
// Source: pkg/codex/platform_dispatch.go OpenCodeDispatcher.InvokeWithProgress
async function spawnOpenCodeWorker(config: WorkerConfig): Promise<WorkerResult> {
  const args = [
    "run",
    "--agent", config.agentName,
    "--format", "json",
    renderOpenCodeSubagentDispatchPrompt(config, assemblePrompt(config)),
  ];
  const child = spawn("opencode", args, { cwd: config.root, env: process.env });
  // ... collect stdout/stderr, parse claims on exit
}
```

### Spawning a Codex Worker
```typescript
// Source: pkg/codex/worker.go RealInvoker.InvokeWithProgress
async function spawnCodexWorker(config: WorkerConfig): Promise<WorkerResult> {
  const lastMessagePath = await writeTempFile("aether-codex-last-*.json");
  const schemaPath = await writeTempFile("aether-codex-schema-*.json", JSON.stringify(workerClaimsSchema()));
  const args = [
    "--sandbox", "workspace-write",
    "--ask-for-approval", "never",
    "exec",
    "--json",
    "--ephemeral",
    "--skip-git-repo-check",
    "--output-last-message", lastMessagePath,
    "--output-schema", schemaPath,
    "--add-dir", codexHomeDir(),
  ];
  const child = spawn("codex", args, { cwd: config.root, env: process.env });
  child.stdin?.write(assemblePrompt(config));
  child.stdin?.end();
  // ... wait for exit, read lastMessagePath, parse claims
}
```

### Parallel Wave Dispatch
```typescript
// Source: pkg/codex/dispatch.go DispatchWaveWithObserver
async function dispatchWave(
  dispatches: WorkerDispatch[],
  parallel: boolean
): Promise<DispatchResult[]> {
  if (!parallel || dispatches.length === 1) {
    const results: DispatchResult[] = [];
    for (const d of dispatches) {
      results.push(await invokeDispatch(d));
    }
    return results;
  }
  return Promise.all(dispatches.map(d => invokeDispatch(d)));
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Go runtime owns full dispatch (build, continue, plan) | TS host owns orchestration; Go owns state finalization | Phase 109+ (TS host prototype) | Separation of concerns: TS host can do visual ceremony and parallel dispatch without recompiling Go |
| Simulated 100ms workers | Real platform CLI subprocesses | Phase 114 (this phase) | Workers actually modify code, run tests, and return real claims |
| Wrapper markdown spawns agents via platform Task tool | TS host spawns via CLI subprocess | Phase 114 (this phase) | Removes dependency on Claude Code / OpenCode wrapper environment for dispatch |

**Deprecated/outdated:**
- `simulateWorkers: true` flag in `DispatchOptions`: will remain for testing but default to `false` in production.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The `claude`, `opencode`, and `codex` CLI binaries are available in PATH or via `AETHER_*_PATH` env vars when the TS host runs. | Environment Availability | If binaries are missing, real dispatch fails. The TS host should fall back to simulation or report clear errors. |
| A2 | The platform CLIs accept the same argument shapes in their current versions as documented in the Go code (tested as of 2026-05-13). | Standard Stack | CLI flags may change in future versions. Version pinning or flag detection may be needed. |
| A3 | Agent definition files (`.claude/agents/ant/*.md`, `.opencode/agents/*.md`, `.codex/agents/*.toml`) are present in the repo or hub when the TS host runs. | Prompt Assembly | Missing agent defs cause worker startup failure. The TS host should validate paths before spawn. |
| A4 | The TS host can reuse the Go runtime's prompt assembly logic as a specification without licensing issues (same repo, same author). | Code Examples | N/A — same codebase. |

## Open Questions

1. **Should the TS host support the `--dispatch-workers` opt-in flag from the Go manifest?**
   - What we know: The Go manifest includes `worker_dispatch_opt_in` and `dispatch_mode` fields. The wrapper markdown checks these before spawning.
   - What's unclear: Whether the TS host should respect `dispatch_mode: "plan-only"` and refuse real dispatch, or always dispatch real workers when `simulateWorkers: false`.
   - Recommendation: Respect the manifest's `dispatch_mode`. If it is `"plan-only"`, refuse real dispatch and error. If `"queen-led"` or `"external-task"`, proceed with real dispatch.

2. **How should the TS host handle the `parallel_mode` field ("in-repo" vs "worktree")?**
   - What we know: Go runtime has `ModeInRepo` and `ModeWorktree` with different dispatch behavior (`dispatchCodexBuildWorkersInRepo` vs worktree isolation).
   - What's unclear: Whether the TS host needs to implement worktree isolation or can start with in-repo only.
   - Recommendation: Start with in-repo parallel dispatch only. Worktree isolation is complex (git worktree creation, baseline snapshotting, claim reconciliation) and can be deferred to a later phase.

3. **Should the TS host emit ceremony events to the Go event bus, or maintain its own event stream?**
   - What we know: The Go runtime emits `ceremony.build.spawn`, `ceremony.build.wave.start`, etc. The TS host's event bridge reads these from the JSONL file.
   - What's unclear: Whether the TS host should write events back to the JSONL bus or just render them locally.
   - Recommendation: The TS host should render events locally via the narrator. Writing to the JSONL bus is Go-owned. If cross-process event visibility is needed, use the Go CLI's `event-bus-publish` command.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Node.js | TS host runtime | ✓ | >=20 | — |
| `aether` Go binary | Go bridge, finalizers | ✓ | v1.0.34 | — |
| `claude` CLI | Claude worker dispatch | [ASSUMED] | — | Skip Claude workers, log warning |
| `opencode` CLI | OpenCode worker dispatch | [ASSUMED] | — | Skip OpenCode workers, log warning |
| `codex` CLI | Codex worker dispatch | [ASSUMED] | — | Skip Codex workers, log warning |
| Git | Worktree mode (deferred) | ✓ | — | In-repo mode fallback |

**Missing dependencies with no fallback:**
- None for the core TS host. If no platform CLIs are available, the TS host can fall back to simulation mode for testing.

**Missing dependencies with fallback:**
- Missing `claude`/`opencode`/`codex` binaries: fall back to simulation mode or report clear availability errors.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Node.js built-in test runner (`node:test`) + `node:assert/strict` |
| Config file | None — see Wave 0 |
| Quick run command | `npm test` (runs `tsx --test test/*.test.ts`) |
| Full suite command | `npm test` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| TS-01 | TS host dispatches real platform workers instead of simulation | integration | `npm test -- test/worker-dispatch.test.ts` | ✅ Existing |
| TS-02 | Workers within a wave run in parallel | integration | `npm test -- test/worker-dispatch.test.ts` | ✅ Existing |
| TS-03 | Worker errors handled with retry, timeout, graceful fallback | integration | `npm test -- test/worker-dispatch.test.ts` | ✅ Existing |

### Sampling Rate
- **Per task commit:** `npm test -- test/worker-dispatch.test.ts`
- **Per wave merge:** `npm test`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `test/platform-dispatcher.test.ts` — covers platform detection and availability probing
- [ ] `test/prompt-assembler.test.ts` — covers agent def loading and prompt assembly parity with Go
- [ ] `test/claims-parser.test.ts` — covers claims parsing for all three platform output formats
- [ ] `test/wave-orchestrator.test.ts` — covers wave grouping, parallel dispatch, timeout, retry

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | — |
| V3 Session Management | no | — |
| V4 Access Control | yes | TS host must not write to `.aether/data/` (boundary contract) |
| V5 Input Validation | yes | Validate all worker claims JSON before passing to Go finalizers; sanitize paths in `files_created`/`files_modified` |
| V6 Cryptography | no | — |

### Known Threat Patterns for Worker Dispatch

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Path traversal in worker claims | Tampering | Normalize and validate all file paths against repo root; reject absolute paths and `..` segments |
| Subprocess injection via task brief | Tampering | Do NOT pass task brief through shell interpolation; use `spawn` with array args |
| Boundary contract violation | Elevation of Privilege | `assertNoDirectDataWrites` throws on violation; Go tests enforce at CI time |

## Sources

### Primary (HIGH confidence)
- `pkg/codex/worker.go` — `WorkerInvoker`, `FakeInvoker`, `RealInvoker`, `ParseWorkerOutput`, prompt assembly, claims schema
- `pkg/codex/platform_dispatch.go` — `ClaudeDispatcher`, `OpenCodeDispatcher`, `CodexDispatcher`, availability probing, agent path resolution
- `pkg/codex/dispatch.go` — `DispatchWaveWithObserver`, `GroupByWave`, parallel dispatch logic
- `cmd/codex_build.go` — Build manifest generation, dispatch creation, `executeCodexBuildDispatches`
- `cmd/codex_build_finalize.go` — `build-finalize` command, completion file schema
- `cmd/spawn.go` — `spawn-log` and `spawn-complete` commands
- `.aether/ts-host/src/worker-dispatch.ts` — Current simulated dispatch implementation
- `.aether/ts-host/src/lifecycle.ts` — Current lifecycle orchestrator
- `.aether/ts-host/src/types.ts` — TypeScript type definitions matching Go structs
- `.aether/ts-host/src/go-bridge.ts` — Go CLI invocation bridge
- `.aether/ts-host/src/boundary-reference.ts` — Boundary contract enforcement

### Secondary (MEDIUM confidence)
- `.claude/commands/ant/build.md` — Wrapper markdown showing how Claude Code spawns agents via Task tool (not CLI subprocess)
- `.opencode/commands/ant/build.md` — Same for OpenCode
- `cmd/queen_wave_lifecycle.go` — Wave lifecycle orchestration in Go (recovery budget, escalation)
- `cmd/codex_build_worktree.go` — Worktree dispatch logic (deferred for TS host)

### Tertiary (LOW confidence)
- None — all claims verified against codebase.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries are existing dependencies or Node.js built-ins.
- Architecture: HIGH — Go runtime provides a complete specification to port.
- Pitfalls: HIGH — all pitfalls are documented in Go code and boundary contract tests.

**Research date:** 2026-05-13
**Valid until:** 2026-06-13 (stable — platform CLI flags rarely change)
