# Phase 187: Crash-Safe Worktrees & Ecosystem Neutrality - Pattern Map

**Mapped:** 2026-08-18
**Files analyzed:** 6 primary change targets (2 net-new possible: a shared dirty/unmerged guard
helper and a preserve-report helper), 4 call sites converted from implicit-destroy to
preserve-and-report, 1 ratchet test file
**Analogs found:** 6 / 6

**Correction to CONTEXT.md line numbers found during this pass** (reported per this agent's
instructions, not smoothed over):
- `cmd/codex_build_worktree.go:1118` (CONTEXT.md's cited line for the hard-coded `go test ./...`
  in `mergePhaseWorktrees`) is actually `wtAbsPath = filepath.Join(root, wtAbsPath)`. The real
  hard-coded test command is at **line 1124**: `testCmd := exec.CommandContext(testCtx, "go",
  "test", "./...")`. The function itself starts at line 1098, matching CONTEXT.md.
- `cmd/worktree.go:749` (CONTEXT.md's cited line) is `defer testCancel()`. The real hard-coded
  test command is two lines down at **line 751**: `testCmd := exec.CommandContext(testCtx, "go",
  "test", "./...")`. Close enough that CONTEXT.md's intent is unambiguous, but the planner should
  cite 751, not 749, in any line-anchored plan text.
- `cmd/verification_command_section.go` is 24 lines total; CONTEXT.md cites line 10 for "the
  comment recording that this exact oversight already happened once" — confirmed, that comment
  block spans lines 10-15.
- No dedicated small "grep a banned literal out of `.go` source and fail" ratchet test exists in
  this codebase today (see "No Analog Found" and "Shared Patterns: Ratchet Shape" below). The
  closest real analogs are `TestSpawnRootSentinelsCoverEveryDocumentedCoordinatorParent`
  (`cmd/spawn_enforce_test.go:690-769`, walks directories of markdown/source and greps lines) and
  `TestRetiredPackagesStayRetired` (`cmd/retired_packages_test.go:29-71`, asserts a deleted path
  stays deleted). Neither is a literal "banned string in `.go` source" ratchet — the planner will
  be composing a new pattern from these two, not copying one wholesale.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `cmd/codex_build_worktree.go` (`removeGitWorktree`, `gcOrphanedWorktrees`) | service (destructive git op) | request-response (shell-out + parse) | `cmd/recover_scanner.go:257` `scanDirtyWorktrees` + `cmd/recover_repair.go:610` `repairDirtyWorktree` | exact (safety-check + preservation) |
| `cmd/codex_build_worktree.go` (`mergePhaseWorktrees`, line 1124) | service (CI-gate + merge) | request-response | `cmd/gate.go:244` `resolveTestCommand` via `cmd/gate.go:176` call site | role-match (wiring job) |
| `cmd/worktree.go` (`worktreeMergeBackCmd`, line 751) | controller (cobra command) | request-response | `cmd/verification_command_section.go:19` call site of `resolveTestCommand` | role-match (wiring job) |
| new/extended: shared dirty+unmerged guard (likely `cmd/codex_build_worktree.go` or a new `cmd/worktree_safety.go`) | utility (safety gate) | request-response | `cmd/recover_scanner.go:257` `scanDirtyWorktrees` | exact |
| new/extended: preserve-and-report helper | utility (state mutation + user-facing report) | request-response | `cmd/recover_repair.go:610` `repairDirtyWorktree` (stash branch) + `cmd/recover_repair.go:144-152` (`isDestructiveCategory`) | exact |
| named destruction command (new subcommand or flag on `worktree-orphan-scan`/`recover`) | controller (cobra command, explicit `--force`) | request-response | `cmd/recover_repair.go:154-167` `confirmRepair` + `isDestructiveCategory` gating | role-match |
| test: crash-safe worktree guard (dirty/unmerged preserved, not deleted) | test | request-response (real git fixture) | `cmd/worktree_test.go:1392-1461` `TestWorktreeMergeBackSuccess` and `cmd/worktree_test.go:1196-1289` `TestWorktreeMergeBackTestsFail` | exact |
| test: ecosystem-neutral merge gate (non-Go project) | test | request-response (real git fixture, non-Go) | same as above, adapted (swap `go.mod`/`go test` fixture for a Node/`package.json` fixture or an unresolved-command fixture) | role-match |
| test: ratchet against `branch -D` on destructive path / literal `go test ./...` on merge path | test (static source ratchet) | batch (file walk) | `cmd/spawn_enforce_test.go:690-769` `TestSpawnRootSentinelsCoverEveryDocumentedCoordinatorParent` (walk+grep shape) + `cmd/retired_packages_test.go:29-71` (deleted-thing-stays-deleted shape) | partial (compose two patterns, no single exact analog) |
| state writes touching `COLONY_STATE.json.Worktrees` (preservation, orphan status) | state-mutation | CRUD | `cmd/codex_build_worktree.go:1059` `gcOrphanedWorktrees`'s `store.UpdateJSONAtomically` call | exact |

## Pattern Assignments

### The safety-check analog: `scanDirtyWorktrees`

**Source:** `cmd/recover_scanner.go:257-319`

This is the ONLY existing uncommitted-changes check in the codebase. It is read-only (a scanner,
not a repair) and demonstrates the exact shell-out + parse + report shape the new guard in front
of `removeGitWorktree` needs.

**Shell-out and porcelain parse** (`cmd/recover_scanner.go:292-302`):
```go
// Run git status --porcelain in the worktree directory.
cmd := exec.Command("git", "-C", wt.Path, "status", "--porcelain")
output, err := cmd.Output()
if err != nil {
    continue
}
if strings.TrimSpace(string(output)) != "" {
    lineCount := len(strings.Split(strings.TrimSpace(string(output)), "\n"))
    issues = append(issues, issueCritical("dirty_worktree", wt.Path,
        fmt.Sprintf("Worktree has %d uncommitted change(s)", lineCount)))
}
```

**Critical gap this phase must close** (`cmd/recover_scanner.go:267-271`):
```go
for _, wt := range state.Worktrees {
    // Skip already-merged or orphaned entries.
    if wt.Status == colony.WorktreeMerged || wt.Status == colony.WorktreeOrphaned {
        continue
    }
```
CONTEXT.md's `<specifics>` section is confirmed by this exact code: `scanDirtyWorktrees` skips
`WorktreeOrphaned` — but `gcOrphanedWorktrees` (below) specifically selects `WorktreeOrphaned`
entries for force-deletion. The one existing safety check structurally cannot see the entries
GC is about to destroy. Any new guard reused for the destructive path must NOT inherit this
skip, or it reproduces the same blind spot on the path that matters most.

**Disk-presence + git-worktree-list cross-reference** (`cmd/recover_scanner.go:264-290`) — also
worth copying as a pre-check before running `git status`, since a stale state entry pointing at a
deleted path will make `git -C <path> status` fail rather than report cleanly:
```go
diskPaths := getGitWorktreePaths()
...
if _, err := os.Stat(wt.Path); os.IsNotExist(err) {
    // state says allocated/in-progress but path is gone
}
if !diskPaths[wt.Path] {
    // state has it, git worktree list does not
}
```

---

### The preservation analog: `repairDirtyWorktree` and destructive classification

**Source:** `cmd/recover_repair.go:602-677` (function), `cmd/recover_repair.go:144-152`
(classification), `cmd/recover_repair.go:154-167` (confirmation gate)

**Stash-not-discard preservation** (`cmd/recover_repair.go:642-648`):
```go
case strings.Contains(msg, "uncommitted change"):
    cmd := exec.Command("git", "-C", issue.File, "stash", "--include-untracked")
    if output, err := cmd.CombinedOutput(); err != nil {
        record.Error = fmt.Sprintf("git stash failed: %v: %s", err, string(output))
        return record
    }
    record.Action = "stash_worktree_changes"
```
This is the direct analog for D-01's "stash or commit, never discard" requirement. Note it uses
`git stash` specifically, not `git add -A && git commit`; D-01 leaves the choice to the planner,
but this is the one existing precedent and reuses cleanly since it is already tested.

**Destructive classification gates behind `--force`** (`cmd/recover_repair.go:144-152`):
```go
// isDestructiveCategory returns true for repair categories that can mutate
// user data (worktree files, manifest content) in potentially irreversible ways.
func isDestructiveCategory(category string) bool {
    switch category {
    case "dirty_worktree", "bad_manifest":
        return true
    }
    return false
}
```

**Interactive confirmation prompt** (`cmd/recover_repair.go:154-167`) — the confirmation channel
used when `--force` is absent; D-02's plain-language reporting requirement is the non-interactive
descendant of this same idea (report on every occurrence rather than gate on it):
```go
func confirmRepair(issue HealthIssue) bool {
    fmt.Fprintf(os.Stderr, "\n  [confirm] %s (%s)\n", issue.Message, issue.File)
    fmt.Fprintf(os.Stderr, "  Apply fix? [y/N]: ")
    reader := bufio.NewReader(os.Stdin)
    response, err := reader.ReadString('\n')
    ...
}
```
D-01 explicitly rules out interactive blocking ("Stopping to ask blocks unattended runs"), so the
new preserve-and-continue path should follow the *classification* half of this pattern
(`isDestructiveCategory`-style gating for the separately named destruction command) but not the
*interactive-prompt* half — `fmt.Fprintf(os.Stderr, ...)` unconditional reporting (no stdin read)
is the right descendant for D-02.

**Existing "mark orphaned instead of delete" precedent already in the target file** — this is the
Reusable Asset CONTEXT.md calls `preserveWorktree`, confirmed at
`cmd/codex_build_worktree.go:389-457` (`reconcileWorktreeWave`). The `preserveWorktree` local
variable (first set at line 412) is only ever used to call
`updateBuildWorktreeStatus(session.Branch, colony.WorktreeOrphaned)` (line 443) — it never calls
`removeGitWorktree`. This confirms CONTEXT.md's claim: "the intent exists; it is undone
downstream" — downstream being `gcOrphanedWorktrees`, which later force-deletes exactly the
`WorktreeOrphaned` entries this function was careful to preserve.

---

### The test-command wiring analog: `resolveTestCommand` and its call sites

**Source (resolver):** `cmd/gate.go:242-290`

**Full resolver body** (`cmd/gate.go:244-289`) — priority chain CLAUDE.md -> CODEBASE.md ->
language sniff -> empty:
```go
func resolveTestCommand() string {
    repoRoot := ""
    if store != nil && strings.TrimSpace(store.BasePath()) != "" {
        repoRoot = filepath.Dir(filepath.Dir(store.BasePath()))
    }
    if repoRoot == "" {
        repoRoot = storage.ResolveAetherRoot(context.Background())
    }
    claudeMD := repoRoot + "/CLAUDE.md"
    if data, err := os.ReadFile(claudeMD); err == nil {
        cmd := extractTestCommand(string(data))
        if cmd != "" { return cmd }
    }
    codebaseMD := repoRoot + "/.aether/data/codebase.md"
    if data, err := os.ReadFile(codebaseMD); err == nil {
        cmd := extractTestCommand(string(data))
        if cmd != "" { return cmd }
    }
    if _, err := os.Stat(repoRoot + "/go.mod"); err == nil { return "go test ./..." }
    if _, err := os.Stat(repoRoot + "/package.json"); err == nil { return "npm test" }
    if _, err := os.Stat(repoRoot + "/Cargo.toml"); err == nil { return "cargo test" }
    if _, err := os.Stat(repoRoot + "/pom.xml"); err == nil { return "mvn test" }
    return ""
}
```
Note the resolver is rooted against `store.BasePath()`'s parent-of-parent (the colony's own
root), not process cwd — there is a comment (`cmd/gate.go:245-249`) explaining a prior bug where
this resolved to the Aether repo itself. Any new call site inside a worktree must pass or resolve
the *worktree's* root the same deliberate way, not assume `resolveAetherRoot()`/`os.Getwd()` is
correct inside a worktree context.

