# Phase 73: Rich Init Research - Pattern Map

**Mapped:** 2026-04-28
**Files analyzed:** 5
**Analogs found:** 5 / 5

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `cmd/init_research.go` | command (extend) | file-I/O + transform | `cmd/init_research.go` (itself) | exact |
| `cmd/init_research_test.go` | test (extend) | request-response | `cmd/init_research_test.go` (itself) | exact |
| `cmd/init_ceremony.go` | command (minor) | request-response | `cmd/init_ceremony.go` (itself) | exact |
| `cmd/codex_visuals.go` | utility (minor) | transform | `cmd/codex_visuals.go` (itself) | exact |
| `pkg/colony/colony.go` | model (minor) | data-structure | `pkg/colony/colony.go` (itself) | exact |

Note: This phase is purely additive to existing files. No new files are created. All work extends established patterns within the same files.

## Pattern Assignments

### `cmd/init_research.go` (command, file-I/O + transform)

**Analog:** `cmd/init_research.go` (self-referential -- extending existing code)

**Struct type pattern** (lines 18-47):
All scan result types are plain structs with JSON tags, defined at the top of the file. New types follow the same convention.
```go
type gitHistoryInfo struct {
    Commits      int    `json:"commits"`
    Contributors int    `json:"contributors"`
    Branch       string `json:"branch,omitempty"`
}

type governanceInfo struct {
    Linters        []string `json:"linters"`
    Formatters     []string `json:"formatters"`
    TestFrameworks []string `json:"test_frameworks"`
    CIConfigs      []string `json:"ci_configs"`
    BuildTools     []string `json:"build_tools"`
}

type pheromoneSuggestion struct {
    Type    string `json:"type"`
    Content string `json:"content"`
    Reason  string `json:"reason"`
}
```

**File existence check helpers** (lines 226-239):
`hasFile()` and `fileContains()` are the canonical utilities for all detection logic. New parser functions should use these.
```go
func hasFile(target, name string) bool {
    _, err := os.Stat(filepath.Join(target, name))
    return err == nil
}

func fileContains(target, name, substr string) bool {
    data, err := os.ReadFile(filepath.Join(target, name))
    if err != nil {
        return false
    }
    return strings.Contains(string(data), substr)
}
```

**Tolerant governance detection pattern** (lines 119-165):
`detectGovernance()` demonstrates the "never fail on individual files" pattern. It uses `os.Stat` checks, deduplicates by label, and appends to category slices. New governance deep-parsing functions should follow this same tolerant approach -- return nil/empty on any parse error, never propagate.
```go
func detectGovernance(target string) governanceInfo {
    info := governanceInfo{}
    seen := make(map[string]string) // label -> category for dedup

    for _, det := range governanceDetectors {
        p := filepath.Join(target, det.file)
        if _, err := os.Stat(p); err == nil {
            key := det.label
            if _, exists := seen[key]; !exists {
                seen[key] = det.category
                switch det.category {
                case "linter":
                    info.Linters = append(info.Linters, det.label)
                // ... other categories
                }
            }
        }
    }
    return info
}
```

**Pheromone suggestion generation pattern** (lines 242-351):
`generatePheromoneSuggestions()` uses sequential `hasFile()`/`fileContains()` checks, each producing a `pheromoneSuggestion` struct. The new patterns (expanding from 10 to ~25) should be appended as additional sequential checks in the same function. No data-driven registry or interface abstraction -- keep it flat with comment grouping.
```go
func generatePheromoneSuggestions(target string, governance governanceInfo) []pheromoneSuggestion {
    var suggestions []pheromoneSuggestion

    // 1. .env or .env.local exists -> REDIRECT about secrets
    if hasFile(target, ".env") || hasFile(target, ".env.local") {
        suggestions = append(suggestions, pheromoneSuggestion{
            Type:    "REDIRECT",
            Content: "never commit secrets or .env files to version control",
            Reason:  ".env file detected in project root",
        })
    }
    // ... more sequential checks

    return suggestions
}
```

**Output envelope pattern** (lines 639-654):
All new analysis results must be added inside the `outputOK()` result map. Never change the envelope structure `{"ok":true,"result":{...}}`. New fields are additive only.
```go
outputOK(map[string]interface{}{
    "detected_type":         detected,
    "languages":             languages,
    "frameworks":            frameworks,
    "goal":                  goal,
    "top_level_dirs":        topLevelDirs,
    "file_count":            fileCount,
    "is_git_repo":           isGitRepo,
    "readme_summary":        readmeSummary,
    "git_history":           gitHistory,
    "governance":            governance,
    "complexity":            complexity,
    "prior_colonies":        priorColonies,
    "pheromone_suggestions": pheromoneSuggestions,
    "charter":               charter,
})
```

