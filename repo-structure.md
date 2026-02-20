# Repo Structure

One-line descriptions of top-level directories and key files.

## Directories

| Directory | Purpose |
|-----------|---------|
| `.aether/` | Colony system — workers.md, aether-utils.sh, docs/, utils/, templates/ |
| `.aether/data/` | Colony state (COLONY_STATE.json, pheromones, spawn tree) — never distributed |
| `.aether/docs/` | Developer reference — error codes, known issues, caste system |
| `.claude/commands/ant/` | 34 Claude Code slash commands (/ant:build, /ant:plan, etc.) |
| `.claude/agents/ant/` | 22 Claude Code subagents (aether-builder, aether-queen, etc.) |
| `.opencode/` | OpenCode equivalent commands and agents |
| `.planning/` | Development history — roadmap, phases, requirements |
| `bin/` | CLI tools — cli.js, validate-package.sh, lib/ |
| `tests/` | AVA unit tests + bash integration tests |
| `src/` | Source modules (thin — main logic in bin/) |

## Key Files

| File | Purpose |
|------|---------|
| `package.json` | npm package config — version, scripts, dependencies |
| `CLAUDE.md` | Claude Code instructions — architecture, rules, workflows |
| `README.md` | Project overview and quick start |
| `CHANGELOG.md` | Version history |
| `TO-DOS.md` | Development backlog |
| `RUNTIME UPDATE ARCHITECTURE.md` | Distribution flow documentation |
