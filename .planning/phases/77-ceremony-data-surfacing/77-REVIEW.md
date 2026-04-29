---
phase: 77-ceremony-data-surfacing
reviewed: 2026-04-29T00:00:00Z
depth: standard
files_reviewed: 7
files_reviewed_list:
  - cmd/init_ceremony.go
  - cmd/codex_visuals.go
  - cmd/circuit_breaker.go
  - cmd/codex_build_worktree.go
  - cmd/codex_workflow_cmds.go
  - cmd/circuit_breaker_event_test.go
  - cmd/init_ceremony_research_test.go
findings:
  critical: 1
  warning: 3
  info: 3
  total: 7
status: issues_found
---

# Phase 77: Code Review Report

**Reviewed:** 2026-04-29T00:00:00Z
**Depth:** standard
**Files Reviewed:** 7
**Status:** issues_found

## Summary

Phase 77 wires three disconnected data paths: research data display in the init ceremony, circuit breaker events through the ceremony event bus, and a `--no-suggest` flag on the build command. The init ceremony and visual rendering changes are clean and well-tested. The circuit breaker refactor introduces a data race when reading `cb.failures` from a goroutine context, and has inconsistent indentation at two call sites. The `--no-suggest` flag is a documentation-only addition with no Go runtime effect.

## Critical Issues

### CR-01: Data race in `emitCircuitBreakerTripped` -- unprotected map read from goroutines

**File:** `cmd/circuit_breaker.go:119`
**Issue:** `emitCircuitBreakerTripped` reads `cb.failures[workerName]` and `cb.threshold` without acquiring `cb.mu`. In `dispatchCodexBuildWorkers` (line 176 of `codex_build_worktree.go`), workers run in goroutines via `go func(i int, dispatch codex.WorkerDispatch)`. Multiple goroutines can call `RecordFailure` (which holds the lock) and then call `emitCircuitBreakerTripped` (which does not) concurrently. Reading `cb.failures[workerName]` -- a map -- without the mutex is a data race in Go.

While `cb.threshold` is never mutated after construction and is safe to read concurrently, `cb.failures` is a `map[string]int` that is mutated under the lock in `RecordFailure`, `RecordSuccess`, and `Reset`. Concurrent reads without synchronization are undefined behavior.

The race is currently not caught by the existing tests because the unit test for `emitCircuitBreakerTripped` runs from a single goroutine. The `TestCircuitBreaker_ConcurrentAccess` test does not exercise `emitCircuitBreakerTripped`.

**Fix:**
```go
func (cb *CircuitBreaker) emitCircuitBreakerTripped(phase colony.Phase, wave int, workerName string) {
	cb.mu.Lock()
	count := cb.failures[workerName]
	threshold := cb.threshold
	cb.mu.Unlock()

	emitBuildCeremonyCircuitBreak(phase, wave, CircuitBreakerEvent{
		WorkerName: workerName,
		Event:      "tripped",
		Reason:     fmt.Sprintf("after %d consecutive failures (threshold: %d)", count, threshold),
	})
}
```

## Warnings

### WR-01: Inconsistent indentation at circuit breaker trip call site (worktree path)

**File:** `cmd/codex_build_worktree.go:326`
**Issue:** The closing brace on line 326 uses 5 tabs while the `else if` on line 324 also uses 5 tabs and the body uses 6 tabs. This produces visually confusing nesting. The code compiles and is functionally correct (Go uses braces, not indentation), but the mismatched indentation makes the control flow harder to verify at a glance.

```go
				} else if cb.RecordFailure(dispatch.WorkerName) {
					cb.emitCircuitBreakerTripped(phase, wave, dispatch.WorkerName)
					}     // <-- 5 tabs, but opens at 5 tabs, body at 6 tabs
				statusErr := ...
```

**Fix:** Align the closing brace to match the opening `else if`:
```go
				} else if cb.RecordFailure(dispatch.WorkerName) {
					cb.emitCircuitBreakerTripped(phase, wave, dispatch.WorkerName)
				}
```

