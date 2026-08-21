# Phase 71: Platform Hardening - Pattern Map

**Mapped:** 2026-04-28
**Files analyzed:** 14 (8 new, 6 modify categories)
**Analogs found:** 11 / 14

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `cmd/smoke_test.go` | test | request-response | `cmd/command_count_test.go` | exact |
| `cmd/*_test.go` (per-subcommand flag tests) | test | request-response | `cmd/pheromone_write_test.go` | exact |
| `cmd/pheromone_write.go` (flag additions) | controller | request-response | `cmd/pheromone_write.go` | self |
| `cmd/midden_cmds.go` (flag additions) | controller | request-response | `cmd/midden_cmds.go` | self |
| `cmd/learning_cmds.go` (flag additions) | controller | request-response | `cmd/learning_cmds.go` | self |
| `cmd/instinct.go` (flag additions) | controller | request-response | `cmd/instinct.go` | self |
| `cmd/spawn_track.go` (flag additions) | controller | request-response | `cmd/spawn_track.go` | self |
| `cmd/spawn.go` (flag additions) | controller | request-response | `cmd/spawn.go` | self |
| `cmd/eventbus.go` (flag additions) | controller | request-response | `cmd/eventbus.go` | self |
| `cmd/flag_cmds.go` (flag additions) | controller | request-response | `cmd/flag_cmds.go` | self |
| `cmd/registry.go` (flag additions) | controller | request-response | `cmd/registry.go` | self |
| `cmd/shelf_cmd.go` (PLAT-01/02 audit) | controller | request-response | `cmd/shelf_cmd.go` | self |
| `cmd/codex_worker_cleanup.go` (incorporate) | controller | event-driven | `cmd/codex_worker_cleanup.go` | self |
| `pkg/codex/process_tracker.go` (incorporate) | service | event-driven | `pkg/codex/process_tracker.go` | self |

## Pattern Assignments

### `cmd/smoke_test.go` (test, request-response) -- NEW FILE

**Analog:** `cmd/command_count_test.go`

**Smoke test skeleton pattern** (lines 1-21 of command_count_test.go):
```go
package cmd

import (
    "testing"
)

// TestSubcommandSmokeTest verifies that all registered subcommands
// respond cleanly to --help. This catches missing flags, panics,
// and broken registrations. Covers PLAT-05.
func TestSubcommandSmokeTest(t *testing.T) {
    commands := rootCmd.Commands()

    for _, cmd := range commands {
        cmd := cmd // capture
        t.Run(cmd.Name(), func(t *testing.T) {
            saveGlobals(t)
            resetRootCmd(t)

            var buf bytes.Buffer
            stdout = &buf
            stderr = &buf

            rootCmd.SetArgs([]string{cmd.Name(), "--help"})
            err := rootCmd.Execute()

            // Subcommand should not panic or return error for --help
            if err != nil {
                t.Errorf("subcommand %q --help returned error: %v", cmd.Name(), err)
            }
            // Should produce some output
            if buf.Len() == 0 {
                t.Errorf("subcommand %q --help produced no output", cmd.Name())
            }
        })
    }
}
```

**Key constraint from pitfall analysis:** Do NOT check exact text -- check exit code 0, non-empty stdout, no stderr errors. See RESEARCH.md Pitfall 4.

---

### Per-subcommand flag tests (test, request-response) -- NEW TESTS

**Analog:** `cmd/pheromone_write_test.go`

