---
phase: 82-loop-detection-telemetry
reviewed: 2026-04-30T00:00:00Z
depth: standard
files_reviewed: 10
files_reviewed_list:
  - cmd/ceremony_emitter.go
  - cmd/circuit_breaker.go
  - cmd/codex_continue.go
  - cmd/codex_plan.go
  - cmd/loop_break_emission_test.go
  - cmd/loop_break_event_test.go
  - cmd/loop_safety_status_test.go
  - cmd/recovery_engine.go
  - cmd/status.go
  - pkg/events/ceremony.go
findings:
  critical: 1
  warning: 3
  info: 2
  total: 6
status: issues_found
---

# Phase 82: Code Review Report

**Reviewed:** 2026-04-30
**Depth:** standard
**Files Reviewed:** 10
**Status:** issues_found

## Summary

Reviewed 10 files implementing loop detection telemetry: a ceremony event system for emitting loop-break events at five detection points (watcher skip, circuit breaker trip, cycle detection, recovery redirect, lifecycle recovery), a circuit breaker for per-worker failure tracking, a recovery menu engine, and a Loop Safety section in `/ant-status`. The implementation is structurally sound with good separation of concerns, but contains one logic bug where the status dashboard shows stale events instead of recent ones, and several moderate-quality concerns around error handling and resource management.

## Critical Issues

### CR-01: Loop Safety section shows oldest events instead of newest events

**File:** `cmd/status.go:148-163`
**Issue:** `loadRecentLoopBreakEvents` calls `bus.Query(topic, since, limit)` which reads the JSONL file top-to-bottom and breaks when `limit` is reached. Since JSONL files are append-only (chronological), this returns the **oldest** N matching events. The code then reverses the slice for "newest-first" display order, but it is still only the oldest N events -- any recent events beyond the limit are silently dropped.

When a user has more than 5 loop-break events in the past 7 days, the Loop Safety table will display the first 5 ever recorded (possibly from days ago) rather than the 5 most recent ones. This defeats the purpose of the dashboard section, which is to show current/recent loop activity.

**Fix:**
Replace `Query` with `Replay` (which collects all matches, sorts, then slices), or reverse-iterate the JSONL file. The simplest fix:

```go
func loadRecentLoopBreakEvents(s *storage.Store) []events.Event {
    if s == nil {
        return nil
    }
    bus := events.NewBus(s, events.DefaultConfig())
    since := time.Now().AddDate(0, 0, -7)
    evts, err := bus.Replay(context.Background(), events.CeremonyTopicLoopBreak, since, 5)
    if err != nil || len(evts) == 0 {
        return nil
    }
    // Replay returns oldest-first; reverse for newest-first display.
    for i, j := 0, len(evts)-1; i < j; i, j = i+1, j-1 {
        evts[i], evts[j] = evts[j], evts[i]
    }
    return evts
}
```

Alternatively, if `Replay` is also undesirable due to reading the full file, add a `QueryNewest` method to the Bus that reads lines in reverse order.

## Warnings

### WR-01: `emitLifecycleCeremony` creates a new Bus instance on every invocation

**File:** `cmd/ceremony_emitter.go:128`
**Issue:** Every call to `emitLifecycleCeremony` creates a fresh `events.NewBus(store, events.DefaultConfig())`. This function is called in tight loops -- for example, `emitLifecycleCeremonySequence` calls it once per wave-start, once per spawn, and once per wave-end. In `emitPlanCeremonyDispatchSequence`, that is 3+ bus creations for 2 dispatches. In `emitContinueCeremonyFlowSequence`, it could be many more. Each `NewBus` call allocates a struct. While not a correctness bug, this is unnecessary allocation in hot paths.

**Fix:** Pass a shared `*events.Bus` as a parameter, or create a package-level lazy-init pattern. For example:

