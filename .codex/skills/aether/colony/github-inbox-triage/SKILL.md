---
name: github-inbox-triage
description: Use when GitHub issues and pull requests need triage against colony priorities before planning
type: colony
domains: [github, triage, issues]
agent_roles: [scout, queen]
workflow_triggers: [plan]
task_keywords: [github, issue, pull request, triage, inbox]
priority: normal
version: "1.0"
---

# Github Inbox Triage

## Purpose

Reviews and triages all open GitHub issues and pull requests against the colony's project templates, contribution guidelines, and current priorities. Ensures the colony's inbox is organized, labeled, and actionable.

## When to Use

- User says "check the inbox" or "triage issues"
- Colony start-of-session ritual
- Before planning to incorporate community feedback
- User wants to know what's waiting in GitHub

## Instructions

### 1. Inbox Scan

```
1. Fetch all open issues via `gh issue list`
2. Fetch all open PRs via `gh pr list`
3. Read project contribution guidelines (CONTRIBUTING.md or similar)
4. Read current ROADMAP.md for active priorities
5. Load label schema from GitHub labels or colony config
```

### 2. Issue Triage

```
For each issue:
  CLASSIFY:
    - bug: Reports of broken behavior
    - feature: New capability requests
    - question: Support or clarification requests
    - maintenance: Code quality, refactoring, tooling

  PRIORITY:
    - critical: Blocking users or security issues
    - high: Affects significant user segment
    - medium: Standard priority
    - low: Nice to have

  ALIGNMENT:
    - Does it match current milestone goals?
    - Does it duplicate an existing issue?
    - Is it actionable with current context?

  LABEL: Apply appropriate labels automatically
```

### 3. PR Triage

```
For each PR:
  CHECK:
    - Does it follow contribution guidelines?
    - Are tests included?
    - Does CI pass?
    - Is it targeting the right branch?
    - Scope: Does it match a planned phase?

  REVIEW STATUS:
    - needs-review: Ready for review
    - needs-changes: Author needs to update
    - blocked: Waiting on something
    - ready-to-merge: All checks pass

  CONFLICT: Check for merge conflicts with current branch
```

### 4. Triage Report

```
 INBOX TRIAGE
   Issues: {total} open ({new} new since last triage)
   PRs: {total} open ({new} new since last triage)

   Critical Issues:
    #{N} {title} -- {classification}

   Aligned with Current Milestone:
    #{N} {title} -- maps to phase {X}
    #{N} {title} -- maps to phase {Y}

   PRs Needing Attention:
    PR#{N} {title} -- CI failing, needs changes
    PR#{N} {title} -- ready to merge

   Quick Wins (low effort, high value):
    #{N} {title} -- estimated effort: {level}

   Stale Items (>30 days no activity):
    #{N} {title} -- consider closing or bumping
```

### 5. Automated Actions

```
With --auto flag:
  - Apply labels to unlabeled issues
  - Comment on issues missing reproduction steps
  - Close duplicate issues with reference
  - Request changes on PRs failing guidelines
  - Stale items: comment warning, close after threshold
```

### 6. Integration with Colony

```
Issues aligned with milestone goals:
  -> Flag for potential phase inclusion
  -> Add to ROADMAP.md if significant enough

Bug reports with reproduction:
  -> Create emergency phase if critical
  -> Queue for next maintenance phase if not

Feature requests:
  -> Add to backlog if aligned with colony goal
  -> Politely close if out of scope
```

## Key Patterns

- **Batch, don't drip**: Triage all items at once for consistent judgment.
- **Align with goals**: Every triage decision considers the colony's current milestone.
- **Automate the easy stuff**: Labeling, stale detection, and guideline checks don't need human judgment.

## Output Format

```
 TRIAGE -- {repo}
   Issues: {open} open | PRs: {open} open
   Critical: {count} | Aligned: {count} | Quick wins: {count}
   Actions taken: {labels applied, comments posted}
   Recommended: {top 3 items to address}
```

## Examples

**Full triage:**
> "23 issues, 5 PRs scanned. 2 critical bugs found (auth bypass, data loss on save). 4 issues align with current milestone. Applied labels to 8 unlabeled issues. 3 PRs need CI fixes. Recommended: fix auth bypass immediately."

**Quick check:**
> "3 new issues since yesterday. 1 feature request aligned with phase 4. 2 support questions auto-responded. No critical items."
