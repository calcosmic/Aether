# Requirements: Aether v1.16

**Defined:** 2026-05-12
**Core Value:** Aether should feel alive and truthful at runtime, not only look clever in wrappers or tests.

## v1.16 Requirements

### Boundary Contract (BOUND)

- [x] **BOUND-01**: Written runtime boundary contract exists that clearly assigns ownership:
  - Go owns: state mutation, validation, atomic writes/locking, manifests, finalizers, install/update/publish, recovery, verification contracts, safe subprocess supervision
  - TypeScript owns: lifecycle orchestration host, platform adapters, worker wave orchestration, Oracle/RALF confidence iteration, ceremony rendering, prompt contract tests
  - Editable assets (Markdown/YAML/TOML) own: command playbooks, agent instructions, worker roles, prompts, ceremony copy, skills, platform surfaces
  - Bash owns: small smoke tests, setup checks, release glue
- [x] **BOUND-02**: Contract is committed to repo and referenced by TypeScript host source code
- [x] **BOUND-03**: Contract includes explicit anti-patterns: no TS direct state writes, no visual output parsing as authority, no wrapper-owned recovery menus

### Classic Baseline (BASE)

- [ ] **BASE-01**: Best Classic version identified with evidence (compare v5.3.0, v5.3.3, v5.4.0 against behavior criteria)
- [ ] **BASE-02**: Smoke-test script exists that checks out Classic tag and verifies it can run `plan -> build 1 -> continue` without errors
- [ ] **BASE-03**: Baseline documented with: selected tag, selection rationale, known limitations, behavior comparison checklist

### Golden Workflow Tests (TEST)

- [x] **TEST-01**: Golden/snapshot test exists for `plan -> build 1 -> continue` lifecycle
- [x] **TEST-02**: Test captures visible ceremony output (stage separators, caste labels, worker banners)
- [x] **TEST-03**: Test captures worker activity (spawn-log entries, dispatch manifests, worker descriptions)
- [x] **TEST-04**: Test captures state side effects (COLONY_STATE.json mutations only after finalizers, no pre-finalize state writes)
- [x] **TEST-05**: Test runs in CI and fails if ceremony, worker activity, or state behavior regresses

### TypeScript Orchestration Host (HOST)

- [ ] **HOST-01**: Minimal TypeScript host prototype exists that can be invoked as `aether-host` or equivalent
- [ ] **HOST-02**: Host calls Go `--plan-only` commands to obtain JSON manifests (not visual output parsing)
- [ ] **HOST-03**: Host dispatches visible platform workers from manifest fields (spawn-log before, spawn-complete after)
- [ ] **HOST-04**: Host calls Go finalizers to commit state changes
- [ ] **HOST-05**: Host never writes `.aether/data/` directly — all state mutation goes through Go finalizers
- [ ] **HOST-06**: Host records spawn lifecycle events (spawn-log / spawn-complete) via Go CLI subcommands
- [ ] **HOST-07**: Host either runs the selected workflow end-to-end or documents the exact blocker with a reproducible test

### Go Safety Invariants (SAFE)

- [ ] **SAFE-01**: Go remains sole authority for COLONY_STATE.json mutation
- [ ] **SAFE-02**: Go finalizers validate manifest provenance before any state write
- [ ] **SAFE-03**: Go locking and atomic write semantics unchanged by TS host presence
- [ ] **SAFE-04**: Install, update, publish commands remain pure Go — no TS involvement
- [ ] **SAFE-05**: Verification contracts (command-guide, parity tests, drift guards) still pass with TS host enabled
- [ ] **SAFE-06**: Existing `aether plan --plan-only` and `aether build --plan-only` behavior unchanged

### Follow-up Migration Map (MAP)

- [ ] **MAP-01**: Written follow-up plan exists for restoring Oracle/RALF confidence iteration
- [ ] **MAP-02**: Written follow-up plan exists for restoring swarm visibility
- [ ] **MAP-03**: Written follow-up plan exists for broader build/continue parity (all flows use TS host)
- [ ] **MAP-04**: Map includes phase numbers, estimated scope, and dependency ordering

## Deferred from v1.16 (Adaptive Caste Orchestration)

| Requirement | Original Description | Deferred Reason |
|-------------|---------------------|-----------------|
| ORCH-01 | Caste relevance registry with keywords/conditions | Superseded by hybrid architecture — caste selection moves to TS control plane |
| ORCH-02 | `queenOrchestrate()` function | Superseded — orchestration responsibility moves to TS host |
| ORCH-03 | Build flow adaptive dispatch | Superseded — build flow dispatch moves to TS host using Go manifests |
| ORCH-04 | Continue flow adaptive review | Superseded — continue review moves to TS host |
| ORCH-05 | Plan/colonize/swarm/seal adaptive dispatch | Superseded — all flows use TS host orchestration |
| ORCH-06 | Tests for caste spawning | Deferred until TS host stabilizes |
| ORCH-07 | Depth flag behavior preservation | Still valid — depth flags remain Go-owned overrides |
| ORCH-08 | Never-dispatched caste paths | Deferred — TS host will define dispatch paths in later milestone |

## Out of Scope

| Feature | Reason |
|---------|--------|
| Full TypeScript rewrite of Aether runtime | Non-goal — Go kernel remains authoritative |
| Restoring raw Bash state mutation | Non-goal — wrapper-owned state writes are unsafe |
| Maintaining Classic and Go as two products | Non-goal — Classic is baseline only, not a product |
| Moving install/update/publish to TypeScript | Non-goal — safety-critical release pipeline stays in Go |
| Making visual output parsing authoritative | Non-goal — JSON manifest contracts are the authority |
| Web UI for orchestration control | Future consideration — CLI-first for now |
| Cross-colony orchestration | Federation deferred to v2+ |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| BOUND-01 | Phase 106 | Pending |
| BOUND-02 | Phase 106 | Pending |
| BOUND-03 | Phase 106 | Pending |
| BASE-01 | Phase 107 | Pending |
| BASE-02 | Phase 107 | Pending |
| BASE-03 | Phase 107 | Pending |
| TEST-01 | Phase 108 | Complete |
| TEST-02 | Phase 108 | Complete |
| TEST-03 | Phase 108 | Complete |
| TEST-04 | Phase 108 | Complete |
| TEST-05 | Phase 108 | Complete |
| HOST-01 | Phase 109 | Pending |
| HOST-02 | Phase 109 | Pending |
| HOST-03 | Phase 109 | Pending |
| HOST-04 | Phase 109 | Pending |
| HOST-05 | Phase 109 | Pending |
| HOST-06 | Phase 109 | Pending |
| HOST-07 | Phase 109 | Pending |
| SAFE-01 | Phase 110 | Pending |
| SAFE-02 | Phase 110 | Pending |
| SAFE-03 | Phase 110 | Pending |
| SAFE-04 | Phase 110 | Pending |
| SAFE-05 | Phase 110 | Pending |
| SAFE-06 | Phase 110 | Pending |
| MAP-01 | Phase 111 | Pending |
| MAP-02 | Phase 111 | Pending |
| MAP-03 | Phase 111 | Pending |
| MAP-04 | Phase 111 | Pending |

**Coverage:**
- v1.16 requirements: 25 total
- Mapped to phases: 25
- Unmapped: 0

---
*Requirements defined: 2026-05-12*
*Last updated: 2026-05-12 after milestone pivot from adaptive caste to hybrid runtime recovery*
