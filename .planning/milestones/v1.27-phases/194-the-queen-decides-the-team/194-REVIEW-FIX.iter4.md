---
phase: 194-the-queen-decides-the-team
fixed_at: 2026-08-23T18:35:00Z
review_path: .planning/phases/194-the-queen-decides-the-team/194-REVIEW.md
iteration: 2
findings_in_scope: 2
fixed: 1
skipped: 1
status: partial
---

# Phase 194: Code Review Fix Report

**Fixed at:** 2026-08-23T18:35:00Z
**Source review:** .planning/phases/194-the-queen-decides-the-team/194-REVIEW.md (iteration 2)
**Iteration:** 2

**Summary:**
- Findings in scope this iteration: 2 (1 Critical, 1 Info)
- Fixed: 1
- Skipped: 1 (marked out of scope by the reviewer)

All work for this iteration was done inside an isolated git worktree
(`gsd-reviewfix/194-72497`, based on `oracle-reinstate` at `4cd7333d`), then
fast-forwarded back onto `oracle-reinstate`. `go build ./...`, `go vet
./cmd/...`, the targeted test set, and the full `go test ./...` all pass.
One test (`TestAvailabilityProbeRetriesOnlyTimeouts`, `pkg/codex`) failed
once in the full-suite run and passed cleanly on an isolated rerun — this is
the pre-documented timing-flaky test, not a regression from this fix.

## Fixed Issues

### CR-01 (residual): The runtime-created waiver row stayed open and resolvable for the whole build

**Files modified:** `cmd/forced_reviewer_waiver.go`, `cmd/spawn.go`,
`cmd/forced_reviewer_waiver_test.go`, `.claude/commands/ant/build.md`,
`.opencode/commands/ant/build.md`, `.claude/commands/ant-build.md`,
`.planning/phases/194-the-queen-decides-the-team/194-07-SUMMARY.md`
**Commit:** `bce085ce`
**Applied fix:**

