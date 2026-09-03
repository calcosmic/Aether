# Deferred Items

## `TestBranchDispositionRecordsAllThreeBranches` expects a pre-archive path

- **Found during:** Plan 199-01 overall verification (`go test ./...`)
- **Observed:** The suite reported 8,567 passing tests, one failure, and 11 skips because the test expects `.planning/phases/196-see-what-it-cost/196-BRANCH-DISPOSITION.md`.
- **Current artifact:** `.planning/milestones/v1.27-phases/196-see-what-it-cost/196-BRANCH-DISPOSITION.md`
- **Isolation check:** The failing test reproduces on its own and does not exercise the cleanup ownership changes from this plan.
- **Why deferred:** This is pre-existing archive-path maintenance outside Plan 199-01's test-cleanup scope.
- **Follow-up:** Make the branch-disposition test archive-aware, or update its fixture path in a dedicated maintenance change.
