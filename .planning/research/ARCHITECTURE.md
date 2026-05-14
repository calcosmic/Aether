# Architecture Research: v1.17 Classic Restoration — Go Events → TS Host → Wrapper Ceremony

**Project:** Aether v1.17 Classic Restoration
**Researched:** 2026-05-13
**Confidence:** HIGH (based on direct source code analysis)

## Executive Summary

Aether v1.17 restores the living orchestration behavior lost during the Bash-to-Go migration. The architecture is a three-layer pipeline: **Go emits structured events** (from `pkg/events/` and `cmd/ceremony_emitter.go`), the **TypeScript host subscribes to those events and orchestrates platform worker dispatch** (`.aether/ts-host/`), and **wrapper markdown renders ceremony** (`.claude/commands/ant/build.md`). The boundary contract from v1.16 remains intact: Go owns all state mutations; the TS host never writes to `.aether/data/`.

The key integration question is: *how does the TS host consume events in real time?* The Go runtime already has three event-delivery mechanisms: (1) a **subprocess narrator pipe** (`cmd/narrator_launcher.go` → Node.js stdin), (2) a **JSONL tail + poll** model via `event-bus-subscribe --stream`, and (3) a **WebSocket/SSE server** (`aether serve`). For the TS host control plane, the recommended path is a **hybrid: JSONL tail for startup replay + subprocess narrator pipe for live events**, because it requires no background server process, works within the existing wrapper→Go→TS call chain, and respects the boundary contract.

## Recommended Architecture

### Layer Diagram (Text)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  WRAPPER LAYER (Markdown — .claude/commands/ant/build.md)                   │
│  ────────────────────────────────────────────────────────                   │
│  • Renders spawn-plan, wave-start, worker-complete, closeout                │
│  • Invokes Go CLI: AETHER_OUTPUT_MODE=visual aether ceremony ...            │
│  • Spawns platform agents (Claude Code Tasks) with caste-labelled names     │
│  • NEVER mutates state; NEVER parses visual output as authority             │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    │ calls
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│  GO RUNTIME LAYER (cmd/ + pkg/)                                             │
│  ─────────────────────────────────────────                                  │
│  • State mutation: COLONY_STATE.json, session, pheromones (file-locked)     │
│  • Manifest generation: build --plan-only, plan --plan-only, continue ...   │
│  • Finalizers: build-finalize, plan-finalize, continue-finalize             │
│  • Ceremony emitter: cmd/ceremony_emitter.go publishes to pkg/events/bus.go │
│  • Visual rendering: cmd/ceremony_cmd.go (spawn-plan, wave-start, etc.)     │
│  • Narrator launcher: cmd/narrator_launcher.go (Node.js subprocess pipe)    │
│  • Event bus CLI: event-bus-subscribe --stream --filter "ceremony.*"        │
│  • Serve command: SSE/WebSocket server (aether serve)                       │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                    ┌───────────────┼───────────────┐
                    │               │               │
                    ▼               ▼               ▼
            ┌──────────────┐ ┌─────────────┐ ┌─────────────┐
            │ Subprocess   │ │ JSONL Tail  │ │ SSE/WS      │
            │ Narrator Pipe│ │ + Poll      │ │ Server      │
            │ (stdin NDJSON│ │ (file watch)│ │ (aether serve│
            │  events)     │ │             │ │  localhost) │
            └──────────────┘ └─────────────┘ └─────────────┘
                    │               │               │
                    └───────────────┼───────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│  TS HOST LAYER (.aether/ts-host/)                                           │
│  ─────────────────────────────────                                          │
│  • go-bridge.ts: execFileSync calls to Go CLI (JSON mode)                   │
│  • lifecycle.ts: plan → build → continue orchestration                      │
│  • worker-dispatch.ts: manifest → spawn-log → platform agent → spawn-complete│
│  • event-bridge.ts (NEW): consume Go events, drive TS-side ceremony         │
│  • ceremony-narrator.ts (NEW): render real-time ceremony from event stream  │
│  • wave-dispatcher.ts (NEW): parallel wave execution with event feedback    │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Data Flow: Plan → Build → Continue

