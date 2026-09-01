---
phase: 196-see-what-it-cost
plan: 08
subsystem: infra
tags: [spend-ledger, token-usage, reachability, branch-disposition, go, tdd]

requires:
  - phase: 196-see-what-it-cost
    provides: "196-01's spendLedger; 196-03's Claude transcript reader; 196-05's writeSpendRowsForRun and resolveWrapperWorkerUsage; 196-06's `aether spend`; 196-07's renderSpendCostLine and the check's own rows"
provides:
  - "TestSpendPipelineIsReachableEndToEnd — the whole path driven through the command tree, proved by cutting each of its four links in turn"
  - "TestNothingThisPhaseAddedIsUncalled — an anti-orphan check for internal entry points, which the command ratchet cannot see"
  - "The direct in-process lane files its own rows: `aether build <n>` and `aether continue` were inert until now"
  - "wrapperUsageRequest.Attached has a producer — the provider's own measurement at the dispatch boundary now reaches the ledger"
  - "196-BRANCH-DISPOSITION.md and the two tests that fail if a branch, a commit or a reason goes missing, or if any of the three branches still exists"
  - "ROADMAP criterion 1 and COST-01 corrected to the shipped shape, so a verifier cannot mark correct code as failing"
affects: [197, 198, 199]

actuals:
  tokens: 14256
  tasks: 2
  commits: 3

tech-stack:
  added: []
  patterns:
    - "A reachability proof drives the real commands and reads the disk, the view and the screen — never the writer, the resolver or the renderer directly"
    - "Every link in a chain is cut in turn and the real failure recorded; a chain test that passes with a link cut is worse than none"
    - "An anti-orphan check for internal entry points, complementing the command-level ratchet which only sees registered commands"
    - "A disposition record whose test fails on a missing reason AND on a surviving branch, so the record cannot drift from the truth in either direction"

key-files:
  created:
    - cmd/spend_pipeline_e2e_test.go
    - cmd/branch_disposition_test.go
    - .planning/phases/196-see-what-it-cost/196-BRANCH-DISPOSITION.md
    - .planning/phases/196-see-what-it-cost/deferred-items.md
  modified:
    - cmd/codex_build.go
    - cmd/codex_continue.go
    - cmd/spend_writer.go
    - cmd/spend_cost_line.go
    - cmd/testdata/golden_build.txt
    - cmd/testdata/golden_continue.txt
    - .planning/ROADMAP.md
    - .planning/REQUIREMENTS.md

key-decisions:
  - "The direct in-process lane was WIRED rather than deferred. 196-07 left it filing nothing, so the cost block on `aether build <n>` and `aether continue` could only ever read 'no token use was recorded'. That was the more painful half of the gap, because the direct lane is the ONLY lane where a provider's own measurement exists at all — codex.ParseUsage runs at that dispatch boundary and nowhere else — so the one lane that had a real number was the one lane that threw it away. Letting a reachability proof pass over an inert lane is this repository's signature failure, and this phase spent two plans avoiding it."
  - "The usage a worker reported is carried on the dispatch and on the check's flow step as `json:\"-\"` — runtime-owned, never serialized. A completion packet is externally shaped; a serialized usage field would let an outside caller simply ASSERT what a run cost. Every figure in this ledger must be one the Go runtime read for itself. This is the same reasoning CR-03 applied to CompletedTaskIDs in Phase 195."
  - "The direct lanes pass an EMPTY platform to the writer. Their workers are subprocesses of the runtime, not subagents inside somebody's chat session, so there is no session transcript or session store belonging to them; naming a chat platform there would send the resolver hunting through somebody else's sessions and attributing them to this run."
  - "The disposition needed TWO tests, not the one the plan named. A test that reads the record can only prove the record is complete; it passes just as happily while all three branches are still sitting in the repository. The criterion is that the branches are GONE, so a second test asserts exactly that — and also that each recorded commit still resolves, because a deletion is only safe if the work is still reachable by commit."
  - "The bookkeeping steps in a check's flow — the deterministic verification commands, signal housekeeping, the learning pass — are excluded from the ledger by their 'system' caste. They cost no worker tokens, and filing a row for one would put a name in the owner's breakdown that never corresponded to anybody."

