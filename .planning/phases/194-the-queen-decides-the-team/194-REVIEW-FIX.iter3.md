---
phase: 194-the-queen-decides-the-team
fixed_at: 2026-08-23T18:10:00Z
review_path: .planning/phases/194-the-queen-decides-the-team/194-REVIEW.md
iteration: 1
findings_in_scope: 7
fixed: 7
skipped: 0
status: all_fixed
---

# Phase 194: Code Review Fix Report

**Fixed at:** 2026-08-23T18:10:00Z
**Source review:** .planning/phases/194-the-queen-decides-the-team/194-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 7 (1 Critical, 3 Warning, 3 Info)
- Fixed: 7
- Skipped: 0

All fixes were applied and committed inside an isolated git worktree
(`gsd-reviewfix/194-21068`, based on `oracle-reinstate`), then fast-forwarded
back onto `oracle-reinstate`. `go build ./...`, `go vet ./...`, and the full
`go test ./...` all pass at HEAD. One test (`TestAvailabilityProbeRetriesOnlyTimeouts`,
`pkg/codex`) failed once in the full-suite run and passed cleanly on an
isolated rerun — this is the pre-documented timing-flaky test, not a
regression from these fixes.

## Fixed Issues

### CR-01: The forced-reviewer waiver had no technical control restricting it to the owner

**Files modified:** `cmd/forced_reviewer_waiver.go`, `cmd/ceremony_team_checkin.go`, `cmd/handoff_decisions_cmd.go`, `cmd/forced_reviewer_waiver_test.go`
**Commit:** `4cd7333d`
**Applied fix:**

Implemented the review's primary recommendation in full:

1. `ensureForcedReviewerWaiverPendingDecision` (new, `forced_reviewer_waiver.go`)
   writes the exact, deterministic waiver question into
   `pending-decisions.json` as a **pending, unresolved** row — but only at
   the moment the check-in card (`aether ceremony team-checkin`,
   `cmd/ceremony_team_checkin.go`) renders that signal as a **live** forced
   reviewer. Idempotent (a repeated render of the same live hit is a no-op)
   and best-effort/non-blocking, so the (otherwise still read-only relative
   to colony state) card render can never fail on this.
2. `resolveForcedReviewerWaiverPendingDecision` (new, `forced_reviewer_waiver.go`)
   is now the **only** way a forced-reviewer-shaped question can be
   answered: it looks up the matching unresolved row the runtime itself
   created and marks it resolved. It never creates a new resolved entry the
   way the general clarification path does.
3. `forcedReviewerWaiverSignalForQuestion` (new, `forced_reviewer_waiver.go`)
   detects whether a candidate `--question` string is shaped like a
   forced-reviewer waiver question by regenerating the deterministic
   sentence for every signal at the phase number **parsed out of the
   question text itself** (never trusting a caller-supplied `--phase` flag,
   which a forger could set to dodge the check).
4. `decisionAnswerCmd.RunE` (`cmd/handoff_decisions_cmd.go`) now branches on
   `forcedReviewerWaiverSignalForQuestion`: if the incoming `--question`
   matches the forced-reviewer shape, it routes through
   `resolveForcedReviewerWaiverPendingDecision` and refuses (recording
   nothing, returning a plain-English error) if no matching runtime-created
   row exists. Every other question keeps the original `recordDecisionAnswer`
   behavior unchanged, so ordinary worker-handoff clarifications are
   unaffected.

**Proof, per this repo's own Definition of Done** (a command that fails when
the requirement is unmet):

- `TestDecisionAnswerCannotForgeAForcedReviewerWaiver` invokes the actual
  `decision-answer` **command path** (`rootCmd.Execute()`, not just the Go
  function `recordDecisionAnswer`) with a correctly-shaped forged question
  and **no** prior card render, and asserts the forced reviewer is still
  present in `queenForcedContinueReviewers`'s output, and that
  `forcedReviewerWaiver` still reports the signal as not waived.
- `TestDecisionAnswerResolvesARuntimeCreatedWaiverRow` proves the real owner
  path still works end to end: render the card (which writes the pending
  row), then run the actual `decision-answer` command against that row, and
  confirm the signal is now waived with the owner's recorded reason.
- The pre-existing `TestAutopilotNeverWaives` is unchanged and still passes
  — necessary, but (as the review itself said) not sufficient on its own,
  which is why the two tests above exist.

