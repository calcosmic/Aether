# Phase 51: Recovery Verification - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-25
**Phase:** 51-recovery-verification
**Areas discussed:** Test structure, seeding strategy, verification depth, compound scenario scope

---

## Test Structure

| Option | Description | Selected |
|--------|-------------|----------|
| One function per scenario | Follow existing e2e_regression_test.go pattern | ✓ |
| Table-driven tests | Parameterize the 7 states into one test function | |
| Sub-tests (t.Run) | Group related scenarios under one function | |

**User's choice:** Claude's discretion (user said "you decide")
**Notes:** Chose one function per scenario — matches existing pattern, each scenario has unique setup that's awkward to parameterize. New file `cmd/e2e_recovery_test.go`.

---

## Seeding Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Helper functions | Per-state seeders that write fixtures to temp dirs | ✓ |
| Hand-crafted fixture files | Static JSON files loaded from testdata/ | |
| Reuse unit test fixtures | Call existing test helpers from recover_test.go | |

**User's choice:** Claude's discretion
**Notes:** Helper functions chosen — most readable and maintainable for E2E tests where each state needs different broken file combinations.

---

## Verification Depth

| Option | Description | Selected |
|--------|-------------|----------|
| Exit code only | Test that recover exits 0/1 correctly | |
| Exit code + re-scan | Verify re-scan returns empty after repair | ✓ |
| Full state assertion | Exit code + re-scan + compare state file contents | |

**User's choice:** Claude's discretion
**Notes:** Chose exit code + re-scan + key state assertions. Goes beyond "didn't crash" to prove state is actually clean, without being brittle on exact JSON comparison.

---

## Compound Scenario Scope

| Option | Description | Selected |
|--------|-------------|----------|
| All 7 at once | Kitchen-sink test with every state | |
| 2-3 realistic combos | Group by safety (safe vs destructive) | ✓ |
| One compound only | Single test with 3-4 states | |

**User's choice:** Claude's discretion
**Notes:** Two compound scenarios: all 5 safe states together, both destructive states together. Mirrors real-world clustering patterns.

---

## Claude's Discretion

- All areas deferred to Claude's discretion
- Test file: `cmd/e2e_recovery_test.go`
- Healthy colony test is highest priority (false positives erode trust)

## Deferred Ideas

None — discussion stayed within phase scope.
