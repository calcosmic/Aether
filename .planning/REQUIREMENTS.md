# Requirements: Aether — v1.27 "The Queen Decides, the Program Checks"

**Defined:** 2026-08-22
**Core Value:** Aether should feel alive and truthful at runtime, not only look clever in wrappers or tests.
**Brief:** `.planning/research/v1.27-milestone-brief.md` · **Rulings:** D11, D12 in `.planning/decisions/2026-08-21-owner-rulings-priority-spec-v3.md`

> CLAUDE.md's Definition of Done governs every line below: a requirement is
> satisfied only when a command exists that someone can run, and that command
> fails when the requirement is unmet. Prefer tests that assert a proportion or
> an invariant over tests that look for a named section. Reproduce the field
> failure first.

## v1 Requirements

### Free checks are the floor (FLOOR)

- [x] **FLOOR-01**: The program's own checks — build, types, lint, tests, "claimed files exist", "each success criterion has evidence" — run on every phase, and no depth flag, review policy, team proposal or `--skip-watchers` can skip them (a test walks every skip path and fails if any path omits a check)
- [x] **FLOOR-02**: A phase with zero reviewer workers advances when the free checks pass and is blocked when they fail (end-to-end through `continue` on a fixture colony, both directions asserted)
- [x] **FLOOR-03**: No gate demands a checker worker implicitly — a criterion bound to `required_checks: watcher` (or any reviewer caste) is satisfied by deterministic evidence when no such worker was dispatched, and `continue-finalize` counts `--reconcile-task` as recorded reconciliation (closes the 2026-08-01 todo)
- [x] **FLOOR-04**: A phase is verified once — the build-side verification stage runs free checks only, agent review lives in `continue`, and no caste is dispatched at both boundaries for the same phase unless the Queen explicitly asks (asserted over a real manifest + continue plan pair)

### The Queen decides the team (TEAM)

- [x] **TEAM-01**: The castes a build requires shrink to builder (non-discovery phases) — no caste is required by inferred mode or keyword alone; `TestWatcherIsAlwaysRequiredOnBuild` is retired through the test-deletion ledger citing D11, and the negative half of the Probe rule (no Probe where nothing is testable) stays
- [x] **TEAM-02**: The program forces a reviewer only for a named high-risk signal — credentials/auth, payments, data deletion, migration, release sign-off — and attaches the signal as a visible reason on the team card (table test: CSV export → none forced; password reset → security reviewer with reason)
- [x] **TEAM-03**: Every worker that spawns carries a one-line reason the owner can read; a proposal naming a worker without a reason is refused by name
- [x] **TEAM-04**: The chat's proposal (`--castes` + `--caste-reason`) is the primary team source; the keyword engine is the autopilot / no-proposal fallback and is retuned to the same floor — a 1-task bug fix yields one dispatch plus free checks on both paths
- [x] **TEAM-05**: The owner's overrides in both directions (`--castes`, `--heavy`, `--light`, the check-in card) keep working and stay test-locked after the floor moves

### Coherent jobs, not one worker per task (JOBS)

- [x] **JOBS-01**: The Queen can group tasks into one job with a reason; the runtime validates dependency order and refuses, by name, a grouping that violates `depends_on`
- [x] **JOBS-02**: A grouped job's brief carries every covered task's criteria and its completion credits every covered task (the existing `covered_task_ids` chain, re-proven end to end)
- [x] **JOBS-03**: Without a proposal, tasks sharing files or a dependency chain cluster into one job beyond consecutive same-caste steps — the CalVault field failure (six file-copy batches) as a fixture yields one worker
- [x] **JOBS-04**: Grouped jobs work in worktree mode (coalescing is no longer disabled there)

### See what it cost (COST)

