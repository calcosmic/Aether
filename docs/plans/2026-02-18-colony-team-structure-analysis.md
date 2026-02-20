# Colony Team Structure Analysis and Improvement Plan

**Date:** 2026-02-18
**Scope:** Full analysis of the 25-agent Aether Colony system — composition, coordination, spawn strategy, workflow patterns, lifecycle, failure handling, context efficiency, and scaling.

---

## Part 1: Team Composition Optimization

### Current State Assessment

The colony currently has 25 agent definitions organized across six clusters. Before prescribing changes, it is important to honestly assess how many of those 25 are actually invoked in normal operation.

**Utilization tiering (estimated from system analysis):**

| Tier | Agents | Evidence |
|------|--------|----------|
| Tier 1 — Invoked on nearly every build | Queen, Builder, Watcher, Scout, Route-Setter | Core workflow: plan, build, verify |
| Tier 2 — Invoked by specific commands | Surveyor (all 4), Chaos, Archaeologist, Tracker, Weaver, Probe | Dedicated slash commands exist |
| Tier 3 — Rarely spawned in practice | Architect, Keeper, Chronicler, Ambassador, Sage | No dedicated trigger; Queen must decide to use them |
| Tier 4 — Effectively orphaned | Guardian, Auditor, Measurer, Includer, Gatekeeper | Quality cluster has no standard spawn trigger in the core build loop |

Tier 4 is the most significant finding. The Quality cluster contains four agents — Guardian, Measurer, Includer, Gatekeeper — that overlap heavily with Watcher's scope (Watcher already checks security, performance, and quality through its "specialist modes") and have no scheduled place in the default build choreography. They are available for on-demand spawning but the Queen has no documented heuristic for when to choose one of them over simply running a thorough Watcher pass.

### Role Overlap Map

The following overlaps create real decision paralysis for the Queen:

```
Architect   ↔   Keeper       Both: synthesize knowledge, extract patterns, maintain docs
Watcher     ↔   Guardian     Both: security review, vulnerability checking
Watcher     ↔   Auditor      Both: code quality review with severity ratings
Chaos       ↔   Probe        Both: edge case investigation; Chaos is read-only, Probe writes tests
Scout       ↔   Archaeologist Both: investigate codebase; Archaeologist scopes to git history
Chronicler  ↔   Architect    Both: produce documentation artifacts
```

### Recommendations: Composition

**Recommendation 1.1 — Consolidate Architect into Keeper.**
Architect and Keeper have near-identical behavioral mandates. Architect's description: "synthesizes knowledge, extracts patterns, coordinates documentation." Keeper's description: "knowledge curation and pattern archiving." The only meaningful distinction is Architect's claim to "glm-5" model preference. Merge these into a single Keeper agent that inherits Architect's synthesis workflow and Keeper's organizational taxonomy. Net saving: one agent file to maintain.

**Recommendation 1.2 — Fold Guardian into Auditor as a named lens.**
Guardian is a focused version of Auditor's "Security Lens." Rather than a separate agent, Guardian's OWASP Top 10 methodology should become a named spawn parameter: `aether-auditor --lens security`. This gives the Queen one fewer agent to reason about and makes the lens explicit in the spawn call. Auditor already has a Security Lens section in its file. Guardian's CVE-specific work (dependency scanning) overlaps with Gatekeeper and should be noted there instead.

**Recommendation 1.3 — Establish clear Chaos vs Probe separation.**
These two agents are easy to confuse. The separation is actually clear and worth preserving — Chaos is strictly read-only investigation (produces a findings report), Probe actively writes tests. The problem is the Queen has no heuristic for when to use which. Add explicit trigger language to each agent:
- Chaos: spawn when you want to understand how a module breaks before touching it
- Probe: spawn when you want to increase test coverage or verify a Chaos finding is covered

**Recommendation 1.4 — Give the Quality cluster a scheduled place in the build loop.**
The root cause of Quality cluster underutilization is not agent quality — it is the absence of a defined moment when they run. Recommendation: define a "quality gate" phase that runs after a successful Watcher pass. The Queen selects from {Gatekeeper, Measurer, Includer} based on phase type:

| Phase type | Quality gate agent |
|------------|-------------------|
| Any phase with new dependencies | Gatekeeper |
| Performance-sensitive feature | Measurer |
| UI/component work | Includer |
| General | Auditor (quality lens) |

