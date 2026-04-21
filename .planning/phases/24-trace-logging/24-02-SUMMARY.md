# Plan 24-02 Summary: Token/Cost Tracking, Artifact Tracing, Summary/Inspect, and Rotation

## Objective
Complete the trace logging system on top of Plan 24-01's core infrastructure: token/cost tracking, worker artifact tracing, trace replay/summary/export for remote debugging, and trace file rotation.

## What Was Done

### Task 1: Add token and cost tracing to agent pool
- **Files:** `pkg/trace/cost.go`, `pkg/agent/pool.go`, `pkg/trace/trace.go`
- Created `pkg/trace/cost.go` with `CalculateCost(model, inputTokens, outputTokens)` using model-specific USD rates per 1K tokens.
- Included rates for: claude-sonnet, claude-opus, claude-haiku, gpt-4, gpt-4-turbo, gpt-3.5-turbo.
- Added `LogTokenUsage` convenience method to `Tracer` with payload: model, input_tokens, output_tokens, usd_cost.
- Wired tracer into agent pool via new `WithTracer(tr *trace.Tracer, runID string)` PoolOption.
- In `poolStreamHandler.OnComplete`, token usage is logged to trace when tracer and runID are set.

### Task 2: Add artifact tracing to build and continue flows
- **Files:** `cmd/codex_build.go`, `cmd/codex_continue.go`, `cmd/codex_build_worktree.go`
- In `runCodexBuild`, after build completes, logs `build.worker` artifact entries for each dispatch with worker name, status, files modified count, and summary.
- In `runCodexContinue`, logs `continue.verification`, `continue.assessment`, and `continue.gates` artifact entries with phase, pass/fail status, counts, and operational issues.
- In `codex_build_worktree.go`, after successful pheromone sync during merge-back, logs `worktree.merge` artifact entries with worker name, files synced, and pheromone summary.

### Task 3: Add trace-summary and trace-inspect commands for remote debugging
- **Files:** `cmd/trace_cmds.go`
- Added `trace-summary --run-id <id>`: produces JSON with run duration, state transition count/list, phase count/list, error count/severities, total token usage (input/output/cost), and intervention count/types.
- Added `trace-inspect --run-id <id> --focus <level>`: shows a focused timeline of just that level with human-readable suggestions (e.g., "3 errors during phase 2", total cost for token focus).
- Both commands output JSON for programmatic consumption.

### Task 4: Add trace file rotation
- **Files:** `pkg/trace/rotate.go`, `cmd/trace_cmds.go`, `cmd/init_cmd.go`, `cmd/session_flow_cmds.go`
- Created `pkg/trace/rotate.go` with `RotateTraceFile(store, maxSizeMB)`:
  - Checks `trace.jsonl` size; if over limit, renames to `trace.YYYY-MM-DD-HHMMSS.jsonl` and creates new empty file.
  - Default maxSizeMB: 50.
- Added `trace-rotate --max-size-mb` CLI command for manual rotation.
- Hooked rotation into `init` and `resume-colony` commands before generating new run_id.

### Task 5: Add end-to-end trace coverage tests
- **Files:** `cmd/trace_cmds_test.go`
- `TestTraceSummaryAndInspect`: seeds a rich trace with state, phase, token, error, artifact, and intervention entries; verifies summary aggregation, inspect focus filtering, and replay level filtering.
- `TestTraceRotateCommand`: writes a 2MB trace file, triggers rotation with `--max-size-mb 1`, verifies old trace is preserved with timestamp suffix and new trace.jsonl exists.
- `TestTraceRotateNoOpWhenUnderLimit`: verifies no rotation when file is under the limit.

## Commits
1. `89de5692` — Add token and cost tracing to agent pool
2. `961d41ae` — Add artifact tracing to build, continue, and worktree flows
3. `205776d3` — Add trace-summary and trace-inspect commands for remote debugging
4. `a3117b1a` — Add trace file rotation and trace-rotate command
5. `97a2cbc3` — Add end-to-end trace coverage tests

## Test Results
- `go test ./pkg/trace/...` — PASS
- `go test ./cmd/... -run "TestTrace.*"` — PASS (8 tests)
- `go test ./...` — PASS (all packages)

## Files Modified
- `pkg/trace/cost.go` (created)
- `pkg/trace/rotate.go` (created)
- `pkg/trace/trace.go`
- `pkg/agent/pool.go`
- `cmd/codex_build.go`
- `cmd/codex_continue.go`
- `cmd/codex_build_worktree.go`
- `cmd/trace_cmds.go`
- `cmd/trace_cmds_test.go`
- `cmd/init_cmd.go`
- `cmd/session_flow_cmds.go`