- [x] **COST-01**: Every build and continue ends with one plain-English cost line — per worker and total tokens, no number at all for a worker whose tool reported none, the total counting only the measured workers and saying so, dollars never the headline, no price table anywhere (carries SPEND-08). *Wording corrected 2026-08-28 (plan 196-08): this line previously promised "measured vs estimated labelled", which D-01 as amended (owner, 2026-08-27) superseded — no estimated figure is ever rendered, because the only estimate available was derived from prompt length and COST-03 forbids that.*
- [x] **COST-02**: Wrapper-path (Claude Code / OpenCode) worker results report token usage so the line is populated on the path the owner actually uses (carries SPEND-02)
- [x] **COST-03**: `aether spend` shows per-worker tokens for the current run, mutating nothing, and no figure derives from a character budget (carries SPEND-07; reinstates the `aether spend` detail view the 2026-08-22 brief approved, reversing the 2026-08-14 cut of SPEND-06)
- [x] **COST-04**: The team card shows each worker's model and the reason for it; no routine role carries `inherit` (chronicler, keeper, includer today); every role left on the expensive model has a recorded reason after the owner rules on the retune proposal
- [x] **COST-05**: The three preserved spend-ledger branches (`worktree-agent-a47f…`, `…a59f…`, `…aa57…`) are reviewed and either salvaged or discarded with a written reason, then deleted

### One answer to "what next?" (NEXT)

- [x] **NEXT-01**: One pure resolver over canonical state returns the eight fields — current state/phase, what changed, open flags and signals, Queen recommendation, exact next command, 2–4 alternatives, context health (KEEP / SAFE / CLEAR_RECOMMENDED with reason codes), recovery/checkpoint state
- [x] **NEXT-02**: Every lifecycle command's closing card (init, discuss, plan, build, continue, pause, resume, seal, update, recover, status) renders from the resolver; quiet and JSON modes carry the same fields
- [x] **NEXT-03**: A ratchet asserts that hand-typed `/ant-…` command strings outside the resolver only shrink from a recorded baseline
- [x] **NEXT-04**: A `SessionStart` hook (`aether hook-session-start`, wired in `.claude/settings.json`) prints the "colony detected" card from the resolver on startup, resume and clear; the CLAUDE.md paragraph asking Claude to remember is removed
- [x] **NEXT-05**: `/ant-pause` is the canonical command and `/ant-pause-colony` an alias generated from an `aliases:` field in the command YAML on all three platforms; `aether update` reconciles and reports the repair
- [x] **NEXT-06**: The resolver never recommends a command absent from the installed command registry

### Put the thrown-away data back on screen (SHOW)

- [x] **SHOW-01**: The chat path renders the same ceremony as the direct path for plan, continue and seal — the JSON-mode finalizer / `ceremony closeout` split is closed (closeout output equals the direct visual, asserted)
- [x] **SHOW-02**: Data already carried is rendered — per-check verification report, success-criteria evidence lines, worker durations and tool counts, plan confidence and iterations, resume per-phase progress, recent decisions, drift note, specialist finding blocks, build-start blocker advisory — asserted as an invariant over the result map, not a named-section check
- [x] **SHOW-03**: `continue` shows live verification progress lines as each check runs
- [x] **SHOW-04**: Seal asks for confirmation and runs the wisdom review before sealing
- [x] **SHOW-05**: The recorded "deliberately dropped" decisions (one-terminal streaming, headed sections, verified safe-to-clear line) are respected and test-locked

### Feed the memory (FEED) — added 2026-08-30 from the whole-system audit

- [x] **FEED-01**: Worker results on build and continue are recorded as learning observations automatically; a fixture run leaves helper-authored sentences in the observation log with no hand-typed memory command
- [x] **FEED-02**: Worker, build and verification failures on build, continue, quick and swarm write midden entries automatically, and the next build brief carries the failure's own wording
- [x] **FEED-03**: Continue creates instincts from verified phase learnings and promotes them into QUEEN.md through the existing consolidation; the promoted list is non-empty on a fixture with real learnings, not hand-seeded
- [x] **FEED-04**: Phase completion and answered decisions emit FEEDBACK pheromones, midden past threshold emits an automatic REDIRECT, and pheromone expiry promotes to eternal memory again; the CLAUDE.md claims for these are backed by tests
- [x] **FEED-05**: Instincts at confidence ≥ 0.8 are promoted to the hive at phase end (continue), not only at seal, honouring `AETHER_HIVE_POLICY`
- [x] **FEED-06**: `/ant-memory-details` shows wisdom entries and pending promotions again and `/ant-status` shows the top instincts by confidence; the bare-count JSON alias is retired

### Memory reaches every helper (WIRE) — added 2026-08-30

