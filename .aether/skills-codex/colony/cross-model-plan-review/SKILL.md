---
name: cross-model-plan-review
description: Use when a plan or high-risk approach needs independent review from another model or CLI
type: colony
domains: [review, cross-model, quality]
agent_roles: [architect, auditor]
workflow_triggers: [plan, continue]
task_keywords: [cross-model, second opinion, peer review, plan review, external review]
priority: normal
version: "1.0"
---

# Cross Model Plan Review

## Purpose

No single AI sees every blind spot. This skill dispatches colony plans to external AI CLIs (Claude, GPT, Gemini, etc.) for independent review, then converges their feedback into a unified assessment. Different models catch different issues.

## When to Use

- User says "get a second opinion" or "cross-review this plan"
- Before executing a complex or high-risk phase
- Architect wants validation of a technical approach
- Quality gate before milestone execution

## Instructions

### 1. Plan Packaging

```
Package the plan for external review:
  1. Extract PLAN.md content
  2. Include relevant PROJECT.md context
  3. Include applicable ADRs and specs
  4. Strip colony-internal references (pheromones, etc.)
  5. Add review prompt with evaluation criteria
```

### 2. Review Dispatch

```
Send to available AI CLIs:
  
  For each available CLI:
    1. Format plan in that CLI's preferred input style
    2. Include evaluation rubric:
       - Completeness: Are all requirements addressed?
       - Risks: What could go wrong?
       - Alternatives: Better approaches exist?
       - Dependencies: Missing dependencies?
       - Testing: Adequate verification plan?
    3. Request structured feedback format
    4. Set response expectations (severity, specificity)
```

### 3. Feedback Collection

```
Each reviewer returns:
{
  "reviewer": "{model_name}",
  "concerns": [
    {
      "severity": "HIGH|MEDIUM|LOW",
      "category": "completeness|risk|alternative|dependency|testing",
      "description": "{what the concern is}",
      "suggestion": "{how to address it}"
    }
  ],
  "overall_assessment": "APPROVE|CONDITIONAL|REJECT",
  "summary": "{brief overall opinion}"
}
```

### 4. Convergence Analysis

```
Merge feedback from all reviewers:
  1. Identify consensus concerns (raised by 2+ reviewers)
  2. Identify unique concerns (raised by only one reviewer)
  3. Rank by severity and consensus weight
  4. Generate conflict report where reviewers disagree

  Convergence report:
  - UNANIMOUS: All reviewers agree -> high confidence
  - MAJORITY: 2+ of 3 agree -> moderate confidence
  - SPLIT: No clear agreement -> needs user judgment
```

### 5. Review Report

```
 CROSS-MODEL REVIEW -- {plan_name}
   Reviewers: {list of models}
   
   Consensus Concerns ({count}):
    [HIGH] {concern} -- agreed by {reviewer_list}
    [MED]  {concern} -- agreed by {reviewer_list}
   
   Unique Concerns ({count}):
    [MED]  {concern} -- only from {model}
   
   Disagreements ({count}):
    {topic}: {model_A} says X, {model_B} says Y
   
   Overall: {APPROVE|CONDITIONAL|REJECT}
   Recommendation: {action based on convergence}
```

### 6. Re-Planning Loop

```
If HIGH concerns exist:
  1. Surface concerns to user
  2. User chooses: address concerns or proceed
  3. If address: re-plan affected sections
  4. Re-submit for review (max 3 cycles)
  5. Stop when no HIGH concerns remain
```

## Key Patterns

- **Independent reviews**: Reviewers don't see each other's feedback until convergence.
- **Convergence over averaging**: Consensus concerns are more reliable than any single opinion.
- **Disagreement is data**: Where models disagree reveals genuine ambiguity.
- **Limited cycles**: Re-review caps at 3 to prevent infinite refinement.

## Output Format

```
 REVIEW | {plan_name} | {reviewer_count} reviewers
   Consensus: {unanimous|majority|split}
   HIGH concerns: {count} | MED: {count} | LOW: {count}
   Verdict: {APPROVE|CONDITIONAL|REJECT}
```

## Examples

**Clean review:**
> "3 reviewers (Claude, GPT-4, Gemini). All APPROVE. 2 MEDIUM concerns raised by 2/3 reviewers. Addressing: adding error handling for edge case in wave 3."

**Split review:**
> "3 reviewers. APPROVE/CONDITIONAL/REJECT. Major disagreement: Claude recommends microservices, GPT recommends monolith. Surfacing to user for architecture decision."
