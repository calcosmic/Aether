---
created: 2026-08-20T00:00:00Z
title: Spec builder — a user-facing command that develops a readable specification before building
area: commands/intent-capture
resolves_phase: 200
source: Owner request, 2026-08-20 session (during phases 188/189 overnight run)
audit_acknowledged:
  milestone: v1.26
  at: 2026-08-22
---

## Idea

The owner asked whether Aether helps a user develop a *spec* for the thing they're
building (or one element of it). Today's answer is "partially": `/ant-discuss`,
`/ant-council` and `/ant-assumptions` do the thinking — clarifying questions,
recorded decisions, surfaced assumptions — but nothing hands the user a readable
specification document to review and sign off before building starts.

The decisions live in colony-internal files (pending-decisions.json,
assumptions.json, COLONY_STATE.json), written for the runtime, not for the owner.

## What the feature would be

A command (working name: `/ant-spec`, or an extension of `/ant-discuss`) that ends
the intent-capture flow by producing a plain-language spec document the owner reads
and approves:

- What this thing does
- What it deliberately does not do
- The decisions made and why (pulled from discuss/council answers)
- How we'll know it works (acceptance criteria a non-technical owner can check)

Scoped to a whole colony goal OR a single element/feature of it.

## Prior art to steal from

The GSD tooling used to build Aether itself has exactly this and it works well:
discuss-phase produces a CONTEXT.md with numbered binding decisions (D-01…),
plans carry must_haves with owner-checkable truths, and the ratio of "spec before
build" to rework has been visibly good across phases 187–189. The feature is a
user-facing port of that pattern into Aether's own command surface.

## Sequencing

Owner-approved as future work, explicitly AFTER the v1.26 hardening round and the
Phase 192 showdown gate. New user-facing scope — do not fold into 188–192.
Per the roadmap's own rule, post-192 phases need a stated hypothesis; this one's
justification is owner demand, not a benchmark number, so schedule it as v1.27+
product work rather than a 193+ remediation phase.
