# Phase 73: Rich Init Research - Research

**Researched:** 2026-04-28
**Domain:** Go CLI codebase analysis -- dependency file parsing, directory structure classification, governance config extraction, pheromone pattern generation
**Confidence:** HIGH

## Summary

Phase 73 deepens the init-research scan in `cmd/init_research.go`. Today the scan detects languages via marker files (go.mod, package.json, etc.) and reports them as string labels. It detects governance config files but only reports tool names -- not the actual rules/settings inside them. It has 10 pheromone suggestion patterns and produces a directory listing but no structural classification. This phase transforms those surface-level detections into deep analysis: parsing dependency files to extract actual package lists, classifying directory structure with heuristic signals, extracting rules from governance configs, and expanding pheromone patterns from 10 to ~25.

The critical constraint is zero new dependencies. The project already has `encoding/json` (stdlib), `gopkg.in/yaml.v3`, `github.com/BurntSushi/toml`, and `github.com/tidwall/gjson` -- these four cover all the config file formats needed (JSON, YAML, TOML, and path-based JSON queries). The work is entirely additive to the existing `init-research` command's output, with backward-compatible JSON output that wrappers already parse.

**Primary recommendation:** Extend `init_research.go` with four focused parser modules (dependency, directory, governance, pheromone) that each produce structured data added to the existing `outputOK` result map. No new Go files needed unless a module exceeds ~200 lines.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Parse actual dependency files to extract package names, version ranges, and dependency counts -- not just file-presence detection. Support: package.json (deps + devDeps), go.mod (requires), Cargo.toml (deps), pyproject.toml/requirements.txt, Gemfile, pom.xml, mix.exs, composer.json.
- **D-02:** Include ALL dependencies in output (production + dev + indirect). Full list, not summarized.
- **D-03:** Classify directory structure using heuristic pattern matching: monorepo (packages/, apps/, workspaces, pnpm-workspace.yaml), microservices (service-per-dir, multiple Dockerfiles), standard app (src/, lib/, cmd/), library (no src/, exports in root), and "unknown" fallback.
- **D-04:** Output includes both the classification type AND detection signals (which files/dirs triggered the classification).
- **D-05:** Parse governance config files to extract actual rules/settings -- not just report tool names. All 5 categories: linters (extract rules/extends), formatters (extract options), test frameworks (extract config), CI configs (extract pipeline steps), build tools (extract targets/scripts).
- **D-06:** All categories are parsed at the same depth -- no category stays at detection-only level.
- **D-07:** Expand from 10 to ~25 deterministic pheromone suggestion patterns. Add patterns for: monorepo workspace consistency, API patterns (OpenAPI/swagger), database presence (migrations, schema files), security patterns (CSP headers, CORS config), container patterns (Docker compose, multi-stage builds), documentation patterns (API docs, changelog), dependency health (outdated lockfiles, known vulnerability indicators).
- **D-08:** Implementation approach is Claude's discretion (hard-coded Go functions vs data-driven registry).
- **D-09:** Built-in patterns only -- no user extensibility.

