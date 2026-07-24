# Aether Polish Roadmap

## Execution Ledger: 2026-07-21

Completed in the unreleased working tree:

- Stage 0/1: nonzero error exits, terminal worker validation, implementation no-op rejection, minimum executed verification, criterion-bound evidence, durable direct/wrapper attempts, exact replay, and conflicting path ownership detection.
- Stage 1: generated read-only Discovery phases now complete only from durable task reports plus criteria/Watcher evidence; implementation phases still require project artifacts.
- Stage 2: hard-kill recovery and exact wrapper packet recovery are journaled; unsealed completion now resumes to `seal`.
- Stage 1: accepted plans now have immutable revision history; completed prefixes are preserved exactly while evidence-bound unfinished replacements are activated atomically and stale plan/build packets are rejected.
- Stage 3 containment: automatic Hive retrieval and promotion are off by default pending the trust redesign.
- Stage 5 contract work: platform differences are typed and returned by `command-guide`; compiled install/update/migration contracts pass in isolated source-candidate environments.

Still gating the next major release:

- Packed npm and release-archive acceptance, checksum/activation interruption, and N-1 rollback.
- Live-provider acceptance for the versioned plan revision; its deterministic compiled state contract now passes.
- One orchestration owner and the removal/quarantine of duplicate runtime authorities.
- Host-enforced permissions or explicit refusal of permission-dependent caste guarantees.
- Whole-wave transactional worktree ownership and reconciliation.
- Evidence-backed project learning and opt-in Hive redesign.
- Command/caste/skill consolidation and a smaller beginner surface.

Size is relative implementation scope: S, M, L, XL. No calendar estimates are implied. Type is remove, consolidate, repair, or add.

## Stage 0: Freeze And Establish Truth

### 0.1 Freeze Expansion And Publish A Claims Register

- **Problem:** New commands, castes, skills, ceremony, and control-plane scaffolds continue to expand a core that is not proven.
- **Evidence:** 396 command catalog entries, 27 castes, 86 skills, two TypeScript systems, and current false-completion journeys.
- **Proposed change:** Freeze feature additions. Create a machine-readable claims register mapping each public claim to owner, status, evidence test, supported platform, and release gate. Remove numerical reliability claims without data.
- **Dependencies:** Maintainer agreement; existing `audit-catalog`.
- **Risk:** Feels like reduced momentum and exposes more unsupported behavior publicly.
- **Test plan:** CI rejects new public commands/castes/claims without register entry and acceptance test.
- **Acceptance criteria:** Every documented capability is `proven`, `limited`, `experimental`, or `unsupported` per platform; no unregistered public command.
- **Size / type:** M / consolidate.

### 0.2 Build A Hermetic Black-Box Harness

- **Problem:** Unit/snapshot tests pass while released install and full journeys fail; one test deletes live state.
- **Evidence:** npm unit tests pass but public install fails; `control-ts` integration test targets repository state; cached Go suite hid memory failure.
- **Proposed change:** Add disposable HOME/repo/provider fixtures, fake adapter executable, network/release fixture server, process-kill hooks, and artifact/state assertions. Force `-count=1` on release Go tests.
- **Dependencies:** None.
- **Risk:** Existing tests may reveal widespread global-path assumptions.
- **Test plan:** Harness itself proves no file outside its temp root changes and detects deliberately injected false success.
- **Acceptance criteria:** Journeys can run concurrently; repository status and user home remain byte-identical; test cache cannot satisfy release gate.
- **Size / type:** L / add.

### 0.3 Inventory And Classify All State Writes

- **Problem:** Go Store, direct `os.WriteFile`, TypeScript tests, wrappers, and projections write overlapping state.
- **Evidence:** Hive direct writes; `control-ts` state deletion; state/session/handoff/planning/Queen/SQLite duplication.
- **Proposed change:** Generate a write inventory with path, caller, lock, transaction, schema, canonical/projection classification, and retry behavior. Fail CI on new unclassified writes under `.aether/data` or global hub state.
- **Dependencies:** Claims register.
- **Risk:** Some intentional absolute-path Store uses need explicit exemptions.
- **Test plan:** Static scan plus runtime file-access trace in core journey.
- **Acceptance criteria:** Every write has one owner and policy; TypeScript/wrappers cannot directly write canonical paths.
- **Size / type:** M / consolidate.

