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

- [ ] **TEAM-01**: The castes a build requires shrink to builder (non-discovery phases) — no caste is required by inferred mode or keyword alone; `TestWatcherIsAlwaysRequiredOnBuild` is retired through the test-deletion ledger citing D11, and the negative half of the Probe rule (no Probe where nothing is testable) stays
- [x] **TEAM-02**: The program forces a reviewer only for a named high-risk signal — credentials/auth, payments, data deletion, migration, release sign-off — and attaches the signal as a visible reason on the team card (table test: CSV export → none forced; password reset → security reviewer with reason)
- [x] **TEAM-03**: Every worker that spawns carries a one-line reason the owner can read; a proposal naming a worker without a reason is refused by name
- [x] **TEAM-04**: The chat's proposal (`--castes` + `--caste-reason`) is the primary team source; the keyword engine is the autopilot / no-proposal fallback and is retuned to the same floor — a 1-task bug fix yields one dispatch plus free checks on both paths
- [x] **TEAM-05**: The owner's overrides in both directions (`--castes`, `--heavy`, `--light`, the check-in card) keep working and stay test-locked after the floor moves

### Coherent jobs, not one worker per task (JOBS)

- [ ] **JOBS-01**: The Queen can group tasks into one job with a reason; the runtime validates dependency order and refuses, by name, a grouping that violates `depends_on`
- [ ] **JOBS-02**: A grouped job's brief carries every covered task's criteria and its completion credits every covered task (the existing `covered_task_ids` chain, re-proven end to end)
- [ ] **JOBS-03**: Without a proposal, tasks sharing files or a dependency chain cluster into one job beyond consecutive same-caste steps — the CalVault field failure (six file-copy batches) as a fixture yields one worker
- [ ] **JOBS-04**: Grouped jobs work in worktree mode (coalescing is no longer disabled there)

### See what it cost (COST)

- [ ] **COST-01**: Every build and continue ends with one plain-English cost line — per worker and total tokens, measured vs estimated labelled, dollars never the headline, no price table anywhere (carries SPEND-08)
- [ ] **COST-02**: Wrapper-path (Claude Code / OpenCode) worker results report token usage so the line is populated on the path the owner actually uses (carries SPEND-02)
- [ ] **COST-03**: `aether spend` shows per-worker tokens for the current run, mutating nothing, and no figure derives from a character budget (carries SPEND-07; reinstates the `aether spend` detail view the 2026-08-22 brief approved, reversing the 2026-08-14 cut of SPEND-06)
- [ ] **COST-04**: The team card shows each worker's model and the reason for it; no routine role carries `inherit` (chronicler, keeper, includer today); every role left on the expensive model has a recorded reason after the owner rules on the retune proposal
- [ ] **COST-05**: The three preserved spend-ledger branches (`worktree-agent-a47f…`, `…a59f…`, `…aa57…`) are reviewed and either salvaged or discarded with a written reason, then deleted

### One answer to "what next?" (NEXT)

- [ ] **NEXT-01**: One pure resolver over canonical state returns the eight fields — current state/phase, what changed, open flags and signals, Queen recommendation, exact next command, 2–4 alternatives, context health (KEEP / SAFE / CLEAR_RECOMMENDED with reason codes), recovery/checkpoint state
- [ ] **NEXT-02**: Every lifecycle command's closing card (init, discuss, plan, build, continue, pause, resume, seal, update, recover, status) renders from the resolver; quiet and JSON modes carry the same fields
- [ ] **NEXT-03**: A ratchet asserts that hand-typed `/ant-…` command strings outside the resolver only shrink from a recorded baseline
- [ ] **NEXT-04**: A `SessionStart` hook (`aether hook-session-start`, wired in `.claude/settings.json`) prints the "colony detected" card from the resolver on startup, resume and clear; the CLAUDE.md paragraph asking Claude to remember is removed
- [ ] **NEXT-05**: `/ant-pause` is the canonical command and `/ant-pause-colony` an alias generated from an `aliases:` field in the command YAML on all three platforms; `aether update` reconciles and reports the repair
- [ ] **NEXT-06**: The resolver never recommends a command absent from the installed command registry

### Put the thrown-away data back on screen (SHOW)

- [ ] **SHOW-01**: The chat path renders the same ceremony as the direct path for plan, continue and seal — the JSON-mode finalizer / `ceremony closeout` split is closed (closeout output equals the direct visual, asserted)
- [ ] **SHOW-02**: Data already carried is rendered — per-check verification report, success-criteria evidence lines, worker durations and tool counts, plan confidence and iterations, resume per-phase progress, recent decisions, drift note, specialist finding blocks, build-start blocker advisory — asserted as an invariant over the result map, not a named-section check
- [ ] **SHOW-03**: `continue` shows live verification progress lines as each check runs
- [ ] **SHOW-04**: Seal asks for confirmation and runs the wisdom review before sealing
- [ ] **SHOW-05**: The recorded "deliberately dropped" decisions (one-terminal streaming, headed sections, verified safe-to-clear line) are respected and test-locked

### Proof (PROOF)

- [ ] **PROOF-05**: The one live benchmark run (former 186-07) is fired with the owner present and its result recorded, whatever it shows
- [ ] **PROOF-06**: The full showdown (former Phase 192) runs on the improved system — Aether interactive, Aether autopilot, GSD — recording tokens, time, interventions, worker count, completion truth, recovery failures and git cleanliness (carries PROOF-01..03)
- [ ] **PROOF-07**: Acceptance gate: a 1-task bug fix costs ≤ 1 worker + free checks; a CSV-export phase ≤ 4 workers across build and continue; median tokens per successful task ≤ 1.5× GSD; unscripted interventions ≤ GSD's; recovery failures 0 — the milestone completes only when this gate passes
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
| TEAM-01 | Phase 194 | Pending |
| TEAM-02 | Phase 194 | Complete |
| TEAM-03 | Phase 194 | Complete |
| TEAM-04 | Phase 194 | Complete |
| TEAM-05 | Phase 194 | Complete |
| JOBS-01 | Phase 195 | Pending |
| JOBS-02 | Phase 195 | Pending |
| JOBS-03 | Phase 195 | Pending |
| JOBS-04 | Phase 195 | Pending |
| COST-01 | Phase 196 | Pending |
| COST-02 | Phase 196 | Pending |
| COST-03 | Phase 196 | Pending |
| COST-04 | Phase 196 | Pending |
| COST-05 | Phase 196 | Pending |
| NEXT-01 | Phase 197 | Pending |
| NEXT-02 | Phase 197 | Pending |
| NEXT-03 | Phase 197 | Pending |
| NEXT-04 | Phase 197 | Pending |
| NEXT-05 | Phase 197 | Pending |
| NEXT-06 | Phase 197 | Pending |
| SHOW-01 | Phase 198 | Pending |
| SHOW-02 | Phase 198 | Pending |
| SHOW-03 | Phase 198 | Pending |
| SHOW-04 | Phase 198 | Pending |
| SHOW-05 | Phase 198 | Pending |
| PROOF-05 | Phase 199 | Pending |
| PROOF-06 | Phase 199 | Pending |
| PROOF-07 | Phase 199 | Pending |
| PROOF-08 | Phase 199 | Pending |

**Coverage:**

- v1 requirements: 33 total
- Mapped to phases: 33
- Unmapped: 0 ✓

---
*Requirements defined: 2026-08-22*
*Last updated: 2026-08-22 after roadmap creation (7 phases, 193-199, 33/33 mapped)*
