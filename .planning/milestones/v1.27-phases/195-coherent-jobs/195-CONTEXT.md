# Phase 195: Coherent Jobs - Context

**Gathered:** 2026-08-27
**Status:** Ready for planning

<domain>
## Phase Boundary

Phase 195 turns several related plan tasks into one dependency-safe job for one
worker. The Queen may propose a grouping with a reason; the Go runtime validates
the proposal, records every covered task, and refuses unsafe ordering before
dispatch. With no proposal, meaningful shared files and dependency chains form
coherent jobs by default. Grouped jobs must credit every task they demonstrably
complete, work in both in-repo and worktree modes, and preserve honest recovery
when only part of a job finishes.

This phase also folds in the owner's explicit request that a decision-free,
one-worker build proceed without a blocking team check-in. It does not change
the deterministic verification floor (Phase 193), choose the worker castes or
risk reviewers (Phase 194), add cost reporting (Phase 196), redesign closing
cards (Phases 197–198), or build the deferred spec-authoring command.

</domain>

<decisions>
## Implementation Decisions

### Grouping boundaries
- **D-01:** Default grouping clusters tasks connected by a dependency chain or
  by meaningful shared implementation files. Incidental overlap through common
  bookkeeping files such as `README.md`, changelogs, or dependency manifests
  does not combine otherwise unrelated work.
- **D-02:** Automatic grouping preserves caste boundaries. The Queen may
  explicitly group related work that originally mapped to different castes,
  but must name one suitable owning worker and state why that worker can carry
  the whole job. The runtime still validates relevance and safety.
- **D-03:** Job size uses a soft limit, not an arbitrary hard task count. An
  unusually large coherent cluster is split unless the Queen gives a specific
  reason why one worker should retain end-to-end ownership.
- **D-04:** Every grouped job's reason names both the relationship and the
  benefit. Example: "Tasks 2–6 modify the same templates, so one Builder avoids
  repeated setup and write conflicts." A generic "these tasks are related" is
  insufficient.

### Invalid grouping recovery
- **D-05:** Validate a proposed job's dependency order before dispatch. If it
  would execute a task before an unmet dependency, refuse the grouping by name
  and show both the offending task and dependency. The unsafe job never runs.
- **D-06:** Keep every safe job and repair only the affected group. The runtime
  visibly substitutes dependency-safe jobs for the refused grouping; it does
  not silently reorder the Queen's explicit proposal and does not explode the
  entire phase back into one-worker-per-task dispatches.
- **D-07:** A real dependency cycle is a hard planning error because no safe
  fallback order exists. Block dispatch for the phase and print the named cycle
  plus the plan-repair action.

### Partial completion and retry
- **D-08:** A grouped job is not all-or-nothing when honest partial proof
  exists. If a worker completes four of six tasks before failing, credit only
  the tasks carrying explicit task-level completion evidence. Never infer
  completion merely because a related file changed or the worker named a task.
- **D-09:** Partial credit must bind each completed task to its own requirements,
  claimed files, and verification evidence through a manifest-validated
  receipt. If no trustworthy task-level receipt exists, none of the grouped
  job's unfinished work is credited.
- **D-10:** Recovery creates a new dependency-safe job containing only the
  unfinished tasks. Revalidate their dependencies against the credited tasks,
  append a new attempt linked to the original grouped job, and never overwrite
  the first worker's receipt or ask a new worker to redo proven work.

### One-worker fast path
- **D-11:** A build with exactly one implementation worker skips the blocking
  team check-in automatically, regardless of how many coherent tasks that job
  covers, provided no forced-reviewer waiver or other owner decision is
  pending. Autopilot remains non-interactive as it is today.
- **D-12:** The fast path still renders a compact, non-blocking summary before
  dispatch. It names the worker, covered tasks, grouping reason, and why no
  approval is required. It is not the full interactive check-in card.
- **D-13:** Any pending owner decision keeps the blocking check-in. In
  particular, a named-risk reviewer announced for the checking step must still
  give the owner the Phase 194 waiver opportunity even if build itself has only
  one Builder.
- **D-14:** Add an explicit `--checkin` override so the owner can force the
  pause for a one-worker build. Supplying `--checkin` and `--no-checkin`
  together is a named flag conflict; the runtime must not guess precedence.

### Codex's Discretion
- Choose the concrete policy mechanism and default list for distinguishing
  meaningful shared implementation paths from incidental housekeeping paths.
- Choose the measurable soft-limit signals (brief size, criteria count, path
  count, or a combination); there must be no unexplained magic task cap.
- Choose the manifest/CLI field names for Queen job proposals, grouped-job
  reasons, and per-task partial receipts, preserving backward compatibility.
- Choose exact plain-English wording and compact-summary layout while retaining
  every fact required by D-04, D-05, and D-12.

### Folded Todos
- `.planning/todos/pending/2026-08-23-one-worker-build-skips-the-checkin-pause.md`
  — reverses Phase 194 D-14 for decision-free one-worker builds. Phase 195 is
  the natural home because grouping several tasks into one job should remove
  both redundant worker startups and the redundant approval pause. D-11–D-14
  preserve owner control when an actual decision remains.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Governing scope and rulings
- `.planning/ROADMAP.md` — Phase 195 goal, dependency, and four success
  criteria; these are the fixed phase boundary.
- `.planning/REQUIREMENTS.md` — JOBS-01 through JOBS-04 verbatim.
- `.planning/research/v1.27-milestone-brief.md` — feature 3, "Coherent jobs,
  not one worker per task," including the CalVault proof fixture.
- `.planning/research/priority-spec-v3-backlog.md` — Stage 6 coherent
  workstreams and the D12 amendment pulling workstream clustering into v1.27.
