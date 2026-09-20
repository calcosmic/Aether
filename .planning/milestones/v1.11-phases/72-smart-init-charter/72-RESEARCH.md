# Phase 72: Smart Init Charter - Research

**Researched:** 2026-04-28
**Domain:** Go CLI colony initialization ceremony with dual-mode (wrapper + native) terminal UX
**Confidence:** HIGH

## Summary

Phase 72 restores the colony charter ceremony that was lost during the shell-to-Go migration. The Go runtime already has most of the building blocks: `init-research` scans repos and generates charter data (4 sections) plus pheromone suggestions (10 patterns), and `init` creates COLONY_STATE.json v3.0. What is missing is: (1) wiring init-research into the init flow so Codex/CLI users get the full ceremony, (2) persisting charter data in COLONY_STATE.json, (3) expanding charter from 4 to 7 sections using data init-research already produces, and (4) a Go-native terminal approval flow with numbered-list prompts.

The key architectural challenge is dual-mode ceremony: Claude Code and OpenCode wrappers already orchestrate the ceremony via markdown (calling `aether init-research` then `aether init`), while Codex and direct CLI users need a Go-native path where the runtime handles scanning, display, and approval. Both paths must produce identical COLONY_STATE.json output.

**Primary recommendation:** Expand the existing `charterData` struct to 7 fields, add a `Charter` field to `ColonyState`, wire `init-research` into the Go-native init path, and implement a simple numbered-list terminal prompt using Go stdlib `bufio.Reader` (same pattern as `confirmRepair` in `recover_repair.go`).

## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** Charter data stored as a sub-object in COLONY_STATE.json with fields: `intent`, `vision`, `governance`, `goals`, `tech_stack`, `key_risks`, `constraints`. Wrappers render as markdown for display; runtime reads JSON for downstream reference.
- **D-02:** No separate charter.md file -- charter is structured data in JSON, not a document.
- **D-03:** Charter has 7 sections: Intent, Vision, Governance, Goals (existing 4) + Tech Stack, Key Risks, Constraints (new 3). New sections pull from data that `init-research` already produces.
- **D-04:** Phase boundary with Phase 73: Phase 72 uses data that `init-research` already provides. Phase 73 adds deeper analysis.
- **D-05:** Revise goal: user provides a new goal string, init-research re-runs, fresh charter presented. Clean restart.
- **D-06:** Reject (cancel): clean exit. No COLONY_STATE.json, no pheromones, no session.json, no artifacts.
- **D-07:** Approval sequence: charter review -> pheromone suggestions (tick-to-approve) -> shelf backlog -> final 3-option approval (proceed / revise / cancel).
- **D-08:** Full Go-native ceremony for Codex and direct CLI users.
- **D-09:** Dual mode: wrapper ceremony (Claude/OpenCode) keeps markdown rendering. Go ceremony (Codex/CLI) uses numbered-list terminal prompts. Both produce same COLONY_STATE.json.
- **D-10:** Go-native prompts use numbered list + user types number. No new dependencies.
- **D-11:** Go ceremony triggered by `aether init` running without a wrapper. Wrappers continue to orchestrate their own ceremony.

### Claude's Discretion

- Exact terminal prompt wording for Go-native ceremony
- Charter section ordering in the rendered output
- How key_risks and constraints are generated from existing scan data (which heuristics map to which sections)
- Whether Go ceremony includes pheromone tick-to-approve or auto-approves all suggestions

### Deferred Ideas (OUT OF SCOPE)

