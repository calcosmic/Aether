# Blockers during the two-week freeze

Every row is one blocker under the definition in `210-FREEZE-START.md`.

| # | Date | Project | What stopped the work (owner's words) | Report bundle | Fixed? |
|---|------|---------|----------------------------------------|---------------|--------|
| 1 | 2026-09-25 | French Fluency | "Phase 1 finished only after manual workarounds. The colony then became unrecoverable: `aether resume` loops, and `aether pause` refuses to run." (from the field report he brought here, which listed 9 defects) | — | Partly, 2026-09-25: status, resume and autopilot were fixed, but the build path was missed (see row 2). Also fixed: three smaller defects from the same report (commits 44294c6b, 8ead5670, f31fde41). Five others went on the defect register, entries 66–70. |
| 2 | 2026-09-25 | French Fluency | "The colony won't start Phase 2. It's the same fault as before: its records say your approved plan doesn't match the live plan, because of the three tasks its own repair step added." (`/ant-build 2`, 17:26, after the 17:15 install) | — | Yes, 2026-09-25 (commit b2ed612b, installed about 18:00): the fix now sits in the one plan check every path shares. |
| 3 | 2026-09-25 | French Fluency | "This is how its going" (brought the French chat here, about 18:35). That chat shows `/ant-run` stopped at phase 3: "The colony's automatic updater rewrote its instructions file and deleted the check commands I'd added, so the checks broke again." | — | Yes, 2026-09-25 (commit 1feead2a, installed about 19:14): the update and setup now keep an owner's Verification Commands section, and code-formatted rows in Aether's own command tables are no longer read as checks. The French check commands were restored and survived two updates. |
| 4 | 2026-09-26 | French Fluency (new "French Basics" project) | "it like suggested that we plan even though it's not able to do the plan and something I don't like about the spec thing is like it says it creates a spec but doesn't even give you the spec to like have a look at" | — | Yes, 2026-09-26 (installed about 14:40): setup now recommends clarifying first, never planning; and, by the owner's choice, /ant-discuss interviews him with about 20 multiple-choice questions, writes every answer into the description, and shows it as a plain-English page (see "Rule changes" in the start record). |
| 5 | 2026-09-26 | French Fluency (French Basics) | Owner typed only `/ant-run` (no words of their own). Autopilot stopped in Phase 1 after about 5 minutes: builder Anvil-7 on task 1.1 came back with status "stopped", which the program doesn't recognise, so the wave failed. Blockers went from 1 to 2 and the Next Up line points to `/ant-resume`. The older "output/ must name a file" planning blocker is still open. | — | Yes, 2026-09-26 (installed about 14:40): the builder had finished with a valid report; Claude Code's notice about the builder's stopped background test run looked like a report and, arriving last, replaced it. Aether now ignores Claude Code's own notices when reading a helper's report. |

## How this is counted

- The count is simply the number of rows.
- A row stays in the table after it is fixed, because the count includes blockers that were
  fixed.
- Anything the owner worked around himself without coming here is deliberately not in the
  table, because the definition fixed before the fortnight says so.

## Noted, not counted

- 2026-09-25, JUCE Plugin Licensing (Release): a chat running a GSD command (not Aether) there
  wrote this row itself: Aether's `.aether/.gitignore` is not tracked in git, so Aether's lock
  and spend files show up as uncommitted work inside git worktrees and GSD's merge step refused
  to merge. Taken out of the count on 2026-09-26 because the owner was not using Aether and did
  not come to Aether's chat about it, so it does not meet the definition. Recorded on the defect
  register instead.
