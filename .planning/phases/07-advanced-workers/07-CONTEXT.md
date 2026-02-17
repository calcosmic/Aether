# Phase 7: Advanced Workers - Context

**Gathered:** 2026-02-18
**Status:** Ready for planning

<domain>
## Phase Boundary

Fix 5 specialized agent commands so they work reliably: oracle, chaos, archaeology, dream, and interpret. These commands already have extensive definitions (100-340 lines each) that passed Layer 2 diagnostic. The goal is surgical repair -- fix what's broken in the underlying wiring, sync copies, and verify each command produces useful output. NOT a rewrite of the command definitions themselves.

</domain>

<decisions>
## Implementation Decisions

### Repair scope
- Command definitions are already well-designed -- do not rewrite them from scratch
- Focus on: fixing broken references to aether-utils.sh functions, syncing the 3 copies (source of truth, Claude Code, OpenCode), and verifying end-to-end execution
- Phase 2-5 fixed the underlying infrastructure (session-is-stale, pheromone-read, spawn-can-spawn, etc.) so these workers should be closer to functional now
- Chaos and archaeology have known sync drift: source of truth uses `swarm-display-inline` but Claude Code copy uses `swarm-display-text` -- resolve to match source of truth

### Worker priority
- Oracle first (most complex, uses tmux background execution, RALF loop)
- Chaos second (produces actionable reports, creates blocker flags)
- Archaeology third (git analysis, read-only)
- Dream fourth (philosophical wanderer, writes to dreams/)
- Interpret fifth (depends on dream output existing)

### Testing approach
- Each worker tested by running it against the Aether codebase itself (self-referential testing)
- Oracle: test with a simple topic, verify research.json and progress.md are created
- Chaos: test against aether-utils.sh, verify JSON report output
- Archaeology: test against a known file with git history
- Dream: test that it creates dream journal file and writes observations
- Interpret: test that it reads an existing dream file and produces assessments
- No automated test suite for these -- they are interactive prompt-based commands

### Output format
- Keep existing output format designs (banners, sections, emoji) -- they are well-designed
- Ensure JSON reports (chaos, oracle) are valid parseable JSON
- Activity log calls at the end of each command should work (verified in Phase 2)

### Sync strategy
- Source of truth: `.aether/commands/claude/` for each command
- Claude Code copy: `.claude/commands/ant/` -- identical to source of truth
- OpenCode copy: `.aether/commands/opencode/` -- adapted with normalize-args and swarm-display-render per OpenCode conventions
- Fix the chaos/archaeology sync drift as part of repair

### Claude's Discretion
- Exact fix approach per command (surgical edits vs targeted rewrites of broken sections)
- Whether to add additional error handling or graceful degradation
- Whether to simplify any overly complex command steps that are unlikely to work in practice
- Whether oracle.sh (the shell script for tmux background execution) needs creation or repair

</decisions>

<specifics>
## Specific Ideas

- The oracle command references `.aether/oracle/oracle.sh` for tmux background execution, but this shell script may not exist -- need to verify and create if missing
- Dream command should create `.aether/dreams/` directory if it doesn't exist (defensive mkdir)
- Interpret reads dreams and offers AskUserQuestion choices -- these interactive patterns should work with Claude Code's UI
- Chaos command's flag-add for critical/high findings integrates with the flag system -- verify flag-add subcommand works post Phase 2 fixes
- All commands reference `constraints.json` for colony context -- this was fixed in Phase 5 (pheromone system)

</specifics>

<deferred>
## Deferred Ideas

None -- discussion stayed within phase scope

</deferred>

---

*Phase: 07-advanced-workers*
*Context gathered: 2026-02-18*
