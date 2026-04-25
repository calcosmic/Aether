# Aether Restoration Project — Common Pitfalls

Date: 2026-04-20

## Introduction

This document outlines critical mistakes and risks to avoid when restoring ceremony to the Aether CLI framework. Based on codebase analysis and existing constraints, these pitfalls could break the system's behavior, violate architectural boundaries, or undermine the user experience.

## Core Architecture Boundaries

### 1. State Management Violation

**Pitfall**: Wrappers (`.claude/commands/ant/*.md`) attempting to mutate state files directly.

**Why it's dangerous**:
- State transitions are owned by the Go runtime (`cmd/codex_build.go`, `cmd/codex_continue.go`)
- wrappers should only present data, not change it
- Could cause race conditions with file locks in `.aether/locks/`

**Anti-patterns**:
```markdown
# BAD: Wrappers reading/writing COLONY_STATE.json
- Reading colony state from COLONY_STATE.json
- Updating session.json directly
- Modifying pheromone.json by hand
```

**Correct approach**: Use runtime commands only:
```bash
# Runtime commands that manage state safely
aether build $PHASE
aether continue
aether status
```

### 2. Wrapper Anti-Pattern: Visual Text as Truth

**Pitfall**: Parsing ANSI-formatted visual output to extract state information.

**Why it's dangerous**:
- Visual output is for humans, not programmatic consumption
- Output format can change without notice
- Runtime provides JSON mode for structured data: `AETHER_OUTPUT_MODE=json`

**Anti-patterns**:
```markdown
# BAD: Scraping visual output
- "Parse the progress bar to determine phase completion"
- "Extract worker status from the spawn plan display"
- "Count completed tasks from visual tables"
```

**Correct approach**:
```markdown
# GOOD: Use JSON mode for structured data
- Runtime provides structured JSON output when AETHER_OUTPUT_MODE=json
- All state data is available through runtime APIs
```

### 3. Generated Wrapper Synchronization Issues

**Pitfall**: Hand-editing generated MD files that have `<!-- Generated from .yaml -->` headers.

**Why it's dangerous**:
- YAML source files in `.aether/commands/*.yaml` are the source of truth
- 49 YAML files exist; their MD wrappers are auto-generated
- Changes get overwritten during regeneration
- 49 YAML files are untracked in git (potential data loss)

**Evidence**: `.claude/commands/ant/build.md` header:
```markdown
<!-- Generated from .aether/commands/build.yaml - DO NOT EDIT DIRECTLY -->
```

**Correct approach**:
```bash
# Always edit .yaml files, never .md files
vim .aether/commands/build.yaml
# Then regenerate wrappers if needed
```

### 4. OpenCode Parity Breaking

**Pitfall**: Updating Claude wrappers but forgetting OpenCode wrappers.

**Why it's dangerous**:
- `.opencode/commands/ant/` and `.claude/commands/ant/` must stay in sync
- Both platforms use the same YAML source
- 52 wrapper files exist in each directory
- Inconsistent UX across platforms

**Verification**:
```bash
# Both should have same number of files
ls .claude/commands/ant/ | wc -l    # 52
ls .opencode/commands/ant/ | wc -l  # 52
```

## Test Integrity Risks

### 5. Ceremony Affecting Runtime Behavior

**Pitfall**: Changes to ceremony layer affecting core runtime behavior.

**Why it's dangerous**:
- 2900+ Go tests must pass
- Visual presentation should not influence runtime logic
- Test patterns in `cmd/build_flow_cmds_test.go` show expected state transitions

**Evidence from tests**:
```go
// TestResumeColonyRotatesStaleSpawnTreeForPausedColony
// Shows resume should clear ghost workers
spawnTreePath := filepath.Join(dataDir, "spawn-tree.txt")
if err := os.WriteFile(spawnTreePath, []byte("ghost worker data"), 0644); err != nil {
    t.Fatalf("failed to seed spawn tree: %v", err)
}
```

### 6. Backward Compatibility Issues

**Pitfall**: Breaking existing colony state when adding new ceremony.

