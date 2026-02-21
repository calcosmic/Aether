# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-20)

**Core value:** Prevent context rot across Claude Code sessions with self-managing colony that learns and guides users
**Current focus:** v3.0 Wisdom & Pheromone Evolution — research complete, requirements defined

## Current Position

Phase: 35
Plan: Phase 35 - Wisdom extraction at seal/entomb
Status: Ready to start - Phase 34 complete
Last activity: 2026-02-20 — Phase 34-03 complete - batch promotion execution, deferred storage, and undo functionality

Progress: [▓▓▓▓▓▓▓▓▓▓] 100% (3/3 plans complete in Phase 34)

## Performance Metrics

**Cumulative:**
- Total plans completed: 96 (v1.0: 27, v1.1: 13, v1.2: 18, v1.3: 12, v1.4: 2, v2.0: 18, v3.0: 6)
- Total requirements validated: 166 (v1.0: 46, v1.1: 14, v1.2: 24, v1.3: 24, v1.4: 1 partial, v2.0: 48, v3.0: 9 requirements)
- Total tests: ~427 passing (427 AVA + 9 bash skipped); all 22 agents fully quality-validated

## Accumulated Context

### Decisions
- 27-01: init.js hub paths fixed — HUB_COMMANDS_CLAUDE/OPENCODE/AGENTS now use HUB_SYSTEM not HUB_DIR (v4.0 structure)
- v2.0 is a major version bump — first time Claude Code gets real agents
- Agents go in `.claude/agents/ant/` subdirectory (user decision — keeps GSD agents separate)
- Hub path: `~/.aether/system/agents-claude/` (separate from OpenCode `agents/` to prevent cross-contamination)
- Agent format: YAML frontmatter + XML body, matching GSD pattern in `.claude/agents/`
- Distribution must be proven in Phase 27 — agents that only work in source repo are not shipped
- PWR-01 through PWR-08 (agent power standards) established in Phase 27 as the conversion checklist template
- v1.4 phases 27-30 absorbed into v2.0 as Phase 31 cleanup
- Description must be quoted in YAML frontmatter (contains colons — unquoted breaks YAML parse)
- Escalation section replaces Spawning Sub-Workers — Claude Code subagents cannot spawn other subagents
- 8 XML sections (role, execution_flow, critical_rules, return_format, success_criteria, failure_modes, escalation, boundaries) define the conversion template for all remaining agents
- 27-03: Watcher has read-only tools (Read, Bash, Grep, Glob — no Write/Edit); explicit tools field enforces this; spawns field removed from return format
- 27-04: Distribution pipeline proven — npm pack includes ant agents only (not GSD agents), hub populated at ~/.aether/system/agents-claude/, second install idempotent
- 28-01: Queen gets Task tool unrestricted — true orchestrator; spawn_tree field removed from return (requires aether-utils.sh); flag-add replaced with structured text note
- 28-01: Caste emoji protocol established — 🔨🐜 Builder, 🔭🐜 Scout, 👁🐜 Watcher, 🗺🐜 Route-Setter/Surveyor
- 28-02: Scout gets WebSearch/WebFetch but no Bash — read-only posture enforced via explicit tools field
- 28-02: Route-Setter Task tool documented with graceful degradation for subagent context where Task may be ineffective
- 28-03: Surveyors get Write tool (not read-only) with scope restricted to .aether/data/survey/ only — locked decision from 28-CONTEXT.md overriding roadmap
- 28-03: OpenCode read_only section maps to boundaries section; consumption section embedded in execution_flow
- 29-02: Probe writes AND runs tests — untested tests are incomplete work; Bash available specifically for this purpose
- 29-02: Weaver revert protocol uses explicit git commands in failure_modes — behavior preservation enforced not merely documented
- 29-02: Changing test expectations to make tests pass is a behavior change, not a refactor — prohibited in Weaver critical_rules
- 29-01: Tracker boundary enforced at schema level — field is named suggested_fix (not fix_applied) to reinforce that Builder applies the fix
- 29-01: Auditor has no Bash — even for running linters; when Bash is needed for an audit dimension, Auditor returns blocked and routes to Builder or Tracker
- 29-01: Cross-reference escalation pattern established — specialists name the agent they route to (Tracker → Builder/Weaver, Auditor → Queen/Probe), not generic "escalate to orchestrator"
- [Phase 29]: 29-03: Forbidden pattern matching uses aether-utils.sh invocation form to avoid false positives from Queen documentation
- [Phase 29]: 29-03: TEST-05 is a Phase 30 tracker — hardcoded to 22, intentionally fails at 14 agents until Phase 30 ships remaining 8 agents
- [Phase 30]: 30-02: Credentials Iron Law named as a titled rule in critical_rules — named constraint pattern for security-critical absolute rules
- [Phase 30]: 30-02: Chronicler has no Bash tool — read-code-not-run-code posture enforced via explicit tools field; Edit restricted to JSDoc/TSDoc comments declared in four locations in agent body
- [Phase 30]: 30-01: Gatekeeper and Includer have no Bash — static analysis only; Builder executes runtime tools (npm audit, axe-core)
- [Phase 30]: 30-01: Includer always includes analysis_method: 'manual static analysis' and documents runtime testing gaps explicitly
- [Phase 30]: 30-01: Archaeologist primary deliverable is regression_risks array — regression prevention framing leads execution_flow
- [Phase 30]: 30-03: READ_ONLY_CONSTRAINTS expanded to 8 agents — Gatekeeper and Includer confirmed in most restrictive tier (no Bash)
- [Phase 31]: 31-02: Archive pattern for historical docs — move to archive/ subdirectory, not delete
- [Phase 31]: 31-02: Castes organized by tier (Core, Orchestration, Specialists, Niche) for README clarity
- [Phase 31]: 31-01: Description text belongs in instruction prose above bash blocks, not inside them
- [Phase 31]: 31-01: CLEAN-03 lint test scans both .claude/commands/ant/ and .opencode/commands/ant/
- [Phase 31]: 31-03: Version 2.0.0 (not 5.0.0) to align with Worker Emergence milestone name per user decision; git tag only, npm publish deferred for dist-tag choice
- [v3.0]: Research conducted for wisdom/pheromone evolution system — 4 research files written (FEATURES, ARCHITECTURE, STACK, PITFALLS) + SUMMARY
- [v3.0]: 25 requirements defined across 6 categories (PHER-EVOL, QUEEN, INT, META, OBS, PRIME)
- [v3.0]: 4 phases defined (32-35) mapping to all 25 requirements
- [Phase 32]: colony-prime() combines queen-read + pheromone-prime into single unified call
- [Phase 32]: Two-level loading: global ~/.aether/QUEEN.md loads first, local extends
- [Phase 32]: QUEEN.md missing = FAIL HARD; pheromones.json missing = warn but continue
- [Phase 32]: 32-02: build.md uses colony-prime() for unified worker context (single call replaces three)
- [Phase 32]: 32-03: init.md calls queen-init at Step 1.6; QUEEN.md template has 5 categories + metadata in HTML comment format
- [Phase 33]: 33-01: learning-observe function records observations with SHA256 content hashing for deduplication
- [Phase 33]: 33-01: Cross-colony accumulation - different colonies contribute to same observation count
- [Phase 33]: 33-01: Threshold detection per wisdom type (philosophy=5, pattern=3, redirect=2, stack=1, decree=0)
- [Phase 33]: 33-02: learning-check-promotion function identifies learnings meeting promotion thresholds
- [Phase 33]: 33-02: Proposals include observation count, threshold, and contributing colonies
- [Phase 33]: 33-03: continue.md displays promotion proposals at phase end (PHER-EVOL-02)
- [Phase 33]: 33-03: User approval required before all promotions (INT-03)
- [Phase 33]: 33-03: queen-promote enforces type validation and thresholds (QUEEN-04)
- [Phase 33]: 33-03: QUEEN.md metadata tracks evolution_log (META-02) and colonies_contributed (META-04)
- [Phase 34]: 34-01: learning-display-proposals shows grouped proposals with checkbox UI, threshold bars, and below-threshold warnings
- [Phase 34]: 34-01: generate-threshold-bar helper with Unicode circles and ASCII fallback
- [Phase 34]: 34-01: All proposals displayed (not just threshold-meeting) to support user override of thresholds
- [Phase 34]: 34-02: parse-selection helper converts 1-indexed user input to 0-indexed array indices with validation
- [Phase 34]: 34-02: learning-select-proposals captures user input, shows preview with below-threshold warnings
- [Phase 34]: 34-02: Confirmation prompt with --yes flag for scripting, --dry-run for testing
- [Phase 34]: 34-03: learning-defer-proposals stores unselected items with deferred_at timestamp and 30-day TTL
- [Phase 34]: 34-03: learning-approve-proposals orchestrates full workflow with batch promotion and undo
- [Phase 34]: 34-03: learning-undo-promotions reverts promotions from QUEEN.md with 24h undo window
- [Phase 34]: 34-03: continue.md integrated with new approval flow and --deferred flag support

### Key Findings from Research
- Subagents cannot spawn other subagents — strip all spawn calls from every converted agent
- Tool inheritance over-permissions agents — explicit tools field required on every agent, no exceptions
- YAML malformation silently drops agents — run `/agents` after every file creation to confirm load
- Vague descriptions kill auto-routing — write as routing triggers, not role labels
- Surveyor XML format is the most mature agent format in Aether — port directly

### Blockers / Concerns
- GSD agent distribution concern resolved: `.claude/agents/ant/` subdirectory keeps Aether agents separate
- Model field behavior in frontmatter unverified — test during Phase 27 implementation

## Session Continuity

Last session: 2026-02-20
Stopped at: Phase 34 complete - tick-to-approve UX fully implemented
Next step: Phase 35 — Wisdom extraction at seal/entomb
