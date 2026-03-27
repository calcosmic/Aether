# Project Research Summary

**Project:** Aether v2.5 Smart Init
**Domain:** Multi-agent colony orchestration -- intelligent initialization, wisdom pipeline population, and per-caste model routing
**Researched:** 2026-03-27
**Confidence:** HIGH

## Executive Summary

Aether v2.5 addresses three interconnected problems. First, `/ant:init` is purely mechanical -- it takes a raw goal string, writes template files, and provides zero intelligence about the codebase being initialized. Second, the wisdom pipeline exists as complete shell infrastructure but has a "liveness gap": QUEEN.md sections and hive brain entries remain template-only because the pipeline depends on AI agents reliably extracting learnings (they often skip it) and hive-promote only fires during `/ant:seal` (most colonies never get sealed). Third, per-caste model routing needs a mechanism that survived the v1 archive because Claude Code's Task tool does not pass environment variables to subagents -- the GSD system solved this with the `model` parameter on Task calls.

The recommended approach is to make init intelligent (research scan, structured prompt generation, approval loop) while keeping the changes as additive as possible. The wisdom pipeline needs deterministic fallbacks at break points where AI-dependent paths silently skip. Model routing should use Claude Code's model slot names (opus/sonnet) passed directly to Task calls, not environment variables.

**The critical tension in this research:** Stack and Architecture research both propose a QUEEN.md v3 format with new `## ` sections (Intent, Vision, Governance, Goals, Architecture Notes). Pitfalls research explicitly identifies this as the most dangerous change possible because 7+ downstream consumers parse QUEEN.md by exact `## Section Name` header matching via awk/grep. The resolution: do NOT add new `## ` sections to QUEEN.md. Instead, write charter content into existing sections (Intent as a Codebase Pattern, Vision as a Codebase Pattern, Governance rules as User Preferences or Codebase Patterns) using existing queen-promote/queen-write-learnings functions that already handle format correctly. This preserves the "living charter" value while eliminating the format-breakage risk. The charter *concept* survives; the new-section mechanism does not.

## Key Findings

### Recommended Stack

All research agrees: zero new npm dependencies. The changes are bash subcommands, Markdown command files, updated templates, and new agent `.md` files.

**Core technologies (unchanged):**
- Bash 4+ + jq 1.6+ -- all shell utilities, no new runtime deps
- Markdown commands (init.md) -- the LLM IS the UI for approval loops
- awk/sed for QUEEN.md section manipulation -- existing pattern, proven reliable

**New components:**
- `.aether/utils/scan.sh` -- new utils module for lightweight repo scanning (<2s target)
- `init-research` subcommand -- repo surface scan returning JSON (tech stack, complexity, prior colonies)
- `init-generate-prompt` subcommand -- bash+jq assembly of structured prompt from goal + research
- `_queen_write_governance()` -- writes charter content to existing QUEEN.md sections (NOT new sections)
- `colony-name` subcommand -- DRY helper for colony name extraction from COLONY_STATE.json
- 6 new agent `.md` files -- Oracle (model: opus, read-heavy with Write tool) and Architect (model: opus, design decisions) for Claude Code + OpenCode + packaging mirrors
- Hive-promote hook in `/ant:continue` -- mid-colony wisdom promotion (currently seal-only)
- Deterministic learning extraction fallback in `continue-advance.md` -- git-diff-based when AI skips extraction

### Expected Features

**Must have (table stakes) -- the intelligence chain:**
- T1: Lightweight repo scan (<2 seconds) -- without this, init remains blind
- T2: Auto-generate structured colony prompt from goal + research -- bash+jq, deterministic
- T3: User approval loop -- LLM-mediated (display Markdown, wait for text response)
- T4: Charter content in QUEEN.md on first init -- using existing write functions
- T5: Subsequent inits update charter without resetting colony state -- update-only, never destroy wisdom
- Wisdom pipeline deterministic fallbacks -- ensure learnings always accumulate even when AI skips
- Hive-promote in continue flow -- cross-colony wisdom accumulates mid-colony, not just at seal

**Should have (competitive differentiators):**
- D2: Suggested pheromones from research (FOCUS/REDIRECT suggestions in approval prompt)
- D3: Colony complexity estimation (informs planning depth)
- T6: Intelligent colonize suggestion (detect stale/missing survey)
- T7: Prior colony knowledge inheritance (chambers/tunnels context)
- Per-caste model routing via Task tool model parameter

