# Autopilot stop, queue, and normal-ending conditions

**Original decision date:** 2026-08-16

**Superseded:** 2026-09-01 by Phase 198.3

**Active authority:** `autopilotTriggerSpecs` in `cmd/autopilot_policy.go`, enforced by `runCompatibilityAutopilot`

## Phase 198.3 supersession

The earlier broad pause scanner and its rule that headless work was queued and
then stopped are no longer active. Phase 198.3 replaced them with typed trigger
codes and a separate disposition for headless and interactive runs.

For beginners: things only the owner can judge go onto a morning list during a
headless run. Broken, newly unsafe, or impossible work still stops immediately.
Reaching a user limit or finishing normally is neither a failure nor a pause.

## Active trigger contract

The strings below are durable control data. Rendered prose may explain them but
must not create a second policy.

| Trigger code | Headless | Interactive | Activates when | Runtime next action |
|---|---|---|---|---|
| `deterministic_verification_failed` | `stop` | `stop` | Deterministic verification still fails after its one bounded repair attempt. | `aether continue` |
| `auditor_score_below_floor` | `stop` | `stop` | A completed Auditor reports a valid score below 60. | `aether continue` |
| `critical_review_finding` | `stop` | `stop` | Current structured reviewer, audit, or security evidence contains a Critical finding. | `aether flags` |
| `blocker_count_increased` | `stop` | `stop` | The unresolved blocker count grows after a live stage baseline. | `aether unblock` |
| `blocker_escalated` | `stop` | `stop` | A blocker gains new `Source=escalation` evidence after the baseline. | `aether unblock` |
| `colony_not_runnable` | `stop` | `stop` | State is missing, corrupt, or cannot validly enter build or continue. | `aether status` |
| `provider_unavailable` | `stop` | `stop` | The required worker provider remains unavailable after its bounded readiness policy, or reports an immediate auth/binary failure. | `aether run` |
| `runtime_verification_needed` | `queue_and_continue` | `pause` | A current criterion needs hands-on owner verification that the program cannot perform. | Copy the emitted `aether decision-answer ...` command. |
| `visual_checkpoint_needed` | `queue_and_continue` | `pause` | Trusted current build claims contain user-interface work requiring an owner's visual judgement. | Copy the emitted `aether decision-answer ...` command. |
| `replan_due` | `queue_and_continue` | `pause` | The configured interval is reached **and** at least one unique, confirmed post-revision lesson exists. | `aether plan` |
| `cancelled` | `normal_stop` | `normal_stop` | The operator or parent context cancels the invocation after durable progress is preserved. | `aether run` |
| `worker_timeout` | `normal_stop` | `normal_stop` | A bounded worker timeout ends unfinished work after durable progress is preserved. | `aether run` |
| `max_phases_reached` | `normal_stop` | `normal_stop` | The explicit `--max-phases` limit is reached. | `aether run` |
| `colony_complete` | `normal_stop` | `normal_stop` | Every planned phase is complete. | `aether seal` |

## Evidence and safety rulings

- Auditor score 60 passes. A High finding remains visible in the phase and
  final report but does not stop by severity alone; only Critical does.
- An existing ordinary blocker is a stage baseline for `aether run`. The run
  may advance only when `compareBlockerSnapshots` reports neither count growth
  nor new escalation. The blocker remains unresolved and visible. Direct
  `aether continue` retains the strict Iron Law and does not receive this
  run-only capability. No path resolves, acknowledges, hides, or rewrites a
  blocker to make the gate pass.
- Runtime and visual owner checkpoints are durable pending decisions. A
  headless run queues them and continues; an interactive run pauses. They stay
  unresolved until the owner uses the exact emitted `aether decision-answer`
  command, and unresolved owner checkpoints block `aether seal`.
- Replan cadence alone is insufficient in headless mode. A current confirmed
  lesson is also required, and one unresolved note is accumulated per active
  plan revision. `--continue` bypasses the checkpoint.
- Provider readiness uses one successful cache entry per platform and phase.
  Build and continue share that entry; the next phase probes again. The default
  readiness timeout is 45 seconds, timeouts get at most two attempts, and auth,
  binary, login, or other failures get one.
- Every real run ending writes one versioned report. Elapsed time is invocation
  wall-clock time; token totals include measured ledger rows only, while absent
  or estimated telemetry stays explicitly unreported. `aether status` returns
  the stored report without reconstructing it from newer data.
- `aether run` never seals, switches providers, skips a phase, or retries a
  whole build. Completion is a normal boundary whose exact next action is
  `aether seal`; the owner still resolves queued work and blockers first.

`TestOvernightRunCompletesSixPhases` is the joined-up stamina proof. The typed
catalogue and focused evidence tests remain the smaller diagnostic anchors.

## Superseded 2026-08-16 record (history only)

Everything in this section is retained as decision history and is **not active
runtime guidance**. In particular, `checkAutopilotPauseConditions` and
`TestGoldenAutopilotPauseConditions` were removed as obsolete authorities.

Classic v5.4.0 enumerated ten pause conditions. The old Go mapping was:

| # | Classic condition | Superseded Go disposition |
|---|---|---|
| 1 | Watcher `verification_passed == false` | `blocked` with task-scoped redispatch |
| 2 | Chaos finding severity critical/high | `critical_chaos_findings` from midden text |
| 3 | New blocker flags created | `active_blockers:N` |
| 4 | Verification loop reported NOT READY | `not_runnable` |
| 5 | Gatekeeper critical CVEs | `gate_failure:<name>` |
| 6 | Auditor score < 60 | Retired at the time because numeric score machinery was thought absent |
| 7 | Unresolved blocker flags | Same mechanism as old condition 3 |
| 8 | Runtime verification needed | `test_failures` on colony state |
| 9 | All phases complete | `completed` stop reason |
| 10 | Replan trigger | Interval-only `replan_due`, default 2 |

The old record also treated `uncommitted_changes` markers and historical
gate/midden text as live pause inputs, and said any headless pause queued one
`autopilot_pause` decision. Phase 198.3 explicitly supersedes those clauses:
current typed stage evidence is authoritative, owner work has dedicated durable
types, and queue-and-continue is different from a terminal stop.

**Retired historical command:** `autopilot-check-replan`. The live run loop has
owned replan evidence and cadence since 2026-08-16; the command name remains in
this marked history only.
