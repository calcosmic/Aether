---
phase: 196-see-what-it-cost
plan: 02
subsystem: infra
tags: [token-usage, spend, anthropic-sdk, agent-pool, ratchet, ast-guard, go, tdd]

requires:
  - phase: 174-spend
    provides: codex.WorkerUsage, ParseUsage, BilledTotalTokens, the source-tag vocabulary
  - phase: 196-01
    provides: the salvaged spend ledger whose measured/estimated split reads the source tag
provides:
  - llm.Usage carrying all four disjoint billed columns on both the single-response and streamed paths
  - agent.WorkerUsageFromStreamUsage — the one conversion into the authoritative worker usage type
  - A cross-path equality invariant proving both accounting lanes total the same hand-written literal
  - TestNoTokenCountIsDerivedFromLength — a package-scoped ratchet, registered in the repo's guard inventory
  - Deletion of EstimateUsage and estimateCharsPerToken; an unreported worker now carries no figure at all
affects: [196-03, 196-04, 196-05, 196-06, 196-07, 196-08]

actuals:
  tokens: 16795
  tasks: 3
  commits: 6

tech-stack:
  added: []
  patterns:
    - "Package-scoped AST ratchets: scan by package directory, not filename pattern, so a file a later plan adds is covered the day it appears"
    - "One stated exemption rule (a _test.go file is a test) asserted directly, instead of a list of individually pardoned files"
    - "Cross-path equality proved against a third hand-written value, never by comparing the code with itself"
    - "Identifier matching by camelCase word-splitting rather than substring, so `silent`/`golden`/`resize` cannot trip a length-word scan"

key-files:
  created:
    - pkg/llm/usage_test.go
    - pkg/agent/pool_usage_test.go
    - cmd/spend_accounting_parity_test.go
    - cmd/spend_no_length_derivation_test.go
  modified:
    - pkg/llm/client.go
    - pkg/llm/streaming.go
    - pkg/agent/pool.go
    - pkg/codex/usage.go
    - pkg/codex/usage_test.go
    - pkg/codex/platform_dispatch.go
    - pkg/codex/worker.go
    - cmd/ci_wiring_gate_test.go
    - cmd/subcommand_reachability_ratchet_test.go
    - .github/workflows/ci.yml

key-decisions:
  - "A pool-converted usage row is tagged UsageSourceSessionTranscript. It cannot be provider-grade (only ParseUsage may set that) and must not be an estimate (it is a real measurement, and the estimate tag routes it into the ledger's estimated subtotal). Session-transcript is the one honest tag in the unchanged vocabulary."
  - "llm.Usage gains the columns but gains no method at all — asserted by reflection. A second total-computing implementation in a second package is exactly how the two lanes drifted apart."
  - "The ratchet scopes by package directory, not filename, because a filename-scoped scan would have excluded pkg/codex/usage.go — the one file the violation actually lived in."
  - "AttachWorkerUsage keeps its config parameter even though nothing reads it now: it is the dispatch boundary's stable shape and the transcript sources in 196-03/04 need the worker identity. Documented in-file as NOT a hook for reviving a length-derived figure."
  - "The guard-file inventory's anti-vacuity floor was raised from 5 to its live count of 11, making the inventory itself a ratchet rather than a floor that stopped tracking reality six guard files ago."

patterns-established:
  - "Registered guards: a new ratchet is appended to wiringGateGuardFiles and to the named CI step's -run filter in the same commit, so the escape-hatch and CI-coverage tests hold it to the same standard as every other guard"
  - "Pre-deletion ratchet evidence: write the rule, run it against the unfixed tree, record which real files it names, then delete — proof the rule had a violation rather than being a rule about nothing"

requirements-completed: [COST-01, COST-02, COST-03]

