---
gsd_state_version: 1.0
milestone: v1.28
milestone_name: Classic Colony Restoration
status: executing
stopped_at: Completed 199-24-PLAN.md
last_updated: "2026-09-04T20:51:22.175Z"
last_activity: 2026-09-04
progress:
  total_phases: 7
  completed_phases: 0
  total_plans: 34
  completed_plans: 24
  percent: 71
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-09-02)

**Core value:** Aether should feel alive and truthful at runtime, not only look clever in wrappers or tests.
**Current focus:** Phase 199 — front-door-and-classic-contract
**Previous milestone:** v1.27 The Queen Decides, the Program Checks — SHIPPED 2026-09-02 (48/48 requirements; audit `tech_debt`, no blockers)
**Product version:** v1.0.66 (installed and source binaries agree in the 2026-09-02 local check)
**Governing backlog:** priority spec v3, ratified 2026-08-21 (D1), order amended 2026-08-22 (D12) — `.planning/research/priority-spec-v3-backlog.md`

## Current Position

Phase: 199 (front-door-and-classic-contract) — EXECUTING
Plan: 23 of 34
Status: Ready to execute
Last activity: 2026-09-04

## Performance Metrics

**Velocity:**

- Active v1.27 plans completed: 84 of 84 across nine phases
- Average duration: — (v1.25 phases averaged 6-11 plans each)
- Total execution time: 0 hours

**By Phase:**

| Plan | Duration | Tasks | Files |
|------|----------|-------|-------|
| Phase 172 P05 | 25min | 2 tasks | 4 files |
| Phase 173 P10 | 70min | 4 tasks | 5 files |
| Phase 193 P01 | 50min | 2 tasks | 7 files |
| Phase 196 P04 | 20min | 2 tasks | 8 files |
| Phase 193 P02 | 62min | 3 tasks | 13 files |
| Phase 193 P03 | 23min | 2 tasks | 2 files |
| Phase 193 P04 | 70min | 3 tasks | 12 files |
| Phase 193 P05 | 43min | 3 tasks | 7 files |
| Phase 194 P01 | 75min | 2 tasks | 9 files |
| Phase 194 P02 | 130min | 2 tasks | 22 files |
| Phase 194 P03 | 210min | 2 tasks | 18 files |
| Phase 194 P04 | 55min | 2 tasks | 5 files |
| Phase 194 P05 | 87min | 2 tasks | 28 files |
| Phase 194 P06 | 55min | 2 tasks | 11 files |
| Phase 194 P07 | 45min | 2 tasks | 5 files |
| Phase 194 P08 | 40min | 2 tasks | 3 files |
| Phase 194 P09 | 48min | 2 tasks | 9 files |
| Phase 195 P01 | 40min | 2 tasks | 2 files |
| Phase 195 P02 | 23min | 2 tasks | 6 files |
| Phase 195 P03 | 95min | 2 tasks | 9 files |
| Phase 195 P04 | 25min | 2 tasks | 8 files |
| Phase 195 P05 | 50min | 2 tasks | 5 files |
| Phase 195 P06 | 100min | 2 tasks | 6 files |
| Phase 195 P07 | 70min | 2 tasks | 7 files |
| Phase 195 P08 | 55 min | 2 tasks | 6 files |
| Phase 195 P10 | 55 min | 1 tasks | 4 files |
| Phase 196 P01 | 22 min | 2 tasks | 2 files |
| Phase 196 P02 | 13 min | 3 tasks | 14 files |
| Phase 196 P03 | 41min | 2 tasks | 5 files |
| Phase 196 P05 | 20min | 3 tasks | 17 files |
| Phase 196 P06 | 41 min | 2 tasks | 9 files |
| Phase 196 P07 | 78 min | 2 tasks | 10 files |
| Phase 196 P08 | 24min | 2 tasks | 12 files |
| Phase 199 P01 | 27min | 2 tasks | 4 files |
| Phase 199 P02 | 18min | 3 tasks | 5 files |
| Phase 199 P03 | 22 min | 2 tasks | 5 files |
| Phase 199 P04 | 31min | 3 tasks | 9 files |
| Phase 199 P05 | 27min | 2 tasks | 4 files |
| Phase 199 P06 | 29min | 3 tasks | 14 files |
| Phase 199 P08 | 58min | 3 tasks | 12 files |
| Phase 199 P09 | 19min | 2 tasks | 9 files |
| Phase 199 P07 | 26min | 2 tasks | 10 files |
| Phase 199 P10 | 27min | 3 tasks | 5 files |
| Phase 199 P19 | 7min | 2 tasks | 5 files |
| Phase 199 P11 | 29min | 3 tasks | 11 files |
| Phase 199 P20 | 12min | 3 tasks | 9 files |
| Phase 199 P12 | 16min | 2 tasks | 6 files |
| Phase 199 P13 | 40min | 2 tasks | 11 files |
| Phase 199 P14 | 28min | 2 tasks | 9 files |
| Phase 199 P15 | 65min | 2 tasks | 7 files |
| Phase 199 P16 | 71min | 3 tasks | 8 files |
| Phase 199 P22 | 13min | 2 tasks | 4 files |
| Phase 199 P17 | 36min | 2 tasks | 8 files |
| Phase 199 P18 | 80min | 2 tasks | 8 files |
| Phase 199 P26 | 62min | 3 tasks | 9 files |
| Phase 199 P21 | 16 min | 3 tasks | 8 files |
| Phase 199 P24 | 10 min | 2 tasks | 5 files |

## Accumulated Context

### Roadmap Evolution

- v1.26 roadmap created 2026-08-08 from 35 requirements across 7 categories (WIRE, SPAWN, SPEND, SEEN, ROSTER, SKILL, PROOF), continuing phase numbering from v1.25's Phase 171
- Research suggested 7 phases; the roadmap ships 8. The single deviation is splitting ROSTER into **176 Roster Reader** (ROSTER-01..02, the shipped 27 YAMLs) and **177 Operator-Authored Agents** (ROSTER-03..08, the user path), so the reader must demonstrably change dispatch output before the user-extension path is planned
- Research's suggested sixth phase ("Delegation Delivery") is **not** in this milestone. The SPAWN requirements cover the *guard* only; nothing gains the ability to delegate recursively in v1.26. Delivery (child result plumbing, follow-on-wave channel, parent-direct spawn) carries a research flag and belongs to a later milestone

### Decisions