**Known, disclosed gap — not fixed in this pass:** the review's Fix section
also asked for a check that rejects a `decision-answer` call issued from a
**spawned worker's context**, distinct from the orchestrating session. I
looked for a reusable marker (the `hookAetherAgentTypePrefix` /
`hookSpawnDenyReason` mechanism in `cmd/hook_cmds.go`, which distinguishes
worker-spawned dispatches from the coordinator) and found it does **not**
apply here: that mechanism only fires on Claude Code's own PreToolUse hook
payload for an `Agent`/`Task` tool call, carrying platform-supplied
`agent_id`/`agent_type` fields Claude Code attaches to that specific hook
event. `aether decision-answer` is an ordinary CLI invocation reached via a
worker's (or the orchestrator's) Bash tool — there is no equivalent
platform-supplied identity attached to that process, and no existing
environment variable in this codebase (`AETHER_WORKER_PLATFORM` and similar
are provider pins, not identity markers) that reliably distinguishes "a
worker's Bash tool ran this" from "the orchestrating session's Bash tool ran
this." Inventing an ad hoc marker (an env var a worker could itself set or
omit, a cwd heuristic, etc.) would be exactly the "weak marker" the fix
guidance warned against — it would not actually be enforceable, only
decorative. This is flagged here for the developer/owner rather than
silently left unaddressed: the runtime-created-row requirement (items 1-4
above) is real, load-bearing protection against forgery from *any* untrusted
caller, worker or otherwise — a worker can no longer wave a signal into
existence out of nothing — but a worker running inside the same shell
session as the orchestrator, at the moment a live card is showing, could
still in principle copy-paste the same command a legitimate owner would run.
Closing that specific residual gap would need either a platform-level
identity signal this repo does not currently receive, or a deliberate
product decision to add one (e.g., a session-scoped one-time token minted
only by the interactive orchestrator and never handed to a worker's prompt).

### WR-01: The file-based forced-reviewer detector trusted the worker's own self-reported changed files

**Files modified:** `cmd/queen_risk_signals.go`, `cmd/codex_continue.go`, `cmd/codex_continue_plan.go`, `cmd/queen_forced_reviewer_test.go`
**Commit:** `cb6f2a10`
**Applied fix:** Added `phaseChangedFilesForRiskSignals`, which unions
`phaseChangedFilesFromHandoffs` (the Builder's self-report) with
`discoverChangedFilesFromGit` — the existing `git diff --name-only
--diff-filter=A/M HEAD` helper this repo already uses at build-finalize time
(`cmd/codex_build_finalize.go`). Both continue dispatch lanes
(`cmd/codex_continue.go`, `cmd/codex_continue_plan.go`) now call this
unioned function instead of the raw handoff-only one. Deliberately **not**
folded into `phaseChangedFilesFromHandoffs` itself, because that function
also drives `commitPhaseAdvance`'s file selection (`cmd/phase_commit.go`) —
widening its output would silently change which files get committed at
phase advance, an unrelated concern.

`TestGitDiffCatchesAFileTheHandoffOmitted` proves the fix: a real git repo
with a tracked-but-unhandoffed migration file still forces the auditor via
`queenForcedContinueReviewers`, even though `phaseChangedFilesFromHandoffs`
alone returns nothing for that phase.

### WR-02: PathPatterns matching was a plain substring, not word-boundary

**Files modified:** `cmd/queen_risk_signals.go`, `cmd/queen_forced_reviewer_test.go`
**Commit:** `0836296c`
**Applied fix:** Added `isPathWordByte` and `matchesPathPatternAtBoundary`,
mirroring the existing prose boundary discipline (`isWordByte`/
`matchesPhraseAtWordBoundary`) but narrower (only ASCII letters/digits count
as "inside a word" for a path — `_`, `-`, `.`, `/` are all boundaries).
`queenRiskSignalHitsFromPaths` now calls the boundary matcher instead of
plain `strings.Contains`. One correction made mid-implementation: a pattern
that already ends/starts with its own separator character (`"migrations/"`,
`"/login"`) supplies its own boundary on that side — requiring *another*
non-word byte immediately past a literal `/` that is already part of the
pattern would have rejected genuine matches like
`"migrations/0007_add_column.sql"`. The final boundary function accounts
for this.

`TestPathPatternMatchingRequiresAWordBoundary` proves both directions: the
exact false positive the review named (`"session"` inside
`repossession_handler.go`) no longer fires, and a genuine `session.go` file
still forces the reviewer.

### WR-03: The `"auth/"` path pattern was directory-anchored and missed auth-named files elsewhere

**Files modified:** `cmd/queen_risk_signals.go`, `cmd/queen_forced_reviewer_test.go`
**Commit:** `ab940c24`
**Applied fix:** Replaced the directory-anchored `"auth/"` PathPattern with
a bare `"auth"`, matched through the same boundary discipline WR-02 added —
consistent with its sibling patterns (`"session"`, `"credential"`,
`"secrets"`), which were already bare words.

`TestBareAuthPathPatternCatchesAuthNamedFiles` proves the fix catches
`auth.go`, `auth_service.go`, and `auth-config.yaml` at any depth (not just
inside a literal `auth/` directory). It also documents, by an explicit
`false`-want test case, the honest limitation this repo already accepts
elsewhere (the `"token"` vs `"tokenizer"` trade-off): a name where `auth` is
fused directly into a longer identifier with no separator character
(`authHandler.go`, `authMiddleware.go`, `oauth.go`) is still outside this
pattern's reach, because word-boundary matching on a lowercased path cannot
recover the camelCase signal that was already erased by `strings.ToLower`.
Widening precision (WR-02) and widening recall (WR-03) pull in opposite
directions for a compound identifier with no separator; I chose to keep the
boundary discipline consistent across all patterns rather than special-case
one entry to be looser, and disclose the limitation in the code comment and
test rather than silently claim full coverage.

### IN-01: The longest-match comparison used the untrimmed pattern length

**Files modified:** `cmd/queen_risk_signals.go`
**Commit:** `0836296c` (same commit as WR-02 — the fix is one line inside
the same replaced comparison)
**Applied fix:** `queenRiskSignalHitsFromPaths` now compares and stores the
same trimmed/lowercased value (`p`) the match test itself runs against,
instead of the raw loop variable (`pattern`). No dedicated regression test
was added beyond the WR-02/WR-03 path-matching tests, which already
exercise this comparison on every call — the review itself noted this was a
latent inconsistency with no live bug today (every table entry already
arrives pre-trimmed and lowercase), so there was nothing behaviorally
different to assert beyond "the existing path-matching tests still pass,"
which they do.

### IN-03: The file-detector's owner-facing sentence named the raw path pattern with its trailing slash

**Files modified:** `cmd/queen_risk_signals.go`, `cmd/queen_forced_reviewer_test.go`
**Commit:** `7bef82d2`
**Applied fix:** Added `forcedReviewerReasonPathLabel`, a rendering-only
trim (`strings.TrimSuffix(pattern, "/")`) applied inside
`forcedReviewerReasonClause` only for `"changed files"`-sourced hits. The
underlying `riskSignalHit.Match` field keeps the raw pattern unchanged
(other callers key off the exact table value), so this is scoped strictly
to the owner-facing sentence.

Updated the pre-existing assertion in `TestChangedFilesCanOnlyAddAForcedReviewer`
(`cmd/queen_forced_reviewer_test.go:340`), which had asserted the rendered
reason contained the literal `"migrations/"` — that assertion now checks the
opposite (contains `"migrations"`, does not contain `"migrations/"`),
reflecting the new, correct behavior. Added a dedicated unit test,
`TestForcedReviewerReasonNeverShowsATrailingSlash`, calling
`forcedReviewerReasonClause` directly against a synthetic hit.

## Skipped Issues

None — all 7 in-scope findings were fixed. (CR-01 has one explicitly
disclosed partial-coverage gap — the worker-context check — documented
above rather than silently left out; the rest of CR-01's fix is complete and
tested.)

## Documentation Corrections

No `.planning/WINDOWS.md` or SUMMARY correction was needed. `WINDOWS.md`'s
own "waive" references are GSD's unrelated windows-tracking vocabulary, not
this phase's forced-reviewer waiver. `CLAUDE.md`'s existing description of
the owner-only waiver ("Only the owner — never the Queen, never
autopilot — can decline a forced reviewer...") remains factually accurate
after this fix; it did not claim technical enforcement against CLI forgery
before, so nothing in it became false. The historical `194-07-SUMMARY.md`
and `194-CONTEXT.md` phase-plan records describing the original (weaker)
implementation were left untouched as historical record, per the review
fixer's scope (fix source code and its own tests, not rewrite completed
phase history).

---

_Fixed: 2026-08-23T18:10:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
