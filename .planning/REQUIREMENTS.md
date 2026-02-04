# Requirements: Aether v4.3

**Defined:** 2026-02-04
**Core Value:** Autonomous Emergence — Worker Ants autonomously spawn Worker Ants; Queen provides signals not commands

## v1 Requirements

Requirements for v4.3 Live Visibility & Auto-Learning. Each maps to roadmap phases.

### Live Visibility

- [x] **VIS-01**: Workers write structured progress lines to `.aether/data/activity.log` as they work (task start, file create/modify, task complete, spawn events)
- [x] **VIS-02**: build.md orchestrates worker spawns sequentially through the Queen rather than delegating all spawning to the Phase Lead, so the user sees incremental results between each worker
- [x] **VIS-03**: Queen displays each worker's activity log output and result summary after each worker returns, before spawning the next

### Auto-Learning

- [ ] **LEARN-01**: build.md Step 7 automatically extracts phase learnings from completed work (errors, events, task outcomes) and writes to memory.json, using the same extraction logic currently in continue.md
- [ ] **LEARN-02**: build.md Step 7 auto-emits FEEDBACK pheromone summarizing what worked and what didn't, validated via pheromone-validate before writing
- [ ] **LEARN-03**: continue.md skips learning extraction if learnings were already extracted by build (detects via event log or memory.json timestamp)

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Advanced Visibility

- **VIS-04**: `/ant:status` shows real-time worker activity when polled from a second terminal during execution
- **VIS-05**: Activity log rotation and cleanup for long-running colonies

## Out of Scope

| Feature | Reason |
|---------|--------|
| Streaming subagent output | Claude Code Task tool doesn't support streaming — architectural constraint |
| Real-time terminal dashboard | Requires persistent daemon process — against Claude-native architecture |
| New commands | v4.0 decision: enrich existing 12 commands, don't add new ones |
| Separate activity viewer command | Activity visibility integrated into build.md flow |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| VIS-01 | Phase 25 | Complete |
| VIS-02 | Phase 25 | Complete |
| VIS-03 | Phase 25 | Complete |
| LEARN-01 | Phase 26 | Pending |
| LEARN-02 | Phase 26 | Pending |
| LEARN-03 | Phase 26 | Pending |

**Coverage:**
- v1 requirements: 6 total
- Mapped to phases: 6
- Unmapped: 0 ✓

---
*Requirements defined: 2026-02-04*
*Last updated: 2026-02-04 after Phase 25 completion*
