# Aether

## What This Is

Aether is a biomimetic AI colony framework: a Go runtime in `cmd/` and `pkg/` that owns state, worker dispatch, verification, memory, and install/update flows, plus companion command surfaces for Claude Code and OpenCode and a runtime-native Codex CLI lane.

`v1.0` restored the lost colony ceremony and runtime visibility surfaces.

`v1.1` made Aether's context layer trustworthy, inspectable, deterministic, and benchmarkable.

`v1.8` added the colony recovery system: `aether recover` detects 7 stuck-state classes, auto-fixes safe issues, prompts for destructive ones, and proves correctness through 10 E2E tests.

## Core Value

**Aether should feel alive and truthful at runtime, not only look clever in wrappers or tests.**

That means:

- worker lifecycle must be inspectable and honest
- dispatch visibility must come from real runtime state
- stale run state must not poison future commands
- verification must lead advancement decisions
- partial success and recovery must be first-class
- stuck colonies must be recoverable with a single command

## Current State

- Go runtime is healthy, v1.0.24 shipped
- All 9 milestones complete (51 phases, 119 plans)
- 2910+ tests passing, full E2E regression coverage
- Stable and dev publish channels with integrity verification
- Plan recovery pipeline hardened (`--force` always recovers)
- Colony recovery system shipped: `aether recover` + `--apply` for stuck-state rescue
- All 50 slash commands working across Claude Code, OpenCode, and Codex CLI

<details>
<summary>Prior State History</summary>

- v1.0 (MVP, Phases 1-6): Colony ceremony and runtime visibility
- v1.1 (Trusted Context, Phases 7-11): Context proof and skill routing
- v1.2 (Live Dispatch Truth, Phases 12-16): Worker dispatch honesty
- v1.3 (Visual Truth, Phases 17-24): Caste identity, stage separators, trace logging
- v1.4 (Self-Healing, Phases 25-30): Medic ant, ceremony integrity
- v1.5 (Runtime Truth Recovery, Phases 31-38): Continue unblock, release v1.0.20
- v1.6 (Release Pipeline, Phases 39-46): Publish hardening, E2E regression
- v1.7 (Planning Pipeline, Phases 47-48): Plan --force recovery, E2E recovery test
- v1.8 (Colony Recovery, Phases 49-51): Stuck-state detection, auto-repair, E2E verification

</details>

## Architecture / Key Patterns

- **Go runtime is authoritative** for state mutations, verification, and CLI truth
- **Wrappers are presentation-only** on Claude/OpenCode
- **Codex is runtime-native**; no markdown wrapper ceremony
- **YAML remains source-of-truth** for generated wrapper commands
- **Runtime proof beats wrapper theater**
- **Shared lifecycle truth matters**; `build`, `plan`, `colonize`, `watch`, `status`, and `continue` should agree on what a worker is doing
- **Recovery is first-class**; stuck colonies get a rescue button, not manual file surgery

## Milestone Sequence

- [x] v1.0 MVP — Phases 1-6
- [x] v1.1 Trusted Context — Phases 7-11
- [x] v1.2 Live Dispatch Truth and Recovery — Phases 12-16
- [x] v1.3 Visual Truth and Core Hardening — Phases 17-24 (shipped 2026-04-21)
- [x] v1.4 Self-Healing Colony — Phases 25-30 (completed 2026-04-21)
- [x] v1.5 Runtime Truth Recovery — Phases 31-38 (completed 2026-04-23, product v1.0.20)
- [x] v1.6 Release Pipeline Integrity — Phases 39-46 (completed 2026-04-24)
- [x] v1.7 Planning Pipeline Recovery — Phases 47-48 (completed 2026-04-24)
- [x] v1.8 Colony Recovery — Phases 49-51 (completed 2026-04-25)
- [ ] v1.9 Review Persistence — Phases 52+

## Current Milestone: v1.9 Review Persistence

**Goal:** Review agent findings survive `/clear` and accumulate across phases so downstream workers can learn from prior reviews.

**Target features:**
- Continue-review worker outcome reports (per-worker `.md` files, mirroring build worker reports)
- Domain-ledger system (`reviews/{domain}/ledger.json`) — 7 domains, structured entries, accumulate across phases
- Colony-prime injection of prior review summaries into worker context
- Review agent definitions updated with Write tool + findings instructions
- Agent mirrors synced (Claude, OpenCode, Codex)
- Status/seal/entomb integration for ledger lifecycle

