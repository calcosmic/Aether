---
name: frontend-ui-audit
description: Use when implemented frontend work needs a visual, accessibility, responsiveness, performance, and interaction audit
type: colony
domains: [ui, review, quality, frontend]
agent_roles: [watcher, auditor, includer, architect]
workflow_triggers: [continue]
task_keywords: [ui, frontend, visual, accessibility, responsive, interaction]
priority: normal
version: "1.0"
---

# Frontend UI Audit

## Purpose

A 6-pillar visual audit of implemented frontend code. Rather than just checking "does it render," this evaluates the UI across six quality dimensions, producing a structured review with specific, actionable findings. The colony builds UIs that look right, work right, and feel right.

## When to Use

- User says "review the UI" or "audit the frontend"
- After a frontend phase completes
- Before shipping visual changes
- User wants to verify design-to-implementation accuracy

## Instructions

### 1. The Six Pillars

```
PILLAR 1 -- CONSISTENCY
  - Visual consistency across components (spacing, colors, typography)
  - Pattern consistency (same interaction patterns everywhere)
  - Naming consistency (component names, prop names)
  - State handling consistency (loading, error, empty, success)

PILLAR 2 -- ACCESSIBILITY
  - Semantic HTML structure
  - ARIA labels and roles where needed
  - Keyboard navigation support
  - Color contrast ratios
  - Screen reader compatibility
  - Focus management

PILLAR 3 -- RESPONSIVENESS
  - Mobile layout (320px+)
  - Tablet layout (768px+)
  - Desktop layout (1024px+)
  - Large screen (1440px+)
  - Touch target sizes
  - Content overflow handling

PILLAR 4 -- PERFORMANCE
  - Bundle size impact
  - Render performance (unnecessary re-renders)
  - Image optimization
  - Lazy loading where appropriate
  - CSS optimization (unused styles, specificity issues)

PILLAR 5 -- AESTHETICS
  - Visual hierarchy (most important things stand out)
  - Whitespace usage (breathing room)
  - Color palette coherence
  - Typography scale and rhythm
  - Micro-interactions (hover states, transitions)

PILLAR 6 -- INTERACTION
  - Loading states (spinners, skeletons, progress)
  - Error states (clear messaging, recovery path)
  - Empty states (helpful guidance, not blank space)
  - Success states (confirmation, next steps)
  - Form validation (inline, clear, timely)
```

### 2. Audit Process

```
For each pillar:
  1. Scan component files for the relevant patterns
  2. Check against best practice standards
  3. Identify specific violations or gaps
  4. Score: PASS, WARN, or FAIL per check
  5. Provide specific fix recommendations with file:line references
```

### 3. Audit Report

```
 UI REVIEW -- Phase {N}: {phase_name}
   
   PILLAR              | Score   | Issues
   
   Consistency         |  PASS | 0 issues
   Accessibility       |  WARN | 3 issues
   Responsiveness      |  PASS | 0 issues
   Performance         |  WARN | 2 issues
   Aesthetics          |  PASS | 1 minor
   Interaction         |  FAIL | 4 issues
   
   Overall Score: {score}/100
   
   Critical Issues:
    Missing alt text on 3 images (accessibility)
    No loading state on data fetch (interaction)
    Form submit has no feedback (interaction)
   
   Warnings:
    Color contrast low on secondary text (accessibility)
    Bundle includes unused icon library (performance)
   
   Details: .aether/phases/phase-{N}/UI-REVIEW.md
```

### 4. Fix Recommendations

```
Every issue includes:
  - Severity: critical, warning, minor
  - File and location
  - What's wrong (in plain English)
  - How to fix (specific code suggestion)
  - Effort estimate
```

## Key Patterns

- **Specific, not vague**: "Button color is #999 on #f5f5f5 background (contrast 2.1:1, needs 4.5:1)" not "colors need work."
- **Every issue has a fix**: Don't flag problems without solutions.
- **Score honestly**: A project with 0 issues is suspicious, not perfect.
- **Pillar balance**: A UI that's beautiful but inaccessible gets flagged hard.

## Output Format

```
 UI | Phase {N}: {score}/100
    {pass} |  {warn} |  {fail} pillars
   Critical: {count} | Warnings: {count}
   Report: UI-REVIEW.md
```

## Examples

**Clean review:**
> "UI review: 85/100. 4 PASS, 2 WARN. 0 critical issues. Warnings: low contrast on footer links, missing focus ring on modal close button. Quick fixes."

**Failed review:**
> "UI review: 42/100. Interaction FAILS: no loading states, no error states, no form validation. Accessibility FAILS: 5 missing ARIA labels. Recommend fixing before ship."
