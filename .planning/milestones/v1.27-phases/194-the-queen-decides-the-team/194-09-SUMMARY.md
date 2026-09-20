---
phase: 194-the-queen-decides-the-team
plan: 09
subsystem: docs
tags: [claude-md, wrapper-triplets, documentation, D-13, D-15, definition-of-done]

# Dependency graph
requires:
  - phase: 194-01
    provides: "queenRiskSignalTable / queenForcedReviewersForPhase -- the five named signals this plan documents verbatim in CLAUDE.md and the wrapper triplets"
  - phase: 194-05
    provides: "queenFallbackTeam, the rewritten isAlwaysRequired continue branch, and the position-free resolveSmartVerificationDepth -- the shipped behaviour this plan's rewrite describes"
  - phase: 194-07
    provides: "the owner-only forced-reviewer waiver (cmd/forced_reviewer_waiver.go) and its check-in card rendering -- what the Team Check-In section's new decline option documents"
  - phase: 194-08
    provides: "TestOneTaskBugFixIsOneWorkerPlusChecks and the measured one-worker baseline -- cited verbatim in both CLAUDE.md and the closed todo"
provides:
  - "CLAUDE.md's Queen-Owned Orchestration and Team Check-In sections describe the shipped floor (builder, or a reviewer forced by one of five named signals) instead of the retired one (Watcher always, production-mode auditor, position-raised depth)"
  - "TestCLAUDEMDDepthTableEvaluates -- parses the depth table's Continue row out of CLAUDE.md and evaluates it against queenMaxWorkersForBudget and isAlwaysRequired, so an edit to the row that disagrees with the runtime now fails a test instead of silently going stale"
  - "all six wrapper files (.claude/commands/ant/build.md, .claude/commands/ant-build.md, .opencode/commands/ant/build.md and the three continue.md copies) corrected to the same rule, still byte-identical triplets"
  - "the folded todo 2026-08-21-weight-classes-pipeline-fits-the-task.md closed and moved to .planning/todos/completed/, naming TestOneTaskBugFixIsOneWorkerPlusChecks as its resolution proof"
affects: []

# Actuals (#2632)
actuals:
  tokens: 13112
  tasks: 2
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "A documentation test that parses the actual row text out of the doc file (rather than asserting a hardcoded Go literal) closes the gap a word-presence check cannot: editing the doc's own claim can now fail the test, which is what makes the claim testable per CLAUDE.md's own Definition of Done."

key-files:
  created: []
  modified:
    - CLAUDE.md
    - cmd/claudemd_verification_depth_test.go
    - .claude/commands/ant/build.md
    - .claude/commands/ant-build.md
    - .opencode/commands/ant/build.md
    - .claude/commands/ant/continue.md
    - .claude/commands/ant-continue.md
    - .opencode/commands/ant/continue.md
    - .planning/todos/completed/2026-08-21-weight-classes-pipeline-fits-the-task.md

key-decisions:
  - "TestCLAUDEMDDepthTableEvaluates parses CLAUDE.md's own Continue table row (via string split on '|', then a 'max (\\d+)' regex) rather than encoding a second, hand-written copy of the expected values in Go. Encoding a second copy would let the doc's prose and the test's expectation drift apart from each other while both still passed against the runtime independently -- exactly the failure mode the plan's prohibition ('no sentence may be added that no test can falsify') exists to close. Parsing the actual cell text means an edit to the row itself is what the acceptance criterion's temporary-edit demonstration exercises."
  - "The Seal row of the depth table was left unchanged. Both 194-CONTEXT.md (D-11/D-13: 'Only build and continue change in this phase; plan, colonize, swarm and seal keep their current required sets') and 194-05-SUMMARY.md confirm seal's required-caste and worker-cap logic was not touched by this milestone; rewriting its row would have been describing behaviour nobody shipped."
  - "cmd/queen_spawn_budget.go's queenBuildSafetyRequiredCastes docstring and internal helpers were read but not further modified -- Task 1's scope is CLAUDE.md and its verification test, not runtime code; the runtime already matches the rewritten prose (verified by running every test named in the new prose before committing)."
  - "Line 90 of continue.md ('The Watcher is not yours to decide; the runtime always includes it') was corrected even though the plan's read_first only named lines 118-135 for this file. queenContinueReviewSpecsWithJudgement (cmd/codex_continue.go:1350) explicitly skips the watcher caste from the heavy-review spawn list, so the claim was false under the current runtime and sat three paragraphs above the corrected paragraph -- leaving it would have reproduced the exact 'two contradictory truths, no way to tell which is current' failure mode this plan's own objective names. In scope under the plan's must_haves truth ('No shipped instruction file still states... any caste is required') rather than a deviation from it."