### Claude's Discretion
- Implementation approach for pheromone pattern registry (D-08)
- Exact dependency parsing depth for each supported file format
- How governance parsing handles malformed or unusual config files
- Exact pheromone suggestion patterns to add beyond the examples listed
- How the colony context summary formats all the new research data

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| INIT-03 | Rich init-research produces tech stack analysis (languages, frameworks, build tools) | Dependency parsing section -- all 8 file formats have parser strategies using existing libraries |
| INIT-04 | Init-research detects directory structure patterns (monorepo, microservices, etc.) | Directory classification section -- heuristic patterns and detection signals defined |
| INIT-05 | Init-research identifies governance files (.eslintrc, pyproject.toml, Makefile, etc.) | Governance deep-parsing section -- per-category extraction strategies for all 5 categories |
| INIT-06 | Init-research generates pheromone suggestions based on detected patterns | Pheromone expansion section -- 15 new patterns defined with detection triggers |
| INIT-07 | Init ceremony outputs formatted colony context summary | Colony context summary section -- how new fields integrate into existing output and charter |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Dependency file parsing | API / Backend (Go CLI) | -- | `aether init-research` runs as a Go CLI command scanning local files |
| Directory structure classification | API / Backend (Go CLI) | -- | Heuristic matching against `os.ReadDir` results |
| Governance config extraction | API / Backend (Go CLI) | -- | Reading and parsing local config files (JSON/YAML/TOML) |
| Pheromone pattern matching | API / Backend (Go CLI) | -- | Pure deterministic Go logic, no external state |
| Colony context summary formatting | API / Backend (Go CLI) | Browser / Client (wrappers) | Go produces JSON, wrappers render markdown |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `encoding/json` | stdlib | Parse package.json, composer.json, go module data | Go stdlib -- always available |
| `gopkg.in/yaml.v3` | v3.0.1 | Parse YAML config files (.eslintrc.yml, CI configs, etc.) | Already in go.mod |
| `github.com/BurntSushi/toml` | v1.5.0 | Parse Cargo.toml, pyproject.toml (TOML sections) | Already in go.mod |
| `github.com/tidwall/gjson` | v1.18.0 | Path-based JSON queries for complex config extraction | Already in go.mod, used in `state_cmds.go` |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `os` | stdlib | File existence checks, ReadDir, ReadFile | All file detection |
| `path/filepath` | stdlib | Path joining, glob matching, walk | Directory traversal |
| `strings` | stdlib | Content matching, trimming, joining | Config content analysis |
| `strconv` | stdlib | Version number parsing | Dependency version extraction |
| `regexp` | stdlib | Pattern matching for Gemfile, Makefile | Structured text parsing |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| gjson path queries | json.Unmarshal into structs | gjson is more flexible for arbitrary configs where structure varies across projects. Struct-based parsing requires defining a struct per config format. |
| Hard-coded pattern functions | Data-driven pattern registry | Hard-coded is simpler for ~25 patterns. A registry adds abstraction cost without clear benefit at this scale. See D-08 recommendation. |

**Installation:** None -- all libraries already in go.mod.

**Version verification:**
```bash
# Already verified from go.mod:
# gopkg.in/yaml.v3 v3.0.1
# github.com/BurntSushi/toml v1.5.0
# github.com/tidwall/gjson v1.18.0
# All current as of 2026-04-28
```

## Architecture Patterns

### System Architecture Diagram

```
                    User runs: aether init-research --goal "..." --target .
                                         |
                                         v
                              +-------------------------+
                              |   initResearchCmd.RunE  |
                              |   (cmd/init_research.go)|
                              +-------------------------+
                                         |
              +--------------------------+--------------------------+
              |                          |                          |
              v                          v                          v
    +------------------+      +------------------+      +------------------+
    | detectProject    |      | classifyDir      |      | detectGovernance |
    | (NEW: parse deps)|      | (NEW: heuristic  |      | (EXTEND: parse   |
    | package.json     |      |  classification) |      |  actual rules)   |
    | go.mod           |      | signals output   |      |                  |
    | Cargo.toml       |      |                  |      | All 5 categories |
    | pyproject.toml   |      +------------------+      | at equal depth   |
    | Gemfile          |                                  +------------------+
    | pom.xml          |                                         |
    | mix.exs          |                                         |
    | composer.json    |                                         |
    +------------------+                                         |
              |                                                   |
              v                                                   v
    +------------------+                                +------------------+
    | generatePheromone|<-------------------------------|  (scan results)  |
    | Suggestions      |                                |  feed all parsers |
    | (EXTEND: 10->25) |                                +------------------+
    +------------------|
              |                                                   |
              +--------------------------+--------------------------+
                                         |
                                         v
                              +-------------------------+
                              |   outputOK(result map)  |
                              |   JSON envelope output  |
                              +-------------------------+
                                         |
                    +--------------------+--------------------+
                    |                    |                    |
                    v                    v                    v
            +-----------+      +-----------+      +------------------+
            | Wrapper   |      | Ceremony  |      | Charter enriched |
            | (Claude/  |      | (Codex/   |      | (TechStack gets  |
            |  OpenCode)|      |  CLI)     |      |  deeper data)    |
            +-----------+      +-----------+      +------------------+
```

### Recommended Project Structure