**Recommendation 1.5 — Create a Colonizer agent file.**
Colonizer exists in workers.md and is referenced by the Queen but has no standalone agent file. This is an inconsistency. The Surveyor cluster essentially replaced Colonizer for structured codebase mapping, but Colonizer fills a different niche: lightweight exploration before planning, not the full four-document survey. Create `.aether/agents/aether-colonizer.md` with a narrow scope: directory tree, entry points, key file locations — a 10-minute orientation pass rather than a full survey.

**Recommendation 1.6 — Optimal team size is 18, not 25.**
After consolidation (Architect into Keeper, Guardian into Auditor lens), adding Colonizer as a proper agent, and retiring the Oracle as a standalone agent file (it is adequately covered by a dedicated command), the effective team becomes:

| Cluster | Agents | Count |
|---------|--------|-------|
| Core | Queen, Builder, Watcher, Scout, Route-Setter, Colonizer | 6 |
| Surveyor | nest, disciplines, provisions, pathogens | 4 |
| Development | Weaver, Probe, Tracker, Ambassador, Chaos | 5 |
| Knowledge | Chronicler, Keeper, Sage | 3 |
| Quality | Auditor, Measurer, Includer, Gatekeeper | 4 |
| **Total** | | **22** |

This is a reduction from 25 to 22 via merges, with better-defined trigger conditions for each agent. The number is still large, but every remaining agent now has a distinct niche.

---

## Part 2: Coordination Protocol Design

### Current State Assessment

The colony has three coordination mechanisms:
1. JSON handoff — each agent returns a structured JSON result that the parent synthesizes
2. Pheromone signals — FOCUS, REDIRECT, FEEDBACK stored in constraints.json
3. Activity log — shared append-only log that all agents write to

These are functional but incomplete. Three specific gaps cause friction:

**Gap A — Parallel work has no conflict protocol.** When two Builders work on related files simultaneously, neither knows what the other is doing. The spawn system logs their existence in spawn-tree.txt but does not communicate their file scopes to each other.

**Gap B — Failure has no defined escalation chain.** When a depth-2 specialist fails, the behavior is undefined. Does the parent retry? Spawn a replacement? Escalate to Queen? The only guidance is the Builder's "3-Fix Rule" (escalate after three failed fixes), but there is no receiving protocol on the parent side.

**Gap C — Context is cold-started.** Each spawned agent begins with no knowledge of what previous agents in the same phase have already discovered or done. The parent must re-summarize everything in the spawn prompt. This is expensive and lossy.

### Recommendations: Coordination

**Recommendation 2.1 — File lock protocol for parallel Builders.**
When a Builder is spawned, it should declare its intended file scope before beginning work:

```bash
bash .aether/aether-utils.sh file-lock-claim "{agent_name}" "{file_paths_csv}"
# Returns: {"claimed": true, "conflicts": []}
```

If another agent has claimed the same file, the spawn prompt should include that information so the new Builder either coordinates (waits, takes a different file) or the parent Prime decides not to parallelize that particular work. The lock is released on spawn completion. This requires a small addition to aether-utils.sh — a simple JSON file in `.aether/data/locks/` per file, similar to the existing flag system.

**Recommendation 2.2 — Define the escalation chain explicitly.**
The current lack of a defined escalation path means failures silently bubble up as blocked JSON, requiring the parent to improvise. The protocol should be:

```
Depth-3 failure:
  → Return {"status": "failed", "escalation_needed": true, "reason": "..."}
  → Parent (depth 2) decides: retry inline or escalate

Depth-2 failure (after 1 retry):
  → Return blocked JSON with flag already created
  → Parent (depth 1 Prime) decides: spawn replacement, skip subtask, or escalate

Depth-1 Prime failure:
  → Return blocked JSON to Queen
  → Queen decides: re-plan phase, re-spawn Prime, or mark phase blocked

Queen-level block:
  → Write to flags.json with severity CRITICAL
  → Surface to user via /ant:flags output
  → DO NOT silently continue
```

The key addition is that each level must actively handle the failure from below rather than passing it up unchanged. The Prime's job is to absorb and triage depth-2 failures; the Queen absorbs depth-1 failures.

**Recommendation 2.3 — Phase scratch pad for shared context.**
Create a lightweight per-phase shared state file: `.aether/data/phase-scratch.json`. This file is:
- Written by the Prime Worker at phase start (summarizing the goal and initial context)
- Readable by all agents in the phase as an optional orientation document
- Appended to by agents when they make discoveries that downstream agents need to know
- Cleared at phase end by the Prime

