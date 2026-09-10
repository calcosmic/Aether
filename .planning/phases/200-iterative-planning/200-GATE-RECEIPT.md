---
phase: 200-iterative-planning
plan: 40
generated_at: 2026-09-10T09:51:51Z
repository_revision: 68a6fd169eb5551ba39e6d365921ed5a340fbb2d
repository_tree: 785cd1c6ffae74bb437d4259e06ee7a9c76d4caa
branch: oracle-reinstate
go_version: go1.26.5 darwin/arm64
implementation_gate: pass
phase_200_owned_gates: pass
owner_product_acceptance: not_claimed
waiver_used: false
supersedes: plan-23-blocked-by-inherited-phase-199-inventory
---

# Phase 200 Implementation Gate Receipt

## Outcome

All seven gates are green on one tested tree. The Phase 199 focused checks that
Plan 23 recorded as an inherited red baseline (the vocabulary subtest and
parent plus the two receipt validators) were repaired under Plan 200-26 and now
pass. The Phase 200 targeted battery, the complete Classic contract corpus, the
public planning journey, source parity, and the repository-wide normal and race
commands all exit zero.

No waiver, expected-failure classification, or skip was used to reach this
result. This receipt is the implementation gate evidence required by executed
Plan 23; it does **not** claim Phase 205 owner product acceptance, publication,
deployment, or release completion.

Plain English: the full test suite — every test, run under the race detector as
well — now passes end to end for the first time in the project's recorded
history. Running the complete suite (which had never before finished) exposed a
backlog of real defects that were hiding behind an always-truncated run; those
were fixed under Plan 200-55 and are listed in its summary.

## Tested Tree

- Base revision: `68a6fd169eb5551ba39e6d365921ed5a340fbb2d`
- Tree: `785cd1c6ffae74bb437d4259e06ee7a9c76d4caa`
- Branch: `oracle-reinstate`
- Go: `go version go1.26.5 darwin/arm64`
- Final evidence timestamp: `2026-09-10T09:51:51Z`
- Dirty files during the final gates: none introduced by this plan. Only these
  pre-existing, user-owned items remained untouched and unstaged, identical
  before and after the run: modified `.planning/config.json` and
  `.planning/ROADMAP.md`; untracked `.gsd/`; untracked
  `.planning/phases/199-front-door-and-classic-contract/199-PATTERNS.md`.
- Execution isolation: shared checkout. The `cmd` package runs through the
  bounded complete-suite controller (Plan 200-55); every other package runs
  normally.

## Final Gate Commands

The seven canonical gates below ran from the repository root in this exact
order, each exiting zero, with UTC start/finish, duration, and a SHA-256 of the
captured stdout+stderr retained. Rows use the canonical command; see the
Execution-Ceiling Amendment note beneath the table for the `-timeout` flag this
machine requires to run the two repository-wide gates to completion.

| Gate | Exact command | Result |
| --- | --- | --- |
| Phase 199 focused | `go test ./cmd -run '^(TestCurrentVocabulary199($\|/)\|TestPhase199GateReceiptSchema$\|TestPhase199GateReceipt$)' -count=1` | PASS — 4s; exit 0; output `dedcf24e…50e8ab6` |
| Phase 200 targeted | `go test ./cmd -run '^Test(RepositoryBootstrapContainment200\|PlanningMutationSession200\|PlanningTimelineConcurrentProcesses200\|SpecificationIntegrity200\|PlanCandidateSemanticIntegrity200\|PlanCandidateAcceptanceIntegrity200\|PlanCandidateAcceptanceConcurrentProcesses200\|PlanningWriterCoverage200\|PlanningWriterConcurrentProcesses200\|PlanningNumericBoundaries200\|BuildStartTransaction200\|BuildStartCallers200\|BuildStartConcurrentProcesses200\|BuildStartLegacyHelpersRetired200\|PlanCandidateExpiry200\|PlanningExpiryPresentation200\|PlanningAdversarial200\|PlanningGapEdgeAccounting200)$' -count=1` | PASS — 205s; exit 0; output `9dff4815…d82610e` |
| Classic contract | `go test ./cmd -run '^TestClassicContract' -count=1` | PASS — 80s; exit 0; output `4da726cd…ced0db7dd` |
| Public planning journey | `go test ./cmd -run '^(TestPlanningPublicPaths200\|TestPlanningRealRepo200)$' -count=1` | PASS — 29s; exit 0; output `fc103bb7…b7b7901e` |
| Source parity | `go run ./cmd/aether source-check` | PASS — 147 surfaces checked: 16 canonical, 5 retired mirrors, 126 generated wrappers; 0 findings; state effect `none`; output `3f018043…42064bb8` |
| Repository-wide | `go test ./... -count=1` | PASS — 1136s; exit 0; all 4,794 discovered `cmd` tests executed exactly once, 20 packages green; output `7569c669…b976d394` |
| Repository-wide race | `go test ./... -race -count=1` | PASS — 1140s; exit 0; all 4,794 discovered `cmd` tests executed exactly once under the race detector, no data-race diagnostics, 20 packages green; output `d024b203…07028117` |

