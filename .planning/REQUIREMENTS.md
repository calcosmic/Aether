# Requirements: Aether v1.7

**Defined:** 2026-04-24
**Core Value:** Aether should feel alive and truthful at runtime, not only look clever in wrappers or tests.

## v1.7 Requirements

### Plan Recovery

- [ ] **PLAN-01** (R069): `aether plan --force` succeeds when no real build work has happened, resetting phase status to allow replanning regardless of current phase state
- [ ] **PLAN-02** (R070): On `--force`, route-setter fallback artifacts are cleared so the route-setter can always write a fresh worker-written plan

### Timeout Policy

- [ ] **TIME-01** (R071): Default scout worker timeout raised from 5m to 10m, reducing premature fallback to local synthesis in larger repos

### Verification

- [ ] **TEST-01** (R072): E2E test proving: init → plan (with timeout/fallback) → `--force` replan → route-setter writes real plan, all without manual state manipulation

## Completed (Prior Milestones)

### v1.6 (R059-R068)
- Status: completed
- Summary: Release pipeline integrity — stable/dev publish sync, stale publish detection, release integrity checks, medic diagnostics, doc alignment, E2E regression, stuck-plan investigation.

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
| Fallback plan quality improvements | v1.7 is about recovery, not fallback plan quality |
| New agent castes | No new agents needed |
| Publish pipeline changes | v1.6 shipped that; not revisiting |
| UI/visual changes | Pipeline recovery is internal behavior |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| PLAN-01 (R069) | Phase 47: Plan Force Recovery | — |
| PLAN-02 (R070) | Phase 47: Plan Force Recovery | — |
| TIME-01 (R071) | Phase 47: Plan Force Recovery | — |
| TEST-01 (R072) | Phase 48: E2E Recovery Verification | — |

**Coverage:**
- v1.7 requirements: 4 total
- Mapped to phases: 4
- Satisfied: 0
- Unmapped: 0

---
*Requirements defined: 2026-04-24*