**Why it's dangerous**:
- Legacy state normalization in `cmd/session_flow_cmds_test.go`:
```go
// TestResumeColonyNormalizesLegacyPausedStateToReady
createTestColonyState(t, dataDir, colony.ColonyState{
    State: colony.State("PAUSED"), // Legacy state
})
```

**State transitions**:
```
READY → EXECUTING → BUILT → COMPLETED
```

## Dependencies and Fragility

### 7. Fragile Dependencies

**Pitfall**: Ceremony layer depending on specific dependency versions.

**Current dependencies** (`go.mod`):
```go
require (
    github.com/BurntSushi/toml v1.5.0
    github.com/anthropics/anthropic-sdk-go v1.29.0
    github.com/gorilla/websocket v1.5.3
    github.com/jedib0t/go-pretty/v6 v6.7.8
    github.com/spf13/cobra v1.10.2
    github.com/spf13/pflag v1.0.9
    github.com/tidwall/gjson v1.18.0
    github.com/tidwall/sjson v1.2.5
    golang.org/x/sync v0.16.0
    golang.org/x/sys v0.34.0
    gopkg.in/yaml.v3 v3.0.1
)
```

**Risk**: ceremony should not create dependency chains that could break the system.

## Visual Presentation Pitfalls

### 8. Reimplementing Runtime Visuals

**Pitfall**: Wrappers trying to replicate visual rendering from `cmd/codex_visuals.go`.

**Why it's dangerous**:
- All visual rendering is centralized in `cmd/codex_visuals.go`
- Wrapper should enhance presentation, not duplicate
- Runtime exposes two modes: JSON and Visual

**Correct separation**:
```markdown
# Runtime owns:
- Banner, progress bar, status formatting
- ANSI color handling
- Caste emoji/color maps
- Spawn plan visualization
- Phase/task status display

# Wrappers may add:
- Colony atmosphere context
- Plain language explanations
- Colony framing of status updates
```

## Data File Management

### 9. Untracked YAML Files

**Pitfall**: 49 YAML files in `.aether/commands/` are untracked in git.

**Impact**: 
- Risk of data loss
- Version control issues
- Difficult to track changes to command definitions

**Solution needed**:
```bash
# These files need to be tracked
git add .aether/commands/*.yaml
```

## Code Quality Risks

### 10. Duplicate Lines in Documentation

**Issue**: QUEEN.md has 262 duplicate lines that need cleaning.

**Impact**: 
- Documentation maintenance burden
- Potential confusion for developers
- Inconsistent information

## Checklist for Ceremony Changes

Before implementing any ceremony changes, verify:

- [ ] **State Safety**: No direct file I/O in wrappers
- [ ] **Source Truth**: Changes made to YAML files only, not MD files
- [ ] **OpenCode Sync**: Both Claude and OpenCode wrappers updated
- [ ] **JSON Mode**: No visual scraping for data extraction
- [ ] **Runtime Boundaries**: No duplication of runtime logic
- [ ] **Test Compatibility**: Changes won't break existing tests
- [ ] **Backward Compatibility**: Legacy state transitions preserved

## References

- `.aether/docs/wrapper-runtime-ux-contract.md` — Official contract
- `.claude/rules/aether-colony.md` — Colony rules
- `cmd/codex_visuals.go` — Visual rendering implementation
- `cmd/go.mod` — Dependencies
- `cmd/build_flow_cmds_test.go` — Build flow test patterns
- `cmd/session_flow_cmds_test.go` — Session flow test patterns
- `.aether/commands/build.yaml` — YAML source example
- `.claude/commands/ant/build.md` — Generated wrapper example

## Files Referenced

- `.aether/commands/` (49 YAML files)
- `.claude/commands/ant/` (52 MD files)
- `.opencode/commands/ant/` (52 MD files)
- `.aether/docs/wrapper-runtime-ux-contract.md`
- `.claude/rules/aether-colony.md`
- `cmd/codex_visuals.go`
- `cmd/go.mod`
- `cmd/build_flow_cmds_test.go`
- `cmd/session_flow_cmds_test.go`
- QUEEN.md (contains duplicates)
