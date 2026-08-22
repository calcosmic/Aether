---
phase: 188-one-truth-for-failures-and-advances
plan: 07
subsystem: context-assembly, state-mutation-safety
tags: [colony-prime, midden, worker-context, state-mutate, json-case-insensitivity, guard-bypass]

# Dependency graph
requires:
  - phase: 188-one-truth-for-failures-and-advances
    provides: midden_shared.go's canonical loadMiddenFile/appendMiddenEntry pair, state-mutate's exact-case current_phase guard (plans 01, 03, 06)
provides:
  - A midden ("Recent Failures") section in buildColonyPrimeOutput, the real function every live worker dispatch calls for its context capsule
  - Case-insensitive current_phase guard enforcement on both state-mutate invocation forms (--field and expression syntax)
affects: [188-VERIFICATION.md re-verification, any future phase touching colony_prime_context.go or state_cmds.go]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Case-insensitive field-name matching before a switch dispatch, not a blanket lowercase of the whole switch (state_cmds.go's dispatchField pattern) -- closes a case-sensitivity bypass for exactly one field without changing behavior for any other field"
    - "Bounded-content protected sections (midden: unacknowledged-only, hard-capped at 5, newest-first) as the safe way to satisfy 'never trimmed' for content that could otherwise grow unboundedly"

key-files:
  created: []
  modified:
    - cmd/colony_prime_context.go
    - cmd/context_weighting.go
    - cmd/midden_unification_test.go
    - cmd/colony_prime_audit_test.go
    - cmd/state_cmds.go
    - cmd/state_cmds_test.go

key-decisions:
  - "Midden section placed in the protected (never-trimmed) tier alongside blockers, per this task's explicit instruction when CLAUDE.md's trim order didn't name midden -- made safe by hard-capping content to 5 unacknowledged entries rather than relying on trim-order luck"
  - "Fixed buildColonyPrimeOutputOpts (the function resolveCodexWorkerContext() actually calls) rather than buildContextCapsuleOutput (the rarely-hit fallback) -- the verification report named the former as the real gap"
  - "Expression-syntax fix normalizes the sub-expression to the canonical lowercase key before writing, not just before guarding -- --field mode already self-heals via its struct round-trip, but expression mode operates on raw JSON bytes throughout and would otherwise leave a stray differently-cased duplicate key in the file forever"

requirements-completed: []

# Metrics
duration: ~45min
completed: 2026-08-21
---

# Phase 188 Plan 07: Close the Two Verification Gaps Summary

**Wired midden.json into colony-prime's real worker-context capsule (not the dead pr-context command), and closed a case-insensitive bypass that let `state-mutate --field CURRENT_PHASE` move a colony's phase with zero precondition checks.**

## Performance

- **Duration:** ~45 min (approximate -- no precise start timestamp captured at session start)
- **Completed:** 2026-08-21
- **Tasks:** 2 (one per verification gap)
- **Files modified:** 6

## Accomplishments

- **Gap 1 closed:** `buildColonyPrimeOutputOpts` (via `resolveCodexWorkerContext()`, called by every live build/continue/colonize/plan/seal/swarm worker dispatch) now renders a "Recent Failures" section sourced from the canonical `loadMiddenFile` helper -- the function genuinely had zero midden-reading code before this fix, confirmed by reproducing the verifier's own probe as a permanent test.
- **Gap 2 closed:** `state-mutate`'s `current_phase` guard now matches case-insensitively on both invocation forms (`--field CURRENT_PHASE` and `.CURRENT_PHASE = N`), closing a bypass where Go's `encoding/json` case-insensitive struct-field matching let a differently-cased write silently move `current_phase` with no precondition check.
- Found and fixed two additional stale test fixtures (`TestColonyPrimeAAC005Audit`, `TestColonyPrimeSectionsPresent`) that were seeding midden data at the legacy nested path and asserting only a self-consistent readback of that same wrong path -- never actually checking the real worker capsule. Both now seed the canonical flat path and assert against `resolveCodexWorkerContext()`'s real output.

## Task Commits

1. **Task 1: Close Gap 2 (state-mutate case-insensitive bypass)** - `dd3e584c` (fix)
2. **Task 2: Close Gap 1 (colony-prime midden wiring)** - `00eb11ce` (fix)

## Files Created/Modified

- `cmd/state_cmds.go` - `executeFieldMode` now dispatches any casing of `current_phase` through the guarded branch (never the generic `setNestedFieldJSON` passthrough); `guardCurrentPhaseSubExpression` compares case-insensitively; new `normalizeCurrentPhaseExpressionCasing` rewrites the sub-expression to the canonical key before it is applied
- `cmd/state_cmds_test.go` - 6 new tests: case-insensitive/mixed-case refusal and matching-guard success, for both `--field` and expression syntax, plus a stray-duplicate-key check
- `cmd/colony_prime_context.go` - new "midden" section in `buildColonyPrimeOutputOpts`: unacknowledged entries only, newest-first, hard-capped at 5, using the canonical `loadMiddenFile` helper
- `cmd/context_weighting.go` - `sectionRelevanceScore("midden")` = 0.90; `protectedSectionPolicy("midden")` = protected, with the reasoning for that choice recorded inline
- `cmd/midden_unification_test.go` - 3 new tests proving the capsule carries a written failure exactly once, omits the section when there are no failures, and omits acknowledged entries; updated the existing test's doc comment to point at the new coverage instead of implying pr-context is the only fix
- `cmd/colony_prime_audit_test.go` - fixed two stale nested-path midden fixtures to use the canonical flat path; added a real assertion that the captured capsule text carries the seeded entry; added "midden" to the 16-section completeness checklist

## Decisions Made

**Trim-order placement (Gap 1).** CLAUDE.md's documented trim order (Rolling summary -> Phase learnings -> Key decisions -> Hive wisdom -> Context capsule -> User preferences -> QUEEN wisdom global/local -> Pheromone signals) never actually named midden -- it didn't exist as a colony-prime section before this fix, so there was nothing to place. Per this task's explicit instruction ("if ambiguous, put it with the failure/blocker tier"), the midden section is marked `protected: true` in `protectedSectionPolicy`, the same never-trimmed status blockers carry. Read `pkg/colony/context_ranking.go`'s `RankContextCandidates` before deciding: protected sections are included unconditionally, with no budget-fit check at all, which would be unsafe for a section whose backing file (`midden.json`) can grow without bound over a colony's life. The section is made safe against that by bounding its *content*, not by relying on trimming: only still-unacknowledged entries are eligible (an acknowledged failure has already been handled), sorted newest-first, hard-capped at 5 with a "+N more unacknowledged" tail -- so "never trimmed" never means "unbounded."

**Which capsule builder to fix.** `resolveCodexWorkerContextWithTrim()` calls `buildColonyPrimeOutput(true)` first and only falls back to `buildContextCapsuleOutput()` (cmd/context.go) when the primary result is empty -- which in practice only happens when `store == nil`, since both functions return empty in that case anyway. The verification report named `buildColonyPrimeOutput` specifically as "the primary path" with "zero references to midden anywhere in the file." I fixed that function only, left the rarely-hit fallback untouched, and confirmed by reading every call site (`proof_cmd.go`'s `buildProofContext` follows the identical either/or pattern, never both) that this is the only place the fix needs to live.

**One-home proof without a full 190-05-style breadth test.** The 190-05 pattern exists because pheromone content had two independent delivery channels (`ContextCapsule` and a separate `PheromoneSection` field) that could both be populated for the same worker. I checked whether an equivalent second channel exists for midden: grepping every midden reference in `cmd/*.go` turned up exactly two categories outside what I touched -- CLI-display code (`status.go`'s guided-action suggestion and unacknowledged-count warning) and archival housekeeping (`entomb_cmd.go`'s 30-day pruning) -- neither of which ever populates a worker's prompt. There is no sibling "MiddenSection" field on `codex.WorkerConfig`/`codex.WorkerDispatch` for midden content to collide with. Given that, the meaningful proof is the "exactly once" assertion inside the one channel that does carry it (`TestOneMiddenEntryReachesColonyPrimeCapsule`), not a duplicated 9-case breadth test guarding against a structurally-absent second channel.

**Expression-syntax normalization goes further than the guard.** The plan only strictly required refusal-when-unguarded for both invocation forms. I additionally added `normalizeCurrentPhaseExpressionCasing`, which rewrites a validated `.CURRENT_PHASE = N` expression to `.current_phase = N` before it reaches `applySubExpression`. Reason: `--field` mode already self-heals any duplicate-key risk because `executeFieldMode`'s `default` branch round-trips through the `colony.ColonyState` Go struct (`json.Marshal` -> `setNestedFieldJSON` -> `json.Unmarshal` -> later `store.SaveJSON`), which collapses two differently-cased keys back into one canonical field before anything is saved. `executeExpression`, by contrast, operates on raw JSON bytes via sjson/gjson from start to finish and never touches the struct -- so even a *legitimately guarded* `.CURRENT_PHASE = N` would otherwise leave a permanent, differently-cased duplicate key sitting in `COLONY_STATE.json`. This was confirmed by direct execution during the RED phase (see Proof below) and is now covered by a dedicated regression test asserting no stray key exists after a successful guarded mutation.

## Deviations from Plan

**1. [Rule 1 - Bug] Fixed two additional stale test fixtures beyond the two named gaps**
- **Found during:** Gap 1 verification pass (checking for other consumers of the fixed function)
- **Issue:** `TestColonyPrimeAAC005Audit` and `TestColonyPrimeSectionsPresent` both seeded midden data at the legacy nested `midden/midden.json` path (the same wrong path 188-VERIFICATION.md's evidence describes `TestGoldenAutopilotPauseConditions` doing) and asserted only a readback from that same wrong path -- never checking whether the entry reached the actual capsule text. This is the identical self-consistency trap the verification report's own probe methodology exists to catch, sitting immediately adjacent to the code I was fixing.
- **Fix:** Re-seeded both fixtures at `middenCanonicalPath` ("midden.json"); added a real assertion in `TestColonyPrimeAAC005Audit` that the captured `resolveCodexWorkerContext()` output contains the seeded message; added "midden" to `TestColonyPrimeSectionsPresent`'s 16-section completeness checklist so a future regression that silently drops the section would be caught.
- **Files modified:** `cmd/colony_prime_audit_test.go`
- **Commit:** `00eb11ce` (part of the Gap 1 commit)

## Issues Encountered

None beyond the two gaps themselves and the stale-fixture deviation documented above.

## Proof (fail-then-pass, quoted)

### Gap 2 -- state-mutate case-insensitive bypass

**RED** (`go test ./cmd/ -run 'TestStateMutateFieldCaseInsensitiveCurrentPhaseRequiresGuard|...' -v`, before the fix):

```
=== RUN   TestStateMutateFieldCaseInsensitiveCurrentPhaseRequiresGuard
    state_cmds_test.go:537: expected state-mutate --field CURRENT_PHASE with no --guard to be refused, got: map[ok:true result:map[field:CURRENT_PHASE updated:true value:99]]
--- FAIL: TestStateMutateFieldCaseInsensitiveCurrentPhaseRequiresGuard (0.00s)
=== RUN   TestStateMutateExpressionCaseInsensitiveCurrentPhaseRequiresGuard
    state_cmds_test.go:655: expected `.CURRENT_PHASE = N` with no --guard to be refused, got: map[ok:true result:map[expr:.CURRENT_PHASE = 99 updated:true]]
--- FAIL: TestStateMutateExpressionCaseInsensitiveCurrentPhaseRequiresGuard (0.00s)
=== RUN   TestStateMutateExpressionCaseInsensitiveCurrentPhaseSucceedsWithMatchingGuard
    state_cmds_test.go:740: COLONY_STATE.json has a stray CURRENT_PHASE key alongside current_phase: {
          ...
          "current_phase": 1,
          ...
          "CURRENT_PHASE": 2
        }
--- FAIL: TestStateMutateExpressionCaseInsensitiveCurrentPhaseSucceedsWithMatchingGuard (0.00s)
FAIL
```
(6 tests run, 5 failed -- exactly reproducing the verifier's two probes, plus the predicted duplicate-key artifact for expression mode.)

**GREEN** (same command, after the fix):

```
=== RUN   TestStateMutateFieldCaseInsensitiveCurrentPhaseRequiresGuard
--- PASS: TestStateMutateFieldCaseInsensitiveCurrentPhaseRequiresGuard (0.00s)
=== RUN   TestStateMutateFieldMixedCaseCurrentPhaseRequiresGuard
--- PASS: TestStateMutateFieldMixedCaseCurrentPhaseRequiresGuard (0.00s)
=== RUN   TestStateMutateFieldCaseInsensitiveCurrentPhaseSucceedsWithMatchingGuard
--- PASS: TestStateMutateFieldCaseInsensitiveCurrentPhaseSucceedsWithMatchingGuard (0.00s)
=== RUN   TestStateMutateExpressionCaseInsensitiveCurrentPhaseRequiresGuard
--- PASS: TestStateMutateExpressionCaseInsensitiveCurrentPhaseRequiresGuard (0.00s)
=== RUN   TestStateMutateExpressionMixedCaseCurrentPhaseRequiresGuard
--- PASS: TestStateMutateExpressionMixedCaseCurrentPhaseRequiresGuard (0.00s)
=== RUN   TestStateMutateExpressionCaseInsensitiveCurrentPhaseSucceedsWithMatchingGuard
--- PASS: TestStateMutateExpressionCaseInsensitiveCurrentPhaseSucceedsWithMatchingGuard (0.00s)
PASS
ok  	github.com/calcosmic/Aether/cmd	0.798s
```

All pre-existing `current_phase` guard tests (`TestStateMutateCurrentPhaseSucceedsWithMatchingGuard`, `TestStateMutateExpressionCurrentPhaseSucceedsWithMatchingGuard`, and the wrong-guard-type/mismatched-target refusal tests) also re-ran green after the fix -- the exact-case success path is unaffected.

### Gap 1 -- colony-prime never reads the midden

**RED** (`go test ./cmd/ -run TestOneMiddenEntryReachesColonyPrimeCapsule -v`, before the fix):

```
=== RUN   TestOneMiddenEntryReachesColonyPrimeCapsule
    midden_unification_test.go:190: [colony-prime regressed] resolveCodexWorkerContext() capsule does not carry the entry written through appendMiddenEntry -- Gap 1 reopened:
        ## Colony State

        Goal: colony-prime midden capsule test
        State: EXECUTING
        Phase: 1
        Phase Name: Testing
        Parallel Mode: in-repo
        ## Review Depth

        Heavy review -- full quality gauntlet
--- FAIL: TestOneMiddenEntryReachesColonyPrimeCapsule (0.00s)
FAIL
```

**GREEN** (same command, after the fix):

```
=== RUN   TestOneMiddenEntryReachesColonyPrimeCapsule
--- PASS: TestOneMiddenEntryReachesColonyPrimeCapsule (0.00s)
=== RUN   TestMiddenCapsuleSectionOmittedWhenNoFailures
--- PASS: TestMiddenCapsuleSectionOmittedWhenNoFailures (0.00s)
=== RUN   TestMiddenCapsuleSectionOmitsAcknowledgedEntries
--- PASS: TestMiddenCapsuleSectionOmitsAcknowledgedEntries (0.00s)
=== RUN   TestOneMiddenEntryReachesAllFourConsumers
--- PASS: TestOneMiddenEntryReachesAllFourConsumers (0.01s)
PASS
```

`TestColonyPrimeAAC005Audit` and `TestColonyPrimeSectionsPresent`, after their fixture fix, also confirmed the section renders with real content:

```
colony_prime_audit_test.go:255:   - midden (Recent Failures): 58 chars
```

### Full-suite verification (foreground, after both fixes)

```
$ go build ./...
(clean, exit 0)
$ go vet ./...
(clean, exit 0)
$ go test ./cmd/ -count=1
ok  	github.com/calcosmic/Aether/cmd	427.292s
```

Zero `--- FAIL` lines, zero `FAIL` package results, across the entire `cmd` package (5000+ tests).

## Known Stubs

None.

## Threat Flags

None. Both fixes tighten an existing check (case-sensitivity of a destructive-field guard) or add read-only context assembly (midden section, sourced from data the colony already owns) -- neither introduces a new network endpoint, auth path, file-access pattern, or schema change at a trust boundary.

## Next Phase Readiness

Both of 188-VERIFICATION.md's gaps are closed with fail-then-pass proof and a clean full-suite run. This plan does not update STATE.md, ROADMAP.md, or REQUIREMENTS.md -- the orchestrator owns those per this plan's explicit instruction. Phase 188 should be eligible for re-verification against all 6 original success criteria.

---
*Phase: 188-one-truth-for-failures-and-advances*
*Plan: 07*
*Completed: 2026-08-21*

## Self-Check: PASSED

- FOUND: `.planning/phases/188-one-truth-for-failures-and-advances/188-07-SUMMARY.md`
- FOUND: `cmd/state_cmds.go`
- FOUND: `cmd/colony_prime_context.go`
- FOUND: commit `dd3e584c` in git history
- FOUND: commit `00eb11ce` in git history
