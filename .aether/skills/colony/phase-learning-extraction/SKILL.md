---
name: phase-learning-extraction
description: Use when completed work should produce reusable decisions, lessons, surprises, and patterns for future phases
type: colony
domains: [knowledge-management, retrospective, continuous-improvement]
agent_roles: [keeper, chronicler]
workflow_triggers: [continue, seal]
task_keywords: [learning, lessons, retrospective, pattern, decision]
priority: normal
version: "1.0"
---

# Phase Learning Extraction

## Purpose

Extract structured decisions, lessons, patterns, and surprises from completed phase artifacts. Produces a LEARNINGS.md file that feeds the colony's knowledge base for future phases and milestones.

## When to Use

- A phase completes (`aether continue` runs successfully)
- A milestone finishes and you want to capture cross-phase insights
- User asks "what did we learn?", "capture learnings", or "retrospective"
- Before archiving a milestone to ensure knowledge is preserved
- When starting a new phase and wanting to review relevant past learnings

## Instructions

### Extract from Phase

1. Read phase artifacts in order:
   - `.aether/phases/{N}/PLAN.md` -- original plan and intent
   - `.aether/phases/{N}/IMPLEMENTATION.md` -- what was actually built
   - `.aether/phases/{N}/TEST_RESULTS.md` -- test outcomes (if exists)
   - `.aether/phases/{N}/REVIEW.md` -- code review findings (if exists)
   - `.aether/data/phase-manifest.json` -- commit references and timing

2. Extract four categories:

   **Decisions** -- Explicit choices made during implementation
   - Format: `{decision}` | {rationale} | {alternatives considered}
   - Source: IMPLEMENTATION.md decision blocks, commit messages, conversation

   **Lessons** -- Things that went wrong or could be improved
   - Format: `{lesson}` | {context} | {action for next time}
   - Source: Test failures, review findings, late-stage changes

   **Patterns** -- Reusable approaches that worked well
   - Format: `{pattern}` | {where used} | {when to reuse}
   - Source: Code structure, test approaches, workflow steps

   **Surprises** -- Unexpected outcomes, good or bad
   - Format: `{surprise}` | {impact} | {mitigation if negative}
   - Source: Performance results, integration issues, scope changes

3. Write `.aether/phases/{N}/LEARNINGS.md`:
   ```markdown
   # Learnings -- Phase {N}: {name}

   ## Decisions ({count})
   | Decision | Rationale | Alternatives |
   |----------|-----------|--------------|
   | {decision} | {why} | {what else was considered} |

   ## Lessons ({count})
   | Lesson | Context | Next Time |
   |--------|---------|-----------|
   | {lesson} | {situation} | {improvement} |

   ## Patterns ({count})
   | Pattern | Used In | Reuse When |
   |---------|---------|------------|
   | {pattern} | {files/areas} | {conditions} |

   ## Surprises ({count})
   | Surprise | Impact | Resolution |
   |----------|--------|------------|
   | {surprise} | {effect} | {how handled} |
   ```

4. Append summary entries to `.aether/data/colony-knowledge.jsonl` for cross-phase querying

### Extract from Milestone

1. Collect all phase LEARNINGS.md files within the milestone
2. Cross-reference to find recurring patterns, repeated lessons, and milestone-wide decisions
3. Produce `.aether/milestones/{N}/LEARNINGS.md` with aggregated insights plus a "Milestone Themes" section highlighting the top 3-5 cross-cutting themes

## Key Patterns

- **Specific over vague**: "Use connection pooling for database queries under load" beats "Database was slow"
- **Actionable**: Every lesson must include what to do differently next time
- **Taggable**: Each entry gets tags from the colony taxonomy (performance, security, ux, architecture, testing, devops)
- **Deduplication**: When extracting from milestone, merge similar entries across phases rather than listing each occurrence

## Output Format

```
Extracted learnings from Phase 3 (Dashboard UI):
  7 decisions | 3 lessons | 4 patterns | 2 surprises
  Written to .aether/phases/3/LEARNINGS.md
  Colony knowledge base updated (47 total entries)
```

## Examples

```
# Extract from completed phase
> learning-extractor --phase 3

# Extract from entire milestone
> learning-extractor --milestone 1

# Review specific category
> learning-extractor --phase 5 --category lessons
```
