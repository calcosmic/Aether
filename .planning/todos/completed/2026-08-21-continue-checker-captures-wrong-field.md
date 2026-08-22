---
created: 2026-08-21T00:00:00Z
title: Continue's embedded verification worker returns the wrong subprocess field as its verdict
area: cmd/codex_continue
source: Downstream field report (Cosmic Dashboard colony, 2026-08-21)
audit_acknowledged:
  milestone: v1.26
  at: 2026-08-22
---

## Problem

`aether continue` spawns its own verification worker in-process. In the downstream colony it
failed 3 of 5 runs, and the tell is what it returned instead of a verdict: one failure's stated
reason was the literal text `/ant-continue 2>&1 | tail -100` — a shell command, not a finding.
Aether is capturing the wrong field from the sub-process it launches.

Compounding: `--skip-watchers` does not rescue it, because some checks specifically require a
checker's sign-off and "skipped" counts as "not passed" — so a broken embedded checker leaves no
in-band escape. The session's workaround (spawn the reviewers directly as external dispatches)
worked, and incidentally caught a real issue the embedded checker would have missed.

## Also fold in (same subsystem): the pause race

`aether continue` runs ~10 minutes. When a run spanned a conversation pause, the session's stop
hook paused the colony underneath it and the completed run's write was rejected as
"colony is paused" — twice. NOTE: the rejection itself is Phase 188's supersession protection
working as designed (better than clobbering the pause). The gap is UX: a completed 10-minute
verification is thrown away. Fix shape: on supersession-by-pause, preserve the completed
verification result for a re-apply after resume (or have continue take a lease the stop hook
respects). Downstream workaround that worked: chain resume+continue in one command.

## Not currently covered

Not in scope of phases 187-191; candidate for a 192-adjacent remediation slot. Relevant to the
showdown gate: "unscripted interventions ≤ GSD's" and "recovery failures = 0".

## Reproduction detail (relayed from the downstream session, 2026-08-21)

The watcher spawned inside `aether continue` failed 3 of 5 runs. The verdict field came back
holding the check's own COMMAND or DESCRIPTION strings rather than a result:

- one failure's stated reason: literally `/ant-continue 2>&1 | tail -100`
- two "successful" runs reported: "Check exit code of plan-only continue run" and
  "Preview what /ant-continue would do without mutating state"

Diagnosis this points at: the code populating the verdict/reason maps the wrong field from the
check definition (its `description`/`command`) instead of the check's actual output/result.
Confirmed compounding factor: `--skip-watchers` does not route around it because bound criteria
listing `watcher` in `required_checks` treat "skipped" as not-passed — the phase stays blocked
either way.

---

**Closed 2026-08-22:** resolved by Phase 191.1 (verification passed 6/6 on 2026-08-21) or disproven/fixed as recorded above; filed at the v1.26 close.
