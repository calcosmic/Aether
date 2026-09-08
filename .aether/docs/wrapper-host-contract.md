# Wrapper-Host Contract

**Status:** Accepted  
**Date:** 2026-05-15  
**Decision:** Wrappers are **host-assisted orchestrators**, not thin pass-throughs.

---

## Context

v1.19 cut over the primary plan/build/heavy-review wrapper paths to use the
TypeScript host (`aether host <workflow>`) as the preferred manifest-generation
spine. v1.20 hardened that host surface so the documented host commands and
flags work reliably.

The open question: should wrappers become thin pass-throughs (just call `aether host` and render output), or do they retain orchestration responsibilities?

## Decision

**Wrappers are host-assisted orchestrators.**

- Wrappers call `aether host` for **host-backed manifest generation**
  (`colonize`, `plan`, `build`, heavy/classic `continue`, and `seal`)
- Wrappers handle **colony ceremony**: worker spawning, wave management, closeout rendering
- Wrappers do **not** duplicate verification, gating, or state mutation logic owned by the Go runtime
- The TS host is the **sole entry point** between wrappers and Go CLI for
  host-backed manifest generation

## Boundary Rules

| Layer | May | Must Not |
|-------|-----|----------|
| **Wrappers** | Call `aether host` for host-backed flows, spawn only runtime-authorized workers, render ceremony, collect a runtime-defined preset or exact owner answer, execute a runtime-returned exact command, add colony framing/narration, read a build dispatch's brief from `brief_path` (a runtime-produced file holding the identical bytes as the inline `brief`, preferred for briefs subject to Read-tool long-line truncation) | Duplicate verification/gating, mutate colony state, edit `SPEC.md` or plan artifacts, compute planning scores/stops, mint receipts/tokens/recommendations, change candidate status or active revision, parse visual output as authoritative, expose raw provider stdout/stderr or auth probe output, document unimplemented future host targets as implemented |
| **TS Host** | Parse flags, call Go CLI via JSON, render dashboards, manage event streams | Write to `.aether/data/` directly, duplicate Go-owned logic, invent provider/auth diagnostics |
| **Go CLI** | Own all state mutations, verification, gating, finalizers, canonical artifact writes, provider availability preflight diagnostics | Spawn platform agents (Claude/OpenCode/Codex workers) |

Current wrapper host-backed manifest surfaces are `colonize`, `plan`, `build`,
heavy/classic `continue`, and `seal`. The TS host also exposes `oracle`,
`watch`, and `swarm` display/lifecycle surfaces, but canonical swarm wrappers
still use the Go plan-only/finalizer path.

## Manifest and Completion Packet Shapes

This is the single place manifest field names are documented, so wrapper prose
can name a step without enumerating a schema.

| Manifest key | Producer | Wrapper obligation |
|--------------|----------|---------------------|
| `result.manifest.dispatch_manifest` | TS host | The sole source of worker names, castes, waves, and briefs for this build — spawn exactly what it names, nothing invented. |
| `dispatch_manifest.execution_plan` | TS host | Respect wave order: serial steps stay serial, parallel steps may spawn together. |
| `dispatch_manifest.context_capsule` | TS host | Read once per build; prepend verbatim once per build ahead of every worker's brief. |
| `dispatch.brief` | TS host | Read one of brief/brief_path verbatim, never merge or reconstruct — the complete runtime-rendered worker prompt. |
| `dispatch.brief_path` | TS host | Read one of brief/brief_path verbatim, never merge or reconstruct — prefer this channel for briefs subject to Read-tool long-line truncation; both carry the same bytes. |
| `dispatch.skill_section` | TS host | Append verbatim after the brief when present; never summarize, reorder, or reconstruct. |
| `dispatch.permission_profile` | TS host | Preserve, never broaden `repository_read_only`; reject `scoped_write` or `test_write` when the host cannot enforce it. |
| `dispatch_manifest.orchestrator_boundary_guidance` | TS host | Route to `aether discuss` when active or `next` is `aether discuss`; request a fresh manifest after resolution. |
| `result.manifest.continue_manifest` | TS host | The sole source of reviewer names, castes, waves, and briefs for heavy continue — spawn exactly what it names. |
| `result.plan_manifest.stage_manifest` | TS host over Go | The sole planning worker authority for the current stage. It binds one `authorization_id`, run/pass, preset, approved Specification revision/hash/receipt, base-plan revision/hash, input frontier, expected caste/result type, and caste-specific predecessor fields. |
| `stage_manifest.expected_caste` | Go CLI | Dispatch exactly one `scout` or one `route_setter`. A Route-Setter stage exists only after the exact Scout receipt; a later Scout exists only after a completed card. |
| `stage_manifest.expected_result_type` | Go CLI | Require exactly `planning-scout-result/v1` or `planning-route-setter-result/v1`; do not add state, score, stop, candidate, acceptance, or next-stage fields to worker output. |
| `result.stage_receipt` | Go CLI | Retain the content-addressed Scout receipt returned by finalization; it proves the consumed manifest/output and resulting boundary but grants no future-stage authority. |
| `result.route_stage_manifest` | Go CLI | When present, this alone authorizes the current pass's Route-Setter and binds the exact Scout receipt plus candidate snapshot. |
| `result.route_stage_receipt` | Go CLI | Retain the content-addressed Route-Setter receipt and render the complete iteration card before acting on the next boundary. |
| `result.scout_stage_manifest` | Go CLI | When present after a card, this alone authorizes the next pass's gap-targeted Scout. |
| `result.decision_cards` | Go CLI | Render the complete evidence-first owner batch and submit only the exact selected answers with the issued resume binding. The host never selects or classifies an answer. |
| `result.iteration_card` | Go CLI | Render once and retain in order: receipts, evidence, five validated dimensions, weakest gap, semantic delta, stop/pause reason, and `evidence_that_would_change`. |
| `result.plan_candidate` | Go CLI | Label the stopped proposal `NOT ACTIVE`; review it through `aether plan --candidate` before exposing any acceptance choice. |
| `result.acceptance_command` | Go CLI | After explicit owner acceptance, execute the full command verbatim. Never reconstruct, shorten, or substitute its Specification/base/timeline/proposal/token bindings. |
| `result.completion_path` | Go CLI | Finalize only the Go-owned path — the durable packet at this path, never the staged completion file directly. |

