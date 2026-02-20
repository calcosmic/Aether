# Project Research Summary

**Project:** v2.0 Worker Emergence — Claude Code Subagents for Aether Colony
**Domain:** Claude Code subagent format translation; multi-runtime agent distribution
**Researched:** 2026-02-20
**Confidence:** HIGH

## Executive Summary

This milestone is a translation task, not a greenfield design. All 22 Aether ant worker roles exist as fully defined OpenCode agents in `.opencode/agents/`. The goal is to create functionally equivalent Claude Code subagents in `.claude/agents/` so that the 34 existing slash commands — which are already wired to spawn `subagent_type="aether-builder"` etc. via the Task tool — can resolve those types to registered agent definitions rather than falling back to a general-purpose agent with an injected role description. The fallback exists and documents itself explicitly in `build.md`; this milestone eliminates it by creating the actual files.

The recommended approach is: write 22 new agent files using YAML frontmatter with XML-structured bodies (matching the pattern of the 11 existing GSD agents in `.claude/agents/`), with explicit tool restrictions per agent class and description fields written as routing signals rather than role labels. Then make three small code changes — add `.claude/agents/` to `package.json` files array, add a sync block to `setupHub()` in `bin/cli.js`, and add a corresponding destination block to `update-transaction.js` — so the agents distribute through the hub to all target repos exactly as OpenCode agents do today. The hub gets a new `agents-claude/` path to keep the two registries strictly separated throughout the distribution chain.

The primary risk is quality decay during conversion: developers copying OpenCode agents verbatim will carry over spawn machinery that silently fails in Claude Code, Aether-specific shell calls that break when `.aether/` is absent, and vague descriptions that prevent auto-routing. Every one of these failure modes is silent — no error, just wrong behavior. The mitigation is a structured conversion checklist applied per agent before merging: strip spawn calls, make `aether-utils.sh` calls optional with `|| true`, write descriptions as routing triggers not role labels, and verify each file loads correctly with `/agents` immediately after creation.

## Key Findings

### Recommended Stack

The stack for this milestone is an extension of the existing Aether stack. No new dependencies are required. The Claude Code subagent format is YAML frontmatter plus a markdown or XML body stored in `.claude/agents/`. All 11 existing GSD agents in that directory use YAML frontmatter with XML-structured bodies, and research confirms this format outperforms flat markdown for complex multi-step agents because named XML sections let the model navigate content under context pressure.

**Core technologies:**
- **YAML frontmatter + XML body**: Required fields are `name` (lowercase letters and hyphens only, no emoji) and `description` (routing signal). Critical optional fields: `tools` (allowlist — must be explicit for every agent), `color` (visual identity in Claude Code UI). XML body for orchestrators and complex workers; flat markdown for simpler specialists.
- **`aether-` name prefix**: All 34 existing slash commands already reference `subagent_type="aether-builder"` etc. The naming convention is not a design decision — it is already locked in by the existing command infrastructure.
- **`inherit` model default**: Use for most agents so the operator's model choice propagates. Reserve `haiku` for read-only researchers (Scout, Chronicler), `sonnet` for implementors and verifiers. Avoid `opus` for all agents — cost multiplies with every colony spawn.
- **No new npm dependencies**: Agent files are markdown with YAML frontmatter. Distribution uses the existing hub sync pipeline with one new hub path (`agents-claude/`).

### Expected Features

This milestone delivers one Claude Code subagent file per ant caste — 22 files total. Research prioritizes them into three tiers based on build workflow dependency.

**Must have (P1 — milestone cannot function without these):**
- `aether-queen` — Orchestrator; spawns all others; already referenced by `/ant:build`, `/ant:init`, `/ant:plan`
- `aether-builder` — Core implementation worker; most-invoked agent in any build
- `aether-watcher` — Quality gate; every build phase requires verification before advancing
- `aether-scout` — Research agent; required by SPBV and Deep Research workflow patterns
- `aether-route-setter` — Planning agent; precedes every implementation phase
- All 4 surveyor variants (`aether-surveyor-nest`, `aether-surveyor-disciplines`, `aether-surveyor-pathogens`, `aether-surveyor-provisions`) — Colony relies on them for codebase context; XML bodies already written in OpenCode

**Should have (P2 — needed for all 6 workflow patterns):**
- `aether-keeper`, `aether-tracker`, `aether-probe`, `aether-weaver`, `aether-auditor`

**Defer (P3 — specialized, invoked only in niche workflows):**
- `aether-chaos`, `aether-archaeologist`, `aether-ambassador`, `aether-chronicler`, `aether-gatekeeper`, `aether-measurer`, `aether-includer`, `aether-sage`

