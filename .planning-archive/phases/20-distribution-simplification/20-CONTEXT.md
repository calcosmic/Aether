# Phase 20: Distribution Simplification - Context

**Gathered:** 2026-02-19
**Status:** Ready for planning

<domain>
## Phase Boundary

Eliminate the runtime/ staging directory so the npm package reads directly from .aether/. Simplify the build pipeline from a 3-step copy-stage-package flow to direct packaging. Unify all distribution paths (system files, slash commands, agent definitions) into a single pipeline. Update all references and documentation to reflect the new structure.

</domain>

<decisions>
## Implementation Decisions

### Cleanup approach
- Delete runtime/ entirely — no redirect, no README stub, clean removal
- Delete sync-to-runtime.sh entirely — no archive copy
- Add a pre-packaging validation step (check required files exist in .aether/) but no file copying
- Update all documentation and code comments that reference runtime/ as part of this phase — not deferred

### What gets published
- Claude's discretion on whether to keep explicit allowlist or switch to exclude-based approach (publish all except private data)
- Updates via `aether update` should clean up files that were removed from distribution — keep target repos tidy
- Unify all three distribution paths (system files, slash commands, agent definitions) into a single pipeline — do this in Phase 20, not later

### Guard rails
- Claude's discretion on pre-commit hook: remove or repurpose for validation
- Auto-check before packaging to verify no private data (colony state, dream journal, research files) would be included
- Claude's discretion on whether auto-check blocks or warns on private data detection
- Include a dry-run mode that shows exactly what would be published without actually publishing

### Migration path
- Auto-cleanup of old runtime/ artifacts when users run `aether update` on the new version
- Major version bump to signal structural change
- One-time migration message shown after update explaining the change
- Version-aware error messages: detect old structure and suggest running `aether update`

### Claude's Discretion
- Allowlist vs exclude-list approach for what gets published
- Pre-commit hook: remove or repurpose
- Auto-check severity: hard block vs warning on private data detection
- Pre-packaging validation implementation details
- Exact format/wording of migration message and version-aware errors

</decisions>

<specifics>
## Specific Ideas

- User wants this to be a clean break — no backward compatibility shims, no deprecated folders hanging around
- Unification of all distribution paths is explicitly desired despite being a bigger change
- Dry-run mode is wanted for confidence before publishing
- Migration messaging is important — users should know what changed, not just have it silently change

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 20-distribution-simplification*
*Context gathered: 2026-02-19*
