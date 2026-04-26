# Porter Ant Design

**Date:** 2026-04-26
**Status:** Approved

## Context

Aether colonies build things, but there's no ant responsible for making sure the finished work actually gets shipped. The colony lifecycle currently jumps from "all phases complete" (seal) to "archived" (entomb) with no verification that the project is ready to publish, deploy, or release.

The Porter ant fills this gap. It's the colony's delivery and alignment specialist — the ant that monitors whether the colony is still building the right thing, owns the project's publishing pipeline, and handles the actual shipping workflow.

## What the Porter Is

The Porter is a general-purpose caste that handles three responsibilities for any colony:

1. **Alignment checking** — Reviews the colony's current state against its original goal. Reports whether the colony is on track or drifting. Triggered at key lifecycle points (mid-colony checks, before seal).

2. **Pipeline ownership** — Discovers and understands how the project goes from "code is done" to "it's live." For Aether, that's the operations guide (dev/stable channels, publish, update, release). For any other project, it reads whatever pipeline docs exist in that repo.

3. **Publishing execution** — Actually runs the publish/release/deploy steps. Follows the project's documented workflow step by step, verifies each step, and handles failures.

### What the Porter is NOT

- Not a builder (doesn't write code)
- Not a watcher (doesn't run tests — though it may verify a build succeeded)
- Not a gatekeeper (doesn't do security scanning — though it checks pipeline security)
- It's the ant that takes the builders' output and gets it into users' hands

## Invocation Model

### Manual: `/ant-porter`

A dedicated slash command with three subcommands:

| Subcommand | When | What it does |
|------------|------|-------------|
| `/ant-porter check` | Mid-colony | Alignment check: goal vs progress, pipeline readiness, blocker scan |
| `/ant-porter publish` | Pre-seal or anytime | Discovers and executes the project's documented publish workflow |
| `/ant-porter status` | Anytime | Quick pipeline health summary |

### Automatic: During seal

When `/ant-seal` runs, it checks if pipeline documentation exists for the project. If found, the seal process runs a Porter alignment check as a pre-seal step. The check is advisory, not blocking — seal warns but proceeds even if Porter finds issues (not every project needs publishing).

## Agent Definition

### Caste Identity

| Property | Value |
|----------|-------|
| Name | `porter` |
| Emoji | `📦` |
| ANSI Color | `95` (light magenta) |
| Model | `sonnet` |
| Name prefixes | `Port`, `Carry`, `Haul`, `Ship`, `Route`, `Ferry`, `Cart`, `Convoy` |

### Tools

Read, Bash, Grep, Glob — no Write/Edit. Porter doesn't modify source code.

### Behavior

1. **On spawn**: Reads project pipeline docs (looks for `PIPELINE.md`, `OPERATIONS-GUIDE.md`, `RELEASE.md`, `Makefile`, `package.json` scripts, or any project-specific docs referenced in colony context)
2. **Alignment check**: Reads colony goal from COLONY_STATE.json, compares against current phase progress, checks for REDIRECT pheromones that might indicate scope drift
3. **Pipeline execution**: Follows the documented workflow step by step, running each command and verifying the result before proceeding
4. **Security awareness**: Checks for secrets in the pipeline, verifies clean working tree before publishing, validates version consistency
5. **Reporting**: Returns structured JSON with pass/fail per step, overall status, and any manual steps the user needs to complete

### Boundaries

- Does not write or modify source code
- Does not skip steps or make assumptions about the pipeline
- Does not commit on its own (reports what needs to be committed, user decides)
- Does not push to remote without explicit instruction in the pipeline docs

## Go Runtime Command

```
aether porter check    — Alignment + pipeline readiness report
aether porter publish  — Execute the documented publish workflow
aether porter status   — Quick pipeline health summary
```

### `porter check`

- Reads COLONY_STATE.json for goal, current phase, pheromones
- Scans for pipeline documentation files in the repo
- Reports: goal alignment score, pipeline readiness, any blockers
- Exit 0 = on track, non-zero = issues found

### `porter publish`

- Discovers and reads the project's pipeline documentation
- Presents the workflow steps it found
- Executes each step, verifying success before proceeding
- Reports per-step results and overall outcome
- If any step fails, stops and reports what needs manual intervention

### `porter status`

- Lightweight summary: last publish, pipeline docs found, current version vs latest, any known issues
- Fast to run, useful for a quick "are we good to ship?" check

## Implementation Scope

### New files to create

| File | Purpose |
|------|---------|
| `.claude/agents/ant/aether-porter.md` | Claude Code agent definition (canonical) |
| `.aether/agents-claude/aether-porter.md` | Packaging mirror (byte-identical) |
| `.opencode/agents/aether-porter.md` | OpenCode agent definition (structural parity) |
| `.codex/agents/aether-porter.toml` | Codex agent definition (TOML format) |
| `.aether/commands/porter.yaml` | YAML source definition for the slash command |
| `.claude/commands/ant/porter.md` | Claude Code slash command wrapper |
| `.opencode/commands/ant/porter.md` | OpenCode slash command wrapper |
| `cmd/porter_cmd.go` | Go runtime command implementation |

### Existing files to modify

| File | Change |
|------|--------|
| `cmd/codex_visuals.go` | Add `porter` to caste maps (emoji, color, label) |
| `cmd/generate_cmds.go` | Add `porter` to caste name prefixes |
| `cmd/codex_build.go` | Add porter caste to agent TOML lookup |
| `cmd/codex_workflow_cmds.go` | Register porter command + seal pre-check |
| `CLAUDE.md` | Add Porter to agents table, update count to 26 |
| `.claude/rules/aether-colony.md` | Add `/ant-porter` to commands table |
| `.opencode/OPENCODE.md` | Add `/ant-porter` to commands table |
| `.codex/CODEX.md` | Add porter command |
| `.aether/workers.md` | Add Porter to worker definitions and spawn protocol |

### No changes needed to

- Seal command core logic (Porter is a pre-check, not a state machine change)
- Existing quality gates (Porter is complementary)
- Build/continue flow (Porter operates at a different lifecycle stage)

## Verification

1. **Agent registration**: Run `aether help porter` — should show the three subcommands
2. **Caste identity**: Verify the `porter` caste appears in visual output with correct emoji and color
3. **Platform parity**: Confirm the agent definition exists in all four locations (Claude, Claude mirror, OpenCode, Codex)
4. **Seal integration**: Run `aether seal` in a colony with pipeline docs — should show Porter pre-check output
5. **Existing tests**: `go test ./...` should still pass (no regressions)
6. **New tests**: Add tests for `porter check` and `porter status` in `cmd/porter_cmd_test.go`
