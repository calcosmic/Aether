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
- **v1.8 Colony Recovery** - Phases 49-51 (shipped 2026-04-25)
- **v1.9 Review Persistence** - Phases 52-56 (shipped 2026-04-26)
- **v1.10 Colony Polish** - Phases 57-69 (shipped 2026-04-28)
- **v1.11 Aether Unification** - Phases 70-79 (shipped 2026-04-30)
- **v1.12 Safe Colony** - Phases 80-87 (shipped 2026-05-01)
- **v1.13 Recovery Hardening & Hive Learning** - Phases 88-92 (shipped 2026-05-03)
- **v1.14 Queen Authority** - Phases 93-99 (shipped 2026-05-04)
- **v1.15 Framework Coherence, Efficiency, and Ship Readiness** - Phases 100-105 (shipped 2026-05-08)
- **v1.16 Hybrid Runtime Boundary and Orchestration Recovery** - Phases 106-111 (shipped 2026-05-13)
- **v1.17 Classic Restoration** - Phases 112-118 (in progress)

## Phases

<details>
<summary>v1.0 through v1.15 Phase Summaries (archived)</summary>

See `.planning/milestones/` for full archived phase details.

</details>

<details>
<summary>v1.16 Hybrid Runtime Boundary and Orchestration Recovery (Phases 106-111) — SHIPPED 2026-05-13</summary>

- [x] Phase 106: Boundary Contract (1/1 plans) — completed 2026-05-12
- [x] Phase 107: Classic Baseline Identification (2/2 plans) — completed 2026-05-12
- [x] Phase 108: Golden Workflow Tests (1/1 plan) — completed 2026-05-12
- [x] Phase 109: TypeScript Orchestration Host Prototype (3/3 plans) — completed 2026-05-12
- [x] Phase 110: Go Safety Invariant Verification (1/1 plan) — completed 2026-05-12
- [x] Phase 111: Follow-up Migration Map (1/1 plan) — completed 2026-05-12

</details>

### v1.17 Classic Restoration (In Progress)

**Milestone Goal:** Restore the full Aether lifecycle to v5.4-level richness — ceremony, Queen intelligence, swarm display, and the whole build/continue flow — with Go as the utility belt and the TS host as the control plane.

- [ ] **Phase 112: Foundation** - Event bridge, shared ceremony config, Node bump, boundary enforcement
- [ ] **Phase 113: Ceremony Narrator** - Render Go events into banners, caste identity, spawn frames, and stage markers
- [ ] **Phase 114: Real Worker Dispatch** - Replace simulation with actual platform worker spawning, parallel waves, error recovery
- [ ] **Phase 115: Swarm Dashboard** - Live terminal dashboard with animated spinners, progress bars, and chamber activity map
- [ ] **Phase 116: Queen Orchestration** - Workflow pattern selection, Builder-Probe Lock, tiered escalation, midden checks
- [ ] **Phase 117: Oracle Enhancement** - Phase-aware prompts, diminishing returns detection, template-specific synthesis
- [ ] **Phase 118: Integration & Parity Verification** - Golden workflow tests, ceremony snapshots, cross-platform smoke, state safety

## Phase Details

### Phase 112: Foundation
**Goal**: TS host can consume Go ceremony events and render basic output; shared config prevents platform drift; boundary contract is enforced
**Depends on**: Phase 111 (v1.16 shipped)
**Requirements**: TS-04, TS-05, TS-06, CER-02
**Success Criteria** (what must be TRUE):
  1. TS host reads Go ceremony events from JSONL stream and emits typed TypeScript events
  2. Shared YAML ceremony config (caste emoji, color, label maps) is loaded and consumed by TS host
  3. Node engine is bumped to >=20 and all new dependencies (chalk, boxen, figlet, ora, cli-progress, log-update, chokidar) install cleanly
  4. Any TS host attempt to write to `.aether/data/` is rejected at runtime or build time
**Plans**: TBD
**UI hint**: yes

### Phase 113: Ceremony Narrator
**Goal**: Users see living ceremony output — banners, caste identity frames, spawn notifications, stage separators — rendered from Go events
**Depends on**: Phase 112
**Requirements**: CER-01, CER-03, CER-04, CER-05, CER-06
**Success Criteria** (what must be TRUE):
  1. Build command wrappers display ASCII banners and stage separators rendered from event stream
  2. Worker spawn notifications include caste emoji, colored label, and deterministic name
  3. Crowned Anthill seal ASCII art renders from editable template (not compiled Go code)
  4. Build summary and closeout rituals display with framed template output
  5. Output falls back to plain text when TTY is not available (three-mode support: json, visual, markdown)
