# Phase 195: Coherent Jobs - Research

**Researched:** 2026-08-27
**Domain:** Go CLI job planning, dependency validation, completion evidence, and worktree reconciliation
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Grouping boundaries
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

#### Invalid grouping recovery
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

#### Partial completion and retry
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

#### One-worker fast path
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

### Deferred Ideas (OUT OF SCOPE)
- A user-facing spec-builder command remains a separate v1.28+ capability. Its
  todo matched only generic words and was not folded into job grouping.
- Cost/token rendering for grouped jobs belongs to Phase 196.
- Universal closing-card and richer live display changes belong to Phases
  197–198; Phase 195 changes only the pre-dispatch fast-path summary it needs.

#### Reviewed Todos (not folded)
- `.planning/todos/pending/2026-08-20-spec-builder-feature.md` — already
  deferred in Phase 194 and unrelated to grouping or completion semantics.
</user_constraints>

Source: `.planning/phases/195-coherent-jobs/195-CONTEXT.md`; text above is copied verbatim from the locked decisions, discretion, folded-todo, and deferred-ideas sections. [VERIFIED: codebase grep]

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| JOBS-01 | The Queen can group tasks into one job with a reason; the runtime validates dependency order and refuses, by name, a grouping that violates `depends_on` | Add structured job proposals, run whole-plan cycle/missing-dependency validation first, then validate each proposal without silently reordering it; retain safe proposals and default-repair only the refused group. [VERIFIED: codebase grep] |
| JOBS-02 | A grouped job's brief carries every covered task's criteria and its completion credits every covered task (the existing `covered_task_ids` chain, re-proven end to end) | Preserve `CoveredTaskIDs`, `findDispatchTasks`, and whole-dispatch reconciliation; add manifest-validated per-task receipts for partial outcomes and map them into task-specific claims. [VERIFIED: codebase grep] |
| JOBS-03 | Without a proposal, tasks sharing files or a dependency chain cluster into one job beyond consecutive same-caste steps — the CalVault field failure (six file-copy batches) as a fixture yields one worker | Replace post-wave consecutive coalescing with a task-level coherent-component pass over dependency and meaningful shared-path edges, then derive job waves. [VERIFIED: codebase grep] |
| JOBS-04 | Grouped jobs work in worktree mode (coalescing is no longer disabled there) | Group before worktree ownership preflight; carry the union of declared paths into one dispatch/worktree; reconcile and merge once per grouped job. [VERIFIED: codebase grep] |
</phase_requirements>

## Summary

The current runtime already contains the seed of coherent jobs: `CoveredTaskIDs`, merged task instructions, unioned declared paths, all-covered-task brief rendering, and completion credit for a wholly successful merged dispatch. The gap is where grouping occurs. `coalesceSequentialDispatches` runs after task waves are built, merges only adjacent single-task waves in the same caste, requires a direct dependency, and is explicitly skipped in worktree mode. It therefore cannot group same-wave tasks sharing meaningful files, non-consecutive dependency components, or any worktree build. [VERIFIED: codebase grep]

Plan one canonical Go-owned job planner before dispatch-wave construction. It should validate the complete task graph fail-closed, accept or refuse Queen proposals independently, form automatic same-caste components from dependency and meaningful-path edges, split oversized automatic components by a measured prompt allowance, and finally build a job DAG. Wrappers propose and render; the Go runtime remains authoritative for the accepted groups, reasons, dependency order, manifests, credit, and state mutations. [VERIFIED: codebase grep]

Partial completion is the highest-risk part. Today a successful dispatch credits every `CoveredTaskID`, while a failed direct build rolls colony state back and the external finalizer only commits a fully successful packet. A task-level receipt must therefore be added to both native and external completion paths, validated against the manifest, projected into task-specific claims, and persisted before any task is credited. Worktree partial credit must occur only after the receipt's accepted files are present in the root checkout; an orphaned worktree alone is not completion evidence. [VERIFIED: codebase grep]

**Primary recommendation:** Implement a pure `planCoherentJobs` layer first, then receipts/retry reconciliation, then worktree and check-in integration; do not broaden `coalesceSequentialDispatches` in place because its post-wave position cannot satisfy JOBS-03 or JOBS-04. [VERIFIED: codebase grep]

For a newcomer: tasks are the individual checklist items; a coherent job is the folder handed to one worker. Phase 195 needs the runtime to pack related checklist items into that folder, prove the order is safe, and still tick only the items the worker actually proved complete.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Queen job proposal | Wrapper/orchestration | Go CLI input parser | The wrapper may propose task IDs, owner, and reason, but Go validates and records the authoritative decision. [VERIFIED: codebase grep] |
| Whole-plan dependency validation | Go runtime/domain | Colony state | Dispatch must fail before worker launch on a missing dependency or named cycle. [VERIFIED: codebase grep] |
| Default coherent grouping | Go runtime/domain | Manifest persistence | A deterministic pure planner turns plan tasks into jobs before waves or worktrees are derived. [VERIFIED: codebase grep] |
| Worker brief composition | Go runtime/presentation | Generated brief files | Existing brief rendering resolves all tasks through `CoveredTaskIDs`; job reason and ownership context join the same brief. [VERIFIED: codebase grep] |
| Full and partial task credit | Go runtime/evidence boundary | Colony state and attempt journal | Only runtime-validated success or task receipts may change task status. [VERIFIED: codebase grep] |
| Worktree execution | Git/worktree adapter | Go job planner | One grouped dispatch gets one worktree and the union of covered paths; merge-back gates credit. [VERIFIED: codebase grep] |
| One-worker check-in decision | Go runtime/policy | Wrapper rendering | The runtime already emits `checkin_requested`; it must also own the automatic fast-path reason and compact summary. [VERIFIED: codebase grep] |

## Project Constraints (from AGENTS.md)

