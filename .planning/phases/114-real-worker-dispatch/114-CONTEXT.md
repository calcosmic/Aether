# Phase 114: Real Worker Dispatch - Context

**Gathered:** 2026-05-13
**Status:** Ready for planning

## Phase Boundary

Replace simulated worker dispatch (100ms delays) in the TS host with real platform CLI subprocess spawning. Workers will invoke `claude`, `opencode`, or `codex` CLIs with assembled prompts, parse their JSON claims output, and report results to Go finalizers.

## Implementation Decisions

### D-01: PlatformDispatcher Abstraction
- **Decision:** A strategy-pattern dispatcher selects the correct CLI binary and argument format per platform.
- **Why:** Encapsulates platform differences (Claude vs OpenCode vs Codex flags, output formats, auth checks) behind a common interface.

### D-02: Prompt Assembly Parity with Go
- **Decision:** The TS host ports Go's `AssemblePrompt`/`AssembleHostedPrompt` logic directly, loading the same agent definitions and injecting the same context sections.
- **Why:** Any drift causes workers to behave differently (miss skills, ignore pheromones, wrong output format).

### D-03: In-Repo Parallel Dispatch First
- **Decision:** Wave orchestrator starts with in-repo parallel dispatch only. Worktree isolation is deferred.
- **Why:** Worktree creation, baseline snapshotting, and claim reconciliation are complex. In-repo dispatch provides immediate value.

### D-04: Claims Parsing with Fallback
- **Decision:** Parse trailing JSON from CLI stdout using Go's `ParseWorkerOutput` logic as spec. Strip ANSI, handle code fences, walk nested events.
- **Why:** Each platform returns claims in a different wrapper (Claude JSONL events, OpenCode streaming, Codex `--output-last-message` file).

## Canonical References

- `.planning/phases/114-real-worker-dispatch/114-RESEARCH.md` — Full research with CLI invocation patterns, pitfalls, and code examples
- `pkg/codex/worker.go` — Go prompt assembly, claims schema, output parsing
- `pkg/codex/platform_dispatch.go` — Platform detection, availability probing
- `pkg/codex/dispatch.go` — Wave grouping, parallel dispatch
- `.aether/ts-host/src/worker-dispatch.ts` — Current simulated dispatch
- `.aether/ts-host/src/lifecycle.ts` — Lifecycle orchestrator
- `.aether/ts-host/src/go-bridge.ts` — Go CLI invocation

## Existing Code Insights

### Reusable Assets
- **worker-dispatch.ts** — spawn-log/complete lifecycle is correct; just replace the simulation body
- **go-bridge.ts** — `callGoJSON`, `writeCompletionFile`, `assertNoDirectDataWrites` all ready
- **boundary-reference.ts** — Boundary contract enforcement already in place
- **types.ts** — `BuildDispatch`, `WorkerResult`, `TerminalWorkerStatus` already defined

### Integration Points
- **lifecycle.ts** — calls `dispatchWorkers()` with `simulateWorkers: true`; change to `false` when real dispatch is ready
- **event-bridge.ts** — already reads ceremony events; narrator renders them

## Deferred Ideas
- **Worktree isolation** — deferred to Phase 116+; start with `parallel_mode: "in-repo"` only
- **Platform CLI version pinning** — nice to have; current approach probes availability at runtime