### 0.4 Capture A Release-Reality Baseline

- **Problem:** Source, npm, GitHub, hub, and binary versions diverge.
- **Evidence:** `1.0.42` source, `1.0.34` GitHub, `1.0.22` npm; mixed-version install reproduced.
- **Proposed change:** Record install/update results for current public releases and archive exact manifests/checksums/member names as fixtures. Mark broken channels visibly.
- **Dependencies:** Hermetic harness.
- **Risk:** Publicly documents a broken latest package.
- **Test plan:** Install each supported release on OS/arch matrix or equivalent archive fixture.
- **Acceptance criteria:** Supported N-1 baseline is explicit; unsupported versions fail before mutation with recovery instruction.
- **Size / type:** S / repair.

## Stage 1: Repair The Core Loop

### 1.1 Make Process Exit And Result Envelopes Agree

- **Problem:** JSON can say `ok:false` while the process exits 0.
- **Evidence:** 534 `outputError` calls across 88 files; live plan/status/refresh errors exited 0.
- **Proposed change:** Return typed command errors through Cobra; render exactly once at root; reserve exit 0 for `ok:true`. Migrate all call sites and add linter/static check.
- **Dependencies:** Baseline tests.
- **Risk:** Existing wrappers may rely on parsing JSON despite exit 0.
- **Test plan:** Table-test every public lifecycle error; wrapper integration checks both stream and status.
- **Acceptance criteria:** Envelope code, shell exit, transition status, and ceremony status always agree.
- **Size / type:** L / repair.

### 1.2 Replace `BUILT` Packet Semantics With Truthful Run States

- **Problem:** `BUILT` and `build_completed` can mean only that dispatch was attempted/prepared.
- **Evidence:** Interrupted live build with no successful workers set `BUILT` and returned success.
- **Proposed change:** Use `dispatching`, `collecting`, `verification_pending`, `paused`, `failed`, and `verified` transition states. Derive them from terminal worker records. Keep legacy `BUILT` only as a compatibility projection.
- **Dependencies:** Journal schema from ADR; exit-code repair.
- **Risk:** State migration and wrapper assumptions.
- **Test plan:** Kill before dispatch, during worker write, after write/before result, and during finalization.
- **Acceptance criteria:** No cancellation/timeout/all-failed run emits completed or a success envelope; retry resumes or safely redispatches only incomplete tasks.
- **Size / type:** L / repair.

### 1.3 Introduce Criteria-Bound Evidence Policy

- **Problem:** Existing path claims and inferred commands are not tied to acceptance criteria; all commands may be skipped and still pass.
- **Evidence:** Journey G completed without app/tests; `no command resolved; skipped` counted within a passing report.
- **Proposed change:** Plan records expected artifacts and verification methods per criterion. Go evaluates required deterministic checks; Watcher evaluates semantic criteria with cited evidence. A required skipped check is unresolved and blocks.
- **Dependencies:** Plan schema; run/evidence ledger.
- **Risk:** Some exploratory/docs phases need explicit non-code policies.
- **Test plan:** No-op worker, irrelevant edit, stale test output, docs-only phase, test-only phase, external task, human approval.
- **Acceptance criteria:** Each completed criterion points to fresh evidence; zero-evidence implementation cannot advance.
- **Size / type:** XL / add and repair.

### 1.4 Make Planning A Resumable Transaction