None -- discussion stayed within phase scope.

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| INIT-01 | Colony charter ceremony runs during `/ant-init` -- scans repo, writes charter, presents for approval | `init-research.go` provides scanning + charter generation; `init_cmd.go` provides state creation; need wiring + persistence + approval flow |
| INIT-02 | Charter approval flow with accept/revise/reject options | Wrapper uses AskUserQuestion (3 options); Go-native needs numbered-list prompt; cancel requires clean-exit (no artifacts) |

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Repo scanning (languages, governance, git) | Go CLI runtime | -- | `init-research.go` already implements all scanning logic |
| Charter generation (7 sections) | Go CLI runtime | -- | `generateCharter()` exists with 4 sections; expand to 7 in Go |
| Charter persistence in COLONY_STATE.json | Go CLI runtime | -- | State mutations belong in runtime per wrapper-runtime contract |
| Terminal approval prompts (Codex/CLI) | Go CLI runtime | -- | D-08: Go-native ceremony for non-wrapper platforms |
| Charter rendering (markdown display) | Wrapper (Claude/OpenCode) | Go visual renderer | Wrappers own presentation; Go provides JSON data |
| Pheromone suggestion approval | Wrapper (Claude/OpenCode) | Go CLI runtime | Wrappers use AskUserQuestion; Go ceremony needs own prompt |
| Shelf backlog management | Go CLI runtime | -- | `loadActiveShelf()` and `shelf-promote-batch` already exist |
| Colony state creation | Go CLI runtime | -- | `initCmd` already creates COLONY_STATE.json v3.0 |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib `bufio` | 1.22+ | Terminal user input (numbered-list prompts) | Only dependency needed; zero-new-deps principle from PROJECT.md |
| Go stdlib `fmt` | 1.22+ | Output formatting | Existing pattern throughout cmd/ |
| Go stdlib `os` | 1.22+ | Stdin/stdout/stderr access | Existing pattern |
| `cobra` | current (go.mod) | CLI command framework | Already in use for all commands |
| `pkg/colony` | current | ColonyState struct, state constants | Schema definition lives here |
| `pkg/storage` | current | JSON persistence via `SaveJSON`/`LoadJSON` | Established pattern throughout cmd/ |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `cmd/codex_visuals.go` | current | Visual rendering (banners, stage markers, dividers) | Charter display in Go-native ceremony |
| `cmd/ceremony_emitter.go` | current | Lifecycle ceremony events | Emit ceremony events for init flow |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `bufio.Reader.ReadString('\n')` for terminal input | `github.com/charmbracelet/bubbletea` TUI framework | Adds dependency; violates zero-new-deps; overkill for simple numbered prompts |
| Separate `charter.json` file | Sub-object in COLONY_STATE.json (D-01) | Separate file requires additional path management; JSON sub-object keeps all state in one place |

**Installation:**
No new packages needed. All requirements satisfied by Go stdlib + existing project packages.

**Version verification:** [VERIFIED: go.mod] No external dependencies to install.

## Architecture Patterns

### System Architecture Diagram

```
User (Claude/OpenCode)                  User (Codex/CLI)
        |                                        |
        v                                        v
  Wrapper markdown                        aether init "goal"
  (init.md)                                     |
        |                                        |
        v                                   ┌────┴──────────────┐
  aether init-research                     │  Go-Native Init    │
  --goal "..." --target .                 │  Ceremony Flow     │
        |                                  │                    │
        v                                  │  1. Run init-     │
  Parse JSON output                       │     research       │
  (charter, pheromones)                   │  2. Generate       │
        |                                  │     7-section      │
        v                                  │     charter        │
  Present charter                         │  3. Display        │
  (markdown rendering)                    │     charter (ANSI) │
        |                                  │  4. Pheromone      │
        v                                  │     approval       │
  Pheromone tick-to-approve               │     (numbered)     │
  (AskUserQuestion)                       │  5. Shelf backlog  │
        |                                  │  6. Final approval │
        v                                  │     (3 options)    │
  3-option approval                       └────┬──────────────┘
  (proceed/revise/cancel)                       |
        |                                       |
        v                                       v
  aether init "$ARGUMENTS"            ┌─────────┴──────────┐
  (with --charter-json flag)          │   COLONY_STATE.json │
        |                            │   + Charter sub-    │
        v                            │     object          │
  Colony initialized                 └────────────────────┘
```

### Recommended Project Structure

