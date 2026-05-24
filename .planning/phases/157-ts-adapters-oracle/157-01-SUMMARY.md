---
phase: 157-ts-adapters-oracle
plan: 01
subsystem: control-ts
milestone: v1.24 Hybrid Architecture Salvage
status: completed
completed_date: "2026-05-24"
tags: [typescript, adapters, stubs, interfaces, registry]
dependency_graph:
  requires: [156-ts-control-plane-core]
  provides: [158-ts-event-stream, 159-ts-end-to-end]
  affects: [control-ts/src/index.ts]
tech_stack:
  added: [TypeScript adapter stubs]
  patterns: [PlatformAdapter interface, factory registry, barrel exports]
key_files:
  created:
    - control-ts/src/adapters/types.ts
    - control-ts/src/adapters/claude.ts
    - control-ts/src/adapters/codex.ts
    - control-ts/src/adapters/opencode.ts
    - control-ts/src/adapters/mcp.ts
    - control-ts/src/adapters/index.ts
    - control-ts/tests/adapters/adapters.test.ts
  modified:
    - control-ts/src/index.ts
decisions:
  - "Mirrored Go runtime dispatch shapes in TypeScript for future bridge compatibility"
  - "Stubs resolve immediately (0ms delay) with deterministic mock results — no real API calls"
  - "Registry uses string-keyed class map for simple factory pattern"
metrics:
  duration: "~8 minutes"
  tasks_completed: 1
  files_created: 7
  files_modified: 1
  tests_added: 10
  tests_passing: 10
---

# Phase 157 Plan 01: Platform Adapter Stubs Summary

**One-liner:** TypeScript adapter stubs for Claude Code, Codex, OpenCode, and MCP with shared interfaces, factory registry, and 10 passing tests.

## What Changed

Created the `control-ts/src/adapters/` directory with:
- **Shared types** (`types.ts`) — `Platform`, `AvailabilityStatus`, `WorkerConfig`, `WorkerResult`, `PlatformAdapter`, and related shapes that mirror the Go runtime dispatch patterns.
- **Four adapter stubs** — `ClaudeAdapter`, `CodexAdapter`, `OpenCodeAdapter`, `McpAdapter`. Each implements `PlatformAdapter` with `dispatch()`, `preflight()`, and `health()` methods. Stubs return deterministic mock results (status `"completed"`, echoed `taskID`/`workerName`/`caste`) and never make real subprocess or API calls.
- **Registry** (`index.ts`) — `createAdapter(platform)` factory and `getAvailableAdapters()` list. Re-exports all types.
- **Public API wiring** (`src/index.ts`) — Added exports for `createAdapter`, `getAvailableAdapters`, and adapter type symbols.
- **Tests** (`tests/adapters/adapters.test.ts`) — 10 tests covering dispatch, preflight, health, interface compliance, factory behavior, and error handling.

## Verification Results

- **Adapter tests:** 10/10 passing (`npm test -- tests/adapters/adapters.test.ts`)
- **TypeScript typecheck:** zero errors (`npm run typecheck`)
- **Manual registry verification:** `getAvailableAdapters()` returns `["claude","codex","opencode","mcp"]`; `createAdapter("claude").platform === "claude"`

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

All adapters are stubs by design for this phase. They echo config fields and return empty arrays / zero counts. Real platform dispatch will be implemented in a future plan when the Go-to-TS bridge is built.

| Stub | File | Reason |
|------|------|--------|
| dispatch mock result | all adapter .ts files | No real API calls in Phase 157 |
| preflight always available | all adapter .ts files | No binary/auth probing yet |
| health always healthy | all adapter .ts files | No health check logic yet |

## Threat Flags

No new security-relevant surface introduced. Stubs do not perform network I/O, subprocess execution, or credential handling.

## Self-Check: PASSED

- [x] All created files exist on disk
- [x] Commit `30ccfe5a` exists in git log
- [x] Tests pass and typecheck is clean
- [x] No modifications to shared orchestrator artifacts (STATE.md, ROADMAP.md)
