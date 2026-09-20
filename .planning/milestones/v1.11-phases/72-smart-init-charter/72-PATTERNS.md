# Phase 72: Smart Init Charter - Pattern Map

**Mapped:** 2026-04-28
**Files analyzed:** 9
**Analogs found:** 9 / 9

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `cmd/init_research.go` | command | request-response | itself (existing) | exact |
| `cmd/init_cmd.go` | command | CRUD (state mutation) | itself (existing) | exact |
| `cmd/init_ceremony.go` | command | event-driven | `cmd/recover_repair.go` | role-match |
| `cmd/init_research_test.go` | test | request-response | itself (existing) | exact |
| `cmd/init_ceremony_test.go` | test | event-driven | `cmd/init_research_test.go` | role-match |
| `cmd/codex_visuals.go` | utility | transform | itself (existing) | exact |
| `cmd/ceremony_emitter.go` | utility | event-driven | itself (existing) | exact |
| `pkg/colony/colony.go` | model | CRUD | itself (existing) | exact |
| `.claude/commands/ant/init.md` | config (wrapper) | request-response | itself (existing) | exact |

## Pattern Assignments

### `cmd/init_research.go` (command, request-response) -- MODIFY

**Analog:** itself -- `cmd/init_research.go`

**Current charterData struct** (lines 48-53):
```go
type charterData struct {
    Intent     string `json:"intent"`
    Vision     string `json:"vision"`
    Governance string `json:"governance"`
    Goals      string `json:"goals"`
}
```

**Action:** Expand to 7 fields (add TechStack, KeyRisks, Constraints). The 3 new sections pull from data already computed: `languages`, `frameworks`, `governance`, `pheromoneSuggestions`, and `isGitRepo` from the existing RunE function (lines 427-571).

**Current generateCharter function** (lines 360-407):
```go
func generateCharter(goal, detected string, governance governanceInfo, readmeSummary string, gitHistory gitHistoryInfo) charterData {
    ch := charterData{}
    // ... fills 4 fields from goal, detected, governance, readmeSummary
    return ch
}
```

**Action:** Add `languages []string`, `frameworks []string`, `isGitRepo bool`, and `pheromoneSuggestions []pheromoneSuggestion` as parameters. Generate 3 new fields:
- `TechStack`: from `languages` + `frameworks` (deduplicate overlapping items)
- `KeyRisks`: from governance gaps (no CI, no tests, no linter, .env without .gitignore, no git repo)
- `Constraints`: from detected governance tools (linter rules, formatter config, test framework, build tool)

**Import pattern** (lines 1-13):
```go
import (
    "io/fs"
    "os"
    "os/exec"
    "path/filepath"
    "sort"
    "strconv"
    "strings"

    "github.com/spf13/cobra"
)
```
No new imports needed -- all required types already in file.

**Output pattern** (lines 553-568):
```go
outputOK(map[string]interface{}{
    "detected_type":         detected,
    "languages":             languages,
    // ... all fields ...
    "charter":               charter,
})
```
The `charter` key already exists in the output map. Its expanded struct will serialize automatically.

**Existing test pattern** (lines 383-418 of `init_research_test.go`):
```go
func TestInitResearchCharter(t *testing.T) {
    saveGlobals(t)
    resetRootCmd(t)
    var buf bytes.Buffer
    stdout = &buf

    s, tmpDir := newTestStore(t)
    defer os.RemoveAll(tmpDir)
    store = s

    target := t.TempDir()
    os.WriteFile(filepath.Join(target, "go.mod"), []byte("module test\n"), 0644)

    rootCmd.SetArgs([]string{"init-research", "--goal", "Build X", "--target", target})

    if err := rootCmd.Execute(); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    env := parseEnvelope(t, buf.String())
    result := env["result"].(map[string]interface{})

    charter := result["charter"].(map[string]interface{})
    if charter["intent"] != "Build X" {
        t.Errorf("charter.intent = %v, want 'Build X'", charter["intent"])
    }
    // ... validates 4 existing fields ...
}
```
**Action:** Expand this test to also verify `tech_stack`, `key_risks`, and `constraints` are non-empty.

---

### `cmd/init_cmd.go` (command, CRUD) -- MODIFY

**Analog:** itself -- `cmd/init_cmd.go`