```
cmd/
├── init_cmd.go              # MODIFY: add --charter-json flag, store charter in state
├── init_ceremony.go         # NEW: Go-native ceremony flow (scan -> display -> approve)
├── init_research.go         # MODIFY: expand charterData to 7 fields, generate 3 new sections
├── init_research_test.go    # MODIFY: add tests for 3 new charter sections
├── codex_visuals.go         # MODIFY: add renderCharterVisual() for 7-section display
├── ceremony_emitter.go      # MODIFY: add init ceremony event emission
└── recover_repair.go        # REFERENCE: confirmRepair() pattern for stdin prompts

pkg/colony/
└── colony.go                # MODIFY: add Charter field to ColonyState struct

.claude/commands/ant/
└── init.md                  # MODIFY: update charter display to show 7 sections

.opencode/commands/ant/
└── init.md                  # MODIFY: mirror Claude Code changes
```

### Pattern 1: Go-Native Terminal Prompt

**What:** A numbered-list prompt using `bufio.Reader` for user selection. The existing `confirmRepair()` function in `recover_repair.go` demonstrates the pattern: prompt on stderr, read from stdin, parse response.

**When to use:** Any Go-native ceremony that needs user interaction (approval, selection, revision).

**Example:**
```go
// Source: [VERIFIED: cmd/recover_repair.go lines 155-167]
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

**Extension for numbered list (Phase 72):**
```go
// Proposed pattern for Go-native ceremony
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

### Pattern 2: Charter Data Expansion

**What:** Expand the existing `charterData` struct from 4 fields to 7, generating the 3 new sections from data `init-research` already produces.

**When to use:** During `generateCharter()` call in `init-research.go`.

**Current struct:**
```go
// Source: [VERIFIED: cmd/init_research.go lines 48-53]
type charterData struct {
    Intent     string `json:"intent"`
    Vision     string `json:"vision"`
    Governance string `json:"governance"`
    Goals      string `json:"goals"`
}
```

**Expanded struct (7 fields):**
```go
type charterData struct {
    Intent     string `json:"intent"`
    Vision     string `json:"vision"`
    Governance string `json:"governance"`
    Goals      string `json:"goals"`
    TechStack  string `json:"tech_stack"`
    KeyRisks   string `json:"key_risks"`
    Constraints string `json:"constraints"`
}
```

**Data source mapping for 3 new sections:**
| Section | Data Source | Heuristic |
|---------|-------------|-----------|
| `tech_stack` | `languages` + `frameworks` from init-research output | "Languages: go, node. Frameworks: go, node." |
| `key_risks` | Inferred from scan gaps | No CI -> "No CI/CD pipeline detected". No tests -> "No test infrastructure". No .gitignore with .env -> "Secret exposure risk". Large file count without tests -> "Complexity risk without test coverage". |
| `constraints` | Inferred from governance + complexity | "Use {detected formatter}" if formatter found. "Follow {detected linter} rules" if linter found. "No formal governance" if none detected. |

### Pattern 3: COLONY_STATE.json Charter Field

**What:** Add a `Charter` sub-object to `ColonyState` so the approved charter is persisted alongside the colony.

**When to use:** After user approves the charter during init.

