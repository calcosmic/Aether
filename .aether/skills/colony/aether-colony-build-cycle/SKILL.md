---
source: shipped
name: aether-colony-build-cycle
description: Use when Codex is asked to colonize, plan, build, continue, swarm, or seal an Aether colony and must mirror wrapper orchestration safely
type: colony
domains: [aether, codex, colonize, planning, build, verification, swarm, seal, orchestration]
agent_roles: [queen, builder, watcher, scout, route_setter, tracker, archaeologist, auditor, probe]
workflow_triggers: [colonize, plan, build, continue, swarm, seal]
task_keywords: [aether colonize, aether plan, aether build, aether continue, aether swarm, aether seal, dispatch manifest, plan-only, finalize]
priority: high
version: "1.0"
---

# Aether Colony Build Cycle

## Installed Codex entrypoints

Use `$ant-colonize`, `$ant-plan`, `$ant-build`, `$ant-continue`, `$ant-swarm`, and `$ant-seal`. This document is installed as
`support/aether-colony-build-cycle.md` beside the public skill directories;
it is private support, not a separately discoverable helper skill. Public skills
resolve its path relative to their installed `SKILL.md`, never the working directory.
The executable commands and runtime-issued IDs remain `aether` identities.
Full workflow coverage and native-worker parity remain pending.

## Purpose

Give Codex the wrapper-equivalent behavior for the lifecycle commands where AI
orchestration matters: `colonize`, `plan`, `build`, `continue`, `swarm`, and `seal`. Runtime JSON
manifests remain authoritative. Codex may spawn workers and summarize results,
but it must not invent state or write state files by hand.

For beginners: the runtime prints the recipe and owns the kitchen ledger. Codex
can coordinate helpers, but it must use the recipe the runtime gave it.

## Required First Step

Run or inspect the guide for the command being handled:

```bash
aether command-guide <colonize|plan|build|continue|swarm|seal> --platform codex
```

If this skill and `command-guide` disagree, follow `command-guide` and update
the skill.

## Phase 199 Front-Door Contract

- Require automatic typed territory freshness from the runtime before planning;
  do not inspect or infer it from files.
- Offer an equal guided-build/Autopilot choice after an accepted plan, without
  preselecting either route. Show displayed Autopilot bounds before it runs.
- Keep concrete repair/debt receipts from runtime results, and allow independent safe-path continuation while only the unsafe or blocked path pauses.
- Autopilot stops at explicit seal. Force flags pass only when directly supplied by the owner. A forced-incomplete closure is not verified success.
- The nine installed public skills coordinate the existing runtime routes.
  Full workflow coverage and native-worker parity remain pending.

## Raw Bypass

If the user explicitly says raw, exact, no orchestration, or "just run this
exact command", run the literal CLI command they provided. Say briefly that the
Codex orchestration layer was bypassed.

## Live Worker Ceremony

For wrapper-orchestrated worker flows, the visible live agent stack is part of
the user experience. Spawn same-wave workers as visible Task/subagent panels
with caste-labelled descriptions. Do not use background-only dispatch as the
ceremony, do not say you will be notified later, and do not replace the live
stack with a markdown worker table.

## Guided Boundary Gate

For `plan`, `build`, heavy external-review `continue`, and `seal`, inspect
`result.orchestrator_boundary_guidance` and the matching manifest
`orchestrator_boundary_guidance` before any spawn ceremony, worker dispatch, or
finalizer packet. If guidance is active or `next` is `aether discuss`, stop the
lifecycle flow, show the guidance summary, route to `aether discuss`, and tell
the user to rerun `after_discuss_next` after the answer is resolved. Then request
a fresh host manifest; never reuse the pre-discuss manifest, and never ask,
answer, or store boundary questions in Codex chat or wrapper state.

## Plan Flow

If the user explicitly supplies `--repair-artifact`, run
`AETHER_OUTPUT_MODE=json aether plan --repair-artifact` before these planning
steps, render its scoped result, and stop this flow. No preset or worker is
needed; conflicting generation/revision flags are refused. An accepted
revision receives dependency validation only, preserving its plan and approval
bindings. Otherwise only the legacy `.aether/data/planning/phase-plan.json`
staging artifact may be repaired. Numeric and semantic task IDs share one
dependency contract across phases. Never rewrite only the active projection,
an accepted candidate or approval records; follow the runtime's next command.