**Defer (v2+):**
- D1: Research-aware charter suggestions (infer governance from codebase patterns) -- needs validation
- D4: Chambers/tunnels context in approval prompt -- nice-to-have
- D5: Goals section auto-populate from /ant:plan -- cross-command integration complexity
- D6: Architecture Notes from research -- can be derived from T1 data later

### Architecture Approach

The architecture follows three independent streams that converge through QUEEN.md:

**Stream 1: Smart Init** transforms init.md from mechanical file creation into: scan -> generate -> approve -> charter. New scan.sh module (10th domain module) provides `_scan_repo()`, `_scan_quick()`, `_scan_survey_status()`, `_scan_chambers()`. Prompt generation is bash+jq assembly (NOT LLM generation) for determinism. Approval loop uses Claude Code's native execution model (display, stop, wait for user response, continue).

**Stream 2: Wisdom Pipeline Hardening** adds deterministic fallbacks at four break points in the existing pipeline: (1) builder synthesis JSON missing `learning.patterns_observed` -- add git-diff pattern extraction, (2) AI agent skipping learning extraction during continue -- add bash-based fallback, (3) hive-promote only fires at seal -- add to continue flow, (4) colony name extraction can silently fail -- add dedicated subcommand.

**Stream 3: Per-Caste Model Routing** uses Claude Code's Task tool `model` parameter (opus/sonnet slots), NOT environment variable passing (proven not to work in archived v1 routing). Requires: test infrastructure refactor first (184 hardcoded model names), then core routing in build-wave.md, then proxy config documentation.

**Major components:**
1. scan.sh -- lightweight repo scanning, new utils module
2. queen-governance functions -- charter content management in queen.sh
3. Oracle + Architect agents -- 6 new `.md` files for spawnable agents
4. colony-name subcommand -- DRY helper for name extraction
5. continue-advance.md hardening -- deterministic fallbacks + hive-promote
6. build-wave.md routing -- model slot resolution + Task tool model parameter

### Critical Pitfalls

1. **QUEEN.md format fragility (CRITICAL)** -- 7+ downstream consumers parse QUEEN.md by exact `## Section Name` header matching. Adding new sections breaks all of them. Resolution: write charter content into existing sections using existing write functions. NEVER add new `## ` headers.

2. **Wisdom pipeline "liveness gap" (CRITICAL)** -- wisdom stays empty because of a chain of soft dependencies where AI agents skip extraction and hive-promote only fires at seal. Resolution: deterministic fallbacks at each break point; move hive-promote to continue flow.

3. **Model routing via environment variables (CRITICAL)** -- proven not to work (v1 archived). Resolution: use Task tool `model` parameter directly, following GSD's working pattern.

4. **184 hardcoded model names in tests (HIGH)** -- changing model-profiles.yaml without updating test mocks causes mass test failures. Resolution: centralize mock profiles before any YAML changes.

5. **Approval loop fatigue (MODERATE)** -- multiple sequential approval prompts cause decision fatigue. Resolution: single batched proposal, accept-all/modify/reject in one interaction.

6. **GLM-5 looping despite proxy constraints (MODERATE)** -- four escape conditions where GLM-5 can bypass proxy temperature/top_p constraints. Resolution: application-level loop detection, document Prime-as-turbo option.

7. **Token budget exhaustion from smart-init content (MODERATE)** -- charter content could crowd out colony-earned wisdom in the 8,000-char budget. Resolution: cap smart-init content at 500 chars, prioritize colony-earned over smart-init in trimming.

## Implications for Roadmap

Based on research, suggested phase structure:

### Phase 1: Scan Module + Test Infrastructure

**Rationale:** Two independent foundations that must be laid first. scan.sh is the data source for all smart init features. Test infrastructure refactor (centralize 184 model name mocks) must happen before any model routing YAML changes. Neither depends on the other, so they can run in parallel.

**Delivers:** `.aether/utils/scan.sh` with 5 functions, centralized test helpers for model profiles, scan-repo/scan-quick dispatch cases.

**Addresses:** T1 (lightweight repo scan), Pitfall 4 (hardcoded test names)

**Avoids:** Starting YAML changes before test mocks are centralized (Pitfall 4), adding scanning logic inline in init.md (anti-pattern)

### Phase 2: Queen Charter + Wisdom Pipeline Fallbacks

**Rationale:** Charter management and wisdom fallbacks both touch queen.sh and the continue flow but in different ways. Charter writes are additive (new functions). Wisdom fallbacks modify existing playbook steps. Both are independent of each other but both depend on understanding the current QUEEN.md format deeply.

