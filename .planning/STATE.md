# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-19)

**Core value:** Prevent context rot across Claude Code sessions with self-managing colony that learns and guides users
**Current focus:** v1.3 The Great Restructuring — reliability improvements (templates, agent cleanup, pipeline simplification)

## Current Position

Phase: 20 of 25 — Distribution Simplification
Plan: 03 of 03
Status: In progress
Last activity: 2026-02-19 — Completed 20-02: removed all runtime/ references from shell code, hooks, build commands, docs

## Performance Metrics

**Cumulative:**
- Total plans completed: 58 (v1.0: 27, v1.1: 13, v1.2: 18)
- Total requirements: 84 validated (v1.0: 46, v1.1: 14, v1.2: 24)
- v1.3 target: 24 requirements across 6 phases
- Total tests: 446 passing (415 AVA + 31 bash), 0 failures

## Accumulated Context

### Decisions
- Scoped v1.3 to reliability over architecture after LLM architect review
- Deferred: Queen split, file locks, JSON schemas, full XML rewrite, phase scratch pad
- Template system identified as highest-impact improvement
- Additive migration for templates: create first, wire commands later
- [Phase 20]: npm 11.x bypasses root .npmignore when files field present — use subdirectory .aether/.npmignore instead
- [Phase 20]: Distribution pipeline: direct .aether/ packaging replaces runtime/ staging (v4.0.0)
- [Phase 20-02]: Pre-commit hook is validation-only (advisory, exits 0 always) — no blocking on commit
- [Phase 20-02]: queen-init template lookup chain: hub (system/) -> dev (.aether/) -> legacy hub (no staging path)
- [Phase 20-02]: ISSUE-004 fully resolved — template path hardcoded to staging dir no longer an issue

### Key Findings from Research
- 7 research docs analyzed (agent architecture, template system, team coordination, distribution chain)
- ~40% of proposals solved theoretical problems — cut to focus on real reliability gains
- Surveyor XML performance comes from prescriptiveness, not XML tags per se
- Template "read and fill" pattern is well-established in production LLM systems
- Escalation chain is the biggest coordination gap (no receiving protocol defined)

### Blockers / Concerns
- None

## Session Continuity

Last session: 2026-02-19T20:25:18Z
Stopped at: Completed 20-02-PLAN.md
Next step: Execute 20-03-PLAN.md (documentation update for v4.0)