**Integration:**
```go
// Source: [VERIFIED: pkg/colony/colony.go lines 159-187]
// Add to ColonyState struct:
type ColonyState struct {
    // ... existing fields ...
    Charter    *Charter    `json:"charter,omitempty"`    // NEW
    // ... existing fields ...
}

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

### Pattern 4: Init Command with Charter Flag

**What:** Add a `--charter-json` flag to `aether init` so wrappers can pass approved charter data to the state creation step.

**When to use:** Wrappers call `aether init --charter-json '{"intent":...}' "goal"` after user approval.

```go
// In init_cmd.go:
initCmd.Flags().String("charter-json", "", "Approved charter data as JSON string")
```

### Anti-Patterns to Avoid

- **Adding bubbletea or any TUI library:** D-10 explicitly says no new dependencies. Use `bufio.Reader` for simple prompts.
- **Writing charter as a separate file:** D-02 says charter is structured data in COLONY_STATE.json, not a document.
- **Duplication between wrapper and Go ceremony:** Both paths produce identical COLONY_STATE.json. The Go ceremony is a complete alternative path, not a subset.
- **Pheromone writing before colony state exists:** Phewrites happen after approval, before state creation. If cancel is chosen, no pheromones are written (D-06).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Terminal user input | Custom readline loop | `bufio.NewReader(os.Stdin).ReadString('\n')` | Already used in `confirmRepair()`, proven pattern |
| JSON persistence | Manual file I/O | `store.SaveJSON()` / `store.LoadJSON()` | Established pattern throughout cmd/, handles file locking |
| Visual rendering | Custom ANSI codes | `renderBanner()`, `renderStageMarker()`, `visualDivider` from `codex_visuals.go` | Consistent with all other visual output |
| Structured output envelope | Manual JSON formatting | `outputOK()` / `outputError()` / `outputWorkflow()` | Standard output contract for all commands |
| Lifecycle ceremony events | Manual event bus publish | `emitLifecycleCeremony()` | Standard ceremony emission used by all lifecycle commands |

**Key insight:** The Go runtime already has all the utility functions needed. This phase is about wiring existing components together, not building new infrastructure.

## Common Pitfalls

### Pitfall 1: Charter Data Lost on Cancel
**What goes wrong:** Phewrites happen before final approval, so canceling leaves orphaned phewrites in the pheromone store.
**Why it happens:** The wrapper ceremony writes phewrites after approval but before `aether init`. If the user cancels after phewrites are written but before init runs, phewrites persist.
**How to avoid:** D-06 requires clean exit -- no phewrites, no session, no artifacts. The Go-native ceremony should write phewrites only after the final "proceed" choice. Wrappers already do this correctly (phewrite happens after approval step).
**Warning signs:** Stale phewrites in a fresh colony that was supposed to be clean.

### Pitfall 2: JSON Field Name Mismatch Between charterData and ColonyState Charter
**What goes wrong:** The `charterData` struct uses `json:"tech_stack"` but the `Charter` struct in `colony.go` uses a different field name.
**How to avoid:** Define a single `Charter` struct in `pkg/colony/` and use it everywhere -- in `init_research.go`, `init_cmd.go`, and the wrappers. Remove the standalone `charterData` type from `init_research.go`.
**Warning signs:** JSON marshal/unmarshal failures in tests.

### Pitfall 3: Go-Native Ceremony Blocks in Non-TTY Environments
**What goes wrong:** Running `aether init` in CI or a pipe fails because it tries to read from stdin.
**Why it happens:** The Go-native ceremony prompts require a terminal. If stdin is not a TTY (piped input), the ceremony should fall back to auto-accept or error gracefully.
**How to avoid:** Check `isatty.Stdin()` before entering the interactive ceremony. If not a TTY, require `--charter-json` flag (non-interactive mode) or exit with a helpful error.
**Warning signs:** CI pipelines hanging on `aether init`.

### Pitfall 4: Wrapper Ceremony Breaks After charterData Expansion
**What goes wrong:** Wrappers parse the `charter` object from `init-research` JSON output and expect exactly 4 fields. After expansion to 7, wrappers may break if they hardcode field access.
**Why it happens:** Wrappers currently render charter as `{charter.intent}`, `{charter.vision}`, etc. New fields need to be added to the wrapper rendering.
**How to avoid:** Update both wrapper files (`.claude/commands/ant/init.md` and `.opencode/commands/ant/init.md`) in the same wave as the charter expansion. Both files are currently identical.
**Warning signs:** Missing charter sections in wrapper output.

### Pitfall 5: ColonyState JSON Serialization Breaks Backward Compatibility
**What goes wrong:** Adding `Charter *Charter` to ColonyState with `omitempty` is safe for reading old state (nil pointer), but any code that unmarshals COLONY_STATE.json without handling the new field could fail.
**How to avoid:** Use pointer type (`*Charter`) with `omitempty` tag. Old state files without `charter` will unmarshal to nil. All code reading `state.Charter` must nil-check.
**Warning signs:** Panics in downstream code accessing `state.Charter.Intent` without nil check.

## Code Examples

### Charter Struct (pkg/colony/)

```go
// Proposed addition to pkg/colony/colony.go
// Charter holds the approved colony charter data.
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

