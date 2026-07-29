# Deferred Items — Phase 163

Out-of-scope discoveries logged during plan execution, per the executor's
scope-boundary rule (only auto-fix issues directly caused by the current
task's changes).

## From 163-05 (suggest-analyze wired into build finalize)

- **`TestCodexReadOnlyProfileSelectsReadOnlySandbox` (`pkg/codex/permission_profile_test.go:83`) fails in this environment.** Failure: `worker startup failed: codex login status failed: timed out; sensitive details omitted`. This is a network/auth-dependent test unrelated to 163-05's changes — `git diff` against the plan's base commit shows zero changes under `pkg/codex/`. Confirmed pre-existing by running `go test ./pkg/... -count=1` and `go test ./... -count=1`, both showing the same isolated failure in `pkg/codex` while every other package (including `cmd`, which contains all of 163-05's changes) passes cleanly. Not fixed — out of scope for this plan.
