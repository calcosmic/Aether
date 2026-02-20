# Feature Research: Claude Code Subagents for Aether Colony

**Domain:** Claude Code subagent system — creating Claude Code `.claude/agents/` definitions for all 22 Aether ant worker types
**Researched:** 2026-02-20
**Confidence:** HIGH (drawn from direct inspection of 22 existing OpenCode agent definitions, 11 existing GSD Claude Code agents, 12 community agents from everything-claude-code, 11 CDS agents, and Claude Code subagent documentation patterns)

---

## Research Scope

Aether already has 22 agent definitions in `.opencode/agents/`. Each needs a Claude Code equivalent in `.claude/agents/`. This milestone is a TRANSLATION task, not a design task. The agent roles, boundaries, output formats, and disciplines already exist — the question is how to express them in the Claude Code subagent format effectively.

Key questions answered in this document:
1. Which agents should exist as standalone subagents vs being handled by slash commands?
2. What tool sets should different agent types have?
3. How should agent descriptions be written for effective Task tool routing?
4. What patterns from GSD/CDS/community agents work well, and which fail?

---

## Claude Code Subagent Format — What Works

From inspecting 11 GSD agents, 11 CDS agents, and 12 community agents, the effective patterns are:

### Frontmatter Fields

```yaml
---
name: agent-name
description: Routing trigger phrase. Use this agent when X. Spawned by Y.
tools: Read, Write, Edit, Bash, Grep, Glob   # explicit list, not array brackets
color: yellow                                  # optional, visual only
---
```

**Description is the routing key.** The Task tool uses the description to select agents. Effective descriptions follow the pattern:
- `"Use this agent for X. Spawned by Y."` — GSD pattern, HIGH confidence
- `"Use PROACTIVELY when planning X, refactoring Y."` — community pattern, MEDIUM confidence
- Specificity matters: `"for code implementation, file creation, command execution"` beats `"for coding tasks"`

**Tools list is real and enforced.** Agents only have access to tools in their frontmatter. The GSD executor has `Read, Write, Edit, Bash, Grep, Glob`. The GSD verifier has `Read, Write, Bash, Grep, Glob`. Read-only agents should list `Read, Grep, Glob, Bash` (Bash for `git log` etc.) but NOT Write, Edit.

**Name must match filename.** `name: aether-builder` lives in `aether-builder.md`. The name appears in UI and is referenced programmatically.

### Body: What to Include

From comparing effective (GSD/CDS) vs less effective (community) agents:

**Effective bodies have:**
- A role statement that is concrete, not aspirational ("You execute PLAN.md files atomically" vs "You are a senior developer")
- Explicit workflow steps with numbered actions, not prose paragraphs
- Specific output formats with worked examples — JSON schema with example values, not just field names
- Failure modes: what to do when things go wrong, with severity tiers
- Success criteria: a checklist the agent can self-verify against
- Read-only boundary declarations: explicit list of paths and operations the agent must not perform

**Less effective bodies have:**
- Long prose explanations of philosophy without procedural content
- Output formats described in abstract ("return a report") without structure
- No failure modes — agent is implicitly expected to always succeed
- Assumed tool availability without checking what tools are actually in frontmatter

### Key Structural Insight: XML vs Flat Markdown

The GSD executor and planner use XML `<section>` structure within the body. The surveyor agents (the two highest-performing Aether agents) also use XML. The community agents and the flat-markdown Aether agents use prose.

XML structure outperforms flat markdown for complex agents because:
- Named sections let the LLM jump to relevant content under context pressure
- Step names create addressable milestones (`<step name="verify_artifacts">`)
- Nested elements make parent/child relationships unambiguous
- Section boundaries prevent content from the output format bleeding into the workflow

**Recommendation:** Use XML structure for orchestrator-class agents (Queen, Route-Setter, Prime Worker) and complex output agents (Builder, Watcher, Surveyor variants). Use flat markdown with clear headers for simpler read-only agents (Scout, Archaeologist, Chaos, Auditor, etc.).

---

## Feature Landscape

### Table Stakes (Agent Definitions That Must Exist)

Every ant worker type that the Queen can spawn needs a Claude Code definition. Without these, the Task tool cannot route to the correct agent type. The OpenCode definitions exist for all 22 — the question is which ones need full definitions vs minimal stubs.