**Current ColonyState creation** (lines 121-145):
```go
state := colony.ColonyState{
    Version:       "3.0",
    Goal:          &goal,
    Scope:         scope,
    ColonyVersion: 0,
    State:         colony.StateREADY,
    CurrentPhase:  0,
    SessionID:     &sessionID,
    RunID:         &runID,
    InitializedAt: &now,
    Plan:          colony.Plan{Phases: []colony.Phase{}},
    Memory: colony.Memory{
        PhaseLearnings: []colony.PhaseLearning{},
        Decisions:      []colony.Decision{},
        Instincts:      []colony.Instinct{},
    },
    Errors:         colony.Errors{...},
    Signals:        []colony.Signal{},
    Graveyards:     []colony.Graveyard{},
    Events:         []string{},
    ParallelMode:   colony.ModeInRepo,
}
```

**Action:** Add `Charter: charter` to the ColonyState initialization, where `charter` comes from either:
1. `--charter-json` flag (parsed from JSON string for wrapper path), or
2. From `runInitResearchInternal()` (for Go-native ceremony path)

**Flag registration pattern** (lines 218-220):
```go
func init() {
    initCmd.Flags().String("scope", string(colony.ScopeProject), "Colony scope: project or meta")
    rootCmd.AddCommand(initCmd)
}
```

**Action:** Add `--charter-json` flag:
```go
initCmd.Flags().String("charter-json", "", "Approved charter data as JSON string")
```

**Store persistence pattern** (lines 147-149):
```go
if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
    outputError(1, fmt.Sprintf("failed to create COLONY_STATE.json: %v", err), nil)
    return nil
}
```
No change needed -- SaveJSON handles the new Charter field automatically.

**Import pattern** (lines 1-14):
```go
import (
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "time"

    "github.com/calcosmic/Aether/pkg/colony"
    "github.com/calcosmic/Aether/pkg/trace"
    "github.com/spf13/cobra"
)
```
Will need `encoding/json` for parsing `--charter-json`.

---

### `cmd/init_ceremony.go` (command, event-driven) -- NEW

**Analog:** `cmd/recover_repair.go` (stdin prompt pattern)

**Terminal prompt pattern** (lines 155-167 of `recover_repair.go`):
```go
func confirmRepair(issue HealthIssue) bool {
    fmt.Fprintf(os.Stderr, "\n  [confirm] %s (%s)\n", issue.Message, issue.File)
    fmt.Fprintf(os.Stderr, "  Apply fix? [y/N]: ")

    reader := bufio.NewReader(os.Stdin)
    response, err := reader.ReadString('\n')
    if err != nil {
        return false
    }

    trimmed := strings.TrimSpace(strings.ToLower(response))
    return trimmed == "y" || trimmed == "yes"
}
```

**Extension for numbered-list approval** (proposed pattern from RESEARCH.md):
```go
func promptNumberedChoice(question string, options []string) int {
    fmt.Fprintf(os.Stderr, "\n%s\n", question)
    for i, opt := range options {
        fmt.Fprintf(os.Stderr, "  %d. %s\n", i+1, opt)
    }
    fmt.Fprintf(os.Stderr, "\n  Choice [1-%d]: ", len(options))

    reader := bufio.NewReader(os.Stdin)
    response, _ := reader.ReadString('\n')
    trimmed := strings.TrimSpace(response)
    n, _ := strconv.Atoi(trimmed)
    if n < 1 || n > len(options) {
        return 0 // invalid
    }
    return n
}
```

**Imports needed** (from `recover_repair.go` lines 1-15):
```go
import (
    "bufio"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "github.com/calcosmic/Aether/pkg/colony"
    "github.com/spf13/cobra"
)
```

**Command registration pattern** (from `init_research.go` lines 423-577):
```go
var initCeremonyCmd = &cobra.Command{
    Use:   "init-ceremony",
    Short: "Run the full colony init ceremony (scan, charter, approve)",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        // ... ceremony logic ...
    },
}

func init() {
    initCeremonyCmd.Flags().String("target", "", "Directory to scan (default: current directory)")
    initCeremonyCmd.Flags().String("scope", string(colony.ScopeProject), "Colony scope")
    rootCmd.AddCommand(initCeremonyCmd)
}
```

