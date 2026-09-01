---
phase: 195-coherent-jobs
plan: 10
subsystem: docs
tags: [claude-md, owner-docs, doc-truth-guard, team-checkin, coherent-jobs, todo-closure]

# Dependency graph
requires:
  - phase: 195-coherent-jobs
    provides: "195-03's Go-validated grouping pass, 195-04/195-06's two-stage receipt boundary, 195-05's decideBuildCheckin/--checkin/compact summary, 195-07's append-only unfinished-only retry, 195-08's worktree admission/sync/root finalization, and 195-09's single cross-surface contract that CLAUDE.md must now agree with"
provides:
  - "CLAUDE.md's Team Check-In section corrected from the pre-195 unconditional pause to the shipped decision-aware one-worker fast path"
  - "CLAUDE.md's new Coherent Jobs and Completion Evidence section: Go-validated grouping before ownership in both in-repo and worktree modes, covered_task_ids as assignment scope, structural admission granting no credit, root-evidence-only completion"
  - "cmd/claudemd_coherent_jobs_test.go: the first executable guard over CLAUDE.md's Phase 195 claims (retired-claim ban, required-claim anchors, cited-test liveness, and a doc-vs-runtime precedence cross-check)"
  - "The folded one-worker todo closed into .planning/todos/completed/ with owner intent preserved verbatim and three named closure tests"
affects: []

# Actuals (#2632)
actuals:
  tokens: 7700
  tasks: 1
  commits: 1

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "CLAUDE.md doc-truth guard: retired-claim ban plus short un-wrappable required anchors plus a cited-test liveness check, following cmd/docs_truth_test.go's forbidden-anchor precedent and 195-09's five-surface anchor pattern"
    - "Doc-vs-runtime cross-check: the documented precedence table is evaluated against decideBuildCheckin itself, mirroring TestCLAUDEMDVerificationDepthClaims rather than asserting prose alone"

key-files:
  created:
    - cmd/claudemd_coherent_jobs_test.go
    - .planning/todos/completed/2026-08-23-one-worker-build-skips-the-checkin-pause.md
  modified:
    - CLAUDE.md
    - .planning/phases/195-coherent-jobs/deferred-items.md

key-decisions:
  - "Added an executable guard over CLAUDE.md rather than only rewriting its prose. CLAUDE.md is precisely the surface that rotted during Phase 195 -- plan 195-05 changed the behaviour on 2026-08-27 and nothing failed while the doc described the old one. 195-09 closed the same gap for five shipped surfaces with shared anchors; CLAUDE.md had no equivalent, which is why it was the one left stale. Recorded as a Rule 2 deviation."
  - "Required doc anchors are deliberately short and the prose is hand-wrapped around them. Markdown line-wrapping broke four anchors on the first GREEN run -- the identical failure 195-09 hit with its verbatim authority sentences. Short anchors kept on one line are the mitigation."
  - "The retired-claim ban is the load-bearing half: 'including a one-worker team' and 'Builds pause for the owner before spawning' were live CLAUDE.md text made untrue by 195-05, and both now fail by name if reintroduced."
  - "The folded todo's original body -- including the owner's verbatim quote, the 194-07 routing rationale, and the explicit warning that an implicit opt-out is a weaker safety posture than --no-checkin -- was preserved unchanged. Only the frontmatter gained resolution fields and a new Resolution section was appended, which states in terms that the warning was honoured (the fast path is decision-aware, not count-aware)."
  - "The pkg/codex availability-probe failure and the disk-exhaustion failures during the race gate were both re-run to green and logged to deferred-items.md rather than fixed here -- neither is caused by this plan's three files, and fixing them would have broadened this plan's ownership."

patterns-established:
  - "A CLAUDE.md behavioural claim now has a command that fails when it is untrue, satisfying the repo's own Definition of Done for documentation rather than relying on the runtime tests alone."

requirements-completed: [JOBS-01, JOBS-02, JOBS-03, JOBS-04]

