# Phase 71: Platform Hardening - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-28
**Phase:** 71-platform-hardening
**Areas discussed:** CLI flag mismatch fix strategy, PLAT-03 dispatch scope, PLAT-05 cross-platform testing, PLAT-01/02 audit, uncommitted work handling

---

## PLAT-01/02: OpenCode Shelf Parity

| Option | Description | Selected |
|--------|-------------|----------|
| Close as already done | Both files already have shelf sections and are identical to Claude's | |
| Keep in scope for deeper audit | Verify runtime support for shelf operations from OpenCode | ✓ |

**User's choice:** Keep in scope for deeper audit — maybe the runtime behind the markdown needs verification, not just the markdown files.
**Notes:** Scout confirmed both OpenCode init.md and entomb.md have shelf sections and are byte-identical to Claude's. The Phase 65 (idea shelving) work appears to have already addressed the markdown side.

---

## CLI Flag Mismatch Fix Strategy

### Fix Approach

| Option | Description | Selected |
|--------|-------------|----------|
| Add missing flags to Go | Add all missing flags/subcommands to Go runtime to match markdown. Markdown = intended API. | ✓ |
| Rewrite markdown to match Go | Rewrite 120+ markdown calls to use only existing Go flags | |
| Fix both sides | Add important flags to Go AND clean up markdown | |

**User's choice:** Add missing flags to Go. Markdown represents the intended API — safer than changing 120+ call sites.

### Fix Scope

| Option | Description | Selected |
|--------|-------------|----------|
| Full fix — all systems working | Fix all 120+ broken calls. Pheromones, memory, midden, spawn, activity all functional. | ✓ |
| Priority fix — critical first | Fix highest-impact systems first, defer lower-priority ones | |
| Go-only — add flags, no markdown | Just add flags to Go, don't touch markdown | |

**User's choice:** Full fix — all systems working after this phase.

### Implementation Approach

| Option | Description | Selected |
|--------|-------------|----------|
| One subcommand at a time | Add flags per subcommand, test each, commit, move on | ✓ |
| All at once | Add all flags across all subcommands, then test everything | |
| Grouped by system | Group by system (pheromone, memory, etc.), add each batch | |

**User's choice:** One subcommand at a time — safer, easier to isolate issues.

---

## PLAT-03: Platform Dispatch Scope

| Option | Description | Selected |
|--------|-------------|----------|
| Verify per-platform dispatch | Test dispatch on each platform (Claude, OpenCode, Codex) | |
| Runtime manifests only | Ensure Go runtime generates correct dispatch manifests for all agent types | ✓ |
| Drop PLAT-03 from scope | Can't fix platform-level behavior from this repo | |

**User's choice:** Runtime manifests only.
**Notes:** User clarified PLAT-03 should NOT be about Codex-specific subagent dispatch. It should cover "whatever AI you are running in a session." The Go runtime generates dispatch manifests; each platform handles its own agent routing.

---

## PLAT-05: Cross-Platform Testing

| Option | Description | Selected |
|--------|-------------|----------|
| Automated smoke test | Test each Go subcommand exits cleanly and produces expected output. Part of test suite. | ✓ |
| Manual spot-checks | Manually run commands on each platform | |
| Both automated + manual | Automated Go CLI tests + manual wrapper verification | |

**User's choice:** Automated smoke test — part of the test suite, runs on every commit.

---

## Existing Uncommitted Work

| Option | Description | Selected |
|--------|-------------|----------|
| Incorporate into phase plan | Planner inspects existing changes and includes them | ✓ |
| Plan fresh, reconcile later | Ignore uncommitted work, plan from scratch | |

**User's choice:** Incorporate into phase plan — don't discard work already in progress.
**Notes:** 20+ modified cmd/ files, 7 new files (process tracker, worker cleanup, process group handling). ~934 insertions, ~260 deletions already in working tree.

---

## Claude's Discretion

None — all decisions were explicitly made by the user.

## Deferred Ideas

None — discussion stayed within phase scope.