patterns-established:
  - "Link-cutting proof: for a test that claims a chain is joined up, cut each link in a backed-up copy, run the test, and record the actual failure text in the summary"
  - "Two-sided disposition test: one test on the written record, one on the world the record describes"

requirements-completed: [COST-05, COST-01, COST-02, COST-03]

coverage:
  - id: D1
    description: "The whole cost path is reachable end to end on the platform-driven lane: a finished build files rows, `aether spend` shows them, and exactly one cost block ends the screen"
    requirement: "COST-03"
    verification:
      - kind: integration
        ref: "cmd/spend_pipeline_e2e_test.go#TestSpendPipelineIsReachableEndToEnd/a_finished_build_files_rows_and_the_detail_view_shows_them"
        status: pass
    human_judgment: false
  - id: D2
    description: "The direct in-process build files rows carrying the provider's own figure, the detail view shows the same figure, and one cost block ends the screen with that total"
    requirement: "COST-01"
    verification:
      - kind: e2e
        ref: "cmd/spend_pipeline_e2e_test.go#TestSpendPipelineIsReachableEndToEnd/the_direct_lane_records_what_its_own_workers_reported"
        status: pass
    human_judgment: false
  - id: D3
    description: "The direct in-process check files its own rows, carrying the provider's figure, without erasing the build's"
    requirement: "COST-02"
    verification:
      - kind: e2e
        ref: "cmd/spend_pipeline_e2e_test.go#TestSpendPipelineIsReachableEndToEnd/the_direct_check_records_what_its_own_reviewers_reported"
        status: pass
    human_judgment: false
  - id: D4
    description: "Cutting any one of the four links — the writer call, the resolver call, the renderer call, the command registration — fails the proof"
    verification:
      - kind: manual_procedural
        ref: "each link cut in a backed-up copy and the test re-run; the real failure text for each is quoted in this summary"
        status: pass
    human_judgment: false
  - id: D5
    description: "Nothing this phase added sits uncalled: every entry point it introduced is invoked from a production file, and the command ratchet's allowlist has not grown"
    verification:
      - kind: unit
        ref: "cmd/spend_pipeline_e2e_test.go#TestNothingThisPhaseAddedIsUncalled"
        status: pass
      - kind: unit
        ref: "cmd/subcommand_reachability_ratchet_test.go#TestOrphanAllowlistOnlyShrinks"
        status: pass
      - kind: unit
        ref: "cmd/subcommand_reachability_ratchet_test.go#TestNoRegisteredSubcommandIsUnreferenced"
        status: pass
    human_judgment: false
  - id: D6
    description: "All three preserved branches are gone, each with its tip commit, verdict and written reason recorded first"
    requirement: "COST-05"
    verification:
      - kind: unit
        ref: "cmd/branch_disposition_test.go#TestBranchDispositionRecordsAllThreeBranches"
        status: pass
      - kind: unit
        ref: "cmd/branch_disposition_test.go#TestDisposedBranchesAreGoneFromTheRepository"
        status: pass
      - kind: manual_procedural
        ref: "git branch --list 'worktree-agent-a47f*' 'worktree-agent-aa57*' 'worktree-agent-a59f*' prints nothing"
        status: pass
    human_judgment: false
  - id: D7
    description: "No written criterion still describes the superseded measured-versus-estimated shape"
    requirement: "COST-01"
    verification:
      - kind: manual_procedural
        ref: "ROADMAP.md Phase 196 criterion 1 and REQUIREMENTS.md COST-01 read for the superseded wording; both corrected with the amendment cited"
        status: pass
    human_judgment: true
    rationale: "Whether a future verifier reads the corrected wording as describing the shipped behaviour is a judgment about prose, not something a test can assert. What IS asserted is the behaviour itself, by the D-01 tests 196-07 landed."

duration: 24 min
completed: 2026-08-28
status: complete
---

# Phase 196 Plan 08: The path proved joined up, and three branches gone Summary

