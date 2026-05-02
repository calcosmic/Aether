# Phase 92: System Hardening & Validation - Context

**Gathered:** 2026-05-02
**Status:** Ready for planning

<domain>
## Phase Boundary

Worker lifecycle is managed with heartbeats and process groups, worker prompts receive fresh complete context at spawn time, and the full v1.13 system is validated end-to-end. This is the capstone phase that ensures all v1.13 components (recovery hardening, gate self-healing, learning foundation, hive intelligence) work together correctly and that worker processes are properly tracked and cleaned up.

Key realization: much of the infrastructure already exists. Process group management (`verification_process_group_unix.go`), stale worker cleanup (`codex_worker_cleanup.go`), and colony-prime context assembly (`colony_prime_context.go` with 12+ sections) are all in place. The gaps are: (1) heartbeat liveness detection, (2) auditing context completeness against AAC-005 and ensuring fresh assembly per-spawn, (3) comprehensive E2E validation, and (4) update integrity tests.

</domain>

<decisions>
## Implementation Decisions

### Worker Heartbeats (PLAT-03)
- **D-01:** File-based heartbeat mechanism. Workers write a timestamp to `.aether/data/heartbeat-{worker-id}.json` at intervals (first immediately, then ~30s throttled). Works across all platforms since workers can write files.
- **D-02:** Heartbeat writes are driven by prompt instruction — the worker prompt includes an instruction to periodically write the heartbeat file. Honest detection (if worker is genuinely stuck, heartbeat stops). Consistent across Claude, OpenCode, and Codex.
- **D-03:** A background goroutine in the Go runtime periodically scans heartbeat files and emits warnings or auto-cleans stuck workers. More responsive than on-demand checks at specific checkpoints.

### Context Refresh (SAFE-05, SAFE-06)
- **D-04:** Audit `buildColonyPrimeOutput()` against AAC-005 requirements (colony-prime, prompt_section, survey context, phase research, matched skills, midden/graveyard cautions) to identify any missing sections. If sections are missing, add them. If they already exist, the gap is timing only.
- **D-05:** Colony-prime context must be assembled fresh immediately before each worker spawn — not cached from session start. The session cache (24h TTL) is fine for reading data sources, but the final assembly happens at dispatch time.

### E2E Smoke Test (VAL-01)
- **D-06:** Write a single Go integration test that exercises the full v1.13 flow: init → build → gate failure → unblock → fixer → continue → learning capture → hive search → skill lifecycle → seal cleanup. Tests everything working together, following the existing `e2e_recovery_test.go` pattern.
- **D-07:** The E2E test covers the full v1.13 system, not just Phase 92 scope. This is the validation that the entire milestone works end-to-end.

### Update Integrity (VAL-02)
- **D-08:** Write an update round-trip test: (1) create a known set of agent and command files, (2) run the update flow, (3) verify all files still exist with correct content. Catches corruption or deletion during update.
- **D-09:** Round-trip test covers both agent definitions (`.claude/agents/ant/*.md`, `.opencode/agents/*.md`, `.codex/agents/*.toml`) AND command files (`.claude/commands/ant/*.md`, `.opencode/commands/ant/*.md`).

### Validation & Error Messages (VAL-03)
- **D-10:** Every new command and file format from v1.13 gets validation with actionable error messages. This includes: gate-results.json format, learning entry format, skill SKILL.md format, colony.db schema, heartbeat file format. Follow existing validation patterns in `cmd/`.

### Claude's Discretion
- Heartbeat file format and exact goroutine monitoring interval
- Specific missing sections in the AAC-005 audit (determined by reading code)
- E2E test structure and which specific v1.13 features to exercise in sequence
- Heartbeat cleanup behavior on session exit (part of existing worker cleanup flow)
- How heartbeat staleness threshold maps to worker timeout behavior

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Worker Lifecycle Infrastructure (existing code to extend)
- `cmd/codex_worker_cleanup.go` — Existing stale worker cleanup before dispatch. Heartbeat monitor extends this.
- `cmd/worker_cleanup_signal_common.go` — Cleanup handler installation and Go test detection.
- `cmd/worker_cleanup_signal_unix.go` — Unix signal handling for worker cleanup.
- `cmd/worker_cleanup_signal_windows.go` — Windows signal handling for worker cleanup.
- `cmd/verification_process_group_unix.go` — `configureVerificationCommandProcessGroup()` with Setpgid, `terminateVerificationCommandProcessGroup()` with SIGTERM then SIGKILL. Worker dispatch should reuse this pattern.
- `cmd/verification_process_group_windows.go` — Windows stub for process group management.
- `pkg/codex/` — Codex worker management package (CleanupStaleWorkers, process tracking).

