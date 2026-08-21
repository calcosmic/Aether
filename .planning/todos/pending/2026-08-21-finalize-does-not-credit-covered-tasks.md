---
created: 2026-08-21T00:00:00Z
title: Build finalize credits only the primary TaskID — CoveredTaskIDs never marked complete
area: cmd/codex_build_finalize.go
source: Downstream field report (Cosmic Dashboard colony, 2026-08-21) + orchestrator code check
---

## Problem

A merged dispatch (one worker covering N tasks via `CoveredTaskIDs`) is briefed on all N tasks
since Phase 189 — but on completion, the receipt side appears to credit only the primary
`TaskID`. `grep CoveredTaskIDs cmd/codex_build_finalize.go` → zero hits; the field lives only in
cmd/codex_build.go (population + brief rendering).

Field impact (verbatim from the downstream report): "The original Phase 1 build sent one worker
to do all six jobs but recorded it as having done only job #1. Five jobs ended up with no record
of who did them. The code was all there and passing; the receipt was missing. That's what blocked
the first /ant-continue."

## Fix shape

Finalize (and continue's task-completion bookkeeping) must credit every ID in
`CoveredTaskIDs`, not just `TaskID` — with a test that builds a real merged dispatch through
`coalesceSequentialDispatches` and asserts all N tasks end `completed` after finalize. This is
the receipt-side completion of Phase 189's brief-side fix; natural pairing.

Relevant to ROADMAP Phase 192's gate: "recovery failures = 0" and "hallucinated completions = 0".
