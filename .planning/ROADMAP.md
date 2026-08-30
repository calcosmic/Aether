# Roadmap: Aether

## Milestones

- ✅ **v1.0 MVP** — Phases 1-6 (shipped)
- ✅ **v1.1 Trusted Context** — Phases 7-11 (shipped)
- ✅ **v1.2 Live Dispatch Truth and Recovery** — Phases 12-16 (shipped 2026-04-21)
- ✅ **v1.3 Visual Truth and Core Hardening** — Phases 17-24 (shipped 2026-04-21)
- ✅ **v1.4 Self-Healing Colony** — Phases 25-30 (completed 2026-04-21)
- ✅ **v1.5 Runtime Truth Recovery** — Phases 31-38 (completed 2026-04-23, product v1.0.20)
- ✅ **v1.6 Release Pipeline Integrity** — Phases 39-46 (completed 2026-04-24)
- ✅ **v1.7 Planning Pipeline Recovery** — Phases 47-48 (completed 2026-04-24)
- ✅ **v1.8 Colony Recovery** — Phases 49-51 (shipped 2026-04-25)
- ✅ **v1.9 Review Persistence** — Phases 52-56 (shipped 2026-04-26)
- ✅ **v1.10 Colony Polish** — Phases 57-69 (shipped 2026-04-28)
- ✅ **v1.11 Aether Unification** — Phases 70-79 (shipped 2026-04-30)
- ✅ **v1.12 Safe Colony** — Phases 80-87 (shipped 2026-05-01)
- ✅ **v1.13 Recovery Hardening & Hive Learning** — Phases 88-92 (shipped 2026-05-03)
- ✅ **v1.14 Queen Authority** — Phases 93-99 (shipped 2026-05-04)
- ✅ **v1.15 Framework Coherence, Efficiency, and Ship Readiness** — Phases 100-105 (shipped 2026-05-08)
- ✅ **v1.16 Hybrid Runtime Boundary and Orchestration Recovery** — Phases 106-111 (shipped 2026-05-13)
- ✅ **v1.17 Classic Restoration** — Phases 112-118 (shipped 2026-05-14) — [Archive](milestones/v1.17-ROADMAP.md)
- ✅ **v1.18 Hybrid Runtime Parity & Release Gate** — Phases 119-123 (shipped 2026-05-14) — [Archive](milestones/v1.18-ROADMAP.md)
- ✅ **v1.19 TypeScript Host Cutover + Oracle Confidence Recovery** — Phases 124-129 (shipped 2026-05-15) — [Archive](milestones/v1.19-ROADMAP.md)
- ✅ **v1.20 Host Contract Hardening and Wrapper Reality Check** — Phases 130-135 (shipped 2026-05-15) — [Archive](milestones/v1.20-ROADMAP.md)
- ✅ **v1.21 Live Colony** — Phases 136-140 (shipped 2026-05-18) — [Archive](milestones/v1.21-ROADMAP.md)
- ✅ **v1.22 Grounded Planning + Ceremony Restore** — Phases 141-144 (shipped 2026-05-19) — [Archive](milestones/v1.22-ROADMAP.md)
- ✅ **v1.23 Daily Driver Reliability** — Phases 145-151 (shipped 2026-05-21) — [Archive](milestones/v1.23-ROADMAP.md)
- ✅ **v1.24 Hybrid Architecture Salvage** — Phases 152-159 (shipped 2026-05-24) — [Archive](milestones/v1.24-ROADMAP.md)
- ⏹ **v1.25 Switch It On** — Phases 160-171 (SUPERSEDED by v1.26 at 24%, 2026-08-13; phase record in `milestones/v1.25-phases/`, roadmap text preserved inside the v1.26 archive)
- ✅ **v1.26 Intelligent Orchestration** — Phases 172-191.1 (shipped 2026-08-22, product v1.0.63, override close) — [Archive](milestones/v1.26-ROADMAP.md)
- 🚧 **v1.27 The Queen Decides, the Program Checks** — Phases 193-199 (in progress, started 2026-08-22)

## Phases

### v1.27 The Queen Decides, the Program Checks

**Goal:** Aether stops burning tokens on workers nobody needed — the model's judgement picks the team and says why, the program's free checks are the floor that never moves, each phase is verified once, the cost is printed, and the owner can see every decision — so a one-task bug fix costs one worker plus checks.

