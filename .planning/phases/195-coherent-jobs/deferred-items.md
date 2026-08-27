# Deferred Items

## Full-suite black-box timeout

- **Found during:** Plan 195-01 overall verification
- **Command:** `go test ./...`
- **Observed:** 1,520 tests passed and 1 skipped before `cmd` reached Go's 10-minute package timeout. The last started `cmd` test in the JSON log was `TestCLIErrorEnvelopeExitsNonZero` in `cmd/blackbox_harness_test.go`.
- **Why deferred:** The black-box harness and CLI error-envelope path were not changed by Plan 195-01. All 47 coherent-job tests pass normally and under `-race`; `go build ./cmd/aether` and `go vet ./...` also pass.
- **Follow-up:** Investigate why the black-box harness child process does not exit in the full-suite environment before relying on `go test ./...` as a bounded gate.
