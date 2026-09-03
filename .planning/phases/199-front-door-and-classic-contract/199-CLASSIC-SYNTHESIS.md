---
artifact: classic-mechanism-synthesis
phase: 199-front-door-and-classic-contract
status: signed
signed_at: 2026-09-03
source_revision: 4c696b67
requirements:
  - SYNTH-01
  - PROOF-01
historical_anchors:
  - 3a5b81c2
  - v5.0.0
  - v5.4
  - v5.4.0
source_study: .planning/phases/199-front-door-and-classic-contract/199-RESEARCH.md
capability_ledger: .planning/research/v1.28-classic-capability-ledger.md
---

# Phase 199 Classic Mechanism Synthesis

## Signed outcome

Phase 199 will restore the Classic Colony's understandable front door, visible agency, and deliberate closure while retaining the current Go runtime as the sole durable authority. The selected design restores the old **information grammar**—colony identity, accepted goal, actors and stage, evidence and state change, unresolved truth, then one safe next action—without restoring prompt-owned writes, inspection-time mutation, best-effort shell bookkeeping, or claims that are not backed by runtime evidence.

This brief is the signed implementation contract for Phase 199. Its ten synthesis rows and 34 routed capability rows are exhaustive for this phase. Historical Dreams and tagged source are evidence only and remain unchanged.

In plain English: keep the parts that made Aether easy to understand and satisfying to use, but rebuild them on the safer machinery that exists now.

### Goal and requirement coverage

- `GOAL`: restore the front door, lived-in colony experience, safe agency, and deliberate closeout as one honest Claude/OpenCode journey backed by runtime truth.
- `SYNTH-01`: every Classic mechanism and all 34 Phase 199 ledger capabilities receive an evidence-backed disposition before runtime implementation.
- `PROOF-01`: the restoration is governed by a versioned executable contract that pairs semantic output with causal state/filesystem assertions.

## Evidence base and method

The comparison uses four canonical Classic anchors and the current repository:

- February 2025 (`3a5b81c2`) supplies the clearest Queen-led front door and orientation grammar, including `git show 3a5b81c2:.claude/commands/ant/help.md` and `git show 3a5b81c2:.claude/commands/ant/status.md`.
- `v5.0.0` supplies the visible Queen → specialist workers → independent Watcher loop in `git show v5.0.0:.claude/commands/ant/build.md`.
- April 2025 (`v5.4` and `v5.4.0`) supplies the mature init, pause/resume, Autopilot, seal, entomb, and maintenance journeys under `git show v5.4:.aether/commands/` and `git show v5.4.0:.aether/commands/run.yaml`.
- Current truth comes from the Go command tree and state machinery in `cmd/`, canonical command sources in `.aether/commands/`, generated Claude/OpenCode commands, `RUNTIME UPDATE ARCHITECTURE.md`, and the repository-grounded audit in `.aether/dreams/2026-09-01-comprehensive-aether-colony-review.md`.

The allowed dispositions are `keep-current`, `restore-modern`, `replace-better`, and `retire-with-proof`. No Phase 199 mechanism or routed capability is retired.

## Classic mechanism reconstruction