- [x] **Phase 193: Free Checks Are the Floor** - The program's own checks run on every phase and can never be skipped, so work can safely finish without a human-like reviewer. (completed 2026-08-22)
- [x] **Phase 194: The Queen Decides the Team** - Which AI helpers get sent becomes judgement, not a fixed rule — a one-task fix costs one worker, not eight. (completed 2026-08-26)
- [x] **Phase 195: Coherent Jobs** - Related tasks bundle into one job for one worker instead of one worker per task. (completed 2026-08-27)
- [x] **Phase 196: See What It Cost** - Every run ends with one honest cost line, and model choices carry a reason. (completed 2026-08-28)
- [x] **Phase 197: One Answer to "What Next?"** - Every command ends by saying exactly what to do next, from one shared source of truth. (completed 2026-08-29)
- [x] **Phase 198: Put the Thrown-Away Data Back on Screen** - The detail the older version used to show comes back, in the chat view too. (completed 2026-08-29)
- [x] **Phase 198.1: Feed the Memory** - Builds and checks write down what they learned and what went wrong again, automatically, so the learning loop stops running on empty. (completed 2026-08-30)
- [ ] **Phase 198.2: Memory Reaches Every Helper** - Planning, colonizing, research and the welcome card get the same memory the builder gets; the codebase map is delivered as content, not filenames.
- [ ] **Phase 198.3: Overnight Stamina** - Autopilot queues what needs your eyes instead of stopping, restores the old stop rules, and can run unattended for a night.
- [ ] **Phase 198.4: Prune the Dead Wood** - Every command nothing calls and every file nothing reads is wired or deleted, with a ratchet so it cannot grow back.
- [ ] **Phase 199: Proof** - A real small task timed against plain Claude, a real overnight run, and a fresh-chat resume prove it is the go-to framework.

## Phase Details

### Phase 193: Free Checks Are the Floor

**Goal**: The program's own automatic checks — does the code build, do the types check, does the linter pass, do the tests pass, do the files a worker claims to have created actually exist, and does each requirement have real evidence — run on every phase and can never be skipped. These checks become the safety floor, so a phase can safely finish even when no AI reviewer ("caste" — a type of AI helper with one job) was sent to check it by hand.

**Depends on**: Nothing (first phase of this milestone)

**Requirements**: FLOOR-01, FLOOR-02, FLOOR-03, FLOOR-04

**Success Criteria** (what must be TRUE):

1. Turning off every optional reviewer (through any speed setting, review policy, or the "skip watchers" flag) still leaves the build/types/lint/tests/files-exist/evidence checks running — a test walks every possible skip path and fails if even one lets a check through unrun (`TestDeterministicChecksCannotBeSkipped`).
2. On a test project with zero reviewer helpers sent, the phase moves forward when the automatic checks pass and is stopped when they fail — proven in both directions through the `continue` command on a fixture project.
3. A requirement that today silently expects a "Watcher" (the reviewer caste that checks work) to have run is satisfied by the automatic checks alone when no Watcher was sent, and running the manual "I already checked this by hand" command (`continue-finalize --reconcile-task`) counts as real proof (`TestGateAcceptsDeterministicEvidenceWithoutReviewer`).
4. A real build followed by a real `continue` (the command that checks work and decides whether to move to the next phase) proves no single AI helper type is sent to check the same phase twice unless the Queen (the coordinator that decides which helpers to send) explicitly asks for that (`TestPhaseVerifiedOnce`).

**Plans**: 5 plans

Plans:
**Wave 1**

- [x] 193-01-PLAN.md — Tracer: a phase with no reviewer is still checked, and the checks decide (wave 1)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 193-02-PLAN.md — The build side runs free checks only; verified once (wave 2)
- [x] 193-03-PLAN.md — Nothing can turn a check off, and the flag text says so (wave 2)

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 193-04-PLAN.md — No gate demands a reviewer: reconciliation, re-run evidence, owner confirmation (wave 3)

**Wave 4** *(blocked on Wave 3 completion)*

- [x] 193-05-PLAN.md — Targeted per phase, full at the end; one bounded fix attempt (wave 4)

### Phase 194: The Queen Decides the Team

**Goal**: Which AI helpers ("workers") get sent to do a job becomes the Queen's judgement call, not a fixed rule. A reviewer worker is only forced onto a job when it touches something genuinely risky — passwords, payments, deleting data, migrations, or a release — and the program says why in plain words. A one-task bug fix costs one worker plus the free checks from Phase 193, not eight.

**Depends on**: Phase 193 (the free checks have to be the safety net in place before workers can be trusted to skip)

**Requirements**: TEAM-01, TEAM-02, TEAM-03, TEAM-04, TEAM-05

**Success Criteria** (what must be TRUE):

