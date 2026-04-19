# Aether Source-of-Truth Map

Updated: 2026-04-18

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
| Codex agents | `.codex/agents/*.toml` | `.aether/agents-codex/*.toml`, `~/.aether/system/codex/` |
| Claude agents | `.claude/agents/ant/*.md` | `.aether/agents-claude/*.md` |
| Shared skills | `.aether/skills/**/SKILL.md` | `.aether/skills-codex/**/SKILL.md`, `~/.aether/system/skills-codex/` |
| Claude commands | `.claude/commands/ant/*.md` | Claude Code |
| OpenCode commands | `.opencode/commands/ant/*.md` | OpenCode |
| Codex guidance | `AGENTS.md`, `.codex/CODEX.md` | Codex CLI |
| Session recovery | `.aether/data/session.json` | `.aether/CONTEXT.md`, `.aether/HANDOFF.md`, `aether resume` |
| Spawn activity | `.aether/data/spawn-tree.txt` | `aether status`, `aether swarm --watch` |

## Verified Inventory

| Category | Location | Count |
|---|---|---:|
| Claude commands | `.claude/commands/ant/*.md` | 46 |
| OpenCode commands | `.opencode/commands/ant/*.md` | 46 |
| Claude agents | `.claude/agents/ant/*.md` | 24 |
| OpenCode agents | `.opencode/agents/*.md` | 24 |
| Codex agents | `.codex/agents/*.toml` | 24 |
| Shared skills | `.aether/skills/**/SKILL.md` | 28 |
| Codex skill mirror | `.aether/skills-codex/**/SKILL.md` | 28 |

## Notes

- `aether resume` is the canonical Codex-facing alias for `resume-colony`.
- `aether run`, `aether watch`, and `aether oracle` are canonical Codex-facing compatibility entrypoints.
- `aether swarm` is now the Codex compatibility entrypoint for explicit swarm routing and live worker watch mode.
- `export-signals` / `import-signals` are flat aliases over the pheromone XML commands.
- Distribution is driven by the Go binary and embedded companion assets; the repo no longer uses `package.json` as the release authority.

## Maintenance Rules

1. Change runtime behavior in `cmd/` / `pkg/` first.
2. Update the Claude/OpenCode markdown mirrors in the same change when command syntax or UX changes.
3. Update Codex docs (`AGENTS.md`, `.codex/CODEX.md`) when native CLI semantics, install/update behavior, safety guarantees, or Codex-specific guidance changes.
4. Keep packaged mirrors synchronized with their source trees.
5. Treat `.aether/data/` and generated handoff/context files as outputs, not specs.