### Generate Tech Stack Section

```go
// Source: [VERIFIED: cmd/init_research.go lines 471-487 -- projectDetectors pattern]
// Extension: generate tech_stack from existing languages/frameworks data
func generateTechStack(languages []string, frameworks []string) string {
    var parts []string
    if len(languages) > 0 {
        parts = append(parts, "Languages: "+strings.Join(languages, ", "))
    }
    if len(frameworks) > 0 {
        // Deduplicate frameworks that overlap with languages
        seen := make(map[string]bool)
        for _, l := range languages {
            seen[l] = true
        }
        var uniqueFW []string
        for _, fw := range frameworks {
            if !seen[fw] {
                uniqueFW = append(uniqueFW, fw)
            }
        }
        if len(uniqueFW) > 0 {
            parts = append(parts, "Frameworks/Tools: "+strings.Join(uniqueFW, ", "))
        }
    }
    if len(parts) == 0 {
        return "No specific tech stack detected"
    }
    return strings.Join(parts, ". ")
}
```

### Generate Key Risks Section

```go
// Heuristic: infer risks from scan gaps
func generateKeyRisks(governance governanceInfo, hasGitRepo bool, pheromoneSuggestions []pheromoneSuggestion) string {
    var risks []string

    if len(governance.CIConfigs) == 0 {
        risks = append(risks, "No CI/CD pipeline detected -- manual deployment risk")
    }
    if len(governance.TestFrameworks) == 0 {
        risks = append(risks, "No test framework detected -- regression risk")
    }
    if len(governance.Linters) == 0 {
        risks = append(risks, "No linter configured -- code quality may drift")
    }
    // Check for secret exposure from pheromone suggestions
    for _, sug := range pheromoneSuggestions {
        if sug.Type == "REDIRECT" && strings.Contains(sug.Content, "secrets") {
            risks = append(risks, "Potential secret exposure -- .env files without .gitignore protection")
            break
        }
    }
    if !hasGitRepo {
        risks = append(risks, "Not a git repository -- no version control")
    }

    if len(risks) == 0 {
        return "No significant risks detected from initial scan"
    }
    return strings.Join(risks, ". ")
}
```

### Generate Constraints Section

```go
// Heuristic: infer constraints from detected governance
func generateConstraints(governance governanceInfo) string {
    var constraints []string

    if len(governance.Linters) > 0 {
        constraints = append(constraints, "Follow "+strings.Join(governance.Linters, "/")+" rules")
    }
    if len(governance.Formatters) > 0 {
        constraints = append(constraints, "Use "+strings.Join(governance.Formatters, "/")+" for code formatting")
    }
    if len(governance.TestFrameworks) > 0 {
        constraints = append(constraints, "Write tests using "+strings.Join(governance.TestFrameworks, "/"))
    }
    if len(governance.BuildTools) > 0 {
        constraints = append(constraints, "Build with "+strings.Join(governance.BuildTools, "/"))
    }

    if len(constraints) == 0 {
        return "No formal constraints detected -- colony should establish conventions"
    }
    return strings.Join(constraints, ". ")
}
```

### Go-Native Ceremony Flow (init_ceremony.go)

