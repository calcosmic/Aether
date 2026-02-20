# Aether Documentation

This directory contains actively maintained documentation for the Aether colony system. 13 files total — no subdirectories.

---

## User-Facing Docs

Distributed to target repos via `aether update` (update allowlist):

| File | Purpose |
|------|---------|
| `pheromones.md` | Pheromone system guide (FOCUS/REDIRECT/FEEDBACK signals) |
| `constraints.md` | Colony constraint definitions |
| `pathogen-schema.md` | Pathogen detection schema |
| `pathogen-schema-example.json` | Pathogen schema example |
| `progressive-disclosure.md` | Progressive disclosure guide |

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

## Architecture

| File | Purpose |
|------|---------|
| `QUEEN_ANT_ARCHITECTURE.md` | Queen escalation chain and coordination patterns |

---

## Development Docs

Packaged in npm, documents active issues and learnings:

| File | Purpose |
|------|---------|
| `known-issues.md` | Active known issues and workarounds |
| `implementation-learnings.md` | Extracted implementation findings |