**Key differentiators vs current OpenCode agents (add in conversion):** Explicit tool restrictions per agent class; description fields as routing triggers ("Use this agent for... Spawned by..."); self-contained operation that degrades gracefully when colony not initialized; standardized failure modes with explicit retry limits; `<success_criteria>` checklists enabling agent self-verification before returning.

**Anti-features to avoid:** Copying OpenCode agents verbatim (carries incompatible spawn machinery); injecting workers.md content into every agent body (the 4,200 token problem); requiring an initialized colony for all agents; generating ant names inside agents (creates infrastructure dependency); emoji in `name` field (breaks YAML parsing).

### Architecture Approach

The architecture adds a second agent registry path to the existing distribution pipeline, keeping Claude Code and OpenCode agents strictly separated throughout the chain. The 34 slash commands already point to the correct `subagent_type` values and require no changes. Only three files require modification beyond creating the 22 new agent files.

**Major components:**
1. **`.claude/agents/aether-*.md` (22 new files)** — Claude Code subagent definitions. Translate role content from OpenCode equivalents; add runtime-appropriate frontmatter (`tools`, `color`). These are the primary deliverable.
2. **`package.json` files array** — Add `".claude/agents/"` entry so agents are included in the npm package. One line, additive, low risk.
3. **`bin/cli.js` setupHub()** — Add ~10-line sync block after existing `.opencode/agents/` sync. Copies `.claude/agents/` to `~/.aether/system/agents-claude/` when `npm install -g .` runs.
4. **`bin/lib/update-transaction.js`** — Add sync block that pulls `~/.aether/system/agents-claude/` to `.claude/agents/` in target repos during `aether update`. Also add to `targetDirs` for dirty-file checks and `verifyIntegrity()`.
5. **Hub path `~/.aether/system/agents-claude/`** — New intermediate store. Kept separate from `~/.aether/system/agents/` (OpenCode) to prevent cross-contamination.

**Key pattern:** Context injection via spawn prompt, not agent startup file reads. The agent file defines role and discipline; the slash command injects runtime context (goal, task, pheromones, archaeology results, queen wisdom) into the Task tool call's prompt parameter. This is already how `build.md` works — preserve it.

**Data flow:** User runs `/ant:build` → slash command reads colony state → constructs spawn prompt → Task tool call with `subagent_type="aether-builder"` → Claude Code resolves to `.claude/agents/aether-builder.md` → agent executes with injected context → returns structured JSON → slash command updates colony state.

### Critical Pitfalls

1. **Vague descriptions kill auto-routing** — Write descriptions as routing triggers ("Use this agent for code implementation, file creation, and build tasks. Spawned by ant-queen."), not role labels ("A builder agent for coding tasks"). Description is the most important line in the file. The difference between an agent that routes correctly and one that never gets invoked is the description.

2. **Subagents cannot spawn other subagents** — Hard platform constraint. OpenCode agents contain spawn machinery (`spawn-can-spawn`, `spawn-log`, Task tool calls) that silently produces nothing in Claude Code. Strip all spawn calls from every converted agent. The Queen in the main conversation or slash command context is the only spawner.

3. **Aether-specific machinery breaks silently** — `bash .aether/aether-utils.sh activity-log ...` fails when `.aether/` is absent. Make all `aether-utils.sh` calls optional with `|| true` or remove them from agent bodies. Replace output format with whatever the calling slash command expects.

4. **Tool inheritance over-permissions agents** — Omitting the `tools` field causes an agent to inherit all tools from the parent session, including Write and Edit for agents intended as read-only. Every agent must have an explicit `tools` allowlist. No exceptions.

5. **YAML malformation silently drops agents** — A single invalid field value (`model: claude-sonnet-4-5` instead of `model: sonnet`, or emoji in `name`) causes the agent to silently fail to load with no error message. Run `/agents` after every file creation to confirm the agent loaded.

## Implications for Roadmap

Based on combined research, the work breaks into four natural phases driven by dependency and risk:

### Phase 1: Core Caste Agents + Distribution Infrastructure

**Rationale:** The build workflow requires builder, watcher, and chaos to exist before any real test is possible. Distribution infrastructure must be proven before declaring any agent "shipped" — an agent that only works in the source repo is not shipped. These are coupled: verifying distribution requires agents to distribute.

