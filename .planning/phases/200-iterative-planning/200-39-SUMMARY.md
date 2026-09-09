---
phase: 200-iterative-planning
plan: 39
subsystem: planning-integration-verification
tags: [go, tdd, cobra, adversarial-testing, classic-contract, public-parity, process-races]

requires:
  - phase: 200-27
    provides: Physically contained repository and storage bootstrap
  - phase: 200-29
    provides: Canonical Specification identity recomputation
  - phase: 200-31
    provides: Independently derived candidate acceptance and execution authority
  - phase: 200-37
    provides: Candidate expiry, stale standing, and refresh-only recovery
  - phase: 200-38
    provides: Shared terminal, JSON, lifecycle-fact, and Next Up candidate presentation
provides:
  - Five integrated hostile-input families exercised through real Cobra child processes
  - A valid two-pass planning journey proving exact acceptance before build or run authority
  - Twenty-two Phase 200 Classic corpus cases bound to executable causal Go proofs
  - Public parity across init, discuss, spec, and plan runtime, source, wrapper, contract, guide, and flag surfaces
  - Before-and-after protection for live GSD, configuration, state, and Phase 199 evidence
affects: [phase-200-verification, classic-parity, planning-public-surfaces, phase-201]

tech-stack:
  added: []
  patterns: [real Cobra child-process attacks, executable corpus proof symbols, read-only public parity, protected-input fingerprints]

key-files:
  created: []
  modified:
    - cmd/planning_adversarial_200_test.go
    - cmd/planning_real_repo_200_test.go
    - cmd/planning_public_paths_200_test.go
    - cmd/classic_contract_test.go
    - cmd/testdata/classic-contract/v1/cases.json
    - cmd/testdata/classic-contract/v1/mechanisms.json

key-decisions:
  - "Integrated lifecycle claims must execute real Cobra or named causal test processes; source-text counters are not sufficient proof."
  - "Managed wrappers remain read-only projections of Go authority, and source-check drift requires a separately owned plan rather than an undeclared wrapper edit."
  - "Only a current candidate exposes its exact acceptance command; stale and expired candidates expose only the exact live aether plan --refresh recovery."

patterns-established:
  - "Causal corpus rows name a public command, positive behavior, hostile refusal, recovery command, and resolvable Go test symbol."
  - "Public read tests snapshot repository inputs before nested integration processes and require structural, content, configuration-value, and Git-ref equality afterward."

requirements-completed: [SYNTH-02, CEC-03, PLAN-01, PLAN-02, PLAN-03, PLAN-04, PLAN-05, PLAN-06]

duration: 50min
completed: 2026-09-09
---

# Phase 200 Plan 39: Integrated Planning Proof Summary

**Real Cobra attacks, causal Classic evidence, and synchronized public surfaces now prove the repaired two-pass planning journey without changing live Aether state or granting build/run authority before exact owner acceptance.**

## Performance

- **Duration:** 50 min
- **Started:** 2026-09-09T04:13:01Z
- **Completed:** 2026-09-09T05:02:42Z
- **Tasks:** 3
- **Files modified:** 6

## Accomplishments

- Exercised specification forgery, candidate semantic forgery, physical path escape, concurrent stale writers, and expiry/stale recovery through real Cobra or deterministic OS-process boundaries, with positive controls and no-change snapshots.
- Completed the valid Balanced two-pass journey and proved that candidate review offers no build/run action until the exact receipt-bound acceptance succeeds, after which both execution routes have matching authority.
- Expanded the Classic contract to 22 Phase 200 cases covering six integrated mechanisms; every new row resolves through the Go AST to an executable causal proof.
- Bound init, discuss, spec, and plan semantics across canonical YAML, Claude/OpenCode managed projections, public contracts, the Codex command guide, live Cobra flags, terminal output, and JSON output.
- Re-ran the Phase 199 vocabulary and gate-receipt nodes and protected live `.planning`, `.gsd`, and Phase 199 inputs with exact before/after fingerprints.

## Task Commits

Each task followed a fail-first TDD boundary and was committed atomically:

1. **Task 1 RED: Define integrated planning attack matrix** - `4938cab8` (test)
2. **Task 1 GREEN: Prove hostile planning boundaries** - `88afd161` (test)
3. **Task 2 RED: Define failing integrated corpus proofs** - `3452d98d` (test)
4. **Task 2 GREEN: Bind Classic corpus to causal proofs** - `3cabbad2` (test)
5. **Task 3 RED: Define failing public parity boundary** - `c212018f` (test)
6. **Task 3 GREEN: Prove public planning parity** - `1f33e7ea` (test)

