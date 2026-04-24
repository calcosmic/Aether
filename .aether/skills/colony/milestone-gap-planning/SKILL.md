---
name: milestone-gap-planning
description: Use when a milestone audit reveals gaps that need to become concrete follow-up phases
type: colony
domains: [milestone, planning, gaps, audit]
agent_roles: [architect, route_setter]
workflow_triggers: [seal, plan]
task_keywords: [gap, missing, follow-up, milestone, phase]
priority: normal
version: "1.0"
---

# Milestone Gap Planning

## Purpose

After a milestone audit reveals gaps -- missing features, incomplete tests, unmet requirements -- this skill automatically creates phases to close those gaps. It transforms "we missed X" into "here's phase 8.1 that delivers X."

## When to Use

- Milestone audit reveals incomplete coverage
- User says "fill the gaps" or "what are we missing"
- After verification fails on acceptance criteria
- Pre-ship check identifies unmet requirements
- User wants to ensure milestone completeness

## Instructions

### 1. Gap Collection

```
Read gap sources:
  1. Milestone audit report (if exists)
  2. UAT items marked as FAILED or PENDING
  3. ROADMAP phases marked INCOMPLETE
  4. Requirements without corresponding tests
  5. Pheromone signals tagged as 'gap' or 'missing'
  6. Verification failures from recent phases
```

### 2. Gap Analysis

```
For each gap:
  CLASSIFY:
    - missing_feature: Required feature not implemented
    - incomplete_test: Feature exists but lacks test coverage
    - quality_issue: Implementation doesn't meet quality bar
    - doc_gap: Missing documentation
    - config_gap: Missing configuration or setup

  SEVERITY:
    - blocking: Must fix before milestone can ship
    - important: Should fix, milestone is degraded without it
    - nice_to_have: Would improve quality but not required

  EFFORT:
    - small: <1 hour, straightforward fix
    - medium: 1-4 hours, moderate complexity
    - large: >4 hours, significant work
```

### 3. Phase Generation

```
Group gaps into phases:
  1. Group by module/area (keep related fixes together)
  2. Consider dependencies between gaps
  3. Prioritize blocking gaps first
  4. Merge small gaps into single phases where logical
  5. Split large gaps into incremental phases

Phase numbering:
  Use decimal phases to insert between existing:
  - After phase 7, add phase 7.1, 7.2, etc.
  - Preserves existing phase numbering
```

### 4. Phase Plan Creation

```
For each generated phase:
  1. Create PLAN.md with specific gap-closing tasks
  2. Define clear verification criteria (the gap is closed when...)
  3. Reference the original gap source (audit item, failed UAT, etc.)
  4. Set priority based on gap severity
  5. Add to ROADMAP.md as gap-closing phase
```

### 5. Gap Report

```
 GAP ANALYSIS -- Milestone {N}
   
   Gaps found: {total}
   Blocking: {count} | Important: {count} | Nice-to-have: {count}
   
   Phases created: {count}
   Phase 7.1: Fix auth token refresh (blocking, small)
   Phase 7.2: Add integration tests for payments (important, medium)
   Phase 7.3: Document API error codes (nice-to-have, small)
   
   Estimated effort: {total_hours} hours
   Recommended order: {phase_order}
```

### 6. Validation

```
After gap-closing phases are created:
  1. Verify no gap is left without a phase
  2. Verify no phase duplicates existing work
  3. Verify all blocking gaps are in earliest phases
  4. Verify phase dependencies are correct
  5. Run dependency check against existing phases
```

## Key Patterns

- **Gaps become phases**: Every identified gap gets its own phase, never lost.
- **Blocking first**: Gaps that prevent shipping are addressed before nice-to-haves.
- **Decimal numbering**: Gap-closing phases use decimal numbers to preserve existing structure.
- **Traceability**: Every phase traces back to the gap that created it.

## Output Format

```
 GAPS | Milestone {N}: {total} gaps -> {phases} phases
   Blocking: {count} -> phases {list}
   Effort: {hours}h | New phases: {list}
   ROADMAP updated with gap-closing phases
```

## Examples

**After audit:**
> "Audit found 8 gaps: 3 blocking, 3 important, 2 nice-to-have. Created 4 gap-closing phases (7.1-7.4). Blocking gaps in 7.1-7.2. Estimated 12 hours total effort. ROADMAP updated."

**Quick gap check:**
> "2 gaps detected: missing rate limiter tests (blocking) and undocumented error codes (nice-to-have). Phase 5.1 created for tests. Error codes deferred to next milestone."