coverage:
  - id: D1
    description: "CLAUDE.md states that a one-worker build with no pending owner decision goes straight through, that --checkin or any pending decision still pauses, and that --checkin with --no-checkin is a named conflict refused before any side effect."
    requirement: "JOBS-01"
    verification:
      - kind: unit
        ref: "cmd/claudemd_coherent_jobs_test.go#TestCLAUDEMDDoesNotClaimUnconditionalCheckinPause"
        status: pass
      - kind: unit
        ref: "cmd/claudemd_coherent_jobs_test.go#TestCLAUDEMDStatesCoherentJobContract"
        status: pass
      - kind: unit
        ref: "cmd/claudemd_coherent_jobs_test.go#TestCLAUDEMDCheckinPrecedenceMatchesRuntime"
        status: pass
    human_judgment: false
  - id: D2
    description: "CLAUDE.md states that default and Queen-proposed grouping is validated in Go before file ownership in both in-repo and worktree modes, and that an unsafe order is refused by name while safe groups survive."
    requirement: "JOBS-01"
    verification:
      - kind: unit
        ref: "cmd/claudemd_coherent_jobs_test.go#TestCLAUDEMDStatesCoherentJobContract (anchors \"grouping pass runs first\", \"refused by name\", \"in-repo\", \"worktree\")"
        status: pass
      - kind: unit
        ref: "cmd/claudemd_coherent_jobs_test.go#TestCLAUDEMDCitesLiveTestsForCoherentJobClaims"
        status: pass
    human_judgment: false
  - id: D3
    description: "CLAUDE.md distinguishes structural receipt admission (grants no credit whatsoever) from root-evidence finalization (the only producer of completion credit), names covered_task_ids as assignment scope only, and states that a partially finished job never reports the project as built."
    requirement: "JOBS-02"
    verification:
      - kind: unit
        ref: "cmd/claudemd_coherent_jobs_test.go#TestCLAUDEMDStatesCoherentJobContract (anchors \"`covered_task_ids` is assignment scope only\", \"grants no credit whatsoever\", \"can never become completion credit\", \"A partially finished job never reports the project as built\")"
        status: pass
    human_judgment: false
  - id: D4
    description: "The folded one-worker todo is at the completed path with the owner's verbatim intent preserved, resolves_phase: 195 set, and TestOneWorkerBuildSkipsCheckin / TestOneWorkerWithForcedReviewerWaiverStillPauses / TestCheckinFlagConflictHasNoSideEffects named as executable closure evidence; the pending path no longer exists."
    requirement: "JOBS-03"
    verification:
      - kind: manual_procedural
        ref: "grep -c 'And then for number three, one worker build should go straight through' .planning/todos/completed/2026-08-23-one-worker-build-skips-the-checkin-pause.md -> 1; ls .planning/todos/pending/2026-08-23-... -> No such file or directory"
        status: pass
      - kind: unit
        ref: "cmd/claudemd_coherent_jobs_test.go#TestCLAUDEMDCitesLiveTestsForCoherentJobClaims (all three named tests exist as live functions in package cmd)"
        status: pass
    human_judgment: false
  - id: D5
    description: "Every Phase 195 gate passes together: focused suite, full suite, race suite, vet, binary build, completion-schema drift check, and the three-way build wrapper byte-parity comparison."
    requirement: "JOBS-04"
    verification:
      - kind: integration
        ref: "go test ./cmd ./pkg/colony ./pkg/codex -run 'Test(Coherent|Job|CalVault|Receipt|Checkin|DetectCycles|Worktree|LifecycleWrapper|CommandGuide|PlatformParity)' -count=1"
        status: pass
      - kind: integration
        ref: "go test ./... -count=1 (cmd 804.009s; all other packages ok)"
        status: pass
      - kind: integration
        ref: "go test ./... -race -count=1 (cmd 826.019s; all other packages ok)"
        status: pass
      - kind: integration
        ref: "go vet ./... && go build ./cmd/aether && go run ./cmd/aether contract-schema --check"
        status: pass
      - kind: integration
        ref: "cmp -s .claude/commands/ant/build.md .claude/commands/ant-build.md && cmp -s .claude/commands/ant/build.md .opencode/commands/ant/build.md"
        status: pass
    human_judgment: false
  - id: D6
    description: "The owner-facing prose reads correctly to someone who has never opened this repository -- every invented word translated inline, and the 'for dummies' paragraphs carrying the real meaning."
    verification: []
    human_judgment: true
    rationale: "Plain-English readability is owner judgement. The tests prove the required facts are present, accurate, and cite live locks; they cannot prove the sentences read well to a non-technical reader."

# Metrics
duration: 55min
completed: 2026-08-27
status: complete
---

# Phase 195 Plan 10: Owner Documentation and Folded-Todo Closure Summary

**CLAUDE.md now describes the shipped runtime instead of the behaviour Phase 194 left behind — and, for the first time, a command fails when it does not.**

## Performance

- **Duration:** ~55 min (about 40 of it inside the phase-wide test gates)
- **Completed:** 2026-08-27
- **Tasks:** 1
- **Files modified:** 4 (2 created, 1 modified, 1 moved)