requirements-completed: [TEAM-01, TEAM-04, TEAM-05]

coverage:
  - id: D1
    description: "CLAUDE.md's Queen-Owned Orchestration section (priority list, smart defaults, depth table, required-worker table, for-dummies paragraph) and Team Check-In section describe the shipped floor -- builder always, a reviewer only when one of five named signals fires -- with every named test verified to exist and pass."
    requirement: TEAM-05
    verification:
      - kind: unit
        ref: "go test ./cmd -run 'TestCLAUDEMDVerificationDepthClaims|TestCLAUDEMDNoOldModes|TestCLAUDEMDDepthTableEvaluates|TestBuildWorkerCapHonoursVerificationDepth|TestBuildDepthCapDoesNotStripRequiredSafetyCastes' -count=1"
        status: pass
      - kind: other
        ref: "grep -c 'Final phase → heavy' CLAUDE.md = 0; grep -c 'always, on every build' CLAUDE.md = 0; grep -c 'always — unchanged, and must stay so' CLAUDE.md = 0; credentials/payments/migration all present in the Queen-Owned Orchestration section"
        status: pass
    human_judgment: false
  - id: D2
    description: "TestCLAUDEMDDepthTableEvaluates reads the Continue row out of CLAUDE.md and evaluates it against queenMaxWorkersForBudget and isAlwaysRequired; demonstrated to fail when the row is edited to disagree with the runtime (Gatekeeper removed from the heavy cell), then restored and re-verified passing."
    requirement: TEAM-05
    verification:
      - kind: unit
        ref: "cmd/claudemd_verification_depth_test.go#TestCLAUDEMDDepthTableEvaluates"
        status: pass
      - kind: other
        ref: "Manual demonstration: heavy cell's 'Gatekeeper' removed -> go test ./cmd -run TestCLAUDEMDDepthTableEvaluates -count=1 -v failed with 'continue/heavy cell does not name gatekeeper, but heavy requires it unconditionally' and 'these must agree'; row restored -> test passes again"
        status: pass
    human_judgment: false
  - id: D3
    description: "All three copies of build.md and all three copies of continue.md are edited identically (no stale claim, --caste-why shown, waive/decline option added) and remain byte-identical triplets."
    requirement: TEAM-04
    verification:
      - kind: other
        ref: "grep -rl 'A build always gets a Watcher' .claude/commands/ .opencode/commands/ | wc -l = 0; md5 identical across all three build.md copies and all three continue.md copies; --caste-why present in every build.md copy; waive/decline present in every build.md Team Check-In stage"
        status: pass
      - kind: unit
        ref: "go test ./cmd -count=1 (whole-package parity/golden tests included)"
        status: pass
    human_judgment: false
  - id: D4
    description: "The folded todo 2026-08-21-weight-classes-pipeline-fits-the-task.md is marked resolved, naming TestOneTaskBugFixIsOneWorkerPlusChecks, and moved to .planning/todos/completed/."
    requirement: TEAM-01
    verification:
      - kind: other
        ref: ".planning/todos/completed/2026-08-21-weight-classes-pipeline-fits-the-task.md exists; .planning/todos/pending/2026-08-21-weight-classes-pipeline-fits-the-task.md does not; file names TestOneTaskBugFixIsOneWorkerPlusChecks"
        status: pass
    human_judgment: false

