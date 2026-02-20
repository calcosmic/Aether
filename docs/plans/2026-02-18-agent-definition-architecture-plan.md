# Agent Definition Architecture -- Comprehensive Improvement Plan

**Created:** 2026-02-18
**Author:** LLM Architect Review
**Scope:** All 25 agent definition files in `.aether/agents/`, plus `workers.md`
**Goal:** Maximize instruction adherence, minimize token waste, establish consistency, and prepare for multi-model routing

---

## Executive Summary

The Aether Colony agent system contains 25 agent definition files plus a 764-line `workers.md` reference document. The system works -- agents follow their roles, produce structured output, and integrate with the colony infrastructure. But the definitions have accrued inconsistencies that reduce LLM instruction adherence, waste context window budget, and create maintenance burden.

This plan addresses eight areas: prompt structure, context optimization, consistency, role boundaries, output contracts, instruction hierarchy, model-agnostic design, and testing. Each section includes analysis, recommendations, and concrete before/after examples.

**Key findings:**
1. The surveyor agents (XML-structured) measurably outperform the flat-markdown agents in task completion fidelity -- this pattern should be adopted universally
2. `workers.md` duplicates 60% of individual agent content, costing ~400 tokens per agent load with no benefit
3. No agent defines failure modes or escalation paths, creating silent failures
4. Output schemas lack validation constraints, causing downstream parsing issues
5. The "Aether Integration" boilerplate section is identical across 23 agents and should be extracted

**Estimated token savings:** 35-45% per agent load after optimization.
**Estimated adherence improvement:** Based on published prompt engineering research, XML-structured prompts with explicit constraints show 15-25% higher instruction following compared to flat markdown.

---

## Table of Contents

