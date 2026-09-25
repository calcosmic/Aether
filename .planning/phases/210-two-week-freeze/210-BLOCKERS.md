# Blockers during the two-week freeze

Every row is one blocker under the definition in `210-FREEZE-START.md`.

| # | Date | Project | What stopped the work (owner's words) | Report bundle | Fixed? |
|---|------|---------|----------------------------------------|---------------|--------|
| 1 | 2026-09-25 | French Fluency | "Phase 1 finished only after manual workarounds. The colony then became unrecoverable: `aether resume` loops, and `aether pause` refuses to run." (from the field report he brought here, which listed 9 defects) | — | Yes, 2026-09-25: the lock-up plus three smaller defects from the same report (commits 44294c6b, 8ead5670, f31fde41). Five others went on the defect register, entries 66–70. |

## How this is counted

- The count is simply the number of rows.
- A row stays in the table after it is fixed, because the count includes blockers that were
  fixed.
- Anything the owner worked around himself without coming here is deliberately not in the
  table, because the definition fixed before the fortnight says so.