Structure:
```json
{
  "phase": "2",
  "goal": "...",
  "discoveries": [
    {"by": "Scout-12", "at": "2026-02-18T10:00:00Z", "note": "auth module uses JWT RS256, not HS256"}
  ],
  "file_locks": {}
}
```

This eliminates the need for the parent to re-summarize everything in each spawn prompt. Agents can self-orient by reading the scratch pad. The Prime updates the scratch pad rather than holding all context.

**Recommendation 2.4 — Standardize the handoff format.**
Currently each caste defines its own JSON output schema. This makes the parent's synthesis logic inconsistent — it must know each caste's schema to extract the relevant fields. Add a mandatory envelope that every agent must return, wrapping their caste-specific payload:

```json
{
  "envelope": {
    "ant_name": "...",
    "caste": "...",
    "status": "completed | failed | blocked",
    "phase": "...",
    "duration_estimate": "...",
    "flags_created": [],
    "files_affected": []
  },
  "payload": {
    // caste-specific fields here
  }
}
```

The Queen and Prime Workers always read `envelope` first. They only dig into `payload` for synthesis. This makes failure detection uniform.

---

## Part 3: Spawn Strategy Optimization

### Current State Assessment

The spawn rules are:
- Depth 0 (Queen): max 4 direct spawns
- Depth 1 (Prime): max 4 sub-spawns
- Depth 2 (Specialist): max 2, only if 3x surprise
- Depth 3: no spawning
- Global cap: 10 workers per phase

These limits are conservative and generally reasonable. The real problems are not the limits themselves but the triggering logic and the absence of a budget-allocation strategy.

### Recommendations: Spawn Strategy

**Recommendation 3.1 — Replace the "3x surprise" rule with explicit spawn triggers.**
The current rule ("spawn if genuinely surprised — task is 3x larger than expected") is subjective and inconsistently applied. Each agent file should instead enumerate its named spawn triggers. For example, Builder's triggers:

```
Spawn a Watcher when: implementation is complete and needs independent verification
Spawn a Scout when: unfamiliar library or API is encountered mid-task
Spawn another Builder when: independent subtasks are discovered that do not share state
DO NOT spawn for: sequential steps in a known workflow, even if there are many of them
```

This changes the spawn decision from a magnitude judgment (how big is this?) to a categorical judgment (does this situation match a known trigger?).

**Recommendation 3.2 — Pre-allocate spawn budget at phase start.**
The Queen should explicitly allocate the spawn budget when dispatching a Prime:

```
Phase 3 (implement user auth):
  Prime Builder: budget = 6 workers
  Breakdown: up to 3 Builders (parallel implementation), 1 Watcher, 1 Scout if needed, 1 reserved
```

This gives the Prime a concrete constraint rather than requiring it to dynamically calculate against the global cap. The Queen tracks total allocation across all Primes to stay within the 10-worker global limit. If a Prime needs to exceed its budget, it escalates to the Queen before spawning (not after).

**Recommendation 3.3 — Differentiate parallel vs serial spawn patterns.**
The current spawn model treats all sub-spawns the same. Two distinct patterns should be named and used deliberately:

**Fan-out pattern:** Spawn N specialists simultaneously, collect all results, synthesize. Use for: parallel research across independent domains, parallel implementation of independent features.
```
Prime → [Scout-A, Scout-B, Scout-C] → wait all → synthesize → continue
```

**Pipeline pattern:** Spawn one specialist, use its output to inform the next spawn. Use for: survey → plan → build → verify sequences.
```
Prime → Route-Setter → (receive plan) → Builder → (receive result) → Watcher
```

The Prime Worker should declare which pattern it is using. This makes spawn trees readable and makes it clear when agents can run in parallel vs must wait.

**Recommendation 3.4 — Allow depth-1 Primes to choose their own caste.**
Currently the Queen selects the Prime's caste (Builder, Scout, etc.) before spawning. This is limiting — the Queen must perfectly predict the nature of the work upfront. An alternative: spawn a generic "Prime" coordinator at depth 1 whose first action is to assess the task and select which specialist castes to spin up. This Prime is always the same agent (could be a lightweight version of Route-Setter) regardless of the phase type. It reads the phase goal, checks the scratch pad, then makes spawn decisions. This adds one level of indirection but gives the system more adaptive capacity.

