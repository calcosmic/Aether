# Aether Colony — Current Context

> **This document is the colony's memory. If context collapses, read this file first.**

---

## System Status

| Field | Value |
|-------|-------|
| **Last Updated** | 2026-04-24T03:40:41Z |
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

Specialist reviews for the launcher slice are complete. Scout recommends build-specific lifecycle insertion points in `cmd/codex_build_worktree.go` and `cmd/codex_build.go`. Watcher listed launcher tests for env gating, missing Node/runtime, early exits, cleanup, event persistence, and JSON non-pollution. Gatekeeper requires absolute `node` + `dist/narrator.js`, no shell/npm/npx/tsx at runtime, non-fatal missing dependencies, and child stdout routed back through Go's visual output mutex.

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

---

## Next Steps

1. Read `.aether/docs/ceremony-revival-v1.6-handoff.md`
2. Implement the launcher slice: `cmd/narrator_launcher.go`, `cmd/ceremony_emitter.go`, tests, and build lifecycle event insertion
3. Run the Go/TS verification block listed in the handoff
4. Commit and push after the launcher slice is green

---

## If Context Collapses

1. Run `aether resume` for the quick dashboard restore
2. Run `aether resume-colony` for the full handoff and task view
3. Read `.aether/HANDOFF.md` if a richer session summary was persisted

### Active Todos
- Implement Go narrator launcher gating and sidecar cleanup
- Emit build ceremony events from the dispatch lifecycle
- Add launcher/emitter tests for JSON safety, missing Node/runtime, early exits, and event persistence
- Run the full Go/TS verification block from `.aether/docs/ceremony-revival-v1.6-handoff.md`