| Classic mechanism | Historical evidence | Actor/control loop | Valuable causal mechanism | Unsafe or incomplete part not restored |
|---|---|---|---|---|
| Front door | `git show 3a5b81c2:.claude/commands/ant/help.md`; `git show v5.4:.aether/commands/help.yaml`; `git show v5.4:.aether/commands/init.yaml` | Queen narrated one journey and joined the goal to repository context. | Stable grouping reduced “what now?” cost while keeping expert tools discoverable. | Wrapper-owned state writes and independently computed Next Up advice. |
| Territory | `git show v5.4:.aether/commands/init.yaml`; `git show v5.4:.aether/commands/colonize.yaml` | Init connected the accepted goal to a repository scan, with colonize available explicitly. | Repository facts became part of the colony charter rather than an optional detached report. | Best-effort multi-file publication and an owner-operated freshness decision. |
| Orientation | `git show 3a5b81c2:.claude/commands/ant/status.md`; `git show v5.4:.aether/commands/status.yaml`; `git show v5.4:.aether/commands/resume.yaml` | Queen repeatedly rendered identity, progress, evidence, unresolved work, and one next action. | A consistent information order made returning to a colony predictable. | Each wrapper collected its own facts, and some reads performed migration or cleanup. |
| Agency | `git show 3a5b81c2:.claude/commands/ant/help.md`; `git show v5.4:.aether/commands/help.yaml` | The owner wrote FOCUS, FEEDBACK, or REDIRECT; later workers sensed the signal. | Memorable verbs exposed steering at the same conceptual level as work. | Prompt presence was treated as effect without typed acknowledgement or causal proof. |
| Autopilot | `git show v5.4.0:.aether/commands/run.yaml` | Queen looped build → policy check → continue/gates → advance/replan. | One visible command reused the guided-work stages and named why it paused. | Wrapper-local control state, incomplete repair accounting, and ambiguous authority to change accepted intent. |
| Pause and resume | `git show v5.4:.aether/commands/pause-colony.yaml`; `git show v5.4:.aether/commands/resume-colony.yaml`; `git show v5.4:.aether/commands/resume.yaml` | Queen captured the episode and later reoriented the owner. | Handoff carried the story needed to resume, not only a paused flag. | Several files changed independently, with no exactly-once transaction or reliable reconstructed provenance. |
| Seal | `git show v5.4:.aether/commands/seal.yaml` | Queen and Sage reviewed evidence and wisdom before an owner-visible Crowned boundary. | Ceremony was tied to a durable lifecycle state and readable bookend. | Learning/state effects could precede consent, and incompleteness was not uniformly fail-closed. |
| Entomb | `git show v5.4:.aether/commands/entomb.yaml` | The owner separately approved archive-and-clear after reviewing a sealed colony. | Two stages allowed inspection before destructive clearing. | Verification was path-oriented and sequential copying/clearing lacked a recoverable content transaction. |
| Maintenance | `git show v5.4:.aether/commands/help.yaml`; the historical command inventory cited by `.planning/research/v1.28-classic-capability-ledger.md` | The owner selected inspectable update, migration, cleanup, registry, skill, or chamber helpers. | Expert mechanisms remained discoverable. | A large flat surface made plumbing look ordinary, and each helper invented its own mutation boundary. |
| Proof | Historical wrapper sources above plus current `cmd/testdata/` | Text fixtures preserved visible language; later Go tests exercised individual mechanisms. | Named examples made intended experience discussable and regression-testable. | Snapshots and command presence cannot prove state causality, rollback, replay, or absence of forbidden writes. |

## Current mechanism audit

| Current area | Current evidence | Stronger modern primitive | Remaining gap selected for Phase 199 |
|---|---|---|---|
| Front door | `.aether/commands/help.yaml`, `.aether/commands/init.yaml`, `cmd/init_cmd.go`, `cmd/wrapper_command_names.go`, `cmd/codex_visuals.go` | Go validates and persists lifecycle truth; canonical YAML generates host surfaces. | Root and host help remain flat, setup is separate, and internal plumbing crowds the ordinary path. |
| Territory | `cmd/survey_staleness.go`, `cmd/codex_colonize.go`, `cmd/codex_plan_finalize.go` | Real manifests, ownership checks, containment, finalizer age/root checks, and plan digest injection. | Freshness is advisory and survey publication is not all-or-nothing. |
| Orientation | `cmd/status.go`, `cmd/phase.go`, `cmd/history.go`, `cmd/next_action.go`, `cmd/next_action_input.go` | One shared next-action resolver already validates candidates against the live command tree. | Fact collection is fragmented; some inspection uses loaders that can write; unavailable facts are not consistently explicit. |
| Agency | `cmd/pheromone_write.go`, `cmd/pheromone_loader.go`, `cmd/colony_prime_context.go` | Signals are sanitized, hashed, deduplicated, decayed, stored, and injected into real prompts. | Phase 203 owns causal live influence; Phase 199 must state delivery/effect/fallback truth without fabricating it. |
| Autopilot | `cmd/compatibility_cmds.go`, `cmd/autopilot_policy.go`, `cmd/autopilot_report.go` | A typed Go loop and pause catalogue handle real attempts and evidence. | Entry preflight can mutate, prerequisite routes are raw, and promised repair/debt/authority semantics are incomplete. |
| Pause and resume | `cmd/session_flow_cmds.go`, `cmd/recovery.go`, `cmd/codex_build_worktree.go` | Freshness, live-process, worktree, goal, and legacy-handoff checks are materially safer. | Public names are duplicated and multi-file pause/resume writes can partially commit or lose provenance. |
| Seal | `cmd/codex_workflow_cmds.go`, `cmd/seal_confirmation.go`, `cmd/seal_final_review.go`, `cmd/codex_visuals.go` | Readiness, blocker, owner-checkpoint, final-review, and explicit force-reason gates exist. | Effects are ordered unsafely and forced closure can share normal success language. |
| Entomb | `cmd/entomb_cmd.go`, `cmd/entomb_cmd_test.go` | Seal-first checks, a chamber manifest, export, path checks, and recovery documentation exist. | Content/cross-reference integrity and clear-after-publish are not one replay-safe transaction. |
| Maintenance | `cmd/command_truth.go`, `cmd/update_cmd.go`, `cmd/maintenance.go`, `cmd/skills.go`, `RUNTIME UPDATE ARCHITECTURE.md` | State migration already demonstrates validation, backup, containment, rollback, and live skill discovery. | Other operations remain flat and independently mutate multiple destinations without a shared journal/receipt. |
| Proof | `cmd/blackbox_harness_test.go`, `cmd/golden_workflow_test.go`, `cmd/classic_command_parity_test.go`, `cmd/testdata/` | The binary harness isolates repository, HOME, data root, and provider adapters and can inspect durable outcomes. | There is no versioned semantic Classic corpus, and text equality alone does not establish causality. |

