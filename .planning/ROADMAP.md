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

## Phases

<details>
<summary>v1.0 through v1.17 Phase Summaries (archived)</summary>

See `.planning/milestones/` for full archived phase details.

</details>

### Current Milestone: v1.18 Hybrid Runtime Parity & Release Gate (In Progress)

**Goal:** Make the hybrid runtime reliable enough to publish and build future features on. Restore confidence by proving Classic parity for the workflows that matter.

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 119. TS Host Reliability | v1.18 | 0/TBD | Not started | — |
| 120. Platform Dispatch Correctness | v1.18 | 0/TBD | Not started | — |
| 121. Go Runtime Test Restoration | v1.18 | 0/TBD | Not started | — |
| 122. Classic Parity Coverage | v1.18 | 0/TBD | Not started | — |
| 123. Dev Publish + Downstream Smoke | v1.18 | 0/TBD | Not started | — |

## Phase Details

### Phase 119: TS Host Reliability
**Goal:** Fix TypeScript typecheck failures, test suite hangs, and temp file races so the TS host is trustworthy.
**Depends on:** Phase 118 (v1.17 shipped)
**Requirements:** REL-01, REL-02, REL-03, REL-04, REL-05
**Success Criteria** (what must be TRUE):
  1. `npm --prefix .aether/ts-host run typecheck` passes with zero errors
  2. `npm --prefix .aether/ts-host test` exits cleanly without hangs
  3. Event bridge teardown awaits full subprocess/readline cleanup
  4. Completion file paths are unique per lifecycle run
  5. Individual tests and suite mode both pass for lifecycle and golden workflow
**Plans**: 1-2 plans

### Phase 120: Platform Dispatch Correctness
**Goal:** Fix Codex prompt passing, add dispatch tests for all three platforms, and ensure simulation fallback is explicit.
**Depends on:** Phase 119
**Requirements:** DSP-01, DSP-02, DSP-03, DSP-04, DSP-05
**Success Criteria**:
  1. Codex dispatch passes the worker prompt to `codex exec`
  2. Claude and OpenCode argument construction are tested
  3. Simulation fallback is explicit and does not mask broken real dispatch
  4. Spawn-log/spawn-complete only records manifest workers
**Plans**: 1-2 plans

### Phase 121: Go Runtime Test Restoration
**Goal:** Resolve workspace cleanup state, fix `go test ./cmd` failures, and restore Go test meaning.
**Depends on:** Phase 119
**Requirements:** GOT-01, GOT-02, GOT-03, GOT-04, GOT-05
**Success Criteria**:
  1. `go test ./cmd` passes with zero failures
  2. Resume dashboard signal injection failure is resolved
  3. Workspace/planning cleanup state is resolved
  4. Scratch files are removed or promoted
  5. Go remains sole authority for `.aether/data` mutation
**Plans**: 1 plan

### Phase 122: Classic Parity Coverage
**Goal:** Verify restored behavior matches Classic v5.4 baseline for build, continue, Oracle, dashboard, install/update, and state mutation.
**Depends on:** Phase 120, Phase 121
**Requirements:** PAR-01, PAR-02, PAR-03, PAR-04, PAR-05, PAR-06, PAR-07
**Success Criteria**:
  1. Golden tests verify build ceremony matches v5.4 baseline
  2. Golden tests verify continue ceremony matches v5.4 baseline
  3. Oracle confidence loop behavior is tested against v5.4 baseline
  4. Swarm/dashboard visibility is tested against v5.4 baseline
  5. Install/update flow is tested against v5.4 baseline
  6. State mutation through approved APIs is tested against v5.4 baseline
  7. Any Classic behavior intentionally not restored is documented
**Plans**: 1-2 plans

### Phase 123: Dev Publish + Downstream Smoke
**Goal:** Publish to dev channel and verify the full workflow in a clean downstream repo.
**Depends on:** Phase 122
**Requirements:** REL-06, REL-07, REL-08
**Success Criteria**:
  1. Dev channel publish succeeds
  2. Downstream smoke test passes: `aether update --force`, `aether init`, `aether plan`, `aether build 1`, `aether continue`, `aether oracle`
  3. Exact blocker list is recorded before stable release
**Plans**: 1 plan

---

*Active roadmap. See `.planning/milestones/` for shipped milestone archives.*
