# Phase 109: TypeScript Orchestration Host Prototype - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-05-12
**Phase:** 109-typescript-orchestration-host-prototype
**Areas discussed:** Runtime model, Worker dispatch, Integration seam, Scope boundary

---

## Host Runtime Model

| Option | Description | Selected |
|--------|-------------|----------|
| Node script invoked by Go | Go spawns `node .aether/ts-host/dist/host.js` as subprocess. Go passes manifest path as CLI arg. | ✓ |
| Standalone CLI binary | Bundle TS host into executable with tsup/pkgroll. More complex build pipeline. | |
| You decide | Let the planner pick. | |

**User's choice:** Node script invoked by Go

| Option | Description | Selected |
|--------|-------------|----------|
| Import from narrator | TS host imports and calls narrator functions for rendering. Narrator stays untouched. | ✓ |
| No ceremony in TS host | TS host doesn't render anything — Go handles all visual output. | |
| You decide | Planner figures it out. | |

**User's choice:** Import from narrator

| Option | Description | Selected |
|--------|-------------|----------|
| .aether/ts-host/ | New package with own package.json, tsconfig.json, src/. Per Phase 106 D-08. | ✓ |
| Same package as narrator | Add host as another entry point in .aether/ts/. | |

**User's choice:** .aether/ts-host/

| Option | Description | Selected |
|--------|-------------|----------|
| Prototype only | Internal prototype to validate boundary. Not shipped, not installed. | ✓ |
| Shipped component | Distribute as part of the package. | |

**User's choice:** Prototype only
**Notes:** User asked "is this just for us testing this?" — confirmed yes, this is the prototype to prove the hybrid architecture works.

---

## Worker Dispatch Mechanism

| Option | Description | Selected |
|--------|-------------|----------|
| Shell exec of platform CLI | TS host spawns claude/opencode CLI as subprocesses per Go manifest. | |
| Delegate to Go subprocess | TS writes worker prompts to files, Go invokes them. | |
| You decide | Let the planner decide. | ✓ |

**User's choice:** You decide (Claude's discretion)

| Option | Description | Selected |
|--------|-------------|----------|
| Real spawn-log tracking | Record spawns via `aether spawn-log-write` / `spawn-complete-write`. Satisfies HOST-06. | ✓ |
| Dry-run logging only | Log to stdout only. Simpler but doesn't prove HOST-06. | |

**User's choice:** Real spawn-log tracking

---

## Integration Seam Shape

| Option | Description | Selected |
|--------|-------------|----------|
| File-based manifest exchange | Go writes manifest to file, TS reads path from CLI arg. Simple, debuggable. | ✓ |
| Stdin/stdout pipe | Go pipes JSON to TS host stdin. Harder to debug. | |
| You decide | Planner decides. | |

**User's choice:** File-based manifest exchange

| Option | Description | Selected |
|--------|-------------|----------|
| JSON mode only | Use AETHER_OUTPUT_MODE=json exclusively. Satisfies anti-pattern #2. | ✓ |
| Both JSON and visual | Read both for richer context. Violates anti-pattern #2. | |

**User's choice:** JSON mode only

---

## Scope Boundary for Prototype

| Option | Description | Selected |
|--------|-------------|----------|
| Full lifecycle for 1 phase | Prove plan→build→continue end-to-end. Golden tests verify behavior. | ✓ |
| Manifest/finalizer handshake only | Just prove the boundary works. No actual worker spawning. | |
| You decide | Planner decides. | |

**User's choice:** Full lifecycle for 1 phase

| Option | Description | Selected |
|--------|-------------|----------|
| Golden test pass/fail | Reuse Phase 108 golden tests against TS host output. | |
| New integration tests only | Write new tests specific to the TS host. | |
| You decide | Planner decides what proves the boundary best. | ✓ |

**User's choice:** You decide (Claude's discretion)

---

## Claude's Discretion

- Worker dispatch implementation (exact spawn mechanism, error handling) — user said "you decide"
- Test strategy for verifying prototype (golden test reuse vs new tests) — user said "you decide"

## Deferred Ideas

None — discussion stayed within phase scope
