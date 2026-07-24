# Provider-Backed Plan Revision

Date: 2026-07-24

Status: implemented and verified through the complete normal and race Go
suites, vet, native build, and Windows amd64 cross-build. No public release
was performed.

## Verdict

The research-to-plan loop is now proven through real provider execution
instead of synthetic planning. A research question flows through a real Oracle
provider process into a persisted evidence artifact; real Scout and
Route-Setter provider processes consume that evidence and produce the
replacement plan; the accepted revision binds the exact evidence artifact,
its content hash, and the planning run; a restarted process restores the
revision; and the revised phase builds and verifies through real Builder and
Watcher provider workers.

For beginners: the previous checkpoint proved that Aether could *store* a
revised plan safely. This checkpoint proves that a real research worker's
finding actually *changes* the plan through real planning workers, and that
the changed plan then drives real build work. The gap recorded in
`11-core-lifecycle-truth-implementation.md` ("The revision journey uses
synthetic replacement planning; real Oracle/Scout/Route-Setter provider
execution remains an acceptance gap") is closed.

## What The Compiled Journey Proves

`TestCLIProviderBackedPlanRevisionJourney` in `cmd/blackbox_harness_test.go`
drives the actual compiled `aether` binary plus the deterministic provider
fixture through:

1. A colony mid-flight: phase 1 completed, phase 2 invalidated by research.
2. `aether oracle` through a real provider process, persisting
   `.aether/oracle/synthesis.md` as the revision evidence artifact.
3. `aether plan --refresh` (no `--synthetic`) with
   `--revision-type research --revision-reason ... --revision-evidence
   .aether/oracle/synthesis.md`. Real Scout and Route-Setter provider
   processes run; the Route-Setter writes
   `.aether/data/planning/phase-plan.json`, and the activated plan reports
   `dispatch_mode: real` and `plan_source: worker-artifact`.
4. The completed phase's JSON is byte-identical across the revision; the
   active revision is number 2 with `research` reason type, a non-empty
   planning run ID, and the Oracle evidence binding.
5. `aether resume` in a fresh process restores the active revision identity
   and reason.
6. `aether build 2` through real Builder and Watcher provider workers
   finalizes `BUILT` with durable attempt claims.
7. `aether continue` enforces the provider-authored bound criteria: the
   `app.txt` artifact fingerprint recorded at build time, the claims check,
   the Watcher result, and the `tests` check all pass, and the colony
   reaches `COMPLETED`.
8. The adapter invocation log records real provider processes for oracle,
   scout, route_setter, builder, and watcher castes.

## Fixture Changes

`cmd/testdata/adapter-fixture/main.go` (test-only infrastructure):

- Detects planning and Oracle castes whose briefs do not carry the
  `- Caste:` header line: Scout via the planning-only response-contract
  marker, Route-Setter via its scout-output instruction line, Oracle via the
  Oracle response-contract heading.
- Scout mode returns a high-confidence `scout_report` in its claims, which
  the planning path consumes through the same
  `scoutReportFromWorkerResult` contract as any provider.
- Route-Setter mode writes `.aether/data/planning/phase-plan.json` with
  `bound_v1` evidence requirements (an `app.txt` artifact binding plus
  `claims`/`watcher`/`tests` checks) and claims the file, which the
  planning path loads through the same `loadWorkerPlanArtifact` freshness
  and claim-validation contract as any provider.
- Oracle detection means the deterministic provider now also writes the
  Oracle response-file payload for `aether oracle` dispatches, so the
  synthesis evidence derives from real provider responses rather than a
  degraded loop.

No production code changed in this checkpoint.

## Typed Assumption/Decision Impact Links: Not Added

The checkpoint instruction was explicit: add typed assumption and decision
impact links only where the journey demonstrates a missing contract, and do
not create another plan store.

The journey demonstrated no missing contract. The existing revision record —
reason type, reason, evidence paths, input and planning evidence hashes,
planning run ID, plan hash, and preserved/superseded/replacement phase IDs —
carried the research rationale through planning, restart, build binding, and
verification without a gap. Adding assumption/decision IDs now would be
speculative structure without a demonstrated consumer, so they remain
unbuilt and this decision is recorded instead.

## Behavioral Finding: Keyword Phase-Mode Inference

The first journey run failed in an instructive way: the Route-Setter
artifact described the replacement phase as a replacement for "the
research-invalidated approach", and `InferPhaseMode`
(`pkg/colony/colony.go:570`) keyword-matched "research" in the description
and classified the replacement phase as `discovery`. The build then
dispatched a research Oracle worker instead of a Builder, produced no
project artifact, and correctly failed criterion verification.

This is existing behavior, not a regression: any planner-authored phase
whose name or description contains words like "research", "explore", or
"spike" silently becomes a read-only discovery phase even when its tasks are
implementation work. A real Route-Setter revising away a failed research
assumption is *likely* to use those words. The fixture now avoids the
keyword, but the sharper fix — letting the planner declare the phase mode
explicitly instead of inferring it from prose — is a candidate for a future
checkpoint and is recorded here rather than silently worked around.

## Verification

| Check | Result |
| --- | --- |
| `go test ./cmd -run '^TestCLIProviderBackedPlanRevisionJourney$' -count=1` | Pass |
| Related revision/adapter cluster (`TestCLIVersionedPlanRevisionSurvivesRestartAndBindsNextBuild`, `TestCLICompiledInstallToSealJourney`, adapter, plan-revision, and codex-plan tests) | Pass |
| `go test ./... -count=1` | Pass; `cmd` 281.468s |
| `go test ./... -race -count=1` | Pass; `cmd` 373.385s |
| `go vet ./...` | Pass |
| Native build and Windows amd64 cross-build | Pass |
| `git diff --check` | Pass |

TypeScript host, retired control, and npm bootstrap suites were not rerun
because this checkpoint changed only Go test infrastructure and one Go test;
the previous checkpoint's results for those suites still stand.

## Remaining Limits

- The proof uses the deterministic provider fixture. It validates the real
  subprocess, prompt, claims, artifact, and finalization contracts; it does
  not validate model behavior, which no offline test can prove.
- The direct Go planning path dispatches Scout and Route-Setter through the
  platform invoker in-process. The host-mediated path (`plan --plan-only`
  to `plan-finalize`) already runs the same workers through the typed
  adapter boundary; both share the revision contract, but only the direct
  path is exercised by this journey.
- Keyword phase-mode inference (above) remains a sharp edge for
  planner-authored phase prose.
- Plan and continue dispatches still lack build-equivalent attempt/worker
  journals, as recorded in `15-durable-build-run-identity.md`.

## Next Checkpoint

Replace first-accepted worktree conflict handling with declared ownership
and one atomic whole-wave reconciliation decision (handoff item 2), followed
by the Hive trust redesign (item 3).
