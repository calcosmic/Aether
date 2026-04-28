---
name: frontend-design-contract
description: Use when frontend work needs an explicit UI, interaction, accessibility, and responsive design contract before implementation
type: colony
domains: [frontend, mobile, design]
agent_roles: [architect, builder, scout]
workflow_triggers: [plan, build]
task_keywords: [ui, frontend, component, screen, layout, responsive, accessibility]
priority: normal
version: "1.0"
---

# Frontend Design Contract

## Purpose

Generates a structured UI design contract (UI-SPEC.md) for frontend phases, ensuring all visual, interaction, and accessibility requirements are locked before code is written. Prevents UI inconsistencies by defining component inventories, visual tokens, responsive behavior, accessibility contracts, and performance budgets as a single source of truth that builders reference during implementation.

## When to Use

- The phase involves building new UI components, pages, or screens
- The phase modifies significant portions of existing UI
- A new design system or pattern library is being introduced
- The phase crosses a user-facing milestone boundary
- User says "design the UI contract" or "spec the frontend"

## Instructions

Generate a `UI-SPEC.md` file in the phase directory with these sections:

### 1. Overview

- Phase goal in user-facing terms
- Target platforms (web, iOS, Android, desktop)
- Design system or component library in use (Material, Ant Design, custom)
- Breakpoints and responsive strategy

### 2. Component Inventory

List every UI component to be built or modified:

```
| Component | Type | Stateful | Interactions | Accessibility Notes |
|-----------|------|----------|--------------|---------------------|
| SearchBar | Input | Yes | Type, clear, submit | aria-label, live results |
```

### 3. Screen Layouts

For each screen or page:
- Layout wireframe description (grid/flex structure)
- Navigation flow: entry points and exit points
- Loading states: skeleton, spinner, progressive loading
- Empty states: first-use, no-results, error
- Error states: inline validation, full-page, retry

### 4. Interaction Specifications

- Click/tap targets: minimum 44x44px touch targets on mobile
- Hover, focus, active, disabled states for interactive elements
- Animations and transitions: duration (150-300ms), easing (ease-in-out)
- Gesture support: swipe, pinch, long-press where applicable
- Keyboard navigation: tab order, shortcuts, focus trap patterns

### 5. Visual Tokens

- Color palette: primary, secondary, semantic (success, warning, error), neutral scale
- Typography: heading scale, body text, monospace, line-height ratios
- Spacing scale: base unit (4px or 8px), component padding, layout margins
- Elevation/shadow system: levels and when to apply each
- Icon system: library, sizes, usage rules

### 6. Responsive Behavior

- Breakpoint definitions with layout changes at each
- Component behavior across sizes: stack, hide, collapse, or adapt
- Image strategy: srcset, art direction, lazy loading thresholds
- Navigation pattern per breakpoint: sidebar, hamburger, tab bar

### 7. Accessibility Contract

- WCAG 2.2 AA compliance target for all components
- Color contrast ratios per text size
- Screen reader announcement strategy for dynamic content
- Focus management plan for modals, navigation, and route changes
- Motion sensitivity: respect `prefers-reduced-motion`

### 8. Performance Budget

- Largest Contentful Paint target
- Cumulative Layout Shift target
- Interaction to Next Paint target
- Maximum bundle size per page
- Image optimization strategy (format, compression, sizing)

## Generation Process

1. **Gather context:** Read the phase plan, existing components, design system docs, and user stories
2. **Inventory components:** List every visual element the phase touches or creates
3. **Specify interactions:** Define how each component behaves for all user actions
4. **Define states:** Document every state each component can be in
5. **Set budgets:** Establish measurable performance and accessibility targets
6. **Review against existing:** Verify consistency with the project's existing design patterns

## Key Patterns

- **Component inventory before code**: Listing every component forces scope clarity and prevents "discovered" UI work mid-phase.
- **Visual tokens as constants**: Colors, spacing, and typography defined once in the contract become the source of truth for implementation.
- **Accessibility as contract**: WCAG compliance targets are explicit and testable, not aspirational.
- **Performance budgets are measurable**: LCP, CLS, and INP targets have specific numbers, not "fast enough."
- **State completeness**: Every component has all states documented (loading, error, empty, success) before implementation starts.

## Output Format

```
 UI-SPEC | Phase {N}: {name}
    Components: {count} to build/modify
    Breakpoints: {list}
    Accessibility: WCAG 2.2 AA
    Performance: LCP <{target}s, CLS <{target}
    Contract: .aether/phases/{N}/UI-SPEC.md
```

## Examples

**Dashboard phase:**
> "Generated UI-SPEC.md for phase 3 (Dashboard). 14 components inventoried including ChartWidget, MetricCard, and DateRangePicker. Responsive breakpoints: mobile (375px), tablet (768px), desktop (1024px). Accessibility: WCAG 2.2 AA with keyboard navigation for all chart interactions. Performance budget: LCP < 2.5s, CLS < 0.1."

**Form-heavy phase:**
> "Generated UI-SPEC.md for phase 5 (User Settings). 8 form components with full interaction specs including validation timing (inline on blur), error message positioning, and mobile keyboard handling. All components meet 44x44px touch target minimums."