1. Run `AETHER_OUTPUT_MODE=visual aether status` and use the runtime's lifecycle
   facts as context. Do not inspect or edit state files to infer authority.
2. Inspect the owner contract first:

```bash
AETHER_OUTPUT_MODE=json aether spec --inspect
```

   Continue only when Go reports the exact current Specification revision/hash
   `APPROVED`, its readable projection synchronized, and affected scope
   reconciled. Follow the returned exact approval, projection-repair, or
   reconciliation action otherwise. Specification approval is not plan
   acceptance.
3. If no valid policy was supplied, run `aether host plan` without a preset and
   render `preset_required` as exactly four equal choices: Fast 80/up to 4
   passes, Balanced 90/up to 6, Deep 95/up to 8, and Exhaustive 99/up to 12.
   Select nothing by default; invalid, cancelled, or interrupted input starts
   no worker.
4. After one exact choice, request the first staged manifest:

```bash
aether host plan --preset <fast|balanced|deep|exhaustive>
```

   Exact target/max flags may bypass only the card when Go maps them to one of
   those policies. Routine read-only phase research is automatic inside the
   selected preset; it has no separate approval checkpoint.
5. If completed phases exist and the owner is revising future work, preserve
   the goal and pass `--refresh`, `--revision-type`, and a concrete
   `--revision-reason`. Research and verification revisions also require
   repository-relative `--revision-evidence` files. Do not create a new colony
   merely to replan.
6. Save the structured response to an approved temporary file outside
   `.aether/data/`. Read only `result.plan_manifest.stage_manifest` as worker
   authority. It binds one authorization ID, run/pass/preset, approved
   Specification, base plan, prior card, input frontier, expected caste/result
   type, and the current caste's predecessor data. The only worker result types
   are `planning-scout-result/v1` and `planning-route-setter-result/v1`.
7. Apply the Guided Boundary Gate before any ceremony or dispatch. A fresh
   post-discuss manifest is mandatory; never reuse the pre-discuss response.
8. Render the runtime-owned spawn and wave ceremony for the current manifest:

```bash
AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony spawn-plan --workflow plan --manifest-file <manifest file>
AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony wave-start --workflow plan --manifest-file <manifest file> --execution-wave <execution_wave>
```

9. Dispatch exactly the one current stage as a visible live Task/subagent
   panel. The first stage is Scout. Route-Setter exists only in a returned
   `route_stage_manifest` bound to the exact Scout receipt. A later Scout exists
   only in a returned `scout_stage_manifest` after a complete iteration card.
   Never dispatch both castes together, predict a next stage, or reuse a
   consumed manifest.
10. Preserve every runtime-provided name, caste, brief, result contract,
    `permission_profile`, evidence frontier, weakest gap, Scout receipt, and
    candidate snapshot. Pass the brief verbatim and enforce its read budget,
    no-repeat guard, output contract, and stop condition. Mark a genuinely
    blocked worker `blocked`; never synthesize completion.
11. Call `aether spawn-log` before the worker and `aether spawn-complete` after
    its terminal result. Render `aether ceremony worker-complete`.
12. Write the unchanged current `plan_manifest` plus exactly one strict
    `scout_result` or `route_result` to an approved temporary completion file.
    Never combine stages, submit legacy whole-chain worker arrays, or reuse the
    packet. Finalize the one stage through:

```bash
AETHER_OUTPUT_MODE=json aether plan-finalize --completion-file <worker completion JSON>
```

13. After Scout finalization, render `stage_receipt`, admitted evidence, gaps,
    and the exact next boundary. Dispatch Route-Setter only from the returned
    `route_stage_manifest`.
14. If `decision_cards` are present, pause. Render the complete evidence-first
    batch, including decision, why now, evidence, Queen recommendation, choice
    consequences, affected IDs, prior-answer/revalidation state, and resume
    condition. Collect all exact owner choices and resubmit only the issued
    resume binding.
