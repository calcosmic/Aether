# Technology Stack Additions: v1.17 Classic Restoration

**Project:** Aether TS Host Ceremony & Swarm Dashboard
**Researched:** 2026-05-13
**Confidence:** HIGH (verified via npm registry + official GitHub sources)

## Executive Summary

The v1.17 milestone restores Classic Aether's living ceremony (banners, ASCII art, spawn notifications, seal rituals) and animated swarm dashboard (per-ant progress, chamber activity map, live refresh) to the TypeScript orchestration host. The TS host already calls Go manifests/finalizers via a subprocess bridge. The new work adds:

1. **Ceremony narrator package** — renders banners, caste identity, stage separators, and ritual text from Go-emitted events
2. **Animated swarm dashboard** — subscribes to the Go event bus (JSONL file), renders live worker activity with spinners and progress bars
3. **Event streaming bridge** — watches the Go JSONL event file and pushes parsed events into the TS host

The stack must stay lean: the existing TS host has zero runtime dependencies. We add only what is strictly necessary for terminal rendering and file watching, avoiding heavy frameworks like React-Ink or Blessed.

## Recommended Stack Additions

### Core Rendering

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| `chalk` | 5.6.2 | ANSI color/style for caste labels, banners, status text | Zero dependencies, chainable API, auto-detects color support, used by 115k+ packages. ESM-native matches our `type: "module"`. |
| `boxen` | 8.0.1 | Framed boxes around ceremony banners, Queen pronouncements, seal rituals | 9 border styles, title support, padding/margin control, integrates with chalk for colored borders. Lightweight (~24 KB). |
| `figlet` | 1.11.0 | ASCII art banners ("CROWNED ANTHILL", "SEALED CHAMBERS") | Full FIGfont spec, sync API (`textSync`), 300+ built-in fonts, TypeScript types available (`@types/figlet` 1.7.0). |
| `ora` | 9.4.0 | Per-worker spinners in swarm dashboard | 100+ spinner styles via `cli-spinners`, state methods (`start`/`succeed`/`fail`), promise helper, TTY-aware auto-disable. |
| `cli-progress` | 3.12.0 | Multi-bar progress display for wave completion, per-ant tool counters | `MultiBar` container supports concurrent independent bars, custom formatters, payload tokens, FPS limiter. No heavy deps. |
| `log-update` | 8.0.0 | Overwrite previous terminal output for live dashboard refresh | Partial redraws reduce flicker; `done()`/`clear()` lifecycle. Used by `ora` and `listr2` internally. |

### File Watching

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| `chokidar` | 5.0.0 | Watch Go's `event-bus.jsonl` for live event streaming to TS host | De-facto standard (30M+ repos). Normalizes `fs.watch` events, macOS filename support, atomic write handling, depth limiting. v5 is ESM-only, Node >=20. |

### Supporting Utilities

| Technology | Version | Purpose | When to Use |
|------------|---------|---------|-------------|
| `strip-ansi` | 7.2.0 | Remove ANSI codes when computing string width for layout | Any width calculation on styled strings (boxen, progress bars, dashboard columns). |
| `wrap-ansi` | 10.0.0 | Wrap styled text to terminal width without breaking ANSI sequences | Long ceremony messages, Queen pronouncements that must fit terminal width. |
| `ansi-escapes` | 7.3.0 | Low-level cursor positioning, screen clearing, scroll regions | If we need custom dashboard layouts beyond what log-update provides. |

### Dev Dependencies

| Technology | Version | Purpose |
|------------|---------|---------|
| `@types/figlet` | 1.7.0 | TypeScript types for figlet (figlet itself is JS, not TS) |
| `@types/cli-progress` | 3.11.6 | TypeScript types for cli-progress |

## What NOT to Add

| Technology | Why Avoid | What to Use Instead |
|------------|-----------|---------------------|
| `ink` (7.0.3) | Requires React 19.2+ peer dependency (~25 deps, ~538 KB). Adds a full reconciler, Yoga layout engine, and React runtime to a host that currently has zero runtime deps. Overkill for ceremony banners + progress bars. | `chalk` + `boxen` + `log-update` for rendering; plain TS classes for state management. |
| `blessed` (0.1.81) | Unmaintained (last release 2018), buggy on modern terminals, no ESM support, heavy widget framework. | `log-update` + `ansi-escapes` for cursor control; custom dashboard layout in plain TS. |
| `listr2` (10.2.1) | Task-list framework with its own rendering engine. Node >=22.13.0 requirement exceeds our Node >=18 engine. Designed for sequential task UIs, not live multi-worker dashboards. | `ora` for individual spinners + `cli-progress` for bars + custom orchestration. |
| `react` / `react-reconciler` | Only needed if we chose Ink. No other terminal library needs React. | Not applicable — avoid Ink. |
| `ws` (WebSocket library) | Go event bus uses JSONL file persistence, not WebSockets. No network streaming needed. | `chokidar` watching JSONL file + `fs.createReadStream` for tailing. |
| `blessed-contrib` | Dashboard widgets for blessed. Blessed itself is deprecated; contrib is doubly so. | Custom dashboard components using `cli-progress`, `ora`, and `log-update`. |
| `terminal-kit` | Full terminal framework with input handling, mouse support, screen buffers. Too heavy for our use case. | `chalk` + `boxen` + `log-update` for output only. |

