---
phase: 187-crash-safe-worktrees-ecosystem-neutrality
verified: 2026-08-19T00:00:00Z
status: gaps_found
score: 8/11 truths verified (3 truths from the original 9 plus 2 new truths surfaced by a full re-sweep of cmd/ this pass)
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 7/9
  gaps_closed:
    - "worktree-merge-back's Step 5 auto-cleanup now consults worktreeDestructionSafety before any destructive git command (GAP-1)"
    - "worktree-cleanup now consults the same guard, gained a --force opt-in that preserves first, and its latent branch-name-as-path bug is fixed (GAP-2)"
    - "init's worktrees-directory wipe now scans the directory on disk (scanUnrecordedWorktrees) independently of COLONY_STATE.json before trusting wtPreserved==0 (GAP-3, WR-06)"
  gaps_remaining: []
  regressions: []
gaps:
  - truth: "No Aether command ever deletes unmerged or dirty work (the phase's literal goal statement)"
    status: failed
    reason: >
      Plan 07 closed the three gaps this verifier found last pass (worktree-merge-back,
      worktree-cleanup, init). Re-running the full enumeration from scratch across the WHOLE
      cmd/ package — not just the three named files — as instructed, found two more live,
      shipped code paths that destroy unmerged or dirty worktree work with zero call to
      worktreeDestructionSafety, in files plan 07 never touched:

      GAP-4 — cmd/recover_repair.go:652, repairDirtyWorktree's "Orphan branch" sub-case runs
      `exec.Command("git", "branch", "-D", branchName)` unconditionally, with no check of any
      kind for unmerged commits on that branch. The issue that triggers this path comes from
      reportOrphanBranches (cmd/worktree.go:487), which classifies a branch as "orphan" purely
      by regex match against the worker-branch naming pattern (`^phase-[1-9]\d*/[a-z0-9-]+$`,
      the exact pattern allocateBuildWorktree uses) plus "no worktree, not tracked in state" —
      it never once checks git log or rev-list for whether the branch's commits are on main.
      Confirmed by direct execution: created a real git repo, committed one file on main,
      branched, committed a second "unmerged work" file on the branch, checked out main, ran
      `git branch -D <branch>` — exit 0, `git log --all --oneline` shows only the first commit;
      the unmerged commit is unreachable from any ref. This is the identical failure mode
      worktreeDestructionSafety Step 4 exists to catch (UnmergedCommitCount > 0 => Safe=false),
      but this call site never invokes it. Reachable via the real, shipped `aether recover
      --apply` command — either `--force` or an interactive `y` at the confirmation prompt
      triggers it, and the prompt only shows the issue's message ("Orphan branch: X (no
      worktree, not tracked in state)"), never the branch's merge status, so consenting is not
      an informed choice. No test file in the repo references repairDirtyWorktree's orphan-branch
      deletion at all — no fail-then-pass proof exists for it, unlike GAP-1/2/3.

      GAP-5 — cmd/entomb_cmd.go and cmd/abandon_cmd.go both call
      clearActiveColonyRuntimeFiles (cmd/entomb_cmd.go:695), which unconditionally
      `os.RemoveAll`s the entire `.aether/worktrees` directory with no call to
      worktreeDestructionSafety and no per-entry check of any kind. entomb is gated behind
      `state.Milestone == "Crowned Anthill"` (seal must have already run), which is a materially
      lower-risk window since sealing implies the colony's work is expected to already be
      merged. abandon has no such gate: `aether abandon --confirm` can be run at ANY point
      during an active colony, including mid-build with worker worktrees allocated and
      possibly holding unmerged/uncommitted output, and its preview (abandonColonySummary)
      shows only goal/phase-counts/instincts/decisions — it never mentions worktrees, dirty
      state, or unmerged work, so an operator gets zero warning before this call wipes
      .aether/worktrees. No test file (cmd/abandon_cmd_test.go) references worktree
      interaction of any kind.

      Both gaps are the same underlying shape as GAP-1/2/3: a real, shippable command that
      performs worktree destruction outside the three call sites plan 07 fixed, invisible to
      the AST reachability guard (neither `recover` nor `abandon` is in lifecycleEntryCommands,
      correctly, since both are human-typed) and — unlike worktree-reap, and unlike the now-fixed
      worktree-merge-back/worktree-cleanup — with no internal safety gate of any kind protecting
      the destructive call.
    artifacts:
      - path: "cmd/recover_repair.go"
        issue: "repairDirtyWorktree's 'Orphan branch' case (line 652) runs `git branch -D` with zero unmerged-commit check; never calls worktreeDestructionSafety."
      - path: "cmd/worktree.go"
        issue: "reportOrphanBranches (line 487) classifies branches as orphaned by name pattern and absence from state/disk only -- never checks mergedness -- feeding the unguarded deletion above."
      - path: "cmd/entomb_cmd.go"
        issue: "clearActiveColonyRuntimeFiles (line 695, called from entomb_cmd.go:138) unconditionally os.RemoveAll's .aether/worktrees with no safety check. Lower risk: gated behind Milestone == Crowned Anthill."
      - path: "cmd/abandon_cmd.go"
        issue: "Calls the same clearActiveColonyRuntimeFiles with NO milestone gate and no worktree mention in its --confirm preview; reachable during an active build."
    missing:
      - "Route repairDirtyWorktree's 'Orphan branch' case through worktreeDestructionSafety (or equivalent rev-list check against main) before `git branch -D`; on unsafe, report and skip rather than delete, matching worktree-reap's contract."
      - "Fix reportOrphanBranches to check mergedness, not just name-pattern-and-absence, before labeling a branch 'orphan' in a way that flows into a repair path with a --force/confirm-only gate."
      - "Route clearActiveColonyRuntimeFiles's worktree removal through scanUnrecordedWorktrees/worktreeDestructionSafety per-entry (as init now does), or at minimum surface dirty/unmerged worktrees in abandon's preview and refuse/preserve on --confirm the same way the three GAP-1/2/3 fixes do."
      - "Add real-git fail-then-pass tests for both: an unmerged orphan branch surviving `aether recover --apply --force`, and a dirty/unmerged worktree surviving `aether abandon --confirm`."