**The whole cost path is now driven end to end through the real commands and proved to fail when any one of its four links is cut — and in proving it, the half of the system that was inert got wired: `aether build <n>` and `aether continue` ran their workers inside the runtime and filed nothing, on the one lane where the provider's own measurement actually exists.**

## Performance

- **Duration:** 24 min
- **Started:** 2026-08-28T12:09:40Z
- **Completed:** 2026-08-28T12:33:00Z
- **Tasks:** 2
- **Files modified:** 12 (4 created, 8 modified)

## Which lanes are covered, plainly

This is the question 196-07 left open, and the answer must be stated rather than implied.

| Lane | What it is | Records cost? |
|---|---|---|
| Platform-driven build | `aether build <n> --plan-only` → the chat wrapper spawns the workers → `aether build-finalize` | **Yes** — since 196-05 |
| Platform-driven check | the same shape, ending in `aether continue-finalize` | **Yes** — since 196-07 |
| Autopilot / host-driven | `aether run` → the TS host dispatches → the same two finalizers | **Yes** — it ends in the finalizers above |
| Direct in-process build | `aether build <n>`, which runs its workers itself | **Yes — wired by this plan.** It filed nothing before. |
| Direct in-process check | `aether continue`, which runs its reviewers and watcher itself | **Yes — wired by this plan.** It filed nothing before. |

Nothing in the build-and-check cycle is inert any more. What is still not recorded anywhere is the cost of the OTHER worker-spawning workflows — planning, colonizing, sealing, and the automatic fix attempt. That is out of this phase's scope, which is explicitly building a phase and checking it, and it is written up with the work needed in `deferred-items.md` rather than left to be rediscovered.

One honest limitation, also recorded there: on the direct lanes the provider's own event is the only possible source of a figure, because those workers are subprocesses of the runtime and have no chat-session record of their own. A worker whose provider says nothing shows a dash and stays out of the total. That is D-01 as amended working exactly as the owner ruled, not a missing wire.

## The link-cutting proof

The chain has four links. Each was cut in turn in a backed-up copy of the file, the test re-run, and the file restored. These are the real failures, quoted from the runs — not descriptions of what should happen.

**Link 1 — the writer call (platform lane), `cmd/codex_build_finalize.go`:**

```
--- FAIL: TestSpendPipelineIsReachableEndToEnd/a_finished_build_files_rows_and_the_detail_view_shows_them
    read ledger .../.aether/data/spend/phase-1-build.json: no such file or directory
--- FAIL: TestSpendPipelineIsReachableEndToEnd/the_check_files_its_own_rows_beside_the_build's
    read ledger .../.aether/data/spend/phase-1-build.json: no such file or directory
```

**Link 1 — the writer call (direct lane), `cmd/codex_build.go`:**

```
--- FAIL: TestSpendPipelineIsReachableEndToEnd/the_direct_lane_records_what_its_own_workers_reported
    `aether build 1` ran its workers in-process and filed no token record at all;
    the cost line on this lane can only ever say nothing was recorded
--- FAIL: TestSpendPipelineIsReachableEndToEnd/the_direct_check_records_what_its_own_reviewers_reported
    read ledger .../spend/phase-1-build.json: no such file or directory
```

This is also the exact failure the test produced BEFORE the wiring existed — the red step, quoted from the first run of the day.

**Link 2 — the resolver call, `cmd/spend_writer.go`:**

```
--- FAIL: TestSpendPipelineIsReachableEndToEnd/the_direct_lane_records_what_its_own_workers_reported
    worker Chip-87's tool reported a figure, but its row records none:
    {AgentName:Chip-87 ... Usage:{InputTokens:0 ... TotalTokens:0 ... Source:}}
    the rows on disk sum to 0, want 857000 — the recorded figures are not the provider's
--- FAIL: TestSpendPipelineIsReachableEndToEnd/the_direct_check_records_what_its_own_reviewers_reported
    every checker's tool reported a figure, but no row on disk records one: [{AgentName:Watch-36 ...}]
```

