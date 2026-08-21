# Test Coverage Matrix

Generated from actual test files in `cmd/*_test.go`. This matrix documents
coverage for the critical data-persistence paths targeted by Phase 147.

## Core Data-Persistence (Wave 1)

| Source File | Test File | Commands Tested | Coverage |
|-------------|-----------|-----------------|----------|
| `cmd/eventbus.go` | `cmd/eventbus_test.go` | event-bus-publish, event-bus-read | FULL |
| `cmd/queen.go` | `cmd/queen_test.go` | queen-compose, queen-verify, queen-promote, queen-to-md | FULL |
| `cmd/instinct.go` | `cmd/instinct_test.go` | instinct-create, instinct-list | FULL |
| `cmd/midden.go` | `cmd/midden_cmds_test.go` | midden-write, midden-recent, midden-review, midden-acknowledge | FULL |
| `cmd/hive.go` | `cmd/hive_test.go` | hive-init, hive-store, hive-read, hive-abstract | FULL |
| `cmd/spawn.go` | `cmd/spawn_test.go` | spawn-tree-load, spawn-tree-active, spawn-efficiency, validate-worker-response | FULL |

## Orchestration (Wave 2)

| Source File | Test File | Commands Tested | Coverage |
|-------------|-----------|-----------------|----------|
| `cmd/autopilot.go` | `cmd/autopilot_test.go` | autopilot-init, autopilot-update, autopilot-status, autopilot-stop (autopilot-check-replan retired 2026-08-16 — the run loop owns replan arithmetic, covered by TestRunAutopilotReplanDue) | FULL |
| `cmd/council.go` | `cmd/council_test.go` | council-deliberate, council-advocate, council-challenger, council-sage, council-history, council-budget-check | FULL |

## Flags & Shelf (Wave 3 — Extended)

| Source File | Test File | Commands Tested | Coverage |
|-------------|-----------|-----------------|----------|
| `cmd/flags.go` | `cmd/flags_test.go` | flag-list, flags (alias) | FULL |
| `cmd/flag_cmds.go` | `cmd/flags_test.go` + `cmd/flag_cmds_test.go` | flag-add, flag-resolve, flag-check-blockers, flag-acknowledge, flag-auto-resolve | FULL |
| `cmd/shelf_cmd.go` | `cmd/shelf_test.go` | shelf-list, shelf-add, shelf-promote, shelf-dismiss | FULL |
| `cmd/shelf_init.go` | `cmd/shelf_test.go` | shelf-promote-batch, shelf-dismiss-batch | FULL |
| `cmd/shelf_seal.go` | `cmd/shelf_test.go` | shelf-detect | FULL |

## Lifecycle Smoke Tests

The following public lifecycle commands already have dedicated test files
with extensive coverage (no additional smoke tests needed):

| Command | Test File(s) | Coverage Type |
|---------|-------------|---------------|
| `build` | `cmd/codex_build_test.go` | integration |
| `continue` | `cmd/codex_continue_test.go` | integration |
| `plan` | `cmd/codex_plan_test.go`, `cmd/codex_plan_finalize_test.go` | integration |
| `seal` | `cmd/codex_seal_test.go` | integration |
| `status` | `cmd/status.go` (setupTestStore tests) | unit + integration |
| `colonize` | `cmd/codex_colonize_test.go` | integration |

Commands with **no dedicated smoke test** but covered indirectly:
- `entomb` — covered via seal/colonize test paths
- `oracle` — covered in `cmd/oracle_loop_test.go` (the live loop),
  `cmd/oracle_brief_test.go` (the scoping ritual), `cmd/oracle_progress_test.go`
  (round logging, `--follow`, `selftest`, `recover`),
  `cmd/oracle_loop_quality_test.go` and `cmd/oracle_promote_test.go`. Runnable
  smoke path: `aether oracle selftest`, which runs one real research round and
  exits non-zero if the dispatcher, agent definition, artifacts or progress log
  are broken. Note `cmd/oracle_iterate_cmd_test.go` covers the *orphaned*
  `aether oracle-iterate` path against `.aether/data/oracle/`, not the live
  loop, which uses `.aether/oracle/`.
- `swarm` — no dedicated test (parallel bug investigation, tested via builder/watcher paths)

## Test Count Summary

| Wave | New Tests | Files | Status |
|------|-----------|-------|--------|
| Wave 1 | 120 | 4 | PASS |
| Wave 2 | 12 | 2 | PASS |
| Wave 3 | 9 | 2 | PASS |
| **Total** | **141+** | **8** | **ALL GREEN** |

Run verification:
```bash
go test ./cmd/... -count=1
```

Last updated: 2026-05-21
