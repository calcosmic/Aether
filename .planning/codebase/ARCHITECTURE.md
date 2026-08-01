<!-- refreshed: 2026-08-01 -->
# Architecture

**Analysis Date:** 2026-08-01

## System Overview

```text
┌─────────────────────────────────────────────────────────────────────┐
│                    Aether CLI (Cobra Root)                          │
│                cmd/aether/main.go → cmd.Execute()                   │
│       rootCmd orchestrates 60+ subcommands across cmd/*.go          │
└──────────┬──────────────────────────────────────────────────────────┘
           │
           ▼
┌─────────────────────────────────────────────────────────────────────┐
│               Command Handlers (500+ files in cmd/)                  │
│  Build, Continue, Seal, Plan, Status, Learning, Pheromones         │
└──────────┬──────────────────────────────────────────────────────────┘
           │
           ▼
┌──────────────────────┬──────────────────────┬───────────────────────┐
│  Core State Layer    │  Persistence Layer   │  Worker System        │
│  Colony State        │  Atomic Writes       │  Agent Pool           │
│  `pkg/colony/`       │  File Locking        │  `pkg/agent/`         │
│                      │  `pkg/storage/`      │  Caste Dispatch       │
└──────────┬───────────┴──────────┬───────────┴───────────┬───────────┘
           │                      │                       │
           ▼                      ▼                       ▼
┌─────────────────────┐ ┌──────────────────────┐ ┌──────────────────┐
│  Event Bus          │ │  Learning Pipeline   │ │  Codex Host      │
│  Pub/Sub + TTL      │ │  Trust Scoring       │ │  TS Dispatch     │
│  `pkg/events/`      │ │  Hive Wisdom         │ │  `pkg/codex/`    │
└─────────────────────┘ │  `pkg/learn/`        │ └──────────────────┘
                         └──────────────────────┘
```

## Component Responsibilities

| Component | Responsibility | Files |
|-----------|----------------|-------|
| **CLI Entry** | Binary startup, error handling | `cmd/aether/main.go`, `cmd/root.go` |
| **Command Routing** | Cobra registration, flag parsing | `cmd/*.go` (500+ files) |
| **State Management** | Colony lifecycle, phases, tasks | `pkg/colony/colony.go` |
| **Event Bus** | Pub/sub messaging, TTL, JSONL log | `pkg/events/bus.go`, `pkg/events/event.go` |
| **Agent Pool** | Worker spawning, caste assignment | `pkg/agent/pool.go`, `pkg/agent/agent.go` |
| **Stream Manager** | Worker output multiplexing | `pkg/agent/stream_multiplexer.go` |
| **Atomic Storage** | Cross-process safe writes, locking | `pkg/storage/storage.go`, `pkg/storage/lock.go` |
| **Learning Pipeline** | Observation capture, instinct promotion | `pkg/learn/learn.go`, `pkg/learn/curator.go` |
| **Hive Wisdom** | Cross-colony knowledge, confidence | `pkg/learn/hive_store.go` |
| **Codex Dispatch** | TS host work distribution | `pkg/codex/dispatch.go`, `pkg/codex/execution_binding.go` |
| **Knowledge Graph** | Code patterns, relationships | `pkg/graph/doc.go` |
| **Output Formatting** | Terminal colors, visuals | `pkg/terminal/output.go` |
| **Streaming LLM** | Token callbacks, stream handlers | `pkg/llm/stream.go` |

## Pattern Overview

**Overall:** Event-driven CLI with layered persistence and caste-based worker orchestration

**Key Characteristics:**
- **Cobra CLI** — 60+ subcommands in `cmd/` with PersistentPreRunE initialization
- **State as JSON** — COLONY_STATE.json is source of truth, mutated atomically via `store.UpdateFile()`
- **Event-driven** — Events published to `event-bus.jsonl`, agents subscribe via Triggers
- **Caste roles** — Workers assigned (builder, watcher, scout, oracle, etc.) based on task type
- **Learning loop** — Observations scored, promoted to instincts, then hive wisdom at seal
- **Streaming output** — Real-time worker tokens via `pkg/llm` handlers

## Layers

**CLI Command Layer:**
- **Purpose:** Parse user input, mutate state, orchestrate agent spawning
- **Location:** `cmd/` (500+ files, one per command or command family)
- **Contains:** Cobra command structs, argument parsing, output formatting
- **Depends on:** `pkg/colony`, `pkg/agent`, `pkg/storage`, `pkg/codex`
- **Used by:** `aether` binary entry point

**State Management:**
- **Purpose:** Define colony lifecycle (phases, tasks, instincts, signals)
- **Location:** `pkg/colony/colony.go` and related types
- **Contains:** ColonyState struct, Phase/Task, WorktreeStatus, State constants
- **Depends on:** `pkg/storage` (persistence), Go stdlib JSON
- **Used by:** All command handlers, build/continue orchestration

