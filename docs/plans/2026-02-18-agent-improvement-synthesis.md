# Agent System Improvement -- Synthesis Overview

**Date:** 2026-02-18
**Status:** Awaiting review
**Companion documents:**
- `2026-02-18-agent-definition-architecture-plan.md` -- How each agent file should be structured (prompt engineering, templates, token optimization)
- `2026-02-18-colony-team-structure-analysis.md` -- How agents work together (team composition, coordination, workflows)

---

## What This Is About

The Aether colony has 25 agent definition files. Two specialist analyses were run in parallel to find improvement opportunities. This document synthesizes their findings into a single prioritized action plan.

---

## The Big Picture (Plain English)

Your agent system works, but it's accumulated some drift. Think of it like a team where everyone has a job description, but some descriptions are outdated, some overlap with other roles, and there's no standard format. The system still functions because the core loop (plan, build, verify) is solid, but there's room to make every agent sharper and the team leaner.

**Three themes emerged:**

1. **Structure matters more than length.** The surveyor agents (which use XML-style formatting) perform better than the flat-markdown agents. Adopting that structure everywhere would improve how reliably agents follow their instructions.

2. **The team has bloat.** Some agents duplicate each other's work (Architect vs Keeper, Guardian vs Auditor). Some are never actually used in the normal workflow. Trimming from 25 to ~22 agents with clear trigger conditions for each makes the Queen's job easier.

3. **Failure is undefined.** No agent currently knows what to do when things go wrong. There's no escalation chain, no error reporting format, no recovery protocol. This is the highest-impact gap to fix.

---

## What Both Analyses Agree On (Highest Confidence)

| Finding | LLM Architect | Agent Organizer |
|---------|:---:|:---:|
| Remove workers.md reference from all agent files | Yes | Yes |
| Add failure modes / escalation to every agent | Yes | Yes |
| Add success criteria to every agent | Yes | Yes |
| Remove dead model references (glm-5, kimi-k2.5) | Yes | Yes |
| Adopt XML structure for all agents | Yes | Yes |
| Consolidate Architect into Keeper | -- | Yes |
| Consolidate Guardian into Auditor | Noted overlap | Yes |
| Standardize output JSON with common base schema | Yes | Yes |
| Define read-only vs read-write per agent | Yes | Yes |
| Compress boilerplate (~32% token savings) | Yes | -- |

---

## Unified Priority Roadmap

### Wave 1: Quick Wins (document changes only, no code)

These can be done immediately with zero risk:

1. **Remove workers.md reference footer** from all 25 agent files
2. **Remove model references** (glm-5, kimi-k2.5, benchmark scores) from all agent files
3. **Remove "Aether Integration" boilerplate** section from all agent files
4. **Add failure_modes section** to every agent (use template from architecture plan)
5. **Add success_criteria section** to every agent
6. **Add read-only/read-write declaration** as first constraint in each agent
7. **Add named spawn triggers** to Builder, Watcher, Scout (replace vague "3x surprise" rule)

**Impact:** ~32% token reduction per agent, failure handling defined, clearer boundaries.

### Wave 2: Template Migration (structured rewrite)

Convert all agents to the standard XML template:

1. **Create the template** at `.aether/docs/agent-template.md`
2. **Migrate core agents first:** Builder, Watcher, Scout, Queen, Route-Setter
3. **Migrate remaining clusters:** Surveyors (align to template), Development, Knowledge, Quality
4. **Create lint script** (`tests/agent-lint.test.js`) to enforce template compliance

**Impact:** Consistent structure across all agents, automated validation.

### Wave 3: Team Consolidation

1. **Merge Architect into Keeper** -- one knowledge agent instead of two
2. **Fold Guardian into Auditor** as a named security lens
3. **Create Colonizer agent file** (currently referenced but missing)
4. **Define workflow pattern library** in Queen's file (SPBV, Investigate-Fix, Deep Research, etc.)
5. **Add quality gate scheduling** -- give Quality cluster agents a defined place in the build loop

**Impact:** 22 agents instead of 25, every agent has a clear trigger, Quality cluster no longer orphaned.

### Wave 4: Coordination Infrastructure (requires code)

1. **Phase scratch pad** (`.aether/data/phase-scratch.json`) for shared context within a phase
2. **File lock protocol** for parallel Builders to avoid conflicts
3. **Escalation chain** implemented in aether-utils.sh
4. **Caste metrics tracking** for effectiveness data
5. **Standardized handoff envelope** wrapping all agent output

**Impact:** Agents share context, parallel work is safe, failures are handled systematically.

### Wave 5: Validation

1. **A/B test** Builder, Watcher, Scout -- old format vs new format on 5 representative tasks each
2. **Measure:** schema compliance, constraint adherence, token efficiency
3. **Refactor workers.md** to be a developer reference only (not loaded by agents at runtime)
4. **Sync and distribute** via `npm install -g .`

---

## Key Decisions Needed From You

Before starting implementation, a few choices:

1. **Agent consolidation scope** -- Are you comfortable merging Architect into Keeper and Guardian into Auditor? Or do you prefer to keep them separate with clearer boundaries?

2. **XML template adoption** -- The plan recommends converting all agents from flat markdown to XML-structured format. This means rewriting every agent file. Worth the investment?

3. **Coordination infrastructure** -- Phase scratch pad and file locks require new code in aether-utils.sh. Want to include these or defer?

4. **Implementation approach** -- Do this as a dedicated colony build phase? Or gradually over multiple sessions?

---

## File Inventory (What's in docs/plans/)

| File | Content | Lines |
|------|---------|-------|
| `2026-02-18-agent-improvement-synthesis.md` | This overview | ~130 |
| `2026-02-18-agent-definition-architecture-plan.md` | Detailed prompt engineering plan with before/after examples | ~1060 |
| `2026-02-18-colony-team-structure-analysis.md` | Detailed team composition and coordination plan | ~607 |

Read the synthesis first. Dive into the detailed plans for specifics on any area.

---

*Ready for your review. Let me know which direction feels right and I'll start building.*
