# Coding Conventions

**Analysis Date:** 2026-08-01

## Naming Patterns

### Files

**Go:**
- Source files: `snake_case.go` (e.g., `state_load.go`, `init_cmd.go`, `assumptions.go`)
- Command files: `<command>_cmd.go` pattern (e.g., `assumptions.go` defines command, flags, and related helpers)
- Test files: `<name>_test.go` co-located with source (e.g., `flags_test.go`, `assumptions_test.go`)
- Package entry point: `<package>.go` or core concept file (e.g., `colony.go`, `storage.go`)

**TypeScript/JavaScript:**
- Source files: `kebab-case.ts` (e.g., `spawn-orchestrator.ts`, `go-command.ts`)
- Test files: `<name>.test.ts` or `<name>.test.js` co-located (e.g., `spawn-orchestrator.test.ts`)
- Compiled output: `dist/` directory with `.d.ts` declarations

### Functions

**Go:**
- Exported functions (public API): `PascalCase` (e.g., `loadActiveColonyState`, `Valid`, `Effective`)
- Private functions: `camelCase` (e.g., `loadColonyStateWithCompatibilityRepair`, `repairLegacyNumericStringFields`)
- Receiver methods: Same case rules as standalone functions
- Cobra command structs: `<verb><noun>Cmd` (e.g., `assumptionsAnalyzeCmd`, `flagListCmd`)
- Run functions: `run<CommandName>` (e.g., `runAssumptionsAnalyze`, `runAssumptionValidate`)
- Helper functions: Descriptive `<verb><noun>` pattern (e.g., `synthesizeAssumptions`, `buildSurfaceAssumption`)
- Render functions: `render<ItemType><Context>` (e.g., `renderAssumptionsAnalyzeVisual`)
- Test functions: `TestDescriptiveName` (e.g., `TestAssumptionsAnalyzeCreatesAssumptions`, `TestBuildCompletionStageMakesWrapperResultResumableWithoutRedispatch`)
- Helper functions in tests: `setup<Context>`, `save<Item>`, `reset<Item>` (e.g., `setupTestStore`, `saveGlobals`, `resetRootCmd`)

**TypeScript:**
- All functions: `camelCase` (e.g., `createSpawnOrchestrator`, `assertSafeGoArgs`, `runGoJSONCommand`)
- Factory functions: `create<Type>` prefix (e.g., `createSpawnOrchestrator`)
- Type validators: `assert<Type>` prefix or `is<Type>` (e.g., `assertSafeGoArgs`)
- Type extraction/conversion: `<source>To<Target>` (e.g., `buildDispatchToWorker`)
- Test names: Descriptive sentences or phrases (e.g., "passes argv arrays through without splitting arguments that contain spaces", "rejects missing command args before invoking Go")

### Variables

**Go:**
- Package-level globals: `camelCase` (e.g., `store`, `stdout`, `stderr`, `rootCmd`, `tracer`)
- Local variables: `camelCase` (e.g., `origStore`, `testHubDir`, `assumptions`, `err`)
- Module-level constants: `UPPER_CASE` or `camelCase` (e.g., `StateIDLE`, `PhasePending`, `assumptionsFile`)
- Error variables: `err<Description>` pattern (e.g., `errNoColonyInitialized`)
- Loop counters: Single letter (`i`, `j`, `k`) or descriptive (`idx`, `count`) when clarity matters
- Receiver variable: Single letter or descriptive (e.g., `(g PlanGranularity)`, `(s ColonyState)`)

**TypeScript:**
- All variables: `camelCase` (e.g., `totalBudget`, `remainingBudget`, `parentName`)
- Loop counters: Descriptive names preferred (e.g., `idx`, `count`) over single letters
- Module-level constants: `UPPER_CASE` (e.g., `MAX_SPAWN_DEPTH`, `DEFAULT_TOTAL_BUDGET`)
- Callback parameters: Match expected signature (e.g., `(chunk: unknown, ...args: unknown[]) => void`)