## Comparative synthesis decisions

Each identifier in this table is authoritative and intentionally occurs exactly once in this artifact.

| Decision | Mechanism and value | Classic evidence | Current evidence | Considered alternatives | Disposition | Modern target | Principal risks | Verification IDs |
|---|---|---|---|---|---|---|---|---|
| `SYN-199-01` | Front door: grouped Queen-led guidance joined a goal to one understandable journey. | `git show 3a5b81c2:.claude/commands/ant/help.md`; `git show v5.4:.aether/commands/help.yaml`; `git show v5.4:.aether/commands/init.yaml` | `.aether/commands/help.yaml`; `.aether/commands/init.yaml`; `cmd/init_cmd.go`; `cmd/wrapper_command_names.go` | Keep flat help; copy Classic wrappers; compose modern runtime primitives. | **restore-modern** | Group the normal `/ant-*` journey, make guided init safely ensure setup/registry and route territory/planning, and keep Go envelopes authoritative. | Hiding expert access; host/runtime drift; active-colony replacement; generated-surface mismatch. | `V-199-FRONT-01`, `V-199-GEN-01`, `V-199-READONLY-01` |
| `SYN-199-02` | Territory: repository context belonged to the lifecycle, not to an owner-selected preparatory chore. | `git show v5.4:.aether/commands/init.yaml`; historical colonize sources cited in the research study | `cmd/survey_staleness.go`; `cmd/codex_colonize.go`; `cmd/codex_plan_finalize.go` | Manual colonize; resurvey every time; typed freshness gate. | **replace-better** | Classify survey evidence as `missing`, `fresh`, `stale`, or `unavailable`; refresh only when needed; publish the full survey atomically. | Stale or future timestamps; partial publication; symlink/path escape; finalizer replay; over-eager scans. | `V-199-TERRITORY-01`, `V-199-TXN-01`, `V-199-READONLY-02` |
| `SYN-199-03` | Orientation: repeated identity → progress → evidence → unresolved truth → Next Up let owners re-enter confidently. | `git show 3a5b81c2:.claude/commands/ant/status.md`; historical status/phase/history/resume sources cited above | `cmd/status.go`; `cmd/phase.go`; `cmd/history.go`; `cmd/next_action.go`; `cmd/next_action_input.go` | Duplicate renderers; enlarge status only; one typed projection. | **replace-better** | Build one read-only lifecycle projection and derive full/compact status plus focused phase, history, recovery, and closeout views from it. | Repair-on-read; hidden unknowns; inconsistent revision/next action; invented actors/cost/evidence. | `V-199-PROJECTION-01`, `V-199-ORIENTATION-01`, `V-199-READONLY-03` |
| `SYN-199-04` | Agency: FOCUS, FEEDBACK, and REDIRECT gave the owner memorable, immediate steering vocabulary. | Classic help and signal sources cited in the evidence base | `cmd/pheromone_write.go`; `cmd/pheromone_loader.go`; `cmd/colony_prime_context.go` | Claim live delivery now; remove signal verbs; expose a truthful delivery contract and fallback. | **restore-modern** | Preserve the verbs and add typed accepted/delivery/acknowledgement/effect/fallback fields; unsupported hosts promise only the next safe boundary. | Fabricated acknowledgement; prompt presence mislabelled as effect; premature Phase 203 behavior. | `V-199-AGENCY-01`, `V-199-TRUTH-01` |
| `SYN-199-05` | Autopilot: one visible build/continue loop reused the guided-work grammar and explained pauses. | `git show v5.4.0:.aether/commands/run.yaml` | `cmd/compatibility_cmds.go`; `cmd/autopilot_policy.go`; `cmd/autopilot_report.go` | Keep current loop; copy the wrapper; complete the Go contract. | **restore-modern** | Add zero-mutation prerequisite preflight, exact host-native routes, an operating contract card, bounded repair receipts, debt accounting, and an accepted-intent authority fence. | Early mutation; hidden retries; repeated consent; silently changed goal/scope/risk/acceptance; incomplete receipts. | `V-199-AUTOPILOT-01`, `V-199-AUTHORITY-01`, `V-199-READONLY-04` |
| `SYN-199-06` | Pause/resume: a narrative handoff preserved the episode across a clean break or interruption. | `git show v5.4:.aether/commands/pause-colony.yaml`; `git show v5.4:.aether/commands/resume-colony.yaml`; `git show v5.4:.aether/commands/resume.yaml` | `cmd/session_flow_cmds.go`; `cmd/recover.go`; `cmd/recovery_engine.go`; `cmd/codex_build_worktree.go` | Keep visible aliases; rename only; transactional recovery and migration. | **replace-better** | Make resume canonical, keep parser compatibility invisible, and reconcile one versioned handoff/receipt with confirmed-versus-reconstructed provenance before exactly-once commit. | Duplicate decisions/attempts/workers/cleanup; stale worktree identity; partial writes; public recovery vocabulary drift. | `V-199-PAUSE-01`, `V-199-RESUME-01`, `V-199-REPLAY-01`, `V-199-TXN-02` |
| `SYN-199-07` | Seal: a Queen/Sage ceremony made verified closure deliberate and retained a readable Crowned colony. | `git show v5.4:.aether/commands/seal.yaml` | `cmd/codex_workflow_cmds.go`; `cmd/seal_confirmation.go`; `cmd/seal_final_review.go`; `cmd/codex_visuals.go` | Flatten ceremony; keep current ordering; transactional honest seal. | **restore-modern** | Keep normal Crowned ceremony and current gates, stage effects after confirmation, and give forced closure a distinct non-success result with exact residual work and authority. | Pre-consent mutation; false completion; malformed blocker/review truth; force presented as normal success. | `V-199-SEAL-01`, `V-199-FORCE-01`, `V-199-TXN-03` |
| `SYN-199-08` | Entomb: separate archive-and-clear let the owner inspect a sealed colony before destructive clearing. | `git show v5.4:.aether/commands/entomb.yaml` | `cmd/entomb_cmd.go`; `cmd/entomb_cmd_test.go` | Merge into seal; remove entomb; strengthen the separate transition. | **restore-modern** | Preserve the locked two-stage lifecycle; stage a digest manifest, validate cross-references, publish chamber+tombstone, then clear through journaled recovery. | Any preflight mutation; content mismatch; partial publish; lost forced marker; active state cleared before verified archive. | `V-199-ENTOMB-01`, `V-199-DIGEST-01`, `V-199-TXN-04` |
| `SYN-199-09` | Maintenance: experts could inspect and operate useful internals without losing the colony metaphor. | `git show v5.4:.aether/commands/help.yaml`; historical maintenance inventory in the capability ledger | `cmd/command_truth.go`; `cmd/update_cmd.go`; `cmd/maintenance.go`; `cmd/skills.go`; `RUNTIME UPDATE ARCHITECTURE.md` | Leave operations flat; delete helpers; expert grouping plus one transaction protocol. | **replace-better** | Group update, migrate, clean, integrity, archive, registry, and generated-surface checks under an expert view using validate → stage → commit → receipt/rollback. | Useful behavior hidden or retired without proof; cross-destination partial writes; symlink escape; rollback ambiguity. | `V-199-MAINT-01`, `V-199-TXN-05`, `V-199-GEN-02` |
| `SYN-199-10` | Proof: examples preserve experience only when public actions are tied to semantic and durable outcomes. | All historical sources cited above and `.aether/dreams/2026-09-01-comprehensive-aether-colony-review.md` | `cmd/blackbox_harness_test.go`; `cmd/golden_workflow_test.go`; `cmd/classic_command_parity_test.go`; `cmd/testdata/` | Snapshots only; unit tests only; versioned semantic corpus. | **replace-better** | Add `cmd/testdata/classic-contract/v1/` cases linking historical anchors to real public invocation, semantic fields, required/forbidden text, pre/post facts, replay, and fault injection. | ANSI/text snapshots mistaken for truth; test-only helper paths; mutated checkout/global HOME; absent causal assertions. | `V-199-CORPUS-01`, `V-199-CAUSAL-01`, `V-199-PLATFORM-01` |

