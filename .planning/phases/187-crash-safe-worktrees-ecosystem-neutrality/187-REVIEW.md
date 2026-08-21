---
phase: 187-crash-safe-worktrees-ecosystem-neutrality
reviewed: 2026-08-18T00:00:00Z
depth: standard
files_reviewed: 14
files_reviewed_list:
  - cmd/worktree_safety.go
  - cmd/worktree_safety_test.go
  - cmd/worktree.go
  - cmd/worktree_merge_gate_test.go
  - cmd/worktree_reap.go
  - cmd/worktree_crash_safety_test.go
  - cmd/codex_build_worktree.go
  - cmd/codex_continue.go
  - cmd/codex_visuals.go
  - cmd/init_cmd.go
  - cmd/session_flow_cmds.go
  - cmd/session_flow_cmds_test.go
  - cmd/subcommand_reachability_ratchet_test.go
  - cmd/merge_phase_worktrees_test.go
findings:
  critical: 5
  warning: 6
  info: 3
  total: 14
status: issues_found
---

# Phase 187: Code Review Report

**Reviewed:** 2026-08-18
**Depth:** standard
**Files Reviewed:** 14
**Status:** issues_found

## Summary

The phase's headline claim — that lifecycle paths (resume, continue, init) no longer
destroy worktrees — holds for the specific route that was rewritten. `gcOrphanedWorktrees`
genuinely stopped calling `removeGitWorktree`, the ratchet test that enforces it is real
(it locates the function body, strips comments, and fails on absence rather than passing
vacuously), and the crash-safety fixtures use real `git init` + `git worktree add`
repositories that actually reach the code under test. `mergePhaseWorktrees` and
`worktreeMergeBackCmd` both refuse before any git command when the test command cannot be
resolved, and both refusals are proven by tests that assert the worktree, the branch, and
the branch's commits still exist afterwards. That is meaningfully better than what was
there before.

The safety property as stated in the phase intent, however, does not hold. Five defects
below were confirmed by executing probe tests against the real code, not by reading it:

- The guard reports **Safe=true** when `entry.Branch` is empty, because `git rev-list
  --count "main.."` is valid git that returns `0` with exit status 0 (CR-01).
- The guard's dirty check answers about the **enclosing repository**, not the entry's
  path, whenever the path is not itself a git worktree — so a stale directory full of
  untracked work reads as "clean and fully merged" in a clean repo (CR-02).
- `removeGitWorktree` runs `git branch -D` **unconditionally**, even after `git worktree
  remove` has already failed. Every caller treats its error return as "nothing happened".
  It is not: the branch is already destroyed (CR-03).
- The preservation function's fallback branch returns `preserved=true, err=nil` having
  taken **no stash at all**. `worktree-reap --force --include-unmerged` only aborts on a
  non-nil error, so it destroys a worktree whose contents were never saved (CR-04).
- `cleanupBuildWorktrees` still calls `removeGitWorktree` with **no safety guard**, and is
  still called on every build at `cmd/codex_build.go:1752`. Build is a lifecycle path. The
  ratchet test does not cover this function, and `codex_build.go` passes the test only
  because the test greps for the string `worktree-reap`, not for destruction (CR-05).

The net effect is that destruction was removed from three of the four automatic paths and
the guard protecting the fourth is defeatable in two ways. The tests are good quality but
they test the paths the implementer had in mind; none of the five confirmed defects is
covered by any test in this phase.

---

## Critical Issues

### CR-01: Empty branch name resolves to "safe to destroy"

**File:** `cmd/worktree_safety.go:96-117`
**Issue:** When `entry.Branch` is `""`, the rev-list argument becomes the string `"main.."`.
That is *valid* git — it means "commits reachable from HEAD but not main" with an empty
right side — and it returns `0` on exit status 0. The error branch never fires, `Sscanf`
succeeds, `unmergedCount` is 0, and the function falls through to `Safe = true, Reason =
"clean and fully merged"`.

Confirmed by execution against the real fixture:

```
PROBE A: Safe=true Reason="clean and fully merged"
```

