---
phase: 173-delegation-guard
plan: 07
subsystem: infra
tags: [hooks, claude-code, spawn-guard, fail-closed, go]

# Dependency graph
requires:
  - phase: 173-delegation-guard
    provides: "173-01's dated verdict HOOK_FIRES_IN_SUBAGENT and the observed payload contract (tool_name \"Agent\"; agent_id/agent_type present only on subagent-originated dispatches) this plan's deny path matches on"
  - phase: 173-delegation-guard
    provides: "173-04's spawnMaxDelegationDepth constant, the shared cap this hook reuses rather than inventing a second number"
provides:
  - "hookSpawnDenyReason: a fail-closed PreToolUse deny path for Agent/Task dispatches, denying whenever the requester's depth cannot be resolved from the fields Claude Code actually supplies"
  - "The only chokepoint in this phase that sits in front of the dispatch tool call itself, rather than inside the child's own turn"
  - "Named residue: no bridge from Claude Code's agent_id to Aether's spawn-tree AgentName; the hook's rule is a heuristic over observed fields, not an identity lookup, and spawn-log's deriveSpawnDepth remains the authoritative enforcement"
affects: [173-10]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Fail-closed inversion of a fail-open sibling: hookSpawnDenyReason copies protectedHookWriteReason's \"reason string on deny, empty on allow\" shape but deliberately inverts its unresolvable-input branch"
    - "Bounded-coverage comment: the deny function's own doc comment names exactly which dispatch levels were empirically observed (173-HOOK-FINDINGS.md, dated) rather than implying coverage the capture did not demonstrate"

key-files:
  created: []
  modified:
    - cmd/hook_cmds.go
    - cmd/hook_cmds_test.go

key-decisions:
  - "agent_type's 'general-purpose' value is treated as unresolved (deny), not as a resolved depth, because 173-HOOK-FINDINGS.md showed agent_type names the REQUESTER's own dispatched type, not the target's -- and .aether/workers.md instructs every worker to dispatch its own helper with subagent_type=\"general-purpose\". A first-tier worker following that fallback verbatim is therefore indistinguishable, on this one field, from a genuine second-tier helper. Guessing either way would be the identity-lookup mistake the code's own comment warns against, so both classify as unresolved."
  - "A first-tier worker is recognized only by an agent_type carrying the repo's aether-* naming prefix (aether-builder, aether-watcher, ...), confirmed against the 27 real agent definitions in .claude/agents/ant/. No other value is treated as resolved, per the plan's instruction to implement exactly the rule the findings support and no more."
  - "Test 4 (TestHookPreToolUseDeniesPastTheDepthCap) exercises the unresolved-general-purpose case, not a genuine resolved-depth-2 cap denial, because the capture in 173-HOOK-FINDINGS.md could not produce a payload that resolves to an authoritative depth-2 requester. The test's own doc comment states this bound explicitly, per the plan's requirement."

patterns-established:
  - "hookSpawnDenyReason(claudeHookInput) string is the single spawn-deny chokepoint on the PreToolUse hook, called from exactly one site in hookPreToolUseCmd.RunE, sibling to the existing Write/Edit branch."

requirements-completed: [SPAWN-04]

# Metrics
duration: ~35min
completed: 2026-08-13
---

# Phase 173 Plan 07: PreToolUse hook fail-closed delegation deny path Summary

**Added `hookSpawnDenyReason` to `cmd/hook_cmds.go`: the PreToolUse hook now refuses an Agent/Task dispatch whenever it cannot resolve who is asking, inverting the fail-open branch of its nearest sibling `protectedHookWriteReason`, matched against the real field names 173-01 captured from a live nested dispatch.**

## Performance

- **Duration:** ~35 min
- **Tasks:** 2 completed
- **Files modified:** 2

## Accomplishments

