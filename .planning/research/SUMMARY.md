# Research Summary: Aether Restoration

**Date:** 2026-04-20
**Sources:** STACK.md, FEATURES.md, ARCHITECTURE.md, PITFALLS.md

## Key Findings

### Stack
- **No new dependencies needed** — Go standard library is sufficient (encoding/json, text/template)
- Existing `outputWorkflow` pattern (JSON + visual) can be extended for ceremony
- YAML→MD generation pattern already works — ceremony fits into wrapper layer
- Risk: Low for scope system, medium for ceremony JSON enhancements

### Features (Old vs Current)
- Old playbooks had rich spawn rituals, caste identity, pheromone signals, archaeologist scans
- Current wrappers are thin runtime executors (~47 lines for build, ~45 for continue)
- Most ceremonial behaviors are **presentation only** — wrappers read existing runtime data
- Some need **runtime JSON enhancements** — new structured fields for ceremony context
- Nothing should be restored that reimplements verification or state mutation in wrappers

### Architecture Integration
- `outputWorkflow` in `cmd/codex_visuals.go` already produces JSON wrappers can consume
- `ColonyState` needs new `Scope` field (omitempty for backward compat)
- Colony-prime context builder can add ceremony section with appropriate priority
- Clear separation: runtime owns state/gating, wrappers own pacing/narration

### Pitfalls to Avoid
1. **State mutation in wrappers** — must never happen
2. **Hand-editing generated MD files** — YAML is source, MD is generated
3. **OpenCode parity breaks** — every Claude change must mirror to OpenCode
4. **Visual text scraping as truth** — use JSON mode for structured data
5. **Scope field breaking existing colonies** — use omitempty

## Implementation Order Recommendation

1. **Housekeeping** (M001) — clean baseline first
2. **Colony Scope** (M002) — small, self-contained Go change
3. **Build Ceremony** (M003) — highest visibility, biggest impact
4. **Continue Ceremony** (M004) — builds on build ceremony patterns
5. **Watch/Status** (M005) — depends on scope from M002
6. **Pheromone Visibility** (M006) — cross-cutting, additive

## Watch Out For
- 2900+ tests must keep passing at every step
- QUEEN.md has 262 duplicate lines — clean before any feature work
- 49 YAML source files untracked in git — commit in housekeeping phase
- `<!-- Generated from .yaml -->` headers in wrappers — don't hand-edit those files
