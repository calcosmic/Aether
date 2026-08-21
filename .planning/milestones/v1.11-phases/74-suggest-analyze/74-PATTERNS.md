# Phase 74: Suggest-Analyze - Pattern Map

**Mapped:** 2026-04-29
**Files analyzed:** 10
**Analogs found:** 8 / 10

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `cmd/suggest_analyze.go` | controller (CLI command) | request-response | `cmd/pheromone_write.go` | exact |
| `cmd/suggest_analyze_test.go` | test | request-response | `cmd/pheromone_write_test.go` | exact |
| `cmd/suggest_approve.go` | controller (CLI command) | request-response | `cmd/pheromones_read.go` | role-match |
| `cmd/suggest_approve_test.go` | test | request-response | `cmd/write_cmds_test.go:135-180` | role-match |
| `cmd/compatibility_cmds.go` | controller (modify) | request-response | (self -- stub at lines 113-133) | self |
| `pkg/colony/colony.go` | model (modify) | CRUD | `pkg/colony/colony.go:175-204` | self |
| `.aether/docs/command-playbooks/build-context.md` | config (modify) | request-response | (self -- Step 4.2 at lines 147-181) | self |
| `.claude/commands/ant/build.md` | config (wrapper) | request-response | `.claude/commands/ant/build.md` | self |
| `.opencode/commands/ant/build.md` | config (wrapper) | request-response | `.opencode/commands/ant/build.md` | self |
| `.codex/CODEX.md` | config (wrapper) | request-response | `.codex/CODEX.md` | self |

## Pattern Assignments

### `cmd/suggest_analyze.go` (CLI command, request-response)

**Analog:** `cmd/pheromone_write.go`

**Imports pattern** (lines 1-18):
```go
import (
    "crypto/rand"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "strings"
    "time"

    "github.com/calcosmic/Aether/pkg/colony"
    "github.com/calcosmic/Aether/pkg/storage"
    "github.com/spf13/cobra"
)
```
Note: `suggest-analyze` will also need `os/exec` for git diff and `os` for working directory. Drop `crypto/rand` and `encoding/hex` unless generating IDs.

**CLI command skeleton with store guard** (lines 20-28):
```go
var suggestAnalyzeCmd = &cobra.Command{
    Use:   "suggest-analyze",
    Short: "Analyze codebase for pheromone suggestions",
    Args:  cobra.NoArgs,
    RunE: func(cmd *cobra.Command, args []string) error {
        if store == nil {
            outputErrorMessage("no store initialized")
            return nil
        }
        // ... command logic
    },
}
```

**outputOK envelope pattern** (lines 211-216):
```go
outputOK(map[string]interface{}{
    "created":  true,
    "signal":   signal,
    "total":    len(pf.Signals),
    "replaced": replaced,
})
```

**Colony state load for source_phase** (lines 113-117):
```go
var cs colony.ColonyState
if loadErr := store.LoadJSON("COLONY_STATE.json", &cs); loadErr == nil && cs.CurrentPhase > 0 {
    signal.SourcePhase = &cs.CurrentPhase
}
```

**Init registration pattern** (lines 306-321):
```go
func init() {
    suggestAnalyzeCmd.Flags().Bool("dry-run", false, "Preview suggestions without persisting")
    suggestAnalyzeCmd.Flags().String("target", ".", "Target directory to analyze")
    rootCmd.AddCommand(suggestAnalyzeCmd)
}
```

**Key reusable functions from same package (cmd/):**
- `generatePheromoneSuggestions(target, governance, dirClass, techStack)` -- `cmd/init_research.go:1310`
- `detectGovernance(target)` -- `cmd/init_research.go:157`
- `classifyDirectory(target)` -- `cmd/init_research.go:297`
- `parseDependencyFiles(target)` -- `cmd/init_research.go:1238` (NOTE: function is named `parseDependencyFiles`, not `parseTechStack`)
- `hasFile(target, name)` -- `cmd/init_research.go` (utility)
- `fileContains(target, file, pattern)` -- `cmd/init_research.go` (utility)
- `sha256Sum(s)` -- `cmd/pheromone_write.go:350-353`
- `generateSignalID()` -- `cmd/pheromone_write.go:356-360`
- `loadActiveColonyState()` -- `cmd/state_load.go:17`
- `colony.SanitizeSignalContent()` -- `pkg/colony/sanitize.go`

---

### `cmd/suggest_analyze_test.go` (test, request-response)

**Analog:** `cmd/pheromone_write_test.go`

**Test setup pattern** (lines 17-37):
```go
func TestSuggestAnalyze_Basic(t *testing.T) {
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

    rootCmd.SetArgs([]string{"suggest-analyze"})
    err := rootCmd.Execute()
    // ...
}
```

