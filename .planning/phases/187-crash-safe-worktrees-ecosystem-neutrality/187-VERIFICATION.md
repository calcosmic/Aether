---
phase: 187-crash-safe-worktrees-ecosystem-neutrality
verified: 2026-08-19T00:00:00Z
status: gaps_found
score: 2/3 roadmap success criteria fully verified; goal statement itself not fully achieved
overrides_applied: 0
gaps:
  - truth: "No Aether command ever deletes unmerged or dirty work"
    status: failed
    reason: >
      The phase's own goal statement is an absolute claim about every Aether command, not just
      the lifecycle-automatic paths (resume/continue/init/build). Two independent, real, shipped
      Cobra commands still destroy a worktree's uncommitted/dirty work unconditionally, with zero
      call to worktreeDestructionSafety and zero test coverage of the dirty-worktree case:
        1. `worktree-merge-back` (cmd/worktree.go:690-877, RunE Step 5, line 839) — after its two
           merge gates (tests, clash detection) pass, it runs
           `git worktree remove <path> --force` unconditionally. Confirmed by direct execution
           against a real git worktree containing only an untracked file: `git worktree remove
           --force` deletes the directory and its untracked content silently, exit 0, no warning.
           Nothing in this command's gates checks whether the worktree itself is dirty — the test
           gate only proves the worktree's committed HEAD passes tests, not that there is no
           uncommitted work sitting alongside it. `resolveTestCommand()` wiring (the criterion-2
           fix) was applied to this exact command; the destructive step it protects was not
           touched by any of plans 01-06.
        2. `worktree-cleanup` (cmd/clash.go:151-180) — unconditionally runs
           `git worktree remove <branch> --force`, no safety guard, no dirty check, no
           preservation of any kind. Entirely outside this phase's scope of work.
      Both are structurally invisible to the new property guard
      (cmd/worktree_destruction_reachability_test.go) because their Cobra RunE functions are not
      in `lifecycleEntryCommands` — they were deliberately grouped with worktree-reap as
      "operator-only, therefore excluded." That grouping is not accurate: worktree-reap genuinely
      gates every destructive call on `worktreeDestructionSafety.Safe` internally; these two
      commands have no internal gate of any kind. Being human-invoked does not make a command
      safe to destroy dirty work in — the phase's own D-01 decision and goal wording say
      "no Aether command," not "no automatic Aether command."
    artifacts:
      - path: "cmd/worktree.go"
        issue: "worktreeMergeBackCmd's Step 5 (line 839) removes the worktree via git worktree remove --force with no dirty-file check and no call to worktreeDestructionSafety; the branch is deleted right after (line 845)."
      - path: "cmd/clash.go"
        issue: "worktreeCleanupCmd (line 151) removes a worktree via git worktree remove --force with zero safety check of any kind."
    missing:
      - "Route worktreeMergeBackCmd's Step 5 destructive cleanup through worktreeDestructionSafety before calling git worktree remove; on unsafe, preserve (per D-01) and report instead of destroying."
      - "Route worktreeCleanupCmd through the same guard, or delete/deprecate it if it is genuinely dead and unreferenced by any active surface."
      - "Add lifecycleEntryCommands (or a second, explicitly-justified root set) that reflects these two commands are reachable from a human/automation invocation, and require them to pass the same per-site .Safe-gating check the AST guard already applies to build/continue/init — or, if the design intent is truly 'operator-typed commands are exempt from D-01,' add an explicit internal guard to each so the goal's literal wording ('no Aether command') still holds."
      - "Add a real-git fail-then-pass test for worktree-merge-back with a dirty (uncommitted, unrelated-to-the-merge) file in the worktree, proving it survives Step 5 the same way TestCrashBetweenDispatchAndFinalizeSurvivesResume proves gcOrphanedWorktrees does."
  - truth: "gcOrphanedWorktrees refuses/stashes dirty or unmerged worktrees and defers destruction to recover's scanner (criterion 1, init path)"
    status: partial
    reason: >
      This holds for every worktree gcOrphanedWorktrees can see. WR-06 from 187-REVIEW.md was
      never closed: cmd/init_cmd.go:226-227 still does `os.RemoveAll(worktrees dir)` whenever
      wtPreserved == 0, and wtPreserved is derived solely from entries tracked in
      COLONY_STATE.json. A worktree directory created by `git worktree add` but not yet appended
      to state (the crash window between codex_build_worktree.go's `git worktree add` and
      `appendBuildWorktreeEntry`, exactly the class of crash this phase targets) is invisible to
      the count and is deleted by RemoveAll without any git-level check, dirty or not.
    artifacts:
      - path: "cmd/init_cmd.go"
        issue: "Lines 226-227: os.RemoveAll of the whole worktrees directory is gated only on a tracked-entry count, not on what is actually present on disk."
    missing:
      - "Before RemoveAll, list the worktrees directory and refuse (or route through worktreeDestructionSafety) if it contains anything not accounted for by the preserved count, per the review's own suggested fix (WR-06)."
