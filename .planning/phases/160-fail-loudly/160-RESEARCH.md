# Phase 160: Fail Loudly - Research

**Researched:** 2026-07-27
**Domain:** Go CLI contract correctness, dead-vs-live prompt/playbook execution paths, worktree-safe debug artifacts, safe deletion of an unused TypeScript package
**Confidence:** HIGH (all claims below were verified directly against the current tree via `grep`/`Read`, not recalled from training data)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Reconciliation table (verified by execution, 2026-07-27)** — the plan must start from this table, not the raw requirement list:

| Requirement | Verified state 2026-07-27 (per CONTEXT.md) |
|---|---|
| LOUD-01 (survey-load) | Done differently: command deleted, zero references remain. Verify: absence + no stale callers |
| LOUD-02 (check-antipattern gate) | Done: positional form fixed, pinned by `cmd/security_gate_drift_test.go` |
| LOUD-03 (five remaining calls) | All exist and error loudly when misused; needs the execution audit to formally pin |
| LOUD-04 (positional drift test) | NOT verified — `cmd/cli_flag_audit_test.go` positional coverage unknown |
| LOUD-05 (execution audit) | NOT done — no run-the-commands audit exists |
| LOUD-06 (stderr suppression) | Mostly done: only 4 benign `2>/dev/null` sites remain (grep fallbacks in archaeology.md, git call in dream.md) |
| LOUD-07 (docs claim consolidation runs) | Almost: structural-learning-stack.md + AGENTS.md corrected; `CLAUDE.md:838` still claims "phase-end at /ant-continue" while `CLAUDE.md:37` says the opposite |
| LOUD-08 (/ant-unblock) | NOT done — `cmd/unblock_cmd.go:126` still points at nonexistent command |
| RETIRE-01 (delete control-ts) | NOT done — directory exists |
| RETIRE-02/03 (keep ts-host, ts) | Confirmed live and load-bearing |
| RETIRE-04 (deleted-test ledger) | NOT done; `control-ts/tests/schemas/policy.schema.test.ts` replacement constraint stands. `.aether/ts-host/test/playbook-loader.test.ts` already deleted (commit `b2b41486`), record retroactively |

**D-01 (failure behavior):** Severity depends on the call. Safety gates (security scan, verification, claims checks) HALT the run when they cannot execute. Context/enrichment calls (survey, skill matching, progress rendering) show a loud, visible warning and the run continues degraded. The plan must classify every audited call into gate vs enrichment and encode the classification where the audit can test it.

**D-02 (/ant-unblock):** Build the missing `/ant-unblock` wrapper on Claude Code and OpenCode (YAML source + generated wrappers, same chain as other commands), so `cmd/unblock_cmd.go`'s guidance becomes true instead of being reworded.

**D-03 (debug artifacts on every failure mode):** Written on parse failure (exists today), timeout, and non-zero exit — with exit code, duration, and provider session id.

**D-04 (retention):** Cap (suggested 50 files or 14 days, pruned on write); wire into `aether data-clean`.

**D-05 (worktree survival):** In worktree mode, debug artifacts write to the tracking root (not the worktree), so they survive `git worktree remove`, and the `(debug: <path>)` hint resolves from the user's cwd.

### Claude's Discretion
- Ledger format/location for RETIRE-04, drift-audit mechanics (LOUD-04/05), gate-vs-enrichment classification details, wrapper wording for /ant-unblock.
- CLAUDE.md:838 fix is a one-line deletion/rewrite — fold into LOUD-07 work.

### Deferred Ideas (OUT OF SCOPE)
- Route-setter/scout permission contradictions (`pkg/codex/permission_profile.go:96`, `cmd/phase_research.go:124`) — Phase 163
- `artifacts` sub-schema only accepts `{}` (`pkg/codex/worker.go:921-926`) — Phase 163
- Pulling Phase 168 visual work earlier — raised, not selected
- `AETHER_PREFLIGHT_TIMEOUT` env knob; deleting dead TS preflight in `platform-dispatcher.ts` — any phase
- Stale worktree copies under `cmd/.aether/worktrees/` — cleanup candidate, no phase assigned

**Hard constraints (do not plan around these):**
- `.aether/ts-host/` and `.aether/ts/` are KEPT — embedded via `/embedded_assets.go:13` (repo root, **not** `cmd/embedded_assets.go` — see Reconciliation Corrections).
- Editing `.aether/ts-host/src/*.ts` requires `npm --prefix .aether/ts-host run build`.
- `build.md` structural ownership belongs to Phase 165. This phase may only make narrow, isolated call-argument fixes inside it.
- Agent bodies must stay Claude/OpenCode parity-identical (`TestClaudeOpenCodeAgentContentParity`, `cmd/codex_e2e_test.go:892`).
- Hub-published files need `aether publish` after edit.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| LOUD-01 | `survey-load` works or playbooks corrected | Confirmed dead: command doesn't exist in Go; only reference is `colony/playbooks/build.md:45`, an unread third-copy playbook. No further code action needed — see Reconciliation Corrections. |
| LOUD-02 | `check-antipattern` Gatekeeper gate works and gates continue | CLI contract fixed (verified), but **zero live caller exists anywhere** — the file that calls it (`continue-gates.md`) is dead documentation, and the live Gatekeeper agent has no Bash tool. See Open Question 1 (most important finding of this research) and Code Examples for the wiring recommendation. |
| LOUD-03 | Five remaining calls fixed | All five (`print-next-up`, `verify-claims`, `state-checkpoint`, `generate-progress-bar`, `skill-detect`) are `cobra.NoArgs`/flag-based Go commands called with wrong positional args **only inside dead `.aether/docs/command-playbooks/*.md` files**. Fix is call-site correction, not Go changes. See Architecture Patterns and Common Pitfalls. |
| LOUD-04 | Drift test catches positional-arg drift | `cmd/cli_flag_audit_test.go` verified line-by-line: its flag-extraction regex only fires on `--flag` tokens; it has no logic that inspects `cmd.Args` (e.g. `cobra.NoArgs`) or notices a bare positional token following a subcommand. Concrete gap documented in Open Question 2. |
| LOUD-05 | Execution audit, not regex | No such audit exists. `cmd/security_gate_drift_test.go` is the only execution-based (not regex-based) precedent in the repo. See Open Question 1 for the generalization pattern and scope boundaries (dead vs. live files). |
| LOUD-06 | No load-bearing stderr suppression | Confirmed exactly 4 sites in the two **live** wrapper directories (`.claude/commands/ant/`, `.opencode/commands/ant/`), matching CONTEXT.md's count precisely. **Disagreement:** 179 additional `2>/dev/null` sites exist in the dead `.aether/docs/command-playbooks/` corpus, uncounted by CONTEXT.md's "4 remaining" framing. See Reconciliation Corrections. |
| LOUD-07 | Docs stop claiming consolidation runs today | **Disagreement with CONTEXT.md:** none of the three named files are corrected. `structural-learning-stack.md` and `AGENTS.md:899` both still assert present-tense that phase-end consolidation runs at `/ant-continue`. Only `CLAUDE.md` (lines 37 and 838, contradicting each other) matches CONTEXT.md's description. See Reconciliation Corrections. |
| LOUD-08 | `/ant-unblock` guidance is true | Confirmed: no `.aether/commands/unblock.yaml`, no `.claude/commands/ant/unblock.md`, no `.opencode/commands/ant/unblock.md` exist anywhere. `cmd/unblock_cmd.go` (136 lines) is a fully working Go command (`aether unblock --phase N [--dispatch] [--fixer-mode ...]`) with no wrapper. See Code Examples for the exact wrapper generation chain and golden files that need updating. |
| RETIRE-01 | Delete `control-ts/` | Confirmed zero Go references, zero CI references, zero embed references, zero goreleaser references. Safe to delete as a standalone commit. |
| RETIRE-02 | Keep `.aether/ts-host/` | Confirmed embedded at `/embedded_assets.go:13` (`all:.aether/ts-host/dist`, `package.json`, `package-lock.json`). Deleting breaks `go build`. |
| RETIRE-03 | Keep `.aether/ts/` | Confirmed embedded at `/embedded_assets.go:12` (source-level, not `dist/`). Deleting breaks `go build`. |
| RETIRE-04 | Ledger for deleted tests | No existing "dead test ledger" pattern in the repo (the closest analog, `review-ledger`, is a different system for phase-review findings). `control-ts/tests/schemas/policy.schema.test.ts` verified as the only schema check for `colony/policies/*.yaml`; `cmd/policy_loader.go` has zero typed schema validation today. See Open Question 4 for a concrete Go-side replacement proposal. |
</phase_requirements>