**Delivers:** `_queen_write_governance()` in queen.sh (writes to existing sections, NOT new sections), deterministic learning extraction fallback in continue-advance.md, hive-promote hook in continue flow, colony-name subcommand.

**Addresses:** T4 (charter content), T5 (safe re-init), wisdom pipeline break points 2-4

**Avoids:** Adding new `## ` sections to QUEEN.md (Pitfall 1), writing directly to QUEEN.md instead of using existing functions

### Phase 3: Init.md Refactor + Agent Definitions

**Rationale:** Now that the foundation (scan.sh, charter functions, wisdom fallbacks) is in place, the user-facing init command can be refactored to use them. The Oracle and Architect agent definitions are independent but belong here because they complete the "missing agents" gap identified in Stack research.

**Delivers:** Refactored init.md with research -> generate -> approve -> charter flow, OpenCode init.md mirror, 6 new agent `.md` files (Oracle + Architect for Claude Code + OpenCode + packaging mirrors).

**Addresses:** T2 (prompt generation), T3 (approval loop), T4+T5 (charter management in init flow), missing agent definitions

**Avoids:** Making approval loop complex (anti-pattern 4 from Architecture), prompt generation as LLM (should be bash+jq)

### Phase 4: Colony-Prime Integration + Model Routing Core

**Rationale:** Colony-prime needs to extract and inject charter content into worker prompts (now that charter content exists in QUEEN.md). Model routing core implements the Task tool model parameter mechanism. Both modify how workers receive context.

**Delivers:** Updated `_extract_wisdom()` for charter content extraction, updated trim order in `_colony_prime()`, model slot resolution in build-wave.md, model logging in spawn-log.

**Addresses:** Charter-to-worker flow, per-caste model routing mechanism (Pitfalls 1, 5, 7 from model routing research)

**Avoids:** Environment variable passing (proven broken), breaking v2 QUEEN.md format reading

### Phase 5: Proxy Verification + Documentation + Validation

**Rationale:** Final integration and hardening. Verify model routing works end-to-end, document config swap workflow, validate QUEEN.md format integrity, update CLAUDE.md.

**Delivers:** End-to-end model verification, config swap documentation, QUEEN.md format validator (queen-validate subcommand), CLAUDE.md updates, full integration test (init -> plan -> build -> verify charter flows to workers).

**Addresses:** Pitfall 2 (config swap), Pitfall 6 (lifecycle edge cases for non-build commands), documentation accuracy

### Phase Ordering Rationale

- Phases 1-2 are parallel-safe (no dependencies between them)
- Phase 3 depends on both Phase 1 (scan.sh) and Phase 2 (charter functions)
- Phase 4 depends on Phase 2 (charter content must exist to extract it) and Phase 1 (test infrastructure must be centralized for model routing)
- Phase 5 depends on all prior phases (end-to-end validation)
- The ordering respects the dependency graph from Architecture research while incorporating the wisdom pipeline and model routing streams

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 3 (Init.md Refactor):** Approval loop UX is untested -- the "LLM-mediated approval" pattern needs validation that it actually feels natural. Consider `/gsd:research-phase` to prototype the UX.
- **Phase 4 (Model Routing Core):** The Task tool `model` parameter behavior with GLM proxy needs empirical validation. The GSD precedent is HIGH confidence but Aether's multi-worker spawn pattern is different from GSD's single-executor pattern. Consider `/gsd:research-phase` to prove the mechanism with a single test case before building the full system.

Phases with standard patterns (skip research-phase):
- **Phase 1 (Scan Module):** Well-documented pattern (9 existing utils modules), straightforward bash functions
- **Phase 2 (Charter + Wisdom):** Follows existing queen.sh function patterns exactly; wisdom fallbacks are well-understood
- **Phase 5 (Validation):** Standard integration testing and documentation

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack (Smart Init) | HIGH | Direct codebase analysis of init.md, queen.sh, all templates. Zero new dependencies. Well-understood execution model. |
| Stack (Wisdom Pipeline) | HIGH | Direct codebase analysis of learning.sh (1553 lines), queen.sh (1242 lines), hive.sh (562 lines). Root cause analysis traced through full pipeline. |
| Stack (Model Routing) | HIGH | Archived v1 proves env-var approach fails. GSD provides working Task tool model parameter pattern. |
| Features | HIGH | Feature list derived from direct codebase gap analysis and PROJECT.md requirements. Competitor analysis is MEDIUM (web search rate-limited). |
| Architecture (Smart Init) | HIGH | All command files, utils, and templates analyzed. scan.sh placement follows established pattern. |
| Architecture (Wisdom) | HIGH | Break points identified through full pipeline tracing. Fix locations precise (specific line numbers). |
| Pitfalls (Smart Init) | HIGH | QUEEN.md consumer analysis exhaustive (7+ consumers identified with line numbers). Token budget math verified. |
| Pitfalls (Model Routing) | HIGH | 184 test assertions counted. Parser divergence traced through both bash and Node.js paths. GLM-5 constraint escape conditions documented from proxy config. |