## Routed Phase 199 capability coverage

Every row below is owned by Phase 199, has exactly one selected disposition, and cites at least one current or historical source path. Later implementation plans may refine mechanics but may not silently change these outcomes.

| Capability | Area | Disposition | Modern target | Cited source path |
|---|---|---|---|---|
| `CAP-006` | entomb | `replace-better` | Preserve safe finalization inside the distinct, canonical archive transaction while keeping seal reviewable. | `.planning/research/v1.28-classic-capability-ledger.md`; `cmd/entomb_cmd.go` |
| `CAP-007` | entomb memory | `replace-better` | Retain learning, signal, and tombstone provenance in the verified archive before clearing active state. | `.planning/research/v1.28-classic-capability-ledger.md`; `cmd/entomb_cmd.go` |
| `CAP-008` | data cleanup | `restore-modern` | Keep cleanup as contained expert maintenance with a checkpoint and explicit result. | `cmd/maintenance.go`; `RUNTIME UPDATE ARCHITECTURE.md` |
| `CAP-013` | FOCUS | `restore-modern` | Keep FOCUS as immediate owner agency and report its honest delivery boundary. | `cmd/pheromone_write.go`; `cmd/pheromone_loader.go` |
| `CAP-015` | history | `restore-modern` | Render history from the shared read-only lifecycle/event projection. | `cmd/history.go`; `cmd/next_action.go` |
| `CAP-016` | help | `restore-modern` | Teach the one normal journey and separate expert/internal surfaces. | `.aether/commands/help.yaml`; `cmd/wrapper_command_names.go` |
| `CAP-017` | registry setup | `replace-better` | Make first-run init perform required registry work and report it rather than exposing plumbing. | `cmd/init_cmd.go`; `cmd/registry.go` |
| `CAP-018` | maturity | `replace-better` | Replace a decorative score with explicit health, capability, and readiness facts in the shared projection. | `cmd/status.go`; `.aether/dreams/2026-09-01-comprehensive-aether-colony-review.md` |
| `CAP-019` | state migration | `restore-modern` | Retain versioned, atomic, reversible migration with explicit legacy handling. | `cmd/command_truth.go`; `cmd/migrate_state_rollback_test.go` |
| `CAP-020` | activity and Next Up | `replace-better` | Derive activity interpretation and one safe next action from the canonical projection. | `cmd/next_action.go`; `cmd/next_action_input.go` |
| `CAP-026` | completion report | `restore-modern` | Produce one focused, evidence-backed closure card and honest seal recommendation. | `cmd/autopilot_report.go`; `cmd/seal_final_review.go` |
| `CAP-027` | phase list | `restore-modern` | Keep phase-list/all as a focused view of the accepted plan and shared projection. | `cmd/phase.go`; `.planning/ROADMAP.md` |
| `CAP-028` | phase detail | `restore-modern` | Show dependencies and success criteria from the accepted plan without recomputing truth. | `cmd/phase.go`; `.planning/ROADMAP.md` |
| `CAP-032` | resume orientation | `restore-modern` | Restore understandable context and one next action after a break. | `cmd/session_flow_cmds.go`; `cmd/next_action.go` |
| `CAP-033` | resume validation | `restore-modern` | Validate attempt, workspace, state, and handoff truth before continuing. | `cmd/session_flow_cmds.go`; `cmd/codex_build_worktree.go` |
| `CAP-034` | Autopilot resumability | `restore-modern` | Survive pause/interruption/resume without duplicate work or evidence. | `cmd/compatibility_cmds.go`; `cmd/recovery_engine.go` |
| `CAP-035` | resume alias | `replace-better` | Make `resume-colony` an invisible compatibility alias for one canonical resume transaction. | `cmd/session_flow_cmds.go`; `.aether/commands/resume.yaml` |
| `CAP-036` | seal boundary | `restore-modern` | Retain explicit owner-visible closure after fail-closed preflight. | `cmd/codex_workflow_cmds.go`; `cmd/seal_confirmation.go` |
| `CAP-037` | seal memory | `restore-modern` | Preserve project/colony memory with source and promotion provenance. | `cmd/codex_workflow_cmds.go`; `cmd/consolidation_lifecycle.go` |
| `CAP-038` | retained learning/future work | `restore-modern` | Render retained lessons and future work without counting either as completed scope. | `cmd/codex_visuals.go`; `cmd/seal_final_review.go` |
| `CAP-039` | seal refusal | `restore-modern` | Fail closed on blockers, malformed truth, missing evidence, and unresolved owner checkpoints. | `cmd/codex_workflow_cmds.go`; `cmd/blocker_snapshot.go` |
| `CAP-040` | final summary/handoff | `restore-modern` | Keep a focused final summary and next-project handoff grounded in exact evidence. | `cmd/codex_visuals.go`; `cmd/next_action.go` |
| `CAP-041` | seal authority | `restore-modern` | Record owner authority, force reason, residuals, and rollback/retention consequences before mutation. | `cmd/seal_confirmation.go`; `cmd/force_seal_test.go` |
| `CAP-042` | seal signals/wisdom | `restore-modern` | Preserve signal and wisdom lifecycles with explicit privacy and promotion boundaries. | `cmd/codex_workflow_cmds.go`; `cmd/hive_policy.go` |
| `CAP-049` | survey freshness display | `restore-modern` | Show survey freshness and let lifecycle policy select refresh. | `cmd/survey_staleness.go`; `cmd/status.go` |
| `CAP-050` | research/Dreams visibility | `restore-modern` | Show local research count/latest item while retaining honest note/evidence status. | `cmd/status.go`; `.aether/dreams/` |
| `CAP-052` | chamber/tunnel integrity | `replace-better` | Replace metaphor-only checks with typed artifact/context digests. | `cmd/entomb_cmd.go`; `RUNTIME UPDATE ARCHITECTURE.md` |
| `CAP-053` | update rollback | `restore-modern` | Checkpoint update and automatically roll back a failed installation or sync. | `cmd/update_cmd.go`; `RUNTIME UPDATE ARCHITECTURE.md` |
| `CAP-059` | survey-status scan | `replace-better` | Make one freshness projection authoritative for status and automatic territory refresh. | `cmd/survey_staleness.go`; `cmd/codex_colonize.go` |
| `CAP-060` | stale session/context | `replace-better` | Fold staleness, context clearing, and summary into safe pause/resume/status behavior. | `cmd/session_flow_cmds.go`; `cmd/status.go` |
| `CAP-062` | skill cache | `replace-better` | Use live skill scan/diff truth and retire obsolete cache-rebuild ceremony only after corpus proof. | `cmd/skills.go`; `AGENTS.md` |
| `CAP-064` | chamber creation/verification | `replace-better` | Use versioned, contained artifact/context digests with migration and replay recovery. | `cmd/entomb_cmd.go`; `cmd/entomb_cmd_test.go` |
| `CAP-065` | tunnel integrity | `replace-better` | Put the useful integrity result in the shared expert maintenance/status view. | `RUNTIME UPDATE ARCHITECTURE.md`; `cmd/status.go` |
| `CAP-068` | charter write | `restore-modern` | Write one visible charter/episode contract from the owner's accepted goal. | `.aether/commands/init.yaml`; `cmd/init_cmd.go` |

