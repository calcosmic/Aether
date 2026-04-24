# Aether Colony — Current Context

> **This document is the colony's memory. If context collapses, read this file first.**

---

## System Status

| Field | Value |
|-------|-------|
| **Last Updated** | 2026-04-24T05:10:05Z |
| **Current Phase** | 1 |
| **Phase Name** | Assumptions and gap audit |
| **Phase Status** | ready |
| **Milestone** | First Mound |
| **Colony Status** | READY |
| **Safe to Clear?** | YES — Plan persisted, ready for the next command |

---

## Current Goal

Restore the agent spawning bridge — bring back full ceremony, real agent dispatch via Agent tool, all worker castes (Archaeologist, Oracle, Architect, Ambassador, Builder waves, Watcher, Measurer, Chaos, Scout), depth prompts, named workers with caste colors, graveyard/midden system, QUEEN.md wisdom pipeline, hive brain, skill injection, cross-agent context flow, tiered escalation, and visually rich output with emojis and caste identity. The Go runtime manages state and produces structured dispatch plans; the markdown wrappers must read those plans and spawn real workers. Everything that made v5.4 feel alive, rebuilt on top of the current Go runtime.

---

## What's In Progress

Blended v1.6 ceremony revival plan persisted: Go owns state/events, TypeScript narrator owns visuals, wrappers own real Task-tool spawning. Current phase: Assumptions and gap audit.

Detailed handoff note: `.aether/dreams/2026-04-24-ceremony-revival-v1.6-handoff.md`.

Important continuity distinction: structured colony state has not advanced past Phase 1. The working tree contains reviewed Phase 2 foundation work; advance only through the normal lifecycle.

Integrated review findings: Keeper continuity fixes; Auditor stream timeout/pagination fixes; Gatekeeper explicit `.aether/ts` embed, nested `node_modules` exclusion, CI/release/dependabot coverage, package license, and runtime-not-wired warning; Watcher phase-plan mirror and real TypeScript narrator tests.

Latest Phase 2 slices: narrator runtime now ships as dependency-free `.aether/ts/dist/narrator.js`; `npm ci` is only for developer/CI checks, not installed runtime use. The narrator can consume Go-owned `visuals-dump --json` caste metadata through `--visuals`. Event-bus stream output is now smoke-tested through the dependency-free narrator runtime. Go auto-launch and `AETHER_NARRATOR` remain deferred until the launcher slice.

Detailed tracked handoff: `.aether/docs/ceremony-revival-v1.6-handoff.md`.

Specialist reviews for the launcher slice are complete. Scout recommended build-specific lifecycle insertion points in `cmd/codex_build_worktree.go` and `cmd/codex_build.go`. Watcher listed launcher tests for env gating, missing Node/runtime, early exits, cleanup, event persistence, and JSON non-pollution. Gatekeeper required absolute `node` + `dist/narrator.js`, no shell/npm/npx/tsx at runtime, non-fatal missing dependencies, and child stdout routed back through Go's visual output mutex. Those launcher requirements are implemented and verified.

Launcher implementation completed: `cmd/narrator_launcher.go` auto-launches the dependency-free Node sidecar for visual build output, never in JSON mode; `cmd/ceremony_emitter.go` persists build ceremony events and forwards the exact persisted event to the sidecar; `cmd/codex_build.go` and `cmd/codex_build_worktree.go` emit prewave, wave start, spawn, tool-use, and wave-end events. User-controlled event text/lists are trimmed before persistence.

Rolling display foundation completed: `.aether/ts/narrator.ts` now keeps a stateful ceremony frame and renders a `COLONY ACTIVITY` view with wave progress plus Active, Completed, Blocked, and Other worker sections. It preserves the compatibility event line first so existing Go smoke tests continue to work. `.aether/ts/dist/narrator.js` has been regenerated.

Hub fallback smoke completed in a temporary HOME: install/update synced the package, fixture-local `.aether/ts` was removed, and `AETHER_NARRATOR=on aether build 1 --synthetic` still rendered `COLONY ACTIVITY` from the installed hub runtime. TTY live redraw/debounce is intentionally deferred until real wrapper output gives a better terminal contract.

Build plan-only bridge slice completed: `aether build <phase> --plan-only` now emits a read-only JSON `dispatch_manifest` for wrapper orchestration. It validates the same buildability gates as real build, includes deterministic worker names plus `agent_name`, wave execution, playbooks, selected tasks, and success criteria, and does not mutate `COLONY_STATE.json` or write build artifacts. Focused Go tests passed.

Build finalize bridge slice completed: `aether build-finalize <phase> --completion-file <path|->` now records externally spawned wrapper Task results as `dispatch_mode: external-task`, writes the build manifest and `last-build-claims.json`, updates/creates spawn-tree statuses, sets colony state to BUILT, and points the next action at `aether continue`. This is the missing runtime-owned input packet after Claude/OpenCode wrappers spawn real agents.

Build wrapper restoration completed: `.aether/commands/build.yaml`, `.claude/commands/ant/build.md`, and `.opencode/commands/ant/build.md` now use the runtime bridge: status -> JSON `build --plan-only` -> parse `dispatch_manifest` -> load `build-wave.md` -> spawn real platform agents with manifest names/castes/waves -> `spawn-log`/`spawn-complete` -> completion JSON -> `build-finalize` -> `/ant-continue`. Wrapper tests now forbid the old visual pass-through build command.

