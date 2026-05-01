# Project Research Summary

**Project:** Aether v1.13 -- Recovery Hardening & Hive Learning
**Domain:** AI colony framework -- build/continue gate hardening, confidence-targeted research loops, SQLite-backed procedural memory
**Researched:** 2026-05-01
**Confidence:** HIGH

## Executive Summary

v1.13 adds two capability layers to Aether's existing Go CLI colony runtime. The first is **recovery hardening**: preventing phantom build advancement (builds that claim success but produced nothing), adding confidence-targeted Oracle planning, converting raw init research into approval-ready briefs, and replacing STOP-wall gate failures with fix/unblock paths plus a new Fixer caste. The second is **hive learning**: a repo-scoped colony memory store with SQLite-backed FTS5 recall, pheromone skills (auto-created procedural memory from verified difficult tasks), Keeper curator with usage tracking and stale detection, and evidence-gated learning triggers that only promote verified successful work into durable wisdom.

The recommended approach is **maximally additive** -- all 31 work packages extend existing infrastructure rather than replacing it. The only new external dependency is `modernc.org/sqlite` (pure Go, no CGO), which means zero impact on cross-compilation or single-binary distribution. JSON files remain authoritative for colony state; SQLite serves purely as a secondary search index for accumulated learning. Every new field uses `omitempty` for backward compatibility. The key risks are provenance validation producing false negatives in worktree mode, Oracle confidence gaming by LLM workers, and the Fixer caste creating infinite gate-recovery loops. All three have concrete prevention strategies documented below.

## Key Findings

### Recommended Stack

v1.13 requires exactly one new external dependency. Everything else builds on existing Go stdlib patterns already proven in the codebase (2900+ tests).

**Core technologies:**
- `modernc.org/sqlite` v1.42.1: Colony memory store with FTS5 recall -- pure Go (no CGO), standard `database/sql` interface, FTS5 built-in, trivial cross-compilation. The only new dep.
- Existing `pkg/storage.Store`: JSON persistence for colony state -- remains authoritative, SQLite is secondary index only.
- Existing `pkg/events.Bus`: Event bus with JSONL persistence -- learning hooks subscribe to new lifecycle topics (`phase.completed`, `seal.completed`, `gate.passed`).
- Existing `pkg/memory` trust scoring: 40/35/25 weighted rubric, 7 trust tiers, 60-day half-life -- extended for evidence-gated learning, not replaced.

### Expected Features

**Must have (table stakes):**
- Build provenance validation (A1) -- prevents phantom advancement where builds claim success but produce zero changes. Core trust issue.
- Recoverable gate failures + /ant-unblock (A4/A5) -- current STOP walls with text instructions are poor UX for an automated system.
- Evidence-gated learning (B5) -- learning from unverified work creates noise that degrades trust in instincts.
- Privacy gate for learning writes (B6) -- writing secrets to colony memory is a security risk.

**Should have (competitive):**
- Fixer caste (A6) -- self-healing gate failures where the colony fixes its own blockers. Strong differentiator.
- Confidence-targeted Oracle (A2) -- iterative research that quantifies confidence and targets a threshold. No other AI colony framework does this.
- Init synthesis (A3) -- converts raw scouting data into an approval-ready launch brief.
- Pheromone skills (B3) -- auto-created procedural memory from verified difficult tasks.
- Keeper curator (B4) -- automated memory hygiene with stale detection and archival.

**Defer (v2+):**
- Worker process lifecycle with heartbeats (A7) -- high cost, moderate value. Current wall-clock timeout suffices until workers run long enough to stall.
- SQLite + FTS recall (B2) as standalone feature -- defer until B1 (unified memory API) proves the volume justifies a database index.

### Architecture Approach

The architecture is additive across three well-defined integration surfaces. Recovery hardening modifies two critical paths: build-finalize (between `applyCodexBuildState` and manifest write) and continue-finalize (between `assessCodexContinue` and gate run). A new `cmd/provenance.go` provides validation logic that both paths call. The Oracle loop is extended in-place with a `--confidence-target` flag and iterative refinement that re-opens low-confidence questions. Hive learning introduces a new `pkg/hive/` package with 6 files (store, recall, hooks, privacy, skill, curator), connected to the existing memory pipeline via event bus subscriptions. The privacy gate intercepts all writes to SQLite before data is stored.

**Major components:**
1. `cmd/provenance.go` (new) -- Build provenance validation that checks claimed files exist and git diff matches. Called by both build-finalize and continue-finalize.
2. `cmd/unblock_cmd.go` (new) + gate-results.json mirror -- Structured gate recovery with /ant-unblock command, per-phase gate results with retry metadata.
3. Fixer caste (new agent, 27th) -- Reads gate failure context, investigates root cause, applies fix, verifies, reports. Dispatched by continue-finalize when gates fail with fixable recovery templates.
4. `pkg/hive/` (new package) -- SQLite-backed learning store with FTS5 recall, privacy gate on all writes, event bus hooks for automatic learning capture, skill lifecycle management, and Keeper curator for memory hygiene.
5. Oracle confidence loop (extended) -- User-settable confidence target, iterative refinement, context capsule cap, external validation requirements for high confidence scores.