**Call site 1 — `checkTestsPass`** (`cmd/gate.go:174-184`), the empty-string handling pattern
D-03's boundary rests on:
```go
func checkTestsPass() gateCheck {
    testCmd := resolveTestCommand()
    if testCmd == "" {
        // No test command found — pass by default (no tests to run)
        return gateCheck{
            Name:   "tests_pass",
            Passed: true,
            Detail: "no test command found, skipping",
        }
    }
    ...
```
**This is the pattern D-03 explicitly REJECTS for the merge-gate case.** `checkTestsPass` passes
by default on an empty resolver result ("no tests to run"). D-03 requires the opposite for a
merge gate: empty result means "cannot determine how to test" -> refuse to merge, preserve the
work, and say why. The planner must not copy this call site's pass-by-default branch; only its
call shape (`testCmd := resolveTestCommand(); if testCmd == "" { ... }`) transfers.

**Call site 2 — `renderVerificationCommandSection`** (`cmd/verification_command_section.go:1-24`,
full file) — the comment recording the prior wiring failure that CONTEXT.md cites, and the
correct empty-string handling for a "skip silently, this isn't a hard gate" context:
```go
// resolveTestCommand has existed since the gate was written, and its only
// caller was the gate itself — which runs *after* the worker has finished. So
// every builder and every reviewer worked out how to run the project's tests
// from scratch, every phase, by reading CLAUDE.md or guessing. ...
func renderVerificationCommandSection() string {
    command := strings.TrimSpace(resolveTestCommand())
    if command == "" {
        return ""
    }
    return fmt.Sprintf("\n## Verification Command\n\nRun this to check your work: `%s`\n", command)
}
```

