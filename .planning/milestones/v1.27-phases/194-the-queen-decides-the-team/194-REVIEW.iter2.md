---
phase: 194-the-queen-decides-the-team
reviewed: 2026-08-23T00:00:00Z
depth: standard
files_reviewed: 58
files_reviewed_list:
  - .aether/docs/command-playbooks/caste-relevance-reference.md
  - .aether/docs/retired-tests-ledger.md
  - .aether/schemas/completion-packet.schema.json
  - .claude/commands/ant-build.md
  - .claude/commands/ant-continue.md
  - .claude/commands/ant/build.md
  - .claude/commands/ant/continue.md
  - .gitignore
  - .opencode/commands/ant/build.md
  - .opencode/commands/ant/continue.md
  - CLAUDE.md
  - cmd/boundary_double_dispatch_test.go
  - cmd/build_attempt_external_test.go
  - cmd/caste_relevance_doc_test.go
  - cmd/caste_relevance_test.go
  - cmd/caste_relevance.go
  - cmd/ceremony_cmd_test.go
  - cmd/ceremony_team_checkin_test.go
  - cmd/ceremony_team_checkin.go
  - cmd/claudemd_verification_depth_test.go
  - cmd/codex_build_finalize_test.go
  - cmd/codex_build_test.go
  - cmd/codex_build.go
  - cmd/codex_continue_plan_test.go
  - cmd/codex_continue_plan.go
  - cmd/codex_continue_test.go
  - cmd/codex_continue.go
  - cmd/codex_dispatch_contract.go
  - cmd/codex_visuals_test.go
  - cmd/codex_workflow_cmds.go
  - cmd/continue_depth_spawn_test.go
  - cmd/continue_fastpath_castes_test.go
  - cmd/floor_unskippable_test.go
  - cmd/forced_reviewer_waiver_test.go
  - cmd/forced_reviewer_waiver.go
  - cmd/golden_workflow_test.go
  - cmd/one_task_bug_fix_test.go
  - cmd/owner_dials_test.go
  - cmd/phase_verified_once_test.go
  - cmd/queen_fallback_team_test.go
  - cmd/queen_forced_reviewer_test.go
  - cmd/queen_judgement_test.go
  - cmd/queen_judgement.go
  - cmd/queen_orchestration_regression_test.go
  - cmd/queen_probe_gating_test.go
  - cmd/queen_relevance_floor_test.go
  - cmd/queen_risk_signals.go
  - cmd/queen_spawn_budget.go
  - cmd/queen_team_choice_test.go
  - cmd/queen_worker_reason_test.go
  - cmd/reclaim_wiring_test.go
  - cmd/review_depth_test.go
  - cmd/review_depth.go
  - cmd/spawn_budget_test.go
  - cmd/testdata/command_catalog.json
  - cmd/testdata/golden_build.txt
  - cmd/testdata/golden_continue.txt
  - cmd/testdata/golden_plan.txt
findings:
  critical: 1
  warning: 3
  info: 3
  total: 7
status: issues_found
---

# Phase 194: Code Review Report

**Reviewed:** 2026-08-23T00:00:00Z
**Depth:** standard
**Files Reviewed:** 58
**Status:** issues_found

## Summary

This phase moves reviewer selection from a fixed floor to Queen judgement plus
a five-signal forced-reviewer table (`cmd/queen_risk_signals.go`), with an
owner-only waiver path (`cmd/forced_reviewer_waiver.go`). I read the core
logic (`queen_risk_signals.go`, `forced_reviewer_waiver.go`,
`queen_judgement.go`, `caste_relevance.go`, `queen_spawn_budget.go`,
`ceremony_team_checkin.go`), the two continue dispatch lanes
(`codex_continue.go`, `codex_continue_plan.go`), and a representative sample
of the new/changed tests, cross-checked against docs
(`caste-relevance-reference.md`, `retired-tests-ledger.md`, `CLAUDE.md`) and
regenerated golden fixtures.

The engineering is careful and well-documented: the word-boundary phrase
matcher, the two-lane parity tests
(`TestBothContinueLanesForceTheSameReviewers`), the "assert on the real
dispatch list, not the decision record" discipline
(`TestReviewerForcedOnlyByNamedRisk`, `TestNoCasteIsDispatchedAtBothBoundaries`),
and the D-04 collapse rule all held up under adversarial reading, and I found
no tautological tests in the sample I traced. My one Critical finding is the
one the review brief specifically asked me to check hardest: the "owner-only"
waiver has no technical enforcement behind it — anything that can run the
`aether` CLI can forge it. The Warnings are about the second, independent
risk-signal detector (matched against a builder's self-reported changed
files): it trusts unverified worker self-report, and its path-matching is
inconsistent (plain substring in one place, word-boundary in the phrase
matcher), producing both a concrete false positive and a concrete false
negative.

## Critical Issues

### CR-01: The forced-reviewer waiver has no technical control restricting it to the owner

**File:** `cmd/forced_reviewer_waiver.go:80-107`, `cmd/handoff_decisions_cmd.go:165-197`