## Accomplishments

- Replaced CLAUDE.md's stale opening claim that builds pause "including a one-worker team" with the shipped decision-aware policy: the fixed precedence (autopilot/`--no-checkin` non-interactive, explicit `--checkin` always pauses, any live pending owner decision pauses even for one worker, exactly one worker with nothing pending takes the fast path, everything else pauses as before), the fact that "one worker" counts workers rather than pieces of work, and the `--checkin`/`--no-checkin` conflict being refused before any side effect.
- Split the corrected check-in policy into four claims that each name their own locks, matching the convention the surrounding CLAUDE.md text already uses — and kept the full check-in card's description intact for the case where the build does still pause.
- Added a new **Coherent Jobs and Completion Evidence** section: dependency/shared-file grouping with incidental bookkeeping paths excluded, Queen proposals validated in Go before dispatch with unsafe order refused by name and safe groups preserved, cycles as a hard planning error, grouping happening before file ownership in both `in-repo` and `worktree` modes, `covered_task_ids` as assignment scope only, the two-stage evidence boundary where admission grants no credit and only root evidence completes a task, preserved-not-destroyed uncredited worktree edits, and unfinished-only append-only retry.
- Wrote both sections' "for dummies" paragraphs in ordinary English — six helpers becoming one, and "the system does not take its word for it, it goes and looks at the actual files".
- Added `cmd/claudemd_coherent_jobs_test.go`, the first executable guard over these CLAUDE.md claims: a retired-claim ban, twelve short required anchors, a check that all 24 cited test names are both quoted in CLAUDE.md and live functions in package `cmd`, and a doc-vs-runtime cross-check evaluating the documented precedence against `decideBuildCheckin` itself.
- Closed the folded one-worker todo into `.planning/todos/completed/` with the owner's verbatim quote and the original 194-07 routing rationale preserved unchanged, `resolves_phase: 195` set, and a Resolution section naming the three required closure tests plus the files that actually changed.

## Task Commits

1. **Task 1: Update owner docs, close the folded todo, and run every phase gate** — `62eb3192` (docs)

**Plan metadata:** (this commit)

The task landed as one commit on purpose: an intermediate commit carrying the new guard while CLAUDE.md was still stale would have left the tree red, and an intermediate commit carrying corrected prose with no guard is exactly the state this plan exists to end. RED was captured and is quoted below instead.

## Files Created/Modified

