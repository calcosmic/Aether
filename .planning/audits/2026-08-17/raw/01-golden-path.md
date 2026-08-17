# Raw report 01 — The real golden path of Aether's lifecycle

*Verbatim output of the "Trace Aether golden path" investigation, 2026-08-17, branch `oracle-reinstate`.*

## 0. Headline findings

1. There is not one golden path; there are TWO parallel execution stacks that share state files but not a dispatch mechanism:
   - Interactive wrapper stack: `/ant-build` etc. fetch a plan-only manifest from Go, the orchestrating LLM spawns platform Task agents, and a Go finalizer validates and commits.
   - Runtime-direct stack: `aether run` (and bare `aether build`) dispatches workers itself by spawning headless provider CLI subprocesses (`claude ...`) from inside the Go process (`cmd/codex_build.go:650,657`; `pkg/codex/platform_dispatch.go:125-140, 893-904`). `/ant-run` uses this stack exclusively (`.claude/commands/ant/run.md:11`) — it skips the wrapper's Queen caste-judgement stage, ceremony, and AskUserQuestion checkpoints.

2. The finalizers are genuinely hardened: manifest SHA-256 binding to a durable attempt journal, 24h freshness windows, on-disk existence checks for every claimed file, phantom-build rejection, and mandatory non-empty handoffs. Most "LLM skipped a step" failures are caught at finalize, not prevented at spawn.

