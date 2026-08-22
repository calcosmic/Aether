---
phase: 173-delegation-guard
plan: 10
subsystem: infra
tags: [ci, go-test, wiring-ratchet, fail-closed, delegation, spawn-guard, roadmap]

# Dependency graph
requires:
  - phase: 173-delegation-guard
    provides: "Plans 02-09's five guard implementations and their test files (spawn_failclosed_test.go, spawn_budget_test.go, spawn_ancestor_test.go, spawn_tree_view_test.go, spawn_reap_test.go) — this plan puts each under the CI ratchet and proves the cross-guard fail-closed property"
  - phase: 172-wiring-proof
    provides: "wiringGateGuardFiles / TestWiringGateStepRunsEveryWiringTest, the named-CI-step ratchet this plan extends rather than replaces"
provides:
  - "Every one of this phase's five new guard test files is enumerated in wiringGateGuardFiles and every top-level test function in them is in the CI -run filter — deleting any of them fails CI by name"
  - "TestNoDelegationGuardContractStubSurvives, proving neither CONTRACT-STUB-PLAN-05 nor -06 survived in code or comment"
  - "TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs, the cross-guard D-19 proof over an enumerated (guard, axis) table, each axis matched to the file that guard genuinely reads"
  - "TestDelegationGuardTableCoversEveryGuardCommand, the self-invalidating guard on the table's own coverage and narrowing"
  - "173-RESIDUE.md, naming nine bounded residues and mapping every ROADMAP success criterion and SPAWN requirement to the command that proves it"
  - "ROADMAP Phase 173 criterion 2 corrected to the D-05-derived depth number"
affects: [174-spend-ledger]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Cross-guard fault-injection table (delegationGuardFaultTable): each guard command carries its own list of fault axes matched to what it actually reads, rather than a cross product of every file against every guard — a per-guard fault the guard is not subject to is refused a slot in the table by construction"
    - "Self-invalidating narrowing assertion: TestDelegationGuardTableCoversEveryGuardCommand AST-parses the three decision-chain source files for a COLONY_STATE.json string literal (comments excluded), so a guard that later starts reading colony state fails the build and names both the table and 173-RESIDUE.md as needing to widen"
    - "Registered-command enumeration over a hardcoded list: the coverage test walks rootCmd.Commands() for spawn-can-spawn*/spawn-log/hook-pre-tool-use, so a future fifth guard command must join the table or fail the build"

key-files:
  created:
    - .planning/phases/173-delegation-guard/173-RESIDUE.md
  modified:
    - cmd/ci_wiring_gate_test.go
    - .github/workflows/ci.yml
    - cmd/spawn_failclosed_test.go
    - .planning/ROADMAP.md

key-decisions:
  - "cmd/testdata/orphan_allowlist.json earned zero removals. The plan candidate list (spawn-can-spawn, spawn-tree-depth, spawn-tree-active, spawn-get-depth, spawn-can-spawn-swarm) was tested empirically against TestNoRegisteredSubcommandIsUnreferenced, one entry removed at a time; all five remained genuine orphans. .aether/workers.md documents aether spawn-can-spawn as the spawn protocol, but workers.md is prose documentation, not one of the ratchet's caller corpora (.claude/commands/ant/, .opencode/commands/ant/, .aether/commands/*.yaml, .aether/utils/hooks/*.js, scripts/*.sh, Go self-invocation) — so it does not count as caller evidence, and no caller was invented to force a removal."
  - "ROADMAP criterion 4 left byte-identical per the plan's explicit instruction. TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs proves each guard denies against the inputs it actually reads, not that all four deny specifically because COLONY_STATE.json is unreadable (three of four never read it) — 173-RESIDUE.md residue 7 states this bound; whether the evidence satisfies the criterion as written is left to verification, not decided or narrowed by this plan."
  - "TestDelegationGuardTableCoversEveryGuardCommand's narrowing check is AST-based (go/parser over cmd/spawn.go, cmd/spawn_budget.go, cmd/spawn_ancestor.go), not grep, specifically so 173-RESIDUE.md's own prose mention of COLONY_STATE.json cannot trip the check that residue document is supposed to make honest."

