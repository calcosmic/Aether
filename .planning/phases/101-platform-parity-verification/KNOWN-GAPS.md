# Platform Parity Known Gaps

**Phase:** 101 (Platform Parity Verification)
**Generated:** 2026-05-07
**Status:** For Phase 105 remediation

## Summary

| Severity | Count |
|----------|-------|
| Critical | 0     |
| Warning  | 1     |
| Info     | 1     |

## Critical

No critical gaps found. All wrapper and YAML command names resolve to valid runtime commands after alias resolution. The alias map covers all 11 YAML-to-Cobra primary name differences. No phantom commands exist.

## Warning

### W-01: Lifecycle commands missing from Codex TOML agents

The 16 lifecycle commands (init, discuss, colonize, plan, build, continue, seal, entomb, publish, update, recover, status, resume, watch, patrol, profile) have YAML definitions and Claude/OpenCode wrappers but no dedicated Codex TOML agent entries.

Codex accesses these commands through `commandGuideCatalog()` (the command-guide surface), not through TOML agents. The command-guide covers all 60 YAML commands as 51 literal entries plus 9 intelligent entries (init, oracle, plan, colonize, swarm, build, continue, seal, discuss), giving 60 unique guide entries with zero overlap.

Severity: Warning because lifecycle commands are the most-used surface and their absence from TOML agents means Codex users must rely on the command-guide path instead of agent-native discovery.

## Info

### I-01: 33 YAML commands have no Codex TOML agent

Of the 60 YAML commands, 27 have corresponding Codex TOML agent files in `.codex/agents/`. The remaining 33 commands have no TOML agent. This is by design -- TOML agents represent worker castes (builder, watcher, scout, etc.), not command wrappers.

All 60 commands are covered by the `commandGuideCatalog()` function which provides Codex orchestration guidance.

## Scope

This report covers command name parity only. Flag parity and description parity are not audited in this phase.

The Codex TOML coverage gap (I-01) is structural -- TOML agents are worker definitions, not command wrappers. This is not a drift issue.

## Verified Surface Counts

| Surface | Count | Notes |
|---------|-------|-------|
| Go runtime (audit-catalog) | 377 Cobra commands | Includes subcommands |
| YAML definitions | 60 files | `.aether/commands/*.yaml` |
| Claude wrappers | 60 files | `.claude/commands/ant/*.md` |
| OpenCode wrappers | 60 files | `.opencode/commands/ant/*.md` |
| Command-guide catalog | 60 unique entries | 51 literal + 9 intelligent |
| Codex TOML agents | 27 files | `.codex/agents/*.toml` (worker castes) |