**Alternative test setup using newTestStore** (from `cmd/write_cmds_test.go:135-143`):
```go
func TestPheromoneWriteDedup(t *testing.T) {
    saveGlobals(t)
    resetRootCmd(t)
    var buf bytes.Buffer
    stdout = &buf

    s, tmpDir := newTestStore(t)
    defer os.RemoveAll(tmpDir)
    store = s
```

**JSON output assertion pattern** (lines 39-46):
```go
output := strings.TrimSpace(buf.String())
var envelope map[string]interface{}
if err := json.Unmarshal([]byte(output), &envelope); err != nil {
    t.Fatalf("invalid JSON output: %v", err)
}
if envelope["ok"] != true {
    t.Fatalf("expected ok:true, got: %s", output)
}
```

**Colony state fixture pattern** (from `cmd/pheromone_write_test.go:604-615`):
```go
goal := "Test suggest analyze"
phase := 1
state := colony.ColonyState{
    Version:      "3.0",
    Goal:         &goal,
    State:        colony.StateEXECUTING,
    CurrentPhase: phase,
    Plan:         colony.Plan{},
}
stateData, _ := json.MarshalIndent(state, "", "  ")
os.WriteFile(dataDir+"/COLONY_STATE.json", stateData, 0644)
```

**Helper functions to use:**
- `saveGlobals(t)` -- `cmd/testing_main_test.go:72`
- `resetRootCmd(t)` -- `cmd/testing_main_test.go:121`
- `newTestStore(t)` -- `cmd/write_cmds_test.go:21` (returns `*storage.Store, string`)

---

### `cmd/suggest_approve.go` (CLI command, request-response)

**Analog:** `cmd/pheromones_read.go`

**Simple read command pattern** (lines 8-42):
```go
var pheromoneReadCmd = &cobra.Command{
    Use:   "pheromone-read",
    Short: "Read pheromone signals",
    Args:  cobra.NoArgs,
    RunE: func(cmd *cobra.Command, args []string) error {
        if store == nil {
            outputErrorMessage("no store initialized")
            return nil
        }

        var pf colony.PheromoneFile
        if err := store.LoadJSON("pheromones.json", &pf); err != nil {
            outputOK(map[string]interface{}{
                "signals": []interface{}{},
            })
            return nil
        }
        // ... filter and output
    },
}
```

**Existing stub to replace** (`cmd/compatibility_cmds.go:113-133`):
```go
var suggestApproveCmd = &cobra.Command{
    Use:   "suggest-approve",
    Short: "Review and approve pheromone suggestions from suggest-analyze",
    RunE: func(cmd *cobra.Command, args []string) error {
        if store == nil {
            outputErrorMessage("no store initialized")
            return nil
        }
        dryRun, _ := cmd.Flags().GetBool("dry-run")
        _, err := loadActiveColonyState()
        if err != nil {
            outputError(1, fmt.Sprintf("failed to read colony state: %v", err), nil)
            return nil
        }
        outputOK(map[string]interface{}{
            "suggestions": []interface{}{},
            "dry_run":     dryRun,
        })
        return nil
    },
}
```

**Pheromone dedup check pattern** (from `cmd/pheromone_write.go:157-181`):
```go
for i := range pf.Signals {
    sig := &pf.Signals[i]
    if !sig.Active {
        continue
    }
    if sig.Type == sigType && sig.ContentHash != nil && *sig.ContentHash == contentHash {
        // Match found -- this suggestion already exists as an active pheromone
        sig.ReinforcementCount = ...
        replaced = true
        break
    }
}
```

**Content hash computation** (from `cmd/pheromone_write.go:87-88`):
```go
h := sha256Sum(content)
contentHash := "sha256:" + h
```

**Writing a pheromone via same-package access** (from `cmd/pheromone_write.go:100-111`):
```go
signal := colony.PheromoneSignal{
    ID:          id,
    Type:        sigType,
    Content:     json.RawMessage(contentJSON),
    Priority:    priority,
    Source:      sourceFlag,
    CreatedAt:   now,
    Active:      true,
    Strength:    &strength,
    ContentHash: &contentHash,
    Tags:        make([]colony.PheromoneTag, 0, len(tags)),
}
```

---

### `cmd/suggest_approve_test.go` (test, request-response)

**Analog:** `cmd/write_cmds_test.go:135-180` (TestPheromoneWriteDedup)

**Dedup test pattern** (lines 135-180):
```go
func TestPheromoneWriteDedup(t *testing.T) {
    saveGlobals(t)
    resetRootCmd(t)
    var buf bytes.Buffer
    stdout = &buf

    s, tmpDir := newTestStore(t)
    defer os.RemoveAll(tmpDir)
    store = s

    // First write
    buf.Reset()
    rootCmd.SetArgs([]string{"pheromone-write", "--type", "FOCUS", "--content", "test dedup"})
    err := rootCmd.Execute()
    // ...

    // Second write with identical type and content
    buf.Reset()
    rootCmd.SetArgs([]string{"pheromone-write", "--type", "FOCUS", "--content", "test dedup"})
    err = rootCmd.Execute()
    // ...

    // Load pheromones.json and verify dedup behavior
    var pf colony.PheromoneFile
    s.LoadJSON("pheromones.json", &pf)

    if len(pf.Signals) != 1 {
        t.Errorf("signal count = %d, want 1", len(pf.Signals))
    }
}
```

