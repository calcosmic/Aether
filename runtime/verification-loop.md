# Verification Loop Discipline

## Purpose

A comprehensive 6-phase quality check that runs before phase advancement or PR creation. Complements the core Verification Discipline (evidence before claims) with systematic quality gates.

## When to Invoke

- After completing a phase (via `/ant:continue`)
- Before creating a PR
- After significant refactoring
- After major changes (every 15 minutes in long sessions)

## The 6 Phases

### Phase 1: Build Verification

```bash
# Detect and run build command
npm run build 2>&1 | tail -30
# OR: pnpm build, cargo build, go build ./..., python -m build
```

**Gate:** If build fails, STOP. Fix before continuing.

### Phase 2: Type Check

```bash
# TypeScript
npx tsc --noEmit 2>&1 | head -30

# Python
pyright . 2>&1 | head -30

# Go
go vet ./... 2>&1 | head -30
```

**Gate:** Report all type errors. Fix critical ones before continuing.

### Phase 3: Lint Check

```bash
# JavaScript/TypeScript
npm run lint 2>&1 | head -30

# Python
ruff check . 2>&1 | head -30

# Go
golangci-lint run 2>&1 | head -30
```

**Gate:** Report warnings. Fix errors before continuing.

### Phase 4: Test Suite

```bash
# Run tests with coverage
npm run test -- --coverage 2>&1 | tail -50

# Python
pytest --cov=. 2>&1 | tail -50
```

Report:
- Total tests: X
- Passed: X
- Failed: X
- Coverage: X% (target: 80%+)

**Gate:** If tests fail, STOP. Fix before continuing.

### Phase 5: Security Scan

```bash
# Check for exposed secrets
grep -rn "sk-\|api_key\|password\s*=" --include="*.ts" --include="*.js" --include="*.py" src/ 2>/dev/null | head -10

# Check for debug artifacts
grep -rn "console\.log\|debugger\|TODO.*REMOVE" --include="*.ts" --include="*.tsx" --include="*.js" src/ 2>/dev/null | head -10
```

Report:
- Potential secrets: X
- Debug artifacts: X

**Gate:** Fix exposed secrets immediately. Debug artifacts are warnings.

### Phase 6: Diff Review

```bash
# Show what changed
git diff --stat
git diff HEAD~1 --name-only
```

Review each changed file for:
- Unintended changes (files touched that shouldn't be)
- Missing error handling
- Potential edge cases
- Leftover debugging code

**Gate:** Review before advancing. Flag concerns.

## Verification Report Format

After running all phases, produce a verification report:

```
VERIFICATION LOOP REPORT
========================

Phase 1: Build      [PASS/FAIL]
Phase 2: Types      [PASS/FAIL] (X errors)
Phase 3: Lint       [PASS/FAIL] (X warnings)
Phase 4: Tests      [PASS/FAIL] (X/Y passed, Z% coverage)
Phase 5: Security   [PASS/FAIL] (X issues)
Phase 6: Diff       [X files changed]

Overall: [READY/NOT READY] for advancement

Issues to Fix:
1. ...
2. ...
```

## Command Detection

Resolve each command (build, test, types, lint) independently using the priority chain below. Use the first source that provides a match for each label; do not mix sources for the same label.

### Command Resolution Priority

1. **CLAUDE.md** — Check the project's CLAUDE.md (in LLM system context) for explicit commands under headings like `Commands`, `Scripts`, `Development`, `Build`, `Testing`, or `Lint`.
2. **CODEBASE.md** — Read `.aether/data/codebase.md` `## Commands` section for commands detected during `/ant:colonize`. Each entry includes source attribution (`claude_md` or `heuristic`).
3. **Fallback Heuristic Table** — Use the table below if neither source provides a command for the needed label.

### Fallback Heuristic Table

| File | Build | Test | Types | Lint |
|------|-------|------|-------|------|
| `package.json` | `npm run build` | `npm test` | `npx tsc --noEmit` | `npm run lint` |
| `Cargo.toml` | `cargo build` | `cargo test` | (built-in) | `cargo clippy` |
| `go.mod` | `go build ./...` | `go test ./...` | `go vet ./...` | `golangci-lint run` |
| `pyproject.toml` | `python -m build` | `pytest` | `pyright .` | `ruff check .` |
| `Makefile` | `make build` | `make test` | (check targets) | `make lint` |

## Skip Conditions

If command doesn't exist for project:
- Skip that phase
- Note "N/A" in report
- Don't fail the loop

## Integration with Hooks

This discipline provides comprehensive review. Hooks catch issues immediately during development. Use both:
- Hooks: Real-time feedback
- Verification Loop: Comprehensive gate before advancement

## Why This Matters

- Catches issues before they compound
- Ensures consistent quality gates
- Prevents shipping broken code
- Creates audit trail of verification
- Builds confidence in phase completion
