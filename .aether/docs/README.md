# Aether Documentation

This directory contains actively maintained documentation for the Aether colony system. 8 files total plus `disciplines/` subdirectory.

---

## User-Facing Docs

Distributed to target repos via `aether update` (update allowlist):

| File | Purpose |
|------|---------|
| `pheromones.md` | Pheromone system guide (FOCUS/REDIRECT/FEEDBACK signals) |

---

## Colony System Docs

Packaged in npm, available to all Aether installations:

| File | Purpose |
|------|---------|
| `caste-system.md` | Worker caste definitions and emoji assignments |
| `QUEEN-SYSTEM.md` | Queen wisdom promotion system |
| `queen-commands.md` | Queen command documentation |
| `QUEEN.md` | Generated Queen wisdom file (repo-specific, auto-updated) |
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

## Archived Docs

Historical documentation moved to `archive/` subdirectory:

- `QUEEN_ANT_ARCHITECTURE.md` - superseded by agent files
- `implementation-learnings.md` - historical findings
- `constraints.md` - content now in agent definitions
- `pathogen-schema.md` - specialized use case
- `pathogen-schema-example.json` - example for schema
- `progressive-disclosure.md` - design philosophy

Archived docs remain available for reference but are not actively maintained.