```
Plan Flow
---------
Wrapper: /ant-plan
  └─> Go: aether plan --plan-only
        └─> Go emits: ceremony.plan.wave.start, ceremony.plan.spawn, ceremony.plan.wave.end
        └─> Go returns JSON: { plan_manifest, dispatches }
  └─> TS Host (optional): receives events via event-bridge, renders live plan ceremony
  └─> Wrapper: renders spawn plan from manifest (AETHER_OUTPUT_MODE=visual aether ceremony spawn-plan)
  └─> Wrapper: spawns planning agents (Scout, Route-Setter)
  └─> Wrapper: builds completion file, calls Go plan-finalize
        └─> Go commits state: COLONY_STATE.json updated with phases

Build Flow
----------
Wrapper: /ant-build 1
  └─> Go: aether build 1 --plan-only
        └─> Go emits: ceremony.build.prewave, ceremony.build.wave.start, ceremony.build.spawn
        └─> Go returns JSON: { dispatch_manifest, dispatches, execution_plan }
  └─> TS Host: event-bridge connects to Go event stream
  └─> TS Host: wave-dispatcher groups dispatches by execution_wave
  └─> TS Host: for each wave:
        • renders wave-start banner (via ceremony-narrator or Go CLI)
        • dispatches platform agents in parallel (worker-dispatch.ts)
        • records spawn-log / spawn-complete via Go CLI
        • Go emits ceremony.build.spawn (starting → running → completed/failed)
        • TS Host receives live status updates via event-bridge
  └─> Wrapper: renders worker-complete lines as agents return
  └─> Wrapper: builds completion packet, calls Go build-finalize
        └─> Go commits state: StateBUILT, claims, spawn-tree, phase tasks

Continue Flow
-------------
Wrapper: /ant-continue
  └─> Go: aether continue --plan-only
        └─> Go emits: ceremony.continue.wave.start, ceremony.continue.spawn, ceremony.continue.wave.end
        └─> Go returns JSON: { continue_manifest, dispatches }
  └─> TS Host: event-bridge receives verification/review events
  └─> Wrapper: spawns verification agents (Watcher, Probe, Auditor, etc.)
  └─> Wrapper: builds completion file, calls Go continue-finalize
        └─> Go commits state: gates, advance to next phase, or block
```

## Component Responsibilities

### Go Runtime (Existing, Modified)

| Component | File | Responsibility | What It Emits |
|-----------|------|----------------|---------------|
| Ceremony Emitter | `cmd/ceremony_emitter.go` | Publishes `events.CeremonyPayload` to `pkg/events/bus.go` | `ceremony.build.*`, `ceremony.plan.*`, `ceremony.continue.*` topics |
| Event Bus | `pkg/events/bus.go` | In-memory pub/sub with JSONL persistence, TTL, crash replay | `Event{ID,Topic,Payload,Source,Timestamp}` |
| Ceremony Topics | `pkg/events/ceremony.go` | Constants for all ceremony moments | 26 topics from `build.prewave` to `hive.promote` |
| Narrator Launcher | `cmd/narrator_launcher.go` | Spawns Node.js subprocess, pipes events to stdin | NDJSON `events.Event` lines |
| Build Command | `cmd/codex_build.go` | Manifest generation, worker dispatch (Go-native), state mutation | `emitBuildCeremony*` calls |
| Ceremony Command | `cmd/ceremony_cmd.go` | Visual rendering from manifest/completion files | ANSI banners, caste identity, stage markers |
| Spawn Commands | `cmd/spawn.go` | `spawn-log`, `spawn-complete`, spawn-tree CRUD | `ceremony.build.spawn` events |
| Serve Command | `cmd/serve.go` | SSE/WebSocket server for external consumers | SSE `event: <type>\ndata: <json>` |
| Event Bus CLI | `cmd/eventbus.go` | `event-bus-publish`, `event-bus-subscribe --stream`, `event-bus-replay` | NDJSON stream to stdout |

