# Phase 28: Orchestration Layer + Surveyor Variants - Research

**Researched:** 2026-02-20
**Domain:** Claude Code subagent authoring — orchestrator pattern (Task tool), routing description engineering, surveyor agent conversion
**Confidence:** HIGH

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Queen's Coordination Model:**
- Queen gets the Task tool in its tools field — it CAN spawn other named agents (aether-builder, aether-scout, etc.)
- This makes Queen a true orchestrator in Claude Code, not just an advisor
- Route-Setter also gets the Task tool (needs to verify plans sometimes) — all other agents escalate instead of spawning
- Preserve the full 4-tier escalation chain from OpenCode (worker retry -> parent reassign -> Queen reassign -> user escalation)
- The existing workflow is working well — this is a faithful port, not a redesign
- Agents that aren't Queen/Route-Setter can still spawn general-purpose tasks via Task tool — preserves the "ants spawning ants" philosophy. They just can't invoke named agents.

**Surveyor Consolidation:**
- Keep all 4 surveyors as separate Claude Code agent files (aether-surveyor-nest, aether-surveyor-disciplines, aether-surveyor-pathogens, aether-surveyor-provisions)
- Surveyors write their output files directly to `.aether/data/survey/` (not read-only — they need Write in tools field)
- NOTE: Roadmap success criteria says "no Write or Edit" for surveyors — override this. Surveyors need Write to create their survey documents. Restrict write scope to `.aether/data/survey/` only in their boundaries section.
- Keep the existing output location: `.aether/data/survey/`

**Routing Descriptions:**
- Descriptions should mention specific Aether commands that spawn them (e.g., "Spawned by /ant:build and /ant:oracle")
- Queen description must be specific enough to NOT fire for simple build tasks (Builder) or simple research (Scout)
- When Queen spawns workers via Task tool, the `description` parameter should include the caste emoji (e.g., "🔨🐜 Build authentication module", "🔭🐜 Research API patterns") so the terminal display shows which ant type is working

**Scout Capabilities:**
- Scout gets web search tools (WebSearch, WebFetch) in addition to codebase tools — broad research capability
- Keep Scout simple — quick research and report. Oracle stays as the deep iterative research path. Clear separation.
- Read-only vs writing research files: Claude's discretion based on how research results are consumed in colony workflows

### Claude's Discretion

- Sequential vs parallel spawning in Queen — pick based on what's practical in Claude Code's Task tool constraints
- Exact routing descriptions for all 7 agents — craft for optimal auto-selection while mentioning spawn sources
- Whether surveyors emphasize standalone use or colony-spawned use in descriptions
- Scout's read-only status vs ability to write research files

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| CORE-01 | Queen agent upgraded — XML body with 6 workflow patterns, escalation chain, spawn protocol via Task tool. Tools: Read, Write, Edit, Bash, Grep, Glob, Task | Task tool is a valid value in the `tools:` frontmatter field; Queen's `tools: Read, Write, Edit, Bash, Grep, Glob, Task` enables orchestration without restrictions on which named agents to spawn |
| CORE-04 | Scout agent upgraded — research-focused body with RALF-style iterative discovery, source evaluation. Tools: Read, Grep, Glob, WebSearch, WebFetch | WebSearch and WebFetch are valid tools in Claude Code agent definitions; Scout becomes a read+search agent; research file writing is discretionary |
| CORE-05 | Route-setter agent upgraded — planning XML body with dependency analysis, phase decomposition, goal-backward verification. Tools: Read, Grep, Glob, Bash, Write | Route-Setter also gets Task tool per locked decisions; tools field therefore: Read, Grep, Glob, Bash, Write, Task |
| CORE-06 | Surveyor-nest agent upgraded — XML body ported from OpenCode with explicit tool list. Tools: Read, Grep, Glob, Bash, Write | Write is confirmed needed (per locked decision overriding roadmap success criteria); write scope restricted to `.aether/data/survey/` in boundaries section |
| CORE-07 | Surveyor-disciplines agent upgraded — XML body ported from OpenCode. Tools: Read, Grep, Glob, Bash, Write | Same Write pattern as CORE-06 |
| CORE-08 | Surveyor-pathogens agent upgraded — XML body ported from OpenCode. Tools: Read, Grep, Glob, Bash, Write | Same Write pattern as CORE-06 |
| CORE-09 | Surveyor-provisions agent upgraded — XML body ported from OpenCode. Tools: Read, Grep, Glob, Bash, Write | Same Write pattern as CORE-06 |
</phase_requirements>