## Installation

```bash
# Runtime dependencies
cd .aether/ts-host
npm install chalk@5.6.2 boxen@8.0.1 figlet@1.11.0 ora@9.4.0 cli-progress@3.12.0 log-update@8.0.0 chokidar@5.0.0 strip-ansi@7.2.0 wrap-ansi@10.0.0 ansi-escapes@7.3.0

# Dev dependencies
npm install -D @types/figlet@1.7.0 @types/cli-progress@3.11.6
```

## Integration Points with Existing TS Host

### 1. Event Streaming Bridge (`src/event-bridge.ts`)

Go persists events to `.aether/data/event-bus.jsonl` (configurable via `events.Config.JSONLFile`). The TS host must:

1. **Initial replay**: Call `aether event-bus replay --topic ceremony.build.* --since <session_start>` to get all ceremony events from the current session.
2. **Live tail**: Use `chokidar` to watch `event-bus.jsonl` for changes, then read new lines via `fs.createReadStream` with a seek offset.
3. **Parse**: Each line is a JSON object matching the `Event` struct (fields: `id`, `topic`, `payload`, `source`, `timestamp`, `ttl_days`, `expires_at`).
4. **Dispatch**: Emit parsed events to an internal `EventEmitter` or typed callback registry.

```typescript
// Conceptual integration
import { watch } from "chokidar";
import { createReadStream } from "node:fs";
import { readFile } from "node:fs/promises";

interface CeremonyEvent {
  id: string;
  topic: string;
  payload: CeremonyPayload; // from pkg/events/ceremony.go
  source: string;
  timestamp: string;
}

// 1. Replay existing events
const replayed = await callGoJSON<CeremonyEvent[]>(bridge, [
  "event-bus", "replay",
  "--topic", "ceremony.build.*",
  "--since", sessionStartISO,
]);

// 2. Watch for new events
const watcher = watch(jsonlPath, { persistent: false });
watcher.on("change", async () => {
  const newLines = await tailNewLines(jsonlPath, lastOffset);
  for (const line of newLines) {
    const evt = JSON.parse(line) as CeremonyEvent;
    eventEmitter.emit(evt.topic, evt);
  }
});
```

**Why chokidar + manual tail instead of a streaming library?**
- The Go bus appends to JSONL atomically (`storage.Store.AppendJSONL`), so `chokidar` `change` events fire on each append.
- A simple offset tracker + `fs.createReadStream({ start: offset })` is sufficient and avoids adding a `tail` or `jsonlines` dependency.
- `jsonlines` parser adds ~15 KB but provides little value over `line.split("\n").map(JSON.parse)` for well-formed Go output.

### 2. Ceremony Narrator Package (`src/ceremony/`)

Consumes ceremony events and renders them using `chalk`, `boxen`, and `figlet`.

```
src/ceremony/
  narrator.ts      -- Main renderer: maps event topics to render functions
  banners.ts       -- figlet banners for milestone names, seal rituals
  caste-render.ts  -- chalk-colored caste identity lines (emoji + label + name)
  stage-markers.ts -- "── Stage Name ──" separators
  boxes.ts         -- boxen wrappers for Queen pronouncements, warnings
```

**Integration with existing types:**
- Uses `CeremonyPayload` fields from `pkg/events/ceremony.go` (already typed in `src/types.ts` as `BuildDispatch` / `WorkerResult` — extend with a `CeremonyPayload` interface).
- Reads caste emoji/color/label maps from the shared YAML ceremony config (CEREMONY-02), not hardcoded.

### 3. Swarm Dashboard (`src/dashboard/`)

Live terminal dashboard showing active workers, wave progress, and chamber activity.

```
src/dashboard/
  dashboard.ts     -- Orchestrates render loop, owns log-update instance
  worker-row.ts    -- Renders a single worker: ora spinner + cli-progress bar + status
  wave-panel.ts    -- Groups worker rows by wave, shows wave-level progress
  chamber-map.ts   -- Shows which project areas have active workers
  renderer.ts      -- log-update integration: compose all panels into single output string
```

**Integration with worker-dispatch.ts:**
- `dispatchWorkers` currently writes to `process.stderr` directly. Replace with event emission:
  - Before dispatch: emit `ceremony.build.spawn` event (or call narrator directly).
  - During dispatch: update dashboard worker row (spinner active).
  - After dispatch: update worker row status (`completed`/`failed`), increment progress bar.
- The dashboard receives events both from the local dispatch loop and from the event bridge (for workers dispatched by other processes or Go directly).

### 4. ANSI Color/Style Management

**Caste color mapping:**
- Go runtime defines `casteColorMap` in `cmd/codex_visuals.go` (ANSI codes).
- TS host reads the shared YAML ceremony config (CEREMONY-02) which exports the same map as hex/named colors.
- `chalk` converts named colors or hex to ANSI: `chalk.hex("#FFD700")` or `chalk.yellow`.