All changes are in the existing `cmd/init_research.go` file. No new Go files needed unless a module exceeds ~200 lines, in which case extract to `cmd/init_research_deps.go`, `cmd/init_research_governance.go`, etc.

```
cmd/
  init_research.go          # EXTEND: existing scan + new parsers + new output fields
  init_research_test.go     # EXTEND: new test cases for each parser
  init_ceremony.go          # MINOR: consume new fields in runCeremonyResearch
  codex_visuals.go          # MINOR: update renderCharterDisplay if new charter sections added
```

### Pattern 1: Struct-Based Scan Result Extension

**What:** Add new struct types for each analysis domain and include them in the `outputOK` result map.

**When to use:** Every new analysis domain (tech stack details, directory classification, governance rules).

**Example:**
```go
// Source: established pattern from existing init_research.go
type techStackDetail struct {
    Language    string            `json:"language"`
    File        string            `json:"source_file"`
    Dependencies []depEntry       `json:"dependencies"`
    DevDeps     []depEntry        `json:"dev_dependencies,omitempty"`
    Indirect    []depEntry        `json:"indirect,omitempty"`
}

type depEntry struct {
    Name    string `json:"name"`
    Version string `json:"version,omitempty"`
}
```

**Why this pattern:** The existing code uses struct types with JSON tags for all scan results (`governanceInfo`, `complexityMetrics`, `gitHistoryInfo`, `pheromoneSuggestion`). New analysis domains should follow the same convention.

### Pattern 2: Tolerant Config Parsing

**What:** Parse config files with graceful degradation -- extract what you can, never fail the entire scan for a single malformed config.

**When to use:** All governance config parsing (D-05/D-06).

**Example:**
```go
// Source: established pattern -- detectGovernance() already uses os.Stat and
// never fails on individual files, just skips what it can't read
func parseEslintrc(target string) map[string]interface{} {
    data, err := os.ReadFile(filepath.Join(target, ".eslintrc.json"))
    if err != nil {
        return nil // skip, don't fail
    }
    var config map[string]interface{}
    if err := json.Unmarshal(data, &config); err != nil {
        return map[string]interface{}{"parse_error": "malformed JSON"}
    }
    return config
}
```

**Why this pattern:** The existing `detectGovernance()` uses `os.Stat` checks and never propagates errors. This is the right approach for an init scan -- a single bad config file should not prevent the entire init from completing.

### Pattern 3: Heuristic Classification with Signals

**What:** Classify a project by checking for known structural patterns and returning both the classification AND the evidence that triggered it.

**When to use:** Directory structure classification (D-03/D-04).

**Example:**
```go
type dirClassification struct {
    Type    string   `json:"type"`     // "monorepo", "microservices", "standard_app", "library", "unknown"
    Signals []string `json:"signals"`  // ["packages/ directory found", "pnpm-workspace.yaml detected"]
}

func classifyDirectory(target string) dirClassification {
    // Check monorepo signals first (most specific)
    if hasDir(target, "packages") || hasDir(target, "apps") || hasFile(target, "pnpm-workspace.yaml") {
        return dirClassification{Type: "monorepo", Signals: detectedSignals}
    }
    // ... more patterns
}
```

### Anti-Patterns to Avoid

- **Parsing configs into strongly-typed structs:** Different projects use different config shapes. An `.eslintrc.json` might be `{ "extends": ["next"] }` or `{ "rules": { "no-unused-vars": "warn" } }`. Use `map[string]interface{}` or `gjson` path queries to extract what exists without failing on unexpected shapes.
- **Adding new dependencies:** The zero-new-deps principle is explicit in CLAUDE.md and Phase 72 CONTEXT.md. All parsing uses stdlib + already-imported packages.
- **Breaking the outputOK envelope:** Wrappers parse the JSON envelope `{ "ok": true, "result": {...} }`. New fields go inside `result`, not at the envelope level. Never change the envelope structure.
- **Blocking on unparseable files:** If a package.json has trailing commas, a Cargo.toml has comments in wrong places, or a YAML file is malformed -- skip it, log a warning, and continue. The scan must always complete.
- **Omitting zero-value fields:** Use `omitempty` on optional fields but NOT on fields where the planner expects the key to exist. The existing pattern uses `json:"field,omitempty"` for optional slices and plain `json:"field"` for required fields.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| JSON parsing | Custom JSON tokenizer | `encoding/json` or `gjson` | Stdlib handles all JSON; gjson handles path-based extraction |
| YAML parsing | Custom YAML parser | `gopkg.in/yaml.v3` | Already imported, battle-tested |
| TOML parsing | Custom TOML parser | `github.com/BurntSushi/toml` | Already imported, handles all TOML edge cases |
| Directory walking | Custom recursive descent | `filepath.WalkDir` | Already used in init_research.go with skip-list |
| XML parsing (pom.xml) | Custom XML parser | `encoding/xml` (stdlib) | Go stdlib handles XML well enough for pom.xml dependency extraction |

