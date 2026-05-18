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
- **v1.20 Host Contract Hardening and Wrapper Reality Check** - Phases 130-135 (shipped 2026-05-15) — [Archive](milestones/v1.20-ROADMAP.md)
- **v1.21 Live Colony** - Phases 136-140 (in progress)

## Phases

<details>
<summary>v1.0 through v1.20 Phase Summaries (archived)</summary>

See `.planning/milestones/` for full archived phase details.

</details>

### Current Milestone: v1.21 Live Colony

**Goal:** Make the TypeScript host the real production orchestrator for Aether -- genuine worker dispatch, visible ceremony, confidence-driven iteration, worker escalation, and reusable wisdom instead of simulation.

- [x] **Phase 136: Production Foundation** - Remove simulation guards, prove real dispatch for build/plan/continue/oracle, add ceremony, skill section verification, dry-run and error diagnostics *(completed 2026-05-18)*
- [ ] **Phase 137: Hive Wisdom Injection** - Call Go hive-read from TS host, inject wisdom into worker prompts, prove graceful degradation and domain tag expansion
- [x] **Phase 138: Worker-to-Worker Spawning** - Workers request child workers mid-build with budget enforcement, spawn depth limits, and result attachment (completed 2026-05-18)
- [ ] **Phase 139: Confidence-Driven Build Iteration** - Build coordinator loops dispatch/evaluate up to 3 iterations with diminishing returns detection and cumulative budget
- [x] **Phase 140: Hardening and Validation** - End-to-end integration tests, wrapper alignment verification, milestone audit (completed 2026-05-18)

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 136. Production Foundation | v1.21 | 4/4 | Complete | 2026-05-18 |
| 137. Hive Wisdom Injection | v1.21 | 0/3 | Not started | - |
| 138. Worker-to-Worker Spawning | v1.21 | 4/4 | Complete   | 2026-05-18 |
| 139. Confidence-Driven Build Iteration | v1.21 | 0/? | Not started | - |
| 140. Hardening and Validation | v1.21 | 3/3 | Complete    | 2026-05-18 |

## Phase Details

### Phase 136: Production Foundation
**Goal**: The TypeScript host dispatches real platform workers for build, plan, continue, and oracle workflows -- not simulation. Ceremony renders with real caste visuals, and skill sections inject correctly.
**Depends on**: Nothing (first phase of milestone)
**Requirements**: HOST-01, HOST-02, HOST-03, HOST-04, HOST-05, HOST-06, HOST-07, HOST-08, HOST-09, SKILL-01, SKILL-02, SKILL-03
**Success Criteria** (what must be TRUE):
  1. Running `aether host build` dispatches real Claude/OpenCode/Codex workers that produce observable file changes, not simulated claims
  2. Running `aether host plan` dispatches real Scout/Route-Setter workers and commits a real plan to colony state
  3. Running `aether host continue` runs real review agent workers and verifies against actual build artifacts
  4. Running `aether host oracle` dispatches real Oracle workers with Go-computed confidence driving loop termination
  5. Ceremony output shows caste emoji, ANSI colors, and stage markers during real dispatch (spawn-plan, wave-start, worker-complete, closeout)
  6. `--dry-run` renders ceremony and manifest without spawning workers; `--simulate` retains old behavior and passes existing smoke test
  7. Missing platform CLI produces a clear diagnostic error identifying which platform is missing, not a generic TS host crash
  8. Worker prompts include a skill section when the Go manifest provides `skill_section`, and omit it cleanly when not present, with no prompt assembly failures from malformed skill data
**Plans:** 4 plans (4 waves, sequential due to file dependencies)

Plans:
**Wave 1**
- [x] 136-01-PLAN.md -- Flip simulation default, verify --simulate retention, add skill section tests

**Wave 2** *(blocked on Wave 1 completion)*
- [x] 136-02-PLAN.md -- Platform error diagnostics, error classification for auth vs timeout

**Wave 3** *(blocked on Wave 2 completion)*
- [x] 136-03-PLAN.md -- Real dispatch pipelines for build, plan, and continue with ceremony

**Wave 4** *(blocked on Wave 3 completion)*
- [x] 136-04-PLAN.md -- Oracle real dispatch and --dry-run ceremony preview

### Phase 137: Hive Wisdom Injection
**Goal**: Colony-prime reads cross-colony hive wisdom during prompt assembly and injects relevant wisdom into worker context, proving that colony B benefits from colony A's learnings.
**Depends on**: Phase 136 (real dispatch must work before wisdom injection matters)
**Requirements**: HIVE-01, HIVE-02, HIVE-03, HIVE-04, HIVE-05, HIVE-06, HIVE-07
**Success Criteria** (what must be TRUE):
  1. Worker prompts contain a "Hive Wisdom" context section when `hive-read` returns matching entries, formatted for worker consumption
  2. A colony with no hive wisdom data functions identically to one with it -- no errors, no empty sections, no behavioral difference
  3. If `hive-read` fails (Go command error, missing hive directory), dispatch proceeds normally with a warning logged -- workers never blocked by hive
  4. Colony B completes a build faster or with fewer retries than colony A (without wisdom) on the same task, when colony A's learnings have been promoted to the hive
  5. Domain tags include technology stack details (e.g., "go", "typescript", "cli") and partial domain matches receive a relevance discount compared to exact matches
