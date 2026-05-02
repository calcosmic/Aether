# Requirements: Aether v1.13 Recovery Hardening & Hive Learning

**Defined:** 2026-05-01
**Core Value:** Aether should feel alive and truthful at runtime, not only look clever in wrappers or tests.

## v1 Requirements

Requirements for v1.13. Each maps to roadmap phases. Source: PRD AAC-001 through AAC-031 + REC-LOOP-01 (60 v1 requirements).

### Build Safety

- [x] **SAFE-01**: Build-complete rejects completion when every worker result is failed, blocked, errored, or missing (AAC-001)
- [x] **SAFE-02**: Build-complete rejects completion when every worker result reports files_modified: 0 for a build phase (AAC-001)
- [x] **SAFE-03**: Continue verification traces every accepted build claim to a successful worker result with files_modified > 0 (AAC-002)
- [x] **SAFE-04**: Continue verification records provenance (worker name, caste, run ID, phase, timestamp, status, files_modified) and rejects claims with missing or stale provenance (AAC-002)
- [ ] **SAFE-05**: Worker prompts include all v5.4 context sections: colony-prime, prompt_section, survey context, phase research, matched skills, midden/graveyard cautions (AAC-005)
- [ ] **SAFE-06**: Context is refreshed immediately before worker spawn, not cached from session start (AAC-005)

### Confidence & Planning

- [x] **CONF-01**: Oracle loop accepts user-settable confidence target via --confidence-target flag (default 95) (AAC-003)
- [x] **CONF-02**: Oracle does not finalize below target unless a hard blocker is reported or max-iteration cap is reached (AAC-003)
- [x] **CONF-03**: Oracle output includes target, final score, iteration count, rubric breakdown, evidence, gaps, original prompt, synthesized prompt, and approval status (AAC-003)
- [x] **CONF-04**: Init command accepts raw user prompt, scouts the repo, and synthesizes an approval-ready launch brief (AAC-004)
- [x] **CONF-05**: Colony launch is blocked until user approves, edits, or rejects the synthesized brief (AAC-004)

### Gate Recovery

- [x] **GATE-01**: Every blocking gate failure shows what went wrong, specific fixes, and two options: fix manually then /ant-continue, or run /ant-unblock (AAC-006)
- [x] **GATE-02**: Forbidden strings (CRITICAL: Do NOT proceed, The phase will NOT advance) removed from gate failure paths (AAC-006)
- [x] **GATE-03**: Watcher Veto leaves all working-tree changes intact (no git stash push, no ROLLBACK_VETO) (AAC-007)
- [x] **GATE-04**: Gate results persist in gate-results.json per phase with statuses: passed, failed, skipped, not-reached (AAC-008)
- [x] **GATE-05**: Previously passed/skipped gates are skipped on retry, except Flags Gate and Watcher Veto which always re-run (AAC-008)
- [x] **GATE-06**: /ant-unblock reads gate-results.json, shows Gate Recovery Summary, and offers to dispatch Fixer (AAC-009)
- [x] **GATE-07**: After Fixer resolves issues, addressed blockers are auto-resolved and /ant-continue retruns (AAC-009)
- [x] **GATE-08**: Fixer caste (27th agent) reads gate context, investigates, fixes, verifies, and reports JSON output (AAC-010)
- [x] **GATE-09**: /ant-status shows Gate Status section when gate-results.json exists for current phase (AAC-011)

### Loop Safety Inheritance

- [x] **LOOP-01**: Smart gate retry (GATE-04/005) respects circuit breaker — hard-stop after N failed retries per phase (REC-LOOP-01)
- [x] **LOOP-02**: /ant-unblock (GATE-06) tracks unblock attempts per phase and refuses after configurable cap with human-intervention message (REC-LOOP-01)
- [x] **LOOP-03**: Fixer dispatch blocked when circuit breaker has tripped for current phase (REC-LOOP-01)
- [x] **LOOP-04**: All new gate/recovery paths wire through existing cycle detection and telemetry from v1.12 (REC-LOOP-01)

### Platform

