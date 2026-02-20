# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-19)

**Core value:** Prevent context rot across Claude Code sessions with self-managing colony that learns and guides users
**Current focus:** v1.3 The Great Restructuring — reliability improvements (templates, agent cleanup, pipeline simplification)

## Current Position

Phase: 24 of 25 — Template Integration
Plan: 02 of 03 (COMPLETE)
Status: Phase 24 Plan 02 Complete
Last activity: 2026-02-20 — Completed 24-02: wired seal.md and entomb.md (both platforms) to templates, refreshed ceremony templates with triumphant/reflective voices

## Performance Metrics

**Cumulative:**
- Total plans completed: 68 (v1.0: 27, v1.1: 13, v1.2: 18, v1.3: 10)
- Total requirements: 96 validated (v1.0: 46, v1.1: 14, v1.2: 24, v1.3: 12)
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
- [Phase 20-03]: Historical runtime/ references preserved with RESOLVED markers rather than deleted — audit value maintained
- [Phase 21-02]: Source heredoc casing preserved exactly in templates — fidelity over normalization
- [Phase 21-02]: Static content (Chamber Contents, Session Note) kept verbatim with no placeholders
- [Phase 21-01]: Used exact plan-specified template content for all three data-structure templates
- [Phase 21-03]: All templates grouped together in REQUIRED_FILES array for readability
- [Phase 22-01]: Queen and builder were already clean before this plan ran — only 7 of 9 agents needed edits; route-setter had no Depth-Based Behavior section
- [Phase 22-02]: 22-01 commit cleaned more agents than message indicated — Quality 4 (guardian, measurer, includer, gatekeeper) already cleaned
- [Phase 22-03]: Special 3 agents pre-completed in 22-01 — same pre-completion pattern across all phase 22 plans
- [Phase 22-03]: Missing OpenCode resume.md (pre-existing 34 vs 33 command mismatch) fixed as Rule 3 deviation
- [Phase 22-03]: Content-level command drift (10+ files) and 2 validate-state test failures are pre-existing known debt
- [Phase 23-03]: Resilience sections placed before first Step (not appended) so LLM reads them before executing any steps
- [Phase 23-03]: Three separate XML tags per command (failure_modes, success_criteria, read_only) per locked user decision
- [Phase 23-03]: Entomb failure_modes includes hard seal-first gate with "STOP -- do not archive" language matching existing Step 2
- [Phase 23-01]: Existing rules (3-Fix Rule, Iron Laws, Verification Discipline) referenced by new sections — additive pattern, no redefinition
- [Phase 23-01]: Builder and Queen include Watcher peer review triggers; Watcher self-verifies (it IS the verifier)
- [Phase 23-01]: 2-attempt retry limit and 3-Fix Rule explicitly distinguished in Builder and Tracker — distinct scopes
- [Phase 23-02]: LOW-risk sections kept concise (5-10 lines) — role is investigation-only, key rule is no-writes
- [Phase 23-02]: Surveyor success_criteria extended in-place with Self-Check + Completion Report headings, not replaced
- [Phase 23-02]: Archaeologist/chaos existing laws reinforced with back-reference in new read_only section
- [Phase 24-template-integration]: OpenCode entomb normalized to reset memory fields via jq template — matches Claude Code, wisdom already promoted to QUEEN.md before reset
- [Phase 24-template-integration]: Ceremony templates have distinct voices: crowned-anthill = triumphant, handoff = reflective
- [Phase 24-template-integration]: OpenCode seal.md had no CROWNED-ANTHILL.md write step — only HANDOFF heredoc wired to template

### Key Findings from Research
- 7 research docs analyzed (agent architecture, template system, team coordination, distribution chain)
- ~40% of proposals solved theoretical problems — cut to focus on real reliability gains
- Surveyor XML performance comes from prescriptiveness, not XML tags per se
- Template "read and fill" pattern is well-established in production LLM systems
- Escalation chain is the biggest coordination gap (no receiving protocol defined)

### Blockers / Concerns
- Pre-existing: Content-level command drift between Claude Code and OpenCode directories (10+ files differ)
- Pre-existing: 2 test failures in validate-state.test.js

## Session Continuity

Last session: 2026-02-20T00:05:08Z
Stopped at: Completed 24-02-PLAN.md
Next step: Execute Phase 24 plan 03
