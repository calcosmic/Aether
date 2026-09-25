# Blockers during the two-week freeze

Every row is one blocker under the definition in `210-FREEZE-START.md`.

| # | Date | Project | What stopped the work (owner's words) | Report bundle | Fixed? |
|---|------|---------|----------------------------------------|---------------|--------|
| 1 | 2026-09-25 | French Fluency | "Phase 1 finished only after manual workarounds. The colony then became unrecoverable: `aether resume` loops, and `aether pause` refuses to run." (from the field report he brought here, which listed 9 defects) | — | Partly, 2026-09-25: status, resume and autopilot were fixed, but the build path was missed (see row 2). Also fixed: three smaller defects from the same report (commits 44294c6b, 8ead5670, f31fde41). Five others went on the defect register, entries 66–70. |
| 2 | 2026-09-25 | French Fluency | "The colony won't start Phase 2. It's the same fault as before: its records say your approved plan doesn't match the live plan, because of the three tasks its own repair step added." (`/ant-build 2`, 17:26, after the 17:15 install) | — | Yes, 2026-09-25 (commit b2ed612b, installed about 18:00): the fix now sits in the one plan check every path shares. |
| 3 | 2026-09-25 | JUCE Plugin Licensing (Release) | Observed by the assistant, not quoted: Aether's `.aether/.gitignore` is not tracked in git, so `.aether/locks/*.lock` and `.aether/data/spend/session.json` appear as untracked files inside every git worktree Aether writes into. GSD's worktree merge gate reads that as uncommitted work and refused to merge finished phase-14 executor branches back (`cleanup_blocked` / `worktree_dirty`). | — | No |
| 4 | 2026-09-25 | French Fluency | "This is how its going" (brought the French chat here, about 18:35). That chat shows `/ant-run` stopped at phase 3: "The colony's automatic updater rewrote its instructions file and deleted the check commands I'd added, so the checks broke again." | — | No |

## How this is counted

- The count is simply the number of rows.
- A row stays in the table after it is fixed, because the count includes blockers that were
  fixed.
- Anything the owner worked around himself without coming here is deliberately not in the
  table, because the definition fixed before the fortnight says so.