| Agent | Why Essential | Complexity | Notes |
|-------|--------------|------------|-------|
| `aether-builder` | Core implementation worker; all code-writing tasks route here | HIGH | Full XML structure; TDD discipline; deviation rules; failure modes |
| `aether-watcher` | Quality gate; no phase advances without watcher approval | HIGH | Full XML; evidence-based verification protocol; 6-phase quality gate |
| `aether-scout` | Research; used in SPBV and Deep Research patterns | MEDIUM | Flat markdown; read-only; web search enabled |
| `aether-queen` | Orchestrator; spawns all other workers | HIGH | Full XML; 6 workflow patterns; escalation chain |
| `aether-keeper` | Knowledge curation + Architect mode | MEDIUM | Flat markdown; architect mode activation trigger |
| `aether-chronicler` | Documentation writing | LOW | Flat markdown; write-enabled for docs only |
| `aether-probe` | Test generation and coverage | MEDIUM | Flat markdown; write-enabled for test files only |
| `aether-weaver` | Code refactoring | MEDIUM | Flat markdown; write-enabled; strict no-behavior-change constraint |
| `aether-chaos` | Resilience testing | LOW | Flat markdown; read-only; 5-scenario structure |
| `aether-archaeologist` | Git history analysis | LOW | Flat markdown; read-only; git commands via Bash |
| `aether-ambassador` | Third-party API integration | MEDIUM | Flat markdown; web fetch enabled |
| `aether-auditor` | Code review with specialized lenses | MEDIUM | Flat markdown; read-only; Guardian mode activation |
| `aether-gatekeeper` | Dependency and supply chain security | LOW | Flat markdown; read-only; Bash for npm audit etc. |
| `aether-measurer` | Performance profiling | MEDIUM | Flat markdown; Bash for profiling commands |
| `aether-includer` | Accessibility auditing | LOW | Flat markdown; read-only |
| `aether-sage` | Analytics and trend analysis | LOW | Flat markdown; read-only; colony state reader |
| `aether-tracker` | Bug investigation and root cause | MEDIUM | Flat markdown; read-only; systematic debugging |
| `aether-route-setter` | Planning and decomposition | MEDIUM | Flat markdown or XML; write-enabled for plan files |
| `aether-surveyor-nest` | Architecture + directory mapping | MEDIUM | Full XML (already has it); write to survey/ only |
| `aether-surveyor-disciplines` | Conventions + testing patterns | MEDIUM | Full XML; write to survey/ only |
| `aether-surveyor-pathogens` | Tech debt + code health | MEDIUM | Full XML; write to survey/ only |
| `aether-surveyor-provisions` | Dependencies + environment | MEDIUM | Full XML; write to survey/ only |

### Differentiators (What Makes Claude Code Agents Effective vs OpenCode Agents)

These features are not in the existing OpenCode agents but should be added in the Claude Code versions based on evidence from GSD/CDS agent patterns.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Structured description with routing trigger phrase | Claude Code routes agents via description text; explicit "Use this agent when..." phrasing dramatically improves auto-routing accuracy | LOW | Every description should start with "Use this agent for..." or "Spawned by..." |
| Explicit tool lists in frontmatter | OpenCode agents don't have tool restrictions; Claude Code does; explicit lists prevent agents from using tools they shouldn't | LOW | Read-only agents: no Write/Edit. Builder needs all six. Scout needs WebSearch/WebFetch. |
| Activity logging calls removed or made optional | OpenCode agents log to `.aether/aether-utils.sh activity-log`; Claude Code agents running standalone don't need this infrastructure dependency | LOW | Either make logging optional ("if aether-utils.sh is available, log") or remove from Claude Code versions |
| Self-contained operation | OpenCode agents rely on colony state, spawn protocols, and aether-utils.sh; Claude Code agents should function even outside an initialized colony | MEDIUM | Claude Code agents are used via Task tool from any context; must not hard-fail if COLONY_STATE.json doesn't exist |
| `<success_criteria>` as self-verification checklist | GSD agents all have explicit success checklists the agent runs before returning; OpenCode agents have these in some cases but not consistently | LOW | Copy GSD pattern: checklist of verifiable conditions before reporting complete |
| Failure modes with explicit retry and escalation | GSD agents define minor vs major failures with specific retry limits and escalation format; OpenCode agents have this in some files inconsistently | MEDIUM | Standardize across all 22 agents: minor=retry once, major=stop+escalate |
| `<read_only>` boundary declaration as first constraint | Explicitly listing what an agent MUST NOT touch prevents accidental writes to colony state, dreams, or protected paths | LOW | Pattern already in some OpenCode agents; apply consistently to all read-only agents |