15. Go alone resolves the answer batch as `direct_resume` or
    `successor_spec_required`. Direct resume returns the exact previously
    authorized stage. A successor is a new DRAFT: follow the exact `aether spec`
    approval action and affected-scope reconciliation before another Scout.
    Never edit SPEC, choose the branch, or synthesize the receipt/state change.
16. Route-Setter proposes plan content and five readiness assessments;
    Go validates evidence and derives overall readiness, semantic delta,
    weakest gap, materiality, and stop policy. After every Route finalization,
    render the complete immutable `iteration_card` before doing anything else:
    fresh evidence, all five before/after values, overall versus target,
    weakest gap, semantic delta, reason, and `evidence_that_would_change`.
17. Continue only from a returned `scout_stage_manifest` targeting the weakest
    evidenced gap. A later material choice pauses only after the completed card.
18. When Go returns `plan_candidate`, label it `NOT ACTIVE` and inspect it:

```bash
AETHER_OUTPUT_MODE=json aether plan --candidate
```

    Render the full proposal, approved Specification/base/timeline bindings,
    complete card history, five scores, residual gaps and evidence that would
    change them, semantic delta, and Queen recommendation with
    producer/rationale/evidence. Planning stop is not acceptance.
19. Only after explicit owner confirmation execute the review result's entire
    `acceptance_command` verbatim. The generic legacy acceptance flag is not a
    shortcut. Stale or divergent acceptance leaves the active plan unchanged
    and routes back to `aether plan --candidate`.
20. Only a successful `acceptance_receipt` makes the PlanRevision READY. Then
    render plan closeout and offer the equal Codex choices
    `$ant-build 1` and `aether run`; preselect neither. For a revision, report
    preserved/affected/superseded/replacement IDs and discard every stale
    stage packet, answer token, candidate view, and acceptance command.

At every step, exact replay retains the existing Go-issued artifact/receipt;
divergent replay stops with state unchanged and the runtime's exact recovery
command. Codex never authors a receipt, score, recommendation, candidate status,
active revision, state transition, or next action.

## Colonize Flow

1. Run:

```bash
aether host colonize <args>
```

2. Save the full JSON envelope to a temporary manifest file outside
   `.aether/data/`.
3. Parse `result.colonize_manifest`. Never parse visual output as state.
4. Render the runtime-owned survey ceremony:

```bash
AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony spawn-plan --workflow colonize --manifest-file <manifest file>
```

5. Dispatch the runtime-specified Surveyor workers through the host platform
   with caste-labelled descriptions, runtime names, briefs, output paths, and
   skill sections.
6. Render `aether ceremony wave-start` before each surveyor wave.
7. Call `aether spawn-log` before each surveyor and `aether spawn-complete`
   after each terminal result.
8. After each terminal result, render `aether ceremony worker-complete`.
9. Finalize through:

```bash
AETHER_OUTPUT_MODE=json aether colonize-finalize --completion-file <worker completion JSON>
```

Then render the wrapper closeout:

```bash
AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow colonize --completion-file <worker completion JSON>
```

## Build Flow

1. Run `AETHER_OUTPUT_MODE=visual aether status`.
2. Surface active REDIRECT, FOCUS, and FEEDBACK signals compactly.
3. Run the Go runtime manifest command:

```bash
aether build <phase> --plan-only
```

4. Save the full JSON envelope to a temporary manifest file outside
   `.aether/data/`.
5. Parse `result.dispatch_manifest`.
5a. Coherent jobs: several related tasks become one job for one worker.
   Grouping is a proposal, never a decision.
   Go owns accepted groups, completion credit, retry, worktree reconciliation, and check-in policy; the wrapper proposes, renders, spawns, and submits.
   - `--job-proposal` is repeatable: one JSON object per group with `name`,
     `task_ids`, `owner_caste`, `relationship`, `benefit`, and optional
     `owner_reason`. A reason must name both the relationship and the
     benefit; "these are related" is not a reason.

