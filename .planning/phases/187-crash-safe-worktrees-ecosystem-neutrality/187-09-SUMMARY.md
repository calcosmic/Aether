---
phase: 187-crash-safe-worktrees-ecosystem-neutrality
plan: 09
subsystem: worktree-safety-guard
tags: [go, ast, static-analysis, crash-safety, verification-remediation, checking-tool]

# Dependency graph
requires:
  - phase: 187-crash-safe-worktrees-ecosystem-neutrality
    provides: worktreeDestructionSafety, branchMergeSafety, scanUnrecordedWorktrees, TestNoRegisteredCommandDestroysWorktreeWithoutSafetyGate (plans 01-08)
provides:
  - os.RemoveAll/os.Rename worktree-path destruction detection (isWorktreePathDestruction) in the AST reachability guard
  - reaching-definition check for the `.Safe` gating condition (findApprovedSafetyVerdictIdents, condReferencesSafe) so a same-named local with a hardcoded field no longer satisfies the guard
  - a second named gating idiom, "collect unsafe verdicts from a range loop then gate on the count" (findApprovedUnsafeAggregateIdents), recognized without a blanket function-level exception
affects: [cmd/worktree_destruction_reachability_test.go, cmd/init_cmd.go (recognized, unchanged), cmd/entomb_cmd.go (recognized, unchanged), cmd/abandon_cmd.go (recognized, unchanged), cmd/recover_repair.go (recognized, unchanged)]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "A destruction detector that only recognizes ONE mechanism (git commands via exec.Command) is blind to any other mechanism achieving the same effect (os.RemoveAll straight off disk, bypassing git entirely) -- the fix is naming the destructive OPERATION's target (a worktree-bearing path), not the tool used to reach it."
    - "A `.Safe`-gating check that matches any SelectorExpr named Safe is defeated by a same-named local variable with a hardcoded field. The fix is a reaching-definition check: the selector's base identifier must trace back, within the enclosing function, to an actual call to an approved verdict-producing function -- not full dataflow analysis, just 'was this specific name ever assigned from a real producer call anywhere in this function.'"
    - "When a stricter detector produces a false positive on a genuinely-safe site, the first response should be extending the detector to recognize the real (different but equally sound) gating idiom in use -- not reaching for a blanket function-name exception. A whole-function exception silences ANY future ungated destruction added to that function, not just the one already-safe call site; it is a strictly weaker guarantee than teaching the detector the pattern."
    - "'Prefer false positives over false negatives' does not mean 'accept every false positive via a written exception' -- it means the detector should err toward flagging, and a human should then judge whether to extend detection (preferred, keeps the guard sound) or add a scoped exception (only when the pattern genuinely cannot be named generically)."

key-files:
  created: []
  modified:
    - cmd/worktree_destruction_reachability_test.go

key-decisions:
  - "BS-1 fix (os.RemoveAll/os.Rename detection): matched on the PATH EXPRESSION naming a worktree-bearing location (filepath.Join(...) containing the literal \"worktrees\", or a bare worktreesDir/worktreeBaseDir identifier) rather than trying to resolve the string a variable was built from generically -- a conservative, legible heuristic per the task's own guidance, documented plainly in isWorktreePathDestruction's own comment as to what it does and does not catch."
  - "BS-2 fix (fake-verdict detection): chose a reaching-definition check over full SSA-style dataflow analysis -- findApprovedSafetyVerdictIdents does one whole-body pass per function collecting every identifier ever assigned from worktreeDestructionSafety/branchMergeSafety, and condReferencesSafe only counts a `.Safe` selector as gating when its base identifier is in that set. Explicitly does not track reassignment, shadowing, or indirection through a second variable -- not needed by any real call site in this codebase."
  - "When the BS-1 fix's stricter os.RemoveAll detector produced two false positives (cmd/init_cmd.go:262's `if wtPreserved == 0`, cmd/entomb_cmd.go:746's `if len(unsafe) > 0 { ...; return nil }`), the first attempt (adding both enclosing functions to the sanctioned-exception maps) was reverted after the BS-1 fail-then-pass proof showed it silenced the exact regression being tested for -- clearActiveColonyRuntimeFiles has only ONE destruction site, so exempting the function by name is indistinguishable from exempting that one call forever, gate or no gate. Replaced with a second named detection rule (findApprovedUnsafeAggregateIdents + aggregateComparisonReferencesIdent) that recognizes the real \"collect-then-count\" gating idiom directly, including a one-hop propagation pass for cmd/init_cmd.go's `wtPreserved += len(unrecordedUnsafe)` shape. Zero new sanctioned exceptions were needed once the detector understood the actual pattern."
  - "Bundled the two kinds of trusted identifier (direct verdict idents, aggregate-count idents) into one safetyGateContext struct threaded through the walker instead of adding a second bare map[string]bool parameter everywhere -- keeps the walker's signature stable if a third gating idiom needs recognizing later."