Worth noting exactly which subtests this cut catches: the two direct-lane ones, because they are the ones whose workers actually report a figure. The platform-lane subtest survives it, honestly — its workers are answered by a completion packet that carries no usage, so its rows are unreported either way. The direct lane is what makes the resolver cut visible at all, which is a second reason wiring it mattered.

**Link 3 — the renderer call, `cmd/codex_workflow_cmds.go`:**

```
--- FAIL: TestSpendPipelineIsReachableEndToEnd/the_direct_lane_records_what_its_own_workers_reported
    `aether build 1` ended with 0 cost block(s), want exactly 1:
      Step 1/5: Prepare (0s)
    ━━ 🔨 B U I L D   D I S P A T C H   1 ━━
    ...
```

**Link 4 — the command registration, `cmd/spend_cmd.go`:**

```
--- FAIL: TestSpendPipelineIsReachableEndToEnd/a_finished_build_files_rows_and_the_detail_view_shows_them
    aether spend returned an error: unknown command "spend" for "aether"
--- FAIL: TestSpendPipelineIsReachableEndToEnd/the_direct_lane_records_what_its_own_workers_reported
    aether spend returned an error: unknown command "spend" for "aether"
```

After each cut the file was restored from its backup and the tree re-verified: `go build ./...`, `go vet ./cmd/ ./pkg/codex/`, `gofmt -l cmd/ pkg/` all clean, and the proof green.

## Accomplishments

- **The proof drives the real commands.** It runs `build-finalize`, `aether build 1`, `aether continue --heavy` and `aether spend` through the command tree, then reads the ledger files off the disk, the detail view's own output, and the ending screen. It never calls the writer, the resolver or the renderer. The provider figure it checks is produced by the production parser (`codex.ParseUsage`, through `codex.AttachWorkerUsage`) reading a real provider event; the expected total, 857,000, is written as a literal summed by hand from the four disjoint columns, so it cannot inherit an arithmetic bug from the function that produced the printed one.
- **The direct lane files rows.** Both halves of it. The build's writer call sits after the build is durably committed; the check's sits at all three points that lane can end — verification blocked, review blocked, and advanced — because a blocked check still spent what it spent.
- **A designed input finally has a producer.** `wrapperUsageRequest.Attached` — the field for "what the provider itself reported at the dispatch boundary" — was populated by no production file at all. Every provider measurement taken at that boundary was discarded before it could reach the ledger. `TestNothingThisPhaseAddedIsUncalled` fails by name if that ever becomes true again.
- **An anti-orphan check the command ratchet cannot provide.** The existing ratchet covers registered commands. It cannot see an internal entry point that exists, has tests, and is invoked by nothing — the exact shape the salvage assessment refused to merge. The new check names nine entry points and the one struct field, and says what would silently stop working for each.
- **The three branches are gone.** Each one's tip commit, verdict and reason were written down first, so the work stays reachable by commit forever even though the names are retired.
- **Two criteria corrected before a verifier could be misled by them.** The roadmap's criterion 1 and COST-01 both still promised a "measured versus estimated" label, which the owner's amended ruling abolished. Read literally, either one would have marked the shipped, correct behaviour as a failure.

## Task Commits

1. **Task 1 (RED): the failing reachability proof** — `e7291a8b` (test)
2. **Task 1 (GREEN): file token rows on the direct lane too** — `07c71333` (feat)
3. **Task 2: corrected wording, disposition record, branch deletions** — `0bfef8c0` (docs)

## Files Created/Modified

