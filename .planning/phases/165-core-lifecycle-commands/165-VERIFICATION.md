---
phase: 165-core-lifecycle-commands
verified: 2026-08-03T15:00:00Z
status: gaps_found
score: 9/10 must-haves verified
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 9/10
  gaps_closed:
    - "init.md's Shelf Backlog stage no longer instructs a hand-write to protected state (session.json / COLONY_STATE.json) — confirmed removed from all three surfaces (.claude, .opencode, flat mirror) and fenced by TestInitWrapperCeremonyContract's expanded forbidden list"
  gaps_remaining: []
  regressions:
    - "New Critical finding from the gap-closure re-review (CR-01, dated 2026-08-03T12:05:11Z): the runtime fix for the old gap introduced a new ordering hazard — shelf-promote-batch now runs and persists to shelf.json during the Shelf Backlog stage, before user consent at Approval, so a cancel/goal-revision/failed-init strands promoted entries and contradicts init.md's own 'write nothing on cancel/failure' claim"
gaps:
  - truth: "A promoted shelf entry's persistence is gated on the same consent boundary as everything else init.md writes ('write nothing on cancel/failure'), consistent with the ordering fix already applied to pheromone writes in the same Approval stage"
    status: failed
    reason: >
      init.md's `## Shelf Backlog` stage (`.claude/commands/ant/init.md:241-262`,
      byte-identical in `.opencode/commands/ant/init.md` and the flat mirror
      `.claude/commands/ant-init.md`) runs "Before colony state creation" and,
      per line 255, calls `aether shelf-promote-batch --ids ... --colony "{goal}"`
      directly — well before the `## Approval` stage (line 272) where the user is
      actually asked to proceed, revise the goal, or cancel. Independently traced
      in cmd/shelf_init.go: `promoteShelfEntry` (line 124) immediately flips the
      entry's `Status` to `promoted` and writes `shelf.json` on that call, and
      `loadActiveShelf` (line 108) filters to `Status == ShelfShelved`, so a
      promoted-then-cancelled entry vanishes from the shelved list with no colony
      ever created to carry it forward. `promotedShelfTodos` (line 171) matches
      strictly on `e.PromotedTo == goal`, and Approval's "revise goal" option can
      change the goal string used to call `aether init` — a revision after Shelf
      Backlog silently orphans the same entries under their old `PromotedTo`
      value. init.md's own `<failure_modes>` block still states for "User Cancels
      At Approval": "Write nothing — no charter call, no pheromone writes,
      nothing persisted" — this is now false: shelf.json is already mutated by
      that point. This is the same ordering principle plan 165-07 itself applied
      to pheromone writes in the same Approval stage (gated on `aether init`
      success, closing the prior WR-05 finding) but did not apply to the shelf
      writes it introduced in the same change. Confirmed via direct code read of
      the current working tree (not a SUMMARY claim); no test in
      cmd/shelf_todo_wiring_test.go or cmd/init_wrapper_ceremony_test.go exercises
      a cancel-after-promote or revise-goal-after-promote scenario. First
      identified in 165-REVIEW.md (2026-08-03T12:05:11Z, CR-01) as a re-review of
      165-07/165-08's gap-closure work.
    artifacts:
      - path: ".claude/commands/ant/init.md"
        issue: "Shelf Backlog stage (lines 241-262) mutates shelf.json before Approval's consent gate (line 272); no reordering or deferral applied"
      - path: ".opencode/commands/ant/init.md"
        issue: "Same instruction, same ordering, byte-identical"
      - path: ".claude/commands/ant-init.md"
        issue: "Flat mirror carries the same ordering"
      - path: "cmd/shelf_init.go"
        issue: "promoteShelfEntry (line 124) writes shelf.json unconditionally on call, with no deferred/staged mode; promotedShelfTodos (line 171) matches on an exact, mutable goal string"
      - path: "cmd/shelf_todo_wiring_test.go"
        issue: "No test exercises cancel-after-promote or revise-goal-after-promote; every test promotes and inits with the identical literal goal"
    missing:
      - "Make shelf promotion atomic with init success, matching the pattern already used for pheromone writes: either move the shelf-promote-batch/shelf-dismiss-batch calls to the Approval stage after aether init returns success (using refined_goal explicitly), or have aether init itself accept the chosen shelf IDs and perform the promotion server-side after colony state is created"
      - "A wiring test that promotes shelf entries, then cancels or inits with a different (revised) goal than the one used at promotion time, and asserts the entries are not silently stranded"
      - "Update init.md's <failure_modes> 'User Cancels At Approval' text to match actual behavior once fixed, or fix the ordering so the existing claim is true again"
