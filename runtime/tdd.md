# Test-Driven Development Discipline

## The Iron Law

```
NO PRODUCTION CODE WITHOUT A FAILING TEST FIRST
```

If you didn't watch the test fail, you don't know if it tests the right thing.

Write code before the test? Delete it. Start over. No exceptions.

## When to Use

**Always:**
- New features
- Bug fixes
- Refactoring
- Behavior changes

**The only exceptions (must be explicit):**
- Throwaway prototypes (will be deleted)
- Generated/config code
- User explicitly opts out

Thinking "skip TDD just this once"? Stop. That's rationalization.

## Red-Green-Refactor Cycle

```
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│   RED ────────► VERIFY RED ────────► GREEN                  │
│   Write         (must fail           Write minimal          │
│   failing       correctly)           code to pass           │
│   test                                    │                 │
│                                           ▼                 │
│                                      VERIFY GREEN           │
│                                      (must pass,            │
│                                       all green)            │
│                                           │                 │
│   ◄───────────── REFACTOR ◄───────────────┘                 │
│   Next test      Clean up                                   │
│                  (stay green)                               │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### RED - Write Failing Test

Write ONE minimal test showing desired behavior.

```typescript
// GOOD: Clear name, tests real behavior, one thing
test('rejects empty email with error message', async () => {
  const result = await submitForm({ email: '' });
  expect(result.error).toBe('Email required');
});

// BAD: Vague name, tests mock not code
test('validation works', async () => {
  const mock = jest.fn();
  // ...
});
```

**Requirements:**
- One behavior per test
- Clear descriptive name
- Real code (mocks only if unavoidable)

### VERIFY RED - Watch It Fail (MANDATORY)

```bash
npm test path/to/test.test.ts
```

Confirm:
- Test **fails** (not errors)
- Failure message is expected
- Fails because feature missing (not typos)

**Test passes immediately?** You're testing existing behavior. Fix test.

**Test errors?** Fix error, re-run until it fails correctly.

### GREEN - Minimal Code

Write **simplest** code to pass the test.

```typescript
// GOOD: Just enough to pass
function validateEmail(email: string): string | null {
  if (!email?.trim()) return 'Email required';
  return null;
}

// BAD: Over-engineered (YAGNI)
function validateEmail(
  email: string,
  options?: { customMessage?: string; allowEmpty?: boolean }
): ValidationResult { /* ... */ }
```

Don't add features beyond the test. Don't refactor other code. Don't "improve."

### VERIFY GREEN - Watch It Pass (MANDATORY)

```bash
npm test path/to/test.test.ts
```

Confirm:
- Test passes
- Other tests still pass
- Output pristine (no errors, warnings)

**Test fails?** Fix code, not test.
**Other tests fail?** Fix now.

### REFACTOR - Clean Up

Only after green:
- Remove duplication
- Improve names
- Extract helpers

**Keep tests green. Don't add behavior.**

### REPEAT

Next failing test for next behavior.

## Test Types

### Unit Tests
- Individual functions
- Component logic
- Pure functions
- < 50ms each

### Integration Tests
- API endpoints
- Database operations
- Service interactions

### E2E Tests
- Critical user flows
- Complete workflows
- Browser automation

## Coverage Requirements

- Minimum 80% coverage target
- All edge cases covered
- Error scenarios tested
- Boundary conditions verified

Verify with:
```bash
npm run test:coverage
```

## Common Rationalizations

| Excuse | Reality |
|--------|---------|
| "Too simple to test" | Simple code breaks. Test takes 30 seconds. |
| "I'll test after" | Tests passing immediately prove nothing. |
| "Already manually tested" | Ad-hoc ≠ systematic. No record, can't re-run. |
| "Deleting X hours is wasteful" | Sunk cost fallacy. Unverified code is debt. |
| "Need to explore first" | Fine. Throw away exploration, start with TDD. |
| "TDD will slow me down" | TDD faster than debugging. |
| "Keep as reference" | You'll adapt it. That's testing after. Delete. |

## Red Flags - STOP and Start Over

If you catch yourself:
- Writing code before test
- Test passes immediately (didn't fail first)
- Can't explain why test failed
- Rationalizing "just this once"
- "I already manually tested it"
- "Keep as reference"
- "This is different because..."

**ALL mean: Delete code. Start over with TDD.**

## Testing Anti-Patterns

| Anti-Pattern | Problem | Fix |
|--------------|---------|-----|
| Testing implementation | Breaks on refactor | Test behavior/output |
| Brittle selectors | `.css-class-xyz` | `[data-testid="x"]` |
| Test interdependence | Tests depend on order | Each test sets up own data |
| Mock everything | Tests mock, not code | Real code, inject deps |
| One giant test | Can't isolate failure | One behavior per test |

## TDD Report Format

When reporting TDD work:

```
🧪 TDD Report
=============

Feature: {what was implemented}

Cycle 1:
  RED: test('rejects empty email')
  Verified: ✓ Failed with "undefined is not 'Email required'"
  GREEN: Added email validation
  Verified: ✓ Passes

Cycle 2:
  RED: test('rejects invalid format')
  Verified: ✓ Failed with expected message
  GREEN: Added regex check
  Verified: ✓ Passes

Coverage: 87% (target: 80%)
All tests: 47/47 passing
```

## Bug Fix with TDD

1. Write failing test reproducing bug
2. Verify it fails (proves bug exists)
3. Fix bug
4. Verify test passes
5. Test now prevents regression

**Never fix bugs without a test.**

## Integration with Debugging

When debugging reveals root cause:
1. Write test that would have caught it
2. Verify test fails
3. Apply fix
4. Verify test passes

This ensures the bug never returns.

## Verification Checklist

Before marking work complete:

- [ ] Every new function has a test
- [ ] Watched each test fail before implementing
- [ ] Each test failed for expected reason
- [ ] Wrote minimal code to pass
- [ ] All tests pass
- [ ] Coverage meets threshold
- [ ] No skipped/disabled tests

Can't check all boxes? You skipped TDD. Start over.
