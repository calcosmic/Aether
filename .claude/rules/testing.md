# Testing

## Test Framework

- **Unit tests:** AVA (`npm test`)
- **Shell tests:** shellcheck + bats (in `tests/bash/`)
- **Integration tests:** `tests/integration/`
- **E2E tests:** `tests/e2e/`

## Test Structure

```javascript
test('description of what is being tested', t => {
  // Arrange
  const input = '...';

  // Act
  const result = functionUnderTest(input);

  // Assert
  t.is(result, expected);
});
```

## Coverage Expectations

- New code should include tests
- Bug fixes should include regression tests
- CLI commands should have integration tests

## Running Tests

```bash
npm test              # All unit tests
npm run test:bash     # Shell script tests
npm run lint:sync     # Verify command sync
```

## Test Isolation

- Each test should be independent
- Use temp directories for file operations
- Clean up resources in `teardown`
