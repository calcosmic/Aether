---
created: 2026-08-21T00:00:00Z
title: One worker doing N tasks cannot be honestly reported — the completion packet allows exactly one task per entry, and every escape route loops
area: cmd/codex_build_finalize.go + completion packet schema + wrapper instructions
source: TWO independent downstream repos (Cosmic Dashboard 2026-08-21; second repo same day)
---

## The reconciled diagnosis

Three findings that looked contradictory are one story:

1. ba5cfe23 (formica session) proved finalize DOES credit CoveredTaskIDs end-to-end — when the
   RUNTIME did the coalescing, the manifest carries the covered list, and the chain expands. True.
2. Both field repos still ended with one worker's six/five tasks credited as ONE. Also true.
3. The bridge: in the field failures the WRAPPER (the orchestrating AI) bundled N dispatches into
   one worker on its own — the manifest still lists N separate dispatches with no covered-chain.
   The worker's result can then only name ONE task: the completion packet format "allows exactly
   one task per worker entry and refuses extra fields" (tested directly downstream — strict
   decoding rejects additions). The other N-1 dispatches stay "planned" → the provenance gate
   ("no completed worker dispatches") and the stage/finalize deadlock follow directly.

## Escape routes tested downstream — all loop

Refiling the report; re-dispatching a builder (the re-dispatch RE-BUNDLED all four tasks into one
again — so the wrapper's bundling tendency is systematic, not a one-off); automatic reconciliation;
skipping the reviewer step; manual reconcile flags. Reconciliation "landed, but without per-file
fingerprints, which is the last thing the gate wants" — the reconcile flow cannot supply per-file
fingerprints post-hoc, so it cannot satisfy the gate it exists to satisfy.

## Fix directions (probably all three)

a. Let honesty be expressible: completion packet accepts a covered/extra-tasks field (validated
   against the manifest's dispatch list), so a bundled worker can file truthfully.
b. Detect the mismatch at finalize: results present for 1 of N dispatches + identical
   changed-files fingerprints → name the situation and offer the repair in-band, instead of the
   provenance refusal.
c. Make the reconcile flow able to complete its own gate (per-file fingerprints computable from
   the working tree at reconcile time — the files exist and are byte-verified; the gate refuses
   only for missing bookkeeping).

## Priority

Two independent repos, same day, same terminal state, five dead escape routes each. This is the
#1 real-world failure mode observed in the field. Direct hit on Phase 192's gates ("recovery
failures = 0", "unscripted interventions ≤ GSD's"). The stage/finalize deadlock fix in flight
(formica session) removes the WORST symptom; this removes the cause.
