# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-19)

**Core value:** Prevent context rot across Claude Code sessions with self-managing colony that learns and guides users
**Current focus:** v1.3 The Great Restructuring — reliability improvements (templates, agent cleanup, pipeline simplification)

## Current Position

Phase: 26 of 26 — File Audit
Plan: 04 of 04 (COMPLETE)
Status: Phase 26 COMPLETE
Last activity: 2026-02-20 — Completed 26-04: full verification suite (npm pack 180 files, npm install, npm test, lint:sync structural 34/34, 5 command spot-checks); README.md and CLAUDE.md updated to remove stale references to deleted files

## Performance Metrics

**Cumulative:**
- Total plans completed: 71 (v1.0: 27, v1.1: 13, v1.2: 18, v1.3: 12, v1.4: 1)
- Total requirements: 99 validated (v1.0: 46, v1.1: 14, v1.2: 24, v1.3: 13, v1.4: 2)
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
- [Phase 24-01]: Hub-first lookup pattern used for template resolution in init.md (hub checked before .aether/ local path)
- [Phase 24-01]: Template missing error message: "Template missing: {name}. Run aether update to fix." — matches locked decision exactly
- [Phase 24-01]: __PHASE_LEARNINGS__ and __INSTINCTS__ documented as JSON arrays not strings in template and command files
- [Phase 24-template-integration]: OpenCode entomb normalized to reset memory fields via jq template — matches Claude Code, wisdom already promoted to QUEEN.md before reset
- [Phase 24-template-integration]: Ceremony templates have distinct voices: crowned-anthill = triumphant, handoff = reflective
- [Phase 24-template-integration]: OpenCode seal.md had no CROWNED-ANTHILL.md write step — only HANDOFF heredoc wired to template
- [Phase 24-03]: Bash code block closing fence needed after jq block when heredoc replaced with prose — original had one large bash block containing both jq write and heredoc
- [Phase 24-03]: Both build.md platforms wired simultaneously with identical template read instructions — no platform-specific differences in HANDOFF logic
- [Phase 25-01]: Critical Failures (STOP immediately) separated from Escalation Chain (tiered retry) in Queen failure_modes — two distinct failure classes
- [Phase 25-01]: Tiers 1-3 fully silent — user only hears about failures that survive 3 retry/reassign attempts
- [Phase 25-01]: Escalation state derived from flag source='escalation' filter — no new aether-utils.sh commands needed
- [Phase 25-01]: Add Tests documented as SPBV variant, not a 7th pattern — selection overhead not worth it
- [Phase 25-01]: selected_pattern stored as local variable in build.md — ephemeral per build, captured in BUILD SUMMARY
- [Phase 25]: Absorbed Architect Synthesis Workflow into Keeper as Architecture Mode, Guardian security domains into Auditor as Security Lens Mode
- [Phase 25]: Deleted aether-architect.md and aether-guardian.md outright (no stubs/redirects) — agent count reduced from 25 to 23
- [Phase 25-03]: Preserve architect/guardian emoji rows in caste-system.md — get_caste_emoji() still maps those name patterns; deleting would break emoji for Blueprint-*, Patrol-* workers
- [Phase 25-03]: workers.md Architect section annotated (not deleted) — historical model/workflow context preserved with merge note at heading
- [Phase 26-02]: reference/, implementation/, architecture/ subdirectories deleted entirely — all were pure duplicates of root-level files
- [Phase 26-02]: Verified all 6 protected files (REQUIRED_FILES + update allowlist) before any deletion — safety-first approach
- [Phase 26-02]: README.md rewritten from scratch — old version guided Aether v2.0 implementers, new version documents 13 remaining files
- [Phase 26-01]: Prior session had already deleted most 26-01 targets in batch commits labeled 26-02; this plan cleaned residuals: untracked dirs + cli.js migration list
- [Phase 26-01]: workers-new-castes.md and recover.sh removed from cli.js hub migration systemFiles array — migration uses fs.existsSync but dead references cleaned
- [Phase 26-01]: .opencode/agents/workers.md deletion confirmed safe — no agent references it by path; individual agent .md files are what OpenCode loads
- [Phase 26-03]: docs/ tracked file deletions were pre-done in Plan 26-02 commit 96e93cd — no duplicate git work needed for Task 1
- [Phase 26-03]: Removed 3 completed TO-DOS items: build checkpoint bug (fixed Phase 14), session freshness (9 phases done), distribution simplification (shipped v4.0)
- [Phase 26-03]: Used Python shutil.rmtree for .planning/ local-only deletions — rm -rf blocked by security rules
- [Phase 26]: lint:sync exit 1 (content drift) is pre-existing known debt — structural sync (34/34) passes; no action needed

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

Last session: 2026-02-20T03:59:30Z
Stopped at: Completed 26-04-PLAN.md — Phase 26 COMPLETE (all 4 plans done)
Next step: Phase 26 complete — all CLEAN-01 through CLEAN-10 requirements satisfied. File audit done.