patterns-established:
  - "delegationGuardFaultAxis{Name, NotExercisable, Verify}: a fault-injection table entry shape that forces a skipped axis to carry a stated reason rather than silently vanishing"

requirements-completed: [SPAWN-04, SPAWN-06]

# Metrics
duration: ~70min
completed: 2026-08-13
---

# Phase 173 Plan 10: CI Ratchet, Cross-Guard Fail-Closed Proof, and Named Residue Summary

**Put all five of this phase's new guard test files under the CI wiring ratchet, added one table-driven test proving every delegation guard denies against the inputs it actually reads (not a file it doesn't), confirmed the allowlist earned zero removals by running the ratchet rather than reading it, and wrote a nine-item residue document plus one ROADMAP number correction.**

## Performance

- **Duration:** ~70 min
- **Tasks:** 4 completed
- **Files modified:** 5 (4 modified, 1 created)

## Accomplishments

- `wiringGateGuardFiles` and the CI `-run` filter now cover `spawn_failclosed_test.go`, `spawn_budget_test.go`, `spawn_ancestor_test.go`, `spawn_tree_view_test.go` and `spawn_reap_test.go` — deleting any test function in any of these five files now fails CI by name, closing the exact gap CLAUDE.md's Definition of Done calls out repeatedly
- `TestNoDelegationGuardContractStubSurvives` proves plan 04's two contract-stub markers (`CONTRACT-STUB-PLAN-05`, `CONTRACT-STUB-PLAN-06`) never shipped, in code or as a leftover comment
- `TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs` proves, in one command, that `spawn-can-spawn`, `spawn-log`, `spawn-can-spawn-swarm` and `hook-pre-tool-use` each deny with a non-empty reason when the specific input each one actually reads is corrupted — never asserting a fault a guard is not subject to — paired with an all-valid control proving the same four guards allow when nothing is wrong
- `TestDelegationGuardTableCoversEveryGuardCommand` makes the proof self-invalidating: it enumerates registered commands (so a future fifth guard must join the table or fail the build), rejects an empty or silently-skipped fault axis, and AST-parses the decision-chain source for a `COLONY_STATE.json` string literal so a guard that starts reading colony state cannot leave the table's narrowing quietly wrong
- The tolerated-orphan list was tested empirically, not read — all five spawn-related candidates remained genuine orphans against the live ratchet, so the file is unchanged; the reason (documentation is not a caller corpus for this specific ratchet) is recorded rather than glossed over
- `173-RESIDUE.md` names nine bounded residues (advisory-vs-authoritative depth checks, the caller-supplied-parent gap, the hook's agent_id-to-AgentName bridge that does not exist, the wall-clock run-membership window, text-not-semantic ancestor matching, no liveness signal, the exact scope of the fail-closed proof, unverified wrapper-tier visibility of refusal reasons, and the zero-removal allowlist finding) and maps every ROADMAP success criterion and SPAWN requirement to the exact command that proves it
- ROADMAP criterion 2's stale `(3 for a 3-deep tree)` — provably wrong under the already-locked D-05 depth convention — is corrected to `2 for a three-level tree`; criterion 4 is byte-identical, gated by an exact-string check and a 2-line diff cap

## Task Commits

Each task was committed atomically:

1. **Task 1: Put this phase's guard files under the ratchet and prove no stub survived** - `bc6b70ac` (test)
2. **Task 2: Prove every delegation guard fails closed on its own inputs, in one test** - `69814de7` (test)
3. **Task 3: Shrink the tolerated-orphan list to match what is now genuinely called** - no commit (zero file changes; see Decisions Made)
4. **Task 4: Write down what this phase does not prove, and correct the one ROADMAP criterion** - `7ea758dd` (docs)

**Plan metadata:** _pending_ (docs: complete plan)

## Files Created/Modified

- `cmd/ci_wiring_gate_test.go` - Added the five new filenames to `wiringGateGuardFiles`; added `TestNoDelegationGuardContractStubSurvives`
- `.github/workflows/ci.yml` - Appended every top-level test name from the five new guard files, plus `TestNoDelegationGuardContractStubSurvives`, `TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs` and `TestDelegationGuardTableCoversEveryGuardCommand`, to the wiring step's `-run` filter. The blanket `go test ./...` release-gate step is untouched.
- `cmd/spawn_failclosed_test.go` - Added `delegationGuardFaultAxis`/`delegationGuardTableEntry` types, the `delegationGuardFaultTable` (spawn-can-spawn: run-state-unreadable, requester-not-recorded; spawn-log: run-state-unreadable, parent-not-recorded; spawn-can-spawn-swarm: colony-state-unreadable, spawn-tree-unreadable; hook-pre-tool-use: requester-identity-unresolvable), `delegationGuardTableAllValidControl`, `TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs`, and `TestDelegationGuardTableCoversEveryGuardCommand`
- `.planning/phases/173-delegation-guard/173-RESIDUE.md` (NEW) - "What this phase proves" (one command per criterion/requirement) and "What this phase does NOT prove" (nine bounded residues), plus a stop-rule section scoped to post-verification
- `.planning/ROADMAP.md` - Phase 173 success criterion 2's parenthetical corrected from `(3 for a 3-deep tree)` to `(2 for a three-level tree — coordinator depth 0, its workers depth 1, their helpers depth 2, per D-05)`; nothing else changed (verified: `git diff --numstat` totals 2 lines)
- `cmd/testdata/orphan_allowlist.json` - unchanged (zero diff; Task 3 earned no removals)

## Decisions Made

- **The tolerated-orphan list earned zero removals from this phase.** See key-decisions above — tested empirically per-candidate against the live ratchet rather than assumed from the plan's prose expectations.
- **ROADMAP criterion 4 was deliberately left untouched.** The cross-guard fail-closed test proves a narrower, more precise claim than the criterion's literal wording (each guard denies against what it reads, not "because COLONY_STATE.json is unreadable" for all four) — `173-RESIDUE.md` residue 7 records the gap; this plan does not resolve it either way, per its own explicit instruction that narrowing is verification's call, made with evidence, not a pre-build judgment.
- **The narrowing-guard AST check excludes comments by construction.** Using `go/parser`/`go/ast` over string literals only (not `grep`) means `173-RESIDUE.md`'s own prose naming `COLONY_STATE.json` can never trip the very check meant to keep that residue's claim honest.

## Deviations from Plan

None - plan executed exactly as written. Task 3's zero-removal outcome was explicitly anticipated and permitted by the plan text ("Do not remove an entry the ratchet still needs... Record in the plan summary... for each kept one the reason it is still an orphan") rather than a deviation from it.

## Red-Proof Evidence (D-22)

**Task 1, red-proof 1** (delete a test function from `cmd/spawn_budget_test.go`, confirm `TestWiringGateStepRunsEveryWiringTest` goes red naming the stale filter alternative):

```
=== RUN   TestWiringGateStepRunsEveryWiringTest
    ci_wiring_gate_test.go:226: CI step "Verify subcommand wiring and CLI flag contracts"'s -run filter contains 1 alternative(s) that match no guard test — go test -run exits 0 when a pattern matches nothing, so a renamed or deleted guard test left in the filter would silently stop running with nothing going red:
          TestSpawnTreeBudgetRefusesTheTwentyFirstHelper
--- FAIL: TestWiringGateStepRunsEveryWiringTest (0.01s)
```

Restored; `git diff cmd/spawn_budget_test.go` showed zero residual changes.

**Task 1, red-proof 2** (re-insert `CONTRACT-STUB-PLAN-05` as a comment in `cmd/spawn_budget.go`, confirm `TestNoDelegationGuardContractStubSurvives` goes red):

```
=== RUN   TestNoDelegationGuardContractStubSurvives
    ci_wiring_gate_test.go:1994: cmd/spawn_budget.go still contains "CONTRACT-STUB-PLAN-05" — a contract stub marker surviving in code OR a comment means the real check may never have replaced the always-allow stub
--- FAIL: TestNoDelegationGuardContractStubSurvives (0.00s)
```

Restored; `git diff cmd/spawn_budget.go` showed zero residual changes.

**Task 2, red-proof 1** (make `spawnTreeBudgetReason` return the empty string on error, confirm the table test goes red naming the run-state-unreadable axis):

```
=== RUN   TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/spawn-can-spawn/run-state-unreadable
    spawn_failclosed_test.go:279: spawn-can-spawn --enforce with an unreadable run state did not exit non-zero: stdout={"ok":true,"result":{"authoritative":false,"can_spawn":true,"depth":0}}
--- FAIL: TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/spawn-can-spawn/run-state-unreadable (0.00s)
=== RUN   TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/spawn-log/run-state-unreadable
    spawn_failclosed_test.go:368: spawn-log with an unreadable run state did not exit non-zero: stdout={"ok":true,"result":{...,"recorded":true,...}}
--- FAIL: TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/spawn-log/run-state-unreadable (0.00s)
```

Both `spawn-can-spawn` and `spawn-log` went red (they share `spawnTreeBudgetReason`'s error path), including the specific `spawn-log` axis the plan named. Restored; `git diff cmd/spawn_budget.go` showed zero residual changes.

**Task 2, red-proof 2** (make `hookSpawnDenyReason` return empty on its unresolved branch, confirm it goes red naming the hook's axis):

```
=== RUN   TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/hook-pre-tool-use/requester-identity-unresolvable
    spawn_failclosed_test.go:571: unmarshal hook output: unexpected end of JSON input ("")
--- FAIL: TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/hook-pre-tool-use/requester-identity-unresolvable (0.00s)
```

Restored; `git diff cmd/hook_cmds.go` showed zero residual changes.

**Task 2, red-proof 3** (add a `COLONY_STATE.json` string literal to `cmd/spawn_budget.go`, confirm `TestDelegationGuardTableCoversEveryGuardCommand` goes red on the narrowing assertion):

```
=== RUN   TestDelegationGuardTableCoversEveryGuardCommand
    spawn_failclosed_test.go:800: cmd/spawn_budget.go:13 contains a COLONY_STATE.json string literal ("COLONY_STATE.json") -- a delegation guard has started reading colony state, so this table's axes and 173-RESIDUE.md's narrowed claim must both be widened to match
--- FAIL: TestDelegationGuardTableCoversEveryGuardCommand (0.00s)
```

Restored; `git diff cmd/spawn_budget.go` showed zero residual changes.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The full test suite (`go test ./... -count=1 -timeout 900s`) passes across every package, and `go vet ./...` is clean.
- Phase 174 (Spend Ledger) depends on Phase 173's parent/depth linkage recorded at spawn time — nothing in this plan touches that recording path; it only proves the guards around it and documents their bound.
- `173-RESIDUE.md` residue 8 (whether the refusal reason renders inside `/ant-build`/`/ant-continue` ceremony narration) is named but not settled — it requires an observed build that hits the cap, which is wrapper-tier work out of scope for this phase.
- Verification is the owner of whether ROADMAP criterion 4 is satisfied as written, per residue 7; this plan deliberately did not decide that question.

## Threat Flags

None - all threat surface introduced by this plan (T-173-52 through T-173-60) was already named in this plan's own `<threat_model>` and mitigated by the tests described above; no new surface beyond what the plan anticipated.

---
*Phase: 173-delegation-guard*
*Completed: 2026-08-13*

## Self-Check: PASSED

- FOUND: cmd/ci_wiring_gate_test.go
- FOUND: .github/workflows/ci.yml
- FOUND: cmd/spawn_failclosed_test.go
- FOUND: .planning/phases/173-delegation-guard/173-RESIDUE.md
- FOUND: .planning/ROADMAP.md
- FOUND commit bc6b70ac (Task 1)
- FOUND commit 69814de7 (Task 2)
- FOUND commit 7ea758dd (Task 4)
