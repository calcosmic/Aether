# Roadmap: Aether

## Milestones

- **v1.0 MVP** - Phases 1-6 (shipped)
- **v1.1 Trusted Context** - Phases 7-11 (shipped)
- **v1.2 Live Dispatch Truth and Recovery** - Phases 12-16 (shipped)
- **v1.3 Visual Truth and Core Hardening** - Phases 17-24 (shipped 2026-04-21)
- **v1.4 Self-Healing Colony** - Phases 25-30 (completed 2026-04-21)
- **v1.5 Runtime Truth Recovery** - Phases 31-38 (completed 2026-04-23, product v1.0.20)
- **v1.6 Release Pipeline Integrity** - Phases 39-46 (completed 2026-04-24)
- **v1.7 Planning Pipeline Recovery** - Phases 47-48 (completed 2026-04-24)
- **v1.8 Colony Recovery** - Phases 49-51

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

<details>
<summary>v1.5 Runtime Truth Recovery (Phases 31-38) -- COMPLETED 2026-04-23</summary>

8 phases, 17 plans, 176 commits. P0 runtime truth fixes, continue unblock, dispatch robustness, cleanup, platform parity, v1.0.20 release, codebase hygiene, Nyquist validation. [Full archive -> milestones/v1.5-ROADMAP.md]

</details>

<details>
<summary>v1.6 Release Pipeline Integrity (Phases 39-46) -- COMPLETED 2026-04-24</summary>

8 phases (including inserted 44.1, 44.2), 10 plans. OpenCode agent frontmatter fix, stable publish hardening, dev channel isolation, stale publish detection, release integrity checks, doc alignment, E2E regression coverage, stuck-plan investigation. [Full archive -> milestones/v1.6-ROADMAP.md]

</details>

<details>
<summary>v1.7 Planning Pipeline Recovery (Phases 47-48) -- COMPLETED 2026-04-24</summary>

2 phases. Plan --force recovery, fallback artifact cleanup, scout timeout, E2E recovery test. [Full archive -> milestones/v1.7-ROADMAP.md]

</details>

### v1.8 Colony Recovery (Phases 49-51)

- [ ] **Phase 49: Stuck-State Scanner and Diagnosis** - Detect all 7 stuck-state classes and present clean diagnosis
- [ ] **Phase 50: Repair Pipeline** - Auto-fix safe issues, prompt for destructive ones, backup-first
- [ ] **Phase 51: Recovery Verification** - E2E tests for all 7 states, compound recovery, no false positives

## Phase Details

### Phase 49: Stuck-State Scanner and Diagnosis
**Goal**: Users can run `aether recover` and see a clear, actionable diagnosis of everything wrong with their stuck colony
**Depends on**: v1.7 complete (Phase 48)
**Requirements**: DETECT-01, DETECT-02, DETECT-03, DETECT-04, DETECT-05, DETECT-06, DETECT-07, OUTP-01, OUTP-02
**Success Criteria** (what must be TRUE):
  1. `aether recover` scans for all 7 stuck-state classes (missing packet, stale workers, partial phase, bad manifest, dirty worktree, broken survey, missing agents) and reports each one found with severity and a one-line description
  2. Output is a clean diagnosis table (not debug output) showing issue class, severity, and fix hint for each detected problem
  3. Command exits 0 when colony is healthy and exits 1 when issues are detected, making it usable in shell scripts
  4. `--json` flag produces structured output with the same diagnosis data for programmatic consumption
**Plans**: TBD

### Phase 50: Repair Pipeline
**Goal**: Users can run `aether recover --apply` and have all safe issues fixed automatically, with confirmation required only for potentially destructive operations
**Depends on**: Phase 49
**Requirements**: REPAIR-01, REPAIR-02, REPAIR-03, REPAIR-04, REPAIR-05
**Success Criteria** (what must be TRUE):
  1. `aether recover --apply` auto-fixes all 5 safe classes (missing packet, stale workers, partial phase, broken survey, missing agents) without user interaction
  2. Dirty worktree and bad manifest repairs prompt the user for confirmation before proceeding
  3. Every repair creates a timestamped backup of `.aether/data/` before mutating any state files
  4. Multi-file repairs are atomic -- if any step fails, all changes roll back to the backup
  5. After repairs, the command re-scans and reports what was fixed and what still needs attention
**Plans**: TBD

### Phase 51: Recovery Verification
**Goal**: Every recovery path is proven correct through automated tests, including edge cases and compound scenarios
**Depends on**: Phase 50
**Requirements**: TEST-01, TEST-02, TEST-03
**Success Criteria** (what must be TRUE):
  1. E2E tests prove recovery works for each of the 7 stuck states individually (each state is seeded, recovered, and verified clean)
  2. E2E test proves recovery works when multiple stuck states exist simultaneously (compound scenario)
  3. Test proves `aether recover` reports zero issues on a healthy, active colony (no false positives)
**Plans**: TBD

## Progress

| Milestone | Phases | Status | Completed |
|-----------|--------|--------|-----------|
| v1.0 | 1-6 | Complete | 2026-04-21 |
| v1.1 | 7-11 | Complete | 2026-04-21 |
| v1.2 | 12-16 | Complete | 2026-04-21 |
| v1.3 | 17-24 | Complete | 2026-04-21 |
| v1.4 | 25-30 | Complete | 2026-04-21 |
| v1.5 | 31-38 | Complete | 2026-04-23 |
| v1.6 | 39-46 | Complete | 2026-04-24 |
| v1.7 | 47-48 | Complete | 2026-04-24 |
| v1.8 | 49-51 | Not started | - |