- The Go CLI runtime is the Codex source of lifecycle truth; wrapper prose cannot become the authority for accepted jobs, task credit, or state changes. [VERIFIED: codebase grep]
- Claude Code and OpenCode are primary maintained surfaces; Codex is best-effort, but native CLI lifecycle, installation/update behavior, state integrity, and worker dispatch must remain safe and usable. [VERIFIED: codebase grep]
- Intelligent build behavior must keep `.aether/commands/build.yaml`, the Claude/OpenCode wrappers, `.aether/skills/colony/aether-colony-build-cycle/SKILL.md`, and `cmd/command_guide.go` aligned. [VERIFIED: codebase grep]
- `.claude/commands/ant/build.md`, `.claude/commands/ant-build.md`, and `.opencode/commands/ant/build.md` are parity copies; the current files are byte-identical and tests enforce their shared contract. [VERIFIED: codebase grep]
- Runtime or documentation changes must preserve plain-English explanations that state what changed, why it matters, and what the user should do. [VERIFIED: codebase grep]
- Required Go verification is `go test ./...`, `go test ./... -race`, `go build ./cmd/aether`, and `go vet ./...`. [VERIFIED: codebase grep]
- `.aether/data/` and `.aether/dreams/` are local-only runtime state; do not publish or treat their current contents as canonical source. [VERIFIED: codebase grep]
- Existing unrelated dirty working-tree changes are user-owned and must not be overwritten or reverted. [VERIFIED: git status]

## Standard Stack

### Core

| Component | Version/Location | Purpose | Why Standard |
|-----------|------------------|---------|--------------|
| Go | 1.26.5 | Runtime types, deterministic planning, validation, persistence, and tests | The module and installed toolchain both use Go 1.26.5; no new language or service is needed. [VERIFIED: go version] |
| `pkg/colony` | in-repo | Plan/task state and named cycle/missing-dependency validation | `DetectCycles` already returns typed `MissingDepError` and a named `CycleError.Tasks` path. [VERIFIED: codebase grep] |
| `cmd` build pipeline | in-repo | Proposal parsing, job planning, manifests, execution, finalization, attempts, and check-in policy | These are the current authoritative lifecycle paths. [VERIFIED: codebase grep] |
| `pkg/codex` | in-repo | Worker dispatch/result/handoff contracts and worktree worker sessions | Existing worker contracts should be extended additively for receipts. [VERIFIED: codebase grep] |
| Cobra flag API | existing transitive project dependency | Repeatable `--job-proposal`, `--checkin`, conflict validation | Build command flags already use Cobra `StringArray` and `Bool`. [VERIFIED: codebase grep] |

### Supporting

| Component | Location | Purpose | When to Use |
|-----------|----------|---------|-------------|
| `CoveredTaskIDs` / `dispatchCoveredTaskIDs` | `cmd/codex_build.go` | Authoritative task membership for a job | Every grouped dispatch and brief/credit lookup. [VERIFIED: codebase grep] |
| `declaredPathsForTask` | `cmd/codex_build_worktree.go` | Existing pre-dispatch task path evidence | Shared-file edges and grouped worktree path union. [VERIFIED: codebase grep] |
| Build attempt journal | `cmd/build_attempt.go` | Append-only dispatch/result history | Persist original grouped outcome and link unfinished-only retry attempts. [VERIFIED: codebase grep] |
| Completion contract generator | `cmd/contract_schema.go`, `.aether/schemas/completion-packet.schema.json` | Keep wrapper/native result JSON and schema synchronized | Any task-receipt contract change. [VERIFIED: codebase grep] |
| Existing brief allowance | `briefTaskContentAllowanceChars = 6000` | Measurable soft job-size signal | Project automatic job content before dispatch; do not introduce a task-count cap. [VERIFIED: codebase grep] |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Pre-wave task-to-job planner | Expand `coalesceSequentialDispatches` | Rejected: post-wave dispatches have already separated same-wave shared-file tasks and worktree ownership has the wrong unit. [VERIFIED: codebase grep] |
| `pkg/colony.DetectCycles` preflight | Use `pkg/agent.TaskGraph` unchanged | The agent graph's current error is generic and task-wave fallback is fail-open; the colony validator already identifies the task and missing dependency or the named cycle. [VERIFIED: codebase grep] |
| Structured relationship/benefit fields | One free-form `reason` string | A single opaque sentence requires fragile natural-language quality checking and cannot structurally enforce D-04. [VERIFIED: codebase grep] |
| Additive receipt fields | New completion protocol/package | Extending current worker results and generated schema preserves existing wrapper and direct lanes. [VERIFIED: codebase grep] |

**Installation:** No external package installation is required. [VERIFIED: codebase grep]

## Architecture Patterns

### System Architecture Diagram

```text
Plan tasks + optional Queen proposals + selected tasks
                    |
                    v
       whole-plan dependency preflight
        | missing/cycle -> named hard refusal
        v
       canonical Go coherent-job planner
        |-- validate proposals -> accept safe groups
        |                     -> refuse/repair affected group only
        |-- cluster remaining tasks by dependency/shared meaningful path
        |-- split automatic oversized components by projected brief size
        v
       job DAG -> execution waves -> dispatch manifest
                    |                 |
                    |                 +-> runtime check-in decision/summary
                    v
           one worker per coherent job
             |                  |
             | in-repo          | worktree: one checkout, unioned paths
             v                  v
        worker result + optional per-task completion receipts
                    |
                    v
       manifest/path/verification structural admission
        | partial -> candidate claims + sync paths, no credit
                    |
                    v
       root-evidence finalization after accepted paths are in root
        | full success -> credit all covered task IDs
        | partial      -> credit finalized CompletedTaskIDs only
        | no evidence  -> credit none
                    |
                    v
       accepted root files + task state + append-only attempt journal
                    |
                    v
       unfinished-only dependency-safe recovery job linked to prior attempt
```

The decision point is before waves: this lets one dependency component span former waves and lets same-wave shared-path tasks become one worktree owner before overlap validation. [VERIFIED: codebase grep]

### Recommended Project Structure