requirements-completed: []

# Metrics
duration: ~95min
completed: 2026-08-19
---

# Phase 187 Plan 09: Close Two Confirmed Blind Spots in the Worktree-Destruction Guard Summary

**Extended the AST property guard (`cmd/worktree_destruction_reachability_test.go`) to detect `os.RemoveAll`/`os.Rename` against worktree-bearing paths and to require the `.Safe` gating condition to trace back to a real safety-verdict call -- both fixes proven fail-then-pass by reintroducing the exact bug shapes the verifier found, watching the old guard pass them, then watching the fixed guard fail them by name. No production destruction paths changed; this plan only strengthens the checking tool.**

## Performance

- **Duration:** ~95 min
- **Completed:** 2026-08-19
- **Files modified:** 1 (`cmd/worktree_destruction_reachability_test.go`)

## What This Closes

187-VERIFICATION.md (status: passed, 11/11) reported the phase's goal as
achieved but flagged the guard itself as **not sound as a standalone
detector**, with two reproduced evasions:

- **BS-1** — the guard's destruction detector had no `os.RemoveAll` pattern
  in its AST vocabulary at all. GAP-5 (the abandon/entomb worktrees-directory
  wipe this phase fixed) destroyed work via a bare `os.RemoveAll`, going
  around git entirely -- the guard would not have caught the very bug it was
  extended to prevent. The verifier proved this by reintroducing GAP-5's
  exact shape and watching both guard tests pass anyway.
- **BS-2** — the gating check (`condReferencesSafe`) matched any
  `SelectorExpr` ending in `.Safe`, with no verification that the selector's
  base actually came from a real safety-verdict call. A locally-declared
  struct literal named like a safety verdict, hardcoded `Safe: true`,
  satisfied the guard while gating nothing. The verifier proved this by
  injecting exactly that shape.

## Accomplishments

**BS-1 (`isWorktreePathDestruction`, `isOSPackageCall`):** Added detection
for `os.RemoveAll`/`os.Rename` calls whose first argument names a
worktree-bearing path -- either a `filepath.Join(...)` call containing the
string literal `"worktrees"` (the shape every production call site in this
codebase actually uses: `cmd/entomb_cmd.go`, `cmd/init_cmd.go`,
`cmd/abandon_cmd.go`), or a bare identifier matching the known
`worktreesDir`/`worktreeBaseDir` variable/constant names. Documented plainly
in the function's own comment what this catches (real call-site shapes) and
what it deliberately does not (paths built through an unrelated-named
intermediate variable, string concatenation instead of `filepath.Join`,
fully dynamic construction) -- a conservative heuristic, not a dataflow
tracer, per this plan's own scope.

**BS-2 (`findApprovedSafetyVerdictIdents`, `condReferencesSafe`):** Replaced
the purely-syntactic `.Safe` match with a reaching-definition check. A new
whole-function-body pass (`findApprovedSafetyVerdictIdents`) collects every
identifier that was directly assigned from a call to
`worktreeDestructionSafety` or `branchMergeSafety` anywhere in the enclosing
function. `condReferencesSafe` now only counts a `X.Safe` selector as gating
when `X` is in that set. A same-named local declared separately (the exact
injected shape: `safety := branchMergeSafety(...)` immediately followed by
`fakeSafety := struct{Safe bool}{Safe: true}`, with the `if` testing
`fakeSafety.Safe`) is correctly rejected, because `fakeSafety` itself was
never assigned from an approved producer.