---

### `cmd/compatibility_cmds.go` (modify -- remove stub)

**Analog:** self (lines 113-133, 147-167)

**Current stub location** (lines 113-133): The `suggestApproveCmd` variable and its body.

**Registration in init()** (lines 161, 166):
```go
suggestApproveCmd.Flags().Bool("dry-run", false, "Preview without persisting approvals")
rootCmd.AddCommand(suggestApproveCmd)
```

**Action:** Remove the `suggestApproveCmd` variable (lines 113-133), its flag registration (line 161), and its `AddCommand` call (line 166). The real command will be registered in `cmd/suggest_approve.go`.

---

### `pkg/colony/colony.go` (model, modify)

**Analog:** self (lines 175-204)

**ColonyState struct** (lines 175-204):
```go
type ColonyState struct {
    Version            string          `json:"version"`
    Goal               *string         `json:"goal"`
    // ... many existing fields ...
    Charter            *Charter           `json:"charter,omitempty"`
}
```

**Action:** Add new fields for pending suggestions storage. Pattern: add `omitempty` JSON tags for optional fields. Use pointer slices or new struct types. Example:
```go
// PendingSuggestions holds unreviewed suggestions from suggest-analyze.
type PendingSuggestion struct {
    ID        string `json:"id"`
    Type      string `json:"type"`
    Content   string `json:"content"`
    Reason    string `json:"reason"`
    CreatedAt string `json:"created_at"`
    Dismissed bool   `json:"dismissed"`
}
```
Add `PendingSuggestions *[]PendingSuggestion` to `ColonyState` struct.

---

### `.aether/docs/command-playbooks/build-context.md` (config, modify)

**Analog:** self (lines 147-181)

**Current DEPRECATED Step 4.2** (lines 147-181):
```markdown
### Step 4.2: Suggest Pheromones (DEPRECATED)

**Conditional step -- skipped if `--no-suggest` flag is passed.**

> **DEPRECATED**: The `suggest-*` commands have been deprecated and will be removed
> in a future version. They return `ok:true` with `deprecated:true` for backward
> compatibility. This step now always skips gracefully.
```

**Action:** Replace the DEPRECATED block with a restored Step 4.2 that calls `aether suggest-analyze` and then `aether suggest-approve`. Follow the same conditional/non-blocking pattern:
- Skip if `--no-suggest` flag
- Non-blocking: never stops the build
- Use Bash tool with `AETHER_OUTPUT_MODE=json`
- Parse JSON output for suggestion count

---

### `.claude/commands/ant/build.md` and `.opencode/commands/ant/build.md` (wrapper, modify)

**Analog:** self (both files are identical, 126 lines each)

**Key insight:** These wrappers currently do NOT call suggest-analyze directly. The suggest flow is in the build playbook (build-context.md), not the wrapper markdown. The wrappers delegate to the playbook via "Playbook Procedure" section.

**Action:** These files likely need NO changes for Phase 74. The build-context.md playbook already orchestrates Step 4.2. The wrappers load the playbook and execute it. If the restored Step 4.2 in build-context.md handles the suggest-analyze call, the wrappers are fine as-is.

**Verify:** Check if the wrappers reference suggest-analyze or if the playbook handles it. Currently the wrappers say "Load `.aether/docs/command-playbooks/build-wave.md`" which loads the wave playbook. The build-context.md is loaded by build-prep.md. The flow is: build.md wrapper -> build-prep.md -> build-context.md -> Step 4.2.

---

### `.codex/CODEX.md` (config, modify)

**Analog:** self

**Key insight:** Codex uses the Go runtime directly (no wrapper markdown). The `aether suggest-analyze` and `aether suggest-approve` commands are already Go CLI commands. Codex build flow calls the Go runtime commands directly.

**Action:** If the Codex build flow references Step 4.2 or suggest-analyze, update those references. Otherwise, no changes needed since Codex uses the Go runtime directly.

---

## Shared Patterns

### Store Initialization Guard
**Source:** `cmd/pheromone_write.go:25-28`
**Apply to:** All new CLI command files
```go
if store == nil {
    outputErrorMessage("no store initialized")
    return nil
}
```

