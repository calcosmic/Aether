# Phase 163 Discussion Log

**Date:** 2026-07-29
**Areas discussed:** 8 (4 initial + 4 follow-on at user request)

## Initial areas

### 1. Diet first, or add first?
Options: diet-first strictly / add-and-measure / measure-only.
**User answered freeform, redefining the area:** cheap-model success was about
grounding ("agents writing the right files") via roadmaps + colonize, "not
necessarily about shortening anything"; pointed at modern code-graph approaches
for context-efficient maps. → D-01/D-02/D-03.

### 2. Worker write rules
Options: sanctioned scratch area (rec) / all writes via CLI / hybrid.
**Selected: Sanctioned scratch area.** → D-04, with folded contract bugs D-05.

### 3. What --print-brief shows
Options: summary + --full (rec) / raw only / summary only.
**Selected: Summary + full on request.** → D-06.

### 4. Honor stale requirement text?
Options: intent over letter (rec) / restore as written.
**Selected: Intent over letter.** → D-07.

## Follow-on areas (user chose "Explore more gray areas")

### 5. The cheap-model proof
Options: real task twice (rec) / scripted benchmark / both.
**User answered freeform: descope** — "Testing should come later when I use
aether to develop real repos I am using." → D-08 (CONTEXT-09 reshaped).

### 6. How charter rules reach workers
Options: rules + gate check (rec) / brief-only MUSTs / advisory.
**Selected: Rules + gate check.** → D-09.

### 7. When the codebase map refreshes
Options: auto at phase start (rec) / manual + staleness warning / manual silent.
**Selected: Manual + staleness warning** (declined the recommendation —
consistent with the user's no-silent-automation stance). → D-10.

### 8. How suggestions surface
Options: end-of-build summary (rec) / mid-build / log only.
**Selected: End-of-build summary.** → D-11.

## Notable

- Two of eight answers were freeform corrections that reshaped the phase (areas
  1 and 5) — the discussion earned its cost precisely there.
- User consistently rejects silent automation (161 descope, map refresh) while
  accepting automation that reports to them (gate checks, staleness warnings,
  end-of-build summaries).
