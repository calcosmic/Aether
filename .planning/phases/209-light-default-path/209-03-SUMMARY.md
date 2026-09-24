---
phase: 209-light-default-path
plan: 03
subsystem: cli
tags: [help, front-door, cobra, owner-ruling, preferences]

requires:
  - phase: 209-light-default-path (plan 01)
    provides: "/ant-go single-door command whose name the new default menu names"
provides:
  - "aether advanced-commands get|set <on|off>, stored with the owner's own hub preferences (Task 1)"
  - "The default front-door help screen shows a short everyday menu instead of the full catalogue, gated behind that setting (Task 2)"
  - "The owner's real ruling on the actual rendered screen, applied as a data edit -- ten commands in two named groups, plus reworded opening/closing lines (Task 4)"
affects: [front-door-help, onboarding, cli-ux]

actuals:
  tokens: 9000
  tasks: 1
  commits: 1

tech-stack:
  added: []
  patterns:
    - "Default-menu membership held as data (frontDoorDefaultMenuGroups) so an owner ruling on a rendered screen is a straight data edit, never a renderer rewrite."
    - "A test that hardcodes the literal approved set independently of the source variable it is guarding, so a later silent change to that variable fails the test by name."

key-files:
  created: []
  modified:
    - cmd/root.go
    - cmd/front_door_209_test.go
    - cmd/front_door_199_test.go

key-decisions:
  - "Owner ruling (2026-09-24, on the real rendered screen) amends D-02's flat six-command proposal to ten commands in two named groups -- 'Everyday commands' (go, init, status, continue, flags, resume, seal) and 'When you need them' (oracle, swarm, dream) -- and approves reworded opening ('No project is set up here yet...') and closing lines. This ruling is the authority for membership, order, grouping and wording; it supersedes the proposal."
  - "TestDefaultMenuShowsOnlyTheApprovedSet's expected list is now a hand-written literal (approvedDefaultMenu209), deliberately not derived from frontDoorDefaultMenuGroups, so a later change to the menu fails this test by name instead of passing quietly. Proved by temporarily removing /ant-dream, confirming the failure named it, then restoring the file byte-identical (verified with diff)."
  - "Fixing TestFrontDoorHelpEmptyStanding (cmd/front_door_199_test.go) was necessary even though this plan's Task 4 <files> block named only cmd/root.go and cmd/front_door_209_test.go: the owner's ruling also reworded the empty-project opening lines that test asserts against the real rendered screen, and front_door_199_test.go is already in the plan's own top-level files_modified list. Documented as a deviation below (Rule 1)."

requirements-completed: [UED-18]

coverage:
  - id: D1
    description: "frontDoorDefaultMenuGroups holds the owner's ruled ten-command, two-group default menu exactly as approved, and the opening/closing lines match the screen he reviewed."
    requirement: "UED-18"
    verification:
      - kind: unit
        ref: "cmd/front_door_209_test.go#TestDefaultMenuShowsOnlyTheApprovedSet"
        status: pass
      - kind: unit
        ref: "cmd/front_door_199_test.go#TestFrontDoorHelpEmptyStanding"
        status: pass
    human_judgment: false
  - id: D2
    description: "Every command the ruling did not put on the default menu (the old full catalogue minus the ten approved) is still registered, unhidden, undeprecated, and runnable when typed directly."
    requirement: "UED-18"
    verification:
      - kind: unit
        ref: "cmd/front_door_209_test.go#TestHiddenCommandsStillRun"
        status: pass
    human_judgment: false
  - id: D3
    description: "TestDefaultMenuShowsOnlyTheApprovedSet actually fails, by name, when the approved menu is silently changed (proves the lock is real, not decorative)."
    verification: []
    human_judgment: true
    rationale: "The proof is an execution-time act (temporarily delete an entry, run the test, observe the named failure, restore the file, diff-verify byte-identical) recorded in this SUMMARY's narrative, not a re-runnable automated check in the suite -- a human reviewing this SUMMARY is the audit trail for that act having actually happened."

duration: 55min
completed: 2026-09-24
status: complete
---

