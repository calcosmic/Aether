# Aether Milestones

## v1.26 Intelligent Orchestration (Shipped: 2026-08-22)

**Closeout type:** override_closeout — closed by owner decision on 2026-08-22 so the
next milestone (v1.27, priority-spec backlog) can start; three items carry forward.
**Known verification overrides:** 22 newly acknowledged, 0 carried forward from a
prior close (see STATE.md Deferred Items).

**Phases completed:** 172, 173, 174 (closed at plan 2 by owner ruling), 175 (shipped
outside the phase system), 180–184 (hardening items H1–H5), 186 (6/7), 187, 188, 189,
190, 191, 191.1 — 66 plans in the phase archive (`milestones/v1.26-phases/`), plus the
2026-08-20/21 sessions that landed outside GSD (team check-in, owner decision routing,
honest no-change results, the review-gate fixes). Product versions v1.0.47 → v1.0.63.

**Stats:** 571 commits, 1,223 files changed, +171,000 / −39,221 lines, 2026-08-08 → 2026-08-22.

**Key accomplishments:**

- **A wiring ratchet that cannot be argued past** — any program feature with no caller
  fails CI, seeded honestly at 278 existing orphans and allowed only to shrink; the
  release gate is proven by execution, not by reading its text (Phase 172).
- **Delegation guards that fail closed** — depth derived from the parent, a whole-run
  ceiling of 20 helpers, cycle detection, tamper-refusing ledgers (Phase 173).
- **Token measurement with stated arithmetic** — the 186× undercount found and fixed,
  cache tokens counted, estimates separated from measurements (Phase 174, plans 1–2).
- **Removing things until a small job costs a small amount** — Aether's own internals
  no longer leak into other projects' workers, skills are chosen by the task, castes
  with nothing to do are refused, the worker cap actually caps (Phases 180–184).
- **Crash-safe worktrees and one truth for failures** — no destruction path without a
  safety gate, one canonical failure log, atomic phase advance, pause cannot be
  clobbered (Phases 187, 188).
- **Workers get the whole brief and nothing twice** — merged dispatches carry every
  covered task; pheromones and handoffs reach a worker exactly once; 39 zero-reader
  config files and 8 dead commands deleted with reappearance ratchets (Phases 189–191).
- **Field hardening from three downstream repos in one day** — bundled workers can
  file honestly, the embedded checker reports real verdicts, out-of-band work has an
  honest re-entry (`aether verify-out-of-band`) that can never fabricate receipts
  (Phase 191.1).
- **The owner in the loop** — builds pause on a team card before spawning; workers
  route judgement calls to the owner instead of guessing; "nothing needed changing" is
  an honest result with evidence (2026-08-21, v1.0.62–63).

### Known Gaps (14 of 35 requirements unchecked at close)

- Carried into v1.27: SPEND-02, SPEND-07, SPEND-08 (the one-line cost summary —
  v1.27 feature 4); PROOF-01..04 (the benchmark — v1.27 feature 7).
- Dropped by recorded owner ruling 2026-08-14: SPEND-05, SPEND-06, SPAWN-09..11
  (worker delegation grant cut — "we are trying to reduce worker count").
- Deferred off the golden path 2026-08-17: SKILL-03, SKILL-04.

### Carried forward

- Phase 185 (one honest cost line) → v1.27 feature 4.
- Phase 186-07 (one live benchmark run, owner present) and Phase 192 (full showdown)
  → v1.27 feature 7, run on the improved system as the ratified order intended.
- Phase 172.1 (CI gate environment integrity) — blocks nothing; stays in the backlog.
- Phase 173 has one pending human check: a delegation refusal seen in a real run.

---

## v1.19 Roadmap Archive: v1.19 TypeScript Host Cutover + Oracle Confidence Recovery (Backfilled: 2026-08-21)

**Note:** Synthesized from archive snapshot by `/gsd-health --backfill`. Original completion date unknown.

---

## v1.18 : Hybrid Runtime Parity & Release Gate (Backfilled: 2026-08-21)

**Note:** Synthesized from archive snapshot by `/gsd-health --backfill`. Original completion date unknown.

---

## v1.17 Roadmap: Aether v1.17 Classic Restoration (Backfilled: 2026-08-21)

**Note:** Synthesized from archive snapshot by `/gsd-health --backfill`. Original completion date unknown.

---

## v1.14 Roadmap: Aether (Backfilled: 2026-08-21)

**Note:** Synthesized from archive snapshot by `/gsd-health --backfill`. Original completion date unknown.

---

## v1.13 Roadmap: Aether (Backfilled: 2026-08-21)

