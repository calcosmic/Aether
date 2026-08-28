# Phase 196 — deferred items

Things found while executing this phase that were deliberately NOT done here,
recorded so they are actionable rather than forgotten. Written by plan 196-08.

## D1 — Four workflows spawn workers and record nothing

**Found during:** 196-08, task 1 (the reachability proof).

Phase 196's promise is scoped to building a phase and checking it, and after
this plan both of those record their cost on both lanes (see the lane table in
`196-08-SUMMARY.md`). Four other workflows also spawn workers and file no token
row at all, so their cost is invisible:

| Workflow | Where its workers are spawned |
|---|---|
| `aether plan` | `cmd/codex_plan.go` |
| `aether colonize` | `cmd/codex_colonize.go` |
| `aether seal` (its final review) | `cmd/seal_final_review.go` |
| the automatic fix attempt | `cmd/check_fix_attempt.go` |

`aether quick` and `aether swarm` are in the same position.

**Why it was not done here.** The ledger is keyed by phase AND workflow word,
and the writer refuses any word other than `build` or `continue` rather than
quietly creating a third file that nothing adds up. Adding these means choosing
what a planning run or a seal is keyed under, and whether the cost line — which
says "What This Phase Has Cost" — should include work that is not a phase.
That is a scope decision, not a wiring gap, and it belongs to whoever picks it
up rather than to a closeout plan.

**What to do.** Decide the key first (a new workflow word per workflow, or one
shared `other` key), then extend `normalizeSpendRowStatus`'s sibling check in
`cmd/spend_ledger.go` and add one `writeSpendRowsForRun` call per workflow. The
end-to-end shape to copy is `fileDirectContinueSpendRows` in
`cmd/spend_writer.go`, and `TestSpendPipelineIsReachableEndToEnd` is the test
shape that would prove it.

## D2 — A directly-spawned worker whose provider emits no usage event

**Found during:** 196-08, task 1.

On the direct lanes the provider's own event is the only source of a figure —
there is no chat-platform session artifact, because the worker is a subprocess
of the runtime rather than a subagent inside somebody's chat session. A worker
whose provider emits no usage event therefore shows a dash and stays out of the
total.

That is correct and deliberate under D-01 as amended (no number is better than a
number the owner cannot trust), and it is not a bug. It is recorded here only so
a later reader who sees a dash on the direct lane knows it means "the tool said
nothing", not "the wiring is missing" — the wiring is proved by
`TestSpendPipelineIsReachableEndToEnd`.

---

*Entries D3, D4 and D5 were added on 2026-08-28, after verification. WR-04,
WR-05 and WR-06 were raised by the code review at 14:51 and carried as "deferred
by agreement" through review iterations 2 and 3, but this file was written at
14:26 and nothing ever came back to it. They existed only inside the review
documents until now. An item left undone on purpose and not written down is
indistinguishable from an item forgotten, which is the bookkeeping failure this
repository's own audits keep finding.*

## D3 — What is left of WR-04 after the closeout deletions

**Found during:** the 196 code review (WR-04); closed in part by the closeout
commits of 2026-08-28.

**Closed, not deferred.** Three symbols added by this phase had no production
caller — `spendPerWorkerAverageTokens`, `spendRollupByParent` and its
`spendParentRollup` type — and `spendRollupByParent` grouped on
`spendRow.ParentName`, a field `writeSpendRowsForRun` never set, so against a
real ledger it would have returned a single `(unattributed)` bucket. Its test
seeded the field by hand and therefore passed over a shape production cannot
emit. All three, plus `spendRow.ParentName` and `spendRow.ToolCount` (declared,
never written, never read), were deleted. The ledger had not had its first
production write, so no schema bump and no migration was owed — the same
argument plan 196-01 used when it ADDED `JobName`. `cmd/spend_ledger.go` now
carries a comment on `spendRow` recording that the absence of both fields is
deliberate.

**Also closed, by an earlier fix:** WR-04's description of `spendLedger.RunID`
as "declared, never written, never read" is stale. The retry fix (CR-02, commit
`1202bcab`) made `RunID` load-bearing: `writeSpendRowsForRun` stamps every row
with the attempt's run id and reads it back to tell a re-finalize of one attempt
(replace its rows) from a second attempt at the phase (add to them). Before that
field existed, a retry erased everything the first attempt spent.