**Import block pattern** (lines 1-14):
Current imports include `io/fs`, `os`, `os/exec`, `path/filepath`, `sort`, `strconv`, `strings`, plus `colony` and `cobra` packages. New dependency parsers will need `encoding/json`, `encoding/xml` (for pom.xml), `regexp` (for Gemfile), and `github.com/BurntSushi/toml` (for Cargo.toml). The `gjson` and `yaml.v3` packages are already in go.mod but not yet imported in this file.
```go
import (
    "io/fs"
    "os"
    "os/exec"
    "path/filepath"
    "sort"
    "strconv"
    "strings"

    "github.com/calcosmic/Aether/pkg/colony"
    "github.com/spf13/cobra"
)
```

---

### `cmd/init_research_test.go` (test, request-response)

**Analog:** `cmd/init_research_test.go` (self-referential)

**Test scaffolding pattern** (lines 11-49):
Every test follows the same 4-step setup: `saveGlobals(t)` -> `resetRootCmd(t)` -> buffer capture -> `newTestStore(t)`. Tests create fixture files in temp dirs, execute the command via `rootCmd.SetArgs(...)`, then parse the JSON envelope with `parseEnvelope(t, buf.String())`.
```go
func TestInitResearchGo(t *testing.T) {
    saveGlobals(t)
    resetRootCmd(t)
    var buf bytes.Buffer
    stdout = &buf

    s, tmpDir := newTestStore(t)
    defer os.RemoveAll(tmpDir)
    store = s

    projectRoot := filepath.Dir(filepath.Dir(s.BasePath()))
    os.WriteFile(filepath.Join(projectRoot, "go.mod"), []byte("module test\n"), 0644)

    rootCmd.SetArgs([]string{"init-research", "--goal", "build CLI", "--target", projectRoot})

    err := rootCmd.Execute()
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    env := parseEnvelope(t, buf.String())
    if env["ok"] != true {
        t.Fatalf("expected ok:true, got: %v", env["ok"])
    }

    result := env["result"].(map[string]interface{})
    // assertions on result fields...
}
```

**Fixture file creation pattern** (lines 184-189):
Tests create temporary files using `t.TempDir()` and `os.WriteFile()`/`os.MkdirAll()`. For dependency parsing tests, create the actual dependency files (e.g., `package.json`, `go.mod`) with known content.
```go
target := t.TempDir()
os.MkdirAll(filepath.Join(target, "src", "pkg"), 0755)
os.WriteFile(filepath.Join(target, "src", "main.go"), []byte("package main"), 0644)
os.WriteFile(filepath.Join(target, "go.mod"), []byte("module test\n"), 0644)
```

**Result field assertion pattern** (lines 31-48):
Assertions access nested fields via type assertions on the result map. The pattern is `result["field"].(concreteType)` with `%v` formatting in error messages.
```go
result := env["result"].(map[string]interface{})
if result["detected_type"] != "go" {
    t.Errorf("detected_type = %v, want go", result["detected_type"])
}
if result["file_count"].(float64) < 1 {
    t.Errorf("file_count = %v, want >= 1", result["file_count"])
}
```

**Governance test pattern** (lines 280-339):
Governance tests create config files and verify category arrays contain expected tool names.
```go
governance := result["governance"].(map[string]interface{})
linters := governance["linters"].([]interface{})
foundESLint := false
for _, l := range linters {
    if l == "ESLint" {
        foundESLint = true
    }
}
if !foundESLint {
    t.Errorf("linters = %v, want to contain ESLint", linters)
}
```

**Test helper locations:**
- `saveGlobals(t)` -- `cmd/testing_main_test.go` line 72
- `resetRootCmd(t)` -- `cmd/testing_main_test.go` line 121
- `newTestStore(t)` -- `cmd/write_cmds_test.go` line 21
- `parseEnvelope(t, output)` -- `cmd/write_cmds_test.go` line 39
- `runGit(t, dir, args...)` -- `cmd/init_cmd_test.go` line 845

---

### `cmd/init_ceremony.go` (command, request-response)

**Analog:** `cmd/init_ceremony.go` (self-referential)

**Research output consumption pattern** (lines 180-259):
`runCeremonyResearch()` captures init-research JSON output by redirecting stdout, parses the envelope, then extracts specific fields. New output fields (tech_stack_detail, dir_classification, governance_details) can be extracted here but Phase 73 may defer wrapper rendering to keep scope tight.
```go
func runCeremonyResearch(goal, target string) (*colony.Charter, []pheromoneSuggestion, error) {
    // ... redirect stdout, run initResearchCmd.RunE, parse envelope ...

    researchResult, _ = envelope["result"].(map[string]interface{})

    // Extract charter
    charterMap, ok := researchResult["charter"].(map[string]interface{})
    if !ok {
        return nil, nil, fmt.Errorf("no charter in init-research output")
    }
    charter := extractCharterFromMap(charterMap)

    // Extract pheromone suggestions
    var suggestions []pheromoneSuggestion
    if sugList, ok := researchResult["pheromone_suggestions"].([]interface{}); ok {
        for _, s := range sugList {
            if sm, ok := s.(map[string]interface{}); ok {
                suggestions = append(suggestions, pheromoneSuggestion{
                    Type:    stringOrEmpty(sm["type"]),
                    Content: stringOrEmpty(sm["content"]),
                    Reason:  stringOrEmpty(sm["reason"]),
                })
            }
        }
    }
    return &charter, suggestions, nil
}
```