**Note:** Synthesized from archive snapshot by `/gsd-health --backfill`. Original completion date unknown.

---

## v1.24 Hybrid Architecture Salvage (Shipped: 2026-05-24)

**Phases completed:** 8 phases, 21 plans, 36 requirements

**Key accomplishments:**

1. Architecture boundary documented with hard rule: "Compiled code may execute behaviour, but editable assets must define behaviour"
2. Classic parity checklist created with 16 items covering all major subsystems, 80% verifiable
3. Behaviour extraction audit classified 162 Go symbols as KEEP_IN_GO, MOVE_TO_TS, MOVE_TO_YAML, etc.
4. 27 agent YAMLs, 28 prompt Markdowns, 9 phase YAMLs, 7 playbooks, and 8 policy YAMLs created under `colony/`
5. Go runtime refactored to load oracle directives and other behaviour from `colony/` files instead of hardcoded strings
6. TypeScript control plane bootstrapped with loaders, orchestrator, and 68 tests passing
7. NDJSON event stream established as shared observable truth between Go and TypeScript
8. End-to-end demo flow runs via `npm run aether:control`, emitting 48 events across init → plan → build lifecycle

**Stats:** 86 commits, 375 files changed, +45,732 / −507 lines, ~4 days

**Tech Debt:**

- `executePlan.test.ts` test isolation — create parent directory before writing COLONY_STATE.json
- Phases 152-156 missing VERIFICATION.md (completed before verification workflow was standard)
- REQUIREMENTS.md traceability table shows all requirements as "Pending" (bookkeeping only)

**Archives:**

- [Roadmap](milestones/v1.24-ROADMAP.md)
- [Requirements](milestones/v1.24-REQUIREMENTS.md)
- [Audit](milestones/v1.24-MILESTONE-AUDIT.md)

---

## v1.23 Daily Driver Reliability (Shipped: 2026-05-21)

**Phases completed:** 7 phases, 27 plans, 30 requirements

**Key accomplishments:**

1. Removed error suppression from 29+ data-persistence CLI calls — errors now surface honestly instead of silently dropping
2. Built command reliability matrix and `audit-catalog` classification report covering all 60+ commands with historical tags
3. Added tests for 12 HIGH-severity untested source files (eventbus, queen, instinct, midden, hive, spawn, autopilot, flags, council, shelf)
4. Restored learning lifecycle (hypothesis/validate/disprove) and all 9 flagship workflows with correct state mutations
5. Aligned Queen execution policy docs with Go runtime; verified 17 deterministic commands spawn zero agents
6. Added provider error classification, build-reconcile recovery, worktree merge-back, and orphan detection
7. Full lifecycle smoke test in downstream repo + TS host e2e tests + Go manifest integration tests

**Stats:** 41 commits, 213 files changed, +20,787 / −9,965 lines, ~2 days
**Go LOC:** ~208K | **TS Tests:** 512 | **Go Tests:** 2900+

**Tech Debt:** Missing VERIFICATION.md and SUMMARY.md process artifacts for phases 146–151 (documentation gaps, not work gaps)

**Archives:**

- [Roadmap](milestones/v1.23-ROADMAP.md)
- [Requirements](milestones/v1.23-REQUIREMENTS.md)
- [Audit](milestones/v1.23-MILESTONE-AUDIT.md)

---

## v1.22 Grounded Planning + Ceremony Restore (Shipped: 2026-05-19)

**Phases completed:** 4 phases, 8 plans, 22 requirements

**Key accomplishments:**

1. Unified 5 divergent skip lists into canonical ScanFilter with 36-entry noise map — survey now ignores .venv, caches, build artifacts
2. Source anchors extracted from clean survey (top 50 repo-owned files) wired through to planner context via anchors.json
3. Plan-grounding validation gate + decision conflict detection with 9 contradiction pairs — plans must reference real files
4. PlaybookLoader TS module restored ceremony — loads 5 build + 2 plan playbooks, YAML slimmed to packaging only
5. 84 new tests proving end-to-end pipeline: survey → anchors → grounding → ceremony → regression

**Stats:** 44 commits, 53 files changed, +7,396 / -151 lines, ~4 hours

**Tech Debt:** None. Zero anti-patterns.

**Archives:**

- [Roadmap](milestones/v1.22-ROADMAP.md)
- [Requirements](milestones/v1.22-REQUIREMENTS.md)
- [Audit](milestones/v1.22-MILESTONE-AUDIT.md)

---

## v1.21 Live Colony (Shipped: 2026-05-18)

**Phases completed:** 5 phases, 18 plans, 25 tasks

