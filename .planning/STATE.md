# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-03-23)

**Core value:** Reliably interpret user requests, decompose into work, verify outputs, and ship correct work with minimal back-and-forth.
**Current focus:** Phase 14 — Planning Depth (v2.1 Production Hardening)

## Current Position

Phase: 14 of 16 (Planning Depth)
Plan: 2 of 2
Status: Phase Complete
Last activity: 2026-03-24 — Completed 14-02 (research context injection into builder/watcher prompts)

Progress: [█████████░] 88%

## Performance Metrics

**Velocity:**
- Total plans completed: 33
- Average duration: 5min
- Total execution time: 2.6 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01-data-purge | 1 | 3min | 3min |
| 02-command-audit-data-tooling | 2 | 12min | 6min |
| 03-pheromone-signal-plumbing | 2 | 8min | 4min |
| 04-pheromone-worker-integration | 2 | 7min | 3.5min |
| 05-learning-pipeline-validation | 2 | 7min | 3.5min |
| 06-xml-exchange-activation | 2 | 5min | 2.5min |
| 07-fresh-install-hardening | 2 | 7min | 3.5min |
| 08-documentation-update | 2 | 6min | 3min |
| 09-quick-wins | 2 | 10min | 5min |
| 10-error-triage | 2 | 28min | 14min |
| 11-dead-code-deprecation | 2 | 9min | 4.5min |
| 12-state-api-verification | 3 | 41min | 13.7min |
| 13-monolith-modularization | 9 | 58min | 6.4min |
| 14-planning-depth | 2 | 4min | 2min |