**Charter extraction pattern** (lines 262-272):
```go
func extractCharterFromMap(m map[string]interface{}) colony.Charter {
    return colony.Charter{
        Intent:      stringOrEmpty(m["intent"]),
        Vision:      stringOrEmpty(m["vision"]),
        Governance:  stringOrEmpty(m["governance"]),
        Goals:       stringOrEmpty(m["goals"]),
        TechStack:   stringOrEmpty(m["tech_stack"]),
        KeyRisks:    stringOrEmpty(m["key_risks"]),
        Constraints: stringOrEmpty(m["constraints"]),
    }
}
```

---

### `cmd/codex_visuals.go` (utility, transform)

**Analog:** `cmd/codex_visuals.go` (self-referential)

**Charter rendering pattern** (lines 450-479):
`renderCharterDisplay()` renders the 7-section charter using `renderBanner()`, `visualDivider`, `renderStageMarker()`, and `emptyFallback()`. If new charter sections are added, they would be rendered here using the same pattern.
```go
func renderCharterDisplay(ch colony.Charter) string {
    var b strings.Builder
    b.WriteString(renderBanner(commandEmoji("init"), "Colony Charter"))
    b.WriteString(visualDivider)
    b.WriteString(renderStageMarker("Charter"))
    b.WriteString("  Intent:      ")
    b.WriteString(emptyFallback(ch.Intent, "(none)"))
    b.WriteString("\n")
    // ... 6 more sections ...
    b.WriteString(visualDivider)
    return b.String()
}
```

---

### `pkg/colony/colony.go` (model, data-structure)

**Analog:** `pkg/colony/colony.go` (self-referential)

**Charter struct** (lines 160-168):
The Charter has 7 string fields with JSON tags. Phase 73 should NOT modify this struct (TechStack stays as a summary string for backward compatibility). New structured data goes into the init-research output map, not into the Charter struct.
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

---

## Shared Patterns

### Output Envelope (all command files)
**Source:** `cmd/helpers.go` lines 18-25
**Apply to:** All new output from `init_research.go`
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

### Tolerant File Parsing (all parser functions)
**Source:** `cmd/init_research.go` lines 119-165
**Apply to:** All new dependency parsers, governance deep parsers
- Use `os.ReadFile()` and return nil/empty on error
- Never propagate parse errors to the caller
- A single malformed config must not stop the scan
- Log nothing -- silently skip unparseable files

### gjson Path-Based JSON Extraction (dependency/governance parsers)
**Source:** `cmd/state_cmds.go` lines 483-507
**Apply to:** package.json parsing, .eslintrc.json parsing, any JSON config extraction
```go
arr := gjson.GetBytes(data, arrayPath)
if !arr.IsArray() {
    return nil, fmt.Errorf("path %q is not an array", arrayPath)
}
for _, item := range arr.Array() {
    // process each item
}
```

### Sequential Detection Pattern (pheromone generation)
**Source:** `cmd/init_research.go` lines 242-351
**Apply to:** All 25 pheromone suggestion checks
- Each check is a simple `if hasFile(...)` or `if fileContains(...)` guard
- Produces a single `pheromoneSuggestion` struct
- Group related checks with comments (secrets, CI, formatting, etc.)
- No interface, no registry, no config file -- flat sequential checks

### Test Scaffolding (all test functions)
**Source:** `cmd/init_research_test.go` lines 11-49
**Apply to:** All new test functions
- `saveGlobals(t)` as first line
- `resetRootCmd(t)` as second line
- Buffer capture for stdout/stderr
- `newTestStore(t)` for store initialization
- `t.TempDir()` for fixture directories
- `parseEnvelope(t, buf.String())` for output parsing

## No Analog Found

None -- all files to be modified are existing files with exact self-analogs. This phase is purely additive extension of established code.

## Metadata

**Analog search scope:** `cmd/` directory (primary), `pkg/colony/` (Charter struct)
**Files scanned:** 6 (init_research.go, init_research_test.go, init_ceremony.go, codex_visuals.go, colony.go, state_cmds.go, helpers.go, testing_main_test.go)
**Pattern extraction date:** 2026-04-28
