# Aether Framework Polish Audit: Executive Verdict

Audit date: 2026-07-21

Audited checkout: `ef12c596` (`v1.24`) plus a dirty, unreleased `1.0.42` working tree

Public releases observed: npm `1.0.22`; GitHub `v1.0.34`

## Post-Audit Implementation Status

The original verdict below remains the forensic baseline for the audited public releases. The unreleased working tree now repairs several of the largest runtime credibility failures:

- Provider failures, timeouts, malformed results, and implementation no-ops fail the process and cannot commit `BUILT`.
- Newly accepted plans require a `bound_v1` criterion-evidence contract. Existing plans are explicitly classified as `legacy_unbound` rather than being credited with proof they do not contain.
- Build attempts are durable. Wrapper completion packets are staged inside the attempt journal, bound to the immutable manifest, replayable exactly once, and surfaced by `resume` without worker redispatch.
- Conflicting worktree path ownership is detected before the losing result is synchronized.
- Cross-project Hive retrieval and seal-time promotion are disabled by default. `AETHER_HIVE_POLICY=read` enables retrieval; `promote` is required for automatic promotion.
- Claude, OpenCode, and Codex differences are represented by a typed platform contract included in `command-guide` output.
- A compiled, isolated CLI journey now passes `install -> lay-eggs -> init -> discuss -> Oracle research -> bound plan -> real provider build/continue -> resume -> seal`.
- A second compiled journey passes install/update preservation, byte-exact migration backup, explicit `legacy_unbound` classification, rollback guidance, platform-agent installation, and binary/hub version agreement.
- A staged `1.0.42` GoReleaser release now produces all configured archives plus checksums, installs through the packed npm candidate, rejects corrupt or wrong-version archives without replacing the old binary, recovers interrupted activation state, and performs a byte-exact N-1 migration rollback without changing project Queen or skill content.
- Versioned plan revision now preserves accepted completed work exactly, replaces only unfinished phases, records rationale and hashed evidence, survives restart, binds the next build, and rejects stale plan/evidence packets.
- Production subprocess dispatch now crosses one Go-owned adapter boundary. The TS host coordinates asynchronous waves but cannot select or launch provider CLIs, explicit provider pins fail without fallback, malformed terminal results fail closed, and `control-ts` is private/retired with isolated test state.
- Build and planning manifests now carry canonical typed permission profiles. Go rejects missing or broadened requests, Codex selects read-only/workspace sandboxes, Claude no longer uses `bypassPermissions`, and OpenCode uses an attested restricted primary router. Narrow test/ledger/survey/doc scopes remain explicitly behavioral inside workspace-write.

These changes move the unreleased runtime from an alpha toolkit toward a credible core framework. They do **not** replace the already-published skewed npm/GitHub artifacts, prove revision through live Oracle/Scout/Route-Setter providers, enforce narrow per-caste path scopes, make whole-wave worktree reconciliation atomic, or make global learning evidence-backed. Platform-native wrappers also remain a distinct explicitly selected launch mode for visible native panels. Aether should not yet be described as polished or fantastic.

## Verdict

Aether is a serious hybrid prototype with several production-grade primitives. It is not yet a polished or dependable software-development framework. The current product is part deterministic runtime, part agent methodology, part prompt/asset distribution system, and part experimental control plane. Those parts do not yet form one trustworthy lifecycle.

For beginners: Aether has built a strong safe, dashboard, filing system, and set of specialist job descriptions. The problem is that the foreman, the filing system, and the dashboard can disagree about whether work actually happened. In one controlled run, every worker failed or was cancelled and `build` still returned `ok:true`; in another, two workers changed the same file and one change silently disappeared; `continue` then completed a project with no application and no tests.

Current maturity: **alpha framework / advanced developer toolkit**. It can dispatch real model processes, persist state, create manifests, validate file claims, generate bounded context, run iterative Oracle research, and recover some interrupted state. Those are valuable foundations. The end-to-end promise is not credible until release installation, exit codes, completion semantics, plan iteration, adapter behavior, memory provenance, and conflict handling are repaired.

## Strongest Implemented Capabilities

1. Go-owned state models, file locks, atomic store helpers, manifests, finalizers, checksums, and many stale-evidence checks.
2. Real Claude, OpenCode, and Codex subprocess dispatch paths in `pkg/codex/`, with streaming and structured claim parsing.
3. Durable Oracle artifacts and an actual multi-iteration research loop.
4. Context assembly with explicit budgets, source ledgers, sanitization, and priority ordering.
5. Build and continue evidence structures that are substantially stronger than trusting a worker's prose alone.
6. Extensive unit and contract coverage, including useful regression tests for stale claims, interrupted builds, path normalization, and finalization.

## Largest Credibility Gaps

| Gap | Direct evidence | Priority |
| --- | --- | --- |
| Public install is broken | `npx --yes aether-colony@latest` (`1.0.22`) downloaded an archive containing `Aether` while bootstrap required `aether` | P0 |
| False build success | Interrupted six-worker build returned `ok:true`, set state `BUILT`, and recorded `build_completed` although all workers failed/timed out/cancelled | P0 |
| False project completion | `continue --skip-watchers` passed four unresolved verification commands as skipped and completed a project containing only an unrelated `shared.txt` | P0 |
| Silent parallel data loss | Two worktree builders created different `shared.txt` contents; both were marked merged and accepted, but only the last result survived | P0 |
| Plan revision needs live-provider qualification | The unreleased deterministic contract now passes, but typed assumption/decision impact and real Oracle/Scout/Route-Setter Journey E remain unproven | P1 |
| Platform architecture is split | Direct Go orchestration, `.aether/ts-host`, wrappers, and `control-ts` all describe or perform overlapping lifecycle behavior; `control-ts` adapters are success-returning stubs | P0 |
| Memory is not trustworthy | Hive accepts arbitrary source names as independent confirmation, reaches `0.95`, lacks contradiction/evidence/decay, writes without a lock, and leaks generic advice across projects | P0 |
| CLI errors are not process errors | `outputError` is used 534 times across 88 Go files and normally returns `nil`; observed JSON errors exited `0` | P0 |
| Permissions are claims, not controls | Codex castes receive `workspace-write`; Claude runs with `bypassPermissions`, including nominally read-only reviewers | P0 |
| Release versions diverge | Dirty source/npm `1.0.42`, GitHub `1.0.34`, npm `1.0.22`; install can pair `1.0.42` assets with `1.0.34` binary | P0 |