**The two sites that must adopt `resolveTestCommand()` instead of hard-coding:**

`cmd/codex_build_worktree.go:1122-1127` (function `mergePhaseWorktrees`, starts line 1098):
```go
// Gate 1: run tests in worktree
testCtx, testCancel := context.WithTimeout(context.Background(), BuildTimeout)
testCmd := exec.CommandContext(testCtx, "go", "test", "./...")
testCmd.Dir = wtAbsPath
_, testErr := testCmd.CombinedOutput()
testCancel()
if testErr != nil {
    failed = append(failed, fmt.Sprintf("%s (tests failed)", entry.Branch))
    continue
}
```
The hard-coded literal is on line 1124, not 1118 as CONTEXT.md states (see correction note at
top). This function has `err error` in its return signature already
(`func mergePhaseWorktrees(phaseNum int) (merged []string, failed []string, err error)`), so an
empty-resolver refuse-and-preserve path can return through the existing `err` without a signature
change — it just needs a new branch alongside the existing `failed = append(...)` branches that
does NOT call `removeGitWorktree`/does not proceed to merge.

`cmd/worktree.go:749-763` (inside `worktreeMergeBackCmd.RunE`, command starts line 690):
```go
// Step 2: Gate 1 -- Run tests in worktree directory
testCtx, testCancel := context.WithTimeout(context.Background(), BuildTimeout)
defer testCancel()
testCmd := exec.CommandContext(testCtx, "go", "test", "./...")
testCmd.Dir = wtAbsPath // CRITICAL: run in worktree directory (absolute path)
testOutput, testErr := testCmd.CombinedOutput()
if testErr != nil {
    // Tests failed -- create blocker and block merge
    blockerDesc := fmt.Sprintf("Merge blocked: tests failed for %s", branch)
    if createErr := createBlocker(store, blockerDesc, "worktree-merge-back"); createErr != nil {
        outputError(2, fmt.Sprintf("tests failed AND failed to create blocker: %v", createErr), nil)
        return nil
    }
    outputError(2, fmt.Sprintf("merge blocked: tests failed for %s: %s", branch, string(testOutput)), nil)
    return nil
}
```
The hard-coded literal is on line 751 (CONTEXT.md cites 749, which is `defer testCancel()` — see
correction note). This site already has a `createBlocker(...)` + `outputError(...)` pattern for
the ordinary test-failure case; the new "cannot determine test command" case should follow the
exact same blocker-creation shape with different wording ("cannot determine how to test this
project" vs "tests failed"), not invent a new reporting channel.

**The `Long` help text also needs updating** (`cmd/worktree.go:693-696`) — CONTEXT.md's claim is
confirmed, this text literally says "go test ./...":
```go
Long: "Merges a tracked worktree branch back to the main branch. " +
    "Two gates must pass before merge: (1) go test ./... in the worktree, " +
    "(2) clash detection to prevent file conflicts. On failure, a blocker " +
    "flag is created. On success, the worktree is cleaned up automatically.",
```

---

### The destructive path with no guard: `removeGitWorktree` / `gcOrphanedWorktrees`

**Source:** `cmd/codex_build_worktree.go:819-837` (function), `1044-1090` (caller)

**The unconditional destruction** (`cmd/codex_build_worktree.go:819-837`) — no dirty check, no
unmerged-commit check, before this phase:
```go
func removeGitWorktree(root, absPath, branch string) error {
    ctx, cancel := context.WithTimeout(context.Background(), GitTimeout)
    defer cancel()

    var errs []string
    if out, err := exec.CommandContext(ctx, "git", "-C", root, "worktree", "remove", absPath, "--force").CombinedOutput(); err != nil {
        errs = append(errs, fmt.Sprintf("worktree remove: %v (output: %s)", err, string(out)))
    }
    if out, err := exec.CommandContext(ctx, "git", "-C", root, "worktree", "prune").CombinedOutput(); err != nil {
        errs = append(errs, fmt.Sprintf("worktree prune: %v (output: %s)", err, string(out)))
    }
    if out, err := exec.CommandContext(ctx, "git", "-C", root, "branch", "-D", branch).CombinedOutput(); err != nil {
        errs = append(errs, fmt.Sprintf("branch delete: %v (output: %s)", err, string(out)))
    }
    ...
}
```
`--force` on `worktree remove` and `-D` (not `-d`) on `branch delete` are both unconditional
destructive flags — this is what CONTEXT.md's criterion 1 targets.

**The caller that selects orphaned entries for destruction**
(`cmd/codex_build_worktree.go:1048-1090`, full function reproduced since it is the exact target):
```go
func gcOrphanedWorktrees() (cleaned int, orphaned int, err error) {
    if store == nil { return 0, 0, fmt.Errorf("no store initialized") }
    if _, statErr := os.Stat(filepath.Join(store.BasePath(), "COLONY_STATE.json")); statErr != nil {
        if os.IsNotExist(statErr) { return 0, 0, nil }
        return 0, 0, statErr
    }
    var state colony.ColonyState
    if err := store.UpdateJSONAtomically("COLONY_STATE.json", &state, func() error {
        root := storage.ResolveAetherRoot(context.Background())
        var remaining []colony.WorktreeEntry
        for _, entry := range state.Worktrees {
            if entry.Status != colony.WorktreeAllocated && entry.Status != colony.WorktreeInProgress && entry.Status != colony.WorktreeOrphaned {
                remaining = append(remaining, entry)
                continue
            }
            absPath := filepath.Join(root, entry.Path)
            if _, statErr := os.Stat(absPath); statErr != nil && os.IsNotExist(statErr) {
                cleaned++
                continue
            }
            if removeErr := removeGitWorktree(root, absPath, entry.Branch); removeErr != nil {
                entry.Status = colony.WorktreeOrphaned
                remaining = append(remaining, entry)
                orphaned++
            } else {
                cleaned++
                // Don't append — entry is removed
            }
        }
        state.Worktrees = remaining
        return nil
    }); err != nil {
        return 0, 0, err
    }
    return cleaned, orphaned, nil
}
```
This confirms CONTEXT.md exactly: no phase filter, and it explicitly includes
`colony.WorktreeOrphaned` in the set selected for destruction (line: `entry.Status !=
colony.WorktreeAllocated && entry.Status != colony.WorktreeInProgress && entry.Status !=
colony.WorktreeOrphaned` — i.e. it proceeds unless status is NONE of these three, meaning
Orphaned entries ARE eligible for `removeGitWorktree`). This is precisely the asymmetry
CONTEXT.md's `<specifics>` section describes: `preserveWorktree` upstream marks an entry
`Orphaned` specifically to protect it, and `gcOrphanedWorktrees` downstream treats `Orphaned` as
a delete-me signal.

---

### The state-mutation analog: `store.UpdateJSONAtomically`

**Source:** `cmd/codex_build_worktree.go:1059-1086` (shown in full above)

Any new state writes this phase introduces (marking preserved entries, recording report events,
updating the destruction command's own state changes) should follow this exact
read-modify-write-atomically shape: `store.UpdateJSONAtomically("COLONY_STATE.json", &state,
func() error { ...mutate state fields directly, return nil or error... })`. This is preferred over
the separate `store.LoadJSON` + `store.SaveJSON` pair used in `cmd/worktree.go`'s
`worktreeMergeBackCmd` (lines 711 and 830) — that older pattern has a race window between load and
save that `UpdateJSONAtomically` closes. Since D-01 requires the merge-gate refusal path to also
avoid destroying anything, and D-02 requires reporting "on every occurrence," any code path that
both reports and mutates state should mutate through `UpdateJSONAtomically`, then report
afterward from the returned counts — matching `gcOrphanedWorktrees`'s own
`cleaned`/`orphaned` return-then-log shape at its two call sites
(`cmd/codex_continue.go:540-543`, `cmd/session_flow_cmds.go:261` before its
`buildResumeDashboardResult()`, and `cmd/init_cmd.go:201-203`).

---

### The four callers that make it dangerous today

**`cmd/codex_continue.go:539-543`** — comment claims non-blocking but the call is synchronous and
the error is discarded (confirmed):
```go
// Background cleanup of orphaned worktrees — non-blocking
gcCleaned, gcOrphaned, _ := gcOrphanedWorktrees()
if gcCleaned > 0 || gcOrphaned > 0 {
    emitVisualProgress(fmt.Sprintf("Worktree cleanup: %d cleaned, %d orphaned", gcCleaned, gcOrphaned))
}
```
`emitVisualProgress` here already reports a count — D-02 needs this widened to name what was
*preserved* (not just cleaned/orphaned) and why, once destruction stops being implicit.

**`cmd/session_flow_cmds.go:260-262`** — the exact path the owner reacted to (resuming can
destroy the work being resumed), confirmed no guard before the call:
```go
// Clean up any orphaned worktrees before resuming
gcCleaned, gcOrphaned, gcErr := gcOrphanedWorktrees()
```
(`gcErr` is captured here, unlike the `continue` site — worth checking downstream in this
function whether it's actually surfaced to the user or silently dropped; not fully traced in this
pass since it's outside the cited line range.)

**`cmd/init_cmd.go:200-205`** — confirmed, followed immediately by a full directory nuke that is
separate from `gcOrphanedWorktrees` but compounds the risk:
```go
// Clean up any leftover worktrees from previous colony
if cleaned, orphaned, err := gcOrphanedWorktrees(); err == nil && (cleaned > 0 || orphaned > 0) {
    fmt.Fprintf(os.Stderr, "warning: cleaned %d stale worktree(s), %d orphaned\n", cleaned, orphaned)
}
// Also remove the worktrees directory entirely to ensure a clean slate
_ = os.RemoveAll(filepath.Join(aetherDir, "worktrees"))
```
This site already does the best of the three at reporting (`fmt.Fprintf(os.Stderr, "warning:
...")`) — closest existing precedent for D-02's "plain language, every occurrence" wording, though
it still needs to become preserve-aware rather than report-after-destroy.

---

## Shared Patterns

### Dirty/unmerged-work detection
**Source:** `cmd/recover_scanner.go:257-319` (`scanDirtyWorktrees`)
**Apply to:** the new guard called from inside `removeGitWorktree`/`gcOrphanedWorktrees`, and
potentially reused directly rather than reimplemented, per CONTEXT.md's Reusable Assets note.
Needs one modification: do not skip `WorktreeOrphaned` when this function is being used to decide
whether the *GC path* may proceed (see gap noted above) — the skip is correct for the read-only
health-report use case but wrong for a pre-destruction gate.

### Stash-based preservation, never discard
**Source:** `cmd/recover_repair.go:642-648` (`repairDirtyWorktree`'s uncommitted-change branch)
**Apply to:** any new preserve-instead-of-delete path in `codex_build_worktree.go`.

### Destructive-operation classification + explicit `--force`
**Source:** `cmd/recover_repair.go:144-152` (`isDestructiveCategory`)
**Apply to:** the named, operator-invoked destruction command D-01 requires — it should classify
itself the same way and require `--force` (or an equivalent explicit flag), never running
implicitly from `resume`, `continue`, or `init`.

### Plain-language stderr reporting on every occurrence
**Source:** `cmd/init_cmd.go:202` (`fmt.Fprintf(os.Stderr, "warning: cleaned %d stale
worktree(s), %d orphaned\n", cleaned, orphaned)`), refined by
`cmd/verification_command_section.go`'s comment on the cost of NOT reporting
(`resolveTestCommand exists but nobody calls it` — the exact prior-oversight class D-02 exists to
prevent from recurring on the preservation path).
**Apply to:** all four call sites (`codex_continue.go:540`, `session_flow_cmds.go:261`,
`init_cmd.go:201`) plus any new named destruction command.

### Test-command resolution instead of hard-coded `go test ./...`
**Source:** `cmd/gate.go:244-289` (`resolveTestCommand`), called via
`cmd/verification_command_section.go:19`'s trim-and-check-empty shape (not `checkTestsPass`'s
pass-by-default shape — see explicit rejection note above).
**Apply to:** `cmd/codex_build_worktree.go:1124` and `cmd/worktree.go:751`.

### Atomic state mutation
**Source:** `cmd/codex_build_worktree.go:1059` (`store.UpdateJSONAtomically`)
**Apply to:** any new writes to `state.Worktrees` for preservation/orphan-status changes.

### Real-git test fixtures
**Source:** `cmd/worktree_test.go:993` (via `cmd/init_cmd_test.go:993` `runGit` helper),
`cmd/worktree_test.go:1196-1289` (`TestWorktreeMergeBackTestsFail`),
`cmd/worktree_test.go:1392-1461` (`TestWorktreeMergeBackSuccess`)
**Apply to:** every new test in this phase. The shape is: `t.TempDir()` -> `runGit(t, tmpDir,
"init")` + author/committer env config -> `checkout -b main` -> write `go.mod` (or, for the
ecosystem-neutrality tests, `package.json`/nothing at all) -> `git add . && git commit` -> `git
worktree add -b <branch> <path> HEAD` inside the same tmpDir -> write a test file (passing or
failing) into the worktree -> build a `colony.WorktreeEntry` + `makeTestStateWithWorktrees` ->
`os.Setenv("AETHER_ROOT", tmpDir)` -> execute the real cobra command via `rootCmd.SetArgs(...)`
and `rootCmd.Execute()` -> assert on `stderrBuf`/`stdoutBuf` content and on reloaded
`COLONY_STATE.json`.

**Contrast — the fixture shape NOT to follow:**
`cmd/codex_build_worktree_test.go:717-756` (`TestCleanupBuildWorktrees`) uses a `Path:
".aether/worktrees/test"` that is never created on disk. Its own code comment admits the
consequence: `// The worktree path doesn't exist on disk, so removal should "succeed" (no-op)`.
This means the actual `git worktree remove --force` / `git branch -D` commands inside
`removeGitWorktree` are never exercised by this test — os.Stat's IsNotExist short-circuits before
`removeGitWorktree` is even called (see the `gcOrphanedWorktrees` body above: `if _, statErr :=
os.Stat(absPath); statErr != nil && os.IsNotExist(statErr) { cleaned++; continue }`). Any new test
claiming to cover the destructive branch must use a real `git worktree add`-created path, per the
`worktree_test.go` fixtures above, not this JSON-only shape.

## No Analog Found

| File/Test | Role | Data Flow | Reason |
|---|---|---|---|
| grep-ratchet against `branch -D` in the destructive-removal function | test (static source ratchet) | batch | No existing test in this codebase does a plain "walk `.go` files, grep for a banned literal string, fail if found outside an allowlist" over Go *source* (as opposed to markdown/YAML/docs corpora). The two nearest shapes — `cmd/spawn_enforce_test.go:690-769` (walk+regex+cross-reference over `.claude`/`.opencode`/`.aether` markdown, not `.go`) and `cmd/retired_packages_test.go:29-71` (asserts a *path* stays deleted, not that a *string* stays absent from source) — must be composed, not copied. The planner should specify: walk `cmd/*.go` (excluding `_test.go` files and excluding the ratchet's own file, matching `subcommand_reachability_ratchet_test.go:1382`'s self-exclusion idiom), grep for the literal `branch", "-D"` / `"-D"` combined with `branch` on the same exec.Command call, and fail listing file:line — mirroring `TestSpawnRootSentinelsCoverEveryDocumentedCoordinatorParent`'s "found no occurrences at all -> fail, this test would pass vacuously forever" self-check at line 750-752, which is itself a `retired_packages_test.go`-style discipline worth carrying over. |
| grep-ratchet against literal `"go test", "./..."` outside `resolveTestCommand`'s own fallback | test (static source ratchet) | batch | Same composition gap as above. The one wrinkle: `resolveTestCommand()` itself legitimately contains the literal `"go test ./..."` as its Go-language fallback (`cmd/gate.go:277`), so the ratchet must allowlist that one file/line rather than banning the string outright — closer in spirit to `cmd/retired_packages_test.go:65` (`for _, needle := range []string{...}` allowlist-of-expected-occurrences) than a blanket ban. |
| ecosystem-neutral (non-Go) merge-gate fixture | test | request-response, real git, non-Go | No existing test builds a Node/Rust/no-recognized-manifest worktree fixture for the merge-back path — all current `worktree_test.go` fixtures write a `go.mod`. This is genuinely new test surface; use the same `runGit`-based fixture construction but omit `go.mod` (and any of `package.json`/`Cargo.toml`/`pom.xml`) to exercise the `resolveTestCommand() == ""` branch, and separately add one with a `package.json` fixture to prove a real non-Go project can merge. |

## Metadata

**Analog search scope:** `cmd/*.go` and `cmd/*_test.go` (excluded `.claude/worktrees/agent-*/`
per instructions — those are stale full repo copies with misleading line numbers)
**Files scanned directly (Read or targeted grep):** `cmd/recover_scanner.go`,
`cmd/recover_repair.go`, `cmd/codex_build_worktree.go`, `cmd/worktree.go`, `cmd/gate.go`,
`cmd/verification_command_section.go`, `cmd/worktree_test.go`, `cmd/codex_build_worktree_test.go`,
`cmd/codex_continue.go` (targeted), `cmd/session_flow_cmds.go` (targeted), `cmd/init_cmd.go`
(targeted), `cmd/spawn_enforce_test.go`, `cmd/retired_packages_test.go`, plus grep sweeps across
`cmd/*_test.go` for ratchet-shaped tests (`ci_wiring_gate_test.go`, `command_call_audit_test.go`,
`regression_test.go`, `subcommand_reachability_ratchet_test.go`,
`go_source_hint_audit_test.go`)
**Pattern extraction date:** 2026-08-18