### TypeScript Host (New + Existing)

| Component | File | Responsibility | Boundary |
|-----------|------|----------------|----------|
| Go Bridge | `src/go-bridge.ts` | `execFileSync` calls to Go CLI with `AETHER_OUTPUT_MODE=json` | Reads Go output; never writes `.aether/data/` |
| Lifecycle | `src/lifecycle.ts` | Orchestrates plan → build → continue via Go plan-only + finalizers | Calls finalizers for state commits |
| Worker Dispatch | `src/worker-dispatch.ts` | Manifest → grouped waves → spawn-log → agent → spawn-complete | Simulated today; real platform dispatch next |
| **Event Bridge** | `src/event-bridge.ts` *(NEW)* | Consumes Go events via JSONL tail or subprocess pipe | Subscribes only; never publishes to bus directly |
| **Ceremony Narrator** | `src/ceremony-narrator.ts` *(NEW)* | Renders real-time ceremony from event stream (ANSI or plain) | Presentation only; no state mutation |
| **Wave Dispatcher** | `src/wave-dispatcher.ts` *(NEW)* | Parallel execution of manifest waves with event-driven status | Drives worker-dispatch; reports back to event bridge |

### Wrapper Layer (Existing, Enhanced)

| Component | File | Responsibility |
|-----------|------|----------------|
| Build Wrapper | `.claude/commands/ant/build.md` | Calls Go for manifest, renders ceremony, spawns agents, calls finalizer |
| Continue Wrapper | `.claude/commands/ant/continue.md` | Calls Go for continue manifest, spawns verification agents, calls finalizer |
| Plan Wrapper | `.claude/commands/ant/plan.md` | Calls Go for plan manifest, spawns planning agents, calls finalizer |

## Event Payload Schema

The canonical event shape is `pkg/events/event.go`:

```go
type Event struct {
    ID        string          `json:"id"`        // evt_{unix}_{4hex}
    Topic     string          `json:"topic"`     // ceremony.build.spawn
    Payload   json.RawMessage `json:"payload"`   // CeremonyPayload
    Source    string          `json:"source"`    // aether-build, aether-spawn
    Timestamp string          `json:"timestamp"` // 2006-01-02T15:04:05Z
    TTLDays   int             `json:"ttl_days"`
    ExpiresAt string          `json:"expires_at"`
}
```

The `CeremonyPayload` (from `pkg/events/ceremony.go`) is the domain-specific body:

```go
type CeremonyPayload struct {
    Phase           int      `json:"phase,omitempty"`
    PhaseName       string   `json:"phase_name,omitempty"`
    Wave            int      `json:"wave,omitempty"`
    SpawnID         string   `json:"spawn_id,omitempty"`
    Caste           string   `json:"caste,omitempty"`
    Name            string   `json:"name,omitempty"`
    TaskID          string   `json:"task_id,omitempty"`
    Task            string   `json:"task,omitempty"`
    Status          string   `json:"status,omitempty"`     // starting, running, completed, failed, timeout, blocked
    Message         string   `json:"message,omitempty"`
    Skill           string   `json:"skill,omitempty"`
    PheromoneType   string   `json:"pheromone_type,omitempty"`
    Strength        float64  `json:"strength,omitempty"`
    Completed       int      `json:"completed,omitempty"`
    Total           int      `json:"total,omitempty"`
    ToolCount       int      `json:"tool_count,omitempty"`
    TokenCount      int      `json:"token_count,omitempty"`
    FilesCreated    []string `json:"files_created,omitempty"`
    FilesModified   []string `json:"files_modified,omitempty"`
    TestsWritten    []string `json:"tests_written,omitempty"`
    Blockers        []string `json:"blockers,omitempty"`
    SuccessCriteria []string `json:"success_criteria,omitempty"`
    LoopType         string  `json:"loop_type,omitempty"`
    DetectionSignal  string  `json:"detection_signal,omitempty"`
    ActionTaken      string  `json:"action_taken,omitempty"`
}
```