### Critical Pitfalls

1. **Build provenance false negatives in worktree mode** -- Workers write to isolated worktrees, but provenance validation resolves paths against the main root. Fix: validate against the worktree root during execution, re-validate after sync-back. Use `git ls-files --others --exclude-standard` for untracked files.
2. **Oracle confidence gaming by LLM workers** -- Self-reported confidence scores are unreliable because LLMs optimize for task completion. Fix: add external validation (finding diversity, code-level evidence requirements), confidence decay when no progress across iterations, context capsule cap at 4000 chars.
3. **Fixer caste creating infinite gate-recovery loops** -- Circuit breaker tracks worker names, but Fixer gets a new deterministic name each spawn so it never trips. Fix: add gate-level circuit breaker (not worker-level), cap Fixer at 2 attempts per gate, require Fixer to run ALL gates after its fix.
4. **SQLite coexistence with JSON file storage** -- WAL mode creates auxiliary files that could be committed to git. Fix: store DB in `.aether/data/hive/` with its own `.gitignore` excluding all files. Use `SetMaxOpenConns(1)` for writes. Never use `FileLocker` on SQLite files.
5. **False learning confidence from lucky passes** -- Gates passing does not mean work is correct, only that tests pass and no critical flags exist. Fix: require multi-repo confirmation for confidence above 0.7, add privacy gate to learning injection that strips cross-repo specifics.

## Implications for Roadmap

Based on research, suggested phase structure:

### Phase 1: Recovery Foundation
**Rationale:** Build provenance is the most critical trust issue -- without it, phantom advancement undermines the entire build/continue contract. Gate state persistence and /ant-unblock are low-cost extensions of existing gate.go logic. Privacy gate is a security baseline needed before any learning features write data.
**Delivers:** Provenance validation at build-complete and continue-verify, gate-results.json mirror with per-phase scoping, /ant-unblock command, privacy/secret scanning for learning writes.
**Addresses:** A1 (provenance), A4/A5 (recoverable gates), B6 (privacy gate).
**Avoids:** Pitfall 1 (provenance false negatives) by implementing checkpoint-based diff and timestamp validation.

### Phase 2: Gate Self-Healing
**Rationale:** The Fixer caste and smart gate retry build directly on Phase 1's gate persistence. The Fixer needs gate failure context (from /ant-unblock infrastructure) to function. Oracle confidence targeting is independent and can build in parallel. Init synthesis is independent and improves onboarding UX.
**Delivers:** Fixer caste (27th agent) across all 4 surfaces, smart gate retry with cooldowns, confidence-targeted Oracle loop, init synthesis step.
**Addresses:** A6 (Fixer), A2 (Oracle), A3 (init synthesis).
**Uses:** Gate state persistence from Phase 1 for Fixer context.
**Avoids:** Pitfall 3 (Fixer infinite loops) by implementing gate-level circuit breaker and Fixer attempt cap.

### Phase 3: Learning Foundation
**Rationale:** Evidence-gated learning triggers depend on Phase 1 provenance validation for evidence. The unified memory API is the foundation for all remaining B features. Repo isolation is low-cost and extends existing hive promotion.
**Delivers:** Evidence-gated learning triggers at build/continue complete, unified colony memory API, repo isolation with hive opt-in/opt-out, learning hooks connected to event bus.
**Addresses:** B5 (evidence rules), B1 (unified memory), B7 (repo isolation).
**Uses:** Provenance validation from Phase 1 for evidence verification.
**Avoids:** Pitfall 5 (false learning confidence) by requiring multi-repo confirmation and implementing semantic dedup.

### Phase 4: Hive Intelligence
**Rationale:** SQLite integration, FTS5 recall, pheromone skills, and Keeper curator all depend on Phase 3's unified memory API. This is the highest-complexity phase and should come last when the foundation is stable.
**Delivers:** SQLite colony.db with WAL mode and FTS5, full-text recall for worker context injection, auto-created pheromone skills from verified difficult tasks, Keeper curator with usage tracking and stale detection.
**Addresses:** B2 (SQLite + FTS), B3 (pheromone skills), B4 (Keeper curator).
**Uses:** Unified memory API from Phase 3, privacy gate from Phase 1.
**Avoids:** Pitfall 4 (SQLite coexistence) by using dedicated `.aether/data/hive/` directory with proper gitignore, single-writer pattern, and FTS health check in patrol.

### Phase 5: System Hardening (optional/deferrable)
**Rationale:** Worker process lifecycle (heartbeats, PID tracking, stale detection) is the only feature with no strong dependency chain. It can be deferred entirely (recommended) or built after Phase 4 if worker stalls become a real problem in practice.
**Delivers:** Worker heartbeat monitoring, PID tracking in colony state, stale worker cleanup during execution.
**Addresses:** A7 (worker lifecycle).
**Avoids:** Pitfall 6 (PID recycling) by using process groups instead of individual PIDs.

### Phase Ordering Rationale