**Event Bus & Pub/Sub:**
- **Purpose:** Async inter-component messaging with TTL expiry
- **Location:** `pkg/events/bus.go`, `pkg/events/event.go`
- **Contains:** Event type (id, topic, payload, ttl), Bus with topic matching
- **Depends on:** `pkg/storage` (JSONL append), Go stdlib
- **Used by:** Agent triggers, learning pipeline events, observation recording

**Worker System:**
- **Purpose:** Spawn, lifecycle, and coordinate agent execution
- **Location:** `pkg/agent/` (pool, registry, spawn tree, stream manager)
- **Contains:** Agent interface (Execute, Caste, Triggers), Pool, StreamManager
- **Depends on:** `pkg/events` (triggers), `pkg/llm` (streaming)
- **Used by:** Build/continue commands, autopilot

**Atomic Persistence:**
- **Purpose:** Crash-safe file operations with cross-process locking
- **Location:** `pkg/storage/storage.go`, `pkg/storage/lock.go`
- **Contains:** Store (atomic writes, read-modify-write), FileLocker (Unix/Windows)
- **Depends on:** OS file I/O, platform-specific locking primitives
- **Used by:** All JSON state mutations, event bus appends, learning storage

**Learning & Wisdom:**
- **Purpose:** Capture observations, score trust, promote to cross-colony hive
- **Location:** `pkg/learn/learn.go`, curator.go, hive_store.go
- **Contains:** Entry/Evidence types, LearnStore interface, Trust scoring
- **Depends on:** `pkg/storage`, SQLite (hive), Go stdlib
- **Used by:** Build completion, seal, hive read for colony-prime injection

**Codex TS Integration:**
- **Purpose:** Dispatch work to TypeScript host (Claude Code, OpenCode)
- **Location:** `pkg/codex/dispatch.go`, execution_binding.go, handoff.go
- **Contains:** Dispatch structs, ExecutionBinding, Handoff (file changes, assumptions)
- **Depends on:** `pkg/colony` (state), OS process management
- **Used by:** Build attempt workers, platform-specific wrappers

**Knowledge Graph:**
- **Purpose:** Store code relationships, patterns, insights
- **Location:** `pkg/graph/doc.go` and related types
- **Contains:** Graph types, persistence interface
- **Depends on:** `pkg/storage`
- **Used by:** Learning, pattern detection, cross-colony recommendations

## Data Flow

### Primary Path: Build Command

1. **CLI invocation:** `aether host build --phase 1` (or `/ant-build 1` from wrapper)
2. **Command handler:** `buildPhaseCmd.RunE()` in `cmd/build_flow_cmds.go`
3. **Load state:** `store.LoadJSON("COLONY_STATE.json", &state)` via `pkg/storage`
4. **Validate:** Phase pending, tasks ready, no blockers
5. **Create attempt:** Record to `build/phase-N/attempts/{attemptID}.json` with manifest
6. **Spawn workers:** Agent pool creates builder/watcher/scout via `pkg/agent`
7. **Dispatch:** Send manifest to TS host via `pkg/codex/dispatch`
8. **Stream output:** Multiplex worker stdout/stderr via `pkg/agent/stream_multiplexer`
9. **Record completion:** Update attempt record, transition phase status
10. **Emit event:** Publish to event bus for learnings pipeline
11. **Promote learnings:** Curator promotes observations to instincts
12. **Mutate state:** `store.UpdateFile()` to update COLONY_STATE.json atomically

### Secondary Path: Seal & Hive Promotion

1. **Seal invocation:** `aether host seal` (or `/ant-seal`)
2. **Archive:** Compress colony records to `.aether/archive/`
3. **Extract instincts:** Load instincts with confidence >= 0.8 from state
4. **Hive promotion:** Call `hive-promote` to abstract and store in `~/.aether/hive/wisdom.json`
5. **QUEEN update:** Append learned patterns to `~/.aether/QUEEN.md`
6. **Finalize:** Mark colony as COMPLETED

### Event Bus Flow

1. **Publish:** `bus.Publish(topic, payload)` → appends JSON line to `event-bus.jsonl`
2. **Query:** `bus.Query(pattern)` with optional ttl filter
3. **TTL pruning:** Events expire after 30 days (default), auto-pruned on read
4. **Lock protection:** File locked during append to prevent corruption across processes

## Key Abstractions

**Caste Enum:**
- Defined in: `pkg/agent/agent.go`
- Values: CasteBuilder, CasteWatcher, CasteScout, CasteOracle, CasteCurator, CasteArchitect, CasteRouteSetter, CasteColonizer, CasteArchaeologist
- Used for: Skill matching, worker spawn selection, behavioral customization