Wrapper prose points here once and does not restate this table.

The planning rows deliberately do not list the retired whole-chain
`result.planning_manifest`, `result.depth_proposal_card`,
`result.research_proposal_card`, or `result.requires_next_iteration` contract.
Planning now advances only through the current stage manifest, its exact
finalizer receipt, and the next boundary selected by Go.

## Planning Authority and Replay

In plain language, a stage manifest is a one-use ticket for one worker; a
receipt proves what Go accepted at that boundary; neither lets the wrapper
decide what happens next.

- Specification approval, a planning stop, and candidate acceptance are three
  separate transitions. Specification approval permits planning against one
  revision. A Go-selected stop creates a reviewable non-active candidate. Only
  the exact candidate `acceptance_command` may activate a PlanRevision.
- Route-Setter proposes five readiness dimensions. Go validates the evidence
  and derives overall readiness, semantic delta, weakest gap, materiality, and
  stop policy. The host renders those results and never recomputes them.
- A material owner answer returns either `direct_resume` or
  `successor_spec_required`. The host never chooses the branch. A successor
  remains DRAFT until exact Specification approval and affected-scope
  reconciliation complete.
- Gaps, completed cards, stop decisions, and candidates retain
  `evidence_that_would_change`. Candidate review retains the Queen
  recommendation's producer, producer ID, evidence IDs, rationale, and
  accept/revise disposition; advice grants no authority.
- Exact replay is idempotent: return the existing artifact/receipt and do not
  duplicate state. Divergent replay or stale identity fails with state
  unchanged and the runtime's exact recovery command. The wrapper may not
  repair a mismatch by selecting newer IDs or editing the packet.

## Platform Display Spelling

Claude Code and OpenCode render owner navigation as `/ant-spec`, `/ant-plan`,
`/ant-build`, and `/ant-run`, while executing the runtime command supplied by
the host. Codex is runtime-native and displays `aether spec`, `aether plan`,
`aether build`, and `aether run`; it does not advertise a native `$ant-*`
surface. Platform spelling changes presentation, never state or authority.

## Terminal Worker Result Belongs to the Wrapper

The terminal structured result a worker must return — `name`, `caste`,
`stage`, `execution_wave`, `task_id`, `status`, `summary`, `files_created`,
`files_modified`, `tests_written`, `blockers`, `duration`, `handoff`, and the
`handoff` object's own required keys (`changed_files`, `commands_run`,
`verification_status`, `known_failures`, `open_decisions`, `assumptions`,
`next_worker_instructions`, `do_not_repeat`, `freshness`) — is **method, not
envelope mechanics**: it describes what a worker owes back, the opposite
direction of data flow from manifest consumption above. It therefore stays
inline in the lifecycle wrappers and is explicitly NOT counted as
envelope-parsing prose by the Phase 165 CMD-02 proportion test.

Direction rule: inbound manifest field shapes live in this document; outbound
worker result obligations live in the wrapper that spawns the worker.

## Provider/Auth Boundary

Provider availability preflight belongs to the Go runtime. Wrappers and the TS
host may surface the sanitized provider, cause, and next action from the
runtime, but must not paste raw provider output, tokens, or auth probe details.

Post-launch provider/API/auth failures are a different class: the worker process
may start and then return provider output instead of Aether worker claims JSON.
Report those as launched-worker provider/API/auth failures using sanitized
wording; do not reinterpret them as missing provider availability preflight.

## Rationale

1. **Thin pass-throughs would lose colony ceremony.** The interactive worker spawning, wave banners, and closeout rendering are core to the Aether user experience. Moving all of that into the TS host would be a massive v1.21+ effort, not v1.20 scope.

2. **The host is reliable for its implemented spine.** The supported host
   commands work end-to-end with correct flags. Direct Go plan-only/finalizer
   calls remain intentional only for flows that are not host manifest targets.

3. **Verification stays in Go.** Wrappers do not reimplement gate logic — they call `aether *-finalize` and let the runtime decide advancement.

4. **Single entry point where implemented.** Wrappers call `aether host` for
   host-backed manifest generation and clearly document exceptions for any
   future targets.

## Migration Notes

- **Remove fallback paths for host-backed flows:** Wrappers previously fell back
  to direct Go CLI calls when the TS host was "unavailable." For host-backed
  plan/build/heavy-continue paths, those fallbacks are dead code and should stay
  removed.
- **Future direction:** As the TS host gains orchestration capabilities (worker dispatch, ceremony rendering), wrappers can become thinner over time. This ADR should be revisited in v1.21+.

## Consequences

- Wrappers remain moderately complex (they handle ceremony)
- But they are simpler than before (no fallback paths, no direct Go CLI calls)
- The boundary is explicit: wrappers orchestrate UX, host generates manifests, Go owns truth
