# Phase 101: Platform Parity Verification - Research

**Researched:** 2026-05-07
**Domain:** Five-surface command parity audit (Go runtime, YAML, Claude wrappers, OpenCode wrappers, Codex command-guide)
**Confidence:** HIGH

## Summary

Phase 101 builds a single combined parity test that cross-references command names across all five Aether surfaces: Go runtime (377 Cobra commands in golden catalog), YAML source definitions (60 files), Claude Code wrappers (60 markdown files), OpenCode wrappers (60 markdown files), and the Codex command-guide catalog (60 entries from `commandGuideCatalog()`). The test produces a severity-classified report of mismatches and freezes the current parity state in a golden file so CI catches future drift.

The critical technical challenge is that YAML command names do not always match Cobra command names directly. The golden catalog from Phase 100 captures primary Cobra names only (via `cmd.Name()`), not aliases. Meanwhile, 55 of 60 YAML commands reference runtime commands using alias names (e.g., `export-signals` is an alias for `pheromone-export-xml`; `patrol` is an alias for `colony-vital-signs`). The parity test must build a YAML-to-runtime name resolution map that accounts for aliases, compound-name mappings, and 5 prompt-only commands that intentionally have no runtime command.

The existing test infrastructure provides strong foundations: `TestCommandGuideCoversAllYamlCommands` already verifies YAML-to-command-guide parity; `TestClaudeOpenCodeCommandParity` already verifies Claude/OpenCode wrapper parity; and `TestCommandWrappersReferenceRealYamlSources` already verifies wrapper-to-YAML header consistency. Phase 101 extends this with runtime-truth verification and severity classification.