### Types

**Go:**
- Struct types: `PascalCase` (e.g., `ColonyState`, `BuildManifest`, `PlanGranularity`)
- Type aliases: `PascalCase` (e.g., `State`, `PhaseStatus`, `WorktreeStatus`)
- Interface types: `PascalCase` (e.g., `Reader`, `Writer`) — rarely used, prefer concrete structs
- Enum-like constants: Grouped with type prefix (e.g., `StateIDLE`, `StateREADY`, `StateEXECUTING`)

**TypeScript:**
- Interface types: `PascalCase` (e.g., `BuildManifest`, `SpawnOrchestrator`, `GoOutput<T>`)
- Type aliases: `PascalCase` (e.g., `GoJSONCaller`)
- Generic types: Capital letters (e.g., `<T>`, `<K>`, `<V>`)
- Exported interface properties: `snake_case` for JSON compatibility (e.g., `phase_name`, `colony_depth`, `dispatch_mode`)
- Union types: Use `|` syntax with clear member names

## Code Style

### Formatting

**Go:**
- Formatted with `gofmt` (standard Go formatter, non-negotiable)
- Indentation: Tabs (Go standard)
- Line length: No fixed limit; `gofmt` handles breaks
- Semicolons: Never used (Go doesn't require them)

**TypeScript:**
- Formatted with TypeScript compiler (tsc) and strict mode
- Indentation: 2 spaces (enforced by tsconfig)
- Line length: No fixed limit; prefer readability
- Target: `ES2022`, `module: "NodeNext"`, `moduleResolution: "NodeNext"`

### Linting

**Go:**
- No ESLint or golangci-yml configuration
- Reliance on: Go's built-in `gofmt`, `go vet`, and structured error handling
- Code reviews catch patterns; no automated rule enforcement

**TypeScript:**
- No ESLint configuration found
- Strict mode enforced: `strict: true`, `noUncheckedIndexedAccess: true`, `exactOptionalPropertyTypes: true`
- Type safety over linting; the compiler is the gatekeeper

### Comments

**When to Comment:**
- Explain "why" for non-obvious behavior (e.g., compatibility repairs, state transitions)
- Document public API functions at package level (Go)
- Explain complex algorithms or decision logic
- Mark temporary workarounds with `TODO` or `FIXME` (searchable for cleanup)
- Clarify state machine transitions or invariants

**Go Comments:**
- Doc comments above exported items: `// FunctionName does X and returns Y.`
- Package-level comment: `// Package name describes purpose.` (first line of any file in the package)
- Private function comments: Optional, use when logic is non-obvious
- Format: `// Single space after comment marker, full sentences`

**TypeScript Comments:**
- JSDoc for exported functions: `/** Create orchestrator. @param opts Config. @returns Instance. */`
- Include `@param` and return type documentation for exported functions
- Inline explanations: `// Why this is needed` on same line or above statement
- Block comments: `/* ... */` for multi-line explanations

**Comment Style (Both Languages):**
- Avoid `//nolint`, `//nocheck` directives; fix the underlying issue
- Don't repeat what the code obviously does; explain the intent
- Update comments when modifying code they describe

## Import Organization

### Go

**Order (strictly enforced by `gofmt`):**
1. Standard library imports (e.g., `"fmt"`, `"os"`, `"encoding/json"`, `"testing"`)
2. Third-party imports (e.g., `"github.com/spf13/cobra"`, `"github.com/anthropics/anthropic-sdk-go"`)
3. Local package imports (e.g., `"github.com/calcosmic/Aether/pkg/colony"`)

**Format:**
```go
import (
    "encoding/json"
    "fmt"
    "os"
    "strings"
    "time"

    "github.com/calcosmic/Aether/pkg/colony"
    "github.com/spf13/cobra"
)
```

### TypeScript

**Order:**
1. Node.js built-in modules: `import { X } from "node:module"`
2. Third-party dependencies: `import lib from "library"`
3. Local modules: `import { X } from "./relative-path.js"`
4. Type imports: `import type { T } from "./types.js"`

**Format:**
```typescript
import { execFileSync } from "node:child_process";
import { readFileSync } from "node:fs";
import yaml from "js-yaml";
import { createSpawnOrchestrator } from "./spawn-orchestrator.js";
import type { SpawnClaim } from "./types.js";
```

**Path Aliases:**
- No aliases defined in this codebase; use relative paths with `.js` extensions (ESM requirement)
- Example: `import { X } from "../src/module.js"` not `import { X } from "@/module"`

## Error Handling

### Pattern

**Go:**
```go
if err != nil {
    return nil, fmt.Errorf("operation context: %w", err)  // Wrap with %w
}
```

**TypeScript:**
```typescript
if (typeof arg !== "string") {
    throw new Error(`Go command arg ${index} must be a string`);
}
```

### Convention

**Go:**
- Always check `err != nil` immediately after operations that return errors
- Wrap errors using `%w` for error chains (enables `errors.Is()` and `errors.As()`)
- Error messages: lowercase, no period, include context (e.g., `"failed to save assumptions: %w"`)
- Named error variables: `var errDescription = errors.New("message")`
- Sentinel errors for specific failure modes (e.g., `errNoColonyInitialized`)
- No throwing exceptions; use error return values exclusively

**TypeScript:**
- Throw `Error` with descriptive messages for validation failures
- Use try/catch for exceptional flow control sparingly
- Return error values when error is expected part of normal flow
- Validate inputs early; throw with context (e.g., `throw new Error("Go command arg X must be Y")`)

### Error Output

Errors flow through the system as JSON envelopes controlled by `AETHER_OUTPUT_MODE`:
```typescript
type GoOutput<T> = {
  ok: boolean;
  result?: T;
  error?: string;
  code?: number;
}
```

Success: `{ ok: true, result: <data> }`
Failure: `{ ok: false, error: "message", code: <number> }`

## Logging

### Framework

**Go:** Standard `log` package via `log.Printf()` (logs to stderr)

**TypeScript:** `console.error()` for errors, `console.log()` for info (tests suppress or capture)

### Pattern

```go
log.Printf("context: message with %v values", value)
```

### When to Log

**Logs (stderr):**
- Unexpected errors that need operator visibility
- Non-JSON output mode (visual mode gets banner + formatted output instead)
- Startup diagnostics or version information

**Structured Output (JSON or visual):**
- Use `outputWorkflow()` for command results
- Use `outputOK()` for JSON success envelope
- Use `outputError()` for JSON error envelope

**Test Code:**
- No logging in tests; logging is suppressed by default
- Use `t.Logf()` only for temporary debug output during development
- Remove `t.Logf()` calls before committing

**Never log:**
- Secrets or credentials (configuration always uses environment variables)
- Large data structures (serialize to JSON output instead)
- Personally identifiable information (PII) without sanitization

## Function Design

### Size

- Prefer small, single-responsibility functions (goal: <50 lines for new code)
- Extract helpers for repeated patterns or complex logic
- Break complex functions into named helpers that make intent clear
- Test size correlates with function size; big functions need comprehensive testing

### Parameters

**Go:**
- Limit to 3-4 parameters; use structs for more
- Receiver parameter counts as the first (methods on types are common)
- Avoid boolean flags; use options structs or named types for configuration

**TypeScript:**
- Use options objects (`interface Options { ... }`) for multiple parameters
- Avoid boolean flags; prefer option structs
- Type all parameters and return values (strict mode enforces this)

### Return Values

**Go:**
- Standard: `(result T, error error)` for fallible operations
- Success-only: `(result T)` for pure functions or guaranteed operations
- Multiple results: Tuple-like with `(type1, type2, error)` pattern
- Named return values: Acceptable for complex functions (e.g., `func F() (count int, err error)`)

**TypeScript:**
- Return typed results directly; types are explicit in signature
- Throw for exceptional cases; return values for expected outcomes
- Use tuples for multiple returns: `[count: number, err: Error | null]`
- Async functions return `Promise<T>`

### Example (Go)

```go
func loadActiveColonyState() (colony.ColonyState, error) {
    if store == nil {
        return colony.ColonyState{}, fmt.Errorf("no store initialized")
    }
    state, err := loadColonyStateWithCompatibilityRepair()
    if err != nil {
        return colony.ColonyState{}, fmt.Errorf("failed to load colony state: %w", err)
    }
    return state, nil
}
```

### Example (TypeScript)

```typescript
export function createSpawnOrchestrator(
  opts: SpawnOrchestratorOptions = {}
): SpawnOrchestrator {
  const totalBudget = opts.totalBudget ?? DEFAULT_TOTAL_BUDGET;
  let consumedBudget = opts.consumedBudget ?? 0;
  
  return {
    get remainingBudget(): number {
      return totalBudget - consumedBudget;
    }
  };
}
```

## Module Design

### Exports (Go)

- Exported (public): Capitalized names (e.g., `type ColonyState struct`, `func LoadState()`)
- Not exported (private): Lowercase names (e.g., `type worktreeStatus string`, `func loadRawJSON()`)
- Package naming: Short, descriptive (e.g., `colony`, `storage`, `codex`, `agent`)
- Packages are namespaces; prefer few large packages over many tiny ones

### Exports (TypeScript)

- Exported: Use `export` keyword on functions, interfaces, and types
- Type imports: Use `import type` for types to avoid circular dependencies
- Re-exports: Avoid barrel files; import directly from source modules
- Index files: Use `export * from "./module.js"` only when consolidating related exports

### Barrel Files

- Go: Not used; each package stands alone
- TypeScript: Not used in this codebase; prefer direct imports from source

### Visibility Pattern

```go
// Public API — exported
func (s ColonyState) EffectiveScope() ColonyScope { ... }

// Private detail — not exported
func normalizeLegacyColonyState(s ColonyState) ColonyState { ... }
```

## Struct/Object Design

### Go Structs

- JSON mapping: Use struct tags `json:"field_name"` for JSON schema compatibility
- Omitting empty fields: `json:"field,omitempty"` for optional fields (null in JSON)
- Nested structs: Inline simple types for clarity; use explicit fields for complex objects
- Pointer receivers: Use `*T` for methods that modify the receiver
- Embedded structs: Use for composition when appropriate (e.g., embedding a `Meta` struct)

**Example:**
```go
type ColonyState struct {
    Version      string    `json:"version"`
    Goal         *string   `json:"goal,omitempty"`
    State        State     `json:"state"`
    CurrentPhase int       `json:"current_phase"`
    Events       []string  `json:"events"`
}
```

### TypeScript Interfaces

- Properties are `camelCase` (interface names are PascalCase)
- Optional properties: Use `?` suffix (e.g., `result?: T`, `error?: string`)
- Property order: Group related fields together logically
- No methods on interfaces; use separate function signatures
- Union types: Clearly document member options

**Example:**
```typescript
export interface BuildManifest {
  phase: number;
  phase_name: string;
  goal?: string;
  root: string;
  tasks: BuildTaskPlan[];
  generated_at: string;
}
```

## Constants and Enums

**Go Enum-like Constants:**
- Group by type with clear prefix (e.g., `StateIDLE`, `StateREADY`, `StateEXECUTING`)
- String-typed: Use named constants for state values
- Validation method: Include a `Valid()` method on the type
- Comparison: Use typed constants, not magic strings

**TypeScript Enum-like Constants:**
- Use `const` with literal types (no `enum` keyword used in this codebase)
- Group related constants in objects or as module-level constants

---

*Convention analysis: 2026-08-01*