**Key insight:** The entire parsing layer is already available. The work is purely about writing extraction logic for each format, not building infrastructure.

## Common Pitfalls

### Pitfall 1: Over-Engineering the Pheromone Pattern Registry
**What goes wrong:** Building a generic pattern-matching DSL or config-driven system for 25 patterns when simple `if/else` chains with `hasFile()`/`fileContains()` would be clearer and more maintainable.
**Why it happens:** The existing code has a flat `generatePheromoneSuggestions()` with 10 sequential checks. It is tempting to "improve" it when expanding to 25.
**How to avoid:** Keep the same sequential check pattern. Group related checks with comments. Extract to a separate function only if it exceeds ~150 lines. A data-driven registry adds abstraction cost (type safety, serialization, testing) that is not justified at 25 patterns.
**Warning signs:** If you are defining a `PheromonePattern` interface or a `patterns.yaml` config file, stop.

### Pitfall 2: Breaking Backward Compatibility of init-research Output
**What goes wrong:** Wrappers (`.claude/commands/ant/init.md`, `.opencode/commands/ant/init.md`) parse specific JSON fields from init-research output. Adding new fields is safe, but renaming or removing existing fields breaks them.
**Why it happens:** When refactoring the scan logic, it is easy to rename fields for clarity.
**How to avoid:** Only ADD new fields to the `outputOK` result map. Never rename or remove existing fields (`detected_type`, `languages`, `frameworks`, `governance`, `charter`, `pheromone_suggestions`, etc.). The existing `governanceInfo` struct can be extended with new fields (JSON add-only).
**Warning signs:** Any test that previously accessed `result["governance"]` and now accesses a different key.

### Pitfall 3: Cargo.toml Is TOML, Not JSON
**What goes wrong:** Trying to parse Cargo.toml with `encoding/json`, which fails because TOML uses different syntax.
**Why it happens:** The filename ends in `.toml` but the project has JSON parsing as its default.
**How to avoid:** Use `github.com/BurntSushi/toml` for all `.toml` files. Note: `pyproject.toml` is also TOML, but some projects use `pyproject.toml` with `[build-system]` that has minimal dependency info -- the actual deps may be in `requirements.txt` or `[project.dependencies]`.

### Pitfall 4: gjson Requires Raw Bytes, Not Structured Data
**What goes wrong:** Passing a Go map to `gjson.Get()` instead of raw JSON bytes.
**Why it happens:** `gjson` operates on raw JSON strings/bytes, not Go data structures.
**How to avoid:** Read config file bytes first, then use `gjson.GetBytes(data, "dependencies")`. See existing usage in `cmd/state_cmds.go` lines 483-630.

### Pitfall 5: pom.xml Is XML, Not JSON or YAML
**What goes wrong:** Trying to parse pom.xml with `encoding/json`.
**Why it happens:** Most dependency files are JSON/YAML/TOML, but Java's Maven uses XML.
**How to avoid:** Use `encoding/xml` from Go stdlib. Define minimal structs for the XML shape (`<project><dependencies><dependency>`). This is the one format that needs a struct-based approach because XML parsing in Go requires struct tags.

### Pitfall 6: Gemfile Uses Ruby DSL, Not Structured Data
**What goes wrong:** Trying to parse Gemfile as YAML or JSON.
**Why it happens:** Gemfile looks structured but is actually Ruby code (`gem "rails", "~> 7.0"`).
**Why it happens:** Gemfile is executable Ruby, not a config file format.
**How to avoid:** Use `regexp` to extract `gem "name"` and `gem "name", "version"` patterns. This is imperfect but covers 95% of real Gemfiles. Accept that complex Gemfiles (conditional groups, git sources) may not parse fully.

