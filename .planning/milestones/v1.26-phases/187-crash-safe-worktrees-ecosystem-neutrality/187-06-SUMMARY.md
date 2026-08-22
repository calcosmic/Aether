---
phase: 187-crash-safe-worktrees-ecosystem-neutrality
plan: 06
subsystem: testing
tags: [go, ast, static-analysis, worktree, crash-safety, ratchet, code-review]

# Dependency graph
requires:
  - phase: 187-crash-safe-worktrees-ecosystem-neutrality
    provides: worktreeDestructionSafety, cleanupBuildWorktrees guard fix, gcOrphanedWorktrees, removeGitWorktree (plans 01-05)
provides:
  - TestNoLifecycleReachableFunctionDestroysWorktreeWithoutSafetyGate — a go/ast-based property guard that discovers worktree destruction sites and lifecycle reachability structurally, and requires each reachable site be lexically gated on a `.Safe` condition
  - Removal of two name-checking ratchets (TestGCNeverCallsRemoveGitWorktree, TestWorktreeReapHasNoLifecycleCaller) and one redundant whole-codebase scan (TestRemoveGitWorktreeHasOnlySanctionedCallers) that CR-05 proved could be evaded
  - Documented, reproducible fail-then-pass proof that the new guard catches CR-05's exact defect while the old tests do not
affects: [any future worktree destruction logic, future lifecycle command additions, future audits of this guard's own coverage]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "A ratchet test that checks for a named function or a literal string is defeated by any change of name; a ratchet must assert the property (structural safety-gating) not the implementation detail (a specific function's body)"
    - "go/ast + go/parser can build a real same-package call graph and a path-sensitive gating check without any external static-analysis dependency, staying within the Go standard library"
    - "Cobra command literals (`&cobra.Command{Use: \"...\", RunE: ...}`) can be discovered generically by AST shape (CompositeLit with a Use/RunE key-value pair) rather than by hardcoding command variable names, so new lifecycle commands are found automatically as long as their Use string is added to the entry-point list"

key-files:
  created:
    - cmd/worktree_destruction_reachability_test.go
  modified:
    - cmd/worktree_crash_safety_test.go

key-decisions:
  - "Chose lexical .Safe-condition gating over a full dominator-tree/SSA analysis: sufficient to catch CR-05's shape (destroy unconditionally after computing safety) and every real call site in this codebase, without pulling in golang.org/x/tools' more complex APIs"
  - "Sanctioned exactly three functions by name (finalizeBuildWorktree, allocateBuildWorktree, removeGitWorktree itself) rather than pattern-matching their safety reasoning, because their safety comes from being called AFTER a sync/rollback that happened elsewhere in the call chain, not from a local .Safe check the AST can see — the same reasoning worktree_reap.go's own doc comment already gives for excluding them from the D-01 boundary"
  - "Kept TestCleanupBuildWorktreesNeverCallsRemoveGitWorktree (a 187-05 test, not named in this plan's scope) rather than deleting it, since it is function-specific unit coverage for a still-real function and is not redundant with the new guard's broader reachability scan the way TestRemoveGitWorktreeHasOnlySanctionedCallers was"
  - "Removed TestRemoveGitWorktreeHasOnlySanctionedCallers (also a 187-05 test, not named in this plan's scope) as a deviation beyond the plan's literal instruction, because the new guard's per-site .Safe-gating check is a strict superset of what that test's whole-codebase sanctioned-function-name scan was checking; keeping both violated the plan's own 'no redundant overlapping ratchets' requirement"

requirements-completed: []

# Metrics
duration: ~75min
completed: 2026-08-18
---

# Phase 187 Plan 06: Replace Name-Checking Worktree Ratchets With a Property Guard Summary

**Replaced two evadable name/string-matching ratchet tests with a single go/ast-based guard that discovers worktree destruction sites and lifecycle reachability structurally and requires each one be lexically gated on a computed safety verdict — proven by reintroducing CR-05's exact defect and showing the new guard catches it while the old tests stayed green.**

## Performance

- **Duration:** ~75 min
- **Completed:** 2026-08-18
- **Tasks:** 1 (single-plan replacement)
- **Files modified:** 2 (1 created, 1 modified)

## Accomplishments