**Plans**: TBD
**UI hint**: yes

### Phase 114: Real Worker Dispatch
**Goal**: TS host dispatches real platform workers in parallel waves with honest error recovery and retry logic
**Depends on**: Phase 112
**Requirements**: TS-01, TS-02, TS-03
**Success Criteria** (what must be TRUE):
  1. `/ant-build` spawns actual Claude Code / OpenCode / Codex agents instead of 100ms simulated delays
  2. Workers within the same wave execute concurrently
  3. Failed workers are retried up to a configured limit; persistent failures escalate gracefully
  4. Worker timeouts are enforced and reported as events
  5. Worktree merge-back is performed between waves when manifest indicates it is required
**Plans**: TBD

### Phase 115: Swarm Dashboard
**Goal**: Users see a live terminal dashboard showing all active workers, their progress, tool usage, and chamber activity
**Depends on**: Phase 113, Phase 114
**Requirements**: SW-01, SW-02, SW-03, SW-04, SW-05, SW-06
**Success Criteria** (what must be TRUE):
  1. Running `/ant-build` shows an animated terminal dashboard with per-worker spinners
  2. Each active worker displays a progress bar with excavation status phrases
  3. Tool usage counters (reads, writes, fetches, commands) update per worker in real time
  4. Chamber activity map shows which project areas have active workers
  5. Elapsed time and token consumption are visible per worker
  6. Dashboard auto-refreshes as Go writes new events to the JSONL file
**Plans**: TBD
**UI hint**: yes

### Phase 116: Queen Orchestration
**Goal**: Queen intelligently selects workflow patterns, enforces Builder-Probe Lock, and manages tiered escalation
**Depends on**: Phase 114
**Requirements**: ORC-01, ORC-02, ORC-03, ORC-04, ORC-05, ORC-06
**Success Criteria** (what must be TRUE):
  1. Queen selects a workflow pattern (SPBV, Investigate-Fix, Refactor, Compliance, Documentation Sprint) based on phase name and content
  2. Builders return `code_written` status; only Probe upgrades worker status to `completed`
  3. Failed workers follow a tiered escalation: retry, then parent reassignment, then Queen reassignment, then user escalation
  4. Intra-build midden threshold breaches auto-emit REDIRECT pheromones
  5. Phase mode (discovery/prototype/production/maintenance) maps to the correct verification depth
  6. Ambassador ant is conditionally spawned for integration tasks
**Plans**: TBD

### Phase 117: Oracle Enhancement
**Goal**: Oracle RALF loop is enriched with phase-aware prompts, diminishing returns detection, and template-specific synthesis
**Depends on**: Phase 116
**Requirements**: ORA-01, ORA-02, ORA-03
**Success Criteria** (what must be TRUE):
  1. Oracle worker briefs include phase-aware directives (survey, investigate, synthesize, verify)
  2. Oracle detects diminishing returns via novelty delta tracking and forces phase advancement when progress stalls
  3. Oracle output includes template-specific synthesis sections for tech-eval, architecture-review, and bug-investigation
**Plans**: TBD

### Phase 118: Integration & Parity Verification
**Goal**: The restored system is proven to match Classic v5.4 behavior through automated tests; state safety is verified
**Depends on**: Phase 115, Phase 116, Phase 117
**Requirements**: PAR-01, PAR-02, PAR-03, PAR-04, CER-07
**Success Criteria** (what must be TRUE):
  1. Golden workflow tests compare ceremony output and behavior against the v5.4 Classic baseline and pass
  2. Ceremony snapshot tests verify banners, spawn plans, and seal rituals match their templates
  3. Cross-platform smoke tests pass for Claude Code, OpenCode, and Codex
  4. State safety tests prove all writes go through Go finalizers; no direct TS host writes to `.aether/data/`
  5. Seal ceremony executes the full multi-step ritual (Sage, Chronicler, wisdom review, commit suggestion)
**Plans**: TBD

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 112. Foundation | v1.17 | 0/TBD | Not started | - |
| 113. Ceremony Narrator | v1.17 | 0/TBD | Not started | - |
| 114. Real Worker Dispatch | v1.17 | 0/TBD | Not started | - |
| 115. Swarm Dashboard | v1.17 | 0/TBD | Not started | - |
| 116. Queen Orchestration | v1.17 | 0/TBD | Not started | - |
| 117. Oracle Enhancement | v1.17 | 0/TBD | Not started | - |
| 118. Integration & Parity | v1.17 | 0/TBD | Not started | - |