**Plan metadata:** `9ad69060` (docs), followed by a scoped provenance-restoration commit.

## Files Created/Modified

- `cmd/planning_adversarial_200_test.go` - Five attack families, all 14 edge mappings, migrated deterministic timeline race, and protected-input integration fingerprint.
- `cmd/planning_real_repo_200_test.go` - Complete valid two-pass candidate review, exact acceptance, and post-acceptance build/run authority journey.
- `cmd/planning_public_paths_200_test.go` - Four-surface lifecycle parity, live flags, exact acceptance/refresh presentation, source-check, and Phase 199 subprocess gates.
- `cmd/classic_contract_test.go` - Semantic-field, duplicate-mechanism, public-vocabulary, AST-symbol, and causal-child-process validation.
- `cmd/testdata/classic-contract/v1/cases.json` - Six bounded integrated Phase 200 proof rows added to the Classic corpus.
- `cmd/testdata/classic-contract/v1/mechanisms.json` - Current public mechanism vocabulary with retired internal finalizer removed from the public surface.

## Decisions Made

- A mapping counts only when its target test runs successfully in a child process. Merely naming a symbol or counting a tag cannot satisfy edge or Classic evidence.
- The existing JSON schema already permits the required semantic metadata under `expected.semantic_fields`; no corpus schema or scenario-enum expansion was needed.
- `aether plan-finalize` remains an internal wrapper invocation where required, but is not described as a public Classic command.
- Source parity stayed clean, so no wrapper, contract, canonical YAML, command-guide, or production file was edited.

## Verification

- Task 1 focused attacks, edge map, and real-repository journey: 29 passed.
- Task 2 full Classic contract: 145 passed.
- Task 3 public/adversarial/Phase 199 focused command: 62 passed; standalone source-check passed.
- Combined plan-specific gate across adversarial, edge, real-repo, Classic, public, and Phase 199 roots: 81 passed.
- Four integrated roots repeated twice with `-count=2`: 100 passed.
- The same four roots under `go test -race`: 50 passed with no data races.
- `rtk go vet ./cmd`, diff checks, deletion checks, and exact six-file ownership passed.
- `go run ./cmd/aether source-check` passed 16 canonical surfaces, 5 retired-mirror absences, and 126 generated wrappers with zero findings and `state_effect: none`.
- Protected hashes after every gate remained exact: STATE `635d8e5c…`, config `90391e73…`, Phase 199 UAT `8d9c95d6…`, Phase 199 PATTERNS `0359d922…`, and aggregate `.gsd` `14dfbf2c…`.
- The repository-wide test suite was not run; this plan explicitly requires only its bounded commands.

## TDD Gate Compliance

- Task 1 RED introduced the integrated attack and real-repository boundaries before the migrated process proof and valid journey made them green.
- Task 2 RED failed because `SYN-200-01` still exposed retired internal `aether plan-finalize` as public; GREEN replaced that public vocabulary and bound all six mechanisms to causal proofs.
- Task 3 RED used explicit failing boundaries for four-surface parity, exact acceptance/recovery, Phase 199 composition, and protected inputs; GREEN passed all four without production edits.
- Git history contains every RED commit before its corresponding GREEN commit.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking Fixture] Added an owned migrated timeline process-race proof**
- **Found during:** Task 1 GREEN
- **Issue:** The pre-existing `TestPlanningTimelineConcurrentProcesses200` fixture stops before dispatch because `dimension_assessments[0].content_hash` does not match its canonical body, so it cannot exercise the intended concurrent-writer boundary.
- **Fix:** Added `TestPlanningTimelineConcurrentProcesses200Migrated` in the owned adversarial file, preserving real OS processes and deterministic pipe barriers while building canonical current fixtures.
- **Files modified:** `cmd/planning_adversarial_200_test.go`
- **Verification:** The migrated test passed alone, through the attack family, through the 14-edge map, twice repeatedly, and under the race detector.
- **Committed in:** `88afd161`

