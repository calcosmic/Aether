# Deferred Items — Phase 197

Out-of-scope discoveries logged during plan execution. Not fixed, per the
executor's scope boundary (only fix issues directly caused by the current
task's changes).

## From 197-05

- **RESOLVED 2026-08-29 (wave 5 gate).** It was not pre-existing: 197-05's new
  `TestCanonicalAliasDelegates` ran the real `pause` command, which repointed
  the package-level `store` at its temp root and never restored it; the
  heartbeat test then inherited a store on a deleted folder. Reproduced
  deterministically with just the two tests together; fixed by `saveGlobals(t)`
  in the leaking subtest. Original note kept below for the record.
- **`TestBuildDispatchStartsHeartbeatMonitor` (cmd/codex_build_test.go) is
  test-order-dependent, not related to this plan's changes.** It passes
  reliably in isolation and alongside its neighbors (`-count=3` all green),
  but fails consistently (twice in a row) when the full `go test ./cmd`
  suite runs (~5000+ tests, ~270s). None of the files this plan's tasks
  touched (session_flow_cmds.go, platform_sync.go, update_cmd.go,
  source_check.go, command_guide.go, wrapper_command_names.go, hook_cmds.go,
  parity_test.go, command_guide_test.go, command_call_audit_test.go, and the
  testdata goldens) have any relationship to worker heartbeat monitoring or
  codex build dispatch. Pre-existing full-suite flakiness, most likely
  timing/resource-contention under parallel test load. Not investigated or
  fixed here — out of this plan's scope.
