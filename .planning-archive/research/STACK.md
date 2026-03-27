# Stack Research: Colony Context Enhancement

**Domain:** Session Restoration with Rich Project Context
**Researched:** 2026-02-21
**Confidence:** HIGH

---

## Context

This is a subsequent milestone for the Aether colony system. The existing stack is validated and shipped in v3.0. This research covers ONLY the additions needed for instant session restoration with rich context, decision logging, and phase documentation.

**Existing Stack (Confirmed Working):**
- Bash 3.2+ with strict mode (`set -euo pipefail`)
- jq 1.6+ for JSON manipulation
- xmllint/xmlstarlet for XML processing
- Node.js 18+ CLI wrapper
- 34 Claude Code slash commands
- 22 specialist subagents
- COLONY_STATE.json for colony state
- QUEEN.md for wisdom storage
- session.json for session continuity
- pheromones.json for signals
- templates/ directory with 9 templates

---

## Recommended Stack Additions

### Core Technologies (No New Dependencies)

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| Bash + jq | Existing | Session restoration, context aggregation | Already proven at scale (12,352 lines in aether-utils.sh). jq handles JSON merging for context assembly. |
| Markdown | Existing | Phase documentation, decision logs | Human-readable, version-controllable, Claude-native format. |
| JSON | Existing | Structured decision storage, context snapshots | Native to existing stack, jq-queriable, git-diffable. |

**Rationale:** No new core dependencies needed. The existing bash/jq architecture is sufficient for all context restoration requirements.

### New Data Files (Not Libraries)

| File | Purpose | Format | Integration Point |
|------|---------|--------|-------------------|
| `.aether/data/decisions.json` | Structured decision log | JSON append-only | `context-update decision` command |
| `.aether/data/phases/` | Per-phase documentation | Markdown per phase | New `phase-doc-*` commands |
| `.aether/data/context-snapshot.json` | Aggregated context for fast restore | JSON cached aggregate | `session-read --rich` command |
| `.aether/HANDOFF.md` | Human-readable session state | Markdown | Existing, enhanced with decision summary |

---

## Integration with Existing Architecture

### Bash Functions to Add (aether-utils.sh)

| Function | Lines | Purpose | Uses |
|----------|-------|---------|------|
| `decision-log` | ~40 | Append structured decision to decisions.json | jq, date, sha256sum |
| `decision-query` | ~50 | Query decisions by phase, type, or date | jq |
| `phase-doc-init` | ~30 | Initialize phase documentation file | cat, date |
| `phase-doc-update` | ~40 | Update phase documentation section | awk |
| `phase-doc-read` | ~30 | Read phase documentation as JSON | jq |
| `context-snapshot-build` | ~60 | Aggregate COLONY_STATE + decisions + QUEEN wisdom + pheromones | jq |
| `context-snapshot-read` | ~30 | Read cached snapshot | cat, jq |
| `session-read-rich` | ~50 | Enhanced session read with full context | Calls snapshot-build if stale |

**Total new bash code:** ~330 lines (fits existing pattern)

### Integration Points

```
Existing Commands → New Functions
─────────────────────────────────
/ant:init         → decision-log (if resuming)
/ant:build        → phase-doc-init (new phase)
/ant:continue     → decision-log (phase completion)
/ant:resume       → session-read-rich (full context)
/ant:decision     → decision-log (user decisions)
/ant:seal         → phase-doc-read (archive docs)
```

---

## Data Schemas

### decisions.json

```json
{
  "version": "1.0",
  "colony_id": "session_1771335865738_rcwosn",
  "decisions": [
    {
      "id": "dec_1771335865_a1b2",
      "timestamp": "2026-02-21T10:00:00Z",
      "phase": 3,
      "category": "technical",
      "decision": "Use JSON over YAML for config",
      "rationale": "jq is already a dependency, simpler stack",
      "made_by": "Queen",
      "context": "Phase 3 planning"
    }
  ]
}
```

### phases/{phase_id}.md

```markdown
# Phase 3: Authentication System

## Goal
Implement JWT-based authentication

## Decisions
- [2026-02-21] Use JWT over session cookies (performance)

## Implementation Notes
- Token expiry: 24 hours
- Refresh token rotation enabled

## Status
COMPLETED — 2026-02-21T14:30:00Z
```