**Delivers:** Working end-to-end proof that Task tool invocation resolves to a registered Claude Code agent instead of the fallback. `aether update` delivers agents to target repos. The fallback comment in `build.md` becomes unreachable for core castes.

**Addresses:** `aether-builder`, `aether-watcher`, `aether-chaos` (core build castes) plus the 3-file distribution infrastructure change (package.json, cli.js, update-transaction.js).

**Avoids:**
- Pitfall 10 (Distribution chain gap) — verify `npm pack --dry-run` includes agents and `aether update` delivers them before Phase 2 begins
- Pitfall 11 (YAML malformation) — run `/agents` after every file creation
- Pitfall 2 (Spawn calls silently fail) — apply conversion checklist to first agents to prove the strip process

### Phase 2: Orchestration Layer + Surveyor Variants

**Rationale:** Queen and Route-Setter are orchestrators that coordinate other agents — they are meaningless until the agents they coordinate exist. Scout enables research phases. All 4 surveyor variants can be ported directly from existing OpenCode XML (low conversion risk, high completeness value). With these 7 agents added, all 6 Queen workflow patterns are structurally supported.

**Delivers:** Complete orchestration layer for Claude Code. Colony workflows can plan, research, and coordinate — not just implement.

**Addresses:** `aether-queen`, `aether-route-setter`, `aether-scout`, all 4 surveyor variants (7 agents).

**Avoids:**
- Pitfall 7 (Over-routing from broad descriptions) — Queen's description must not make it a catch-all; read all descriptions side-by-side after definition to check for ambiguity
- Pitfall 8 (Context budget waste) — surveyor agents already have appropriate XML length from OpenCode; preserve length, don't pad

### Phase 3: Specialist Agents (P2)

**Rationale:** P2 agents complete the remaining workflow patterns but are not on the critical path for the core build loop. These are lower complexity (flat markdown, read-only or scoped write access) and can be ported quickly now that the conversion checklist is proven from Phases 1 and 2.

**Delivers:** Full P2 agent set: `aether-keeper`, `aether-tracker`, `aether-probe`, `aether-weaver`, `aether-auditor`.

**Avoids:**
- Pitfall 4 (Aether machinery breaks silently) — apply the same conversion checklist proven in Phase 1
- Pitfall 5 (Tool over-permission) — read-only agents need explicit tools allowlist; verify agent cannot write files before merging

### Phase 4: Niche Agents (P3) + Integration Validation

**Rationale:** P3 agents are invoked only in specialized workflows (documentation sprints, resilience testing, dependency audits). Deferring them until Phase 4 lets the team validate the conversion approach fully on higher-value agents first. Integration validation at the end confirms output format compatibility between all converted agents and the slash commands that consume them.

**Delivers:** Remaining 8 agents (`aether-chaos`, `aether-archaeologist`, `aether-ambassador`, `aether-chronicler`, `aether-gatekeeper`, `aether-measurer`, `aether-includer`, `aether-sage`) plus end-to-end integration tests confirming colony state is correctly updated after agent runs.

**Avoids:**
- Pitfall 9 (Output format mismatch) — run each slash command end-to-end before declaring Phase 4 complete; verify state files updated correctly
- Pitfall 3 (Name collision) — run `/agents` in a fresh session to confirm project-scope agents win over any user-level duplicates from previous installs

### Phase Ordering Rationale

- Distribution infrastructure is in Phase 1 (not Phase 4) because shipping agents without verifying distribution means they only work in the Aether source repo — exactly the failure mode documented in Pitfall 10.
- Orchestrators (Queen, Route-Setter) are in Phase 2 not Phase 1 because they require the core caste agents to already exist before meaningful orchestration is possible.
- The 4 surveyor variants are grouped in Phase 2 because they share an output directory (`.aether/data/survey/`) and have no inter-dependencies; they can be ported in parallel with no file conflicts.
- P3 agents are Phase 4 because they have no role in the primary build loop. Deferring them lets the conversion checklist be proven on higher-value agents first.
- Integration validation is last because it tests the full system, not individual agents.

### Research Flags

Phases needing deeper research during planning:
- **Phase 1 (Distribution infrastructure):** The specific lines to modify in `bin/cli.js` and `update-transaction.js` are documented in ARCHITECTURE.md but should be re-verified against the actual current file state before modifying. The distribution chain changed significantly in the v4.0 restructuring; confirm line numbers are current before writing code.
- **Phase 1 (GSD agent distribution concern):** Adding `.claude/agents/` to `package.json` files array will distribute the 11 GSD agents alongside the 22 Aether agents. GSD agents reference GSD-specific tooling (`gsd-tools.cjs`) that is not meaningful in non-GSD repos. Decide before Phase 1 implementation begins: filter sync by prefix (`aether-*.md` only) or move Aether agents into `.claude/agents/ant/` subdirectory. The subdirectory approach is architecturally cleaner.

