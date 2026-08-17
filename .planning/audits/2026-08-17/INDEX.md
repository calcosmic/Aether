# 2026-08-17 Audit Corpus — Index

The evidence base for the Audit Addendum (phases 186–192). Produced in one session:
a deep truth audit (seven parallel investigations), a hostile falsification review of
that audit (five refutation investigations), and the approved implementation programme.

## Where everything lives

| Artifact | Location |
|---|---|
| Truth Audit (synthesized, 11 sections) | `TRUTH-AUDIT.md` (this dir) · also published as private artifact "Aether Truth Audit" |
| Hostile Review (A–G, retractions, go/no-go) | `HOSTILE-REVIEW.md` (this dir) · also artifact "Audit Under Fire" |
| Approved implementation programme | `~/.claude/plans/the-audit-and-adversarial-declarative-bird.md` (frozen outcomes §1 = the frozen target) |
| Raw investigation reports (verbatim agent output) | `raw/` (this dir) — 12 files |
| Governance | ROADMAP.md "Audit Addendum" section; HARDENING-PLAN.md 2026-08-17 addendum |

## Raw reports

First round (truth audit):
- `raw/01-golden-path.md` — the real lifecycle trace, SOLID/FRAGILE boundaries, two-stack finding
- `raw/02-state-recovery.md` — state correctness + recovery; worktree GC data-loss finding
- `raw/03-execution-truth.md` — enforcement points, bypasses, hallucination defenses
- `raw/04-context-assembly.md` — worker prompt composition, four context holes
- `raw/05-learning-queen.md` — liveness of learning stack + Queen intelligence; zero-reader configs
- `raw/06-gsd-analysis.md` — what makes GSD dependable, including its non-guarantees
- `raw/07-roadmap-tests.md` — planning-vs-reality divergence, test/proof infrastructure

Second round (hostile falsification):
- `raw/08-refute-defects.md` — defect list re-derived; 2 claims knocked down, 1 worsened
- `raw/09-refute-liveness.md` — liveness re-derived; policy-YAML dev-only finding, dead bridge
- `raw/10-refute-process.md` — v1.0.56 retraction, third adverse datapoint
- `raw/11-refute-gsd-comparison.md` — two verdicts reversed, measured line counts
- `raw/12-authority-context-map.md` — split-brain map, eager/lazy context flow, duplications

## Reading order for a fresh session

1. `HOSTILE-REVIEW.md` (the corrected, surviving conclusions — start here)
2. `TRUTH-AUDIT.md` (fuller narrative; carries a correction notice)
3. `raw/` as needed for file:line evidence on any specific claim

Caveat: raw reports 01–07 predate the hostile review; where they conflict with
`HOSTILE-REVIEW.md`, the hostile review wins (its §B lists every retraction).