**Test setup pattern** (lines 17-48 of pheromone_write_test.go):
```go
func TestSomeCommand_SomeFlag(t *testing.T) {
    saveGlobals(t)
    resetRootCmd(t)
    var buf bytes.Buffer
    stdout = &buf

    tmpDir := t.TempDir()
    dataDir := tmpDir + "/.aether/data"
    os.MkdirAll(dataDir, 0755)
    s, _ := storage.NewStore(dataDir)
    store = s

    os.Setenv("AETHER_ROOT", tmpDir)
    defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

    rootCmd.SetArgs([]string{"command-name", "--flag", "value"})

    err := rootCmd.Execute()
    if err != nil {
        t.Fatalf("command returned error: %v", err)
    }

    output := strings.TrimSpace(buf.String())
    var envelope map[string]interface{}
    if err := json.Unmarshal([]byte(output), &envelope); err != nil {
        t.Fatalf("invalid JSON output: %v", err)
    }
    if envelope["ok"] != true {
        t.Fatalf("expected ok:true, got: %s", output)
    }
    // ... flag-specific assertions
}
```

**Required imports for every test file:**
```go
import (
    "bytes"
    "encoding/json"
    "os"
    "strings"
    "testing"

    "github.com/calcosmic/Aether/pkg/colony"
    "github.com/calcosmic/Aether/pkg/storage"
)
```

**Critical test helpers** (from `cmd/testing_main_test.go` lines 72-130):
- `saveGlobals(t)` -- MUST be called first in every test that assigns to `store`, `stdout`, or `stderr`
- `resetRootCmd(t)` -- MUST be called before `rootCmd.Execute()` to prevent flag leakage
- `cleanupTestWorktrees()` -- auto-called in `TestMain`, no need to invoke manually

---

### `cmd/pheromone_write.go` (controller, request-response) -- MODIFY

**Analog:** self (current file)

**Current flags** (lines 306-314):
```go
func init() {
    pheromoneWriteCmd.Flags().String("type", "", "Signal type: FOCUS, REDIRECT, or FEEDBACK (required)")
    pheromoneWriteCmd.Flags().String("content", "", "Signal content (required)")
    pheromoneWriteCmd.Flags().String("priority", "", "Priority: low, normal, high (default based on type)")
    pheromoneWriteCmd.Flags().Float64("strength", 0, "Signal strength (default 1.0)")
    pheromoneWriteCmd.Flags().StringArray("tag", nil, "Signal tags (repeatable)")
    pheromoneWriteCmd.Flags().String("source", "cli", "Signal source (default \"cli\")")
    pheromoneWriteCmd.Flags().String("reason", "", "Reason for the signal (optional)")
    pheromoneWriteCmd.Flags().String("ttl", "", "Override expiry duration: Nd (days), Nh (hours), Nw (weeks) (optional)")

    pheromoneExpireCmd.Flags().String("id", "", "Signal ID to expire (required)")
    // ...
}
```

**Flag registration pattern:** All flags go in `init()` using `cmd.Flags().String(...)` with defaults. Never use `Required: true` -- validate manually in RunE.

---

### `cmd/midden_cmds.go` (controller, request-response) -- MODIFY

**Analog:** self (current file)

**Current flags** (lines 485-499):
```go
func init() {
    middenRecentFailuresCmd.Flags().Int("limit", 10, "Max entries to return")
    middenAcknowledgeCmd.Flags().String("id", "", "Entry ID to acknowledge")
    middenAcknowledgeCmd.Flags().String("category", "", "Category to acknowledge")
    middenSearchCmd.Flags().String("query", "", "Search query (required)")
    middenTagCmd.Flags().String("id", "", "Entry ID (required)")
    middenTagCmd.Flags().String("tag", "", "Tag to add (required)")
    middenCollectCmd.Flags().String("branch", "", "Branch name")
    middenCollectCmd.Flags().String("merge-sha", "", "Merge commit SHA")
    middenHandleRevertCmd.Flags().String("sha", "", "Revert commit SHA (required)")
    middenPruneCmd.Flags().Int("days", 30, "Remove entries older than N days")
    middenWriteCmd.Flags().String("category", "general", "Failure category")
    middenWriteCmd.Flags().String("message", "", "Failure message (required)")
    middenWriteCmd.Flags().String("source", "unknown", "Failure source")
    // ...
}
```

