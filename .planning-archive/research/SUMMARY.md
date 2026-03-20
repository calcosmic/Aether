# Project Research Summary

**Project:** Colony Context Enhancement — Instant Session Restoration
**Domain:** AI-assisted development with session continuity
**Researched:** 2026-02-21
**Confidence:** HIGH

---

## Executive Summary

Aether's existing colony system already has 80% of what's needed for instant session restoration. The foundation — COLONY_STATE.json, session.json, CONTEXT.md, and QUEEN.md — provides goal tracking, phase management, and wisdom accumulation. The gap is **rich context assembly**: a single, sub-5-minute "state of the colony" snapshot that lets a new session understand where we are, why we chose this path, and what to do next.

The recommended approach is **enhance existing, create minimal new**. Rather than duplicating GSD's multi-file planning structure, we build on Aether's proven JSON-based state management. Create **NEST.md** as a generated snapshot document (not a new source of truth), enhance existing commands to populate decision history, and add a `colony-snapshot` utility that assembles context from existing sources. This keeps the architecture clean while delivering the "instant restoration" experience.

Key risks center on **schema migration** (adding fields without breaking existing colonies), **context overload** (dumping too much information), and **stale session detection** (knowing when context is fresh vs. when to regenerate). These are mitigated through additive-only schema changes, recency-weighted pruning (last 5 decisions, last 10 events), and unified freshness checking.

---

## Key Findings

### Recommended Stack

No new dependencies required. The existing bash/jq architecture (12,352 lines in aether-utils.sh) is sufficient. All additions are data files and bash functions, not libraries.

**Core technologies:**
- **Bash + jq (existing)**: Session restoration, context aggregation — already proven at scale
- **Markdown (existing)**: Phase documentation, decision logs — human-readable, Claude-native
- **JSON (existing)**: Structured decision storage — native to existing stack, git-diffable

**New data files (not libraries):**
- `.aether/NEST.md`: Generated colony snapshot for instant restoration
- `.aether/data/decisions.json`: Structured decision log (append-only)
- `.aether/data/phases/`: Per-phase documentation
- `.aether/data/context-snapshot.json`: Cached aggregate for fast restore

**New bash functions (~330 lines total):**
- `colony-snapshot`: Assemble NEST.md from all sources
- `decision-log`: Append structured decision to decisions.json
- `decision-query`: Query decisions by phase, type, or date
- `session-read-rich`: Enhanced session read with full context

---

### Expected Features

**Must have (table stakes):**
- **Goal Preservation** — Already in COLONY_STATE.json, session.json
- **Current Phase/Position** — session.json has current_phase, needs enhancement
- **Recent Activity Log** — activity.log exists, needs summarization
- **Active Constraints** — pheromones.json already captures REDIRECTs
- **Next Action Clarity** — session.json has suggested_next, needs context

**Should have (competitive differentiators):**
- **Decision Archaeology** — Show not just *what* was decided but *why*
- **NEST.md Snapshot** — Single-file colony state for instant restoration
- **Phase-Aware Resumption** — Resume knows if mid-phase, between phases, or ad-hoc
- **Pheromone Persistence** — User signals survive session breaks (already implemented)

**Defer (v2+):**
- **TRAILS/ Decision History** — Structured decision archive (last 20 with rationale)
- **BROOD/ Phase Index** — Quick-reference phase completion status
- **Auto-NEST-update** — Update NEST.md on significant events
- **ROYAL-CHAMBER/ Cross-colony search** — Search decisions across archived colonies

---

### Architecture Approach

**Principle: Enhance existing, create minimal new.**

The existing architecture has COLONY_STATE.json (source of truth), CONTEXT.md (session notes), QUEEN.md (wisdom), and session.json (tracking). Rather than replacing these, we:

1. **Enhance COLONY_STATE.json**: Add `phase_summaries[]` array, populate `decisions[]` (currently empty)
2. **Create NEST.md**: New generated snapshot — single document for instant context restoration
3. **Create colony-snapshot**: New aether-utils.sh subcommand — assembles NEST.md from all sources
4. **Enhance resume.md**: Read NEST.md instead of multiple files, render richer dashboard
5. **Enhance build.md/continue.md**: Write decisions during execution, update phase summaries

**Major components:**
1. **colony-snapshot** — Reads COLONY_STATE.json, CONTEXT.md, QUEEN.md, session.json; writes NEST.md
2. **NEST.md** — Generated snapshot with goal, position, recent decisions, next action
3. **Enhanced /ant:resume** — Uses NEST.md for rich restoration dashboard
4. **Decision logging** — Populate memory.decisions[] during builds

---

### Critical Pitfalls

1. **State Schema Migration Hell** — Adding fields to COLONY_STATE.json breaks existing colonies.
   - *Avoid:* Additive-only changes, default value injection (`jq '.field // default'`), auto-migration on access
   - *Address in:* Phase 1 (Core Infrastructure)

2. **Session Stale vs. Fresh Confusion** — Can't distinguish "user cleared intentionally" vs "session crashed."
   - *Avoid:* Intent tracking (record why session ended), unified freshness check, explicit resume required
   - *Address in:* Phase 1 (Core Infrastructure)

3. **Context Overload** — Dumping everything overwhelms users and exceeds context windows.
   - *Avoid:* Recency-weighted pruning (last 5 decisions, last 10 events), hierarchical context, explicit "you are here" marker
   - *Address in:* Phase 2 (Context Aggregation)

4. **The memory.decisions[] Gap** — Decisions array is currently always empty. Adding restoration without populating it creates a "zombie feature."
   - *Avoid:* Ensure /ant:continue and other commands populate decisions before enabling restoration
   - *Address in:* Phase 2 (Context Aggregation)

5. **Backwards Compatibility Break** — Modifying existing commands breaks behavior users rely on.
   - *Avoid:* New commands for new behavior, opt-in flags, output format versioning
   - *Address in:* All phases

---

## Implications for Roadmap

Based on research, suggested phase structure:

### Phase 1: Core Infrastructure
**Rationale:** Schema safety and freshness detection are foundational. Must be solid before building on them.
**Delivers:**
- Schema validation and auto-migration for COLONY_STATE.json
- Unified session freshness detection (fix stale vs fresh confusion)
- Template composition pattern (prevent template explosion)
- Lock safety verification
**Addresses:** Goal preservation, current phase tracking, active constraints
**Avoids:** State Schema Migration Hell, Session Stale vs Fresh Confusion, Lock Contention

### Phase 2: Context Aggregation
**Rationale:** Depends on Phase 1 infrastructure. Builds the core "instant restoration" capability.
**Delivers:**
- colony-snapshot subcommand
- NEST.md generation from all sources
- Decision logging in /ant:continue and /ant:build
- Phase summary population
- Recency-weighted pruning (prevent context overload)
**Uses:** Bash + jq stack, new data files (decisions.json, phases/)
**Implements:** colony-snapshot component, NEST.md snapshot
**Avoids:** Context Overload, The memory.decisions[] Gap, Continue Here Marker Ambiguity

### Phase 3: Enhanced Resume
**Rationale:** Depends on Phase 2's NEST.md. Delivers the user-facing "instant restoration" experience.
**Delivers:**
- Enhanced /ant:resume command reading NEST.md
- Rich restoration dashboard with goal, position, decisions, next action
- Context validation (sanity checks, repo matching)
- Preview before restore
**Implements:** Enhanced resume.md component
**Avoids:** Context Restoration Without Validation

### Phase 4: Decision Archaeology
**Rationale:** Adds the "why" behind decisions — key differentiator. Depends on Phase 2's decision logging.
**Delivers:**
- Decision rationale capture in build/continue commands
- TRAILS/decisions.json with full rationale
- Decision query commands (/ant:decision, /ant:phase-doc)
**Addresses:** Decision Archaeology differentiator
**Avoids:** Wisdom Drift from Auto-Promotion (always require user approval)