deferred: []
human_verification: []
---

# Phase 187: Crash-Safe Worktrees & Ecosystem Neutrality Verification Report

**Phase Goal:** No Aether command ever deletes unmerged or dirty work; worktree merge works outside Go repos.
**Verified:** 2026-08-19
**Status:** gaps_found
**Re-verification:** Yes — after gap closure (plan 187-07)

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Kill between dispatch and finalize → resume → work still present, surfaced by a named command | ✓ VERIFIED | `TestCrashBetweenDispatchAndFinalizeSurvivesResume` — unchanged since last pass, re-confirmed passing. |
| 2 | `gcOrphanedWorktrees` refuses/stashes dirty or unmerged worktrees, defers destruction to an explicit command | ✓ VERIFIED | Unchanged, re-confirmed by source read. |
| 3 | `cleanupBuildWorktrees` (build path) never destroys unsafe work | ✓ VERIFIED | Unchanged, re-confirmed by source read (CR-05). |
| 4 | `removeGitWorktree` never deletes a branch after worktree removal has already failed | ✓ VERIFIED | Unchanged, re-confirmed by source read (CR-03). |
| 5 | Preservation never reports success without having actually saved anything | ✓ VERIFIED | Unchanged (CR-04). |
| 6 | `worktree-reap` (sole *previously* known named destruction command) is operator-invoked and internally gated | ✓ VERIFIED | Unchanged; gates every destructive call on `.Safe`. |
| 7 | `worktree-merge-back`'s Step 5 cleanup does not destroy dirty/unmerged work (GAP-1) | ✓ VERIFIED | Read `cmd/worktree.go:834-876` directly: `worktreeDestructionSafety` is computed and branched on before any destructive git command; unsafe path stashes/preserves, marks entry `WorktreeOrphaned`, returns `cleaned_up:false`. `TestWorktreeMergeBackPreservesUncommittedWorkAfterMerge` run directly — passes. |
| 8 | `worktree-cleanup` does not destroy dirty/unmerged work, and its path-resolution bug is fixed (GAP-2) | ✓ VERIFIED | Read `cmd/clash.go:150-266` directly: `resolveWorktreePathForBranch` resolves the real path via `git worktree list --porcelain`; `worktreeDestructionSafety` gates the removal; unsafe-without-force refuses and reports; unsafe-with-force preserves (stashes) first via `preserveWorktreeWork`, only then destroys. `TestWorktreeCleanupRefusesToDestroyDirtyWorktree` and `TestWorktreeCleanupRemovesCleanMergedWorktree` run directly — both pass. |
| 9 | `init`'s worktrees-directory wipe does not silently destroy a not-yet-state-tracked worktree (GAP-3 / WR-06) | ✓ VERIFIED | Read `cmd/init_cmd.go:216-262` and `cmd/worktree_safety.go:287-370` directly: `scanUnrecordedWorktrees` lists the worktrees directory on disk, confirms each unknown entry is genuinely the top level of its own git worktree (`--show-toplevel` identity check, guards the same footgun `worktreeDestructionSafety` Step 2.5 guards), evaluates it with `worktreeDestructionSafety`, adds unsafe entries to `wtPreserved`, and reports each before the `RemoveAll` decision. `TestInitPreservesUnrecordedWorktreeWithUncommittedWork` and `TestInitStillWipesWorktreesDirectoryWhenTrulyEmpty` run directly — both pass. |
| 10 | **No Aether command ever deletes unmerged or dirty work (the phase's literal, absolute goal statement)** | ✗ FAILED | A full re-sweep of every destruction pattern (`git worktree remove`, `git branch -D/-d`, `git stash drop`, `git clean -fd`, force-checkout, `os.RemoveAll` on any worktree-related path) across the ENTIRE `cmd/` package — not just the four files plan 07 touched — found two more live, unguarded destruction paths: `cmd/recover_repair.go`'s orphan-branch deletion (GAP-4) and `cmd/entomb_cmd.go`/`cmd/abandon_cmd.go`'s shared worktrees-directory wipe (GAP-5, `abandon` is the higher-risk half — no milestone gate). See Gaps. |
| 11 | The three remaining sanctioned exceptions from `lifecycleEntryCommands`'s exclusion list (worktree-merge-back, worktree-cleanup being newly-fixed-but-still-excluded, worktree-reap) are each genuinely justified | ✓ VERIFIED (for the guard's own stated scope) | The guard's doc comment was updated in plan 07 to state explicitly that exclusion from `lifecycleEntryCommands` is NOT a safety claim, only an "automatic vs. human-typed" boundary, and that the real proof for the two now-fixed commands lives in `cmd/worktree_operator_destruction_test.go`, not in reachability analysis. That framing is honest and now accurate for those two. It does NOT extend to `recover` or `abandon`, which are also human-typed and also correctly excluded from the guard by the same logic — but unlike `worktree-reap`/`worktree-merge-back`/`worktree-cleanup`, they have no internal gate at all, so the guard's "being excluded is not a safety claim" caveat is exactly where GAP-4/5 hide. |