**Note:** The memory note from 2026-04-11 (16 days old per RESEARCH.md A2) says pheromone flags are missing, but the code shows they already exist. The planner should verify the ACTUAL mismatch by comparing markdown calls against these Go flags, not relying on the stale memory note.

---

### `cmd/spawn.go` (controller, request-response) -- MODIFY

**Analog:** self (current file)

**Current flags** (lines 357-383):
```go
func init() {
    spawnLogCmd.Flags().String("parent", "", "Parent agent name (required)")
    spawnLogCmd.Flags().String("caste", "", "Agent caste (required)")
    spawnLogCmd.Flags().String("name", "", "Agent name (required)")
    spawnLogCmd.Flags().String("id", "", "Legacy alias for child agent name")
    spawnLogCmd.Flags().String("task", "", "Task description (required)")
    spawnLogCmd.Flags().String("description", "", "Legacy alias for task description")
    spawnLogCmd.Flags().Int("depth", 0, "Spawn depth (required)")

    spawnCompleteCmd.Flags().String("name", "", "Agent name to complete (required)")
    spawnCompleteCmd.Flags().String("status", "", "Status: completed, failed, blocked (default: completed)")
    spawnCompleteCmd.Flags().String("summary", "", "Completion summary (optional)")
    // ...
}
```

---

### `cmd/spawn_track.go` (controller, request-response) -- MODIFY

**Analog:** self (current file)

**Current flags** (lines 157-162):
```go
func init() {
    spawnTrackCmd.Flags().String("action", "", "Action: start, check, or clear (required)")
    spawnTrackCmd.Flags().String("agent", "", "Agent name (required for start/check)")
    spawnTrackCmd.Flags().String("task", "", "Task ID (required for start)")
    spawnTrackCmd.Flags().Int("timeout", 0, "Timeout in seconds (0 = no timeout)")
    rootCmd.AddCommand(spawnTrackCmd)
}
```

---

### `cmd/flag_cmds.go` (controller, request-response) -- MODIFY

**Analog:** self (current file)

**Current flags** (lines 336-348):
```go
func init() {
    flagAddCmd.Flags().String("title", "", "Flag title/description (required)")
    flagAddCmd.Flags().String("severity", "", "Severity: critical, high, low (required)")
    flagAddCmd.Flags().String("source", "", "Source of the flag")
    flagAddCmd.Flags().String("type", "", "Flag type: blocker, issue, note (default: issue)")
    flagAddCmd.Flags().String("description", "", "Detailed description (defaults to title)")
    flagAddCmd.Flags().Int("phase", 0, "Phase number (0 means no phase)")

    flagResolveCmd.Flags().String("id", "", "Flag ID to resolve (required)")
    flagResolveCmd.Flags().String("message", "", "Resolution message")
    flagAcknowledgeCmd.Flags().String("id", "", "Flag ID to acknowledge (required)")
    flagAutoResolveCmd.Flags().Int("max-days", 7, "Maximum age in days for auto-resolution")
    // ...
}
```

---

### `cmd/registry.go` (controller, request-response) -- MODIFY

**Analog:** self (current file)

**Current flags** (lines 181-187):
```go
func init() {
    registryAddCmd.Flags().String("repo", "", "Repository path")
    registryAddCmd.Flags().String("path", "", "Repository path (alias for --repo)")
    registryAddCmd.Flags().String("domain", "", "Comma-separated domain tags")
    registryAddCmd.Flags().String("tags", "", "Comma-separated domain tags (alias for --domain)")
    registryAddCmd.Flags().String("goal", "", "Colony goal")
    registryAddCmd.Flags().Bool("active", true, "Set colony as active (default: true)")
    // ...
}
```

---

### `cmd/learning_cmds.go` (controller, request-response) -- MODIFY

**Analog:** self (current file)