```text
cmd/
├── coherent_jobs.go                 # pure proposal/default grouping and job-DAG planning
├── coherent_jobs_test.go            # graph, reasons, limits, CalVault, repair fixtures
├── coherent_job_receipts.go         # shared native/external receipt validation and credit set
├── coherent_job_receipts_test.go    # partial evidence and retry fixtures
├── codex_build.go                   # call planner; dispatch/state integration
├── codex_build_worktree.go          # grouped ownership and accepted partial sync
├── codex_build_finalize.go          # external receipt projection/finalization
├── build_attempt.go                 # append-only retry linkage
├── ceremony_team_checkin.go         # runtime fast-path policy and compact summary
├── codex_workflow_cmds.go           # additive proposal/check-in flags
└── contract_schema.go               # generated completion contract
pkg/codex/
├── worker.go                        # additive TaskReceipts on native result
├── handoff.go                       # reuse verification evidence vocabulary
└── dispatch.go                      # grouped job metadata if shared with worktree executor
.aether/schemas/
└── completion-packet.schema.json    # regenerated, never hand-edited
```

Keep the grouping algorithm and two-stage receipt boundary in small pure files even if the serialized structs remain near existing manifest/result structs. This avoids adding more policy to the already broad `codex_build.go` and allows direct table tests. [VERIFIED: codebase grep]

### Pattern 1: Validate Globally, Repair Locally

**What:** Run `colony.DetectCycles` across the current plan before constructing any job. Then validate each explicit proposal independently against unique task membership, dependency order, owner, reason fields, and selected-task scope. A bad proposal produces a named refusal and its tasks return to the automatic planner; unrelated accepted proposals remain intact. [VERIFIED: codebase grep]

**When to use:** Every direct and plan-only build, before a manifest, worktree, worker brief, or attempt dispatch is created. [VERIFIED: codebase grep]

**Required refusal payload:** `proposal name`, `offending task`, `unmet dependency`, `repair action`, and the runtime-generated safe replacement jobs. A real plan cycle bypasses local repair and blocks the phase with the named cycle. [VERIFIED: codebase grep]

### Pattern 2: Coherent Components Before Waves

**What:** Build one task-level graph with directed dependency edges and undirected meaningful-path edges. Automatic edges only join tasks with the same resolved caste. Candidate jobs are connected components; tasks inside each component receive a deterministic topological order, with original plan order as the tie-breaker for independent shared-path tasks. Split an oversized automatic component only at dependency-safe boundaries. [VERIFIED: codebase grep]

**When to use:** After accepted explicit proposals have claimed their tasks and before the job DAG/execution waves are derived. [VERIFIED: codebase grep]

**Path policy:** Reuse normalized, exact, repository-relative paths from `declaredPathsForTask`. Ignore overlap that consists only of housekeeping basenames such as README variants, changelog/history/release-note files, and ecosystem dependency manifests or lockfiles (`go.mod`, `go.sum`, `package.json`, lockfiles, `pyproject.toml`, `Cargo.toml`, `Cargo.lock`, `Gemfile`, `Gemfile.lock`). A component is joined only when at least one shared path is not incidental. Add the policy as a named pure function and table-test every included basename. [VERIFIED: codebase grep]

**Soft limit:** Project the task-relevant brief characters using the same content classes the worker brief renders (goal, constraints, hints, criteria, and evidence requirements) and compare against the existing named 6,000-character allowance. Do not count tasks. Do not split a single oversized task. Preserve an explicitly proposed oversized group only when its structured relationship/benefit explains end-to-end ownership. [VERIFIED: codebase grep]

### Pattern 3: Additive Queen Proposal Contract

Use a repeatable structured flag rather than parallel comma-separated arrays:

```text
--job-proposal '{
  "name":"calvault-file-copy",
  "task_ids":["T-195-02","T-195-03","T-195-04","T-195-05","T-195-06"],
  "owner_caste":"builder",
  "relationship":"the tasks modify the same templates",
  "benefit":"one Builder avoids repeated setup and write conflicts",
  "owner_reason":""
}'
```

Recommended additive manifest fields are `job_proposals`, `job_decisions`, and dispatch-level `job_name`, `job_reason`, `job_source` (`queen`, `automatic`, or `single`). Keep existing `task_id` and `covered_task_ids` unchanged for compatibility. Require `owner_reason` when an explicit proposal crosses the tasks' automatically resolved caste boundary, and normalize the owner through the existing Queen caste-resolution machinery. [VERIFIED: codebase grep]

The wrapper should make its proposal after reading the first runtime plan, then re-fetch plan-only with the proposal exactly as Phase 194 re-fetches with caste reasons. The runtime returns accepted/refused/repaired decisions and the wrapper renders them; the wrapper must never mutate manifest dispatches itself. [VERIFIED: codebase grep]

### Pattern 4: Receipt-Driven Partial Credit

Add optional `task_receipts` to native and external worker results. A receipt should include `task_id`, successful status (`completed` or `completed_no_change`), summary, task-specific created/modified/test paths, and the existing handoff verification fields. The manifest remains authoritative for that task's criteria and evidence requirements; the worker does not redefine them. [VERIFIED: codebase grep]

Use one shared two-stage contract in native, external, and worktree lanes:

1. `admitCoherentJobTaskReceipts` performs structural admission only. It checks
   that each task ID is unique and belongs to `dispatchCoveredTaskIDs`, status is
   successful, summary is non-empty, paths normalize within the repository and
   are a subset of aggregate worker claims, handoff verification reports `pass`
   with concrete commands/evidence, and claims bind to the manifest task's own
   criteria/evidence requirements. It returns candidate task claims plus the
   normalized sync-path set. It does not read root artifacts and cannot return
   `CompletedTaskIDs`.
2. `finalizeCoherentJobTaskReceiptEvidence` consumes only admitted candidates
   after their accepted paths are present in the root checkout. It computes
   artifact evidence with `attachBuildArtifactEvidence`, drops any candidate
   whose root evidence fails, and only then returns task-specific claims and
   runtime-owned `CompletedTaskIDs`. Worker-provided hashes are never
   authoritative.