Coverage result: 34 routed rows, 34 selected dispositions, zero retirements, and zero unrouted Phase 199 ledger entries.

## Locked owner-decision traceability

The “Synthesis row” column deliberately uses the human area label so the authoritative synthesis identifier remains unique in the decision table.

| Decision | Locked owner contract | Synthesis row | Verification consequence |
|---|---|---|---|
| `D-01` | Claude/OpenCode ordinary vocabulary is `/ant-*`; raw `aether` commands are inspectable plumbing. | Front door; Maintenance | Default help separates ordinary and expert/internal surfaces; generated hosts stay source-linked. |
| `D-02` | After plan acceptance, guided build and Autopilot are equal choices with no recommendation. | Front door; Autopilot | Help/init/plan closeouts present both choices at equal visual weight. |
| `D-03` | Autopilot is legal only after init and accepted planning; early invocation is read-only and names the exact route. | Autopilot | Pre/post repository and state digests match on missing prerequisite paths. |
| `D-04` | A valid invocation is consent and shows goal, range, signals, and pause conditions before first mutation. | Autopilot | Contract-card fields precede any attempt/state write; there is no redundant confirmation. |
| `D-05` | Autopilot covers all remaining phases by default and reports bounded repair, debt, and stop evidence. | Autopilot | Completion and pause reports enumerate attempted range, repairs, residual debt, and proof. |
| `D-06` | Autonomous detail changes may not alter goal, promised behavior, scope, risk, or acceptance without the owner. | Autopilot | Any material delta stops before mutation and records the requested decision. |
| `D-07` | Signals affect live work only when provable; otherwise the next-safe-boundary fallback is explicit. | Agency | Ack/effect fields require typed evidence and unsupported hosts never invent them. |
| `D-08` | Swarm work stays localized to the affected job; substantive Swarm repair remains later work. | Agency; Autopilot | Phase 199 can name a boundary/fallback but cannot implement the later Swarm control loop. |
| `D-09` | Full status is authoritative and compact status is optional. | Orientation | Both render from one projection and agree on identifiers, truth status, and next action. |
| `D-10` | Closeouts are focused and rich rather than dashboard dumps. | Orientation; Seal | Command-specific views select relevant projection fields without recomputing facts. |
| `D-11` | Queen ceremony is strong but never invents actors, activity, spend, evidence, or outcomes. | Front door; Orientation; Seal | Every ceremonial claim maps to a typed runtime field with provenance. |
| `D-12` | Status snapshot and live/watch activity are distinct; substantive typed activity is later work. | Orientation | Snapshot reports persisted facts and honestly labels unavailable live telemetry. |
| `D-13` | The only ordinary public commands are `/ant-pause` and `/ant-resume`; suffixed names are invisible compatibility. | Pause and resume | Generated/public membership and help omit the aliases while parser compatibility remains testable. |
| `D-14` | Pause/resume uses a structured, exactly-once handoff transaction. | Pause and resume | Fault/replay cases cannot duplicate decisions, attempts, workers, or cleanup. |
| `D-15` | Resume is the sole recovery path and distinguishes confirmed from reconstructed facts. | Pause and resume | Recovery provenance is explicit and no separate ordinary recover command is offered. |
| `D-16` | Force seal is owner-only, visibly incomplete, and never labelled normal success. | Seal | Result discriminator, wording, completed counts, and residuals differ from normal closure. |
| `D-17` | Seal crowns and retains the colony; entomb is a separate optional archive-and-clear transition. | Seal; Entomb | Sealed state remains inspectable; entomb clears only after digest and cross-reference verification. |