### WR-02: Inconsistent indentation at circuit breaker trip call site (serial path)

**File:** `cmd/codex_build_worktree.go:437-439`
**Issue:** Same indentation inconsistency as WR-01 in the serial (`InRepo`) dispatch path. The `emitCircuitBreakerTripped` call uses 5 tabs while the `else if` body on the previous call site at line 438 uses 4 tabs for the opening and 5 tabs for the body. The closing brace at line 439 uses 4 tabs.

**Fix:** Ensure consistent indentation matching the surrounding `if`/`else if` block:
```go
			} else if cb.RecordFailure(dispatch.WorkerName) {
				cb.emitCircuitBreakerTripped(phase, wave, dispatch.WorkerName)
			}
```

### WR-03: Dead conditional branch in `runCeremonyResearch`

**File:** `cmd/init_ceremony.go:202-209`
**Issue:** The `if origStdout != nil` / `else` branches are identical -- both create a new `bytes.Buffer` and assign it to `stdout`. The `else` branch will execute `bytes.NewBuffer(nil); stdout = buf` which is exactly the same as the `if` branch. The comment "If stdout is already a buffer, wrap it" suggests different behavior was intended, but no wrapping occurs.

```go
		if origStdout != nil {
			buf = bytes.NewBuffer(nil)
			stdout = buf
		} else {
			buf = bytes.NewBuffer(nil)
			stdout = buf
		}
```

**Fix:** Remove the dead conditional:
```go
		buf := bytes.NewBuffer(nil)
		stdout = buf
```

## Info

### IN-01: Custom `testContains`/`testSearchString` reinvents `strings.Contains`

**File:** `cmd/init_ceremony_research_test.go:82-92`
**Issue:** `testContains` and `testSearchString` are manual re-implementations of `strings.Contains` from the standard library. The codebase already uses `strings.Contains` extensively in other test files (over 30 occurrences in `cmd/` alone). The custom implementation adds unnecessary code and diverges from project conventions.

**Fix:** Replace usages with `strings.Contains`:
```go
func TestRenderResearchDisplayProducesOutputWithAllSections(t *testing.T) {
	// ...
	for _, want := range expected {
		if !strings.Contains(output, want) {
			t.Errorf("renderResearchDisplay output missing %q", want)
		}
	}
}
```

### IN-02: `--no-suggest` flag is registered but never read in Go code

**File:** `cmd/codex_workflow_cmds.go:978`
**Issue:** The `--no-suggest` flag is registered on `buildCmd` but never read via `cmd.Flags().GetBool("no-suggest")` anywhere in the Go codebase. The plan notes this is intentional -- the flag is consumed by wrapper playbooks (build-context.md Step 4.2) which parse CLI args textually. However, the flag registration in Go means `aether build --no-suggest` will be accepted silently with no effect. This is documented behavior but could confuse users who expect the Go runtime to honor it.

**Fix:** Either:
- Add a comment on the flag registration line explaining it exists for playbook consumption only, or
- Wire the flag into the build command's RunE to actually skip suggest-analyze when set.

### IN-03: Silent error swallowing in `extractCeremonyResearchData`

**File:** `cmd/init_ceremony.go:282, 288, 294, 300`
**Issue:** Four `_ = json.Unmarshal(b, &data.*)` calls silently discard deserialization errors. While this is acceptable for display-only cosmetic data (the function returns an empty struct on failure and `renderResearchDisplay` handles that gracefully), there is no logging or debugging hook for when extraction fails. If a future developer introduces a struct field mismatch, the failure will be completely invisible.

**Fix:** Consider logging at debug level when unmarshal fails, or add a comment explicitly documenting the intentional silent-swallow design:
```go
// Intentionally silent: research data is display-only. If extraction fails,
// the zero-value struct will cause renderResearchDisplay to return "".
_ = json.Unmarshal(b, &data.TechStackDetail)
```

---

_Reviewed: 2026-04-29T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
