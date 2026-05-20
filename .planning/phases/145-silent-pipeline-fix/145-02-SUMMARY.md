---
phase: 145-silent-pipeline-fix
plan: 02
status: complete
started: 2026-05-20T22:51:00Z
completed: 2026-05-20T23:05:00Z
---

# Plan 145-02: Session Cleanup + Event Cap — Summary

## Status

Complete. All tasks verified.

## What Was Built

### Task 1: Session.json cleanup on init
`cmd/init_cmd.go` already contained the session cleanup at line 121:
```go
// Clear stale session from any prior colony to prevent old decisions from leaking in.
_ = os.Remove(filepath.Join(dataDir, "session.json"))
```

### Task 2: Event array cap in storage layer
`pkg/storage/storage.go` already contained `capEventsArray` function (lines 162-194):
- Caps `events` array at 100 entries for `COLONY_STATE.json` only
- Uses generic `map[string]interface{}` to avoid importing `pkg/colony`
- Logs warning to stderr when trimming: `"warning: event array capped at 100 (dropped N old events)"`
- Keeps the last 100 events (FIFO eviction)

### Task 3: Tests
Existing tests cover both behaviors:
- `cmd/init_cmd_test.go:TestInitCmd_ReplaceSessionJSON` — verifies stale session is replaced
- `cmd/init_cmd_test.go:TestInitCmd_SessionJSONNoErrorWhenMissing` — verifies init succeeds without existing session
- `pkg/storage/storage_malformed_test.go:TestSaveJSON_CapsEventsArray` — verifies 120 events → 100
- `pkg/storage/storage_malformed_test.go:TestSaveJSON_NoCapForOtherFiles` — verifies no cap on other files
- `pkg/storage/storage_malformed_test.go:TestSaveJSON_NoCapWhenUnderLimit` — verifies no cap when under 100

## Verification

```bash
go test ./cmd/... ./pkg/storage/... -v
# All tests pass
```

## Acceptance Criteria

- [x] `cmd/init_cmd.go` removes existing session.json before creating new one
- [x] `pkg/storage/storage.go` caps events at 100 for COLONY_STATE.json
- [x] Cap uses generic map approach (no circular import)
- [x] Warning logged when events dropped
- [x] All tests pass