```go
// Source pattern: [VERIFIED: cmd/recover_repair.go confirmRepair()]
// Proposed: init_ceremony.go -- full Go-native ceremony

// runNativeCeremony orchestrates the complete init ceremony for Codex/CLI users.
func runNativeCeremony(goal, target string, scope colony.ColonyScope) error {
    // Step 1: Run init-research
    research := runInitResearchInternal(goal, target)
    if research == nil {
        return fmt.Errorf("init-research failed")
    }

    // Step 2: Display codebase summary
    renderCeremonySummary(research)

    // Step 3: Display charter (7 sections)
    charter := research.charter
    renderCharterDisplay(charter)

    // Step 4: Pheromone suggestions (auto-approve or tick-to-approve)
    // Claude's discretion: recommend auto-approve for Go ceremony
    // (tick-to-approve requires complex terminal UI)
    approvedPheromones := autoApprovePheromones(research.pheromoneSuggestions)

    // Step 5: Final approval loop
    for {
        choice := promptFinalApproval()
        switch choice {
        case 1: // Proceed
            writeApprovedPheromones(approvedPheromones)
            return createColonyWithCharter(goal, scope, charter)
        case 2: // Revise goal
            newGoal := promptNewGoal()
            if newGoal == "" {
                continue // empty input, re-prompt
            }
            return runNativeCeremony(newGoal, target, scope) // clean restart
        case 3: // Cancel
            return nil // clean exit, no state
        }
    }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Shell-based init ceremony (618-line bash) | Go `init-research` + `init` commands | April 2026 (Go migration) | Ceremony only works via wrapper; Codex/CLI users skip it |
| 4-section charter (intent, vision, governance, goals) | 7-section charter (+ tech_stack, key_risks, constraints) | Phase 72 (planned) | Richer charter from existing scan data |
| Charter not persisted | Charter as sub-object in COLONY_STATE.json | Phase 72 (planned) | Charter available for downstream reference |
| Wrapper-only ceremony | Dual-mode: wrapper + Go-native | Phase 72 (planned) | All 3 platforms get full ceremony |

**Deprecated/outdated:**
- Shell-based init ceremony: removed during Go migration, never re-ported. Concepts live on in `init_research.go`.
- `charterData` struct with 4 fields: will be replaced by 7-field `Charter` in `pkg/colony/`.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Go ceremony can auto-approve phewrites instead of tick-to-approve (Claude's discretion) | Architecture Patterns | If user wants tick-to-approve, need more complex terminal UI -- increases scope |
| A2 | Wrappers will accept `--charter-json` flag on `aether init` | Pattern 4 | If wrappers cannot parse/pass JSON, need alternative mechanism (e.g., separate `aether charter-set` command) |
| A3 | `isatty.Stdin()` check is sufficient to detect non-TTY environments | Pitfall 3 | In some environments (pipelines, containers) stdin may appear as TTY but not support interactive input |

## Open Questions (RESOLVED)

1. **RESOLVED: Should Go ceremony include pheromone tick-to-approve or auto-approve?**
   - Decision: Auto-approve for Go ceremony. Wrappers retain tick-to-approve. Auto-approve displays suggestions in ceremony output so Codex users see what was applied.

2. **RESOLVED: Should `init-research` output the expanded charter directly, or should `init` compute it?**
   - Decision: Expand `generateCharter()` in `init_research.go` to produce all 7 sections. Keeps research output self-contained for both wrapper and Go-native paths.

## Environment Availability

> Step 2.6: SKIPPED (no external dependencies identified)

All changes are Go code modifications using existing project packages and Go stdlib. No new tools, services, or CLIs needed.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) + existing test helpers |
| Config file | none -- uses project-wide test helpers in cmd/ |
| Quick run command | `go test ./cmd/ -run "TestInit" -timeout 30s` |
| Full suite command | `go test ./... -race -timeout 120s` |

### Phase Requirements -> Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| INIT-01 | init-research generates 7-section charter | unit | `go test ./cmd/ -run TestInitResearchCharter -x` | Yes (needs expansion) |
| INIT-01 | COLONY_STATE.json includes charter sub-object after init | unit | `go test ./cmd/ -run TestInitWithCharter -x` | No -- Wave 0 |
| INIT-01 | Go-native ceremony displays all 7 charter sections | unit | `go test ./cmd/ -run TestInitCeremonyDisplay -x` | No -- Wave 0 |
| INIT-02 | Proceed option creates colony with charter | unit | `go test ./cmd/ -run TestInitCeremonyProceed -x` | No -- Wave 0 |
| INIT-02 | Revise option re-runs research with new goal | unit | `go test ./cmd/ -run TestInitCeremonyRevise -x` | No -- Wave 0 |
| INIT-02 | Cancel option produces no artifacts | unit | `go test ./cmd/ -run TestInitCeremonyCancel -x` | No -- Wave 0 |
| INIT-02 | --charter-json flag stores charter in state | unit | `go test ./cmd/ -run TestInitCharterJSONFlag -x` | No -- Wave 0 |
| INIT-01 | Wrapper receives 7-section charter from init-research | integration | `go test ./cmd/ -run TestInitResearchCharterExpanded -x` | No -- Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./cmd/ -run "TestInit" -timeout 30s`
- **Per wave merge:** `go test ./... -race -timeout 120s`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps

- [ ] `cmd/init_research_test.go` -- expand `TestInitResearchCharter` to verify 7 fields (tech_stack, key_risks, constraints non-empty)
- [ ] `cmd/init_ceremony_test.go` -- new file for Go-native ceremony tests (proceed, revise, cancel flows)
- [ ] `cmd/init_cmd.go` tests -- add test for `--charter-json` flag storing charter in COLONY_STATE.json
- [ ] `pkg/colony/colony_test.go` -- add test for ColonyState with Charter field round-trip

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V5 Input Validation | yes | Validate `--charter-json` flag input (JSON parse + field sanitization) |
| V6 Cryptography | no | N/A for this phase |

### Known Threat Patterns for Go CLI

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Malicious JSON in --charter-json | Tampering | Validate JSON structure, sanitize string fields (use existing `sanitizeQueenInline` pattern) |
| Path traversal in target flag | Tampering | `filepath.Clean()` + existing directory validation in init-research |

## Sources

### Primary (HIGH confidence)

- [VERIFIED: cmd/init_research.go] -- Full scan: charterData struct (4 fields), generateCharter(), generatePheromoneSuggestions(), detectGovernance(), projectDetectors, governanceDetectors
- [VERIFIED: cmd/init_cmd.go] -- Full scan: initCmd creates COLONY_STATE.json v3.0, session.json, activity.log, recovery artifacts
- [VERIFIED: pkg/colony/colony.go] -- Full scan: ColonyState struct (no Charter field currently), state constants, scope types
- [VERIFIED: .claude/commands/ant/init.md] -- Full scan: wrapper ceremony flow (init-research -> charter -> phewrites -> shelf -> approval -> init)
- [VERIFIED: .opencode/commands/ant/init.md] -- Full scan: identical to Claude Code wrapper
- [VERIFIED: cmd/recover_repair.go] -- confirmRepair() pattern for terminal user input
- [VERIFIED: cmd/codex_visuals.go] -- renderBanner(), renderStageMarker(), renderInitVisual(), outputWorkflow()
- [VERIFIED: cmd/ceremony_emitter.go] -- emitLifecycleCeremony() for lifecycle events
- [VERIFIED: .aether/docs/wrapper-runtime-ux-contract.md] -- Full scan: runtime owns state, wrappers own presentation
- [VERIFIED: .planning/research/FEATURES.md] -- A1 section: Smart Init Ceremony gaps analysis
- [VERIFIED: cmd/init_research_test.go] -- 10 existing tests for init-research

### Secondary (MEDIUM confidence)

- [VERIFIED: .planning/ROADMAP.md] -- Phase 72 goal, success criteria, dependencies
- [VERIFIED: .planning/REQUIREMENTS.md] -- INIT-01, INIT-02 definitions

### Tertiary (LOW confidence)

- None -- all claims verified against source code.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all components verified in go.mod and source code
- Architecture: HIGH -- dual-mode pattern well-established in wrapper-runtime contract, existing code patterns documented
- Pitfalls: HIGH -- identified from source code analysis, backward compatibility concerns based on ColonyState struct inspection

**Research date:** 2026-04-28
**Valid until:** 30 days (stable domain -- Go stdlib patterns, no external dependencies)