The native/in-repo and external lanes call the two stages back-to-back because
their accepted files are already in root. Worktree execution calls admission in
the worker-result boundary, syncs only its candidate paths to root, then calls
finalization. This shared ordering avoids validating against an orphan checkout
or crediting before root evidence exists. [VERIFIED: codebase grep]

For a wholly successful grouped dispatch, absence of receipts may remain backward-compatible and credit every covered task because the manifest and whole-dispatch success already establish the existing contract. For any failed/interrupted/timeout grouped dispatch, only valid explicit receipts may populate a runtime-owned `completed_task_ids` credit set; `covered_task_ids` must continue to mean assignment, not completion. [VERIFIED: codebase grep]

### Pattern 5: Append-Only Unfinished Retry

Persist the failed grouped result and accepted receipt credits before returning a recoverable error. Create a new attempt containing only uncovered/uncredited task IDs, re-run dependency validation with previously credited tasks treated as satisfied, and add an optional parent attempt/job reference to the new attempt record. Never mutate the first attempt's dispatch, receipts, or completion digest. [VERIFIED: codebase grep]

The direct lane currently records a failed attempt and then restores the original colony state; the external lane sets `BUILT` only after a fully accepted packet. The plan must therefore introduce an explicit partial-terminal reconciliation transition rather than attempting to squeeze partial outcomes through the current full-success path. [VERIFIED: codebase grep]

### Pattern 6: Worktree Credit Follows Accepted Sync

Full success is straightforward: remove the worktree-mode grouping exclusion, convert one grouped dispatch into one `pkg/codex.WorkerDispatch`, union its declared paths, allocate one worktree/session, and merge it once. Grouping must occur before `validateDeclaredWorktreeOwnership`, so tasks that intentionally share a file are one owner rather than a false same-wave conflict. [VERIFIED: codebase grep]

For partial completion, run structural admission first to obtain candidate sync
paths without any credit. Sync only those admitted paths to the root checkout,
reject touched paths not attributable to an admitted receipt, then run shared
root-evidence finalization and commit only the resulting `CompletedTaskIDs`.
Preserve uncredited edits in the orphan/recovery path. This ordering keeps
worktree evidence equal to in-repo evidence without a circular dependency
between validation, sync, and credit. [VERIFIED: codebase grep]

### Pattern 7: Runtime-Owned One-Worker Fast Path

Add `--checkin` beside `--no-checkin` and reject their combination before plan-only creates or binds an attempt. A pure runtime policy should return `checkin_requested`, a machine-readable reason, and a compact summary after final jobs and pending decisions are known. [VERIFIED: codebase grep]

Recommended precedence:

1. Autopilot/no-checkin stays non-interactive.
2. `--checkin` forces a pause.
3. Any live owner decision, including a forced-reviewer waiver, forces a pause.
4. Exactly one implementation dispatch and no additional build worker takes the fast path.
5. All other builds pause.

The compact summary should reuse the deterministic worker identity, covered-task rendering, and accepted `job_reason`; it must state the worker, every covered task, relationship/benefit, and “no owner decision is pending, so dispatch continues.” [VERIFIED: codebase grep]

### Anti-Patterns to Avoid

- **Post-wave-only merging:** It cannot see independent same-wave shared-file work and preserves the worktree false-conflict boundary. [VERIFIED: codebase grep]
- **Fail-open cycle fallback:** Current `taskWaves` emits all remaining tasks when no task is ready; never let that behavior reach Phase 195 planning. [VERIFIED: codebase grep]
- **Silent proposal reordering:** D-05/D-06 require refusal plus visible repair, not making the Queen's unsafe order appear accepted. [VERIFIED: codebase grep]
- **Reason prose heuristics:** Separate relationship and benefit fields provide a stable structural check; avoid judging whether a single sentence “sounds specific.” [VERIFIED: codebase grep]
- **Using `covered_task_ids` as a partial success list:** It currently defines what the job owns; overloading it would make assignment and evidence indistinguishable. [VERIFIED: codebase grep]
- **Crediting touched files:** A file change is shared evidence, not proof that each task's criteria passed. [VERIFIED: codebase grep]
- **Crediting an unmerged worktree:** Root-checkout state and later deterministic verification would disagree with recorded completion. [VERIFIED: codebase grep]
- **Wrapper-authored manifest groups:** It bypasses runtime dependency, reason, ownership, and attempt validation. [VERIFIED: codebase grep]
- **A hard task-count cap:** It violates D-03 and poorly approximates actual worker context size. [VERIFIED: codebase grep]
- **Mutating the failed attempt into a retry:** Attempt history and completion receipts are deliberately append-only. [VERIFIED: codebase grep]

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Cycle/missing dependency detection | A second ad-hoc DFS inside grouping | `pkg/colony.DetectCycles` and its typed errors | It already names missing dependencies and cycles and is covered by tests. [VERIFIED: codebase grep] |
| Caste name normalization/reason admission | Job-specific string aliases | Existing `resolveCasteName` and Queen judgement paths | Keeps job owner semantics aligned with Phase 194. [VERIFIED: codebase grep] |
| Path normalization/root containment | Raw string equality or prefix checks | Existing declared-path and completion-claim normalization | Worktree and completion paths already share repository-boundary protections. [VERIFIED: codebase grep] |
| Brief aggregation | A second job template | `findDispatchTasks`, `dispatchCoveredTaskIDs`, and current task-section renderer | Existing regression tests prove every merged task's constraints and criteria appear. [VERIFIED: codebase grep] |
| Artifact digests | Worker-supplied checksums | `attachBuildArtifactEvidence` over root files | Runtime-computed hashes preserve the trust boundary. [VERIFIED: codebase grep] |
| Completion schema | A manually maintained JSON copy | `aether contract-schema --write/--check` | The schema is generated from Go contract structs and drift-tested. [VERIFIED: codebase grep] |
| Retry persistence | Overwrite the last result | Existing attempt journal plus an additive parent link | The journal already records status transitions and immutable result evidence. [VERIFIED: codebase grep] |
| Worktree allocation/merge | Custom git shell orchestration | Current `pkg/codex` worktree sessions and merge-back pipeline | Existing lifecycle handles sessions, orphan detection, conflicts, and merge reporting. [VERIFIED: codebase grep] |

