# Deferred Items — Phase 202.1 Classic Visual Voice

## Pre-existing failure discovered during Plan 01 Task 1 verification

**Test:** `TestNextActionNeverHardcoded` (named in Plan 01 Task 1's `<verify>` block)

**Finding:** `cmd/swarm_cmd.go:runSwarmDestroy` (line ~552) hand-writes a
`"next": "aether status"` literal into a result map instead of routing it
through `resolveNextAction`/`applyNextActionToResult`. This is caught by the
hardcode ratchet as a new, unrecorded hand-typed command-advice site.

**Confirmed out of scope for this plan:**
- `cmd/swarm_cmd.go` carries zero diff against this plan's base commit
  (`2757534e2b8f35d4099ac1307786d3ed02598698`) — this executor made no
  change to that file.
- `cmd/swarm_cmd.go` is not in Plan 01's declared `files_modified` list, and
  the fix (routing swarm's repair-not-restored result through the shared
  next-action resolver) requires understanding `runSwarmDestroy`'s
  intervention/repair contract, which is outside this phase's research
  domain (Classic visual voice/glyph density).
- `cmd/testdata/next_action_hardcode_baseline.json` was last touched in
  Phase 200 plan 54, well before the swarm repair code that introduced this
  literal — confirming the drift predates Phase 202.1 entirely.

**Action:** Not auto-fixed (Scope Boundary — pre-existing failure in an
unrelated file). Every other test named in Plan 01 Task 1's `<verify>` block
passes. Route this through the normal continue/verify cycle for
`swarm_cmd.go`'s owning phase, or open a follow-up plan.

---

## Known-red baseline at phase start (recorded 2026-09-12, orchestrator)

Verified by checking out the phase base commit `2757534e` into a scratch
worktree and running these tests there. All four fail identically at that
commit, before any Phase 202.1 work existed. They are NOT caused by this
phase and must NOT be absorbed into it.

| Test | Failure at base |
|---|---|
| `TestCurrentVocabulary199` (`tracked-occurrences-are-exhaustively-classified`) | 195 tracked keys vs 193 inventory keys; `199-PATTERNS.md` `legacy_pause`/`legacy_resume` unclassified |
| `TestGoldenBuildVisualOutput` | golden stale: output emits a `── Colony ──` block the snapshot does not carry |
| `TestGoldenContinueVisualOutput` | same stale `── Colony ──` drift |
| `TestPhase199GateReceipt` | protected ownership fingerprint changed or receipt is stale |
| `TestAuditCatalogGolden` | catalog golden stale: got 234193 bytes, want 234067 |
| `TestNoWorkerWithoutStatedReason` | `measurer had a stated reason and must spawn; castes = [builder]` |
| `TestHumanFacingOutputGoesThroughWriteVisualOutput` | `watch_live.go:runColonyLiveRefreshLoop` lines 425/426/440 write direct to stdout/stderr, bypassing `writeVisualOutput` |

The last three were added on a second pass: the first full-suite reading was
truncated by a `head -40` on the orchestrator's own grep and under-reported the
set. All seven were re-run at `2757534e` and fail there with byte-identical
messages.

**`TestNoWorkerWithoutStatedReason` is worth the owner's attention separately.**
It is one of the tests CLAUDE.md names as locking the Queen-Owned Orchestration
contract, and it is currently red on this branch.

**Gate rule for the remaining waves of 202.1:** the post-merge test gate is
judged against this set. Only a failure *outside* these four counts as
breakage introduced by a wave. The two golden snapshots are deliberately NOT
refreshed here — refreshing them would silently absorb pre-existing drift
into this phase's diff.

## Fixed during Wave 1 close-out (orchestrator)

1. `cmd/swarm_cmd.go` — the hardcoded next-action literal above. Fixed;
   `TestNextActionNeverHardcoded` failed before and passes after.
2. Plan 01 placed the next-step glyph *between* the `Choice: ` label and the
   command (`Choice: ➡️ Run ...`), breaking the existing guardrail in
   `golden_workflow_test.go` that requires `Choice: Run \`/ant-build 2\`` to
   appear intact — violating plan 01's own must-have that the existing
   next-action guardrails pass unmodified. Split `nextActionSuggestionBody`
   out of `nextActionSuggestionLine` so the glyph leads the whole line
   (`➡️ Choice: Run ...`) with `voiceLine` still the single funnel.
3. Plan 01 gave the Goal line the `phase` glyph (📍). The owner ratified
   crown = the project's goal, and the February reference block this phase
   restores starts literally `👑 Goal:`. Changed to the `goal` glyph.

## Orchestration lesson (recorded 2026-09-12)

Do NOT run the whole-suite gate while executor agents are running. This repo's
full-suite controller uses a `serial-shared-checkout` lane, and executors in
their own worktrees still share the repo root and the `~/.aether/` hub. A
confirmation run started alongside four Wave 2 executors reported
`full-suite controller failed: lane serial-shared-checkout: exit status 1` and
produced two failures that pass in isolation (`TestSkillIsUserCreatedShipped`
and the controller lane itself). Gate runs must be serialised against dispatch.
