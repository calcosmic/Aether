# Phase 207: A Messy Practice Project Is the Release Gate - Research

**Researched:** 2026-09-22
**Domain:** Headless Claude Code chat automation (`claude -p`), Go CLI journey/eval-gate harnesses, filesystem trap construction (symlinks, case collisions, nested repos), flake classification
**Confidence:** MEDIUM-HIGH — every load-bearing claim below was verified against this machine's real `claude --help`/`claude --version`, the official headless docs (fetched live), or this repository's own source (file + line range, quoted). The one LOW-confidence item (the sixth 2026-09-21 blocker) is flagged explicitly because it changes what UED-09 can prove inside this phase.

<user_constraints>
## User Constraints (from CONTEXT.md)

207-CONTEXT.md carries no `## Decisions` / `## Claude's Discretion` / `## Deferred Ideas` sections — it is a pointer file only. Its full content, verbatim:

> **Requirements:** UED-07 to UED-09. **Decision:** `.planning/decisions/2026-09-21-v1.29-use-it-every-day.md`. **Research:** `.planning/research/2026-09-21-popular-frameworks.md`, `.planning/research/2026-09-21-reliability-and-delivery.md`.
>
> Not yet planned. Keep it small; the proof is a real run in a real chat; one review round.

Binding constraints pulled from the milestone decision record and REQUIREMENTS.md (these function as locked decisions for this phase — see `.planning/decisions/2026-09-21-v1.29-use-it-every-day.md` and `.planning/REQUIREMENTS.md`):

- **One rule for the whole milestone: no new features and no new strict rules.** A committed script + a test/gate that can fail is not "a feature" in the product sense, but the planner must not add a new *hard runtime refusal* to the `aether` binary as part of this phase — the gate belongs at the release-process layer (Makefile / publish runbook / CI), not as a new `aether` subcommand that blocks something it didn't block before.
- Each v1.29 phase stays small; its proof is **a real run in a real chat**, never paperwork; **one review round**.
- Assertions are on **files produced and commands the chat ran, never on wording**.
- **Three trials**; failures sorted into **flaky and real**.
- **Money and turn caps** on every `claude -p` invocation.
- **Hooks and menu commands must be loaded** — i.e. the journey must NOT pass `--bare` (see Q1 below; `--bare` explicitly skips hooks, custom commands, subagents and CLAUDE.md, and the docs say it "will become the default for -p in a future release" — the plan must pin the non-bare path explicitly so a future default flip does not silently hollow out the gate).
- Proof required: **the journey catches each of 2026-09-21's six blockers when its fix is removed** (UED-09).
- The journey becomes **the gate for every release from here on** (ROADMAP.md Phase 207 success criterion 4) — this phase must decide *where* that gate plugs in, since none exists today (see Q2 below).

