# Requirements: Aether Repair & Stabilization

**Defined:** 2026-02-17
**Core Value:** Context preservation, clear workflow guidance, self-improving colony

## v1 Requirements

### Command Infrastructure

- [x] **CMD-01**: /ant:lay-eggs starts new colony with pheromone preservation
- [x] **CMD-02**: /ant:init initializes after lay-eggs
- [x] **CMD-03**: /ant:colonize analyzes existing codebase
- [x] **CMD-04**: /ant:plan generates project plan
- [x] **CMD-05**: /ant:build executes phase with worker spawning
- [x] **CMD-06**: /ant:continue verifies, extracts learnings, advances phase
- [x] **CMD-07**: /ant:status shows colony dashboard
- [x] **CMD-08**: All commands find correct files (no hallucinations)

### Visual Experience

- [x] **VIS-01**: Swarm display shows ants working (not bash text scroll)
- [x] **VIS-02**: Emoji caste identity visible in output
- [x] **VIS-03**: Colors for different castes
- [x] **VIS-04**: Progress indication during builds
- [x] **VIS-05**: Stage banners use ant-themed names (DIGESTING, EXCAVATING, etc.)
- [x] **VIS-06**: GSD-style formatting for phase transitions

### Context Rot Prevention

- [x] **CTX-01**: Session state persists across /clear
- [x] **CTX-02**: Clear "next command" guidance at phase boundaries
- [x] **CTX-03**: Context document tells next session what was happening

### State Integrity

- [x] **STA-01**: COLONY_STATE.json updates correctly on all operations
- [x] **STA-02**: No file path hallucinations (commands find right files)
- [x] **STA-03**: Files created in correct repositories

### Pheromone System

- [x] **PHER-01**: FOCUS signal attracts attention to areas
- [x] **PHER-02**: REDIRECT signal warns away from patterns
- [x] **PHER-03**: FEEDBACK signal calibrates behavior
- [x] **PHER-04**: Auto-injection of learned patterns into new work
- [x] **PHER-05**: Instincts applied to builders/watchers

### Colony Lifecycle

- [x] **LIF-01**: /ant:seal creates Crowned Anthill milestone
- [x] **LIF-02**: /ant:entomb archives colony to chambers
- [x] **LIF-03**: /ant:tunnels browses archived colonies

### Advanced Workers

- [x] **ADV-01**: /ant:oracle performs deep research (RALF loop)
- [x] **ADV-02**: /ant:chaos performs resilience testing
- [x] **ADV-03**: /ant:archaeology analyzes git history
- [x] **ADV-04**: /ant:dream philosophical wanderer writes wisdom
- [x] **ADV-05**: /ant:interpret validates dreams against reality

### XML Integration

- [x] **XML-01**: Pheromones stored/retrieved via XML format
- [x] **XML-02**: Wisdom exchange uses XML structure
- [x] **XML-03**: Registry uses XML for cross-colony communication

### Session Management

- [x] **SES-01**: /ant:pause-colony saves state and creates handoff
- [x] **SES-02**: /ant:resume-colony restores full context
- [x] **SES-03**: /ant:watch shows live colony visibility

### Colony Documentation

- [x] **DOC-01**: Phase learnings extracted and documented (ant-themed)
- [x] **DOC-02**: Colony memories stored with ant naming (pheromones.md)
- [x] **DOC-03**: Progress tracked with ant metaphors (nursery, chambers)
- [x] **DOC-04**: Handoff documents use ant themes

### Error Handling

- [x] **ERR-01**: No 401 authentication errors during normal operation
- [x] **ERR-02**: Agents stop spawning (no infinite loops)
- [x] **ERR-03**: Clear error messages when things fail

## v2 Requirements

(None yet — all current work is v1)

## Out of Scope

(None — all features stay in scope, repair what exists)

## Traceability

| Requirement | Phase | Status |
|------------|-------|--------|
| CMD-01 through CMD-08 | Phase 9 | VERIFIED PASS |
| VIS-01 through VIS-06 | Phase 9 | VERIFIED PASS |
| CTX-01 through CTX-03 | Phase 9 | VERIFIED PASS |
| STA-01 through STA-03 | Phase 9 | VERIFIED PASS |
| PHER-01 through PHER-05 | Phase 9 | VERIFIED PASS |
| LIF-01 through LIF-03 | Phase 6 + Phase 9 | VERIFIED PASS |
| ADV-01 through ADV-05 | Phase 9 | VERIFIED PASS |
| XML-01 through XML-03 | Phase 9 | VERIFIED PASS |
| SES-01 through SES-03 | Phase 9 | VERIFIED PASS |
| DOC-01 through DOC-04 | Phase 9 | VERIFIED PASS |
| ERR-01 through ERR-03 | Phase 9 | VERIFIED PASS |

---

*Requirements defined: 2026-02-17*
*Last updated: 2026-02-18 — Phase 9 verification complete — 46/46 requirements PASS*
