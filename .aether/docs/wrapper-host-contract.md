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
| **Wrappers** | Call `aether host` for host-backed flows, spawn workers, render ceremony, add colony framing/narration | Duplicate verification/gating, mutate colony state, parse visual output as authoritative, expose raw provider stdout/stderr or auth probe output, document unimplemented future host targets as implemented |
| **TS Host** | Parse flags, call Go CLI via JSON, render dashboards, manage event streams | Write to `.aether/data/` directly, duplicate Go-owned logic, invent provider/auth diagnostics |
| **Go CLI** | Own all state mutations, verification, gating, finalizers, canonical artifact writes, provider availability preflight diagnostics | Spawn platform agents (Claude/OpenCode/Codex workers) |

Current wrapper host-backed manifest surfaces are `colonize`, `plan`, `build`,
heavy/classic `continue`, and `seal`. The TS host also exposes `oracle`,
`watch`, and `swarm` display/lifecycle surfaces, but canonical swarm wrappers
still use the Go plan-only/finalizer path.

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
