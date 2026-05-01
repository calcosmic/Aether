# Aether Milestones

## v1.11 Aether Unification

**Shipped:** 2026-04-30
**Phases:** 70-79 (10 phases) | **Plans:** 18

### Accomplishments

1. Removed self-hosting artifacts (stale agents, duplicate commands, orphaned companion files)
2. Hardened 3-platform experience (PLAT-01 fix, Codex visual parity, wrapper updates)
3. Restored Smart Init intelligence (charter ceremony, rich init-research, suggest-analyze)
4. Built intelligence core (research rendering, circuit breaker events, data surfacing)
5. Improved user-facing flows (UX polish, ceremony data display, test coverage)
6. Documentation and validation hygiene (Nyquist compliance, empty summaries populated)

### Stats

- 59 commits, 1982 files changed, +361,826 / -77,084 lines
- 18/18 plans completed, 10/10 phases verified
- Timeline: 75 days (2026-02-14 to 2026-04-30)

### Known Tech Debt

- Phase 71: dispatch manifest test covers 1/25 agent types
- Phase 71: state-mutate --verify-only and --revert flags registered but never read by RunE
- Phase 71: suggest-approve returns hardcoded empty suggestions (compatibility stub)
- Phase 72: 2 human verification items pending
- Phase 76: 4 human verification items pending

---

## v1.10 Colony Polish

**Shipped:** 2026-04-28
**Phases:** 57-69 (14 phases) | **Plans:** 34

### Accomplishments

1. QUEEN.md pipeline fixed with normalized deduplication and extended section reading
2. Smart review depth system with CLI flags determining light/heavy review phases
3. Gate failure recovery wired into continue playbooks with recovery templates and skip logic
4. Oracle loop fix with research formulation, depth selection, and state persistence
5. Porter ant registered as 26th caste with visual identity and seal lifecycle wiring
6. Hive Brain promotion automatically wired into seal ceremony for high-confidence instincts
7. Idea shelving system with persistent colony backlog, auto-shelve at seal, and init surfacing
8. Full lifecycle ceremony across seal, init, status, entomb, resume, discuss, chaos, oracle, patrol

### Stats

- 204 commits, 452 files changed, +53,409 / -562 lines
- 35/35 requirements satisfied, 13/14 phases verified
- 18/18 cross-phase connections wired, 6/6 E2E flows complete
- Audit: tech_debt (5 non-critical items — no blockers)

### Known Tech Debt

- Phase 64.1 missing VERIFICATION.md
- OpenCode init.md + entomb.md missing shelf sections
- REQUIREMENTS.md checkboxes not ticked (bookkeeping only)

---

## v1.9 Review Persistence

**Shipped:** 2026-04-26
**Phases:** 52-56 (5 phases)

### Accomplishments

1. 7-domain review ledger CRUD with colony-prime injection
2. Review agent Write tools with scoped guardrails across 4 surfaces
3. Full review lifecycle (seal/entomb/status/init)

---

## v1.8 Colony Recovery

**Shipped:** 2026-04-25
**Phases:** 49-51 (3 phases)

### Accomplishments

1. Stuck-state scanner with 7 detection classes
2. Auto-repair pipeline with safe/destructive categorization
3. E2E recovery verification

---

## v1.7 Planning Pipeline Recovery

**Shipped:** 2026-04-24
**Phases:** 47-48 (2 phases)

### Accomplishments

1. Plan `--force` recovery from corrupted state
2. E2E recovery test coverage

---

## v1.6 Release Pipeline Integrity

**Shipped:** 2026-04-24
**Phases:** 39-46 (8 phases)

### Accomplishments

1. Publish hardening with integrity verification
2. E2E regression coverage
3. Codebase hygiene and command parity

---

## v1.5 Runtime Truth Recovery

**Shipped:** 2026-04-23
**Phases:** 31-38 (8 phases)
**Product:** v1.0.20

### Accomplishments

1. Continue unblock and dispatch fixes
2. Release decision pipeline
3. Nyquist validation backfill

---

## v1.4 Self-Healing Colony

**Shipped:** 2026-04-21
**Phases:** 25-30 (6 phases)

### Accomplishments

1. Medic ant with health scanning and repair
2. Ceremony integrity checks
3. Trace diagnostics

---

## v1.3 Visual Truth and Core Hardening

**Shipped:** 2026-04-21
**Phases:** 17-24 (8 phases)

### Accomplishments

1. Caste identity and visual UX restoration
2. Stage separators and ceremony markers
3. Emoji consistency and spawn lists

---

## v1.2 Live Dispatch Truth and Recovery

**Shipped:** 2026-04-20
**Phases:** 12-16 (5 phases)

### Accomplishments

1. Worker execution robustness and honest activity tracking
2. Verification-led continue with partial success
3. Recovery reconciliation and runtime UX

---

## v1.1 Trusted Context

**Shipped:** 2026-04-19
**Phases:** 7-11 (5 phases)

### Accomplishments

1. Context ledger and skill routing foundation
2. Prompt integrity and trust boundaries
3. Trust-weighted context assembly

---

## v1.0 MVP

**Shipped:** 2026-04-18
**Phases:** 1-6 (6 phases)

### Accomplishments

1. Colony ceremony and runtime visibility
2. Pheromone system with steering signals
3. Structural learning stack and curation

## v1.12 — Safe Colony (2026-05-01)

**Phases:** 8 | **Plans:** 16 | **Tasks:** 16+ | **Requirements:** 11/11 satisfied

### Delivered
- Loop-proof colony: 6 LOOP requirements covering watcher auto-skip, recovery redirect, circuit breaker, cycle detection, lifecycle exclusion, and telemetry
- Independent 3-level planning depth (light/standard/deep) with CLI flag and manifest integration
- Independent 3-level verification depth (light/standard/heavy) with 3-tier dispatch
- Smart depth defaults based on phase position and code change risk
- Depth selection UI with banner display and user override
- Depth persistence from plan through build to continue (resolveEffectiveContinueDepth)
- Code review fix: boolean flag preservation through depth resolution

### Tech Debt
- Phase 81 missing 81-01-SUMMARY.md (documentation only)
- Phase 85 missing both SUMMARY.md files (documentation only)
- REQUIREMENTS.md used text "Complete" instead of markdown [x] checkboxes
