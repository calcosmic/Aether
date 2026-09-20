# Phase 203: Biological Runtime - Research

**Researched:** 2026-09-12
**Domain:** Recursive worker recruitment, exactly-once result delivery, and outcome-governed pheromone influence, layered on the existing Go spawn-admission kernel
**Confidence:** MEDIUM — the Go-side admission substrate is HIGH confidence (read directly, line-cited); the platform-nesting mechanics are MEDIUM/LOW confidence (web-sourced, dated, and self-contradictory across sources) and MUST be empirically probed by this phase's own code, not assumed from documentation.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Helper Freedom**
- D-01: Recruitment is automatic within program-enforced limits and visible as it happens — no approval prompt by default. The program (BIO-02 admission) is the leash, not an owner gate.
- D-02: Recruited helpers may recruit further (chains) as long as depth/budget/permission rules hold. Chains are the point of the phase; the limits are the safety.
- D-03: A refused recruitment never stalls work: the helper carries on alone, and the refusal is recorded and surfaced. No mid-run pause, no owner question on refusal.

**What the Owner Sees**
- D-04: Live view: one line per recruit joining the run (who, why, cost so far), rendered inline in the working terminal. Hard constraint carried from 202 UAT: the owner never uses a second terminal — all liveness renders inline in the session where the command runs. No second-terminal watch flows.
- D-05: End-of-run summary shows the family tree with costs — who recruited whom, what each branch did and cost, refusals noted.
- D-06: Refusals show inline at the moment they happen AND in the summary.

**Pheromone Influence / Note Power**
- D-07: Suggested pheromones take effect only after owner approval via a quick tick-to-approve queue.
- D-08: When a note changes a decision mid-run: one inline line at the moment it bites, plus a summary list of every decision the owner's notes changed.
- D-09: Outcome-weighted auto-tuning is ON: good outcomes strengthen a note, bad/neutral outcomes weaken or quarantine it, full immutable history kept, owner can pin or revoke any note.
- D-10: Cross-project (imported) notes start quarantined and are released only by the owner, through the same tick-to-approve queue as suggestions.

**Limits**
- D-11: Default chain depth is 2 levels (a helper can recruit, and that recruit can recruit once more). Raisable per-run by explicit flag; default stays modest because speed is the owner's stated priority.
- D-12: Recruits draw from the same team budget the phase already has — no separate recruit allowance, no new dial.

### Claude's Discretion
- Wire shapes, schema versions, ledger/manifest formats for RecruitmentIntent/RecruitmentResult/TrophallaxisPacket (BIO-01, BIO-04, BIO-05).
- Probe design for platform availability (BIO-03): configurable, bounded, outside the repository — engineering choices are Claude's.
- Consolidation of the five existing pheromone write sites into one canonical bus (BIO-07) — pure internals.
- How CEC-07 credit records are stored and joined to outcomes.

### Deferred Ideas (OUT OF SCOPE)
- None raised during discussion.

### Reviewed Todos (not folded)
- Spec builder feature (`.planning/todos/2026-08-20-spec-builder-feature.md`) — unrelated to biological runtime.

### Folded Todos
- `2026-08-01-ts-host-preflight-hardcoded-timeout.md` — BIO-03 requires probes to be configurable, bounded, process-tree-safe, and isolated from the repository; this todo's hardcoded 20s timeout is the counter-example BIO-03 must retire.
- `2026-08-27-worker-turnaround-is-too-slow.md` — folded as a constraint, not a work item: recruitment admission must add no meaningful latency to the plain no-recruitment path.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SYNTH-05 | Mechanism study: reconstruct prior recruitment/delegation/trophallaxis/pheromone behavior — real, simulated, platform-dependent — then reconcile with current Go authorization/ledgers before design. | §Platform Mechanism Study below is the required raw material; a `203-CLASSIC-SYNTHESIS.md` must still be authored from it before plans are approved (SYNTH-07 gate, not satisfied by this document alone). |
| CEC-07 | A recruitment result, pheromone, memory item, or specialist contribution earns credit only when the runtime records the decision it changed and the later verified outcome, including neutral/harmful results. | §Existing Credit-Record Substrate (`AgencyReceiptEvidence`, `SwarmPhase202Limitation`) — the join-to-outcome plumbing exists but is stubbed pending this phase. |
| BIO-01 | Typed `RecruitmentIntent`: authenticated parent/attempt identity, requested caste/capability, bounded objective, reason, evidence, permissions, urgency, scope, cost limits. | §Existing Admission Substrate — `spawnDecisionInput` today carries only requester name/depth/caste/task; BIO-01 is a superset wire format, greenfield. |
| BIO-02 | Atomic admission: depth, tree budget, cycles, duplicates, permissions, paths, cost, parent authority, evidence; atomic manifest amendment before dispatch; unavailable truth denies launch. | `spawnCanSpawnDecision` (depth→budget→ancestor-cycle) is the exact chokepoint pattern to extend with permission/path/cost checks; `PermissionProfileForCaste` and `validateSpendContainedPath` are the substrate for the two missing dimensions. |
| BIO-03 | Honest platform dispatch: native bind or root/host-mediated fallback with exact workspace lease; configurable/bounded/isolated probes; unsupported nesting reported before launch, never faked. | §Platform Mechanism Study — this is the crux finding of the whole phase: no worker caste's agent definition grants the Task/Agent tool, and Claude Code's own nested-subagent capability is independently disputed by an open bug. The existing `platform-dispatcher.ts` subprocess-spawn fallback is the honest path already proven at depth 1. |
| BIO-04 | Durable `RecruitmentResult`: binds child/intent/dispatch/generation/terminal status/evidence/artifacts/handoff/usage exactly once; missing/duplicated/altered/timed-out/replayed drives explicit recovery. | `PauseHandoff.HandoffID` + `LifecycleTransactionReference` + `LifecycleReceiptReference` is the exact idempotency-key/replay pattern already proven for pause/resume; reuse rather than invent. |
| BIO-05 | `TrophallaxisPacket`: scoped child-result carrier to an acknowledged parent continuation or one named follow-on, which records the downstream decision it made from it. | `workerHandoffRecord` (cmd/codex_dispatch_contract.go) is the direct structural precedent — same fields, needs an explicit acknowledgement + decision-join layer BIO-05 adds. |
| BIO-06 | Status/watch/recovery/cost/depth cover the whole governed subtree without weakening immutable build-attempt authorization. | `agent.SpawnTree` already tracks the whole tree (parent/depth/status per entry); Phase 202's typed ceremony bus (`CeremonyTopicBuildSpawn`) already carries spawn events to watch — extend, don't replace. |
| BIO-07 | Canonical pheromone bus: one sanitizer/normalizer/deduplicating writer/provenance model/effective scope-expiry-strength resolver across user/runtime/import/learning paths; cross-project input starts quarantined. | `writePheromoneSignal` (cmd/pheromone_write.go) is ALREADY the single write chokepoint every other file funnels through — the real gap is the resolver (`resolvePheromoneSection`) and a quarantine state, not five competing writers. |
| BIO-08 | Evidence-backed influence: accept/edit/reject/reinforce/defer/expire/revoke/appeal with immutable history; declared decision-domain adapters record influence and outcome-weighted strengthening/weakening/quarantine. | `colony.PendingSuggestion` + `suggest_approve.go`'s tick-to-approve queue is the exact reusable mechanism D-07/D-10 call for; `AgencyReceiptEvidence.ChangedDecision`/`EffectEvidence` is the existing (currently unpopulated) outcome-join shape. |
</phase_requirements>

