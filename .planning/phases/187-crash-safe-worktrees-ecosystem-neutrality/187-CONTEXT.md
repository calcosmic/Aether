# Phase 187: Crash-Safe Worktrees & Ecosystem Neutrality - Context

**Gathered:** 2026-08-18
**Status:** Ready for planning
**Source:** discuss-phase (owner decisions D-01..D-05, 2026-08-18), grounded in a read-only
reconnaissance of the worktree lifecycle performed during this session

<domain>
## Phase Boundary

This phase delivers two things:

1. **No Aether command ever destroys unmerged or uncommitted work.** Today several paths call
   `git worktree remove --force` followed by `git branch -D`, with no dirty check and no
   unmerged-commit check. This phase makes destruction conditional on the work being provably
   recoverable, and preserves it otherwise.
2. **Worker isolation works outside Go repositories.** The merge gate hard-codes `go test ./...`
   in two places, so in a Node, Python or Rust project the gate can never pass and work never
   merges back.

Out of scope: changing how worktrees are allocated, changing the parallel-execution model,
introducing a new recovery UI, and any change to the in-repo (non-worktree) execution path
beyond what these two goals require. Reworking `WorktreeMerged`'s dual meaning is noted in
`<deferred>`, not done here.

</domain>

<decisions>
## Implementation Decisions

### Preservation policy — what happens to work that cannot be safely deleted

- **D-01 (locked):** When Aether encounters worktree work that is dirty (uncommitted changes)
  or holds unmerged commits, it **preserves the work and continues** — it does not delete, and
  it does not stop and ask. Preservation means committing or stashing the changes onto the
  worktree's own branch so nothing is lost, and keeping the branch. The run proceeds.

  *Rationale from the owner:* refusing to act at all lets leftovers accumulate silently — that
  is exactly how ten stranded branches piled up between May and July 2026. Stopping to ask
  blocks unattended runs, which defeats autopilot. Save-and-continue is the only option that is
  both non-destructive and non-blocking.

  *Implication the planner must honour:* destruction is deferred to an explicit, named
  operator-invoked command — never to an automatic cleanup running inside `resume`, `continue`
  or `init`.

- **D-02 (locked):** Every preservation, refusal, or skip is **reported in plain language on
  every occurrence**. Silent handling is prohibited on this path.

  *Rationale:* the current cleanup is silent and synchronous inside `resume`, which is why the
  earlier losses went unnoticed for months. A destructive-capable path that says nothing is
  indistinguishable from one that works.

### Ecosystem neutrality — what happens when the test command is unknown

- **D-03 (locked):** When Aether cannot determine how to test a project, it **refuses to merge
  and says why**. It never merges unverified work, and it never destroys the work it declined
  to merge — the branch and worktree are preserved per D-01.

  *Rationale:* merging unverified work would put unchecked changes into the user's project;
  asking interactively would block unattended runs. Refusing while preserving is the only
  option that is safe in both directions.

  *Boundary:* "cannot determine" means `resolveTestCommand()` returns empty. A resolved command
  that runs and fails is an ordinary test failure, not this case, and keeps existing behaviour.

### Claude's Discretion

- The exact preservation mechanism (commit-on-branch vs `git stash` vs both, and which is
  chosen when) — provided nothing is discarded and the result is recoverable by a named
  command that the run tells the user about.
- The precise wording and channel of the D-02 report, provided it appears on every occurrence
  and is legible to a non-technical reader.
- Whether the named destruction command is a new subcommand or a flag on the existing
  `worktree-orphan-scan` / `recover` surface — provided it is explicit, operator-invoked, and
  never runs implicitly from `resume`, `continue`, `init` or a build.
- How `resolveTestCommand()` is threaded into both merge sites (shared helper vs direct call),
  and whether the two duplicate merge-gate implementations are unified — provided both sites
  stop hard-coding `go test ./...`.
- Test-fixture strategy, provided the criteria below are met with real git repositories where
  the behaviour under test is a git behaviour.

### Folded Todos

None. Two todos scored 0.6 against this phase and both were reviewed and rejected as unrelated
— see `<deferred>`.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase governance
- `.planning/ROADMAP.md` § "Phase 187" — the goal and the three success criteria this phase is
  judged against; note `Depends on: nothing` since the 2026-08-18 rescope
