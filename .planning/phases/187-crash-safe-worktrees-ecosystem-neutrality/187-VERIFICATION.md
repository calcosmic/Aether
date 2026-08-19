---
phase: 187-crash-safe-worktrees-ecosystem-neutrality
verified: 2026-08-19T00:00:00Z
status: passed
score: 11/11 must-haves verified
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 8/11
  gaps_closed:
    - "cmd/recover_repair.go's orphan-branch delete now calls branchMergeSafety before `git branch -D`; unsafe branches are preserved and reported, not deleted (GAP-4)"
    - "cmd/entomb_cmd.go / cmd/abandon_cmd.go's shared clearActiveColonyRuntimeFiles now scans the worktrees directory via scanUnrecordedWorktrees before RemoveAll, and leaves unsafe entries in place; abandon's confirmation preview now states worker_workspaces_with_work in plain language before --confirm (GAP-5)"
    - "The AST reachability guard was extended from a 7-command hand-maintained root set to every command actually registered on rootCmd (TestNoRegisteredCommandDestroysWorktreeWithoutSafetyGate), closing the review-sweep cycle that found new gaps on every pass"
  gaps_remaining: []
  regressions: []
---

# Phase 187: Crash-Safe Worktrees & Ecosystem Neutrality Verification Report

**Phase Goal:** No Aether command ever deletes unmerged or dirty work; worktree merge works outside Go repos.
**Verified:** 2026-08-19
**Status:** passed
**Re-verification:** Yes — third full pass, after plan 187-08 closed GAP-4 and GAP-5 and extended the guard

## Goal Achievement

### GAP-4 and GAP-5: Verified Fixed in Source (not from SUMMARY)

| Gap | Claim | Verified by reading | Verdict |
|-----|-------|---------------------|---------|
| GAP-4 | `repairDirtyWorktree`'s orphan-branch case gates `git branch -D` on `branchMergeSafety` | `cmd/recover_repair.go:655-677` — `safety := branchMergeSafety(root, branchName)`; `if !safety.Safe` returns early via `reportWorktreePreservation` + `record.Action = "skip_delete_unmerged_orphan_branch"`; the delete (`exec.Command("git", "-C", root, "branch", "-D", branchName)`) only runs after the `if` block, on the safe path. `branchMergeSafety` (`cmd/worktree_safety.go:134-191`) runs a real `git rev-list --count <integration>..<branch> --` and resolves `Safe=false` on ANY error (fail-closed) or `unmergedCount > 0`. Not a stub — it is a genuine mergedness check, delegated to from `worktreeDestructionSafety` itself so there is one implementation. | ✓ VERIFIED |
| GAP-5 | `clearActiveColonyRuntimeFiles` scans before wiping `.aether/worktrees`, both callers gated | `cmd/entomb_cmd.go:695-749` — `scanUnrecordedWorktrees` runs against the worktrees dir; `if len(unsafe) > 0` reports each preservation and `return nil` WITHOUT calling `os.RemoveAll(worktreesDir)`; the RemoveAll only executes after that guard clause. `abandon_cmd.go:61,131-158` — `countWorktreesHoldingWork` runs BEFORE the colony-state reset (state.Worktrees still populated), checks both tracked (`worktreeDestructionSafety`) and unrecorded (`scanUnrecordedWorktrees`) entries, and the count flows into `renderAbandonPreviewVisual`'s `Warning:` line and a stated caveat that unsafe workspaces "will be kept, not deleted, even after --confirm." | ✓ VERIFIED |

**Behavioral confirmation (not just source reading):** `go test -run TestAbandonPreservesUnrecordedWorktreeWithUncommittedWork` and `go test -run 'TestRecoverApplyPreservesUnmergedOrphanBranch|TestRecoverApplyDeletesTrulyMergedOrphanBranch'` — all PASS against the current, unmodified source (see Adversarial Probes below, where these same tests are shown to FAIL against reintroduced regressions and then confirmed to pass again after restoration).

### Adversarial Probe of the Extended Guard — THE central question this pass asked

The guard (`cmd/worktree_destruction_reachability_test.go`) was probed with real source-code injection, not read-only inspection. Each probe: patch source, run both guard tests + the relevant real-git behavioral test, record results, restore source, confirm restoration byte-identical via `diff` + `git status --short` (clean), rebuild, rerun to confirm normal state.