## Summary

This phase's hardest problem is not designing new machinery — it is discovering that most of the *safety* machinery already exists and is well-built (`aether spawn-log` / `spawn-can-spawn` already atomically enforce depth, whole-run budget, and ancestor-cycle refusal, with fail-closed error handling and a midden trail), while the thing the phase is actually named for — a worker recruiting a helper mid-task — has **no live path to a caste that is allowed to do it**. Every one of the 25 non-Queen agent definitions in `.claude/agents/ant/*.md` omits `Task` from its `tools:` frontmatter line [VERIFIED: .claude/agents/ant/aether-builder.md:6, .claude/agents/ant/aether-watcher.md, and 23 others — see table below], so a Builder, Watcher, Scout, etc. spawned today has no way to invoke the Task tool at all, regardless of what `.aether/workers.md` narrates about "Workers can spawn sub-workers directly using the Task tool" [VERIFIED: .aether/workers.md:266]. That narration is aspirational text describing a mechanism the tool grants never turned on — this project's own documented signature failure ("machinery exists but was never switched on").

Compounding this, Claude Code's own platform-level support for nested subagent-of-a-subagent spawning is itself disputed: the current docs page states nesting is on by default up to three layers as of v2.1.219 [CITED: code.claude.com/docs/en/sub-agents], but an OPEN upstream bug (anthropics/claude-code#80036, filed 2026-07-22, version 2.1.217, still open) reports that `general-purpose` and `claude` subagent types — the exact type `.aether/workers.md:266` tells workers to use — retain NO `Agent`/`Task` tool when spawned, and only the unrelated `fork` type does [CITED: github.com/anthropics/claude-code/issues/80036]. A second, older issue (#61993, filed 2026-05-24, v2.1.146) reporting the identical defect was closed "not planned" with no maintainer explanation [CITED: github.com/anthropics/claude-code/issues/61993]. This is a live, contested platform behavior, not a settled fact — BIO-03's "configurable, bounded... probe" requirement is not a nice-to-have here, it is the ONLY way this phase can honestly know, on the machine it is running on, whether depth-2 native nesting works at all.

Fortunately there is already a proven, honest fallback in this codebase: the autopilot/host-driven lane (`aether host build`) never uses nested Task-tool calls for its depth-1 workers at all — it shells a real OS subprocess via Node's `child_process.spawn(binary, args, { cwd: config.root, ... })` [VERIFIED: .aether/ts-host/src/platform-dispatcher.ts:412] against the `claude`/`opencode` CLI binaries, scoped to a `cwd` that is exactly the "workspace lease" vocabulary this codebase already uses elsewhere (`cmd/codex_build.go:67`, `cmd/codex_verify_advance.go:72`). Every worker caste already has Bash tool access [VERIFIED: all 27 `.claude/agents/ant/*.md` frontmatter `tools:` lines], so this same subprocess-spawn pattern is available to a depth-1 worker recruiting a depth-2 helper TODAY, without depending on any Task-tool nesting fix landing or not landing upstream. This is the load-bearing architectural choice this research recommends: **BIO-03's honest platform dispatch should default to the root/host-mediated subprocess fallback (proven, already in production for depth-1), and treat native Task-tool nesting as an optional, probed, depth-limited enhancement only where the probe confirms it actually works** — never the other way around.