- `.planning/HARDENING-PLAN.md` — the finish line and its 2026-08-17 addendum
- `CLAUDE.md` § "Definition of Done" — a requirement is satisfied only when a command exists
  that fails when the requirement is unmet

### The code under change
- `cmd/codex_build_worktree.go:819` — `removeGitWorktree`: `worktree remove --force` then
  `branch -D`; the unconditional destruction both criteria 1 and 3 target
- `cmd/codex_build_worktree.go:1048` — `gcOrphanedWorktrees`: selects `Allocated`,
  `InProgress` and `Orphaned` entries with no phase filter and no dirty/unmerged check
- `cmd/codex_build_worktree.go:1098` — `mergePhaseWorktrees`: hard-coded `go test ./...` at
  line 1118; the function criterion 3 requires real-path tests for
- `cmd/worktree.go:691` — `worktree-merge-back` command: the second hard-coded `go test ./...`
  at line 749; its `Long` help text also states the Go command literally
- `cmd/gate.go:244` — `resolveTestCommand()`: the existing resolver (CLAUDE.md →
  `.aether/data/codebase.md` → language sniff → empty) that criterion 2 requires be used
- `cmd/verification_command_section.go:10` — the comment recording that this exact
  "resolveTestCommand exists but nobody calls it" oversight already happened once

### The callers that make it dangerous
- `cmd/codex_continue.go:540` — GC inside `continue`; comment claims "background,
  non-blocking" but it is synchronous and the error is discarded
- `cmd/session_flow_cmds.go:261` — GC inside `resume`, before restoring; the path where
  resuming can destroy the work being resumed
- `cmd/init_cmd.go:201` — GC inside `init`, followed by `os.RemoveAll` of the worktrees dir
- `cmd/codex_build.go:481` — `detectOrphanedWorktrees` guard; filters to the *current* phase
  and so never catches a same-phase crash

### The safety precedent to follow
- `cmd/recover_scanner.go:257` — `scanDirtyWorktrees`: the only existing uncommitted-changes
  check (`git status --porcelain`); note it explicitly skips `WorktreeOrphaned` entries
- `cmd/recover_repair.go:610` — `repairDirtyWorktree`: stashes rather than discards, and is
  classified destructive behind `--force` (`recover_repair.go:148`)

### Recorded history — this has happened before
- `.planning/WORKTREE-BRANCH-AUDIT-2026-07-27.md` — the audit of ten stranded branches; also
  documents that `git rev-list --count` overstates loss (diverged lineage, not lost commits)
- `.planning/codebase/CONCERNS.md` § "Interrupted Phase Builds Strand Worktree Branches" and
  § "Untracked Plan Files Lost on Worktree Merge-Back"
- `.planning/codebase/CONCERNS.md` § ".planning is Gitignored but Tracked" — new files under
  `.planning/` need `git add -f` or they silently vanish from commits

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `resolveTestCommand()` (`cmd/gate.go:244`) — already answers "how is this project tested",
  already handles Go, Node, Rust and Maven, already has tests. Criterion 2 is a wiring job,
  not new logic.
- `scanDirtyWorktrees` (`cmd/recover_scanner.go:257`) — an existing, tested
  `git status --porcelain` check that can be lifted or shared rather than reinvented.
- `repairDirtyWorktree` (`cmd/recover_repair.go:610`) — an existing stash-not-discard
  preservation routine; D-01's mechanism may be able to reuse it directly.
- `preserveWorktree` (`cmd/codex_build_worktree.go:411-435`) — already marks a worktree
  `Orphaned` instead of deleting on conflict. The intent exists; it is undone downstream.

### Established Patterns
- The repo's own Definition of Done requires a *runnable command that fails when the
  requirement is unmet*. Criterion 1's "fail-then-pass test" is written in that spirit and
  should be a real test, not a documented claim.