- `.planning/decisions/2026-08-21-owner-rulings-priority-spec-v3.md` — D12's
  owner priority: stop burning tokens on pointless spawning.
- `.planning/phases/194-the-queen-decides-the-team/194-CONTEXT.md` — the
  per-worker reason contract, named-risk reviewer/waiver behavior, and the old
  always-pause decision this phase deliberately narrows.
- `.planning/todos/pending/2026-08-23-one-worker-build-skips-the-checkin-pause.md`
  — the owner's explicit one-worker fast-path request and files/tests it touches.

### Existing grouping and field evidence
- `.planning/HARDENING-PLAN.md` — Phase 184/H5 field record: six CalVault
  file-copy workers became one in in-repo mode, with worktree mode deliberately
  excluded at that time.
- `cmd/dispatch_coalesce_test.go` — executable CalVault fixture and current
  invariants for dependent, independent, same-caste, and merged work.
- `cmd/codex_build.go` — `codexBuildDispatch`, `CoveredTaskIDs`,
  `DeclaredPaths`, `coalesceSequentialDispatches`, `dispatchesFormOneJob`, and
  the current worktree exclusion.
- `cmd/merged_dispatch_task_credit_test.go` — end-to-end lock proving a
  runtime-coalesced worker credits every covered task.
- `cmd/wrapper_bundled_completion_test.go` — manifest-validated
  `covered_task_ids`, duplicate/unknown/unevidenced-claim rejection, and the
  wrapper-bundled completion path.
- `.planning/milestones/v1.26-phases/191.1-field-hardening-close-the-four-2026-08-21-field-reported-def/191.1-CONTEXT.md`
  — FIELD-02 decisions and the completion-packet trust boundary.
- `.planning/todos/completed/2026-08-21-completion-packet-cannot-express-bundled-work.md`
  — the two-repo field failure that distinguished runtime coalescing from
  wrapper-created bundling.

### Check-in and dependency machinery
- `pkg/agent/task_graph.go` — existing dependency graph, topological ordering,
  missing-dependency, and cycle validation primitives.
- `cmd/ceremony_team_checkin.go` — runtime-owned team check-in rendering.
- `cmd/ceremony_team_checkin_test.go` — `TestOneWorkerTeamStillPauses`, which
  must be inverted or replaced to lock D-11 while keeping pending decisions safe.
- `.claude/commands/ant/build.md` — canonical wrapper check-in flow; keep the
  `.claude/commands/ant-build.md` and `.opencode/commands/ant/build.md` copies
  byte-identical when changing it.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `coalesceSequentialDispatches` already merges an explicit dependency chain
  and `mergeDispatchInto` already combines instructions, declared paths, and
  `CoveredTaskIDs`; Phase 195 extends this path rather than building a second
  grouping engine.
- `taskWaves` plus `pkg/agent/task_graph.go` already provide dependency-aware
  wave construction and cycle detection that can validate Queen proposals.
- `CoveredTaskIDs` already travels through the manifest and completion
  reconciliation. Runtime-coalesced and wrapper-bundled completion tests
  establish the trust checks to reuse for grouped and partial receipts.
- Phase 194's `selected_reasons`/per-worker reason path and team-card rendering
  provide the transport for grouped-job reasons.

### Established Patterns
- Go owns validation, manifest truth, completion credit, and state changes;
  wrappers may propose and render but must not invent authoritative groups.
- Assert the final dispatch list and task state, not only a decision record.
- Reproduce the real CalVault six-batch shape first, then keep independent work
  parallel and fail closed on missing, duplicated, or unevidenced task claims.
- Build attempts and retries are append-only; later recovery must not rewrite
  the first worker's outcome.

### Integration Points
- `plannedBuildDispatchesWithJudgement` and the dispatch manifest need one
  canonical grouping pass shared by proposed and no-proposal paths.
- Worktree mode currently bypasses coalescing in `cmd/codex_build.go`; grouped
  jobs must reconcile one worker/worktree against the union of covered task
  paths without weakening path-conflict checks.
- `mergeExternalBuildResults`, `reconcileCompletedBuildTasks`, and the
  completion-packet schema are the receipt boundary for full and partial task
  credit.
- `result.checkin_requested`, `--no-checkin`, the ceremony renderer, and all
  three wrapper copies implement the current blocking check-in and must agree
  on the decision-free one-worker fast path plus `--checkin` override.

</code_context>

<specifics>
## Specific Ideas

- Acceptance fixture: six dependent CalVault file-copy batches over the same
  source list produce one Builder, not six, with no explicit grouping proposal.
- Reason example: "Tasks 2–6 modify the same templates, so one Builder avoids
  repeated setup and write conflicts."
- Owner intent from 2026-08-23: "one worker build should go straight through."
  "Straight through" means no blocking approval, not invisible dispatch.
- User-facing refusals, summaries, and repair guidance must use plain English
  understandable by someone who has never opened the repository.

</specifics>

<deferred>
## Deferred Ideas

- A user-facing spec-builder command remains a separate v1.28+ capability. Its
  todo matched only generic words and was not folded into job grouping.
- Cost/token rendering for grouped jobs belongs to Phase 196.
- Universal closing-card and richer live display changes belong to Phases
  197–198; Phase 195 changes only the pre-dispatch fast-path summary it needs.

### Reviewed Todos (not folded)
- `.planning/todos/pending/2026-08-20-spec-builder-feature.md` — already
  deferred in Phase 194 and unrelated to grouping or completion semantics.

</deferred>

---

*Phase: 195-coherent-jobs*
*Context gathered: 2026-08-27*
