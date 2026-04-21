# Plan 24-01 Summary: Core Trace Logging Infrastructure

## Objective
Build the core trace logging infrastructure so every colony run leaves a durable, structured trace.

## What Was Done

### Task 1: Define TraceEntry and TraceLevel types
- **File:** `pkg/trace/trace.go`
- Defined `TraceLevel` string type with 8 constants: state, phase, pheromone, error, recovery, intervention, token, artifact.
- Defined `TraceEntry` struct with ID, RunID, Timestamp, Level, Topic, Payload, Source.
- Implemented `Tracer` wrapping `*storage.Store` with `Log()` calling `AppendJSONL("trace.jsonl", entry)`.
- Added convenience methods: `LogStateTransition`, `LogPhaseChange`, `LogError`, `LogPheromone`, `LogIntervention`.

### Task 2: Add RunID to ColonyState and generate on init/resume
- **Files:** `pkg/colony/colony.go`, `cmd/init_cmd.go`, `cmd/session_flow_cmds.go`
- Added `RunID *string` field to `ColonyState` (pointer, omitempty).
- `init` generates a `runID` (goal + timestamp + random suffix) and stores it in colony state.
- `resume-colony` generates a new `runID` for stale sessions (freshness check fails).
- `pause-colony` preserves the existing `runID`.

### Task 3: Hook state transitions into trace
- **Files:** `cmd/state_cmds.go`, `cmd/root.go`
- Added package-level `tracer *trace.Tracer` in `cmd/root.go`, initialized alongside `store`.
- After successful state mutation (`state-mutate --field state`), calls `tracer.LogStateTransition()`.
- No import cycle between `pkg/colony` and `pkg/trace` — trace is logged by caller only.

### Task 4: Hook phase changes into trace
- **Files:** `cmd/build_flow_cmds.go`, `cmd/autopilot.go`, `cmd/codex_build.go`
- `update-progress` logs phase status changes.
- `autopilot-update` logs autopilot phase transitions when colony state has a `run_id`.
- `codex_build.go` logs phase start, completion, and failure via `LogPhaseChange`.

### Task 5: Hook pheromone signals and errors into trace
- **Files:** `cmd/pheromone_write.go`, `cmd/error_cmds.go`
- `pheromone-write` logs pheromone signal creation with `LogPheromone`.
- `error-add` logs error records with `LogError`, including phase and severity.

### Task 6: Hook human interventions into trace
- **Files:** `cmd/hook_cmds.go`, `cmd/discuss.go`, `cmd/session_flow_cmds.go`
- `hook-pre-tool-use` logs blocked and redirect interventions.
- `hook-stop` logs stop-hook blocks.
- `discuss --resolve` logs discussion resolutions.
- `resume-colony` logs stale spawn clearing with new `run_id`.

### Task 7: Add trace-replay and trace-export CLI commands
- **File:** `cmd/trace_cmds.go`
- `trace-replay --run-id <id> [--level <levels>] [--since <RFC3339>] [--limit N]` returns JSON array of entries.
- `trace-export --run-id <id> [--output <path>]` writes filtered entries to file or stdout.
- Both commands registered in `cmd/root.go`.

### Task 8: Add tests for Tracer and trace hooks
- **Files:** `pkg/trace/trace_test.go`, `cmd/trace_cmds_test.go`
- `pkg/trace/trace_test.go` tests `Log` appends valid JSON, convenience methods produce correct shapes, and nil store returns error without panic.
- `cmd/trace_cmds_test.go` tests `trace-replay` filtering by run_id and level, `trace-export` writes valid JSON, and end-to-end tests verify trace entries after state mutation and stale session resume.

## Commits
1. `552edc01` — Hook state transitions into trace logging
2. `2d00e55d` — Hook phase changes into trace logging
3. `0245b068` — Hook pheromone signals and errors into trace logging
4. `3e95df7e` — Hook human interventions into trace logging
5. `f2ef61ef` — Add trace-replay and trace-export CLI commands
6. `9102bb25` — Add tests for trace commands and end-to-end hooks

## Test Results
- `go test ./pkg/trace/...` — PASS
- `go test ./cmd/... -run "TestTrace.*"` — PASS (5 tests)
- `go test ./...` — PASS (all packages)

## Files Modified
- `pkg/trace/trace.go` (created)
- `pkg/trace/trace_test.go` (created)
- `pkg/colony/colony.go`
- `cmd/root.go`
- `cmd/init_cmd.go`
- `cmd/session_flow_cmds.go`
- `cmd/state_cmds.go`
- `cmd/build_flow_cmds.go`
- `cmd/autopilot.go`
- `cmd/codex_build.go`
- `cmd/pheromone_write.go`
- `cmd/error_cmds.go`
- `cmd/hook_cmds.go`
- `cmd/discuss.go`
- `cmd/trace_cmds.go` (created)
- `cmd/trace_cmds_test.go` (created)
