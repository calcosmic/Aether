# Phase 118: Integration & Parity Verification - Research

**Gathered:** 2026-05-14
**Status:** Ready for planning

## Existing Test Infrastructure

### TS Host Tests (`.aether/ts-host/test/`)
- **18 test files** covering: boundary enforcement, caste config, dashboard, event bridge, go bridge, host, lifecycle, narrator, oracle events, platform dispatcher, prompt assembler, queen, renderers, template loader, wave orchestrator, worker dispatch
- **boundary.test.ts** — Already verifies no direct writes to `.aether/data/`, `tmpdir` usage, `GO_OWNED_PATHS` coverage
- **lifecycle.test.ts** — 10 tests covering full plan→build→continue sequence, QueenOrchestrator integration, dashboard lifecycle, error handling
- **dashboard.test.ts** — 10 tests covering worker widgets, chamber map, Oracle visibility
- **queen.test.ts** — 21 tests covering workflow patterns, builder-probe lock, midden check, escalation

### Go Tests
- `go test ./cmd/...` — Oracle loop tests pass
- `go test ./pkg/events/...` — Ceremony event tests pass

### What's Already Proven
1. State safety: TS host cannot write to `.aether/data/` (runtime + build-time enforcement)
2. Lifecycle integration: plan → build → continue works end-to-end with QueenOrchestrator
3. Ceremony rendering: banners, spawn frames, stage markers, caste identity all tested
4. Dashboard: worker widgets, chamber map, Oracle phase/iteration display tested

## Gaps for Phase 118

1. **Golden workflow tests** — No explicit "classic v5.4 baseline" comparison tests. Need snapshot tests that capture expected ceremony output for a standard build/continue flow.
2. **Ceremony snapshot tests** — Templates are tested individually but not as full ceremony snapshots (banner + spawn + stage + closeout together).
3. **Cross-platform smoke tests** — Only single-platform tests exist. No verification that Claude Code, OpenCode, and Codex command wrappers stay in sync.
4. **Seal ceremony tests** — The full seal ritual (Sage, Chronicler, wisdom review, commit suggestion) is documented but not automated in the TS host.

## Phase Boundary

This phase is **verification-only** — no new features, only tests and assertions that the restored system matches expected behavior. It closes the v1.17 milestone.