### TypeScript Mirror (Recommended)

```typescript
// src/event-bridge.ts
export interface CeremonyEvent {
  id: string;
  topic: string;
  payload: CeremonyPayload;
  source: string;
  timestamp: string;
}

export interface CeremonyPayload {
  phase?: number;
  phase_name?: string;
  wave?: number;
  spawn_id?: string;
  caste?: string;
  name?: string;
  task_id?: string;
  task?: string;
  status?: "starting" | "running" | "completed" | "failed" | "timeout" | "blocked";
  message?: string;
  skill?: string;
  completed?: number;
  total?: number;
  tool_count?: number;
  files_created?: string[];
  files_modified?: string[];
  tests_written?: string[];
  blockers?: string[];
}
```

## Integration Point: How Go Streams Events to TS Host

### Option Analysis

| Mechanism | How It Works | Pros | Cons | Recommendation |
|-----------|--------------|------|------|----------------|
| **A. Subprocess Narrator Pipe** | Go spawns `node narrator.js`, pipes NDJSON events to stdin | Zero config; works today in `cmd/narrator_launcher.go`; live stream | Requires Node.js; one-way only; tied to Go lifecycle | **Primary for live events** |
| **B. JSONL Tail + Poll** | TS host runs `event-bus-subscribe --stream --filter "ceremony.*"` | No server; uses existing CLI; bidirectional possible via separate CLI calls | Polling latency (250ms default); stdout parsing | **Primary for startup replay + fallback** |
| **C. WebSocket via `aether serve`** | TS host connects to `ws://localhost:8080/ws/agents` | Lowest latency; true bidirectional | Requires background server process; port conflicts; overkill for local CLI | **Defer — useful for swarm dashboard later** |
| **D. Named Pipe / Domain Socket** | Go writes to Unix domain socket, TS reads | Low latency; no polling | Platform-specific; requires cleanup; not yet implemented | **Reject — adds complexity without benefit** |
| **E. File Watcher on JSONL** | TS watches `.aether/data/event-bus.jsonl` for changes | Simple; no subprocess | Race conditions; no atomic append guarantees on all platforms | **Reject — unreliable** |

### Recommended Hybrid Approach

**Phase 1 (v1.17): JSONL Tail + Subprocess Pipe**

1. **Startup replay:** Before dispatching any workers, the TS host calls:
   ```
   AETHER_OUTPUT_MODE=json aether event-bus-replay --topic ceremony.build.spawn --since <build_start_time>
   ```
   This gives the TS host all events already emitted (prewave, wave-start) so it can render the current state accurately.

2. **Live stream:** The TS host spawns:
   ```
   aether event-bus-subscribe --stream --filter "ceremony.build.*" --poll-interval 100ms
   ```
   as a long-running subprocess. It parses NDJSON lines from stdout and updates its internal ceremony state.

3. **Alternative (narrator pipe):** If the wrapper invokes the TS host as a subprocess (instead of the other way around), Go's existing `maybeLaunchNarrator` can pipe events directly to the TS host's stdin. This is how the old `narrator.js` worked.

**Phase 2 (v1.18+): WebSocket for Swarm Dashboard**
When the live terminal dashboard (SWARM-01) is built, `aether serve` becomes useful. The dashboard connects via WebSocket and receives all `ceremony.*` events in real time.

## Integration Point: How TS Host Signals Wrapper

The TS host does not "signal" the wrapper directly. Instead, the wrapper invokes the TS host as part of its command flow:

```
Wrapper build.md:
  1. Calls Go for manifest (JSON)
  2. Calls Go for visual spawn-plan (optional — can delegate to TS host)
  3. Invokes TS host: node .aether/ts-host/dist/host.js build-wave --manifest-file <tmp>
     └─> TS host:
         a. Reads manifest
         b. Starts event-bridge (event-bus-subscribe --stream)
         c. For each wave:
            - Renders wave-start (via ceremony-narrator.ts or Go CLI)
            - Dispatches platform agents (Claude Code Tasks)
            - Records spawn-log / spawn-complete via Go CLI
            - Receives live events, updates progress
         d. Builds completion packet
  4. Calls Go build-finalize with completion file
  5. Calls Go for visual closeout
```

The TS host writes progress to **stderr** (or a temp status file) so the wrapper can display it. The wrapper remains the user-facing orchestrator; the TS host is the execution engine.

## Integration Point: How Wrapper Renders Ceremony

Today, the wrapper renders ceremony by calling Go CLI commands:

```
AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony spawn-plan --workflow build --manifest-file <file>
AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony wave-start --workflow build --manifest-file <file> --execution-wave <n>
AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony worker-complete --workflow build --worker-file <file>
AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow build --completion-file <file>
```

In v1.17, this stays mostly the same, but with two changes:

1. **TS host can render lightweight ceremony** (wave-start banners, worker status lines) directly via `ceremony-narrator.ts`, reducing Go CLI round-trips.
2. **Go visual commands remain the canonical renderer** for complex output (spawn-plan, closeout) to ensure parity across platforms.

The wrapper decides: for simple real-time updates, use TS host output; for formal ceremony moments, use Go CLI.

## New Components Required

| Component | File | Purpose | Depends On |
|-----------|------|---------|------------|
| Event Bridge | `src/event-bridge.ts` | Connects to Go event bus (JSONL tail or narrator pipe), emits typed events to TS consumers | `go-bridge.ts` (to spawn CLI) |
| Ceremony Narrator | `src/ceremony-narrator.ts` | Renders ANSI/plain ceremony from event stream: wave banners, worker status, progress bars | `event-bridge.ts`, caste color map (YAML or inline) |
| Wave Dispatcher | `src/wave-dispatcher.ts` | Accepts manifest, groups by execution_wave, runs parallel dispatch, feeds events to event-bridge | `worker-dispatch.ts`, `event-bridge.ts` |
| Caste Config Loader | `src/caste-config.ts` | Loads shared YAML caste emoji/color/label map (CEREMONY-02) | YAML parser (or inline JSON) |

## Modified Components

| Component | File | Change |
|-----------|------|--------|
| Go Bridge | `src/go-bridge.ts` | Add `spawnGoStream` helper for long-running CLI processes (event-bus-subscribe) |
| Worker Dispatch | `src/worker-dispatch.ts` | Replace simulation with real platform dispatch; integrate with event-bridge for status updates |
| Lifecycle | `src/lifecycle.ts` | Add event-bridge startup/shutdown; wire wave-dispatcher instead of sequential loop |
| Host Entry | `src/host.ts` | Add `build-wave`, `watch-events` commands |
| Build Wrapper | `.claude/commands/ant/build.md` | Add TS host invocation step; remove redundant Go ceremony calls where TS host handles it |

## Boundary Contract Preservation

The v1.16 boundary contract (`runtime-boundary-contract.md`) must remain intact:

| Rule | How v1.17 Upholds It |
|------|----------------------|
| Go owns all state mutation | TS host still calls `build-finalize`, `plan-finalize`, `continue-finalize`. No direct `.aether/data/` writes. |
| TS host calls Go plan-only for manifests | `lifecycle.ts` unchanged — still uses `callGoJSON` with `--plan-only`. |
| TS host never invents workers | `wave-dispatcher.ts` dispatches only from `dispatch_manifest.dispatches`. |
| No visual output parsing as authority | TS host consumes NDJSON from `event-bus-subscribe`, not ANSI output. |
| Bash is glue only | No new Bash logic; TS host is Node.js. |

