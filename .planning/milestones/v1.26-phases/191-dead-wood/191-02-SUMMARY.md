---
phase: 191-dead-wood
plan: 02
subsystem: infra
tags: [go, colony-prime, dispatch-contract, dead-code, cwd-relative-loader, ratchet-test]

# Dependency graph
requires:
  - phase: 191-dead-wood (plan 01, parallel)
    provides: nothing directly consumed; both plans independently touch cmd/policy_schema_test.go (see Deviations)
provides:
  - colony/prompts/colony-prime.md deleted; loadColonyPrimeTemplates() permanently uses its Go-compiled defaults (unchanged -- nothing needed folding)
  - colony/policies/dispatch-contract.yaml deleted; loadDispatchContractPolicy() permanently uses its Go-compiled defaults (2 of 13 fields deliberately NOT folded -- see Deviations)
  - TestColonyPrimeAndDispatchContractDoNotReappear -- reappearance ratchet for both files, proven fail-then-pass
affects: [191-07 (final verification plan, depends on this plan's Wave 1 completion)]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Frozen testdata copy for before/after loader regression tests (cmd/testdata/191-02-*-original.{md,yaml}) -- lets a before/after test outlive the real file's deletion in the same task"
    - "Documented, narrowly-scoped fold exceptions: when a plan's literal fold-then-delete instruction would regress deliberate, tested behavior, keep the tested value, document why on the constant, and prove the exception explicitly in the regression test rather than asserting blanket equality"

key-files:
  created:
    - cmd/codex_dispatch_contract_test.go
    - cmd/colony_prime_and_dispatch_contract_ratchet_test.go
    - cmd/testdata/191-02-colony-prime-original.md
    - cmd/testdata/191-02-dispatch-contract-original.yaml
  modified:
    - cmd/colony_prime_context_test.go
    - cmd/codex_dispatch_contract.go
    - cmd/policy_schema_test.go

key-decisions:
  - "Did NOT fold colony/policies/dispatch-contract.yaml's stale planning fallback_behavior/fallback_visibility text into the Go defaults, despite the plan's literal fold-then-delete instruction, because three independent pre-existing sources (a unit test, a visual-output test, and a golden fixture) proved the CURRENT Go fallback is the deliberate, tested, correct value and the YAML was itself stale"
  - "Reused the existing cmd/dispatch_contract_test.go's resetDispatchPolicyState() instead of writing a second reset function -- 191-02-PLAN.md's read_first flagged this as something to confirm, not assume, and a dedicated test file for loadDispatchContractPolicy() (assumed absent by 191-CONTEXT.md) already existed under a different name"

requirements-completed: []

# Metrics
duration: 58min
completed: 2026-08-21
---

# Phase 191 Plan 02: Fold-and-Delete colony-prime.md and dispatch-contract.yaml Summary

**Deleted both CWD-relative silent-fallback loader targets (colony/prompts/colony-prime.md, colony/policies/dispatch-contract.yaml) after proving byte-identical output; caught and reverted a fold that would have silently regressed a tested fail-closed planning-fallback design.**

## Performance

- **Duration:** 58 min
- **Started:** 2026-08-21T00:53:00Z (approx, worktree creation)
- **Completed:** 2026-08-21T01:51:15Z
- **Tasks:** 3 (Task 1 baseline capture, Task 2 colony-prime.md, Task 3 dispatch-contract.yaml)
- **Files modified:** 7 (3 modified, 4 created; 2 files deleted)

## Accomplishments

- `colony/prompts/colony-prime.md` deleted. Every field the Go loader actually reads already matched its compiled default -- zero fold edits needed.
- `colony/policies/dispatch-contract.yaml` deleted. 11 of 13 fields matched; 2 fields were deliberately NOT folded because doing so broke tested, deliberate, shipped behavior (full evidence below).
- Both loaders proven byte-identical (or, for the 2 documented exceptions, proven to now consistently show the *correct* tested value) both by Go-level regression test and by real dev-checkout CLI capture.
- One reappearance ratchet (`TestColonyPrimeAndDispatchContractDoNotReappear`) covers both files, proven fail-then-pass for each independently.
- Full `go test ./cmd/... -count=1` passes (run three times across ~380-490s each, foreground, after every material change).

## Task Commits

1. **Task 1: Baseline-capture both loaders' real dev-checkout output** - no commit (read-only task; produced only `/tmp` scratch artifacts, no repo files modified)
2. **Task 2: Fold, delete, and prove colony-prime.md byte-identical** - `b893a8b7` (feat)
3. **Task 3: Fold, delete, and prove dispatch-contract.yaml byte-identical; add both reappearance ratchets** - `09d509bd` (feat)

## Files Created/Modified

- `colony/prompts/colony-prime.md` - deleted (0 divergent fields; nothing to fold)
- `colony/policies/dispatch-contract.yaml` - deleted (2 fields deliberately not folded; see Deviations)
- `cmd/colony_prime_context_test.go` - added `buildRichColonyPrimeFixture` helper and `TestColonyPrimeMdDeletionProducesByteIdenticalOutput`, a before/after regression test exercising all 16 colony-prime.md template sections
- `cmd/codex_dispatch_contract.go` - doc-comment-only change: explains why `fallbackPlanningFallbackBehavior`/`fallbackPlanningFallbackVisibility` deliberately do not match the deleted YAML's stale text (values themselves are byte-identical to the pre-plan committed state)
- `cmd/codex_dispatch_contract_test.go` - new file: `TestDispatchContractYamlDeletionProducesByteIdenticalOutput`, a before/after regression test with an explicit, evidenced exception for the 2 documented fields
- `cmd/colony_prime_and_dispatch_contract_ratchet_test.go` - new file: `TestColonyPrimeAndDispatchContractDoNotReappear`, Tier-1 existence-check ratchet for both deleted files
- `cmd/testdata/191-02-colony-prime-original.md` - frozen byte-copy of colony-prime.md's original content, captured before deletion
- `cmd/testdata/191-02-dispatch-contract-original.yaml` - frozen byte-copy of dispatch-contract.yaml's original content, captured before deletion
- `cmd/policy_schema_test.go` - removed 3 now-dangling schema-check rows for the deleted `dispatch-contract.yaml` (see Deviations)

## Task 1: Field-by-Field Diff Tables

### colony/prompts/colony-prime.md (16 sections, all consumed fields)

Every `writeSectionHeader`/`sectionString`/`fmtOrFallback` call site across `cmd/colony_prime_context.go` and `cmd/codex_dispatch_contract.go` (the file-overlap noted in 191-CONTEXT.md) was enumerated by full-file read and cross-checked with `grep -rn "getSectionTemplate\|sectionString(\|fmtOrFallback(\|writeSectionHeader("` across all of `cmd/*.go` (excluding tests) to confirm the list is exhaustive.

| Section | Fields checked | Verdict |
|---|---|---|
| state | header, goal_format, state_format, phase_format, phase_name_format, task_format, tasks_header, parallel_mode_format | MATCH (all 8) |
| review_depth | header, light_text, standard_text, heavy_text, default_text | MATCH (all 5) |
| pheromones | header, signal_format | MATCH (2) |
| pheromones | lifecycle_context | **N/A -- unread.** No call site references `sectionTemplate.LifecycleContext` anywhere; `colonyLifecycleSignalContext(state)` is a separate Go function, not template-driven |
| instincts | header, instinct_format | MATCH (2) |
| decisions | header, decision_format (verified byte-for-byte via `od -c`, including the em-dash) | MATCH (2) |
| learnings | header, phase_header_format, learning_format | MATCH (3) |
| worker_handoffs | header, worker_header_format, status_format, summary_format | MATCH (4) |
| worker_handoffs | list_format | **N/A -- unread.** `appendHandoffList` hardcodes `"- %s: %s\n"` directly, never calls `sectionString`/`fmtOrFallback` |
| hive_wisdom | header, entry_format | MATCH (2) |
| learned_memory | header, entry_format | MATCH (2) |
| global_queen_md | header, entry_format | MATCH (2) |
| user_preferences | header, entry_format | MATCH (2) |
| prior_reviews | header | MATCH (1) |
| prior_reviews | domain_format, domain_count_only_format | **N/A -- unread.** `buildPriorReviewsSection` builds domain lines with hardcoded `fmt.Sprintf("- %s (%d open)", ...)`, never calls `sectionString`/`fmtOrFallback` for these fields |
| local_queen_wisdom | header, entry_format | MATCH (2) |
| clarified_intent | header | MATCH (1) |
| blockers | header, blocker_format | MATCH (2) |
| medic_health | header, scan_timestamp_format, issue_format, issue_file_format | MATCH (4) |

**Result: every field the loader's code actually reads already matched its Go compiled default. Zero fold edits were needed to `cmd/colony_prime_context.go`.** The 4 unread fields (`lifecycle_context`, `list_format`, `domain_format`, `domain_count_only_format`) are pre-existing dead template surface, orthogonal to this plan's deletion (deleting the file changes nothing about them either way, since nothing ever consulted them through the template lookup).

### colony/policies/dispatch-contract.yaml (13 modeled fields + 3 unmodeled)

| Field | YAML value | Go fallback (pre-edit) | Verdict |
|---|---|---|---|
| execution_models.survey | "1 wave, parallel read-only worker execution" | `fallbackSurveyExecutionModel` | MATCH |
| execution_models.planning | "2 staged workers, scout then route-setter" | `fallbackPlanningExecutionModel` | MATCH |
| execution_models.planning_extended | "%d staged planning workers, scout plus route-setter with supporting castes" | inline literal, `planningDispatchContractForDispatches` | MATCH |
| deadline_policies.survey | "Each surveyor gets its own timeout..." | `fallbackSurveyDeadlinePolicy` | MATCH |
| deadline_policies.planning | "Each planning worker gets its own timeout..." | `fallbackPlanningDeadlinePolicy` | MATCH |
| dependency_behaviors.survey | "Surveyors are independent read-only workers..." | `fallbackSurveyDependencyBehavior` | MATCH |
| dependency_behaviors.planning | "Real worker dispatch requires an authenticated platform dispatcher. Route-setter execution depends on the scout completing first." | `fallbackPlanningDependencyBehavior` | MATCH |
| dependency_behaviors.planning_extended | "...Supporting planning castes may contribute evidence..." | `fallbackPlanningExtendedDependencyBehavior` | MATCH |
| fallback_behaviors.survey | "If any surveyor fails... synthesize survey artifacts locally..." | `fallbackSurveyFallbackBehavior` | MATCH |
| **fallback_behaviors.planning** | "If the scout or route-setter fails... **synthesize planning artifacts locally**..." | `fallbackPlanningFallbackBehavior` = "...**does not fall back to local synthesis**... only explicit `` `aether plan --synthetic` `` may produce..." | **DIVERGE -- deliberately NOT folded, see Deviations** |
| result_collection_policies.survey | "Wrapper result artifacts must stay outside .aether/data..." | `fallbackSurveyResultCollectionPolicy` | MATCH |
| result_collection_policies.planning | "A structurally valid completed result wins..." | `fallbackPlanningResultCollectionPolicy` | MATCH |
| fallback_visibility.survey | [dispatch_mode, survey_warning, provider_diagnostics, artifact_source] | `fallbackSurveyFallbackVisibility` | MATCH |
| **fallback_visibility.planning** | [dispatch_mode, planning_warning, **provider_diagnostics**, artifact_source, plan_source, planning_loop] (6) | `fallbackPlanningFallbackVisibility` = [dispatch_mode, planning_warning, **synthetic, synthetic_warning**, artifact_source, plan_source, planning_loop] (7) | **DIVERGE -- deliberately NOT folded, see Deviations** |
| max_workers_per_phase | 10 | *(no field)* | **N/A -- unmodeled.** `dispatchContractPolicy` struct has no field for this key at all |
| spawn_depth_limits | {0:4,1:4,2:2,3:0} | *(no field)* | **N/A -- unmodeled** |
| timeout_defaults | {build:600, continue:300, plan:300, verify:180} | *(no field)* | **N/A -- unmodeled** (the `effective*Timeout` functions use separate hardcoded package vars, unconnected to this YAML) |

**Result: 11 of 13 modeled fields matched. 2 fields (both "planning" fallback fields) diverged and were deliberately NOT folded -- see Deviations for the full evidence trail. 3 fields were never modeled by the loader at all** (confirmed unread by production code, but WAS read by `cmd/policy_schema_test.go`'s independent schema check -- see Deviations).

## Task 2/3: Real Dev-Checkout CLI Capture (Layer 2 Proof)

Both captures were produced by the real compiled `aether` binary, run from this repo's root (the only environment where the CWD-relative `colony/...` paths ever resolve), before touching any file and again after both files were deleted and all fixes landed.

### colony-prime (`aether colony-prime --compact`, `.result.context` field)

```
$ diff /tmp/aether-191-baseline-colony-prime.txt /tmp/aether-191-final-colony-prime.txt
$ echo $?
0
```

Empty diff. Byte-identical.

### dispatch-contract (temporary probe command exercising `surveyDispatchContractWithTimeout`/`planningDispatchContractWithTimeout`/`renderDispatchContract` -- see Deviations for why a probe was needed)

```
$ diff /tmp/aether-191-baseline-dispatch-contract.txt /tmp/aether-191-final-dispatch-contract.txt
13c13
<     "fallback_behavior": "If the scout or route-setter fails, blocks, or times out after dispatch starts, emit dispatch_mode=fallback and synthesize planning artifacts locally while preserving any real worker artifacts that landed first.",
---
>     "fallback_behavior": "Normal planning does not fall back to local synthesis. If Scout or Route-Setter workers are unavailable, blocked, failed, or timed out, stop with recovery guidance; only explicit `aether plan --synthetic` may produce dispatch_mode=synthetic local preview artifacts.",
17c17,18
<       "provider_diagnostics",
---
>       "synthetic",
>       "synthetic_warning",
31c32
<   "planning_rendered": "...Fallback: If the scout or route-setter fails... synthesize planning artifacts locally... Visibility: dispatch_mode, planning_warning, provider_diagnostics, artifact_source, plan_source, planning_loop...",
---
>   "planning_rendered": "...Fallback: Normal planning does not fall back to local synthesis... Visibility: dispatch_mode, planning_warning, synthetic, synthetic_warning, artifact_source, plan_source, planning_loop...",
$ echo $?
1
```

**This diff is real, expected, and deliberate** -- it touches exactly the 2 documented exception fields and nothing else (survey's map/rendered text and every other planning field: `execution_model`, `wave_count`, `worker_count`, timeouts, `dependency_behavior`, `deadline_policy`, `coordination_path`, `artifact_paths`, `result_artifact_paths`, `result_collection_policy` are all byte-identical, confirmed by the same diff showing zero other hunks). See Deviations for why.

## Task 3: Reappearance Ratchet Fail-Then-Pass Transcript

```
$ go test ./cmd/ -run 'TestColonyPrimeAndDispatchContractDoNotReappear' -v -count=1
# (both files deleted) -> PASS/PASS

$ cp cmd/testdata/191-02-colony-prime-original.md colony/prompts/colony-prime.md
$ go test ./cmd/ -run 'TestColonyPrimeAndDispatchContractDoNotReappear' -v -count=1
--- FAIL: TestColonyPrimeAndDispatchContractDoNotReappear/ColonyPrimeMdStaysDeleted
    colony_prime_and_dispatch_contract_ratchet_test.go:34: colony/prompts/colony-prime.md has reappeared -- ...
--- PASS: TestColonyPrimeAndDispatchContractDoNotReappear/DispatchContractYamlStaysDeleted

$ rm colony/prompts/colony-prime.md
$ go test ... # -> PASS/PASS again

$ cp cmd/testdata/191-02-dispatch-contract-original.yaml colony/policies/dispatch-contract.yaml
$ go test ./cmd/ -run 'TestColonyPrimeAndDispatchContractDoNotReappear' -v -count=1
--- PASS: TestColonyPrimeAndDispatchContractDoNotReappear/ColonyPrimeMdStaysDeleted
--- FAIL: TestColonyPrimeAndDispatchContractDoNotReappear/DispatchContractYamlStaysDeleted
    colony_prime_and_dispatch_contract_ratchet_test.go:43: colony/policies/dispatch-contract.yaml has reappeared -- ...

$ rm colony/policies/dispatch-contract.yaml
$ go test ... # -> PASS/PASS, final state
```

Each subtest failed independently, naming exactly its own file, confirming the ratchet isolates failures per-file rather than aggregating (per 191-PATTERNS.md's Tier-1 shape).

## Decisions Made

- **Zero fold needed for colony-prime.md.** All 16 sections' consumed fields already matched their compiled defaults; the before/after regression test passed on the first run, before any edit.
- **Deliberately did not fold 2 of dispatch-contract.yaml's fields** (see Deviations #1 below) -- the single most consequential finding of this plan.
- **Reused the existing `resetDispatchPolicyState()`** from `cmd/dispatch_contract_test.go` rather than writing a second reset function, since both files compile into the same `cmd` test binary.
- **Chose a temporary hidden probe cobra command** (`probe-191-02-dispatch-contract`, added and removed within this plan, never committed) for the dispatch-contract Layer-2 CLI capture, since no existing standalone CLI command renders it the way `aether colony-prime` does for loader 1. Explicitly permitted by 191-CONTEXT.md's "Claude's Discretion" list.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug, self-caught] Reverted a fold that would have regressed tested, deliberate fail-closed behavior**
- **Found during:** Task 3, after the initial fold + `go test ./cmd/... -count=1` full-suite run
- **Issue:** Following the plan's literal fold-then-delete instruction, I folded `colony/policies/dispatch-contract.yaml`'s stale `fallback_behaviors.planning` text ("synthesize planning artifacts locally...") and `fallback_visibility.planning` list into `cmd/codex_dispatch_contract.go`'s `fallbackPlanningFallbackBehavior`/`fallbackPlanningFallbackVisibility` constants, exactly as 191-CONTEXT.md's D-04 process describes. This immediately broke 4 pre-existing, passing tests: `TestPlanIncludesDispatchContract` (`cmd/codex_plan_test.go`), `TestPlanVisualOutputShowsDispatchContractDetails` (`cmd/codex_visuals_test.go`), and `TestGoldenPlanVisualOutput` (against the committed `cmd/testdata/golden_plan.txt` golden fixture) -- all three independently assert the fallback text says "does not fall back to local synthesis... only explicit `aether plan --synthetic`" and the visibility list includes `synthetic`/`synthetic_warning`. None of these three tests could ever have exercised the real YAML file (Go's test runner sets CWD to the package directory, so the bare `colony/policies/...` path never resolves in a test) -- meaning all three were deliberately asserting the Go fallback's OWN text as correct, and the colony/ file had drifted stale relative to a real, shipped, fail-closed design change, not the reverse.
- **Fix:** Reverted `fallbackPlanningFallbackBehavior` and `fallbackPlanningFallbackVisibility` to their exact original (pre-plan, git HEAD) values -- confirmed byte-for-byte via `git show HEAD:cmd/codex_dispatch_contract.go` diffed against the restored text. Added doc comments on both constants explaining the exception and pointing to this SUMMARY. Rewrote `cmd/codex_dispatch_contract_test.go`'s before/after regression test to assert full field-by-field byte-identity for every field EXCEPT these two (via a named, reusable exception list), and to separately assert the post-deletion value is specifically the tested fail-closed text -- not merely "different from before" -- so the exception is proven, not silently excused.
- **Files modified:** `cmd/codex_dispatch_contract.go`, `cmd/codex_dispatch_contract_test.go`
- **Verification:** `TestPlanIncludesDispatchContract`, `TestPlanVisualOutputShowsDispatchContractDetails`, `TestGoldenPlanVisualOutput`, and `TestDispatchContractYamlDeletionProducesByteIdenticalOutput` all pass; full `go test ./cmd/... -count=1` passes (380-490s, 3 separate foreground runs)
- **Committed in:** `09d509bd` (Task 3 commit)

**2. [Rule 3 - Blocking] Removed 3 dangling schema-check rows from `cmd/policy_schema_test.go`**
- **Found during:** Task 3, full-suite verification after deleting `colony/policies/dispatch-contract.yaml`
- **Issue:** `TestPolicySchemaRequiredFields` independently reads `colony/policies/dispatch-contract.yaml` directly (a completely separate code path from `loadDispatchContractPolicy()` -- a raw `os.ReadFile` + generic YAML unmarshal, asserting 3 fields: `max_workers_per_phase`, `spawn_depth_limits`, `timeout_defaults`). This is a genuine second reader of the file that Task 1's "multi-surface zero-readership proof" sweep (focused on `loadDispatchContractPolicy()`'s own struct/call sites) did not catch. Deleting the file broke this test with "read failed: no such file or directory."
- **Fix:** Removed the 3 `dispatch-contract.yaml` rows from `policySchemaChecks`, with a comment explaining why (file deleted in this plan, fields were never modeled by the runtime loader in the first place). Also corrected the file's own top-of-file doc comment, which described dispatch-contract.yaml as a currently-existing "live file" -- now stale as a direct result of this deletion.
- **Files modified:** `cmd/policy_schema_test.go`
- **Verification:** `TestPolicySchemaRequiredFields` passes
- **Committed in:** `09d509bd` (Task 3 commit)
- **Cross-plan overlap flag:** `cmd/policy_schema_test.go` also validates `model-routing.yaml`, `memory-rules.yaml`, and `autopilot.yaml` -- the exact 3 files sibling plan 191-01 deletes (per its own ROADMAP criterion 1 scope). 191-CONTEXT.md's Integration Points section states "Plans 191-01 through 191-06 share zero files with each other," but this file is very likely a genuine, independently-discovered exception: 191-01's own full-suite verification will hit the same "file not found" failure for its 3 files' rows and will need to edit this same file. My edit here touches only the `dispatch-contract.yaml` block (a 7-line removal plus a comment), non-adjacent to where 191-01's rows would be -- a trivial line-based merge in the worst case, but flagging for the orchestrator to check at merge time.

---

**Total deviations:** 2 auto-fixed (1 self-caught regression, 1 blocking/missing-reference fix)
**Impact on plan:** Deviation #1 is the most consequential finding of this plan -- executing the literal plan instruction would have shipped a real, silent regression to every installed Aether user (weakening a fail-closed safety design back to a "silently synthesize locally" one) for the sake of an abandoned dev-checkout-only config file. Caught by running the full test suite before considering Task 3 done, per this plan's own "verify by execution" discipline. Deviation #2 is a mechanical, low-risk consequence of deletion with a flagged (not yet confirmed) cross-plan file-overlap risk.

## Issues Encountered

- **`TestCLIExternalAdapterBuildContract/writes_claimed_output`** failed once during a full-suite run with "worker timeout after 1s". Re-ran in isolation (`go test ./cmd/ -run TestCLIExternalAdapterBuildContract -v -count=1`) and it passed cleanly in 23s including that subtest. This is a pre-existing, timing-sensitive test unrelated to this plan's files (not in `files_modified`, not touched by either commit) -- consistent with known CI flakiness in this repo. Not fixed (out of scope, SCOPE BOUNDARY rule); confirmed clean on the final 2 full-suite foreground runs (`go test ./cmd/... -count=1`, 384s and 323s, both `ok`).
- **The plan's assumption that `cmd/codex_dispatch_contract.go` had "no dedicated test file today"** (191-CONTEXT.md `<code_context>`) was incorrect -- `cmd/dispatch_contract_test.go` already existed with 5 tests covering `loadDispatchContractPolicy()`. Did not affect the outcome: `cmd/codex_dispatch_contract_test.go` (the plan's literal `files_modified` target, a different filename) was created as instructed, and its `resetDispatchPolicyState()` reuse from the existing file worked without modification, confirming 191-CONTEXT.md's own hedge ("read_first must confirm whether one is needed... following ... shape if so") was resolved as "not needed."

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Both of this plan's loaders (colony-prime.md, dispatch-contract.yaml) are resolved: deleted, folded where safe, deliberately not folded where evidence proved the fold would regress tested behavior, and locked against reappearance.
- Sibling plans 191-03 (`visuals.md`) and 191-04 (`review-depth.yaml`) remain for the other 2 of the 4 CWD-relative loaders -- untouched by this plan.
- **Flag for 191-07 (final verification, Wave 2):** check whether sibling plan 191-01 also edited `cmd/policy_schema_test.go` (see Deviation #2's cross-plan overlap note) and confirm the merged file still passes `TestPolicySchemaRequiredFields` with all four deleted policy files' rows removed cleanly.
- Per this plan's instructions, STATE.md and ROADMAP.md were deliberately NOT updated -- that is the orchestrator's job after all Wave 1 plans land.

---
*Phase: 191-dead-wood*
*Completed: 2026-08-21*

## Self-Check: PASSED

All 6 created/modified files under `cmd/` confirmed present (`test -f`, individually, one call per file). Both deleted files (`colony/prompts/colony-prime.md`, `colony/policies/dispatch-contract.yaml`) confirmed absent. Both task commit hashes (`b893a8b7`, `09d509bd`) confirmed present via `git log --oneline --all`. This SUMMARY.md itself confirmed present at its stated path.