- The hook now sits in front of the Task tool call itself and can refuse a delegation *before* the platform acts on it -- the only guard in this phase with that property; every other guard runs inside the child's own turn after it already exists
- `hookSpawnDenyReason` allows the main session (no `agent_id`) and a first-tier worker (`agent_type` carrying the repo's `aether-*` naming convention), and denies everything else, including the documented `general-purpose` fallback, which the findings proved is ambiguous between a first-tier worker and a second-tier helper
- The deny path uses `spawnMaxDelegationDepth` directly -- the same constant `spawn-can-spawn`/`spawn-log` enforce -- so raising or lowering the CLI cap cannot silently leave the hook out of sync
- Four new named tests prove the guard cannot ship as either an always-block (would stop the coordinator dispatching its own workers) or a no-op (would let an unidentifiable requester through); both directions were red-proofed by temporarily breaking `hookSpawnDenyReason` and confirming the correct tests flip to red, then restoring and confirming zero residual diff
- A comment block above `hookSpawnDenyReason` states its coverage as a bounded, dated claim tied to `173-HOOK-FINDINGS.md`, names the missing `agent_id`-to-`AgentName` bridge as residue, and states that `spawn-log`'s `deriveSpawnDepth` remains the authoritative enforcement

## Task Commits

Each task was committed atomically:

1. **Task 1: A fail-closed Task branch on the existing hook** - `959f1f05` (feat)
2. **Task 2: Red-proofs for the hook's deny and allow paths** - `00692f03` (test)

**Plan metadata:** _pending_ (docs: complete plan — added by the orchestrator after all worktree agents in this wave merge)

## Files Created/Modified

- `cmd/hook_cmds.go` - Added `hookAetherAgentTypePrefix` constant and `hookSpawnDenyReason(claudeHookInput) string`; wired a new `Agent`/`Task` branch into `hookPreToolUseCmd.RunE` as a sibling of the existing `Write`/`Edit` branch, following the same `tracer.LogIntervention`-then-`emitHookBlock` pattern with hook key `hook.pre-tool-use.spawn-deny`. The existing `Write`/`Edit` branch is unchanged in structure.
- `cmd/hook_cmds_test.go` - Added four tests: `TestHookPreToolUseDeniesUnresolvedRequesterDepth`, `TestHookPreToolUseAllowsMainSessionDispatch`, `TestHookPreToolUseAllowsFirstTierWorkerDispatch`, `TestHookPreToolUseDeniesPastTheDepthCap`. Payload fixtures copy the shape and field names from the real capture in `173-HOOK-FINDINGS.md` (`session_id`, `agent_id`, `agent_type`, `tool_name: "Agent"`, `tool_input`), not invented names.

## Decisions Made

- **`general-purpose` classifies as unresolved, not as a resolved depth.** See key-decisions above -- this is the load-bearing reading of 173-HOOK-FINDINGS.md's finding that `agent_type` names the requester, not the target, and it is what keeps the guard from silently waving through a second-tier helper that happens to reuse the documented fallback string.
- **First-tier recognition is by `aether-*` prefix, not a hardcoded list of the 27 caste names.** Both satisfy "a value matching the repo's named aether-* agent types"; the prefix match is less fragile to a new caste being added later and was confirmed against every file in `.claude/agents/ant/`.
- **Test 4's cap-denial scenario is the unresolved case, not a true resolved-depth-2 case**, because the capture cannot produce one -- documented explicitly in the test's own comment, per the plan's requirement not to claim coverage the findings did not demonstrate.

## Deviations from Plan

None - plan executed exactly as written. The verdict gate at the top of Task 1 (stop on `INCONCLUSIVE`) did not trigger: `173-HOOK-FINDINGS.md`'s `VERDICT:` line reads `HOOK_FIRES_IN_SUBAGENT`, so implementation proceeded using the observed field names (`agent_id`, `agent_type`, `tool_name: "Agent"`) exactly as captured, with no struct-tag corrections needed (the hypothesis field names from 173-01 Task 1 already matched the real capture).

## Red-Proof Evidence (D-22)

Performed after Task 2's four tests were all green, `hookSpawnDenyReason`'s body was temporarily replaced (not committed) to always return `""`:

```
=== RUN   TestHookPreToolUseDeniesUnresolvedRequesterDepth
    hook_cmds_test.go:509: unmarshal hook output: unexpected end of JSON input ("")
--- FAIL: TestHookPreToolUseDeniesUnresolvedRequesterDepth (0.00s)
=== RUN   TestHookPreToolUseAllowsMainSessionDispatch
--- PASS: TestHookPreToolUseAllowsMainSessionDispatch (0.00s)
=== RUN   TestHookPreToolUseAllowsFirstTierWorkerDispatch
--- PASS: TestHookPreToolUseAllowsFirstTierWorkerDispatch (0.00s)
=== RUN   TestHookPreToolUseDeniesPastTheDepthCap
    hook_cmds_test.go:616: unmarshal hook output: unexpected end of JSON input ("")
--- FAIL: TestHookPreToolUseDeniesPastTheDepthCap (0.00s)
```

Tests 1 and 4 (the deny paths) went red as required, while tests 2 and 3 (the allow paths, the negative controls) stayed green -- proving the deny tests cannot be satisfied by a hook that allows every dispatch.

Then temporarily replaced to always return a constant deny string:

```
=== RUN   TestHookPreToolUseDeniesUnresolvedRequesterDepth
--- PASS: TestHookPreToolUseDeniesUnresolvedRequesterDepth (0.00s)
=== RUN   TestHookPreToolUseAllowsMainSessionDispatch
    hook_cmds_test.go:547: main session dispatch was blocked: {"decision":"block","reason":"always deny (temporary, D-22)"}
--- FAIL: TestHookPreToolUseAllowsMainSessionDispatch (0.00s)
=== RUN   TestHookPreToolUseAllowsFirstTierWorkerDispatch
    hook_cmds_test.go:576: first-tier worker's own dispatch was blocked: {"decision":"block","reason":"always deny (temporary, D-22)"}
--- FAIL: TestHookPreToolUseAllowsFirstTierWorkerDispatch (0.00s)
=== RUN   TestHookPreToolUseDeniesPastTheDepthCap
--- PASS: TestHookPreToolUseDeniesPastTheDepthCap (0.00s)
```

Tests 2 and 3 (the allow paths) went red as required, proving those tests genuinely require an allow answer and cannot be satisfied by a hook that blocks every dispatch -- which would also have broken the coordinator's own worker dispatches and every build. (The always-deny stub also incidentally failed two pre-existing, unrelated capture tests that happen to use `tool_name: "Task"` with an empty payload -- expected collateral of a deliberately broken stub, not a real regression.) The function was then restored verbatim; `git diff cmd/hook_cmds.go` showed zero residual changes, and all four new tests plus every pre-existing `TestHookPreToolUse*`/`TestHookStop*`/`TestHookPreCompact*` test passed green afterward.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 10's final-sweep assertions have one more delegation guard to confirm: `hookSpawnDenyReason` exists, is called from exactly one site, and uses `spawnMaxDelegationDepth` directly.
- No blockers for plan 10.

## Threat Flags

None - all threat surface introduced by this plan (T-173-34 through T-173-39) was already named in this plan's own `<threat_model>` and is not new surface beyond what the plan anticipated.

---
*Phase: 173-delegation-guard*
*Completed: 2026-08-13*

## Self-Check: PASSED

- FOUND: cmd/hook_cmds.go
- FOUND: cmd/hook_cmds_test.go
- FOUND: .planning/phases/173-delegation-guard/173-07-SUMMARY.md
- FOUND commit 959f1f05 (Task 1)
- FOUND commit 00692f03 (Task 2)
