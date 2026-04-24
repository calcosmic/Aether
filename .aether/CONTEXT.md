# Aether Colony — Current Context

> **This document is the colony's memory. If context collapses, read this file first.**

---

## System Status

| Field | Value |
|-------|-------|
| **Last Updated** | 2026-04-24T12:00:05Z |
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

Blended v1.6 ceremony revival plan persisted: Go owns state/events, TypeScript narrator owns visuals, wrappers own real Task-tool spawning. Current persisted colony phase: Assumptions and gap audit.

Detailed handoff note: `.aether/dreams/2026-04-24-ceremony-revival-v1.6-handoff.md`.

Important continuity distinction: structured colony state has not advanced past Phase 1. Git commits and `.aether/docs/ceremony-revival-v1.6-handoff.md` are the branch implementation progress source of truth. Do not manually edit `COLONY_STATE.json` to match code progress; advance it only through normal Aether lifecycle commands if needed.

Integrated review findings: Keeper continuity fixes; Auditor stream timeout/pagination fixes; Gatekeeper explicit `.aether/ts` embed, nested `node_modules` exclusion, CI/release/dependabot coverage, package license, and runtime-not-wired warning; Watcher phase-plan mirror and real TypeScript narrator tests.

Implemented Phase 2/3 foundation: narrator runtime ships as dependency-free `.aether/ts/dist/narrator.js`; `npm ci` is only for developer/CI checks, not installed runtime use. The narrator consumes Go-owned `visuals-dump --json` caste metadata through `--visuals`, event-bus stream output is smoke-tested through the runtime, Go auto-launch is implemented behind `AETHER_NARRATOR`, and JSON mode stays free of narrator output.

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

Plan plan-only bridge slice completed: `aether plan --plan-only --depth <fast|balanced|deep|exhaustive>` now emits a read-only JSON `plan_manifest`/`planning_manifest` for wrapper-spawned Scout and Route-Setter agents. Depth maps to existing plan granularity: fast=sprint, balanced=milestone, deep=quarter, exhaustive=major. It does not mutate colony state, write planning artifacts, or spawn Go-side planning workers. `plan-finalize` remains the next runtime surface.

Plan finalize bridge slice completed: `aether plan-finalize --completion-file <path|->` now consumes the wrapper planning manifest plus terminal Scout/Route-Setter results, validates manifest/state/dispatch identity, writes canonical SCOUT/ROUTE-SETTER/phase-plan/phase-research artifacts, records spawn-tree statuses, updates colony state to READY with the selected plan granularity, and routes to the first build phase.

Plan wrapper restoration completed: `.aether/commands/plan.yaml`, `.claude/commands/ant/plan.md`, and `.opencode/commands/ant/plan.md` now use the runtime bridge: depth ceremony -> status -> JSON `plan --plan-only --depth <choice>` -> parse `plan_manifest`/`planning_manifest` -> spawn Scout wave 1 then Route-Setter wave 2 -> `spawn-log`/`spawn-complete` -> completion JSON -> `plan-finalize` -> route to `/ant-build 1`. Wrapper tests now forbid the old direct visual plan pass-through.

Phase 4 specialist execution-plan slice is in the working tree: `dispatch_manifest` now includes `execution_plan`, each dispatch includes `execution_wave`, full builds sequence Archaeologist/Oracle/Architect/conditional Ambassador before builder/scout waves and Probe/Watcher/Measurer/Chaos after them, and Claude/OpenCode wrappers execute the runtime-owned execution plan. Task-scoped redispatch remains narrow.

TypeScript lifecycle-context rendering slice is in the working tree: `.aether/ts/narrator.ts` now treats `ceremony.<stage>.wave.start/end` generically, tracks non-build stages in the activity title, and keeps skill, pheromone, and chamber events visible in a short `Context` section. `.aether/ts/test/narrator.test.ts` covers continue-style wave events plus skill/pheromone/chamber notices, and `.aether/ts/dist/narrator.js` has been regenerated. Tracked details are in `.aether/docs/ceremony-revival-v1.6-handoff.md`.

