# Project Research Summary

**Project:** Aether Colony System v1.1 Bug Fixes
**Domain:** CLI-based AI agent orchestration framework
**Researched:** 2026-02-14
**Confidence:** HIGH

## Executive Summary

The Aether Colony System v1.1 milestone addresses five critical bugs discovered during real-world usage of the v1.0 infrastructure. These bugs represent fundamental reliability issues: phase advancement loops that waste compute resources, overly broad git checkpoints that risk user data loss (documented case: 1,145 lines nearly lost), missing deterministic builds due to absent package-lock.json, and misleading output timing from background task execution. The existing architecture — a four-layer system (Queen, Constraints, Worker, Utility) with state managed via COLONY_STATE.json — is sound but needs hardening at the edges.

Expert practice for CLI-based orchestration systems emphasizes three principles evident in this research: (1) state transitions must be idempotent with verification gates, (2) user data boundaries must be enforced by allowlist (never blocklist), and (3) deterministic builds require lockfile commitment. The recommended approach for v1.1 is to treat these as infrastructure hardening rather than feature additions — fix the foundation before building higher. This means prioritizing the data-loss-prevention fix (targeted git checkpoints) and deterministic builds (package-lock.json) as P0, while deferring nice-to-have improvements like version-aware notifications to v1.2.

Key risks center on concurrency and state management. The system spawns parallel workers but lacks comprehensive file locking for state updates. The checkpoint system uses `git stash` which captures all dirty files by default — a dangerous operation that has already caused near-data-loss. Mitigation requires: (1) replacing stash with explicit file copies based on a strict allowlist, (2) adding idempotency keys to phase transitions, and (3) implementing proper test coverage for sync functions before modifying them.

## Key Findings

### Recommended Stack

The existing stack (Node.js with commander.js, AVA for testing, bash utilities) requires minimal additions for v1.1. The focus is on testing infrastructure and deterministic builds rather than new runtime dependencies.

**Core technologies (keep):**
- **Node.js >=16.0.0**: Runtime — meets requirements, no changes needed
- **commander ^12.1.0**: CLI argument parsing — already in use, proven stable
- **AVA ^6.0.0**: Unit testing — already configured, keep for consistency
- **picocolors ^1.1.1**: Colored output — lightweight, already in use

**Required additions:**
- **package-lock.json**: Deterministic dependency installation — generate via `npm install`, commit to repo, use `npm ci` in CI
- **sinon ^17.0.0**: Test spies/stubs/mocks — industry standard for JS mocking, required for unit testing sync functions
- **proxyquire ^2.1.3**: Dependency injection for testing — enables mocking `fs` module in cli.js tests
- **tmp ^0.2.1**: Temporary file/directory creation — handles OS-specific temp locations, auto-cleanup

**What to avoid:**
- Jest (heavier, slower than AVA)
- mock-fs (less flexible than sinon + proxyquire)
- Git stash with `--include-untracked` (causes data loss)

### Expected Features

