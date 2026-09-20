# Phase 147: Critical Test Coverage - Research

**Researched:** 2026-05-21
**Domain:** Go testing patterns for Aether CLI data-persistence commands
**Confidence:** HIGH

## Summary

This phase adds test files for the 12 most critical untested source files in `cmd/`, covering event bus, queen, instinct, midden, hive, spawn, autopilot, flags, council, and shelf commands. The codebase already has 2,605+ tests in `cmd/` and robust test infrastructure (`newTestStore`, `saveGlobals`, `resetRootCmd`, `parseEnvelope`).

The primary gap is not testing technique but **dedicated test files for specific command files**. Many commands are tested indirectly through integration tests (e.g., `hive-promote` tested via seal ceremony, `midden-write` tested via security tests), but the dedicated source files lack their own `_test.go` files that prove every data-persistence path works.

**Primary recommendation:** Create 11 new `_test.go` files (flags already has one) using the established `newTestStore` + `rootCmd.Execute()` pattern, focusing on data-persistence paths: JSON loads, JSON saves, JSONL appends, file creation, and state mutations.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Event bus pub/sub | API / Backend | — | `events.Bus` handles persistence via `storage.Store.AppendJSONL` |
| Queen wisdom (QUEEN.md) | API / Backend | — | `queen.go` commands mutate hub and local markdown files |
| Instinct CRUD | API / Backend | — | `instinct.go` commands read/write `instincts.json` |
| Midden failure tracking | API / Backend | — | `midden_cmds.go` commands read/write `midden.json` |
| Hive wisdom | API / Backend | — | `hive.go` commands read/write `hive/wisdom.json` via `os` calls |
| Spawn tracking | API / Backend | — | `spawn.go` commands mutate `spawn-tree.txt` via `agent.SpawnTree` |
| Autopilot state | API / Backend | — | `autopilot.go` commands read/write `autopilot/state.json` |
| Flags / pending decisions | API / Backend | — | `flags.go` reads `pending-decisions.json` and `flags.json` |
| Council deliberation | API / Backend | — | `council.go` commands read/write `council/history.json` |
| Shelf entries | API / Backend | — | `shelf_cmd.go` commands read/write `shelf.json` |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go standard testing | 1.23 | Test framework | Built-in, no dependencies |
| `github.com/spf13/cobra` | v1.8+ | CLI command framework | Already used for all commands |
| `github.com/calcosmic/Aether/pkg/storage` | local | Atomic file operations | All data persistence goes through this |
| `github.com/calcosmic/Aether/pkg/events` | local | Event bus with JSONL persistence | Used by eventbus and spawn ceremony |
| `github.com/calcosmic/Aether/pkg/agent` | local | Spawn tree tracking | Used by spawn commands |
| `github.com/calcosmic/Aether/pkg/colony` | local | Domain types (MiddenFile, InstinctsFile, etc.) | Used by all data-persistence commands |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `bytes.Buffer` | stdlib | Capture stdout/stderr | Every CLI execution test |
| `testing` temp dirs | stdlib | Isolated filesystem per test | `t.TempDir()` or `newTestStore` |
| `json.Unmarshal` | stdlib | Parse command output envelopes | Every test that checks results |

**Installation:** No new dependencies required — all are already in `go.mod`.

**Version verification:** `go version go1.23.0` confirmed via `go test ./cmd/...`.

## Architecture Patterns

### System Architecture Diagram

```
Test File (cmd/*_test.go)
    |
    |-- newTestStore(t) --> temp dir + storage.Store
    |-- store = s
    |-- stdout = &buf
    |-- rootCmd.SetArgs(["command", "--flag", "value"])
    |-- rootCmd.Execute()
    |
    v
Command Handler (cmd/*.go)
    |
    |-- store.LoadJSON("file.json", &struct)  <-- read path
    |-- business logic / mutations
    |-- store.SaveJSON("file.json", struct)   <-- write path
    |-- outputOK(result) or outputError(...)
    |
    v
storage.Store (pkg/storage/storage.go)
    |
    |-- AtomicWrite / LoadJSON / AppendJSONL
    |-- FileLocker for cross-process safety
    |
    v
Filesystem (temp dir per test)
```