# Metrics
duration: 48min
completed: 2026-08-23
status: complete
---

# Phase 194 Plan 9: The Documentation Now Describes What Shipped Summary

**CLAUDE.md's Queen-Owned Orchestration and Team Check-In sections, and all six wrapper command files, rewritten from the retired floor (Watcher on every build, production-mode auditor, phase position raising depth) to the shipped one (builder always, a reviewer only when one of five named signals fires, declinable only by the owner) — with a new test that parses CLAUDE.md's own table and fails if it drifts from the runtime again, plus the weight-classes todo closed on the measured one-worker number.**

## Performance

- **Duration:** 48 min
- **Started:** 2026-08-23T16:12:00Z (approx, following 194-08's completion)
- **Completed:** 2026-08-23T17:00:17Z
- **Tasks:** 2 completed
- **Files modified:** 9 (CLAUDE.md, cmd/claudemd_verification_depth_test.go, 6 wrapper files, 1 todo moved+edited)

## Accomplishments

- **CLAUDE.md rewritten, not appended.** The "Queen-Owned Orchestration" priority list dropped its position-based smart-default row; the smart-defaults list lost "Final phase → heavy" entirely and states the owner's 2026-08-23 choice explicitly; the depth table's Continue row now describes "no reviewer required unless a named risk forces one" at light/standard and "the full review panel — an explicit owner override" at heavy, with the same worker-cap numbers (3/4/6) unchanged; the four-row "was required / is required" caste table was replaced by a two-row table (Builder always; a forced reviewer only for one of the five named signals, with the signal-to-caste mapping spelled out). A new paragraph states plainly that the program's own checks (build, vet, tests, lint) are unchanged and run at every depth regardless of worker count. The "for dummies" paragraph was rewritten from scratch to describe who turns up by default, what makes a reviewer turn up, and that "light" never switches off a reviewer a risky phase genuinely needs.
- **Team Check-In section corrected** to state that `REQUIRED` now means exactly "the worker that writes the code, or a reviewer forced by a named signal (signal shown)," and that only the owner — never the Queen, never autopilot — can decline a forced reviewer, with the decline recorded and scoped to one signal on one phase. Six new test names cited (`TestRequiredMeansBuilderOrNamedSignal`, `TestTeamCheckinDoesNotMutate`, `TestOnlyTheOwnerCanWaiveAForcedReviewer`, `TestWaiverCoversOneSignalOnOnePhase`, `TestWaivedSignalStaysWaivedWhenTheFilesRedetectIt`, `TestAutopilotNeverWaives`), all confirmed to exist and pass before citing them.
- **`cmd/claudemd_verification_depth_test.go` split as the plan required.** `TestCLAUDEMDVerificationDepthClaims` lost its "Final phase → heavy" word-presence check and its runtime evaluation of a final-phase fixture (now a pure runtime regression guard with no doc-text dependency, since the doc no longer makes that claim at all). `TestCLAUDEMDNoOldModes` gained three new negative assertions for the three retired-floor phrases. New `TestCLAUDEMDDepthTableEvaluates` parses the Continue row's three cells directly out of CLAUDE.md's own table and evaluates the worker-cap numbers against `queenMaxWorkersForBudget` and the required-caste claims against `isAlwaysRequired` for gatekeeper/auditor/probe/watcher at each depth.
- **Demonstrated the doc-runtime link is real, not decorative.** Temporarily removed "Gatekeeper" from the heavy cell of the Continue row, re-ran `TestCLAUDEMDDepthTableEvaluates`, and it failed with `continue/heavy cell does not name gatekeeper, but heavy requires it unconditionally` and `continue/heavy: doc names gatekeeper=false, isAlwaysRequired=true -- these must agree`. Restored the row and re-ran; passes again.
- **All six wrapper files corrected identically.** In `build.md`: the "Builder and Watcher are not yours to decide" line narrowed to Builder alone (with the discovery-mode Scout carve-out stated); the worked-examples table's last row corrected from "Builder and Watcher suffice" to "the Builder alone is enough"; the "Applying the decision" example now shows `--caste-why` alongside `--castes`/`--caste-reason`, with prose explaining a worker named without its own `--caste-why` is refused by name; "What the runtime will do to your choice" rewritten to state the new floor (Builder restored, a reviewer added only for one of five named things and only at the checking step); the Team Check-In stage's Purpose sentence corrected to distinguish the Builder (never offered for removal) from a forced reviewer (offered for removal only to the owner, only as an explicit recorded decline), and gained a fourth AskUserQuestion option ("Decline a required reviewer") that states the consequence before naming anything and relays the runtime's own `aether decision-answer` command verbatim. In `continue.md`: the "Watcher is not yours to decide" line (which was false under the current runtime — `queenContinueReviewSpecsWithJudgement` explicitly skips the watcher caste from the review-spec list) corrected to state nothing is unconditional except a forced reviewer; the re-fetch example gained `--caste-why`; the "same floors apply... Watcher is restored" paragraph rewritten to the five-signal rule with the correct escape hatch (the owner's own explicit decline, not "not available at any cost"). All three copies of each file confirmed byte-identical (md5) after the edit.
- **`.aether/docs/wrapper-runtime-ux-contract.md` checked and left unmodified** — grepped for "watcher", "always gets", "required cast", "floor"; the only hit (`--skip-watchers` in the continue command-line example) remains accurate and unrelated to the retired floor. The plan's own instruction was conditional ("if any rule there restates the old floor") and none did.
- **Weight-classes todo closed.** `.planning/todos/pending/2026-08-21-weight-classes-pipeline-fits-the-task.md` retitled `[CLOSED 2026-08-23] ...`, given a `## Closed` section explaining the resolution (shrunken floor + fallback team + owner dials, no separate lane) and naming `TestOneTaskBugFixIsOneWorkerPlusChecks` (194-08) as the proof, then moved to `.planning/todos/completed/` — the same convention already used by five other 2026-08-21 todos in that directory.

