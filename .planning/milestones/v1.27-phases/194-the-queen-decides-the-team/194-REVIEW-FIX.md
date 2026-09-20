---
phase: 194-the-queen-decides-the-team
fixed_at: 2026-08-26T22:15:51Z
review_path: .planning/phases/194-the-queen-decides-the-team/194-REVIEW.md
iteration: 6
findings_in_scope: 7
fixed: 7
skipped: 0
status: all_fixed
---

# Phase 194: Code Review Fix Report

**Fixed at:** 2026-08-26T22:15:51Z
**Source review:** `.planning/phases/194-the-queen-decides-the-team/194-REVIEW.md`
**Iteration:** 6

**Summary:**

- Findings in scope: 7 (6 Critical, 1 Warning)
- Fixed: 7
- Skipped: 0

In plain English: protected reviewer waivers can no longer be approved through a
generic side door, dispatch stops if it cannot durably close the waiver window,
and every decline command already shown to the owner remains usable. Explicit
reviewer proposals now survive normalization, wrapper options cannot corrupt the
phase sent to spawn logging, Codex recovery text uses commands Codex supports,
and the relevance reference matches the live policy.

## Fixed Issues

### CR-01: The public generic resolver bypasses the owner waiver capability

**Status:** Fixed — requires human verification (security/authorization logic)

**Files modified:** `cmd/pending_decision.go`,
`cmd/forced_reviewer_waiver_test.go`

**Commit:** `d97d4443`

**Applied fix:** The generic pending-decision resolver now refuses
`forced-reviewer-waiver` rows and directs the caller to the protected
`decision-answer` path. Generic decision listings redact the attempt identifier
and both scalar and plural capability digests while keeping the decision ID
observable.

**Fail-before evidence:** New end-to-end regression
`TestGenericPendingDecisionResolverCannotWaiveForcedReviewer` failed at
`forced_reviewer_waiver_test.go:404`: the generic command resolved the protected
row without the displayed capability.

**Pass-after evidence:** The same regression and the focused pending-decision /
forced-reviewer set passed (16 tests).

### CR-02: Dispatch proceeds when persistence fails to close the waiver window

**Status:** Fixed — requires human verification (security/dispatch-state logic)

**Files modified:** `cmd/forced_reviewer_waiver.go`, `cmd/spawn.go`,
`cmd/forced_reviewer_waiver_test.go`

**Commit:** `82ea2ee2`

**Applied fix:** Closing the dispatch window now atomically persists its earliest
durable marker and returns persistence errors. `spawn-log` closes that marker
before recording a worker and fails closed when persistence fails; pending-row
cleanup remains a best-effort second layer.

**Fail-before evidence:** Fault-injection regression
`TestSpawnLogFailsClosedWhenWaiverWindowCannotPersist` failed because
`spawn-log` succeeded even though the durable marker path could not be written.

**Pass-after evidence:** The new regression plus dispatch-window closure tests
passed (4 tests).

### CR-03: Re-rendering a check-in invalidates the decline command already shown

**Status:** Fixed — requires human verification (capability lifecycle logic)

**Files modified:** `cmd/forced_reviewer_waiver.go`,
`cmd/forced_reviewer_waiver_test.go`, `cmd/pending_decision.go`

**Commit:** `d053f48c`

**Applied fix:** Re-rendering the same phase/attempt/signal keeps one pending
decision and appends the new capability digest instead of replacing the old one.
Every displayed raw capability remains valid, raw capabilities are never
persisted, and the older scalar digest remains backward compatible.

**Fail-before evidence:** New regression
`TestRepeatedCheckinRenderKeepsFirstWaiverCapabilityLive` failed because the
second render invalidated the first displayed command.

**Pass-after evidence:** The complete focused waiver set passed (16 tests).

### CR-04: Valid aliased or comma-separated reviewer proposals are removed at the final continue boundary

**Status:** Fixed — requires human verification (reviewer reconciliation logic)

**Files modified:** `cmd/codex_continue.go`,
`cmd/forced_reviewer_waiver_test.go`

**Commit:** `f5dcd2ca`

**Applied fix:** Final reconciliation now builds its proposal-preservation set
through the same canonical normalization used by the rest of the Queen flow, so
aliases such as `security` and comma-packed values such as
`gatekeeper,auditor` preserve their canonical reviewers.