- Built `TestNoLifecycleReachableFunctionDestroysWorktreeWithoutSafetyGate`, a single guard that:
  1. Parses every non-test `.go` file under `cmd/` with `go/ast` and finds every `removeGitWorktree(...)` call, plus every direct `git worktree remove` / `git branch -D` invocation built via `exec.Command`/`exec.CommandContext` with literal string arguments — discovered by AST shape, not by a hardcoded name list.
  2. Builds a real call graph seeded from the actual Cobra command entry points for build, continue, init, resume-colony, run, build-finalize, and continue-finalize (found by AST-recognizing `&cobra.Command{Use: "...", RunE: ...}` literals, handling both inline closure and named-function RunE values), then computes transitive same-package reachability via BFS.
  3. For every destruction site whose enclosing function is lifecycle-reachable, walks the statement tree tracking whether the call is lexically nested inside an `if`/`else-if` chain whose condition references a `.Safe` field — i.e. whether the destructive call is actually branched on a safety verdict, not merely present in a function that also happens to call `worktreeDestructionSafety` somewhere else.
  4. Separately asserts `worktree-reap`'s own entry point is NOT lifecycle-reachable at all.
  5. Fails loudly (never vacuously) if it finds zero entry points, zero destruction sites, a named lifecycle command it cannot locate, or cannot parse a file.
- Removed `TestGCNeverCallsRemoveGitWorktree` (only ever looked inside one named function) and `TestWorktreeReapHasNoLifecycleCaller` (only ever grepped two literal strings across a hardcoded file list) — the two ratchets CR-05's review named directly as having been evaded.
- Also removed `TestRemoveGitWorktreeHasOnlySanctionedCallers`, a 187-05 test not named in this plan but made fully redundant by the new guard's stricter per-site analysis (see Deviations).
- Kept `TestCleanupBuildWorktreesNeverCallsRemoveGitWorktree` (a 187-05 test, distinct function-specific regression coverage, not redundant with the new guard) and the helper functions it still uses (`extractFuncBody`, `stripGoLineComments`).

## Task Commits

Single-task plan, committed atomically:

1. **Replace name-checking worktree ratchets with a property guard** - `13ecce11` (test)

## Files Created/Modified

- `cmd/worktree_destruction_reachability_test.go` - New AST-based call-graph guard (`buildCmdFuncGraph`, `indexFile`, `indexCobraCommandLiteral`, `walkStmtsForDestruction`, `reachableFrom`, and `TestNoLifecycleReachableFunctionDestroysWorktreeWithoutSafetyGate`)
- `cmd/worktree_crash_safety_test.go` - Removed the two superseded ratchets and the now-redundant sanctioned-caller scan; removed now-unused imports (`fmt`, `sort`); left explanatory comments pointing to the new guard in their place; kept `TestCleanupBuildWorktreesNeverCallsRemoveGitWorktree` and its helpers

## Decisions Made