## Task Commits

Each task was committed atomically:

1. **Task 1: Rewrite the two CLAUDE.md sections that describe the old floor, and make the table evaluate** - `3eda51ff` (docs)
2. **Task 2: Correct the six wrapper files, the UX contract, and close the folded todo** - `252075d9` (docs, rename only — see Deviations) and `94e4b7f5` (docs, content) — both carry Task 2's work

**Plan metadata:** (this commit, following)

## Files Created/Modified

- `CLAUDE.md` — "Queen-Owned Orchestration" and "Team Check-In and Owner Decisions" sections rewritten per D-06/D-11/D-13/D-01/D-04/D-03.
- `cmd/claudemd_verification_depth_test.go` — split `TestCLAUDEMDVerificationDepthClaims`, extended `TestCLAUDEMDNoOldModes`, added `TestCLAUDEMDDepthTableEvaluates`.
- `.claude/commands/ant/build.md`, `.claude/commands/ant-build.md`, `.opencode/commands/ant/build.md` — corrected, byte-identical.
- `.claude/commands/ant/continue.md`, `.claude/commands/ant-continue.md`, `.opencode/commands/ant/continue.md` — corrected, byte-identical.
- `.planning/todos/completed/2026-08-21-weight-classes-pipeline-fits-the-task.md` — closed, moved from `pending/`.

## Decisions Made

See `key-decisions` in frontmatter. The two most consequential: (1) the new test parses CLAUDE.md's own table text rather than hardcoding a second expectation in Go, so the acceptance criterion's "edit the table, watch it fail" demonstration is testing the real link, not a decorative one; (2) `continue.md` line 90 was corrected even though the plan's `read_first` only named lines 118-135, because it made the identical false claim about Watcher three paragraphs above the corrected paragraph and would have reproduced the "two contradictory truths" failure this plan exists to close.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Todo move and wrapper-file staging split across two commits**
- **Found during:** Task 2 commit
- **Issue:** `git add` was called with both the wrapper files and the (already-renamed-via-`git mv`) todo path expressed as its old `pending/` location; the stale pathspec caused `git add` to error before staging any of the listed paths, so the commit that followed only carried the pre-staged `git mv` rename, not the wrapper-file edits.
- **Fix:** Re-ran `git add` against the correct (post-rename) paths for all seven remaining files and committed again with the same message.
- **Files modified:** none beyond what Task 2 already touched — this was a staging-sequence error, not a content error.
- **Verification:** `git status --short` clean after the second commit; both commits' combined diff matches the intended Task 2 changes exactly (confirmed via `git diff --stat` across the commit range).
- **Committed in:** `252075d9` (rename only) and `94e4b7f5` (content) — both are Task 2's work, split by an operational mistake rather than a deliberate two-part change.