- **Problem:** Worker can write a useful plan then timeout; canonical state stays empty; retry deletes it.
- **Evidence:** Journey A real Route-Setter artifact and subsequent synthetic overwrite.
- **Proposed change:** Journal planning attempt, preserve worker artifacts by run ID/hash, collect late terminal results, and allow human acceptance of a valid partial proposal. Never clear artifacts until superseded and archived.
- **Dependencies:** Journal/evidence ledger; adapter result collection.
- **Risk:** Multiple proposals require clear selection/supersession semantics.
- **Test plan:** Timeout after artifact write, malformed completion, late result, retry, human accept, conflicting proposals.
- **Acceptance criteria:** Valid artifact survives interruption; retry does not destroy it; accepted plan and source evidence share run/hash.
- **Size / type:** L / repair.

### 1.5 Implement Real Plan Revisions

**Implementation status (2026-07-22): core repair complete; live-provider acceptance pending.** `Plan` now records an active revision and immutable snapshots. Refresh preserves the exact completed prefix, replaces only the unfinished suffix, remaps phase/task dependencies, and binds each accepted revision to reason, evidence paths, evidence-content hash, planning evidence, and parent revision. Plan finalization is atomic; packets created against changed state or changed evidence fail without mutation. Build manifests bind the active revision so superseded work cannot finalize. Status, resume, context, Go command guidance, Claude/OpenCode wrappers, Codex skills, and the TypeScript host carry the same contract.

- **Problem:** A completed phase prevents refresh, so research/verification cannot revise future work.
- **Evidence:** Explicit `cannot force-replan after completed phases`; Journey E.
- **Proposed change:** Add immutable plan revisions with stable node IDs, supersession, dependency remap, split/merge/reorder operations, completed-node preservation, and change rationale.
- **Dependencies:** Canonical plan schema and planning transaction.
- **Risk:** Complex migration from integer phase IDs and command references.
- **Test plan:** Contradicted assumption, split oversized phase, merge empty phases, reorder dependency, preserve completed evidence, prevent duplicate task execution.
- **Acceptance criteria:** Research can invalidate affected future nodes without losing completed unaffected work; `status` explains why revision changed.
- **Size / type:** Implemented as L / repair and consolidate. Remaining typed assumption/decision impact graph is part of 1.6.

### 1.6 Connect Research To Decisions And Plans

- **Problem:** Oracle artifacts persist, but planners only consume them if model discovery happens to find them.
- **Evidence:** Journey A planner cited Oracle output; no deterministic plan input contract requires it.
- **Proposed change:** Oracle emits typed findings: fact/hypothesis, source, confidence, scope, affected assumption/decision, recommended action. Plan requests list required research inputs and hashes.
- **Dependencies:** Plan revisions; research schema.
- **Risk:** Over-structuring exploratory research.
- **Test plan:** Conflicting sources, low confidence loop, sufficient-evidence stop, finding invalidates one phase, irrelevant finding ignored.
- **Acceptance criteria:** Full trace exists from question to source to conclusion to decision to plan revision to implementation constraint to verification.
- **Size / type:** L / consolidate.

### 1.7 Define And Implement The Adapter Contract

**Implementation status (2026-07-22): production ownership and the first typed
permission slice are complete; durable run handles remain.** Go now owns provider
selection, hard pins, preflight, prompt assembly, subprocess launch, result
parsing, and diagnostic redaction for direct/TS-host subprocess runs. The TS host
coordinates asynchronous waves through the Go boundary. `control-ts` is retired
fail-closed and its tests cannot touch live colony state. Scout/Includer use an
enforced read-only profile; all current write-capable castes use an enforced
workspace boundary with narrower prompt rules explicitly marked behavioral.

- **Problem:** Platform selection can ignore pins, permissions differ silently, and `control-ts` adapters return empty success.
- **Evidence:** Claude 402 under Codex pin; workspace-write/bypassPermissions; stub adapter source.
- **Proposed change:** Extend the implemented Go adapter slice with durable run handles, cancellation/reattachment, and execution-owner journal binding. Add narrow write profiles only when path-level enforcement is real.
- **Dependencies:** Journal/run IDs and result schema.
- **Risk:** Some platform APIs cannot enforce read-only or native subagents equivalently.
- **Test plan:** Pin each platform, unavailable auth, cancellation, timeout, malformed output, read-only denial, real smoke.
- **Acceptance criteria:** A selected adapter is the only invoked provider; unsupported permission blocks before dispatch; no stub is production-registered.
- **Size / type:** XL / consolidate and repair.

