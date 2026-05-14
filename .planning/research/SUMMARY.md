# Project Research Summary

**Project:** Aether v1.17 Classic Restoration
**Domain:** CLI colony framework — hybrid Go runtime + TypeScript orchestration host
**Researched:** 2026-05-13
**Confidence:** HIGH

## Executive Summary

Aether v1.17 restores the living ceremony, animated swarm dashboard, and intelligent Queen orchestration that defined Classic v5.4, but within the modern hybrid architecture: Go runtime owns state and emits structured events; the TypeScript host consumes those events and dispatches platform workers; wrapper markdown renders the user-facing experience. This is not a rewrite — it is a restoration. The recommended approach is to build the event bridge first, then layer ceremony rendering and swarm display on top, then wire real worker dispatch and workflow patterns. The biggest risks are state corruption (if TS host writes directly to `.aether/data/`), ceremony drift across platforms, and scope creep disguised as improvement. Mitigation is strict boundary enforcement, shared YAML ceremony config, and golden parity tests locked to the v5.4 baseline.

## Key Findings

### Recommended Stack

The TS host currently has zero runtime dependencies. v1.17 adds a lean, purpose-built terminal rendering stack — no heavy frameworks like Ink or Blessed. The core additions are `chalk` + `boxen` + `figlet` for ceremony banners and boxes, `ora` + `cli-progress` for animated spinners and progress bars, `log-update` for live dashboard refresh, and `chokidar` for watching Go's JSONL event file. All packages are ESM-native and compatible with Node >=20 (the TS host engine should bump from >=18 to >=20, since Node 18 is already EOL). Dev dependencies are only `@types/figlet` and `@types/cli-progress`. See `STACK.md` for full version table and rejected alternatives.

**Core technologies:**
- `chalk@5.6.2`: ANSI color/style — zero deps, auto-detects color support, chainable API
- `boxen@8.0.1`: Framed boxes for banners — 9 border styles, lightweight (~24 KB)
- `figlet@1.11.0`: ASCII art banners — sync API, 300+ fonts, types via `@types/figlet`
- `ora@9.4.0`: Per-worker spinners — 100+ styles, TTY-aware auto-disable
- `cli-progress@3.12.0`: Multi-bar progress — `MultiBar` container for concurrent workers
- `log-update@8.0.0`: Live terminal refresh — partial redraws, `done()`/`clear()` lifecycle
- `chokidar@5.0.0`: Watch Go JSONL event file — de-facto standard, atomic-write handling

### Expected Features

**Must have (table stakes):**
- Event bridge (Go JSONL -> TS host) — infrastructure for everything else
- Real worker dispatch (replace 100ms simulation) — the host must actually spawn agents
- Parallel wave execution — concurrent workers per wave
- Error recovery / retry / timeout — graceful fallback when workers fail

**Should have (differentiators):**
- Ceremony narrator — ASCII banners, caste identity, stage separators, spawn notifications
- Animated swarm dashboard — per-ant spinners, progress bars, chamber activity map, live refresh
- Builder-Probe Lock — builders return `code_written`, Probe upgrades to `completed`
- Workflow pattern selection — Queen picks pattern based on phase name
- Tiered escalation — retry -> reassignment -> Queen -> user

**Defer (v1.18+):**
- WebSocket/SSE server for events — JSONL tail is sufficient now
- Oracle phase-aware prompts / diminishing returns — Go-side, not blocking ceremony restoration
- Real-time web dashboard — out of scope per PROJECT.md
- Cross-colony ledger sharing — out of scope per PROJECT.md

### Architecture Approach

The architecture is a three-layer pipeline: **Go emits events** (from `pkg/events/` and `cmd/ceremony_emitter.go`), the **TS host subscribes and orchestrates dispatch** (`.aether/ts-host/`), and **wrappers render ceremony** (`.claude/commands/ant/build.md`). The TS host consumes events via a hybrid of JSONL tail (for startup replay) and subprocess narrator pipe (for live streaming). This requires no background server, respects the boundary contract, and works within the existing wrapper->Go->TS call chain. The TS host never writes to `.aether/data/`; it calls Go finalizers for all state commits.

**Major components:**
1. **Event Bridge** (`src/event-bridge.ts`) — Consumes Go events via JSONL tail or subprocess pipe; emits typed events to TS consumers
2. **Ceremony Narrator** (`src/ceremony-narrator.ts`) — Renders ANSI/plain ceremony from event stream: wave banners, worker status, progress bars
3. **Wave Dispatcher** (`src/wave-dispatcher.ts`) — Parallel execution of manifest waves with event-driven status updates
4. **Caste Config Loader** (`src/caste-config.ts`) — Loads shared YAML caste emoji/color/label map (prevents platform drift)
5. **Go Runtime** (existing, modified) — Emits `ceremony.*` events, provides manifest + finalizers, owns all state mutation

### Critical Pitfalls