## Summary

This phase is not a feature-build phase — it is a correctness-and-truth audit of an existing system, so "research" here means **direct code archaeology**, not library selection. Every one of CONTEXT.md's reconciliation claims was re-verified against the current tree (commit `ec70f7e4`). Most of the table holds up. Two do not, and one item is technically true but incomplete in a way that matters for how the phase should be planned.

**The single most important finding:** `check-antipattern`'s CLI contract fix (LOUD-02, marked "Done" by CONTEXT.md) only repairs a command that **nothing in the live system calls**. The file that was supposed to call it, `.aether/docs/command-playbooks/continue-gates.md`, is confirmed-dead reference documentation — the TS host's `playbook-loader.ts` (the only thing that ever read these files) was deleted in commit `b2b41486`, and the live `.claude/commands/ant/continue.md` wrapper never mentions this file. Worse: the actually-dispatched "Gatekeeper" agent (`.claude/agents/ant/aether-gatekeeper.md`) has `tools: Read, Grep, Glob, Write` — **no Bash** — and its role is dependency/license supply-chain auditing, not source-code secret scanning. So even in the heavy-review path where a Gatekeeper agent does run, it is physically incapable of invoking `aether check-antipattern`. Success criterion #2 of this phase's own ROADMAP entry ("the Gatekeeper security gate actually executes during continue, and its pass/fail result visibly affects the continue outcome") is **not satisfied** by the CLI fix alone. Fortunately, the wiring target already exists half-built: `cmd/gate.go` has a fully-specified `"anti_pattern"` gate classification (`softBlock` tier, recovery template, auto-resolve threshold) that **no code in the repo ever populates**. This is the same "fully written, zero readers" pattern the entire v1.25 milestone is about — just found one phase early. See Open Question 1 for the concrete recommendation.

**Second finding:** LOUD-07 is materially more open than CONTEXT.md states. Only `CLAUDE.md` shows the described contradiction (line 37 honest, line 838 still false). `.aether/docs/structural-learning-stack.md` and `AGENTS.md:899` are **entirely uncorrected** — both assert in the present tense, with no caveat anywhere in either file, that phase-end consolidation runs at every `/ant-continue`. This is exactly the failure mode `CLAUDE.md`'s own Definition of Done section calls out by name.

**Third finding:** LOUD-06's "4 benign sites" count is correct but scoped only to the two live wrapper directories. The dead `.aether/docs/command-playbooks/` corpus contains 179 more `2>/dev/null` occurrences. Since these files don't execute, this is not a live-behavior bug — but the planner needs to decide explicitly whether LOUD-05's execution audit is scoped to live files only (recommended) or all documented calls including dead reference material.

**Primary recommendation:** Treat every one of the seven "broken calls" as two separate problems with two separate fixes: (1) a documentation-correctness fix inside `.aether/docs/command-playbooks/*.md` (cheap, mechanical, satisfies LOUD-03/04/05's literal text), and (2) for `check-antipattern` only, a real Go-native wiring fix into the live continue gate pipeline (the only one of the seven whose absence the ROADMAP's success criteria treat as user-observable). Do not conflate the two, and do not assume fixing (1) satisfies (2).

## Reconciliation Corrections

Presented as corrections to CONTEXT.md's table, each independently re-verified:

