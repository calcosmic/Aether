# Structural Learning Stack

> Memory consolidation pipeline: from raw observations to trusted wisdom.

---

## Overview

The Structural Learning Stack is how Aether converts raw, repeated observations into
colony wisdom that workers can act on. Each stage adds structure: trust scores quantify
reliability, the graph records relationships between instincts, curation ants clean and
promote, and the event bus connects everything loosely.

The stack runs automatically at phase-end and seal — workers never call it directly.

---

## Architecture Diagram

```
                          learning-observe
                               │
                    [learning-observations.json]
                               │
                         trust-calculate
                               │
                               ▼
                    [instinct-store] ──────────► [instinct-graph.json]
                               │                  (graph-link edges)
                               │
                    ┌──────────▼──────────┐
                    │   Curation Ants     │
                    │  (phase-end / seal) │
                    └──────────┬──────────┘
                               │
               ┌───────────────┼───────────────┐
               ▼               ▼               ▼
          [QUEEN.md]    [event-bus.jsonl]  [archived]
        (via herald)    (consolidation     (low-trust
                         events)           instincts)
```

---

## Pipeline Stages

### 1. Observation (pkg/memory/observe.go — `_learning_observe`)

Records an observation of a learning pattern into `.aether/data/learning-observations.json`.

- **Deduplication:** A SHA-256 hash of the content is used as the key. If an identical observation already exists, `observation_count` is incremented and `last_seen` is updated rather than creating a duplicate entry.
- **Trust score on write:** For new observations, `trust-calculate` is called immediately using the provided `source_type` and `evidence_type` (defaults: `observation` / `anecdotal`). The computed score is stored as `trust_score`.
- **Backup rotation:** A 3-file rotating backup (`*.bak.1`, `*.bak.2`, `*.bak.3`) is maintained on every write. On read, if the file is corrupt, the most recent valid backup is restored automatically.
- **Valid wisdom types:** `philosophy`, `pattern`, `redirect`, `stack`, `decree`, `failure`

Observation schema fields: `content_hash`, `content`, `wisdom_type`, `observation_count`, `first_seen`, `last_seen`, `colonies`, `trust_score`, `source_type`, `evidence_type`, `compression_level`.

---

### 2. Trust Scoring Engine (pkg/memory/trust.go)

A pure calculation module. No state is read or written. All functions accept `--flag value` arguments and return JSON.

**Weighted formula:**
```
score = 0.40 × source_score + 0.35 × evidence_score + 0.25 × activity_score
```

**Source types (40% weight):**

| Type | Score |
|------|-------|
| user_feedback | 1.0 |
| error_resolution | 0.9 |
| success_pattern | 0.8 |
| observation | 0.6 |
| heuristic | 0.4 |

**Evidence types (35% weight):**

| Type | Score |
|------|-------|
| test_verified | 1.0 |
| multi_phase | 0.9 |
| single_phase | 0.7 |
| anecdotal | 0.4 |

**Activity score (25% weight):** `0.5 ^ (days_since_last_use / 60)` — a 60-day half-life decay from the last observed use.

**Floor:** Score is never below `0.2`.

**Trust tiers:**

| Score range | Tier | Index |
|-------------|------|-------|
| 0.90 – 1.00 | canonical | 0 |
| 0.80 – 0.89 | trusted | 1 |
| 0.70 – 0.79 | established | 2 |
| 0.60 – 0.69 | emerging | 3 |
| 0.45 – 0.59 | provisional | 4 |
| 0.30 – 0.44 | suspect | 5 |
| 0.20 – 0.29 | dormant | 6 |

---

### 3. Event Bus (pkg/events/event.go)

A JSONL append-log at `.aether/data/event-bus.jsonl`. Each line is one JSON event object.

**Event fields:** `id`, `topic`, `payload`, `source`, `timestamp`, `ttl_days`, `expires_at`

- **Pub/sub pattern:** Publish to a topic with `event-publish`; subscribe by topic pattern with `event-subscribe`. Patterns support exact match or prefix wildcard (e.g., `"consolidation.*"`).
- **TTL:** Default 30 days per event. Events with `expires_at` in the past are excluded from subscriptions and removed by cleanup.
- **Concurrency:** File locking via `acquire_lock`/`release_lock` on every append and cleanup rewrite.
- **Replay:** `event-replay` returns events for a topic from a given ISO-8601 timestamp, sorted chronologically.
- **Cleanup:** `event-cleanup` atomically rewrites the JSONL file, keeping only non-expired events. Supports `--dry-run`.

---

### 4. Instinct Storage (pkg/memory/promote.go)