Phases with well-documented patterns (standard, no additional research needed):
- **Phase 2 (Surveyor variants):** Surveyor XML bodies already exist in `.opencode/agents/`. Conversion is frontmatter translation plus tool restrictions. No design work needed.
- **Phase 3 and Phase 4 (Specialist agents):** All follow flat markdown pattern with explicit tools. Apply the conversion checklist mechanically.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Format verified against official Claude Code docs and 4 working repos with production subagents. YAML field values confirmed against actual loaded agents in `.claude/agents/`. |
| Features | HIGH | All 22 agent roles exist in OpenCode; tool access matrix derived from direct inspection of GSD and community agent patterns. Priority tiers based on build workflow analysis and existing slash command dependency mapping. |
| Architecture | HIGH | Distribution pipeline verified by reading actual source files (`package.json`, `bin/cli.js` lines 956-989, `update-transaction.js` lines 861-865). Direct source inspection; no assumptions. |
| Pitfalls | HIGH | Primary pitfalls from official Claude Code docs (subagent spawn constraint, tool inheritance model). Secondary from direct inspection of failing vs working agents across 3 repos. YAML load failures confirmed from GitHub issue threads. |

**Overall confidence:** HIGH

### Gaps to Address

- **GSD agent distribution concern:** Adding `.claude/agents/` to `package.json` files array will distribute the 11 GSD agents to every Aether user's repo. GSD agents reference `gsd-tools.cjs` which is not present in non-GSD repos. Resolution options: (a) filter the sync function to copy only `aether-*.md` files, or (b) move Aether agents into `.claude/agents/ant/` subdirectory. This decision must be made before Phase 1 distribution work begins — it affects the hub path and UpdateTransaction destination.

- **Output format compatibility:** The slash commands that consume agent output were written for OpenCode agent output schemas. ARCHITECTURE.md documents the expected JSON fields (`ant_name`, `task_id`, `status`, `summary`, `tool_count`, `files_created`, `files_modified`, `tests_written`, `blockers`). Read the slash commands that consume each agent's output before writing the output format section of the converted agent. Confirm schemas match before declaring Phase 4 complete.

- **Model field behavior:** STACK.md recommends `inherit` as the default. PITFALLS.md suggests `haiku` for Scout/Chronicler and `sonnet` for Builder/Watcher. FEATURES.md anti-features section lists `model` frontmatter as potentially ignored. These sources give slightly different guidance. During Phase 1 implementation, test whether the `model` field in frontmatter actually affects subagent model selection and set the field accordingly across all 22 agents.

## Sources

### Primary (HIGH confidence)
- Official Claude Code documentation at `https://code.claude.com/docs/en/sub-agents` — complete frontmatter field reference, tool inheritance model, subagent spawn constraint, auto-routing mechanics
- `/Users/callumcowie/repos/Aether/package.json` — files array, confirmed `.claude/agents/` absent from current distribution
- `/Users/callumcowie/repos/Aether/bin/cli.js` lines 956-989 — setupHub() sync logic, existing agent sync pattern
- `/Users/callumcowie/repos/Aether/bin/lib/update-transaction.js` lines 861-865 — UpdateTransaction sync target paths, agents distribution destination
- `/Users/callumcowie/repos/Aether/.claude/commands/ant/build.md` — subagent_type references confirmed, fallback comment documented
- `/Users/callumcowie/repos/Aether/.claude/agents/gsd-executor.md`, `gsd-planner.md`, `gsd-verifier.md` — working Claude Code agent reference implementations
- `/Users/callumcowie/repos/Aether/.opencode/agents/*.md` — all 22 OpenCode agent definitions (direct inspection for translation)
- GitHub issues #8558 and #5185 (anthropics/claude-code) — YAML load failure patterns confirmed

### Secondary (MEDIUM confidence)
- `/Users/callumcowie/repos/everything-claude-code/agents/*.md` — 12 community agents; quality range examples; routing pattern comparison
- `/Users/callumcowie/repos/cosmic-dev-system/agents/*.md` — 11 CDS agents; confirms GSD XML patterns independently
- Practitioner blogs (claudekit.cc, claudefa.st, pubnub.com) — subagent common mistakes and routing patterns

---
*Research completed: 2026-02-20*
*Ready for roadmap: yes*
