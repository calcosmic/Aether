---
name: testing
description: Use when the project needs automated testing with Jest, Vitest, pytest, or similar frameworks
type: domain
domains: [testing, quality, tdd]
agent_roles: [builder]
detect_files: ["*.test.*", "*.spec.*"]
detect_packages: ["jest", "vitest", "pytest"]
priority: normal
version: "1.0"
---

# Testing Best Practices

## Test Structure

Follow the Arrange-Act-Assert pattern: set up the conditions, perform the action, check the result. Each test should verify one behavior. If a test name contains "and", it probably tests two things -- split it.

Name tests to describe behavior, not implementation: "returns empty array when no items match filter" not "test filterItems function". Good test names serve as documentation.

## Test-Driven Development

Write the test before the implementation. Start with the simplest failing test, write minimal code to make it pass, then refactor. This cycle (Red-Green-Refactor) prevents over-engineering and ensures every line of production code is covered.

When fixing bugs, write a test that reproduces the bug first. Then fix the code. The test proves the bug existed and prevents regression.

## What to Test

Test behavior, not implementation. Test public interfaces, not private methods. If you refactor internals without changing behavior, tests should still pass. Tests that break during refactoring are testing the wrong thing.

Focus coverage on: business logic, data transformations, error handling paths, edge cases (empty inputs, boundary values, null/undefined). Skip testing: framework boilerplate, simple getters/setters, third-party library behavior.

## Mocking

Mock external dependencies (APIs, databases, file system), not your own code. Over-mocking creates tests that pass even when the real system is broken. Use the lightest mock possible -- spy on a method rather than replacing an entire module.

Reset mocks between tests to prevent state leakage. In Jest/Vitest, use `beforeEach(() => { vi.clearAllMocks() })`. In pytest, fixtures handle cleanup automatically.

## Test Performance

Fast tests get run often. Slow tests get skipped. Keep unit tests under 100ms each. Isolate slow integration tests into a separate suite that runs less frequently.

Avoid `sleep` or timeouts in tests. Use fake timers (`vi.useFakeTimers()`) for time-dependent code. Wait for conditions with polling utilities, not fixed delays.

## Test Organization

Colocate tests with source files (`Button.tsx` next to `Button.test.tsx`) or use a parallel `__tests__/` directory. Be consistent across the project.

Use `describe` blocks to group related tests. Use `it` or `test` for individual cases. Setup shared across a `describe` block goes in `beforeEach`, not duplicated in each test.

## Continuous Integration

Tests must pass before merging. Run the full suite in CI on every pull request. Fail the build on test failures -- never merge with known failures. Aim for meaningful coverage (80%+) over line-count metrics.
