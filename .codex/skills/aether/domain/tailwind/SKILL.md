---
name: tailwind
description: Use when the project uses Tailwind CSS for utility-first styling
type: domain
domains: [css, frontend, styling]
agent_roles: [builder]
detect_files: ["tailwind.config.*"]
detect_packages: ["tailwindcss"]
priority: normal
version: "1.0"
---

# Tailwind CSS Best Practices

## Utility-First Approach

Apply styles directly with utility classes rather than writing custom CSS. Resist the urge to create `.btn` or `.card` classes -- use component abstractions in your framework instead (React components, Vue components, partials). Custom CSS should be the exception, not the default.

## Class Organization

Order classes consistently: layout (flex, grid, position) first, then sizing (w, h, p, m), then typography (text, font), then visual (bg, border, shadow), then states (hover, focus). Use Prettier with `prettier-plugin-tailwindcss` to automate class sorting.

When a class list gets long (10+ utilities), extract it into a component rather than using `@apply`. The `@apply` directive defeats the purpose of utility-first and creates hidden CSS that is harder to maintain.

## Responsive Design

Tailwind is mobile-first. Write base styles for mobile, then add responsive modifiers: `sm:`, `md:`, `lg:`, `xl:`. Never start with desktop styles and subtract.

Test responsive behavior at each breakpoint. Use the browser's device toolbar, not just resizing the window.

## Customization

Extend the theme in `tailwind.config.js` rather than using arbitrary values. If you find yourself writing `bg-[#1a2b3c]` more than once, add it to your theme colors. Arbitrary values signal missing design tokens.

Use CSS variables for dynamic values (dark mode, user themes): define variables in your CSS, reference them in the Tailwind config with `var(--color-primary)`.

## Common Gotchas

Tailwind purges unused classes in production. Dynamic class names assembled with string concatenation will be purged because the scanner cannot detect them. Always use complete class names: write `text-red-500` not `text-${color}-500`. Use the `safelist` config option for truly dynamic classes.

Avoid `!important` modifiers (`!text-red-500`) -- they indicate specificity fights that usually mean structural issues. Fix the cascade instead.

## Dark Mode

Use the `dark:` variant with the `class` strategy for manual control, or `media` strategy for OS-level preference. Be consistent across the project -- mixing strategies causes unpredictable behavior.