coverage:
  - id: D1
    description: "The chat-model client reports every column it is billed for, on both the single-response and the streamed path"
    requirement: "COST-01"
    verification:
      - kind: unit
        ref: "pkg/llm/usage_test.go#TestUsageCarriesEveryBilledColumn"
        status: pass
      - kind: unit
        ref: "pkg/llm/usage_test.go#TestStreamedUsageCarriesEveryBilledColumn"
        status: pass
      - kind: unit
        ref: "pkg/llm/usage_test.go#TestUsageWithoutCacheActivityIsUnchanged"
        status: pass
    human_judgment: false
  - id: D2
    description: "The agent pool reports the authoritative usage type — all four columns in the completion event, the whole value in the callback — and never claims provider grade"
    requirement: "COST-02"
    verification:
      - kind: unit
        ref: "pkg/agent/pool_usage_test.go#TestPoolCompletionEventCarriesEveryColumn"
        status: pass
      - kind: unit
        ref: "pkg/agent/pool_usage_test.go#TestPoolUsageCallbackHandsOverAuthoritativeUsage"
        status: pass
      - kind: unit
        ref: "pkg/agent/pool_usage_test.go#TestPoolUsageIsNeverProviderGrade"
        status: pass
    human_judgment: false
  - id: D3
    description: "Both accounting lanes produce the same total for the same four raw columns, and the invariant breaks if either lane drops one (D-05)"
    requirement: "COST-02"
    verification:
      - kind: unit
        ref: "cmd/spend_accounting_parity_test.go#TestBothAccountingPathsAgreeOnTheTotal"
        status: pass
    human_judgment: false
  - id: D4
    description: "Nothing in the four scanned packages derives a token count from a character or prompt length, and a registered guard fails if anything ever does again (D-01 as amended)"
    requirement: "COST-03"
    verification:
      - kind: unit
        ref: "cmd/spend_no_length_derivation_test.go#TestNoTokenCountIsDerivedFromLength"
        status: pass
      - kind: unit
        ref: "cmd/ci_wiring_gate_test.go#TestWiringGateStepRunsEveryWiringTest"
        status: pass
      - kind: unit
        ref: "cmd/subcommand_reachability_ratchet_test.go#TestWiringGuardsHaveNoRuntimeEscapeHatch"
        status: pass
    human_judgment: false
  - id: D5
    description: "A worker whose tool reported nothing carries an empty usage value; the character-derived helper and its ratio constant no longer exist"
    requirement: "COST-03"
    verification:
      - kind: unit
        ref: "pkg/codex/usage_test.go#TestUnreportedUsageIsEmptyNotInvented"
        status: pass
      - kind: unit
        ref: "pkg/codex/usage_test.go#TestEveryDispatchLeavesAUsageRow"
        status: pass
    human_judgment: false

duration: 13min
completed: 2026-08-28
status: complete
---

# Phase 196 Plan 02: One Authoritative Token Count Summary

**The cache-blind second accounting path is gone — `llm.Usage` carries all four disjoint billed columns on both response paths, the agent pool converts once into `codex.WorkerUsage`, both lanes are locked to the same hand-written total, and the only producer of a length-derived token count in the tree is deleted at source under a package-scoped ratchet registered in the repo's own guard inventory.**

## Performance

- **Duration:** 13 min
- **Started:** 2026-08-28T09:54:46Z
- **Completed:** 2026-08-28T10:08:33Z
- **Tasks:** 3 (each RED then GREEN)
- **Files modified:** 14 (4 created, 10 modified)

## Accomplishments

