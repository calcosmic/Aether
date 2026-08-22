# Plan 160-05 — Summary

**Status:** Complete (3/3 tasks)
**Requirements:** LOUD-09 (decisions D-03 / D-04 / D-05)
**Completed:** 2026-07-27

> **Provenance note:** tasks 1–2 were executed by the assigned executor agent, which was
> then terminated by an account spend limit before writing task 3 (the tests). The
> orchestrator merged its implementation forward onto current main and completed task 3
> inline, including deliberate-regression proofs for every assertion.

## Why this plan existed

On 2026-07-27 three `/ant-plan` runs failed and the real cause was invisible — the evidence
either was never written or did not survive. Debug artifacts existed only for *parse
failure*; a timeout or a non-zero exit left nothing behind. The bar CONTEXT.md set: could an
operator diagnose the failure from the terminal plus the debug file alone, without hunting
session transcripts?

## What changed

**Task 1 — evidence on every failure mode** (`797c5715`, D-03)
`writeHostedWorkerOutputDebug` now has four call sites, not one: `timeout`, `non_zero_exit`,
`provider_error_envelope`, `parse_failure`. Each artifact carries `exit_code` (−1 where no
real exit status exists, rather than a fabricated 0), `duration_ms`, `provider_run_id`, and
`failure_mode`. All four route through the existing chokepoint, so `safeHostedWorkerArgs`
redaction and `sanitizeWorkerDiagnosticOutput` still apply — no new sanitisation path was
written, and none is bypassed.

**Task 2 — retention and data-clean** (`319c216c`, D-04)
`WorkerDebugRetentionMaxAge` (14 days) and `WorkerDebugRetentionMaxFiles` (50) enforced on
write via `pruneWorkerDebugArtifacts`, and wired into `aether data-clean` via
`pruneWorkerDebugDirectory`. Pruning is best-effort and never fails the write — and never
evicts the artifact just written, which is the operator's evidence for the failure they are
about to see.

**D-05 — worktree survival.** All call sites pass `workerTrackingRoot(config)`, not
`config.Root`, so artifacts land on the tracking root and survive `git worktree remove`. The
returned path is repo-relative so the `(debug: <path>)` hint resolves from the user's cwd.

**Task 3 — tests** (this commit)
`pkg/codex/worker_debug_artifact_test.go` (6 tests) and
`cmd/maintenance_worker_debug_test.go` (4 tests).

## A blind spot caught and closed

The first five content tests call `writeHostedWorkerOutputDebug` **directly**. They would all
still pass if the timeout and non-zero-exit call sites were deleted outright — the precise
regression D-03 exists to prevent, and the same defect found in plan 160-04's first draft.

`TestWorkerDebugCallSitesCoverEveryFailureModeAndUseTrackingRoot` closes it with a
source-level invariant: every failure mode must have a call site, and every call site must
resolve the tracking root.

## Deliberate-regression proofs

Each was re-introduced, observed, and reverted:

| Regression introduced | Test | Observed failure |
|---|---|---|
| Deleted the `timeout` call site | `TestWorkerDebugCallSites…` | `no writeHostedWorkerOutputDebug call site records FailureMode "timeout" — that failure would leave no evidence on disk (D-03)` |
| Call site writes to `config.Root` | `TestWorkerDebugCallSites…` | `call site does not pass workerTrackingRoot(config); artifacts written to the worktree root are destroyed by \`git worktree remove\` (D-05)` |
| `--dry-run` made to delete | `TestDataCleanWorkerDebugDryRunDoesNotMutate` | `dry run deleted files: 8 before, 3 after — an inspection command must not mutate state` |

That last one is the historical bug CLAUDE.md names by title: `consolidation-phase-end
--dry-run` and `consolidation-seal --dry-run` wrote to `instincts.json` for months while
their help text read *"Report without modifying."* It is now locked for `data-clean` too.

## Verification

```
go build ./cmd/aether                              → clean
go test ./pkg/codex -run 'WorkerDebug|Prune…'      → ok (6 tests)
go test ./cmd -run 'DataCleanWorkerDebug'          → ok (4 tests)
go test ./... -count=1                             → exit 0, 18 packages
```

## Files

- `pkg/codex/platform_dispatch.go` (modified — 3 new call sites, retention constants, prune)
- `pkg/codex/platform_dispatch_test.go` (modified — existing tests updated for new signature)
- `cmd/maintenance.go` (modified — `pruneWorkerDebugDirectory` + data-clean reporting)
- `pkg/codex/worker_debug_artifact_test.go` (created — 6 tests)
- `cmd/maintenance_worker_debug_test.go` (created — 4 tests)

## Commits

- `797c5715` feat(160-05): write debug artifacts on timeout and non-zero-exit paths
- `319c216c` feat(160-05): cap worker-debug retention and wire into data-clean
- (this commit) test(160-05): cover debug-artifact failure modes, retention, and dry-run safety
