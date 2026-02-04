# Project Research Summary

**Project:** Aether v4.4 -- Colony Hardening & Real-World Readiness
**Domain:** Multi-agent colony system hardening (Claude Code-native, stigmergic coordination)
**Researched:** 2026-02-04
**Confidence:** HIGH

## Executive Summary

Aether v4.4 addresses 23 actionable findings from the first real-world field test. The research reveals a system whose core architecture is sound but whose execution layer has critical bugs (pheromone decay growing instead of decaying, activity log overwriting instead of appending), missing safety mechanisms (no file conflict prevention, no error phase attribution), and unnecessary friction (10 manual approvals for 5 straightforward phases). The good news: every fix stays within the existing bash+jq stack with zero new dependencies. The bad news: the most ambitious v4.4 goal -- recursive ant spawning -- is blocked by a verified Claude Code platform constraint (Task tool unavailable to subagents, GitHub #4182). The architecture must route around this reality rather than fight it.

The recommended approach is a six-phase build that starts with bug fixes and safety (decay math, activity log, error attribution, conflict prevention), then UX friction reduction (context clear prompting, auto-continue), then colony intelligence improvements (adaptive complexity, watcher scoring calibration, multi-ant colonization), then new automation capabilities (auto-reviewer, pheromone-first flow, tech debt reporting), then architecture evolution (two-tier learning, spawn tree engine), and finally polish (archivist ant, pheromone docs, colonizer visuals). This ordering is driven by three constraints: broken foundations invalidate every feature built on top; UX friction causes user abandonment faster than missing features; and the spawn tree engine depends on validated platform behavior and calibrated quality signals.

The key risks are: (1) building recursive spawning without validating the platform constraint first -- a 30-minute test prevents weeks of wrong architecture; (2) adding auto-review before fixing the flat 8/10 watcher scores -- automation of meaningless scores is worse than no automation; and (3) global learning promotion creating stale cross-project knowledge that silently degrades output quality. All three are mitigable with the phased approach and explicit validation gates described below.

## Key Findings

### Recommended Stack

No new dependencies. All v4.4 work uses the existing bash+jq stack. The critical stack insight is that the pheromone decay bug was NOT a formula error -- the formula in `aether-utils.sh` is mathematically correct. The root cause was deployment: during the field test, `aether-utils.sh` did not exist in the target repo, so Claude fell back to LLM-computed exponential math and got the sign wrong, producing growth instead of decay. The fix is two-part: defensive guards in the utility (clamp negative elapsed, cap at initial strength, floor at zero) AND eliminating all LLM math fallback paths.

**Core technologies (unchanged):**
- **bash 4.0+ / jq 1.6+**: All utilities, decay math, state management -- already the foundation
- **ANSI escape codes**: Color-coded output per caste (cyan=colonizer, yellow=route-setter, green=builder, magenta=watcher, blue=scout) -- no tput dependency needed
- **JSON files**: Two-tier learning storage (`~/.aether/` for global, `.aether/data/` for project) -- consistent with existing state management
- **noclobber pattern**: Atomic lock acquisition for file conflict prevention -- already implemented in `file-lock.sh`

**Critical version note:** jq's `exp()` function (used for decay) returns IEEE754 doubles -- sufficient for signal strengths rounded to 3-6 decimal places. LLMs must NEVER compute transcendental functions; always delegate to `aether-utils.sh`.

**New aether-utils.sh subcommands (5 total, ~43 lines, stays under 400-line budget):**
- `learning-promote`, `learning-global-read`, `error-add-phased`, `activity-log-append`, `progress-bar`

### Expected Features

**Must have (table stakes -- every production multi-agent system has these):**
- File conflict prevention in parallel execution -- field-tested failure (notes 10, 13)
- Persistent cross-session memory with structured retrieval
- Error attribution to execution context (phase, worker, caste)
- State persistence before context clear -- non-negotiable per field note 5
- Auto-continue / batch execution -- eliminates 10 manual approvals
- Automated code review at build boundaries with calibrated scoring
- Adaptive complexity scaling -- full colony overhead kills simple projects (field note 31)
- Task delegation depth limits with anti-loop protection

**Should have (differentiators -- no competitor has these):**
- Stigmergic conflict prevention via pheromone file-ownership signals
- Multi-perspective colonization (3 colonizers with synthesis)
- Pheromone-driven planning (signals shape plan before creation)
- Bayesian caste learning (track which castes succeed on which task types)
- Tiered learning with project-to-global promotion (unique cross-project meta-learning)
- Tech debt surfacing as colony output (aggregate cross-phase issues)
- Pheromone recommendations to user (ants guide the Queen)

**Defer (v5+):**
- Unlimited recursive delegation -- anti-feature; context degrades at each level, vulnerable to recursive blocking attacks
- Agent-to-agent direct messaging -- destroys stigmergic model
- Complex memory hierarchies (vector DB, embeddings) -- overkill for JSON-scale data
- Web dashboard for colony visualization -- breaks CLI-only constraint

### Architecture Approach

The architecture remains Queen-mediated hub-and-spoke, with four new subsystems layered on top: (1) a spawn tree engine where workers signal sub-spawn needs through structured output blocks and the Queen fulfills them, working around the Task tool nesting limitation; (2) a two-tier learning system with project-local memory and global cross-project knowledge in `~/.aether/`; (3) an adaptive complexity system with three modes (LIGHTWEIGHT/STANDARD/FULL) set at colonization time via a single `mode` field in COLONY_STATE.json; and (4) auto-spawned lifecycle ants (reviewer after builders, archivist at milestones) that reuse existing worker specs with modified prompts rather than adding new castes.

**Major components:**
1. **Spawn Tree Engine** -- replaces flat wave execution with tree-structured delegation; workers include `SPAWN REQUEST` blocks in output; Queen parses and fulfills; depth tracked logically in COLONY_STATE.json
2. **Two-Tier Learning** -- `memory.json` (project, existing) + `~/.aether/learnings.json` (global, new); promotion via occurrence counting across projects; global learnings injected as FEEDBACK pheromones at `/ant:init`
3. **Adaptive Complexity** -- single `mode` field in COLONY_STATE.json; colonizer assesses file count, language diversity, dependency density, task count; inline conditional checks in all commands (no duplicate command files)
4. **Lifecycle Ants** -- reviewer (watcher-ant in "review mode") auto-spawned after builders; archivist (architect-ant in "hygiene mode") auto-spawned at FULL-mode milestones; advisory only, never blocking
5. **Same-File Conflict Prevention** -- Phase Lead groups overlapping-file tasks to same worker at planning time; file-lock.sh provides byte-level safety net; no git worktrees or container isolation needed

### Critical Pitfalls

1. **Recursive spawning blocked by Claude Code platform (CP-1)** -- Task tool unavailable to subagents. Validate with a 30-minute test before designing any recursive feature. If blocked, use Queen-mediated spawn tree (workers request, Queen fulfills). Never attempt `claude -p` subprocess hack.

2. **Context telephone at delegation depth (CP-2)** -- Information degrades at each delegation level. Include verbatim colony goal at EVERY spawn depth. Limit effective depth to 2 until depth-3 proves value. Add "delegation chain" section to every Task prompt showing full hierarchy.

3. **Auto-reviewer creates blocking bottleneck (CP-3)** -- Must fix watcher scoring rubric FIRST. Reviewer must be advisory only (never blocking). Max 2 build-review iterations per task, then log remaining as tech debt. Only CRITICAL findings trigger rebuild.

4. **Same-file parallel write conflicts (CP-4)** -- Already observed in field test. Fix at planning time: tasks touching same file go to same worker. Sequential fallback for unavoidable overlaps. Do NOT use git worktrees.

5. **Global learning creates stale knowledge (CP-5)** -- Start with empty global tier, manual promotion only. Cap at 50 entries with decay. Tag learnings by tech stack; filter at retrieval time. Never auto-promote without user approval gate.

## Implications for Roadmap

Based on combined research, here is the recommended six-phase structure.

### Phase 1: Bug Fixes & Safety Foundation
**Rationale:** Broken foundations invalidate every feature built on top. These are field-tested bugs with known fixes.
**Delivers:** Reliable pheromone decay, persistent activity logs, phase-attributed errors, wired decision logging, same-file conflict prevention
**Addresses:** Pheromone decay math (field note 17), activity log overwrite (field note 19), error phase attribution (field note 18), decision log wiring (field note 20), same-file conflicts (field notes 10, 13)
**Avoids:** CP-4 (file conflicts), pheromone growth bug
**Stack:** Defensive guards in `aether-utils.sh` (`activity-log-append`, `error-add-phased`), conflict prevention rule in Phase Lead prompt
**Risk:** LOW -- all fixes are well-understood with verified root causes

### Phase 2: Critical UX & Friction Reduction
**Rationale:** Without these, users abandon the system due to friction, not capability. Field note 5 marks context clear prompting as non-negotiable.
**Delivers:** Context-clear-safe commands (every command ends with "safe to /clear"), auto-continue mode (`/ant:continue --all`), pheromone-first flow (colonize suggests pheromone injection before planning)
**Addresses:** Context clear prompting (field note 5), auto-continue (field note 26), pheromone-first flow reordering
**Avoids:** User abandonment from 10 manual approvals; state loss on context clear
**Stack:** Prompt text changes in command files; `validate-state` call before clear suggestion
**Risk:** LOW -- prompt text changes and UX flow reordering only

### Phase 3: Colony Intelligence & Quality Signals
**Rationale:** Calibrated quality signals are prerequisites for auto-review (Phase 4) and adaptive complexity (Phase 5). Multi-ant colonization and aggressive parallelism unlock colony-level improvements.
**Delivers:** Watcher scoring rubric with meaningful variance, multi-ant colonization with synthesis, aggressive wave parallelism, Phase Lead auto-approval for low-complexity plans
**Addresses:** Watcher scoring (field note 24), multi-ant colonization (field note 7), parallelism (field note 10), auto-approval (field note 22)
**Avoids:** CP-3 prerequisite (review fatigue -- must calibrate scoring before adding automation)
**Stack:** Rubric definition in watcher-ant.md; colonize.md multi-colonizer spawning; build.md parallelism in wave execution
**Risk:** MEDIUM -- watcher rubric needs empirical validation to confirm score variance

### Phase 4: Automation & New Capabilities
**Rationale:** With calibrated quality signals in place, automation becomes meaningful. Auto-reviewer, tech debt reporting, and animated indicators all depend on reliable scoring and persistent logs.
**Delivers:** Auto-spawned reviewer (advisory, post-wave), auto-spawned debugger (on test failure), tech debt report generation, pheromone recommendations after builds, animated build indicators (progress bars, caste colors)
**Addresses:** Auto-reviewer/debugger (field note 8), tech debt surfacing (field note 11), pheromone recommendations (field note 9), animated indicators (field note 15)
**Avoids:** CP-3 (review fatigue -- reviewer is advisory only, severity-gated display, max 2 iterations)
**Stack:** build.md Step 5c modification for reviewer auto-spawn; ANSI progress bars via `progress-bar` subcommand; tech debt aggregation from activity log
**Risk:** MEDIUM -- auto-reviewer false positive rate needs tuning; animated output limited by Task tool buffering (static progress bars are the fallback)

### Phase 5: Architecture Evolution
**Rationale:** Two-tier learning and the spawn tree engine are the most complex additions. They benefit from having all preceding infrastructure (bug fixes, calibrated scoring, auto-review) in place. Platform constraint validation must happen before spawn tree implementation.
**Delivers:** Two-tier learning system (project + global with manual promotion), spawn tree engine (Queen-mediated recursive delegation), adaptive complexity mode (LIGHTWEIGHT/STANDARD/FULL per phase)
**Addresses:** Tiered learning (field note 12), recursive spawning (field note 23), adaptive complexity (field notes 31, 21)
**Avoids:** CP-1 (platform limitation -- validate first), CP-2 (context telephone -- depth limit 2), CP-5 (stale knowledge -- manual promotion, cap at 50), CP-6 (wrong mode -- user confirmation always)
**Stack:** `~/.aether/learnings.json` (new global store), COLONY_STATE.json `mode` and `spawn_tree` fields, 3 new aether-utils.sh subcommands, build.md major rewrite for tree execution
**Risk:** HIGH -- spawn tree is the core execution loop rewrite; two-tier learning promotion heuristics are unproven; adaptive mode thresholds need empirical tuning

### Phase 6: Polish & Safety Rails
**Rationale:** Lowest priority, highest sensitivity to user trust. Ship last with maximum safety rails and dry-run enforcement.
**Delivers:** Organizer/archivist ant (report-only file staleness), pheromone user documentation, colonizer visual output restoration, learning promotion mechanism (auto-suggestion with batch user approval)
**Addresses:** Archivist ant (field note 14), pheromone docs (field note 16), colonizer visuals (field note 15), learning promotion UX
**Avoids:** CP-7 (false deletion -- report only, protected file patterns, dry-run enforcement for first 3 runs)
**Stack:** architect-ant.md with archivist task prompt; protected file pattern list; documentation in pheromone command files
**Risk:** LOW-MEDIUM -- archivist has trust risk if safety rails are insufficient; documentation is straightforward

### Phase Ordering Rationale

- **Phase 1 before everything:** Bug fixes remove invalid system behavior that would corrupt testing of all subsequent features. Conflict prevention is required before any increase in parallelism.
- **Phase 2 before Phase 3:** UX friction reduction lets developers actually USE the system through multi-phase sessions, generating the field data needed to validate Phase 3 improvements.
- **Phase 3 before Phase 4:** Calibrated watcher scoring is a hard prerequisite for meaningful auto-review. Adding auto-review on top of flat 8/10 scores is theater.
- **Phase 4 before Phase 5:** Automation features (auto-reviewer, tech debt) validate the quality signal pipeline that adaptive complexity and the spawn tree engine depend on for mode selection and sub-spawn quality gates.
- **Phase 5 before Phase 6:** Architecture evolution provides the infrastructure (global learning tier, spawn tree) that the archivist and learning promotion mechanism build upon.
- **Phase 6 last:** Archivist and learning promotion are high-sensitivity features that benefit from all prior infrastructure and maximum user trust built through successful earlier phases.

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 3 (Colony Intelligence):** Watcher scoring rubric design requires testing against intentionally varied code quality to confirm meaningful score differentiation. Multi-colonizer synthesis pattern is novel -- no documented precedent.
- **Phase 5 (Architecture Evolution):** Spawn tree engine requires a 30-minute platform validation test before any implementation. Adaptive mode thresholds (file count, language count, task count) need empirical calibration. Two-tier learning promotion heuristics (substring matching for deduplication) are crude and may need refinement.

Phases with standard patterns (skip research-phase):
- **Phase 1 (Bug Fixes):** All fixes have verified root causes and known solutions. Defensive math guards, append mode, field addition are straightforward.
- **Phase 2 (UX):** Context clear prompting and auto-continue are prompt text changes with no architectural risk.
- **Phase 4 (Automation):** Auto-reviewer follows well-documented patterns (Zencoder post-execution verification, Amazon Q review agent). ANSI progress bars are a solved problem.
- **Phase 6 (Polish):** Archivist is report-only (no architectural risk). Documentation is documentation.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Zero new dependencies. All solutions verified against existing codebase. Decay math root cause confirmed by testing actual field data. |
| Features | MEDIUM-HIGH | Table stakes validated against AutoGen, CrewAI, LangGraph, OpenAI Agents SDK. Differentiators are novel (no competitor comparison possible). Anti-features well-supported by industry failures. |
| Architecture | HIGH | Based on direct codebase analysis of all 13 commands, 6 worker specs, 3 utilities. Platform constraints verified via Claude Code documentation and GitHub issues. |
| Pitfalls | HIGH | 5 of 7 critical pitfalls grounded in Aether's own field test data. Remaining 2 supported by multi-agent systems research (Google DeepMind, ROMA framework, 30 Failure Modes study). |

**Overall confidence:** HIGH

### Gaps to Address

- **Recursive spawning platform validation:** Must test whether Task tool is truly unavailable to subagents in current Claude Code version before designing spawn tree. A 30-minute test at the start of Phase 5 planning resolves this definitively.
- **Watcher scoring rubric calibration:** No existing data on what makes a "good" rubric for Aether's code review context. Must test against varied quality code during Phase 3 implementation. Expect 2-3 iterations to calibrate.
- **Adaptive mode thresholds:** File count, language count, and task count thresholds for LIGHTWEIGHT/STANDARD/FULL mode selection are educated guesses. Need 5+ real project runs to validate. Start conservative (default to STANDARD/FULL, only trigger LIGHTWEIGHT for obviously trivial projects).
- **Global learning substring matching:** The proposed deduplication mechanism (case-insensitive substring match) will produce false positives for short strings and false negatives for paraphrased learnings. Acceptable for v4.4 MVP but should be revisited if global tier exceeds 20 entries.
- **Auto-continue user agency:** Design question -- full auto (run all phases) vs semi-auto (auto-continue but pause on watcher failures). Field data from Phase 2 deployment will inform the right default.
- **CLI animation limitations:** Task tool buffers all output. Live spinners and streaming progress are impossible. Static progress bars between worker completions are the only viable pattern. This is a platform constraint, not a gap to fix.

## Sources

### Primary (HIGH confidence -- verified by testing or direct analysis)
- Aether v4.3 codebase: 13 commands, 6 worker specs, 3 utility files, 6 state files
- v5 Field Notes: 32 notes from 2026-02-04 live test on filmstrip project
- Decay math verification: jq `exp()` tested with known values; negative elapsed confirmed as root cause
- [Claude Code Task tool limitation -- GitHub #4182](https://github.com/anthropics/claude-code/issues/4182)
- [Claude Code subagent documentation](https://code.claude.com/docs/en/sub-agents)
- [Anthropic multi-agent research system](https://www.anthropic.com/engineering/multi-agent-research-system)

### Secondary (MEDIUM-HIGH confidence -- verified with official sources + multiple community sources)
- Multi-agent coordination: [Google DeepMind Scaling Agent Systems](https://arxiv.org/html/2512.08296v1), ROMA framework, [Galileo coordination strategies](https://galileo.ai/blog/multi-agent-coordination-strategies)
- Competitor analysis: AutoGen, CrewAI, LangGraph, OpenAI Agents SDK official documentation
- File conflict prevention: [Swarm-IOSM](https://dev.to/rokoss21/parallel-agents-are-easy-shipping-without-chaos-isnt-1kek), Cursor 2.0 worktree approach, Claude multi-agent file locking
- Memory architectures: [CrewAI memory docs](https://docs.crewai.com/en/concepts/memory), [G-Memory hierarchical MAS](https://arxiv.org/abs/2506.07398), [Agentic Memory](https://arxiv.org/html/2601.01885v1)
- Code review accuracy: [CodeAnt analysis](https://www.codeant.ai/blogs/ai-code-review-accuracy), Stack Overflow 2025 Developer Survey

### Tertiary (MEDIUM confidence -- single source or community patterns)
- Two-tier memory promotion heuristics: [AIS practical memory patterns](https://www.ais.com/practical-memory-patterns-for-reliable-longer-horizon-agent-workflows/), [arXiv Memory OS](https://arxiv.org/abs/2506.06326)
- Claude Code swarm orchestration: [community gist -- TeammateTool pattern](https://gist.github.com/kieranklaassen/4f2aba89594a4aea4ad64d753984b2ea)
- Dead code cleanup: [Meta SCARF framework](https://engineering.fb.com/2023/10/24/data-infrastructure/automating-dead-code-cleanup/), [Varonis archival best practices](https://www.varonis.com/blog/4-secrets-for-archiving-stale-data-efficiently)

---
*Research completed: 2026-02-04*
*Ready for roadmap: yes*
