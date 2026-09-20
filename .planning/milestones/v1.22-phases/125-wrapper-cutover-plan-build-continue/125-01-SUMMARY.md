# Plan 125-01 Summary: Update Source-of-Truth Layer for TS Host Delegation

**Phase:** 125 — Wrapper Cutover (Plan/Build/Continue)  
**Plan:** 125-01  
**Status:** Complete  
**Commit:** cdb9bb19

## What Was Done

### Task 1: Update YAML Command Sources
Updated `.aether/commands/plan.yaml`, `.aether/commands/build.yaml`, and `.aether/commands/continue.yaml`:

- **runtime.command** now uses `aether host <workflow>` as the primary path:
  - `plan`: `aether host plan --depth <fast|balanced|deep|exhaustive> --planning-depth <light|standard|deep>`
  - `build`: `aether host build $ARGUMENTS`
  - `continue` (heavy path): `aether host continue --verification-depth heavy $ARGUMENTS`
- **runtime.fallback** added to all three YAML files with direct Go CLI commands for when the TS host is unavailable (Node missing, assets not built)
- **wrapper_additions.orchestration** updated to reference TS host commands with fallback language
- Finalizer calls (`plan-finalize`, `build-finalize`, `continue-finalize`) remain direct Go CLI commands per the boundary contract (TS host does not handle finalizers)
- All guardrails, follow_up, and non-command prose preserved exactly

### Task 2: Update cmd/command_guide.go and Codex Skill
Updated `cmd/command_guide.go`:
- Plan entry PreSteps: manifest fetch now references `aether host plan` with fallback
- Build entry PreSteps: manifest fetch now references `aether host build` with fallback
- Continue entry PreSteps: heavy review manifest fetch now references `aether host continue` with fallback

Updated `.aether/skills/colony/aether-colony-build-cycle/SKILL.md`:
- Plan Flow step 3: references `aether host plan` with fallback
- Build Flow step 3: references `aether host build` with fallback
- Continue Flow heavy path: references `aether host continue` with fallback

## Deviation from Plan

None. All tasks executed as specified.

## Verification

- `grep -n "aether host" .aether/commands/plan.yaml .aether/commands/build.yaml .aether/commands/continue.yaml` — all three files show TS host references
- `grep -n "aether host" cmd/command_guide.go` — 3 references (plan, build, continue)
- `grep -n "aether host" .aether/skills/colony/aether-colony-build-cycle/SKILL.md` — 3 references (plan, build, continue)
- `go test ./cmd -run TestCommandGuide -v` — PASS
- `go test ./...` — all packages PASS

## Cross-Platform Drift Guard

All four artifacts updated together:
1. `.aether/commands/plan.yaml` — source of truth
2. `.aether/commands/build.yaml` — source of truth
3. `.aether/commands/continue.yaml` — source of truth
4. `cmd/command_guide.go` — Codex orchestration guide
5. `.aether/skills/colony/aether-colony-build-cycle/SKILL.md` — Codex skill