# Phase 209 Plan 03: A Light Default Path -- Task 4 (Owner Ruling Applied) Summary

**The default `--help` screen now shows the owner's own ten-command, two-group ruling -- not the planner's six-command proposal -- locked down by a test written to fail by name if it silently drifts again.**

## Performance

- **Duration (this continuation, Task 4 only):** 55 min
- **Completed:** 2026-09-24T16:05:45Z
- **Tasks:** 1 (Task 4 of 4; Tasks 1-3 completed by prior agents/sessions)
- **Files modified:** 3

## Accomplishments

- `frontDoorDefaultMenuGroups` (`cmd/root.go`) now holds the owner's literal 2026-09-24 ruling: ten commands in two named groups, "Everyday commands" (`/ant-go`, `/ant-init`, `/ant-status`, `/ant-continue`, `/ant-flags`, `/ant-resume`, `/ant-seal`) and "When you need them" (`/ant-oracle`, `/ant-swarm`, `/ant-dream`) -- the three the owner named unprompted as tools he actually reaches for. `frontDoorDefaultMenu` is kept as a flattened view of the two groups for callers that only care whether a command is on the default screen at all.
- The empty-project opening lines and the closing line now match the exact screen the owner reviewed and approved: "No project is set up here yet. Start one with /ant-init \"what you want built\", or just describe a job with /ant-go and it will work out the rest." and a closing line naming `aether advanced-commands set on`, with the earlier "These are the everyday commands." lead-in dropped per his approved wording.
- `TestDefaultMenuShowsOnlyTheApprovedSet` was rewritten to hardcode the owner's ten-entry, two-group ruling as an independent literal (`approvedDefaultMenu209`), dated to the ruling, instead of deriving its expectations from the very variable it is meant to guard. Proved this actually locks the set: temporarily removed `/ant-dream` from `frontDoorDefaultMenuGroups`, re-ran the test, confirmed it failed naming exactly that entry, then restored `cmd/root.go` from a pre-edit backup and verified the restore was byte-identical with `diff` before proceeding.
- `TestHiddenCommandsStillRun` passes unchanged against the final set (it derives "demoted" from the live `frontDoorDefaultMenu`, which is the correct behavior for that guarantee).
- `TestFrontDoorHelpEmptyStanding` (`cmd/front_door_199_test.go`) updated to assert the approved reworded opening against the real rendered screen, via a whitespace-normalized comparison so the exact line-wrap point (a renderer concern, not part of the ruling) can't make the test brittle.

## Task Commits

1. **Task 4: Apply the owner's ruling and lock the approved set** - `b57790dd` (feat)

Prior tasks (already committed by earlier agents, not part of this continuation):
- Task 1: One machine-wide setting for showing everything - `310e5e28`
- Task 2: The default menu is a short list of data, everything else only hidden - `8ace58af`
- Task 3: Checkpoint (owner ruling) - no commit (decision only)

**Plan metadata:** commit follows this SUMMARY (docs).

## Files Created/Modified

- `cmd/root.go` - `frontDoorDefaultMenuGroups` now holds the owner's ruled ten-command, two-group set; `frontDoorDefaultMenu` flattens it; opening/closing lines reworded per the approved screen; `frontDoorDefaultGroupTitle` kept as a compatibility alias to the new `frontDoorDefaultEverydayGroupTitle` constant.
- `cmd/front_door_209_test.go` - `TestDefaultMenuShowsOnlyTheApprovedSet` rewritten against a hand-written literal (`approvedDefaultMenu209`) with group-order and exact-copy assertions; doc comments updated from "six" to "ten, two-group".
- `cmd/front_door_199_test.go` - `TestFrontDoorHelpEmptyStanding` updated to assert the approved reworded opening lines against the real rendered screen.

## Decisions Made