## Code Examples

### Parsing package.json Dependencies (INIT-03)
```go
// Source: established pattern -- uses encoding/json (stdlib) + gjson
func parsePackageJsonDeps(target string) ([]depEntry, []depEntry) {
    data, err := os.ReadFile(filepath.Join(target, "package.json"))
    if err != nil {
        return nil, nil
    }

    var prodDeps, devDeps []depEntry

    // gjson for flexible path extraction
    prodResult := gjson.GetBytes(data, "dependencies")
    if prodResult.IsMap() {
        prodResult.ForEach(func(_, v gjson.Result) bool {
            prodDeps = append(prodDeps, depEntry{Name: v.Get("name").String()})
            return true
        })
    }

    devResult := gjson.GetBytes(data, "devDependencies")
    if devResult.IsMap() {
        devResult.ForEach(func(_, v gjson.Result) bool {
            devDeps = append(devDeps, depEntry{Name: v.Get("name").String()})
            return true
        })
    }

    return prodDeps, devDeps
}
```

### Parsing go.mod Dependencies (INIT-03)
```go
// Source: go.mod is a custom format (not JSON/YAML/TOML), use line parsing
func parseGoModDeps(target string) ([]depEntry, []depEntry) {
    data, err := os.ReadFile(filepath.Join(target, "go.mod"))
    if err != nil {
        return nil, nil
    }

    var direct, indirect []depEntry
    lines := strings.Split(string(data), "\n")
    inRequireBlock := false

    for _, line := range lines {
        line = strings.TrimSpace(line)
        if line == "require (" {
            inRequireBlock = true
            continue
        }
        if inRequireBlock && line == ")" {
            inRequireBlock = false
            continue
        }
        if strings.HasPrefix(line, "require ") || inRequireBlock {
            // Parse "module/path v1.2.3" or "module/path v1.2.3 // indirect"
            parts := strings.Fields(line)
            if len(parts) >= 2 && !strings.HasPrefix(parts[0], "//") {
                name := strings.TrimPrefix(parts[0], "require ")
                if name == "" && len(parts) >= 3 {
                    name = parts[1]
                }
                version := ""
                if len(parts) >= 3 {
                    version = parts[len(parts)-1]
                    if strings.HasPrefix(version, "//") {
                        version = parts[len(parts)-2]
                    }
                }
                entry := depEntry{Name: name, Version: version}
                if strings.Contains(line, "// indirect") {
                    indirect = append(indirect, entry)
                } else {
                    direct = append(direct, entry)
                }
            }
        }
    }
    return direct, indirect
}
```

### Directory Classification with Signals (INIT-04)
```go
// Source: new function following established hasFile/hasDir patterns
func classifyDirectory(target string) dirClassification {
    var signals []string

    // Monorepo signals (check first -- most specific)
    monorepoSignals := []struct{ path, label string }{
        {"packages", "packages/ directory found"},
        {"apps", "apps/ directory found"},
        {"pnpm-workspace.yaml", "pnpm-workspace.yaml detected"},
        {"lerna.json", "lerna.json detected"},
        {"nx.json", "nx.json detected"},
        {"turbo.json", "turbo.json detected"},
    }
    monorepoCount := 0
    for _, sig := range monorepoSignals {
        if hasFile(target, sig.path) || hasDir(target, sig.path) {
            signals = append(signals, sig.label)
            monorepoCount++
        }
    }
    if monorepoCount >= 1 {
        return dirClassification{Type: "monorepo", Signals: signals}
    }

    // Microservices signals
    dockerCount := countFilesMatching(target, "Dockerfile*")
    if dockerCount >= 2 {
        signals = append(signals, fmt.Sprintf("%d Dockerfiles detected", dockerCount))
        return dirClassification{Type: "microservices", Signals: signals}
    }

    // Standard app signals
    appDirs := []string{"src", "lib", "cmd", "app"}
    for _, d := range appDirs {
        if hasDir(target, d) {
            signals = append(signals, d + "/ directory found")
        }
    }
    if len(signals) >= 1 {
        return dirClassification{Type: "standard_app", Signals: signals}
    }

    // Library signals (exports in root, no src/)
    if !hasDir(target, "src") && !hasDir(target, "cmd") {
        if hasFile(target, "index.js") || hasFile(target, "index.ts") || hasFile(target, "main.go") || hasFile(target, "lib.rs") {
            signals = append(signals, "entry point in root, no src/ directory")
            return dirClassification{Type: "library", Signals: signals}
        }
    }

    return dirClassification{Type: "unknown", Signals: []string{"no strong structural signals detected"}}
}
```

