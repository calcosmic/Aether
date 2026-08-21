---
created: 2026-08-21T00:00:00Z
title: Stage/finalize deadlock — attempt "committed" while dispatches stay "planned"; both exits point at each other
area: cmd/codex_build_finalize.go + cmd/provenance.go
source: Downstream field report (Cosmic Dashboard colony, 2026-08-21) + orchestrator trace
---

## Problem (reproduced downstream; recover fixed it)

Recording build results is two steps: stage, then finalize. In the failure, staging marked the
attempt "committed" but left the workers marked "planned". Then:
- finalize refused: "build attempt X is already committed, so this different completion packet
  cannot replace it… run `aether continue`" (cmd/codex_build_finalize.go:869)
- continue refused: "continue provenance: no completed worker dispatches found — build did not
  produce verifiable results" (cmd/provenance.go:148) and pointed back at the build.
Circular; no in-band exit. `aether recover` diagnosed and cleared it correctly (good — 187's
work made recover trustworthy), but the state should be unreachable.

## Root-cause shape

An inconsistent two-store write: the attempt journal and the dispatch statuses are updated
non-atomically, so a crash/failure between them leaves attempt=committed + dispatches=planned.
This is exactly Phase 188's bug class, but 188's ratchets cover COLONY_STATE.json writes — the
attempt-journal/dispatch-status pair is a different store pair and is not covered.

## Fix shape

Either make the stage step atomic across both stores (one UpdateJSONAtomically-style commit), or
make one side self-heal: finalize's "already committed" branch should notice dispatches are still
planned and either roll the attempt back to a re-finalizable state or complete the missing half —
never emit two mutually-pointing refusals. Plus a ratchet in the 188 house style covering this
store pair. Reproducible per the field report.

Relevant to ROADMAP 192's gate: "recovery failures = 0".
