# Aether Source-of-Truth Map

Updated: 2026-04-20

This document defines which files are authoritative for runtime behavior, which files are distribution mirrors, and which docs are explanatory only.

## Platform Support Policy

- **Primary platforms:** Claude Code and OpenCode. Their command and agent surfaces are the main maintained UX.
- **Secondary platform:** Codex CLI. Codex has best-effort support for the native `aether` workflow.
- **Release expectation:** Codex should remain safe, usable, and honest about what it supports. Claude/OpenCode parity drift is still higher priority than Codex UX drift.

## Authority Order

1. **Executable runtime**
   - `cmd/`
   - `pkg/`
   - Why: this is the Go implementation the `aether` binary actually runs.

2. **Codex runtime surface**
   - `AGENTS.md`
   - `.codex/CODEX.md`
   - `.codex/agents/*.toml`
   - Why: Codex uses the direct CLI plus TOML agents, not slash commands. This is a supported secondary surface.

3. **Slash-command platform surfaces**
   - `.claude/commands/ant/*.md`
   - `.opencode/commands/ant/*.md`
   - `.claude/agents/ant/*.md`
   - `.opencode/agents/*.md`
   - Why: these are the primary user-facing contracts for Claude Code and OpenCode.

4. **Packaged mirrors**
   - `.aether/agents-claude/*.md`
   - `.aether/agents-codex/*.toml`
   - `.aether/skills-codex/**/SKILL.md`
   - Why: these ship with installs and must stay aligned with their source trees.

5. **Guidance and playbooks**
   - `.aether/docs/command-playbooks/*.md`
   - `.aether/docs/*.md`
   - Why: these explain or orchestrate behavior but do not override the Go runtime.

6. **Mutable state**
   - `.aether/data/*.json`
   - `.aether/CONTEXT.md`
   - `.aether/HANDOFF.md`
   - Why: these are runtime outputs, never the source of policy.

## Ownership Map

| Area | Source of truth | Mirror / consumer |
|---|---|---|
| Go runtime | `cmd/`, `pkg/` | `aether` binary |
| Codex agents (shipped surface) | `.codex/agents/aether-*.toml` | `.aether/agents-codex/*.toml`, `~/.aether/system/codex/` |
| Codex local GSD helpers | `.codex/agents/gsd-*.toml` | Repo-local GSD workflows only (not packaged) |
| Claude agents | `.claude/agents/ant/*.md` | `.aether/agents-claude/*.md` |
| Shared skills | `.aether/skills/**/SKILL.md` | `.aether/skills-codex/**/SKILL.md`, `~/.aether/system/skills-codex/` |
| Claude commands | `.claude/commands/ant/*.md` | Claude Code |
| OpenCode commands | `.opencode/commands/ant/*.md` | OpenCode |
| Codex guidance | `AGENTS.md`, `.codex/CODEX.md` | Codex CLI |
| Session recovery | `.aether/data/session.json` | `.aether/CONTEXT.md`, `.aether/HANDOFF.md`, `aether resume` |
| Spawn activity | `.aether/data/spawn-tree.txt` | `aether status`, `aether swarm --watch` |
| Slash-command wrapper specs | `.aether/commands/*.yaml` | `.claude/commands/ant/*.md`, `.opencode/commands/ant/*.md` |

## Verified Inventory

| Category | Location | Count |
|---|---|---:|
| YAML wrapper specs | `.aether/commands/*.yaml` | 50 |
| Claude commands | `.claude/commands/ant/*.md` | 50 |
| OpenCode commands | `.opencode/commands/ant/*.md` | 50 |
| Claude agents | `.claude/agents/ant/*.md` | 25 |
| OpenCode agents | `.opencode/agents/*.md` | 25 |
| Codex agents (shipped Aether surface) | `.codex/agents/aether-*.toml` | 25 |
| Codex helper agents (repo-local GSD) | `.codex/agents/gsd-*.toml` | 0 |
| Codex packaging mirror | `.aether/agents-codex/*.toml` | 25 |
| Shared skills | `.aether/skills/**/SKILL.md` | 83 |
| Codex skill mirror | `.aether/skills-codex/**/SKILL.md` | 83 |

## Notes

- `aether resume` is the canonical Codex-facing alias for `resume-colony`.
- `aether run`, `aether watch`, and `aether oracle` are canonical Codex-facing compatibility entrypoints.
- `aether swarm` is now the Codex compatibility entrypoint for explicit swarm routing and live worker watch mode.
- `export-signals` / `import-signals` are flat aliases over the pheromone XML commands.
- Distribution is driven by the Go binary and embedded companion assets; the repo no longer uses `package.json` as the release authority.
- `.codex/agents/*.toml` contains both the shipped 24 `aether-*` agents and repo-local `gsd-*` workflow helpers. Packaging, install, setup, update, and parity checks operate only on the shipped `aether-*` surface.

## Maintenance Rules

1. Change runtime behavior in `cmd/` / `pkg/` first.
2. Update the Claude/OpenCode markdown mirrors in the same change when command syntax or UX changes.
3. Update Codex docs (`AGENTS.md`, `.codex/CODEX.md`) when native CLI semantics, install/update behavior, safety guarantees, or Codex-specific guidance changes.
4. Keep packaged mirrors synchronized with their source trees.
5. Treat `.aether/data/` and generated handoff/context files as outputs, not specs.

## Wrapper-Runtime UX Boundary

The Go runtime and wrapper markdown files have distinct ownership responsibilities:

### Runtime Owns (Go — `cmd/`, `pkg/`)
- State mutations (COLONY_STATE.json, session files)
- Phase transitions and gating
- Verification and testing
- Persistence and file locking
- Next-step routing logic
- Visual output rendering (`cmd/codex_visuals.go`)
- ANSI color handling, banner formatting, progress bars
- Caste identity (emoji maps, color maps)

### Wrappers May Add (Markdown — `.claude/`, `.opencode/`)
- Colony framing and atmosphere (Queen persona, ant metaphor)
- Pre/post-build narration and context
- Pacing guidance (what to do before, during, after)
- Error recovery suggestions (within runtime-provided boundaries)
- User-facing summaries (within length constraints)

### Wrappers Must Not
- Mutate state files directly
- Replay build/continue orchestration logic
- Parse visual text as truth
- Duplicate verification or gating logic
- Add extra option menus not provided by the runtime

### YAML Source Chain
- `.aether/commands/*.yaml` — Source definitions for wrapper commands (49 files)
- `.claude/commands/ant/*.md` — Claude Code wrappers (generated from YAML sources)
- `.opencode/commands/ant/*.md` — OpenCode wrappers (generated from YAML sources)
- YAML files define: name, description, runtime command, guardrails, follow-up actions
- Wrapper content must stay within the boundaries defined by their YAML source

### Codex UX
- Codex does NOT use wrapper markdown
- Codex UX comes entirely from the runtime visual renderer (`cmd/codex_visuals.go`)
- Codex improvements should target the Go runtime, not wrapper simulation
- Codex agents use `.codex/agents/*.toml` definitions, not slash commands
