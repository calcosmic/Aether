# Phase 115: Swarm Dashboard - Context

**Gathered:** 2026-05-13
**Status:** Ready for planning

## Phase Boundary

Build a live terminal dashboard that shows all active workers, their progress, tool usage, and chamber activity map. The dashboard consumes ceremony events from the event bridge (built in Phase 112) and renders a structured UI using `log-update`, `ora`, `cli-progress`, and `boxen`.

## Implementation Decisions

### D-01: Dashboard Suppresses Narrator stdout
- **Decision:** When the dashboard is active, the narrator's stdout writes are suppressed. Events still flow to both modules, but the dashboard owns the terminal surface.
- **Why:** Two modules writing to stdout simultaneously causes visual corruption. The dashboard renders the complete frame atomically via `log-update`.

### D-02: Chamber Map Groups by File Directory
- **Decision:** The chamber activity map groups workers by the directory prefix of their claimed files (`files_created`, `files_modified`).
- **Why:** Shows which project areas have active work. No semantic analysis needed — just path prefix grouping.

### D-03: Progress Proxy via Tool Count
- **Decision:** Worker progress percentage is estimated from `tool_count` relative to a heuristic maximum (20 tools = 100%).
- **Why:** Workers don't report real-time progress. Tool count is a reasonable proxy for "how much work has been done."

### D-04: Dashboard Defaults to On in TTY
- **Decision:** Dashboard is active by default when `process.stdout.isTTY` is true and `--no-dashboard` is not passed.
- **Why:** The dashboard is the primary user-facing output for the build command. Non-TTY falls back to markdown/json mode.

## Canonical References

- `.planning/phases/115-swarm-dashboard/115-RESEARCH.md` — Library inventory, ASCII mockup, architecture
- `.planning/phases/113-ceremony-narrator/113-02-SUMMARY.md` — Narrator event dispatch
- `.planning/phases/114-real-worker-dispatch/114-02-SUMMARY.md` — Wave orchestrator and lifecycle
- `.aether/ts-host/src/event-bridge.ts` — Event consumption API
- `.aether/ts-host/src/narrator.ts` — Event-to-render dispatch
- `.aether/ts-host/src/types.ts` — CeremonyEvent, CeremonyPayload types
- `.aether/ts-host/src/lifecycle.ts` — Lifecycle orchestrator
- `cmd/codex_build_progress.go` — Go progress bar reference

## Existing Code Insights

### Reusable Assets
- **event-bridge.ts** — already consumes ceremony events. Dashboard subscribes alongside narrator.
- **narrator.ts** — can be told to suppress stdout writes when dashboard is active.
- **caste-config.ts** — emoji, color, label maps for dashboard worker identity.
- **Package.json** — `ora`, `cli-progress`, `log-update`, `boxen`, `chalk` all present.

### Integration Points
- **lifecycle.ts** — will start/stop dashboard around worker dispatch.
- **index.ts** — will pass `--no-dashboard` flag to lifecycle options.
- **Go `codex_build_progress.go`** — Go still emits progress events; dashboard renders them.
