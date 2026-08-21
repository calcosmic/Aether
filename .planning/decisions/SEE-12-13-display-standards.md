# SEE-12 / SEE-13 — Display and task-packet standards

**Decision date:** 2026-08-16
**Status:** settled; each claim is tied to a named test.

## SEE-12 — Progressive-disclosure display standard

Human-facing output follows the classic v5.4.0 house style:

1. **Headed sections, not machine tables.** An emoji heading with a
   plain-English parenthetical (`🎯 FOCUS (Pay attention here)`), one item per
   line, nested `└──` detail beneath. Fixed-width column headers with dash
   rules and bordered table renders are forbidden in human displays.
   Locked by `TestHumanDisplaysUseHeadedSectionsNotMachineTables`
   (invariant: catches the next flattening, whatever it is called), with a
   shrink-only allowlist for the few genuine numeric tables.
2. **Compact by default, detail nested and capped.** Per-item detail renders
   under the item, capped per category with an honest `(+N more)` overflow —
   never silently truncated, never a wall.
   Locked by `TestContinueRendersWorkerFindings`.
3. **Summaries explain themselves.** A score or status line carries its
   components (`TestStatusRendersHealthBreakdown`) or a plain-English
   parenthetical; codes and counts never stand alone.
4. **Operator moments get one distinct frame** — `renderDecisionBlock`
   (`TestDecisionBlockFramesOperatorMoments`).

## SEE-13 — Worker task-packet standard

A worker brief is written for a worker with zero prior context. Every
generated packet carries, inspectably:

- **The task itself** — the Assignment section, one action per task.
- **What done means** — task/phase success criteria.
- **How to check it** — the verification command counted as task content
  (see the rationale comment in `TestBuildWorkerBriefIsMostlyTask`).
- **Proportion invariant** — task-relevant content stays the majority of the
  packet; scaffolding may never outweigh the task
  (`TestBuildWorkerBriefIsMostlyTask`).

Conformance on a real generated artifact:
`TestPrintBriefMatchesTaskPacketStandard`.
