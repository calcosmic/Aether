# CLAUDE.md — Aether Development Guide

> **Current Version:** v1.0.83
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
| Version | v1.0.83 |
| Slash commands | 64 (Claude) + 64 (OpenCode); Codex uses nine public ant skills + native CLI + 27 TOML agents |
| Agent definitions | 27 |
| Skills | 86 (55 colony + 31 domain) |
| Go binary | `aether` CLI (Go binary in cmd/) |
| Tests | 5000+ passing |
| Architecture doc | `RUNTIME UPDATE ARCHITECTURE.md` |

## Platform Policy

- **Primary platforms:** Claude Code and OpenCode. These are the main maintained user surfaces.
- **Secondary platform:** Codex CLI. Codex has best-effort support for the direct `aether` workflow.
- **Expectation:** keep Claude/OpenCode command and agent UX aligned first. Keep Codex safe, usable, and accurate about its native CLI capabilities.

## Codex Public Entrypoints

Pick an ant skill in Codex to start the matching workflow. The skill reads Aether's
instructions; the `aether` executable still owns state, checks, and authorization.

Aether's Codex actions use skills, with exactly nine public names:

| Skill | Workflow |
|-------|----------|
| `$ant-init` | Start a colony |
| `$ant-discuss` | Clarify intent |
| `$ant-oracle` | Research a concern |
| `$ant-colonize` | Survey existing code |
| `$ant-plan` | Plan phases |
| `$ant-build` | Build a phase |
| `$ant-continue` | Verify and advance |
| `$ant-swarm` | Route work or watch workers |
| `$ant-seal` | Seal and retain work for review |

The remaining 55 action skills and native-worker parity belong to later inserted
phases. These nine names establish entrypoints, not complete lifecycle parity.

The selected installation root is `~/.codex/skills/aether/`, qualified with Codex
CLI 0.154.0. Each public `ant-<command>/SKILL.md` reads private support relative to
its own installed file, not the working directory:

- `support/aether-colony-creation.md`
- `support/aether-colony-research.md`
- `support/aether-colony-build-cycle.md`

These three ordinary files are private support, not public helper skills. Worker
skills still arrive automatically in runtime dispatch briefs. Start a fresh Codex
session after skill files are copied or removed; unchanged updates need no refresh.

When a user types a literal `aether ...` passthrough command such as `status`,
`update`, `focus`, `pheromones`, or `reference-list`, execute that exact command
first. The installed binary and `aether --help` are the runtime source of truth.

For the nine lifecycle actions above, run or inspect
`aether command-guide <command> --platform codex` and follow the matching public
skill and its private support. If the user explicitly says raw, exact,
no-interview, or no-orchestration, execute the requested CLI command directly.
Do not reinterpret a literal passthrough command as a vague workflow request.

For lifecycle shell execution, prefer `AETHER_OUTPUT_MODE=visual aether ...`
unless the user explicitly wants JSON. Preserve exact arguments. Do not preface
literal passthrough execution with repo archaeology or skill narration; the CLI
output is primary, with at most one short sentence of extra explanation.

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

### How much proof a change needs

The rule above has one setting, and that is deliberate for anything that can be
wrong invisibly. But it has been applied at full force to a comment and to a
trust boundary alike, and mutation-testing a wording change buys nothing.

**Full rigour — a failing command, and the fix proved by breaking it:**

- Anything that decides work is complete, credited, verified, or advanced.
- Anything a worker or a wrapper can influence: parsed output, submitted
  packets, reported evidence. Treat it as untrusted input.
- Anything touching money, tokens, deletion, or another person's data.
- Anything with more than one lane — if the direct path and the chat path can
  disagree, both get proved, because a guarantee that holds only on the path
  nobody uses is worth nothing.
- Any claim written into a shipped document about how the runtime behaves.

**A passing test is enough:**

- Wording, comments, help text, formatting.
- A message no decision reads.
- Renames and moves where the behaviour is unchanged and the compiler agrees.

**The one thing that never relaxes, at either level:** a test must be able to
fail. A fixture built in a shape the runtime cannot produce is not a weaker
test, it is a false certificate — and it is how this project has repeatedly
shipped a broken feature with a green suite. Derive fixture values the way the
runtime derives them; do not type a plausible-looking literal.

*For dummies: prove things hard when being wrong would be invisible — money,
deletion, anything that says work is finished. Prove things normally when being
wrong would be obvious the moment you looked. And never write a check that
cannot fail, whichever side of that line you are on.*

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
| Codex UX | Go runtime visuals + nine public skills | Skills coordinate runtime-owned operations; no new state authority |

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
- Codex visuals come from the runtime renderer; public skill instructions follow runtime command guides

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

**Which spawn list — owner's ruling, 2026-09-12.** A specialist the Queen names
runs at the **check**, not during the build. This had been ambiguous in the worst
possible way: two tests asserted opposite behaviour for the identical scenario, so
one of them was necessarily red, and it stayed red for two days while reading as a
code bug. It was never a code bug — the runtime already ran the specialist at the
check, matching the rule above that a forced reviewer "joins at the check, not the
build" and D11 rule 4 (a phase is verified once). `TestQueenChoiceReachesTheDispatchList`
and `TestNoWorkerWithoutStatedReason` now assert on the continue dispatch list, and
both also assert the specialist does NOT additionally run during the build. The
guarantee is unchanged and still absolute — the Queen's decision must reach a real
spawn list, never stop at a record — only the boundary is now named.

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

### Coherent Jobs and Completion Evidence (2026-08-27)