**Key insight:** The hard problems are authority and evidence, not graph traversal. Reuse the runtime's manifest, claim, worktree, and attempt boundaries so one worker can own more work without making completion less trustworthy. [VERIFIED: codebase grep]

## Common Pitfalls

### Pitfall 1: Grouping After Task Waves

**What goes wrong:** CalVault-like shared-file tasks in one wave remain separate, and worktree ownership rejects their overlap. [VERIFIED: codebase grep]

**Why it happens:** The current coalescer accepts dispatches, not plan tasks, and only merges adjacent single-dispatch waves joined by a direct dependency. [VERIFIED: codebase grep]

**How to avoid:** Produce jobs from task-level components first, then derive job waves and worktree dispatches.

**Warning signs:** Tests pass for a linear two-task chain but fail for six independent batches over the same source list or for non-consecutive dependency members.

### Pitfall 2: Existing Cycle Handling Is Split Across Two Implementations

**What goes wrong:** Build-time `taskWaves` can place cyclic remaining tasks into a wave instead of refusing dispatch. [VERIFIED: codebase grep]

**Why it happens:** Plan acceptance has `pkg/colony.DetectCycles`, but build-wave construction does not call it and has a fallback for “no ready tasks.” [VERIFIED: codebase grep]

**How to avoid:** Make whole-plan `DetectCycles` a mandatory build preflight and test that no manifest/attempt/worker is created after the error.

**Warning signs:** A cycle test asserts only an error string but does not assert zero dispatch side effects.

### Pitfall 3: Successful Dispatch Credit Is Too Coarse for Partial Results

**What goes wrong:** Reusing current `completedBuildTaskIDs` on a failed group either credits nothing or, if status is forced successful, credits every covered task. [VERIFIED: codebase grep]

**Why it happens:** Current reconciliation uses dispatch terminal status and then adds all `CoveredTaskIDs`; there is no per-task success set. [VERIFIED: codebase grep]

**How to avoid:** Add a runtime-owned validated credit set and use it only for partial terminal outcomes; preserve the all-covered behavior for whole success.

**Warning signs:** A four-of-six fixture passes without checking the exact two unfinished task statuses.

### Pitfall 4: Aggregate Claims Lose Task Ownership

**What goes wrong:** Files can be bound only to a grouped dispatch's primary `TaskID`, even though later tasks are marked complete. [VERIFIED: codebase grep]

**Why it happens:** Aggregate native/external claim builders use the dispatch's primary task unless explicit task claims exist. [VERIFIED: codebase grep]

**How to avoid:** Admit every receipt against its task's manifest requirements, then project only root-finalized candidates into `codexBuildClaims.TaskClaims` under their own task IDs.

**Warning signs:** Phase status advances but `last-build-claims.json` lacks entries for covered tasks two through six.

### Pitfall 5: Worktree Partial Credit Before Merge

**What goes wrong:** Colony state says a task completed although root files and later checks cannot see its changes. [VERIFIED: codebase grep]

**Why it happens:** Failed workers are not part of the normal completed-branch merge path, so their files remain in an orphaned checkout. [VERIFIED: codebase grep]

**How to avoid:** Make admitted candidate-path synchronization a prerequisite for root-evidence finalization and partial credit.

**Warning signs:** A receipt test inspects only worktree files, or a retry creates the same output again in root.

### Pitfall 6: Proposal Repair Accidentally Destroys Safe Grouping

**What goes wrong:** One unsafe Queen group causes every task to revert to one worker per task. [VERIFIED: codebase grep]

**Why it happens:** Validation returns one global error or fallback restarts from raw dispatches.

**How to avoid:** Reserve global failure for real graph corruption/cycles; keep per-proposal decisions and send only refused members through the automatic planner.

**Warning signs:** A fixture with two safe proposals and one unsafe proposal changes all three groups.

### Pitfall 7: Check-in Fast Path Hides a Real Decision

**What goes wrong:** One Builder bypasses the owner's named-risk reviewer waiver or another pending decision. [VERIFIED: codebase grep]

**Why it happens:** A naive `len(dispatches) == 1` check ignores Phase 194 boundary guidance and forced-reviewer records.

**How to avoid:** Decide after final job/caste policy and pending-decision state are known; test both forced-reviewer and generic owner-decision cases.

**Warning signs:** The fast-path test has only a single worker fixture and no decision-bearing counterexample.

### Pitfall 8: Flag Conflict Mutates State

**What goes wrong:** `--checkin --no-checkin` creates an attempt or manifest before returning an error. [VERIFIED: codebase grep]

**Why it happens:** The current `checkin_requested` value is attached after plan-only execution rather than validated before it. [VERIFIED: codebase grep]

**How to avoid:** Reject the conflict in command parsing before calling either plan-only or direct build execution; assert no state/artifact changes.

## Code Examples

Verified patterns and implementation-shaped recommendations:

### Whole-plan dependency preflight

```go
// Source: pkg/colony/cycle.go [VERIFIED: codebase grep]
if err := colony.DetectCycles(state.Plan.Phases); err != nil {
	var cycleErr *colony.CycleError
	var missingErr *colony.MissingDepError
	switch {
	case errors.As(err, &cycleErr):
		return nil, fmt.Errorf(
			"cannot dispatch phase %d: dependency cycle %s; repair depends_on in the plan",
			phase.ID, strings.Join(cycleErr.Tasks, " -> "))
	case errors.As(err, &missingErr):
		return nil, fmt.Errorf(
			"cannot dispatch task %s: dependency %s does not exist; repair depends_on in the plan",
			missingErr.Task, missingErr.MissingDep)
	default:
		return nil, err
	}
}
```

### Deterministic meaningful-path edge