**Visual output pattern** (from `codex_visuals.go` lines 423-448, `renderInitVisual`):
```go
func renderInitVisual(goal, scope, sessionID, dataDir string) string {
    var b strings.Builder
    b.WriteString(renderBanner(commandEmoji("init"), "Colony Init"))
    b.WriteString(visualDivider)
    b.WriteString(renderStageMarker("Colony"))
    b.WriteString("Queen charter accepted.\n")
    b.WriteString("Goal: ")
    b.WriteString(goal)
    b.WriteString("\n")
    // ...
    b.WriteString(renderNextUp(...))
    b.WriteString(renderContextClearGuidance())
    return b.String()
}
```

**Ceremony emitter pattern** (from `ceremony_emitter.go` lines 115-130):
```go
func emitLifecycleCeremony(topic string, payload events.CeremonyPayload, source string) {
    if store == nil || strings.TrimSpace(topic) == "" {
        return
    }
    source = strings.TrimSpace(source)
    if source == "" {
        source = "aether"
    }
    payload = trimCeremonyPayload(payload)
    raw, err := payload.RawMessage()
    if err != nil {
        return
    }
    bus := events.NewBus(store, events.DefaultConfig())
    _, _ = bus.Publish(context.Background(), topic, raw, source)
}
```

**Output envelope pattern** (from `helpers.go` lines 18-25):
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

---

### `cmd/init_research_test.go` (test, request-response) -- MODIFY

**Analog:** itself -- `cmd/init_research_test.go`

**Test helper pattern** (from `testing_main_test.go` lines 72-148):
```go
func saveGlobals(t *testing.T) {
    t.Helper()
    origStore := store
    // ... capture all globals ...
    t.Cleanup(func() {
        store = origStore
        // ... restore all globals ...
    })
}

func resetRootCmd(t *testing.T) {
    t.Helper()
    t.Cleanup(func() {
        rootCmd.SetArgs([]string{})
        rootCmd.SetOut(os.Stdout)
        resetFlags(rootCmd)
    })
}
```

**Test structure pattern** (from `init_research_test.go` lines 383-418):
```go
func TestInitResearchCharter(t *testing.T) {
    saveGlobals(t)
    resetRootCmd(t)
    var buf bytes.Buffer
    stdout = &buf

    s, tmpDir := newTestStore(t)
    defer os.RemoveAll(tmpDir)
    store = s

    target := t.TempDir()
    // ... set up test fixtures ...

    rootCmd.SetArgs([]string{"init-research", "--goal", "...", "--target", target})
    if err := rootCmd.Execute(); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    env := parseEnvelope(t, buf.String())
    result := env["result"].(map[string]interface{})
    charter := result["charter"].(map[string]interface{})
    // ... assertions ...
}
```

**Action:** Add test cases for the 3 new charter fields:
- `TestInitResearchCharterTechStack` -- verify `tech_stack` is non-empty when `go.mod` is present
- `TestInitResearchCharterKeyRisks` -- verify `key_risks` contains expected risk when no CI config exists
- `TestInitResearchCharterConstraints` -- verify `constraints` is non-empty when linter/formatter configs exist

---

### `cmd/init_ceremony_test.go` (test, event-driven) -- NEW

**Analog:** `cmd/init_research_test.go`

**Test structure to follow** (same pattern as above):
- `saveGlobals(t)` / `resetRootCmd(t)` setup
- `newTestStore(t)` for store initialization
- `rootCmd.SetArgs([]string{...})` for command arguments
- `parseEnvelope(t, buf.String())` for output parsing

**Specific test cases needed:**
1. `TestInitCeremonyProceed` -- mock stdin with "1", verify COLONY_STATE.json created with charter
2. `TestInitCeremonyRevise` -- mock stdin with "2" then new goal, verify re-research happens
3. `TestInitCeremonyCancel` -- mock stdin with "3", verify no COLONY_STATE.json created
4. `TestInitCeremonyCharterJSONFlag` -- verify `--charter-json` flag stores charter in state

---

### `cmd/codex_visuals.go` (utility, transform) -- MODIFY

**Analog:** itself -- `cmd/codex_visuals.go`

