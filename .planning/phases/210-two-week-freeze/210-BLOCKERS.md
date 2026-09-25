# Blockers during the two-week freeze

Every row is one blocker under the definition in `210-FREEZE-START.md`.

| # | Date | Project | What stopped the work (owner's words) | Report bundle | Fixed? |
|---|------|---------|----------------------------------------|---------------|--------|
| 1 | 2026-09-25 | French Fluency | "Phase 1 finished only after manual workarounds. The colony then became unrecoverable: `aether resume` loops, and `aether pause` refuses to run." (from the field report he brought here, which listed 9 defects) | — | Partly, 2026-09-25: status, resume and autopilot were fixed, but the build path was missed (see row 2). Also fixed: three smaller defects from the same report (commits 44294c6b, 8ead5670, f31fde41). Five others went on the defect register, entries 66–70. |
| 2 | 2026-09-25 | French Fluency | "The colony won't start Phase 2. It's the same fault as before: its records say your approved plan doesn't match the live plan, because of the three tasks its own repair step added." (`/ant-build 2`, 17:26, after the 17:15 install) | — | No |

## How this is counted

- The count is simply the number of rows.
- A row stays in the table after it is fixed, because the count includes blockers that were
  fixed.
- Anything the owner worked around himself without coming here is deliberately not in the
  table, because the definition fixed before the fortnight says so.