### Recommended Project Structure

Tests live alongside source files in `cmd/`:

```
cmd/
├── eventbus.go              <-- NEEDS eventbus_test.go
├── eventbus_subscribe_test.go   (exists: 5 tests, covers subscribe/stream only)
├── eventbus_timeout_test.go     (exists: 3 tests, covers timeout flags only)
├── queen.go                 <-- NEEDS queen_test.go
├── queen_local_test.go          (exists: 8 tests)
├── queen_global_test.go         (exists: 2 tests)
├── ... (many other queen_*_test.go files)
├── instinct.go              <-- NEEDS instinct_test.go
├── instinct_runtime_test.go     (exists: 2 tests, covers runtime loading only)
├── midden_cmds.go           <-- NEEDS midden_cmds_test.go
├── hive.go                  <-- NEEDS hive_test.go
├── hive_runtime_test.go         (exists: 2 tests, covers promoteToHive only)
├── spawn.go                 <-- NEEDS spawn_test.go
├── spawn_runs.go
├── spawn_track.go
├── write_cmds_test.go           (exists: 49 tests, covers spawn-log, spawn-complete, spawn-tree-depth, spawn-efficiency, spawn-can-spawn, validate-worker-response)
├── autopilot.go             <-- NEEDS autopilot_test.go
├── flags.go                 <-- HAS flags_test.go (6 tests)
├── council.go               <-- NEEDS council_test.go
├── shelf_cmd.go             <-- NEEDS shelf_cmd_test.go
├── shelf_test.go                (exists: 5 tests, covers shelf-add, shelf-promote, shelf-dismiss, shelf-list, shelf-filter)
```

### Pattern 1: CLI Execution Test (Standard)
**What:** Execute a Cobra command via `rootCmd.Execute()` and verify JSON output + filesystem state.
**When to use:** All command tests that mutate data.
**Example:**
```go
// Source: cmd/flags_test.go (verified in codebase)
func TestFlagsListJSON(t *testing.T) {
    saveGlobals(t)
    resetRootCmd(t)

    var buf bytes.Buffer
    stdout = &buf

    s, tmpDir := setupTestStore(t)
    defer os.RemoveAll(tmpDir)

    os.Setenv("AETHER_ROOT", tmpDir)
    defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

    store = s

    rootCmd.SetArgs([]string{"flag-list", "--json"})

    err := rootCmd.Execute()
    if err != nil {
        t.Fatalf("flag-list --json returned error: %v", err)
    }

    output := buf.String()
    var envelope map[string]interface{}
    if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &envelope); err != nil {
        t.Fatalf("output is not valid JSON: %v, got: %s", err, output)
    }
    if envelope["ok"] != true {
        t.Errorf("expected ok=true, got: %v", envelope["ok"])
    }
}
```

### Pattern 2: Direct Function Test
**What:** Call a helper function directly with a `storage.Store` and verify return values + side effects.
**When to use:** Pure helper functions that don't depend on Cobra flag parsing (e.g., `filterFlags`, `readShelfFile`, `writeShelfFile`).
**Example:**
```go
// Source: cmd/shelf_test.go (verified in codebase)
func TestShelfFileNotExist(t *testing.T) {
    tmpDir := t.TempDir()
    dataDir := tmpDir + "/.aether/data"
    os.MkdirAll(dataDir, 0755)
    s, _ := storage.NewStore(dataDir)

    sf, err := readShelfFile(s)
    if err != nil {
        t.Fatalf("readShelfFile returned error: %v", err)
    }
    if len(sf.Entries) != 0 {
        t.Fatalf("expected 0 entries, got %d", len(sf.Entries))
    }
}
```

### Pattern 3: Hub Store Test
**What:** Set `AETHER_HUB_DIR` environment variable to redirect hub-level operations (QUEEN.md, hive) to a temp directory.
**When to use:** Any command that uses `hubStore()` or `resolveHubPath()`.
**Example:**
```go
// Source: cmd/queen_local_test.go (verified in codebase)
hubDir := filepath.Join(tmpDir, "hub")
os.MkdirAll(hubDir, 0755)
origHub := os.Getenv("AETHER_HUB_DIR")
os.Setenv("AETHER_HUB_DIR", hubDir)
t.Cleanup(func() { os.Setenv("AETHER_HUB_DIR", origHub) })
```