**Agent Interface:**
- Methods: Name(), Caste(), Triggers(), Execute(ctx, event)
- Extension: Implement + register to add new worker behavior
- Used by: Agent pool for dispatch, event bus for trigger matching

**Trigger System:**
- Pattern: Topic (with `*` wildcard), optional Filter map
- Matching: TopicMatch() in events package supports prefix/exact match
- Use: Agents subscribe to specific event topics for auto-activation

**Evidence & Trust Scoring:**
- Fields: RunID, Phase, Workers, FilesTouched, GatesPassed, Confidence, Timestamp
- Confidence range: 0.0 to 1.0, starts at 0.75 for pattern observations
- Promotion: Instinct promoted to hive when confidence >= 0.8 at seal

**Handoff Pattern:**
- Contents: Changed files, commands run, verification status, open decisions, assumptions
- Storage: `.aether/data/handoffs/worker-handoffs.json`
- Injection: Prepended to next worker's prompt as "Previous Worker Handoff"

**Build Attempt Record:**
- Stored at: `build/phase-N/attempts/{id}.json`
- Fields: ID, Phase, Status, StartedAt, Dispatches, Claims, WorkerRuns, History
- Lifecycle: prepared → awaiting_external → dispatching → terminal → built/failed/interrupted

## Entry Points

**Binary Entry:**
- File: `cmd/aether/main.go`
- Logic: `func main()` → `cmd.Execute()` → `rootCmd.Execute()`
- Invocation: `aether <subcommand> [args]`
- Exit: 0 on success, 1+ on error

**Root Command:**
- File: `cmd/root.go` (line 170+)
- PersistentPreRunE: Initializes `store` + `tracer` unless command in skip list
- Subcommands: Registered via `rootCmd.AddCommand()` in init blocks throughout `cmd/*.go`

**Key Command Entry Points:**
- **Build:** `cmd/build_flow_cmds.go` → buildPhaseCmd.RunE()
- **Continue:** `cmd/continue_flow_cmds.go` → continueCmd.RunE()
- **Seal:** `cmd/seal.go` → sealCmd.RunE()
- **Plan:** `cmd/plan_generate_cmd.go` → planGenerateCmd.RunE()
- **Status:** `cmd/status_display.go` → statusDisplayCmd.RunE()

## Architectural Constraints

- **Single-threaded per invocation:** Each CLI command runs sequentially; concurrency via TS host dispatch
- **Global state initialization:** `store` and `tracer` init once in rootCmd.PersistentPreRunE
- **No circular imports:** `pkg/` never imports `cmd/`; clean dependency tree
- **Cross-process safety:** All file ops use `store` with FileLocker (platform-specific)
- **JSON mutations atomic:** Always wrapped in `store.UpdateFile()` with exclusive lock
- **Agent dispatch:** Two entry points only: build phase → agent.Pool.Spawn(), continue → agent.Pool.Spawn()

## Anti-Patterns to Avoid

### State Mutation Without Lock

**What:** Code reads state.json, modifies in memory, writes without atomic operation
**Why wrong:** Race condition if two commands run concurrently; corruption on crash mid-write
**Do this:** Use `store.UpdateFile(path, mutator)` — `pkg/storage/storage.go:60`

### Direct os.ReadFile/WriteFile for JSON

**What:** Code uses `os.ReadFile()` / `os.WriteFile()` directly on JSON
**Why wrong:** No atomic guarantee, no validation, lost versions
**Do this:** Use `store.LoadJSON()` / `store.SaveJSON()` — `pkg/storage/storage.go`

### Event Trigger Without Caste Filter

**What:** Agent listens to event topic without checking caste or phase
**Why wrong:** Agent may execute in wrong context; tasks interfere
**Do this:** Include Trigger.Filter or caste check in Execute() — `pkg/agent/agent.go:30`

### Learning Without Provenance

**What:** Code adds learning entry without full Evidence struct
**Why wrong:** Can't trace source; can't cross-validate across colonies
**Do this:** Always include Evidence with RunID, Phase, Workers — `pkg/learn/learn.go:45`

## Error Handling

**Strategy:** Commands return error to Cobra; CLI formats and exits

**Patterns:**
- Command RunE returns `error`
- Root command catches via Cobra
- Recoverable errors include recovery command in output
- Fatal errors exit with code 1
- Build failures logged to midden for learning

## Cross-Cutting Concerns

**Logging:** No structured logs in Go layer; output only via `pkg/terminal`
**Validation:** Phase/task checks in command handlers before state mutation
**Authentication:** None in Go binary; TS wrappers handle platform auth
**Observability:** Tracer records command invocations to `.aether/data/trace.jsonl`

---

*Architecture analysis: 2026-08-01*
