# Aether Colony — Current Context

> **This document is the colony's memory. If context collapses, read this file first.**

---

## System Status

| Field | Value |
|-------|-------|
| **Last Updated** | 2026-05-05T01:41:05Z |
| **Current Phase** | 4 |
| **Phase Name** | Agent-Delegate for oracle Command |
| **Phase Status** | pending |
| **Milestone** | First Mound |
| **Colony Status** | READY |
| **Safe to Clear?** | YES — Plan persisted, ready for the next command |

---

## Current Goal

Fix agent dispatch platform mismatch for Claude Code and OpenCode agent sessions

---

## What's In Progress

Pre-compact snapshot (auto): state=READY phase=4 goal=Fix agent dispatch platform mismatch for Claude Code and OpenCode agent sessions task=Agent-Delegate for oracle Command

---

## Active Constraints (REDIRECT Signals)

| Constraint | Source | Date Set |
|------------|--------|----------|
| Do not change NewWorkerInvoker() to return FakeInvoker inside agents — that produces synthetic results. The correct path is plan-only + host-agent dispatch +... | pheromone | active |
| Do not modify --plan-only or -finalize code paths — they already work. Build agent-delegate as a thin routing layer on top. | pheromone | active |
| Do not solve reviewer timeouts by increasing timeouts or deleting the specialist reviewer agents; solve it with intent-aware orchestration and advisory-vs-bl... | pheromone | active |

---

## Active Pheromones

*None active*

---

## Open Blockers

- test
- test
- test

---

## Tasks For Phase 4 — Agent-Delegate for oracle Command

- [ ] Add agent-delegate guard in oracle_loop.go before invoker call
- [ ] Tests for oracle agent-delegate guard
- [ ] Update .claude/commands/ant/oracle.md and .opencode/commands/ant/oracle.md for agent-delegate

---

## Recent Decisions

| Date | Decision | Rationale | Made By |
|------|----------|-----------|---------|
| — | No recorded decisions | — | — |

---

## Recent Activity (Last 5 Events)

- 2026-05-04T23:48:38Z|build_dispatched|build|Dispatched 6 workers for phase 2
- 2026-05-05T00:01:14Z|build_completed|build-finalize|Phase 2 external Task workers recorded
- 2026-05-05T00:05:58Z|phase_started|build|Phase 3: Agent-Delegate for swarm Command
- 2026-05-05T00:05:58Z|build_dispatched|build|Dispatched 6 workers for phase 3
- 2026-05-05T01:07:09Z|build_completed|build-finalize|Phase 3 external Task workers recorded

---

## Next Steps

1. Run `aether build 4`
2. Run `aether phase --number 4` to inspect the tracked phase details
3. Run `aether resume-colony` after a context clear if you want the full recovery view

---

## If Context Collapses

1. Run `aether resume` for the quick dashboard restore
2. Run `aether resume-colony` for the full handoff and task view
3. Read `.aether/HANDOFF.md` if a richer session summary was persisted

### Active Todos
- Add agent-delegate guard in oracle_loop.go before invoker call
- Tests for oracle agent-delegate guard
- Update .claude/commands/ant/oracle.md and .opencode/commands/ant/oracle.md for agent-delegate
