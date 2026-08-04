# Learning System Authority

> Decision record: which learning system is authoritative, what happened to every
> duplicated stage, and the Hive Brain default.

**Status:** Decided
**Date:** 2026-08-04
**Implemented in:** v1.25 Phase 162 (Switch On Learning)
**Requirements:** LEARN-03, LEARN-04

---

## Context

Two systems under confusingly similar names both touch "learning" in this repo.

**`pkg/memory`** is the pipeline `CLAUDE.md`'s Wisdom Pipeline and Structural
Learning Stack sections describe: `Observe -> Promote -> Queen -> Consolidate`
(`pkg/memory/pipeline.go`), 60-day half-life trust decay and 7 trust tiers
(`pkg/memory/trust.go`), instinct storage with provenance (`pkg/memory/promote.go`),
and the eight curation ants (`pkg/agent/curation/*`) orchestrated at seal. Before
this phase, it was fully built and never invoked by anything — `consolidation-phase-end`
and `consolidation-seal` (`cmd/graph_consolidation_cmds.go:217`, `:315`) existed as
working CLI subcommands with zero runtime callers.

**`pkg/learn`** is two different things sharing one package name:

- `pkg/learn/wrappers.go` is a **thin forwarding shim**, not a competing
  implementation. `learn.NewPipeline` calls `memory.NewPipeline`;
  `learn.ConsolidationResult` is a type alias for `memory.ConsolidationResult`;
  every exported function on `learn.Pipeline`, `learn.ObservationService`,
  `learn.PromoteService`, and `learn.ConsolidationService` delegates directly to
  its `pkg/memory` counterpart with no independent logic. `cmd/graph_consolidation_cmds.go`
  imports it directly to construct the pipeline both consolidation subcommands run.
  It exists to avoid an import cycle between `cmd/` and `pkg/memory`, not to
  compete with `pkg/memory`. **It must not be retired** — doing so breaks both
  consolidation subcommands.
- `pkg/learn/learn.go`, `curator.go`, and `colony_store.go` are the real second
  system: `Entry`/`Evidence` capture, trust scoring, and classification, persisted
  to `entries.json` and `colony.db` via `learn.NewColonyStore`. This is the system
  `captureContinueLearning` (`cmd/codex_continue_finalize.go:1290`) runs on every
  `/ant-continue`, feeding the `## LEARNED MEMORY` worker-context section. Unlike
  `pkg/memory`, it was already live before this phase.

## Decision 1 — Authority

**`pkg/memory` is authoritative.** It is the pipeline the documentation describes,
the one with consolidation, decay, archival, and the eight curation ants — the
destination this phase switches on. Plan 03 gave `consolidation-phase-end` its
first runtime callers (both `/ant-continue` paths, on durable phase advance only);
plan 04 gave `consolidation-seal` its first runtime caller (`runSealConsolidation`,
invoked from `completeSealRuntime`).

**`pkg/learn`'s Entry/Evidence capture layer (`learn.go`/`curator.go`/`colony_store.go`)
is explicitly subordinate.** It is the live capture and trust-scoring front-end that
feeds the authoritative pipeline's observation stream, and it stays: it is the only
currently-working capture path, and retiring it would remove the colony's only
means of observing anything before `pkg/memory` has a capture surface of its own.
`pkg/memory/pipeline.go`'s `PipelineConfig.LearningValidator` callback field already
anticipates this relationship as a one-directional bridge (see Decision 3).

`pkg/learn/wrappers.go` is neither authoritative nor subordinate — it is
infrastructure, a call-routing shim into the authoritative system, and is excluded
from this decision's "one wins, one is retired" framing entirely.

## Decision 2 — Duplicated stages

Three candidate duplications were checked. One was real; the other two were not.

**QUEEN.md instinct promotion at seal — duplicated, resolved in `pkg/memory`'s favor.**
Before plan 04, `completeSealRuntime` had its own promotion loop
(`promoteInstinctLocal`, confidence >= 0.8, no application-history requirement)
writing to QUEEN.md's Wisdom section, entirely independent of `pkg/memory`'s
`QueenService.PromoteInstinct` (confidence >= 0.75 AND >= 3 recorded applications),
which plan 02 made reachable by resolving the promotion target to the actual file
colony-prime reads (`consolidationQueenPath()` -> `localQueenPath()`, i.e.
`.aether/QUEEN.md`, not the dead `.aether/data/QUEEN.md` the bare `"QUEEN.md"`
literal previously resolved to). Plan 04 reconciled the two writers: `runSealConsolidation()`
(the authoritative pipeline) now runs before the seal's own local/hive promotion
loop in source order, and that loop skips any instinct ID the pipeline already
promoted this seal via an ID skip-set — but still counts the ID toward the
seal summary's total, since the colony did promote it, a different writer did the
writing.