1. **TS Host Writes State Directly (Frankenstein Regression)** — TS host must never write to `.aether/data/`. Prevention: ESLint rule banning `fs` imports for data paths, runtime bridge rejection, contract tests. See PITFALLS.md Pitfall 1.
2. **Duplicate Orchestration Logic in Go and TS** — Go generates the full `execution_plan`; TS host dispatches strictly from manifest. No TS-side wave logic. See PITFALLS.md Pitfall 2.
3. **Ceremony Drift Across Platforms** — Use shared YAML ceremony config consumed by Go, TS host, and wrappers. Codex fallback via Go `codex_visuals.go`. See PITFALLS.md Pitfall 3.
4. **Rewriting Instead of Restoring (Scope Creep)** — v5.4 baseline is read-only. Golden tests lock behavior. Any deviation must be explicitly approved as intentional. See PITFALLS.md Pitfall 4.
5. **Race Conditions in Hybrid Event Streaming** — Use `event-bus-subscribe --stream` (not polling). Handle duplicate event IDs idempotently. Replay from timestamp on crash recovery. See PITFALLS.md Pitfall 5.
6. **Animated Dashboard Breaks in Non-TTY** — Three output modes: `json` (machine), `visual` (TTY ANSI), `markdown` (plain text). TS host respects `AETHER_OUTPUT_MODE`. See PITFALLS.md Pitfall 6.
7. **Builder-Probe Lock Bypassed** — TS host must not translate `code_written` to `completed`. Only Go finalizer (or Probe result) upgrades status. See PITFALLS.md Pitfall 7.
8. **Worktree Merge-Back Orphans Code** — Manifest includes `requires_merge_back: true` in worktree mode. TS host calls `aether worktree-merge-back` between waves. See PITFALLS.md Pitfall 10.

## Implications for Roadmap

Based on research, suggested phase structure:

### Phase 1: Foundation — Event Bridge + Ceremony Config
**Rationale:** Everything else depends on the TS host being able to consume Go events. The shared YAML ceremony config prevents drift from day one.
**Delivers:** `event-bridge.ts`, `caste-config.ts`, shared `ceremony.yaml`, basic TypeScript types for `CeremonyEvent` / `CeremonyPayload`.
**Addresses:** Event bridge (Category 1), ceremony config foundation (Category 2).
**Avoids:** Pitfall 3 (drift), Pitfall 5 (races — uses subscription not polling).
**Research flag:** LOW — event bus already exists in Go; patterns are well-documented.

### Phase 2: Ceremony Narrator
**Rationale:** Once events flow, render them. This restores the "living" feel and provides immediate user-visible value.
**Delivers:** `ceremony-narrator.ts`, `banners.ts`, `caste-render.ts`, `stage-markers.ts`, `boxes.ts`.
**Addresses:** ASCII banners, spawn notifications, worker completion lines, build summary block (Category 2).
**Avoids:** Pitfall 6 (non-TTY breakage — implement three output modes here).
**Research flag:** LOW — rendering libraries are standard; output modes are known pattern.

### Phase 3: Real Worker Dispatch + Parallel Waves
**Rationale:** Replace simulation with actual platform agent spawning. Parallel wave execution is a prerequisite for the swarm dashboard to have meaningful data.
**Delivers:** Updated `worker-dispatch.ts` with real platform spawn, `wave-dispatcher.ts` for parallel execution, error recovery / retry / timeout.
**Addresses:** Real worker dispatch, parallel waves, error recovery (Category 1).
**Avoids:** Pitfall 2 (duplicate orchestration — dispatch strictly from manifest), Pitfall 7 (Builder-Probe Lock), Pitfall 10 (worktree merge-back).
**Research flag:** MEDIUM — platform agent spawning varies by environment; needs validation.

### Phase 4: Swarm Dashboard
**Rationale:** The animated dashboard is the visual payoff. It needs real events and real workers, so it comes after dispatch works.
**Delivers:** `dashboard.ts`, `worker-row.ts`, `wave-panel.ts`, `chamber-map.ts`, `renderer.ts`.
**Addresses:** Animated spinners, per-ant progress bars, tool counters, elapsed time, chamber activity map, live refresh (Category 3).
**Avoids:** Pitfall 6 (non-TTY), Pitfall 5 (event races).
**Research flag:** LOW — dashboard components are standard terminal UI patterns.

### Phase 5: Queen Orchestration Intelligence
**Rationale:** Workflow patterns, Builder-Probe Lock, and tiered escalation are safety and quality features. They depend on real dispatch and event flow.
**Delivers:** Workflow pattern selection, Builder-Probe Lock enforcement, tiered escalation chain, intra-build midden checks.
**Addresses:** Workflow patterns, Builder-Probe Lock, tiered escalation, phase mode awareness (Category 4).
**Avoids:** Pitfall 7 (Builder-Probe Lock), Pitfall 9 (infinite escalation loops — escalation level in completion file).
**Research flag:** MEDIUM — escalation logic has edge cases; needs careful testing.

### Phase 6: Oracle Behavioral Richness
**Rationale:** Oracle improvements are Go-side prompt/template changes. They can proceed in parallel with TS host work but are lower priority than ceremony restoration.
**Delivers:** Phase-aware prompts, diminishing returns detection, template synthesis, signal injection.
**Addresses:** Oracle RALF loop richness (Category 5).
**Avoids:** Pitfall 8 (Oracle state loss — stateless TS host loop, re-read from Go each iteration).
**Research flag:** LOW — prompt engineering, no new infrastructure.