## Verification contract

| Verification ID | Required proof |
|---|---|
| `V-199-FRONT-01` | Fresh and existing repositories see one guided journey with safe active-colony handling. |
| `V-199-GEN-01` | Canonical YAML and every managed Claude/OpenCode output agree through the production generator. |
| `V-199-GEN-02` | Maintenance and compatibility changes do not leave orphan or hand-edited generated artifacts. |
| `V-199-READONLY-01` | Help and init prerequisite inspection cause no mutation before accepted intent. |
| `V-199-READONLY-02` | Survey freshness inspection is hash-identical and reports missing/fresh/stale/unavailable truth. |
| `V-199-READONLY-03` | Status, phase, history, and recovery inspection leave repository/state digests unchanged. |
| `V-199-READONLY-04` | Invalid Autopilot entry leaves all scoped durable paths unchanged. |
| `V-199-TERRITORY-01` | Freshness and refresh policy cover never/fresh/stale/unavailable/future/baseline-change cases. |
| `V-199-PROJECTION-01` | All ordinary views agree on episode, revision, phase, blockers, evidence status, and next-action key. |
| `V-199-ORIENTATION-01` | Full, compact, and focused views preserve the locked information grammar without invented truth. |
| `V-199-AGENCY-01` | Signal acceptance, delivery boundary, and fallback are distinguishable machine-readable outcomes. |
| `V-199-TRUTH-01` | No renderer claims live acknowledgement or causal effect without typed evidence. |
| `V-199-AUTOPILOT-01` | Valid run displays its operating contract and executes the accepted remaining range. |
| `V-199-AUTHORITY-01` | Material goal/behavior/scope/risk/acceptance changes stop at an owner decision boundary. |
| `V-199-PAUSE-01` | Pause writes one structured handoff or no committed transition. |
| `V-199-RESUME-01` | Clean and reconstructed resume converge on one honest next action with provenance. |
| `V-199-REPLAY-01` | Retry/replay is idempotent across handoff, attempts, decisions, worktrees, and cleanup. |
| `V-199-SEAL-01` | Normal closure is fail-closed, evidence-backed, Crowned, and retains active sealed state. |
| `V-199-FORCE-01` | Forced closure records owner/reason/residuals and cannot render or count as normal success. |
| `V-199-ENTOMB-01` | Unsealed/inconsistent input refuses before mutation; valid archive remains a distinct optional transition. |
| `V-199-DIGEST-01` | Any content or cross-reference mismatch prevents active-state clearing. |
| `V-199-MAINT-01` | Expert operations expose dry-run/result/rollback truth without entering the ordinary journey. |
| `V-199-TXN-01` | Survey multi-artifact publication recovers or rolls back exactly once. |
| `V-199-TXN-02` | Pause/resume failure injection converges without partial semantic state. |
| `V-199-TXN-03` | Seal performs no closure effects before successful preflight and owner confirmation. |
| `V-199-TXN-04` | Entomb publishes and verifies archive/tombstone before any clear, with replay-safe recovery. |
| `V-199-TXN-05` | Maintenance follows validate → stage → commit → receipt/rollback across every destination. |
| `V-199-CORPUS-01` | A versioned loader rejects unknown fields, invalid groups, duplicates, and incomplete coverage. |
| `V-199-CAUSAL-01` | Every behavior case asserts semantic output plus state/filesystem causality, replay, or forbidden mutation. |
| `V-199-PLATFORM-01` | Claude and OpenCode invoke real public adapters; unsupported behavior is explicit rather than simulated as success. |

