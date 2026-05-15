# Wrapper-Host Contract

**Status:** Accepted  
**Date:** 2026-05-15  
**Decision:** Wrappers are **host-assisted orchestrators**, not thin pass-throughs.

---

## Context

v1.19 cut over wrappers to use the TypeScript host (`aether host <workflow>`) as the primary path for manifest generation. v1.20 hardened the host surface so all documented commands and flags work reliably.

The open question: should wrappers become thin pass-throughs (just call `aether host` and render output), or do they retain orchestration responsibilities?

## Decision

**Wrappers are host-assisted orchestrators.**

- Wrappers call `aether host` for **manifest generation** (`--plan-only`)
- Wrappers handle **colony ceremony**: worker spawning, wave management, closeout rendering
- Wrappers do **not** duplicate verification, gating, or state mutation logic owned by the Go runtime
- The TS host is the **sole entry point** between wrappers and Go CLI for manifest generation

## Boundary Rules

| Layer | May | Must Not |
|-------|-----|----------|
| **Wrappers** | Call `aether host`, spawn workers, render ceremony, add colony framing/narration | Call Go CLI directly, duplicate verification/gating, mutate colony state, parse visual output as authoritative |
| **TS Host** | Parse flags, call Go CLI via JSON, render dashboards, manage event streams | Write to `.aether/data/` directly, duplicate Go-owned logic |
| **Go CLI** | Own all state mutations, verification, gating, finalizers, canonical artifact writes | Spawn platform agents (Claude/OpenCode/Codex workers) |

## Rationale

1. **Thin pass-throughs would lose colony ceremony.** The interactive worker spawning, wave banners, and closeout rendering are core to the Aether user experience. Moving all of that into the TS host would be a massive v1.21+ effort, not v1.20 scope.

2. **The host is reliable.** Phase 130 proved all 7 subcommands work end-to-end with correct flags. The fallback direct Go CLI calls in wrappers are now dead code.

3. **Verification stays in Go.** Wrappers do not reimplement gate logic — they call `aether *-finalize` and let the runtime decide advancement.

4. **Single entry point.** Wrappers call ONLY `aether host` for manifest generation. No direct `aether plan`, `aether build`, etc. from wrappers.

## Migration Notes

- **Remove fallback paths:** Wrappers previously fell back to direct Go CLI calls when the TS host was "unavailable." Since the host is now stable, these fallbacks are dead code and should be removed.
- **Future direction:** As the TS host gains orchestration capabilities (worker dispatch, ceremony rendering), wrappers can become thinner over time. This ADR should be revisited in v1.21+.

## Consequences

- Wrappers remain moderately complex (they handle ceremony)
- But they are simpler than before (no fallback paths, no direct Go CLI calls)
- The boundary is explicit: wrappers orchestrate UX, host generates manifests, Go owns truth