**Several related tasks become one job for one worker, and the program — not
the Queen, not the wrapper — decides whether that grouping is safe.** Tasks are
grouped when a real dependency chain or a genuinely shared implementation file
connects them; brushing the same bookkeeping file (a README, a changelog, a
dependency list) never fuses unrelated work, and an unusually large cluster is
split unless a specific reason is given for one worker keeping it end to end.
The Queen may propose her own grouping, with a reason naming both the
relationship and the benefit ("these tasks all rewrite the same templates, so
one worker avoids repeating the setup and fighting over the same files"), but
the proposal is validated in Go before anything is dispatched. A group that
would run a task before something it depends on is refused by name, showing the
offending task and the dependency, and dependency-safe jobs are substituted for
that one group only — every other group survives untouched, and the phase is
never blown back apart into one worker per task. A genuine circular dependency
is a hard planning error: the phase stops and prints the loop plus the repair
to make. Locked by `TestCoherentJobProposalOrderRefusedByName`,
`TestCoherentJobRepairKeepsSafeProposals`, `TestCoherentJobGraphErrorsHaveNoPlan`,
`TestCoherentJobsIgnoreIncidentalPaths`, and
`TestCalVaultSixBatchesPlanAsOneCoherentJob`.

**Grouping happens before anyone owns a file, in both working modes.** Whether
workers share one checkout (`in-repo`) or each gets an isolated copy
(`worktree`), the grouping pass runs first and the finished job — not the
individual task — is what claims ownership of files. One job therefore owns the
combined list of its own tasks' files and cannot collide with itself, while two
genuinely different jobs reaching for the same file in the same round are still
refused before any worker starts, naming the contested file and both jobs. The
real-world case that first exposed this — six workers each re-reading the same
source list to copy files from it — now runs as one worker, one isolated copy,
one branch and one merge back, in `worktree` mode as well as `in-repo`. Locked by
`TestCalVaultSixBatchesBecomeOneWorktreeJob`,
`TestGroupedWorktreeOwnsUnionedPaths`, and
`TestDistinctWorktreeJobsStillRejectOverlap`.

**A job's task list says what a worker was asked to do. It never says what got
done.** `covered_task_ids` is assignment scope only. Finishing credit comes from
a two-stage boundary. The first stage checks a worker's task-by-task receipts on
their shape alone and grants no credit whatsoever — it produces candidates, not
completions. Only the second stage, which checks each claim against the files
actually present in the real project, can mark a task complete. In `worktree`
mode the admitted files are copied back in between those two stages, so a file
that exists only inside a worker's private copy
can never become completion credit.
If a worker finishes four of six tasks, exactly those four are credited
and the other two stay pending; anything the worker touched that no admitted
receipt claimed is neither copied back nor deleted — the copy is kept and you
are told which files and which branch.
A partially finished job never reports the project as built.
Locked by `TestCoherentJobReceiptAdmission`,
`TestCoherentJobReceiptFinalization`,
`TestGroupedWorktreePartialReceiptsSyncBeforeCredit`,
`TestGroupedWorktreeUncreditedEditsRemainOrphaned`, and
`TestExternalGroupedPartialPersistsExactTaskState`.

**Retrying picks up only the uncredited tasks.** Recovery plans a new
dependency-safe job containing exactly the unfinished work, revalidates its
order against what was already credited, and is recorded as a new attempt linked
to the one it came from rather than overwriting it — so no fresh worker is ever
asked to redo proven work. The command you are handed names those unfinished
tasks one by one, so running it starts work on only them. Locked by
`TestCoherentJobRetryContainsOnlyUnfinishedTasks`,
`TestGroupedJobRetryNeverReassignsCreditedTasks`, and
`TestPartialRetryCommandNeverRedispatchesCreditedWork` — the last one takes the
command the owner is literally handed, feeds its own arguments back through the
real planner, and fails if any already-proven task turns up in the resulting
list of work.

*For dummies: instead of sending six helpers to do six related things — each one
starting cold and re-reading the same files — the system now sends one helper to
do all six as a single piece of work, after checking that bundling them cannot
make anything run out of order. And when a helper says "I finished the first
four", the system does not take its word for it: it goes and looks at the actual
files in your project, and only what it can genuinely see finished gets ticked
off. The rest stays on the list and is retried, and nothing the helper did is
thrown away in the meantime.*

### Team Check-In and Owner Decisions (2026-08-21, one-worker fast path 2026-08-27)

**A build that needs one worker, with nothing left for you to decide, goes straight through.**
The runtime decides this, not the wrapper, in one fixed
order: autopilot and `aether build --no-checkin` stay non-interactive; an
explicit `aether build --checkin` always pauses, because you asked for it;
anything still waiting on your decision pauses even for a single worker;
exactly one worker with nothing pending takes the fast path; and everything
else pauses exactly as before. Supplying
`--checkin` and `--no-checkin` together is refused by name
before the build opens an attempt, writes a plan,
writes a checkpoint, or touches any saved state — the program never guesses
which flag you meant. "One worker" counts workers, not jobs of work: a single
worker carrying six grouped tasks is still one worker. Locked by
`TestBuildCheckinDecisionMatrix`, `TestOneWorkerBuildSkipsCheckin`, and
`TestCheckinFlagConflictHasNoSideEffects`.

**The fast path still shows you the plan — it just does not ask.** Before
spawning, it prints a short summary naming the worker, every task that worker is
taking, why those tasks belong together and what grouping them buys, and the
plain statement that nothing needs your approval so dispatch continues. It is
never the full check-in card and never asks a question. Locked by
`TestOneWorkerFastPathSummaryCarriesEveryFact` and
`TestFastPathSummaryIsNonBlocking`.

**Anything still waiting on your decision keeps the pause, even for one
worker.** Exactly three live records count as waiting on you: a safety reviewer
forced by one of the five named risk signals that you have not yet declined, an
unanswered question the coordinator raised before planning, and an unanswered
question a worker handed back. None of these is ever inferred from how many
workers were sent or from the wording of a rendered card. Locked by
`TestOneWorkerWithForcedReviewerWaiverStillPauses`,
`TestOneWorkerWithBoundaryQuestionStillPauses`,
`TestOneWorkerWithPersistedOwnerDecisionStillPauses`, and
`TestPendingDecisionStillRendersFullCheckinCard`.

**When the build does pause, the card itself is unchanged.** After the spawn
plan renders, the wrapper shows a check-in card (`aether ceremony team-checkin`)
— one line per worker with the Queen's reason, `REQUIRED` or `OPTIONAL` marking,
and the castes already pruned — then asks: proceed, trim optional workers, or
redirect. `REQUIRED` means exactly one of two things: the worker writing the
code, or a reviewer forced by one of the five named signals above, with that
signal named beside the reviewer. Only the owner — never the Queen, never
autopilot — can decline a forced reviewer; declining is recorded through the
same mechanism as answering a clarification, covers that one signal on that one
phase, and stays declined even if the same signal is re-detected later from
changed files. Autopilot never sees the card and can never decline a reviewer.
Locked by `TestRequiredMeansBuilderOrNamedSignal`,
`TestTeamCheckinDoesNotMutate`, `TestOnlyTheOwnerCanWaiveAForcedReviewer`,
`TestWaiverCoversOneSignalOnOnePhase`,
`TestWaivedSignalStaysWaivedWhenTheFilesRedetectIt`,
`TestAutopilotNeverWaives`, and
`TestRenderCeremonyTeamCheckinStillRendersFullCardForOneWorkerWhenCalled`.

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

*For dummies: if the work needs one helper and there is nothing sitting with you
to decide, the build tells you who it is sending and what they are doing, then
gets on with it — no question, no waiting. If it needs more than one helper, or
if anything is still waiting on you (a safety check you have not signed off, a
question you were asked), it stops and shows you the full team card first, just
as before: safety checkers stay, extras are yours to drop, and the system
remembers what you drop. You can always force the stop by adding `--checkin`, or
skip it with `--no-checkin`; asking for both at once is refused rather than
guessed. When a worker hits a question only you can answer, it asks you at the
next natural break instead of sending another helper to guess. And five places
that used to send helpers with nothing to do have been shut off.*

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
pheromone signals, blocker flags, user preferences, clarified intent, parallel
mode, and context capsule — all within a token budget (see Token Budget
below). Phase learnings and key decisions were removed from this list: neither
had a runtime writer, so the sections could never carry content
(`buildColonyPrimeOutput`, `TestEveryMemoryPackPartHasALiveWriter`).

Build, planning and research briefs also carry two further bounded slots,
each outside the colony-prime budget above: a condensed codebase-map digest
(`resolveSurveyDigestSection`, `TestSurveyorSentenceReachesAllThreeBriefs`)
and, on every phase but the first, a previous-phase carry-forward naming
what failed or was flagged in the phase immediately before it plus the
closing summary the owner read (`resolvePreviousPhaseCarryForward`,
`TestPreviousPhaseFailureReachesTheNextBuildersBrief`). Neither slot ever
draws from the colony-prime budget or the other's budget
(`TestBriefGrowthIsCappedAtTheTwoNewSlots`).

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
├── commands/ant/        # 64 slash commands
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
| `~/.codex/skills/aether/` | Nine public `ant-*/SKILL.md` entrypoints plus three private `support/*.md` bodies |

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

### Promotion to the Hive: at every check, and at seal

A lesson confident enough to share across projects (confidence >= 0.8) used to
wait for `/ant-seal` — a project that is never formally sealed, which is most
of them, contributed nothing. It now also leaves at the end of every check
(`/ant-continue`), on both check lanes, immediately after phase-end
consolidation (`TestStrongInstinctReachesTheSharedStoreAtCheck`).

Both the check-time path and the seal path (Step 3.7) call the exact same
gate and the exact same writer:

1. Extracts instincts with confidence >= 0.8 from `instincts.json`
2. Promotes each via the shared hive writer, honouring `AETHER_HIVE_POLICY`
   identically at both points (`TestHivePromotionAtCheckHonoursThePolicySwitch`)
3. **NON-BLOCKING** — promotion failures are logged but never stop the check
   or the seal (`TestHiveFailureNeverBlocksThePhase`)

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

Failures are logged automatically, on every path that can produce one:
- A helper fails to finish its job, on either the direct build path or the
  chat-driven build path (`TestFailedBuildWorkerReachesTheNextBriefOnTheDelegateLane`)
- The program's own build/type/formatting/test check fails, on either check
  lane (`TestFailedCheckWritesOneFailureRecordOnBothLanes`)
- A one-off question (`/ant-quick`) fails to get an answer
  (`TestQuickFailureReachesTheFailureLog`)
- A bug-investigation helper fails or times out, on either lane
  (`TestSwarmWorkerFailureReachesTheFailureLogOnBothLanes`)
- Autopilot never retries a whole build; provider readiness owns bounded retries
  (`TestAutopilotWholeBuildRetryIsNotWired`)

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

> **Always pass `-timeout 90m`.** The `cmd` suite needs about 21 minutes and Go's
> default per-package timeout is 10. On timeout the suite's own controller stops
> early and prints a complete-looking per-lane summary of the fraction it
> finished — a truncated run reads exactly like a clean one. Before trusting any
> FAIL list, check the `FULL-SUITE ... discovered=N executed=N` headline and
> confirm the two numbers are equal. Measured 2026-09-13: an unqualified
> `go test ./...` reported a result after running 1635 of 5299 tests, with 30
> lanes never started.

```bash
# Run Go tests
go test ./... -count=1 -timeout 90m

# Run Go tests with race detection
go test ./... -race -count=1 -timeout 90m

# Verify Go binary builds
go build ./cmd/aether

# Run Go vet
go vet ./...

# Verify goreleaser config
goreleaser check

# Build snapshot (no tag required)
goreleaser build --snapshot --clean

# Verify binary works
aether version

# Run all Go tests (or: make test)
go test ./... -race -count=1 -timeout 90m
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

The runtime greets you itself. Open a chat, resume one, or carry one on after
clearing it, and the program prints a short card: what the project is, how far
along it is, the one thing to run next, and a couple of alternatives. A folder
with no project set up in it gets nothing at all.

This used to be a paragraph asking the assistant to remember to look at the
saved session file and tell you what it found — a request that ran only when it
was noticed. It is now `aether hook-session-start`, registered in
`.claude/settings.json` for the three moments you arrive with no context
(opening, resuming, and continuing after a clear). The card comes from the same
decision every other command's closing message comes from, so two surfaces can
never name different next steps. Nothing is restored automatically; the card
tells you what to run and you decide.

Locked by `TestSessionStartCardReflectsState`,
`TestSessionStartCardIsSilentWithoutAColony`,
`TestSessionStartHookDoesNotMutate` (the greeting reads and never writes),
`TestSessionStartHookIsRegistered`, and
`TestSessionGreetingIsNotDelegatedToTheAssistant` — the last of which fails if
the old request returns to any shipped document.

The card also names what the colony remembers about you: how many
preferences you have set, with one example; the three strongest habits it has
learned, each in the habit's own words rather than under a heading; and one
line naming the last helper and what it left for the next one. Each of the
three is content, never a heading with nothing under it — a project with none
of them yet shows none of the three lines, no "nothing learned yet" filler.
"Strongest" means highest confidence, not most recent, and the card and the
project dashboard read that ranking from the one shared function, so the two
can never disagree about which habits are strongest. Every other closing
message this program prints — including the one `aether continue` shows — is
byte-identical to before this was added.

Locked by `TestGreetingCarriesPreferencesHabitsAndTheLastNote`,
`TestGreetingOmitsWhatTheProjectDoesNotHaveYet`,
`TestGreetingStaysSilentWithoutAProject`,
`TestClosingCardsAreUnchangedByTheGreetingBlock` (every other closing card is
untouched), `TestGreetingWithMemoryStillDoesNotMutate` (the two new readers
that now also look at the hub still write nothing), and
`TestStrongestHabitsHaveOneRankingRule` (one ranking rule, shared with the
project dashboard, and nothing can add a second one without this test
catching it by name).

---

## Wisdom Pipeline

The Wisdom Pipeline is the core learning loop of Aether. Colony work produces
observations that flow through the system and become reusable wisdom.

### Pipeline Stages

| Stage | Subcommand | What actually calls it now (proof) |
|-------|-----------|--------|
| 1. Observe | `memory-capture "learning"` | A build worker's own sentence, and a check's reviewer lessons/weak spots, are captured automatically on every build and check lane (`TestBuildWorkerLessonsBecomeObservations`, `TestCheckWorkerLessonsBecomeObservationsOnBothLanes`) |
| 1a. Trust score | `trust-score-compute` | Assigns weighted trust score to observation (40/35/25, 7 tiers) (`TestTrustScoreCompute`) |
| 1b. Event bus | `event-bus-publish` | Publishes scored event to JSONL event bus with TTL (`TestEventBusPublishCreatesEvent`) |
| 2. Auto-promote | phase-end consolidation (`runPhaseEndConsolidation`) | Runs automatically at the end of every check, after two observations of the same pattern (`TestWorkerLessonBecomesQueenFileWisdom`) |
| 3. Instinct | phase-end consolidation | The same automatic pass stores the promoted lesson with its provenance (`TestWorkerLessonBecomesQueenFileWisdom`) |
| 4. QUEEN.md | phase-end consolidation, gated on genuine use | Writes to QUEEN.md's Instincts section only once the lesson has actually been handed to a helper and used (`TestWorkerLessonBecomesQueenFileWisdom`, `TestQueenPromotionNeverHappensWithoutRecordedUse`) |
| 5. Inject | `colony-prime` prompt_section | QUEEN.md wisdom + instincts injected into worker context (`TestColonyPrimeWithInstincts`) |
| 6. Hive store | phase-end promotion, and seal promotion | A strong lesson (confidence >= 0.8) reaches the shared cross-project store at the end of every check, not only at project close, under the same on/off switch (`TestStrongInstinctReachesTheSharedStoreAtCheck`, `TestHivePromotionAtCheckHonoursThePolicySwitch`) |
| 7. Hive read | `hive-read` | Retrieves cross-colony wisdom scoped by domain, injected into every build/continue worker via colony-prime (`TestColonyPrimeWithHiveWisdom`) and, since 198.2, shown to the research helper too — the same top-5 selection, no research-specific limit (`TestResearchHelperIsShownTheSharedLessons`, `TestResearchHelperSeesTheSameSelectionAsEveryoneElse`) |

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

## Live Colony, Swarm, and Oracle (v1.28)

Phase 202 restored three owner-visible screens on top of one shared, durable
event trail -- the live colony view (`aether watch`), Swarm's four-lens bug
diagnosis (`/ant-swarm`), and Oracle's iterative research loop (`/ant-oracle`).
Nothing rendered by any of the three is invented; each fact traces to
something the colony actually recorded.

**Live colony view.** `aether watch` shows exactly one of three screens,
chosen by the runtime alone from what actually happened -- the wrapper you
type the command into never decides: a live dashboard that updates in place
while something is running, a replay of the most recent run (what ran, how it
ended, what it cost) when nothing is running but something has run before,
and an honest empty card when nothing has ever run. The live screen keeps the
current wave's workers in depth and compresses the rest of the colony to one
line, with a short ticker strip at the bottom naming the last few things that
happened. Locked by `TestEveryLiveEventGoesThroughOneBoundary` (one event
trail feeds every screen), `TestOneLiveEventModelOnly` (no second, competing
event system), `TestWatchResolvesThreeBranchesFromEvidenceAlone` (the
three-screen choice comes from evidence alone), `TestWatchIdle199NoFakeLiveness`
(the empty card never invents activity), `TestLiveDashboardShowsCurrentWaveInDepth`
(current wave in depth, the rest compressed to one line), and
`TestSwarmCardUsesSharedCasteIdentity` (the caste emoji, color, and name
system already built for other screens is reused here, not duplicated).

**Swarm's four-lens diagnosis.** `/ant-swarm` on a bug sends four genuinely
different investigators at it in the same wave -- one traces the bug through
the project's git history, one searches the working code for the same
pattern elsewhere, one traces the actual error, and one researches outside
sources -- then renders one comparison card naming where they agree, where
they disagree, and which fix ranks highest with its evidence. The top fix
then applies automatically through the same save-a-checkpoint,
verify-it-worked, roll-back-if-it-failed safety net every other repair path
in the colony already uses -- no approval prompt mid-flight, just the full
story afterward. If the same problem survives three repair attempts,
automatic repair stops and a plain-language case goes to the owner instead.
Locked by `TestSwarmInvestigationRunsFourLenses` and
`TestFourSwarmLensesProduceDistinctEvidence` (the four investigators are
genuinely distinct, not four copies of one prompt),
`TestSwarmComparisonSurfacesSharedCausesAndContradictions` (the comparison
card), `TestSwarmRepairCheckpointsBeforeTheFixWave` and
`TestSwarmRepairRollsBackOnFailedVerification` (the fix never runs before its
own checkpoint, and a failed fix always rolls back), and
`TestThirdStrikeRendersAnArchitecturalCase` (the three-strike escalation
survives unchanged). Finished Swarm runs are kept, not deleted -- removal
requires the run's exact identifier and digest, never a loose prefix match,
proven by `TestSwarmRemovalRequiresIdentifierAndDigest`.

**Oracle's iterative research.** `/ant-oracle` clarifies the actual question
exactly once before any research round runs, then works autonomously --
rounds, confidence, and any contradictions found are all visible live in the
colony view the same way any other worker is, with no per-round check-ins.
Depth picks from the same four names planning already uses -- Fast, Balanced,
Deep, Exhaustive -- each showing its own confidence target and round limit,
rather than Oracle inventing a separate dial of its own. The finished answer
leads with the actionable recommendation in plain language, states confidence
and any open questions honestly right beneath it, and keeps the full source
trail further down for anyone who wants to read it -- never sources first.
Locked by `TestOraclePresetLabelsMatchPlanningVocabulary` (the shared
four-name picker), `TestOracleRoundsReachTheLiveStream` (rounds are visible
live, not only through polling), and `TestSynthesisLeadsWithTheRecommendation`
and `TestSynthesisStatesConfidenceInOrdinaryWords` (the answer leads with the
recommendation, never the sources).

**Shared standing vocabulary.** Work that is useful but not yet verified --
Swarm's unproven repair ideas, Oracle's partial research, plan research, and
Dreams notes -- is now labelled with the same honest words everywhere it
appears, naming what would make it verified, rather than three different
subsystems inventing their own wording for the same idea. Locked by
`TestOneStandingVocabularyAcrossSubsystems`.

**Recovery stays part of the run it happened inside.** A recovery decision
taken while a build or check is actively running is recorded on that same
run's own live screen -- the workers already shown keep running, with the
recovery decision shown alongside them, rather than the screen jumping to a
separate, near-empty recovery view of its own. Locked by
`TestRecoveryDecisionKeepsTheBuildEpisodeLive` (the decision lands on the
real open episode it happened inside) and
`TestWatchFollowsTheMostRecentlyStartedOpenEpisode` (the live view keeps
following the most recently started still-open episode, never whichever
episode merely owns the newest single event).

**A running research round is genuinely shown as running.** While Oracle is
actively iterating on a question, the live view shows it as running --
exactly like any other worker -- and finishing the run or stopping it by
hand ends that live view honestly. A research round whose background
process has died is never shown as still running, even in a colony where an
earlier build has already finished. Locked by `TestOracleRoundIsLiveWhileItRuns`
(a round genuinely in flight resolves live), `TestOracleEpisodeBoundaryIsWiredIntoTheLoop`
(the live boundary is provably called by the research loop itself, not
merely defined somewhere unused), and `TestAbandonedOracleRoundIsNotLive` (a
round whose controller process is gone, or whose state has moved on, is
never shown as still running).

*For dummies: type `aether watch` and you get one truthful screen -- what's
happening now, what just finished, or an honest "nothing yet" -- never a
made-up status. Ask Swarm to chase down a bug and four different kinds of
investigation run at once, then you get one card explaining which fix won and
why, applied safely with an automatic undo if it doesn't work, and a plain
warning if the same bug beats three attempts in a row. Ask Oracle a question
and it asks you one round of clarifying questions, then researches on its own
and hands back an answer that leads with what to do, not a wall of sources.
And anything not yet proven -- a repair idea, a half-finished bit of research
-- always says so in the same words, everywhere you see it. A recovery
decision made mid-build no longer jumps you to a separate screen, and a
research round genuinely shows as running while it works and stops looking
that way the moment it finishes, is stopped, or its background process
dies.*

---

## Classic Visual Voice (v1.28)

Phase 202.1 restored the Classic look on the ordinary screens an owner
actually reads by default — not just the banner at the top, but every line
beneath it. `aether plan`, `aether discuss`, `aether spec`, `aether status`,
`aether build`, `aether continue`, `aether seal`, and the what-next card that
closes every command now open each content line with a small symbol naming
what kind of line it is (a goal, a phase, a warning, a finished task...), the
same symbol system the live colony view (`aether watch`) already used.
Nothing here is a one-time paint job: every one of the eight ordinary screens
is registered into one shared list (the "voice corpus"), and a named check
fails if a future screen is ever added without carrying the look, or if an
existing screen's look quietly fades back to plain text.

**The corpus knows which screens are supposed to be voiced, and notices if
one goes missing.** Eight screen families are named once; a fresh screen
family — say a ninth ordinary lifecycle screen added later — that is never
registered is caught by name, not discovered later by an owner. Locked by
`TestEveryOrdinaryScreenIsMeasuredForVoice`.

**Every registered screen is measured, not eyeballed.** A number derived from
the actual Classic-era screens (a "reference figure" computed from real
February 2026 source, never typed in by hand) sets the bar every current
screen must clear: the proportion of its own lines that carry a leading
symbol. A screen that decorates one line in fifty and calls it done still
fails. Locked by `TestEveryVoicedScreenMeetsTheReferenceDensity`.

**No screen shows the program's own internal bookkeeping where a sentence
belongs.** An internal state name (like `between_commands_boundary`) or a raw
`key=value` pair is never printed to the owner; a value like that is
translated into an ordinary sentence at the moment it becomes text. Locked by
`TestVoicedScreensCarryNoRawStateToken`.

**Every screen still speaks plain English.** A word this project invented —
"colony" (this project), "caste" (a kind of helper), "pheromone" (a steering
note), and so on — is never printed without explaining it, in the same
sentence, the first time it appears on a given screen. Locked by
`TestVoicedScreensSpeakPlainEnglish`.

**There is exactly one table of symbols, never two.** Every screen reads its
symbols from the same shared table the live colony view already used — a
structural check refuses a second, hand-rolled symbol table anywhere in the
program, so two screens can never quietly disagree about what a symbol means.
Locked by `TestVoiceGlyphsHaveOneTable`.

**A saved or resumed session never shows a raw internal code where a
sentence belongs.** Pausing or resuming a project used to be able to print an
internal code plus a private session ID straight onto the screen read next;
that class of mistake is now closed at its one true source (the writer, not
the reader), and a structural check refuses any future code from bypassing
that source. Locked by `TestLifecycleEventSentenceTypeCannotBeBypassed`.

*For dummies: every screen you actually look at day to day — planning,
asking a question, checking status, building, finishing a phase, sealing a
project — now carries the Classic look back: a little symbol at the start of
each line telling you what kind of information it is, worded in plain
English, never a raw code where a sentence belongs. And it can't quietly fade
away again — six separate checks fail the moment any of that stops being
true.*

---

## Biological Runtime (v1.28, Phase 203)

A helper working on a piece of the job can now ask the program for backup,
instead of only the instructions describing that ability. The program itself
decides whether the request is granted, through the exact same one gate
every ordinary helper assignment already goes through (`TestOneAdmissionAuthority`,
`TestRecruitmentTracerEndToEnd`). The check looks at how deep the chain of
asks has already gone (a hard cap of two hops, `TestSpawnCanSpawnDeniesPastDepthCap`),
how many helpers the whole run has already used in total
(`TestRecruitmentAdmissionCostAllowsWhenWithinRemaining`), whether the new
helper is even allowed near what it says it needs to touch
(`TestRecruitmentAdmissionPermissionDeniesReadOnlyCaste`,
`TestRecruitmentAdmissionPathDeniesOutsideColonyRoot`), and whether someone
is already doing that exact job
(`TestRecruitmentAdmissionDuplicateDeniesPendingSameSubtree`).

Every dispatched helper is told the ability exists, on every lane: the
menu-command build lane you type into, the direct build lane autopilot uses,
the check lane's reviewers, and all three assistant platforms' own helper
definitions
(`TestEveryDispatchedWorkerIsToldHowToAskForHelp`,
`TestNativeBuildLaneWorkerIsToldHowToAskForHelp`,
`TestContinueLaneWorkersAreToldHowToAskForHelp`,
`TestCodexAgentDefinitionsCarryTheRecruitInvitation`), from exactly one
source so the lanes cannot drift apart (`TestTheRecruitInstructionHasOneSource`).
Two exceptions, and they are deliberate: the security reviewer and the
quality reviewer hold no shell at all, so they are never told to run any
command — an earlier attempt to do that with a different command blocked a
phase outright (`TestReviewSpecsDoNotInstructBashlessCastes`). Both lanes
reach the same verdict on the same request, including the four checks the
autopilot lane used to skip (`TestBothLanesUseOneReasonVocabulary`).

A "no" never stops the work: the helper carries on and finishes the task
alone, and the command itself still reports success, not a failure
(`TestRecruitmentTracerEndToEnd`). The owner never approves a routine backup
request — the program's own limits are the leash, not an approval prompt.
What the owner does see, live, in the one window they are already using,
never a second screen: one line the moment a helper joins
(`TestInlineRecruitLine`), one line the moment a request is refused
(`TestInlineRefusalLine`), and — once the run ends — the whole family tree of
who asked for backup, what each branch cost, and every refusal along the way
(`TestRecruitmentFamilyTree`, `TestFamilyTreeAndCostBlockAgree`).

The steering notes this program leaves for itself (short reminders like
"focus here" or "never do this again") now get more trusted the more they
turn out to actually help, and less trusted — or shelved entirely — the more
they turn out not to, based on what a note genuinely did afterward, never
merely on whether a helper saw it
(`TestNoteStrengthTuningHelpfulNeutralHarmfulMovements`,
`TestNoteStrengthTuningQuarantinesOnHarmfulThreshold`). A note the owner
pinned in place is never moved by this automatic tuning, in either direction
(`TestPinnedNoteIsNeverTuned`).

None of this costs anything on an ordinary run that never asks for backup:
the check that would decide a request, the file that would record one, and
the pass that tunes notes afterward all measurably do nothing when there is
nothing to do (`TestNoRecruitmentPathCostsNothing`, `TestNoNewMandatoryStep`,
`TestTuningPassIsFreeWithoutCredit`).

**One honest limit, left open rather than hidden.** The depth check above
trusts a short, fixed list of coordinator names on its own word alone, with
nothing yet proving that a caller claiming one of those names really is the
coordinator. This gap predates this phase and is tracked, not silently
fixed, here (`.planning/WINDOWS.md`).

*For dummies: a helper that gets stuck can now ask the program for backup,
and the program — never the assistant — decides yes or no, using real
limits: how deep the chain of asks has gone, how many helpers the whole job
has already used, whether the new helper is allowed near what it wants, and
whether someone is already on it. A "no" never stops the work — the helper
just carries on alone — and either way you see it happen live, right in the
one window you're already using, plus a full family tree with costs once the
run finishes. The short reminders the program leaves for itself get more
trusted the more they genuinely help and less trusted (or set aside) the
more they don't — unless you pinned one yourself, which nothing else can
move. And none of this slows down an ordinary run where nobody asks for
backup — that has been measured, not just promised. One gap is recorded
rather than hidden: the depth check currently takes a short list of
coordinator names on trust, with no way yet to prove a caller really is
who it claims to be.*

---

## Learning Governor (v1.28, Phase 204)

The evidence-gated outcome ledger the previous phase built (a record of
whether a suggestion actually helped, did nothing, or made things worse) now
has a real production writer, reached from both places the program checks
its own work, and it earns credit only when a real decision changed and a
real effect was measured afterward — never merely because a note was
delivered or a phase happened to pass (`TestPhaseApplicationCreditTracerEndToEnd`,
`TestPhaseApplicationCreditIsReachedFromBothCheckLanes`, `TestCreditRequiresBothFacts`).

A lesson the program recorded but never checked is no longer shown to a
helper under a heading that calls it proven. One shared rule now decides, in
exactly one place, what counts as verified everywhere the program makes that
claim (`TestHypothesisIsNeverRenderedAsVerified`, `TestOneLearningStatusVocabulary`,
`TestAutopilotLessonsRequireValidatedStatus`).

A lesson recorded only as a guess is meant to be promoted to genuinely
verified automatically, at the end of every check, on both check lanes,
once the program's own records show it truly helped — a corroborated,
independently-checked application, never a helper's own claim, and never
merely having been read or acted on. That gating rule is real and correctly
refuses everything short of genuine proof
(`TestActedOnIsNotEnoughToValidate`, `TestUncorroboratedClaimIsNeverValidated`,
`TestHypothesisPromoterIsReachedFromBothCheckLanes`), but the promotion
itself cannot happen in the running program yet: nothing today connects a
recorded guess to the proof that it helped, so this pass finds nothing to
promote on any real check, ever — a confirmed, openly recorded gap rather
than a silent one (`TestHypothesisPromotionNeverCrossesTheIdentifierGap`;
see WINDOWS.md entry 44, reopened 2026-09-15).

Every remembered record — an instinct (a lesson the system learned and now
reuses automatically), a logged failure, a learned entry — now carries its
own version number and says where it came from, so an old record already on
disk is read as the older shape it actually is rather than guessed at. Every
field on every one of these records is either filled in by something real or
sits on a list that names the reason it is empty and can only ever get
shorter, never longer (`TestNewRecordsCarryVersionAndLineage`,
`TestLegacyRecordsReadAsLegacy`, `TestEveryMemoryStoreFieldHasALiveWriter`,
`TestMemoryStoreFieldExceptionsOnlyShrink`, `TestRetiredFieldWithoutOwnerAgreementIsRefused`).

Every run the program does now leaves a permanent record of what it cost,
what it decided and how it ended — one that outlives the live activity
screen's own thirty-day memory, so a run from a month ago can still be
looked up after the live screen has forgotten it (`TestEpisodeOutcomeSurvivesTheLiveFeedWindow`,
`TestEveryLifecycleLaneWritesADurableOutcome`, `TestOneLiveEventModelOnly`).

Guidance — an instinct (a lesson the system learned) or a steering note
handed to a helper — now moves through nine tracked states, from merely
available to genuinely helpful or harmful, and a helper's own claim that it
consulted or acted on a piece of guidance is independently checked against
what the program can actually see for itself (which files really changed,
what decision really got made), never taken on the helper's word alone. A
lesson must have genuinely helped at least once before it is promoted into
the shared instruction file every helper reads (`TestGuidanceStateVocabularyIsClosed`,
`TestCorroboratedClaimBecomesConsulted`, `TestCorroborationNeverReadsTheWorkersOwnText`,
`TestPromotionRequiresAHelpfulApplication`).

This project's own confirmed failures are now a versioned bank of regression
fixtures, and every entry traces back to a real incident this project
already agreed was real — never a plausible-looking guess. Every entry is
either guarded by a named, real check or sits on a counted list of fixtures
still waiting for one, and that list may only shrink (`TestSeededBankFixturesAllCiteRealProvenance`,
`TestEveryFixtureNamesItsGuardOrIsCountedUnguarded`).

The test suite now runs as seven named, budgeted gates — fast, focused,
integration, provider, overnight, race and release — each proving it
actually ran every test it found rather than quietly stopping partway
through and looking clean anyway. This closes the exact cause behind three
separate entries this project's own defect register already recorded
(`TestEvalGateVocabularyMatchesTheManifest`, `TestTruncatedGateRunFails`,
`TestEverySentinelStillExists`).

A proposed change to the program's own settings or routing can now be tried
side by side with its current behaviour, graded by a judge the proposal
itself structurally cannot reach, edit, or swap out. A change that only
looks better on the work it was allowed to see — and would actually fail on
work it was kept from seeing — is named as exactly that, never reported as
an improvement (`TestCandidateStoreIsAppendOnlyAndRefusesEdits`,
`TestComparisonResultReachesTheDurableLedger`, `TestHoldoutResolutionHappensOnlyInTheCommandLayer`).

The judge itself is now a real grader, not a placeholder that always said
"pass" — it tells a genuinely beneficial idea apart from a harmful or
overfit one by checking it against this project's own bank of confirmed
past incidents (`TestShadowGraderDistinguishesBeneficialFromHarmful`,
`TestOverfitCandidateIsRefusedAtTheGate`), and the whole sequence — compare,
admit, and start the trial — now runs by itself, at the end of every check,
on both check lanes, with no person needing to type a command
(`TestAutomaticImprovementPassTracerEndToEnd`,
`TestAutomaticImprovementPassIsReachedFromBothCheckLanes`).

A new hand-run command, `aether improve`, lets you check on its own how well
the program's own suggestions have actually been doing, without waiting for
the next check to run: by itself it only reads and reports the two honest
figures above, changing nothing on disk, and `--declare`/`--compare` reach
the exact same mechanism this section describes, by hand, for trying one
idea deliberately (`TestNoRegisteredSubcommandIsUnreferenced`).

Exactly two kinds of thing may ever be changed this way — which facts the
program remembers about a project, and how it routes a task — and nine
kinds of thing may never be, no matter how confident the program becomes:
your saved preferences, a skill, a workflow, your actual source code, a
security setting, a permission, a verification step, an external action
such as sending or contacting something, or anything that deletes data. The
nine have no code path into the automatic route at all — refusing one is
not a rule the program chooses to follow, it is a route that was never
built (`TestOnlyTwoScopesAreCanaryPromotable`, `TestRetainedAuthorityCannotBecomeCanaryPromotable`,
`TestEachRetainedScopeRefusalNamesItsAuthority`).

Finally, how often the program's own suggestions genuinely helped and how
often a person had to step in and stop something are now two separate,
honestly-reported figures that can never be blended into one misleading
score (`TestTwoFiguresAreNeverCombined`). And when the same reason for
stepping in keeps recurring — the owner intervening for the same kind of
reason on three or more separate runs — the program now genuinely writes up
that case itself, on its own isolated branch, for a person to read
(`TestRepeatedInterventionProposesExactlyOneSourceChange`,
`TestAutomaticProposalReplayCreatesNoSecondBranch`); that proposal can only
ever become an ordinary, reviewable code change waiting for a person to look
at it — there is no code path anywhere that lets it approve, merge, publish
or deploy itself, on this new automatic path any more than the old,
never-yet-called one (`TestSourceProposalCannotMergePublishOrDeploy`).

*For dummies: the program's memory of what it has learned is now honest, in
a few ways. A guess it never checked is never shown to you or a helper as
"proven" — one clear rule decides that everywhere. Every remembered fact —
an instinct (a lesson the system learned and reuses automatically), a note,
a logged failure — now says its own version and where it came from, and
nothing is silently missing without a reason written down for it. Every job
the program runs leaves a permanent receipt of what it cost and how it
ended, even after the live activity screen has forgotten it a month later.
A confirmed bug becomes a permanent test so the same mistake can't quietly
happen twice. Test runs are now split into seven named, timed batches that
each prove they ran everything they were supposed to, closing three
separate cases of a "clean-looking" test run that had actually stopped
partway through. A proposed improvement gets tried out fairly, checked by a
judge it cannot tamper with or peek at answers from, and an improvement
that only looks good on the questions it was shown is called out by name,
not credited. Only two very safe, easy-to-undo kinds of change — which
facts the program remembers about your project, and how it routes a task to
a helper — are ever allowed to run automatically on a small, watched,
reversible trial basis, called a "canary" here. Nine other things — your
preferences, a skill, a workflow, your actual code, a security setting, a
permission, a verification step, sending or contacting something outside
the program, or deleting anything — can never be changed this way, no
matter how sure the program is; there simply is no route built for the
program to do it without you. A lesson also gets promoted from "just a
guess" to genuinely proven automatically, at the end of every check, but
only once it has actually helped someone, checked independently rather than
taken on a helper's word. A new command, `aether improve`, lets you check
how the program's own suggestions have been doing whenever you want,
without waiting for the next check. And when the program notices it keeps
needing you to step in for the same reason, it now writes up that pattern
itself as a plain-English case on its own branch for you to read — but the
most it can ever do with its own code is write up that idea for a person to
review; it can never approve, merge, publish, or apply that change itself.*

---

## The Core Insight

The system's pieces are now **connected**:
- Pheromones update context (colony-prime injects signals into worker prompts)
- A finished check and an answered worker question each leave a note the next
  helpers read -- written at the end of a check (`aether continue`) and the
  moment the owner answers a worker's question, never during a build
  (`TestFinishedPhaseLeavesANoteNamingWhatItProduced`,
  `TestAnsweredQuestionLeavesANoteCarryingTheAnswer`)
- A worker's own lesson becomes a habit the whole project reuses: from the
  worker's own sentence, through one lesson learned, to a reusable instinct,
  to the shared instruction file every helper reads
  (`TestWorkerLessonBecomesQueenFileWisdom`) -- and it never reaches that
  file without being genuinely handed to a helper and used first
  (`TestQueenPromotionNeverHappensWithoutRecordedUse`)
- The failure log steers the colony automatically: three unacknowledged
  failures of the same kind produce one "don't do this" note; two produce
  none (`TestThreeFailuresOfOneKindProduceOneRedirect`,
  `TestTwoFailuresProduceNoRedirect`)
- Hive Brain crosses colony boundaries (domain-scoped wisdom -> colony-prime)
- A strong-enough lesson reaches every other project on the machine at the
  end of every check, not only when a project is formally finished, under
  the same on/off switch either way
  (`TestStrongInstinctReachesTheSharedStoreAtCheck`,
  `TestHivePromotionAtCheckHonoursThePolicySwitch`)
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

*Updated for Aether v1.0.83 — 2026-09-21*
