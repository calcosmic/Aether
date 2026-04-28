---
name: focused-codebase-scan
description: Use when a worker needs a quick targeted codebase scan before planning or implementation
type: colony
domains: [codebase, analysis, scanning]
agent_roles: [surveyor-provisions, surveyor-nest, surveyor-disciplines, surveyor-pathogens, scout]
workflow_triggers: [colonize, plan]
task_keywords: [scan, quick, targeted, map, survey]
priority: normal
version: "1.0"
---

# Focused Codebase Scan

## Purpose

A rapid, focused codebase assessment that produces targeted documents without the overhead of a full analysis. Think of it as the colony sending a scout ahead -- quick reconnaissance, not a full survey. Perfect when you need specific answers fast.

## When to Use

- User says "scan the codebase" or "quick check on the code"
- Need fast answers before committing to deep analysis
- Checking a specific area: security, performance, dependencies
- Validating that a recent change didn't break patterns
- Pre-phase reconnaissance to understand scope

## Instructions

### 1. Scan Modes

```
QUICK:     Top-level scan. Framework, language, structure. <30 seconds.
TARGETED:  Focused scan of specific area. Dependencies, security, API, etc.
HEALTH:    Codebase health check. Tests, coverage, debt indicators.
SECURITY:  Security-focused scan. Auth patterns, secrets, vulnerabilities.
DEPS:      Dependency analysis. Outdated, vulnerable, unused packages.
PATTERNS:  Convention scan. Code style, naming, architectural patterns.
```

### 2. Quick Scan Protocol

```
1. Detect project type from config files (package.json, etc.)
2. Count files by type and directory
3. Identify entry points and main modules
4. Check for test framework and CI configuration
5. Scan for obvious anti-patterns (TODOs, FIXMEs, hardcoded values)
6. Produce summary report
```

### 3. Targeted Scan Protocol

```
1. Scope the scan to the requested area
2. Read only relevant files (e.g., only dependency files for DEPS mode)
3. Analyze within scope, ignore everything else
4. Produce focused report with actionable findings
```

### 4. Health Check Protocol

```
Metrics scanned:
  - Test coverage: Test files vs source files ratio
  - Dependency health: Outdated, vulnerable, deprecated
  - Code complexity: File sizes, function lengths
  - Debt indicators: TODO, FIXME, HACK comment counts
  - Documentation: README presence, inline doc coverage
  - Consistency: Pattern adherence across modules
```

### 5. Output Documents

Write scan results to `.aether/data/scans/`:

```
scan-{mode}-{timestamp}.md

Sections:
  Summary:       One-paragraph overview
  Findings:      Key discoveries (good and bad)
  Metrics:       Relevant numbers
  Recommendations: Top 3 actions based on findings
  Raw Data:      Detailed findings for reference
```

### 6. Scan Depth Control

```
--shallow:   File names and directory structure only
--normal:    File contents scanned, patterns detected (default)
--deep:      Full AST analysis where possible, data flow tracing
```

## Key Patterns

- **Fast over thorough**: A scan that takes 30 seconds is better than a perfect analysis that takes 30 minutes.
- **Actionable findings**: Every finding should have an implied action.
- **Focused scope**: Targeted scans ignore everything outside their scope.
- **Composable**: Multiple targeted scans can replace one full analysis.

## Output Format

```
 SCAN -- {mode} | {project_name}
   Files scanned: {count} | Duration: {time}
   Health score: {1-10}
   
   Key Findings:
    {finding 1}
    {finding 2}
    {finding 3}
   
   Recommendations:
   1. {top recommendation}
   2. {second recommendation}
   3. {third recommendation}
   
   Report: .aether/data/scans/scan-{mode}-{timestamp}.md
```

## Examples

**Quick scan:**
> "Node.js/Express API, 47 source files, 12 test files. Uses PostgreSQL via Prisma. 5 TODOs found, 2 FIXMEs. Health: 7/10. Recommend: add tests for auth module."

**Security scan:**
> "3 findings: hardcoded JWT secret in config.ts:42, no rate limiting on auth endpoints, CORS allows all origins. Health: 4/10. Recommend: immediate secret rotation."
