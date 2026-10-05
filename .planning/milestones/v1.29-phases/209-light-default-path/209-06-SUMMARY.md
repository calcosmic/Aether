---
phase: 209-light-default-path
plan: 06
subsystem: cli-wrappers
tags: [wrapper-text, planning-dispatch, ant-go, ant-plan, definition-of-done]

requires:
  - phase: 209-light-default-path
    provides: "209-04's single-door route decision and 209-05's real-session measurement that found the gap (defect register entry 60)"
provides:
  - "A falsifiable test (TestGoPlanningHandoffCanActuallyStartPlanning) that reads the real, on-disk /ant-go wrapper text and fails by name when its planning hand-off cannot reach real planning dispatch"
  - "A rewritten planning hand-off, in all four /ant-go wrapper sources, that delegates by name to /ant-plan's own flow starting at its dispatch stage, instead of naming a command that starts no worker"
affects: [210]

actuals:
  tokens: 5700
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "Delegate-by-name wrapper hand-offs: a short door (`/ant-go`) points at a longer wrapper's own named sections (`/ant-plan`'s Scout Stage, Route-Setter Stage, etc.) rather than copying their steps, so the two cannot drift apart."

key-files:
  created:
    - cmd/go_planning_handoff_test.go
  modified:
    - .aether/commands/go.yaml
    - .claude/commands/ant-go.md
    - .claude/commands/ant/go.md
    - .opencode/commands/ant/go.md

key-decisions:
  - "The falsifiability check tests substance (does the hand-off name a real dispatch entry point, or explicitly delegate to /ant-plan's own flow by name?) rather than a magic word, so a future rewording that still fails to dispatch cannot slip past it by accident."
  - "The fix delegates to /ant-plan's own flow by name (naming its Scout Stage, First-Pass Owner Decision Boundary, Route-Setter Stage, Iteration Card and Timeline, Continue/Pause/Stop, Candidate Review, and Exact Candidate Acceptance sections) rather than copying any of their steps, per the plan's 'delegate, do not copy' constraint — two copies of a 281-line flow would drift."

patterns-established:
  - "A wrapper-text falsifiability test extracts the exact hand-off bullet by its opening/closing markers and asserts on its substance, mirroring cmd/planning_public_paths_200_test.go and cmd/platform_doc_hygiene_test.go rather than inventing a new way to locate wrapper sources."

requirements-completed: [UED-17]

coverage:
  - id: D1
    description: "A check exists that reads the real /ant-go wrapper text and fails when the planning hand-off cannot reach real planning dispatch"
    requirement: "UED-17"
    verification:
      - kind: unit
        ref: "cmd/go_planning_handoff_test.go#TestGoPlanningHandoffCanActuallyStartPlanning"
        status: pass
    human_judgment: false
  - id: D2
    description: "The /ant-go planning hand-off, across all four hand-maintained wrapper sources, delegates by name to /ant-plan's own dispatch flow instead of naming an inert command"
    requirement: "UED-17"
    verification:
      - kind: unit
        ref: "cmd/go_planning_handoff_test.go#TestGoPlanningHandoffCanActuallyStartPlanning"
        status: pass
      - kind: unit
        ref: "cmd/classic_command_parity_test.go#TestClassicCommandParity"
        status: pass
      - kind: unit
        ref: "cmd/go_default_path_test.go#TestGoalReachesBuiltWorkWithNoExtraSteps"
        status: pass
    human_judgment: true
    rationale: "Whether an unattended live assistant session actually stops asking the owner a question can only be conclusively re-proven by re-running a real, metered chat session through /ant-go's medium job (as 209-TIMING.md run 4 did) — that is out of this plan's scope (no timing harness may be added per A-02) and is properly a human/owner call on whether the wrapper-text evidence here is sufficient to close defect register entry 60."

duration: 15min
completed: 2026-09-24
status: complete
---

# Phase 209 Plan 06: The /ant-go Planning Hand-off Now Delegates to Real Dispatch Summary

**`/ant-go`'s planning hand-off no longer names a command that starts no worker — it now delegates by name to `/ant-plan`'s own dispatch flow, proved by a check that was red on the real wrapper text before the fix and green after.**

