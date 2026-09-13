---
phase: 203-biological-runtime
reviewed: 2026-09-13T21:40:00Z
depth: standard
files_reviewed: 196
parts:
  - 203-REVIEW-part-a.md
  - 203-REVIEW-part-b.md
  - 203-REVIEW-part-c.md
  - 203-REVIEW-part-d.md
findings:
  critical: 5
  warning: 13
  info: 6
  total: 24
status: issues
---

# Phase 203: Biological Runtime — Code Review

**Reviewed:** 2026-09-13
**Depth:** standard
**Scope:** 196 files (69 non-test Go sources, 69 Go test files, 5 TypeScript host sources + 7 host test files, 18 docs, 10 fixture/ratchet JSON), derived from the 15 plan SUMMARY `key-files` lists cross-checked against `git diff 8c6b9257^..HEAD`.
**Method:** four parallel reviewers over disjoint partitions — (a) recruitment core, admission, dispatch, results, platform adapters; (b) pheromone bus, influence history, trophallaxis, credit; (c) live surfaces, status/visuals, and the wiring evidence; (d) build/continue/plan lifecycle and the shipped documentation claims. Full evidence in the four part files listed above.

## Verdict

The phase's mechanical core is genuinely built and largely well-proven. `bindRecruitmentResult`'s exactly-once/replay/generation logic is sound; the "validate, record, then decide" durability ordering is real; workspace containment and process-group-safe timeout in `dispatchRecruitment` are correctly wired; the three cost/no-regression tests (`TestNoRecruitmentPathCostsNothing`, `TestNoNewMandatoryStep`, `TestTuningPassIsFreeWithoutCredit`) measure real call counts, AST reachability and file existence rather than anything tautological; and all 17 tests CLAUDE.md's Biological Runtime section cites exist and assert what the prose beside them says.

Against that, five Critical findings. Two are authorization fail-opens. Two are gaps between a shipped claim and what the wiring actually delivers — this project's named signature failure mode. One is a durability/history gap that makes a requirement's headline claim false for three of its eight declared actions.

## Critical

### CR-01 — The host/autopilot recruitment lane skips four of the five admission dimensions the native lane enforces
`cmd/recruitment_admission.go:39-57`, `.aether/ts-host/src/spawn-orchestrator.ts:1-20,108-142`, `cmd/recruitment_lane_test.go:298-360` · *part a*

`spawn-orchestrator.ts`'s own header comment, and `203-CLASSIC-SYNTHESIS.md:206` (acceptance test SYN-203-02), both state that a host-lane and a native-lane recruitment against the same ledger state produce the same allow/deny answer. They do not. `processClaims` calls `aether spawn-can-spawn`, which runs under origin `spawnOriginSpawnCanSpawn` — mapped to an **empty** check list. Permission, path, cost and duplicate are therefore never evaluated on that lane; only depth, whole-run budget and ancestor-cycle are shared. The test whose name most resembles a parity check, `TestBothLanesUseOneReasonVocabulary`, asserts the *opposite* — it fails if that origin's check list is ever non-empty. `203-09-SUMMARY.md:51` records the divergence as deliberate, but it was never reconciled with the synthesis document's written acceptance criterion or the shipped source comment.

**Failure scenario:** a read-only caste requesting a write workspace is refused with `reason: "permission"` via `aether recruit`, and admitted via the autopilot/host lane.

### CR-02 — Depth cap is bypassable by self-assertion (fail-open authorization)
`cmd/spawn.go:22`, `cmd/recruitment.go`, `cmd/recruitment_admission.go` · *part a* · tracks `.planning/WINDOWS.md` #18

`spawnParentIsRoot` matches an unauthenticated, caller-supplied `--parent` string against the fixed sentinel list `{Queen, Prime-1, Swarm}` and grants depth 0 with `DepthIsAuthoritative=true`, requiring no spawn-tree entry. `aether recruit --parent Queen` therefore passes the depth check regardless of the caller's real depth, defeating `spawnMaxDelegationDepth` — the runaway-spawn and cost control. Confirmed still present in current code; pre-dates Phase 203 (inherited from Phase 173) but is now reachable through one more command than before. Everything else on that path is correctly fail-closed.

### CR-03 — BIO-08's "append-only history with a recorded actor" is false for three of its eight declared actions
`cmd/pheromone_approval.go`, `pkg/colony/colony.go:347-353` · *part b*

`approvePendingNote`, `editPendingNote` and `rejectPendingNote` never call `appendInfluenceHistory` and record no actor at all. The codebase's own comment on `PendingSuggestion.Action` concedes it is "a single scalar record, not that history itself." `TestEveryDeclaredActionIsReachable` only checks the function and flag exist — never that an action produces a history entry. That is a false certificate for the requirement's headline claim.

### CR-04 — Owner-only pheromone actions are gated by a self-reported flag that defaults to "owner"
`cmd/pheromone_mgmt.go:417` · *part b*

Revoke, appeal, pin and unpin are authorized purely by `--actor`, which defaults to `"owner"` with no caller authentication. Any process that can run `aether` — including an ordinary dispatched worker — can revoke a permanent REDIRECT constraint, or arm/disarm outcome tuning, by simply omitting the flag.

### CR-05 — `aether recruit` is still unreachable from the native/direct build lane and from Codex-platform workers
`cmd/codex_build.go:4733` (sole caller of `renderRecruitmentInvitation`), `renderCodexBuildWorkerBrief`, `.codex/agents/*.toml` · *part c*, corroborated by *part d* WR-01

