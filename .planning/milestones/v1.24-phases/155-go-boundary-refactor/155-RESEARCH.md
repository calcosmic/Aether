# Phase 155: Go Boundary Refactor — Research

**Phase:** 155
**Date:** 2026-05-23
**Source:** 152-BEHAVIOUR_EXTRACTION_AUDIT.md, ROADMAP.md, REQUIREMENTS.md

---

## Scope

EXTRACT-07: "Go loads behaviour from files. Where Go previously contained hardcoded strings, it now loads from `colony/` and `.aether/` assets at runtime."

Phase 154 created the editable assets. Phase 155 makes Go read them.

---

## Extraction Audit Summary

| Classification | Count | Phase 155 Relevance |
|----------------|-------|---------------------|
| KEEP_IN_GO | 42 | None — pure spine |
| MOVE_TO_TS | 8 | Deferred to Phase 156+ |
| MOVE_TO_YAML | 6 | **In scope** — load from YAML |
| MOVE_TO_MARKDOWN | 14 | **In scope** — load from Markdown |
| MOVE_TO_JSON | 4 | Partial — schema already in TS |
| DELETE | 2 | Remove dead code |
| MIXED | 18 | **In scope** — extract behaviour strings |

---

## Highest-Impact Targets

### 1. Visual/Ceremony Config (`cmd/codex_visuals.go`)
**Classification:** MOVE_TO_MARKDOWN
**Target:** `.aether/config/visuals.md` or `colony/ceremony/visuals.md`

Hardcoded:
- `casteColorMap` — ANSI color codes per caste
- `casteEmojiMap` — single emoji per caste
- `casteLabelMap` — human-readable caste name
- `renderAetherWordmark`, `renderBanner`, `renderStageMarker`

These are pure behaviour — no runtime logic. Safe to extract first.

### 2. Caste Relevance (`cmd/caste_relevance.go`)
**Classification:** MOVE_TO_TS (deferred)
**Note:** Full orchestration move is Phase 156. For Phase 155, we can extract the hardcoded caste scoring tables into a YAML config that Go reads, making future TS takeover easier.

### 3. Colony Prime Context (`cmd/colony_prime_context.go`)
**Classification:** MIXED
**Target:** `colony/prompts/colony-prime.md`

Hardcoded prompt section headers and template text:
- "## Colony State"
- "## Pheromone Signals"
- "## Active Instincts"
- "## Key Decisions"
- "## Phase Learnings"
- "## HIVE WISDOM"
- etc.

The context assembly algorithm stays in Go; the section text loads from file.

### 4. Dispatch Contract (`cmd/codex_dispatch_contract.go`)
**Classification:** MIXED
**Target:** `colony/policies/dispatch-contract.yaml`

Hardcoded spawn budget rules and timeout defaults now live in `colony/policies/dispatch-contract.yaml`. Go should load them instead of hardcoding.

### 5. Review Depth (`cmd/review_depth.go`)
**Classification:** MOVE_TO_TS (deferred)
**Note:** Like caste relevance, full move is Phase 156. Phase 155 can load the keyword tables and depth rules from YAML.

### 6. Mixed Ceremony Files
Multiple files have `render*` functions that emit hardcoded visual/ceremony text:
- `cmd/ceremony_cmd.go` → `colony/ceremony/*.md`
- `cmd/ceremony_emitter.go` → `colony/ceremony/emitter.md`
- `cmd/init_ceremony.go` → `colony/ceremony/init.md`
- `cmd/medic_cmd.go` → `colony/ceremony/medic.md`
- `cmd/status.go` → `colony/ceremony/status.md`

---

## What Phase 155 Does NOT Do

- Does NOT remove Go as runtime spine
- Does NOT move orchestration logic to TypeScript (Phase 156+)
- Does NOT change CLI commands or user-facing behaviour
- Does NOT delete the original `.aether/docs/command-playbooks/` files
- Does NOT modify platform wrappers (`.claude/`, `.opencode/`, `.codex/`)

---

## Recommended Approach

1. **Wave 1:** Extract pure visual/ceremony config (MOVE_TO_MARKDOWN targets)
   - `cmd/codex_visuals.go` → load from `colony/ceremony/visuals.md`
   - Add file-loading helpers with fallback to hardcoded defaults
   - Add tests for file-loading path

2. **Wave 2:** Extract structured policy config (MOVE_TO_YAML targets)
   - `cmd/codex_dispatch_contract.go` → load from `colony/policies/dispatch-contract.yaml`
   - `cmd/review_depth.go` → load keyword tables from YAML
   - Add tests for YAML loading path

3. **Wave 3:** Extract prompt section text (MIXED targets)
   - `cmd/colony_prime_context.go` → load section templates from `colony/prompts/colony-prime.md`
   - Select MIXED ceremony files → load from `colony/ceremony/*.md`
   - Keep spine logic in Go; only move text

---

## Verification Strategy

- `go test ./...` must pass with no regressions
- `aether version` must report correctly
- Spot-check: `go test ./cmd/... -run Visuals` (new tests for file loading)
- Spot-check: `go test ./cmd/... -run DispatchContract` (new tests for YAML loading)
- Colony lifecycle smoke test: `/ant-init` + `/ant-status` still works

---

## Risk Assessment

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| File-loading path fails at runtime | Low | Always fallback to hardcoded defaults |
| Performance regression | Low | Load once at init, cache in memory |
| Test breakage from changed strings | Medium | Tests should assert structure, not exact strings |
| Mixed-file refactoring scope creep | Medium | Strict boundary: only move text, not logic |

---

## Key Files to Read Before Planning

- `cmd/codex_visuals.go` — visual config hardcoding
- `cmd/colony_prime_context.go` — prompt section text
- `cmd/codex_dispatch_contract.go` — dispatch contract rules
- `cmd/review_depth.go` — depth keyword tables
- `colony/policies/dispatch-contract.yaml` — already exists from Phase 154
- `152-BEHAVIOUR_EXTRACTION_AUDIT.md` — complete symbol list