**Fail-before evidence:** Both subtests of
`TestWaiverReconciliationPreservesCanonicalExplicitProposal` failed because
Gatekeeper was removed for aliased and comma-packed proposals.

**Pass-after evidence:** The new table regression and related continue-waiver
tests passed (7 tests).

### CR-05: Wrapper options can prevent the dispatch window from ever closing

**Status:** Fixed

**Files modified:** `.claude/commands/ant-build.md`,
`.claude/commands/ant/build.md`, `.opencode/commands/ant/build.md`,
`cmd/queen_judgement_test.go`

**Commit:** `4c18e170`

**Applied fix:** All three byte-identical build wrappers pass the trusted numeric
`<phase_id>` cross-stage value to `spawn-log --phase`; raw `$ARGUMENTS`
never reaches that command.

**Fail-before evidence:** New wrapper expansion regression
`TestBuildWrapperSpawnLogUsesTrustedPhaseID` showed options from
`1 --verification-depth heavy` leaking into the spawn-log command.

**Pass-after evidence:** The regression and related wrapper test passed (2
tests), and `cmp` confirmed all three wrapper copies are identical.

### CR-06: Blocked Codex output still recommends unsupported slash commands

**Status:** Fixed

**Files modified:** `cmd/codex_continue.go`,
`cmd/codex_continue_finalize.go`, `cmd/codex_visuals.go`,
`cmd/codex_visuals_test.go`, `cmd/critics_bring_solutions_test.go`,
`cmd/gate.go`, `cmd/gate_test.go`

**Commit:** `470e8a33`

**Applied fix:** Blocked and gate recovery guidance is canonicalized as native
`aether ...` commands. Codex keeps those commands, while the existing
translation layer renders platform-appropriate slash commands only for Claude
and OpenCode.

**Fail-before evidence:** The real Codex blocked-output regression and the
synthetic Codex translation regression both failed because `/ant-*` guidance
was present.

**Pass-after evidence:** The broadened blocked/gate recovery set passed (29
tests), including negative Codex assertions and positive Claude/OpenCode
translation assertions.

### WR-01: The source-of-truth relevance reference still documents deleted policies

**Status:** Fixed

**Files modified:**
`.aether/docs/command-playbooks/caste-relevance-reference.md`,
`cmd/caste_relevance_doc_test.go`

**Commit:** `555368b5`

**Applied fix:** The reference now matches live registry keywords, special
scoring rules, required-only fallback behavior, explicit-proposal rules, build
versus continue timing, and current implementation functions. Deleted
production-only, light/standard floor, and automatic optional-worker policies
are no longer described.

**Fail-before evidence:** Strengthened live-policy regression
`TestCasteRelevanceDoc_RegistryAndGatedFallbackMatchPolicy` failed the Auditor
row, Chronicler row, and three retired gated-policy claims.

**Pass-after evidence:** All four `TestCasteRelevanceDoc_*` tests passed.

## Verification Summary

- Finding-specific fail-before/pass-after regressions: all 7 findings covered.
- Combined focused Phase 194 regression command: 20 tests passed in `cmd`.
- `go vet ./...`: pass.
- `go build ./cmd/aether`: pass.
- `go test ./... -count=1 -timeout 900s -p 1` in the mandated
  `/tmp/sv-194-reviewfix-*` isolated worktree: 7,194 passed, 1 failed, 11
  skipped across all 20 packages. The sole failure was
  `pkg/storage.TestResolveAetherRoot_GitFallback`: macOS resolved the worktree
  as `/private/tmp/...` while the assertion compared it with the equivalent
  `/tmp/...` spelling. No Phase 194 code was implicated.
- `go test ./pkg/storage -run '^TestResolveAetherRoot_GitFallback$' -count=1`
  from the canonical `/Users/callumcowie/repos/Aether` path: pass (1 test).
- Canonical-path `go test ./... -count=1 -timeout 900s -p 1` rerun after the
  seven fix commits were fast-forwarded: pass — 7,195 tests passed across all
  20 packages, with no packages or tests omitted.

## Skipped Issues

None.

---

_Fixed: 2026-08-26T22:15:51Z_
_Fixer: the agent (gsd-code-fixer)_
_Iteration: 6_