- `llm.Usage` gained `CacheReadInputTokens` and `CacheCreationInputTokens`, filled identically on the single-response path (`convertSDKMessage`) and the streamed path (`AccumulateStream`). It deliberately declares **no method at all** — asserted by reflection — because a second total-computing implementation in a second package is how the two lanes drifted apart.
- `agent.WorkerUsageFromStreamUsage` is the single conversion into the authoritative type. The pool's completion event now carries five keys (four columns plus the total from `BilledTotalTokens()`), and the usage callback hands over the whole value instead of two loose numbers.
- `TestBothAccountingPathsAgreeOnTheTotal` carries one set of raw columns through both lanes and compares each independently produced total against `102550`, written out by hand. Its own non-vacuity is proved: a column-dropped variant on either lane must disagree.
- `EstimateUsage` and `estimateCharsPerToken` are deleted. The dispatch boundary attaches nothing when the provider reported nothing, so an unmeasured worker carries the zero value that downstream reads as "not reported".
- `TestNoTokenCountIsDerivedFromLength` scans `pkg/codex`, `pkg/llm`, `pkg/agent` and all of `cmd` by **package directory**, exempting only `_test.go` by one stated rule, and is registered in `wiringGateGuardFiles` with the inventory floor raised from 5 to its live count of 11.

## Task Commits

1. **Task 1 RED** — `4801ab46` (test) — failing tests for every billed column on both paths
2. **Task 1 GREEN** — `5de8e3d2` (feat) — cache columns added and filled on both paths
3. **Task 2 RED** — `a924f48b` (test) — parity and pool usage tests, both packages fail to compile
4. **Task 2 GREEN** — `d5912e1e` (feat) — one conversion in the pool; widened callback and event payload
5. **Task 3 RED** — `8aae54a0` (test) — the ratchet, failing on the live producer
6. **Task 3 GREEN** — `ea692557` (feat) — deletion at source, guard registered, CI filter widened

## The pre-deletion ratchet run (the load-bearing evidence)

Written first and run against the tree **before** any deletion. Command:

```
go test ./cmd -run 'TestNoTokenCountIsDerivedFromLength' -count=1 -timeout 300s
```

Real output:

```
--- FAIL: TestNoTokenCountIsDerivedFromLength (0.15s)
    spend_no_length_derivation_test.go:115: 6 site(s) derive a token count from a character or prompt length — D-01 as amended forbids this ANYWHERE: a worker whose tool reported nothing shows no number at all, never a guess wearing a label:
          pkg/codex/platform_dispatch.go:243 EstimateUsage — the character-derived usage fallback deleted by Phase 196 plan 02
          pkg/codex/usage.go:83 estimateCharsPerToken — the characters-per-token ratio constant deleted by Phase 196 plan 02
          pkg/codex/usage.go:88 EstimateUsage — the character-derived usage fallback deleted by Phase 196 plan 02
          pkg/codex/usage.go:92 estimateCharsPerToken — names a characters-to-tokens ratio used in an arithmetic expression
          pkg/codex/usage.go:92 estimateCharsPerToken — the characters-per-token ratio constant deleted by Phase 196 plan 02
          pkg/codex/usage.go:92 tokens — is a token count computed by scaling a character or prompt length
FAIL
FAIL	github.com/calcosmic/Aether/cmd	0.958s
```

**It names `pkg/codex/usage.go`.** That is the specific thing the plan asked to be recorded: the scan is wide enough to see the file the violation actually lives in. An earlier draft of this plan scoped the ratchet to spend/wrapper-usage filenames, which would have excluded that very file and produced a green rule about nothing. It also names all three shape rules firing independently (the forbidden identifier, the ratio operand, the token-named target), and it produced **zero false positives** across the ~450 production files in the four scanned packages.

After the deletion the same command returns `ok`. To confirm the green is not the ratchet going blind, a real on-disk violation was planted in `pkg/llm` and the scan re-run:

```
--- FAIL: TestNoTokenCountIsDerivedFromLength (0.16s)
    spend_no_length_derivation_test.go:115: 1 site(s) derive a token count from a character or prompt length ...
          pkg/llm/zz_planted_check.go:5 tokenEstimate — is a token count computed by scaling a character or prompt length
```

The planted file was then deleted and the scan returned to `ok`. The same property is permanently executable as the subtest `a planted violation is caught in every scanned package`, which parses three planted shapes at a path inside each of the four package directories on every run.

## Other RED evidence (real output)

