# Requirements: Aether v2.6

**Defined:** 2026-03-29
**Core Value:** The system must reliably interpret a user request, decompose it into executable work, verify outputs, and ship correct work with minimal user back-and-forth.

## v2.6 Requirements

Requirements for v2.6 Bugfix & Hardening. Each maps to roadmap phases.

### Safety

- [ ] **SAFE-01**: All ant_name values are escaped in grep patterns (grep -F) and JSON output (jq) across spawn.sh, swarm.sh, spawn-tree.sh, aether-utils.sh (21 locations)
- [ ] **SAFE-02**: Cross-colony information bleed eliminated — colony name extraction uses _colony_name(), LOCK_DIR passed as parameter, namespace enforcement on shared files
- [ ] **SAFE-03**: All dynamic values in JSON construction use jq escaping instead of string interpolation (14 locations across utils/)
- [ ] **SAFE-04**: Lock is released on JSON validation failure in atomic_write — callers audited for trap-based cleanup

### Infrastructure

- [ ] **INFRA-01**: Colony depth selector (light/standard/deep/full) stored in COLONY_STATE.json, gating Oracle and Scout spawns in build playbooks, default standard
- [ ] **INFRA-02**: Model routing verified end-to-end — either wired into agent spawning or dead code removed with decision documented
- [x] **INFRA-03**: YAML command generator produces both Claude and OpenCode command markdown from single YAML source files
- [ ] **INFRA-04**: XML system integrated into core commands (seal auto-exports, entomb archives XML, init can import)

### Maintenance

- [ ] **MAINT-01**: Old 2.x npm versions deprecated on registry
- [ ] **MAINT-02**: Error code reference document generated from error-handler.sh and distributed in docs/
- [ ] **MAINT-03**: Unused models[] awk array removed from aether-utils.sh

## Future Requirements

Deferred to future milestones. Tracked but not in current roadmap.

### RAG Pipeline (v2.7+)

- **RAG-01**: Chroma vector store integration with /ant:ingest command
- **RAG-02**: Hybrid retrieval (keyword + semantic search)
- **RAG-03**: Chunk re-ranking with cross-encoder model
- **RAG-04**: Context compression within colony-prime token budget
- **RAG-05**: Agent integration (Scout, Builder, Oracle use RAG)
- **RAG-06**: /ant:recall CLI command for direct RAG queries

### Agent Competition (v3.0+)

- **COMP-01**: Competition mode task definitions (collaborate/compete/cross-check)
- **COMP-02**: Arbiter agent for comparing independent proposals
- **COMP-03**: Cross-check verification mode with independent verifiers
- **COMP-04**: /ant:compete command for ad-hoc competition

### Polish (future)

- **PLSH-01**: Chamber specialization (code zones: fungus garden, nursery, refuse pile)
- **PLSH-02**: Colony Constitution (self-critique principles)
- **PLSH-03**: Worker Quality Scores (reputation system)
- **PLSH-04**: Colony Sleep (memory consolidation during pause)
- **PLSH-05**: /ant:forage explicit research command

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Multi-ant parallel execution | Brief says "DO NOT IMPLEMENT yet" — needs design discussion first |
| Per-colony file subdirectories | Overkill for single-colony-per-repo usage; namespace fix in SAFE-02 is sufficient |
| Full aether-utils.sh rewrite | Extract modules, don't rewrite from scratch (existing constraint) |
| Web/TUI dashboard | CLI tool, ASCII dashboards work in terminal (existing constraint) |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| SAFE-01 | Phase 33 | Pending |
| SAFE-02 | Phase 34 | Pending |
| SAFE-03 | Phase 33 | Pending |
| SAFE-04 | Phase 33 | Pending |
| INFRA-01 | Phase 35 | Pending |
| INFRA-02 | Phase 35 | Pending |
| INFRA-03 | Phase 36 | Complete |
| INFRA-04 | Phase 37 | Pending |
| MAINT-01 | Phase 38 | Pending |
| MAINT-02 | Phase 38 | Pending |
| MAINT-03 | Phase 38 | Pending |

**Coverage:**
- v2.6 requirements: 11 total
- Mapped to phases: 11
- Unmapped: 0

---
*Requirements defined: 2026-03-29*
*Last updated: 2026-03-29 after roadmap creation*
