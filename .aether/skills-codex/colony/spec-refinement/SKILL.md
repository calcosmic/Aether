---
name: spec-refinement
description: Use when a phase needs falsifiable requirements or ambiguity reduction before planning or building
type: colony
domains: [requirements, specification, analysis]
agent_roles: [architect, route_setter, watcher]
workflow_triggers: [discuss, plan]
task_keywords: [spec, requirement, ambiguity, clarify, underspecified]
priority: normal
version: "1.0"
---

# Spec Refinement

## Purpose
Produces a SPEC.md with falsifiable requirements by iteratively refining ambiguous phase descriptions through Socratic questioning. Uses quantitative ambiguity scoring across 4 weighted dimensions to gate spec quality -- the spec is not done until overall ambiguity drops to 0.20 or below.

## When to Use
- `aether spec` is invoked for a phase without a SPEC.md
- The architect identifies a phase as "underspecified" during roadmap review
- A phase has been discussed but requirements remain vague or contradictory
- The queen requests clarification before committing builder resources
- A phase failed execution due to spec ambiguity (retroactive refinement)

## Instructions

### Step 1 -- Initial Ambiguity Assessment
Read the phase description from `.aether/roadmap.md` and score it across 4 dimensions:

| Dimension | Weight | Measures |
|-----------|--------|----------|
| **Functional Clarity** | 0.35 | Are inputs, outputs, and behaviors explicitly defined? |
| **Boundary Definition** | 0.25 | Are edge cases, error states, and scope limits specified? |
| **Integration Contracts** | 0.25 | Are interfaces with other phases/systems defined? |
| **Acceptance Criteria** | 0.15 | Are there concrete, testable success conditions? |

Each dimension scores 0.0 (fully ambiguous) to 1.0 (fully specified). Overall ambiguity:
```
ambiguity = 1.0 - (functional * 0.35 + boundary * 0.25 + integration * 0.25 + acceptance * 0.15)
```

Gate: `ambiguity <= 0.20` to proceed. If above, continue to refinement.

### Step 2 -- Generate Refinement Questions
For each dimension scoring below 0.8, generate targeted Socratic questions:

**Functional Clarity** questions probe:
- "What happens when {input} is {edge case}?"
- "What is the expected output when {condition}?"
- "Who triggers this action and what do they see?"

**Boundary Definition** questions probe:
- "What is the maximum/minimum {value}?"
- "What happens at {limit}?"
- "Which {cases} are explicitly out of scope?"

**Integration Contracts** questions probe:
- "What does {phase X} expect from this phase?"
- "What format/schema does {downstream consumer} need?"
- "What happens if {dependency} is unavailable?"

**Acceptance Criteria** questions probe:
- "How would you verify {requirement} works?"
- "What test would prove {behavior}?"
- "What would make this phase fail verification?"

### Step 3 -- Resolve Questions
For each question, either:
1. Find the answer in existing colony artifacts (colony.md, roadmap.md, previous phase outputs)
2. Infer the answer from codebase patterns and conventions
3. Present the question to the queen (or autonomous mode: make a reasoned assumption and flag it)

Document every assumption in the SPEC.md under a "Assumptions" section.

### Step 4 -- Re-score and Loop
Re-score all 4 dimensions based on resolved questions. If ambiguity <= 0.20, proceed. Otherwise, generate another round of questions targeting the weakest dimension. Maximum 4 refinement rounds.

### Step 5 -- Write SPEC.md
Produce the specification with this structure:

```markdown
# Phase {N} Specification: {Title}

## Overview
{2-3 sentence summary}

## Requirements

### R1: {Requirement Title}
- **Given**: {precondition}
- **When**: {trigger}
- **Then**: {expected outcome}
- **Priority**: {must-have | should-have | nice-to-have}
- **Falsifiable**: {how to prove this requirement is NOT met}

### R2: {Requirement Title}
...

## Scope Boundaries
- Included: {explicit in-scope items}
- Excluded: {explicit out-of-scope items}
- Deferred: {items for future phases}

## Integration Points
| Interface | Direction | Contract |
|-----------|-----------|----------|
| {name} | {inbound/outbound} | {format/schema} |

## Assumptions
1. {assumption} -- {source: queen | inferred from {artifact} | convention}

## Ambiguity Score
| Dimension | Score |
|-----------|-------|
| Functional Clarity | {0.XX} |
| Boundary Definition | {0.XX} |
| Integration Contracts | {0.XX} |
| Acceptance Criteria | {0.XX} |
| **Overall Ambiguity** | **{0.XX}** |
```

## Key Patterns

### Falsifiability Check
Every requirement must include a falsifiable condition -- a specific scenario that would prove the requirement is NOT satisfied. If you cannot write one, the requirement is too vague.

### Assumption Tracking
Never silently assume. Every assumption gets documented with its source. During planning, assumptions become risk items.

### Progressive Refinement
Each round should focus on the weakest dimension. Resist the urge to re-refine already-strong areas -- target the lowest-scoring dimension for maximum ambiguity reduction per round.

## Output Format
- `.aether/phases/{phase}/SPEC.md` -- the refined specification
- Updates to `.aether/phases/{phase}/state.md` with `spec_ambiguity: {score}`

## Examples

### Example 1 -- REST API Phase Spec
Initial description: "Add user authentication." Ambiguity: 0.72.

Round 1 questions: "What auth methods? JWT or session? How are passwords stored? What about password reset? Rate limiting on login?"

After resolution: JWT tokens, bcrypt hashing, email-based reset, 5 req/min rate limit. Ambiguity drops to 0.15. SPEC.md written.

### Example 2 -- Ambiguous Frontend Phase
Initial: "Build the dashboard." Ambiguity: 0.85.

Round 1 (Functional): 0.3 -> generates questions about widgets, data sources, refresh behavior.
Round 2 (Boundary): 0.2 -> questions about mobile responsiveness, dark mode, accessibility.
Round 3 (Integration): 0.4 -> questions about API endpoints, real-time data, caching.
Round 4 (Acceptance): 0.5 -> questions about performance budgets, loading states.

Final ambiguity: 0.18. SPEC.md written with 12 requirements, 3 assumptions flagged.

### Example 3 -- Quick Gate Pass
Phase "Update dependencies" has clear scope. Initial ambiguity: 0.10. All dimensions above 0.85. SPEC.md written immediately with minimal refinement -- 3 requirements, no assumptions.
