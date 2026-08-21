# Phase 157: TS Adapters & Oracle — Research

**Phase:** 157
**Date:** 2026-05-23
**Source:** ROADMAP.md, REQUIREMENTS.md

---

## Scope

Build platform adapter stubs and Oracle loop stub in TypeScript.

Requirements: CONTROL-07, CONTROL-08

---

## What Needs to Be Built

### 1. Platform Adapters (CONTROL-07)
**Directory:** `control-ts/src/adapters/`

The Go runtime dispatches to different platforms (Claude Code, Codex, OpenCode, MCP). The TypeScript control plane needs equivalent adapter interfaces so it can eventually dispatch workers through the same platforms.

Adapters needed:
- **Claude Code adapter** — matches `cmd/dispatch_platform_helpers.go` patterns
- **Codex adapter** — matches `cmd/codex_worker_artifacts.go` patterns
- **OpenCode adapter** — similar to Claude Code but with OpenCode-specific paths
- **MCP adapter** — stub for Model Context Protocol integration

Each adapter needs:
- `dispatch(params: DispatchParams): Promise<DispatchResult>` — send a worker task
- `preflight(): Promise<AvailabilityStatus>` — check if platform is available
- `health(): Promise<HealthStatus>` — check connection health

### 2. Oracle Loop Stub (CONTROL-08)
**Directory:** `control-ts/src/oracle/`

The Oracle loop does deep research with confidence evaluation. From Go:
- `cmd/oracle_loop.go` has `runOracleLoop()`, `buildOracleRubric()`, `buildOracleQuestions()`
- Confidence evaluation via `oracleOverallConfidence()`
- Research planning via `buildOracleWorkerConfig()`

Oracle module needs:
- `evaluateConfidence(query: string, findings: Finding[]): number` — score 0-1
- `buildResearchPlan(query: string, confidence: number): ResearchPlan` — plan next steps
- `runOracleIteration(query: string, iteration: number): Promise<OracleResult>` — single research pass

---

## Interfaces to Match

From Go dispatch patterns (cmd/dispatch_platform_helpers.go):
- `dispatchAvailabilityDiagnosticMessage` — platform availability check
- `dispatchProviderDiagnostics` — provider connection diagnostics
- `providerProbeCommand` — probe command for health checks

From Go oracle patterns (cmd/oracle_loop.go):
- `buildOracleRubric` — evaluation rubric for findings
- `buildOracleQuestions` — question generation
- `oracleOverallConfidence` — confidence scoring

---

## Design Decisions

1. **Adapters are stubs** — no actual API calls to Anthropic/OpenAI in this phase. They return mock results to prove the interface works.
2. **Oracle is a stub** — no real research loop. It simulates confidence evaluation and research planning.
3. **Interfaces are the deliverable** — the Go runtime will eventually call these through a bridge. Clean interfaces matter more than implementation depth.

---

## Testing Strategy

- `tests/adapters/` — verify each adapter has dispatch, preflight, health methods
- `tests/oracle/` — verify confidence evaluation, research planning, iteration
- Integration test: adapter + oracle + runner from Phase 156