### outputOK JSON Envelope
**Source:** `cmd/helpers.go:18-25`
**Apply to:** All CLI command success paths
```go
func outputOK(result interface{}) {
    resultJSON, err := json.Marshal(result)
    if err != nil {
        outputError(2, fmt.Sprintf("failed to marshal command result: %v", err), nil)
        return
    }
    fmt.Fprintf(stdout, "{\"ok\":true,\"result\":%s}\n", string(resultJSON))
}
```

### outputError JSON Envelope
**Source:** `cmd/helpers.go:28+`
**Apply to:** All CLI command error paths
```go
outputError(1, "error message", nil)
```

### loadActiveColonyState
**Source:** `cmd/state_load.go:17-27`
**Apply to:** Commands that need colony state (suggest-analyze, suggest-approve)
```go
func loadActiveColonyState() (colony.ColonyState, error) {
    if store == nil {
        return colony.ColonyState{}, fmt.Errorf("no store initialized")
    }
    state, err := loadColonyStateWithCompatibilityRepair()
    if err != nil {
        if errors.Is(err, os.ErrNotExist) {
            return colony.ColonyState{}, errNoColonyInitialized
        }
        return colony.ColonyState{}, fmt.Errorf("failed to load colony state: %w", err)
    }
```

### Colony State Save
**Source:** `cmd/pheromone_write.go:188-191`
**Apply to:** suggest-analyze (persisting pending suggestions)
```go
if err := store.SaveJSON("COLONY_STATE.json", pf); err != nil {
    outputError(2, fmt.Sprintf("failed to save pheromones: %v", err), nil)
    return nil
}
```

### Content Hash for Dedup
**Source:** `cmd/pheromone_write.go:87-88, 350-353`
**Apply to:** suggest-analyze (comparing suggestions against active pheromones)
```go
func sha256Sum(s string) string {
    h := sha256.Sum256([]byte(s))
    return hex.EncodeToString(h[:])
}
// Usage:
h := sha256Sum(content)
contentHash := "sha256:" + h
```

### Signal Sanitization
**Source:** `cmd/pheromone_write.go:91-95`
**Apply to:** suggest-analyze (before storing suggestion content)
```go
sanitized, err := colony.SanitizeSignalContent(content)
if err != nil {
    outputError(1, fmt.Sprintf("invalid signal content: %v", err), nil)
    return nil
}
```

### Event Emission (Lifecycle Ceremony)
**Source:** `cmd/pheromone_write.go:204-209`
**Apply to:** suggest-approve (when user approves a suggestion)
```go
emitLifecycleCeremony(events.CeremonyTopicPheromoneEmit, events.CeremonyPayload{
    PheromoneType: sigType,
    Strength:      strength,
    Status:        status,
    Message:       extractText(signal.Content),
}, "aether-pheromone")
```

## No Analog Found

Files with no close match in the codebase (planner should use RESEARCH.md patterns instead):

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| (none) | -- | -- | All files have direct analogs in the codebase |

**Note:** The "build-specific extra patterns" (TODO density, large files, test coverage gaps) are new pattern detectors that do not yet exist. They should follow the same pattern as `generatePheromoneSuggestions()` in `cmd/init_research.go:1308+` -- deterministic file/directory checks using `hasFile()`, `fileContains()`, `hasDir()`, and similar utilities.

## Key Technical Notes for Planner

1. **Function name correction:** The tech stack parsing function is `parseDependencyFiles(target string) []techStackDetail` at `cmd/init_research.go:1238`, NOT `parseTechStack()`.

2. **Same-package access:** All new files in `cmd/` can directly access unexported types (`pheromoneSuggestion`, `governanceInfo`, `dirClassification`, `techStackDetail`) and functions (`generatePheromoneSuggestions`, `detectGovernance`, `classifyDirectory`, `parseDependencyFiles`, `hasFile`, `fileContains`, `sha256Sum`, `generateSignalID`, `loadActiveColonyState`) because they are all in package `cmd`.

3. **Wrapper files may not need changes:** The build wrappers (`.claude/commands/ant/build.md`, `.opencode/commands/ant/build.md`) delegate to the playbook system. The suggest-analyze flow is orchestrated in `build-context.md:147-181`. Restoring Step 4.2 in the playbook should be sufficient.

4. **Non-blocking requirement:** Per RESEARCH.md Pitfall 3, suggest-analyze must return `ok: true` with `suggestions: []` on any error, never `ok: false`. The build must never stop at Step 4.2.

5. **PheromoneFile vs ColonyState storage:** Pheromone signals go in `pheromones.json` via the `PheromoneFile` struct. Pending suggestions (unreviewed) go in `COLONY_STATE.json` via new fields on `ColonyState`. These are two different storage locations with different structs.

## Metadata

**Analog search scope:** `cmd/`, `pkg/colony/`, `.aether/docs/command-playbooks/`, `.claude/commands/ant/`, `.opencode/commands/ant/`, `.codex/`
**Files scanned:** 12
**Pattern extraction date:** 2026-04-29
