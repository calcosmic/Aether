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
- **v1.19 TypeScript Host Cutover + Oracle Confidence Recovery** - Phases 124-129 (shipped 2026-05-15) — [Archive](milestones/v1.19-ROADMAP.md)

## Phases

<details>
<summary>v1.0 through v1.19 Phase Summaries (archived)</summary>

See `.planning/milestones/` for full archived phase details.

</details>

### Current Milestone: v1.20 Host Contract Hardening and Wrapper Reality Check

**Goal:** Make the TS host contract honest — docs, CLI flags, tests, and wrapper behavior agree on who owns what. Harden the host surface so every documented command works, every flag is tested, and the wrapper/host boundary is explicit.

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 130. Host Surface Completeness | v1.20 | — | Not started | — |
| 131. Test Coverage for Host Commands | v1.20 | — | Not started | — |
| 132. Wrapper Ownership Decision | v1.20 | — | Not started | — |
| 133. Lifecycle Honesty | v1.20 | — | Not started | — |
| 134. Oracle Storage Pattern | v1.20 | — | Not started | — |
| 135. Doc-CLI Alignment Smoke Test | v1.20 | — | Not started | — |

## Phase Details

### Phase 130: Host Surface Completeness
**Goal:** Ensure `aether host` exposes every workflow the docs claim: plan, build, continue, oracle, watch, swarm, lifecycle.
**Depends on:** Phase 129 (v1.19 shipped)
**Requirements:** HSC-01, HSC-02, HSC-03, HSC-04, HSC-05, HSC-06, HSC-07
**Success Criteria:**
  1. All 7 `aether host` subcommands exist and execute end-to-end
  2. `--depth`, `--planning-depth`, `--verification-depth`, `--no-dashboard` flags accepted
  3. No undocumented or broken host subcommands
**Plans:** 1-2 plans

### Phase 131: Test Coverage for Host Commands
**Goal:** Documented commands execute in tests with realistic flag combinations.
**Depends on:** Phase 130
**Requirements:** TCV-01, TCV-02, TCV-03, TCV-04, TCV-05
**Success Criteria:**
  1. `aether host plan --depth balanced --planning-depth standard` tested
  2. `aether host continue --verification-depth heavy` tested
  3. `aether host watch --no-dashboard` tested
  4. `aether host swarm <target> --no-dashboard` tested
  5. 100% of documented `aether host` commands have test coverage
**Plans:** 1 plan

### Phase 132: Wrapper Ownership Decision
**Goal:** Decide and document whether wrappers are "host-assisted orchestrators" or thin pass-throughs.
**Depends on:** Phase 130
**Requirements:** WRO-01, WRO-02, WRO-03, WRO-04
**Success Criteria:**
  1. ADR or contract doc records the ownership decision
  2. Wrapper code updated to match (no duplicate verification/gating)
  3. Wrapper README explains the boundary
**Plans:** 1 plan

### Phase 133: Lifecycle Honesty
**Goal:** Remove or gate synthetic lifecycle shortcuts.
**Depends on:** Phase 132
**Requirements:** LHO-01, LHO-02, LHO-03
**Success Criteria:**
  1. No production shortcuts that skip real worker dispatch
  2. Any remaining synthetic paths behind `--simulate`
  3. `--simulate` documented and tested
**Plans:** 1 plan

### Phase 134: Oracle Storage Pattern
**Goal:** Move Oracle iteration state to locked/atomic Go-owned storage.
**Depends on:** Phase 132
**Requirements:** ORS-01, ORS-02, ORS-03
**Success Criteria:**
  1. Oracle state uses same atomic storage as colony state
  2. TS host reads via Go CLI JSON, never writes directly
  3. Interrupt recovery works for Oracle sessions
**Plans:** 1 plan

### Phase 135: Doc-CLI Alignment Smoke Test
**Goal:** Executable YAML smoke test so docs cannot agree with each other while the CLI rejects commands.
**Depends on:** Phase 131
**Requirements:** DCA-01, DCA-02, DCA-03
**Success Criteria:**
  1. YAML smoke test exercises every documented command
  2. Test fails if documented flag is rejected by CLI
  3. Smoke test runs in CI or release gate
**Plans:** 1 plan

---

*Active roadmap. See `.planning/milestones/` for shipped milestone archives.*