An entry can reach this state through a hand-edited or migrated `COLONY_STATE.json`, a
partial write, or any code path that appends a `WorktreeEntry` before the branch is
assigned. `worktree-reap --force` then calls `removeGitWorktree(root, safety.Path, "")`,
which runs `git worktree remove <path> --force` — destroying the directory — and
`git branch -D ""`.

Note the same defect is not caught by the dirty check, because the worktree in the probe
was genuinely clean; only the unmerged-commit check could have caught it, and that is the
check being defeated.

**Fix:** Validate the branch before either git query, and fail closed:

```go
// Step 0: a nameless branch cannot be checked, and uncertainty is not permission.
if strings.TrimSpace(entry.Branch) == "" {
    result.Safe = false
    result.Reason = "this worker workspace has no branch recorded, so its work cannot be checked"
    return result
}
```

Additionally, pass the revision range after `--` protection or validate against
`agentBranchRe` / `humanBranchPrefixes` (`validateBranchName`, `cmd/worktree.go:39`) so a
branch name beginning with `-` cannot be read by git as a flag.

---

### CR-02: The dirty check answers about the wrong repository

**File:** `cmd/worktree_safety.go:68-83`
**Issue:** `git -C <absPath> status --porcelain` does not fail when `absPath` is a plain
directory — git walks *up* the directory tree and answers about the first enclosing
repository it finds. Because tracked worktrees live at `.aether/worktrees/...` inside the
colony root, a stale entry whose worktree registration is gone (crash during
`git worktree add`, manual `rm -rf .git/worktrees/...`, a restored backup) resolves to the
**root repo's** status, not that directory's.

If the root repo is clean, the guard reports the entry as clean. Confirmed by execution,
using a repo whose `.gitignore` contains `.aether/` (the realistic configuration, since
`.aether/data` is documented as local-only):

```
PROBE D: Safe=true Reason="clean and fully merged" Dirty=0
```

The directory contained an untracked `precious.txt` at the time. The guard reported
"clean and fully merged". The comment at line 66-67 states "A guard that cannot see must
refuse, never assume safe" — but the guard *can* see, it is simply seeing the wrong thing,
which is worse than not seeing at all.

The existing test `TestWorktreeSafetyRefusesWhenGitCannotAnswer`
(`cmd/worktree_safety_test.go:133-148`) appears to cover this and does not: it points at
`root + "/.aether/not-a-worktree"`, which is inside the repo, and passes only because the
untracked directory it just created makes the *root* dirty. It is asserting the right
outcome for the wrong reason, and would still pass if the guard were removed entirely in
the clean-root case.

**Fix:** Confirm the path is the top level of its own worktree before trusting any status
answer:

```go
topCtx, topCancel := context.WithTimeout(context.Background(), GitTimeout)
topOut, topErr := exec.CommandContext(topCtx, "git", "-C", absPath,
    "rev-parse", "--show-toplevel").Output()
topCancel()
if topErr != nil {
    result.Safe = false
    result.Reason = "cannot determine whether this worktree has unsaved changes"
    return result
}
resolvedTop, _ := filepath.EvalSymlinks(strings.TrimSpace(string(topOut)))
resolvedAbs, _ := filepath.EvalSymlinks(absPath)
if resolvedTop != resolvedAbs {
    result.Safe = false
    result.Reason = "this folder is no longer a separate worker workspace, so its contents cannot be checked"
    return result
}
```

Then fix `TestWorktreeSafetyRefusesWhenGitCannotAnswer` to use a clean root, so it fails
without the guard.

---

### CR-03: `removeGitWorktree` deletes the branch even after removal has already failed

**File:** `cmd/codex_build_worktree.go:819-837`
**Issue:** The three git commands run unconditionally in sequence and their errors are only
*accumulated*. When `git worktree remove` fails, `git branch -D <branch>` still runs and
still succeeds — `-D` is the force variant, so it deletes unmerged branches without
complaint. The function then returns a non-nil error.

Every caller reads that error as "the destruction did not happen":