## Suggested Build Order (Respecting Dependencies)

```
Phase A: Foundation (can start immediately)
  1. caste-config.ts — load YAML caste maps (no dependencies)
  2. event-bridge.ts — consume Go event stream (depends on go-bridge.ts)
  3. ceremony-narrator.ts — render from events (depends on event-bridge, caste-config)

Phase B: Dispatch (depends on Phase A)
  4. wave-dispatcher.ts — parallel wave execution (depends on worker-dispatch, event-bridge)
  5. Update worker-dispatch.ts — real platform dispatch, emit events

Phase C: Integration (depends on Phase B)
  6. Update lifecycle.ts — wire wave-dispatcher, event-bridge lifecycle
  7. Update host.ts — add build-wave command
  8. Update build.md wrapper — invoke TS host, reduce Go ceremony calls

Phase D: Verification
  9. Golden parity tests (PARITY-01) — compare TS host output against v5.4 baseline
  10. E2E tests for event bridge → ceremony narrator → wave dispatcher pipeline
```

## Scalability Considerations

| Concern | At 1 worker | At 10 workers | At 50 workers |
|---------|-------------|---------------|---------------|
| Event volume | ~20 events/build | ~200 events/build | ~1000 events/build |
| JSONL file size | Negligible | ~500 KB | ~2 MB |
| Polling overhead | 100ms latency OK | 100ms OK | Consider WebSocket |
| TS host memory | <10 MB | <20 MB | <50 MB |
| Parallel wave dispatch | Sequential fine | 2-3 parallel | Needs worktree mode |

## Anti-Patterns to Avoid

| Anti-Pattern | Why Bad | What To Do Instead |
|--------------|---------|-------------------|
| TS host writes to `.aether/data/event-bus.jsonl` | Violates boundary contract; races with Go file locking | TS host subscribes via CLI or pipe; never writes |
| Wrapper parses TS host ANSI output for state | Fragile; visual output is for humans only | Wrapper uses Go JSON CLI for state; TS host events for progress |
| TS host invents workers not in manifest | Breaks provenance; finalizer will reject | Dispatch only from `dispatch_manifest.dispatches` |
| Go emits events but TS host ignores them | Wasted work; ceremony feels dead | TS host must consume and render every event |
| Ceremony rendering duplicated in Go and TS | Maintenance burden; drift | Go owns canonical complex rendering; TS owns lightweight real-time |

## Sources

- `pkg/events/bus.go` — Event bus implementation (in-memory pub/sub, JSONL persistence)
- `pkg/events/event.go` — Event struct and topic matching
- `pkg/events/ceremony.go` — Ceremony topics and payload schema
- `cmd/ceremony_emitter.go` — Build ceremony emitter, lifecycle ceremony sequences
- `cmd/ceremony_cmd.go` — Visual ceremony rendering commands
- `cmd/narrator_launcher.go` — Node.js subprocess narrator pipe
- `cmd/eventbus.go` — Event bus CLI (publish, query, replay, subscribe)
- `cmd/serve.go` — SSE/WebSocket streaming server
- `cmd/spawn.go` — spawn-log, spawn-complete with ceremony emission
- `cmd/codex_build.go` — Build manifest generation, dispatch planning
- `.aether/ts-host/src/lifecycle.ts` — Existing TS lifecycle orchestrator
- `.aether/ts-host/src/worker-dispatch.ts` — Existing worker dispatch
- `.aether/ts-host/src/go-bridge.ts` — Go CLI bridge
- `.claude/commands/ant/build.md` — Build wrapper (ceremony invocation pattern)
- `.aether/references/contracts/runtime-boundary-contract.md` — Boundary contract
