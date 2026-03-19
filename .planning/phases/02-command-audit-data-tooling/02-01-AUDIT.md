# Slash Command Audit — Phase 02, Plan 01

**Date:** 2026-03-19
**Scope:** All 37 slash commands in `.claude/commands/ant/`
**Auditor:** GSD executor

## Audit Criteria

1. **Structure**: Valid YAML frontmatter with `name` and `description` fields
2. **Subcommand references**: Every `aether-utils.sh <subcommand>` call references a subcommand that exists
3. **File references**: Every file path referenced exists on disk
4. **No eliminated features**: No references to `runtime/` or removed directories
5. **Stale content**: Flag references to superseded concepts
6. **Agent references**: If the command spawns agents via Task tool, the agent name exists in `.claude/agents/ant/`

## Results

| Command | Status | Issues Found |
|---------|--------|-------------|
| archaeology.md | pass | None |
| build.md | pass | None |
| chaos.md | pass | None |
| colonize.md | pass | None |
| continue.md | pass | None |
| council.md | pass | None |
| dream.md | pass | None |
| entomb.md | pass | None |
| feedback.md | pass | None |
| flag.md | pass | None |
| flags.md | pass | None |
| focus.md | pass | None |
| help.md | warning | Frontmatter `name: help` missing `ant:` prefix (inconsistent with other commands) |
| history.md | pass | None |
| init.md | pass | None |
| insert-phase.md | pass | None |
| interpret.md | pass | None |
| lay-eggs.md | pass | None |
| maturity.md | pass | None |
| memory-details.md | warning | Frontmatter `name: memory-details` missing `ant:` prefix (inconsistent with other commands) |
| migrate-state.md | warning | References v1->v2.0 migration; state is now v3.0. Intentionally stale as a one-time migration tool. |
| oracle.md | pass | None |
| organize.md | pass | None |
| pause-colony.md | pass | None |
| phase.md | pass | None |
| pheromones.md | pass | None |
| plan.md | pass | Fixed: removed broken `.aether/planning.md` reference (file did not exist) |
| redirect.md | pass | None |
| resume-colony.md | pass | None |
| resume.md | warning | Frontmatter `name: resume` missing `ant:` prefix (inconsistent with other commands) |
| seal.md | pass | None |
| status.md | pass | None |
| swarm.md | pass | None |
| tunnels.md | pass | None |
| update.md | pass | None |
| verify-castes.md | warning | References LiteLLM proxy and model routing setup; these are documentation of a specific setup, not a general requirement |
| watch.md | pass | None |

## Summary

- **Pass:** 32 commands
- **Warning:** 5 commands (3 naming inconsistencies, 1 intentionally stale migration tool, 1 setup-specific documentation)
- **Fail:** 0 commands

### Fixes Applied

**plan.md** -- FIXED: Removed the broken reference `Read .aether/planning.md for full reference.` from the route-setter agent prompt (line 298). The planning discipline rules were already provided inline immediately after the reference, so no content was lost.

### Warning Details

**help.md, memory-details.md, resume.md** — These three commands use frontmatter `name` values without the `ant:` prefix (`help`, `memory-details`, `resume`). All other 34 commands use the `ant:` prefix convention. This is a minor naming inconsistency that does not affect functionality since Claude Code slash commands derive their invocation name from the file path, not the frontmatter.

**migrate-state.md** — References v1->v2.0 migration while the current state format is v3.0. This is intentionally stale because the command is a one-time migration tool for older colonies. The command itself documents that it can be removed "after v5.1 ships." No fix needed.

**verify-castes.md** — References LiteLLM proxy health check (`curl -s http://localhost:4000/health`) and model routing configuration. This is documentation of one specific development setup, not a universal requirement. The command acknowledges model-per-caste routing was attempted but is not possible with Claude Code's Task tool. No fix needed — this is informational documentation.

### Subcommand Coverage

All `aether-utils.sh` subcommands referenced across the 37 commands were verified against the case statement entries. The following subcommands are used and confirmed to exist:

- `activity-log`, `autofix-checkpoint`, `autofix-rollback`, `chamber-create`, `chamber-list`, `chamber-verify`, `changelog-append`, `colony-archive-xml`, `colony-prime`, `context-capsule`, `context-update`, `eternal-init`, `flag-acknowledge`, `flag-add`, `flag-check-blockers`, `flag-list`, `flag-resolve`, `generate-ant-name`, `generate-commit-message`, `generate-progress-bar`, `instinct-create`, `learning-approve-proposals`, `learning-check-promotion`, `learning-promote`, `learning-promote-auto`, `load-state`, `memory-capture`, `memory-metrics`, `midden-write`, `milestone-detect`, `pheromone-count`, `pheromone-display`, `pheromone-export-xml`, `pheromone-import-xml`, `pheromone-read`, `pheromone-write`, `print-next-up`, `queen-init`, `registry-add`, `resume-dashboard`, `session-clear`, `session-init`, `session-mark-resumed`, `session-read`, `session-update`, `session-verify-fresh`, `spawn-complete`, `spawn-log`, `suggest-analyze`, `swarm-cleanup`, `swarm-findings-add`, `swarm-findings-init`, `swarm-solution-set`, `unload-state`, `update-progress`, `validate-state`, `version-check-cached`

All confirmed present in the aether-utils.sh case statement.

### Agent References

Commands that spawn agents via Task tool (all confirmed present in `.claude/agents/ant/`):

| Command | Agent Referenced | Status |
|---------|-----------------|--------|
| colonize.md | aether-surveyor-provisions, aether-surveyor-nest, aether-surveyor-disciplines, aether-surveyor-pathogens | All exist |
| organize.md | aether-keeper | Exists |
| plan.md | aether-scout, aether-route-setter | Both exist |
| seal.md | aether-sage, aether-chronicler | Both exist |
| swarm.md | aether-archaeologist, aether-scout, aether-tracker | All exist |

### File References

All playbook references in build.md and continue.md verified:
- `.aether/docs/command-playbooks/build-prep.md` — exists
- `.aether/docs/command-playbooks/build-context.md` — exists
- `.aether/docs/command-playbooks/build-wave.md` — exists
- `.aether/docs/command-playbooks/build-verify.md` — exists
- `.aether/docs/command-playbooks/build-complete.md` — exists
- `.aether/docs/command-playbooks/continue-verify.md` — exists
- `.aether/docs/command-playbooks/continue-gates.md` — exists
- `.aether/docs/command-playbooks/continue-advance.md` — exists
- `.aether/docs/command-playbooks/continue-finalize.md` — exists

Other referenced files verified:
- `.aether/workers.md` — exists
- `.aether/QUEEN.md` — exists
- `.aether/CONTEXT.md` — exists
- `.aether/model-profiles.yaml` — exists
- `.aether/archive/model-routing/` — exists
- `TO-DOS.md` — exists
- `.aether/utils/swarm-display.sh` — exists
- `.aether/utils/watch-spawn-tree.sh` — exists
- `.aether/utils/colorize-log.sh` — exists
- `.aether/utils/chamber-compare.sh` — exists

Missing file:
- `.aether/planning.md` — **DOES NOT EXIST** (referenced by plan.md)