**Visual rendering pattern** (lines 423-448):
```go
func renderInitVisual(goal, scope, sessionID, dataDir string) string {
    var b strings.Builder
    b.WriteString(renderBanner(commandEmoji("init"), "Colony Init"))
    b.WriteString(visualDivider)
    b.WriteString(renderStageMarker("Colony"))
    // ... content sections ...
    b.WriteString(renderNextUp(...))
    b.WriteString(renderContextClearGuidance())
    return b.String()
}
```

**Helper functions to use** (from same file):
- `renderBanner(emoji, title)` -- line 259: `fmt.Sprintf("━━ %s %s ━━\n", emoji, spacedTitle(title))`
- `renderStageMarker(title)` -- line 274: `"── " + title + " ──\n"`
- `visualDivider` -- line 15: `"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"`
- `commandEmoji(command)` -- line 174: maps command name to emoji
- `emptyFallback(value, fallback)` -- line 2452: returns fallback if value is empty

**Action:** Add `renderCharterDisplay(charter)` function for Go-native ceremony to display all 7 charter sections with stage markers and ANSI formatting.

---

### `cmd/ceremony_emitter.go` (utility, event-driven) -- MODIFY

**Analog:** itself -- `cmd/ceremony_emitter.go`

**Emit pattern** (lines 115-130):
```go
func emitLifecycleCeremony(topic string, payload events.CeremonyPayload, source string) {
    if store == nil || strings.TrimSpace(topic) == "" {
        return
    }
    // ... publish to event bus ...
}
```

**Action:** Emit init ceremony events at key points:
- Colony scanned
- Charter generated
- Charter approved
- Colony initialized

---

### `pkg/colony/colony.go` (model, CRUD) -- MODIFY

**Analog:** itself -- `pkg/colony/colony.go`

**Current ColonyState struct** (lines 159-187):
```go
type ColonyState struct {
    Version            string          `json:"version"`
    Goal               *string         `json:"goal"`
    Scope              ColonyScope     `json:"scope,omitempty"`
    // ... many fields ...
    RunID              *string            `json:"run_id,omitempty"`
    GateResults        []GateResultEntry  `json:"gate_results,omitempty"`
}
```

**Action:** Add Charter field:
```go
type ColonyState struct {
    // ... existing fields ...
    Charter            *Charter           `json:"charter,omitempty"`
    // ... rest of existing fields ...
}
```

**New Charter struct** (to be added near the ColonyState definition):
```go
type Charter struct {
    Intent      string `json:"intent"`
    Vision      string `json:"vision"`
    Governance  string `json:"governance"`
    Goals       string `json:"goals"`
    TechStack   string `json:"tech_stack"`
    KeyRisks    string `json:"key_risks"`
    Constraints string `json:"constraints"`
}
```

**Backward compatibility note:** Using `*Charter` with `omitempty` ensures old COLONY_STATE.json files without `charter` unmarshal to nil. All downstream code must nil-check before accessing fields.

**Test pattern** (from `colony_test.go` lines 1-35):
```go
package colony

import (
    "bytes"
    "encoding/json"
    "errors"
    "os"
    "testing"
    "time"
)

func TestValidTransitions(t *testing.T) {
    tests := []struct {
        from State
        to   State
    }{
        {StateREADY, StateEXECUTING},
        // ...
    }
    for _, tt := range tests {
        t.Run(string(tt.from)+"->"+string(tt.to), func(t *testing.T) {
            if err := Transition(tt.from, tt.to); err != nil {
                t.Fatalf("expected no error for %s->%s, got: %v", tt.from, tt.to)
            }
        })
    }
}
```

**Action:** Add `TestCharterRoundTrip` in `colony_test.go` -- marshal ColonyState with Charter, unmarshal, verify fields preserved.

---

### `.claude/commands/ant/init.md` (config/wrapper, request-response) -- MODIFY

**Analog:** itself -- `.claude/commands/ant/init.md`

**Current charter display** (lines 26-32):
```markdown
## Colony Charter

Present the charter for user review:

```
**Intent:** {charter.intent}
**Vision:** {charter.vision}
**Governance:** {charter.governance}
**Goals:** {charter.goals}
```
```

**Action:** Expand to 7 sections:
```markdown
**Intent:** {charter.intent}
**Vision:** {charter.vision}
**Governance:** {charter.governance}
**Goals:** {charter.goals}
**Tech Stack:** {charter.tech_stack}
**Key Risks:** {charter.key_risks}
**Constraints:** {charter.constraints}
```

