# Raw report 06 — GSD (Get Shit Done) Architecture & Reliability Analysis

*Verbatim output of the "Analyze GSD dependability properties" investigation, 2026-08-17.*

**Where it lives:** Thin skill routers in `/Users/callumcowie/.claude/skills/gsd-*/SKILL.md` (~1-3KB each) that `@`-reference the real logic in `/Users/callumcowie/.claude/get-shit-done/` — `workflows/` (93 files), `templates/` (39), `references/` (62), plus agent definitions in `/Users/callumcowie/.claude/agents/gsd-*.md` (~35 agents). One correction to the task premise: GSD is **not** purely markdown + conventions. It ships a deterministic Node.js CLI (`gsd-sdk`, delegating to `sdk/dist/cli.js`; also `bin/gsd-tools.cjs`) that owns state mutation, frontmatter validation, plan-structure checks, artifact/key-link verification, and commits. However, the LLM must *choose* to call it — there is no hook forcing the call, which is the system's real enforcement boundary.

## 1. Thin orchestrators

- `skills/gsd-execute-phase/SKILL.md`: "Orchestrator stays lean: discover plans, analyze dependencies, group into waves, spawn subagents, collect results... Context budget: ~15% orchestrator, 100% fresh per subagent."
- `workflows/execute-phase.md` `<core_principle>`: "Orchestrator coordinates, not executes." `<context_efficiency>`: "Orchestrator: ~10-15% context... Subagents: fresh context each... No context bleed."
- Spawns pass **paths, not content**: "Pass paths only — executors read files themselves with their fresh context window."
- One-shot init call collapses dozens of reads into one: `INIT=$(gsd-sdk query init.execute-phase ...)`.
- Checkpoint continuation uses **fresh agents, not resume**: "Resume relies on internal serialization that breaks with parallel tool calls. Fresh agents with explicit state are more reliable."
- Why reliable: each executor starts at 0% context so quality never degrades from accumulated orchestration chatter.

## 2. Bounded context — PLAN.md as a self-contained prompt

- "Plans are prompts, not documents that become prompts" (`agents/gsd-planner.md`). A PLAN.md carries frontmatter (`phase, plan, type, wave, depends_on, files_modified, autonomous, requirements, must_haves{truths, artifacts, key_links}`), an `<objective>`, `<context>` with `@`-file references, 2-3 `<task>` elements each with mandatory `<files>/<action>/<verify>/<done>`, a `<threat_model>`, `<verification>`, and `<success_criteria>`.
- Task anatomy: files must be "exact file paths" ("Bad: 'the auth files'"), verify must be "a specific automated command that runs in < 60 seconds", done must be measurable. Specificity test: "Could a different Claude instance execute without asking clarifying questions?"
- Context-budget math explicit: quality degradation curve (0-30% PEAK ... 70%+ POOR), "Plans should complete within ~50% context", tasks sized at 10-30% context, sizing in files-touched, never time.
- **Interfaces embedded in the plan** so executors don't explore: an `<interfaces>` block with extracted type signatures — "blueprints versus 'build me a house'." Interface-first task ordering prevents "the scavenger-hunt anti-pattern".
- Real example: `.planning/phases/174-spend-ledger/174-01-PLAN.md` — must_haves truths like "Anthropic's documented 50/100,000/2,000/500 example yields total input 102,050", artifacts with `contains:` patterns, key_links with regex patterns.

## 3. Planning discipline — research → plan → checker loop

- `/gsd-plan-phase` orchestrates `gsd-phase-researcher` (→ RESEARCH.md) → `gsd-planner` (→ PLAN.md files) → `gsd-plan-checker` in a bounded revision loop: max 3 iterations, stall detection, then an Escalation Gate to the human.
- The checker (`agents/gsd-plan-checker.md`, 12+ dimensions): requirement coverage (missing = blocking), task completeness (Files/Action/Verify/Done), dependency correctness, key-links planned, scope sanity (5+ tasks/plan = blocker), must-have derivation (user-observable, not "bcrypt installed"), context compliance (locked user decisions D-XX each need an implementing task; deferred ideas must be absent), **scope-reduction detection** (scans for "v1", "simplified", "static for now", "stub" — always a BLOCKER), architectural tier compliance, Nyquist compliance, cross-plan data contracts, CLAUDE.md compliance, research-resolution.
- Prompted adversarially: "Assume every plan set is flawed until evidence proves otherwise."
- Structure checks are deterministic: `gsd-sdk query verify.plan-structure` and `frontmatter.validate`.

## 4. Executable plans & atomic progress