### Phase 5: Polish & Backwards Compatibility
**Rationale:** Ensures existing colonies work with new features. Migration and testing.
**Delivers:**
- Migration path for existing colonies (v1.0, v2.0, v3.0)
- Backwards compatibility verification
- Documentation updates
- Performance optimization (caching, lazy loading)
**Avoids:** Data Loss During Migration, Backwards Compatibility Break

---

### Phase Ordering Rationale

- **Phase 1 first:** Schema safety and freshness detection are prerequisites. Can't build context aggregation on shaky foundations.
- **Phase 2 before Phase 3:** NEST.md must exist before resume.md can use it.
- **Phase 2 before Phase 4:** Decision logging must work before we can query decisions.
- **Phase 5 last:** Migration and compatibility testing need the features to be complete.

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 2:** Decision capture timing — when exactly to write decisions? (needs UX research)
- **Phase 4:** Decision rationale UI — how to capture "why" without interrupting flow? (needs UX research)

Phases with standard patterns (skip research-phase):
- **Phase 1:** Schema migration patterns already exist in migrate-state.md
- **Phase 3:** Resume command pattern already exists
- **Phase 5:** Migration patterns established in v3.0

---

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | No new dependencies; builds on proven bash/jq architecture |
| Features | HIGH | Based on direct analysis of existing Aether codebase |
| Architecture | HIGH | Direct codebase analysis; GSD patterns map cleanly to Aether |
| Pitfalls | HIGH | Aether-specific risks identified from actual state files |

**Overall confidence:** HIGH

All recommendations build on existing, proven patterns. The existing architecture (COLONY_STATE.json, session.json, CONTEXT.md, QUEEN.md) provides a solid foundation. No experimental technologies or unproven approaches.

### Gaps to Address

1. **Decision capture UX:** When exactly to prompt for decision logging? During build, or after? Need to validate during Phase 2 planning.
2. **NEST.md freshness threshold:** How often to regenerate? Every command, hourly, or on-demand? Recommend starting with on-demand (in resume.md) and measuring.
3. **Phase summary detail:** How much detail per phase? Recommend 2-3 bullet points, validate with users.

---

## Ant-Themed Naming

| Concept | Generic Name | Ant-Themed Name | Status |
|---------|--------------|-----------------|--------|
| Session restoration doc | CONTEXT.md | **NEST.md** | New |
| Decision history | decisions/ | **TRAILS/** | Future (v2) |
| Phase documentation | phases/ | **BROOD/** | Future (v2) |
| Decision archive | archive/ | **ROYAL-CHAMBER/** | Future (v2) |
| Context injection | prime | **colony-prime** | Already exists |
| Resume command | resume | **/ant:resume** | Already exists |

**Naming principles:**
1. Use ant colony metaphors consistently
2. Prefer concrete nouns (NEST, TRAIL, BROOD) over abstract (CONTEXT, HISTORY)
3. Align with existing pheromone metaphor
4. Keep it pronounceable — "check the NEST" not "check the session-restoration-document"

---

## Sources

### Primary (HIGH confidence)
- `.aether/data/COLONY_STATE.json` — existing state schema
- `.aether/data/session.json` — session tracking implementation
- `.aether/aether-utils.sh` — utility functions (session-read, load-state, context-update)
- `.claude/commands/ant/resume.md` — resume command implementation
- `.aether/CONTEXT.md` — current context document
- `.aether/docs/QUEEN.md` — wisdom system
- `.aether/docs/pheromones.md` — signal system

### Secondary (MEDIUM confidence)
- GSD research patterns — architecture mapping reference
- CLI tool best practices — state management patterns

---

*Research completed: 2026-02-21*
*Ready for roadmap: yes*