## Stage 2: Make Context Continuity Dependable

### 2.1 Make Context Files Rebuildable Projections

- **Problem:** Context, handoff, session, state, Queen, and reports can disagree or be stale.
- **Evidence:** Resume reported handoff removed and present; corrupted recovery lost plan; context capsule was minimal.
- **Proposed change:** Generate each projection from journal/evidence revision with source hashes, generated time, branch/workspace fingerprint, and explicit omissions.
- **Dependencies:** Journal and state write inventory.
- **Risk:** Existing platform prompts expect current Markdown shape.
- **Test plan:** Delete every projection and rebuild; compare semantic content; mutate branch/manual files and verify invalidation.
- **Acceptance criteria:** Deleting projections loses no truth; stale projection is never injected; next action is sufficient in blind resume test.
- **Size / type:** L / consolidate.

### 2.2 Universal Interruption Recovery

- **Problem:** Each command has its own checkpoint/force/reconcile behavior.
- **Evidence:** Planning loses proposal; build false-completes; handoff recovers reduced state.
- **Proposed change:** One transition protocol with start, heartbeat, pause, abort, commit and idempotency key. Resume replays journal and queries adapter run handles before deciding redispatch.
- **Dependencies:** Adapter and journal.
- **Risk:** Old platforms may not support process reattachment.
- **Test plan:** Kill at every lifecycle boundary and byte-offset fault injection during state/projection writes.
- **Acceptance criteria:** No false advancement, duplicate completed task, or manual cleanup in supported boundaries; exact next action provided.
- **Size / type:** XL / consolidate.

### 2.3 Workspace And Branch Reconciliation

- **Problem:** Manual edits, rebase, branch change, and generated setup files can be mistaken for implementation or stale evidence.
- **Evidence:** Setup appeared unreconciled; current status sees dirty tree; claims use path existence more than ownership.
- **Proposed change:** Record base commit/tree, dirty-path digest, ownership set, and transition-time diff. Require explicit adoption/rebase of evidence after workspace fingerprint changes.
- **Dependencies:** Evidence ledger.
- **Risk:** Uncommitted workflows need ergonomic reconciliation.
- **Test plan:** Manual edit, branch checkout, rebase, generated file, unrelated dirty file, submodule/symlink.
- **Acceptance criteria:** Aether distinguishes pre-existing, worker-owned, generated, manual, and conflicting changes.
- **Size / type:** L / add.

## Stage 3: Make Queen And Hive Knowledge Trustworthy

### 3.1 Separate Knowledge Types And Scopes

- **Problem:** Preferences, facts, decisions, observations, conventions, and wisdom share prose stores.
- **Evidence:** Queen/Hive/context blending and cross-project contamination.
- **Proposed change:** Typed knowledge schema with scope, evidence, validity interval, branch/revision, owner, confidence method, contradictions, supersession and expiry.
- **Dependencies:** Journal and projection work.
- **Risk:** Migration may quarantine most legacy wisdom.
- **Test plan:** Project override, stale convention, branch-only rule, preference conflict, deleted evidence.
- **Acceptance criteria:** Precedence is deterministic; every injected item shows type/scope/source; legacy unknowns are advisory only.
- **Size / type:** L / consolidate.

### 3.2 Rebuild Project Learning Pipeline

- **Problem:** Rich `pkg/memory` logic and simpler command pipelines diverge; consolidation test fails.
- **Evidence:** Fresh and race Go suites fail `TestPipeline_Consolidation`.
- **Proposed change:** One observation -> evidence -> validation -> convention pipeline. Require independent evidence and later outcome; add contradiction and revocation. Fix or remove unused stages.
- **Dependencies:** Typed knowledge.
- **Risk:** Fewer automatic promotions may make learning appear less active.
- **Test plan:** duplicate, contradiction, incorrect high confidence, stale/deleted source, decay, revocation, replay.
- **Acceptance criteria:** Full uncached/race suite passes; no promotion without source evidence and rule explanation.
- **Size / type:** XL / repair and consolidate.