**Score:** 8/11 truths verified (10 sub-truths tracking the goal's constituent claims, plus truth 11 about the guard's honesty). The two failures are new instances of the same underlying gap class as the three that were just closed: a real, shipped command destroys worktree content outside the set of call sites this phase's plans have so far enumerated, invisible to the AST guard by design (human-typed) and without any internal gate of its own.

### Deferred Items

None. Both new gaps are live, present-day defects in currently-shipped commands, not items scheduled for a later phase.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `cmd/worktree_safety.go` | Shared destruction-safety gate, fail-closed | ✓ VERIFIED | Unchanged, re-read directly. `scanUnrecordedWorktrees` addition confirmed present and correctly wired into `init_cmd.go`. |
| `cmd/worktree.go` (`worktreeMergeBackCmd`) | Does not destroy dirty work (GAP-1) | ✓ VERIFIED | Fixed, confirmed by direct read + test run. |
| `cmd/clash.go` (`worktreeCleanupCmd`) | Does not destroy dirty work, path bug fixed (GAP-2) | ✓ VERIFIED | Fixed, confirmed by direct read + test run. |
| `cmd/init_cmd.go` | Worktrees directory cleanup accounts for unrecorded worktrees (GAP-3) | ✓ VERIFIED | Fixed, confirmed by direct read + test run. |
| `cmd/recover_repair.go` (`repairDirtyWorktree`) | Orphan-branch deletion should not destroy unmerged commits | ✗ UNGUARDED (GAP-4, new) | `git branch -D` with zero merge check; no test coverage. |
| `cmd/entomb_cmd.go` / `cmd/abandon_cmd.go` (`clearActiveColonyRuntimeFiles`) | Worktrees-directory wipe should not destroy unmerged/dirty work | ✗ UNGUARDED (GAP-5, new) | `entomb` gated behind sealed milestone (lower risk); `abandon` has no gate and no warning in its preview; no test coverage for either. |
| `cmd/worktree_destruction_reachability_test.go` | Structural property guard, honest about its own scope | ✓ VERIFIED with the same scope caveat as last pass, now also covering `recover`/`abandon` | The guard's algorithm is unchanged and correct for what it claims to check (lifecycle-automatic paths). Its doc comment already states plainly that exclusion from `lifecycleEntryCommands` is not a safety claim — that caveat now also covers `recover` and `abandon`, which were always excluded (correctly, as human-typed commands) but were never brought under any internal gate the way worktree-merge-back/worktree-cleanup were in this same plan. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `worktree.go` (`worktree-merge-back`) Step 5 | `worktreeDestructionSafety` | direct call | ✓ WIRED | Confirmed by direct read, line 841. |
| `clash.go` (`worktree-cleanup`) | `worktreeDestructionSafety` | direct call | ✓ WIRED | Confirmed by direct read, line 228. |
| `init_cmd.go` | `scanUnrecordedWorktrees` → `worktreeDestructionSafety` | direct call | ✓ WIRED | Confirmed by direct read, lines 226-253. |
| `recover_repair.go` (`repairDirtyWorktree`, Orphan branch case) | `worktreeDestructionSafety` | **none** | ✗ NOT WIRED (GAP-4) | Confirmed by full read of the function; only calls `exec.Command("git", "branch", "-D", ...)` directly. |
| `entomb_cmd.go` / `abandon_cmd.go` | `clearActiveColonyRuntimeFiles` → `worktreeDestructionSafety` | **none** | ✗ NOT WIRED (GAP-5) | Confirmed by full read; the function performs a bare `os.RemoveAll` with no per-entry check. |