1. On an ordinary job, the only helper required by default is the one that writes the code (the "builder" caste); no other helper type is required just because of guessed project type or keyword — the old rule that always sent a reviewer (`TestWatcherIsAlwaysRequiredOnBuild`) is formally retired.
2. A side-by-side test shows a plain CSV-export feature gets no forced reviewer, while a password-reset feature gets a security reviewer, with the reason ("this touches credentials") shown on screen (`TestReviewerForcedOnlyByNamedRisk`).
3. Every worker that gets sent carries a one-sentence, plain-English reason the owner can read; if a proposed team includes a worker with no reason, that worker is rejected by name, not silently allowed (`TestNoWorkerWithoutStatedReason`).
4. A sample one-task bug fix — measured today at 8 workers, versus 3-4 back in the older (v5.4.0) version of Aether — is sent one worker plus the free checks, whether the team came from the assistant's own proposal or the automatic fallback (`TestOneTaskBugFixIsOneWorkerPlusChecks`).
5. The owner's manual controls — picking helpers by name, asking for "heavy" or "light" review, and the pre-build "here's who I'm sending, OK?" check-in card — all still work after this change.

**Plans**: 9 plans

*Every plan depends on the one before it. That is deliberate, not an oversight:
`queen_spawn_budget.go`, `caste_relevance.go`, `queen_risk_signals.go`,
`codex_continue.go`, `codex_build.go` and `ceremony_team_checkin.go` are each
edited by three or more plans, so running any two side by side would collide in
the same file. Each plan names its own shared files in its objective.*

Plans:
**Wave 1**

- [x] 194-01-PLAN.md — Tracer: named risk forces one reviewer, recorded at build and dispatched once at the checking step (wave 1)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 194-02-PLAN.md — The build floor shrinks to the worker that writes the code; every test asserting a deleted rule is reconciled or retired (wave 2)

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 194-03-PLAN.md — A reason for every worker; a worker named without one is refused by name (wave 3)

**Wave 4** *(blocked on Wave 3 completion)*

- [x] 194-04-PLAN.md — The card carries per-phase reasons only, names the signal, and REQUIRED stops meaning anything else (wave 4)

**Wave 5** *(blocked on Wave 4 completion)*

- [x] 194-05-PLAN.md — The automatic team is retuned to the same floor; the owner's three dials mean what they say (wave 5)

**Wave 6** *(blocked on Wave 5 completion)*

- [x] 194-06-PLAN.md — Both checking-step lanes carry the same team: changed files can raise a reviewer, and the owner's list works on the path people run (wave 6)

**Wave 7** *(blocked on Wave 6 completion)*

- [x] 194-07-PLAN.md — The waiver: owner-only, one signal, one phase, recorded with a reason — plus the owner check-in on the card (wave 7)

**Wave 8** *(blocked on Wave 7 completion)*

- [x] 194-08-PLAN.md — The one-task bug fix proved at one worker; the double-dispatch window closed on a passing test (wave 8)

**Wave 9** *(blocked on Wave 8 completion)*

- [x] 194-09-PLAN.md — Every stale document corrected: CLAUDE.md, the six wrapper files, the UX contract, the folded todo (wave 9)

### Phase 195: Coherent Jobs

**Goal**: The Queen can bundle several related tasks into one job for one worker, instead of sending a separate worker per task. Related tasks (same files, or one depends on another) get grouped by default even with no explicit instruction — so, for example, six near-identical file-copy steps become one job instead of six.

**Depends on**: Phase 194 (grouping tasks into jobs only makes sense once the Queen, not a fixed keyword rule, is deciding the team)

**Requirements**: JOBS-01, JOBS-02, JOBS-03, JOBS-04

**Success Criteria** (what must be TRUE):

1. The Queen can combine multiple tasks into one job with a stated reason; if a proposed grouping would do a task before something it depends on, it is rejected by name instead of silently going through.
2. A combined job's instructions carry every task's requirements, and finishing the job marks every one of those tasks as done, not just the first (`TestMergedDispatchCreditsEveryCoveredTask`).
3. Given the real six-batch file-copy failure from a past project as a test fixture, the default grouping (with no explicit instruction) produces one worker instead of six, because the tasks share files and depend on each other.
4. The same grouping behavior works when workers run in their own isolated copy of the code ("worktree" mode) — grouping is no longer switched off there.

**Plans**: 10 plans

Plans:
**Wave 1**

- [x] 195-01-PLAN.md — Pure coherent-job contracts, dependency validation, local repair, and CalVault component planning (wave 1)
- [x] 195-02-PLAN.md — Additive task-receipt contract shared by native/external results and the generated schema (wave 1)

**Wave 2** *(blocked on job-planner contracts)*

- [x] 195-03-PLAN.md — Wire grouping before waves/worktree ownership; prove the in-repo CalVault job and grouped briefs (wave 2)

**Wave 3** *(parallel runtime integrations after manifest contracts)*

- [x] 195-04-PLAN.md — Shared receipt validator and exact native four-of-six task credit (wave 3)
- [x] 195-05-PLAN.md — Decision-aware one-worker check-in fast path, compact summary, and `--checkin` override (wave 3)

**Wave 4** *(blocked on native receipt semantics)*