This is an experimental recommendation — implement only after the basic coordination improvements are stable.

---

## Part 4: Workflow Pattern Library

The Queen should have an explicit library of named choreographies rather than improvising team composition each time. The following patterns cover the majority of real use cases.

### Pattern A: Survey-Plan-Build-Verify (SPBV)
The standard development loop.

```
Phase 0: Survey (if codebase not yet mapped)
  Queen → [Surveyor-Nest, Surveyor-Disciplines, Surveyor-Provisions, Surveyor-Pathogens]
  (parallel fan-out, all write to .aether/data/survey/)

Phase 1: Plan
  Queen → Route-Setter
  (Route-Setter reads survey docs, produces phase plan JSON)

Phase N: Build
  Queen → Prime Builder
  Prime Builder → [Builder-A, Builder-B] (parallel, non-overlapping files)
  Builder-A or B → Watcher (after implementation complete)

Phase N+1: Quality Gate
  Queen → appropriate quality agent (see Part 1, Rec 1.4)
```

Total workers in typical build phase: 4-6

### Pattern B: Investigate-Fix (IF)
For bug reports and regressions.

```
Queen → Tracker (investigate root cause)
Tracker → Scout (if external library involved)
Tracker → (returns root cause to Queen)
Queen → Builder (implement fix with root cause in context)
Builder → Watcher (verify fix)
Builder → Chaos (probe for related edge cases)
```

Total workers: 3-5

### Pattern C: Deep Research
For architecture decisions, library selection, or complex unknowns.

```
Queen → Prime Scout
Prime Scout → [Scout-A (docs), Scout-B (codebase), Scout-C (external)] (parallel fan-out)
Prime Scout → Archaeologist (if historical context needed)
Prime Scout → Keeper (to archive findings)
(Prime Scout synthesizes, returns recommendations to Queen)
```

Total workers: 3-5

### Pattern D: Refactor Loop
For cleaning up existing code without behavior change.

```
Queen → Weaver (analyze and execute refactor)
Weaver → Probe (verify behavior preserved via test coverage)
Probe → Watcher (run full test suite)
Queen → Auditor (quality lens on refactored code)
```

Total workers: 4

### Pattern E: Compliance Audit
For pre-release or periodic quality checks.

```
Queen → [Guardian/Auditor, Gatekeeper, Measurer, Includer] (parallel fan-out — all are read-only)
(each returns findings JSON)
Queen → Chronicler (produces consolidated audit report)
```

Total workers: 5

### Pattern F: Documentation Sprint
For documentation-heavy phases.

```
Queen → Scout (understand what needs documenting)
Scout → (returns inventory of gaps)
Queen → Chronicler (write docs for the gaps)
Chronicler → Keeper (archive new patterns discovered during documentation)
```

Total workers: 3

### Pattern Selection Heuristic for the Queen

| Phase keyword | Pattern |
|---------------|---------|
| "implement", "build", "add feature" | SPBV if not recently surveyed; Build phase only if surveyed |
| "fix", "debug", "regression" | Investigate-Fix |
| "research", "evaluate", "should we use" | Deep Research |
| "refactor", "clean up", "simplify" | Refactor Loop |
| "audit", "security review", "pre-release" | Compliance Audit |
| "document", "update README" | Documentation Sprint |

---

## Part 5: Agent Lifecycle Management

### Current State Assessment

Agents are spawned on demand and retired when they return their JSON result. There is no concept of agent health, warm-up, or longitudinal effectiveness tracking. The only persistent data is the spawn-tree.txt log and the activity log.

### Recommendations: Lifecycle

**Recommendation 5.1 — Context priming protocol (warm-up).**
Every spawned agent should follow a three-step orientation before beginning work:

Step 1: Read phase scratch pad (`.aether/data/phase-scratch.json`) — understand what has already been learned in this phase.

Step 2: Read relevant survey documents — based on phase type, load the appropriate BLUEPRINT/CHAMBERS/DISCIPLINES files. This should be declared by the Prime in the spawn prompt ("for this phase, you need BLUEPRINT.md and DISCIPLINES.md").

Step 3: Check pheromone signals — read constraints.json for any REDIRECT signals that apply to the work. REDIRECT signals are hard constraints; the agent should confirm it has read them before proceeding.

This replaces the current cold-start behavior where each agent must rediscover context. Estimated context savings: 20-30% reduction in redundant exploration tool calls.

