# Requirements: Aether v1.21 Live Colony

**Defined:** 2026-05-18
**Core Value:** Aether should feel alive and truthful at runtime, not only look clever in wrappers or tests.

## v1.21 Requirements

Requirements for the Live Colony milestone. Each maps to roadmap phases starting at 136.

### Production Host (HOST)

- [ ] **HOST-01**: TS host dispatches real platform workers (Claude, OpenCode, Codex) with `simulateWorkers=false` as default, `--simulate` as opt-in
- [ ] **HOST-02**: Build workers execute in parallel waves with dependency ordering, completion files written to tmpdir, and Go build-finalizer called atomically
- [ ] **HOST-03**: Plan workers (Scout, Route-Setter) execute through the TS host with real platform dispatch and plan-finalizer commits real plan to colony state
- [ ] **HOST-04**: Continue verification runs through the TS host with real worker dispatch for review agents
- [ ] **HOST-05**: Oracle lifecycle dispatches real Oracle workers (not simulated) with Go-computed confidence driving loop termination
- [ ] **HOST-06**: Production ceremony renders to stderr (spawn-plan, wave-start, worker-complete, closeout) with caste emoji, ANSI colors, and stage markers for all real dispatches
- [ ] **HOST-07**: `--dry-run` flag available on all host commands: executes manifest fetch and ceremony rendering without spawning worker subprocesses
- [ ] **HOST-08**: Missing platform CLI produces clear error with Go-owned provider diagnostic, not a generic TS host error
- [ ] **HOST-09**: `--simulate` flag retains simulation behavior for testing; existing lifecycle smoke test still passes

### Worker Escalation (SPAWN)

- [x] **SPAWN-01**: Workers return `spawns[]` field in claims; TS host consumes it to dispatch child workers with budget enforcement
- [x] **SPAWN-02**: Maximum spawn depth of 2 (workers can spawn sub-workers, sub-workers cannot spawn further)
- [x] **SPAWN-03**: All child spawns count against the Queen spawn budget (sub-workers reduce available budget for manifest workers)
- [x] **SPAWN-04**: Child spawn results attach to parent worker's handoff for downstream worker visibility
- [x] **SPAWN-05**: Go `spawn-log` and `spawn-complete` record child spawn entries with parent reference (not just "Queen")
- [x] **SPAWN-06**: Spawn requests exceeding budget are logged and skipped, not queued (fail-closed, not fail-open)

### Confidence Iteration (ITER)

- [ ] **ITER-01**: Build coordinator loops `build --plan-only → dispatch → finalize → evaluate` up to 3 iterations maximum
- [ ] **ITER-02**: Confidence metrics come from Go finalizer gate results, not worker self-reports
- [ ] **ITER-03**: Diminishing returns detection stops the loop early when confidence delta is below 5% for 2 consecutive iterations
- [ ] **ITER-04**: Reusable `ConfidenceLoop` class extracted from Oracle lifecycle, applicable to plan, build, and continue workflows
- [ ] **ITER-05**: Cumulative worker budget across iterations (not per-iteration), preventing runaway token spend
- [ ] **ITER-06**: Iteration progress visible in ceremony output (iteration number, current confidence, delta from last)

### Hive Wisdom (HIVE)

- [ ] **HIVE-01**: `hive-injector.ts` calls `aether hive-read` during prompt assembly and formats results for worker context injection
- [ ] **HIVE-02**: Colony-prime reads hive wisdom scoped by domain tags and injects it into worker prompts as context
- [ ] **HIVE-03**: First-time colonies without hive wisdom work identically to colonies with it (graceful degradation)
- [ ] **HIVE-04**: `hive-read` failure does not block dispatch; warning logged, workers proceed without hive context
- [ ] **HIVE-05**: Integration test proves colony B completes a build faster or with fewer retries than colony A without wisdom, using the same task
- [ ] **HIVE-06**: Domain tags expanded to include technology stack details (e.g., "go", "typescript", "cli") for better relevance matching
- [ ] **HIVE-07**: Partial domain matches receive relevance discount (exact match = full confidence, partial = reduced)

### Skill Verification (SKILL)

- [ ] **SKILL-01**: Worker prompts contain skill section when Go manifest dispatch includes `skill_section`
- [ ] **SKILL-02**: Worker prompts omit skill section when dispatch has no `skill_section` (no empty section headers)
- [ ] **SKILL-03**: Skill injection never fails the prompt assembly; missing or malformed skill section is handled gracefully

## Future Requirements

Deferred to v1.22+.

### Multi-Level Spawn Trees
- Grandchild workers (depth > 2) with deadlock detection
- User-defined spawn policies

### Confidence Enhancement
- Learned confidence thresholds based on historical colony data
- Per-task confidence scoring (not just per-build)

### Hive Enhancement
- Cross-colony ledger sharing (blocked by code-specific path staleness)
- Vector embeddings for semantic wisdom search
- Automatic domain tag inference from codebase analysis

## Out of Scope

| Feature | Reason |
|---------|--------|
| Cross-colony ledger sharing | Code-specific paths go stale across repos |
| Ledger web UI | CLI-only for now |
| Self-mutating agents | Explicit deferral in PROJECT.md |
| New platforms | Claude, OpenCode, Codex are the maintained surfaces |
| Worktree merge-back automation | Out of scope per v1.20 non-goals |
| Interactive shell | Deferred to future milestone |

## Traceability

| REQ-ID | Phase | Status |
|--------|-------|--------|
| HOST-01 | Phase 136 | Pending |
| HOST-02 | Phase 136 | Pending |
| HOST-03 | Phase 136 | Pending |
| HOST-04 | Phase 136 | Pending |
| HOST-05 | Phase 136 | Pending |
| HOST-06 | Phase 136 | Pending |
| HOST-07 | Phase 136 | Pending |
| HOST-08 | Phase 136 | Pending |
| HOST-09 | Phase 136 | Pending |
| SPAWN-01 | Phase 138 | Complete |
| SPAWN-02 | Phase 138 | Complete |
| SPAWN-03 | Phase 138 | Complete |
| SPAWN-04 | Phase 138 | Complete |
| SPAWN-05 | Phase 138 | Complete |
| SPAWN-06 | Phase 138 | Complete |
| ITER-01 | Phase 139 | Pending |
| ITER-02 | Phase 139 | Pending |
| ITER-03 | Phase 139 | Pending |
| ITER-04 | Phase 139 | Pending |
| ITER-05 | Phase 139 | Pending |
| ITER-06 | Phase 139 | Pending |
| HIVE-01 | Phase 137 | Pending |
| HIVE-02 | Phase 137 | Pending |
| HIVE-03 | Phase 137 | Pending |
| HIVE-04 | Phase 137 | Pending |
| HIVE-05 | Phase 137 | Pending |
| HIVE-06 | Phase 137 | Pending |
| HIVE-07 | Phase 137 | Pending |
| SKILL-01 | Phase 136 | Pending |
| SKILL-02 | Phase 136 | Pending |
| SKILL-03 | Phase 136 | Pending |

*Coverage: 31/31 requirements mapped to phases 136-140.*
