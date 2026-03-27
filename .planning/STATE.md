# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-03-27)

**Core value:** Reliably interpret user requests, decompose into work, verify outputs, and ship correct work with minimal back-and-forth.
**Current focus:** v2.5 Smart Init -- Phase 30: Charter Management

## Current Position

Phase: 30 of 32 (Charter Management)
Plan: 1 of 2 complete
Status: 30-01 charter functions complete -- ready for 30-02
Last activity: 2026-03-27 — 30-01 charter-write and colony-name subcommands (2 tasks, 2 commits, 616 tests passing)

Progress: [███░░░░░░] 23%

## Performance Metrics

**Velocity (from v2.1):**
- Total plans completed: 62
- Average duration: 5min
- Total execution time: 5.1 hours

**Recent Trend:**
- v2.4: 4 phases completed (25-28), 8 plans total, all shipped 2026-03-27
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

### Pending Todos

None yet.

### Blockers/Concerns

- Approval loop UX is unvalidated -- the LLM-mediated approval pattern needs user testing (flagged in research)
- Token budget impact of charter content is uncertain -- research recommends 500-char cap for smart-init content to avoid crowding colony-earned wisdom

## Session Continuity

Last session: 2026-03-27
Stopped at: Completed 30-01 charter functions (2 tasks, 2 commits, charter-write + colony-name subcommands)
Resume file: None