### Data-Flow Trace (Level 4)

| Path | Data Source | Produces Real Verdict | Status |
|------|-------------|------------------------|--------|
| `reportOrphanBranches` → `scanDirtyWorktrees` "Orphan branch" issue → `repairDirtyWorktree` | Branch name regex + absence from state/disk | No — never queries git for mergedness | ✗ HOLLOW SAFETY CHECK — the issue's own message ("no worktree, not tracked in state") is true and accurate, but a human reading "Orphan branch" reasonably infers "safe to delete", when the branch may hold live unmerged work. This is the actual root cause of GAP-4: the input to the repair function is already mis-labeled before the repair function's own missing gate compounds it. |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| `git branch -D` silently discards an unmerged commit | Real git repo: commit on main, branch, second commit on branch, checkout main, `git branch -D <branch>` | Exit 0, `git log --all --oneline` shows only the first commit — the second is unreachable from any ref | ✓ CONFIRMED (this is the exact mechanism `repairDirtyWorktree`'s Orphan-branch case invokes unguarded — the function's only intervening code between the switch-case entry and this call is the `exec.Command` construction itself, confirmed by direct read of lines 610-656) |
| `worktree-merge-back` preserves dirty content post-fix | `go test -run TestWorktreeMergeBackPreservesUncommittedWorkAfterMerge -v` | PASS | ✓ PASS |
| `worktree-cleanup` preserves dirty content post-fix | `go test -run TestWorktreeCleanupRefusesToDestroyDirtyWorktree -v` | PASS | ✓ PASS |
| `worktree-cleanup` still removes genuinely clean worktrees | `go test -run TestWorktreeCleanupRemovesCleanMergedWorktree -v` | PASS | ✓ PASS |
| `init` preserves an unrecorded dirty worktree | `go test -run TestInitPreservesUnrecordedWorktreeWithUncommittedWork -v` | PASS | ✓ PASS |
| `init` still wipes a truly empty worktrees directory | `go test -run TestInitStillWipesWorktreesDirectoryWhenTrulyEmpty -v` | PASS | ✓ PASS |
| AST reachability guard still holds for lifecycle paths | `go test -run TestNoLifecycleReachableFunctionDestroysWorktreeWithoutSafetyGate -v` | PASS | ✓ PASS |
| `go build ./...` | `go build ./...` | Clean | ✓ PASS |

