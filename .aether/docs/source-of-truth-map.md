# Aether Source-of-Truth Map

Updated: 2026-03-24 (v2.1 accuracy sweep -- post-modularization counts, domain modules, playbook corrections)

## Purpose

Define which files are authoritative for system behavior, which files are derived, and where documentation is currently out of sync.

## Authority Precedence (Highest to Lowest)

1. **Executable runtime**
- `.aether/aether-utils.sh` and sourced scripts in `.aether/utils/`
- Node CLI runtime in `bin/cli.js` and `bin/lib/*`
- Why: these files are what actually execute.

2. **Command execution specifications**
- `.claude/commands/ant/*.md`
- Why: these drive slash-command behavior in Claude Code.

2.5. **Agent behavior specifications**
- `.claude/agents/ant/*.md` and `.opencode/agents/*.md`
- Why: these define worker behavior when commands spawn subagents.

3. **State/data artifacts (runtime outputs, not policy)**
- `.aether/data/*` (state, pheromones, observations, flags)
- Why: these represent current colony state, not command intent.

4. **Human-facing documentation**
- Root `README.md`
- `.aether/docs/*.md`
- Why: explanatory, not executable.

## Ownership Map

| Domain | Authoritative Files | Notes |
|---|---|---|
| Core deterministic operations | `.aether/aether-utils.sh` | Dispatcher that loads domain modules on demand (~5,200 lines) |
| Slash command orchestration | `.claude/commands/ant/*.md` | Includes build/continue orchestrators |
| Worker behavior specs | `.claude/agents/ant/*.md` | 24 Claude agent definitions (Builder, Watcher, etc.) |
| Packaged Claude agent mirror | `.aether/agents-claude/*.md` | Distribution mirror; must stay byte-identical with `.claude/agents/ant/*.md` |
| OpenCode command surface | `.opencode/commands/ant/*.md` | 45 OpenCode command files; structure parity with Claude commands |
| OpenCode worker behavior specs | `.opencode/agents/*.md` | 24 OpenCode agent definitions |
| Domain modules | `.aether/utils/{flag,spawn,session,suggest,queen,swarm,learning,pheromone,state-api}.sh` | Extracted from aether-utils.sh in Phase 13; sourced on demand |
| Build/continue split stages | `.aether/docs/command-playbooks/*.md` | Loaded by orchestrators; executable instruction docs |
| Output templates | `.aether/templates/*` | Templates for generated state/handoff/wisdom/session artifacts |
| Colony wisdom source | `.aether/QUEEN.md` + `~/.aether/QUEEN.md` | Read by `queen-read`/`colony-prime` |
| User preferences | `~/.aether/QUEEN.md` (User Preferences section) | Managed by `/ant:preferences`; read by `queen-read`/`colony-prime` |
| Hive wisdom (cross-repo) | `~/.aether/eternal/memory.json` | High-value signals shared across colonies; read by `colony-prime` |
| Pheromone runtime state | `.aether/data/pheromones.json` | Active signals with TTL/decay semantics |
| Autopilot run state | `.aether/data/run-state.json` | Tracks autopilot phase, pause reasons, iteration count |
| Session/state | `.aether/data/COLONY_STATE.json`, `.aether/data/*.json` | Mutable runtime state |
| Colony registry | `~/.aether/registry.json` | All registered repos with metadata and domain tags |
| Package/distribution scope | `package.json` (`files`) + `.npmignore` | Defines what ships vs excluded |
| Repo-level onboarding docs | `README.md`, `CLAUDE.md` | Should reflect runtime/commands, never override them |

## Confirmed Implementation Facts

- Utility dispatcher and subcommands are implemented in `.aether/aether-utils.sh` (`case "$cmd" in`).
- Context continuity primitives are runtime-backed in `.aether/aether-utils.sh`:
  - `context-capsule` (compact prompt context),
  - `rolling-summary` (bounded narrative log),
  - `pheromone-prime --compact` and `colony-prime --compact`.
- `queen-init` creates `.aether/QUEEN.md` (not `.aether/docs/QUEEN.md`).
- `queen-read` and `colony-prime` read from `~/.aether/QUEEN.md` and `.aether/QUEEN.md`.
- `queen-promote` writes to `.aether/QUEEN.md`.
- `build.md` and `continue.md` are now orchestrators that load split playbooks under `.aether/docs/command-playbooks/`.
- Orchestrators run playbooks as staged instruction sets (Read-tool execution model), not as bash subcommand wrappers.
- Cross-platform surfaces are present:
  - 45 Claude commands and 45 OpenCode commands
  - 24 Claude agents and 24 OpenCode agents
