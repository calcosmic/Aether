# AETHER Error Ledger

**Purpose**: Track mistakes to prevent recurrence. Every error is an opportunity to improve the system.

---

## How to Log

### DO Log

- AI-generated code that caused bugs
- Patterns that failed in practice
- Context that caused issues
- Tests that failed unexpectedly
- Architectural decisions that didn't work
- Performance issues discovered
- Cross-platform problems

### DON'T Log

- User-requested intentional behavior
- Known limitations documented elsewhere
- Transient issues (network, etc.)
- External tool failures (not our fault)

---

## Error Log Format

```markdown
### [YYYY-MM-DD] Error Title

**Symptom**: What went wrong
**Root Cause**: Why it happened
**Fix Applied**: How it was fixed
**Prevention Added**:
  - Constraint added: CONSTRAINT-XXX
  - Pattern added: stdlib/pattern-name.md
  - Snippet added: snippets/language.code-snippets
**Related**: Links to docs, examples, or issues
**Category**: error:category (for flag threshold)
```

---

## Logged Errors

*No errors logged yet. AETHER is learning!*

---

## Statistics

| Category | Count | Status |
|----------|-------|--------|
| context:overload | 0 | ✓ |
| context:missing | 0 | ✓ |
| pattern:wrong-source | 0 | ✓ |
| pattern:anti | 0 | ✓ |
| constraint:violation | 0 | ✓ |
| gate:skipped | 0 | ✓ |
| gate:failing | 0 | ✓ |
| test:missing | 0 | ✓ |
| test:failing | 0 | ✓ |
| git:bad-commit | 0 | ✓ |
| dependency:unstable | 0 | ✓ |
| architecture:drift | 0 | ✓ |

---

## Prevention Actions Taken

*No prevention actions yet.*

---

**Last Updated**: 2026-02-01
**Ledger Version**: 1.0
