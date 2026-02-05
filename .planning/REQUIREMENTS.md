# Requirements: Aether v4.4

**Defined:** 2026-02-04
**Core Value:** Stigmergic Emergence — Worker Ants detect capability gaps and spawn specialists through pheromone-guided coordination

## v1 Requirements

Requirements for v4.4 Colony Hardening & Real-World Readiness. Each maps to roadmap phases.

### Bug Fixes

- [x] **BUG-01**: Pheromone decay math produces correct decreasing strength over time (FOCUS strength no longer grows)
- [x] **BUG-02**: Activity log appends across phases instead of overwriting (Phases 1-N entries preserved)
- [x] **BUG-03**: Errors logged to errors.json include phase attribution field
- [x] **BUG-04**: Decisions made during execution phases are recorded in memory.json decisions array

### Critical UX

- [x] **UX-01**: Every command that completes meaningful work prompts user with "safe to /clear" after verifying state persistence
- [x] **UX-02**: Auto-continue mode (`/ant:continue --all`) runs remaining phases without manual approval at each boundary

### Colony Intelligence

- [x] **INT-01**: Colonize command spawns multiple ants that review codebase independently and synthesize findings
- [x] **INT-02**: Tasks touching the same file are assigned to the same worker by Phase Lead during planning
- [x] **INT-03**: Phase Lead assigns independent tasks to parallel waves more aggressively
- [x] **INT-04**: Phase Lead auto-approves plans for phases below a complexity threshold
- [x] **INT-05**: Watcher scoring rubric produces meaningfully varied scores (not flat 8/10)
- [x] **INT-06**: Tech debt report generated at project completion aggregating cross-phase persistent issues
- [x] **INT-07**: Colony adapts overhead to project complexity (LIGHTWEIGHT/STANDARD/FULL mode)

### Automation

- [x] **AUTO-01**: Reviewer ant auto-spawns after builder waves (advisory only, severity-gated, max 2 iterations)
- [x] **AUTO-02**: Debugger ant auto-spawns on test failure
- [x] **AUTO-03**: Pheromone recommendations surfaced to user after builds (e.g. "Recommended: /ant:focus ...")
- [x] **AUTO-04**: Animated build indicators with ANSI progress bars and caste-colored output
- [x] **AUTO-05**: Colonizer command has visual output with emojis and progress markers

### Architecture

- [x] **ARCH-01**: Two-tier learning system — project-local (memory.json) + global (~/.aether/learnings.json) with manual promotion
- [x] **ARCH-02**: Spawn tree engine — workers signal sub-spawn needs, Queen fulfills (Queen-mediated recursive delegation with depth limit 2)
- [x] **ARCH-03**: Adaptive complexity mode set at colonization time via mode field in COLONY_STATE.json

### Flow & Documentation

- [x] **FLOW-01**: Pheromone-first flow — colonize suggests pheromone injection before planning
- [x] **FLOW-02**: Organizer/archivist ant reports stale files, dead code, orphaned configs (report-only, conservative)
- [x] **FLOW-03**: Pheromone user documentation — when/why to use FOCUS, REDIRECT, FEEDBACK with practical scenarios

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Distribution

- **DIST-01**: Installable NPM package for global install
- **DIST-02**: Deployment model for external repos (.aether/ bootstrapping)
- **DIST-03**: Auto-update mechanism for published changes

### Advanced Colony

- **ADV-01**: Unlimited recursive delegation (beyond depth 2)
- **ADV-02**: Agent-to-agent direct messaging
- **ADV-03**: Auto-promote learnings to global tier without user approval

## Out of Scope

| Feature | Reason |
|---------|--------|
| NPM packaging/distribution | Deferred until core system stabilizes (field note 16) |
| External repo deployment model | Deferred until core system stabilizes (field note 6) |
| Web dashboard / GUI | Breaks CLI-only constraint |
| Vector DB / embeddings for memory | Overkill for JSON-scale data |
| Unlimited recursive spawning | Anti-feature — context degrades at each level (field note 32) |
| Agent-to-agent messaging | Destroys stigmergic coordination model |
| Auto-promotion of global learnings | Risk of stale cross-project knowledge (CP-5) |
| New commands beyond existing 12 | Enrichment over proliferation constraint |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| BUG-01 | Phase 27 | Complete |
| BUG-02 | Phase 27 | Complete |
| BUG-03 | Phase 27 | Complete |
| BUG-04 | Phase 27 | Complete |
| INT-02 | Phase 27 | Complete |
| UX-01 | Phase 28 | Complete |
| UX-02 | Phase 28 | Complete |
| FLOW-01 | Phase 28 | Complete |
| INT-01 | Phase 29 | Complete |
| INT-03 | Phase 29 | Complete |
| INT-04 | Phase 29 | Complete |
| INT-05 | Phase 29 | Complete |
| INT-07 | Phase 29 | Complete |
| ARCH-03 | Phase 29 | Complete |
| AUTO-01 | Phase 30 | Complete |
| AUTO-02 | Phase 30 | Complete |
| AUTO-03 | Phase 30 | Complete |
| AUTO-04 | Phase 30 | Complete |
| AUTO-05 | Phase 30 | Complete |
| INT-06 | Phase 30 | Complete |
| ARCH-01 | Phase 31 | Complete |
| ARCH-02 | Phase 31 | Complete |
| FLOW-02 | Phase 32 | Complete |
| FLOW-03 | Phase 32 | Complete |

**Coverage:**
- v1 requirements: 24 total
- Mapped to phases: 24
- Unmapped: 0

---
*Requirements defined: 2026-02-04*
*Last updated: 2026-02-05 — Phase 32 requirements (FLOW-02, FLOW-03) + Phase 31 (ARCH-01, ARCH-02) marked Complete — all 24/24 requirements complete*