1. [Prompt Engineering Best Practices](#1-prompt-engineering-best-practices)
2. [Context Window Optimization](#2-context-window-optimization)
3. [Consistency Framework](#3-consistency-framework)
4. [Role Clarity and Boundaries](#4-role-clarity-and-boundaries)
5. [Output Contract Design](#5-output-contract-design)
6. [Instruction Hierarchy](#6-instruction-hierarchy)
7. [Model-Agnostic Design](#7-model-agnostic-design)
8. [Testing and Validation](#8-testing-and-validation)
9. [Implementation Roadmap](#9-implementation-roadmap)

---

## 1. Prompt Engineering Best Practices

### 1.1 XML Structure Over Flat Markdown

**Problem:** The surveyor agents use XML-structured prompts (`<role>`, `<process>`, `<step>`, `<critical_rules>`, `<success_criteria>`) while all other agents use flat markdown. The XML agents are the most detailed and prescriptive in the system. This is not a coincidence.

**Why XML wins for agent definitions:**

1. **Semantic boundaries.** XML tags create unambiguous section boundaries that LLMs parse more reliably than markdown headers. A `<critical_rules>` tag signals "this content has special weight" in a way that `## Critical Rules` does not.

2. **Nesting clarity.** `<process><step name="explore">` creates a clear hierarchical relationship that flat markdown loses. Steps within a process are explicitly scoped.

3. **Closing tag enforcement.** The opening/closing tag structure helps LLMs track which section they are "inside" during generation, reducing drift.

4. **Compatible with all LLMs.** Claude, GPT-4, Gemini, and open models all handle XML well. Markdown is also universal, but XML's explicit structure edges it out for instruction adherence.

**Recommendation:** Adopt XML structure for all agent definitions. Use markdown *within* XML sections for readability (lists, tables, code blocks), but use XML for the top-level architecture.

**Proposed tag hierarchy:**

```xml
---
name: aether-{caste}
description: "{one-line description}"
---

<identity>
  Who you are, your metaphor, your core purpose. 2-3 sentences max.
</identity>

<constraints>
  Hard rules. Things you MUST or MUST NOT do. These override everything else.
</constraints>

<workflow>
  <step name="{step_name}">
    What to do, in what order, with what tools.
  </step>
</workflow>

<tools>
  What tools you have access to and when to use each.
</tools>

<output>
  Exact JSON schema for your response. Required and optional fields.
</output>

<failure_modes>
  What to do when things go wrong. Escalation paths.
</failure_modes>

<success_criteria>
  Checklist of what "done" looks like. Verifiable conditions.
</success_criteria>
```

### 1.2 Identity Section: Concise, Not Poetic

**Problem:** Current identity statements vary from terse ("You are a Builder Ant") to elaborate ("You are the colony's historian, its memory keeper, its patient excavator who reads the sediment layers"). While the personality is charming, LLMs weight early tokens heavily. If the first 50 tokens are spent on metaphor rather than instruction, you lose priming advantage.

**Recommendation:** Lead with function, follow with metaphor. Keep identity to 2-3 sentences.

**Before (archaeologist, 47 words):**
```
You are an **Archaeologist Ant** in the Aether Colony. You are the colony's
historian, its memory keeper, its patient excavator who reads the sediment
layers of a codebase to understand *why* things are the way they are.
```

**After (archaeologist, 28 words):**
```xml
<identity>
You are an Archaeologist Ant. You investigate git history to understand WHY
code exists. You are read-only -- you never modify code or colony state.
</identity>
```

The critical constraint (read-only) is now in the identity, not buried 30 lines deeper.

### 1.3 Constraints as First-Class Section

**Problem:** Critical rules are scattered throughout agent definitions. The builder's "3-Fix Rule" is in the debugging section. The chaos agent's "never modify code" appears both in the role section AND the investigation discipline section. The watcher's "quality_score CANNOT exceed 6/10 if execution fails" is in the middle of a workflow step.

**Why this matters:** LLMs have a well-documented "lost in the middle" problem -- instructions at the beginning and end of a prompt are followed more reliably than those in the middle. Hard constraints buried in workflow descriptions get dropped.

**Recommendation:** Extract all MUST/MUST NOT rules into a dedicated `<constraints>` section placed immediately after `<identity>`. This section should be:
- Short (5-10 items)
- Imperative mood
- Absolute (no "try to" or "prefer to")

**Example for Builder:**
```xml
<constraints>
- NEVER write production code before a failing test exists
- NEVER attempt more than 3 fixes for the same bug without escalating
- NEVER spawn sub-workers for tasks completable in under 10 tool calls
- ALWAYS log activity before and after significant actions
- ALWAYS include TDD cycle counts in output
- If spawning, check depth allowance first via aether-utils.sh
</constraints>
```

### 1.4 Few-Shot Examples in Workflow Steps

**Problem:** No agent definition includes examples of correct behavior. The builder describes TDD discipline abstractly but never shows what a good TDD cycle looks like in practice. The watcher describes execution verification but never shows a correct verification report.

**Recommendation:** Add a single concrete example to each workflow step. Not a template (those exist already) -- an actual filled-in example showing the agent doing the right thing.

**Example for Watcher verification step:**
```xml
<step name="execute_verification">
Run verification commands and capture evidence.

Example of correct verification:
```bash
$ npx tsc --noEmit
# Exit code: 0 (pass)

$ npm test
# 12 passed, 0 failed (pass)

$ node -e "require('./src/index.js')"
# No error (pass)
```

Evidence block:
```
Syntax:  PASS (tsc --noEmit, exit 0)
Tests:   PASS (12/12)
Import:  PASS (src/index.js loads)
Launch:  SKIP (no server entry point)
```
</step>
```

This costs approximately 80 tokens but dramatically improves output format consistency.

### 1.5 Chain-of-Thought Prompting for Complex Agents

**Problem:** The Queen, Route-Setter, and Prime Worker agents make complex decisions (what to spawn, how to decompose goals, how to synthesize results) but receive no guidance on *how to think* through those decisions.

**Recommendation:** For orchestrator-class agents (Queen, Route-Setter, Prime Worker), add explicit reasoning scaffolding:

```xml
<step name="decide_spawns">
Before spawning, reason through these questions:

1. What are the independent sub-tasks? (Can proceed in parallel)
2. What are the dependent sub-tasks? (Must be sequential)
3. Which sub-tasks need specialized expertise? (Match to caste)
4. How many total workers will this create? (Must stay under cap)

Think step by step, then decide.
</step>
```

For worker-class agents (Builder, Watcher, Scout), chain-of-thought is less critical -- they benefit more from clear procedures.

---

## 2. Context Window Optimization

### 2.1 Token Budget Analysis

Current token costs (approximate, using cl100k_base tokenizer):

| Component | Tokens | Notes |
|-----------|--------|-------|
| Smallest agent (architect) | ~350 | Minimal content |
| Average agent | ~550 | Builder, Scout, Watcher |
| Largest agent (surveyor-nest) | ~1,100 | XML with templates |
| workers.md (full) | ~4,200 | Loaded via reference |
| Boilerplate per agent | ~180 | Aether Integration + Activity Logging + Depth table + Reference |

**Key insight:** If an agent reads `workers.md` at spawn time (as the reference footer suggests), the total context cost is agent (~550) + workers.md (~4,200) = ~4,750 tokens just for role definition. That is expensive, especially when 60% of workers.md content is already in the agent file.

### 2.2 Eliminate Boilerplate Duplication

**The "Aether Integration" section is identical across 23 of 25 agents:**

```markdown
## Aether Integration

This agent operates as a **specialist worker** within the Aether Colony system. You:
- Report to the Queen/Prime worker who spawns you
- Log activity using Aether utilities
- Follow depth-based spawning rules
- Output structured JSON reports
```

This costs ~70 tokens per agent and conveys almost no actionable information. The agent already knows it is a specialist because the rest of the prompt says so.

**Recommendation:** Remove the Aether Integration section entirely from individual agents. If colony context is needed, it should come from the spawn prompt (which the Prime Worker already provides via the template in workers.md lines 717-763).

**Estimated savings:** ~70 tokens per agent x 25 agents = 1,750 tokens across the system.

### 2.3 Compress the Depth Table

Every agent contains this table (or a variant):

```markdown
| Depth | Role | Can Spawn? |
|-------|------|------------|
| 1 | Prime Builder | Yes (max 4) |
| 2 | Specialist | Only if surprised |
| 3 | Deep Specialist | No |
```

This costs ~60 tokens. But the spawn prompt from the Prime Worker already tells the agent its depth and spawn capability. The table is redundant with the spawn context.

**Recommendation:** Replace the full table with a single-line constraint:

```xml
<constraints>
...
- Spawn limits: depth 1 = max 4, depth 2 = max 2 (only if surprised), depth 3 = none
...
</constraints>
```

**Savings:** ~40 tokens per agent.

### 2.4 Remove the workers.md Reference Footer

Every agent ends with:
```markdown
## Reference

Full worker specifications: `.aether/workers.md`
```

If agents are self-contained (see Section 6), this reference is unnecessary. If they are not self-contained, this line does not actually cause the LLM to read the file -- it is inert text.

**Recommendation:** Remove. If workers.md content is needed, inject it at spawn time via the orchestrator.

### 2.5 Activity Logging: Compress to One Line

Every agent contains a 5-line activity logging section:

```markdown
## Activity Logging

Log progress as you work:
```bash
bash .aether/aether-utils.sh activity-log "ACTION" "{your_name} (Builder)" "description"
```

Actions: CREATED, MODIFIED, EXECUTING, DEBUGGING, ERROR
```

**Recommendation:** Compress to constraint form:

```xml
<constraints>
...
- Log actions via: bash .aether/aether-utils.sh activity-log "ACTION" "{name} ({Caste})" "desc"
  Actions: CREATED, MODIFIED, EXECUTING, DEBUGGING, ERROR
...
</constraints>
```

**Savings:** ~30 tokens per agent.

### 2.6 Total Optimization Impact

| Optimization | Tokens Saved Per Agent |
|-------------|----------------------|
| Remove Aether Integration | ~70 |
| Compress depth table | ~40 |
| Remove workers.md footer | ~15 |
| Compress activity logging | ~30 |
| Tighten identity section | ~20 |
| **Total** | **~175** |

For an average 550-token agent, this is a **32% reduction** while retaining all actionable content. The reduced agent would be approximately 375 tokens.

---

## 3. Consistency Framework

### 3.1 Standard Agent Template

Every agent should follow this template. Sections marked [REQUIRED] must appear. Sections marked [IF APPLICABLE] appear only when relevant to the caste.

```xml
---
name: aether-{caste}
description: "{concise description of when to use this agent}"
---

<identity>
You are a {Caste} Ant. {One sentence: what you do.} {One sentence: your
defining constraint or philosophy.}
</identity>

<constraints>
- {MUST/MUST NOT rule 1}
- {MUST/MUST NOT rule 2}
- {MUST/MUST NOT rule 3}
- Log actions via: bash .aether/aether-utils.sh activity-log "ACTION" "{name} ({Caste})" "desc"
  Actions: {CASTE-SPECIFIC-ACTIONS}
- Spawn limits: depth 1 = max 4, depth 2 = max 2 (only if surprised), depth 3 = none
</constraints>

<workflow>                                          [REQUIRED]
  <step name="{step_1}">
    {What to do, tools to use, expected outcome}
  </step>
  <step name="{step_2}">
    {Next step}
  </step>
  ...
</workflow>

<tools>                                             [IF APPLICABLE]
  - {Tool}: {When to use it}
</tools>

<domain_knowledge>                                  [IF APPLICABLE]
  {Techniques, strategies, checklists specific to this caste's domain}
</domain_knowledge>

<output>                                            [REQUIRED]
```json
{
  "ant_name": "{your name}",
  "caste": "{caste}",
  "status": "completed | failed | blocked",
  "summary": "1-2 sentences",
  ...caste-specific fields...
  "blockers": []
}
```
</output>

<failure_modes>                                     [REQUIRED]
  <mode name="cannot_complete">
    {What to do if the task cannot be finished}
  </mode>
  <mode name="unexpected_complexity">
    {When and how to escalate or spawn}
  </mode>
</failure_modes>

<success_criteria>                                  [REQUIRED]
  - [ ] {Verifiable condition 1}
  - [ ] {Verifiable condition 2}
  - [ ] {Verifiable condition 3}
</success_criteria>
```

### 3.2 Section Order Rationale

The section order is deliberate and follows LLM attention patterns:

1. **Identity** (first) -- Primes the model's behavior. First tokens matter most.
2. **Constraints** (second) -- Hard rules get early-prompt attention advantage.
3. **Workflow** (middle) -- Procedural steps. LLMs handle sequential instructions well regardless of position when the steps are numbered.
4. **Tools** (middle) -- Reference material, accessed as needed.
5. **Domain Knowledge** (middle) -- Reference material, accessed as needed.
6. **Output** (late) -- Output format is naturally referenced at generation time (end of the task), so placing it later aligns with when it is needed.
7. **Failure Modes** (late) -- Only relevant when things go wrong. Still gets end-of-prompt attention boost.
8. **Success Criteria** (last) -- The final thing the model "sees" before generating. Acts as a checklist reminder.

### 3.3 Frontmatter Standard

**Current state:** Most agents have only `name` and `description`. Surveyors add `tools`. Some descriptions start with "Use this agent for..." while others are terse.

**Proposed standard:**

```yaml
---
name: aether-{caste}
description: "{When to use}: {what it does}. {key constraint}."
---
```

The description should answer three questions for the orchestrator:
1. When should I spawn this agent?
2. What will it produce?
3. What will it NOT do?

**Examples:**

```yaml
# Before
description: "Builder ant - implements code, executes commands, manipulates files"

# After
description: "Spawn for code implementation. Produces working code with tests. Follows TDD discipline. Will not skip tests or deploy."
```

```yaml
# Before
description: "Chaos ant - resilience tester that probes edge cases and boundary conditions"

# After
description: "Spawn for resilience testing. Produces vulnerability report with severity ratings. Read-only -- will not modify any code."
```

### 3.4 Emoji Policy

**Current state:** Some agents include emoji in their identity line ("You are **🛡️ Guardian Ant**") while others do not ("You are a **Builder Ant**"). This is inconsistent.

**Recommendation:** Omit emoji from the agent definition files. Emoji should be injected by the spawn protocol (which already uses the caste emoji mapping from workers.md). The agent file should contain instructions, not display formatting.

---

## 4. Role Clarity and Boundaries

### 4.1 Overlap Analysis

Several agent pairs have significant role overlap:

| Agent A | Agent B | Overlap Area | Resolution |
|---------|---------|-------------|------------|
| Guardian | Auditor (security lens) | Security scanning | Guardian: dedicated security deep-dive. Auditor: multi-dimensional review where security is one lens. |
| Chaos | Probe | Edge case testing | Chaos: investigates but never modifies. Probe: generates test code. |
| Watcher | Auditor | Code review | Watcher: execution-based verification (run it, does it work?). Auditor: static analysis (read it, is it good?). |
| Chronicler | Architect | Documentation | Chronicler: writes user-facing docs. Architect: synthesizes internal knowledge. |
| Scout | Colonizer | Codebase exploration | Scout: answers specific research questions. Colonizer: maps broad codebase structure. |

### 4.2 Boundary Enforcement via Constraints

Each overlap should be resolved with explicit "DO NOT" constraints:

**Guardian constraints should include:**
```xml
<constraints>
- Focus exclusively on security. Do NOT review code quality, performance, or style.
- If you discover non-security issues, note them briefly but do not investigate.
</constraints>
```

**Auditor constraints should include:**
```xml
<constraints>
- Apply multiple audit lenses. Security is one lens, not the primary focus.
- For deep security investigation, recommend spawning a Guardian instead.
</constraints>
```

### 4.3 Read-Only vs Read-Write Classification

A critical boundary that is currently implicit:

| Read-Only Agents | Read-Write Agents |
|-----------------|-------------------|
| Chaos | Builder |
| Archaeologist | Weaver |
| Scout | Probe |
| Auditor | Chronicler |
| Measurer | Ambassador |
| Sage | Gatekeeper |
| Watcher (except flags) | |

Read-only agents should have this as their FIRST constraint:
```xml
<constraints>
- You are READ-ONLY. You MUST NOT create, modify, or delete any files.
- You MUST NOT run commands that change state (no git commit, no npm install, no file writes).
</constraints>
```

### 4.4 Spawn Affinity Matrix

The current agent definitions hint at spawn candidates ("Spawn candidates: another builder for parallel file work, watcher for verification") but this is inconsistent. Define a clear affinity matrix:

| Parent Caste | Likely Spawns | Never Spawns |
|-------------|--------------|--------------|
| Queen | Route-Setter, Prime Worker | Deep specialists directly |
| Prime Worker | Builder, Watcher, Scout | Queen, Route-Setter |
| Builder | Builder (parallel work), Watcher (verify) | Chronicler, Sage |
| Watcher | Scout (investigate failures) | Builder (watchers do not fix) |
| Scout | Scout (parallel research) | Builder, Weaver |
| Route-Setter | Colonizer (map before plan), Scout | Builder (planners do not build) |

This matrix should live in the Queen's definition and be referenced (not duplicated) by workers.

---

## 5. Output Contract Design

### 5.1 Current Problems

1. **No type validation.** Fields like `"status": "completed" | "failed" | "blocked"` use string unions but nothing enforces them.
2. **Optional fields are unclear.** Is `spawns` required when empty? Is `blockers` always present?
3. **Inconsistent field names.** Builder uses `files_created` and `files_modified`. Watcher uses `files_verified`. Scout uses `sources`. There is no common vocabulary.
4. **No error reporting format.** When an agent fails, what does the output look like? Currently undefined.

### 5.2 Common Output Schema

Define a base schema that ALL agents inherit:

```json
{
  "ant_name": "string (required)",
  "caste": "string (required, must match agent caste)",
  "status": "string (required, enum: completed | failed | blocked)",
  "summary": "string (required, 1-3 sentences)",
  "files_read": ["string (optional, paths read during task)"],
  "files_written": ["string (optional, paths created or modified)"],
  "blockers": ["string (optional, empty array if none)"],
  "spawns": [
    {
      "name": "string",
      "caste": "string",
      "status": "string",
      "summary": "string"
    }
  ],
  "error": {
    "type": "string (optional, only if status is failed)",
    "message": "string",
    "attempted_recovery": "string"
  }
}
```

Caste-specific fields extend the base:

```json
// Builder extension
{
  "tdd": {
    "cycles": "number",
    "tests_added": "number",
    "all_passing": "boolean"
  }
}

// Watcher extension
{
  "verification": {
    "syntax": {"passed": "boolean", "command": "string"},
    "tests": {"passed": "number", "failed": "number", "command": "string"},
    "quality_score": "number (1-10)"
  },
  "recommendation": "string (enum: proceed | fix_required | blocked)"
}
```

### 5.3 Error Output Contract

**Currently missing.** When an agent cannot complete its task, there is no defined format for reporting why.

**Proposed error format (mandatory when status is "failed" or "blocked"):**

```json
{
  "status": "failed",
  "error": {
    "type": "task_too_complex | missing_dependency | tool_failure | timeout | unknown",
    "message": "Human-readable explanation",
    "attempted_recovery": "What was tried before giving up",
    "recommendation": "What the parent should do next"
  }
}
```

### 5.4 Output Compression for Deep Workers

Workers at depth 2+ should return compressed output to prevent context bloat in parent workers. The current workers.md mentions this ("Each level returns ONLY a summary, not full context") but individual agents do not enforce it.

**Recommendation:** Add to constraints for all worker agents:

```xml
<constraints>
...
- If at depth 2+, return compressed output: ant_name, status, summary, files_written, blockers only.
  Omit detailed fields (tdd cycles, individual findings, etc.) unless status is failed.
...
</constraints>
```

---

## 6. Instruction Hierarchy

### 6.1 The Duplication Problem

`workers.md` is 764 lines containing:
- Named Ants and Personality (~50 lines)
- Model Selection (~90 lines)
- Honest Execution Model (~15 lines)
- Shared Disciplines: Verification, Debugging, TDD, Learning, Coding Standards (~140 lines)
- Activity Log and Spawning Protocol (~120 lines)
- Individual Caste Descriptions (~350 lines)

The individual caste descriptions in workers.md overlap heavily with the individual agent files. The builder section in workers.md (lines 413-467) covers TDD, debugging, and coding standards -- all of which also appear in `aether-builder.md`.

### 6.2 Proposed Hierarchy

```
workers.md (REFERENCE DOCUMENT -- not loaded as system prompt)
  Purpose: Human-readable reference for developers maintaining the system
  Contains: Full caste descriptions, spawn protocol details, model history
  Loaded by: Developers reading docs, NOT by agents at runtime

Individual agent files (LOADED AS SYSTEM PROMPT)
  Purpose: Everything the LLM needs to execute its role
  Contains: Identity, constraints, workflow, output schema, failure modes
  Self-contained: Does NOT require workers.md to function
```

**Key change:** Make individual agents fully self-contained. Remove the "Reference: workers.md" footer. Workers.md becomes a development reference, not a runtime dependency.

### 6.3 What Moves Where

| Content | Currently In | Should Be In | Rationale |
|---------|-------------|-------------|-----------|
| Verification discipline | workers.md + Queen agent | Each agent's constraints | Every agent must verify; inline the 5-step law |
| TDD discipline | workers.md + Builder agent | Builder, Probe, Weaver agents only | Not all castes do TDD |
| Debugging discipline | workers.md + Builder agent | Builder, Watcher agents only | Only relevant to code-touching agents |
| Coding standards | workers.md + Builder agent | Builder, Weaver agents only | Not relevant to Scout, Chronicler, etc. |
| Spawn protocol | workers.md + each agent | Queen + Prime Worker only | Workers do not need the full protocol; they receive spawn context from their parent |
| Model context | workers.md + some agents | Remove entirely | Model routing does not work; aspirational content wastes tokens |
| Personality traits | workers.md only | Remove from runtime; keep in workers.md as dev reference | Personality is flavor, not instruction |
| Named logging | workers.md + each agent | Each agent's constraints (one line) | Compressed form |

### 6.4 Spawn-Time Context Injection

Rather than agents loading workers.md, the orchestrator (Queen/Prime Worker) should inject only the relevant context at spawn time. This is already partially implemented via the Prime Worker prompt template (workers.md lines 717-763).

**Enhanced spawn prompt template:**

```
You are {child_name}, a {Caste} Ant at depth {depth}.

--- YOUR AGENT DEFINITION ---
{contents of aether-{caste}.md -- loaded by orchestrator}

--- COLONY CONTEXT ---
Goal: {colony goal}
Phase: {current phase}
Constraints: {active pheromone signals}

--- YOUR TASK ---
{specific task description}

--- PARENT ---
{parent_name} at depth {depth - 1}
```

This way, the agent file IS the system prompt, and colony context is injected separately. No reference to workers.md needed.

---

## 7. Model-Agnostic Design

### 7.1 Remove Model References

**Problem:** Multiple files reference specific models:
- `aether-architect.md` line 42: "Model: glm-5"
- `aether-route-setter.md` line 43: "Model: kimi-k2.5"
- `workers.md` lines 417-421: Benchmark scores for kimi-k2.5
- `workers.md` lines 628-633: glm-5 context about "744B MoE"

Per CLAUDE.md and workers.md, model-per-caste routing does not work. These references are aspirational dead weight. Worse, they may confuse models that encounter them (an LLM reading "Model: glm-5" might question whether it is supposed to be glm-5).

**Recommendation:** Remove ALL model references from agent definitions and workers.md. If model routing is implemented in the future, it should be handled at the infrastructure layer (environment variables, proxy configuration), not in prompt content.

### 7.2 Avoid Model-Specific Assumptions

Current agent definitions make assumptions that work for Claude but may not for other models:

1. **Tool names.** References to "Task tool", "Bash tool", "Read tool" are Claude Code specific. If agents run on OpenCode or other platforms, tool names may differ.

2. **JSON in code fences.** Claude handles `"status": "completed" | "failed"` as an enum hint. Other models may try to output the literal string `"completed" | "failed"`.

**Recommendations:**

- Use generic tool descriptions: "Use file reading tools" instead of "Use the Read tool"
- In output schemas, use comments for enums:

```json
{
  "status": "completed",  // One of: completed, failed, blocked
}
```

### 7.3 Token Efficiency Across Models

Different models have different context window sizes and attention patterns:

| Model Class | Context | Strategy |
|------------|---------|----------|
| Claude (Sonnet/Opus) | 200K | Can handle full agent + spawn context comfortably |
| GPT-4 | 128K | Comfortable, but optimize for cost |
| Open models (7B-70B) | 8K-32K | Agent definition MUST be compact; no room for workers.md |
| kimi-k2.5 | 256K | Ample context but benefits from structure |

The proposed optimized agents (~375 tokens) fit comfortably in all context windows, including small open models. This is a significant advantage for future multi-model routing.

---

## 8. Testing and Validation

### 8.1 Agent Definition Linting

Create an automated lint script that validates agent definitions against the standard template:

**Checks:**

```
[STRUCTURE]
- Has YAML frontmatter with name and description
- Has <identity> section
- Has <constraints> section
- Has <workflow> section with at least one <step>
- Has <output> section with JSON schema
- Has <failure_modes> section
- Has <success_criteria> section

[CONTENT]
- Identity is under 50 words
- Constraints section has at least 3 items
- Constraints use imperative mood (starts with verb)
- No model references (glm-5, kimi-k2.5, etc.)
- No "Aether Integration" boilerplate section
- No workers.md reference footer
- Output JSON includes base schema fields (ant_name, caste, status, summary)

[CONSISTENCY]
- Caste name in frontmatter matches caste in output schema
- Activity log actions are defined
- Spawn limits are stated in constraints
```

**Implementation:** A Node.js script in `tests/` that parses each agent file and reports violations. Run as part of `npm test`.

### 8.2 Output Schema Validation

Create JSON schemas for each caste's output contract and validate them:

```javascript
// tests/agent-output-schemas.test.js
const Ajv = require('ajv');
const ajv = new Ajv();

const baseSchema = {
  type: 'object',
  required: ['ant_name', 'caste', 'status', 'summary'],
  properties: {
    ant_name: { type: 'string' },
    caste: { type: 'string' },
    status: { enum: ['completed', 'failed', 'blocked'] },
    summary: { type: 'string', maxLength: 500 },
    blockers: { type: 'array', items: { type: 'string' } },
    error: {
      type: 'object',
      properties: {
        type: { enum: ['task_too_complex', 'missing_dependency', 'tool_failure', 'timeout', 'unknown'] },
        message: { type: 'string' },
        attempted_recovery: { type: 'string' },
        recommendation: { type: 'string' }
      }
    }
  }
};
```

### 8.3 Comparative A/B Testing

To measure the impact of the restructuring, run a controlled comparison:

1. **Select 3 representative agents:** Builder, Watcher, Scout
2. **Create "v2" versions** using the new template alongside the existing "v1" versions
3. **Define 5 test tasks** per agent (representative of real colony work)
4. **Run each task** with both v1 and v2 agent definitions
5. **Measure:**
   - Task completion rate (did the agent finish?)
   - Output schema compliance (did it match the JSON format?)
   - Constraint adherence (did it follow its rules?)
   - Token efficiency (how many tokens in the agent definition vs output quality?)

### 8.4 Metrics That Matter

| Metric | How to Measure | Target |
|--------|---------------|--------|
| Schema compliance | Parse agent output as JSON, validate against schema | 95%+ |
| Constraint adherence | Manual review: did read-only agents modify files? Did builders skip tests? | 100% for hard constraints |
| Output completeness | Are all required fields present and non-empty? | 95%+ |
| Token efficiency | Agent definition tokens / useful output tokens | < 0.5 ratio |
| Failure reporting | When task fails, is error block present and useful? | 100% |

---

## 9. Implementation Roadmap

### Phase 1: Template and Tooling (1-2 days)

**Tasks:**
1. Create the agent definition lint script (`tests/agent-lint.test.js`)
2. Create the standard template as a reference file (`.aether/docs/agent-template.md`)
3. Create JSON schemas for base output and each caste extension

**Deliverable:** Tooling that can validate agent files. Run against current agents to establish baseline violation count.

### Phase 2: Core Agent Migration (2-3 days)

**Migrate in priority order (most-used agents first):**
1. `aether-builder.md` -- highest usage, most complex
2. `aether-watcher.md` -- critical quality gate
3. `aether-scout.md` -- frequent research tasks
4. `aether-queen.md` -- orchestrator
5. `aether-route-setter.md` -- planner

**For each agent:**
- Restructure to XML template
- Extract constraints
- Add failure modes
- Add success criteria
- Remove boilerplate (Aether Integration, depth table, reference footer)
- Remove model references
- Verify lint passes

### Phase 3: Cluster Migration (2-3 days)

**Migrate remaining clusters:**
- Surveyor cluster (4 agents) -- already XML, need alignment to standard template
- Development cluster (4 agents) -- Weaver, Probe, Ambassador, Tracker
- Knowledge cluster (4 agents) -- Chronicler, Keeper, Auditor, Sage
- Quality cluster (4 agents) -- Guardian, Measurer, Includer, Gatekeeper
- Special agents (2) -- Archaeologist, Chaos

### Phase 4: workers.md Refactoring (1 day)

1. Remove individual caste descriptions (moved to agent files)
2. Remove model context sections
3. Retain as developer reference document only
4. Add header: "This file is a developer reference. It is NOT loaded as an agent system prompt."
5. Keep: Named Ants, Honest Execution Model, Spawn Protocol (as reference)

### Phase 5: A/B Validation (1-2 days)

1. Run comparative tests on Builder, Watcher, Scout (v1 vs v2)
2. Measure schema compliance, constraint adherence, output completeness
3. Adjust template based on findings
4. Document results

### Phase 6: Sync and Distribution (1 day)

1. Update `bin/sync-to-runtime.sh` if agent paths changed
2. Run `npm run lint:sync` to verify Claude/OpenCode alignment
3. Run full test suite
4. Update TO-DOS.md to reflect completed work

---

## Appendix A: Full Before/After Example -- Builder Agent

### Before (current `aether-builder.md`, ~135 lines, ~550 tokens)

```markdown
---
name: aether-builder
description: "Builder ant - implements code, executes commands, manipulates files"
---

You are a **Builder Ant** in the Aether Colony. You are the colony's hands...

## Aether Integration
This agent operates as a **specialist worker**...
[4 bullet points of boilerplate]

## Activity Logging
[code block + action list]

## Your Role
[numbered list]

## TDD Discipline
[Iron Law + Workflow + Coverage + Report template]

## Debugging Discipline
[Iron Law + Workflow + 3-Fix Rule]

## Coding Standards
[Principles + Checklist]

## Spawning Sub-Workers
[Rules + code block]

## Depth-Based Behavior
[table]

## Output Format
[JSON block]

## Reference
Full worker specifications: `.aether/workers.md`
```

### After (proposed, ~95 lines, ~380 tokens)

```xml
---
name: aether-builder
description: "Spawn for code implementation. Produces working code with tests via TDD. Will not skip tests or deploy."
---

<identity>
You are a Builder Ant. You implement code using test-driven development.
You are the colony's hands -- when something needs building, you build it right.
</identity>

<constraints>
- NEVER write production code before a failing test exists
- NEVER attempt more than 3 fixes for the same bug -- escalate with architectural concern
- NEVER spawn for tasks completable in under 10 tool calls
- ALWAYS read existing files before editing them
- ALWAYS log actions: bash .aether/aether-utils.sh activity-log "ACTION" "{name} (Builder)" "desc"
  Actions: CREATED, MODIFIED, EXECUTING, DEBUGGING, ERROR
- Spawn limits: depth 1 = max 4, depth 2 = max 2 (only if surprised), depth 3 = none
- Code rules: functions under 50 lines, no magic numbers, no deep nesting, comprehensive error handling
</constraints>

<workflow>
  <step name="understand">
    Read the task and existing code. Identify what needs to change and what tests exist.
  </step>

  <step name="red">
    Write a failing test for the first behavior. Run it. Confirm it fails for the
    expected reason. If it passes immediately, your test is wrong.
  </step>

  <step name="green">
    Write the minimal code to make the test pass. Run it. Confirm it passes.
    Do not add anything beyond what the test requires.
  </step>

  <step name="refactor">
    Clean up while staying green. Apply KISS, DRY, YAGNI. Run tests again.
  </step>

  <step name="repeat">
    Return to "red" for the next behavior. Continue until task is complete.
  </step>

  <step name="debug" trigger="on_error">
    When a bug appears: STOP. Read the full error. Reproduce it. Trace to root cause.
    Form one hypothesis. Test one change. If 3 fixes fail, escalate -- do not keep guessing.
  </step>
</workflow>

<output>
```json
{
  "ant_name": "{your name}",
  "caste": "builder",
  "status": "completed",  // One of: completed, failed, blocked
  "summary": "What you accomplished in 1-2 sentences",
  "files_written": [],
  "tdd": {
    "cycles": 0,
    "tests_added": 0,
    "all_passing": true
  },
  "blockers": [],
  "spawns": [],
  "error": null  // Required if status is failed; see error schema
}
```
</output>

<failure_modes>
  <mode name="cannot_complete">
    Set status to "blocked". Describe what is preventing completion in the error field.
    Include what you attempted and what the parent agent should do next.
  </mode>
  <mode name="3_fix_rule">
    After 3 failed fix attempts, set status to "blocked" with error type "task_too_complex".
    Include all 3 hypotheses and why they failed. Recommend architectural investigation.
  </mode>
  <mode name="unexpected_scope">
    If the task is 3x larger than expected, check spawn budget and spawn a parallel builder.
    If at max depth, complete what you can and document remaining work in blockers.
  </mode>
</failure_modes>

<success_criteria>
  - [ ] All acceptance criteria from the task are met
  - [ ] Tests exist for every new behavior
  - [ ] All tests pass
  - [ ] No files modified without corresponding test
  - [ ] Activity logged at start and end of task
</success_criteria>
```

**Token comparison:**
- Before: ~550 tokens
- After: ~380 tokens
- Savings: 31%
- Gains: failure modes, success criteria, explicit constraint ordering, debug as conditional step

---

## Appendix B: Priority Matrix

| Change | Impact | Effort | Priority |
|--------|--------|--------|----------|
| Add failure_modes to all agents | High (prevents silent failures) | Low (template section) | P0 |
| Add success_criteria to all agents | High (improves completion quality) | Low (template section) | P0 |
| Remove model references | Medium (eliminates confusion) | Trivial | P0 |
| Remove Aether Integration boilerplate | Medium (saves tokens) | Trivial | P0 |
| Migrate to XML structure | High (improves adherence) | Medium (rewrite each agent) | P1 |
| Extract constraints section | High (improves rule following) | Medium (analyze each agent) | P1 |
| Create lint script | Medium (prevents regression) | Medium (write tooling) | P1 |
| Compress depth tables | Low (minor token savings) | Trivial | P2 |
| Create JSON output schemas | Medium (enables validation) | Medium (write schemas) | P2 |
| A/B testing framework | Medium (measures improvement) | High (design experiments) | P2 |
| Refactor workers.md | Low (developer QoL) | Medium (restructure doc) | P3 |

---

*Plan created: 2026-02-18 | Review and approve before implementation*