**2. [Rule 2 - Missing Critical] continue.md line 90's false Watcher claim was not in the plan's named line range**
- **Found during:** Task 2, reading the full `continue.md` file before editing (read_first gate)
- **Issue:** "The Watcher is not yours to decide; the runtime always includes it" (line 90, inside "Decide the review team before spawning") makes the same retired claim as the paragraph the plan explicitly named for correction (lines 118-135), and is contradicted by the runtime (`queenContinueReviewSpecsWithJudgement` skips watcher from the review-spec list entirely — confirmed by reading `cmd/codex_continue.go:1350`).
- **Fix:** Corrected the sentence to state that the phase's own build/test check already ran once and is not re-spawned, and that nothing is unconditional in the reviewer team except a signal-forced reviewer.
- **Files modified:** all three `continue.md` copies (same edit, propagated by file copy).
- **Verification:** `grep -n -i "restored\|always includes\|watcher" continue.md` shows no remaining stale claim; `go test ./cmd -count=1` green.
- **Committed in:** `94e4b7f5`

---

**Total deviations:** 2 auto-fixed (1 blocking — a staging-sequence mistake corrected before the commit was considered final; 1 missing-critical — a second stale claim in the same file that the plan's must_haves truth already covered even though the read_first line range did not name it). No scope creep: both fixes stayed inside the two files Task 2 already had open, and both are required by the plan's own prohibition against leaving a testable-but-false claim in a shipped instruction file.

## Issues Encountered

None beyond the staging-sequence deviation documented above.

## Known Stubs

None.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- Phase 194 (The Queen Decides the Team) is now fully documented to match its shipped behaviour across CLAUDE.md and all six wrapper files; `.planning/WINDOWS.md` was already closed by plan 194-08.
- This was the phase's final plan (9 of 9, per STATE.md). No blockers for phase advancement.

---
*Phase: 194-the-queen-decides-the-team*
*Completed: 2026-08-23*

## Self-Check: PASSED

- FOUND: `CLAUDE.md`
- FOUND: `cmd/claudemd_verification_depth_test.go`
- FOUND: `.claude/commands/ant/build.md`
- FOUND: `.planning/todos/completed/2026-08-21-weight-classes-pipeline-fits-the-task.md`
- FOUND: commit `3eda51ff` in `git log --oneline --all`
- FOUND: commit `252075d9` in `git log --oneline --all`
- FOUND: commit `94e4b7f5` in `git log --oneline --all`
- `go build ./cmd/aether` — pass
- `go vet ./...` — pass
- `go test ./cmd -run 'TestCLAUDEMDVerificationDepthClaims|TestCLAUDEMDNoOldModes|TestCLAUDEMDDepthTableEvaluates|TestBuildWorkerCapHonoursVerificationDepth|TestBuildDepthCapDoesNotStripRequiredSafetyCastes' -count=1` — pass
- `go test ./cmd -count=1` — pass (two full runs, 381.4s and 405.4s, zero failures both times)
- `grep -c 'Final phase → heavy' CLAUDE.md` — 0
- `grep -c 'always, on every build' CLAUDE.md` — 0
- `grep -c 'always — unchanged, and must stay so' CLAUDE.md` — 0
- `grep -rl 'A build always gets a Watcher' .claude/commands/ .opencode/commands/ | wc -l` — 0
- `md5` identical across all three `build.md` copies and all three `continue.md` copies