deferred: []
---

# Phase 165: Core Lifecycle Commands Verification Report

**Phase Goal:** `init`, `plan`, `build`, and `continue` wrappers are rewritten to carry engineering method — stage purpose, files to read, spawn choreography, stop conditions — instead of protocol instructions whose primary job is parsing `result.manifest.dispatch_manifest`. Phase 165 is the sole structural owner of `build.md` for this milestone.
**Verified:** 2026-08-03
**Status:** gaps_found
**Re-verification:** Yes — after gap-closure plans 165-07 (shelf-to-todo runtime wiring, init.md hand-write fence, pheromone ordering) and 165-08 (read/write-boundary reconciliation, Clarification Gate ordering)

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Reading build.md/plan.md/continue.md/init.md shows stage purpose, files-to-read guidance, spawn choreography, and stop conditions (ROADMAP SC1, CMD-01) | ✓ VERIFIED | Unchanged from prior pass; `**Purpose:**`/`**Reads:**`/`**Stop conditions:**` blocks intact; `TestBuildWrapperStageSkeletonAndParity`, `TestContinueWrapperStageSkeletonAndParity`, `TestPlanWrapperStageSkeleton`, `TestInitWrapperStageSkeletonAndParity` all PASS (re-run this pass) |
| 2 | Grepping the four wrappers for envelope-parsing/manifest-file-writing as primary job returns effectively zero (ROADMAP SC2, CMD-02) | ✓ VERIFIED | `TestLifecycleWrappersDoNotParseEnvelopeAsPrimaryJob` PASSES; no plan 07/08 change touched envelope-parsing prose |
| 3 | A user reading build.md can describe each stage's purpose and worker inputs without opening Go source (ROADMAP SC3, CMD-03) | ✓ VERIFIED | Recorded human checkpoint in 165-VALIDATION.md (dated 2026-08-03, verdict "Approved") predates and is unaffected by 165-07/08's changes, which did not touch build.md's stage prose (only its `<read_only>` block wording) |
| 4 | `/ant-chaos`, `/ant-archaeology`, `/ant-dream`, `/ant-oracle`, `/ant-swarm`, `/ant-sage`, `/ant-colonize`, `/ant-council` continue to work unchanged (ROADMAP SC4, CMD-04) | ✓ VERIFIED | `TestSpecialistCommandSurfacesUnchanged` re-run this pass, PASSES (17-file SHA-256 ledger + command-guide reachability); 165-07/08 touched only init/build/continue/plan surfaces, none of the 8 specialist commands |
| 5 | build.md's structural rewrite is committed by Phase 165 alone; ownership/merge order is traceable in the file (ROADMAP SC5, CMD-05) | ✓ VERIFIED | `TestBuildMdOwnershipHandshake` re-run this pass, PASSES; ownership comment and trailer marker unchanged by 165-08 (which only edited the `<read_only>` block, not the ownership header) |
| 6 | init.md never instructs a hand-write to protected state (session.json / COLONY_STATE.json), consistent with its own `<read_only>` block and the plan's CRITICAL D-2 fence | ✓ VERIFIED (gap closed) | The line 257 hand-append phrasing ("append `[shelf:{category}] {text}` to `active_todos` in the session file or colony state") is gone from all three surfaces (`.claude`, `.opencode`, flat mirror) — confirmed by direct grep returning zero matches. Replaced with a description of the runtime `todos` array from `shelf-promote-batch`. `shelfEntryToTodo` (cmd/shelf_init.go) now has a real runtime call site (`promotedShelfTodos`), not just test call sites. `TestInitWrapperCeremonyContract`'s forbidden list gained the exact phrasing and passes |
| 7 | init.md writes pheromones only after `aether init` succeeds, matching its own "write nothing on cancel/failure" claim | ✓ VERIFIED (literal) / ⚠️ Claim still false for shelf writes | `.claude/commands/ant/init.md:283-284` confirms `aether init` runs before `aether pheromone-write`, gated on success ("If `aether init` failed, skip this step entirely"); `TestInitWrapperCeremonyContract/approval_writes_pheromones_only_after_init_succeeds` PASSES. However, the wrapper's own broader claim ("write nothing on cancel/failure") is still false in practice because of the new Truth 8 gap below — the fix was applied correctly to pheromones but not extended to shelf-promote-batch, which the same plan introduced |
| 8 | A promoted shelf entry's persistence is gated on the same consent boundary as init's other writes, so a cancel/revised-goal/failed-init cannot strand it | ✗ FAILED | See Gaps below (fresh code-review CR-01, independently confirmed by direct code trace this pass) |
| 9 | build/continue's `<read_only>` blocks agree with their own Guardrails lists (prior review WR-01) | ✓ VERIFIED (gap closed) | Both now read "This wrapper never reads or writes, by hand: ..." matching plan.md's wording; `TestLifecycleWrapperReadOnlyBlocksAreConsistent` PASSES across all 12 surfaces (4 verbs × 3 surfaces) |
| 10 | plan.md's Clarification Gate runs before Decision Moment 2 mutates research-approval state (prior review WR-04) | ✓ VERIFIED (gap closed) | `## Clarification Gate` now precedes `## Decision Moment 2 — Research Batch` in file order (confirmed by grep line numbers: 50 then 63); `TestPlanWrapperCeremonyContract`'s `inOrder` assertion updated and passes |