- [x] **PLAT-01**: OpenCode agent hub template generates valid name field; aether update preserves it (AAC-012)
- [x] **PLAT-02**: LLM provider baseURL is separated from worker callback/messaging URL; missing callback fails before worker spawn with clear config error (AAC-013)
- [ ] **PLAT-03**: Workers emit periodic heartbeats (first immediately, then throttled to ~30s intervals) (AAC-014)
- [ ] **PLAT-04**: Workers spawn in managed process groups (Setpgid on Unix, stub on Windows) (AAC-015)
- [ ] **PLAT-05**: Worker PIDs are tracked and killed on exit (SIGTERM then SIGKILL after ~2s) (AAC-016)
- [ ] **PLAT-06**: Stale workers from previous sessions are detected and cleaned before new dispatch (AAC-017)

### Hive Learning Foundation

- [x] **HIVE-01**: Aether-native learning concepts mapped from Hermes, with MIT notice if code is referenced (AAC-019)
- [x] **HIVE-02**: Colony memory store supports add, replace, remove, compact with configurable character/token budgets (AAC-020)
- [x] **HIVE-03**: Colony memory injected into init/oracle/worker prompts as frozen snapshot; failed/empty builds never create durable memory (AAC-020)
- [x] **HIVE-04**: SQLite colony.db (WAL mode) with tables for runs, workers, gates, memories, skills, decisions, trajectories, and schema_version (AAC-021)
- [x] **HIVE-05**: FTS5 search indexes for worker summaries, gate failures, decisions, and memory text via aether hive search (AAC-021)
- [x] **HIVE-06**: Schema migrations are versioned, idempotent, and migration-safe (AAC-021)

### Pheromone Skills

- [x] **SKIL-01**: Repo-local pheromone skills stored in .aether/hive/skills/active/ with SKILL.md format including evidence frontmatter (AAC-022)
- [x] **SKIL-02**: Skills use progressive disclosure — prompt includes index only, full content loads only when matched (AAC-022)
- [x] **SKIL-03**: Skill actions: create, patch, edit, delete/archive, view, list, search, pin, promote (AAC-022)
- [x] **SKIL-04**: Keeper Curator tracks usage (view/use/patch counts), auto-transitions unused skills active -> stale -> archived (AAC-023)
- [x] **SKIL-05**: Pinned skills are immutable to both auto-transitions and agent writes (AAC-023)
- [x] **SKIL-06**: Archived skills are recoverable, never auto-deleted (AAC-023)

### Learning Pipeline

- [x] **LRN-01**: Post-run learning triggered only after verified successful outcomes; failed/empty/phantom runs logged as transient only (AAC-024)
- [x] **LRN-02**: Every durable learning entry includes evidence: source run ID, worker, files touched, tests/gates passed, confidence, scope (AAC-024)
- [x] **LRN-03**: Promotion from repo to hive requires privacy scan and explicit user approval (AAC-024)
- [x] **LRN-04**: Learned context injected into init/oracle/worker prompts ranked by phase, caste, file path, recency, confidence (AAC-025)
- [x] **LRN-05**: Repo isolation — two repos do not see each other's repo-local memory; hive entries are generic and redacted (AAC-026)
- [x] **LRN-06**: Export/import of repo learning packs with manifest, redaction report, and preview-before-apply (AAC-026)

### Privacy & Auto-Skills

- [x] **PRIV-01**: Privacy/secret scan runs before any memory, skill, trajectory, or promotion write (AAC-029)
- [x] **PRIV-02**: Common credential files, API keys, tokens, SSH keys, env files, and local user paths are blocked or redacted (AAC-029)
- [x] **PRIV-03**: Learning entries classified as repo-local, hive-shareable, blocked, or needs-user-approval (AAC-029)
- [x] **PRIV-04**: Trajectory records stored locally with strict redaction; export requires approval and redaction report (AAC-028)
- [x] **PRIV-05**: Learning writes can be disabled by config and by per-command flag (AAC-029)
- [x] **AUTO-01**: Auto-created repo-local skills after difficult verified tasks (configurable: off/propose/auto, default propose) (AAC-031)
- [x] **AUTO-02**: Hard rejection rules prevent skill creation from failed, zero-modification, phantom, unresolved vetoed, or secret-containing runs (AAC-031)
- [x] **AUTO-03**: Auto-created skills include source evidence, verification steps, confidence score, privacy scan result, and repo fingerprint (AAC-031)
- [x] **AUTO-04**: aether update never overwrites repo-local learned skills (AAC-031)

### System Validation

- [ ] **VAL-01**: Full smoke test from init/oracle through phase advancement with gate failure, unblock, fixer, continue, and process cleanup (AAC-018)
- [ ] **VAL-02**: All generated/mirrored files (agents, commands) survive aether update (AAC-018)
- [ ] **VAL-03**: Every new command/file format has validation and actionable errors (AAC-018)