```go
// Recommended; sources: cmd/codex_build_worktree.go and D-01 [VERIFIED: codebase grep]
func tasksShareMeaningfulPath(left, right colony.Task) bool {
	rightPaths := stringSet(declaredPathsForTask(right))
	for _, path := range declaredPathsForTask(left) {
		if _, shared := rightPaths[path]; shared && !isIncidentalGroupingPath(path) {
			return true
		}
	}
	return false
}
```

### Partial credit selection

```go
// Recommended; source behavior: cmd/codex_build.go completedBuildTaskIDs
// [VERIFIED: codebase grep]
func creditedTaskIDs(dispatch codexBuildDispatch) []string {
	if dispatch.Status == "completed" || isNoChangeExternalBuildStatus(dispatch.Status) {
		return dispatchCoveredTaskIDs(dispatch)
	}
	// CompletedTaskIDs is populated only by post-root-evidence finalization.
	return uniqueSortedStrings(dispatch.CompletedTaskIDs)
}
```

### Check-in policy

```go
// Recommended; source behavior: cmd/codex_workflow_cmds.go and Phase 194 waiver state
// [VERIFIED: codebase grep]
func decideBuildCheckin(in checkinInput) checkinDecision {
	if in.Checkin && in.NoCheckin {
		return checkinDecision{Err: errors.New("--checkin conflicts with --no-checkin")}
	}
	if in.NoCheckin || in.Autopilot {
		return checkinDecision{Requested: false, Reason: "non-interactive mode"}
	}
	if in.Checkin || in.PendingOwnerDecision {
		return checkinDecision{Requested: true, Reason: "owner decision is pending or pause was forced"}
	}
	if len(in.ImplementationDispatches) == 1 && in.TotalBuildDispatches == 1 {
		return checkinDecision{Requested: false, Reason: "one worker and no owner decision is pending"}
	}
	return checkinDecision{Requested: true, Reason: "multiple workers require team confirmation"}
}
```

## Prescriptive Plan Ordering

1. **Wave 0 — lock graph/job contracts:** Add pure whole-plan preflight, proposal types/parser/decision records, coherent component planning, incidental-path policy, soft-limit projection, and job-DAG tests. Reproduce six CalVault batches as one automatic job and keep unrelated work parallel. [VERIFIED: codebase grep]
2. **Wave 1 — integrate manifests and briefs:** Replace post-wave-only grouping with the canonical planner in both direct and plan-only paths; add backward-compatible job fields; preserve every covered task in briefs; update deterministic names to seed grouped jobs from phase, owner, and ordered covered IDs while retaining single-task stability. [VERIFIED: codebase grep]
3. **Wave 2 — build the receipt trust boundary:** Extend native/external result structs and generated schema, implement shared structural admission plus root-evidence finalization, task-specific claims, exact partial credit, and append-only parent-linked retry. Cover full success, four-of-six failure, no-receipt failure, duplicate/unknown/out-of-scope/path-laundering receipts, and no-change. [VERIFIED: codebase grep]
4. **Wave 3 — enable worktree jobs:** Remove the grouping exclusion, ensure preflight sees grouped ownership, union declared paths, test one worktree for six tasks, and add accepted partial-path sync before credit. Preserve unrelated overlap conflicts and orphan recovery. [VERIFIED: codebase grep]
5. **Wave 4 — one-worker check-in policy:** Add `--checkin` and early conflict validation; derive the policy after final jobs and pending decisions; render the compact summary; invert the old one-worker pause test and keep waiver/autopilot/multi-worker counterexamples. [VERIFIED: codebase grep]
6. **Wave 5 — surface parity and end-to-end gates:** Update command catalog, generated schema, YAML source, three byte-identical wrappers, Codex build-cycle skill, command guide, `CLAUDE.md`/relevant docs, then run focused, full, race, vet, build, contract, and wrapper-parity checks. [VERIFIED: codebase grep]

This order prevents worktree and check-in code from depending on unstable job semantics, and prevents partial state mutations from being designed independently in the direct and external lanes. [VERIFIED: codebase grep]

## State of the Art

| Current Approach | Phase 195 Approach | Impact |
|------------------|--------------------|--------|
| Merge adjacent single-task waves only when same caste and directly dependent | Form task-level coherent components from full dependency chains and meaningful shared paths before waves | CalVault and non-consecutive relationships become one job. [VERIFIED: codebase grep] |
| Skip coalescing in worktree mode | Group first, then allocate one worktree to the unioned job paths | Same-file coherent work no longer creates multiple owners. [VERIFIED: codebase grep] |
| Whole successful dispatch credits every covered task; failed dispatch credits none | Keep whole-success compatibility and add receipt-validated partial credit | Four-of-six can be honest without treating the group as all-or-nothing. [VERIFIED: codebase grep] |
| Wrapper carries reasons per caste, not per job | Structured proposal carries group relationship, benefit, owner, and cross-caste owner reason | Multiple same-caste jobs retain distinct rationales. [VERIFIED: codebase grep] |
| One worker still pauses unless `--no-checkin` | Decision-free one-worker build automatically renders a compact summary and continues; `--checkin` forces pause | Removes a redundant owner interaction without hiding decisions. [VERIFIED: codebase grep] |

**Deprecated/outdated after this phase:**

- Direct use of `coalesceSequentialDispatches` as the authoritative grouping engine; retain only as a compatibility shim or remove after all callers use coherent jobs. [VERIFIED: codebase grep]
- Wrapper instruction “if `checkin_requested` is false (`--no-checkin` was passed)” because false will also represent the automatic one-worker fast path. [VERIFIED: codebase grep]
- Documentation stating every build pauses before spawning; that becomes conditional on worker count and pending decisions. [VERIFIED: codebase grep]

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| — | None. Recommendations are derived from locked context decisions and inspected runtime/tests; no package, compliance, retention, or external-service assumption is required. | — | — |

## Open Questions (RESOLVED)

