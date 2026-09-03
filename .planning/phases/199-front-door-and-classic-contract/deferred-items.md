# Deferred Items

## `TestBranchDispositionRecordsAllThreeBranches` expects a pre-archive path

- **Found during:** Plan 199-01 overall verification (`go test ./...`)
- **Observed:** The suite reported 8,567 passing tests, one failure, and 11 skips because the test expects `.planning/phases/196-see-what-it-cost/196-BRANCH-DISPOSITION.md`.
- **Current artifact:** `.planning/milestones/v1.27-phases/196-see-what-it-cost/196-BRANCH-DISPOSITION.md`
- **Isolation check:** The failing test reproduces on its own and does not exercise the cleanup ownership changes from this plan.
- **Why deferred:** This is pre-existing archive-path maintenance outside Plan 199-01's test-cleanup scope.
- **Follow-up:** Make the branch-disposition test archive-aware, or update its fixture path in a dedicated maintenance change.

## `TestColonyPrimeMdDeletionProducesByteIdenticalOutput` is suite-order sensitive

- **Found during:** Plan 199-03 overall verification (`go test ./...` with the known archived-path test excluded)
- **Observed:** The aggregate normal suite reported 8,636 passing tests, one failure, and 11 skips because colony-prime output differed before versus after deleting a generated `COLONY_PRIME.md`.
- **Isolation check:** The exact failing test passed immediately on its own; Plan 199-03 changes only additive `pkg/colony` lifecycle contracts and optional JSON fields.
- **Race check:** The full race suite passed 8,636 tests across 20 packages when this suite-order-sensitive test and the existing archived-path fixture were excluded.
- **Why deferred:** The failure is an unrelated command-test interaction and is not reproducible in isolation, so changing command behavior here would exceed Plan 199-03's lifecycle-contract scope.
- **Follow-up:** Audit shared process/environment or generated-file state in the command test corpus under a dedicated test-isolation change.