Broad checkpoint verification passed on 2026-04-24T11:27:54Z: TS install/typecheck/test/build, `gofmt`, `git diff --check`, `go test ./cmd -count=1`, `go test ./... -count=1 -timeout 300s`, and `go vet ./...`.

GitHub PR status: PR #5, PR #6, and PR #7 are merged. Current implementation branch is `codex/ceremony-lifecycle-events-v16`, based on `origin/main` at PR #7.

Go-side wrapper smoke passed with `AETHER_OUTPUT_MODE=json go run ./cmd/aether build 1 --plan-only`: the standard-depth manifest emitted builder execution wave 11, Probe wave 12, and Watcher wave 13. True Claude/OpenCode Task-tool smoke remains pending because Codex cannot execute those platform wrapper tools directly.

Phase 6 first Go-backed lifecycle ceremony slice is complete in the working tree: `emitLifecycleCeremony` persists non-build ceremony events to `event-bus.jsonl`; `pheromone-write` now emits `ceremony.pheromone.emit`; `aether seal` now emits `ceremony.chamber.seal`. Focused tests, `go test ./cmd -count=1`, `go test ./... -count=1 -timeout 300s`, `go vet ./...`, and `git diff --check` passed.

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
- 2026-04-24T05:20:37Z|phase5_plan_manifest|build|plan --plan-only read-only manifest added for wrapper-spawned Scout and Route-Setter agents
- 2026-04-24T05:33:11Z|phase5_plan_finalize|build|plan-finalize added for wrapper-spawned Scout and Route-Setter results
- 2026-04-24T05:41:20Z|phase5_plan_wrappers|build|Claude/OpenCode plan wrappers restored as depth-prompted manifest-driven Scout/Route-Setter orchestrators
- 2026-04-24T10:44:47Z|phase6_ts_lifecycle_context|build|TS narrator now tracks generic lifecycle wave topics and context notices for skills, pheromones, and chamber sealing
- 2026-04-24T11:03:37Z|reconciliation|docs|Progress matrix added; PR #5 confirmed merged/closed; COLONY_STATE documented as lifecycle state, not branch implementation progress
- 2026-04-24T11:19:09Z|phase4_specialist_execution_plan|build|Build manifests now include execution_plan/execution_wave and sequence pre-wave/post-wave specialists explicitly
- 2026-04-24T11:27:54Z|checkpoint_verified|test|Phase 4 specialist execution-plan and TS lifecycle-context checkpoint passed TS, Go, vet, and whitespace verification
- 2026-04-24T11:33:58Z|draft_pr_opened|github|Draft PR #6 opened and branch ancestry synced with main's PR #5 squash commit
- 2026-04-24T11:36:13Z|go_side_wrapper_smoke|test|PR #6 clean/mergeable; build --plan-only emitted execution_plan for builder/probe/watcher waves
- 2026-04-24T11:56:31Z|phase6_lifecycle_events|build|Pheromone and chamber seal ceremony events now persist through Go-owned lifecycle hooks
- 2026-04-24T12:00:05Z|phase6_lifecycle_verified|test|Pheromone/chamber ceremony event slice passed focused, full cmd, full repo, vet, and whitespace checks

---

## Next Steps

1. Commit/push the Phase 6 pheromone/chamber ceremony event slice
2. Continue lifecycle event emission for plan, continue, colonize, and worker/skill context
3. Run true Claude/OpenCode build wrapper smoke with platform Task-tool agents when available
4. Keep Go as state/event source of truth; wrappers own Task-tool spawning

---

## If Context Collapses

1. Run `aether resume` for the quick dashboard restore
2. Run `aether resume-colony` for the full handoff and task view
3. Read `.aether/HANDOFF.md` if a richer session summary was persisted

### Active Todos
- Live smoke the build wrapper path with real platform agents
