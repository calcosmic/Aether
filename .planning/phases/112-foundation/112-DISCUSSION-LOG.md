# Phase 112: Foundation - Discussion Log

**Date:** 2026-05-13
**Participants:** User (visionary), Claude (builder)
**Mode:** User deferred all decisions to Claude's discretion

## Gray Areas Identified

1. **Event delivery mechanism** — JSONL tail vs WebSocket vs subprocess pipe
2. **Ceremony config format** — YAML with parser dep vs inline JSON
3. **Node engine** — Bump to >=20 or stay >=18 with older packages

## Decisions

### Event Delivery: JSONL tail (D-01, D-02)
- **Rationale:** No background server needed, respects boundary contract, works within existing wrapper→Go→TS call chain. Go already writes JSONL. WebSocket deferred to v1.18+. Subprocess pipe rejected for tight coupling.
- **Source:** Architecture research (.planning/research/ARCHITECTURE.md)

### Ceremony Config: YAML at `.aether/config/ceremony.yaml` (D-03, D-04)
- **Rationale:** Human-editable is the core goal of v1.17. `js-yaml` (~100KB) is a small cost. Config includes caste maps, colors, labels, stage separators, banner templates, excavation phrases.
- **Source:** Feature research (.planning/research/FEATURES-V117.md)

### Node Engine: >=20 (D-05)
- **Rationale:** Node 18 is End-of-Life. Required for `chokidar` v5 and `log-update` v8. TS host is a dev tool, not end-user runtime.
- **Source:** Stack research (.planning/research/STACK.md)

### Boundary Enforcement: Maintain existing contract (D-06)
- **Rationale:** Already enforced by `boundary-reference.ts` and `boundary_contract_test.go`. Event bridge is read-only.
- **Source:** Boundary contract, migration map research

## Deferred Ideas

- WebSocket event streaming → v1.18+
- Real-time web dashboard → Out of scope per PROJECT.md
- YAML schema validation → Nice to have, not critical

## Notes

User explicitly deferred all decisions with "you decide." Research outputs from 4 parallel agents (stack, architecture, features, pitfalls) provided sufficient evidence for confident choices.