**Why chalk over manual ANSI escape codes:**
- Auto-detects color support (disables in CI, non-TTY).
- Chainable API: `chalk.bold.yellow.bgBlue`.
- No dependencies, ~44 KB, battle-tested.

## Engine Compatibility

| Package | Node Engine | Our Host (>=18) | Status |
|---------|-------------|-----------------|--------|
| chalk 5.6.2 | no engine specified | OK | Compatible |
| boxen 8.0.1 | >=18 | OK | Compatible |
| figlet 1.11.0 | >=17 | OK | Compatible |
| ora 9.4.0 | >=20 | OK | Compatible (dev/test on 18 may need --no-engine-strict) |
| cli-progress 3.12.0 | >=4 | OK | Compatible |
| log-update 8.0.0 | >=22 | **Caution** | May need Node >=20 in practice; test on 18 |
| chokidar 5.0.0 | >=20.19.0 | **Caution** | v5 requires Node >=20; if host must run on 18, use chokidar 4.0.3 |
| strip-ansi 7.2.0 | no engine | OK | Compatible |
| wrap-ansi 10.0.0 | no engine | OK | Compatible |
| ansi-escapes 7.3.0 | no engine | OK | Compatible |

**Decision on Node >=18 vs >=20:**
- The TS host `package.json` specifies `"node": ">=18"`.
- `chokidar` v5 requires Node >=20.19.0. If we must support Node 18, downgrade to `chokidar@4.0.3` (still ESM, 1 dependency, Node >=14).
- `log-update` v8 requires Node >=22. If Node 18 support is required, downgrade to `log-update@5.0.1` (still works, fewer features).
- **Recommendation:** Bump TS host engine to `>=20` since Node 18 reaches End-of-Life in April 2025 (already past). All team/dev machines run Node 25+.

## Alternatives Considered

### Terminal Rendering Framework: Ink vs Custom

| Criterion | Ink (React) | Custom (chalk+boxen+ora) |
|-----------|-------------|--------------------------|
| Dependencies | 25+ (React, reconciler, Yoga, ws, etc.) | 0-5 (chalk has 0, boxen has ~5, ora has ~8) |
| Bundle size | ~538 KB | ~150 KB total |
| Learning curve | React knowledge required | Plain TS/Node |
| Animation | `useAnimation` hook | `ora` spinners + `log-update` refresh loop |
| Layout | Flexbox via Yoga | Manual string composition + `wrap-ansi` |
| Maintenance risk | Tied to React release cycle | Decoupled, smaller surface area |
| Fit for Aether | Overkill for banners + bars | Purpose-built for ceremony + dashboard |

**Verdict:** Custom stack. Ink is excellent for complex interactive CLIs (like Claude Code itself), but Aether's ceremony needs are primarily output rendering, not interactive input handling. The dependency weight and React peer dependency are unjustified.

### File Watching: chokidar vs Node fs.watch

| Criterion | chokidar | Node fs.watch |
|-----------|----------|---------------|
| Cross-platform | Normalizes macOS/Linux/Windows quirks | Platform-specific behavior |
| Event quality | `add`/`change`/`unlink` with filenames | Often emits `rename` with no context |
| Atomic writes | `awaitWriteFinish` option | Not supported |
| Dependencies | 1 (v4/v5) | 0 |
| Size | ~82 KB | 0 |

**Verdict:** chokidar. The normalized events and atomic-write handling are worth the 82 KB for reliable event streaming from Go's JSONL appends.

### Progress Bars: cli-progress vs custom

| Criterion | cli-progress | Custom ANSI |
|-----------|--------------|-------------|
| Multi-bar | Native `MultiBar` | Manual cursor positioning |
| Formatting | Placeholder tokens + custom formatters | Manual string building |
| FPS limiting | Built-in | Manual |
| ETA calculation | Built-in | Manual |

**Verdict:** cli-progress. The `MultiBar` container is exactly what the swarm dashboard needs for concurrent per-worker bars. Re-implementing cursor math and ETA is error-prone.

## Sources

- npm registry verified versions (2026-05-13):
  - `chalk@5.6.2`, `boxen@8.0.1`, `figlet@1.11.0`, `ora@9.4.0`, `cli-progress@3.12.0`, `log-update@8.0.0`, `chokidar@5.0.0`
- GitHub repositories (official):
  - https://github.com/chalk/chalk
  - https://github.com/sindresorhus/boxen
  - https://github.com/sindresorhus/ora
  - https://github.com/sindresorhus/log-update
  - https://github.com/npkgz/cli-progress
  - https://github.com/paulmillr/chokidar
  - https://github.com/patorjk/figlet.js
- Aether codebase (existing types and event bus):
  - `.aether/ts-host/src/types.ts` — Go manifest/completion types
  - `.aether/ts-host/src/go-bridge.ts` — Subprocess bridge with JSON envelope
  - `pkg/events/ceremony.go` — Ceremony event topics and payload shape
  - `pkg/events/bus.go` — JSONL persistence and pub/sub mechanics