### Anti-Patterns to Avoid
- **Don't use `t.TempDir()` without setting `AETHER_ROOT` or `COLONY_DATA_DIR`:** Commands depend on these env vars to resolve the store path.
- **Don't forget `saveGlobals(t)` and `resetRootCmd(t)`:** Package-level globals (`store`, `stdout`, flag filter vars) leak between tests.
- **Don't test Cobra flag parsing in isolation:** The project pattern is integration-style tests via `rootCmd.Execute()`.
- **Don't use real `~/.aether/` hub in tests:** Always redirect via `AETHER_HUB_DIR`.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Test store setup | Custom temp dir + manual file copy | `newTestStore(t)` or `setupTestStore(t)` | Already handles `COLONY_DATA_DIR`, fixture copying, and `storage.NewStore` |
| JSON output parsing | Manual string splitting | `parseEnvelope(t, output)` | Standardized JSON envelope parser used across 100+ tests |
| Global state cleanup | Manual defer per global | `saveGlobals(t)` | Captures and restores all mutable package globals automatically |
| Cobra flag reset | Manual flag value reset | `resetRootCmd(t)` | Recursively resets all local flags to defaults across the command tree |
| Test fixtures | Inline JSON in tests | `cmd/testdata/*.json` | Existing fixtures for flags, colony state, pheromones, etc. |

**Key insight:** The test infrastructure is mature. The work is applying existing patterns to uncovered files, not inventing new testing approaches.

## Runtime State Inventory

Not applicable — this is a greenfield test-creation phase. No runtime state renames or migrations.

## Common Pitfalls

### Pitfall 1: Missing `saveGlobals` Causes Cross-Test Pollution
**What goes wrong:** Test A sets `store = s` and `stdout = &buf`. Test B runs next and writes to the closed buffer or wrong store.
**Why it happens:** `cmd` package uses package-level globals for `store`, `stdout`, `stderr`, and filter variables.
**How to avoid:** Every test that mutates globals must call `saveGlobals(t)` as its first action.
**Warning signs:** Panics on closed buffers, "no store initialized" errors in tests that set a store, flaky test behavior.

### Pitfall 2: Cobra Flag Leakage Between Tests
**What goes wrong:** Test A runs `rootCmd.SetArgs([]string{"command", "--flag", "value"})`. Test B runs the same command without the flag, but the flag value from Test A persists.
**Why it happens:** Cobra local flags persist across `Execute()` calls unless explicitly reset.
**How to avoid:** Call `resetRootCmd(t)` before any test that calls `rootCmd.Execute()`.
**Warning signs:** Tests pass in isolation but fail when run as a suite; unexpected flag values in test output.

### Pitfall 3: Hub Operations Write to Real `~/.aether/`
**What goes wrong:** Tests for `queen-init`, `hive-init`, `hive-store` create or modify files in the user's actual hub directory.
**Why it happens:** `resolveHubPath()` calls `os.UserHomeDir()` unless `AETHER_HUB_DIR` is set.
**How to avoid:** Set `AETHER_HUB_DIR` to a temp directory in any test that triggers hub operations.
**Warning signs:** Tests leave artifacts in `~/.aether/`; tests behave differently on different machines.

### Pitfall 4: Event Bus Tests Are Timing-Sensitive
**What goes wrong:** Tests for `event-bus-subscribe --stream` fail intermittently because the goroutine that publishes events hasn't run yet.
**Why it happens:** Streaming uses a polling loop with `time.Ticker`.
**How to avoid:** Use short poll intervals (e.g., `10ms`) and `time.Sleep()` in the publisher goroutine, or use `--max-events 1` with a short `--timeout`.
**Warning signs:** Flaky passes/failures; "expected one NDJSON event line" errors.

