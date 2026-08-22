---
created: 2026-08-21T00:00:00Z
title: Work done outside the dispatch machinery has no honest re-entry path — the only exit is fabricating worker receipts, which the system rightly refuses
area: verification/reconciliation ceremony
source: Third downstream repo report, 2026-08-21 (photo-scoring colony)
audit_acknowledged:
  milestone: v1.26
  at: 2026-08-22
---

## The situation (field-verified)

Owner + AI did real work together BY HAND, outside the build machinery — rewrote files the colony
had last verified in an earlier state ("awaiting photo intake"). All work is on disk and good. The
ledger cannot be brought to agree with reality:

- Closing the open attempt would require filing results for nine workers that never ran,
  describing verification that never happened — fabrication, which the session correctly refused
  (the honesty rules held; this is the system's values working).

- "The runtime correctly refuses to sign off on evidence it didn't produce, and it has no cheap
  way to re-verify." Four attempts, then a justified stop: "It's costing more than it returns."

Credit where due, from the same report: the ceremony caught a fail-open .gitignore, four claims
without evidence, and three botched hand-edits. The gates work. They lack an honest exit.

## The cross-report pattern (three repos, one day)

1. Cosmic Dashboard + repo two: the ORCHESTRATOR bundled N tasks into one worker → the completion
   packet cannot express it → planned-forever dispatches → deadlock
   ([[2026-08-21-completion-packet-cannot-express-bundled-work]],
   [[2026-08-21-stage-finalize-deadlock]]).

2. This repo: work arrived OUTSIDE the pipeline entirely (owner pair-work) → no re-entry.

Common root: the ledger accepts evidence only from its own dispatch pipeline. Reality that arrives
any other way — bundled worker, hand edit, pair session — cannot be recorded honestly, and the
honesty rules then (correctly) block the dishonest routes. Dead end by design collision.

## Fix shape

An explicitly-named, owner-invoked out-of-band reconciliation ceremony: re-verify CURRENT disk
state against the phase's own success criteria, fresh (spawn real verification against what
exists, not against worker receipts), and record the result with `verified-out-of-band`
provenance — never synthetic worker results. The attempt closes citing that verification. The
gates keep their teeth; honesty gets a door.

## Priority

Three independent repos in one day dead-ended on this family. Direct hit on Phase 192's gates
("recovery failures = 0", "unscripted interventions ≤ GSD's"). Recommend folding this + the
bundling expressibility + the checker wrong-field into one pre-showdown field-hardening phase,
alongside the deadlock fix already in flight (formica session).