**Task 1 RED** — `go test ./pkg/llm -run 'Test(UsageCarriesEveryBilledColumn|StreamedUsageCarriesEveryBilledColumn|UsageWithoutCacheActivityIsUnchanged)' -count=1`:

```
# github.com/calcosmic/Aether/pkg/llm [github.com/calcosmic/Aether/pkg/llm.test]
pkg/llm/usage_test.go:59:15: got.Usage.CacheReadInputTokens undefined (type Usage has no field or method CacheReadInputTokens)
pkg/llm/usage_test.go:63:15: got.Usage.CacheCreationInputTokens undefined (type Usage has no field or method CacheCreationInputTokens)
pkg/llm/usage_test.go:123:4: unknown field CacheReadInputTokens in struct literal of type Usage
FAIL	github.com/calcosmic/Aether/pkg/llm [build failed]
```

**Task 2 RED** — both packages:

```
# github.com/calcosmic/Aether/pkg/agent [github.com/calcosmic/Aether/pkg/agent.test]
pkg/agent/pool_usage_test.go:105:17: cannot use func(usage codex.WorkerUsage) {…} (value of type func(usage codex.WorkerUsage)) as TokenUsageCallback value in struct literal
FAIL	github.com/calcosmic/Aether/pkg/agent [build failed]

# github.com/calcosmic/Aether/cmd [github.com/calcosmic/Aether/cmd.test]
cmd/spend_accounting_parity_test.go:58:20: undefined: agent.WorkerUsageFromStreamUsage
FAIL	github.com/calcosmic/Aether/cmd [build failed]
```

**The registration hazard fired exactly as the plan predicted.** After appending the guard to `wiringGateGuardFiles`:

```
--- FAIL: TestWiringGateStepRunsEveryWiringTest (0.01s)
    ci_wiring_gate_test.go:203: CI step "Verify subcommand wiring and CLI flag contracts"'s -run filter does not match 1 guard test(s) — they would silently stop running under the named step even though `go test ./...` still finds them, which is exactly the illegible-failure mode D-15 exists to prevent:
          TestNoTokenCountIsDerivedFromLength
```

The workflow edit **landed directly**; the permission layer did not refuse it this time, so the worktree fallback path was not needed. The registration was never at risk of being dropped.

## Acceptance criteria — commands run and results

**Task 1**

| Criterion | Command | Result |
|---|---|---|
| Three column tests pass | `go test ./pkg/llm -run 'Test(UsageCarriesEveryBilledColumn\|StreamedUsageCarriesEveryBilledColumn\|UsageWithoutCacheActivityIsUnchanged)' -count=1` | PASS |
| Every expected number is a literal | constants `docExample*` in `pkg/llm/usage_test.go`, taken from the provider's documented disjoint-column example; no call into code under test | PASS (by inspection of the test source) |
| Usage declares no total-computing method | reflection subtest `the usage type exposes no total-computing method` — fails on ANY method, not just a total | PASS |
| `go vet ./pkg/llm` clean | `go vet ./pkg/llm` | PASS (exit 0) |

**Task 2**

| Criterion | Command | Result |
|---|---|---|
| Three pool tests pass | `go test ./pkg/agent -run 'Test(PoolCompletionEventCarriesEveryColumn\|PoolUsageCallbackHandsOver\|PoolUsageIsNeverProviderGrade)' -count=1` | PASS (`ok ... 0.644s`) |
| Parity test passes and fails on a dropped column | `go test ./cmd -run 'TestBothAccountingPathsAgreeOnTheTotal' -count=1` | PASS; the `dropping a column on either lane breaks the agreement` subtest asserts divergence in both directions |
| Expected total is one hand-written literal | `wantBilledTotal int64 = 102550`, declared once with the four addends written out in the doc comment | PASS |
| No production file outside the codex usage type adds columns | subtest `nothing outside the codex usage type adds token columns itself` (AST scan of all four packages) | PASS — and demonstrated non-vacuous: disabling the `usage.go` exemption made it name 5 sites at `pkg/codex/usage.go:270` and `:279` |