### 3.3 Quarantine And Redesign Hive

- **Problem:** Arbitrary source strings can create 0.95 confidence; no lock/contradiction/decay; injection leaks across projects.
- **Evidence:** Journey F; `cmd/hive.go` schema and direct write.
- **Proposed change:** Disable automatic Hive injection/promotion by default. Require stable repository identities, evidence fingerprints, domain compatibility, contradiction sets, revocation/decay, locked writes, and user opt-in.
- **Dependencies:** Knowledge schema and project pipeline.
- **Risk:** Existing users lose implicit global tips until reviewed.
- **Test plan:** two incompatible repos, fake repo IDs, repeated same evidence, sanitizer payload, concurrency, unrelated retrieval.
- **Acceptance criteria:** No project-specific rule enters unrelated context; false lesson can be traced and revoked; concurrent updates are lossless.
- **Size / type:** XL / remove and redesign.

### 3.4 Narrow Queen Responsibility

- **Problem:** Queen currently names coordination, routing, memory, gate decisions, promotion, ceremony, and state.
- **Evidence:** Queen commands/files and multiple `queen-state`, audit and recovery records.
- **Proposed change:** Queen becomes generated project coordination view and policy decision identity. Global preferences and Hive have separate owners; low-level state remains Go core.
- **Dependencies:** Projections and knowledge types.
- **Risk:** Migration of familiar commands/narrative.
- **Test plan:** Rebuild Queen from journal; compare current goal/decision/blocker/next; ensure no direct state mutation through Queen prose.
- **Acceptance criteria:** Deleting `QUEEN.md` and regenerating changes no lifecycle or knowledge truth.
- **Size / type:** M / consolidate.

## Stage 4: Prove Ant Specialization And Pheromone Coordination

### 4.1 Reduce To Five Core Castes

- **Problem:** 27 prompts do not correspond to 27 enforced capabilities and create worker/token overhead.
- **Evidence:** Nine-worker light bug plan; six-worker notes phase; common permissions/tools.
- **Proposed change:** Core Queen, Scout, Route-Setter, Builder, Watcher. Convert other castes to skills/policies/plugins; retain ant names and show plain role label.
- **Dependencies:** Adapter permissions and acceptance policy.
- **Risk:** Perceived loss of identity/features.
- **Test plan:** A/B/C journeys with minimal colony; compare quality/cost to current roster.
- **Acceptance criteria:** Each core caste has unique input/output/permission contract; optional role improves a predefined metric before default use.
- **Size / type:** L / remove and consolidate.

### 4.2 Make Signals A Coordination Primitive

- **Problem:** Pheromones are expiring prompt injections with no runtime conflict or consumption record.
- **Evidence:** Conflicting identical FOCUS/REDIRECT injected together; display expiry mismatch.
- **Proposed change:** Add scope, issuer, provenance, effective priority, conflict set, policy action, consumer acknowledgment, decision link and outcome. Compute expiry at read time.
- **Dependencies:** Journal and selection policy.
- **Risk:** More structure may reduce casual steering ergonomics.
- **Test plan:** all conflict/stale/mid-phase/automatic/user-correction cases from brief.
- **Acceptance criteria:** User can inspect exactly which decision changed; unresolved equal-priority conflict pauses instead of silently choosing.
- **Size / type:** L / repair.

### 4.3 Make Parallelism Ownership-Safe

- **Problem:** Worktree merge can silently discard overlapping edits; in-repo builders are serial despite parallel language.
- **Evidence:** Journey G.
- **Proposed change:** Require predicted ownership sets; detect overlap before dispatch; merge with base/tree hashes; detect untracked/content conflicts; preserve worktrees on failure; journal reconciliation.
- **Dependencies:** Workspace fingerprint and adapter result hashes.
- **Risk:** Dynamic edits make prediction imperfect and reduce parallel opportunities.
- **Test plan:** same tracked file, same untracked file, rename/delete, generated file, dependent tasks, non-overlapping work.
- **Acceptance criteria:** No accepted result is lost; unresolved overlap pauses with both versions preserved; docs call shared-tree task waves serial.
- **Size / type:** XL / repair.

