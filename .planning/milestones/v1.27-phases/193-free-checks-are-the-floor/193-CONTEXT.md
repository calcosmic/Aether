# Phase 193: Free Checks Are the Floor - Context

**Gathered:** 2026-08-22
**Status:** Ready for planning

<domain>
## Phase Boundary

The program's own deterministic checks become the verification floor on every
phase: build, types, lint, tests (targeted per phase, full at the end), "the
files a worker claims to have changed exist", and "each success criterion has
evidence". They can never be skipped by a depth flag, a review policy, a team
proposal or `--skip-watchers`. A phase with zero reviewer workers advances when
they pass and is blocked when they fail. No gate demands a checker worker
implicitly. A phase is verified once: the build side runs free checks only,
agent review lives in `continue`. Delivers FLOOR-01..04.

NOT in this phase: which workers the Queen (the coordinator) sends and which
castes are required (Phase 194); grouping tasks into jobs (195); the cost line
(196); the closing "what next?" card (197); live progress display while checks
run (198 — this phase emits the data, 198 renders it).

</domain>

<decisions>
## Implementation Decisions

### Projects with nothing mechanical to check
- **D-01:** When no build/types/lint/test command resolves for a project (a
  notes vault, a design-doc repo), the floor is "claimed files exist + each
  criterion has evidence". The closing card says plainly "no tests to run in
  this project". **No checker worker is sent for that reason.** Today zero
  executed checks hands verification to a Watcher (`runCodexContinueVerification`,
  `cmd/codex_continue.go` ~1585-1600); that fallback goes. The existing project
  note remains the way an owner adds a check (`## Verification Commands` in
  `CLAUDE.md` / `AGENTS.md` / `.opencode/OPENCODE.md` / `.codex/CODEX.md`, or
  `## Commands` in `.aether/data/codebase.md`); no new prompt is added.

### When a free check fails and no reviewer was sent
- **D-02:** Auto-send **one** builder fix attempt, then stop. The failed check's
  output reaches the builder as a compact failure index (not the whole log),
  together with the task(s) whose files are implicated. If the re-run check
  still fails, `continue` blocks and the card shows the failing check and the
  single exact command to re-run the builder by hand. Never a Watcher to
  diagnose first; never a second automatic attempt. This is distinct from the
  2026-08-21 "recovery decides but never retries" ruling: that ruling concerns
  recovery re-running *failed or interrupted workers*; this is one bounded fix
  at the verification floor, chosen by the owner on 2026-08-22.
  — **Reversibility:** reversible — one attempt-count constant and one branch.
- **D-03 (Claude):** the fix attempt is a new attempt in the attempt journal,
  never overwriting the first builder's result (spec §2.5/§2.6 append-only), it
  carries its own reason ("fixing failed check: tests"), and it is counted on
  the team card and the cost line (Phase 196 renders).

### Proving each success criterion without a reviewer
- **D-04:** Bind a check at plan time where possible; fall back to builder
  evidence re-run by the program. The planner binds each criterion to a concrete
  check through the existing `evidence_requirements`
  (`CriterionEvidenceRequirement.Checks` / `.Artifacts`: a command, an artifact
  that must exist, a named test). Unbound criteria get the synthetic default
  **minus `watcher`**: `claims` plus the matching free check. The program re-runs
  the verification command the builder reports in its handoff (`commands_run`)
  and confirms its `changed_files` exist. The worker's word alone never
  satisfies a criterion.
- **D-05 (Claude — owner said "you decide"):** a criterion no machine can prove
  is marked `needs_owner_confirmation`. The phase advances; the closing card
  lists the criterion for the owner; `aether seal` blocks until the owner has
  confirmed it through the existing decision-answer path; no reviewer worker is
  spawned for it. — **Reversibility:** costly — adds a criterion state that seal
  and the cards read; undoing touches seal, continue and the card fields.