- `CLAUDE.md` — rewrote the Team Check-In section's opening claim into four locked claims; added the Coherent Jobs and Completion Evidence section; rewrote both "for dummies" paragraphs (+135/−23 lines)
- `cmd/claudemd_coherent_jobs_test.go` — new: `TestCLAUDEMDDoesNotClaimUnconditionalCheckinPause`, `TestCLAUDEMDStatesCoherentJobContract`, `TestCLAUDEMDCitesLiveTestsForCoherentJobClaims`, `TestCLAUDEMDCheckinPrecedenceMatchesRuntime`
- `.planning/todos/completed/2026-08-23-one-worker-build-skips-the-checkin-pause.md` — moved from `pending/`, original body preserved verbatim, resolution frontmatter and a Resolution section appended
- `.planning/todos/pending/2026-08-23-one-worker-build-skips-the-checkin-pause.md` — no longer exists (the move; the only deletion in this plan's commit, and intentional)
- `.planning/phases/195-coherent-jobs/deferred-items.md` — two out-of-scope gate observations logged

## RED Evidence

The guard was written and run **before** CLAUDE.md changed. Real output:

```
--- FAIL: TestCLAUDEMDDoesNotClaimUnconditionalCheckinPause (0.00s)
    CLAUDE.md still contains retired check-in claim "including a one-worker team" -- decideBuildCheckin (cmd/ceremony_team_checkin.go) takes the one-worker fast path when no owner decision is pending
    CLAUDE.md still contains retired check-in claim "Builds pause for the owner before spawning" -- ...
--- FAIL: TestCLAUDEMDStatesCoherentJobContract (0.00s)
    CLAUDE.md is missing required Phase 195 claim "goes straight through"
    CLAUDE.md is missing required Phase 195 claim "counts workers, not jobs of work"
    CLAUDE.md is missing required Phase 195 claim "`--checkin` and `--no-checkin` together is refused"
    CLAUDE.md is missing required Phase 195 claim "grouping pass runs first"
    CLAUDE.md is missing required Phase 195 claim "`covered_task_ids` is assignment scope only"
    CLAUDE.md is missing required Phase 195 claim "grants no credit whatsoever"
    CLAUDE.md is missing required Phase 195 claim "can never become completion credit"
    CLAUDE.md is missing required Phase 195 claim "A partially finished job never reports the project as built"
    CLAUDE.md is missing required Phase 195 claim "only the uncredited tasks"
--- FAIL: TestCLAUDEMDCitesLiveTestsForCoherentJobClaims (0.07s)
    CLAUDE.md no longer cites TestOneWorkerBuildSkipsCheckin as a lock for its Phase 195 claims
    CLAUDE.md no longer cites TestOneWorkerWithForcedReviewerWaiverStillPauses as a lock for its Phase 195 claims
    CLAUDE.md no longer cites TestCheckinFlagConflictHasNoSideEffects as a lock for its Phase 195 claims
    ... (all 24 cited tests) ...
FAIL	github.com/calcosmic/Aether/cmd	0.892s
```

`TestCLAUDEMDCheckinPrecedenceMatchesRuntime` passed on that same pre-change run, which is the honest result: it asserts the runtime, and the runtime was already correct — the doc was the defect. It fails if `decideBuildCheckin`'s precedence ever stops matching what CLAUDE.md documents.

After the change: `go test ./cmd -run 'TestCLAUDEMD' -count=1` → `ok github.com/calcosmic/Aether/cmd 0.731s`.

A partial-GREEN run in between is worth recording because it is the same trap plan 195-09 hit: four anchors failed purely because markdown line-wrapping split them (`goes straight through`, `` `--checkin` and `--no-checkin` together is refused ``, `can never become completion credit`, `A partially finished job never reports the project as built`). The prose was hand-wrapped to keep each anchor on one line.

## Verification Commands and Results

Every command in the plan's `<acceptance_criteria>` and `<verification>` was run. Real results:

- `go test ./cmd ./pkg/colony ./pkg/codex -run 'Test(Coherent|Job|CalVault|Receipt|Checkin|DetectCycles|Worktree|LifecycleWrapper|CommandGuide|PlatformParity)' -count=1` → `ok cmd 56.580s`, `ok pkg/colony 0.465s`, `ok pkg/codex 0.804s [no tests to run]`
- `go run ./cmd/aether contract-schema --check` → `{"ok":true,"result":{"drift":false,"path":".aether/schemas/completion-packet.schema.json"}}`
- `cmp -s .claude/commands/ant/build.md .claude/commands/ant-build.md` → exit 0
- `cmp -s .claude/commands/ant/build.md .opencode/commands/ant/build.md` → exit 0
- `go build ./cmd/aether` → clean. `go vet ./...` → clean. `gofmt -l cmd/ pkg/` → empty
- `go test ./... -count=1` → `ok cmd 804.009s`; every other package `ok`. Run in two halves (non-`cmd` packages, then `cmd`) purely because a single invocation exceeds the harness's 10-minute foreground limit. One flake on the first non-`cmd` half, re-run to green — see Issues Encountered.
- `go test ./... -race -count=1` → `ok cmd 826.019s`; every other package `ok`. Same two-half split. First attempt aborted on disk exhaustion — see Issues Encountered.
- `TestAuditCatalogGolden` passes with no golden update: this plan added no CLI flag and no subcommand.

## Decisions Made

See `key-decisions` in the frontmatter. The load-bearing one: CLAUDE.md got a guard, not just a rewrite. This repo's Definition of Done says a documentation claim about runtime behaviour must be testable or removed, and CLAUDE.md's check-in claim had been false since 2026-08-27 with nothing failing. Correcting the sentence without adding the command that fails when it is wrong would have reproduced the exact condition that made this plan necessary.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] CLAUDE.md's corrected claims had no executable guard**
- **Found during:** Task 1, before editing CLAUDE.md
- **Issue:** The plan's `files_modified` named only `CLAUDE.md` and the two todo paths. Rewriting the prose alone would have left the corrected claims exactly as unprotected as the stale ones were — and the stale ones survived five plans of this same phase without a single command failing. `cmd/docs_truth_test.go` already establishes this repo's forbidden-anchor pattern for markdown claims, and plan 195-09 applied it to five shipped surfaces; CLAUDE.md was the one surface left without it, which is why it rotted.
- **Fix:** Added `cmd/claudemd_coherent_jobs_test.go` — a retired-claim ban, required-claim anchors, a cited-test liveness check, and a doc-vs-runtime precedence cross-check.
- **Files modified:** `cmd/claudemd_coherent_jobs_test.go` (new)
- **Verification:** Seen RED on the unmodified CLAUDE.md (quoted above), GREEN after; `go vet ./...` and `gofmt -l cmd/ pkg/` clean; the full and race suites both pass with it in place.
- **Committed in:** `62eb3192`
- **Scope note:** this adds a file to package `cmd` and therefore technically widens the plan's declared surface. It adds no production code, no CLI flag, and no subcommand — only a guard over the file the plan already owns. It was not used to satisfy or weaken any gate: every gate the plan named was run unchanged and passed.

