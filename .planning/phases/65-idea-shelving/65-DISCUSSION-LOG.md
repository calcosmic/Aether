# Phase 65: Idea Shelving - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-27
**Phase:** 65-idea-shelving
**Areas discussed:** What gets auto-shelved at seal, How ideas surface at init, Shelf data model, Entomb preservation

---

## What gets auto-shelved at seal

| Option | Description | Selected |
|--------|-------------|----------|
| Low-confidence instincts only | Instincts 0.5-0.8 get shelved | |
| Low-confidence + unaddressed pheromones | Add expired FOCUS pheromones | |
| Everything above + dreams/notes | Also scan user notes | |
| You decide | | |

**User's choice:** Custom framing — shelving is "ideas and to-do's that should be deferred to a future colony because they'd mess up current development scope."

**Follow-up — how items get onto the shelf:**

| Option | Description | Selected |
|--------|-------------|----------|
| Auto-detect at seal | Seal scans and shelves automatically | |
| Explicit user shelving | User runs `/ant-shelve` mid-colony | |
| Both | Explicit + automatic safety net | ✓ |
| You decide | | |

**Follow-up — threshold for auto-detection:**

| Option | Description | Selected |
|--------|-------------|----------|
| Shelf everything | No minimum bar | |
| Moderate bar | Reinforcement > 1, confidence >= 0.5, explicit language | |
| High bar | Confidence >= 0.5 + 2+ reinforcements | |
| You decide | | |

**User's choice:** Custom — "it should prompt the user to tick the boxes of the things to shelf" (checkbox prompt at seal, user selects which detected candidates to actually shelf)

**Follow-up — recurring REDIRECTs:**

| Option | Description | Selected |
|--------|-------------|----------|
| At seal, same prompt | Included in checkbox prompt | ✓ |
| Automatic, no prompt | Auto-shelve without asking | |
| Separate review step | Dedicated REDIRECT review | |
| You decide | | |

---

## How ideas surface at init

| Option | Description | Selected |
|--------|-------------|----------|
| Show everything | No filtering, user picks from full backlog | ✓ |
| Filter by repo type | Match domain tags | |
| Filter by keywords | Scan colony goal for matches | |
| Both domain + keyword | Combined relevance | |

**Follow-up — interaction model:**

| Option | Description | Selected |
|--------|-------------|----------|
| Tick-to-approve | Same as pheromone suggestions | |
| Numbered list with promote/defer | Per-item control | ✓ |
| Brief list with promote-all/skip | Fast path | |
| You decide | | |

**Follow-up — what promoted ideas become:**

| Option | Description | Selected |
|--------|-------------|----------|
| A FOCUS pheromone | Steers worker attention | |
| A colony goal or sub-task | Informs planning | |
| A user preference in QUEEN.md | Long-lived guidance | |
| A REDIRECT pheromone | Hard constraint | |

**User's choice:** Custom — "Specific todos — tracked work items"

---

## Shelf data model

| Option | Description | Selected |
|--------|-------------|----------|
| Use proposed model | text, source, created_at, category, confidence, tags, promoted_to, status | ✓ |
| Add priority field | high/medium/low | |
| Add tags and trigger conditions | keywords + conditional surfacing | |
| Add user notes and related files | richer context | |

**User's choice:** "Looks good" (accepted builder's recommendation)

**Final model:** `text`, `source`, `created_at`, `category`, `confidence`, `tags`, `promoted_to`, `status`, `auto_detected`

---

## Entomb preservation

| Option | Description | Selected |
|--------|-------------|----------|
| Standalone shelf.json in chamber | Full data preserved in chamber dir | ✓ |
| Merge into chamber manifest | Single archive file | |
| Both | Full data + manifest summary | |
| You decide | | |

---

## Claude's Discretion

- Exact wording of seal shelf prompt
- Number of items shown per page if backlog is large
- Whether dismissed items appear in init (recommend: no)
- Exact todo format when promoted (standardize with existing todo system)
- Keyword extraction algorithm for auto-tagging

## Deferred Ideas

None — discussion stayed within phase scope.