- `cmd/spend_pipeline_e2e_test.go` — the end-to-end proof across all four lanes, the provider-reporting dispatcher, and the anti-orphan check
- `cmd/branch_disposition_test.go` — the record test and the branches-are-gone test
- `cmd/codex_build.go` — the usage field on a dispatch (unserialized), carried from the dispatch boundary, and the direct build's writer call
- `cmd/codex_continue.go` — the same field on the check's flow step, populated for reviewers and for the watcher, and the writer call at all three direct-check exits
- `cmd/spend_writer.go` — `Attached` populated from the dispatches, `spendWorkerFlowSteps` (bookkeeping steps are not workers), and `fileDirectContinueSpendRows`
- `cmd/spend_cost_line.go` — the one-worker sentences, which previously read "any of the 1 worker"
- `cmd/testdata/golden_build.txt`, `cmd/testdata/golden_continue.txt` — refreshed; both direct lanes now list their worker instead of saying nothing was recorded
- `.planning/phases/196-see-what-it-cost/196-BRANCH-DISPOSITION.md` — the record
- `.planning/phases/196-see-what-it-cost/deferred-items.md` — the four workflows still recording nothing, with the work needed
- `.planning/ROADMAP.md`, `.planning/REQUIREMENTS.md` — criterion 1 and COST-01 corrected, with the amendment cited

## Decisions Made

See `key-decisions` in the frontmatter. The one most likely to be questioned later:

**Wiring the direct lane was in scope, and deferring it would not have been honest.** The plan's own objective is to ask whether the parts are joined up. Finding that a whole lane files nothing and then writing a passing reachability proof over the lane that does work would have been the precise failure CLAUDE.md's Definition of Done exists to prevent — and the irony is sharp: the inert lane is the only one where a provider actually reports a figure, so the system was reading a real measurement at the dispatch boundary and throwing it away, while the lane that recorded rows had to reconstruct figures from session artifacts. The wiring is 62 lines in one file, 24 in another, and every one of them is covered by the proof.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] The direct in-process lane filed no token rows**

- **Found during:** Task 1
- **Issue:** `aether build <n>` and `aether continue` run their workers inside the runtime and never called the writer. Their cost block could only ever read "No token use was recorded for this phase". The phase's own criterion 1 says every build and continue ends with the run total.
- **Fix:** Carried the provider's own figure from the dispatch boundary onto the build dispatch and the check's flow step (both `json:"-"`), and called the shared writer at the end of the direct build and at all three direct-check exits.
- **Files modified:** `cmd/codex_build.go`, `cmd/codex_continue.go`, `cmd/spend_writer.go`
- **Verification:** the two new direct-lane subtests, plus the writer-call cut which fails them by name.
- **Committed in:** `07c71333`

**2. [Rule 2 - Missing Critical] `wrapperUsageRequest.Attached` had no producer anywhere**

- **Found during:** Task 1
- **Issue:** The resolver's field for the provider's own measurement was written only by tests. In production it was always empty, so the directly-spawned measurement path was dead code with a passing test suite — the shape this phase exists to stop shipping.
- **Fix:** `writeSpendRowsForRun` now builds it from the dispatches it was handed. On the platform lane the field is the zero value (dispatch usage is never serialized), so nothing there changed and no outside caller gained the ability to assert what a run cost.
- **Files modified:** `cmd/spend_writer.go`
- **Verification:** `TestNothingThisPhaseAddedIsUncalled` fails by name without it; the resolver cut proves the path is live.
- **Committed in:** `07c71333`

**3. [Rule 2 - Missing Critical] The cost line read "any of the 1 worker"**

- **Found during:** Task 1 (visible in the refreshed goldens once the direct lanes started filing rows)
- **Issue:** Two sentences in the cost block were written for the plural case only, and a one-worker build is now the ordinary case after Phase 194 shrank the team. "No tool reported a figure for any of the 1 worker" is not English, on the one surface whose whole purpose is to be read by someone non-technical.
- **Fix:** Separate one-worker sentences. The plural wording is untouched, so the existing assertions on it still hold.
- **Files modified:** `cmd/spend_cost_line.go`
- **Verification:** both refreshed goldens read cleanly; the existing cost-line tests pass unchanged.
- **Committed in:** `07c71333`

### Departures from the plan text

**4. The plan names a test that does not exist.** Its acceptance criterion cites `TestNoOrphanRuntimeCommands`. There is no such test; the orphan ratchet's real names are `TestNoRegisteredSubcommandIsUnreferenced` and `TestOrphanAllowlistOnlyShrinks`. Both were run and both pass, with no new allowlist entry. Recorded rather than quietly substituted.

