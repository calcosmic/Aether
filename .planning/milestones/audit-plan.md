# Audit Plan: Comprehensive Aether Colony Review

## Goal
Audit whether the full Aether system is actually working properly across Claude Code, OpenCode, and Codex CLI across milestones v1.0, v1.1, and v1.2.

## Phases

### Phase 1: Runtime Truth & Core CLI
- Verify Go subcommands exist and work correctly
- Audit caste identity system (codex_visuals.go)
- Check agent spawning visibility
- Verify state mutation commands (init, plan, build, continue, seal, etc.)
- Test key CLI commands return expected output

### Phase 2: Wrapper Honesty & CLI Flag Mismatch
- Audit Claude Code wrappers (.claude/commands/ant/*.md)
- Audit OpenCode wrappers (.opencode/commands/ant/*.md)
- Search for CLI flag mismatches between markdown wrappers and Go CLI flags
- Verify wrappers delegate to runtime rather than faking behavior

### Phase 3: Codex Parity & Cross-Platform
- Audit Codex agents (.codex/agents/*.toml)
- Audit CODEX.md
- Compare Codex command/agent coverage with Claude/OpenCode
- Check if Codex reflects the same truth model

### Phase 4: Worker Dispatch, Lifecycle & Recovery
- Audit build, continue, plan, colonize, watch, status implementations
- Check recovery and partial-success flows
- Check context proof system
- Check skill routing and prompt integrity

### Phase 5: Documentation, Tests & Versioning
- Check for stale docs
- Check test coverage
- Check install/update/versioning
- Check cross-platform parity

## Success Criteria
- All major bugs, regressions, misleading UX, parity gaps, stale docs, and missing tests identified
- Each finding documented with file path, severity, and suggested fix
- Final report synthesizes all findings