1. **How should partial worktree files be transferred without accepting unrelated edits?**
   - What we know: normal worktree merge-back operates on completed branches, failed worktrees are preserved, and D-08/D-09 require partial credit when trustworthy proof exists. [VERIFIED: codebase grep]
   - What's unclear: the current runtime has no receipt-scoped partial merge primitive. [VERIFIED: codebase grep]
   - **Resolution:** Use the shared two-stage receipt contract. Plan 195-04
     defines structural admission (candidate task claims and normalized sync
     paths, no credit) plus root-evidence finalization (the only stage allowed
     to emit `CompletedTaskIDs`); Plan 195-06 reuses both stages for the external
     lane. Plan 195-08 owns the worktree wiring and shared receipt module while
     it runs admission, syncs only admitted paths to root, then finalizes root
     evidence before credit. Unattributed edits remain in the orphaned checkout.

2. **What counts as a pending owner decision beyond the named-risk waiver?**
   - What we know: Phase 194 already emits forced-reviewer waiver opportunities and orchestrator-boundary guidance, while D-13 deliberately includes “other owner decision.” [VERIFIED: codebase grep]
   - What's unclear: those signals currently live in more than one result/state structure rather than one `PendingOwnerDecision` field. [VERIFIED: codebase grep]
   - **Resolution:** Plan 195-05 centralizes
     `buildHasPendingOwnerDecision` as a pure predicate over live
     phase/attempt-scoped forced-reviewer waiver records, unanswered
     orchestrator boundary questions, and any other persisted unanswered
     owner-decision record already consumed by the build boundary. Each source
     gets an independent one-worker counterexample; no rendered prose is used
     as policy input.

Both questions are resolved by named implementation plans and executable tests;
neither remains open for executor interpretation.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|-------------|-----------|---------|----------|
| Go | Runtime, schema generator, tests | ✓ | 1.26.5 darwin/arm64 | None needed. [VERIFIED: go version] |
| Git | Worktree allocation/reconciliation tests | ✓ | 2.55.0 | In-repo unit fixtures for non-worktree logic only; JOBS-04 still requires Git. [VERIFIED: git version] |
| Node.js | Existing project/tooling helpers where invoked | ✓ | v26.7.0 | Not required by the core Go implementation. [VERIFIED: node version] |
| `gsd-sdk` | Planning workflow metadata/commit | ✓ | 1.42.3 | Manual scoped git commit. [VERIFIED: gsd-sdk version] |

**Missing dependencies with no fallback:** None. [VERIFIED: environment probe]

**Missing dependencies with fallback:** None. [VERIFIED: environment probe]

## Verification Strategy

`.planning/config.json` explicitly sets `workflow.nyquist_validation` to `false`, so the formal generated `Validation Architecture` section is intentionally omitted. Phase 195 still needs the following test-first verification matrix because its requirements change state and evidence trust boundaries. [VERIFIED: codebase grep]

### Existing Infrastructure

| Property | Value |
|----------|-------|
| Framework | Go standard `testing`, Go 1.26.5 [VERIFIED: codebase grep] |
| Primary package configs | `go.mod`; no separate test config required [VERIFIED: codebase grep] |
| Focused quick command | `go test ./cmd ./pkg/colony ./pkg/codex -run 'Test(Coherent|Job|CalVault|Receipt|Checkin|DetectCycles|Worktree)' -count=1` |
| Full suite | `go test ./...` |
| Race suite | `go test ./... -race` |
| Static/build gates | `go vet ./... && go build ./cmd/aether` |
| Contract drift | `go run ./cmd/aether contract-schema --check` |

The pre-research focused baseline passed 16 selected existing tests across `cmd`, `pkg/colony`, and `pkg/agent`, including current coalescing, covered-task brief/credit, wrapper completion, worktree overlap, one-worker pause, and cycle fixtures. The completion schema drift check also passed. [VERIFIED: go test]

### Requirement-to-Test Map

| Requirement | Required Behavior | Test Layer | Concrete Fixtures/Assertions |
|-------------|-------------------|------------|------------------------------|
| JOBS-01 | Safe proposal accepted; generic/missing reason refused; cross-caste owner reason required; bad internal order names task+dependency and repairs only that group; named cycle blocks with zero side effects | Pure unit + command integration | Add `coherent_jobs_test.go`; extend Queen judgement/build plan-only tests; assert final dispatches and decision records, not only message text. [VERIFIED: codebase grep] |
| JOBS-02 | Brief contains every task's constraints/criteria/evidence; whole success credits all; four-of-six credits exactly four; absent/duplicate/unknown/path-laundered receipt credits none; retry contains only two and links parent | Brief + contract + finalizer + state integration | Extend `codex_build_test.go`, `merged_dispatch_task_credit_test.go`, `wrapper_bundled_completion_test.go`, `build_attempt*_test.go`; add native/external receipt parity cases. [VERIFIED: codebase grep] |
| JOBS-03 | Six CalVault batches become one Builder without proposal; same meaningful file joins; incidental-only file does not; dependency component spans non-consecutive plan positions; different castes stay separate automatically; soft limit splits safely | Pure planner + manifest integration | Move/extend `dispatch_coalesce_test.go` with exact six-batch fixture and one-worker assertion; add deterministic order and oversized component tables. [VERIFIED: codebase grep] |
| JOBS-04 | Same six tasks produce one worktree/branch/session and unioned ownership; unrelated same-wave overlap still refuses; partial receipt files reach root before credit; uncredited edits remain recoverable | Worktree integration | Extend `codex_build_worktree_test.go`; assert dispatch/worktree count, merged root files, exact task statuses, conflict preservation, and orphan diagnostics. [VERIFIED: codebase grep] |
| D-11–D-14 | Decision-free one-worker false check-in plus compact facts; `--checkin` true; flag conflict no mutation; forced reviewer/generic pending decision true; two workers true; autopilot unchanged | Pure policy + command/wrapper parity | Replace `TestOneWorkerTeamStillPauses`; extend `ceremony_team_checkin_test.go`, forced-reviewer tests, command catalog, and wrapper contract tests. [VERIFIED: codebase grep] |

### Sampling Rate