The subordinate loop is kept deliberately, not deleted: `QueenEligible` requires
at least 3 recorded applications, a bar young colonies never clear, so removing the
subordinate writer would make seal promote nothing for most colonies for most of
their life. The invariant that replaced the duplication is **one instinct produces
exactly one QUEEN.md entry**, enforced by `TestSealDoesNotDoublePromoteInstincts`
and `TestSealStillPromotesInstinctsWithoutApplicationHistory` (`cmd/seal_ceremony_test.go`).

**Hive promotion at seal — not duplicated.** `pkg/memory` has no hive stage at all;
the seal ceremony's existing hive-promotion loop remains its sole implementation.
Nothing to reconcile here.

**Promotion target reachability — not a duplication, but a load-bearing prerequisite
fixed by plan 02.** Before plan 02, every consolidation call site passed the bare
literal `"QUEEN.md"` to `pkg/memory`, which `storage.Store.resolvePath` silently
joined onto the store base directory, producing `.aether/data/QUEEN.md` — a file
colony-prime never reads. `consolidationQueenPath()` now resolves to
`localQueenPath()` instead, and `readQUEENMd`'s section allowlist was widened to
ingest `## Instincts`, so a promoted instinct is actually visible to worker prompt
assembly. Tested by `TestConsolidationQueenPathIsTheFileColonyPrimeReads`,
`TestPromotedInstinctReachesWorkerPrompt`, and `TestReadQUEENMdIngestsInstinctsSection`
(`cmd/consolidation_promotion_target_test.go`).

## Decision 3 — LearningValidator stays unwired, deliberately

`pkg/memory/pipeline.go`'s `PipelineConfig.LearningValidator` callback field is
invoked by the pipeline's observation handler when an observation's trust score is
>= 0.8, but `pipelineConfigForStore()` (`cmd/graph_consolidation_cmds.go:16`) never
assigns it — it sets only `ColonyName` and `QueenPath`. This means the field is
always `nil` at every one of `pkg/memory`'s runtime call sites today.

**This is a stated choice, not an oversight.** The field exists as a bridge from
the authoritative pipeline back into the subordinate capture layer (`pkg/learn`'s
Entry/Evidence system). Wiring it would mean `pkg/memory` calling back into
`pkg/learn` on every high-trust observation — inverting the dependency direction
Decision 1 establishes, where `pkg/memory` is authoritative and `pkg/learn`'s
capture layer is subordinate and upstream of it, never downstream. This phase
leaves the bridge unpopulated on purpose; a future phase that wants bidirectional
validation should treat that as a new, separately-justified decision, not a gap
this one left open by accident.

## Decision 4 — The Hive Brain default

**The default flips to `promote` (D-01).** `AETHER_HIVE_POLICY` unset or empty now
resolves to `promote`: cross-colony wisdom is both read into worker context and
high-confidence instincts are promoted to the hive automatically at seal. Before
this phase, the unset default was `off` (`hivePolicyOff`,
`cmd/hive_policy.go` pre-Phase-162), and a second, independent gate —
`hiveRetrievalOptedIn()` (`cmd/hive.go:303`, a per-colony consent file) — silently
vetoed retrieval even when an operator had explicitly set the policy to enable it.

**The consent-file mechanism is retired entirely (D-02).** Its path resolution,
read/write functions, and the `hive-opt-in`/`hive-opt-out` cobra commands were
deleted along with all four call sites that checked it
(`cmd/hive.go`, `cmd/context_weighting.go`, `cmd/colony_prime_context.go` twice).
There is now exactly one control surface for cross-colony wisdom flow, not two —
one honest switch beats two gates where the second silently overrides the first.

**Full resolution table:**

| `AETHER_HIVE_POLICY` value | Resolves to | Worker retrieval | Seal-time promotion |
|---|---|---|---|
| unset / empty | `promote` | Yes | Yes |
| `read` / `inject` / `on` | `read` | Yes | No |
| `promote` / `full` | `promote` | Yes | Yes |
| `off` | `off` | No | No |
| any unrecognized value (typo) | `off` | No | No — plus a one-line stderr warning naming the offending value |

