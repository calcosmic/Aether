# Roadmap: Aether

## Milestones

- **v1.0 MVP** - Phases 1-6 (shipped)
- **v1.1 Trusted Context** - Phases 7-11 (shipped)
- **v1.2 Live Dispatch Truth and Recovery** - Phases 12-16 (shipped 2026-04-21)
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
- **v1.17 Classic Restoration** - Phases 112-118 (shipped 2026-05-14) — [Archive](milestones/v1.17-ROADMAP.md)
- **v1.18 Hybrid Runtime Parity & Release Gate** - Phases 119-123 (shipped 2026-05-14) — [Archive](milestones/v1.18-ROADMAP.md)

## Phases

<details>
<summary>v1.0 through v1.18 Phase Summaries (archived)</summary>

See `.planning/milestones/` for full archived phase details.

</details>

### Current Milestone: v1.19 TypeScript Host Cutover + Oracle Confidence Recovery

**Goal:** Make the TypeScript host the primary orchestration layer for real user workflows. Cut over plan/build/continue wrappers from manual Go command chains to host-backed orchestration. Restore Oracle/RALF confidence iteration through the hybrid manifest/finalizer boundary.

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 124. Host Entry Point | v1.19 | 0/TBD | Not started | — |
| 125. Wrapper Cutover: Plan/Build/Continue | v1.19 | 0/TBD | Not started | — |
| 126. Oracle Iteration Manifests | v1.19 | 0/TBD | Not started | — |
| 127. TS Host Oracle Lifecycle | v1.19 | 0/TBD | Not started | — |
| 128. Swarm/Watch Host Bridge | v1.19 | 0/TBD | Not started | — |
| 129. Release Gate | v1.19 | 0/TBD | Not started | — |

## Phase Details

### Phase 124: Host Entry Point
**Goal:** Add a stable `aether host <workflow>` entry point that runs the built TypeScript host from installed Aether assets.
**Depends on:** Phase 123 (v1.18 shipped)
**Requirements:** HEP-01, HEP-02, HEP-03, HEP-04, HEP-05, HEP-06, HEP-07
**Success Criteria**:
  1. `aether host lifecycle` runs the TS host end-to-end
  2. `aether host plan/build/continue/oracle` exist and delegate to TS
  3. Publish/update installs and builds the TS host correctly
  4. Fallback messaging when Node/host deps are missing
**Plans**: 1-2 plans

### Phase 125: Wrapper Cutover — Plan/Build/Continue
**Goal:** Update Claude/OpenCode wrappers and YAML command sources to call the TS host instead of manually performing plan-only/finalizer/spawn-log steps.
**Depends on:** Phase 124
**Requirements:** WCO-01, WCO-02, WCO-03, WCO-04, WCO-05, WCO-06, WCO-07
**Success Criteria**:
  1. Claude plan/build/continue wrappers call TS host
  2. OpenCode plan/build/continue wrappers call TS host
  3. Go CLI direct commands still work for fallback
  4. Wrapper files are materially shorter
**Plans**: 1-2 plans

### Phase 126: Oracle Iteration Manifests
**Goal:** Add Go commands for Oracle iteration so Go owns state, confidence math, and finalizers.
**Depends on:** Phase 124
**Requirements:** OIM-01, OIM-02, OIM-03, OIM-04
**Success Criteria**:
  1. `oracle-iterate --plan-only` returns iteration manifest
  2. `oracle-iterate-finalize` commits Oracle state
  3. Go owns all Oracle state writes
**Plans**: 1 plan

### Phase 127: TS Host Oracle Lifecycle
**Goal:** Add `runOracleLifecycle()` in TS that dispatches Oracle workers and loops until confidence target/max iterations/stop.
**Depends on:** Phase 126
**Requirements:** TOL-01, TOL-02, TOL-03, TOL-04, TOL-05
**Success Criteria**:
  1. TS dispatches Oracle workers via platform dispatcher
  2. TS loops until confidence target, max iterations, or stop
  3. TS never writes directly to `.aether/data/oracle`
**Plans**: 1 plan

### Phase 128: Swarm/Watch Host Bridge
**Goal:** Add TS host watch/swarm display using Go JSON output for structured display.
**Depends on:** Phase 125
**Requirements:** SWB-01, SWB-02, SWB-03
**Success Criteria**:
  1. TS host watch uses Go JSON output
  2. TS host swarm uses Go JSON output
  3. Live visibility without wrapper visual parsing
**Plans**: 1 plan

### Phase 129: Release Gate
**Goal:** Verify typecheck, tests, downstream smoke, and cross-platform consistency.
**Depends on:** Phase 125, Phase 127, Phase 128
**Requirements:** REL-01, REL-02, REL-03, REL-04, REL-05
**Success Criteria**:
  1. `npm run typecheck` passes
  2. `npm test` passes
  3. `go test ./...` passes
  4. Downstream smoke test passes
  5. All 3 platforms behave consistently
**Plans**: 1 plan

---

*Active roadmap. See `.planning/milestones/` for shipped milestone archives.*