- **The organising finding: most of v1.26 already exists and was never wired to a caller.** Four independent researchers converged on this by reading and *running* this repo's code. The recursion policy engine (`.aether/ts-host/src/spawn-orchestrator.ts`), the depth guard (`spawn-can-spawn` ignores its own `--depth`), the 27-caste roster (`colony/agents/*.yaml`, zero readers), the skill lifecycle (8 of 9 commands unreferenced), the selection rationale (composed, carried to the manifest, discarded), and token measurement (parsed, never persisted). The framing is **switch on and prove**, not **design and build**
- **Phase ordering is load-bearing, not stylistic.** WIRE first (a ratchet written after the capabilities gets shaped to whatever shipped) → SPAWN before SPEND (parent/depth linkage is recorded at spawn time and cannot be retrofitted to past runs) → SPEND before ROSTER/SKILL (extensibility changes what workers cost; without a ledger every later claim is unfalsifiable) → ROSTER reader before the user path → SKILL security inside the skill phase → PROOF last
- **ARCHITECTURE vs PITFALLS disagreement resolved in favour of guard-before-ledger.** ARCHITECTURE argued spend must ship first because delegation budgets set without spend data are guesses; PITFALLS argued the guard must ship first because the ledger's subtree roll-up needs linkage recorded at spawn time. Synthesis: the guard enforces with a deliberately conservative default (depth 2, tree total ~20 — the TS host's already-chosen numbers), and the ledger then supplies the evidence to retune those numbers with the measurement recorded
- **Success criteria must be able to fail.** CLAUDE.md's Definition of Done governs. Two corrections from this repo's history shaped them: a criterion reading "fewer than 27 castes loaded" already passed and proved nothing, and a token ledger's unit test asserted the same arithmetic its parser used, so a 186x undercount shipped green. The spend criteria therefore name the provider's documented figures (102,050 / 102,550), not the intent. Where possible criteria assert a proportion or invariant (grand total equals the sum of the `self` column; the ratchet allowlist may only shrink; the depth-cap numbers hold with five user agents installed)
- **Decision-shaped, not build-shaped: SPAWN-06.** What depth 0 means. The repo contradicts itself today (`build.md` hardcodes `--depth 1` for manifest workers, `workers.md` hardcodes `--depth 0` for their children). This is a ruling to record; the number matters less than one convention existing. Once recorded it fixes the expected value in Phase 173's depth-derivation criterion
- **Two further rulings the phases must make explicitly rather than discover:** (a) Phase 176 — whether the roster or `colony/policies/model-routing.yaml` owns model routing; both exist with zero readers, and wiring one without ruling on the other recreates the original condition. (b) Phase 177 — the user-agent namespace must be excluded from `TestCanonicalAgentSourcesRemainAligned` by decision, not by a hand-grown exemption list, or a user adding one agent either breaks CI or gets an agent that silently vanishes on two platforms
- **The TypeScript host is still KEPT** (carried from v1.25). `.aether/ts-host/src/spawn-orchestrator.ts` is a complete, correct ~150-line specification for Phase 173's Go port — it does not need rewriting, it needs a caller
- **Explicitly out of scope, already ruled:** automatic model routing (user decision 2026-07-28, reaffirmed — cheap-model capability means the framework carries the intelligence, not that it picks models), a compressed inter-agent language, intra-wave peer communication, a tokenizer dependency, user-configurable depth flags, an interactive agent-authoring wizard, an agent/skill marketplace, and further compression of the worker brief
- [Phase 172]: Named CI step 'Verify subcommand wiring and CLI flag contracts' added alongside (not replacing) the blanket go test ./... release gate, so a wiring failure reads as a wiring problem
- [Phase 172]: ROADMAP.md Phase 172/178 criteria corrected to the scan's real numbers: 278 seeded orphans, 6 owned by phase 178, replacing the pre-scan assumption of 8
- [Phase 173]: Plan 10: orphan_allowlist.json earned zero removals -- all five spawn-related candidates remain genuine orphans against the live reachability ratchet because .aether/workers.md is not a caller corpus this ratchet scans
- [Phase 173]: Plan 10: ROADMAP criterion 4 left byte-identical -- the cross-guard fail-closed proof establishes a narrower claim than the criterion's literal wording; whether it is satisfied is left to verification, per 173-RESIDUE.md residue 7
- [Phase 193]: The deterministic floor (`runDeterministicFloor`, one body for both continue lanes) is the only source of a pass; a reviewer verdict can only add a block. Locked by `TestDeterministicFloorIsTheOnlySourceOfAPass` and `TestBothContinueLanesApplyTheSameFloor`
- [Phase 193]: Build-side reviewer dispatch is gated on the Queen's explicit proposal, not the required-caste floor; the Watcher is verified once (`TestPhaseVerifiedOnce`). Probe/Auditor/Gatekeeper double-dispatch is recorded in `.planning/WINDOWS.md` #1 for Phase 194
- [Phase 193]: Builder-reported commands are re-run only when they match a fixed allowlist of build/test runners, contain no shell metacharacters, and run via argv — never `sh -c` (code-review fix CR-01); owner-facing commands are shell-quoted (CR-02)
- [Phase 193]: An unprovable criterion becomes `needs_owner_confirmation`: the phase advances, seal blocks until `aether decision-answer` records the answer; a failing free check with no reviewer sent gets exactly one automatic builder fix attempt, append-only in the attempt journal, then blocks with one command
- [Phase 193]: Deterministic floor promoted to the primary verification anchor; a dispatched reviewer verdict can only ADD a block, never supply the pass — runDeterministicFloor uses the build-time watcher (already resolved), never a live continue-time dispatch -- avoids re-running shell verification twice for a reviewer brief and guarantees a reviewer cannot supply a pass. Both continue lanes now share this one function.
- [Phase 193]: syntheticCriterionRequirements default checks drop watcher (D-06) — Unbound criteria now get claims plus the matching free check, never a reviewer caste by default; a dispatched reviewer that fails still blocks separately.
- [Phase 193]: [Phase 193 P02] Build's verification-stage watcher dispatch now gates on the Queen's explicit proposal (queenAskedFor/queenJudgement.Proposed), not the effective caste set that always includes watcher via the required-caste floor -- the required-caste floor itself is untouched (Phase 194's territory).
- [Phase 193]: [Phase 193 P02] Discovered (flagged, not fixed): probe, auditor and gatekeeper still legitimately double-dispatch on production/security phases with no explicit Queen proposal, because build and continue independently derive the same required caste from the same phase content. Recorded in .planning/WINDOWS.md for Phase 194 (moves the required-caste floor).
- [Phase 193]: 193-03: no implementation change needed in cmd/codex_continue.go -- runDeterministicFloor (193-01) already runs unconditionally before every reviewer-skip path is consulted, so all ten FLOOR-01 skip-path rows pass against pre-existing code — Plan explicitly allows this outcome; tests kept as regression guards
- [Phase 193]: 193-03: --skip-watchers help text corrected to state plainly that only AI reviewer workers are skipped and the program's own checks (build, types, lint, tests) always run; TestNoContinueFlagClaimsToSkipAChecked guards every continueCmd flag going forward — D-01 (193-CONTEXT.md); plain-English mandate (CLAUDE.md)
- [Phase 193]: FLOOR-03 closed: reconcile-task counted as evidence on both continue lanes, builder evidence re-run by the program (D-04), unprovable criteria marked needs_owner_confirmation and seal blocks until answered (D-05) — continueTasksSupportAdvancement's H-04 branch now requires only task.Verified for a reconciled task; reRunBuilderReportedEvidence re-executes a builder's reported commands itself; owner_confirmation_pending gate + checkSealBlockers extension route through the existing decision-answer and force-override paths
- [Phase 193]: 193-04: Tasks 2 (builder evidence re-run) and 3 (owner confirmation) landed in one commit, not two -- both edit the same per-check evaluation loop in evaluatePhaseCriterionEvidence and could not be cleanly split by git hunk — Task 1 (reconciliation) was cleanly separable via git add -p and is its own commit
- [Phase 193]: Verification tests are scoped to the Go packages a phase's changed files touched (D-07), falling back to the full run whenever that scope cannot be honestly derived or on the plan's final phase; only the tests command is ever scoped. — Narrowing a compiler or linter changes what it can see, so build/types/lint always run as configured; only the tests command has a safe scoped runner (go test ./dir/...).
- [Phase 193]: A failing free check with no reviewer dispatched draws exactly one automatic builder fix attempt (D-02/D-03), recorded as a brand-new append-only entry in the build attempt journal; a second automatic attempt never happens and continue blocks with one exact re-run command if the re-run still fails. — The fix attempt must never overwrite the original result and must be visibly countable on the team card and cost line; a new attempt record is the same append-only discipline the out-of-band verification record already established.
- [Phase 194]: Ordinary non-discovery work gets exactly the Builder by default; five named risks can force one explained reviewer at continue, and build never dispatches that forced reviewer.
- [Phase 194]: Every proposed worker needs its own `--caste-why` reason; missing reasons are refused by name, while runtime-added workers get plain-English reasons rather than score arithmetic or roster blurbs.
- [Phase 194]: Light and standard add no unconditional reviewers, heavy adds the full applicable panel, changed files can only add risk, and both continue lanes use the same forced-reviewer derivation.
- [Phase 194]: Forced-reviewer waivers are owner-only, one signal on one phase, capability-protected, and accepted only before dispatch; the pending one-worker check-in-pause reversal moves to Phase 195 discussion.
- [Phase 194]: `TestOneTaskBugFixIsOneWorkerPlusChecks` proves 1 worker on both paths, `TestNoCasteIsDispatchedAtBothBoundaries` closes WINDOWS #1, and the shipped documentation is test-locked to the runtime.
- [Phase 195]: Automatic grouping contracts the full phase graph only when the prospective same-caste component remains dependency-safe. — This prevents a valid task DAG from becoming a cyclic job DAG.
- [Phase 195]: Only exact declared implementation paths create automatic edges; ubiquitous documentation, manifests, and lockfiles are incidental. — Bookkeeping overlap must not collapse independent work.
- [Phase 195]: Automatic groups split against briefTaskContentAllowanceChars, while a single oversized task or accepted Queen proposal remains intact. — Brief content, not a magic task count, governs automatic job size.
- [Phase 195]: Keep covered_task_ids as assignment scope and task_receipts as distinct optional evidence. — Semantic receipt admission belongs to Plan 195-04.
- [Phase 195]: Reuse one codex.TaskReceipt and WorkerHandoff vocabulary across native and external completion JSON. — One reflected Go contract prevents lane drift and excludes worker-authored criteria or authoritative hashes.
- [Phase 195]: Coherent-job planning runs before task waves and before declared-worktree-ownership validation on every build lane, so same-wave declared overlap becomes one job instead of a pre-dispatch conflict error; `coalesceSequentialDispatches` keeps no live production caller.
- [Phase 195]: Grouped briefs gain an additive `## Covered Task Contracts` section carrying only what has no legacy single-task home (evidence requirements and otherwise-unlisted declared paths), so a one-task brief stays byte-for-byte unchanged.
- [Phase 195]: Every dispatch consumer iterates `dispatchCoveredTaskIDs` rather than the bare `TaskID` — abandoned-build recovery and continue assessment both previously lost every covered task after the first.
- [Phase 195]: Owner-facing wording is `covered tasks`, not `dependent tasks`; automatic grouping is no longer dependency-only, so the old phrase had become false.
- [Phase 195]: Two-stage receipt trust boundary (admitCoherentJobTaskReceipts / finalizeCoherentJobTaskReceiptEvidence) is the only path that can populate CompletedTaskIDs; completedBuildTaskIDs keeps whole-success crediting all covered tasks and reads CompletedTaskIDs for any other terminal status, never touched files or CoveredTaskIDs membership.
- [Phase 195]: Native/in-repo lane (executeCodexBuildDispatches) resolves receipts through both stages immediately after a worker's terminal result is known, since in-repo files already live in root; the external/wrapper lane's crediting wiring is deferred to plan 195-06, which reuses the same two functions unchanged.
- [Phase ?]: [Phase 195] decideBuildCheckin implements D-11..D-14 as a pure policy; buildHasPendingOwnerDecision reads three live sources (forced-reviewer waiver, orchestrator boundary question, worker handoff open decision) -- never dispatch count alone.
- [Phase ?]: [Phase 195] 195-05 Tasks 1 and 2 landed in one commit (9ca561c0) -- both edit the same buildCheckinDecisionInput/buildCheckinDecision types and the same plan-only branch, could not be cleanly split by git hunk (same precedent as 193-04).
- [Phase 195]: External build-finalize reuses the native lane's shared receipt admission/finalization boundary unchanged -- no second, external-only validator was written.
- [Phase 195]: BUILT-transition gating falls back to the union of every dispatch's own covered tasks when no explicit task selection is passed, so the ordinary no-filter build isn't misread as trivially fully-credited.
- [Phase 195]: Provenance's phantom-build guard now accepts a failed worker carrying genuine, evidenced task receipts as real partial provenance; zero receipts still reject the whole packet, unchanged.
- [Phase 195]: [Phase 195]: planCoherentJobRetry reuses the canonical planCoherentJobs grouping pass for a parent job's unfinished tasks (single all-inclusive proposal over the uncredited subset) rather than a second dependency graph; a new buildAttemptPartial attempt status (distinct from built/failed) plus ParentAttemptID/ParentJobName (begin-then-attach, same discipline as checkFixAttemptRecord) make D-10 recovery append-only on both build lanes -- retry only ever follows accepted credit, never a total failure with zero receipts.
- [Phase 195]: Build coherent-job contract defined once and asserted by three guards over five surfaces — Each guard re-typing its own anchor list would be a fourth surface able to go stale, reproducing the drift the guards exist to catch
- [Phase 195]: Three Go-authority sentences are compared byte-for-byte across YAML, guide, skill and wrappers — A keyword check would accept five surfaces each paraphrasing ownership differently, which is this repo's documented failure mode
- [Phase 198.1]: One memory-feed boundary (`recordDispatchWorkerOutcome`, cmd/memory_feed.go) is the only place a build worker's terminal result becomes a failure record or an observation; both build lanes call it and an AST guard (`TestEveryBuildLaneFeedsMemoryThroughOneBoundary`) refuses a third path. Feeding memory never fails a build or a check (`TestFeedingMemoryNeverFailsABuild`, `TestHiveFailureNeverBlocksThePhase`).
- [Phase 198.1]: Worker-authored text is untrusted on every store: all four midden writers and the observation path run `colony.SanitizeSignalContent` before storing, because midden entries are replayed verbatim into later worker briefs (review CR-02, fixed cc48873a). Dedup is exact (category, message) among unacknowledged entries, so a caller's message must carry phase/worker/status attribution (documented on `appendMiddenEntryOnce`).
- [Phase 198.1]: An instinct's "use" is a recorded fact — counted once per passing phase only when its text genuinely appeared in the capsule a worker received (`TestQueenPromotionNeverHappensWithoutRecordedUse`); the three-use gate to QUEEN.md is therefore satisfiable for the first time.
- [Phase 198.1]: Notes (pheromones) are written by the program at exactly three moments — a finished check, an answered worker question (never the seal's own "finish anyway?" confirmation), and three unacknowledged failures of one kind — and a valuable expiring note is copied to eternal memory first. Nothing is emitted during a build.
- [Phase 198.1]: Hive promotion runs at the end of every check under the same `AETHER_HIVE_POLICY` switch and ≥0.8 gate as seal; `--no-learn` skips only the legacy learning-entry capture — observations and failure records are still written so a blocked check leaves a trail (documented on the flag; review WR-01).
- [Phase 198.1]: Every learning-loop sentence in CLAUDE.md now names its test, and an AST guard fails if a cited test name stops existing (plan 05).
- [Phase 196]: spendTotals.ProviderUSD deleted rather than gated: nothing read it and an unread money field on the totals struct is a standing invitation to render one (D-01)
- [Phase 196]: The currency ban is scoped to the ledger's own types; codex.WorkerUsage keeps its USDCost and is skipped by name
- [Phase 196]: An empty ledger row status is refused rather than folded to failed, so a durable ledger never records a wrong outcome
- [Phase 196]: FIX 1-2's independence is enforced by an AST guard, not a comment, per CLAUDE.md's Definition of Done
- [Phase 196]: Pool-converted usage rows are tagged session-transcript, never provider-grade and never estimate, so a real measurement is not filed under the guessed subtotal
- [Phase 196]: The no-length-derivation ratchet scopes by package directory, not filename — a filename-scoped scan would have excluded the one file the violation lived in
- [Phase 196]: The guard-file inventory floor is raised to its live count on every registration, making the inventory itself a ratchet
- [Phase 196]: Deduplicate Claude transcript usage by message.id, corroborated by requestId; a usage-bearing line with no identifier is counted once on its own — Re-measured on this machine's whole corpus (1,726 transcripts): only two top-level line types carry usage — assistant at .message.usage and user at .toolUseResult.usage, at different positions. The user line is a subagent completion record with no message.id.
- [Phase 196]: The spend containment helper resolves symlinks on the deepest existing ancestor of a path that does not exist yet — The old fallback compared an evaluated root against an unevaluated candidate, so on macOS containment depended on whether the file had been written. Every existing refusal case still fails closed.
- [Phase 196]: The salvaged OpenCode reader was merged only inside the plan that wires it (D-03); its unsynchronised worker-name pattern cache is deleted outright rather than locked, because it was also unbounded and per-call compilation costs nothing
- [Phase 196]: One containment helper, validateSpendContainedPath, serves both platform validators, and the singleness is asserted by an AST count rather than a comment
- [Phase 196]: The closeout cost block is the LAST thing on every ending screen, on every lane, so its position is one rule rather than two that can drift apart
- [Phase 196]: The finalizers render no cost block; the three terminal screens do — that placement is what makes exactly-one-per-lane structural rather than dependent on output mode
- [Phase 196]: Compact magnitudes are truncated, never rounded up, so a headline can never claim a run cost more than its rows say it did
- [Phase 196]: The check files its rows before any result envelope is assembled, so a blocked check still records what it spent
- [Phase 196]: A run that filed no rows leaves no ledger file at all — an empty file is indistinguishable from a run that cost nothing
- [Phase 196]: A worker is accounted under codexBuildDispatch.Name (the deterministic per-worker name the platform's session titles carry), never AgentName, which several workers in one build share
- [Phase 196]: "Reported" keys on the usage source tag, never on a number — a worker that genuinely billed zero and one whose tool said nothing both present as zero, and D-01 renders those two differently
- [Phase 196]: The orchestrating session's own turns are returned separately from the worker list, neither shown as a dispatched worker nor discarded (the choice 196-03 left open)
- [Phase 196]: A dispatch whose status is outside the ledger vocabulary is dropped with a named note rather than sinking the whole fail-closed ledger; nothing invents a status the dispatch never stated
- [Phase 196]: Figures print exactly as recorded, unabbreviated, because shortening is arithmetic the read-only view is forbidden to do
- [Phase 196]: aether spend reads the current phase through loadColonyState, not loadActiveColonyState, which can persist a legacy repair and would make an inspection command a writer
- [Phase 196]: An estimated ledger row renders the dash sentinel exactly like an empty one — D-01 as amended forbids rendering an estimate at all
- [Phase 198.2]: Plan and colonize delegate manifests carry one runtime-assembled memory capsule; wrappers only pass it through, and triplet parity prevents the installed Claude/OpenCode lane from silently receiving less context.
- [Phase 198.2]: Oracle finish and stop paths share one file/register/promote body; useful partial research is labelled, strong findings carry origin labels, and empty runs write nothing.
- [Phase 198.2]: `decisions` and `learnings` were removed from the memory pack because nothing wrote them; an AST-discovered writer invariant with a zero-floor exception ratchet prevents another permanently empty part.
- [Phase 198.2]: The survey digest and previous-phase carry-forward use two separate named brief budgets, sanitise helper-authored text, and add no worker dispatch or owner check-in.
- [Phase 198.2]: The owner chose to advance while the WIRE-06 real-world check remains untested; this is verification debt, not a pass and not evidence of a defect.
- [Phase 198.3]: External swarm finalization now validates durable IDs before mutation, accepts only runtime-issued manifest digests, makes exact replay read-only, and rejects changed replay before rewriting strike truth.
- [Phase 198.3]: Seal and reviewer compatibility paths use the same fail-closed storage/severity rules as current schemas; malformed legacy evidence cannot become an empty safe set.
- [Phase 198.3]: Visual checkpoint paths and generations derive only after persisted claims match the exact built attempt's terminal claims; replan cadence filters active session/goal before revision projection.
- [Phase 198.3]: Complete verification is no longer inconclusive: 8,566 normal and 8,566 race-instrumented Go tests passed across 20 packages; Phase 198.3 scored 94/94 truths and 6/6 STAM requirements.
- [Milestone sequencing]: The owner rejected a standalone Phase 198.4 planning gate. v1.27 ends at verified Phase 198.3; the 35,921-word Dreams report is v1.28's authoritative brief, and the 72-row ledger is traceability rather than a 69-question owner gate.
- [Milestone sequencing]: Classic functional/experience restoration belongs inside v1.28. Owner-watched proof follows the restored product; Codex lifecycle skills remain a separate later milestone.
- [Milestone method]: Classic feature names are research headings, not selected implementations. Every Phase 199-204 must reconstruct the exact old mechanism and owner value, audit the current Go path, compare keep/restore/replace choices, and link a written synthesis into its plans, tests, CAP dispositions, and verification.
- [Milestone audit]: v1.27 scored 48/48 requirements, 9/9 delivered phases, 8/8 integration seams, and 4/4 end-to-end flows. Verdict `tech_debt`: no blocking gaps; Phase 198.2 field UAT and documented warnings carry forward.
- [Phase 199]: Every test that can allocate a worktree in the source checkout registers ownership before command execution. — Pre-registration guarantees bounded cleanup even when allocation exits partway through.
- [Phase 199]: Test teardown may delete only exact repository/path/branch triples registered by the creating test. — Naming conventions and Aether-shaped paths are not evidence of test ownership.
- [Phase 199]: Restore Classic information grammar and ceremony through modern Go truth, not prompt-owned writes or shell-era handling. — Preserves Classic intent without reviving superseded implementation mechanisms.
- [Phase 199]: Require every behavior corpus case to combine semantic assertions with causal state assertions or forbidden artifacts. — Prevents output-only proof from masking wrong state transitions.
- [Phase 199]: Compose source hygiene from production discovery, managed headers, sync rules, and peer parity while ignoring unmanaged paths. — Keeps the gate aligned with the real source-of-truth pipeline and preserves custom files.
- [Phase 199]: All Phase 199 lifecycle evidence uses one lifecycle/v1 schema and exported validated Go contracts. — One durable vocabulary prevents command-specific maps and incompatible wire meanings.
- [Phase 199]: New lifecycle evidence is pointer-backed and omitted when absent. — Legacy colonies remain readable and missing evidence stays unknown rather than becoming fabricated success.
- [Phase 199]: Forced-incomplete seal outcomes never satisfy verified completion. — Owner-forced closure must remain visibly distinct from evidence-backed verification.
- [Phase 199]: Lifecycle orientation reads direct bytes and records missing, malformed, or unavailable provenance instead of repairing state.
- [Phase 199]: Accepted plans expose guided build and Autopilot as a coequal choice set with no preferred marker.
- [Phase 199]: Ambiguous recovery uses resume; sealed colonies use status with entomb retained only as an optional alternative.
- [Phase 199]: Keep immutable coordinator intent separate from mutable progress so recovery never rewrites the evidence that authorized a mutation.
- [Phase 199]: Stage outputs and pre-images on each destination root, then replace targets only through temporary files in the target directory.
- [Phase 199]: Classify missing transaction evidence as unknown and altered evidence as conflicting; only matching digests may finish or roll back.
- [Phase 199]: Read-only and store-free command annotations suppress first-run and lock initialization writes for expert inspections.
- [Phase 199]: Live skills use source plus relative path identity and SHA-256 byte digests, with every inventory read directly from repository and hub sources.
- [Phase 199]: Skill inventory and drift remain nested below maintenance; standalone cache/list/diff root commands stay deleted.
- [Phase 199]: A fresh territory snapshot authenticates repository identity, canonical root, source revision, and every consumed artifact digest, including anchors.json.
- [Phase 199]: Automatic refresh writes worker output to a canonical candidate directory and publishes only a complete, validated snapshot through lifecycle_transaction.
- [Phase 199]: The plan manifest carries immutable territory evidence and finalization revalidates that exact evidence against the current repository before accepting a plan.
- [Phase 199]: Status loads lifecycle facts once, projects once, and treats full/compact as views of that same result. — One semantic result prevents the compact and full dashboards from disagreeing.
- [Phase 199]: Health and readiness use discrete evidence-backed states instead of percentage-derived maturity labels. — Verified, blocked, degraded, unavailable, and unknown remain distinguishable and cite their evidence.
- [Phase 199]: Status wrappers only invoke the Go runtime with argument passthrough; they never inspect host files, costs, or activity. — The Go lifecycle projection stays the sole authority across Claude and OpenCode.
- [Phase 199]: Root help is a curated journey map backed by the shared lifecycle projection; raw Cobra inventory remains command-specific or expert.
- [Phase 199]: Guided Claude and OpenCode init refuse an existing colony before storage opens; the raw runtime confirmation escape hatch remains compatible.
- [Phase 199]: Accepted intent is durable accepted-charter/v1 episode data, not presentation text reconstructed after initialization.
- [Phase 199]: Help wrappers invoke the visual Go command once and return stdout unchanged instead of maintaining a second help implementation.
- [Phase 199]: History event sources remain evidence sources; only recorded spawn-tree identities become actors. — This prevents command names from being presented as people or workers without evidence.
- [Phase 199]: Untimestamped lifecycle receipts remain visible after timestamped history evidence. — History must not invent a chronology from the read time or a nearby event.
- [Phase 199]: Phase exposes shared current_phase separately from requested detail/list/all selection. — All focused selectors must retain the same phase standing as status and history.
- [Phase 199]: Use the live Go maintenance catalog as the operation-inventory authority and require canonical/generated surfaces to match its ordered IDs. — This keeps wrapper discoverability mechanically tied to executable runtime truth.
- [Phase 199]: Keep every public maintenance wrapper to one visual runtime invocation; Go alone reads evidence, mutates state, verifies results, emits receipts, and rolls back. — A thin adapter cannot become a competing lifecycle authority.
- [Phase 199]: Expose live skill inspection only below /ant-maintenance and add no Codex-native $ant-* maintenance surface in this milestone. — This preserves the Phase 191 root-command deletion and the approved later-Codex boundary.
- [Phase 199]: Autopilot validates one accepted goal and an ordered remaining-phase set from immutable LifecycleFacts before opening its mutating loop. — Invalid entry must remain zero-write.
- [Phase 199]: Owner-authority changes use a separate typed fence outside the legacy ordered stage-trigger catalogue. — This preserves catalogue compatibility while pausing before owner-controlled changes.
- [Phase 199]: Without a Phase 202 typed event source, Watch reports idle and treats spawn/history rows as timestamped recorded evidence, never current liveness. — Stale files cannot prove an active ant.
- [Phase 199]: Run wrappers invoke the Go runtime exactly once; invocation is consent to displayed bounds and typed owner-authority results stop the wrapper. — The runtime remains the sole lifecycle authority.
- [Phase 199]: Claude and OpenCode init close with exact /ant-plan, while Codex closes with exact aether plan.
- [Phase 199]: Guided init surfaces synthesize intent, while Go alone owns setup, persistence, territory evidence, refusal, and result truth.
- [Phase 199]: Codex parity extends raw aether orchestration without restoring a native $ant-* lifecycle surface.
- [Phase 199]: A signal can claim live delivery or acknowledgement only from linked named lifecycle evidence, and measured effect only from linked changed-decision evidence.
- [Phase 199]: Swarm localizes only from an exact recorded active task ID and treats both dependencies and dependent tasks as the affected path; only other active tasks may continue independently.
- [Phase 199]: Go remains the only signal mutator and evidence authority; wrappers invoke the runtime but cannot manufacture acknowledgement or causal-effect claims.
- [Phase 199]: Pause and resume use the handoff ID as their idempotency key, so interrupted commits resume one journal and verified replay returns one existing receipt.
- [Phase 199]: Resume mutates only confirmed or explicitly reconstructed evidence; conflicting state, session, repository, worktree, activity, or receipt facts stop with no runnable-state write.
- [Phase 199]: Legacy pause-colony/resume-colony inputs are exact-token pre-Cobra redirects only through 1.28 and never appear in Cobra aliases, help, completion, or canonical pause wrappers.
- [Phase 199]: Hidden pre-Cobra lifecycle redirects are input migration only and are excluded from public wrapper inventory and alias repair. — Parser compatibility must not become product vocabulary.
- [Phase 199]: Retired installed wrappers are deleted only when the managed header proves Aether ownership; same-named custom files survive unchanged. — A filename alone cannot authorize deletion of user content.
- [Phase 199]: The current recovery contract teaches canonical resume only; bounded compatibility remains expiring parser plumbing. — There is one public return path while the migration window remains invisible.
- [Phase 199]: Use one typed SealPreflight and durable SealOutcome as completion truth; only a direct owner may request forced_incomplete with --force and a nonblank reason.
- [Phase 199]: Commit every authoritative seal artifact through the lifecycle transaction while retaining active COLONY_STATE.json for review; status remains primary and entomb optional.
- [Phase 199]: Treat issue-severity findings as retained residual risks, while blockers and unresolved owner checkpoints prevent normal verified closure.
- [Phase 199]: Permit Hive promotion only after verified closure under promote/unset policy, after sanitization and sensitivity filtering, with deterministic replay-safe receipts.
- [Phase 199]: Derive one stable entomb transaction and chamber identity from the verified seal outcome so interruption resumes instead of creating duplicate archives. — Stable identity makes retry and replay converge on one chamber and receipt.
- [Phase 199]: Treat an archive as valid only when every enumerated source byte, digest, kind, path, and closure cross-reference verifies before any publication or clear. — Directory existence cannot prove content integrity or closure truth.
- [Phase 199]: Keep invocation and owner confirmation separate from seal; platform wrappers call the Go runtime exactly once and never reproduce archive or clearing logic. — D-17 keeps sealed state reviewable until the owner explicitly chooses archive-and-clear.
- [Phase 199]: Keep legacy recover and abandon tokens parseable but hidden, store-free, and zero-write so old automation receives safe migration guidance without retaining authority. — Parser compatibility must not preserve product vocabulary or mutation authority.
- [Phase 199]: Read recovery evidence directly through LifecycleFacts and raw manifest/claims readers, while reserving all restoration for aether resume. — Expert diagnosis stays causally read-only and cannot become a second recovery owner.
- [Phase 199]: Maintenance previews close over exact typed targets and expected baseline digests; commit never performs a second broad discovery pass.
- [Phase 199]: Cleanup prefixes, globs, and content guesses are not deletion authority; a versioned aether-runtime manifest with exact path, owner, and current digest is required.
- [Phase 199]: Registry identity is derived from the canonical repository path, and duplicate identity/path or active-plus-final-stats conflicts fail before any write.
- [Phase 199]: A live maintenance interruption after durable intent attempts coordinator rollback immediately and returns its rollback receipt; unprovable rollback retains recovery-required evidence.
- [Phase 199]: Focused closeouts consume an existing LifecycleProjection and fail closed on revision disagreement. — This keeps lifecycle policy and Next Up owned by one projection.
- [Phase 199]: Zero-write refusals resolve from already-read, read-only state facts. — Rendering must not initialize a store or create lock files.
- [Phase 199]: Sealed colonies keep status primary and entomb optional; only verified archive-and-clear may offer init. — This preserves owner review authority and prevents failed archival from claiming a fresh-start boundary.
- [Phase 199]: Resume wrappers invoke the Go runtime exactly once and render its Confirmed, Reconstructed, Conflicting, or Unknown result without inspecting or selecting recovery evidence themselves.
- [Phase 199]: The existing Codex command guide exposes pause and resume only; bounded legacy parser compatibility remains invisible runtime plumbing.
- [Phase 199]: After seal, status is the primary review action and entomb remains a separate optional owner-confirmed archive-and-clear action.

### Pending Todos

- [x] Get user approval on the rescope — given 2026-08-14 ("go ahead, cut them all")
- [x] Plan Phase 172: Wiring Proof — shipped
- [x] Plan Phase 173: Delegation Guard — shipped
- [x] Plan Phase 174: Spend Ledger — closed partial at plan 2, rest cut
- [x] Phases 180–184: hardening fixes — shipped 2026-08-14/15 (v1.0.54)
- [x] Phase 175: Orchestration Visibility — shipped 2026-08-16 (outside the phase system; SHIP-PROGRESS Item 3)
- [x] Truth audit + hostile review + implementation programme approved — 2026-08-17
- [~] Phase 186: rescoped 2026-08-18 — plans 01-06 shipped; plan 07 cut from twelve runs to one non-gating smoke run, to be fired when convenient
- [x] Phase 193: Free Checks Are the Floor — executed and verified 2026-08-22 (v1.27)
- [x] Discuss + plan Phase 194: The Queen Decides the Team — planned 2026-08-23 (9 plans, 9 waves)
- [x] Execute and verify Phase 194: The Queen Decides the Team — complete 2026-08-26 (9/9 plans, 9/9 UAT)
- [x] Phase 195: Coherent Jobs — complete 2026-08-27 (10/10 plans; decision-free one-worker fast path shipped)
- [x] Plan Phase 196: See What It Cost — planned 2026-08-27 (8 plans, 4 waves; checked twice, 2 blockers found and fixed)
- [x] Plan and execute Phase 196: See What It Cost — complete 2026-08-28 (8/8 plans, 5/5 criteria, 3 review rounds + verification)
- [x] Phase 197: One Answer to "What Next?" — complete 2026-08-29 (7/7 plans)
- [x] Phase 198: Put the Thrown-Away Data Back on Screen — complete 2026-08-29 (9/9 plans)
- [x] Phase 198.1: Feed the Memory — complete 2026-08-30 (6/6 plans)
- [x] Phase 198.2: Memory Reaches Every Helper — complete 2026-08-31 (8/8 plans; one real-world UAT carried as debt)
- [x] Phase 198.3: Overnight Stamina — 22/22 formal plans plus six final trust-boundary repairs; verified complete 2026-09-02
- [x] Audit v1.27 at Phase 198.3 — complete 2026-09-02 (`tech_debt`, no blockers)
- [x] Close and archive v1.27 with the audited debt acknowledged — complete 2026-09-02
- [x] Initialize v1.28 Classic Colony Restoration from the comprehensive Dreams report — approved 2026-09-02
- [x] Route the 72-row capability ledger without re-interviewing the owner — 72/72 routed; final dispositions require phase synthesis
- [ ] Discuss Phase 199 and complete its front-door/lifecycle Classic-to-Go synthesis before implementation planning ← **current**
- [ ] Run owner-watched proof only after v1.28 restoration is usable
- [ ] Plan Phase 187: Crash-Safe Worktrees & Ecosystem Neutrality
- [ ] Plan Phase 188: One Truth for Failures and Advances (independent — may run in parallel with 187)
- [ ] Plan Phase 189: Complete Worker Contract (≥2 plans; independent — may run in parallel with 187/188)
- [ ] Plan Phase 190: Lean, Non-Duplicated Delivery (after 189)
- [ ] Plan Phase 191: Dead Wood (independent — may run in parallel with 187-190)
- [ ] Plan Phase 185: One Honest Cost Line (after 190)
- [ ] Plan Phase 192: Final Showdown
- ~~Plan Phases 176, 177, 178~~ — cut 2026-08-14; ~~Phase 179~~ — remapped 2026-08-17 into 186+192

### Blockers/Concerns

- ⛔ [Classic preservation] No historical capability may be deleted merely because current code has no caller. The report and 72-row ledger must inform v1.28 requirements and phase traceability.
- 🧭 [Owner intent] Do not repeat the Golden Era discovery interview. Ask only genuinely unresolved choices not answered by the comprehensive report.
- ℹ️ [Phase 198.2 verification] One real-world memory-pack UAT remains untested because the owner has not yet used the finished behavior on another project. It is acknowledged debt; review with `$gsd-audit-uat` and rerun with `$gsd-verify-work 198.2` after field use.
- ℹ️ [Phase 198.2 transition] `phase.complete` returned Phase 198.3 correctly but again left the roadmap checkbox and STATE `current_phase` frontmatter stale; both were corrected inline. The transition mutation still needs a later runtime fix.
- ℹ️ [Phase 198.2 side finding] This repo's local wisdom file contains backticks rejected by the current safety filter, so that local wisdom is withheld from helper prompts. This predates Phase 198.2 and was not changed here.
- ~~Roadmap is unapproved~~ — resolved 2026-08-14. This line was stale: it said not to start Phase 172 until sign-off, and 172, 173 and part of 174 were then executed anyway. Recorded because a gate that gets ignored without anyone noticing is the failure mode this project keeps rediscovering.
- **Phases 176, 177 and 178 are cut, not deferred.** If a future session finds Phase 173's delegation guards bounding a capability nobody has, that is expected and accepted — Phase 177 was the grant, and it was cut deliberately. Do not "restore" it.
- **Phase 173 and Phase 178 carry research flags from the research phase.** Platform nested-spawn behaviour is MEDIUM confidence and both vendors broke it within the last quarter (Claude Code strips the Agent tool from some subagent types; OpenCode has an open "subagents can infinitely recurse, no max depth" defect). The skill supply-chain threat surface is actively evolving. Re-verify vendor docs and open issues at planning time for both
- **OpenCode hook parity is unverified.** No confirmed equivalent to Claude Code's `PreToolUse` deny gate was found. If SPAWN-04's guard must hold on OpenCode, Go-side depth enforcement becomes mandatory rather than defence-in-depth — verify before planning that requirement
- **Codex has no native subagent nesting.** Delegation on the Codex lane must be *off* and reported honestly, never emulated. Emulation would create a second divergent orchestration path — the exact failure v1.24 and v1.25 spent two milestones unwinding
- Two gaps surfaced during roadmapping that have **no supporting requirement** and were deliberately not added to scope (see Coverage Notes in the roadmap return): the char-budget-vs-spend naming guardrail (Pitfall 8), and orphaned-child reaping with an operator command (Pitfall 5). Both are recorded as phase guardrails; if either is to be enforced, it needs a requirement and a user decision
- ~~`pkg/trace/cost.go` holds a stale model-price table~~ — **RESOLVED, found stale 2026-08-27.** The file no longer exists and no price table or token-to-dollar arithmetic remains anywhere in the Go source (`grep -rln 'CalculateCost|pricePerToken|costPerToken|InputPrice' --include='*.go'` returns nothing). The owner ruling this blocker demanded — delete as dead code, or fix its arithmetic — is moot; it was deleted at some point between v1.26 and now. Phase 174's D-02 ("no model-price table is ever built") therefore holds in fact as well as in principle, which is what Phase 196 needs before it puts a cost line on screen.

- `gopkg.in/yaml.v3` is archived and author-declared unmaintained. v1.26 makes YAML the format a non-technical user hand-writes, which changes the risk profile. The migration to `go.yaml.in/yaml/v3` is a mechanical import swap and is deliberately unbundled — it gets its own plan, not a rider on the roster work

## Deferred Items

Carried forward from v1.25 (superseded) and v1.23:

| Category | Item | Status | Deferred At |
|----------|------|--------|-------------|
| Visibility | SEE (14) — rich terminal visibility beyond SEEN-01..03 | Deferred | v1.26 roadmap |
| Typing | TYPED (8) — required `mode` field, removal of prose inference | Deferred | v1.26 roadmap |
| Reclaim | RECLAIM (9) — wider unreachable-command sweep beyond WIRE-01's ratchet | Deferred | v1.26 roadmap |
| Locking | LOCK (4) — state locking and lifecycle transactions | Deferred | v1.26 roadmap |
| Models | MODEL — automatic model selection | Rejected | user decision 2026-07-28 |
| Delegation | Recursive delegation *delivery* (child result plumbing, follow-on wave, parent-direct spawn) | Deferred | v1.26 roadmap — guard only this milestone |
| Hygiene | `gopkg.in/yaml.v3` → `go.yaml.in/yaml/v3` migration | Tracked, unbundled | v1.26 roadmap |
| Catalog | CATALOG-01, CATALOG-02 | Deferred | v1.25 roadmap |
| Test coverage | TEST-01, TEST-02 | Deferred | v1.25 roadmap |
| Workflow | WORKFLOW-01 … WORKFLOW-09 | Deferred | v1.25 roadmap |
| Runtime | RUNTIME-01, RUNTIME-02 | Deferred | v1.25 roadmap |

Acknowledged at the v1.26 close (2026-08-22). Each is carried in `.planning/research/v1.27-milestone-brief.md` or is an archived-phase leftover; acknowledging suppresses the audit line only until the artifact changes again.

| Category | Item | Status | Deferred At | Milestone |
|----------|------|--------|-------------|-----------|
| uat_gaps | 173/173-HUMAN-UAT.md | partial | 2026-08-22 | v1.26 |
| uat_gaps | 72/72-HUMAN-UAT.md | partial | 2026-08-22 | v1.26 |
| uat_gaps | 76/76-HUMAN-UAT.md | partial | 2026-08-22 | v1.26 |
| verification_gaps | 173/173-VERIFICATION.md | human_needed | 2026-08-22 | v1.26 |
| verification_gaps | 71/71-VERIFICATION.md | gaps_found | 2026-08-22 | v1.26 |
| verification_gaps | 72/72-VERIFICATION.md | human_needed | 2026-08-22 | v1.26 |
| verification_gaps | 76/76-VERIFICATION.md | human_needed | 2026-08-22 | v1.26 |
| todos | 2026-08-01-finalize-reconcile-task-evidence-gate.md | (presence-only) | 2026-08-22 | v1.26 |
| todos | 2026-08-01-ts-host-preflight-hardcoded-timeout.md | (presence-only) | 2026-08-22 | v1.26 |
| todos | 2026-08-20-spec-builder-feature.md | (presence-only) | 2026-08-22 | v1.26 |
| todos | 2026-08-21-completion-packet-cannot-express-bundled-work.md | (presence-only) | 2026-08-22 | v1.26 |
| todos | 2026-08-21-continue-checker-captures-wrong-field.md | (presence-only) | 2026-08-22 | v1.26 |
| todos | 2026-08-21-finalize-covered-tasks-DISPROVEN.md | (presence-only) | 2026-08-22 | v1.26 |
| todos | 2026-08-21-no-reentry-path-for-out-of-band-work.md | (presence-only) | 2026-08-22 | v1.26 |
| todos | 2026-08-21-stage-finalize-deadlock-FIXED.md | (presence-only) | 2026-08-22 | v1.26 |
| todos | 2026-08-21-weight-classes-pipeline-fits-the-task.md | (presence-only) | 2026-08-22 | v1.26 |
| deferred_items | 172/deferred-items.md: 172-00 (Task 1)  - **`.aether/docs/command-playbooks/continue-advance.md` — malf | acknowledged | 2026-08-22 | v1.26 |
| deferred_items | 172/deferred-items.md: 172-07 (both tasks)  - **`pkg/codex` `TestCodexReadOnlyProfileSelectsReadOnlySan | acknowledged | 2026-08-22 | v1.26 |
| deferred_items | 188/deferred-items.md: 2. `cmd/colony_prime_audit_test.go` and `cmd/medic_repair_test.go` seed the nest | acknowledged | 2026-08-22 | v1.26 |
| deferred_items | 190/deferred-items.md: D-190-01-A: Pheromone signals and prior-worker handoffs render twice in the plan | acknowledged | 2026-08-22 | v1.26 |
| deferred_items | 190/deferred-items.md: Stale ceremony-adapter snapshot fixtures (pre-existing, unrelated to hive_sectio | acknowledged | 2026-08-22 | v1.26 |
| deferred_items | 191.1/deferred-items.md: 191.1-01: `TestPackedNPMReleaseCandidateContract` fails on a pre-existing versio | acknowledged | 2026-08-22 | v1.26 |

Acknowledged at the v1.27 close (2026-09-02). These five artifacts are carried
forward explicitly; none is a failed v1.27 automated requirement or integration
flow. The Phase 198.2 rows describe the same owner-acknowledged field-use check.

| Category | Item | Status | Deferred At | Milestone |
|----------|------|--------|-------------|-------------|
| todos | 2026-08-01-ts-host-preflight-hardcoded-timeout.md | pending | 2026-09-02 | v1.27 |
| todos | 2026-08-20-spec-builder-feature.md | pending | 2026-09-02 | v1.27 |
| todos | 2026-08-27-worker-turnaround-is-too-slow.md | pending (high) | 2026-09-02 | v1.27 |
| uat_gaps | 198.2/198.2-UAT.md | partial | 2026-09-02 | v1.27 |
| verification_gaps | 198.2/198.2-VERIFICATION.md | human_needed | 2026-09-02 | v1.27 |

## Session Continuity

Last session: 2026-09-04T20:51:22.166Z
Stopped at: Completed 199-24-PLAN.md
Resume file: None

## Operator Next Steps

- Execute 199-23-PLAN.md, the first incomplete Phase 199 plan.