**Issue:** The whole design of D-03 rests on the claim (stated in this file's
own doc comments) that `aether decision-answer` is "the owner's ONLY way to
decline a forced reviewer." `forcedReviewerWaiver` (queen_risk_signals'
consumer) treats a phase's forced reviewer as waived purely by finding a
resolved `pending-decisions.json` entry whose normalized text matches the
deterministic waiver question (`forcedReviewerWaiverQuestionText`, built only
from `phaseID` + one of exactly five fixed `PlainEnglish` strings that are
hard-coded in `queenRiskSignalTable` and printed in the check-in card).

`decisionAnswerCmd` (`cmd/handoff_decisions_cmd.go:165-197`) has no
authentication, no TTY check, and no requirement that the question match a
pre-existing pending decision — `recordDecisionAnswer` (line 102) simply
appends a new resolved entry for whatever `--question`/`--answer`/`--phase`
text it is given. Because the waiver question text is fully deterministic and
its only two free variables (`phaseID`, one of five fixed sentences) are both
public knowledge available to anything reading `queen_risk_signals.go` or the
rendered check-in card, any process capable of invoking the `aether` binary —
a Builder's own Bash tool, a misfiring script, a prompt-injected instruction
buried in phase content, or a future automation that shells out — can forge:

```
aether decision-answer --question "Phase 7: a security reviewer is being added because this touches logins and passwords. Waive it?" --answer "auto" --phase 7
```

and silently disable the security or quality reviewer the whole feature exists
to guarantee, for that phase, with no proof a human ever saw it.

The only control that exists is procedural, not technical: the wrapper
markdown (`.claude/commands/ant/build.md:270-273`) instructs the orchestrating
agent to relay the command "for the operator to run" and wait, and
`TestAutopilotNeverWaives` (`cmd/forced_reviewer_waiver_test.go:122-136`) only
proves `autopilot.go`'s Go source never calls the Go function
`recordDecisionAnswer` directly — it does not, and cannot, prove nothing ever
invokes the `aether decision-answer` CLI from outside that one code path.
Given this repo's own Definition of Done ("a requirement is satisfied only
when a command exists that someone can run, and that command fails when the
requirement is unmet"), the owner-only guarantee this whole plan is built
around is not actually enforced by any command — it can be silently
bypassed by anything else with shell access.

**Fix:** Require the waiver to reference a pending-decision row that the
runtime itself created (e.g. have the check-in-card render step or
`aether ceremony team-checkin` write the exact pending, *unresolved*
question into `pending-decisions.json` when it renders a live forced
reviewer, and have `decision-answer` only resolve an existing row rather than
create arbitrary new ones for this class of question) — or, at minimum, add a
runtime check that rejects a `decision-answer` call for a forced-reviewer
question unless invoked with a distinct flag/environment marker that only the
true human-operated wrapper session can set, and add a test that proves an
`aether decision-answer` call with no such marker cannot waive a signal (not
just that `autopilot.go`'s source lacks a direct function call).

## Warnings

### WR-01: The file-based forced-reviewer detector trusts the worker's own self-reported changed files, with no independent verification

**File:** `cmd/queen_risk_signals.go:190-225`, `pkg/codex/worker.go:1083`, `cmd/phase_commit.go:71-94`

**Issue:** D-02's second detector (`queenRiskSignalHitsFromPaths`) is sold as
an independent safety net that catches risk the phase's own wording missed —
"the plan's own wording quotes the matched phrase... a changed file names the
matched path pattern." But its only input, `phaseChangedFilesFromHandoffs`,
is built entirely from `ChangedFiles` on the worker's own handoff record
(`pkg/codex/worker.go:1083`: `claims.FilesCreated`/`FilesModified`/
`TestsWritten`, all self-reported by the same Builder whose work is being
reviewed) — never cross-checked against `git diff --name-only` or
`git status --porcelain`. A Builder that omits a sensitive file from its
self-reported claims (by mistake, by an incomplete claim, or by a
prompt-injected instruction telling it to under-report) silently defeats the
exact safety net this detector exists to provide, and nothing downstream
notices, because the "independent" detector isn't independent of the worker
it is meant to check.

**Fix:** Feed `queenRiskSignalHitsFromPaths` from an actual `git diff
--name-only` against the phase's base commit (already used elsewhere in this
codebase, e.g. `cmd/porter_cmd.go:712`), in addition to or instead of the
worker's self-reported claims, so the detector cannot be defeated by an
incomplete or dishonest handoff.

### WR-02: PathPatterns matching is a plain substring, not word-boundary — a bare pattern like `"session"` false-positives inside unrelated words

**File:** `cmd/queen_risk_signals.go:89, 198-225`

**Issue:** The phrase matcher (`matchesPhraseAtWordBoundary`) deliberately
requires a boundary on both ends specifically to avoid matching `"token"`
inside `"tokenizer"`. The path-pattern matcher
(`queenRiskSignalHitsFromPaths`) uses plain `strings.Contains(path, p)` with
no boundary check at all, and several of the table's `PathPatterns` are bare
words with no directory-separator anchor: `"session"`, `"credential"`,
`"secrets"`, `"delete"`, `"purge"`, `"billing"`, `"checkout"`, `"payment"`,
`"stripe"` (`queen_risk_signals.go:89,99,123`).

