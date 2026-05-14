# Phase 112: Foundation - Context

**Gathered:** 2026-05-13
**Status:** Ready for planning

## Phase Boundary

Build the event bridge from Go runtime to TypeScript host, establish shared ceremony config, bump Node engine, and enforce the boundary contract. This is the foundation everything downstream depends on — without it, Phase 113 (Ceremony Narrator) has no events to render and Phase 114 (Real Worker Dispatch) has no orchestration backbone.

## Implementation Decisions

### Event Delivery Mechanism
- **D-01:** Use **JSONL tail** for event streaming from Go to TS host. Go already writes JSONL via `pkg/events/`. The TS host replays existing events via `aether event-bus-replay`, then watches the JSONL file with `chokidar` and tails new lines. This requires no background server, respects the boundary contract, and works within the existing wrapper→Go→TS call chain.
- **D-02:** WebSocket via `aether serve` is a future enhancement (Phase 118+). Not needed for v1.17. Subprocess narrator pipe is rejected — it tightly couples Go and TS lifecycles and complicates error recovery.

### Ceremony Config Format
- **D-03:** Use **YAML** for shared ceremony config. The point of v1.17 is making ceremony editable by humans. Adding `js-yaml` (~100KB) is a tiny cost for human-readable config. The config lives at `.aether/config/ceremony.yaml`.
- **D-04:** Config includes: caste emoji map, caste ANSI color map, caste label map, stage separator style, naming convention, banner templates (by command), and excavation status phrases for swarm display.

### Node Engine
- **D-05:** Bump Node engine to **>=20**. Node 18 is End-of-Life. This unlocks `chokidar` v5 and `log-update` v8 which are needed for the swarm dashboard. The TS host is a dev tool, not end-user runtime — requiring Node >=20 is reasonable.

### Boundary Enforcement
- **D-06:** The TS host must never write to `.aether/data/`. All state mutations go through Go finalizers. This is already enforced by `boundary-reference.ts` and `boundary_contract_test.go`. The event bridge only reads from JSONL, never writes.

### Claude's Discretion
- The exact structure of the YAML ceremony config is left to the implementer — it should be intuitive and match the existing caste identity patterns in `cmd/codex_visuals.go`.
- The TypeScript event type definitions can mirror the Go `CeremonyPayload` struct from `pkg/events/ceremony.go`.

## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Architecture & Contracts
- `.planning/research/ARCHITECTURE.md` — Event bridge design, data flow diagrams, component responsibilities
- `.planning/research/STACK.md` — Recommended TS libraries (chalk, boxen, figlet, ora, cli-progress, chokidar, js-yaml)
- `.aether/references/contracts/runtime-boundary-contract.md` — Go/TS ownership boundaries (if exists)
- `.planning/REQUIREMENTS.md` — Phase 112 requirements (TS-01 through TS-06, CER-02)
- `.planning/ROADMAP.md` — Phase 112 goal and success criteria

### Existing Code
- `.aether/ts-host/src/` — Existing TS host scaffold (types.ts, go-bridge.ts, host.ts, lifecycle.ts, worker-dispatch.ts, boundary-reference.ts)
- `pkg/events/ceremony.go` — Go ceremony event topics and payload definitions
- `cmd/ceremony_emitter.go` — Go ceremony emitter that publishes to event bus
- `cmd/codex_visuals.go` — Current Go rendering logic (to be replaced by event emission)

## Existing Code Insights

### Reusable Assets
- **TS host scaffold** — `go-bridge.ts` already calls Go CLI with JSON output mode and parses the envelope. The event bridge extends this pattern.
- **`boundary-reference.ts`** — Already lists Go-owned paths. Event bridge must respect these.
- **Event bus in Go** — `pkg/events/` already publishes ceremony topics. No new Go code needed for basic event emission.

### Established Patterns
- **Go emits events, something else renders** — The ceremony emitter already has a dual path (event bus + optional narrator subprocess). The TS host becomes the "something else."
- **JSON-mediated contract** — `--plan-only` / finalizer pattern proven in v1.16. Event streaming follows the same philosophy: Go produces structured data, consumer renders.

### Integration Points
- **TS host `lifecycle.ts`** — Will need to start the event bridge before orchestrating workers and stop it after.
- **Wrapper `build.md`** — Will call `aether ceremony spawn-plan` for display (Go renders complex moments), but lightweight real-time updates come from the TS host via events.
- **Go `ceremony_emitter.go`** — Already publishes events. May need minor changes to ensure all relevant moments emit events (currently some rendering happens directly in `codex_visuals.go` without event emission).

## Specific Ideas

- The shared YAML config should be editable without rebuilding anything. A user should be able to change the Builder emoji from 🔨 to 🛠️ and see it immediately on the next `/ant-build`.
- Event payloads should include enough context for the TS host to render without needing additional Go calls. For example, a `ceremony.build.spawn` event should include the caste, worker name, task description, and wave number.

## Deferred Ideas

- **WebSocket event streaming** — More responsive than JSONL tail. Belongs in v1.18+ if swarm dashboard latency is unacceptable.
- **Real-time web dashboard** — Out of scope per PROJECT.md.
- **Compiled ceremony config validation** — Schema validation for `.aether/config/ceremony.yaml` to catch typos early. Nice to have, not critical for v1.17.

---

*Phase: 112-Foundation*
*Context gathered: 2026-05-13*
