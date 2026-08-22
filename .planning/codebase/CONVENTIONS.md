---
last_mapped_commit: 92252d01
---

# Coding Conventions

**Analysis Date:** 2026-08-22

## Naming Patterns

**Files:**
- Source files: lowercase with underscores (e.g., `state_load.go`, `update_availability.go`)
- Test files: `{module}_test.go` (co-located with source)
- Constants/configuration: descriptive lowercase with underscores

**Functions:**
- Exported: Title case, descriptive (e.g., `LoadActiveColonyState()`, `BuildPorterReadiness()`)
- Private: camelCase, descriptive (e.g., `loadColonyStateWithCompatibilityRepair()`, `emitPromptIntegrityEvents()`)
- Test helpers: `test{Action}` or descriptive camelCase (e.g., `saveGlobals()`, `resetRootCmd()`)
- Command functions: lowercase descriptive, often with `Cmd` suffix for cobra commands

**Variables:**
- camelCase throughout: `origStore`, `dataDir`, `taskID`, `endedRunRecord`
- Error variables: `errNoColonyInitialized`, `loadErr`, `rawErr` (err prefix with descriptive suffix)
- Short-lived loop variables: standard `i`, `j`, but prefer descriptive names when non-trivial

**Types/Structs:**
- Exported: Title case (e.g., `ColonyState`, `CatalogEntry`, `VerificationErrorClass`)
- Private: lowercase starting letter (e.g., `advancePhaseParams`, `autopilotState`, `buildAttemptRecord`)
- Struct fields: Title case if exported, camelCase if private

**Constants:**
- Package-level: camelCase or SCREAMING_SNAKE_CASE (both used):
  - camelCase common: `versionCacheFile`, `autopilotStatePath`, `assumptionsFile`
  - SCREAMING_SNAKE_CASE for critical markers: `installedVersionMarkerRel`, `spendSessionRel`
- Cobra command groups: descriptive names
- Regex patterns: descriptive name with `Re` suffix (e.g., `aetherPathLiteralRe`)

## Code Style

**Formatting:**
- Standard `gofmt` (implicit, no `.gofmt` config file)
- Standard `go vet` used in CI (`make lint`)
- No explicit prettier or eslint equivalents (Go-only project)

**Linting:**
- `go vet ./...` enforced in CI
- goreleaser validation: `goreleaser check` in CI
- Race detector: `go test ./... -race` run in CI with 2400s timeout

**Go Version:**
- Go 1.26.5 required
- Module: `github.com/calcosmic/Aether`

## Import Organization

**Order:**
1. Standard library imports (alphabetized): `context`, `encoding/json`, `fmt`, `os`, `strings`, etc.
2. Blank line separator
3. External imports (alphabetized): `github.com/anthropics/anthropic-sdk-go`, `github.com/spf13/cobra`, etc.

**Pattern Example:**
```go
import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)
```

**No Path Aliases:**
- All imports use full canonical paths
- No import aliases except where unavoidable (rare)

## Error Handling

**Patterns:**
- Explicit error checking: `if err != nil { return fmt.Errorf(...) }`
- Error wrapping with `%w` for context propagation
- Named error variables for sentinel checks: `errors.Is(err, os.ErrNotExist)`
- Custom error variables with `err` prefix: `var errNoColonyInitialized = errors.New("no colony initialized")`
- No panic-on-error; all errors are either returned or logged

**Testing Error Assertions:**
```go
if err := someFunction(); err != nil {
	t.Fatalf("setup failed: %v", err)  // Fatal for test setup
}
if len(result) == 0 {
	t.Errorf("unexpected empty result")  // Error for test assertion
}
```

**Error Messages:**
- Lowercase start (standard Go style)
- Context-rich, include what failed and why
- Never include stack traces in error messages

## Logging & Output

**Framework:** No centralized logging framework; uses direct `stdout`/`stderr` writes and structured event emission.

**Patterns:**
- `fmt.Fprintf(stdout, "message\n")` for human output
- `fmt.Fprintf(stderr, "error\n")` for errors
- JSON output via `json.Marshal()` and `fmt.Fprint(stdout, ...)`
- Event bus: `bus.Publish(context.Background(), topic, payload, source)` for structured events