Concretely: the credentials/auth signal's pattern `"session"` matches any path
containing that substring anywhere — including `possession.go` or
`repossession_handler.go` (a plausible file name in, say, a leasing or
lending feature). `"possession"` contains `"session"` starting at index 3
(`po` + `ssession`... `s-e-s-s-i-o-n`), so a completely unrelated file would
force the security reviewer under the wrong signal name (the reason sentence
would say "this touches logins and passwords" when the file has nothing to
do with sessions).

**Fix:** Route `PathPatterns` through the same word/segment-boundary
discipline the phrase table already has for text — e.g. require the pattern
to be preceded/followed by a path separator, `_`, `-`, `.`, or start/end of
string, the way `isWordByte` already draws that line for prose.

### WR-03: The `"auth/"` path pattern is directory-anchored and misses auth-named files that aren't inside an `auth/` directory

**File:** `cmd/queen_risk_signals.go:89`

**Issue:** Inconsistent with WR-02's bare-word patterns, `"auth/"` requires a
literal trailing slash, so it only matches auth code living inside a
directory literally named `auth`. A file like `internal/authHandler.go`,
`cmd/oauth.go`, or `pkg/authMiddleware.go` does not contain `"auth/"` as a
substring (the character after `auth` is a letter or `.`, not `/`), and none
of these plausible file names contain `"login"`, `"session"`, `"credential"`,
or `"secrets"` either — so a real auth-surface file can slip past the
file-based detector entirely, with the change relying solely on the
plan-wording detector to catch it. Given `"token"` is deliberately excluded
from the phrase table too (the documented "token bucket" false-alarm), this
combination widens the file-detector's blind spot for authentication code
specifically.

**Fix:** Add a bare `"auth"` pattern (subject to WR-02's boundary fix) rather
than only the directory-anchored `"auth/"` form, so auth-named files at any
depth are caught the same way session/credential/secrets files already are.

## Info

### IN-01: `queenRiskSignalHitsFromPaths`'s longest-match comparison uses the untrimmed pattern length, not the matched (`p`) length

**File:** `cmd/queen_risk_signals.go:210-219`

**Issue:** `best = pattern` is compared and assigned using the original
`pattern` (loop variable), while the actual substring test runs against `p :=
strings.ToLower(strings.TrimSpace(pattern))`. Since every entry in
`queenRiskSignalTable.PathPatterns` is already trimmed and lowercase, `pattern
== p` always holds today and there is no live bug — but the two variables
diverging is exactly the kind of latent inconsistency that becomes a real bug
the next time someone adds a pattern with incidental whitespace or mixed
case.

**Fix:** Compare and store `p` consistently (`if strings.Contains(path, p) &&
len(p) > len(best) { best = p }`), matching what `queenRiskSignalHits`
already does for phrases.

### IN-02: The doc-numbers test's assertion is a loose numeric substring check

**File:** `cmd/caste_relevance_doc_test.go:204-213`

**Issue:** `TestCasteRelevanceDoc_SpawnBudgetNumbersMatch` correctly asserts
the *code* value against `want` (a real, meaningful check), but the
doc-consistency half — `strings.Contains(content, strconv.Itoa(got))` — only
proves the digit sequence appears somewhere in a long markdown file, not that
it appears as the documented spawn-budget number for that flow. For small
values (3, 4, 5, 6, 8) that will trivially match unrelated numbers elsewhere
in the doc (list markers, header counts, other tables), so this half of the
test would not fail if the doc's specific claim for a given flow/depth
drifted from the code as long as the digit appeared anywhere else on the
page. This is a pre-existing pattern (not introduced by this phase), so it's
noted for awareness rather than requiring action in this phase.

**Fix (optional, future work):** Anchor the check to the specific table row/
line for that flow, or drop the doc-content half in favor of the pre-existing
`docstest`-style structured extraction other doc tests in this repo already
use.

### IN-03: The file-detector's owner-facing sentence names the raw path pattern including its trailing slash

**File:** `cmd/forced_reviewer_waiver.go:62-68`, `cmd/queen_risk_signals.go:307-316`

**Issue:** `forcedReviewerReasonClause` renders a file-detected hit as `the
files changed touched "auth/"` — the literal internal `PathPatterns` string,
trailing slash included, reaches the owner-facing sentence verbatim. This is
a minor plain-English miss relative to this repo's own CLAUDE.md mandate
("translate jargon... every single time"): `"auth/"` reads as an internal
identifier fragment, not a sentence a non-technical owner would write.

**Fix:** Render the matched path pattern without the trailing directory
separator (or word it as "a file in the `auth` folder" rather than quoting
the raw pattern), consistent with how `forcedReviewerReason` already avoids
raw caste/signal identifiers everywhere else.

---

_Reviewed: 2026-08-23T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