**Recommendation 5.2 — Effectiveness metadata per caste.**
Add a lightweight performance tracking file: `.aether/data/caste-metrics.json`. Structure:

```json
{
  "builder": {
    "spawns_total": 47,
    "status_completed": 44,
    "status_failed": 2,
    "status_blocked": 1,
    "avg_flags_created": 0.3,
    "last_spawned": "2026-02-18T10:00:00Z"
  }
}
```

The Sage agent (analytics) reads this file. The aether-utils.sh `spawn-complete` command writes to it. This gives Sage real data to analyze rather than scanning raw activity logs.

**Recommendation 5.3 — Distinguish "failed" from "blocked".**
Currently `status: "failed"` covers both "I tried and could not complete the task" (internal failure) and "I cannot proceed because of an external blocker" (external block). These require different responses. Formalize three terminal states:

- `completed` — work done, meets success criteria
- `failed` — attempted, could not succeed (structural problem, 3-fix rule triggered, etc.)
- `blocked` — cannot proceed without external input (missing dependency, conflicting requirement, missing access)

A failed result tells the parent to consider a different approach. A blocked result tells the parent to resolve the external condition first. The Queen handles these differently.

---

## Part 6: Failure Handling and Recovery

### Current State Assessment

This is the most significant gap in the current design. The failure protocol is essentially: agents return `"status": "failed"` in their JSON, and the parent is expected to decide what to do. No formal escalation chain exists. The Builder has a "3-Fix Rule" that says to escalate, but there is no documented receiving protocol.

### Recommended Failure Handling Architecture

**Level 1 — Self-recovery (agent level).**
Before returning `failed`, an agent should attempt one of three recovery strategies:

1. Retry with a narrower scope (if the task was ambiguous)
2. Spawn a Scout to research the unknown (if the failure was due to lack of information)
3. Apply the 3-Fix Rule (Builder/Tracker specific): if three distinct fix attempts fail, stop and escalate — do not invent a fourth

If all self-recovery attempts fail, set `"escalation_needed": true` in the envelope and return.

**Level 2 — Prime triage (depth-1 level).**
The Prime receives a failed/blocked result from a depth-2 specialist. The Prime's decision tree:

```
Is the failure blocking the phase goal?
  YES → Can it be retried with a different specialist?
    YES → Spawn replacement (if within budget), log retry in scratch pad
    NO  → Escalate to Queen with full context
  NO  → Mark subtask as skipped, continue with remaining work, note in phase summary
```

The Prime should never silently absorb a failure that affects the phase outcome.

**Level 3 — Queen decision (depth-0 level).**
The Queen receives a failed phase from a Prime. The Queen's decision tree:

```
Is the failure due to a missing prerequisite?
  YES → Re-order phases (run prerequisite first), re-queue this phase
  NO  → Is the failure recoverable with more information?
    YES → Spawn a Scout/Archaeologist to investigate, then re-attempt
    NO  → Flag as CRITICAL blocker, surface to user, pause colony
```

The Queen should never continue to the next phase if the current phase's success criteria were not met. The "Iron Law" already establishes this — the failure handling protocol makes it actionable.

**Recommended flag severity mapping:**

| Failure type | Flag severity | Auto-pause? |
|-------------|---------------|-------------|
| Test failure after implementation | HIGH | No — retry with Watcher |
| 3-Fix Rule triggered | HIGH | No — escalate to Queen for re-planning |
| Phase blocked (external dependency) | CRITICAL | Yes — surface to user |
| Security finding (CRITICAL severity) | CRITICAL | Yes — do not advance phase |
| Queen-level re-plan failed twice | CRITICAL | Yes — human required |

**Graceful degradation strategy:**
If a quality-gate agent (Measurer, Includer, Gatekeeper) fails, the phase can still complete with a warning rather than a full block. These agents produce recommendations, not hard gates (unless they find CRITICAL severity issues). A failed Measurer means "we don't have performance data" not "the feature is broken." Model this explicitly: quality gate agents return `"advisory": true` when their findings are non-blocking.

---

## Part 7: Context Efficiency

### Current State Assessment

Each spawned agent currently receives:
- The full agent definition file (loaded as system prompt)
- A spawn prompt constructed by the parent, which includes parent context, task description, pheromone signals
- A reference to read `.aether/workers.md` (a large document)

The workers.md reference is the primary inefficiency. Agents are instructed to read it as part of initialization, but workers.md is a long document that contains information about all castes — most of which is irrelevant to the spawned agent. Every agent that reads workers.md consumes tokens describing 21 other castes it will never become.