- `cmd/worktree_reap.go:114-121` — on error, reports `"removal failed (%v); branch %s was
  left alone"` and re-appends the entry. The branch is not left alone; it is gone.
- `cmd/worktree_reap.go:154-158` — same shape in the `--include-unmerged` path.
- `cmd/codex_build_worktree.go:811-815` (`finalizeBuildWorktree`) — on error, marks the
  entry `WorktreeOrphaned` "to protect it", pointing at a branch that no longer exists.
- `cmd/codex_build_worktree.go:1026-1029` (`cleanupBuildWorktrees`) — same.

Confirmed by execution against a branch carrying a unique unmerged commit:

```
PROBE F: removeGitWorktree err = worktree cleanup failed: worktree remove: exit status 128 ...
PROBE F: branch --list after = "" (sha was 118634a2a1d65b993e34ee949891191314d11b6b)
```

and end-to-end through the reap command:

```
PROBE G: branch after reap="", entries kept=1
```

The user is told the work was kept. It was not. This is a silent data-loss path that also
produces a permanently unreconcilable state entry.

**Fix:** Make the sequence fail-fast, and never delete the branch if the worktree still
exists:

```go
func removeGitWorktree(root, absPath, branch string) error {
    ctx, cancel := context.WithTimeout(context.Background(), GitTimeout)
    defer cancel()

    if out, err := exec.CommandContext(ctx, "git", "-C", root,
        "worktree", "remove", absPath, "--force").CombinedOutput(); err != nil {
        // Do NOT continue to branch deletion — the working copy still exists,
        // so its branch is the only handle left on that work.
        return fmt.Errorf("worktree remove: %v (output: %s)", err, string(out))
    }
    if out, err := exec.CommandContext(ctx, "git", "-C", root,
        "worktree", "prune").CombinedOutput(); err != nil {
        return fmt.Errorf("worktree prune: %v (output: %s)", err, string(out))
    }
    if out, err := exec.CommandContext(ctx, "git", "-C", root,
        "branch", "-D", branch).CombinedOutput(); err != nil {
        return fmt.Errorf("branch delete: %v (output: %s)", err, string(out))
    }
    return nil
}
```

---

### CR-04: The "cannot determine" preservation branch saves nothing but reports success

**File:** `cmd/worktree_safety.go:186-191`
**Issue:** The `default` case is reached whenever `Safe == false` but neither
`DirtyFileCount` nor `UnmergedCommitCount` is set — i.e. exactly the cases where git could
not answer (`worktree_safety.go:71-75` and `:100-104`, `:113-117`). It returns
`preserved=true, detail="could not check worktree state, so branch %s was left alone",
err=nil` without executing any command.

Confirmed by execution:

```
PROBE H: preserved=true err=<nil> detail="could not check worktree state, so branch phase-1/probe-undet was left alone"
```

In `gcOrphanedWorktrees` this is harmless — that path keeps the entry regardless. But in
`worktree-reap`'s `--force --include-unmerged` path (`cmd/worktree_reap.go:144-158`) the
only thing standing between the user and destruction is `preserveErr != nil`:

```go
_, detail, preserveErr := preserveWorktreeWork(root, entry, safety)
if preserveErr != nil { /* refuse */ }
visualFprintf(stderr, "Removing worker workspace on branch %s — its changes were saved first (%s)...")
removeGitWorktree(root, safety.Path, entry.Branch)
```

So the command prints "its changes were saved first" — quoting a detail string that says
the opposite — and then destroys a worktree whose contents were never examined, let alone
stashed. The returned `preserved` boolean is discarded at the call site (`_,`), which is
what makes the mismatch invisible.

**Fix:** Return `preserved=false` for the undeterminable case, and have the reap path
refuse on `!preserved` as well as on error:

```go
default:
    // State could not be determined, so nothing could be saved. Report that
    // honestly — a caller about to destroy must not read this as "saved".
    return false, fmt.Sprintf("could not check the work on branch %s, so nothing could be saved", entry.Branch), nil
```

