# Requirements: Aether v3.0 Restore the Soul

**Defined:** 2026-02-03
**Core Value:** Autonomous Emergence — Worker Ants autonomously spawn Worker Ants; Queen provides signals not commands

## v3.0 Requirements

Requirements for restoring the sophistication, visual identity, and depth lost during the v3-rebuild. Each maps to roadmap phases.

### Visual Identity

- [x] **VIS-01**: Commands display box-drawing headers for major sections
- [x] **VIS-02**: Multi-step commands show step progress with [✓]/[→]/[ ] indicators
- [x] **VIS-03**: Pheromone display includes computed decay strength bars
- [x] **VIS-04**: Worker activity grouped by status with emoji indicators

### Specialist Watchers

- [ ] **WATCH-01**: watcher-ant.md contains 4 specialist modes (security, performance, quality, test-coverage)
- [ ] **WATCH-02**: Mode activation triggered by pheromone context
- [ ] **WATCH-03**: Each mode has severity rubric (Critical/High/Medium/Low)
- [ ] **WATCH-04**: Each mode has specific detection pattern checklist

### Worker Spec Depth

- [ ] **SPEC-01**: Each worker spec includes pheromone math examples (sensitivity × strength = effective signal)
- [ ] **SPEC-02**: Each worker spec includes combination effects for conflicting signals
- [ ] **SPEC-03**: Each worker spec includes feedback interpretation guide
- [ ] **SPEC-04**: Each worker spec includes event awareness at startup
- [ ] **SPEC-05**: Each worker spec includes spawning scenario with full Task tool prompt example

### Error Tracking

- [ ] **ERR-01**: errors.json stores error records with id, category, severity, description, root_cause, phase, timestamp
- [ ] **ERR-02**: build.md logs errors to errors.json when phase encounters failures
- [ ] **ERR-03**: Pattern flagging triggers after 3 occurrences of same error category
- [ ] **ERR-04**: status.md displays recent errors and flagged patterns

### Colony Memory

- [ ] **MEM-01**: memory.json stores phase_learnings, decisions, and patterns arrays
- [ ] **MEM-02**: continue.md extracts learnings at phase boundaries before advancing
- [ ] **MEM-03**: Commands log significant decisions to memory.json
- [ ] **MEM-04**: Workers read relevant memory entries at startup for context

### Event Awareness

- [ ] **EVT-01**: events.json stores event records with id, type, source, content, timestamp
- [ ] **EVT-02**: Commands write events on state changes (init, phase start/complete, errors, spawns)
- [ ] **EVT-03**: Workers read events.json at startup and filter by timestamp for recent events
- [ ] **EVT-04**: init.md creates all JSON state files (errors.json, memory.json, events.json)

### Enhanced Dashboard

- [ ] **DASH-01**: status.md shows full colony health with workers, pheromones, errors, memory, events
- [ ] **DASH-02**: Pheromone section shows each active signal with computed decay bar
- [ ] **DASH-03**: Error section shows recent errors and flagged patterns from errors.json
- [ ] **DASH-04**: Memory section shows recent learnings from memory.json

### Phase Review

- [ ] **REV-01**: continue.md shows phase completion summary before advancing
- [ ] **REV-02**: Phase review shows tasks completed, key decisions, errors encountered
- [ ] **REV-03**: Learning extraction stores insights to memory.json before phase transition

### Spawn Tracking

- [ ] **SPAWN-01**: COLONY_STATE.json includes spawn_outcomes field per caste
- [ ] **SPAWN-02**: build.md records spawn events when Phase Lead is spawned
- [ ] **SPAWN-03**: continue.md records spawn success/failure on phase completion
- [ ] **SPAWN-04**: Workers check spawn history confidence before spawning (alpha / (alpha + beta))

## v3.x Requirements

Deferred to future release. Tracked but not in current roadmap.

### Advanced Features

- **ADV-01**: Real-time event streaming UI — Users see events flow in real-time
- **ADV-02**: Web-based colony dashboard — Visual GUI for colony monitoring
- **ADV-03**: Automated LLM behavior testing — Programmatic LLM validation framework
- **ADV-04**: Event replay for time-travel debugging — Full colony state snapshotting

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Python runtime restoration | Claude-native model replaces Python; commands use Read/Write/Task tools |
| Bash event bus restoration | 890-line event-bus.sh replaced by simple events.json log |
| Bash utility scripts | Memory-search.sh, spawn-tracker.sh etc. replaced by JSON state |
| Separate specialist watcher files | 4 modes folded into watcher-ant.md (per constraint: no new commands) |
| New commands beyond existing 12 | Restore by enriching existing commands, not adding new ones |
| External dependencies | No vector DBs, embedding services, or external tools |
| Worker specs exceeding ~200 lines | Keep deep but focused; trim not cut |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| VIS-01 | Phase 14 | Complete |
| VIS-02 | Phase 14 | Complete |
| VIS-03 | Phase 14 | Complete |
| VIS-04 | Phase 14 | Complete |
| ERR-01 | Phase 15 | Pending |
| ERR-02 | Phase 15 | Pending |
| ERR-03 | Phase 15 | Pending |
| ERR-04 | Phase 15 | Pending |
| MEM-01 | Phase 15 | Pending |
| MEM-02 | Phase 15 | Pending |
| MEM-03 | Phase 15 | Pending |
| MEM-04 | Phase 15 | Pending |
| EVT-01 | Phase 15 | Pending |
| EVT-02 | Phase 15 | Pending |
| EVT-03 | Phase 15 | Pending |
| EVT-04 | Phase 15 | Pending |
| WATCH-01 | Phase 16 | Pending |
| WATCH-02 | Phase 16 | Pending |
| WATCH-03 | Phase 16 | Pending |
| WATCH-04 | Phase 16 | Pending |
| SPEC-01 | Phase 16 | Pending |
| SPEC-02 | Phase 16 | Pending |
| SPEC-03 | Phase 16 | Pending |
| SPEC-04 | Phase 16 | Pending |
| SPEC-05 | Phase 16 | Pending |
| DASH-01 | Phase 17 | Pending |
| DASH-02 | Phase 17 | Pending |
| DASH-03 | Phase 17 | Pending |
| DASH-04 | Phase 17 | Pending |
| REV-01 | Phase 17 | Pending |
| REV-02 | Phase 17 | Pending |
| REV-03 | Phase 17 | Pending |
| SPAWN-01 | Phase 17 | Pending |
| SPAWN-02 | Phase 17 | Pending |
| SPAWN-03 | Phase 17 | Pending |
| SPAWN-04 | Phase 17 | Pending |

**Coverage:**
- v3.0 requirements: 36 total
- Mapped to phases: 36
- Unmapped: 0

---

*Requirements defined: 2026-02-03*
*Last updated: 2026-02-03 after Phase 14 completion*
