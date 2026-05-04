# Phase 96: Auto-Recovery Orchestrator - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-05-03
**Phase:** 96-auto-recovery-orchestrator
**Areas discussed:** Recovery Sequence Order, Build vs Continue Split, RECV-04 vs Phase 95 Overlap, Budget Tracking

---

## Recovery Sequence Order

| Option | Description | Selected |
|--------|-------------|----------|
| Peer first, retry second | Reassign to peer before retrying same worker | |
| Retry first, peer second | Retry same worker up to budget before reassigning | |
| Classification-dependent | Transient → retry first, systemic → reassign first | ✓ |

**User's choice:** Classification-dependent
**Notes:** The existing 3-tier classification (recoverable/requires-attempt/blocking) provides enough control — no finer tiers needed.

### Follow-up: Transient retry count before peer reassignment

| Option | Description | Selected |
|--------|-------------|----------|
| 1-2 retries then reassign | Gives transient issues a chance before involving another worker | |
| Full budget then reassign | Retry same worker up to full budget (3) | |
| 1 retry then immediate peer | Fastest escalation to healthy workers | ✓ |

**User's choice:** 1 retry then immediate peer
**Notes:** Balances giving transient issues a chance with fast escalation.

### Follow-up: Existing tiers sufficient?

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, existing tiers work | Blocking skips retry, requires-attempt tries once, recoverable gets retry+peer | ✓ |
| No, need finer tiers | Add sub-classifications for retry-first vs reassign-first | |

**User's choice:** Yes, existing tiers work

---

## Build vs Continue Split

| Option | Description | Selected |
|--------|-------------|----------|
| Unified orchestrator | One autoRecover() function called from both build and continue | |
| Separate wiring points | Build handles retry/reassign, continue handles Fixer — no shared code | |
| Shared logic, separate callers | Core decision function is shared, build and continue call it with different sources | |

**User's choice:** You decide (Claude's discretion)
**Notes:** User deferred the architectural choice to Claude.

---

## RECV-04 vs Phase 95 Overlap

| Option | Description | Selected |
|--------|-------------|----------|
| Fixer as recovery strategy | Phase 96 wires Fixer into the retry/reassign/fixer sequence as a third strategy | ✓ |
| Phase 95 covers it | No new Fixer dispatch needed, Phase 95 already dispatches Fixer | |
| Expand to all gates | Dispatch Fixer for all non-hard_block gate failures | |

**User's choice:** Fixer as recovery strategy
**Notes:** Same Fixer, different trigger context. Phase 95 dispatches Fixer as a gate outcome; Phase 96 dispatches Fixer as a recovery strategy in the retry sequence.

---

## Budget Tracking

### Where to track budget

| Option | Description | Selected |
|--------|-------------|----------|
| Recovery-log file | Add recovery_budget object to recovery-log file, co-located with actions | |
| COLONY_STATE.json | Add per-phase counters to colony state | |
| Count existing entries | Scan recovery log entries to compute budget usage | |

**User's choice:** You decide (Claude's discretion)
**Notes:** User deferred storage choice to Claude.

### Budget reset strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Per-wave reset | Each wave gets its own budget, matching circuit breaker D-06 | ✓ |
| Phase-wide budget | 3 retries total across all waves | |
| Per-worker budget | Each worker gets its own retry counter | |

**User's choice:** Per-wave reset
**Notes:** Matches circuit breaker per-wave reset pattern.

---

## Claude's Discretion

- Build vs continue architecture (unified vs shared logic vs separate) — user said "you decide"
- Budget tracking location (recovery-log vs COLONY_STATE vs count entries) — user said "you decide"

## Deferred Ideas

None — discussion stayed within phase scope.