See `key-decisions` in frontmatter above.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug/stale test] Updated `TestFrontDoorHelpEmptyStanding` outside Task 4's named `<files>` block**
- **Found during:** Task 4, immediately after implementing the owner's reworded opening lines
- **Issue:** Task 4's own `<files>` tag names only `cmd/root.go` and `cmd/front_door_209_test.go`, written when the plan anticipated the owner's ruling as "a straight data edit" to the flat six-command list. The owner's actual ruling also reworded the empty-project opening lines ("No project is set up here yet..." replacing "No colony is active..."). `cmd/front_door_199_test.go`'s `TestFrontDoorHelpEmptyStanding` renders the real screen and hardcoded the old opening text, so it would have failed (correctly) once the opening lines changed -- leaving it unfixed would have shipped a broken build, and the file itself is already listed in this plan's own top-level `files_modified` frontmatter (added for Task 1's wrapper-doc work), so this is inside the plan's declared scope even though outside this one task's narrower file tag.
- **Fix:** Rewrote the test's assertion to compare a whitespace-normalized rendering of the real screen against the approved opening sentence (robust to the renderer's own line-wrap point, which the owner's ruling explicitly says is the renderer's job, not the ruling's).
- **Files modified:** `cmd/front_door_199_test.go`
- **Verification:** `go test ./cmd -run TestFrontDoorHelpEmptyStanding -v` passes; full targeted suite (see Verification below) passes.
- **Committed in:** `b57790dd` (Task 4 commit)

---

**Total deviations:** 1 auto-fixed (test correction, Rule 1)
**Impact on plan:** Necessary to keep the build green after implementing the owner's actual (wider) ruling; no scope creep beyond what the ruling itself required.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Verification

Ran (bounded, foreground-waited, well under the 10-minute repo ceiling):

```
go build ./cmd/aether && go vet ./cmd
go test ./cmd -count=1 -timeout 600s -run 'TestFrontDoor|TestDefaultMenu|TestHiddenCommandsStillRun|TestAdvancedSetting|TestAskingForTheMenuWritesNothing|TestClassicCommandParity|TestCLIFlagAudit' -v
```

All passed. `TestNoRegisteredSubcommandIsUnreferenced` was independently confirmed to fail identically at the pre-Task-4 baseline commit (`8ace58af`, via a throwaway `git worktree`, removed after) -- it is the known pre-existing `aether codex-native-worker context-ack` orphan named in this plan's repo-specific rules, unrelated to this task's changes.

### The real rendered screen (record of what shipped)

Built from the committed state (`git rev-parse HEAD` = `b57790dd...`) and run in a directory with no Aether project set up, exactly as a person first meets it:

```
$ AETHER_OUTPUT_MODE=visual AETHER_PLATFORM=claude NO_COLOR=1 COLUMNS=100 aether --help
No project is set up here yet.
Start one with /ant-init "what you want built", or just describe a job with /ant-go and it will work
  out the rest.

Usage: /ant-help [command]

Everyday commands
  /ant-go "<what you want>"  Do one piece of ordinary work, from a typo fix to a whole feature.
  /ant-init "goal"  Start a new project with one goal.
  /ant-status       Show where the project actually stands right now.
  /ant-continue     Check what was built, then move on.
  /ant-flags        Show what is waiting on your decision.
  /ant-resume       Pick back up after a break.
  /ant-seal         Mark the work finished.

When you need them
  /ant-oracle       Research a question properly, going over it until the answer is solid.
  /ant-swarm "<bug>"  Chase down a confusing bug from four angles at once.
  /ant-dream        Let it look around the project and brainstorm what it notices.

Everything else still works exactly as before when you type it directly -- turn the full list back
  on for this machine with: aether advanced-commands set on
```

This matches the exact set, order, grouping, and wording the owner approved on 2026-09-24, content-for-content (line-wrap points are the renderer's own responsive-width behavior, not part of the ruling, per the owner's own note on that distinction).

## Next Phase Readiness

- Plan 209-03 is complete: all four tasks (setting, default menu, checkpoint, owner ruling applied) are done and committed.
- Plan 209-04 ("One sentence reaches built work with no owner steps in between") and 209-05 (timing writeup) remain to execute this phase.
- No blockers carried forward from this plan.

---
*Phase: 209-light-default-path*
*Completed: 2026-09-24*
