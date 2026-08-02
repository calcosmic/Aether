# Phase 165: Core Lifecycle Commands - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-02
**Phase:** 165-core-lifecycle-commands
**Areas discussed:** Todo folding, Voice & audience, Where the JSON protocol goes, Stage structure & template, build.md ownership handshake

---

## Todo Folding

| Option | Description | Selected |
|--------|-------------|----------|
| Fold neither | Both todos are runtime behavior changes; keep in backlog | ✓ |
| Fold the reconcile-task todo | Include continue-finalize evidence-gate fix | |
| Fold the preflight todo | Include ts-host preflight timeout work | |

**User's choice:** Fold neither
**Notes:** Both stay in backlog; recorded as Reviewed Todos in CONTEXT.md deferred section.

---

## Gray Areas (Voice & audience, JSON protocol, Stage template, build.md handshake)

All four areas were selected — and then delegated in the same response.

**User's choice (verbatim):** "Above all else, we've gone over this. I just
wanted to have that rich ceremony shit that we used to have. I don't think
you need too much of my input. I think it's clear. You need to synthesize,
you know, review, reread things, spawn multiple agents if you have to to
figure this out. What is it that we need so we can get this up and running?"

**Notes:** User's core desire is restoring the rich Classic colony ceremony.
All four implementation areas delegated to Claude. In response, an
archaeology agent excavated the v5.4.0 Classic baseline and produced a
32-element ceremony inventory, ownership delta table, and 11 regression
warnings (persisted as 165-ARCHAEOLOGY.md). Decisions D-00 through D-11 in
CONTEXT.md were synthesized from that evidence and locked.

Key synthesis finding presented to no one but recorded here: the "rich
ceremony" never lived in build.md/continue.md — at v5.4.0 they were thin
playbook loaders. The ceremony text lives in .aether/docs/command-playbooks/
and must be mined and inlined, because the loader itself is now forbidden by
tests. Ceremony has been restored seven times since v5.4.0 and evaporated
each time; D-09 makes this restoration the one that ships with tests that
fail when the ceremony is absent.

## Claude's Discretion

All four gray areas (voice, protocol placement, stage template, ownership
handshake) — user delegated explicitly. Additionally: exact narration
wording, which runtime ceremony commands each wrapper invokes, plan/wave
split.

## Deferred Ideas

- Nine renderer-owned ceremony gaps (wave-failure banner, escalation banner,
  verification grid, pattern announce, graveyard caution, survey-loaded
  banner, visual checkpoint, project-complete block, resumption line) →
  Phase 168 input.
- Milestone/maturity framing in build/continue → Phase 168 decides.
- Two reviewed todos (reconcile-task evidence gate; preflight timeout) →
  backlog.