**Current flags** (lines 477-494):
```go
func init() {
    learningApproveProposalsCmd.Flags().Bool("all", false, "Approve all pending proposals")
    learningApproveProposalsCmd.Flags().String("ids", "", "Comma-separated list of proposal IDs")
    learningDeferProposalsCmd.Flags().Bool("all", false, "Defer all pending proposals")
    learningDeferProposalsCmd.Flags().String("ids", "", "Comma-separated list of proposal IDs")
    learningExtractFallbackCmd.Flags().String("category", "", "Learning category (required)")
    learningExtractFallbackCmd.Flags().String("content", "", "Primary content (required)")
    learningExtractFallbackCmd.Flags().String("fallback", "", "Fallback content if extraction fails")
    learningInjectCmd.Flags().String("category", "", "Observation category (required)")
    learningInjectCmd.Flags().String("content", "", "Observation content (required)")
    learningInjectCmd.Flags().Float64("trust-score", 0.5, "Trust score for the observation")
    learningInjectCmd.Flags().String("source", "manual", "Source of the observation")
    learningSelectProposalsCmd.Flags().String("category", "", "Filter by category")
    learningUndoPromotionsCmd.Flags().Int("count", 1, "Number of recent promotions to undo")
    // ...
}
```

---

### `cmd/instinct.go` (controller, request-response) -- MODIFY

**Analog:** self (current file)

**Current flags** (lines 307-327):
```go
func init() {
    instinctCreateCmd.Flags().StringVar(&instinctTrigger, "trigger", "", "Trigger pattern (required)")
    instinctCreateCmd.Flags().StringVar(&instinctAction, "action", "", "Action pattern (required)")
    instinctCreateCmd.Flags().Float64Var(&instinctConfidence, "confidence", 0.75, "Initial confidence")
    instinctCreateCmd.Flags().StringVar(&instinctDomain, "domain", "", "Domain tag")
    instinctCreateCmd.Flags().StringVar(&instinctSource, "source", "observation", "Source type")
    instinctCreateCmd.Flags().StringVar(&instinctEvidence, "evidence", "", "Supporting evidence")
    instinctReadTrustedCmd.Flags().Float64Var(&instinctMinScore, "min-score", 0.5, "Minimum trust score")
    instinctReadTrustedCmd.Flags().StringVar(&instinctDomain, "domain", "", "Filter by domain")
    instinctReadTrustedCmd.Flags().IntVar(&instinctLimit, "limit", 20, "Maximum results")
    instinctDecayAllCmd.Flags().IntVar(&instinctDecayDays, "days", 30, "Days of decay to apply")
    instinctDecayAllCmd.Flags().BoolVar(&instinctDryRun, "dry-run", false, "Report without modifying")
    instinctArchiveCmd.Flags().StringVar(&instinctID, "id", "", "Instinct ID to archive (required)")
    // ...
}
```

---

### `cmd/eventbus.go` (controller, request-response) -- MODIFY

**Analog:** self (current file)

**Current flags** (lines 317-353):
```go
func init() {
    // event-bus-publish
    eventBusPublishCmd.Flags().StringVar(&eventTopic, "topic", "", "Event topic (required)")
    eventBusPublishCmd.Flags().StringVar(&eventPayload, "payload", "", "JSON payload (required)")
    eventBusPublishCmd.Flags().StringVar(&eventSource, "source", "", "Event source")
    eventBusPublishCmd.Flags().DurationVar(&eventTimeout, "timeout", eventBusDefaultTimeout, "Timeout")

    // event-bus-query
    eventBusQueryCmd.Flags().StringVar(&eventPattern, "pattern", "", "Topic pattern (supports trailing * wildcard)")
    eventBusQueryCmd.Flags().StringVar(&eventSince, "since", "", "Only return events after this RFC3339 timestamp")
    eventBusQueryCmd.Flags().IntVar(&eventLimit, "limit", 0, "Maximum events to return (0 for default)")
    // ... plus replay, cleanup, subscribe with similar patterns
}
```

