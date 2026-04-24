---
name: milestone-audit
description: Use before sealing a milestone to compare original intent with delivered work, quality, and remaining gaps
type: colony
domains: [milestone, audit, verification]
agent_roles: [auditor, architect, queen]
workflow_triggers: [seal]
task_keywords: [milestone, audit, complete, seal, delivered]
priority: normal
version: "1.0"
---

# Milestone Audit

## Purpose

Before a milestone is sealed as complete, this skill audits it against the original intent. It compares what was planned in PROJECT.md and the ROADMAP against what was actually delivered, identifying gaps, scope changes, quality issues, and lessons learned.

## When to Use

- User says "audit the milestone" or "verify completion"
- Before sealing a completed colony
- End-of-milestone quality gate
- User wants to verify deliverables match requirements

## Instructions

### 1. Intent Extraction

```
Read original intent from:
  1. PROJECT.md -- Colony goal and success criteria
  2. ROADMAP.md -- Planned phases and deliverables
  3. Phase PLAN.md files -- Specific phase goals
  4. SPEC.md files -- Technical specifications
  5. Requirements -- Acceptance criteria and UAT items
```

### 2. Delivery Assessment

```
For each planned deliverable:
  1. Is it implemented? (check source code)
  2. Is it tested? (check test coverage)
  3. Is it documented? (check docs)
  4. Does it meet acceptance criteria? (check UAT results)
  
  Score:
    DELIVERED:     Fully implemented, tested, and documented
    PARTIAL:       Implemented but incomplete (missing tests or docs)
    MISSING:       Not implemented
    CHANGED:       Implemented differently than planned (note changes)
    EXTRA:         Delivered but not originally planned (bonus work)
```

### 3. Scope Analysis

```
Compare planned vs actual scope:
  
  Scope expansion:
    - Features added that weren't planned
    - Phases added beyond original ROADMAP
    - Additional work discovered during execution
  
  Scope reduction:
    - Features dropped due to complexity
    - Phases deferred to future milestones
    - Simplifications made during implementation
  
  Record each scope change with:
    - What changed
    - Why it changed
    - Who made the decision
    - Impact on milestone goal
```

### 4. Quality Audit

```
Cross-cutting quality checks:
  1. Test coverage: Is it above threshold?
  2. Lint/typecheck: Clean?
  3. Security: Any known vulnerabilities?
  4. Performance: Meets requirements?
  5. Documentation: Up to date?
  6. Error handling: Comprehensive?
  7. Logging/monitoring: Adequate?
```

### 5. Audit Report

```
 MILESTONE AUDIT -- {milestone_name}
   Goal: {original_goal}
   
   DELIVERY SCORECARD:
   Phase 1:  DELIVERED   -- {deliverables}
   Phase 2:  DELIVERED   -- {deliverables}
   Phase 3:  PARTIAL    -- {what's missing}
   Phase 4:  DELIVERED   -- {deliverables}
   Phase 5:  CHANGED    -- {what changed and why}
   
   Overall: {delivered}/{total} fully delivered ({percentage}%)
   
   SCOPE CHANGES:
   + {added scope item} (reason)
   - {removed scope item} (reason)
   
   QUALITY:
   Test coverage: {percentage}%
   Lint: {clean/issues}
   Security: {clean/concerns}
   
   GAPS: {count} items need attention
   1. {gap description} -- blocking/non-blocking
   2. {gap description} -- blocking/non-blocking
   
   VERDICT: {APPROVED|CONDITIONAL|REJECTED}
   Conditions: {if CONDITIONAL, what must be fixed}
   
   Lessons Learned:
    {lesson 1}
    {lesson 2}
   
   Report: .aether/MILESTONE-AUDIT.md
```

### 6. Recommendations

```
If APPROVED:
  -> Ready to seal. Recommend pr-shipper or colony-cleanup.

If CONDITIONAL:
  -> List specific items that must be fixed
  -> Route to milestone-gap-planner to create fix phases
  -> Re-audit after fixes

If REJECTED:
  -> Major gaps identified. Recommend re-planning affected phases.
  -> Route to colony-navigator for next steps.
```

## Key Patterns

- **Original intent is sacred**: The PROJECT.md goal is the north star.
- **Partial is not delivered**: A feature without tests is a gap, not a win.
- **Changes are data**: Scope changes aren't failures -- they're information.
- **Lessons feed forward**: Every audit produces lessons for the next milestone.

## Output Format

```
 AUDIT | Milestone: {name} | {percentage}% delivered
   Verdict: {APPROVED|CONDITIONAL|REJECTED}
   Gaps: {blocking} blocking | {total} total
   Report: MILESTONE-AUDIT.md
```

## Examples

**Approved milestone:**
> "Milestone 'Auth System' audit: 95% delivered. 5/5 phases DELIVERED, 1 PARTIAL (missing rate limiter docs). Verdict: APPROVED. Minor doc gap can be addressed post-ship."

**Conditional milestone:**
> "Milestone audit: 72% delivered. Phase 3 PARTIAL (missing integration tests), Phase 5 CHANGED (simplified caching). Verdict: CONDITIONAL. Fix: add integration tests for phase 3. Then re-audit."