- **D-06:** Remove `watcher` from `syntheticCriterionRequirements` defaults
  (`cmd/criterion_evidence.go` ~147) and make `evaluateCriterionCheck("watcher")`
  satisfied by deterministic evidence when no watcher was dispatched (FLOOR-03);
  a watcher that *was* dispatched and failed still blocks. Plans authored before
  this phase with `required_checks: watcher` keep working under the same rule —
  compatibility, no migration.

### How much to run each time
- **D-07:** Targeted per phase, full at the end. The default verification runs
  the project's checks scoped to what the phase touched (derived from the
  workers' `changed_files`/claims → packages or directories, language-aware
  where a scoped runner exists, e.g. `go test ./pkg/x/...`; otherwise the full
  run). The full suite runs on the final phase of the plan, at seal, and
  whenever scoping cannot be derived. The card states which ran ("targeted: 3
  packages" / "full suite"). Keep today's verification timeout default; a
  timed-out check is a failed check and follows D-02.
  — **Reversibility:** reversible — a scoping function with a full-run fallback.

### Verified once (FLOOR-04) — from the brief, confirmed
- **D-08:** The build-side "verification" stage dispatches no watcher
  (`cmd/codex_build.go` ~1212, the `queenCastes["watcher"]` dispatch). Build
  finalize records the free checks as a report, not an advancement gate —
  advancing stays `continue`'s job. Agent review lives in `continue` only, and
  no caste is dispatched at both boundaries for the same phase unless the Queen
  explicitly asks. Phase 194 moves the required-caste floor; 193 only stops the
  duplicate dispatch and makes continue's gates deterministic-first. If
  `TestWatcherIsAlwaysRequiredOnBuild` breaks here rather than in 194, retire it
  in this phase through the test-deletion ledger citing D11 and say so.

### Claude's Discretion
- D-05 above (unprovable criteria).
- Exact scoping heuristics for targeted runs; which missing commands read as
  "skipped: no command" versus failed; the data fields the card needs (Phases
  197/198 own the wording).
- `--skip-watchers` keeps its name (it now only skips reviewer workers, never
  checks); its help text is corrected.

### Folded Todos
- `.planning/todos/pending/2026-08-01-finalize-reconcile-task-evidence-gate.md`
  — continue-finalize's `implementation_evidence` gate ignores
  `--reconcile-task` (`cmd/codex_continue.go` ~3236, `assessment.PositiveEvidence`
  on the finalize path). Folded into FLOOR-03: operator-recorded reconciliation
  counts as evidence on both lanes.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Rulings and brief
- `.planning/decisions/2026-08-22-queen-decides-program-checks.md` — ruling D11: deterministic checks are the floor, reviewer workers are judgement, reviewers forced only by named risk, verify once.
- `.planning/decisions/2026-08-21-owner-rulings-priority-spec-v3.md` — rulings D1–D12 (D12 is the v1.27 order amendment).
- `.planning/research/v1.27-milestone-brief.md` — feature 1 "Free checks are the floor", the named tests, the measured baseline.
- `.planning/research/priority-spec-v3-backlog.md` — the ratified order; spec §2.10 ("if review policy is none, the runtime must not synthesize a hidden Watcher requirement") and §7 ("QUICK means zero reviewer agents, not zero deterministic validation").
- `~/Downloads/Aether_Priority_Implementation_Spec_v3.md` (external, the owner's copy) §2.10, §2.12, §5.1 (large tool-output externalisation — the failure-index idea in D-02), §7.
- `CLAUDE.md` — "Definition of Done": a requirement is satisfied only when a command exists that fails when it is unmet; prefer invariants over named-section checks.

### Code this phase changes
- `cmd/codex_continue.go` — `runCodexContinueVerification` (~1560-1700: deterministic steps, claims, the watcher decision including the zero-executed-checks fallback and three auto-skip paths), `runCodexContinueReview` (~1319: `--skip-watchers`), the gate list (~3196-3262, `implementation_evidence`), `requiredVerificationChecks` (~3027), `resolveCodexVerificationCommands` (~2471: the project-note sources).
- `cmd/criterion_evidence.go` — `syntheticCriterionRequirements` (~143: default checks include `watcher`), `evaluateCriterionCheck` (~600), the `check != "watcher"` deterministic flag (~501).
- `cmd/codex_build.go` ~1205-1225 — the build-side "verification" watcher dispatch (D-08).
- `cmd/codex_build_finalize.go` — where the build's free-check report is recorded (D-08).
- `cmd/verify_out_of_band.go` — Phase 191.1's fresh verification of current disk state against a phase's criteria; reuse its re-verify machinery for D-04, and keep its "never fabricate worker receipts" guard intact.
- `cmd/codex_continue_finalize.go` — the wrapper lane of continue; every rule above must hold on both lanes (the 2026-08-21 review gate found the in-process lane half-wired).
- `.planning/milestones/v1.26-phases/191.1-field-hardening-close-the-four-2026-08-21-field-reported-def/191.1-CONTEXT.md` — FIELD decisions (checker verdict field D-05, out-of-band provenance).

### Tests that assert the old floor (coordinate with Phase 194)
- `TestWatcherIsAlwaysRequiredOnBuild`, `TestQueenCannotDropTheWatcher` (cmd/queen_judgement_test.go, cmd/queen_spawn_budget_test.go) — retired in 194 via the test-deletion ledger; see D-08 for the case where 193 must do it.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- The deterministic steps already exist: `runVerificationStep` for build/types/lint/tests with a timeout; `verifyCodexBuildClaims` (claimed files exist); criterion evidence binding and evaluation (`CriterionEvidenceRequirement`, `evaluateCriterionCheck`); mandatory worker handoff fields `commands_run`, `changed_files`, `verification_status` (since Phase 189); `aether verify-out-of-band` (fresh verification against disk).
- The project check note: "## Verification Commands" sections (CLAUDE.md / AGENTS.md / OPENCODE.md / CODEX.md) and "## Commands" in `.aether/data/codebase.md`, merged by `resolveCodexVerificationCommands`.

### Established Patterns
- Gates are named `gateCheck` entries with `FixHint` and `RecoveryOptions`; the removed always-pass `operational_evidence` gate is the precedent that "a gate that cannot gate is worse than none".
- Four watcher auto-skip paths already exist (`--skip-watchers`, provider unavailable, host boundary, consecutive-failure threshold). This phase replaces them with one rule instead of adding a fifth.
- Tests assert proportions and invariants, reproduce the field failure first (fail-then-pass), and every skip path must be walked by `TestDeterministicChecksCannotBeSkipped`.

### Integration Points
- `continue` on both lanes (in-process and wrapper/external via `continue-finalize`).
- `build-finalize` (free-check report), `seal` (the `needs_owner_confirmation` block), and the closing-card data fields consumed by Phases 197/198.
- The attempt journal (`cmd/build_attempt.go`) for D-03's append-only fix attempt.

</code_context>

<specifics>
## Specific Ideas

- The owner's words, 2026-08-22: "I don't even necessarily want a watcher every time… it's meant to use the model's intelligence to delegate roles."
- Measured baseline: a 1-task bug fix is sent 8 workers today (three of them verification on the continue side); v5.4.0 sent 3-4. This phase removes the duplicate verification dispatch; 194 removes the rest.
- Card wording must be plain English ("no tests to run in this project", "targeted: 3 packages") — no repo vocabulary.

</specifics>

<deferred>
## Deferred Ideas

- Which workers are required and which reviewers are forced → Phase 194.
- Live progress lines while checks run → Phase 198 (193 emits the data).
- Large tool-output externalisation as a general facility (spec §5.1) → later stage; 193 builds only the compact failure index D-02 needs.

### Reviewed Todos (not folded)
- `2026-08-21-weight-classes-pipeline-fits-the-task.md` — Phase 194 (tagged `resolves_phase: 194`).
- `2026-08-20-spec-builder-feature.md` — SPEC-first init, v1.28+; not this milestone.
- `2026-08-01-ts-host-preflight-hardcoded-timeout.md` — the TS-host probe twin; unrelated to the floor.

</deferred>

---

*Phase: 193-free-checks-are-the-floor*
*Context gathered: 2026-08-22*