**Task 3**

| Criterion | Command | Result |
|---|---|---|
| Ratchet passes; fails on a planted violation in each package | `go test ./cmd -run 'TestNoTokenCountIsDerivedFromLength' -count=1 -v` | PASS, with subtests `a planted violation is caught in every scanned package` and `the only exemption is that a _test.go file is a test` both PASS |
| Pre-deletion run names the codex file | recorded above | PASS — names `pkg/codex/usage.go` (3 sites) and `pkg/codex/platform_dispatch.go` |
| Replacement codex test passes | `go test ./pkg/codex -run 'TestUnreportedUsageIsEmptyNotInvented' -count=1` | PASS |
| Helper and ratio constant gone; dispatch attaches nothing | `grep -rn 'EstimateUsage\|estimateCharsPerToken' --include='*.go' .` returns only the ratchet's own forbidden-identifier map | PASS |
| Guard registered, floor raised, both guard tests pass | `go test ./cmd -run 'Test(WiringGuardsHaveNoRuntimeEscapeHatch\|WiringGateStepRunsEveryWiringTest)' -count=1` | PASS (`ok ... 5.234s`) |
| Full codex package passes; tag constants unchanged | `go test ./pkg/codex -count=1`; `grep` of the three `UsageSource*` constants | PASS — constants byte-identical |

**Plan-level verification**

| Command | Result |
|---|---|
| `go test ./pkg/llm ./pkg/agent ./pkg/codex -count=1` | `ok` × 3 (0.747s / 6.966s / 31.515s) |
| `go test ./cmd -run 'Test(BothAccountingPathsAgree\|NoTokenCountIsDerivedFromLength\|InternalWorkerResultCarriesProviderUsage\|WiringGuardsHaveNoRuntimeEscapeHatch\|WiringGateStepRunsEveryWiringTest)' -count=1` | `ok  github.com/calcosmic/Aether/cmd  0.913s` |
| `go build ./cmd/aether` | exit 0 |
| `go build ./...` | exit 0 |
| `go vet ./pkg/... ./cmd` | exit 0 |
| `gofmt -l cmd/ pkg/` | no output |
| `go test ./cmd -run 'Test(SpendLedger\|SpendRow\|Ledger)' -count=1` | `ok` — 196-01's salvaged ledger tests still green |

## Files Created/Modified

- `pkg/llm/client.go` — `Usage` gains the two cache columns and a doc comment forbidding it any method; `convertSDKMessage` fills all four
- `pkg/llm/streaming.go` — `AccumulateStream` fills the cache columns on `message_start` and takes a restated column from a cumulative `message_delta` only when the delta actually stated one
- `pkg/llm/usage_test.go` — three tests over the provider's documented disjoint-column example, plus the reflection assertion that `Usage` has no methods
- `pkg/agent/pool.go` — `WorkerUsageFromStreamUsage` (the one conversion); `TokenUsageCallback` widened to hand over `codex.WorkerUsage`; completion payload carries four columns plus the total and the source tag
- `pkg/agent/pool_usage_test.go` — completion-event, callback and source-tag tests, including "a stream that reported nothing carries no figure at all"
- `pkg/codex/usage.go` — `EstimateUsage` and `estimateCharsPerToken` deleted, replaced by a comment recording what was removed and why
- `pkg/codex/platform_dispatch.go` — the dispatch boundary attaches nothing when the provider reported nothing; the retained `config` parameter is documented as not a hook for reviving the guess
- `pkg/codex/worker.go` — `assembledPromptChars` deleted (its only caller was the deleted fallback)
- `pkg/codex/usage_test.go` — `TestEstimateUsageIsNeverMistakenForAMeasurement` replaced by `TestUnreportedUsageIsEmptyNotInvented`; the dispatch-boundary table now expects an empty row for an unreported worker
- `cmd/spend_accounting_parity_test.go` — the cross-lane equality invariant and the "nothing else adds columns" AST guard
- `cmd/spend_no_length_derivation_test.go` — the D-01 ratchet
- `cmd/ci_wiring_gate_test.go` — the ratchet appended to `wiringGateGuardFiles`
- `cmd/subcommand_reachability_ratchet_test.go` — inventory floor raised 5 → 11
- `.github/workflows/ci.yml` — the named wiring-gate step's `-run` filter now names `TestNoTokenCountIsDerivedFromLength`

