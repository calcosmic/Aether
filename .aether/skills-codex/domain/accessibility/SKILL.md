---
name: accessibility
description: Use when the project uses web or mobile interfaces requiring WCAG 2.2 AA compliance, ARIA patterns, or assistive technology support
type: domain
domains: [frontend, mobile, ux]
agent_roles: [builder]
detect_files: ["*.tsx", "*.jsx", "*.vue", "*.html", "*.swift", "*.kt"]
detect_packages: ["axe-core", "@axe-core/react", "eslint-plugin-jsx-a11y", "lighthouse"]
priority: normal
version: "1.0"
---

# Accessibility Best Practices (WCAG 2.2 AA)

## ARIA Patterns

- Use `role`, `aria-label`, `aria-describedby`, and `aria-expanded` to communicate widget semantics
- Apply `aria-live="polite"` for dynamic content updates (toasts, search results); `aria-live="assertive"` only for critical alerts
- Use `aria-hidden="true"` on decorative elements; never hide focusable elements with `aria-hidden`
- Implement landmark regions: `<nav>`, `<main>`, `<aside>`, `<footer>` -- avoid redundant `role` attributes on semantic elements
- Use `aria-checked`, `aria-selected`, and `aria-pressed` for toggle states in custom controls

## Keyboard Navigation

- All interactive elements must be reachable via `Tab` and activatable via `Enter` or `Space`
- Implement roving `tabindex` for composite widgets (toolbars, tab lists, menus): only one item in the tab sequence at a time
- Provide visible focus indicators: use `:focus-visible` to show outlines; never use `outline: none` without a replacement
- Support `Escape` to close modals, dropdowns, and overlays; return focus to the triggering element
- Implement skip links (`Skip to main content`) as the first focusable element on every page

## Screen Reader Testing

- Test with VoiceOver (macOS/iOS), NVDA (Windows), and TalkBack (Android) -- each has different behaviors
- Use semantic HTML as the foundation: headings (`<h1>`-`<h6>`), lists, tables, and form elements
- Provide `alt` text for informative images; use `alt=""` for decorative images
- Label form inputs with `<label for="id">` or `aria-labelledby`; use `aria-describedby` for validation messages
- Test announcements in sequence: screen readers read in DOM order, not visual order

## Color and Contrast

- Ensure minimum 4.5:1 contrast ratio for normal text, 3:1 for large text (WCAG AA)
- Never use color as the sole indicator of state -- pair with icons, patterns, or text labels
- Support dark mode and high-contrast mode: test with forced-colors media query
- Use relative color units (`rem`, `em`) and respect `prefers-color-scheme` and `prefers-contrast`
- Validate contrast with browser DevTools (CSS overview panel) or axe DevTools extension

## Focus Management

- Trap focus within open modals and dialogs: Tab and Shift+Tab cycle within the dialog boundary
- Move focus to new content after dynamic updates (route changes, modal opens, accordion expands)
- Manage focus on SPA navigation: announce page changes with `aria-live` and move focus to `<main>`
- Use `tabindex="-1"` for programmatically focusable elements that should not be in the tab sequence
- Restore focus to the trigger element when closing overlays and modals
