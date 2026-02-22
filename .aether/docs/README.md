# Aether Documentation

This directory contains actively maintained documentation for the Aether colony system.

Runtime/behavior authority remains in:
- `.aether/aether-utils.sh` (execution)
- `.claude/commands/ant/*.md` (command flow)

Docs in this directory are explanatory references and should not override runtime behavior.

Distribution note:
- `aether update` does not sync repo-local session files like `.aether/CONTEXT.md` and `.aether/HANDOFF.md`.

---

## User-Facing Docs

Distributed to target repos via `aether update` (update allowlist):

| File | Purpose |
|------|---------|
| `pheromones.md` | Pheromone system guide (FOCUS/REDIRECT/FEEDBACK signals) |
| `source-of-truth-map.md` | Authority map and docs/runtime drift tracking |

---

## Colony System Docs

Packaged in npm, available to all Aether installations:

| File | Purpose |
|------|---------|
| `caste-system.md` | Worker caste definitions and emoji assignments |
| `QUEEN-SYSTEM.md` | Queen wisdom promotion system |
| `queen-commands.md` | Queen command documentation |
| `xml-utilities.md` | XML utility/runtime integration reference |
| `.aether/QUEEN.md` | Generated Queen wisdom file (repo-specific, auto-updated) |
| `error-codes.md` | Error code reference (E_* constants) |

---

## Development Docs

Packaged in npm, documents active issues:

| File | Purpose |
|------|---------|
| `known-issues.md` | Active known issues and workarounds |

---

## Worker Disciplines

Training protocols that govern worker behavior (in `disciplines/` subdirectory):

| File | Purpose |
|------|---------|
| `disciplines/DISCIPLINES.md` | Discipline index and overview |
| `disciplines/verification.md` | No completion claims without evidence |
| `disciplines/verification-loop.md` | 6-phase quality gate before advancement |
| `disciplines/debugging.md` | Systematic root cause investigation |
| `disciplines/tdd.md` | Test-first development |
| `disciplines/learning.md` | Pattern detection with validation |
| `disciplines/coding-standards.md` | Universal code quality rules |

---

## Command Playbooks

Split playbooks used by orchestrator commands:

| File | Purpose |
|------|---------|
| `command-playbooks/build-prep.md` | Build preparation and validation |
| `command-playbooks/build-context.md` | Context and survey loading |
| `command-playbooks/build-wave.md` | Worker wave orchestration |
| `command-playbooks/build-verify.md` | Watcher/measurer/chaos verification |
| `command-playbooks/build-complete.md` | Build synthesis and session updates |
| `command-playbooks/continue-verify.md` | Continue verification setup |
| `command-playbooks/continue-gates.md` | Continue quality/security gates |
| `command-playbooks/continue-advance.md` | State advancement and pheromone/learning steps |
| `command-playbooks/continue-finalize.md` | Handoff/changelog/session finalization |

---

## Archived Docs

Historical documentation moved to `archive/` subdirectory:

- `QUEEN_ANT_ARCHITECTURE.md` - superseded by agent files
- `implementation-learnings.md` - historical findings
- `constraints.md` - content now in agent definitions
- `pathogen-schema.md` - specialized use case
- `pathogen-schema-example.json` - example for schema
- `progressive-disclosure.md` - design philosophy

Archived docs remain available for reference but are not actively maintained.