**5. The disposition needed two tests, not one.** The plan asks for `TestBranchDispositionRecordsAllThreeBranches`, which exists and passes. But a test that reads a document cannot fail while the branches it describes are still present — it passed happily before the deletions. `TestDisposedBranchesAreGoneFromTheRepository` is the one that actually fails when the requirement is unmet, and it did fail before the deletions:

```
--- FAIL: TestDisposedBranchesAreGoneFromTheRepository
    these branches were recorded as deleted but still exist:
    worktree-agent-a47f78913caf6fde0
      worktree-agent-a59fd3ee68644ea21
      worktree-agent-aa57076cd698da76f
```

It also asserts each recorded commit still resolves, so the record cannot claim recoverability it does not have.

---

**Total deviations:** 3 auto-fixed (all missing-critical), 2 departures from the plan text recorded.
**Impact on plan:** the three fixes are what make the plan's own first success criterion — "the cost path is proven reachable end to end" — true rather than true-on-one-lane. No scope creep: nothing was added that the proof does not cover.

## Issues Encountered

None unresolved. The four workflows that still record nothing (planning, colonizing, sealing, the automatic fix attempt) are out of this phase's scope and are written up in `deferred-items.md` with the decision they need and the code shape to copy.

## Verification

```
go test ./cmd -run 'Test(SpendPipelineIsReachableEndToEnd|NothingThisPhaseAddedIsUncalled|
                        BranchDisposition|DisposedBranches|Orphan|
                        NoRegisteredSubcommandIsUnreferenced|NoTokenCountIsDerivedFromLength)' -count=1   ok
go test ./cmd -run 'Test(DocumentedSubcommandsAreSeverityClassified|PlatformParityGolden|
                        RegressionSnapshot|HumanFacingOutputGoesThroughWriteVisualOutput|Golden|
                        LifecycleWrapper|Spend|CostLine|EndsWithOneCostLine|
                        NoLaneRendersTwoCostLines|Continue)' -count=1                                     ok
go test ./cmd -run 'Test(Build|Dispatch|Coherent|Job|Watcher|Review|Ceremony)' -count=1                    ok
go test ./cmd -run 'Test(SpendLedger|OpenCode)' -count=1  (run BEFORE any branch was deleted)              ok
go build ./...                                                                                             ok
go vet ./cmd/ ./pkg/codex/                                                                                 ok
gofmt -l cmd/ pkg/                                                                                         no output
git branch --list 'worktree-agent-a47f*' 'worktree-agent-aa57*' 'worktree-agent-a59f*'                     prints nothing
```

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- Phase 196 is complete. Every build and every check, on every lane, ends with one honest cost block; `aether spend` shows the detail; nothing this phase added sits uncalled; and the three branches are gone with their reasons and commits recorded.
- Phase 199's end-to-end proof can rely on the cost path being live on the lane a person actually types, not only on the wrapper lane.
- One open item is handed forward deliberately, in `deferred-items.md`: planning, colonizing, sealing and the automatic fix attempt spawn workers and record nothing. Closing it needs a scope decision about how a non-phase run is keyed, not more wiring.

## Self-Check: PASSED

Files claimed as created exist on disk:

```
FOUND: cmd/spend_pipeline_e2e_test.go
FOUND: cmd/branch_disposition_test.go
FOUND: .planning/phases/196-see-what-it-cost/196-BRANCH-DISPOSITION.md
FOUND: .planning/phases/196-see-what-it-cost/deferred-items.md
```

Commits claimed exist in git:

```
FOUND: e7291a8b  test(196-08): add the failing end-to-end reachability proof
FOUND: 07c71333  feat(196-08): file token rows on the direct lane too
FOUND: 0bfef8c0  docs(196-08): correct the superseded cost wording and dispose of three branches
```

Branches claimed deleted are gone:

```
GONE: worktree-agent-a47f78913caf6fde0 (was be1e160b)
GONE: worktree-agent-aa57076cd698da76f (was ecd98b5c)
GONE: worktree-agent-a59fd3ee68644ea21 (was 1cbf3615)
```

---
*Phase: 196-see-what-it-cost*
*Completed: 2026-08-28*
