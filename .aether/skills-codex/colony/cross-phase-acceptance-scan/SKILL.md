---
name: cross-phase-acceptance-scan
description: Use when acceptance criteria or UAT gaps need to be scanned across phases before milestone closure
type: colony
domains: [uat, testing, cross-phase, audit]
agent_roles: [auditor, scout, watcher]
workflow_triggers: [continue, seal]
task_keywords: [uat, acceptance, cross-phase, untested, ship]
priority: normal
version: "1.0"
---

# Cross Phase Acceptance Scan

## Purpose

Scans across all phases to find outstanding UAT (User Acceptance Testing) items -- acceptance criteria that were defined but never verified, test gaps that span phase boundaries, and integration scenarios that fell between phases. The colony's quality safety net.

## When to Use

- User says "check all UAT items" or "what's untested?"
- Before shipping to ensure all acceptance criteria are met
- Before milestone audit to pre-identify gaps
- After all phases complete, before sealing
- User wants a comprehensive test status report

## Instructions

### 1. UAT Item Collection

```
Scan every phase for UAT items:
  1. Read each PLAN.md for acceptance criteria sections
  2. Read each SPEC.md for requirements and test cases
  3. Check for UAT markdown files in phase directories
  4. Check verification logs for passed/failed items
  5. Scan codebase for test files and their coverage
```

### 2. Status Classification

```
For each UAT item:
  VERIFIED:   Test exists and passes. Acceptance criteria met.
  PARTIAL:    Some aspects tested, others not. Criteria partially met.
  UNTESTED:   No test exists. Acceptance criteria unverified.
  FAILED:     Test exists but fails. Criteria NOT met.
  SKIPPED:    Explicitly skipped with documented reason.
  STALE:      Test exists but references removed functionality.
```

### 3. Cross-Phase Analysis

```
Identify cross-cutting concerns:
  
  INTEGRATION GAPS:
    - Phase A produces output that Phase B consumes
    - Is the integration tested? (often falls between phases)
    - Check: API contracts, data formats, event schemas
  
  END-TO-END GAPS:
    - User journey spans multiple phases
    - Is the complete journey tested?
    - Check: signup->onboard->first-action flows
  
  BOUNDARY CONDITIONS:
    - Error handling at phase boundaries
    - What happens when Phase A output is unexpected by Phase B?
    - Check: error propagation, fallback behavior
```

### 4. UAT Report

```
 UAT CROSS-SCAN -- Milestone {N}
   
   Total UAT items: {count}
    VERIFIED: {count} ({percentage}%)
    PARTIAL:  {count} ({percentage}%)
    UNTESTED: {count} ({percentage}%)
    FAILED:   {count} ({percentage}%)
    SKIPPED:  {count} ({percentage}%)
    STALE:    {count} ({percentage}%)
   
   Cross-Phase Integration Gaps:
    Phase 2 -> Phase 3: API contract untested at boundary
    Phase 4 -> Phase 5: Data format mismatch potential
   
   End-to-End Gaps:
    User signup -> first action: not tested end-to-end
    Payment -> receipt: happy path only, error path untested
   
   Priority Fixes:
   1.  {failed/critical item} -- Phase {N}
   2.  {untested/critical item} -- Phase {N}
   3.  {partial item} -- Phase {N}
   
   Report: .aether/UAT-CROSS-SCAN.md
```

### 5. Remediation Routing

```
Based on scan results:
  FAILED items -> Route to phase-forensics (why did it fail?)
  UNTESTED items -> Route to validation-gap-filler (generate tests)
  PARTIAL items -> Route to review-auto-fixer (complete coverage)
  STALE items -> Route to colony-cleanup (remove dead tests)
  Integration gaps -> Route to milestone-gap-planner (create test phase)
```

## Key Patterns

- **Cross-phase is where bugs hide**: Individual phase tests pass; integration fails.
- **Every acceptance criterion needs a test**: If it was important enough to specify, it's important enough to verify.
- **Stale tests are technical debt**: Tests that reference removed code should be cleaned up.
- **Priority by risk**: Failed and untested critical items get fixed first.

## Output Format

```
 UAT SCAN | {verified}/{total} verified ({percentage}%)
   Failed: {count} | Untested: {count} | Partial: {count}
   Cross-phase gaps: {count} | E2E gaps: {count}
   Priority fixes: {top 3}
```

## Examples

**Comprehensive scan:**
> "47 UAT items scanned. 38 verified (81%), 4 partial, 3 untested, 2 stale. Cross-phase: API contract between phases 3 and 4 untested. E2E: user signup flow has no end-to-end test. 3 priority fixes identified."

**Clean scan:**
> "23 UAT items, all verified (100%). No cross-phase gaps. No E2E gaps. Colony ready for milestone audit and ship."