### Phase 7: Integration + Parity Verification
**Rationale:** Wire everything together and prove parity with Classic v5.4.
**Delivers:** Updated `lifecycle.ts`, `host.ts`, `build.md` wrapper, golden parity tests, end-to-end pipeline tests.
**Addresses:** Integration (Category 1-5), verification.
**Avoids:** Pitfall 1 (state writes), Pitfall 4 (scope creep — golden tests enforce baseline), Pitfall 14 (simulated code ships as real).
**Research flag:** MEDIUM — golden tests need careful design to avoid brittleness (Pitfall 15).

### Phase Ordering Rationale

- **Foundation first:** Event bridge is the data layer for everything else. Without it, ceremony and dashboard have nothing to render.
- **Ceremony before dispatch:** Ceremony narrator can be tested with synthetic events while real dispatch is being built. This gives user-visible progress early.
- **Dispatch before dashboard:** Dashboard needs real worker status events. Simulated workers would produce a fake dashboard.
- **Orchestration after dispatch:** Workflow patterns and escalation operate on the dispatch layer. They are safety features, not prerequisites.
- **Oracle last:** Oracle richness is independent of TS host ceremony. It can proceed in parallel but is lower priority for the milestone.
- **Verification at the end:** Golden parity tests need all components stable. Running them too early creates noise from incomplete features.

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 3 (Real Worker Dispatch):** Platform agent spawning mechanics vary by environment (Claude Code Tasks, OpenCode, Codex). Needs validation of spawn-log / spawn-complete flow.
- **Phase 5 (Queen Orchestration):** Escalation chain edge cases and circuit breaker behavior. Classic v5.4 had infinite loop bugs here.
- **Phase 7 (Parity Verification):** Golden test design — how to compare hybrid output against Classic Bash output without brittle string snapshots.

Phases with standard patterns (skip research-phase):
- **Phase 1 (Event Bridge):** Well-documented in Go codebase; `event-bus-subscribe --stream` is existing CLI.
- **Phase 2 (Ceremony Narrator):** Standard terminal rendering libraries; output mode pattern already in Go.
- **Phase 4 (Swarm Dashboard):** Standard terminal UI components; no novel algorithms.
- **Phase 6 (Oracle Richness):** Prompt engineering; no new infrastructure or patterns.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Verified via npm registry + official GitHub sources. All versions confirmed ESM-native and Node >=20 compatible. Rejected alternatives well-justified. |
| Features | HIGH | Direct comparison against Classic v5.4 codebase. Feature gaps are well-documented in migration map and ceremony-revival handoff. |
| Architecture | HIGH | Based on direct source code analysis of Go event bus, TS host, and wrapper contracts. Three integration options analyzed; hybrid approach is lowest-risk. |
| Pitfalls | HIGH | Derived from runtime boundary contract, migration map, known issues, and MEMORY.md entries. Many pitfalls reference existing tests and detection mechanisms. |

**Overall confidence:** HIGH

### Gaps to Address

- **Node engine bump:** TS host currently specifies `>=18`. `chokidar@5` requires `>=20.19.0`, `log-update@8` requires `>=22`. Decision needed: bump to `>=20` or downgrade packages. Recommendation: bump to `>=20` (Node 18 EOL April 2025).
- **Worktree merge-back in manifest:** The `requires_merge_back` flag is recommended but not yet in the Go manifest schema. Needs a small Go change in Phase 3.
- **Escalation level in completion file:** Completion file schema may need `escalation_level` field added for Pitfall 9 prevention. Needs Go change.
- **Golden test baseline:** Classic v5.4 output must be captured and stored before any restoration work begins, or parity tests have no reference.
- **Codex skill update:** Any change to plan/build/continue orchestration must update 5 artifacts together (command guide, YAML, Claude wrapper, OpenCode wrapper, Codex skill). Manual process is error-prone.

## Sources

### Primary (HIGH confidence)
- `STACK.md` — npm registry verified versions, GitHub official repos, Aether codebase integration points
- `ARCHITECTURE.md` — Direct source analysis of `pkg/events/`, `cmd/ceremony_*.go`, `.aether/ts-host/src/`, wrapper markdown
- `FEATURES-V117.md` — Classic v5.4 comparison, 4 research agent synthesis, complexity assessment
- `PITFALLS.md` — Runtime boundary contract, migration map, ceremony-revival handoff, wrapper-runtime UX contract, state contract, known issues, command playbooks, MEMORY.md

### Secondary (MEDIUM confidence)
- Bazel BEP / Turborepo / GitHub CLI `gh run watch` — External tool comparison for ceremony and dashboard patterns
- LangGraph / OpenAI Agents JS / Temporal — Workflow pattern comparison

### Tertiary (LOW confidence)
- k9s / Lazygit / Docker Compose `up` — TUI/dashboard analogs, mostly for "what to avoid" validation

---
*Research completed: 2026-05-13*
*Ready for roadmap: yes*