The acceptance test `TestNoRegisteredSubcommandIsUnreferenced` genuinely stopped being a false positive — the allowlist was not widened, and `.claude/agents/ant/*.md` really are loaded as worker system prompts. But the invitation text reaches a worker's prompt on **one** lane only. It never reaches: workers dispatched by `executeCodexBuildDispatches` (the native/direct lane, which autopilot uses), any of the 27 Codex agent definitions (0 mention "recruit"), or any worker dispatched during `aether continue` (Watcher, Gatekeeper, Auditor, Probe). No test exercises those call chains, so the gap is invisible to the current gate — and `TestTheRecruitInstructionHasOneSource` structurally guarantees the single emitter stays single. CLAUDE.md's generic framing ("A helper working on a piece of the job can now ask the program for backup") overstates the delivered scope.

## Warning

| # | Finding | Location | Part |
|---|---------|----------|------|
| WR-01 | Outcome tuning can double-apply a strength step when the signal save succeeds but the paired history-append fails — the record is not marked processed on that path, violating its own stated idempotency guarantee | `cmd/pheromone_outcome.go:302-323` | b |
| WR-02 | Re-importing the same pheromone export creates duplicate signal IDs and permanently orphans the duplicate's quarantine flag (nothing can clear it) | `cmd/exchange.go` | b |
| WR-03 | `trophallaxisSanitizePacketText` skips `ChangedFiles` and `CommandsRun` despite claiming to sanitize "every free-text field" — both are worker-authored | `cmd/trophallaxis.go` | b |
| WR-04 | A failed skip-history write for pinned/revoked notes is silently swallowed and never retried | `cmd/pheromone_outcome.go` | b |
| WR-05 | `aether status`'s Classic-voice guarantee still measures `renderLifecycleStatus`, which the real status command never calls (it renders via `renderDashboard`) — confirmed still true | `cmd/status.go`, `cmd/lifecycle_status_render.go` | c |
| WR-06 | `renderPlanningStopVisual` remains a fully orphaned renderer with zero non-test callers — that screen is never drawn | `cmd/planning_visuals.go:550` | c |
| WR-07 | The reachability ratchet and wiring gate can silently not execute in CI without the run being marked red — a live risk for exactly the tests this review's CR-05 verdict depends on | `.github/workflows/ci.yml`, `cmd/testing_main_test.go` | c |
| WR-08 | Continue/check-time workers are never told `aether recruit` exists (narrower restatement of CR-05; fix or narrow the doc claim) | `cmd/codex_continue.go` | d |
| WR-09 | Deliberately retiring a plan candidate is persisted with `"candidate_expired"` — the same failure reason as an unattended timeout — mislabeling an owner's explicit rejection; untested | `cmd/plan_revision.go:1182` | d |
| WR-10 | Wrong reason code (`"scope"` instead of `recruitmentReasonUnresolved`) on a recruitment-intent durable-write failure | `cmd/recruitment.go:147` | a |
| WR-11 | The TypeScript native-nesting probe `recruitment-probe.ts` has no caller anywhere in the host (src or dist) — an orphan | `.aether/ts-host/src/recruitment-probe.ts` | a |
| WR-12 | `recruitmentEvidenceAltered` reads evidence file paths from disk with no path-containment check (currently unreachable, but unguarded) | `cmd/recruitment_recovery.go` | a |
| WR-13 | The "no recruitment cost" AST scan relies on two hand-maintained symbol lists; a new recruitment/credit helper not added to them escapes the scan silently, weakening the guarantee without failing any test | `cmd/recruitment_latency_test.go:37-47`, `cmd/recruitment_test.go:432-450` | d |

## Info

| # | Finding | Part |
|---|---------|------|
| IN-01 | `-Inf` signal strength is excluded under the "below-floor" reason rather than "malformed" | b |
| IN-02 | The prompt-injection heuristic phrase list is narrow | b |
| IN-03 | `workerDisciplineCallerFiles`'s comment and the family-tree / decision-changed lines are sound and correctly wired — no action needed, recorded for completeness | c |
| IN-04 | `recruitmentRecoveryNextAction`'s placeholder argument counts are correctly matched per recovery class | c |
| IN-05 | `TestCLAUDEMDBiologicalRuntimeClaimsCiteLiveTests` only proves a cited test name resolves to a real function, not that the function asserts the claim beside it — a shared limitation of this ratchet pattern, not new to this phase | d |
| IN-06 | `planningStageResumeWorkerSpec` passes a literal `""` goal into `planningWorkerSpecsForGoal`, correct only because that parameter is currently ignored | d |

## What the review confirms is sound

- Exactly-once result binding, including replay, conflict and execution-generation handling (`bindRecruitmentResult`).
- The durable-intent-before-decision ordering, and the refusal-never-aborts-the-parent contract.
- Workspace containment and process-group-safe termination on dispatch.
- The three cost/no-regression measurements — real counters and AST scans, not tautologies.
- All 17 CLAUDE.md Biological Runtime test citations resolve to real, assertive tests; `.claude/rules/aether-colony.md` and `.aether/rules/aether-colony.md` are byte-identical and consistent with CLAUDE.md.
- The `codex_plan_stage_resume.go` mechanism is genuinely called from `runCodexPlanPlanOnly` and its tests drive the real coordinator, not fixtures.
- The orphan allowlist was **not** widened to make the acceptance test pass.

---

_Reviewers: four parallel gsd-code-reviewer agents. Full per-finding evidence in the four part files._
