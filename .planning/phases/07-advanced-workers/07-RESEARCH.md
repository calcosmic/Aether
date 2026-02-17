# Phase 7: Advanced Workers - Research

**Researched:** 2026-02-18
**Domain:** Aether command repair (shell command definitions for specialized agents)
**Confidence:** HIGH

## Summary

Phase 7 repairs 5 specialized agent commands: oracle, chaos, archaeology, dream, and interpret. All 5 commands already have extensive, well-designed definitions (100-340 lines each) that passed Layer 2 diagnostic (file exists, correct format). The underlying infrastructure they depend on (swarm-display, activity-log, flag-add, session-verify-fresh, pheromone system) was fixed in Phases 2-5.

**Primary finding:** The commands are closer to functional than expected. The main issues are sync drift between the 3 copies (source of truth, Claude Code, OpenCode), and the chaos/archaeology Claude Code copies using `swarm-display-text` where the source of truth uses `swarm-display-inline`. The `oracle.sh` shell script for tmux background execution already exists.

**Primary recommendation:** Focus on sync repair and targeted fixes rather than rewrites. Test each command against the Aether codebase itself.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- Repair scope: surgical repair, not rewrites
- Worker priority: oracle, chaos, archaeology, dream, interpret (in that order)
- Testing: self-referential testing against Aether codebase
- Sync strategy: source of truth in .aether/commands/claude/, Claude Code copy identical, OpenCode adapted with normalize-args
- Output format: keep existing designs

### Claude's Discretion
- Exact fix approach per command
- Whether to add additional error handling
- Whether to simplify overly complex command steps
- Whether oracle.sh needs repair

### Deferred Ideas (OUT OF SCOPE)
None
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| ADV-01 | /ant:oracle performs deep research (RALF loop) | oracle.md exists (380 lines), oracle.sh exists, underlying functions verified |
| ADV-02 | /ant:chaos performs resilience testing | chaos.md exists (341 lines), flag-add function verified, sync drift identified |
| ADV-03 | /ant:archaeology analyzes git history | archaeology.md exists (332 lines), sync drift identified, git commands verified |
| ADV-04 | /ant:dream philosophical wanderer writes wisdom | dream.md exists (257 lines), all copies in sync, dreams/ directory pattern verified |
| ADV-05 | /ant:interpret validates dreams against reality | interpret.md exists (256 lines), OpenCode copy missing normalize-args adaptation |
</phase_requirements>

## Current State Audit

### Sync Status Matrix

| Command | SoT -> Claude Code | SoT -> OpenCode | Issue |
|---------|--------------------|-----------------|----- |
| oracle | MATCH | normalize-args adapted | OK |
| chaos | DIFFERS (swarm-display-text vs -inline) | normalize-args adapted | Sync drift in Claude Code |
| archaeology | DIFFERS (swarm-display-text vs -inline) | normalize-args adapted | Sync drift in Claude Code |
| dream | MATCH | normalize-args adapted | OK |
| interpret | MATCH | NOT adapted (identical to SoT) | Missing OpenCode adaptation |

### Dependency Functions (all verified present in aether-utils.sh)

| Function | Line | Status |
|----------|------|--------|
| swarm-display-init | ~2285 | EXISTS |
| swarm-display-update | ~2310 | EXISTS |
| swarm-display-inline | ~2434 | EXISTS |
| swarm-display-text | ~2705 | EXISTS (added in Phase 3) |
| activity-log | ~562 | EXISTS |
| flag-add | ~1158 | EXISTS |
| session-verify-fresh | ~3560 | EXISTS |
| session-clear | varies | EXISTS |
| pheromone-read | added Phase 2 | EXISTS |

### oracle.sh Status

File: `.aether/oracle/oracle.sh` (4452 bytes, executable)
Status: EXISTS and is executable
Created: 2026-02-16

### Dream Directory

`.aether/dreams/` -- needs defensive mkdir in dream command (may or may not exist in target repos)

## Architecture Patterns

### Three-Copy Sync Pattern (established in Phases 3-6)

1. Source of truth: `.aether/commands/claude/{command}.md`
2. Claude Code copy: `.claude/commands/ant/{command}.md` -- IDENTICAL to source of truth
3. OpenCode copy: `.aether/commands/opencode/{command}.md` -- adapted with:
   - Step -1: `normalized_args=$(bash .aether/aether-utils.sh normalize-args "$@")`
   - All `$ARGUMENTS` replaced with `$normalized_args`
   - `swarm-display-inline` replaced with `swarm-display-text` (OpenCode convention)

### Command Definition Pattern

Each command follows the same structure:
1. Frontmatter (name, description)
2. Role description and persona
3. Steps: Parse arguments -> Initialize visual -> Load context -> Execute core logic -> Output report -> Log activity
4. Guidelines section

## Common Pitfalls

### Pitfall 1: Swarm Display Function Mismatch
**What goes wrong:** Claude Code copy uses `swarm-display-text` but source of truth uses `swarm-display-inline`
**Why it happens:** Phase 3 added `swarm-display-text` and some copies got updated inconsistently
**How to avoid:** Always sync FROM source of truth TO copies, never the other direction

### Pitfall 2: Missing OpenCode Adaptation
**What goes wrong:** OpenCode copy is identical to source of truth, missing normalize-args step
**Why it happens:** Interpret was not adapted when other commands were
**How to avoid:** All OpenCode copies need Step -1 normalize-args and $normalized_args substitution

## Open Questions

1. **Oracle RALF loop quality** -- oracle.sh exists but hasn't been tested post-Phase-2 fixes. May need repairs.
   - Recommendation: Inspect oracle.sh, fix if needed, verify research.json creation

2. **Interpret AskUserQuestion compatibility** -- interpret uses AskUserQuestion for interactive choices. This should work in Claude Code but needs verification.
   - Recommendation: Test interactivity pattern during verification

## Sources

### Primary (HIGH confidence)
- Direct codebase inspection of all 5 command files across 3 locations
- Direct inspection of aether-utils.sh for function existence
- Phase 1 diagnostic report confirming Layer 2 PASS for all 5 commands
- diff comparisons between all copy locations

---

*Research date: 2026-02-18*
*Valid until: 2026-03-18 (stable codebase patterns)*