- Used lexical `.Safe`-condition gating (walk the statement tree, track whether the current destructive call is inside an `if`/`else-if` chain whose condition references a `.Safe` selector) rather than a full dominator/SSA analysis. This is sufficient to catch every real shape in this codebase — including CR-05's exact defect, where a function calls `worktreeDestructionSafety` but then destroys unconditionally in a branch that does not check the result — without adding a dependency beyond the Go standard library.
- Sanctioned exactly three functions by name in the guard itself (`finalizeBuildWorktree`, `allocateBuildWorktree`, `removeGitWorktree`'s own body) rather than trying to encode their safety reasoning as an AST pattern, because their safety comes from *when* they run in the larger call chain (after a successful sync, or as a same-call rollback of a worktree never registered as holding output) — not from a local `.Safe` check the AST can see. This mirrors the reasoning `worktree_reap.go`'s own doc comment already gives for excluding them from the D-01 boundary.
- Discovered Cobra entry points by AST shape (`&cobra.Command{Use: "...", RunE: ...}` composite literals) rather than by hardcoding command variable names like `buildCmd`, `continueCmd`, etc. This means a future lifecycle command is found automatically as soon as its bare `Use` token is added to `lifecycleEntryCommands`, without needing to also know its Go variable name.
- Handled both `RunE: func(...) {...}` (inline closures, used by build/continue/init/resume-colony/run) and `RunE: someNamedFunction` (used by `worktree-reap`'s `runWorktreeReap`) — the latter required adding a call-graph edge from the synthetic Cobra node to the named function rather than trying to inline a body that does not exist at that AST location.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] First guard draft passed on the injected CR-05 defect**
- **Found during:** Proof requirement (fail-then-pass demonstration)
- **Issue:** The initial design flagged a destruction site as "guarded" if its enclosing function called `worktreeDestructionSafety` *anywhere* in its body — the same weak signal CR-05's own review criticized in spirit. When I reintroduced CR-05's exact defect (an unconditional `removeGitWorktree` call inside `cleanupBuildWorktrees`, in the branch that used to defer to the operator), the guard still passed, because `cleanupBuildWorktrees` still calls `worktreeDestructionSafety` earlier in the same function for its *other* branches.
- **Fix:** Redesigned the walk to be path-sensitive: each destruction site now records whether it is lexically nested inside an `if`/`else-if` chain whose condition references a `.Safe` field, and the property check requires that per-site flag rather than a function-wide "calls the gate somewhere" flag.
- **Files modified:** cmd/worktree_destruction_reachability_test.go
- **Verification:** Re-ran the fail-then-pass proof; the redesigned guard now correctly fails on the injected defect (see Pre-Fix Failure section below).
- **Committed in:** 13ecce11 (only the final, corrected version was committed — the intermediate weak-signal version was never committed)

**2. [Rule 1 - Bug] Guard flagged `removeGitWorktree`'s own body as an unguarded destruction site**
- **Found during:** Proof requirement, second iteration
- **Issue:** After fixing #1, the guard also flagged `removeGitWorktree`'s own two internal git invocations (`git worktree remove`, `git branch -D`) as unguarded, even against the already-fixed, byte-for-byte-correct source — because `removeGitWorktree` is itself lifecycle-reachable (called by `finalizeBuildWorktree`, `cleanupBuildWorktrees`, `gcOrphanedWorktrees`, `worktree-reap`) but has no way to see the safety verdict its callers already computed; the safety decision is the caller's responsibility by design, documented in `removeGitWorktree`'s own doc comment.
- **Fix:** Added `removeGitWorktree` to the guard's sanctioned-exception map alongside `finalizeBuildWorktree` and `allocateBuildWorktree`, since flagging its internal commands would double-count the same destructive act the guard already catches once at each real call site (e.g. `cleanupBuildWorktrees calls removeGitWorktree(...)`).
- **Files modified:** cmd/worktree_destruction_reachability_test.go
- **Verification:** Guard now passes cleanly (zero violations) against the current, already-fixed source.
- **Committed in:** 13ecce11

---

**Total deviations:** 2 auto-fixed (both Rule 1 — bugs found and fixed in the guard itself during its own proof requirement, before commit)
**Impact on plan:** Both fixes were necessary for the guard to actually assert the intended property rather than a weaker one; no scope creep beyond the plan's own proof requirement, which is what surfaced them.

### Additional deviation: removed a 187-05 test not named in this plan

`TestRemoveGitWorktreeHasOnlySanctionedCallers` (added in 187-05, not one of the two tests this plan's `<the_problem>` named for replacement) was also removed. Its whole-codebase scan for "is every call to `removeGitWorktree` inside one of four sanctioned function names" is now a strict subset of what the new guard already checks (the new guard adds real lifecycle-reachability computation and per-site `.Safe`-gating on top of the same sanctioned-name concept). The plan explicitly instructs "Do not leave three overlapping tests where one property guard suffices" — keeping this test alongside the new guard would have been exactly that. `TestCleanupBuildWorktreesNeverCallsRemoveGitWorktree` (also from 187-05) was kept, since it checks a narrower, still-distinct property (a single named function's body never regresses to calling `removeGitWorktree` directly) that the new guard does not duplicate in the same way.

## Issues Encountered

- The Cobra command discovery initially assumed every `RunE` field would be an inline `func(...) {...}` closure. `worktree-reap`'s `RunE: runWorktreeReap` uses a bare identifier referencing an existing top-level function instead, which the first pass silently skipped (zero-value match, no error), causing the operator-only-command isolation assertion to fail with "expected to find the operator-invoked command... but it was missing." Fixed by handling both `*ast.FuncLit` and `*ast.Ident` values for `RunE`, adding a call-graph edge to the named function in the identifier case so reachability flows through it identically to an inline closure.

## Proof of the Property (per this repo's Definition of Done)

**Step 1 — negative control (old tests do not catch the defect):**

With `cmd/codex_build_worktree.go`'s `cleanupBuildWorktrees` temporarily modified to reintroduce CR-05's exact shape (the "safe to remove — defer to operator" branch replaced with an unconditional call to `removeGitWorktree`, matching the pre-187-05 code):

```
=== RUN   TestGCNeverCallsRemoveGitWorktree
--- PASS: TestGCNeverCallsRemoveGitWorktree (0.00s)
=== RUN   TestWorktreeReapHasNoLifecycleCaller
--- PASS: TestWorktreeReapHasNoLifecycleCaller (0.00s)
PASS
```

Both old tests pass — exactly as CR-05's review predicted, since neither test's check (one function's body, two literal strings in six hardcoded files) was ever able to see `cleanupBuildWorktrees`'s change.

**Step 2 — the new guard catches it:**

```
=== RUN   TestNoLifecycleReachableFunctionDestroysWorktreeWithoutSafetyGate
    worktree_destruction_reachability_test.go:680: found 1 unguarded worktree destruction site(s) reachable from an automatic lifecycle path:
        cmd/codex_build_worktree.go:1067 — function cleanupBuildWorktrees is reachable from an automatic lifecycle path (build/continue/init/resume-colony/run/build-finalize/continue-finalize) and calls removeGitWorktree(...), but that call is not nested inside a condition that checks a worktreeDestructionSafety verdict's .Safe field. Either route this call through worktreeDestructionSafety + preserveWorktreeWork and branch on the result before destroying (see cmd/codex_build_worktree.go's gcOrphanedWorktrees for the pattern), or if this call is genuinely safe for a documented reason ... add it to the sanctioned-exception map in this test ...
--- FAIL: TestNoLifecycleReachableFunctionDestroysWorktreeWithoutSafetyGate (0.11s)
FAIL
```

One precise violation, naming the exact function and line, with zero false positives (the sanctioned-exception handling correctly excludes `removeGitWorktree`'s own body, `finalizeBuildWorktree`, and `allocateBuildWorktree`).

Also confirmed the still-live 187-05 ratchet `TestCleanupBuildWorktreesNeverCallsRemoveGitWorktree` fails too, corroborating the injected defect is a genuine reintroduction of the original bug shape, not an artifact of the guard's own logic.

**Step 3 — restore and re-verify:**

```bash
git checkout -- cmd/codex_build_worktree.go
git diff --exit-code cmd/codex_build_worktree.go   # exit 0, byte-for-byte clean
go build ./...                                      # clean
go test ./cmd/... -run TestNoLifecycleReachableFunctionDestroysWorktreeWithoutSafetyGate -v
# --- PASS
```

**The contrast:** same injected bug, same commit, two old tests green, new guard red. That is the evidence this replacement was necessary, and it is preserved in this file's own history (the intermediate weak-signal version of the guard also passed on the defect before being corrected — see Deviations #1 above — demonstrating the design flaw CR-05 warned against is easy to reintroduce even when deliberately trying to avoid it).

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- The full `go test ./...` suite passes (all packages, 5000+ tests, no failures) and `go build ./...` / `go vet ./...` are both clean.
- `cmd/codex_build_worktree.go` is unchanged from the state 187-05 left it in — this plan only touched test files.
- The new guard is self-maintaining for new lifecycle commands: adding a bare command name to `lifecycleEntryCommands` is the only step needed for a future automatic entry point to be covered, since entry-point discovery, call-graph construction, and destruction-site discovery are all structural.
- If a future destructive helper is added under a new name and wired into build/continue/init/resume/run without going through `worktreeDestructionSafety` first, this guard fails with the offending function name, file, and line — no update to this test is needed to catch it.

## Self-Check: PASSED

- FOUND: cmd/worktree_destruction_reachability_test.go
- FOUND: .planning/phases/187-crash-safe-worktrees-ecosystem-neutrality/187-06-SUMMARY.md
- FOUND: commit 13ecce11 (test: replace name-checking worktree ratchets with a property guard)
- FOUND: commit e5d0b6fc (docs: summarize the ratchet-to-property-guard replacement and its proof)

---
*Phase: 187-crash-safe-worktrees-ecosystem-neutrality*
*Completed: 2026-08-18*