```bash
aether build --job-proposal '{"name":"templates","task_ids":["2","3","4"],"owner_caste":"builder","relationship":"these tasks edit the same templates","benefit":"one worker avoids repeated setup and write conflicts"}' <phase> --plan-only
```

   - With no proposal the runtime still groups tasks joined by a dependency
     chain or by meaningful shared implementation files. Incidental overlap
     through a README, changelog, or dependency manifest joins nothing.
   - Read `job_decisions` and relay it plainly. Each entry's `status` is
     `accepted` or `refused`; a refusal names `offending_task_id`,
     `dependency_id`, and the `replacement_job_names` the runtime
     substituted. Only the refused group is repaired.
   - Each dispatch carries `job_name`, `job_reason`, `job_source` (`queen`,
     `automatic`, `single`, or `retry`) and `covered_task_ids` in order.
     Render them; never edit them, and never re-propose a refused grouping
     unchanged.
   - A real dependency cycle blocks dispatch for the whole phase, names the
     cycle, and names the plan repair. Surface it and stop.
5b. Team check-in: `checkin_requested` is the runtime's decision and
   `checkin_reason` says why. Never infer either from the flags passed.
   `one_worker_fast_path` means one worker with nothing left for the owner to
   decide -- render `checkin_summary` as a short non-blocking note and
   continue. `non_interactive` (autopilot or `--no-checkin`) skips the stage.
   `explicit_checkin`, `pending_owner_decision`, and `default_pause` all keep
   the full blocking check-in, including the forced-reviewer waiver flow.
   `--checkin` is the owner override that forces the pause on a decision-free
   one-worker build; combining it with `--no-checkin` is refused by name.
   For a blocking check-in only, before asking the owner to approve the team, run `AETHER_OUTPUT_MODE=json aether ceremony team-checkin --workflow build --manifest-file <manifest_file>`. Display `result.approval_card` verbatim in a fenced text block in the visible conversation immediately before the approval choices; a collapsed tool result or a generic sentence about the team is not enough. The card names each worker, its assignment and wave, and the required reviewers that run afterward. Keep the runtime roster unchanged. If an older runtime has no `approval_card`, render the visual team-checkin ceremony and relay its roster visibly before asking. Read `result.optional`, `result.waived`, and `result.waive_commands` for the existing trim/decline flow.
6. Apply the Guided Boundary Gate before rendering spawn ceremonies or spawning
   build workers.
7. Render the user-facing spawn ceremony:

```bash
AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony spawn-plan --workflow build --manifest-file <manifest file>
```

8. For Codex build, use the non-launching native bridge. The host's native
   `spawn_agent` is the sole launcher; never invoke a provider subprocess or
   `internal-worker-adapter` for the same assignment. Keep the runtime-selected
   team and execution waves; a small job can use one Builder plus checks.
9. Write requests as strict JSON in an absolute regular file under a new
   `aether-worker-request-*` directory in the system temporary directory. All
   operations require `schema_version: 1`, `phase`, and the exact manifest
   `execution_binding`. Reserve also requires `worker_name`, `task_id`, actual
   `host_session_id`, canonical `workspace`, and `host_permission: workspace_write`.

   For a task dispatch use its nonempty `task_id`. Auxiliary jobs such as an
   independent Watcher may omit that raw field. Use the runtime's
   `normalizedDispatchTaskID` convention: trim `stage`, `caste`, and `name`,
   join them with hyphens, lowercase, replace spaces with hyphens, then trim
   outer hyphens. Keep the accepted manifest unchanged; after reservation use
   the saved `worker.task_id` for all later operations.

```bash
aether codex-native-worker reserve --request <absolute temporary request file>
```

   Unsupported workspace/permission requests refuse before launch. Only a fresh
   `launch_allowed: true` permits one native `spawn_agent`. Use the returned
   dispatch's role and name; pass `worker.native.prompt` verbatim, including its
   wait instruction. Go assembles it from the capsule, verified
   `dispatch.brief_path` bytes (using inline `dispatch.brief` only when no path
   exists), matched skills and current new answers. Do not reconstruct the prompt.
   The child must wait without checks or edits. Never launch on replay.
   Per-child read-only or narrow write restrictions, separate native worktrees,
   and Aether-governed nesting are unsupported. A request that requires governed
   nesting sets `require_governed_nesting: true` and is refused. Inherited
   permissions do not prove separate child isolation. Never silently change lanes.