### Design Context

#### Part A: Persist Continue-Review Worker Findings to Disk

When `/ant-continue` runs, it spawns 4 review workers (Watcher, Gatekeeper, Auditor, Probe) that produce detailed findings — security scores, quality ratings, coverage gaps, weak spots, architectural mismatches. These are returned as structured JSON from the agents but are never written to individual files on disk. Only a one-line summary per worker survives in `review.json`. When the user `/clear`s context and starts the next build, Phase N+1 builders have no access to the review findings from Phase N's continue pass.

Build workers don't have this problem — they get per-worker `.md` files written to `worker-reports/{name}.md` via `writeCodexBuildOutcomeReports()`. Continue-review workers have no equivalent.

**Changes required:**
1. Extend `codexContinueWorkerFlowStep` struct with `Blockers []string`, `Duration float64`, `Report string` fields
2. Add `Report` field to `codexContinueExternalDispatch` struct
3. Preserve new fields in `mergeExternalContinueResults()`
4. Add `writeCodexContinueOutcomeReports()` and `renderCodexContinueWorkerOutcomeReport()` functions in `codex_continue_finalize.go`
5. Call report writer in `runCodexContinueFinalize()` after `review.json` is written
6. Update wrapper completion packet instructions in `.claude/commands/ant/continue.md` and `.opencode/commands/ant/continue.md` to include `report` field
7. Add `strconv` import to `codex_continue_finalize.go`

**Key files:** `cmd/codex_continue.go`, `cmd/codex_continue_plan.go`, `cmd/codex_continue_finalize.go`, `.claude/commands/ant/continue.md`, `.opencode/commands/ant/continue.md`

#### Part B: Review Findings Domain-Ledger

Seven read-only review agents (Gatekeeper, Auditor, Chaos, Watcher, Archaeologist, Measurer, Tracker) return rich structured JSON findings during builds and continue flows, but only a summary string gets captured in `review.json`. The detailed findings — specific files, line numbers, severity ratings, categories, suggestions — are lost when the session ends.

**Domain-ledger structure:**
```
.aether/data/reviews/
  security/ledger.json    # Gatekeeper, Auditor
  quality/ledger.json     # Auditor, Watcher
  performance/ledger.json # Measurer, Auditor
  resilience/ledger.json  # Chaos
  testing/ledger.json     # Watcher, Probe
  history/ledger.json     # Archaeologist
  bugs/ledger.json        # Tracker
```

**Agent-to-domain mapping:** Gatekeeper→security, Auditor→quality/security/performance, Chaos→resilience, Watcher→testing/quality, Archaeologist→history, Measurer→performance, Tracker→bugs

**Ledger entry format:** `{id, phase, phase_name, agent, agent_name, generated_at, status, severity, file, line, category, description, suggestion}` with deterministic IDs like `sec-2-001`, computed summary with counts by severity.

**Go runtime subcommands:** `review-ledger-write`, `review-ledger-read`, `review-ledger-summary`, `review-ledger-resolve` in new `cmd/review_ledger.go`

**Colony-prime integration:** New `prior-reviews` section injected into worker prompts showing open findings per domain (priority 8, between pheromones and instincts).

**Agent changes:** 7 agent files get Write tool + findings instructions + write-scope guardrails. Mirrors synced to `.aether/agents-claude/`, `.opencode/agents/`, `.codex/agents/`.

**Lifecycle:** Colony-scoped (accumulates across phases, archived at seal). High-value patterns promote to Hive Brain via existing instinct pipeline. Not cross-colony — findings contain code-specific file paths that go stale.

## Next Move

Plan next milestone with `/gsd-new-milestone`.

## Evolution

This document evolves at phase transitions and milestone boundaries.

*Last updated: 2026-04-26 after starting v1.9 milestone*

## Explicit Deferrals

These remain promising but are not the next best move:

- pheromone markets and reputation exchange
- swarm memory beyond the current hive/wisdom path
- federation / inter-colony coordination
- self-mutating agents / evolution engine