| Requirement | CONTEXT.md claim | What I verified | Verdict |
|---|---|---|---|
| LOUD-02 | "Done: positional form fixed, pinned by `cmd/security_gate_drift_test.go`" | True for the CLI contract (`cmd/security_gate_drift_test.go:18-65`, `cmd/security_cmds.go:25-55`). **But** zero live callers exist: no Go code, no `.claude`/`.opencode` wrapper, no agent definition invokes `check-antipattern` (verified by repo-wide grep excluding `control-ts/` and the dead command-playbooks corpus). The Gatekeeper agent (`.claude/agents/ant/aether-gatekeeper.md:5`) has no Bash tool. | **Materially incomplete** — CLI fix ≠ "gate actually executes during continue" (ROADMAP SC#2). |
| LOUD-06 | "Mostly done: only 4 benign `2>/dev/null` sites remain" | Confirmed exactly 4 in `.claude/commands/ant/*.md` + `.opencode/commands/ant/*.md` (archaeology.md ×3 grep fallbacks, dream.md ×1 git call — identical across both platform mirrors). Also found 179 occurrences in `.aether/docs/command-playbooks/*.md` (`grep -rn "2>/dev/null" .aether/docs/command-playbooks/*.md \| wc -l` → 179), none counted in CONTEXT.md's "4 remaining." | **True but incomplete** — correct only if scoped to live wrapper dirs. |
| LOUD-07 | "Almost: structural-learning-stack.md + AGENTS.md corrected; CLAUDE.md:838 still claims..." | `structural-learning-stack.md:14` ("The stack runs automatically at phase-end and seal — workers never call it directly"), `:207` ("Runs at the end of every phase (`/ant-continue`)"), `:227` (table row asserting the same) — all present-tense, uncorrected, no caveat in the file. `AGENTS.md:899` ("Lifecycle integration: phase-end at `aether continue`, full at `aether seal`") — also uncorrected, present tense. `AGENTS.md` header shows `Last Updated: 2026-05-20`, confirming it predates the Close-the-Loops session entirely. `CLAUDE.md:37` (honest) vs `CLAUDE.md:838` (false) confirmed exactly as described. | **Two of three files are NOT corrected** — only CLAUDE.md matches the described state. |
| Canonical ref: `cmd/embedded_assets.go:13` | Cited as the ts-host embed location | The `//go:embed all:.aether/ts-host/dist ...` directive is at `/embedded_assets.go:13` — **repo root**, package `aetherassets`, not inside `cmd/`. Line number is correct; path is not. | **Minor path error** — worth fixing in any downstream reference so a future `grep cmd/embedded_assets.go` doesn't come up empty. |
| LOUD-01, RETIRE-01/02/03 | Various "done" / "confirmed" claims | All independently re-verified true — see Phase Requirements table above for citations. | **Confirmed accurate.** |

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| CLI subcommand contracts (Args, flags) | Backend / Go runtime (`cmd/`) | — | Cobra command definitions are the single source of truth for what a call accepts; wrappers/docs must conform to them, not vice versa. |
| Drift/audit test suite | Backend / Go runtime (`cmd/*_test.go`) | — | Tests run in CI and locally via `go test`; no other tier can enforce cross-file contract consistency. |
| Playbook / wrapper markdown (`.aether/docs/command-playbooks/`, `.claude/commands/ant/`, `.opencode/commands/ant/`) | Orchestrating LLM prompt surface | Backend (source-of-truth headers) | These files are literally the prompt injected into Claude Code / OpenCode when a slash command runs. `.aether/docs/command-playbooks/*.md` is a third, currently-unexecuted layer (reference documentation only — confirmed no loader exists after `b2b41486`). |
| Continue gate pipeline (pass/fail decisions) | Backend / Go runtime (`cmd/gate.go`, `cmd/codex_continue*.go`) | — | Gate classification, auto-resolve thresholds, and hard/soft-block tiers are Go-owned constants; no wrapper or TS host code computes gate pass/fail. |
| Worker debug artifacts | Backend / Go runtime (`pkg/codex/platform_dispatch.go`) | Filesystem (`.aether/data/worker-debug/`) | Written by the same process that runs the worker subprocess; must resolve the tracking root (not worktree root) to survive `git worktree remove`. |
| `control-ts/` deletion | Filesystem / build tooling | Backend (`go build`, `aether publish`, `aether integrity`) | Pure removal; verified zero Go embed/CI/publish references, so no backend code path changes as a result. |
| `/ant-unblock` wrapper | Command wrapper markdown + YAML source | Backend (`cmd/unblock_cmd.go`, already complete) | The Go command is fully implemented; only the YAML source (`.aether/commands/unblock.yaml`) and the two markdown wrappers are missing. |

## Architecture Patterns

### Live vs. dead prompt-execution surfaces (the load-bearing distinction for this entire phase)

Aether has **three** layers of markdown that look similar but have completely different execution status as of the current tree:

1. **`.claude/commands/ant/*.md` / `.opencode/commands/ant/*.md` (60 each)** — LIVE. These are the literal prompt text injected when a user runs a slash command in Claude Code / OpenCode. Verified by reading `.claude/commands/ant/build.md` and `.claude/commands/ant/continue.md` in full: they drive real `aether` CLI invocations and real Task-tool worker spawns.
2. **`.aether/docs/command-playbooks/*.md`** — DEAD as of the current tree. Confirmed three independent ways:
   - CLAUDE.md's own "Command Playbooks (Reference Material)" section states they are "NOT loaded by the runtime and NOT injected into worker or orchestrator prompts."
   - `.claude/commands/ant/build.md` and `continue.md` never mention any command-playbooks filename (verified by full read — zero references to `build-context.md`, `build-prep.md`, `build-wave.md`, `build-complete.md`, `continue-gates.md`, `continue-verify.md`).
   - The only code that ever loaded them, `.aether/ts-host/src/playbook-loader.ts` (and its test), was **deleted** in commit `b2b41486` ("fix: stop losing worker research and mislabelling dispatch failures") on 2026-07-27 — the same session CONTEXT.md's reconciliation table cites. Commit `5b01f361` (one session earlier) already stopped injecting loaded playbook text into worker briefs: *"Playbooks remain loaded for orchestration; they are simply no longer glued onto individual worker prompts."* Then `b2b41486` removed the loader entirely.
3. **`colony/playbooks/*.md`, `colony/prompts/*`** — DEAD, a third parallel copy (confirmed no Go reader exists; this is explicitly RECLAIM-06 territory for Phase 170, out of scope here).

**Why this matters for planning:** All seven of the phase's "confirmed-broken calls" live exclusively in layer 2 or 3. Fixing their syntax is good documentation hygiene and will make LOUD-04/LOUD-05's audit pass, but does **not** change what a user observes running `/ant-build` or `/ant-continue` today, because nothing executes these files. Only `check-antipattern` is named in a hard, observable success criterion (ROADMAP SC#2), so it is the one call in this set that needs an actual live-path fix, not just a documentation fix.

### The gate classification scaffold already has a slot for this

`cmd/gate.go:610-627` defines `gateClassifications`, a compile-time map from gate name to `(tier, rationale)`. It already contains:

```go
// cmd/gate.go:621
"anti_pattern": {softBlock, "Critical patterns are actionable but non-blocking when addressed"},
```

This is distinct from `"gatekeeper"` (hardBlock, CVE/security-review focused, populated by the heavy-review dispatch's `review-ledger-write` calls). `"anti_pattern"` has a recovery template (`cmd/gate.go:532`) and an auto-resolve threshold entry (`cmd/gate.go:715`) — but **zero code anywhere produces a `GateCheckResult` named `"anti_pattern"`** (verified: no non-gate.go, non-test reference to the string `"anti_pattern"` exists in `cmd/`). This is the exact "fully specified, zero readers" pattern the whole v1.25 milestone targets, just discovered one phase early. D-01's classification work has almost no new design to do — the softBlock tier for antipattern findings was already decided by whoever wrote `gate.go`; it only needs a producer function.

### Recommended shape for wiring `check-antipattern` into continue (Open Question 1's answer)

Follow the existing `gateCheck` producer pattern (`checkNoCriticalFlags`, `cmd/gate.go:323-354`; `checkAllTasksCompleted`, `cmd/gate.go:357-411`): a new function, e.g. `checkAntiPatternGate(files []string) gateCheck`, that:
1. Takes the list of files changed during the phase. This is already collected in `cmd/codex_continue.go:2755` as `append(append(claims.FilesCreated, claims.FilesModified...), claims.TestsWritten...)` — reuse this, do not re-derive it from git.
2. Calls the same scanning logic `checkAntipatternCmd`'s `RunE` uses (`cmd/security_cmds.go:35-210`) — this logic should be extracted into a shared, unexported function callable both from the CLI command and from the new gate check, so the CLI contract and the gate check can never drift apart.
3. Returns `Passed: false` with the critical findings summarized when any file has criticals, `Passed: true` otherwise.
4. Is invoked from the continue pipeline (default fast path, not only the heavy-review dispatch path) so it runs on every continue at every verification depth, consistent with D-01 (this is a safety gate, not enrichment).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Antipattern/secret scanning logic | A second regex scanner for the new gate check | The existing scan logic in `cmd/security_cmds.go:70-210`, extracted into a shared function | Two independent implementations of the same ~6-pattern scan will drift; `check-antipattern` CLI and the new gate check must call one function. |
| Gate tier/classification model | A new classification scheme for gate-vs-enrichment | The existing `gateClassificationTier` enum (`hardBlock`/`softBlock`/`advisory`, `cmd/gate.go:595-599`) and the pre-populated `"anti_pattern"` entry | It already exists, fully specified, for exactly this purpose. |
| Changed-file discovery for the new gate | New git-diff parsing | `claims.FilesCreated` / `claims.FilesModified` / `claims.TestsWritten`, already assembled during continue (`cmd/codex_continue.go:2755`) | Avoids a second, possibly-inconsistent notion of "what changed this phase." |
| Debug-artifact retention pruning | A new cron-like retention system | Extend `cmd/maintenance.go`'s existing `dataCleanCmd` (currently prunes `pheromones.json` test artifacts only, `cmd/maintenance.go:18-80`) with a parallel step for `.aether/data/worker-debug/*.json` | One `data-clean` entry point, one mental model, one command for the user. |
| Worktree-safe path resolution | New logic to detect "am I in a worktree" | `workerTrackingRoot(config)` (`pkg/codex/worker.go:648-653`), already used for process tracking and designed for exactly this | Reuse the exact function already proven correct for a parallel problem (process tracking across worktrees). |
| `/ant-unblock` wrapper backend | New Go logic for gate recovery | `cmd/unblock_cmd.go` (136 lines, already complete: `--phase`, `--dispatch`, `--fixer-mode` flags, full recovery summary rendering) | The Go side of LOUD-08 needs zero changes; only the wrapper (YAML + 2×.md) is missing. |
| Policy schema validation replacement (RETIRE-04) | A full YAML policy loader/validator subsystem | A narrow Go test that parses `colony/policies/*.yaml` into loose structs/maps and asserts the same required fields `control-ts/tests/schemas/policy.schema.test.ts` currently asserts (`default_provider`, `memory_rules.*`, `skill_creation.*`, `safety_gates.*`, `dispatch_contract.*`) | `MODEL-01` (Phase 161) will build the real typed loader; Phase 160 only needs a replacement *test*, not a replacement *feature*. Overbuilding here duplicates Phase 161's work. |

**Key insight:** almost nothing in this phase requires new subsystems — every "don't hand-roll" entry above is "reuse a function/struct/enum that already exists three lines away from where the fix needs to land." That matches the ROADMAP's own framing ("not writing new subsystems").

## Common Pitfalls

### Pitfall 1: Treating "fixed the CLI call" as equivalent to "fixed the user-observable bug"
**What goes wrong:** A task fixes `aether check-antipattern`'s Args declaration (already done) and calls LOUD-02 satisfied.
**Why it happens:** The requirement text and the CLI fix share a command name, making it easy to conflate contract-correctness with runtime-wiring.
**How to avoid:** Require the plan to show a *live* invocation path (a real `/ant-continue` run, or a Go integration test that drives the actual continue command end-to-end) exercising `check-antipattern`'s logic, not just a unit test of the CLI command in isolation.
**Warning signs:** A verification command that only runs `go test -run TestCheckAntipattern` — this proves the CLI contract, not that continue calls it.

### Pitfall 2: Fixing all 179 dead-doc `2>/dev/null` sites under the "LOUD-06" banner
**What goes wrong:** A task edits all of `.aether/docs/command-playbooks/*.md` to remove stderr suppression, consuming significant effort on files nothing executes.
**Why it happens:** LOUD-05's text ("Every `aether` invocation across the playbooks and wrappers") reads as if it requires covering this whole corpus with equal priority to the live wrappers.
**How to avoid:** Scope LOUD-06's *behavior* fix (removing suppression that hides real failures from a real user) to the two live wrapper directories only (already at 4 known sites). Treat the command-playbooks corpus as a LOUD-05 audit *coverage* target (the audit should still enumerate and validate these calls for documentation correctness) but not a LOUD-06 *stderr-visibility* target, since nothing is currently silenced from a user who never sees these files execute.
**Warning signs:** A diff touching more than ~10 files under `.aether/docs/command-playbooks/` for a "stderr suppression" task.

### Pitfall 3: Rewriting `verify-claims` to accept the three positional args the dead playbook passes it
**What goes wrong:** `continue-verify.md:468` calls `aether verify-claims ".aether/data/last-build-claims.json" "<watcher_json_or_path>" "<test_exit_code>"`, but the actual Go command (`cmd/verify_claims.go:10-13`, `cobra.NoArgs`) takes no arguments at all and instead reads `COLONY_STATE.json` directly and runs fixed internal consistency checks. A tempting "fix" is to add positional-arg support to the Go command so the playbook's call becomes valid.
**Why it happens:** The call site strongly implies a feature (external claims-file comparison) that was never built.
**How to avoid:** This phase's goal statement explicitly excludes "writing new subsystems." The correct fix is to change the *call site* to `aether verify-claims` (no args), matching what the command actually does — not to build new comparison logic into the Go command. Same pattern applies to `skill-detect` (dead doc passes `"$(pwd)"`; real command takes no args) and `state-checkpoint` (dead doc passes a checkpoint name positionally; real command requires `--name`).
**Warning signs:** Any diff to `cmd/verify_claims.go`, `cmd/skills.go`, or `cmd/state_extra.go` for this phase — none of the three Go commands need to change; only their dead callers do.

### Pitfall 4: Assuming `cmd/embedded_assets.go` exists
**What goes wrong:** A task description or verification step references `cmd/embedded_assets.go:13` (as CONTEXT.md's canonical_refs does) and fails to find the file.
**Why it happens:** The embed directive is at the **repo root** `/embedded_assets.go` (package `aetherassets`), not inside `cmd/`.
**How to avoid:** Use the verified path `/embedded_assets.go:13` in any plan task or verification command.

### Pitfall 5: Writing debug artifacts to `config.Root` instead of the tracking root
**What goes wrong:** D-05's new timeout/non-zero-exit debug writes (added alongside the two existing call sites) pass `config.Root` to `writeHostedWorkerOutputDebug`, which is the worktree path in worktree mode — the artifact is deleted by `finalizeBuildWorktree` (`cmd/codex_build_worktree.go:452`) before anyone can read it.
**Why it happens:** `config.Root` is the natural-looking parameter to reach for; `workerTrackingRoot(config)` is a different, less obvious function used elsewhere in the same file for the same class of problem.
**How to avoid:** All `writeHostedWorkerOutputDebug` call sites (existing two at `pkg/codex/platform_dispatch.go:1032` and `:1040`, plus the two new ones for timeout/non-zero-exit) should pass `workerTrackingRoot(config)`, not `config.Root`.
**Warning signs:** A debug file that exists during the worker run but is gone when the user tries to open the path printed in the error message.

## Code Examples

### The exact gap in `cmd/cli_flag_audit_test.go` (LOUD-04)

```go
// cmd/cli_flag_audit_test.go:32 — extracts subcommand + a flags-only trailing group
re := regexp.MustCompile(`aether\s+([\w][\w-]*)\s+((?:--[\w][\w-]*(?:=\S*|\s+\S*)?\s*)*)`)
// ...
// cmd/cli_flag_audit_test.go:117 — only ever inspects `--flag` tokens from group 2
flagRe := regexp.MustCompile(`--([\w][\w-]*)`)
flagMatches := flagRe.FindAllStringSubmatch(flagsStr, -1)
```
Given the line `aether skill-detect "$(pwd)"` (from `build-context.md:220`), the outer regex's second capture group matches zero repetitions (the quoted `$(pwd)` token doesn't start with `--`), so `flagsStr` is empty and the positional argument is invisible to the test entirely. The test only ever validates: (a) does the subcommand exist, (b) do all `--flag` tokens exist on that subcommand. It never consults `cmd.Args` (e.g. `cobra.NoArgs`, `cobra.MaximumNArgs(1)`) and never notices a bare (non-flag) token following the subcommand name. **Concrete fix shape:** extend the outer regex (or add a second pass) to capture the raw remainder-of-line after the subcommand, split it into shell-like tokens, count how many are non-flag positional tokens, and compare that count against each registered command's `cobra.Command.Args` validator (accessible via reflection on the func value is fragile — better: maintain/derive a small `map[string]int` of max-positional-args per command, or call `cmd.Args(cmd, positionalTokens)` directly since `cobra.PositionalArgs` is literally `func(cmd *Command, args []string) error` and can be invoked with a synthetic args slice built from the parsed tokens).

### Wrapper generation chain for `/ant-unblock` (LOUD-08 / Open Question 5)

Minimal existing pattern to copy (`.aether/commands/preferences.yaml` + `.claude/commands/ant/preferences.md`, verified in full):
```yaml
# .aether/commands/preferences.yaml
name: ant-preferences
description: "🧠 Add or list user preferences in hub QUEEN.md"
source_of_truth: "Use the Go `aether` CLI as the source of truth."
runtime:
  command: "AETHER_OUTPUT_MODE=visual aether preferences $ARGUMENTS"
guardrails:
  - "Do not write colony state files, session files, or pheromone files by hand from this command spec."
  - "If docs and runtime disagree, runtime wins."
```
```markdown
<!-- Aether-managed: runtime spec at .aether/commands/preferences.yaml. Synced by aether update. -->
---
name: ant-preferences
description: "🧠 Add or list user preferences in hub QUEEN.md"
---
...
```
The generated-header format is enforced by `cmd/command_source_hygiene_test.go:14` (`TestCommandWrappersReferenceRealYamlSources`), which regexes for `^<!-- Aether-managed: runtime spec at (\.aether/commands/[^ ]+\.yaml)\. Synced by aether update\. -->$` on line 1 of every wrapper and fails if the referenced YAML file doesn't exist. For `/ant-unblock`, the runtime command line is already known from `cmd/unblock_cmd.go:12-19` (`Use: "unblock"`, flags `--phase`, `--fixer-mode`, `--dispatch`).

**Goldens that need `-update-golden` after adding the wrapper** (all three confirmed to exist with this exact flag):
- `cmd/testdata/command_catalog.json` (checked by `cmd/audit_catalog_test.go:12`)
- `cmd/testdata/parity_snapshot.json` (checked by `cmd/parity_test.go:159`)
- `cmd/testdata/regression_snapshot.json` (checked by `cmd/regression_test.go`, includes a `command_count` field currently at 398 — this counts registered Go subcommands, not slash-command wrappers, so adding `/ant-unblock`'s wrapper markdown alone should **not** change this number; only `command_catalog.json`/`parity_snapshot.json` should need regeneration unless a new Go subcommand is also added, which it is not — `unblock` already exists).

Also update: `CLAUDE.md`'s Quick Reference table ("Slash commands | 60 (Claude) + 60 (OpenCode)") to 61/61 for consistency (not enforced by a test today, but leaving it stale is the exact class of doc-drift this phase is about).

### Debug-artifact insertion points for D-03 (`pkg/codex/platform_dispatch.go`)

```go
// pkg/codex/platform_dispatch.go:1013-1018 — timeout path, NO debug write today
if ctx.Err() == context.DeadlineExceeded {
    reportedTimeout := duration.Round(time.Millisecond)
    if duration >= time.Second {
        reportedTimeout = duration.Round(time.Second)
    }
    return WorkerResult{..., Error: fmt.Errorf("worker timeout after %v", reportedTimeout)}, nil
    // ^ gap: no writeHostedWorkerOutputDebug call here
}
if waitErr != nil {
    // cmd/platform_dispatch.go:1020-1022 — non-zero exit path, NO debug write today
    return WorkerResult{..., Error: classifyHostedExecutionError(...)}, nil
    // ^ gap: no writeHostedWorkerOutputDebug call here
}
```
Compare to the two existing call sites that already do this correctly (`:1032`, `:1040`), which follow the pattern `if debugPath := writeHostedWorkerOutputDebug(config.Root, ...); debugPath != "" { err = fmt.Errorf("%w (debug: %s)", err, debugPath) }`. The fix is to add the same call in both gap branches — remembering to pass `workerTrackingRoot(config)` instead of `config.Root` per Pitfall 5, and to extend the `payload` map in `writeHostedWorkerOutputDebug` (`pkg/codex/platform_dispatch.go:1219-1253`) with `duration`, `exit_code` (derivable from `waitErr` via `exec.ExitError.ExitCode()`), and a provider-session-id field if one is available on `config` (check `config.ProviderRunID`, already used at line 978 for process tracking — reuse it).

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| TS host loads and executes `.aether/docs/command-playbooks/*.md` as literal orchestration instructions via `playbook-loader.ts` | Playbooks are unread reference documentation only; Go/wrapper markdown is the sole live execution path | Commit `5b01f361` (2026-07-25, stopped injection) then `b2b41486` (2026-07-27, deleted the loader) | Any requirement or doc referencing "the playbooks" as something the runtime executes is now describing history, not current behavior — this affects how LOUD-01/02/03/05/06 should be scoped. |
| `check-antipattern` required `--file` only, `cobra.NoArgs` | Accepts positional arg or `--file`, `cobra.MaximumNArgs(1)` | Same 2026-07-27 session (pinned by `cmd/security_gate_drift_test.go`) | The CLI contract is fixed; the *caller* problem (nobody calls it) is what remains, per Open Question 1. |

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The recommended wiring point for `check-antipattern` is the default/fast continue path (not only the heavy-review dispatch path) | Architecture Patterns, Code Examples | If the intended design is "only heavy review runs security scanning," a lighter touch (wiring only into the heavy dispatch's Gatekeeper task, and separately fixing the agent's tool access) would be a smaller, different change. This is `[ASSUMED]` — not confirmed by any design doc; it follows from the phase's own success criterion #2 language ("actually executes during continue" with no depth qualifier) plus D-01's classification of security scans as gates that must run, but the exact depth binding (light/standard/heavy) is a planning decision, not a verified fact. |
| A2 | `claims.FilesCreated`/`FilesModified`/`TestsWritten` (from `cmd/codex_continue.go:2755`) is an appropriate and sufficient file list for the new antipattern gate check | Architecture Patterns, Don't Hand-Roll | If worker claims are incomplete or a worker lies about what it touched, the gate could miss a file with a real secret. This is the same trust model already used elsewhere in continue, so the risk is not new, but it is `[ASSUMED]` that reusing it for security scanning specifically is acceptable given the higher stakes. |
| A3 | `command_count` in `cmd/testdata/regression_snapshot.json` (398) counts registered Go subcommands and will not change from adding the `/ant-unblock` *wrapper* alone | Code Examples | `[ASSUMED]` from reading the field name and the fact that `unblock` is already a registered Go subcommand today (confirmed at `cmd/unblock_cmd.go:135`, `rootCmd.AddCommand(unblockCmd)`). If the counting logic in `cmd/regression_test.go` actually counts something else (e.g. markdown wrapper files), this assumption is wrong and the golden needs a `-update-golden` run regardless — low risk either way since the recommended plan already includes a golden-update task. |

## Open Questions

1. **LOUD-05 execution audit mechanics — and whether it should cover dead files at all.**
   - What we know: `cmd/security_gate_drift_test.go` is the only execution-based (not regex-based) precedent — it spins up the real command via `rootCmd.SetArgs(...)`/`rootCmd.Execute()` against a temp fixture and asserts exit code + stderr content + parsed envelope fields (`cmd/security_gate_drift_test.go:18-65`). `cmd/cli_flag_audit_test.go` is regex-only and has the positional-arg blind spot documented in Code Examples. Neither test enumerates calls from *all* the described sources (`.claude/commands/ant/*.md`, `.opencode/commands/ant/*.md`, `.aether/commands/*.yaml`, `.aether/docs/command-playbooks/*.md`) and actually executes each one.
   - What's unclear: whether "every documented CLI call" (LOUD-05's text) is meant to include the 179-`2>/dev/null`, fully-dead `command-playbooks` corpus, or only the two live wrapper directories plus the YAML sources.
   - Recommendation: build a generalized execution-audit test that (a) reuses `cli_flag_audit_test.go`'s markdown-scanning regex as a starting point but extends it to detect the shape of positional arguments (count of non-`--` tokens after the subcommand) and cross-check against each command's `cobra.Args` validator by invoking it directly with a synthetic args slice (cheap — no subprocess, no fixtures needed for this half); (b) for a curated subset of commands that are read-only and safe to actually execute in-process against a temp store (state inspection commands like `verify-claims`, `state-checkpoint`, `skill-detect`, `print-next-up`, `generate-progress-bar`, `check-antipattern`), follow the `security_gate_drift_test.go` pattern of driving `rootCmd.Execute()` against the exact invocation form found in markdown, using a temp `newTestStore` fixture — this is what makes the audit "execution, not regex" for LOUD-05's literal text; (c) explicitly scope the corpus to `.claude/commands/ant/`, `.opencode/commands/ant/`, `.aether/commands/*.yaml`, and `.aether/docs/command-playbooks/` (for documentation-correctness only, not runtime-behavior claims), and record in the test's doc comment that the last directory is confirmed non-executing as of this phase so a future reader doesn't assume otherwise.

2. **LOUD-04 positional-argument drift — confirmed gap, concrete fix shape given above** (see Code Examples). No remaining ambiguity; this is ready to plan directly.

3. **Gate vs. enrichment classification (D-01) — where it should live.**
   - What we know: `cmd/gate.go`'s `gateClassifications` map (`hardBlock`/`softBlock`/`advisory`) already exists and already has an unpopulated `"anti_pattern"` entry at `softBlock`.
   - What's unclear: whether the *other* six audited calls (`survey-load` — moot, deleted; `print-next-up`, `verify-claims`, `state-checkpoint`, `generate-progress-bar`, `skill-detect`) need their own classification entries, given they are enrichment/context calls per D-01's own examples ("survey, skill matching, progress rendering") and none of them currently participate in the gate pipeline at all (they're standalone utility commands, not gate checks).
   - Recommendation: only `check-antipattern`/`anti_pattern` needs a gate-tier classification, because it is the only one of the seven that is genuinely safety-relevant per D-01's own criteria (security scan). The other five are enrichment calls whose "loud failure" requirement is satisfied by fixing their dead-doc call sites (so the audit passes) plus, if any of them are ever reconnected to a live path by a future phase, a warning-and-continue pattern rather than a gate. Do not build new gate-classification entries for `print-next-up`/`verify-claims`/`state-checkpoint`/`generate-progress-bar`/`skill-detect` in this phase — there is no live call site for them to gate.

4. **RETIRE-01/04 deletion safety — ledger format and schema-test replacement.**
   - What we know: `control-ts/` has zero Go/CI/embed/goreleaser references (verified). `control-ts/tests/schemas/policy.schema.test.ts` validates 7 policy YAMLs (`model-routing.yaml`, `memory-rules.yaml`, `skill-creation.yaml`, `safety-gates.yaml`, `dispatch-contract.yaml`, `pheromone-lifecycle.yaml`, and more per the file — confirmed by reading the first ~60 lines) against a Zod schema, reading from `control-ts/tests/fixtures/policies/*.yaml` (fixture copies, not the live `colony/policies/*.yaml` — worth double-checking these fixtures match the real files before deletion, since a passing test today could already be testing stale fixtures). `cmd/policy_loader.go` has a generic `loadYAMLPolicy(path string, out interface{})` but no typed struct or schema assertions for any policy file today (zero non-test references to `model-routing` in `cmd/`).
   - What's unclear: exact ledger file location/format (explicitly Claude's discretion per CONTEXT.md).
   - Recommendation: (a) ledger as a simple markdown file, e.g. `.aether/docs/retired-tests-ledger.md`, following the existing convention of `.aether/docs/known-issues.md` — one entry per deleted test file with columns for original path, what it covered, disposition (`dead-with-no-replacement` | `recovered-by:<test>`), and the commit that removed it. Retroactively record `.aether/ts-host/test/playbook-loader.test.ts` (deleted in `b2b41486`) as `dead-with-no-replacement` (the feature it tested, playbook loading/injection, was deliberately removed, not relocated). (b) For `policy.schema.test.ts`'s replacement, write a narrow Go test in `cmd/` (e.g. `cmd/policy_schema_test.go`) that reads each file under `colony/policies/*.yaml`, unmarshals into `map[string]interface{}`, and asserts presence + type of the same fields the vitest file currently checks (`model_routing.default_provider`, `memory_rules.max_learnings`, `skill_creation.allowed`, `safety_gates.security_scan`, etc.) — this is deliberately narrower than a full typed loader (that's Phase 161/MODEL-01's job) but satisfies RETIRE-04's "replacement must exist before or alongside deletion" bar.

5. **LOUD-08 `/ant-unblock` wrapper — mechanics confirmed, see Code Examples.** One residual question: Codex has no wrapper markdown per CLAUDE.md's Platform Policy ("Codex UX | Go runtime only — No wrapper markdown"). Since `cmd/unblock_cmd.go` already exists and works as a plain CLI command, Codex users can already run `aether unblock` directly — confirm (not verify further; out of this research's grep-scope) that no Codex-side guidance text needs the same fix, by checking `.codex/CODEX.md` for any stale `/ant-unblock` reference during planning.

6. **Debug-artifact trio (D-03/D-04/D-05) — mechanics fully grounded, see Code Examples and Don't Hand-Roll.** No open ambiguity remaining; ready to plan directly against the cited line numbers.

7. **LOUD-06 residual stderr suppression — see Reconciliation Corrections.** Recommendation: fix the 4 confirmed live sites (pin with a test asserting these specific `2>/dev/null` occurrences are the *only* ones in `.claude/commands/ant/` + `.opencode/commands/ant/`, so a proportion/invariant test catches regressions per this project's Definition of Done — e.g. "grep count of `2>/dev/null` in these two directories equals 4" rather than a hardcoded list of files, so any *new* suppression trips the test too). Do not attempt the 179 dead-doc sites under this requirement.

8. **LOUD-07 doc claims — see Reconciliation Corrections for exact current state of all three files plus the bonus finding below.**
   - Bonus, not in requirement scope but same failure class: `.aether/docs/PARITY_CLASSIC_VS_GO.md:59` claims `check-antipattern` "runs after verification" with status `MATCH` and a golden-test citation — this is the same kind of unverified behavioral claim LOUD-07 targets, just not in one of the three named files. Recommend flagging it to whoever owns that doc (not necessarily fixing it in this phase, since it's outside the three files named in the requirement) or folding a one-line correction into the same commit that fixes CLAUDE.md:838, at the planner's discretion.

## Environment Availability

Skipped — this phase is code/config/deletion changes within the existing repo, with no new external tool, service, or runtime dependency. All verification commands listed in the phase brief (`go test ./...`, `go build ./cmd/aether`, `go vet ./...`, `aether version --check`, `aether integrity`, `goreleaser check`) already run successfully in this environment per the repo's existing CI/test conventions; none are new to this phase.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` package (stdlib), invoked via `go test` |
| Config file | none — standard `go test ./...` |
| Quick run command | `go test ./cmd/... ./pkg/... -run 'TestCheckAntipattern\|TestCLIFlagAudit\|TestGatekeeperPlaybooksUsePositionalForm' -count=1` |
| Full suite command | `go test ./... -count=1` (add `-race` for the pre-merge pass per CLAUDE.md's Verification Commands) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| LOUD-01 | `survey-load` absent, no stale callers | unit/static | `go test ./cmd -run TestSurveyLoadAbsentAndUncalled -count=1` | ❌ Wave 0 (new test asserting: no `survey-load` Go command registered, and zero references to the string `survey-load` outside `control-ts/` and this new test file) |
| LOUD-02 | `check-antipattern` gates a live continue run | integration | `go test ./cmd -run TestContinueAntiPatternGateBlocksOnCriticalFinding -count=1` | ❌ Wave 0 — must drive a real continue-style gate evaluation (not just the CLI command) with a fixture file containing a hardcoded secret and assert the gate result is `failed`/hard-or-soft-blocked per the chosen classification |
| LOUD-03 | Five remaining calls correct in their (now-fixed) call sites | static/doc | `go test ./cmd -run TestCommandPlaybookCallsMatchCobraContracts -count=1` | ❌ Wave 0 — the generalized audit from Open Question 1 |
| LOUD-04 | Drift test catches positional-arg drift | unit (self-test) | `go test ./cmd -run TestCLIFlagAuditDetectsPositionalDrift -count=1` | ❌ Wave 0 — a test that intentionally feeds the audit a known-bad positional call and asserts it fails |
| LOUD-05 | Execution audit covers every documented call | integration | `go test ./cmd -run TestCommandPlaybookCallsMatchCobraContracts -count=1` (same test as LOUD-03; one audit, multiple requirements) | ❌ Wave 0 |
| LOUD-06 | No load-bearing stderr suppression in live wrappers | invariant | `go test ./cmd -run TestLiveWrapperStderrSuppressionCount -count=1` | ❌ Wave 0 — proportion/count invariant test (exactly 4 `2>/dev/null` sites in the two live wrapper dirs), per Open Question 7 |
| LOUD-07 | Docs don't claim live consolidation | static/doc | `go test ./cmd -run TestDocsDoNotClaimConsolidationRunsToday -count=1` | ❌ Wave 0 — grep-based test over `CLAUDE.md`, `AGENTS.md`, `.aether/docs/structural-learning-stack.md` for present-tense consolidation-runs claims |
| LOUD-08 | `/ant-unblock` wrapper exists and matches Go command | existing pattern | `go test ./cmd -run TestCommandWrappersReferenceRealYamlSources -count=1` (already exists, will now also cover `unblock.yaml`) plus `go test ./cmd -run TestClaudeOpenCodeCommandParity -count=1` if that test exists (verify name during planning) | ✅ harness exists (`cmd/command_source_hygiene_test.go:14`), ❌ fixture (the yaml/md files themselves) |
| RETIRE-01 | `control-ts/` deleted, build/publish/integrity still succeed | e2e | `go build ./cmd/aether && aether integrity` | ✅ commands exist, run post-deletion |
| RETIRE-02/03 | `.aether/ts-host/`, `.aether/ts/` unchanged | invariant | `git diff --stat -- .aether/ts-host .aether/ts` returns empty after the control-ts deletion commit | ✅ git itself is the check |
| RETIRE-04 | Deleted tests recorded in ledger; schema replacement exists | static/doc + unit | `go test ./cmd -run TestPolicySchemaValidation -count=1` (new) + manual review of `.aether/docs/retired-tests-ledger.md` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** the quick run command above, scoped to the specific requirement's new test(s).
- **Per wave merge:** `go test ./cmd/... ./pkg/... -count=1`.
- **Phase gate:** `go test ./... -count=1 -race`, `go build ./cmd/aether`, `go vet ./...`, `aether integrity` all green before `/gsd-verify-work`.

### Wave 0 Gaps
- [ ] `cmd/policy_schema_test.go` — RETIRE-04 replacement for `control-ts/tests/schemas/policy.schema.test.ts`
- [ ] `cmd/command_playbook_execution_audit_test.go` (name at planner's discretion) — the generalized LOUD-03/04/05 audit from Open Question 1
- [ ] `cmd/live_wrapper_stderr_test.go` (or folded into an existing file) — LOUD-06's invariant test
- [ ] `cmd/doc_consolidation_claims_test.go` (or folded) — LOUD-07's static-doc test
- [ ] `cmd/continue_antipattern_gate_test.go` — LOUD-02's live-wiring integration test
- [ ] `.aether/commands/unblock.yaml`, `.claude/commands/ant/unblock.md`, `.opencode/commands/ant/unblock.md` — LOUD-08 fixtures, not tests, but required before `TestCommandWrappersReferenceRealYamlSources` can cover them meaningfully
- [ ] `.aether/docs/retired-tests-ledger.md` — RETIRE-04 ledger, new file

*(No existing test infrastructure gap for RETIRE-01/02/03 — `go build`, `aether integrity`, and `git diff` are sufficient and already available.)*

## Security Domain

`security_enforcement` is not set to `false` in `.planning/config.json` (absent = enabled), so this section is required. This phase is unusually security-relevant for a "fix broken CLI calls" phase because its central open question (Open Question 1) is precisely a currently-non-functioning secret-scanning gate.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V5 Input Validation | yes | Cobra `Args` validators (`cobra.NoArgs`, `cobra.MaximumNArgs`) already enforce shape on every audited command; the fix is closing the *test* gap (LOUD-04), not adding new validation. |
| V6 Cryptography | no | Not applicable — no cryptographic operations touched by this phase. |
| Secrets/credential exposure (not a numbered ASVS category here, but the direct subject of `check-antipattern`) | yes | `cmd/security_cmds.go:169` regex-based scan for `api_key`/`secret`/`password`/`token`/`access_key` patterns in changed files — explicitly documented in `CLAUDE.md:653` as "~6 patterns — not a full security scanner." This phase's job is to make sure this *existing, limited* scanner actually runs, not to expand its coverage. |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Hardcoded credential committed to a worker-modified file, never scanned because the scanning gate has no live caller | Information Disclosure | Wire `check-antipattern` (or its extracted shared function) into the live continue gate pipeline per Open Question 1 — this *is* the mitigation this phase should deliver. |
| Worker debug artifact (`.aether/data/worker-debug/*.json`) capturing raw provider stdout/stderr including a leaked credential from a failing worker | Information Disclosure | `safeHostedWorkerArgs` (`pkg/codex/platform_dispatch.go:1262`) already redacts prompt/schema/system-prompt argument *values* by flag identity; `sanitizeWorkerDiagnosticOutput` (referenced at `pkg/codex/platform_dispatch.go:1012`, `:1243`) already sanitizes stdout/stderr excerpts before they're written. D-03's new timeout/non-zero-exit debug writes must reuse these same sanitization calls (they already do, since the fix is adding *calls* to the existing `writeHostedWorkerOutputDebug` function, which internally calls `sanitizeWorkerDiagnosticOutput` and `workerOutputExcerpt` on every write) — no new sanitization logic needed, just don't bypass the existing function. |

## Sources

### Primary (HIGH confidence — direct code/file verification this session)
- `cmd/security_cmds.go`, `cmd/security_gate_drift_test.go`, `cmd/cli_flag_audit_test.go` — CLI contract and existing drift-test behavior
- `cmd/gate.go` (full classification map, gateCheck pattern, pre/post-continue gate functions)
- `cmd/unblock_cmd.go`, `cmd/command_source_hygiene_test.go`, `.aether/commands/preferences.yaml`, `.claude/commands/ant/preferences.md` — wrapper generation chain
- `pkg/codex/platform_dispatch.go`, `pkg/codex/worker.go`, `cmd/codex_build_worktree.go` — debug-artifact and worktree-root resolution
- `.claude/commands/ant/build.md`, `.claude/commands/ant/continue.md` — confirmed live wrapper content, confirmed no reference to command-playbooks files
- `.claude/agents/ant/aether-gatekeeper.md` — confirmed no Bash tool, confirmed dependency/license-only scope
- `/embedded_assets.go` — confirmed embed directives and correct path (not `cmd/embedded_assets.go`)
- `control-ts/package.json`, `control-ts/tests/schemas/policy.schema.test.ts`, `cmd/policy_loader.go`, `colony/policies/model-routing.yaml` — RETIRE-01/04 grounding
- `AGENTS.md:899`, `.aether/docs/structural-learning-stack.md:14,207,227`, `CLAUDE.md:37,838` — LOUD-07 doc-claim verification
- `git log --oneline -20`, `git show b70cd103/5b01f361/b2b41486 --stat` — repo history confirming the playbook-loader deletion timeline and the "v1.25" milestone-numbering collision (see note below)
- `.planning/phases/160-fail-loudly/160-CONTEXT.md`, `.planning/REQUIREMENTS.md`, `.planning/ROADMAP.md`, `.planning/STATE.md`, `CLAUDE.md`

### Secondary (MEDIUM confidence)
- `.aether/docs/PARITY_CLASSIC_VS_GO.md:59` — read and cross-checked against code, found to be itself an unverified/stale claim (see Open Question 8 bonus finding); treated as evidence of the pattern, not as a trustworthy source in itself.

### Tertiary (LOW confidence)
- None — no claim in this document rests solely on an unverified web search or training-data recall; every factual claim about the current repo state was checked directly.

## Metadata

**Confidence breakdown:**
- Live-vs-dead execution surface mapping: HIGH — verified via three independent methods (CLAUDE.md's own doc claim, full read of the live wrappers, git archaeology of the playbook-loader deletion).
- Gate-wiring recommendation (Open Question 1): HIGH on the *diagnosis* (zero live callers, confirmed exhaustively by grep), MEDIUM on the *exact* recommended insertion point (default vs. heavy-review path is a design choice, flagged as `[ASSUMED]` A1).
- RETIRE-01/02/03 safety: HIGH — exhaustive grep across Go, CI, embeds, goreleaser config.
- RETIRE-04 replacement-test scope: MEDIUM — the fixture files (`control-ts/tests/fixtures/policies/*.yaml`) were not diffed against the live `colony/policies/*.yaml` for drift; recommend the planner add a quick diff check as a task precondition.
- LOUD-08 mechanics: HIGH — direct pattern match against an existing, working wrapper.

**Research date:** 2026-07-27
**Valid until:** ~7 days (fast-moving area — this exact repo shipped three "close the loops" commits in the 48 hours preceding this research, including one that deleted the playbook loader this research depends on being gone; re-verify live-vs-dead status before executing if planning is delayed).

**Note on repo history:** commit `b70cd103` ("docs: session handoff — v1.25 complete, reviewed, and published", 2026-07-26) refers to a *different, earlier* body of work than the "v1.25 Switch It On" milestone this phase belongs to (per `.planning/STATE.md`, that milestone is still unapproved as of this research). The milestone-number label "v1.25" appears to have been reused across two different planning cycles in this repo's history. This is not a blocker for Phase 160, but a planner or future researcher should not assume "v1.25" uniquely identifies one body of work when grepping commit history.