**Overall confidence:** HIGH

### Gaps to Address

- **Approval loop UX (LOW confidence):** No user testing data on whether the LLM-mediated approval pattern feels natural. The research says it works within Claude Code's execution model, but UX perception is unvalidated. Mitigation: prototype early in Phase 3 and adjust based on feel.

- **Token budget math with charter content (MEDIUM confidence):** Architecture research estimates charter adds ~1000-1500 chars. Pitfalls research recommends a 500-char cap. The actual impact depends on how much charter content users accept. Mitigation: measure after Phase 3 implementation and adjust caps.

- **GLM-5 behavior under per-caste routing (MEDIUM confidence):** The proxy mapping (opus -> glm-5, sonnet -> glm-5-turbo) is assumed to work based on proxy config. But GLM-5's tendency to loop when spawning sub-workers (Pitfall 3) could make Prime-as-opus dangerous. Mitigation: consider Prime-as-turbo despite being a "reasoning" caste; add loop detection early.

- **Competitor feature accuracy (LOW confidence):** Web search was rate-limited for both FEATURES.md and STACK.md competitor analysis. Claims about Cursor/Windsurf/Copilot Workspace/Aider are based on training data, not current documentation. This does not affect implementation decisions but may affect positioning claims.

## The QUEEN.md Tension: Resolution

The core tension between Stack/Architecture (new v3 format with charter sections) and Pitfalls (never add new sections) requires a clear recommendation:

**Recommendation: Write charter content into existing v2 sections. Do NOT create a v3 format with new `## ` headers.**

Specifically:
- Colony Intent -> written to `## Codebase Patterns` with `[charter]` tag
- Colony Vision -> written to `## Codebase Patterns` with `[charter]` tag
- Governance rules -> written to `## User Preferences` with `[charter]` tag
- Goals -> written to `## Codebase Patterns` with `[charter]` tag (auto-updated by /ant:plan)

This uses existing queen-promote and queen-write-learnings functions which already handle section formatting, METADATA updates, dedup, and evolution log entries. The charter *concept* (intent, vision, governance, goals) survives as content within the existing structure. The format remains v2-compatible. All 7+ downstream consumers continue to work without modification.

The trade-off: charter content is less visually prominent in QUEEN.md (mixed into Codebase Patterns rather than in its own section). This is acceptable because the primary consumer of charter content is colony-prime (which extracts by content, not by section header) and the approval prompt (which shows charter in a formatted display regardless of where it is stored in QUEEN.md).

## Sources

### Primary (HIGH confidence)
- Direct codebase analysis of init.md (388 lines), queen.sh (1242 lines), learning.sh (1553 lines), hive.sh (562 lines), pheromone.sh (colony-prime, ~800 lines)
- Direct analysis of all 22 existing agent definitions in `.claude/agents/ant/`
- `.aether/archive/model-routing/README.md` -- v1 routing failure analysis
- `.claude/get-shit-done/references/model-profile-resolution.md` -- GSD working pattern
- model-profiles.yaml, model-profiles.js, model-verify.js -- current model system
- `.aether/templates/QUEEN.md.template` -- v2 format specification
- 580+ tests across 6 test files with 184 model name occurrences
- `.planning/PROJECT.md` -- milestone scope and user feedback

### Secondary (MEDIUM confidence)
- User testing feedback from PROJECT.md: "init is purely mechanical", "users forget colonize", "subsequent inits reset everything"
- CLI UX research on approval fatigue, "present-don't-commit" pattern (training data)
- npm/Terraform/cargo approval UX patterns (established precedent)
- Existing completion-report.md inheritance logic in init.md Step 2.6

### Tertiary (LOW confidence)
- Competitor feature analysis (Cursor, Windsurf, Copilot Workspace, Aider) -- web search rate-limited
- Whether approval loop UX feels natural (needs user testing)
- Whether research-aware charter suggestions produce high-quality governance text (untested)

---
*Research completed: 2026-03-27*
*Ready for roadmap: yes*