### Context Assembly (SAFE-05/06)
- `cmd/colony_prime_context.go` — `buildColonyPrimeOutput()` assembles 12+ context sections. Audit this against AAC-005 requirements.
- `cmd/context.go` — Context capsule assembly.
- `pkg/colony/context_ranking.go` — `RankContextCandidates()` with budget-aware trimming.
- `pkg/cache/session_cache.go` — Session cache with 24h TTL. Fine for data reads but final assembly must be fresh.

### E2E Test Pattern
- `cmd/e2e_recovery_test.go` — Existing E2E recovery test. Pattern reference for the full v1.13 smoke test.
- `cmd/integration_test.go` — Integration test including colony-prime test.
- `cmd/codex_continue_test.go` — Continue flow tests with gate scenarios.

### Update Integrity
- `cmd/install_cmd.go` — Install/update command implementation.
- `cmd/setup_cmd_test.go` — Setup and install tests.
- `.aether/agents-claude/` — Claude agent mirror (packaging). Must survive update.
- `.aether/agents-codex/` — Codex agent mirror (packaging). Must survive update.

### Prior Phase Context (dependency chain)
- `.planning/phases/91-hive-intelligence/91-CONTEXT.md` — SQLite ColonyStore, skill lifecycle, auto-skills, Keeper curator.
- `.planning/phases/90-learning-foundation/90-CONTEXT.md` — Learning triggers, unified memory API, repo isolation.
- `.planning/phases/89-gate-self-healing-smart-planning/89-CONTEXT.md` — Fixer caste, unblock dispatch, Oracle confidence, gate-results persistence.
- `.planning/phases/88-recovery-foundation/88-CONTEXT.md` — Provenance validation, privacy gate, gate state persistence.

### Requirements
- `.planning/REQUIREMENTS.md` — SAFE-05/06, PLAT-03/04/05/06, VAL-01/02/03
- `.planning/ROADMAP.md` § Phase 92 — Success criteria and goal definition

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `cmd/verification_process_group_unix.go` — `configureVerificationCommandProcessGroup()` with Setpgid, `terminateVerificationCommandProcessGroup()` with SIGTERM then SIGKILL. Worker dispatch should reuse this pattern.
- `cmd/codex_worker_cleanup.go` — `cleanupStaleWorkersBeforeDispatch()` already cleans stale workers. Heartbeat monitor adds liveness detection on top.
- `pkg/codex/` — `CleanupStaleWorkers()` function returns stale/terminated/killed counts. Heartbeat monitor feeds into this.
- `cmd/colony_prime_context.go` — `buildColonyPrimeOutput()` assembles the full context with ranking and budget management. The audit checks completeness; the per-spawn call ensures freshness.
- `cmd/e2e_recovery_test.go` — E2E test pattern: create temp dir, init state, run commands in sequence, verify outcomes. The v1.13 smoke test follows this.

### Established Patterns
- Process group management: build tags for Unix (`//go:build !windows`) and Windows stubs
- Worker cleanup: signal handlers registered via `workerCleanupHandlerInstalled` atomic.Bool
- Colony-prime sections: each section is a `colonyPrimeSection` with name, title, source, content, priority, trust scores
- E2E tests: temp directory setup, state initialization, sequential command execution, outcome verification
- File locking: `pkg/storage/` patterns for concurrent-safe file access

### Integration Points
- Heartbeat monitor goroutine: starts at build dispatch, stops at build completion. Reads heartbeat files in `.aether/data/`.
- Colony-prime freshness: `resolveCodexWorkerContext()` already checks for minimum 128 chars. Per-spawn means calling `buildColonyPrimeOutput()` at each worker dispatch point.
- E2E test: exercises all v1.13 commands in sequence (init, build, gate-fail, unblock, fixer, continue, learn, hive-search, skill, seal)
- Update round-trip: install → verify files → update → verify files unchanged. Tests the full publish/update pipeline.

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches following established patterns.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 92-System Hardening & Validation*
*Context gathered: 2026-05-02*