Persistent trust-scored instincts at `.aether/data/instincts.json` (schema version 1.0).

**Instinct schema:**
```json
{
  "id": "inst_<timestamp>_<hex>",
  "trigger": "...",
  "action": "...",
  "domain": "...",
  "trust_score": 0.75,
  "trust_tier": "established",
  "confidence": 0.8,
  "provenance": {
    "source": "...",
    "source_type": "observation",
    "evidence": "...",
    "created_at": "<ISO8601>",
    "last_applied": null,
    "application_count": 0
  },
  "application_history": [],
  "related_instincts": [],
  "archived": false
}
```

- **Deduplication:** First 50 characters of `trigger` are matched against existing active entries. On match, confidence is boosted to `max(existing, new)` and the trust score is recomputed.
- **50-instinct cap:** When active count exceeds 50 after insert, the entry with the lowest `trust_score` is soft-deleted (`archived: true`).
- **Decay:** `instinct-decay-all` applies the same 60-day half-life decay to all active entries. Entries whose decayed score falls below `0.25` are automatically archived.
- **Read:** `instinct-read-trusted` returns active entries filtered by `min-score` (default 0.5), optionally by domain, sorted by `trust_score` descending.

---

### 5. Graph Layer (pkg/graph/graph.go)

Tracks directed relationships between instincts at `.aether/data/instinct-graph.json`.

**Edge schema:** `edge_id`, `source`, `target`, `relationship`, `weight`, `created_at`

**Relationship types:** `reinforces`, `contradicts`, `extends`, `supersedes`, `related`

**Operations:**

| Function | Description |
|----------|-------------|
| `graph-link` | Create or update a directed edge. Duplicate `source+target+relationship` updates the weight. Default weight: 0.5. |
| `graph-neighbors` | 1-hop lookup. Direction: `out`, `in`, or `both`. Optional relationship filter. |
| `graph-reach` | BFS traversal up to N hops (max 3). Returns `{id, hop, path}` for each reachable node. Optional `--min-weight` filter. Early exit when no new nodes are found. |
| `graph-cluster` | Finds clusters of strongly connected instincts using greedy union-find over qualifying edges. Defaults: `min-edges=2`, `min-weight=0.3`. Returns `{nodes, edge_count, avg_weight}` per cluster. |

---

### 6. Curation Ants

Eight specialized ants that maintain memory quality. The orchestrator (`curation-run`) runs all eight in sequence. Sentinel can abort the run early if critical corruption is detected.

**Execution order:**

```
sentinel → nurse → critic → herald → janitor → archivist → librarian → scribe
```

**Ant responsibilities:**

| Ant | Responsibility |
|-----|---------------|
| sentinel | Health check on all memory stores. Aborts subsequent steps if critical corruption is found. |
| nurse | Recalculates trust scores for observations and instincts using current elapsed days. |
| critic | Detects contradictions between instincts (overlapping triggers with conflicting actions). |
| herald | Promotes high-trust instincts to QUEEN.md Patterns/Philosophies section. |
| janitor | Removes expired events from the event bus; prunes old archived instincts. |
| archivist | Archives active instincts whose trust score has fallen below a configurable threshold (default: 0.3 at seal). |
| librarian | Collects inventory statistics across all memory stores. |
| scribe | Generates a markdown consolidation report. |

All steps are non-blocking — a failure in one step is logged and execution continues.

---

### 7. Consolidation Pipeline

Two consolidation modes, each calling into the curation ants.

**Phase-end (lightweight) — `consolidation-phase-end`:**

Runs at the end of every phase (`/ant-continue`). Executes three ants only: `nurse → herald → janitor`. Publishes a `consolidation.phase_end` event on the event bus. All three steps are non-blocking.

**Seal (full) — `consolidation-seal`:**

Runs once during `/ant-seal`. Executes five steps:

1. `curation-run` — full 8-ant orchestration
2. `instinct-decay-all` — final trust decay pass across all active instincts
3. `curation-archivist --threshold 0.3` — archive borderline instincts
4. `event-publish` — publish `consolidation.seal` event
5. `curation-scribe` — generate final consolidation report

All steps are non-blocking. The seal report path is returned in the output.

---

## Integration Points

| Trigger | Stack call | Effect |
|---------|-----------|--------|
| `/ant-continue` | `consolidation-phase-end` | nurse + herald + janitor; phase_end event |
| `/ant-seal` | `consolidation-seal` | full 8-ant curation + decay + archive + seal event + report |
| `/ant-build` (pattern capture) | `learning-observe` | Records observation with trust score |
| `colony-prime` | `instinct-read-trusted` | Injects trusted instincts into worker prompts |

