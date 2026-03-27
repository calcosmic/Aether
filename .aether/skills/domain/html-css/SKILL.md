---
name: html-css
description: Use when the project uses plain HTML, CSS, or SCSS for web pages
type: domain
domains: [frontend, styling, markup]
agent_roles: [builder]
detect_files: ["*.html", "*.css", "*.scss"]
priority: normal
version: "1.0"
---

# HTML & CSS Best Practices

## Semantic HTML

Use the right element for the job: `<nav>` for navigation, `<main>` for primary content, `<article>` for self-contained content, `<section>` for thematic groupings, `<aside>` for tangential content. Avoid div-soup -- excessive `<div>` nesting with no semantic meaning.

Use `<button>` for actions, `<a>` for navigation. Never put click handlers on `<div>` or `<span>` elements -- they are not keyboard accessible.

## Accessibility Fundamentals

Every `<img>` needs an `alt` attribute. Decorative images get `alt=""`. Every form input needs a `<label>` linked via `for`/`id`. Heading levels (`h1`-`h6`) must not skip levels -- go `h1`, `h2`, `h3`, never `h1` to `h3`.

Ensure color contrast ratios meet WCAG AA (4.5:1 for text, 3:1 for large text). Test with browser DevTools accessibility panel.

## CSS Architecture

Avoid deeply nested selectors (more than 3 levels). Flat, specific class selectors are faster and easier to override. Prefer BEM naming (`.block__element--modifier`) for vanilla CSS projects to keep specificity predictable.

Never use `!important` to fix layout problems. It signals a specificity battle -- fix the root cause by reducing selector complexity.

## Layout

Use CSS Grid for two-dimensional layouts (rows and columns simultaneously). Use Flexbox for one-dimensional layouts (a row or a column). Avoid floats for layout -- they exist for wrapping text around images, not for page structure.

Set `box-sizing: border-box` globally. Without it, padding and borders add to element dimensions, making math unpredictable.

## Responsive Design

Use relative units (`rem`, `em`, `%`, `vw`, `vh`) over fixed pixels for sizing and spacing. Set font sizes in `rem` so they respect user browser settings.

Write mobile-first media queries: base styles for small screens, `min-width` breakpoints for larger ones.

## Performance

Minimize CSS file count and size. Remove unused selectors. Avoid expensive properties in animations: stick to `transform` and `opacity` for smooth 60fps transitions. Never animate `width`, `height`, `top`, or `left`.

Place `<link rel="stylesheet">` in the `<head>`. Place `<script>` before `</body>` or use `defer`/`async` attributes to avoid blocking rendering.
