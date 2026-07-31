# Deferred Items — Phase 163.1

## Plan 07

### Pre-existing, out-of-scope test failure

- **Test:** `TestCodexReadOnlyProfileSelectsReadOnlySandbox` in `pkg/codex/permission_profile_test.go`
- **Observed during:** `go test ./... -race` run for plan 163.1-07 verification
- **Failure:** `worker startup failed: codex login status failed: timed out; sensitive details omitted`
- **Why out of scope:** This plan's `files_modified` are `cmd/publish_cmd.go`, `cmd/install_cmd.go`,
  `cmd/release_pipeline_test.go`, and `cmd/e2e_publish_version_test.go` — all in the hub-publish
  version-resolution path. `pkg/codex/permission_profile_test.go` is unrelated: it invokes the real
  `codex` CLI binary's login status check, which times out in this sandboxed execution environment
  (no interactive auth / network access to the codex CLI's login flow). Not touched by, and not
  caused by, this plan's changes.
- **Action:** Not fixed. Logged here per the executor's scope-boundary rule rather than fixed inline.
  `go test ./cmd/... -count=1` (the package this plan actually touches) passed cleanly, and all other
  `pkg/...` packages passed under `-race`.
