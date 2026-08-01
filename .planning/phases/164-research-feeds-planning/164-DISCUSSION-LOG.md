# Phase 164: Research Feeds Planning - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-01
**Phase:** 164-research-feeds-planning
**Areas discussed:** Queen's research decision, Re-research on replan, Confidence readout & early accept, Depth proposal ceremony

---

## Todo Cross-Reference (pre-discussion)

| Option | Description | Selected |
|--------|-------------|----------|
| Fold neither | Both matched by keyword only; preflight is 163.2's executed scope, reconcile-task is a continue bug | ✓ |
| Fold preflight todo | Likely redundant — 163.2 executed it | |
| Fold reconcile-task todo | Scope widening — continue bug, not research | |

**User's choice:** Fold neither

---

## Queen's Research Decision

**Q1: When should research decisions surface?**

| Option | Description | Selected |
|--------|-------------|----------|
| One batched proposal | One list after route drafts; approve/flip in one interaction (tick-to-approve) | ✓ |
| Decide-and-announce | Queen proceeds immediately; override only by re-running with flags | |
| Per-phase prompts | Separate confirm per phase; most interruptions | |

**Q2: What drives the needs-research call?**

| Option | Description | Selected |
|--------|-------------|----------|
| Queen judgment + runtime hints | Go computes cheap signals; Queen decides and writes the reason | ✓ |
| Pure runtime heuristics | Deterministic keyword rules; testable but templated, contra Phase 167 direction | |
| Queen judgment alone | Natural reasons, least testable | |

**Q3: What happens to a user override?**

| Option | Description | Selected |
|--------|-------------|----------|
| Recorded as a decision | Lands in existing decision/assumption model; replans re-propose | ✓ |
| Applied silently | Effect only for this run | |
| Recorded + sticky on replan | Replans default to the override | |

**Q4: Which caste runs research?**

| Option | Description | Selected |
|--------|-------------|----------|
| Scout, Oracle on escalation | Scout default; Oracle when loop stalls at deep/exhaustive | ✓ |
| Scout always | Current single-caste design | |
| Oracle always | Heavy on every plan | |
| You decide | Claude picks | |

---

## Re-research on Replan

**Q1: Replan with existing findings?**

| Option | Description | Selected |
|--------|-------------|----------|
| Always re-research | RESEARCH-04 / v5.4.0 behavior; once-per-phase within a single run unchanged | ✓ |
| Re-research, but show cost | Same plus per-phase flip-off in the batch | |
| Staleness-based | Reuse young findings — drifts back to the failure mode | |

**Q2: Old findings file?**

| Option | Description | Selected |
|--------|-------------|----------|
| Overwrite in place | One durable artifact per phase | ✓ |
| Archive then overwrite | Timestamped siblings accumulate | |
| You decide | | |

**Q3: Re-research scope on a big replan?**

| Option | Description | Selected |
|--------|-------------|----------|
| All phases the Queen flags | Batched proposal re-runs as the cost control | ✓ |
| Only changed phases | Unchanged text ≠ unchanged world | |
| Everything, no gate | No proposal step on replans | |

**Q4: Research worker fails?**

| Option | Description | Selected |
|--------|-------------|----------|
| Warn loudly, proceed | Research is enrichment, not a gate (Phase 160) | ✓ |
| Halt and ask | Flaky worker holds the plan hostage | |
| Auto-retry once, then warn | | |

---

## Confidence Readout & Early Accept

**Q1: Live readout appearance?**

| Option | Description | Selected |
|--------|-------------|----------|
| Full colony ceremony | Caste emoji + color + ant name + per-iteration confidence line | ✓ |
| One line per iteration | No identity flourish | |
| Progress summary only | Loop progress invisible | |

**Q2: How does early accept work?**

| Option | Description | Selected |
|--------|-------------|----------|
| Prompt on stall or near-target | Asked only when worth asking | ✓ |
| Ask at every iteration boundary | Click marathon on deep runs | |
| Upfront only | Flag before the run | |

**Q3: Where does the confidence number come from?**

| Option | Description | Selected |
|--------|-------------|----------|
| Evidence-scored + self-check | Runtime scores checkable evidence, blends self-assessment; reproducible | ✓ |
| Researcher self-score | Cheap model can claim 95% on thin findings | |
| You decide | | |

---

## Depth Proposal Ceremony

**Q1: How are proposals presented?**

| Option | Description | Selected |
|--------|-------------|----------|
| One proposal card | Consolidated card at plan start | |
| Staged questions | Three separate asks | |
| Announce, override by flag | Re-run to change | |

**User's choice:** Free-text — "Maybe it's best for it to offer them as multiple choice selections… for a user to have to type everything that just seems like a pain in the ass."
**Notes:** Resolved as multiple-choice selections with the Queen's recommendation pre-marked and a plain-English reason per knob; the plan flow gets exactly two tap-to-approve moments (research batch + depth card). Confirmed with user before continuing.

**Q2: Fast preset vs RESEARCH-08's 80%/4?**

| Option | Description | Selected |
|--------|-------------|----------|
| Queen leans skip on fast | Batch defaults to skip; flipped-on phases run 80%/4 | ✓ |
| Fast researches at 80%/4 | Literal reading; slows fast | |
| Fast always skips | Contradicts the requirement table | |

**Q3: Autopilot behavior?**

| Option | Description | Selected |
|--------|-------------|----------|
| Auto-accept, record, log | Autonomy stays autonomous; nothing silent | ✓ |
| Smart-pause on proposals | Babysitting session | |
| Pause only on re-research | One special case to remember | |

---

## Mid-Discussion Context

The user sent a north-star message mid-discussion (captured verbatim in
CONTEXT.md `<specifics>`): the milestone's purpose is getting back all the
developed functionality — charter, colonize, Oracle's RALF-style loop,
networked colony memory, pheromone proposals, emojis and caste colors — as a
working framework. This phase restores the iterate-until-confident research
behavior within that arc.

## Claude's Discretion

- Survey-context plumbing into research brief and planner context (RESEARCH-05 mechanics)
- Evidence-scoring formula internals (reproducibility rule binds)
- Runtime-hint computation for the Queen's decision
- Oracle escalation threshold and brief shape
- Go/TS boundary for depth→target/iteration binding
- Ceremony wording, proposal card copy, log-line formats
- Whether the six-section research file format changes to support scoring

## Deferred Ideas

- ts-host preflight timeout todo — remains with Phase 163.2 (executed)
- continue-finalize --reconcile-task evidence-gate todo — remains pending for a future phase