- Autopilot subcommands (`autopilot-init`, `autopilot-update`, `autopilot-status`, `autopilot-stop`, `autopilot-check-replan`) manage autonomous build/continue cycles via `/ant:run`.
- `registry-list` lists all registered repos with metadata including domain tags.
- `.aether/agents-claude/*.md` mirrors `.claude/agents/ant/*.md` for packaging/distribution.
- `npm run lint:sync` enforces:
  - command parity checks,
  - Claude/OpenCode agent structural parity (count + filenames),
  - Claude/`.aether` agent mirror exact parity (count + filenames + content hash).
- File contents between Claude and OpenCode command/agent files currently differ; parity is structural, not byte-identical.

## Verified Inventory Snapshot

| Category | Location | Count | Status |
|---|---|---:|---|
| Core utility entrypoint | `.aether/aether-utils.sh` | 1 | Active |
| Sourced shell utilities | `.aether/utils/*.sh` | ~29 | Active (9 domain modules + infrastructure + XML) |
| XML utility scripts | `.aether/utils/xml-*.sh` | 5 | Active (see drift note) |
| Slash commands (Claude) | `.claude/commands/ant/*.md` | 45 | Active |
| Slash commands (OpenCode) | `.opencode/commands/ant/*.md` | 45 | Active (content differs from Claude variants) |
| Agent definitions (Claude) | `.claude/agents/ant/*.md` | 24 | Active |
| Agent mirror (packaging) | `.aether/agents-claude/*.md` | 24 | Active mirror (must match Claude agent files exactly) |
| Agent definitions (OpenCode) | `.opencode/agents/*.md` | 24 | Active (content differs from Claude variants) |
| Command playbooks | `.aether/docs/command-playbooks/*.md` | 9 | Active (5 build + 4 continue; excludes full/README) |
| Templates (all types) | `.aether/templates/*` | 12 | Active |
| Disciplines | `.aether/docs/disciplines/*.md` | 7 | Active |
| Tests (all files) | `tests/**` | ~140 | Active |

## Drift Findings (Docs vs Implementation)

No high-confidence drift items currently tracked in this document after the 2026-02-22 alignment pass.

## Canonical Read Order (For Contributors)

When determining "how Aether works now", read in this order:

1. `.aether/aether-utils.sh`
2. `.claude/commands/ant/build.md` and `.claude/commands/ant/continue.md`
3. `.aether/docs/command-playbooks/*.md`
4. `.claude/agents/ant/*.md`
5. Remaining `.claude/commands/ant/*.md`
6. `.opencode/commands/ant/*.md` and `.opencode/agents/*.md` (for cross-surface checks)
7. `README.md` and `.aether/docs/*.md`

## Maintenance Rules

1. If runtime behavior changes, update command specs in the same PR.
2. If command specs change, update docs in the same PR.
3. Docs must never introduce paths/types not accepted by runtime.
4. Treat this file as the index of authority boundaries.
5. Keep `.aether/agents-claude/*.md` synchronized with `.claude/agents/ant/*.md` (enforced by `npm run lint:sync`).

## Immediate Follow-up Checklist

1. [x] Fix QUEEN path references in `.aether/docs/QUEEN-SYSTEM.md` and `.aether/docs/queen-commands.md`.
2. [x] Update threshold descriptions in `.aether/docs/QUEEN-SYSTEM.md` to match runtime defaults.
3. [x] Update root `README.md` command count (35 -> 36 -> 37 -> 40 after Phases 2, 6, 7).
4. [x] Update `.aether/docs/README.md` to include `command-playbooks/` and clarify docs-vs-runtime authority.
5. [x] Review `bootstrap-system` allowlist in `.aether/aether-utils.sh` for stale doc entries.
6. [x] Add agent definitions to Ownership Map and authority hierarchy.
7. [x] Document playbook staged Read-tool execution model.
8. [x] Clarify OpenCode structural parity and non-identical content status.
9. [x] Add template ownership and verified inventory snapshot.
10. [x] Clarify XML utility status in dedicated docs (`.aether/docs/xml-utilities.md`).