### Pitfall 5: `outputError` Behavior Depends on `AETHER_OUTPUT_MODE`
**What goes wrong:** Tests expect JSON error output but get plain text because the env var isn't set.
**Why it happens:** `outputError` checks `AETHER_OUTPUT_MODE` to decide between JSON and visual output.
**How to avoid:** Call `forceJSONOutputModeForTest(t)` in tests that assert on error output format.
**Warning signs:** JSON unmarshal failures on error output; tests pass when run individually but not in suite (TestMain sets the env var, but some tests may override it).

## Code Examples

### Creating a Test for a New Command File

```go
package cmd

import (
    "bytes"
    "encoding/json"
    "os"
    "strings"
    "testing"

    "github.com/calcosmic/Aether/pkg/storage"
)

func TestCommandName_DataPersistencePath(t *testing.T) {
    saveGlobals(t)
    resetRootCmd(t)

    var buf bytes.Buffer
    stdout = &buf

    s, tmpDir := newTestStore(t)
    defer os.RemoveAll(tmpDir)
    store = s

    // For hub commands:
    // hubDir := filepath.Join(tmpDir, "hub")
    // os.MkdirAll(hubDir, 0755)
    // t.Setenv("AETHER_HUB_DIR", hubDir)

    rootCmd.SetArgs([]string{"command-name", "--required-flag", "value"})

    err := rootCmd.Execute()
    if err != nil {
        t.Fatalf("command-name returned error: %v", err)
    }

    output := strings.TrimSpace(buf.String())
    var envelope map[string]interface{}
    if err := json.Unmarshal([]byte(output), &envelope); err != nil {
        t.Fatalf("output is not valid JSON: %v, got: %s", err, output)
    }
    if envelope["ok"] != true {
        t.Fatalf("expected ok=true, got: %v", envelope["ok"])
    }

    // Verify filesystem state
    // var data SomeStruct
    // if err := s.LoadJSON("file.json", &data); err != nil { ... }
}
```

### Testing Hub-Level Commands

```go
// Source: cmd/queen_local_test.go pattern (verified)
hubDir := filepath.Join(tmpDir, "hub")
os.MkdirAll(hubDir, 0755)
origHub := os.Getenv("AETHER_HUB_DIR")
os.Setenv("AETHER_HUB_DIR", hubDir)
t.Cleanup(func() { os.Setenv("AETHER_HUB_DIR", origHub) })
```

### Testing Event Bus Streaming