**Visibility:**
- NO_COLOR environment variable respected (e.g., `os.Getenv("NO_COLOR")`)
- Output mode: `AETHER_OUTPUT_MODE` env var controls JSON vs. human output

## Comments

**When to Comment:**
- Before complex functions explaining purpose and constraints
- Inline for non-obvious logic
- DO NOT comment obvious code
- Important architectural decisions and reasons

**JSDoc/Godoc Style:**
```go
// LoadActiveColonyState loads the current colony's state with compatibility
// repairs for schema evolution. Returns errNoColonyInitialized if no colony
// is initialized.
func loadActiveColonyState() (colony.ColonyState, error) {
```

**Test Comments:**
- Before tests explaining the test's intent and what invariant it pins
- Comments inside tests explaining non-obvious setup or assertion

**Architecture Notes:**
- Comments referencing related files: `(see xyz.go for implementation)`
- Comments referencing test cases: `(pinned by TestXYZ)`
- Comments about versions/history: cite commit hashes or phase numbers

## Function Design

**Size:** Aim for focused functions (typically 20-50 lines); complex logic broken into helpers.

**Parameters:**
- Prefer explicit parameters over global state when possible
- Use `*testing.T` as first parameter in test helpers
- Context as first parameter for functions with cancellation/timeout needs

**Return Values:**
- Error as last return value (Go convention)
- Multiple returns for (value, error) or (value, ok bool)
- No named return values except when they provide real documentation value
- Never use named returns for multiple returns without clear meaning

**Receiver Methods:**
- Pointer receivers for methods that mutate state
- Value receivers for read-only methods on small types

## Module Design

**Exports:**
- Title-case names are exported (public)
- lowercase names are private (package-local)
- No module initialization side effects
- Exported types get godoc comments

**Packages:**
- `cmd/` — Contains all command implementations and most business logic
- `pkg/agent/` — Agent pool, spawn tree, curation
- `pkg/colony/` — Colony types and state management
- `pkg/codex/` — Codex integration
- `pkg/storage/` — File storage and JSON persistence
- `pkg/events/` — Event bus implementation
- `pkg/memory/` — Learning pipeline and instincts
- `pkg/graph/` — Knowledge graph persistence
- `pkg/smoke/` — Smoke/acceptance testing utilities

**Internal Organization:**
- Single command file per major feature (e.g., `build_*.go`, `continue_*.go`)
- Shared utilities in underscore-prefixed files (e.g., `build_help.go`, `continue_verify.go`)
- Test files co-located (same package, `_test.go` suffix)
- No package-level init functions
- Global mutable state (like `store`, `stdout`) clearly marked as such in comments

## Type Design

**Struct Composition:**
- Embed types only when truly implementing composition (no anonymous embed just for field access)
- Use explicit named fields; avoid struct{} for option bags (use typed Options struct if needed)
- JSON struct tags: `json:"field_name,omitempty"` standard pattern

**Interfaces:**
- Small, focused interfaces (1-3 methods)
- Accept interfaces, return concrete types (standard Go pattern)
- Used primarily for testing (mocking and injection)

## Constants vs Variables

**When to Use Constants:**
- Configuration values that never change: `const installTimeout = 30 * time.Second`
- Magic numbers with meaning: `const maxPhaseCount = 20`
- File paths and relative paths: `const installedVersionMarkerRel = "data/installed-version.json"`

**When to Use Variables:**
- Values that are computed at runtime or in `init()`
- Global state that's mutated (e.g., `var store *storage.Store`)
- Test fixtures: `var testHubDir string`

## Concurrency

**Patterns:**
- Mutex for protecting shared state: `sync.Mutex`, `sync.RWMutex`
- Channel-based patterns for worker pools (used in agent dispatching)
- Context for cancellation and timeouts
- No goroutine leaks; always wait for spawned goroutines to exit

**Race Detector:**
- Run with `-race` flag in tests and CI
- All races must be fixed, not suppressed

---

*Convention analysis: 2026-08-22*