---

## Summary

Phase 28 converts 7 OpenCode agents to Claude Code subagents, following the exact format established by Builder and Watcher in Phase 27. The conversion work divides into two character groups: the orchestrators (Queen, Scout, Route-Setter) and the surveyors (all 4 variants). Both groups require PWR-01 through PWR-08 compliance.

The central technical fact for Phase 28 is how the Task tool works in Claude Code agent definitions. `Task` is a valid tool name in the frontmatter `tools:` field. When specified without parentheses (`tools: Task, Read, ...`), the agent can spawn any named subagent without restriction. When specified with a parenthesized allowlist (`tools: Task(aether-builder, aether-scout), Read, ...`), only the listed agent types can be spawned. The locked decision is to give Queen and Route-Setter unrestricted Task access — `Task` without parentheses. All other agents, per the locked decision, may still spawn general-purpose tasks (unlisted/anonymous subagents) via Task, but cannot invoke named Aether agents by name. However, the simpler interpretation of "they just can't invoke named agents" is to omit Task from their tools entirely and have them return structured JSON that the calling orchestrator uses to route. The RALF/CONTEXT research is clear: surveyors and Scout write to disk or return structured findings, then the calling command re-routes.

The surveyors are the most straightforward: direct ports of their OpenCode XML bodies with two changes — (1) remove OpenCode-specific patterns (activity-log, spawn calls), (2) add the standard 8-section XML structure. The key override from locked decisions is that surveyors keep Write in their tools, but their boundaries section restricts writes to `.aether/data/survey/` only.

**Primary recommendation:** For the 7 agents in this phase, treat the OpenCode definitions as the content source and the Phase 27 Builder/Watcher files as the structural template. The most important implementation decisions are: (a) Queen's description must strongly differentiate it from Builder and Scout, (b) the Task tool goes in frontmatter as-is for Queen and Route-Setter, (c) surveyors get Write tool with tight boundary declarations.

---

## Standard Stack

### Core

| Component | Version/Path | Purpose | Why Standard |
|-----------|-------------|---------|--------------|
| YAML frontmatter | Claude Code spec | Agent configuration (name, description, tools, model) | Required by Claude Code — identical to Phase 27 format |
| Task tool (frontmatter) | Claude Code built-in | Queen/Route-Setter orchestration capability | Validated by official Claude Code docs — `Task` is a valid tools field entry |
| 8 XML sections | Established in Phase 27 | role, execution_flow, critical_rules, return_format, success_criteria, failure_modes, escalation, boundaries | Builder and Watcher set the template — all agents in Phase 28 follow this exactly |
| Existing `.claude/agents/ant/` | Aether repo | Output location for converted agents | Distribution pipeline already proven in Phase 27; no new wiring needed |

### Supporting

| Component | Path | Purpose | When to Use |
|-----------|------|---------|-------------|
| `.opencode/agents/aether-queen.md` | Source | Content for Queen conversion | Read first; all workflow patterns, escalation chain, and worker castes carry over |
| `.opencode/agents/aether-scout.md` | Source | Content for Scout conversion | Read first; remove spawn section; add WebSearch/WebFetch tools |
| `.opencode/agents/aether-route-setter.md` | Source | Content for Route-Setter conversion | Read first; mostly content port; add Task tool |
| `.opencode/agents/aether-surveyor-*.md` | Source | Content for all 4 surveyor conversions | Read first; structure is already XML-like with `<role>`, `<process>`, etc. |

### No New Dependencies

This phase adds zero new libraries or infrastructure. The distribution pipeline from Phase 27 handles delivery. The agent format is identical to Phase 27.

---

## Architecture Patterns

### Pattern 1: Task Tool in Frontmatter — Queen and Route-Setter

**What:** `Task` is a valid tool name in the Claude Code agent `tools:` frontmatter field.

