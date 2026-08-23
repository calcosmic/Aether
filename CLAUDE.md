# CLAUDE.md — Aether Development Guide

> **Current Version:** v1.0.63
> **Last Updated:** 2026-08-17

> ## READ THIS BEFORE YOU WRITE ANYTHING TO THE OWNER
>
> The owner is non-technical. Every word this repo invented — ratchet, residue,
> caste, pheromone, colony, seal, midden, orphan, allowlist, hub, wrapper, Queen —
> means nothing to him. Translate each one inline, every single time, including in
> questions and option labels. Test output is jargon too: say "I ran the full test
> suite and everything passed", never "18/18 packages, race detection, vet clean".
> Finding IDs (CR-06, WR-04) and criterion numbers are internal bookkeeping — say
> what the problem *is* instead of naming its code.
>
> **The check:** re-read your message as someone who has never opened a file here.
> If any sentence needs a file, a code, or a repo word to make sense, rewrite it.
>
> Full guidance in "Communication Style" below. This has been restated three times
> because it kept being missed — the fix is behaviour, not more text. Do not edit
> this file in response to a too-technical answer. Rewrite the answer.

---

## Quick Reference

| What | Count/Status |
|------|--------------|
| Version | v1.0.63 |
| Slash commands | 60 (Claude) + 60 (OpenCode); Codex uses native CLI + 27 TOML agents |
| Agent definitions | 27 |
| Skills | 86 (55 colony + 31 domain) |
| Go binary | `aether` CLI (Go binary in cmd/) |
| Tests | 5000+ passing |
| Architecture doc | `RUNTIME UPDATE ARCHITECTURE.md` |

## Platform Policy

- **Primary platforms:** Claude Code and OpenCode. These are the main maintained user surfaces.
- **Secondary platform:** Codex CLI. Codex has best-effort support for the direct `aether` workflow.
- **Expectation:** keep Claude/OpenCode command and agent UX aligned first. Keep Codex safe, usable, and accurate about its native CLI capabilities.

## Definition of Done

**A requirement is satisfied only when a command exists that someone can run, and that command fails when the requirement is unmet.**

Not a commit. Not "covered by existing tests." Not "existing behavior." Not a checked box in a summary.

This rule exists because the project's own audits caught the alternative failing:

- `.planning/v1.14-MILESTONE-AUDIT.md:406` — *"the 96-02 SUMMARY claimed this wiring existed, but the implementation was never added."* The milestone shipped 19/19.
- v1.23 marked `WORKFLOW-01..11` **Satisfied** with VERIFICATION.md missing, SUMMARY.md missing, and the traceability box unchecked. Its own note: *"This is 1–2 hours of paperwork, not engineering work."*
- `suggest-analyze` was ticked complete in v1.11. Its only call site sat in a playbook the runtime never loaded, behind `2>/dev/null`, followed by *"If suggest-analyze returns error: Skip without error, continue."* It had never executed once.
- The learning pipeline has been declared restored in v1.10, v1.11, v1.13 and v1.23. `consolidation-phase-end` and `consolidation-seal` had no caller until v1.25 (Phase 162) wired them into the continue and seal paths, locked by named wiring tests.

18 of 25 milestones have been framed around restoring, recovering or repairing something previously marked done. That is what a non-executable definition of done produces.

**Corollaries:**

- An inspection or `--dry-run` command must not mutate state. `consolidation-phase-end --dry-run` and `consolidation-seal --dry-run` wrote to `instincts.json` for months despite the flag reading *"Report without modifying"* — fixed in v1.25 and locked by `TestConsolidationPhaseEndDryRunDoesNotMutate` / `TestConsolidationSealDryRunDoesNotMutate`.
- A documentation claim about runtime behaviour must be testable or removed. Three files described phase-end consolidation running for months while it never ran.
- Prefer a test that asserts a **proportion or an invariant** over one that asserts a section exists. `TestBuildWorkerBriefIsMostlyTask` fails if framework scaffolding ever outweighs the task again, whatever the new scaffolding is called. A test that only checks for a named section cannot catch its replacement.

## Communication Style

- Always include a short plain-English, "for dummies" explanation alongside technical details when explaining work to the user.
- Explain what changed, why it matters, and what it means for the user before going deep on implementation details.
- Translate jargon the first time it appears. Summarize command output and error logs unless exact lines are needed for the next action.

### The owner is non-technical. This repo's vocabulary is invented.

Aether names things after an ant colony. **None of these words mean anything
outside this repo**, and using one untranslated is the same as using a word
the reader has never seen. Assume zero prior knowledge of every term below,
every time — including in questions, option labels, and progress updates.

| Repo word | What it actually means |
|---|---|
| Colony | One project Aether is working on, start to finish |
| Queen | The coordinator that decides which helpers to send at a task |
| Caste / worker | A type of AI helper with one job (write code, check the work, research) |
| Pheromone | A note you leave that steers the helpers ("focus here", "never do this") |
| Instinct / wisdom | A lesson the system learned and now reuses automatically |
| Hive | Shared lessons pooled across all your projects, not just this one |
| Hub (`~/.aether/`) | The installed copy on your machine that every project reads from |
| Wrapper | The `/ant-…` menu commands you type; they call the real program underneath |
| Runtime / the Go binary | The actual program that does the work and holds the truth |
| Seal / entomb | Mark a project finished / file it away in the archive |
| Midden | The log of things that went wrong, kept so the system stops repeating them |
| Playbook | An old instruction document; most are no longer read by the program |
| Orphan | A feature that was built but nothing ever calls, so it never runs |
| Ratchet | An automatic check that can only get stricter, never looser |
| Allowlist | The written list of known exceptions a check agrees to ignore for now |

### Questions must be answerable without opening a file

When asking the owner to choose between options, each option must state the
real-world consequence in ordinary words **before** any file path, command
name, or repo term appears. Paths and identifiers are evidence for the choice,
not the description of it — put them at the end, or in a skippable aside.

