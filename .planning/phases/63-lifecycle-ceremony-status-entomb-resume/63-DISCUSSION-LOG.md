# Phase 63: Lifecycle Ceremony -- Status, Entomb, Resume - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md -- this log preserves the alternatives considered.

**Date:** 2026-04-27
**Phase:** 63-lifecycle-ceremony-status-entomb-resume
**Areas discussed:** Status Version Display, Near-Miss Wisdom Extraction, Stale Pheromone Detection

---

## Status Version Display

| Option | Description | Selected |
|--------|-------------|----------|
| Near top (after goal) | After colony goal line but before progress metrics | ✓ |
| With metadata section | Alongside milestone/depth/granularity info | |
| Footer line | At the very bottom as a CLI version footer | |

**User's choice:** Near top (after goal)

| Option | Description | Selected |
|--------|-------------|----------|
| Show both always | "Runtime: v1.0.24 | Hub: v1.0.25 MISMATCH" | |
| Show mismatch only | Only add hub comparison when versions differ | |
| You decide | Claude picks best presentation | |

**User's choice:** Should warn that they aren't matching (show both always, warn on mismatch)

| Option | Description | Selected |
|--------|-------------|----------|
| Type breakdown | "Signals: 2 FOCUS, 1 REDIRECT" | |
| With expiry note | "Signals: 3 active (2 FOCUS expire at seal, 1 REDIRECT persists)" | ✓ |
| You decide | Claude picks format | |

**User's choice:** With expiry note

---

## Near-Miss Wisdom Extraction

| Option | Description | Selected |
|--------|-------------|----------|
| Store in chamber only | Preserve in archive, user promotes manually | |
| Attempt hive promotion | Auto-call hive-promote, log failures non-blocking | |
| Log + suggest | Log to chamber manifest + output suggestion line | ✓ |

**User's choice:** Log + suggest

| Option | Description | Selected |
|--------|-------------|----------|
| Spawn trees + manifests + reviews | Session-scoped artifacts only | |
| Full temp sweep | Also clean old midden, expired pheromones, old session snapshots | ✓ |
| You decide | Claude decides what's safe to delete | |

**User's choice:** Full temp sweep

| Option | Description | Selected |
|--------|-------------|----------|
| Full stats | Phase count, plans, learnings, instincts, seal date, duration | ✓ |
| Minimal stats | Phase count, seal date, duration only | |
| You decide | Claude picks | |

**User's choice:** Full stats

---

## Stale Pheromone Detection

| Option | Description | Selected |
|--------|-------------|----------|
| Warning only | Output line listing stale signals | |
| Interactive prompt | Ask keep/clean per signal | ✓ |
| Auto-clean | Remove without asking, log what was removed | |

**User's choice:** Interactive prompt

| Option | Description | Selected |
|--------|-------------|----------|
| Runtime data + wrapper interaction | Go outputs JSON, wrappers handle prompt. Codex gets warning only. | ✓ |
| All in runtime | Go handles everything with --auto-clean-stale flag | |

**User's choice:** Runtime data + wrapper interaction

| Option | Description | Selected |
|--------|-------------|----------|
| Phase comparison | source_phase < current phase = stale | ✓ |
| Phase + age + strength | Also consider signal age and decay | |
| You decide | Claude picks heuristic | |

**User's choice:** Phase comparison

---

## Claude's Discretion

- Exact formatting of version line and signal summary
- Midden age threshold field/comparison details
- Stale signal JSON output structure
- Chamber manifest near-miss format
- Mismatch warning wording

## Deferred Ideas

None -- discussion stayed within phase scope