deferred: []
human_verification: []
---

# Phase 187: Crash-Safe Worktrees & Ecosystem Neutrality Verification Report

**Phase Goal:** No Aether command ever deletes unmerged or dirty work; worktree merge works outside Go repos.
**Verified:** 2026-08-19
**Status:** gaps_found
**Re-verification:** No — initial verification (post-187-REVIEW.md, post-187-05/06 remediation)

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Kill between dispatch and finalize → resume → work still present, surfaced by a named command | ✓ VERIFIED | `TestCrashBetweenDispatchAndFinalizeSurvivesResume` (cmd/worktree_crash_safety_test.go:139) builds a real git worktree, commits + leaves uncommitted content, runs `gcOrphanedWorktrees()`, and asserts the branch, the committed file, and the uncommitted content (directly or via stash) all survive. Ran it directly — passes. |
| 2 | `gcOrphanedWorktrees` refuses/stashes dirty or unmerged worktrees, defers destruction to an explicit command | ✓ VERIFIED (for tracked entries) | `worktreeDestructionSafety` (cmd/worktree_safety.go) fails closed on empty branch (CR-01), wrong-repo dirty answers (CR-02), and undeterminable state; `gcOrphanedWorktrees` (cmd/codex_build_worktree.go:1119) never calls `removeGitWorktree`; destruction lives only in `worktree-reap` (cmd/worktree_reap.go), gated by `.Safe` on every branch. |
| 2b | ...but init's cleanup of the worktrees directory itself is safe | ✗ FAILED | WR-06 (187-REVIEW.md) unaddressed: `cmd/init_cmd.go:226-227` still `os.RemoveAll`s the whole worktrees directory based on a tracked-entry count, not on disk contents — a worktree created but not yet state-tracked (the exact crash window this phase targets) is silently destroyed. |
| 3 | `cleanupBuildWorktrees` (build path) never destroys unsafe work | ✓ VERIFIED | CR-05 fix confirmed by direct source read (cmd/codex_build_worktree.go:1026-1088): no call to `removeGitWorktree` anywhere in the function; unsafe entries are preserved and marked orphaned. `TestCleanupBuildWorktreesSurvivesCrashOnBuildPath` and `TestCleanupBuildWorktreesNeverCallsRemoveGitWorktree` both pass. |
| 4 | `removeGitWorktree` never deletes a branch after worktree removal has already failed | ✓ VERIFIED | CR-03 fix confirmed by direct source read (cmd/codex_build_worktree.go:829-845): `worktree remove` failure returns immediately, `branch -D` is unreachable on that path. `TestRemoveGitWorktreeDoesNotDeleteBranchWhenRemovalFails` passes. |
| 5 | Preservation never reports success without having actually saved anything | ✓ VERIFIED | CR-04 fix confirmed: `preserveWorktreeWork`'s undeterminable-state branch (cmd/worktree_safety.go:223-234) returns `preserved=false`; `worktree-reap --include-unmerged`'s consumer (cmd/worktree_reap.go:158-171) refuses on `!preservedOK`. `TestPreserveWorktreeWorkReportsFalseWhenStateUndeterminable` and `TestWorktreeReapIncludeUnmergedRefusesUndeterminableState` pass. |
| 6 | The sole destructive command (`worktree-reap`) is genuinely operator-invoked only | ✓ VERIFIED | No Go caller invokes `runWorktreeReap`; not present in any lifecycle command file, not present in any active playbook. |
| 7 | **No Aether command ever deletes unmerged or dirty work (the phase's literal goal statement)** | ✗ FAILED | `worktree-merge-back` (cmd/worktree.go:839) and `worktree-cleanup` (cmd/clash.go:164) both run `git worktree remove --force` unconditionally, with zero dirty check and zero call to `worktreeDestructionSafety`. Confirmed by direct execution: `git worktree remove --force` on a real worktree containing only an untracked file deletes it silently, exit 0. Neither command is covered by the new AST reachability guard because both are deliberately excluded from its root set (grouped with worktree-reap as "operator-only" — but unlike worktree-reap, neither has any internal safety gate). See Gaps. |
| 8 | Worktree-mode build completes in a non-Go fixture repo | ✓ VERIFIED | `TestMergePhaseWorktreesUsesProjectTestCommandInNodeRepo` (cmd/merge_phase_worktrees_test.go:109) — real git repo, real `git worktree add`, a Node fixture (`package.json` + CLAUDE.md's `npm test` line), real commit merged via `mergePhaseWorktrees(1)`. Passes. |
| 9 | `mergePhaseWorktrees` gains real-path tests | ✓ VERIFIED | Four new tests beyond the prior empty-list-only test: Node success, refusal-when-undeterminable, refusal-preserves-worktree, Go-still-works. All use real git fixtures, not mocks. All pass. |

**Score:** 7/9 truths verified. The two failures are both instances of the same underlying gap: commands outside the phase's chosen "lifecycle path" boundary that still destroy work unconditionally, contradicting the phase's own literal goal wording.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `cmd/worktree_safety.go` | Shared destruction-safety gate, fail-closed | ✓ VERIFIED | CR-01/CR-02/CR-04 fixes present, read directly, tests pass |
| `cmd/worktree_reap.go` | Sole operator-invoked destruction command | ✓ VERIFIED | Gated on `.Safe` at every destructive call site |
| `cmd/codex_build_worktree.go` (`removeGitWorktree`) | Fail-fast, no branch delete after failed removal | ✓ VERIFIED | CR-03 fix present |
| `cmd/codex_build_worktree.go` (`cleanupBuildWorktrees`) | Routed through safety guard, no direct destruction | ✓ VERIFIED | CR-05 fix present |
| `cmd/codex_build_worktree.go` (`mergePhaseWorktrees`) | Uses `resolveTestCommand()`, real-path tested | ✓ VERIFIED | Confirmed, live caller at codex_build_finalize.go:569 |
| `cmd/worktree.go` (`worktreeMergeBackCmd`) | Uses `resolveTestCommand()` (criterion 2) AND does not destroy dirty work (goal) | ⚠️ PARTIAL | Test-command wiring done; destructive Step 5 unguarded — see Gaps |
| `cmd/clash.go` (`worktreeCleanupCmd`) | Not in phase scope, but exists and destroys unconditionally | ✗ UNGUARDED | Out of scope for the phase's plans, but violates the phase's own goal statement |
| `cmd/init_cmd.go` | Worktrees directory cleanup only removes what is confirmed empty of work | ✗ FAILED (WR-06) | `os.RemoveAll` still gated only on a tracked-entry count, not disk contents |
| `cmd/worktree_destruction_reachability_test.go` | Structural property guard replacing evadable name-checking ratchets | ✓ VERIFIED with a scope caveat | Genuinely path-sensitive (`.Safe`-gating, not "calls the gate somewhere"); genuinely catches CR-05's exact shape when reintroduced (confirmed via 187-06's documented fail-then-pass proof, re-verified by reading the guard's logic). Caveat: its root set of lifecycle entry points excludes `worktree-merge-back` and `worktree-cleanup`, so it cannot see the two gaps found above — not a bug in the guard's algorithm, but a scope decision that leaves those two commands permanently outside its enforcement. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `codex_build_finalize.go:569` | `mergePhaseWorktrees` | direct call | ✓ WIRED | Confirmed live caller, build-finalize is a lifecycle entry point |
| `mergePhaseWorktrees` | `resolveTestCommand()` | direct call | ✓ WIRED | Confirmed, resolved once before the loop |
| `session_flow_cmds.go` (resume) | `gcOrphanedWorktrees` | direct call | ✓ WIRED | Confirmed by grep, matches 187-CONTEXT.md's documented caller list |
| `codex_continue.go` | `gcOrphanedWorktrees` | direct call | ✓ WIRED | Confirmed |
| `init_cmd.go` | `gcOrphanedWorktrees` | direct call | ✓ WIRED | Confirmed, but followed by the unguarded `RemoveAll` (WR-06) |
| `codex_build.go:1758` | `cleanupBuildWorktrees` | direct call | ✓ WIRED | Confirmed, error now surfaced (was previously swallowed) |
| `worktree.go` (`worktree-merge-back`) | `worktreeDestructionSafety` | **none** | ✗ NOT WIRED | No import, no call anywhere in the file — confirmed by grep and full read of the RunE body |
| `clash.go` (`worktree-cleanup`) | `worktreeDestructionSafety` | **none** | ✗ NOT WIRED | Confirmed by full read |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| `git worktree remove --force` silently destroys untracked/uncommitted work | Real git repo + worktree + untracked file + `git worktree remove --force` | Directory and file gone, exit 0, no error | ✓ CONFIRMED (this is the mechanism `worktree-merge-back` and `worktree-cleanup` invoke unguarded) |
| `go build ./...` | `go build ./...` | Clean | ✓ PASS |
| Full worktree-related test suite | `go test ./cmd/... -run "...worktree...safety...merge..."` (37 tests) | All pass | ✓ PASS |

### Probe Execution

No `scripts/*/tests/probe-*.sh` files exist for this phase; the phase's own "fail-then-pass" evidence is embedded in Go tests, which were run directly above rather than via a shell probe script. N/A.

### Requirements Coverage

No formal REQUIREMENTS.md IDs are declared in any of this phase's PLAN frontmatter (`requirements-completed: []` in both 187-05 and 187-06 SUMMARYs); phase is judged solely against ROADMAP.md success criteria, covered above.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `cmd/worktree.go` | 839 | Unconditional `git worktree remove --force` with no safety gate | 🛑 BLOCKER | Directly contradicts the phase's literal goal statement for a command the roadmap's own criterion 2 names |
| `cmd/clash.go` | 164 | Unconditional `git worktree remove --force` with no safety gate | ⚠️ WARNING | Out of the phase's stated plan scope, but a live, shippable command that violates the same goal |
| `cmd/init_cmd.go` | 226-227 | `os.RemoveAll` gated on a count, not on disk contents (WR-06) | ⚠️ WARNING | Narrow crash window, but the exact class of crash this phase exists to protect against |
| `cmd/worktree_destruction_reachability_test.go` | 531-539 | `lifecycleEntryCommands` deliberately excludes `worktree-merge-back` and `worktree-cleanup`, grouping them with the genuinely-internally-gated `worktree-reap` | ⚠️ WARNING (guard scope gap, not a guard bug) | A future reader trusts this guard as proof of the D-01 property; it is not proof against these two commands |

No `TBD`/`FIXME`/`XXX` markers found in phase-modified files.

### Human Verification Required

None — all findings above were confirmed by direct code reading and direct execution (real git commands against a real worktree), not requiring subjective/visual judgment.

### Gaps Summary

The phase made genuine, verified progress: all five defects from 187-REVIEW.md (CR-01 through CR-05) are correctly fixed and each is now covered by a real-git fail-then-pass test, confirmed by reading the fix and running the tests directly rather than trusting the SUMMARYs. The replacement of the two evadable name-checking ratchets with a structural, path-sensitive AST guard (187-06) is a real improvement — its `.Safe`-gating logic is genuinely path-sensitive (verified by reading `recordCallsInExpr`/`walkStmtsForDestruction`), not merely "calls the gate somewhere in the function."

However, the phase's own goal statement is unqualified: **"No Aether command ever deletes unmerged or dirty work."** Two live, shippable Cobra commands still violate this exactly as stated:

1. `worktree-merge-back` (`cmd/worktree.go`) — the very command ROADMAP.md's criterion 2 names as needing `resolveTestCommand()` wiring. That wiring was done. Its destructive cleanup step, three lines below the merge, was not brought under `worktreeDestructionSafety` at any point across six plans and one remediation round. Confirmed by direct execution that `git worktree remove --force` silently discards uncommitted work, and by reading the full command body that no dirty check of any kind precedes it.
2. `worktree-cleanup` (`cmd/clash.go`) — entirely outside any of this phase's plans, but a real, currently-shipped command with the identical unguarded destructive call.

A third, narrower gap (WR-06, `cmd/init_cmd.go`) remains from the original review and was not part of 187-05's five-defect scope: `init`'s directory-level cleanup can still destroy an un-state-tracked worktree created during the exact crash window this phase targets.

The new property guard (187-06) is real and does what it claims for the paths it was built to cover (build/continue/init/resume/run/build-finalize/continue-finalize). It cannot be relied on as proof of the phase's absolute goal, because `worktree-merge-back` and `worktree-cleanup` were deliberately placed outside its root set on the reasoning that "operator-invoked" implies safety-equivalent to `worktree-reap` — which is false: `worktree-reap` is internally gated on every path, these two are not gated at all.

**This may be intentional scope-narrowing** (the phase's plans and CONTEXT.md consistently frame the D-01 boundary as "lifecycle paths: resume, continue, init, build" and treat `worktree-merge-back`/`worktree-cleanup` as human-typed commands outside that boundary). If so, the fix is either to narrow the phase's own goal wording to match ("no *automatic* Aether command"), or to accept the gap explicitly. As written, the ROADMAP goal and this repo's Definition of Done ("a requirement is satisfied only when a command exists that fails when the requirement is unmet") are not met for these two commands — no such failing command exists for them yet.

**This looks intentional in framing but not in execution.** To accept this deviation, add to VERIFICATION.md frontmatter:

```yaml
overrides:
  - must_have: "No Aether command ever deletes unmerged or dirty work"
    reason: "D-01's boundary is scoped to automatic lifecycle paths only; worktree-merge-back and worktree-cleanup are human-typed commands treated the same as worktree-reap, and their destructive behavior is accepted as an operator's explicit choice."
    accepted_by: "<owner>"
    accepted_at: "<ISO timestamp>"
```

Absent that explicit acceptance, this is a gap requiring a closure plan.

---

_Verified: 2026-08-19_
_Verifier: Claude (gsd-verifier)_