- [x] **WIRE-01**: The Claude/OpenCode plan and colonize manifests carry the context capsule; a per-command test runs the real delegate lane and asserts a sentinel sentence from each memory source (QUEEN.md, hive, instincts, midden, pheromones, clarified intent, handoffs) in the assembled prompt — native-lane-only tests do not satisfy this
- [x] **WIRE-02**: The phase-research scout brief carries hive wisdom content; the instruction to write a hive section it cannot see is removed
- [x] **WIRE-03**: A condensed survey (colonize) digest reaches build, plan and research briefs within budget; a test asserts a surveyor-authored sentence arrives and that deleting the survey changes the prompt
- [x] **WIRE-04**: A completed Oracle run registers its output into the colony's research docs automatically; its findings reach the next builder without `--research <path>` or a manual promote
- [x] **WIRE-05**: The session-start card carries user preferences, top instincts and the last handoff, each content-locked
- [x] **WIRE-06**: No colony-prime capsule section is empty by construction — `PhaseLearnings` and `Decisions` gain a live writer or are removed; an invariant test asserts every section has a writer
- [x] **WIRE-07**: The previous phase's outcome.md, verification.json and review.json content reaches the next phase's build brief

### Overnight stamina (STAM) — added 2026-08-30

- [ ] **STAM-01**: Headless autopilot queues runtime-verification and visual checkpoints as pending decisions and continues; only genuine blockers halt the run
- [ ] **STAM-02**: The named pause contract is restored — auditor score floor, critical audit finding, runtime verification needed, escalated flags — and `aether run --dry-run` lists every trigger by name
- [ ] **STAM-03**: The replan pause reports learnings accumulated since the last plan
- [ ] **STAM-04**: The run summary reports elapsed wall-clock time alongside the cost line
- [ ] **STAM-05**: The pre/post-build blocker-count gate (`flag-check-blockers`) is live in run and status
- [ ] **STAM-06**: A fixture colony of ≥ 6 phases with simulated workers completes unattended under one `aether run`; the test fails if any phase halts for a queueable reason

### Prune the dead wood (PRUNE) — added 2026-08-30