10. Save the actual child ID returned by the host. Bind with the same identity
    fields plus `launch_id` from `worker.provider_run_id`, `child_id`,
    `dispatch_sha256` and `prompt_sha256` from `worker.native`:

```bash
aether codex-native-worker bind --request <absolute temporary request file>
```

    Only after successful binding, send `worker.native.release` verbatim to that
    same child through the host native messaging tool. A stopped or uncertain
    launch stays unresolved; elapsed time never permits a duplicate child.
11. Retain raw child tool events and the child's terminal response. Immediately
    record that response before waiting on another child or staging:

```bash
aether codex-native-worker record --request <absolute temporary request file>
```

    Include all bound fields, `result` (the actual child's JSON terminal result),
    `source_event_id` (the terminal AgentMessage item's actual host ID), and
    `source_event_sha256` (SHA-256 of that exact raw JSONL line, excluding its
    trailing newline). Preserve the source line without reserializing it.
    The runtime accepts the installed Builder's `ant_name`, `tdd`, and
    `code_written` result, normalizing name/status in Go while preserving raw
    child JSON. If both `name` and `ant_name` appear they must agree. Every handoff's
    `verification_status` must be `pass`, `fail`, `partial`, `not_run`, or
    `unknown`. If the runtime rejects malformed output, ask the same child to
    correct its response before recording; never rewrite or replace an accepted
    terminal result.
    Missing/empty or mismatched results refuse. Never supply provider usage from
    worker prose. Unknown outcomes remain incomplete, never reported as success.
    Raw child-attributed `token_usage_record` events exist, but native usage is
    uncollected by Aether. Empty saved Usage means unreported, not zero cost.
    Parent totals and worker text cannot supply child measurements. Encrypted
    message exports show call/child linkage but cannot independently prove exact
    plaintext delivery.
    Keep `spawn-log`/`spawn-complete` and the visible native child panel truthful;
    render `ceremony worker-complete` after a saved terminal result.

11a. **Codex native material questions and answers.** While a bound child is
    working, relay its actual question through the runtime before asking the
    owner. Keep the same saved assignment/child fields and add
    `question: {question_id: <stable actual host question/event key>,
    question: <the child's exact question>}`:

```bash
aether codex-native-worker question --request <same bound worker request file>
```

    After saving a terminal result, call this operation without `question` to
    admit its saved handoff `open_decisions`. The runtime derives their stable
    keys from the saved terminal event and handoff location. Show each pending
    `decision.description` verbatim. The returned `answer_request_path` has the
    exact `native_binding` and question with an empty answer. Copy only the
    owner's actual answer into that file, then execute the exact returned
    `answer_command`:

```bash
aether decision-answer --native-request <runtime-returned answer_request_path>
```

    Never invent an answer, change its binding or use the generic text answer
    route. Native material clarification cannot authorize a checkpoint or
    reviewer waiver. Harness/predeclared answers are test authorization, never
    owner testimony. A repeated identical answer returns its original receipt;
    a conflicting or stale answer refuses without creating a replacement.

11b. For an answered, current live child, obtain the runtime's exact message:

```bash
aether codex-native-worker context --request <same bound worker request file>
```

    `awaiting_delivery` returns `context_delivery.payload`, `payload_sha256`,
    `decision_ids` and the exact `child_id`. Send those bytes unchanged to that
    child using the host native message operation. A read never acknowledges
    delivery. Only after observing the send complete, call `observe` with the
    exact binding, `observation_status: context_delivered`, the returned
    `context_delivery`, `context_send: {status: completed, child_id,
    message_sha256: <payload_sha256>}`, and the actual `observed_at`,
    `source_event_id` and `source_event_sha256`. Failed or queued sends remain
    unacknowledged; delivery alone does not prove consultation or useful influence.

    After public resume and inspection, run `question` for that same child.
    `answered` means do not re-ask; `no_updates` means no message is needed,
    and `delivered` is a historical receipt. A stale or paused refusal keeps the
    choice pending: use the runtime recovery path, never relabel old text.
    Terminal questions and answers remain attached to their saved handoff.
    Do not reopen a terminal child or forward its answer to another worker;
    surface the runtime-returned `next_command`.

12. A fresh parent first runs public `aether resume`, then
    `aether codex-native-worker inspect --phase <phase>`; the exact saved-binding
    `inspect --request <file>` route remains available. Inspection is read-only.
    Reuse saved terminal results without
    repeating the helper's edits. If real host evidence is inaccessible, report
    the missing capability; never invent it or respawn the finished helper.
    Resume only saved never-started assignments that the runtime admits.
    An unresolved launch stays unresolved; reconnect only to its actual child.
    Record real observations with `aether codex-native-worker observe --request <file>`.
    An interrupt request records `cancel_requested`, never `cancelled`.
    Only an actual host cancellation acknowledgement permits `cancelled`;
    idle, close, release, elapsed time and process loss are not that evidence.
    Never copy native results to legacy subprocess result files. The existing
    native journal and Go-owned stage remain the saved-result authority.
13. Task receipts cover actual proved work. Assigned `covered_task_ids` are scope,
    not credit. A worker that finishes only part of its job submits a
    `task_receipts` array: one entry per proved task with `task_id`, `status`,
    `summary`, `files_created`, `files_modified`, `tests_written`, and its own
    `handoff`. A task with no receipt is unfinished; never infer completion from
    a related file change.
    An accepted task receipt is admission, not completion credit: only the runtime's root-backed finalization can grant `completed_task_ids`.
    Never author `covered_task_ids` or `completed_task_ids` by hand in a manifest or in colony state; the runtime owns both.
14. Once required terminal records exist, stage from the existing journal with
    schema version, phase, and execution binding only:

```bash
AETHER_OUTPUT_MODE=json aether codex-native-worker stage --request <absolute temporary request file>
```

    Parse `result.completion_path`. The saved worker results already survive a
    stopped chat before this aggregate exists. Staging never launches or credits.

15. Finalize through the durable packet only:

```bash
AETHER_OUTPUT_MODE=json aether build-finalize <phase> --completion-file <Go-owned completion_path>
```

Then render the wrapper closeout:

```bash
AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow build --completion-file <Go-owned completion_path>
```

16. Read the finalizer's own answer instead of assuming a job finished whole.
   Each dispatch's `completed_task_ids` is what the runtime actually
   credited. `recovery_job` set to true means part of the job was proven and
   part was not: `unfinished_task_ids` lists what remains,
   `parent_attempt_id` and `retry_attempt_id` link the appended recovery
   attempt to the original one, and `recovery_command` is the exact command
   that redispatches only the unfinished tasks. Relay `recovery_command`;
   never ask a new worker to redo credited work.
17. The separate direct/subprocess route's worktree mode takes one job, one worktree, one branch, and one
   merge-back. The runtime admits receipts, syncs only what it admitted back
   to the project root, then credits. Anything the worker touched but never
   proved is neither synced nor destroyed -- it stays on a preserved branch
   the runtime names. Report that plainly rather than as lost or as done.
   Native build admission refuses worktree mode; the native route must not
   apply these subprocess instructions or allocate replacement worktrees.

## Continue Flow

Default path:

```bash
AETHER_OUTPUT_MODE=visual aether continue --verification-depth standard <args>
```

Use external review orchestration only when the user explicitly requested
`--classic-ceremony`, heavy review, or the runtime asks for wrapper-spawned
review workers. In that case, request the runtime manifest:

```bash
aether host continue --dry-run --classic-ceremony <args>
```

Save the JSON manifest envelope to a temporary file, parse
`result.manifest.continue_manifest`, apply the Guided Boundary Gate before rendering
spawn ceremonies or spawning reviewers, and spawn only the planned reviewers as
visible live Task/subagent panels with caste-labelled descriptions. Use
`aether ceremony spawn-plan`, `aether ceremony wave-start`, and
`aether ceremony worker-complete` around the live reviewers. Call
`aether spawn-log` before each reviewer and `aether spawn-complete` after each
terminal result. Pass each reviewer brief verbatim; it contains read cache
discipline. If a reviewer keeps re-reading the same unchanged file or artifact,
mark it `blocked` with the missing context instead of waiting through another
loop. Collect results, finalize through `aether continue-finalize`, then render.

```bash
AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow continue --completion-file <worker completion JSON>
```

## Swarm Flow

Watch mode stays direct:

```bash
AETHER_OUTPUT_MODE=visual aether swarm --watch
```

For bug-destroyer targets, use the external worker contract:

```bash
AETHER_OUTPUT_MODE=json aether swarm --plan-only <problem>
```

1. Save the full JSON envelope to a temporary manifest file outside `.aether/data/`.
2. Parse `result.swarm_manifest`. Never parse visual output as state.
3. Render the runtime-owned spawn ceremony with `aether ceremony spawn-plan`.
4. Preserve manifest wave order: investigation workers first, then builder,
   then watcher.
5. Use runtime-provided names, castes, roles, task IDs, briefs, and response
   contracts.
6. Render `aether ceremony wave-start` before each same-wave group.
7. Spawn each same-wave group as visible live Task/subagent panels with
   caste-labelled descriptions.
8. Call `aether spawn-log` before each worker and `aether spawn-complete` after
   each terminal result.
9. After each terminal result, render `aether ceremony worker-complete`.
10. Finalize through:

```bash
AETHER_OUTPUT_MODE=json aether swarm-finalize --completion-file <worker completion JSON>
```

Then render the wrapper closeout:

```bash
AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow swarm --completion-file <worker completion JSON>
```

## Seal Flow

1. Run `AETHER_OUTPUT_MODE=visual aether status`.
2. Run:

```bash
aether host seal <args>
```

3. If the runtime returns blockers or recovery guidance, surface that and stop.
4. Save the full JSON envelope to a temporary manifest file outside `.aether/data/`.
5. Parse `result.seal_manifest`.
6. Apply the Guided Boundary Gate before rendering spawn ceremonies or spawning
   final-review workers.
7. Render the runtime-owned spawn ceremony with `aether ceremony spawn-plan`.
8. Use runtime-provided names, castes, task IDs, briefs, and skill sections.
9. Render `aether ceremony wave-start` before each final-review wave.
10. Spawn final-review workers as visible live Task/subagent panels with
   caste-labelled descriptions.
11. Call `aether spawn-log` before each worker and `aether spawn-complete` after
   each terminal result.
12. After each terminal result, render `aether ceremony worker-complete`.
13. Finalize through:

```bash
AETHER_OUTPUT_MODE=json aether seal-finalize --completion-file <worker completion JSON>
```

14. Render the wrapper closeout:

```bash
AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow seal --completion-file <worker completion JSON>
```

15. Follow runtime Porter readiness output only after `seal-finalize` succeeds.
   Do not run delivery commands unless the user chooses them.
16. Keep sealing explicit: the Go runtime owns final review, preflight,
    confirmation, transaction, and rendering. After sealing, run
    `AETHER_OUTPUT_MODE=visual aether status` first to review retained state;
    `aether entomb` remains a separate optional owner-confirmed archive-and-clear
    action. Never invoke entomb automatically.

## Guardrails

- Do not write `.aether/data/COLONY_STATE.json`, `session.json`, `CONTEXT.md`,
  `HANDOFF.md`, planning artifacts, or pheromone files by hand.
- Do not invent worker names, castes, task IDs, waves, or dispatches.
- Preserve each manifest worker's typed `permission_profile`. Never broaden `repository_read_only`, and never present `behavioral_restrictions` under `workspace_write` as host-enforced isolation.
- Do not parse visual output for authoritative state. Use JSON mode for
  manifests.
- If Claude/OpenCode lifecycle wrapper behavior changes, update the matching
  YAML, this skill, and `cmd/command_guide.go` together.
