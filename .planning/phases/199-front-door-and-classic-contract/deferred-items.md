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

## Legacy Next Up snapshots still encode the pre-199 lifecycle policy

- **Found during:** Plan 199-04 broader command-package verification (`go test ./cmd -count=1`)
- **Observed:** The package run reported 7,088 passing tests, 60 failures, and 10 skips. Most failures assert the policy this plan deliberately supersedes: discuss before plan, build preferred over run, targeted recover/build-force routes instead of resume, mandatory entomb after seal, or golden cards containing those older recommendations.
- **Isolation check:** The Plan 199-04 contract suite passes all 36 focused tests, and the existing read-only-loader and token-accounting guard tests pass. The package still compiles and reaches the later tests; the mismatches are expectation-level changes rather than a failure of the new projection path.
- **Why deferred:** Plans 199-09, 199-10, 199-26, 199-32, and 199-33 own the command-specific renderers, focused closeouts, retired recovery wording, and post-seal status migration. Updating those snapshots here would pre-empt their scoped migrations and hide the deliberate transition.
- **Follow-up:** Update each legacy expectation alongside its owning renderer migration, then run the exact full and race repository gates in Plan 199-29.
