# Requirements: Aether v2.0 Reactive Event Integration

**Defined:** 2026-02-02
**Core Value:** Autonomous Emergence - Worker Ants autonomously spawn Worker Ants; Queen provides signals not commands

## v2 Requirements

Requirements for v2.0 reactive event integration. Each maps to roadmap phases.

### Event Polling

- [x] **POLL-01**: Worker Ant calls `get_events_for_subscriber()` at execution start to check for relevant events
- [x] **POLL-02**: Worker Ant subscribes to event topics (phase_complete, error, spawn_request, task_started, task_completed, task_failed)
- [x] **POLL-03**: Worker Ant calls `mark_events_delivered()` after processing events to prevent reprocessing
- [x] **POLL-04**: Worker Ant receives only events matching its subscription criteria (topic filtering)
- [x] **POLL-05**: Different Worker Ant castes prioritize different events based on caste-specific sensitivity profiles

### Visual Indicators

- [x] **VISUAL-01**: User sees activity state (🟢 ACTIVE, ⚪ IDLE, 🔴 ERROR, ⏳ PENDING) for each Worker Ant in status output
- [x] **VISUAL-02**: Command output shows step progress during multi-step operations (e.g., "Step 1/3: Initializing...")
- [x] **VISUAL-03**: `/ant:status` displays visual dashboard showing all Worker Ant activity with emoji indicators
- [x] **VISUAL-04**: User sees pheromone signal strength visually using progress bars (e.g., `[======] 1.0` for full strength)

### E2E Testing

- [x] **TEST-01**: E2E test guide documents init workflow with steps, expected outputs, and verification checks
- [x] **TEST-02**: E2E test guide documents execute workflow with autonomous spawning verification
- [x] **TEST-03**: E2E test guide documents spawning workflow with Bayesian confidence verification
- [x] **TEST-04**: E2E test guide documents memory workflow with DAST compression verification
- [x] **TEST-05**: E2E test guide documents voting workflow with weighted voting and Critical veto verification
- [x] **TEST-06**: E2E test guide documents event workflow with polling, delivery, and tracking verification

### Documentation

- [x] **DOCS-01**: All path references in `.aether/utils/` script comments are accurate
- [x] **DOCS-02**: All docstrings in `.claude/commands/ant/` prompts have accurate path references

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
| POLL-01 | Phase 11 | Complete |
| POLL-02 | Phase 11 | Complete |
| POLL-03 | Phase 11 | Complete |
| POLL-04 | Phase 11 | Complete |
| POLL-05 | Phase 11 | Complete |
| VISUAL-01 | Phase 12 | Complete |
| VISUAL-02 | Phase 12 | Complete |
| VISUAL-03 | Phase 12 | Complete |
| VISUAL-04 | Phase 12 | Complete |
| DOCS-01 | Phase 12 | Complete |
| DOCS-02 | Phase 12 | Complete |
| TEST-01 | Phase 13 | Complete |
| TEST-02 | Phase 13 | Complete |
| TEST-03 | Phase 13 | Complete |
| TEST-04 | Phase 13 | Complete |
| TEST-05 | Phase 13 | Complete |
| TEST-06 | Phase 13 | Complete |

**Coverage:**
- v2 requirements: 16 total
- Mapped to phases: 16
- Unmapped: 0 ✓

---

*Requirements defined: 2026-02-02*
*Last updated: 2026-02-02 after Phase 13 completion - v2.0 SHIPPED*
