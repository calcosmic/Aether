---
schema_version: "1.0"
id: prompt-budget-trim-field-guide
kind: field-guide
category: field-guides
title: Prompt Budget And Trim Field Guide
description: "Context capsule budget system, trim order, and weighted scoring for worker prompt assembly."
output_types: [context-review, prompt-review, architecture-review]
agent_roles: [architect, builder, watcher, oracle, queen, scout]
task_types: [context, budget, trim, prompt, capsule]
task_keywords: [budget, trim, capsule, context, chars, compact, freshness, weight, colony-prime, priority, order, section]
workflow_triggers: [build, continue, oracle]
priority: high
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 4600
---

# Prompt Budget And Trim Field Guide

This guide describes how Aether assembles worker prompts within a character
budget, what gets trimmed when the budget is exceeded, and how section priority
determines what survives.

## For Beginners

Every time Aether sends a task to a worker, it builds a prompt that includes
context: the colony goal, learned patterns, user preferences, signals, and
more. There is a limit to how much text fits in a prompt. The budget system
ensures the most important information stays, while less important information
gets trimmed first. Think of it as packing a suitcase: the essentials go in
first, and nice-to-haves only come along if there is room.

## Budget Values

| Mode | Budget | When Applied |
|------|--------|--------------|
| Normal | 6,000 characters | Default for all build and continue operations |
| Compact | 3,000 characters | `--compact` flag or auto-detected when context is small |

The budget applies to the colony-prime context capsule only. Skills have their
own separate 8,000-character budget and are not affected by capsule trimming.

## Section Sources and Contents

The context capsule assembles content from these sources, each producing a
section of the prompt:

| Section | Source | What It Contains |
|---------|--------|-----------------|
| ROLLING SUMMARY | Phase history | High-level summary of phase work so far |
| PHASE LEARNINGS | Instincts, observations | Patterns learned during the current phase |
| KEY DECISIONS | Decision log | Important decisions made during the colony |
| HIVE WISDOM | `~/.aether/hive/wisdom.json` | Cross-colony patterns filtered by domain |
| PROJECT REQUIREMENTS | Colony goal, constraints | What the project needs to achieve |
| CONTEXT CAPSULE | Handoffs, phase state | Current task-specific context |
| QUEEN WISDOM (Global) | `~/.aether/QUEEN.md` | Hub-level wisdom and preferences |
| QUEEN WISDOM (Local) | `.aether/QUEEN.md` | Project-level wisdom and notes |
| PROJECT BRAIN CORE | Colony state core | Essential colony identity and status |
| USER PREFERENCES | Hub preferences section | Communication style, expertise level |
| ACTIVE SIGNALS | `pheromones.json` | FOCUS, REDIRECT, FEEDBACK signals |
| BLOCKERS | Active blockers | Current blocking issues |

## Trim Order

When the assembled context exceeds the budget, sections are trimmed in this
order. Sections listed first are trimmed first (lowest retention priority).
Sections listed last are trimmed last (highest retention priority).

| Priority | Section | Trim Behavior |
|----------|---------|---------------|
| 1 (first trimmed) | ROLLING SUMMARY | Summarized or truncated; least actionable |
| 2 | PHASE LEARNINGS | Older learnings dropped first |
| 3 | KEY DECISIONS | Less recent decisions dropped first |
| 4 | HIVE WISDOM | Lower-confidence entries dropped first |
| 5 | PROJECT REQUIREMENTS | Redundant requirements condensed |
| 6 | CONTEXT CAPSULE | Handoffs trimmed before phase state |
| 7 | QUEEN WISDOM (Global) | Hub wisdom trimmed after local |
| 8 | QUEEN WISDOM (Local) | Project wisdom preserved longer |
| 9 | PROJECT BRAIN CORE | Colony identity; trimmed last before signals |
| 10 | USER PREFERENCES | User preferences; high retention |
| 11 (last trimmed) | ACTIVE SIGNALS | Pheromones; highest retention priority |
| NEVER trimmed | BLOCKERS | Always included regardless of budget |

### Important: BLOCKERS Are Sacred

The BLOCKERS section is **never trimmed**. If blockers exist, they are always
included in the prompt, even if this means the total exceeds the budget. This
ensures workers always know about active blocking issues.

## Freshness Scoring

Within a section, content is prioritized by freshness. Newer content is
retained over older content when trimming is necessary.

### Freshness Signals

- **Timestamp.** Each learning, decision, and pheromone has a creation timestamp.
- **Access tracking.** Hive wisdom tracks last access time via `hive-read`.
- **Phase relevance.** Content from the current phase is fresher than content
  from earlier phases.

### Freshness-Weighted Trimming

When a section must be trimmed:

1. Items are sorted by freshness (newest first).
2. Items are included in order until the section fits its allocated budget.
3. Remaining items are dropped.
4. A trim log entry records what was removed.

## Relevance Weighting

Not all content within a section is equally relevant to the current worker.

### Worker Role Matching

- Builders receive stronger weighting for implementation-related content
- Watchers receive stronger weighting for verification and quality content
- Scouts receive stronger weighting for discovery and research content
- Oracle receives stronger weighting for analysis and evaluation content

### Signal Amplification

Content that overlaps with active pheromone signals receives a relevance boost.
For example, if a FOCUS signal targets "security", then security-related
learnings and decisions are weighted higher and less likely to be trimmed.

## Compact Mode

When compact mode is active (3,000 character budget):

1. ROLLING SUMMARY is dropped entirely
2. PHASE LEARNINGS is limited to the single most recent learning
3. KEY DECISIONS is limited to the single most recent decision
4. HIVE WISDOM is limited to the top entry by confidence
5. Remaining sections are proportionally reduced

Compact mode is appropriate when the worker task is small and focused, making
extensive context unnecessary.

## Practical Implications

**For Builders:** If your task seems to lack context, the budget may have
trimmed relevant sections. Check the trim log in `.aether/data/` if available.

**For Watchers:** When verifying, note that builders may have worked with
trimmed context. Consider whether missing context could explain issues.

**For Architects:** When designing new prompt sections, consider where they
fall in the trim order. Sections that are critical should be placed low in
the trim order (high retention priority).

**For Queen:** The colony-prime command handles trimming automatically. The
Queen does not need to manage budgets manually, but should understand that
compact mode provides less context to workers.
