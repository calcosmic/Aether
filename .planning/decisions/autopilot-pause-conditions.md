# Autopilot pause conditions — classic-to-Go disposition

**Decision date:** 2026-08-16
**Status:** settled — the pause engine (`checkAutopilotPauseConditions`,
cmd/autopilot.go) is wired into the real run loop (`runCompatibilityAutopilot`),
checked after every build and after every phase advancement. Proven by
`TestGoldenAutopilotPauseConditions` (the parity anchor named in
PARITY_CLASSIC_VS_GO.md, which did not exist until this change).

Classic v5.4.0 enumerated ten pause conditions. Each one's Go disposition:

| # | Classic condition | Go disposition |
|---|---|---|
| 1 | Watcher `verification_passed == false` | `blocked` — the continue result blocks the run with a task-scoped redispatch (`aether build N --task N.N`) |
| 2 | Chaos finding severity critical/high | `critical_chaos_findings` — midden entries with category `chaos` and a `critical` tag |
| 3 | New blocker flags created | `active_blockers:N` — unresolved `blocker` entries on the pending-decision queue |
| 4 | Verification loop reported NOT READY | `not_runnable` — the default non-runnable state exit |
| 5 | Gatekeeper critical CVEs | `gate_failure:<name>` — any recorded gate result with `passed: false`; CVE findings are a gate result in the Go runtime, so this condition subsumes them |
| 6 | Auditor score < 60 | **Retired.** The Go runtime has no numeric auditor score; quality is expressed through gate results and verification, both already covered by conditions 1 and 5. Re-adding a score would be new machinery, not restoration. |
| 7 | Unresolved blocker flags | Same mechanism as #3 (`active_blockers:N`) |
| 8 | Runtime verification needed | `test_failures` — test-failure signals on colony state |
| 9 | All phases complete | `completed` stop reason + the restored celebrations |
| 10 | Replan trigger | `replan_due` — interval default restored to the classic 2 (was 0/off); `--continue` skips the next checkpoint |

Additional Go-only condition kept: `uncommitted_changes` (marker file), plus
`cancelled` / `timeout` / `max_phases_reached` as loop discipline the classic
prose loop never had.

**Headless:** a pause under `--headless` queues an `autopilot_pause` pending
decision (deduplicated) so the reason is reviewable later — the classic
headless contract. `TestRunHeadlessQueuesPendingDecisionOnPause`.

**Retired command:** `autopilot-check-replan` (no caller anywhere; the loop
owns the replan arithmetic). Orphan allowlist shrank 291 → 290; test coverage
recovered by `TestRunAutopilotReplanDue`; ledger entry in
`.aether/docs/retired-tests-ledger.md`.
