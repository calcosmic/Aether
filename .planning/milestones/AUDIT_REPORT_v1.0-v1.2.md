# Aether Colony Audit Report: v1.0 -- v1.2

**Audit Date:** 2026-04-21
**Audited Version:** v1.0.17
**Scope:** Runtime truth, wrapper honesty, Codex parity, worker dispatch, lifecycle, recovery, docs, tests, versioning
**Method:** Live CLI testing, source inspection, wrapper comparison, test execution

---

## Executive Summary

**Overall Assessment: SOLID with 3 confirmed bugs and 1 performance issue.**

The Aether system is largely working as intended. The Go runtime owns truth correctly. Wrappers are honest and delegate to the runtime. All 2900+ tests pass. Agent and command counts are consistent across all three platforms. Versioning is accurate.

The main issues are:
1. **3 CLI flag mismatches** between wrapper markdown and Go CLI flags
2. **`aether plan` hangs** when spawning planning workers (reproducible)
3. **No standalone `plan_cmd.go`** -- plan logic lives in `codex_plan.go` (naming inconsistency)

---

## Findings by Area

### 1. Runtime Truth & Core CLI -- MOSTLY GOOD

**What's working:**
- All core commands exist and have proper help: `init`, `build`, `continue`, `status`, `watch`, `colonize`, `run`, `seal`, `update`, `version`, `proof`, `skill-list`, `parallel-mode`, `hive-read`, `hive-store`, `midden-write`, `midden-review`, `event-bus-publish`, and 80+ subcommands.
- Caste identity system is implemented in `cmd/codex_visuals.go` with proper color maps, emoji maps, and label maps.
- State mutation commands (`init`, `build`, `continue`, `seal`) actually mutate `.aether/data/COLONY_STATE.json`.
- `aether version` returns `1.0.17` correctly.
- All tests pass: `go test ./cmd/...` and `go test ./pkg/...` both green.

**Issues found:**

| Severity | Issue | Evidence |
|----------|-------|----------|
| HIGH | `aether plan` hangs indefinitely when spawning Scout + Route-Setter workers | Reproduced twice: initial plan hung, `plan --force` timed out after 30s. Process `aether plan` stays alive but produces no output beyond dispatch banner. |
| MEDIUM | Plan command implementation is in `codex_plan.go` instead of `plan_cmd.go` | File naming implies Codex-only, but this is the universal plan command. Misleading for contributors. |
| LOW | `aether resume` and `aether resume-colony` are aliases with no flags or options | Acceptable for simple restore, but no `--full` vs `--quick` differentiation as docs suggest. |

**Suggested fixes:**
- Add a timeout or progress indicator to `aether plan` worker spawning. Consider making planning synchronous with timeout rather than open-ended agent spawn.
- Rename `codex_plan.go` to `plan_cmd.go` for clarity.

---

### 2. Wrapper Honesty & CLI Flag Mismatch -- 3 BUGS CONFIRMED

**What's working:**
- Wrappers are **honest**. They explicitly instruct:
  - "Do not write `.aether/data/COLONY_STATE.json`, `session.json`, `constraints.json`, or `pheromones.json` by hand" (`init.md:17`)
  - "Do NOT read or write colony state files by hand" (`build.md:71`, `continue.md:75`)
  - "The runtime owns all state transitions, worker dispatch, verification, and next-step truth" (`build.md:57`)
- Wrappers add colony framing (Queen persona, narration, stage markers) but do NOT duplicate verification or gating logic.
- 49 commands in Claude, 49 in OpenCode, 49 YAML sources. Perfect count parity.

**Issues found:**

| Severity | Issue | File | Evidence |
|----------|-------|------|----------|
| MEDIUM | `council-deliberate --proposal` flag does not exist; actual flag is `--topic` | `.claude/commands/ant/council.md:13` and `.opencode/commands/ant/council.md:13` | `aether council-deliberate --help` shows `--topic string` |
| MEDIUM | `data-clean --dry-run` flag does not exist; actual flag is `--confirm` (default dry-run) | `.claude/commands/ant/data-clean.md:20` and `.opencode/commands/ant/data-clean.md:20` | `aether data-clean --help` shows only `--confirm` |
| MEDIUM | `generate-ant-name --caste` flag does not exist; actual arg is positional `[caste]` | `.claude/commands/ant/quick.md:36` and `.opencode/commands/ant/quick.md:36` | `aether generate-ant-name --help` shows `[caste]` positional, `--seed int` |

**Suggested fixes:**
- Update `.claude/commands/ant/council.md` and `.opencode/commands/ant/council.md` to use `--topic`
- Update `.claude/commands/ant/data-clean.md` and `.opencode/commands/ant/data-clean.md` to use `--confirm` (explain it defaults to dry-run)
- Update `.claude/commands/ant/quick.md` and `.opencode/commands/ant/quick.md` to use positional arg: `aether generate-ant-name scout`
- Add a CI check or linter that validates all `aether <cmd> --<flag>` patterns in markdown against the actual CLI