Continue plan-only bridge slice completed: `aether continue --plan-only` now runs runtime-owned verification/claim checks and emits a read-only JSON `continue_manifest` for wrapper-spawned Watcher, Gatekeeper, Auditor, and Probe agents. It does not mutate state, write continue reports, or spawn Go-side review workers. `continue-finalize` remains the next runtime surface.

Continue finalize bridge slice completed: `aether continue-finalize --completion-file <path|->` now re-loads current state, re-runs verification and claim checks, merges wrapper Watcher/Gatekeeper/Auditor/Probe results, writes verification/gate/review/continue reports, records continue worker flow without duplicating wrapper spawn-log entries, and advances or blocks through the existing runtime gates and atomic state transition.

Continue wrapper restoration completed: `.aether/commands/continue.yaml`, `.claude/commands/ant/continue.md`, and `.opencode/commands/ant/continue.md` now use the runtime bridge: status -> JSON `continue --plan-only` -> parse `continue_manifest` -> spawn Watcher first, then Gatekeeper/Auditor/Probe with manifest names/castes/waves -> `spawn-log`/`spawn-complete` -> completion JSON -> `continue-finalize` -> route to `/ant-build N+1`, `/ant-continue`, or `/ant-seal`. Wrapper tests now forbid the old visual pass-through continue command.

---

## Active Constraints (REDIRECT Signals)

*None active*

---

## Active Pheromones

*None active*

---

## Open Blockers

*None active*

---

## Tasks For Phase 1 — Assumptions and gap audit

- [ ] Verify Node, embed, event bus, ANSI, and v5.4 playbook assumptions
- [ ] Map current build, continue, and plan wrappers against the v5.4 direct-spawn playbooks
- [ ] Persist the blended v1.6 ceremony plan for resume and review

---

## Recent Decisions

| Date | Decision | Rationale | Made By |
|------|----------|-----------|---------|
| — | No recorded decisions | — | — |

---

## Recent Activity

- 2026-04-24T02:26:51Z|review_reconciliation|review|Specialist continuity and gatekeeper findings integrated; structured state remains Phase 1 ready
- 2026-04-24T02:40:41Z|review_verified|review|Specialist review reconciliation complete; TS narrator tests, phase-plan mirror, full Go/TS verification passed
- 2026-04-24T03:12:26Z|phase2_runtime_packaging|build|Narrator runtime made dependency-free via dist/narrator.js; install/update embeds runtime artifact; full Go/TS verification passed
- 2026-04-24T03:19:03Z|phase2_visual_contract|build|Narrator consumes visuals-dump caste metadata through --visuals; Go-owned identity contract verified
- 2026-04-24T03:24:26Z|phase2_stream_smoke|test|Event-bus stream to dependency-free narrator runtime smoke added; full Go/TS/race verification passed
- 2026-04-24T03:40:41Z|handoff|docs|Tracked v1.6 launcher and remaining-phase implementation handoff created for context recovery
- 2026-04-24T04:06:48Z|phase2_launcher|build|Go narrator launcher and build ceremony emitter implemented; full Go/TS/race/vet verification passed
- 2026-04-24T04:12:13Z|phase3_display_foundation|build|TS narrator activity frame added with active/completed/blocked worker tests
- 2026-04-24T04:17:03Z|phase3_smoke|test|Temp HOME installed-hub fallback smoke and multi-wave activity fixture passed
- 2026-04-24T04:25:35Z|phase4_plan_manifest|build|Read-only build --plan-only dispatch_manifest added for Claude/OpenCode wrapper spawning bridge
- 2026-04-24T04:34:19Z|phase4_finalize|build|build-finalize external-task manifest and claims recording added for wrapper-spawned agents
- 2026-04-24T04:42:16Z|phase4_build_wrappers|build|Claude/OpenCode build wrappers restored as real manifest-driven orchestrators
- 2026-04-24T04:51:45Z|phase5_continue_plan|build|continue --plan-only read-only manifest added for wrapper-spawned verification/review agents
- 2026-04-24T05:00:39Z|phase5_continue_finalize|build|continue-finalize added for wrapper-spawned verification/review results
- 2026-04-24T05:10:05Z|phase5_continue_wrappers|build|Claude/OpenCode continue wrappers restored as manifest-driven verification/review orchestrators

---

## Next Steps

1. Run full verification for the continue-wrapper restoration
2. Commit and push the continue-wrapper checkpoint
3. Start plan wrapper orchestration: depth prompt, Scout + Route-Setter manifests, finalizer/state handoff
4. Keep Go as state/event source of truth; wrappers own Task-tool spawning

---

## If Context Collapses

1. Run `aether resume` for the quick dashboard restore
2. Run `aether resume-colony` for the full handoff and task view
3. Read `.aether/HANDOFF.md` if a richer session summary was persisted

### Active Todos
- Commit and push the continue-wrapper checkpoint
- Restore plan wrappers as real orchestrators