- [x] 195-06-PLAN.md — External/wrapper receipt parity and exact partial finalization (wave 4)

**Wave 5** *(blocked on both completion lanes)*

- [x] 195-07-PLAN.md — Append-only, parent-linked, unfinished-only recovery jobs (wave 5)

**Wave 6** *(blocked on retry and receipt truth)*

- [x] 195-08-PLAN.md — One grouped worktree plus receipt-scoped partial sync before credit (wave 6)

**Wave 7** *(blocked on final runtime behavior)*

- [x] 195-09-PLAN.md — Atomically synchronize canonical YAML, Codex guide/skill, and all three build wrappers (wave 7)

**Wave 8** *(blocked on parity-critical contract synchronization)*

- [x] 195-10-PLAN.md — Update owner docs, close the folded todo, and run all phase-wide gates (wave 8)

### Phase 196: See What It Cost

**Goal**: Every build (running a phase) and continue (checking and advancing it) ends with one honest, plain-English line stating how many tokens (the unit AI usage is billed in) it cost — never a dollar figure as the headline. The team card that shows who is being sent also shows which AI model each one uses and why, so nothing quietly uses an expensive model with no reason on record.

**Depends on**: Phase 194 (there is nothing coherent to put a cost on until the Queen, not a keyword engine, is choosing the team)

**Requirements**: COST-01, COST-02, COST-03, COST-04, COST-05

**Success Criteria** (what must be TRUE):

1. Every build and continue ends with one line stating token usage per worker and in total, showing no number at all for a worker whose tool reported none and counting only the measured workers in the total — no dollar amount as the headline, no price table anywhere (`TestBuildEndsWithOneCostLine`).

   *Corrected 2026-08-28 (plan 196-08). This criterion previously said "clearly marking which numbers are measured versus estimated". D-01 as amended (owner, 2026-08-27, in `phases/196-see-what-it-cost/196-CONTEXT.md`) abolished the estimated figure entirely: guessing a worker's usage from the length of its prompt was the only estimate mechanism that existed, and criterion 3 below forbids exactly that, so a marked estimate would have been the forbidden thing wearing a label. The owner ruled: show no number at all. The named test and the no-dollar clause are unchanged.*

2. On the Claude Code / OpenCode chat path — the one the owner actually uses day to day — worker results report their real token usage, so that cost line has real numbers instead of blanks.
3. Running `aether spend` shows token usage per worker for the current run without changing any files on disk, and none of its numbers are guessed from text length.
4. The pre-build team card names each worker's AI model and the reason for that choice; routine roles (the documentation writer, the knowledge-keeper, the accessibility checker) no longer silently default ("inherit") to whatever model was last used, and any worker kept on the expensive model has a written reason (`TestRoutineBuilderIsSonnetNeverInherit`, `TestOpusRequiresRecordedReason`).
5. Three abandoned branches of half-finished cost-tracking work sitting in the codebase are each reviewed and either merged in (with a written reason) or deleted (with a written reason) — none left dangling.

**Plans**: 8 plans

Plans:
**Wave 1** *(independent foundations — no shared files)*

- [x] 196-01-PLAN.md — Salvage the spend ledger with its four assessed fixes: no currency relay, job attribution, independent aggregation invariant, one status vocabulary (wave 1)
- [x] 196-02-PLAN.md — One authoritative token type: every billed column on the chat-model client, both accounting paths locked to one total, the invented length-derived figure deleted at source, and a ratchet forbidding its return (wave 1)
- [x] 196-03-PLAN.md — The Claude transcript reader, deduplicated by tool use identifier, proved by a real fixture that keeps its duplicates (wave 1)
- [x] 196-04-PLAN.md — Pin the routine roles to the cheaper model and show every expensive role's written reason on the team card (wave 1)

**Wave 2** *(blocked on the ledger and the transcript reader)*

- [x] 196-05-PLAN.md — Salvage the OpenCode reader inside the plan that wires it: one resolver, and a build that writes its rows (wave 2)
- [x] 196-06-PLAN.md — `aether spend`: the read-only per-worker view, proved to mutate nothing and to invent nothing (wave 2)

**Wave 3** *(blocked on rows being written and the token types reconciled)*

- [x] 196-07-PLAN.md — Continue-lane rows, then the one honest cost line ending every build and every continue (wave 3)

**Wave 4** *(phase closeout)*

- [x] 196-08-PLAN.md — Prove the path is joined up end to end, then delete the three branches with their written reasons recorded (wave 4)

### Phase 197: One Answer to "What Next?"

**Goal**: Every Aether command ends by telling the owner, in one consistent way, what just happened and exactly what to type next — instead of each command guessing or occasionally naming a command that no longer exists. Opening a new chat session in a project that already has a colony (an in-progress project Aether is tracking) greets the owner with where things stand.