If an option cannot be understood without knowing what a word in it means,
the option is written wrong. Rewrite it; do not expect the owner to look it up.

## UX Architecture

Aether uses a hybrid UX model: the Go runtime owns truth, platform wrappers own presentation.

### Ownership Model

| Layer | Owner | What |
|-------|-------|------|
| State mutations | Go runtime (`cmd/`) | Colony state, phase transitions, verification, gating |
| Visual rendering | Go runtime (`cmd/codex_visuals.go`) | Banners, progress bars, caste identity, stage markers |
| Colony framing | Wrapper markdown (`.claude/`, `.opencode/`) | Queen persona, narration, pacing, pre/post-build context |
| Codex UX | Go runtime only | No wrapper markdown — Codex is runtime-native |

### Caste Identity System

Each worker caste has three visual components:
1. **Emoji prefix** — caste glyph followed by the ant, the classic v5.4.0 house style (e.g., `🔨🐜` for Builder, `👁️🐜` for Watcher; the generic fallback stays a single `🐜`)
2. **ANSI-colored label** — the primary identity, colored by caste (e.g., "Builder" in yellow)
3. **Deterministic name** — hash-based per caste+task (e.g., "Mason-67")

Format: `🔨🐜 Builder Mason-67  Task description` (locked by `TestCasteIdentityUsesHouseStyle`)

Color maps in `cmd/codex_visuals.go`: `casteColorMap` (ANSI codes), `casteEmojiMap` (single emoji), `casteLabelMap` (human-readable name). Functions: `casteIdentity()`, `casteLabel()`, `casteEmoji()`.

### Stage Markers

Build and continue output uses `── Stage Name ──` separators between sections (Context, Tasks, Dispatch, Verification, Housekeeping, Next Phase, Colony Complete).

### YAML Source Chain

```
.aether/commands/*.yaml          ← Source definitions (name, runtime command, guardrails)
    ↓ (generation)
.claude/commands/ant/*.md        ← Claude Code wrappers
.opencode/commands/ant/*.md      ← OpenCode wrappers
    ↓ (delegation)
cmd/codex_*.go                   ← Go runtime (authoritative)
```

### Wrapper-Runtime Contract

Full contract documented in `.aether/docs/wrapper-runtime-ux-contract.md`. Key rules:
- Wrappers may add colony framing and narration but must not mutate state
- Wrappers must not duplicate verification or gating logic
- Codex gets UX improvements through the runtime renderer only

### Queen-Owned Orchestration

The Queen chooses execution and review depth autonomously by default. Users
should not need to remember `--verification-depth` or timeout flag combinations.

**How depth is chosen (priority order):**
1. Explicit `--heavy` or `--light` flag (user override)
2. Explicit `--verification-depth <light|standard|heavy>` (user override)
3. Keyword match in phase name (security/auth/release → heavy)
4. Smart default based on phase mode and risk

