# Phase 162: Switch On Learning - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-04
**Phase:** 162-switch-on-learning
**Areas discussed:** Hive Brain default, What you see & failure behavior, The before/after proof

---

## Area Selection

Four gray areas were offered: Which system is authoritative (LEARN-03), Hive
Brain default (LEARN-04), What you see & failure behavior (LEARN-01/02), The
before/after proof (LEARN-05). The user selected the latter three; the
LEARN-03 reconciliation was left unselected and treated as delegated to
Claude (locked as D-09/D-10 in CONTEXT.md).

---

## Hive Brain default

| Option | Description | Selected |
|--------|-------------|----------|
| On, full (Recommended) | Read cross-colony wisdom AND promote at seal; matches "everything networked together"; promotion non-blocking, 200-cap LRU | ✓ |
| On, read-only | Automatic read, promotion stays opt-in | |
| Keep off, fix the docs | No behavior change; honesty fix only | |

**User's choice:** On, full

| Option | Description | Selected |
|--------|-------------|----------|
| One switch: env var (Recommended) | Default `promote` when unset; `=off`/`=read` overrides; consent-file mechanism retired | ✓ |
| Keep both, flip defaults | Env var and consent file both survive as opt-outs | |
| Per-colony opt-out | Global default on, per-colony `hive: off` setting | |

**User's choice:** One switch: env var
**Notes:** Ended area after two questions ("Next area").

---

## What you see & failure behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Runtime, at phase-end (Recommended) | continue-finalize invokes consolidation when a phase completes/advances; no wrapper protocol; no clash with Phase 165 | ✓ |
| Runtime, every continue | Runs on every continue including mid-phase loops | |
| Wrapper-invoked step | continue.md instructs the orchestrator to run the subcommand | |

**User's choice:** Runtime, at phase-end

| Option | Description | Selected |
|--------|-------------|----------|
| Warn loudly, never block (Recommended) | Phase advances / seal completes with unmissable "advanced WITHOUT consolidation" warning; enrichment, not gate | ✓ |
| Block the continue/seal | Failure stops the lifecycle command | |
| Silent log only | Log file, no interruption (listed for completeness) | |

**User's choice:** Warn loudly, never block

| Option | Description | Selected |
|--------|-------------|----------|
| Full colony ceremony (Recommended) | Caste-styled learning beat at phase end; eight named curation ants announce at seal, plus decay counts and archive path | ✓ |
| Compact summary block | Plain stage section, numbers only | |
| Detailed listing | Every promoted instinct in full (belongs in /ant-memory-details) | |

**User's choice:** Full colony ceremony

| Option | Description | Selected |
|--------|-------------|----------|
| Always show it (Recommended) | Zero-activity phases still print "colony observed nothing new this phase" | ✓ |
| Only when there's activity | Beat appears only on promotions | |

**User's choice:** Always show it

---

## The before/after proof

| Option | Description | Selected |
|--------|-------------|----------|
| Two-layer proof (Recommended) | Deterministic CI test on brief assembly (populated vs wiped) + one recorded real before/after run as a phase artifact | ✓ |
| Recorded demo only | One-time artifact, no permanent regression guard | |
| Repeatable proof command | `aether learning-proof` runs a real worker twice on demand; Phase 171 territory | |

**User's choice:** Two-layer proof
**Notes:** User then chose "I'm ready for context".

---

## Claude's Discretion

- LEARN-03 reconciliation (unselected area): pkg/memory authoritative,
  pkg/learn subordinate capture layer — locked as D-09/D-10
- Ceremony wording/formats, seal artifact location/contents, phase-advance
  detection mechanics, decision-record location, test fixture mechanics,
  standalone subcommand retention

## Deferred Ideas

- Repeatable `learning-proof` harness → Phase 171
- `queen-seed-from-hive` reconnection → Phase 170 (RECLAIM-09)
- Hive trust redesign → shelved at milestone level
- Reviewed-not-folded todos: continue-finalize `--reconcile-task` gate bug;
  ts-host preflight timeout (both keyword matches, previously deferred by
  163.2/164/165)