---

### `cmd/shelf_cmd.go` (controller, request-response) -- PLAT-01/02 AUDIT

**Analog:** self (current file)

**Shelf commands already registered** (verified from RESEARCH.md):
- `shelf-list` with `--status` and `--json` flags
- `shelf-add` with `--text`, `--category`, `--source`, `--tag` flags
- `shelf-promote-batch` and `shelf-dismiss-batch` (verified to exist)

**Audit task:** Verify the OpenCode wrapper flow correctly invokes these subcommands. No code changes expected -- this is a manual audit task.

---

### `cmd/codex_worker_cleanup.go` (controller, event-driven) -- INCORPORATE

**Analog:** self (already exists in working tree)

**Pattern** (full file, 33 lines):
```go
package cmd

import (
    "fmt"
    "strings"

    "github.com/calcosmic/Aether/pkg/codex"
)

func cleanupStaleWorkersBeforeDispatch(root string) {
    root = strings.TrimSpace(root)
    if root == "" {
        return
    }
    result, err := codex.CleanupStaleWorkers(root)
    if err != nil {
        emitVisualProgress(fmt.Sprintf("Worker cleanup warning: %v", err))
        return
    }
    if len(result.Stale) == 0 && len(result.Failures) == 0 {
        return
    }
    emitVisualProgress(fmt.Sprintf(
        "Worker cleanup: %d stale worker(s), %d terminated, %d force-killed",
        len(result.Stale),
        len(result.Terminated),
        len(result.Killed),
    ))
    if len(result.Failures) > 0 {
        emitVisualProgress(fmt.Sprintf("Worker cleanup warning: %s", strings.Join(result.Failures, "; ")))
    }
}
```

---

### `pkg/codex/process_tracker.go` (service, event-driven) -- INCORPORATE

**Analog:** self (already exists in working tree)

**Key types** (lines 22-48):
```go
type TrackedProcess struct {
    PID        int       `json:"pid"`
    WorkerName string    `json:"worker_name,omitempty"`
    Caste      string    `json:"caste,omitempty"`
    Platform   string    `json:"platform,omitempty"`
    Root       string    `json:"root,omitempty"`
    SpawnedAt  time.Time `json:"spawned_at"`
}

type CleanupResult struct {
    Stale      []TrackedProcess `json:"stale,omitempty"`
    Terminated []int            `json:"terminated,omitempty"`
    Killed     []int            `json:"killed,omitempty"`
    Failures   []string         `json:"failures,omitempty"`
}
```

---

### `pkg/codex/process_group_unix.go` (service, event-driven) -- INCORPORATE

**Analog:** self (already exists in working tree)

**Platform-specific pattern** (lines 1-47):
```go
//go:build !windows

package codex

import (
    "os/exec"
    "strconv"
    "strings"
    "syscall"
)

func workerSysProcAttr() *syscall.SysProcAttr {
    return &syscall.SysProcAttr{Setpgid: true}
}

func terminateWorkerProcess(pid int) error {
    return syscall.Kill(-pid, syscall.SIGTERM)
}

func killWorkerProcess(pid int) error {
    return syscall.Kill(-pid, syscall.SIGKILL)
}
```

---

## Shared Patterns

### Test Infrastructure

**Source:** `cmd/testing_main_test.go`
**Apply to:** ALL test files in `cmd/`

```go
// Every test MUST call these two functions first:
saveGlobals(t)   // captures mutable globals, restores on cleanup
resetRootCmd(t)  // resets rootCmd args and all local flags to defaults

// Standard test setup for commands that need a store:
tmpDir := t.TempDir()
dataDir := tmpDir + "/.aether/data"
os.MkdirAll(dataDir, 0755)
s, _ := storage.NewStore(dataDir)
store = s
os.Setenv("AETHER_ROOT", tmpDir)
```