**Smart defaults:**
- Discovery mode → light
- Production mode → at least standard, heavy for high-risk phases
- Phase name contains "security", "auth", "release", etc. → heavy
- Where a phase sits in the plan no longer matters: the last phase of a plan
  gets the same automatic depth a middle phase with identical mode and risk
  would get (owner's choice, 2026-08-23). An explicit `--heavy` flag still
  raises depth on any phase, position included.

**What each depth means:**

| Flow | Light | Standard | Heavy |
|------|-------|----------|-------|
| **Build** | max 5 workers | the phase's own budget (4–8) | at least 8 |
| **Continue** | max 3 workers, no reviewer required unless a named risk forces one | max 4 workers, same | + Gatekeeper + Auditor + Probe, the full review panel (max 6) — an explicit owner override |
| **Seal** | no Gatekeeper, Auditor or Probe (max 4 workers) | max 4 | max 5 |

Build depth adjusts the phase's own mode/risk budget: **light** lowers it,
**heavy** raises it, and **standard** leaves it alone — standard means the
Queen's ordinary judgement, which mode and risk already express. A flat standard
ceiling would weaken exactly the phases that need most help.

**A reviewer is required only by a named risk signal — never because a phase
"feels" risky.** Every other implicit floor from earlier versions of this
section is gone: a build no longer requires a Watcher unconditionally, and
continue's light and standard depths require nothing unconditionally either.
What is actually required:

| Worker | Required when |
|--------|----------------|
| Builder | on any non-discovery build — the worker that writes the code. A discovery-mode build sends one Scout instead, because research is the deliverable there, not code. |
| A forced reviewer | only when the phase's own wording, or (at the checking step) its changed files, names one of exactly five signals: **credentials/auth** (passwords, logins, sessions, secrets) → security reviewer (Gatekeeper); **payments** → security reviewer; **release sign-off** → security reviewer; **data deletion** → quality reviewer (Auditor); **database migration** → quality reviewer. The check-in card and the checking step both name the exact signal that forced the reviewer, e.g. "a security reviewer will check this — this touches logins and passwords." |

A build's own checks — build, vet, tests, lint — are unchanged by any of this
and run on every phase at every depth, whether or not a reviewer is sent; a
falling worker count is exclusively the review team shrinking, never the
program's own verification.

A one-task bug fix with none of the five signals now gets exactly one worker
(the Builder) across the whole build-and-check cycle, on both the judged and
the no-proposal/autopilot path — down from the eight workers (Builder,
Watcher, Auditor, Probe, Tracker at build; Watcher, Auditor, Probe at
continue) the same fix measurably drew before this ruling, because the old
floor sent a Watcher, an Auditor and a Probe to every build and continue pass
regardless of what the phase actually needed. Only the owner can decline a
forced reviewer, with a reason that is written down (see Team Check-In
below); the Queen and autopilot never can.

Asserted by `TestReviewerForcedOnlyByNamedRisk` (the five-signal table,
evaluated on the real continue dispatch list), `TestNoWorkerWithoutStatedReason`
(a worker named without a reason is refused by name, the rest of the team
still goes), `TestOneTaskBugFixIsOneWorkerPlusChecks` (the one-worker
measurement above), `TestNoCasteIsDispatchedAtBothBoundaries` (no caste is
sent independently by both the build and the checking step for the same
phase), and `TestQueenChoiceReachesTheDispatchList` (a Queen decision is
asserted on the actual spawn list, never an intermediate record a later step
could silently override).

These numbers are asserted by `TestBuildWorkerCapHonoursVerificationDepth` and
`TestCLAUDEMDDepthTableEvaluates`. They were previously documented but not
implemented: the build branch consulted only mode and risk, so a *light*
build of a production phase returned 8 and a *heavy* build of a discovery
phase returned 5. If this table and the code disagree again, those tests fail.

*For dummies: by default the Queen sends only the person writing the code.
A second reviewer only turns up if the work touches one of five specific
risky things — passwords and logins, payments, deleting data, changing the
database's structure, or signing off a release — and joins at the check, not
the build, with the card saying exactly which of the five it was. The
program's own checks (does it build, do the tests pass) are unchanged by any
of this and run on every phase at every depth — a smaller worker count never
means less checking. Picking "light" never switches off a reviewer a risky
phase genuinely needs; only the owner can decide not to send one, and that
decision is written down.*

### Team Check-In and Owner Decisions (2026-08-21)

**Builds pause for the owner before spawning — including a one-worker team.**
After the spawn plan renders, the wrapper shows a check-in card (`aether
ceremony team-checkin`) — one line per worker with the Queen's reason,
`REQUIRED` or `OPTIONAL` marking, and the castes already pruned — then asks:
proceed, trim optional workers, or redirect. `REQUIRED` now means exactly one
of two things: the worker writing the code, or a reviewer forced by one of
the five named signals above, with that signal named beside the reviewer.
Only the owner — never the Queen, never autopilot — can decline a forced
reviewer; declining is recorded through the same mechanism as answering a
clarification, covers that one signal on that one phase, and stays declined
even if the same signal is re-detected later from changed files.
`aether build --no-checkin` skips the pause; autopilot never sees it and can
never decline a reviewer. Locked by `TestRequiredMeansBuilderOrNamedSignal`,
`TestTeamCheckinDoesNotMutate`, `TestOnlyTheOwnerCanWaiveAForcedReviewer`,
`TestWaiverCoversOneSignalOnOnePhase`,
`TestWaivedSignalStaysWaivedWhenTheFilesRedetectIt`, and
`TestAutopilotNeverWaives`.

**Workers' questions route to the owner, not to more agents.** Handoff
`open_decisions` surface via `aether handoff-decisions`; the owner's answers
are stored by `aether decision-answer` as resolved clarifications and reach
every later worker prompt as CLARIFIED INTENT. Wrappers ask at wave
boundaries, the post-build checkpoint, and before continue's steering
checkpoint; an unanswered question never blocks. Locked end-to-end by
`TestResolvedOpenDecisionReachesNextWorkerPrompt`; the worker-side rule (ask,
don't guess) by `TestResponseContractTellsWorkersToRouteJudgementCalls`.

**Spawn-economy floors added in the same round:** seal's Probe needs testable
code (`TestSealProbeRequiresTestableCode`); swarm's Scout/Archaeologist are
relevance-selected, only the tracker/builder/watcher trio is mandatory
(`TestSwarmTrivialBugSkipsHistoryAndResearch`); every build-selectable caste
must have a dispatch path (`TestEveryBuildSelectableCasteCanDispatch`); the
fast continue path honors `--castes`
(`TestContinueFastPathHonoursCasteProposal`); light colonize sends two
surveyors, not four (`TestColonizeLightDepthTrimsSurveyors`).

*For dummies: before spending anything, the build now shows you its team and
waits for your OK — safety checkers stay, extras are yours to drop, and the
system remembers what you drop. When a worker hits a question only you can
answer, it asks you at the next natural break instead of sending another
helper to guess. And five places that used to send helpers with nothing to do
have been shut off.*

Wrappers should explain the Queen's choice briefly in plain English and reserve
manual depth flags for advanced overrides. If docs and runtime disagree, runtime wins.

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                     AETHER REPO (this repo)                      │
│                                                                  │
│   cmd/                 ← Go source code (primary)              │
│   ├── main.go         CLI entry point                          │
│   └── *.go            Command implementations (80+ subcommands)│
│                                                                  │
│   pkg/                 ← Shared Go packages                    │
│   ├── agent/          Agent pool, spawn tree, curation ants    │
│   ├── downloader/     Binary download + extraction             │
│   ├── events/         Event bus with TTL                       │
│   ├── exchange/       XML import/export                        │
│   ├── graph/          Knowledge graph persistence              │
│   ├── memory/         Learning pipeline, instincts, promotion  │
│   └── storage/        JSON store, file locking                 │
│                                                                  │
│   .aether/             ← Source companion files for hub publish │
│   ├── commands/*.yaml   Slash command source definitions        │
│   ├── skills/           colony/ (55) + domain/ (31)           │
│   ├── docs/             Distributed documentation              │
│   ├── templates/        Colony state, pheromones, etc.          │
│   ├── utils/            Runtime helper docs and transforms      │
│   └── exchange/         XML exchange modules                    │
│                                                                  │
│   .aether/data/        ← LOCAL ONLY (gitignored)               │
│   .aether/dreams/      ← LOCAL ONLY (gitignored)               │
│                                                                  │
│   .claude/commands/ant/ ← Slash commands (Claude Code)         │
│   .claude/agents/ant/   ← Agent definitions (Claude Code)      │
│   .opencode/commands/ant/ ← Slash commands (OpenCode)          │
│   .opencode/agents/     ← Agent definitions (OpenCode)         │
│   .codex/agents/        ← Agent definitions (Codex, TOML)      │
│   .codex/CODEX.md       ← Codex commands + rules               │
│                                                                  │
│   ~/.aether/           ← HUB (cross-colony, user-level)        │
│   ├── system/          Global agents, commands, skills, docs, templates│
│   ├── QUEEN.md         (wisdom + user preferences)              │
│   ├── hive/            (Hive Brain — cross-colony wisdom)       │
│   │   └── wisdom.json  (200-entry cap, LRU eviction)           │
│   └── eternal/         (hive memory — high-value signals)       │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

Colony-prime assembles worker context from: QUEEN.md wisdom, eternal memory,
pheromone signals, phase learnings, key decisions, blocker flags, user preferences,
clarified intent, parallel mode, and context capsule — all within a token budget
(see Token Budget below).

### Worker Handoff Context Transfer

Build, plan, colonize, and continue dispatches now carry forward a structured
worker handoff in addition to colony-prime context and skill injection. Each
worker result must include a `handoff` object with changed files, commands run,
verification status, known failures, open decisions, assumptions, next-worker
instructions, things not to repeat, and freshness. The runtime stores recent
handoffs in `.aether/data/handoffs/worker-handoffs.json` and injects relevant
entries into later worker prompts as `Previous Worker Handoffs`.

For dummies: this is the short relay note passed from one worker to the next.
It lets faster or smaller models start with the useful facts instead of spending
their first turn rediscovering what the previous worker already learned.

**See `RUNTIME UPDATE ARCHITECTURE.md` for complete distribution flow.**

---

## Development Workflow

### Editing System Files

| What you're changing | Where to edit | Why |
|---------------------|---------------|-----|
| Worker definitions | Aether repo `.aether/` `workers.md` | Source of truth, published to hub |
| Go commands | `cmd/` | Go source code |
| User docs | `.aether/docs/` | Distributed directly |
| Slash commands | `.claude/commands/ant/` | Claude Code commands |
| OpenCode commands | `.opencode/commands/ant/` | OpenCode commands |
| Agent definitions | `.claude/agents/ant/` | Claude Code agents |
| OpenCode agents | `.opencode/agents/` | OpenCode worker definitions |
| Codex agents | `.codex/agents/` | Codex worker definitions (TOML format) |
| Your notes | `.aether/dreams/` | Never distributed |
| Dev docs | `.aether/docs/known-issues.md` | Distributed |

### Autopilot

`/ant-run` — Autopilot that builds, verifies, learns, and advances through phases automatically.

```bash
/ant-run                       # Run all remaining phases
/ant-run --max-phases 2        # Run at most 2 phases then stop
/ant-run --replan-interval 3   # Suggest replan every 3 phases
/ant-run --continue            # Resume after replan pause
/ant-run --dry-run             # Preview the autopilot plan
```

Smart pause conditions: test failures, critical Chaos findings, security gate failures, quality gate failures, runtime verification needed, replan suggestions.

### Intent Capture and Learning

Claude now exposes three pre-build intent commands that route through the shared
Go runtime:

| Command | When to use it | What it does |
|---------|----------------|--------------|
| `/ant-discuss` | After `/ant-init`, before `/ant-plan` | Captures clarifications, stores them as pending decisions, and emits `REDIRECT` pheromones for hard constraints once resolved |
| `/ant-assumptions` | After `/ant-plan`, before `/ant-build` | Surfaces current plan assumptions, writes `assumptions.json`, and emits `FOCUS` / `FEEDBACK` pheromones from the analysis |
| `/ant-profile` | When reviewing or refreshing learned user behavior | Reads or refreshes the behavioral profile and promotes top `[profiled]` directives into `QUEEN.md` |

Typical guided flow:

```bash
/ant-init "Build feature X"
/ant-discuss
/ant-plan
/ant-assumptions
/ant-build 1
/ant-continue
/ant-profile
```

### Publishing Changes

Authoritative runbook: `.aether/docs/publish-update-runbook.md`

```bash
# 1. Edit canonical source files in the Aether repo
vim .aether/commands/build.yaml

# 2. Commit changes
git add .
git commit -m "your message"

# 3. Publish to the hub (recommended)
aether publish

# 4. In other repos, pull updates
aether update --force      # or /ant-update
```

Runtime note:
- `aether publish` is the primary publish command. It builds the binary, syncs companion files to the hub, and verifies version agreement automatically.
- `aether install --package-dir "$PWD"` still works for backward compatibility but does not include version verification.
- `aether publish --channel dev` publishes to the dev channel (`~/.aether-dev/`) for isolated source development.
- `aether integrity` validates the full release pipeline chain (source, binary, hub, companion files, downstream simulation).
- `aether version --check` verifies binary and hub versions agree (exit 0 = match, non-zero = mismatch).
- `aether update` in other repos only syncs from the shared hub. It does not publish local source changes, and without `--force` it can leave stale Aether-managed files behind.
- `aether update` automatically runs stale publish detection — critical stale publishes block the update with a recovery command.
- `aether update --force --download-binary` is the published-release path when you also need the release runtime binary.
- For isolated source-development on this machine, publish the dev channel instead: `aether publish --channel dev --binary-dest "$HOME/.local/bin"` and then use `aether-dev update --force` in target repos. This keeps `~/.aether-dev/` and `aether-dev` separate from the public stable runtime.
- If `aether update --force` shows `Commands (claude)` or `Commands (opencode)` as `0 copied, 0 unchanged`, the hub publish is incomplete. Republish from the Aether repo first, then rerun `aether update --force` in the target repo.
- If the change modifies publish/install/update logic, bootstrap once with `go run ./cmd/aether publish --channel stable --binary-dest "$HOME/.local/bin"`.

---

## Key Directories

### .aether/ (Source of Truth)

```
.aether/
├── workers.md           # Worker definitions, spawn protocol
├── utils/               # Runtime utilities
│   ├── oracle/oracle.md # Oracle loop instructions (loaded by /ant-oracle)
│   └── queen-to-md.xsl  # XSL transform for queen wisdom export
├── skills/              # colony/ (55) + domain/ (31) skill definitions
├── templates/           # 12 templates (colony-state, pheromones, etc.)
├── docs/                # Distributed documentation
├── exchange/            # XML exchange modules (pheromone-xml, wisdom-xml)
├── commands/            # YAML source definitions for slash commands
├── data/                # LOCAL ONLY (never distributed)
│   ├── COLONY_STATE.json  # Colony state with phase tracking + parallel_mode
│   ├── pending-decisions.json
│   ├── assumptions.json
│   ├── behavior-observations.jsonl
│   ├── pheromones.json
│   ├── constraints.json
│   ├── midden/          # Failure tracking
│   └── survey/          # Territory survey results
├── dreams/              # LOCAL ONLY (session notes)
└── oracle/              # LOCAL ONLY (deep research)
```

### .claude/ (Claude Code)

```
.claude/
├── commands/ant/        # 60 slash commands
│   ├── init.md          # Colony initialization
│   ├── discuss.md       # Clarify intent before planning
│   ├── plan.md          # Phase planning
│   ├── assumptions.md   # Surface plan assumptions
│   ├── build.md         # Build orchestrator (loads split playbooks)
│   ├── continue.md      # Continue orchestrator (loads split playbooks)
│   ├── profile.md       # Inspect or refresh the behavioral profile
│   └── ...
├── agents/ant/          # 27 agent definitions
│   ├── aether-builder.md
│   ├── aether-watcher.md
│   ├── aether-scout.md
│   └── ...
└── rules/               # Development rules
    ├── coding-standards.md
    ├── testing.md
    └── ...
```

### .codex/ (Codex CLI)

```
.codex/
├── agents/              # 27 agent definitions (TOML format)
│   ├── aether-builder.toml
│   ├── aether-watcher.toml
│   ├── aether-scout.toml
│   └── ...
└── CODEX.md             # Codex commands + rules
```

### Command Playbooks (Reference Material)

`.aether/docs/command-playbooks/*.md` are reference documentation for the
build/continue methodology. Since v1.25 they are NOT loaded by the runtime and
NOT injected into worker or orchestrator prompts — workers receive the lean
runtime-composed brief instead. Execution behavior lives in one of two
distinct flows, not a single "host-manifest flow":

- **Interactive wrapper (the primary, documented path):**
  `aether build $ARGUMENTS --plan-only` returns a manifest, the wrapper
  spawns workers from it directly, then `aether build-finalize` commits the
  result. `.claude/commands/ant/build.md` explicitly forbids running
  `aether host build` from this path — "the TS host hop is off the
  interactive build path."
- **Autopilot / host-driven (a separate lane, reached only via `aether run`
  or another host-driven invocation — never the interactive wrapper):**
  `aether host build` → the TS host fetches the same `dispatch_manifest` →
  the host itself dispatches workers → `aether build-finalize`.

Authority note:
- `.claude/commands/ant/build.md` describes the plan-only/wrapper-driven flow
  above, not the host-manifest flow — it never touches the TS host.
  `continue.md`'s relationship to the host is different again: its default
  path also never touches the host and runs verification entirely inside the
  Go runtime process, with no manifest step at all; only its opt-in
  heavy-review path (`--classic-ceremony` / `--verification-depth heavy`)
  fetches a manifest from `aether host continue --dry-run` — a real host
  touch point, but dry-run only, where the wrapper still performs the
  reviewer spawning rather than the host dispatching itself.
- OpenCode maintains mirrored command specs in `.opencode/commands/ant/*.md`.
- Agent parity model: `.claude/agents/ant/*.md`, `.opencode/agents/*.md`, and
  `.codex/agents/*.toml` are canonical platform sources. `aether publish`
  installs those sources into the global hub; there are no repo-local
  packaging mirrors.

---

## Rule Modules

Consolidated guidelines in `.claude/rules/`:
- `@rules/aether-colony.md` — Complete colony system guide (single source)

*(Previous separate rule files have been consolidated into aether-colony.md)*

---

## The 27 Agents

| Tier | Agent | Role |
|------|-------|------|
| Core | Builder | Implements code, TDD-first |
| Core | Watcher | Tests, validates, quality gates |
| Orchestration | Queen | Orchestrates phases, spawns workers |
| Orchestration | Scout | Researches, gathers information |
| Orchestration | Route-Setter | Plans phases, breaks down goals |
| Orchestration | Architect | Architecture design, structural planning |
| Surveyor | surveyor-nest | Maps directory structure |
| Surveyor | surveyor-disciplines | Documents conventions |
| Surveyor | surveyor-pathogens | Identifies tech debt |
| Surveyor | surveyor-provisions | Maps dependencies |
| Specialist | Keeper | Preserves knowledge |
| Specialist | Tracker | Investigates bugs |
| Specialist | Probe | Coverage analysis |
| Specialist | Weaver | Refactoring specialist |
| Specialist | Auditor | Quality gate |
| Specialist | Fixer | Autonomous repair |
| Specialist | Medic | Colony health diagnosis and repair |
| Niche | Chaos | Resilience testing |
| Niche | Archaeologist | Excavates git history |
| Niche | Gatekeeper | Security gate |
| Niche | Includer | Accessibility audits |
| Niche | Measurer | Performance analysis |
| Niche | Sage | Wisdom synthesis |
| Niche | Oracle | Deep research, actionable recommendations |
| Niche | Ambassador | External integrations |
| Niche | Chronicler | Documentation |
| Niche | Porter | Post-seal publish and deploy |

---

## Pheromone System

User-colony communication via signals:

| Signal | Command | Priority | Use For |
|--------|---------|----------|---------|
| FOCUS | `/ant-focus "<area>"` | normal | "Pay attention here" |
| REDIRECT | `/ant-redirect "<avoid>"` | high | "Don't do this" (hard constraint) |
| FEEDBACK | `/ant-feedback "<note>"` | low | "Adjust based on this observation" |

**Before builds:** FOCUS + REDIRECT to steer
**After builds:** FEEDBACK to adjust
**Hard constraints:** REDIRECT (will break)
**Gentle nudges:** FEEDBACK (preferences)

**Viewing Signals:**
- `/ant-pheromones` — Full table of all active signals
- `pheromone-display` subcommand — Formatted output with strength % and decay

**Signal Injection:**
- Colony-prime injects active signals into worker prompts via `prompt_section`
- Builder, Watcher, and Scout agents have `pheromone_protocol` sections that instruct them how to act on injected signals
- Signals are grouped by type (FOCUS, REDIRECT, FEEDBACK) in the injected prompt section

**Content Deduplication (v2.0):**
- Each signal gets a SHA-256 `content_hash` on creation
- Writing a duplicate (same type + content hash) reinforces the existing signal instead of creating a new one (strength maxed, `reinforcement_count` incremented)
- `suggest-analyze` deduplicates suggestions against existing active signals and session suggestions

**Prompt Injection Sanitization (v2.0):**
- Pheromone content is sanitized before storage: XML structural tags rejected, angle brackets escaped, shell injection patterns blocked
- Text patterns that attempt LLM instruction override (e.g., "ignore previous instructions") are rejected
- Content capped at 500 characters

**Exchange:**
- `/ant-export-signals` — Export pheromone signals to XML for cross-colony sharing
- `/ant-import-signals` — Import pheromone signals from XML

**Automatic Suggestions:**
- `suggest-analyze` — Analyzes codebase for patterns worth capturing as pheromones
- `suggest-approve` — Tick-to-approve UI for reviewing suggestions
- Runs during build (Step 4.2) to propose contextually relevant signals

**Files:**
- `.aether/data/pheromones.json` — Active signals
- `.aether/data/constraints.json` — Focus areas and constraints (legacy, eventual deprecation)
- `.aether/docs/pheromones.md` — Full guide

---

## Skills System

Skills provide reusable behavior modules and domain knowledge that workers can load
on demand. They come in two categories:

- **Colony skills** (55) — Behavioral patterns that shape how workers operate
  (e.g., TDD discipline, error handling conventions, commit style)
- **Domain skills** (31) — Technical knowledge for specific frameworks, languages,
  or tools (e.g., React patterns, Go idioms, database optimization)

### Where Skills Live

| Location | Purpose |
|----------|---------|
| `.aether/skills/` | Shipped skill source of truth in the Aether repo |
| `~/.aether/system/skills/` | Published hub mirror of shipped skills |
| `~/.aether/skills/domain/` | Custom user-created domain skills |
| repo `.aether/skills/` | Repo-specific custom skills only |
| `~/.codex/skills/aether/` | Small Codex shim set; the `aether-skill-loader` shim explains that skill content already arrives automatically in dispatch responses |

### How Matching Works

Matching and injection are Go functions (`matchSkillsForWorkflow`, `renderSkillInjectResult`
in `cmd/skills.go`) called directly, in-process, from the worker-brief assembler — not a
separate CLI step. Phase 191 deleted the 8 standalone CLI wrappers that used to expose this
as `skill-index`/`skill-detect`/`skill-match`/`skill-inject`/`skill-list`/`skill-diff`/
`skill-parse-frontmatter`/`skill-cache-rebuild` (SKILL-01 — confirmed dead CLI surface with no
caller anywhere; the underlying functions were preserved unconditionally because
`composeBuildManifestBrief` calls them for every worker brief). For each worker:

1. A live scan of installed skills is built (no separate index-build step)
2. Each skill is scored against the current worker using:
   - Workflow context (`build`, `colonize`, `plan`, `continue`)
   - Worker role (builder, watcher, etc.)
   - Task keywords from the worker assignment
   - Detected file/package patterns matched against the codebase
3. Top 3 colony skills + top 3 domain skills are selected per worker

### Skill Injection

Skill content is injected separately from colony-prime context:

- Own 8K character budget (independent of the colony-prime token budget)
- Injected into builder and watcher prompts automatically as part of brief assembly

### Custom Skills

- `/ant-skill-create` — Oracle-powered skill generation from a description
- Manual creation: add a `SKILL.md` file to `~/.aether/skills/domain/`
- Each skill uses frontmatter (name, category, detect patterns, roles)

### Update Safety

- Manifest-based tracking ensures shipped skills update cleanly
- User-created or user-modified skills are never overwritten during `aether update`

---

## Token Budget

Colony-prime assembles worker context within a character budget to avoid prompt bloat:

| Mode | Budget | When |
|------|--------|------|
| Normal | 8,000 chars | Default |
| Compact | 4,000 chars | `--compact` flag or auto-detected |

**Trim order** (first trimmed = lowest retention priority):
1. Rolling summary (trimmed first -- lowest retention priority)
2. Phase learnings
3. Key decisions
4. Hive wisdom
5. Context capsule
6. User preferences
7. QUEEN wisdom (global)
8. QUEEN wisdom (local)
9. Pheromone signals / active signals (trimmed last -- highest retention priority)

Blockers are NEVER trimmed, regardless of budget.

Trimmed sections are logged for debugging. See `pheromone-write` subcommand for implementation.

---

## Hive Brain (Cross-Colony Wisdom)

The Hive Brain is the intelligent layer for cross-colony knowledge sharing. It stores
generalized wisdom derived from colony instincts, scoped by domain, and shared across
all colonies on the same machine.

Cross-colony flow is controlled by a single switch, `AETHER_HIVE_POLICY`: unset or
empty resolves to `promote` (worker retrieval AND automatic seal-time promotion
both on by default), `read` enables retrieval only, `off` disables both, and any
unrecognized value fails safe to `off` with a stderr warning naming it. There is no
separate per-colony consent gate. See `.aether/docs/learning-system-authority.md`
for the full decision record and its reasoning.

### Storage

```
~/.aether/hive/
└── wisdom.json          # Cross-colony wisdom (200-entry cap, LRU eviction)
```

- Entries are capped at 200; least-recently-used entries are evicted when full
- Hub-level file locking prevents concurrent write corruption

### Subcommands

| Subcommand | Purpose |
|------------|---------|
| `hive-init` | Initialize `~/.aether/hive/` directory and empty `wisdom.json` |
| `hive-store` | Store a wisdom entry with deduplication, merge, and 200-cap enforcement |
| `hive-read` | Read wisdom with domain filtering, confidence threshold, and access tracking |
| `hive-abstract` | Generalize a repo-specific instinct into cross-colony wisdom text |
| `hive-promote` | Orchestrate the abstract + store pipeline (end-to-end promotion) |

### Domain-Scoped Retrieval

Colony-prime retrieves hive wisdom scoped to the current project's domain:

1. Reads domain tags from the colony registry entry for the current repo
2. Calls `hive-read` with those domain tags and a confidence threshold
3. Injects domain-relevant wisdom into worker prompts as `HIVE WISDOM (Cross-Colony Patterns)`
4. **Fallback chain:** hive -> eternal -> empty (graceful degradation if no wisdom exists)

### Seal Promotion Hook

During `/ant-seal` (Step 3.7), high-confidence instincts are promoted to the hive:

1. Extracts instincts with confidence >= 0.8 from `COLONY_STATE.json`
2. Promotes each via `hive-promote` with `--text` and `--source-repo`
3. **NON-BLOCKING** — promotion failures are logged but never stop the seal

### Multi-Repo Confidence Boosting

When the same wisdom is confirmed across multiple repositories, confidence increases:

| Repos Confirming | Confidence |
|-----------------|------------|
| 2 repos | 0.70 |
| 3 repos | 0.85 |
| 4+ repos | 0.95 |

- Confidence is **never downgraded** — uses max of current value and tier value
- Same-repo re-promotion is deduplicated (no duplicate entries)

### Legacy: Eternal Memory

The older eternal memory system (`~/.aether/eternal/`) remains as a fallback:

- `~/.aether/eternal/memory.json` — High-value signals promoted from expired pheromones
- `eternal-store` / `eternal-init` subcommands still functional
- Colony-prime falls back to eternal memory when hive has no matching entries

---

## User Preferences

`/ant-preferences` stores user preferences in the hub `~/.aether/QUEEN.md`
under the `## User Preferences` section. Colony-prime also honors repo-local
preferences from `.aether/QUEEN.md`.

| Command | Purpose |
|---------|---------|
| `/ant-preferences "text"` | Add a user preference to hub QUEEN.md |
| `/ant-preferences --list` | List all user preferences |

- Preferences capture communication style, expertise level, and decision patterns
- Colony-prime injects user preferences into worker context
- `/ant-profile` promotes learned directives into the same section with a `[profiled]` prefix
- Max 500 characters per preference entry

---

## Registry

Colony registry tracks all repos using Aether (`~/.aether/registry/`):

- **Domain tags** — Categorize colonies by domain (e.g., `["web", "api"]`)
- **Colony goal tracking** — `last_colony_goal` stored per repo entry
- **Active status** — `active_colony` flag per repo
- Legacy entries are auto-normalized with default values on read

---

## Quality Gates

New agents integrated into continue.md:

### Gatekeeper (Security)
- Runs after verification passes
- Scans for exposed secrets, debug artifacts via `check-antipattern` (~6 patterns -- not a full security scanner)
- Creates blockers if security issues found

### Auditor (Quality)
- Runs after Gatekeeper passes
- Analyzes code quality metrics
- Reports quality gate status

### Probe (Coverage)
- Analyzes test coverage gaps
- Reports coverage percentage
- Suggests additional tests

### Measurer (Performance)
- Performance analysis
- Identifies slow operations
- Reports performance metrics

---

## Midden System (Failure Tracking)

The midden tracks failures for colony learning:

- `.aether/data/midden/midden.json` — Failure records
- `midden-write` — Log a failure
- `midden-recent-failures` — Query recent failures
- `midden-review` — Review unacknowledged midden entries grouped by category
- `midden-acknowledge` — Mark midden entries as addressed by id or category

Failures are logged during:
- Build failures (build.md)
- Approach changes (tracked for wisdom)

**Data Maintenance:**
- `/ant-data-clean` — Remove test artifacts from colony data files (pheromones, constraints, midden)

---

## Memory Health System

Colony memory is tracked and displayed:

- `/ant-status` — Shows memory health table
- `/ant-memory-details` — Drill-down view
- `/ant-resume` — Shows memory health section

Metrics tracked:
- Events count
- Learnings count
- Instincts count
- Pheromones count
- Memory age

---

## Changelog System

Automated changelog collection:

- `changelog-append` — Append entry to CHANGELOG.md
- `changelog-collect-plan-data` — Collect plan data for changelog

---

## Milestone Names

| Milestone | Meaning |
|-----------|---------|
| First Mound | First runnable |
| Open Chambers | Feature work underway |
| Brood Stable | Tests consistently green |
| Ventilated Nest | Perf/latency acceptable |
| Sealed Chambers | Interfaces frozen |
| Crowned Anthill | Release ready |
| New Nest Founded | Next major version |

---

## Verification Commands

```bash
# Run Go tests
go test ./...

# Run Go tests with race detection
go test ./... -race

# Verify Go binary builds
go build ./cmd/aether

# Run Go vet
go vet ./...

# Verify goreleaser config
goreleaser check

# Build snapshot (no tag required)
goreleaser build --snapshot --clean
go test ./...

# Verify binary works
aether version

# Run all Go tests
go test ./... -race
```

---

## Session Freshness Detection

All stateful commands use timestamp verification to detect stale sessions:

**Pattern:**
1. Capture `SESSION_START=$(date +%s)` before spawning agents
2. Check file freshness with `session-verify-fresh --command <name>`
3. Auto-clear stale files or prompt user based on command type
4. Verify files are fresh after spawning

**Protected Commands** (never auto-clear):
- `init` — COLONY_STATE.json is precious
- `seal` — Archives are precious
- `entomb` — Chambers are precious

---

## Session Recovery

On the first message of a new conversation, check if `.aether/data/session.json` exists. If it does:

1. Read the file briefly to check for `colony_goal`
2. If a goal exists, display:
   ```
   Previous colony session detected: "{goal}"
   Run /ant-resume to restore context, or continue with a new topic.
   ```
3. Do NOT auto-restore — wait for the user to explicitly run `/ant-resume`

---

## Wisdom Pipeline

The Wisdom Pipeline is the core learning loop of Aether. Colony work produces
observations that flow through the system and become reusable wisdom.

### Pipeline Stages

| Stage | Subcommand | Output |
|-------|-----------|--------|
| 1. Observe | `memory-capture "learning"` | Records observation to learning-observations.json |
| 1a. Trust score | `trust-score-compute` | Assigns weighted trust score to observation (40/35/25, 7 tiers) |
| 1b. Event bus | `event-bus-publish` | Publishes scored event to JSONL event bus with TTL |
| 2. Auto-promote | (internal: `learning-promote-auto`) | Triggers after threshold (2 observations for patterns) |
| 3. Instinct | `instinct-create` | Stores in COLONY_STATE.json with provenance |
| 4. QUEEN.md | `queen-promote` | Writes to QUEEN.md Patterns/Philosophies section |
| 5. Inject | `colony-prime` prompt_section | QUEEN.md wisdom + instincts injected into worker context |
| 6. Hive store | `hive-promote` | Abstracts instinct, stores in hive wisdom.json (confidence >= 0.8) |
| 7. Hive read | `hive-read` | Retrieves cross-colony wisdom scoped by domain |

**See `.aether/docs/structural-learning-stack.md` for the full Structural Learning Stack documentation.**

### Key Thresholds

- **Auto-promotion:** Pattern observations need 2 captures to trigger; confidence starts at 0.75
- **Hive promotion:** Instincts with confidence >= 0.8 are promoted to Hive Brain at `/ant-seal`
- **Cross-colony boost:** Multi-repo confirmation raises confidence (2 repos = 0.70, 4+ = 0.95)

See [Hive Brain](#hive-brain-cross-colony-wisdom) for cross-colony wisdom details.

---

## Structural Learning Stack

The memory consolidation pipeline that upgrades the Wisdom Pipeline with
trust scoring, graph relationships, and automated curation.

**See `.aether/docs/structural-learning-stack.md` for complete documentation.**

Key additions:
- Trust scoring engine (40/35/25 weighted, 60-day half-life, 7 tiers)
- JSONL event bus with pub/sub and TTL cleanup
- Standalone instinct storage with full provenance
- jq-based graph layer for instinct relationships
- 8 curation ants with orchestrated execution
- Lifecycle integration: both `consolidation-phase-end` and `consolidation-seal` have runtime callers — phase-end consolidation runs automatically on durable phase advance during `/ant-continue`, and the full eight-ant seal pass runs automatically during `/ant-seal`. Both remain directly invocable as the manual inspection path, and both are non-blocking. See `.aether/docs/learning-system-authority.md` for the authority decision and the named tests enforcing this

### Curation Ants

| Ant | Role |
|-----|------|
| `orchestrator` | Coordinates curation pipeline execution |
| `archivist` | Archives and retrieves historical observations |
| `critic` | Evaluates instinct quality and confidence |
| `herald` | Broadcasts high-confidence instincts to hive |
| `janitor` | Cleans stale events and expired TTL entries |
| `librarian` | Indexes and catalogs instinct relationships |
| `nurse` | Heals low-confidence instincts with supporting evidence |
| `scribe` | Records curation decisions and audit trail |
| `sentinel` | Guards against instinct corruption and conflicts |

---

## Parallel Execution Modes

Aether supports two parallel execution strategies, selected at colony init:

| Mode | Value | Description |
|------|-------|-------------|
| In-repo (default) | `"in-repo"` | Workers share the working tree directly |
| Worktree | `"worktree"` | Each worker gets an isolated git worktree, then file and pheromone changes sync back into the main root |

### How It Works

1. **`/ant-init`** prompts for strategy selection (Step 6.5) during colony setup
2. The choice is saved as `parallel_mode` in `COLONY_STATE.json` (omitempty, defaults to `"in-repo"`)
3. **Colony-prime** injects the current parallel mode into worker context
4. **The build runtime** conditionally allocates worktrees when mode is `"worktree"`
5. **The continue runtime** performs mode-aware cleanup/merge checks after worktree sync-back
6. **Status and resume** commands display the active parallel mode

### CLI Subcommands

| Subcommand | Purpose |
|------------|---------|
| `parallel-mode get` | Read current parallel mode from colony state |
| `parallel-mode set <mode>` | Change parallel mode ("in-repo" or "worktree") |

### Choosing a Mode

- **In-repo** is simpler and works well for most projects. Workers write directly to the repo.
- **Worktree** provides isolation -- each worker gets its own git worktree, and successful file plus pheromone changes are synced back into the main root. Requires a clean working tree and is best for larger tasks with multiple workers touching different files.

---

## The Core Insight

The system's pieces are now **connected**:
- Pheromones update context (colony-prime injects signals into worker prompts)
- Decisions become pheromones (auto-emit during builds)
- Learnings become instincts (observation to promotion pipeline -- see Wisdom Pipeline above)
- Midden affects behavior (threshold auto-REDIRECT)
- Hive Brain crosses colony boundaries (domain-scoped wisdom -> colony-prime)
- Instincts promote to hive at seal (confidence >= 0.8 -> hive-promote)
- Multi-repo confirmation boosts confidence (2 repos = 0.7, 4+ = 0.95)
- User preferences shape worker behavior (QUEEN.md -> colony-prime)
- Autopilot chains build-verify-advance with smart pausing (/ant-run)

**The ongoing challenge is maintenance** -- keeping documentation accurate,
data files clean, and test coverage comprehensive as features evolve.

---

## For OpenCode / Codex

For OpenCode-specific rules and agents, see `.opencode/OPENCODE.md`

For Codex-specific rules and agents, see `.codex/CODEX.md`

---

*Updated for Aether v1.0.63 — 2026-08-21*