**Score:** 9/10 truths verified (Truth 8 is a new gap surfaced by this re-verification's re-review, replacing the prior pass's Truth 10 which is now closed)

### Deferred Items

None. CR-01 is not addressed by any later milestone phase — it is a direct, unresolved consequence of this phase's own gap-closure work (165-07) and must be closed within this phase's scope.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `cmd/shelf_init.go` | `shelfTodoPrefix`, `promotedShelfTodos`, `mergeShelfTodos`, `todos` array on `shelf-promote-batch` | ✓ VERIFIED | All present and match plan 07's must_haves; `promoteShelfEntry` confirmed to write unconditionally on call (this is the root of the new gap, not a missing-artifact issue) |
| `cmd/init_cmd.go` | seeds `session.ActiveTodos` via `promotedShelfTodos(store, goal)` | ✓ VERIFIED | `promotedShelfTodos(store, goal)` call confirmed present |
| `cmd/shelf_todo_wiring_test.go` | 4 named tests proving promote→init→session.json→refresh chain | ✓ VERIFIED | All 4 exported tests exist and PASS; does not cover the cancel/revise-goal scenario (see gap) |
| `cmd/init_wrapper_ceremony_test.go` | Forbidden-string fence + ordering assertion | ✓ VERIFIED | `active_todos` and related phrases fenced; `approval_writes_pheromones_only_after_init_succeeds` subtest passes; does not fence CR-01's promote-before-consent ordering |
| `.claude/commands/ant/init.md` (+ mirrors) | Shelf Backlog routed through runtime; Approval pheromone gating | ✓ VERIFIED (content), ⚠️ ordering hazard remains | Hand-write phrasing gone; pheromone gating correct; shelf-promote-batch call itself still unconditionally precedes consent |
| `cmd/lifecycle_wrapper_contract_test.go` | `TestLifecycleWrapperReadOnlyBlocksAreConsistent` (new in 165-08) | ✓ VERIFIED | Exists, covers 12 surfaces, PASSES |
| `.claude/commands/ant/build.md`, `.claude/commands/ant/continue.md` (+ mirrors) | `<read_only>` reconciled with Guardrails | ✓ VERIFIED | Wording matches plan.md's model exactly |
| `.claude/commands/ant/plan.md` (+ mirrors) | Clarification Gate relocated ahead of Decision Moment 2 | ✓ VERIFIED | Confirmed by heading order and passing ordering test |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `cmd/init_cmd.go` | `cmd/shelf_init.go promotedShelfTodos` | `session.ActiveTodos` seeding at colony creation | ✓ WIRED | Confirmed by grep + passing `TestInitSeedsSessionTodosFromPromotedShelf` |
| `cmd/recovery_snapshot.go` / `cmd/session_cmds.go` | `cmd/shelf_init.go mergeShelfTodos` | session refresh preserves shelf-prefixed todos | ✓ WIRED | Confirmed by grep + passing `TestSessionRefreshPreservesShelfTodos` |
| `.claude/commands/ant/init.md` Shelf Backlog stage | `cmd/shelf_init.go promoteShelfEntry` | `aether shelf-promote-batch` call | ⚠️ WIRED BUT MISTIMED | The link exists and functions, but fires before the Approval consent gate rather than after — this is the substance of the CR-01 gap, not a missing link |
| `cmd/init_wrapper_ceremony_test.go` | canonical + `.opencode` + flat-mirror init.md | forbidden-string fence | ✓ WIRED | Confirmed passing for the closed gap; does not cover the new one |
| `cmd/lifecycle_wrapper_contract_test.go` | `.claude`/`.opencode`/flat-mirror build/continue/plan/init.md | `read_only_block_matches_guardrails` subtest | ✓ WIRED | Confirmed passing across all 12 surfaces |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Full lifecycle/ceremony/shelf test suite compiles and passes | `go test ./cmd/... -run 'TestLifecycle\|TestShelf\|TestInitSeeds\|TestSessionRefreshPreserves\|TestMergeShelfTodos\|TestInitWrapper\|TestBuildWrapper\|TestContinueWrapper\|TestPlanWrapper\|TestBuildMdOwnership\|TestBriefPath\|TestSpecialistCommand\|TestWrapperHostContract' -count=1` | All PASS, 0 failures | ✓ PASS |
| `go build ./cmd/aether` succeeds | `go build ./cmd/aether` | exit 0, no output | ✓ PASS |
| Old hand-append phrasing absent from all init.md surfaces | `grep -n "append \`\[shelf:\|to \`active_todos\`" .claude/commands/ant/init.md .opencode/commands/ant/init.md .claude/commands/ant-init.md` | 0 matches | ✓ PASS |
| `<read_only>` blocks match across build.md/continue.md | direct read of both files | Identical wording ("never reads or writes, by hand") | ✓ PASS |
| Clarification Gate precedes Decision Moment 2 in plan.md | `grep -n "^## Clarification Gate\|^## Decision Moment 2"` | Clarification Gate at line 50, Decision Moment 2 at line 63 | ✓ PASS |
| All named gap-closure commits (165-07, 165-08) present in history | `git log --oneline -1 <hash>` for 9 hashes | All 9 found | ✓ PASS |
| CR-01 ordering hazard reproducible by code trace | Read `init.md` Shelf Backlog (line 241 "Before colony state creation") vs. Approval (line 272); read `promoteShelfEntry`/`promotedShelfTodos` in `cmd/shelf_init.go` | Confirms unconditional write on Shelf Backlog call, exact-string match on goal | ✗ FAIL (confirms gap) |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| CMD-01 | 165-02..05, 165-06, 165-07, 165-08 | Wrappers carry engineering method | ✓ SATISFIED | Stage-skeleton tests pass on all 4 wrappers; gap-closure plans did not regress this |
| CMD-02 | 165-01, 165-06 | No wrapper's primary job is envelope-parsing | ✓ SATISFIED | Ratio invariant + singular contract pointer tests pass, untouched by 165-07/08 |
| CMD-03 | 165-06 | build.md describable stage-by-stage without opening Go source | ✓ SATISFIED | Dated human verdict predates and is unaffected by 165-07/08 |
| CMD-04 | 165-06 | 8 specialist/delight commands keep working unchanged | ✓ SATISFIED | SHA-256 ledger unaffected; 165-07/08 touched none of the 8 surfaces |
| CMD-05 | 165-02 | build.md sole structural owner, merge order traceable | ✓ SATISFIED | Ownership handshake test passes; 165-08 only touched the `<read_only>` block, not the ownership header |

