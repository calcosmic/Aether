---
name: acceptance-test-generation
description: Use when acceptance criteria need unit, integration, or end-to-end tests generated from implementation context
type: colony
domains: [testing, quality-assurance, verification]
agent_roles: [watcher, probe, builder]
workflow_triggers: [build, continue]
task_keywords: [test, acceptance, criteria, coverage, e2e, unit]
priority: normal
version: "1.0"
---

# Acceptance Test Generation

## Purpose

Generate unit and end-to-end tests from acceptance criteria found in phase specifications. Follows the RED-GREEN verification cycle: write failing tests first, then confirm they pass against the implementation. Auto-detects the project's test framework and conventions so generated tests blend seamlessly with existing ones.

## When to Use

- After implementation is complete and acceptance criteria need verification
- When a phase spec or PLAN.md includes acceptance criteria that lack corresponding tests
- When expanding test coverage for existing functionality
- When the colony or a human requests test generation for specific features

## Instructions

### 1. Gather Context

Collect these inputs before generating any tests:

- **Acceptance criteria**: Read from the active phase's SPEC.md, PLAN.md, or a provided list
- **Source files**: Identify the implementation files the criteria relate to
- **Existing tests**: Scan the project for existing test files to understand conventions, structure, and framework
- **Project config**: Check package.json, pyproject.toml, Cargo.toml, go.mod, or equivalent for test dependencies

### 2. Auto-Detect Test Framework

Scan the project for test framework indicators:

| Indicator | Framework | Language |
|-----------|-----------|----------|
| `jest`, `vitest`, `@testing-library` in package.json | Jest / Vitest | TypeScript/JS |
| `pytest`, `unittest` imports | pytest / unittest | Python |
| `#[cfg(test)]`, `#[test]` | built-in | Rust |
| `_test.go` files | built-in | Go |
| `junit`, `mockito` in build.gradle/pom.xml | JUnit | Java |
| `RSpec.describe` patterns | RSpec | Ruby |
| `cypress`, `playwright` in package.json | Cypress / Playwright | E2E (JS) |
| `selenium` imports | Selenium | E2E (multi) |

If multiple frameworks exist (e.g., Jest for unit + Playwright for E2E), use each for its appropriate test type.

### 3. Map Acceptance Criteria to Test Cases

For each acceptance criterion, create one or more test cases:

1. **Parse the criterion** into a testable assertion. "Users can reset their password" becomes "Given a registered user, when they request a password reset, then a reset token is generated and emailed."
2. **Identify the test type**:
   - **Unit test**: Tests a single function, method, or class in isolation
   - **Integration test**: Tests interaction between two or more components
   - **E2E test**: Tests a full user flow through the system
3. **Define preconditions** (setup), **actions** (exercise), and **expected outcomes** (assert)
4. **Cover edge cases**: empty inputs, boundary values, error conditions, concurrent access where applicable

### 4. Generate Tests (RED Phase)

Write test files following project conventions:

- Place tests in the same directory structure the project uses (e.g., `__tests__/`, `tests/`, `*_test.go`)
- Mirror source file naming: `src/auth/login.ts` becomes `src/auth/__tests__/login.test.ts`
- Use the project's assertion style (expect, assert, should, etc.)
- Include descriptive test names that read as specifications

**Structure each test as:**

```
describe/context block naming the feature or criterion
  -> it/test block describing the specific scenario
    -> arrange: set up preconditions and test data
    -> act: call the function or simulate the user action
    -> assert: verify the expected outcome
```

**For E2E tests:**
- Use page object models if the project has them
- Test critical user journeys end-to-end
- Include visual assertions where the framework supports them
- Add retry logic for flaky interactions

### 5. Run Tests (RED Verification)

Execute the generated tests and confirm they **fail** with the expected assertion errors (not syntax errors or import failures). This proves the tests are valid and will catch regressions.

If a test passes immediately, either:
- The implementation already covers the criterion (note this)
- The test assertion is too weak (strengthen it)
- The test is testing the wrong thing (rewrite it)

### 6. Run Tests (GREEN Verification)

If the implementation is already complete, run all tests again to confirm they pass. If any fail:
- Analyze the failure message
- Determine if the test or the implementation is incorrect
- Fix whichever is wrong and re-run

### 7. Report Results

Output a summary:

```
Test Generation Report
======================
Framework detected: Jest (unit) + Playwright (E2E)
Acceptance criteria mapped: 8
Test files created: 4
  - src/auth/__tests__/login.test.ts (3 tests)
  - src/auth/__tests__/password-reset.test.ts (2 tests)
  - e2e/auth-flow.spec.ts (2 tests)
  - src/api/__tests__/users.test.ts (4 tests)
RED verification: All tests fail as expected
GREEN verification: 10/11 pass (1 pending fix in password-reset)
Coverage estimate: auth module 78% -> 94%
```

## Key Patterns

### Given-When-Then Mapping

Translate acceptance criteria into test structure:

| Criterion Language | Test Structure |
|--------------------|---------------|
| "Given X, when Y, then Z" | `describe('X')` -> `it('should Z when Y')` |
| "The system shall..." | `it('shall ...')` with assertions on system output |
| "If A then B, else C" | Two tests: `it('returns B when A')` and `it('returns C when not A')` |

### Mock Boundaries

When generating tests, identify external dependencies and mock them at natural boundaries:
- **Database**: Mock at the repository/data-access layer
- **HTTP calls**: Mock at the API client or service layer
- **File system**: Use in-memory fixtures or temp directories
- **Time**: Inject a clock or use framework time mocks

### Test Data Factories

Create reusable factory functions for test data rather than duplicating setup:

```typescript
function createTestUser(overrides = {}) {
  return { id: '1', email: 'test@example.com', role: 'user', ...overrides };
}
```

## Output Format

Produces test files in the project's test directory structure and prints a generation report to stdout. Does not modify source implementation files.

## Examples

### Example 1: Generate tests for a phase

```
Generate tests for phase 4 -- the user authentication phase
```

Reads phase 4's acceptance criteria, scans `src/auth/` for implementation, detects Jest as the framework, generates unit tests for login, registration, password reset, and an E2E test for the full sign-up flow.

### Example 2: Generate tests for specific criteria

```
Generate tests for these criteria:
- User can create an account with email and password
- User cannot create an account with duplicate email
- User receives confirmation email after registration
```

Maps each criterion to one or more test cases. The duplicate email check gets both a unit test (service layer) and an integration test (API response).

### Example 3: Generated test file

```typescript
import { describe, it, expect, beforeEach } from 'vitest';
import { UserService } from '../user-service';
import { createTestUser } from './helpers/factories';

describe('UserService', () => {
  let service: UserService;

  beforeEach(() => {
    service = new UserService(mockRepository, mockEmailSender);
  });

  it('creates an account with valid email and password', async () => {
    const user = await service.create('new@example.com', 'SecureP@ss1');
    expect(user.id).toBeDefined();
    expect(user.email).toBe('new@example.com');
  });

  it('rejects duplicate email registration', async () => {
    await service.create('dup@example.com', 'pass123');
    await expect(
      service.create('dup@example.com', 'pass456')
    ).rejects.toThrow('Email already registered');
  });

  it('sends confirmation email after registration', async () => {
    await service.create('new@example.com', 'SecureP@ss1');
    expect(mockEmailSender.send).toHaveBeenCalledWith(
      expect.objectContaining({ to: 'new@example.com', type: 'confirmation' })
    );
  });
});
```