## Central Recommendation

Adopt **Option B: deterministic Go core plus explicit agent-runtime adapters**.

Go should exclusively own the versioned canonical state, migrations, locks, lifecycle transition validation, evidence ledger, recovery, install/update, and release integrity. A narrow adapter contract should own model preflight, worker launch, streaming, cancellation, capabilities, sandbox permissions, and structured results. Claude, OpenCode, and Codex must each implement and pass that contract. Editable assets define prompts and policies, but they must not become competing state or lifecycle engines.

Retire `control-ts` as a production path unless its adapters become real. Consolidate its useful schemas into the adapter contract. Keep one orchestration host during migration: either the mature `.aether/ts-host` behind the contract or the direct Go orchestrator, not both as equal public authorities. Markdown artifacts, `QUEEN.md`, `CONTEXT.md`, `HANDOFF.md`, surveys, and reports become derived, versioned projections of canonical records.

Opportunity cost: Aether must freeze visible feature work and spend a substantial milestone deleting or quarantining duplicated behavior. This delays new castes, platforms, ceremony, dashboards, and marketplace work. It is the cost of making existing claims true.

## Direct Answer To The Core Question

| Action | What must happen |
| --- | --- |
| **Preserve** | Go state/locking/finalizer/download primitives, real process dispatch, bounded context ledgers, structured claims, durable Oracle artifacts, ant identity, and short-lived user steering |
| **Simplify** | Reduce the public lifecycle to start, discuss/research, plan/revise, build, verify, status/resume, steer, doctor/update, and seal; reduce the default colony to five castes |
| **Repair** | Public install/version agreement, exit codes, cancellation/timeout semantics, criteria verification, plan artifact recovery, explicit adapter selection, signal expiry/conflicts, and brownfield grounding |
| **Remove** | Success-returning adapter stubs, in-memory production event stubs, default specialist gauntlets, automatic Hive influence, untruthful completion ceremony, and public exposure of internal finalizers/storage utilities |
| **Consolidate** | One canonical journal/evidence model, one active orchestration owner, one generated platform role source, one command contract, one project knowledge pipeline, and one verification decision record |
| **Redesign** | Plan revisions, adapter capability/permission contract, worktree ownership/merge protocol, typed knowledge scopes, Queen as a projection, and release activation/rollback |
| **Prove** | Released-artifact Journeys A-H, including no-op rejection, interruption recovery, changed-evidence replanning, cross-project isolation, overlapping edits, and N-1 upgrade |

## Top Five Actions

1. Block all success transitions unless an evidence policy for the phase passes; skipped verification is not passing verification.
2. Define one transactional state/event/evidence model and make every Markdown/JSON report a projection with run ID, source, and freshness.
3. Fix the public release train and test `npx`, direct Go download, install, update, rollback, and version agreement from released artifacts.
4. Replace implicit platform selection and success-returning stubs with a capability-negotiated adapter contract and enforced permissions.
5. Add black-box journey tests for interruption, no-op workers, changed assumptions, cross-project memory, and overlapping worktree edits.

## Freeze Now

- New castes, commands, memory layers, platforms, ceremony variants, domain skills, dashboards, marketplace work, and autonomous routing features.
- Further extraction of behavior into editable assets until the canonical-state and adapter boundaries are enforced.
- Reliability percentages, parity claims, and "self-organizing" or "never repeats a mistake" claims.

## Delete, Merge, Or Move Out Of Core

- Delete or quarantine `control-ts` success stubs and in-memory event stubs from runtime registration.
- Merge the four surveyor agents into one Colonizer/Surveyor role with four analysis lenses.
- Merge Keeper and Sage; merge Medic and Fixer; make Archaeologist, Weaver, Ambassador, Chronicler, Porter, Chaos, Includer, Measurer, and Gatekeeper optional skills/policies unless a distinct permission or tool boundary is proven.
- Hide the 356 public utility commands behind internal APIs, `aether debug`, or extension tooling. Preserve a beginner surface of roughly ten lifecycle commands.
- Consolidate `COLONY_STATE.json`, `session.json`, planning/build reports, Queen decision files, and SQLite projections behind a single event/evidence store.

## Is Aether Better Than GSD-Type Frameworks?

**Not overall today.** Aether is more ambitious and has stronger low-level ideas for persistent state, evidence, recovery, and scoped learning. It is objectively worse at installation reliability, time to first useful result, conceptual economy, lifecycle coherence, and trust in completion. A GSD, Superpowers, OpenSpec, or plain-agent user can reach a useful artifact faster and can more readily understand what controls behavior. Aether becomes better only when its persistent state can demonstrably survive interruption and changed evidence while preventing false completion. That is the differentiator to prove.

## Evidence Classification

The audit uses these statuses: proven and working; working with limitations; partially implemented; implemented but disconnected; stubbed or simulated; documentation-only; duplicated; obsolete; regressed; and unclear. No numerical reliability claim is retained without a reproducible test and explicit denominator.