## v2 Requirements

Deferred to future milestone.

### Plugin/Hook Extensions

- **HOOK-01**: Minimal hook registry for learning-relevant events (pre_worker_prompt, post_worker_result, post_gate_result, etc.) (AAC-027)
- **HOOK-02**: Project-local plugins disabled by default, require explicit trust flag (AAC-027)

## Out of Scope

| Feature | Reason |
|---------|--------|
| Cross-colony ledger sharing | Findings contain code-specific paths that go stale across repos |
| Auto-block on critical findings | Conflicts with existing continue-review blocking |
| Real-time ledger sync across agents | YAGNI -- agents write during build/continue, not concurrently |
| Ledger web UI | CLI-only for now |
| CGO SQLite dependency | Pure Go modernc.org/sqlite chosen for cross-platform builds |
| Hermes Agent runtime dependency | Concepts ported to Aether-native Go, not forked as sidecar |
| Full state machine transitions | Deferred from v1.11 (INTEL-06) |
| Council system | Deferred from v1.11 (INTEL-07) |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| SAFE-01 | Phase 88 | Complete |
| SAFE-02 | Phase 88 | Complete |
| SAFE-03 | Phase 88 | Complete |
| SAFE-04 | Phase 88 | Complete |
| SAFE-05 | Phase 92 | Pending |
| SAFE-06 | Phase 92 | Pending |
| CONF-01 | Phase 89 | Complete |
| CONF-02 | Phase 89 | Complete |
| CONF-03 | Phase 89 | Complete |
| CONF-04 | Phase 89 | Complete |
| CONF-05 | Phase 89 | Complete |
| GATE-01 | Phase 88 | Complete |
| GATE-02 | Phase 88 | Complete |
| GATE-03 | Phase 88 | Complete |
| GATE-04 | Phase 88 | Complete |
| GATE-05 | Phase 88 | Complete |
| GATE-06 | Phase 89 | Complete |
| GATE-07 | Phase 89 | Complete |
| GATE-08 | Phase 89 | Complete |
| GATE-09 | Phase 89 | Complete |
| LOOP-01 | Phase 88 | Complete |
| LOOP-02 | Phase 89 | Complete |
| LOOP-03 | Phase 89 | Complete |
| LOOP-04 | Phase 89 | Complete |
| PLAT-01 | Phase 89 | Complete |
| PLAT-02 | Phase 89 | Complete |
| PLAT-03 | Phase 92 | Pending |
| PLAT-04 | Phase 92 | Pending |
| PLAT-05 | Phase 92 | Pending |
| PLAT-06 | Phase 92 | Pending |
| HIVE-01 | Phase 90 | Complete |
| HIVE-02 | Phase 90 | Complete |
| HIVE-03 | Phase 90 | Complete |
| HIVE-04 | Phase 91 | Complete |
| HIVE-05 | Phase 91 | Complete |
| HIVE-06 | Phase 91 | Complete |
| SKIL-01 | Phase 91 | Complete |
| SKIL-02 | Phase 91 | Complete |
| SKIL-03 | Phase 91 | Complete |
| SKIL-04 | Phase 91 | Complete |
| SKIL-05 | Phase 91 | Complete |
| SKIL-06 | Phase 91 | Complete |
| LRN-01 | Phase 90 | Complete |
| LRN-02 | Phase 90 | Complete |
| LRN-03 | Phase 90 | Complete |
| LRN-04 | Phase 90 | Complete |
| LRN-05 | Phase 90 | Complete |
| LRN-06 | Phase 90 | Complete |
| PRIV-01 | Phase 88 | Complete |
| PRIV-02 | Phase 88 | Complete |
| PRIV-03 | Phase 90 | Complete |
| PRIV-04 | Phase 90 | Complete |
| PRIV-05 | Phase 90 | Complete |
| AUTO-01 | Phase 91 | Complete |
| AUTO-02 | Phase 91 | Complete |
| AUTO-03 | Phase 91 | Complete |
| AUTO-04 | Phase 91 | Complete |
| VAL-01 | Phase 92 | Pending |
| VAL-02 | Phase 92 | Pending |
| VAL-03 | Phase 92 | Pending |

**Coverage:**
- v1 requirements: 60 total
- Mapped to phases: 60
- Unmapped: 0

---
*Requirements defined: 2026-05-01*
*Last updated: 2026-05-01 after roadmap creation*
