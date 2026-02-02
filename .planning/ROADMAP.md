# Roadmap: Aether v2.0 Reactive Event Integration

## Overview

Aether v2.0 transforms the Queen Ant Colony from a prompt-based autonomous agent framework into a reactive, testable, and user-friendly multi-agent system. This milestone adds three critical capabilities: **event polling integration** (Worker Ants react asynchronously to colony events), **visual indicators** (users see colony activity at a glance), and **E2E testing** (manual test guide validates core workflows). Building on v1's proven foundation (19 commands, 10 Worker Ants, 879-line event bus), v2.0 completes the reactive architecture without introducing new dependencies.

## Milestones

- ✅ **v1.0 Queen Ant Colony** - Phases 3-10 (shipped 2026-02-02)
- 🚧 **v2.0 Reactive Event Integration** - Phases 11-13 (in progress)

## Phases

<details>
<summary>✅ v1.0 Queen Ant Colony (Phases 3-10) - SHIPPED 2026-02-02</summary>

**Full details archived in:** [milestones/v1-ROADMAP.md](.planning/milestones/v1-ROADMAP.md)

**Summary:**
- 8 phases (3-10), 44 plans, 156 must-haves verified
- Autonomous spawning with Bayesian meta-learning
- Pheromone communication with time-based decay
- Triple-layer memory (Working → Short-term → Long-term)
- Multi-perspective verification with weighted voting
- Event-driven coordination with pub/sub event bus
- Production-ready with comprehensive testing

</details>

### 🚧 v2.0 Reactive Event Integration (In Progress)

**Milestone Goal:** Enable Worker Ants to react asynchronously to colony events through proactive event polling, with enhanced visual feedback and comprehensive testing.

- [ ] **Phase 11: Event Polling Integration** - Worker Ants react to colony events via polling
- [ ] **Phase 12: Visual Indicators & Documentation** - CLI visual feedback and path cleanup
- [ ] **Phase 13: E2E Testing** - Manual test guide for core workflows

## Phase Details

### Phase 11: Event Polling Integration

**Goal**: Worker Ants detect and react to colony events by polling the event bus at execution boundaries, enabling asynchronous coordination without persistent processes.

**Depends on**: Phase 10 (Colony Maturity - v1.0 complete)

**Requirements**: POLL-01, POLL-02, POLL-03, POLL-04, POLL-05

**Success Criteria** (what must be TRUE):
1. Worker Ant calls `get_events_for_subscriber()` at execution start and checks for relevant events
2. Worker Ant subscribes to event topics (phase_complete, error, spawn_request, task_started, task_completed, task_failed)
3. Worker Ant calls `mark_events_delivered()` after processing events to prevent reprocessing
4. Worker Ant receives only events matching its subscription criteria (topic filtering works)
5. Different Worker Ant castes prioritize different events based on caste-specific sensitivity profiles

**Plans**: 3 plans in 2 waves

- [ ] [11-01-PLAN.md](.planning/phases/11-event-polling-integration/11-01-PLAN.md) — Add event polling to 6 base caste Worker Ants (colonizer, route-setter, builder, watcher, scout, architect)
- [ ] [11-02-PLAN.md](.planning/phases/11-event-polling-integration/11-02-PLAN.md) — Add event polling to 4 specialist Watchers (security, performance, quality, test-coverage)
- [ ] [11-03-PLAN.md](.planning/phases/11-event-polling-integration/11-03-PLAN.md) — Create and run integration test suite for event polling

### Phase 12: Visual Indicators & Documentation

**Goal**: Users see colony activity at a glance through emoji-based status indicators, progress bars, and structured output, with all documentation path references corrected.

**Depends on**: Phase 11

**Requirements**: VISUAL-01, VISUAL-02, VISUAL-03, VISUAL-04, DOCS-01, DOCS-02

**Success Criteria** (what must be TRUE):
1. User sees activity state (🟢 ACTIVE, ⚪ IDLE, 🔴 ERROR, ⏳ PENDING) for each Worker Ant in status output
2. Command output shows step progress during multi-step operations (e.g., "Step 1/3: Initializing...")
3. `/ant:status` displays visual dashboard showing all Worker Ant activity with emoji indicators
4. User sees pheromone signal strength visually using progress bars (e.g., `[======] 1.0` for full strength)
5. All path references in `.aether/utils/` script comments are accurate
6. All docstrings in `.claude/commands/ant/` prompts have accurate path references

**Plans**: TBD

### Phase 13: E2E Testing

**Goal**: Comprehensive manual test guide documents all core workflows with steps, expected outputs, and verification checks for validating colony behavior.

**Depends on**: Phase 11, Phase 12

**Requirements**: TEST-01, TEST-02, TEST-03, TEST-04, TEST-05, TEST-06

**Success Criteria** (what must be TRUE):
1. E2E test guide documents init workflow with steps, expected outputs, and verification checks
2. E2E test guide documents execute workflow with autonomous spawning verification
3. E2E test guide documents spawning workflow with Bayesian confidence verification
4. E2E test guide documents memory workflow with DAST compression verification
5. E2E test guide documents voting workflow with weighted voting and Critical veto verification
6. E2E test guide documents event workflow with polling, delivery, and tracking verification

**Plans**: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 11 → 12 → 13

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 3-10 | v1.0 | 44/44 | Complete | 2026-02-02 |
| 11. Event Polling Integration | v2.0 | 3/3 | Complete | 2026-02-02 |
| 12. Visual Indicators & Documentation | v2.0 | 0/TBD | Not started | - |
| 13. E2E Testing | v2.0 | 0/TBD | Not started | - |

---

*Aether v2: Queen Ant Colony - Autonomous Emergence in Claude-Native Form*
