---
name: pull-request-shipping
description: Use when verified work is ready to become a clean pull request and release-ready branch
type: colony
domains: [shipping, pull-requests, github]
agent_roles: [builder, queen]
workflow_triggers: [seal, ship]
task_keywords: [ship, pull request, pr, merge, release]
priority: normal
version: "1.0"
---

# Pull Request Shipping

## Purpose

The complete ship workflow in one skill: create a clean PR branch filtering out colony artifacts, generate a well-structured pull request, trigger review, and prepare for merge. The colony's path from "done building" to "shipped code."

## When to Use

- User says "ship it" or "create a PR"
- All phases in a milestone are complete and verified
- User wants to merge work into the main branch
- After colony verification passes

## Instructions

### 1. Pre-Ship Checks

```
Before shipping, verify:
  1. All phases in scope are marked COMPLETE
  2. No blockers or unresolved pheromones
  3. All UAT items pass
  4. No uncommitted changes
  5. Lint and typecheck pass
  6. Tests pass (if test framework detected)
  7. No secrets in staged files
```

### 2. Branch Preparation

```
1. Create clean branch from main/master:
   git checkout main && git pull
   git checkout -b {colony-name}/{milestone-slug}

2. Cherry-pick or filter commits:
   - Include: all source code changes
   - Exclude: .aether/ directory changes (colony artifacts)
   - Exclude: colony data files
   - Include: docs/ changes if documentation was updated

3. Verify clean diff:
   - Review filtered diff for completeness
   - Ensure no colony-only files leaked in
```

### 3. PR Generation

```
Generate PR body from colony context:

Title: {milestone_name} -- {summary}

Body:
  ## Summary
  {auto-generated from ROADMAP completed phases}
  
  ## Changes
  {per-phase breakdown of what was built}
  
  ## Testing
  {test coverage summary, UAT results}
  
  ## Screenshots (if applicable)
  {UI changes before/after}
  
  ## Checklist
  - [x] All phases complete
  - [x] Tests pass
  - [x] No secrets committed
  - [x] Documentation updated
```

### 4. Review Trigger

```
1. Create PR via `gh pr create`
2. Request reviewers if configured
3. Apply labels: colony, milestone-{N}
4. Trigger CI/CD checks
5. Post colony-summary as PR comment
```

### 5. Merge Preparation

```
After review approval:
  1. Rebase on latest main (if behind)
  2. Squash commits if preferred (colony commits are granular)
  3. Verify CI passes on final branch
  4. Wait for merge approval from user
  
  Merge options:
  --squash    Squash all colony commits into one
  --rebase    Rebase colony commits onto main
  --merge     Standard merge commit
```

### 6. Post-Ship

```
After merge:
  1. Delete feature branch
  2. Tag release if applicable
  3. Update ROADMAP with merge info
  4. Emit ship-complete pheromone (strength 1.0)
  5. Seal colony if all work shipped
  6. Archive colony artifacts
```

## Key Patterns

- **Clean branches**: PR branches contain only source changes, never colony artifacts.
- **Auto-generated PRs**: PR description comes from colony context, not manual writing.
- **Ship is a ceremony**: Pre-ship checks ensure nothing broken reaches main.
- **User has final say**: Never merge without explicit user approval.

## Output Format

```
 SHIP | PR #{number}: {title}
   Branch: {branch_name}
   Commits: {count} | Files changed: {count}
   CI: {pending|passing|failing}
   Review: {requested|approved|changes-needed}
   URL: {pr_url}
```

## Examples

**Full ship:**
> "PR #47 created: 'Auth System -- Complete authentication milestone'. 23 commits, 47 files changed. CI passing. Review requested from @tech-lead. Ready for merge after approval."

**Ship with issues:**
> "Pre-ship check: 2 issues found. Test coverage at 72% (threshold: 80%). Lint error in auth.ts:42. Fix these before shipping."