## Decisions Made

- **A pool-converted row is tagged `UsageSourceSessionTranscript`.** The plan said "a pool-converted row takes the tag its origin honestly supports" and forbade changing the vocabulary. `provider` is ruled out by the type's own documented boundary (only `ParseUsage` may set it). `estimate` is ruled out on a stronger ground than taste: the salvaged ledger's measured/estimated split keys on `Estimated()`, so tagging a real measurement as an estimate would file live token counts under the guessed subtotal. `session-transcript` already means exactly "a genuine measurement the Go runtime read itself, not provider-grade", which is what this is.
- **The conversion sets no total.** `TotalTokens` is left zero so `BilledTotalTokens()` falls through to the authoritative sum. This is what makes the parity invariant meaningful rather than circular.
- **An all-zero stream converts to the zero value, not a tagged empty row.** A tagged row with no numbers is a row claiming to be a measurement of zero. D-01 as amended wants absence, and `Empty()` already reports absence when the source tag is blank.
- **The ratchet scopes by package directory.** Stated in the plan and confirmed by the pre-deletion run: a filename-scoped scan would have missed `pkg/codex/usage.go` entirely.
- **Identifier matching splits camelCase into words.** Substring matching on `len`/`size` would have fired on `silent`, `golden` and `resize` across ~450 files. Word-splitting produced zero false positives on the first run.
- **The inventory floor was raised to the live count (11), not nudged.** The old floor of 5 had stopped tracking reality six guard files ago, so two-thirds of the inventory could have been deleted silently.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Deleted `assembledPromptChars`, which the plan did not name**

- **Found during:** Task 3
- **Issue:** `WorkerConfig.assembledPromptChars` in `pkg/codex/worker.go` existed solely to feed the deleted fallback. Its doc comment read "It exists only to give an unmeasured run a non-zero, clearly-labelled estimate" — a documentation claim about runtime behaviour that stopped being true the moment the fallback went. CLAUDE.md makes that a named rule: such a claim must be testable or removed. Left in place it would also have been a ready-made helper for reviving the exact shape D-01 forbids.
- **Fix:** Deleted the method. `pkg/codex/worker.go` was not in the plan's `files_modified`; nothing else referenced it.
- **Verification:** `go build ./...` exit 0, `go test ./pkg/codex -count=1` ok.
- **Committed in:** `ea692557`

**2. [Rule 2 - Missing Critical] Made "nothing else adds token columns" executable**

- **Found during:** Task 2
- **Issue:** The plan's acceptance criterion "No production file outside the codex usage type computes a token total by adding columns" had no enforcing command. CLAUDE.md's Definition of Done treats an unenforced criterion as unsatisfied, and this repo's history is a list of criteria that were ticked without one.
- **Fix:** Added the subtest `nothing outside the codex usage type adds token columns itself` to the parity test — an AST walk over all four packages that names any `+` chain with two or more raw token-column operands, exempting only `pkg/codex/usage.go`.
- **Verification:** Passes; and disabling the exemption made it name the five real summation sites in `usage.go`, proving it is not vacuous.
- **Committed in:** `d5912e1e`

**3. [Rule 2 - Missing Critical] Kept `AttachWorkerUsage`'s now-unread `config` parameter, documented**

