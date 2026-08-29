# Phase 198: Put the Thrown-Away Data Back on Screen - Research

**Researched:** 2026-08-29
**Domain:** Go CLI runtime display/rendering (cmd/ package) — no external libraries, no web research
**Confidence:** HIGH (all claims below are `[VERIFIED: cmd/<file>.go:<line>]` from a Read this session, or `[CITED]` from a canonical planning doc already required reading; no `[ASSUMED]` claims — see Assumptions Log)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** Each verification check in `continue` emits a start line when it begins and a result line when it ends — e.g. `Running tests…` then `Tests ✓ 12/12 (4s)`. Two lines per check, append-only, in the one terminal (per SEE-03). Never a single in-place updating line.
- **D-02:** A failed check's result line carries a one-line plain-English reason inline (`Tests ✗ 2 of 12 failed — login test, export test`). Full detail waits for the closing summary.
- **D-03:** Reviewer workers (Watcher, Auditor, Probe, Gatekeeper…) get the same live start/finish treatment, and the finish line carries the duration and tool-call count (`Watcher Keen-12 done (3m 10s, 14 tool calls)`). These are the same duration/tool-count figures SHOW-02 restores in the summary; live lines and summary must read from one source.
- **D-04:** Before asking "finish this project?", seal shows a short state-of-play card: phases done vs total, any checks still failing, any open warnings/flags, and what finishing will do (archive, pool lessons into the hive). Not the full closing summary, not a bare y/n.
- **D-05:** The wisdom review ("what did we learn") runs before the confirmation question. Its lessons are recorded even if the owner then says no; saying no leaves the project unsealed.
- **D-06:** If something is failing or unresolved at seal time, the card names it and asks a second, explicit question (`Finish anyway with 2 checks failing?`). A yes is recorded together with the named problem, through the same owner-decision mechanism as reviewer waivers — never inferred from card wording. Seal is never silently refused and never warned-only.
- **D-07:** Autopilot never seals. It runs up to the last phase, then stops and hands the owner the finish command. Reversibility: costly — a later "hands-off to the end" feature would have to overturn this in writing, and the seal-confirmation test must fail if autopilot ever reaches the seal path.
- **D-08:** When a build starts with a blocker present, the program prints a plain-English heads-up naming it and asks one question: carry on, or stop and deal with it. It proceeds as answered. Not warn-and-continue, not refuse.
- **D-09:** "Blocker" means hard stops only: the last `continue` on this phase ended blocked/failed, a forced reviewer waiver still waiting on the owner, or an open owner question (the same three "waiting on you" records Phase 197's check-in logic already recognises). FOCUS/FEEDBACK notes and ordinary flags do not trigger it. `--no-checkin` / autopilot paths print the heads-up and continue without asking (non-interactive by ruling).
- **D-10:** Every restored item appears every time, in the compact house style (SEE-12: emoji heading + plain-English parenthetical, one line per item, nested `└──` detail capped per category with an honest `(+N more)`). No separate "show more" command.
- **D-11:** Evidence lines read requirement + what proved it: `✓ Login works — proved by: 3 tests passed, auth.go present`. Never a bare tick.
- **D-12:** The chat path and the direct path render one screen from one renderer: `ceremony closeout` must produce byte-equal output to the direct visual for plan, continue and seal (`TestWrapperPathRendersSameCeremonyAsDirectPath`). The fix is structural — the finalizers' JSON result map is the single source and the closeout renders all of it — not a second copy of each finalizer's screen.
- **Carried forward — binding, not re-asked:**
  - SEE-03 (2026-08-16): one terminal, append-only lines, no tmux/second window, no repaintable panel. Live progress in D-01/D-03 goes through `emitVisualProgress` / `writeVisualOutput`.
  - SEE-12 (2026-08-16): headed sections not machine tables; `TestHumanDisplaysUseHeadedSectionsNotMachineTables` and its allowlist still apply to every new render.
  - "Verified safe-to-clear line" stays; the old "clear context now?" prompt stays out. SHOW-05 locks all three drops with a test.
  - Phase 196 D-01: duration/tool-count/token figures are measured or shown as `—  not reported`; never estimated.
  - Phase 197 S-01/S-04/S-05: next-step commands come from the shared "what next" logic and are translated per platform only in the visual writer; wrapper triplets are edited byte-identically; all owner-facing text is plain English with repo words translated inline.

### Claude's Discretion

- Exact wording, ordering and emoji of every heading and line, within SEE-12.
- How many "recent decisions" the resume card shows and how the drift note is phrased.
- Where the per-check progress hook lives inside the continue verification loop, provided both the default fast path and the heavy-review path emit it.
- Test design for `TestRenderedVisualsShowEveryCarriedField`: it must be an invariant over the finalizer result map (every carried key is rendered or explicitly allow-listed with a reason, allowlist shrink-only), not a named-section check.

### Deferred Ideas (OUT OF SCOPE)

- Spec builder command (`.planning/todos/2026-08-20-spec-builder-feature.md`) — new command, parked for v1.28+.
- ts-host preflight timeout hardcoded — unrelated to display.
- Any repaintable panel, second terminal, or bordered table (v1.27 "explicitly not in this milestone" list).
- Changing what any command *does*; new commands or menu entries.

</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SHOW-01 | Chat path renders the same ceremony as direct path for plan/continue/seal | Q1/Q2 below: the exact structural gap (`closeoutCompletionDetails` cherry-picks fields; `outputWorkflow` discards the already-computed richer visual in JSON mode) and the fix shape |
| SHOW-02 | Data already carried is rendered: per-check report, evidence lines, worker durations/tool counts, plan confidence/iterations, resume per-phase progress, recent decisions, drift note, specialist findings, build-start blocker advisory | Q1, Q4, Q5, Q6 — exact struct fields, which are carried-but-unrendered vs. genuinely absent |
| SHOW-03 | `continue` shows live verification progress lines as each check runs | Q3 — exact function (`runVerificationStep`) and exact shared body (`runDeterministicFloor`) both lanes call |
| SHOW-04 | Seal asks for confirmation and runs the wisdom review before sealing | Q7 — exact call order in `completeSealRuntime`, owner-decision mechanism to reuse |
| SHOW-05 | Deliberately-dropped decisions stay dropped, test-locked | Q9, Q10 — confirmed clean (no tmux/box-drawing residue); exact function housing the kept line |

</phase_requirements>

## Summary

This phase is a pure Go-codebase archaeology and rendering-consolidation problem — no third-party libraries, no external services. Every one of the eleven research questions resolved to a specific, previously-unlisted file:line, and the picture that emerges is consistent across all of them: **the runtime already computes almost everything SHOW-02 asks to restore; the loss happens at exactly two seams** — (1) `outputWorkflow` (cmd/codex_visuals.go:362-371) discards the finalizer's own richly-rendered visual string entirely when `AETHER_OUTPUT_MODE=json`, and (2) `closeoutCompletionDetails` (cmd/closeout_cmd.go:86-200) re-derives a much thinner `result` map from the raw completion JSON, extracting only ~15 of the 20-40 keys each finalizer actually writes, which `renderCeremonyCloseoutVisual` (cmd/ceremony_cmd.go:544-653) then renders even more thinly than that.

The single structural fix implied by D-12 and by the code itself: stop re-deriving a second, thinner `result` map in `closeoutCompletionDetails`, and instead feed the **whole raw completion JSON** (already saved to disk, already structurally identical after a JSON round-trip to what the in-process render call receives — confirmed by reading `renderContinueWorkerFlowValue`'s existing dual-type handling, cmd/codex_visuals.go:2408-2435, which already branches on `[]codexContinueWorkerFlowStep` vs `[]interface{}` for exactly this reason) to the **same** `renderPlanVisual` / `renderContinueVisual` / `renderSealVisual` functions the direct path calls, rather than to the separate, narrower `renderCeremonyCloseoutVisual`. This makes `TestWrapperPathRendersSameCeremonyAsDirectPath` true by construction rather than by keeping two renderers in sync by hand.

For SHOW-03's live per-check lines, `runVerificationStep` (cmd/codex_continue.go:3208-3282) is the single function invoked by `runDeterministicFloor` (cmd/deterministic_floor.go:48-107), which is itself explicitly documented and tested (`TestBothContinueLanesApplyTheSameFloor`) as the one shared body both the in-process fast lane (`runCodexContinueVerification`) and the wrapper/heavy lane (`runCodexContinueVerificationSnapshot`, cmd/codex_continue_plan.go:264) call. Placing the start/finish emit inside `runVerificationStep` structurally guarantees both lanes stream it — the exact "guarantee that holds on both lanes" CLAUDE.md and D-01..D-03 require. `codexVerificationStep` currently has **no duration field at all** — this is a genuine new addition, not a rendering fix.

For SHOW-04 (seal), `completeSealRuntime` (cmd/codex_workflow_cmds.go:558-786) already runs the wisdom review (`runSealConsolidation()`, line 649) — but **before any confirmation exists at all**: the `sealCmd` RunE (cmd/codex_workflow_cmds.go:485-535) goes straight from a blocker check to `completeSealRuntime`, with no confirmation step of any kind on the runtime side (confirmation today lives only in wrapper prose asking the user to type `--force`). D-04/D-05/D-06 require inserting a genuine confirmation gate between the wisdom review and the state mutation (`state.State = colony.StateCOMPLETED`, line 688) — the wisdom review must run first and its results must be recorded even on a "no" answer, which requires reordering nothing (consolidation already runs first) but does require a **new stop-and-ask point** that does not exist today.

For SHOW-05, the codebase is clean: no `tmux` references, no box-drawing characters in any renderer, and the "verified safe-to-clear" line lives at exactly one function, `renderContextClearGuidanceForPlatform` (cmd/codex_visuals.go:633-653) — a durable test target.

**Primary recommendation:** Do not "restore" SHOW-02's items by hand-patching `renderCeremonyCloseoutVisual`'s existing narrow sections one at a time. Instead, make `ceremony closeout` call the identical finalizer-specific renderer the direct path calls (with the finalizer's raw result map, JSON-round-tripped), and treat any field that renderer does not yet show (worker tool-count in the summary table, resume per-phase progress, resume recent-decisions/events, per-check verification detail beyond a tally) as a **new rendering addition made once**, shared by both paths by construction — because several of these (worker duration+tool-count table, per-check names in the verification summary) are missing on the *direct* path too, not merely on the chat path.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Per-check verification report (build/types/lint/tests) | Go runtime (cmd/deterministic_floor.go, cmd/codex_continue.go) | — | Deterministic floor is computed entirely in-process; no chat/host boundary involved |
| Live check start/finish lines | Go runtime (cmd/codex_continue.go `runVerificationStep`) via `emitVisualProgress` | Wrapper (renders nothing extra; the Go process's own stdout streams to the chat) | Streaming exit is one function (`writeVisualOutput`); wrapper only decides whether to run the fast (`aether continue`) or heavy (`aether host continue --dry-run`) path, never re-implements the checks |
| Reviewer worker duration/tool-count | Go runtime (`emitCodexDispatchWorkerFinished`, cmd/codex_build_progress.go) for in-process dispatch; **Wrapper-reported** completion packet fields (`codexExternalBuildWorkerResult.Duration/ToolCount`) for wrapper-spawned reviewers | — | A wrapper-spawned reviewer's tool-call count can only be reported by the wrapper (the Go process never sees the platform Agent tool's internal tool calls); trust boundary already exists at `mergeExternalBuildResults` |
| Plan confidence/iterations | Go runtime (`runCodexPlanFinalize`, cmd/codex_plan_finalize.go) | — | Computed entirely server-side; already serialized in the finalizer's `result["confidence"]` / `result["planning_loop"]` |
| Resume per-phase progress / recent decisions / drift | Go runtime (`buildResumeDashboardResult`, cmd/context.go) | — | `state.Memory.Decisions` and `state.Plan.Phases` are already loaded server-side; no host/chat involvement |
| Seal confirmation + wisdom review | Go runtime (`completeSealRuntime`, cmd/codex_workflow_cmds.go) | Wrapper (renders the card, relays the owner's typed answer back as a flag/decision) | State mutation (`state.State = colony.StateCOMPLETED`) must never happen before the owner's answer is recorded; the Go process is the only place that can enforce ordering |
| Build-start blocker advisory | Go runtime (must compute the "three waiting on you" predicate before dispatch) | Wrapper (renders the one-question prompt, relays "carry on"/"stop") | `buildHasPendingOwnerDecision` (cmd/ceremony_team_checkin.go:465) already exists as the trusted predicate for two of the three signals; the third ("last continue blocked") needs a new read of `continue.json`'s `Advanced` field |
| Chat-path vs direct-path parity (D-12) | Go runtime (`ceremony closeout` must call the same renderer) | Wrapper (passes the completion file path only; never re-renders) | The wrapper today only orchestrates dispatch; all rendering already claims to be runtime-owned (`.aether/docs/wrapper-runtime-ux-contract.md`) — the gap is a second, thinner Go renderer, not a wrapper doing rendering |

## Findings

### Q1 — Finalizer result maps: carried vs. rendered

**Plan finalize** (`runCodexPlanFinalize`, result map built at cmd/codex_plan_finalize.go:444-478, and the mid-loop `requires_next_iteration` variant at cmd/codex_plan_finalize.go:863-882):

| Key | Populated by | Rendered on DIRECT path? | Rendered by `ceremony closeout`? |
|---|---|---|---|
| `phases`, `count` | codex_plan_finalize.go:449-450 | Yes — `renderPlanVisual` cmd/codex_visuals.go:1506-1508 ("Plan size: N phases") | Partial — `writeCeremonyPlanSummary` (cmd/ceremony_cmd.go:719-798) only if `result["completion_phases"]` or `result["state_phases"]` present; `closeoutCompletionDetails` populates `completion_phases` from `ceremonyPhaseSummariesFromCompletion(raw)` (cmd/closeout_cmd.go:195-198) — present but the task/hint detail differs from direct's own phase task rendering |
| `confidence` (map: `overall`) | codex_plan_finalize.go:455 | Yes — codex_visuals.go:1488-1490 ("Confidence: N% overall") | **No** — `closeoutCompletionDetails` never extracts `confidence`; only `plan_confidence_percent` from **live COLONY_STATE** (`state.Plan.Confidence`), a different, later-computed value, is used (cmd/ceremony_cmd.go:239-241) |
| `planning_loop` (Iterations/TargetConfidence/MaxIterations/StopReason) | codex_plan_finalize.go:456 | Yes — codex_visuals.go:1491-1505 ("Planning loop: target N%, X/Y iteration(s), stop=…") | **No** — no key in `closeoutCompletionDetails`'s extraction list at all |
| `gaps`, `research_warning`, `planning_warning`, `grounding_warnings` | codex_plan_finalize.go:466-491 | Yes — various sections in `renderPlanVisual` | **No** |
| `dispatch_mode`, `artifact_source`, `plan_source` | codex_plan_finalize.go:469-472 | Provenance-only, not directly rendered as prose but used to select messaging | **No** |
| `unresolved_clarifications`, `clarification_warning` | codex_plan_finalize.go:475 (+ set elsewhere) | Yes — codex_visuals.go:1514-1519 | **No** |
| `next` | codex_plan_finalize.go:477 | Feeds `renderNextActionCard`/closing card | Yes, via `completion_next` → `result["next"]` (cmd/closeout_cmd.go:135-137) — one of the few keys that DOES survive |
| `orchestrator_boundary_guidance` (via `addOrchestratorBoundaryGuidance`) | codex_plan_finalize.go:493 | Yes | Partial/unclear — not in `closeoutCompletionDetails`'s explicit key list; would need to be added |

**Continue finalize** (block path result map cmd/codex_continue_finalize.go:1184-1211; advance path cmd/codex_continue_finalize.go:1324-1351):

| Key | Populated by | Rendered on DIRECT path (`renderContinueVisual`/`renderContinueBlockedVisual`)? | Rendered by `ceremony closeout`? |
|---|---|---|---|
| `verification` (`codexContinueVerificationReport`: `Steps[]`, `Claims`, `Watcher`, `Criteria[]`, `Warnings[]`) | codex_continue_finalize.go:1193, 1332 | Partial — `renderContinueVerificationSummaryMap` (codex_visuals.go:2380-2406) shows only a **tally** ("Verification: N passed, M skipped" + claims summary), never per-check names, commands, or the `Criteria[]` evidence list | **No** — `closeoutCompletionDetails` never extracts `verification` at all; `renderCeremonyCloseoutVisual` has no verification section |
| `gates` (`codexContinueGateReport`: `Checks[]gateCheck{Name,Passed,Detail,FixHint}`) | codex_continue_finalize.go:1196, 1335 | Partial — `renderContinueGateSummaryMap` (codex_visuals.go:2514-2530) shows only "Gates: N/M passed", never per-gate names (`manifest_present`, `verification_steps_passed`, `implementation_evidence`, `owner_confirmation_pending`, `anti_pattern`, `charter_compliance` — cmd/codex_continue.go:3437-3644, cmd/gate.go) | **No** |
| `task_evidence` (`[]codexCriterionVerification`: `Criterion, Evidence[], Summary, State`) | codex_continue_finalize.go:1195 (assessment.Tasks) | **Not directly rendered as evidence lines** in `renderContinueVisual` (only "Operational evidence" list from `operational_issues`, a different field) | **No** |
| `worker_flow` (`[]codexContinueWorkerFlowStep`: `Name, Caste, Status, Summary, Findings[], Duration float64, Usage(unexported)`) | codex_continue_finalize.go:1200, 1342 | Yes — `renderContinueWorkerFlowValue` (codex_visuals.go:2408-2435) shows caste/name/status/summary + capped findings/recommendations/weak-spots/edge-cases/blockers, but **never Duration and never a tool count (no field exists)** | **No** — `closeoutWorkerMaps` (cmd/closeout_cmd.go:222-240) only looks at keys `"dispatches"`, `"results"`, `"workers"` — it **never looks at `worker_flow`**, so continue's worker detail is invisible to closeout entirely; only a raw tally (via a different aggregation) survives |
| `review` (`codexContinueReviewReport`) | codex_continue_finalize.go:1213, 1336 | Feeds `worker_flow` merge (already counted above) | **No** |
| `signal_housekeeping` | codex_continue_finalize.go:1346 | Yes — "Housekeeping" section, codex_visuals.go:2140-2147 | **No** |
| `next_phase`/`next_phase_name` | codex_continue_finalize.go:1349-1350 | Yes — "Next Phase" section | Partial via generic `next`/`current_phase` handling |
| `recovery_instructions`, `plan_revision_option` | codex_continue_finalize.go:1205-1217 | Rendered in blocked visual's recovery section | **No** |

**Build finalize** (`codexExternalBuildWorkerResult` struct, cmd/codex_build_finalize.go:62-113, is the *worker-submitted* shape merged into the manifest's dispatch records — `Duration float64` and `ToolCount int` ARE both JSON-serialized fields here, `json:"duration,omitempty"` / `json:"tool_count,omitempty"`, lines 109-110):

| Key | Populated by | Rendered on DIRECT path? | Rendered by `ceremony closeout`? |
|---|---|---|---|
| Per-dispatch `Duration`/`ToolCount` | worker-submitted, merged by `mergeExternalBuildResults` | **Not shown in `renderBuildFinalizeVisual`'s final summary either** (cmd/codex_visuals.go:2058-2076 calls `renderSpawnPlanForDispatches` → `writeDispatchExecutionStatus`, cmd/codex_visuals.go:4417-4448, which renders status/blockers only — no duration, no tool count) | Partial — `writeCeremonyWorkerSummary` (cmd/ceremony_cmd.go:800-830) sums `tool_count` across all workers into one aggregate line ("Tools: N calls across workers") but never per-worker |

**Seal finalize** (`completeSealRuntime` result map, cmd/codex_workflow_cmds.go:775-786; `sealFinalReviewReport`, cmd/seal_final_review.go:66-84):

| Key | Populated by | Rendered on DIRECT path (`renderSealVisual`)? | Rendered by `ceremony closeout`? |
|---|---|---|---|
| `sealed`, `milestone`, `summary`, `next` | codex_workflow_cmds.go:775-780 | Yes | Yes (generic `next`/`state` handling) |
| `force_sealed`, `force_reason`, `unverified_phases`, `overridden_blockers` | codex_workflow_cmds.go:781-785 | Yes (override reporting) | **No** — not in `closeoutCompletionDetails` extraction list |
| Consolidation beats (`sealConsolidationSummary`) | rendered directly to stdout via `renderSealConsolidationBeats`/`emitSealConsolidationCeremony` (codex_workflow_cmds.go:648-649) — **not carried in the result map at all**, printed as a side effect during the call | Yes, printed once during `completeSealRuntime` | **No** — this is printed only once, synchronously, at seal time; if `ceremony closeout` runs later against a saved completion file, this text is gone forever unless captured. **Genuine structural gap: this data is never in JSON, only ever in stdout.** |
| `sealFinalReviewReport` (`Findings[]`, `PostSealBacklog[]`, `ReusableLessons[]`, `BlockingIssues[]`) | seal_final_review.go — saved to `seal/final-review.json`, referenced via `enrichment.FinalReview` in `buildSealSummary` (writes to CROWNED-ANTHILL.md) | Written to a file, not the terminal result map | **No** |

**Conclusion for Q1:** The single largest, cleanest fix is architectural (per D-12): stop building a second `result` map in `closeoutCompletionDetails` and instead pass the full raw completion JSON straight into the same `render*Visual` function each finalizer already calls internally. The remaining gaps (worker tool-count in final summary tables, per-check verification names, per-gate names, task evidence lines) are **absent on the direct path too** and need genuinely new rendering code, written once, shared by construction.

### Q2 — How the wrappers call things

Confirmed byte-identical via `diff` this session: `.claude/commands/ant/{seal,continue,plan}.md` == `.opencode/commands/ant/{seal,continue,plan}.md` (exit 0, no diff output for all three).

Exact lines (all `[VERIFIED: file:line, read this session]`):

- **Plan** (`.claude/commands/ant/plan.md`):
  - Line 140: `AETHER_OUTPUT_MODE=json aether plan-finalize --completion-file <completion_file>`
  - Line 148: `AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow plan --completion-file <completion_file>`
  - Line 143: branches on `requires_next_iteration: true` from the JSON result to decide whether to loop instead of closing out.
- **Continue** (`.claude/commands/ant/continue.md`):
  - Line 59: fast/default path is `AETHER_OUTPUT_MODE=visual aether continue --verification-depth standard $ARGUMENTS` — **note: the default path never touches `continue-finalize` or `ceremony closeout` at all**; it is one single visual-mode call to `aether continue` directly.
  - Line 156: heavy path fetches the manifest via `aether host continue --dry-run --classic-ceremony $ARGUMENTS` (TS host, dry-run only).
  - Line 196: `AETHER_OUTPUT_MODE=json aether continue-finalize --completion-file <completion_file>` (heavy path only).
  - Line 202: `AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow continue --completion-file <completion_file>` (heavy path only).
  - **Implication for D-12/SHOW-01:** `ceremony closeout` for continue is reached ONLY on the heavy-review path. The default/fast continue path already renders the full direct visual (`renderContinueVisual`) in-process, so it is unaffected by the closeout gap — SHOW-01/D-12 for continue matters specifically for the heavy-review (wrapper-spawned reviewer) flow, which is the flow that already carries the richest `worker_flow` data.
- **Seal** (`.claude/commands/ant/seal.md`):
  - Line 26: `aether host seal $ARGUMENTS` (manifest generation, TS host — **different from build/continue's "never touches host" default**: seal's default hosted flow always goes through the host).
  - Line 15: raw bypass is `AETHER_OUTPUT_MODE=visual aether seal $ARGUMENTS` (only when user explicitly asks for "raw/exact/direct/no-orchestration").
  - Line 141: `AETHER_OUTPUT_MODE=json aether seal-finalize --completion-file <file>`.
  - Line 145: `AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow seal --completion-file <completion_file>`.

**`--completion-file` contents and loading:** The wrapper writes the finalizer's raw output (or a hand-assembled `{seal_manifest, dispatches}` JSON before calling `-finalize`) to a temp file. `closeoutCompletionDetails` (cmd/closeout_cmd.go:86-116) reads it, and if the top-level JSON has a `"result"` key that itself looks like a manifest/worker envelope, it substitutes `raw = nested` (line 112-116) — i.e., the completion file convention already supports a `{result: {...}}` wrapper shape, which is exactly the shape `outputOK(result)` (JSON mode) produces for a finalizer call.

**Parity tests found (grep for opencode/claude parity):**
- `cmd/lifecycle_wrapper_contract_test.go:135` — `TestLifecycleFlatMirrorsMatchCanonical`: asserts every `.claude/commands/ant/*.md` is byte-identical to its "flat mirror" (`~/.claude/commands/ant-<verb>.md`, the shape `aether install` writes — see `flatMirrorPath`, cmd/lifecycle_wrapper_contract_test.go:32). This is a **third** copy beyond `.claude`/`.opencode`, produced by an install step, not hand-maintained, but the flat mirror committed under test must still match byte-for-byte.
- `cmd/seal_wrapper_ceremony_test.go:10`, `cmd/plan_wrapper_ceremony_test.go:10`, `cmd/continue_wrapper_ceremony_test.go:13` — `Test{Seal,Plan,Continue}WrapperCeremonyContract`: assert required substrings present in **both** `.claude` and `.opencode` copies (loop over both paths), including the exact `AETHER_OUTPUT_MODE=json aether {seal,continue,plan}-finalize --completion-file` and `AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow {seal,continue,plan} --completion-file` strings.
- Non-mirror documentation (`cmd/contracts/*.md`, `colony/playbooks/*.md`) are **deliberately different formats** (flag/argument reference tables vs. stage-by-stage narrative), NOT required to be byte-identical — confirmed by diff showing substantially different content and by `TestWrapperHostContractDocumentsManifestShapes` testing prose content, not byte parity, against `.aether/docs/wrapper-host-contract.md`.

### Q3 — Streaming channel

`emitVisualProgress(visual string)` (cmd/codex_visuals.go:424-433): no-ops unless `shouldRenderVisualOutput(stdout)` AND `streamingAllowedForCurrentCommand()` are both true; trims and appends `\n\n`; calls `writeVisualOutput(stdout, visual+"\n\n")`.

`writeVisualOutput(w io.Writer, text string)` (cmd/codex_visuals.go:447-...): the single exit for every byte of human-facing visual output; performs command-name translation (raw `aether` command strings → platform slash-commands) so translation cannot be bypassed by a new call site.

**Continue verification loop — one shared body, two lanes:**

- `runVerificationStep(ctx, root, name, required, command, timeout) codexVerificationStep` (cmd/codex_continue.go:3208-3282) is the **single** function that shells out for one named check (`build`, `types`, `lint`, `tests`). It has **no start/finish emit today**, and `codexVerificationStep` has **no duration field** (struct: `Name, Command, Passed, Skipped, Required, Blocked, TimedOut, TimeoutSeconds, ExitCode, ErrorClass, Summary, Output` — cmd/codex_continue.go:33-46).
- `runDeterministicFloor(ctx, root, phase, manifest, watcher, verificationTimeout) deterministicFloorResult` (cmd/deterministic_floor.go:48-107) calls `runVerificationStep` exactly four times (lines 62-65: build/types/lint/tests) and is explicitly documented (comment, lines 33-46) and tested (`TestBothContinueLanesApplyTheSameFloor`, cmd/deterministic_floor_test.go:143) as **the single body both continue lanes call**:
  - In-process/fast lane: `runCodexContinueVerification` (cmd/codex_continue.go:1772) → `runDeterministicFloor` at line 1785.
  - Wrapper/heavy lane (via `aether continue --plan-only`, which `aether host continue --dry-run` shells out to): `runCodexContinueVerificationSnapshot` (cmd/codex_continue_plan.go:264) → `runDeterministicFloor` at line 270.
- **A third call site, `cmd/verify_out_of_band.go:187-190`, is a deliberately separate, human-operator-only command** (`aether verify-out-of-band <phase>`) with its own reachability test (`TestVerifyOutOfBandHasNoLifecycleCaller`) proving it is never invoked by `continue`/`build`. Not a one-lane violation — a third, intentionally isolated command.
- **Conclusion: the per-check start/finish hook belongs inside `runVerificationStep` itself** (or immediately wrapping its call inside `runDeterministicFloor`'s loop, lines 62-65) — this is the one place that structurally reaches both the fast and heavy lanes per `TestBothContinueLanesApplyTheSameFloor`'s own guarantee. New code must also add a `Duration` (or `time.Since`) field to `codexVerificationStep` since none exists.
- The continue.md wrapper text itself confirms the checks run exactly once per continue attempt regardless of lane: "The phase's own build/test check already ran once, before continue started — it is not re-spawned here" (`.claude/commands/ant/continue.md:90`).

**Reviewer worker live lines — already partially built:**

- `runtimeVisualDispatchObserver(spawnTree, activePrefix, wave) codex.DispatchObserver` (cmd/dispatch_runtime.go:31-50) is the observer wired into in-process reviewer dispatch (e.g., watcher dispatch during continue, cmd/codex_continue.go ~line 1552-1553 passes `runtimeVisualDispatchObserver(spawnTree, "Continue review active", wave)`), and dispatches to `emitCodexDispatchWorkerStarted` / `emitCodexDispatchWorkerFinished` (cmd/codex_build_progress.go:106-180) based on the dispatch's lifecycle status.
- `emitCodexDispatchWorkerFinished` (cmd/codex_build_progress.go:143-180) **already renders duration** on the finish line (`%.1fs`, line 170-172, from `result.WorkerResult.Duration`) but **never renders tool count** — `result.WorkerResult.ToolCount` (pkg/codex/worker.go:80, `int // Number of tool calls reported`) exists on the same struct and is simply not referenced in this function. D-03's gap for the in-process lane is a one-line addition to an existing, already-wired function — not a new mechanism.
- For the heavy-review lane, the equivalent live lines come from the wrapper's own `aether ceremony wave-start`/`worker-complete` calls (continue.md lines 182, 188), which render from a worker JSON file the wrapper writes after each reviewer returns — the wrapper is trusted to report `duration`/`tool_count` there because the Go process never sees a platform Agent tool's internal tool-call count.

### Q4 — Worker duration / tool-count storage

- **Continue's in-process worker flow:** `codexContinueWorkerFlowStep` (cmd/codex_continue.go:1568-1585) has `Duration float64 json:"duration,omitempty"` (set at cmd/codex_continue.go:1580: `step.Duration = result.WorkerResult.Duration.Seconds()`) but **no `ToolCount` field at all**, and `Usage codex.WorkerUsage` is explicitly `json:"-"` (never serialized — line ~1585, by design: "a figure that crossed a wire could be asserted by an outside caller rather than measured by the runtime"). `grep -n ".ToolCount" cmd/codex_continue*.go` returns zero matches. **This is a genuine schema gap**: D-03's "reviewer finish line carries duration and tool-call count" cannot be satisfied for the summary/continue path without adding a serializable `ToolCount int` field to `codexContinueWorkerFlowStep`, populated from `result.WorkerResult.ToolCount` alongside the existing `Duration` assignment at line 1580.
- **Build's worker-submitted (wrapper-reported) shape:** `codexExternalBuildWorkerResult` (cmd/codex_build_finalize.go:62-113) already has both `Duration float64 json:"duration,omitempty"` and `ToolCount int json:"tool_count,omitempty"` (lines 109-110) — this IS the pattern to mirror onto continue's struct.
- **Rendering:** `writeCeremonyWorkerSummary` (cmd/ceremony_cmd.go:800-830) already sums `intValue(worker["tool_count"])` across all workers into a single aggregate ("Tools: N calls across workers", line 812-818) on the closeout path — proving the closeout renderer CAN read `tool_count` when present, it just has no per-worker line, and continue's `worker_flow` array is never even scanned there (`closeoutWorkerMaps`, cmd/closeout_cmd.go:222-240, only reads keys `"dispatches"`, `"results"`, `"workers"`).
- **Phase 196's "— not reported" rule:** implemented once, in `cmd/spend_cost_line.go` (`spendCostLineFigure`, lines ~107-113; `spendNotReportedFigure` sentinel), for the TOKEN cost line specifically — this is a distinct system from worker duration/tool-count (spend ledgers explicitly have **no ToolCount field by design**: `cmd/spend_ledger.go:68-79`'s comment states ToolCount "was declared, never written and never read. Both were removed in the Phase 196 closeout... A field on a durable record that production never fills is a schema lie"). **Do not conflate the two**: SHOW-02's worker duration/tool-count table is a `codexContinueWorkerFlowStep`/`codexExternalBuildWorkerResult` concern; Phase 196's cost line is a `spendRow`/token concern. The same "never estimate, dash when absent" philosophy should extend to the new duration/tool-count rendering, but it is a new instance of the pattern, not a reuse of `spend_cost_line.go`'s code.

### Q5 — Confidence/iterations at cmd/codex_visuals.go:1488-1500

Inside `renderPlanVisual` (function starts cmd/codex_visuals.go:1399). Fed by:
- `result["confidence"].(map[string]interface{})` → `confidence["overall"]` (int) — populated at cmd/codex_plan_finalize.go:455 from a `codexPlanConfidence`-shaped value (Overall int).
- `result["planning_loop"]` — handles BOTH a typed `codexPlanningLoop` struct (direct in-process call) and a `map[string]interface{}` (JSON-round-tripped) via two type-switch branches (lines 1491 and 1498) — this dual-type handling is the exact precedent for making closeout reuse the direct renderer safely.
- `codexPlanningLoop` struct (cmd/codex_plan.go:190-202): `TargetConfidence, MaxIterations, StallThreshold, StallLimit, Accept, Iterations, StopReason, FinalConfidence, AcceptedBelowTarget, Gaps[], History[]` — all JSON-tagged, all serializable.
- Populated on BOTH the completed-plan result map (codex_plan_finalize.go:455-456) and the mid-loop `requires_next_iteration: true` result map (codex_plan_finalize.go:872-873) — so a partial/iterating plan run also carries this data, useful if the phase wants an iteration-in-progress display too (not required by SHOW-02, but available).

### Q6 — Resume card

- **Hook (`aether hook-session-start`, cmd/hook_cmds.go:333-358)**: renders `renderNextActionCard(resolveNextAction(in))` — the Phase 197 NEXT-01 minimal 8-field card ONLY. By design (comment lines 314-327) it "decides nothing" and adds no data of its own — **this is NOT the surface SHOW-02's "resume per-phase progress + recent decisions + drift note" targets**; that richer detail belongs to the separate `resume-colony`/`resume` command's fuller dashboard.
- **`resume-colony` command** (cmd/session_flow_cmds.go:176), aliased `resume` — builds its result via `buildResumeDashboardResult()` (cmd/context.go:116) and renders via `renderResumeVisual(result, handoffText, full)` (cmd/codex_visuals.go:2995).
- **`buildResumeDashboardResult` already populates, and `renderResumeVisual` never reads:**
  - `result["recent"] = {"decisions": recentDecisions, "events": recentEvents}` (cmd/context.go:276-279), where `recentDecisions := extractRecentDecisions(state.Memory.Decisions, 5)` (line 241) and `recentEvents := extractRecentEvents(state.Events, 10)` (line 244). **Confirmed by reading `renderResumeVisual` in full (cmd/codex_visuals.go:2995-3220+): it references `current`, `next_phase`, `session`, `signals`, `blockers`, `survey`, `memory_health`, `recovery`, `freshness`, `worktrees_preserved`, `stale_signals`, `worktree_gc_error` — it never once references `result["recent"]` or `result["drill_down"]`.** This is precisely the milestone brief's documented gap: "resume recent decisions (data carried, never read)" (`.planning/research/v1.27-milestone-brief.md:209`).
  - `colony.Decision` struct (pkg/colony/colony.go:777-783): `ID, Phase int, Claim, Rationale, Timestamp string` — the exact fields to render as recent-decisions lines.
- **"Resume per-phase progress" (a per-phase breakdown, not the single `current.phase`/`total_phases` fraction already shown) does NOT exist in `buildResumeDashboardResult` today** — only an overall `phase: N/M` fraction (cmd/context.go:262-269) is computed. A genuinely new loop over `state.Plan.Phases[].Status` (values: `colony.PhasePending = "pending"`, `colony.PhaseInProgress = "in_progress"`, `colony.PhaseCompleted = "completed"` — `[VERIFIED: pkg/colony/colony.go:30,32,33]`) would need to be added to the result map and rendered.
- **"Drift note"**: `grep -rn "drift"` across `cmd/*.go` and `pkg/**/*.go` returns **zero matches** for any resume/decision-drift concept (the only hit, `cmd/command_guide.go:491 intelligentCommandDriftGuards`, is an unrelated command-guide feature). This confirms the milestone brief's "resume drift note" is listed under **Gaps (absent today)**, not "carried, never read" — it is genuinely new and its exact meaning/computation is left to Claude's Discretion per CONTEXT.md ("how the drift note is phrased"). No existing data source computes a divergence signal for resume; the plan should either derive a minimal one from available data (e.g., comparing `state.Plan.Phases` structure against the last recorded plan revision, or noting time-since-last-decision) or scope it conservatively.

### Q7 — Seal path

- **Entry point:** `sealCmd` RunE (cmd/codex_workflow_cmds.go:485-535). Order today: `--plan-only` branch returns early (line 496-503); otherwise `validateSealReady` (blocker/incomplete-phase check, line 508) → `checkSealBlockers` (line 517) → if blockers and no `--force`, `renderRecoveryMenu` and **return** (line 521-523, i.e., seal refuses); if `--force`, print a WARNING line and continue (line 524-526) → build `sealOverride` (line 529) → **`return completeSealRuntime(state, override)`** (line 535). **There is no confirmation step of any kind on the runtime side today** — the only "confirmation" that exists is wrapper prose asking the user to explicitly type `--force --reason "..."` when blockers exist, which is not the D-04/D-06 state-of-play card + explicit second question this phase must add.
- **`completeSealRuntime`** (cmd/codex_workflow_cmds.go:558-786) runs, in this exact order:
  1. Snapshot instinct entries eligible for promotion (line 570: `loadActiveInstinctEntriesFromStore`).
  2. **`runSealConsolidation()`** (line 649) — this IS the wisdom review (the eight-ant curation pass). Runs unconditionally, before any state mutation.
  3. Local + hive instinct promotion loop (lines 664-703).
  4. `renderSealConsolidationBeats` + `emitSealConsolidationCeremony` printed to stdout (lines 668-670) — **printed once, synchronously; never captured into the JSON result map** (see Q1 finding — this text is lost to a later `ceremony closeout` call against a saved completion file).
  5. **State mutation**: `state.State = colony.StateCOMPLETED` (line 688), events appended (690-698), `store.SaveJSON("COLONY_STATE.json", state)` (line 700).
  6. CROWNED-ANTHILL.md written (line 741-744), lifecycle ceremony emitted, registry updated, final `result` map built and rendered (lines 774-786).
- **D-05's requirement ("wisdom review runs before the confirmation question... recorded even if the owner then says no") is satisfiable without reordering `runSealConsolidation`'s position** relative to state mutation — it already runs first. What's missing is a **new stop-and-ask point inserted between `checkSealBlockers` (or `runSealConsolidation`, per D-05's exact ordering) and the `state.State = colony.StateCOMPLETED` mutation**, which does not exist today at any point in this call chain.
- **`--plan-only`/`--dry-run` on seal**: `runSealPlanOnly` (cmd/seal_final_review.go:234-...) is a separate function entirely, called from the `sealCmd` RunE's early-return branch (line 497) — confirmed it never reaches `completeSealRuntime` and therefore never mutates state, matching CLAUDE.md's dry-run-must-not-mutate corollary structurally (the plan-only path is a structurally different function, not a flag check inside the mutating one).
- **Owner-decision recording mechanism for reviewer waivers** (to reuse for D-06's "finish anyway" recorded decision): `cmd/forced_reviewer_waiver.go` — waivers are recorded as `colony.Decision`-shaped entries with `Source: "forced-reviewer-waiver"` (line 483), matched against `Phase` and `AttemptID` (line 598), and consulted via `applyForcedReviewerWaivers` (referenced from `buildHasPendingOwnerDecision`, cmd/ceremony_team_checkin.go:467). `cmd/pending_decision.go` is the general decision-storage layer (`pending-decision-add`/`-list`/`-resolve` CLI subcommands exist per CLAUDE.md's Registry, but are pre-existing generic infrastructure, not seal-specific). **Recommendation**: D-06's "finish anyway" answer should be recorded via the same `Decision`-with-`Source`-tag pattern (e.g. `Source: "seal-force-confirmation"`), never inferred from a rendered string, matching `TestOnlyTheOwnerCanWaiveAForcedReviewer`'s precedent of "never inferred from card wording."
- **Autopilot and seal (D-07)**: `cmd/codex_run*.go` — grep confirms `codex_run.go` and `codex_run_flow.go` exist; a dedicated search for whether autopilot's phase loop ever calls `completeSealRuntime`/`sealCmd` directly should be a build-time task item (search for "seal" inside `cmd/codex_run*.go` during planning/implementation) since this file set was not read exhaustively this session — **flagged as an open item for the planner to verify with `grep -n seal cmd/codex_run*.go` before writing the D-07 lock test**, rather than asserted here as verified.

### Q8 — Blocker heads-up (D-08/D-09)

- **The three "waiting on you" records** used by `TestBuildCheckinDecisionMatrix` (cmd/ceremony_team_checkin_test.go:370) are computed by `buildHasPendingOwnerDecision(manifest codexBuildManifest) (bool, string)` (cmd/ceremony_team_checkin.go:465-478), checking in order:
  1. An unwaived forced-reviewer signal (`riskSignalHitsFromRecords` + `applyForcedReviewerWaivers`, lines 466-470).
  2. `manifest.BoundaryQuestionCount > 0` — an unanswered orchestrator boundary question (lines 471-473).
  3. `pendingHandoffDecisions(manifest.Phase)` — a worker's unanswered handoff question (lines 474-476).
  Returns `(true, plain-English-reason)` on the first hit, `(false, "")` otherwise. This function is directly reusable for two of D-09's three predicates.
- **The check-in pause decision itself**: `decideBuildCheckin(input buildCheckinDecisionInput) buildCheckinDecision` (cmd/ceremony_team_checkin.go:406-444) — evaluated in fixed order: autopilot/`--no-checkin` → non-interactive (line 407-413); explicit `--checkin` → force pause (414-420); `PendingOwnerDecision` → force pause (421-431); exactly-one-implementation-dispatch → fast path (432-438); else → default pause (439-443).
- **D-09's third predicate ("the last `continue` on this phase ended blocked/failed") has NO existing structured field.** It is recorded ONLY as a pipe-delimited event string appended to `state.Events`: `"{ts}|continue_blocked|continue(-finalize)|Continue blocked before advancement"` (cmd/codex_continue.go:4091, cmd/codex_continue_finalize.go:1169). **Recommendation: do not parse event strings** (fragile, no test coverage today asserts the string format is stable) — instead read the structured `codexContinueReport.Advanced bool` field (cmd/codex_continue.go:107-127, `json:"advanced"`) from the phase's own `continue.json` artifact (path via `continuePlanArtifactsPath(phase.ID, "continue.json")`), which is the authoritative, already-serialized source both the fast and heavy continue lanes write.
- **`--no-checkin`/autopilot non-interactive branch**: `decideBuildCheckin`'s first branch (line 407-413) — `input.Autopilot || input.NoCheckin` → `Requested: false`. D-09 requires the heads-up to still PRINT (not silently skip) on this branch, which is new behavior distinct from `decideBuildCheckin`'s existing "no pause" semantics — the print must happen regardless of `Requested`, only the *asking* is gated by it.

### Q9 — Shrink-only allowlist pattern (template for `TestRenderedVisualsShowEveryCarriedField`)

Two distinct existing patterns, both viable templates:

1. **JSON-file + frozen-baseline pattern** (`cmd/testdata/orphan_allowlist.json` + `cmd/testdata/orphan_allowlist_baseline.json`, asserted by `TestOrphanAllowlistOnlyShrinks`, cmd/subcommand_reachability_ratchet_test.go:1496-1520): the live allowlist is a JSON array of `{name, reason, owner_phase}` entries; the test loads both files and asserts every name in the CURRENT live list also exists in the frozen baseline (line 1500-1515) — i.e., new entries require a baseline edit too (an explicit, reviewed act), while entries can be silently removed as they get fixed. This is the heavier-weight pattern, appropriate for a large, slowly-shrinking list.
2. **In-test Go map literal pattern** (`TestHumanDisplaysUseHeadedSectionsNotMachineTables`, cmd/display_house_style_test.go:62-94): `allowed := map[string]string{"file.go": "reason", ...}` declared directly in the test function; any file not in the map that trips the detection pattern fails. Simpler, no baseline file, no separate freeze — appropriate for a small, stable exception set.

Given `TestRenderedVisualsShowEveryCarriedField` is scoped to "every carried key of a finalizer's result map is rendered or explicitly allow-listed with a reason, allowlist shrink-only" (CONTEXT.md, Claude's Discretion), and the exception set is likely to be small (a handful of internal/provenance-only keys per finalizer, e.g. `dispatch_mode`, `artifact_source`), **pattern 2 (in-test map literal) is the better fit** unless the exception count turns out to be large — reserve pattern 1 for if the allowlist grows past what's comfortable to review as an inline literal.

### Q10 — The three deliberately-dropped items (SHOW-05)

- **tmux / second terminal**: `grep -rn "tmux" cmd/*.go .claude/commands/ant/*.md` → zero matches. Clean.
- **Box-drawing borders**: `grep -rln $'┌\|┐\|└\|┘\|╔\|╗\|╚\|╝' cmd/*.go` (excluding `_test.go`) → zero matches. The only box-drawing-adjacent character in use is `└──` (nested detail marker, SEE-12's own compact-detail convention — e.g. `renderContinueWorkerFlowDetail`, cmd/codex_visuals.go:2492-2512), which is a KEPT convention, not a dropped border. A lock test must distinguish "no `┌┐╔╗` etc." from "no `└──`" (the latter is required, not forbidden).
- **"Clear context now?" prompt vs. the kept "safe to clear" line**: `grep -rn "clear context now"` (case-insensitive) across `cmd/*.go` and wrapper markdown → zero matches for the old question form. The KEPT line lives at exactly one function: **`renderContextClearGuidanceForPlatform(platform string) string`** (cmd/codex_visuals.go:633-653) — returns either `"Handoff not confirmed on disk — don't clear your context yet. Run \`aether status\` first.\n"` (line 641/643, when no handoff file) or `"Handoff saved (.aether/HANDOFF.md) — safe to clear your context now."` + a resume-command suffix (line 646-652, when the handoff exists). Two other call sites echo variants of the same **statement** (never a question): `cmd/medic_ceremony.go:94,96` ("It's safe to clear your context now") and `.claude/commands/ant/build.md:400` (menu option text, gated on the closeout's Handoff section actually saying "saved"). **A SHOW-05 lock test should**: (a) assert no string matching a "?" -terminated clear-context question exists anywhere in `cmd/*.go` or wrapper markdown, (b) assert `renderContextClearGuidanceForPlatform` still returns a *statement* form containing "safe to clear" for the handoff-exists branch, (c) assert no `table.NewWriter()` / fixed-width dash-rule usage outside the existing SEE-12 allowlist (already covered by `TestHumanDisplaysUseHeadedSectionsNotMachineTables`, which this phase should NOT duplicate but MAY extend if new files are added).

### Q11 — Test harness facts

- **Buffer capture**: `stdout` is a package-level `io.Writer = os.Stdout` (cmd/root.go:161). Tests redirect it: `saveGlobals(t)` (cmd/testing_main_test.go:124-...) snapshots and restores `store`, `stdout`, `stderr`, and various flag globals via `t.Cleanup`; individual tests then do `var buf bytes.Buffer; stdout = &buf` and read `buf.String()` after invoking the command function directly (pattern confirmed at cmd/testing_main_test.go and cmd/context_test.go:56-76, `TestResumeDashboard`). A second, narrower helper `saveGlobalsCmd(t)` (cmd/context_test.go:56-66) exists for tests that only need `stdout`/`stderr`/`store` restored.
- **Fixture builders in runtime shape**: `cmd/finality_parity_test.go` has dedicated builder functions — `parityContinueManifest(opts ...func(*codexContinuePlanManifest)) codexContinuePlanManifest` (line 20-43) and `paritySealManifest(opts ...func(*sealPlanManifest)) sealPlanManifest` (line 47-...) — both use a functional-options pattern with sane defaults, matching CLAUDE.md's "derive fixture values the way the runtime derives them" rule. Multiple `render*Visual` calls appear directly in test files passing a hand-built `result map[string]interface{}{...}` (e.g. `cmd/learning_beat_test.go`, `cmd/lifecycle_card_agreement_test.go`) — these are the precedent for constructing a completion-file-shaped JSON fixture that exercises `ceremony closeout`'s JSON round-trip path specifically (as opposed to the in-process typed-struct path), which is exactly the distinction `TestWrapperPathRendersSameCeremonyAsDirectPath` needs to exploit: build the SAME logical result via (a) calling the finalizer function directly (typed structs) and (b) `json.Marshal`+`json.Unmarshal` round-tripping the same value into a `map[string]interface{}` (simulating the completion file), then diff the two renders.
- **Test run cost**: the `cmd` package test suite is large (per project memory: ~12 min for the full suite). Recommend `go test ./cmd -run 'TestWrapperPathRendersSameCeremonyAsDirectPath|TestRenderedVisualsShowEveryCarriedField|TestSealConfirmation|TestSealAutopilotNeverReachesSealPath|TestBuildStartBlockerAdvisory|TestContinueLiveCheckLines'` (or equivalent name patterns once written) during iterative development, and reserve a full `go test ./cmd` run for the final verification pass per plan/wave gate, consistent with the "sampling rate" pattern used elsewhere in this codebase's Nyquist validation sections.

## Standard Stack

No external libraries apply — this phase is entirely internal Go rendering/wiring work using only the existing `cmd/` package, `pkg/colony`, and `pkg/codex` types already in the repository. No `npm install`/`pip install`/`cargo add` of any kind.

### Package Legitimacy Audit

Not applicable — no new external packages are installed by this phase.

## Architecture Patterns

### System Architecture Diagram

```
                    ┌─────────────────────────────────────────────┐
                    │  Finalizer (runCodexPlanFinalize /           │
                    │  runCodexContinueFinalize / completeSealRuntime) │
                    │  Builds a rich `result map[string]interface{}`  │
                    └───────────────┬───────────────────────────────┘
                                    │
                    ┌───────────────┴────────────────┐
                    │                                 │
             AETHER_OUTPUT_MODE=visual         AETHER_OUTPUT_MODE=json
             (direct CLI use)                  (wrapper-driven use)
                    │                                 │
                    ▼                                 ▼
        renderPlanVisual/                    outputOK(result)  ← visual string
        renderContinueVisual/                 DISCARDED HERE
        renderSealVisual                     (cmd/codex_visuals.go:362-371)
        (full detail)                                 │
                    │                                 ▼
                    │                     result written to completion file
                    │                                 │
                    │                                 ▼
                    │                     closeoutCompletionDetails()
                    │                     (cmd/closeout_cmd.go:86-200)
                    │                     re-derives a THINNER result map,
                    │                     extracting ~15 of 20-40 keys
                    │                                 │
                    │                                 ▼
                    │                     renderCeremonyCloseoutVisual()
                    │                     (cmd/ceremony_cmd.go:544-653)
                    │                     renders even less than that
                    ▼                                 ▼
             Direct CLI terminal              Chat / wrapper terminal
             (rich)                           (thin — THE GAP)

  PROPOSED FIX (D-12): route the right-hand branch through the SAME
  render*Visual function as the left, fed the completion file's raw JSON
  (already round-trips correctly per renderContinueWorkerFlowValue's
  existing dual-type handling), eliminating closeoutCompletionDetails'
  narrow re-derivation as the rendering source of truth.
```

### Recommended Project Structure

No new directories. All work lands in existing files:

```
cmd/
├── codex_continue.go          # runVerificationStep: add Duration field + start/finish emit (SHOW-03)
├── codex_continue_finalize.go # codexContinueWorkerFlowStep: add ToolCount field (SHOW-02/D-03)
├── codex_build_progress.go    # emitCodexDispatchWorkerFinished: add tool-count to finish line (D-03)
├── ceremony_cmd.go            # renderCeremonyCloseoutVisual: replace with call into render*Visual (D-12)
├── closeout_cmd.go            # closeoutCompletionDetails: pass through raw map instead of narrow extraction (D-12)
├── codex_workflow_cmds.go     # completeSealRuntime / sealCmd: insert confirmation gate (SHOW-04)
├── context.go                 # buildResumeDashboardResult: add per-phase progress list, drift note (SHOW-02)
├── codex_visuals.go           # renderResumeVisual: render result["recent"] (SHOW-02); renderContinueVisual: per-check/per-gate detail
├── ceremony_team_checkin.go   # buildHasPendingOwnerDecision-style predicate reused/extended for build-start advisory (D-08/D-09)
└── testdata/ (or inline maps) # new allowlist for TestRenderedVisualsShowEveryCarriedField (SHOW-05/D-12 discretion)
```

### Pattern 1: Dual-type rendering helpers (already established, reuse it)

**What:** Render functions that accept `interface{}` and type-switch between the in-process typed struct and the JSON-round-tripped `map[string]interface{}`/`[]interface{}` shape, so the same function serves both the direct and completion-file paths.
**When to use:** Any new rendering code this phase adds that must work identically whether called with a live Go struct or a value loaded from a completion JSON file.
**Example (existing precedent):**
```go
// Source: cmd/codex_visuals.go:2408-2435 (read this session)
func renderContinueWorkerFlowValue(b *strings.Builder, raw interface{}) {
	switch flow := raw.(type) {
	case []codexContinueWorkerFlowStep:
		// direct in-process path: full render
	case []interface{}:
		renderContinueWorkerFlowMap(b, flow) // JSON-round-tripped path
	}
}
```

### Pattern 2: Shared deterministic-floor body (already established, reuse for D-01..D-03)

**What:** One function both continue lanes call, structurally proven identical by a comparison test.
**When to use:** Placing the per-check live-progress hook.
**Example:**
```go
// Source: cmd/deterministic_floor.go:48-65 (read this session)
// This is the single body both runCodexContinueVerification (in-process lane)
// and runCodexContinueVerificationSnapshot (wrapper/external lane) call, so
// lane parity is structural rather than a discipline
// (TestBothContinueLanesApplyTheSameFloor).
steps := []codexVerificationStep{
    runVerificationStep(ctx, root, "build", requiredChecks["build"], commands.Build, verificationTimeout),
    runVerificationStep(ctx, root, "types", requiredChecks["types"], commands.Type, verificationTimeout),
    runVerificationStep(ctx, root, "lint", requiredChecks["lint"], commands.Lint, verificationTimeout),
    runVerificationStep(ctx, root, "tests", requiredChecks["tests"], commands.Test, verificationTimeout),
}
```

### Anti-Patterns to Avoid

- **Patching `renderCeremonyCloseoutVisual` section-by-section to add each missing field individually:** this keeps two renderers in sync by hand, exactly the failure mode D-12 exists to end. Prefer routing closeout through the same `render*Visual` function.
- **Parsing the `"...|continue_blocked|..."` event string to detect "last continue ended blocked" (D-09):** fragile, untested string format. Read the structured `codexContinueReport.Advanced` field from `continue.json` instead.
- **Conflating the spend-ledger's "never estimate, dash when absent" rule with worker duration/tool-count rendering:** they are different data models (`spendRow` vs. `codexContinueWorkerFlowStep`/`codexExternalBuildWorkerResult`); apply the same *philosophy*, not the same code.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Detecting "is there anything left for the owner to decide" | A new ad-hoc check | `buildHasPendingOwnerDecision` (cmd/ceremony_team_checkin.go:465) | Already covers 2 of D-09's 3 predicates, already tested by `TestBuildCheckinDecisionMatrix` |
| Recording an owner's forced override decision | A bespoke boolean flag on COLONY_STATE | The `colony.Decision`-with-`Source`-tag pattern (`cmd/forced_reviewer_waiver.go`) | Matches `TestOnlyTheOwnerCanWaiveAForcedReviewer`'s precedent: never inferred from rendered text, always a named, queryable record |
| Streaming a progress line | A bespoke `fmt.Fprintln(stdout, ...)` call | `emitVisualProgress` (cmd/codex_visuals.go:424) | Single point that respects output-mode gating and command-ceremony-level streaming allowance (`streamingAllowedForCurrentCommand`) |
| A shrink-only exception list | A new bespoke mechanism | Either the JSON+baseline pattern (`orphan_allowlist.json`) or the in-test Go map pattern (`TestHumanDisplaysUseHeadedSectionsNotMachineTables`) | Both already exist, tested, and understood by the team |

**Key insight:** Nearly every mechanism this phase needs (dual-type rendering, shared verification body, pending-owner-decision predicate, decision recording, shrink-only allowlists, output-mode-gated streaming) already exists somewhere in this codebase for an adjacent purpose. The work is almost entirely **wiring and extension**, not new mechanism design — consistent with this repo's own documented failure mode ("machinery exists but was never switched on").

## Common Pitfalls

### Pitfall 1: Assuming "carried in the result map" means "renderable with zero new code"
**What goes wrong:** Several SHOW-02 items (worker duration+tool-count table, per-check verification names, per-gate names) are carried in the finalizer's result map but are **also absent from the direct path's own rendering** (confirmed for `renderBuildFinalizeVisual`, `renderContinueVerificationSummaryMap`, `renderContinueGateSummaryMap`). Treating SHOW-02 as "just fix closeout to match direct" will silently fail to add these — because direct doesn't have them either.
**Why it happens:** The phase description's framing ("data already carried and never rendered") is true in aggregate but not per-item; some items need new rendering code shared by both paths, not just closeout parity.
**How to avoid:** For each SHOW-02 sub-item, explicitly check (a) is it in the result map, (b) is it rendered on direct, (c) is it rendered on closeout — three independent yes/no answers, not one.
**Warning signs:** A task described as "wire closeout to show X" when direct doesn't show X either.

### Pitfall 2: Reordering `runSealConsolidation` relative to state mutation when it's already correctly ordered
**What goes wrong:** Assuming D-04/D-05 requires moving the wisdom-review call earlier in `completeSealRuntime`. It doesn't — `runSealConsolidation()` already runs at line 649, well before `state.State = colony.StateCOMPLETED` at line 688. The actual gap is inserting a NEW stop-and-ask point, not reordering.
**Why it happens:** The phase description says "runs before the confirmation question," which reads as an ordering requirement, when the confirmation question doesn't exist yet at all.
**How to avoid:** Read `completeSealRuntime`'s full body before planning any reordering; the wisdom review's position is already correct.

### Pitfall 3: One-lane guarantees for the live check-progress hook
**What goes wrong:** Placing the emit call inside `runCodexContinueVerification` (cmd/codex_continue.go:1772, the in-process lane's caller) instead of inside `runVerificationStep`/`runDeterministicFloor` would mean the heavy/wrapper lane (`runCodexContinueVerificationSnapshot`, cmd/codex_continue_plan.go:264) never streams it, even though both lanes run the same checks.
**Why it happens:** `runCodexContinueVerification` looks like "the" continue verification function because it's the one called from the interactive default path — but `runDeterministicFloor` is the actual shared body.
**How to avoid:** Place the hook inside `runVerificationStep` itself (called identically from both lanes via `runDeterministicFloor`), and verify with `TestBothContinueLanesApplyTheSameFloor`-style field-by-field comparison that both lanes' output includes the new field.

### Pitfall 4: Fixture literals that don't match the real JSON round-trip shape
**What goes wrong:** Hand-typing a `result := map[string]interface{}{"verification": someStruct, ...}` fixture for a closeout test uses the TYPED struct, which is what the in-process path receives — but a real completion file has already been through `json.Marshal`/`Unmarshal`, so nested structs become `map[string]interface{}` and slices of structs become `[]interface{}`. A test built the wrong way will pass against code that would fail against a real completion file.
**Why it happens:** It's easier to construct the Go-native shape by hand than to round-trip it.
**How to avoid:** For `TestWrapperPathRendersSameCeremonyAsDirectPath` and any fixture feeding `ceremony closeout`, always construct the fixture via `json.Marshal` then `json.Unmarshal` into `map[string]interface{}`, never by hand-typing nested structs — this is the CLAUDE.md "derive fixture values the way the runtime derives them" rule applied directly.

### Pitfall 5: Confusing seal's default flow (host-mediated) with build/continue's default flow (never touches host)
**What goes wrong:** CLAUDE.md documents build/continue's interactive wrapper as never touching `aether host`. Seal's default (non-raw-bypass) flow DOES touch `aether host seal` (`.claude/commands/ant/seal.md:26`). Assuming seal mirrors build/continue's "never touches host" rule when writing SHOW-04 tests would be wrong.
**Why it happens:** CLAUDE.md's ownership-split table generalizes across commands; seal is the exception.
**How to avoid:** Re-read `.claude/commands/ant/seal.md`'s own "Ownership Split" / flow description rather than assuming CLAUDE.md's build/continue table applies verbatim.

## Code Examples

### Reading the deterministic floor's four checks (verified pattern to extend)
```go
// Source: cmd/deterministic_floor.go:59-66 (read this session)
steps := []codexVerificationStep{
    runVerificationStep(ctx, root, "build", requiredChecks["build"], commands.Build, verificationTimeout),
    runVerificationStep(ctx, root, "types", requiredChecks["types"], commands.Type, verificationTimeout),
    runVerificationStep(ctx, root, "lint", requiredChecks["lint"], commands.Lint, verificationTimeout),
    runVerificationStep(ctx, root, "tests", requiredChecks["tests"], commands.Test, verificationTimeout),
}
```

### The exact JSON-mode discard that causes the closeout gap
```go
// Source: cmd/codex_visuals.go:362-371 (read this session)
func outputWorkflow(result interface{}, visual string) {
	if shouldRenderVisualOutput(stdout) {
		if !strings.HasSuffix(visual, "\n") {
			visual += "\n"
		}
		writeVisualOutput(stdout, visual)
		return
	}
	outputOK(result) // <-- `visual` (the rich, already-computed render) is discarded entirely here
}
```

### The narrow key extraction that thins the chat path further
```go
// Source: cmd/closeout_cmd.go:222-240 (read this session)
func closeoutWorkerMaps(raw map[string]interface{}) []map[string]interface{} {
	workers := []map[string]interface{}{}
	seen := map[string]bool{}
	for _, key := range []string{"dispatches", "results", "workers"} { // <-- never "worker_flow"
		for _, worker := range mapSliceValue(raw[key]) {
			...
		}
	}
	return workers
}
```

## Runtime State Inventory

Not applicable — this is not a rename/refactor/migration phase (no strings are being renamed, no datastores relocated). Confirmed by re-reading the phase goal and CONTEXT.md: this phase adds rendering and one new confirmation gate, touching no persisted key names, no external service configuration, no OS-registered state, no secrets, and no build artifacts.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Autopilot (`cmd/codex_run*.go`) never calls `completeSealRuntime`/`sealCmd` today — asserted as an open item to verify, not confirmed by a full read this session | Q7 | If autopilot's code already has a code path reaching seal, D-07's lock test needs to name and close that path explicitly rather than assume it doesn't exist; low risk since D-07 explicitly requires a test regardless |

**All other findings in this document are `[VERIFIED]` — confirmed by reading the named file:line this session** (the specific quotes/field lists given beside each citation are what makes the tag checkable). No package-legitimacy or external-documentation claims apply, since no external packages or APIs are used by this phase.

## Open Questions

1. **Exact wording/data source for the "drift note" (Claude's Discretion)**
   - What we know: no existing computation of any resume-time "drift" signal exists anywhere in the codebase (confirmed zero-match grep).
   - What's unclear: whether "drift" should mean (a) time-since-last-decision, (b) a structural diff between the current plan and the last plan revision, or (c) something else entirely — the milestone brief lists it as a classic-v5.4.0 feature to restore but this session found no code trace of what the old implementation computed (the v5.4.0 code itself is not present in this repository to inspect).
   - Recommendation: since CONTEXT.md explicitly leaves this to Claude's Discretion, scope it conservatively at planning time — e.g., a note derived from `state.Plan.Revision`/`planRevisionSummary(state.Plan)` (already computed at cmd/context.go:290, carried but not surfaced in the resume card either) rather than inventing a new drift-detection algorithm.

2. **Whether seal's consolidation-beats text (printed to stdout, never in the JSON result map) needs to be captured for D-04's state-of-play card**
   - What we know: `renderSealConsolidationBeats`/`emitSealConsolidationCeremony` print directly during `completeSealRuntime`, synchronously, before the result map is built.
   - What's unclear: D-04's card needs "any checks still failing, any open warnings/flags" — this can be sourced from `checkSealBlockers`/`scanHighSeverityOpen` (already computed earlier in the same function) without needing the consolidation beats text specifically, since D-05 places the wisdom review AFTER the state-of-play card conceptually in the interaction flow (card → confirm → wisdom review runs as part of completing) — but the CURRENT code runs consolidation before ANY of this. The planner must decide: does the state-of-play card (D-04) need to move to BEFORE `runSealConsolidation()` is even called, with the wisdom review's own output then appended after confirmation but before the final mutation? This is a real sequencing decision the plan must make explicit, since D-04 (card) and D-05 (wisdom review runs before the confirmation question) together imply an order — **card → wisdom review → confirmation question → mutation** — that does not match today's **wisdom review → (nothing) → mutation** order in a way that's a simple insertion; it requires moving the confirmation-and-question logic to straddle the existing consolidation call, not simply prepending a gate.

## Environment Availability

Not applicable — this phase has no external tool/service/runtime dependencies beyond the existing Go toolchain and repository test suite, which are already confirmed working (per `CLAUDE.md`'s Verification Commands section and the fact that this is an active, building Go module).

## Validation Architecture

Skipped — `.planning/config.json` sets `workflow.nyquist_validation: false` explicitly for this repository.

**Note for the planner:** even without the formal Nyquist section, every named test below is required by the ROADMAP's own success criteria and must exist as a real, failing-until-fixed Go test per CLAUDE.md's Definition of Done:
- `TestWrapperPathRendersSameCeremonyAsDirectPath` (SHOW-01) — `go test ./cmd -run TestWrapperPathRendersSameCeremonyAsDirectPath -v`
- `TestRenderedVisualsShowEveryCarriedField` (SHOW-02) — `go test ./cmd -run TestRenderedVisualsShowEveryCarriedField -v`
- A new live-check-lines test extending `deterministic_floor_test.go`'s `TestBothContinueLanesApplyTheSameFloor` precedent (SHOW-03)
- A new seal-confirmation test and a new autopilot-never-seals test (SHOW-04/D-07)
- A new build-start blocker advisory test extending `ceremony_team_checkin_test.go`'s `TestBuildCheckinDecisionMatrix` precedent (D-08/D-09)
- A new deliberately-dropped-decisions lock test (SHOW-05)

None of these exist today (confirmed by grep, this session) — all are Wave 0 test-authoring work. Recommend a shared fixture helper that round-trips a finalizer result map through `json.Marshal`/`Unmarshal` into completion-file shape (see Pitfall 4), reused by the SHOW-01 and SHOW-02 tests, likely added alongside `cmd/finality_parity_test.go`'s existing builder pattern.

Suite cost note: the `cmd` package suite is large; use `go test ./cmd -run '<pattern>'` while iterating and reserve `go test ./... -race` for the phase gate.

## Security Domain

Skipped — `.planning/config.json` sets `workflow.security_enforcement: false` explicitly for this repository. Noted regardless: this phase touches no authentication, session, payment, deletion, or migration surface, and the one integrity-relevant concern (a wrapper-submitted completion file misreporting worker duration/tool-count) is already mitigated by the existing `mergeExternalBuildResults`/`mergeExternalContinueResults` validation against the manifest's own dispatch list — new rendering code should read from those already-validated structures rather than re-parsing raw wrapper submissions independently.

## Sources

### Primary (HIGH confidence — all `[VERIFIED]`, read this session)
- cmd/ceremony_cmd.go (full read of lines 1-900, 1200-1300)
- cmd/closeout_cmd.go (full read)
- cmd/codex_visuals.go (targeted reads: 340-460, 1399-1520, 2058-2560, 2995-3220, 4300-4450)
- cmd/codex_continue.go (targeted reads: 30-130, 580-660, 860-905, 1550-1600, 1772-1905, 3200-3290)
- cmd/codex_continue_finalize.go (targeted reads: 130-145, 1155-1360)
- cmd/codex_continue_plan.go (targeted read: 260-300)
- cmd/deterministic_floor.go (full read of lines 1-70)
- cmd/codex_plan_finalize.go (targeted reads: 425-495, 840-880)
- cmd/codex_build_finalize.go (targeted reads: 60-115)
- cmd/codex_build_progress.go (targeted reads: 100-210)
- cmd/dispatch_runtime.go (targeted read: 31-75)
- cmd/codex_workflow_cmds.go (targeted reads: 480-900)
- cmd/seal_final_review.go (targeted reads: 1-80, 380-510)
- cmd/context.go (targeted reads: 100-320)
- cmd/session_flow_cmds.go (targeted reads: 90-320)
- cmd/hook_cmds.go (targeted reads: 300-365)
- cmd/ceremony_team_checkin.go (targeted reads: 380-478)
- cmd/spend_cost_line.go, cmd/spend_ledger.go (targeted reads)
- cmd/criterion_evidence.go, cmd/gate.go (targeted reads)
- pkg/codex/worker.go, pkg/codex/usage.go (targeted reads)
- pkg/colony/colony.go (targeted reads: struct/const definitions)
- cmd/testdata/orphan_allowlist.json, cmd/subcommand_reachability_ratchet_test.go, cmd/display_house_style_test.go (allowlist pattern reads)
- cmd/finality_parity_test.go, cmd/testing_main_test.go, cmd/context_test.go (test-harness pattern reads)
- .claude/commands/ant/{seal,continue,plan}.md (full reads), diffed against .opencode counterparts (byte-identical, confirmed via `diff`)
- .planning/research/v1.27-milestone-brief.md (required reading, quoted directly for gap/drop lists)

### Secondary (MEDIUM confidence)
- None — no web sources used for this codebase-first research phase.

### Tertiary (LOW confidence)
- None.

## Metadata

**Confidence breakdown:**
- Standard stack: N/A — no external stack, all internal Go code
- Architecture: HIGH — every claim traced to a specific file:line read this session
- Pitfalls: HIGH — each pitfall derived directly from a confirmed code-reading finding, not speculation

**Research date:** 2026-08-29
**Valid until:** Until the next commit touching cmd/codex_visuals.go, cmd/ceremony_cmd.go, cmd/closeout_cmd.go, or cmd/codex_continue*.go — this is fast-moving internal code, not a stable external API. Re-verify file:line citations if significant time passes before planning executes.