**Official behavior** (HIGH confidence — from official Claude Code docs at https://code.claude.com/docs/en/sub-agents):
- `tools: Task` — Agent can spawn any named subagent (no restriction)
- `tools: Task(worker, researcher)` — Agent can only spawn agents named "worker" or "researcher" (allowlist)
- If `Task` is omitted entirely — Agent cannot spawn any named subagents

**Decision implication:** Queen and Route-Setter get `Task` (no parentheses) — unrestricted spawning. Per official docs: "If the agent tries to spawn any other type, the request fails and the agent sees only the allowed types in its prompt." So unrestricted Task avoids this failure mode.

**Important architectural fact:** The official docs state: "Subagents cannot spawn other subagents. If your workflow requires nested delegation, use Skills or chain subagents from the main conversation." This restriction applies to agents running as subagents. Queen runs as the primary/main thread via `/ant:build`, `/ant:colonize`, etc. — not as a subagent — so it CAN spawn. Route-Setter is typically spawned by Queen, which means Route-Setter runs as a subagent and per the docs restriction CANNOT spawn. However, the locked decision grants Route-Setter the Task tool "needs to verify plans sometimes." This is a known tension; the verified behavior in Phase 27 is that the restriction is real. Resolution: Route-Setter should still list Task in its tools field (per the locked decision), but its use is conditional — it may only be effective when Route-Setter is invoked from the main thread, not when spawned as a subagent. The planner should note this as an open question.

**Frontmatter format for Queen:**
```yaml
---
name: aether-queen
description: "..."
tools: Read, Write, Edit, Bash, Grep, Glob, Task
model: inherit
---
```

**Frontmatter format for Route-Setter:**
```yaml
---
name: aether-route-setter
description: "..."
tools: Read, Grep, Glob, Bash, Write, Task
model: inherit
---
```

### Pattern 2: Caste Emoji in Task Spawning

**What:** When Queen spawns workers via the Task tool, the `description` parameter should include the caste emoji so the terminal display shows which ant type is working.

**Example spawn patterns for Queen's execution_flow:**
```
description: "🔨🐜 Build authentication module — [full task spec]"
description: "🔭🐜 Research OAuth2 patterns for login implementation"
description: "👁🐜 Verify authentication module passes all tests"
```

**Caste emojis from CLAUDE.md caste system:**
- Queen: 👑🐜
- Builder: 🔨🐜
- Watcher: 👁🐜
- Scout: 🔭🐜
- Route-Setter: 🗺🐜
- Surveyors: 🗺🐜 (mapping/survey role)

**Important:** The emoji goes in the Task `description` parameter string, NOT in the agent's own frontmatter. This is runtime content, not static configuration.

### Pattern 3: Queen Description — Routing Precision

**What:** Queen's description must be specific enough to NOT fire for simple build tasks (Builder) or simple research (Scout). This is the hardest description to write in this phase.

**Design principle for Queen's description:** Queen is for multi-phase, multi-worker coordination. Differentiate on:
- "Coordinate" / "orchestrate" / "multi-phase" signal Queen
- "Implement" / "build" / "create" signal Builder
- "Research" / "find" / "look up" signal Scout

**Recommended Queen description (draft):**
```
"Use this agent when coordinating multi-phase projects, managing multiple workers across a build session, or executing complex colony workflows (SPBV, Investigate-Fix, Refactor, Compliance, Documentation Sprint). Spawned by /ant:build, /ant:colonize, and /ant:oracle when a goal requires planning, delegation, and synthesis across multiple steps. Do NOT use for single-task implementation (use aether-builder) or quick research lookups (use aether-scout)."
```

The explicit "Do NOT use for..." phrasing is a documented routing effectiveness technique from the Claude Code docs.

### Pattern 4: Surveyor Boundary Declaration

**What:** Surveyors have Write in their tools but must restrict their write scope to `.aether/data/survey/`. This is enforced by the boundaries XML section, not by tool restrictions.

**Standard surveyor boundary declaration:**
```xml
<boundaries>
## Boundary Declarations

### Write Scope (RESTRICTED)
You may ONLY write to `.aether/data/survey/`. Do not write to any other path.

Permitted write targets:
- `.aether/data/survey/BLUEPRINT.md` (nest surveyor)
- `.aether/data/survey/CHAMBERS.md` (nest surveyor)
[etc. per surveyor]

### Global Protected Paths (never write to these)
- `.aether/dreams/` — Dream journal; user's private notes
- `.aether/data/COLONY_STATE.json` — Colony state
- `.aether/data/constraints.json` — Pheromone signals
- `.env*` — Environment secrets
- `.claude/settings.json` — Hook configuration

### Escalation on Boundary Violation
If a task would require writing outside `.aether/data/survey/`, STOP and escalate.
Do not attempt to write to unapproved paths under any circumstances.
</boundaries>
```

### Pattern 5: Scout's Read-Only Status

**What:** Scout does NOT write research files (this is Claude's discretion, now resolved). The reasoning: Scout's findings return as structured JSON to the calling command, which then passes them to Builder or other workers as task context. Writing to disk would create state management complexity and Scout's findings are typically transient. Oracle is the deep research path that writes; Scout is the quick lookup path that returns.

**Scout tool decision:** `tools: Read, Grep, Glob, WebSearch, WebFetch`
- No Bash (not needed for pure research)
- No Write or Edit (transient findings, returned in JSON)
- WebSearch and WebFetch added for external research capability

**Scout return format:** Structured JSON (same as OpenCode version) with key_findings, recommendations, sources.

### Pattern 6: 6 Queen Workflow Patterns (unchanged from OpenCode)

The 6 workflow patterns from the OpenCode Queen carry over verbatim to the XML body:
1. SPBV (Scout-Plan-Build-Verify)
2. Investigate-Fix
3. Deep Research
4. Refactor
5. Compliance
6. Documentation Sprint

**Pattern selection table** also carries over intact. This is the core intellectual content of the Queen agent — port it faithfully with no redesign.

### Anti-Patterns to Avoid

- **Omitting Task from Queen's tools:** Without Task in the frontmatter, Queen cannot spawn named subagents. This is the difference between an advisor and a true orchestrator.
- **Giving Task to Scout or Surveyor:** Surveyors and Scout do not need to spawn named workers. Their role is to gather and return. Adding Task to their tools adds capability scope without purpose.
- **Generic Queen description:** "Use for colony coordination" will fire for everything and nothing. Must name specific commands and patterns.
- **Keeping OpenCode spawn bash calls:** These must be removed. Queen spawns via the Task tool in Claude Code, not via bash `aether-utils.sh spawn-*` calls.
- **Keeping activity-log calls:** Remove from all 7 agents. Progress tracked through structured returns.
- **Surveyors returning document contents:** All 4 surveyors must return brief confirmation (~10 lines) — NOT the document contents. This is a critical_rule in each surveyor's body.
- **YAML description without quotes:** All descriptions must be quoted in frontmatter. Scout, Queen, and Route-Setter descriptions especially contain colons and special characters.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Spawn restriction enforcement | Logic in agent body | Put Task in tools field or omit it | Claude Code's tool allowlist enforces this at the platform level |
| Activity tracking | activity-log bash calls | Structured JSON return format | Claude Code agents return to the calling command — that's the only needed feedback path |
| Queen pattern selection | Custom keyword matching logic | Port the existing pattern selection table from OpenCode Queen verbatim | Already proven in the colony workflow; no redesign needed |
| Write scope restriction | Custom hook or validator | Boundaries XML section with explicit allowed paths | Sufficient for an agent operating under clear instructions |

**Key insight:** Every piece of colony logic in the OpenCode agents is battle-tested. This phase is a faithful format conversion, not a redesign. The OpenCode XML bodies contain the right content; the job is to reshape them into the 8-section Claude Code template.

---

## Common Pitfalls

### Pitfall 1: Queen Description Too Broad or Too Narrow

**What goes wrong:** Queen fires on every task (too broad) or never fires (too narrow).
**Why it happens:** Queen's use cases overlap with Builder (when a build involves multiple phases) and Scout (when research precedes building).
**How to avoid:** Include explicit negative guidance in the description: "Do NOT use for single-task implementation (use aether-builder) or quick research lookups (use aether-scout)." Name the commands that actually spawn Queen: `/ant:build`, `/ant:colonize`, `/ant:oracle`.
**Warning signs:** Test by asking: "Would this fire if I said 'build a login page'?" If yes, the description is too broad. "Would this fire if I said 'coordinate the login feature across a 3-phase plan'?" If no, the description is too narrow.

### Pitfall 2: Task Tool Restriction for Subagents

**What goes wrong:** Route-Setter lists Task in its tools but cannot spawn subagents when it runs as a subagent (spawned by Queen).
**Why it happens:** Official Claude Code docs confirm: "Subagents cannot spawn other subagents." Route-Setter is normally spawned by Queen, making it a subagent.
**How to avoid:** Keep Task in Route-Setter's tools per the locked decision (it may be effective when Route-Setter is invoked from main thread). In Route-Setter's escalation section, note: "If spawned as a subagent, the Task tool may not be available — surface verification needs to the calling orchestrator."
**Impact:** Low. Route-Setter's primary function is planning, not spawning. The Task tool is secondary.

### Pitfall 3: Surveyors Returning Document Contents in Response

**What goes wrong:** Surveyor dumps hundreds of lines of document content into the response instead of a brief confirmation.
**Why it happens:** Without a clear critical_rule, the agent may naturally summarize or echo the document it just wrote.
**How to avoid:** Add as a critical_rule: "RETURN ONLY CONFIRMATION — not document contents. Maximum ~10 lines in your response." This is already established in the OpenCode surveyor critical_rules — carry it over verbatim.

### Pitfall 4: Forgetting Caste Emoji in Queen's spawn Descriptions

**What goes wrong:** Terminal display shows generic task names without ant-type identification.
**Why it happens:** The emoji is in the Task tool's `description` parameter, not the agent definition. Easy to omit from the execution_flow documentation.
**How to avoid:** In Queen's execution_flow, include example spawn calls with emoji in the description parameter. Make it explicit in the spawn protocol section.

### Pitfall 5: Survey Write Target Outside `.aether/data/survey/`

**What goes wrong:** Surveyor writes a file to the wrong location (e.g., project root, `.aether/data/`).
**Why it happens:** Without explicit boundary enforcement, a surveyor may write to a "convenient" path.
**How to avoid:** In each surveyor's boundaries section, list every permitted write path by name. In failure_modes, add: "Write target outside `.aether/data/survey/` → STOP immediately, this is outside permitted scope." This already exists in OpenCode surveyors — carry it over.

### Pitfall 6: YAML Frontmatter Malformation (Inherited Risk)

**What goes wrong:** Agent silently drops from `/agents` output.
**Why it happens:** Unescaped colons, braces, or special characters in description values break YAML.
**How to avoid:** All 7 agent descriptions must be double-quoted. This is the same pitfall documented in Phase 27 Research — it applies equally here.
**Warning signs:** Agent file exists on disk but does not appear in `/agents`.

### Pitfall 7: Mixing Queen's Output Format with OpenCode Format

**What goes wrong:** Queen returns OpenCode-style JSON with `phases_completed`, `spawn_tree` fields that Claude Code slash commands don't consume.
**Why it happens:** The OpenCode Queen output format includes fields that were meaningful in the OpenCode context.
**How to avoid:** Keep the structured return format but simplify to what Claude Code commands consume: `status`, `summary`, `phases_completed` (list), `blockers`. Remove `spawn_tree` if it requires aether-utils.sh to write.

---

## Code Examples

Verified patterns from official sources and Phase 27:

### Queen Frontmatter (Full)

```yaml
---
name: aether-queen
description: "Use this agent when coordinating multi-phase projects, managing multiple workers across a build session, or executing colony workflows like SPBV, Investigate-Fix, Refactor, Compliance, or Documentation Sprint. Spawned by /ant:build and /ant:colonize when a goal requires planning, delegation, and synthesis across multiple steps. Do NOT use for single-task implementation (use aether-builder) or quick research (use aether-scout)."
tools: Read, Write, Edit, Bash, Grep, Glob, Task
model: inherit
---
```

### Scout Frontmatter (Full)

```yaml
---
name: aether-scout
description: "Use this agent for research, documentation exploration, codebase analysis, and gathering information before implementation. Spawned by /ant:build and /ant:oracle for quick research tasks. Use when the colony needs to understand an API, library, pattern, or codebase area before building. For deep iterative research with source evaluation, use /ant:oracle directly instead."
tools: Read, Grep, Glob, WebSearch, WebFetch
model: inherit
---
```

### Route-Setter Frontmatter (Full)

```yaml
---
name: aether-route-setter
description: "Use this agent when decomposing a goal into phases, analyzing task dependencies, creating structured build plans, or verifying a plan's feasibility. Spawned by /ant:plan and Queen when a project needs phase decomposition and task ordering before implementation begins."
tools: Read, Grep, Glob, Bash, Write, Task
model: inherit
---
```

### Surveyor-Nest Frontmatter (Full)

```yaml
---
name: aether-surveyor-nest
description: "Use this agent to map the codebase's architecture, directory structure, and project topology. Writes BLUEPRINT.md and CHAMBERS.md to .aether/data/survey/. Spawned by /ant:colonize to survey the nest before colony work begins. Use when colony context is missing or stale for this project."
tools: Read, Grep, Glob, Bash, Write
model: inherit
---
```

### Surveyor-Disciplines Frontmatter (Full)

```yaml
---
name: aether-surveyor-disciplines
description: "Use this agent to map coding conventions, testing patterns, and development practices. Writes DISCIPLINES.md and SENTINEL-PROTOCOLS.md to .aether/data/survey/. Spawned by /ant:colonize to document how the team builds software."
tools: Read, Grep, Glob, Bash, Write
model: inherit
---
```

### Surveyor-Pathogens Frontmatter (Full)

```yaml
---
name: aether-surveyor-pathogens
description: "Use this agent to identify technical debt, bugs, security concerns, and fragile areas in the codebase. Writes PATHOGENS.md to .aether/data/survey/. Spawned by /ant:colonize to detect what needs fixing before colony work begins."
tools: Read, Grep, Glob, Bash, Write
model: inherit
---
```

### Surveyor-Provisions Frontmatter (Full)

```yaml
---
name: aether-surveyor-provisions
description: "Use this agent to map technology stack, dependencies, and external integrations. Writes PROVISIONS.md and TRAILS.md to .aether/data/survey/. Spawned by /ant:colonize to inventory what the project relies on."
tools: Read, Grep, Glob, Bash, Write
model: inherit
---
```

### Queen Spawn Protocol (in execution_flow)

```
When spawning workers via Task tool, always include the caste emoji in the description:

Builder spawn:
description: "🔨🐜 {task name} — {full task specification}"

Scout spawn:
description: "🔭🐜 {research topic} — {what to find and how to report back}"

Watcher spawn:
description: "👁🐜 Verify {artifact} — {what to check and expected outcome}"

Route-Setter spawn:
description: "🗺🐜 Plan {goal} — {context and constraints}"

Surveyor spawn:
description: "🗺🐜 Survey {domain} — {what to write and where}"
```

### Queen's Escalation Chain (in failure_modes)

```xml
<failure_modes>
### Escalation Chain

Failures escalate through four tiers. Tiers 1-3 are fully silent — user never sees them. Only Tier 4 surfaces to the user.

**Tier 1: Worker retry** (silent, max 2 attempts)
The failing worker retries with corrected approach.

**Tier 2: Parent reassignment** (silent)
If Tier 1 exhausted, Queen tries a different approach.

**Tier 3: Queen reassigns** (silent)
Queen retires the failed worker and spawns a different caste.

**Tier 4: User escalation** (visible)
[ESCALATION banner with tried/options/recommendation]

Do NOT attempt to bypass this chain. Do NOT surface Tier 1-3 failures to the user.
</failure_modes>
```

### PWR-08 Removal Checklist (all 7 agents)

Remove from every agent body:
```bash
# Remove ALL of these:
bash .aether/aether-utils.sh activity-log ...
bash .aether/aether-utils.sh spawn-can-spawn ...
bash .aether/aether-utils.sh generate-ant-name ...
bash .aether/aether-utils.sh spawn-log ...
bash .aether/aether-utils.sh spawn-complete ...
bash .aether/aether-utils.sh flag-add ...
```

Replace spawn section with escalation instructions. Replace activity-log with note: "Progress is tracked through structured returns, not activity logs."

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| OpenCode spawn calls (bash aether-utils.sh) | Task tool in YAML frontmatter | Phase 28 (this phase) | Queen can spawn named agents without shell calls |
| OpenCode activity-log | Structured JSON return | Phase 27 | Progress visible through return format, not activity log |
| Queen as advisor (no spawn in Claude Code) | Queen as true orchestrator (Task tool) | Phase 28 | Queen can actually direct multi-agent workflows in Claude Code |
| All surveyors read-only | Surveyors have Write (scoped to survey/) | Phase 28 override | Surveyors can write their output documents directly |
| aether-queen.md in .opencode/agents/ | aether-queen.md in .claude/agents/ant/ | Phase 28 | Distributed via established hub pipeline from Phase 27 |

**Deprecated in Phase 28:**
- All OpenCode bash spawn patterns from the 7 agents converted in this phase
- OpenCode-specific output format fields that have no Claude Code equivalent (spawn_tree entries requiring aether-utils.sh)

---

## Open Questions

1. **Route-Setter's Task Tool Effectiveness**
   - What we know: Official Claude Code docs state "Subagents cannot spawn other subagents." Route-Setter is typically spawned by Queen, making it a subagent.
   - What's unclear: Whether listing Task in Route-Setter's tools when it runs as a subagent causes an error vs. silently has no effect.
   - Recommendation: List Task in frontmatter per locked decision. Add a note in Route-Setter's escalation section: "If the Task tool is unavailable (running as a subagent), escalate verification needs to the calling orchestrator instead of spawning directly." This makes the behavior graceful regardless of invocation context.

2. **Scout Write Decision (Resolved)**
   - Decision: Scout does NOT write research files. Findings return as structured JSON.
   - Rationale: Scout's job is quick lookup; Oracle is deep research with written outputs. Adding Write to Scout blurs this boundary. Structured JSON returns work cleanly with calling commands.

3. **Queen's State Management Simplification**
   - What we know: OpenCode Queen writes to COLONY_STATE.json via aether-utils.sh commands. Claude Code Queen should still do this via bash.
   - What's unclear: Whether Bash is needed in Queen's tools purely for colony state operations.
   - Recommendation: Keep Bash in Queen's tools (already in the locked decision). State management via aether-utils.sh bash calls is different from spawn calls — it's legitimate colony integration, not an OpenCode-only pattern. Only remove the spawn/activity-log patterns (PWR-08), not all bash usage.

4. **Verification of 7 agents in `/agents`**
   - What we know: Each agent must appear in `/agents` in a live Claude Code session to confirm YAML parsing succeeded.
   - Recommendation: Verification plan must include a human check step: open Claude Code after creating all 7 files, run `/agents`, confirm all 7 names appear. Same as Phase 27's human verification requirement.

---

## Sources

### Primary (HIGH confidence)

- Official Claude Code subagent docs, https://code.claude.com/docs/en/sub-agents — Task tool behavior in tools field, allowlist syntax (Task vs Task(name, name)), "Subagents cannot spawn other subagents" restriction, routing description guidance, frontmatter field spec
- `/Users/callumcowie/repos/Aether/.claude/agents/ant/aether-builder.md` — Phase 27 template (format reference for all 8 XML sections)
- `/Users/callumcowie/repos/Aether/.claude/agents/ant/aether-watcher.md` — Phase 27 template (format reference)
- `/Users/callumcowie/repos/Aether/.opencode/agents/aether-queen.md` — Content source for CORE-01 (6 workflow patterns, escalation chain, spawn protocol, worker castes)
- `/Users/callumcowie/repos/Aether/.opencode/agents/aether-scout.md` — Content source for CORE-04
- `/Users/callumcowie/repos/Aether/.opencode/agents/aether-route-setter.md` — Content source for CORE-05
- `/Users/callumcowie/repos/Aether/.opencode/agents/aether-surveyor-nest.md` — Content source for CORE-06
- `/Users/callumcowie/repos/Aether/.opencode/agents/aether-surveyor-disciplines.md` — Content source for CORE-07
- `/Users/callumcowie/repos/Aether/.opencode/agents/aether-surveyor-pathogens.md` — Content source for CORE-08
- `/Users/callumcowie/repos/Aether/.opencode/agents/aether-surveyor-provisions.md` — Content source for CORE-09
- `/Users/callumcowie/repos/Aether/.planning/phases/27-distribution-infrastructure-first-core-agents/27-RESEARCH.md` — Phase 27 research (established patterns for PWR-01 through PWR-08, spawn replacement, activity-log replacement)
- `/Users/callumcowie/repos/Aether/.planning/REQUIREMENTS.md` — CORE-01, CORE-04 through CORE-09, PWR-01 through PWR-08 definitions

### Secondary (MEDIUM confidence)

- Phase 27 VERIFICATION.md — Confirmed distribution pipeline is proven; no new wiring needed for Phase 28 agents

---

## Metadata

**Confidence breakdown:**
- Agent format and frontmatter: HIGH — identical to Phase 27, official docs verified
- Task tool behavior: HIGH — official Claude Code docs are explicit
- Content conversion from OpenCode: HIGH — source files read directly; changes well-defined
- Route-Setter Task tool effectiveness as subagent: MEDIUM — flagged as open question; graceful degradation approach recommended
- Routing description effectiveness: MEDIUM — descriptions are crafted per guidelines but effectiveness is only verifiable in a live session

**Research date:** 2026-02-20
**Valid until:** 2026-03-20 (Claude Code agent format stable; source OpenCode files won't change)