- **Found during:** Task 3
- **Issue:** After the deletion nothing read `config`. Silently leaving an unexplained unused parameter on an exported boundary invites a future reader to fill it back in with the thing that was just removed.
- **Fix:** Kept the parameter (it is the stable boundary shape the transcript sources in 196-03/04 will need for worker identity) and documented in-file that it is explicitly not a hook for reviving a length-derived figure, naming the ratchet that would catch it.
- **Verification:** `go vet ./pkg/codex` exit 0; the ratchet fails on any such revival, demonstrated with a planted file.
- **Committed in:** `ea692557`

---

**Total deviations:** 3 auto-fixed (3 missing-critical). **Impact on plan:** All three tighten requirements the plan already stated. One file outside `files_modified` was touched (`pkg/codex/worker.go`), to delete dead code left behind by the plan's own deletion. No architectural change, no new dependency, no scope creep.

## Issues Encountered

None that required rescue. The one predicted hazard — registering the guard breaking `TestWiringGateStepRunsEveryWiringTest` until the CI filter names it — fired on cue and was resolved by editing `.github/workflows/ci.yml` directly. That edit was permitted in this session, so the worktree fallback the plan authorised was not needed.

## Known Stubs

None. Every symbol added is exercised by a test in the same commit.

One name is now dormant rather than stubbed: `codex.UsageSourceEstimate` and `WorkerUsage.Estimated()` remain in the vocabulary with no production writer, because the plan explicitly forbids touching the source-tag constants (196-01's salvaged ledger tests are written against them, and 196-07 chooses the renderer's word for the tag). This is a stated interface constraint of the plan, not an omission.

## Threat Flags

None. No network endpoint, auth path, file-access pattern or schema change at a trust boundary. The one workflow-file edit widens a test filter and does not alter permissions, triggers or the release gate — `TestReleaseGateWorkflowShapeIsWhitelisted`, `TestBlanketGateCheckRejectsADecoyStep` and `TestGateProbeCatchesEveryKnownGateNeutering` were all re-run and pass.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **D-05 is closed, so the renderer in wave 3 is now safe to build.** There is one authoritative token type, both lanes agree, and the agreement is locked by a test that fails if either drifts.
- **D-01 as amended is enforced, not merely promised.** Any later plan in this phase that tries to reach for a character-count figure — in the ledger, the renderer, or `aether spend` — fails `TestNoTokenCountIsDerivedFromLength` in CI under its own named step. Files those plans add to the four scanned packages are covered the day they appear, with no guard update needed.
- **Names later plans depend on:** `agent.WorkerUsageFromStreamUsage`, `llm.Usage.{CacheReadInputTokens,CacheCreationInputTokens}`, and the unchanged `codex.WorkerUsage` API including `BilledTotalTokens()`, `TotalInputTokens()`, `Empty()`, `Measured()`, `Estimated()`.
- **One behavioural change downstream plans must expect:** `AttachWorkerUsage` no longer guarantees a non-empty usage row. A dispatch whose provider reported nothing now returns `WorkerUsage{}`. Any writer must render that as "not reported" (D-01) rather than as zero tokens, and must not fold it into the measured total.
- `TokenUsageCallback`'s signature changed. It had no caller anywhere in the repository, so nothing broke, but a future caller now receives the whole authoritative value.

## Self-Check: PASSED

- `pkg/llm/usage_test.go` — FOUND on disk
- `pkg/agent/pool_usage_test.go` — FOUND on disk
- `cmd/spend_accounting_parity_test.go` — FOUND on disk
- `cmd/spend_no_length_derivation_test.go` — FOUND on disk
- `4801ab46`, `5de8e3d2`, `a924f48b`, `d5912e1e`, `8aae54a0`, `ea692557` — all FOUND in `git log`
- All task acceptance criteria re-run after the final commit; all pass (tables above)
- Plan-level verification re-run after the final commit: three packages ok, cmd guard subset ok, `go build ./...` exit 0, `go vet ./pkg/... ./cmd` exit 0, `gofmt -l cmd/ pkg/` empty
- Working tree clean of unstaged changes to any file this plan touched

---
*Phase: 196-see-what-it-cost*
*Completed: 2026-08-28*