```go
preservedOK, detail, preserveErr := preserveWorktreeWork(root, entry, safety)
if preserveErr != nil || !preservedOK {
    reportWorktreePreservation(safety, fmt.Sprintf("could not save the work before removal; branch %s was left alone", entry.Branch))
    remaining = append(remaining, entry)
    preservedBranches = append(preservedBranches, entry.Branch)
    continue
}
```

---

### CR-05: `cleanupBuildWorktrees` still destroys worktrees from the build lifecycle path, unguarded

**File:** `cmd/codex_build_worktree.go:1002-1042`, called at `cmd/codex_build.go:1752`
**Issue:** The phase intent states that no code path reachable from automatic lifecycle
operations may destroy a worktree holding uncommitted or unmerged work. Build is such a
path, and `cleanupBuildWorktrees` runs unconditionally on every build:

```go
// cmd/codex_build.go:1752
cleaned, orphaned, _ := cleanupBuildWorktrees(phase.ID)
```

It selects every `Allocated` or `InProgress` entry for the phase and calls
`removeGitWorktree` directly (`:1026`). It never calls `worktreeDestructionSafety`, never
calls `preserveWorktreeWork`, and never calls `reportWorktreePreservation`. `Allocated` and
`InProgress` are precisely the statuses a crash between dispatch and finalize leaves
behind — the scenario this phase exists to make safe.

The comment introduced on the *sibling* function claims destruction "now lives only in the
explicitly named, operator-invoked `worktree-reap` command"
(`cmd/codex_build_worktree.go:1057-1059`). That is not accurate while this function exists
and is called.

The ratchet does not catch it for two separate reasons:
- `TestGCNeverCallsRemoveGitWorktree` (`cmd/worktree_crash_safety_test.go:348`) extracts
  only `gcOrphanedWorktrees`'s body by name.
- `TestWorktreeReapHasNoLifecycleCaller` (`:417`) greps lifecycle files for the literal
  strings `runWorktreeReap` and `worktree-reap`. `codex_build.go` contains neither — it
  destroys via a different function, so the test passes while the property fails.

The build path also swallows the error entirely (`, _ :=`), so a failure here is invisible.

**Fix:** Route this function through the same guard, matching `gcOrphanedWorktrees`:

```go
safety := worktreeDestructionSafety(root, entry)
if !safety.Safe {
    _, detail, preserveErr := preserveWorktreeWork(root, entry, safety)
    if preserveErr != nil {
        detail = fmt.Sprintf("could not save automatically (%v); branch %s was left alone", preserveErr, entry.Branch)
    }
    reportWorktreePreservation(safety, fmt.Sprintf("phase %d: %s", entry.Phase, detail))
    entry.Status = colony.WorktreeOrphaned
    remaining = append(remaining, entry)
    orphaned++
    continue
}
```

and widen the ratchet so it asserts the property rather than one function name — scan every
lifecycle file for `removeGitWorktree` and require each occurrence to be guard-preceded, or
assert that `removeGitWorktree`'s only callers are in `worktree_reap.go` and
`allocateBuildWorktree`'s own rollback.

---

## Warnings

### WR-01: `mergePhaseWorktrees` checks out main in the user's working tree and never restores it

**File:** `cmd/codex_build_worktree.go:1224-1240`; same shape at `cmd/worktree.go:811-823`
**Issue:** The merge loop runs `git -C <root> checkout main` (falling back to `master`) in
the **main working tree**, then merges. On any subsequent failure path — merge conflict at
`:1237`, clash detected at `:1218`, tests failing at `:1207` on a later entry — the function
returns with the user's checkout left on `main` rather than whatever branch they were on.
There is no restore. Running a build from a feature branch silently relocates the user's
HEAD.

It also does not check whether the working tree is clean before checking out, so a checkout
that would clobber local modifications is attempted and its failure is reported only as
`"checkout failed"`.

**Fix:** Capture the starting ref (`git rev-parse --abbrev-ref HEAD`) before the loop,
refuse to proceed if `git status --porcelain` is non-empty, and `defer` a checkout back to
the original ref.

### WR-02: Merge-back gate runs the project's test command with no timeout distinction and unbounded output in a blocker