| # | Probe | Guard tests (narrow + extended) | Real-git behavioral test | Verdict |
|---|-------|----------------------------------|---------------------------|---------|
| 1 | Reintroduce GAP-4's exact original shape (bare `git branch -D`, no safety check) in `recover_repair.go` | Narrow: PASS (correctly excluded — human-typed). Extended: **FAILED**, correctly, naming `cmd/recover_repair.go:660` and `repairDirtyWorktree` | N/A (this is the SUMMARY's own documented proof, re-confirmed) | Guard works as designed for THIS shape |
| 2 | Reintroduce GAP-5's exact original shape (unconditional `os.RemoveAll(worktreesDir)`, strip the `scanUnrecordedWorktrees`/`len(unsafe)>0` gate entirely) in `entomb_cmd.go`'s `clearActiveColonyRuntimeFiles` | **BOTH PASS** — neither guard flags anything | `TestAbandonPreservesUnrecordedWorktreeWithUncommittedWork` **FAILS**, correctly (`...but it is gone: stat .../mid-build-notes.txt: no such file or directory`) | **EVASION FOUND: the guard's AST detector has no `os.RemoveAll` pattern at all.** `recordCallsInExpr` only flags `removeGitWorktree(...)` calls and literal `git worktree remove`/`git branch -D` args passed to `exec.Command`/`exec.CommandContext`. A destructive `os.RemoveAll` on a worktree directory — the exact mechanism `init`, `entomb`, and `abandon` all use — is invisible to both guard tests, structurally, not merely unreachable or sanctioned. The real-git test is the only thing that would catch this regression. |
| 3 | Satisfy the guard's `.Safe`-selector check textually without real gating: `_ = branchMergeSafety(root, branchName)` (result discarded), then `fakeSafety := struct{ Safe bool }{Safe: true}; if !fakeSafety.Safe { ...preserve... }` — condition is always false, delete always runs | **BOTH PASS** — neither guard flags anything | `TestRecoverApplyPreservesUnmergedOrphanBranch` **FAILS**, correctly (`expected branch ... to still exist ... but "git branch --list" returned ""`) | **EVASION FOUND: `condReferencesSafe` is purely syntactic** (`ast.Inspect` for any `SelectorExpr` ending in `.Safe`) — it has no data-flow analysis to confirm the `.Safe` field traces back to a real `worktreeDestructionSafety`/`branchMergeSafety` call result. Any locally-declared struct with a field literally named `Safe`, hardcoded true, satisfies the guard's gating check. The real-git test is again the only thing that catches this. |
| 4 | Structural (read-only, not injected): `cmd/host_cmd.go` registers 9 subcommands via inline `hostCmd.AddCommand(&cobra.Command{...})` — an anonymous composite literal, not a `var X = &cobra.Command{...}` declaration and not a bare identifier argument | N/A — static read | N/A | **Real gap in root-set derivation, currently inert.** `indexCobraCommandLiteral` (Step 1) is only invoked from the `var`-declaration path in `indexFile`, so an inline literal is never indexed as an entry node at all. `indexRegistrationCalls` (Step 1.5) only records `*ast.Ident` arguments to `AddCommand`, so this inline literal is invisible to `registeredIdents` too. Currently harmless: every one of these 9 subcommands (`colonize`, `lifecycle`, `plan`, `build`, `continue`, `seal`, `oracle`, `watch`, `swarm` under `aether host`) delegates entirely to `makeHostSubcommand`, which shells out to a Node subprocess — no Go-level worktree logic lives there for the guard to miss today. But if a future Go-level destructive helper were added inline this way, in this file or a new one using the same idiom, it would not be picked up as an entry point candidate by either guard test — a distinct blind spot from #2 and #3 above (this one is about which commands get scanned at all, not what counts as a destruction site or a gate). |