The unrecognized-value row is a deliberate fail-safe: a typo must not silently
widen cross-repo data flow, so it fails closed rather than falling through to
`promote`. This resolves `RESEARCH.md`'s open assumption A3 in favor of the
fail-safe branch. Tested by `TestHiveRuntimePolicyDefault` and
`TestHiveRuntimePolicyUnrecognizedWarns` (`cmd/hive_policy_test.go`).

**Scope boundary (D-03):** this is a default change and a documentation
correction — nothing more. Hive *trust redesign* (how confidence is computed,
contradiction handling, revocation mechanics) remains explicitly shelved per
`REQUIREMENTS.md`'s Non-Goals and is out of scope for this decision.

## Consequences

What is now true at runtime, and the named test that fails if it stops being true:

| Claim | Enforced by |
|---|---|
| `AETHER_HIVE_POLICY` unset/empty resolves to `promote`; `off` disables both retrieval and promotion; unrecognized values fail safe to `off` with a stderr warning | `TestHiveRuntimePolicyDefault`, `TestHiveRuntimePolicyUnrecognizedWarns` (`cmd/hive_policy_test.go`) |
| Worker retrieval is on by default and the `off` switch disables it; no consent-file gate exists anymore | `TestHiveWorkerReadIsOnByDefaultAndOffSwitchDisablesIt` (`cmd/hive_policy_test.go`) |
| `consolidation-phase-end`'s pipeline promotion target resolves to the actual `.aether/QUEEN.md` colony-prime reads, never the dead store-relative path | `TestConsolidationQueenPathIsTheFileColonyPrimeReads` (`cmd/consolidation_promotion_target_test.go`) |
| A promoted instinct's `## Instincts` section is ingested into `buildColonyPrimeOutput`'s worker-facing prompt | `TestPromotedInstinctReachesWorkerPrompt`, `TestReadQUEENMdIngestsInstinctsSection` (`cmd/consolidation_promotion_target_test.go`) |
| `/ant-continue` invokes phase-end consolidation on durable phase advance only, non-blocking, and never on a blocked continue | `TestContinueAdvanceInvokesPhaseEndConsolidation`, `TestExternalContinueAdvanceInvokesPhaseEndConsolidation`, `TestContinueWithoutAdvanceDoesNotConsolidate`, `TestRunPhaseEndConsolidationIsNonBlockingOnFailure` (`cmd/consolidation_lifecycle_test.go`) |
| `/ant-seal` invokes the full eight-ant curation pass, is never a dry run, and writes the curation report artifact | `TestRunSealConsolidationRunsAllEightAnts`, `TestRunSealConsolidationIsNeverDryRun`, `TestRunSealConsolidationWritesReportArtifact`, `TestRunSealConsolidationNonBlockingOnFailure` (`cmd/consolidation_lifecycle_test.go`) |
| Seal's two QUEEN.md instinct writers cannot double-promote the same instinct, and the subordinate writer still promotes instincts without application history | `TestSealDoesNotDoublePromoteInstincts`, `TestSealStillPromotesInstinctsWithoutApplicationHistory` (`cmd/seal_ceremony_test.go`) |
| Seal renders all eight named curation ants and the report path in its output | `TestSealRendersEightNamedAnts`, `TestSealRendersReportPath` (`cmd/seal_ceremony_test.go`) |
| A `--dry-run` on either consolidation subcommand never mutates state, including the relocated QUEEN.md target | `TestConsolidationPhaseEndDryRunDoesNotMutate`, `TestConsolidationSealDryRunDoesNotMutate` (`cmd/consolidation_dryrun_test.go`) |

No claim about runtime behaviour appears above without a named test — per
`CLAUDE.md`'s Definition of Done, an uncheckable claim about this pipeline is
removed rather than written, which is the exact failure this document exists to
correct after v1.10, v1.11, v1.13, and v1.23 each declared the pipeline restored
without one.

`cmd/docs_truth_test.go`'s `TestLearningDecisionRecordExists` pins this file's
continued existence and its mention of both `pkg/memory` and `AETHER_HIVE_POLICY`;
`TestLearningDocsDoNotClaimUnwiredConsolidation` pins that the retired claims this
document corrects do not reappear in `CLAUDE.md`, `AGENTS.md`, or
`.aether/docs/structural-learning-stack.md`.
