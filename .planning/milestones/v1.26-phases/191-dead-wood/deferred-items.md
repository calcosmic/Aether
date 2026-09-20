# Deferred Items — Phase 191, Plan 05

Discovered during execution of 191-05 (delete dead trace/pool code and the 8 unused
skill-lifecycle commands). Logged per the executor's scope-boundary rule: found in a file
this plan touches, but not caused by this plan's changes and not part of SKILL-01's
reviewed 8-command set — not fixed, not silently dropped.

## 1. `command_catalog.json`: 4 pre-existing entries with no classification

`scripts/verify_catalog_classified.py` reports 4 commands missing `classification` /
`since_version` / `historical_presence`: `phase-commits`, `research`, `spawn-orphans`,
`worktree-reap`. Confirmed via `git show HEAD:cmd/testdata/command_catalog.json` that all
four already lacked this data before any 191-05 edit — pre-existing, unrelated to the
skill-lifecycle deletion. `cmd/audit_catalog_test.go`'s Go-level catalog tests
(`TestAuditCatalogGolden`, `TestCatalogCompleteness`, `TestCatalogSchema`,
`TestAuditCatalogClassificationCoverage`) do not exercise this Python script and all pass;
the Python script is not part of this plan's `go test` proof requirement. Candidate for a
future, separate ruling.

## 2. `cmd/skills.go`: `skillMatchesWorkspace` has zero callers anywhere

`skillMatchesWorkspace(root string, entry skillIndexEntry) bool` (calls
`skillWorkspaceMatchReasons` internally) has no caller anywhere in `cmd/` or
`cmd/*_test.go` — confirmed by grep. It is not called by any of the 8 SKILL-01 commands
deleted in this plan (so its deadness is not a byproduct of this plan's own edits), and it
is not named by 191-CONTEXT.md's SKILL-01 scope. Left untouched per the conservative
default ("when in doubt, keep a file/function and log it"). Candidate for a future ruling.