- Grep-ratchets are the established guard against a deleted anti-pattern returning
  (see Phase 191's criteria). A ratchet against `branch -D` and against literal
  `"go test ./..."` on the merge path fits the house style.
- Destructive operations elsewhere are gated behind an explicit `--force` and classified as
  destructive (`cmd/recover_repair.go:148`). D-01's named destruction command should match
  that existing convention rather than invent a new one.

### Integration Points
- Three production callers of `gcOrphanedWorktrees` (`continue`, `resume`, `init`) all change
  behaviour under D-01 — none may destroy implicitly any more.
- Two merge implementations exist and both hard-code the Go test command
  (`cmd/codex_build_worktree.go:1118` and `cmd/worktree.go:749`). Fixing one and not the other
  leaves the bug live on the path that is actually reached during a build.
- `worktree-orphan-scan` mutates state (`cmd/worktree.go:414`) by flipping entries to
  `Orphaned`, which currently *promotes them to deletion candidates*. Under D-01 that
  promotion must no longer imply destruction.

### Test-coverage reality (informs criterion 3)
- `gcOrphanedWorktrees` has **no test of its own**; its only two test-file appearances are
  fixture teardown with the result discarded.
- `mergePhaseWorktrees` is tested **only with an empty worktree list**
  (`TestMergePhaseWorktreesEmpty`) — the test gate, clash gate, checkout fallback and merge
  are entirely uncovered.
- `TestCleanupBuildWorktrees` deliberately points at a nonexistent path, so the destructive
  branch is never taken. No existing test puts a dirty or unmerged worktree in front of any
  deletion path.
- The merge-back CLI tests (`cmd/worktree_test.go:1134+`) *do* use real `git init` fixtures
  with a real `go.mod`, and are the model to follow for real-path tests.

</code_context>

<specifics>
## Specific Ideas

- The owner's framing of criterion 1's failure mode, in his words during this session: resuming
  to recover work must never be the thing that destroys it. The `resume` caller
  (`cmd/session_flow_cmds.go:261`) is the specific path he was shown and reacted to.
- The asymmetry worth closing explicitly: `recover` treats a dirty worktree as *critical,
  destructive-to-fix, confirmation-required*, while `gcOrphanedWorktrees` silently force-deletes
  the same worktree from `resume`. Two parts of one program disagree about how dangerous the
  same state is. That disagreement, not just the deletion, is the defect.
- `scanDirtyWorktrees` skips exactly the `WorktreeOrphaned` entries that GC will force-delete,
  so the one existing safety check never sees the cases that need it most.
- Preservation must survive the repo's own gitignore trap: anything written under `.planning/`
  needs `git add -f`, or the preservation commit silently omits it.

</specifics>

<deferred>
## Deferred Ideas

- **`WorktreeMerged` means two different things.** In the wave reconciler
  (`cmd/codex_build_worktree.go:446-452`) it means "files were copied to the root" and the
  branch is then force-deleted; in `mergePhaseWorktrees` it means an actual `git merge`. Worth
  unifying, but it is a semantics change to the execution model and belongs in its own phase.
- **Two duplicate merge-gate implementations.** `cmd/worktree.go:691` and
  `cmd/codex_build_worktree.go:1098` implement the same gates separately. Consolidating them is
  optional here — D-03 only requires both stop hard-coding the Go command. Full consolidation
  is a candidate for Phase 190 (Lean, Non-Duplicated Delivery).
- **`detectOrphanedWorktrees` current-phase blind spot** (`cmd/codex_build.go:1178-1180`) — it
  filters to the current phase, so a same-phase crash is never caught. In scope only insofar as
  criterion 1's resume test touches it; a full fix to the guard's selection logic is separate.
- **`BuildTimeout` of 120s caps both merge gates** (`cmd/timeouts.go:11`). A large non-Go test
  suite may exceed it once D-03 makes non-Go projects reachable. Flagged, not fixed here.

### Reviewed Todos (not folded)
- *"continue-finalize does not count --reconcile-task as recorded reconciliation for the
  implementation_evidence gate"* (score 0.6, `cmd/continue`) — matched on generic keywords
  (finalize, gate, phase). Concerns the evidence gate, not worktree safety. Not folded.
- *"ts-host preflight timeout hardcoded, not configurable, runs in repo cwd"* (score 0.6,
  `ts-host`) — matched on "repo"/"aether". A hard-coded timeout in the TypeScript host is a
  different subsystem from the worktree merge gate. Not folded, though it rhymes with the
  `BuildTimeout` note above.

</deferred>

---

*Phase: 187-Crash-Safe Worktrees & Ecosystem Neutrality*
*Context gathered: 2026-08-18*