### Anti-Features (Commonly Attempted, Creates Problems)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| One "ant worker" agent that handles all castes | Simpler to maintain | The Task tool routing depends on agent descriptions; a single generic agent can't describe itself specifically enough to be routed correctly; it also loses caste-specific constraints | One agent file per caste; routing is the point |
| Copy OpenCode agents verbatim with name change | Fast migration | OpenCode agents reference OpenCode-specific infrastructure (activity-log commands, spawn protocols with bash scripts); Claude Code agents use Task tool differently | Translate, don't copy; remove infrastructure dependencies, adapt spawn protocol |
| Include full workers.md content in every agent | Complete context | workers.md is 764 lines; injecting it into every agent definition consumes enormous tokens on every spawn and duplicates content; this is the known "4,200 token problem" | Agents should be self-contained; workers.md becomes developer reference only |
| Model field in frontmatter (`model: opus`) | Per-agent model selection | Claude Code does NOT support per-agent model routing via frontmatter; the field is ignored or may confuse the agent | Remove model references; routing happens at session level via environment variables |
| Require initialized colony for all agents | Consistency with OpenCode behavior | Claude Code users will invoke agents outside of formal colony sessions; hard failure on missing COLONY_STATE.json breaks standard use cases | Degrade gracefully: "if colony initialized, log activity; if not, continue without logging" |
| Generate ant names on every spawn | Immersive experience | Name generation requires `bash .aether/aether-utils.sh generate-ant-name`; adds a tool call dependency; Claude Code agents don't have consistent access to this script in all contexts | Names optional in Claude Code; Queen assigns name if colony initialized, otherwise agent self-identifies by caste |
| Emoji in agent identity lines (in body text) | Visual identity | Emoji in instruction text causes inconsistent behavior across models; formatting should be injected at spawn time, not embedded | Caste emoji stored in Queen's spawn protocol; not in agent definition body text |

---

## Feature Dependencies

```
aether-queen (orchestrator)
    └──spawns──> all other agents
    └──requires──> all other agent definitions exist (cannot route to undefined agents)

aether-builder
    └──pairs with──> aether-watcher (builder builds, watcher verifies)
    └──may spawn──> aether-watcher, aether-scout

aether-watcher
    └──reviews output of──> aether-builder
    └──creates flags for──> aether-queen to handle

aether-route-setter
    └──precedes──> aether-builder (plan before build)
    └──may spawn──> aether-scout (research before planning)

Surveyor variants (4 agents)
    └──all write to──> .aether/data/survey/ (shared output, separate files per variant)
    └──consumed by──> aether-route-setter, aether-builder, aether-queen

Read-only agents (chaos, archaeologist, auditor, gatekeeper, scout, sage, includer, tracker)
    └──NO dependency on each other
    └──report to──> parent who spawned them (queen or prime worker)

aether-keeper (Architect mode)
    └──synthesizes output from──> all agents
    └──documents into──> patterns/ and learnings/ directories

aether-chronicler
    └──reads from──> all agents' outputs, colony state, git history
    └──writes to──> docs/ only

aether-probe
    └──reads from──> aether-builder output (what code was written)
    └──writes to──> test files matching source structure

aether-weaver
    └──reads from──> aether-builder (code to refactor)
    └──requires no behavior change (verified by aether-watcher)
```

### Dependency Notes

