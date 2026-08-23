# Phase 194: The Queen Decides the Team - Context

**Gathered:** 2026-08-23
**Status:** Ready for planning

<domain>
## Phase Boundary

Which workers get sent becomes the Queen's judgement call with a stated reason,
not a fixed rule. The castes a build requires shrink to `builder` (non-discovery
phases). The program forces a reviewer only for one of five named high-risk
signals — credentials/auth, payments, data deletion, database structure changes
(migrations), release sign-off — and says which signal on the team card. Every
worker that spawns carries a one-line plain-English reason; a proposal naming a
worker without one is refused by name. The chat's proposal (`--castes` +
per-worker reasons) is the primary team source; the keyword engine is the
autopilot / no-proposal fallback and is retuned to the same floor. The owner's
overrides (`--castes`, `--heavy`, `--light`, the check-in card) keep working.
A one-task bug fix is sent one worker plus the Phase 193 free checks on both
paths. Delivers TEAM-01..05 and closes `.planning/WINDOWS.md` #1.

NOT in this phase: grouping tasks into jobs (195); the cost line and model
display on the card (196 renders the reasons this phase records); the closing
"what next?" card (197); restoring Classic display data (198); the benchmark
and showdown (199); a spec-writing command (deferred, see below); any change to
the deterministic floor itself (193, untouched).

</domain>

<decisions>
## Implementation Decisions

### What counts as risky (TEAM-02)
- **D-01:** The named high-risk signals are exactly the five from ruling D11:
  **credentials/auth** (passwords, logins, sessions, secrets), **payments**,
  **data deletion**, **database structure changes / migrations**, **release
  sign-off**. Nothing else forces a reviewer: not inferred mode ("production"),
  not blast-radius wording ("core runtime", "state machine" — today's medium
  risk list), not phase position. Claude chose this on the owner's behalf (the
  owner answered "I don't know"); it is the list the owner ratified on
  2026-08-22. — **Reversibility:** reversible — one signal table in one place;
  every firing prints its signal on the card, so a wrong entry is visible the
  first time it fires.
- **D-02:** Two detectors, one signal table, add-only on the second. The plan's
  wording (`collectPhaseText`: name + description + tasks) decides up front so
  the pre-build card can show it. At the checking step (`continue`) the
  builder's reported `changed_files` are matched against the same five signals
  (path patterns such as `migrations/`, `auth/`, `payment`) and may **add** a
  forced reviewer; the file detector never removes one. Matching is
  word-boundary; prefer phrases over lone ambiguous words ("token" alone is a
  known false alarm — "token bucket rate limiter"). A false alarm is acceptable
  because of D-03. — **Reversibility:** reversible — the file detector is a
  separate function the continue side can stop calling.
- **D-03:** Only the owner can waive a forced reviewer, on the pre-build
  check-in card, with a reason that is recorded (through the existing
  `decision-answer` path, like a clarification). The Queen/assistant cannot
  waive it; `--light` cannot drop it; autopilot never waives. A waiver covers
  **one signal for one phase**: the same signal re-detected from changed files
  stays waived; a different signal still adds. The card shows "waived by owner:
  <reason>". — **Reversibility:** costly — the card gains a choice and the
  runtime a recorded decision kind that both build and continue read.

### The forced reviewer (TEAM-02)
- **D-04:** One reviewer per signal, not a pair. `gatekeeper` (security
  reviewer) for credentials/auth, payments, release sign-off; `auditor` (code
  reviewer) for data deletion and migrations. A phase carrying signals from
  both classes gets both. The Queen may add more by proposal.
  `TestHighRiskPhaseKeepsBothReviewers` ("high risk ⇒ auditor AND gatekeeper")
  contradicts this and is retired or rewritten to the one-per-signal rule.
- **D-05:** The forced reviewer runs at the checking step (`continue`), the
  single review pass Phase 193 established (FLOOR-04) — never at build. The
  build manifest and the check-in card **announce** it: "a security reviewer
  will check this at the verification step — this touches logins". The build
  records the forced set (caste, signal, reason, source = plan wording) in its
  decision; `continue` reads that record, adds any D-02 file-detected signal,
  and dispatches. One derivation, one boundary — this is what closes
  `.planning/WINDOWS.md` #1 (build and continue each computing "always
  required" independently). — **Reversibility:** costly — the record is a
  contract between `build-finalize` and both continue lanes.
