---
name: evaluation-coverage-audit
description: Use when completed work needs evaluation coverage audited across correctness, security, performance, and quality dimensions
type: colony
domains: [evaluation, testing, quality, audit]
agent_roles: [auditor, probe, watcher]
workflow_triggers: [continue]
task_keywords: [evaluation, coverage, missing, partial, quality gate]
priority: normal
version: "1.0"
---

# Evaluation Coverage Audit

## Purpose

After a phase is executed, this skill retroactively audits whether the evaluation criteria were actually covered. It scores each eval dimension (correctness, performance, security, etc.) as COVERED, PARTIAL, or MISSING, and produces an actionable remediation plan for gaps.

## When to Use

- User says "check eval coverage" or "audit the tests"
- After phase execution to verify quality
- Before shipping to ensure nothing was missed
- Milestone completion review

## Instructions

### 1. Eval Dimensions

```
Standard evaluation dimensions:
  CORRECTNESS:   Does it do what was specified?
  EDGE_CASES:    Does it handle edge cases and errors?
  PERFORMANCE:   Does it meet performance requirements?
  SECURITY:      Is it secure against known threats?
  ACCESSIBILITY: Is it accessible (if UI)?
  COMPATIBILITY: Does it work across required environments?
  INTEGRATION:   Does it integrate correctly with dependencies?
  DATA_INTEGRITY: Does it maintain data consistency?
  ERROR_HANDLING: Does it handle failures gracefully?
  OBSERVABILITY: Can we monitor and debug it?
```

### 2. Coverage Audit

```
For each dimension:
  1. Read the phase PLAN.md for stated evaluation criteria
  2. Read the phase output artifacts (code, tests, configs)
  3. Search for explicit test coverage of this dimension
  4. Search for implicit coverage (tests that incidentally cover it)
  5. Search for gaps (areas with no test or verification)

  Score:
    COVERED:  Explicit tests exist and pass
    PARTIAL:  Some tests exist but not comprehensive
    MISSING:  No test coverage for this dimension
```

### 3. Audit Report

```
 EVAL COVERAGE -- Phase {N}: {name}
   
   Dimension          | Score     | Evidence
   
   Correctness        |  COVERED | 12 tests, all pass
   Edge Cases         |  PARTIAL | 3/5 edge cases tested
   Performance        |  MISSING | No perf tests
   Security           |  COVERED | Auth + input validation
   Compatibility      |  PARTIAL | Only Chrome tested
   Integration        |  COVERED | 8 integration tests
   Error Handling     |  PARTIAL | Happy path covered
   Data Integrity     |  COVERED | Constraint tests pass
   Observability      |  MISSING | No logging/metrics
   
   Overall: {covered}/{total} COVERED, {partial} PARTIAL, {missing} MISSING
   Coverage score: {percentage}%
```

### 4. Remediation Plan

```
For each MISSING dimension:
  1. Describe what's missing specifically
  2. Recommend specific tests to add
  3. Estimate effort to add coverage
  4. Mark as blocking or non-blocking for phase completion

For each PARTIAL dimension:
  1. Describe what's covered and what's not
  2. Recommend additional tests
  3. Estimate effort
  4. Mark as recommended or optional
```

### 5. Cross-Phase Coverage

```
When run across all phases:
  1. Identify dimensions consistently MISSING across phases
  2. Identify patterns (e.g., security always PARTIAL)
  3. Generate colony-wide remediation recommendations
  4. Feed into milestone-gap-planner if gaps are significant
```

## Key Patterns

- **Score honestly**: A generous PARTIAL is worse than an honest MISSING.
- **Evidence-based**: Every score references specific test files or gaps.
- **Actionable gaps**: Every MISSING item comes with a fix recommendation.
- **Cross-phase learning**: Patterns in gaps reveal systemic issues.

## Output Format

```
 EVAL | Phase {N}: {score}% coverage
    {covered} |  {partial} |  {missing}
   Remediation: {blocking} blocking, {recommended} recommended
   Report: .aether/phases/phase-{N}/EVAL-REVIEW.md
```

## Examples

**Phase audit:**
> "Phase 3 eval coverage: 62%. 5 COVERED, 2 PARTIAL, 2 MISSING. Blocking: performance tests missing (API latency not verified). Recommended: add edge case tests for null inputs."

**Colony-wide pattern:**
> "Cross-phase eval: security is PARTIAL in 4/6 phases. Colony-wide recommendation: add security testing to standard phase template."