Out of scope for this phase (from REQUIREMENTS.md "Out of scope" and the milestone rule): any new Aether feature, any new strict rule, Codex native work, and republishing to npm/GitHub Release before Phase 210.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| UED-07 | A committed script builds the practice project with every trap listed in the roadmap. Proof: the script runs clean twice and a test asserts each trap exists. | Q3/Q4 below: each of the 9 named traps mapped to concrete filesystem/git mechanics and, where one exists, to the exact Aether code path it exercises. `scripts/proof-screens-reach-the-owner.sh` (Phase 206) is the closest committed precedent for "build a scratch project, then assert" — reuse its isolation pattern (`AETHER_HUB_DIR`, scratch `$WORK`, `trap ... EXIT`), not its content. |
| UED-08 | The whole journey runs through a real chat with hooks and menu commands loaded, asserting on files and recorded commands, never wording; three trials; money and turn caps. | Q1: exact `claude` flags verified on this machine (`claude --version` 2.1.278) and against the official headless docs (fetched live 2026-09-22). Q5: which machine-readable surfaces to assert on. Q6: trial/flake design (no existing precedent in this repo; cline's pass@k / pass^k pattern from `.planning/research/2026-09-21-reliability-and-delivery.md` is the nearest external precedent). |
| UED-09 | The journey catches each of 2026-09-21's six blockers when its fix is removed. | Q4: five of the six blockers are mapped to an exact commit, file, and function, each independently `git show`n this session. **The sixth has no landed fix as of this research — see the Open Question flagged HIGH-RISK below; this changes what UED-09 can honestly prove inside Phase 207.** |
</phase_requirements>

## Summary

Phase 206 already built and committed a working, real-`claude -p` proof harness — `scripts/proof-screens-reach-the-owner.sh` — that does almost exactly what UED-08 asks for at small scale: builds `aether` from source, installs it into an **isolated hub** (`AETHER_HUB_DIR`, `HOME` left alone because Claude Code's own login lives there), runs `aether init`/`aether update --force` in a scratch git repo, then drives one real `claude -p` turn with `--output-format stream-json --verbose --allowedTools Bash --settings <path> --max-turns 6` (+ `--max-budget-usd 1.00` when the installed `claude` supports it, detected via `claude --help`), retries once on a transient failure (rate-limit/overloaded/timeout), and asserts on the JSONL stream's `system`/`informational` messages and Bash `tool_use` blocks — never on assistant prose. **This is the template to extend, not reinvent.** The extension for Phase 207 is: (1) a *much messier* scratch project (9 named traps) built by a **separate, committed builder script**, satisfying UED-07 on its own; (2) a **longer, multi-turn chat** (init → colonize → discuss → spec → plan ×2 → build → continue → pause → resume → seal → entomb → init again) driven via `--continue`/`--resume <session_id>` across several `claude -p` invocations rather than one; (3) **three trials** with transient-vs-real failure classification (no existing precedent in this repo — model it on cline's `evals/analysis/patterns/cline-failures.yaml` regex classifier, already researched and cited in `.planning/research/2026-09-21-reliability-and-delivery.md`); (4) a **revert-and-rerun loop** over the five (see below) landed 2026-09-21 fixes.

Official Claude Code docs (fetched live today) confirm the load-bearing mechanics: `-p` loads hooks/custom-commands/CLAUDE.md **by default** — only `--bare` skips them, and `--bare` is explicitly *not* the default yet, so the journey simply omits `--bare` (no special flag needed to keep hooks/commands loaded). A typed slash command (e.g. `/ant-init "goal"`) **is expanded before running** when included literally in the prompt string, in both interactive and `-p` sessions — this is independently confirmed by reading a real captured transcript fixture in this repo (`cmd/testdata/stop-hook/menu-command-transcript.jsonl`, captured from an actual `/ant-status` session): the first `user` message's content is literally `<command-message>ant-status</command-message>\n<command-name>/ant-status</command-name>`, followed by a second `isMeta:true` message inlining the wrapper markdown file's full body. Multi-turn chaining works via `claude -p "next" --continue` (most recent conversation) or `--resume "$session_id"` (captured from an earlier call's `--output-format json | jq -r '.session_id'`), and `--resume` finds the session by ID across the whole machine, not just the invoking directory (since Claude Code v2.1.223) — so the journey's later steps can run from a fresh subshell without carrying forward any state but the session ID.

The **release gate itself does not exist yet**. `smoke-daily-driver.sh` and `proof-screens-reach-the-owner.sh` are both real, working, committed harnesses, but neither is called from `Makefile`'s default targets used by CI, nor from `.github/workflows/ci.yml` or `release.yml` (verified: `grep -n "smoke\|proof-screens\|journey" cmd/publish_cmd.go .github/workflows/*.yml Makefile` finds only the unrelated GoReleaser "Binary smoke test" step). Neither CI workflow carries an `ANTHROPIC`/`CLAUDE` credential (verified: no matches in either workflow file), so a `claude -p`-driven journey **cannot run unattended in GitHub Actions today** without adding a secret — out of scope for a "keep it small, no new strict rules" phase. The natural, already-precedented home for this gate is the existing `cmd/testdata/eval-gates/gates.json` + `Makefile` `eval-gate-*` family (seven named gates today: fast/focused/integration/provider/overnight/race/release), which already models exactly this shape of problem — the `provider` gate is `"requires": "provider_credentials"` and is "never run by default" for lack of them. A new `journey` gate following that same schema (`requires: "claude_cli_credentials"` or reused `"provider_credentials"`, `budget_seconds` sized for 3 trials of a ~12-command chat) is the most consistent design, referenced from the `release` gate's own purpose text and from `.aether/docs/publish-update-runbook.md`'s existing "Preflight" checklist (which already has exactly this shape: a manual step the owner/Claude runs and confirms before `aether publish`, e.g. "`git status --porcelain` must be empty"). This keeps the milestone's "no new strict rules" promise: nothing in the `aether` binary starts refusing anything new; a new pre-publish *procedural* step is added instead.

**Primary recommendation:** Extend `proof-screens-reach-the-owner.sh`'s isolation pattern into two new committed artifacts — a builder script (UED-07, standalone, testable by running it twice) and a `//go:build journey`-tagged Go test (skipped unless `AETHER_JOURNEY=1`, matching the phase description's own Q6 suggestion and this repo's existing build-tag-gated eval-gate convention) that shells out to `claude` across the full lifecycle with `--continue`/`--resume`, asserts on JSONL transcript tool-calls and on-disk state (never prose), runs 3 trials with a transient-failure retry-once policy, and is registered as a new `journey` eval-gate referenced by the publish runbook as the pre-release check.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Practice-project construction (9 traps) | CLI / local filesystem+git script | — | Pure local filesystem/git state construction; no network, no Aether runtime involved in *building* the fixture |
| Driving the real chat | External process (`claude` CLI) | Go test / shell harness (orchestrator) | The chat itself is Claude Code's own process; Aether's harness only launches it, captures its JSONL stream, and asserts |
| Hook firing / menu-command expansion | Claude Code runtime (hooks.md, headless.md) | `aether hook-*` subcommands (already built, Phase 206) | Aether does not control whether hooks fire — Claude Code does, based on `.claude/settings.json` in the scratch project (installed by the real `aether update --force`, per the existing proof script's own gate 1) |
| Assertion surfaces (files, JSONL tool calls) | Go test / shell harness | `.aether/data/COLONY_STATE.json`, `aether history --json`, `AETHER_OUTPUT_MODE=json` outputs | Machine-readable, already-existing Aether surfaces (Q5) plus the Claude Code transcript are the two evidence sources; nothing new needs to be built to produce them |
| Trial/flake classification, budget/turn caps | New harness code (Go test or shell) | — | No existing Aether mechanism does this; net-new but small, modeled on cline's external pattern |
| Release-gate wiring | `Makefile` + `cmd/testdata/eval-gates/gates.json` + `.aether/docs/publish-update-runbook.md` | — | Process/documentation layer, not the `aether` binary — keeps the "no new strict rules" promise |

## Q1 — Driving a real Claude Code chat headlessly

**Machine facts (verified today):** `claude --version` → `2.1.278 (Claude Code)`. `go version` → `go1.26.5 darwin/arm64` (repo's `go.mod` also pins `go 1.26.5`). `git`, `jq`, `node`, `npm` all present.

**`-p`/`--print` loads hooks and commands by default** `[CITED: code.claude.com/docs/en/headless, fetched 2026-09-22]`: "Without [`--bare`], a `-p` session runs the hooks in a project's `.claude/settings.json`... Bare mode is useful for CI and scripts where you need the same result on every machine... `--bare` is the recommended mode for scripted and SDK calls, and will become the default for `-p` in a future release." The journey **must never pass `--bare`** and should say so in a comment, exactly as `proof-screens-reach-the-owner.sh` already does implicitly by omitting it — the plan should make this an explicit, named non-goal so a future default flip is caught rather than silently hollowing out the gate. `--setting-sources <user,project,local>` (comma-separated) exists as a belt-and-braces pin if the planner wants to be extra explicit that project settings load; `proof-screens-reach-the-owner.sh` instead pins settings directly via `--settings <path-to-scratch-project's-.claude/settings.json>`, which is the more precise choice for an isolated scratch repo (it does not depend on `cwd` resolution).

**Slash commands expand in `-p` mode** `[CITED: code.claude.com/docs/en/headless]`: "User-invoked skills and custom commands work. Include `/skill-name` in the prompt string and Claude Code expands it before running." `[VERIFIED: cmd/testdata/stop-hook/menu-command-transcript.jsonl]` (Phase 206's own captured fixture, read this session) — the first line of a real `/ant-status` session's transcript is:

```
{"parentUuid":"1bd2719f-...","type":"user","message":{"role":"user","content":"<command-message>ant-status</command-message>\n<command-name>/ant-status</command-name>"},...}
```

followed immediately by a second `user` message with `"isMeta":true,"turnCompanion":true` whose `content` is the *entire* `.claude/commands/ant/status.yaml`-derived wrapper markdown, inlined verbatim (confirmed by reading the file: it begins `<!-- Aether-managed: runtime spec at .aether/commands/status.yaml. Synced by aether update. -->`). **This is exactly the detection signal the journey should assert on**: grep the transcript for `<command-name>/ant-init</command-name>` (etc.) to prove the menu command, not a raw `aether` invocation, is what ran.

**Multi-turn chaining** `[CITED: code.claude.com/docs/en/headless]`:
```bash
claude -p "Review this codebase for performance issues"
claude -p "Now focus on the database queries" --continue
# or, to pin a specific session:
session_id=$(claude -p "Start a review" --output-format json | jq -r '.session_id')
claude -p "Continue that review" --resume "$session_id"
```
`--resume` finds the session by ID **anywhere on the machine** (not just the invoking directory) since v2.1.223 — the installed 2.1.278 has this. For a ~12-command journey (init, colonize, discuss, spec, plan, plan, build, continue, pause, resume, seal, entomb, init), the plan should capture `session_id` from the very first `-p --output-format json` call and reuse `--resume "$session_id"` for every subsequent step, each as its own `claude -p ... --resume "$session_id" --output-format stream-json --verbose` invocation — this keeps each step's transcript/exit-code separately assertable (which step failed) while remaining one continuous conversation.

**Caps** `[VERIFIED: claude --help, this machine]`:
- `--max-turns <n>` — already used by the Phase 206 proof script (`--max-turns 6` for one command).
- `--max-budget-usd <amount>` — print-mode only; `proof-screens-reach-the-owner.sh` gates its use behind `claude --help 2>&1 | grep -q -- '--max-budget-usd'` and only adds it if present — copy this defensive pattern; **confirmed present** on this machine's 2.1.278, but the pattern of checking first is still correct future-proofing.
- `--allowedTools <tools>` and `--permission-mode <acceptEdits|auto|bypassPermissions|manual|dontAsk|plan>` — for a 12-step lifecycle journey that must actually write files, `--permission-mode acceptEdits` (auto-approves file writes + common filesystem commands) or `bypassPermissions` is more realistic than the single-command proof script's `--allowedTools Bash` alone; `[CITED: headless.md]` "the built-in starting permission mode [for `-p`] is Manual on every plan" — an unattended multi-step journey needs an explicit mode, not the default.
- Wall-clock: `timeout <seconds> claude ...` (already used: `timeout 300` in the existing proof script) — a longer journey needs a larger per-step timeout; size it per step, not per whole journey, so one slow step doesn't eat the budget for later ones.
- `--output-format stream-json --verbose` for the assertable NDJSON transcript; `--include-hook-events` adds hook lifecycle events to the stream if the plan wants to assert "a hook genuinely fired" directly from the stream rather than from the on-disk JSONL transcript file.

**Hooks actually registered in this repo today** `[VERIFIED: .claude/settings.json, read via python3 json.load this session]`: `PreToolUse`, `PostToolUse`, `Stop`, `PreCompact`, `SessionStart`, plus a top-level `statusLine`. **There is no `UserPromptSubmit` hook registered.** The phase's additional_context lists "SessionStart, PostToolUse, Stop, UserPromptSubmit" as the hooks to check — the fourth does not exist in this codebase. The journey's "hooks loaded" assertion should check the five that are real (via `jq -e '.hooks.PreToolUse and .hooks.PostToolUse and .hooks.Stop and .hooks.PreCompact and .hooks.SessionStart and .statusLine'` against the scratch project's installed `.claude/settings.json`, mirroring gate 1 of `proof-screens-reach-the-owner.sh`), not a `UserPromptSubmit` hook that was never built.

## Q2 — Existing harness in the repo

`[VERIFIED: scripts/, read this session]` Three relevant committed scripts, none of which is today's release gate:

- **`scripts/proof-screens-reach-the-owner.sh`** (Phase 206) — the closest precedent. Full pattern read this session (203 lines): builds `aether` from source into `$WORK/aether`; isolates `AETHER_HUB_DIR` only (deliberately leaves `HOME` alone because Claude Code's own login lives there — the journey should keep this exact rule); `git init`s a scratch repo; runs the real `aether init` then `aether update --force` (the install path under test); asserts the resulting `.claude/settings.json` really registers what it should (`jq -e`); runs one real `claude -p` turn with a retry-once-on-transient policy; asserts delivered content against a *primary* selector (`system`/`informational` stream messages) with a documented *fallback* selector (scan all string leaves except inside `tool_result` blocks) in case the exact envelope shape changes in a future Claude Code release — this fallback-with-documented-reason pattern is worth copying wholesale for the journey's own assertions.
- **`scripts/smoke-daily-driver.sh`** (pre-206) — drives `aether` commands directly (`aether update --force`, `aether host plan --dry-run`, `aether build 1 --plan-only`), never a real chat. This is the REPLAY-tier pattern from the reliability research (§1d) — useful precedent for a *cheaper, non-chat* version of some journey steps, but does not itself satisfy UED-08 (which explicitly requires "a real chat").
- **`scripts/smoke-test-classic.sh`** — unrelated (Classic v5.4.0 regression, isolates `HOME` fully).

**None of the three is wired into a release gate.** `[VERIFIED: grep across cmd/publish_cmd.go, .github/workflows/*.yml, Makefile this session]` — `Makefile`'s only reference to `smoke-daily-driver.sh` is the `smoke:` target, itself uncalled by any other target or by CI. `.github/workflows/ci.yml` and `release.yml` both only match "smoke" in their own unrelated "Binary smoke test" step (a `goreleaser` snapshot version check). Neither workflow references `claude`/`ANTHROPIC` credentials at all (`grep -n "ANTHROPIC\|CLAUDE_CODE\|claude" .github/workflows/*.yml` → no output) — **a `claude -p`-driven gate cannot run in CI today without adding a secret**, which the milestone's "keep it small" framing argues against doing in this phase.

**Where "the release gate" plugs in today:** `cmd/publish_cmd.go` is the entry point for `aether publish`, and its help text points at `aether integrity --source --channel stable` (or `--channel dev`) as a post-publish check `[VERIFIED: cmd/publish_cmd.go:239-241]`. `.aether/docs/publish-update-runbook.md` already documents a manual "Preflight" checklist item (`git status --porcelain` must be empty) that the owner/Claude runs and confirms by hand before every publish — this is the existing pattern for a **procedural, not code-enforced** gate. `cmd/testdata/eval-gates/gates.json` (Phase 204's LEARN-05 work) already names **seven** gates — `fast`, `focused`, `integration`, `provider`, `overnight`, `race`, `release` — each with a `requires` field (`"none"`, `"provider_credentials"`, `"long_wall_clock"`) and a `budget_seconds`; the `release` gate is explicitly documented as `"the union of fast, focused, integration and race, plus the sentinel list — the gate that must pass"` `[VERIFIED: cmd/testdata/eval-gates/gates.json, read this session]` and today does **not** include anything resembling the messy-project journey. The `Makefile`'s `EVAL_GATE_CHECK` macro (`eval-gate-fast`, `eval-gate-focused`, etc.) already implements exactly the "a truncated run must not read as clean" discipline (`discovered==executed` accounting) this repo's CLAUDE.md calls out as load-bearing — the same discipline the journey's 3-trial harness needs for its own trial accounting.

**Recommendation:** add a new `journey` gate to `gates.json` (`requires: "claude_cli_credentials"` — a new, honestly-named requirement distinct from the existing `provider_credentials`, since "provider" in this repo's other gates means an *LLM-worker-under-test* API key, not an interactive `claude` CLI login), reference it from `release`'s purpose text, and add the corresponding manual step to `publish-update-runbook.md`'s Preflight section (mirroring the existing `git status --porcelain` bullet). This satisfies "the journey is the gate for every release from here on" as a *procedural* requirement without adding a new refusal to the `aether` binary itself.

## Q3/Q4 — Trap mechanics and the six 2026-09-21 blockers

**The decision record's exact six-blocker list** `[VERIFIED: .planning/decisions/2026-09-21-v1.29-use-it-every-day.md:5-8, quoted verbatim]`: "a folder missing from a write allowlist, two archive names differing only by letter case, a fingerprint no helper can compute, a planning run pinned to a superseded specification, a pause fingerprint that followed folder shortcuts, a status card advising a menu command that does not exist."

Each was traced to an exact commit + file + function this session (`git show`/`git log -1 --format=%B` + `Read` on the current source):

| # | Blocker (decision record wording) | Commit | File : Function (line range, quoted) | Journey step that exercises it |
|---|---|---|---|---|
| 1 | folder missing from a write allowlist | `cc0d2d07` "Unblock territory refresh: runtime ordered a write its own hook denied" | `cmd/hook_cmds.go:846-871`, `sanctionedDataWritePrefixes` — verified list: `"/.aether/data/planning/"`, `"/.aether/data/phase-research/"`, `"/.aether/data/survey/"`, `"/.aether/data/worker-debug/"`, `"/.aether/data/territory-candidates/"` (the fifth entry is the 2026-09-21 addition; the fix removed here is exactly this one list entry) | `/ant-colonize` (or `/ant-plan`) territory refresh, when the survey is stale enough to trigger a refresh — Q3's "out-of-date code map" trap is what forces this code path to run |
| 2 | two archive names differing only by letter case | `57e563ba` "Keep archive names distinct when they differ only by case" | `cmd/entomb_cmd.go:594-599`, `entombArchiveKey` — verified body: `func entombArchiveKey(archivePath string) string { return strings.ToLower(filepath.ToSlash(archivePath)) }`, and `entombDistinctDataArchivePath` (line 605) which prefixes a clashing data file's archive name with `"data-"` | `/ant-entomb` — trap: a real file at `.aether/HANDOFF.md` *and* a real file at `.aether/data/handoff.md` (different directories, same basename modulo case when the archive flattens both into one directory) |
| 3 | a fingerprint no helper can compute | `0c14cd8f` "Let a research helper register new plan evidence honestly" (+ `e764e79f`, the paired brief-wording fix) | `cmd/codex_plan_finalize.go:2102`, `normalizePlanningScoutStageContent` — the fix makes Go derive the hash/id/digest/scope itself and ignore any hash a Scout worker sends; the file's own comment block (read this session, lines ~2126-2129) states: "new_evidence entry is therefore always treated as an untrusted source" | `/ant-plan` run **twice** — the second pass is exactly where a Scout submits `new_evidence` for a second-round finding; already documented in CLAUDE.md's "Planning evidence" section as the mechanism this fix touches |
| 4 | a planning run pinned to a superseded specification | `f210d5d4` "Do not resume a planning run made for a superseded specification" | `cmd/codex_plan_stage_resume.go:292-294`, `planningRunIsSuperseded` — verified body: `func planningRunIsSuperseded(state planningStageState, approved planningStageSpecificationBinding) bool { return !samePlanningScoutSpecification(state.Specification, approved) }`; wired into `discoverParkedPlanningRun` (line 296) | The "specification corrected mid-planning" trap, at the second `/ant-plan` pass — a run parked mid-plan, then the specification corrected and re-approved before the run resumes |
| 5 | a pause fingerprint that followed folder shortcuts | `b74d25d8` "Let pause fingerprint a project that has folder shortcuts" | `cmd/session_flow_cmds.go:1556-1579` (`repositoryDirtyDigest`) and `:1588-1616` (`pauseDirtyPathDigest`) — verified: the fix switches from `os.ReadFile` (follows symlinks) to `os.Lstat` + explicit symlink-target-string hashing; the function's own comment (read this session) states the exact two prior failure modes: "an unsaved shortcut to a folder (or a nested git repository) failed the whole pause with 'is a directory', and a shortcut to a file outside the project used to pull that file's bytes into the fingerprint" | `/ant-pause` with an uncommitted (dirty) symlink-to-a-folder AND/OR an uncommitted nested git repository present — **both** traps map to this one fix, per the commit's own wording |
| 6 | a status card advising a menu command that does not exist | **No commit found — NOT YET FIXED.** See Open Question below (HIGH-RISK for UED-09). | `cmd/status.go:233` (warning text) and `:732` (`guidedAction.Command`), `renderStatusVisual`/`middenReviewGuidedAction`-shaped code — verified today's live source still reads `fmt.Sprintf("%d unacknowledged failure(s). Run \`aether midden-review\` to inspect.", unackCount)`, and `.claude/commands/ant/` (64 files, listed this session) has **no `midden-review.md`** — there is no `/ant-midden-review` menu command | Trap: leftover junk data (seeded unacknowledged `midden.json` entries) reaching `/ant-status` |

**Trap → journey step / blocker mapping (full 9-trap list from ROADMAP.md, quoted verbatim: "a shortcut to a folder, a shortcut loop, two names differing only by letter case, a nested project, long folder names, an out-of-date code map, a specification corrected mid-planning, leftover junk data, unsaved changes"):**

| Trap | Concrete construction | Journey step | Blocker(s) exercised |
|---|---|---|---|
| shortcut to a folder (symlink → dir) | `ln -s <real-dir> shortcut-dir`, then modify/leave it uncommitted | `/ant-pause` | #5 |
| shortcut loop (symlink cycle) | `ln -s a b; ln -s b a` (or a longer cycle) | robustness only — no historical blocker names this specifically; `filepath.Walk`/`filepath.WalkDir` (36 call sites across `cmd/` and `pkg/`, `[VERIFIED: grep this session]`) do not follow symlinked directories by default in Go's stdlib, so a hang is unlikely; the risk class is a container/path-containment check (20 files call `filepath.EvalSymlinks`, `[VERIFIED: grep this session]`) mis-resolving or erroring on a cyclic target — assert "the command exits (not hangs) and either succeeds or fails with a named reason," not a specific bug | none named; general robustness net |
| two names differing only by letter case | **Two real, differently-located files whose relative *archive* path collides only by case** — e.g. `.aether/HANDOFF.md` + `.aether/data/handoff.md` (this is literally the shape of the real 2026-09-21 bug — verified via `git show 57e563ba`'s commit message, which names these exact two paths). **Caution:** macOS's default APFS volume is case-insensitive-but-case-preserving `[ASSUMED: general macOS knowledge, standard for a default install]` — you cannot create `Foo` and `foo` as two *sibling* entries in the *same* directory on such a volume; `git/git`'s own `report_collided_checkout()` documents that a clone with such colliding paths on a case-insensitive filesystem "SUCCEEDS and WARNS, listing each collided path" rather than refusing `[CITED: .planning/research/2026-09-21-reliability-and-delivery.md §2, sourced from git's `unpack-trees.c`]`. The two-different-directories construction above sidesteps this filesystem limitation entirely and is what the real bug actually was. | `/ant-entomb` | #2 |
| nested project (nested git repo, presumably with its own `.aether`) | `git init` a subdirectory inside the scratch project, optionally `aether lay-eggs` inside it too | `/ant-pause` (per the b74d25d8 commit message, a nested git repo hit the *same* "is a directory" failure as a symlinked folder) — also worth separately exercising at `/ant-colonize` since **no existing code path was found that explicitly detects or handles a nested `.git`/`.aether`** `[VERIFIED: grep -rn "nested.*git\|nested.*\.aether" cmd/ pkg/ — no matches this session]`; this is genuinely unexplored territory and is exactly the kind of thing the journey exists to surface | #5 (via pause); possibly a new, yet-undiscovered defect at colonize/survey time |
| long folder names | An entry-point path deep/long enough to exceed the 40-character canonical-lineage cap | `/ant-discuss` | **Not one of the six.** This is a real, already-fixed 2026-09-21 bug (`1c3a1299` "Fix discuss refusing long entry-point paths"; `cmd/discuss.go:2158-2174`, `specPublicPathLineage`, and `pkg/colony/specification.go:578-579`, verified: `if len(lineage) > 40 { return "", fmt.Errorf("semantic lineage exceeds 40 canonical characters") }`) — useful as a bonus regression check, but does not count toward UED-09's "six blockers" proof since the decision record's list does not name it |
| out-of-date code map | Seed `.aether/data/survey/territory-snapshot.json` (or its metadata) with a `GeneratedAt` old enough / a `SourceRevision` behind enough commits to trip `classifySurveyFreshness` (`cmd/survey_staleness.go:121`, `surveyStaleCommitThreshold = 25` commits, `[VERIFIED: cmd/survey_staleness.go:21]`) into `SurveyFreshnessStale`, forcing a refresh | `/ant-colonize` or `/ant-plan` (territory refresh) | #1 |
| specification corrected mid-planning | Park a planning run mid-flight (e.g. interrupt after the discuss/spec-approval step, before the plan finalizes), then re-approve a changed specification, then resume | `/ant-plan` (second pass) | #4 |
| leftover junk data | Seed `.aether/data/midden.json` with unacknowledged failure entries (and/or stray rows in `.aether/data/pheromones.json`) | `/ant-status` | #6 (currently unprovable — see Open Question) |
| unsaved changes (dirty working tree) | Leave uncommitted edits/untracked files at the moment `/ant-pause` runs | `/ant-pause` | #5 |

**Net finding:** 8 of the 9 named traps map cleanly onto a journey step; the 9th (shortcut loop) is a general robustness check with no named historical blocker. The **"plan twice"** structural requirement in the phase description (not a filesystem trap at all) is what actually exercises blocker #3 (the fingerprint bug) — this should be called out explicitly in the plan so it isn't mistaken for something the fixture script needs to construct on disk.

## Q5 — Assertion strategy

Candidates, in order of how directly they support "files produced and commands run, never wording":

1. **The Claude Code JSONL transcript's `tool_use` blocks** (Bash commands the chat actually ran) — `[VERIFIED: cmd/testdata/stop-hook/menu-command-transcript.jsonl]`, the same surface `proof-screens-reach-the-owner.sh` already parses via `jq -c 'select(.message.content != null) | .message.content[] | select(.type=="tool_use" and .name=="Bash")'`.
2. **`<command-name>` tags** in the transcript's `user` messages — proves a *menu command*, not a raw CLI call, drove each lifecycle step (see Q1).
3. **`.aether/data/COLONY_STATE.json`** phase/state fields — the authoritative on-disk record of what the colony actually did, independent of anything the chat said.
4. **`AETHER_OUTPUT_MODE=json aether status` / `aether history --json`** — stable, already-existing machine-readable projections; `AETHER_OUTPUT_MODE=visual` is what the *owner* sees, `=json` is what a test should parse.
5. **`.aether/data/handoffs/`** — worker handoff records, useful for asserting what a build/continue step believed it accomplished.
6. **Plain file existence/content** on disk (the practice project's own files, after build/continue) — the most direct "files produced" evidence.

None of these require new Aether surfaces to be built — all six already exist. The one piece of net-new plumbing is the **journey's own trial-result JSON** (Q6) recording which trial passed/failed/was-transient, since nothing in this repo currently writes that.

## Q6 — Three trials, flaky vs real, budget

No existing precedent in this repo (`[VERIFIED: grep -rn "flaky\|pass@k\|pass\^k\|transient" cmd/*.go — no relevant matches beyond unrelated retry comments]`). The nearest real, working precedent is external and already researched: cline's `evals/smoke-tests/README.md` runs 3 trials by default and reports `pass@k` (any trial passes) *and* `pass^k` (all trials pass = reliability), and `evals/analysis/patterns/cline-failures.yaml` classifies each failure by regex into `transient` (429/timeout/503 → retriable, doesn't count against the product) vs. `harness`/`environment`/`auth`/`policy`/`provider_bug` `[CITED: .planning/research/2026-09-21-reliability-and-delivery.md §1e]`. `proof-screens-reach-the-owner.sh` already implements a **1-retry-on-transient** policy (`grep -qiE 'rate.?limit|overloaded|timed?.?out'`) for a single step; extending this to a **3-trial, whole-journey** loop is a straightforward generalization of the same pattern, not new design.

**Recommendation:**
- Build tag: `//go:build journey`, skipped unless `AETHER_JOURNEY=1` (matches this repo's own build-tag-gated eval-gate convention — `integration`/`provider`/`overnight` are all "no test declares this tag yet" placeholders following exactly this shape, `[VERIFIED: cmd/testdata/eval-gates/gates.json]`).
- Per-trial budget: size each `claude -p` step's `--max-turns` tightly to what the menu command's own wrapper markdown needs (most `/ant-*` wrappers are "run one `aether` command, relay the output" — 3-6 turns per step, matching the existing proof script's `--max-turns 6`), and a per-trial `--max-budget-usd` ceiling (e.g. $2-3, sized against real observed cost — no data exists yet in this repo for a 12-step chat, so the plan should treat this number as **provisional and instrumented**, not fixed).
- Write results to a JSON report (e.g. `.aether/data/worker-debug/journey-report.json` — inside the already-sanctioned `worker-debug/` write-allowlist prefix, `[VERIFIED: cmd/hook_cmds.go:846-871]`, so the journey harness itself doesn't need a new allowlist entry) so a Go test can assert on trial classification without re-parsing raw JSONL.
- "Fails in all 3 = real; fails in 1-2 = flaky, but still logged and reported" is the natural pass^k-style rule — but the milestone's "one review round" framing argues for keeping the classifier's regex table small (429/rate-limit/overloaded/timeout only) rather than importing cline's full 5-category system.

## Package Legitimacy Audit

Not applicable — this phase adds no new external package dependency. The tools involved (`go`, `git`, `jq`, `claude` CLI, `node`/`npm` already present for other parts of the repo) are all already required by existing committed scripts in this repo (`scripts/proof-screens-reach-the-owner.sh` already lists `go, git, jq, claude` as required tools). No `npm install`/`pip install`/`cargo add` is implicated by anything in this research.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Multi-turn `claude -p` chat state | A custom session/state file | `--resume "$session_id"` (captured from the first call's `--output-format json`) | Claude Code already persists the full conversation; `--resume` finds it by ID anywhere on the machine (v2.1.223+) — reinventing this loses hook context and CLAUDE.md continuity between steps |
| Truncated-run detection for 3 trials | A bespoke pass/fail tally | The `discovered==executed` accounting pattern this repo's `Makefile`'s `EVAL_GATE_CHECK` already implements | This repo has three documented historical incidents of a truncated test run reading as clean (`CLAUDE.md`'s Verification Commands section) — the same failure class applies to a chat that silently stops mid-journey; reuse the proven accounting idiom rather than re-deriving it |
| Deciding whether a `claude -p` failure is worth re-running | Custom heuristics | The transient-vs-real regex classifier already prototyped in `proof-screens-reach-the-owner.sh` (`rate.?limit\|overloaded\|timed?.?out`), generalized per cline's external precedent | Already proven working in this exact repo, on this exact CLI, today |

**Key insight:** almost nothing about this phase's *mechanics* is new invention — Phase 206 already solved "drive one real `claude -p` turn, assert on the transcript, isolate the hub, retry once on transient failure." Phase 207's job is composition (many steps, many trials, many traps) and honest accounting (does the fix-revert loop really prove what UED-09 claims), not new technique.

## Common Pitfalls

### Pitfall 1: `--bare` becoming the default
**What goes wrong:** a future Claude Code release flips `-p`'s default to `--bare`, silently disabling hooks and custom commands — the exact thing UED-08 requires be loaded.
**Why it happens:** the docs already say this is planned ("will become the default for -p in a future release").
**How to avoid:** never rely on the *absence* of a flag; the journey should assert (e.g. via `--include-hook-events` in the stream, or the `<command-name>` tag check) that hooks/commands genuinely fired, not merely that `--bare` wasn't passed.
**Warning signs:** the journey passes green but the transcript shows no `hook_started`/`hook_response` events and no `<command-name>` tags.

### Pitfall 2: Treating the sixth blocker as already fixed
**What goes wrong:** the plan writes a "revert fix #6, expect journey to fail" step against a commit that doesn't exist, and either invents one or silently drops the check.
**Why it happens:** the decision record lists six blockers as a single retrospective paragraph; five have obvious, easily-`git blame`-able fixes; the sixth (status card advising `aether midden-review`, no `/ant-midden-review` wrapper) is explicitly still-open, named as a **Phase 208 (UED-13) deliverable** in ROADMAP.md's own "Known items" list.
**How to avoid:** see the Open Question below — the plan should treat #6 as "assert now (expected initially red), closed by Phase 208," not as a revert-a-commit test.
**Warning signs:** a plan task reads "revert the midden-review menu-command fix" with no commit hash attached.

### Pitfall 3: Asserting on assistant prose
**What goes wrong:** a test greps the chat's own text response for a phrase; the model paraphrases and the test flakes on wording alone, not behavior.
**Why it happens:** it's the easiest thing to assert on when writing the test quickly.
**How to avoid:** every assertion in this journey must resolve to (a) a `tool_use`/Bash-call entry, (b) a `<command-name>` tag, or (c) on-disk state (`COLONY_STATE.json`, file existence, `aether status --output-format json`) — exactly the discipline the milestone decision record names explicitly ("never on wording") and `proof-screens-reach-the-owner.sh` already follows.
**Warning signs:** `grep -i "success"` or similar against the chat's final text.

### Pitfall 4: Case-only filename trap constructed the wrong way
**What goes wrong:** the fixture script tries to create two sibling files in one directory differing only by case (`Foo.md`/`foo.md`) on the developer's default macOS filesystem and it silently overwrites/aliases, so the trap never actually exists.
**Why it happens:** macOS's default APFS volume is case-insensitive but case-preserving.
**How to avoid:** construct the trap the way the real 2026-09-21 bug actually happened — two files in **different directories** whose *archive-relative* names collide only by case (`.aether/HANDOFF.md` + `.aether/data/handoff.md`) — this is what `entombArchiveKey`/`entombDistinctDataArchivePath` actually guards against, and it's trivially constructible on any filesystem.
**Warning signs:** the fixture-build script's "assert each trap exists" check for this trap passes even on a fresh checkout with no traps applied (a sign the check isn't actually distinguishing anything).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | macOS's default APFS volume is case-insensitive-but-case-preserving, making a same-directory case-only filename pair impossible to construct directly | Q3/Q4 trap table | Low — this is standard, widely-documented macOS behavior; even if wrong on some exotic volume config, the recommended construction (different directories) sidesteps the question entirely and is what the real bug actually was |
| A2 | A per-trial `--max-budget-usd` ceiling of roughly $2-3 is a reasonable starting point for a ~12-step lifecycle chat | Q6 | Medium — no real cost data exists for a chat this long in this repo; the plan should instrument and adjust rather than treat this as fixed. Flag explicitly as provisional. |
| A3 | The `journey` eval-gate should use a distinct `requires` value (`"claude_cli_credentials"`) rather than reusing `"provider_credentials"` | Q2 | Low — purely a naming choice; either works functionally, but reusing `provider_credentials` would conflate "an LLM worker under test has API keys" with "the developer machine has an interactive `claude` login," which are different failure modes for CI purposes |

**If this table is empty:** N/A — see entries above; none of the six historical-blocker mappings (#1-#5) are assumptions — each was independently confirmed by reading the actual current source this session.

## Open Questions

1. **UED-09's sixth blocker has no landed fix — this is a planning-blocking discovery, not a minor gap.**
   - What we know: `[VERIFIED: cmd/status.go:233,732, read this session]` the status card today still reads `"Run \`aether midden-review\` to inspect."` and `.claude/commands/ant/` (64 files, listed this session) has no `midden-review.md` wrapper. ROADMAP.md's own Phase 208 section names this exact defect as a "Known item" still to be fixed there (UED-13: "the status card advises `aether midden-review`, which has no menu command and uses an unexplained invented word"). Extensive git-log searching (`--since=2026-09-20 --until=2026-09-22`, grepping for "menu command", "advis", "unblock", "does not exist") across every commit in that window found no commit that added a `/ant-midden-review` wrapper or otherwise closed this gap.
   - What's unclear: whether UED-09 ("the journey catches each of 2026-09-21's six blockers when its fix is removed") is meant literally for all six inside *this* phase, or whether the planner is expected to notice this exact gap and scope Phase 207's revert-loop to the five blockers that do have fixes, while still building the *assertion* for #6 now (in a state that is honestly expected to fail until Phase 208 lands, at which point it starts passing without further journey changes).
   - Recommendation: build the assertion for all six traps/blockers, including #6, in the journey now. For #6 specifically, do **not** frame it as "revert commit X" (there is no commit to revert) — frame it as "assert the status card never advises a command with no menu wrapper" as a standing, general-purpose check (closely related to the `TestSlashCommandGuidancePointsAtRealCommands` pattern already named in Phase 208's REQUIREMENTS.md entry for UED-13), which will be **red today** and is expected to turn green only once Phase 208 ships its fix. Document this honestly in the plan and in UED-09's proof note rather than silently building only 5 of 6, or inventing a fix inside Phase 207 that belongs to Phase 208's scope. This is a discuss-phase-worthy question if the owner wants to weigh in on which framing to use — it directly affects whether Phase 207 can claim UED-09 "done" before Phase 208 ships.

2. **Exact per-trial cost/turn budget for the full 12-step chat.**
   - What we know: a single-command chat (Phase 206's proof) used `--max-turns 6` and (implicitly, via the conditional flag) up to `--max-budget-usd 1.00`.
   - What's unclear: no data exists for a chat this much longer; sizing too tight risks false failures (hitting the cap mid-legitimate-work), sizing too loose risks real cost/time overruns across 3 trials × 12 steps.
   - Recommendation: instrument the first real run generously, record actual observed cost/turns per step, then tighten the caps in a follow-up commit — treat the first numbers in the plan as provisional (see Assumption A2).

3. **Nested-git-repo / nested-`.aether` handling has no existing code to point to.**
   - What we know: no grep hit for any existing detection/handling of a nested `.git` or nested `.aether` anywhere in `cmd/` or `pkg/`.
   - What's unclear: whether this trap will surface a real, previously-undiscovered defect (which is exactly the point of Phase 207) or whether Aether's existing repo-root resolution already happens to be robust to it by construction (e.g. because it always resolves from `cwd` upward to the nearest `.git`, never recursing into subdirectories unprompted).
   - Recommendation: treat this trap as genuinely exploratory — build it into the fixture, run the full journey against it, and let whatever breaks (if anything) become the discovery this phase is named for, rather than assuming a specific existing bug it must catch.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `go` | Build the binary under test, run the `journey`-tagged test | ✓ | go1.26.5 darwin/arm64 (matches `go.mod`'s `go 1.26.5`) | — |
| `git` | Scratch project, trap construction, gate scripts | ✓ | 2.55.0 | — |
| `jq` | Parsing JSONL/JSON assertions (existing pattern from `proof-screens-reach-the-owner.sh`) | ✓ | jq-1.8.1 | — |
| `claude` CLI | Driving the real chat | ✓ | 2.1.278 (Claude Code) | None viable for UED-08's "real chat" requirement — this is a hard local/CI-credential dependency, not something with a graceful fallback |
| `node`/`npm` | Not directly needed by the journey itself, but present for other repo tooling | ✓ | v26.7.0 / 11.19.0 | — |
| ANTHROPIC/Claude Code auth in GitHub Actions | Running the journey unattended in CI | ✗ (verified: no secret references in `ci.yml`/`release.yml`) | — | The journey stays a **local, manual pre-publish gate** for this phase — matching `proof-screens-reach-the-owner.sh`'s and `smoke-daily-driver.sh`'s existing pattern of being Makefile targets nobody wires into CI |

**Missing dependencies with no fallback:** none that block *building* this phase's deliverables — the one real gap (CI credentials for an unattended run) is a scope decision, not a missing tool, and this research recommends treating it as out of scope for Phase 207 (local gate only).

**Missing dependencies with fallback:** none.

## Validation Architecture

*(Included per this invocation's explicit request — `.planning/config.json`'s `workflow.nyquist_validation` is `false`, which would ordinarily skip this section, but the phase's own additional_context explicitly asked for it as one of the specific questions to answer.)*

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` package (`go test`), matching every other command in this repo; shell-script harness pattern (`scripts/*.sh` + `set -euo pipefail` + `trap ... EXIT`) for the parts that must shell out to `claude` |
| Config file | New: `cmd/testdata/eval-gates/gates.json` gains a `journey` entry (none exists yet) |
| Quick run command | `go build ./cmd/aether` (fixture-builder script's own self-check, "runs clean twice") — a few seconds |
| Full suite command | `AETHER_JOURNEY=1 go test -tags journey ./... -run TestJourney -v` (or a dedicated `./journey/` package, per the phase description's own suggestion) — expected to run 3 trials × ~12 chat steps, realistically 15-45+ minutes wall clock depending on caps chosen (Open Question 2) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| UED-07 | The committed builder script constructs all 9 traps; running it twice is clean (idempotent/re-runnable) | integration (shell + Go assertion) | `./scripts/build-messy-practice-project.sh <dest>` run twice, then `go test ./cmd -run TestMessyPracticeProjectHasEveryTrap` (or similar, asserting each trap's on-disk/git signature) | ❌ Wave 0 — neither the builder script nor the asserting test exists yet |
| UED-08 | Full lifecycle journey via a real chat, 3 trials, money/turn caps, asserts on files+commands not wording | journey (build-tag-gated Go test shelling out to `claude`) | `AETHER_JOURNEY=1 go test -tags journey ./... -run TestJourney -v` | ❌ Wave 0 — does not exist; nearest precedent is `scripts/proof-screens-reach-the-owner.sh` (single-step, not multi-trial) |
| UED-09 | Reverting each of the (five landed + one pending, see Open Question 1) 2026-09-21 fixes makes the journey fail at the matching step | journey, parameterized per-fix (likely `git stash`/checkout of the pre-fix commit for each file in the Q3/Q4 table, rerun the relevant journey step, assert failure, restore) | Same `TestJourney` harness, invoked once per reverted fix in a loop (a CI-unfriendly, deliberately manual/local proof given it needs to check out old commits inside the working tree) — the plan should specify exactly how reverts are staged (a throwaway worktree, per this repo's own documented lesson in CLAUDE.md's "Concurrent sessions" / "gate worktree path needs Aether" memory notes, **not** the main checkout mid-test) | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go build ./cmd/aether` + `go vet ./cmd/...` (fast, no chat)
- **Per wave merge:** the fixture-builder script run twice + its asserting Go test (UED-07), without the full chat-driven journey (keeps iteration fast)
- **Phase gate:** the full `AETHER_JOURNEY=1` journey (3 trials, full lifecycle) run at least once before the phase is marked verified, plus the revert-loop for each of the mapped fixes

### Wave 0 Gaps
- [ ] `scripts/build-messy-practice-project.sh` (or equivalent, committed per UED-07's literal wording "a committed script") — covers UED-07
- [ ] A Go test asserting each of the 9 traps' on-disk/git signature after the builder script runs — covers UED-07
- [ ] `journey`-tagged Go test (or `journey/` package) driving the full `claude -p`/`--resume` lifecycle chat — covers UED-08
- [ ] Trial/flake classification + JSON report writer (Q6) — covers UED-08
- [ ] The revert-loop harness/documentation for UED-09, explicitly scoped per Open Question 1 (5 landed fixes + 1 honestly-still-red check) — covers UED-09
- [ ] `journey` entry added to `cmd/testdata/eval-gates/gates.json` + `eval-gate-journey` Makefile target + publish-runbook Preflight bullet — covers "the journey is the gate for every release from here on" (ROADMAP success criterion 4, not itself a numbered UED requirement but explicit in the phase's success criteria)

## Sources

### Primary (HIGH confidence — read or executed this session)
- `.planning/phases/207-messy-practice-project-gate/207-CONTEXT.md`
- `.planning/REQUIREMENTS.md`
- `.planning/STATE.md` (partial — frontmatter + Current Position + Decisions log through Phase 199)
- `.planning/ROADMAP.md` (milestone section + full Phase 207/208 text + traceability table header)
- `.planning/decisions/2026-09-21-v1.29-use-it-every-day.md`
- `.planning/research/2026-09-21-reliability-and-delivery.md`
- `.planning/research/2026-09-21-popular-frameworks.md`
- `.planning/phases/206-screens-reach-the-owner/206-CONTEXT.md`
- `scripts/proof-screens-reach-the-owner.sh` (full 203 lines read)
- `scripts/smoke-daily-driver.sh` (full 209 lines read)
- `cmd/testdata/stop-hook/menu-command-transcript.jsonl`, `menu-command-hide-screen-stop-payload.json` (real captured Claude Code session fixtures)
- `cmd/hook_cmds.go:846-871` (`sanctionedDataWritePrefixes`, `protectedHookWriteReason`)
- `cmd/entomb_cmd.go:594-609` (`entombArchiveKey`, `entombDistinctDataArchivePath`)
- `cmd/session_flow_cmds.go:1552-1616` (`repositoryDirtyDigest`, `pauseDirtyPathDigest`)
- `cmd/codex_plan_finalize.go:2102-2144` (`normalizePlanningScoutStageContent`)
- `cmd/codex_plan_stage_resume.go:292-306` (`planningRunIsSuperseded`, `discoverParkedPlanningRun`)
- `cmd/discuss.go:2158-2174` (`specPublicPathLineage`), `pkg/colony/specification.go:570-585` (`CanonicalSpecItemID`, 40-char cap)
- `cmd/survey_staleness.go:1-80` (`SurveyFreshnessReasonCode`, `surveyStaleCommitThreshold`)
- `cmd/status.go:220-245, 720-740` (midden-review advisory text)
- `cmd/proof_cmd.go` (full file — confirmed unrelated to a "journey" concept, just context/skill proof)
- `cmd/testdata/eval-gates/gates.json` (full gate table)
- `Makefile` (full `EVAL_GATE_CHECK` macro and targets)
- `.github/workflows/ci.yml`, `.github/workflows/release.yml` (full content read)
- `.aether/docs/publish-update-runbook.md` (first ~60 lines)
- `.claude/settings.json` (hooks registered — read via `python3 json.load`)
- `git log`/`git show` on: `cc0d2d07`, `57e563ba`, `32d209fb`, `0c14cd8f`, `e764e79f`, `f210d5d4`, `9336e5ff`, `b74d25d8`, `7df8360b`, `1c3a1299`, `3276eb71`, `2b7901a5`, `a97b1f68`, `f463c47b`, `2713dc87`, `0bf1fbf1`, `5367ff91`, `198e4da8`, `bb8dec56`, `d83c0156`, `4275c479`, `28512d14`, `67916762`, `dd885403`, `8dc9454d`, `a10b5ea5`
- `claude --help`, `claude --version` (this machine, 2026-09-22, Claude Code 2.1.278)
- `code.claude.com/docs/en/headless` (fetched live via WebFetch, 2026-09-22)

### Secondary (MEDIUM confidence)
- `.planning/research/2026-09-21-reliability-and-delivery.md` §1a-1e, §2, §3 (WebSearch-verified against named files in obra/superpowers, anthropics/claude-code-action, cline, goose, SWE-agent, jj, git, terraform, syncthing, restic, brew — this research doc itself states every claim cites a file read that day; treated here as CITED, not independently re-verified this session)
- `.planning/research/2026-09-21-popular-frameworks.md` (same status)
- WebSearch result on slash-command expansion in `-p` mode (corroborates, rather than solely sources, the `code.claude.com/docs/en/headless` fetch and the transcript fixture)

### Tertiary (LOW confidence)
- A1 (macOS case-insensitivity default) — standard platform knowledge, not verified against this specific machine's actual volume format this session

## Metadata

**Confidence breakdown:**
- Claude Code CLI mechanics (Q1): HIGH — live `--help`/`--version` output plus a same-day official-docs fetch plus an actual captured transcript fixture in this repo, all agreeing
- Existing harness / release-gate wiring (Q2): HIGH — directly grepped/read every file named
- Six-blocker mapping (Q3/Q4): HIGH for five of six (each independently `git show`n and cross-checked against current source); the sixth is a **confirmed gap**, not a confidence problem — it genuinely has no fix yet, which is itself the finding
- Trial/flake design (Q6): MEDIUM — no in-repo precedent exists; recommendation is grounded in already-researched external precedent (cline) plus this repo's own existing single-trial retry pattern, but the exact numbers (budget, turns) are provisional

**Research date:** 2026-09-22
**Valid until:** ~14 days (fast-moving area — this milestone is under active daily development, and blocker #6's status could change under Phase 208 at any time)
