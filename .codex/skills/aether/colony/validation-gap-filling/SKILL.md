---
name: validation-gap-filling
description: Use when implementation exists but tests or validation evidence are missing or incomplete
type: colony
domains: [testing, validation, quality]
agent_roles: [builder, probe]
workflow_triggers: [build, continue]
task_keywords: [validation gap, missing tests, coverage, test gap, unverified]
priority: normal
version: "1.0"
---

# Validation Gap Filling

## Purpose

Completed phases often have implementation but insufficient tests. This skill analyzes the implementation and UAT criteria, identifies what's not tested, and auto-generates tests to fill the gaps. The colony doesn't just build -- it validates.

## When to Use

- User says "add missing tests" or "fill test gaps"
- After eval-coverage-reviewer identifies MISSING dimensions
- Before shipping to boost test coverage
- Phase has implementation but low test coverage

## Instructions

### 1. Gap Identification

```
1. Read phase PLAN.md for UAT criteria and acceptance tests
2. Read phase eval-coverage audit (if exists)
3. Scan implementation files for exported functions/endpoints
4. Scan existing test files for what's already covered
5. Diff: implementation surface vs test coverage = gaps
```

### 2. Test Generation Strategy

```
For each gap:
  UNIT_TEST:
    - Function/method has no direct test
    - Generate: input/output test, edge case test, error test

  INTEGRATION_TEST:
    - Module interaction not tested
    - Generate: happy path, error path, boundary conditions

  E2E_TEST:
    - User flow not tested
    - Generate: complete flow test with setup/teardown

  PROPERTY_TEST:
    - Invariants should hold for all inputs
    - Generate: property-based tests for core logic
```

### 3. Test Generation Protocol

```
For each missing test:
  1. Read the implementation code thoroughly
  2. Identify the contract (inputs, outputs, side effects)
  3. Generate test following existing test patterns in the repo
  4. Use the project's test framework (detect from existing tests)
  5. Follow naming convention of existing tests
  6. Include setup, execution, assertion, cleanup
  7. Test should be independent (no test ordering dependency)
```

### 4. Test Quality Standards

```
Every generated test must:
   Have a clear, descriptive name
   Test one thing (single responsibility)
   Be deterministic (same input -> same result)
   Be independent (no dependency on other tests)
   Clean up after itself (no leftover state)
   Follow project test conventions
   Cover the specific gap it was created for
```

### 5. Generation Report

```
 VALIDATION GAP FILL -- Phase {N}
   
   Gaps identified: {count}
   Tests generated: {count}
   Types:
     Unit: {count} | Integration: {count} | E2E: {count}
   
   Files created:
    {test_file_1} -- {tests_in_file} tests
    {test_file_2} -- {tests_in_file} tests
   
   Coverage improvement: {before}% -> {after}%
   All tests pass: {yes/no}
```

### 6. Verification

```
After generation:
  1. Run all new tests -- verify they pass
  2. Run full test suite -- verify no regressions
  3. Check coverage improvement
  4. If any test fails: fix the test (not the implementation)
  5. Commit tests with message referencing the gap
```

## Key Patterns

- **Follow existing patterns**: New tests should look like they were always there.
- **Fix the test, not the code**: If a generated test reveals a bug, that's a finding -- don't silently patch it.
- **Independent tests**: No test should depend on another test running first.
- **Coverage is a side effect**: The goal is correct behavior, not a coverage percentage.

## Output Format

```
 FILLED | Phase {N}: {tests_added} tests added
   Coverage: {before}% -> {after}%
   Files: {list of new test files}
   All passing: {yes/no}
```

## Examples

**Fill gaps:**
> "Phase 3 had 7 untested functions. Generated 14 tests (7 unit, 4 integration, 3 edge case). Coverage: 52% -> 89%. All tests pass."

**Framework detection:**
> "Detected Jest test framework. Following existing patterns from auth.test.ts. Generated 5 tests for payment module in payment.test.ts. Running suite... all pass."
