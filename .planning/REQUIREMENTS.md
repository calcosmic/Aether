# Requirements: Aether v1.6

**Defined:** 2026-04-23
**Core Value:** Aether should feel alive and truthful at runtime, not only look clever in wrappers or tests.

## v1.6 Requirements

### Publish Integrity

- [ ] **PUB-01** (R059): Stable publish updates stable binary and stable hub to the same version atomically
- [ ] **PUB-02** (R060): Dev publish updates only `aether-dev` binary and `~/.aether-dev` hub with zero stable contamination
- [x] **PUB-03** (R061): Downstream `aether update --force` detects and reports stale or incomplete publishes instead of silently succeeding
- [ ] **PUB-04** (R061): Downstream `aether-dev update --force` detects and reports stale or incomplete publishes instead of silently succeeding

### Release Validation

- [x] **REL-01** (R062): Release integrity check validates source version, binary version, hub version, companion-file surfaces, and downstream update result together
- [x] **REL-02** (R063): Medic/dedicated diagnostics flag incomplete stable and dev publishes with exact recovery commands
- [ ] **REL-03** (R064): Operations guide, publish-update-runbook, and AGENTS.md match actual runtime behavior exactly
- [ ] **REL-04** (R065): End-to-end regression coverage for both stable and dev publish/update flows

### Evidence & Consistency

- [ ] **EVD-01** (R066): Archived release and milestone evidence is internally consistent — no contradictions after ship
- [ ] **EVD-02** (R067): Verify whether stuck `aether plan` issue still reproduces in freshly updated stable and dev repos; if yes, fix with regression test

### OpenCode Blocker

- [ ] **OPN-01** (R068): Aether ships valid OpenCode agent frontmatter — OpenCode startup in downstream repos no longer crashes

## Completed (Prior Milestones)

### v1.5 (R045-R058)
- Status: completed
- Summary: Runtime truth recovery — worker dispatch honesty, continue truth, git-verified claims, atomic state, cleanup, platform parity, release decision.

### v1.3-v1.4 (R027-R044)
- Status: completed
- Summary: Visual UX restoration, core path hardening, recovery, trace logging, medic ant, ceremony integrity, self-healing colony features.

## Deferred

- **PERF-01** (R016): Pheromone markets and reputation exchange
- **FED-01** (R017): Federation and inter-colony coordination
- **EVO-01** (R018): Evolution engine / self-modifying agents

## Out of Scope

| Feature | Reason |
|---------|--------|
| goreleaser replacement | Current release tooling works; fix the pipeline around it |
| npm package restructuring | Companion file delivery works; fix version sync |
| New agent castes | No new agents needed for pipeline integrity |
| Feature expansion | v1.6 is pipeline integrity only — no new colony features |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| OPN-01 (R068) | Phase 39: OpenCode Agent Frontmatter Fix | Pending |
| PUB-01 (R059) | Phase 40: Stable Publish Hardening | Pending |
| PUB-02 (R060) | Phase 41: Dev-Channel Isolation | Pending |
| PUB-03 (R061) | Phase 44.2: Command Hygiene and Agent Parity | Complete (44.2-01) |
| PUB-04 (R061) | Phase 42: Downstream Stale-Publish Detection | Pending |
| REL-01 (R062) | Phase 44.2: Command Hygiene and Agent Parity | Complete (44.2-01) |
| REL-02 (R063) | Phase 43: Release Integrity Checks and Diagnostics | Complete (43-02) |
| REL-03 (R064) | Phase 44: Doc Alignment and Archive Consistency | Pending |
| EVD-01 (R066) | Phase 44: Doc Alignment and Archive Consistency | Pending |
| REL-04 (R065) | Phase 45: End-to-End Regression Coverage | Pending |
| EVD-02 (R067) | Phase 46: Stuck-Plan Investigation and Release Decision | Pending |

**Coverage:**
- v1.6 requirements: 11 total
- Mapped to phases: 11
- Unmapped: 0

---
*Requirements defined: 2026-04-23*
*Last updated: 2026-04-23 after milestone v1.6 definition*
