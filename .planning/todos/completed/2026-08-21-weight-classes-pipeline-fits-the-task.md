---
created: 2026-08-21T00:00:00Z
title: Weight classes — the pipeline should fit the task; small work should get the small machine automatically
area: orchestration/UX
source: Owner feedback, 2026-08-21, after using Aether in downstream repos
resolves_phase: 194
audit_acknowledged:
  milestone: v1.26
  at: 2026-08-22
---

## The owner's observation (verbatim intent)

Builds feel heavy: sub-agent spawns, reviewers, long verification runs — "not always necessary.
Sometimes, with Claude, you create a plan and a spec and then you build it and then you tweak
things. And that seems almost better for certain things."

## What's true on both sides

- The heavy machinery caught 20+ real defects across phases 187-191 that every worker's
  self-check had passed. On complex/multi-part/high-stakes work it is the product.

- On small work (a button, a copy tweak, a small bug) it is disproportionate: 10-minute
  verification runs and reviewer panels for picture-hanging. The field sessions felt this.

- Partial machinery already exists and should be surfaced better TODAY: `/ant-quick`, `--light`,
  discovery-mode → light, and the 187-hardened caste pruning (Probe only where testable code
  exists, Gatekeeper only on security signals).

## The feature

A routing layer whose first question is "how big is this, really?" — task-size classification
(by scope of files/criteria/risk, not by asking the owner to know flags) that selects a lane:

- **Featherweight:** spec → build → tweak, single worker, owner in the loop, receipts kept to the
  honest minimum (changed files + verification status). No reviewer panel, no research pass.

- **Middleweight:** today's standard depth.
- **Heavyweight:** today's heavy depth (security/final/high-risk unchanged — safety floors from
  187 are never dropped by lane choice).

The Definition of Done still applies in every lane — a featherweight task still ends with a
command that fails if the requirement is unmet; what shrinks is ceremony, never honesty.

## Sequencing + evidence base

Post-showdown (v1.27), per the roadmap's own rule that extensions need a hypothesis tied to a
benchmark number. This one's hypothesis is direct: Phase 192 measures median tokens per successful
task (gate: ≤1.5× GSD's); the showdown data will show exactly where the cost lives, and this
feature targets that number. Related: [[2026-08-20-spec-builder-feature]] — the featherweight
lane's front door is likely the spec-builder's "review and approve, then build" flow.