All five CMD IDs remain literally satisfied by their formal test coverage. As in the prior verification pass, this report flags a phase-level gap (Truth 8 / CR-01) that is not literally covered by CMD-01..05's wording but falls squarely within this project's Definition of Done: an explicit documentation claim ("write nothing on cancel/failure") in a file this phase owns is demonstrably false in the current working tree, the underlying mechanism is the exact "state mutated before consent, keyed on a value that can still change" pattern this project has hit before, and it was introduced by this phase's own gap-closure work (165-07) while fixing a sibling instance of the identical principle (pheromone-write ordering) in the same stage.

REQUIREMENTS.md's traceability table still shows CMD-01..05 as `[ ]` / "Pending" — expected, not a gap; updated only once the phase is signed off (consistent with the prior verification's note).

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `.claude/commands/ant/init.md` (+ `.opencode`, flat mirror) | 241-262 vs 272-291 | Shelf-promote-batch/dismiss-batch calls persist to shelf.json before the Approval consent gate; contradicts the file's own "write nothing on cancel/failure" claim | 🛑 Blocker | CR-01 (fresh review, 2026-08-03T12:05:11Z), independently confirmed this pass — see Truth 8 / gap |
| `cmd/init_ceremony.go` | 561 | `createCeremonyColony` (the `aether init-ceremony` / Codex-facing path) hard-codes `ActiveTodos: []string{}`, so the shelf-to-todo fix (165-07) only covers one of two colony-creation paths | ⚠️ Warning | review WR-02; does not affect CMD-04 (init-ceremony isn't one of the 8 named specialist commands) but is a silent capability gap on a parallel path |
| `cmd/recovery_snapshot.go` | 417, 575 | CONTEXT.md/HANDOFF.md recompute tasks from state and ignore the merged `session.ActiveTodos`, so shelf-seeded todos never appear in recovery documents | ⚠️ Warning | review WR-03; the durable data exists but isn't surfaced in the two documents meant to carry it forward |
| `cmd/shelf_todo_wiring_test.go` | 30-31, 93-94, 139-140 | `defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))` evaluates the restore value immediately, leaking a deleted temp dir into `AETHER_ROOT` for the rest of the package test run | ⚠️ Warning | review WR-04 (test-only correctness issue, no production impact) |
| `cmd/shelf_init.go` | 43-63 | `shelf-promote-batch` returns `ok: true` even when every requested ID fails; wrapper only surfaces `todos`, not `failed` | ⚠️ Warning | review WR-05; total failure renders as silent success |
| `cmd/lifecycle_wrapper_contract_test.go` | 129-134 | Stale doc comment describes a test as "currently RED for build and init" when it now passes for all four verbs | ⚠️ Warning | review WR-06; misleading to future readers, no functional impact |

### Human Verification Required

None outstanding. CMD-03's manual read-through was already discharged by a recorded, dated, blocking human-verify checkpoint (165-06), unaffected by the gap-closure plans. The remaining gap (Truth 8 / CR-01) is a code-traceable defect, not a subjective UX judgment call, and does not require human testing to confirm.

## Gaps Summary

**One blocker, newly surfaced by the gap-closure re-review (not present in the prior verification pass, which closed a different init.md defect):** the fix in plan 165-07 that routed the old "hand-append todo" instruction through the runtime introduced a fresh ordering hazard. `init.md`'s Shelf Backlog stage still calls `aether shelf-promote-batch`/`shelf-dismiss-batch` *before* the Approval stage's consent gate ("Before colony state creation," line 241), and `promoteShelfEntry` writes `shelf.json` unconditionally the moment that command runs. 165-07 correctly gated the sibling pheromone-write in the same Approval stage on `aether init` success — closing the prior WR-05 finding — but did not apply the same gating to the shelf-promote-batch call it introduced/left in place. The result: a user who promotes shelf items and then cancels, revises the goal, or hits a failed `aether init` at Approval has already had those entries flipped to `Status: promoted` and removed from the shelved list, with no colony created to carry them forward as todos (`promotedShelfTodos` requires an exact match on the final init goal). This directly contradicts init.md's own `<failure_modes>` claim: "User Cancels At Approval ... Write nothing — no charter call, no pheromone writes, nothing persisted." No test in `cmd/shelf_todo_wiring_test.go` or `cmd/init_wrapper_ceremony_test.go` exercises a cancel-after-promote or revise-goal-after-promote scenario, so nothing currently prevents this from shipping.

This was weighed directly against plan 165-07's own must-have truth ("init.md writes pheromones only after `aether init` succeeds, matching its own 'write nothing on cancel/failure' claim"): that specific truth is satisfied literally (pheromones genuinely are gated correctly), but the broader claim it references — "write nothing on cancel/failure" — is still false in the current tree because of a write plan 165-07 itself introduced and did not gate the same way. Applying the same standard the prior verification pass used for the (now-closed) hand-write gap — a documented behavioral claim that is demonstrably false in the working tree is a phase-level gap regardless of whether any single CMD-01..05 ID's literal wording covers it — this is classified as a blocking gap, not a warning.

**Six warnings** (review WR-01 through WR-06) were independently confirmed in the codebase; none individually blocks the phase's core stage-skeleton/ceremony/ownership deliverables (CMD-01..05 all still pass their formal tests), but WR-02 (second colony-creation path never wired) and WR-05 (silent total-failure success) touch the same class of "capability exists on one path, silently absent or misleading on another" that this project's Definition of Done specifically warns about, and are worth closing alongside the blocker.

**What was confirmed closed this pass:** the prior verification's Truth 10 (init.md's literal hand-append instruction to `active_todos`) is gone from all three surfaces and fenced by a passing test. Prior review WR-01 (read/write-boundary contradiction in build.md/continue.md) and WR-04 (plan.md Clarification Gate ordering) are both fixed and fenced by passing tests (`TestLifecycleWrapperReadOnlyBlocksAreConsistent`, updated `inOrder` assertion in `TestPlanWrapperCeremonyContract`). All CMD-01..05 formal test coverage remains green. Full `go build ./cmd/aether` and the full named lifecycle/ceremony/shelf test set were re-run directly in this verification pass (not taken from SUMMARY claims) and pass.

---

_Verified: 2026-08-03_
_Verifier: Claude (gsd-verifier)_
