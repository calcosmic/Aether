# Roadmap: Aether

## Milestones

- **v1.0 MVP** - Phases 1-6 (shipped)
- **v1.1 Trusted Context** - Phases 7-11 (shipped)
- **v1.2 Live Dispatch Truth and Recovery** - Phases 12-16 (shipped)
- **v1.3 Visual Truth and Core Hardening** - Phases 17-24 (shipped 2026-04-21)
- **v1.4 Self-Healing Colony** - Phases 25-30 (completed 2026-04-21)
- **v1.5 Runtime Truth Recovery** - Phases 31-36 (in progress)

## Phases

<details>
<summary>v1.0 MVP (Phases 1-6) -- SHIPPED</summary>

- Phase 1: Housekeeping and Foundation
- Phase 2: Colony Scope System
- Phase 3: Restore Build Ceremony
- Phase 4: Restore Continue Ceremony
- Phase 5: Living Watch and Status Surfaces
- Phase 6: Pheromone Visibility and Steering

</details>

<details>
<summary>v1.1 Trusted Context (Phases 7-11) -- SHIPPED</summary>

- Phase 7: Context Ledger and Skill Routing Foundation
- Phase 8: Prompt Integrity and Trust Boundaries
- Phase 9: Trust-Weighted Context Assembly
- Phase 10: Curation Spine and Structural Learning
- Phase 11: Competitive Proof Surfaces and Evaluation

</details>

<details>
<summary>v1.2 Live Dispatch Truth and Recovery (Phases 12-16) -- SHIPPED</summary>

- Phase 12: Dispatch Truth Model and Run Scoping
- Phase 13: Live Workflow Visibility Across Colonize, Plan, and Build
- Phase 14: Worker Execution Robustness and Honest Activity Tracking
- Phase 15: Verification-Led Continue and Partial Success
- Phase 16: Recovery, Reconciliation, and Runtime UX Finalization

</details>

<details>
<summary>v1.3 Visual Truth and Core Hardening (Phases 17-24) -- SHIPPED 2026-04-21</summary>

- Phase 17: Slash Command Format Audit
- Phase 18: Visual UX Restoration -- Caste Identity and Spawn Lists
- Phase 19: Visual UX Restoration -- Stage Separators and Ceremony
- Phase 20: Visual UX Restoration -- Emoji Consistency
- Phase 21: Codex CLI Visual Parity
- Phase 22: Core Path Hardening
- Phase 23: Recovery and Continuity
- Phase 24: Full Instrumentation -- Trace Logging

</details>

<details>
<summary>v1.4 Self-Healing Colony (Phases 25-30) -- COMPLETED 2026-04-21</summary>

- Phase 25: Medic Ant Core -- Health diagnosis command, colony data scanner
- Phase 26: Auto-Repair -- Fix common colony data issues with `--fix` flag
- Phase 27: Medic Skill -- Healthy state specification skill file
- Phase 28: Ceremony Integrity -- Verify wrapper/runtime parity
- Phase 29: Trace Diagnostics -- Remote debugging via trace export analysis
- Phase 30: Medic Worker Integration -- Caste integration, auto-spawn

</details>

### v1.5 Runtime Truth Recovery (In Progress)

**Milestone Goal:** Recover runtime truth -- close P0 bypass paths, fix atomic state advancement, and restore colony continue orchestration so active colonies can unblock.

**Milestone Goal:** Give Aether the ability to diagnose and repair its own colony data, ceremony integrity, and runtime state -- reducing manual intervention and preventing documentation gaps.

## Phases

- [x] **Phase 31: P0 Runtime Truth Fixes** -- Invoker honesty, dispatch errors, continue bypass closure, git-verified claims, atomic state (completed 2026-04-22)
- [ ] **Phase 32: Continue Unblock** -- Full continue orchestration recovery
- [ ] **Phase 33: Dispatch Fixes** -- P1 dispatch robustness improvements
- [ ] **Phase 34: Cleanup** -- Stale worktrees, branches, blockers
- [ ] **Phase 35: Platform Parity** -- Claude/OpenCode/Codex alignment
- [ ] **Phase 36: Release Decision** -- v1.0.20 cut and ship decision

## Phase Details

### Phase 31: P0 Runtime Truth Fixes
**Goal:** Close all P0 truth bugs from the oracle audit -- honest invokers, error propagation, and atomic state.
**Requirements:** R045 (FakeInvoker), R046 (dispatch errors), R047 (continue bypass), R048 (reconcile), R049 (git claims), R050 (no env dismissal), R051 (atomic state)
**Status:** Complete (2026-04-22) -- 4 plans, 16 commits, 12 validation tests green

### Phase 32: Continue Unblock
**Goal:** Recover full continue orchestration so active colonies can advance past phase 2. Detect abandoned builds, produce actionable recovery messages, clear stale reports, ensure end-to-end pipeline works.
**Plans:** 2 plans

Plans:
- [x] 32-01-PLAN.md -- Abandoned build detection and recovery (TDD) (completed 2026-04-22)
- [x] 32-02-PLAN.md -- Stale report cleanup and E2E pipeline recovery (TDD) (completed 2026-04-22)

**Status:** Complete (2026-04-22) -- 2 plans, 5 commits, 5 new tests green

### Phase 33: Dispatch Fixes
**Goal:** Address P1 dispatch robustness issues from the oracle audit.
**Status:** Planned

### Phase 34: Cleanup
**Goal:** Address stale worktrees, orphaned branches, and stale blocker flags from prior colony work.
**Requirements:** R056 (stale worktrees), R057 (stale branches), R058 (blocker flags)
**Plans:** 3 plans

Plans:
- [x] 34-01-PLAN.md -- Backup colony data and evaluate candidate commits for preservation (completed 2026-04-23)
- [x] 34-02-PLAN.md -- Bulk worktree and branch cleanup with interactive confirmation (completed 2026-04-23)
- [ ] 34-03-PLAN.md -- Interactive blocker flag review and resolution

**Status:** In progress (2/3 plans complete)

### Phase 35: Platform Parity
**Goal:** Align Claude/OpenCode/Codex command and agent UX.
**Status:** Planned

### Phase 36: Release Decision
**Goal:** Cut v1.0.20, validate release readiness, or defer.
**Status:** Planned

## Progress

| Phase | Milestone | Plans | Status | Completed |
|-------|-----------|-------|--------|-----------|
| 1-6 | v1.0 | 13/13 | Complete | 2026-04-21 |
| 7-11 | v1.1 | 10/10 | Complete | 2026-04-21 |
| 12-16 | v1.2 | 12/12 | Complete | 2026-04-21 |
| 17-24 | v1.3 | 17/17 | Complete | 2026-04-21 |
| 25-30 | v1.4 | 6/6 | Complete | 2026-04-21 |
| 31 | v1.5 | 4/4 | Complete | 2026-04-22 |
| 32 | v1.5 | 2 | Planned | -- |
| 33-36 | v1.5 | 0 | Planned | -- |
