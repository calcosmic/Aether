# Requirements: Aether v2.0 Reactive Event Integration

**Defined:** 2026-02-02
**Core Value:** Autonomous Emergence - Worker Ants autonomously spawn Worker Ants; Queen provides signals not commands

## v2 Requirements

Requirements for v2.0 reactive event integration. Each maps to roadmap phases.

### Event Polling

- [ ] **POLL-01**: Worker Ant calls `get_events_for_subscriber()` at execution start to check for relevant events
- [ ] **POLL-02**: Worker Ant subscribes to event topics (phase_complete, error, spawn_request, task_started, task_completed, task_failed)
- [ ] **POLL-03**: Worker Ant calls `mark_events_delivered()` after processing events to prevent reprocessing
- [ ] **POLL-04**: Worker Ant receives only events matching its subscription criteria (topic filtering)
- [ ] **POLL-05**: Different Worker Ant castes prioritize different events based on caste-specific sensitivity profiles

### Visual Indicators

- [ ] **VISUAL-01**: User sees activity state (🟢 ACTIVE, ⚪ IDLE, 🔴 ERROR, ⏳ PENDING) for each Worker Ant in status output
- [ ] **VISUAL-02**: Command output shows step progress during multi-step operations (e.g., "Step 1/3: Initializing...")
- [ ] **VISUAL-03**: `/ant:status` displays visual dashboard showing all Worker Ant activity with emoji indicators
- [ ] **VISUAL-04**: User sees pheromone signal strength visually using progress bars (e.g., `[======] 1.0` for full strength)

### E2E Testing

- [ ] **TEST-01**: E2E test guide documents init workflow with steps, expected outputs, and verification checks
- [ ] **TEST-02**: E2E test guide documents execute workflow with autonomous spawning verification
- [ ] **TEST-03**: E2E test guide documents spawning workflow with Bayesian confidence verification
- [ ] **TEST-04**: E2E test guide documents memory workflow with DAST compression verification
- [ ] **TEST-05**: E2E test guide documents voting workflow with weighted voting and Critical veto verification
- [ ] **TEST-06**: E2E test guide documents event workflow with polling, delivery, and tracking verification

### Documentation

- [ ] **DOCS-01**: All path references in `.aether/utils/` script comments are accurate
- [ ] **DOCS-02**: All docstrings in `.claude/commands/ant/` prompts have accurate path references

## v2.x Requirements

Deferred to future release. Tracked but not in current roadmap.

### Enhanced Features

- **VISUAL-10**: Real-time event streaming UI - Users see events flow in real-time
- **TEST-10**: Historical event replay - Test suite replays events from events.json for deterministic testing

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Push-based event delivery (background daemons) | Breaks Claude-native model; requires persistent processes |
| Automated LLM testing only | LLMs are non-deterministic; manual tests catch reasoning issues |
| Color-based indicators (ANSI colors) | Not universally supported; breaks in some terminals/log files |
| Complex event schemas (Avro/Protobuf) | Overkill for colony-scale; adds build step and schema registry |
| Web-based dashboard | Breaks Claude-native workflow; requires separate server |
| Real-time event streaming | Creates complexity without value for prompt-based agents |
| Command consolidation (19 → 9-11) | Planned for v3; out of scope for v2 |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| POLL-01 | Phase 1 | Pending |
| POLL-02 | Phase 1 | Pending |
| POLL-03 | Phase 1 | Pending |
| POLL-04 | Phase 1 | Pending |
| POLL-05 | Phase 1 | Pending |
| VISUAL-01 | Phase 2 | Pending |
| VISUAL-02 | Phase 2 | Pending |
| VISUAL-03 | Phase 2 | Pending |
| VISUAL-04 | Phase 2 | Pending |
| TEST-01 | Phase 3 | Pending |
| TEST-02 | Phase 3 | Pending |
| TEST-03 | Phase 3 | Pending |
| TEST-04 | Phase 3 | Pending |
| TEST-05 | Phase 3 | Pending |
| TEST-06 | Phase 3 | Pending |
| DOCS-01 | Phase 2 | Pending |
| DOCS-02 | Phase 2 | Pending |

**Coverage:**
- v2 requirements: 16 total
- Mapped to phases: 16
- Unmapped: 0 ✓

---

*Requirements defined: 2026-02-02*
*Last updated: 2026-02-02 after initial definition*
