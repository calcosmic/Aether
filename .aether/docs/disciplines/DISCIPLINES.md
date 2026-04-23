# Aether Colony Disciplines

**Updated:** 2026-03-24

## Overview

The Aether ant colony system includes 7 disciplines (6 core + 1 role-specific) that govern worker behavior. These disciplines are infused directly into worker prompts and command execution.

## Honest Execution Model

**What the colony provides:**
- Task organization and decomposition (real)
- State persistence across sessions (real)
- Parallel execution via Task tool (real, when explicitly used)
- Structured verification gates (real)

**What it does NOT provide:**
- Automatic parallel execution (must be explicitly spawned)
- Magic emergence (follows structured commands)
- Guaranteed correctness (verification catches issues)

## Discipline Reference

### Core Disciplines (All Workers)

| Discipline | File | Purpose |
|-----------|------|---------|
| Verification | `verification.md` | No completion claims without evidence |
| Verification Loop | `verification-loop.md` | 6-phase quality gate before advancement |
| Debugging | `debugging.md` | Systematic root cause investigation |
| TDD | `tdd.md` | Test-first development |
| Learning | `learning.md` | Pattern detection with validation |
| Coding Standards | `coding-standards.md` | Universal code quality rules |

### Role-Specific Disciplines

| Discipline | File | Applies To |
|-----------|------|------------|
| Planning | `planning.md` (planned, not yet created) | Route-Setter |

## Learning Validation

Learnings are NOT automatically trusted. They follow a validation lifecycle:

```
hypothesis → validated → (or) disproven
```

- **hypothesis**: Recorded but not yet tested (default)
- **validated**: Tested and confirmed working
- **disproven**: Found to be incorrect

Instincts track success/failure counts and can be automatically disproven.

## Verification Enforcement

`/ant-continue` enforces verification:
- Build must pass
- Tests must pass
- Success criteria must have evidence
- **Phase will NOT advance without passing verification**

No workarounds. Fix issues and re-run.

## Command Integration

| Command | Key Behaviors |
|---------|---------------|
| `/ant-build` | Real parallelism via Task tool, honest logging |
| `/ant-continue` | Mandatory verification gate, learning extraction |
| `/ant-plan` | Bite-sized task planning |
| `/ant-status` | Colony state with instincts |

## File Structure

```
.aether/
├── workers.md               # Worker roles + honest execution model
├── docs/disciplines/
│   ├── coding-standards.md  # Code quality rules
│   ├── debugging.md         # Systematic debugging
│   ├── learning.md          # Colony learning system
│   ├── tdd.md               # Test-driven development
│   ├── verification.md      # Evidence before claims
│   ├── verification-loop.md # 6-phase quality gate
│   └── DISCIPLINES.md       # This file
```

Note: `planning.md` is referenced in the Role-Specific Disciplines table above but has not yet been created.

## Reinstall After Updates

```bash
cd /Users/callumcowie/repos/Aether && npm run postinstall
```