### Recommendations: Context Efficiency

**Recommendation 7.1 — Remove the workers.md read instruction from agent files.**
The "## Reference / Full worker specifications: .aether/workers.md" section appears in every agent file and instructs the agent to read the full workers.md on initialization. Remove this instruction. If an agent needs to know about another caste's behavior (e.g., Builder needs to know what Watcher expects), that information should be in the Builder's own file as a "Handoff expectations" section.

The workers.md document should be a reference for the Queen and for humans, not a document that spawned workers read at runtime.

**Recommendation 7.2 — Keep agent files under 150 lines.**
Review each agent file against this budget:
- Frontmatter + identity: ~10 lines
- Core role and workflow: ~30 lines
- Domain-specific detail (techniques, checklists): ~50 lines
- Spawn triggers and depth rules: ~20 lines
- Output format JSON: ~30 lines
- Total: ~140 lines

Files that exceed 150 lines are carrying information that should be in a reference document or removed entirely. The Surveyor agents (which use XML tags and embedded templates) are the worst offenders — surveyor-nest.md is 273 lines. The template content in these files could be moved to external template files and referenced rather than embedded.

**Recommendation 7.3 — Lazy-load survey documents.**
Survey documents (BLUEPRINT.md, CHAMBERS.md, etc.) are currently loaded by convention — agents are expected to know which ones to read. This is efficient in principle but requires agents to re-discover the documents each time. Formalize this: the Prime Worker reads the phase type from the phase plan and includes explicit document paths in the spawn prompt:

```
Context documents for this phase (load before starting):
- .aether/data/survey/BLUEPRINT.md
- .aether/data/survey/DISCIPLINES.md
Do NOT load PROVISIONS.md, TRAILS.md, PATHOGENS.md unless you encounter a specific need.
```

This prevents agents from speculatively loading all survey documents "just in case."

**Recommendation 7.4 — Compress JSON output.**
The current output format uses verbose field names and nested objects that are fine for human readability but consume tokens unnecessarily in agent-to-agent communication. For internal handoffs (Prime to Queen), a compressed summary format is better:

```
Builder-42: completed | files: auth.ts, auth.test.ts | tests: 6 added, 100% passing | flags: none
```

The full JSON output is still written to `.aether/data/results/` for audit purposes, but the in-context return to the parent is the compressed summary. The parent asks for the full JSON only if it needs specific details.

**Recommendation 7.5 — Unify the XML vs JSON format inconsistency.**
Surveyor agents use XML tags (`<role>`, `<process>`, `<step name="...">`) while all other agents use Markdown headers. This is the most prominent structural inconsistency in the codebase. The XML format has real advantages — it creates explicit scope boundaries and is easier to parse programmatically. The recommendation is to decide one way:

Option A (recommended): Convert all agents to use a lightweight XML structure for the major sections (identity, workflow, rules, output). This gives consistent machine-parseable structure across all 22 agents.

Option B: Convert Surveyor agents to use the Markdown format used by everyone else, trading parsability for consistency with the majority.

The Surveyor XML format is genuinely better designed. Option A is preferable.

---

## Part 8: Scaling Considerations

### Adding a New Agent Type

The current process for adding a new caste is:
1. Write `.aether/agents/aether-{name}.md`
2. Write `.opencode/agents/aether-{name}.md` (duplicate)
3. Add the caste to workers.md
4. Add it to the Queen's caste list in aether-queen.md
5. Run `npm run lint:sync` to verify sync

This process has three duplication problems: the agent file is maintained in two locations, the caste information lives in three places (agent file, workers.md, queen.md), and there is no template to guide new caste creation.

**Recommendation 8.1 — Create a new caste template.**
Add `.aether/agents/TEMPLATE.md` with the standard structure, placeholder content, and a checklist of what must be filled in. This reduces onboarding friction for new castes.

**Recommendation 8.2 — Define the "caste contract."**
Every new caste must fulfill:
- A unique trigger condition that no existing caste already handles
- A read-only vs read-write declaration (critical for parallel work safety)
- Named spawn triggers (per Recommendation 3.1)
- Envelope-compatible JSON output (per Recommendation 2.4)
- Advisory vs blocking behavior declaration (per Recommendation 6.0)

Document this contract in `.aether/docs/caste-contract.md`.