- **Per grouping task:** run pure `cmd` coherent-job tests and `pkg/colony` cycle tests.
- **Per receipt/retry task:** run native plus external finalizer/attempt tests together; never validate only one lane.
- **Per worktree task:** run targeted worktree tests with temporary repositories.
- **Per wrapper/docs task:** run wrapper parity and command catalog tests plus contract schema drift.
- **Phase gate:** `go test ./...`, `go test ./... -race`, `go vet ./...`, `go build ./cmd/aether`, and `go run ./cmd/aether contract-schema --check` all green. [VERIFIED: codebase grep]

### Test Gaps the Plan Must Create

- [ ] Pure proposal/default planner fixtures including the six-batch CalVault case, non-consecutive dependencies, incidental-only overlap, deterministic split, safe-proposal preservation, missing dependency, and cycle side-effect checks.
- [ ] Task-receipt contract/validator fixtures shared across direct and external results.
- [ ] Four-of-six partial reconciliation and unfinished-only parent-linked retry fixture.
- [ ] Worktree grouped full-success and receipt-scoped partial-sync fixture.
- [ ] Check-in policy matrix and compact-summary facts.
- [ ] Generated completion schema and command-catalog snapshots after additive flags/fields.

## What Might Have Been Missed

- Deterministic worker naming currently derives from task position/goal; grouped names should incorporate the ordered covered IDs so manifests, attempt diagnostics, and worktree owners do not collide as grouping changes. Preserve the existing seed for single-task dispatches to avoid unnecessary output churn. [VERIFIED: codebase grep]
- `renderCoveredTaskSummary` and `dispatchCoveredTasksNote` describe merged IDs as dependent tasks; shared-file jobs make that wording false. Generalize it to “covered tasks.” [VERIFIED: codebase grep]
- Worktree conflict diagnostics often name only the primary `TaskID`; grouped diagnostics should include job name and covered IDs while retaining the primary ID for compatibility. [VERIFIED: codebase grep]
- The wrapper's current `checkin_requested == false` explanation assumes `--no-checkin`; it must distinguish explicit non-interactive mode from the automatic one-worker policy. [VERIFIED: codebase grep]
- Completion contract changes must propagate through `pkg/codex.WorkerResult`, internal results, external packet types, result collection diagnostics, handoff persistence, schema generation, and both claim aggregators. Updating only the JSON schema or finalizer leaves lane drift. [VERIFIED: codebase grep]
- Selected-task builds require proposals to be wholly inside the selected set, or visibly repaired against only selected unfinished tasks; a proposal must not pull a completed/unselected task back into execution. [VERIFIED: codebase grep]
- Same-file grouping depends on available declared path evidence. When a task has no exact artifact/hint path, group it only through dependency edges or an explicit Queen proposal; do not infer ownership from prose tokens. [VERIFIED: codebase grep]

## Sources

### Primary (HIGH confidence)

- `.planning/phases/195-coherent-jobs/195-CONTEXT.md` — locked decisions, discretion, boundary, and canonical references. [VERIFIED: codebase grep]
- `.planning/REQUIREMENTS.md` and `.planning/ROADMAP.md` — JOBS-01 through JOBS-04 and phase success criteria. [VERIFIED: codebase grep]
- `AGENTS.md` and `/Users/callumcowie/.codex/RTK.md` — project workflow, platform parity, verification, communication, and tool constraints. [VERIFIED: codebase grep]
- `cmd/codex_build.go`, `cmd/codex_build_worktree.go`, `cmd/codex_build_finalize.go`, `cmd/build_attempt.go`, and `cmd/build_worker_run.go` — current dispatch, coalescing, execution, completion, rollback, state credit, worktree, and attempt behavior. [VERIFIED: codebase grep]
- `pkg/colony/cycle.go`, `pkg/agent/task_graph.go`, and `cmd/codex_visuals.go` — dependency validation and current task-wave behavior. [VERIFIED: codebase grep]
- `pkg/codex/worker.go`, `pkg/codex/handoff.go`, `pkg/codex/dispatch.go`, `cmd/contract_schema.go`, and `.aether/schemas/completion-packet.schema.json` — result/evidence contracts. [VERIFIED: codebase grep]
- `cmd/dispatch_coalesce_test.go`, `cmd/merged_dispatch_task_credit_test.go`, `cmd/wrapper_bundled_completion_test.go`, `cmd/codex_build_test.go`, `cmd/codex_build_worktree_test.go`, `cmd/ceremony_team_checkin_test.go`, and `pkg/colony/cycle_test.go` — executable current invariants and regression seams. [VERIFIED: go test]
- `.aether/commands/build.yaml`, `.claude/commands/ant/build.md`, `.claude/commands/ant-build.md`, `.opencode/commands/ant/build.md`, `.aether/skills/colony/aether-colony-build-cycle/SKILL.md`, and `cmd/command_guide.go` — orchestration and parity surfaces. [VERIFIED: codebase grep]

### Secondary (MEDIUM confidence)

- None. This phase changes internal repository behavior; current source and executable tests are more authoritative than external documentation.

### Tertiary (LOW confidence)

- None.

## Metadata

**Confidence breakdown:**

- Standard stack: HIGH — versions and dependencies were probed locally; no new package is recommended. [VERIFIED: environment probe]
- Architecture: HIGH — recommendations follow the inspected direct, external, worktree, manifest, wrapper, and attempt paths. [VERIFIED: codebase grep]
- Pitfalls: HIGH — each is reproduced by a current control flow, schema omission, locked counterexample, or existing regression fixture. [VERIFIED: codebase grep]
- Partial worktree sync detail: MEDIUM — the required trust ordering is locked, but the receipt-scoped sync primitive does not yet exist and must be designed/tested in implementation. [VERIFIED: codebase grep]

**Research date:** 2026-08-27
**Valid until:** 2026-09-26 (30 days; internal Go architecture is stable, but Phase 195 implementation will intentionally invalidate current paths)

**Configuration notes:** Formal Nyquist Validation Architecture and Security Domain sections are omitted because `.planning/config.json` explicitly sets `workflow.nyquist_validation: false` and `security_enforcement: false`. [VERIFIED: codebase grep]
