---
created: 2026-10-04T00:00:00Z
title: Helpers only when they earn it — as quick as plain Claude, plus the memory
area: orchestration/performance
source: Owner, 2026-10-04, after the Finish the Track deck finished and a comparison with Gambit and Glitch Cat Club
resolves_phase: candidate for the phase after 210 (decided at the trial's close, on or after 2026-10-09)
priority: high
---

## The owner's words (dictated, 2026-10-04)

"What could we do in a later GSD phase in order to make it that Aether really is truly a flexible
system, you know, for even if you just want to do like small builds and all these sorts of things
without spawning so many helpers but like the queen is able to orchestrate when it's necessary to
and when it's not necessary to … maybe the user is able to select things through multiple choice
quite specifically like do you want helpers for this? If the queen proposes which ones to select …
and then if you are doing things like small ones it's not just burning through tokens … a lot of
people are saying with agent systems it's just markdown files and folders … I feel like maybe
Aether's gone astray."

He agreed ("sounds good") to the proposal below being recorded for the post-trial decision.

## Plain English

The Aether program itself costs nothing to run; tokens are burned by the AI helpers it sends and
by the long instructions each one reads. Keep the program (it is what catches a chat that skips
its instructions, e.g. the 2026-10-03 seal where the chat summarised the reviewer brief instead of
passing it on). Cut the AI work around each job.

## Evidence

- `.planning/phases/209-light-default-path/209-TIMING.md` (2026-09-24), against plain Claude:
  small job 36.4s / $0.44 vs 17.0s / $0.40; medium job 833.4s / $6.44 vs 23.5s / $0.41
  (35x slower, 16x dearer, and it stopped to ask a question part-way).
- Finish the Track deck (2026-10-02/03): phase 5 dispatched seven builders; phase 2 used one
  builder but ~983K tokens across 5 worker runs because the project kept getting stuck (freeze
  rows 16-18); about 6.2M tokens for the whole project.
- Memory `why-owner-stopped-using-aether` (2026-08-29): the bar is "as snappy as plain Claude on a
  small task, plus the memory"; the memory is the value, the helper ceremony is the cost.
- Related, about building Aether rather than using it:
  `2026-08-27-worker-turnaround-is-too-slow.md`.

## Already built (do not rebuild)

- One-helper fast path: a build needing one helper with nothing waiting on the owner goes straight
  through (2026-08-27).
- Reviewers forced only by five named risks; a one-task bug fix is one helper (D11).
- Team check-in card: each helper with its reason, REQUIRED/OPTIONAL, proceed / trim / redirect.
- `/ant-go` picks the quick or planned route; `/ant-quick` does one small job with one helper.

## Proposal

1. **No helpers by default** for small and medium jobs: the main chat does the work with Aether's
   memory, tracking and checks. Helpers only when the job is genuinely large or parallel.
2. **The coordinator proposes, the owner chooses — only for big jobs.** Multiple choice with a time
   and cost estimate per option (e.g. "do it myself, ~3 min, ~$1" vs "three helpers, ~6 min,
   ~$4"). No question on small jobs, to keep momentum.
3. **A cost line on every finish**, so the owner sees what each job spent.
4. **Cheaper models for routine helper work, measured first** — Gambit's approach
   (github.com/joshsymonds/gambit: a tested model per role, never escalated).
5. **Memory pushed into the main chat on every prompt** by a UserPromptSubmit hook, so retrieval
   never depends on the model searching (Glitch Cat Club's graph-memory-starter pattern,
   github.com/Glitch-Cat-Club/graph-memory-starter). Start with plain search, not a graph — its
   author says most people do not need the graph. His demo (one planted three-hop question) took
   Haiku from wrong to right; treat that as a lead, not proof.
6. **Review rules that stop phantom work** (Kem Tossoun, "Adversarial AI Coding", youtube.com/watch?v=e2nc5wvSG4s):
   one claim per pass; a fresh read-only helper, never the author; evidence and a reproduction or it
   is not a finding; one round; every finding first answers "can a real user reach this?" or is
   dropped. Fits the owner's 2026-10-04 seal ruling (list suggestions, never apply them).
7. **Done means passing the stopwatch.** The phase counts as done only when the 209-style small and
   medium job comes within an agreed margin of plain Claude in time and cost.

## Constraints

- Nothing here starts before the two-week freeze ends (2026-10-09); the trial's close chooses the
  next phase.
- Do not weaken the program's own checks to buy speed; they cost no tokens.