## Performance

- **Duration:** 15 min
- **Started:** 2026-09-24T19:13:00Z (approx.)
- **Completed:** 2026-09-24T19:16:36Z
- **Tasks:** 2
- **Files modified:** 5 (1 created, 4 modified)

## Accomplishments

- Wrote `TestGoPlanningHandoffCanActuallyStartPlanning`, a test that reads the real on-disk text of all four `/ant-go` wrapper sources and asserts each planning-route hand-off either names the real dispatch entry point (`aether host plan`) or explicitly delegates to `/ant-plan`'s own flow by name — run against the committed text before any wrapper edit, it failed on all four files for the right reason (see "Falsifiability proof" below).
- Rewrote the planning-route hand-off bullet identically across `.aether/commands/go.yaml`, `.claude/commands/ant-go.md`, `.claude/commands/ant/go.md`, and `.opencode/commands/ant/go.md`: it now tells the assistant to skip `/ant-plan`'s "Approved Specification Preflight" and "Choose Planning Preset" sections (already settled by the single door: preset `fast`, specification already handled) and carry out the rest of `/ant-plan`'s own flow by name — starting with `aether host plan --preset fast $ARGUMENTS` to get a real dispatch manifest, then its Scout Stage, First-Pass Owner Decision Boundary, Route-Setter Stage, Iteration Card and Timeline, Continue/Pause/Stop, Candidate Review, and Exact Candidate Acceptance sections — through to a real accepted plan, then `/ant-build`'s own flow to build it.
- Added a plain statement that this hand-off must not stop anywhere along the way to hand any choice — clarifying intent, specification approval, or how to proceed — back to the owner, closing the exact gap 209-TIMING.md run 4 measured (the assistant stopped and asked "just build it, or use Aether?").

## Falsifiability proof (Task 1, run before any wrapper edit)

```
=== RUN   TestGoPlanningHandoffCanActuallyStartPlanning
=== RUN   TestGoPlanningHandoffCanActuallyStartPlanning/.aether/commands/go.yaml
    go_planning_handoff_test.go:70: .aether/commands/go.yaml's planning-route hand-off names no way to actually start planning. It must either call the real dispatch entry point ("aether host plan" -- the command /ant-plan's own wrapper uses to obtain a dispatch manifest and start a real Scout worker) or explicitly delegate to /ant-plan's own flow by name as the thing to carry out. Naming only "aether plan --preset fast" gives a live assistant nothing to act on, and an unattended session will stop and ask the owner instead (see 209-TIMING.md run 4 and defect register entry 60).
=== RUN   TestGoPlanningHandoffCanActuallyStartPlanning/.claude/commands/ant-go.md
    go_planning_handoff_test.go:70: .claude/commands/ant-go.md's planning-route hand-off names no way to actually start planning. [same reason]
=== RUN   TestGoPlanningHandoffCanActuallyStartPlanning/.claude/commands/ant/go.md
    go_planning_handoff_test.go:70: .claude/commands/ant/go.md's planning-route hand-off names no way to actually start planning. [same reason]
=== RUN   TestGoPlanningHandoffCanActuallyStartPlanning/.opencode/commands/ant/go.md
    go_planning_handoff_test.go:70: .opencode/commands/ant/go.md's planning-route hand-off names no way to actually start planning. [same reason]
--- FAIL: TestGoPlanningHandoffCanActuallyStartPlanning (0.00s)
    --- FAIL: TestGoPlanningHandoffCanActuallyStartPlanning/.aether/commands/go.yaml (0.00s)
    --- FAIL: TestGoPlanningHandoffCanActuallyStartPlanning/.claude/commands/ant-go.md (0.00s)
    --- FAIL: TestGoPlanningHandoffCanActuallyStartPlanning/.claude/commands/ant/go.md (0.00s)
    --- FAIL: TestGoPlanningHandoffCanActuallyStartPlanning/.opencode/commands/ant/go.md (0.00s)
FAIL
FAIL	github.com/calcosmic/Aether/cmd	1.024s
```