- **Per-task commits**, staged file-by-file ("NEVER `git add .`"), conventional format `{type}({phase}-{plan})`, hash recorded per task, post-commit deletion and untracked-file checks.
- **Deviation rules**: Rules 1-3 (fix bugs, add missing critical functionality, unblock) apply automatically; Rule 4 (architectural change) forces STOP-and-ask. Out-of-scope goes to `deferred-items.md`. Hard cap: 3 auto-fix attempts per task. Analysis-paralysis guard after 5 consecutive reads with no write.
- **Checkpoint protocol**: `human-verify` (90%), `decision` (9%), `human-action` (1%). Automation-first: "Users NEVER run CLI commands." Auth errors are gates, not failures. On checkpoint the agent STOPs and returns a structured table; a *fresh* continuation agent is spawned with that state.
- **SUMMARY.md**: dependency graph frontmatter, substantive one-liner required, deviations documented per-rule with commit hashes, stub tracking, threat-surface flags.
- **Self-check before claiming done**: verify claimed files exist and commit hashes exist, append `## Self-Check: PASSED/FAILED`. The orchestrator independently spot-checks the same claims.

## 5. State — STATE.md and deterministic resume

- `.planning/STATE.md` is the session digest: position, progress bar, velocity, decisions, blockers, Session Continuity. "Keep STATE.md under 100 lines. It's a DIGEST, not an archive."
- Updates via SDK verbs (`state.advance-plan`, `state.update-progress` recalculating from SUMMARYs **on disk**, `state.record-session`), not freehand edits.
- **Pause**: machine-readable `HANDOFF.json` + human `.continue-here.md` with a blocking/advisory anti-patterns table, committed as WIP.
- **Resume**: priority chain HANDOFF.json → `.continue-here.md` → *derived* incompleteness (PLAN without SUMMARY) → interrupted-agent record. STATE.md reconstructible from PROJECT.md + ROADMAP.md + SUMMARYs.
- **Determinism source: the filesystem, not STATE.md.** A `safe_resume_gate` cross-checks git: plan-tagged commits with missing SUMMARY → stop and offer close-out/re-execute/mark-and-skip.

## 6. Verification / UAT — two independent layers

- **gsd-verifier** (goal-backward): "Do NOT trust SUMMARY.md claims." Four artifact levels: exists → substantive → wired (ORPHANED otherwise) → data flows (catches HOLLOW components). Key-link regex verification; anti-pattern scan; behavioral spot-checks; probe execution ("SUMMARY.md probe pass claims are not evidence"). Any FAILED truth → `gaps_found`; any human-only item → `human_needed`. Gaps emit structured YAML that `--gaps` planning parses into gap-closure plans, then re-verifies. Escape hatch: `overrides:` with reason/accepted_by, counted as "PASSED (override)".
- **gsd-verify-work** (human UAT): "Show expected, ask if reality matches", one test at a time, persisted `{phase}-UAT.md` survives `/clear`, auto-injected Cold Start Smoke Test when server/db files touched.

## 7. Completion — layered "done"

Task: `<done>` + `<verify>` + commit. Plan: SUMMARY + Self-Check + orchestrator spot-check + tests-gated tracking updates. Phase: post-merge build/test gate, regression gate, schema-drift gate, code-review gate, verifier `passed`, then `phase.complete` flips ROADMAP/STATE/REQUIREMENTS and scans for verification debt. Milestone: `/gsd-audit-milestone`, `/gsd-audit-uat`.

## 8. Golden path & decision load

`/gsd-new-project` → per phase: discuss (optional) → plan (research→plan→check) → execute (waves→gates→verifier) → verify-work → auto-transition → complete-milestone. Every workflow ends by printing the exact next command. A cautious user makes ~5-10 decisions per phase; auto-mode ~1.

## 9. Failure / interruption

Per-task commits mean completed work survives; re-running skips SUMMARYed plans; safe_resume_gate catches half-done; stall surveillance polls git. Worktree hazards carry issue-numbered scar tissue (#2075 git clean prohibition, #2924 HEAD guards, #3097 cwd drift, #2070 SUMMARY rescue, #2384 bulk-deletion revert, #1756 orchestrator-file restore). Recovery unit is one task; truth re-derived from git/filesystem.

## 10. What GSD does NOT guarantee

Three enforcement tiers: (1) deterministic when invoked (gsd-sdk — but nothing forces the call; hooks exist in design, switched off); (2) cross-checked convention (every checker is an LLM; prompts saturated with issue-numbered scar tissue because discipline failed once each time); (3) pure prompt discipline (deviation-rule judgment, adversarial stance, keyword scope-detection, key-link regex presence ≠ execution, optional behavioral checks, UAT depends on the human, auto-mode silently converts gates to first-option-wins).

Specific non-guarantees: no atomic transaction across STATE/ROADMAP/REQUIREMENTS (drift acknowledged); nothing verifies the verifier; "Keep verification fast. Use grep/file checks, not running the app" for most truths; wave parallelism rests on honestly-declared `files_modified`; the 3-iteration loop can exhaust; ~1,800-line orchestrator workflow files face the very context problem GSD solves for subagents.

**Bottom line:** GSD's dependability = (a) fresh-context isolation per unit of work, (b) self-sufficient plans with embedded verification, (c) every claim has an independent adversarial re-checker, (d) all progress derivable from disk+git. What it lacks — and a Go runtime can provide — is *involuntary* enforcement: GSD's gates fire only if the LLM reads the paragraph telling it to fire them.