The Go-side admission kernel (`spawnCanSpawnDecision` in `cmd/spawn.go:312`) is a genuinely excellent foundation: it checks depth, then whole-run budget (`spawnTreeBudgetReason`), then ancestor-cycle repetition (`spawnAncestorCycleReason`), each denying-by-name with a human-readable `Detail` string, fail-closed on every unreadable-state branch (D-19 discipline already established), and it already writes a midden entry on budget exhaustion. BIO-02's job is to extend this exact chokepoint — not replace it — with the two dimensions it does not yet check: **permission** (substrate already exists: `pkg/codex/permission_profile.go`'s `PermissionProfileForCaste`/`ResolvePermissionProfile`) and **path/cost** (substrate already exists: `cmd/spend_session_capture.go`'s `validateSpendContainedPath` containment pattern, and the existing spend ledger). BIO-01's typed `RecruitmentIntent` is genuinely greenfield — nothing named it before — but its wire shape should be a superset of today's `spawnDecisionInput`, not a parallel structure.

For exactly-once result delivery (BIO-04) and trophallaxis (BIO-05), do not invent a new idempotency mechanism: `PauseHandoff.HandoffID` plus `LifecycleTransactionReference`/`LifecycleReceiptReference` (`pkg/colony/lifecycle.go:474-503`) is this exact pattern, already proven for pause/resume replay safety, and `workerHandoffRecord` (`cmd/codex_dispatch_contract.go:487-506`) is the direct structural precedent for a scoped result-carrier.

For pheromones (BIO-07/BIO-08), the good news is the "five write sites" named in CONTEXT.md are mostly already funneling through one function: `createPheromoneSignal` (cmd/codex_workflow_cmds.go:1878) calls `persistPheromoneSignal` which calls `writePheromoneSignal` (cmd/pheromone_write.go:43) — the actual single write chokepoint already exists. The real fragmentation is downstream: the effective-scope/strength resolver that reaches worker briefs (`resolvePheromoneSection`, cmd/codex_build.go:4749) is a separate, simpler read path with no provenance or quarantine concept, and the outcome-credit join this phase needs is a literal, still-present stub: `SwarmPhase202Limitation = "Typed live checkpoint, pause, and resume machinery awaits Phase 202."` (cmd/agency_contract.go:15, still referenced at line 357) — a limitation string written when Phase 202 hadn't shipped yet, now stale since Phase 202 IS shipped, and this phase must be the one that finally reads real evidence there instead of printing that sentence forever. The tick-to-approve queue D-07/D-10 need already exists end-to-end: `colony.PendingSuggestion` + `suggest_approve.go`'s approve/dismiss/list commands, currently used only for `suggest-analyze` output.

**Primary recommendation:** Extend `spawnCanSpawnDecision` with permission/path/cost checks and a typed `RecruitmentIntent` input (BIO-01/02); default child dispatch to the already-proven subprocess/workspace-lease fallback rather than betting on disputed native Task-tool nesting, with a bounded, isolated, configurable probe deciding per-session whether native nesting is even attempted (BIO-03); reuse `PauseHandoff`'s handoff-ID/transaction/receipt pattern verbatim for `RecruitmentResult` (BIO-04) and `workerHandoffRecord`'s shape for `TrophallaxisPacket` (BIO-05); extend the existing `CeremonyTopicBuildSpawn` event and `agent.SpawnTree` to carry the whole governed subtree into watch/status (BIO-06); and finish wiring `writePheromoneSignal`'s already-singular write path into a real effective-scope/quarantine resolver and finally replace `SwarmPhase202Limitation` with genuine 202-event-sourced evidence (BIO-07/BIO-08/CEC-07).

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Recruitment admission (depth/budget/cycle/permission/path/cost) | API/Backend (Go runtime, `cmd/spawn*.go`) | — | "The program decides, not the wrapper" is this codebase's structural rule; `spawnCanSpawnDecision` is already the sole chokepoint and must stay so. |
| Recruitment intent authoring (deciding *whether* to recruit) | Client (the LLM worker process itself, via its own reasoning + Bash tool) | API/Backend (admission gate) | The worker decides it needs help; Go decides whether it may have it. Mirrors today's Queen-selects/Go-admits split. |
| Child dispatch (actually starting the recruit) | API/Backend (host-mediated subprocess spawn, `platform-dispatcher.ts`) | Client (native Task-tool nesting, where probed-available) | The proven, already-shipped mechanism for depth-1 dispatch is a Go/TS-mediated OS subprocess, not an in-session Task-tool call; depth-2 should follow the same tier by default. |
| Result return / exactly-once binding | API/Backend (Go lifecycle-transaction + receipt) | — | Matches the existing pause/resume authority; must never be a wrapper-owned or worker-self-reported fact. |
| Trophallaxis / knowledge handback | API/Backend (Go handoff store) | Client (worker reads the handed-back packet from its own brief) | Same split as today's worker-handoff relay: Go persists, wrapper/worker reads. |
| Live recruit/refusal rendering (D-04/D-06) | Client (inline terminal render, same session) | API/Backend (typed event source) | 202's hard constraint: inline-only, no second terminal; the Go event bus is the source of truth the same session's own render reads. |
| Pheromone write/normalize/dedupe | API/Backend (`writePheromoneSignal`, one function) | — | Already true today; BIO-07 must not create a second writer. |
| Pheromone effective-scope/quarantine resolution for worker briefs | API/Backend (a new resolver, replacing/extending `resolvePheromoneSection`) | — | Currently a naive read with no provenance; must become the "single source of truth" CONTEXT.md's Integration Points calls for. |
| Suggested/imported note approval (tick-to-approve) | Client (owner-facing queue render) | API/Backend (`colony.PendingSuggestion` store) | Same split `suggest_approve.go` already implements; reuse, do not duplicate. |

## Standard Stack

This phase is pure internal Go/TypeScript wiring inside an existing monorepo — no new external runtime dependency is anticipated for any BIO requirement. Do not introduce a new package (queueing library, actor framework, workflow engine, etc.) to satisfy BIO-01..08; every needed primitive (atomic file update, event bus, transaction/receipt, tick-to-approve queue) already exists in `pkg/storage`, `pkg/events`, and `pkg/colony`.

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| (none new) | — | — | The existing `pkg/storage.Store.UpdateJSONAtomically`/`AtomicWrite`, `pkg/events.Bus`, and `pkg/colony` lifecycle types cover every mechanism this phase needs. |

### Package Legitimacy Audit

Not applicable — no external packages are installed or recommended by this research. If implementation discovers a genuine gap (e.g., a process-tree-safe subprocess supervisor beyond Node's `child_process`), run the Package Legitimacy Gate protocol at that time; do not pre-approve anything here.

## Architecture Patterns

### System Architecture Diagram

```text
 Worker (depth N, mid-task, running as an OS process or Task-tool subagent)
   │
   │  1. decides it needs help (reasoning inside its own context)
   ▼
 RecruitmentIntent (BIO-01) — worker calls a new Go subcommand
   (parent identity, caste, objective, reason, evidence, permissions,
    urgency, scope, cost limit) — analogous to today's `aether spawn-log`
   but carrying the superset of fields BIO-01 requires
   │
   ▼
 Go admission gate (BIO-02) — extends spawnCanSpawnDecision:
   ┌─────────────────────────────────────────────────────────┐
   │ 1. depth check      (spawnMaxDelegationDepth, existing)  │
   │ 2. tree budget check (spawnTreeBudgetReason, existing)   │
   │ 3. ancestor-cycle    (spawnAncestorCycleReason, existing)│
   │ 4. permission check  (NEW — PermissionProfileForCaste)   │
   │ 5. path containment  (NEW — validateSpendContainedPath   │
   │                        pattern, scoped to a workspace)   │
   │ 6. cost/budget check (NEW — same team budget, D-12: no   │
   │                        separate recruit allowance)       │
   │ 7. duplicate check   (NEW — same caste+objective already │
   │                        pending in this subtree)          │
   └─────────────────────────────────────────────────────────┘
   │ deny (record + surface, D-03/D-06)      │ allow (atomic manifest
   ▼                                         │ amendment + midden-free path)
 Refusal recorded + inline "wanted backup,   ▼
 didn't get it" line (D-06); worker          Child dispatch (BIO-03)
 continues alone (D-03)                        │
                                                ├─ probe (bounded, isolated,
                                                │   outside repo) decides:
                                                │   native Task-tool nesting
                                                │   available? (per-session,
                                                │   NOT assumed from docs)
                                                │
                                                ├─ YES → native Task/Agent
                                                │   call (only if caste's
                                                │   own agent frontmatter
                                                │   grants Task — currently
                                                │   NONE do except Queen/
                                                │   Route-Setter)
                                                │
                                                └─ NO / unsupported →
                                                    root-mediated fallback:
                                                    child_process.spawn(
                                                      claude|opencode CLI,
                                                      cwd: <workspace lease>)
                                                    — the proven depth-1
                                                    pattern in
                                                    platform-dispatcher.ts
                                                       │
                                                       ▼
                                        Child runs, produces a result
                                                       │
                                                       ▼
                              RecruitmentResult (BIO-04) — exactly-once bind
                              via LifecycleTransactionReference + Receipt +
                              a HandoffID-shaped idempotency key (reused from
                              PauseHandoff pattern); replay of the SAME id
                              returns the SAME receipt, never re-applies
                                                       │
                                                       ▼
                              TrophallaxisPacket (BIO-05) — scoped handback,
                              shaped like workerHandoffRecord, requiring an
                              explicit acknowledgement from the parent OR one
                              named follow-on consumer before it counts as
                              "reached" for CEC-07 purposes
                                                       │
                                                       ▼
                        Recorded downstream decision (CEC-07) — the parent's
                        NEXT action changes because of this packet, and that
                        change itself becomes a LifecycleDecision entry
                                                       │
                                                       ▼
             Whole-subtree status/watch/cost/depth (BIO-06) — extends
             agent.SpawnTree (already whole-tree-aware) + Phase 202's
             typed ceremony event stream (CeremonyTopicBuildSpawn) so the
             live view and end-of-run family tree (D-04/D-05) read ONE
             source, never a second parallel accounting
                                                       │
                                                       ▼
   Pheromone bus (BIO-07/BIO-08) — writePheromoneSignal (already singular)
   + a new effective-scope/quarantine resolver replacing resolvePheromoneSection
   + colony.PendingSuggestion tick-to-approve queue (already built, reused
     for both D-07 suggestions and D-10 quarantined imports)
   + outcome-weighted strengthen/weaken/quarantine reading REAL evidence from
     the event stream above, replacing the still-live SwarmPhase202Limitation
     stub string
```

### Recommended Project Structure

Follow existing `cmd/` conventions (flat package, one concern per file, co-located `_test.go`). Suggested new files, none replacing existing ones:

```
cmd/
├── recruitment_intent.go       # BIO-01 typed RecruitmentIntent + validation
├── recruitment_admission.go    # BIO-02 — extends spawnCanSpawnDecision with
│                                #   permission/path/cost/duplicate checks;
│                                #   calls the existing depth/budget/cycle
│                                #   functions in spawn.go/spawn_budget.go/
│                                #   spawn_ancestor.go rather than reimplementing
├── recruitment_dispatch.go     # BIO-03 — probe + subprocess fallback +
│                                #   native-nesting attempt, gated by probe result
├── recruitment_result.go       # BIO-04 — RecruitmentResult binding, reusing
│                                #   PauseHandoff-shaped idempotency
├── trophallaxis.go             # BIO-05 — packet + acknowledgement + decision join
├── pheromone_resolver.go       # BIO-07 — the ONE effective scope/strength/
│                                #   quarantine resolver, replacing the naive
│                                #   read inside resolvePheromoneSection
└── pheromone_outcome.go        # BIO-08/CEC-07 — outcome-weighted strengthen/
                                 #   weaken/quarantine, reading real event
                                 #   evidence instead of SwarmPhase202Limitation

.aether/ts-host/src/
└── recruitment-probe.ts        # BIO-03 bounded/isolated/configurable probe
                                 #   for native nesting support, run OUTSIDE
                                 #   the repository working directory
```

### Pattern 1: Extend the existing admission chokepoint, don't fork it
**What:** `spawnCanSpawnDecision` (cmd/spawn.go:312) is a package-level function variable already designed to be the single point every future check gets added to (its doc comment explicitly says depth→budget→ancestor-cycle "each named and each denying on the first hit"). BIO-02's permission/path/cost/duplicate checks must be added as further ordered steps in this same function, not a parallel admission path for recruitment.
**When to use:** Any new admission rule for a spawn/recruitment decision.
**Example:**
```go
// Source: cmd/spawn.go:312 (read this session)
var spawnCanSpawnDecision = func(in spawnDecisionInput) spawnDecisionResult {
	prospectiveDepth := in.RequesterDepth + 1
	if prospectiveDepth > spawnMaxDelegationDepth { /* ... deny ... */ }
	if reason := spawnTreeBudgetReason(in); reason != "" { /* ... deny ... */ }
	if reason := spawnAncestorCycleReason(in); reason != "" { /* ... deny ... */ }
	// BIO-02 adds here: permission, path containment, cost, duplicate-intent
	return spawnDecisionResult{Allowed: true}
}
```

### Pattern 2: Admission-refusal-by-name, never silent trimming
**What:** Every existing refusal in this codebase names the exact offending thing and why (`"%s is at depth %d; a helper spawned from here would be depth %d, past the cap of %d"`, cmd/spawn.go:322-325; the coherent-jobs pattern in `TestCoherentJobProposalOrderRefusedByName` cited in CONTEXT.md). BIO-02 refusals and D-06's inline/summary refusal lines must follow the same convention — name the caste, the reason class (depth/budget/cycle/permission/path/cost/duplicate), and the human sentence, never a bare boolean.
**When to use:** Every BIO-02 deny path and every D-03/D-06 refusal render.

### Pattern 3: Idempotency key + transaction + receipt (reuse verbatim)
**What:** `PauseHandoff` (pkg/colony/lifecycle.go:474-503) carries `HandoffID`, a `LifecycleTransactionReference`, and an optional `LifecycleReceiptReference`; STATE.md's own decision log confirms "Pause and resume use the handoff ID as the idempotency key, so interrupted commits resume one journal and verified replay returns one existing receipt." BIO-04's `RecruitmentResult` needs exactly this shape: an ID that a duplicate/replayed completion report is matched against, returning the stored receipt rather than re-applying the result.
**When to use:** BIO-04's exactly-once binding; do not invent a second idempotency mechanism.

### Pattern 4: Root-mediated subprocess dispatch, already proven at depth 1
**What:** `spawnWorker` (`.aether/ts-host/src/platform-dispatcher.ts:399-435`, read this session) launches `spawn(binary, args, { cwd: config.root, signal, env })` against the resolved CLI binary, with a 10-minute AbortController timeout. This is the "root/host-mediated fallback with an exact workspace lease" BIO-03 asks for, already shipping for the autopilot/host-driven lane's depth-1 dispatch.
**When to use:** As BIO-03's DEFAULT dispatch mechanism for depth-2 recruitment, because (a) every worker caste already has Bash access to invoke a CLI subprocess itself, and (b) native Task-tool nesting is disputed platform behavior (see Common Pitfalls below) that this phase cannot assume works.

### Anti-Patterns to Avoid
- **Assuming native Task-tool nesting works because the docs say it does by default:** An open upstream bug (#80036) reports the exact opposite for the subagent types this repo's own `.aether/workers.md:266` instructs workers to use. Never ship BIO-03 without a runtime probe backing the claim.
- **Granting `Task` to every caste's frontmatter as the "fix":** Even if done, this only helps if the platform actually honors it for that `subagent_type` — the probe result must gate whether native nesting is attempted, not the frontmatter grant alone.
- **A second pheromone writer:** `writePheromoneSignal` is already the single chokepoint (cmd/pheromone_write.go:43); do not add a sixth write site while "consolidating" the five named ones — three of the five (`assumptions.go`, `discuss.go`, `codex_workflow_cmds.go`) already call it via `createPheromoneSignal`/`persistPheromoneSignal`.
- **A parallel recruitment ledger separate from `agent.SpawnTree`:** BIO-06 requires whole-subtree coverage; `agent.SpawnTree` already records parent/depth/status per entry across the whole run and is read by `spawn-tree-active`, `spawn-tree-depth`, `spawn-efficiency`. Extend it (new caste/status/cost fields as needed) rather than building a second tree structure recruitment must keep in sync with.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Depth/whole-run-budget/cycle admission | A new recruitment-specific limiter | `spawnCanSpawnDecision` + `spawnTreeBudgetReason` + `spawnAncestorCycleReason` (cmd/spawn.go, cmd/spawn_budget.go, cmd/spawn_ancestor.go) | Already fail-closed (D-19 discipline), already midden-logged, already exhaustively tested (`cmd/spawn_ancestor_test.go`, `cmd/spawn_budget_test.go`, `cmd/spawn_failclosed_test.go`). |
| Per-caste filesystem/behavioral permission | A new recruitment permission model | `pkg/codex/permission_profile.go` — `PermissionProfileForCaste`, `ResolvePermissionProfile`, `PermissionDecisionFor` | Already resolves per-caste + per-platform enforcement; BIO-02's "permissions" admission dimension is this, scoped to the recruit's declared caste. |
| Path/workspace containment | New path-scoping logic for a recruit's workspace lease | `cmd/spend_session_capture.go`'s `validateSpendContainedPath` pattern (filepath.Rel + leading-".." rejection, symlink-resolved on the deepest existing ancestor) | This exact bug class (macOS symlink resolution differing pre/post file creation) was already found and fixed here (STATE.md Phase 196 decision log); reusing it avoids reintroducing it. |
| Exactly-once result binding / replay safety | A new dedup/idempotency table | `LifecycleTransactionReference` + `LifecycleReceiptReference` + `HandoffID` pattern (pkg/colony/lifecycle.go) | Proven in production for pause/resume; CLAUDE.md's own Definition of Done cites this pattern's rigor bar directly ("Classify missing transaction evidence as unknown and altered evidence as conflicting; only matching digests may finish or roll back."). |
| Owner approval queue for suggested/imported content | A new "quarantine inbox" UI/store | `colony.PendingSuggestion` + `suggest_approve.go` (list/approve/dismiss, dedup by content hash) | D-07 and D-10 both explicitly ask for "the same tick-to-approve queue," not two ceremonies; this is that queue. |
| Live inline event rendering | A new event schema for recruitment | Phase 202's typed ceremony bus (`pkg/events`, `CeremonyTopicBuildSpawn` already exists at pkg/events/ceremony.go:8) | `TestEveryLiveEventGoesThroughOneBoundary` (202) locks this as the single event boundary; adding a `ceremony.recruitment.*` topic family is additive, a second bus is not. |
| Sanitizing worker/import-authored pheromone text | New XSS/injection scrubbing | `colony.SanitizeSignalContent` (already used by all midden writers per STATE.md Phase 198.1 decision log) | Worker-authored and cross-project-imported text is exactly the untrusted-input case this function already exists for. |

**Key insight:** This phase's risk is not "build the wrong thing" — the primitives (admission gate, idempotency pattern, event bus, approval queue, sanitizer, permission model, path containment) all already exist and are well-tested in isolation. The risk is exactly this project's documented failure mode: shipping BIO-01..08 as new files that reference these primitives in a comment but never actually call them from the real dispatch/verification path, leaving a second orphaned "biological runtime" next to the first orphaned SPAWN-* one. Every plan for this phase should be able to name the existing call site it is extending.

## Common Pitfalls

### Pitfall 1: Trusting the platform-nesting docs without a same-session probe
**What goes wrong:** Design BIO-03 around "Claude Code supports 3 layers of nested subagents by default" (the current docs claim), ship it, and depth-2 recruitment silently degrades to depth-1-only or errors, because the actual runtime strips the Agent/Task tool from `general-purpose`/`claude`-typed nested subagents.
**Why it happens:** The capability's own history is genuinely unstable — introduced in v2.1.172 (2026-06-09) at 5 layers, disabled by default in v2.1.217 (2026-07-21), re-enabled at depth 3 by default in v2.1.219 (2026-07-24), all within about 6 weeks, per third-party changelogs [CITED: dev.classmethod.jp/en/articles/20260722-cc-updates-v2-1-217, www.turboai.dev/blog/claude-code-env-vars-v2-1-217] — these are secondary sources, not Anthropic's own changelog, and should be treated as [ASSUMED]-tier until the planner reproduces the behavior directly.
**How to avoid:** BIO-03's probe must run an actual empirical check at session start (or lazily on first recruitment attempt) — spawn a real subagent of the exact caste/subagent_type this repo uses, ask it to report whether `Agent`/`Task` appears in its own available tool list, and cache the honest result for that session/version. Never branch on a hardcoded "supported platforms" table.
**Warning signs:** A recruitment that "succeeds" per Go's admission gate but the child never actually starts (the parent's Task call silently returns nothing or errors); BIO-03 explicitly requires this to "fail before execution," which requires the probe to run BEFORE the admission gate reports allow, not after.

### Pitfall 2: Treating `.aether/workers.md`'s existing "Spawning Sub-Workers" section as already-implemented behavior
**What goes wrong:** Planning BIO-01..03 as "wire the existing protocol into Go" when the existing protocol (workers.md lines 264-380) was never backed by an actual tool grant on any caste.
**Why it happens:** The doc reads exactly like an implemented feature — it has a "Step-by-Step Spawn Protocol," exact bash commands, exact Task-tool invocation text — but zero of the 25 non-Queen/Route-Setter agent `.md` files grant `Task` in their `tools:` frontmatter [VERIFIED: grep across all 27 `.claude/agents/ant/*.md` files, this session — only `aether-queen.md` (`tools: Read, Write, Edit, Bash, Grep, Glob, Task`) and `aether-route-setter.md` (`tools: Read, Grep, Glob, Bash, Write, Task`) include it].
**How to avoid:** Treat workers.md's recruitment protocol as design intent to formalize under BIO-01, not as prior art to "just wire up." Any plan must explicitly decide, per caste, whether that caste is allowed to recruit (frontmatter change) and must not assume the doc's existence means the capability exists.
**Warning signs:** A plan that says "restore" or "reconnect" recruitment rather than "build" it.

### Pitfall 3: A second, uncalled admission engine (repeat of the ts-host `SpawnOrchestrator` history)
**What goes wrong:** `.aether/ts-host/src/spawn-orchestrator.ts`'s `createSpawnOrchestrator` is instantiated in `host.ts:1105` with a real budget and depth, but `.processClaims()` — the only method that actually applies its depth/budget checks — is never called anywhere in `host.ts` [VERIFIED: `grep -n "spawnOrchestrator\.\|processClaims(" .aether/ts-host/src/host.ts` returned zero matches for either, this session]. This is the exact shape of orphan this phase must not reproduce: a correctly-written policy engine, constructed with real data, that nothing ever consults.
**Why it happens:** The v1.26 SPAWN work built the TS-side policy engine as "future use" infrastructure and moved the actual enforcement into Go (`cmd/spawn.go`'s `spawnCanSpawnDecision`) without deleting or wiring the TS twin.
**How to avoid:** Either delete `createSpawnOrchestrator`/`synthesizeChildDispatch` as confirmed dead code, or make BIO-02's Go admission gate the ONLY chokepoint and have the TS host call INTO it (via the Go binary) rather than maintain its own parallel budget arithmetic. Do not "fix" the orphan by finally calling `processClaims()` — that would create two independent, potentially disagreeing budget counters (the Go `spawnTreeBudgetMax=20` whole-run ledger vs. this TS-side `DEFAULT_TOTAL_BUDGET=20` per-orchestrator-instance count).
**Warning signs:** Any new code path that computes "remaining budget" without calling into `spawnTreeBudgetState()`.

### Pitfall 4: Printing `SwarmPhase202Limitation` forever
**What goes wrong:** CEC-07 and BIO-08's outcome-join requirements get "satisfied" by code that still returns the literal string `"Typed live checkpoint, pause, and resume machinery awaits Phase 202."` (cmd/agency_contract.go:15, still assigned at line 357) [VERIFIED: this session, `grep -n "SwarmPhase202Limitation" cmd/*.go`], because nobody wired the now-shipped Phase 202 event stream into `AgencyReceiptEvidence.ChangedDecision`/`EffectEvidence`.
**Why it happens:** The string was accurate when written (Phase 202 hadn't shipped); it silently became false the moment Phase 202 merged, and nothing ratchets against a stale "awaits Phase N" string once N ships.
**How to avoid:** This phase's plan must include a task that replaces this constant's call site with a real query against the Phase 202 event/ceremony trail, and a test asserting the limitation string is no longer reachable once a decision genuinely changed.
**Warning signs:** Any BIO-08 test that only checks the SHAPE of `AgencyReceiptEvidence` rather than asserting a real `ChangedDecision` gets populated from a real recorded event.

### Pitfall 5: Latency regression on the plain no-recruitment path
**What goes wrong:** Adding permission/path/cost/duplicate checks to the admission gate, plus a per-session platform probe, adds meaningful wall-clock time to every single build even when nothing ever recruits — directly violating the folded-todo constraint that "recruitment admission must add no meaningful latency to the plain no-recruitment path."
**Why it happens:** A probe that shells out to check native-nesting support, or a permission resolution that re-reads and re-parses files on every admission check, is invisible in a unit test but adds real seconds across dozens of dispatches in a build with many workers, none of which ever recruit.
**How to avoid:** Cache the platform probe result once per session/run (not per recruitment attempt); make permission/path/cost checks pure in-memory comparisons against already-loaded state, matching the existing depth/budget/cycle checks' cost profile (all in-memory once the ledger is parsed).
**Warning signs:** A build's own closeout cost/timing line (D-14 from Phase 201) shows a new segment attributable to recruitment admission on phases that never recruited.

## Code Examples

### The existing three-check admission chokepoint (extend this)
```go
// Source: cmd/spawn.go:307-338 (read this session)
var spawnCanSpawnDecision = func(in spawnDecisionInput) spawnDecisionResult {
	prospectiveDepth := in.RequesterDepth + 1
	if prospectiveDepth > spawnMaxDelegationDepth {
		// ... deny with named requester, current depth, prospective depth, cap ...
	}
	if reason := spawnTreeBudgetReason(in); reason != "" {
		return spawnDecisionResult{Allowed: false, Reason: "budget", Detail: reason}
	}
	if reason := spawnAncestorCycleReason(in); reason != "" {
		return spawnDecisionResult{Allowed: false, Reason: "ancestor-cycle", Detail: reason}
	}
	return spawnDecisionResult{Allowed: true}
}
```

### The existing depth-1 subprocess dispatch (the honest BIO-03 fallback)
```ts
// Source: .aether/ts-host/src/platform-dispatcher.ts:399-435 (read this session)
export async function spawnWorker(config: WorkerConfig): Promise<SpawnResult> {
  const binary = resolveBinaryName(config.platform);
  const args = buildArgs(config);
  const abortController = new AbortController();
  const timeoutId = setTimeout(() => abortController.abort(), 10 * 60 * 1000);
  return new Promise<SpawnResult>((resolve) => {
    const child = spawn(binary, args, {
      cwd: config.root,          // <-- this IS the "exact workspace lease"
      signal: abortController.signal,
      env: { ...process.env },
    });
    // ... collect stdout/stderr, resolve on exit/abort ...
  });
}
```

### The existing idempotent transaction/receipt shape (reuse for RecruitmentResult)
```go
// Source: pkg/colony/lifecycle.go:474-503 (read this session)
type PauseHandoff struct {
	SchemaVersion string `json:"schema_version"`
	HandoffID     string `json:"handoff_id"`
	// ... OutcomeKind, AttemptID, RunID, SafeBoundary, RestartPoint ...
	Transaction LifecycleTransactionReference `json:"transaction"`
	Receipt     *LifecycleReceiptReference    `json:"receipt,omitempty"`
	Recovery    *LifecycleRecovery            `json:"recovery,omitempty"`
	Provenance  RecoveryProvenance            `json:"provenance"`
}
```

### The already-singular pheromone writer (do not add a sixth call site)
```go
// Source: cmd/codex_workflow_cmds.go:1903-1917 (read this session)
func persistPheromoneSignal(sigType, content, sourceFlag, reasonFlag, ttlFlag string, strength float64, priority string) (colony.PheromoneSignal, bool, int, error) {
	signal, reinforced, err := writePheromoneSignal(sigType, content, priority, sourceFlag, reasonFlag, ttlFlag, strength, nil)
	// ... total count from pheromones.json ...
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| v1.26 SPAWN-* work: TS-side `SpawnOrchestrator` policy engine as the intended enforcement point | Go's `spawnCanSpawnDecision` (cmd/spawn.go) became the real enforcement point; TS engine still exists but is uncalled | Phase 173 (v1.26) | BIO-02 must extend the Go side; the TS side is a dead-code decision the phase should make explicitly (delete or bridge), not ignore. |
| Claude Code: subagents are always leaf nodes, cannot spawn further subagents | Native nested spawning introduced (v2.1.172, 2026-06-09), briefly disabled-by-default (v2.1.217, 2026-07-21), re-enabled at default depth 3 (v2.1.219, 2026-07-24) | June-July 2026 | This capability's own history within the last ~3 months shows it is actively unstable; BIO-03 must not treat any single doc snapshot as durable truth. |
| `.aether/workers.md`'s recruitment protocol assumed workers have Task access | Confirmed this session: none do, except Queen/Route-Setter | N/A (never actually true) | BIO-01 is genuinely new work, not "reconnecting" an existing wire. |

**Deprecated/outdated:**
- `SwarmPhase202Limitation` string (cmd/agency_contract.go:15) — accurate before Phase 202 shipped, false now; this phase should remove or replace its call site.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Claude Code's documented default nested-subagent depth (3 layers, `CLAUDE_CODE_MAX_SUBAGENT_SPAWN_DEPTH`) is accurate for the version this repo's users actually run. | Platform Mechanism Study / Pitfall 1 | If wrong (older/newer CLI version, or the disputed #80036 behavior is still live), BIO-03's native-nesting branch never fires and every recruitment silently falls back — which is safe (fails closed to the proven subprocess path) but means D-02's "chains" value is delivered entirely through the subprocess fallback, not native nesting, which should be an explicit design decision, not a surprise. |
| A2 | The version-history timeline (v2.1.172 → v2.1.217 → v2.1.219 → v2.1.224) sourced from third-party blogs (dev.classmethod.jp, turboai.dev, dev.to) is accurate. | State of the Art | These are not Anthropic's own changelog; if a date or version number is off, it doesn't change the phase's recommended architecture (probe-first, subprocess-default) but would misdate the "why this is unstable" narrative in the required CLASSIC-SYNTHESIS.md. |
| A3 | GitHub issue #80036 (open, 2026-07-22) still accurately describes current behavior at the time this phase implements BIO-03 (2026-09-12 research date). | Platform Mechanism Study / Pitfall 1 | If Anthropic fixed this between now and implementation, the probe (which must be built regardless, per BIO-03's own wording) will simply report native nesting as available — no rework needed, since the design is probe-driven rather than hardcoded either way. |
| A4 | No new external package is needed for any BIO requirement. | Standard Stack | If implementation discovers e.g. a genuine need for a process-tree-safe supervisor beyond `child_process`, the Package Legitimacy Gate must be run at that time — this claim only says nothing is pre-approved, not that nothing will ever be needed. |

**Assumption A2/A3 in particular should be spot-checked once by the planner or the CLASSIC-SYNTHESIS.md author with a fresh WebSearch immediately before implementation, since this capability has changed roughly monthly for the last several months.**

## Open Questions

1. **Should any existing caste (Builder, Watcher, etc.) be granted `Task` in its frontmatter, or should ALL depth-2 recruitment go through the subprocess fallback exclusively for v1?**
   - What we know: The subprocess fallback is proven, honest, and available to every caste today via Bash. Native nesting is disputed and version-sensitive.
   - What's unclear: Whether native nesting (when the probe confirms it works) offers a meaningfully better owner experience (e.g., cheaper — same context/session vs. a fresh CLI process cost) that's worth the platform risk.
   - Recommendation: Ship the subprocess fallback as the only path in v1 of this phase; treat native nesting as a follow-on enhancement gated by the probe, never a launch-blocking dependency. This keeps SYNTH-05's "platform-dependent" honesty requirement satisfied without betting the phase's completion on an upstream bug's fate.

2. **Does D-11's "default chain depth is 2 levels... raisable per-run by explicit flag" map onto the EXISTING `spawnMaxDelegationDepth = 2` constant, or is that constant already "2" for an unrelated historical reason and this phase must make it configurable for the first time?**
   - What we know: `spawnMaxDelegationDepth = 2` already exists and is exactly the number D-11 wants as a default [VERIFIED: cmd/spawn.go:274, "const spawnMaxDelegationDepth = 2"]. `.aether/workers.md` documents the identical depth-2 rule as the existing colony-wide standard, unrelated to recruitment intents specifically.
   - What's unclear: Whether D-11's "raisable per-run by explicit flag" requires making this a per-run configurable value (a genuine behavior change to a constant that today is fixed and, per its sibling `spawnTreeBudgetMax`'s comment, was deliberately made non-configurable "because a limit you can raise under pressure is a limit that gets raised") — this is a direct tension between D-11 (owner wants a raise flag) and this codebase's own stated philosophy for the sibling budget constant.
   - Recommendation: Surface this tension explicitly in the CLASSIC-SYNTHESIS.md and/or to the owner during planning — D-11 is a locked decision, but the sibling constant's own comment is a direct philosophical counter-argument the planner should not silently override without acknowledging it.

## Sources

### Primary (HIGH confidence — read directly this session)
- `cmd/spawn.go`, `cmd/spawn_budget.go`, `cmd/spawn_ancestor.go` — the full existing admission chokepoint, depth/budget/cycle logic, fail-closed discipline.
- `pkg/agent/spawn_tree.go` — whole-run spawn ledger, parse/format contract, live/terminal status vocabulary.
- `pkg/colony/lifecycle.go` (PauseHandoff, SealOutcome, SignalDeliveryReceipt, SignalAcknowledgement) — idempotency/transaction/receipt pattern.
- `cmd/codex_dispatch_contract.go` (workerHandoffRecord) — handoff/trophallaxis structural precedent.
- `cmd/agency_contract.go`, `cmd/codex_workflow_cmds.go`, `cmd/pheromone_write.go` — pheromone write chokepoint and outcome-evidence stub.
- `.aether/ts-host/src/spawn-orchestrator.ts`, `.aether/ts-host/src/host.ts`, `.aether/ts-host/src/platform-dispatcher.ts`, `.aether/ts-host/src/worker-dispatch.ts` — TS-side orphaned policy engine vs. proven subprocess dispatch.
- All 27 `.claude/agents/ant/*.md` frontmatter `tools:` lines — grep this session confirming only Queen/Route-Setter grant `Task`.
- `.aether/workers.md` (lines 1-80, 260-380) — the documented-but-unbacked worker recruitment protocol.
- `.planning/codebase/ARCHITECTURE.md`, `.planning/codebase/CONCERNS.md` — layering and named defects (dated 2026-08-22; largely still accurate per this session's direct code reading, with the exception that Phase 201/202 have since resolved some named concerns).
- `.planning/REQUIREMENTS.md`, `.planning/ROADMAP.md` §Phase 203, `.planning/phases/203-biological-runtime/203-CONTEXT.md`, `.planning/phases/201-.../201-CONTEXT.md`, `.planning/phases/202-.../202-CONTEXT.md`.

### Secondary (MEDIUM confidence)
- code.claude.com/docs/en/sub-agents — official current documentation on nested subagent depth limits and tool inheritance filters.
- github.com/anthropics/claude-code/issues/80036 (open, filed 2026-07-22) — reports Agent/Task tool stripped from `general-purpose`/`claude` nested subagents.
- github.com/anthropics/claude-code/issues/61993 (closed "not planned," filed 2026-05-24) — same defect class, no maintainer explanation given.

### Tertiary (LOW confidence — third-party, unverified against Anthropic's own changelog)
- dev.to/ucjung/claude-code-subagents-can-now-spawn-subagents-4h2a
- dev.classmethod.jp/en/articles/20260722-cc-updates-v2-1-217
- www.turboai.dev/blog/claude-code-env-vars-v2-1-217
- OpenCode nested-subagent issues (anomalyco/opencode #8114, #18100, #9280) — informational only; OpenCode is the secondary platform per project policy and was not verified against this repo's own OpenCode agent definitions in this session (recommend the planner repeat the `.opencode/agents/*` frontmatter grep before finalizing BIO-03 for OpenCode parity).

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependency, all primitives read directly from this repo.
- Architecture (Go-side admission/idempotency/handoff substrate): HIGH — read directly, line-cited, cross-checked against STATE.md's own decision log.
- Architecture (platform dispatch mechanics): MEDIUM/LOW — genuinely contested and fast-moving upstream behavior; the recommended design (probe-first, subprocess-default) is chosen specifically to be robust to this uncertainty rather than to resolve it.
- Pitfalls: HIGH for the in-repo orphan patterns (verified by direct grep/read this session); MEDIUM for the platform-nesting pitfall (web-sourced).

**Research date:** 2026-09-12
**Valid until:** 7 days for the platform-nesting claims (this capability has changed roughly monthly upstream — re-verify immediately before implementation); 30 days for the in-repo architecture findings (stable unless another phase lands first).