**What remains deferred:** nothing of WR-04's substance. What remains is the
question WR-04 raised in passing and this phase did not answer — *should* a
ledger row carry which worker spawned it? A per-parent roll-up is a genuinely
useful view ("the Queen's own workers cost X, the Auditor's sub-workers cost
Y"), and `agent.SpawnEntry.ParentName` already holds the fact at spawn time. It
was deleted rather than wired because wiring it is a feature decision with a
display surface attached, not a gap left in this phase's own promise.

**What to do if it is picked up.** Populate the field at write time in
`writeSpendRowsForRun` (`cmd/spend_writer.go`) from the spawn entry, not at read
time; add the roll-up to `aether spend` behind its own heading; and assert in
`TestBuildFinalizeFilesTheRunsRows` that filed rows carry a non-empty parent, so
the field can never again exist without a writer.
`TestNothingThisPhaseAddedIsUncalled` now derives its own inventory and will
fail on the day a re-added field or function has no production user, so this
cannot silently regress.

## D4 — The team card states a per-worker model on OpenCode, where nothing routes by role

**Found during:** the 196 code review (WR-05). Confirmed still live at
verification.

**What is wrong.** Every line of the pre-build team check-in card ends with
either *"(kept on the more expensive model because it …)"* or *"(the cheaper
model)"*, from `resolveCasteModel` → `casteModelSlot`
(`cmd/ceremony_team_checkin.go`, `cmd/codex_visuals.go`), which mirrors the
`model:` line in `.claude/agents/ant/aether-<role>.md`.

`.opencode/commands/ant/build.md` renders that same card from the same runtime
renderer, and **no file under `.opencode/agents/` carries a `model:` line at
all** — all 27 checked. On OpenCode the model is whatever the user has
configured, so on that platform the card asserts a routing fact that is not
true, in the one surface whose entire purpose is telling a non-technical owner
what he is about to pay for. `TestCasteModelSlotMatchesAgentFrontmatter`
(`cmd/caste_model_test.go:147`) reads `.claude/` only, so nothing detects the
divergence.

CLAUDE.md's Platform Policy names OpenCode a primary platform, and its
Definition of Done requires that a documentation claim about runtime behaviour
be testable or removed — the same rule applied to a runtime-rendered claim.

**Why it was not done here.** It is a cross-platform behaviour change to a
surface the owner reads, not a defect inside this phase's own promise
(criterion 4 is scoped to the card naming the model and the reason, which it
does). The two honest fixes point in different directions — suppress the clause
on platforms that do not route by role, or give OpenCode's agents real `model:`
frontmatter so the claim becomes true — and choosing between them is a platform
decision.

**What to do.** Either (a) gate the model clause on the platform actually
routing by role:

```go
if !platformRoutesModelsByRole(buildHostPlatform()) {
    // no model clause: the platform does not route per role, so naming one
    // would assert something this card cannot know.
}
```

or (b) add `model:` frontmatter to all 27 `.opencode/agents/*.md` to match the
Claude table. Either way, extend `TestCasteModelSlotMatchesAgentFrontmatter` to
read `.opencode/agents/` as well, so the two platforms can never silently
disagree again.

## D5 — The closeout cost-line tests never drive the branch real runs take

**Found during:** the 196 code review (WR-06). Re-measured at verification, and
**recorded here at a corrected severity.**

**What is wrong — precisely.** `renderCeremonyCloseout` resolves which phase's
cost to show as `completion_phase`, falling back to `current_phase`
(`cmd/ceremony_cmd.go:610-613, 638-642`). `completion_phase` is set only when
the completion packet carries a manifest key `closeoutManifest` recognises.
`costLineCompletionFileForTest` (`cmd/ceremony_closeout_spend_test.go:67-77`)
writes a packet with a bare top-level `"phase": 1` and no manifest key, so every
one of `TestBuildEndsWithOneCostLine`, `TestContinueEndsWithOneCostLine`,
`TestCloseoutCostLineCountIsAssertedNotAssumed` and
`TestNoLaneRendersTwoCostLines` exercises only the fallback.

**Corrected severity: untested, not broken.** The review inferred the production
branch might be wrong. Verification ran it by hand with the real binary against
a packet carrying a real `dispatch_manifest`, with the colony's `current_phase`
deliberately advanced to 2, and it correctly rendered **phase 1**. The branch
works.

What was measured, and is the actual defect: **deleting the `completion_phase`
branch entirely leaves the whole spend / cost / closeout / end-to-end test set
green** (mutation M3 in `196-VERIFICATION.md`). So criterion 1's guarantee —
"every build and continue ends with one honest cost line" — is proved by
`TestBuildEndsWithOneCostLine` **only on the fallback route**. The route real
runs take is unlocked: nothing fails if it is broken or removed.

The fallback IS wrong when reached — `cmd/closeout_cmd.go:46` reads
`state.CurrentPhase` after the check has advanced the colony, so a manifest-less
packet renders the next phase's block. Production does not normally reach it:
`build-finalize` hard-errors on a packet without `dispatch_manifest`
(`cmd/codex_build_finalize.go:379`), the heavy continue closeout always carries
`continue_manifest`, and the default continue path prints its own ending screen,
which IS driven end to end by `TestNoLaneRendersTwoCostLines`.

**Why it was not done here.** It is a test-fidelity fix rather than a
behaviour fix, and the behaviour it would lock was independently confirmed
correct by running it.

**What to do.** Give `costLineCompletionFileForTest` a real `dispatch_manifest`
(and its continue sibling a `continue_manifest`) so the existing four tests
drive the production branch, and add one case asserting the cost block names the
phase that was just run when `current_phase` has already advanced past it. The
proof that it worked is the same mutation: delete the `completion_phase` branch
and a named test must fail.