**Recommendation 8.3 — Domain-specific caste extensions.**
Some projects will need castes that do not belong in the core Aether distribution — for example, a `data-engineer` caste for a data-heavy project, or a `mobile-ui` caste for a React Native project. These project-specific castes should:
- Live in `.aether/agents/` like any other caste
- NOT be synced to the hub (add to the allowlist exclusion for project-specific files)
- Follow the caste contract
- Be declared in a `.aether/data/custom-castes.json` manifest so the Queen knows they exist

This prevents project-specific castes from polluting the core Aether distribution while still allowing full customization.

**Recommendation 8.4 — Caste versioning.**
When a caste's behavioral contract changes materially (not just documentation), increment a version in the frontmatter:

```yaml
---
name: aether-builder
version: 2
description: "..."
---
```

The Queen reads caste versions when loading agents. If a caste version has changed since the last colony session, the Queen notes it in the session log. This is useful when debugging inconsistent behavior across sessions — "Builder v2 was introduced mid-colony, which changed TDD behavior."

---

## Summary: Priority Order for Implementation

The recommendations above are numerous. This section provides a prioritized sequence.

### Immediate — High impact, low effort

1. **Remove workers.md read instructions from agent files** (Rec 7.1) — one-line change per agent file, reduces context load on every spawn
2. **Define the escalation chain explicitly** (Rec 2.2) — document-only change, no code required
3. **Add explicit spawn triggers to each agent file** (Rec 3.1) — improves Queen decision quality immediately
4. **Define the three terminal states** (Rec 5.3) — document change, reduces ambiguity in failure handling

### Short term — Require modest implementation

5. **Create phase-scratch.json** (Rec 2.3) — small addition to aether-utils.sh, high coordination value
6. **Standardize the handoff envelope** (Rec 2.4) — requires updating all agent output formats
7. **File lock protocol for parallel Builders** (Rec 2.1) — requires small aether-utils.sh addition
8. **Consolidate Architect into Keeper** (Rec 1.1) — one file merge, update Queen's caste list
9. **Fold Guardian into Auditor as a lens** (Rec 1.2) — update Auditor file, remove Guardian file, update Queen's list

### Medium term — Require workflow redesign

10. **Workflow pattern library in Queen's file** (Rec Part 4) — rewrite aether-queen.md dispatch section
11. **Pre-allocate spawn budget at phase start** (Rec 3.2) — requires Queen-level logic change
12. **Quality gate scheduling** (Rec 1.4) — requires build loop change
13. **Create Colonizer agent file** (Rec 1.5) — new file, narrow scope
14. **Caste metrics tracking** (Rec 5.2) — aether-utils.sh addition + Sage integration

### Long term — Architectural improvements

15. **XML format unification** (Rec 7.5) — affects all 22 agent files
16. **Domain-specific caste extensions** (Rec 8.3) — requires hub allowlist changes
17. **Caste versioning** (Rec 8.4) — low urgency, high long-term value
18. **Agent file size budget enforcement** (Rec 7.2) — ongoing editorial discipline

---

## Key Files Referenced

| File | Role in recommendations |
|------|------------------------|
| `/Users/callumcowie/repos/Aether/.aether/agents/aether-queen.md` | Add workflow pattern library, update caste list |
| `/Users/callumcowie/repos/Aether/.aether/agents/aether-builder.md` | Add explicit spawn triggers, remove workers.md read |
| `/Users/callumcowie/repos/Aether/.aether/agents/aether-watcher.md` | Clarify vs Guardian/Auditor scope |
| `/Users/callumcowie/repos/Aether/.aether/agents/aether-auditor.md` | Absorb Guardian's security lens |
| `/Users/callumcowie/repos/Aether/.aether/agents/aether-keeper.md` | Absorb Architect's synthesis workflow |
| `/Users/callumcowie/repos/Aether/.aether/agents/aether-architect.md` | Candidate for consolidation into Keeper |
| `/Users/callumcowie/repos/Aether/.aether/agents/aether-guardian.md` | Candidate for consolidation into Auditor |
| `/Users/callumcowie/repos/Aether/.aether/agents/aether-surveyor-nest.md` | Example of oversized agent file (273 lines) |
| `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh` | Needs: file-lock-claim, phase-scratch, caste-metrics additions |
| `/Users/callumcowie/repos/Aether/.aether/workers-new-castes.md` | Source for model assignment rationale (Weaver/Leafcutter/Soldier) |
