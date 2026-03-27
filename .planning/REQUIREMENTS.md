# Requirements: Aether v2.4 Living Wisdom

**Defined:** 2026-03-27
**Core Value:** Reliably interpret user requests, decompose into work, verify outputs, and ship correct work with minimal back-and-forth.

## v2.4 Requirements

Requirements for Living Wisdom milestone. Each maps to roadmap phases.

### Agent Definitions

- [x] **AGNT-01**: Oracle has a dedicated agent definition file (.claude/agents/ant/aether-oracle.md) with opus model slot routing
- [x] **AGNT-02**: Architect has a dedicated agent definition file (.claude/agents/ant/aether-architect.md) with opus model slot routing
- [x] **AGNT-03**: Oracle and Architect agent files are mirrored to OpenCode (.opencode/agents/) and packaging (.aether/agents-claude/)
- [x] **AGNT-04**: Oracle agent is spawnable by Queen during builds (not just via /ant:oracle command)
- [x] **AGNT-05**: Architect agent has design-create mode (can write architecture docs, not just read-only)

### Wisdom Pipeline

- [ ] **PIPE-01**: queen-write-learnings is called during /ant:continue, writing phase learnings to QUEEN.md
- [ ] **PIPE-02**: hive-promote is called during /ant:continue, promoting instincts to hive brain
- [ ] **PIPE-03**: Builder learning extraction has a deterministic fallback (git-diff-based) when AI agents skip learning output
- [ ] **PIPE-04**: Users see visible feedback when wisdom is written (e.g., "3 learnings recorded, 1 instinct promoted" in continue output)

### Quality & Validation

- [ ] **VAL-01**: End-to-end integration test verifies: build → continue → QUEEN.md populated → hive brain populated
- [ ] **VAL-02**: Instinct deduplication uses content normalization (not just SHA-256 exact match) so semantically similar instincts consolidate

## v3 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Observability

- **OBS-01**: /ant:wisdom command to view accumulated wisdom status
- **OBS-02**: Wisdom dashboard in /ant:status showing QUEEN.md entries, hive brain count, instinct count

### Advanced Wisdom

- **WIS-01**: instinct-apply records when instincts are used in practice (success/failure feedback loop)
- **WIS-02**: QUEEN.md v1/v2 auto-migration called in build flow (queen-migrate)
- **WIS-03**: Hive promotion also fires during /ant:entomb (not just /ant:seal)

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| /ant:wisdom command | Nice-to-have observability, not core to making wisdom work |
| instinct-apply feedback loop | Requires real colony runs to validate, defer until pipeline is wired |
| QUEEN.md format migration | Auto-migration exists but is edge-case, handle manually if needed |
| hive-promote in entomb | Users who want cross-colony wisdom can run /ant:seal; entomb is archive-only |
| Builder learning quality validation | LLM behavior is hard to test deterministically; pipeline wiring is the priority |
| Multi-repo wisdom coordination | Architecture-level change, defer to future milestone |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| AGNT-01 (Oracle agent file) | 25 | Pending |
| AGNT-02 (Architect agent file) | 25 | Pending |
| AGNT-03 (Agent mirrors) | 25 | Pending |
| AGNT-04 (Oracle spawnable by Queen) | 25 | Pending |
| AGNT-05 (Architect design-create mode) | 25 | Pending |
| PIPE-01 (queen-write-learnings in continue) | 26 | Pending |
| PIPE-02 (hive-promote in continue) | 26 | Pending |
| PIPE-04 (Visible wisdom feedback) | 26 | Pending |
| PIPE-03 (Deterministic fallback) | 27 | Pending |
| VAL-02 (Content normalization dedup) | 27 | Pending |
| VAL-01 (E2E integration test) | 28 | Pending |

**Coverage:**
- v2.4 requirements: 11 total
- Mapped to phases: 11
- Unmapped: 0

---
*Requirements defined: 2026-03-27*
*Last updated: 2026-03-27 after roadmap creation*
