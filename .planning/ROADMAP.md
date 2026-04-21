# Roadmap: Aether

## Milestones

- ✅ **v1.0 MVP** - Phases 1-6 (shipped)
- ✅ **v1.1 Trusted Context** - Phases 7-11 (shipped)
- ✅ **v1.2 Live Dispatch Truth and Recovery** - Phases 12-16 (shipped)
- ✅ **v1.3 Visual Truth and Core Hardening** - Phases 17-24 (shipped 2026-04-21)
- 🚧 **v1.4 Self-Healing Colony** - Phases 25-30 (in progress)

## Phases

<details>
<summary>✅ v1.0 MVP (Phases 1-6) — SHIPPED</summary>

- Phase 1: Housekeeping and Foundation
- Phase 2: Colony Scope System
- Phase 3: Restore Build Ceremony
- Phase 4: Restore Continue Ceremony
- Phase 5: Living Watch and Status Surfaces
- Phase 6: Pheromone Visibility and Steering

</details>

<details>
<summary>✅ v1.1 Trusted Context (Phases 7-11) — SHIPPED</summary>

- Phase 7: Context Ledger and Skill Routing Foundation
- Phase 8: Prompt Integrity and Trust Boundaries
- Phase 9: Trust-Weighted Context Assembly
- Phase 10: Curation Spine and Structural Learning
- Phase 11: Competitive Proof Surfaces and Evaluation

</details>

<details>
<summary>✅ v1.2 Live Dispatch Truth and Recovery (Phases 12-16) — SHIPPED</summary>

- Phase 12: Dispatch Truth Model and Run Scoping
- Phase 13: Live Workflow Visibility Across Colonize, Plan, and Build
- Phase 14: Worker Execution Robustness and Honest Activity Tracking
- Phase 15: Verification-Led Continue and Partial Success
- Phase 16: Recovery, Reconciliation, and Runtime UX Finalization

</details>

<details>
<summary>✅ v1.3 Visual Truth and Core Hardening (Phases 17-24) — SHIPPED 2026-04-21</summary>

- Phase 17: Slash Command Format Audit
- Phase 18: Visual UX Restoration — Caste Identity and Spawn Lists
- Phase 19: Visual UX Restoration — Stage Separators and Ceremony
- Phase 20: Visual UX Restoration — Emoji Consistency
- Phase 21: Codex CLI Visual Parity
- Phase 22: Core Path Hardening
- Phase 23: Recovery and Continuity
- Phase 24: Full Instrumentation — Trace Logging

</details>

### 🚧 v1.4 Self-Healing Colony (In Progress)

**Milestone Goal:** Give Aether the ability to diagnose and repair its own colony data, ceremony integrity, and runtime state — reducing manual intervention and preventing documentation gaps.

## Phases

- [x] **Phase 25: Medic Ant Core** — Health diagnosis command, colony data scanner, structured health report (completed 2026-04-21)
- [x] **Phase 26: Auto-Repair** — Fix common colony data issues with `--fix` flag, logged repairs (completed 2026-04-21)
- [x] **Phase 27: Medic Skill** — Healthy state specification skill file for all colony data (completed 2026-04-21)
- [ ] **Phase 28: Ceremony Integrity** — Verify wrapper/runtime parity, stage markers, emoji consistency
- [ ] **Phase 29: Trace Diagnostics** — Remote debugging via trace export analysis
- [ ] **Phase 30: Medic Worker Integration** — Caste integration, auto-spawn on detected health issues

## Phase Details

### Phase 25: Medic Ant Core
**Goal:** Users can run `/ant:medic` or `aether medic` to get a structured health report of their colony.
**Depends on:** Phase 24
**Requirements:** R039, R044
**Success Criteria:**
1. `aether medic` command exists and runs without errors
2. Scans COLONY_STATE.json, pheromones.json, session.json, constraints.json, trace.jsonl
3. Reports corruption, staleness, inconsistency, or missing fields
4. Output is human-readable with clear severity levels
5. Read-only by default; no mutations without `--fix`

### Phase 26: Auto-Repair
**Goal:** The Medic can fix common issues automatically when invoked with `--fix`.
**Depends on:** Phase 25
**Requirements:** R040
**Success Criteria:**
1. `--fix` flag repairs stale spawn state
2. `--fix` removes orphaned worktree entries
3. `--fix` rebuilds missing indexes
4. `--fix` fixes corrupted JSON structures
5. Every repair is logged to trace.jsonl with before/after state

### Phase 27: Medic Skill
**Goal:** Colony health rules are documented in a versioned skill file.
**Depends on:** Phase 25
**Requirements:** R041
**Success Criteria:**
1. `.aether/skills/colony/medic.md` exists with healthy state specifications
2. Documents schema, required fields, valid values for all colony files
3. Documents common failure modes and remedies
4. Auto-loaded by colony-prime when Medic worker spawns
5. Kept in sync with code changes

### Phase 28: Ceremony Integrity
**Goal:** Wrapper/runtime drift is caught automatically before it causes issues.
**Depends on:** Phase 25
**Requirements:** R042
**Success Criteria:**
1. Verifies stage markers present in all state-changing commands
2. Verifies emoji consistency against commandEmojiMap/casteEmojiMap
3. Verifies context-clear guidance is runtime-owned
4. Reports mismatches between Claude/OpenCode wrappers and runtime
5. Integrates into `/ant:patrol` or build gate checks

### Phase 29: Trace Diagnostics
**Goal:** A maintainer can debug user-reported issues from trace export alone.
**Depends on:** Phase 25, Phase 24
**Requirements:** R043
**Success Criteria:**
1. Accepts trace export JSON and produces diagnostic summary
2. Reconstructs state transition timeline
3. Identifies error clusters and intervention points
4. Suggests fixes based on trace patterns
5. Works without access to user's repo

### Phase 30: Medic Worker Integration
**Goal:** The Medic is a first-class colony worker that can auto-spawn on health issues.
**Depends on:** Phase 25-29
**Requirements:** R044
**Success Criteria:**
1. Medic caste exists with emoji, color, and label
2. Can be spawned manually (`/ant:medic`) or automatically
3. Auto-spawns when stale session, corrupted state, or critical blocker detected
4. Produces structured health report with recommended next steps
5. Integrates into colony-prime context assembly

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1-6 | v1.0 | 13/13 | Complete | 2026-04-21 |
| 7-11 | v1.1 | 10/10 | Complete | 2026-04-21 |
| 12-16 | v1.2 | 12/12 | Complete | 2026-04-21 |
| 17-24 | v1.3 | 17/17 | Complete | 2026-04-21 |
| 25 | v1.4 | 1/3 | Complete    | 2026-04-21 |
| 26 | v1.4 | 1/1 | Complete    | 2026-04-21 |
| 27 | v1.4 | 0/1 | Complete    | 2026-04-21 |
| 28 | v1.4 | 0/TBD | Not started | - |
| 29 | v1.4 | 0/TBD | Not started | - |
| 30 | v1.4 | 0/TBD | Not started | - |
