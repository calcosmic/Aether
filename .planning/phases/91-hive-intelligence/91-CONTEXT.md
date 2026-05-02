# Phase 91: Hive Intelligence - Context

**Gathered:** 2026-05-02
**Status:** Ready for planning

<domain>
## Phase Boundary

Colony learning is backed by SQLite with full-text search recall, pheromone skills are auto-created from verified difficult tasks, and the Keeper curator maintains memory hygiene across the lifecycle. This phase swaps the JSON-based ColonyStore from Phase 90 into SQLite, adds FTS5 search, implements the full skill lifecycle (create/patch/edit/delete/archive/pin/promote), auto-creates skills from evidence-based difficulty detection, and runs the Keeper curator to transition unused skills through active → stale → archived stages.

</domain>

<decisions>
## Implementation Decisions

### SQLite Schema & Migration
- **D-01:** Single colony.db in .aether/data/ with all tables (runs, workers, gates, memories, skills, decisions, trajectories, schema_version). One database, one WAL mode, one migration file. Interconnected tables need to talk to each other.
- **D-02:** Go migration runner — no third-party dependencies. Map of version number to migration function, runs on startup, idempotent by checking schema_version. ~50 lines of Go.

### FTS5 Search
- **D-03:** Unified FTS5 virtual table indexing all searchable content (memories, worker summaries, decisions, gate failures). One index, simpler queries, good enough for colony-scale data.
- **D-04:** Natural language search syntax. Users type queries like "memory leak test failure" and FTS5 handles tokenization and ranking automatically. No field-specific filter syntax.

### Auto-Skill Creation
- **D-05:** Evidence-based difficulty detection. A task is "difficult" if: it required retry/replan, took significantly longer than estimated, or had multiple worker failures before success. Phase 90's learning entries already capture this evidence data.
- **D-06:** Auto mode is the default. After a difficult task, skills are created and immediately active. Off and propose modes remain available as config options for users who want more control.
- **D-07:** Hard rejection rules prevent skill creation from: failed runs, zero-modification runs, phantom builds, and runs containing secrets. These are non-negotiable safety gates.

### Keeper Curator Lifecycle
- **D-08:** 14 days per lifecycle stage. Active → stale after 14 days unused, stale → archived after another 14 days. Skills get a full month before archival.
- **D-09:** Pinned skills are fully exempt from auto-transitions. They stay active forever unless the user explicitly unpins or archives them. Immutable to both auto-transitions and agent writes.

### Claude's Discretion
- Migration runner implementation details (table structures, index design)
- FTS5 ranking algorithm configuration (BM25 weights)
- Skill content generation prompt and format
- Keeper curator execution timing (on continue, on seal, periodic)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase 90 (Direct Dependency)
- `.planning/phases/90-learning-foundation/90-CONTEXT.md` — Unified memory API decisions (D-05 through D-16), ColonyStore interface, HiveStore wrapping
- `pkg/learn/learn.go` — LearnStore interface, Entry/Evidence/Classification types that SQLite will persist
- `pkg/learn/colony_store.go` — Current JSON-based ColonyStore — Phase 91 replaces storage layer with SQLite
- `pkg/learn/classify.go` — Classification rules (D-11) that determine skill sharing scope
- `pkg/learn/evidence.go` — Evidence collection — difficulty detection reads from this
- `pkg/learn/trigger.go` — Learning eligibility trigger — auto-skill creation hooks here
- `pkg/learn/wrappers.go` — Delegation wrappers — cmd/ imports only pkg/learn/
- `pkg/learn/export.go` — Export/import for portable learning packs

### Phase 88 (Dependency)
- `.planning/phases/88-recovery-foundation/88-CONTEXT.md` — Provenance validation (SAFE-03/04) and privacy gate (PRIV-01/02)
- `cmd/provenance.go` — Build/continue provenance validation
- `cmd/security_cmds.go` — Privacy/antipattern scanner

### Existing Infrastructure
- `cmd/hive.go` — hive-init, hive-store, hive-read, hive-abstract, hive-promote (200-entry cap, LRU)
- `pkg/memory/trust.go` — Trust scoring engine (40/35/25 weighted, 7 tiers)
- `pkg/colony/context_ranking.go` — ContextCandidate scoring with budget-aware trimming
- `pkg/storage/` — JSON store with file locking — patterns for SQLite replacement
- `.aether/skills/` — Existing skill format (SKILL.md with frontmatter)
- `cmd/codex_continue_finalize.go` — Learning capture hook (Phase 90 wiring)

### Requirements
- `.planning/REQUIREMENTS.md` — HIVE-04/05/06, SKIL-01/02/03/04/05/06, AUTO-01/02/03/04

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `pkg/learn/colony_store.go` — ColonyStore interface and JSON implementation. SQLite ColonyStore will implement the same interface, making it a drop-in replacement per Phase 90's D-07.
- `pkg/storage/json_store.go` — Existing file locking and JSON persistence patterns. SQLite WAL mode replaces this locking model.
- `.aether/skills/` — Existing SKILL.md format with frontmatter (name, category, detect patterns, roles). Auto-created skills should use this format.
- `cmd/codex_continue_finalize.go` — Learning capture fires after gates pass. Auto-skill creation can hook into the same point.

### Established Patterns
- Repo isolation: colony data in `.aether/data/` subdirectory (D-06 from Phase 90)
- Evidence-gated triggers: only verified successful outcomes produce durable memory
- Progressive disclosure: skill index in prompt, full content loads on match
- `aether update` protection: user-created/modified files never overwritten

### Integration Points
- ColonyStore interface swap: Phase 91 implements the same LearnStore interface with SQLite backing. HiveStore stays unchanged.
- Learning capture in continue-finalize: auto-skill creation hooks into the existing learning capture flow
- Skill injection in colony-prime: existing skill-index/skill-match/skill-inject pipeline handles learned skills

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 91-Hive Intelligence*
*Context gathered: 2026-05-02*