**Current init invocation** (line 74):
```markdown
- Then run `AETHER_OUTPUT_MODE=visual aether init "$ARGUMENTS"`.
```

**Action:** Change to pass charter data:
```markdown
- Then run `AETHER_OUTPUT_MODE=visual aether init --charter-json '<charter JSON>' "$ARGUMENTS"`.
```

---

### `.opencode/commands/ant/init.md` (config/wrapper, request-response) -- MODIFY

**Analog:** `.claude/commands/ant/init.md` (currently identical)

The two wrapper files are currently identical. Changes made to `.claude/commands/ant/init.md` must be mirrored here. The only difference is line 50 where Claude uses "AskUserQuestion" and OpenCode uses "Ask with 3 options".

**Action:** Mirror all charter display and init invocation changes from the Claude wrapper.

---

## Shared Patterns

### JSON Output Envelope
**Source:** `cmd/helpers.go` lines 18-57
**Apply to:** All Go commands (`init_research.go`, `init_cmd.go`, `init_ceremony.go`)
```go
func outputOK(result interface{}) {
    resultJSON, err := json.Marshal(result)
    if err != nil {
        outputError(2, fmt.Sprintf("failed to marshal command result: %v", err), nil)
        return
    }
    fmt.Fprintf(stdout, "{\"ok\":true,\"result\":%s}\n", string(resultJSON))
}

func outputError(code int, message string, details interface{}) {
    if shouldRenderVisualOutput(stderr) {
        fmt.Fprint(stderr, renderVisualError(message, details))
        return
    }
    envelope := struct {
        OK      bool        `json:"ok"`
        Error   string      `json:"error"`
        Code    int         `json:"code"`
        Details interface{} `json:"details,omitempty"`
    }{OK: false, Error: message, Code: code}
    // ... marshal and write to stderr ...
}
```

### Visual Output Rendering
**Source:** `cmd/codex_visuals.go` lines 217-226
**Apply to:** `init_ceremony.go`, `codex_visuals.go` (new render function)
```go
func outputWorkflow(result interface{}, visual string) {
    if shouldRenderVisualOutput(stdout) {
        if !strings.HasSuffix(visual, "\n") {
            visual += "\n"
        }
        writeVisualOutput(stdout, visual)
        return
    }
    outputOK(result)
}
```

### State Persistence via store.SaveJSON
**Source:** `cmd/init_cmd.go` lines 147-149
**Apply to:** `init_cmd.go` (no change needed), `init_ceremony.go`
```go
if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
    outputError(1, fmt.Sprintf("failed to create COLONY_STATE.json: %v", err), nil)
    return nil
}
```

### Test Helpers
**Source:** `cmd/testing_main_test.go` lines 72-148, `cmd/write_cmds_test.go` lines 21-58
**Apply to:** `init_ceremony_test.go`
```go
// Setup: saveGlobals(t), resetRootCmd(t), newTestStore(t), defer os.RemoveAll(tmpDir)
// Execute: rootCmd.SetArgs([]string{...}), rootCmd.Execute()
// Assert: parseEnvelope(t, buf.String()), field checks
```

### Terminal Input via bufio.Reader
**Source:** `cmd/recover_repair.go` lines 155-167
**Apply to:** `init_ceremony.go` (numbered-list prompts)
```go
reader := bufio.NewReader(os.Stdin)
response, err := reader.ReadString('\n')
if err != nil {
    return false
}
trimmed := strings.TrimSpace(strings.ToLower(response))
```

### Lifecycle Ceremony Emission
**Source:** `cmd/ceremony_emitter.go` lines 115-130
**Apply to:** `init_ceremony.go`
```go
emitLifecycleCeremony("colony:init:scanned", payload, "aether")
emitLifecycleCeremony("colony:init:charter-approved", payload, "aether")
```

## No Analog Found

All files have close analogs in the codebase. No files require patterns from RESEARCH.md alone.

| File | Reason |
|------|--------|
| (none) | All files map to existing codebase patterns |

## Metadata

**Analog search scope:** `cmd/`, `pkg/colony/`, `.claude/commands/ant/`, `.opencode/commands/ant/`
**Files scanned:** 12
**Pattern extraction date:** 2026-04-28