### Execution-Ceiling Amendment (owner-approved, 2026-09-10)

The two repository-wide gates require an explicit generous `-timeout` to run to
completion on this machine, and the receipt records their measured wall time
rather than a sub-11-minute claim. This is a deliberate, owner-accepted
amendment to Plan 23's original sub-11-minute expectation:

- The `cmd` package alone contains 4,794 top-level tests; several hundred spawn
  real `aether`/`git` subprocesses. Process creation serializes in-kernel,
  capping throughput near ~345 tests/min regardless of parallel workers, so the
  complete corpus floors near ~19 minutes (normal and race alike).
- A bare `go test ./...` carries Go's default 10-minute timeout; the tool sends
  SIGQUIT at that deadline. The Plan 200-55 controller reads the caller's
  `-test.timeout` and stops **orderly** just under it — printing complete
  per-lane accounting of what ran and what could not — rather than dying
  signal-killed. Given an explicit budget (here `-timeout=40m` for normal,
  `-timeout=95m` for race), it runs the whole corpus.
- The measured standard is therefore **~19 minutes per full gate**, ~20 minutes
  for race. The owner accepted this standard on 2026-09-10 and queued per-test
  subprocess-cost reduction as its own backlog item (ROADMAP Pending Todo,
  Phase 201 / `WORK-08`).

The literal canonical commands, with Markdown escaping removed:

```sh
go test ./cmd -run '^(TestCurrentVocabulary199($|/)|TestPhase199GateReceiptSchema$|TestPhase199GateReceipt$)' -count=1
go test ./cmd -run '^Test(RepositoryBootstrapContainment200|PlanningMutationSession200|PlanningTimelineConcurrentProcesses200|SpecificationIntegrity200|PlanCandidateSemanticIntegrity200|PlanCandidateAcceptanceIntegrity200|PlanCandidateAcceptanceConcurrentProcesses200|PlanningWriterCoverage200|PlanningWriterConcurrentProcesses200|PlanningNumericBoundaries200|BuildStartTransaction200|BuildStartCallers200|BuildStartConcurrentProcesses200|BuildStartLegacyHelpersRetired200|PlanCandidateExpiry200|PlanningExpiryPresentation200|PlanningAdversarial200|PlanningGapEdgeAccounting200)$' -count=1
go test ./cmd -run '^TestClassicContract' -count=1
go test ./cmd -run '^(TestPlanningPublicPaths200|TestPlanningRealRepo200)$' -count=1
go run ./cmd/aether source-check
go test ./... -count=1   # + explicit -timeout=40m on this machine
go test ./... -race -count=1   # + explicit -timeout=95m on this machine
```

## What the complete run first exposed and fixed (Plan 200-55)

Because the suite had never before run to completion, extending coverage
surfaced eight genuine defects, all repaired before this receipt:

1. `pending-decisions.json` strict decode rejected the blocker-flag fields that
   legitimately share the file (broke overnight autopilot).
2. `spec --repair-projection` and discuss settlement opened no planning mutation
   session, so every projection repair failed.
3. Whole-failure `build-finalize` could not commit: guards demanded a partial
   retry plan that a zero-credit dispatch can never produce.
4. Second-round plan acceptance (revising an already-built plan) was rejected,
   erased completed-work credit when forced, and then wedged build authority as
   permanently unreconciled — three symmetric fixes in the plan-impact and
   plan-revision derivation.
5. Shelf commands failed on a fresh repository (not-exist detection missed
   storage's error wording).
6. Recovery-orchestrator and preflight source-order fixtures predated the
   canonical build-start contract (200-34).
7. Platform-dependent Phase 199 tests were pinned to a fixed platform.
8. The Phase 199 gate-receipt validator broke when `.planning/config.json` was
   committed; the commit was reverted to preserve the recorded pre-existing
   fingerprint.

## Automated Verification Versus Product Acceptance

These gates establish implementation semantics, source parity, and absence of
detected Go data races across all executed code. They do not replace owner
review of the product experience. The two human UAT checks Plan 40 carries
remain outstanding for the standard GSD verifier and Phase 205:

1. In disposable real repositories, exercise both Claude and OpenCode managed
   surfaces through init, discuss, specification review/approval, a two-pass
   Balanced plan, candidate review, and exact acceptance — confirming Scout and
   Route-Setter visibly alternate, every card explains evidence/scores/weakest
   gap/delta/stop cause, material decisions alone interrupt, and build/run
   appears only after acceptance.
2. Review the specification, decision, iteration, candidate, acceptance, impact,
   and recovery screens at narrow, normal, and wide terminal widths — confirming
   authority is never clipped or ambiguous and expiry plus stale-before-expiry
   recovery read clearly without JSON.

Phase 205 remains responsible for owner product acceptance; separate
publish/deploy/release workflows remain required before distribution.
