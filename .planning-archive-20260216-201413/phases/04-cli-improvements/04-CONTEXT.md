# Phase 4: CLI Improvements - Context

**Gathered:** 2026-02-13
**Status:** Ready for planning

<domain>
## Phase Boundary

Migrate CLI argument parsing to commander.js with colored output, auto-help, and subcommand structure. Focus on better UX while maintaining backward compatibility. Phase 5 (State & Context Restoration) is a separate concern.

</domain>

<decisions>
## Implementation Decisions

### Subcommand structure
- **Flat command structure** — all commands at top level (no `aether flag add` grouping)
- **kebab-case** naming convention — `flag-add`, `check-state`, `spawn-worker`
- Existing commands with emojis should be preserved as-is
- Command argument style: **Claude's discretion** (positional vs named flags)

### Color usage
- **Define a palette** — create a consistent CLI color scheme
- **Enabled by default** — `--no-color` flag for disabling
- **Both headers and output** colored — full color throughout
- Use picocolors library (lighter than chalk, already scoped in requirements)

### Help text design
- **Separate sections** for CLI commands vs slash commands
- **Show mapping** — `aether status` maps to `/ant:status` in help
- **Full examples with args and sample output** — not just syntax placeholders

### Backward compatibility
- **Deprecation warnings** — old syntax works but warns users
- **Time-based removal** — remove after 2-3 major versions
- Warnings should suggest the new command to use

### Claude's Discretion
- Positional vs named flag design for multi-argument commands
- Exact color palette choices (base on Aether brand or standard terminal colors)
- Exact wording of deprecation messages
- Auto-help implementation details

</decisions>

<specifics>
## Specific Ideas

- Keep existing emoji-based command names (they're already done properly)
- Help should make it clear: "These are CLI commands; /ant:* are Claude Code slash commands"

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 04-cli-improvements*
*Context gathered: 2026-02-13*
