# Phase 44: Doc Alignment and Archive Consistency - Context

**Gathered:** 2026-04-23
**Status:** Ready for planning
**Source:** /gsd-discuss-phase 44

<domain>
## Phase Boundary

Documentation alignment — ensure all docs match actual runtime behavior after Phases 39-43 changes. Audit, fix, and verify documentation across the entire Aether documentation surface. Ensure archived v1.5 evidence is internally consistent.
</domain>

<decisions>
## Implementation Decisions

### Doc Audit Scope
- **Comprehensive audit** — Not just the three docs in success criteria. Audit all Aether documentation surfaces: AETHER-OPERATIONS-GUIDE.md, publish-update-runbook.md, AGENTS.md, CLAUDE.md, CODEX.md, OPENCODE.md, wrapper commands (.claude/commands/ant/, .opencode/commands/ant/), and skills docs (.aether/skills/).

### Behavior Documentation Depth
- **Full command reference** — Document the complete command set and options for publish, update, integrity, medic --deep, and install. Not just changes from Phases 39-43. Makes docs useful as standalone reference.

### Archive Consistency Depth
- **Narrative + versions check** — Verify summary narratives don't contradict each other, version numbers are consistent across all summaries and verifications, and no orphaned references remain. Pragmatic, not exhaustive.

### Fix Mode
- **Auto-fix** — Fix inaccurate docs directly when found. Don't just report issues. Get the job done in one pass.
</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Core Documentation
- `.aether/docs/AETHER-OPERATIONS-GUIDE.md` — Operations guide (primary audit target)
- `.aether/docs/publish-update-runbook.md` — Publish/update runbook (primary audit target)
- `.claude/agents/ant/*.md` — AGENTS.md operator flows (25 agent definitions)
- `CLAUDE.md` — Project-level instructions
- `.codex/CODEX.md` — Codex commands + rules
- `.opencode/OPENCODE.md` — OpenCode rules

### Runtime Commands (source of truth for actual behavior)
- `cmd/publish_cmd.go` — Publish command (Phase 40)
- `cmd/update_cmd.go` — Update command (Phase 40-42)
- `cmd/integrity_cmd.go` — Integrity command (Phase 43)
- `cmd/medic_scanner.go` — Medic deep scan with scanIntegrity (Phase 43)
- `cmd/install_cmd.go` — Install command
- `cmd/runtime_channel.go` — Channel resolution

### Prior Phase Context
- `.planning/phases/40-stable-publish-hardening/` — Stable publish changes
- `.planning/phases/41-dev-channel-isolation/` — Dev channel isolation
- `.plasing/phases/42-downstream-stale-publish-detection/` — Stale publish detection
- `.planning/phases/43-release-integrity-checks/` — Integrity command + medic wiring
</canonical_refs>

<specifics>
## Specific Ideas

- The operations guide verification checklist must pass as written — test each step
- The runbook must match actual `aether publish` and `aether update --force` behavior for both stable and dev channels
- AGENTS.md must describe the new `scanIntegrity` medic deep scan behavior
- CLAUDE.md's "Publishing Changes" section may reference outdated steps
- Wrapper commands in .claude/commands/ant/ and .opencode/commands/ant/ may reference outdated flags or behaviors
- Skills in .aether/skills/ may reference outdated command behaviors
- Archived v1.5 milestone docs should be checked for internal contradictions

## Deferred Ideas

None — scope is clear from audit decisions.
</specifics>

---

*Phase: 44-doc-alignment-and-archive-consistency*
*Context gathered: 2026-04-23 via /gsd-discuss-phase*