### Probe Execution

No `scripts/*/tests/probe-*.sh` files exist for this phase; evidence is embedded in Go tests, run directly above rather than via a shell probe script. N/A.

### Requirements Coverage

No formal REQUIREMENTS.md IDs are declared in this phase's PLAN frontmatter (`requirements-completed: []` in 187-07's SUMMARY, matching prior plans); phase is judged solely against ROADMAP.md success criteria.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `cmd/recover_repair.go` | 652 | Unconditional `git branch -D` with no mergedness check | 🛑 BLOCKER | Directly contradicts the phase's literal goal statement; confirmed by direct execution that this permanently discards unmerged commits |
| `cmd/worktree.go` | 487-570 | `reportOrphanBranches` classifies branches as safe-to-delete-adjacent ("orphan") without ever checking mergedness | ⚠️ WARNING | Root cause feeding GAP-4; the mislabel, not just the missing gate, is the defect |
| `cmd/entomb_cmd.go` / `cmd/abandon_cmd.go` | 695, 84, 138 | `clearActiveColonyRuntimeFiles` unconditionally wipes `.aether/worktrees`; `abandon` has no milestone gate and no worktree warning in its preview | 🛑 BLOCKER (abandon) / ⚠️ WARNING (entomb, lower risk due to seal gate) | `abandon --confirm` can run mid-build with live worker worktrees present and gives zero warning before wiping them |

No `TBD`/`FIXME`/`XXX` markers found in the newly-examined files.

### Human Verification Required

None — all findings above were confirmed by direct code reading and direct execution (real git commands against a real repo), not requiring subjective/visual judgment.

### Gaps Summary

Plan 187-07 is a genuine, verified fix for exactly the three gaps the previous verification pass named. Each of GAP-1, GAP-2, and GAP-3 is confirmed fixed by direct source reading (not by trusting the SUMMARY) and by running the corresponding real-git fail-then-pass tests directly — all pass. `worktree-merge-back` and `worktree-cleanup` now call `worktreeDestructionSafety` before any destructive git command, exactly like `worktree-reap` already did. `init` now scans the worktrees directory on disk independently of `COLONY_STATE.json` before trusting its preserved-count, closing the exact crash window (worktree created, state entry not yet appended) this phase exists to protect. `go build ./...` is clean.