- [ ] **PRUNE-01**: Every runtime subcommand has a live caller or is deleted; a ratchet test holds the orphan allowlist (baseline 117) and fails if it grows
- [ ] **PRUNE-02**: Every file written under `.aether/data` has a live reader or is no longer written; same ratchet (baseline 25)
- [ ] **PRUNE-03**: Every CLAUDE.md / shipped-doc claim about runtime behaviour is backed by a named test or removed
- [ ] **PRUNE-04**: The 72 dropped-since-v5.4 capabilities are triaged in a ledger (restored in a named phase, or dropped with the owner's reason); a test fails if any audit subject is absent from the ledger

### Proof (PROOF) — rewritten 2026-08-30 to the owner's own bar

- [ ] **PROOF-05**: One small real task in an owner project is run twice — plain Claude and Aether — with wall-clock, interventions and worker count recorded for both, whatever they show (carries former 186-07)
- [ ] **PROOF-06**: Aether finishes that task within 1.5× plain Claude's time with one worker plus free checks, and leaves phase, learnings and handoff records updated
- [ ] **PROOF-07**: A real overnight autopilot run on that project covers ≥ 3 phases unattended; any stops are queued-not-halted; the summary shows elapsed time and cost
- [ ] **PROOF-08**: A task interrupted mid-session resumes in a fresh session without the owner re-explaining what was in progress (carries PROOF-04)

## Future Requirements (deferred, not abandoned — ratified order behind v1.27)

- SPEC-first `/ant-init` and `/ant-spec` (spec §9; todo `2026-08-20-spec-builder-feature.md`)
- Whole-colony seal audit with flag dispositions RESOLVED / CARRY_TO_SHELF / PROPOSE_NEXT_INIT / ARCHIVE_NOTE / FORCE_SEAL_DEBT and immutable hashed seal packets (spec §6)
- Quota/budget controller, rate-limit interruption as resumable, large tool-output externalization (spec §5.1)
- Stage 1 bookkeeping remainder: append-only results, explicit supersession, evidence fingerprints, proof-obligation preflight (spec §2) — do the minimum inside a v1.27 phase only if it needs one, and say so
- Oracle planning (spec §14), Portal (spec §15), progressive skills and agent slimming (spec §12), Claude hooks beyond SessionStart (spec §13)
- Phase 172.1 CI gate environment integrity; Phase 173's one pending human check
- SKILL-03/04 skill validation on the Claude path; CATALOG-01/02 and TEST-01/02

## Out of Scope (explicit exclusions)

| Feature | Reason |
|---------|--------|
| A separate "featherweight lane" mode | Falls out of TEAM-01..04: a small job is the Queen choosing one worker and no reviewers |
| Making the chat the Queen wholesale (shell-era model) | Unwinds "the program owns the truth"; the proposal mechanism already lets the chat decide within the floor |
| Dollars as the headline, any price table | Owner ruling 2026-08-13; tokens only |
| All reviewers on the expensive model | Spec §21 |
| Removing the deterministic floor for anything | D11: safety is the floor, not a mandatory reviewer agent |
| Cross-platform pixel parity | Spec §21; semantic parity only |

## Traceability

Filled by the roadmapper. Each requirement maps to exactly one phase.

| Requirement | Phase | Status |
|-------------|-------|--------|
| FLOOR-01 | Phase 193 | Complete |
| FLOOR-02 | Phase 193 | Complete |
| FLOOR-03 | Phase 193 | Complete |
| FLOOR-04 | Phase 193 | Complete |
| TEAM-01 | Phase 194 | Complete |
| TEAM-02 | Phase 194 | Complete |
| TEAM-03 | Phase 194 | Complete |
| TEAM-04 | Phase 194 | Complete |
| TEAM-05 | Phase 194 | Complete |
| JOBS-01 | Phase 195 | Complete |
| JOBS-02 | Phase 195 | Complete |
| JOBS-03 | Phase 195 | Complete |
| JOBS-04 | Phase 195 | Complete |
| COST-01 | Phase 196 | Complete |
| COST-02 | Phase 196 | Complete |
| COST-03 | Phase 196 | Complete |
| COST-04 | Phase 196 | Complete |
| COST-05 | Phase 196 | Complete |
| NEXT-01 | Phase 197 | Complete |
| NEXT-02 | Phase 197 | Complete |
| NEXT-03 | Phase 197 | Complete |
| NEXT-04 | Phase 197 | Complete |
| NEXT-05 | Phase 197 | Complete |
| NEXT-06 | Phase 197 | Complete |
| SHOW-01 | Phase 198 | Complete |
| SHOW-02 | Phase 198 | Complete |
| SHOW-03 | Phase 198 | Complete |
| SHOW-04 | Phase 198 | Complete |
| SHOW-05 | Phase 198 | Complete |
| FEED-01 | Phase 198.1 | Complete |
| FEED-02 | Phase 198.1 | Complete |
| FEED-03 | Phase 198.1 | Complete |
| FEED-04 | Phase 198.1 | Complete |
| FEED-05 | Phase 198.1 | Complete |
| FEED-06 | Phase 198.1 | Complete |
| WIRE-01 | Phase 198.2 | Complete |
| WIRE-02 | Phase 198.2 | Complete |
| WIRE-03 | Phase 198.2 | Complete |
| WIRE-04 | Phase 198.2 | Complete |
| WIRE-05 | Phase 198.2 | Complete |
| WIRE-06 | Phase 198.2 | Complete |
| WIRE-07 | Phase 198.2 | Complete |
| STAM-01 | Phase 198.3 | Pending |
| STAM-02 | Phase 198.3 | Pending |
| STAM-03 | Phase 198.3 | Pending |
| STAM-04 | Phase 198.3 | Pending |
| STAM-05 | Phase 198.3 | Pending |
| STAM-06 | Phase 198.3 | Pending |
| PRUNE-01 | Phase 198.4 | Pending |
| PRUNE-02 | Phase 198.4 | Pending |
| PRUNE-03 | Phase 198.4 | Pending |
| PRUNE-04 | Phase 198.4 | Pending |
| PROOF-05 | Phase 199 | Pending |
| PROOF-06 | Phase 199 | Pending |
| PROOF-07 | Phase 199 | Pending |
| PROOF-08 | Phase 199 | Pending |

**Coverage:**

- v1 requirements: 56 total
- Mapped to phases: 56
- Unmapped: 0 ✓

---
*Requirements defined: 2026-08-22*
*Last updated: 2026-08-30 after the whole-system audit (11 phases, 193-199 incl. 198.1-198.4, 56/56 mapped)*