---

## Subcommand Reference

| Subcommand | Module | Usage |
|-----------|--------|-------|
| `trust-calculate` | pkg/memory/trust.go | `--source <type> --evidence <type> --days-since <N>` |
| `trust-decay` | pkg/memory/trust.go | `--score <float> --days <N>` |
| `trust-tier` | pkg/memory/trust.go | `--score <float>` |
| `event-publish` | pkg/events/event.go | `--topic <topic> --payload <json> [--source <src>] [--ttl <days>]` |
| `event-subscribe` | pkg/events/event.go | `--topic <pattern> [--since <ISO>] [--limit <N>]` |
| `event-cleanup` | pkg/events/event.go | `[--dry-run]` |
| `event-replay` | pkg/events/event.go | `--topic <topic> --since <ISO> [--limit <N>]` |
| `instinct-store` | pkg/memory/promote.go | `--trigger <t> --action <a> --domain <d> --confidence <f> --source <s> --evidence <e> [--source-type <type>]` |
| `instinct-read-trusted` | pkg/memory/promote.go | `[--min-score <f>] [--domain <d>] [--limit <N>]` |
| `instinct-decay-all` | pkg/memory/promote.go | `[--days <N>] [--dry-run]` |
| `instinct-archive` | pkg/memory/promote.go | `--id <id>` |
| `graph-link` | pkg/graph/graph.go | `--source <id> --target <id> --relationship <type> [--weight <float>]` |
| `graph-neighbors` | pkg/graph/graph.go | `--id <id> [--direction out\|in\|both] [--relationship <type>]` |
| `graph-reach` | pkg/graph/graph.go | `--id <id> --hops <N> [--min-weight <float>]` |
| `graph-cluster` | pkg/graph/graph.go | `[--min-edges <N>] [--min-weight <float>]` |
| `curation-sentinel` | pkg/agent/curation/sentinel.go | `[--dry-run]` |
| `curation-nurse` | pkg/agent/curation/nurse.go | `[--dry-run]` |
| `curation-critic` | pkg/agent/curation/critic.go | `[--dry-run]` |
| `curation-herald` | pkg/agent/curation/herald.go | `[--dry-run]` |
| `curation-janitor` | pkg/agent/curation/janitor.go | `[--dry-run]` |
| `curation-archivist` | pkg/agent/curation/archivist.go | `[--threshold <f>] [--dry-run]` |
| `curation-librarian` | pkg/agent/curation/librarian.go | `[--dry-run]` |
| `curation-scribe` | pkg/agent/curation/scribe.go | `[--dry-run]` |
| `curation-run` | pkg/agent/curation/orchestrator.go | `[--dry-run] [--verbose]` |
| `consolidation-phase-end` | pkg/memory/consolidate.go | `[--dry-run]` |
| `consolidation-seal` | pkg/memory/consolidate.go | `[--dry-run]` |

---

## Test Coverage

| Test file | What it covers |
|-----------|---------------|
| `tests/bash/test-pkg/memory/trust.go` | `trust-calculate`, `trust-decay`, `trust-tier`; formula weights, floor, tiers |
| `tests/bash/test-pkg/events/event.go` | `event-publish`, `event-subscribe`, `event-cleanup`, `event-replay`; TTL, locking, wildcard patterns |
| `tests/bash/test-pkg/memory/promote.go` | `instinct-store`, `instinct-read-trusted`, `instinct-decay-all`, `instinct-archive`; 50-cap, dedup, decay archival |
| `pkg/memory/instincts_test.go` | Instinct application and provenance tracking |
| `tests/bash/test-pkg/graph/graph.go` | `graph-link`, `graph-neighbors`, `graph-reach`, `graph-cluster`; BFS traversal, cluster detection |
| `pkg/agent/curation/sentinel_test.go` | Core curation ant behavior (sentinel, nurse, critic) |
| `pkg/agent/curation/janitor_test.go` | Operational curation ants (herald, janitor, archivist, librarian, scribe) |
| `tests/bash/test-curation-orchestrator.go` | `curation-run` full sequence, sentinel abort path |
| `tests/bash/test-pkg/memory/consolidate.go` | `consolidation-phase-end`; phase-end step sequence, event publishing |
| `tests/bash/test-pkg/memory/consolidate.go` | `consolidation-seal`; full seal sequence, dry-run mode |
| `pkg/memory/observe_test.go` | `learning-observe`; trust score on write, deduplication, observation_count increment |
| `pkg/storage/backup_test.go` | Backup rotation and corrupt-file recovery logic |
| `pkg/memory/trust_test.go` | Trust scoring integration with oracle/research paths |
