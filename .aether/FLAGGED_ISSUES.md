# AETHER Flagged Issues

**Status**: No flagged issues ✓

---

## What Are Flagged Issues?

When an error category occurs **3 times**, it gets automatically flagged here. This indicates a systemic issue that needs a permanent solution (usually a new constraint).

---

## Flag Threshold

Default: **3 occurrences** per error category

Configurable in `.aether/CONSTRAINTS.yaml`

---

## Pending Issues

*No pending flagged issues. AETHER is running clean!*

---

## Resolved Issues

*No resolved issues yet.*

---

## Flag History

| Date | Category | Occurrences | Action Taken | Status |
|------|----------|-------------|--------------|--------|
| - | - | - | - | - |

---

## How to Resolve Flagged Issues

1. **Review the error pattern** - Look at ERROR_LEDGER.md entries
2. **Identify root cause** - Why does this keep happening?
3. **Create constraint** - Add to `.aether/CONSTRAINTS.yaml`
4. **Add prevention** - Update patterns, snippets, or docs
5. **Mark resolved** - Move from Pending to Resolved

---

## Session Start Alert

When this file contains pending issues, session start shows:

```
⚠️ FLAGGED ISSUES: X issues pending

These MUST be addressed before continuing:
→ Run /aether:flags to review
→ Workflow BLOCKED until acknowledged
```

---

**Last Updated**: 2026-02-01
**Flag Threshold**: 3 occurrences