**Must have (P0 — table stakes fixes):**
- **Targeted git checkpoints** — Only stash system files (.aether/*.md, .claude/commands/ant/, .opencode/commands/ant/, runtime/, bin/cli.js), never user data (TO-DOs.md, .aether/data/, .aether/dreams/, .aether/oracle/)
- **Deterministic dependency builds** — Commit package-lock.json for reproducible installs
- **Unit tests for core sync functions** — Test `syncDirWithCleanup`, `hashFileSync`, `generateManifest` in isolation
- **Synchronous worker spawns** — Remove `run_in_background: true` from build.md Steps 5.1, 5.4, 5.4.2 to fix output timing

**Should have (P1 — reliability improvements):**
- **Phase advancement guards** — Prevent AI loops by verifying phase completion before advancement
- **Cross-repo sync reliability** — Better error handling for dirty repos, network failures, partial updates

**Defer (P2 — v1.2+):**
- **Version-aware update notifications** — Non-blocking version check at command start
- **Checkpoint recovery tracking** — Stash operation logging with auto-suggest recovery

### Architecture Approach

The v1.1 fixes integrate with the existing four-layer architecture (Queen, Constraints, Worker, Utility) while maintaining compatibility with COLONY_STATE.json patterns. Key architectural decisions:

**Phase advancement flow** — Add idempotency checks before state mutation. Use file-lock.sh for lock acquisition, atomic-write.sh for state updates, and validate-state command for verification. Generate idempotency keys from timestamp + random bytes, check events array before advancing.

**Safe checkpoint system** — Replace `git stash` with explicit file copies to `.aether/checkpoints/<id>/`. Use allowlist: only files in `.aether/` (excluding data/), `.claude/commands/ant/`, `.opencode/commands/ant/`, `.opencode/agents/`, `bin/`. Store metadata (commit hash, timestamp, file hashes) in checkpoint.json.

**Update system with rollback** — Implement two-phase commit: backup current state, sync files with hash verification, update version.json atomically only after successful sync. On failure, restore from backup manifest.

**Test architecture** — Three layers: unit tests (mocked fs with sinon/proxyquire), integration tests (temp directories with tmp package), E2E tests (existing bash-based tests). Prioritize hash verification, cleanup safety, and idempotency property tests.

### Critical Pitfalls

1. **Git stash captures user data (CRITICAL)** — The checkpoint system currently stashes ALL dirty files. Prevention: explicit allowlist approach, verify `git status --porcelain` before stash, never use `--include-untracked`, test stash scope with `git stash show -p`.

2. **Phase advancement loops (CRITICAL)** — State machine lacks guard conditions, causing infinite loops. Prevention: enforce Iron Law (no advancement without verification evidence), check `state != "COMPLETED"` before allowing transition, acquire state lock during transition, log audit trail with evidence.

3. **Update command stashes without recovery path (HIGH)** — Stash created but not popped if update fails. Prevention: always pop on success, warn prominently on failure with recovery command, include timestamp in stash message, record stash ref in state file.

4. **run_in_background causes misleading output timing (HIGH)** — Build summary appears before agent notifications. Prevention: remove `run_in_background` flag (multiple Task calls already run in parallel without it), use foreground Task calls with TaskOutput blocking.

5. **Missing unit tests for core sync functions (HIGH)** — Changes risk breaking update mechanism without detection. Prevention: mock filesystem for isolated tests, property-based idempotency testing, edge case coverage (empty files, permissions, symlinks).

## Implications for Roadmap

Based on research, suggested phase structure for v1.1:

### Phase 1: Foundation (Week 1)
**Rationale:** Safe checkpoint system provides rollback capability for subsequent fixes; testing infrastructure enables verification of other fixes. No dependencies on other work.
**Delivers:** Targeted checkpoint system (no data loss), unit test framework with mocking, hash verification tests
**Addresses:** Targeted git checkpoints (P0), Unit tests for core sync (P0)
**Avoids:** Git stash captures user data, Missing unit tests pitfalls
**Research flag:** SKIP — implementation patterns are standard, no deep research needed

### Phase 2: Core Fixes (Week 2)
**Rationale:** Update system depends on safe checkpoint system from Phase 1. Phase advancement depends on testing infrastructure from Phase 1 for idempotency tests.
**Delivers:** Update system with rollback capability, phase advancement guards with idempotency keys
**Addresses:** Cross-repo sync reliability (P1), Phase advancement guards (P1)
**Avoids:** Phase advancement loops, Update stash not recovered pitfalls
**Research flag:** SKIP — patterns documented in ARCHITECTURE.md, standard state machine practices

### Phase 3: Integration & Polish (Week 3)
**Rationale:** Remove background task flags (isolated fix), complete integration testing, verify all fixes work together.
**Delivers:** Fixed output timing, complete integration test suite, E2E verification
**Addresses:** Synchronous worker spawns (P0)
**Avoids:** Misleading output timing, Race conditions in shared state pitfalls
**Research flag:** SKIP — simple flag removal, well-understood timing issue

### Phase Ordering Rationale

- **Checkpoint system first** — Provides rollback safety net for subsequent changes; fixes critical data loss risk immediately
- **Testing infrastructure with Phase 1** — Cannot safely modify sync functions without tests; prevents regression
- **Update system after checkpoint** — Depends on safe backup/restore mechanism
- **Phase advancement after testing** — Requires idempotency tests to verify fix
- **Output timing last** — Isolated change with no dependencies, can be done anytime

### Research Flags

Phases likely needing deeper research during planning: **NONE** — v1.1 is bug fixes only, patterns are well-documented in existing research.

Phases with standard patterns (skip research-phase):
- **Phase 1 (Foundation):** Checkpoint allowlist pattern is straightforward; sinon/proxyquire are industry standard
- **Phase 2 (Core Fixes):** State machine idempotency is well-understood pattern; two-phase commit is standard distributed systems practice
- **Phase 3 (Integration):** Flag removal is trivial; integration testing patterns established in existing E2E tests

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Based on existing codebase patterns and Node.js ecosystem standards; additions are industry-standard testing libraries |
| Features | HIGH | Based on documented bugs in TO-DOs.md, CONCERNS.md; real data loss incident provides clear requirements |
| Architecture | HIGH | Based on existing Aether codebase analysis; four-layer architecture is proven in v1.0 |
| Pitfalls | HIGH | Based on actual bugs encountered; near-data-loss incident validates severity |

**Overall confidence:** HIGH

### Gaps to Address

- **Allowlist completeness:** Must audit all `.aether/` subdirectories to confirm which are system vs user data before implementing checkpoint fix. Verification: review `.aether/data/`, `.aether/dreams/`, `.aether/oracle/` contents.
- **Test coverage baseline:** Current test structure needs review to determine exact mocking strategy for cli.js functions. Verification: inspect existing test files and cli.js exports.
- **CI integration:** Need to verify CI uses `npm ci` after package-lock.json is added. Verification: check `.github/workflows/` or equivalent.

## Sources

### Primary (HIGH confidence)
- Aether TO-DOs.md — Documented bugs with full context (data loss from stash, output ordering)
- Aether CONCERNS.md — Technical debt and security audit
- Aether codebase: `bin/cli.js` — syncDirWithCleanup, updateRepo implementation
- Aether codebase: `.aether/aether-utils.sh` — State management commands
- Aether codebase: `.claude/commands/ant/build.md` — Worker spawn patterns
- npm documentation: package-lock.json for deterministic installs
- Sinon.js documentation: Standard mocking library for JavaScript

### Secondary (MEDIUM confidence)
- Aether ARCHITECTURE.md — State management patterns (from prior research)
- Aether progress.md — Race condition fixes, idempotency issues
- Existing E2E tests: `tests/e2e/test-update.sh`, `tests/e2e/test-update-all.sh` — Test patterns

### Tertiary (LOW confidence)
- None — v1.1 research based entirely on existing codebase and documented bugs

---

*Research completed: 2026-02-14*
*Ready for roadmap: yes*
