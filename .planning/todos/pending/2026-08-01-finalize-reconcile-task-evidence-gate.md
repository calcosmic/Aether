---
created: 2026-08-01T12:00:00Z
title: continue-finalize does not count --reconcile-task as recorded reconciliation for the implementation_evidence gate
area: cmd/continue
source: Phase 163.1 code-review fix pass (WR-04 strengthening), commit 8a516aa7
resolves_phase: 193
audit_acknowledged:
  milestone: v1.26
  at: 2026-08-22
---

## Problem

On the external wrapper path (`aether continue --plan-only` → `aether
continue-finalize`), the positive-evidence gate at `cmd/codex_continue.go:3061`
("verification passed but no implementation evidence or reconciliation was
recorded") still blocks advancement even when the operator supplied
`--reconcile-task` — the finalize path apparently does not register that
reconciliation as recorded evidence, while the direct `aether continue` path
does (it surfaces only the softer "manually reconciled" warning).

Discovered when WR-04's strengthened test pinned the exact blocking classes on
the finalize path: the sole blocker there is the `implementation_evidence`
gate, not the direct path's reconcile warning. Pre-existing behavior, out of
scope for the 163.1 fix pass.

## Impact

An external wrapper that legitimately reconciles a task can pass all criteria
yet remain `blocked: true` on finalize — the same class of
"direct path works, wrapper path silently differs" asymmetry that phase 163.1
existed to close (its req 5 closed this asymmetry for read-only evidence).

## Where to look

- `cmd/codex_continue.go:3061` — the implementation_evidence gate
- `cmd/codex_continue_finalize.go` — finalize path evidence assembly
- `cmd/continue_criterion_evidence_finalize_test.go` — the
  `...CriteriaPassAndDetectsTamper` test pins current (blocked) behavior and
  documents the two allowed blocking classes