- Phase 1 first because provenance validation is the trust foundation everything else builds on, and privacy gate is a security prerequisite.
- Phase 2 groups gate self-healing and Oracle/init improvements -- they are independent of each other but both depend on Phase 1 gate infrastructure.
- Phase 3 introduces the learning pipeline foundation -- evidence rules need provenance, unified memory is the API layer for all B features.
- Phase 4 is the SQLite heavy-lift -- deferred as late as possible because it introduces the only new dependency and has the highest complexity.
- Phase 5 is explicitly optional/deferrable because current wall-clock timeouts are adequate for most use cases.

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 2 (Fixer caste):** Agent prompt design for gate investigation is novel -- no existing agent does root-cause analysis on gate failures. Needs prompt engineering research.
- **Phase 4 (SQLite integration):** FTS5 external content pattern with sync triggers needs validation against modernc.org/sqlite specifics. Schema migration strategy across Aether versions needs design.
- **Phase 4 (Pheromone skills):** Auto-creating skills from task context requires a template design for procedural memory. No existing pattern in the codebase for this.

Phases with standard patterns (skip research-phase):
- **Phase 1 (Recovery foundation):** Provenance validation, gate persistence, and privacy scanning all extend existing patterns with well-understood implementations.
- **Phase 3 (Learning foundation):** Evidence rules are threshold checks on existing gate/claim data. Unified memory API is CRUD over existing JSON files.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | `modernc.org/sqlite` v1.42.1 verified via Context7 docs, pkg.go.dev, and community sources. FTS5 support confirmed. Pure Go eliminates CGO concerns. |
| Features | HIGH | All 31 work packages mapped to existing code with specific integration points (file paths, line numbers). No feature requires unknown technology. |
| Architecture | HIGH | Based on direct analysis of 316 cmd/*.go files and 12 pkg/ packages. All integration surfaces identified with specific hook points. Additive pattern verified. |
| Pitfalls | HIGH | All 15 pitfalls derived from direct codebase analysis. False negatives in provenance validation, Oracle confidence gaming, and Fixer loops are verified risks with concrete reproduction paths. |

**Overall confidence:** HIGH

### Gaps to Address

- **SQLite FTS5 with modernc.org/sqlite specifically:** FTS5 is confirmed for SQLite generally, but the exact behavior of external content tables and sync triggers with the modernc.org/sqlite pure-Go driver should be validated with a quick spike in Phase 4 planning.
- **Fixer caste prompt design:** No existing agent performs automated gate failure investigation. The prompt engineering for root-cause analysis, fix proposal, and verification is uncharted territory for Aether. Prototype during Phase 2 planning.
- **Context capsule size under triple injection:** Colony-prime (8K) + skills (8K) + learned context (proposed 2K) = 18K chars of injected context. Research recommends a 20K total budget, but the actual impact on worker quality needs validation during Phase 4 when hive recall is wired into colony-prime.
- **Worktree mode provenance:** All four research files identify worktree parallel mode as a provenance risk. The fix (validate against worktree root, re-validate after sync-back) is straightforward but needs explicit test coverage.

## Sources

### Primary (HIGH confidence)
- Direct codebase analysis: 316 `cmd/*.go` files, 12 `pkg/` packages (2026-05-01)
- `pkg/memory/trust.go` -- Trust scoring engine (40/35/25 weighted, 7 tiers, half-life)
- `cmd/codex_build_finalize.go` -- Build-finalize path (lines 144-457)
- `cmd/codex_continue_finalize.go` -- Continue-finalize path (lines 115-226)
- `cmd/oracle_loop.go` -- Oracle RALF loop with confidence tracking
- `cmd/gate.go` -- Gate system with recovery templates (lines 1-700)
- `cmd/circuit_breaker.go` -- Circuit breaker with per-worker tracking
- `cmd/hive.go` -- Existing Hive Brain (cross-colony wisdom in JSON)
- `pkg/events/bus.go` -- Event bus with pub/sub and JSONL persistence
- `pkg/codex/process_tracker.go` -- Worker process tracking
- `pkg/storage/storage.go` -- Atomic file operations and file locking
- `pkg/codex/process_group_unix.go` -- Process group management (Setpgid, SIGTERM/SIGKILL)
- `cmd/security_cmds.go` -- Existing check-antipattern scanner (6 patterns)
- `cmd/skills.go` -- Skill system (parse, index, detect, match, inject, diff)
- `.planning/PROJECT.md` -- v1.13 requirements (AAC-001 through AAC-031, REC-LOOP-01)

### Secondary (MEDIUM confidence)
- modernc.org/sqlite: Context7 `/modernc-org/sqlite` docs, pkg.go.dev (v1.42.1)
- SQLite FTS5 documentation: https://www.sqlite.org/fts5.html
- Secret scanning patterns: Gitleaks (https://github.com/gitleaks/gitleaks), TruffleHog
- SQLite WAL mode concurrency: Reddit r/sqlite, Stack Overflow, Tessl Registry best practices

---
*Research completed: 2026-05-01*
*Ready for roadmap: yes*
