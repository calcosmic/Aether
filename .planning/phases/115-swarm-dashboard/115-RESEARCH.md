# Phase 115: Swarm Dashboard - Research

**Researched:** 2026-05-13
**Domain:** Terminal UI dashboard for live worker monitoring
**Confidence:** HIGH

## Summary

Phase 115 builds a live terminal dashboard that shows all active workers, their progress, tool usage, and chamber activity map. The TS host already consumes Go ceremony events via the event bridge (Phase 112) and renders them via the narrator (Phase 113). The dashboard augments/replaces simple stdout rendering with a structured live UI.

**Dependencies already in place:**
- `event-bridge.ts` — streams ceremony events from Go JSONL
- `narrator.ts` — dispatches events to renderers
- `chalk`, `ora`, `cli-progress`, `log-update` — all in package.json
- `pkg/events/ceremony.go` — Go event schema with all needed fields

## Key Findings

### Existing Terminal UI Libraries
| Library | Version | In package.json? | Purpose |
|---------|---------|-------------------|---------|
| `ora` | 9.4.0 | Yes | Animated spinners per worker |
| `cli-progress` | 3.12.0 | Yes | Progress bars for wave completion |
| `log-update` | 8.0.0 | Yes | In-place terminal updates (dashboard refresh) |
| `chalk` | 5.6.2 | Yes | ANSI colors for caste identity |
| `boxen` | 8.0.1 | Yes | Framed panels for dashboard sections |

### Ceremony Events Available
From `types.ts` CEREMONY_TOPICS and CeremonyPayload:
- `ceremony.build.wave.start` — wave number, phase
- `ceremony.build.spawn` — caste, name, task, spawn_id
- `ceremony.build.tool_use` — tool_count, token_count
- `ceremony.build.wave.end` — completed, total
- `ceremony.build.circuit_break` — blockers, status

### Dashboard Architecture
The dashboard runs as a separate module that:
1. Subscribes to the event bridge (same as narrator)
2. Maintains an in-memory model of active workers
3. Renders the full dashboard on every significant event
4. Uses `log-update` to replace the previous frame in-place

### Go Dashboard Reference
`cmd/codex_build_progress.go` contains Go's progress bar implementation:
- `emitVisualProgress` writes progress updates
- Uses ANSI escape sequences for in-place updates
- Shows worker counts, completed/total, and stage name

The TS host dashboard should be richer than Go's simple progress bar.

## Recommended Implementation

### Dashboard Layout (ASCII mockup)
```
┌─ Swarm Dashboard ──────────────────────────────────────────────┐
│ Wave 1 of 3  │  Builders: 3 active  │  Watchers: 1 queued       │
├────────────────────────────────────────────────────────────────┤
│ 🔨 Builder Mason-67    ●●●●○○○○  50%  Tools: 12  Tokens: 4.2k   │
│ 🔍 Scout Ranger-12     ●●●○○○○○  37%  Tools: 8   Tokens: 2.1k    │
│ 👁️ Watcher Hawk-03     ●○○○○○○○  12%  Tools: 3   Tokens: 0.8k    │
├────────────────────────────────────────────────────────────────┤
│ Chamber Activity                                               │
│  src/renderers/     ████████░░  80%  (2 workers)               │
│  src/narrator.ts    ██████░░░░  60%  (1 worker)                │
│  test/              ████░░░░░░  40%  (1 worker)                │
├────────────────────────────────────────────────────────────────┤
│ Elapsed: 02:34  │  Est. remaining: 01:45  │  Next: Wave 2      │
└────────────────────────────────────────────────────────────────┘
```

### Files to Create
- `dashboard.ts` — Main dashboard controller, event handler, render loop
- `worker-widget.ts` — Per-worker spinner, progress, stats
- `chamber-map.ts` — Project area activity visualization
- `dashboard-renderer.ts` — Frame assembly using log-update + boxen

### Files to Modify
- `narrator.ts` — Option to suppress stdout when dashboard is active
- `lifecycle.ts` — Start/stop dashboard around worker dispatch
- `index.ts` — `--dashboard` flag (default: true when TTY detected)

## Open Questions

1. **Should the dashboard and narrator coexist?**
   - Recommendation: Dashboard suppresses narrator's stdout writes when active. Events still flow to both, but dashboard owns the terminal.

2. **How to detect "project areas" for the chamber map?**
   - Recommendation: Group workers by the directory prefix of their claimed files (files_created/files_modified). Extract from ceremony events.

3. **Progress percentage for workers?**
   - Workers don't report real-time progress. Recommendation: Use tool_count as a proxy (more tools = more progress). Animate spinner for active, show checkmark for completed.

## Sources
- `.aether/ts-host/src/event-bridge.ts`
- `.aether/ts-host/src/narrator.ts`
- `.aether/ts-host/src/types.ts`
- `cmd/codex_build_progress.go`
- `pkg/events/ceremony.go`