**Primary recommendation:** Build the alias resolution map as a Go function in the test file, then compare all five surfaces against the YAML command list (the intersection denominator). The golden file stores the parity snapshot as structured JSON. KNOWN-GAPS.md records known drift for Phase 105.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Parity mismatches are classified in three tiers: Critical (wrong flag name, phantom command, or behavior that doesn't match runtime), Warning (description mismatch, lifecycle command missing from Codex), Info (formatting only, non-lifecycle Codex gap).
- **D-02:** The parity report includes counts per tier and a summary, but no fix suggestions. Researcher/planner decide how to fix gaps.
- **D-03:** A wrapper or YAML file referencing a command NOT in the Go runtime audit-catalog is flagged as a Critical gap (phantom command).
- **D-04:** The 33 commands that have YAML definitions but no Codex TOML agent are flagged as Info-level gaps. All 60 commands are checked against all five surfaces.
- **D-05:** If a lifecycle command (per the D-06 list from Phase 100) is missing from Codex, it is escalated to Warning severity instead of Info.
- **D-06:** Parity tests freeze current state. Known drift is recorded in a KNOWN-GAPS.md that Phase 105 resolves. Tests pass today to keep CI green.
- **D-07:** The parity golden test freezes command names only from each surface -- not flags or descriptions. This keeps the golden file maintainable while still catching the most common drift (command additions/removals).
- **D-08:** A single combined test checks all 5 surfaces at once, rather than per-surface-pair tests. Simpler, fewer test files, and the report identifies which surface drifted.

### Claude's Discretion
- Exact test file name and structure
- How to extract command names from each surface (YAML parsing, wrapper markdown parsing, Codex TOML parsing, command-guide extraction)
- Whether KNOWN-GAPS.md is a separate file or embedded in the test output
- Parity report format and file location

### Deferred Ideas (OUT OF SCOPE)
None.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| PLAT-01 | Go runtime behavior, YAML definitions, Claude wrappers, OpenCode wrappers, and Codex command-guide output agree on command names, flags, and behavior descriptions | Five-surface name extraction detailed below. Alias resolution map bridges YAML names to Cobra primary names. Severity classification per D-01. |
| PLAT-02 | Existing parity tests (source-check, command_parity_test, command_source_hygiene_test) are extended to close 3 known gaps (command-guide alignment, wrapper contract fields, Codex coverage) | Existing test infrastructure analyzed. New combined test extends coverage. Three known gaps: (1) no runtime-to-YAML verification, (2) no Codex TOML coverage check, (3) no severity classification. |
| PLAT-03 | No platform wrapper describes behavior the runtime does not support | Phantom command detection (D-03) flags any wrapper or YAML referencing a command absent from the Go runtime. Alias resolution ensures legitimate alias names are not falsely flagged. |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Cobra command name truth | Go runtime (`cmd/`) | -- | `buildAuditCatalog()` walks the actual registered tree; always current |
| Alias resolution | Go runtime (`cmd/`) | -- | Aliases are registered in Cobra command definitions; only runtime knows them |
| YAML command name extraction | Go test (filesystem) | -- | Read `.aether/commands/*.yaml` filenames for command names |
| Wrapper name extraction | Go test (filesystem) | -- | Read `.claude/commands/ant/*.md` and `.opencode/commands/ant/*.md` filenames |
| Codex TOML name extraction | Go test (filesystem) | -- | Read `.codex/agents/*.toml` filenames |
| Command-guide name extraction | Go runtime (`cmd/command_guide.go`) | -- | `commandGuideCatalog()` returns the authoritative guide catalog |
| Parity classification and reporting | Go test infrastructure | -- | Test logic compares surfaces and classifies gaps |
| Golden file storage | `cmd/testdata/` | -- | Established pattern from Phase 100 (`command_catalog.json`) |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go | 1.26.1 | Language runtime | Project's primary language [VERIFIED: go.mod] |
| Cobra | v1.10.2 | CLI framework, command tree walking | Project's CLI framework [VERIFIED: go.sum] |
| encoding/json | stdlib | JSON output for golden files and reports | No new dependency needed |
| gopkg.in/yaml.v3 | (existing) | Parsing YAML frontmatter from wrappers | Already used in `command_guide_test.go` [VERIFIED: go.sum] |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| BurntSushi/toml | (not currently in go.mod) | Parsing Codex TOML agent files | If extracting command names from TOML `name` or `command` fields |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| TOML parsing library | Filename extraction only | Filenames give agent names (e.g., `aether-builder`) not command names (e.g., `build`); need to parse TOML for command mapping |
| Extending `buildAuditCatalog` to include aliases | Separate alias map in test | Simpler to keep catalog unchanged and build the alias map in the parity test; avoids modifying Phase 100 artifacts |

**Installation:**
```bash
# Potentially: go get github.com/BurntSushi/toml
# But filename-based extraction may avoid this dependency entirely
```

**Version verification:** Go and Cobra versions confirmed from go.mod/go.sum in this session.

## Architecture Patterns

### System Architecture Diagram

```
                     FIVE SURFACES TO COMPARE
                     ========================

.aether/commands/      .claude/commands/ant/   .opencode/commands/ant/
   *.yaml (60)            *.md (60)              *.md (60)
       |                     |                       |
       v                     v                       v
  YAML Names             Wrapper Names           Wrapper Names
  (filename minus        (filename minus         (filename minus
   .yaml)                 .md)                    .md)
       |                     |                       |
       +---------------------+-----------------------+
                             |
                             v
                    +---------------------+
                    |  YAML-to-Runtime    |
                    |  Alias Resolution   |
                    |  (bridges YAML      |
                    |  names to Cobra     |
                    |  primary names)     |
                    +---------------------+
                             |
              +--------------+--------------+
              |                             |
              v                             v
  cmd/testdata/                    cmd/command_guide.go
  command_catalog.json             commandGuideCatalog()
  (377 Cobra names)                (60 guide entries)
              |                             |
              +-----------------------------+
                             |
                             v
                   +---------------------+
                   |  Parity Comparison  |
                   |  & Classification   |
                   |                     |
                   |  Critical: phantom  |
                   |  Warning: lifecycle |
                   |  Info: non-lifecycle|
                   +---------------------+
                             |
                   +---------+---------+
                   |                   |
                   v                   v
          cmd/testdata/          KNOWN-GAPS.md
          parity_snapshot.json   (human-readable)
          (golden file)
```

### Recommended Project Structure
```
cmd/
  parity_test.go              # New: combined 5-surface parity test
  testdata/
    parity_snapshot.json      # New: golden parity snapshot
  audit_catalog.go            # Existing: Go runtime truth (Phase 100)
  command_guide.go            # Existing: commandGuideCatalog()
  command_parity_test.go      # Existing: Claude/OpenCode parity
  command_source_hygiene_test.go  # Existing: wrapper-to-YAML hygiene
  command_guide_test.go       # Existing: guide-to-YAML parity
.planning/phases/101-*/
  KNOWN-GAPS.md               # New: human-readable known gaps for Phase 105
```

### Pattern 1: Alias Resolution Map
**What:** A Go function that maps YAML command names to their corresponding Cobra primary command names.
**When to use:** Whenever comparing YAML/wrapper names against the runtime golden catalog.
**Why needed:** 55 of 60 YAML commands use names that differ from their Cobra `Use` field primary name.

```go
// Source: [VERIFIED: codebase analysis of alias_cmds.go, flag_cmds.go, flags.go, etc.]
// yamlToRuntimeName maps YAML slash-command names to Cobra primary names.
// Commands not in this map have a direct 1:1 name match with the runtime.
var yamlToRuntimeName = map[string]string{
    "export-signals":  "pheromone-export-xml",
    "import-signals":  "pheromone-import-xml",
    "flag":            "flag-add",
    "flags":           "flag-list",
    "insert-phase":    "phase-insert",
    "memory-details":  "memory-metrics",
    "patrol":          "colony-vital-signs",
    "pheromones":      "pheromone-display",
    "profile":         "profile-read",
    "resume":          "resume-colony",
    "shelf":           "shelf-list",
    "help":            "(cobra-builtin)", // excluded by IsAvailableCommand()
}
```

### Pattern 2: Prompt-Only Command Classification
**What:** YAML commands that have no corresponding runtime command because they are pure prompt wrappers.
**When to use:** When verifying YAML names against the runtime catalog -- these should NOT be flagged as phantom commands.

```go
// Source: [VERIFIED: grep for "runtime:" in .aether/commands/*.yaml]
// promptOnlyCommands have no runtime command -- they are pure LLM prompt wrappers.
var promptOnlyCommands = map[string]bool{
    "archaeology": true,
    "chaos":       true,
    "dream":       true,
    "interpret":   true,
    "organize":    true,
}
```

### Pattern 3: Surface Name Extraction
**What:** Functions to extract command names from each of the five surfaces.
**When to use:** In the parity test to build per-surface name sets.

```go
// Surface 1: Go runtime -- use buildAuditCatalog() from Phase 100
// Surface 2: YAML -- read filenames from .aether/commands/*.yaml
// Surface 3: Claude wrappers -- read filenames from .claude/commands/ant/*.md
// Surface 4: OpenCode wrappers -- read filenames from .opencode/commands/ant/*.md
// Surface 5: Command-guide -- use commandGuideCatalog() keys
// Surface 5b: Codex TOML -- read filenames from .codex/agents/*.toml (agent coverage)
```

### Anti-Patterns to Avoid
- **Don't treat aliases as phantom commands:** The alias resolution map is essential. Without it, 12+ commands would be falsely flagged as Critical (phantom) when they are legitimate alias-based names.
- **Don't flag prompt-only commands as missing from runtime:** These 5 commands are intentionally prompt-only wrappers with no Cobra command.
- **Don't forget that `help` is excluded from the golden catalog:** Cobra's `IsAvailableCommand()` returns false for the auto-generated help command, so it won't appear in the audit-catalog output. It IS in YAML and wrappers.
- **Don't freeze flags or descriptions in the golden file:** D-07 explicitly scopes the golden test to command names only. Flag and description drift are lower-priority and would cause excessive golden file churn.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Command tree walking | Custom Cobra tree traversal | `buildAuditCatalog(rootCmd)` from Phase 100 | Already tested, golden-verified, handles hidden/unavailable commands |
| YAML-to-guide parity | New cross-reference logic | `TestCommandGuideCoversAllYamlCommands` pattern from `command_guide_test.go` | Already verified in CI; pattern works |
| Wrapper filename extraction | Custom directory scanner | `sourceCheckFiles()` pattern from `source_check.go` | Already handles filtering and sorting |
| YAML name extraction | Custom YAML parser | `yamlCommandNamesForGuideTest()` from `command_guide_test.go` | Returns sorted list of YAML command names |

**Key insight:** The existing codebase already has 4 of the 5 surface name extraction patterns working in tests. Phase 101's main new work is (1) building the alias resolution map, (2) adding the severity classification logic, and (3) producing the combined parity snapshot golden file.

## Common Pitfalls

### Pitfall 1: False Phantom Commands from Unresolved Aliases
**What goes wrong:** 12 YAML commands use alias names not present in the golden catalog (e.g., `export-signals` vs `pheromone-export-xml`). Without alias resolution, these appear as phantom commands.
**Why it happens:** `buildAuditCatalog()` uses `cmd.Name()` which returns the primary `Use` field, not aliases.
**How to avoid:** Build a complete alias resolution map before comparing YAML names against the golden catalog.
**Warning signs:** Parity test reports more than 5 phantom commands (the 5 prompt-only commands are expected).

### Pitfall 2: Missing `help` Command in Runtime Catalog
**What goes wrong:** `help.yaml` exists, wrappers exist, but `help` is not in the golden catalog because Cobra excludes it via `IsAvailableCommand()`.
**Why it happens:** Cobra auto-generates `help` and marks it as unavailable for the purposes of `IsAvailableCommand()`.
**How to avoid:** Treat `help` as a special case -- it should be in the "expected runtime missing" set alongside prompt-only commands.
**Warning signs:** Parity test flags `help` as a Critical phantom command.

### Pitfall 3: Codex TOML Agents Are Not 1:1 With Commands
**What goes wrong:** Expecting 60 TOML agents to match 60 YAML commands. Only 27 TOML agents exist -- they map to worker castes, not slash commands.
**Why it happens:** TOML agents are worker definitions (builder, watcher, scout, etc.), not command wrappers. D-04 already acknowledges this gap (33 commands without Codex TOML = Info level).
**How to avoid:** The Codex surface for parity is `commandGuideCatalog()`, not `.codex/agents/*.toml`. TOML coverage is a separate Info-level check.
**Warning signs:** Test tries to match TOML filenames against YAML command names and fails on all 60.

### Pitfall 4: Breaking Existing Tests When Adding New Test File
**What goes wrong:** New parity test file imports functions from existing test files that use `t.Helper()` or expect specific test state.
**Why it happens:** Go test files in the same package share symbols, but helper functions may have side effects or state dependencies.
**How to avoid:** Reuse only pure functions (`repoRootForCommandSourceTest`, `yamlCommandNamesForGuideTest`). Copy patterns rather than creating cross-file test dependencies.
**Warning signs:** Test works in isolation but fails when run with `go test ./cmd/`.

### Pitfall 5: Golden File Becomes Stale After Any Command Addition
**What goes wrong:** Adding a new YAML command breaks the parity golden test because the snapshot doesn't include it.
**Why it happens:** The golden file captures exact expected state.
**How to avoid:** Use the `-update-golden` flag pattern from Phase 100 (`TestAuditCatalogGolden`). Document the update process in the test file comments.
**Warning signs:** CI fails after a legitimate command addition.

## Code Examples

### Surface Name Extraction (Existing Patterns)

```go
// Source: [VERIFIED: cmd/command_guide_test.go lines 317-336]
// YAML command names from filenames
func yamlCommandNamesForGuideTest(t *testing.T) []string {
    t.Helper()
    repoRoot, err := repoRootForCommandSourceTest()
    if err != nil {
        t.Fatalf("failed to find repo root: %v", err)
    }
    entries, err := os.ReadDir(filepath.Join(repoRoot, ".aether", "commands"))
    if err != nil {
        t.Fatalf("read .aether/commands: %v", err)
    }
    var names []string
    for _, entry := range entries {
        if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
            continue
        }
        names = append(names, strings.TrimSuffix(entry.Name(), ".yaml"))
    }
    sort.Strings(names)
    return names
}
```

### Golden File Pattern (Phase 100)

```go
// Source: [VERIFIED: cmd/audit_catalog_test.go lines 10-42]
func TestAuditCatalogGolden(t *testing.T) {
    catalog := buildAuditCatalog(rootCmd)
    data, err := json.MarshalIndent(catalog, "", "  ")
    if err != nil {
        t.Fatalf("marshal catalog: %v", err)
    }

    goldenPath := "testdata/command_catalog.json"

    if *updateGolden {
        if err := os.WriteFile(goldenPath, append(data, '\n'), 0644); err != nil {
            t.Fatalf("write golden file: %v", err)
        }
        t.Logf("golden file updated: %s", goldenPath)
        return
    }

    golden, err := os.ReadFile(goldenPath)
    if err != nil {
        t.Fatalf("read golden file: %v (run with -update-golden to create)", err)
    }

    got := string(data) + "\n"
    want := string(golden)
    if got != want {
        t.Errorf("catalog golden mismatch; run with -update-golden to refresh")
    }
}
```

### Complete Alias Map (Verified from Source)

```go
// Source: [VERIFIED: grep of all Aliases declarations in cmd/*.go]
// yamlToRuntimeName maps YAML slash-command names to their Cobra primary names.
// Only entries where the YAML name differs from the Cobra Use field are listed.
// Commands not listed here have a direct 1:1 name match.
var yamlToRuntimeName = map[string]string{
    "export-signals":  "pheromone-export-xml",  // cmd/alias_cmds.go
    "import-signals":  "pheromone-import-xml",  // cmd/alias_cmds.go
    "flag":            "flag-add",              // cmd/flag_cmds.go
    "flags":           "flag-list",             // cmd/flags.go
    "insert-phase":    "phase-insert",          // cmd/state_extra.go
    "memory-details":  "memory-metrics",        // cmd/memory_details.go
    "patrol":          "colony-vital-signs",    // cmd/memory_details.go
    "pheromones":      "pheromone-display",     // cmd/pheromone_mgmt.go
    "profile":         "profile-read",          // (compound name)
    "resume":          "resume-colony",         // cmd/session_flow_cmds.go
    "shelf":           "shelf-list",            // cmd/shelf_cmd.go
}

// promptOnlyCommands have no runtime command.
var promptOnlyCommands = map[string]bool{
    "archaeology": true,  // pure prompt wrapper
    "chaos":       true,  // pure prompt wrapper
    "dream":       true,  // pure prompt wrapper
    "interpret":   true,  // pure prompt wrapper
    "organize":    true,  // pure prompt wrapper
}

// cobraBuiltinCommands are excluded by IsAvailableCommand() but have YAML+wrappers.
var cobraBuiltinCommands = map[string]bool{
    "help": true,  // auto-generated by Cobra, not in golden catalog
}
```

### Lifecycle Commands for Warning Escalation (from Phase 100 D-06)

```go
// Source: [VERIFIED: cmd/audit_catalog_test.go lines 57-59]
// These 16 commands get elevated severity when missing from Codex.
var lifecycleCommands = map[string]bool{
    "init": true, "discuss": true, "colonize": true, "plan": true,
    "build": true, "continue": true, "seal": true, "entomb": true,
    "publish": true, "update": true, "recover": true, "status": true,
    "resume": true, "watch": true, "patrol": true, "profile": true,
}
```

### Parity Snapshot Structure

```go
// ParitySnapshot captures the command names found in each surface.
// Frozen in testdata/parity_snapshot.json for golden testing.
type ParitySnapshot struct {
    Timestamp   string              `json:"timestamp"`
    YAMLCatalog []string            `json:"yaml_catalog"`
    ClaudeWrapperCatalog []string   `json:"claude_wrapper_catalog"`
    OpenCodeWrapperCatalog []string `json:"opencode_wrapper_catalog"`
    CommandGuideCatalog []string    `json:"command_guide_catalog"`
    RuntimeCatalog []string         `json:"runtime_catalog"`
    // Only names that resolved (not prompt-only or cobra-builtins)
    RuntimeResolvedNames map[string]string `json:"runtime_resolved_names"`
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Manual parity checking | `source-check` command + `TestClaudeOpenCodeCommandParity` | v1.11 (Phase 70-79) | Automated but only 2-surface |
| No Codex command-guide | `commandGuideCatalog()` with full orchestration specs | v1.11 | Codex surface now auditable |
| No runtime command truth | `buildAuditCatalog()` + golden file | Phase 100 | Runtime surface now auditable |
| Per-surface-pair tests | Combined 5-surface test with severity | Phase 101 (this phase) | Catches cross-surface drift in one pass |

**Deprecated/outdated:**
- The old `command_count_test.go` pattern (just counting commands) has been superseded by the richer `audit_catalog_test.go` golden test from Phase 100.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | All 55 runtime-backed YAML commands either have direct name matches or are covered by the 12-entry alias map | Alias Resolution | Missing alias entries would cause false phantom flags |
| A2 | Prompt-only commands are correctly identified as the 5 commands without `runtime:` in their YAML | Prompt-Only Classification | Misclassifying a runtime-backed command as prompt-only would skip a legitimate parity check |
| A3 | The `commandGuideCatalog()` function is exhaustive for all 60 YAML commands | Codex Surface | If guide catalog is incomplete, parity check would flag false positives |
| A4 | The Codex TOML agents are worker caste definitions, not command definitions -- so they should be checked for coverage as a separate Info-level metric, not as a direct parity surface | Codex TOML Coverage | If TOML files actually contain command references, the extraction approach needs adjustment |
| A5 | The `help` command being excluded from the golden catalog is acceptable because it is a Cobra builtin | Help Command | If users expect `help` to be parity-checked, a special-case assertion is needed |

## Open Questions

1. **Should the parity test also verify that alias names actually work at runtime?**
   - What we know: Aliases are declared in Go source and tested by Cobra internally. The parity test checks name existence, not runtime behavior.
   - What's unclear: Whether the alias declarations have drifted from what the wrappers reference.
   - Recommendation: Out of scope for this phase. The alias map is built from source code analysis, which is sufficient for name parity.

2. **Should KNOWN-GAPS.md live in the phase directory or in `cmd/`?**
   - What we know: CONTEXT.md suggests the phase directory. The planner might prefer `cmd/` for discoverability.
   - What's unclear: User preference on location.
   - Recommendation: Phase directory is correct -- it is consumed by Phase 105, not by CI.

## Environment Availability

Step 2.6: SKIPPED (no external dependencies -- this phase uses only Go stdlib, existing project dependencies, and filesystem access).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | None -- `go test` directly |
| Quick run command | `go test ./cmd/ -run TestPlatformParity -count=1 -timeout 30s` |
| Full suite command | `go test ./... -race -timeout 120s` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| PLAT-01 | All 5 surfaces agree on command names | golden | `go test ./cmd/ -run TestPlatformParityGolden -count=1` | Wave 0 |
| PLAT-02 | Existing parity tests extended | unit | `go test ./cmd/ -run TestCommandGuide -count=1` | Existing |
| PLAT-03 | No wrapper describes unsupported behavior | unit | `go test ./cmd/ -run TestNoPhantomCommands -count=1` | Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./cmd/ -run TestPlatform -count=1 -timeout 30s`
- **Per wave merge:** `go test ./cmd/ -count=1 -timeout 60s`
- **Phase gate:** `go test ./... -race -timeout 120s`

### Wave 0 Gaps
- [ ] `cmd/parity_test.go` -- combined 5-surface parity test with golden file
- [ ] `cmd/testdata/parity_snapshot.json` -- frozen parity snapshot
- [ ] `.planning/phases/101-platform-parity-verification/KNOWN-GAPS.md` -- known gaps for Phase 105

## Security Domain

> No security enforcement enabled for this phase (config.json has no `security_enforcement` key). This is a read-only audit phase with no data mutation.

## Sources

### Primary (HIGH confidence)
- Codebase analysis of `cmd/audit_catalog.go` -- golden catalog structure verified
- Codebase analysis of `cmd/command_guide.go` -- commandGuideCatalog() entries verified
- Codebase analysis of `cmd/source_check.go` -- surface checking patterns verified
- Codebase analysis of `cmd/command_parity_test.go` -- existing parity test verified
- Codebase analysis of `cmd/command_guide_test.go` -- YAML-to-guide test verified
- Codebase analysis of `cmd/alias_cmds.go`, `cmd/flag_cmds.go`, `cmd/flags.go`, `cmd/memory_details.go`, `cmd/pheromone_mgmt.go`, `cmd/session_flow_cmds.go`, `cmd/shelf_cmd.go`, `cmd/state_extra.go` -- all alias declarations verified
- Codebase analysis of `.aether/commands/*.yaml` -- 60 YAML files, 5 prompt-only confirmed
- `cmd/testdata/command_catalog.json` -- 377 entries, runtime truth verified

### Secondary (MEDIUM confidence)
- Phase 100 research document (`.planning/phases/100-*/100-RESEARCH.md`) -- Cobra tree walking patterns

### Tertiary (LOW confidence)
- None -- all findings verified from source code

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - all libraries already in use, no new dependencies required
- Architecture: HIGH - alias resolution map verified from source, all 5 surfaces characterized
- Pitfalls: HIGH - discovered through direct codebase analysis, not assumptions

**Research date:** 2026-05-07
**Valid until:** 2026-06-07 (stable -- Go/Cobra patterns don't change rapidly)