**File:** `cmd/worktree.go:773-788`
**Issue:** `testCmd.CombinedOutput()` failure is reported by embedding the entire captured
output into both the error message and — indirectly — the user-facing output at `:786`.
A verbose test suite produces a multi-megabyte error string. Separately, when the context
deadline fires the error is indistinguishable from a genuine test failure, so the blocker
says "tests failed" when the truth is "tests timed out", which sends the user to debug the
wrong thing. `checkTestsPass` (`cmd/gate.go:202-209`) handles this distinction correctly;
this path does not.

**Fix:** Check `testCtx.Err() == context.DeadlineExceeded` before interpreting `testErr`,
and truncate captured output to a bounded prefix (the ~200-char cap used at
`cmd/gate.go:221-223`) before putting it in a message.

### WR-03: `mergePhaseWorktrees` never records a successful merge in state

**File:** `cmd/codex_build_worktree.go:1242`
**Issue:** After a successful `git merge`, the function appends the branch to `merged` and
moves on. It never sets `state.Worktrees[i].Status = colony.WorktreeMerged`, and never
saves state. The entry therefore stays `InProgress` forever. The next
`gcOrphanedWorktrees` pass sees an `InProgress` entry that is now clean and fully merged,
reports it as "safe to remove", and the state file accumulates entries for work that was
merged long ago.

The test acknowledges this and declines to cover it
(`cmd/merge_phase_worktrees_test.go:270-275`, "a pre-existing gap ... out of scope here").
It is in scope for this phase's own correctness: the preservation report the phase adds is
driven by exactly these statuses.

**Fix:** Wrap the loop in `store.UpdateJSONAtomically` and set the status on each merged
entry, mirroring `worktreeMergeBackCmd`'s Step 6 (`cmd/worktree.go:851-858`).

### WR-04: Test command is executed from repo-controlled text with no allowlist

**File:** `cmd/worktree.go:775-777`; `cmd/codex_build_worktree.go:1165-1166, 1203`
**Issue:** `resolveTestCommand()` reads `CLAUDE.md` or `.aether/data/codebase.md` from the
repository under work and returns free text; both merge gates then split it on whitespace
and pass element 0 to `exec.CommandContext` as the binary. Confirmed:

```
PROBE I: resolveTestCommand()="go test ./... -run TestNothing -count=1" -> exec argv [go test ./... -run TestNothing -count=1]
```

There is no shell, so this is not classic shell injection — but any repository Aether is
pointed at can name an arbitrary executable and arbitrary flags that both merge gates will
then run. `extractTestCommand` (`cmd/gate.go:293-319`) only requires the *line* to contain
the substring `go test`; the returned command is `line[idx:]`, so a `CLAUDE.md` line
reading `go test; anything` yields `go test; anything` and the argv becomes
`["go","test;","anything"]`. Combined with a hostile `PATH` entry or a line crafted to
start at a different `idx`, this is a real execution surface. It is pre-existing in
`checkTestsPass`, but this phase newly routes two more destructive-adjacent gates through
it.