Red for the right reason, named all four wrapper sources by their real repo path, and said in plain terms what was missing — not "assertion failed". This is what made proceeding to Task 2 honest: a version of this test that had come out green before Task 2 would have been a false certificate and would have needed to be rewritten, per the plan's own instruction.

After Task 2's edits, the same test passes on the real, on-disk text:

```
=== RUN   TestGoPlanningHandoffCanActuallyStartPlanning
=== RUN   TestGoPlanningHandoffCanActuallyStartPlanning/.aether/commands/go.yaml
=== RUN   TestGoPlanningHandoffCanActuallyStartPlanning/.claude/commands/ant-go.md
=== RUN   TestGoPlanningHandoffCanActuallyStartPlanning/.claude/commands/ant/go.md
=== RUN   TestGoPlanningHandoffCanActuallyStartPlanning/.opencode/commands/ant/go.md
--- PASS: TestGoPlanningHandoffCanActuallyStartPlanning (0.00s)
```

## Task Commits

Each task was committed atomically:

1. **Task 1: A check that fails on the text as it stands today** - `c041b515` (test)
2. **Task 2: Make the hand-off reach planning for real** - `ff3a1435` (fix)

**Plan metadata:** (this commit)

## Files Created/Modified

- `cmd/go_planning_handoff_test.go` - New falsifiability test reading the real four `/ant-go` wrapper sources and asserting the planning-route hand-off reaches real dispatch and forbids stopping to ask the owner
- `.aether/commands/go.yaml` - Canonical wrapper source: planning-route hand-off rewritten to delegate by name to `/ant-plan`'s dispatch flow
- `.claude/commands/ant-go.md` - Managed projection, byte-identical bullet change
- `.claude/commands/ant/go.md` - Managed projection, byte-identical bullet change
- `.opencode/commands/ant/go.md` - Managed projection, byte-identical bullet change

## Decisions Made

- The falsifiability test asserts on substance (a real dispatch command, or an explicit named delegation to `/ant-plan`'s own flow) rather than requiring one exact sentence, so a future rewording that still fails to reach dispatch is still caught.
- Delegation, not duplication: the fix names `/ant-plan`'s own section titles (Scout Stage, Route-Setter Stage, etc.) as the flow to carry out rather than copying any of their staged commands into `/ant-go`'s own wrapper text — this keeps one source of truth for how planning dispatches, per the plan's explicit prohibition against copying the 281-line flow.
- The three managed markdown wrapper projections (`.claude/commands/ant-go.md`, `.claude/commands/ant/go.md`, `.opencode/commands/ant/go.md`) were kept byte-identical to each other, matching this repo's existing hand-maintained-triplet convention; the canonical YAML source carries the same sentence in its own numbered-list format.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Defect register entry 60's proposed resolution ("close the gap now" — 209-VERIFICATION.md option 1) is delivered: `/ant-go`'s planning hand-off now reaches real dispatch, proved by a check that reads the real wrapper text and was red before the fix.
- UED-17 is re-tickable: its own stated re-tick condition ("re-tick when the wrapper hand-off is proved by a check that fails on the real wrapper text") is met — this plan's `update_requirements` step marks it complete.
- What this plan does NOT re-prove: a fresh, real, metered `claude` session actually completing the medium job through `/ant-go` with zero owner-typed steps, the way 209-TIMING.md run 4 attempted and failed. That would be a second real-session measurement, out of this plan's scope (no new timing harness per A-02), and is the honest human-judgment gap recorded in this SUMMARY's `coverage` block above.
- Nothing else in phase 209 moved: `cmd/go_default_path_test.go` was not modified, and the golden/parity/severity/command-guide tests named in the plan's verification block all still pass.

---
*Phase: 209-light-default-path*
*Completed: 2026-09-24*

## Self-Check: PASSED

- FOUND: cmd/go_planning_handoff_test.go
- FOUND: .aether/commands/go.yaml
- FOUND: .claude/commands/ant-go.md
- FOUND: .claude/commands/ant/go.md
- FOUND: .opencode/commands/ant/go.md
- FOUND commit c041b515 (Task 1)
- FOUND commit ff3a1435 (Task 2)
