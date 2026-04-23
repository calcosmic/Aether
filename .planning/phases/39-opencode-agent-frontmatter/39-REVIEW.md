---
phase: 39-opencode-agent-frontmatter
reviewed: 2026-04-23T12:00:00Z
depth: quick
files_reviewed: 10
files_reviewed_list:
  - .opencode/agents/aether-ambassador.md
  - .opencode/agents/aether-architect.md
  - .opencode/agents/aether-builder.md
  - .opencode/agents/aether-queen.md
  - .opencode/agents/aether-watcher.md
  - .opencode/agents/aether-scout.md
  - cmd/codex_e2e_test.go
  - cmd/opencode_agent_schema_test.go
  - cmd/opencode_agent_validate_test.go
  - cmd/platform_sync.go
findings:
  critical: 0
  warning: 2
  info: 1
  total: 3
status: issues_found
---

# Phase 39: Code Review Report

**Reviewed:** 2026-04-23T12:00:00Z
**Depth:** quick
**Files Reviewed:** 10
**Status:** issues_found

## Summary

Reviewed 6 OpenCode agent frontmatter rewrites and 4 Go source/test files. The OpenCode agent files have correct frontmatter (tools-as-object, hex colors, provider/model format, no name field). The validation logic in `platform_sync.go` is thorough with a smart double-parse approach to catch string-typed tools. Two substantive issues found: a missing agent in the test list and a gap in the production validator.

## Warnings

### WR-01: all24AgentNames missing aether-medic -- list has 24 entries, should have 25

**File:** `cmd/codex_e2e_test.go:683-709`
**Issue:** The `all24AgentNames` variable contains 24 agent names but `aether-medic` is missing. The actual `.opencode/agents/` directory contains 25 files. The variable name and comment both say "24" when the project has 25 agents (confirmed by CLAUDE.md and by the `expectedCount = 25` constant at line 864). This means `TestCodexInstallSetupUpdate_All24Agents` only tests 24 of 25 agents, leaving `aether-medic` untested in the install/setup/update pipeline.

**Fix:**
```go
// all25AgentNames is the canonical list of all 25 Aether agent names.
var all25AgentNames = []string{
	"aether-ambassador",
	"aether-archaeologist",
	"aether-architect",
	"aether-auditor",
	"aether-builder",
	"aether-chaos",
	"aether-chronicler",
	"aether-gatekeeper",
	"aether-includer",
	"aether-keeper",
	"aether-medic",       // <-- add this
	"aether-measurer",
	"aether-oracle",
	"aether-probe",
	"aether-queen",
	"aether-route-setter",
	"aether-sage",
	"aether-scout",
	"aether-surveyor-disciplines",
	"aether-surveyor-nest",
	"aether-surveyor-pathogens",
	"aether-surveyor-provisions",
	"aether-tracker",
	"aether-watcher",
	"aether-weaver",
}
```

### WR-02: validateOpenCodeAgentFile does not validate the mode field

**File:** `cmd/platform_sync.go:138-226`
**Issue:** The `validateOpenCodeAgentFile` function validates description, tools, color, name (absence), and model, but does NOT validate the `mode` field. The `openCodeAgentFrontmatter` struct has a `Mode` field but no Rule checks it. An agent with missing or invalid mode (e.g., `mode: foobar`) would pass validation and be installed via `aether install` / `aether update` without error. The `opencode_agent_schema_test.go` test does validate mode (Rule 2), but the production validator that gates installation does not.

**Fix:** Add a Rule between the existing rules (e.g., after the description check) in `validateOpenCodeAgentFile`:
```go
// Rule: mode must be a valid value
mode := strings.TrimSpace(fm.Mode)
if mode == "" {
    return fmt.Errorf("%s is missing mode in frontmatter", relPath)
}
if mode != "primary" && mode != "subagent" && mode != "all" {
    return fmt.Errorf("%s mode %q must be primary, subagent, or all", relPath, mode)
}
```

## Info

### IN-01: Duplicate YAML unmarshal in validateOpenCodeAgentFile is intentional but costly

**File:** `cmd/platform_sync.go:161-184`
**Issue:** The function unmarshals the YAML frontmatter twice: first into a typed struct (`openCodeAgentFrontmatter`) and then into a raw `map[string]interface{}`. The comment at line 179-181 explains the rationale (detecting tools-as-string which would silently produce a nil map in struct unmarshal). This is correct and intentional, but it means every agent file is parsed twice. For 25 small files this is negligible.

**Fix:** No action needed. The double-parse is a deliberate correctness tradeoff. If this ever becomes a concern, the entire function could be rewritten to use only the raw map approach.

---

_Reviewed: 2026-04-23T12:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: quick_