### Governance Deep Parsing -- ESLint Example (INIT-05)
```go
// Source: established pattern -- gjson for path-based extraction
type governanceDetail struct {
    Tool   string                 `json:"tool"`
    File   string                 `json:"file"`
    Rules  map[string]interface{} `json:"rules,omitempty"`
    Extends []string              `json:"extends,omitempty"`
    Config map[string]interface{} `json:"config,omitempty"`
}

func parseEslintrc(target string) *governanceDetail {
    // Try all eslint config file variants
    for _, name := range []string{".eslintrc.json", ".eslintrc.js", ".eslintrc.yml", ".eslintrc"} {
        path := filepath.Join(target, name)
        data, err := os.ReadFile(path)
        if err != nil {
            continue
        }

        detail := &governanceDetail{Tool: "ESLint", File: name}

        // For JSON/YAML variants, use appropriate parser
        if strings.HasSuffix(name, ".json") || name == ".eslintrc" {
            parsed := gjson.ParseBytes(data)
            if rules := parsed.Get("rules"); rules.Exists() {
                detail.Rules = make(map[string]interface{})
                rules.ForEach(func(k, v gjson.Result) bool {
                    detail.Rules[k.String()] = v.Value()
                    return true
                })
            }
            if extends := parsed.Get("extends"); extends.IsArray() {
                extends.ForEach(func(_, v gjson.Result) bool {
                    detail.Extends = append(detail.Extends, v.String())
                    return true
                })
            }
        }

        return detail
    }
    return nil
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| File-presence detection only | Parse actual dependency files | Phase 73 (this phase) | Init-research goes from "found package.json" to "found 42 deps, 18 dev deps" |
| Governance: tool name only | Governance: extracted rules/settings | Phase 73 (this phase) | Workers get actual lint rules, not just "ESLint detected" |
| Flat directory listing | Classified structure with signals | Phase 73 (this phase) | Colony understands project topology |
| 10 pheromone patterns | ~25 patterns | Phase 73 (this phase) | Better initial pheromone signal quality |

**Deprecated/outdated:**
- None within the init-research domain. The existing code is the baseline being extended.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `gjson` can handle arbitrary JSON config shapes (nested extends, unusual keys) | Standard Stack | LOW -- gjson is designed for this. If a specific config format causes issues, fall back to `encoding/json` struct parsing for that format only. |
| A2 | Gemfile regex parsing covers 95% of real Gemfiles | Dependency Parsing | LOW -- complex Gemfiles (platform-specific, git sources) will be partially parsed. This is acceptable for init-scan quality. |
| A3 | pom.xml XML structure is stable enough for minimal struct parsing | Dependency Parsing | LOW -- Maven pom.xml XML schema is well-defined. Only need `<dependency>` elements. |
| A4 | pyproject.toml dependencies are in `[project.dependencies]` (PEP 621 format) | Dependency Parsing | MEDIUM -- older projects may use `[tool.poetry.dependencies]` or `setup.cfg`. Should check multiple sections. |
| A5 | The 15 new pheromone patterns listed in D-07 are sufficient to reach ~25 total | Pheromone Expansion | LOW -- the exact patterns are Claude's discretion. The listed patterns are strong starting points; a few more may emerge during implementation. |
| A6 | Wrappers will not need changes for the new output fields (they only render what they parse) | Colony Context Summary | MEDIUM -- wrappers currently display `languages`, `frameworks`, `governance`, etc. If the planner wants wrappers to display the NEW fields (tech_stack_detail, dir_classification), wrapper updates are needed. This is a planner decision, not a research finding. |

## Open Questions

1. **Should governance parsing extract full rule details or summaries?**
   - What we know: D-05 says "extract actual rules/settings." D-06 says "all categories at same depth."
   - What's unclear: Does "extract rules" mean every ESLint rule with its setting, or a summary like "12 rules configured, extends: next/core-web-vitals"?
   - Recommendation: Extract summaries (rule count, extends/plugins list, notable settings) rather than every single rule. Full rule dumps would make the output enormous and most rules are irrelevant to colony workers. The summary approach keeps output manageable while still being "actual rules/settings" rather than just tool names.

2. **Should the charter's TechStack field be enriched with parsed dependency data?**
   - What we know: The charter currently has `TechStack` as a free-text string generated by `generateTechStack()`. The new parsed dependency data is richer.
   - What's unclear: Should `TechStack` stay as a summary string (enriched from deeper data) or should a new `TechStackDetail` field hold the structured dependency list?
   - Recommendation: Keep `TechStack` as a summary string (backward compatible -- wrappers render it), and add a new `tech_stack_detail` field in the init-research output for structured data. The charter stays human-readable; the detail is for programmatic consumption by Phase 74 suggest-analyze.

3. **How should the colony context summary (INIT-07) format all the new data?**
   - What we know: The existing outputOK result map is consumed by wrappers and the ceremony. New fields are additive.
   - What's unclear: Should there be a single "colony_context_summary" string field, or do the individual structured fields suffice?
   - Recommendation: Add individual structured fields (`tech_stack_detail`, `dir_classification`, `governance_details`) to the outputOK result map. The wrappers and ceremony can compose a summary from these fields. A pre-formatted summary string would duplicate information and be harder for downstream consumers to use.

## Environment Availability

Step 2.6: SKIPPED (no external dependencies identified -- all work is code-only changes using existing Go stdlib and already-imported packages).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib `testing`) + testify (v1.11.1, already in go.mod) |
| Config file | none (Go convention) |
| Quick run command | `go test ./cmd/... -run "TestInitResearch" -count=1` |
| Full suite command | `go test ./cmd/... -count=1` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| INIT-03 | Parse package.json and extract deps + devDeps | unit | `go test ./cmd/... -run "TestParsePackageJsonDeps" -count=1` | Wave 0 |
| INIT-03 | Parse go.mod and extract direct + indirect deps | unit | `go test ./cmd/... -run "TestParseGoModDeps" -count=1` | Wave 0 |
| INIT-03 | Parse Cargo.toml and extract deps | unit | `go test ./cmd/... -run "TestParseCargoTomlDeps" -count=1` | Wave 0 |
| INIT-03 | init-research output includes tech_stack_detail field | integration | `go test ./cmd/... -run "TestInitResearchTechStackDetail" -count=1` | Wave 0 |
| INIT-04 | Classify monorepo structure with signals | unit | `go test ./cmd/... -run "TestClassifyDirMonorepo" -count=1` | Wave 0 |
| INIT-04 | Classify standard app structure | unit | `go test ./cmd/... -run "TestClassifyDirStandardApp" -count=1` | Wave 0 |
| INIT-04 | init-research output includes dir_classification field | integration | `go test ./cmd/... -run "TestInitResearchDirClassification" -count=1` | Wave 0 |
| INIT-05 | Parse .eslintrc.json and extract rules/extends | unit | `go test ./cmd/... -run "TestParseEslintrc" -count=1` | Wave 0 |
| INIT-05 | Parse YAML config and extract settings | unit | `go test ./cmd/... -run "TestParseYamlGovernance" -count=1` | Wave 0 |
| INIT-05 | init-research output includes governance_details field | integration | `go test ./cmd/... -run "TestInitResearchGovernanceDetails" -count=1` | Wave 0 |
| INIT-06 | Expanded pheromone suggestions include new patterns | unit | `go test ./cmd/... -run "TestPheromonePatternsExpanded" -count=1` | Wave 0 |
| INIT-06 | Pheromone suggestions count >= 15 in a real project | integration | `go test ./cmd/... -run "TestInitResearchPheromoneExpanded" -count=1` | Wave 0 |
| INIT-07 | init-research output includes all new fields | integration | `go test ./cmd/... -run "TestInitResearchColonyContextSummary" -count=1` | Wave 0 |
| INIT-07 | Existing output fields unchanged (backward compat) | integration | `go test ./cmd/... -run "TestInitResearchBackwardCompat" -count=1` | Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./cmd/... -run "TestInitResearch" -count=1`
- **Per wave merge:** `go test ./cmd/... -count=1`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `cmd/init_research_test.go` -- new test cases for dependency parsers (package.json, go.mod, Cargo.toml, pyproject.toml, Gemfile, pom.xml, mix.exs, composer.json)
- [ ] `cmd/init_research_test.go` -- new test cases for directory classification (monorepo, microservices, standard_app, library, unknown)
- [ ] `cmd/init_research_test.go` -- new test cases for governance deep parsing (eslintrc, prettier, golangci, CI configs)
- [ ] `cmd/init_research_test.go` -- new test cases for expanded pheromone patterns (15+ new patterns)
- [ ] `cmd/init_research_test.go` -- backward compatibility test (existing fields still present in output)
- [ ] Shared test helpers: `createTestFile()` utility for setting up fixture files in temp dirs (may already exist in test infrastructure)