**Plans:** 3 plans (3 waves, sequential due to file dependencies)

Plans:
**Wave 1**
- [ ] 137-01-PLAN.md -- Create hive-injector.ts module, types, and unit tests (HIVE-01, HIVE-02, HIVE-06, HIVE-07)

**Wave 2** *(blocked on Wave 1 completion)*
- [ ] 137-02-PLAN.md -- Integrate hive wisdom into prompt-assembler and worker-dispatch pipeline (HIVE-03)

**Wave 3** *(blocked on Wave 2 completion)*
- [ ] 137-03-PLAN.md -- Integrate hive-read into dispatched runners in host.ts with graceful degradation and integration tests (HIVE-04, HIVE-05)

### Phase 138: Worker-to-Worker Spawning
**Goal**: Workers can request additional workers mid-build through the `spawns` field, with orchestrator-level budget enforcement, spawn depth limits, and child results attached to parent handoffs.
**Depends on**: Phase 136 (real dispatch path must be stable before extending it)
**Requirements**: SPAWN-01, SPAWN-02, SPAWN-03, SPAWN-04, SPAWN-05, SPAWN-06
**Success Criteria** (what must be TRUE):
  1. A worker that returns `spawns[]` in its claims causes the TS host to dispatch child workers with the same platform dispatch path
  2. Sub-workers can also request spawns, but grandchildren are rejected and logged -- maximum spawn depth of 2 enforced
  3. All child spawns reduce the available Queen spawn budget -- total workers (manifest + spawned) never exceed the budget
  4. Child spawn results appear in the parent worker's handoff, visible to downstream workers in the same wave
  5. Spawn requests that would exceed the remaining budget are logged and skipped, never queued -- fail-closed behavior
  6. Go `spawn-log` and `spawn-complete` record child entries with parent worker references, not just "Queen"
**Plans:** 4/4 plans complete

Plans:
- [ ] (to be planned)

### Phase 139: Confidence-Driven Build Iteration
**Goal**: The build coordinator loops plan/dispatch/evaluate up to 3 iterations with diminishing returns detection, cumulative budget tracking, and visible iteration progress in ceremony output.
**Depends on**: Phase 136 (real dispatch), Phase 137 (hive wisdom informs iteration quality), Phase 138 (spawning may be needed during iteration)
**Requirements**: ITER-01, ITER-02, ITER-03, ITER-04, ITER-05, ITER-06
**Success Criteria** (what must be TRUE):
  1. A build that fails verification on first pass is automatically re-dispatched with specific feedback from Go finalizer gate results, without user intervention
  2. The loop stops early when confidence delta is below 5% for 2 consecutive iterations, even if the hard cap of 3 has not been reached
  3. Worker budget is cumulative across iterations -- if iteration 1 uses 70% of budget, iteration 2 only has 30% remaining, preventing runaway token spend
  4. Ceremony output shows iteration number, current confidence score, and delta from the previous iteration after each evaluation
  5. Confidence metrics come from Go finalizer gate results (test pass rate, coverage, quality scores), not from worker self-reports
**Plans:** 4 plans (4 waves, sequential due to file dependencies)

Plans:
- [ ] (to be planned)

### Phase 140: Hardening and Validation
**Goal**: End-to-end integration tests prove the full production pipeline works, wrapper alignment is verified, and the milestone is audit-ready for ship.
**Depends on**: Phases 136-139 (all features must be complete)
**Requirements**: (validation phase -- covers gap-filling and cross-cutting concerns from all categories)
**Success Criteria** (what must be TRUE):
  1. An end-to-end test runs `aether host plan` through `aether host build` through `aether host continue` with --simulate (the test-safe execution mode) and verifies ceremony output, state mutations, and completion files
  2. Wrapper commands (Claude plan/build/continue) produce equivalent ceremony and state outcomes to direct `aether host` invocations
  3. No regression in existing TS test suite (220+ tests) or Go test suite (2900+ tests)
  4. Temp files and completion artifacts are cleaned up after normal completion and after graceful shutdown
**Plans:** 3/3 plans complete

Plans:
**Wave 1**
- [x] 140-01-PLAN.md -- Wrapper ceremony alignment tests for all commands (D-01, D-02, D-03)

**Wave 2** *(blocked on Wave 1 completion)*
- [x] 140-02-PLAN.md -- Cross-phase integration tests combining hive + spawn + iteration pipeline

**Wave 3** *(blocked on Wave 2 completion)*
- [x] 140-03-PLAN.md -- Milestone audit test and full regression verification