---

### 3. Codex Parity -- GOOD

**What's working:**
- All 24 agents present in `.codex/agents/*.toml`.
- Agent mirrors are **byte-identical**:
  - `.aether/agents-claude/` == `.claude/agents/ant/` (verified with `diff -rq`)
  - `.aether/agents-codex/` == `.codex/agents/` (verified with `diff -rq`)
- TOML agents contain equivalent content to MD agents (same TDD discipline, activity logging instructions, etc.)
- `CODEX.md` references 40 commands, all of which exist in the Go CLI.
- Codex is truly runtime-native: no wrapper commands directory, just `CODEX.md` + agents.

**Issues found:**

| Severity | Issue | Evidence |
|----------|-------|----------|
| LOW | Codex agents use `aether activity-log --command` syntax in their instructions, but `--command` is not a valid flag for `activity-log` | Checked `aether activity-log --help` -- no `--command` flag exists. Activity-log takes free-form args. |

**Suggested fix:**
- Update Codex agent instructions to use `aether activity-log "ACTION" "details"` instead of `aether activity-log --command`

---

### 4. Worker Dispatch, Lifecycle & Recovery -- MOSTLY GOOD

**What's working:**
- `aether build` exists with `--task` flag for redispatch.
- `aether continue` exists with `--reconcile-task` flag.
- `aether run` (autopilot) exists with `--max-phases`, `--replan-interval`, `--dry-run`, `--headless`.
- `aether watch` exists with `--interval` and `--once`.
- `aether proof` exists and inspects context/skill proof.
- `aether skill-list` works and returns JSON.
- Recovery commands (`resume`, `resume-colony`) exist.
- Build/continue playbooks exist in `.aether/docs/command-playbooks/`.

**Issues found:**

| Severity | Issue | Evidence |
|----------|-------|----------|
| HIGH | `aether plan` worker spawning hangs (same as Phase 1) | Reproduced. The Scout/Route-Setter dispatch never completes or produces output. |
| LOW | `aether resume` has no differentiation between "quick restore" and "full recovery view" | Both `resume` and `resume-colony` are aliases with identical help. Docs claim `/ant:resume` is quick and `/ant:resume-colony` is full, but the CLI makes no distinction. |

**Suggested fixes:**
- Fix `aether plan` worker spawning timeout (same as Phase 1).
- Add `--full` flag to `resume-colony` or differentiate the aliases in the Go code.

---

### 5. Documentation, Tests & Versioning -- GOOD

**What's working:**
- All tests pass: `cmd` (32.5s) and all `pkg/*` packages green.
- Version is consistent: `1.0.17` in `.aether/version.json`, `aether version` CLI, README badge, CHANGELOG.
- Documentation counts are accurate: 24 agents, 49 commands, 28 skills.
- `CLAUDE.md`, `README.md`, `CHANGELOG.md` all reference the correct version.
- No stale file paths detected in top-level docs.

**Issues found:**

| Severity | Issue | Evidence |
|----------|-------|----------|
| LOW | `RUNTIME UPDATE ARCHITECTURE.md` doc title uses old naming convention | File exists and is referenced, but "UPDATE" in the filename suggests it was a working doc that became canonical. |
| LOW | `codex_plan.go` filename is misleading (see Phase 1) | Same issue. |

---

## Summary Table

| # | Severity | Area | Finding | Status |
|---|----------|------|---------|--------|
| 1 | HIGH | Runtime | `aether plan` hangs when spawning workers | **Bug** |
| 2 | MEDIUM | Runtime | `codex_plan.go` should be `plan_cmd.go` | **Naming debt** |
| 3 | MEDIUM | Wrappers | `council-deliberate --proposal` → actual flag is `--topic` | **Flag mismatch** |
| 4 | MEDIUM | Wrappers | `data-clean --dry-run` → actual flag is `--confirm` | **Flag mismatch** |
| 5 | MEDIUM | Wrappers | `generate-ant-name --caste` → positional arg `[caste]` | **Flag mismatch** |
| 6 | LOW | Codex | Agents reference `activity-log --command` which doesn't exist | **Doc bug** |
| 7 | LOW | Recovery | `resume` and `resume-colony` are identical aliases | **UX gap** |
| 8 | LOW | Docs | `RUNTIME UPDATE ARCHITECTURE.md` has working-doc name | **Naming debt** |

---

## Overall Verdict

**The Aether system is working.**

- Runtime truth is real, not faked.
- Wrappers are honest and delegate correctly.
- Codex parity is maintained (24 agents, runtime-native).
- All tests pass.
- Versioning is consistent.

The main risks are:
1. **The `plan` hang** -- this affects the most common user flow (`/ant:plan`). It needs a timeout or synchronous fallback.
2. **The 3 CLI flag mismatches** -- these will cause wrapper instructions to fail when executed. They need one-line fixes.
3. **The Codex `activity-log` instruction** -- similar small fix.

No critical security issues, no data loss risks, no major parity gaps. The system is solid for v1.0.17.