However, re-running the destruction-pattern sweep from scratch across the entire `cmd/` package — as this pass's instructions required, specifically because the previous sweep missed `cmd/clash.go` entirely — found two more live, shipped code paths that destroy worktree content with no use of `worktreeDestructionSafety` and no test coverage:

1. **`cmd/recover_repair.go`'s orphan-branch deletion (GAP-4).** `aether recover --apply` classifies a branch as "orphan" purely by name pattern and absence from state/disk (`reportOrphanBranches`, `cmd/worktree.go:487`) — never checking whether its commits are merged — then `repairDirtyWorktree`'s "Orphan branch" case runs `git branch -D` on it unconditionally. Confirmed by direct execution that this exact git command permanently discards an unmerged commit, exit 0, no warning. The confirmation prompt this command requires (absent `--force`) shows only the mis-labeled "orphan" message, never the branch's actual merge status, so an operator's "yes" is not informed consent to lose work.

2. **`cmd/entomb_cmd.go` and `cmd/abandon_cmd.go`'s shared `clearActiveColonyRuntimeFiles` (GAP-5).** Both call the same function, which unconditionally `os.RemoveAll`s the whole `.aether/worktrees` directory. `entomb` is gated behind the colony already being sealed, a materially lower-risk window. `abandon` has no such gate — `aether abandon --confirm` can run at any point during an active build, and its preview never mentions worktrees or unmerged work, so it can silently discard live worker output with zero warning.

Both are the same underlying failure mode as the three gaps this plan just fixed: a real, shippable Cobra command performing worktree destruction, deliberately (and correctly) outside the AST guard's `lifecycleEntryCommands` scope because it is human-typed — but, unlike `worktree-reap` and the two commands this plan fixed, with no internal safety gate protecting the destructive call at all.

**This does not look like an intentional scope decision.** Unlike the original GAP-1/2/3 framing (where a case could be made that "operator-typed = D-01-exempt" was the design intent, even if not literally what the goal says), `recover` and `abandon` were not discussed anywhere in this phase's plans, CONTEXT.md, or REVIEW.md — they were simply never looked at. `cmd/worktree_safety.go`'s own doc comment for `preserveWorktreeWork` even says its stash approach was "already used and tested at cmd/recover_repair.go's repairDirtyWorktree" for the dirty-file case, showing the phase's authors read that file and modeled part of the fix on it — but the orphan-branch sub-case three lines below in the same switch statement was never brought under the same discipline.

Given this repo's Definition of Done ("a requirement is satisfied only when a command exists that someone can run, and that command fails when the requirement is unmet"), no such failing command exists yet for either of these two paths — `cmd/abandon_cmd_test.go` has zero worktree references, and no test anywhere exercises `repairDirtyWorktree`'s orphan-branch deletion.

If this is accepted as an explicit scope boundary rather than closed, add to this file's frontmatter:

```yaml
overrides:
  - must_have: "No Aether command ever deletes unmerged or dirty work"
    reason: "D-01's boundary covers only the destruction call sites this phase's plans have enumerated (build/continue/init lifecycle paths, plus worktree-merge-back, worktree-cleanup, and worktree-reap as explicitly-scoped operator commands). recover --apply and abandon --confirm are accepted as out-of-scope for this phase; their orphan-branch and directory-wipe behavior is deferred to a follow-up."
    accepted_by: "<owner>"
    accepted_at: "<ISO timestamp>"
```

Absent that explicit acceptance, this is a gap requiring a closure plan — the same shape of fix plan 07 just successfully applied three times (call `worktreeDestructionSafety` before the destructive git command; preserve on unsafe; add a real-git fail-then-pass test), applied to two more call sites.

---

_Verified: 2026-08-19_
_Verifier: Claude (gsd-verifier)_
