# colony/policies — per-file dispositions (161 residue)

**Decision date:** 2026-08-16
**Status:** rulings recorded. Phase 161 prescribed a wire-or-delete reckoning
for the zero-reader policy files; Phase 176 absorbed it and was cut. The
residue was owned by nothing until this record.

Reader census (verified by grep over cmd/ and pkg/, non-test, 2026-08-16):

| File | Go readers | Ruling |
|---|---|---|
| `dispatch-contract.yaml` | 1 (`cmd/codex_dispatch_contract.go`) | **Wired — keep.** |
| `review-depth.yaml` | 1 (`cmd/review_depth.go`) | **Wired — keep.** |
| `oracle-phase-directives.yaml` | 2 (`cmd/oracle_loop.go`) | **Wired — keep.** |
| `model-routing.yaml` | 0 | **Delete in a dedicated change.** Automatic model routing was rejected by the owner 2026-07-28 and reaffirmed in the v1.26 Out-of-Scope table. A routing file that nothing may ever read by default is a standing invitation to re-wire the rejected feature. Never default-on; if a strictly manual opt-in is ever wanted, it starts from a new decision, not from this file. |
| `autopilot.yaml` | 0 | **Delete in a dedicated change.** The autopilot's behaviour (pause conditions, replan cadence) now lives in `runCompatibilityAutopilot` with its disposition table in `.planning/decisions/autopilot-pause-conditions.md`; a parallel YAML nothing reads can only drift from it. |
| `memory-rules.yaml`, `pheromone-lifecycle.yaml`, `signal-rules.yaml`, `skill-creation.yaml` | 0 | **Delete in a dedicated change.** Each describes behaviour the runtime implements in Go with named tests; the YAML copies have never been read and can only contradict the code. |
| `safety-gates.yaml` | 0 | **Delete in a dedicated change** — with a caveat: Phase 161 wanted it to "actually gate something observable". The observable gates today are `GateResults` + `checkAutopilotPauseConditions`, both Go-owned and test-locked. Re-introducing YAML-configurable safety gates would make safety operator-weakenable, which the ratchet philosophy forbids. |

**Why "dedicated change" and not now:** the repo's retirement pattern
(test-deletion ledger, hub-publish propagation for `colony/` contents) makes
file deletion a change class of its own; bundling it into a restoration round
is how deletions get half-done. The rulings above are the decision; the
deletion commit executes them.

**colony/ distribution question (Phase 161 criterion 4/5):** the three wired
files resolve through a fallback chain that already covers downstream repos
(source checkout → hub copy → compiled defaults; see the resolution comment at
`cmd/oracle_loop.go:1666`), and dispatch works from compiled defaults when
`colony/` is absent. No further distribution machinery is needed for the
files that survive.