### context-snapshot.json

```json
{
  "snapshot_at": "2026-02-21T15:00:00Z",
  "colony_goal": "Build auth system",
  "current_phase": 3,
  "phase_summary": "Authentication complete, starting authorization",
  "recent_decisions": [...],
  "active_pheromones": [...],
  "queen_wisdom_summary": [...],
  "next_action": "/ant:build 4"
}
```

---

## Alternatives Considered

| Approach | Why Not Chosen | When It Would Be Better |
|----------|----------------|-------------------------|
| SQLite for decisions | Overkill for append-only log; adds dependency | If we need complex queries or millions of records |
| YAML for phase docs | jq doesn't natively parse YAML; adds complexity | If human editing is primary use case |
| Redis for session cache | Requires external service; violates local-first principle | Multi-user or distributed colony scenario |
| Git notes for decisions | Fragile, hard to query programmatically | If decisions must follow code in git history |

---

## What NOT to Add

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| SQLite | Single-purpose dependency, overkill for append log | JSON file + jq queries |
| YAML parser | No native jq support, adds complexity | Markdown with structured JSON frontmatter if needed |
| External database | Violates local-first, offline-first architecture | Filesystem + git |
| Complex ORM/ODM | Not needed for simple schemas | Bash functions + jq |
| Real-time sync | Premature optimization | File watchers if truly needed later |
| Binary serialization | Not human-readable, harder to debug | JSON (compressed with jq -c if size matters) |

---

## Command Additions

### New Slash Commands

| Command | Purpose | Implementation |
|---------|---------|----------------|
| `/ant:decision "text"` | Log a decision with context | Calls `decision-log` |
| `/ant:phase-doc [phase]` | View/edit phase documentation | Calls `phase-doc-read`/`update` |
| `/ant:context` | Show aggregated context snapshot | Calls `session-read-rich` |

### Enhanced Existing Commands

| Command | Enhancement |
|---------|-------------|
| `/ant:resume` | Use `session-read-rich` for full context |
| `/ant:continue` | Auto-log completion decisions |
| `/ant:build` | Auto-initialize phase documentation |

---

## Performance Considerations

| Concern | At 100 decisions | At 1,000 decisions | Mitigation |
|---------|------------------|-------------------|------------|
| decisions.json read | <10ms | ~50ms | Lazy load, cache snapshot |
| Context snapshot build | ~20ms | ~100ms | Build once, cache until mutation |
| Phase doc render | ~5ms | ~5ms | Individual files, no growth concern |

**Snapshot invalidation:** Rebuild on any COLONY_STATE.json write, decision-log append, or pheromone change.

---

## Version Compatibility

| Component | Requires | Notes |
|-----------|----------|-------|
| decision-log | jq 1.6+ | Uses `--arg` and `--argjson` |
| context-snapshot | jq 1.6+ | Uses `reduce` for aggregation |
| phase-doc | awk (any) | POSIX-compliant patterns |
| session-read-rich | All above | Graceful degradation if snapshot stale |

---

## Migration Path

**From existing COLONY_STATE.json:**

1. `memory.decisions` array already exists — migrate to `decisions.json` on first `/ant:decision` call
2. Phase info exists in `plan.phases` — generate `phases/{id}.md` on first `/ant:phase-doc` call
3. Session continuity exists in `session.json` — enhance with snapshot reference

**Backward compatibility:** All new files are additive; existing commands work without them.

---

## Sources

- `.aether/aether-utils.sh` lines 6808-7056 — Session continuity commands (verified working)
- `.aether/data/COLONY_STATE.json` — Existing state schema
- `.aether/data/session.json` — Existing session format
- `.aether/data/learning-observations.json` — Pattern for append-only JSON
- `.claude/commands/ant/resume.md` — Current resume implementation
- `.aether/CONTEXT.md` — Current context documentation approach

---

*Stack research for: Colony Context Enhancement — Instant Session Restoration*
*Researched: 2026-02-21*
*Confidence: HIGH — All recommendations build on proven existing patterns*