### Output Envelope Format

**Source:** `cmd/helpers_test.go` lines 11-36
**Apply to:** ALL controller and test files

```go
// All commands output JSON envelopes:
outputOK(map[string]interface{}{"key": "value"})     // success
outputError(1, "message", nil)                        // failure
outputErrorMessage("message")                         // no-store error

// Tests should parse the envelope:
var envelope map[string]interface{}
json.Unmarshal([]byte(output), &envelope)
if envelope["ok"] != true { /* fail */ }
result := envelope["result"].(map[string]interface{})
```

### Store Guard Pattern

**Source:** Every `cmd/*.go` file
**Apply to:** ALL controller files

```go
RunE: func(cmd *cobra.Command, args []string) error {
    if store == nil {
        outputErrorMessage("no store initialized")
        return nil  // NOTE: returns nil, not error
    }
    // ... actual logic
},
```

### Flag Registration Convention

**Source:** All `init()` functions across `cmd/`
**Apply to:** ALL flag additions (PLAT-04)

```go
// Two styles used in the codebase:
// Style 1: Direct flags (most common)
cmd.Flags().String("flag-name", "default", "Description")

// Style 2: Package-level vars (used by eventbus.go, instinct.go)
var someVar string
cmd.Flags().StringVar(&someVar, "flag-name", "default", "Description")

// CRITICAL RULES:
// 1. All flags MUST have defaults (never Required: true on the flag itself)
// 2. Validation happens in RunE, not via Cobra's MarkFlagRequired
// 3. Register flags in init(), never in the command definition
// 4. Registration order: flags first, then rootCmd.AddCommand()
```

### Subcommand Registration Convention

**Source:** All `init()` functions across `cmd/`
**Apply to:** ALL new subcommands (for the 6 that don't exist)

```go
func init() {
    // 1. Register flags
    newCmd.Flags().String("flag", "", "Description")

    // 2. Add to root
    rootCmd.AddCommand(newCmd)
}
```

### Command Count Guard

**Source:** `cmd/command_count_test.go`
**Apply to:** Smoke test file

```go
// After adding new subcommands, the count guard must be updated.
// Currently: len(commands) < 145
// After adding 6 missing subcommands, this should be >= 151
// The smoke test should also verify this count.
```

### Error Handling Pattern

**Source:** All `cmd/*.go` RunE functions
**Apply to:** ALL controller modifications

```go
// Commands never return Go errors to Cobra. They use:
outputError(exitCode, "message", nil)  // writes JSON error to stderr
outputErrorMessage("message")           // for no-store guard
return nil                              // always return nil from RunE

// This means broken flags don't crash the CLI -- they produce ok:false JSON.
// The markdown wrappers use `2>/dev/null || true` which silently swallows these.
```

## No Analog Found

Files with no close match in the codebase (planner should use RESEARCH.md patterns instead):

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `cmd/smoke_test.go` | test | request-response | No existing smoke test; use `command_count_test.go` as skeleton plus RESEARCH.md Pattern 2 |
| 6 missing subcommands | controller | request-response | Subcommands don't exist yet; need to identify them first, then create using standard patterns above |

## Metadata

**Analog search scope:** `cmd/` (289 Go files), `pkg/codex/` (12 Go files)
**Files scanned:** 14 source files read in full, 100+ files globbed/grepped
**Pattern extraction date:** 2026-04-28

**Key finding:** The memory note from 2026-04-11 (claiming 120+ broken CLI calls with missing flags) may be stale. Several flags it references (e.g., `--type`, `--content`, `--priority`, `--source`, `--reason`, `--ttl` for pheromone-write) already exist in the Go runtime. The planner MUST run a systematic audit: extract all `aether <subcommand>` calls from markdown files and playbooks, compare against `Flags()` registrations in Go, to identify the ACTUAL remaining mismatches. Do NOT trust the memory note blindly.