- **D-06:** Every other implicit floor goes: "production mode ⇒ auditor"
  (`queenBuildSafetyReviewRequired`'s mode branch), "watcher always"
  (`queenBuildSafetyRequiredCastes` line 142), and "final phase ⇒ heavy" in the
  smart default (`resolveSmartVerificationDepth`: position no longer raises
  verification depth; an explicit `--heavy` still does). **The last phase of a
  plan is treated like any other phase** — owner's choice 2026-08-23.
- **D-07:** Build-side required castes shrink to `builder` on non-discovery
  phases (TEAM-01). `probe` is never required; it keeps only its negative rule
  (never where nothing is testable — `queenPhaseProducesTestableCode` stays as
  a *refusal* gate for proposals, and the negative half of
  `TestProbeIsRequiredOnlyWhereItCanFindSomething` stays). `watcher` is neither
  required nor dispatched as a build-side worker (193 D-08 already stopped the
  dispatch; this phase removes the requirement).

### A reason for every helper (TEAM-03)
- **D-08:** A proposal carries one reason per worker. A worker named without a
  reason is refused **for that worker only**, by name, in `caste_decision`
  (e.g. `refused_no_reason: [probe]`) and in the Summary line; the rest of the
  team is sent. The wrapper relays the refusal and may re-propose with reasons.
  A single team-level `--caste-reason` string no longer satisfies TEAM-03 for
  any worker (it may survive as the team summary). The flag shape is Claude's
  discretion.
- **D-09:** A generic caste description (`casteRosterProduces`, the roster's
  `produces` blurb) is **not** a reason — the card may show it as "what it
  does", never in the reason slot. A reason must say why *this phase* needs the
  worker (the task, the file, the risk, or the owner's note). The same rule
  binds the fallback engine: a fallback pick must carry a plain-English
  sentence ("Tracker — the phase describes a bug to investigate"), never
  "Score 20 >= threshold 15" (`queenCandidateDispatches`, `caste_relevance.go`
  ~176); a pick the engine cannot word is not sent. Runtime-added workers carry
  runtime-written reasons: builder "writes the code for N task(s): …"; forced
  reviewer "forced: this touches logins (the plan mentions 'password reset')".
- **D-10:** Reasons travel with the worker: the check-in card (one per worker,
  beside REQUIRED/OPTIONAL), the manifest (`selected_reasons` is the existing
  per-caste slot), and the dispatch/attempt record — so Phase 196's cost line
  and Phases 197/198's cards render them without recomputation. REQUIRED on the
  card now means exactly "builder, or forced by a named signal (signal shown)".

### Autopilot and the owner's dials (TEAM-04, TEAM-05)
- **D-11:** With no proposal (autopilot `aether run`, or a wrapper that never
  re-fetches with `--castes`), the fallback team is **builder plus any forced
  reviewer** — the keyword engine no longer adds optional specialists on build
  or continue. The relevance registry stays for *refusing* proposals (a proposed
  caste scoring zero relevance is still refused, as today) and may list
  candidates for the Queen to consider; it no longer selects. Owner's
  rationale: that engine is what sent 8 workers to a one-task bug fix.
  — **Reversibility:** reversible — a gate on the selector.
- **D-12 (Claude):** discovery-mode phases with no proposal get one `scout`
  (research is the deliverable; builder is suppressed there today). Only build
  and continue change in this phase; plan, colonize, swarm and seal keep their
  current required sets.
- **D-13:** `--heavy` = the full review panel at the checking step
  (`gatekeeper` + `auditor` + `probe`, probe still subject to testable code):
  the owner's explicit ask outranks the Queen's judgement. On build, heavy
  stays a ceiling (≥ 8 workers), never a floor. `--light` = the Queen's team
  minus optional reviewers; it can never drop a forced reviewer or a free
  check. `--castes` keeps working on build and continue, fast path included.
  Continue's required set by depth: light/standard → none; heavy → the panel.
  CLAUDE.md's "Queen-Owned Orchestration" depth table and "Team Check-In"
  section must be rewritten to match (a documentation claim about runtime
  behaviour must be testable or removed).
- **D-14:** The check-in card keeps **pausing for the owner's OK on every
  build, including a one-worker team** — owner's choice 2026-08-23 ("pause and
  ask, same as today"). `--no-checkin` remains the opt-out; autopilot never
  sees the card. The card gains the waive choice (D-03).

### Tests and the retirement ledger (TEAM-01, CLAUDE.md Definition of Done)
- **D-15:** Retire through `.aether/docs/retired-tests-ledger.md` citing
  `.planning/decisions/2026-08-22-queen-decides-program-checks.md` (D11):
  `TestWatcherIsAlwaysRequiredOnBuild`, `TestQueenCannotDropTheWatcher`,
  `TestSafetyCastesSurviveProbeGating` (asserts auditor on production),
  `TestHighRiskPhaseKeepsBothReviewers` (or rewrite per D-04), and the positive
  half of `TestProbeIsRequiredOnlyWhereItCanFindSomething`. Keep and extend:
  `TestQueenCannotSkipSecurityReviewOnSecurityWork` (now also asserts the
  reason), `TestGatekeeperNeedsASecuritySignal`,
  `TestQueenChoiceReachesTheDispatchList` (assert the spawn list, never the
  decision record), `TestTeamCheckinCardShowsReasonAndRequiredMarking`,
  `TestContinueFastPathHonoursCasteProposal`,
  `TestBuildWorkerCapHonoursVerificationDepth`, `TestPhaseVerifiedOnce`.
  New, named by the roadmap: `TestReviewerForcedOnlyByNamedRisk` (table: CSV
  export → none; password reset → gatekeeper + reason; refund button →
  gatekeeper; delete stale accounts → auditor; add a column/migration →
  auditor; a "production" phase with no signal → none),
  `TestNoWorkerWithoutStatedReason` (refused by name, rest sent; generic blurb
  rejected; fallback reason is a sentence), `TestOneTaskBugFixIsOneWorkerPlusChecks`
  (end to end on both the proposal path and the fallback path, build **and**
  continue counted). Close WINDOWS #1 with a test that walks a real manifest +
  continue plan pair for a production/security phase with no proposal and
  asserts no caste appears at both boundaries; then
  `gsd-tools windows fixed 1`.

### Claude's Discretion
- Signal vocabulary and file-path patterns for D-01/D-02 (word-boundary,
  phrases over lone words); whether the existing policy override
  (`loadReviewDepthPolicy` keyword lists) remains the extension point.
- The per-worker reason flag shape (D-08) and manifest field names.
- The one-scout discovery fallback (D-12).
- Ledger entry wording; the CLAUDE.md rewrite; the wrapper text in all three
  byte-identical copies of `build.md` / `continue.md` ("A build always gets a
  Watcher" must go).
- Whether `TestHighRiskPhaseKeepsBothReviewers` is retired or rewritten.

### Folded Todos
- `.planning/todos/pending/2026-08-21-weight-classes-pipeline-fits-the-task.md`
  — "small jobs should get the small machine automatically" (tagged
  `resolves_phase: 194`). Resolved by D-07, D-11 and D-13 without a lane
  switch: a small job is the Queen choosing one worker; the no-proposal
  fallback sends one worker; reviewers are judgement or named risk; the free
  checks never shrink. Mark the todo resolved when
  `TestOneTaskBugFixIsOneWorkerPlusChecks` passes.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Rulings and brief
- `.planning/decisions/2026-08-22-queen-decides-program-checks.md` — ruling D11: reviewers are judgement with a stated reason; forced only by a named high-risk signal; verify once; the tests it names; the retirement of `TestWatcherIsAlwaysRequiredOnBuild`.
- `.planning/decisions/2026-08-21-owner-rulings-priority-spec-v3.md` — rulings D1–D12 (D3: the chat proposes, Go validates and records; D4: safety outranks everything, redefined by D11 as the deterministic floor + named-risk forcing).
- `.planning/research/v1.27-milestone-brief.md` — feature 2 "The Queen decides the team"; the measured 8-worker baseline; "a separate featherweight lane is dropped: it falls out of feature 2".
- `.planning/research/priority-spec-v3-backlog.md` — spec §2.10 (no hidden Watcher requirement when review policy is none), §7 (QUICK = zero reviewer agents, not zero validation).
- `.planning/REQUIREMENTS.md` — TEAM-01..05 verbatim; "Out of Scope" rows (no featherweight lane; the chat is not the Queen wholesale).
- `.planning/WINDOWS.md` — open entry #1 (Probe/Auditor/Gatekeeper double-dispatch); `/gsd-ship` blocks while open.
- `.planning/phases/193-free-checks-are-the-floor/193-CONTEXT.md` — D-06 (watcher dropped from synthetic criterion defaults), D-08 (build-side verification dispatches no watcher; "Phase 194 moves the required-caste floor").
- `CLAUDE.md` — "Definition of Done"; "Queen-Owned Orchestration" and "Team Check-In and Owner Decisions" (both describe the old floor and must be rewritten, not appended to); the plain-English mandate for anything the owner reads.

### Code this phase changes
- `cmd/queen_spawn_budget.go` — `queenBuildSafetyRequiredCastes` (~139: watcher always, builder non-discovery, probe if testable, auditor if high risk or production, gatekeeper if high risk or security wording), `queenPhaseHasSecuritySignal` (~187: today's keyword list), `queenBuildSafetyReviewRequired` (~211: the production-mode branch to delete), `queenSpawnBudgetForPhase` (~27: `RequiredCastes`, `MaxWorkers`), `queenPhaseProducesTestableCode` (~345).
- `cmd/queen_judgement.go` — `queenApplyJudgement` (~91: refuses zero-relevance castes, restores required castes, trims to budget; `Rationale` is one string), `queenCasteDecisionSummary` (~365: `caste_decision` fields), `Summary()` (~54: "required for this phase regardless").
- `cmd/caste_relevance.go` — `isAlwaysRequired` (~316: build delegates to the spawn-budget floor; continue: light → watcher, standard → watcher + probe-if-testable, heavy → watcher+gatekeeper+auditor+probe), `queenCandidateDispatches` (~158–190: the selector and its "Score %d >= threshold %d" rationale), `casteRelevanceScore` (~107), `spawnThreshold` (~271), `isCasteSuppressed` (~410: discovery suppresses builder).
- `cmd/review_depth.go` — `phaseRiskLevel` (~288: keyword over phase **name only**; `securityRiskKeywordsFallback` ~62 includes "token", "session", "audit"), `resolveSmartVerificationDepth` (~321: final phase ⇒ heavy — D-06 removes the position rule), `loadReviewDepthPolicy` (policy-file override of keyword lists).
- `cmd/codex_continue.go` — `queenContinueReviewSpecsWithJudgement` (~1269: skips watcher, maps castes to review specs), `runCodexContinueReview` (~1331), `plannedContinueReviewDispatches` (~1431); and `cmd/codex_continue_plan.go` `plannedExternalContinueDispatches` (~320) — both continue lanes must read the build's forced-reviewer record (D-05).
- `cmd/codex_dispatch_contract.go` — `SelectedReasons` / `PrunedReasons` (~461, composed ~570 from `queenSpawnBudgetDecisions`): the existing per-caste reason slots D-10 reuses.
- `cmd/ceremony_team_checkin.go` — `renderCeremonyTeamCheckin` (~30: REQUIRED/OPTIONAL marking, reason fallback to `casteRosterProduces` — D-09 forbids that fallback in the reason slot), the waive choice (D-03/D-14).
- `cmd/codex_workflow_cmds.go` — `--castes`, `--caste-reason` (~1376 build, ~1395 continue), `--heavy`/`--light` (~133, ~232), `--no-checkin` (~172).
- `cmd/codex_build.go` — `plannedBuildDispatchesWithJudgement` (~277), `manifest.CasteDecision` (~322); `cmd/codex_build_finalize.go` — where the forced-reviewer record is persisted for continue (D-05).
- `cmd/codex_continue_finalize.go` — `continueReviewCastesArePlannedSubset` (~1066).
- `cmd/autopilot.go` — passes no proposal; exercises the D-11 fallback.
- `.claude/commands/ant/build.md` (~168–210, "A build always gets a Watcher; credential, auth, and release-gate work always gets a security review") and `continue.md` (~121), plus the `.opencode` and flat-mirror copies — byte-identical triplets, edit all three.
- `.aether/docs/retired-tests-ledger.md` — the RETIRE-04 format (original path, what it covered, disposition, removed in).

### Tests (existing; see D-15 for keep / retire)
- `cmd/queen_probe_gating_test.go` — `TestProbeIsRequiredOnlyWhereItCanFindSomething` (29), `TestWatcherIsAlwaysRequiredOnBuild` (83), `TestSafetyCastesSurviveProbeGating` (~97), `TestGatekeeperNeedsASecuritySignal` (162), `TestHighRiskPhaseKeepsBothReviewers` (222).
- `cmd/queen_judgement_test.go` — `TestQueenCannotDropTheWatcher` (29), `TestQueenCannotSkipSecurityReviewOnSecurityWork` (48), `TestQueenChoiceReachesTheDispatchList` (215).
- `cmd/claudemd_verification_depth_test.go` — `TestBuildWorkerCapHonoursVerificationDepth` (107).
- `cmd/ceremony_team_checkin_test.go` (38), `cmd/continue_fastpath_castes_test.go` (26), `cmd/phase_verified_once_test.go` — `TestPhaseVerifiedOnce` (225).

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- The proposal pipeline already exists end to end: `--castes` + `--caste-reason` → `queenApplyJudgement` → `caste_decision` in the manifest → wrapper relays `caste_decision.summary`. This phase changes what the floor contains and adds per-worker reasons; it does not build a new channel.
- Per-caste reason slots already exist in the dispatch contract (`selected_reasons`, `pruned_reasons`) and are already rendered by the check-in card and `codex_visuals.go` (~3927). Today they are filled with score arithmetic; D-09 fills them with sentences.
- The check-in card (`aether ceremony team-checkin`) already shows REQUIRED/OPTIONAL, per-caste reasons and a "Not sent" list; the owner's trim re-fetches the manifest and records a preference via `memory-capture`. The waive choice (D-03) follows the same shape, recorded through `decision-answer`.
- Relevance refusal (`casteRelevanceScore == 0` ⇒ refused) already guards proposals against zero-work specialists; it stays.
- Phase 193's single verification pass (`runDeterministicFloor`, build-time watcher resolved once, `TestPhaseVerifiedOnce`) is the boundary the forced reviewer attaches to.
- The policy-file override for keyword lists (`loadReviewDepthPolicy`) is an existing extension point for per-project signal words.

### Established Patterns
- Assert the spawn list, not the decision record (`TestQueenChoiceReachesTheDispatchList`): the manifest once recorded a Queen choice that `applyBuildDispatchPolicyCastes` deleted one line later. Every "X is sent / not sent" claim in this phase must be asserted on the dispatch list.
- Fail-then-pass: reproduce the field failure first (the 8-worker bug fix as a fixture) and make the named test fail before the fix.
- Invariants over named sections; a test that can only pass proves nothing (the v1.26 lesson).
- No test leaves the repo without a ledger disposition (RETIRE-04).
- Wrapper triplets are byte-identical by test; edit all three copies.
- Plain English on every owner-facing string; reasons must pass the "someone who has never opened a file" check.

### Integration Points
- Build (`plannedBuildDispatchesWithJudgement`, `queenCasteDecisionSummary`) and both continue lanes (`runCodexContinueReview` in-process; `plannedExternalContinueDispatches` + `continue-finalize` wrapper lane) — the 2026-08-21 review gate found the in-process lane half-wired; every rule must hold on both.
- `build-finalize` persists the forced-reviewer record; `continue` reads it (D-05).
- The check-in card (D-03, D-10, D-14); Phase 196 (cost line, model + reason) and 197/198 (cards) consume the reasons this phase records.
- `aether run` autopilot (D-11 fallback) and `/ant-quick`.
- CLAUDE.md "Queen-Owned Orchestration" / "Team Check-In" sections, `.aether/docs/` wrapper contract, and the command guide — documentation must be corrected, not extended.

</code_context>

<specifics>
## Specific Ideas

- The owner's ruling words (2026-08-22): "I don't even necessarily want a watcher every time… it's meant to use the model's intelligence to delegate roles."
- Measured baseline: a 1-task bug fix is sent 8 workers today (build: builder, watcher, auditor, probe, tracker; continue: watcher, auditor, probe); v5.4.0 sent 3–4. 193 removed the duplicate verification dispatch; this phase removes the rest. Target: one worker plus free checks, both paths.
- The card must say the signal every time a reviewer is forced: "security reviewer — this touches logins (the plan mentions 'password reset')".
- On the risk list the owner said "I don't know" twice; Claude chose the five from D11 and promised it is reversible and visible on every firing. Do not re-ask; do change it if a firing looks wrong.
- The owner wants the pre-build pause kept even for a one-worker team — control over speed there.
- Example verdicts the wrapper already teaches (build.md): "change the button copy from Submit to Save" → nobody extra; "let people stay signed in between visits" → security reviewer (that is session handling); "swap the payment provider" → security reviewer (money and credentials).

</specifics>

<deferred>
## Deferred Ideas

- A spec-writing command (`/ant-spec` or an `/ant-discuss` extension) that hands the owner a readable specification to sign off before building — the owner first asked to fold it in, then agreed to keep it parked: new user-facing capability, listed in REQUIREMENTS.md "Future Requirements" (spec §9). Not this milestone.
- Per-project custom risk words with their own UI — the policy-file override exists; no new surface in this phase.
- Rendering of forced-reviewer and per-worker reasons on the cost line → Phase 196; on the closing and session-start cards → Phases 197/198.
- Grouping several tasks into one worker's job → Phase 195.

### Reviewed Todos (not folded)
- `2026-08-20-spec-builder-feature.md` — see above; deferred to v1.28+.
- `2026-08-01-ts-host-preflight-hardcoded-timeout.md` — TS-host probe timeout; matched on generic words only, unrelated to team selection.

</deferred>

---

*Phase: 194-the-queen-decides-the-team*
*Context gathered: 2026-08-23*
