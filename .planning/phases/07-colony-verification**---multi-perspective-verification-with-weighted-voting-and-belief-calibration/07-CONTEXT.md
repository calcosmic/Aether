# Phase 7: Colony Verification - Context

**Gathered:** 2026-02-01
**Status:** Ready for planning

## Phase Boundary

Multi-perspective verification system where Worker Ants spawn specialized Watcher perspectives (Security, Performance, Quality, Test-coverage) in parallel. Each Watcher casts weighted votes (APPROVE/REJECT) based on historical reliability. Supermajority (67%) required for approval. Votes update meta-learning confidence scores, creating a feedback loop where the colony learns which Watchers are reliable for which types of verification.

---

## Implementation Decisions

### Watcher Specialization

Four Watcher castes spawn in parallel, each with domain-specific verification logic:

**Security Watcher**
- Checks: OWASP Top 10 vulnerabilities, injection attacks (SQL, NoSQL, command, LDAP), XSS vectors, authentication/authorization issues, input validation, sensitive data exposure
- Depth: Static analysis + pattern matching (no dynamic execution)
- Priority: Critical severity (auth bypass, injection) > High (XSS, sensitive data) > Medium (input validation gaps)

**Performance Watcher**
- Checks: Time complexity analysis (big-O notation for algorithms), I/O operations (database queries N+1 problems), resource usage (memory leaks, file handles), blocking operations
- Depth: Algorithmic analysis, not benchmarking (no actual execution)
- Priority: Critical (infinite loops, O(n²) where n is large) > High (N+1 queries) > Medium (unoptimized loops)

**Quality Watcher**
- Checks: Code maintainability (function length, cyclomatic complexity), readability (naming conventions, magic numbers), code smell patterns (duplicate code, long parameter lists), adherence to project conventions
- Depth: Linting rules + heuristic analysis
- Priority: Critical (unmaintainable code > 10 complexity) > High (duplicates > 5 lines) > Medium (naming inconsistencies)

**Test-Coverage Watcher**
- Checks: Test completeness (missing test paths), edge cases (boundary conditions, null/empty handling), assertion quality (meaningful assertions, not just "no error"), coverage thresholds
- Depth: Static analysis of test files vs implementation
- Priority: Critical (untested critical paths) > High (missing edge cases) > Medium (low assertion density)

### Weighting & Belief Calibration

**Starting weights:** All Watchers start at equal weight (1.0). No caste-based bias initially.

**Vote impact on weights:**
- Correct APPROVE (verified by successful phase outcome): weight +0.1
- Correct REJECT (verified by fix addressing identified issues): weight +0.15
- Incorrect APPROVE (issues found later): weight -0.2
- Incorrect REJECT (false positive, approved after review): weight -0.1
- Weight range: 0.1 (minimum) to 3.0 (maximum)

**Domain expertise bonus:** When a Watcher's domain matches the issue type, its weight is doubled for that vote:
- Security Watcher weight ×2 for security issues
- Performance Watcher weight ×2 for performance issues
- Quality Watcher weight ×2 for quality issues
- Test-Coverage Watcher weight ×2 for testing issues

### Voting & Aggregation

**Supermajority requirement:** 67% of weighted votes must be APPROVE. With 4 Watchers:
- 4/4 APPROVE = 100% → APPROVED
- 3/4 APPROVE = 75% → APPROVED (above threshold)
- 2/4 APPROVE = 50% → REJECTED (below threshold)
- 1/4 or 0/4 APPROVE = REJECTED

**Tie-breaking:** With 4 Watchers, a 2-2 tie is 50% which is below 67% threshold → REJECTED. Queen can manually override via `/ant:continue` if needed.

**Issue aggregation:**
- Dedupe: Same issue reported by multiple Watchers appears once with "Multiple Watchers" tag
- Prioritize: Issues sorted by severity (Critical > High > Medium) then by weight sum (higher weight Watchers' issues first)
- All included: All issues from all Watchers included in report, no filtering
- Veto power: One Critical severity REJECT from any Watcher blocks approval regardless of other APPROVEs (Critical veto)

**Vote combination:**
- Each Watcher returns: `{decision: "APPROVE"|"REJECT", issues: [{severity, category, description}]}`
- Colony aggregates: Unified issue list with deduping, severity prioritization, Watcher attribution
- Final decision: Based on weighted vote percentage AND Critical veto check

### Meta-Learning Integration

**Connection to Phase 8 (Bayesian confidence scoring):**

Verification votes directly update spawn confidence in `meta_learning.specialist_confidence`:

- Successful verification (APPROVED, no issues found later): `specialist_confidence[specialist_type][task_type] += 0.1`
- Failed verification (REJECTED, issues found): `specialist_confidence[specialist_type][task_type] -= 0.2`
- Verification corrections (issues found after approval): `specialist_confidence -= 0.15` (penalty for missing issues)

**Task type mapping:**
- Security Watcher votes → `security_verification` task type confidence
- Performance Watcher votes → `performance_verification` task type confidence
- Quality Watcher votes → `quality_verification` task type confidence
- Test-Coverage Watcher votes → `test_coverage_verification` task type confidence

**Specialist selection influence:**
Phase 8's `get_specialist_confidence()` uses verification confidence when recommending which specialist to spawn for verification tasks. Higher confidence in a Watcher → higher likelihood of spawning that Watcher for future verification.

**Learning feedback loop:**
1. Worker Ant spawns specialist (Phase 6)
2. Colony spawns 4 Watchers (Phase 7)
3. Watchers vote, update their own weights (Phase 7)
4. Vote outcomes update specialist spawn confidence (Phase 8)
5. Phase 8 uses both Watcher weights AND specialist confidence for recommendations
6. Colony learns which combinations work best

### Claude's Discretion

- Exact heuristic thresholds (e.g., cyclomatic complexity > 10 = Critical, magic numbers > 3 occurrences)
- Specific linting rules for Quality Watcher
- Exact OWASP coverage depth for Security Watcher
- Time complexity detection patterns (manual annotation vs automatic analysis)
- Test coverage minimum thresholds (e.g., 80% branch coverage)
- Weight adjustment increments (current values are suggestions, can be tuned)

---

## Specific Ideas

No specific requirements — standard multi-perspective verification patterns. Inspiration from:
- Ensemble methods in machine learning (weighted voting)
- Belief calibration in distributed systems
- Code review practices (multiple reviewers, domain expertise)

---

## Deferred Ideas

None — discussion stayed within phase scope.

---

*Phase: 07-colony-verification*
*Context gathered: 2026-02-01*