```go
func emitLifecycleCeremony(topic string, payload events.CeremonyPayload, source string) {
    if store == nil || strings.TrimSpace(topic) == "" {
        return
    }
    payload = trimCeremonyPayload(payload)
    raw, err := payload.RawMessage()
    if err != nil {
        return
    }
    bus := lifecycleBus() // lazy singleton
    _, _ = bus.Publish(context.Background(), topic, raw, source)
}
```

### WR-02: `json.Marshal` error silently discarded in `renderRecoveryMenu`

**File:** `cmd/recovery_engine.go:229`
**Issue:** `detailBytes, _ := json.Marshal(recoveryDetails)` discards the error. If marshaling fails (e.g., due to an unsupported type in `details`), the JSON output will contain `"details":null` instead of the intended structured data. The user sees no indication that the recovery details were lost. While `null` is valid JSON, the missing data means the machine-readable recovery envelope is incomplete.

**Fix:**
```go
detailBytes, err := json.Marshal(recoveryDetails)
if err != nil {
    detailBytes = []byte("{}")
    // Log or emit a warning; the error envelope is still valid JSON.
}
```

### WR-03: `trimCeremonyText` does not add "..." suffix for limit values 2 and 3

**File:** `cmd/ceremony_emitter.go:537-549`
**Issue:** When `limit` is 2 or 3, the function returns `value[:limit]` without the `"..."` truncation indicator. This means consumers that check for the `"..."` suffix (like `TestEmitLoopBreakEventTrimsPayload` at `cmd/loop_break_event_test.go:110`) could be confused. Currently the only call site uses `ceremonyTextLimit` (500), so this is not triggered in practice, but it is a latent inconsistency that will surface if the limit constant is ever lowered or if the function is reused with a small limit.

**Fix:**
```go
func trimCeremonyText(value string, limit int) string {
    value = strings.TrimSpace(value)
    if limit <= 0 || len(value) <= limit {
        return value
    }
    if limit <= 3 {
        // For very small limits, return raw truncation without suffix
        return value[:limit]
    }
    return strings.TrimSpace(value[:limit-3]) + "..."
}
```

This is already the behavior, but the comment should document the intentional design choice. Alternatively, unify to always add `"..."` when truncation occurs, even for small limits:

```go
func trimCeremonyText(value string, limit int) string {
    value = strings.TrimSpace(value)
    if limit <= 0 || len(value) <= limit {
        return value
    }
    suffix := ""
    if limit > 3 {
        suffix = "..."
        value = strings.TrimSpace(value[:limit-3])
    } else {
        value = value[:limit]
    }
    return value + suffix
}
```

## Info

### IN-01: Inconsistent indentation in `CeremonyTopics()` return slice

**File:** `pkg/events/ceremony.go:77`
**Issue:** `CeremonyTopicBuildCircuitBreak` has an extra tab of indentation compared to all other entries in the `CeremonyTopics()` slice. This is a cosmetic issue but indicates the line was added without matching the existing formatting.

**Fix:** Remove the extra indentation:
```go
        CeremonyTopicBuildWaveEnd,
        CeremonyTopicBuildCircuitBreak,  // was indented with extra tab
        CeremonyTopicPlanWaveStart,
```

### IN-02: Test `TestLoadRecentLoopBreakEventsQuery` uses `time.Sleep` for timestamp separation

**File:** `cmd/loop_safety_status_test.go:72`
**Issue:** The test uses `time.Sleep(time.Millisecond * 10)` in a loop of 7 iterations to ensure distinct timestamps. This adds ~70ms of unnecessary sleep to the test. The timestamps are passed as string parameters to `emitLoopBreakEvent`, so the test could use synthetic timestamps instead of sleeping.

**Fix:** Instead of relying on wall-clock time, generate timestamps explicitly:
```go
base := time.Now()
for i := 0; i < 7; i++ {
    ts := base.Add(time.Duration(i) * time.Minute).String()
    // Use ts in the detection signal or action to ensure ordering
}
```

---

_Reviewed: 2026-04-30_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