```go
// Source: cmd/eventbus_subscribe_test.go pattern (verified)
go func() {
    time.Sleep(30 * time.Millisecond)
    bus := events.NewBus(s, events.DefaultConfig())
    payload, _ := (events.CeremonyPayload{...}).RawMessage()
    bus.Publish(context.Background(), events.CeremonyTopicBuildSpawn, payload, "unit-test")
}()

rootCmd.SetArgs([]string{
    "event-bus-subscribe", "--stream", "--filter", "ceremony.*",
    "--poll-interval", "10ms", "--timeout", "1s", "--max-events", "1",
})
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Shell-based atomic writes | `storage.Store.AtomicWrite` | v1.0.20+ | All file operations now go through Go `storage` package |
| Shell spawn-tree.txt parsing | `agent.SpawnTree` with mutex | v1.0.20+ | Thread-safe spawn tracking |
| Direct COLONY_STATE.json writes | `state-mutate` with targeted jq | Phase 145 (2026-05-20) | Prevents Frankenstein state corruption |
| Silent error suppression (`2>/dev/null \|\| true`) | Honest error propagation | Phase 145 (2026-05-20) | Tests can now assert on failure paths |

**Deprecated/outdated:**
- Direct JSON file writes from LLM context: replaced by `state-mutate` subcommand.
- `flags.json` as primary file: `pending-decisions.json` is now primary, `flags.json` is fallback.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The "12 HIGH-severity untested source files" refers to the 10 source files listed in the phase description (eventbus, queen, instinct, midden, hive, spawn, autopilot, flags, council, shelf) plus 2 additional files within those domains | Summary | If the 12 files are different, the plan will miss required coverage. The roadmap says "eventbus, queen, instinct, midden, hive, spawn, autopilot, flags, council, shelf" which is 10 domains; the 12 count likely includes sub-files like `spawn_runs.go` and `spawn_track.go` or counts `eventbus.go` + `eventbus_subscribe.go` equivalents. |
| A2 | `flags_test.go` (6 tests) is considered adequate for `flags.go` and does not need replacement | Standard Stack | If flags coverage is insufficient, the planner may need to add more flags tests. Current tests cover list, alias, filter by type, filter by status, and empty state. |
| A3 | `shelf_test.go` (5 tests) covers `shelf_cmd.go` adequately and does not need a separate `shelf_cmd_test.go` | Standard Stack | Current shelf tests exercise add, promote, dismiss, list, and filter. If the planner decides shelf_cmd needs its own file, it would duplicate existing coverage. |
| A4 | The `events` and `agent` packages already have their own test files (32 and 16 tests respectively), so this phase focuses on the `cmd/` command wrappers, not the underlying packages | Summary | If the requirement includes pkg-level tests, the scope is larger than documented. The roadmap specifically says "source files" and lists `cmd/*.go` patterns. |

## Open Questions

1. **What are the exact 12 files?**
   - What we know: The roadmap lists 10 domains (eventbus, queen, instinct, midden, hive, spawn, autopilot, flags, council, shelf) and says "12 HIGH-severity untested source files."
   - What's unclear: Whether the 12 count includes sub-files (e.g., `spawn_runs.go`, `spawn_track.go`) or whether some domains have 2 files each.
   - Recommendation: Plan for the 10 domains first. If the count must reach 12, include `spawn_runs.go` and `spawn_track.go` or split large test files into domain-specific files.

2. **Does TEST-02 (smoke tests for public lifecycle commands) overlap with existing tests?**
   - What we know: Commands like `build`, `continue`, `plan`, `seal`, `entomb`, `colonize`, `status`, `oracle`, `swarm` have extensive existing tests (e.g., `codex_build_test.go` has 45 tests, `codex_continue_test.go` has 119 tests).
   - What's unclear: Whether these count as "smoke or fixture tests with executable evidence" for TEST-02, or if dedicated lightweight smoke tests are needed.
   - Recommendation: Audit existing tests for each public lifecycle command. If any command lacks at least one executable test, add a smoke test.

3. **Is there a preferred test file naming convention?**
   - What we know: Most tests use `{source}_test.go` (e.g., `flags.go` -> `flags_test.go`). Some use `{source}_{aspect}_test.go` (e.g., `queen_local_test.go`, `eventbus_subscribe_test.go`).
   - What's unclear: Whether new tests should be consolidated into a single `{source}_test.go` or split by aspect.
   - Recommendation: Use a single `{source}_test.go` per source file for consistency, unless the source file is very large (like `queen.go` with 19 functions + 10 commands).

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | All tests | Yes | 1.23 | — |
| `go test` | Test execution | Yes | built-in | — |
| Git | Test cleanup (worktree pruning) | Yes | 2.47 | — |
| Node.js | `eventbus_subscribe_test.go` narrator pipe | Optional | — | Skip with `t.Skip` |

**Missing dependencies with no fallback:** None.

**Missing dependencies with fallback:** Node.js is optional for one existing test; no action needed.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go standard testing (no external framework) |
| Config file | None — tests use `TestMain` in `testing_main_test.go` |
| Quick run command | `go test ./cmd/... -run 'TestEventBus\|TestQueen\|TestInstinct\|TestMidden\|TestHive\|TestSpawn\|TestAutopilot\|TestFlags\|TestCouncil\|TestShelf' -count=1` |
| Full suite command | `go test ./cmd/...` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| TEST-01 | Test files for 12 HIGH-severity source files | unit/integration | `go test ./cmd/...` | Partial — flags_test.go and shelf_test.go exist; 9 files need new tests |
| TEST-02 | Public lifecycle commands have smoke/fixture tests | smoke | `go test ./cmd/... -run 'TestBuild\|TestContinue\|TestPlan\|TestSeal\|TestEntomb\|TestColonize\|TestStatus\|TestOracle\|TestSwarm'` | Yes — extensive tests exist in codex_*_test.go files |
| TEST-03 | Test matrix documents coverage | documentation | Manual verification | No — needs creation |

### Sampling Rate
- **Per task commit:** `go test ./cmd/... -run 'Test{Domain}' -count=1`
- **Per wave merge:** `go test ./cmd/...`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `cmd/eventbus_test.go` — covers REQ-TEST-01 (eventbus publish, query, replay, cleanup)
- [ ] `cmd/queen_test.go` — covers REQ-TEST-01 (queen read, promote, thresholds, migrate, write-learnings)
- [ ] `cmd/instinct_test.go` — covers REQ-TEST-01 (instinct-create, read-trusted, decay-all, archive)
- [ ] `cmd/midden_cmds_test.go` — covers REQ-TEST-01 (midden-recent-failures, review, acknowledge, search, tag, collect, handle-revert, cross-pr-analysis, prune)
- [ ] `cmd/hive_test.go` — covers REQ-TEST-01 (hive-init, hive-store, hive-read, hive-abstract, hive-promote CLI)
- [ ] `cmd/spawn_test.go` — covers REQ-TEST-01 (spawn-tree-load, spawn-tree-active, validate-worker-response already tested; may focus on spawn-log/complete edge cases or spawn_runs.go)
- [ ] `cmd/autopilot_test.go` — covers REQ-TEST-01 (autopilot-init, update, status, stop, check-replan, set-headless, headless-check)
- [ ] `cmd/council_test.go` — covers REQ-TEST-01 (council-deliberate, advocate, challenger, sage, history, budget-check)
- [ ] `cmd/TEST_MATRIX.md` or similar — covers REQ-TEST-03

*(Note: `flags_test.go` and `shelf_test.go` already exist and cover their respective domains.)

## Security Domain

This phase does not introduce new security-sensitive code. Tests for existing commands should verify:
- Error paths do not leak internal file paths in production (tests can check error messages).
- JSON parsing failures are handled gracefully (no panics on malformed input).
- Hub directory redirection in tests prevents accidental modification of `~/.aether/`.

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V5 Input Validation | Yes | Commands validate required flags via `mustGetString` / `mustGetInt`; tests should verify missing-flag behavior |

## Sources

### Primary (HIGH confidence)
- Codebase inspection of `cmd/eventbus.go`, `cmd/queen.go`, `cmd/instinct.go`, `cmd/midden_cmds.go`, `cmd/hive.go`, `cmd/spawn.go`, `cmd/autopilot.go`, `cmd/flags.go`, `cmd/council.go`, `cmd/shelf_cmd.go` — function signatures, data-persistence paths, command definitions.
- Codebase inspection of `cmd/flags_test.go`, `cmd/shelf_test.go`, `cmd/queen_local_test.go`, `cmd/eventbus_subscribe_test.go`, `cmd/write_cmds_test.go`, `cmd/hive_runtime_test.go`, `cmd/security_cmds_test.go`, `cmd/compatibility_cmds_test.go` — existing test patterns.
- Codebase inspection of `cmd/testing_main_test.go`, `cmd/helpers_test.go`, `cmd/write_cmds_test.go` (lines 19-47) — test infrastructure (`newTestStore`, `parseEnvelope`, `saveGlobals`, `resetRootCmd`).
- Codebase inspection of `pkg/storage/storage.go`, `pkg/events/bus.go`, `pkg/agent/spawn_tree.go` — underlying persistence mechanisms.

### Secondary (MEDIUM confidence)
- `go test ./cmd/...` execution result (92.990s, all passing) — confirms test infrastructure is healthy.
- `go test ./pkg/events/... ./pkg/agent/... ./pkg/storage/...` — confirms underlying packages have their own tests.

### Tertiary (LOW confidence)
- None — all claims verified against codebase.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries are already in use and verified.
- Architecture: HIGH — patterns are well-established across 100+ existing tests.
- Pitfalls: HIGH — derived from direct observation of existing test infrastructure and common Go testing issues.

**Research date:** 2026-05-21
**Valid until:** 2026-06-21 (stable stack, low churn expected)