**2. [Rule 3 - Blocking] The race gate could not run — the machine's disk was full**
- **Found during:** Task 1, `go test ./... -race -count=1`
- **Issue:** Six `pkg/agent/curation` tests and four packages failed with `TempDir: mkdir ...: no space left on device` and `[build failed]`. `df -h /` reported 571Mi available on a 1.8Ti volume at 100% capacity. No assertion was false; the race-instrumented build had nowhere to write.
- **Fix:** `go clean -cache`, freeing 7.2G of regenerable Go build cache (7.8Gi free afterwards). No user data touched.
- **Verification:** The identical command then passed every non-`cmd` package, and `go test ./cmd -race -count=1` → `ok 826.019s`.
- **Committed in:** n/a (no file change)

---

**Total deviations:** 2 auto-fixed (1 missing critical, 1 blocking).
**Impact on plan:** No gate was weakened, skipped, or narrowed; no prior plan's files were touched. The one added file is a guard over this plan's own deliverable.

## Issues Encountered

- **`TestAvailabilityProbeRetriesOnlyTimeouts` (`pkg/codex`) failed once under full-suite load** with `a probe that stalled once and then answered was reported as a failure: timed out`. Re-running the single test (`ok 4.542s`) and the whole package alone (`ok 22.658s`) both pass, and it passed again in both later full runs including under `-race`. The fixture sets a 2-second wall-clock budget and its own comment concedes the fragility ("the budget has to clear /bin/sh startup on a machine running the whole suite"). This plan modified `CLAUDE.md`, one todo file, and added one `cmd` test file — nothing in `pkg/codex`. Logged to `deferred-items.md` under the scope boundary rather than fixed here; it is the same wall-clock-literal class already logged by plan 195-08.
- **Markdown line-wrapping broke four required anchors** on the first GREEN attempt. Resolved by hand-wrapping the prose around them. Recorded because it is the second time in this phase the same mechanism bit (195-09 hit it with verbatim authority sentences in the Codex skill).
- **The full and race suites each exceed the 10-minute foreground command limit** and were run in two halves (non-`cmd` packages, then `cmd`). This is a harness constraint, not a change of gate: every package in `./...` was run under both `-count=1` and `-race -count=1`.

## Known Stubs

None. Every claim added to CLAUDE.md is asserted by a named test that was seen to fail before the change, and every test name cited in CLAUDE.md is verified to exist as a live function in package `cmd`.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 195 is complete: JOBS-01 through JOBS-04 all have executable proofs, and every shipped surface — the Go runtime, the command YAML, the Codex guide, the Codex build-cycle skill, all three byte-identical build wrappers, and now CLAUDE.md — describes the same contract, each with a guard that fails when it drifts.
- Deferred to future work, logged in `deferred-items.md`: wall-clock literals in test fixtures (`TestAvailabilityProbeRetriesOnlyTimeouts` and the earlier 195-08 entries), `cmd/criterion_owner_confirmation.go` not being gofmt-clean, and the deliberate decision not to widen the completion-packet path-laundering guard for worktree-only claims.
- Explicitly untouched, as the plan required: Phase 196 cost display, Phases 197–198 closing-card work, and the deferred spec-builder command.

## Self-Check

- `CLAUDE.md` confirmed modified in `62eb3192` (+135/−23).
- `cmd/claudemd_coherent_jobs_test.go` confirmed present on disk and in `62eb3192` (+223).
- `.planning/todos/completed/2026-08-23-one-worker-build-skips-the-checkin-pause.md` confirmed present; the owner's verbatim quote confirmed intact (`grep -c` → 1).
- `.planning/todos/pending/2026-08-23-one-worker-build-skips-the-checkin-pause.md` confirmed absent (`ls` → No such file or directory).
- Commit `62eb3192` confirmed present in `git log --oneline`.
- `git diff --diff-filter=D HEAD~1 HEAD` reports exactly one deletion: the pending todo path, which is the intentional move.
- Every `<acceptance_criteria>` command and every plan-level `<verification>` command was run, with real output quoted above.

## Self-Check: PASSED

---
*Phase: 195-coherent-jobs*
*Completed: 2026-08-27*
