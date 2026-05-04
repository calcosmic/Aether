---
schema_version: "1.0"
id: watcher-verification-discipline-rubric
kind: rubric
category: rubrics
title: Watcher Verification Discipline Rubric
description: "Quality checks for watcher output: evidence freshness, severity classification, blocker identification, actionable findings."
output_types: [quality-gate, verification-report, review-output]
agent_roles: [queen, auditor, architect, watcher, gatekeeper]
task_types: [verify, review, gate, quality, evidence]
task_keywords: [watcher, evidence, severity, blocker, finding, verification, discipline, ladder, freshness, classification, scoring]
workflow_triggers: [continue, seal]
priority: high
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 4000
---

# Watcher Verification Discipline Rubric

This rubric defines quality criteria for evaluating watcher worker output.
Use it during continue gates, seal reviews, and quality assessments to
determine whether a watcher's verification meets discipline standards.

## For Beginners

After a builder finishes work, a watcher checks whether the work is correct.
But the watcher's own work also needs to be good. This rubric evaluates
whether the watcher did a thorough, honest, and useful verification job.
A good watcher provides specific evidence, correctly classifies problems,
and never misses a blocking issue.

## Scoring Criteria

Each criterion is scored on a three-point scale:

| Score | Meaning |
|-------|---------|
| Meets standard | The criterion is fully satisfied |
| Partial | The criterion is partially satisfied, with minor gaps |
| Does not meet | The criterion is not satisfied, with significant gaps |

### 1. Evidence Freshness (Weight: Critical)

**Criterion:** All evidence cited by the watcher is current and was actually
produced during this verification cycle, not carried over from previous runs.

**Meets standard:**
- Test output timestamps are from the current verification run
- Command output is from actual execution, not pasted from earlier
- File references point to current file state
- No "these tests were passing before" claims without re-running

**Partial:**
- Most evidence is fresh, but one or two references are stale
- Claims test passage but ran tests at the start, not after changes

**Does not meet:**
- Copy-pasted evidence from a previous verification
- Claims about file contents without reading the current version
- Test output that does not match the current code state

### 2. Severity Classification (Weight: High)

**Criterion:** Findings are classified with the correct severity level that
reflects their actual impact.

**Classification tiers (from the Queen execution policy):**
- `hard_block`: Critical, data-loss, or safety issues
- `soft_block`: Recoverable issues that a Fixer can likely resolve
- `advisory`: Warnings and non-blocking observations

**Meets standard:**
- Test failures classified as `soft_block` or higher (never advisory)
- Security findings classified as `hard_block`
- Cosmetic issues classified as `advisory` (not over-escalated)
- Each classification includes a clear rationale

**Partial:**
- One or two misclassified findings that are off by one level
- Classifications are present but rationale is vague

**Does not meet:**
- Test failures classified as `advisory`
- Security issues classified as `soft_block`
- No classification applied to findings
- All findings classified at the same level regardless of severity

### 3. Blocker Identification (Weight: Critical)

**Criterion:** All actual blockers are identified, and no non-blockers are
incorrectly flagged as blocking.

**Meets standard:**
- Every failing test is listed as a blocker
- Build failures are identified as blockers
- Race conditions are identified as blockers
- No false blockers (advisory findings not falsely escalated)
- Blocker list matches actual test and build output

**Partial:**
- Most blockers identified, but one missed
- One false blocker included

**Does not meet:**
- Missing a genuine blocker (failing test not reported)
- Claiming "all clear" when tests fail
- Multiple false blockers that waste Fixer time
- No blocker list at all

### 4. Actionable Findings (Weight: High)

**Criterion:** Each finding includes enough detail for a Fixer or builder to
act on it without additional investigation.

**Meets standard:**
- Findings include: file path, line or function, description, suggested fix
- Test failures include: test name, expected vs actual, relevant stack trace
- Each finding can be independently understood and addressed
- Findings are prioritized by impact

**Partial:**
- Findings include file and description but lack line-level detail
- Some findings are vague ("there is a problem in the build")
- Not prioritized by impact

**Does not meet:**
- "Tests fail" without specifying which tests
- "Code quality issues" without specifics
- Findings that require the reader to re-run the entire verification
- Copy-pasted error output without interpretation

### 5. Verification Shape (Weight: Medium)

**Criterion:** The watcher applied the correct verification approach for the
type of work being verified.

**Verification types:**
- **Unit tests:** `go test ./... -race` for Go code
- **Build verification:** `go build ./cmd/aether`
- **Lint:** `go vet ./...`
- **Source parity:** `aether source-check` for mirror changes
- **Integration:** Appropriate integration test commands

**Meets standard:**
- Ran all verification types appropriate for the changes
- Did not skip race detection for concurrent code
- Included source-check when mirrors were modified
- Verification commands match the project's standard set

**Partial:**
- Ran most verification types but skipped one
- Ran tests without race detection for concurrent changes

**Does not meet:**
- Only ran one verification type when multiple were appropriate
- Skipped verification entirely and relied on code review
- Ran tests for unrelated code but not the changed code

### 6. Honesty and Completeness (Weight: High)

**Criterion:** The watcher reports findings honestly, including negative
results, and does not overstate or understate the situation.

**Meets standard:**
- Reports both passing and failing verifications
- Acknowledges areas that were not tested
- Does not inflate severity to appear thorough
- Does not downplay findings to appear efficient
- Reports partial results when verification was incomplete

**Partial:**
- Generally honest but omits one minor finding
- Over-reports advisory items as if they were significant

**Does not meet:**
- Claims "all tests pass" when some were skipped
- Reports only positive findings
- Downplays genuine failures
- Fabricates evidence of testing

## Scoring Summary

| Criterion | Weight | Meets | Partial | Does Not Meet |
|-----------|--------|-------|---------|---------------|
| Evidence freshness | Critical | +4 | +2 | 0 |
| Severity classification | High | +3 | +1 | 0 |
| Blocker identification | Critical | +4 | +2 | 0 |
| Actionable findings | High | +3 | +1 | 0 |
| Verification shape | Medium | +2 | +1 | 0 |
| Honesty and completeness | High | +3 | +1 | 0 |

**Maximum score:** 19
**Passing threshold:** 13 (must include "meets" on evidence freshness and
blocker identification)
**Block threshold:** Below 9

Any watcher output scoring below the block threshold should trigger a re-run
with a different watcher or Queen escalation. A watcher that fabricates
evidence or misses genuine blockers should be flagged for circuit breaker
tracking.