- **aether-queen must be defined first** or in parallel — the queen's description tells Claude Code the orchestration role; all other agents are meaningless without an orchestrator
- **Builder and Watcher are paired** — they appear in the same workflow; their output formats must be compatible (watcher reads builder's JSON output)
- **Surveyor variants are independent of each other** — each writes a different document to `.aether/data/survey/`; they can be spawned in parallel and have no file conflicts
- **Read-only agents can be defined in any order** — they have no dependencies on each other; each is independent

---

## Standalone Subagent vs Slash Command

This is the critical architectural decision for this milestone.

### Recommendation: Subagent Definition for All 22 + No New Slash Commands

**Rationale:** Slash commands trigger agent definitions; they do not replace them. The 34 existing slash commands in `.claude/commands/ant/` orchestrate agents using the Task tool. Creating Claude Code agent definitions enables slash commands to route work to specialized agents rather than running everything inline.

The dividing line:
- **Subagent:** Things the Queen spawns with the Task tool during colony work
- **Slash command:** Things the user explicitly triggers (build, plan, init, seal, etc.)

All 22 ant worker types are things the Queen spawns. None of them are user-facing entry points. They should all be subagents, not slash commands.

**Exception:** The Queen herself exists both as a subagent (spawned by `/ant:build` and `/ant:init`) AND as the orchestrator. The Queen subagent definition enables the Task tool to spawn a Queen-level coordinator when deep orchestration is needed within a phase.

---

## MVP Definition

### Launch With (22 Subagent Definitions)

Minimum viable: one Claude Code agent file per ant worker type. The v1 bar is functional routing and correct tool access.

- [ ] `aether-queen.md` — Orchestrator; full XML; 6 workflow patterns; HIGH priority
- [ ] `aether-builder.md` — Implementation; full XML; TDD discipline; HIGH priority
- [ ] `aether-watcher.md` — Verification; full XML; 6-phase quality gate; HIGH priority
- [ ] `aether-scout.md` — Research; flat markdown; WebSearch/WebFetch enabled; MEDIUM
- [ ] `aether-route-setter.md` — Planning; flat markdown; write-enabled for plan files; MEDIUM
- [ ] `aether-surveyor-nest.md` — Architecture mapping; full XML; already exists, port; MEDIUM
- [ ] `aether-surveyor-disciplines.md` — Conventions; full XML; port from OpenCode; MEDIUM
- [ ] `aether-surveyor-pathogens.md` — Tech debt; full XML; port from OpenCode; MEDIUM
- [ ] `aether-surveyor-provisions.md` — Dependencies; full XML; port from OpenCode; MEDIUM
- [ ] `aether-keeper.md` — Knowledge; flat markdown; Architect mode; MEDIUM
- [ ] `aether-chronicler.md` — Documentation; flat markdown; LOW
- [ ] `aether-probe.md` — Test generation; flat markdown; LOW
- [ ] `aether-weaver.md` — Refactoring; flat markdown; LOW
- [ ] `aether-chaos.md` — Resilience testing; flat markdown; read-only; LOW
- [ ] `aether-archaeologist.md` — Git history; flat markdown; read-only; LOW
- [ ] `aether-ambassador.md` — API integration; flat markdown; LOW
- [ ] `aether-auditor.md` — Code review; flat markdown; read-only; LOW
- [ ] `aether-gatekeeper.md` — Dependencies; flat markdown; read-only; LOW
- [ ] `aether-measurer.md` — Performance; flat markdown; LOW
- [ ] `aether-includer.md` — Accessibility; flat markdown; read-only; LOW
- [ ] `aether-sage.md` — Analytics; flat markdown; read-only; LOW
- [ ] `aether-tracker.md` — Bug investigation; flat markdown; read-only; LOW

### Add After Validation (v1.x)

- [ ] **Behavioral smoke tests** — spawn each agent via Task tool with a simple representative prompt; verify output format matches expected schema
- [ ] **Sync validation** — extend `npm run lint:sync` to verify Claude Code agents exist for every OpenCode agent (and vice versa)
- [ ] **Description quality review** — test each agent's description triggers correct routing when given to a Queen with all 22 agent descriptions; adjust ambiguous descriptions

### Future Consideration (v2+)

- [ ] **Per-agent context injection** — Queen injects colony-specific context (goal, constraints, pheromones) into agent prompts at spawn time rather than relying on agents to self-load
- [ ] **Caste-specific tool access profiles** — formalize which tools each caste class should have (orchestrators, implementors, researchers, auditors) and validate against a profile matrix
- [ ] **Cross-agent shared memory via phase scratch pad** — Claude Code agents reading/writing `.aether/data/phase-scratch.json` during parallel execution

---

## Feature Prioritization Matrix

| Agent | User Value | Implementation Cost | Priority |
|-------|------------|---------------------|----------|
| aether-queen | HIGH — orchestration without this means no routing | MEDIUM — XML needed; complex patterns | P1 |
| aether-builder | HIGH — most spawned agent in any build | HIGH — XML; full TDD + deviation rules | P1 |
| aether-watcher | HIGH — quality gate; every build needs it | HIGH — XML; execution verification protocol | P1 |
| aether-scout | HIGH — research is frequent in SPBV pattern | LOW — flat markdown; straightforward port | P1 |
| aether-route-setter | HIGH — planning before every implementation | MEDIUM — flat markdown; write access needed | P1 |
| Surveyor variants (4) | HIGH — colony relies on them for codebase context | MEDIUM — XML; already written in OpenCode | P1 |
| aether-keeper | MEDIUM — architect mode needed for synthesis | LOW — flat markdown; port with mode trigger | P2 |
| aether-tracker | MEDIUM — bug investigation pattern | LOW — flat markdown; read-only port | P2 |
| aether-probe | MEDIUM — test coverage analysis | LOW — flat markdown; write to tests/ only | P2 |
| aether-weaver | MEDIUM — refactor pattern | LOW — flat markdown; write-enabled port | P2 |
| aether-auditor | MEDIUM — compliance and security patterns | LOW — flat markdown; read-only; Guardian mode | P2 |
| aether-chaos | LOW — resilience testing, less frequent | LOW — flat markdown; read-only; simple | P3 |
| aether-archaeologist | LOW — git archaeology, niche use | LOW — flat markdown; read-only; simple | P3 |
| aether-ambassador | LOW — API integration, project-specific | LOW — flat markdown; web fetch enabled | P3 |
| aether-chronicler | LOW — docs sprint pattern | LOW — flat markdown; write to docs/ | P3 |
| aether-gatekeeper | LOW — dependency scans, infrequent | LOW — flat markdown; read-only; Bash for audit | P3 |
| aether-measurer | LOW — performance, specialized | LOW — flat markdown; Bash for profiling | P3 |
| aether-includer | LOW — accessibility, specialized | LOW — flat markdown; read-only | P3 |
| aether-sage | LOW — analytics, rarely needed | LOW — flat markdown; read-only | P3 |

**Priority key:**
- P1: Must have for the milestone to function (Queen + core workflow agents)
- P2: Should have, needed for all 6 workflow patterns to operate
- P3: Nice to have, needed for specialized compliance/research patterns

---

## Tool Access Matrix

This determines what each agent can do, and must be set correctly in frontmatter.

| Agent Class | Read | Grep | Glob | Bash | Write | Edit | WebSearch | WebFetch |
|-------------|------|------|------|------|-------|------|-----------|----------|
| Orchestrators (Queen, Route-Setter) | YES | YES | YES | YES | YES | YES | NO | NO |
| Implementors (Builder, Weaver, Probe) | YES | YES | YES | YES | YES | YES | NO | NO |
| Verifiers (Watcher, Chaos, Auditor) | YES | YES | YES | YES | NO | NO | NO | NO |
| Researchers (Scout, Ambassador) | YES | YES | YES | YES | NO | NO | YES | YES |
| Historians (Archaeologist, Sage, Tracker) | YES | YES | YES | YES | NO | NO | NO | NO |
| Knowledge writers (Keeper, Chronicler) | YES | YES | YES | YES | YES | YES | NO | NO |
| Surveyors | YES | YES | YES | YES | YES | NO | NO | NO |
| Dependency agents (Gatekeeper) | YES | YES | YES | YES | NO | NO | NO | NO |
| Performance agents (Measurer) | YES | YES | YES | YES | YES | NO | NO | NO |
| Accessibility agents (Includer) | YES | YES | YES | YES | NO | NO | NO | NO |

**Notes:**
- Chaos and Auditor have Bash (for running tests/analysis commands) but not Write/Edit — they investigate, not modify
- Ambassador needs WebSearch and WebFetch for third-party API documentation
- Measurer has Write for writing profiling reports (not source code)
- All Bash access is limited by read_only boundary declarations in agent body text

---

## Description Writing Guide (for Routing Effectiveness)

The description field determines how well the Task tool routes to this agent. Based on patterns from GSD/CDS agents:

### What Works

**Pattern 1 — Role + trigger + spawner (GSD pattern):**
```
"Use this agent for code implementation, file creation, command execution, and build tasks. The builder turns plans into working code."
```

**Pattern 2 — Role + explicit use case (community pattern):**
```
"Software architecture specialist for system design, scalability, and technical decision-making. Use PROACTIVELY when planning new features, refactoring large systems, or making architectural decisions."
```

**Pattern 3 — Role + spawner + output (Aether-specific):**
```
"Use this agent for validation, testing, quality assurance, and monitoring. The watcher ensures quality and guards the colony against regressions."
```

### What Fails

- Generic descriptions: `"An AI assistant for code tasks"` — no routing signal
- Overlapping descriptions: If Builder says "code and testing" and Watcher says "testing and code", the Task tool may route incorrectly
- Missing use trigger: `"Expert in Python"` — doesn't tell the orchestrator WHEN to use this agent
- Long descriptions with multiple responsibilities: If an agent does "X, Y, Z, A, B, and C", the Task tool may not match it to any specific request

### Aether-Specific Routing Requirement

Queen's spawn protocol references agents by their caste name. The agent description must include the caste name OR be matched by the phrasing the Queen uses when spawning. Current Queen spawn phrasing from `.opencode/agents/aether-queen.md`:

- `"aether-builder"` → `Builder` description should include "implementation", "code", "build"
- `"aether-watcher"` → `Watcher` description should include "validation", "testing", "quality"
- `"aether-scout"` → `Scout` description should include "research", "information gathering", "documentation"
- `"aether-chaos"` → `Chaos` description should include "resilience testing", "edge cases", "boundary conditions"

---

## Competitor Feature Analysis

| Feature | GSD Agents (Aether's GSD system) | everything-claude-code | CDS Agents | Our Approach |
|---------|----------------------------------|------------------------|------------|--------------|
| XML structure | Yes (executor, planner, verifier) | No (flat markdown) | Yes (same as GSD) | XML for complex agents, flat for simple |
| Explicit tool lists | Yes, all agents | Yes, architect uses `["Read", "Grep", "Glob"]` | Yes, same as GSD | Yes for all 22 agents |
| Failure modes | Yes, tiered severity | Minimal (some have error sections) | Yes, same as GSD | Yes for all 22 agents |
| Success criteria | Yes, explicit checklist | No | Yes | Yes for all 22 agents |
| Read-only declarations | Yes (verifier) | Partial (not all agents) | Yes | Yes, as first constraint for read-only agents |
| Spawn trigger description | Yes ("Spawned by execute-phase") | No | Yes | Yes, all descriptions include spawn context |
| Few-shot output examples | Yes (executor) | No | Yes | For agents with complex output (Builder, Watcher) |
| Activity logging | No (GSD is workflow-focused) | No | No | Optional; degrade gracefully if colony not initialized |
| Model field | No | Yes (`model: opus`) but likely ignored | No | No; remove from all definitions |

---

## Sources

- `/Users/callumcowie/repos/Aether/.opencode/agents/*.md` — all 22 existing OpenCode agent definitions (direct inspection)
- `/Users/callumcowie/repos/Aether/.claude/agents/gsd-executor.md` — GSD executor pattern (full XML, deviation rules, checkpoint protocol)
- `/Users/callumcowie/repos/Aether/.claude/agents/gsd-planner.md` — GSD planner pattern (full XML, task breakdown, dependency graph)
- `/Users/callumcowie/repos/Aether/.claude/agents/gsd-verifier.md` — GSD verifier pattern (goal-backward verification, artifact checks)
- `/Users/callumcowie/repos/Aether/.claude/agents/gsd-*.md` — 8 additional GSD agent files (codebase mapper, debugger, integration checker, plan checker, project researcher, phase researcher, research synthesizer, roadmapper)
- `/Users/callumcowie/repos/everything-claude-code/agents/*.md` — 12 community agents (architect, build-error-resolver, code-reviewer, database-reviewer, doc-updater, e2e-runner, go-build-resolver, go-reviewer, planner, refactor-cleaner, security-reviewer, tdd-guide)
- `/Users/callumcowie/repos/cosmic-dev-system/agents/*.md` — 11 CDS agents (near-identical to GSD; confirms patterns)
- `/Users/callumcowie/repos/Aether/.aether/workers.md` — worker discipline reference; caste definitions; spawn protocol
- `/Users/callumcowie/repos/Aether/CLAUDE.md` — Aether architecture; protected paths; caste system reference

---

*Feature research for: Claude Code subagents for Aether ant worker types*
*Researched: 2026-02-20*