**Depends on**: Phase 194 (the "next step" advice needs to reflect the Queen's real team decisions, not the old keyword logic)

**Requirements**: NEXT-01, NEXT-02, NEXT-03, NEXT-04, NEXT-05, NEXT-06

**Success Criteria** (what must be TRUE):

1. One shared piece of logic, given the project's saved state, produces everything the owner needs: where things stand, what changed, any open flags, the recommended next step, the exact command to run, 2-4 other options, whether it is safe to close the chat, and any paused/recoverable state.
2. Every command's closing message (starting, discussing, planning, building, continuing, pausing, resuming, sealing/finishing, updating, recovering, checking status) is generated from that same shared logic, and the machine-readable version carries the identical information (`TestEveryLifecycleCommandEndsWithNextAction`).
3. An automatic check counts how many places in the codebase still hand-type a command name instead of using the shared logic, and fails the build if that count ever grows from today's recorded baseline (`TestNextActionNeverHardcoded`).
4. Starting a new chat session in a project with an existing colony shows a "here's where you left off" card automatically, generated by the same shared logic (`TestSessionStartCardReflectsState`).
5. Typing the short command `/ant-pause` and the older long name `/ant-pause-colony` both work identically, and running the update command detects and fixes any project where the short version is missing (`TestCanonicalAliasDelegates`).
6. The recommended next command is never one that is not actually available in the current project — recommending a command that does not exist is a failing test case.

**Plans**: 7 plans

Plans:
**Wave 1** *(the shared logic, before anything renders from it)*

- [x] 197-01-PLAN.md — The one resolver over saved state, and the gate that makes recommending a command this project does not have impossible (wave 1)

**Wave 2** *(blocked on the resolver existing)*

- [x] 197-02-PLAN.md — Collapse the four rival next-step deciders onto the resolver, and build the one closing card every command will render (wave 2)

**Wave 3** *(blocked on the card; two independent surfaces, no shared files)*

- [x] 197-03-PLAN.md — The session-start greeting the runtime owns, the removal of the five documents asking the assistant to remember, and a reachability scan that counts hooks as callers (wave 3)
- [x] 197-04-PLAN.md — Starting, discussing, scanning, planning, building, continuing and the shared closeout all end with the card (wave 3)

**Wave 4** *(blocked on the command goldens settling)*

- [x] 197-05-PLAN.md — `/ant-pause` becomes the real command with the long name an alias declared once, and updating a project restores a missing copy out loud (wave 4)

**Wave 5** *(blocked on the short pause command existing to be recommended)*

- [x] 197-06-PLAN.md — Pausing, resuming, sealing, updating, recovering and status end with the card, and the last private decider goes (wave 5)

**Wave 6** *(phase closeout — the baseline is measured only once everything has moved)*

- [x] 197-07-PLAN.md — Criterion 2's coverage test and criterion 3's measured hardcode ratchet, both registered and running in CI (wave 6)

### Phase 198: Put the Thrown-Away Data Back on Screen

**Goal**: Restore the detail the older (v5.4.0) version of Aether used to show — which checks passed, evidence for each requirement, how long each worker took, plan confidence, resume progress, recent decisions, and warnings — that the program already calculates today but currently throws away instead of displaying. The chat view (Claude Code / OpenCode) shows the same level of detail as the direct command-line view.

**Depends on**: Phase 197 (the closing cards this phase enriches are the same ones generated by the shared "what next" logic built in Phase 197)

**Requirements**: SHOW-01, SHOW-02, SHOW-03, SHOW-04, SHOW-05

**Success Criteria** (what must be TRUE):

1. What the owner sees in the chat for planning, continuing, and sealing (finishing) a project matches what the direct command-line view shows — no more thinner "chat mode" summary (`TestWrapperPathRendersSameCeremonyAsDirectPath`).
2. Every piece of information the program already calculates is shown, not silently dropped: which checks passed, evidence for each requirement, how long each worker took, plan confidence, resume progress by phase, recent decisions, a drift warning, specialist findings, and a heads-up before build if something is blocked — checked as a rule that never lets these slip through, not a page that happens to have the right heading (`TestRenderedVisualsShowEveryCarriedField`).
3. While `continue` (the check-and-advance command) is running, the owner sees each check's progress appear live as it happens, not only a summary at the very end.
4. Sealing (marking a project finished) always asks the owner to confirm first and always runs the "what did we learn" review before finishing.
5. Three things the team deliberately chose to leave out — one shared terminal window, boxed section borders, and the old "clear the chat now?" prompt — stay out, and a test locks that choice so they do not quietly come back.

**Plans**: 9 plans

Plans:
**Wave 1**

- [x] 198-01-PLAN.md — Tracer: the chat path renders the direct continue screen from one renderer (wave 1)
- [x] 198-02-PLAN.md — Live start/finish lines for every check, on both continue lanes (wave 1)
- [x] 198-03-PLAN.md — Seal state-of-play card, wisdom review before the question, recorded finish-anyway, autopilot never seals (wave 1)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 198-04-PLAN.md — Parity expanded to planning and finishing; criterion 1 closed (wave 2)
- [x] 198-05-PLAN.md — Worker duration and tool-call count, measured, live and in the summary (wave 2)
- [x] 198-07-PLAN.md — Build-start blocker heads-up and its one question (wave 2)

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 198-06-PLAN.md — Per-check report, per-gate names, evidence lines, specialist findings (wave 3)

**Wave 4** *(blocked on Wave 3 completion)*

- [x] 198-08-PLAN.md — Resume per-phase progress, recent decisions, drift note (wave 4)
- [x] 198-09-PLAN.md — Carried-field invariant with shrink-only allowlist, plus the deliberate-drops lock (wave 4)

**UI hint**: yes

### Phase 198.1: Feed the Memory

**Goal**: The learning loop already runs at the end of every phase, but nothing on a normal build or check writes anything for it to learn from — so it runs on empty. This phase reconnects the sources: worker results become observations, failures become midden entries, verified learnings become instincts and Queen-file wisdom, phase completions and decisions leave feedback notes, and strong instincts reach the hive without waiting for a seal. Everything the 2026-08-30 audit found starved at the source is fed again, automatically, on the path the owner actually uses.

**Depends on**: Phase 198

**Requirements**: FEED-01, FEED-02, FEED-03, FEED-04, FEED-05, FEED-06

**Success Criteria** (what must be TRUE):

1. After a build and a check on a fixture project whose helpers report real findings, the observation log contains sentences those helpers wrote — with no one typing a memory command by hand.
2. A failed worker, failed build, or failed verification on build, check, quick, and swarm writes a failure record, and the very next build brief carries that failure's own wording.
3. A check that verifies real learnings creates instincts and promotes them into the Queen file through the existing consolidation; the "promoted" list is non-empty on the fixture, not hand-seeded.
4. A completed phase and an answered decision each leave a feedback note; a run of failures past the threshold leaves an automatic "don't do this" note; the CLAUDE.md sentences claiming these are true again.
5. An instinct at 0.8 confidence or above reaches the hive at the end of a check, not only at seal, under the existing hive policy switch.
6. The memory drill-down view shows wisdom entries and pending promotions again, and status shows the top instincts — the bare-number JSON is gone.

**Audit findings closed** (from `.planning/audits/2026-08-30-whole-system-audit.md`): continue observation→instinct→QUEEN promotion; continue auto-emitted pheromones; build/continue midden-write; build learning capture; instinct-create on continue; hive-promote at continue; memory-details; midden-threshold auto-REDIRECT; pheromone-expire→eternal promotion.

**Plans**: 6 plans across 3 waves

Plans:

- [x] 198.1-01-PLAN.md — wave 1: a failed build worker's own words reach the failure log and the next build brief; worker lessons become observations; one feed boundary for both build lanes
- [x] 198.1-02-PLAN.md — wave 2: reviewer lessons and failures become memory on both check lanes; a failing free check leaves a record; quick and swarm failures too
- [x] 198.1-03-PLAN.md — wave 2: record which instincts a worker was actually given, count a use when the phase passes, and let the existing consolidation carry a lesson into the Queen file
- [x] 198.1-06-PLAN.md — wave 2: the memory drill-down shows real sentences again, reports when the Queen file changed, and the status instinct list is labelled by strength
- [x] 198.1-04-PLAN.md — wave 3: a finished phase and an answered question each leave a note, three failures of one kind leave an automatic don't-do-this note, and a valuable expiring note is kept in long-term memory
- [x] 198.1-05-PLAN.md — wave 3: strong lessons reach the shared cross-project store at every check under the existing switch, and every learning claim in CLAUDE.md names a test

### Phase 198.2: Memory Reaches Every Helper

**Goal**: The memory Aether keeps — Queen-file rules, hive lessons, instincts, failures, focus notes, the owner's answers, the last helper's relay note, the colonize map, Oracle research — reaches every helper that should read it, as actual content, on the command lane the owner actually runs. Today the builder and reviewers get it; the planner, the surveyors, the research scout, and the welcome card get nothing or a list of filenames. The proof standard for this phase is one test per command that looks for a sentence from each memory source inside the prompt actually sent, and fails when any source is missing.

**Depends on**: Phase 198.1

**Requirements**: WIRE-01, WIRE-02, WIRE-03, WIRE-04, WIRE-05, WIRE-06, WIRE-07

**Success Criteria** (what must be TRUE):

1. `/ant-plan` and `/ant-colonize` from Claude and OpenCode deliver the full memory capsule to the planner and every surveyor; a table test runs each command's real delegate path and asserts a sentinel sentence from every memory source (Queen file, hive, instincts, failures, notes, clarified intent, handoffs). A test that exercises only the native lane does not count.
2. The research scout is shown the hive wisdom it is asked to summarise; the instruction to invent it is gone.
3. A condensed version of the colonize reports reaches build, plan, and research briefs within the existing budget; deleting the reports changes the prompt, and a test asserts a surveyor-authored sentence arrives.
4. A finished Oracle run registers its output for later helpers by itself; no second or third hand-typed command is needed for its findings to reach a builder.
5. The welcome card carries the owner's preferences, the top instincts, and the last handoff — each content-locked, not heading-locked.
6. No capsule section is empty by construction: "Phase Learnings" and "Key Decisions" either have a live writer or are removed, locked by an invariant that every section has a writer.
7. The previous phase's outcome, verification, and review content reaches the next phase's build brief.

**Audit findings closed**: Claude-lane plan/colonize capsule gap; phase-research scout pointer-only brief; colonize survey orphaned; Oracle workspace/research orphaned; session card near-empty; dead PhaseLearnings/Decisions sections; build/phase-N outcome/verification/review unwired; SCOUT.md pointer-only; hive-read for plan priming.

**Plans**: 8 plans (4 waves)

Plans:
**Wave 1**

- [x] 198.2-01-PLAN.md — wave 1: the memory pack reaches the planner and the surveyors on the Claude/OpenCode lane (WIRE-01)
- [x] 198.2-02-PLAN.md — wave 1: a finished or stopped deep-research run files, registers and promotes itself (WIRE-04)
- [x] 198.2-03-PLAN.md — wave 1: the session greeting carries preferences, the strongest habits and the last relay note (WIRE-05)
- [x] 198.2-04-PLAN.md — wave 1: dead memory-pack parts removed, every remaining part has a live writer (WIRE-06)
- [x] 198.2-05-PLAN.md — wave 1: both check lanes persist the phase's closing summary word-for-word (WIRE-07)

**Wave 2** *(blocked on Wave 1 completion)*

- [ ] 198.2-06-PLAN.md — wave 2: the codebase-map digest reaches the build, planning and research briefs as content (WIRE-03)

**Wave 3** *(blocked on Wave 2 completion)*

- [ ] 198.2-08-PLAN.md — wave 3: the research helper is shown the shared cross-project lessons it was being asked to invent (WIRE-02)

**Wave 4** *(blocked on Wave 3 completion)*

- [ ] 198.2-07-PLAN.md — wave 4: the previous phase's failures and closing summary reach the next build brief, under a proven growth cap (WIRE-07)

### Phase 198.3: Overnight Stamina

**Goal**: Autopilot can be left alone for a night again. The old version had ten named reasons to stop and queued the things that merely needed the owner's eyes; the current one has six and halts on nearly everything. This phase restores the stop contract, queues instead of halting, says why a replan is due, reports what the night cost, and proves a long unattended multi-phase run completes.

**Depends on**: Phase 198.2

**Requirements**: STAM-01, STAM-02, STAM-03, STAM-04, STAM-05, STAM-06

**Success Criteria** (what must be TRUE):

1. In headless mode, a phase needing hand-testing or a visual checkpoint is queued as a pending decision and the run continues; only genuine blockers halt it.
2. The named stop contract is back — quality-score floor, critical audit finding, runtime verification needed, escalated flags — and `--dry-run` lists every trigger by name before the run starts.
3. The replan pause says how many lessons were learned since the last plan, not just a phase count.
4. The run summary shows elapsed wall-clock time and the honest cost line.
5. The pre- and post-build blocker gate is live again in run and status.
6. A fixture colony of at least six phases with simulated helpers completes unattended, end to end, under a single `aether run`, and the test fails if any phase stops for a reason that should have been queued.

**Audit findings closed**: autopilot-* subcommands orphaned; flag-check-blockers orphaned; insert-phase guided flow dropped; swarm 3-strike escalation dropped; failure-classify / recovery-log orphaned; medic-auto-spawn-check orphaned; status escalated-flags count dropped.

**Plans**: TBD

### Phase 198.4: Prune the Dead Wood

**Goal**: Nothing built stays disconnected. The audit found 117 runtime commands nothing calls, 25 files written and never read, and 72 capabilities from the old version that vanished silently. Each is wired in by an earlier phase, deleted, or recorded as deliberately dropped with the owner's reason — and a ratchet locks the count so it can only fall.

**Depends on**: Phase 198.3

**Requirements**: PRUNE-01, PRUNE-02, PRUNE-03, PRUNE-04

**Success Criteria** (what must be TRUE):

1. Every runtime subcommand has a live caller (wrapper, hook, host, or Go call site on a command path) or is deleted; a ratchet test carries the allowlist of remaining orphans and fails if it grows.
2. Every file written under `.aether/data` has a live reader or is no longer written; same ratchet.
3. Every sentence in CLAUDE.md and shipped docs that describes runtime behaviour is backed by a named test or removed — starting with "decisions become pheromones", "midden affects behavior", and the event-bus wisdom stage.
4. The 72 dropped-since-5.4 capabilities are triaged in a ledger: restored (naming the phase), or dropped with a reason the owner gave; a test fails if any audit subject is missing from the ledger.

**Plans**: TBD

### Phase 199: Proof

**Goal**: Prove — with the owner watching, on a real project — that Aether is the go-to framework: as quick as plain Claude on a small task, with a memory plain Claude does not have, able to run a night unattended, and able to pick up in a fresh chat without being re-briefed. The milestone only counts as finished when the owner has felt each of these, not when a test says so.

**Depends on**: Phases 193–198.4 (proof requires the finished, reconnected system)

**Requirements**: PROOF-05, PROOF-06, PROOF-07, PROOF-08

**Success Criteria** (what must be TRUE):

1. One small real task in one of the owner's own projects is done twice — once with plain Claude, once through Aether — with wall-clock time, interventions, and worker count recorded for both, whatever they show.
2. Aether finishes that task within 1.5× plain Claude's time, with one worker plus the free checks, and leaves the project record (phase, learnings, handoff) updated where plain Claude leaves nothing.
3. A real overnight autopilot run on that project covers at least three phases unattended; its stops, if any, are all in the queued-not-halted category, and its summary shows elapsed time and cost.
4. A task interrupted mid-session resumes in a brand-new chat with the welcome card carrying enough that the owner never re-explains what was in progress.

**Plans**: TBD

<details>
<summary>✅ v1.26 Intelligent Orchestration (Phases 172-191.1) — SHIPPED 2026-08-22 (override close)</summary>

Rescoped to hardening 2026-08-14 (`HARDENING-PLAN.md`), extended by the 2026-08-17
audit addendum (Phases 186-192). Full text: `milestones/v1.26-ROADMAP.md`.

- [x] Phase 172: Wiring Proof (14/14 plans) — verified 2026-08-12
- [ ] Phase 172.1: Gate Environment Integrity — not started; blocks nothing; stays in the backlog
- [x] Phase 173: Delegation Guard (13/13 plans) — verified 2026-08-13; one human check pending (a refusal seen in a real run)
- [~] Phase 174: Spend Ledger — closed at plan 2 of 9 by owner ruling 2026-08-14 (token measurement shipped; ledger machinery cut)
- [x] Phase 175: Orchestration Visibility — shipped 2026-08-16 outside the phase system (SHIP-PROGRESS Item 3)
- [x] Phases 180-184: Hardening H1-H5 — shipped 2026-08-14/15 (v1.0.54)
- [ ] Phase 185: One Honest Cost Line — carried to v1.27 (feature 4)
- [~] Phase 186: Baseline Showdown (Light) — 6/7 plans; 186-07 (one live run, owner present) carried to v1.27 (feature 7)
- [x] Phase 187: Crash-Safe Worktrees & Ecosystem Neutrality (9 plans) — verified
- [x] Phase 188: One Truth for Failures and Advances (7 plans) — verified
- [x] Phase 189: Complete Worker Contract (3 plans) — verified
- [x] Phase 190: Lean, Non-Duplicated Delivery (6 plans) — verified
- [x] Phase 191: Dead Wood (7 plans) — verified 2026-08-21
- [x] Phase 191.1: Field Hardening (4 plans) — verified 2026-08-21
- [ ] Phase 192: Final Showdown — carried to v1.27 (feature 7); the "stop building when 192 passes" rule was superseded by the owner's ratification of the priority spec as the backlog (D1, 2026-08-21)

</details>

<details>
<summary>⏹ v1.25 Switch It On (Phases 160-171) — SUPERSEDED 2026-08-13 at 24%</summary>

Phases 160, 162, 163, 163.1, 163.2, 164, 165 shipped (record in `milestones/v1.25-phases/`);
166-171 absorbed into v1.26 or moved to Future Requirements (SEE, TYPED, RECLAIM, LOCK).
Full text preserved inside `milestones/v1.26-ROADMAP.md`.

</details>

<details>
<summary>✅ v1.17 – v1.24 — see per-milestone archives linked above</summary>

Phase details for 112-159 live in their milestone archives under `milestones/`.

</details>

<details>
<summary>✅ v1.0 – v1.16 (Phases 1-111)</summary>

Earlier milestones; summaries in `MILESTONES.md` and `PROJECT.md` history.

</details>
