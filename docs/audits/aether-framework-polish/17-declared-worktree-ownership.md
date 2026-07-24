# Declared Worktree Ownership And Atomic Wave Reconciliation

Date: 2026-07-24

Status: implemented and verified through the complete normal and race Go
suites, vet, native build, and Windows amd64 cross-build. No public release
was performed.

## Verdict

Worktree-mode parallel builds no longer accept the first worker to finish and
orphan the rest. Each task now declares the paths it owns before workers run;
two tasks in the same wave declaring the same path fail the build before any
worker starts; and each wave reconciles as one atomic decision — either every
accepted worker's output syncs in deterministic order, or the whole wave is
rejected and nothing reaches the root checkout.

For beginners: parallel workers used to race — whoever finished first got
their changes in, and the loser's work was silently set aside. Now every
worker announces up front which files it owns, double-booking is caught
before anyone starts, and each group of parallel workers is treated like a
single transaction: the whole group lands, or none of it does.

## Declared Ownership Contract

- `declaredPathsForTask` (`cmd/codex_build_worktree.go`) computes a task's
  owned paths from task-level criterion evidence artifacts and from hints
  that are exactly one repo-relative file path. Paths under `.aether/` never
  participate in ownership.
- Declarations flow `Task -> codexBuildDispatch.DeclaredPaths ->
  codex.WorkerDispatch.DeclaredPaths`, so they appear in the build manifest
  and the durable attempt record.
- `validateDeclaredWorktreeOwnership` runs before any worker dispatch in
  worktree mode. Two tasks in the same wave declaring the same path is a
  build-stopping error naming the wave, the path, and both tasks, with the
  remediation (disjoint declarations, separate waves, or in-repo mode).
- Declared overlap across waves is legal. Waves execute sequentially and a
  later wave's worktree inherits the earlier wave's synced output, so
  dependent tasks can legitimately advance the same file. This fixes the old
  registry's false conflict on sequential dependent tasks.

## Atomic Wave Reconciliation

`reconcileWorktreeWave` (`cmd/codex_build_worktree.go`) replaces the
per-finisher claim-and-sync. Worker goroutines only run the provider and
collect results; after the wave joins, one decision is made:

- A worker touching a path declared by a different same-wave task is an
  ownership violation.
- A path produced by more than one completed worker with no declared owner
  to arbitrate is an ownership violation.
- With no violations, every accepted worker syncs in deterministic dispatch
  order, followed by pheromone merge and terminal journaling.
- With any violation, nothing syncs. Violating workers fail with the exact
  paths and owners; conflict-free workers are marked `blocked` rather than
  silently accepted; every worktree in the wave is preserved as orphan
  evidence.
- Terminal results are journaled only after the decision, so the attempt
  journal always matches the reconciled outcome.

## Regression Evidence

`cmd/codex_build_worktree_test.go`:

| Invariant | Test |
| --- | --- |
| Two parallel workers producing one undeclared path reject the whole wave; root carries no partial output; both worktrees preserved; journal records reconciliation errors | `TestBuildWorktreeModeRejectsOverlappingUntrackedPaths` (rewritten) |
| Same-wave declared overlap fails before any worker invocation | `TestBuildWorktreeModeRejectsDeclaredOverlapBeforeDispatch` |
| Dependent tasks in different waves may advance the same declared file; later wave inherits earlier content; build reaches BUILT | `TestBuildWorktreeModeAllowsDeclaredOverlapAcrossWaves` |
| A worker touching another task's declared path fails with the declaring task named; the innocent worker is blocked; nothing syncs | `TestBuildWorktreeModeRejectsDeclaredPathViolation` |
| Single-worker isolation, sync-back, and worktree cleanup unchanged | `TestBuildWorktreeModeDispatchesIntoIsolatedRoots` |
| Pheromone merge on accepted sync unchanged | `TestBuildWorktreeModeMergesPheromoneChangesBackToRoot` |

## Verification

| Check | Result |
| --- | --- |
| `go test ./cmd -run 'TestBuildWorktreeMode' -count=1` | Pass (6 tests) |
| Worktree/reconcile/circuit-breaker/attempt/dispatch cluster | Pass |
| `go test ./pkg/... -count=1` and `-race -count=1` | Pass |
| `go test ./cmd -count=1` | Pass; 288.971s |
| `go test ./cmd -race -count=1` | Pass; 376.740s |
| `go vet ./...` | Pass |
| Native build and Windows amd64 cross-build | Pass |
| `git diff --check` | Pass |

## Remaining Limits

- Declarations cover evidence artifacts and single-path hints. A worker that
  writes a path nobody declared is still legal unless it collides with
  another worker in the same wave; collision then rejects the wave. Richer
  declaration (directories, generated bundles) remains future work.
- A mid-sync I/O failure during an accepted wave can still leave a partial
  sync; the wave-atomic guarantee covers ownership conflicts, not filesystem
  errors.
- Same-path sharing in one wave (e.g., two tasks legitimately appending to
  `go.mod`) is rejected; the supported answer is separate waves or in-repo
  mode. Content-level merge remains out of scope.
- The external wrapper path (`build-finalize` with `mergePhaseWorktrees`)
  keeps its own git-merge clash gates and is unchanged by this checkpoint.
- Duplicate execution of one task through circuit-breaker peer
  redistribution is treated as a conflict if both copies produce the same
  path, which is stricter than the old same-name exemption and intentional.

## Next Checkpoint

Redesign Hive knowledge around evidence IDs, stable repository identity,
contradiction, revocation, decay, locks, and opt-in retrieval before
enabling it (handoff item 3).