## Explicit phase boundaries

These are deliberate boundaries, not missing analysis or permission to implement them early:

- **Phase 200:** substantive planning behavior—intent ambiguity, research depth, living-plan revision, plan-vs-reality, and accepted-plan adaptation.
- **Phase 201:** substantive guided build/continue work-cycle behavior—Queen team selection, worker execution, verification, bounded recovery, and quick-work convergence.
- **Phase 202:** substantive Swarm/Oracle/live-colony behavior—localized repair loops, Oracle synthesis, typed activity streams, and live acknowledgements.
- **Phase 203:** substantive pheromone influence—canonical causal delivery, decision adapters, propagation, suggestion governance, and measured effects.
- **Phase 204:** substantive learning—outcome-backed observation, instinct, Queen/Hive promotion, failure-to-evaluation conversion, and consolidation truth.
- **Phase 205:** owner acceptance and proof—complete cross-platform restoration corpus, end-to-end parity, physical/manual checks, and release-readiness evidence.
- **Codex-native `$ant-*`:** native Codex skill/command implementation belongs to the later Codex milestone. Phase 199 may keep raw Codex runtime contracts honest but must not introduce that deferred public surface.

Phase 199 may define typed placeholders, projection fields, and honest fallback wording needed at its boundaries. It must not simulate the later mechanism or claim it is operational.

## Rejected alternatives and non-goals

- Byte-for-byte Classic restoration is rejected because it would restore prompt-owned mutation, duplicated truth, and best-effort shell semantics.
- Flattening the colony into generic task-runner wording is rejected because the Queen/caste/stage grammar explained real work and remains part of the product value.
- Making status the canonical data collector and copying it is rejected; status must first consume a pure shared projection.
- Merging entomb into seal is rejected because the owner-locked lifecycle requires reviewable sealed state before optional clearing.
- Output snapshots as sole proof are rejected because they cannot establish causality, zero mutation, recovery, or replay safety.
- New public lifecycle/helper commands are out of scope unless the signed rows require them and generated membership is proved.

## Signature

**Disposition:** approved for implementation.

**Signed basis:** the completed Phase 199 research study, locked owner decisions, the v1.28 capability ledger, current Go/YAML/generated surfaces, and the comprehensive historical review named in this artifact.

**Change control:** implementation detail may vary when it preserves the selected outcome, safety boundary, verification contract, and explicit phase boundary. Any change to owner-visible behavior, scope, risk, acceptance, seal/entomb distinction, or platform boundary requires a new owner decision and an updated signed synthesis.