**Recent Trend:**
- Last 5 plans: 10min, 6min, 5min, 2min, 2min
- Trend: fast (playbook-only edits, no code changes)

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- [Roadmap v2.1]: Quick wins first (6 independent fixes) to establish green baseline before structural work
- [Roadmap v2.1]: Error triage before modularization to prevent refactoring death spiral
- [Roadmap v2.1]: State API facade (QUAL-04) before domain extraction (QUAL-05/06/07) — dependency order is non-negotiable
- [Roadmap v2.1]: Documentation last — every prior code change makes earlier doc corrections stale
- [Roadmap v2.1]: Dead code deprecation (warnings) before removal — one-cycle confirmation across all 3 surfaces
- [09-01]: Learning-observations uses .bak.N naming (not create_backup) for recovery compatibility
- [09-01]: state-checkpoint uses create_backup (timestamped naming) matching existing atomic-write patterns
- [09-01]: All backups corrupted = hard stop (not auto-reset) per user decision
- [09-02]: state-write uses E_UNKNOWN (not E_INTERNAL) because E_INTERNAL is undefined
- [09-02]: Trimming markers use [trimmed]/[!trimmed] distinct from recovery warning markers
- [10-01]: [error] prefix for _aether_log_error -- distinct from json_err (JSON), recovery (⚠), budget ([trimmed])
- [10-01]: SUPPRESS:OK categories: cleanup, read-default, existence-test, cross-platform, idempotent, validation
- [10-01]: 60 type/command-v idioms left uncommented (universally understood)
- [10-01]: 35 lazy/dangerous patterns deferred to Plans 02/03
- [10-02]: Actual lazy count was ~25 (not ~110) -- Plan 01 was more thorough than research estimated
- [10-02]: grep -c on variables is SUPPRESS:OK (grep exit-code handling, not lazy suppression)
- [10-02]: acquire_lock on registry deferred to Plan 03 (dangerous write-path suppression)
- [11-01]: Deprecation warning uses printf >&2 to avoid breaking JSON stdout contracts
- [11-01]: Warning format '[deprecated] name -- will be removed in v3.0' for grep-ability
- [11-01]: Only 3 of 18 deprecated commands appeared in named help sections; rest only in flat commands array
- [11-02]: Use 2>/dev/null (not 2>&1) for tests parsing JSON stdout from deprecated subcommands
- [11-02]: Use spawnSync in Node.js tests for separate stderr capture (execSync merges stdio)
- [12-01]: futureISO(30) for dynamic test dates -- prevents recurring expiration failures
- [12-01]: state-read-field returns raw for internal callers; subcommand entry wraps in json_ok
- [12-01]: _state_migrate extracted from validate-state for reuse by state-api.sh
- [12-02]: Missing builder claims file = graceful skip (not error) for first-time runs
- [12-02]: Conservative watcher (says fail when tests pass) is not fabrication; only opposite direction blocks
- [12-02]: verify-claims returns json_ok even for blocked status (ok:true, verification_status:"blocked")
- [12-03]: env.X in jq expressions for _state_mutate parameter injection (env vars set inline before function call)
- [12-03]: Read-only migrations use _state_read_field('.') piped to jq for complex multi-field queries
- [12-03]: grave-add uses jq-side type coercion (tonumber, null detection) instead of bash pre-formatting
- [12-03]: spawn-complete wraps _state_mutate in error handler (non-critical event logging path)
- [13-01]: Verbatim extraction -- no refactoring during domain moves, structural change only
- [13-01]: json_ok response uses .result field (not .data) -- existing contract preserved in smoke tests
- [13-02]: Verbatim extraction from 3 non-contiguous ranges -- same no-refactoring policy as Plan 01
- [13-02]: get_caste_emoji stays in main file -- available at call time since sourcing defines functions, not calls them
- [13-03]: Verbatim extraction from 2 non-contiguous ranges -- same no-refactoring policy as Plans 01 and 02
- [13-03]: _rotate_spawn_tree moved with session-init -- only caller, keeps helper co-located with consumer
- [13-04]: Verbatim extraction of contiguous block -- same no-refactoring policy as Plans 01-03
- [13-04]: get_type_emoji moved into suggest.sh -- only caller is _suggest_approve, keeps helper co-located
- [13-04]: Cross-domain pheromone-write calls preserved as subprocess dispatch (bash $0) -- no conversion to direct function calls
- [13-05]: Verbatim extraction of non-contiguous blocks -- same no-refactoring policy as Plans 01-04
- [13-05]: _extract_wisdom_sections moved into queen.sh -- only caller is _queen_read, keeps helper co-located
- [13-05]: get_wisdom_threshold and get_wisdom_thresholds_json stay in main file -- shared by queen and learning domains
- [13-06]: Verbatim extraction of 2 non-contiguous blocks -- same no-refactoring policy as Plans 01-05
- [13-06]: Local helper functions renamed with _sw_ prefix to avoid namespace collisions (format_duration, render_progress_bar, etc.)
- [13-06]: ANSI color variables prefixed with _SW_ inside display functions to avoid global pollution
- [13-06]: Plan listed autofix-restore/autofix-apply but actual subcommands are autofix-checkpoint/autofix-rollback (17 total)
- [13-07]: Verbatim extraction of 3 non-contiguous blocks -- same no-refactoring policy as Plans 01-06
- [13-07]: get_wisdom_threshold and get_wisdom_thresholds_json stay in main file -- shared by queen and learning domains
- [13-07]: memory-capture stays in main file -- orchestrates learning-observe/learning-promote-auto via subprocess, not a learning domain function
- [13-08]: Verbatim extraction of contiguous block -- same no-refactoring policy as Plans 01-07
- [13-08]: _extract_wisdom stays as nested function inside _colony_prime -- only caller, preserves original structure
- [13-08]: hive-*/midden-write one-liner dispatches between pheromone blocks left in place (already extracted to their own modules)
- [13-09]: colony-archive-xml added to existing chamber-utils.sh (Option A) -- colony lifecycle related, avoids creating 137-line standalone file
- [13-09]: Verbatim extraction with no refactoring -- same policy as Plans 01-08
- [13-09]: Phase 13 complete: 11,663 -> 5,262 lines (55% reduction), 9 modules, 72 subcommands extracted
- [14-02]: Research context gets own 16K character budget (separate from colony-prime 8K and skills 12K)
- [14-02]: Injection order: archaeology -> integration -> research -> grave -> midden -> prompt_section -> skill_section
- [14-02]: Missing RESEARCH.md gracefully degrades (empty research_context, build continues)

### Pending Todos

None yet.

### Blockers/Concerns

- Resolved: Phase 14 (Planning Depth) complete -- research always runs (user decision: no skip flag), loaded with 16K budget in build
- Risk: ~10 dangerous suppressions remain for Plan 03 (create_backup, acquire_lock, direct state writes, jq transforms)
- Resolved: context-continuity test failure fixed in 12-01 (QUAL-09 complete)

## Session Continuity

Last session: 2026-03-24
Stopped at: Completed 14-02-PLAN.md (research context injection) -- Phase 14 complete (2 of 2 plans)
Resume file: None
