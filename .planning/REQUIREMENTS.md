# Requirements: Aether Repair & Stabilization

**Defined:** 2026-02-17
**Core Value:** Context preservation, clear workflow guidance, self-improving colony

## v1 Requirements

### Command Infrastructure

- [ ] **CMD-01**: /ant:lay-eggs starts new colony with pheromone preservation
- [ ] **CMD-02**: /ant:init initializes after lay-eggs
- [ ] **CMD-03**: /ant:colonize analyzes existing codebase
- [ ] **CMD-04**: /ant:plan generates project plan
- [ ] **CMD-05**: /ant:build executes phase with worker spawning
- [ ] **CMD-06**: /ant:continue verifies, extracts learnings, advances phase
- [ ] **CMD-07**: /ant:status shows colony dashboard
- [ ] **CMD-08**: All commands find correct files (no hallucinations)

### Visual Experience

- [ ] **VIS-01**: Swarm display shows ants working (not bash text scroll)
- [ ] **VIS-02**: Emoji caste identity visible in output
- [ ] **VIS-03**: Colors for different castes
- [ ] **VIS-04**: Progress indication during builds
- [ ] **VIS-05**: Stage banners use ant-themed names (DIGESTING, EXCAVATING, etc.)
- [ ] **VIS-06**: GSD-style formatting for phase transitions

### Context Rot Prevention

- [x] **CTX-01**: Session state persists across /clear
- [ ] **CTX-02**: Clear "next command" guidance at phase boundaries
- [x] **CTX-03**: Context document tells next session what was happening

### State Integrity

- [ ] **STA-01**: COLONY_STATE.json updates correctly on all operations
- [ ] **STA-02**: No file path hallucinations (commands find right files)
- [ ] **STA-03**: Files created in correct repositories

### Pheromone System

- [ ] **PHER-01**: FOCUS signal attracts attention to areas
- [ ] **PHER-02**: REDIRECT signal warns away from patterns
- [ ] **PHER-03**: FEEDBACK signal calibrates behavior
- [ ] **PHER-04**: Auto-injection of learned patterns into new work
- [ ] **PHER-05**: Instincts applied to builders/watchers

### Colony Lifecycle

- [ ] **LIF-01**: /ant:seal creates Crowned Anthill milestone
- [ ] **LIF-02**: /ant:entomb archives colony to chambers
- [ ] **LIF-03**: /ant:tunnels browses archived colonies

### Advanced Workers

- [ ] **ADV-01**: /ant:oracle performs deep research (RALF loop)
- [ ] **ADV-02**: /ant:chaos performs resilience testing
- [ ] **ADV-03**: /ant:archaeology analyzes git history
- [ ] **ADV-04**: /ant:dream philosophical wanderer writes wisdom
- [ ] **ADV-05**: /ant:interpret validates dreams against reality

### XML Integration

- [ ] **XML-01**: Pheromones stored/retrieved via XML format
- [ ] **XML-02**: Wisdom exchange uses XML structure
- [ ] **XML-03**: Registry uses XML for cross-colony communication

### Session Management

- [ ] **SES-01**: /ant:pause-colony saves state and creates handoff
- [ ] **SES-02**: /ant:resume-colony restores full context
- [ ] **SES-03**: /ant:watch shows live colony visibility

### Colony Documentation

- [ ] **DOC-01**: Phase learnings extracted and documented (ant-themed)
- [ ] **DOC-02**: Colony memories stored with ant naming (pheromones.md)
- [ ] **DOC-03**: Progress tracked with ant metaphors (nursery, chambers)
- [ ] **DOC-04**: Handoff documents use ant themes

### Error Handling

- [ ] **ERR-01**: No 401 authentication errors during normal operation
- [ ] **ERR-02**: Agents stop spawning (no infinite loops)
- [ ] **ERR-03**: Clear error messages when things fail

## v2 Requirements

(None yet — all current work is v1)

## Out of Scope

(None — all features stay in scope, repair what exists)

## Traceability

| Requirement | Phase | Status |
|------------|-------|--------|
| CMD-01 through CMD-08 | TBD | Pending |
| VIS-01 through VIS-06 | TBD | Pending |
| CTX-01 through CTX-03 | TBD | Pending |
| STA-01 through STA-03 | TBD | Pending |
| PHER-01 through PHER-05 | TBD | Pending |
| LIF-01 through LIF-03 | TBD | Pending |
| ADV-01 through ADV-05 | TBD | Pending |
| XML-01 through XML-03 | TBD | Pending |
| SES-01 through SES-03 | TBD | Pending |
| DOC-01 through DOC-04 | TBD | Pending |
| ERR-01 through ERR-03 | TBD | Pending |

---

*Requirements defined: 2026-02-17*
*Last updated: 2026-02-17 after initial assessment*