**Second-order fix -- the aggregation idiom:** Applying BS-1's stricter
`os.RemoveAll` detector against the real codebase produced two false
positives: `cmd/init_cmd.go:262`'s `if wtPreserved == 0` and
`cmd/entomb_cmd.go:746`'s `if len(unsafe) > 0 { ...; return nil }` are both
genuinely gated, correctly-working preservation checks (GAP-3's and GAP-5's
own fixes) -- they just express the gate as a count built across a loop
rather than a single `.Safe` selector. The first fix attempt (blanket
function-name exceptions for `cobra:init` and `clearActiveColonyRuntimeFiles`)
was reverted after the BS-1 fail-then-pass proof showed it silenced the exact
regression under test: `clearActiveColonyRuntimeFiles` has exactly one
destruction site, so exempting the function by name is indistinguishable from
exempting that one call unconditionally, whether or not it stays gated.
Replaced with a second named detection rule
(`findApprovedUnsafeAggregateIdents`, `aggregateComparisonReferencesIdent`)
that recognizes the real "range over an approved verdict slice, collect
every `!v.Safe` entry, gate on `len(...)`/`==0`/`>0`" idiom directly,
including a one-hop propagation pass for `cmd/init_cmd.go`'s
`wtPreserved += len(unrecordedUnsafe)` shape (a count folded into a second
variable before the actual gate tests it). Zero new sanctioned exceptions
were needed in the end -- both real sites are recognized by the detector
itself.

## Proof: Fail-Then-Pass, Actually Observed

Per this repo's Definition of Done and this plan's explicit proof
requirement, each blind spot was demonstrated by reintroducing the exact bug
shape into the real production file, confirmed against the OLD guard first,
then against the FIXED guard, then the source was restored byte-for-byte and
confirmed clean via `git diff --exit-code`.

**BS-1 — reintroduced GAP-5's exact shape in `cmd/entomb_cmd.go`'s
`clearActiveColonyRuntimeFiles`** (stripped the `scanUnrecordedWorktrees`
scan and the `len(unsafe) > 0` gate entirely, left a bare
`os.RemoveAll(worktreesDir)`):

Against the OLD (pre-fix) guard:
```
$ go test ./cmd/... -run TestNoLifecycleReachableFunctionDestroysWorktreeWithoutSafetyGate -v
--- PASS
$ go test ./cmd/... -run TestNoRegisteredCommandDestroysWorktreeWithoutSafetyGate -v
--- PASS
```
Both guards passed the reintroduced bug -- confirming the blind spot exactly
as the verifier reported it. The real-git behavioral test
(`TestAbandonPreservesUnrecordedWorktreeWithUncommittedWork`) correctly
failed against the same injection:
```
worktree_operator_destruction_test.go:829: expected the worker workspace's
uncommitted file to survive `abandon --confirm`, but it is gone: stat
.../.aether/worktrees/phase-2-builder-abandoned/mid-build-notes.txt: no such
file or directory
```

Against the FIXED guard (same injected file, source otherwise unchanged):
```
$ go test ./cmd/... -run TestNoRegisteredCommandDestroysWorktreeWithoutSafetyGate -v
worktree_destruction_reachability_test.go:1657: found 1 unguarded worktree
destruction site(s) reachable from a registered command:
cmd/entomb_cmd.go:726 — function clearActiveColonyRuntimeFiles is reachable
from a registered Aether command and calls os.RemoveAll(...) targeting a
worktree-bearing path, but that call is not nested inside a condition that
checks a worktreeDestructionSafety verdict's .Safe field. [...]
--- FAIL
```
(The narrow lifecycle-only guard still correctly passes -- `entomb` is
operator-typed, outside its declared scope by design; the extended
all-commands guard is the one that must catch this, and does.)

Source restored via `cp` from a pre-injection backup; `git diff --exit-code
cmd/entomb_cmd.go` clean; `go build ./...` clean; both guards pass again.

**BS-2 — reintroduced the fake-verdict shape in `cmd/recover_repair.go`'s
`repairDirtyWorktree`** (kept the real `safety := branchMergeSafety(root,
branchName)` call but discarded its result, then gated on
`fakeSafety := struct{ Safe bool }{Safe: true}` instead):

Against the OLD (pre-fix) guard:
```
$ go test ./cmd/... -run TestNoLifecycleReachableFunctionDestroysWorktreeWithoutSafetyGate -v
--- PASS
$ go test ./cmd/... -run TestNoRegisteredCommandDestroysWorktreeWithoutSafetyGate -v
--- PASS
```
Both guards passed the fake-verdict injection. The real-git behavioral test
(`TestRecoverApplyPreservesUnmergedOrphanBranch`) correctly failed:
```
worktree_operator_destruction_test.go:621: expected branch
"phase-3/orphan-unmerged" to still exist after `recover --apply`, but `git
branch --list` returned: ""
```

Against the FIXED guard (same injected file):
```
$ go test ./cmd/... -run TestNoRegisteredCommandDestroysWorktreeWithoutSafetyGate -v
worktree_destruction_reachability_test.go:1657: found 1 unguarded worktree
destruction site(s) reachable from a registered command:
cmd/recover_repair.go:675 — function repairDirtyWorktree is reachable from a
registered Aether command and calls git branch -D/-d (direct exec), but that
call is not nested inside a condition that checks a
worktreeDestructionSafety verdict's .Safe field. [...]
--- FAIL
```

