# Phase 108: Golden Workflow Tests - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-05-12
**Phase:** 108-golden-workflow-tests
**Areas discussed:** Ceremony scope

---

## Ceremony Scope

**Q1: What ceremony elements should the golden test capture?**

| Option | Description | Selected |
|--------|-------------|----------|
| Full visual snapshot | Capture full visual output (ANSI-stripped) as golden text file. Most sensitive to regressions. | ✓ |
| JSON manifest snapshot | Capture JSON from --plan-only mode. Less sensitive to visual changes. | |
| Key assertions | Comprehensive assertion lists like existing tests. Flexible but less thorough. | |

**User's choice:** Full visual snapshot
**Notes:** Recommended approach — catches any ceremony regression including structural changes.

---

**Q2: Should the golden snapshot strip ANSI color codes or keep them?**

| Option | Description | Selected |
|--------|-------------|----------|
| Strip ANSI | Golden files contain clean text. Tests don't break on color tweaks. Easier to read and diff. | ✓ |
| Keep ANSI | Catches color changes too. Harder to read golden files and diff. | |

**User's choice:** Strip ANSI
**Notes:** Clean text golden files are easier to maintain and review in diffs.

---

**Q3: When ceremony output intentionally changes, how should golden files be updated?**

| Option | Description | Selected |
|--------|-------------|----------|
| -update flag | Run tests with -update flag to regenerate golden files. Standard Go golden test pattern. | ✓ |
| Manual update | Tests fail on any diff, manually update golden files. | |

**User's choice:** -update flag
**Notes:** Standard Go pattern — CI fails if golden is stale, developers update with -update flag.

---

## Claude's Discretion

- Test implementation format (Go golden test files)
- State mutation assertion approach
- CI integration alongside existing tests
- Golden file location
- Whether to also snapshot JSON output from --plan-only mode

## Deferred Ideas

None — discussion stayed within phase scope