All four production-file injections (#1–#3) were restored and confirmed byte-identical to the pre-injection source via `diff` (not `git diff`, to avoid conflating with any legitimate change) and `git status --short` showing a clean tree. `go build ./cmd/...` and the full guard/behavioral test set were re-run after each restoration and pass normally.

### Verdict on the guard's soundness

**The guard is not sound as a sole gate, but the safety net it sits inside of is.** Two concrete, reproduced evasions exist (`os.RemoveAll` invisible to detection; syntactic-only `.Safe` matching with no data-flow tracing) plus one structural blind spot in root-set derivation (inline `AddCommand` literals, currently inert). None of these are hypothetical — each was demonstrated by actually breaking production code, in memory, in this session, and confirming the guard passes while the real-git behavioral test correctly fails and catches the exact same regression. This repo's Definition of Done language — "a command exists that someone can run, and that command fails when the requirement is unmet" — is satisfied here by the **combination**: `cmd/worktree_operator_destruction_test.go`'s real-git fail-then-pass tests are the actual backstop for GAP-4 and GAP-5's specific mechanisms (`git branch -D`, `os.RemoveAll` on the worktrees directory), and they do fail correctly on both reintroduced regressions. The AST guard adds a second, faster, broader-surfaced check for the `removeGitWorktree`/`git worktree remove`/`git branch -D`-via-exec pattern specifically, which is real and valuable, but its own doc comments overstate its coverage: it does not defend against every destructive git operation an author might reach for (`os.RemoveAll`, `git clean -fd`, `git stash drop`, `git checkout -f`, `git worktree remove --force` invoked via a different shape than the ones it string-matches), and it cannot distinguish a genuine safety verdict from a same-named local variable.

This does not reopen GAP-4 or GAP-5 — both are genuinely fixed today, confirmed by direct reading and by real-git tests passing against the actual, unmodified source. What it means is: **the review-sweep cycle is not fully closed by this guard alone.** A sixth gap of the `os.RemoveAll`-shaped or fake-`.Safe`-shaped kind, in a file nobody has opened yet, would not be caught by either guard test — it would only be caught if a human (or a future review sweep) also wrote a real-git fail-then-pass test for that specific new call site, the same manual work this repo has now done three times. The guard narrows where a human needs to look; it does not yet make that manual step unnecessary.

**This is reported as a WARNING, not a BLOCKER**, because: (a) no currently-shipped code exploits either evasion — every real destructive call site found via a fresh full-file sweep of `cmd/` this pass (see below) is genuinely gated in production; (b) the real-git behavioral tests that exist today do catch regressions in the two mechanisms GAP-4/GAP-5 fixed; (c) the phase's literal goal ("no Aether command ever deletes unmerged or dirty work") is a claim about the current codebase's behavior, which holds, not a claim about the guard's own completeness as a detector.

### Full Fresh Sweep of cmd/ for Unguarded Destruction (this pass, independent of the guard)

Searched for every destruction pattern the task named, across the whole `cmd/` package, and traced each hit to its enclosing safety check by hand:

| Pattern | Sites found | Gated? |
|---------|-------------|--------|
| `removeGitWorktree(...)` calls | `worktree_reap.go` (`if safety.Safe`), `codex_build_worktree.go` `gcOrphanedWorktrees`/`cleanupBuildWorktrees` (both `if !safety.Safe`-style), `clash.go` `worktreeCleanupCmd`, `worktree.go` `worktreeMergeBackCmd` | All gated |
| `git worktree remove` / `git branch -D`/`-d` via `exec.Command` | Same sites as above, plus `recover_repair.go`'s orphan-branch case | All gated (branch-only case uses `branchMergeSafety`) |
| `os.RemoveAll` on a worktree-shaped path | `entomb_cmd.go:746` / `init_cmd.go:262` (`.aether/worktrees`, both behind `len(unsafe)==0`/`wtPreserved==0` guard clauses), `codex_build_worktree.go:721` (`allocateBuildWorktree`'s own pre-allocation cleanup of a path that does not yet hold worker output — same shape the guard already sanctions for this function's registered-worktree rollback) | All gated or provably pre-creation |
| `git stash drop` | None found | N/A |
| `git clean -fd` | None found | N/A |
| force-checkout (`git checkout -f`/`--force`) | None found | N/A |
| `os.Rename` of a worktree path | None found | N/A |

No new live gap found. GAP-4 and GAP-5 were the last two, and both are closed.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `cmd/worktree_safety.go` (`branchMergeSafety`) | Real mergedness check for a bare branch, fail-closed | ✓ VERIFIED | Read directly; genuine `git rev-list --count` check, `Safe=false` on any error or `unmergedCount>0`. |
| `cmd/recover_repair.go` (`repairDirtyWorktree`) | Orphan-branch delete gated on `branchMergeSafety` | ✓ VERIFIED | Read directly; confirmed by `TestRecoverApplyPreservesUnmergedOrphanBranch` / `TestRecoverApplyDeletesTrulyMergedOrphanBranch` passing against unmodified source. |
| `cmd/entomb_cmd.go` / `cmd/abandon_cmd.go` (`clearActiveColonyRuntimeFiles`, `countWorktreesHoldingWork`) | Worktrees-directory wipe gated; abandon's preview warns | ✓ VERIFIED | Read directly; confirmed by `TestAbandonPreservesUnrecordedWorktreeWithUncommittedWork` / `TestAbandonPreviewMentionsWorkerWorkspacesWithWork` passing against unmodified source. |
| `cmd/worktree_destruction_reachability_test.go` (`TestNoRegisteredCommandDestroysWorktreeWithoutSafetyGate`) | Property guard over every registered command | ⚠️ ORPHANED-BLIND-SPOT (sound for the pattern it detects; blind to `os.RemoveAll` and to textual-only `.Safe` matches; blind to inline `AddCommand` literals) | Confirmed by adversarial injection, three separate probes, all restored cleanly afterward. |
| `cmd/worktree_operator_destruction_test.go` | Real-git fail-then-pass tests, the actual backstop | ✓ VERIFIED | All named tests pass against unmodified source; confirmed to catch both reintroduced regressions the AST guard missed. |

### Requirements Coverage

No formal REQUIREMENTS.md IDs are declared for this phase; judged against ROADMAP.md success criteria, confirmed below.

### ROADMAP Success Criteria

| # | Criterion | Status | Evidence |
|---|-----------|--------|----------|
| 1 | Fail-then-pass test: kill between dispatch and finalize → resume → work still present and surfaced with a named command | ✓ VERIFIED | `go test -run TestCrashBetweenDispatchAndFinalizeSurvivesResume -v` — PASS. Output includes the plain-English preservation line: "Kept the work on branch phase-3/builder-crashed instead of deleting it ... run: aether recover". |
| 2 | Worktree-mode build completes in a non-Go fixture repo; merge gate and `worktree-merge-back` use `resolveTestCommand()`, not hard-coded `go test ./...` | ✓ VERIFIED | `resolveTestCommand()` confirmed called at `cmd/worktree.go:750` and `cmd/codex_build_worktree.go:1211`. `TestMergePhaseWorktreesUsesProjectTestCommandInNodeRepo` exists and passes, exercising a real non-Go (Node) fixture repo. |
| 3 | `mergePhaseWorktrees` gains real-path tests | ✓ VERIFIED | `cmd/merge_phase_worktrees_test.go` — 4 tests, all against real `t.TempDir()` git repos with real `exec.Command("git", ...)` assertions (worktree list, branch list, log), not mocks. |

### Anti-Patterns Found

None in the files touched by plan 187-08. No `TBD`/`FIXME`/`XXX` markers.

### Behavioral Spot-Checks / Full Suite

| Check | Command | Result |
|-------|---------|--------|
| Build | `go build ./cmd/...` | Clean |
| Vet | `go vet ./...` | Clean |
| Full suite, race detection | `go test ./... -race` | All packages PASS (cmd package: 401.4s) |
| Working tree after all adversarial probes | `git status --short` | Clean — every injected probe was restored byte-for-byte |
| GAP-4 fail-then-pass | `go test -run TestRecoverApplyPreservesUnmergedOrphanBranch\|TestRecoverApplyDeletesTrulyMergedOrphanBranch` | Both PASS |
| GAP-5 fail-then-pass | `go test -run TestAbandonPreservesUnrecordedWorktreeWithUncommittedWork\|TestAbandonPreviewMentionsWorkerWorkspacesWithWork` | Both PASS |
| Narrow guard | `go test -run TestNoLifecycleReachableFunctionDestroysWorktreeWithoutSafetyGate` | PASS |
| Extended guard | `go test -run TestNoRegisteredCommandDestroysWorktreeWithoutSafetyGate` | PASS |

### Human Verification Required

None — every finding above (including the two guard evasions) was demonstrated by direct source injection and test execution, not by inference or subjective judgment.

### Gaps Summary

No blocking gap remains. GAP-4 and GAP-5 are genuinely fixed, confirmed both by direct source reading and by real-git fail-then-pass tests passing against the current, unmodified codebase. A fresh, independent full-file sweep of `cmd/` for every destruction pattern named in this pass's instructions found no new live, unguarded destruction site — the three prior sweeps (missing `clash.go`, then missing `recover`/`abandon`/`entomb`) have not surfaced a fourth round.

The adversarial review of the extended guard found it is **not fully sound as a standalone detector**: it has no `os.RemoveAll` pattern in its AST vocabulary (proven by reintroducing GAP-5's exact regression and watching both guard tests pass while the real-git test correctly failed), and its `.Safe`-gating check is purely syntactic with no data-flow verification that the referenced field came from an actual safety verdict (proven the same way for GAP-4's mechanism). A third, currently-inert gap exists in root-set derivation for inline `AddCommand(&cobra.Command{...})` registrations. These are reported as a WARNING rather than a BLOCKER because nothing in the current codebase exploits them — every real site found this pass is genuinely gated — and because the real-git behavioral test suite (`cmd/worktree_operator_destruction_test.go`) does catch regressions in exactly the two mechanisms these evasions target. The practical implication for future work: a fourth review sweep, or a fourth GAP-N, is still possible in principle if a future change uses `os.RemoveAll`, `git stash drop`, `git clean -fd`, force-checkout, or a fake `.Safe`-named local variable in a new destructive call site — the AST guard would not flag it, and only a dedicated real-git test for that specific call site would. This is a recommendation for follow-up guard-hardening work, not a reason to withhold passing this phase, whose literal goal (no live command destroys unmerged/dirty work today) is verified true.

---

_Verified: 2026-08-19_
_Verifier: Claude (gsd-verifier)_