**2. [Rule 3 - Blocking Fixture] Anchored the Classic positive journey inside candidate lifetime**
- **Found during:** Task 2 GREEN
- **Issue:** An existing Classic causal journey reviewed a candidate with a wall-clock value outside its now-enforced lifetime, preventing the positive acceptance path from proving its mechanism.
- **Fix:** Used `candidate.CreatedAt.Add(time.Minute)` so the proof remains deterministic and strictly before expiry.
- **Files modified:** `cmd/classic_contract_test.go`
- **Verification:** The complete Classic contract passed with 145 checks, including every causal child proof.
- **Committed in:** `3cabbad2`

---

**Total deviations:** 2 auto-fixed (2 Rule 3 blocking fixtures)
**Impact on plan:** Both corrections stayed inside the six owned files, preserved deterministic process semantics, and added no production or public authority.

## Issues Encountered

- The original unowned `TestPlanningTimelineConcurrentProcesses200` remains blocked at `cmd/planning_mutation_session_200_test.go:360` by its noncanonical dimension-assessment hash. The owned migrated proof covers the required race without weakening validation.
- Two pre-existing unowned projection-repair tests, `TestSpecCommandRepairProjectionWithoutChangingAuthority` and `TestSpecProjectionDetectsAndRepairsTamperOrDeletionFromCanonicalState`, still invoke repair without the now-required mutation session. They were unsuitable as the new corpus proof; `TestSpecificationIntegrity200` supplies the canonical recomputation proof instead.
- These are out-of-scope fixture migrations, not regressions in the six owned files. No full-suite waiver or false green was recorded.
- The installed GSD state advance again removed Phase 200 identity/provenance keys and calculated milestone-wide phase completion into the current-phase progress fields. The established scoped closeout restored `current_phase`, `current_phase_name`, the zero-complete-current-phase convention, and `state_head`; Plan 40 remains the next incomplete plan. The requirements command reported the eight IDs as not found because their checked entries include titles inside the bold span, but all eight were already visibly `[x]` in `REQUIREMENTS.md`, so that file required no edit.
- No authentication, dependency, package, architectural, or source-parity blocker occurred.

## Known Stubs

None. The only matched phrase, “not available,” is a real assertion diagnostic for live command resolution, not placeholder behavior.

## Threat Surface

- No production writer, network endpoint, authentication path, dependency, schema, wrapper, contract, or public command was added.
- Hostile tests cover elevation through forged authority, tampering across public contracts, physical containment, concurrent effect ordering, and expiry recovery.
- Protected-input fingerprints include content, structure, metadata, parsed configuration values, and Git refs around nested public and Phase 199 subprocesses.

## Independent Verification Still Required

The five source-plan prohibitions remain explicitly unresolved until the independent verifier records closure; this execution does not self-approve them:

- Plan 27: no repository-scoped lifecycle mutation may escape physically contained data roots.
- Plan 29: copied digest fields alone may not confer approved, accepted, buildable, or runnable authority.
- Plan 31: persisted candidate semantic deltas or authority impacts may not serve as their own proof.
- Plan 37: stale or expired candidates may not revive or extend an old acceptance action.
- Plan 38: no presentation may expose stale accept/build/run actions or hide whether state changed.

## Human Verification Still Required

All three human judgments remain explicitly unresolved; automated success is not owner approval:

- **PLAN-03:** Judge whether the live terminal hierarchy makes current Specification authority immediately understandable.
- **PLAN-05:** Judge whether the live candidate decision and early-stale explanation are clear enough to act on confidently.
- **SYNTH-02:** Judge whether the live Claude/OpenCode journey substantively feels like the Classic Scout-to-Route-Setter research and confidence loop from the February/April experience.

## User Setup Required

None - no packages, credentials, migrations, or external services were added.

## Next Phase Readiness

- Plan 39's bounded implementation, focused, combined, repeated, race, vet, source-parity, ownership, and protected-input gates are green.
- Phase 200 final verification can independently close or retain the five prohibitions and collect the three human judgments without rediscovering the causal proof map.
- The Classic-feel question is deliberately handed to a human; no test result has been substituted for that experience judgment.

## Self-Check: PASSED

- All six declared test/corpus files and this summary exist.
- Commits `4938cab8`, `88afd161`, `3452d98d`, `3cabbad2`, `c212018f`, `1f33e7ea`, and metadata commit `9ad69060` exist in RED/GREEN/closeout order.
- Focused, combined, repeated, race, vet, source-parity, diff, ownership, deletion, and protected-input checks pass.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-09*
