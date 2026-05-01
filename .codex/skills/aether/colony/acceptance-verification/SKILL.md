---
name: acceptance-verification
description: Use when delivered functionality needs acceptance-criteria verification before a phase advances
type: colony
domains: [verification, acceptance-testing, quality-assurance]
agent_roles: [watcher, auditor]
workflow_triggers: [continue]
task_keywords: [acceptance, uat, criterion, criteria, verification, sign-off]
priority: normal
version: "1.0"
---

# Acceptance Verification

## Purpose

Conversational User Acceptance Testing that walks through each acceptance criterion interactively, confirms pass/fail status, and auto-diagnoses issues when verification fails. Produces a VERIFICATION.md artifact documenting the full UAT session with evidence for each criterion.

## When to Use

- After a phase's implementation is complete and ready for acceptance sign-off
- Before marking a phase as done in the colony workflow
- When re-verifying a phase after bug fixes have been applied
- When a human reviewer wants guided verification of delivered functionality

## Instructions

### 1. Load Acceptance Criteria

Identify and load the acceptance criteria for the target phase:

- Read from SPEC.md, PLAN.md, or the phase manifest
- Parse each criterion into a discrete, testable statement
- Number them sequentially (AC-01, AC-02, AC-03, ...)
- If criteria reference user stories, decompose into concrete testable behaviors

If no formal criteria exist, infer them from:
- The phase description and goals in ROADMAP.md
- The PLAN.md task list and deliverables
- Any user-facing behavior described in the implementation

### 2. Prepare Verification Environment

Before testing, ensure the system is in a verifiable state:

- Confirm the application builds and starts without errors
- Check that any required services (database, API, etc.) are running
- Identify the entry points for each criterion (URLs, CLI commands, API endpoints)
- Prepare test data or identify existing data that satisfies preconditions

If the environment cannot be prepared, report this as a blocker with specific details on what failed and how to resolve it.

### 3. Walk Through Each Criterion

For each acceptance criterion, perform this verification loop:

**A. Present the criterion**
State the criterion clearly, including any preconditions and expected outcomes.

**B. Attempt verification**
Execute the verification using the most appropriate method:
- **Manual walkthrough**: Describe the steps a user would take, simulate mentally or via code inspection
- **Automated check**: Run existing tests, scripts, or tooling that covers this criterion
- **Code inspection**: Read the implementation to confirm the logic satisfies the criterion
- **Live test**: If the app is running, interact with it to confirm the behavior

**C. Record the result**
Classify as one of:

| Status | Meaning |
|--------|---------|
| PASS | Criterion fully satisfied with evidence |
| FAIL | Criterion not met -- specific failure described |
| PARTIAL | Some aspects met, others not -- details provided |
| BLOCKED | Cannot verify due to environment or dependency issues |
| SKIPPED | Not applicable in current context (with justification) |

**D. If FAIL or PARTIAL -- auto-diagnose**
When a criterion fails, immediately diagnose the root cause:

1. **Trace the failure path**: Follow the code path from entry point to failure point
2. **Identify the defect**: Pinpoint the specific code location and logic error
3. **Classify the defect**:
   - Implementation bug (code doesn't match intent)
   - Missing implementation (feature partially built)
   - Configuration issue (wrong env, missing config)
   - Integration mismatch (components disagree on contract)
4. **Suggest a fix**: Provide a concrete remediation -- code change, config update, or additional implementation needed
5. **Assess impact**: Note whether this failure blocks other criteria

### 4. Handle Interactive Verification

When running in interactive mode, present each criterion to the user for hands-on verification:

```
--- AC-03: User receives email confirmation after registration ---

Precondition: A valid email server is configured
Steps:
1. Register a new account with email "test@example.com"
2. Check the email inbox for the test account
3. Confirm the email contains a verification link

Expected: Verification email arrives within 60 seconds

Did this pass? (pass/fail/partial/blocked/skip) [describe what you observed]
```

Collect the user's response and record it. If they report a failure, probe for details:
- What did you see instead of the expected behavior?
- Were there any error messages?
- Can you reproduce it consistently?

### 5. Compile VERIFICATION.md

After all criteria are verified, produce the verification artifact:

```markdown
# UAT Verification -- Phase {N}: {Phase Name}
**Date:** {ISO date}
**Verifier:** {agent name or human name}
**Result:** {PASS_COUNT}/{TOTAL_COUNT} criteria passed

## Summary
{2-3 sentence overall assessment of phase readiness}

## Criteria Results

| ID | Criterion | Status | Evidence |
|----|-----------|--------|----------|
| AC-01 | ... | PASS | ... |
| AC-02 | ... | FAIL | ... |

## Detailed Results

### AC-01: {criterion text}
**Status:** PASS
**Evidence:** Confirmed by running `pytest tests/test_registration.py::test_new_user` -- test passes. Inspected `src/auth/register.ts:42-58` confirms email dispatch logic.

### AC-02: {criterion text}
**Status:** FAIL
**Evidence:** Registration returns 500 when email service is unreachable. No fallback or retry logic exists.
**Diagnosis:** Missing error handling in `src/auth/register.ts:47` -- the `sendEmail` call is not wrapped in try/catch.
**Suggested fix:** Add retry with exponential backoff and graceful degradation (queue email for later delivery).
**Impact:** Blocks AC-03 (email confirmation cannot be verified until this is fixed).

## Outstanding Issues
{List of unresolved FAIL/PARTIAL items with priority ordering}

## Recommendation
{READY / READY WITH CAVEATS / NOT READY -- with justification}
```

### 6. Re-verification After Fixes

If issues were found and fixed, re-run verification only for the failed criteria. Update VERIFICATION.md with the re-verification results, keeping the original findings for audit trail.

## Key Patterns

### Verification Methods by Criterion Type

| Criterion Type | Verification Method |
|---------------|-------------------|
| "User can X" | Walk through the user flow, confirm each step succeeds |
| "System responds with Y" | Trigger the condition, inspect the response |
| "Error Z is handled gracefully" | Force the error condition, confirm graceful handling |
| "Performance: X under Y seconds" | Time the operation, compare against threshold |
| "No regression in X" | Run existing test suite for X, confirm all pass |
| "Data is persisted correctly" | Create data, restart system, confirm data survives |

### Diagnosis Depth Levels

When a failure is detected, diagnose at increasing depth:

1. **Surface**: What the user sees (error message, wrong behavior)
2. **Logic**: Which code path was taken and where it diverged from expected
3. **Root cause**: Why the logic is wrong (misunderstood requirement, edge case, typo)
4. **Systemic**: Whether this indicates a pattern that may affect other criteria

### Pass Confidence Levels

| Level | Meaning | When to Use |
|-------|---------|-------------|
| Confirmed | Directly observed working | Live test passed, user confirmed |
| Inspected | Code logic verified correct | Read the code, no live test possible |
| Tested | Automated test covers it | Existing test suite passes for this criterion |
| Assumed | Likely works but not verified | Low-risk criterion, similar to verified ones |

## Output Format

Produces `VERIFICATION.md` in the phase directory or specified output path.

## Examples

### Example 1: Verify a phase

```
Verify phase 2 -- user management
```

Loads acceptance criteria from the phase 2 plan, walks through each criterion, runs existing tests where available, inspects code for the rest, produces VERIFICATION.md with full results.

### Example 2: Re-verify after fixes

```
Re-verify AC-03 and AC-05 from phase 2 after bug fixes
```

Only re-checks the two previously failed criteria, appends re-verification results to the existing VERIFICATION.md.

### Example 3: VERIFICATION.md excerpt

```markdown
## Detailed Results

### AC-01: Users can register with email and password
**Status:** PASS (Confirmed)
**Evidence:** Created test account via POST /api/register with valid payload.
Received 201 response with user object. Confirmed user record exists in database.

### AC-02: Duplicate email registration returns 409 Conflict
**Status:** PASS (Tested)
**Evidence:** Existing test `tests/api/auth.test.ts:23` -- "rejects duplicate email".
Ran `npm test -- --grep "duplicate"` -- 1 test passed.

### AC-03: Password must be at least 8 characters
**Status:** FAIL
**Evidence:** POST /api/register with password "abc" returns 201 instead of 400.
**Diagnosis:** Validation middleware in `src/middleware/validate.ts:15` checks `password`
field existence but not length. Missing `minLength` validator.
**Suggested fix:** Add `.isLength({ min: 8 })` to password validation chain.
**Impact:** Low -- does not block other criteria.
```