**Key accomplishments:**

- Verified real dispatch is default, --simulate is opt-in, and skill section injection handles all edge cases through 7 new tests
- Added plain English platform diagnostics and error classification with 16 new tests -- auth errors halt, timeouts continue, all messages are human-readable.
- Added dispatched runner with full build/plan/continue pipelines and ceremony -- 16 new tests across host.test.ts and host-integration.test.ts.
- Wired Oracle lifecycle for real dispatch with ceremony rendering and added --dry-run ceremony preview with DRY RUN badge -- 11 new tests across oracle-lifecycle.test.ts and host-integration.test.ts.
- Plan:
- One-liner:
- 1. [Rule 3 - Blocking] spawn-orchestrator.ts pre-committed from prior execution
- 1. [Rule 1 - Bug] Fixed pre-existing TypeScript errors in types.ts
- confidence-evaluator.ts
- Confidence-driven build iteration loop in host.ts with wave extraction, feedback injection, ceremony markers, and 7 integration tests
- E2E tests proving full iteration cycle with ceremony output, feedback injection, and diminishing returns detection
- Ceremony alignment validation tests confirming all dispatched command wrappers contain expected invocation instructions and parity matrix ceremony_steps are lifecycle-complete across all 18 commands
- Integration tests proving hive wisdom injection, spawn budget tracking, and confidence iteration compose correctly in the build pipeline, plus a plan-build-continue lifecycle end-to-end test
- Milestone audit test verifying all 31 v1.21 requirements map to test files, plus full regression confirmation across 464 TS tests and 18 Go packages

---

## v1.20 — Host Contract Hardening and Wrapper Reality Check

**Shipped:** 2026-05-15
**Phases:** 6 (130-135) | **Plans:** 7 | **Commits:** 14

### Accomplishments

1. **Host Surface Completeness** — All 7 `aether host` subcommands exist and execute end-to-end with documented flags
2. **Test Coverage for Host Commands** — 23 TS host tests exercising realistic flag combinations
3. **Wrapper Ownership Decision** — Wrappers are "host-assisted orchestrators"; boundary documented in ADR
4. **Lifecycle Honesty** — Simulation gated behind explicit `--simulate`; errors on missing platforms without it
5. **Oracle Storage Pattern** — Atomic Go-owned storage with interrupt recovery and stale pending auto-clear
6. **Doc-CLI Alignment Smoke Test** — Auto-discovers 60+ YAML commands, validates flags via Cobra introspection, blocks release gate on host-critical mismatches

### Key Decisions

- Wrappers call `aether host` for manifests but do not duplicate verification or mutate state
- Host-critical flag mismatches are blocking; non-host mismatches are warnings
- Oracle state uses the same atomic storage pattern as colony state

### Tech Debt

- `HostCriticalCombinations` hard-codes test cases (not auto-discoverable)
- `exactOptionalPropertyTypes` workaround in lifecycle tests

### Archives

- [Roadmap](milestones/v1.20-ROADMAP.md)
- [Requirements](milestones/v1.20-REQUIREMENTS.md)
- [Audit](milestones/v1.20-MILESTONE-AUDIT.md)

---

## v1.16 Hybrid Runtime Boundary and Orchestration Recovery (Shipped: 2026-05-13)

**Phases completed:** 6 phases, 9 plans, 13 tasks

**Key accomplishments:**

- v5.4.0 selected as Classic baseline with 16-module behavioral checklist extracted from actual git tag source code, 5 known limitations with workarounds, and classification matching Phase 106 contract (3 Restore in TS, 11 Keep in Go, 3 Obsolete)
- Standalone Bash smoke test that checks out Classic v5.4.0 in an isolated worktree and verifies colony init, state mutation across sync-state, 16-module inventory, wrapper command ceremony content, and CLI subcommand execution
- Four golden snapshot tests capturing plan/build/continue ceremony output and state mutation transitions across the full colony lifecycle
- TypeScript host scaffold with Go subprocess bridge, manifest type definitions, boundary enforcement, and nine integration tests
- Worker dispatch module with spawn-log/complete lifecycle recording via Go CLI, wave-grouped execution, and five integration tests against real Go binary
- Full lifecycle orchestrator driving plan->build->continue through Go manifests and finalizers, with 16 tests proving end-to-end completion and boundary enforcement
- 6 Go test functions proving Go remains sole state mutation authority and finalizer gatekeeper when TS host orchestrates worker dispatch
- Complete migration map for Oracle/RALF, Swarm Visibility, and Build/Continue Parity with 8 phases, 20 requirements, and sequential dependency ordering

---

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
