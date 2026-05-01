---
name: code-review
description: Use when changed source files need review for bugs, security, code quality, or maintainability risks
type: colony
domains: [code-quality, security, maintainability, best-practices]
agent_roles: [watcher, auditor, probe]
workflow_triggers: [continue]
task_keywords: [review, audit, quality, bug, maintainability]
priority: normal
version: "1.0"
---

# Code Review

## Purpose

Multi-depth code review with severity classification. Scans source files changed during a phase or across specified paths, identifying bugs, security vulnerabilities, style violations, and maintainability concerns. Every finding is classified by severity so maintainers can triage efficiently.

## When to Use

- After completing a phase build and before marking it done
- Before merging a branch or creating a pull request
- On an ad-hoc basis when you want a quality gate on changed files
- When the colony or a human requests a review of specific files or directories

## Instructions

### 1. Determine Review Scope

Identify which files to review. Accept any of these inputs:

- A phase number (review all files changed since that phase started)
- A git diff range (e.g. `main..feature-branch`)
- An explicit list of file paths
- A directory glob pattern

If no scope is given, default to all uncommitted changes (staged + unstaged + untracked).

### Remediation Handoff

When findings are fixable, classify them before recommending action:

- Auto-fixable: narrow, mechanical, covered by tests, and low blast radius.
- Needs builder: requires design judgment, cross-file coordination, or new behavior.
- Needs human decision: changes product behavior, public APIs, migrations, or security posture.

Group fixable findings into atomic units so each can be implemented, tested, and reverted independently.

### 2. Select Review Depth

The caller may specify a depth level. If not specified, use **standard**.

| Depth | What It Checks | Typical Use |
|-------|---------------|-------------|
| quick | Syntax errors, obvious bugs, critical security issues | Pre-commit hook, rapid feedback |
| standard | quick + code smells, naming, missing error handling, test coverage gaps | Phase gate, PR review |
| deep | standard + performance hotspots, concurrency issues, dependency risks, architectural drift | Release readiness, security-sensitive code |

### 3. Run the Review

For each file in scope, perform these checks in order:

**A. Static analysis layer**
- Parse the file's AST (mentally model the structure)
- Flag unreachable code, unused imports/variables, duplicate logic
- Check type consistency (if language is typed)

**B. Bug detection layer**
- Off-by-one errors, null/undefined dereferences, incorrect conditionals
- Resource leaks (unclosed files, connections, handles)
- Race conditions in concurrent code
- Incorrect error propagation or swallowed exceptions

**C. Security layer (quick depth stops here)**
- Input validation gaps (injection, XSS, SSRF, path traversal)
- Hardcoded secrets, credentials, or API keys
- Insecure defaults (disabled TLS, weak hashing, permissive CORS)
- Missing authentication or authorization checks on sensitive operations

**D. Quality layer (standard depth stops here)**
- Function complexity (cyclomatic > 10 is a warning)
- Naming clarity and consistency with project conventions
- Missing or misleading documentation on public APIs
- Dead code or commented-out code blocks

**E. Architecture layer (deep only)**
- Circular dependencies between modules
- Tight coupling that hinders testability
- Violations of the project's stated architecture patterns
- Performance anti-patterns (N+1 queries, unnecessary allocations, unbounded growth)

### 4. Classify Every Finding

Assign one of three severities:

| Severity | Meaning | Action Required |
|----------|---------|-----------------|
| critical | Will cause failures, data loss, or security breach in production | Must fix before merge |
| warning | Likely to cause problems or degrades maintainability | Should fix soon |
| info | Style preference, minor improvement, or educational note | Optional |

For each finding include:
- **File and line range** where the issue lives
- **Rule ID** (e.g. `SEC-001`, `BUG-003`, `QUAL-012`) from the pattern catalog below
- **Severity** classification
- **Description** of the issue in plain language
- **Suggested fix** -- either a code snippet or a clear remediation step

### 5. Produce REVIEW.md

Write a structured review document:

```markdown
# Code Review -- {scope description}
**Depth:** {quick|standard|deep}
**Date:** {ISO date}
**Files Reviewed:** {count}
**Findings:** {critical count} critical, {warning count} warning, {info count} info

## Summary
{1-3 sentence overall assessment}

## Findings

### Critical
{list or "None found"}

### Warning
{list or "None found"}

### Info
{list or "None found"}

## Metrics
| Metric | Value |
|--------|-------|
| Files reviewed | N |
| Lines scanned | N |
| Findings per 100 lines | N |
| Critical findings | N |
```

## Key Patterns

### Rule Catalog

| ID | Category | Description |
|----|----------|-------------|
| SEC-001 | Security | Hardcoded secret or credential |
| SEC-002 | Security | Missing input validation |
| SEC-003 | Security | Insecure crypto or hashing |
| SEC-004 | Security | SQL/NoSQL injection vector |
| SEC-005 | Security | Overly permissive access control |
| BUG-001 | Bug | Null/undefined dereference risk |
| BUG-002 | Bug | Off-by-one or boundary error |
| BUG-003 | Bug | Resource leak |
| BUG-004 | Bug | Swallowed exception |
| BUG-005 | Bug | Race condition |
| QUAL-001 | Quality | High cyclomatic complexity |
| QUAL-002 | Quality | Unused import or variable |
| QUAL-003 | Quality | Missing error handling |
| QUAL-004 | Quality | Dead or commented-out code |
| QUAL-005 | Quality | Inconsistent naming |
| PERF-001 | Performance | N+1 query pattern |
| PERF-002 | Performance | Unbounded collection growth |
| ARCH-001 | Architecture | Circular dependency |
| ARCH-002 | Architecture | Layer violation |

### Language-Specific Adjustments

- **TypeScript/JavaScript**: Check for `any` type abuse, missing null checks, unhandled promise rejections
- **Python**: Check for mutable default arguments, bare `except`, missing type hints on public functions
- **Rust**: Check for `unwrap()` on user-controlled values, unnecessary `.clone()`, lifetime issues
- **Go**: Check for unhandled errors, goroutine leaks, missing context propagation
- **Java**: Check for unclosed streams, catch-all exceptions, thread safety issues

## Output Format

Produces `REVIEW.md` in the current working directory or a specified output path.

## Examples

### Example 1: Review a phase

```
Review phase 3 -- standard depth
```

The skill identifies all files changed since phase 3 began, runs standard-depth checks, and writes REVIEW.md.

### Example 2: Review specific files at deep depth

```
Deep review on src/auth/login.ts src/auth/session.ts
```

Full security + performance + architecture analysis on the two authentication files.

### Example 3: Quick review before commit

```
Quick review of staged changes
```

Rapid scan for syntax errors, critical bugs, and security showstoppers only. Fast enough for a pre-commit hook.

### Example 4: Review output sample

```markdown
# Code Review -- Phase 3 (standard)
**Depth:** standard
**Date:** 2026-04-22
**Files Reviewed:** 7
**Findings:** 1 critical, 3 warning, 2 info

## Summary
One critical hardcoded API key found in config.ts. Three warnings around
missing error handling in the data layer. Overall quality is good once the
critical finding is resolved.

## Findings

### Critical

- **SEC-001** `src/config.ts:14` -- Hardcoded API key in `const API_KEY = "sk-abc123"`.
  Move to environment variable: `const API_KEY = process.env.API_KEY`

### Warning

- **QUAL-003** `src/data/fetch.ts:27` -- `fetchUser` does not handle network errors.
  Wrap in try/catch and return a typed error result.
- **BUG-001** `src/data/fetch.ts:45` -- `user.address.street` will throw if `address` is null.
  Use optional chaining: `user.address?.street`
- **QUAL-001** `src/data/transform.ts:12-58` -- `normalizeRecord` has cyclomatic complexity of 14.
  Extract validation sub-functions to reduce branching.
```
