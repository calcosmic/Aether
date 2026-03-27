# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-03-27)

**Core value:** Reliably interpret user requests, decompose into work, verify outputs, and ship correct work with minimal back-and-forth.
**Current focus:** v2.5 Smart Init -- Phase 32: Intelligence Enhancements

## Current Position

Phase: 32 of 32 (Intelligence Enhancements) -- IN PROGRESS
Plan: 2 of 3 complete
Status: Plan 32-02 complete -- intelligence-enriched approval prompt in init.md
Last activity: 2026-03-27 -- 32-02 enriched init prompt (2 tasks, 2 commits, 616 tests passing)

Progress: [████████░] 78%

## Performance Metrics

**Velocity (from v2.1):**
- Total plans completed: 66
- Average duration: 5min
- Total execution time: 5.6 hours

**Recent Trend:**
- v2.4: 4 phases completed (25-28), 8 plans total, all shipped 2026-03-27
- v2.5: Phase 30 complete (2 plans), Phase 31 complete (2 plans), Phase 32 in progress (1/3 plans)
- v2.3: 4 phases completed (21-24), 10 plans total
- v2.2: 4 phases completed (17-20), 5 plans total

*Updated after each plan completion*

## Accumulated Context

### Decisions

Recent decisions affecting v2.5 work:

- [v2.5]: 4-phase structure follows dependency chain: scan module (foundation) -> charter functions (QUEEN.md writes) -> init.md rewrite (integration) -> intelligence enhancements (enrichment)
- [v2.5]: QUEEN.md charter content written into EXISTING v2 sections only -- User Preferences for intent/vision, Codebase Patterns for governance/goals. No new `## ` headers (7+ downstream consumers parse by exact header)
- [v2.5]: scan.sh is a new bash utils module (10th domain module) -- init-research subcommand provides structured JSON research data
- [v2.5]: Prompt generation is deterministic bash+jq assembly within init.md, NOT LLM-generated
- [v2.5]: Approval loop is LLM-mediated (Claude Code is the UI) -- display Markdown, wait for user response, continue
- [v2.5]: Sub-scan functions return raw JSON via stdout, entry point _scan_init_research wraps final assembly in json_ok
- [v2.5]: Complexity thresholds: large (500+ files OR 8+ depth OR 50+ deps), medium (100+ OR 5+ OR 15+), small otherwise
- [v2.5]: Survey staleness uses 7-day window with COLONY_STATE.json timestamp as primary, file mtime as fallback
- [v2.5]: Stale survey test requires all 7 survey docs (completeness check precedes staleness check in scan.sh)
- [v2.5]: assert_json_has_field from test-helpers.sh only supports top-level keys -- use jq -e for nested paths
- [v2.5]: macOS sed lacks \u Unicode escape -- use awk for title case conversion
- [v2.5]: Charter entries counted in METADATA stats to prevent drift on repeated re-inits
- [v2.5]: xml-utils.sh requires exchange/ directory at source time -- test setup must copy both utils/ and exchange/
- [v2.5]: macOS head -n -1 not portable -- use sed '$d' for stripping last line from sed range extraction
- [v2.5]: json_err writes error JSON to stderr -- error path tests must capture stderr (2>&1)
- [v2.5]: Re-init mode skips all template writes (COLONY_STATE, constraints, runtime files) -- only charter-write + session-init to preserve state
- [v2.5]: Max 2 revision rounds for init approval prompt -- forces final approve/cancel decision after edits
- [v2.5]: Scan failure degrades gracefully with fallback values -- init never stops because init-research fails
- [v2.5]: Max 3 prior colonies shown in colony_context (most recent first by directory name sort)
- [v2.5]: Pheromone suggestions use 10 deterministic pattern checks, not LLM inference
- [v2.5]: Governance rules focus on process/standards (TDD, linting, CI), not technology choices
- [v2.5]: Legacy manifest formats (phases_completed as array) handled via jq type check
- [v2.5]: Prior Context section uses conditional display -- omitted entirely when no prior colonies exist (no placeholder)
- [v2.5]: Governance rules pre-populated as semicolon-separated text in Charter Governance field
- [v2.5]: Pheromone auto-apply uses 30d TTL and source "system:init" via pheromone-write calls

### Pending Todos

None yet.

### Blockers/Concerns

- Approval loop UX is unvalidated -- the LLM-mediated approval pattern needs user testing (flagged in research)
- Token budget impact of charter content is uncertain -- research recommends 500-char cap for smart-init content to avoid crowding colony-earned wisdom

## Session Continuity

Last session: 2026-03-27
Stopped at: Completed 32-02 enriched init prompt (2 tasks, 2 commits, 616 tests passing)
Resume file: None