3. Remaining fragile joints: (a) verbatim brief delivery to workers, (b) "worker results" are compiled by the orchestrator LLM from subagent chat output — nothing proves a Task agent ever ran, (c) the build boundary gate is prose-only (only seal's finalizer hard-blocks — `cmd/seal_final_review.go:441` is the lone caller of `unresolvedOrchestratorBoundaryGuidanceError`, per `cmd/orchestrator_boundary_guidance.go:85`), and (d) a "legacy" escape hatch in attempt binding (`cmd/build_attempt.go:461-462`) accepts a manifest with no attempt_id/attempt_path.

4. Wrapper line counts: build.md 358, init.md 336, continue.md 268, plan.md 213, seal.md 189, run.md 18.

## 1. /ant-init

Wrapper (336 lines): `aether init-research` (:16), scan summary (:73-100), prior colonies (:102-125), interview + SYNTHESIZE refined_goal/charter/≤3 pheromones (:127-171), colony-mode choice (:198-217), tick-to-approve pheromones (:219-251), shelf backlog (:253-283), then `aether init --charter-json ...` (:316), pheromone-write after success (:317). Go: `init-research` (cmd/init_research.go:1918, 2,151 lines, deterministic, no worker invoker). `aether init` (cmd/init_cmd.go:24-250): sealed/active colony requires `--confirm-reinit` with mandatory backup (:61-116, :184-198); clears prior residue (:174-177); charter validated for length only (:120-132). Boundary: semantic content LLM-authored; persistence Go-validated. FRAGILE in content, SOLID in persistence.

## 2. /ant-plan

Wrapper (213 lines): `aether host plan` (:25), depth card verbatim (:26), re-fetch with depth flags (:41), clarification/boundary gate (:53-61), `plan-research-approve` (:72-78), ceremony, Scout wave then Route-Setter (:98-127), completion JSON → `aether plan-finalize` (:140), loop on `requires_next_iteration` (:143). Guardrail: no direct `aether plan` (:199).

Go: `host plan` (cmd/host_cmd.go:47-52) execs Node on `.aether/ts-host/dist/host.js` — a manifest broker that shells back into `aether plan --plan-only` (command-registry.ts:74-76,159-166). Without `--dry-run` the TS host can spawn workers itself (host.ts:1504-1515) — a third dispatch channel. `plan-finalize` (cmd/codex_plan_finalize.go:244+): iteration contract (:258), manifest root/freshness (:261-274), confidence-evidence consistency (:321), and the evidence contract on the NEW plan — every phase carries success criteria with bound `evidence_requirements` (cmd/criterion_evidence.go:111-128). Loop exits computed in `evaluatePlanningLoopIteration` (:666), Go-owned. Boundary: mechanics SOLID; plan substance and confidence LLM-reported.

## 3. /ant-build — hop by hop

Wrapper (358 lines): `aether status` (:62) → signals (:75-83) → frame (:93) → `aether build $N --plan-only`, envelope to temp file (:105-117) → Queen team decision, optional `--castes` re-fetch (:119-207) → boundary gate (:211-223) → `ceremony spawn-plan` (:236) → per wave: wave-start, per worker spawn-log → Task spawn → spawn-complete → worker JSON → worker-complete (:266-274) → completion JSON → `build-completion-stage` → `build-finalize` (:286-302) → closeout → AskUserQuestion (:316-323). Guardrail: never `aether host build`, never bare `aether build` (:348).

Go manifest side (`runCodexBuildPlanOnlyWithOptions`, cmd/codex_build.go:207-381): pre-build gates (:228-244), stale-attempt supersede (:246-262), Queen policy + depth (:265-272), judgement + safety floors (:273, :1100), runtime-composed briefs (`attachBuildDispatchContext` :3080-3098, `composeBuildManifestBrief` :3110, `renderCodexBuildWorkerBrief` :2659-2806), attempt journal + manifest SHA binding (cmd/build_attempt.go:193-264) before return (:354-379).

Go finalize side (`runCodexBuildFinalize` :379-634): plan-only check (:388-390), phase (:391), root (:397), colony mode (:406), plan-revision/state-hash staleness (:655-674), attempt binding — manifest bytes SHA-match durable record, state unchanged, superseded rejected (build_attempt.go:458-521), completion-digest idempotency (:422-434), 24h freshness, criterion evidence (:444), task-set match (:447), then the atomic packet contract (:464-466): structural validation of raw bytes; every claimed path EXISTS in repo (`validateAndNormalizeClaimPathToRoot` :1629-1675); one terminal result per dispatch, identity match, duplicates rejected (`mergeExternalBuildResults` :977-1072); handoff shape validated (:1053), empty handoffs from completed workers rejected (:1145-1147); phantom-build rejection (cmd/provenance.go:14-61). Then atomic COLONY_STATE commit to BUILT (:586-596), worktree merge-back if applicable (:568-582), spawn tree + handoffs persisted, suggest-analyze once (:603). `build-completion-stage` (:157-217) stages the packet durably for crash recovery.

Cost of one /ant-build (4 dispatches, 2 waves): ~10-12 CLI invocations, 4 Task spawns, ~6 file writes, 1 AskUserQuestion — roughly 30-35 tool calls, plus reading 6-22KB brief per worker.

Fragile specifics:
- **brief_path mostly does not exist.** build.md:259 calls it "the preferred channel"; `BriefPath` is set only in `writeCodexBuildArtifacts` (:2071), called only from the DIRECT dispatch path (:606, :684) — never plan-only. Interactive path always gets inline briefs, exactly the truncation case the prose warns about.
- **No proof a Task agent ran.** spawn-log/spawn-complete are telemetry; build-finalize never cross-checks them (:1506 rebuilds the tree from dispatches). An orchestrator doing work in-context and fabricating results passes every check if files exist — claims are existence-only, and `claimsOrAggregate` falls back to `git status` discovery when claims are empty (:1309-1314).
- **Boundary gate is prose on build.** Unbound manifests fall into the `Legacy: true` branch (build_attempt.go:461-462) and finalize creates a NEW attempt (:493-498). Only seal enforces the boundary gate.
- Wave ordering/serial-vs-parallel and "do not continue after wave failure" are prose only.

## 4. /ant-continue

Wrapper (268 lines): DEFAULT is one runtime call: `aether status` then `AETHER_OUTPUT_MODE=visual aether continue --verification-depth standard` (:52-59); "no reviewer spawning happens on this path" (:64-66, :260). Heavy path on request: `aether host continue --dry-run --classic-ceremony` (:146), spawn reviewers, `continue-finalize` (:186).

Go default (`runCodexContinue`, cmd/codex_continue.go:523-1073) — the strongest link: requires build packet (:567-570); abandoned-build detection (:595-639); verification EXECUTED by Go — build/types/lint/tests via `sh -c` with timeouts (:1548-1552, `runVerificationStep` :2901, exec :3914-3916); builder claims re-checked against disk (`verifyCodexBuildClaims` :3023-3086); Watcher spawned by the Go process itself through the platform invoker (:1588-1611); escape hatches runtime-decided and recorded: `--skip-watchers` (:1581), provider-unavailable + shell green → auto-skip (:1589-1592), N consecutive watcher failures → auto-skip (:1593-1599), zero shell checks → responsibility to watcher with warning (:1570-1577). Gates evaluated and persisted (:676-713). Advance is one atomic read-modify-write with staleness guard (`validateRuntimeStateStillCurrent` :915-958). After durable advance only: learning capture (:987), phase-end consolidation (:994), per-phase git commit (:1000).

Heavy path (`continue-finalize`, cmd/codex_continue_finalize.go:123+): manifest provenance/freshness/state (:127-160), reviewer merge with identity checks (:173), findings persisted (:184), and decisively RE-RUNS its own Go verification snapshot and evaluates gates live (:191, :232). Wrapper-spawned reviewers cannot vote a failing build through.

## 5. /ant-seal

Wrapper (189 lines): raw bypass `aether seal` if user asks (:11-19, no review workers); otherwise `host seal` (:26) → Gatekeeper/Auditor/Probe dispatches (:47-50), force-seal only as explicit user question with user-authored reason (:31-41), boundary gate (:52-59), reviewers, `seal-finalize` (:148), Porter on user choice (:164-173).

Go: `runSealPlanOnly` (cmd/seal_final_review.go:236-300): `validateSealReady`, force requires reason (:245-247). `seal-finalize` (:400-490+): provenance/freshness/phase, hard block on unresolved boundary guidance (:441 — the only finalizer that does), gate passes only with completed evidence from every required caste (:472-476), blocking findings block unless `--force` (recorded as OverriddenReviewBlocks). Then `completeSealRuntime` (cmd/codex_workflow_cmds.go:443-674): eight-ant consolidation, instinct promotion ≥0.8, FOCUS expiry, state → COMPLETED, forced seals named in events and CROWNED-ANTHILL.md (:577-584, :1158-1174), registry update, Porter readiness. Fragility: reviewer FINDINGS content LLM self-reported; raw bypass reaches completeSealRuntime with NO final review wave.

## 6. /ant-run

Wrapper (18 lines): "Execute `aether run $ARGUMENTS` directly". Go (cmd/compatibility_cmds.go:231-283 → :386-551): READY → `runCodexBuildWithOptions` (REAL dispatch, subprocess invoker, rollback-on-failure), one retry (:452-468); EXECUTING/BUILT → `runCodexContinue`; pause on blocked (:512-523), classic pause conditions (cmd/autopilot.go:49-107), replan pause every N phases (:536-539). Divergences: autopilot never runs plan; skips Queen caste judgement entirely (`QueenCastes` never set :447-449); no platform Task agents. plan.md's /ant-run clauses (:29, :76) describe behavior the binary never exercises.

## 7. The chain

USER ACTION →(SOLID)→ COMMAND →(FRAGILE: 189-358 lines of prose; outcomes checked, steps not)→ ORCHESTRATOR →(SOLID: plan-only reads; finalizers reject hand-mutated state)→ STATE READ →(SOLID composition / FRAGILE delivery: "verbatim" is prose; brief_path absent)→ CONTEXT CONSTRUCTION →(SOLID: waves, floors in Go; Queen proposal re-validated)→ WORKER SELECTION →(FRAGILE: nothing verifies spawning; SOLID backstop: one terminal result per dispatch — omission caught, fabrication not)→ DISPATCH →(FRAGILE interactive: execution outside runtime observation; SOLID on run/watcher path: Go supervises subprocess)→ EXECUTION →(FRAGILE: orchestrator compiles results; brief never teaches the handoff schema; no agent def mentions "handoff")→ RESULT COLLECTION →(SOLID: packet contract)→ VERIFICATION →(SOLID: Go executes tests, re-checks claims)→ STATE MUTATION →(SOLID: atomic with supersession guard)→ RECOVERY/ADVANCE →(SOLID mechanics / FRAGILE routing)→ FINAL OUTPUT →(FRAGILE: ceremony/prose, display-only by design)

## 8. Wrapper/runtime/docs disagreements

1. CLAUDE.md says execution lives in "the host-manifest flow (`aether host build` → ...)"; build.md:348 forbids `aether host build`; real flow is direct `aether build --plan-only`.
2. build.md:259 sells brief_path; plan-only never produces it.
3. Asymmetric manifest fetching: build direct; continue heavy path via TS host; continue.md's own caste re-fetch example (:120) uses direct.
4. plan.md /ant-run clauses never exercised by the binary.
5. Continue heavy path has no completion staging (no continue-completion-stage analog).
6. build.md boundary gate implies enforcement only seal provides.
7. build.md:261-262 makes the handoff "mandatory" for workers, but neither brief nor agent definitions communicate the schema — the orchestrator invents/extracts it.

## 9. Bottom line

Every arrow that TOUCHES COLONY_STATE.json goes through a Go finalizer with binding, freshness, evidence, and atomicity checks, and `/ant-continue` is nearly wrapper-proof because Go runs the tests itself. What remains LLM-trust-based is the front half of every build: that the manifest's workers were really spawned, really received their briefs verbatim, and that results describe work done by those workers. The system stopped trying to make the LLM obey and built a customs checkpoint the LLM must pass through.