Iteration 1 closed the forgery half of CR-01 (a `decision-answer` call with
no prior card render is refused). It left one gap: the pending row the card
writes the moment it shows a live forced reviewer had no expiry, so it
stayed open and answerable for the entire build — the owner's default
answer, "proceed," never calls `decision-answer` at all, so a later call
(a worker's own Bash tool, a stray script, a prompt-injected instruction)
running the exact command the card had legitimately displayed could still
waive a reviewer the owner never declined.

The fix closes the decline window the moment dispatch genuinely begins,
anchored at **`spawn-log`** (`cmd/spawn.go`), not `build-finalize` or the
build attempt's own `StartedAt`. Both alternatives were checked and ruled
out first: `beginBuildAttempt`'s `StartedAt` is written during `aether build
--plan-only` itself, **before** the check-in card is even rendered to the
owner (would close the window before it ever opens); `build-finalize` runs
**after** every worker has already completed (would close it far too late
to matter). `spawn-log` is what the wrapper calls once per worker, right
before the platform's Task tool actually spawns that worker — the earliest
point in the runtime that fires after the owner has seen the card (and
answered it, or chosen to proceed) and before any worker's own Bash tool
could possibly run.

1. `spawn-log` gained an optional `--phase` flag (default 0, so every
   existing caller that doesn't pass it — none did before this fix — keeps
   working unchanged). When a caller does pass it, the first spawn for that
   phase calls the new `closeForcedReviewerWaiverWindowForPhase`
   (`cmd/forced_reviewer_waiver.go`), which:
   - records the phase's dispatch-start time in a new small file,
     `phase-dispatch-started.json` (earliest time wins on repeat calls —
     one call per worker spawned in the same build), and
   - deletes any still-**pending** (unresolved) forced-reviewer waiver row
     for that phase. A row the owner already resolved (a genuine decline
     made before dispatch began) is left untouched.
2. `resolveForcedReviewerWaiverPendingDecision` now also refuses directly
   once dispatch has started for a phase, even if a matching pending row
   somehow still exists (belt-and-suspenders against a race with the
   deletion above) — `decision-answer` then returns the same plain-English
   refusal message CR-01's iteration-1 fix already built for "no card was
   ever rendered": *"Nothing was recorded. Phase N is not currently waiting
   on an answer for that reviewer..."*
3. `forcedReviewerWaiver` (the continue-time check both dispatch lanes go
   through) now also rejects a resolved row whose `ResolvedAt` falls at or
   after the phase's recorded dispatch-start time — defense in depth against
   a resolved entry inserted by directly editing `pending-decisions.json`
   rather than through the CLI, which would bypass the two checks above
   entirely.
4. The wrapper triplet (`.claude/commands/ant/build.md`,
   `.opencode/commands/ant/build.md`, `.claude/commands/ant-build.md`, kept
   byte-identical) now passes `--phase $ARGUMENTS` on the existing
   `spawn-log` line in Worker Spawning — the same line every build path
   (checked-in, trimmed, redirected, and the `--no-checkin`/autopilot lane,
   which all funnel through the same Worker Spawning section) already runs
   before every worker.
5. Corrected a now-stale claim in `194-07-SUMMARY.md` ("no persisted state
   of its own") that this and the iteration-1 fix both made false.

**Owner's real decline path (item 4 of the fix guidance) still works
end-to-end:** card renders → owner runs `decision-answer` at the check-in
pause, before any worker is dispatched → `spawn-log` later closes the
window for that phase → the resolved row is untouched (only *pending* rows
are deleted) and its `ResolvedAt` predates dispatch-start, so
`forcedReviewerWaiver` still honors it at continue time.

**`--no-checkin` and autopilot (item 5):** neither path ever renders the
check-in card, so no pending row is ever created. `spawn-log`'s window-close
call is a no-op in that case — no error, no stray file, and the forced
reviewer stays live at continue time (proved directly, see tests below).

**Proof, per this repo's own Definition of Done** — the reviewer's exact
proof-of-concept turned into permanent tests, run through the real command
path:

- `TestForcedReviewerDeclineWindowClosesWhenDispatchBegins`: render the
  card (no decline), run the real `spawn-log` CLI with `--phase` (dispatch
  begins), then attempt `decision-answer` with the card's own question —
  asserts the reviewer stays forced and `forcedReviewerWaiver` reports it
  not waived. This is the reviewer's own proof-of-concept made permanent.
- `TestOwnerDeclineBeforeDispatchStaysHonoredOnceDispatchBegins`: the
  owner's real path — `decision-answer` before `spawn-log` — still results
  in the reviewer being waived after dispatch begins.
- `TestSpawnLogWithNoCardRenderedIsANoOpAndReviewerStaysForced`: `spawn-log
  --phase` with no prior card render creates no waiver row and the forced
  reviewer stays live, covering `--no-checkin`/autopilot.
- The pre-existing `TestDecisionAnswerCannotForgeAForcedReviewerWaiver` and
  `TestDecisionAnswerResolvesARuntimeCreatedWaiverRow` (iteration 1) still
  pass unchanged.

**Scope boundary, disclosed rather than silently left out:** the TS host
lane (`.aether/ts-host/src/worker-dispatch.ts`, reached only via `aether
host build` / `aether run`, never the interactive wrapper) also calls
`spawn-log` but was **not** updated to pass `--phase`. This is intentional,
not an oversight: that lane is autopilot/host-driven and, per this repo's
own `build.md` ("Autopilot (`/ant-run`) never runs this stage — it does not
run this wrapper and can never decline a reviewer"), never renders the
check-in card and so never creates a pending row for `spawn-log` to need to
close. Wiring `--phase` through there too would be pure defense-in-depth
symmetry with no behavior it currently protects, at the cost of a
TypeScript source change plus a `dist/` rebuild in a package this fix
otherwise does not touch — left as a disclosed follow-up rather than bundled
into this fix.

## Skipped Issues

### IN-01: `TestCasteRelevanceDoc_SpawnBudgetNumbersMatch`'s doc-consistency check is a loose numeric substring match

**File:** `cmd/caste_relevance_doc_test.go:204-213`
**Reason:** Explicitly marked by the reviewer as "unchanged, pre-existing,
out of scope" for this phase, carried forward from iteration 1 (there
numbered IN-02) with no change across either fix round. Per the fix
guidance, only tightened if small and safe — this one requires anchoring the
check to a specific table row/line (or switching to the repo's existing
`docstest`-style structured extraction), which is a genuine behavior change
to a doc-consistency test, not a small mechanical edit, so it was left as
disclosed future work rather than risked in this pass.
**Original issue:** `strings.Contains(content, strconv.Itoa(got))` only
proves the digit sequence appears somewhere in the doc, not that it appears
as the documented number for that specific flow/depth row.

## Findings Fixed In Prior Iterations (carried forward)

Iteration 1 (commits `0836296c`, `ab940c24`, `7bef82d2`, `cb6f2a10`,
`4cd7333d`, full detail in the superseded
`194-REVIEW-FIX.iter3.md`) fixed all findings from the original
`194-REVIEW.md` iteration 1 review:

- **CR-01** (forgery half): `aether decision-answer` can no longer forge a
  waiver with no prior card render — `4cd7333d`.
- **WR-01**: the changed-files risk detector now unions the builder's
  self-reported files with an independent `git diff` — `cb6f2a10`.
- **WR-02**: path-pattern matching now requires a word boundary instead of
  plain substring — `0836296c`.
- **WR-03**: the `"auth"` path pattern is no longer directory-anchored —
  `ab940c24`.
- **IN-01** (iteration 1 numbering): the longest-match comparison now uses
  the trimmed pattern length — fixed in `0836296c`.
- **IN-03**: the owner-facing reason no longer shows a raw path pattern's
  trailing slash — `7bef82d2`.

This iteration's fix (CR-01 residual, `bce085ce`) closes the gap iteration
2's re-review found in iteration 1's CR-01 fix; all six findings above
remain fixed and untouched by this iteration's change.

---

_Fixed: 2026-08-23T18:35:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 2_