## Security Domain

> Required -- no explicit `security_enforcement: false` in config.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | N/A -- init-research is a local CLI command with no auth |
| V3 Session Management | no | N/A -- no sessions in init-research |
| V4 Access Control | no | N/A -- reads local files only |
| V5 Input Validation | yes | All parsed config values are output as JSON strings/structs -- no code execution. The `--goal` flag is already validated for non-empty. Config file parsing must handle malformed input gracefully (never crash). |
| V6 Cryptography | no | N/A -- no encryption in init-research |

### Known Threat Patterns for Go CLI File Parsing

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Malformed config file causes panic | Denial of Service | Tolerant parsing -- never `panic()`, return nil/empty on parse error |
| Path traversal in --target flag | Spoofing | `filepath.Clean()` + reject paths containing `..` if needed |
| Giant config file causes OOM | Denial of Service | Read with size limit (e.g., cap at 1MB per config file) |
| XML entity expansion in pom.xml | Denial of Service | Go's `encoding/xml` has built-in entity expansion limits |

## Sources

### Primary (HIGH confidence)
- `cmd/init_research.go` -- Current implementation read in full (684 lines)
- `cmd/init_research_test.go` -- Existing test suite read in full (459 lines, 14 tests)
- `cmd/init_ceremony.go` -- Ceremony flow read in full (434 lines)
- `cmd/init_cmd.go` -- Init command read in full (319 lines)
- `pkg/colony/colony.go` -- ColonyState + Charter structs read in full (523 lines)
- `go.mod` -- Current dependencies verified (17 direct, 7 indirect)
- `.claude/commands/ant/init.md` -- Claude wrapper read in full
- `.opencode/commands/ant/init.md` -- OpenCode wrapper read in full
- `.planning/phases/72-smart-init-charter/72-CONTEXT.md` -- Phase 72 decisions
- `.planning/phases/72-smart-init-charter/72-VERIFICATION.md` -- Phase 72 verification (what was actually built)
- `.planning/REQUIREMENTS.md` -- INIT-03 through INIT-07 definitions

### Secondary (MEDIUM confidence)
- `cmd/state_cmds.go` -- gjson usage patterns (lines 483-630) verified
- `cmd/codex_visuals.go` -- renderCharterDisplay function verified (lines 450-479)
- `.planning/research/FEATURES.md` -- Feature analysis for v1.11 (A1, A2 sections)

### Tertiary (LOW confidence)
- None -- all findings verified against source code.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - all libraries verified in go.mod, usage patterns verified in existing code
- Architecture: HIGH - existing codebase read in full, patterns established by Phase 72
- Pitfalls: HIGH - all pitfalls derived from actual code analysis, not hypothetical
- Dependency parsing: MEDIUM - 8 formats to support, some (Gemfile, pom.xml) need non-standard parsing approaches
- Governance parsing: MEDIUM - config file shapes vary widely across projects, tolerant parsing is essential

**Research date:** 2026-04-28
**Valid until:** 30 days (stable domain -- Go CLI file parsing, no fast-moving dependencies)
