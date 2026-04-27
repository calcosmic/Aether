# Phase 61: Porter Ant - Context

**Gathered:** 2026-04-27
**Status:** Ready for planning

<domain>
## Phase Boundary

Add a 26th caste (Porter) to Aether that surfaces interactive publish/push/deploy options after seal completes. Porter is a guided delivery wizard — it runs real commands, not just suggests them. Includes a standalone `/ant-porter` command and a `porter check` subcommand for pipeline readiness validation.

Five requirements: PORT-01 (caste registration), PORT-02 (agent across 4 surfaces), PORT-03 (seal lifecycle wiring), PORT-04 (slash command), PORT-05 (porter check).

</domain>

<decisions>
## Implementation Decisions

### Caste Identity
- **D-01:** Porter emoji is 📦 (per REQUIREMENTS.md PORT-01). Gatekeeper changes from 📦 to ⚔️ to resolve the emoji conflict.
- **D-02:** Porter color and label follow existing caste patterns in `casteColorMap` and `casteLabelMap` in `cmd/codex_visuals.go`. Planner picks appropriate ANSI color (not already used by another caste).
- **D-03:** Gatekeeper's emoji change affects all 3 visual maps plus the Gatekeeper agent definition files across all 4 surfaces.

### Interaction Model
- **D-04:** Dual-path interaction: Go runtime prints a post-seal readiness summary (works for all platforms including Codex), and the seal wrapper markdown adds an interactive Q&A on top (Claude/OpenCode only).
- **D-05:** Post-seal options presented to user: (1) Publish to hub, (2) Push to git remote, (3) Create GitHub release, (4) Deploy, (5) Skip for now. User picks one or more.
- **D-06:** Porter is a guided wizard — it runs the actual commands (`aether publish`, `git push`, `goreleaser`, etc.) and reports results, not just informational output.

### Porter Check Scope
- **D-07:** `porter check` runs a full pipeline validation plus downstream simulation: version agreement (binary vs hub vs source), uncommitted git changes, hub publish staleness, binary freshness, downstream dry run, test status (`go test ./...`), changelog completeness.
- **D-08:** Reuse logic from existing `aether integrity` and `aether medic --deep` where possible rather than reimplementing checks.

### Error Handling
- **D-09:** Porter stops on first failure and reports what failed. User decides whether to retry, skip the failed step, or abort. Does NOT continue on error.
- **D-10:** Each Porter step reports success/failure clearly so the user knows exactly what completed and what didn't.

### Agent Definition
- **D-11:** Porter agent follows the same XML structure as other agents: `<role>`, `<execution_flow>`, `<critical_rules>`, `<pheromone_protocol>`, `<return_format>`.
- **D-12:** Porter gets Read, Write, Edit, Bash, Grep, Glob tools — it needs Bash to run publish/push commands.
- **D-13:** Mirrored across all 4 surfaces per established pattern: `.claude/agents/ant/` (canonical), `.aether/agents-claude/` (byte-identical), `.opencode/agents/` (structural parity), `.codex/agents/` (TOML).

### Claude's Discretion
- Exact ANSI color for Porter caste (must not conflict with existing 25 colors)
- Exact wording of Porter agent definition sections
- How to render the post-seal readiness summary in Go runtime output
- How to structure the `porter check` output format
- What "deploy" means specifically for Aether (npm publish? goreleaser? both?)
- Whether porter check runs as a Cobra subcommand of `aether` or as a separate binary

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Visual System (caste registration)
- `cmd/codex_visuals.go` — casteEmojiMap, casteColorMap, casteLabelMap (lines 28-107). Add porter entry, change gatekeeper emoji.
- `cmd/codex_visuals_test.go` — Visual output tests

### Seal Lifecycle (integration point)
- `cmd/codex_workflow_cmds.go` — sealCmd (line 241), buildSealSummary (line 633). Post-seal Porter hook goes here.
- `.claude/commands/ant/seal.md` — Seal wrapper markdown (add post-seal Porter section)

### Publish Pipeline (Porter runs these)
- `cmd/publish_cmd.go` — `aether publish` command (line 13), runPublish (line 35)
- `cmd/install_cmd.go` — Install/update commands
- `.aether/docs/publish-update-runbook.md` — Full publish workflow documentation

### Integrity Checking (reuse for porter check)
- `cmd/integrity` — Existing integrity checks (if a separate file exists) or integrity-related functions in other files
- `cmd/medic_cmd.go` — Medic deep scan includes version agreement and staleness checks
- `pkg/storage/lock.go` — File locking for publish safety

### Agent Definition Pattern
- `.claude/agents/ant/aether-medic.md` — Medic agent as template (similar lifecycle integration role)
- `.claude/agents/ant/aether-gatekeeper.md` — Gatekeeper agent (needs emoji update)

### Command Registration
- `.aether/commands/*.yaml` — YAML source definitions for slash commands
- `.claude/commands/ant/*.md` — Claude Code wrappers
- `.opencode/commands/ant/*.md` — OpenCode wrappers

### Requirements
- `.planning/REQUIREMENTS.md` — PORT-01 through PORT-05
- `.planning/ROADMAP.md` — Phase 61 success criteria

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `cmd/codex_visuals.go` — Three maps for caste registration. Adding a new entry is mechanical.
- `cmd/codex_workflow_cmds.go` sealCmd — Already has post-seal hooks (event publishing, session update). Porter hooks go in the same area.
- `cmd/publish_cmd.go` — `runPublish()` already handles binary build, hub sync, version verification. Porter wraps this, doesn't replace it.
- `aether integrity` — Validates source/binary/hub/downstream chain. `porter check` should reuse this logic.
- `aether medic --deep` — Already checks version agreement and stale publishes.

### Established Patterns
- New caste = add to 3 maps + create agent definition + mirror to 4 surfaces + create YAML command + generate wrappers
- Cobra subcommands follow the pattern in `cmd/root.go` registration
- Visual output uses `renderBanner()` + `outputWorkflow()` pattern
- Wrapper markdown is presentation-only — Go runtime owns state mutations

### Integration Points
- Post-seal: after `outputWorkflow()` in sealCmd, run Porter readiness check and output summary
- Seal wrapper: add `## Post-Seal: Porter Delivery` section after the existing instinct promotion step
- `/ant-porter` command: new YAML in `.aether/commands/porter.yaml`, generated to Claude/OpenCode wrappers
- `porter check` subcommand: new Cobra command under rootCmd, or a subcommand of a `porter` parent command

</code_context>

<specifics>
## Specific Ideas

- Porter is a delivery ant — it carries the colony's work to the outside world (hub, git, releases)
- The guided wizard UX should feel like `aether publish` but interactive and post-seal
- `porter check` output should use the same visual style as `aether status` and `aether medic` — colored severity indicators, section headers
- The Gatekeeper emoji change (📦 → ⚔️) is a clean swap — just change the map entry and agent definitions. The crossed swords fit the "security gate" role.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 61-porter-ant*
*Context gathered: 2026-04-27*