### 4.4 Measure Specialization Value

- **Problem:** Caste count and ceremonies are used as proxies for better engineering.
- **Evidence:** No controlled outcome evidence found; routing is keyword/hard-coded.
- **Proposed change:** Run repeatable task corpus comparing minimal colony, added specialist, and plain adapter on correctness, criteria coverage, cost, time, regressions and human interventions.
- **Dependencies:** Hermetic harness and minimal castes.
- **Risk:** Model variance and provider changes complicate inference.
- **Test plan:** Multiple seeded runs and fixed repository fixtures; report distributions, not reliability percentages from prompt checks.
- **Acceptance criteria:** Default specialist must show material predefined benefit without unacceptable cost; otherwise remain optional.
- **Size / type:** L / add.

## Stage 5: Simplify Installation, Commands, And Platform Behaviour

### 5.1 Ship One Atomic Release Set

- **Problem:** npm, GitHub, binary and companions are independently published/installed.
- **Evidence:** Public versions diverge and mixed version install succeeds.
- **Proposed change:** Release manifest binds version, binary digests, companion digest, npm version, schema migrations, minimum/maximum state version, and signatures. Stage install then atomically activate; retain rollback.
- **Dependencies:** Release harness and state migrations.
- **Risk:** Cross-platform CI and npm/GitHub credentials become hard release dependencies.
- **Test plan:** Matrix install, missing asset, checksum mismatch, npm absent, interrupted activation, rollback, N-1 migration.
- **Acceptance criteria:** Release is unpublished/failed if any artifact or live install disagrees; active installation is never mixed.
- **Size / type:** XL / consolidate and repair.

### 5.2 Consolidate The Command Surface

- **Problem:** 396 catalog entries obscure the simple product path.
- **Evidence:** 356 public utilities; duplicate host/direct/finalizer commands.
- **Proposed change:** Publish beginner/guided/advanced namespaces from Minimum Viable Aether; mark internal finalizers hidden; deprecate aliases with machine-readable replacements.
- **Dependencies:** Claims register and adapter boundary.
- **Risk:** Scripts using internal commands need migration.
- **Test plan:** Catalog snapshot, deprecation output, completion/help usability test, script compatibility fixtures.
- **Acceptance criteria:** Top-level help presents no more than core jobs plus advanced namespaces; internal commands remain callable only under documented internal/testing contract.
- **Size / type:** L / remove and consolidate.

### 5.3 Publish An Honest Platform Contract

- **Problem:** Shared lifecycle wording hides different native agents, permissions, fallback, and wrapper semantics.
- **Evidence:** Codex delegate exclusion, OpenCode indirect Task dispatch, Claude bypass permissions, pin failure.
- **Proposed change:** Version adapter capabilities and expose `aether adapters inspect`. Wrappers are generated thin invocations. Docs show supported/degraded/unsupported by feature.
- **Dependencies:** Adapter contract.
- **Risk:** Visible capability gaps may reduce perceived parity.
- **Test plan:** Same manifest through every supported adapter; compare required invariants and declared limitations.
- **Acceptance criteria:** No platform claims a guarantee it cannot enforce; incompatible state cannot be produced.
- **Size / type:** L / consolidate.

### 5.4 Retire Duplicate TypeScript Authorities

- **Problem:** `.aether/ts-host` and `control-ts` overlap; generated JS/dist duplicates TS.
- **Evidence:** 40 source and 42 test TS/JS duplicate basenames; control adapters/event stream are stubs.
- **Proposed change:** Choose one host or no external host after vertical-slice comparison. Delete/quarantine the other; build generated JS only in release artifacts, not alongside source where avoidable.
- **Dependencies:** Adapter contract and core vertical slice.
- **Risk:** Useful tests/schemas may be lost without careful salvage.
- **Test plan:** Dependency/use graph; parity suite before removal; source package build from clean checkout.
- **Acceptance criteria:** Exactly one production orchestration owner per run; no success stub reachable; one source file per behavior.
- **Size / type:** L / remove and consolidate.