Source restored via `cp` from a pre-injection backup; `git diff --exit-code
cmd/recover_repair.go` clean; `go build ./...` clean; both guards pass
again.

**Guard against the real, unmodified codebase (final state):**
```
$ go test ./cmd/... -run 'TestNoLifecycleReachableFunctionDestroysWorktreeWithoutSafetyGate|TestNoRegisteredCommandDestroysWorktreeWithoutSafetyGate' -v
--- PASS: TestNoLifecycleReachableFunctionDestroysWorktreeWithoutSafetyGate
--- PASS: TestNoRegisteredCommandDestroysWorktreeWithoutSafetyGate
PASS
```
No new sanctioned exceptions were added -- both `cmd/init_cmd.go:262` and
`cmd/entomb_cmd.go:746`, the two sites the stricter `os.RemoveAll` detector
newly examines, are recognized as genuinely gated by the new aggregate-count
detection rule rather than exempted by name.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] First attempt at silencing the two aggregation-idiom false positives used blanket function-name exceptions, which the BS-1 proof then showed defeats the guard's own purpose**
- **Found during:** Running the BS-1 fail-then-pass proof after the first
  attempt at closing the false positives
- **Issue:** Adding `cobra:init` and `clearActiveColonyRuntimeFiles` to the
  sanctioned-exception maps made the FIXED guard pass the reintroduced
  GAP-5 regression too -- the exact defect this plan exists to make
  detectable. A whole-function exception cannot distinguish "the known-safe
  aggregation gate is present" from "the gate was deleted and only the
  destructive call remains," because it exempts the function by name, not
  the specific already-safe call site.
- **Fix:** Reverted the blanket exceptions. Implemented
  `findApprovedUnsafeAggregateIdents` and `aggregateComparisonReferencesIdent`
  to recognize the real gating idiom (range over an approved verdict slice,
  collect entries where `!v.Safe`, gate on the resulting count) directly, so
  both real sites are detected as gated on their own merits and a future
  regression to either site is still caught.
- **Files modified:** `cmd/worktree_destruction_reachability_test.go` (no
  production files affected -- caught and corrected before the final
  commit)
- **Commit:** folded into 35cde118 (the only commit this plan makes; the
  incorrect intermediate state was never committed)

### Deferred Issues

None. Both blind spots named in the task were closed within scope; no
out-of-scope defects were surfaced.

## Verification

- `go build ./...` — clean
- `go vet ./...` — clean
- `gofmt -l cmd/worktree_destruction_reachability_test.go` — clean
- `go test ./... -race` — all packages pass, including the full `cmd`
  package (390.3s)
- `TestNoLifecycleReachableFunctionDestroysWorktreeWithoutSafetyGate` and
  `TestNoRegisteredCommandDestroysWorktreeWithoutSafetyGate` — both pass
  against the real, unmodified codebase, zero new sanctioned exceptions
- Real-git behavioral tests re-run to confirm no regression:
  `TestRecoverApplyPreservesUnmergedOrphanBranch`,
  `TestRecoverApplyDeletesTrulyMergedOrphanBranch`,
  `TestAbandonPreservesUnrecordedWorktreeWithUncommittedWork`,
  `TestAbandonStillClearsWorktreesDirectoryWhenTrulyEmpty`,
  `TestAbandonPreviewMentionsWorkerWorkspacesWithWork`,
  `TestInitPreservesUnrecordedWorktreeWithUncommittedWork`,
  `TestInitStillWipesWorktreesDirectoryWhenTrulyEmpty`,
  `TestWorktreeCleanupRefusesToDestroyDirtyWorktree`,
  `TestWorktreeCleanupRemovesCleanMergedWorktree`,
  `TestWorktreeMergeBackPreservesUncommittedWorkAfterMerge` — all pass
- Both fail-then-pass proofs (BS-1, BS-2) run against isolated production
  files, restored byte-for-byte via `git diff --exit-code`, confirmed clean

## Self-Check: PASSED

- `cmd/worktree_destruction_reachability_test.go` — FOUND (modified)
- Commit `35cde118` — FOUND (`git log --oneline` confirms)
- No production files modified — confirmed via `git status --short` showing
  only the guard file, and `git diff --diff-filter=D --name-only HEAD~1
  HEAD` showing zero deletions