**Fix:** Constrain the resolved command to a known set of launchers
(`go`, `npm`, `yarn`, `pnpm`, `cargo`, `mvn`, `pytest`, `make`) and reject anything else with
the same "cannot determine how to test this project" refusal that already exists. Reject
commands containing shell metacharacters (`;`, `&`, `|`, `` ` ``, `$(`) outright.

### WR-05: `worktree-reap --force` swallows removal errors in the `--include-unmerged` path

**File:** `cmd/worktree_reap.go:154-158`
**Issue:** The clean-path removal failure at `:114-121` calls
`reportWorktreePreservation` so the user hears about it. The `--include-unmerged` removal
failure at `:154-158` does not — it appends to `remaining` and `preservedBranches`
silently, discarding `removeErr` entirely. The user has just been told "Removing worker
workspace on branch X — its changes were saved first" at `:153`, and then nothing. Given
CR-03, the branch may in fact already be deleted at this point.

`reportWorktreePreservation`'s own doc comment (`cmd/worktree_safety.go:215-221`) states it
"must write unconditionally on every call" and warns against callers that suppress it. This
is the suppression it warns about.

**Fix:** Mirror the `:114-121` branch — call `reportWorktreePreservation` with the error
detail before continuing.

### WR-06: `init` deletes the whole worktrees directory based on a count, not on what is in it

**File:** `cmd/init_cmd.go:220-231`
**Issue:** The guard is `if wtPreserved == 0 { os.RemoveAll(<aetherDir>/worktrees) }`. That
count comes from `gcOrphanedWorktrees`, which only examines entries **tracked in
`COLONY_STATE.json`** with `Allocated`/`InProgress`/`Orphaned` status. Any worktree
directory present on disk but absent from state — the exact result of a crash during
`allocateBuildWorktree` between `git worktree add` (`:732`) and
`appendBuildWorktreeEntry` (`:738`) — contributes 0 to `wtPreserved` and is then deleted
by `RemoveAll`, along with everything in it. `RemoveAll` is not `git worktree remove`; it
does not consult git and cannot refuse.

The window is small but it is precisely the crash window this phase is named after, and
`init` is documented as one of the three recovery paths.

**Fix:** Before removing, list what is actually on disk and refuse if the directory is
non-empty:

```go
if wtPreserved == 0 {
    entries, _ := os.ReadDir(filepath.Join(aetherDir, "worktrees"))
    if len(entries) == 0 {
        _ = os.RemoveAll(filepath.Join(aetherDir, "worktrees"))
    } else {
        fmt.Fprintf(os.Stderr, "found %d leftover worker workspace folder(s) from a previous colony that are not in the records — they were left in place rather than deleted\n", len(entries))
    }
}
```

---

## Info

### IN-01: The banned-word test list omits the words the guidance actually names

**File:** `cmd/worktree_safety_test.go:251`, `cmd/worktree_crash_safety_test.go:290`
**Issue:** Both tests check `[]string{"orphan", "gc", "prune"}`. The doc comment they
enforce (`cmd/worktree_safety.go:197-199`) names `"orphaned"`, `"GC"`, `"ratchet"`, and
`"residue"`; CLAUDE.md's glossary additionally lists caste, pheromone, colony, seal,
midden, hive, and worker. `worktree-reap`'s own help text and the string at
`cmd/codex_build_worktree.go:1122` use "worker workspace", which is fine, but the tests
would not catch a regression that reintroduced "midden" or "caste".
**Fix:** Extend the banned list to the glossary terms the project rule actually enumerates,
and share one list between both tests rather than duplicating the literal.

### IN-02: `generateWorktreeID` ignores the error from `rand.Read`

**File:** `cmd/worktree.go:81-85`
**Issue:** `rand.Read(rnd)` discards its error return. On failure `rnd` stays all-zero and
every ID in that process collapses to `wt_<timestamp>_00000000`, which collides for
worktrees allocated within the same second — plausible for a parallel wave, since
`allocateBuildWorktree` calls this once per worker. Pre-existing, but the ID is now
load-bearing for state reconciliation.
**Fix:** `if _, err := rand.Read(rnd); err != nil { return fmt.Sprintf("wt_%d_%d", time.Now().Unix(), time.Now().UnixNano()) }`.

### IN-03: Duplicated safety-gate logic between the two merge paths

**File:** `cmd/worktree.go:749-804` and `cmd/codex_build_worktree.go:1161-1243`
**Issue:** The test-command resolution, the refusal message, the clash gate, the
main/master checkout fallback, and the merge are near-identical in both places, with the
refusal text duplicated word-for-word in two long string literals. They have already
diverged: `worktreeMergeBackCmd` creates a blocker flag on refusal, `mergePhaseWorktrees`
does not; `worktreeMergeBackCmd` updates state after merging, `mergePhaseWorktrees` does not
(WR-03). Two copies with different behaviour is how the next divergence lands unnoticed.
**Fix:** Extract a shared `runWorktreeMergeGates(root, wtAbsPath, branch) error` used by
both, keeping only the blocker-creation and state-update differences at the call sites.

---

_Reviewed: 2026-08-18_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