## Stage 6: Polish Documentation And Onboarding

### 6.1 Generate Documentation From Contracts

- **Problem:** Docs describe repo-local agents/skills and parity not matching installed/runtime reality.
- **Evidence:** Fresh scaffold docs conflict with actual ten-file setup and global assets.
- **Proposed change:** Generate command reference, adapter matrix, state diagrams, role roster, and release version from schemas/catalog/manifests. Keep narrative guides short and tested.
- **Dependencies:** Consolidated commands/platforms/release.
- **Risk:** Generated docs can still be misleading if schemas lack semantics.
- **Test plan:** Link/path/version checks; execute every quickstart command from released package.
- **Acceptance criteria:** Fresh user can follow one path without internal commands; every output gives accurate next action.
- **Size / type:** M / consolidate.

### 6.2 Build The Minimum Public Demo

- **Problem:** Current demos/snapshots can prove ceremony while hiding empty work.
- **Evidence:** `control-ts` demo and current live false completion.
- **Proposed change:** A fixed small brownfield fixture where user clarifies a change, research invalidates one assumption, plan revises, one real worker implements, verifier catches then clears a seeded failure, session resumes, and seal emits evidence.
- **Dependencies:** All core stages.
- **Risk:** Real model variability; use both deterministic and recorded real-adapter variants.
- **Test plan:** Automated deterministic run plus published unedited real run transcript/artifacts.
- **Acceptance criteria:** Demo artifact can be cloned, rerun, and audited; no hidden manual state edits.
- **Size / type:** M / add.

## Stage 7: Advanced Differentiation After Gates Pass

### 7.1 Adaptive Oracle And Research Providers

- **Problem:** Oracle can consume time without changing decisions; external provider contract is not stable.
- **Evidence:** 13% one-iteration result and disconnected planning link.
- **Proposed change:** Add provider plugins, marginal-value stop policy, conflict resolution, source freshness, and plan-impact requirement.
- **Dependencies:** Core research decision loop passes.
- **Risk:** Network cost, prompt injection, source licensing/quality.
- **Test plan:** Low/high confidence, conflicting sources, no decision impact, offline repo evidence first.
- **Acceptance criteria:** Every continued iteration identifies an unresolved decision it can change; otherwise stops.
- **Size / type:** L / add.

### 7.2 Optional Specialist And Delivery Plugins

- **Problem:** Valuable niche roles are currently default surface without stable extension boundaries.
- **Evidence:** Caste audit and shared permissions.
- **Proposed change:** Versioned plugin contract for skills, reviewers, research providers, verification policies, and delivery adapters. No custom lifecycle state writes.
- **Dependencies:** Core adapters/policies stable across two releases.
- **Risk:** Untrusted code/content, dependency and support burden.
- **Test plan:** Capability sandbox, signature/source validation, compatibility test kit, uninstall/upgrade.
- **Acceptance criteria:** Plugin failure cannot corrupt state or claim completion; removal leaves core colony resumable.
- **Size / type:** XL / add.

### 7.3 Reintroduce Proven Parallel Waves And Hive

- **Problem:** These are differentiators but currently unsafe.
- **Evidence:** Journeys F and G.
- **Proposed change:** Enable only behind acceptance-qualified policies; publish measured benefit and residual risk.
- **Dependencies:** Stage 3 and 4 acceptance.
- **Risk:** Reopens complexity and cross-project trust surface.
- **Test plan:** Full adversarial suites plus outcome/cost comparison.
- **Acceptance criteria:** Zero silent merge loss; zero unrelated knowledge injection in corpus; measurable benefit over disabled baseline.
- **Size / type:** XL / repair/add.
