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

## Phases

<details>
<summary>v1.0 through v1.15 Phase Summaries (archived)</summary>

See `.planning/milestones/` for full archived phase details.

</details>

### v1.16 Hybrid Runtime Boundary and Orchestration Recovery (In Progress)

- [x] **Phase 106: Boundary Contract** -- Write and commit the runtime boundary contract that assigns ownership to Go, TypeScript, editable assets, and Bash (completed 2026-05-12)
- [ ] **Phase 107: Classic Baseline Identification** -- Identify and smoke-test the best Classic version (likely v5.4.0) as a behavior comparison anchor
- [ ] **Phase 108: Golden Workflow Tests** -- Add snapshot/golden tests for `plan -> build 1 -> continue` covering ceremony, worker activity, and state side effects
- [ ] **Phase 109: TypeScript Orchestration Host Prototype** -- Build minimal TS host that calls Go manifests/finalizers, dispatches visible workers, records spawn-log/spawn-complete, never writes `.aether/data` directly
- [ ] **Phase 110: Go Safety Invariant Verification** -- Ensure Go remains sole authority for state mutation, finalizers, locking, install/update/publish, verification contracts
- [ ] **Phase 111: Follow-up Migration Map** -- Produce concrete next steps for Oracle confidence iteration, swarm visibility, and broader build/continue parity

## Phase Details

### Phase 106: Boundary Contract
**Goal**: A written runtime boundary contract exists that clearly assigns ownership to Go, TypeScript, editable assets, and Bash, and is committed to the repo
**Depends on**: Nothing (first phase of v1.16)
**Requirements**: BOUND-01, BOUND-02, BOUND-03
**Success Criteria** (what must be TRUE):
  1. A single committed document clearly lists what Go owns, what TypeScript owns, what editable assets own, and what Bash may still do
  2. The document is referenced by a comment or import in the TypeScript host source code
  3. The document includes explicit anti-patterns: no TS direct state writes, no visual output parsing as authority, no wrapper-owned recovery menus
Plans:
- [x] 106-01-PLAN.md -- Write boundary contract, TS host skeleton, Go integration test (completed 2026-05-12)

### Phase 107: Classic Baseline Identification
**Goal**: The best Classic version is identified with evidence and a smoke-test script verifies it can run `plan -> build 1 -> continue` without errors
**Depends on**: Phase 106
**Requirements**: BASE-01, BASE-02, BASE-03
**Success Criteria** (what must be TRUE):
  1. A documented comparison of v5.3.0, v5.3.3, and v5.4.0 against behavior criteria exists with a selected tag and rationale
  2. A smoke-test script checks out the selected Classic tag and verifies the lifecycle runs without errors
  3. Baseline documentation includes selected tag, selection rationale, known limitations, and a behavior comparison checklist
**Plans**: 2 plans
Plans:
- [ ] 107-01-PLAN.md -- Create Classic baseline document with version comparison and behavioral checklist
- [ ] 107-02-PLAN.md -- Create standalone Bash smoke test for Classic v5.4.0

### Phase 108: Golden Workflow Tests
**Goal**: Golden/snapshot tests exist for the `plan -> build 1 -> continue` lifecycle and run in CI, failing on ceremony, worker activity, or state behavior regressions
**Depends on**: Phase 107
**Requirements**: TEST-01, TEST-02, TEST-03, TEST-04, TEST-05
**Success Criteria** (what must be TRUE):
  1. A golden/snapshot test captures the full `plan -> build 1 -> continue` lifecycle output
  2. The test captures visible ceremony output (stage separators, caste labels, worker banners)
  3. The test captures worker activity (spawn-log entries, dispatch manifests, worker descriptions)
  4. The test captures state side effects (COLONY_STATE.json mutations only after finalizers, no pre-finalize state writes)
  5. The test runs in CI and fails if ceremony, worker activity, or state behavior regresses
**Plans**: 1 plan
Plans:
- [ ] 108-01-PLAN.md -- Create golden lifecycle snapshot tests for plan/build/continue and state mutation verification

### Phase 109: TypeScript Orchestration Host Prototype
**Goal**: A minimal TypeScript host can drive `plan -> build 1 -> continue` through Go manifests and finalizers without direct state writes, producing visible worker activity and ceremony
**Depends on**: Phase 108
**Requirements**: HOST-01, HOST-02, HOST-03, HOST-04, HOST-05, HOST-06, HOST-07
**Success Criteria** (what must be TRUE):
  1. A minimal TypeScript host prototype exists and can be invoked as `aether-host` or equivalent
  2. The host calls Go `--plan-only` commands to obtain JSON manifests (not visual output parsing)
  3. The host dispatches visible platform workers from manifest fields (spawn-log before, spawn-complete after)
  4. The host calls Go finalizers to commit state changes
  5. The host never writes `.aether/data/` directly — all state mutation goes through Go finalizers
  6. The host records spawn lifecycle events (spawn-log / spawn-complete) via Go CLI subcommands
  7. The host either runs the selected workflow end-to-end or documents the exact blocker with a reproducible test
**Plans**: TBD

### Phase 110: Go Safety Invariant Verification
**Goal**: Go remains the sole authority for state mutation, finalizers, locking, install/update/publish, and verification contracts, with tests proving invariants hold when the TS host is present
**Depends on**: Phase 109
**Requirements**: SAFE-01, SAFE-02, SAFE-03, SAFE-04, SAFE-05, SAFE-06
**Success Criteria** (what must be TRUE):
  1. Go remains sole authority for COLONY_STATE.json mutation — no other process writes it
  2. Go finalizers validate manifest provenance before any state write
  3. Go locking and atomic write semantics are unchanged by TS host presence
  4. Install, update, publish commands remain pure Go — no TS involvement
  5. Verification contracts (command-guide, parity tests, drift guards) still pass with TS host enabled
  6. Existing `aether plan --plan-only` and `aether build --plan-only` behavior is unchanged
**Plans**: TBD

### Phase 111: Follow-up Migration Map
**Goal**: A written follow-up plan exists with phase numbers, estimated scope, and dependency ordering for restoring Oracle/RALF confidence iteration, swarm visibility, and broader build/continue parity
**Depends on**: Phase 110
**Requirements**: MAP-01, MAP-02, MAP-03, MAP-04
**Success Criteria** (what must be TRUE):
  1. A written follow-up plan exists for restoring Oracle/RALF confidence iteration
  2. A written follow-up plan exists for restoring swarm visibility
  3. A written follow-up plan exists for broader build/continue parity (all flows use TS host)
  4. The map includes phase numbers, estimated scope, and dependency ordering
**Plans**: TBD

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 106. Boundary Contract | v1.16 | 0/1 | Planning complete | - |
| 107. Classic Baseline Identification | v1.16 | 0/2 | Planning complete | - |
| 108. Golden Workflow Tests | v1.16 | 0/1 | Planning complete | - |
| 109. TypeScript Orchestration Host Prototype | v1.16 | 0/TBD | Not started | - |
| 110. Go Safety Invariant Verification | v1.16 | 0/TBD | Not started | - |
| 111. Follow-up Migration Map | v1.16 | 0/TBD | Not started | - |
